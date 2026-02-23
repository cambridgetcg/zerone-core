package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/home/types"
)

// NewTxCmd returns the transaction commands for the home module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Home module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateHomeCmd(),
		NewUpdateHomeCmd(),
		NewRegisterKeyCmd(),
		NewRevokeKeyCmd(),
		NewStartSessionCmd(),
		NewEndSessionCmd(),
		NewSetSpendingLimitCmd(),
	)

	return txCmd
}

// NewCreateHomeCmd creates a CLI command for MsgCreateHome.
func NewCreateHomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-home [name]",
		Short: "Create a new agent home",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCreateHome{
				Owner: clientCtx.GetFromAddress().String(),
				Name:  args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateHomeCmd creates a CLI command for MsgUpdateHome.
func NewUpdateHomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-home [home-id] --name [name] --status [status]",
		Short: "Update an agent home",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			name, _ := cmd.Flags().GetString("name")
			status, _ := cmd.Flags().GetString("status")

			msg := &types.MsgUpdateHome{
				Owner:  clientCtx.GetFromAddress().String(),
				HomeId: args[0],
				Name:   name,
				Status: status,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("name", "", "New home name")
	cmd.Flags().String("status", "", "New home status")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRegisterKeyCmd creates a CLI command for MsgRegisterKey.
func NewRegisterKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-key [home-id] [key-hash] [key-type] [role] [permissions]",
		Short: "Register a key for a home",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			permissions := strings.Split(args[4], ",")
			for i := range permissions {
				permissions[i] = strings.TrimSpace(permissions[i])
			}

			msg := &types.MsgRegisterKey{
				Owner:       clientCtx.GetFromAddress().String(),
				HomeId:      args[0],
				KeyHash:     args[1],
				KeyType:     args[2],
				Role:        args[3],
				Permissions: permissions,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRevokeKeyCmd creates a CLI command for MsgRevokeKey.
func NewRevokeKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-key [home-id] [key-hash]",
		Short: "Revoke a registered key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRevokeKey{
				Owner:   clientCtx.GetFromAddress().String(),
				HomeId:  args[0],
				KeyHash: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewStartSessionCmd creates a CLI command for MsgStartSession.
func NewStartSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start-session [home-id] [key-hash] [permissions]",
		Short: "Start a session for a registered key",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			permissions := strings.Split(args[2], ",")
			for i := range permissions {
				permissions[i] = strings.TrimSpace(permissions[i])
			}

			msg := &types.MsgStartSession{
				Signer:               clientCtx.GetFromAddress().String(),
				HomeId:               args[0],
				KeyHash:              args[1],
				RequestedPermissions: permissions,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewEndSessionCmd creates a CLI command for MsgEndSession.
func NewEndSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "end-session [home-id] [session-id]",
		Short: "End an active session",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEndSession{
				Signer:    clientCtx.GetFromAddress().String(),
				HomeId:    args[0],
				SessionId: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSetSpendingLimitCmd creates a CLI command for MsgSetSpendingLimit.
func NewSetSpendingLimitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-spending-limit [home-id] [key-type] [max-amount] [period-blocks]",
		Short: "Set a spending limit for a key type",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			periodBlocks, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgSetSpendingLimit{
				Owner:        clientCtx.GetFromAddress().String(),
				HomeId:       args[0],
				KeyType:      args[1],
				MaxAmount:    args[2],
				PeriodBlocks: periodBlocks,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
