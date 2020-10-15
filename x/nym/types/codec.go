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
	"github.com/cosmos/cosmos-sdk/codec"
)

// RegisterCodec registers concrete types on codec
func RegisterCodec(cdc *codec.Codec) {
	// this line is used by starport scaffolding # 1
	cdc.RegisterConcrete(MsgCreateGateway{}, "nym/CreateGateway", nil)
	cdc.RegisterConcrete(MsgSetGateway{}, "nym/SetGateway", nil)
	cdc.RegisterConcrete(MsgDeleteGateway{}, "nym/DeleteGateway", nil)
	cdc.RegisterConcrete(MsgCreateMixnode{}, "nym/CreateMixnode", nil)
	cdc.RegisterConcrete(MsgSetMixnode{}, "nym/SetMixnode", nil)
	cdc.RegisterConcrete(MsgDeleteMixnode{}, "nym/DeleteMixnode", nil)
}

// ModuleCdc defines the module codec
var ModuleCdc *codec.Codec

func init() {
	ModuleCdc = codec.New()
	RegisterCodec(ModuleCdc)
	codec.RegisterCrypto(ModuleCdc)
	ModuleCdc.Seal()
}
