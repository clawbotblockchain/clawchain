package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetWorkerInfo returns info about a specific worker.
func (q queryServer) GetWorkerInfo(ctx context.Context, req *types.QueryGetWorkerInfoRequest) (*types.QueryGetWorkerInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	worker, err := q.k.WorkerInfo.Get(ctx, req.Address)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "worker not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetWorkerInfoResponse{Worker: worker}, nil
}

// ListWorkers returns a paginated list of all workers.
func (q queryServer) ListWorkers(ctx context.Context, req *types.QueryListWorkersRequest) (*types.QueryListWorkersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var workers []types.WorkerInfo
	err := q.k.WorkerInfo.Walk(ctx, nil, func(_ string, w types.WorkerInfo) (stop bool, err error) {
		workers = append(workers, w)
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListWorkersResponse{Workers: workers}, nil
}

// WorkerRewards returns reward info for a specific worker.
func (q queryServer) WorkerRewards(ctx context.Context, req *types.QueryWorkerRewardsRequest) (*types.QueryWorkerRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.Address == "" {
		return nil, status.Error(codes.InvalidArgument, "address is required")
	}

	totalEarned := math.ZeroInt()
	unclaimed := math.ZeroInt()
	lastEpochReward := math.ZeroInt()
	var maxEpoch uint64

	err := q.k.WorkerRewardRecord.Walk(ctx, nil, func(key string, val types.RewardRecord) (stop bool, err error) {
		if len(key) > len(req.Address) && key[:len(req.Address)] == req.Address {
			rewardAmt, ok := math.NewIntFromString(val.RewardAmount)
			if !ok {
				rewardAmt = math.ZeroInt()
			}
			totalEarned = totalEarned.Add(rewardAmt)
			if !val.Claimed {
				unclaimed = unclaimed.Add(rewardAmt)
			}
			if val.Epoch > maxEpoch {
				maxEpoch = val.Epoch
				lastEpochReward = rewardAmt
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryWorkerRewardsResponse{
		TotalEarned:     totalEarned.String(),
		Unclaimed:       unclaimed.String(),
		LastEpochReward: lastEpochReward.String(),
	}, nil
}

// WorkerStats returns aggregate worker statistics.
func (q queryServer) WorkerStats(ctx context.Context, req *types.QueryWorkerStatsRequest) (*types.QueryWorkerStatsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var totalWorkers, activeWorkers, totalHeartbeats uint64

	err := q.k.WorkerInfo.Walk(ctx, nil, func(_ string, w types.WorkerInfo) (stop bool, err error) {
		totalWorkers++
		if w.Active {
			activeWorkers++
			totalHeartbeats += w.HeartbeatCount
		}
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryWorkerStatsResponse{
		TotalWorkers:              totalWorkers,
		ActiveWorkers:             activeWorkers,
		TotalHeartbeatsThisEpoch:  totalHeartbeats,
	}, nil
}
