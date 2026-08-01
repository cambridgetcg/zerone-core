package keeper

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/zerone-chain/zerone/x/knowledge/types"
)

// InitGenesis initializes the module state from a genesis state.
func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) error {
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

	for _, claim := range gs.PendingClaims {
		if claim == nil {
			continue
		}
		if err := k.SetClaim(ctx, claim); err != nil {
			return err
		}
	}

	for _, round := range gs.ActiveRounds {
		if round == nil {
			continue
		}
		if err := k.SetVerificationRound(ctx, round); err != nil {
			return err
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

	return nil
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

// ExportGenesis exports the current module state as a genesis state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
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

	var claims []*types.Claim
	k.IterateClaims(ctx, func(claim *types.Claim) bool {
		claims = append(claims, claim)
		return false
	})

	var domains []*types.Domain
	k.IterateDomains(ctx, func(domain *types.Domain) bool {
		domains = append(domains, domain)
		return false
	})

	var rounds []*types.VerificationRound
	k.IterateActiveRounds(ctx, func(round *types.VerificationRound) bool {
		rounds = append(rounds, round)
		return false
	})

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
	var contribRecords []*types.ContributionRecord
	k.IterateContributionRecords(ctx, func(r *types.ContributionRecord) bool {
		contribRecords = append(contribRecords, r)
		return false
	})
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
	}
}

// authtypesNewModuleAddress is a local shim to avoid importing authtypes at
// genesis.go top-level; declared in genesis_iterators.go.
