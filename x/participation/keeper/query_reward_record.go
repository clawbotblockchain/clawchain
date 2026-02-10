package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) ListRewardRecord(ctx context.Context, req *types.QueryAllRewardRecordRequest) (*types.QueryAllRewardRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	rewardRecords, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.RewardRecord,
		req.Pagination,
		func(_ string, value types.RewardRecord) (types.RewardRecord, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllRewardRecordResponse{RewardRecord: rewardRecords, Pagination: pageRes}, nil
}

func (q queryServer) GetRewardRecord(ctx context.Context, req *types.QueryGetRewardRecordRequest) (*types.QueryGetRewardRecordResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, err := q.k.RewardRecord.Get(ctx, req.Index)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetRewardRecordResponse{RewardRecord: val}, nil
}
