package cli

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	"github.com/zerone-chain/zerone/x/auth/types"
)

// GetTxCmd returns the transaction commands for this module.
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Zerone auth transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		CmdRegisterAccount(),
		CmdRotateKey(),
		CmdFreezeAccount(),
		CmdUnfreezeAccount(),
	)

	return txCmd
}

// CmdRegisterAccount registers a new Zerone account.
func CmdRegisterAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-account [did] [public-key] [account-type]",
		Short: "Register a new Zerone account with DID mapping",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			opKeyHash, _ := cmd.Flags().GetString("operational-key-hash")
			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgRegisterAccount{
				Sender:             clientCtx.GetFromAddress().String(),
				Did:                args[0],
				PublicKey:          args[1],
				AccountType:        args[2],
				OperationalKeyHash: opKeyHash,
				Metadata:           metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("operational-key-hash", "", "Hash of initial operational key")
	cmd.Flags().String("metadata", "", "Account metadata (JSON string)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdRotateKey rotates the operational key.
func CmdRotateKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate-key [new-op-key-hex] [auth-sig-hex]",
		Short: "Rotate operational key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newKey, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid new key hex: %w", err)
			}

			authSig, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid auth signature hex: %w", err)
			}

			msg := &types.MsgRotateKey{
				Sender:                 clientCtx.GetFromAddress().String(),
				NewOperationalKey:      newKey,
				AuthorizationSignature: authSig,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdFreezeAccount freezes an account.
func CmdFreezeAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freeze-account [address]",
		Short: "Freeze an account (self or authority)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			reason, _ := cmd.Flags().GetString("reason")

			msg := &types.MsgFreezeAccount{
				Sender:  clientCtx.GetFromAddress().String(),
				Address: args[0],
				Reason:  reason,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("reason", "", "Reason for freezing")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CmdUnfreezeAccount unfreezes an account.
func CmdUnfreezeAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfreeze-account [address]",
		Short: "Unfreeze a frozen account (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUnfreezeAccount{
				Authority: clientCtx.GetFromAddress().String(),
				Address:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

