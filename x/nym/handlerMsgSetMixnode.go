// Copyright 2020 Nym Technologies SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nym

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/nymtech/nym/validator/nym/x/nym/keeper"
	"github.com/nymtech/nym/validator/nym/x/nym/types"
)

func handleMsgSetMixnode(ctx sdk.Context, k keeper.Keeper, msg types.MsgSetMixnode) (*sdk.Result, error) {
	var mixnode = types.Mixnode{
		Creator:  msg.Creator,
		ID:       msg.ID,
		PubKey:   msg.PubKey,
		Layer:    msg.Layer,
		Version:  msg.Version,
		Host:     msg.Host,
		Location: msg.Location,
		Stake:    msg.Stake,
	}
	if !msg.Creator.Equals(k.GetMixnodeOwner(ctx, msg.ID)) { // Checks if the the msg sender is the same as the current owner
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "Incorrect Owner") // If not, throw an error
	}

	k.SetMixnode(ctx, mixnode)

	return &sdk.Result{Events: ctx.EventManager().Events()}, nil
}
