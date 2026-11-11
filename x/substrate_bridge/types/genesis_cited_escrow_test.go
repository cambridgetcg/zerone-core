package types

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// With the cited-reward escrow marker (cited_reward_escrow_enabled) on,
// settlement escrows a cited-fact link's reward through the witness-reward
// entries (0x8C, 0x8D). These tests hold the genesis validator to that: the
// entry is accepted only when the genesis being validated carries the marker,
// it must pass every check a witness entry passes, and a link with pending
// claims is refused in both states.

const (
	citedEscrowAdapter  = "adapter-v1"
	citedEscrowID       = "att-1-1"
	citedEscrowWitness  = "att-1-2"
	citedEscrowDeadline = uint64(11)
	// The index of the cited entry's 0x8C record in citedEscrowGenesis.
	citedEscrowRewardIndex = 4
)

type citedEscrowGenesisCase struct {
	attestation *ExternalAttestation
	reward      witnessPendingRewardGenesis
	adapters    []*AdapterRegistration
	dropIndex   bool
}

func newCitedEscrowGenesisCase() *citedEscrowGenesisCase {
	return &citedEscrowGenesisCase{
		attestation: &ExternalAttestation{
			AttestationId: citedEscrowID,
			AdapterId:     citedEscrowAdapter,
			Submitter:     "zrn1citer",
			BondUzrn:      "1000000",
			Status:        AttestationStatus_ATTESTATION_STATUS_SETTLED,
			RewardUzrn:    "0",
			Link: &SubstrateLink{
				AdapterId:  citedEscrowAdapter,
				Source:     &ExternalSource{AdapterId: citedEscrowAdapter, SourceId: "source-cited"},
				CitedFacts: []*FactCitation{{FactId: "fact-1", CitationType: CitationType_CITATION_TYPE_CITES}},
			},
		},
		reward: witnessPendingRewardGenesis{
			AttestationID: citedEscrowID,
			AdapterID:     citedEscrowAdapter,
			Recipient:     "zrn1citer",
			Amount:        "1332000",
			Deadline:      citedEscrowDeadline,
		},
		adapters: []*AdapterRegistration{{
			AdapterId: citedEscrowAdapter,
			Status:    AdapterStatus_ADAPTER_STATUS_ACTIVE,
		}},
	}
}

// citedEscrowGenesis is an armed genesis holding one settled cited-fact
// attestation with its reward escrowed, as settlement leaves it with the
// marker on, beside one settled witness-only attestation with a witness
// reward escrowed.
func citedEscrowGenesis(t *testing.T, c *citedEscrowGenesisCase, markerOn bool) *GenesisState {
	t.Helper()
	witness := &ExternalAttestation{
		AttestationId: citedEscrowWitness,
		AdapterId:     citedEscrowAdapter,
		Submitter:     "zrn1witness",
		BondUzrn:      "1000000",
		Status:        AttestationStatus_ATTESTATION_STATUS_SETTLED,
		RewardUzrn:    "0",
		Link: &SubstrateLink{
			AdapterId: citedEscrowAdapter,
			Source:    &ExternalSource{AdapterId: citedEscrowAdapter, SourceId: "source-witness"},
		},
	}
	witnessReward := witnessPendingRewardGenesis{
		AttestationID: citedEscrowWitness,
		AdapterID:     citedEscrowAdapter,
		Recipient:     "zrn1witness",
		Amount:        "500",
		Deadline:      citedEscrowDeadline,
	}

	attValue, err := proto.Marshal(c.attestation)
	require.NoError(t, err)
	witnessValue, err := proto.Marshal(witness)
	require.NoError(t, err)
	rewardValue, err := json.Marshal(c.reward)
	require.NoError(t, err)
	witnessRewardValue, err := json.Marshal(witnessReward)
	require.NoError(t, err)

	entries := []*GenesisStateEntry{
		{Key: AttestationKey(c.attestation.AttestationId), Value: attValue},
		{Key: AttestationKey(witness.AttestationId), Value: witnessValue},
		{Key: AttestationByStatusKey(uint8(c.attestation.Status), c.attestation.AttestationId), Value: []byte{0x01}},
		{Key: AttestationByStatusKey(uint8(witness.Status), witness.AttestationId), Value: []byte{0x01}},
		{Key: WitnessPendingRewardKey(c.reward.AttestationID), Value: rewardValue},
		{Key: WitnessPendingRewardKey(witnessReward.AttestationID), Value: witnessRewardValue},
		{Key: WitnessDeadlineIndexKey(witnessReward.Deadline, witnessReward.AttestationID), Value: []byte{0x01}},
		{Key: SourceRefKey(citedEscrowAdapter, "source-cited"), Value: []byte(c.attestation.AttestationId)},
		{Key: SourceRefKey(citedEscrowAdapter, "source-witness"), Value: []byte(witness.AttestationId)},
		{Key: AttestationIDCounterKey, Value: binary.AppendUvarint(nil, 2)},
		{Key: DedupeArmedKey, Value: []byte{0x01}},
	}
	if !c.dropIndex {
		entries = append(entries, &GenesisStateEntry{
			Key:   WitnessDeadlineIndexKey(c.reward.Deadline, c.reward.AttestationID),
			Value: []byte{0x01},
		})
	}

	genesis := DefaultGenesis()
	genesis.CitedRewardEscrowEnabled = markerOn
	genesis.Adapters = c.adapters
	genesis.StateEntries = entries
	return genesis
}

// With the marker on in the genesis, a cited-fact escrow is accepted. With it
// off, the same entry is refused exactly as before the marker existed. The
// witness-only escrow beside it is accepted in both states.
func TestGenesisValidateCitedEscrowFollowsTheGenesisMarker(t *testing.T) {
	require.Equal(t, WitnessPendingRewardKey(citedEscrowID),
		citedEscrowGenesis(t, newCitedEscrowGenesisCase(), true).StateEntries[citedEscrowRewardIndex].Key)

	require.NoError(t, citedEscrowGenesis(t, newCitedEscrowGenesisCase(), true).Validate())
	require.EqualError(t, citedEscrowGenesis(t, newCitedEscrowGenesisCase(), false).Validate(),
		"state entry at index 4 witness reward belongs to a non-witness attestation")

	for _, markerOn := range []bool{false, true} {
		c := newCitedEscrowGenesisCase()
		c.attestation.Link.CitedFacts = nil
		require.NoError(t, citedEscrowGenesis(t, c, markerOn).Validate(), "witness-only entries, marker %v", markerOn)
	}
}

// With the marker on, a cited-fact escrow still has to be consistent with its
// attestation by every check a witness escrow meets. Each case breaks one.
func TestGenesisValidateCitedEscrowStillRefusesInconsistentEntries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*citedEscrowGenesisCase)
		want   string
	}{
		{"link with pending claims", func(c *citedEscrowGenesisCase) {
			c.attestation.Link.PendingClaims = []*PendingClaim{{ClaimContent: "pending", Domain: "d"}}
		}, "state entry at index 4 witness reward belongs to a non-witness attestation"},
		{"no link", func(c *citedEscrowGenesisCase) {
			c.attestation.Link = nil
		}, "state entry at index 4 witness reward belongs to a non-witness attestation"},
		{"adapter differs from the attestation", func(c *citedEscrowGenesisCase) {
			c.reward.AdapterID = "adapter-v2"
			c.adapters = append(c.adapters, &AdapterRegistration{AdapterId: "adapter-v2"})
		}, "state entry at index 4 witness adapter does not match its attestation"},
		{"recipient differs from the submitter", func(c *citedEscrowGenesisCase) {
			c.reward.Recipient = "zrn1other"
		}, "state entry at index 4 witness recipient does not match its attestation"},
		{"attestation not settled", func(c *citedEscrowGenesisCase) {
			c.attestation.Status = AttestationStatus_ATTESTATION_STATUS_PARTIAL
		}, "state entry at index 4 witness reward belongs to a non-settled attestation"},
		{"link adapter differs", func(c *citedEscrowGenesisCase) {
			c.attestation.Link.AdapterId = "adapter-v2"
		}, "state entry at index 4 witness link adapter does not match its attestation"},
		{"reward already paid", func(c *citedEscrowGenesisCase) {
			c.attestation.RewardUzrn = "1332000"
		}, "state entry at index 4 pending witness attestation must record zero paid reward"},
		{"adapter missing", func(c *citedEscrowGenesisCase) {
			c.attestation.AdapterId = "adapter-gone"
			c.attestation.Link.AdapterId = "adapter-gone"
			c.reward.AdapterID = "adapter-gone"
		}, `state entry at index 4 witness adapter "adapter-gone" is missing`},
		{"deadline index missing", func(c *citedEscrowGenesisCase) {
			c.dropIndex = true
		}, "state entry at index 4 is missing its witness deadline index"},
		{"zero amount", func(c *citedEscrowGenesisCase) {
			c.reward.Amount = "0"
		}, "state entry at index 4 has incomplete or invalid witness reward fields"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newCitedEscrowGenesisCase()
			tc.mutate(c)
			require.ErrorContains(t, citedEscrowGenesis(t, c, true).Validate(), tc.want)
		})
	}
}
