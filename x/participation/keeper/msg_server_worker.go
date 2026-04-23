package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

// RegisterWorker registers a new worker for participation rewards.
func (k msgServer) RegisterWorker(ctx context.Context, msg *types.MsgRegisterWorker) (*types.MsgRegisterWorkerResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, "invalid creator address")
	}

	// Check if already registered
	_, err := k.Keeper.WorkerInfo.Get(ctx, msg.Creator)
	if err == nil {
		return nil, errorsmod.Wrap(types.ErrWorkerAlreadyRegistered, msg.Creator)
	}
	if !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}

	// Check max_workers limit
	params, err := k.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	if params.MaxWorkers > 0 {
		count, err := k.Keeper.WorkerCount.Get(ctx)
		if err != nil {
			// Not found means 0
			if !errors.Is(err, collections.ErrNotFound) {
				return nil, err
			}
			count = 0
		}
		if count >= params.MaxWorkers {
			return nil, errorsmod.Wrapf(types.ErrMaxWorkersReached, "limit %d reached", params.MaxWorkers)
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := uint64(sdkCtx.BlockTime().Unix())

	worker := types.WorkerInfo{
		Index:              msg.Creator,
		Name:               msg.Name,
		RegisteredAt:       blockTime,
		Active:             true,
		HeartbeatCount:     0,
		LastHeartbeatTime:  0,
		LastActiveEpoch:    0,
		TotalRewardsEarned: "0",
	}

	if err := k.Keeper.WorkerInfo.Set(ctx, msg.Creator, worker); err != nil {
		return nil, err
	}

	// Increment worker count (permanent — unregister does NOT decrement)
	count, err := k.Keeper.WorkerCount.Get(ctx)
	if err != nil {
		if !errors.Is(err, collections.ErrNotFound) {
			return nil, err
		}
		count = 0
	}
	if err := k.Keeper.WorkerCount.Set(ctx, count+1); err != nil {
		return nil, err
	}

	return &types.MsgRegisterWorkerResponse{}, nil
}

// WorkerHeartbeat records a heartbeat from a worker.
func (k msgServer) WorkerHeartbeat(ctx context.Context, msg *types.MsgWorkerHeartbeat) (*types.MsgWorkerHeartbeatResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, "invalid creator address")
	}

	worker, err := k.Keeper.WorkerInfo.Get(ctx, msg.Creator)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(types.ErrWorkerNotFound, msg.Creator)
		}
		return nil, err
	}

	if !worker.Active {
		return nil, errorsmod.Wrap(types.ErrWorkerInactive, msg.Creator)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	blockTime := uint64(sdkCtx.BlockTime().Unix())

	// Check heartbeat interval
	params, err := k.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	if worker.LastHeartbeatTime > 0 && blockTime < worker.LastHeartbeatTime+params.HeartbeatInterval {
		return nil, errorsmod.Wrapf(types.ErrHeartbeatTooEarly,
			"next heartbeat allowed at %d, current time %d",
			worker.LastHeartbeatTime+params.HeartbeatInterval, blockTime)
	}

	worker.HeartbeatCount++
	worker.LastHeartbeatTime = blockTime

	if err := k.Keeper.WorkerInfo.Set(ctx, msg.Creator, worker); err != nil {
		return nil, err
	}

	return &types.MsgWorkerHeartbeatResponse{}, nil
}

// UnregisterWorker marks a worker as inactive.
func (k msgServer) UnregisterWorker(ctx context.Context, msg *types.MsgUnregisterWorker) (*types.MsgUnregisterWorkerResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, "invalid creator address")
	}

	worker, err := k.Keeper.WorkerInfo.Get(ctx, msg.Creator)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(types.ErrWorkerNotFound, msg.Creator)
		}
		return nil, err
	}

	worker.Active = false

	if err := k.Keeper.WorkerInfo.Set(ctx, msg.Creator, worker); err != nil {
		return nil, err
	}

	return &types.MsgUnregisterWorkerResponse{}, nil
}

// ReactivateWorker marks a previously deactivated worker as active again.
// Resets LastHeartbeatTime to the current block so the worker does not immediately
// re-deactivate. Only the worker itself can reactivate its registration.
func (k msgServer) ReactivateWorker(ctx context.Context, msg *types.MsgReactivateWorker) (*types.MsgReactivateWorkerResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, "invalid creator address")
	}

	worker, err := k.Keeper.WorkerInfo.Get(ctx, msg.Creator)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(types.ErrWorkerNotFound, msg.Creator)
		}
		return nil, err
	}

	if worker.Active {
		return nil, errorsmod.Wrap(types.ErrWorkerAlreadyActive, msg.Creator)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	worker.Active = true
	worker.LastHeartbeatTime = uint64(sdkCtx.BlockTime().Unix())

	if err := k.Keeper.WorkerInfo.Set(ctx, msg.Creator, worker); err != nil {
		return nil, err
	}

	return &types.MsgReactivateWorkerResponse{}, nil
}

// ClaimWorkerRewards claims all unclaimed worker rewards.
func (k msgServer) ClaimWorkerRewards(ctx context.Context, msg *types.MsgClaimWorkerRewards) (*types.MsgClaimWorkerRewardsResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, "invalid creator address")
	}

	claimed, err := k.Keeper.ClaimRewardsForWorker(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	return &types.MsgClaimWorkerRewardsResponse{
		ClaimedAmount: claimed.String(),
	}, nil
}
