package participation

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "ListValidatorMetrics",
					Use:       "list-validator-metrics",
					Short:     "List all validator_metrics",
				},
				{
					RpcMethod:      "GetValidatorMetrics",
					Use:            "get-validator-metrics [id]",
					Short:          "Gets a validator_metrics",
					Alias:          []string{"show-validator-metrics"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				},
				{
					RpcMethod: "GetEpochInfo",
					Use:       "get-epoch-info",
					Short:     "Gets a epoch_info",
					Alias:     []string{"show-epoch-info"},
				},
				{
					RpcMethod: "ListRewardRecord",
					Use:       "list-reward-record",
					Short:     "List all reward_record",
				},
				{
					RpcMethod:      "GetRewardRecord",
					Use:            "get-reward-record [id]",
					Short:          "Gets a reward_record",
					Alias:          []string{"show-reward-record"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "index"}},
				},
				{
					RpcMethod:      "Metrics",
					Use:            "metrics [validator-address]",
					Short:          "Query metrics",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "validator_address"}},
				},

				{
					RpcMethod:      "Leaderboard",
					Use:            "leaderboard ",
					Short:          "Query leaderboard",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},

				{
					RpcMethod:      "Rewards",
					Use:            "rewards [validator-address]",
					Short:          "Query rewards",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "validator_address"}},
				},

				// this line is used by ignite scaffolding # autocli/query
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "ClaimRewards",
					Use:            "claim-rewards ",
					Short:          "Send a claim-rewards tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				// this line is used by ignite scaffolding # autocli/tx
			},
		},
	}
}
