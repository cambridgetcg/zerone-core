package keeper

import (
	"bytes"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/zerone-chain/zerone/x/knowledge/types"
	"google.golang.org/protobuf/proto"
	"strings"

	v3 "github.com/zerone-chain/zerone/x/knowledge/migrations/v3"
	v4 "github.com/zerone-chain/zerone/x/knowledge/migrations/v4"
	v5 "github.com/zerone-chain/zerone/x/knowledge/migrations/v5"
)

// Migrator handles in-place store migrations.
type Migrator struct {
	keeper Keeper
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 migrates from version 1 to version 2.
// Writes a verifiable marker to confirm the migration ran successfully.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return m.keeper.WriteMigrationMarker(ctx, "migration_v2_complete", "true")
}

// Migrate2to3 migrates from version 2 to version 3.
// Backfills R29 param defaults for zero-valued fields after upgrade.
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	return v3.Migrate(ctx, m.keeper)
}

// Migrate3to4 migrates from version 3 to version 4 (Route B Wave 10).
// Backfills TraceSchema if missing; records a verifiable marker.
// See docs/UPGRADE_PROTOCOL.md for the canonical pattern this follows.
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	return v4.Migrate(ctx, m.keeper)
}

// Migrate4to5 migrates from version 4 to version 5.
// Drops eleven dead anti-slop / FARM / citation-gaming params from Params
// (defined but never read by any keeper path) and records a verifiable marker.
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	return v5.Migrate(ctx, m.keeper)
}

// Migrate5to6 marks the coordinated activation boundary for the consolidation
// safety release. The release adds consensus behavior and message surfaces but
// does not require an in-place rewrite of existing knowledge records.
func (m Migrator) Migrate5to6(ctx sdk.Context) error {
	return m.keeper.WriteMigrationMarker(ctx, "migration_v6_complete", "true")
}

// Migrate6to7 is invoked only by the app's separately named, completed-H3
// guarded upgrade. It never rewrites old counters, financial obligations or
// missing history. The cache makes failure atomic; the marker makes replay a
// no-op rather than resetting a subsequently enabled cohort or pruned latch.
func (m Migrator) Migrate6to7(ctx sdk.Context) error {
	k := m.keeper
	cache, write := ctx.CacheContext()
	marker, found, err := k.ReadMigrationMarkerPresenceChecked(cache, "migration_v7_complete")
	if err != nil {
		return err
	}
	if found {
		if marker != "true" {
			return fmt.Errorf("invalid v7 migration marker")
		}
		p, err := k.getFactUseParams(cache)
		if err != nil {
			return err
		}
		state, err := k.GetFactUsePruningState(cache)
		if err != nil {
			return err
		}
		retained, err := k.HasRetainedFactUseReceipts(cache)
		if err != nil {
			return err
		}
		return types.ValidateFactUseNonEconomicState(p, state, retained)
	}
	store := k.storeService.OpenKVStore(cache)
	bz, err := store.Get(types.ParamsKey)
	if err != nil {
		return err
	}
	if bz == nil {
		return fmt.Errorf("knowledge v6 params absent")
	}
	p := new(types.Params)
	if err := proto.Unmarshal(bz, p); err != nil {
		return err
	}
	p.FactUseEnabled = false
	p.FactUseConsumers = nil
	p.FactUseMaxPerConsumerEpoch = types.MaxFactUsePerConsumerEpoch
	p.FactUseMaxPerEpoch = types.MaxFactUsePerEpoch
	if err := p.Validate(); err != nil {
		return err
	}
	if err := k.SetParams(cache, p); err != nil {
		return err
	}
	state, err := k.GetFactUsePruningState(cache)
	if err != nil {
		return err
	}
	if state.EverReported || len(state.NextKey) != 0 {
		return fmt.Errorf("unexpected feedback state before v7")
	}
	if err := k.InitFactUseReceipts(cache, nil, state); err != nil {
		return err
	}
	if err := k.repairTerminalFeedbackChallenges(cache); err != nil {
		return err
	}
	if err := k.WriteMigrationMarker(cache, "migration_v7_complete", "true"); err != nil {
		return err
	}
	write()
	return nil
}

// Only the old constructors' explicit challenge text + retained terminal round
// and matching claim verdict justify repair. Unknown/missing rounds, ambiguous
// claims and any live contradiction/challenge prevent repair. A uniquely matched
// REJECT ceases to block only when its completion provably precedes a later
// stranded submission. No legacy round metadata or event is invented. The new
// transition records this migration.
func (k Keeper) repairTerminalFeedbackChallenges(ctx sdk.Context) error {
	claims := []*types.Claim{}
	knownClaims := map[string]bool{}
	unattributedLiveRound := false
	rounds := map[string][]*types.VerificationRound{}
	if err := k.scanDurableRecords(ctx, types.ClaimKeyPrefix, func(key, bz []byte) error {
		c := new(types.Claim)
		if err := proto.Unmarshal(bz, c); err != nil {
			return err
		}
		if !bytes.Equal(key, types.ClaimKey(c.Id)) {
			return fmt.Errorf("claim key mismatch")
		}
		claims = append(claims, c)
		knownClaims[c.Id] = true
		return nil
	}); err != nil {
		return err
	}
	if err := k.scanDurableRecords(ctx, types.VerificationRoundKeyPrefix, func(key, bz []byte) error {
		r := new(types.VerificationRound)
		if err := proto.Unmarshal(bz, r); err != nil {
			return err
		}
		if !bytes.Equal(key, types.RoundKey(r.Id)) {
			return fmt.Errorf("round key mismatch")
		}
		rounds[r.ClaimId] = append(rounds[r.ClaimId], r)
		if !knownClaims[r.ClaimId] && r.Phase != types.VerificationPhase_VERIFICATION_PHASE_COMPLETE && r.Phase != types.VerificationPhase_VERIFICATION_PHASE_EXPIRED {
			unattributedLiveRound = true
		}
		return nil
	}); err != nil {
		return err
	}
	// Without a live round's claim its target is unknowable. Defer all repairs
	// rather than asserting that no live challenger remains for a candidate.
	if unattributedLiveRound {
		return nil
	}
	blocked := map[string]bool{}
	latestRejectedCompletion := map[string]uint64{}
	eligible := []*types.Claim{}
	migrationHeight := uint64(ctx.BlockHeight())
	for _, c := range claims {
		rs := rounds[c.Id]
		terminal := len(rs) == 1 && (rs[0].Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE || rs[0].Phase == types.VerificationPhase_VERIFICATION_PHASE_EXPIRED)
		referenceMatches := terminal && (c.VerificationRoundId == "" || c.VerificationRoundId == rs[0].Id)
		matching := referenceMatches && ((c.Status == types.ClaimStatus_CLAIM_STATUS_INSUFFICIENT && rs[0].Verdict == types.Verdict_VERDICT_INCONCLUSIVE) || (c.Status == types.ClaimStatus_CLAIM_STATUS_MALFORMED && rs[0].Verdict == types.Verdict_VERDICT_MALFORMED))
		constructor := c.ProvisionalFactId != "" && (strings.HasPrefix(c.FactContent, "Challenge of fact "+c.ProvisionalFactId+": ") || strings.HasPrefix(c.FactContent, "Provisional challenge of fact "+c.ProvisionalFactId+": "))
		if c.ProvisionalFactId != "" {
			switch {
			case constructor && matching:
				eligible = append(eligible, c)
			case constructor && terminal && c.Status == types.ClaimStatus_CLAIM_STATUS_REJECTED &&
				rs[0].Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE && rs[0].Verdict == types.Verdict_VERDICT_REJECT &&
				terminalFeedbackChallengeChronology(c, rs[0], migrationHeight):
				// A proven REJECT may precede a later stranded challenge. Retain
				// its completion height rather than treating it as a live blocker.
				if rs[0].VerdictBlock > latestRejectedCompletion[c.ProvisionalFactId] {
					latestRejectedCompletion[c.ProvisionalFactId] = rs[0].VerdictBlock
				}
			default:
				blocked[c.ProvisionalFactId] = true
			}
		}
		if !terminal {
			for _, r := range c.Relations {
				if r != nil && r.Relation == types.RelationType_RELATION_TYPE_CONTRADICTS {
					blocked[r.TargetFactId] = true
				}
			}
		}
	}
	// Claims are in canonical key order. Only the first qualifying strand writes
	// a restoration; subsequent candidates see the restored status.
	for _, c := range eligible {
		if blocked[c.ProvisionalFactId] {
			continue
		}
		if rejectedAt := latestRejectedCompletion[c.ProvisionalFactId]; rejectedAt > 0 {
			// A REJECT restores its target. To explain today's CHALLENGED state,
			// the stranded challenge must have STARTED strictly afterwards, not
			// merely completed afterwards. Equal heights cannot prove ordering.
			if !terminalFeedbackChallengeChronology(c, rounds[c.Id][0], migrationHeight) || rejectedAt >= c.SubmittedAtBlock {
				continue
			}
		}
		fact, found, err := k.getFeedbackFact(ctx, c.ProvisionalFactId)
		if err != nil {
			return err
		}
		if !found || fact.Status != types.FactStatus_FACT_STATUS_CHALLENGED {
			continue
		}
		// The retained claim and terminal round prove this repair's cause. Record
		// only the transition performed now, never a synthetic earlier challenge,
		// verdict event, completion metadata, or gap-filling history.
		if err := k.restoreChallengedFactOnInconclusive(ctx, c, "migration_tok_feedback_v1_terminal_challenge"); err != nil {
			return err
		}
	}
	return nil
}

// Chronology is required to disregard an earlier rejection, not synthesized for
// legacy records whose timing is missing. Both old challenge constructors create
// their round at submission height. EXPIRED starvation did not set VerdictBlock;
// its retained aggregation deadline instead bounds when it could have closed.
func terminalFeedbackChallengeChronology(c *types.Claim, r *types.VerificationRound, height uint64) bool {
	if c.VerificationRoundId == "" || c.VerificationRoundId != r.Id || c.SubmittedAtBlock == 0 ||
		r.StartedAtBlock != c.SubmittedAtBlock || r.StartedAtBlock > height {
		return false
	}
	switch r.Phase {
	case types.VerificationPhase_VERIFICATION_PHASE_COMPLETE:
		return r.VerdictBlock >= r.StartedAtBlock && r.VerdictBlock <= height
	case types.VerificationPhase_VERIFICATION_PHASE_EXPIRED:
		return r.Verdict == types.Verdict_VERDICT_INCONCLUSIVE && r.VerdictBlock == 0 &&
			r.AggregationDeadline > r.StartedAtBlock && r.AggregationDeadline <= height
	default:
		return false
	}
}
