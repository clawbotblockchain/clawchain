package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:              DefaultParams(),
		ValidatorMetricsMap: []ValidatorMetrics{},
		EpochInfo:           nil,
		RewardRecordMap:     []RewardRecord{},
		WorkerMap:           []WorkerInfo{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	validatorMetricsIndexMap := make(map[string]struct{})

	for _, elem := range gs.ValidatorMetricsMap {
		index := fmt.Sprint(elem.Index)
		if _, ok := validatorMetricsIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for validatorMetrics")
		}
		validatorMetricsIndexMap[index] = struct{}{}
	}
	rewardRecordIndexMap := make(map[string]struct{})

	for _, elem := range gs.RewardRecordMap {
		index := fmt.Sprint(elem.Index)
		if _, ok := rewardRecordIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for rewardRecord")
		}
		rewardRecordIndexMap[index] = struct{}{}
	}

	workerIndexMap := make(map[string]struct{})
	for _, elem := range gs.WorkerMap {
		index := fmt.Sprint(elem.Index)
		if _, ok := workerIndexMap[index]; ok {
			return fmt.Errorf("duplicated index for worker")
		}
		workerIndexMap[index] = struct{}{}
	}

	return gs.Params.Validate()
}
