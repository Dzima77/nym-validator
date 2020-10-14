package nym

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/nymtech/nym/validator/nym/x/nym/types"
	"github.com/nymtech/nym/validator/nym/x/nym/keeper"
)

// Handle a message to delete name
func handleMsgDeleteGateway(ctx sdk.Context, k keeper.Keeper, msg types.MsgDeleteGateway) (*sdk.Result, error) {
	if !k.GatewayExists(ctx, msg.ID) {
		// replace with ErrKeyNotFound for 0.39+
		return nil, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, msg.ID)
	}
	if !msg.Creator.Equals(k.GetGatewayOwner(ctx, msg.ID)) {
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "Incorrect Owner")
	}

	k.DeleteGateway(ctx, msg.ID)
	return &sdk.Result{}, nil
}
