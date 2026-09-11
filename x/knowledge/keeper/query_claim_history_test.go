package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	corestore "cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

func putClaimHistoryRecord(t *testing.T, f handoffFixture, key []byte, record proto.Message) {
	t.Helper()
	value, err := marshalOpts.Marshal(record)
	require.NoError(t, err)
	f.ctx.KVStore(f.keys[0]).Set(key, value)
}

func setupClaimHistory(t *testing.T) handoffFixture {
	t.Helper()
	f := setupRelationGenesis(t, nil)
	f.ctx = f.ctx.WithChainID("claim-history-local")
	require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: "fact", ClaimId: "claim", Status: types.FactStatus_FACT_STATUS_VERIFIED}))
	require.NoError(t, f.k.EnableRecordIntegrity(f.ctx))
	require.NoError(t, f.k.EnableReviewNeutrality(f.ctx))
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())
	// Imported retained-record fixtures, not synthetic signed transactions or
	// assertions that a historical nominal fee was actually paid.
	claim := &types.Claim{Id: "claim", FactContent: "original text", Domain: "physics", Status: types.ClaimStatus_CLAIM_STATUS_ACCEPTED, VerificationRoundId: "round-old", Stake: "200000"}
	putClaimHistoryRecord(t, f, types.ClaimKey(claim.Id), claim)
	round := &types.VerificationRound{Id: "round-old", ClaimId: claim.Id, StartedAtBlock: 10, Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: types.Verdict_VERDICT_ACCEPT, VerdictBlock: 20}
	putClaimHistoryRecord(t, f, types.RoundKey(round.Id), round)
	return f
}

func TestClaimHistoryPreservesRecordsAndSelectsDirectChallenges(t *testing.T) {
	f := setupClaimHistory(t)
	rootClaim, ok := f.k.GetClaim(f.ctx, "claim")
	require.True(t, ok)
	oldRound, ok := f.k.GetVerificationRound(f.ctx, "round-old")
	require.True(t, ok)
	verifier := sdk.AccAddress(bytes.Repeat([]byte{0x51}, 20)).String()
	attestation := &types.ReviewAttestation{Reason: "I reproduced the stated calculation", MethodId: "M-CHECK", Scope: "table 1 only", EvidenceIds: []string{"observed-input", "calculation"}}
	salt := bytes.Repeat([]byte{0x67}, 16)
	hash, err := types.ComputeReviewCommitmentV2("original-review-chain", "round-v2", verifier, "reject", 712345, salt, attestation)
	require.NoError(t, err)
	round := &types.VerificationRound{
		Id: "round-v2", ClaimId: "claim", StartedAtBlock: 50, Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE,
		Verdict: types.Verdict_VERDICT_INCONCLUSIVE, VerdictBlock: 90, CommitmentScheme: 2, CommitmentChainId: "original-review-chain",
		SelectedVerifiers: []string{verifier}, Commits: []*types.CommitEntry{{Verifier: verifier, CommitHash: hash, CommittedAtBlock: 51}},
		Reveals:                  []*types.RevealEntry{{Verifier: verifier, Vote: "reject", Confidence: 712345, Salt: salt, RevealedAtBlock: 70, Attestation: attestation}},
		VerifierRewardSettlement: &types.VerifierRewardSettlement{CreatedAtBlock: 90, Payments: []*types.VerifierRewardPayment{{Verifier: verifier, Amount: "5", Withheld: "0"}}, WithheldTotal: "0"},
	}
	putClaimHistoryRecord(t, f, types.RoundKey(round.Id), round)
	expired := &types.VerificationRound{Id: "round-expired", ClaimId: "claim", StartedAtBlock: 30, Phase: types.VerificationPhase_VERIFICATION_PHASE_EXPIRED}
	putClaimHistoryRecord(t, f, types.RoundKey(expired.Id), expired)
	// These deliberately incomplete selectors must not hide retained primaries.
	f.ctx.KVStore(f.keys[0]).Set(types.ClaimRoundIndexKey("claim"), []byte("round-v2"))
	f.ctx.KVStore(f.keys[0]).Set(activeRoundKey("round-v2"), []byte{1})
	secondFact := &types.Fact{Id: "another-derived", ClaimId: "claim", Status: types.FactStatus_FACT_STATUS_PRUNED, MethodId: "historic-method", OutgoingRelations: []*types.FactRelation{{TargetFactId: "embedded-only"}}}
	putClaimHistoryRecord(t, f, types.FactKey(secondFact.Id), secondFact)
	relatedClaims := []*types.Claim{
		{Id: "challenge", ProvisionalFactId: "fact", ChallengedClaimId: "claim", ArgumentText: "exact challenge reason", CounterClaim: "different countertext", EvidenceIds: []string{"challenge-evidence"}, Status: types.ClaimStatus_CLAIM_STATUS_REJECTED},
		{Id: "contradiction", Relations: []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: "fact"}, {Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: "another-derived"}, {Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: "fact"}}, Status: types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT},
		{Id: "claim-id-only", ChallengedClaimId: "claim", Status: types.ClaimStatus_CLAIM_STATUS_PENDING},
	}
	for _, claim := range relatedClaims {
		putClaimHistoryRecord(t, f, types.ClaimKey(claim.Id), claim)
		r := &types.VerificationRound{Id: "round-" + claim.Id, ClaimId: claim.Id, Phase: types.VerificationPhase_VERIFICATION_PHASE_COMMIT}
		putClaimHistoryRecord(t, f, types.RoundKey(r.Id), r)
	}
	derivedChallenge := &types.Fact{Id: "challenge-derived", ClaimId: "challenge", Status: types.FactStatus_FACT_STATUS_ACTIVE}
	putClaimHistoryRecord(t, f, types.FactKey(derivedChallenge.Id), derivedChallenge)
	for _, claim := range []*types.Claim{
		{Id: "recursive", ProvisionalFactId: "challenge-derived"},
		{Id: "support-only", Relations: []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_SUPPORTS, TargetFactId: "fact"}}},
		{Id: "canonical-neighbor"},
	} {
		putClaimHistoryRecord(t, f, types.ClaimKey(claim.Id), claim)
	}
	neighbor := &types.Fact{Id: "neighbor", ClaimId: "canonical-neighbor"}
	putClaimHistoryRecord(t, f, types.FactKey(neighbor.Id), neighbor)
	edge := &types.FactRelation{SourceFactId: "fact", TargetFactId: "neighbor", Relation: types.RelationType_RELATION_TYPE_REQUIRES, CreatedAtBlock: 63, Creator: "historic-author", Inference: types.InferenceType_INFERENCE_TYPE_EMPIRICAL, InferenceStrengthBps: 654321, MethodId: "original-method"}
	incoming := &types.FactRelation{SourceFactId: "neighbor", TargetFactId: "fact", Relation: types.RelationType_RELATION_TYPE_CONTRADICTS}
	require.NoError(t, f.k.SetFactRelation(f.ctx, edge))
	require.NoError(t, f.k.SetFactRelation(f.ctx, incoming))
	transitions := []*types.StatusTransition{
		{FactId: "fact", Seq: 3, PriorStatus: types.FactStatus_FACT_STATUS_VERIFIED, NewStatus: types.FactStatus_FACT_STATUS_CONTESTED, BlockHeight: 62},
		{FactId: "fact", Seq: 9, PriorStatus: types.FactStatus_FACT_STATUS_CONTESTED, NewStatus: types.FactStatus_FACT_STATUS_ACTIVE, BlockHeight: 80, CauseId: "literal-cause"},
	}
	// Preserve a deliberately sparse retained-history fixture; the query must
	// neither recreate the missing first transition nor fill sequence gaps.
	f.ctx.KVStore(f.keys[0]).Delete(types.StatusTransitionKey("fact", 1))
	for _, tr := range transitions {
		putClaimHistoryRecord(t, f, types.StatusTransitionKey(tr.FactId, tr.Seq), tr)
	}
	f.ctx.KVStore(f.keys[0]).Set(types.StatusTransitionSeqKey("fact"), binary.AppendUvarint(nil, 12))
	before := f.snapshot(t)
	q := NewQueryServerImpl(f.k)
	response, err := q.ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim", AtBlockHeight: 100})
	require.NoError(t, err)
	require.Equal(t, "claim-history-local", response.ChainId)
	require.Equal(t, uint64(100), response.BlockHeight)
	require.True(t, proto.Equal(rootClaim, response.Record.Claim))
	require.Len(t, response.Record.Rounds, 3)
	for i, want := range []*types.VerificationRound{oldRound, expired, round} {
		require.True(t, proto.Equal(want, response.Record.Rounds[i]))
	}
	require.Empty(t, response.Record.Rounds[0].Reveals)
	require.Empty(t, response.Record.MissingRoundIds)
	require.Len(t, response.Record.Facts, 2)
	require.True(t, proto.Equal(secondFact, response.Record.Facts[0].Fact))
	require.Empty(t, response.Record.Facts[0].OutgoingRelations)
	factHistory := response.Record.Facts[1]
	require.Len(t, factHistory.OutgoingRelations, 1)
	require.True(t, proto.Equal(edge, factHistory.OutgoingRelations[0]))
	require.True(t, proto.Equal(incoming, factHistory.IncomingRelations[0]))
	require.Len(t, factHistory.StatusTransitions, 2)
	for i, want := range transitions {
		require.True(t, proto.Equal(want, factHistory.StatusTransitions[i]))
	}
	require.Empty(t, factHistory.StatusTransitions[0].CauseId)
	require.Len(t, response.RelatedClaims, 3)
	require.Equal(t, "challenge", response.RelatedClaims[0].Record.ClaimId)
	require.True(t, proto.Equal(relatedClaims[0], response.RelatedClaims[0].Record.Claim))
	require.Len(t, response.RelatedClaims[0].Record.Facts, 1)
	require.Len(t, response.RelatedClaims[0].Links, 2)
	require.Equal(t, "claim-id-only", response.RelatedClaims[1].Record.ClaimId)
	require.Equal(t, "contradiction", response.RelatedClaims[2].Record.ClaimId)
	require.Len(t, response.RelatedClaims[2].Links, 2)
	require.Equal(t, "relations.contradicts", response.RelatedClaims[2].Links[0].Field)
	require.Equal(t, "another-derived", response.RelatedClaims[2].Links[0].TargetId)
	for _, entry := range response.RelatedClaims {
		require.Len(t, entry.Record.Rounds, 1)
	}
	again, err := q.ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
	require.NoError(t, err)
	require.True(t, proto.Equal(response, again))
	require.Equal(t, before, f.snapshot(t))
	require.Empty(t, f.ctx.EventManager().Events())
}

func TestClaimHistoryMissingRecordsRemainAbsent(t *testing.T) {
	f := setupClaimHistory(t)
	q := NewQueryServerImpl(f.k)
	f.ctx.KVStore(f.keys[0]).Delete(types.RoundKey("round-old"))
	response, err := q.ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
	require.NoError(t, err)
	require.Equal(t, []string{"round-old"}, response.Record.MissingRoundIds)
	require.Empty(t, response.Record.Rounds)
	f.ctx.KVStore(f.keys[0]).Delete(types.ClaimKey("claim"))
	response, err = q.ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
	require.NoError(t, err)
	require.Nil(t, response.Record.Claim)
	require.Len(t, response.Record.Facts, 1)
	require.Empty(t, response.Record.MissingRoundIds)
	round := &types.VerificationRound{Id: "orphan-round", ClaimId: "orphan", Phase: types.VerificationPhase_VERIFICATION_PHASE_EXPIRED}
	putClaimHistoryRecord(t, f, types.RoundKey(round.Id), round)
	response, err = q.ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "orphan"})
	require.NoError(t, err)
	require.Nil(t, response.Record.Claim)
	require.True(t, proto.Equal(round, response.Record.Rounds[0]))
	claim := &types.Claim{Id: "asserted-link", ChallengedClaimId: "missing-root"}
	putClaimHistoryRecord(t, f, types.ClaimKey(claim.Id), claim)
	response, err = q.ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "missing-root"})
	require.NoError(t, err)
	require.Nil(t, response.Record.Claim)
	require.Len(t, response.RelatedClaims, 1)
	response, err = q.ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "entirely-absent"})
	require.Nil(t, response)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestClaimHistoryActualRetainedSDKContext(t *testing.T) {
	f := setupClaimHistory(t)
	ms := f.ctx.MultiStore().(storetypes.CommitMultiStore)
	require.Equal(t, int64(1), ms.Commit().Version)
	claim, ok := f.k.GetClaim(f.ctx, "claim")
	require.True(t, ok)
	claim.FactContent = "later retained text"
	putClaimHistoryRecord(t, f, types.ClaimKey(claim.Id), claim)
	require.Equal(t, int64(2), ms.Commit().Version)
	oldStore, err := ms.CacheMultiStoreWithVersion(1)
	require.NoError(t, err)
	oldCtx := f.ctx.WithMultiStore(oldStore).WithBlockHeight(1)
	q := NewQueryServerImpl(f.k)
	old, err := q.ClaimHistory(oldCtx, &types.QueryClaimHistoryRequest{Id: "claim", AtBlockHeight: 1})
	require.NoError(t, err)
	require.Equal(t, "original text", old.Record.Claim.FactContent)
	require.Equal(t, uint64(1), old.BlockHeight)
	current, err := q.ClaimHistory(f.ctx.WithBlockHeight(2), &types.QueryClaimHistoryRequest{Id: "claim", AtBlockHeight: 2})
	require.NoError(t, err)
	require.Equal(t, "later retained text", current.Record.Claim.FactContent)
	response, err := q.ClaimHistory(f.ctx.WithBlockHeight(2), &types.QueryClaimHistoryRequest{Id: "claim", AtBlockHeight: 1})
	require.Nil(t, response)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestClaimHistoryInvalidRequests(t *testing.T) {
	f := setupClaimHistory(t)
	q := NewQueryServerImpl(f.k)
	for _, req := range []*types.QueryClaimHistoryRequest{nil, {}, {Id: string([]byte{0xff})}, {Id: strings.Repeat("x", 257)}, {Id: "claim", AtBlockHeight: ^uint64(0)}} {
		response, err := q.ClaimHistory(f.ctx, req)
		require.Nil(t, response)
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	}
	response, err := q.ClaimHistory(f.ctx.WithBlockHeight(-1), &types.QueryClaimHistoryRequest{Id: "claim"})
	require.Nil(t, response)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestClaimHistoryCorruptionRefusesWholeResponse(t *testing.T) {
	for _, name := range []string{"claim-decode", "claim-key", "claim-enum", "claim-overflow", "round-scheme-overflow", "round-enum", "fact-overflow", "fact-unknown", "nested-unknown", "mirror-missing", "mirror-unequal", "relation-overflow", "history-duplicate", "history-counter", "claim-round-owner", "unrelated-corruption"} {
		t.Run(name, func(t *testing.T) {
			f := setupClaimHistory(t)
			store := f.ctx.KVStore(f.keys[0])
			appendVarint := func(key []byte, tag protowire.Number, value uint64) {
				bytes := bytes.Clone(store.Get(key))
				bytes = protowire.AppendVarint(protowire.AppendTag(bytes, tag, protowire.VarintType), value)
				store.Set(key, bytes)
			}
			relation := &types.FactRelation{SourceFactId: "fact", TargetFactId: "other", Relation: types.RelationType_RELATION_TYPE_SUPPORTS}
			require.NoError(t, f.k.SetFactRelation(f.ctx, relation))
			fk, rk := types.FactRelationKey("fact", "other"), types.FactRelationReverseKey("other", "fact")
			switch name {
			case "claim-decode":
				store.Set(types.ClaimKey("claim"), []byte{0xff})
			case "claim-key":
				store.Set(types.ClaimKey("wrong"), store.Get(types.ClaimKey("claim")))
			case "claim-enum":
				claim := &types.Claim{Id: "claim", Status: 999}
				putClaimHistoryRecord(t, f, types.ClaimKey(claim.Id), claim)
			case "claim-overflow":
				appendVarint(types.ClaimKey("claim"), 7, 1<<32|uint64(types.ClaimStatus_CLAIM_STATUS_ACCEPTED))
			case "round-scheme-overflow":
				appendVarint(types.RoundKey("round-old"), 13, 1<<32)
			case "round-enum":
				round := &types.VerificationRound{Id: "round-old", ClaimId: "claim", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE, Verdict: 999}
				putClaimHistoryRecord(t, f, types.RoundKey(round.Id), round)
			case "fact-overflow":
				appendVarint(types.FactKey("fact"), 12, 1<<32|uint64(types.FactStatus_FACT_STATUS_VERIFIED))
			case "fact-unknown":
				appendVarint(types.FactKey("fact"), 999, 1)
			case "nested-unknown":
				claim := &types.Claim{Id: "claim", Relations: []*types.ClaimRelation{{TargetFactId: "other"}}}
				claim.Relations[0].ProtoReflect().SetUnknown(protowire.AppendVarint(protowire.AppendTag(nil, 99, protowire.VarintType), 1))
				putClaimHistoryRecord(t, f, types.ClaimKey(claim.Id), claim)
			case "mirror-missing":
				store.Delete(rk)
			case "mirror-unequal":
				relation.Creator = "other"
				putClaimHistoryRecord(t, f, rk, relation)
			case "relation-overflow":
				appendVarint(fk, 3, 1<<32|uint64(relation.Relation))
				store.Set(rk, store.Get(fk))
			case "history-duplicate", "history-counter":
				tr := &types.StatusTransition{FactId: "fact", Seq: 7, BlockHeight: 42}
				key := types.StatusTransitionKey(tr.FactId, tr.Seq)
				putClaimHistoryRecord(t, f, key, tr)
				if name == "history-duplicate" {
					appendVarint(key, 1, 7)
				} else {
					store.Set(types.StatusTransitionSeqKey("fact"), []byte{6})
				}
			case "claim-round-owner":
				round := &types.VerificationRound{Id: "round-old", ClaimId: "somebody-else", Phase: types.VerificationPhase_VERIFICATION_PHASE_COMPLETE}
				putClaimHistoryRecord(t, f, types.RoundKey(round.Id), round)
			case "unrelated-corruption":
				store.Set(types.ClaimKey("unrelated"), []byte{0xff})
			}
			before := f.snapshot(t)
			response, err := NewQueryServerImpl(f.k).ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
			require.Nil(t, response)
			require.Equal(t, codes.Internal, status.Code(err))
			require.Equal(t, before, f.snapshot(t))
			require.Empty(t, f.ctx.EventManager().Events())
		})
	}
}

func TestClaimHistoryStoreErrorsRefuseWholeResponse(t *testing.T) {
	for _, mode := range []string{"error", "close"} {
		f := setupClaimHistory(t)
		before := f.snapshot(t)
		bad := f.k
		bad.storeService = historyIteratorFaultService{bad.storeService, mode}
		response, err := NewQueryServerImpl(bad).ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
		require.Nil(t, response)
		require.Equal(t, codes.Internal, status.Code(err))
		require.Equal(t, before, f.snapshot(t))
	}
	for _, prefix := range [][]byte{[]byte(ReviewNeutralityEnabledStoreKey), types.StatusTransitionSeqKeyPrefix, types.FactRelationReversePrefix} {
		f := setupClaimHistory(t)
		require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: "fact", TargetFactId: "other"}))
		before := f.snapshot(t)
		*f.kFault = handoffFault{"get", prefix, false}
		response, err := NewQueryServerImpl(f.k).ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
		require.Nil(t, response)
		require.Equal(t, codes.Internal, status.Code(err))
		require.Equal(t, before, f.snapshot(t))
	}
}

type claimHistoryBudgetService struct {
	corestore.KVStoreService
	rows int
}

func (s claimHistoryBudgetService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return claimHistoryBudgetStore{s.KVStoreService.OpenKVStore(ctx), s.rows}
}

type claimHistoryBudgetStore struct {
	corestore.KVStore
	rows int
}

func (s claimHistoryBudgetStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	if bytes.Equal(start, types.FactKeyPrefix) {
		return &claimHistoryBudgetIterator{rows: s.rows}, nil
	}
	return s.KVStore.Iterator(start, end)
}

type claimHistoryBudgetIterator struct {
	corestore.Iterator
	rows, cursor int
}

func (it *claimHistoryBudgetIterator) Valid() bool { return it.cursor < it.rows }
func (it *claimHistoryBudgetIterator) Next()       { it.cursor++ }
func (it *claimHistoryBudgetIterator) Key() []byte {
	return types.FactKey(fmt.Sprintf("unrelated-%06d", it.cursor))
}
func (it *claimHistoryBudgetIterator) Value() []byte {
	value, _ := marshalOpts.Marshal(&types.Fact{Id: fmt.Sprintf("unrelated-%06d", it.cursor)})
	return value
}
func (it *claimHistoryBudgetIterator) Error() error { return nil }
func (it *claimHistoryBudgetIterator) Close() error { return nil }

func TestClaimHistoryResourceCeilings(t *testing.T) {
	t.Run("shared-entry-count", func(t *testing.T) {
		f := setupClaimHistory(t)
		bad := f.k
		bad.storeService = claimHistoryBudgetService{bad.storeService, ToKMaxReadEntries}
		response, err := NewQueryServerImpl(bad).ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
		require.Nil(t, response)
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
	t.Run("raw-bytes", func(t *testing.T) {
		f := setupClaimHistory(t)
		for i := 0; i < 17; i++ {
			fact := &types.Fact{Id: fmt.Sprintf("unrelated-%02d", i), Content: strings.Repeat("x", 1<<20)}
			putClaimHistoryRecord(t, f, types.FactKey(fact.Id), fact)
		}
		response, err := NewQueryServerImpl(f.k).ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
		require.Nil(t, response)
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
	t.Run("serialized-output", func(t *testing.T) {
		f := setupClaimHistory(t)
		fact := &types.Fact{Id: "fact", ClaimId: "claim", Content: strings.Repeat("x", ToKMaxOutputBytes)}
		putClaimHistoryRecord(t, f, types.FactKey(fact.Id), fact)
		response, err := NewQueryServerImpl(f.k).ClaimHistory(f.ctx, &types.QueryClaimHistoryRequest{Id: "claim"})
		require.Nil(t, response)
		require.Equal(t, codes.ResourceExhausted, status.Code(err))
	})
}
