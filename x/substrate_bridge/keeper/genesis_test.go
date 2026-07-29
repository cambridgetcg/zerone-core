package keeper_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/substrate_bridge/keeper"
	"github.com/zerone-chain/zerone/x/substrate_bridge/types"
)

func TestGenesisRoundTripPreservesReplayAndEconomicState(t *testing.T) {
	source, sourceCtx := setupSubstrateBridgeKeeperWithBank(t)
	const adapterID = "agenttool-invocation-v1"
	require.NoError(t, source.WriteAdapter(sourceCtx, &types.AdapterRegistration{
		AdapterId:         adapterID,
		Status:            types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		WitnessRewardUzrn: "222000",
	}))

	upstreamID := source.NextAttestationID(sourceCtx)
	downstreamID := source.NextAttestationID(sourceCtx)
	require.Equal(t, "att-1-1", upstreamID)
	require.Equal(t, "att-1-2", downstreamID)

	upstream := &types.ExternalAttestation{
		AttestationId:    upstreamID,
		AdapterId:        adapterID,
		WorkClassId:      "translation",
		Submitter:        testSubmitter("upstream"),
		BondUzrn:         "222000",
		Link:             sourcedLink(adapterID, "source-upstream", 10),
		Status:           types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
		SubmittedAtBlock: 10,
		RewardUzrn:       "0",
	}
	downstream := &types.ExternalAttestation{
		AttestationId:    downstreamID,
		AdapterId:        adapterID,
		WorkClassId:      "training",
		Submitter:        testSubmitter("downstream"),
		BondUzrn:         "222000",
		Link:             sourcedLink(adapterID, "source-downstream", 20),
		Status:           types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
		SubmittedAtBlock: 20,
	}
	require.NoError(t, source.WriteAttestation(sourceCtx, upstream))
	require.NoError(t, source.WriteAttestation(sourceCtx, downstream))

	source.SetSourceRef(sourceCtx, adapterID, "source-upstream", upstreamID)
	source.SetSourceRef(sourceCtx, adapterID, "source-downstream", downstreamID)
	require.NoError(t, source.LinkPendingClaim(sourceCtx, "legacy-claim-1", downstreamID))
	require.NoError(t, source.CreateLineageEdge(sourceCtx, &types.LineageEdge{
		UpstreamAttestationId:   upstreamID,
		DownstreamAttestationId: downstreamID,
		CitationType:            types.CitationType_CITATION_TYPE_EXTENDS,
		ContributionShareBps:    2500,
		SettlementPaymentUzrn:   "111",
	}))
	source.WriteLineageAccumulator(sourceCtx, &types.LineageRoyaltyAccumulator{
		AttestationId:     upstreamID,
		CumulativeUzrn:    "111",
		LastUpdatedBlock:  20,
		IncomingEdgeCount: 1,
	})
	source.SetWitnessPendingReward(sourceCtx, keeper.WitnessPendingReward{
		AttestationId: upstreamID,
		AdapterId:     adapterID,
		Recipient:     upstream.Submitter,
		Amount:        "222000",
		Deadline:      99,
	})

	exported := source.ExportGenesis(sourceCtx)
	require.NoError(t, exported.Validate())
	require.NotEmpty(t, exported.StateEntries)
	for _, entry := range exported.StateEntries {
		require.True(t, types.IsAllowedGenesisStateKey(entry.Key), "unexpected exported key %x", entry.Key)
	}

	imported, importedCtx := newUnarmedKeeper(t)
	require.NoError(t, imported.InitGenesis(importedCtx, exported))
	require.True(t, proto.Equal(exported, imported.ExportGenesis(importedCtx)),
		"export/import/export must preserve every replay/economic entry exactly")

	require.True(t, imported.IsDedupeArmed(importedCtx))
	holder, found := imported.GetSourceRef(importedCtx, adapterID, "source-upstream")
	require.True(t, found)
	require.Equal(t, upstreamID, holder)

	var awaiting []string
	imported.IterateAttestationsByStatus(
		importedCtx,
		types.AttestationStatus_ATTESTATION_STATUS_AWAITING_RESOLUTION,
		func(id string) bool {
			awaiting = append(awaiting, id)
			return false
		},
	)
	require.Equal(t, []string{downstreamID}, awaiting)
	require.Equal(t, []string{"legacy-claim-1"}, imported.PendingClaimsFor(importedCtx, downstreamID))

	var forwardEdges []*types.LineageEdge
	imported.IterateForwardLineage(importedCtx, upstreamID, func(edge *types.LineageEdge) bool {
		forwardEdges = append(forwardEdges, edge)
		return false
	})
	require.Len(t, forwardEdges, 1)
	require.Equal(t, downstreamID, forwardEdges[0].DownstreamAttestationId)

	accumulator, found := imported.GetLineageAccumulator(importedCtx, upstreamID)
	require.True(t, found)
	require.Equal(t, "111", accumulator.CumulativeUzrn)
	pendingReward, found := imported.GetWitnessPendingReward(importedCtx, upstreamID)
	require.True(t, found)
	require.Equal(t, uint64(99), pendingReward.Deadline)

	// The counter is singleton state rather than derivable metadata. Preserving it
	// keeps post-import IDs on the exact pre-export sequence.
	require.Equal(t, "att-1-3", imported.NextAttestationID(importedCtx))

	// Regression: the bank export retains previously minted supply, so the
	// imported source-holder must still reject a hash-varied replay before bond
	// movement or another settlement can mint.
	_, err := submitSourced(
		t,
		imported,
		importedCtx,
		sourcedLink(adapterID, "source-upstream", 999),
	)
	require.ErrorIs(t, err, types.ErrDuplicateSource)
}

func TestGenesisRoundTripPreservesUnarmedFailClosedState(t *testing.T) {
	source, sourceCtx := newUnarmedKeeper(t)
	const adapterID = "agenttool-invocation-v1"
	registerActiveAdapter(t, source, sourceCtx, adapterID)
	require.NoError(t, source.WriteAttestation(sourceCtx, &types.ExternalAttestation{
		AttestationId:    "att-1-1",
		AdapterId:        adapterID,
		Submitter:        testSubmitter("historical"),
		Status:           types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
		SubmittedAtBlock: 1,
		Link:             sourcedLink(adapterID, "historical-source", 1),
	}))
	require.False(t, source.IsDedupeArmed(sourceCtx))

	exported := source.ExportGenesis(sourceCtx)
	imported, importedCtx := newUnarmedKeeper(t)
	require.NoError(t, imported.InitGenesis(importedCtx, exported))
	require.False(t, imported.IsDedupeArmed(importedCtx),
		"absence of the arming marker must survive when imported history exists")

	_, err := submitSourced(
		t,
		imported,
		importedCtx,
		sourcedLink(adapterID, "new-source", 2),
	)
	require.ErrorIs(t, err, types.ErrDedupeNotArmed)
}

func TestInitGenesisRejectsWitnessIdentityMismatchBeforeItCanRepeatAfterRelaunch(t *testing.T) {
	source, sourceCtx, _, _ := setupSubstrateBridgeKeeperFull(t)
	const adapterID = "agenttool-invocation-v1"
	require.NoError(t, source.WriteAdapter(sourceCtx, &types.AdapterRegistration{
		AdapterId:         adapterID,
		Status:            types.AdapterStatus_ADAPTER_STATUS_ACTIVE,
		WitnessRewardUzrn: "222000",
	}))

	attestationID := source.NextAttestationID(sourceCtx)
	submitter := testSubmitter("witness-relaunch")
	require.NoError(t, source.WriteAttestation(sourceCtx, &types.ExternalAttestation{
		AttestationId: attestationID,
		AdapterId:     adapterID,
		Submitter:     submitter,
		Status:        types.AttestationStatus_ATTESTATION_STATUS_SETTLED,
		RewardUzrn:    "0",
		Link:          sourcedLink(adapterID, "source-relaunch", 1),
	}))
	source.SetSourceRef(sourceCtx, adapterID, "source-relaunch", attestationID)
	source.SetWitnessPendingReward(sourceCtx, keeper.WitnessPendingReward{
		AttestationId: attestationID,
		AdapterId:     adapterID,
		Recipient:     submitter,
		Amount:        "222000",
		Deadline:      10,
	})

	exported := source.ExportGenesis(sourceCtx)
	require.NoError(t, exported.Validate())

	// If this embedded ID were imported under the original key, the sweep
	// would load it by attestationID but delete the pending/deadline keys using
	// "other-attestation". The original due entry would then pay every block.
	maliciousValue, err := json.Marshal(keeper.WitnessPendingReward{
		AttestationId: "other-attestation",
		AdapterId:     adapterID,
		Recipient:     submitter,
		Amount:        "222000",
		Deadline:      10,
	})
	require.NoError(t, err)
	for _, entry := range exported.StateEntries {
		if len(entry.Key) > 0 && entry.Key[0] == types.WitnessPendingRewardPrefix[0] {
			entry.Value = maliciousValue
		}
	}

	imported, importedCtx, _, vesting := newUnarmedKeeperFull(t)
	err = imported.InitGenesis(importedCtx, exported)
	require.ErrorContains(t, err, "witness reward key does not match its attestation_id")
	require.False(t, imported.IsDedupeArmed(importedCtx), "validation must fail before any imported state is written")
	_, found := imported.GetWitnessPendingReward(importedCtx, attestationID)
	require.False(t, found)

	require.NoError(t, imported.BeginBlocker(importedCtx.WithBlockHeight(20)))
	require.Empty(t, vesting.minted, "a rejected relaunch snapshot must never reach the mint path")
}
