package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// Frozen creation and provenance bodies from
// bc2a58fccb3e9bbb43647b0e51490fad034f34fd:rounds.go. Only the creation
// function name changed. Neither body calls a repaired leaf.
func (k Keeper) createFactFromClaimLegacy(ctx context.Context, claim *types.Claim, round *types.VerificationRound, confidence uint64) (string, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())
	neutral := round.ReviewPolicyVersion == types.ReviewPolicyNeutral

	factID := GenerateFactID(claim.Id, height)

	// A conjecture is a question the chain agreed to hold open, not a fact it
	// believes. The panel that returned ACCEPT was asked "is this well-posed
	// and falsifiable?", never "is this true", so the resulting node must
	// carry none of the standing that an accepted assertion carries.
	isConjecture := claim.ClaimType == types.ClaimType_CLAIM_TYPE_CONJECTURE

	// Calculate fitness epoch from current height
	params, _ := k.GetParams(ctx)
	epochBorn := uint64(0)
	if params.FitnessEpochBlocks > 0 {
		epochBorn = height / params.FitnessEpochBlocks
	}

	// Freeze the submitter's calibration at acceptance time. This is the
	// TVW snapshot that Popper-weights the fact's future revenue. Reading
	// it fresh at query time would let a submitter inflate calibration
	// after the fact lands and retroactively harvest more training value
	// — the gate has to bite at creation, not at payout.
	var calibrationSnapshot uint64
	if !neutral {
		if cal, ok := k.GetAgentCalibration(ctx, claim.Submitter); ok && cal != nil {
			calibrationSnapshot = cal.CalibrationScoreBps
		}
		if calibrationSnapshot == 0 {
			calibrationSnapshot = 500_000 // legacy default
		}
	}

	fact := &types.Fact{
		Id:                     factID,
		Content:                claim.FactContent,
		Domain:                 claim.Domain,
		Category:               claim.Category,
		Confidence:             confidence,
		Submitter:              claim.Submitter,
		SubmittedAtBlock:       claim.SubmittedAtBlock,
		VerifiedAtBlock:        height,
		LastVerifiedBlock:      height,
		References:             claim.References,
		Status:                 types.FactStatus_FACT_STATUS_VERIFIED,
		ClaimId:                claim.Id,
		FalsificationPredicate: claim.FalsificationPredicate,
		ClaimType:              claim.ClaimType,
		Structure:              claim.Structure,
		CanonicalForm:          claim.CanonicalForm,
		CanonicalHash:          claim.CanonicalHash,
		// Methodology (Phase 1): copy from claim; default to M-LEGACY if unset.
		MethodId: ResolveMethodId(claim.MethodId),
		// Reasoning trace (Phase 9): structured derivation flows through to
		// the accepted Fact as first-class training data.
		ReasoningTrace: claim.ReasoningTrace,
		// Fitness fields
		FitnessScore:        params.FitnessInitialScore,
		FitnessUpdatedBlock: height,
		EpochBorn:           epochBorn,
		// Metabolism fields
		Energy:            params.MetabolismInitialEnergy,
		EnergyCap:         params.MetabolismEnergyCap,
		EnergyLastUpdated: height,
		// Popper-weighted TVW: calibration snapshot frozen here. Any future
		// calibration drift has no retroactive effect on this fact's TVW.
		SubmitterCalibrationSnapshotBps: calibrationSnapshot,
	}

	// A conjecture enters at PROVISIONAL with zero confidence. Zero is not a
	// low opinion — it is the absence of one, and it is what keeps the node
	// inert everywhere confidence is read: it can anchor no dependency floor,
	// it clears no training tier, and it cannot be mistaken downstream for
	// something the chain stands behind.
	if isConjecture {
		fact.Status = types.FactStatus_FACT_STATUS_PROVISIONAL
		fact.Confidence = 0
	}

	// Apply domain carrying capacity birth pressure (R29-1)
	if neutral {
		energy, err := k.applyNeutralReviewBirthPressure(ctx, claim.Domain, fact.Energy)
		if err != nil {
			return "", err
		}
		fact.Energy = energy
	} else {
		fact.Energy = k.ApplyBirthPressure(ctx, claim.Domain, fact.Energy)
	}

	// Apply role bonus — claim type × account type (R28-5)
	// Skipped for conjectures: every confidence adjustment below is
	// multiplicative and would be a no-op on zero, but running them anyway
	// would mean a future additive term silently gives a question standing.
	if !neutral && !isConjecture {
		accountType := k.getAccountType(ctx, claim.Submitter)
		fact.Confidence = ApplyRoleBonusToConfidence(fact.Confidence, claim.ClaimType, accountType, params)
	}

	// Apply dual validation bonus for partnership claims (R28-5)
	// Scale by weaker role's accuracy in the domain (R29-3)
	if !neutral && claim.PartnershipId != "" {
		agentAcc, humanAcc := k.GetRoleAccuracies(ctx, claim.Domain)
		weakerAccuracy := agentAcc
		if humanAcc < weakerAccuracy || weakerAccuracy == 0 {
			weakerAccuracy = humanAcc
		}
		if weakerAccuracy > 0 {
			scaledBonusBps := safeMulDiv(params.DualValidationBonusBps, weakerAccuracy, BPS)
			fact.Confidence = safeMulDiv(fact.Confidence, 1_000_000+scaledBonusBps, 1_000_000)
		} else {
			// No track record yet — use full static bonus
			fact.Confidence = ApplyDualValidationBonus(fact.Confidence, params)
		}
	}

	// Apply confidence ceiling (stratum + global MaxConfidence hard cap)
	fact.Confidence = k.ClampConfidence(ctx, fact.Confidence, claim.Domain)
	if k.ontologyKeeper != nil && claim.Domain != "" {
		stratum, err := k.ontologyKeeper.GetStratumForDomain(ctx, claim.Domain)
		if err == nil && stratum != "" {
			fact.Stratum = stratum
		}
	}

	// ─── Epistemic provenance (ToK Wave 2) ─────────────────────────────
	// axiom_distance = min(cited_facts.axiom_distance) + 1, or 0 if no cites
	// (the fact is foundational). dependency_confidence_floor inherits the
	// weakest cited support's effective confidence so proof chains can't
	// claim more confidence than their foundations.
	// Cited targets are fetched once here and shared with the karma cited-
	// edge loop below — one IAVL read per distinct target on this
	// BeginBlocker path, not two.
	citedTargets := k.fetchCitedTargets(ctx, claim)
	dist, floor := computeProvenance(claim, citedTargets)
	fact.AxiomDistance = dist
	fact.DependencyConfidenceFloor = floor
	// Clamp the fact's own confidence to the inherited floor if it exists.
	if floor > 0 && fact.Confidence > floor {
		fact.Confidence = floor
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.confidence_clamped_to_floor",
			sdk.NewAttribute("fact_id", factID),
			sdk.NewAttribute("dependency_floor_bps", fmt.Sprintf("%d", floor)),
			sdk.NewAttribute("axiom_distance", fmt.Sprintf("%d", dist)),
		))
	}

	// Plan §5.2 row 1: with claim integrity on, a conjecture's creation
	// (UNSPECIFIED→PROVISIONAL) records verification/<claim id>, which
	// inference already gives a VERIFIED creation. Marker off: unchanged.
	createCtx := ctx
	if isConjecture {
		var err error
		if createCtx, err = k.claimIntegrityTransitionCause(ctx, factID, causeVerification, claim.Id); err != nil {
			return "", err
		}
	}
	if err := k.SetFact(createCtx, fact); err != nil {
		return "", err
	}

	// Update domain stats for carrying capacity (R29-1)
	k.IncrementDomainFactCount(ctx, fact.Domain, true, fact.Energy)

	// Index fact by structured subject and tags
	if fact.Structure != nil {
		if err := k.IndexFactBySubject(ctx, fact); err != nil {
			return "", fmt.Errorf("failed to index fact by subject: %w", err)
		}
	}

	// ─── Niche registration (competitive dynamics) ─────────────────────
	nicheKey := k.ComputeNicheKey(fact)
	fact.NicheKey = nicheKey
	fact.NicheLeader = true // new fact starts as leader-candidate
	fact.NicheRank = 1
	fact.NicheSize = 1
	currentLeader, hasLeader := k.GetNicheLeader(ctx, nicheKey)
	if hasLeader {
		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.niche_challenger",
			sdk.NewAttribute("new_fact", fact.Id),
			sdk.NewAttribute("current_leader", currentLeader.Id),
			sdk.NewAttribute("niche", nicheKey),
			sdk.NewAttribute("domain", fact.Domain),
		))
		fact.NicheLeader = false // not leader yet — competition will decide
		fact.NicheSize = uint64(len(k.GetNicheMembers(ctx, nicheKey))) + 1
	}
	if err := k.UpdateNicheIndex(ctx, fact); err != nil {
		return "", fmt.Errorf("failed to update niche index: %w", err)
	}
	// Re-save fact with niche fields set
	if err := k.SetFact(ctx, fact); err != nil {
		return "", fmt.Errorf("failed to save fact with niche fields: %w", err)
	}

	// Index fact by canonical hash
	if fact.CanonicalHash != "" {
		take, err := k.acceptedFactMayTakeCanonicalEntry(ctx, fact.CanonicalHash, claim.Id)
		if err != nil {
			return "", err
		}
		if take {
			if err := k.SetCanonicalHash(ctx, fact.CanonicalHash, fact.Id); err != nil {
				return "", fmt.Errorf("failed to index fact by canonical hash: %w", err)
			}
		}
	}

	// Convert claim relations to fact relations and store in graph index.
	// Inference type/strength (ToK Wave 1) propagate from ClaimRelation to
	// FactRelation so proof-tree auditors can see HOW each edge was derived.
	for _, claimRel := range claim.Relations {
		factRel := &types.FactRelation{
			SourceFactId:         factID,
			TargetFactId:         claimRel.TargetFactId,
			Relation:             claimRel.Relation,
			CreatedAtBlock:       height,
			Creator:              claim.Submitter,
			Inference:            claimRel.Inference,
			InferenceStrengthBps: claimRel.InferenceStrengthBps,
		}
		if enabled, err := k.RecordIntegrityEnabled(ctx); err != nil {
			return "", err
		} else if enabled {
			factRel.MethodId = claimRel.MethodId
		}
		if err := k.SetFactRelation(ctx, factRel); err != nil {
			return "", fmt.Errorf("failed to store fact relation: %w", err)
		}

		// Track new citation for metabolism energy income
		k.IncrementNewCitationEpoch(ctx, claimRel.TargetFactId)

		// K-alpha: the chain's first giving-side acknowledgment — being cited
		// is recognition addressed to the cited fact's AUTHOR, not the fact.
		// Self-cites are emitted too, marked self="true" by the helper so
		// K-beta pricing can discount them; recognition itself skips nobody.
		// A fact with no recorded submitter yields no edge — an empty
		// beneficiary would be an unattributable credit (guard matched with
		// the substrate_bridge external-cite emitter).
		if target := citedTargets[claimRel.TargetFactId]; target != nil && target.Submitter != "" {
			emitKarmaEdge(sdkCtx, "cited", target.Submitter, claim.Submitter, target.Id, target.Domain)
		}

		sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
			"zerone.knowledge.fact_relation_created",
			sdk.NewAttribute("source", factRel.SourceFactId),
			sdk.NewAttribute("target", factRel.TargetFactId),
			sdk.NewAttribute("relation", factRel.Relation.String()),
		))
	}

	// ─── Lineage registration (reproduction) ──────────────────────────────
	// Check for reproductive relations (REFINES or GENERALIZES imply parent-child)
	for _, rel := range claim.Relations {
		if rel.Relation == types.RelationType_RELATION_TYPE_REFINES ||
			rel.Relation == types.RelationType_RELATION_TYPE_GENERALIZES {
			parentFact, found := k.GetFact(ctx, rel.TargetFactId)
			if !found {
				continue
			}

			// Check max children
			if uint64(len(parentFact.ChildFactIds)) >= params.ReproductionMaxChildren {
				k.Logger(ctx).Info("parent at max children", "parent", parentFact.Id)
				continue
			}

			// Set lineage on child
			fact.ParentFactId = parentFact.Id
			fact.LineageDepth = parentFact.LineageDepth + 1
			fact.LineageRootId = parentFact.LineageRootId
			if fact.LineageRootId == "" {
				fact.LineageRootId = parentFact.Id // parent is the root
			}

			// Inherit fitness from parent
			inheritedFitness := safeMulDiv(parentFact.FitnessScore, params.ReproductionChildFitnessInheritanceBps, 1_000_000)
			fact.FitnessScore = inheritedFitness

			// Update parent: add child, bump progeny, energy bonus
			parentFact.ChildFactIds = append(parentFact.ChildFactIds, fact.Id)
			parentFact.ProgenyCount++
			parentFact.Energy += params.ReproductionParentEnergyBonus
			if parentFact.Energy > params.MetabolismEnergyCap {
				parentFact.Energy = params.MetabolismEnergyCap
			}
			_ = k.SetFact(ctx, parentFact)

			// Propagate progeny count up the lineage
			k.PropagateProgenyCount(ctx, parentFact.ParentFactId)

			// Save updated child fact with lineage fields
			_ = k.SetFact(ctx, fact)

			sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
				"zerone.knowledge.lineage_created",
				sdk.NewAttribute("child_fact_id", fact.Id),
				sdk.NewAttribute("parent_fact_id", parentFact.Id),
				sdk.NewAttribute("lineage_depth", fmt.Sprintf("%d", fact.LineageDepth)),
				sdk.NewAttribute("inherited_fitness", fmt.Sprintf("%d", inheritedFitness)),
			))

			break // Only one parent relationship
		}
	}

	// Survival-gate: ESCROW the submitter reward until the fact survives its
	// challenge window. Nothing is minted at accept — the reward is issued only on a
	// challenge-win or an unchallenged window expiry (releaseSurvivalReward), and
	// cancelled if the fact is disproven (cancelSurvivalReward). Issuance follows
	// survival, not acceptance. The original partnership-split / direct-vesting
	// routing runs unchanged at release time (routeSubmitterReward).
	//
	// A conjecture escrows nothing. This is the single most important line in
	// the mechanism: the proposer of a question earns no reward on any path,
	// so there is no revenue to farm by asking, and the only way value moves
	// is when someone destroys the conjecture. Nor may a conjecture fill a
	// knowledge bounty — a sponsor who paid for an answer must not be settled
	// with a question.
	if !isConjecture {
		if !neutral {
			if err := k.EscrowSubmitterReward(ctx, fact, claim); err != nil {
				return "", err
			}
		}
		k.ClaimBountyForFact(ctx, fact, claim)
	}

	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		"zerone.knowledge.fact_created",
		sdk.NewAttribute("fact_id", factID),
		sdk.NewAttribute("claim_id", claim.Id),
		sdk.NewAttribute("domain", claim.Domain),
		sdk.NewAttribute("confidence", fmt.Sprintf("%d", fact.Confidence)),
	))

	return factID, nil
}

func computeProvenance(claim *types.Claim, targets map[string]*types.Fact) (uint32, uint64) {
	const bps uint64 = 1_000_000

	// Per-edge contributions: (factID → strengthBps). References default to
	// full strength; Relations use declared strength (UNSPECIFIED or 0 → BPS).
	edges := make(map[string]uint64)
	for _, ref := range claim.References {
		if ref == "" {
			continue
		}
		// Plain reference = full-strength preservation.
		if prev, ok := edges[ref]; !ok || bps > prev {
			edges[ref] = bps
		}
	}
	for _, rel := range claim.Relations {
		if rel.Relation == types.RelationType_RELATION_TYPE_CONTRADICTS {
			continue
		}
		if rel.TargetFactId == "" {
			continue
		}
		strength := rel.InferenceStrengthBps
		if strength == 0 || strength > bps {
			strength = bps
		}
		// If the same fact is cited via both References and Relations, keep
		// the stronger edge (less weakening) — the submitter declared a
		// stronger claim somewhere.
		if prev, ok := edges[rel.TargetFactId]; !ok || strength > prev {
			edges[rel.TargetFactId] = strength
		}
	}

	if len(edges) == 0 {
		return 0, 0
	}

	var minDist uint32 = ^uint32(0)
	var minFloor uint64
	var floorInitialized bool

	for factID, strength := range edges {
		target := targets[factID]
		if target == nil {
			continue
		}
		if target.AxiomDistance < minDist {
			minDist = target.AxiomDistance
		}
		// target.effective_confidence = min(own, inherited floor)
		eff := target.Confidence
		if target.DependencyConfidenceFloor > 0 && target.DependencyConfidenceFloor < eff {
			eff = target.DependencyConfidenceFloor
		}
		// Weaken by inference strength (ToK Wave 6).
		contribution := safeMulDiv(eff, strength, bps)
		if !floorInitialized || contribution < minFloor {
			minFloor = contribution
			floorInitialized = true
		}
	}

	if minDist == ^uint32(0) {
		return 0, 0
	}
	return minDist + 1, minFloor
}
