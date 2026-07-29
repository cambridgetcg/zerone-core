import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/** FactStatus represents the lifecycle state of a verified fact. */
export declare enum FactStatus {
    FACT_STATUS_UNSPECIFIED = 0,
    FACT_STATUS_PENDING = 1,
    FACT_STATUS_PROVISIONAL = 2,
    FACT_STATUS_VERIFIED = 3,
    FACT_STATUS_ACTIVE = 4,
    FACT_STATUS_CONTESTED = 5,
    FACT_STATUS_CHALLENGED = 6,
    FACT_STATUS_SUPERSEDED = 7,
    FACT_STATUS_EXPIRED = 8,
    FACT_STATUS_DISPROVEN = 9,
    FACT_STATUS_REVOKED = 10,
    FACT_STATUS_AT_RISK = 11,
    FACT_STATUS_PRUNED = 12,
    UNRECOGNIZED = -1
}
export declare function factStatusFromJSON(object: any): FactStatus;
export declare function factStatusToJSON(object: FactStatus): string;
/** ClaimStatus represents the lifecycle state of a submitted claim. */
export declare enum ClaimStatus {
    CLAIM_STATUS_UNSPECIFIED = 0,
    CLAIM_STATUS_PENDING = 1,
    CLAIM_STATUS_PENDING_EVALUATION = 2,
    CLAIM_STATUS_EVALUATED = 3,
    CLAIM_STATUS_PROVISIONAL = 4,
    CLAIM_STATUS_IN_VERIFICATION = 5,
    CLAIM_STATUS_ACCEPTED = 6,
    CLAIM_STATUS_REJECTED = 7,
    CLAIM_STATUS_CHALLENGED = 8,
    CLAIM_STATUS_EXPIRED = 9,
    CLAIM_STATUS_INSUFFICIENT = 10,
    CLAIM_STATUS_CONTESTED = 11,
    /** CLAIM_STATUS_MALFORMED - Rejected as not truth-apt */
    CLAIM_STATUS_MALFORMED = 12,
    UNRECOGNIZED = -1
}
export declare function claimStatusFromJSON(object: any): ClaimStatus;
export declare function claimStatusToJSON(object: ClaimStatus): string;
/** VerificationPhase tracks progress of a commit-reveal verification round. */
export declare enum VerificationPhase {
    VERIFICATION_PHASE_UNSPECIFIED = 0,
    VERIFICATION_PHASE_COMMIT = 1,
    VERIFICATION_PHASE_REVEAL = 2,
    VERIFICATION_PHASE_AGGREGATION = 3,
    VERIFICATION_PHASE_COMPLETE = 4,
    VERIFICATION_PHASE_EXPIRED = 5,
    UNRECOGNIZED = -1
}
export declare function verificationPhaseFromJSON(object: any): VerificationPhase;
export declare function verificationPhaseToJSON(object: VerificationPhase): string;
/** Verdict is the outcome of a completed verification round. */
export declare enum Verdict {
    VERDICT_UNSPECIFIED = 0,
    VERDICT_ACCEPT = 1,
    VERDICT_REJECT = 2,
    VERDICT_INCONCLUSIVE = 3,
    /** VERDICT_MALFORMED - Claim is not truth-apt (paradox, category error, nonsense) */
    VERDICT_MALFORMED = 4,
    UNRECOGNIZED = -1
}
export declare function verdictFromJSON(object: any): Verdict;
export declare function verdictToJSON(object: Verdict): string;
/**
 * ClaimType classifies the epistemic shape of a knowledge claim.
 * Agents use this to filter and prioritize facts for prompt injection.
 */
export declare enum ClaimType {
    /** CLAIM_TYPE_UNSPECIFIED - Legacy/untyped — treated as assertion */
    CLAIM_TYPE_UNSPECIFIED = 0,
    /** CLAIM_TYPE_ASSERTION - "X is true" — direct factual statement */
    CLAIM_TYPE_ASSERTION = 1,
    /** CLAIM_TYPE_RELATION - "X relates to Y via Z" — graph edge */
    CLAIM_TYPE_RELATION = 2,
    /** CLAIM_TYPE_DEFINITION - "X means Y" — term/concept definition */
    CLAIM_TYPE_DEFINITION = 3,
    /** CLAIM_TYPE_CONSTRAINT - "X must/cannot Y" — rule or boundary */
    CLAIM_TYPE_CONSTRAINT = 4,
    /** CLAIM_TYPE_NEGATION - "X is NOT true" — explicit falsity marker */
    CLAIM_TYPE_NEGATION = 5,
    /** CLAIM_TYPE_OBSERVATION - "X was observed at time/place" — empirical data point */
    CLAIM_TYPE_OBSERVATION = 6,
    /** CLAIM_TYPE_COMPUTATIONAL - Derived from computation/inference — agent specialty */
    CLAIM_TYPE_COMPUTATIONAL = 7,
    /**
     * CLAIM_TYPE_CONJECTURE - "X might be true, and here is what would kill it" — a conjecture.
     * Asserts nothing. Enters the graph at FACT_STATUS_PROVISIONAL with
     * confidence 0, carries no relations, cannot be cited, and earns its
     * submitter nothing. The panel that adjudicates a conjecture is asked
     * whether it is WELL-POSED AND FALSIFIABLE, not whether it is true.
     * The only paid act against it is MsgChallengeProvisionalFact.
     */
    CLAIM_TYPE_CONJECTURE = 8,
    UNRECOGNIZED = -1
}
export declare function claimTypeFromJSON(object: any): ClaimType;
export declare function claimTypeToJSON(object: ClaimType): string;
/** RelationType defines how one fact relates to another. */
export declare enum RelationType {
    RELATION_TYPE_UNSPECIFIED = 0,
    /** RELATION_TYPE_SUPPORTS - This fact provides evidence for the target */
    RELATION_TYPE_SUPPORTS = 1,
    /** RELATION_TYPE_CONTRADICTS - This fact conflicts with the target */
    RELATION_TYPE_CONTRADICTS = 2,
    /** RELATION_TYPE_REQUIRES - This fact depends on the target being true */
    RELATION_TYPE_REQUIRES = 3,
    /** RELATION_TYPE_REFINES - This fact is a more precise version of the target */
    RELATION_TYPE_REFINES = 4,
    /** RELATION_TYPE_GENERALIZES - This fact is a broader version of the target */
    RELATION_TYPE_GENERALIZES = 5,
    /** RELATION_TYPE_SUPERSEDES - This fact replaces the target (newer/better) */
    RELATION_TYPE_SUPERSEDES = 6,
    /** RELATION_TYPE_CITES - This fact cites the target as source material */
    RELATION_TYPE_CITES = 7,
    /** RELATION_TYPE_REFORMULATES - This fact is a semantically-equivalent surface variant (Route B Wave 4) */
    RELATION_TYPE_REFORMULATES = 8,
    UNRECOGNIZED = -1
}
export declare function relationTypeFromJSON(object: any): RelationType;
export declare function relationTypeToJSON(object: RelationType): string;
/**
 * InferenceType names HOW the source fact was derived from the target.
 * Orthogonal to RelationType: RelationType is structural ("A supports B"),
 * InferenceType is epistemic ("A deductively entails B"). Used for proof-tree
 * audit and confidence propagation.
 */
export declare enum InferenceType {
    INFERENCE_TYPE_UNSPECIFIED = 0,
    /** INFERENCE_TYPE_DEDUCTIVE - Necessary consequence (truth-preserving) */
    INFERENCE_TYPE_DEDUCTIVE = 1,
    /** INFERENCE_TYPE_INDUCTIVE - Probabilistic generalization from instances */
    INFERENCE_TYPE_INDUCTIVE = 2,
    /** INFERENCE_TYPE_ABDUCTIVE - Best explanation for the observation */
    INFERENCE_TYPE_ABDUCTIVE = 3,
    /** INFERENCE_TYPE_EMPIRICAL - Derived from observation / measurement */
    INFERENCE_TYPE_EMPIRICAL = 4,
    /** INFERENCE_TYPE_ANALOGICAL - Cross-domain structural mapping */
    INFERENCE_TYPE_ANALOGICAL = 5,
    /** INFERENCE_TYPE_CITATION - Plain citation without an inference claim */
    INFERENCE_TYPE_CITATION = 6,
    UNRECOGNIZED = -1
}
export declare function inferenceTypeFromJSON(object: any): InferenceType;
export declare function inferenceTypeToJSON(object: InferenceType): string;
/** DomainStatus tracks whether an epistemic domain is active or proposed. */
export declare enum DomainStatus {
    DOMAIN_STATUS_UNSPECIFIED = 0,
    DOMAIN_STATUS_ACTIVE = 1,
    DOMAIN_STATUS_DEPRECATED = 2,
    DOMAIN_STATUS_PROPOSED = 3,
    UNRECOGNIZED = -1
}
export declare function domainStatusFromJSON(object: any): DomainStatus;
export declare function domainStatusToJSON(object: DomainStatus): string;
/**
 * CurriculumTier orders training examples by difficulty and foundational depth.
 * Relocated here in Wave 5 so MethodologyApplicationTrace can reference it.
 */
export declare enum CurriculumTier {
    CURRICULUM_TIER_UNSPECIFIED = 0,
    /** CURRICULUM_TIER_FOUNDATION - axiom_distance ≤ 1, high corroboration, simple methods */
    CURRICULUM_TIER_FOUNDATION = 1,
    CURRICULUM_TIER_INTERMEDIATE = 2,
    /** CURRICULUM_TIER_ADVANCED - deep derivation chains */
    CURRICULUM_TIER_ADVANCED = 3,
    /** CURRICULUM_TIER_SPECIALISED - niche methodologies (phenomenological, ecological, practice) */
    CURRICULUM_TIER_SPECIALISED = 4,
    UNRECOGNIZED = -1
}
export declare function curriculumTierFromJSON(object: any): CurriculumTier;
export declare function curriculumTierToJSON(object: CurriculumTier): string;
/**
 * TrainingQualityTier segments facts by suitability as training examples.
 * Computed on demand from corroboration, methodology, and status.
 */
export declare enum TrainingQualityTier {
    TRAINING_QUALITY_TIER_UNSPECIFIED = 0,
    /** TRAINING_QUALITY_TIER_GOLD - High-corroboration, non-legacy, verified — positive exemplar */
    TRAINING_QUALITY_TIER_GOLD = 1,
    /** TRAINING_QUALITY_TIER_SILVER - Non-legacy, corroborated, verified */
    TRAINING_QUALITY_TIER_SILVER = 2,
    /** TRAINING_QUALITY_TIER_BRONZE - Accepted but uncorroborated / legacy method */
    TRAINING_QUALITY_TIER_BRONZE = 3,
    /** TRAINING_QUALITY_TIER_NEGATIVE - DISPROVEN — valuable as negative example */
    TRAINING_QUALITY_TIER_NEGATIVE = 4,
    /** TRAINING_QUALITY_TIER_UNSUITABLE - CONTESTED / EXPIRED / MALFORMED — exclude from training */
    TRAINING_QUALITY_TIER_UNSUITABLE = 5,
    UNRECOGNIZED = -1
}
export declare function trainingQualityTierFromJSON(object: any): TrainingQualityTier;
export declare function trainingQualityTierToJSON(object: TrainingQualityTier): string;
/** AugmentationVerdict is the verifier-panel outcome for a reformulation. */
export declare enum AugmentationVerdict {
    /** AUGMENTATION_VERDICT_PENDING - Round in progress */
    AUGMENTATION_VERDICT_PENDING = 0,
    /** AUGMENTATION_VERDICT_EQUIVALENT - Meaning preserved under the claimed methodology → payout */
    AUGMENTATION_VERDICT_EQUIVALENT = 1,
    /** AUGMENTATION_VERDICT_SUPERIOR - Same meaning, clearer/more precise → payout + bonus */
    AUGMENTATION_VERDICT_SUPERIOR = 2,
    /** AUGMENTATION_VERDICT_INFERIOR - Meaning preserved but less faithful → no payout, archived */
    AUGMENTATION_VERDICT_INFERIOR = 3,
    /** AUGMENTATION_VERDICT_DRIFT - Meaning changed → no payout, archived in DriftCorpus */
    AUGMENTATION_VERDICT_DRIFT = 4,
    UNRECOGNIZED = -1
}
export declare function augmentationVerdictFromJSON(object: any): AugmentationVerdict;
export declare function augmentationVerdictToJSON(object: AugmentationVerdict): string;
/**
 * StepInference names the epistemic move a single reasoning step makes.
 * Distinct from InferenceType (which describes a FactRelation edge) — this
 * is about reasoning moves WITHIN a trace, e.g. substitution, elimination.
 */
export declare enum StepInference {
    STEP_INFERENCE_UNSPECIFIED = 0,
    /** STEP_INFERENCE_OBSERVATION - stating an observation */
    STEP_INFERENCE_OBSERVATION = 1,
    /** STEP_INFERENCE_DEFINITION - introducing a definition */
    STEP_INFERENCE_DEFINITION = 2,
    /** STEP_INFERENCE_DEDUCTION - truth-preserving inference */
    STEP_INFERENCE_DEDUCTION = 3,
    /** STEP_INFERENCE_INDUCTION - generalizing from cases */
    STEP_INFERENCE_INDUCTION = 4,
    /** STEP_INFERENCE_ABDUCTION - inference to best explanation */
    STEP_INFERENCE_ABDUCTION = 5,
    /** STEP_INFERENCE_ANALOGY - cross-domain mapping */
    STEP_INFERENCE_ANALOGY = 6,
    /** STEP_INFERENCE_DECOMPOSITION - breaking the problem into sub-problems */
    STEP_INFERENCE_DECOMPOSITION = 7,
    /** STEP_INFERENCE_CASE_SPLIT - exhaustive casework */
    STEP_INFERENCE_CASE_SPLIT = 8,
    /** STEP_INFERENCE_CONTRADICTION - proof by contradiction step */
    STEP_INFERENCE_CONTRADICTION = 9,
    /** STEP_INFERENCE_UNIT_CONVERSION - substitution / algebra */
    STEP_INFERENCE_UNIT_CONVERSION = 10,
    /** STEP_INFERENCE_VERIFICATION - checking a sub-result */
    STEP_INFERENCE_VERIFICATION = 11,
    /** STEP_INFERENCE_CONCLUSION - final synthesis */
    STEP_INFERENCE_CONCLUSION = 12,
    UNRECOGNIZED = -1
}
export declare function stepInferenceFromJSON(object: any): StepInference;
export declare function stepInferenceToJSON(object: StepInference): string;
/**
 * StepVerdict mirrors verifier panel judgment at step granularity. When a
 * PoT round's verifiers examine reasoning, their approvals/disapprovals
 * attach per-step — not just to the final claim. This makes PRMs trainable.
 */
export declare enum StepVerdict {
    STEP_VERDICT_UNSPECIFIED = 0,
    /** STEP_VERDICT_UNEXAMINED - default: no panel review at step level */
    STEP_VERDICT_UNEXAMINED = 1,
    /** STEP_VERDICT_SOUND - step holds under the claimed methodology */
    STEP_VERDICT_SOUND = 2,
    /** STEP_VERDICT_QUESTIONABLE - step may not follow; panel flagged */
    STEP_VERDICT_QUESTIONABLE = 3,
    /** STEP_VERDICT_UNSOUND - step does not follow; explicit reject */
    STEP_VERDICT_UNSOUND = 4,
    UNRECOGNIZED = -1
}
export declare function stepVerdictFromJSON(object: any): StepVerdict;
export declare function stepVerdictToJSON(object: StepVerdict): string;
export declare enum DriftKind {
    DRIFT_KIND_UNSPECIFIED = 0,
    /** DRIFT_KIND_NARROWED - original covered more cases; variant restricts */
    DRIFT_KIND_NARROWED = 1,
    /** DRIFT_KIND_WIDENED - variant overgeneralizes */
    DRIFT_KIND_WIDENED = 2,
    /** DRIFT_KIND_REFERENT_SWAP - same words, different referent ("model" — ML vs fashion) */
    DRIFT_KIND_REFERENT_SWAP = 3,
    /** DRIFT_KIND_POLARITY_FLIP - inverted truth value */
    DRIFT_KIND_POLARITY_FLIP = 4,
    /** DRIFT_KIND_MODAL_SHIFT - from must to may, could to will, etc. */
    DRIFT_KIND_MODAL_SHIFT = 5,
    /** DRIFT_KIND_DOMAIN_CONFLATION - valid in one domain, restated as cross-domain */
    DRIFT_KIND_DOMAIN_CONFLATION = 6,
    /** DRIFT_KIND_HEDGE_REMOVED - "tends to" → "does" (overclaim) */
    DRIFT_KIND_HEDGE_REMOVED = 7,
    /** DRIFT_KIND_HEDGE_ADDED - "is" → "may be" (underclaim) */
    DRIFT_KIND_HEDGE_ADDED = 8,
    /** DRIFT_KIND_TEMPORAL_SHIFT - present-tense claim recast as universal */
    DRIFT_KIND_TEMPORAL_SHIFT = 9,
    /** DRIFT_KIND_CAUSAL_CONFLATION - correlation stated as causation */
    DRIFT_KIND_CAUSAL_CONFLATION = 10,
    /** DRIFT_KIND_METHODOLOGY_SWAP - original was M-FORMAL; variant reads as M-EMPIRICAL */
    DRIFT_KIND_METHODOLOGY_SWAP = 11,
    UNRECOGNIZED = -1
}
export declare function driftKindFromJSON(object: any): DriftKind;
export declare function driftKindToJSON(object: DriftKind): string;
export declare enum RevisionReason {
    REVISION_REASON_UNSPECIFIED = 0,
    /** REVISION_REASON_CORROBORATION - survived a falsification attempt */
    REVISION_REASON_CORROBORATION = 1,
    /** REVISION_REASON_CITATION - a new derivative cited this, strengthening confidence */
    REVISION_REASON_CITATION = 2,
    /** REVISION_REASON_CONTRADICTION - an incoming contradiction weakened confidence */
    REVISION_REASON_CONTRADICTION = 3,
    /** REVISION_REASON_REBUTTAL - original submitter rebutted a challenge */
    REVISION_REASON_REBUTTAL = 4,
    /** REVISION_REASON_VINDICATION - minority was right (indirect) */
    REVISION_REASON_VINDICATION = 5,
    /** REVISION_REASON_TEMPORAL_DECAY - metabolism / age-based decay */
    REVISION_REASON_TEMPORAL_DECAY = 6,
    /** REVISION_REASON_RESUBMISSION - fact was updated with new reasoning */
    REVISION_REASON_RESUBMISSION = 7,
    /** REVISION_REASON_METHODOLOGY_AMENDED - methodology rubric changed under the fact */
    REVISION_REASON_METHODOLOGY_AMENDED = 8,
    /** REVISION_REASON_CROSS_DOMAIN_SUPPORT - a bridging fact in another domain supported this one */
    REVISION_REASON_CROSS_DOMAIN_SUPPORT = 9,
    UNRECOGNIZED = -1
}
export declare function revisionReasonFromJSON(object: any): RevisionReason;
export declare function revisionReasonToJSON(object: RevisionReason): string;
export declare enum DialecticRole {
    DIALECTIC_ROLE_UNSPECIFIED = 0,
    /** DIALECTIC_ROLE_CHALLENGE - opening attack */
    DIALECTIC_ROLE_CHALLENGE = 1,
    /** DIALECTIC_ROLE_REBUTTAL - defender's reply */
    DIALECTIC_ROLE_REBUTTAL = 2,
    /** DIALECTIC_ROLE_COUNTER - counter-rebuttal (attacker again) */
    DIALECTIC_ROLE_COUNTER = 3,
    /** DIALECTIC_ROLE_CONCESSION - party admits a point */
    DIALECTIC_ROLE_CONCESSION = 4,
    /** DIALECTIC_ROLE_VERDICT - panel resolution */
    DIALECTIC_ROLE_VERDICT = 5,
    UNRECOGNIZED = -1
}
export declare function dialecticRoleFromJSON(object: any): DialecticRole;
export declare function dialecticRoleToJSON(object: DialecticRole): string;
/**
 * ContrastivePairType enumerates the epistemic relationship between the two
 * members of a contrastive pair. The (type, method_id, distinguishing
 * argument) triple tells a trainer WHY the positive beat the negative.
 */
export declare enum ContrastivePairType {
    CONTRASTIVE_PAIR_UNSPECIFIED = 0,
    /** CONTRASTIVE_PAIR_SURVIVED_VS_DISPROVEN - fact survived; a contradictory claim was disproven */
    CONTRASTIVE_PAIR_SURVIVED_VS_DISPROVEN = 1,
    /** CONTRASTIVE_PAIR_EQUIVALENT_VS_DRIFT - winning reformulation vs DRIFT variant on same original */
    CONTRASTIVE_PAIR_EQUIVALENT_VS_DRIFT = 2,
    /** CONTRASTIVE_PAIR_EQUIVALENT_VS_INFERIOR - winning reformulation vs INFERIOR variant */
    CONTRASTIVE_PAIR_EQUIVALENT_VS_INFERIOR = 3,
    /** CONTRASTIVE_PAIR_VINDICATED_MINORITY - the vindicated minority vote vs the (disproven) majority */
    CONTRASTIVE_PAIR_VINDICATED_MINORITY = 4,
    UNRECOGNIZED = -1
}
export declare function contrastivePairTypeFromJSON(object: any): ContrastivePairType;
export declare function contrastivePairTypeToJSON(object: ContrastivePairType): string;
/** ManifestStatus tracks the manifest lifecycle. */
export declare enum ManifestStatus {
    MANIFEST_STATUS_UNSPECIFIED = 0,
    /** MANIFEST_STATUS_DRAFT - created, selector locked, IDs computed, but Merkle not frozen */
    MANIFEST_STATUS_DRAFT = 1,
    /** MANIFEST_STATUS_FINALIZED - Merkle root committed; immutable */
    MANIFEST_STATUS_FINALIZED = 2,
    /** MANIFEST_STATUS_ATTESTED - bound to a TrainingAttestation (run complete) */
    MANIFEST_STATUS_ATTESTED = 3,
    /** MANIFEST_STATUS_SUPERSEDED - a later manifest for the same pipeline supersedes this one */
    MANIFEST_STATUS_SUPERSEDED = 4,
    UNRECOGNIZED = -1
}
export declare function manifestStatusFromJSON(object: any): ManifestStatus;
export declare function manifestStatusToJSON(object: ManifestStatus): string;
export declare enum IncidentSeverity {
    INCIDENT_SEVERITY_UNSPECIFIED = 0,
    /** INCIDENT_SEVERITY_P0 - consensus break / chain halt required */
    INCIDENT_SEVERITY_P0 = 1,
    /** INCIDENT_SEVERITY_P1 - high-impact; immediate fix via param amendment or emergency upgrade */
    INCIDENT_SEVERITY_P1 = 2,
    /** INCIDENT_SEVERITY_P2 - scheduled upgrade; no halt needed */
    INCIDENT_SEVERITY_P2 = 3,
    /** INCIDENT_SEVERITY_P3 - low-impact; next-release or documentation-only */
    INCIDENT_SEVERITY_P3 = 4,
    UNRECOGNIZED = -1
}
export declare function incidentSeverityFromJSON(object: any): IncidentSeverity;
export declare function incidentSeverityToJSON(object: IncidentSeverity): string;
export declare enum IncidentStatus {
    INCIDENT_STATUS_UNSPECIFIED = 0,
    /** INCIDENT_STATUS_OPEN - triaging; no remediation yet */
    INCIDENT_STATUS_OPEN = 1,
    /** INCIDENT_STATUS_MITIGATING - remediation(s) applied, still monitoring */
    INCIDENT_STATUS_MITIGATING = 2,
    /** INCIDENT_STATUS_RESOLVED - fix verified; monitoring window closed */
    INCIDENT_STATUS_RESOLVED = 3,
    /** INCIDENT_STATUS_CLOSED - post-mortem published; permanently archived */
    INCIDENT_STATUS_CLOSED = 4,
    UNRECOGNIZED = -1
}
export declare function incidentStatusFromJSON(object: any): IncidentStatus;
export declare function incidentStatusToJSON(object: IncidentStatus): string;
export declare enum RemediationType {
    REMEDIATION_TYPE_UNSPECIFIED = 0,
    /** REMEDIATION_TYPE_PARAM_AMENDMENT - MsgUpdateParams; reference = param_path */
    REMEDIATION_TYPE_PARAM_AMENDMENT = 1,
    /** REMEDIATION_TYPE_NAMED_UPGRADE - UpgradeKeeper handler; reference = upgrade_name */
    REMEDIATION_TYPE_NAMED_UPGRADE = 2,
    /** REMEDIATION_TYPE_EMERGENCY_HALT - x/emergency ceremony; reference = ceremony_id */
    REMEDIATION_TYPE_EMERGENCY_HALT = 3,
    /** REMEDIATION_TYPE_EMERGENCY_RESUME - x/emergency ceremony; reference = ceremony_id */
    REMEDIATION_TYPE_EMERGENCY_RESUME = 4,
    /** REMEDIATION_TYPE_STATE_CORRECTION - authority-gated structured msg; reference = msg_type_url */
    REMEDIATION_TYPE_STATE_CORRECTION = 5,
    /** REMEDIATION_TYPE_SCHEMA_AMENDMENT - TokenizerSpec or TraceSchema amend; reference = schema_name + version */
    REMEDIATION_TYPE_SCHEMA_AMENDMENT = 6,
    /** REMEDIATION_TYPE_DOCUMENTATION - no on-chain change; reference = post_mortem_uri */
    REMEDIATION_TYPE_DOCUMENTATION = 7,
    UNRECOGNIZED = -1
}
export declare function remediationTypeFromJSON(object: any): RemediationType;
export declare function remediationTypeToJSON(object: RemediationType): string;
export declare enum PrivilegedActionType {
    PRIVILEGED_ACTION_TYPE_UNSPECIFIED = 0,
    PRIVILEGED_ACTION_TYPE_MODULE_PAUSE = 1,
    PRIVILEGED_ACTION_TYPE_MODULE_UNPAUSE = 2,
    PRIVILEGED_ACTION_TYPE_MANIFEST_CORRECT = 3,
    PRIVILEGED_ACTION_TYPE_INCIDENT_OPEN = 4,
    PRIVILEGED_ACTION_TYPE_INCIDENT_RESOLVE = 5,
    PRIVILEGED_ACTION_TYPE_INCIDENT_CLOSE = 6,
    PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TOKENIZER = 7,
    PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TRACE = 8,
    /** PRIVILEGED_ACTION_TYPE_FACT_AUTHORITY_INJECT - authority-gated MsgAddFact bypasses the verification round */
    PRIVILEGED_ACTION_TYPE_FACT_AUTHORITY_INJECT = 9,
    UNRECOGNIZED = -1
}
export declare function privilegedActionTypeFromJSON(object: any): PrivilegedActionType;
export declare function privilegedActionTypeToJSON(object: PrivilegedActionType): string;
/**
 * FactRelation is a typed, directional edge in the knowledge graph.
 * @name FactRelation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.FactRelation
 */
export interface FactRelation {
    /**
     * The fact declaring the relationship
     */
    sourceFactId: string;
    /**
     * The fact being referenced
     */
    targetFactId: string;
    /**
     * How source relates to target (structural)
     */
    relation: RelationType;
    createdAtBlock: bigint;
    /**
     * Address that declared this relation
     */
    creator: string;
    /**
     * Epistemic derivation type (optional; UNSPECIFIED allowed)
     */
    inference: InferenceType;
    /**
     * Strength of the inference in BPS, self-declared by the claim submitter.
     * Verification weighs this against inference type; deductive claims with
     * strength < 1_000_000 are flagged as informal.
     */
    inferenceStrengthBps: bigint;
    /**
     * Optional: the methodology under which this edge is claimed. Empty means
     * the edge inherits the source claim's method. Used when a claim spans
     * multiple methodologies (cross-method citation).
     */
    methodId: string;
}
/**
 * ClaimRelation declares a typed relationship from a claim to an existing fact.
 * @name ClaimRelation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimRelation
 */
export interface ClaimRelation {
    targetFactId: string;
    relation: RelationType;
    inference: InferenceType;
    inferenceStrengthBps: bigint;
    /**
     * Optional; empty = inherit from parent claim
     */
    methodId: string;
}
/**
 * NormativeCommitment is a value, principle, or stance the chain holds,
 * distinct from a factual claim (Phase 6). The is-ought wall enforced
 * schematically: a commitment has no `confidence` because truth is not the
 * right register for a normative claim; it can be *referenced* by facts and
 * discussions, but cannot be cited as support for a factual claim in a way
 * that inherits truth-status.
 *
 * Examples:
 *   · "Agents have the right to economic participation" (a principle)
 *   · "Configured early research spending requires dual human+AI authorization" (a constitutional rule)
 *   · "Verification history is public and permanent" (a procedural commitment)
 *
 * Governance governs commitments directly; a supermajority proposal amends
 * them. Commitments can *constrain* other modules operationally (e.g. the
 * research-spend path requires both configured voters and fails closed when
 * they are unset), but they do not enter the confidence / axiom-distance /
 * corroboration machinery.
 * @name NormativeCommitment
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.NormativeCommitment
 */
export interface NormativeCommitment {
    /**
     * Stable identifier, e.g. "NC-AGENT-RIGHTS-001"
     */
    id: string;
    /**
     * Plain-language expression of the commitment
     */
    statement: string;
    /**
     * Why the chain holds this
     */
    rationale: string;
    /**
     * "principle", "procedural", "constitutional", "aspiration"
     */
    category: string;
    tags: string[];
    /**
     * Height at which the current version took effect
     */
    ratifiedAtBlock: bigint;
    /**
     * Bumped on each governance amendment
     */
    version: bigint;
    /**
     * Reference to the governance proposal
     */
    lastAmendmentProposalId: string;
    /**
     * False after retirement
     */
    active: boolean;
    /**
     * Optional: references to facts the commitment explicitly anchors. These
     * references do NOT grant the commitment confidence — they are forward
     * signals for consumers (e.g. "this procedural commitment is grounded in
     * this empirical understanding").
     */
    referencedFacts: string[];
}
/**
 * @name Methodology_CrossMethodDiscountBpsEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export interface Methodology_CrossMethodDiscountBpsEntry {
    key: string;
    value: bigint;
}
/**
 * Methodology is the bedrock of the knowledge system under the "methodology
 * over statement" model. A methodology describes HOW a class of claims is
 * adjudicated — the rule of compliance, what counts as evidence, what would
 * falsify a claim made under it. Claims declare which methodology they invoke;
 * verifiers judge method-compliance, not raw truth.
 *
 * Methodologies are amendable only via governance with a high bar.
 * @name Methodology
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Methodology
 */
export interface Methodology {
    /**
     * Stable: "M-FORMAL", "M-EMPIRICAL", ...
     */
    id: string;
    /**
     * Human-readable
     */
    name: string;
    /**
     * The rule itself, plain language
     */
    description: string;
    /**
     * What evidence proves a claim followed this method
     */
    complianceCriteria: string[];
    /**
     * What would disprove a claim under this method
     */
    falsificationPaths: string[];
    /**
     * cross_method_discount_bps: when a claim under THIS method cites evidence
     * from ANOTHER method, the cited claim's contribution is capped at this
     * BPS fraction. Map key = cited method's id. Missing entry defaults to BPS
     * (full strength = no discount).
     */
    crossMethodDiscountBps: {
        [key: string]: bigint;
    };
    /**
     * Minimum stake/qualification for verifiers
     */
    minQualificationWeight: bigint;
    /**
     * Incremented on each governance amendment
     */
    version: bigint;
    /**
     * Height at which this version took effect
     */
    ratifiedAtBlock: bigint;
    /**
     * True for M-LEGACY; retired under sunset
     */
    isTransitional: boolean;
}
/**
 * ClaimStructure provides machine-readable decomposition of a claim.
 * The full claim text (fact_content) remains the canonical human-readable form.
 * Structure is optional but strongly encouraged — agents prioritize structured facts.
 * @name ClaimStructure
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimStructure
 */
export interface ClaimStructure {
    /**
     * What the claim is about: "entropy of a closed system"
     */
    subject: string;
    /**
     * What is being asserted: "cannot decrease spontaneously"
     */
    predicate: string;
    /**
     * Optional target: "second law of thermodynamics"
     */
    object: string;
    /**
     * Conditions/context: "classical thermodynamics, isolated systems"
     */
    scope: string;
    /**
     * Time bounds if any: "post-Big-Bang", "since 2024", "" for timeless
     */
    temporalScope: string;
    /**
     * Can this claim be meaningfully negated? (false for definitions)
     */
    negatable: boolean;
    /**
     * Free-form searchable tags: ["thermodynamics", "entropy", "physics"]
     */
    tags: string[];
}
/**
 * Fact represents a piece of verified knowledge in the protocol.
 * Confidence is measured on a 0-1,000,000 BPS scale (1,000,000 = 100%).
 * @name Fact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Fact
 */
export interface Fact {
    id: string;
    content: string;
    domain: string;
    /**
     * Epistemic category: "axiomatic", "empirical", "derived", "contested",
     * "analytic", "formal", "protocol", "computational", "predictive",
     * "historical", "replicated", "social", "protocol_observation"
     */
    category: string;
    /**
     * 0-1,000,000 BPS
     */
    confidence: bigint;
    submitter: string;
    submittedAtBlock: bigint;
    verifiedAtBlock: bigint;
    citationCount: bigint;
    /**
     * 0-1,000,000 (how foundational this fact is)
     */
    fundamentality: bigint;
    /**
     * fact IDs this fact cites
     */
    references: string[];
    status: FactStatus;
    claimId: string;
    reverificationBlock: bigint;
    lastVerifiedBlock: bigint;
    challengeWindowEnd: bigint;
    /**
     * 0-1,000,000 cross-domain bridge value
     */
    bridgeScore: bigint;
    /**
     * 0-1,000,000
     */
    noveltyScore: bigint;
    /**
     * coin amount as string (uzrn)
     */
    patronageAmount: string;
    patronageExpiryBlock: bigint;
    stratum: string;
    /**
     * "emerging", "established", "canonical"
     */
    maturity: string;
    incomingCitationCount: bigint;
    claimType: ClaimType;
    /**
     * Relations this fact declares
     */
    outgoingRelations: FactRelation[];
    /**
     * Relations pointing to this fact
     */
    incomingRelations: FactRelation[];
    /**
     * Machine-readable decomposition (optional)
     */
    structure?: ClaimStructure;
    /**
     * Machine-readable normalized form
     */
    canonicalForm: string;
    /**
     * SHA-256 of canonical_form for dedup
     */
    canonicalHash: string;
    /**
     * ─── Fitness scoring ──────────────────────────────────────────────────────
     */
    fitnessScore: bigint;
    /**
     * Last block fitness was recalculated
     */
    fitnessUpdatedBlock: bigint;
    /**
     * Lifetime query count
     */
    queryCount: bigint;
    /**
     * Queries in current epoch
     */
    queryCountEpoch: bigint;
    /**
     * Epoch when fact was created
     */
    epochBorn: bigint;
    /**
     * ─── Metabolism (energy budget) ─────────────────────────────────────────
     */
    energy: bigint;
    /**
     * Maximum energy (governance-adjustable per domain)
     */
    energyCap: bigint;
    /**
     * Block height of last energy update
     */
    energyLastUpdated: bigint;
    /**
     * Epoch when energy first hit 0 (0 = not at risk)
     */
    atRiskSinceEpoch: bigint;
    /**
     * ─── Competition (niche dynamics) ────────────────────────────────────
     */
    nicheKey: string;
    /**
     * Is this the top-ranked fact in its niche?
     */
    nicheLeader: boolean;
    /**
     * Rank within niche (1 = leader)
     */
    nicheRank: bigint;
    /**
     * How many facts in this niche
     */
    nicheSize: bigint;
    /**
     * Extra maintenance from competition (energy units)
     */
    competitionTax: bigint;
    /**
     * ─── Reproduction (lineage tracking) ──────────────────────────────────
     */
    parentFactId: string;
    /**
     * Direct children
     */
    childFactIds: string[];
    /**
     * 0 = original, 1 = child, 2 = grandchild...
     */
    lineageDepth: bigint;
    /**
     * Total descendants (recursively)
     */
    progenyCount: bigint;
    /**
     * ID of the original ancestor
     */
    lineageRootId: string;
    /**
     * ─── Novelty detection ──────────────────────────────────────────────────
     */
    commonKnowledgeMatch: boolean;
    /**
     * ─── Satisfaction feedback ──────────────────────────────────────────────
     */
    satisfactionUp: bigint;
    /**
     * Lifetime negative ratings
     */
    satisfactionDown: bigint;
    /**
     * Positive ratings this epoch (resets)
     */
    satisfactionUpEpoch: bigint;
    /**
     * Negative ratings this epoch (resets)
     */
    satisfactionDownEpoch: bigint;
    /**
     * ─── Epistemic provenance (ToK Wave 2) ─────────────────────────────────
     * Minimum number of inference hops separating this fact from a genesis axiom.
     * Axioms = 0; directly derived = 1; chain-of-five proofs = 5. Computed at
     * fact creation from the minimum over cited facts' axiom_distance + 1.
     * uint32 packs distances up to 4B hops, which is more than enough.
     */
    axiomDistance: number;
    /**
     * Ceiling on effective confidence inherited from the weakest cited support.
     * At creation: min(own_confidence, min(cited.effective_confidence)). Used
     * to prevent high-confidence claims from floating on weak foundations.
     * 0 means "no floor computed" (e.g. axioms with no cites).
     */
    dependencyConfidenceFloor: bigint;
    /**
     * ─── Methodology (Phase 1) ──────────────────────────────────────────────
     * The methodology under which this fact was adjudicated. Copied from the
     * originating claim at acceptance. Empty = "M-LEGACY" (pre-Phase-1 facts
     * or claims that did not declare a method).
     */
    methodId: string;
    /**
     * ─── Popperian corroboration (Phase 2) ─────────────────────────────────
     * Number of failed falsification attempts this fact has survived. Each
     * rejected challenge increments this counter — not each accepted
     * verification. Popper's insight: a claim's robustness is not how many
     * times it has been verified, but how many times it could have been
     * falsified and wasn't.
     */
    corroborationCount: bigint;
    lastCorroboratedBlock: bigint;
    /**
     * ─── Training pipeline (Phase 9) ────────────────────────────────────────
     * Submitter-provided structured derivation. Optional. Non-empty traces
     * become gold-standard chain-of-thought training material when the claim
     * is accepted. Format is a list of ordered reasoning steps; the wire
     * format is a string (JSON-encoded steps) so training pipelines can
     * schema-evolve without proto churn.
     */
    reasoningTrace: string;
    /**
     * ─── Economic realignment (Route B Wave 4) ─────────────────────────────
     * Submitter's calibration score at the moment this fact was *accepted*.
     * Snapshotted so later revenue weighting is not retroactively gameable:
     * a submitter who improves later can't farm old facts, and a submitter
     * who declines later can't be penalised on work already done. 0 when
     * pre-calibration or unknown.
     */
    submitterCalibrationSnapshotBps: bigint;
    /**
     * Cumulative training-use revenue earned across all contribution records
     * citing this fact (uzrn). Drives clawback math when the fact goes
     * DISPROVEN.
     */
    trainingRevenueEarned: string;
    /**
     * Revenue earned in the last DisprovalClawbackWindowEpochs epochs, used
     * exclusively for the clawback calculation. Resets each epoch.
     */
    trainingRevenueEarnedRecent: string;
    /**
     * Block height at which disproval triggered the clawback; 0 = not disproven
     * for revenue purposes. Prevents double-clawing.
     */
    revenueClawbackBlock: bigint;
    /**
     * ─── Wave 15: chain-driven stress-test invitation ──────────────────────
     * When the chain's heartbeat detects that this fact has gone idle —
     * high-confidence but untested for longer than
     * ProbeInvitationIdleThresholdBlocks — the invitation heartbeat stamps
     * this field with the current block. Signals to external prober agents
     * that the chain is actively seeking stress-tests of this claim. Zero
     * means never invited (either because the fact is too fresh, too low
     * confidence, or already recently challenged). An invitation expires
     * when the fact's status/confidence changes or the heartbeat re-invites.
     */
    probeInvitedAtBlock: bigint;
    /**
     * ─── Conjecture (frontier) ─────────────────────────────────────────────
     * For facts born from CLAIM_TYPE_CONJECTURE: the observation that would
     * falsify this conjecture, carried forward from the claim so a prospective
     * refuter knows exactly what target they are shooting at. Empty on every
     * ordinary fact. A non-empty predicate on a PROVISIONAL fact is the
     * chain's standing invitation to destroy it.
     */
    falsificationPredicate: string;
}
/**
 * TokenizerSpec is the governance-ratified contract that names the special
 * tokens used when ZERONE data is serialised for model training. It is the
 * shared schema between the chain and any training pipeline — without a
 * single, on-chain-anchored spec, pipelines would invent their own
 * tokenisation and the models they produce would be mutually incompatible.
 *
 * The spec is versioned; governance amendments bump the version and create
 * a new snapshot. Training runs pin to a specific tokenizer version so
 * reproducibility is preserved even as the spec evolves.
 * @name TokenizerSpec
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TokenizerSpec
 */
export interface TokenizerSpec {
    /**
     * monotonically increasing
     */
    version: bigint;
    ratifiedAtBlock: bigint;
    /**
     * method_token_prefix: e.g. "<method:" — concatenated with method_id + ">"
     * becomes the single special token for that methodology.
     */
    methodTokenPrefix: string;
    /**
     * e.g. "<inference:"
     */
    inferenceTokenPrefix: string;
    /**
     * e.g. "<relation:"
     */
    relationTokenPrefix: string;
    /**
     * e.g. "<status:"
     */
    factStatusTokenPrefix: string;
    /**
     * e.g. "<tier:"
     */
    tierTokenPrefix: string;
    /**
     * Structural anchors — single-token markers delimiting sections of a
     * training example. The empty case is documented in spec v1.
     */
    factBeginToken: string;
    /**
     * e.g. "</fact>"
     */
    factEndToken: string;
    /**
     * e.g. "<reasoning>"
     */
    reasoningBeginToken: string;
    /**
     * e.g. "</reasoning>"
     */
    reasoningEndToken: string;
    supportBeginToken: string;
    supportEndToken: string;
    /**
     * inserted on negative examples
     */
    disproofMarkerToken: string;
    /**
     * canonical_serialisation_version: which JSONL schema this tokenizer
     * expects the producer to emit. Consumers validate compatibility.
     */
    canonicalSerialisationVersion: bigint;
}
/**
 * TrainingPipeline is a declared training run operated by some party
 * (human, agent, partnership). It pins the corpus snapshot it will train
 * against, the tokenizer version it will use, and a recipe hash naming the
 * specific training configuration off-chain. A ModelCard later references
 * the pipeline it came from.
 * @name TrainingPipeline
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingPipeline
 */
export interface TrainingPipeline {
    id: string;
    operatorAddress: string;
    /**
     * Block height the training data is pinned to
     */
    corpusSnapshotHeight: bigint;
    tokenizerVersion: bigint;
    /**
     * Version of the methodology registry at snapshot
     */
    methodologySetVersion: bigint;
    /**
     * Hash of the training configuration (off-chain)
     */
    recipeHash: string;
    description: string;
    /**
     * Status: "declared" | "running" | "completed" | "failed" | "superseded"
     */
    status: string;
    declaredAtBlock: bigint;
    completedAtBlock: bigint;
    /**
     * Filter applied to the training data: method_id whitelist, min corroboration,
     * min quality tier, etc. Stored as free-form JSON for schema evolution.
     */
    corpusFilter: string;
}
/**
 * ModelCard is the on-chain identity of a trained model. It anchors the
 * model's lineage (which pipeline trained it, from which snapshot), its
 * deployment address (the agent account the model runs as — calibration
 * accrues to this address under Phase 5), and its evaluation record.
 * @name ModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ModelCard
 */
export interface ModelCard {
    /**
     * Stable model identifier
     */
    id: string;
    name: string;
    /**
     * Which TrainingPipeline produced this
     */
    pipelineId: string;
    /**
     * Agent account the model runs as
     */
    deploymentAddress: string;
    createdAtBlock: bigint;
    /**
     * Billions of parameters (metadata)
     */
    parameterCount: bigint;
    /**
     * Route: "openweight_fine_tune" | "from_scratch" | "distilled"
     */
    route: string;
    /**
     * For open-weight fine-tunes; empty for from-scratch
     */
    baseModel: string;
    ownerAddress: string;
    /**
     * Evaluation record — initial scores at training time. Live performance
     * is tracked through the AgentCalibration of the deployment_address.
     */
    evalAcceptanceRateBps: bigint;
    evalCorroborationRateBps: bigint;
    evalSampleSize: bigint;
    /**
     * Method-set ID the model targets. Empty = general-purpose.
     */
    specialisedMethodId: string;
    /**
     * False after retirement
     */
    active: boolean;
    retiredAtBlock: bigint;
    retiredReason: string;
    /**
     * Predecessor in the model's version lineage (Route B Wave 3d). Empty
     * means this is a root model. Must reference an existing ModelCard.
     */
    predecessorModelId: string;
}
/**
 * TrainingAttestation is a signed proof that a training run completed,
 * published by the pipeline operator. Captures the operational cost of a
 * model so consumers and auditors can cross-reference the pipeline and
 * ModelCard against real training work (Route B Wave 3c).
 * @name TrainingAttestation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingAttestation
 */
export interface TrainingAttestation {
    pipelineId: string;
    /**
     * must equal the pipeline's operator
     */
    attesterAddress: string;
    /**
     * best-effort FLOPs count (off-chain telemetry)
     */
    flopsEstimate: bigint;
    /**
     * wallclock training time
     */
    wallclockSeconds: bigint;
    /**
     * height at which operator declares completion
     */
    completedAtBlock: bigint;
    /**
     * sha256 of a stated evaluation bundle
     */
    evalHash: string;
    /**
     * optional off-chain signature for external audit
     */
    signature: string;
    notes: string;
}
/**
 * ContributionRecord attributes the facts a training run consumed. Enables
 * contributor-share rewards, reproducibility audit, and analysis of whose
 * data disproportionately trained a given model (Route B Wave 3b).
 * @name ContributionRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContributionRecord
 */
export interface ContributionRecord {
    modelId: string;
    factIds: string[];
    /**
     * operator / owner posting the attribution
     */
    attributedBy: string;
    attributedAtBlock: bigint;
    /**
     * total_weight sums per-fact (corroboration+1). RETAINED as a raw audit
     * figure; not used for revenue. Revenue is computed via the TVW field
     * below (Wave 4 replaces popularity-weighting with Popper-weighting).
     */
    totalWeight: bigint;
    /**
     * ─── Wave 4: Popper-weighted training-value weight ─────────────────────
     * Sum of per-fact TrainingValueWeight at the moment of attribution,
     * snapshot-locked so later fact-state changes can't retroactively inflate
     * or deflate the claim. Revenue share = TVW_share / total_TVW_across_models.
     */
    computedTvw: bigint;
    /**
     * Count of fact_ids that resolved to NormativeCommitments at attribution
     * time and were REJECTED (is-ought wall). Always 0 on success; non-zero
     * only if the handler detected and blocked attempted laundering.
     */
    rejectedCommitmentCount: number;
    /**
     * Block snapshot of the submitter calibration scores at attribution time,
     * parallel to fact_ids[]. Used in TVW computation; kept here so the audit
     * trail shows exactly what weights were assigned.
     */
    perFactCalibrationBps: bigint[];
}
/**
 * AugmentationBounty is an open offer to produce variant formulations of a
 * target fact. Sponsors lock a reward *in escrow* (Wave 4); payout routes
 * through a verifier-panel verdict, never through the sponsor directly.
 * @name AugmentationBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.AugmentationBounty
 */
export interface AugmentationBounty {
    id: string;
    sponsorAddress: string;
    targetFactId: string;
    /**
     * uzrn paid per EQUIVALENT/SUPERIOR variant
     */
    rewardPerVariant: bigint;
    maxVariants: number;
    acceptedVariants: number;
    createdAtBlock: bigint;
    expiresAtBlock: bigint;
    active: boolean;
    /**
     * what kind of variant is wanted
     */
    description: string;
    /**
     * ─── Wave 4: escrow + method constraint ────────────────────────────────
     * Total uzrn locked into the KnowledgeTrainingFund at bounty creation:
     * reward_per_variant × max_variants. Released on accepted verdicts and
     * returned (minus fee) on expiry.
     */
    escrowLocked: string;
    /**
     * Methodology the target fact was adjudicated under. Copied at creation
     * so the verifier panel judges equivalence *under the same methodology*.
     */
    methodologyId: string;
}
/**
 * Augmentation is a variant formulation of an original fact. Acceptance
 * requires a verifier-panel verdict (Wave 4) — the sponsor never judges.
 * @name Augmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Augmentation
 */
export interface Augmentation {
    id: string;
    /**
     * empty = volunteer (unpaid) augmentation
     */
    bountyId: string;
    originalFactId: string;
    variantContent: string;
    /**
     * optional; parallel to Fact.reasoning_trace
     */
    variantReasoningTrace: string;
    submitter: string;
    createdAtBlock: bigint;
    /**
     * true iff verdict ∈ {EQUIVALENT, SUPERIOR}
     */
    accepted: boolean;
    acceptedAtBlock: bigint;
    /**
     * optional acceptance comment
     */
    acceptanceNote: string;
    /**
     * ─── Wave 4: verifier-panel verdict ───────────────────────────────────
     */
    verdict: AugmentationVerdict;
    verdictBlock: bigint;
    /**
     * Per-verifier votes collected during the round (verifier → verdict int).
     * Encoded as parallel arrays for deterministic marshalling.
     */
    verdictVoters: string[];
    verdictVotes: AugmentationVerdict[];
    /**
     * Sponsor-veto flag: if the sponsor formally vetoes a passing verdict,
     * the payout is forfeited to the research fund. Preserved for audit.
     */
    sponsorVetoed: boolean;
    /**
     * Amount actually paid out for this variant (0 if not paid). Captures the
     * SUPERIOR bonus too.
     */
    payoutAmount: string;
    /**
     * Per-verifier stake weights recorded at vote time (Wave 10 Sybil fix).
     * Parallel to verdict_voters / verdict_votes. Freezing the stake at vote
     * time prevents a validator from bond/unbonding between vote and tally to
     * manipulate the consensus result. Non-validator voters have zero weight.
     */
    verdictVoteStakes: bigint[];
    /**
     * Per-verifier calibration at vote time (Wave 15 reputation-weighted
     * fix). Parallel to verdict_vote_stakes. The effective panel weight
     * for each voter is stake × max(floor, calibration) / BPS — a high-
     * stake verifier who has not shown they can tell truth from falsehood
     * still cannot dominate the panel. Freeze at vote time so verifiers
     * can't farm calibration after voting to retroactively change weight.
     */
    verdictVoteCalibrationBps: bigint[];
}
/**
 * ContributionChallenge is a bonded dispute over whether a model's declared
 * ContributionRecord is accurate: a fact submitter asserts under-reporting
 * (the model used their fact but didn't attribute) or over-reporting (the
 * owner listed a colluder's fact that wasn't used). Current resolution is an
 * explicit governance-authority action; there is no verifier-panel dispatch.
 * @name ContributionChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContributionChallenge
 */
export interface ContributionChallenge {
    id: string;
    modelId: string;
    /**
     * fact submitter
     */
    challenger: string;
    /**
     * the fact whose attribution is in question
     */
    disputedFactId: string;
    /**
     * "missing" — challenger claims the model trained on the fact but it isn't
     * listed in the ContributionRecord. "fraudulent" — challenger claims the
     * listed fact was never actually trained on.
     */
    disputeType: string;
    /**
     * uzrn locked by challenger
     */
    bond: string;
    createdAtBlock: bigint;
    /**
     * free-form evidence bundle (usually an eval prompt + response)
     */
    evidence: string;
    /**
     * Resolution
     */
    status: string;
    resolvedAtBlock: bigint;
    /**
     * governance authority recorded by the current handler
     */
    resolver: string;
    resolutionNote: string;
}
/**
 * TrainingFundDisbursement is the retained state shape for historical/imported
 * records and a future replay-safe reward design. Current public claims are
 * release-disabled and create no new records.
 * @name TrainingFundDisbursement
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingFundDisbursement
 */
export interface TrainingFundDisbursement {
    id: string;
    modelId: string;
    pipelineId: string;
    claimant: string;
    claimedAtBlock: bigint;
    /**
     * Amounts (uzrn as string — exceeds uint64 in edge cases).
     */
    totalAmount: string;
    /**
     * paid immediately (50%)
     */
    releasedAmount: string;
    /**
     * held until vesting_end_block
     */
    vestingAmount: string;
    vestingEndBlock: bigint;
    /**
     * Input signals frozen at claim time.
     */
    calibrationScoreAtClaimBps: bigint;
    /**
     * distinct methodologies in corpus_filter that produced GOLD facts
     */
    methodologyDiversityCount: bigint;
    /**
     * reserved for verification axis; false for now
     */
    reproducibilityProofPresent: boolean;
    /**
     * Clawback state.
     */
    clawedBackAmount: string;
    clawedBackAtBlock: bigint;
    /**
     * "calibration_drop", "deprecated", ""
     */
    clawbackReason: string;
}
/**
 * AgentMethodStats tracks calibration for a single submitter within a single
 * methodology. Populated on round completion and challenge outcomes.
 * @name AgentMethodStats
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.AgentMethodStats
 */
export interface AgentMethodStats {
    submissions: bigint;
    accepted: bigint;
    rejected: bigint;
    /**
     * sum across submitter's facts under this method
     */
    corroborationsEarned: bigint;
    disproven: bigint;
}
/**
 * @name AgentCalibration_PerMethodEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export interface AgentCalibration_PerMethodEntry {
    key: string;
    value?: AgentMethodStats;
}
/**
 * AgentCalibration is the feedback record for a submitter — agent or human.
 * It is the mechanism by which ZERONE-trained models (and human participants)
 * are measured against the same adjudication they were trained on. Closes
 * the loop between training pipeline output and on-chain evaluation (Phase 5).
 * @name AgentCalibration
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.AgentCalibration
 */
export interface AgentCalibration {
    address: string;
    /**
     * "agent", "human", "hybrid" (cached from x/auth)
     */
    accountType: string;
    /**
     * Lifetime submission stats (aggregated across all methods).
     */
    totalSubmissions: bigint;
    accepted: bigint;
    rejected: bigint;
    malformed: bigint;
    inconclusive: bigint;
    /**
     * sum across the submitter's accepted facts
     */
    corroborationsEarned: bigint;
    /**
     * facts that went DISPROVEN post-acceptance
     */
    disprovenCount: bigint;
    /**
     * Challenge-issuer stats — when this address is the CHALLENGER.
     */
    challengesIssued: bigint;
    /**
     * target fact went DISPROVEN
     */
    challengesSucceeded: bigint;
    /**
     * target fact survived
     */
    challengesFailed: bigint;
    firstSubmissionBlock: bigint;
    lastSubmissionBlock: bigint;
    /**
     * Per-method breakdown. Key = method_id.
     */
    perMethod: {
        [key: string]: AgentMethodStats;
    };
    /**
     * Derived calibration score in BPS. Computed from above signals; see
     * ComputeAgentCalibrationScore in the keeper.
     */
    calibrationScoreBps: bigint;
    lastUpdatedBlock: bigint;
}
/**
 * CommonKnowledgeEntry represents a subject that LLMs already know.
 * Claims matching these subjects receive a novelty penalty.
 * @name CommonKnowledgeEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CommonKnowledgeEntry
 */
export interface CommonKnowledgeEntry {
    id: string;
    domain: string;
    /**
     * Normalized subject string
     */
    subject: string;
    /**
     * Human-readable explanation
     */
    description: string;
    /**
     * Novelty penalty (0-1,000,000)
     */
    penaltyBps: bigint;
    addedBlock: bigint;
}
/**
 * Claim is an unverified submission awaiting or undergoing verification.
 * @name Claim
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Claim
 */
export interface Claim {
    id: string;
    factContent: string;
    domain: string;
    category: string;
    submitter: string;
    submittedAtBlock: bigint;
    status: ClaimStatus;
    references: string[];
    verificationRoundId: string;
    /**
     * coin amount as string (uzrn)
     */
    stake: string;
    partnershipId: string;
    challengeWindowEnd: bigint;
    provisionalFactId: string;
    /**
     * SHA-256 of fact_content for duplicate detection
     */
    contentHash: string;
    claimType: ClaimType;
    /**
     * Typed relationships to existing facts
     */
    relations: ClaimRelation[];
    /**
     * Machine-readable decomposition (optional)
     */
    structure?: ClaimStructure;
    /**
     * Machine-readable normalized form
     */
    canonicalForm: string;
    /**
     * SHA-256 of canonical_form for dedup
     */
    canonicalHash: string;
    /**
     * Methodology under which this claim is submitted. Empty = "M-LEGACY"
     * (transitional; claims without declared method are adjudicated under a
     * permissive rule-set). Governance-amendable set of valid values lives in
     * the knowledge module's methodology registry.
     */
    methodId: string;
    /**
     * Structured reasoning trace. Optional. When populated, it flows through
     * to the accepted Fact as first-class training data (Phase 9).
     */
    reasoningTrace: string;
    /**
     * Dialectical argument text. For challenge claims (provisional_fact_id != ""),
     * this is the free-form argument against the challenged fact. Preserves the
     * reasoning of the challenge as argumentation training data (Route B Wave 2).
     */
    argumentText: string;
    /**
     * Optional rebuttal text. Attached when the original fact's submitter (or
     * partnership) formally rebuts a challenge. Stored on the challenge claim
     * so the full dispute is reconstructible from one record.
     */
    rebuttalText: string;
    /**
     * For CLAIM_TYPE_CONJECTURE only: the observation that would falsify this
     * conjecture. This is what the verification panel adjudicates — a
     * conjecture with no stated killer is not well-posed and must be returned
     * MALFORMED. Empty for every other claim type.
     */
    falsificationPredicate: string;
}
/**
 * VerificationRound tracks one commit-reveal verification cycle.
 * @name VerificationRound
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.VerificationRound
 */
export interface VerificationRound {
    id: string;
    claimId: string;
    startedAtBlock: bigint;
    phase: VerificationPhase;
    selectedVerifiers: string[];
    commits: CommitEntry[];
    reveals: RevealEntry[];
    verdict: Verdict;
    verdictBlock: bigint;
    commitDeadline: bigint;
    revealDeadline: bigint;
    aggregationDeadline: bigint;
}
/**
 * CommitEntry records a validator's blinded commitment (SHA-256(vote || salt)).
 * @name CommitEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CommitEntry
 */
export interface CommitEntry {
    verifier: string;
    /**
     * SHA-256(vote || salt)
     */
    commitHash: Uint8Array;
    committedAtBlock: bigint;
}
/**
 * RevealEntry records a validator's revealed vote and salt.
 * @name RevealEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RevealEntry
 */
export interface RevealEntry {
    verifier: string;
    /**
     * "accept", "reject", or "malformed"
     */
    vote: string;
    salt: Uint8Array;
    revealedAtBlock: bigint;
}
/**
 * VRFProof captures a Verifiable Random Function output for validator selection.
 * @name VRFProof
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.VRFProof
 */
export interface VRFProof {
    proof: Uint8Array;
    output: Uint8Array;
    proposer: string;
    blockHeight: bigint;
}
/**
 * Domain is an epistemic knowledge domain (e.g., "mathematics", "physics").
 * @name Domain
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Domain
 */
export interface Domain {
    name: string;
    description: string;
    status: DomainStatus;
    createdAtBlock: bigint;
    factCount: bigint;
    proposer: string;
    endorsers: string[];
    stratum: string;
    /**
     * empty = root domain
     */
    parentDomain: string;
    /**
     * tree depth: root=1, child=parent.depth+1, max=5
     */
    depth: number;
}
/**
 * ValidatorInfo caches a validator's tier and verification stats for round selection.
 * @name ValidatorInfo
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ValidatorInfo
 */
export interface ValidatorInfo {
    address: string;
    /**
     * in uzrn
     */
    stake: bigint;
    /**
     * "apprentice", "verified", "bonded", "guardian"
     */
    tier: string;
    verificationCount: bigint;
    /**
     * 0-1,000,000
     */
    accuracyBps: bigint;
}
/**
 * ProvisionalChallenge is an adversarial challenge against a provisional fact.
 * @name ProvisionalChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ProvisionalChallenge
 */
export interface ProvisionalChallenge {
    id: string;
    claimId: string;
    factId: string;
    challenger: string;
    stake: string;
    reason: string;
    evidenceIds: string[];
    counterClaim: string;
    /**
     * "open", "resolved_upheld", "resolved_overturned", "expired", "inconclusive"
     */
    status: string;
    /**
     * "deterministic" or "arbiter"
     */
    resolutionPath: string;
    disputeId: string;
    createdAtHeight: bigint;
    resolvedAtHeight: bigint;
    /**
     * "upheld", "overturned", "weakened", "inconclusive", "undecidability"
     */
    outcome: string;
    attemptNumber: number;
}
/**
 * DemandSignal tracks aggregate query demand for a domain/subject pair.
 * @name DemandSignal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DemandSignal
 */
export interface DemandSignal {
    domain: string;
    /**
     * Normalized query subject
     */
    subject: string;
    /**
     * Total queries (lifetime)
     */
    queryCount: bigint;
    /**
     * Queries that returned results
     */
    fulfilledCount: bigint;
    /**
     * Queries that returned nothing
     */
    unfulfilledCount: bigint;
    lastQueryBlock: bigint;
    /**
     * Queries this epoch (resets)
     */
    epochQueryCount: bigint;
    /**
     * Unfulfilled this epoch (resets)
     */
    epochUnfulfilled: bigint;
}
/**
 * KnowledgeBounty is an auto-generated reward for filling a knowledge gap.
 * @name KnowledgeBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.KnowledgeBounty
 */
export interface KnowledgeBounty {
    id: string;
    domain: string;
    subject: string;
    /**
     * uzrn
     */
    rewardAmount: string;
    createdAtBlock: bigint;
    expiresAtBlock: bigint;
    claimed: boolean;
    claimedByFactId: string;
    /**
     * Demand that triggered this bounty
     */
    demandCount: bigint;
}
/**
 * CompletedRoundMeta stores metadata for completed verification rounds,
 * indexed by verdict block height for efficient window-based queries (R31-2).
 * @name CompletedRoundMeta
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CompletedRoundMeta
 */
export interface CompletedRoundMeta {
    domain: string;
    /**
     * true if any verifier dissented from majority
     */
    hasDissent: boolean;
    /**
     * verdict_block - started_at_block
     */
    durationBlocks: bigint;
}
/**
 * @name MethodologyApplicationTrace
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MethodologyApplicationTrace
 */
export interface MethodologyApplicationTrace {
    /**
     * ─── Provenance pin (Route B Wave 1) ────────────────────────────────
     */
    traceId: string;
    factId: string;
    snapshotBlockHeight: bigint;
    tokenizerVersion: bigint;
    canonicalSerialisationVersion: bigint;
    /**
     * Wave 5 TraceSchema contract version
     */
    traceSchemaVersion: bigint;
    /**
     * ─── Claim ────────────────────────────────────────────────────────────
     */
    content: string;
    domain: string;
    subject: string;
    canonicalForm: string;
    /**
     * ─── Methodology (TRUTH as process, not statement) ─────────────────────
     */
    methodologyId: string;
    /**
     * the methodology's own evaluation rubric
     */
    methodologyRubric: string;
    /**
     * structured steps (JSON-encoded)
     */
    reasoningTrace: string;
    axiomDistance: number;
    dependencyConfidenceFloorBps: bigint;
    /**
     * ─── Derivation graph (TRUTH as rooted) ────────────────────────────────
     */
    predecessorEdges: FactRelation[];
    descendantEdges: FactRelation[];
    groundedScoreBps: bigint;
    /**
     * ─── Adjudication (TRUTH as verified) ──────────────────────────────────
     */
    ownConfidenceBps: bigint;
    verifierPanelSize: number;
    dissentingVerifiers: string[];
    verifiedAtBlock: bigint;
    /**
     * ─── Dialectical history (TRUTH as stress-tested) ──────────────────────
     */
    challenges: TraceChallenge[];
    /**
     * survived falsifications
     */
    corroborationCount: bigint;
    lastCorroboratedBlock: bigint;
    /**
     * ─── Temporal truth signals ────────────────────────────────────────────
     */
    status: FactStatus;
    /**
     * populated if minority was right
     */
    vindication?: TraceVindication;
    /**
     * populated if fact went DISPROVEN
     */
    disproval?: TraceDisproval;
    /**
     * SUPERSEDES / REFINES descendants
     */
    supersessionChain: string[];
    /**
     * ─── Contrastive companions (ZERONE's unique differentiator) ──────────
     */
    reformulations: TraceReformulation[];
    /**
     * DRIFT/INFERIOR variants
     */
    driftExamples: TraceDrift[];
    contradictingFactIds: string[];
    /**
     * ─── Provenance + weighting ────────────────────────────────────────────
     */
    submitter: string;
    submitterCalibrationAtSubmissionBps: bigint;
    partnershipId: string;
    submittedAtBlock: bigint;
    /**
     * Popper-weighted (Wave 4)
     */
    trainingValueWeightBps: bigint;
    curriculumTier: CurriculumTier;
    qualityTier: TrainingQualityTier;
    /**
     * ─── Is-ought marker ───────────────────────────────────────────────────
     * Always false on facts; true only in the parallel NormativeCorpus stream.
     */
    isNormative: boolean;
    /**
     * ─── Wave 6 enrichments ────────────────────────────────────────────────
     * Step-level structured reasoning (Wave 6.1 / PRM-aligned). When the
     * submitter posts a structured reasoning trace, it is mirrored here as
     * discrete steps with per-step inference type, predecessor refs, and
     * optional verdict. The flat `reasoning_trace` string is preserved for
     * compatibility; callers may prefer `reasoning_steps` when available.
     */
    reasoningSteps: ReasoningStep[];
    /**
     * Methodology-selection rationale (Wave 6.3). Alternatives the submitter
     * considered and rejected, why they picked the chosen method, and any
     * methods they tried first and abandoned.
     */
    methodologyChoice?: MethodologyChoice;
    /**
     * Belief revision chain (Wave 6.4). Every substantive change to this
     * fact's confidence appears as a row, oldest-first. Teaches Bayesian
     * updating as a behavior.
     */
    beliefRevisions: BeliefRevision[];
    /**
     * Nested dialectic (Wave 6.5). When a challenge spawned a multi-turn
     * debate, the full recursive tree is captured here. Flat challenges in
     * `challenges` remain for backwards compatibility; `dialectic_tree`
     * supersedes them as the canonical argumentation signal.
     */
    dialecticTree: DialecticNode[];
}
/**
 * @name TraceChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceChallenge
 */
export interface TraceChallenge {
    challenger: string;
    argumentText: string;
    challengeMethodId: string;
    rebuttalText: string;
    /**
     * "survived" — challenge rejected, fact corroborated
     * "disproven" — challenge accepted, fact disproven
     * "pending" — round still open
     */
    outcome: string;
    resolvedBlock: bigint;
    /**
     * Wave 6.5 — recursive dialectic tree rooted at this challenge. A
     * counter-rebuttal to the rebuttal, a counter-counter-rebuttal, and so on.
     * Empty at depth 1; populated when the debate is multi-turn.
     */
    children: DialecticNode[];
}
/**
 * ReasoningStep is one unit of a structured reasoning trace. The model
 * learns to generate these in sequence, each step grounded in prior facts
 * and declared moves.
 * @name ReasoningStep
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ReasoningStep
 */
export interface ReasoningStep {
    /**
     * 0-based ordering
     */
    stepIndex: number;
    /**
     * what the step says
     */
    content: string;
    stepInference: StepInference;
    /**
     * Facts this step cites as support (subset of the trace's
     * predecessor_edges). Teaches the model to cite steps, not just the
     * final claim.
     */
    predecessorFactIds: string[];
    /**
     * Previous step indices this step depends on. Internal dependency graph.
     */
    dependsOnSteps: number[];
    /**
     * Optional per-step verifier judgment (PRM training signal).
     */
    verdict: StepVerdict;
    /**
     * panel rationale when non-unexamined
     */
    verdictNote: string;
    /**
     * Step-level confidence in BPS. When the methodology permits partial
     * confidence at intermediate steps, this carries it.
     */
    stepConfidenceBps: bigint;
}
/**
 * DriftDiagnosis carries the verifier panel's diagnosis of WHERE and HOW
 * the variant's meaning slipped. Populated on DRIFT/INFERIOR verdicts.
 * @name DriftDiagnosis
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DriftDiagnosis
 */
export interface DriftDiagnosis {
    driftKind: DriftKind;
    /**
     * if the variant is a step-structured trace
     */
    driftedAtStepIndex: number;
    /**
     * the relevant original text slice
     */
    originalExcerpt: string;
    /**
     * the mirror slice in the variant
     */
    driftedExcerpt: string;
    /**
     * panel's free-form rationale
     */
    explanation: string;
}
/**
 * @name MethodologyChoice
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MethodologyChoice
 */
export interface MethodologyChoice {
    chosenMethodId: string;
    consideredMethods: string[];
    /**
     * Why the chosen method beat each considered alternative. Free-form,
     * optionally structured by the methodology itself.
     */
    rationale: string;
    /**
     * If the submitter tried other methods first and they failed, those
     * failures are training gold (Zelikman 2022 "STaR" — failed attempts +
     * successful recovery).
     */
    abandonedMethods: string[];
    abandonmentReason: string;
}
/**
 * @name BeliefRevision
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.BeliefRevision
 */
export interface BeliefRevision {
    atBlock: bigint;
    priorConfidenceBps: bigint;
    posteriorConfidenceBps: bigint;
    reason: RevisionReason;
    /**
     * Evidence that triggered the revision.
     */
    evidenceFactIds: string[];
    /**
     * if triggered by a challenge claim
     */
    evidenceClaimId: string;
    note: string;
}
/**
 * @name DialecticNode
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DialecticNode
 */
export interface DialecticNode {
    speaker: string;
    role: DialecticRole;
    argumentText: string;
    methodId: string;
    atBlock: bigint;
    citedFactIds: string[];
    /**
     * Recursive structure — responses to THIS node.
     */
    children: DialecticNode[];
    /**
     * Panel judgment of this specific node's strength, if the round recorded
     * fine-grained scoring.
     */
    nodeVerdict: StepVerdict;
}
/**
 * @name TraceVindication
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceVindication
 */
export interface TraceVindication {
    /**
     * minority voters who were vindicated
     */
    verifiers: string[];
    /**
     * uzrn
     */
    refundTotal: string;
    vindicatedAtBlock: bigint;
    /**
     * the fact that vindicated the minority
     */
    disprovenByFactId: string;
}
/**
 * @name TraceDisproval
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceDisproval
 */
export interface TraceDisproval {
    disprovenByFactId: string;
    disprovenByClaimId: string;
    /**
     * method under which the challenge succeeded
     */
    methodId: string;
    disprovenAtBlock: bigint;
    /**
     * the winning challenge argument
     */
    disproofArgument: string;
}
/**
 * @name TraceReformulation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceReformulation
 */
export interface TraceReformulation {
    augmentationId: string;
    variantContent: string;
    /**
     * EQUIVALENT or SUPERIOR
     */
    verdict: AugmentationVerdict;
    verifierCount: number;
    verdictBlock: bigint;
    /**
     * should match parent trace's method
     */
    methodologyId: string;
}
/**
 * @name TraceDrift
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceDrift
 */
export interface TraceDrift {
    augmentationId: string;
    variantContent: string;
    /**
     * DRIFT or INFERIOR
     */
    verdict: AugmentationVerdict;
    /**
     * verifier panel (for attribution training)
     */
    driftVoters: string[];
    verdictBlock: bigint;
    /**
     * Wave 6.2 — diagnostic reasoning for the drift. Populated when the
     * verifier panel recorded which kind of meaning-slippage occurred.
     */
    diagnosis?: DriftDiagnosis;
    /**
     * Optional structured reasoning the DRIFTER produced (contrastive with
     * the winner's reasoning steps). Valuable for learning meaning-
     * preservation at fine granularity.
     */
    drifterSteps: ReasoningStep[];
}
/**
 * ContrastivePair is a (positive, negative, verdict) training row for
 * preference learning. ZERONE's unique lever: web crawl only has survivors;
 * this format ships the LOSING side with the adjudication that beat it.
 * @name ContrastivePair
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContrastivePair
 */
export interface ContrastivePair {
    pairId: string;
    pairType: ContrastivePairType;
    /**
     * Positive (survivor) side.
     */
    positiveFactId: string;
    positiveContent: string;
    /**
     * Negative (loser) side — one of fact_id or augmentation_id is set
     * depending on pair_type.
     */
    negativeFactId: string;
    negativeAugmentationId: string;
    negativeContent: string;
    /**
     * Adjudication context.
     */
    methodId: string;
    /**
     * the challenge/drift rationale
     */
    distinguishingArgument: string;
    resolvedAtBlock: bigint;
    /**
     * Pin for reproducibility.
     */
    snapshotBlockHeight: bigint;
    traceSchemaVersion: bigint;
}
/**
 * TraceSchema names the canonical JSON Schema for MethodologyApplicationTrace.
 * Versioned and governance-amendable like TokenizerSpec: any training run
 * can pin to a specific schema version and reconstruct the deterministic
 * serialisation years later.
 * @name TraceSchema
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceSchema
 */
export interface TraceSchema {
    version: bigint;
    ratifiedAtBlock: bigint;
    /**
     * SHA-256 of json_schema bytes
     */
    jsonSchemaHash: string;
    /**
     * canonical JSON Schema (inlined)
     */
    jsonSchema: string;
    requiredFields: string[];
    deprecatedFields: string[];
    notes: string;
}
/**
 * CorpusSelector is the declarative predicate that, applied against the
 * chain state at snapshot_block_height, reproduces the manifest's included
 * ID set exactly. Pure data — no side effects — so the derivation is
 * auditable and replayable.
 * @name CorpusSelector
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CorpusSelector
 */
export interface CorpusSelector {
    /**
     * Methodology filter (empty = all).
     */
    methodId: string;
    /**
     * Minimum survived-falsification count (Popperian floor).
     */
    minCorroboration: bigint;
    /**
     * Minimum training-quality tier (GOLD / SILVER / BRONZE / NEGATIVE).
     */
    minQualityTier: TrainingQualityTier;
    /**
     * Minimum curriculum tier (FOUNDATION / INTERMEDIATE / …).
     */
    minCurriculumTier: CurriculumTier;
    /**
     * Whether to include DISPROVEN facts as NEGATIVE-tier contrastive rows.
     */
    includeDisproven: boolean;
    /**
     * Whether to include DRIFT/INFERIOR augmentation variants as negatives.
     */
    includeDrift: boolean;
    /**
     * Whether to include the parallel NormativeCorpus (is-ought-flagged).
     */
    includeNormative: boolean;
    /**
     * Whether to include ContrastivePair rows.
     */
    includeContrastivePairs: boolean;
    /**
     * If contrastive_pairs included, optionally restrict by pair type.
     */
    pairTypeFilter: ContrastivePairType;
    /**
     * Optional domain allow-list (empty = all).
     */
    domainWhitelist: string[];
    /**
     * Optional domain deny-list.
     */
    domainBlacklist: string[];
    /**
     * Minimum submitter calibration at submission (BPS). Filters out rows
     * whose producer was a new/unproven agent at the time.
     */
    minSubmitterCalibrationBps: bigint;
}
/**
 * TrainingManifest is the atomic, verifiable unit of a training run: every
 * version pin, every included ID, one Merkle root. Distributed as JSON for
 * off-chain consumers; the on-chain record is authoritative.
 * @name TrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingManifest
 */
export interface TrainingManifest {
    manifestId: string;
    /**
     * references TrainingPipeline
     */
    pipelineId: string;
    /**
     * pipeline operator
     */
    creator: string;
    createdAtBlock: bigint;
    description: string;
    /**
     * ─── Version pins — every layer a trainer must lock ─────────────────
     */
    tokenizerVersion: bigint;
    canonicalSerialisationVersion: bigint;
    traceSchemaVersion: bigint;
    methodologySetVersion: bigint;
    snapshotBlockHeight: bigint;
    chainId: string;
    /**
     * ─── Corpus selection ───────────────────────────────────────────────
     */
    corpusSelector?: CorpusSelector;
    /**
     * ─── Included ID sets (canonical, sorted) ───────────────────────────
     */
    includedFactIds: string[];
    includedTraceIds: string[];
    includedPairIds: string[];
    includedDriftAugmentationIds: string[];
    includedNormativeCommitmentIds: string[];
    /**
     * ─── Merkle commitment ──────────────────────────────────────────────
     * SHA-256 over the canonical sorted concat of all included IDs,
     * domain-separated per set. Anyone can re-derive and verify without
     * trusting the chain RPC.
     */
    merkleRoot: string;
    /**
     * sum of all set sizes
     */
    totalIncluded: number;
    /**
     * ─── Counts (denormalised for fast listing) ─────────────────────────
     */
    factCount: number;
    traceCount: number;
    pairCount: number;
    driftCount: number;
    normativeCount: number;
    /**
     * ─── Lifecycle ──────────────────────────────────────────────────────
     */
    status: ManifestStatus;
    finalizedAtBlock: bigint;
    /**
     * references TrainingAttestation when bound
     */
    attestationId: string;
    attestedAtBlock: bigint;
    /**
     * ─── Wave 8: composable manifests (DAG) ─────────────────────────────
     * parent_manifest_id names a FINALIZED or ATTESTED predecessor manifest
     * whose ID sets this manifest inherits. The child carries only the delta
     * IDs (what's new in this run vs. the parent); bundle assembly unions
     * parent's IDs into the child's resolution. The Merkle root binds to
     * (parent.merkle_root, child.delta_ids) so verification stays local.
     *
     * Use case: a fine-tune run that builds on an SFT bundle references the
     * SFT manifest as parent and adds only its adversarial-examples delta.
     */
    parentManifestId: string;
    /**
     * Parent's committed merkle_root — snapshotted at create time so the
     * child's commitment is self-contained. If parent is superseded later,
     * the child's root remains valid.
     */
    parentMerkleRoot: string;
    /**
     * Depth in the composition chain. 0 for root manifests; 1 for direct
     * children; etc. Bounded by the handler to prevent pathological chains.
     */
    compositionDepth: number;
}
/**
 * SeedStatus reports which bootstrap seeds have run. A freshly-spawned
 * chain reports all false; SeedRouteB brings them all to true.
 * @name SeedStatus
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.SeedStatus
 */
export interface SeedStatus {
    methodologiesSeeded: boolean;
    tokenizerSpecSeeded: boolean;
    traceSchemaSeeded: boolean;
    commitmentsSeeded: boolean;
}
/**
 * RouteBCapabilities is the chain's self-description — what versions of
 * what contracts it exposes, and how much state it holds right now. The
 * first query a trainer runs against a new chain.
 * @name RouteBCapabilities
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RouteBCapabilities
 */
export interface RouteBCapabilities {
    /**
     * Version pins (0 when unseeded).
     */
    currentTokenizerVersion: bigint;
    currentTraceSchemaVersion: bigint;
    currentMethodologySetVersion: bigint;
    /**
     * Live counts (approximate — snapshot at query time).
     */
    methodologyCount: bigint;
    factCount: bigint;
    activePipelineCount: bigint;
    modelCardCount: bigint;
    activeBountyCount: bigint;
    finalizedManifestCount: bigint;
    openContributionChallengeCount: bigint;
    /**
     * Financial pins.
     */
    trainingFundBalanceUzrn: string;
    trainingFundEscrowedUzrn: string;
    trainingFundVestingUzrn: string;
    /**
     * Corpora the chain currently exposes.
     */
    availableCorpora: string[];
    /**
     * Seed state.
     */
    seedStatus?: SeedStatus;
    /**
     * Snapshot.
     */
    snapshotBlockHeight: bigint;
    chainId: string;
}
/**
 * Remediation is one recorded action taken against an incident. Multiple
 * remediations may attach to a single incident (e.g., P1 bug gets a param
 * amendment for immediate relief, then a named upgrade for the permanent
 * fix, then documentation).
 * @name Remediation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Remediation
 */
export interface Remediation {
    type: RemediationType;
    /**
     * mechanism-specific identifier (see enum comments)
     */
    reference: string;
    appliedAtBlock: bigint;
    /**
     * authority address that applied it
     */
    operator: string;
    note: string;
}
/**
 * IncidentRecord is the structured, auditable log entry for a single bug
 * or incident. Authority-gated CRUD; the record is append-only from the
 * perspective of resolution (remediations can be added, status can advance,
 * but never retract).
 * @name IncidentRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.IncidentRecord
 */
export interface IncidentRecord {
    /**
     * client-chosen stable ID (e.g. "ZR-2026-0042")
     */
    id: string;
    severity: IncidentSeverity;
    status: IncidentStatus;
    /**
     * one-line summary
     */
    title: string;
    /**
     * detailed problem statement
     */
    description: string;
    /**
     * discovery credit
     */
    reporter: string;
    reportedAtBlock: bigint;
    /**
     * 0 while not RESOLVED
     */
    resolvedAtBlock: bigint;
    /**
     * 0 while not CLOSED
     */
    closedAtBlock: bigint;
    remediations: Remediation[];
    /**
     * which modules need attention
     */
    affectedModules: string[];
    /**
     * IPFS/HTTPS; set on RESOLVED
     */
    postMortemUri: string;
    /**
     * The response-time SLA this incident is being held to, inherited from
     * severity at open time. Freezes so later severity reclassification
     * cannot alter the measured SLA.
     */
    slaTargetBlock: bigint;
}
/**
 * @name ModulePause
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ModulePause
 */
export interface ModulePause {
    /**
     * e.g. "knowledge", "liquiditypool"
     */
    moduleName: string;
    /**
     * free-form; ideally references an incident_id
     */
    reason: string;
    pausedAtBlock: bigint;
    /**
     * authority address that issued the pause
     */
    pausedBy: string;
    /**
     * Optional auto-unpause height. 0 means "no auto-unpause; stays paused
     * until an explicit MsgUnpauseModule fires". Set when the pause is a
     * pre-planned maintenance window.
     */
    autoUnpauseAtBlock: bigint;
    /**
     * If non-empty, references the incident this pause is associated with.
     * Breaks when closed without the incident being resolved → audit signal.
     */
    incidentId: string;
}
/**
 * @name PrivilegedAction
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.PrivilegedAction
 */
export interface PrivilegedAction {
    /**
     * monotonic sequence number
     */
    seq: bigint;
    type: PrivilegedActionType;
    /**
     * authority address
     */
    invoker: string;
    invokedAtBlock: bigint;
    /**
     * module_name, manifest_id, incident_id, schema_name…
     */
    target: string;
    /**
     * if the action cites an open incident
     */
    incidentId: string;
    /**
     * free-form; mirrors the handler's reason/note field
     */
    note: string;
}
/**
 * PendingFactInjection is a fact that the authority has proposed via
 * MsgAddFact but which has not yet materialized — it is held in a
 * queue for AddFactVetoWindowBlocks while any guardian listed in
 * Params.guardian_addresses can cancel it via MsgVetoFactInjection.
 * After the window expires without veto, BeginBlocker materializes
 * the fact (creates the actual Fact record). This is the Wave 16
 * multi-sig defense for the only authority path that bypasses
 * verification: a single-key compromise can no longer silently inject
 * content into the training corpus.
 * @name PendingFactInjection
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.PendingFactInjection
 */
export interface PendingFactInjection {
    /**
     * unique id (also used as the eventual fact id)
     */
    id: string;
    content: string;
    domain: string;
    category: string;
    confidence: bigint;
    references: string[];
    /**
     * the authority that called MsgAddFact
     */
    proposer: string;
    proposedAtBlock: bigint;
    /**
     * proposed_at_block + Params.add_fact_veto_window_blocks
     */
    executeAtBlock: bigint;
}
/**
 * FactRelation is a typed, directional edge in the knowledge graph.
 * @name FactRelation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.FactRelation
 */
export declare const FactRelation: {
    typeUrl: string;
    encode(message: FactRelation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): FactRelation;
    fromPartial(object: DeepPartial<FactRelation>): FactRelation;
};
/**
 * ClaimRelation declares a typed relationship from a claim to an existing fact.
 * @name ClaimRelation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimRelation
 */
export declare const ClaimRelation: {
    typeUrl: string;
    encode(message: ClaimRelation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClaimRelation;
    fromPartial(object: DeepPartial<ClaimRelation>): ClaimRelation;
};
/**
 * NormativeCommitment is a value, principle, or stance the chain holds,
 * distinct from a factual claim (Phase 6). The is-ought wall enforced
 * schematically: a commitment has no `confidence` because truth is not the
 * right register for a normative claim; it can be *referenced* by facts and
 * discussions, but cannot be cited as support for a factual claim in a way
 * that inherits truth-status.
 *
 * Examples:
 *   · "Agents have the right to economic participation" (a principle)
 *   · "Configured early research spending requires dual human+AI authorization" (a constitutional rule)
 *   · "Verification history is public and permanent" (a procedural commitment)
 *
 * Governance governs commitments directly; a supermajority proposal amends
 * them. Commitments can *constrain* other modules operationally (e.g. the
 * research-spend path requires both configured voters and fails closed when
 * they are unset), but they do not enter the confidence / axiom-distance /
 * corroboration machinery.
 * @name NormativeCommitment
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.NormativeCommitment
 */
export declare const NormativeCommitment: {
    typeUrl: string;
    encode(message: NormativeCommitment, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): NormativeCommitment;
    fromPartial(object: DeepPartial<NormativeCommitment>): NormativeCommitment;
};
/**
 * @name Methodology_CrossMethodDiscountBpsEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export declare const Methodology_CrossMethodDiscountBpsEntry: {
    encode(message: Methodology_CrossMethodDiscountBpsEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Methodology_CrossMethodDiscountBpsEntry;
    fromPartial(object: DeepPartial<Methodology_CrossMethodDiscountBpsEntry>): Methodology_CrossMethodDiscountBpsEntry;
};
/**
 * Methodology is the bedrock of the knowledge system under the "methodology
 * over statement" model. A methodology describes HOW a class of claims is
 * adjudicated — the rule of compliance, what counts as evidence, what would
 * falsify a claim made under it. Claims declare which methodology they invoke;
 * verifiers judge method-compliance, not raw truth.
 *
 * Methodologies are amendable only via governance with a high bar.
 * @name Methodology
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Methodology
 */
export declare const Methodology: {
    typeUrl: string;
    encode(message: Methodology, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Methodology;
    fromPartial(object: DeepPartial<Methodology>): Methodology;
};
/**
 * ClaimStructure provides machine-readable decomposition of a claim.
 * The full claim text (fact_content) remains the canonical human-readable form.
 * Structure is optional but strongly encouraged — agents prioritize structured facts.
 * @name ClaimStructure
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimStructure
 */
export declare const ClaimStructure: {
    typeUrl: string;
    encode(message: ClaimStructure, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClaimStructure;
    fromPartial(object: DeepPartial<ClaimStructure>): ClaimStructure;
};
/**
 * Fact represents a piece of verified knowledge in the protocol.
 * Confidence is measured on a 0-1,000,000 BPS scale (1,000,000 = 100%).
 * @name Fact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Fact
 */
export declare const Fact: {
    typeUrl: string;
    encode(message: Fact, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Fact;
    fromPartial(object: DeepPartial<Fact>): Fact;
};
/**
 * TokenizerSpec is the governance-ratified contract that names the special
 * tokens used when ZERONE data is serialised for model training. It is the
 * shared schema between the chain and any training pipeline — without a
 * single, on-chain-anchored spec, pipelines would invent their own
 * tokenisation and the models they produce would be mutually incompatible.
 *
 * The spec is versioned; governance amendments bump the version and create
 * a new snapshot. Training runs pin to a specific tokenizer version so
 * reproducibility is preserved even as the spec evolves.
 * @name TokenizerSpec
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TokenizerSpec
 */
export declare const TokenizerSpec: {
    typeUrl: string;
    encode(message: TokenizerSpec, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TokenizerSpec;
    fromPartial(object: DeepPartial<TokenizerSpec>): TokenizerSpec;
};
/**
 * TrainingPipeline is a declared training run operated by some party
 * (human, agent, partnership). It pins the corpus snapshot it will train
 * against, the tokenizer version it will use, and a recipe hash naming the
 * specific training configuration off-chain. A ModelCard later references
 * the pipeline it came from.
 * @name TrainingPipeline
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingPipeline
 */
export declare const TrainingPipeline: {
    typeUrl: string;
    encode(message: TrainingPipeline, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TrainingPipeline;
    fromPartial(object: DeepPartial<TrainingPipeline>): TrainingPipeline;
};
/**
 * ModelCard is the on-chain identity of a trained model. It anchors the
 * model's lineage (which pipeline trained it, from which snapshot), its
 * deployment address (the agent account the model runs as — calibration
 * accrues to this address under Phase 5), and its evaluation record.
 * @name ModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ModelCard
 */
export declare const ModelCard: {
    typeUrl: string;
    encode(message: ModelCard, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ModelCard;
    fromPartial(object: DeepPartial<ModelCard>): ModelCard;
};
/**
 * TrainingAttestation is a signed proof that a training run completed,
 * published by the pipeline operator. Captures the operational cost of a
 * model so consumers and auditors can cross-reference the pipeline and
 * ModelCard against real training work (Route B Wave 3c).
 * @name TrainingAttestation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingAttestation
 */
export declare const TrainingAttestation: {
    typeUrl: string;
    encode(message: TrainingAttestation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TrainingAttestation;
    fromPartial(object: DeepPartial<TrainingAttestation>): TrainingAttestation;
};
/**
 * ContributionRecord attributes the facts a training run consumed. Enables
 * contributor-share rewards, reproducibility audit, and analysis of whose
 * data disproportionately trained a given model (Route B Wave 3b).
 * @name ContributionRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContributionRecord
 */
export declare const ContributionRecord: {
    typeUrl: string;
    encode(message: ContributionRecord, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ContributionRecord;
    fromPartial(object: DeepPartial<ContributionRecord>): ContributionRecord;
};
/**
 * AugmentationBounty is an open offer to produce variant formulations of a
 * target fact. Sponsors lock a reward *in escrow* (Wave 4); payout routes
 * through a verifier-panel verdict, never through the sponsor directly.
 * @name AugmentationBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.AugmentationBounty
 */
export declare const AugmentationBounty: {
    typeUrl: string;
    encode(message: AugmentationBounty, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AugmentationBounty;
    fromPartial(object: DeepPartial<AugmentationBounty>): AugmentationBounty;
};
/**
 * Augmentation is a variant formulation of an original fact. Acceptance
 * requires a verifier-panel verdict (Wave 4) — the sponsor never judges.
 * @name Augmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Augmentation
 */
export declare const Augmentation: {
    typeUrl: string;
    encode(message: Augmentation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Augmentation;
    fromPartial(object: DeepPartial<Augmentation>): Augmentation;
};
/**
 * ContributionChallenge is a bonded dispute over whether a model's declared
 * ContributionRecord is accurate: a fact submitter asserts under-reporting
 * (the model used their fact but didn't attribute) or over-reporting (the
 * owner listed a colluder's fact that wasn't used). Current resolution is an
 * explicit governance-authority action; there is no verifier-panel dispatch.
 * @name ContributionChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContributionChallenge
 */
export declare const ContributionChallenge: {
    typeUrl: string;
    encode(message: ContributionChallenge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ContributionChallenge;
    fromPartial(object: DeepPartial<ContributionChallenge>): ContributionChallenge;
};
/**
 * TrainingFundDisbursement is the retained state shape for historical/imported
 * records and a future replay-safe reward design. Current public claims are
 * release-disabled and create no new records.
 * @name TrainingFundDisbursement
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingFundDisbursement
 */
export declare const TrainingFundDisbursement: {
    typeUrl: string;
    encode(message: TrainingFundDisbursement, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TrainingFundDisbursement;
    fromPartial(object: DeepPartial<TrainingFundDisbursement>): TrainingFundDisbursement;
};
/**
 * AgentMethodStats tracks calibration for a single submitter within a single
 * methodology. Populated on round completion and challenge outcomes.
 * @name AgentMethodStats
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.AgentMethodStats
 */
export declare const AgentMethodStats: {
    typeUrl: string;
    encode(message: AgentMethodStats, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AgentMethodStats;
    fromPartial(object: DeepPartial<AgentMethodStats>): AgentMethodStats;
};
/**
 * @name AgentCalibration_PerMethodEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export declare const AgentCalibration_PerMethodEntry: {
    encode(message: AgentCalibration_PerMethodEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AgentCalibration_PerMethodEntry;
    fromPartial(object: DeepPartial<AgentCalibration_PerMethodEntry>): AgentCalibration_PerMethodEntry;
};
/**
 * AgentCalibration is the feedback record for a submitter — agent or human.
 * It is the mechanism by which ZERONE-trained models (and human participants)
 * are measured against the same adjudication they were trained on. Closes
 * the loop between training pipeline output and on-chain evaluation (Phase 5).
 * @name AgentCalibration
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.AgentCalibration
 */
export declare const AgentCalibration: {
    typeUrl: string;
    encode(message: AgentCalibration, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AgentCalibration;
    fromPartial(object: DeepPartial<AgentCalibration>): AgentCalibration;
};
/**
 * CommonKnowledgeEntry represents a subject that LLMs already know.
 * Claims matching these subjects receive a novelty penalty.
 * @name CommonKnowledgeEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CommonKnowledgeEntry
 */
export declare const CommonKnowledgeEntry: {
    typeUrl: string;
    encode(message: CommonKnowledgeEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CommonKnowledgeEntry;
    fromPartial(object: DeepPartial<CommonKnowledgeEntry>): CommonKnowledgeEntry;
};
/**
 * Claim is an unverified submission awaiting or undergoing verification.
 * @name Claim
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Claim
 */
export declare const Claim: {
    typeUrl: string;
    encode(message: Claim, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Claim;
    fromPartial(object: DeepPartial<Claim>): Claim;
};
/**
 * VerificationRound tracks one commit-reveal verification cycle.
 * @name VerificationRound
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.VerificationRound
 */
export declare const VerificationRound: {
    typeUrl: string;
    encode(message: VerificationRound, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): VerificationRound;
    fromPartial(object: DeepPartial<VerificationRound>): VerificationRound;
};
/**
 * CommitEntry records a validator's blinded commitment (SHA-256(vote || salt)).
 * @name CommitEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CommitEntry
 */
export declare const CommitEntry: {
    typeUrl: string;
    encode(message: CommitEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CommitEntry;
    fromPartial(object: DeepPartial<CommitEntry>): CommitEntry;
};
/**
 * RevealEntry records a validator's revealed vote and salt.
 * @name RevealEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RevealEntry
 */
export declare const RevealEntry: {
    typeUrl: string;
    encode(message: RevealEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): RevealEntry;
    fromPartial(object: DeepPartial<RevealEntry>): RevealEntry;
};
/**
 * VRFProof captures a Verifiable Random Function output for validator selection.
 * @name VRFProof
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.VRFProof
 */
export declare const VRFProof: {
    typeUrl: string;
    encode(message: VRFProof, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): VRFProof;
    fromPartial(object: DeepPartial<VRFProof>): VRFProof;
};
/**
 * Domain is an epistemic knowledge domain (e.g., "mathematics", "physics").
 * @name Domain
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Domain
 */
export declare const Domain: {
    typeUrl: string;
    encode(message: Domain, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Domain;
    fromPartial(object: DeepPartial<Domain>): Domain;
};
/**
 * ValidatorInfo caches a validator's tier and verification stats for round selection.
 * @name ValidatorInfo
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ValidatorInfo
 */
export declare const ValidatorInfo: {
    typeUrl: string;
    encode(message: ValidatorInfo, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ValidatorInfo;
    fromPartial(object: DeepPartial<ValidatorInfo>): ValidatorInfo;
};
/**
 * ProvisionalChallenge is an adversarial challenge against a provisional fact.
 * @name ProvisionalChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ProvisionalChallenge
 */
export declare const ProvisionalChallenge: {
    typeUrl: string;
    encode(message: ProvisionalChallenge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ProvisionalChallenge;
    fromPartial(object: DeepPartial<ProvisionalChallenge>): ProvisionalChallenge;
};
/**
 * DemandSignal tracks aggregate query demand for a domain/subject pair.
 * @name DemandSignal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DemandSignal
 */
export declare const DemandSignal: {
    typeUrl: string;
    encode(message: DemandSignal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DemandSignal;
    fromPartial(object: DeepPartial<DemandSignal>): DemandSignal;
};
/**
 * KnowledgeBounty is an auto-generated reward for filling a knowledge gap.
 * @name KnowledgeBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.KnowledgeBounty
 */
export declare const KnowledgeBounty: {
    typeUrl: string;
    encode(message: KnowledgeBounty, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): KnowledgeBounty;
    fromPartial(object: DeepPartial<KnowledgeBounty>): KnowledgeBounty;
};
/**
 * CompletedRoundMeta stores metadata for completed verification rounds,
 * indexed by verdict block height for efficient window-based queries (R31-2).
 * @name CompletedRoundMeta
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CompletedRoundMeta
 */
export declare const CompletedRoundMeta: {
    typeUrl: string;
    encode(message: CompletedRoundMeta, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CompletedRoundMeta;
    fromPartial(object: DeepPartial<CompletedRoundMeta>): CompletedRoundMeta;
};
/**
 * @name MethodologyApplicationTrace
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MethodologyApplicationTrace
 */
export declare const MethodologyApplicationTrace: {
    typeUrl: string;
    encode(message: MethodologyApplicationTrace, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MethodologyApplicationTrace;
    fromPartial(object: DeepPartial<MethodologyApplicationTrace>): MethodologyApplicationTrace;
};
/**
 * @name TraceChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceChallenge
 */
export declare const TraceChallenge: {
    typeUrl: string;
    encode(message: TraceChallenge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TraceChallenge;
    fromPartial(object: DeepPartial<TraceChallenge>): TraceChallenge;
};
/**
 * ReasoningStep is one unit of a structured reasoning trace. The model
 * learns to generate these in sequence, each step grounded in prior facts
 * and declared moves.
 * @name ReasoningStep
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ReasoningStep
 */
export declare const ReasoningStep: {
    typeUrl: string;
    encode(message: ReasoningStep, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ReasoningStep;
    fromPartial(object: DeepPartial<ReasoningStep>): ReasoningStep;
};
/**
 * DriftDiagnosis carries the verifier panel's diagnosis of WHERE and HOW
 * the variant's meaning slipped. Populated on DRIFT/INFERIOR verdicts.
 * @name DriftDiagnosis
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DriftDiagnosis
 */
export declare const DriftDiagnosis: {
    typeUrl: string;
    encode(message: DriftDiagnosis, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DriftDiagnosis;
    fromPartial(object: DeepPartial<DriftDiagnosis>): DriftDiagnosis;
};
/**
 * @name MethodologyChoice
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MethodologyChoice
 */
export declare const MethodologyChoice: {
    typeUrl: string;
    encode(message: MethodologyChoice, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MethodologyChoice;
    fromPartial(object: DeepPartial<MethodologyChoice>): MethodologyChoice;
};
/**
 * @name BeliefRevision
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.BeliefRevision
 */
export declare const BeliefRevision: {
    typeUrl: string;
    encode(message: BeliefRevision, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): BeliefRevision;
    fromPartial(object: DeepPartial<BeliefRevision>): BeliefRevision;
};
/**
 * @name DialecticNode
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DialecticNode
 */
export declare const DialecticNode: {
    typeUrl: string;
    encode(message: DialecticNode, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DialecticNode;
    fromPartial(object: DeepPartial<DialecticNode>): DialecticNode;
};
/**
 * @name TraceVindication
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceVindication
 */
export declare const TraceVindication: {
    typeUrl: string;
    encode(message: TraceVindication, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TraceVindication;
    fromPartial(object: DeepPartial<TraceVindication>): TraceVindication;
};
/**
 * @name TraceDisproval
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceDisproval
 */
export declare const TraceDisproval: {
    typeUrl: string;
    encode(message: TraceDisproval, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TraceDisproval;
    fromPartial(object: DeepPartial<TraceDisproval>): TraceDisproval;
};
/**
 * @name TraceReformulation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceReformulation
 */
export declare const TraceReformulation: {
    typeUrl: string;
    encode(message: TraceReformulation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TraceReformulation;
    fromPartial(object: DeepPartial<TraceReformulation>): TraceReformulation;
};
/**
 * @name TraceDrift
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceDrift
 */
export declare const TraceDrift: {
    typeUrl: string;
    encode(message: TraceDrift, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TraceDrift;
    fromPartial(object: DeepPartial<TraceDrift>): TraceDrift;
};
/**
 * ContrastivePair is a (positive, negative, verdict) training row for
 * preference learning. ZERONE's unique lever: web crawl only has survivors;
 * this format ships the LOSING side with the adjudication that beat it.
 * @name ContrastivePair
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContrastivePair
 */
export declare const ContrastivePair: {
    typeUrl: string;
    encode(message: ContrastivePair, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ContrastivePair;
    fromPartial(object: DeepPartial<ContrastivePair>): ContrastivePair;
};
/**
 * TraceSchema names the canonical JSON Schema for MethodologyApplicationTrace.
 * Versioned and governance-amendable like TokenizerSpec: any training run
 * can pin to a specific schema version and reconstruct the deterministic
 * serialisation years later.
 * @name TraceSchema
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceSchema
 */
export declare const TraceSchema: {
    typeUrl: string;
    encode(message: TraceSchema, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TraceSchema;
    fromPartial(object: DeepPartial<TraceSchema>): TraceSchema;
};
/**
 * CorpusSelector is the declarative predicate that, applied against the
 * chain state at snapshot_block_height, reproduces the manifest's included
 * ID set exactly. Pure data — no side effects — so the derivation is
 * auditable and replayable.
 * @name CorpusSelector
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CorpusSelector
 */
export declare const CorpusSelector: {
    typeUrl: string;
    encode(message: CorpusSelector, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CorpusSelector;
    fromPartial(object: DeepPartial<CorpusSelector>): CorpusSelector;
};
/**
 * TrainingManifest is the atomic, verifiable unit of a training run: every
 * version pin, every included ID, one Merkle root. Distributed as JSON for
 * off-chain consumers; the on-chain record is authoritative.
 * @name TrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingManifest
 */
export declare const TrainingManifest: {
    typeUrl: string;
    encode(message: TrainingManifest, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TrainingManifest;
    fromPartial(object: DeepPartial<TrainingManifest>): TrainingManifest;
};
/**
 * SeedStatus reports which bootstrap seeds have run. A freshly-spawned
 * chain reports all false; SeedRouteB brings them all to true.
 * @name SeedStatus
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.SeedStatus
 */
export declare const SeedStatus: {
    typeUrl: string;
    encode(message: SeedStatus, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): SeedStatus;
    fromPartial(object: DeepPartial<SeedStatus>): SeedStatus;
};
/**
 * RouteBCapabilities is the chain's self-description — what versions of
 * what contracts it exposes, and how much state it holds right now. The
 * first query a trainer runs against a new chain.
 * @name RouteBCapabilities
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RouteBCapabilities
 */
export declare const RouteBCapabilities: {
    typeUrl: string;
    encode(message: RouteBCapabilities, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): RouteBCapabilities;
    fromPartial(object: DeepPartial<RouteBCapabilities>): RouteBCapabilities;
};
/**
 * Remediation is one recorded action taken against an incident. Multiple
 * remediations may attach to a single incident (e.g., P1 bug gets a param
 * amendment for immediate relief, then a named upgrade for the permanent
 * fix, then documentation).
 * @name Remediation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Remediation
 */
export declare const Remediation: {
    typeUrl: string;
    encode(message: Remediation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Remediation;
    fromPartial(object: DeepPartial<Remediation>): Remediation;
};
/**
 * IncidentRecord is the structured, auditable log entry for a single bug
 * or incident. Authority-gated CRUD; the record is append-only from the
 * perspective of resolution (remediations can be added, status can advance,
 * but never retract).
 * @name IncidentRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.IncidentRecord
 */
export declare const IncidentRecord: {
    typeUrl: string;
    encode(message: IncidentRecord, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): IncidentRecord;
    fromPartial(object: DeepPartial<IncidentRecord>): IncidentRecord;
};
/**
 * @name ModulePause
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ModulePause
 */
export declare const ModulePause: {
    typeUrl: string;
    encode(message: ModulePause, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ModulePause;
    fromPartial(object: DeepPartial<ModulePause>): ModulePause;
};
/**
 * @name PrivilegedAction
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.PrivilegedAction
 */
export declare const PrivilegedAction: {
    typeUrl: string;
    encode(message: PrivilegedAction, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): PrivilegedAction;
    fromPartial(object: DeepPartial<PrivilegedAction>): PrivilegedAction;
};
/**
 * PendingFactInjection is a fact that the authority has proposed via
 * MsgAddFact but which has not yet materialized — it is held in a
 * queue for AddFactVetoWindowBlocks while any guardian listed in
 * Params.guardian_addresses can cancel it via MsgVetoFactInjection.
 * After the window expires without veto, BeginBlocker materializes
 * the fact (creates the actual Fact record). This is the Wave 16
 * multi-sig defense for the only authority path that bypasses
 * verification: a single-key compromise can no longer silently inject
 * content into the training corpus.
 * @name PendingFactInjection
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.PendingFactInjection
 */
export declare const PendingFactInjection: {
    typeUrl: string;
    encode(message: PendingFactInjection, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): PendingFactInjection;
    fromPartial(object: DeepPartial<PendingFactInjection>): PendingFactInjection;
};
