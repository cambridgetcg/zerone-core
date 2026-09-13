package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ClaimHistory is a bounded view of records retained in this SDK query context.
// It never treats a requested height as permission to relabel current state.
func (q *queryServer) ClaimHistory(ctx context.Context, req *types.QueryClaimHistoryRequest) (*types.QueryClaimHistoryResponse, error) {
	if req == nil || req.Id == "" || !utf8.ValidString(req.Id) || len(req.Id) > 256 {
		return nil, status.Error(codes.InvalidArgument, "id must be nonempty UTF-8 of at most 256 bytes")
	}
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if height < 0 {
		return nil, status.Error(codes.Internal, "negative SDK query height")
	}
	if req.AtBlockHeight != 0 && req.AtBlockHeight != uint64(height) {
		return nil, status.Error(codes.InvalidArgument, "at_block_height must match the actual SDK query context; use a retained historical query height")
	}
	response, err := q.keeper.buildClaimHistory(ctx, req.Id)
	if err != nil {
		return nil, knowledgeRecordQueryError(err)
	}
	if response.Record.Claim == nil && len(response.Record.Rounds) == 0 && len(response.Record.Facts) == 0 && len(response.RelatedClaims) == 0 {
		return nil, status.Error(codes.NotFound, "no retained claim-history records for this id")
	}
	return response, nil
}

// buildClaimHistory scans each primary namespace once. Facts are retained only
// in bounded query memory while direct related claims are selected; no index is
// created and neither related claims nor neighboring facts expand recursively.
func (k Keeper) buildClaimHistory(ctx context.Context, id string) (*types.QueryClaimHistoryResponse, error) {
	k, guard := k.guardedToK()
	recordIntegrity, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		return nil, err
	}
	neutral, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil {
		return nil, err
	}
	funded, err := k.FundSettlementEnabled(ctx)
	if err != nil {
		return nil, err
	}
	fundedClaims := make(map[string]*types.Claim)
	fundedRoundCounts := make(map[string]int)
	factsByClaim := make(map[string][]*types.Fact)
	if err := k.walkClaimHistory(ctx, types.FactKeyPrefix, guard, func(key, value []byte) error {
		fact := new(types.Fact)
		if err := decodeClaimHistoryRecord(key, value, fact); err != nil {
			return err
		}
		if fact.ClaimId != "" {
			factsByClaim[fact.ClaimId] = append(factsByClaim[fact.ClaimId], fact)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	rootFacts := make(map[string]bool)
	for _, fact := range factsByClaim[id] {
		rootFacts[fact.Id] = true
	}
	root := &types.ClaimHistoryRecord{ClaimId: id}
	records := map[string]*types.ClaimHistoryRecord{id: root}
	var related []*types.RelatedClaimHistory
	if err := k.walkClaimHistory(ctx, types.ClaimKeyPrefix, guard, func(key, value []byte) error {
		claim := new(types.Claim)
		if err := decodeClaimHistoryRecord(key, value, claim); err != nil {
			return err
		}
		if err := types.ValidateReviewPolicyVersion(claim.ReviewPolicyVersion, neutral); err != nil {
			return err
		}
		if claim.ReviewPolicyVersion == types.ReviewPolicyNeutral && !recordIntegrity {
			return fmt.Errorf("neutral claim without record-integrity activation")
		}
		if err := types.ValidateClaimFundingTerms(claim, funded); err != nil {
			return err
		}
		if claim.FundingTerms != nil {
			fundedClaims[claim.Id] = claim
		}
		if claim.Id == id {
			root.Claim = claim
			return nil
		}
		links := claimHistoryLinks(claim, id, rootFacts)
		if len(links) == 0 {
			return nil
		}
		record := &types.ClaimHistoryRecord{ClaimId: claim.Id, Claim: claim}
		records[claim.Id] = record
		related = append(related, &types.RelatedClaimHistory{Record: record, Links: links})
		return nil
	}); err != nil {
		return nil, err
	}
	// Keep the full round primary set, including terminal and non-selected
	// historical rounds. The active and claim->round indexes are not an inventory.
	roundClaims := make(map[string]string)
	if err := k.walkClaimHistory(ctx, types.VerificationRoundKeyPrefix, guard, func(key, value []byte) error {
		round := new(types.VerificationRound)
		if err := decodeClaimHistoryRecord(key, value, round); err != nil {
			return err
		}
		if err := types.ValidateReviewPolicyVersion(round.ReviewPolicyVersion, neutral); err != nil {
			return err
		}
		if err := types.ValidateVerificationRoundRecord(round, recordIntegrity); err != nil {
			return err
		}
		if claim := fundedClaims[round.ClaimId]; claim != nil {
			if err := types.ValidateClaimFundingRound(claim, round); err != nil {
				return err
			}
			fundedRoundCounts[claim.Id]++
		} else if round.ClaimRefundSettlement != nil {
			return fmt.Errorf("refund record has no funded claim")
		}
		roundClaims[round.Id] = round.ClaimId
		if record := records[round.ClaimId]; record != nil {
			if record.Claim != nil && record.Claim.ReviewPolicyVersion != round.ReviewPolicyVersion {
				return fmt.Errorf("retained claim and round review policies disagree")
			}
			record.Rounds = append(record.Rounds, round)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	for id, claim := range fundedClaims {
		if fundedRoundCounts[id] != 1 || claim.VerificationRoundId == "" || roundClaims[claim.VerificationRoundId] != id {
			return nil, fmt.Errorf("funded claim lacks its sole selected round")
		}
	}
	sort.Slice(related, func(i, j int) bool { return related[i].Record.ClaimId < related[j].Record.ClaimId })
	ordered := []*types.ClaimHistoryRecord{root}
	for _, entry := range related {
		ordered = append(ordered, entry.Record)
	}
	for _, record := range ordered {
		sort.Slice(record.Rounds, func(i, j int) bool {
			a, b := record.Rounds[i], record.Rounds[j]
			if a.StartedAtBlock != b.StartedAtBlock {
				return a.StartedAtBlock < b.StartedAtBlock
			}
			return a.Id < b.Id
		})
		if record.Claim != nil && record.Claim.VerificationRoundId != "" {
			selected := record.Claim.VerificationRoundId
			owner, found := roundClaims[selected]
			if !found {
				record.MissingRoundIds = []string{selected}
			} else if owner != record.ClaimId {
				return nil, fmt.Errorf("claim references a round belonging to another claim")
			}
		}
		facts := factsByClaim[record.ClaimId]
		sort.Slice(facts, func(i, j int) bool { return facts[i].Id < facts[j].Id })
		for _, fact := range facts {
			history, err := k.claimHistoryFact(ctx, fact, guard)
			if err != nil {
				return nil, err
			}
			record.Facts = append(record.Facts, history)
		}
	}
	if guard.err != nil {
		return nil, guard.err
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	response := &types.QueryClaimHistoryResponse{
		ChainId: sdkCtx.ChainID(), BlockHeight: uint64(sdkCtx.BlockHeight()), Record: root, RelatedClaims: related,
	}
	if proto.Size(response) > ToKMaxOutputBytes {
		return nil, ErrToKResourceLimit
	}
	return response, nil
}

func claimHistoryLinks(claim *types.Claim, rootID string, rootFacts map[string]bool) []*types.ClaimHistoryLink {
	var links []*types.ClaimHistoryLink
	seen := make(map[string]bool)
	add := func(field, target string) {
		key := field + "\x00" + target
		if !seen[key] {
			links = append(links, &types.ClaimHistoryLink{Field: field, TargetId: target})
			seen[key] = true
		}
	}
	if claim.ProvisionalFactId != "" && rootFacts[claim.ProvisionalFactId] {
		add("provisional_fact_id", claim.ProvisionalFactId)
	}
	if claim.ChallengedClaimId == rootID {
		add("challenged_claim_id", rootID)
	}
	for _, relation := range claim.Relations {
		if relation.Relation == types.RelationType_RELATION_TYPE_CONTRADICTS && rootFacts[relation.TargetFactId] {
			add("relations.contradicts", relation.TargetFactId)
		}
	}
	sort.Slice(links, func(i, j int) bool {
		if links[i].Field != links[j].Field {
			return links[i].Field < links[j].Field
		}
		return links[i].TargetId < links[j].TargetId
	})
	return links
}

func (k Keeper) claimHistoryFact(ctx context.Context, fact *types.Fact, guard *tokQueryGuard) (*types.ClaimHistoryFact, error) {
	// The stored adjacency and history keys use distinct delimiters. Refuse a
	// selected identity that cannot select either exact namespace unambiguously.
	if strings.ContainsAny(fact.Id, "/\x00") {
		return nil, fmt.Errorf("ambiguous retained fact identity")
	}
	result := &types.ClaimHistoryFact{Fact: fact}
	store := k.storeService.OpenKVStore(ctx)
	for _, outgoing := range []bool{true, false} {
		prefix := types.FactRelationsBySourcePrefix(fact.Id)
		if !outgoing {
			prefix = types.FactRelationsByTargetPrefix(fact.Id)
		}
		if err := k.walkClaimHistory(ctx, prefix, guard, func(key, value []byte) error {
			relation := new(types.FactRelation)
			if err := decodeClaimHistoryRecord(key, value, relation); err != nil {
				return err
			}
			if err := types.ValidateFactRelationRecord(relation); err != nil {
				return err
			}
			if outgoing && relation.SourceFactId != fact.Id || !outgoing && relation.TargetFactId != fact.Id {
				return fmt.Errorf("relation does not belong to selected fact adjacency")
			}
			mirrorKey := types.FactRelationReverseKey(relation.TargetFactId, relation.SourceFactId)
			if !outgoing {
				mirrorKey = types.FactRelationKey(relation.SourceFactId, relation.TargetFactId)
			}
			mirror, err := store.Get(mirrorKey)
			if err != nil {
				return err
			}
			if !bytes.Equal(value, mirror) {
				return fmt.Errorf("missing or unequal mirrored relation payload")
			}
			if outgoing {
				result.OutgoingRelations = append(result.OutgoingRelations, relation)
			} else {
				result.IncomingRelations = append(result.IncomingRelations, relation)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	if err := k.walkClaimHistory(ctx, types.StatusTransitionPrefixForFact(fact.Id), guard, func(key, value []byte) error {
		transition := new(types.StatusTransition)
		if err := decodeClaimHistoryRecord(key, value, transition); err != nil {
			return err
		}
		if transition.FactId != fact.Id {
			return fmt.Errorf("status transition belongs to another fact")
		}
		result.StatusTransitions = append(result.StatusTransitions, transition)
		return nil
	}); err != nil {
		return nil, err
	}
	// Counter gaps and a counter ahead of retained history are historical data;
	// a malformed or lagging counter must not masquerade as a readable history.
	counter, err := store.Get(types.StatusTransitionSeqKey(fact.Id))
	if err != nil {
		return nil, err
	}
	var seq uint64
	if counter != nil {
		seq, err = decodeHistoryCounter(counter)
		if err != nil {
			return nil, err
		}
	}
	if n := len(result.StatusTransitions); n > 0 && seq < result.StatusTransitions[n-1].Seq {
		return nil, fmt.Errorf("status history counter missing or behind retained transitions")
	}
	return result, nil
}

func (k Keeper) walkClaimHistory(ctx context.Context, prefix []byte, guard *tokQueryGuard, visit func([]byte, []byte) error) (err error) {
	it, err := k.storeService.OpenKVStore(ctx).Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, it.Close(), guard.err) }()
	for ; it.Valid(); it.Next() {
		if err := visit(it.Key(), it.Value()); err != nil {
			return err
		}
	}
	return historyIteratorError(it)
}

// Query-only decoding deliberately refuses protobuf normalization (duplicate
// fields, integer truncation, unknown nested fields) before selecting records.
// It does not change any runtime writer or the existing ToK query semantics.
func decodeClaimHistoryRecord(key, value []byte, record proto.Message) error {
	if err := proto.Unmarshal(value, record); err != nil {
		return err
	}
	if err := validateClaimHistoryFields(record.ProtoReflect()); err != nil {
		return err
	}
	canonical, err := marshalOpts.Marshal(record)
	if err != nil || !bytes.Equal(canonical, value) {
		return fmt.Errorf("noncanonical retained protobuf record: %v", err)
	}
	var expected []byte
	switch record := record.(type) {
	case *types.Claim:
		if record.Id == "" {
			return fmt.Errorf("empty retained claim identity")
		}
		expected = types.ClaimKey(record.Id)
	case *types.VerificationRound:
		if record.Id == "" || record.ClaimId == "" {
			return fmt.Errorf("empty retained round identity")
		}
		expected = types.RoundKey(record.Id)
	case *types.Fact:
		if record.Id == "" {
			return fmt.Errorf("empty retained fact identity")
		}
		expected = types.FactKey(record.Id)
	case *types.FactRelation:
		expected = types.FactRelationKey(record.SourceFactId, record.TargetFactId)
		if bytes.HasPrefix(key, types.FactRelationReversePrefix) {
			expected = types.FactRelationReverseKey(record.TargetFactId, record.SourceFactId)
		}
	case *types.StatusTransition:
		if record.FactId == "" || record.Seq == 0 {
			return fmt.Errorf("empty retained status history identity")
		}
		expected = types.StatusTransitionKey(record.FactId, record.Seq)
	default:
		return fmt.Errorf("unsupported claim-history record type")
	}
	if !bytes.Equal(key, expected) {
		return fmt.Errorf("retained record key/payload mismatch")
	}
	return nil
}

func validateClaimHistoryFields(message protoreflect.Message) error {
	if len(message.GetUnknown()) != 0 {
		return fmt.Errorf("unknown retained protobuf fields")
	}
	var err error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		check := func(descriptor protoreflect.FieldDescriptor, v protoreflect.Value) error {
			if descriptor.Kind() == protoreflect.EnumKind && descriptor.Enum().Values().ByNumber(v.Enum()) == nil {
				return fmt.Errorf("unknown retained enum value")
			}
			if descriptor.Message() != nil {
				return validateClaimHistoryFields(v.Message())
			}
			return nil
		}
		switch {
		case field.IsMap():
			value.Map().Range(func(_ protoreflect.MapKey, v protoreflect.Value) bool {
				err = check(field.MapValue(), v)
				return err == nil
			})
		case field.IsList():
			for i := 0; i < value.List().Len() && err == nil; i++ {
				err = check(field, value.List().Get(i))
			}
		default:
			err = check(field, value)
		}
		return err == nil
	})
	return err
}
