package cli

import (
	"context"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"testing"
)

func TestReviewCLIExplicitVersionAndExactAttestation(t *testing.T) {
	cmd := &cobra.Command{}
	addReviewFlags(cmd)
	cmd.SetContext(context.Background())
	require.NoError(t, cmd.Flags().Set("commitment-scheme", "2"))
	_, _, err := resolveReviewRound(cmd, client.Context{}, "round")
	require.Error(t, err)
	require.NoError(t, cmd.Flags().Set("commitment-chain-id", "original-chain"))
	scheme, chain, err := resolveReviewRound(cmd, client.Context{}, "round")
	require.NoError(t, err)
	require.Equal(t, uint32(2), scheme)
	require.Equal(t, "original-chain", chain)
	_, _, err = reviewFromFlags(cmd, scheme)
	require.Error(t, err)
	reason := "Checked 'UTC': $(not-a-command) α"
	require.NoError(t, cmd.Flags().Set("review-reason", reason))
	require.NoError(t, cmd.Flags().Set("confidence", "800000"))
	require.NoError(t, cmd.Flags().Set("review-evidence", "first,unsplit"))
	require.NoError(t, cmd.Flags().Set("review-evidence", "second"))
	att, confidence, err := reviewFromFlags(cmd, scheme)
	require.NoError(t, err)
	require.Equal(t, reason, att.Reason)
	require.Equal(t, []string{"first,unsplit", "second"}, att.EvidenceIds)
	command := reviewRevealCommand("round", "accept", "00", scheme, chain, confidence, att)
	require.Contains(t, command, "--commitment-chain-id 'original-chain'")
	require.Contains(t, command, "--confidence 800000")
	require.Contains(t, command, "--review-reason "+shellArgument(reason))
	require.Contains(t, command, "--review-evidence 'first,unsplit' --review-evidence 'second'")
	_, _, err = reviewFromFlags(cmd, types.CommitmentSchemeLegacy)
	require.Error(t, err)
	require.NoError(t, cmd.Flags().Set("commitment-scheme", "1"))
	_, _, err = resolveReviewRound(cmd, client.Context{}, "round")
	require.Error(t, err)
}
func TestReviewCLINoImplicitLegacyOnMissingRound(t *testing.T) {
	cmd := &cobra.Command{}
	addReviewFlags(cmd)
	cmd.SetContext(context.Background())
	_, _, err := resolveReviewRound(cmd, client.Context{}, "missing")
	require.Error(t, err)
	require.NoError(t, cmd.Flags().Set("commitment-scheme", "0"))
	scheme, chain, err := resolveReviewRound(cmd, client.Context{}, "missing")
	require.NoError(t, err)
	require.Zero(t, scheme)
	require.Empty(t, chain)
}
