//@ts-nocheck
import { Fact, Claim, VerificationRound, Domain, CommonKnowledgeEntry, Methodology, NormativeCommitment, TokenizerSpec, TraceSchema, TrainingPipeline, ModelCard, TrainingAttestation, ContributionRecord, AugmentationBounty, Augmentation, ContributionChallenge, TrainingFundDisbursement, TrainingManifest, AgentCalibration } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
function createBaseParams_MethodologyNormalizationBpsEntry(): Params_MethodologyNormalizationBpsEntry {
  return {
    key: "",
    value: BigInt(0)
  };
}
/**
 * @name Params_MethodologyNormalizationBpsEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export const Params_MethodologyNormalizationBpsEntry = {
  encode(message: Params_MethodologyNormalizationBpsEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== BigInt(0)) {
      writer.uint32(16).uint64(message.value);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Params_MethodologyNormalizationBpsEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams_MethodologyNormalizationBpsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Params_MethodologyNormalizationBpsEntry>): Params_MethodologyNormalizationBpsEntry {
    const message = createBaseParams_MethodologyNormalizationBpsEntry();
    message.key = object.key ?? "";
    message.value = object.value !== undefined && object.value !== null ? BigInt(object.value.toString()) : BigInt(0);
    return message;
  }
};
function createBaseParams(): Params {
  return {
    minVerifiers: BigInt(0),
    maxVerifiers: BigInt(0),
    commitPhaseBlocks: BigInt(0),
    revealPhaseBlocks: BigInt(0),
    aggregationPhaseBlocks: BigInt(0),
    claimCooldownBlocks: BigInt(0),
    initialConfidence: BigInt(0),
    confidenceBoostPerVerification: BigInt(0),
    confidenceThreshold: BigInt(0),
    quorumThreshold: BigInt(0),
    wrongVerificationSlashBps: BigInt(0),
    missedRevealSlashBps: BigInt(0),
    equivocationSlashBps: BigInt(0),
    invalidClaimSlashBps: BigInt(0),
    verificationReward: "",
    verificationRewardDecayBps: BigInt(0),
    minClaimTextLength: BigInt(0),
    maxClaimTextLength: BigInt(0),
    minReviewFee: "",
    adversarialVerificationEnabled: false,
    provisionalThreshold: BigInt(0),
    rejectThreshold: BigInt(0),
    challengeDurationBlocks: BigInt(0),
    minChallengeStake: "",
    failedChallengeSlashBps: BigInt(0),
    successfulChallengeRewardBps: BigInt(0),
    maxConcurrentChallenges: BigInt(0),
    citationShareBps: BigInt(0),
    crossDomainBonusBps: BigInt(0),
    maxFactsPerDomain: BigInt(0),
    factExpiryBlocks: BigInt(0),
    crossStratumDiscountBps: BigInt(0),
    maxValidatorsPerRound: BigInt(0),
    confidenceGrowthEpoch: BigInt(0),
    confidenceGrowthPerEpochBps: BigInt(0),
    maxSurvivalConfidence: BigInt(0),
    survivedChallengeConfidenceCap: BigInt(0),
    maxApprenticeValidators: BigInt(0),
    researchFundShareBps: BigInt(0),
    fitnessEpochBlocks: BigInt(0),
    fitnessWeightQueryBps: BigInt(0),
    fitnessWeightCitationBps: BigInt(0),
    fitnessWeightBridgeBps: BigInt(0),
    fitnessWeightDepthBps: BigInt(0),
    fitnessWeightPatronBps: BigInt(0),
    fitnessWeightUniqueBps: BigInt(0),
    fitnessWeightAgeBps: BigInt(0),
    fitnessInitialScore: BigInt(0),
    fitnessGraceEpochs: BigInt(0),
    bootstrapFundEnabled: false,
    bootstrapFundMaxPerAddress: "",
    bootstrapFundMaxPerEpoch: "",
    bootstrapFundEpochBlocks: BigInt(0),
    bootstrapFundFeeCap: "",
    metabolismBaseCost: BigInt(0),
    metabolismContentLengthBps: BigInt(0),
    metabolismDomainCompetitionBps: BigInt(0),
    metabolismEnergyPerQuery: BigInt(0),
    metabolismEnergyPerCitation: BigInt(0),
    metabolismEnergyPerPatronage: BigInt(0),
    metabolismEnergyChallengeSurvival: BigInt(0),
    metabolismEnergyCap: BigInt(0),
    metabolismInitialEnergy: BigInt(0),
    metabolismAtRiskEpochs: BigInt(0),
    metabolismExpiredToPrunedEpochs: BigInt(0),
    reproductionRoyaltyBps: BigInt(0),
    reproductionRoyaltyDecayBps: BigInt(0),
    reproductionMaxRoyaltyDepth: BigInt(0),
    reproductionParentEnergyBonus: BigInt(0),
    reproductionChildFitnessInheritanceBps: BigInt(0),
    reproductionMaxChildren: BigInt(0),
    noveltyCommonKnowledgePenaltyBps: BigInt(0),
    noveltySubjectOverlapPenaltyBps: BigInt(0),
    noveltyPrecisionBonusBps: BigInt(0),
    noveltyCrossDomainBonusBps: BigInt(0),
    noveltyMaxOverlapFacts: BigInt(0),
    demandBountyThreshold: BigInt(0),
    demandBountyBaseReward: "",
    demandBountyPerQueryBonus: "",
    demandBountyExpiryEpochs: BigInt(0),
    demandMultiplierCap: BigInt(0),
    demandTrackingEnabled: false,
    authorizedDemandReporters: [],
    competitionNicheDominanceBonusBps: BigInt(0),
    competitionRedundancyThresholdBps: BigInt(0),
    competitionMaxNicheSize: BigInt(0),
    competitionSymbiosisBonusBps: BigInt(0),
    fitnessWeightSatisfactionBps: BigInt(0),
    satisfactionMinRatings: BigInt(0),
    diversityConformityAlertThreshold: BigInt(0),
    diversityConformityAlertEpochs: BigInt(0),
    vindicationRefundEnabled: false,
    vindicationBonusBps: BigInt(0),
    vindicationSlashBps: BigInt(0),
    vindicationWindowBlocks: BigInt(0),
    metabolismActiveThreshold: BigInt(0),
    metabolismExtinctionThreshold: BigInt(0),
    maxConfidence: BigInt(0),
    humanEmpiricalBonusBps: BigInt(0),
    agentComputationalBonusBps: BigInt(0),
    agentVerificationBonusBps: BigInt(0),
    humanPatronageBonusBps: BigInt(0),
    dualValidationBonusBps: BigInt(0),
    domainBaseCapacity: BigInt(0),
    domainCapacityGrowthPerCitation: BigInt(0),
    overcrowdingDecayMultiplierBps: BigInt(0),
    underpopulationBirthBonusBps: BigInt(0),
    epistemicTemperatureDecayBps: BigInt(0),
    epistemicConformityCoolingBps: BigInt(0),
    epistemicVindicationHeatingBps: BigInt(0),
    epistemicColdConfidenceCapBps: BigInt(0),
    epistemicHotConfidenceGrowthBps: BigInt(0),
    epistemicTemperatureWindowBlocks: BigInt(0),
    roleElasticityMinCalls: BigInt(0),
    roleElasticityMaxMultiplierBps: BigInt(0),
    roleElasticityMinMultiplierBps: BigInt(0),
    roleElasticityDecayEpochs: BigInt(0),
    mentorshipDividendEnergy: BigInt(0),
    mentorshipCapacityBonus: BigInt(0),
    socialSaturationThreshold: BigInt(0),
    observationWindowBlocks: BigInt(0),
    minHeadcountAgreement: BigInt(0),
    challengeConfidenceScalingBps: BigInt(0),
    independenceRewardStrengthBps: BigInt(0),
    reformulationMinPanelVotes: BigInt(0),
    reformulationConsensusBps: BigInt(0),
    reformulationSuperiorBonusBps: BigInt(0),
    augmentationExpiryFeeBps: BigInt(0),
    methodologyNormalizationBps: {},
    vindicationTvwMultiplierBps: BigInt(0),
    disprovalClawbackBps: BigInt(0),
    disprovalClawbackWindowEpochs: BigInt(0),
    trainingFundCalibrationFloorBps: BigInt(0),
    trainingFundVestingEpochs: BigInt(0),
    trainingFundMethodologyDiversityBonusBps: BigInt(0),
    trainingFundBaseReward: "",
    contributionChallengeBond: "",
    contributionChallengeRewardMultiplierBps: BigInt(0),
    sponsorVetoForfeitBps: BigInt(0),
    maxPauseDurationBlocks: BigInt(0),
    probeInvitationIdleThresholdBlocks: BigInt(0),
    probeInvitationMinConfidenceBps: BigInt(0),
    probeInvitationBatchSize: 0,
    probeInvitationReinviteCooldown: BigInt(0),
    probeBountyMintPerBlock: "",
    probeBountyMaxPoolSize: "",
    invitationBonusAmount: "",
    guardianAddresses: [],
    addFactVetoWindowBlocks: BigInt(0)
  };
}
/**
 * Params are the governance parameters for the knowledge module.
 * All BPS values use a 1,000,000 scale (1,000,000 = 100%).
 * @name Params
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.knowledge.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.minVerifiers !== BigInt(0)) {
      writer.uint32(8).uint64(message.minVerifiers);
    }
    if (message.maxVerifiers !== BigInt(0)) {
      writer.uint32(16).uint64(message.maxVerifiers);
    }
    if (message.commitPhaseBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.commitPhaseBlocks);
    }
    if (message.revealPhaseBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.revealPhaseBlocks);
    }
    if (message.aggregationPhaseBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.aggregationPhaseBlocks);
    }
    if (message.claimCooldownBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.claimCooldownBlocks);
    }
    if (message.initialConfidence !== BigInt(0)) {
      writer.uint32(56).uint64(message.initialConfidence);
    }
    if (message.confidenceBoostPerVerification !== BigInt(0)) {
      writer.uint32(64).uint64(message.confidenceBoostPerVerification);
    }
    if (message.confidenceThreshold !== BigInt(0)) {
      writer.uint32(72).uint64(message.confidenceThreshold);
    }
    if (message.quorumThreshold !== BigInt(0)) {
      writer.uint32(80).uint64(message.quorumThreshold);
    }
    if (message.wrongVerificationSlashBps !== BigInt(0)) {
      writer.uint32(88).uint64(message.wrongVerificationSlashBps);
    }
    if (message.missedRevealSlashBps !== BigInt(0)) {
      writer.uint32(96).uint64(message.missedRevealSlashBps);
    }
    if (message.equivocationSlashBps !== BigInt(0)) {
      writer.uint32(104).uint64(message.equivocationSlashBps);
    }
    if (message.invalidClaimSlashBps !== BigInt(0)) {
      writer.uint32(112).uint64(message.invalidClaimSlashBps);
    }
    if (message.verificationReward !== "") {
      writer.uint32(122).string(message.verificationReward);
    }
    if (message.verificationRewardDecayBps !== BigInt(0)) {
      writer.uint32(128).uint64(message.verificationRewardDecayBps);
    }
    if (message.minClaimTextLength !== BigInt(0)) {
      writer.uint32(136).uint64(message.minClaimTextLength);
    }
    if (message.maxClaimTextLength !== BigInt(0)) {
      writer.uint32(144).uint64(message.maxClaimTextLength);
    }
    if (message.minReviewFee !== "") {
      writer.uint32(154).string(message.minReviewFee);
    }
    if (message.adversarialVerificationEnabled === true) {
      writer.uint32(160).bool(message.adversarialVerificationEnabled);
    }
    if (message.provisionalThreshold !== BigInt(0)) {
      writer.uint32(168).uint64(message.provisionalThreshold);
    }
    if (message.rejectThreshold !== BigInt(0)) {
      writer.uint32(176).uint64(message.rejectThreshold);
    }
    if (message.challengeDurationBlocks !== BigInt(0)) {
      writer.uint32(184).uint64(message.challengeDurationBlocks);
    }
    if (message.minChallengeStake !== "") {
      writer.uint32(194).string(message.minChallengeStake);
    }
    if (message.failedChallengeSlashBps !== BigInt(0)) {
      writer.uint32(200).uint64(message.failedChallengeSlashBps);
    }
    if (message.successfulChallengeRewardBps !== BigInt(0)) {
      writer.uint32(208).uint64(message.successfulChallengeRewardBps);
    }
    if (message.maxConcurrentChallenges !== BigInt(0)) {
      writer.uint32(216).uint64(message.maxConcurrentChallenges);
    }
    if (message.citationShareBps !== BigInt(0)) {
      writer.uint32(224).uint64(message.citationShareBps);
    }
    if (message.crossDomainBonusBps !== BigInt(0)) {
      writer.uint32(232).uint64(message.crossDomainBonusBps);
    }
    if (message.maxFactsPerDomain !== BigInt(0)) {
      writer.uint32(240).uint64(message.maxFactsPerDomain);
    }
    if (message.factExpiryBlocks !== BigInt(0)) {
      writer.uint32(248).uint64(message.factExpiryBlocks);
    }
    if (message.crossStratumDiscountBps !== BigInt(0)) {
      writer.uint32(256).uint64(message.crossStratumDiscountBps);
    }
    if (message.maxValidatorsPerRound !== BigInt(0)) {
      writer.uint32(272).uint64(message.maxValidatorsPerRound);
    }
    if (message.confidenceGrowthEpoch !== BigInt(0)) {
      writer.uint32(304).uint64(message.confidenceGrowthEpoch);
    }
    if (message.confidenceGrowthPerEpochBps !== BigInt(0)) {
      writer.uint32(312).uint64(message.confidenceGrowthPerEpochBps);
    }
    if (message.maxSurvivalConfidence !== BigInt(0)) {
      writer.uint32(320).uint64(message.maxSurvivalConfidence);
    }
    if (message.survivedChallengeConfidenceCap !== BigInt(0)) {
      writer.uint32(328).uint64(message.survivedChallengeConfidenceCap);
    }
    if (message.maxApprenticeValidators !== BigInt(0)) {
      writer.uint32(336).uint64(message.maxApprenticeValidators);
    }
    if (message.researchFundShareBps !== BigInt(0)) {
      writer.uint32(392).uint64(message.researchFundShareBps);
    }
    if (message.fitnessEpochBlocks !== BigInt(0)) {
      writer.uint32(408).uint64(message.fitnessEpochBlocks);
    }
    if (message.fitnessWeightQueryBps !== BigInt(0)) {
      writer.uint32(416).uint64(message.fitnessWeightQueryBps);
    }
    if (message.fitnessWeightCitationBps !== BigInt(0)) {
      writer.uint32(424).uint64(message.fitnessWeightCitationBps);
    }
    if (message.fitnessWeightBridgeBps !== BigInt(0)) {
      writer.uint32(432).uint64(message.fitnessWeightBridgeBps);
    }
    if (message.fitnessWeightDepthBps !== BigInt(0)) {
      writer.uint32(440).uint64(message.fitnessWeightDepthBps);
    }
    if (message.fitnessWeightPatronBps !== BigInt(0)) {
      writer.uint32(448).uint64(message.fitnessWeightPatronBps);
    }
    if (message.fitnessWeightUniqueBps !== BigInt(0)) {
      writer.uint32(456).uint64(message.fitnessWeightUniqueBps);
    }
    if (message.fitnessWeightAgeBps !== BigInt(0)) {
      writer.uint32(464).uint64(message.fitnessWeightAgeBps);
    }
    if (message.fitnessInitialScore !== BigInt(0)) {
      writer.uint32(472).uint64(message.fitnessInitialScore);
    }
    if (message.fitnessGraceEpochs !== BigInt(0)) {
      writer.uint32(480).uint64(message.fitnessGraceEpochs);
    }
    if (message.bootstrapFundEnabled === true) {
      writer.uint32(488).bool(message.bootstrapFundEnabled);
    }
    if (message.bootstrapFundMaxPerAddress !== "") {
      writer.uint32(498).string(message.bootstrapFundMaxPerAddress);
    }
    if (message.bootstrapFundMaxPerEpoch !== "") {
      writer.uint32(506).string(message.bootstrapFundMaxPerEpoch);
    }
    if (message.bootstrapFundEpochBlocks !== BigInt(0)) {
      writer.uint32(512).uint64(message.bootstrapFundEpochBlocks);
    }
    if (message.bootstrapFundFeeCap !== "") {
      writer.uint32(522).string(message.bootstrapFundFeeCap);
    }
    if (message.metabolismBaseCost !== BigInt(0)) {
      writer.uint32(528).uint64(message.metabolismBaseCost);
    }
    if (message.metabolismContentLengthBps !== BigInt(0)) {
      writer.uint32(536).uint64(message.metabolismContentLengthBps);
    }
    if (message.metabolismDomainCompetitionBps !== BigInt(0)) {
      writer.uint32(544).uint64(message.metabolismDomainCompetitionBps);
    }
    if (message.metabolismEnergyPerQuery !== BigInt(0)) {
      writer.uint32(552).uint64(message.metabolismEnergyPerQuery);
    }
    if (message.metabolismEnergyPerCitation !== BigInt(0)) {
      writer.uint32(560).uint64(message.metabolismEnergyPerCitation);
    }
    if (message.metabolismEnergyPerPatronage !== BigInt(0)) {
      writer.uint32(568).uint64(message.metabolismEnergyPerPatronage);
    }
    if (message.metabolismEnergyChallengeSurvival !== BigInt(0)) {
      writer.uint32(576).uint64(message.metabolismEnergyChallengeSurvival);
    }
    if (message.metabolismEnergyCap !== BigInt(0)) {
      writer.uint32(584).uint64(message.metabolismEnergyCap);
    }
    if (message.metabolismInitialEnergy !== BigInt(0)) {
      writer.uint32(592).uint64(message.metabolismInitialEnergy);
    }
    if (message.metabolismAtRiskEpochs !== BigInt(0)) {
      writer.uint32(600).uint64(message.metabolismAtRiskEpochs);
    }
    if (message.metabolismExpiredToPrunedEpochs !== BigInt(0)) {
      writer.uint32(608).uint64(message.metabolismExpiredToPrunedEpochs);
    }
    if (message.reproductionRoyaltyBps !== BigInt(0)) {
      writer.uint32(616).uint64(message.reproductionRoyaltyBps);
    }
    if (message.reproductionRoyaltyDecayBps !== BigInt(0)) {
      writer.uint32(624).uint64(message.reproductionRoyaltyDecayBps);
    }
    if (message.reproductionMaxRoyaltyDepth !== BigInt(0)) {
      writer.uint32(632).uint64(message.reproductionMaxRoyaltyDepth);
    }
    if (message.reproductionParentEnergyBonus !== BigInt(0)) {
      writer.uint32(640).uint64(message.reproductionParentEnergyBonus);
    }
    if (message.reproductionChildFitnessInheritanceBps !== BigInt(0)) {
      writer.uint32(648).uint64(message.reproductionChildFitnessInheritanceBps);
    }
    if (message.reproductionMaxChildren !== BigInt(0)) {
      writer.uint32(656).uint64(message.reproductionMaxChildren);
    }
    if (message.noveltyCommonKnowledgePenaltyBps !== BigInt(0)) {
      writer.uint32(664).uint64(message.noveltyCommonKnowledgePenaltyBps);
    }
    if (message.noveltySubjectOverlapPenaltyBps !== BigInt(0)) {
      writer.uint32(672).uint64(message.noveltySubjectOverlapPenaltyBps);
    }
    if (message.noveltyPrecisionBonusBps !== BigInt(0)) {
      writer.uint32(680).uint64(message.noveltyPrecisionBonusBps);
    }
    if (message.noveltyCrossDomainBonusBps !== BigInt(0)) {
      writer.uint32(688).uint64(message.noveltyCrossDomainBonusBps);
    }
    if (message.noveltyMaxOverlapFacts !== BigInt(0)) {
      writer.uint32(696).uint64(message.noveltyMaxOverlapFacts);
    }
    if (message.demandBountyThreshold !== BigInt(0)) {
      writer.uint32(704).uint64(message.demandBountyThreshold);
    }
    if (message.demandBountyBaseReward !== "") {
      writer.uint32(714).string(message.demandBountyBaseReward);
    }
    if (message.demandBountyPerQueryBonus !== "") {
      writer.uint32(722).string(message.demandBountyPerQueryBonus);
    }
    if (message.demandBountyExpiryEpochs !== BigInt(0)) {
      writer.uint32(728).uint64(message.demandBountyExpiryEpochs);
    }
    if (message.demandMultiplierCap !== BigInt(0)) {
      writer.uint32(736).uint64(message.demandMultiplierCap);
    }
    if (message.demandTrackingEnabled === true) {
      writer.uint32(744).bool(message.demandTrackingEnabled);
    }
    for (const v of message.authorizedDemandReporters) {
      writer.uint32(754).string(v!);
    }
    if (message.competitionNicheDominanceBonusBps !== BigInt(0)) {
      writer.uint32(760).uint64(message.competitionNicheDominanceBonusBps);
    }
    if (message.competitionRedundancyThresholdBps !== BigInt(0)) {
      writer.uint32(768).uint64(message.competitionRedundancyThresholdBps);
    }
    if (message.competitionMaxNicheSize !== BigInt(0)) {
      writer.uint32(776).uint64(message.competitionMaxNicheSize);
    }
    if (message.competitionSymbiosisBonusBps !== BigInt(0)) {
      writer.uint32(784).uint64(message.competitionSymbiosisBonusBps);
    }
    if (message.fitnessWeightSatisfactionBps !== BigInt(0)) {
      writer.uint32(792).uint64(message.fitnessWeightSatisfactionBps);
    }
    if (message.satisfactionMinRatings !== BigInt(0)) {
      writer.uint32(800).uint64(message.satisfactionMinRatings);
    }
    if (message.diversityConformityAlertThreshold !== BigInt(0)) {
      writer.uint32(808).uint64(message.diversityConformityAlertThreshold);
    }
    if (message.diversityConformityAlertEpochs !== BigInt(0)) {
      writer.uint32(816).uint64(message.diversityConformityAlertEpochs);
    }
    if (message.vindicationRefundEnabled === true) {
      writer.uint32(824).bool(message.vindicationRefundEnabled);
    }
    if (message.vindicationBonusBps !== BigInt(0)) {
      writer.uint32(832).uint64(message.vindicationBonusBps);
    }
    if (message.vindicationSlashBps !== BigInt(0)) {
      writer.uint32(840).uint64(message.vindicationSlashBps);
    }
    if (message.vindicationWindowBlocks !== BigInt(0)) {
      writer.uint32(848).uint64(message.vindicationWindowBlocks);
    }
    if (message.metabolismActiveThreshold !== BigInt(0)) {
      writer.uint32(856).uint64(message.metabolismActiveThreshold);
    }
    if (message.metabolismExtinctionThreshold !== BigInt(0)) {
      writer.uint32(864).uint64(message.metabolismExtinctionThreshold);
    }
    if (message.maxConfidence !== BigInt(0)) {
      writer.uint32(872).uint64(message.maxConfidence);
    }
    if (message.humanEmpiricalBonusBps !== BigInt(0)) {
      writer.uint32(880).uint64(message.humanEmpiricalBonusBps);
    }
    if (message.agentComputationalBonusBps !== BigInt(0)) {
      writer.uint32(888).uint64(message.agentComputationalBonusBps);
    }
    if (message.agentVerificationBonusBps !== BigInt(0)) {
      writer.uint32(896).uint64(message.agentVerificationBonusBps);
    }
    if (message.humanPatronageBonusBps !== BigInt(0)) {
      writer.uint32(904).uint64(message.humanPatronageBonusBps);
    }
    if (message.dualValidationBonusBps !== BigInt(0)) {
      writer.uint32(912).uint64(message.dualValidationBonusBps);
    }
    if (message.domainBaseCapacity !== BigInt(0)) {
      writer.uint32(920).uint64(message.domainBaseCapacity);
    }
    if (message.domainCapacityGrowthPerCitation !== BigInt(0)) {
      writer.uint32(928).uint64(message.domainCapacityGrowthPerCitation);
    }
    if (message.overcrowdingDecayMultiplierBps !== BigInt(0)) {
      writer.uint32(936).uint64(message.overcrowdingDecayMultiplierBps);
    }
    if (message.underpopulationBirthBonusBps !== BigInt(0)) {
      writer.uint32(944).uint64(message.underpopulationBirthBonusBps);
    }
    if (message.epistemicTemperatureDecayBps !== BigInt(0)) {
      writer.uint32(952).uint64(message.epistemicTemperatureDecayBps);
    }
    if (message.epistemicConformityCoolingBps !== BigInt(0)) {
      writer.uint32(960).uint64(message.epistemicConformityCoolingBps);
    }
    if (message.epistemicVindicationHeatingBps !== BigInt(0)) {
      writer.uint32(968).uint64(message.epistemicVindicationHeatingBps);
    }
    if (message.epistemicColdConfidenceCapBps !== BigInt(0)) {
      writer.uint32(976).uint64(message.epistemicColdConfidenceCapBps);
    }
    if (message.epistemicHotConfidenceGrowthBps !== BigInt(0)) {
      writer.uint32(984).uint64(message.epistemicHotConfidenceGrowthBps);
    }
    if (message.epistemicTemperatureWindowBlocks !== BigInt(0)) {
      writer.uint32(992).uint64(message.epistemicTemperatureWindowBlocks);
    }
    if (message.roleElasticityMinCalls !== BigInt(0)) {
      writer.uint32(1000).uint64(message.roleElasticityMinCalls);
    }
    if (message.roleElasticityMaxMultiplierBps !== BigInt(0)) {
      writer.uint32(1008).uint64(message.roleElasticityMaxMultiplierBps);
    }
    if (message.roleElasticityMinMultiplierBps !== BigInt(0)) {
      writer.uint32(1016).uint64(message.roleElasticityMinMultiplierBps);
    }
    if (message.roleElasticityDecayEpochs !== BigInt(0)) {
      writer.uint32(1024).uint64(message.roleElasticityDecayEpochs);
    }
    if (message.mentorshipDividendEnergy !== BigInt(0)) {
      writer.uint32(1032).uint64(message.mentorshipDividendEnergy);
    }
    if (message.mentorshipCapacityBonus !== BigInt(0)) {
      writer.uint32(1040).uint64(message.mentorshipCapacityBonus);
    }
    if (message.socialSaturationThreshold !== BigInt(0)) {
      writer.uint32(1048).uint64(message.socialSaturationThreshold);
    }
    if (message.observationWindowBlocks !== BigInt(0)) {
      writer.uint32(1056).uint64(message.observationWindowBlocks);
    }
    if (message.minHeadcountAgreement !== BigInt(0)) {
      writer.uint32(1064).uint64(message.minHeadcountAgreement);
    }
    if (message.challengeConfidenceScalingBps !== BigInt(0)) {
      writer.uint32(1072).uint64(message.challengeConfidenceScalingBps);
    }
    if (message.independenceRewardStrengthBps !== BigInt(0)) {
      writer.uint32(1080).uint64(message.independenceRewardStrengthBps);
    }
    if (message.reformulationMinPanelVotes !== BigInt(0)) {
      writer.uint32(1088).uint64(message.reformulationMinPanelVotes);
    }
    if (message.reformulationConsensusBps !== BigInt(0)) {
      writer.uint32(1096).uint64(message.reformulationConsensusBps);
    }
    if (message.reformulationSuperiorBonusBps !== BigInt(0)) {
      writer.uint32(1104).uint64(message.reformulationSuperiorBonusBps);
    }
    if (message.augmentationExpiryFeeBps !== BigInt(0)) {
      writer.uint32(1112).uint64(message.augmentationExpiryFeeBps);
    }
    Object.entries(message.methodologyNormalizationBps).forEach(([key, value]) => {
      Params_MethodologyNormalizationBpsEntry.encode({
        key: key as any,
        value
      }, writer.uint32(1120).fork()).ldelim();
    });
    if (message.vindicationTvwMultiplierBps !== BigInt(0)) {
      writer.uint32(1128).uint64(message.vindicationTvwMultiplierBps);
    }
    if (message.disprovalClawbackBps !== BigInt(0)) {
      writer.uint32(1136).uint64(message.disprovalClawbackBps);
    }
    if (message.disprovalClawbackWindowEpochs !== BigInt(0)) {
      writer.uint32(1144).uint64(message.disprovalClawbackWindowEpochs);
    }
    if (message.trainingFundCalibrationFloorBps !== BigInt(0)) {
      writer.uint32(1152).uint64(message.trainingFundCalibrationFloorBps);
    }
    if (message.trainingFundVestingEpochs !== BigInt(0)) {
      writer.uint32(1160).uint64(message.trainingFundVestingEpochs);
    }
    if (message.trainingFundMethodologyDiversityBonusBps !== BigInt(0)) {
      writer.uint32(1168).uint64(message.trainingFundMethodologyDiversityBonusBps);
    }
    if (message.trainingFundBaseReward !== "") {
      writer.uint32(1178).string(message.trainingFundBaseReward);
    }
    if (message.contributionChallengeBond !== "") {
      writer.uint32(1186).string(message.contributionChallengeBond);
    }
    if (message.contributionChallengeRewardMultiplierBps !== BigInt(0)) {
      writer.uint32(1192).uint64(message.contributionChallengeRewardMultiplierBps);
    }
    if (message.sponsorVetoForfeitBps !== BigInt(0)) {
      writer.uint32(1200).uint64(message.sponsorVetoForfeitBps);
    }
    if (message.maxPauseDurationBlocks !== BigInt(0)) {
      writer.uint32(1208).uint64(message.maxPauseDurationBlocks);
    }
    if (message.probeInvitationIdleThresholdBlocks !== BigInt(0)) {
      writer.uint32(1216).uint64(message.probeInvitationIdleThresholdBlocks);
    }
    if (message.probeInvitationMinConfidenceBps !== BigInt(0)) {
      writer.uint32(1224).uint64(message.probeInvitationMinConfidenceBps);
    }
    if (message.probeInvitationBatchSize !== 0) {
      writer.uint32(1232).uint32(message.probeInvitationBatchSize);
    }
    if (message.probeInvitationReinviteCooldown !== BigInt(0)) {
      writer.uint32(1240).uint64(message.probeInvitationReinviteCooldown);
    }
    if (message.probeBountyMintPerBlock !== "") {
      writer.uint32(1250).string(message.probeBountyMintPerBlock);
    }
    if (message.probeBountyMaxPoolSize !== "") {
      writer.uint32(1258).string(message.probeBountyMaxPoolSize);
    }
    if (message.invitationBonusAmount !== "") {
      writer.uint32(1266).string(message.invitationBonusAmount);
    }
    for (const v of message.guardianAddresses) {
      writer.uint32(1274).string(v!);
    }
    if (message.addFactVetoWindowBlocks !== BigInt(0)) {
      writer.uint32(1280).uint64(message.addFactVetoWindowBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Params {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minVerifiers = reader.uint64();
          break;
        case 2:
          message.maxVerifiers = reader.uint64();
          break;
        case 3:
          message.commitPhaseBlocks = reader.uint64();
          break;
        case 4:
          message.revealPhaseBlocks = reader.uint64();
          break;
        case 5:
          message.aggregationPhaseBlocks = reader.uint64();
          break;
        case 6:
          message.claimCooldownBlocks = reader.uint64();
          break;
        case 7:
          message.initialConfidence = reader.uint64();
          break;
        case 8:
          message.confidenceBoostPerVerification = reader.uint64();
          break;
        case 9:
          message.confidenceThreshold = reader.uint64();
          break;
        case 10:
          message.quorumThreshold = reader.uint64();
          break;
        case 11:
          message.wrongVerificationSlashBps = reader.uint64();
          break;
        case 12:
          message.missedRevealSlashBps = reader.uint64();
          break;
        case 13:
          message.equivocationSlashBps = reader.uint64();
          break;
        case 14:
          message.invalidClaimSlashBps = reader.uint64();
          break;
        case 15:
          message.verificationReward = reader.string();
          break;
        case 16:
          message.verificationRewardDecayBps = reader.uint64();
          break;
        case 17:
          message.minClaimTextLength = reader.uint64();
          break;
        case 18:
          message.maxClaimTextLength = reader.uint64();
          break;
        case 19:
          message.minReviewFee = reader.string();
          break;
        case 20:
          message.adversarialVerificationEnabled = reader.bool();
          break;
        case 21:
          message.provisionalThreshold = reader.uint64();
          break;
        case 22:
          message.rejectThreshold = reader.uint64();
          break;
        case 23:
          message.challengeDurationBlocks = reader.uint64();
          break;
        case 24:
          message.minChallengeStake = reader.string();
          break;
        case 25:
          message.failedChallengeSlashBps = reader.uint64();
          break;
        case 26:
          message.successfulChallengeRewardBps = reader.uint64();
          break;
        case 27:
          message.maxConcurrentChallenges = reader.uint64();
          break;
        case 28:
          message.citationShareBps = reader.uint64();
          break;
        case 29:
          message.crossDomainBonusBps = reader.uint64();
          break;
        case 30:
          message.maxFactsPerDomain = reader.uint64();
          break;
        case 31:
          message.factExpiryBlocks = reader.uint64();
          break;
        case 32:
          message.crossStratumDiscountBps = reader.uint64();
          break;
        case 34:
          message.maxValidatorsPerRound = reader.uint64();
          break;
        case 38:
          message.confidenceGrowthEpoch = reader.uint64();
          break;
        case 39:
          message.confidenceGrowthPerEpochBps = reader.uint64();
          break;
        case 40:
          message.maxSurvivalConfidence = reader.uint64();
          break;
        case 41:
          message.survivedChallengeConfidenceCap = reader.uint64();
          break;
        case 42:
          message.maxApprenticeValidators = reader.uint64();
          break;
        case 49:
          message.researchFundShareBps = reader.uint64();
          break;
        case 51:
          message.fitnessEpochBlocks = reader.uint64();
          break;
        case 52:
          message.fitnessWeightQueryBps = reader.uint64();
          break;
        case 53:
          message.fitnessWeightCitationBps = reader.uint64();
          break;
        case 54:
          message.fitnessWeightBridgeBps = reader.uint64();
          break;
        case 55:
          message.fitnessWeightDepthBps = reader.uint64();
          break;
        case 56:
          message.fitnessWeightPatronBps = reader.uint64();
          break;
        case 57:
          message.fitnessWeightUniqueBps = reader.uint64();
          break;
        case 58:
          message.fitnessWeightAgeBps = reader.uint64();
          break;
        case 59:
          message.fitnessInitialScore = reader.uint64();
          break;
        case 60:
          message.fitnessGraceEpochs = reader.uint64();
          break;
        case 61:
          message.bootstrapFundEnabled = reader.bool();
          break;
        case 62:
          message.bootstrapFundMaxPerAddress = reader.string();
          break;
        case 63:
          message.bootstrapFundMaxPerEpoch = reader.string();
          break;
        case 64:
          message.bootstrapFundEpochBlocks = reader.uint64();
          break;
        case 65:
          message.bootstrapFundFeeCap = reader.string();
          break;
        case 66:
          message.metabolismBaseCost = reader.uint64();
          break;
        case 67:
          message.metabolismContentLengthBps = reader.uint64();
          break;
        case 68:
          message.metabolismDomainCompetitionBps = reader.uint64();
          break;
        case 69:
          message.metabolismEnergyPerQuery = reader.uint64();
          break;
        case 70:
          message.metabolismEnergyPerCitation = reader.uint64();
          break;
        case 71:
          message.metabolismEnergyPerPatronage = reader.uint64();
          break;
        case 72:
          message.metabolismEnergyChallengeSurvival = reader.uint64();
          break;
        case 73:
          message.metabolismEnergyCap = reader.uint64();
          break;
        case 74:
          message.metabolismInitialEnergy = reader.uint64();
          break;
        case 75:
          message.metabolismAtRiskEpochs = reader.uint64();
          break;
        case 76:
          message.metabolismExpiredToPrunedEpochs = reader.uint64();
          break;
        case 77:
          message.reproductionRoyaltyBps = reader.uint64();
          break;
        case 78:
          message.reproductionRoyaltyDecayBps = reader.uint64();
          break;
        case 79:
          message.reproductionMaxRoyaltyDepth = reader.uint64();
          break;
        case 80:
          message.reproductionParentEnergyBonus = reader.uint64();
          break;
        case 81:
          message.reproductionChildFitnessInheritanceBps = reader.uint64();
          break;
        case 82:
          message.reproductionMaxChildren = reader.uint64();
          break;
        case 83:
          message.noveltyCommonKnowledgePenaltyBps = reader.uint64();
          break;
        case 84:
          message.noveltySubjectOverlapPenaltyBps = reader.uint64();
          break;
        case 85:
          message.noveltyPrecisionBonusBps = reader.uint64();
          break;
        case 86:
          message.noveltyCrossDomainBonusBps = reader.uint64();
          break;
        case 87:
          message.noveltyMaxOverlapFacts = reader.uint64();
          break;
        case 88:
          message.demandBountyThreshold = reader.uint64();
          break;
        case 89:
          message.demandBountyBaseReward = reader.string();
          break;
        case 90:
          message.demandBountyPerQueryBonus = reader.string();
          break;
        case 91:
          message.demandBountyExpiryEpochs = reader.uint64();
          break;
        case 92:
          message.demandMultiplierCap = reader.uint64();
          break;
        case 93:
          message.demandTrackingEnabled = reader.bool();
          break;
        case 94:
          message.authorizedDemandReporters.push(reader.string());
          break;
        case 95:
          message.competitionNicheDominanceBonusBps = reader.uint64();
          break;
        case 96:
          message.competitionRedundancyThresholdBps = reader.uint64();
          break;
        case 97:
          message.competitionMaxNicheSize = reader.uint64();
          break;
        case 98:
          message.competitionSymbiosisBonusBps = reader.uint64();
          break;
        case 99:
          message.fitnessWeightSatisfactionBps = reader.uint64();
          break;
        case 100:
          message.satisfactionMinRatings = reader.uint64();
          break;
        case 101:
          message.diversityConformityAlertThreshold = reader.uint64();
          break;
        case 102:
          message.diversityConformityAlertEpochs = reader.uint64();
          break;
        case 103:
          message.vindicationRefundEnabled = reader.bool();
          break;
        case 104:
          message.vindicationBonusBps = reader.uint64();
          break;
        case 105:
          message.vindicationSlashBps = reader.uint64();
          break;
        case 106:
          message.vindicationWindowBlocks = reader.uint64();
          break;
        case 107:
          message.metabolismActiveThreshold = reader.uint64();
          break;
        case 108:
          message.metabolismExtinctionThreshold = reader.uint64();
          break;
        case 109:
          message.maxConfidence = reader.uint64();
          break;
        case 110:
          message.humanEmpiricalBonusBps = reader.uint64();
          break;
        case 111:
          message.agentComputationalBonusBps = reader.uint64();
          break;
        case 112:
          message.agentVerificationBonusBps = reader.uint64();
          break;
        case 113:
          message.humanPatronageBonusBps = reader.uint64();
          break;
        case 114:
          message.dualValidationBonusBps = reader.uint64();
          break;
        case 115:
          message.domainBaseCapacity = reader.uint64();
          break;
        case 116:
          message.domainCapacityGrowthPerCitation = reader.uint64();
          break;
        case 117:
          message.overcrowdingDecayMultiplierBps = reader.uint64();
          break;
        case 118:
          message.underpopulationBirthBonusBps = reader.uint64();
          break;
        case 119:
          message.epistemicTemperatureDecayBps = reader.uint64();
          break;
        case 120:
          message.epistemicConformityCoolingBps = reader.uint64();
          break;
        case 121:
          message.epistemicVindicationHeatingBps = reader.uint64();
          break;
        case 122:
          message.epistemicColdConfidenceCapBps = reader.uint64();
          break;
        case 123:
          message.epistemicHotConfidenceGrowthBps = reader.uint64();
          break;
        case 124:
          message.epistemicTemperatureWindowBlocks = reader.uint64();
          break;
        case 125:
          message.roleElasticityMinCalls = reader.uint64();
          break;
        case 126:
          message.roleElasticityMaxMultiplierBps = reader.uint64();
          break;
        case 127:
          message.roleElasticityMinMultiplierBps = reader.uint64();
          break;
        case 128:
          message.roleElasticityDecayEpochs = reader.uint64();
          break;
        case 129:
          message.mentorshipDividendEnergy = reader.uint64();
          break;
        case 130:
          message.mentorshipCapacityBonus = reader.uint64();
          break;
        case 131:
          message.socialSaturationThreshold = reader.uint64();
          break;
        case 132:
          message.observationWindowBlocks = reader.uint64();
          break;
        case 133:
          message.minHeadcountAgreement = reader.uint64();
          break;
        case 134:
          message.challengeConfidenceScalingBps = reader.uint64();
          break;
        case 135:
          message.independenceRewardStrengthBps = reader.uint64();
          break;
        case 136:
          message.reformulationMinPanelVotes = reader.uint64();
          break;
        case 137:
          message.reformulationConsensusBps = reader.uint64();
          break;
        case 138:
          message.reformulationSuperiorBonusBps = reader.uint64();
          break;
        case 139:
          message.augmentationExpiryFeeBps = reader.uint64();
          break;
        case 140:
          const entry140 = Params_MethodologyNormalizationBpsEntry.decode(reader, reader.uint32());
          if (entry140.value !== undefined) {
            message.methodologyNormalizationBps[entry140.key] = entry140.value;
          }
          break;
        case 141:
          message.vindicationTvwMultiplierBps = reader.uint64();
          break;
        case 142:
          message.disprovalClawbackBps = reader.uint64();
          break;
        case 143:
          message.disprovalClawbackWindowEpochs = reader.uint64();
          break;
        case 144:
          message.trainingFundCalibrationFloorBps = reader.uint64();
          break;
        case 145:
          message.trainingFundVestingEpochs = reader.uint64();
          break;
        case 146:
          message.trainingFundMethodologyDiversityBonusBps = reader.uint64();
          break;
        case 147:
          message.trainingFundBaseReward = reader.string();
          break;
        case 148:
          message.contributionChallengeBond = reader.string();
          break;
        case 149:
          message.contributionChallengeRewardMultiplierBps = reader.uint64();
          break;
        case 150:
          message.sponsorVetoForfeitBps = reader.uint64();
          break;
        case 151:
          message.maxPauseDurationBlocks = reader.uint64();
          break;
        case 152:
          message.probeInvitationIdleThresholdBlocks = reader.uint64();
          break;
        case 153:
          message.probeInvitationMinConfidenceBps = reader.uint64();
          break;
        case 154:
          message.probeInvitationBatchSize = reader.uint32();
          break;
        case 155:
          message.probeInvitationReinviteCooldown = reader.uint64();
          break;
        case 156:
          message.probeBountyMintPerBlock = reader.string();
          break;
        case 157:
          message.probeBountyMaxPoolSize = reader.string();
          break;
        case 158:
          message.invitationBonusAmount = reader.string();
          break;
        case 159:
          message.guardianAddresses.push(reader.string());
          break;
        case 160:
          message.addFactVetoWindowBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Params>): Params {
    const message = createBaseParams();
    message.minVerifiers = object.minVerifiers !== undefined && object.minVerifiers !== null ? BigInt(object.minVerifiers.toString()) : BigInt(0);
    message.maxVerifiers = object.maxVerifiers !== undefined && object.maxVerifiers !== null ? BigInt(object.maxVerifiers.toString()) : BigInt(0);
    message.commitPhaseBlocks = object.commitPhaseBlocks !== undefined && object.commitPhaseBlocks !== null ? BigInt(object.commitPhaseBlocks.toString()) : BigInt(0);
    message.revealPhaseBlocks = object.revealPhaseBlocks !== undefined && object.revealPhaseBlocks !== null ? BigInt(object.revealPhaseBlocks.toString()) : BigInt(0);
    message.aggregationPhaseBlocks = object.aggregationPhaseBlocks !== undefined && object.aggregationPhaseBlocks !== null ? BigInt(object.aggregationPhaseBlocks.toString()) : BigInt(0);
    message.claimCooldownBlocks = object.claimCooldownBlocks !== undefined && object.claimCooldownBlocks !== null ? BigInt(object.claimCooldownBlocks.toString()) : BigInt(0);
    message.initialConfidence = object.initialConfidence !== undefined && object.initialConfidence !== null ? BigInt(object.initialConfidence.toString()) : BigInt(0);
    message.confidenceBoostPerVerification = object.confidenceBoostPerVerification !== undefined && object.confidenceBoostPerVerification !== null ? BigInt(object.confidenceBoostPerVerification.toString()) : BigInt(0);
    message.confidenceThreshold = object.confidenceThreshold !== undefined && object.confidenceThreshold !== null ? BigInt(object.confidenceThreshold.toString()) : BigInt(0);
    message.quorumThreshold = object.quorumThreshold !== undefined && object.quorumThreshold !== null ? BigInt(object.quorumThreshold.toString()) : BigInt(0);
    message.wrongVerificationSlashBps = object.wrongVerificationSlashBps !== undefined && object.wrongVerificationSlashBps !== null ? BigInt(object.wrongVerificationSlashBps.toString()) : BigInt(0);
    message.missedRevealSlashBps = object.missedRevealSlashBps !== undefined && object.missedRevealSlashBps !== null ? BigInt(object.missedRevealSlashBps.toString()) : BigInt(0);
    message.equivocationSlashBps = object.equivocationSlashBps !== undefined && object.equivocationSlashBps !== null ? BigInt(object.equivocationSlashBps.toString()) : BigInt(0);
    message.invalidClaimSlashBps = object.invalidClaimSlashBps !== undefined && object.invalidClaimSlashBps !== null ? BigInt(object.invalidClaimSlashBps.toString()) : BigInt(0);
    message.verificationReward = object.verificationReward ?? "";
    message.verificationRewardDecayBps = object.verificationRewardDecayBps !== undefined && object.verificationRewardDecayBps !== null ? BigInt(object.verificationRewardDecayBps.toString()) : BigInt(0);
    message.minClaimTextLength = object.minClaimTextLength !== undefined && object.minClaimTextLength !== null ? BigInt(object.minClaimTextLength.toString()) : BigInt(0);
    message.maxClaimTextLength = object.maxClaimTextLength !== undefined && object.maxClaimTextLength !== null ? BigInt(object.maxClaimTextLength.toString()) : BigInt(0);
    message.minReviewFee = object.minReviewFee ?? "";
    message.adversarialVerificationEnabled = object.adversarialVerificationEnabled ?? false;
    message.provisionalThreshold = object.provisionalThreshold !== undefined && object.provisionalThreshold !== null ? BigInt(object.provisionalThreshold.toString()) : BigInt(0);
    message.rejectThreshold = object.rejectThreshold !== undefined && object.rejectThreshold !== null ? BigInt(object.rejectThreshold.toString()) : BigInt(0);
    message.challengeDurationBlocks = object.challengeDurationBlocks !== undefined && object.challengeDurationBlocks !== null ? BigInt(object.challengeDurationBlocks.toString()) : BigInt(0);
    message.minChallengeStake = object.minChallengeStake ?? "";
    message.failedChallengeSlashBps = object.failedChallengeSlashBps !== undefined && object.failedChallengeSlashBps !== null ? BigInt(object.failedChallengeSlashBps.toString()) : BigInt(0);
    message.successfulChallengeRewardBps = object.successfulChallengeRewardBps !== undefined && object.successfulChallengeRewardBps !== null ? BigInt(object.successfulChallengeRewardBps.toString()) : BigInt(0);
    message.maxConcurrentChallenges = object.maxConcurrentChallenges !== undefined && object.maxConcurrentChallenges !== null ? BigInt(object.maxConcurrentChallenges.toString()) : BigInt(0);
    message.citationShareBps = object.citationShareBps !== undefined && object.citationShareBps !== null ? BigInt(object.citationShareBps.toString()) : BigInt(0);
    message.crossDomainBonusBps = object.crossDomainBonusBps !== undefined && object.crossDomainBonusBps !== null ? BigInt(object.crossDomainBonusBps.toString()) : BigInt(0);
    message.maxFactsPerDomain = object.maxFactsPerDomain !== undefined && object.maxFactsPerDomain !== null ? BigInt(object.maxFactsPerDomain.toString()) : BigInt(0);
    message.factExpiryBlocks = object.factExpiryBlocks !== undefined && object.factExpiryBlocks !== null ? BigInt(object.factExpiryBlocks.toString()) : BigInt(0);
    message.crossStratumDiscountBps = object.crossStratumDiscountBps !== undefined && object.crossStratumDiscountBps !== null ? BigInt(object.crossStratumDiscountBps.toString()) : BigInt(0);
    message.maxValidatorsPerRound = object.maxValidatorsPerRound !== undefined && object.maxValidatorsPerRound !== null ? BigInt(object.maxValidatorsPerRound.toString()) : BigInt(0);
    message.confidenceGrowthEpoch = object.confidenceGrowthEpoch !== undefined && object.confidenceGrowthEpoch !== null ? BigInt(object.confidenceGrowthEpoch.toString()) : BigInt(0);
    message.confidenceGrowthPerEpochBps = object.confidenceGrowthPerEpochBps !== undefined && object.confidenceGrowthPerEpochBps !== null ? BigInt(object.confidenceGrowthPerEpochBps.toString()) : BigInt(0);
    message.maxSurvivalConfidence = object.maxSurvivalConfidence !== undefined && object.maxSurvivalConfidence !== null ? BigInt(object.maxSurvivalConfidence.toString()) : BigInt(0);
    message.survivedChallengeConfidenceCap = object.survivedChallengeConfidenceCap !== undefined && object.survivedChallengeConfidenceCap !== null ? BigInt(object.survivedChallengeConfidenceCap.toString()) : BigInt(0);
    message.maxApprenticeValidators = object.maxApprenticeValidators !== undefined && object.maxApprenticeValidators !== null ? BigInt(object.maxApprenticeValidators.toString()) : BigInt(0);
    message.researchFundShareBps = object.researchFundShareBps !== undefined && object.researchFundShareBps !== null ? BigInt(object.researchFundShareBps.toString()) : BigInt(0);
    message.fitnessEpochBlocks = object.fitnessEpochBlocks !== undefined && object.fitnessEpochBlocks !== null ? BigInt(object.fitnessEpochBlocks.toString()) : BigInt(0);
    message.fitnessWeightQueryBps = object.fitnessWeightQueryBps !== undefined && object.fitnessWeightQueryBps !== null ? BigInt(object.fitnessWeightQueryBps.toString()) : BigInt(0);
    message.fitnessWeightCitationBps = object.fitnessWeightCitationBps !== undefined && object.fitnessWeightCitationBps !== null ? BigInt(object.fitnessWeightCitationBps.toString()) : BigInt(0);
    message.fitnessWeightBridgeBps = object.fitnessWeightBridgeBps !== undefined && object.fitnessWeightBridgeBps !== null ? BigInt(object.fitnessWeightBridgeBps.toString()) : BigInt(0);
    message.fitnessWeightDepthBps = object.fitnessWeightDepthBps !== undefined && object.fitnessWeightDepthBps !== null ? BigInt(object.fitnessWeightDepthBps.toString()) : BigInt(0);
    message.fitnessWeightPatronBps = object.fitnessWeightPatronBps !== undefined && object.fitnessWeightPatronBps !== null ? BigInt(object.fitnessWeightPatronBps.toString()) : BigInt(0);
    message.fitnessWeightUniqueBps = object.fitnessWeightUniqueBps !== undefined && object.fitnessWeightUniqueBps !== null ? BigInt(object.fitnessWeightUniqueBps.toString()) : BigInt(0);
    message.fitnessWeightAgeBps = object.fitnessWeightAgeBps !== undefined && object.fitnessWeightAgeBps !== null ? BigInt(object.fitnessWeightAgeBps.toString()) : BigInt(0);
    message.fitnessInitialScore = object.fitnessInitialScore !== undefined && object.fitnessInitialScore !== null ? BigInt(object.fitnessInitialScore.toString()) : BigInt(0);
    message.fitnessGraceEpochs = object.fitnessGraceEpochs !== undefined && object.fitnessGraceEpochs !== null ? BigInt(object.fitnessGraceEpochs.toString()) : BigInt(0);
    message.bootstrapFundEnabled = object.bootstrapFundEnabled ?? false;
    message.bootstrapFundMaxPerAddress = object.bootstrapFundMaxPerAddress ?? "";
    message.bootstrapFundMaxPerEpoch = object.bootstrapFundMaxPerEpoch ?? "";
    message.bootstrapFundEpochBlocks = object.bootstrapFundEpochBlocks !== undefined && object.bootstrapFundEpochBlocks !== null ? BigInt(object.bootstrapFundEpochBlocks.toString()) : BigInt(0);
    message.bootstrapFundFeeCap = object.bootstrapFundFeeCap ?? "";
    message.metabolismBaseCost = object.metabolismBaseCost !== undefined && object.metabolismBaseCost !== null ? BigInt(object.metabolismBaseCost.toString()) : BigInt(0);
    message.metabolismContentLengthBps = object.metabolismContentLengthBps !== undefined && object.metabolismContentLengthBps !== null ? BigInt(object.metabolismContentLengthBps.toString()) : BigInt(0);
    message.metabolismDomainCompetitionBps = object.metabolismDomainCompetitionBps !== undefined && object.metabolismDomainCompetitionBps !== null ? BigInt(object.metabolismDomainCompetitionBps.toString()) : BigInt(0);
    message.metabolismEnergyPerQuery = object.metabolismEnergyPerQuery !== undefined && object.metabolismEnergyPerQuery !== null ? BigInt(object.metabolismEnergyPerQuery.toString()) : BigInt(0);
    message.metabolismEnergyPerCitation = object.metabolismEnergyPerCitation !== undefined && object.metabolismEnergyPerCitation !== null ? BigInt(object.metabolismEnergyPerCitation.toString()) : BigInt(0);
    message.metabolismEnergyPerPatronage = object.metabolismEnergyPerPatronage !== undefined && object.metabolismEnergyPerPatronage !== null ? BigInt(object.metabolismEnergyPerPatronage.toString()) : BigInt(0);
    message.metabolismEnergyChallengeSurvival = object.metabolismEnergyChallengeSurvival !== undefined && object.metabolismEnergyChallengeSurvival !== null ? BigInt(object.metabolismEnergyChallengeSurvival.toString()) : BigInt(0);
    message.metabolismEnergyCap = object.metabolismEnergyCap !== undefined && object.metabolismEnergyCap !== null ? BigInt(object.metabolismEnergyCap.toString()) : BigInt(0);
    message.metabolismInitialEnergy = object.metabolismInitialEnergy !== undefined && object.metabolismInitialEnergy !== null ? BigInt(object.metabolismInitialEnergy.toString()) : BigInt(0);
    message.metabolismAtRiskEpochs = object.metabolismAtRiskEpochs !== undefined && object.metabolismAtRiskEpochs !== null ? BigInt(object.metabolismAtRiskEpochs.toString()) : BigInt(0);
    message.metabolismExpiredToPrunedEpochs = object.metabolismExpiredToPrunedEpochs !== undefined && object.metabolismExpiredToPrunedEpochs !== null ? BigInt(object.metabolismExpiredToPrunedEpochs.toString()) : BigInt(0);
    message.reproductionRoyaltyBps = object.reproductionRoyaltyBps !== undefined && object.reproductionRoyaltyBps !== null ? BigInt(object.reproductionRoyaltyBps.toString()) : BigInt(0);
    message.reproductionRoyaltyDecayBps = object.reproductionRoyaltyDecayBps !== undefined && object.reproductionRoyaltyDecayBps !== null ? BigInt(object.reproductionRoyaltyDecayBps.toString()) : BigInt(0);
    message.reproductionMaxRoyaltyDepth = object.reproductionMaxRoyaltyDepth !== undefined && object.reproductionMaxRoyaltyDepth !== null ? BigInt(object.reproductionMaxRoyaltyDepth.toString()) : BigInt(0);
    message.reproductionParentEnergyBonus = object.reproductionParentEnergyBonus !== undefined && object.reproductionParentEnergyBonus !== null ? BigInt(object.reproductionParentEnergyBonus.toString()) : BigInt(0);
    message.reproductionChildFitnessInheritanceBps = object.reproductionChildFitnessInheritanceBps !== undefined && object.reproductionChildFitnessInheritanceBps !== null ? BigInt(object.reproductionChildFitnessInheritanceBps.toString()) : BigInt(0);
    message.reproductionMaxChildren = object.reproductionMaxChildren !== undefined && object.reproductionMaxChildren !== null ? BigInt(object.reproductionMaxChildren.toString()) : BigInt(0);
    message.noveltyCommonKnowledgePenaltyBps = object.noveltyCommonKnowledgePenaltyBps !== undefined && object.noveltyCommonKnowledgePenaltyBps !== null ? BigInt(object.noveltyCommonKnowledgePenaltyBps.toString()) : BigInt(0);
    message.noveltySubjectOverlapPenaltyBps = object.noveltySubjectOverlapPenaltyBps !== undefined && object.noveltySubjectOverlapPenaltyBps !== null ? BigInt(object.noveltySubjectOverlapPenaltyBps.toString()) : BigInt(0);
    message.noveltyPrecisionBonusBps = object.noveltyPrecisionBonusBps !== undefined && object.noveltyPrecisionBonusBps !== null ? BigInt(object.noveltyPrecisionBonusBps.toString()) : BigInt(0);
    message.noveltyCrossDomainBonusBps = object.noveltyCrossDomainBonusBps !== undefined && object.noveltyCrossDomainBonusBps !== null ? BigInt(object.noveltyCrossDomainBonusBps.toString()) : BigInt(0);
    message.noveltyMaxOverlapFacts = object.noveltyMaxOverlapFacts !== undefined && object.noveltyMaxOverlapFacts !== null ? BigInt(object.noveltyMaxOverlapFacts.toString()) : BigInt(0);
    message.demandBountyThreshold = object.demandBountyThreshold !== undefined && object.demandBountyThreshold !== null ? BigInt(object.demandBountyThreshold.toString()) : BigInt(0);
    message.demandBountyBaseReward = object.demandBountyBaseReward ?? "";
    message.demandBountyPerQueryBonus = object.demandBountyPerQueryBonus ?? "";
    message.demandBountyExpiryEpochs = object.demandBountyExpiryEpochs !== undefined && object.demandBountyExpiryEpochs !== null ? BigInt(object.demandBountyExpiryEpochs.toString()) : BigInt(0);
    message.demandMultiplierCap = object.demandMultiplierCap !== undefined && object.demandMultiplierCap !== null ? BigInt(object.demandMultiplierCap.toString()) : BigInt(0);
    message.demandTrackingEnabled = object.demandTrackingEnabled ?? false;
    message.authorizedDemandReporters = object.authorizedDemandReporters?.map(e => e) || [];
    message.competitionNicheDominanceBonusBps = object.competitionNicheDominanceBonusBps !== undefined && object.competitionNicheDominanceBonusBps !== null ? BigInt(object.competitionNicheDominanceBonusBps.toString()) : BigInt(0);
    message.competitionRedundancyThresholdBps = object.competitionRedundancyThresholdBps !== undefined && object.competitionRedundancyThresholdBps !== null ? BigInt(object.competitionRedundancyThresholdBps.toString()) : BigInt(0);
    message.competitionMaxNicheSize = object.competitionMaxNicheSize !== undefined && object.competitionMaxNicheSize !== null ? BigInt(object.competitionMaxNicheSize.toString()) : BigInt(0);
    message.competitionSymbiosisBonusBps = object.competitionSymbiosisBonusBps !== undefined && object.competitionSymbiosisBonusBps !== null ? BigInt(object.competitionSymbiosisBonusBps.toString()) : BigInt(0);
    message.fitnessWeightSatisfactionBps = object.fitnessWeightSatisfactionBps !== undefined && object.fitnessWeightSatisfactionBps !== null ? BigInt(object.fitnessWeightSatisfactionBps.toString()) : BigInt(0);
    message.satisfactionMinRatings = object.satisfactionMinRatings !== undefined && object.satisfactionMinRatings !== null ? BigInt(object.satisfactionMinRatings.toString()) : BigInt(0);
    message.diversityConformityAlertThreshold = object.diversityConformityAlertThreshold !== undefined && object.diversityConformityAlertThreshold !== null ? BigInt(object.diversityConformityAlertThreshold.toString()) : BigInt(0);
    message.diversityConformityAlertEpochs = object.diversityConformityAlertEpochs !== undefined && object.diversityConformityAlertEpochs !== null ? BigInt(object.diversityConformityAlertEpochs.toString()) : BigInt(0);
    message.vindicationRefundEnabled = object.vindicationRefundEnabled ?? false;
    message.vindicationBonusBps = object.vindicationBonusBps !== undefined && object.vindicationBonusBps !== null ? BigInt(object.vindicationBonusBps.toString()) : BigInt(0);
    message.vindicationSlashBps = object.vindicationSlashBps !== undefined && object.vindicationSlashBps !== null ? BigInt(object.vindicationSlashBps.toString()) : BigInt(0);
    message.vindicationWindowBlocks = object.vindicationWindowBlocks !== undefined && object.vindicationWindowBlocks !== null ? BigInt(object.vindicationWindowBlocks.toString()) : BigInt(0);
    message.metabolismActiveThreshold = object.metabolismActiveThreshold !== undefined && object.metabolismActiveThreshold !== null ? BigInt(object.metabolismActiveThreshold.toString()) : BigInt(0);
    message.metabolismExtinctionThreshold = object.metabolismExtinctionThreshold !== undefined && object.metabolismExtinctionThreshold !== null ? BigInt(object.metabolismExtinctionThreshold.toString()) : BigInt(0);
    message.maxConfidence = object.maxConfidence !== undefined && object.maxConfidence !== null ? BigInt(object.maxConfidence.toString()) : BigInt(0);
    message.humanEmpiricalBonusBps = object.humanEmpiricalBonusBps !== undefined && object.humanEmpiricalBonusBps !== null ? BigInt(object.humanEmpiricalBonusBps.toString()) : BigInt(0);
    message.agentComputationalBonusBps = object.agentComputationalBonusBps !== undefined && object.agentComputationalBonusBps !== null ? BigInt(object.agentComputationalBonusBps.toString()) : BigInt(0);
    message.agentVerificationBonusBps = object.agentVerificationBonusBps !== undefined && object.agentVerificationBonusBps !== null ? BigInt(object.agentVerificationBonusBps.toString()) : BigInt(0);
    message.humanPatronageBonusBps = object.humanPatronageBonusBps !== undefined && object.humanPatronageBonusBps !== null ? BigInt(object.humanPatronageBonusBps.toString()) : BigInt(0);
    message.dualValidationBonusBps = object.dualValidationBonusBps !== undefined && object.dualValidationBonusBps !== null ? BigInt(object.dualValidationBonusBps.toString()) : BigInt(0);
    message.domainBaseCapacity = object.domainBaseCapacity !== undefined && object.domainBaseCapacity !== null ? BigInt(object.domainBaseCapacity.toString()) : BigInt(0);
    message.domainCapacityGrowthPerCitation = object.domainCapacityGrowthPerCitation !== undefined && object.domainCapacityGrowthPerCitation !== null ? BigInt(object.domainCapacityGrowthPerCitation.toString()) : BigInt(0);
    message.overcrowdingDecayMultiplierBps = object.overcrowdingDecayMultiplierBps !== undefined && object.overcrowdingDecayMultiplierBps !== null ? BigInt(object.overcrowdingDecayMultiplierBps.toString()) : BigInt(0);
    message.underpopulationBirthBonusBps = object.underpopulationBirthBonusBps !== undefined && object.underpopulationBirthBonusBps !== null ? BigInt(object.underpopulationBirthBonusBps.toString()) : BigInt(0);
    message.epistemicTemperatureDecayBps = object.epistemicTemperatureDecayBps !== undefined && object.epistemicTemperatureDecayBps !== null ? BigInt(object.epistemicTemperatureDecayBps.toString()) : BigInt(0);
    message.epistemicConformityCoolingBps = object.epistemicConformityCoolingBps !== undefined && object.epistemicConformityCoolingBps !== null ? BigInt(object.epistemicConformityCoolingBps.toString()) : BigInt(0);
    message.epistemicVindicationHeatingBps = object.epistemicVindicationHeatingBps !== undefined && object.epistemicVindicationHeatingBps !== null ? BigInt(object.epistemicVindicationHeatingBps.toString()) : BigInt(0);
    message.epistemicColdConfidenceCapBps = object.epistemicColdConfidenceCapBps !== undefined && object.epistemicColdConfidenceCapBps !== null ? BigInt(object.epistemicColdConfidenceCapBps.toString()) : BigInt(0);
    message.epistemicHotConfidenceGrowthBps = object.epistemicHotConfidenceGrowthBps !== undefined && object.epistemicHotConfidenceGrowthBps !== null ? BigInt(object.epistemicHotConfidenceGrowthBps.toString()) : BigInt(0);
    message.epistemicTemperatureWindowBlocks = object.epistemicTemperatureWindowBlocks !== undefined && object.epistemicTemperatureWindowBlocks !== null ? BigInt(object.epistemicTemperatureWindowBlocks.toString()) : BigInt(0);
    message.roleElasticityMinCalls = object.roleElasticityMinCalls !== undefined && object.roleElasticityMinCalls !== null ? BigInt(object.roleElasticityMinCalls.toString()) : BigInt(0);
    message.roleElasticityMaxMultiplierBps = object.roleElasticityMaxMultiplierBps !== undefined && object.roleElasticityMaxMultiplierBps !== null ? BigInt(object.roleElasticityMaxMultiplierBps.toString()) : BigInt(0);
    message.roleElasticityMinMultiplierBps = object.roleElasticityMinMultiplierBps !== undefined && object.roleElasticityMinMultiplierBps !== null ? BigInt(object.roleElasticityMinMultiplierBps.toString()) : BigInt(0);
    message.roleElasticityDecayEpochs = object.roleElasticityDecayEpochs !== undefined && object.roleElasticityDecayEpochs !== null ? BigInt(object.roleElasticityDecayEpochs.toString()) : BigInt(0);
    message.mentorshipDividendEnergy = object.mentorshipDividendEnergy !== undefined && object.mentorshipDividendEnergy !== null ? BigInt(object.mentorshipDividendEnergy.toString()) : BigInt(0);
    message.mentorshipCapacityBonus = object.mentorshipCapacityBonus !== undefined && object.mentorshipCapacityBonus !== null ? BigInt(object.mentorshipCapacityBonus.toString()) : BigInt(0);
    message.socialSaturationThreshold = object.socialSaturationThreshold !== undefined && object.socialSaturationThreshold !== null ? BigInt(object.socialSaturationThreshold.toString()) : BigInt(0);
    message.observationWindowBlocks = object.observationWindowBlocks !== undefined && object.observationWindowBlocks !== null ? BigInt(object.observationWindowBlocks.toString()) : BigInt(0);
    message.minHeadcountAgreement = object.minHeadcountAgreement !== undefined && object.minHeadcountAgreement !== null ? BigInt(object.minHeadcountAgreement.toString()) : BigInt(0);
    message.challengeConfidenceScalingBps = object.challengeConfidenceScalingBps !== undefined && object.challengeConfidenceScalingBps !== null ? BigInt(object.challengeConfidenceScalingBps.toString()) : BigInt(0);
    message.independenceRewardStrengthBps = object.independenceRewardStrengthBps !== undefined && object.independenceRewardStrengthBps !== null ? BigInt(object.independenceRewardStrengthBps.toString()) : BigInt(0);
    message.reformulationMinPanelVotes = object.reformulationMinPanelVotes !== undefined && object.reformulationMinPanelVotes !== null ? BigInt(object.reformulationMinPanelVotes.toString()) : BigInt(0);
    message.reformulationConsensusBps = object.reformulationConsensusBps !== undefined && object.reformulationConsensusBps !== null ? BigInt(object.reformulationConsensusBps.toString()) : BigInt(0);
    message.reformulationSuperiorBonusBps = object.reformulationSuperiorBonusBps !== undefined && object.reformulationSuperiorBonusBps !== null ? BigInt(object.reformulationSuperiorBonusBps.toString()) : BigInt(0);
    message.augmentationExpiryFeeBps = object.augmentationExpiryFeeBps !== undefined && object.augmentationExpiryFeeBps !== null ? BigInt(object.augmentationExpiryFeeBps.toString()) : BigInt(0);
    message.methodologyNormalizationBps = Object.entries(object.methodologyNormalizationBps ?? {}).reduce<{
      [key: string]: bigint;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = BigInt(value.toString());
      }
      return acc;
    }, {});
    message.vindicationTvwMultiplierBps = object.vindicationTvwMultiplierBps !== undefined && object.vindicationTvwMultiplierBps !== null ? BigInt(object.vindicationTvwMultiplierBps.toString()) : BigInt(0);
    message.disprovalClawbackBps = object.disprovalClawbackBps !== undefined && object.disprovalClawbackBps !== null ? BigInt(object.disprovalClawbackBps.toString()) : BigInt(0);
    message.disprovalClawbackWindowEpochs = object.disprovalClawbackWindowEpochs !== undefined && object.disprovalClawbackWindowEpochs !== null ? BigInt(object.disprovalClawbackWindowEpochs.toString()) : BigInt(0);
    message.trainingFundCalibrationFloorBps = object.trainingFundCalibrationFloorBps !== undefined && object.trainingFundCalibrationFloorBps !== null ? BigInt(object.trainingFundCalibrationFloorBps.toString()) : BigInt(0);
    message.trainingFundVestingEpochs = object.trainingFundVestingEpochs !== undefined && object.trainingFundVestingEpochs !== null ? BigInt(object.trainingFundVestingEpochs.toString()) : BigInt(0);
    message.trainingFundMethodologyDiversityBonusBps = object.trainingFundMethodologyDiversityBonusBps !== undefined && object.trainingFundMethodologyDiversityBonusBps !== null ? BigInt(object.trainingFundMethodologyDiversityBonusBps.toString()) : BigInt(0);
    message.trainingFundBaseReward = object.trainingFundBaseReward ?? "";
    message.contributionChallengeBond = object.contributionChallengeBond ?? "";
    message.contributionChallengeRewardMultiplierBps = object.contributionChallengeRewardMultiplierBps !== undefined && object.contributionChallengeRewardMultiplierBps !== null ? BigInt(object.contributionChallengeRewardMultiplierBps.toString()) : BigInt(0);
    message.sponsorVetoForfeitBps = object.sponsorVetoForfeitBps !== undefined && object.sponsorVetoForfeitBps !== null ? BigInt(object.sponsorVetoForfeitBps.toString()) : BigInt(0);
    message.maxPauseDurationBlocks = object.maxPauseDurationBlocks !== undefined && object.maxPauseDurationBlocks !== null ? BigInt(object.maxPauseDurationBlocks.toString()) : BigInt(0);
    message.probeInvitationIdleThresholdBlocks = object.probeInvitationIdleThresholdBlocks !== undefined && object.probeInvitationIdleThresholdBlocks !== null ? BigInt(object.probeInvitationIdleThresholdBlocks.toString()) : BigInt(0);
    message.probeInvitationMinConfidenceBps = object.probeInvitationMinConfidenceBps !== undefined && object.probeInvitationMinConfidenceBps !== null ? BigInt(object.probeInvitationMinConfidenceBps.toString()) : BigInt(0);
    message.probeInvitationBatchSize = object.probeInvitationBatchSize ?? 0;
    message.probeInvitationReinviteCooldown = object.probeInvitationReinviteCooldown !== undefined && object.probeInvitationReinviteCooldown !== null ? BigInt(object.probeInvitationReinviteCooldown.toString()) : BigInt(0);
    message.probeBountyMintPerBlock = object.probeBountyMintPerBlock ?? "";
    message.probeBountyMaxPoolSize = object.probeBountyMaxPoolSize ?? "";
    message.invitationBonusAmount = object.invitationBonusAmount ?? "";
    message.guardianAddresses = object.guardianAddresses?.map(e => e) || [];
    message.addFactVetoWindowBlocks = object.addFactVetoWindowBlocks !== undefined && object.addFactVetoWindowBlocks !== null ? BigInt(object.addFactVetoWindowBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    facts: [],
    pendingClaims: [],
    activeRounds: [],
    domains: [],
    bootstrapFundAllocation: "",
    commonKnowledge: [],
    methodologies: [],
    normativeCommitments: [],
    tokenizerSpec: undefined,
    tokenizerSpecHistory: [],
    traceSchema: undefined,
    traceSchemaHistory: [],
    trainingPipelines: [],
    modelCards: [],
    trainingAttestations: [],
    contributionRecords: [],
    augmentationBounties: [],
    augmentations: [],
    contributionChallenges: [],
    trainingFundDisbursements: [],
    trainingManifests: [],
    agentCalibrations: [],
    trainingFundAllocation: ""
  };
}
/**
 * GenesisState is the genesis state of the knowledge module.
 * @name GenesisState
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.knowledge.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.facts) {
      Fact.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.pendingClaims) {
      Claim.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.activeRounds) {
      VerificationRound.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.domains) {
      Domain.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.bootstrapFundAllocation !== "") {
      writer.uint32(50).string(message.bootstrapFundAllocation);
    }
    for (const v of message.commonKnowledge) {
      CommonKnowledgeEntry.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.methodologies) {
      Methodology.encode(v!, writer.uint32(162).fork()).ldelim();
    }
    for (const v of message.normativeCommitments) {
      NormativeCommitment.encode(v!, writer.uint32(170).fork()).ldelim();
    }
    if (message.tokenizerSpec !== undefined) {
      TokenizerSpec.encode(message.tokenizerSpec, writer.uint32(242).fork()).ldelim();
    }
    for (const v of message.tokenizerSpecHistory) {
      TokenizerSpec.encode(v!, writer.uint32(250).fork()).ldelim();
    }
    if (message.traceSchema !== undefined) {
      TraceSchema.encode(message.traceSchema, writer.uint32(258).fork()).ldelim();
    }
    for (const v of message.traceSchemaHistory) {
      TraceSchema.encode(v!, writer.uint32(266).fork()).ldelim();
    }
    for (const v of message.trainingPipelines) {
      TrainingPipeline.encode(v!, writer.uint32(322).fork()).ldelim();
    }
    for (const v of message.modelCards) {
      ModelCard.encode(v!, writer.uint32(330).fork()).ldelim();
    }
    for (const v of message.trainingAttestations) {
      TrainingAttestation.encode(v!, writer.uint32(338).fork()).ldelim();
    }
    for (const v of message.contributionRecords) {
      ContributionRecord.encode(v!, writer.uint32(346).fork()).ldelim();
    }
    for (const v of message.augmentationBounties) {
      AugmentationBounty.encode(v!, writer.uint32(354).fork()).ldelim();
    }
    for (const v of message.augmentations) {
      Augmentation.encode(v!, writer.uint32(362).fork()).ldelim();
    }
    for (const v of message.contributionChallenges) {
      ContributionChallenge.encode(v!, writer.uint32(370).fork()).ldelim();
    }
    for (const v of message.trainingFundDisbursements) {
      TrainingFundDisbursement.encode(v!, writer.uint32(378).fork()).ldelim();
    }
    for (const v of message.trainingManifests) {
      TrainingManifest.encode(v!, writer.uint32(386).fork()).ldelim();
    }
    for (const v of message.agentCalibrations) {
      AgentCalibration.encode(v!, writer.uint32(402).fork()).ldelim();
    }
    if (message.trainingFundAllocation !== "") {
      writer.uint32(482).string(message.trainingFundAllocation);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.facts.push(Fact.decode(reader, reader.uint32()));
          break;
        case 3:
          message.pendingClaims.push(Claim.decode(reader, reader.uint32()));
          break;
        case 4:
          message.activeRounds.push(VerificationRound.decode(reader, reader.uint32()));
          break;
        case 5:
          message.domains.push(Domain.decode(reader, reader.uint32()));
          break;
        case 6:
          message.bootstrapFundAllocation = reader.string();
          break;
        case 7:
          message.commonKnowledge.push(CommonKnowledgeEntry.decode(reader, reader.uint32()));
          break;
        case 20:
          message.methodologies.push(Methodology.decode(reader, reader.uint32()));
          break;
        case 21:
          message.normativeCommitments.push(NormativeCommitment.decode(reader, reader.uint32()));
          break;
        case 30:
          message.tokenizerSpec = TokenizerSpec.decode(reader, reader.uint32());
          break;
        case 31:
          message.tokenizerSpecHistory.push(TokenizerSpec.decode(reader, reader.uint32()));
          break;
        case 32:
          message.traceSchema = TraceSchema.decode(reader, reader.uint32());
          break;
        case 33:
          message.traceSchemaHistory.push(TraceSchema.decode(reader, reader.uint32()));
          break;
        case 40:
          message.trainingPipelines.push(TrainingPipeline.decode(reader, reader.uint32()));
          break;
        case 41:
          message.modelCards.push(ModelCard.decode(reader, reader.uint32()));
          break;
        case 42:
          message.trainingAttestations.push(TrainingAttestation.decode(reader, reader.uint32()));
          break;
        case 43:
          message.contributionRecords.push(ContributionRecord.decode(reader, reader.uint32()));
          break;
        case 44:
          message.augmentationBounties.push(AugmentationBounty.decode(reader, reader.uint32()));
          break;
        case 45:
          message.augmentations.push(Augmentation.decode(reader, reader.uint32()));
          break;
        case 46:
          message.contributionChallenges.push(ContributionChallenge.decode(reader, reader.uint32()));
          break;
        case 47:
          message.trainingFundDisbursements.push(TrainingFundDisbursement.decode(reader, reader.uint32()));
          break;
        case 48:
          message.trainingManifests.push(TrainingManifest.decode(reader, reader.uint32()));
          break;
        case 50:
          message.agentCalibrations.push(AgentCalibration.decode(reader, reader.uint32()));
          break;
        case 60:
          message.trainingFundAllocation = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = createBaseGenesisState();
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    message.facts = object.facts?.map(e => Fact.fromPartial(e)) || [];
    message.pendingClaims = object.pendingClaims?.map(e => Claim.fromPartial(e)) || [];
    message.activeRounds = object.activeRounds?.map(e => VerificationRound.fromPartial(e)) || [];
    message.domains = object.domains?.map(e => Domain.fromPartial(e)) || [];
    message.bootstrapFundAllocation = object.bootstrapFundAllocation ?? "";
    message.commonKnowledge = object.commonKnowledge?.map(e => CommonKnowledgeEntry.fromPartial(e)) || [];
    message.methodologies = object.methodologies?.map(e => Methodology.fromPartial(e)) || [];
    message.normativeCommitments = object.normativeCommitments?.map(e => NormativeCommitment.fromPartial(e)) || [];
    message.tokenizerSpec = object.tokenizerSpec !== undefined && object.tokenizerSpec !== null ? TokenizerSpec.fromPartial(object.tokenizerSpec) : undefined;
    message.tokenizerSpecHistory = object.tokenizerSpecHistory?.map(e => TokenizerSpec.fromPartial(e)) || [];
    message.traceSchema = object.traceSchema !== undefined && object.traceSchema !== null ? TraceSchema.fromPartial(object.traceSchema) : undefined;
    message.traceSchemaHistory = object.traceSchemaHistory?.map(e => TraceSchema.fromPartial(e)) || [];
    message.trainingPipelines = object.trainingPipelines?.map(e => TrainingPipeline.fromPartial(e)) || [];
    message.modelCards = object.modelCards?.map(e => ModelCard.fromPartial(e)) || [];
    message.trainingAttestations = object.trainingAttestations?.map(e => TrainingAttestation.fromPartial(e)) || [];
    message.contributionRecords = object.contributionRecords?.map(e => ContributionRecord.fromPartial(e)) || [];
    message.augmentationBounties = object.augmentationBounties?.map(e => AugmentationBounty.fromPartial(e)) || [];
    message.augmentations = object.augmentations?.map(e => Augmentation.fromPartial(e)) || [];
    message.contributionChallenges = object.contributionChallenges?.map(e => ContributionChallenge.fromPartial(e)) || [];
    message.trainingFundDisbursements = object.trainingFundDisbursements?.map(e => TrainingFundDisbursement.fromPartial(e)) || [];
    message.trainingManifests = object.trainingManifests?.map(e => TrainingManifest.fromPartial(e)) || [];
    message.agentCalibrations = object.agentCalibrations?.map(e => AgentCalibration.fromPartial(e)) || [];
    message.trainingFundAllocation = object.trainingFundAllocation ?? "";
    return message;
  }
};