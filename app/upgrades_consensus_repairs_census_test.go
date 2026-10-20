package app

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"

	"testing"

	"cosmossdk.io/core/header"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	cc "github.com/zerone-chain/zerone/x/capture_challenge/types"
	cd "github.com/zerone-chain/zerone/x/capture_defense/types"
	emergency "github.com/zerone-chain/zerone/x/emergency/types"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	ibcrl "github.com/zerone-chain/zerone/x/ibcratelimit/types"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	bridge "github.com/zerone-chain/zerone/x/substrate_bridge/types"
	"google.golang.org/protobuf/proto"
)

func TestConsensusRepairsRunsInRealPreBlock(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	height := app.CommitMultiStore().Commit().Version + 1
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: height}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	ctx = ctx.WithBlockHeight(height).WithHeaderInfo(header.Info{Height: height, ChainID: ctx.ChainID()})
	cache, write := ctx.CacheContext()
	_, err := app.PotPreBlocker(cache, &abci.RequestFinalizeBlock{Height: height})
	require.NoError(t, err)
	for _, marker := range app.ConsensusRepairsV1Markers() {
		on, err := marker.Enabled(cache)
		require.NoError(t, err)
		require.True(t, on)
		off, err := marker.Enabled(ctx)
		require.NoError(t, err)
		require.False(t, off)
	}
	write()
	done, err := app.UpgradeKeeper.GetDoneHeight(ctx, plan.Name)
	require.NoError(t, err)
	require.Equal(t, height, done)
}

func TestConsensusRepairsCensusRetainsAndSurfacesProspectiveChanges(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	ctx = ctx.WithBlockHeight(100)
	putProto := func(store string, key []byte, value proto.Message) {
		data, err := proto.Marshal(value)
		require.NoError(t, err)
		ctx.KVStore(app.keys[store]).Set(key, data)
	}
	putJSON := func(store string, key []byte, value interface{}) {
		data, err := json.Marshal(value)
		require.NoError(t, err)
		ctx.KVStore(app.keys[store]).Set(key, data)
	}
	// A valid historical runoff pair (all original parent IDs remain intact).
	for id := uint64(1); id <= 4; id++ {
		p := &gov.SeatElectionProposal{ProposalId: id, Stage: "runoff"}
		if id > 2 {
			p.IsRunoff = true
			p.Stage = "voting"
			p.RunoffParentIds = []uint64{1, 2}
		}
		putJSON("zerone_gov", gov.SeatElectionKey(id), p)
	}
	ctx.KVStore(app.keys["zerone_gov"]).Set([]byte{0x12}, binary.BigEndian.AppendUint64(nil, 5))
	aug := &knowledge.Augmentation{Id: "held", Verdict: knowledge.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING, VerdictVotes: []knowledge.AugmentationVerdict{knowledge.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING}}
	putProto("knowledge", knowledge.AugmentationKey(aug.Id), aug)
	putProto("capture_defense", []byte("\x05physics/old-round"), &cd.VerificationHistoryEntry{Domain: "physics", RoundId: "old-round", SubmitBlocks: []uint64{1, 1}})
	haltBytes, err := proto.Marshal(&emergency.EmergencyHaltProposal{Id: "unknown", Reason: "unclassified"})
	require.NoError(t, err)
	putProto("emergency", emergency.CeremonyKey("unknown"), &emergency.EmergencyCeremony{Id: "unknown", Type: "halt", Phase: "finalized", StartBlock: 1, ProposalData: haltBytes})
	putProto("substrate_bridge", bridge.AdapterKey("distinguished"), &bridge.AdapterRegistration{AdapterId: "distinguished", RequiredQualificationDomain: "physics", MinQualificationStatus: bridge.QualificationStatus_QUALIFICATION_STATUS_DISTINGUISHED})
	flowKey := append([]byte("\x02channel-0\x00"), binary.BigEndian.AppendUint64(nil, 1)...)
	putJSON("zerone_ibcrl", flowKey, map[string]interface{}{"channel_id": "channel-0", "sequence": 1, "denom": "uzrn", "amount": "5"})
	before := claimIntegrityAllStores(t, app, ctx)
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
	data, err := json.Marshal(report)
	require.NoError(t, err)
	for _, reason := range []string{"invalid_retained_vote", "integer variance", "unknown halt reason", "conservative_forward_count=1", "unsupported qualification minimum", "legacy packet flow retained"} {
		require.Contains(t, string(data), reason)
	}
	require.Len(t, report.Digests, len(consensusRepairsCensusNames))
	for _, f := range report.Findings {
		require.Equal(t, "held", f.Outcome)
	}
	// The counter must exceed the highest stored proposal ID, not merely equal it.
	ctx.KVStore(app.keys["zerone_gov"]).Set([]byte{0x12}, binary.BigEndian.AppendUint64(nil, 4))
	_, err = app.CensusConsensusRepairsV1(ctx)
	require.ErrorContains(t, err, "seat counter")
}

func TestConsensusRepairsCensusRefusesSemanticCorruption(t *testing.T) {
	for _, scenario := range []string{"rejected challenge", "refundable deficit", "missing index", "negative pool", "overflow window", "malformed generation", "pending rollback"} {
		t.Run(scenario, func(t *testing.T) {
			app, ctx, _ := consensusRepairsFixture(t)
			ctx = ctx.WithBlockHeight(100)
			putProto := func(store string, key []byte, value proto.Message) {
				data, err := proto.Marshal(value)
				require.NoError(t, err)
				ctx.KVStore(app.keys[store]).Set(key, data)
			}
			putJSON := func(store string, key []byte, value interface{}) {
				data, err := json.Marshal(value)
				require.NoError(t, err)
				ctx.KVStore(app.keys[store]).Set(key, data)
			}
			switch scenario {
			case "rejected challenge":
				putProto("capture_challenge", []byte("\x01rejected"), &cc.CaptureChallenge{Id: "rejected", Status: cc.ChallengeStatus_CHALLENGE_STATUS_RESOLVED, Resolution: &cc.CaptureResolution{Outcome: cc.ChallengeOutcome_CHALLENGE_OUTCOME_REJECTED}})
			case "refundable deficit":
				putProto("capture_challenge", []byte("\x01pending"), &cc.CaptureChallenge{Id: "pending", Status: cc.ChallengeStatus_CHALLENGE_STATUS_UNDER_REVIEW, Stake: "1"})
			case "missing index":
				putProto("knowledge", knowledge.AugmentationKey("unindexed"), &knowledge.Augmentation{Id: "unindexed", BountyId: "pool"})
			case "negative pool":
				putProto("capture_challenge", []byte("\x02physics"), &cc.DomainBountyPool{Domain: "physics", Balance: "-1"})
			case "overflow window":
				putJSON("zerone_ibcrl", []byte("\x01channel-0\x00uzrn"), &ibcrl.RateLimit{ChannelId: "channel-0", Denom: "uzrn", MaxSend: "1", MaxRecv: "1", WindowBlocks: math.MaxUint64})
			case "malformed generation":
				ctx.KVStore(app.keys["zerone_ibcrl"]).Set([]byte("\x03channel-0\x00uzrn"), []byte{1})
			case "pending rollback":
				putJSON("zerone_gov", []byte("\x14rollback"), &gov.PhaseTransitionProposal{LipID: "rollback", IsRollback: true, Stage: "pending_activation"})
			}
			before := claimIntegrityAllStores(t, app, ctx)
			_, err := app.CensusConsensusRepairsV1(ctx)
			require.Error(t, err)
			require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
		})
	}
}
func TestConsensusRepairsCensusBoundsOnlySelectedPrefixes(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	cache, _ := ctx.CacheContext()
	store := cache.KVStore(app.keys["knowledge"])
	for i := 0; i <= consensusRepairsMaxRecords; i++ {
		// Unrelated history is hashed and measured, not retained or bounded.
		store.Set([]byte(fmt.Sprintf("unrelated/%d", i)), []byte{1})
	}
	large, err := app.CensusConsensusRepairsV1(cache)
	require.NoError(t, err)
	require.Equal(t, report.Records+consensusRepairsMaxRecords+1, large.Records)
	require.Equal(t, report.Censuses, large.Censuses)
	require.Equal(t, report.Digests, large.Digests)
	require.NotEqual(t, report.Digest, large.Digest)
	for i := 0; i <= consensusRepairsMaxRecords; i++ {
		store.Set([]byte(fmt.Sprintf("\x62bounded/%d", i)), []byte{1})
	}
	_, err = app.CensusConsensusRepairsV1(cache)
	require.ErrorContains(t, err, "reformulations census resource ceiling")
}

// Assert the public census method's context shape independently of callers.
var _ func(*ZeroneApp, sdk.Context) (ConsensusRepairsCensus, error) = (*ZeroneApp).CensusConsensusRepairsV1
