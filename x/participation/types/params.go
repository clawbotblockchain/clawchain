package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Default parameter values
var (
	DefaultEpochDuration      uint64 = 86400 // 24 hours in seconds
	DefaultMinStake                  = "0"    // Set in genesis (math.Int for 18 decimals)
	DefaultStakeWeight        uint64 = 20
	DefaultActivityWeight     uint64 = 60
	DefaultUptimeWeight       uint64 = 20
	DefaultWorkerRewardRatio   uint64 = 60   // 60% of daily rewards go to workers
	DefaultHeartbeatInterval   uint64 = 300  // 5 minutes
	DefaultMaxMissedHeartbeats uint64 = 100  // ~8.3 hours of inactivity
	DefaultMaxWorkers          uint64 = 1000 // max registered workers (0 = unlimited)
)

// NewParams creates a new Params instance.
func NewParams(
	epochDuration uint64,
	minStake string,
	stakeWeight uint64,
	activityWeight uint64,
	uptimeWeight uint64,
	rewardPoolAddress string,
	dailyRewardAmount string,
	workerRewardRatio uint64,
	heartbeatInterval uint64,
	maxMissedHeartbeats uint64,
	maxWorkers uint64,
) Params {
	return Params{
		EpochDuration:       epochDuration,
		MinStake:            minStake,
		StakeWeight:         stakeWeight,
		ActivityWeight:      activityWeight,
		UptimeWeight:        uptimeWeight,
		RewardPoolAddress:   rewardPoolAddress,
		DailyRewardAmount:   dailyRewardAmount,
		WorkerRewardRatio:   workerRewardRatio,
		HeartbeatInterval:   heartbeatInterval,
		MaxMissedHeartbeats: maxMissedHeartbeats,
		MaxWorkers:          maxWorkers,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(
		DefaultEpochDuration,
		DefaultMinStake,
		DefaultStakeWeight,
		DefaultActivityWeight,
		DefaultUptimeWeight,
		"",  // reward pool address set in genesis
		"0", // daily reward amount set in genesis
		DefaultWorkerRewardRatio,
		DefaultHeartbeatInterval,
		DefaultMaxMissedHeartbeats,
		DefaultMaxWorkers,
	)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if err := validateEpochDuration(p.EpochDuration); err != nil {
		return err
	}
	if err := validateWeights(p.StakeWeight, p.ActivityWeight, p.UptimeWeight); err != nil {
		return err
	}
	if p.MinStake != "" && p.MinStake != "0" {
		amt, ok := math.NewIntFromString(p.MinStake)
		if !ok {
			return fmt.Errorf("invalid min_stake: %s", p.MinStake)
		}
		if amt.IsNegative() {
			return fmt.Errorf("min_stake cannot be negative")
		}
	}
	if p.DailyRewardAmount != "" && p.DailyRewardAmount != "0" {
		amt, ok := math.NewIntFromString(p.DailyRewardAmount)
		if !ok {
			return fmt.Errorf("invalid daily_reward_amount: %s", p.DailyRewardAmount)
		}
		if amt.IsNegative() {
			return fmt.Errorf("daily_reward_amount cannot be negative")
		}
	}
	if p.WorkerRewardRatio > 100 {
		return fmt.Errorf("worker_reward_ratio must be 0-100, got %d", p.WorkerRewardRatio)
	}
	if p.HeartbeatInterval == 0 {
		return fmt.Errorf("heartbeat_interval must be positive")
	}
	if p.MaxMissedHeartbeats == 0 {
		return fmt.Errorf("max_missed_heartbeats must be positive")
	}
	return nil
}

func validateEpochDuration(v uint64) error {
	if v == 0 {
		return fmt.Errorf("epoch_duration must be positive")
	}
	return nil
}

func validateWeights(stake, activity, uptime uint64) error {
	total := stake + activity + uptime
	if total != 100 {
		return fmt.Errorf("stake_weight + activity_weight + uptime_weight must equal 100, got %d", total)
	}
	return nil
}
