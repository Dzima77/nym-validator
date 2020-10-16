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

package cli

import (
	"bufio"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	"github.com/nymtech/nym/validator/nym/x/nym/types"
)

func GetCmdCreateGateway(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "create-gateway [identityKey] [sphinxKey] [clientListener] [mixnetListener] [location]",
		Short: "Creates a new gateway",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			argsIdentityKey := string(args[0])
			argsSphinxKey := string(args[1])
			argsClientListener := string(args[2])
			argsMixnetListener := string(args[3])
			argsLocation := string(args[4])

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			msg := types.NewMsgCreateGateway(cliCtx.GetFromAddress(), string(argsIdentityKey), string(argsSphinxKey), string(argsClientListener), string(argsMixnetListener), string(argsLocation))
			err := msg.ValidateBasic()
			if err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
}

func GetCmdSetGateway(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "set-gateway [id]  [identityKey] [sphinxKey] [clientListener] [mixnetListener] [location]",
		Short: "Set a new gateway",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			argsIdentityKey := string(args[1])
			argsSphinxKey := string(args[2])
			argsClientListener := string(args[3])
			argsMixnetListener := string(args[4])
			argsLocation := string(args[5])

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			msg := types.NewMsgSetGateway(cliCtx.GetFromAddress(), id, string(argsIdentityKey), string(argsSphinxKey), string(argsClientListener), string(argsMixnetListener), string(argsLocation))
			err := msg.ValidateBasic()
			if err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
}

func GetCmdDeleteGateway(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-gateway [id]",
		Short: "Delete a new gateway by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))

			msg := types.NewMsgDeleteGateway(args[0], cliCtx.GetFromAddress())
			err := msg.ValidateBasic()
			if err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
}
