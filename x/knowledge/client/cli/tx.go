package cli

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// parseClaimType maps a CLI string to a ClaimType enum value.
func parseClaimType(s string) (types.ClaimType, error) {
	switch strings.ToLower(s) {
	case "", "assertion":
		return types.ClaimType_CLAIM_TYPE_ASSERTION, nil
	case "relation":
		return types.ClaimType_CLAIM_TYPE_RELATION, nil
	case "definition":
		return types.ClaimType_CLAIM_TYPE_DEFINITION, nil
	case "constraint":
		return types.ClaimType_CLAIM_TYPE_CONSTRAINT, nil
	case "negation":
		return types.ClaimType_CLAIM_TYPE_NEGATION, nil
	case "observation":
		return types.ClaimType_CLAIM_TYPE_OBSERVATION, nil
	default:
		return 0, fmt.Errorf("unknown claim type %q: must be assertion, relation, definition, constraint, negation, or observation", s)
	}
}

// parseRelationType maps a CLI string to a RelationType enum value.
func parseRelationType(s string) (types.RelationType, error) {
	switch strings.ToLower(s) {
	case "supports":
		return types.RelationType_RELATION_TYPE_SUPPORTS, nil
	case "contradicts":
		return types.RelationType_RELATION_TYPE_CONTRADICTS, nil
	case "requires":
		return types.RelationType_RELATION_TYPE_REQUIRES, nil
	case "refines":
		return types.RelationType_RELATION_TYPE_REFINES, nil
	case "generalizes":
		return types.RelationType_RELATION_TYPE_GENERALIZES, nil
	case "supersedes":
		return types.RelationType_RELATION_TYPE_SUPERSEDES, nil
	default:
		return 0, fmt.Errorf("unknown relation type %q: must be supports, contradicts, requires, refines, generalizes, or supersedes", s)
	}
}

// GetTxCmd returns the root transaction command for the knowledge module.
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Knowledge module transaction commands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		NewSubmitClaimCmd(),
		NewSubmitCommitmentCmd(),
		NewSubmitRevealCmd(),
		NewChallengeFactCmd(),
		NewAddFactCmd(),
		NewSubmitContradictionCmd(),
		NewPatronizeFactCmd(),
		NewProposeDomainCmd(),
		NewEndorseDomainProposalCmd(),
		NewChallengeDomainProposalCmd(),
		NewRegisterStratumCmd(),
		NewChallengeProvisionalFactCmd(),
		NewProposeResearchFundCmd(),
		NewVoteResearchProposalCmd(),
		NewExecuteResearchProposalCmd(),
		NewRateFactCmd(),
		NewReportDemandCmd(),
	)

	return txCmd
}

// NewSubmitClaimCmd creates a CLI command for MsgSubmitClaim.
func NewSubmitClaimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-claim [fact-content] [domain] [category] [review-fee]",
		Short: "Submit a knowledge claim for verification (review fee is non-refundable)",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			refs, _ := cmd.Flags().GetString("references")
			var references []string
			if refs != "" {
				references = strings.Split(refs, ",")
			}

			partnershipId, _ := cmd.Flags().GetString("partnership-id")

			claimTypeStr, _ := cmd.Flags().GetString("claim-type")
			claimType, err := parseClaimType(claimTypeStr)
			if err != nil {
				return err
			}

			// Parse typed relations
			relationsStr, _ := cmd.Flags().GetString("relations")
			var relations []*types.ClaimRelation
			if relationsStr != "" {
				for _, pair := range strings.Split(relationsStr, ",") {
					parts := strings.SplitN(pair, ":", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid relation format %q: expected type:fact_id", pair)
					}
					relType, err := parseRelationType(parts[0])
					if err != nil {
						return err
					}
					relations = append(relations, &types.ClaimRelation{
						TargetFactId: parts[1],
						Relation:     relType,
					})
				}
			}

			// Parse structured claim fields
			var structure *types.ClaimStructure
			subject, _ := cmd.Flags().GetString("subject")
			predicate, _ := cmd.Flags().GetString("predicate")
			if subject != "" || predicate != "" {
				object, _ := cmd.Flags().GetString("object")
				scope, _ := cmd.Flags().GetString("scope")
				temporalScope, _ := cmd.Flags().GetString("temporal-scope")
				negatable, _ := cmd.Flags().GetBool("negatable")
				tagsStr, _ := cmd.Flags().GetString("tags")
				var tags []string
				if tagsStr != "" {
					tags = strings.Split(tagsStr, ",")
				}
				structure = &types.ClaimStructure{
					Subject:       subject,
					Predicate:     predicate,
					Object:        object,
					Scope:         scope,
					TemporalScope: temporalScope,
					Negatable:     negatable,
					Tags:          tags,
				}
			}

			canonicalForm, _ := cmd.Flags().GetString("canonical")
			sponsored, _ := cmd.Flags().GetBool("sponsored")

			msg := &types.MsgSubmitClaim{
				Submitter:     clientCtx.GetFromAddress().String(),
				FactContent:   args[0],
				Domain:        args[1],
				Category:      args[2],
				Stake:         args[3],
				References:    references,
				PartnershipId: partnershipId,
				ClaimType:     claimType,
				Relations:     relations,
				Structure:     structure,
				CanonicalForm: canonicalForm,
				Sponsored:     sponsored,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("references", "", "Comma-separated fact IDs to reference")
	cmd.Flags().String("partnership-id", "", "Partnership ID for collaborative claims")
	cmd.Flags().String("claim-type", "assertion", "Claim type: assertion (default), relation, definition, constraint, negation, observation")
	cmd.Flags().String("relations", "", "Typed relations: supports:FACT_ID,contradicts:FACT_ID,requires:FACT_ID")
	cmd.Flags().String("subject", "", "Claim subject (structured)")
	cmd.Flags().String("predicate", "", "Claim predicate (structured)")
	cmd.Flags().String("object", "", "Claim object (structured, optional)")
	cmd.Flags().String("scope", "", "Claim scope/conditions (structured, optional)")
	cmd.Flags().String("temporal-scope", "", "Time bounds (structured, optional)")
	cmd.Flags().Bool("negatable", true, "Mark claim as negatable (default true)")
	cmd.Flags().String("tags", "", "Comma-separated tags")
	cmd.Flags().String("canonical", "", "Explicit canonical form (auto-derived from structure if omitted)")
	cmd.Flags().Bool("sponsored", false, "Request bootstrap fund sponsorship (fund pays review fee)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSubmitCommitmentCmd creates a CLI command for MsgSubmitCommitment.
//
// Two modes:
//   - expert:  submit-commitment <round-id> <commit-hash-hex>   (you computed the hash)
//   - hospitable: submit-commitment <round-id> --vote accept    (we compute it correctly)
//
// The hospitable mode exists because the commit preimage is domain-tagged
// ("ZRN.commit.v1:<round>:<vote>:<confidence>:<salt-hex>") and the CLI reveal
// always sends confidence=0 — a hand-rolled hash with any other shape fails the
// reveal with ErrRevealMismatch. The CLI computes it via the same
// types.ComputeCommitmentHash the chain verifies with, so it cannot drift.
func NewSubmitCommitmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-commitment [round-id] [commit-hash-hex]",
		Short: "Submit a verification commitment (commit-reveal phase 1)",
		Long: `Commit to a vote on a verification round.

Easiest path (salt is generated and the hash computed for you):
  zeroned tx knowledge submit-commitment <round-id> --vote accept --from me
The command prints the salt and the exact reveal command to run after the
commit phase ends. Optionally persist it with --salt-out <file>.

Expert path (you already computed the domain-tagged hash):
  zeroned tx knowledge submit-commitment <round-id> <commit-hash-hex> --from me`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			roundID := args[0]
			vote, _ := cmd.Flags().GetString("vote")
			saltHex, _ := cmd.Flags().GetString("salt")
			saltOut, _ := cmd.Flags().GetString("salt-out")

			var commitHash []byte
			switch {
			case len(args) == 2 && vote != "":
				return fmt.Errorf("give either a commit-hash argument or --vote, not both")
			case len(args) == 2:
				if saltHex != "" || saltOut != "" {
					return fmt.Errorf("--salt/--salt-out only apply with --vote (with a precomputed hash you already own the salt)")
				}
				commitHash, err = hex.DecodeString(args[1])
				if err != nil {
					return fmt.Errorf("invalid commit-hash hex: %w", err)
				}
			case vote != "":
				switch vote {
				case "accept", "reject", "malformed":
				default:
					return fmt.Errorf("--vote must be accept, reject, or malformed (got %q)", vote)
				}
				var salt []byte
				if saltHex != "" {
					salt, err = hex.DecodeString(saltHex)
					if err != nil {
						return fmt.Errorf("invalid --salt hex: %w", err)
					}
				} else {
					salt = make([]byte, 16)
					if _, err := rand.Read(salt); err != nil {
						return fmt.Errorf("salt generation failed: %w", err)
					}
					saltHex = hex.EncodeToString(salt)
				}
				// Confidence is 0 because the CLI reveal path cannot set it —
				// committing with any other value makes the reveal unmatchable.
				commitHash = types.ComputeCommitmentHash(roundID, vote, 0, salt)
				if saltOut != "" {
					if err := os.WriteFile(saltOut, []byte(saltHex+"\n"), 0o600); err != nil {
						return fmt.Errorf("could not persist salt before broadcasting (refusing to commit a vote we could not reveal): %w", err)
					}
				}
			default:
				return fmt.Errorf("provide a commit-hash argument or --vote accept|reject|malformed")
			}

			msg := &types.MsgSubmitCommitment{
				Verifier:   clientCtx.GetFromAddress().String(),
				RoundId:    roundID,
				CommitHash: commitHash,
			}

			if err := tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg); err != nil {
				return err
			}

			if vote != "" {
				// Guidance goes to stderr so -o json / --generate-only stdout
				// stays machine-readable. Wording is broadcast-aware: we only
				// see the CheckTx response here, so we teach the code check
				// instead of celebrating unconditionally.
				out := cmd.ErrOrStderr()
				genOnly, _ := cmd.Flags().GetBool(flags.FlagGenerateOnly)
				dryRun, _ := cmd.Flags().GetBool(flags.FlagDryRun)
				fmt.Fprintln(out, "")
				if genOnly || dryRun {
					fmt.Fprintln(out, "Tx NOT broadcast (generate-only/dry-run). KEEP THIS SALT anyway — the commitment hash above is bound to it:")
				} else {
					fmt.Fprintln(out, "Broadcast. If the response above shows code: 0, your commitment is sealed (verify: zeroned q tx <txhash>).")
					fmt.Fprintln(out, "KEEP THIS SALT — without it your vote cannot be revealed, and an unrevealed commit is slashable:")
				}
				fmt.Fprintln(out, "  salt: "+saltHex)
				if saltOut != "" {
					fmt.Fprintln(out, "  (also saved to "+saltOut+")")
				}
				fmt.Fprintln(out, "After the commit phase ends, reveal with exactly:")
				fmt.Fprintln(out, "  zeroned tx knowledge submit-reveal "+roundID+" "+vote+" "+saltHex+" --from <same-key>")
				fmt.Fprintln(out, "Find the claim this round belongs to: zeroned q knowledge verification-round "+roundID)
				fmt.Fprintln(out, "Then follow it live: zeroned q knowledge claim-watch <claim-id>")
			}
			return nil
		},
	}

	cmd.Flags().String("vote", "", "compute the commitment for this vote (accept|reject|malformed) instead of passing a hash")
	cmd.Flags().String("salt", "", "hex salt to use with --vote (default: 16 random bytes, printed after broadcast)")
	cmd.Flags().String("salt-out", "", "also write the salt to this file (0600) before broadcasting")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSubmitRevealCmd creates a CLI command for MsgSubmitReveal.
func NewSubmitRevealCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-reveal [round-id] [vote] [salt-hex]",
		Short: "Submit a verification reveal (commit-reveal phase 2)",
		Long:  `Vote must be "accept", "reject", or "malformed". Salt is the hex-encoded salt used in commitment.`,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			salt, err := hex.DecodeString(args[2])
			if err != nil {
				return err
			}

			msg := &types.MsgSubmitReveal{
				Verifier: clientCtx.GetFromAddress().String(),
				RoundId:  args[0],
				Vote:     args[1],
				Salt:     salt,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewChallengeFactCmd creates a CLI command for MsgChallengeFact.
func NewChallengeFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-fact [fact-id] [stake] [reason]",
		Short: "Challenge an existing fact",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			evidenceStr, _ := cmd.Flags().GetString("evidence-ids")
			var evidenceIds []string
			if evidenceStr != "" {
				evidenceIds = strings.Split(evidenceStr, ",")
			}

			msg := &types.MsgChallengeFact{
				Challenger:  clientCtx.GetFromAddress().String(),
				FactId:      args[0],
				Stake:       args[1],
				Reason:      args[2],
				EvidenceIds: evidenceIds,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("evidence-ids", "", "Comma-separated evidence IDs")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewAddFactCmd creates a CLI command for MsgAddFact (authority only).
func NewAddFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-fact [content] [domain] [category] [confidence]",
		Short: "Add a fact directly (authority only)",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			confidence, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return err
			}

			refsStr, _ := cmd.Flags().GetString("references")
			var references []string
			if refsStr != "" {
				references = strings.Split(refsStr, ",")
			}

			msg := &types.MsgAddFact{
				Authority:  clientCtx.GetFromAddress().String(),
				Content:    args[0],
				Domain:     args[1],
				Category:   args[2],
				Confidence: confidence,
				References: references,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("references", "", "Comma-separated fact IDs to reference")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewSubmitContradictionCmd creates a CLI command for MsgSubmitContradiction.
func NewSubmitContradictionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-contradiction [fact-id] [counter-claim] [stake] [reason]",
		Short: "Submit a contradiction to an existing fact",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			domain, _ := cmd.Flags().GetString("domain")
			category, _ := cmd.Flags().GetString("category")
			evidenceStr, _ := cmd.Flags().GetString("evidence-ids")
			var evidenceIds []string
			if evidenceStr != "" {
				evidenceIds = strings.Split(evidenceStr, ",")
			}

			msg := &types.MsgSubmitContradiction{
				Submitter:   clientCtx.GetFromAddress().String(),
				FactId:      args[0],
				CounterClaim: args[1],
				Stake:       args[2],
				Reason:      args[3],
				Domain:      domain,
				Category:    category,
				EvidenceIds: evidenceIds,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("domain", "", "Knowledge domain")
	cmd.Flags().String("category", "", "Epistemic category")
	cmd.Flags().String("evidence-ids", "", "Comma-separated evidence IDs")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewPatronizeFactCmd creates a CLI command for MsgPatronizeFact.
func NewPatronizeFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patronize-fact [fact-id] [amount] [duration-blocks]",
		Short: "Patronize a fact with staking support",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			durationBlocks, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgPatronizeFact{
				Patron:         clientCtx.GetFromAddress().String(),
				FactId:         args[0],
				Amount:         args[1],
				DurationBlocks: durationBlocks,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProposeDomainCmd creates a CLI command for MsgProposeDomain.
func NewProposeDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-domain [name] [description] [stratum] [stake]",
		Short: "Propose a new knowledge domain",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgProposeDomain{
				Proposer:    clientCtx.GetFromAddress().String(),
				Name:        args[0],
				Description: args[1],
				Stratum:     args[2],
				Stake:       args[3],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewEndorseDomainProposalCmd creates a CLI command for MsgEndorseDomainProposal.
func NewEndorseDomainProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "endorse-domain [proposal-id]",
		Short: "Endorse a domain proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgEndorseDomainProposal{
				Endorser:   clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewChallengeDomainProposalCmd creates a CLI command for MsgChallengeDomainProposal.
func NewChallengeDomainProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-domain [proposal-id] [reason]",
		Short: "Challenge a domain proposal",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgChallengeDomainProposal{
				Challenger: clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Reason:     args[1],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRegisterStratumCmd creates a CLI command for MsgRegisterStratum.
func NewRegisterStratumCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-stratum [name] [description] [confidence-ceiling]",
		Short: "Register a new epistemic stratum (authority only)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			ceiling, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			parentsStr, _ := cmd.Flags().GetString("parent-strata")
			var parents []string
			if parentsStr != "" {
				parents = strings.Split(parentsStr, ",")
			}

			msg := &types.MsgRegisterStratum{
				Authority:         clientCtx.GetFromAddress().String(),
				Name:              args[0],
				Description:       args[1],
				ConfidenceCeiling: ceiling,
				ParentStrata:      parents,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("parent-strata", "", "Comma-separated parent stratum names")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewChallengeProvisionalFactCmd creates a CLI command for MsgChallengeProvisionalFact.
func NewChallengeProvisionalFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-provisional [claim-id] [fact-id] [stake] [reason]",
		Short: "Challenge a provisional fact during its grace period",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			counterClaim, _ := cmd.Flags().GetString("counter-claim")
			evidenceStr, _ := cmd.Flags().GetString("evidence-ids")
			var evidenceIds []string
			if evidenceStr != "" {
				evidenceIds = strings.Split(evidenceStr, ",")
			}

			msg := &types.MsgChallengeProvisionalFact{
				Challenger:   clientCtx.GetFromAddress().String(),
				ClaimId:      args[0],
				FactId:       args[1],
				Stake:        args[2],
				Reason:       args[3],
				EvidenceIds:  evidenceIds,
				CounterClaim: counterClaim,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String("counter-claim", "", "Counter-claim text")
	cmd.Flags().String("evidence-ids", "", "Comma-separated evidence IDs")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewProposeResearchFundCmd creates a CLI command for MsgProposeResearchFund.
func NewProposeResearchFundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "propose-research-fund [title] [description] [amount] [recipient] [voting-period-blocks]",
		Short: "Propose a research fund allocation",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			votingPeriod, err := strconv.ParseUint(args[4], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgProposeResearchFund{
				Proposer:           clientCtx.GetFromAddress().String(),
				Title:              args[0],
				Description:        args[1],
				Amount:             args[2],
				Recipient:          args[3],
				VotingPeriodBlocks: votingPeriod,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewVoteResearchProposalCmd creates a CLI command for MsgVoteResearchProposal.
func NewVoteResearchProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vote-research-proposal [proposal-id] [vote:true/false]",
		Short: "Vote on a research fund proposal",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vote, err := strconv.ParseBool(args[1])
			if err != nil {
				return err
			}

			msg := &types.MsgVoteResearchProposal{
				Voter:      clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
				Vote:       vote,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewExecuteResearchProposalCmd creates a CLI command for MsgExecuteResearchProposal.
func NewExecuteResearchProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute-research-proposal [proposal-id]",
		Short: "Execute an approved research fund proposal (authority only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgExecuteResearchProposal{
				Authority:  clientCtx.GetFromAddress().String(),
				ProposalId: args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewRateFactCmd creates a CLI command for MsgRateFact.
// Requires a prior query receipt for the fact (emitted by QueryFact).
func NewRateFactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-fact [fact-id] [useful] [memo]",
		Short: "Rate a queried fact as useful or not (max 256-char memo, optional)",
		Long: `Rate a fact you previously queried.
  useful: "true" (satisfied) or "false" (not satisfied)
  memo:   optional reason, max 256 chars. Pass "" for no memo.
Requires a valid query receipt; each receipt rates at most once.`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			useful, err := strconv.ParseBool(args[1])
			if err != nil {
				return fmt.Errorf("invalid useful flag %q: must be true or false", args[1])
			}

			memo := ""
			if len(args) == 3 {
				memo = args[2]
			}

			msg := &types.MsgRateFact{
				Rater:  clientCtx.GetFromAddress().String(),
				FactId: args[0],
				Useful: useful,
				Memo:   memo,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewReportDemandCmd creates a CLI command for MsgReportDemand.
// Reads a JSON file containing an array of DemandReport objects:
//
//	[
//	  {"domain":"physics","subject":"gravity","queries":10,"fulfilled":7,"unfulfilled":3},
//	  ...
//	]
func NewReportDemandCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report-demand [reports-json-file]",
		Short: "Report batched agent query-demand signals (whitelisted reporters only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			raw, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read %s: %w", args[0], err)
			}

			var reports []*types.DemandReport
			if err := json.Unmarshal(raw, &reports); err != nil {
				return fmt.Errorf("parse reports JSON: %w", err)
			}
			if len(reports) == 0 {
				return fmt.Errorf("no reports in %s", args[0])
			}

			msg := &types.MsgReportDemand{
				Reporter: clientCtx.GetFromAddress().String(),
				Reports:  reports,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
