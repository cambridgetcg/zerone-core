package keeper

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// initGenesisLegacy preserves the InitGenesis body from
// 6a56b621082064483af1ced7718462766d9ae756 verbatim. An absent accounting_state
// takes this path directly; FX7 changes none of its helpers or live callers.
func (k Keeper) initGenesisLegacy(ctx context.Context, gs *types.GenesisState) error {
	if err := types.ValidateFactRelationGenesis(gs); err != nil {
		return err
	}
	// Check the whole pending slice before any genesis writes. The ordinary
	// module validator also checks it, but direct keeper imports must not drop
	// or partially replay an ambiguous obligation list.
	if err := types.ValidateSurvivalPendingRewards(gs.SurvivalPendingRewards); err != nil {
		return err
	}
	if err := types.ValidateGenesisRounds(gs); err != nil {
		return err
	}
	if err := types.ValidateKnowledgeHistoryGenesis(gs); err != nil {
		return err
	}
	if gs.Params != nil {
		if err := k.SetParams(ctx, gs.Params); err != nil {
			return err
		}
	}

	// Seed the methodology registry (Phase 1: methodology over statement).
	// These six methodologies plus M-LEGACY are the "bedrock" under the new
	// model — rules of truth-seeking, not truth statements.
	if err := k.SeedDefaultMethodologies(ctx); err != nil {
		return err
	}
	// Seed the normative commitment registry (Phase 6: is-ought wall).
	// Commitments are values the chain has adopted — schema-distinct from
	// facts so the chain cannot mint currency from normative claims dressed
	// as factual ones.
	if err := k.SeedDefaultCommitments(ctx); err != nil {
		return err
	}
	// Seed the default tokenizer spec v1 (Route B). Training pipelines pin
	// to a specific version for reproducibility; amendments bump version.
	if err := k.SeedDefaultTokenizerSpec(ctx); err != nil {
		return err
	}
	// Seed the default trace schema v1 alongside the tokenizer contract. A
	// fresh chain must expose a complete Route B serialization surface without
	// relying on the first caller of SeedRouteB to repair genesis.
	if err := k.SeedDefaultTraceSchema(ctx); err != nil {
		return err
	}

	for _, domain := range gs.Domains {
		if domain == nil {
			continue
		}
		if err := k.SetDomain(ctx, domain); err != nil {
			return err
		}
	}

	for _, fact := range gs.Facts {
		if fact == nil {
			continue
		}
		if err := k.SetFact(ctx, fact); err != nil {
			return err
		}
	}
	if gs.ClaimIntegrityEnabled {
		// The subject (0x32) and tag (0x33) indexes are written only when a
		// round creates a fact, and the export carries no index. Rebuild them
		// for every imported fact, in the order the live chain created them,
		// so FactsByTag and FactsBySubject answer as on the exporting chain
		// (plan §3.6, §5.5). With the flag off nothing is indexed, as on
		// knowledge 11.
		for _, fact := range factsInCreationOrder(gs) {
			if err := k.IndexFactBySubject(ctx, fact); err != nil {
				return fmt.Errorf("index imported fact %s: %w", fact.Id, err)
			}
		}
	}

	// Seed common knowledge registry
	ckEntries := gs.CommonKnowledge
	if len(ckEntries) == 0 {
		// Fresh genesis — seed from code defaults
		ckEntries = DefaultCommonKnowledgeEntries()
	}
	for _, entry := range ckEntries {
		if entry == nil {
			continue
		}
		if err := k.SetCommonKnowledgeEntry(ctx, entry); err != nil {
			return fmt.Errorf("failed to seed common knowledge entry: %w", err)
		}
	}

	// Fund the bootstrap module account up to the declared genesis allocation
	// (R19-7). The field is exported as the module account's existing bank
	// balance, and x/bank imports that balance before x/knowledge runs. Treating
	// it as an additive mint would therefore double the fund on every
	// export/import restart. Fresh genesis files still start from a zero module
	// balance and receive the full allocation.
	if err := k.ensureGenesisFundBalance(ctx, types.BootstrapFundModuleName, gs.BootstrapFundAllocation); err != nil {
		return fmt.Errorf("failed to fund bootstrap fund: %w", err)
	}

	// ─── Route B Wave 8: import Route B state (survivable upgrades) ──────
	// Order matters: singletons first, then registries, then records that
	// depend on them. Seed calls above are idempotent — they only run when
	// the exported state is empty (fresh genesis). If the genesis payload
	// carries explicit methodologies/commitments/specs, we overwrite the
	// seeded defaults so an upgrade preserves exact prior state.

	for _, m := range gs.Methodologies {
		if m != nil {
			if err := k.SetMethodology(ctx, m); err != nil {
				return fmt.Errorf("import methodology %s: %w", m.Id, err)
			}
		}
	}
	for _, c := range gs.NormativeCommitments {
		if c != nil {
			if err := k.SetNormativeCommitment(ctx, c); err != nil {
				return fmt.Errorf("import commitment %s: %w", c.Id, err)
			}
		}
	}
	// Historical tokenizer specs BEFORE current, so current overwrites
	// singleton and history fills out.
	for _, s := range gs.TokenizerSpecHistory {
		if s != nil {
			if err := k.SetTokenizerSpecHistory(ctx, s); err != nil {
				return fmt.Errorf("import tokenizer_spec v%d history: %w", s.Version, err)
			}
		}
	}
	if gs.TokenizerSpec != nil {
		if err := k.SetTokenizerSpec(ctx, gs.TokenizerSpec); err != nil {
			return fmt.Errorf("import tokenizer_spec current: %w", err)
		}
	}
	for _, s := range gs.TraceSchemaHistory {
		if s != nil {
			if err := k.SetTraceSchemaHistory(ctx, s); err != nil {
				return fmt.Errorf("import trace_schema v%d history: %w", s.Version, err)
			}
		}
	}
	if gs.TraceSchema != nil {
		if err := k.SetTraceSchema(ctx, gs.TraceSchema); err != nil {
			return fmt.Errorf("import trace_schema current: %w", err)
		}
	}

	for _, p := range gs.TrainingPipelines {
		if p != nil {
			if err := k.SetTrainingPipeline(ctx, p); err != nil {
				return fmt.Errorf("import pipeline %s: %w", p.Id, err)
			}
		}
	}
	for _, card := range gs.ModelCards {
		if card != nil {
			if err := k.SetModelCard(ctx, card); err != nil {
				return fmt.Errorf("import model_card %s: %w", card.Id, err)
			}
		}
	}
	for _, a := range gs.TrainingAttestations {
		if a != nil {
			if err := k.SetTrainingAttestation(ctx, a); err != nil {
				return fmt.Errorf("import attestation %s: %w", a.PipelineId, err)
			}
		}
	}
	for _, cr := range gs.ContributionRecords {
		if cr != nil {
			if err := k.SetContributionRecord(ctx, cr); err != nil {
				return fmt.Errorf("import contribution record %s: %w", cr.ModelId, err)
			}
		}
	}
	for _, b := range gs.AugmentationBounties {
		if b != nil {
			if err := k.SetAugmentationBounty(ctx, b); err != nil {
				return fmt.Errorf("import bounty %s: %w", b.Id, err)
			}
		}
	}
	for _, aug := range gs.Augmentations {
		if aug != nil {
			if err := k.SetAugmentation(ctx, aug); err != nil {
				return fmt.Errorf("import augmentation %s: %w", aug.Id, err)
			}
		}
	}
	for _, ch := range gs.ContributionChallenges {
		if ch != nil {
			if err := k.SetContributionChallenge(ctx, ch); err != nil {
				return fmt.Errorf("import challenge %s: %w", ch.Id, err)
			}
		}
	}
	for _, d := range gs.TrainingFundDisbursements {
		if d != nil {
			if err := k.SetTrainingFundDisbursement(ctx, d); err != nil {
				return fmt.Errorf("import disbursement %s: %w", d.Id, err)
			}
		}
	}
	for _, m := range gs.TrainingManifests {
		if m != nil {
			if err := k.SetTrainingManifest(ctx, m); err != nil {
				return fmt.Errorf("import manifest %s: %w", m.ManifestId, err)
			}
		}
	}
	for _, c := range gs.AgentCalibrations {
		if c != nil {
			if err := k.SetAgentCalibration(ctx, c); err != nil {
				return fmt.Errorf("import calibration %s: %w", c.Address, err)
			}
		}
	}

	// As with the bootstrap fund, this is a target balance rather than an
	// additive mint so exported bank balances round-trip without inflation.
	if err := k.ensureGenesisFundBalance(ctx, types.TrainingFundModuleName, gs.TrainingFundAllocation); err != nil {
		return fmt.Errorf("failed to fund training fund: %w", err)
	}

	// Load doctrine Facts (SL-M1): every commitment in every doctrine
	// becomes a verified Fact under domain=doctrine_*. Cross-doctrine
	// "Echoes:" become real SUPPORTS/REQUIRES/REFINES edges.
	if err := k.LoadDoctrineFacts(ctx); err != nil {
		return fmt.Errorf("load doctrine facts: %w", err)
	}
	if err := k.ImportFactRelationGenesis(ctx, gs); err != nil {
		return fmt.Errorf("import fact relations: %w", err)
	}
	for _, reward := range gs.SurvivalPendingRewards {
		if err := k.SetSurvivalPendingReward(ctx, SurvivalPendingReward{
			ClaimId: reward.ClaimId, FactId: reward.FactId,
			Recipient: reward.Recipient, Amount: reward.Amount,
			Category: reward.Category, PartnershipId: reward.PartnershipId,
			Deadline: reward.Deadline,
		}); err != nil {
			return fmt.Errorf("import survival pending reward %s: %w", reward.FactId, err)
		}
	}

	// Loading facts may have emitted transient initial history. Replace it with
	// exactly the supplied history, including explicit absence, before enabling
	// new execution semantics. This does not manufacture past review evidence.
	if err := k.ImportKnowledgeHistory(ctx, gs.StatusTransitions, gs.CascadeEvents, gs.StatusTransitionCounters); err != nil {
		return err
	}
	if gs.RecordIntegrityEnabled {
		if err := k.EnableRecordIntegrity(ctx); err != nil {
			return err
		}
	}
	if gs.ReviewNeutralityEnabled {
		if err := k.EnableReviewNeutrality(ctx); err != nil {
			return err
		}
	}
	if gs.ClaimRecordsEnabled {
		if err := k.EnableClaimRecords(ctx); err != nil {
			return err
		}
	}
	if gs.FundSettlementEnabled {
		if err := k.EnableFundSettlement(ctx); err != nil {
			return err
		}
	}
	if gs.ClaimIntegrityEnabled {
		if err := k.EnableClaimIntegrity(ctx); err != nil {
			return err
		}
	}
	// Only an explicit true writes the marker; absence keeps legacy semantics.
	if gs.ReformulationVindicationOrderingEnabled {
		if err := k.EnableReformulationVindicationOrdering(ctx); err != nil {
			return err
		}
	}
	if gs.InconclusiveRoundAccountingEnabled {
		if err := k.EnableInconclusiveRoundAccounting(ctx); err != nil {
			return err
		}
	}
	if gs.CalibrationSmoothingEnabled {
		if err := k.EnableCalibrationSmoothing(ctx); err != nil {
			return err
		}
	}
	if gs.ZeroConfidenceFloorEnabled {
		if err := k.EnableZeroConfidenceFloor(ctx); err != nil {
			return err
		}
	}
	claims := gs.PendingClaims
	if gs.ClaimIntegrityEnabled {
		// Each claim's first save below takes its dedup entries, and the file
		// lists claims in ID order. Replay them in admission order so every
		// entry names the latest admitted record again (plan §5.4). With the
		// flag off the file order is kept, as on knowledge 11.
		claims = claimsInAdmissionOrder(gs.PendingClaims)
	}
	for _, claim := range claims {
		if claim == nil {
			continue
		}
		if err := k.SetClaim(ctx, claim); err != nil {
			return err
		}
	}
	if err := k.importVerificationRounds(ctx, gs); err != nil {
		return err
	}
	if err := k.RebuildPendingVerifierRewardIndex(ctx); err != nil {
		return err
	}
	if gs.ClaimIntegrityEnabled {
		// A relaunch from a knowledge-11 export carries the fact edges that
		// equal-ID augmentations replaced, and no migration runs on it.
		// Restore them as Migrate11to12 does, from the imported facts, claims,
		// rounds and augmentations. With the flag off nothing is restored, as
		// on knowledge 11. BaseApp.InitChain discards the events emitted
		// here, so the node log is the import's record of each pair.
		restoration, err := k.RestoreAugmentationReplacedEdges(ctx)
		if err != nil {
			return fmt.Errorf("restore augmentation-replaced fact edges: %w", err)
		}
		k.logAugmentationEdgeRestoration(ctx, "genesis", restoration)
	}

	return nil
}

// claimsInAdmissionOrder returns the non-nil claims stably sorted by
// SubmittedAtBlock, then ID. The input slice is not modified.
func claimsInAdmissionOrder(claims []*types.Claim) []*types.Claim {
	ordered := make([]*types.Claim, 0, len(claims))
	for _, claim := range claims {
		if claim != nil {
			ordered = append(ordered, claim)
		}
	}
	slices.SortStableFunc(ordered, func(a, b *types.Claim) int {
		if c := cmp.Compare(a.SubmittedAtBlock, b.SubmittedAtBlock); c != 0 {
			return c
		}
		return cmp.Compare(a.Id, b.Id)
	})
	return ordered
}

// factsInCreationOrder returns the facts that carry a structure (the only ones
// IndexFactBySubject indexes), stably sorted into the order a live chain
// creates them: by VerifiedAtBlock, a round-created fact's creation height,
// which nothing rewrites; then by the round ID its claim names, because
// AdvanceRoundPhases completes one block's rounds in round-ID order; then by
// fact ID. Indexing in this order leaves each single-valued subject entry
// naming the fact created last, as on the live chain. A fact whose claim is
// not in the file sorts before the claimed facts of its block. The input
// slice is not modified; the map is only read.
func factsInCreationOrder(gs *types.GenesisState) []*types.Fact {
	roundOfClaim := make(map[string]string, len(gs.PendingClaims))
	for _, claim := range gs.PendingClaims {
		if claim != nil {
			roundOfClaim[claim.Id] = claim.VerificationRoundId
		}
	}
	ordered := make([]*types.Fact, 0, len(gs.Facts))
	for _, fact := range gs.Facts {
		if fact != nil && fact.Structure != nil {
			ordered = append(ordered, fact)
		}
	}
	slices.SortStableFunc(ordered, func(a, b *types.Fact) int {
		if c := cmp.Compare(a.VerifiedAtBlock, b.VerifiedAtBlock); c != 0 {
			return c
		}
		if c := cmp.Compare(roundOfClaim[a.ClaimId], roundOfClaim[b.ClaimId]); c != 0 {
			return c
		}
		return cmp.Compare(a.Id, b.Id)
	})
	return ordered
}

// ensureGenesisFundBalance brings a module account up to its declared genesis
// allocation without reminting coins that x/bank already restored. This keeps
// the historical fresh-genesis contract (a zero balance receives the full
// allocation) while making old exported genesis files safe to import: those
// files contain both the bank balance and the same balance-as-allocation value.
func (k Keeper) ensureGenesisFundBalance(ctx context.Context, moduleName, allocation string) error {
	if allocation == "" {
		return nil
	}
	target, ok := sdkmath.NewIntFromString(allocation)
	if !ok || target.IsNegative() || target.String() != allocation {
		return fmt.Errorf("allocation must be a canonical non-negative base-10 integer within 256 bits")
	}
	if k.bankKeeper == nil || target.IsZero() {
		return nil
	}

	moduleAddr := sdk.AccAddress(authtypesNewModuleAddress(moduleName))
	current := k.bankKeeper.GetBalance(ctx, moduleAddr, "uzrn").Amount
	if current.GTE(target) {
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin("uzrn", target.Sub(current)))
	return k.bankKeeper.MintCoins(ctx, moduleName, coins)
}

// exportGenesisLegacy preserves the ExportGenesis body from
// 6a56b621082064483af1ced7718462766d9ae756 verbatim. The genesis-only wrapper
// adds the primary accounting inventory without changing live behavior.
func (k Keeper) exportGenesisLegacy(ctx context.Context) *types.GenesisState {
	pending, pendingErr := k.GetAllSurvivalPendingRewards(ctx)
	if pendingErr != nil {
		// The module export interface has no error result. Refuse the export
		// rather than producing an apparently complete file that loses claims.
		panic(fmt.Errorf("export survival pending rewards: %w", pendingErr))
	}
	survivalRewards := make([]*types.SurvivalPendingReward, 0, len(pending))
	for _, reward := range pending {
		survivalRewards = append(survivalRewards, &types.SurvivalPendingReward{
			ClaimId: reward.ClaimId, FactId: reward.FactId,
			Recipient: reward.Recipient, Amount: reward.Amount,
			Category: reward.Category, PartnershipId: reward.PartnershipId,
			Deadline: reward.Deadline,
		})
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		p := types.DefaultParams()
		params = &p
	}

	var facts []*types.Fact
	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		facts = append(facts, fact)
		return false
	})
	relationState, err := k.ExportFactRelationGenesis(ctx)
	if err != nil {
		panic(fmt.Errorf("export fact relations: %w", err))
	}
	if err := types.ValidateFactRelationGenesis(&types.GenesisState{Facts: facts, FactRelationState: relationState}); err != nil {
		panic(fmt.Errorf("export fact relation references: %w", err))
	}

	claims, err := k.GetAllClaimsChecked(ctx)
	if err != nil {
		panic(fmt.Errorf("export knowledge claims: %w", err))
	}

	var domains []*types.Domain
	k.IterateDomains(ctx, func(domain *types.Domain) bool {
		domains = append(domains, domain)
		return false
	})

	enabled, err := k.RecordIntegrityEnabled(ctx)
	if err != nil {
		panic(err)
	}
	neutral, err := k.ReviewNeutralityEnabled(ctx)
	if err != nil {
		panic(err)
	}
	claimRecords, err := k.ClaimRecordsEnabled(ctx)
	if err != nil {
		panic(err)
	}
	fundSettlement, err := k.FundSettlementEnabled(ctx)
	if err != nil {
		panic(err)
	}
	claimIntegrity, err := k.ClaimIntegrityEnabled(ctx)
	if err != nil {
		panic(err)
	}
	reformulationVindicationOrdering, err := k.ReformulationVindicationOrderingEnabled(ctx)
	if err != nil {
		panic(err)
	}
	inconclusiveRoundAccounting, err := k.InconclusiveRoundAccountingEnabled(ctx)
	if err != nil {
		panic(err)
	}
	calibrationSmoothing, err := k.CalibrationSmoothingEnabled(ctx)
	if err != nil {
		panic(err)
	}
	zeroConfidenceFloor, err := k.ZeroConfidenceFloorEnabled(ctx)
	if err != nil {
		panic(err)
	}
	transitions, cascades, historyCounters, err := k.ExportKnowledgeHistory(ctx)
	if err != nil {
		panic(fmt.Errorf("export knowledge history: %w", err))
	}
	allRounds, err := k.GetAllVerificationRoundsChecked(ctx)
	if err != nil {
		panic(fmt.Errorf("export verification rounds: %w", err))
	}
	var rounds, completedRounds []*types.VerificationRound
	for _, round := range allRounds {
		if round.Phase == types.VerificationPhase_VERIFICATION_PHASE_COMPLETE || round.Phase == types.VerificationPhase_VERIFICATION_PHASE_EXPIRED {
			completedRounds = append(completedRounds, round)
		} else {
			rounds = append(rounds, round)
		}
	}
	if err := types.ValidateGenesisRounds(&types.GenesisState{PendingClaims: claims, ActiveRounds: rounds, CompletedRounds: completedRounds, RecordIntegrityEnabled: enabled, ReviewNeutralityEnabled: neutral, ClaimRecordsEnabled: claimRecords, FundSettlementEnabled: fundSettlement, ClaimIntegrityEnabled: claimIntegrity}); err != nil {
		panic(fmt.Errorf("export knowledge round references: %w", err))
	}

	// Export bootstrap fund balance as allocation (for restart)
	fundBalance := k.GetBootstrapFundBalance(ctx)
	allocation := fundBalance.Amount.String()

	// Export common knowledge entries
	commonKnowledge := k.GetAllCommonKnowledge(ctx)

	// ─── Route B Wave 8: full state export ───────────────────────────────
	var methodologies []*types.Methodology
	k.IterateMethodologies(ctx, func(m *types.Methodology) bool {
		methodologies = append(methodologies, m)
		return false
	})

	commitments := k.GetAllNormativeCommitments(ctx)

	var tokenizerSpec *types.TokenizerSpec
	if s, ok := k.GetTokenizerSpec(ctx); ok {
		tokenizerSpec = s
	}
	var tokenizerHistory []*types.TokenizerSpec
	k.IterateTokenizerSpecHistory(ctx, func(s *types.TokenizerSpec) bool {
		tokenizerHistory = append(tokenizerHistory, s)
		return false
	})

	var traceSchema *types.TraceSchema
	if s, ok := k.GetTraceSchema(ctx); ok {
		traceSchema = s
	}
	var traceHistory []*types.TraceSchema
	k.IterateTraceSchemaHistory(ctx, func(s *types.TraceSchema) bool {
		traceHistory = append(traceHistory, s)
		return false
	})

	var pipelines []*types.TrainingPipeline
	k.IterateTrainingPipelines(ctx, func(p *types.TrainingPipeline) bool {
		pipelines = append(pipelines, p)
		return false
	})
	var cards []*types.ModelCard
	k.IterateModelCards(ctx, func(c *types.ModelCard) bool {
		cards = append(cards, c)
		return false
	})
	var attestations []*types.TrainingAttestation
	k.IterateTrainingAttestations(ctx, func(a *types.TrainingAttestation) bool {
		attestations = append(attestations, a)
		return false
	})
	contribRecords, contributionErr := k.GetAllContributionRecordsChecked(ctx)
	if contributionErr != nil {
		panic(fmt.Errorf("export contribution records: %w", contributionErr))
	}
	var bounties []*types.AugmentationBounty
	k.IterateAugmentationBounties(ctx, func(b *types.AugmentationBounty) bool {
		bounties = append(bounties, b)
		return false
	})
	var augmentations []*types.Augmentation
	k.IterateAugmentations(ctx, func(a *types.Augmentation) bool {
		augmentations = append(augmentations, a)
		return false
	})
	var challenges []*types.ContributionChallenge
	k.IterateAllContributionChallenges(ctx, func(c *types.ContributionChallenge) bool {
		challenges = append(challenges, c)
		return false
	})
	var disbursements []*types.TrainingFundDisbursement
	k.IterateTrainingFundDisbursements(ctx, func(d *types.TrainingFundDisbursement) bool {
		disbursements = append(disbursements, d)
		return false
	})
	var manifests []*types.TrainingManifest
	k.IterateTrainingManifests(ctx, func(m *types.TrainingManifest) bool {
		manifests = append(manifests, m)
		return false
	})
	var calibrations []*types.AgentCalibration
	k.IterateAgentCalibrations(ctx, func(c *types.AgentCalibration) bool {
		calibrations = append(calibrations, c)
		return false
	})

	// Training fund balance — round-trip the module account.
	var trainingFundAllocation string
	if k.bankKeeper != nil {
		addr := sdk.AccAddress(authtypesNewModuleAddress(types.TrainingFundModuleName))
		bal := k.bankKeeper.GetBalance(ctx, addr, "uzrn")
		trainingFundAllocation = bal.Amount.String()
	}

	return &types.GenesisState{
		Params:                    params,
		Facts:                     facts,
		FactRelationState:         relationState,
		PendingClaims:             claims,
		Domains:                   domains,
		ActiveRounds:              rounds,
		BootstrapFundAllocation:   allocation,
		CommonKnowledge:           commonKnowledge,
		Methodologies:             methodologies,
		NormativeCommitments:      commitments,
		TokenizerSpec:             tokenizerSpec,
		TokenizerSpecHistory:      tokenizerHistory,
		TraceSchema:               traceSchema,
		TraceSchemaHistory:        traceHistory,
		TrainingPipelines:         pipelines,
		ModelCards:                cards,
		TrainingAttestations:      attestations,
		ContributionRecords:       contribRecords,
		AugmentationBounties:      bounties,
		Augmentations:             augmentations,
		ContributionChallenges:    challenges,
		TrainingFundDisbursements: disbursements,
		TrainingManifests:         manifests,
		AgentCalibrations:         calibrations,
		TrainingFundAllocation:    trainingFundAllocation,
		SurvivalPendingRewards:    survivalRewards,
		RecordIntegrityEnabled:    enabled,
		ReviewNeutralityEnabled:   neutral,
		ClaimRecordsEnabled:       claimRecords,
		FundSettlementEnabled:     fundSettlement,
		ClaimIntegrityEnabled:     claimIntegrity,
		CompletedRounds:           completedRounds,
		StatusTransitions:         transitions,
		CascadeEvents:             cascades,
		StatusTransitionCounters:  historyCounters,

		ReformulationVindicationOrderingEnabled: reformulationVindicationOrdering,
		InconclusiveRoundAccountingEnabled:      inconclusiveRoundAccounting,
		CalibrationSmoothingEnabled:             calibrationSmoothing,
		ZeroConfidenceFloorEnabled:              zeroConfidenceFloor,
	}
}

// authtypesNewModuleAddress is a local shim to avoid importing authtypes at
// genesis.go top-level; declared in genesis_iterators.go.
