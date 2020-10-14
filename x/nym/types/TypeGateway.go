package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Gateway models a Nym gateway, which acts as an entry/exit point and also provides storage for offline nodes.
type Gateway struct {
	Creator        sdk.AccAddress `json:"creator" yaml:"creator"`
	ID             string         `json:"id" yaml:"id"`
	IdentityKey    string         `json:"identityKey" yaml:"identityKey"`
	SphinxKey      string         `json:"sphinxKey" yaml:"sphinxKey"`
	Layer          int32          `json:"layer" yaml:"layer"`
	ClientListener string         `json:"clientListener" yaml:"clientListener"`
	MixnetListener string         `json:"mixnetListener" yaml:"mixnetListener"`
	Location       string         `json:"location" yaml:"location"`
}
