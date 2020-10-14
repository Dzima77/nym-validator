package cli

import (
	"bufio"
	"github.com/spf13/cobra"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	"github.com/nymtech/nym/validator/nym/x/nym/types"
)

func GetCmdCreateGateway(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "create-gateway [identityKey] [sphinxKey] [layer] [clientListener] [mixnetListener] [location]",
		Short: "Creates a new gateway",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			argsIdentityKey := string(args[0])
			argsSphinxKey := string(args[1])
			argsLayer, _ := strconv.ParseInt(args[2], 10, 64)
			argsClientListener := string(args[3])
			argsMixnetListener := string(args[4])
			argsLocation := string(args[5])

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			msg := types.NewMsgCreateGateway(cliCtx.GetFromAddress(), string(argsIdentityKey), string(argsSphinxKey), int32(argsLayer), string(argsClientListener), string(argsMixnetListener), string(argsLocation))
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
		Use:   "set-gateway [id]  [identityKey] [sphinxKey] [layer] [clientListener] [mixnetListener] [location]",
		Short: "Set a new gateway",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			argsIdentityKey := string(args[1])
			argsSphinxKey := string(args[2])
			argsLayer, _ := strconv.ParseInt(args[3], 10, 64)
			argsClientListener := string(args[4])
			argsMixnetListener := string(args[5])
			argsLocation := string(args[6])

			cliCtx := context.NewCLIContext().WithCodec(cdc)
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			msg := types.NewMsgSetGateway(cliCtx.GetFromAddress(), id, string(argsIdentityKey), string(argsSphinxKey), int32(argsLayer), string(argsClientListener), string(argsMixnetListener), string(argsLocation))
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
