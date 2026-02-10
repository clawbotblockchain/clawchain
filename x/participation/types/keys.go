package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "participation"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a dependency with x/gov.
	GovModuleName = "gov"

	// RewardPoolName is the name of the module account that holds participation rewards.
	RewardPoolName = "participation_reward_pool"

	// BondDenom is the staking/bond denomination for ClawChain.
	BondDenom = "aclaw"
)

// ParamsKey is the prefix to retrieve all Params
var ParamsKey = collections.NewPrefix("p_participation")

var (
	EpochInfoKey            = collections.NewPrefix("epochInfo/value/")
	ValidatorMetricsKey     = collections.NewPrefix("validatorMetrics/value/")
	RewardRecordKey         = collections.NewPrefix("rewardRecord/value/")
	WorkerInfoKey           = collections.NewPrefix("workerInfo/value/")
	WorkerRewardRecordKey   = collections.NewPrefix("workerRewardRecord/value/")
	WorkerCountKey          = collections.NewPrefix("workerCount/value/")
)
