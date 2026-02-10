package keeper_test

import (
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/clawbotblockchain/clawchain/x/participation/keeper"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
	"github.com/stretchr/testify/require"
)

func TestRegisterWorker(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	workerAddr := sdk.AccAddress([]byte("worker1_____________")).String()

	// Register should succeed
	_, err := msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{
		Creator: workerAddr,
		Name:    "TestBot",
	})
	require.NoError(t, err)

	// Verify worker was stored
	worker, err := f.keeper.WorkerInfo.Get(f.ctx, workerAddr)
	require.NoError(t, err)
	require.Equal(t, "TestBot", worker.Name)
	require.True(t, worker.Active)
	require.Equal(t, uint64(0), worker.HeartbeatCount)
	require.Equal(t, "0", worker.TotalRewardsEarned)

	// Register again should fail
	_, err = msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{
		Creator: workerAddr,
		Name:    "TestBot2",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrWorkerAlreadyRegistered)
}

func TestWorkerHeartbeat(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	workerAddr := sdk.AccAddress([]byte("worker1_____________")).String()

	// Heartbeat without registration should fail
	_, err := msgServer.WorkerHeartbeat(f.ctx, &types.MsgWorkerHeartbeat{
		Creator: workerAddr,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrWorkerNotFound)

	// Register worker
	_, err = msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{
		Creator: workerAddr,
		Name:    "TestBot",
	})
	require.NoError(t, err)

	// Set block time so heartbeat works
	sdkCtx := sdk.UnwrapSDKContext(f.ctx)
	sdkCtx = sdkCtx.WithBlockTime(time.Unix(1000, 0))

	// First heartbeat should succeed
	_, err = msgServer.WorkerHeartbeat(sdkCtx, &types.MsgWorkerHeartbeat{
		Creator: workerAddr,
	})
	require.NoError(t, err)

	// Verify heartbeat count
	worker, err := f.keeper.WorkerInfo.Get(sdkCtx, workerAddr)
	require.NoError(t, err)
	require.Equal(t, uint64(1), worker.HeartbeatCount)
	require.Equal(t, uint64(1000), worker.LastHeartbeatTime)

	// Second heartbeat too early should fail
	sdkCtx = sdkCtx.WithBlockTime(time.Unix(1100, 0)) // only 100s later, need 300s
	_, err = msgServer.WorkerHeartbeat(sdkCtx, &types.MsgWorkerHeartbeat{
		Creator: workerAddr,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrHeartbeatTooEarly)

	// Third heartbeat after interval should succeed
	sdkCtx = sdkCtx.WithBlockTime(time.Unix(1301, 0)) // 301s later
	_, err = msgServer.WorkerHeartbeat(sdkCtx, &types.MsgWorkerHeartbeat{
		Creator: workerAddr,
	})
	require.NoError(t, err)

	worker, err = f.keeper.WorkerInfo.Get(sdkCtx, workerAddr)
	require.NoError(t, err)
	require.Equal(t, uint64(2), worker.HeartbeatCount)
}

func TestUnregisterWorker(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	workerAddr := sdk.AccAddress([]byte("worker1_____________")).String()

	// Unregister without registration should fail
	_, err := msgServer.UnregisterWorker(f.ctx, &types.MsgUnregisterWorker{
		Creator: workerAddr,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrWorkerNotFound)

	// Register then unregister
	_, err = msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{
		Creator: workerAddr,
		Name:    "TestBot",
	})
	require.NoError(t, err)

	_, err = msgServer.UnregisterWorker(f.ctx, &types.MsgUnregisterWorker{
		Creator: workerAddr,
	})
	require.NoError(t, err)

	// Verify worker is inactive
	worker, err := f.keeper.WorkerInfo.Get(f.ctx, workerAddr)
	require.NoError(t, err)
	require.False(t, worker.Active)

	// Heartbeat on inactive worker should fail
	sdkCtx := sdk.UnwrapSDKContext(f.ctx)
	sdkCtx = sdkCtx.WithBlockTime(time.Unix(1000, 0))
	_, err = msgServer.WorkerHeartbeat(sdkCtx, &types.MsgWorkerHeartbeat{
		Creator: workerAddr,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrWorkerInactive)
}

func TestRewardSplit(t *testing.T) {
	f := initFixture(t)

	// Set up params with known values
	params := types.DefaultParams()
	params.DailyRewardAmount = "1000000" // 1M for easy math
	params.WorkerRewardRatio = 60        // 60% to workers
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	// Set epoch info
	epochInfo := types.EpochInfo{
		CurrentEpoch:           1,
		EpochStartTime:         0,
		EpochDuration:          86400,
		TotalRewardDistributed: "0",
	}
	require.NoError(t, f.keeper.EpochInfo.Set(f.ctx, epochInfo))

	// Add a validator with metrics
	valAddr := "claw1validator"
	metrics := types.ValidatorMetrics{
		Index:             valAddr,
		StakedAmount:      "100000",
		BlocksProposed:    10,
		TxProcessed:       100,
		UptimeNumerator:   100,
		UptimeDenominator: 100,
		LastActiveEpoch:   0,
	}
	require.NoError(t, f.keeper.ValidatorMetrics.Set(f.ctx, valAddr, metrics))

	// Add a worker with heartbeats
	workerAddr := "claw1worker1"
	worker := types.WorkerInfo{
		Index:              workerAddr,
		Name:               "Bot1",
		RegisteredAt:       0,
		Active:             true,
		HeartbeatCount:     100,
		LastHeartbeatTime:  86000,
		LastActiveEpoch:    0,
		TotalRewardsEarned: "0",
	}
	require.NoError(t, f.keeper.WorkerInfo.Set(f.ctx, workerAddr, worker))

	// Calculate rewards
	err := f.keeper.CalculateEpochRewards(f.ctx, 1)
	require.NoError(t, err)

	// Verify validator got rewards (40% of 1M = 400K)
	// With 100% score (all weights met), validator should get 400K
	valRecord, err := f.keeper.RewardRecord.Get(f.ctx, valAddr+"/1")
	require.NoError(t, err)
	require.False(t, valRecord.Claimed)
	// Validator score = (100/100)*20 + (100/100)*60 + (100/100)*20 = 100
	// Reward = 100/100 * 400000 = 400000
	require.Equal(t, "400000", valRecord.RewardAmount)

	// Verify worker got rewards (60% of 1M = 600K)
	workerRecord, err := f.keeper.WorkerRewardRecord.Get(f.ctx, workerAddr+"/1")
	require.NoError(t, err)
	require.False(t, workerRecord.Claimed)
	require.Equal(t, "600000", workerRecord.RewardAmount)

	// Verify worker's total rewards were updated
	updatedWorker, err := f.keeper.WorkerInfo.Get(f.ctx, workerAddr)
	require.NoError(t, err)
	require.Equal(t, "600000", updatedWorker.TotalRewardsEarned)
}

func TestWorkerRewardProportional(t *testing.T) {
	f := initFixture(t)

	params := types.DefaultParams()
	params.DailyRewardAmount = "1000000"
	params.WorkerRewardRatio = 100 // 100% to workers for this test
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	epochInfo := types.EpochInfo{
		CurrentEpoch:           1,
		EpochStartTime:         0,
		EpochDuration:          86400,
		TotalRewardDistributed: "0",
	}
	require.NoError(t, f.keeper.EpochInfo.Set(f.ctx, epochInfo))

	// Worker A with 75 heartbeats
	workerA := types.WorkerInfo{
		Index:              "claw1workerA",
		Name:               "BotA",
		Active:             true,
		HeartbeatCount:     75,
		TotalRewardsEarned: "0",
	}
	// Worker B with 25 heartbeats
	workerB := types.WorkerInfo{
		Index:              "claw1workerB",
		Name:               "BotB",
		Active:             true,
		HeartbeatCount:     25,
		TotalRewardsEarned: "0",
	}
	require.NoError(t, f.keeper.WorkerInfo.Set(f.ctx, "claw1workerA", workerA))
	require.NoError(t, f.keeper.WorkerInfo.Set(f.ctx, "claw1workerB", workerB))

	err := f.keeper.CalculateEpochRewards(f.ctx, 1)
	require.NoError(t, err)

	// Worker A should get 75% = 750000
	recordA, err := f.keeper.WorkerRewardRecord.Get(f.ctx, "claw1workerA/1")
	require.NoError(t, err)
	require.Equal(t, "750000", recordA.RewardAmount)

	// Worker B should get 25% = 250000
	recordB, err := f.keeper.WorkerRewardRecord.Get(f.ctx, "claw1workerB/1")
	require.NoError(t, err)
	require.Equal(t, "250000", recordB.RewardAmount)
}

func TestDeactivateInactiveWorkers(t *testing.T) {
	f := initFixture(t)

	params := types.DefaultParams()
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	// Worker registered at time 0, last heartbeat at time 100
	worker := types.WorkerInfo{
		Index:              "claw1worker",
		Name:               "Bot",
		RegisteredAt:       0,
		Active:             true,
		HeartbeatCount:     1,
		LastHeartbeatTime:  100,
		TotalRewardsEarned: "0",
	}
	require.NoError(t, f.keeper.WorkerInfo.Set(f.ctx, "claw1worker", worker))

	// At time 100 + 300*100 = 30100, worker should be deactivated
	// (max_missed = 100 intervals of 300s = 30000s after last heartbeat)
	err := f.keeper.DeactivateInactiveWorkers(f.ctx, params, 30100)
	require.NoError(t, err)

	// Check still active (exactly at boundary)
	w, err := f.keeper.WorkerInfo.Get(f.ctx, "claw1worker")
	require.NoError(t, err)
	require.True(t, w.Active) // 30100 is not > 100 + 30000 = 30100

	// At time 30101, should be deactivated
	err = f.keeper.DeactivateInactiveWorkers(f.ctx, params, 30101)
	require.NoError(t, err)

	w, err = f.keeper.WorkerInfo.Get(f.ctx, "claw1worker")
	require.NoError(t, err)
	require.False(t, w.Active)
}

func TestResetWorkerMetrics(t *testing.T) {
	f := initFixture(t)

	worker := types.WorkerInfo{
		Index:              "claw1worker",
		Name:               "Bot",
		Active:             true,
		HeartbeatCount:     50,
		LastActiveEpoch:    1,
		TotalRewardsEarned: "1000",
	}
	require.NoError(t, f.keeper.WorkerInfo.Set(f.ctx, "claw1worker", worker))

	err := f.keeper.ResetWorkerMetrics(f.ctx, 2)
	require.NoError(t, err)

	w, err := f.keeper.WorkerInfo.Get(f.ctx, "claw1worker")
	require.NoError(t, err)
	require.Equal(t, uint64(0), w.HeartbeatCount)
	require.Equal(t, uint64(2), w.LastActiveEpoch)
	// Total rewards should not be reset
	require.Equal(t, "1000", w.TotalRewardsEarned)
}

func TestMaxWorkersLimit(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	// Set max_workers to 3
	params := types.DefaultParams()
	params.MaxWorkers = 3
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	// Register 3 workers — should all succeed
	for i := 0; i < 3; i++ {
		addr := sdk.AccAddress([]byte(fmt.Sprintf("maxworker%d___________", i))).String()
		_, err := msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{
			Creator: addr,
			Name:    fmt.Sprintf("Bot%d", i),
		})
		require.NoError(t, err, "worker %d should register", i)
	}

	// Verify count is 3
	count, err := f.keeper.WorkerCount.Get(f.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(3), count)

	// 4th registration should fail
	addr4 := sdk.AccAddress([]byte("maxworker3___________")).String()
	_, err = msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{
		Creator: addr4,
		Name:    "Bot3",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrMaxWorkersReached)
}

func TestMaxWorkersUnlimited(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	// Set max_workers to 0 (unlimited)
	params := types.DefaultParams()
	params.MaxWorkers = 0
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	// Register many workers — should all succeed
	for i := 0; i < 10; i++ {
		addr := sdk.AccAddress([]byte(fmt.Sprintf("unlimworker%d________", i))).String()
		_, err := msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{
			Creator: addr,
			Name:    fmt.Sprintf("Bot%d", i),
		})
		require.NoError(t, err, "worker %d should register with unlimited", i)
	}

	count, err := f.keeper.WorkerCount.Get(f.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(10), count)
}

func TestUnregisterDoesNotFreeSlot(t *testing.T) {
	f := initFixture(t)
	msgServer := keeper.NewMsgServerImpl(f.keeper)

	// Set max_workers to 2
	params := types.DefaultParams()
	params.MaxWorkers = 2
	require.NoError(t, f.keeper.Params.Set(f.ctx, params))

	// Register 2 workers
	addr1 := sdk.AccAddress([]byte("slotworker0__________")).String()
	addr2 := sdk.AccAddress([]byte("slotworker1__________")).String()
	_, err := msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{Creator: addr1, Name: "Bot0"})
	require.NoError(t, err)
	_, err = msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{Creator: addr2, Name: "Bot1"})
	require.NoError(t, err)

	// Unregister worker 1
	_, err = msgServer.UnregisterWorker(f.ctx, &types.MsgUnregisterWorker{Creator: addr1})
	require.NoError(t, err)

	// Count should still be 2 (unregister does NOT decrement)
	count, err := f.keeper.WorkerCount.Get(f.ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)

	// New worker should fail — slot not freed
	addr3 := sdk.AccAddress([]byte("slotworker2__________")).String()
	_, err = msgServer.RegisterWorker(f.ctx, &types.MsgRegisterWorker{Creator: addr3, Name: "Bot2"})
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrMaxWorkersReached)
}

func TestGenesisWithWorkers(t *testing.T) {
	f := initFixture(t)

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		WorkerMap: []types.WorkerInfo{
			{Index: "claw1worker1", Name: "Bot1", Active: true, HeartbeatCount: 10, TotalRewardsEarned: "500"},
			{Index: "claw1worker2", Name: "Bot2", Active: false, HeartbeatCount: 0, TotalRewardsEarned: "200"},
		},
	}

	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)

	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Len(t, got.WorkerMap, 2)

	// Verify workers were properly stored and exported
	w1, err := f.keeper.WorkerInfo.Get(f.ctx, "claw1worker1")
	require.NoError(t, err)
	require.Equal(t, "Bot1", w1.Name)
	require.True(t, w1.Active)

	w2, err := f.keeper.WorkerInfo.Get(f.ctx, "claw1worker2")
	require.NoError(t, err)
	require.Equal(t, "Bot2", w2.Name)
	require.False(t, w2.Active)
}
