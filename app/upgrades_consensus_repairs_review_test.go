package app

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/require"
	alignment "github.com/zerone-chain/zerone/x/alignment/types"
	cd "github.com/zerone-chain/zerone/x/capture_defense/types"
	emergency "github.com/zerone-chain/zerone/x/emergency/types"
	gov "github.com/zerone-chain/zerone/x/gov/types"
	knowledge "github.com/zerone-chain/zerone/x/knowledge/types"
	ontology "github.com/zerone-chain/zerone/x/ontology/types"
	qualification "github.com/zerone-chain/zerone/x/qualification/types"
	staking "github.com/zerone-chain/zerone/x/staking/types"
	"google.golang.org/protobuf/proto"
)

func repairPutJSON(t *testing.T, app *ZeroneApp, ctx sdk.Context, store string, key []byte, value interface{}) {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	ctx.KVStore(app.keys[store]).Set(key, raw)
}

func repairPutProto(t *testing.T, app *ZeroneApp, ctx sdk.Context, store string, key []byte, value proto.Message) {
	t.Helper()
	raw, err := proto.Marshal(value)
	require.NoError(t, err)
	ctx.KVStore(app.keys[store]).Set(key, raw)
}

func TestConsensusRepairsRegistryLocationsAndSelections(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	markers, locations, selections := app.ConsensusRepairsV1Markers(), consensusRepairsMarkerLocations(), ConsensusRepairsV1GenesisSelections()
	require.Len(t, markers, 10)
	require.Len(t, locations, len(markers))
	require.Len(t, selections, len(markers))
	for i, marker := range markers {
		t.Run(marker.Module, func(t *testing.T) {
			require.Equal(t, ActivationSelection{marker.Module, marker.Name}, selections[i])
			cache, _ := ctx.CacheContext()
			before := claimIntegrityAllStores(t, app, cache)
			require.NoError(t, marker.Enable(cache))
			after := claimIntegrityAllStores(t, app, cache)
			key := locations[i].store + ":" + string(locations[i].key)
			require.NotContains(t, before, key)
			require.Equal(t, []byte{1}, after[key])
			delete(after, key)
			require.Equal(t, before, after, "Enable must write precisely locations[i]")
		})
	}
}

type repairGenesisFault struct {
	archivedConsensusSibling
	corrupt func(sdk.Context)
}

func (m repairGenesisFault) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, raw json.RawMessage) {
	m.HasGenesis.InitGenesis(ctx, cdc, raw)
	m.corrupt(ctx)
}

func TestConsensusRepairsInitChainReadsEveryMarkerBack(t *testing.T) {
	for i, selection := range ConsensusRepairsV1GenesisSelections() {
		for _, malformed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/corrupt=%t", selection.Module, malformed), func(t *testing.T) {
				app := NewZeroneApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()))
				genesis := sdk053IBC10GenesisWithValidator(t, app)
				require.NoError(t, ValidateConsensusRepairsGenesisSelection(genesis))
				original := app.ModuleManager.Modules[selection.Module]
				loc := consensusRepairsMarkerLocations()[i]
				app.ModuleManager.Modules[selection.Module] = repairGenesisFault{
					archivedConsensusSibling{original.(appmodule.AppModule), original.(module.HasGenesis), app.ModuleManager.GetVersionMap()[selection.Module]},
					func(ctx sdk.Context) {
						if malformed {
							ctx.KVStore(app.keys[loc.store]).Set(loc.key, []byte{2})
						} else {
							ctx.KVStore(app.keys[loc.store]).Delete(loc.key)
						}
					},
				}
				raw, err := json.Marshal(genesis)
				require.NoError(t, err)
				ctx := app.NewUncachedContext(false, cmtproto.Header{})
				cache, _ := ctx.CacheContext() // BaseApp discards failed InitChain state.
				_, err = app.InitChainer(cache, &abci.RequestInitChain{AppStateBytes: raw})
				require.ErrorContains(t, err, selection.Module+"."+selection.Name+" readback")
				require.Empty(t, app.KnowledgeKeeper.ReadMigrationMarker(cache, accountingAuthorityNativeMarker))
			})
		}
	}
}

type writingCensusBank struct {
	bankkeeper.Keeper
	mutate func(context.Context)
}

func (bank writingCensusBank) GetBalance(ctx context.Context, account sdk.AccAddress, denom string) sdk.Coin {
	bank.mutate(ctx)
	return bank.Keeper.GetBalance(ctx, account, denom)
}

func TestConsensusRepairsCensusDiscardsFutureHelperWrites(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	app.BankKeeper = writingCensusBank{app.BankKeeper, func(goCtx context.Context) {
		c := sdk.UnwrapSDKContext(goCtx)
		c.KVStore(app.keys["knowledge"]).Set([]byte("unwanted-census-write"), []byte{1})
		c.EventManager().EmitEvent(sdk.NewEvent("unwanted-census-event"))
	}}
	before := claimIntegrityAllStores(t, app, ctx)
	events := append(sdk.Events{}, ctx.EventManager().Events()...)
	_, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
	require.Equal(t, events, ctx.EventManager().Events())
	height := app.CommitMultiStore().Commit().Version + 1
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: height}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	ctx = ctx.WithBlockHeight(height).WithHeaderInfo(header.Info{Height: height})
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	require.False(t, ctx.KVStore(app.keys["knowledge"]).Has([]byte("unwanted-census-write")))
	for _, event := range ctx.EventManager().Events() {
		require.NotEqual(t, "unwanted-census-event", event.Type)
	}
}

func TestConsensusRepairsStreamingMergeMatchesExactImage(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	writes := map[string]map[string][]byte{
		"knowledge": {"\x00insert-before": {1}, "\xffinsert-after": {2}, "\x06": []byte("replacement")},
		"bank":      {"\x00uzrn": []byte("replacement")},
	}
	expected, claimBefore, err := app.consensusRepairsStream(ctx, writes, nil)
	require.NoError(t, err)
	cache, _ := ctx.CacheContext()
	for store, entries := range writes {
		for key, value := range entries {
			cache.KVStore(app.keys[store]).Set([]byte(key), value)
		}
	}
	actual, claimAfter, err := app.consensusRepairsStream(cache, nil, nil)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	require.Equal(t, claimBefore, claimAfter)
	for _, mutation := range []string{"insert", "delete", "replace", "claim-integrity"} {
		t.Run(mutation, func(t *testing.T) {
			changed, _ := cache.CacheContext()
			store := changed.KVStore(app.keys["knowledge"])
			switch mutation {
			case "insert":
				store.Set([]byte("unexpected"), []byte{1})
			case "delete":
				store.Delete([]byte("\x00insert-before"))
			case "replace":
				store.Set([]byte("\x00insert-before"), []byte{9})
			case "claim-integrity":
				store.Set([]byte{0x87}, []byte{2})
			}
			digest, claim, err := app.consensusRepairsStream(changed, nil, nil)
			require.NoError(t, err)
			require.NotEqual(t, expected, digest)
			if mutation == "claim-integrity" {
				require.NotEqual(t, claimBefore, claim)
			}
		})
	}
}

func TestConsensusRepairsZeroStakeDiscontinuityTruthTable(t *testing.T) {
	for _, registry := range []string{"empty", "inactive", "active-zero", "active-positive"} {
		for _, mint := range []int64{0, 100} {
			for _, supply := range []int64{0, 200} {
				t.Run(fmt.Sprintf("%s/mint=%d/supply=%d", registry, mint, supply), func(t *testing.T) {
					app, ctx, _ := consensusRepairsFixture(t)
					address := settlementAddress(240).String()
					key := append([]byte{1}, []byte(address)...)
					ctx.KVStore(app.keys["zerone_staking"]).Delete(key)
					stake := int64(0)
					if registry == "active-positive" {
						stake = 50
					}
					if registry != "empty" {
						repairPutJSON(t, app, ctx, "zerone_staking", key, &staking.Validator{OperatorAddress: address, IsActive: strings.HasPrefix(registry, "active-"), TotalStake: fmt.Sprint(stake)})
					}
					ctx.KVStore(app.keys["vesting_rewards"]).Set([]byte{0x0b}, []byte(fmt.Sprint(mint)))
					// Synthetic bank input, not a claim of balanced imported state.
					raw, err := sdk.IntValue.Encode(sdkmath.NewInt(supply))
					require.NoError(t, err)
					ctx.KVStore(app.keys["bank"]).Set([]byte("\x00uzrn"), raw)
					report, err := app.CensusConsensusRepairsV1(ctx)
					if stake == 0 && mint == 0 && supply > 0 {
						require.ErrorContains(t, err, "changes neutral to zero")
					} else {
						require.NoError(t, err)
					}
					raw, err = json.Marshal(report)
					require.NoError(t, err)
					require.Contains(t, string(raw), fmt.Sprintf("legacy_staking_ratio_bps=%d fixed_staking_ratio_bps=%d", consensusRepairsRatio(big.NewInt(stake), big.NewInt(mint)), consensusRepairsRatio(big.NewInt(stake), big.NewInt(supply))))
				})
			}
		}
	}
	require.Equal(t, alignment.NeutralBPS, consensusRepairsRatio(big.NewInt(0), big.NewInt(0)))
	require.Equal(t, uint64(0), consensusRepairsRatio(big.NewInt(0), big.NewInt(1)))
	require.Equal(t, alignment.BPS/4, consensusRepairsRatio(big.NewInt(1), big.NewInt(4)))
	require.Equal(t, alignment.BPS, consensusRepairsRatio(big.NewInt(9), big.NewInt(1)))
}

func TestConsensusRepairsDomainStatusesAreHeld(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	before, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	for i, status := range []string{"active", "archived", "arbitrary", "", "proposed", "deprecated", "archived"} {
		name := fmt.Sprintf("review-status-%d", i)
		repairPutProto(t, app, ctx, "zerone_ontology", []byte("\x02"+name), &ontology.Domain{Name: name, Status: status})
	}
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.Equal(t, before.DomainStatuses["active"]+1, report.DomainStatuses["active"])
	require.Equal(t, 2, report.DomainStatuses["archived"])
	for _, status := range []string{"archived", "arbitrary", "", "proposed", "deprecated"} {
		found := false
		for _, finding := range report.Findings {
			if strings.Contains(finding.Reason, fmt.Sprintf("inactive domain status=%q", status)) {
				require.Equal(t, "held", finding.Outcome)
				found = true
			}
		}
		require.True(t, found, status)
	}
}

func TestConsensusRepairsNULIdentitiesHeldWithoutMaskingFaults(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	repairPutProto(t, app, ctx, "knowledge", knowledge.AugmentationBountyKey("b\x00x"), &knowledge.AugmentationBounty{Id: "b\x00x"})
	for _, aug := range []*knowledge.Augmentation{{Id: "a", BountyId: "b\x00x"}, {Id: "x\x00a", BountyId: "b"}} {
		repairPutProto(t, app, ctx, "knowledge", knowledge.AugmentationKey(aug.Id), aug)
		ctx.KVStore(app.keys["knowledge"]).Set(knowledge.AugmentationByBountyKey(aug.BountyId, aug.Id), []byte{1})
	}
	before := claimIntegrityAllStores(t, app, ctx)
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
	raw, err := json.Marshal(report)
	require.NoError(t, err)
	require.Contains(t, string(raw), "bounties=1 augmentations_or_bounty_references=2")
	require.Contains(t, string(raw), "primary_records=2")
	for _, fault := range []string{"decode", "identity", "missing index", "foreign index"} {
		t.Run(fault, func(t *testing.T) {
			cache, _ := ctx.CacheContext()
			store := cache.KVStore(app.keys["knowledge"])
			switch fault {
			case "decode":
				store.Set(knowledge.AugmentationKey("a"), []byte{0xff})
			case "identity":
				repairPutProto(t, app, cache, "knowledge", knowledge.AugmentationKey("a"), &knowledge.Augmentation{Id: "other"})
			case "missing index":
				store.Delete(knowledge.AugmentationByBountyKey("b", "x\x00a"))
			case "foreign index":
				store.Set(knowledge.AugmentationByBountyKey("foreign", "x\x00a"), []byte{1})
			}
			_, err := app.CensusConsensusRepairsV1(cache)
			require.Error(t, err)
		})
	}
}

func TestConsensusRepairsExpiredHoldsOnlyPositiveAmounts(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	for _, amount := range []string{"", "0", "00", "42"} {
		key := "\x01validator/domain" + amount
		repairPutProto(t, app, ctx, "qualification", []byte(key), &qualification.DomainQualification{Validator: "validator", Domain: "domain" + amount, Status: qualification.QualificationStatus_QUALIFICATION_STATUS_EXPIRED, StakedAmount: amount})
	}
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	var holds []ConsensusRepairsFinding
	for _, finding := range report.Findings {
		if finding.Module == "qualification" && finding.Key != "" {
			holds = append(holds, finding)
		}
	}
	require.Len(t, holds, 1)
	require.Equal(t, hex.EncodeToString([]byte("\x01validator/domain42")), holds[0].Key)
	require.Equal(t, "held", holds[0].Outcome)
}

func TestConsensusRepairsHistoriesAndFlowsHaveBoundedReceiptSummaries(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	baseline, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	for i := 0; i < 100; i++ {
		round := fmt.Sprint(i)
		repairPutProto(t, app, ctx, "capture_defense", []byte("\x05physics/"+round), &cd.VerificationHistoryEntry{Domain: "physics", RoundId: round, SubmitBlocks: []uint64{1, 2}})
		key := append([]byte("\x02channel-0\x00"), binary.BigEndian.AppendUint64(nil, uint64(i))...)
		repairPutJSON(t, app, ctx, "zerone_ibcrl", key, map[string]interface{}{"channel_id": "channel-0", "sequence": i, "denom": "uzrn", "amount": "1"})
	}
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.Len(t, report.Findings, len(baseline.Findings), "no permanent per-history/per-flow finding")
	for _, name := range []string{"verification_histories", "ibc_refunds"} {
		require.Equal(t, baseline.Censuses[name].Records+100, report.Censuses[name].Records)
		require.NotEqual(t, baseline.Digests[name], report.Digests[name])
	}
}

func TestConsensusRepairsLargeUnselectedStateActivates(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	baseline, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	// Synthetic row-volume fixture, not real observations or staking history.
	// It exercises the carrier, not merely the standalone census, above the
	// removed global row ceiling in the stores implicated by the review.
	for i := 0; i < 100_001; i++ {
		key := binary.BigEndian.AppendUint64([]byte{byte(3 + i%3)}, uint64(i))
		ctx.KVStore(app.keys["alignment"]).Set(key, []byte("synthetic history"))
	}
	for i := 0; i < 10_000; i++ {
		ctx.KVStore(app.keys["staking"]).Set(binary.BigEndian.AppendUint64([]byte{0x50}, uint64(i)), []byte("synthetic historical info"))
	}
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.Greater(t, report.Records, consensusRepairsMaxRecords)
	require.Equal(t, baseline.Stores["alignment"].Records+100_001, report.Stores["alignment"].Records)
	require.Equal(t, baseline.Prefixes["staking"]["50"].Records+10_000, report.Prefixes["staking"]["50"].Records)
	require.Equal(t, baseline.Digests, report.Digests)
	require.Equal(t, baseline.Censuses, report.Censuses)
	for name, reads := range report.Reads {
		var sum ConsensusRepairsSize
		for _, read := range reads {
			sum.Records += read.Records
			sum.Bytes += read.Bytes
		}
		require.Equal(t, report.Censuses[name], sum, name)
	}
	height := app.CommitMultiStore().Commit().Version + 1
	plan := upgradetypes.Plan{Name: UpgradeNameConsensusRepairsV1, Height: height}
	require.NoError(t, app.UpgradeKeeper.ScheduleUpgrade(ctx, plan))
	ctx = ctx.WithBlockHeight(height).WithHeaderInfo(header.Info{Height: height})
	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, plan))
	for _, marker := range app.ConsensusRepairsV1Markers() {
		on, err := marker.Enabled(ctx)
		require.NoError(t, err)
		require.True(t, on)
	}
	t.Logf("synthetic carrier source: records=%d bytes=%d", report.Records, report.Bytes)
}

func TestConsensusRepairsBankReadsAreExactSDKKeys(t *testing.T) {
	app, ctx, _ := consensusRepairsFixture(t)
	require.NoError(t, app.BankKeeper.MintCoins(ctx, "tokens", sdk.NewCoins(sdk.NewInt64Coin("uzrn", 3))))
	for _, module := range []string{"qualification", "capture_challenge"} {
		require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, "tokens", module, sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
	}
	report, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	for _, name := range []string{"qualification_escrow", "challenge_backing", "alignment_inputs"} {
		bankRows := 0
		for _, read := range report.Reads[name] {
			if read.Store != "bank" {
				continue
			}
			key, err := hex.DecodeString(read.Prefix)
			require.NoError(t, err)
			require.True(t, read.Exact)
			require.True(t, ctx.KVStore(app.keys["bank"]).Has(key))
			require.Equal(t, 1, read.Records)
			require.Equal(t, len(key)+len(ctx.KVStore(app.keys["bank"]).Get(key)), read.Bytes)
			bankRows++
		}
		require.Equal(t, 1, bankRows, name)
	}
	// Transfer between unrelated accounts changes bank's full image, but none
	// of the two escrow balances or the supply key read by these censuses.
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(ctx, "tokens", "knowledge", sdk.NewCoins(sdk.NewInt64Coin("uzrn", 1))))
	after, err := app.CensusConsensusRepairsV1(ctx)
	require.NoError(t, err)
	require.NotEqual(t, report.Digest, after.Digest)
	require.Equal(t, report.Digests, after.Digests)
	require.Equal(t, report.Reads, after.Reads)
}

func TestConsensusRepairsRollbackMetadataAndLIPFaultsStillBlock(t *testing.T) {
	for _, fault := range []string{"decode", "identity", "target", "stage", "missing LIP", "malformed LIP", "LIP id", "category", "rollback flag"} {
		t.Run(fault, func(t *testing.T) {
			app, ctx, _ := consensusRepairsFixture(t)
			meta := &gov.PhaseTransitionProposal{LipID: "rollback", TargetPhase: gov.ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR, IsRollback: true, Stage: gov.PhaseTransitionStagePending}
			lip := &gov.LIP{Id: "rollback", Category: gov.CategoryPhaseRollback}
			switch fault {
			case "identity":
				meta.LipID = "other"
			case "target":
				meta.TargetPhase = 99
			case "stage":
				meta.Stage = "unknown"
			case "LIP id":
				lip.Id = "other"
			case "category":
				lip.Category = gov.CategoryText
			case "rollback flag":
				meta.IsRollback = false
			}
			repairPutJSON(t, app, ctx, "zerone_gov", []byte("\x14rollback"), meta)
			repairPutJSON(t, app, ctx, "zerone_gov", gov.LIPKey("rollback"), lip)
			switch fault {
			case "decode":
				ctx.KVStore(app.keys["zerone_gov"]).Set([]byte("\x14rollback"), []byte("invalid"))
			case "missing LIP":
				ctx.KVStore(app.keys["zerone_gov"]).Delete(gov.LIPKey("rollback"))
			case "malformed LIP":
				ctx.KVStore(app.keys["zerone_gov"]).Set(gov.LIPKey("rollback"), []byte("invalid"))
			}
			_, err := app.CensusConsensusRepairsV1(ctx)
			require.ErrorContains(t, err, "census blocker")
		})
	}
}

type repairHookSpy struct {
	appmodule.AppModule
	begin, end *int
}

func (m repairHookSpy) BeginBlock(context.Context) error {
	*m.begin++
	return fmt.Errorf("begin sentinel")
}
func (m repairHookSpy) EndBlock(context.Context) error { *m.end++; return fmt.Errorf("end sentinel") }

func TestConsensusRepairsArchivedFixturesPreserveBlockHooks(t *testing.T) {
	begin, end := 0, 0
	wrapper := archivedConsensusSibling{AppModule: repairHookSpy{begin: &begin, end: &end}, version: 1}
	require.ErrorContains(t, wrapper.BeginBlock(context.Background()), "begin sentinel")
	require.ErrorContains(t, wrapper.EndBlock(context.Background()), "end sentinel")
	require.Equal(t, 1, begin)
	require.Equal(t, 1, end)
}

func TestConsensusRepairsPendingRollbacksPredictActualCurrentState(t *testing.T) {
	for _, tc := range []struct {
		name        string
		halts       []string
		gridlock    bool
		staleTarget bool
		activation  uint64
		outcome     string
	}{
		{"unjustified", nil, false, false, 20, "cancel"},
		{"scoped", []string{"scoped"}, false, false, 20, "execute"},
		{"unknown", []string{"unknown"}, false, false, 20, "cancel"},
		{"unknown with scoped", []string{"unknown", "scoped"}, false, false, 20, "execute"},
		{"gridlock with unknown", []string{"unknown"}, true, false, 20, "execute"},
		{"old halt", []string{"old"}, false, false, 20, "cancel"},
		{"future halt", []string{"future"}, false, false, 20, "cancel"},
		{"stale target", []string{"scoped"}, false, true, 20, "cancel"},
		{"unscheduled", []string{"scoped"}, false, false, 0, "execute"},
		{"future activation", []string{"unknown"}, false, false, 21, "cancel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, ctx, _ := consensusRepairsFixture(t)
			ctx = ctx.WithBlockHeight(20)
			app.ZeroneGovKeeper.SetResearchFundGovernanceState(ctx, &gov.ResearchFundGovernanceState{CurrentPhase: gov.ResearchFundPhase_RESEARCH_FUND_PHASE_OBSERVER, PhaseStartedAtBlock: 10})
			for i, halt := range tc.halts {
				id := fmt.Sprint(i)
				start := uint64(10)
				reason := emergency.HaltReasonResearchFund
				switch halt {
				case "unknown":
					reason = "unknown"
				case "old":
					start = 9
				case "future":
					start = 21
				}
				payload, err := proto.Marshal(&emergency.EmergencyHaltProposal{Id: id, Reason: reason})
				require.NoError(t, err)
				repairPutProto(t, app, ctx, "emergency", emergency.CeremonyKey(id), &emergency.EmergencyCeremony{Id: id, Type: string(emergency.CeremonyHalt), Phase: string(emergency.PhaseFinalized), StartBlock: start, ProposalData: payload})
			}
			if tc.gridlock {
				for i := 1; i <= gov.RollbackGridlockThreshold; i++ {
					repairPutJSON(t, app, ctx, "zerone_gov", gov.ResearchSpendKey(uint64(i)), &gov.ResearchSpendProposal{ProposalId: uint64(i), Stage: string(gov.ResearchStageExpired)})
				}
			}
			meta := &gov.PhaseTransitionProposal{LipID: "rollback", TargetPhase: gov.ResearchFundPhase_RESEARCH_FUND_PHASE_GENESIS_PAIR, IsRollback: true, Stage: gov.PhaseTransitionStagePending, ActivationBlock: tc.activation}
			if tc.staleTarget {
				meta.TargetPhase = gov.ResearchFundPhase_RESEARCH_FUND_PHASE_BALANCED
			}
			repairPutJSON(t, app, ctx, "zerone_gov", []byte("\x14rollback"), meta)
			repairPutJSON(t, app, ctx, "zerone_gov", gov.LIPKey(meta.LipID), &gov.LIP{Id: meta.LipID, Category: gov.CategoryPhaseRollback})
			before := claimIntegrityAllStores(t, app, ctx)
			report, err := app.CensusConsensusRepairsV1(ctx)
			require.NoError(t, err)
			require.Equal(t, before, claimIntegrityAllStores(t, app, ctx))
			require.Len(t, report.PendingRollbacks, 1)
			prediction := report.PendingRollbacks[0]
			require.Equal(t, meta.LipID, prediction.LIPID)
			require.Equal(t, tc.activation, prediction.ActivationBlock)
			require.Equal(t, tc.outcome, prediction.Outcome)
			cache, _ := ctx.CacheContext()
			app.ZeroneGovKeeper.WithHaltReasonScoping(true).BeginBlockPhaseTransition(cache)
			actual, found := app.ZeroneGovKeeper.GetPhaseTransitionMeta(cache, meta.LipID)
			require.True(t, found)
			want := gov.PhaseTransitionStagePending
			if prediction.DueNow {
				want = gov.PhaseTransitionStageCancelled
				if tc.outcome == "execute" {
					want = gov.PhaseTransitionStageActivated
				}
			}
			require.Equal(t, want, actual.Stage)
		})
	}
}
