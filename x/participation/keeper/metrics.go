package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

// GetOrCreateMetrics returns existing metrics for a validator or creates new empty ones.
func (k Keeper) GetOrCreateMetrics(ctx context.Context, valAddr string) (types.ValidatorMetrics, error) {
	metrics, err := k.ValidatorMetrics.Get(ctx, valAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.ValidatorMetrics{
				Index:             valAddr,
				StakedAmount:      "0",
				BlocksProposed:    0,
				TxProcessed:       0,
				UptimeNumerator:   0,
				UptimeDenominator: 0,
				LastActiveEpoch:   0,
			}, nil
		}
		return types.ValidatorMetrics{}, err
	}
	return metrics, nil
}

// RecordBlockProposed increments the block proposal count for a validator.
func (k Keeper) RecordBlockProposed(ctx context.Context, valAddr string) error {
	metrics, err := k.GetOrCreateMetrics(ctx, valAddr)
	if err != nil {
		return err
	}
	metrics.BlocksProposed++
	return k.ValidatorMetrics.Set(ctx, valAddr, metrics)
}

// RecordTxProcessed increments the transaction processing count for a validator.
func (k Keeper) RecordTxProcessed(ctx context.Context, valAddr string, txCount uint64) error {
	metrics, err := k.GetOrCreateMetrics(ctx, valAddr)
	if err != nil {
		return err
	}
	metrics.TxProcessed += txCount
	return k.ValidatorMetrics.Set(ctx, valAddr, metrics)
}

// UpdateUptime records a validator's block signing status.
func (k Keeper) UpdateUptime(ctx context.Context, valAddr string, signed bool) error {
	metrics, err := k.GetOrCreateMetrics(ctx, valAddr)
	if err != nil {
		return err
	}
	metrics.UptimeDenominator++
	if signed {
		metrics.UptimeNumerator++
	}
	return k.ValidatorMetrics.Set(ctx, valAddr, metrics)
}

// ResetMetrics resets per-epoch metrics for all validators while preserving uptime.
func (k Keeper) ResetMetrics(ctx context.Context, currentEpoch uint64) error {
	var toUpdate []types.ValidatorMetrics
	err := k.ValidatorMetrics.Walk(ctx, nil, func(_ string, val types.ValidatorMetrics) (stop bool, err error) {
		val.BlocksProposed = 0
		val.TxProcessed = 0
		val.UptimeNumerator = 0
		val.UptimeDenominator = 0
		val.LastActiveEpoch = currentEpoch
		toUpdate = append(toUpdate, val)
		return false, nil
	})
	if err != nil {
		return err
	}
	for _, m := range toUpdate {
		if err := k.ValidatorMetrics.Set(ctx, m.Index, m); err != nil {
			return err
		}
	}
	return nil
}

// ResetWorkerMetrics resets heartbeat counts for all active workers at epoch boundary.
func (k Keeper) ResetWorkerMetrics(ctx context.Context, currentEpoch uint64) error {
	var toUpdate []types.WorkerInfo
	err := k.WorkerInfo.Walk(ctx, nil, func(_ string, w types.WorkerInfo) (stop bool, err error) {
		if w.Active {
			w.HeartbeatCount = 0
			w.LastActiveEpoch = currentEpoch
			toUpdate = append(toUpdate, w)
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	for _, w := range toUpdate {
		if err := k.WorkerInfo.Set(ctx, w.Index, w); err != nil {
			return err
		}
	}
	return nil
}

// DeactivateInactiveWorkers marks workers as inactive if they've missed too many heartbeat intervals.
func (k Keeper) DeactivateInactiveWorkers(ctx context.Context, params types.Params, blockTime uint64) error {
	maxMissedDuration := params.HeartbeatInterval * params.MaxMissedHeartbeats

	var toDeactivate []types.WorkerInfo
	err := k.WorkerInfo.Walk(ctx, nil, func(_ string, w types.WorkerInfo) (stop bool, err error) {
		if !w.Active {
			return false, nil
		}
		// If the worker has never sent a heartbeat, check against registration time
		lastActivity := w.LastHeartbeatTime
		if lastActivity == 0 {
			lastActivity = w.RegisteredAt
		}
		if blockTime > lastActivity+maxMissedDuration {
			w.Active = false
			toDeactivate = append(toDeactivate, w)
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	for _, w := range toDeactivate {
		if err := k.WorkerInfo.Set(ctx, w.Index, w); err != nil {
			return err
		}
	}
	return nil
}
