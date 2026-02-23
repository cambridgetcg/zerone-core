package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/zerone-chain/zerone/x/channels/types"
)

// NewQueryCmd returns the query commands for the channels module.
func NewQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Channels module query commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		NewQueryParamsCmd(),
		NewQueryChannelCmd(),
		NewQueryChannelsByPayerCmd(),
		NewQueryChannelsByReceiverCmd(),
		NewQueryDisputeCmd(),
	)

	return queryCmd
}

// NewQueryParamsCmd returns the command to query channels module params.
func NewQueryParamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the channels module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryParamsRequest{}
			resp := &types.QueryParamsResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.channels.v1.Query/Params", req, resp); err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryChannelCmd returns the command to query a specific channel.
func NewQueryChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channel [channel-id]",
		Short: "Query a specific payment channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryChannelRequest{ChannelId: args[0]}
			resp := &types.QueryChannelResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.channels.v1.Query/Channel", req, resp); err != nil {
				return fmt.Errorf("failed to query channel: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryChannelsByPayerCmd returns the command to query channels by payer.
func NewQueryChannelsByPayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-payer [payer-address]",
		Short: "Query all payment channels for a payer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryByPayerRequest{Payer: args[0]}
			resp := &types.QueryByPayerResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.channels.v1.Query/ChannelsByPayer", req, resp); err != nil {
				return fmt.Errorf("failed to query channels by payer: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryChannelsByReceiverCmd returns the command to query channels by receiver.
func NewQueryChannelsByReceiverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-receiver [receiver-address]",
		Short: "Query all payment channels for a receiver",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryByReceiverRequest{Receiver: args[0]}
			resp := &types.QueryByReceiverResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.channels.v1.Query/ChannelsByReceiver", req, resp); err != nil {
				return fmt.Errorf("failed to query channels by receiver: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// NewQueryDisputeCmd returns the command to query a channel dispute.
func NewQueryDisputeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispute [channel-id]",
		Short: "Query the dispute for a payment channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			req := &types.QueryDisputeRequest{ChannelId: args[0]}
			resp := &types.QueryDisputeResponse{}
			if err := clientCtx.Invoke(cmd.Context(), "/zerone.channels.v1.Query/Dispute", req, resp); err != nil {
				return fmt.Errorf("failed to query dispute: %w", err)
			}

			return clientCtx.PrintObjectLegacy(resp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
