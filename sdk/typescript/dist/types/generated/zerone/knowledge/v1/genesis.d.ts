import { Fact, Claim, VerificationRound, Domain, CommonKnowledgeEntry, Methodology, NormativeCommitment, TokenizerSpec, TraceSchema, TrainingPipeline, ModelCard, TrainingAttestation, ContributionRecord, AugmentationBounty, Augmentation, ContributionChallenge, TrainingFundDisbursement, TrainingManifest, AgentCalibration } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name Params_MethodologyNormalizationBpsEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export interface Params_MethodologyNormalizationBpsEntry {
    key: string;
    value: bigint;
}
/**
 * Params are the governance parameters for the knowledge module.
 * All BPS values use a 1,000,000 scale (1,000,000 = 100%).
 * @name Params
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Params
 */
export interface Params {
    /**
     * ─── Core verification ───────────────────────────────────────────────────
     */
    minVerifiers: bigint;
    /**
     * default: 22
     */
    maxVerifiers: bigint;
    /**
     * default: 200
     */
    commitPhaseBlocks: bigint;
    /**
     * default: 200
     */
    revealPhaseBlocks: bigint;
    /**
     * default: 50
     */
    aggregationPhaseBlocks: bigint;
    /**
     * default: 50
     */
    claimCooldownBlocks: bigint;
    /**
     * ─── Confidence scoring ──────────────────────────────────────────────────
     */
    initialConfidence: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    confidenceBoostPerVerification: bigint;
    /**
     * default: 770,000 (77%) acceptance
     */
    confidenceThreshold: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    quorumThreshold: bigint;
    /**
     * ─── Verifier slashing ──────────────────────────────────────────────────
     */
    wrongVerificationSlashBps: bigint;
    /**
     * default: 100,000 (10%)
     */
    missedRevealSlashBps: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    equivocationSlashBps: bigint;
    /**
     * DEPRECATED; no longer used; runtime-immutable
     */
    invalidClaimSlashBps: bigint;
    /**
     * ─── Compatibility-only reward metadata ─────────────────────────────────
     * Round payout divides the 55% review-fee pool and does not use these values.
     * Runtime updates must preserve them.
     */
    verificationReward: string;
    verificationRewardDecayBps: bigint;
    /**
     * ─── Claim validation ────────────────────────────────────────────────────
     */
    minClaimTextLength: bigint;
    /**
     * default: 1,000
     */
    maxClaimTextLength: bigint;
    /**
     * default: "100000" (0.1 ZRN) — non-refundable review fee
     */
    minReviewFee: string;
    /**
     * ─── Adversarial verification ────────────────────────────────────────────
     */
    adversarialVerificationEnabled: boolean;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    provisionalThreshold: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    rejectThreshold: bigint;
    /**
     * default: 34,272 (1 day)
     */
    challengeDurationBlocks: bigint;
    /**
     * default: "11000000" (11 ZRN)
     */
    minChallengeStake: string;
    /**
     * compatibility-only; settlement uses fixed routing; runtime-immutable
     */
    failedChallengeSlashBps: bigint;
    /**
     * default: 300,000 (30%)
     */
    successfulChallengeRewardBps: bigint;
    /**
     * compatibility-only; not enforced; runtime-immutable
     */
    maxConcurrentChallenges: bigint;
    /**
     * ─── Citation economics ──────────────────────────────────────────────────
     */
    citationShareBps: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    crossDomainBonusBps: bigint;
    /**
     * ─── Extended governance params ─────────────────────────────────────────
     */
    maxFactsPerDomain: bigint;
    /**
     * compatibility-only; no expiry sweep reads it; runtime-immutable
     */
    factExpiryBlocks: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    crossStratumDiscountBps: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    maxValidatorsPerRound: bigint;
    /**
     * default: 1,111 blocks
     */
    confidenceGrowthEpoch: bigint;
    /**
     * default: 11,000 (1.1%)
     */
    confidenceGrowthPerEpochBps: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    maxSurvivalConfidence: bigint;
    /**
     * default: 880,000 (88%)
     */
    survivedChallengeConfidenceCap: bigint;
    /**
     * compatibility-only; not enforced; runtime-immutable
     */
    maxApprenticeValidators: bigint;
    /**
     * Compatibility-only and runtime-immutable. Review-fee routing uses a
     * hard-coded residual ~3.33% after its other fixed shares.
     */
    researchFundShareBps: bigint;
    /**
     * ─── Fitness scoring ─────────────────────────────────────────────────────
     */
    fitnessEpochBlocks: bigint;
    /**
     * Weight for query rate
     */
    fitnessWeightQueryBps: bigint;
    /**
     * Weight for citation rate
     */
    fitnessWeightCitationBps: bigint;
    /**
     * Weight for bridge score
     */
    fitnessWeightBridgeBps: bigint;
    /**
     * Weight for dependency depth
     */
    fitnessWeightDepthBps: bigint;
    /**
     * Weight for active patronage
     */
    fitnessWeightPatronBps: bigint;
    /**
     * Weight for uniqueness
     */
    fitnessWeightUniqueBps: bigint;
    /**
     * Weight for age penalty
     */
    fitnessWeightAgeBps: bigint;
    /**
     * Score assigned at birth (grace period)
     */
    fitnessInitialScore: bigint;
    /**
     * Epochs before age penalty kicks in
     */
    fitnessGraceEpochs: bigint;
    /**
     * ─── Bootstrap fund (R19-7) ────────────────────────────────────────────
     */
    bootstrapFundEnabled: boolean;
    /**
     * Max sponsored claims per address (lifetime)
     */
    bootstrapFundMaxPerAddress: string;
    /**
     * Max sponsored claims per epoch (rate limit)
     */
    bootstrapFundMaxPerEpoch: string;
    /**
     * Epoch length in blocks for rate limiting
     */
    bootstrapFundEpochBlocks: bigint;
    /**
     * Max fee the fund will cover per claim (uzrn)
     */
    bootstrapFundFeeCap: string;
    /**
     * ─── Metabolism ──────────────────────────────────────────────────────────
     */
    metabolismBaseCost: bigint;
    /**
     * Additional cost per 100 chars of content (BPS of base)
     */
    metabolismContentLengthBps: bigint;
    /**
     * Additional cost per 100 facts in domain (BPS of base)
     */
    metabolismDomainCompetitionBps: bigint;
    /**
     * Energy gained per query
     */
    metabolismEnergyPerQuery: bigint;
    /**
     * Energy gained per new citation
     */
    metabolismEnergyPerCitation: bigint;
    /**
     * Energy gained per patronage epoch
     */
    metabolismEnergyPerPatronage: bigint;
    /**
     * One-time energy for surviving challenge
     */
    metabolismEnergyChallengeSurvival: bigint;
    /**
     * Maximum energy a fact can hold
     */
    metabolismEnergyCap: bigint;
    /**
     * Starting energy for new facts
     */
    metabolismInitialEnergy: bigint;
    /**
     * Epochs at 0 energy before expiry
     */
    metabolismAtRiskEpochs: bigint;
    /**
     * Epochs after expiry before pruning
     */
    metabolismExpiredToPrunedEpochs: bigint;
    /**
     * ─── Reproduction ────────────────────────────────────────────────────
     */
    reproductionRoyaltyBps: bigint;
    /**
     * Decay per generation (BPS of previous)
     */
    reproductionRoyaltyDecayBps: bigint;
    /**
     * Max generations for royalty propagation
     */
    reproductionMaxRoyaltyDepth: bigint;
    /**
     * Energy bonus to parent when child is created
     */
    reproductionParentEnergyBonus: bigint;
    /**
     * % of parent fitness inherited by child
     */
    reproductionChildFitnessInheritanceBps: bigint;
    /**
     * Max direct children per fact
     */
    reproductionMaxChildren: bigint;
    /**
     * ─── Novelty detection ──────────────────────────────────────────────────
     */
    noveltyCommonKnowledgePenaltyBps: bigint;
    /**
     * Penalty per existing fact with same subject
     */
    noveltySubjectOverlapPenaltyBps: bigint;
    /**
     * Bonus if more precise than existing
     */
    noveltyPrecisionBonusBps: bigint;
    /**
     * Bonus if subject spans multiple domains
     */
    noveltyCrossDomainBonusBps: bigint;
    /**
     * Cap on overlap penalty (after N, no more penalty)
     */
    noveltyMaxOverlapFacts: bigint;
    /**
     * ─── Agent demand ────────────────────────────────────────────────
     */
    demandBountyThreshold: bigint;
    /**
     * Base bounty reward (uzrn)
     */
    demandBountyBaseReward: string;
    /**
     * Additional reward per unfulfilled query (uzrn)
     */
    demandBountyPerQueryBonus: string;
    /**
     * Epochs before unclaimed bounty expires
     */
    demandBountyExpiryEpochs: bigint;
    /**
     * Max demand multiplier for energy (BPS)
     */
    demandMultiplierCap: bigint;
    /**
     * Enable/disable demand tracking
     */
    demandTrackingEnabled: boolean;
    /**
     * Addresses allowed to report demand
     */
    authorizedDemandReporters: string[];
    /**
     * ─── Competition (niche dynamics) ──────────────────────────────────
     */
    competitionNicheDominanceBonusBps: bigint;
    /**
     * Below this ratio of leader fitness = redundant (BPS)
     */
    competitionRedundancyThresholdBps: bigint;
    /**
     * Max facts per niche before forced pruning
     */
    competitionMaxNicheSize: bigint;
    /**
     * Fitness bonus per SUPPORTS link to healthy fact (BPS)
     */
    competitionSymbiosisBonusBps: bigint;
    /**
     * ─── Query satisfaction ──────────────────────────────────────────────
     */
    fitnessWeightSatisfactionBps: bigint;
    /**
     * Minimum ratings before satisfaction affects fitness (default: 3)
     */
    satisfactionMinRatings: bigint;
    /**
     * ─── Consensus diversity (R28-2) ──────────────────────────────────
     */
    diversityConformityAlertThreshold: bigint;
    /**
     * Consecutive low-diversity epochs before alert (default: 3)
     */
    diversityConformityAlertEpochs: bigint;
    /**
     * ─── Retroactive vindication (R28-1) ──────────────────────────────
     */
    vindicationRefundEnabled: boolean;
    /**
     * % of majority slash pool as bonus to vindicated minority (default: 2000 = 20%)
     */
    vindicationBonusBps: bigint;
    /**
     * Slash rate for majority on disproven fact (default: 500 = 5%)
     */
    vindicationSlashBps: bigint;
    /**
     * How long escrowed entries are eligible (default: 100000)
     */
    vindicationWindowBlocks: bigint;
    /**
     * ─── Multi-level energy thresholds (R28-4) ──────────────────────────
     */
    metabolismActiveThreshold: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    metabolismExtinctionThreshold: bigint;
    /**
     * Hard cap on confidence (default: 880,000 = 88%)
     */
    maxConfidence: bigint;
    /**
     * ─── Role bonuses (R28-5) ──────────────────────────────────────────────
     */
    humanEmpiricalBonusBps: bigint;
    /**
     * +15% confidence for agent COMPUTATIONAL claims (BPS)
     */
    agentComputationalBonusBps: bigint;
    /**
     * +20% vote weight for agent verifiers (BPS)
     */
    agentVerificationBonusBps: bigint;
    /**
     * +10% energy boost for human patrons (BPS)
     */
    humanPatronageBonusBps: bigint;
    /**
     * +25% confidence for partnership claims (BPS)
     */
    dualValidationBonusBps: bigint;
    /**
     * ─── Domain carrying capacity (R29-1) ──────────────────────────────────
     */
    domainBaseCapacity: bigint;
    /**
     * Capacity bonus per inbound cross-domain citation (default: 1)
     */
    domainCapacityGrowthPerCitation: bigint;
    /**
     * Decay multiplier at 2× capacity (default: 1,500,000 = 150%)
     */
    overcrowdingDecayMultiplierBps: bigint;
    /**
     * Energy bonus for facts in sparse domains (default: 200,000 = 20%)
     */
    underpopulationBirthBonusBps: bigint;
    /**
     * ─── Epistemic temperature (R29-2) ──────────────────────────────────
     */
    epistemicTemperatureDecayBps: bigint;
    /**
     * Cooling per high-conformity epoch (default: 50,000 = 5%)
     */
    epistemicConformityCoolingBps: bigint;
    /**
     * Heating per vindication event (default: 100,000 = 10%)
     */
    epistemicVindicationHeatingBps: bigint;
    /**
     * Max confidence in cold domains (default: 600,000 = 60%)
     */
    epistemicColdConfidenceCapBps: bigint;
    /**
     * Confidence growth multiplier in hot domains (default: 1,500,000 = 150%)
     */
    epistemicHotConfidenceGrowthBps: bigint;
    /**
     * Lookback window for vindication counting (default: 10,000)
     */
    epistemicTemperatureWindowBlocks: bigint;
    /**
     * ─── Domain role elasticity (R29-3) ──────────────────────────────────
     */
    roleElasticityMinCalls: bigint;
    /**
     * Max bonus scaling (default: 2,000,000 = 200%)
     */
    roleElasticityMaxMultiplierBps: bigint;
    /**
     * Min bonus scaling (default: 500,000 = 50%)
     */
    roleElasticityMinMultiplierBps: bigint;
    /**
     * Blocks between 5% decay cycles (default: 100)
     */
    roleElasticityDecayEpochs: bigint;
    /**
     * ─── Mentorship dividends (R31-5: Water → Wood) ──────────────────────
     */
    mentorshipDividendEnergy: bigint;
    /**
     * Carrying capacity bonus per graduation (default: 5)
     */
    mentorshipCapacityBonus: bigint;
    /**
     * ─── Social verification adjustment (R31-2: Water → Fire) ──────────────
     */
    socialSaturationThreshold: bigint;
    /**
     * Lookback window for verification health metrics (default: 10000)
     */
    observationWindowBlocks: bigint;
    /**
     * ─── Consensus integrity (T1 mitigation) ────────────────────────────────
     * Minimum distinct verifier headcount that must vote with the verdict, in
     * addition to the stake-weighted ConfidenceThreshold. Prevents a single
     * large-stake coalition from promoting claims past consensus.
     */
    minHeadcountAgreement: bigint;
    /**
     * ─── Popperian challenge stake (Wave 14c: stress-test, don't shield) ────
     * Effective MinChallengeStake = base × max(floor, 1 - target.Confidence ×
     * scaling / BPS). Scales INVERSELY with target confidence: the more the
     * community trusts a claim, the cheaper it is to probe it. Truth stands
     * firm under challenge because of its nature — the chain invites the
     * testing of high-confidence facts rather than taxing it.
     * See keeper.ChallengeStakeFloorBps (10% floor) for the spam guard.
     */
    challengeConfidenceScalingBps: bigint;
    /**
     * ─── Schelling-point mitigation (T3) ────────────────────────────────────
     * Verifier reward is multiplied by (1 - conformity_bps × strength / BPS²)
     * so crowd-followers earn less than independent investigators.
     * Score source: ValidatorIndependenceScore in x/knowledge/keeper/diversity.go.
     */
    independenceRewardStrengthBps: bigint;
    /**
     * ─── Route B Wave 4: economic realignment ───────────────────────────────
     * Reformulation rounds: how many verifier votes before a verdict is final.
     */
    reformulationMinPanelVotes: bigint;
    /**
     * Fraction of verifiers that must agree to finalize a non-pending verdict.
     */
    reformulationConsensusBps: bigint;
    /**
     * SUPERIOR verdict bonus as a multiplier of reward_per_variant.
     */
    reformulationSuperiorBonusBps: bigint;
    /**
     * Percentage of remaining-bounty escrow charged as a "kept-market-open"
     * fee when the sponsor cancels or the bounty expires with unspent escrow.
     */
    augmentationExpiryFeeBps: bigint;
    /**
     * Methodology normalization table. Keys are methodology_id strings in the
     * Methodology registry; values are multipliers in BPS applied to TVW so
     * methodologies with naturally-low corroboration don't get starved.
     * Absent keys → 1,000,000 (1.0×). Set at genesis.
     */
    methodologyNormalizationBps: {
        [key: string]: bigint;
    };
    /**
     * Vindication multiplier — facts vindicated from minority status get this
     * fraction × TVW. >1.0× rewards epistemic courage.
     */
    vindicationTvwMultiplierBps: bigint;
    /**
     * Disproval clawback — after a fact goes DISPROVEN, this fraction of the
     * recent training revenue is clawed back from the submitter into the
     * research fund.
     */
    disprovalClawbackBps: bigint;
    /**
     * compatibility-only; not read; runtime-immutable
     */
    disprovalClawbackWindowEpochs: bigint;
    /**
     * Training-fund post-hoc disbursement params.
     * Retained for the release-sealed training-disbursement API; current
     * execution returns before reading these fields. Runtime-immutable.
     */
    trainingFundCalibrationFloorBps: bigint;
    trainingFundVestingEpochs: bigint;
    trainingFundMethodologyDiversityBonusBps: bigint;
    trainingFundBaseReward: string;
    /**
     * Contribution challenge — challenger bond size (uzrn as string).
     */
    contributionChallengeBond: string;
    /**
     * Compatibility-only; reward minting is disabled until a replay-safe,
     * one-shot entitlement exists. Runtime updates must preserve this field.
     */
    contributionChallengeRewardMultiplierBps: bigint;
    /**
     * Sponsor-veto forfeiture: if a sponsor vetoes a passing verdict, they
     * forfeit this fraction of the variant payout to the research fund.
     */
    sponsorVetoForfeitBps: bigint;
    /**
     * ─── Wave 14: internal-hack resilience ───────────────────────────────
     * Upper bound on any single module pause window. Even if authority
     * sets auto_unpause_at_block=0 (intending indefinite), the handler
     * caps it to this many blocks from pause-time. Forces compromised
     * authority's DoS to self-resolve within a governance-configured
     * window rather than persisting indefinitely.
     */
    maxPauseDurationBlocks: bigint;
    /**
     * ─── Wave 15: chain-driven stress-test invitation ──────────────────────
     * Truth stands firm under challenge because of its nature — and the
     * chain no longer waits for challenges to arrive. A per-block heartbeat
     * scans for high-confidence facts that have gone idle (no challenge in
     * threshold blocks) and emits invitation events, signaling to external
     * prober agents that the substrate is actively seeking stress-tests of
     * those claims. The chain manufactures demand for its own audit.
     */
    probeInvitationIdleThresholdBlocks: bigint;
    /**
     * default: 700,000 (70%) — only invite probes on facts worth testing
     */
    probeInvitationMinConfidenceBps: bigint;
    /**
     * default: 10 — max invitations emitted per block (bounds BeginBlocker work)
     */
    probeInvitationBatchSize: number;
    /**
     * default: 100_000 blocks — don't re-invite the same fact back-to-back
     */
    probeInvitationReinviteCooldown: bigint;
    /**
     * ─── Wave 15: probe bounty pool ───────────────────────────────────────
     * A purpose-built minting stream that funds probe rewards. Moves
     * epistemic-auditing funding out of general governance (protocol
     * treasury) into a dedicated pool so the chain self-sustains its own
     * audit economy. successful-probe bonuses draw from this pool first;
     * if the pool is empty, fall back to protocol treasury.
     */
    probeBountyMintPerBlock: string;
    /**
     * cap on pool balance (default "1000000000000" = 1,000,000 ZRN)
     */
    probeBountyMaxPoolSize: string;
    /**
     * ─── Wave 15b: invitation bonuses ──────────────────────────────────────
     * When a prober acts on a fact the chain has explicitly invited for
     * stress-testing, pay them a flat bonus from the probe bounty pool
     * regardless of outcome. Converts invitations from demand signals
     * into standing offers — the chain isn't just saying "please probe
     * this", it's saying "here's uzrn waiting for whoever does."
     * Paid in addition to stake refund, participation reward, and
     * success amplification.
     */
    invitationBonusAmount: string;
    /**
     * ─── Wave 16: guardian-veto window ────────────────────────────────────
     * The privileged-action log made authority abuse queryable; the
     * guardian-veto window makes it preventable. The set of addresses
     * listed in guardian_addresses can VETO authority-gated fact
     * injection during the time window between proposal and
     * materialization. Closes the persistent-authority-compromise gap
     * documented in Wave 14: the chain no longer relies on a single key
     * for actions that bypass the verifier panel.
     *
     * The single authority can still propose actions cheaply; legitimate
     * operations are not blocked. But any one of the configured
     * guardians can cancel a malicious or mistaken proposal during the
     * veto window. Trust shifts from "the authority key is honest" to
     * "no individual can unilaterally inject untruth."
     */
    guardianAddresses: string[];
    /**
     * default 0 (immediate); set > 0 to enable veto window
     */
    addFactVetoWindowBlocks: bigint;
}
/**
 * GenesisState is the genesis state of the knowledge module.
 * @name GenesisState
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    facts: Fact[];
    pendingClaims: Claim[];
    activeRounds: VerificationRound[];
    domains: Domain[];
    /**
     * Optional target module balance restored/minted during InitGenesis.
     * Protocol default is "0"; deployment artifacts must disclose any nonzero
     * allocation.
     */
    bootstrapFundAllocation: string;
    /**
     * Seeded common knowledge entries
     */
    commonKnowledge: CommonKnowledgeEntry[];
    methodologies: Methodology[];
    normativeCommitments: NormativeCommitment[];
    /**
     * current
     */
    tokenizerSpec?: TokenizerSpec;
    /**
     * historical versions
     */
    tokenizerSpecHistory: TokenizerSpec[];
    /**
     * current
     */
    traceSchema?: TraceSchema;
    /**
     * historical versions
     */
    traceSchemaHistory: TraceSchema[];
    trainingPipelines: TrainingPipeline[];
    modelCards: ModelCard[];
    trainingAttestations: TrainingAttestation[];
    contributionRecords: ContributionRecord[];
    augmentationBounties: AugmentationBounty[];
    augmentations: Augmentation[];
    contributionChallenges: ContributionChallenge[];
    trainingFundDisbursements: TrainingFundDisbursement[];
    trainingManifests: TrainingManifest[];
    /**
     * Agent calibration (Phase 5) — also Route-B-adjacent.
     */
    agentCalibrations: AgentCalibration[];
    /**
     * Training fund initial balance (uzrn, minted on InitGenesis; analogous
     * to bootstrap_fund_allocation but for the KnowledgeTrainingFund module
     * account).
     */
    trainingFundAllocation: string;
}
/**
 * @name Params_MethodologyNormalizationBpsEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export declare const Params_MethodologyNormalizationBpsEntry: {
    encode(message: Params_MethodologyNormalizationBpsEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params_MethodologyNormalizationBpsEntry;
    fromPartial(object: DeepPartial<Params_MethodologyNormalizationBpsEntry>): Params_MethodologyNormalizationBpsEntry;
};
/**
 * Params are the governance parameters for the knowledge module.
 * All BPS values use a 1,000,000 scale (1,000,000 = 100%).
 * @name Params
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * GenesisState is the genesis state of the knowledge module.
 * @name GenesisState
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
