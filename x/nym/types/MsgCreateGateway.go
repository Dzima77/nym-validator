package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/google/uuid"
)

var _ sdk.Msg = &MsgCreateGateway{}

type MsgCreateGateway struct {
  ID      string
  Creator sdk.AccAddress `json:"creator" yaml:"creator"`
  IdentityKey string `json:"identityKey" yaml:"identityKey"`
  SphinxKey string `json:"sphinxKey" yaml:"sphinxKey"`
  Layer int32 `json:"layer" yaml:"layer"`
  ClientListener string `json:"clientListener" yaml:"clientListener"`
  MixnetListener string `json:"mixnetListener" yaml:"mixnetListener"`
  Location string `json:"location" yaml:"location"`
}

func NewMsgCreateGateway(creator sdk.AccAddress, identityKey string, sphinxKey string, layer int32, clientListener string, mixnetListener string, location string) MsgCreateGateway {
  return MsgCreateGateway{
    ID: uuid.New().String(),
		Creator: creator,
    IdentityKey: identityKey,
    SphinxKey: sphinxKey,
    Layer: layer,
    ClientListener: clientListener,
    MixnetListener: mixnetListener,
    Location: location,
	}
}

func (msg MsgCreateGateway) Route() string {
  return RouterKey
}

func (msg MsgCreateGateway) Type() string {
  return "CreateGateway"
}

func (msg MsgCreateGateway) GetSigners() []sdk.AccAddress {
  return []sdk.AccAddress{sdk.AccAddress(msg.Creator)}
}

func (msg MsgCreateGateway) GetSignBytes() []byte {
  bz := ModuleCdc.MustMarshalJSON(msg)
  return sdk.MustSortJSON(bz)
}

func (msg MsgCreateGateway) ValidateBasic() error {
  if msg.Creator.Empty() {
    return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "creator can't be empty")
  }
  return nil
}