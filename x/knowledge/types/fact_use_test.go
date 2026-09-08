package types_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"testing"

	"cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Public deterministic address bytes only; no key generation or signing.
func factUseConsumer(n byte) string {
	return sdk.AccAddress(bytes.Repeat([]byte{n}, 20)).String()
}

func factUseParams() *types.Params {
	p := types.DefaultParams()
	p.FactUseEnabled = true
	p.FactUseConsumers = []string{factUseConsumer(1)}
	p.FitnessWeightQueryBps = 0
	p.FitnessWeightSatisfactionBps = 0
	p.MetabolismEnergyPerQuery = 0
	return &p
}

func factUseReceipt() *types.FactUseReceipt {
	return &types.FactUseReceipt{
		Version: types.FactUseReceiptVersion, Epoch: 0,
		Consumer: factUseConsumer(1), FactId: "fact-1", UseHeight: 9,
		ExpiryHeight: 20, Rating: types.FactUseRating_FACT_USE_RATING_UNRATED,
	}
}

func TestFactUseDefaultsAndPolicy(t *testing.T) {
	p := types.DefaultParams()
	require.False(t, p.FactUseEnabled)
	require.Empty(t, p.FactUseConsumers)
	require.Equal(t, uint64(100), p.FactUseMaxPerConsumerEpoch)
	require.Equal(t, uint64(1000), p.FactUseMaxPerEpoch)
	require.NoError(t, p.Validate())
	require.NoError(t, factUseParams().Validate())

	for name, mutate := range map[string]func(*types.Params){
		"query weight":        func(p *types.Params) { p.FitnessWeightQueryBps = 1 },
		"satisfaction weight": func(p *types.Params) { p.FitnessWeightSatisfactionBps = 1 },
		"query energy":        func(p *types.Params) { p.MetabolismEnergyPerQuery = 1 },
		"zero account cap":    func(p *types.Params) { p.FactUseMaxPerConsumerEpoch = 0 },
		"account ceiling":     func(p *types.Params) { p.FactUseMaxPerConsumerEpoch = 101 },
		"zero global cap":     func(p *types.Params) { p.FactUseMaxPerEpoch = 0 },
		"global ceiling":      func(p *types.Params) { p.FactUseMaxPerEpoch = 1001 },
		"unbounded cap":       func(p *types.Params) { p.FactUseMaxPerEpoch = math.MaxUint64 },
		"inverted caps":       func(p *types.Params) { p.FactUseMaxPerEpoch = 99 },
		"zero epoch":          func(p *types.Params) { p.FitnessEpochBlocks = 0 },
		"overflow epoch":      func(p *types.Params) { p.FitnessEpochBlocks = math.MaxUint64 },
		"noncanonical":        func(p *types.Params) { p.FactUseConsumers = []string{strings.ToUpper(factUseConsumer(1))} },
		"duplicate":           func(p *types.Params) { p.FactUseConsumers = []string{factUseConsumer(1), factUseConsumer(1)} },
		"invalid address":     func(p *types.Params) { p.FactUseConsumers = []string{"zrn1notanaddress"} },
		"oversize address":    func(p *types.Params) { p.FactUseConsumers = []string{strings.Repeat("x", 129)} },
		"cohort ceiling": func(p *types.Params) {
			p.FactUseConsumers = nil
			for i := byte(1); i <= 33; i++ {
				p.FactUseConsumers = append(p.FactUseConsumers, factUseConsumer(i))
			}
		},
	} {
		t.Run(name, func(t *testing.T) { p := factUseParams(); mutate(p); require.Error(t, p.Validate()) })
	}
	p2 := factUseParams()
	p2.FactUseConsumers = nil // enabled but empty admits nobody, never everyone
	require.NoError(t, p2.Validate())
	for i := byte(1); i <= 32; i++ {
		p2.FactUseConsumers = append(p2.FactUseConsumers, factUseConsumer(i))
	}
	require.NoError(t, p2.Validate())
	p2.FactUseEnabled = false
	p2.FactUseMaxPerEpoch = 0
	require.Error(t, p2.Validate(), "disabled is not permission for zero-as-unlimited limits")
}

func TestFactUseParamChangeFreezesEpoch(t *testing.T) {
	pruning := &types.FactUsePruningState{EverReported: true}
	for _, enabled := range []bool{false, true} {
		for _, retained := range []bool{false, true} {
			current := factUseParams()
			current.FactUseEnabled = enabled
			proposed := proto.Clone(current).(*types.Params)
			proposed.FactUseEnabled = false
			proposed.FitnessEpochBlocks++
			err := types.ValidateFactUseParamChange(current, proposed, pruning, retained)
			if enabled || retained {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}
	}
	current := factUseParams()
	current.FactUseEnabled = false
	proposed := proto.Clone(current).(*types.Params)
	proposed.FactUseEnabled = true
	proposed.FitnessEpochBlocks++
	require.Error(t, types.ValidateFactUseParamChange(current, proposed, pruning, false))
	proposed.FitnessEpochBlocks = current.FitnessEpochBlocks
	require.NoError(t, types.ValidateFactUseParamChange(current, proposed, pruning, true))
	require.Error(t, types.ValidateFactUseParamChange(nil, proposed, pruning, false))
}

func TestFactUseNonEconomicStateLifecycle(t *testing.T) {
	legacy := types.DefaultParams()
	pruning := &types.FactUsePruningState{} // explicit new migration state, not a reset helper
	enabled := factUseParams()
	require.NoError(t, types.ValidateFactUseParamChange(&legacy, enabled, pruning, false))
	require.False(t, pruning.EverReported, "enabling without accepting a report must not fabricate history")

	// Model state supplied by the future writer; this is not a keeper test.
	pruning.EverReported = true
	pruning.NextKey = types.FactUseReceiptKey(0, factUseConsumer(1), "fact-1")
	require.NoError(t, types.ValidateFactUseNonEconomicState(enabled, pruning, true))
	disabled := proto.Clone(enabled).(*types.Params)
	disabled.FactUseEnabled = false
	require.NoError(t, types.ValidateFactUseParamChange(enabled, disabled, pruning, true))
	require.Error(t, types.ValidateFactUseParamChange(disabled, &legacy, pruning, true))

	// Pruning removes the cursor and last receipt, never the persisted latch.
	pruning.NextKey = nil
	before := proto.Clone(pruning)
	require.NoError(t, types.ValidateFactUseNonEconomicState(disabled, pruning, false))
	require.Error(t, types.ValidateFactUseParamChange(disabled, &legacy, pruning, false))
	resized := proto.Clone(disabled).(*types.Params)
	resized.FitnessEpochBlocks++
	require.NoError(t, types.ValidateFactUseParamChange(disabled, resized, pruning, false), "permanent latch alone must not freeze epochs")
	reenabled := proto.Clone(resized).(*types.Params)
	reenabled.FactUseEnabled = true
	require.NoError(t, types.ValidateFactUseParamChange(resized, reenabled, pruning, false))
	for _, mutate := range []func(*types.Params){
		func(p *types.Params) { p.FitnessWeightQueryBps = 1 },
		func(p *types.Params) { p.FitnessWeightSatisfactionBps = 1 },
		func(p *types.Params) { p.MetabolismEnergyPerQuery = 1 },
	} {
		bad := proto.Clone(reenabled).(*types.Params)
		mutate(bad)
		require.Error(t, types.ValidateFactUseParamChange(resized, bad, pruning, false))
	}
	require.True(t, proto.Equal(before, pruning), "validation must retain the latch without reinitializing pruning state")
}

func TestFactUseNonEconomicStateRejectsMissingLatch(t *testing.T) {
	p := factUseParams()
	p.FactUseEnabled = false
	for _, pruning := range []*types.FactUsePruningState{nil, {}} {
		require.NoError(t, types.ValidateFactUseNonEconomicState(p, pruning, false))
		require.ErrorContains(t, types.ValidateFactUseNonEconomicState(p, pruning, true), "ever_reported")
		require.ErrorContains(t, types.ValidateFactUseParamChange(p, p, pruning, true), "ever_reported")
		require.False(t, pruning.GetEverReported(), "validation must reject, not repair, missing history")
	}
	require.Error(t, types.ValidateFactUseNonEconomicState(nil, nil, false))
	require.Error(t, types.ValidateFactUseParamChange(p, nil, nil, false))
	legacy := types.DefaultParams()
	require.NoError(t, types.ValidateFactUseNonEconomicState(&legacy, nil, false), "no latch and no reports preserves legacy economics, without authorizing new economics")
}

func TestFactUseEpochBoundariesAndRetention(t *testing.T) {
	for _, tc := range []struct {
		height        int64
		epoch, expiry uint64
	}{
		{9, 0, 20}, {10, 1, 30}, {11, 1, 30}, {19, 1, 30}, {20, 2, 40},
	} {
		epoch, err := types.FactUseEpoch(tc.height, 10)
		require.NoError(t, err)
		require.Equal(t, tc.epoch, epoch)
		expiry, err := types.FactUseExpiryHeight(epoch, 10)
		require.NoError(t, err)
		require.Equal(t, tc.expiry, expiry)
	}
	_, err := types.FactUseEpoch(-1, 10)
	require.Error(t, err)
	_, err = types.FactUseEpoch(1, 0)
	require.Error(t, err)
	_, err = types.FactUseExpiryHeight(math.MaxUint64, 10)
	require.Error(t, err)
	_, err = types.FactUseExpiryHeight(math.MaxInt64/10, 10)
	require.Error(t, err)
	r := factUseReceipt()
	require.NoError(t, r.Validate(10))
	r.Rating = types.FactUseRating_FACT_USE_RATING_USEFUL
	r.RatingHeight = 9
	require.NoError(t, r.Validate(10))
	r.RatingHeight = 10
	require.Error(t, r.Validate(10), "retained marker in next epoch cannot be rated")
	for name, mutate := range map[string]func(*types.FactUseReceipt){
		"version zero":       func(r *types.FactUseReceipt) { r.Version = 0 },
		"unknown version":    func(r *types.FactUseReceipt) { r.Version = 2 },
		"bad epoch":          func(r *types.FactUseReceipt) { r.Epoch = 1 },
		"use zero":           func(r *types.FactUseReceipt) { r.UseHeight = 0 },
		"use overflow":       func(r *types.FactUseReceipt) { r.UseHeight = math.MaxUint64 },
		"expiry":             func(r *types.FactUseReceipt) { r.ExpiryHeight++ },
		"unknown rating":     func(r *types.FactUseReceipt) { r.Rating = 99 },
		"unspecified rating": func(r *types.FactUseReceipt) { r.Rating = types.FactUseRating_FACT_USE_RATING_UNSPECIFIED },
		"unrated height":     func(r *types.FactUseReceipt) { r.RatingHeight = 9 },
		"early rating": func(r *types.FactUseReceipt) {
			r.Rating = types.FactUseRating_FACT_USE_RATING_NOT_USEFUL
			r.RatingHeight = 8
		},
	} {
		t.Run(name, func(t *testing.T) { r := factUseReceipt(); mutate(r); require.Error(t, r.Validate(10)) })
	}
}

func TestFactUseKeysAndCanonicalConsumer(t *testing.T) {
	consumer := factUseConsumer(1)
	canonical, err := types.CanonicalFactUseConsumer(consumer)
	require.NoError(t, err)
	require.Equal(t, consumer, canonical)
	for _, invalid := range []string{"", " " + consumer, strings.ToUpper(consumer), "zrn1invalid", strings.Repeat("x", 129)} {
		_, err := types.CanonicalFactUseConsumer(invalid)
		require.Error(t, err)
	}
	key := types.FactUseReceiptKey(7, consumer, "fact-1")
	require.Equal(t, byte(0x90), key[0])
	require.Equal(t, uint64(7), binary.BigEndian.Uint64(key[1:9]))
	require.Equal(t, consumer+"\x00fact-1", string(key[9:]))
	epoch, decoded, factID, err := types.ParseFactUseReceiptKey(key)
	require.NoError(t, err)
	require.Equal(t, uint64(7), epoch)
	require.Equal(t, consumer, decoded)
	require.Equal(t, "fact-1", factID)
	require.True(t, bytes.HasPrefix(key, types.FactUseReceiptConsumerPrefix(7, consumer)))
	require.True(t, bytes.Compare(key, types.FactUseReceiptKey(8, consumer, "fact-1")) < 0)
	require.NotEqual(t, key, types.FactUseReceiptKey(7, factUseConsumer(2), "fact-1"))
	require.Equal(t, append([]byte{0x91}, sdk.Uint64ToBigEndian(7)...), types.FactUseEpochCountKey(7))
	require.Equal(t, append(append([]byte{0x92}, sdk.Uint64ToBigEndian(7)...), []byte(consumer)...), types.FactUseConsumerCountKey(7, consumer))
	require.Equal(t, []byte{0x93}, types.FactUsePruningStateKey)
	for _, invalid := range [][]byte{nil, {0x90}, types.QueryReceiptKey(consumer, "fact-1"), types.FactUseReceiptKey(7, consumer, "a/b"), types.FactUseReceiptKey(7, "bad", "fact-1"), append(key, 0)} {
		_, _, _, err := types.ParseFactUseReceiptKey(invalid)
		require.Error(t, err)
	}
	key[0] = 0
	require.Equal(t, []byte{0x90}, types.FactUseReceiptPrefix, "constructors must not mutate prefix backing storage")
}

func TestFactUseValidateBasicAndSigner(t *testing.T) {
	msg := &types.MsgReportFactUse{Consumer: factUseConsumer(1), FactId: "fact-1"}
	require.NoError(t, msg.ValidateBasic())
	for _, id := range []string{"", "a/b", "a\x00b", "white space", "事", strings.Repeat("a", 129)} {
		msg.FactId = id
		require.Error(t, msg.ValidateBasic())
	}
	msg.FactId = strings.Repeat("a", 128)
	require.NoError(t, msg.ValidateBasic())
	msg.Consumer = strings.ToUpper(msg.Consumer)
	require.Error(t, msg.ValidateBasic())
	msg.Consumer = factUseConsumer(1)
	require.Error(t, (*types.MsgReportFactUse)(nil).ValidateBasic())

	registry, err := cdctypes.NewInterfaceRegistryWithOptions(cdctypes.InterfaceRegistryOptions{
		ProtoFiles: protoregistry.GlobalFiles,
		SigningOptions: signing.Options{
			AddressCodec:          address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
			ValidatorAddressCodec: address.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		},
	})
	require.NoError(t, err)
	types.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	signers, _, err := cdc.GetMsgV1Signers(msg)
	require.NoError(t, err)
	require.Equal(t, [][]byte{bytes.Repeat([]byte{1}, 20)}, signers)
	packed, err := cdctypes.NewAnyWithValue(msg)
	require.NoError(t, err)
	require.Equal(t, "/zerone.knowledge.v1.MsgReportFactUse", packed.TypeUrl)
	var unpacked sdk.Msg
	require.NoError(t, registry.UnpackAny(&cdctypes.Any{TypeUrl: packed.TypeUrl, Value: packed.Value}, &unpacked))
	require.True(t, proto.Equal(msg, unpacked.(*types.MsgReportFactUse)))

	rating := &types.MsgRateFact{Rater: factUseConsumer(1), FactId: "fact-1", Memo: strings.Repeat("é", 128)}
	require.NoError(t, rating.ValidateBasic())
	rating.Memo += "x"
	require.Error(t, rating.ValidateBasic())
	rating.Memo = string([]byte{0xff})
	require.Error(t, rating.ValidateBasic())
	require.Error(t, (*types.MsgRateFact)(nil).ValidateBasic())

	fields := (&types.QueryFactRequest{}).ProtoReflect().Descriptor().Fields()
	require.True(t, fields.ByNumber(2).Options().(*descriptorpb.FieldOptions).GetDeprecated())
	require.True(t, fields.ByNumber(3).Options().(*descriptorpb.FieldOptions).GetDeprecated())
	claimField := (&types.Claim{}).ProtoReflect().Descriptor().Fields().ByNumber(25)
	require.Equal(t, "challenge_evidence_ids", string(claimField.Name()))
}

func TestChallengeEvidenceBounds(t *testing.T) {
	ids := make([]string, 16)
	for i := range ids {
		ids[i] = fmt.Sprintf("fact-%d", i)
	}
	for _, provisional := range []bool{false, true} {
		validate := func(reason string, evidence []string) error {
			if provisional {
				return (&types.MsgChallengeProvisionalFact{FactId: "target", Reason: reason, EvidenceIds: evidence}).ValidateBasic()
			}
			return (&types.MsgChallengeFact{FactId: "target", Reason: reason, EvidenceIds: evidence}).ValidateBasic()
		}
		require.NoError(t, validate(strings.Repeat("a", types.MaxChallengeReasonBytes), ids))
		require.NoError(t, validate("Reason without cited evidence", nil))
		require.Error(t, validate(strings.Repeat("a", types.MaxChallengeReasonBytes+1), ids))
		require.Error(t, validate(" \n\t", ids))
		require.Error(t, validate(string([]byte{0xff}), ids))
		require.Error(t, validate("Reason", append(append([]string{}, ids...), "extra")))
		require.Error(t, validate("Reason", []string{"fact-1", "fact-1"}))
		require.Error(t, validate("Reason", []string{"bad/id"}))
	}
}

func TestFeedbackGenesisChallengeEvidenceBounds(t *testing.T) {
	ids := make([]string, types.MaxChallengeEvidenceIDs)
	for i := range ids {
		ids[i] = fmt.Sprintf("fact-%d", i)
	}
	for name, evidence := range map[string][]string{
		"17 IDs":      append(append([]string{}, ids...), "extra"),
		"duplicate":   {"fact-1", "fact-1"},
		"overlong ID": {strings.Repeat("a", types.MaxFactUseFactIDBytes+1)},
		"separator":   {"bad/id"},
	} {
		t.Run(name, func(t *testing.T) {
			gs := types.DefaultGenesis()
			gs.PendingClaims = []*types.Claim{nil, {Id: "challenge", ChallengeEvidenceIds: evidence}}
			require.Error(t, gs.Validate())
		})
	}
	for _, reason := range []string{"", " \n\t", strings.Repeat("a", types.MaxChallengeReasonBytes+1)} {
		gs := types.DefaultGenesis()
		gs.PendingClaims = []*types.Claim{nil, {Id: "legacy", ArgumentText: reason, ChallengeEvidenceIds: ids}}
		before := proto.Clone(gs)
		require.NoError(t, gs.Validate(), "new transaction reason limits must not reject historical claims")
		require.True(t, proto.Equal(before, gs), "validation must not rewrite historical claims")
		gs.PendingClaims[1].ChallengeEvidenceIds = nil
		require.NoError(t, gs.Validate(), "absent historical evidence stays absent")
	}
}

func TestFeedbackGenesisCompletionConsistency(t *testing.T) {
	for name, mutate := range map[string]func(*types.GenesisState){
		"same round at two heights": func(gs *types.GenesisState) {
			gs.CompletedRounds = nil // the round record need not exist to detect this conflict
			other := proto.Clone(gs.CompletedRoundRecords[0]).(*types.CompletedRoundRecord)
			other.VerdictBlock++
			gs.CompletedRoundRecords = append(gs.CompletedRoundRecords, other)
		},
		"mismatched height":    func(gs *types.GenesisState) { gs.CompletedRoundRecords[0].VerdictBlock++ },
		"zero height mismatch": func(gs *types.GenesisState) { gs.CompletedRounds[0].VerdictBlock = 0 },
		"active round": func(gs *types.GenesisState) {
			gs.ActiveRounds, gs.CompletedRounds = gs.CompletedRounds, nil
			gs.ActiveRounds[0].Phase = types.VerificationPhase_VERIFICATION_PHASE_COMMIT
		},
		"terminal round in active list": func(gs *types.GenesisState) {
			gs.ActiveRounds, gs.CompletedRounds = gs.CompletedRounds, nil
		},
		"nonterminal completed round": func(gs *types.GenesisState) {
			gs.CompletedRounds[0].Phase = types.VerificationPhase_VERIFICATION_PHASE_REVEAL
		},
	} {
		t.Run(name, func(t *testing.T) {
			gs := feedbackGenesis()
			mutate(gs)
			require.Error(t, gs.Validate())
		})
	}
	for name, mutate := range map[string]func(*types.GenesisState){
		"standalone metadata": func(gs *types.GenesisState) { gs.CompletedRounds = nil },
		"sparse metadata":     func(gs *types.GenesisState) { gs.CompletedRoundRecords = nil },
		"expired round": func(gs *types.GenesisState) {
			gs.CompletedRounds[0].Phase = types.VerificationPhase_VERIFICATION_PHASE_EXPIRED
		},
		"different rounds at same height": func(gs *types.GenesisState) {
			other := proto.Clone(gs.CompletedRoundRecords[0]).(*types.CompletedRoundRecord)
			other.RoundId = "historical-round"
			gs.CompletedRoundRecords = append(gs.CompletedRoundRecords, other)
		},
	} {
		t.Run(name, func(t *testing.T) {
			gs := feedbackGenesis()
			mutate(gs)
			before := proto.Clone(gs)
			require.NoError(t, gs.Validate())
			require.True(t, proto.Equal(before, gs), "validation must not synthesize missing history")
		})
	}
}

func TestFactUseParamChangeCannotMonetizeReportedCounters(t *testing.T) {
	pruning := &types.FactUsePruningState{EverReported: true}
	for _, enabled := range []bool{true, false} {
		for _, retained := range []bool{true, false} {
			for name, mutate := range map[string]func(*types.Params){
				"query weight":        func(p *types.Params) { p.FitnessWeightQueryBps = 1 },
				"satisfaction weight": func(p *types.Params) { p.FitnessWeightSatisfactionBps = 1 },
				"query energy":        func(p *types.Params) { p.MetabolismEnergyPerQuery = 1 },
			} {
				t.Run(fmt.Sprintf("enabled=%t/retained=%t/%s", enabled, retained, name), func(t *testing.T) {
					current := factUseParams()
					current.FactUseEnabled = enabled
					proposed := proto.Clone(current).(*types.Params)
					proposed.FactUseEnabled = false
					mutate(proposed)
					require.Error(t, types.ValidateFactUseParamChange(current, proposed, pruning, retained), "self-reported counters outlive disabled beta and pruned receipts")
				})
			}
		}
	}
}

func feedbackGenesis() *types.GenesisState {
	gs := types.DefaultGenesis()
	gs.Params = factUseParams()
	gs.Params.FactUseEnabled = false // historical reports remain non-economic while disabled
	gs.Params.FitnessEpochBlocks = 10
	gs.Facts = []*types.Fact{{Id: "fact-1"}, {Id: "fact-2"}}
	gs.PendingClaims = []*types.Claim{{Id: "challenge", ArgumentText: "Preserved reason", ChallengeEvidenceIds: []string{"fact-2"}}}
	gs.FactRelations = []*types.FactRelation{{SourceFactId: "fact-1", TargetFactId: "fact-2", Relation: types.RelationType_RELATION_TYPE_SUPPORTS}}
	gs.StatusTransitions = []*types.StatusTransition{{FactId: "fact-1", Seq: 3, BlockHeight: 8, PriorStatus: types.FactStatus_FACT_STATUS_VERIFIED, NewStatus: types.FactStatus_FACT_STATUS_CHALLENGED}}
	gs.StatusTransitionSequences = []*types.StatusTransitionSequence{{FactId: "fact-1", LastSequence: 7}, {FactId: "no-recorded-history", LastSequence: 5}}
	gs.CascadeEvents = []*types.CascadeEvent{{Seq: 4, DisprovenFactId: "fact-2", DescendantFactId: "fact-1", BlockHeight: 9}}
	gs.CompletedRounds = []*types.VerificationRound{{Id: "round-1", ClaimId: "challenge", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, VerdictBlock: 9}}
	gs.CompletedRoundRecords = []*types.CompletedRoundRecord{{RoundId: "round-1", VerdictBlock: 9, Meta: &types.CompletedRoundMeta{Domain: "math", HasDissent: true, DurationBlocks: 8}}}
	gs.FactUseReceipts = []*types.FactUseReceipt{factUseReceipt()}
	gs.FactUsePruning = &types.FactUsePruningState{NextKey: types.FactUseReceiptKey(0, factUseConsumer(1), "fact-1"), EverReported: true}
	return gs
}

func TestFeedbackGenesisReportedLatch(t *testing.T) {
	for name, mutate := range map[string]func(*types.GenesisState){
		"retained without state":    func(gs *types.GenesisState) { gs.FactUsePruning = nil },
		"retained with false latch": func(gs *types.GenesisState) { gs.FactUsePruning.EverReported = false },
		"pruned query weight": func(gs *types.GenesisState) {
			gs.FactUseReceipts = nil
			gs.Params.FitnessWeightQueryBps = 1
		},
		"pruned satisfaction weight": func(gs *types.GenesisState) {
			gs.FactUseReceipts = nil
			gs.Params.FitnessWeightSatisfactionBps = 1
		},
		"pruned query energy": func(gs *types.GenesisState) {
			gs.FactUseReceipts = nil
			gs.Params.MetabolismEnergyPerQuery = 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			gs := feedbackGenesis()
			mutate(gs)
			before := proto.Clone(gs)
			require.Error(t, gs.Validate())
			require.True(t, proto.Equal(before, gs), "validation must neither synthesize nor clear the latch")
		})
	}
}

func TestFeedbackGenesisPrunedLatchRoundTripAndLegacyAbsence(t *testing.T) {
	gs := feedbackGenesis()
	gs.FactUseReceipts = nil
	gs.FactUsePruning.NextKey = nil // last receipt and cursor gone, permanent latch remains
	require.NoError(t, gs.Validate())
	for _, jsonEncoding := range []bool{false, true} {
		var bz []byte
		var err error
		decoded := &types.GenesisState{}
		if jsonEncoding {
			bz, err = protojson.Marshal(gs)
			require.NoError(t, err)
			require.NoError(t, protojson.Unmarshal(bz, decoded))
		} else {
			bz, err = proto.MarshalOptions{Deterministic: true}.Marshal(gs)
			require.NoError(t, err)
			require.NoError(t, proto.Unmarshal(bz, decoded))
		}
		require.True(t, proto.Equal(gs, decoded))
		require.True(t, decoded.FactUsePruning.GetEverReported())
		require.Empty(t, decoded.FactUsePruning.NextKey)
		require.Empty(t, decoded.FactUseReceipts)
		require.NoError(t, decoded.Validate())
		decoded.Params.MetabolismEnergyPerQuery = 1
		require.Error(t, decoded.Validate(), "import must retain the economic wall after the last receipt")
	}
	legacy := types.DefaultGenesis()
	require.Nil(t, legacy.FactUsePruning)
	require.False(t, legacy.FactUsePruning.GetEverReported())
	require.NoError(t, legacy.Validate())
	require.Nil(t, legacy.FactUsePruning, "absent historical state must not be synthesized")
	legacy.FactUsePruning = &types.FactUsePruningState{}
	require.NoError(t, legacy.Validate(), "new migration state defaults false without altering legacy counters")
	require.False(t, legacy.FactUsePruning.EverReported)
	field := legacy.FactUsePruning.ProtoReflect().Descriptor().Fields().ByNumber(2)
	require.Equal(t, "ever_reported", string(field.Name()))
}

func TestFeedbackGenesisWireAndJSONRoundTrip(t *testing.T) {
	gs := feedbackGenesis()
	require.NoError(t, gs.Validate())
	bz, err := proto.MarshalOptions{Deterministic: true}.Marshal(gs)
	require.NoError(t, err)
	decoded := &types.GenesisState{}
	require.NoError(t, proto.Unmarshal(bz, decoded))
	require.True(t, proto.Equal(gs, decoded))
	json, err := protojson.Marshal(gs)
	require.NoError(t, err)
	decoded.Reset()
	require.NoError(t, protojson.Unmarshal(json, decoded))
	require.True(t, proto.Equal(gs, decoded))
	require.NoError(t, decoded.Validate())
	require.Empty(t, decoded.PendingClaims[0].References, "challenge evidence is not provenance")
	require.Equal(t, uint64(7), decoded.StatusTransitionSequences[0].LastSequence, "never infer counter from history length/max")
}

func TestFeedbackGenesisQuotaCeilingsPreserveLoweredPolicy(t *testing.T) {
	gs := feedbackGenesis()
	gs.FactUseReceipts = nil
	for i := 0; i < 1000; i++ {
		r := factUseReceipt()
		r.Consumer = factUseConsumer(byte(i/100 + 1))
		r.FactId = fmt.Sprintf("fact-%d", i)
		gs.FactUseReceipts = append(gs.FactUseReceipts, r)
	}
	require.NoError(t, gs.Validate())
	gs.Params.FactUseMaxPerConsumerEpoch = 1
	gs.Params.FactUseMaxPerEpoch = 1
	require.NoError(t, gs.Validate(), "do not reject old admitted receipts after quota or cohort changes")
	r := factUseReceipt()
	r.Consumer = factUseConsumer(11)
	r.FactId = "one-too-many"
	gs.FactUseReceipts = append(gs.FactUseReceipts, r)
	require.ErrorContains(t, gs.Validate(), "epoch hard ceilings")
	gs.FactUseReceipts = gs.FactUseReceipts[:101]
	gs.FactUseReceipts[100].Consumer = factUseConsumer(1)
	require.ErrorContains(t, gs.Validate(), "epoch hard ceilings")
}

func TestFeedbackGenesisRejectsConflictingKeys(t *testing.T) {
	for name, mutate := range map[string]func(*types.GenesisState){
		"relation": func(gs *types.GenesisState) { gs.FactRelations = append(gs.FactRelations, gs.FactRelations[0]) },
		"transition": func(gs *types.GenesisState) {
			gs.StatusTransitions = append(gs.StatusTransitions, gs.StatusTransitions[0])
		},
		"counter": func(gs *types.GenesisState) {
			gs.StatusTransitionSequences = append(gs.StatusTransitionSequences, gs.StatusTransitionSequences[0])
		},
		"counter too low": func(gs *types.GenesisState) { gs.StatusTransitionSequences[0].LastSequence = 2 },
		"missing counter": func(gs *types.GenesisState) { gs.StatusTransitionSequences = nil },
		"cascade":         func(gs *types.GenesisState) { gs.CascadeEvents = append(gs.CascadeEvents, gs.CascadeEvents[0]) },
		"round":           func(gs *types.GenesisState) { gs.ActiveRounds = gs.CompletedRounds },
		"nonterminal": func(gs *types.GenesisState) {
			gs.CompletedRounds[0].Phase = types.VerificationPhase_VERIFICATION_PHASE_COMMIT
		},
		"completion metadata": func(gs *types.GenesisState) {
			gs.CompletedRoundRecords = append(gs.CompletedRoundRecords, gs.CompletedRoundRecords[0])
		},
		"receipt":     func(gs *types.GenesisState) { gs.FactUseReceipts = append(gs.FactUseReceipts, gs.FactUseReceipts[0]) },
		"nil receipt": func(gs *types.GenesisState) { gs.FactUseReceipts = []*types.FactUseReceipt{nil} },
		"cursor":      func(gs *types.GenesisState) { gs.FactUsePruning.NextKey = []byte{0x3e} },
		"capacity": func(gs *types.GenesisState) {
			gs.FactUseReceipts = make([]*types.FactUseReceipt, types.MaxFactUseRetainedReceipts+1)
		},
	} {
		t.Run(name, func(t *testing.T) { gs := feedbackGenesis(); mutate(gs); require.Error(t, gs.Validate()) })
	}
}
