package keeper_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

const importGoldenSubject = "import golden subject S"

// v11ImportGenesis is a hand-written genesis with the four knowledge-11
// selections and no claim integrity. Its facts carry structures and tags and
// are listed out of creation order: the last fact on subject S in the file is
// not the one created last.
func v11ImportGenesis() *types.GenesisState {
	gs := types.DefaultGenesis()
	gs.ClaimIntegrityEnabled = false // knowledge 11 has no claim-integrity flag.
	// Nor the consensus-repairs-v1 marker (0x90) that native genesis now sets.
	gs.ReformulationVindicationOrderingEnabled = false
	// Nor the consensus-repairs-v2 markers (0x91 onwards).
	gs.InconclusiveRoundAccountingEnabled = false
	gs.CalibrationSmoothingEnabled = false
	gs.ZeroConfidenceFloorEnabled = false
	structure := func(subject string, tags ...string) *types.ClaimStructure {
		return &types.ClaimStructure{Subject: subject, Predicate: "has dry mass in grams", Object: "3.5", Scope: "dry sample at 20 C", Tags: tags}
	}
	fact := func(id string, verifiedAt uint64, s *types.ClaimStructure) *types.Fact {
		return &types.Fact{
			Id: id, Content: "Synthetic import golden fact " + id + ".", Domain: "physics", Category: "empirical",
			Status: types.FactStatus_FACT_STATUS_VERIFIED, Confidence: 800_000,
			SubmittedAtBlock: verifiedAt - 5, VerifiedAtBlock: verifiedAt, LastVerifiedBlock: verifiedAt,
			Structure: s,
		}
	}
	gs.Facts = []*types.Fact{
		fact("f0000000000000000000000000000000", 40, structure(importGoldenSubject, "golden-shared", "Golden-F")),
		fact("a0000000000000000000000000000000", 30, structure(importGoldenSubject, "golden-shared")),
		fact("c0000000000000000000000000000000", 40, structure(importGoldenSubject, "golden-shared", " golden-c ")),
		fact("b0000000000000000000000000000000", 20, &types.ClaimStructure{Predicate: "is tagged without a subject", Tags: []string{"golden-shared", ""}}),
		fact("d0000000000000000000000000000000", 10, nil),
	}
	return gs
}

// v11GoldenImportStoreDigest was recorded on a4272bfa (knowledge v11): the
// whole knowledge store after InitGenesis(v11ImportGenesis()) on
// newKnowledgeStoreFixture("v11-import-golden"), with the flag line dropped
// because a4272bfa has no flag (318 entries). That import writes the facts and
// no subject or tag entry. With the flag off no later change may move it.
const v11GoldenImportStoreDigest = "97c340e38df0d479aacaff35a7b658ce1bcff6970612182f61a9868e4a4f7528"

func TestV11_GoldenImportStoreDigest(t *testing.T) {
	f := newKnowledgeStoreFixture(t, "v11-import-golden")
	require.NoError(t, f.k.InitGenesis(f.ctx, v11ImportGenesis()))
	require.Empty(t, f.factIndexEntries(t), "knowledge 11 indexes no imported fact")
	digest, entries := f.storeDigest(t)
	t.Logf("knowledge store after the v11 import: entries=%d sha256=%s", entries, digest)
	require.Equal(t, v11GoldenImportStoreDigest, digest)
}

// With the flag on, InitGenesis indexes every imported fact that carries a
// structure, as IndexFactBySubject does when a round creates a fact: each
// non-blank tag (trimmed, lowercased) and a non-empty subject. Facts are
// indexed in creation order (VerifiedAtBlock, the round ID the fact's claim
// names, then the fact ID), so the one subject entry names the fact created
// last; here two facts share block 40 and name no claim, so the greater fact
// ID wins. A fact from a genesis file is indexed too, although no round
// created it.
func TestClaimIntegrity_ImportIndexesInCreationOrder(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		t.Run(fmt.Sprintf("reversed=%t", reversed), func(t *testing.T) {
			gs := v11ImportGenesis()
			gs.ClaimIntegrityEnabled = true
			if reversed {
				slices.Reverse(gs.Facts)
			}
			f := newClaimIntegrityGenesisFixture(t, "claim-integrity-import-order", gs)
			f.requireFactIndexes(t, []importIndexTag{
				{"golden-shared", []string{"a0000000000000000000000000000000", "b0000000000000000000000000000000", "c0000000000000000000000000000000", "f0000000000000000000000000000000"}},
				{"golden-f", []string{"f0000000000000000000000000000000"}},
				{"GOLDEN-C", []string{"c0000000000000000000000000000000"}},
				{"has dry mass in grams", nil},
			}, []importIndexSubject{
				{importGoldenSubject, "f0000000000000000000000000000000"},
				{"is tagged without a subject", ""},
			})
			require.Len(t, f.factIndexEntries(t), 1+6, "one subject entry; six tag entries")
		})
	}
}
