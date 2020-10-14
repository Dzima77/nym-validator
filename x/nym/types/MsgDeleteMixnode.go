package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDeleteMixnode{}

// MsgDeleteMixnode ...
type MsgDeleteMixnode struct {
	ID      string         `json:"id" yaml:"id"`
	Creator sdk.AccAddress `json:"creator" yaml:"creator"`
}

// NewMsgDeleteMixnode ...
func NewMsgDeleteMixnode(id string, creator sdk.AccAddress) MsgDeleteMixnode {
	return MsgDeleteMixnode{
		ID:      id,
		Creator: creator,
	}
}

// Route ...
func (msg MsgDeleteMixnode) Route() string {
	return RouterKey
}

// Type ...
func (msg MsgDeleteMixnode) Type() string {
	return "DeleteMixnode"
}

// GetSigners ...
func (msg MsgDeleteMixnode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.AccAddress(msg.Creator)}
}

// GetSignBytes ...
func (msg MsgDeleteMixnode) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic ...
func (msg MsgDeleteMixnode) ValidateBasic() error {
	if msg.Creator.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "creator can't be empty")
	}
	return nil
}
