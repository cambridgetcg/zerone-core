package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
)

// scanDurableRecords uses canonical namespaces, never lossy legacy iterators.
// Callbacks only read; iterators are closed before callers mutate state.
func (k Keeper) scanDurableRecords(ctx context.Context, prefix []byte, f func([]byte, []byte) error) error {
	it, err := k.storeService.OpenKVStore(ctx).Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return err
	}
	for ; it.Valid(); it.Next() {
		if err = f(it.Key(), it.Value()); err != nil {
			break
		}
	}
	return errors.Join(err, feedbackIteratorError(it), it.Close())
}

func validateDurableGenesis(gs *types.GenesisState) error {
	bounded := func(s string, max int) bool { return len(s) <= max && utf8.ValidString(s) }
	status := func(s types.FactStatus) bool { _, ok := types.FactStatus_name[int32(s)]; return ok }
	rounds := make(map[string]bool)
	for _, group := range [][]*types.VerificationRound{gs.ActiveRounds, gs.CompletedRounds} {
		for _, r := range group {
			if r == nil || !bounded(r.Id, 128) || r.Id == "" || !bounded(r.ClaimId, 128) {
				return fmt.Errorf("invalid durable round identifiers")
			}
			if r.ClaimId != "" && rounds[r.ClaimId] {
				return fmt.Errorf("conflicting claim-round index")
			}
			rounds[r.ClaimId] = r.ClaimId != ""
		}
	}
	for _, r := range gs.ActiveRounds {
		if r.Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE || r.Phase == types.VerificationPhase_VERIFICATION_PHASE_EXPIRED {
			return fmt.Errorf("active round is terminal")
		}
	}
	for _, r := range gs.FactRelations {
		_, relationOK := types.RelationType_name[int32(r.Relation)]
		_, inferenceOK := types.InferenceType_name[int32(r.Inference)]
		if !relationOK || !inferenceOK || r.InferenceStrengthBps > 1_000_000 || r.CreatedAtBlock > math.MaxInt64 || !bounded(r.Creator, 128) || !bounded(r.MethodId, 4096) {
			return fmt.Errorf("invalid relation metadata")
		}
	}
	for _, t := range gs.StatusTransitions {
		if !status(t.PriorStatus) || !status(t.NewStatus) || t.PriorStatus == t.NewStatus || t.BlockHeight > math.MaxInt64 || !bounded(t.CauseEventType, 4096) || !bounded(t.CauseId, 128) {
			return fmt.Errorf("invalid status transition fields")
		}
	}
	for _, e := range gs.CascadeEvents {
		if !status(e.PriorStatus) || !status(e.NewStatus) || e.BlockHeight > math.MaxInt64 || !bounded(e.ChallengeClaimId, 128) || !bounded(e.EdgeRelation, 128) {
			return fmt.Errorf("invalid cascade fields")
		}
	}
	for _, r := range gs.CompletedRoundRecords {
		if r.VerdictBlock > math.MaxInt64 || !bounded(r.RoundId, 128) || !bounded(r.Meta.Domain, 128) {
			return fmt.Errorf("invalid completion record")
		}
	}
	return nil
}

func (k Keeper) initDurableGenesis(ctx context.Context, gs *types.GenesisState) error {
	store := k.storeService.OpenKVStore(ctx)
	put := func(key []byte, m proto.Message) error {
		bz, err := marshalOpts.Marshal(m)
		if err != nil {
			return err
		}
		return store.Set(key, bz)
	}
	for _, r := range gs.FactRelations {
		if err := k.SetFactRelation(ctx, r); err != nil {
			return err
		}
	}
	for _, t := range gs.StatusTransitions {
		if err := put(types.StatusTransitionKey(t.FactId, t.Seq), t); err != nil {
			return err
		}
	}
	for _, s := range gs.StatusTransitionSequences {
		if err := store.Set(types.StatusTransitionSeqKey(s.FactId), binary.AppendUvarint(nil, s.LastSequence)); err != nil {
			return err
		}
	}
	for _, e := range gs.CascadeEvents {
		if err := put(types.CascadeEventKey(e.DisprovenFactId, e.Seq), e); err != nil {
			return err
		}
		if err := store.Set(types.CascadeEventByDescendantKey(e.DescendantFactId, e.DisprovenFactId), []byte{1}); err != nil {
			return err
		}
	}
	for _, r := range gs.CompletedRounds {
		if err := k.setFeedbackRound(ctx, r); err != nil {
			return err
		}
	}
	// Do not compute duration/domain/dissent for old rounds without metadata.
	for _, r := range gs.CompletedRoundRecords {
		if err := k.IndexCompletedRound(ctx, r.VerdictBlock, r.RoundId, r.Meta); err != nil {
			return err
		}
	}
	return k.InitFactUseReceipts(ctx, gs.FactUseReceipts, gs.FactUsePruning)
}

func (k Keeper) exportDurableGenesis(ctx context.Context, gs *types.GenesisState) error {
	// Replace legacy best-effort projections for the canonical records covered
	// by this durability boundary. Absence stays absence, including metadata.
	gs.Facts = nil
	gs.PendingClaims = nil
	gs.ActiveRounds = nil
	scans := []struct {
		prefix []byte
		read   func([]byte, []byte) error
	}{
		{types.FactKeyPrefix, func(key, bz []byte) error {
			r := new(types.Fact)
			if err := proto.Unmarshal(bz, r); err != nil {
				return err
			}
			if !bytes.Equal(key, types.FactKey(r.Id)) {
				return fmt.Errorf("fact key mismatch")
			}
			gs.Facts = append(gs.Facts, r)
			return nil
		}},
		{types.ClaimKeyPrefix, func(key, bz []byte) error {
			r := new(types.Claim)
			if err := proto.Unmarshal(bz, r); err != nil {
				return err
			}
			if !bytes.Equal(key, types.ClaimKey(r.Id)) {
				return fmt.Errorf("claim key mismatch")
			}
			gs.PendingClaims = append(gs.PendingClaims, r)
			return nil
		}},
		{types.FactRelationPrefix, func(key, bz []byte) error {
			r := new(types.FactRelation)
			if err := proto.Unmarshal(bz, r); err != nil {
				return err
			}
			if !bytes.Equal(key, types.FactRelationKey(r.SourceFactId, r.TargetFactId)) {
				return fmt.Errorf("relation key mismatch")
			}
			gs.FactRelations = append(gs.FactRelations, r)
			return nil
		}},
		{types.StatusTransitionKeyPrefix, func(key, bz []byte) error {
			r := new(types.StatusTransition)
			if err := proto.Unmarshal(bz, r); err != nil {
				return err
			}
			if !bytes.Equal(key, types.StatusTransitionKey(r.FactId, r.Seq)) {
				return fmt.Errorf("transition key mismatch")
			}
			gs.StatusTransitions = append(gs.StatusTransitions, r)
			return nil
		}},
		{types.StatusTransitionSeqKeyPrefix, func(key, bz []byte) error {
			seq, err := decodeStatusSequence(bz)
			if err != nil {
				return err
			}
			gs.StatusTransitionSequences = append(gs.StatusTransitionSequences, &types.StatusTransitionSequence{FactId: string(key[1:]), LastSequence: seq})
			return nil
		}},
		{types.CascadeEventKeyPrefix, func(key, bz []byte) error {
			r := new(types.CascadeEvent)
			if err := proto.Unmarshal(bz, r); err != nil {
				return err
			}
			if !bytes.Equal(key, types.CascadeEventKey(r.DisprovenFactId, r.Seq)) {
				return fmt.Errorf("cascade key mismatch")
			}
			gs.CascadeEvents = append(gs.CascadeEvents, r)
			return nil
		}},
		{types.VerificationRoundKeyPrefix, func(key, bz []byte) error {
			r := new(types.VerificationRound)
			if err := proto.Unmarshal(bz, r); err != nil {
				return err
			}
			if !bytes.Equal(key, types.RoundKey(r.Id)) {
				return fmt.Errorf("round key mismatch")
			}
			if r.Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE || r.Phase == types.VerificationPhase_VERIFICATION_PHASE_EXPIRED {
				gs.CompletedRounds = append(gs.CompletedRounds, r)
			} else {
				gs.ActiveRounds = append(gs.ActiveRounds, r)
			}
			return nil
		}},
		{types.CompletedRoundIndexPrefix, func(key, bz []byte) error {
			if len(key) <= 9 {
				return fmt.Errorf("invalid completion key")
			}
			meta := new(types.CompletedRoundMeta)
			if err := proto.Unmarshal(bz, meta); err != nil {
				return err
			}
			gs.CompletedRoundRecords = append(gs.CompletedRoundRecords, &types.CompletedRoundRecord{RoundId: string(key[9:]), VerdictBlock: binary.BigEndian.Uint64(key[1:9]), Meta: meta})
			return nil
		}},
	}
	for _, scan := range scans {
		if err := k.scanDurableRecords(ctx, scan.prefix, scan.read); err != nil {
			return fmt.Errorf("export knowledge prefix %x: %w", scan.prefix, err)
		}
	}
	var err error
	gs.FactUseReceipts, err = k.ExportFactUseReceipts(ctx)
	if err != nil {
		return err
	}
	gs.FactUsePruning, err = k.GetFactUsePruningState(ctx)
	if err != nil {
		return err
	}
	if err := gs.Validate(); err != nil {
		return err
	}
	return validateDurableGenesis(gs)
}
