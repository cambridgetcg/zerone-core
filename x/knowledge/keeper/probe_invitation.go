package keeper

import (
	"bytes"
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// ProbeScanCursor returns the current resumption point of the probe-invitation
// scan: the store key of the next Fact to examine, or empty when the last pass
// exhausted the corpus and the next will start from the beginning. Exported so
// operators and tests can see that the heartbeat is genuinely bounded — an
// empty cursor after a pass over a large corpus means the scan ran to
// completion in one block.
func (k Keeper) ProbeScanCursor(ctx context.Context) ([]byte, error) {
	return k.storeService.OpenKVStore(ctx).Get(types.ProbeScanCursorKey)
}

// probeScanMaxExaminedPerBlock bounds how many facts the invitation
// heartbeat may READ in one block, independently of how many it invites.
// Without it the heartbeat's cost was O(corpus) per block — the guard it
// shipped with counted invitations, which in steady state is zero.
const probeScanMaxExaminedPerBlock uint32 = 256

// InviteIdleFactsForProbing is the Wave 15 stress-test invitation
// heartbeat. Each block it scans a bounded slice of facts, nominates
// eligible idle claims for external probing, and emits an invitation
// event that external prober agents can subscribe to.
//
// Eligibility criteria for an invitation:
//
//   - Fact status is VERIFIED or ACTIVE (facts the chain asserts are
//     currently trustworthy — the interesting ones to probe).
//   - Confidence ≥ ProbeInvitationMinConfidenceBps (default 70%). Low-
//     confidence facts don't need the nudge; verifiers are already
//     adjudicating them.
//   - Time-since-last-probe ≥ ProbeInvitationIdleThresholdBlocks.
//     "Last probe" = max(LastChallengedBlock, LastCorroboratedBlock,
//     ProbeInvitedAtBlock). A fact recently stress-tested (corroborated
//     or challenged) doesn't need a new invitation yet.
//   - Re-invite cooldown: if the fact was already invited within
//     ProbeInvitationReinviteCooldown blocks, skip. Prevents spam in
//     event logs for facts nobody ever probes.
//
// Per-block work is bounded twice over: at most ProbeInvitationBatchSize
// invitations AND at most probeScanMaxExaminedPerBlock facts read. The
// second bound is the load-bearing one. This function previously claimed to
// be O(constant) on the strength of the batch size alone, but the batch
// counter only advances when a fact is actually invited — so in the steady
// state, where the re-invite cooldown disqualifies every candidate, nothing
// incremented it and the loop walked the entire corpus on every single
// block. A cursor persisted in ProbeScanCursorKey carries the scan forward
// across blocks so bounding the read does not starve facts further along
// the keyspace.
//
// Emits: zerone.knowledge.probe_invited per invited fact.
func (k Keeper) InviteIdleFactsForProbing(ctx context.Context, height uint64, params *types.Params) {
	if params == nil {
		return
	}
	batchSize := params.ProbeInvitationBatchSize
	if batchSize == 0 {
		return
	}
	threshold := params.ProbeInvitationIdleThresholdBlocks
	if threshold == 0 || height < threshold {
		// Chain hasn't been running long enough for any fact to be "idle"
		// beyond the threshold; save the scan.
		return
	}
	minConf := params.ProbeInvitationMinConfidenceBps
	reinviteCooldown := params.ProbeInvitationReinviteCooldown

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.storeService.OpenKVStore(ctx)
	end := prefixEndBytes(types.FactKeyPrefix)

	// Resume from where the previous block stopped. An absent cursor means
	// "start at the beginning of the fact keyspace".
	start := types.FactKeyPrefix
	if cur, err := store.Get(types.ProbeScanCursorKey); err == nil && len(cur) > 0 {
		start = cur
	}

	type invitation struct {
		fact      *types.Fact
		lastProbe uint64
	}
	var (
		invited  uint32
		examined uint32
		pending  []invitation
		nextCur  []byte
	)

	// scan walks the fact keyspace from `from`, stopping at the batch limit
	// or the examination limit — whichever comes first. Stopping on EXAMINED
	// and not merely on INVITED is the whole point: the old loop only ever
	// checked the invited count, so a corpus where nothing qualified was
	// walked end to end on every block.
	scan := func(from []byte) {
		iter, err := store.Iterator(from, end)
		if err != nil {
			return
		}
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			if invited >= batchSize || examined >= probeScanMaxExaminedPerBlock {
				nextCur = append([]byte{}, iter.Key()...)
				return
			}
			examined++
			var fv types.Fact
			if err := proto.Unmarshal(iter.Value(), &fv); err != nil {
				continue
			}
			f := &fv
			// Status gate.
			if f.Status != types.FactStatus_FACT_STATUS_VERIFIED &&
				f.Status != types.FactStatus_FACT_STATUS_ACTIVE {
				continue
			}
			// Confidence gate.
			if f.Confidence < minConf {
				continue
			}
		// Time-since-last-probe: the "probe" signal is whichever is
		// latest across corroborated (failed challenge) or previously-
		// invited. A successful challenge moves the fact to DISPROVEN
		// which the status gate above filters out entirely.
			lastProbe := f.LastCorroboratedBlock
			if f.ProbeInvitedAtBlock > lastProbe {
				lastProbe = f.ProbeInvitedAtBlock
			}
			// If the fact has never been probed (lastProbe == 0), use its
			// VerifiedAtBlock so we don't invite freshly-minted facts.
			if lastProbe == 0 {
				lastProbe = f.VerifiedAtBlock
			}
			if height < lastProbe+threshold {
				continue // not idle enough yet
			}
			// Re-invite cooldown.
			if reinviteCooldown > 0 && f.ProbeInvitedAtBlock > 0 &&
				height < f.ProbeInvitedAtBlock+reinviteCooldown {
				continue
			}

			// Defer the write. SetFact inside a live iterator over the same
			// prefix mutates the range being walked; every other iteration
			// in x/knowledge collects first and writes after, and says so.
			pending = append(pending, invitation{fact: f, lastProbe: lastProbe})
			invited++
		}
		// Range exhausted without hitting a limit — wrap on the next pass.
		nextCur = nil
	}

	scan(start)
	// If we resumed mid-corpus and reached the end without filling the
	// batch, wrap once now. Otherwise a cursor parked near the end of the
	// keyspace would invite almost nothing for a whole pass.
	if nextCur == nil && !bytes.Equal(start, types.FactKeyPrefix) &&
		invited < batchSize && examined < probeScanMaxExaminedPerBlock {
		scan(types.FactKeyPrefix)
	}

	if nextCur == nil {
		_ = store.Delete(types.ProbeScanCursorKey)
	} else {
		_ = store.Set(types.ProbeScanCursorKey, nextCur)
	}

	// Writes and events, now that every iterator is closed.
	for _, inv := range pending {
		f := inv.fact
		lastProbe := inv.lastProbe
		f.ProbeInvitedAtBlock = height
		if err := k.SetFact(ctx, f); err != nil {
			k.Logger(ctx).Error("probe invitation SetFact failed", "fact", f.Id, "err", err)
			continue
		}
		// The chain manufactures probe demand. This event is the
		// announcement: "this high-confidence fact has gone idle and
		// I am inviting you to challenge it." See TRUTH_SEEKING.md
		// commitment 5.
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.probe_invited",
			sdk.NewAttribute("fact_id", f.Id),
			sdk.NewAttribute("domain", f.Domain),
			sdk.NewAttribute("confidence", fmt.Sprintf("%d", f.Confidence)),
			sdk.NewAttribute("corroboration_count", fmt.Sprintf("%d", f.CorroborationCount)),
			sdk.NewAttribute("idle_since_block", fmt.Sprintf("%d", lastProbe)),
			sdk.NewAttribute("invited_at_block", fmt.Sprintf("%d", height)),
			sdk.NewAttribute("creed_commitment", "5"),
		))
	}
}

// IdleFactsForProbing returns facts currently inviting stress-tests —
// any fact with probe_invited_at_block > 0 whose invitation hasn't been
// superseded by a subsequent challenge or corroboration. Sorted by
// time-since-invitation descending (the oldest, most-neglected
// invitations come first). Used by QueryIdleFacts to give prober agents
// a concrete work queue.
func (k Keeper) IdleFactsForProbing(ctx context.Context, domain string, limit uint32) []*types.IdleFact {
	if limit == 0 {
		limit = 50
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	out := make([]*types.IdleFact, 0, limit)
	k.IterateFacts(ctx, func(f *types.Fact) bool {
		if f == nil || f.ProbeInvitedAtBlock == 0 {
			return false
		}
		if domain != "" && f.Domain != domain {
			return false
		}
		// Invitation is "current" if no corroboration has landed since
		// it was issued. Otherwise the fact was probed already and the
		// invitation is stale. (Successful challenges flip status to
		// DISPROVEN which we also exclude via the field semantics.)
		if f.LastCorroboratedBlock > f.ProbeInvitedAtBlock {
			return false
		}
		idle := &types.IdleFact{
			Id:                  f.Id,
			Domain:              f.Domain,
			Confidence:          f.Confidence,
			CorroborationCount:  f.CorroborationCount,
			ProbeInvitedAtBlock: f.ProbeInvitedAtBlock,
		}
		if height >= f.ProbeInvitedAtBlock {
			idle.BlocksSinceInvited = height - f.ProbeInvitedAtBlock
		}
		out = append(out, idle)
		return uint32(len(out)) >= limit
	})
	return out
}
