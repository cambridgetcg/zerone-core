package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/schedule/types"
)

func NewQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.CLIName,
		Short:                      "Query durable transfer schedules and receipts",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(newScheduleQueryCmd(), newCreatorSchedulesQueryCmd(), newReceiptQueryCmd(), newReceiptsQueryCmd(), newParamsQueryCmd())
	return cmd
}

func newScheduleQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule [schedule-id]",
		Short: "Query a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			response := new(types.QueryScheduleResponse)
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.schedule.v2.Query/Schedule", &types.QueryScheduleRequest{Id: args[0]}, response); err != nil {
				return fmt.Errorf("query schedule: %w", err)
			}
			return clientCtx.PrintObjectLegacy(response)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func newCreatorSchedulesQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-creator [creator]",
		Short: "List schedules created by an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			startAfter, _ := cmd.Flags().GetString("start-after")
			limit, _ := cmd.Flags().GetUint32("limit")
			response := new(types.QuerySchedulesByCreatorResponse)
			request := &types.QuerySchedulesByCreatorRequest{Creator: args[0], StartAfter: startAfter, Limit: limit}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.schedule.v2.Query/SchedulesByCreator", request, response); err != nil {
				return fmt.Errorf("query creator schedules: %w", err)
			}
			return clientCtx.PrintObjectLegacy(response)
		},
	}
	cmd.Flags().String("start-after", "", "exclusive schedule id cursor")
	cmd.Flags().Uint32("limit", 0, "result limit (0 uses the server default)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func newReceiptQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "receipt [occurrence-id]",
		Short: "Query an immutable execution receipt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			response := new(types.QueryReceiptResponse)
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.schedule.v2.Query/Receipt", &types.QueryReceiptRequest{OccurrenceId: args[0]}, response); err != nil {
				return fmt.Errorf("query receipt: %w", err)
			}
			return clientCtx.PrintObjectLegacy(response)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func newReceiptsQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "receipts [schedule-id]",
		Short: "List execution receipts for a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			startText, _ := cmd.Flags().GetString("start-sequence")
			start, err := strconv.ParseUint(startText, 10, 32)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetUint32("limit")
			response := new(types.QueryReceiptsByScheduleResponse)
			request := &types.QueryReceiptsByScheduleRequest{ScheduleId: args[0], StartSequence: uint32(start), Limit: limit}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.schedule.v2.Query/ReceiptsBySchedule", request, response); err != nil {
				return fmt.Errorf("query receipts: %w", err)
			}
			return clientCtx.PrintObjectLegacy(response)
		},
	}
	cmd.Flags().String("start-sequence", "1", "inclusive execution sequence cursor")
	cmd.Flags().Uint32("limit", 0, "result limit (0 uses the server default)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func newParamsQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query scheduler parameters and admission state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			response := new(types.QueryParamsResponse)
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.schedule.v2.Query/Params", &types.QueryParamsRequest{}, response); err != nil {
				return fmt.Errorf("query params: %w", err)
			}
			return clientCtx.PrintObjectLegacy(response)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
