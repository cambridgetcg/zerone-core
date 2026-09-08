package keeper

// Read-only adapters live beside the selectors, not consensus storage helpers:
// callers share one request budget and never hydrate persisted Fact records.
import (
	"bytes"
	"context"
	"encoding/json"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

const (
	ToKReadMaxNodes    = 128
	ToKReadMaxEdges    = 512
	ToKReadMaxExamined = 1024
	ToKReadMaxBytes    = 256 * 1024
)

type tokReadBudgetKey struct{}
type tokReadBudget struct {
	examined, materialized, edges int
	err                           error
	facts                         map[string]*types.Fact
}

func withToKReadBudget(ctx context.Context) context.Context {
	if ctx.Value(tokReadBudgetKey{}) != nil {
		return ctx
	}
	return context.WithValue(ctx, tokReadBudgetKey{}, &tokReadBudget{facts: map[string]*types.Fact{}})
}
func readBudget(ctx context.Context) *tokReadBudget {
	return ctx.Value(tokReadBudgetKey{}).(*tokReadBudget)
}
func (b *tokReadBudget) check(ctx context.Context) error {
	if b.err != nil {
		return b.err
	}
	if err := ctx.Err(); err != nil {
		b.err = status.FromContextError(err).Err()
	}
	return b.err
}
func (b *tokReadBudget) exhaust(label string) error {
	b.err = status.Errorf(codes.ResourceExhausted, "knowledge read exceeds %s budget", label)
	return b.err
}
func (b *tokReadBudget) examine(ctx context.Context) error {
	if err := b.check(ctx); err != nil {
		return err
	}
	if b.examined >= ToKReadMaxExamined {
		return b.exhaust("1024 examined records")
	}
	b.examined++
	return nil
}
func (b *tokReadBudget) materialize(n int) error {
	if n > ToKReadMaxBytes-b.materialized {
		return b.exhaust("256 KiB materialized records")
	}
	b.materialized += n
	return nil
}
func (b *tokReadBudget) edge() error {
	if b.edges >= ToKReadMaxEdges {
		return b.exhaust("512 returned edges")
	}
	b.edges++
	return nil
}
func readNode(ctx context.Context, visited map[string]bool, id string) error {
	if visited[id] {
		return nil
	}
	b := readBudget(ctx)
	if err := b.check(ctx); err != nil {
		return err
	}
	if len(visited) >= ToKReadMaxNodes {
		return b.exhaust("128 nodes")
	}
	visited[id] = true
	return nil
}
func readEdge(ctx context.Context, edges map[string]*types.ToKEdge, rel *types.FactRelation) error {
	key := rel.SourceFactId + "->" + rel.TargetFactId + "|" + rel.Relation.String()
	if _, ok := edges[key]; ok {
		return nil
	}
	if err := readBudget(ctx).edge(); err != nil {
		return err
	}
	edges[key] = &types.ToKEdge{FromFactId: rel.SourceFactId, ToFactId: rel.TargetFactId, Relation: rel.Relation.String(), Inference: rel.Inference.String()}
	return nil
}

// scanReadPrefix charges BEFORE accessing the value or decoding it. Errors are
// never converted to empty history. The callback may stop only for selector
// semantics (e.g. an explicit frontier limit), never a resource limit.
func (k Keeper) scanReadPrefix(ctx context.Context, pfx []byte, visit func([]byte, []byte) (bool, error)) (retErr error) {
	b := readBudget(ctx)
	if err := b.check(ctx); err != nil {
		return err
	}
	iter, err := k.storeService.OpenKVStore(ctx).Iterator(pfx, prefixEndBytes(pfx))
	if err != nil {
		return status.Errorf(codes.Internal, "knowledge iterator: %v", err)
	}
	defer func() {
		// Preserve an existing cancellation/budget/error status, but never turn
		// an exposed close failure into a successful (possibly partial) read.
		if err := iter.Close(); err != nil && retErr == nil {
			retErr = status.Errorf(codes.Internal, "knowledge iterator close: %v", err)
		}
	}()
	for ; iter.Valid(); iter.Next() {
		if err := b.examine(ctx); err != nil {
			return err
		}
		value := iter.Value()
		if err := b.materialize(len(value)); err != nil {
			return err
		}
		stop, err := visit(iter.Key(), value)
		if err != nil {
			return err
		}
		if stop {
			return nil
		}
	}
	if err := feedbackIteratorError(iter); err != nil {
		return status.Errorf(codes.Internal, "knowledge iterator: %v", err)
	}
	return b.check(ctx)
}

// readFact retains GetFact's caller shape but latches storage/cancellation errors;
// entrypoints check the latch before returning a not-found or a successful read.
func (k Keeper) readFact(ctx context.Context, id string) (*types.Fact, bool) {
	b := readBudget(ctx)
	if b.check(ctx) != nil {
		return nil, false
	}
	if f, ok := b.facts[id]; ok {
		return f, f != nil
	}
	if b.examine(ctx) != nil {
		return nil, false
	}
	value, err := k.storeService.OpenKVStore(ctx).Get(types.FactKey(id))
	if err != nil {
		b.err = status.Errorf(codes.Internal, "knowledge fact read: %v", err)
		return nil, false
	}
	if value == nil {
		b.facts[id] = nil
		return nil, false
	}
	if b.materialize(len(value)) != nil {
		return nil, false
	}
	f, err := decodeFact(value)
	if err != nil {
		b.err = err
		return nil, false
	}
	if f.Id != id {
		b.err = status.Error(codes.Internal, "fact index mismatch")
		return nil, false
	}
	b.facts[id] = f
	return f, true
}

func (k Keeper) readRelations(ctx context.Context, id string, incoming bool) ([]*types.FactRelation, error) {
	pfx := types.FactRelationsBySourcePrefix(id)
	if incoming {
		pfx = types.FactRelationsByTargetPrefix(id)
	}
	var out []*types.FactRelation
	err := k.scanReadPrefix(ctx, pfx, func(key, value []byte) (bool, error) {
		rel := &types.FactRelation{}
		if err := proto.Unmarshal(value, rel); err != nil {
			return false, status.Error(codes.Internal, "malformed canonical relation")
		}
		expected := types.FactRelationKey(rel.SourceFactId, rel.TargetFactId)
		if incoming {
			expected = types.FactRelationReverseKey(rel.TargetFactId, rel.SourceFactId)
		}
		if !bytes.Equal(expected, key) {
			return false, status.Error(codes.Internal, "canonical relation index mismatch")
		}
		out = append(out, rel)
		return false, nil
	})
	return out, err
}
func (k Keeper) hydrateReadFact(ctx context.Context, fact *types.Fact) (*types.Fact, error) {
	out := proto.Clone(fact).(*types.Fact)
	var err error
	out.OutgoingRelations, err = k.readRelations(ctx, fact.Id, false)
	if err != nil {
		return nil, err
	}
	out.IncomingRelations, err = k.readRelations(ctx, fact.Id, true)
	if err != nil {
		return nil, err
	}
	for range out.OutgoingRelations {
		if err := readBudget(ctx).edge(); err != nil {
			return nil, err
		}
	}
	for range out.IncomingRelations {
		if err := readBudget(ctx).edge(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Count embedded legacy Fact relation arrays too: their payload is preserved in
// ToK bundles, but they cannot bypass the returned-edge ceiling.
func checkReadShape(ctx context.Context, message protoreflect.Message, nodes, edges *int) error {
	switch message.Interface().(type) {
	case *types.Fact:
		*nodes++
	case *types.FactRelation, *types.ToKEdge:
		*edges++
	}
	if *nodes > ToKReadMaxNodes {
		return readBudget(ctx).exhaust("128 returned nodes")
	}
	if *edges > ToKReadMaxEdges {
		return readBudget(ctx).exhaust("512 returned edges")
	}
	var err error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if field.Message() == nil || field.IsMap() {
			return true
		}
		if field.IsList() {
			list := value.List()
			for i := 0; i < list.Len(); i++ {
				if err = checkReadShape(ctx, list.Get(i).Message(), nodes, edges); err != nil {
					return false
				}
			}
		} else {
			err = checkReadShape(ctx, value.Message(), nodes, edges)
		}
		return err == nil
	})
	return err
}

func checkReadOutput(ctx context.Context, msg proto.Message) error {
	var nodes, edges int
	if err := checkReadShape(ctx, msg.ProtoReflect(), &nodes, &edges); err != nil {
		return err
	}
	if err := readBudget(ctx).check(ctx); err != nil {
		return err
	}
	if proto.Size(msg) > ToKReadMaxBytes {
		return readBudget(ctx).exhaust("256 KiB output")
	}
	// Include default fields: REST emits defaults. This also counts base64
	// expansion of the duplicated JSONL payload in a ToK response.
	data, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(msg)
	if err != nil {
		return status.Errorf(codes.Internal, "knowledge encoding: %v", err)
	}
	if len(data) > ToKReadMaxBytes {
		return readBudget(ctx).exhaust("256 KiB output")
	}
	return readBudget(ctx).check(ctx)
}

func (k Keeper) readStatusHistory(ctx context.Context, id string) ([]*types.StatusTransition, error) {
	var out []*types.StatusTransition
	err := k.scanReadPrefix(ctx, types.StatusTransitionPrefixForFact(id), func(key, v []byte) (bool, error) {
		record := &types.StatusTransition{}
		if err := proto.Unmarshal(v, record); err != nil {
			return false, status.Error(codes.Internal, "malformed status history")
		}
		if record.FactId != id || !bytes.Equal(key, types.StatusTransitionKey(record.FactId, record.Seq)) {
			return false, status.Error(codes.Internal, "status history index mismatch")
		}
		out = append(out, record)
		return false, nil
	})
	return out, err
}
func (k Keeper) readCascadeEvents(ctx context.Context, id string) ([]*types.CascadeEvent, error) {
	var out []*types.CascadeEvent
	err := k.scanReadPrefix(ctx, types.CascadeEventPrefixForDisproof(id), func(key, v []byte) (bool, error) {
		record := &types.CascadeEvent{}
		if err := proto.Unmarshal(v, record); err != nil {
			return false, status.Error(codes.Internal, "malformed cascade history")
		}
		if record.DisprovenFactId != id || !bytes.Equal(key, types.CascadeEventKey(record.DisprovenFactId, record.Seq)) {
			return false, status.Error(codes.Internal, "cascade history index mismatch")
		}
		out = append(out, record)
		return false, nil
	})
	return out, err
}
func (k Keeper) readVindications(ctx context.Context, id string) ([]types.VindicationRecord, error) {
	var out []types.VindicationRecord
	err := k.scanReadPrefix(ctx, types.VindicationRecordPrefixForFact(id), func(_, v []byte) (bool, error) {
		var record types.VindicationRecord
		if err := json.Unmarshal(v, &record); err != nil {
			return false, status.Error(codes.Internal, "malformed vindication history")
		}
		out = append(out, record)
		return false, nil
	})
	return out, err
}
func (k Keeper) readSupersessionChain(ctx context.Context, id string) ([]string, error) {
	var chain []string
	visited := map[string]bool{id: true}
	for depth := 0; depth < 8; depth++ {
		rels, err := k.readRelations(ctx, id, true)
		if err != nil {
			return nil, err
		}
		next := ""
		for _, rel := range rels {
			if rel.Relation == types.RelationType_RELATION_TYPE_SUPERSEDES && !visited[rel.SourceFactId] {
				next = rel.SourceFactId
				break
			}
		}
		if next == "" {
			break
		}
		chain = append(chain, next)
		visited[next] = true
		id = next
	}
	return chain, nil
}
