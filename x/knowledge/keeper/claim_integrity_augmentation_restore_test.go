package keeper_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/log"
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
	restoration, err := f.k.RestoreAugmentationReplacedEdges(f.ctx)
	require.NoError(t, err)
	require.Equal(t, [2]int{0, 1}, [2]int{len(restoration.Restored), len(restoration.Unrestorable)})
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

// relaunchAtHeightOne imports gs into a fresh store at initial height 1, as a
// relaunch from an export does (zerone-dev-1's runtime accepts only initial
// height 0 or 1, deploy/networks/zerone-dev-1/runtime.py check_genesis). Block
// heights start again; the imported records keep the heights of the chain
// that wrote them.
func relaunchAtHeightOne(t *testing.T, chainID string, gs *types.GenesisState) *v11Fixture {
	t.Helper()
	f := newKnowledgeStoreFixture(t, chainID)
	f.ctx = f.ctx.WithBlockHeight(1)
	require.NoError(t, f.k.InitGenesis(f.ctx, gs))
	return f
}

// After a relaunch at initial height 1, a verdict's height and a fact's
// verified_at_block come from different chains, so "the fact existed at the
// verdict" cannot be read from them. A knowledge-11 chain relaunched from an
// export, whose equal-ID verdict ran at height 2 against a fact accepted at
// height 100+ on the old chain, replaced that fact's own edge all the same.
// The forward entry proves it: F's claim names D, and F's creation writes its
// edge to D with the claim's submitter as creator, so had F been created after
// the verdict its own write, not the verdict's, would hold the pair. The
// upgrade and an import with the flag on restore the edge.
func TestRestoreAugmentationReplacedEdgesAfterRelaunchAtHeightOne(t *testing.T) {
	source, _, factF, factD := restoreScenario(t, "rl", restoreRequires)
	own, _ := source.pair(factF, factD)
	exported := source.k.ExportGenesis(source.ctx)
	require.False(t, exported.ClaimIntegrityEnabled, "a knowledge-11 export")

	relaunched := relaunchAtHeightOne(t, "v12-restore-relaunch", exported)
	relaunched.advanceTo(t, 2)
	require.Equal(t, own, relaunched.replaceOwnEdge(t, factF, factD))
	aug, found := relaunched.k.GetAugmentation(relaunched.ctx, factF)
	require.True(t, found)
	require.Less(t, aug.VerdictBlock, relaunched.fact(t, factF).VerifiedAtBlock, "heights from two chains")

	t.Run("upgrade", func(t *testing.T) {
		relaunched.inCache(func() {
			relaunched.freshEvents()
			relaunched.migrate11to12(t)
			forward, reverse := relaunched.pair(factF, factD)
			require.Equal(t, own, forward, "F's own edge, exactly as its creation wrote it")
			require.Equal(t, own, reverse)
			require.Equal(t, []map[string]string{{"fact_id": factF, "target_id": factD, "augmentation_id": factF}}, v11Event(relaunched.ctx, restoredEvent))
			require.Empty(t, v11Event(relaunched.ctx, unrestorableEvent))
		})
	})
	t.Run("import with the flag on", func(t *testing.T) {
		gs := relaunched.k.ExportGenesis(relaunched.ctx)
		gs.ClaimIntegrityEnabled = true
		imported := relaunchAtHeightOne(t, "v12-restore-relaunch-again", gs)
		forward, reverse := imported.pair(factF, factD)
		require.Equal(t, own, forward)
		require.Equal(t, own, reverse)
		require.Equal(t, []map[string]string{{"fact_id": factF, "target_id": factD, "augmentation_id": factF}}, v11Event(imported.ctx, restoredEvent))
	})
}

// The height comparison still decides whether a pair whose fact cannot be
// traced to a claim is reported, and only that; it never decides a write. On
// one chain a claimless fact created after the verdict never had an edge to
// replace and is not reported. After a relaunch at initial height 1 the
// heights may come from two chains, so such a pair can go unreported (this
// case) or be reported although nothing was replaced; nothing is written
// either way (spec §1.5, "Which pairs").
func TestRestoreAugmentationReplacedEdgesClaimlessReportAfterRelaunch(t *testing.T) {
	source := setupV11SemanticsTest(t)
	_, factD, _ := source.contraTarget(t, "v12-rs-rc-d", "Synthetic v12 restoration (rc): fact D, which a genesis fact cites.")
	const genesisG = "v12-rs-relaunch-genesis-fact"
	require.NoError(t, source.k.SetFact(source.ctx, &types.Fact{
		Id: genesisG, Content: "Synthetic v12 restoration: a claimless fact verified on the old chain.", Domain: dedupDomain,
		Status: types.FactStatus_FACT_STATUS_VERIFIED, VerifiedAtBlock: uint64(source.ctx.BlockHeight()),
	}))
	require.NoError(t, source.k.SetFactRelation(source.ctx, &types.FactRelation{
		SourceFactId: genesisG, TargetFactId: factD, Relation: types.RelationType_RELATION_TYPE_CITES,
	}))
	relaunched := relaunchAtHeightOne(t, "v12-restore-relaunch-claimless", source.k.ExportGenesis(source.ctx))
	relaunched.advanceTo(t, 2)
	relaunched.replaceOwnEdge(t, genesisG, factD)
	replaced, _ := relaunched.pair(genesisG, factD)

	before := relationContractSnapshot(t, relaunched.ctx, relaunched.key)
	relaunched.freshEvents()
	relaunched.migrate11to12(t)
	requireOnlyActivationChanged(t, before, relationContractSnapshot(t, relaunched.ctx, relaunched.key), nil)
	forward, _ := relaunched.pair(genesisG, factD)
	require.Equal(t, replaced, forward, "left as it is")
	require.Empty(t, v11Event(relaunched.ctx, restoredEvent))
	require.Empty(t, v11Event(relaunched.ctx, unrestorableEvent), "the documented residual: unreported after a relaunch")
}

// captureLog sends the fixture's log to a buffer, as JSON lines.
func (f *v11Fixture) captureLog() *bytes.Buffer {
	buf := &bytes.Buffer{}
	f.ctx = f.ctx.WithLogger(log.NewLogger(buf, log.OutputJSONOption(), log.ColorOption(false)))
	return buf
}

// restorationLog returns the restoration's log lines, in order, without
// their time.
func restorationLog(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		entry := map[string]any{}
		require.NoError(t, json.Unmarshal(line, &entry))
		if message, _ := entry["message"].(string); strings.HasPrefix(message, "claim integrity: augmentation-replaced fact edge") {
			delete(entry, "time")
			out = append(out, entry)
		}
	}
	return out
}

// The restoration's record outside the events. BaseApp.InitChain discards
// the events InitGenesis emits (CometBFT 0.38's ResponseInitChain carries
// none), so an import's restorations would leave no trace but the store. The
// node log carries one line with the counts and then one line per pair, in
// augmentation key order, for an import and for the upgrade (whose events
// are also in the FinalizeBlock events of the upgrade height). The log is not
// state; it is written after the restoration succeeded.
func TestRestoreAugmentationReplacedEdgesLogsEveryPair(t *testing.T) {
	source, _, factF, factD := restoreScenario(t, "lg", restoreRequires)
	source.nextBlock(t)
	source.replaceOwnEdge(t, factF, factD)
	const genesisG = "v12-rs-log-genesis-fact"
	require.NoError(t, source.k.SetFact(source.ctx, &types.Fact{Id: genesisG, Content: "Synthetic v12 restoration: a claimless fact whose edge is logged.", Domain: dedupDomain, Status: types.FactStatus_FACT_STATUS_VERIFIED}))
	require.NoError(t, source.k.SetFactRelation(source.ctx, &types.FactRelation{SourceFactId: genesisG, TargetFactId: factD, Relation: types.RelationType_RELATION_TYPE_CITES}))
	source.replaceOwnEdge(t, genesisG, factD)
	require.Less(t, factF, genesisG, "augmentation key order")
	exported := source.k.ExportGenesis(source.ctx)

	want := func(origin string) []map[string]any {
		line := func(message string, fields map[string]any) map[string]any {
			fields["level"], fields["module"], fields["message"], fields["origin"] = "info", "x/knowledge", message, origin
			return fields
		}
		return []map[string]any{
			line("claim integrity: augmentation-replaced fact edges", map[string]any{"restored": float64(1), "unrestorable": float64(1)}),
			line("claim integrity: augmentation-replaced fact edge restored", map[string]any{"fact_id": factF, "target_id": factD, "augmentation_id": factF}),
			line("claim integrity: augmentation-replaced fact edge unrestorable", map[string]any{"fact_id": genesisG, "target_id": factD, "augmentation_id": genesisG, "reason": "fact has no claim"}),
		}
	}
	t.Run("import", func(t *testing.T) {
		gs := proto.Clone(exported).(*types.GenesisState)
		gs.ClaimIntegrityEnabled = true
		imported := newKnowledgeStoreFixture(t, "v12-restore-log-import")
		buf := imported.captureLog()
		require.NoError(t, imported.k.InitGenesis(imported.ctx, gs))
		require.Equal(t, want("genesis"), restorationLog(t, buf))
	})
	t.Run("native genesis logs the counts", func(t *testing.T) {
		native := newKnowledgeStoreFixture(t, "v12-restore-log-native")
		buf := native.captureLog()
		require.NoError(t, native.k.InitGenesis(native.ctx, types.DefaultGenesis()))
		require.Equal(t, []map[string]any{{
			"level": "info", "module": "x/knowledge", "origin": "genesis",
			"message": "claim integrity: augmentation-replaced fact edges", "restored": float64(0), "unrestorable": float64(0),
		}}, restorationLog(t, buf))
	})
	t.Run("import with the flag off", func(t *testing.T) {
		imported := newKnowledgeStoreFixture(t, "v12-restore-log-import-off")
		buf := imported.captureLog()
		require.NoError(t, imported.k.InitGenesis(imported.ctx, exported))
		require.Empty(t, restorationLog(t, buf), "knowledge 11 restores nothing")
	})
	t.Run("upgrade", func(t *testing.T) {
		buf := source.captureLog()
		source.migrate11to12(t)
		require.Equal(t, want("upgrade"), restorationLog(t, buf))
	})
	t.Run("a restoration that fails after a pair logs no pair", func(t *testing.T) {
		f, _, factF, factD := restoreScenario(t, "lr", restoreRequires)
		f.nextBlock(t)
		f.replaceOwnEdge(t, factF, factD)
		// A later pair, in key order, whose fact names a malformed claim record.
		const brokenG, brokenClaim = "v12-rs-log-broken-fact", "v12-rs-log-broken-claim"
		require.Less(t, factF, brokenG)
		require.NoError(t, f.k.SetFact(f.ctx, &types.Fact{Id: brokenG, ClaimId: brokenClaim, Content: "Synthetic v12 restoration: a fact whose claim record is malformed.", Domain: dedupDomain, Status: types.FactStatus_FACT_STATUS_VERIFIED}))
		f.vdPassAugmentation(t, brokenG, factD)
		f.ctx.KVStore(f.key).Set(types.ClaimKey(brokenClaim), []byte{0xff})
		buf := f.captureLog()
		require.ErrorContains(t, keeper.NewMigrator(f.k).Migrate11to12(claimIntegrityOwnerContext(t, f.ctx)), "augmentation "+brokenG)
		require.Empty(t, restorationLog(t, buf), "nothing was restored, so nothing is logged")
	})
}
