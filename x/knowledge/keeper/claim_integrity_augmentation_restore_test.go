package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/keeper"
	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Restoring the fact edges that equal-ID augmentations replaced before
// activation (knowledge v12; approved by Yu on 2026-10-17; spec §1.5, plan
// §8). Before 0x87 a passing augmentation verdict wrote its REFORMULATES edge
// at <augmentation ID>/<original fact>, so an augmentation of D whose ID
// equalled fact F's, where F had its own edge to D, replaced that edge
// (forward key and reverse entry). Every scenario below makes the replacement
// with the real writers and the marker off, then runs Migrate11to12.

const (
	restoredEvent     = "zerone.knowledge.augmentation_edge_restored"
	unrestorableEvent = "zerone.knowledge.augmentation_edge_unrestorable"
)

// augEdge is the entry a passing verdict writes, marker off.
func augEdge(t *testing.T, augID, original string, verdictBlock uint64) []byte {
	t.Helper()
	bz, err := proto.MarshalOptions{Deterministic: true}.Marshal(&types.FactRelation{
		SourceFactId: augID, TargetFactId: original, Relation: types.RelationType_RELATION_TYPE_REFORMULATES, CreatedAtBlock: verdictBlock,
	})
	require.NoError(t, err)
	return bz
}

// pair reads the forward and reverse entries of the edge source→target.
func (f *v11Fixture) pair(source, target string) (forward, reverse []byte) {
	kv := f.ctx.KVStore(f.key)
	return kv.Get(types.FactRelationKey(source, target)), kv.Get(types.FactRelationReverseKey(target, source))
}

// replaceOwnEdge snapshots F's own edge to D, then passes an augmentation of
// D whose ID is F's through the real writer, marker off, and requires that it
// replaced both entries. It returns F's own entry. The verdict runs in the
// current block, which may be F's creation block.
func (f *v11Fixture) replaceOwnEdge(t *testing.T, factF, factD string) []byte {
	t.Helper()
	require.False(t, f.claimIntegrityEnabled(t), "the replacement happens before activation")
	own, ownReverse := f.pair(factF, factD)
	require.NotEmpty(t, own)
	require.Equal(t, own, ownReverse)
	f.vdPassAugmentation(t, factF, factD)
	aug, found := f.k.GetAugmentation(f.ctx, factF)
	require.True(t, found)
	forward, reverse := f.pair(factF, factD)
	replaced := augEdge(t, factF, factD, aug.VerdictBlock)
	require.Equal(t, replaced, forward, "the verdict replaced F's own edge")
	require.Equal(t, replaced, reverse)
	require.NotEqual(t, own, replaced)
	return own
}

func (f *v11Fixture) migrate11to12(t *testing.T) {
	t.Helper()
	require.NoError(t, keeper.NewMigrator(f.k).Migrate11to12(claimIntegrityOwnerContext(t, f.ctx)))
}

// requireOnlyActivationChanged requires that the store after Migrate11to12
// differs from before only by the marker, the receipt and the given rows,
// which must hold want.
func requireOnlyActivationChanged(t *testing.T, before, after map[string][]byte, want map[string][]byte) {
	t.Helper()
	require.Equal(t, []byte{1}, after[keeper.ClaimIntegrityEnabledStoreKey])
	require.Equal(t, []byte("true"), after[claimIntegrityReceiptKey()])
	rest := make(map[string][]byte, len(after))
	for key, value := range after {
		rest[key] = value
	}
	delete(rest, keeper.ClaimIntegrityEnabledStoreKey)
	delete(rest, claimIntegrityReceiptKey())
	expected := make(map[string][]byte, len(before))
	for key, value := range before {
		expected[key] = value
	}
	for key, value := range want {
		require.Contains(t, before, key, "a restored row existed before")
		expected[key] = value
	}
	require.Equal(t, expected, rest)
}

// restoreShapes are the claim-born edges an equal-ID augmentation replaced.
var restoreShapes = []struct {
	name      string
	relations func(factD string) []*types.ClaimRelation
	cascade   bool // the restored edge is one the cascade follows
	r1        bool // the restored edge is a CONTRADICTS edge R1 reads
	sameBlock bool // the verdict runs in F's creation block, after it
}{
	{"REQUIRES", restoreRequires, true, false, false},
	{"REQUIRES, verdict in F's creation block", restoreRequires, true, false, true},
	{"CITES with inference, strength and method", restoreCitesWithMethod, false, false, false},
	{"CONTRADICTS", func(d string) []*types.ClaimRelation {
		return []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_CONTRADICTS, TargetFactId: d}}
	}, false, true, false},
	{"named twice, the last relation stored", func(d string) []*types.ClaimRelation {
		return []*types.ClaimRelation{
			{Relation: types.RelationType_RELATION_TYPE_SUPPORTS, TargetFactId: d},
			{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: d, InferenceStrengthBps: 400_000},
		}
	}, true, false, false},
}

func restoreRequires(d string) []*types.ClaimRelation {
	return []*types.ClaimRelation{{Relation: types.RelationType_RELATION_TYPE_REQUIRES, TargetFactId: d}}
}

func restoreCitesWithMethod(d string) []*types.ClaimRelation {
	return []*types.ClaimRelation{{
		Relation: types.RelationType_RELATION_TYPE_CITES, TargetFactId: d, Inference: types.InferenceType_INFERENCE_TYPE_INDUCTIVE,
		InferenceStrengthBps: 700_000, MethodId: types.MethodologyHistorical,
	}}
}

// restoreScenario accepts D, then F from a claim with the shape's relations,
// on a knowledge-11 store. It returns F's claim, F and D.
func restoreScenario(t *testing.T, tag string, relations func(string) []*types.ClaimRelation) (*v11Fixture, string, string, string) {
	t.Helper()
	f := setupV11SemanticsTest(t)
	_, factD, _ := f.contraTarget(t, "v12-rs-"+tag+"-d", "Synthetic v12 restoration ("+tag+"): fact D, which fact F names.")
	claimF := f.mustSubmitClaim(t, v11Addr("v12-rs-"+tag+"-f"), "Synthetic v12 restoration ("+tag+"): fact F, whose claim names fact D.", dedupDomain,
		func(m *types.MsgSubmitClaim) { m.Relations = relations(factD) })
	factF := f.accept(t, claimF, v11Reviewers()...)
	return f, claimF, factF, factD
}

// The edge F's creation wrote is restored byte for byte, both entries, and
// nothing else changes but the marker and the receipt. Each restored pair is
// reported. The cascade follows a restored REQUIRES edge again: before the
// upgrade a refutation of D left F VERIFIED, after it F becomes CONTESTED. R1
// reads a restored CONTRADICTS edge again: with the marker on but the edge
// left replaced, D (CONTESTED by F's knowledge-11 admission) left CONTESTED
// for ACTIVE when it survived a challenge; after the upgrade F holds it.
func TestMigrate11to12RestoresEdgesReplacedByEqualIDAugmentations(t *testing.T) {
	for i, shape := range restoreShapes {
		t.Run(shape.name, func(t *testing.T) {
			tag := string(rune('a' + i))
			f, _, factF, factD := restoreScenario(t, tag, shape.relations)
			if !shape.sameBlock {
				f.nextBlock(t)
			}
			own := f.replaceOwnEdge(t, factF, factD)
			aug, found := f.k.GetAugmentation(f.ctx, factF)
			require.True(t, found)
			require.Equal(t, shape.sameBlock, aug.VerdictBlock == f.fact(t, factF).VerifiedAtBlock)
			if shape.cascade {
				f.inCache(func() {
					refute := f.challenge(t, v11Addr("v12-rs-"+tag+"-pre"), factD)
					f.decide(t, refute, "accept", v11Reviewers()...)
					require.Equal(t, csDisproven, f.fact(t, factD).Status)
					require.Equal(t, csVerified, f.fact(t, factF).Status, "the residual: the cascade misses F")
				})
			}
			if shape.r1 {
				require.Equal(t, csContested, f.fact(t, factD).Status, "knowledge 11 contested D when F's claim was admitted")
				f.inCache(func() {
					require.NoError(t, f.k.EnableClaimIntegrity(f.ctx))
					survived := f.challenge(t, v11Addr("v12-rs-"+tag+"-pre"), factD)
					f.decide(t, survived, "reject", v11Reviewers()...)
					require.Equal(t, vdActive, f.fact(t, factD).Status, "the residual: R1 misses F")
				})
			}
			before := relationContractSnapshot(t, f.ctx, f.key)
			f.freshEvents()
			f.migrate11to12(t)

			forward, reverse := f.pair(factF, factD)
			require.Equal(t, own, forward, "F's own edge, exactly as its creation wrote it")
			require.Equal(t, own, reverse)
			requireOnlyActivationChanged(t, before, relationContractSnapshot(t, f.ctx, f.key), map[string][]byte{
				string(types.FactRelationKey(factF, factD)):        own,
				string(types.FactRelationReverseKey(factD, factF)): own,
			})
			require.Equal(t, []map[string]string{{"fact_id": factF, "target_id": factD, "augmentation_id": factF}}, v11Event(f.ctx, restoredEvent))
			require.Empty(t, v11Event(f.ctx, unrestorableEvent))

			if shape.cascade {
				refute := f.challenge(t, v11Addr("v12-rs-"+tag+"-post"), factD)
				f.decide(t, refute, "accept", v11Reviewers()...)
				require.Equal(t, csDisproven, f.fact(t, factD).Status)
				require.Equal(t, csContested, f.fact(t, factF).Status, "the cascade follows the restored edge")
			}
			if shape.r1 {
				survived := f.challenge(t, v11Addr("v12-rs-"+tag+"-post"), factD)
				f.decide(t, survived, "reject", v11Reviewers()...)
				require.Equal(t, csContested, f.fact(t, factD).Status, "R1 reads the restored CONTRADICTS edge: F still stands")
				require.Empty(t, v11Event(f.ctx, csIncompleteEvent))
			}
		})
	}
}

// Nothing is restored, and nothing reported, where no fact's own edge was
// replaced: an ordinary augmentation ID, an equal ID whose target F's claim
// does not name (a stray edge, §6.6), a failing verdict (no edge written), a
// fact created after the verdict whose own write came last, and a fact with
// no claim created after the verdict.
func TestMigrate11to12RestoresNothingElse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		build func(t *testing.T, f *v11Fixture)
	}{
		{"ordinary augmentation ID", func(t *testing.T, f *v11Fixture) {
			_, _, factD := f.vdDependent(t, "n1")
			f.vdPassAugmentation(t, "v12-rs-ordinary-variant", factD)
			forward, _ := f.pair("v12-rs-ordinary-variant", factD)
			require.NotEmpty(t, forward)
		}},
		{"equal ID, a target F's claim does not name", func(t *testing.T, f *v11Fixture) {
			_, factF, factD := f.vdDependent(t, "n2")
			_, factE, _ := f.contraTarget(t, "v12-rs-n2-e", "Synthetic v12 restoration (n2): fact E, which F's claim does not name.")
			own, _ := f.pair(factF, factD)
			f.vdPassAugmentation(t, factF, factE)
			stray, _ := f.pair(factF, factE)
			require.NotEmpty(t, stray, "a stray REFORMULATES edge under F's prefix")
			after, _ := f.pair(factF, factD)
			require.Equal(t, own, after)
		}},
		{"failing verdict", func(t *testing.T, f *v11Fixture) {
			_, factF, factD := f.vdDependent(t, "n3")
			own, _ := f.pair(factF, factD)
			require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{Id: factF, OriginalFactId: factD, VariantContent: "Synthetic v12 restoration: a drifting variant."}))
			require.NoError(t, f.k.ApplyFinalizedAugmentationVerdict(f.ctx, factF, types.AugmentationVerdict_AUGMENTATION_VERDICT_DRIFT))
			after, _ := f.pair(factF, factD)
			require.Equal(t, own, after, "a failing verdict writes no edge")
		}},
		{"fact created after the verdict, its own write last", func(t *testing.T, f *v11Fixture) {
			_, factD, _ := f.contraTarget(t, "v12-rs-n4-d", "Synthetic v12 restoration (n4): fact D, which fact F names.")
			claimF := f.mustSubmitClaim(t, v11Addr("v12-rs-n4-f"), "Synthetic v12 restoration (n4): fact F, created after the verdict.", dedupDomain,
				func(m *types.MsgSubmitClaim) { m.Relations = restoreRequires(factD) })
			future := keeper.GenerateFactID(claimF, f.roundOf(t, claimF).CommitDeadline+1)
			f.vdPassAugmentation(t, future, factD)
			require.Equal(t, future, f.accept(t, claimF, v11Reviewers()...))
			forward, reverse := f.pair(future, factD)
			var edge types.FactRelation
			require.NoError(t, proto.Unmarshal(forward, &edge))
			require.Equal(t, types.RelationType_RELATION_TYPE_REQUIRES, edge.Relation, "F's creation wrote its own edge last")
			require.Equal(t, forward, reverse)
		}},
		{"fact with no claim created after the verdict", func(t *testing.T, f *v11Fixture) {
			_, _, factD := f.vdDependent(t, "n5")
			const late = "v12-rs-late-genesis-fact"
			f.vdPassAugmentation(t, late, factD)
			aug, found := f.k.GetAugmentation(f.ctx, late)
			require.True(t, found)
			require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{
				Id: late, Content: "Synthetic v12 restoration: a claimless fact created after the verdict.", Domain: dedupDomain,
				Status: types.FactStatus_FACT_STATUS_VERIFIED, VerifiedAtBlock: aug.VerdictBlock + 1,
			}))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setupV11SemanticsTest(t)
			tc.build(t, f)
			before := relationContractSnapshot(t, f.ctx, f.key)
			f.freshEvents()
			f.migrate11to12(t)
			requireOnlyActivationChanged(t, before, relationContractSnapshot(t, f.ctx, f.key), nil)
			require.Empty(t, v11Event(f.ctx, restoredEvent))
			require.Empty(t, v11Event(f.ctx, unrestorableEvent))
		})
	}
}

// A replaced edge that cannot be rebuilt exactly is left as it is and
// reported: F has no claim (genesis, doctrine or an authority injection), its
// claim is missing, its ID does not derive from its claim and verification
// height, the reverse entry was rewritten after the verdict, or its relation
// carries a method_id that its round cannot place.
func TestMigrate11to12ReportsUnrestorableEdges(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		build        func(t *testing.T, f *v11Fixture) (factF, factD string)
	}{
		{"fact with no claim", "fact has no claim", func(t *testing.T, f *v11Fixture) (string, string) {
			_, factD, _ := f.contraTarget(t, "v12-rs-u1-d", "Synthetic v12 restoration (u1): fact D, which a genesis fact requires.")
			const genesisF = "v12-rs-genesis-fact"
			require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{
				Id: genesisF, Content: "Synthetic v12 restoration: a fact from a genesis file, with no claim.", Domain: dedupDomain,
				Status: types.FactStatus_FACT_STATUS_VERIFIED,
			}))
			require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{
				SourceFactId: genesisF, TargetFactId: factD, Relation: types.RelationType_RELATION_TYPE_REQUIRES,
			}))
			f.nextBlock(t)
			f.replaceOwnEdge(t, genesisF, factD)
			return genesisF, factD
		}},
		{"claim missing", "", func(t *testing.T, f *v11Fixture) (string, string) {
			claimF, factF, factD := f.vdDependent(t, "u2")
			f.nextBlock(t)
			f.replaceOwnEdge(t, factF, factD)
			f.ctx.KVStore(f.key).Delete(types.ClaimKey(claimF))
			return factF, factD
		}},
		{"fact ID does not derive from its claim", "fact id does not derive from its claim and verification height", func(t *testing.T, f *v11Fixture) (string, string) {
			_, factF, factD := f.vdDependent(t, "u3")
			f.nextBlock(t)
			f.replaceOwnEdge(t, factF, factD)
			fact := f.fact(t, factF)
			fact.VerifiedAtBlock--
			bz, err := proto.MarshalOptions{Deterministic: true}.Marshal(fact)
			require.NoError(t, err)
			f.ctx.KVStore(f.key).Set(types.FactKey(factF), bz)
			return factF, factD
		}},
		{"reverse entry rewritten after the verdict", "reverse entry is not the augmentation's edge", func(t *testing.T, f *v11Fixture) (string, string) {
			_, factF, factD := f.vdDependent(t, "u4")
			f.nextBlock(t)
			f.replaceOwnEdge(t, factF, factD)
			other, err := proto.MarshalOptions{Deterministic: true}.Marshal(&types.FactRelation{
				SourceFactId: factF, TargetFactId: factD, Relation: types.RelationType_RELATION_TYPE_CITES, CreatedAtBlock: 1,
			})
			require.NoError(t, err)
			f.ctx.KVStore(f.key).Set(types.FactRelationReverseKey(factD, factF), other)
			return factF, factD
		}},
		{"method_id on a legacy-scheme round", "relation method_id is set but the claim's round does not prove record integrity at the fact's creation", func(t *testing.T, f *v11Fixture) (string, string) {
			_, claimF, factF, factD := restoreScenarioOn(t, f, "u5", restoreCitesWithMethod)
			f.nextBlock(t)
			f.replaceOwnEdge(t, factF, factD)
			round := f.roundOf(t, claimF)
			require.Equal(t, types.CommitmentSchemeReviewV2, round.CommitmentScheme)
			round.CommitmentScheme = types.CommitmentSchemeLegacy
			bz, err := proto.MarshalOptions{Deterministic: true}.Marshal(round)
			require.NoError(t, err)
			f.ctx.KVStore(f.key).Set(types.RoundKey(round.Id), bz)
			return factF, factD
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := setupV11SemanticsTest(t)
			factF, factD := tc.build(t, f)
			reason := tc.reason
			if reason == "" {
				reason = "claim " + f.fact(t, factF).ClaimId + " of the fact is missing"
			}
			replaced, _ := f.pair(factF, factD)
			before := relationContractSnapshot(t, f.ctx, f.key)
			f.freshEvents()
			f.migrate11to12(t)
			requireOnlyActivationChanged(t, before, relationContractSnapshot(t, f.ctx, f.key), nil)
			forward, _ := f.pair(factF, factD)
			require.Equal(t, replaced, forward, "left as it is")
			require.Equal(t, []map[string]string{{"fact_id": factF, "target_id": factD, "augmentation_id": factF, "reason": reason}}, v11Event(f.ctx, unrestorableEvent))
			require.Empty(t, v11Event(f.ctx, restoredEvent))
		})
	}
}

// restoreScenarioOn is restoreScenario on an existing fixture.
func restoreScenarioOn(t *testing.T, f *v11Fixture, tag string, relations func(string) []*types.ClaimRelation) (*v11Fixture, string, string, string) {
	t.Helper()
	_, factD, _ := f.contraTarget(t, "v12-rs-"+tag+"-d", "Synthetic v12 restoration ("+tag+"): fact D, which fact F names.")
	claimF := f.mustSubmitClaim(t, v11Addr("v12-rs-"+tag+"-f"), "Synthetic v12 restoration ("+tag+"): fact F, whose claim names fact D.", dedupDomain,
		func(m *types.MsgSubmitClaim) { m.Relations = relations(factD) })
	return f, claimF, f.accept(t, claimF, v11Reviewers()...), factD
}

// The augmentation inventory is bounded by records and by bytes, as the
// record-integrity activation inventory is. Above either bound, and on a
// malformed augmentation record, the upgrade fails and writes nothing; no
// record is skipped.
func TestMigrate11to12AugmentationInventoryBound(t *testing.T) {
	setup := func(t *testing.T) (*v11Fixture, int) {
		f := setupV11SemanticsTest(t)
		_, factF, factD := f.vdDependent(t, "bd")
		f.nextBlock(t)
		f.replaceOwnEdge(t, factF, factD)
		f.vdPassAugmentation(t, "v12-rs-bound-variant", factD)
		require.NoError(t, f.k.SetAugmentation(f.ctx, &types.Augmentation{Id: "v12-rs-bound-pending", OriginalFactId: factD, VariantContent: "Synthetic v12 restoration: a pending variant."}))
		size := 0
		it := f.ctx.KVStore(f.key).Iterator(types.AugmentationKeyPrefix, []byte{types.AugmentationKeyPrefix[0] + 1})
		for ; it.Valid(); it.Next() {
			size += len(it.Key()) + len(it.Value())
		}
		require.NoError(t, it.Close())
		return f, size
	}
	refused := func(t *testing.T, f *v11Fixture, want string) {
		t.Helper()
		before := relationContractSnapshot(t, f.ctx, f.key)
		err := keeper.NewMigrator(f.k).Migrate11to12(claimIntegrityOwnerContext(t, f.ctx))
		require.ErrorContains(t, err, want)
		require.Equal(t, before, relationContractSnapshot(t, f.ctx, f.key), "a refused upgrade writes nothing")
		require.False(t, f.claimIntegrityEnabled(t))
	}
	accepted := func(t *testing.T, f *v11Fixture) {
		t.Helper()
		f.inCache(func() {
			f.freshEvents()
			f.migrate11to12(t)
			require.Len(t, v11Event(f.ctx, restoredEvent), 1)
		})
	}

	t.Run("records", func(t *testing.T) {
		f, _ := setup(t)
		keeper.SetAugmentationRestoreMaxRecordsForTest(t, 3)
		accepted(t, f)
		keeper.SetAugmentationRestoreMaxRecordsForTest(t, 2)
		refused(t, f, "augmentation inventory exceeds 2 records")
	})
	t.Run("bytes", func(t *testing.T) {
		f, size := setup(t)
		keeper.SetAugmentationRestoreMaxBytesForTest(t, size)
		accepted(t, f)
		keeper.SetAugmentationRestoreMaxBytesForTest(t, size-1)
		refused(t, f, "augmentation inventory exceeds")
	})
	t.Run("malformed record", func(t *testing.T) {
		f, _ := setup(t)
		f.ctx.KVStore(f.key).Set(types.AugmentationKey("v12-rs-bound-mismatch"), []byte{0x0a, 0x01, 'x'})
		refused(t, f, "augmentation record")
	})
}

// Running the restoration again restores nothing and writes nothing; it
// reports an unrestorable pair again, since that pair is left as it is. The
// migration itself runs once (its receipt), and the receipt keeps the value
// the startup check reads.
func TestRestoreAugmentationReplacedEdgesIsIdempotent(t *testing.T) {
	f, _, factF, factD := restoreScenario(t, "id", restoreRequires)
	f.nextBlock(t)
	own := f.replaceOwnEdge(t, factF, factD)
	const genesisF = "v12-rs-idem-genesis-fact"
	require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: genesisF, Content: "Synthetic v12 restoration: a claimless fact.", Domain: dedupDomain, Status: types.FactStatus_FACT_STATUS_VERIFIED}))
	require.NoError(t, f.k.SetFactRelation(f.ctx, &types.FactRelation{SourceFactId: genesisF, TargetFactId: factD, Relation: types.RelationType_RELATION_TYPE_CITES}))
	f.replaceOwnEdge(t, genesisF, factD)

	f.freshEvents()
	f.migrate11to12(t)
	require.Len(t, v11Event(f.ctx, restoredEvent), 1)
	require.Len(t, v11Event(f.ctx, unrestorableEvent), 1)
	forward, _ := f.pair(factF, factD)
	require.Equal(t, own, forward)
	require.Equal(t, "true", f.k.ReadMigrationMarker(f.ctx, claimIntegrityReceipt))

	settled := relationContractSnapshot(t, f.ctx, f.key)
	f.freshEvents()
	restored, unrestorable, err := f.k.RestoreAugmentationReplacedEdges(f.ctx)
	require.NoError(t, err)
	require.Equal(t, [2]int{0, 1}, [2]int{restored, unrestorable})
	require.Equal(t, settled, relationContractSnapshot(t, f.ctx, f.key))
	require.Empty(t, v11Event(f.ctx, restoredEvent))
	require.Error(t, keeper.NewMigrator(f.k).Migrate11to12(claimIntegrityOwnerContext(t, f.ctx)), "the migration runs once")
	require.Equal(t, settled, relationContractSnapshot(t, f.ctx, f.key))
}

// A relaunch from a knowledge-11 export (plan §3.7, D4) runs no migration.
// With the flag on, InitGenesis restores the replaced edge exactly as the
// migration does, and an export of the result imports again unchanged. With
// the flag off the import keeps the replacement, as knowledge 11 does.
func TestClaimIntegrity_ImportRestoresEdgesReplacedByEqualIDAugmentations(t *testing.T) {
	source, _, factF, factD := restoreScenario(t, "im", restoreCitesWithMethod)
	source.nextBlock(t)
	own := source.replaceOwnEdge(t, factF, factD)
	replaced, _ := source.pair(factF, factD)
	exported := source.k.ExportGenesis(source.ctx)
	require.False(t, exported.ClaimIntegrityEnabled)

	for _, on := range []bool{true, false} {
		t.Run(map[bool]string{true: "flag on", false: "flag off"}[on], func(t *testing.T) {
			gs := proto.Clone(exported).(*types.GenesisState)
			gs.ClaimIntegrityEnabled = on
			imported := newClaimIntegrityGenesisFixture(t, "v12-restore-import", gs)
			forward, reverse := imported.pair(factF, factD)
			if !on {
				require.Equal(t, replaced, forward, "knowledge 11 imports the replacement")
				require.Equal(t, replaced, reverse)
				require.Empty(t, v11Event(imported.ctx, restoredEvent))
				return
			}
			require.Equal(t, own, forward, "F's own edge, exactly as its creation wrote it")
			require.Equal(t, own, reverse)
			require.Equal(t, []map[string]string{{"fact_id": factF, "target_id": factD, "augmentation_id": factF}}, v11Event(imported.ctx, restoredEvent))
			require.Empty(t, v11Event(imported.ctx, unrestorableEvent))

			again := imported.k.ExportGenesis(imported.ctx)
			reimported := newClaimIntegrityGenesisFixture(t, "v12-restore-reimport", again)
			require.Empty(t, v11Event(reimported.ctx, restoredEvent), "nothing is left to restore")
			require.True(t, proto.Equal(again.FactRelationState, reimported.k.ExportGenesis(reimported.ctx).FactRelationState))
		})
	}
}
