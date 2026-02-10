package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/clawbotblockchain/clawchain/x/participation/keeper"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

func createNRewardRecord(keeper keeper.Keeper, ctx context.Context, n int) []types.RewardRecord {
	items := make([]types.RewardRecord, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)
		items[i].RewardAmount = strconv.FormatUint(uint64(i), 10)
		items[i].StakeScore = uint64(i)
		items[i].ActivityScore = uint64(i)
		items[i].UptimeScore = uint64(i)
		items[i].Claimed = true
		items[i].Epoch = uint64(i)
		_ = keeper.RewardRecord.Set(ctx, items[i].Index, items[i])
	}
	return items
}

func TestRewardRecordQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNRewardRecord(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetRewardRecordRequest
		response *types.QueryGetRewardRecordResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetRewardRecordRequest{
				Index: msgs[0].Index,
			},
			response: &types.QueryGetRewardRecordResponse{RewardRecord: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetRewardRecordRequest{
				Index: msgs[1].Index,
			},
			response: &types.QueryGetRewardRecordResponse{RewardRecord: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetRewardRecordRequest{
				Index: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetRewardRecord(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestRewardRecordQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNRewardRecord(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllRewardRecordRequest {
		return &types.QueryAllRewardRecordRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListRewardRecord(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.RewardRecord), step)
			require.Subset(t, msgs, resp.RewardRecord)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListRewardRecord(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.RewardRecord), step)
			require.Subset(t, msgs, resp.RewardRecord)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListRewardRecord(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.RewardRecord)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListRewardRecord(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
