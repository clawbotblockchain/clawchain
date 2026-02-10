package types_test

import (
	"testing"

	"github.com/clawbotblockchain/clawchain/x/participation/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ValidatorMetricsMap: []types.ValidatorMetrics{{Index: "0"}, {Index: "1"}}, EpochInfo: &types.EpochInfo{CurrentEpoch: 33,
					EpochStartTime:         39,
					EpochDuration:          24,
					TotalRewardDistributed: "16",
				}, RewardRecordMap: []types.RewardRecord{{Index: "0"}, {Index: "1"}}},
			valid: true,
		}, {
			desc: "duplicated validatorMetrics",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ValidatorMetricsMap: []types.ValidatorMetrics{
					{
						Index: "0",
					},
					{
						Index: "0",
					},
				},
				EpochInfo: &types.EpochInfo{CurrentEpoch: 33,
					EpochStartTime:         39,
					EpochDuration:          24,
					TotalRewardDistributed: "16",
				}, RewardRecordMap: []types.RewardRecord{{Index: "0"}, {Index: "1"}}},
			valid: false,
		}, {
			desc: "duplicated rewardRecord",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				RewardRecordMap: []types.RewardRecord{
					{
						Index: "0",
					},
					{
						Index: "0",
					},
				},
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
