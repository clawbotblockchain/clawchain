package keeper

import (
	"context"

	"cosmossdk.io/math"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) Rewards(ctx context.Context, req *types.QueryRewardsRequest) (*types.QueryRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "validator_address is required")
	}

	totalEarned := math.ZeroInt()
	unclaimed := math.ZeroInt()
	lastEpochReward := math.ZeroInt()
	var maxEpoch uint64

	// Walk all reward records for this validator
	err := q.k.RewardRecord.Walk(ctx, nil, func(key string, val types.RewardRecord) (stop bool, err error) {
		// Key format: "valAddr/epoch"
		if len(key) > len(req.ValidatorAddress) && key[:len(req.ValidatorAddress)] == req.ValidatorAddress {
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

	return &types.QueryRewardsResponse{
		TotalEarned:     totalEarned.String(),
		Unclaimed:       unclaimed.String(),
		LastEpochReward: lastEpochReward.String(),
	}, nil
}
