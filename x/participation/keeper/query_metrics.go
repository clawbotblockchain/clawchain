package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) Metrics(ctx context.Context, req *types.QueryMetricsRequest) (*types.QueryMetricsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "validator_address is required")
	}

	metrics, err := q.k.ValidatorMetrics.Get(ctx, req.ValidatorAddress)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "validator metrics not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Calculate uptime percentage string
	uptime := "0.00%"
	if metrics.UptimeDenominator > 0 {
		pct := float64(metrics.UptimeNumerator) / float64(metrics.UptimeDenominator) * 100
		uptime = fmt.Sprintf("%.2f%%", pct)
	}

	// Get current epoch
	epochInfo, err := q.k.EpochInfo.Get(ctx)
	currentEpoch := uint64(0)
	if err == nil {
		currentEpoch = epochInfo.CurrentEpoch
	}

	return &types.QueryMetricsResponse{
		StakedAmount:   metrics.StakedAmount,
		BlocksProposed: metrics.BlocksProposed,
		TxProcessed:    metrics.TxProcessed,
		Uptime:         uptime,
		CurrentEpoch:   currentEpoch,
	}, nil
}
