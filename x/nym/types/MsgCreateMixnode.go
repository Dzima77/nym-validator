package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/google/uuid"
)

var _ sdk.Msg = &MsgCreateMixnode{}

// MsgCreateMixnode ...
type MsgCreateMixnode struct {
	ID         string
	Creator    sdk.AccAddress `json:"creator" yaml:"creator"`
	PubKey     string         `json:"pubKey" yaml:"pubKey"`
	Layer      int32          `json:"layer" yaml:"layer"`
	Version    string         `json:"version" yaml:"version"`
	Host       string         `json:"host" yaml:"host"`
	Location   string         `json:"location" yaml:"location"`
	Reputation int32          `json:"reputation" yaml:"reputation"`
}

// NewMsgCreateMixnode constructor for MsgCreateMixnode
func NewMsgCreateMixnode(creator sdk.AccAddress, pubKey string, layer int32, version string, host string, location string, reputation int32) MsgCreateMixnode {
	return MsgCreateMixnode{
		ID:         uuid.New().String(),
		Creator:    creator,
		PubKey:     pubKey,
		Layer:      layer,
		Version:    version,
		Host:       host,
		Location:   location,
		Reputation: reputation,
	}
}

// Route ...
func (msg MsgCreateMixnode) Route() string {
	return RouterKey
}

// Type ...
func (msg MsgCreateMixnode) Type() string {
	return "CreateMixnode"
}

// GetSigners ...
func (msg MsgCreateMixnode) GetSigners() []sdk.AccAddress {
	return []sdk.AccAddress{sdk.AccAddress(msg.Creator)}
}

// GetSignBytes ...
func (msg MsgCreateMixnode) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic ...
func (msg MsgCreateMixnode) ValidateBasic() error {
	if msg.Creator.Empty() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "creator can't be empty")
	}
	return nil
}
