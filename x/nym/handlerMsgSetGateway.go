package nym

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/nymtech/nym/validator/nym/x/nym/keeper"
	"github.com/nymtech/nym/validator/nym/x/nym/types"
)

func handleMsgSetGateway(ctx sdk.Context, k keeper.Keeper, msg types.MsgSetGateway) (*sdk.Result, error) {
	var gateway = types.Gateway{
		Creator:        msg.Creator,
		ID:             msg.ID,
		IdentityKey:    msg.IdentityKey,
		SphinxKey:      msg.SphinxKey,
		Layer:          msg.Layer,
		ClientListener: msg.ClientListener,
		MixnetListener: msg.MixnetListener,
		Location:       msg.Location,
	}
	if !msg.Creator.Equals(k.GetGatewayOwner(ctx, msg.ID)) { // Checks if the the msg sender is the same as the current owner
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "Incorrect Owner") // If not, throw an error
	}

	k.SetGateway(ctx, gateway)

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}
