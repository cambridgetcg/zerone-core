package types

import (
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestGenesisValidateRejectsNilAdapter(t *testing.T) {
	genesis := DefaultGenesis()
	genesis.Adapters = []*AdapterRegistration{
		{AdapterId: "valid-adapter"},
		nil,
	}

	require.EqualError(t, genesis.Validate(), "adapter at index 1 must not be nil")
}

func TestGenesisValidateRestrictsRawStateKeyspaces(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{name: "empty", key: nil},
		{name: "params", key: ParamsKey},
		{name: "adapter record", key: AdapterKey("adapter-v1")},
		{name: "adapter status index", key: AdapterByStatusKey(1, "adapter-v1")},
		{name: "unknown prefix", key: []byte{0xFF, 0x01}},
		{name: "malformed source ref", key: append([]byte{}, SourceRefPrefix...)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			genesis := DefaultGenesis()
			genesis.StateEntries = []*GenesisStateEntry{{Key: tt.key, Value: []byte{0x01}}}
			require.ErrorContains(t, genesis.Validate(), "forbidden or malformed key")
		})
	}
}

func TestGenesisValidateRejectsDuplicateRawStateKey(t *testing.T) {
	key := AttestationKey("att-1-1")
	genesis := DefaultGenesis()
	genesis.StateEntries = []*GenesisStateEntry{
		{Key: key, Value: []byte{0x01}},
		{Key: append([]byte{}, key...), Value: []byte{0x02}},
	}

	require.ErrorContains(t, genesis.Validate(), "duplicate state entry key")
}

func TestGenesisValidateAcceptsReplayEconomicKeyspaces(t *testing.T) {
	const adapterID = "adapter-v1"
	upstream := &ExternalAttestation{
		AttestationId: "att-1-1",
		AdapterId:     adapterID,
		Submitter:     "zrn1upstream",
		Status:        AttestationStatus_ATTESTATION_STATUS_SETTLED,
		RewardUzrn:    "0",
		Link: &SubstrateLink{
			AdapterId: adapterID,
			Source:    &ExternalSource{SourceId: "source-upstream"},
		},
	}
	downstream := &ExternalAttestation{
		AttestationId: "att-1-2",
		AdapterId:     adapterID,
		Submitter:     "zrn1downstream",
		Status:        AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
		Link: &SubstrateLink{
			AdapterId:     adapterID,
			Source:        &ExternalSource{SourceId: "source-downstream"},
			PendingClaims: []*PendingClaim{{ClaimContent: "pending"}},
		},
	}
	edge := &LineageEdge{
		UpstreamAttestationId:   upstream.AttestationId,
		DownstreamAttestationId: downstream.AttestationId,
		CitationType:            CitationType_CITATION_TYPE_EXTENDS,
		SettlementPaymentUzrn:   "1",
	}
	accumulator := &LineageRoyaltyAccumulator{
		AttestationId:  upstream.AttestationId,
		CumulativeUzrn: "1",
	}
	witnessReward, err := json.Marshal(witnessPendingRewardGenesis{
		AttestationID: upstream.AttestationId,
		AdapterID:     adapterID,
		Recipient:     "zrn1upstream",
		Amount:        "222000",
		Deadline:      10,
	})
	require.NoError(t, err)

	upstreamValue, err := proto.Marshal(upstream)
	require.NoError(t, err)
	downstreamValue, err := proto.Marshal(downstream)
	require.NoError(t, err)
	edgeValue, err := proto.Marshal(edge)
	require.NoError(t, err)
	accumulatorValue, err := proto.Marshal(accumulator)
	require.NoError(t, err)

	edgeID := EdgeID(upstream.AttestationId, downstream.AttestationId)
	genesis := DefaultGenesis()
	genesis.Adapters = []*AdapterRegistration{{
		AdapterId:         adapterID,
		Status:            AdapterStatus_ADAPTER_STATUS_ACTIVE,
		WitnessRewardUzrn: "222000",
	}}
	genesis.StateEntries = []*GenesisStateEntry{
		{Key: LineageEdgeKey(edgeID), Value: edgeValue},
		{Key: LineageByUpstreamKey(upstream.AttestationId, edgeID), Value: []byte{0x01}},
		{Key: LineageByDownstreamKey(downstream.AttestationId, edgeID), Value: []byte{0x01}},
		{Key: LineageRoyaltyAccumulatorKey(upstream.AttestationId), Value: accumulatorValue},
		{Key: AttestationKey(upstream.AttestationId), Value: upstreamValue},
		{Key: AttestationKey(downstream.AttestationId), Value: downstreamValue},
		{Key: AttestationByStatusKey(uint8(upstream.Status), upstream.AttestationId), Value: []byte{0x01}},
		{Key: AttestationByStatusKey(uint8(downstream.Status), downstream.AttestationId), Value: []byte{0x01}},
		{Key: PendingFactIndexKey("claim-1"), Value: []byte(downstream.AttestationId)},
		{Key: AttestationPendingClaimsKey(downstream.AttestationId, "claim-1"), Value: []byte{0x01}},
		{Key: AttestationIDCounterKey, Value: binary.AppendUvarint(nil, 2)},
		{Key: WitnessPendingRewardKey(upstream.AttestationId), Value: witnessReward},
		{Key: WitnessDeadlineIndexKey(10, upstream.AttestationId), Value: []byte{0x01}},
		{Key: SourceRefKey(adapterID, upstream.Link.Source.SourceId), Value: []byte(upstream.AttestationId)},
		{Key: SourceRefKey(adapterID, downstream.Link.Source.SourceId), Value: []byte(downstream.AttestationId)},
		{Key: DedupeArmedKey, Value: []byte{0x01}},
	}

	require.NoError(t, genesis.Validate())
}

func TestGenesisValidateRejectsInvalidPrimaryEncoding(t *testing.T) {
	genesis := DefaultGenesis()
	genesis.StateEntries = []*GenesisStateEntry{{
		Key:   AttestationKey("att-1-1"),
		Value: []byte{0x01},
	}}

	require.ErrorContains(t, genesis.Validate(), "invalid external attestation")
}

func TestGenesisValidateRejectsAttestationKeyValueIdentityMismatch(t *testing.T) {
	attestationValue, err := proto.Marshal(&ExternalAttestation{
		AttestationId: "att-1-2",
		Status:        AttestationStatus_ATTESTATION_STATUS_SETTLED,
	})
	require.NoError(t, err)

	genesis := DefaultGenesis()
	genesis.StateEntries = []*GenesisStateEntry{{
		Key:   AttestationKey("att-1-1"),
		Value: attestationValue,
	}}

	require.ErrorContains(t, genesis.Validate(), "attestation key does not match")
}

func TestGenesisValidateRejectsWitnessKeyValueIdentityMismatch(t *testing.T) {
	witnessReward, err := json.Marshal(witnessPendingRewardGenesis{
		AttestationID: "att-1-2",
		AdapterID:     "adapter-v1",
		Recipient:     "zrn1recipient",
		Amount:        "222000",
		Deadline:      10,
	})
	require.NoError(t, err)

	genesis := DefaultGenesis()
	genesis.StateEntries = []*GenesisStateEntry{{
		Key:   WitnessPendingRewardKey("att-1-1"),
		Value: witnessReward,
	}}

	require.ErrorContains(t, genesis.Validate(), "witness reward key does not match")
}

func TestGenesisValidateRejectsMissingAndOrphanSecondaryIndexes(t *testing.T) {
	attestation := &ExternalAttestation{
		AttestationId: "att-1-1",
		Status:        AttestationStatus_ATTESTATION_STATUS_SETTLED,
	}
	attestationValue, err := proto.Marshal(attestation)
	require.NoError(t, err)

	t.Run("missing status index", func(t *testing.T) {
		genesis := DefaultGenesis()
		genesis.StateEntries = []*GenesisStateEntry{{
			Key:   AttestationKey(attestation.AttestationId),
			Value: attestationValue,
		}}
		require.ErrorContains(t, genesis.Validate(), "missing its attestation status index")
	})

	t.Run("orphan status index", func(t *testing.T) {
		genesis := DefaultGenesis()
		genesis.StateEntries = []*GenesisStateEntry{{
			Key:   AttestationByStatusKey(uint8(attestation.Status), attestation.AttestationId),
			Value: []byte{0x01},
		}}
		require.ErrorContains(t, genesis.Validate(), "orphan or mismatched attestation status index")
	})
}

func TestGenesisValidateRejectsArmedStateWithoutCounterOrSourceHolder(t *testing.T) {
	const adapterID = "adapter-v1"
	attestation := &ExternalAttestation{
		AttestationId: "att-1-1",
		AdapterId:     adapterID,
		Status:        AttestationStatus_ATTESTATION_STATUS_SETTLED,
		Link: &SubstrateLink{
			AdapterId: adapterID,
			Source:    &ExternalSource{SourceId: "source-1"},
		},
	}
	attestationValue, err := proto.Marshal(attestation)
	require.NoError(t, err)

	baseEntries := []*GenesisStateEntry{
		{Key: AttestationKey(attestation.AttestationId), Value: attestationValue},
		{
			Key:   AttestationByStatusKey(uint8(attestation.Status), attestation.AttestationId),
			Value: []byte{0x01},
		},
		{Key: DedupeArmedKey, Value: []byte{0x01}},
	}

	t.Run("counter", func(t *testing.T) {
		genesis := DefaultGenesis()
		genesis.StateEntries = append([]*GenesisStateEntry(nil), baseEntries...)
		require.ErrorContains(t, genesis.Validate(), "must include the attestation counter")
	})

	t.Run("source holder", func(t *testing.T) {
		genesis := DefaultGenesis()
		genesis.StateEntries = append(
			append([]*GenesisStateEntry(nil), baseEntries...),
			&GenesisStateEntry{Key: AttestationIDCounterKey, Value: binary.AppendUvarint(nil, 1)},
		)
		require.ErrorContains(t, genesis.Validate(), "has no source-ref holder")
	})
}

func TestGenesisValidateRejectsNonCanonicalOrRegressedCounter(t *testing.T) {
	t.Run("non-canonical uvarint", func(t *testing.T) {
		genesis := DefaultGenesis()
		genesis.StateEntries = []*GenesisStateEntry{{
			Key:   AttestationIDCounterKey,
			Value: []byte{0x81, 0x00}, // value 1, encoded with a redundant byte
		}}
		require.ErrorContains(t, genesis.Validate(), "non-canonical attestation counter")
	})

	t.Run("below generated id sequence", func(t *testing.T) {
		attestation := &ExternalAttestation{
			AttestationId: "att-9-2",
			Status:        AttestationStatus_ATTESTATION_STATUS_SETTLED,
		}
		attestationValue, err := proto.Marshal(attestation)
		require.NoError(t, err)

		genesis := DefaultGenesis()
		genesis.StateEntries = []*GenesisStateEntry{
			{Key: AttestationKey(attestation.AttestationId), Value: attestationValue},
			{
				Key:   AttestationByStatusKey(uint8(attestation.Status), attestation.AttestationId),
				Value: []byte{0x01},
			},
			{Key: AttestationIDCounterKey, Value: binary.AppendUvarint(nil, 1)},
		}
		require.ErrorContains(t, genesis.Validate(), "below stored attestation sequence")
	})
}

func TestGenesisValidateRejectsReleasableSourceHolderOverMintedTwin(t *testing.T) {
	const adapterID = "adapter-v1"
	settled := &ExternalAttestation{
		AttestationId: "att-1-1",
		Status:        AttestationStatus_ATTESTATION_STATUS_SETTLED,
		Link: &SubstrateLink{
			AdapterId: adapterID,
			Source:    &ExternalSource{SourceId: "shared-source"},
		},
	}
	inFlight := &ExternalAttestation{
		AttestationId: "att-1-2",
		Status:        AttestationStatus_ATTESTATION_STATUS_READY,
		Link: &SubstrateLink{
			AdapterId: adapterID,
			Source:    &ExternalSource{SourceId: "shared-source"},
		},
	}
	settledValue, err := proto.Marshal(settled)
	require.NoError(t, err)
	inFlightValue, err := proto.Marshal(inFlight)
	require.NoError(t, err)

	genesis := DefaultGenesis()
	genesis.StateEntries = []*GenesisStateEntry{
		{Key: AttestationKey(settled.AttestationId), Value: settledValue},
		{Key: AttestationKey(inFlight.AttestationId), Value: inFlightValue},
		{
			Key:   AttestationByStatusKey(uint8(settled.Status), settled.AttestationId),
			Value: []byte{0x01},
		},
		{
			Key:   AttestationByStatusKey(uint8(inFlight.Status), inFlight.AttestationId),
			Value: []byte{0x01},
		},
		{Key: AttestationIDCounterKey, Value: binary.AppendUvarint(nil, 2)},
		{
			Key:   SourceRefKey(adapterID, "shared-source"),
			Value: []byte(inFlight.AttestationId),
		},
		{Key: DedupeArmedKey, Value: []byte{0x01}},
	}

	require.ErrorContains(t, genesis.Validate(), "releasable while a stronger claim exists")
}
