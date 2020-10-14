package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDeleteGateway{}

type MsgDeleteGateway struct {
  ID      string         `json:"id" yaml:"id"`
  Creator sdk.AccAddress `json:"creator" yaml:"creator"`
}

func NewMsgDeleteGateway(id string, creator sdk.AccAddress) MsgDeleteGateway {
  return MsgDeleteGateway{
    ID: id,
		Creator: creator,
	}
}

func (msg MsgDeleteGateway) Route() string {
  return RouterKey
}

func (msg MsgDeleteGateway) Type() string {
  return "DeleteGateway"
}

func (msg MsgDeleteGateway) GetSigners() []sdk.AccAddress {
  return []sdk.AccAddress{sdk.AccAddress(msg.Creator)}
}

func (msg MsgDeleteGateway) GetSignBytes() []byte {
  bz := ModuleCdc.MustMarshalJSON(msg)
  return sdk.MustSortJSON(bz)
}

func (msg MsgDeleteGateway) ValidateBasic() error {
  if msg.Creator.Empty() {
    return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "creator can't be empty")
  }
  return nil
}