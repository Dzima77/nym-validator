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
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	"github.com/nymtech/nym/validator/nym/x/nym/types"
)

func GetCmdCreateMixnode(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "create-mixnode [pubKey] [layer] [version] [host] [location] [reputation]",
		Short: "Creates a new mixnode",
		Args:  cobra.MinimumNArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			argsPubKey := string(args[0])
			argsLayer, _ := strconv.ParseInt(args[1], 10, 64)
			argsVersion := string(args[2])
			argsHost := string(args[3])
			argsLocation := string(args[4])
			argsReputation, _ := strconv.ParseInt(args[5], 10, 64)

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			msg := types.NewMsgCreateMixnode(cliCtx.GetFromAddress(), string(argsPubKey), int32(argsLayer), string(argsVersion), string(argsHost), string(argsLocation), int32(argsReputation))
			err := msg.ValidateBasic()
			if err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
}

func GetCmdSetMixnode(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "set-mixnode [id]  [pubKey] [layer] [version] [host] [location] [reputation]",
		Short: "Set a new mixnode",
		Args:  cobra.MinimumNArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			argsPubKey := string(args[1])
			argsLayer, _ := strconv.ParseInt(args[2], 10, 64)
			argsVersion := string(args[3])
			argsHost := string(args[4])
			argsLocation := string(args[5])
			argsReputation, _ := strconv.ParseInt(args[6], 10, 64)

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			msg := types.NewMsgSetMixnode(cliCtx.GetFromAddress(), id, string(argsPubKey), int32(argsLayer), string(argsVersion), string(argsHost), string(argsLocation), int32(argsReputation))
			err := msg.ValidateBasic()
			if err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
}

func GetCmdDeleteMixnode(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "delete-mixnode [id]",
		Short: "Delete a new mixnode by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))

			msg := types.NewMsgDeleteMixnode(args[0], cliCtx.GetFromAddress())
			err := msg.ValidateBasic()
			if err != nil {
				return err
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
}
