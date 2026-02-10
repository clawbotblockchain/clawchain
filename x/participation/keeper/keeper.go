package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	authority    []byte

	authKeeper    types.AuthKeeper
	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper

	Schema             collections.Schema
	Params             collections.Item[types.Params]
	ValidatorMetrics   collections.Map[string, types.ValidatorMetrics]
	EpochInfo          collections.Item[types.EpochInfo]
	RewardRecord       collections.Map[string, types.RewardRecord]
	WorkerInfo         collections.Map[string, types.WorkerInfo]
	WorkerRewardRecord collections.Map[string, types.RewardRecord]
	WorkerCount        collections.Item[uint64]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,
	authKeeper types.AuthKeeper,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService:  storeService,
		cdc:           cdc,
		addressCodec:  addressCodec,
		authority:     authority,
		authKeeper:    authKeeper,
		bankKeeper:    bankKeeper,
		stakingKeeper: stakingKeeper,
		Params:             collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		ValidatorMetrics:   collections.NewMap(sb, types.ValidatorMetricsKey, "validatorMetrics", collections.StringKey, codec.CollValue[types.ValidatorMetrics](cdc)),
		EpochInfo:          collections.NewItem(sb, types.EpochInfoKey, "epochInfo", codec.CollValue[types.EpochInfo](cdc)),
		RewardRecord:       collections.NewMap(sb, types.RewardRecordKey, "rewardRecord", collections.StringKey, codec.CollValue[types.RewardRecord](cdc)),
		WorkerInfo:         collections.NewMap(sb, types.WorkerInfoKey, "workerInfo", collections.StringKey, codec.CollValue[types.WorkerInfo](cdc)),
		WorkerRewardRecord: collections.NewMap(sb, types.WorkerRewardRecordKey, "workerRewardRecord", collections.StringKey, codec.CollValue[types.RewardRecord](cdc)),
		WorkerCount:        collections.NewItem(sb, types.WorkerCountKey, "workerCount", collections.Uint64Value),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
