package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	corestore "cosmossdk.io/core/store"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// These are query resource ceilings, not changes to any historical graph root.
// Exceeding one refuses the entire response; no partial graph is returned.
const (
	ToKMaxNodes       = 8192
	ToKMaxEdges       = 32768
	ToKMaxReadEntries = 65536
	ToKMaxReadBytes   = 16 * 1024 * 1024
	ToKMaxOutputBytes = 8 * 1024 * 1024
)

var ErrToKResourceLimit = errors.New("ToK query resource limit exceeded")

type tokQueryGuard struct {
	entries, bytes int
	err            error
}

func (g *tokQueryGuard) fail(err error) error {
	if err != nil && g.err == nil {
		g.err = err
	}
	return g.err
}

func (g *tokQueryGuard) read(key, value []byte) error {
	if g.err != nil {
		return g.err
	}
	g.entries++
	g.bytes += len(key) + len(value)
	if g.entries > ToKMaxReadEntries || g.bytes > ToKMaxReadBytes {
		return g.fail(ErrToKResourceLimit)
	}
	if value == nil {
		return nil
	}
	if err := validateToKStoredRecord(key, value); err != nil {
		return g.fail(fmt.Errorf("%w: %v", ErrToKInconsistentState, err))
	}
	return nil
}

// Guard all reads, including helper functions with historical bool/slice APIs.
// Their swallowed storage errors remain latched and refuse the outer query.
func (k Keeper) guardedToK() (Keeper, *tokQueryGuard) {
	if s, ok := k.storeService.(tokGuardService); ok {
		return k, s.guard
	}
	g := &tokQueryGuard{}
	k.storeService = tokGuardService{k.storeService, g}
	return k, g
}

type tokGuardService struct {
	corestore.KVStoreService
	guard *tokQueryGuard
}

func (s tokGuardService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return tokGuardStore{s.KVStoreService.OpenKVStore(ctx), s.guard}
}

type tokGuardStore struct {
	corestore.KVStore
	guard *tokQueryGuard
}

func (s tokGuardStore) Get(key []byte) ([]byte, error) {
	if s.guard.err != nil {
		return nil, s.guard.err
	}
	v, err := s.KVStore.Get(key)
	if err != nil {
		return nil, s.guard.fail(fmt.Errorf("%w: %v", ErrToKInconsistentState, err))
	}
	if err = s.guard.read(key, v); err != nil {
		return nil, err
	}
	return v, nil
}
func (s tokGuardStore) Has(key []byte) (bool, error) { v, err := s.Get(key); return v != nil, err }
func (s tokGuardStore) Set(_, _ []byte) error {
	return s.guard.fail(fmt.Errorf("%w: query attempted a write", ErrToKInconsistentState))
}
func (s tokGuardStore) Delete(_ []byte) error {
	return s.guard.fail(fmt.Errorf("%w: query attempted a delete", ErrToKInconsistentState))
}
func (s tokGuardStore) Iterator(start, end []byte) (corestore.Iterator, error) {
	return s.iterator(start, end, false)
}
func (s tokGuardStore) ReverseIterator(start, end []byte) (corestore.Iterator, error) {
	return s.iterator(start, end, true)
}
func (s tokGuardStore) iterator(start, end []byte, reverse bool) (corestore.Iterator, error) {
	if s.guard.err != nil {
		return nil, s.guard.err
	}
	var it corestore.Iterator
	var err error
	if reverse {
		it, err = s.KVStore.ReverseIterator(start, end)
	} else {
		it, err = s.KVStore.Iterator(start, end)
	}
	if err != nil {
		return nil, s.guard.fail(fmt.Errorf("%w: %v", ErrToKInconsistentState, err))
	}
	return &tokGuardIterator{Iterator: it, guard: s.guard}, nil
}

type tokGuardIterator struct {
	corestore.Iterator
	guard    *tokQueryGuard
	examined bool
}

func (it *tokGuardIterator) Valid() bool {
	if it.guard.err != nil {
		return false
	}
	if !it.Iterator.Valid() {
		if err := historyIteratorError(it.Iterator); err != nil {
			it.guard.fail(fmt.Errorf("%w: %v", ErrToKInconsistentState, err))
		}
		return false
	}
	if !it.examined {
		it.examined = true
		if it.guard.read(it.Iterator.Key(), it.Iterator.Value()) != nil {
			return false
		}
	}
	return true
}
func (it *tokGuardIterator) Next() { it.Iterator.Next(); it.examined = false }
func (it *tokGuardIterator) Close() error {
	err := it.Iterator.Close()
	if err != nil {
		it.guard.fail(fmt.Errorf("%w: %v", ErrToKInconsistentState, err))
	}
	return err
}

func validateToKStoredRecord(key, value []byte) error {
	if len(key) == 0 {
		return fmt.Errorf("empty state key")
	}
	var msg proto.Message
	var expected []byte
	switch key[0] {
	case types.FactKeyPrefix[0]:
		v := &types.Fact{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		if v.Id == "" {
			return fmt.Errorf("empty fact id")
		}
		if _, ok := types.FactStatus_name[int32(v.Status)]; !ok {
			return fmt.Errorf("unknown fact status")
		}
		msg = v
		expected = types.FactKey(v.Id)
	case types.ClaimKeyPrefix[0]:
		if err := validateRawClaimRecord(value); err != nil {
			return err
		}
		v := &types.Claim{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.ClaimKey(v.Id)
	case types.VerificationRoundKeyPrefix[0]:
		if err := validateRawRoundRecord(value); err != nil {
			return err
		}
		v := &types.VerificationRound{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.RoundKey(v.Id)
	case types.ParamsKey[0]:
		v := &types.Params{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.ParamsKey
	case types.TokenizerSpecKey[0]:
		v := &types.TokenizerSpec{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.TokenizerSpecKey
	case types.TraceSchemaKey[0]:
		v := &types.TraceSchema{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.TraceSchemaKey
	case types.MethodologyKeyPrefix[0]:
		v := &types.Methodology{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.MethodologyKey(v.Id)
	case types.NormativeCommitmentKeyPrefix[0]:
		v := &types.NormativeCommitment{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.NormativeCommitmentKey(v.Id)
	case types.AugmentationKeyPrefix[0]:
		v := &types.Augmentation{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.AugmentationKey(v.Id)
	case types.AugmentationBountyKeyPrefix[0]:
		v := &types.AugmentationBounty{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		msg = v
		expected = types.AugmentationBountyKey(v.Id)
	case types.VindicationRecordPrefix[0]:
		var v types.VindicationRecord
		if err := decodeToKVindicationRecord(value, &v); err != nil {
			return err
		}
		if v.FactId == "" || v.Verifier == "" || !bytes.Equal(key, types.VindicationRecordKey(v.FactId, v.Verifier)) {
			return fmt.Errorf("vindication key/payload mismatch")
		}
	case types.FactRelationPrefix[0], types.FactRelationReversePrefix[0]:
		v := &types.FactRelation{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		if v.SourceFactId == "" || v.TargetFactId == "" {
			return fmt.Errorf("empty relation identity")
		}
		msg = v
		if key[0] == types.FactRelationPrefix[0] {
			expected = types.FactRelationKey(v.SourceFactId, v.TargetFactId)
		} else {
			expected = types.FactRelationReverseKey(v.TargetFactId, v.SourceFactId)
		}
	case types.StatusTransitionKeyPrefix[0]:
		v := &types.StatusTransition{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		if v.FactId == "" || v.Seq == 0 {
			return fmt.Errorf("invalid transition identity")
		}
		if err := validateHistoryStatuses(v.PriorStatus, v.NewStatus); err != nil {
			return err
		}
		msg = v
		expected = types.StatusTransitionKey(v.FactId, v.Seq)
	case types.CascadeEventKeyPrefix[0]:
		v := &types.CascadeEvent{}
		if err := proto.Unmarshal(value, v); err != nil {
			return err
		}
		if v.DisprovenFactId == "" || v.DescendantFactId == "" || v.Seq == 0 {
			return fmt.Errorf("invalid cascade identity")
		}
		if err := validateHistoryStatuses(v.PriorStatus, v.NewStatus); err != nil {
			return err
		}
		msg = v
		expected = types.CascadeEventKey(v.DisprovenFactId, v.Seq)
	}
	if msg != nil && hasUnknownRecordFields(msg.ProtoReflect()) {
		return fmt.Errorf("unknown protobuf fields at %x", key)
	}
	if msg != nil && !bytes.Equal(key, expected) {
		return fmt.Errorf("record key/payload mismatch at %x", key)
	}
	return nil
}

// cachekv reports these two sentinels after normal exhaustion. Every other
// iterator failure and every Close failure remains an error.
func historyIteratorError(it corestore.Iterator) error {
	err := it.Error()
	if !it.Valid() && err != nil && (err.Error() == "invalid cacheMergeIterator" || err.Error() == "invalid memIterator") {
		return nil
	}
	return err
}

func validateHistoryStatuses(prior, next types.FactStatus) error {
	if _, ok := types.FactStatus_name[int32(prior)]; !ok {
		return fmt.Errorf("unknown prior status")
	}
	if _, ok := types.FactStatus_name[int32(next)]; !ok {
		return fmt.Errorf("unknown new status")
	}
	return nil
}

// Vindication rows are flat legacy JSON. Reject duplicate or aliased fields so
// a swallowed decoder failure cannot become a plausible incomplete bundle.
func decodeToKVindicationRecord(raw []byte, out *types.VindicationRecord) error {
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid vindication UTF-8")
	}
	allowed := map[string]bool{"verifier": true, "fact_id": true, "refund_amount": true, "bonus_amount": true, "vindicated_at": true, "disproven_by": true, "round_id": true}
	seen := map[string]bool{}
	d := json.NewDecoder(bytes.NewReader(raw))
	tok, err := d.Token()
	if err != nil || tok != json.Delim('{') {
		return fmt.Errorf("invalid vindication JSON object")
	}
	for d.More() {
		tok, err = d.Token()
		if err != nil {
			return err
		}
		key, ok := tok.(string)
		if !ok || !allowed[key] || seen[key] {
			return fmt.Errorf("invalid or duplicate vindication field")
		}
		seen[key] = true
		var value json.RawMessage
		if err = d.Decode(&value); err != nil {
			return err
		}
	}
	if tok, err = d.Token(); err != nil || tok != json.Delim('}') {
		return fmt.Errorf("invalid vindication object end")
	}
	if _, err = d.Token(); err != io.EOF {
		return fmt.Errorf("trailing vindication JSON")
	}
	return json.Unmarshal(raw, out)
}
