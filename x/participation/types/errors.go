package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/participation module sentinel errors
var (
	ErrInvalidSigner       = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrNoRewardsToClaim    = errors.Register(ModuleName, 1101, "no unclaimed rewards found")
	ErrValidatorNotFound   = errors.Register(ModuleName, 1102, "validator not found")
	ErrMetricsNotFound     = errors.Register(ModuleName, 1103, "validator metrics not found")
	ErrInvalidAddress      = errors.Register(ModuleName, 1104, "invalid address")
	ErrRewardPoolEmpty     = errors.Register(ModuleName, 1105, "reward pool is empty")
	ErrEpochNotInitialized = errors.Register(ModuleName, 1106, "epoch info not initialized")
	ErrInsufficientStake       = errors.Register(ModuleName, 1107, "insufficient stake to participate")
	ErrWorkerAlreadyRegistered = errors.Register(ModuleName, 1108, "worker already registered")
	ErrWorkerNotFound          = errors.Register(ModuleName, 1109, "worker not found")
	ErrHeartbeatTooEarly       = errors.Register(ModuleName, 1110, "heartbeat too early, must wait for interval")
	ErrWorkerInactive          = errors.Register(ModuleName, 1111, "worker is inactive")
	ErrMaxWorkersReached       = errors.Register(ModuleName, 1112, "maximum number of registered workers reached")
)
