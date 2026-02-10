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

func (q queryServer) ListValidatorMetrics(ctx context.Context, req *types.QueryAllValidatorMetricsRequest) (*types.QueryAllValidatorMetricsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	validatorMetricss, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.ValidatorMetrics,
		req.Pagination,
		func(_ string, value types.ValidatorMetrics) (types.ValidatorMetrics, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllValidatorMetricsResponse{ValidatorMetrics: validatorMetricss, Pagination: pageRes}, nil
}

func (q queryServer) GetValidatorMetrics(ctx context.Context, req *types.QueryGetValidatorMetricsRequest) (*types.QueryGetValidatorMetricsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	val, err := q.k.ValidatorMetrics.Get(ctx, req.Index)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetValidatorMetricsResponse{ValidatorMetrics: val}, nil
}
