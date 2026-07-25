package keeper_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// The probe-invitation heartbeat's scan bound.
//
// InviteIdleFactsForProbing runs from BeginBlocker on EVERY block. Its only
// stopping condition was `invited >= ProbeInvitationBatchSize`, and that
// counter advances only when a fact is actually INVITED. In the steady state
// — where the re-invite cooldown disqualifies every candidate — nothing was
// invited, so the loop walked the entire fact corpus on every single block,
// while its doc comment claimed O(constant).
//
// The scan is now bounded by facts EXAMINED and resumes from a persisted
// cursor. These tests pin the bound via the cursor, which is the externally
// observable consequence: a cursor left set means the scan stopped early
// rather than running to the end of the corpus.

func TestInviteIdleFacts_ScanStopsBeforeExhaustingCorpus(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	params, _ := k.GetParams(ctx)

	// A corpus larger than the per-block examination bound, none of it
	// eligible: PENDING facts never pass the status gate.
	const corpus = 400
	for i := 0; i < corpus; i++ {
		if err := k.SetFact(ctx, &types.Fact{
			Id:     fmt.Sprintf("fact-%04d", i),
			Status: types.FactStatus_FACT_STATUS_PENDING,
		}); err != nil {
			t.Fatalf("SetFact: %v", err)
		}
	}

	height := params.ProbeInvitationIdleThresholdBlocks + 1
	ctx = ctx.WithBlockHeight(int64(height))
	k.InviteIdleFactsForProbing(ctx, height, params)

	cursor, err := k.ProbeScanCursor(ctx)
	if err != nil {
		t.Fatalf("reading cursor: %v", err)
	}
	if len(cursor) == 0 {
		t.Fatal("no scan cursor after one block over a 400-fact corpus — the scan " +
			"ran to completion, which is the O(corpus)-per-block heartbeat")
	}
}

// Bounding the READ must not starve facts further along the keyspace. This is
// the property a naive "only ever examine the first N" cap would silently
// break — and silently starving later facts while still emitting invitations
// for the early ones is exactly the kind of partial failure that reads as
// success. Over successive blocks every fact must get its turn, including the
// last one in key order.
func TestInviteIdleFacts_LaterFactsAreNotStarved(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	params, _ := k.GetParams(ctx)

	// A corpus more than twice the per-block examination bound, entirely
	// eligible, so progress is observable as invitations.
	const corpus = 600
	lastID := fmt.Sprintf("fact-%04d", corpus-1)
	for i := 0; i < corpus; i++ {
		if err := k.SetFact(ctx, &types.Fact{
			Id:              fmt.Sprintf("fact-%04d", i),
			Status:          types.FactStatus_FACT_STATUS_VERIFIED,
			Confidence:      900_000,
			VerifiedAtBlock: 1,
		}); err != nil {
			t.Fatalf("SetFact: %v", err)
		}
	}

	height := params.ProbeInvitationIdleThresholdBlocks + 10
	ctx = ctx.WithBlockHeight(int64(height))

	first, _ := k.ProbeScanCursor(ctx)
	if len(first) != 0 {
		t.Fatal("expected no cursor before the first pass")
	}

	// batchSize invitations per block; give it enough blocks to cover the
	// corpus with room to spare.
	blocks := 2 * (corpus/int(params.ProbeInvitationBatchSize) + 1)
	for i := 0; i < blocks; i++ {
		k.InviteIdleFactsForProbing(ctx, height, params)
	}

	last, found := k.GetFact(ctx, lastID)
	if !found {
		t.Fatalf("%s missing", lastID)
	}
	if last.ProbeInvitedAtBlock == 0 {
		t.Fatalf("%s — the final fact in key order — was never invited after %d "+
			"blocks; the bounded scan starves the tail of the corpus", lastID, blocks)
	}

	// Count only this test's fixtures: setupKnowledgeTest also seeds the 47
	// genesis doctrine facts, which the scan legitimately covers too.
	invited := 0
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if strings.HasPrefix(f.Id, "fact-") && f.ProbeInvitedAtBlock > 0 {
			invited++
		}
		return false
	})
	if invited != corpus {
		t.Fatalf("covered %d/%d fixtures — the scan does not reach the whole corpus", invited, corpus)
	}
}

// A single block must never invite more than the batch size, even when the
// whole corpus is eligible.
func TestInviteIdleFacts_RespectsBatchSizeWhenAllEligible(t *testing.T) {
	k, ctx := setupKnowledgeTest(t)
	params, _ := k.GetParams(ctx)

	const corpus = 50
	for i := 0; i < corpus; i++ {
		if err := k.SetFact(ctx, &types.Fact{
			Id:              fmt.Sprintf("fact-%04d", i),
			Status:          types.FactStatus_FACT_STATUS_VERIFIED,
			Confidence:      900_000,
			VerifiedAtBlock: 1,
		}); err != nil {
			t.Fatalf("SetFact: %v", err)
		}
	}

	height := params.ProbeInvitationIdleThresholdBlocks + 10
	ctx = ctx.WithBlockHeight(int64(height))
	k.InviteIdleFactsForProbing(ctx, height, params)

	invited := 0
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f.ProbeInvitedAtBlock == height {
			invited++
		}
		return false
	})
	if uint32(invited) > params.ProbeInvitationBatchSize {
		t.Fatalf("invited %d facts in one block, batch size is %d",
			invited, params.ProbeInvitationBatchSize)
	}
	if invited == 0 {
		t.Fatal("invited nothing despite an entirely eligible corpus — the " +
			"bounded scan has broken the honest path")
	}
}
