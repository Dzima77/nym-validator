package nym

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/nymtech/nym/validator/nym/x/nym/types"
	"github.com/nymtech/nym/validator/nym/x/nym/keeper"
)

func handleMsgCreateGateway(ctx sdk.Context, k keeper.Keeper, msg types.MsgCreateGateway) (*sdk.Result, error) {
	var gateway = types.Gateway{
		Creator: msg.Creator,
		ID:      msg.ID,
    	IdentityKey: msg.IdentityKey,
    	SphinxKey: msg.SphinxKey,
    	Layer: msg.Layer,
    	ClientListener: msg.ClientListener,
    	MixnetListener: msg.MixnetListener,
    	Location: msg.Location,
	}
	k.CreateGateway(ctx, gateway)

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}
