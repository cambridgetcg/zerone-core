package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/vesting_rewards/types"
)

// NewQueryCmd returns the query commands for the vesting_rewards module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Vesting rewards module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryVestingScheduleCmd(),
		NewQuerySchedulesByRecipientCmd(),
		NewQueryActiveSchedulesCmd(),
		NewQueryBlockRewardCmd(),
		NewQueryParamsCmd(),
		NewQueryResearchFundBalanceCmd(),
		NewQueryFounderShareStatusCmd(),
	)

	return queryCmd
}

// NewQueryVestingScheduleCmd creates a CLI command to query a vesting schedule by ID.
func NewQueryVestingScheduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule [vesting-id]",
		Short: "Query a vesting schedule by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryVestingScheduleRequest{VestingId: args[0]}
			resp := &types.QueryVestingScheduleResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.vesting_rewards.v1.Query/VestingSchedule", req, resp); err != nil {
				return fmt.Errorf("failed to query vesting schedule: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQuerySchedulesByRecipientCmd creates a CLI command to query vesting schedules by recipient.
func NewQuerySchedulesByRecipientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedules-by-recipient [recipient]",
		Short: "Query all vesting schedules for a recipient address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryVestingSchedulesByRecipientRequest{Recipient: args[0]}
			resp := &types.QueryVestingSchedulesByRecipientResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.vesting_rewards.v1.Query/VestingSchedulesByRecipient", req, resp); err != nil {
				return fmt.Errorf("failed to query schedules by recipient: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryActiveSchedulesCmd creates a CLI command to query active vesting schedules.
func NewQueryActiveSchedulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "active-schedules",
		Short: "Query all active vesting schedules with optional limit and offset",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			limit, _ := cmd.Flags().GetUint32("limit")
			offset, _ := cmd.Flags().GetUint32("offset")

			req := &types.QueryActiveVestingSchedulesRequest{
				Limit:  limit,
				Offset: offset,
			}
			resp := &types.QueryActiveVestingSchedulesResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.vesting_rewards.v1.Query/ActiveVestingSchedules", req, resp); err != nil {
				return fmt.Errorf("failed to query active schedules: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	cmd.Flags().Uint32("limit", 0, "Maximum number of schedules to return")
	cmd.Flags().Uint32("offset", 0, "Number of schedules to skip")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryBlockRewardCmd creates a CLI command to query block reward distribution.
func NewQueryBlockRewardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block-reward [block-height]",
		Short: "Query block reward distribution at a specific height",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			blockHeight, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid block height: %w", err)
			}

			req := &types.QueryBlockRewardDistributionRequest{BlockHeight: blockHeight}
			resp := &types.QueryBlockRewardDistributionResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.vesting_rewards.v1.Query/BlockRewardDistribution", req, resp); err != nil {
				return fmt.Errorf("failed to query block reward distribution: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryParamsCmd creates a CLI command to query vesting_rewards module parameters.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the vesting rewards module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.vesting_rewards.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryResearchFundBalanceCmd creates a CLI command to query the research fund balance.
func NewQueryResearchFundBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "research-fund-balance",
		Short: "Query the current research fund module account balance",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryResearchFundBalanceRequest{}
			resp := &types.QueryResearchFundBalanceResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.vesting_rewards.v1.Query/ResearchFundBalance", req, resp); err != nil {
				return fmt.Errorf("failed to query research fund balance: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryFounderShareStatusCmd creates a CLI command to query the founder share status.
func NewQueryFounderShareStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "founder-share-status",
		Short: "Query the current founder share configuration and activation status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryFounderShareStatusRequest{}
			resp := &types.QueryFounderShareStatusResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.vesting_rewards.v1.Query/FounderShareStatus", req, resp); err != nil {
				return fmt.Errorf("failed to query founder share status: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
