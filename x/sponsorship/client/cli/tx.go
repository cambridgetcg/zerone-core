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
The fact submitter must be the wallet preassigned by --worker-address.

Sponsor cannot override verification — the chain decides what counts (UW M3,
commitment 8). A bound v2 bounty cannot be canceled before its deadline;
after expiry the sponsor can reclaim any remaining escrow.

Every digest flag is required and uses bare lowercase 64-hex SHA-256. The work
spec must include the exact task and input semantics; acceptance commits the
evaluator/policy. Raw material stays off chain.

Example:
  zeroned tx sponsorship create-bounty mathematics 1000000 10 5000 \
    --work-spec-hash <sha256> --acceptance-hash <sha256> \
    --input-root <sha256> --environment-root <sha256> \
    --worker-address zrn1... --from sponsor`,
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
			workSpecHash, _ := cmd.Flags().GetString("work-spec-hash")
			acceptanceHash, _ := cmd.Flags().GetString("acceptance-hash")
			inputRoot, _ := cmd.Flags().GetString("input-root")
			environmentRoot, _ := cmd.Flags().GetString("environment-root")
			workerAddress, _ := cmd.Flags().GetString("worker-address")
			minCorroborations, err := cmd.Flags().GetUint64("min-corroborations")
			if err != nil {
				return err
			}
			msg := &types.MsgCreateBountyOrder{
				Sponsor:          clientCtx.GetFromAddress().String(),
				Domain:           args[0],
				PricePerArtifact: args[1],
				TargetCount:      uint32(targetCount),
				DurationBlocks:   durationBlocks,
				WorkContract: &types.WorkContract{
					WorkSpecHash:      workSpecHash,
					AcceptanceHash:    acceptanceHash,
					InputRoot:         inputRoot,
					EnvironmentRoot:   environmentRoot,
					MinCorroborations: minCorroborations,
					WorkerAddress:     workerAddress,
				},
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("work-spec-hash", "", "Work-spec SHA-256 (bare lowercase 64-hex)")
	cmd.Flags().String("acceptance-hash", "", "Acceptance/evaluator SHA-256 (bare lowercase 64-hex)")
	cmd.Flags().String("input-root", "", "Input manifest root (bare lowercase 64-hex)")
	cmd.Flags().String("environment-root", "", "Execution environment root (bare lowercase 64-hex)")
	cmd.Flags().String("worker-address", "", "Preassigned worker payout address (zrn1...)")
	cmd.Flags().Uint64("min-corroborations", 0, "Minimum survived formal challenges required in addition to challenge-window maturity")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewFulfillBountyCmd builds a tx that pays the submitter of [fact-id]
// the bounty's per-artifact price, provided the fact is verified, in the
// bounty's domain, submitted within the window, and not already fulfilled.
// The signer must be the worker stored in fact.Submitter.
func NewFulfillBountyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fulfill-bounty [bounty-id] [fact-id]",
		Short: "Trigger payout from a bounty to the submitter of a qualifying fact",
		Long: `The stored fact submitter must sign fulfillment, so the worker chooses
which matching offer consumes its fact and receipt. The chain enforces all
eligibility checks (bounty active, exact computational contract match,
challenge window elapsed, required corroborations survived, sponsorship-global
fact/receipt/worker-bound-nullifier replay tombstones). Payout flows from bounty escrow to
fact.Submitter; no caller-selected payee is accepted.

Example:
  zeroned tx sponsorship fulfill-bounty bounty-1 fact-abc --from fact-submitter`,
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
		Long: `Only the original sponsor can cancel. Bound v2 bounties become
cancelable only after their deadline; legacy unbound bounties remain
cancelable so old escrow is recoverable. Refunds escrow_remaining to the
sponsor. FULFILLED and CANCELED bounties cannot be canceled.

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
