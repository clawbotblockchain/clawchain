package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

// CalculateEpochRewards computes and distributes rewards for the completed epoch.
// Splits daily reward between validators and workers based on worker_reward_ratio.
func (k Keeper) CalculateEpochRewards(ctx context.Context, epoch uint64) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	dailyReward, ok := math.NewIntFromString(params.DailyRewardAmount)
	if !ok || dailyReward.IsZero() {
		return nil // no rewards to distribute
	}

	// Check reward pool balance
	rewardPoolAddr := k.authKeeper.GetModuleAddress(types.RewardPoolName)
	if rewardPoolAddr == nil {
		return fmt.Errorf("reward pool module account not found")
	}

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return err
	}

	poolBalance := k.bankKeeper.GetBalance(ctx, rewardPoolAddr, bondDenom)
	if poolBalance.Amount.LT(dailyReward) {
		dailyReward = poolBalance.Amount
	}
	if dailyReward.IsZero() {
		return nil
	}

	// Split daily reward between validators and workers
	workerRatio := params.WorkerRewardRatio
	workerShare := dailyReward.Mul(math.NewIntFromUint64(workerRatio)).Quo(math.NewIntFromUint64(100))
	validatorShare := dailyReward.Sub(workerShare)

	totalDistributed := math.ZeroInt()

	// --- Distribute validator rewards ---
	if validatorShare.IsPositive() {
		distributed, err := k.distributeValidatorRewards(ctx, params, validatorShare, epoch)
		if err != nil {
			return err
		}
		totalDistributed = totalDistributed.Add(distributed)
	}

	// --- Distribute worker rewards ---
	if workerShare.IsPositive() {
		distributed, err := k.distributeWorkerRewards(ctx, workerShare, epoch)
		if err != nil {
			return err
		}
		totalDistributed = totalDistributed.Add(distributed)
	}

	// Update epoch info with total distributed
	epochInfo, err := k.EpochInfo.Get(ctx)
	if err != nil {
		return err
	}
	prevDistributed, _ := math.NewIntFromString(epochInfo.TotalRewardDistributed)
	if prevDistributed.IsNil() {
		prevDistributed = math.ZeroInt()
	}
	epochInfo.TotalRewardDistributed = prevDistributed.Add(totalDistributed).String()
	return k.EpochInfo.Set(ctx, epochInfo)
}

// distributeValidatorRewards distributes the validator share of rewards.
func (k Keeper) distributeValidatorRewards(ctx context.Context, params types.Params, validatorShare math.Int, epoch uint64) (math.Int, error) {
	type valData struct {
		addr    string
		metrics types.ValidatorMetrics
		stake   math.LegacyDec
	}

	var validators []valData
	totalStake := math.LegacyZeroDec()
	totalTx := math.LegacyZeroDec()

	err := k.ValidatorMetrics.Walk(ctx, nil, func(_ string, val types.ValidatorMetrics) (stop bool, err error) {
		stakeInt, _ := math.NewIntFromString(val.StakedAmount)
		stake := math.LegacyNewDecFromInt(stakeInt)
		validators = append(validators, valData{
			addr:    val.Index,
			metrics: val,
			stake:   stake,
		})
		totalStake = totalStake.Add(stake)
		totalTx = totalTx.Add(math.LegacyNewDecFromInt(math.NewIntFromUint64(val.TxProcessed)))
		return false, nil
	})
	if err != nil {
		return math.ZeroInt(), err
	}

	if len(validators) == 0 {
		return math.ZeroInt(), nil
	}

	stakeWeight := math.LegacyNewDecFromInt(math.NewIntFromUint64(params.StakeWeight))
	activityWeight := math.LegacyNewDecFromInt(math.NewIntFromUint64(params.ActivityWeight))
	uptimeWeight := math.LegacyNewDecFromInt(math.NewIntFromUint64(params.UptimeWeight))

	totalShareDec := math.LegacyNewDecFromInt(validatorShare)
	totalDistributed := math.ZeroInt()

	for _, v := range validators {
		stakeScore := math.LegacyZeroDec()
		if totalStake.IsPositive() {
			stakeScore = v.stake.Quo(totalStake).Mul(stakeWeight)
		}

		activityScore := math.LegacyZeroDec()
		valTx := math.LegacyNewDecFromInt(math.NewIntFromUint64(v.metrics.TxProcessed))
		if totalTx.IsPositive() {
			activityScore = valTx.Quo(totalTx).Mul(activityWeight)
		}

		uptimeScore := math.LegacyZeroDec()
		if v.metrics.UptimeDenominator > 0 {
			uptimeFraction := math.LegacyNewDecFromInt(math.NewIntFromUint64(v.metrics.UptimeNumerator)).
				Quo(math.LegacyNewDecFromInt(math.NewIntFromUint64(v.metrics.UptimeDenominator)))
			uptimeScore = uptimeFraction.Mul(uptimeWeight)
		}

		totalScore := stakeScore.Add(activityScore).Add(uptimeScore)
		reward := totalScore.Quo(math.LegacyNewDec(100)).Mul(totalShareDec).TruncateInt()

		if reward.IsPositive() {
			recordKey := fmt.Sprintf("%s/%d", v.addr, epoch)
			record := types.RewardRecord{
				Index:         recordKey,
				RewardAmount:  reward.String(),
				StakeScore:    stakeScore.TruncateInt().Uint64(),
				ActivityScore: activityScore.TruncateInt().Uint64(),
				UptimeScore:   uptimeScore.TruncateInt().Uint64(),
				Claimed:       false,
				Epoch:         epoch,
			}
			if err := k.RewardRecord.Set(ctx, recordKey, record); err != nil {
				return math.ZeroInt(), err
			}
			totalDistributed = totalDistributed.Add(reward)
		}
	}

	return totalDistributed, nil
}

// distributeWorkerRewards distributes the worker share of rewards proportional to heartbeat counts.
func (k Keeper) distributeWorkerRewards(ctx context.Context, workerShare math.Int, epoch uint64) (math.Int, error) {
	type workerData struct {
		addr           string
		heartbeatCount uint64
	}

	var activeWorkers []workerData
	var totalHeartbeats uint64

	err := k.WorkerInfo.Walk(ctx, nil, func(_ string, w types.WorkerInfo) (stop bool, err error) {
		if w.Active && w.HeartbeatCount > 0 {
			activeWorkers = append(activeWorkers, workerData{
				addr:           w.Index,
				heartbeatCount: w.HeartbeatCount,
			})
			totalHeartbeats += w.HeartbeatCount
		}
		return false, nil
	})
	if err != nil {
		return math.ZeroInt(), err
	}

	if len(activeWorkers) == 0 || totalHeartbeats == 0 {
		return math.ZeroInt(), nil
	}

	totalShareDec := math.LegacyNewDecFromInt(workerShare)
	totalHeartbeatsDec := math.LegacyNewDecFromInt(math.NewIntFromUint64(totalHeartbeats))
	totalDistributed := math.ZeroInt()

	for _, w := range activeWorkers {
		// Reward proportional to heartbeat count
		workerHeartbeatsDec := math.LegacyNewDecFromInt(math.NewIntFromUint64(w.heartbeatCount))
		reward := workerHeartbeatsDec.Quo(totalHeartbeatsDec).Mul(totalShareDec).TruncateInt()

		if reward.IsPositive() {
			recordKey := fmt.Sprintf("%s/%d", w.addr, epoch)
			record := types.RewardRecord{
				Index:        recordKey,
				RewardAmount: reward.String(),
				Claimed:      false,
				Epoch:        epoch,
			}
			if err := k.WorkerRewardRecord.Set(ctx, recordKey, record); err != nil {
				return math.ZeroInt(), err
			}

			// Update worker's total rewards earned
			worker, err := k.WorkerInfo.Get(ctx, w.addr)
			if err != nil {
				return math.ZeroInt(), err
			}
			prevTotal, _ := math.NewIntFromString(worker.TotalRewardsEarned)
			if prevTotal.IsNil() {
				prevTotal = math.ZeroInt()
			}
			worker.TotalRewardsEarned = prevTotal.Add(reward).String()
			worker.LastActiveEpoch = epoch
			if err := k.WorkerInfo.Set(ctx, w.addr, worker); err != nil {
				return math.ZeroInt(), err
			}

			totalDistributed = totalDistributed.Add(reward)
		}
	}

	return totalDistributed, nil
}

// ClaimRewardsForValidator claims all unclaimed rewards for a validator address.
func (k Keeper) ClaimRewardsForValidator(ctx context.Context, valAddr string) (math.Int, error) {
	totalClaimed := math.ZeroInt()

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return totalClaimed, err
	}

	var toClaim []string
	var amounts []math.Int
	err = k.RewardRecord.Walk(ctx, nil, func(key string, val types.RewardRecord) (stop bool, err error) {
		if !val.Claimed && len(key) > len(valAddr) && key[:len(valAddr)] == valAddr {
			toClaim = append(toClaim, key)
			rewardAmt, _ := math.NewIntFromString(val.RewardAmount)
			if rewardAmt.IsNil() {
				rewardAmt = math.ZeroInt()
			}
			amounts = append(amounts, rewardAmt)
			totalClaimed = totalClaimed.Add(rewardAmt)
		}
		return false, nil
	})
	if err != nil {
		return math.ZeroInt(), err
	}

	if totalClaimed.IsZero() {
		return totalClaimed, types.ErrNoRewardsToClaim
	}

	recipientAddr, err := sdk.AccAddressFromBech32(valAddr)
	if err != nil {
		return math.ZeroInt(), err
	}

	coins := sdk.NewCoins(sdk.NewCoin(bondDenom, totalClaimed))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.RewardPoolName, recipientAddr, coins); err != nil {
		return math.ZeroInt(), err
	}

	for i, key := range toClaim {
		record, err := k.RewardRecord.Get(ctx, key)
		if err != nil {
			return math.ZeroInt(), err
		}
		record.Claimed = true
		_ = amounts[i]
		if err := k.RewardRecord.Set(ctx, key, record); err != nil {
			return math.ZeroInt(), err
		}
	}

	return totalClaimed, nil
}

// ClaimRewardsForWorker claims all unclaimed rewards for a worker address.
func (k Keeper) ClaimRewardsForWorker(ctx context.Context, workerAddr string) (math.Int, error) {
	totalClaimed := math.ZeroInt()

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return totalClaimed, err
	}

	var toClaim []string
	err = k.WorkerRewardRecord.Walk(ctx, nil, func(key string, val types.RewardRecord) (stop bool, err error) {
		if !val.Claimed && len(key) > len(workerAddr) && key[:len(workerAddr)] == workerAddr {
			toClaim = append(toClaim, key)
			rewardAmt, _ := math.NewIntFromString(val.RewardAmount)
			if rewardAmt.IsNil() {
				rewardAmt = math.ZeroInt()
			}
			totalClaimed = totalClaimed.Add(rewardAmt)
		}
		return false, nil
	})
	if err != nil {
		return math.ZeroInt(), err
	}

	if totalClaimed.IsZero() {
		return totalClaimed, types.ErrNoRewardsToClaim
	}

	recipientAddr, err := sdk.AccAddressFromBech32(workerAddr)
	if err != nil {
		return math.ZeroInt(), err
	}

	coins := sdk.NewCoins(sdk.NewCoin(bondDenom, totalClaimed))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.RewardPoolName, recipientAddr, coins); err != nil {
		return math.ZeroInt(), err
	}

	for _, key := range toClaim {
		record, err := k.WorkerRewardRecord.Get(ctx, key)
		if err != nil {
			return math.ZeroInt(), err
		}
		record.Claimed = true
		if err := k.WorkerRewardRecord.Set(ctx, key, record); err != nil {
			return math.ZeroInt(), err
		}
	}

	return totalClaimed, nil
}
