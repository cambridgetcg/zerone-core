package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func validComputationalCommitment(payee string) *types.ComputationalCommitment {
	c := &types.ComputationalCommitment{
		WorkSpecHash:    strings.Repeat("1", 64),
		AcceptanceHash:  strings.Repeat("2", 64),
		InputRoot:       strings.Repeat("3", 64),
		EnvironmentRoot: strings.Repeat("4", 64),
		ArtifactRoot:    strings.Repeat("5", 64),
		EvidenceRoot:    strings.Repeat("6", 64),
	}
	c.WorkReceiptHash = types.ComputeWorkReceiptHash(c, payee)
	return c
}

func TestMsgSubmitClaim_ValidateBasic_ComputationalTypeWall(t *testing.T) {
	payee := "zrn1computationalpayee"
	valid := validComputationalCommitment(payee)
	require.NoError(t, (&types.MsgSubmitClaim{
		Submitter: payee, ClaimType: types.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
		MethodId: types.MethodologyComputational, ComputationalCommitment: valid,
	}).ValidateBasic())

	for _, field := range []string{
		"work_spec_hash", "acceptance_hash", "input_root", "environment_root",
		"artifact_root", "evidence_root", "work_receipt_hash",
	} {
		t.Run("missing_"+field, func(t *testing.T) {
			c := *valid
			switch field {
			case "work_spec_hash":
				c.WorkSpecHash = ""
			case "acceptance_hash":
				c.AcceptanceHash = ""
			case "input_root":
				c.InputRoot = ""
			case "environment_root":
				c.EnvironmentRoot = ""
			case "artifact_root":
				c.ArtifactRoot = ""
			case "evidence_root":
				c.EvidenceRoot = ""
			case "work_receipt_hash":
				c.WorkReceiptHash = ""
			}
			err := (&types.MsgSubmitClaim{
				Submitter: payee, ClaimType: types.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
				MethodId: types.MethodologyComputational, ComputationalCommitment: &c,
			}).ValidateBasic()
			require.Error(t, err)
		})
	}

	require.Error(t, (&types.MsgSubmitClaim{
		Submitter: payee, ClaimType: types.ClaimType_CLAIM_TYPE_ASSERTION,
		ComputationalCommitment: valid,
	}).ValidateBasic(), "non-computational claims must not carry computational provenance")
	require.Error(t, (&types.MsgSubmitClaim{
		Submitter: payee, ClaimType: types.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
		ComputationalCommitment: valid,
	}).ValidateBasic(), "computational claims must not resolve silently to M-LEGACY")
}

func TestMsgSubmitClaim_ValidateBasic_ReasoningTraceBoundaries(t *testing.T) {
	require.NoError(t, (&types.MsgSubmitClaim{ReasoningTrace: strings.Repeat("x", types.MaxReasoningTraceBytes)}).ValidateBasic())
	require.Error(t, (&types.MsgSubmitClaim{ReasoningTrace: strings.Repeat("x", types.MaxReasoningTraceBytes+1)}).ValidateBasic())
}

func TestWorkReceiptHash_BindsEveryFieldAndPayee(t *testing.T) {
	base := validComputationalCommitment("zrn1alice")
	want := base.WorkReceiptHash
	mutations := []*types.ComputationalCommitment{}
	for i := 0; i < 6; i++ {
		c := *base
		switch i {
		case 0:
			c.WorkSpecHash = strings.Repeat("a", 64)
		case 1:
			c.AcceptanceHash = strings.Repeat("b", 64)
		case 2:
			c.InputRoot = strings.Repeat("c", 64)
		case 3:
			c.EnvironmentRoot = strings.Repeat("d", 64)
		case 4:
			c.ArtifactRoot = strings.Repeat("e", 64)
		case 5:
			c.EvidenceRoot = strings.Repeat("f", 64)
		}
		mutations = append(mutations, &c)
	}
	for _, mutation := range mutations {
		require.NotEqual(t, want, types.ComputeWorkReceiptHash(mutation, "zrn1alice"))
	}
	require.NotEqual(t, want, types.ComputeWorkReceiptHash(base, "zrn1bob"))
}

func TestWorkReceiptHash_CrossLanguageVector(t *testing.T) {
	c := validComputationalCommitment("zrn1v3jkzervd9hx2ttfdejx27pdwdcx7m3dwexsmf")
	require.Equal(t,
		"341ed4d5e3e2fc399d75def7ced694b6889d2108aa4691a1594c52f9bf85724c",
		types.ComputeWorkReceiptHash(c, "zrn1v3jkzervd9hx2ttfdejx27pdwdcx7m3dwexsmf"),
	)
}

func TestGenesisValidate_ComputationalCompatibilityAndTypeWall(t *testing.T) {
	payee := "zrn1genesiscomputationalworker"
	gs := types.DefaultGenesis()
	gs.Facts = append(gs.Facts, &types.Fact{
		Id: "legacy-computational-fact", ClaimType: types.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
		Submitter: payee,
	})
	gs.PendingClaims = append(gs.PendingClaims, &types.Claim{
		Id: "legacy-computational-claim", ClaimType: types.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
		Submitter: payee,
	})
	require.NoError(t, gs.Validate(), "pre-v7 nil computational records must remain export/import compatible")

	bound := validComputationalCommitment(payee)
	gs.Facts = append(gs.Facts, &types.Fact{
		Id: "bound-computational-fact", ClaimType: types.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
		Submitter: payee, MethodId: types.MethodologyComputational, ComputationalCommitment: bound,
	})
	require.NoError(t, gs.Validate())

	badType := *gs
	badType.Facts = append([]*types.Fact(nil), gs.Facts...)
	badType.Facts = append(badType.Facts, &types.Fact{
		Id: "assertion-with-provenance", ClaimType: types.ClaimType_CLAIM_TYPE_ASSERTION,
		Submitter: payee, ComputationalCommitment: validComputationalCommitment(payee),
	})
	require.Error(t, badType.Validate(), "noncomputational genesis records cannot carry a commitment")

	badReceipt := *gs
	badReceipt.Facts = append([]*types.Fact(nil), gs.Facts...)
	c := *validComputationalCommitment(payee)
	c.WorkReceiptHash = strings.Repeat("f", 64)
	badReceipt.Facts = append(badReceipt.Facts, &types.Fact{
		Id: "bound-bad-receipt", ClaimType: types.ClaimType_CLAIM_TYPE_COMPUTATIONAL,
		Submitter: payee, MethodId: types.MethodologyComputational, ComputationalCommitment: &c,
	})
	require.Error(t, badReceipt.Validate(), "bound genesis receipts must commit the stored submitter")
}
