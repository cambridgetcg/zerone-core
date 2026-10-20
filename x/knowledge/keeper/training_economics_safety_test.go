package keeper_test

import (
	"errors"
	"fmt"
	"math"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func TestReformulationActivationCache(t *testing.T) {
	f := newKnowledgeStoreFixture(t, "economic-determinism-test")
	key := []byte(keeper.ReformulationVindicationOrderingEnabledStoreKey)
	require.Equal(t, []byte{0x90}, key)
	on, err := f.k.ReformulationVindicationOrderingEnabled(f.ctx)
	require.NoError(t, err)
	require.False(t, on)
	before := relationContractSnapshot(t, f.ctx, f.key)
	cached, _ := f.ctx.CacheContext()
	require.NoError(t, f.k.EnableReformulationVindicationOrdering(cached))
	require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key), "discarded activation cache changes nothing")
	cached, write := f.ctx.CacheContext()
	require.NoError(t, f.k.EnableReformulationVindicationOrdering(cached))
	write()
	on, err = f.k.ReformulationVindicationOrderingEnabled(f.ctx)
	require.NoError(t, err)
	require.True(t, on)
	after := relationContractSnapshot(t, f.ctx, f.key)
	require.Equal(t, []byte{1}, after[string(key)])
	delete(after, string(key))
	require.Equal(t, before, after, "activation changes only its selector")
	before = relationContractSnapshot(t, f.ctx, f.key)
	require.NoError(t, f.k.EnableReformulationVindicationOrdering(f.ctx))
	require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key), "idempotent, not an upgrade receipt")

	for _, bad := range [][]byte{{0}, {2}, {1, 1}} {
		t.Run(fmt.Sprintf("malformed-%x", bad), func(t *testing.T) {
			f.ctx.KVStore(f.key).Set(key, bad)
			before := relationContractSnapshot(t, f.ctx, f.key)
			on, err := f.k.ReformulationVindicationOrderingEnabled(f.ctx)
			require.ErrorContains(t, err, "malformed reformulation and vindication ordering marker")
			require.False(t, on)
			require.Error(t, f.k.EnableReformulationVindicationOrdering(f.ctx))
			require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key), "never repair a malformed marker implicitly")
		})
	}
}

func TestReformulationActivationMarkerErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		store markerErrorStore
	}{
		{"read", markerErrorStore{getErr: errors.New("marker read failed")}},
		{"write", markerErrorStore{setErr: errors.New("marker write failed")}},
		{"empty-present", markerErrorStore{getValue: []byte{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			k := keeper.NewKeeper(markerStoreService{store: tc.store}, nil, "", nil, nil)
			err := k.EnableReformulationVindicationOrdering(sdk.Context{})
			require.Error(t, err)
			if tc.store.getErr != nil {
				require.ErrorIs(t, err, tc.store.getErr)
			} else if tc.store.setErr != nil {
				require.ErrorIs(t, err, tc.store.setErr)
			} else {
				require.ErrorContains(t, err, "malformed")
			}
		})
	}
}

func TestReformulationActivationMalformedMarkerStopsConsumers(t *testing.T) {
	f := newKnowledgeStoreFixture(t, "economic-determinism-test")
	require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{Id: "pending", Submitter: "author"}))
	require.NoError(t, f.k.SetVindicationPending(f.ctx, "expired", []types.VindicationEntry{{FactId: "expired", Height: 1, SlashAmount: "3"}}))
	f.ctx.KVStore(f.key).Set([]byte(keeper.ReformulationVindicationOrderingEnabledStoreKey), []byte{2})
	before := relationContractSnapshot(t, f.ctx, f.key)
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	response, err := f.server.VoteOnAugmentation(f.ctx, &types.MsgVoteOnAugmentation{AugmentationId: "pending", Verifier: "voter", Vote: types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT})
	require.ErrorContains(t, err, "malformed reformulation and vindication ordering marker")
	require.Nil(t, response)
	f.k.PruneExpiredVindications(f.ctx, 100, 10)
	require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key))
	require.Equal(t, sdk.Events{sdk.NewEvent("zerone.knowledge.reformulation_processing_refused",
		sdk.NewAttribute("operation", "vindication_pruning"), sdk.NewAttribute("reason", "activation_marker_unreadable"))}, f.ctx.EventManager().Events())
}

func TestAugmentationVoteRange(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, throughMessage := range []bool{false, true} {
			for _, vote := range []int32{math.MinInt32, -1, 0, 1, 2, 3, 4, 5, 7, math.MaxInt32} {
				t.Run(fmt.Sprintf("enabled=%t/message=%t/vote=%d", enabled, throughMessage, vote), func(t *testing.T) {
					k, ctx, bank, staking := newKnowledgeOrderFixture(t)
					if enabled {
						require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
					}
					params, err := k.GetParams(ctx)
					require.NoError(t, err)
					params.ReformulationMinPanelVotes = 1
					require.NoError(t, k.SetParams(ctx, params))
					staking.addValidator("voter", 100_000, "bonded")
					original := &types.Augmentation{Id: "variant", Submitter: "author"}
					require.NoError(t, k.SetAugmentation(ctx, original))
					ctx = ctx.WithEventManager(sdk.NewEventManager())
					var finalized bool
					var verdict types.AugmentationVerdict
					if throughMessage {
						response, callErr := keeper.NewMsgServerImpl(k).VoteOnAugmentation(ctx, &types.MsgVoteOnAugmentation{AugmentationId: "variant", Verifier: "voter", Vote: types.AugmentationVerdict(vote)})
						err = callErr
						if response != nil {
							finalized, verdict = response.VerdictFinalized, response.FinalizedVerdict
						}
					} else {
						finalized, verdict, err = k.RecordAugmentationVote(ctx, "variant", "voter", types.AugmentationVerdict(vote))
					}
					got, ok := k.GetAugmentation(ctx, "variant")
					require.True(t, ok)
					if vote == 0 || (enabled && (vote < 1 || vote > 4)) {
						require.Error(t, err)
						require.False(t, finalized)
						require.True(t, proto.Equal(original, got), "refusal must not append a ballot or finalize")
						require.Empty(t, ctx.EventManager().Events())
					} else {
						require.NoError(t, err)
						require.True(t, finalized)
						require.Equal(t, types.AugmentationVerdict(vote), verdict)
						require.Equal(t, []types.AugmentationVerdict{types.AugmentationVerdict(vote)}, got.VerdictVotes)
					}
					require.True(t, bank.GetSupply(ctx, "uzrn").Amount.IsZero(), "no issuance in this synthetic voting case")
				})
			}
		}
	}
}

func TestReformulationActivationLegacyAndActivation(t *testing.T) {
	equivalent := types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT
	drift := types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT
	for _, enabled := range []bool{false, true} {
		for _, large := range []bool{false, true} {
			t.Run(fmt.Sprintf("enabled=%t/large=%t", enabled, large), func(t *testing.T) {
				k, ctx, _, staking := newKnowledgeOrderFixture(t)
				params, err := k.GetParams(ctx)
				require.NoError(t, err)
				params.ReformulationMinPanelVotes = 3
				params.ReformulationConsensusBps = 500_000
				require.NoError(t, k.SetParams(ctx, params))
				stakes := []uint64{100_000, 100_000}
				if large {
					stakes = []uint64{200_000_000_000_000, 100_000_000_000_000}
				}
				require.NoError(t, k.SetAugmentation(ctx, &types.Augmentation{Id: "variant", Submitter: "author"}))
				for i, vote := range []types.AugmentationVerdict{equivalent, drift} {
					voter := fmt.Sprintf("voter-%d", i)
					staking.addValidator(voter, stakes[i], "bonded")
					finalized, _, err := k.RecordAugmentationVote(ctx, "variant", voter, vote)
					require.NoError(t, err)
					require.False(t, finalized)
				}
				before, _ := k.GetAugmentation(ctx, "variant")
				if enabled {
					require.NoError(t, k.EnableReformulationVindicationOrdering(ctx))
				}
				after, _ := k.GetAugmentation(ctx, "variant")
				require.True(t, proto.Equal(before, after), "activation retains all old ballot snapshots")
				staking.addValidator("zero-weight", 0, "bonded")
				finalized, verdict, err := k.RecordAugmentationVote(ctx, "variant", "zero-weight", equivalent)
				require.NoError(t, err)
				if large {
					require.Equal(t, enabled, finalized, "old overflowing threshold waits; enabled exact arithmetic qualifies")
					if enabled {
						require.Equal(t, equivalent, verdict)
					}
				} else {
					require.Equal(t, !enabled, finalized, "old tie selects some verdict; enabled tie waits")
					if finalized {
						require.Contains(t, []types.AugmentationVerdict{equivalent, drift}, verdict)
					} else {
						require.Equal(t, types.AugmentationVerdict_AUGMENTATION_VERDICT_PENDING, verdict)
						// New weight may break a retained tie; this is not a permanent veto.
						staking.addValidator("tie-breaker", 100_000, "bonded")
						finalized, verdict, err = k.RecordAugmentationVote(ctx, "variant", "tie-breaker", drift)
						require.NoError(t, err)
						require.True(t, finalized)
						require.Equal(t, drift, verdict)
					}
				}
			})
		}
	}
}

func TestReformulationActivationPreservesInvalidHistory(t *testing.T) {
	for _, finalized := range []bool{false, true} {
		for _, invalid := range []types.AugmentationVerdict{-1, 0, 7} {
			t.Run(fmt.Sprintf("finalized=%t/vote=%d", finalized, invalid), func(t *testing.T) {
				f := newKnowledgeStoreFixture(t, "economic-determinism-test")
				aug := &types.Augmentation{Id: "legacy", Submitter: "author", VerdictVoters: []string{"old-voter"}, VerdictVotes: []types.AugmentationVerdict{invalid}, VerdictVoteStakes: []uint64{100_000}, VerdictVoteCalibrationBps: []uint64{1_000_000}}
				if finalized {
					aug.Verdict = types.AugmentationVerdict_AUGMENTATION_VERDICT_SUPERIOR
					aug.Accepted = true
					aug.PayoutAmount = "1500"
				}
				require.NoError(t, f.k.SetAugmentation(f.ctx, aug))
				require.NoError(t, f.k.EnableReformulationVindicationOrdering(f.ctx))
				before := relationContractSnapshot(t, f.ctx, f.key)
				response, err := f.server.VoteOnAugmentation(f.ctx, &types.MsgVoteOnAugmentation{AugmentationId: "legacy", Verifier: "new-voter", Vote: types.AugmentationVerdict_AUGMENTATION_VERDICT_EQUIVALENT})
				if finalized {
					require.ErrorContains(t, err, "already final")
				} else {
					require.ErrorContains(t, err, "invalid retained vote")
				}
				require.Nil(t, response)
				require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key))
				require.Empty(t, f.ctx.EventManager().Events())
			})
		}
	}
}

func TestReformulationActivationVindicationLegacy(t *testing.T) {
	k, ctx, bank, _ := newKnowledgeOrderFixture(t)
	on, err := k.ReformulationVindicationOrderingEnabled(ctx)
	require.NoError(t, err)
	require.False(t, on)
	coins := sdk.NewCoins(sdk.NewInt64Coin("uzrn", 5))
	require.NoError(t, bank.MintCoins(ctx, orderTestMint, coins))
	require.NoError(t, bank.SendCoinsFromModuleToModule(ctx, orderTestMint, types.VindicationEscrowModuleName, coins))
	for _, id := range []string{"fact-a", "fact-b"} {
		amount := "3"
		if id == "fact-b" {
			amount = "4"
		}
		require.NoError(t, k.SetVindicationPending(ctx, id, []types.VindicationEntry{{FactId: id, Height: 1, SlashAmount: amount}}))
	}
	ctx = ctx.WithEventManager(sdk.NewEventManager())
	k.PruneExpiredVindications(ctx, 100, 10)
	var ids []string
	for _, event := range ctx.EventManager().Events() {
		if event.Type == "zerone.knowledge.vindication_expired" {
			ids = append(ids, event.Attributes[0].Value)
		}
	}
	require.ElementsMatch(t, []string{"fact-a", "fact-b"}, ids, "legacy makes no key-order promise")
	wantPaid := int64(3)
	if ids[0] == "fact-b" {
		wantPaid = 4
	}
	require.Equal(t, wantPaid, bank.GetBalance(ctx, authtypes.NewModuleAddress("development_fund"), "uzrn").Amount.Int64())
	require.Equal(t, 5-wantPaid, bank.GetBalance(ctx, authtypes.NewModuleAddress(types.VindicationEscrowModuleName), "uzrn").Amount.Int64())
	require.EqualValues(t, 5, bank.GetSupply(ctx, "uzrn").Amount.Int64())
	require.Empty(t, k.GetAllVindicationPending(ctx))
}
