package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/internal/activation"
	capturekeeper "github.com/zerone-chain/zerone/x/capture_defense/keeper"
	capturetypes "github.com/zerone-chain/zerone/x/capture_defense/types"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	stakingkeeper "github.com/zerone-chain/zerone/x/staking/keeper"
	stakingtypes "github.com/zerone-chain/zerone/x/staking/types"
)

// These are in-memory SDK-store tests. Capture defense uses its real production
// adapter and stores; staking is a call recorder, not a monetary transfer.
type q2Staking struct {
	types.StakingKeeper
	slashes []VerifierSlash
}

func (*q2Staking) GetEffectiveStake(context.Context, string) (uint64, error) { return 1, nil }
func (s *q2Staking) SlashValidator(_ context.Context, addr string, bps uint64) error {
	s.slashes = append(s.slashes, VerifierSlash{Verifier: addr, SlashBps: bps})
	return nil
}

type q2StoreProbe struct {
	reads, writes                int
	calibrationReads, floorReads int
	failOp                       string
	failPrefix                   []byte
}

type q2StoreService struct {
	corestore.KVStoreService
	probe *q2StoreProbe
}

func (s q2StoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return q2Store{s.KVStoreService.OpenKVStore(ctx), s.probe}
}

type q2Store struct {
	corestore.KVStore
	probe *q2StoreProbe
}

func (s q2Store) Get(key []byte) ([]byte, error) {
	if bytes.Equal(key, types.CalibrationSmoothingEnabledKey) {
		s.probe.calibrationReads++
	}
	if bytes.Equal(key, types.ZeroConfidenceFloorEnabledKey) {
		s.probe.floorReads++
	}
	if bytes.Equal(key, types.InconclusiveRoundAccountingEnabledKey) {
		s.probe.reads++
	}
	if s.probe.failOp == "get" && bytes.HasPrefix(key, s.probe.failPrefix) {
		return nil, errors.New("Q2 injected read failure")
	}
	return s.KVStore.Get(key)
}
func (s q2Store) Set(key, value []byte) error {
	s.probe.writes++
	if s.probe.failOp == "set" && bytes.HasPrefix(key, s.probe.failPrefix) {
		return errors.New("Q2 injected write failure")
	}
	return s.KVStore.Set(key, value)
}
func (s q2Store) Delete(key []byte) error {
	s.probe.writes++
	return s.KVStore.Delete(key)
}

type q2Fixture struct {
	k       Keeper
	capture capturekeeper.Keeper
	ctx     sdk.Context
	keys    []*storetypes.KVStoreKey
	staking *q2Staking
	probe   *q2StoreProbe
}

func newQ2Fixture(t *testing.T, enabled bool) *q2Fixture {
	t.Helper()
	db := dbm.NewMemDB()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.SetIAVLSyncPruning(true)
	keys := []*storetypes.KVStoreKey{storetypes.NewKVStoreKey(types.StoreKey), storetypes.NewKVStoreKey(capturetypes.StoreKey), storetypes.NewKVStoreKey(stakingtypes.StoreKey)}
	for _, key := range keys {
		ms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}
	require.NoError(t, ms.LoadLatestVersion())
	ctx := sdk.NewContext(ms, cmtproto.Header{ChainID: "q2-in-memory", Height: 100}, false, log.NewNopLogger())
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	probe, staking := new(q2StoreProbe), new(q2Staking)
	k := NewKeeper(q2StoreService{runtime.NewKVStoreService(keys[0]), probe}, cdc, "authority", nil, staking)
	gs := types.DefaultGenesis()
	gs.InconclusiveRoundAccountingEnabled = false
	gs.CalibrationSmoothingEnabled = false
	gs.ZeroConfidenceFloorEnabled = false
	gs.RecordIntegrityEnabled = false
	gs.ReviewNeutralityEnabled = false
	gs.ClaimRecordsEnabled = false
	gs.FundSettlementEnabled = false
	gs.ClaimIntegrityEnabled = false
	require.NoError(t, k.InitGenesis(ctx, gs))
	capture := capturekeeper.NewKeeper(runtime.NewKVStoreService(keys[1]), cdc, "authority")
	capture.InitGenesis(ctx, capturetypes.DefaultGenesis())
	k.SetCaptureDefenseKeeper(capturekeeper.NewKnowledgeCaptureDefenseAdapter(capture))
	if enabled {
		require.NoError(t, k.EnableInconclusiveRoundAccounting(ctx))
	}
	return &q2Fixture{k: k, capture: capture, ctx: ctx, keys: keys, staking: staking, probe: probe}
}

func (f *q2Fixture) seed(t *testing.T, id string, votes []string) *types.VerificationRound {
	t.Helper()
	claim := &types.Claim{Id: id, Submitter: "submitter", Domain: "physics", Category: "empirical", FactContent: "Q2 fixture " + id, Stake: "1000000", ClaimType: types.ClaimType_CLAIM_TYPE_ASSERTION}
	round := &types.VerificationRound{Id: "round-" + id, ClaimId: id, StartedAtBlock: 50, CommitDeadline: 70, RevealDeadline: 90, AggregationDeadline: 110, Phase: types.VerificationPhase_VERIFICATION_PHASE_REVEAL}
	claim.VerificationRoundId = round.Id
	require.NoError(t, f.k.SetClaim(f.ctx, claim))
	for i, vote := range votes {
		addr := fmt.Sprintf("verifier-%d", i)
		round.Commits = append(round.Commits, &types.CommitEntry{Verifier: addr})
		if vote != "" {
			round.Reveals = append(round.Reveals, &types.RevealEntry{Verifier: addr, Vote: vote})
		}
		f.capture.UpdateReputation(f.ctx, addr, claim.Domain, "", true)
	}
	require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
	return round
}

func (f *q2Fixture) resetObservation() {
	f.ctx = f.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
	f.probe.reads, f.probe.writes = 0, 0
	f.probe.calibrationReads, f.probe.floorReads = 0, 0
	f.staking.slashes = nil
}

func (f *q2Fixture) snapshot(t *testing.T) map[string]string {
	t.Helper()
	ctx := f.ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	out := map[string]string{}
	for _, key := range f.keys {
		it := ctx.KVStore(key).Iterator(nil, nil)
		for ; it.Valid(); it.Next() {
			out[key.Name()+":"+string(it.Key())] = string(it.Value())
		}
		require.NoError(t, it.Close())
	}
	return out
}

func runQ2Block(fn func() error) (err error, panicText string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			panicText = fmt.Sprint(recovered)
		}
	}()
	err = fn()
	return
}

func TestQ2MarkerOffMatchesLegacyStateEventsErrorsAndGas(t *testing.T) {
	for _, votes := range [][]string{
		nil, {"", "", "", ""}, {"accept", "", "", ""},
		{"accept", "accept", "accept", ""},
		{"accept", "accept", "reject", "reject", ""},
		{"accept", "accept", "accept", "reject"},
		{"accept", "accept", "accept", "accept"},
		{"reject", "reject", "reject", "reject", ""},
		{"malformed", "malformed", "malformed", "malformed", ""},
	} {
		for _, height := range []int64{80, 100, 120, 1000} {
			for _, fault := range []string{"", "get-params", "set-round", "unknown-policy"} {
				t.Run(fmt.Sprintf("votes=%v/height=%d/fault=%s", votes, height, fault), func(t *testing.T) {
					actual, legacy := newQ2Fixture(t, false), newQ2Fixture(t, false)
					for _, f := range []*q2Fixture{actual, legacy} {
						round := f.seed(t, "parity", votes)
						f.ctx = f.ctx.WithBlockHeight(height)
						switch fault {
						case "get-params":
							f.probe.failOp, f.probe.failPrefix = "get", types.ParamsKey
						case "set-round":
							f.probe.failOp, f.probe.failPrefix = "set", types.RoundKey(round.Id)
						case "unknown-policy":
							round.ReviewPolicyVersion = 99
							require.NoError(t, f.k.setVerificationRoundLegacy(f.ctx, round))
						}
						f.resetObservation()
					}
					wantErr, wantPanic := runQ2Block(func() error { return legacy.k.beginBlockInconclusiveAccountingLegacy(legacy.ctx) })
					gotErr, gotPanic := runQ2Block(func() error { return actual.k.BeginBlocker(actual.ctx) })
					require.Equal(t, wantPanic, gotPanic)
					if fault == "get-params" {
						require.NotEmpty(t, gotPanic, "preserve the pre-existing route-B nil-params panic")
					}
					require.Equal(t, fmt.Sprint(wantErr), fmt.Sprint(gotErr))
					require.Equal(t, legacy.ctx.GasMeter().GasConsumed(), actual.ctx.GasMeter().GasConsumed())
					require.Equal(t, legacy.ctx.EventManager().Events(), actual.ctx.EventManager().Events())
					require.Equal(t, legacy.snapshot(t), actual.snapshot(t))
					require.Equal(t, legacy.staking.slashes, actual.staking.slashes)
					require.Equal(t, 1, actual.probe.reads)
					require.Zero(t, legacy.probe.reads)
				})
			}
		}
	}
}

func TestQ2InconclusiveAccountingThroughBeginBlock(t *testing.T) {
	for _, tc := range []struct {
		name   string
		votes  []string
		height int64
	}{
		{"quorum-fail", []string{"accept", "reject", "", ""}, 100},
		{"no-reveals", []string{"", "", "", ""}, 100},
		{"split-at-quorum", []string{"accept", "accept", "reject", "reject", ""}, 100},
		{"effective-quorum-fail-late", []string{"accept", "accept", "accept", ""}, 120},
		{"expired-without-aggregation", []string{"accept", "", "", ""}, 120},
		{"three-of-four-unchanged", []string{"accept", "accept", "accept", "reject"}, 100},
	} {
		for _, enabled := range []bool{false, true} {
			for _, reward := range []string{"3000000", "0"} {
				t.Run(fmt.Sprintf("%s/enabled=%t/reward=%s", tc.name, enabled, reward), func(t *testing.T) {
					f := newQ2Fixture(t, enabled)
					round := f.seed(t, tc.name, tc.votes)
					params, err := f.k.GetParams(f.ctx)
					require.NoError(t, err)
					params.VerificationReward = reward
					require.NoError(t, f.k.SetParams(f.ctx, params))
					f.ctx = f.ctx.WithBlockHeight(tc.height)
					f.resetObservation()
					require.NoError(t, f.k.BeginBlocker(f.ctx))
					require.Equal(t, 1, f.probe.reads, "one marker read, regardless of commits/reveals")
					stored, found := f.k.GetVerificationRound(f.ctx, round.Id)
					require.True(t, found)
					require.Equal(t, types.Verdict_VERDICT_INCONCLUSIVE, stored.Verdict)
					if tc.name == "expired-without-aggregation" {
						require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_EXPIRED, stored.Phase)
					} else {
						require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, stored.Phase)
					}
					var wantSlashes []VerifierSlash
					if enabled {
						for i, vote := range tc.votes {
							if vote == "" {
								wantSlashes = append(wantSlashes, VerifierSlash{Verifier: fmt.Sprintf("verifier-%d", i), SlashBps: params.MissedRevealSlashBps})
							}
						}
					}
					require.Equal(t, wantSlashes, f.staking.slashes)
					for _, reveal := range round.Reveals {
						rep, found := f.capture.GetGlobalReputation(f.ctx, reveal.Verifier)
						require.True(t, found)
						want := uint64(1)
						if !enabled && stored.Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE {
							want = 2
						}
						require.Equal(t, want, rep.TotalVerifications)
						domain, found := f.capture.GetDomainReputation(f.ctx, "physics", reveal.Verifier)
						require.True(t, found)
						require.Equal(t, want, domain.Verifications)
					}
					_, history := f.capture.GetVerificationHistory(f.ctx, "physics", round.Id)
					require.Equal(t, stored.Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, history, "history is not suppressed")
					f.ctx = f.ctx.WithBlockHeight(tc.height + 1)
					require.NoError(t, f.k.BeginBlocker(f.ctx))
					require.Equal(t, wantSlashes, f.staking.slashes, "terminal rounds are not slashed again")
				})
			}
		}
	}
}

func TestQ2ExpiredPathPreservesKnowledgeAndCaptureEvents(t *testing.T) {
	off, on := newQ2Fixture(t, false), newQ2Fixture(t, true)
	for _, f := range []*q2Fixture{off, on} {
		f.seed(t, "expiry-events", []string{"accept", "", "", ""})
		f.ctx = f.ctx.WithBlockHeight(120)
		f.resetObservation()
		require.NoError(t, f.k.BeginBlocker(f.ctx))
	}
	// The recording staking fixture emits nothing. Its added slash calls
	// must not change the expiry, KARMA or capture-defense event stream.
	require.Equal(t, off.ctx.EventManager().Events(), on.ctx.EventManager().Events())
	want, got := off.snapshot(t), on.snapshot(t)
	delete(got, types.StoreKey+":"+string(types.InconclusiveRoundAccountingEnabledKey))
	require.Equal(t, want, got)
	require.Empty(t, off.staking.slashes)
	require.Len(t, on.staking.slashes, 3)
}

func TestQ2SelectionReadOnceForMultipleRoundsAndNoAdapterLeak(t *testing.T) {
	f := newQ2Fixture(t, true)
	f.seed(t, "a-inconclusive", []string{"accept", "", "", ""})
	decisive := f.seed(t, "b-decisive", []string{"reject", "reject", "reject", "reject", ""})
	f.resetObservation()
	require.NoError(t, f.k.BeginBlocker(f.ctx))
	require.Equal(t, 1, f.probe.reads)
	round, found := f.k.GetVerificationRound(f.ctx, decisive.Id)
	require.True(t, found)
	require.Equal(t, types.Verdict_VERDICT_REJECT, round.Verdict)
	for _, reveal := range decisive.Reveals {
		rep, found := f.capture.GetGlobalReputation(f.ctx, reveal.Verifier)
		require.True(t, found)
		require.EqualValues(t, 3, rep.TotalVerifications, "two fixture seeds plus decisive completion")
	}
}

func TestQ2MarkerFaultBeforeAnyWrite(t *testing.T) {
	for _, value := range [][]byte{{0}, {2}, {1, 0}, nil} {
		t.Run(fmt.Sprintf("value=%v", value), func(t *testing.T) {
			f := newQ2Fixture(t, false)
			f.seed(t, "fault", []string{"accept", "", "", ""})
			if value == nil {
				f.probe.failOp, f.probe.failPrefix = "get", types.InconclusiveRoundAccountingEnabledKey
			} else {
				f.ctx.KVStore(f.keys[0]).Set(types.InconclusiveRoundAccountingEnabledKey, value)
			}
			before := f.snapshot(t)
			f.resetObservation()
			err := f.k.BeginBlocker(f.ctx)
			require.ErrorIs(t, err, activation.ErrMarkerFault)
			require.Equal(t, 1, f.probe.reads)
			require.Zero(t, f.probe.writes)
			require.Zero(t, f.ctx.GasMeter().GasConsumed())
			require.Empty(t, f.ctx.EventManager().Events())
			require.Empty(t, f.staking.slashes)
			require.Equal(t, before, f.snapshot(t))
		})
	}
}

// Digests pin exact function bodies, including comments, to foundation
// 6a56b621082064483af1ced7718462766d9ae756 rather than to the candidate.
func TestQ2LegacyBodiesPinnedToFoundation(t *testing.T) {
	for _, tc := range []struct{ file, name, digest string }{
		{"inconclusive_round_accounting_legacy.go", "beginBlockInconclusiveAccountingLegacy", "281c78cecbe99c4fa20fabf93ea3e9d4609b2d399bde542163f7522a47d6071c"},
		{"phases.go", "AdvanceRoundPhases", "68cace829387694a641bfe6f6dc1aa4f70aa847889d403d5ded71f93a69bc1e9"},
		{"phases.go", "performAggregation", "5254b7d947a73010cc324b5d7ad1aecbf18067f20b6b6ea66a1648985f3e7ed3"},
		{"confidence.go", "AggregateVerificationResult", "3f155b1f49bb266079812d021e7f122978503a062735fe1d868475021db82a55"},
		{"confidence.go", "calculateRewardsAndSlashes", "a8d19b86a0064e7754001ca55f2a999f1e8bec13c4ce37b6d881e534c54a3e0e"},
		{"rounds.go", "CompleteRound", "f62aa9e20e0093531e63b8cbaccdcafcfc856e7a48e2119da6307e7cc427e225"},
		{"rounds.go", "completeRound", "606e12c1974949e20afce50a01ece7963a208aac4c679e35cc35ed6c518bfc93"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, err := os.ReadFile(tc.file)
			require.NoError(t, err)
			set := token.NewFileSet()
			file, err := parser.ParseFile(set, tc.file, source, 0)
			require.NoError(t, err)
			found := false
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != tc.name {
					continue
				}
				found = true
				body := source[set.Position(fn.Body.Pos()).Offset:set.Position(fn.Body.End()).Offset]
				require.Equal(t, tc.digest, fmt.Sprintf("%x", sha256.Sum256(body)))
			}
			require.True(t, found)
		})
	}
}

func TestQ2DecisiveAggregationAndPolicyAreUnchanged(t *testing.T) {
	for _, vote := range []string{"accept", "reject", "malformed"} {
		for _, reward := range []string{"3000000", "0"} {
			t.Run(vote+"/reward="+reward, func(t *testing.T) {
				f := newQ2Fixture(t, true)
				round := f.seed(t, "decisive", []string{vote, vote, vote, vote, ""})
				params, err := f.k.GetParams(f.ctx)
				require.NoError(t, err)
				params.VerificationReward = reward
				require.NoError(t, f.k.SetParams(f.ctx, params))
				want, err := f.k.AggregateVerificationResult(f.ctx, round)
				require.NoError(t, err)
				got, err := f.k.aggregateInconclusiveAccounting(f.ctx, round, true)
				require.NoError(t, err)
				require.Equal(t, want, got)
				require.NotEqual(t, types.Verdict_VERDICT_INCONCLUSIVE, got.Verdict)
			})
		}
	}
}

type q2SlashBank struct {
	stakingtypes.BankKeeper
	transfers []sdk.Coins
}

func (b *q2SlashBank) SendCoinsFromModuleToModule(_ context.Context, _, _ string, coins sdk.Coins) error {
	b.transfers = append(b.transfers, coins)
	return nil
}

func TestQ2RealStakingAdapterPreservesCustodyBoundary(t *testing.T) {
	for _, sealed := range []bool{false, true} {
		t.Run(fmt.Sprintf("accounting-sealed=%t", sealed), func(t *testing.T) {
			f := newQ2Fixture(t, true)
			round := f.seed(t, "real-staking", []string{""})
			addr := sdk.AccAddress(bytes.Repeat([]byte{42}, 20)).String()
			round.Commits[0].Verifier = addr
			require.NoError(t, f.k.SetVerificationRound(f.ctx, round))
			bank := new(q2SlashBank)
			staking := stakingkeeper.NewKeeper(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), f.keys[2], nil, bank, "authority")
			staking.SetParams(f.ctx, stakingtypes.DefaultParams())
			staking.SetValidator(f.ctx, &stakingtypes.Validator{OperatorAddress: addr, TotalStake: "1000000", SelfDelegation: "1000000", DelegatedStake: "0"})
			if sealed {
				// Model the selected custody boundary without a bank genesis or
				// an activation census. This is an adapter test, not a carrier test.
				f.ctx.KVStore(f.keys[2]).Set(stakingtypes.AccountingSafetyKey, []byte{1})
			}
			f.k.stakingKeeper = stakingkeeper.NewStakingKeeperAdapter(staking)
			f.resetObservation()
			require.NoError(t, f.k.BeginBlocker(f.ctx))
			val, found := staking.GetValidator(f.ctx, addr)
			require.True(t, found)
			if sealed {
				require.Equal(t, "1000000", val.TotalStake)
				require.Empty(t, bank.transfers)
			} else {
				require.Equal(t, "900000", val.TotalStake)
				require.Equal(t, []sdk.Coins{sdk.NewCoins(sdk.NewInt64Coin("uzrn", 100000))}, bank.transfers)
			}
			wantEvent := "zerone.staking.validator_slashed"
			if sealed {
				wantEvent = "zerone.staking.monetary_slash_not_applied"
			}
			foundEvent := false
			for _, event := range f.ctx.EventManager().Events() {
				foundEvent = foundEvent || event.Type == wantEvent
			}
			require.True(t, foundEvent)
		})
	}
}

func TestQ2NeutralPolicyRetainsNoMonetaryDebt(t *testing.T) {
	for _, votes := range [][]string{{"accept", "reject", ""}, {"", "", ""}, {"accept", "reject", "malformed", ""}} {
		t.Run(fmt.Sprint(votes), func(t *testing.T) {
			f, _, round := neutralReviewFixture(t, votes)
			require.NoError(t, f.k.EnableInconclusiveRoundAccounting(f.ctx))
			result, err := f.k.aggregateInconclusiveAccounting(f.ctx, round, true)
			require.NoError(t, err)
			require.Empty(t, result.Slashes)
			require.Len(t, result.Rewards, len(round.Reveals))
			// The fixture's forbidden staking and capture adapters panic if used.
			require.NoError(t, f.k.BeginBlocker(f.ctx))
			stored, found := f.k.GetVerificationRound(f.ctx, round.Id)
			require.True(t, found)
			require.Equal(t, types.Verdict_VERDICT_INCONCLUSIVE, stored.Verdict)
			require.Equal(t, types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, stored.Phase)
		})
	}
}
