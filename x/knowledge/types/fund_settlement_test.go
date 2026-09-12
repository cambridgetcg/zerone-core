package types_test

import (
	"bytes"
	"math"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func fundingClaim(kind types.ClaimFundingKind, amount uint64) *types.Claim {
	review := types.ClaimReviewBudget(amount)
	terms := &types.ClaimFundingTerms{PolicyVersion: 1, Kind: kind, PaidAmount: strconv.FormatUint(amount, 10), ReviewBudget: strconv.FormatUint(review, 10), RefundableAmount: "0", RetainedFee: "0"}
	if kind == types.ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE {
		terms.RetainedFee = strconv.FormatUint(amount-review, 10)
	} else {
		terms.RefundableAmount = strconv.FormatUint(amount-review, 10)
	}
	return &types.Claim{Id: "funded-claim", Submitter: sdk.AccAddress(bytes.Repeat([]byte{71}, 20)).String(), Stake: terms.PaidAmount, ReviewPolicyVersion: types.ReviewPolicyNeutral, FundingTerms: terms}
}

func fundedRound(t *testing.T, claim *types.Claim, votes []string) *types.VerificationRound {
	t.Helper()
	r := &types.VerificationRound{Id: "funded-round", ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
		StartedAtBlock: 1, CommitDeadline: 3, RevealDeadline: 5, AggregationDeadline: 7,
		CommitmentScheme: types.CommitmentSchemeReviewV2, CommitmentChainId: "funding-test", ReviewPolicyVersion: types.ReviewPolicyNeutral,
		Verdict: types.Verdict_VERDICT_INCONCLUSIVE, VerdictBlock: 7}
	claim.VerificationRoundId = r.Id
	review, _ := types.SettlementAmount(claim.FundingTerms.ReviewBudget)
	refund, _ := types.SettlementAmount(claim.FundingTerms.RefundableAmount)
	for i, vote := range votes {
		address := sdk.AccAddress(bytes.Repeat([]byte{byte(i + 72)}, 20)).String()
		salt := bytes.Repeat([]byte{byte(i + 1)}, 16)
		att := &types.ReviewAttestation{Reason: "Synthetic scoped review; no independence claimed."}
		hash, err := types.ComputeReviewCommitmentV2(r.CommitmentChainId, r.Id, address, vote, 800000, salt, att)
		require.NoError(t, err)
		r.SelectedVerifiers = append(r.SelectedVerifiers, address)
		r.Commits = append(r.Commits, &types.CommitEntry{Verifier: address, CommitHash: hash, CommittedAtBlock: 2})
		r.Reveals = append(r.Reveals, &types.RevealEntry{Verifier: address, Vote: vote, Salt: salt, Confidence: 800000, Attestation: att, RevealedAtBlock: 4})
	}
	if len(votes) == 0 {
		refund += review
	} else if review > 0 {
		r.VerifierRewardSettlement = &types.VerifierRewardSettlement{CreatedAtBlock: 7, WithheldTotal: "0"}
		for i, reveal := range r.Reveals {
			amount := review / uint64(len(votes))
			if i == 0 {
				amount += review % uint64(len(votes))
			}
			r.VerifierRewardSettlement.Payments = append(r.VerifierRewardSettlement.Payments, &types.VerifierRewardPayment{Verifier: reveal.Verifier, Amount: strconv.FormatUint(amount, 10), Withheld: "0"})
		}
	}
	if refund > 0 {
		r.ClaimRefundSettlement = &types.ClaimRefundSettlement{Recipient: claim.Submitter, Amount: strconv.FormatUint(refund, 10), CreatedAtBlock: 7}
	}
	return r
}

func TestClaimFundingTermsAndImmutableAdmission(t *testing.T) {
	for _, kind := range []types.ClaimFundingKind{types.ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE, types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT} {
		for _, amount := range []uint64{1, 2, 100, 101, math.MaxUint64} {
			claim := fundingClaim(kind, amount)
			require.NoError(t, types.ValidateClaimFundingTerms(claim, true))
			require.Error(t, types.ValidateClaimFundingTerms(claim, false))
		}
	}
	good := fundingClaim(types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT, 101)
	for name, mutate := range map[string]func(*types.Claim){
		"missing payer":   func(c *types.Claim) { c.Submitter = "" },
		"different stake": func(c *types.Claim) { c.Stake = "100" },
		"noncanonical":    func(c *types.Claim) { c.Stake, c.FundingTerms.PaidAmount = "0101", "0101" },
		"overflow": func(c *types.Claim) {
			c.Stake, c.FundingTerms.PaidAmount = "18446744073709551616", "18446744073709551616"
		},
		"unknown version":    func(c *types.Claim) { c.FundingTerms.PolicyVersion = 2 },
		"legacy review":      func(c *types.Claim) { c.ReviewPolicyVersion = 0 },
		"kind zero":          func(c *types.Claim) { c.FundingTerms.Kind = 0 },
		"unknown kind":       func(c *types.Claim) { c.FundingTerms.Kind = 7 },
		"inflated budget":    func(c *types.Claim) { c.FundingTerms.ReviewBudget = "56" },
		"inflated bond":      func(c *types.Claim) { c.FundingTerms.RefundableAmount = "47" },
		"retained challenge": func(c *types.Claim) { c.FundingTerms.RetainedFee = "1" },
		"unknown field":      func(c *types.Claim) { c.FundingTerms.ProtoReflect().SetUnknown([]byte{0x38, 1}) },
	} {
		t.Run(name, func(t *testing.T) {
			c := proto.Clone(good).(*types.Claim)
			mutate(c)
			require.Error(t, types.ValidateClaimFundingTerms(c, true))
		})
	}
	require.NoError(t, types.ValidateClaimFundingTerms(&types.Claim{}, false), "legacy absence is not inferred funding")
	require.Error(t, types.ValidateClaimFundingUpdate(&types.Claim{}, good), "no retroactive terms")
	for name, mutate := range map[string]func(*types.Claim){
		"remove terms":          func(c *types.Claim) { c.FundingTerms = nil },
		"change payer":          func(c *types.Claim) { c.Submitter = sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String() },
		"change identity":       func(c *types.Claim) { c.Id = "other" },
		"change selected round": func(c *types.Claim) { c.VerificationRoundId = "other" },
	} {
		t.Run(name, func(t *testing.T) {
			before := proto.Clone(good).(*types.Claim)
			before.VerificationRoundId = "fixed"
			after := proto.Clone(before).(*types.Claim)
			mutate(after)
			require.Error(t, types.ValidateClaimFundingUpdate(before, after))
		})
	}
	after := proto.Clone(good).(*types.Claim)
	after.VerificationRoundId = "first-round"
	require.NoError(t, types.ValidateClaimFundingUpdate(good, after))
}

func TestClaimFundingRoundAccountsEveryTerminalOutcome(t *testing.T) {
	for _, kind := range []types.ClaimFundingKind{types.ClaimFundingKind_CLAIM_FUNDING_KIND_REVIEW_FEE, types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT} {
		for _, amount := range []uint64{1, 2, 101, math.MaxUint64} {
			for _, votes := range [][]string{nil, {"accept"}, {"accept", "reject", "malformed"}} {
				c := fundingClaim(kind, amount)
				r := fundedRound(t, c, votes)
				for _, verdict := range []types.Verdict{types.Verdict_VERDICT_ACCEPT, types.Verdict_VERDICT_REJECT, types.Verdict_VERDICT_MALFORMED, types.Verdict_VERDICT_INCONCLUSIVE} {
					r.Verdict = verdict
					require.NoError(t, types.ValidateClaimFundingRound(c, r), "the financial disposition is independent of the scientific verdict")
				}
			}
		}
	}
	c := fundingClaim(types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT, 101)
	r := fundedRound(t, c, []string{"accept", "reject", "malformed"})
	for name, mutate := range map[string]func(*types.VerificationRound){
		"erase refund":        func(r *types.VerificationRound) { r.ClaimRefundSettlement = nil },
		"refund mismatch":     func(r *types.VerificationRound) { r.ClaimRefundSettlement.Amount = "47" },
		"redirect refund":     func(r *types.VerificationRound) { r.ClaimRefundSettlement.Recipient = r.Reveals[0].Verifier },
		"erase reviewer plan": func(r *types.VerificationRound) { r.VerifierRewardSettlement = nil },
		"inflate reward":      func(r *types.VerificationRound) { r.VerifierRewardSettlement.Payments[0].Amount = "20" },
		"penalize dissent": func(r *types.VerificationRound) {
			r.VerifierRewardSettlement.Payments[1].Amount = "17"
			r.VerifierRewardSettlement.Payments[1].Withheld = "1"
			r.VerifierRewardSettlement.WithheldTotal = "1"
		},
		"invent recipient":     func(r *types.VerificationRound) { r.VerifierRewardSettlement.Payments[0].Verifier = c.Submitter },
		"split paid markers":   func(r *types.VerificationRound) { r.ClaimRefundSettlement.PaidAtBlock = 8 },
		"unknown refund field": func(r *types.VerificationRound) { r.ClaimRefundSettlement.ProtoReflect().SetUnknown([]byte{0x28, 1}) },
		"early reveal":         func(r *types.VerificationRound) { r.Reveals[0].RevealedAtBlock = 2 },
		"wrong claim":          func(r *types.VerificationRound) { r.ClaimId = "other" },
		"second round":         func(r *types.VerificationRound) { r.Id = "other" },
		"expired":              func(r *types.VerificationRound) { r.Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED },
	} {
		t.Run(name, func(t *testing.T) {
			bad := proto.Clone(r).(*types.VerificationRound)
			mutate(bad)
			require.Error(t, types.ValidateClaimFundingRound(c, bad))
		})
	}
	require.Error(t, types.ValidateClaimFundingRound(nil, r))
	require.Error(t, types.ValidateClaimFundingRound(&types.Claim{}, r))
	paid := proto.Clone(r).(*types.VerificationRound)
	paid.ClaimRefundSettlement.PaidAtBlock, paid.VerifierRewardSettlement.PaidAtBlock = 8, 8
	require.NoError(t, types.ValidateClaimFundingRound(c, paid))
	require.NoError(t, types.ValidateClaimRefundSettlementUpdate(r, paid))
	require.Error(t, types.ValidateClaimRefundSettlementUpdate(paid, r), "paid cannot revert")
	changed := proto.Clone(paid).(*types.VerificationRound)
	changed.ClaimRefundSettlement.Amount = "45"
	require.Error(t, types.ValidateClaimRefundSettlementUpdate(paid, changed))
}

func TestClaimFundingRawWireRefusesMergeAndNormalization(t *testing.T) {
	wrap := func(field protowire.Number, value []byte) []byte {
		return protowire.AppendBytes(protowire.AppendTag(nil, field, protowire.BytesType), value)
	}
	for _, field := range []protowire.Number{types.ClaimFundingTermsField, types.RoundClaimRefundField} {
		c := fundingClaim(types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT, 101)
		r := fundedRound(t, c, nil)
		var m proto.Message = c.FundingTerms
		if field == types.RoundClaimRefundField {
			m = r.ClaimRefundSettlement
		}
		raw, err := proto.Marshal(m)
		require.NoError(t, err)
		valid := wrap(field, raw)
		present, err := types.RawFundSettlementFieldPresent(valid, field)
		require.NoError(t, err)
		require.True(t, present)
		present, err = types.RawFundSettlementFieldPresent(nil, field)
		require.NoError(t, err)
		require.False(t, present)
		badValues := map[string][]byte{
			"duplicate outer":   append(bytes.Clone(valid), valid...),
			"wrong outer wire":  protowire.AppendVarint(protowire.AppendTag(nil, field, protowire.VarintType), 1),
			"duplicate nested":  wrap(field, append(bytes.Clone(raw), raw...)),
			"unknown nested":    wrap(field, append(bytes.Clone(raw), 0x38, 1)),
			"malformed nested":  wrap(field, []byte{0xff}),
			"huge nested":       wrap(field, make([]byte, 4097)),
			"wrong nested wire": wrap(field, []byte{0x09, 0, 0, 0, 0, 0, 0, 0, 0}),
		}
		if field == types.ClaimFundingTermsField {
			badValues["uint32 overflow"] = wrap(field, protowire.AppendVarint([]byte{0x08}, 1<<32|1))
			badValues["enum overflow"] = wrap(field, protowire.AppendVarint([]byte{0x10}, 1<<32|1))
			badValues["unknown policy"] = wrap(field, []byte{0x08, 2})
			badValues["unknown kind"] = wrap(field, []byte{0x10, 3})
			badValues["nonminimal numeric"] = wrap(field, []byte{0x08, 0x81, 0})
		} else {
			badValues["height overflow"] = wrap(field, append([]byte{0x18}, bytes.Repeat([]byte{0xff}, 11)...))
			badValues["nonminimal numeric"] = wrap(field, []byte{0x18, 0x81, 0})
		}
		for name, bad := range badValues {
			t.Run(strconv.Itoa(int(field))+"/"+name, func(t *testing.T) { _, err := types.RawFundSettlementFieldPresent(bad, field); require.Error(t, err) })
		}
	}
}

func TestClaimFundingGenesisRequiresCompleteSelectedFinancialRecords(t *testing.T) {
	c := fundingClaim(types.ClaimFundingKind_CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT, 101)
	r := fundedRound(t, c, nil)
	good := &types.GenesisState{RecordIntegrityEnabled: true, ReviewNeutralityEnabled: true, ClaimRecordsEnabled: true, FundSettlementEnabled: true,
		PendingClaims: []*types.Claim{c}, CompletedRounds: []*types.VerificationRound{r}}
	require.NoError(t, types.ValidateGenesisRounds(good))
	for name, mutate := range map[string]func(*types.GenesisState){
		"disabled":            func(g *types.GenesisState) { g.FundSettlementEnabled = false },
		"missing predecessor": func(g *types.GenesisState) { g.ClaimRecordsEnabled = false },
		"no claim":            func(g *types.GenesisState) { g.PendingClaims = nil },
		"missing terminal":    func(g *types.GenesisState) { g.CompletedRounds = nil },
		"missing pointer":     func(g *types.GenesisState) { g.PendingClaims[0].VerificationRoundId = "" },
		"dangling pointer":    func(g *types.GenesisState) { g.PendingClaims[0].VerificationRoundId = "other" },
		"erase terms":         func(g *types.GenesisState) { g.PendingClaims[0].FundingTerms = nil },
		"erase refund":        func(g *types.GenesisState) { g.CompletedRounds[0].ClaimRefundSettlement = nil },
		"extra primary round": func(g *types.GenesisState) {
			second := proto.Clone(g.CompletedRounds[0]).(*types.VerificationRound)
			second.Id = "second-round"
			g.CompletedRounds = append(g.CompletedRounds, second)
		},
	} {
		t.Run(name, func(t *testing.T) {
			bad := proto.Clone(good).(*types.GenesisState)
			mutate(bad)
			require.Error(t, types.ValidateGenesisRounds(bad))
		})
	}
	legacy := proto.Clone(good).(*types.GenesisState)
	legacy.PendingClaims[0].FundingTerms = nil
	legacy.CompletedRounds[0].ClaimRefundSettlement = nil
	require.NoError(t, types.ValidateGenesisRounds(legacy), "old retained records acquire no inferred refund")
	legacy.CompletedRounds = nil
	require.NoError(t, types.ValidateGenesisRounds(legacy), "missing legacy terminal history stays explicitly missing")
	require.True(t, types.DefaultGenesis().FundSettlementEnabled)
}
