package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/sponsorship/types"
)

func NewTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Sponsorship module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		NewCreateBountyCmd(),
		NewFulfillBountyCmd(),
		NewCancelBountyCmd(),
	)
	return txCmd
}

// NewCreateBountyCmd builds a tx that escrows price × target from the
// signer into the sponsorship module account and records an ACTIVE
// BountyOrder. The signer is the sponsor; the sponsor is its --from address.
func NewCreateBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-bounty [domain] [price-per-artifact-uzrn] [target-count] [duration-blocks]",
		Short: "Escrow funds against a typed bounty for verified work in a domain",
		Long: `Create a bounty: escrow price_per_artifact × target_count uzrn into the
sponsorship module account. Verified facts in [domain] submitted after this
block trigger payouts of [price-per-artifact-uzrn] to the fact's submitter,
up to [target-count] fulfillments, until the [duration-blocks] window expires.

Sponsor cannot override verification — the chain decides what counts (UW M3,
commitment 8). Sponsor can cancel and reclaim remaining escrow at any time.

Example:
  zeroned tx sponsorship create-bounty mathematics 1000000 10 5000 --from sponsor`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			targetCount, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return err
			}
			durationBlocks, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return err
			}
			msg := &types.MsgCreateBountyOrder{
				Sponsor:          clientCtx.GetFromAddress().String(),
				Domain:           args[0],
				PricePerArtifact: args[1],
				TargetCount:      uint32(targetCount),
				DurationBlocks:   durationBlocks,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFulfillBountyCmd builds a tx that pays the submitter of [fact-id]
// the bounty's per-artifact price, provided the fact is verified, in the
// bounty's domain, submitted within the window, and not already fulfilled.
// Anyone can be the caller; the chain reads the worker from fact.Submitter.
func NewFulfillBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fulfill-bounty [bounty-id] [fact-id]",
		Short: "Trigger payout from a bounty to the submitter of a qualifying fact",
		Long: `Permissionless: anyone can trigger fulfillment. The chain enforces all
eligibility checks (bounty active, fact verified, domain matches, fact
submitted after bounty start, not already fulfilled). Payout flows from
bounty escrow to fact.Submitter, NOT to the caller.

Example:
  zeroned tx sponsorship fulfill-bounty bounty-1 fact-abc --from anyone`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgFulfillBounty{
				Caller:   clientCtx.GetFromAddress().String(),
				BountyId: args[0],
				FactId:   args[1],
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCancelBountyCmd builds a tx that closes an ACTIVE or EXPIRED bounty
// and refunds remaining escrow to the sponsor. Only the original sponsor
// can cancel.
func NewCancelBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-bounty [bounty-id]",
		Short: "Close a bounty and reclaim remaining escrow",
		Long: `Only the original sponsor can cancel. Refunds escrow_remaining to the
sponsor. FULFILLED bounties have no remaining escrow; CANCELED bounties
cannot be re-canceled.

Example:
  zeroned tx sponsorship cancel-bounty bounty-1 --from sponsor`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgCancelBountyOrder{
				Sponsor:  clientCtx.GetFromAddress().String(),
				BountyId: args[0],
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
