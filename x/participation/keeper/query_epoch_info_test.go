package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/clawbotblockchain/clawchain/x/participation/keeper"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

func TestEpochInfoQuery(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	item := types.EpochInfo{}
	err := f.keeper.EpochInfo.Set(f.ctx, item)
	require.NoError(t, err)

	tests := []struct {
		desc     string
		request  *types.QueryGetEpochInfoRequest
		response *types.QueryGetEpochInfoResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetEpochInfoRequest{},
			response: &types.QueryGetEpochInfoResponse{EpochInfo: item},
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetEpochInfo(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}
