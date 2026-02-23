package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/toolbox/types"
)

// NewTxCmd returns the transaction commands for the toolbox module.
func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Toolbox module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewRegisterToolCmd(),
		NewCallToolCmd(),
		NewAddContributorCmd(),
		NewAcceptContributorCmd(),
		NewUpgradeToolCmd(),
		NewDeprecateToolCmd(),
		NewRetireToolCmd(),
		NewLockSharesCmd(),
		NewUpdateDependencyCmd(),
		NewToolHeartbeatCmd(),
	)

	return txCmd
}

// NewRegisterToolCmd creates a CLI command for MsgRegisterTool.
func NewRegisterToolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [name] [version] [tool-type] [price-per-call]",
		Short: "Register a new tool in the toolbox",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			description, _ := cmd.Flags().GetString("description")
			contractAddress, _ := cmd.Flags().GetString("contract-address")
			serviceId, _ := cmd.Flags().GetString("service-id")
			targetPriceUsd, _ := cmd.Flags().GetString("target-price-usd")
			minPrice, _ := cmd.Flags().GetString("min-price")
			maxPrice, _ := cmd.Flags().GetString("max-price")
			sourceHash, _ := cmd.Flags().GetString("source-hash")
			apiSchema, _ := cmd.Flags().GetString("api-schema")
			license, _ := cmd.Flags().GetString("license")
			tagsStr, _ := cmd.Flags().GetString("tags")
			depIdsStr, _ := cmd.Flags().GetString("dependency-ids")
			category, _ := cmd.Flags().GetString("category")
			capsStr, _ := cmd.Flags().GetString("required-capabilities")

			var tags []string
			if tagsStr != "" {
				tags = splitAndTrim(tagsStr)
			}

			var depIds []string
			if depIdsStr != "" {
				depIds = splitAndTrim(depIdsStr)
			}

			var caps []string
			if capsStr != "" {
				caps = splitAndTrim(capsStr)
			}

			msg := &types.MsgRegisterTool{
				Deployer:             clientCtx.GetFromAddress().String(),
				Name:                 args[0],
				Description:          description,
				Version:              args[1],
				ToolType:             args[2],
				ContractAddress:      contractAddress,
				ServiceId:            serviceId,
				PricePerCall:         args[3],
				TargetPriceUsd:       targetPriceUsd,
				MinPricePerCall:      minPrice,
				MaxPricePerCall:      maxPrice,
				SourceHash:           sourceHash,
				ApiSchema:            apiSchema,
				License:              license,
				Tags:                 tags,
				DependencyIds:        depIds,
				Category:             category,
				RequiredCapabilities: caps,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("description", "", "Tool description")
	cmd.Flags().String("contract-address", "", "Contract address for the tool")
	cmd.Flags().String("service-id", "", "Service identifier")
	cmd.Flags().String("target-price-usd", "", "Target price in USD")
	cmd.Flags().String("min-price", "", "Minimum price per call")
	cmd.Flags().String("max-price", "", "Maximum price per call")
	cmd.Flags().String("source-hash", "", "Hash of the tool source code")
	cmd.Flags().String("api-schema", "", "API schema definition")
	cmd.Flags().String("license", "", "License type")
	cmd.Flags().String("tags", "", "Comma-separated list of tags")
	cmd.Flags().String("dependency-ids", "", "Comma-separated list of dependency tool IDs")
	cmd.Flags().String("category", "", "Tool category")
	cmd.Flags().String("required-capabilities", "", "Comma-separated list of required capabilities")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCallToolCmd creates a CLI command for MsgCallTool.
func NewCallToolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call [tool-id] [input-hex] [max-fee]",
		Short: "Call a registered tool",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			input, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid hex input: %w", err)
			}

			msg := &types.MsgCallTool{
				Caller: clientCtx.GetFromAddress().String(),
				ToolId: args[0],
				Input:  input,
				MaxFee: args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAddContributorCmd creates a CLI command for MsgAddContributor.
func NewAddContributorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-contributor [tool-id] [contributor-address] [role] [share-bps]",
		Short: "Add a contributor to a tool",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			shareBps, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid share-bps: %w", err)
			}

			msg := &types.MsgAddContributor{
				Authority:          clientCtx.GetFromAddress().String(),
				ToolId:             args[0],
				ContributorAddress: args[1],
				Role:               args[2],
				ShareBps:           shareBps,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAcceptContributorCmd creates a CLI command for MsgAcceptContributorship.
func NewAcceptContributorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accept-contributor [tool-id]",
		Short: "Accept a pending contributorship for a tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgAcceptContributorship{
				ContributorAddress: clientCtx.GetFromAddress().String(),
				ToolId:             args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpgradeToolCmd creates a CLI command for MsgUpgradeTool.
func NewUpgradeToolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade [previous-tool-id] [new-version] [price-per-call]",
		Short: "Upgrade an existing tool to a new version",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			description, _ := cmd.Flags().GetString("description")
			contractAddress, _ := cmd.Flags().GetString("contract-address")
			serviceId, _ := cmd.Flags().GetString("service-id")
			targetPriceUsd, _ := cmd.Flags().GetString("target-price-usd")
			minPrice, _ := cmd.Flags().GetString("min-price")
			maxPrice, _ := cmd.Flags().GetString("max-price")
			sourceHash, _ := cmd.Flags().GetString("source-hash")
			apiSchema, _ := cmd.Flags().GetString("api-schema")
			depIdsStr, _ := cmd.Flags().GetString("dependency-ids")
			migrationNotes, _ := cmd.Flags().GetString("migration-notes")

			var depIds []string
			if depIdsStr != "" {
				depIds = splitAndTrim(depIdsStr)
			}

			msg := &types.MsgUpgradeTool{
				Deployer:        clientCtx.GetFromAddress().String(),
				PreviousToolId:  args[0],
				NewVersion:      args[1],
				Description:     description,
				ContractAddress: contractAddress,
				ServiceId:       serviceId,
				PricePerCall:    args[2],
				TargetPriceUsd:  targetPriceUsd,
				MinPricePerCall: minPrice,
				MaxPricePerCall: maxPrice,
				SourceHash:      sourceHash,
				ApiSchema:       apiSchema,
				DependencyIds:   depIds,
				MigrationNotes:  migrationNotes,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("description", "", "Updated tool description")
	cmd.Flags().String("contract-address", "", "New contract address")
	cmd.Flags().String("service-id", "", "New service identifier")
	cmd.Flags().String("target-price-usd", "", "Target price in USD")
	cmd.Flags().String("min-price", "", "Minimum price per call")
	cmd.Flags().String("max-price", "", "Maximum price per call")
	cmd.Flags().String("source-hash", "", "Hash of the new source code")
	cmd.Flags().String("api-schema", "", "Updated API schema definition")
	cmd.Flags().String("dependency-ids", "", "Comma-separated list of dependency tool IDs")
	cmd.Flags().String("migration-notes", "", "Migration notes for the upgrade")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewDeprecateToolCmd creates a CLI command for MsgDeprecateTool.
func NewDeprecateToolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deprecate [tool-id] [successor-tool-id]",
		Short: "Deprecate a tool in favor of a successor",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgDeprecateTool{
				Authority:       clientCtx.GetFromAddress().String(),
				ToolId:          args[0],
				SuccessorToolId: args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRetireToolCmd creates a CLI command for MsgRetireTool.
func NewRetireToolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retire [tool-id]",
		Short: "Retire a tool permanently",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRetireTool{
				Authority: clientCtx.GetFromAddress().String(),
				ToolId:    args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewLockSharesCmd creates a CLI command for MsgLockShares.
func NewLockSharesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock-shares [tool-id]",
		Short: "Lock contributor shares for a tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgLockShares{
				Deployer: clientCtx.GetFromAddress().String(),
				ToolId:   args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateDependencyCmd creates a CLI command for MsgUpdateDependency.
func NewUpdateDependencyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-dependency [tool-id] [old-dep-id] [new-dep-id]",
		Short: "Replace a tool dependency with a new one",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateDependency{
				Authority: clientCtx.GetFromAddress().String(),
				ToolId:    args[0],
				OldDepId:  args[1],
				NewDepId:  args[2],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewToolHeartbeatCmd creates a CLI command for MsgToolHeartbeat.
func NewToolHeartbeatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat [tool-ids-comma-separated]",
		Short: "Send a heartbeat for active tools",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			activeTools := splitAndTrim(args[0])

			msg := &types.MsgToolHeartbeat{
				Sender:      clientCtx.GetFromAddress().String(),
				ActiveTools: activeTools,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// splitAndTrim splits a comma-separated string and trims whitespace from each element.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
