package types_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func admissionDigest(label string) string {
	sum := sha256.Sum256([]byte(label))
	return hex.EncodeToString(sum[:])
}

func admissionAddress(t *testing.T, marker byte) string {
	t.Helper()
	address, err := bech32.ConvertAndEncode("zrn", bytesOf(marker, 20))
	require.NoError(t, err)
	return address
}

func bytesOf(value byte, count int) []byte {
	result := make([]byte, count)
	for i := range result {
		result[i] = value
	}
	return result
}

func admissionUint64(value uint64) *uint64 {
	return &value
}

func validAdmissionCandidate(t *testing.T, marker byte, stake, qualification uint64) types.VerifierAdmissionCandidateEvidence {
	t.Helper()
	return types.VerifierAdmissionCandidateEvidence{
		ValidatorAddress:       admissionAddress(t, marker),
		ConsensusKeyHash:       admissionDigest(string([]byte{'k', marker})),
		ControllerIdentityHash: admissionDigest(string([]byte{'c', marker})),
		ControllerReviewStatus: types.ControllerReviewReviewed,
		BondStatus:             types.VerifierBonded,
		BondedStake:            admissionUint64(stake),
		QualificationStatus:    types.DomainQualified,
		QualificationWeight:    admissionUint64(qualification),
		ProviderEvidenceHash:   admissionDigest(string([]byte{'e', marker})),
	}
}

func validAdmissionInput(t *testing.T) types.VerifierAdmissionSnapshotInput {
	t.Helper()
	return types.VerifierAdmissionSnapshotInput{
		RoundID:           "round-compute-7",
		ClaimID:           "claim-compute-7",
		Domain:            "mathematics",
		ObservedHeight:    770,
		ObservedAppHash:   admissionDigest("app-height-770"),
		InputEvidenceRoot: admissionDigest("complete-provider-manifest-height-770"),
		MinimumSeats:      2,
		Candidates: []types.VerifierAdmissionCandidateEvidence{
			validAdmissionCandidate(t, 0x03, 300, 70),
			validAdmissionCandidate(t, 0x01, 100, 90),
			validAdmissionCandidate(t, 0x02, 200, 80),
		},
	}
}

func cloneAdmissionSnapshot(t *testing.T, snapshot *types.VerifierAdmissionSnapshot) *types.VerifierAdmissionSnapshot {
	t.Helper()
	encoded, err := json.Marshal(snapshot)
	require.NoError(t, err)
	var cloned types.VerifierAdmissionSnapshot
	require.NoError(t, json.Unmarshal(encoded, &cloned))
	return &cloned
}

func TestBuildVerifierAdmissionSnapshot_CanonicalHeightPinnedEvidence(t *testing.T) {
	input := validAdmissionInput(t)
	snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
	require.NoError(t, err)
	require.NoError(t, types.VerifyVerifierAdmissionSnapshot(snapshot))
	require.Equal(t, types.VerifierAdmissionSnapshotVersion, snapshot.Version)
	require.Equal(t, input.ObservedHeight, snapshot.ObservedHeight)
	require.Equal(t, input.ObservedAppHash, snapshot.ObservedAppHash)
	require.Equal(t, input.InputEvidenceRoot, snapshot.InputEvidenceRoot)
	require.Len(t, snapshot.Observations, 3)
	require.Len(t, snapshot.Seats, 3)

	for i := 1; i < len(snapshot.Observations); i++ {
		require.Less(t, snapshot.Observations[i-1].ValidatorAddress, snapshot.Observations[i].ValidatorAddress)
	}
	for _, seat := range snapshot.Seats {
		require.Positive(t, seat.BondedStake)
		require.Equal(t, seat.BondedStake, seat.SelectionWeight,
			"selection has no balance, virtual-stake, or minimum-weight fallback")
	}

	reversed := validAdmissionInput(t)
	for left, right := 0, len(reversed.Candidates)-1; left < right; left, right = left+1, right-1 {
		reversed.Candidates[left], reversed.Candidates[right] = reversed.Candidates[right], reversed.Candidates[left]
	}
	canonicalAgain, err := types.BuildVerifierAdmissionSnapshot(reversed)
	require.NoError(t, err)
	require.Equal(t, snapshot, canonicalAgain,
		"provider iteration order must not change the panel evidence")
}

func TestBuildVerifierAdmissionSnapshot_KnownIneligibleCandidatesHaveZeroWeight(t *testing.T) {
	valid := validAdmissionCandidate(t, 0x01, 111, 90)
	unreviewed := validAdmissionCandidate(t, 0x02, 222, 90)
	unreviewed.ControllerReviewStatus = types.ControllerReviewUnreviewed
	unbonded := validAdmissionCandidate(t, 0x03, 0, 90)
	unbonded.BondStatus = types.VerifierUnbonded
	zeroStake := validAdmissionCandidate(t, 0x04, 0, 90)
	unqualified := validAdmissionCandidate(t, 0x05, 555, 0)
	unqualified.QualificationStatus = types.DomainUnqualified
	zeroQualification := validAdmissionCandidate(t, 0x06, 666, 0)

	input := validAdmissionInput(t)
	input.MinimumSeats = 1
	input.Candidates = []types.VerifierAdmissionCandidateEvidence{
		zeroQualification,
		unqualified,
		zeroStake,
		unbonded,
		unreviewed,
		valid,
	}
	snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
	require.NoError(t, err)
	require.Len(t, snapshot.Seats, 1)
	require.Equal(t, valid.ValidatorAddress, snapshot.Seats[0].ValidatorAddress)

	wantReasons := map[string]string{
		unreviewed.ValidatorAddress:        types.VerifierAdmissionExclusionUnreviewedController,
		unbonded.ValidatorAddress:          types.VerifierAdmissionExclusionUnbonded,
		zeroStake.ValidatorAddress:         types.VerifierAdmissionExclusionZeroBondedStake,
		unqualified.ValidatorAddress:       types.VerifierAdmissionExclusionUnqualified,
		zeroQualification.ValidatorAddress: types.VerifierAdmissionExclusionZeroQualification,
	}
	for _, observation := range snapshot.Observations {
		if observation.ValidatorAddress == valid.ValidatorAddress {
			require.True(t, observation.Admitted)
			require.Equal(t, uint64(111), observation.SelectionWeight)
			continue
		}
		require.False(t, observation.Admitted)
		require.Zero(t, observation.SelectionWeight,
			"every known-ineligible candidate must have exactly zero weight")
		require.Equal(t, wantReasons[observation.ValidatorAddress], observation.ExclusionReason)
	}
	require.NoError(t, types.VerifyVerifierAdmissionSnapshot(snapshot))
}

func TestBuildVerifierAdmissionSnapshot_UnknownEvidenceFailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.VerifierAdmissionSnapshotInput)
		want   string
	}{
		{name: "height", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.ObservedHeight = 0 }, want: "observed_height"},
		{name: "app hash", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.ObservedAppHash = "" }, want: "observed_app_hash"},
		{name: "input evidence", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.InputEvidenceRoot = "" }, want: "input_evidence_root"},
		{name: "consensus key", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.Candidates[0].ConsensusKeyHash = "" }, want: "consensus_key_hash"},
		{name: "controller identity", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.Candidates[0].ControllerIdentityHash = "" }, want: "controller_identity_hash"},
		{name: "controller review", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.Candidates[0].ControllerReviewStatus = "" }, want: "controller review status"},
		{name: "bond status", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.Candidates[0].BondStatus = "" }, want: "bond status"},
		{name: "bonded stake", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.Candidates[0].BondedStake = nil }, want: "bonded stake is unknown"},
		{name: "qualification status", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.Candidates[0].QualificationStatus = "" }, want: "qualification status"},
		{name: "qualification weight", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.Candidates[0].QualificationWeight = nil }, want: "qualification weight is unknown"},
		{name: "provider evidence", mutate: func(in *types.VerifierAdmissionSnapshotInput) { in.Candidates[0].ProviderEvidenceHash = "" }, want: "provider_evidence_hash"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validAdmissionInput(t)
			test.mutate(&input)
			snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
			require.Nil(t, snapshot)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestBuildVerifierAdmissionSnapshot_RejectsControllerAndValidatorAmbiguity(t *testing.T) {
	t.Run("one validator appears twice", func(t *testing.T) {
		input := validAdmissionInput(t)
		input.Candidates[1] = input.Candidates[0]
		snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
		require.Nil(t, snapshot)
		require.ErrorContains(t, err, "duplicate validator_address")
	})

	t.Run("one controller claims two seats", func(t *testing.T) {
		input := validAdmissionInput(t)
		input.Candidates[1].ControllerIdentityHash = input.Candidates[0].ControllerIdentityHash
		snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
		require.Nil(t, snapshot)
		require.ErrorContains(t, err, "ambiguously controls validators")
	})

	t.Run("one consensus key claims two validators", func(t *testing.T) {
		input := validAdmissionInput(t)
		input.Candidates[1].ConsensusKeyHash = input.Candidates[0].ConsensusKeyHash
		snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
		require.Nil(t, snapshot)
		require.ErrorContains(t, err, "ambiguously identifies validators")
	})
}

func TestVerifyVerifierAdmissionSnapshot_RejectsConsensusKeyAliasBeforeDigest(t *testing.T) {
	snapshot, err := types.BuildVerifierAdmissionSnapshot(validAdmissionInput(t))
	require.NoError(t, err)

	mutated := cloneAdmissionSnapshot(t, snapshot)
	mutated.Observations[1].ConsensusKeyHash = mutated.Observations[0].ConsensusKeyHash
	mutated.Seats[1].ConsensusKeyHash = mutated.Seats[0].ConsensusKeyHash
	err = types.VerifyVerifierAdmissionSnapshot(mutated)
	require.ErrorContains(t, err, "snapshot consensus_key_hash")
}

func TestBuildVerifierAdmissionSnapshot_RefusesInadequateReviewedPanel(t *testing.T) {
	input := validAdmissionInput(t)
	input.MinimumSeats = 3
	input.Candidates[0].BondedStake = admissionUint64(0)
	snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
	require.Nil(t, snapshot)
	require.ErrorContains(t, err, "below required minimum")
}

func TestVerifyVerifierAdmissionSnapshot_DetectsEveryEvidenceClassMutation(t *testing.T) {
	snapshot, err := types.BuildVerifierAdmissionSnapshot(validAdmissionInput(t))
	require.NoError(t, err)

	tests := []struct {
		name   string
		mutate func(*types.VerifierAdmissionSnapshot)
	}{
		{name: "height", mutate: func(s *types.VerifierAdmissionSnapshot) { s.ObservedHeight++ }},
		{name: "domain", mutate: func(s *types.VerifierAdmissionSnapshot) { s.Domain = "physics" }},
		{name: "app hash", mutate: func(s *types.VerifierAdmissionSnapshot) { s.ObservedAppHash = admissionDigest("other-app") }},
		{name: "input evidence", mutate: func(s *types.VerifierAdmissionSnapshot) { s.InputEvidenceRoot = admissionDigest("other-input") }},
		{name: "provider evidence", mutate: func(s *types.VerifierAdmissionSnapshot) {
			s.Observations[0].ProviderEvidenceHash = admissionDigest("other-record")
		}},
		{name: "bonded stake", mutate: func(s *types.VerifierAdmissionSnapshot) { s.Observations[0].BondedStake++ }},
		{name: "qualification", mutate: func(s *types.VerifierAdmissionSnapshot) { s.Observations[0].QualificationWeight++ }},
		{name: "decision", mutate: func(s *types.VerifierAdmissionSnapshot) { s.Observations[0].SelectionWeight++ }},
		{name: "seat", mutate: func(s *types.VerifierAdmissionSnapshot) { s.Seats[0].SelectionWeight++ }},
		{name: "observation order", mutate: func(s *types.VerifierAdmissionSnapshot) {
			s.Observations[0], s.Observations[1] = s.Observations[1], s.Observations[0]
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutated := cloneAdmissionSnapshot(t, snapshot)
			test.mutate(mutated)
			require.Error(t, types.VerifyVerifierAdmissionSnapshot(mutated))
		})
	}
}

func TestBuildVerifierAdmissionSnapshot_RejectsNoncanonicalAddressAndQualificationContradiction(t *testing.T) {
	t.Run("uppercase bech32 alias", func(t *testing.T) {
		input := validAdmissionInput(t)
		input.Candidates[0].ValidatorAddress = strings.ToUpper(input.Candidates[0].ValidatorAddress)
		snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
		require.Nil(t, snapshot)
		require.ErrorContains(t, err, "canonical lowercase")
	})

	t.Run("unqualified nonzero weight", func(t *testing.T) {
		input := validAdmissionInput(t)
		input.Candidates[0].QualificationStatus = types.DomainUnqualified
		snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
		require.Nil(t, snapshot)
		require.ErrorContains(t, err, "unqualified observation has nonzero qualification weight")
	})

	t.Run("unbonded nonzero SDK-bonded stake", func(t *testing.T) {
		input := validAdmissionInput(t)
		input.Candidates[0].BondStatus = types.VerifierUnbonded
		snapshot, err := types.BuildVerifierAdmissionSnapshot(input)
		require.Nil(t, snapshot)
		require.ErrorContains(t, err, "unbonded observation has nonzero SDK-bonded stake")
	})
}

func TestVerifierAdmissionSnapshot_CanonicalVector(t *testing.T) {
	snapshot, err := types.BuildVerifierAdmissionSnapshot(validAdmissionInput(t))
	require.NoError(t, err)
	require.Equal(t,
		"8ee5fb11bf2ea69cbefd6f236614830ffd8d90c5009f291835902ac3b4a48d16",
		snapshot.SnapshotHash,
	)
}
