package keeper

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"cosmossdk.io/math"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LeaderboardEntry represents a validator's ranking entry.
type LeaderboardEntry struct {
	Address        string `json:"address"`
	StakedAmount   string `json:"staked_amount"`
	BlocksProposed uint64 `json:"blocks_proposed"`
	TxProcessed    uint64 `json:"tx_processed"`
	Uptime         string `json:"uptime"`
	Score          string `json:"score"`
}

func (q queryServer) Leaderboard(ctx context.Context, req *types.QueryLeaderboardRequest) (*types.QueryLeaderboardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	params, err := q.k.Params.Get(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	stakeWeight := math.LegacyNewDecFromInt(math.NewIntFromUint64(params.StakeWeight))
	activityWeight := math.LegacyNewDecFromInt(math.NewIntFromUint64(params.ActivityWeight))
	uptimeWeight := math.LegacyNewDecFromInt(math.NewIntFromUint64(params.UptimeWeight))

	// Collect all validator metrics
	type scored struct {
		entry LeaderboardEntry
		score math.LegacyDec
	}

	totalStake := math.LegacyZeroDec()
	totalTx := math.LegacyZeroDec()
	var allMetrics []types.ValidatorMetrics

	err = q.k.ValidatorMetrics.Walk(ctx, nil, func(_ string, val types.ValidatorMetrics) (stop bool, err error) {
		allMetrics = append(allMetrics, val)
		stakeAmt, _ := math.NewIntFromString(val.StakedAmount)
		totalStake = totalStake.Add(math.LegacyNewDecFromInt(stakeAmt))
		totalTx = totalTx.Add(math.LegacyNewDecFromInt(math.NewIntFromUint64(val.TxProcessed)))
		return false, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var entries []scored
	for _, m := range allMetrics {
		// Calculate composite score
		stakeScore := math.LegacyZeroDec()
		if totalStake.IsPositive() {
			mStake, _ := math.NewIntFromString(m.StakedAmount)
			stakeScore = math.LegacyNewDecFromInt(mStake).
				Quo(totalStake).Mul(stakeWeight)
		}

		actScore := math.LegacyZeroDec()
		if totalTx.IsPositive() {
			actScore = math.LegacyNewDecFromInt(math.NewIntFromUint64(m.TxProcessed)).
				Quo(totalTx).Mul(activityWeight)
		}

		upScore := math.LegacyZeroDec()
		if m.UptimeDenominator > 0 {
			upScore = math.LegacyNewDecFromInt(math.NewIntFromUint64(m.UptimeNumerator)).
				Quo(math.LegacyNewDecFromInt(math.NewIntFromUint64(m.UptimeDenominator))).
				Mul(uptimeWeight)
		}

		total := stakeScore.Add(actScore).Add(upScore)
		uptimePct := "0.00%"
		if m.UptimeDenominator > 0 {
			pct := float64(m.UptimeNumerator) / float64(m.UptimeDenominator) * 100
			uptimePct = json.Number(fmt.Sprintf("%.2f%%", pct)).String()
		}

		entries = append(entries, scored{
			entry: LeaderboardEntry{
				Address:        m.Index,
				StakedAmount:   m.StakedAmount,
				BlocksProposed: m.BlocksProposed,
				TxProcessed:    m.TxProcessed,
				Uptime:         uptimePct,
				Score:          total.String(),
			},
			score: total,
		})
	}

	// Sort by score descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score.GT(entries[j].score)
	})

	// Convert to JSON string
	leaderboard := make([]LeaderboardEntry, len(entries))
	for i, e := range entries {
		leaderboard[i] = e.entry
	}

	data, err := json.Marshal(leaderboard)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryLeaderboardResponse{
		Validators: string(data),
	}, nil
}
