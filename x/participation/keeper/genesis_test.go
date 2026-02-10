package keeper_test

import (
	"testing"

	"github.com/clawbotblockchain/clawchain/x/participation/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:              types.DefaultParams(),
		ValidatorMetricsMap: []types.ValidatorMetrics{{Index: "0"}, {Index: "1"}}, EpochInfo: &types.EpochInfo{CurrentEpoch: 64,
			EpochStartTime:         16,
			EpochDuration:          62,
			TotalRewardDistributed: "99",
		}, RewardRecordMap: []types.RewardRecord{{Index: "0"}, {Index: "1"}}}

	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
	require.EqualExportedValues(t, genesisState.ValidatorMetricsMap, got.ValidatorMetricsMap)
	require.EqualExportedValues(t, genesisState.EpochInfo, got.EpochInfo)
	require.EqualExportedValues(t, genesisState.RewardRecordMap, got.RewardRecordMap)

}
