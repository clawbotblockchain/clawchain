package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	for _, elem := range genState.ValidatorMetricsMap {
		if err := k.ValidatorMetrics.Set(ctx, elem.Index, elem); err != nil {
			return err
		}
	}
	if genState.EpochInfo != nil {
		if err := k.EpochInfo.Set(ctx, *genState.EpochInfo); err != nil {
			return err
		}
	}
	for _, elem := range genState.RewardRecordMap {
		if err := k.RewardRecord.Set(ctx, elem.Index, elem); err != nil {
			return err
		}
	}
	for _, elem := range genState.WorkerMap {
		if err := k.WorkerInfo.Set(ctx, elem.Index, elem); err != nil {
			return err
		}
	}

	// Initialize worker count from genesis worker map
	if err := k.WorkerCount.Set(ctx, uint64(len(genState.WorkerMap))); err != nil {
		return err
	}

	return k.Params.Set(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := k.ValidatorMetrics.Walk(ctx, nil, func(_ string, val types.ValidatorMetrics) (stop bool, err error) {
		genesis.ValidatorMetricsMap = append(genesis.ValidatorMetricsMap, val)
		return false, nil
	}); err != nil {
		return nil, err
	}
	epochInfo, err := k.EpochInfo.Get(ctx)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}
	genesis.EpochInfo = &epochInfo
	if err := k.RewardRecord.Walk(ctx, nil, func(_ string, val types.RewardRecord) (stop bool, err error) {
		genesis.RewardRecordMap = append(genesis.RewardRecordMap, val)
		return false, nil
	}); err != nil {
		return nil, err
	}
	if err := k.WorkerInfo.Walk(ctx, nil, func(_ string, w types.WorkerInfo) (stop bool, err error) {
		genesis.WorkerMap = append(genesis.WorkerMap, w)
		return false, nil
	}); err != nil {
		return nil, err
	}

	return genesis, nil
}
