package keeper_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

type reformulationReadProbe struct {
	markerReads     int
	writes          int
	writeBeforeRead bool
	markerError     error
	malformed       bool
	indexError      error
	indexReads      []string
}

type reformulationProbeService struct {
	corestore.KVStoreService
	probe *reformulationReadProbe
}

func (s reformulationProbeService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return reformulationProbeStore{s.KVStoreService.OpenKVStore(ctx), s.probe}
}

type reformulationProbeStore struct {
	corestore.KVStore
	probe *reformulationReadProbe
}

func (s reformulationProbeStore) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.ReformulationVindicationOrderingEnabledKey) {
		s.probe.markerReads++
		if s.probe.markerError != nil {
			return nil, s.probe.markerError
		}
		if s.probe.malformed {
			return []byte{2}, nil
		}
	}
	return s.KVStore.Get(key)
}

func (s reformulationProbeStore) Set(key, value []byte) error {
	s.probe.writes++
	s.probe.writeBeforeRead = s.probe.writeBeforeRead || s.probe.markerReads == 0
	return s.KVStore.Set(key, value)
}

func (s reformulationProbeStore) Delete(key []byte) error {
	s.probe.writes++
	s.probe.writeBeforeRead = s.probe.writeBeforeRead || s.probe.markerReads == 0
	return s.KVStore.Delete(key)
}

func (s reformulationProbeStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	if len(start) > 0 && start[0] == 0x65 {
		s.probe.indexReads = append(s.probe.indexReads, string(start))
		if s.probe.indexError != nil {
			return nil, s.probe.indexError
		}
	}
	return s.KVStore.Iterator(start, end)
}

// Real SDK stores and bank transfers; only unrelated augmentation count varies.
// This protects all hot consumers, not just a mocked lookup. The synthetic
// staking adapter and issuance are the same as the pinned predecessor fixture.
func TestReformulationGasIndependentOfUnrelatedAugmentations(t *testing.T) {
	for _, operation := range []string{"vote", "payout", "veto", "expiry", "begin-block"} {
		t.Run(operation, func(t *testing.T) {
			var baseline uint64
			for _, n := range []int{0, 10, 100, 1000} {
				t.Run(fmt.Sprint(n), func(t *testing.T) {
					probe := &reformulationReadProbe{}
					k, ctx, bank, staking := newKnowledgeOrderFixtureWithStore(t, func(s corestore.KVStoreService) corestore.KVStoreService {
						return reformulationProbeService{s, probe}
					})
					params, err := k.GetParams(ctx)
					require.NoError(t, err)
					params.ReformulationMinPanelVotes = 1
					params.FitnessEpochBlocks = 0
					params.ProbeInvitationIdleThresholdBlocks = 0
					params.ProbeBountyMintPerBlock = "0"
					params.VindicationRefundEnabled = true
					params.VindicationWindowBlocks = 10
					require.NoError(t, k.SetParams(ctx, params))
					sponsor := sdk.AccAddress([]byte("indexed-sponsor-0001"))
					author := sdk.AccAddress([]byte("indexed-author-00001"))
					coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1500))
					require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
					require.NoError(t, bank.SendCoinsFromModuleToAccount(ctx, orderTestMint, sponsor, coins))
					require.NoError(t, k.EscrowAugmentationBounty(ctx, sponsor.String(), "target", sdkmath.NewInt(1500)))
					require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: "target", SponsorAddress: sponsor.String(), Active: true, MaxVariants: 1, RewardPerVariant: 1000, ExpiresAtBlock: 101}))
					aug := &types.Augmentation{Id: "target-variant", BountyId: "target", Submitter: author.String()}
					if operation == "veto" {
						aug.Verdict = 1
					}
					require.NoError(t, k.SetAugmentation(ctx, aug))
					// Invalid ballots elsewhere must neither increase gas nor block
					// this pool. SetAugmentation maintains the real 0x65 index.
					for i := 0; i < n; i++ {
						require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: fmt.Sprintf("unrelated-%04d", i), BountyId: "other", VerdictVotes: []types.AugmentationVerdict{7}}))
					}
					staking.addValidator("voter", 100_000, "bonded")
					require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
					*probe = reformulationReadProbe{}
					ctx = ctx.WithBlockHeight(1000).WithGasMeter(storetypes.NewGasMeter(10_000_000)).WithEventManager(sdk.NewEventManager())
					server := keeper.NewMsgServerImpl(k)
					switch operation {
					case "vote":
						response, callErr := server.VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: aug.Id, Verifier: "voter", Vote: 1})
						require.NoError(t, callErr)
						require.True(t, response.VerdictFinalized)
					case "payout":
						err = k.ReleaseAugmentationPayout(ctx, "target", author.String(), sdkmath.NewInt(1000))
					case "veto":
						_, err = server.SponsorVetoAugmentation(ctx, &types.MsgSponsorVetoAugmentation{AugmentationId: aug.Id, Sponsor: sponsor.String()})
					case "expiry":
						err = k.ProcessRouteBLifecycle(ctx)
					case "begin-block":
						err = k.BeginBlocker(ctx)
					}
					require.NoError(t, err)
					gas := ctx.GasMeter().GasConsumed()
					require.Equal(t, 1, probe.markerReads, "exactly one snapshot, including finalization and pruning")
					require.False(t, probe.writeBeforeRead, "snapshot precedes the first write")
					require.NotEmpty(t, probe.indexReads)
					for _, prefix := range probe.indexReads {
						require.Equal(t, string(types.AugmentationByBountyPrefix("target")), prefix)
					}
					if n == 0 {
						baseline = gas
					}
					t.Logf("AW118b_GAS operation=%s unrelated=%d gas=%d", operation, n, gas)
					require.Equal(t, baseline, gas, "gas must not grow with unrelated augmentations")
				})
			}
		})
	}
}

func TestReformulationMarkerFaultsPropagateBeforeWrites(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		for _, operation := range []string{"vote", "veto", "begin-block", "expiry", "prune"} {
			t.Run(fmt.Sprintf("malformed=%t/%s", malformed, operation), func(t *testing.T) {
				probe := &reformulationReadProbe{}
				k, ctx, _, _ := newKnowledgeOrderFixtureWithStore(t, func(s corestore.KVStoreService) corestore.KVStoreService { return reformulationProbeService{s, probe} })
				readErr := errors.New("injected marker read failure")
				*probe = reformulationReadProbe{malformed: malformed}
				if !malformed {
					probe.markerError = readErr
				}
				var err error
				switch operation {
				case "vote":
					_, err = keeper.NewMsgServerImpl(k).VoteOnAugmentation(ctx, nil)
				case "veto":
					_, err = keeper.NewMsgServerImpl(k).SponsorVetoAugmentation(ctx, nil)
				case "begin-block":
					err = k.BeginBlocker(ctx)
				case "expiry":
					err = k.ProcessRouteBLifecycle(ctx)
				case "prune":
					err = k.PruneExpiredVindications(ctx, 1000, 10)
				}
				require.ErrorIs(t, err, activation.ErrMarkerFault)
				if !malformed {
					require.ErrorIs(t, err, readErr)
				}
				require.Equal(t, 1, probe.markerReads)
				require.Zero(t, probe.writes)
				require.Empty(t, ctx.EventManager().Events())
			})
		}
	}
}

func TestReformulationHoldEventsOnlyOnTransitions(t *testing.T) {
	k, ctx, _, staking := newKnowledgeOrderFixture(t)
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	params.ReformulationMinPanelVotes = 2
	params.ReformulationConsensusBps = 500_000
	require.NoError(t, k.SetParams(ctx, params))
	require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
	require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: "new-tie", Submitter: "author", VerdictVoters: []string{"first"}, VerdictVotes: []types.AugmentationVerdict{1}, VerdictVoteStakes: []uint64{100_000}}))
	staking.addValidator("second", 100_000, "bonded")
	staking.addValidator("zero-weight", 0, "bonded")
	staking.addValidator("breaker", 200_000, "bonded")
	for _, tc := range []struct {
		voter     string
		vote      types.AugmentationVerdict
		wantEvent string
		final     bool
	}{{"second", 4, "zerone.knowledge.reformulation_held", false}, {"zero-weight", 1, "", false}, {"breaker", 1, "zerone.knowledge.reformulation_hold_released", true}} {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		response, err := keeper.NewMsgServerImpl(k).VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: "new-tie", Verifier: tc.voter, Vote: tc.vote})
		require.NoError(t, err)
		require.Equal(t, tc.final, response.VerdictFinalized)
		var transitions sdk.Events
		for _, event := range ctx.EventManager().Events() {
			if strings.HasPrefix(event.Type, "zerone.knowledge.reformulation_hold") || event.Type == "zerone.knowledge.reformulation_held" {
				transitions = append(transitions, event)
			}
		}
		if tc.wantEvent == "" {
			require.Empty(t, transitions, "continuing tie emits nothing")
		} else {
			require.Equal(t, sdk.Events{sdk.NewEvent(tc.wantEvent, sdk.NewAttribute("augmentation_id", "new-tie"), sdk.NewAttribute("original_fact_id", ""), sdk.NewAttribute("bounty_id", ""), sdk.NewAttribute("reason", "tied_top_weight"))}, transitions)
		}
	}
}

func TestReformulationIndexFailsClosed(t *testing.T) {
	for _, corruption := range []string{"missing-record", "bad-record", "wrong-bounty", "wrong-id", "bad-index"} {
		t.Run(corruption, func(t *testing.T) {
			f := newKnowledgeStoreFixture(t, "hold-index")
			require.NoError(t, f.k.EnableReformulationVindicationOrdering(f.ctx))
			aug := &types.Augmentation{Id: "a", BountyId: "pool"}
			require.NoError(t, f.k.SetAugmentation(f.ctx, aug))
			store := f.ctx.KVStore(f.key)
			switch corruption {
			case "missing-record":
				store.Delete(types.AugmentationKey("a"))
			case "bad-record":
				store.Set(types.AugmentationKey("a"), []byte{0xff})
			case "wrong-bounty":
				aug.BountyId = "another"
				require.NoError(t, f.k.SetAugmentation(f.ctx, aug))
			case "wrong-id":
				require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{Id: "b", BountyId: "pool"}))
				store.Set(types.AugmentationKey("a"), store.Get(types.AugmentationKey("b")))
			case "bad-index":
				store.Set(types.AugmentationByBountyKey("pool", "a"), []byte{2})
			}
			before := relationContractSnapshot(t, f.ctx, f.key)
			err := f.k.ReleaseAugmentationPayout(f.ctx, "pool", "not-an-address", sdkmath.NewInt(1))
			require.ErrorContains(t, err, "reformulation hold:")
			require.NotErrorIs(t, err, activation.ErrMarkerFault)
			require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key))
		})
	}
}

func TestReformulationCarrierCensusChecksIndexCompleteness(t *testing.T) {
	f := newKnowledgeStoreFixture(t, "hold-census-index")
	require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{Id: "retained", BountyId: "pool", VerdictVotes: []types.AugmentationVerdict{7}}))
	f.ctx.KVStore(f.key).Delete(types.AugmentationByBountyKey("pool", "retained"))
	_, err := f.k.CensusReformulationHolds(f.ctx)
	require.ErrorContains(t, err, "missing or malformed bounty index")
}

func TestReformulationExpiryIgnoresUnrelatedCorruption(t *testing.T) {
	f := newKnowledgeStoreFixture(t, "hold-expiry-index")
	require.NoError(t, f.k.EnableReformulationVindicationOrdering(f.ctx))
	require.NoError(t, f.k.SetAugmentationBounty(f.ctx, &types.AugmentationBounty{Id: "due", Active: true, ExpiresAtBlock: 1}))
	require.NoError(t, f.k.SetAugmentationBounty(f.ctx, &types.AugmentationBounty{Id: "later", Active: true, ExpiresAtBlock: 1000}))
	require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{Id: "bad", BountyId: "later"}))
	f.ctx.KVStore(f.key).Set(types.AugmentationKey("bad"), []byte{0xff})
	require.NoError(t, f.k.ProcessRouteBLifecycle(f.ctx.WithBlockHeight(10)))
	due, ok := f.k.GetAugmentationBounty(f.ctx, "due")
	require.True(t, ok)
	require.False(t, due.Active)
	later, ok := f.k.GetAugmentationBounty(f.ctx, "later")
	require.True(t, ok)
	require.True(t, later.Active)
	_, err := f.k.CensusReformulationHolds(f.ctx)
	require.Error(t, err, "only the explicit carrier census scans everything")
}

func TestReformulationOrdinaryIndexFailureDoesNotHalt(t *testing.T) {
	probe := &reformulationReadProbe{}
	k, ctx, _, _ := newKnowledgeOrderFixtureWithStore(t, func(s corestore.KVStoreService) corestore.KVStoreService { return reformulationProbeService{s, probe} })
	require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
	require.NoError(t, k.SetAugmentationBounty(ctx, &types.AugmentationBounty{Id: "due", Active: true, ExpiresAtBlock: 1}))
	*probe = reformulationReadProbe{indexError: errors.New("ordinary index failure")}
	require.NoError(t, k.BeginBlocker(ctx.WithBlockHeight(102)), "only marker faults introduce new halts")
	require.Equal(t, 1, probe.markerReads)
	bounty, ok := k.GetAugmentationBounty(ctx, "due")
	require.True(t, ok)
	require.True(t, bounty.Active)
	require.Equal(t, sdk.Events{sdk.NewEvent("zerone.knowledge.reformulation_processing_refused", sdk.NewAttribute("operation", "bounty_expiry"), sdk.NewAttribute("reason", "hold_index_unreadable"))}, ctx.EventManager().Events())
}
