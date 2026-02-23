package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/tree/types"
)

// NewTxCmd returns the transaction commands for the tree module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Tree module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewCreateProjectCmd(),
		NewAddTaskCmd(),
		NewDeployServiceCmd(),
		NewCallServiceCmd(),
	)

	return txCmd
}

// NewCreateProjectCmd creates a CLI command for MsgCreateProject.
func NewCreateProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-project [name] [description]",
		Short: "Create a new project in the tree",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCreateProject{
				Creator:     clientCtx.GetFromAddress().String(),
				Title:       args[0],
				Description: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAddTaskCmd creates a CLI command for MsgAddTask.
func NewAddTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-task [project-id] [title] [description] [bounty-amount]",
		Short: "Add a task to a project with a bounty",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAddTask{
				Creator:     clientCtx.GetFromAddress().String(),
				ProjectId:   args[0],
				Title:       args[1],
				Description: args[2],
				Bounty:      args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDeployServiceCmd creates a CLI command for MsgDeployService.
func NewDeployServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy-service [project-id] [name] [description] [endpoint] [price-per-call]",
		Short: "Deploy a service leaf on a project",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDeployService{
				Deployer:     clientCtx.GetFromAddress().String(),
				Name:         args[0],
				Description:  args[1],
				Endpoint:     args[2],
				PricePerCall: args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCallServiceCmd creates a CLI command for MsgCallService.
func NewCallServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call-service [service-id] [input-data] [payment-type]",
		Short: "Call a deployed service (payment-type: direct, subscription, channel)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgCallService{
				Caller:    clientCtx.GetFromAddress().String(),
				ServiceId: args[0],
				Payload:   []byte(args[1]),
				MaxFee:    args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
