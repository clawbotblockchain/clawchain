package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"github.com/clawbotblockchain/clawchain/x/participation/types"
)

func (k msgServer) ClaimRewards(ctx context.Context, msg *types.MsgClaimRewards) (*types.MsgClaimRewardsResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidAddress, "invalid creator address")
	}

	claimed, err := k.Keeper.ClaimRewardsForValidator(ctx, msg.Creator)
	if err != nil {
		return nil, err
	}

	return &types.MsgClaimRewardsResponse{
		ClaimedAmount: claimed.String(),
	}, nil
}
