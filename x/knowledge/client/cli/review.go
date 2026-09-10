package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func addReviewFlags(cmd *cobra.Command) {
	cmd.Flags().Uint32("commitment-scheme", 0, "offline override: 0 legacy or 2 reviewer-bound (default: query the round)")
	cmd.Flags().String("commitment-chain-id", "", "original round chain; required with offline scheme 2")
	cmd.Flags().Uint64("confidence", 0, "review confidence on the 0–1000000 scale; repeat exactly on reveal")
	cmd.Flags().String("review-reason", "", "what was checked or why this vote was reached; required for scheme 2")
	cmd.Flags().String("review-method", "", "optional registered methodology ID")
	cmd.Flags().String("review-scope", "", "limits of this review")
	cmd.Flags().StringArray("review-evidence", nil, "evidence reference; repeat for each reference, preserving order")
}

func resolveReviewRound(cmd *cobra.Command, clientCtx client.Context, roundID string) (uint32, string, error) {
	chain, _ := cmd.Flags().GetString("commitment-chain-id")
	if cmd.Flags().Changed("commitment-scheme") {
		scheme, _ := cmd.Flags().GetUint32("commitment-scheme")
		switch scheme {
		case types.CommitmentSchemeLegacy:
			if chain != "" {
				return 0, "", fmt.Errorf("legacy commitments have no chain-context field")
			}
		case types.CommitmentSchemeReviewV2:
			if err := types.ValidateRecordText("commitment chain", chain, types.MaxCommitmentContextBytes, true); err != nil {
				return 0, "", err
			}
		default:
			return 0, "", fmt.Errorf("unknown commitment scheme %d", scheme)
		}
		return scheme, chain, nil
	}
	if chain != "" {
		return 0, "", fmt.Errorf("--commitment-chain-id requires explicit --commitment-scheme")
	}
	response := new(types.QueryVerificationRoundResponse)
	if err := clientCtx.Invoke(cmd.Context(), types.Query_VerificationRound_FullMethodName, &types.QueryVerificationRoundRequest{Id: roundID}, response); err != nil {
		return 0, "", fmt.Errorf("read commitment scheme from round: %w (offline generation requires explicit --commitment-scheme and, for scheme 2, --commitment-chain-id)", err)
	}
	if response.Round == nil || response.Round.Id != roundID {
		return 0, "", fmt.Errorf("round query returned mismatched identity")
	}
	round := response.Round
	if round.CommitmentScheme != types.CommitmentSchemeLegacy && round.CommitmentScheme != types.CommitmentSchemeReviewV2 {
		return 0, "", fmt.Errorf("unknown commitment scheme %d", round.CommitmentScheme)
	}
	if round.CommitmentScheme == types.CommitmentSchemeLegacy && round.CommitmentChainId != "" {
		return 0, "", fmt.Errorf("legacy round has unexpected chain context")
	}
	if round.CommitmentScheme == types.CommitmentSchemeReviewV2 {
		if err := types.ValidateRecordText("commitment chain", round.CommitmentChainId, types.MaxCommitmentContextBytes, true); err != nil {
			return 0, "", err
		}
	}
	return round.CommitmentScheme, round.CommitmentChainId, nil
}

func reviewFromFlags(cmd *cobra.Command, scheme uint32) (*types.ReviewAttestation, uint64, error) {
	confidence, _ := cmd.Flags().GetUint64("confidence")
	reason, _ := cmd.Flags().GetString("review-reason")
	method, _ := cmd.Flags().GetString("review-method")
	scope, _ := cmd.Flags().GetString("review-scope")
	evidence, _ := cmd.Flags().GetStringArray("review-evidence")
	if scheme == types.CommitmentSchemeLegacy {
		if reason != "" || method != "" || scope != "" || len(evidence) != 0 {
			return nil, 0, fmt.Errorf("legacy rounds cannot bind review attestation fields")
		}
		return nil, confidence, nil
	}
	if scheme != types.CommitmentSchemeReviewV2 {
		return nil, 0, fmt.Errorf("unknown commitment scheme")
	}
	att := &types.ReviewAttestation{MethodId: method, Reason: reason, EvidenceIds: evidence, Scope: scope}
	if err := types.ValidateReviewAttestation(att); err != nil {
		return nil, 0, err
	}
	if confidence > types.BPS {
		return nil, 0, fmt.Errorf("confidence exceeds1000000")
	}
	return att, confidence, nil
}

func shellArgument(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

func reviewRevealCommand(roundID, vote, salt string, scheme uint32, chain string, confidence uint64, att *types.ReviewAttestation) string {
	args := []string{"zeroned", "tx", "knowledge", "submit-reveal", shellArgument(roundID), shellArgument(vote), shellArgument(salt), "--from", "'<same-key>'", "--commitment-scheme", strconv.FormatUint(uint64(scheme), 10), "--confidence", strconv.FormatUint(confidence, 10)}
	if scheme == types.CommitmentSchemeReviewV2 {
		args = append(args, "--commitment-chain-id", shellArgument(chain), "--review-reason", shellArgument(att.Reason))
		if att.MethodId != "" {
			args = append(args, "--review-method", shellArgument(att.MethodId))
		}
		if att.Scope != "" {
			args = append(args, "--review-scope", shellArgument(att.Scope))
		}
		for _, ref := range att.EvidenceIds {
			args = append(args, "--review-evidence", shellArgument(ref))
		}
	}
	return strings.Join(args, " ")
}
