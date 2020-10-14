package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSetGateway{}

type MsgSetGateway struct {
  ID      string      `json:"id" yaml:"id"`
  Creator sdk.AccAddress `json:"creator" yaml:"creator"`
  IdentityKey string `json:"identityKey" yaml:"identityKey"`
  SphinxKey string `json:"sphinxKey" yaml:"sphinxKey"`
  Layer int32 `json:"layer" yaml:"layer"`
  ClientListener string `json:"clientListener" yaml:"clientListener"`
  MixnetListener string `json:"mixnetListener" yaml:"mixnetListener"`
  Location string `json:"location" yaml:"location"`
}

func NewMsgSetGateway(creator sdk.AccAddress, id string, identityKey string, sphinxKey string, layer int32, clientListener string, mixnetListener string, location string) MsgSetGateway {
  return MsgSetGateway{
    ID: id,
		Creator: creator,
    IdentityKey: identityKey,
    SphinxKey: sphinxKey,
    Layer: layer,
    ClientListener: clientListener,
    MixnetListener: mixnetListener,
    Location: location,
	}
}

func (msg MsgSetGateway) Route() string {
  return RouterKey
}

func (msg MsgSetGateway) Type() string {
  return "SetGateway"
}

func (msg MsgSetGateway) GetSigners() []sdk.AccAddress {
  return []sdk.AccAddress{sdk.AccAddress(msg.Creator)}
}

func (msg MsgSetGateway) GetSignBytes() []byte {
  bz := ModuleCdc.MustMarshalJSON(msg)
  return sdk.MustSortJSON(bz)
}

func (msg MsgSetGateway) ValidateBasic() error {
  if msg.Creator.Empty() {
    return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "creator can't be empty")
  }
  return nil
}