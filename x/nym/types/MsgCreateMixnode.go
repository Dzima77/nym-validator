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

package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/google/uuid"
)

var _ sdk.Msg = &MsgCreateMixnode{}

type MsgCreateMixnode struct {
	ID       string
	Creator  sdk.AccAddress `json:"creator" yaml:"creator"`
	PubKey   string         `json:"pubKey" yaml:"pubKey"`
	Layer    int32          `json:"layer" yaml:"layer"`
	Version  string         `json:"version" yaml:"version"`
	Host     string         `json:"host" yaml:"host"`
	Location string         `json:"location" yaml:"location"`
	Stake    int32          `json:"stake" yaml:"stake"`
}

func NewMsgCreateMixnode(creator sdk.AccAddress, pubKey string, layer int32, version string, host string, location string, stake int32) MsgCreateMixnode {
	return MsgCreateMixnode{
		ID:       uuid.New().String(),
		Creator:  creator,
		PubKey:   pubKey,
		Layer:    layer,
		Version:  version,
		Host:     host,
		Location: location,
		Stake:    stake,
	}
}

func (msg MsgCreateMixnode) Route() string {
	return RouterKey
}

func (msg MsgCreateMixnode) Type() string {
	return "CreateMixnode"
}

func (msg MsgCreateMixnode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.AccAddress(msg.Creator)}
}

func (msg MsgCreateMixnode) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg MsgCreateMixnode) ValidateBasic() error {
	if msg.Creator.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "creator can't be empty")
	}
	return nil
}
