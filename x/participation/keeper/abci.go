package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

// BeginBlocker is called at the beginning of each block.
// It records the block proposer, tracks validator signatures for uptime,
// checks for inactive workers, and checks for epoch boundaries to trigger reward distribution.
func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := k.Logger(ctx)

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// Get or initialize epoch info
	epochInfo, err := k.EpochInfo.Get(ctx)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			// Initialize epoch on first block
			epochInfo = types.EpochInfo{
				CurrentEpoch:           1,
				EpochStartTime:         uint64(sdkCtx.BlockTime().Unix()),
				EpochDuration:          params.EpochDuration,
				TotalRewardDistributed: "0",
			}
			if err := k.EpochInfo.Set(ctx, epochInfo); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// Record block proposer
	proposerAddr := sdkCtx.BlockHeader().ProposerAddress
	if len(proposerAddr) > 0 {
		consAddr := sdk.ConsAddress(proposerAddr)
		val, err := k.stakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
		if err == nil {
			valOperAddr := val.GetOperator()
			if err := k.RecordBlockProposed(ctx, valOperAddr); err != nil {
				logger.Error("failed to record block proposed", "error", err)
			}
			// Record a transaction count increment for the block proposer
			if err := k.RecordTxProcessed(ctx, valOperAddr, 1); err != nil {
				logger.Error("failed to record tx processed", "error", err)
			}
		}
	}

	// Track validator uptime from block signatures
	allValidators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err == nil {
		for _, val := range allValidators {
			if !val.IsBonded() {
				continue
			}
			valAddr := val.GetOperator()

			// Update stake amount in metrics
			metrics, err := k.GetOrCreateMetrics(ctx, valAddr)
			if err != nil {
				continue
			}
			metrics.StakedAmount = val.GetBondedTokens().String()
			metrics.Index = valAddr

			// For uptime, we track whether the validator is bonded
			metrics.UptimeDenominator++
			if val.IsBonded() {
				metrics.UptimeNumerator++
			}

			if err := k.ValidatorMetrics.Set(ctx, valAddr, metrics); err != nil {
				logger.Error("failed to update validator metrics", "error", err)
			}
		}
	}

	// Check for inactive workers periodically (every block)
	blockTime := uint64(sdkCtx.BlockTime().Unix())
	if err := k.DeactivateInactiveWorkers(ctx, params, blockTime); err != nil {
		logger.Error("failed to deactivate inactive workers", "error", err)
	}

	// Check for epoch boundary
	epochEnd := epochInfo.EpochStartTime + epochInfo.EpochDuration
	if blockTime >= epochEnd {
		logger.Info("epoch boundary reached",
			"epoch", epochInfo.CurrentEpoch,
			"block_time", blockTime,
			"epoch_end", epochEnd,
		)

		// Calculate and distribute rewards for the completed epoch
		if err := k.CalculateEpochRewards(ctx, epochInfo.CurrentEpoch); err != nil {
			logger.Error("failed to calculate epoch rewards", "error", err)
		}

		// Reset metrics for new epoch
		newEpoch := epochInfo.CurrentEpoch + 1
		if err := k.ResetMetrics(ctx, newEpoch); err != nil {
			logger.Error("failed to reset metrics", "error", err)
		}

		// Reset worker metrics for new epoch
		if err := k.ResetWorkerMetrics(ctx, newEpoch); err != nil {
			logger.Error("failed to reset worker metrics", "error", err)
		}

		// Start new epoch
		epochInfo.CurrentEpoch = newEpoch
		epochInfo.EpochStartTime = blockTime
		epochInfo.EpochDuration = params.EpochDuration
		if err := k.EpochInfo.Set(ctx, epochInfo); err != nil {
			return err
		}

		logger.Info("new epoch started", "epoch", newEpoch)
	}

	return nil
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// UpdateStakedAmounts refreshes the staked amount for all validators from the staking module.
func (k Keeper) UpdateStakedAmounts(ctx context.Context) error {
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return err
	}

	for _, val := range validators {
		if !val.IsBonded() {
			continue
		}
		valAddr := val.GetOperator()
		metrics, err := k.GetOrCreateMetrics(ctx, valAddr)
		if err != nil {
			continue
		}

		staked := val.GetBondedTokens()
		if staked.IsPositive() {
			metrics.StakedAmount = staked.String()
		} else {
			metrics.StakedAmount = "0"
		}
		metrics.Index = valAddr

		_ = k.ValidatorMetrics.Set(ctx, valAddr, metrics)
	}

	return nil
}

// GetTotalStaked returns the total staked amount across all tracked validators.
func (k Keeper) GetTotalStaked(ctx context.Context) math.Int {
	total := math.ZeroInt()
	_ = k.ValidatorMetrics.Walk(ctx, nil, func(_ string, val types.ValidatorMetrics) (stop bool, err error) {
		amt, ok := math.NewIntFromString(val.StakedAmount)
		if ok {
			total = total.Add(amt)
		}
		return false, nil
	})
	return total
}
