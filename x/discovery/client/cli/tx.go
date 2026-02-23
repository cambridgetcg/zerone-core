package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/discovery/types"
)

// NewTxCmd returns the transaction commands for the discovery module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Discovery module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterProfileCmd(),
		NewUpdateProfileCmd(),
		NewHeartbeatCmd(),
		NewDeregisterProfileCmd(),
	)

	return txCmd
}

// NewRegisterProfileCmd creates a CLI command for MsgRegisterProfile.
func NewRegisterProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-profile [display-name] [stake] --domains [d1,d2] --description [desc] --metadata [meta]",
		Short: "Register a new agent discovery profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			domainsStr, _ := cmd.Flags().GetString("domains")
			var domains []string
			if domainsStr != "" {
				domains = strings.Split(domainsStr, ",")
				for i := range domains {
					domains[i] = strings.TrimSpace(domains[i])
				}
			}

			description, _ := cmd.Flags().GetString("description")
			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgRegisterProfile{
				Sender:      clientCtx.GetFromAddress().String(),
				DisplayName: args[0],
				Stake:       args[1],
				Domains:     domains,
				Description: description,
				Metadata:    metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("domains", "", "Comma-separated list of domains")
	cmd.Flags().String("description", "", "Profile description")
	cmd.Flags().String("metadata", "", "Profile metadata")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateProfileCmd creates a CLI command for MsgUpdateProfile.
func NewUpdateProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-profile --display-name [name] --description [desc] --metadata [meta]",
		Short: "Update an existing agent discovery profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			displayName, _ := cmd.Flags().GetString("display-name")
			description, _ := cmd.Flags().GetString("description")
			metadata, _ := cmd.Flags().GetString("metadata")

			msg := &types.MsgUpdateProfile{
				Sender:      clientCtx.GetFromAddress().String(),
				DisplayName: displayName,
				Description: description,
				Metadata:    metadata,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("display-name", "", "New display name")
	cmd.Flags().String("description", "", "New description")
	cmd.Flags().String("metadata", "", "New metadata")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewHeartbeatCmd creates a CLI command for MsgHeartbeat.
func NewHeartbeatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Send a heartbeat to signal liveness",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgHeartbeat{
				Sender: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDeregisterProfileCmd creates a CLI command for MsgDeregisterProfile.
func NewDeregisterProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deregister-profile",
		Short: "Deregister an agent discovery profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDeregisterProfile{
				Sender: clientCtx.GetFromAddress().String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
