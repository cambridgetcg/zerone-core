package cli

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	anypb "google.golang.org/protobuf/types/known/anypb"

	"github.com/zerone-chain/zerone/x/icaauth/types"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "ICA auth module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterAccountCmd(),
		NewSubmitTxCmd(),
	)
	return txCmd
}

func NewRegisterAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-account [connection-id]",
		Short: "Register an interchain account on a remote chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRegisterAccount{
				Owner:        clientCtx.GetFromAddress().String(),
				ConnectionId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewSubmitTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-tx [connection-id] [msgs-json-file] [timeout-ns]",
		Short: "Submit a transaction via an interchain account",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			connectionID := args[0]

			// Read messages from JSON file
			bz, err := os.ReadFile(args[1])
			if err != nil {
				return err
			}

			var anyMsgs []*codectypes.Any
			if err := json.Unmarshal(bz, &anyMsgs); err != nil {
				return err
			}

			// Convert to protobuf Any
			protoMsgs := make([]*anypb.Any, len(anyMsgs))
			for i, a := range anyMsgs {
				protoMsgs[i] = &anypb.Any{
					TypeUrl: a.TypeUrl,
					Value:   a.Value,
				}
			}

			timeoutNs, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgSubmitTx{
				Owner:        clientCtx.GetFromAddress().String(),
				ConnectionId: connectionID,
				Msgs:         protoMsgs,
				TimeoutNs:    timeoutNs,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
