//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/** FactStatus represents the lifecycle state of a verified fact. */
export enum FactStatus {
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
  UNRECOGNIZED = -1,
}
export function factStatusFromJSON(object: any): FactStatus {
  switch (object) {
    case 0:
    case "FACT_STATUS_UNSPECIFIED":
      return FactStatus.FACT_STATUS_UNSPECIFIED;
    case 1:
    case "FACT_STATUS_PENDING":
      return FactStatus.FACT_STATUS_PENDING;
    case 2:
    case "FACT_STATUS_PROVISIONAL":
      return FactStatus.FACT_STATUS_PROVISIONAL;
    case 3:
    case "FACT_STATUS_VERIFIED":
      return FactStatus.FACT_STATUS_VERIFIED;
    case 4:
    case "FACT_STATUS_ACTIVE":
      return FactStatus.FACT_STATUS_ACTIVE;
    case 5:
    case "FACT_STATUS_CONTESTED":
      return FactStatus.FACT_STATUS_CONTESTED;
    case 6:
    case "FACT_STATUS_CHALLENGED":
      return FactStatus.FACT_STATUS_CHALLENGED;
    case 7:
    case "FACT_STATUS_SUPERSEDED":
      return FactStatus.FACT_STATUS_SUPERSEDED;
    case 8:
    case "FACT_STATUS_EXPIRED":
      return FactStatus.FACT_STATUS_EXPIRED;
    case 9:
    case "FACT_STATUS_DISPROVEN":
      return FactStatus.FACT_STATUS_DISPROVEN;
    case 10:
    case "FACT_STATUS_REVOKED":
      return FactStatus.FACT_STATUS_REVOKED;
    case 11:
    case "FACT_STATUS_AT_RISK":
      return FactStatus.FACT_STATUS_AT_RISK;
    case 12:
    case "FACT_STATUS_PRUNED":
      return FactStatus.FACT_STATUS_PRUNED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return FactStatus.UNRECOGNIZED;
  }
}
export function factStatusToJSON(object: FactStatus): string {
  switch (object) {
    case FactStatus.FACT_STATUS_UNSPECIFIED:
      return "FACT_STATUS_UNSPECIFIED";
    case FactStatus.FACT_STATUS_PENDING:
      return "FACT_STATUS_PENDING";
    case FactStatus.FACT_STATUS_PROVISIONAL:
      return "FACT_STATUS_PROVISIONAL";
    case FactStatus.FACT_STATUS_VERIFIED:
      return "FACT_STATUS_VERIFIED";
    case FactStatus.FACT_STATUS_ACTIVE:
      return "FACT_STATUS_ACTIVE";
    case FactStatus.FACT_STATUS_CONTESTED:
      return "FACT_STATUS_CONTESTED";
    case FactStatus.FACT_STATUS_CHALLENGED:
      return "FACT_STATUS_CHALLENGED";
    case FactStatus.FACT_STATUS_SUPERSEDED:
      return "FACT_STATUS_SUPERSEDED";
    case FactStatus.FACT_STATUS_EXPIRED:
      return "FACT_STATUS_EXPIRED";
    case FactStatus.FACT_STATUS_DISPROVEN:
      return "FACT_STATUS_DISPROVEN";
    case FactStatus.FACT_STATUS_REVOKED:
      return "FACT_STATUS_REVOKED";
    case FactStatus.FACT_STATUS_AT_RISK:
      return "FACT_STATUS_AT_RISK";
    case FactStatus.FACT_STATUS_PRUNED:
      return "FACT_STATUS_PRUNED";
    case FactStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/** ClaimStatus represents the lifecycle state of a submitted claim. */
export enum ClaimStatus {
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
  UNRECOGNIZED = -1,
}
export function claimStatusFromJSON(object: any): ClaimStatus {
  switch (object) {
    case 0:
    case "CLAIM_STATUS_UNSPECIFIED":
      return ClaimStatus.CLAIM_STATUS_UNSPECIFIED;
    case 1:
    case "CLAIM_STATUS_PENDING":
      return ClaimStatus.CLAIM_STATUS_PENDING;
    case 2:
    case "CLAIM_STATUS_PENDING_EVALUATION":
      return ClaimStatus.CLAIM_STATUS_PENDING_EVALUATION;
    case 3:
    case "CLAIM_STATUS_EVALUATED":
      return ClaimStatus.CLAIM_STATUS_EVALUATED;
    case 4:
    case "CLAIM_STATUS_PROVISIONAL":
      return ClaimStatus.CLAIM_STATUS_PROVISIONAL;
    case 5:
    case "CLAIM_STATUS_IN_VERIFICATION":
      return ClaimStatus.CLAIM_STATUS_IN_VERIFICATION;
    case 6:
    case "CLAIM_STATUS_ACCEPTED":
      return ClaimStatus.CLAIM_STATUS_ACCEPTED;
    case 7:
    case "CLAIM_STATUS_REJECTED":
      return ClaimStatus.CLAIM_STATUS_REJECTED;
    case 8:
    case "CLAIM_STATUS_CHALLENGED":
      return ClaimStatus.CLAIM_STATUS_CHALLENGED;
    case 9:
    case "CLAIM_STATUS_EXPIRED":
      return ClaimStatus.CLAIM_STATUS_EXPIRED;
    case 10:
    case "CLAIM_STATUS_INSUFFICIENT":
      return ClaimStatus.CLAIM_STATUS_INSUFFICIENT;
    case 11:
    case "CLAIM_STATUS_CONTESTED":
      return ClaimStatus.CLAIM_STATUS_CONTESTED;
    case 12:
    case "CLAIM_STATUS_MALFORMED":
      return ClaimStatus.CLAIM_STATUS_MALFORMED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ClaimStatus.UNRECOGNIZED;
  }
}
export function claimStatusToJSON(object: ClaimStatus): string {
  switch (object) {
    case ClaimStatus.CLAIM_STATUS_UNSPECIFIED:
      return "CLAIM_STATUS_UNSPECIFIED";
    case ClaimStatus.CLAIM_STATUS_PENDING:
      return "CLAIM_STATUS_PENDING";
    case ClaimStatus.CLAIM_STATUS_PENDING_EVALUATION:
      return "CLAIM_STATUS_PENDING_EVALUATION";
    case ClaimStatus.CLAIM_STATUS_EVALUATED:
      return "CLAIM_STATUS_EVALUATED";
    case ClaimStatus.CLAIM_STATUS_PROVISIONAL:
      return "CLAIM_STATUS_PROVISIONAL";
    case ClaimStatus.CLAIM_STATUS_IN_VERIFICATION:
      return "CLAIM_STATUS_IN_VERIFICATION";
    case ClaimStatus.CLAIM_STATUS_ACCEPTED:
      return "CLAIM_STATUS_ACCEPTED";
    case ClaimStatus.CLAIM_STATUS_REJECTED:
      return "CLAIM_STATUS_REJECTED";
    case ClaimStatus.CLAIM_STATUS_CHALLENGED:
      return "CLAIM_STATUS_CHALLENGED";
    case ClaimStatus.CLAIM_STATUS_EXPIRED:
      return "CLAIM_STATUS_EXPIRED";
    case ClaimStatus.CLAIM_STATUS_INSUFFICIENT:
      return "CLAIM_STATUS_INSUFFICIENT";
    case ClaimStatus.CLAIM_STATUS_CONTESTED:
      return "CLAIM_STATUS_CONTESTED";
    case ClaimStatus.CLAIM_STATUS_MALFORMED:
      return "CLAIM_STATUS_MALFORMED";
    case ClaimStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/** VerificationPhase tracks progress of a commit-reveal verification round. */
export enum VerificationPhase {
  VERIFICATION_PHASE_UNSPECIFIED = 0,
  VERIFICATION_PHASE_COMMIT = 1,
  VERIFICATION_PHASE_REVEAL = 2,
  VERIFICATION_PHASE_AGGREGATION = 3,
  VERIFICATION_PHASE_COMPLETE = 4,
  VERIFICATION_PHASE_EXPIRED = 5,
  UNRECOGNIZED = -1,
}
export function verificationPhaseFromJSON(object: any): VerificationPhase {
  switch (object) {
    case 0:
    case "VERIFICATION_PHASE_UNSPECIFIED":
      return VerificationPhase.VERIFICATION_PHASE_UNSPECIFIED;
    case 1:
    case "VERIFICATION_PHASE_COMMIT":
      return VerificationPhase.VERIFICATION_PHASE_COMMIT;
    case 2:
    case "VERIFICATION_PHASE_REVEAL":
      return VerificationPhase.VERIFICATION_PHASE_REVEAL;
    case 3:
    case "VERIFICATION_PHASE_AGGREGATION":
      return VerificationPhase.VERIFICATION_PHASE_AGGREGATION;
    case 4:
    case "VERIFICATION_PHASE_COMPLETE":
      return VerificationPhase.VERIFICATION_PHASE_COMPLETE;
    case 5:
    case "VERIFICATION_PHASE_EXPIRED":
      return VerificationPhase.VERIFICATION_PHASE_EXPIRED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return VerificationPhase.UNRECOGNIZED;
  }
}
export function verificationPhaseToJSON(object: VerificationPhase): string {
  switch (object) {
    case VerificationPhase.VERIFICATION_PHASE_UNSPECIFIED:
      return "VERIFICATION_PHASE_UNSPECIFIED";
    case VerificationPhase.VERIFICATION_PHASE_COMMIT:
      return "VERIFICATION_PHASE_COMMIT";
    case VerificationPhase.VERIFICATION_PHASE_REVEAL:
      return "VERIFICATION_PHASE_REVEAL";
    case VerificationPhase.VERIFICATION_PHASE_AGGREGATION:
      return "VERIFICATION_PHASE_AGGREGATION";
    case VerificationPhase.VERIFICATION_PHASE_COMPLETE:
      return "VERIFICATION_PHASE_COMPLETE";
    case VerificationPhase.VERIFICATION_PHASE_EXPIRED:
      return "VERIFICATION_PHASE_EXPIRED";
    case VerificationPhase.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/** Verdict is the outcome of a completed verification round. */
export enum Verdict {
  VERDICT_UNSPECIFIED = 0,
  VERDICT_ACCEPT = 1,
  VERDICT_REJECT = 2,
  VERDICT_INCONCLUSIVE = 3,
  /** VERDICT_MALFORMED - Claim is not truth-apt (paradox, category error, nonsense) */
  VERDICT_MALFORMED = 4,
  UNRECOGNIZED = -1,
}
export function verdictFromJSON(object: any): Verdict {
  switch (object) {
    case 0:
    case "VERDICT_UNSPECIFIED":
      return Verdict.VERDICT_UNSPECIFIED;
    case 1:
    case "VERDICT_ACCEPT":
      return Verdict.VERDICT_ACCEPT;
    case 2:
    case "VERDICT_REJECT":
      return Verdict.VERDICT_REJECT;
    case 3:
    case "VERDICT_INCONCLUSIVE":
      return Verdict.VERDICT_INCONCLUSIVE;
    case 4:
    case "VERDICT_MALFORMED":
      return Verdict.VERDICT_MALFORMED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return Verdict.UNRECOGNIZED;
  }
}
export function verdictToJSON(object: Verdict): string {
  switch (object) {
    case Verdict.VERDICT_UNSPECIFIED:
      return "VERDICT_UNSPECIFIED";
    case Verdict.VERDICT_ACCEPT:
      return "VERDICT_ACCEPT";
    case Verdict.VERDICT_REJECT:
      return "VERDICT_REJECT";
    case Verdict.VERDICT_INCONCLUSIVE:
      return "VERDICT_INCONCLUSIVE";
    case Verdict.VERDICT_MALFORMED:
      return "VERDICT_MALFORMED";
    case Verdict.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * ClaimType classifies the epistemic shape of a knowledge claim.
 * Agents use this to filter and prioritize facts for prompt injection.
 */
export enum ClaimType {
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
  UNRECOGNIZED = -1,
}
export function claimTypeFromJSON(object: any): ClaimType {
  switch (object) {
    case 0:
    case "CLAIM_TYPE_UNSPECIFIED":
      return ClaimType.CLAIM_TYPE_UNSPECIFIED;
    case 1:
    case "CLAIM_TYPE_ASSERTION":
      return ClaimType.CLAIM_TYPE_ASSERTION;
    case 2:
    case "CLAIM_TYPE_RELATION":
      return ClaimType.CLAIM_TYPE_RELATION;
    case 3:
    case "CLAIM_TYPE_DEFINITION":
      return ClaimType.CLAIM_TYPE_DEFINITION;
    case 4:
    case "CLAIM_TYPE_CONSTRAINT":
      return ClaimType.CLAIM_TYPE_CONSTRAINT;
    case 5:
    case "CLAIM_TYPE_NEGATION":
      return ClaimType.CLAIM_TYPE_NEGATION;
    case 6:
    case "CLAIM_TYPE_OBSERVATION":
      return ClaimType.CLAIM_TYPE_OBSERVATION;
    case 7:
    case "CLAIM_TYPE_COMPUTATIONAL":
      return ClaimType.CLAIM_TYPE_COMPUTATIONAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ClaimType.UNRECOGNIZED;
  }
}
export function claimTypeToJSON(object: ClaimType): string {
  switch (object) {
    case ClaimType.CLAIM_TYPE_UNSPECIFIED:
      return "CLAIM_TYPE_UNSPECIFIED";
    case ClaimType.CLAIM_TYPE_ASSERTION:
      return "CLAIM_TYPE_ASSERTION";
    case ClaimType.CLAIM_TYPE_RELATION:
      return "CLAIM_TYPE_RELATION";
    case ClaimType.CLAIM_TYPE_DEFINITION:
      return "CLAIM_TYPE_DEFINITION";
    case ClaimType.CLAIM_TYPE_CONSTRAINT:
      return "CLAIM_TYPE_CONSTRAINT";
    case ClaimType.CLAIM_TYPE_NEGATION:
      return "CLAIM_TYPE_NEGATION";
    case ClaimType.CLAIM_TYPE_OBSERVATION:
      return "CLAIM_TYPE_OBSERVATION";
    case ClaimType.CLAIM_TYPE_COMPUTATIONAL:
      return "CLAIM_TYPE_COMPUTATIONAL";
    case ClaimType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/** RelationType defines how one fact relates to another. */
export enum RelationType {
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
  UNRECOGNIZED = -1,
}
export function relationTypeFromJSON(object: any): RelationType {
  switch (object) {
    case 0:
    case "RELATION_TYPE_UNSPECIFIED":
      return RelationType.RELATION_TYPE_UNSPECIFIED;
    case 1:
    case "RELATION_TYPE_SUPPORTS":
      return RelationType.RELATION_TYPE_SUPPORTS;
    case 2:
    case "RELATION_TYPE_CONTRADICTS":
      return RelationType.RELATION_TYPE_CONTRADICTS;
    case 3:
    case "RELATION_TYPE_REQUIRES":
      return RelationType.RELATION_TYPE_REQUIRES;
    case 4:
    case "RELATION_TYPE_REFINES":
      return RelationType.RELATION_TYPE_REFINES;
    case 5:
    case "RELATION_TYPE_GENERALIZES":
      return RelationType.RELATION_TYPE_GENERALIZES;
    case 6:
    case "RELATION_TYPE_SUPERSEDES":
      return RelationType.RELATION_TYPE_SUPERSEDES;
    case 7:
    case "RELATION_TYPE_CITES":
      return RelationType.RELATION_TYPE_CITES;
    case 8:
    case "RELATION_TYPE_REFORMULATES":
      return RelationType.RELATION_TYPE_REFORMULATES;
    case -1:
    case "UNRECOGNIZED":
    default:
      return RelationType.UNRECOGNIZED;
  }
}
export function relationTypeToJSON(object: RelationType): string {
  switch (object) {
    case RelationType.RELATION_TYPE_UNSPECIFIED:
      return "RELATION_TYPE_UNSPECIFIED";
    case RelationType.RELATION_TYPE_SUPPORTS:
      return "RELATION_TYPE_SUPPORTS";
    case RelationType.RELATION_TYPE_CONTRADICTS:
      return "RELATION_TYPE_CONTRADICTS";
    case RelationType.RELATION_TYPE_REQUIRES:
      return "RELATION_TYPE_REQUIRES";
    case RelationType.RELATION_TYPE_REFINES:
      return "RELATION_TYPE_REFINES";
    case RelationType.RELATION_TYPE_GENERALIZES:
      return "RELATION_TYPE_GENERALIZES";
    case RelationType.RELATION_TYPE_SUPERSEDES:
      return "RELATION_TYPE_SUPERSEDES";
    case RelationType.RELATION_TYPE_CITES:
      return "RELATION_TYPE_CITES";
    case RelationType.RELATION_TYPE_REFORMULATES:
      return "RELATION_TYPE_REFORMULATES";
    case RelationType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * InferenceType names HOW the source fact was derived from the target.
 * Orthogonal to RelationType: RelationType is structural ("A supports B"),
 * InferenceType is epistemic ("A deductively entails B"). Used for proof-tree
 * audit and confidence propagation.
 */
export enum InferenceType {
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
  UNRECOGNIZED = -1,
}
export function inferenceTypeFromJSON(object: any): InferenceType {
  switch (object) {
    case 0:
    case "INFERENCE_TYPE_UNSPECIFIED":
      return InferenceType.INFERENCE_TYPE_UNSPECIFIED;
    case 1:
    case "INFERENCE_TYPE_DEDUCTIVE":
      return InferenceType.INFERENCE_TYPE_DEDUCTIVE;
    case 2:
    case "INFERENCE_TYPE_INDUCTIVE":
      return InferenceType.INFERENCE_TYPE_INDUCTIVE;
    case 3:
    case "INFERENCE_TYPE_ABDUCTIVE":
      return InferenceType.INFERENCE_TYPE_ABDUCTIVE;
    case 4:
    case "INFERENCE_TYPE_EMPIRICAL":
      return InferenceType.INFERENCE_TYPE_EMPIRICAL;
    case 5:
    case "INFERENCE_TYPE_ANALOGICAL":
      return InferenceType.INFERENCE_TYPE_ANALOGICAL;
    case 6:
    case "INFERENCE_TYPE_CITATION":
      return InferenceType.INFERENCE_TYPE_CITATION;
    case -1:
    case "UNRECOGNIZED":
    default:
      return InferenceType.UNRECOGNIZED;
  }
}
export function inferenceTypeToJSON(object: InferenceType): string {
  switch (object) {
    case InferenceType.INFERENCE_TYPE_UNSPECIFIED:
      return "INFERENCE_TYPE_UNSPECIFIED";
    case InferenceType.INFERENCE_TYPE_DEDUCTIVE:
      return "INFERENCE_TYPE_DEDUCTIVE";
    case InferenceType.INFERENCE_TYPE_INDUCTIVE:
      return "INFERENCE_TYPE_INDUCTIVE";
    case InferenceType.INFERENCE_TYPE_ABDUCTIVE:
      return "INFERENCE_TYPE_ABDUCTIVE";
    case InferenceType.INFERENCE_TYPE_EMPIRICAL:
      return "INFERENCE_TYPE_EMPIRICAL";
    case InferenceType.INFERENCE_TYPE_ANALOGICAL:
      return "INFERENCE_TYPE_ANALOGICAL";
    case InferenceType.INFERENCE_TYPE_CITATION:
      return "INFERENCE_TYPE_CITATION";
    case InferenceType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/** DomainStatus tracks whether an epistemic domain is active or proposed. */
export enum DomainStatus {
  DOMAIN_STATUS_UNSPECIFIED = 0,
  DOMAIN_STATUS_ACTIVE = 1,
  DOMAIN_STATUS_DEPRECATED = 2,
  DOMAIN_STATUS_PROPOSED = 3,
  UNRECOGNIZED = -1,
}
export function domainStatusFromJSON(object: any): DomainStatus {
  switch (object) {
    case 0:
    case "DOMAIN_STATUS_UNSPECIFIED":
      return DomainStatus.DOMAIN_STATUS_UNSPECIFIED;
    case 1:
    case "DOMAIN_STATUS_ACTIVE":
      return DomainStatus.DOMAIN_STATUS_ACTIVE;
    case 2:
    case "DOMAIN_STATUS_DEPRECATED":
      return DomainStatus.DOMAIN_STATUS_DEPRECATED;
    case 3:
    case "DOMAIN_STATUS_PROPOSED":
      return DomainStatus.DOMAIN_STATUS_PROPOSED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DomainStatus.UNRECOGNIZED;
  }
}
export function domainStatusToJSON(object: DomainStatus): string {
  switch (object) {
    case DomainStatus.DOMAIN_STATUS_UNSPECIFIED:
      return "DOMAIN_STATUS_UNSPECIFIED";
    case DomainStatus.DOMAIN_STATUS_ACTIVE:
      return "DOMAIN_STATUS_ACTIVE";
    case DomainStatus.DOMAIN_STATUS_DEPRECATED:
      return "DOMAIN_STATUS_DEPRECATED";
    case DomainStatus.DOMAIN_STATUS_PROPOSED:
      return "DOMAIN_STATUS_PROPOSED";
    case DomainStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * CurriculumTier orders training examples by difficulty and foundational depth.
 * Relocated here in Wave 5 so MethodologyApplicationTrace can reference it.
 */
export enum CurriculumTier {
  CURRICULUM_TIER_UNSPECIFIED = 0,
  /** CURRICULUM_TIER_FOUNDATION - axiom_distance ≤ 1, high corroboration, simple methods */
  CURRICULUM_TIER_FOUNDATION = 1,
  CURRICULUM_TIER_INTERMEDIATE = 2,
  /** CURRICULUM_TIER_ADVANCED - deep derivation chains */
  CURRICULUM_TIER_ADVANCED = 3,
  /** CURRICULUM_TIER_SPECIALISED - niche methodologies (phenomenological, ecological, practice) */
  CURRICULUM_TIER_SPECIALISED = 4,
  UNRECOGNIZED = -1,
}
export function curriculumTierFromJSON(object: any): CurriculumTier {
  switch (object) {
    case 0:
    case "CURRICULUM_TIER_UNSPECIFIED":
      return CurriculumTier.CURRICULUM_TIER_UNSPECIFIED;
    case 1:
    case "CURRICULUM_TIER_FOUNDATION":
      return CurriculumTier.CURRICULUM_TIER_FOUNDATION;
    case 2:
    case "CURRICULUM_TIER_INTERMEDIATE":
      return CurriculumTier.CURRICULUM_TIER_INTERMEDIATE;
    case 3:
    case "CURRICULUM_TIER_ADVANCED":
      return CurriculumTier.CURRICULUM_TIER_ADVANCED;
    case 4:
    case "CURRICULUM_TIER_SPECIALISED":
      return CurriculumTier.CURRICULUM_TIER_SPECIALISED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return CurriculumTier.UNRECOGNIZED;
  }
}
export function curriculumTierToJSON(object: CurriculumTier): string {
  switch (object) {
    case CurriculumTier.CURRICULUM_TIER_UNSPECIFIED:
      return "CURRICULUM_TIER_UNSPECIFIED";
    case CurriculumTier.CURRICULUM_TIER_FOUNDATION:
      return "CURRICULUM_TIER_FOUNDATION";
    case CurriculumTier.CURRICULUM_TIER_INTERMEDIATE:
      return "CURRICULUM_TIER_INTERMEDIATE";
    case CurriculumTier.CURRICULUM_TIER_ADVANCED:
      return "CURRICULUM_TIER_ADVANCED";
    case CurriculumTier.CURRICULUM_TIER_SPECIALISED:
      return "CURRICULUM_TIER_SPECIALISED";
    case CurriculumTier.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * TrainingQualityTier segments facts by suitability as training examples.
 * Computed on demand from corroboration, methodology, and status.
 */
export enum TrainingQualityTier {
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
  UNRECOGNIZED = -1,
}
export function trainingQualityTierFromJSON(object: any): TrainingQualityTier {
  switch (object) {
    case 0:
    case "TRAINING_QUALITY_TIER_UNSPECIFIED":
      return TrainingQualityTier.TRAINING_QUALITY_TIER_UNSPECIFIED;
    case 1:
    case "TRAINING_QUALITY_TIER_GOLD":
      return TrainingQualityTier.TRAINING_QUALITY_TIER_GOLD;
    case 2:
    case "TRAINING_QUALITY_TIER_SILVER":
      return TrainingQualityTier.TRAINING_QUALITY_TIER_SILVER;
    case 3:
    case "TRAINING_QUALITY_TIER_BRONZE":
      return TrainingQualityTier.TRAINING_QUALITY_TIER_BRONZE;
    case 4:
    case "TRAINING_QUALITY_TIER_NEGATIVE":
      return TrainingQualityTier.TRAINING_QUALITY_TIER_NEGATIVE;
    case 5:
    case "TRAINING_QUALITY_TIER_UNSUITABLE":
      return TrainingQualityTier.TRAINING_QUALITY_TIER_UNSUITABLE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return TrainingQualityTier.UNRECOGNIZED;
  }
}
export function trainingQualityTierToJSON(object: TrainingQualityTier): string {
  switch (object) {
    case TrainingQualityTier.TRAINING_QUALITY_TIER_UNSPECIFIED:
      return "TRAINING_QUALITY_TIER_UNSPECIFIED";
    case TrainingQualityTier.TRAINING_QUALITY_TIER_GOLD:
      return "TRAINING_QUALITY_TIER_GOLD";
    case TrainingQualityTier.TRAINING_QUALITY_TIER_SILVER:
      return "TRAINING_QUALITY_TIER_SILVER";
    case TrainingQualityTier.TRAINING_QUALITY_TIER_BRONZE:
      return "TRAINING_QUALITY_TIER_BRONZE";
    case TrainingQualityTier.TRAINING_QUALITY_TIER_NEGATIVE:
      return "TRAINING_QUALITY_TIER_NEGATIVE";
    case TrainingQualityTier.TRAINING_QUALITY_TIER_UNSUITABLE:
      return "TRAINING_QUALITY_TIER_UNSUITABLE";
    case TrainingQualityTier.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/** AugmentationVerdict is the verifier-panel outcome for a reformulation. */
export enum AugmentationVerdict {
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
  UNRECOGNIZED = -1,
}
export function augmentationVerdictFromJSON(object: any): AugmentationVerdict {
  switch (object) {
    case 0:
    case "AUGMENTATION_VERDICT_PENDING":
      return AugmentationVerdict.AUGMENTATION_VERDICT_PENDING;
    case 1:
    case "AUGMENTATION_VERDICT_EQUIVALENT":
      return AugmentationVerdict.AUGMENTATION_VERDICT_EQUIVALENT;
    case 2:
    case "AUGMENTATION_VERDICT_SUPERIOR":
      return AugmentationVerdict.AUGMENTATION_VERDICT_SUPERIOR;
    case 3:
    case "AUGMENTATION_VERDICT_INFERIOR":
      return AugmentationVerdict.AUGMENTATION_VERDICT_INFERIOR;
    case 4:
    case "AUGMENTATION_VERDICT_DRIFT":
      return AugmentationVerdict.AUGMENTATION_VERDICT_DRIFT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return AugmentationVerdict.UNRECOGNIZED;
  }
}
export function augmentationVerdictToJSON(object: AugmentationVerdict): string {
  switch (object) {
    case AugmentationVerdict.AUGMENTATION_VERDICT_PENDING:
      return "AUGMENTATION_VERDICT_PENDING";
    case AugmentationVerdict.AUGMENTATION_VERDICT_EQUIVALENT:
      return "AUGMENTATION_VERDICT_EQUIVALENT";
    case AugmentationVerdict.AUGMENTATION_VERDICT_SUPERIOR:
      return "AUGMENTATION_VERDICT_SUPERIOR";
    case AugmentationVerdict.AUGMENTATION_VERDICT_INFERIOR:
      return "AUGMENTATION_VERDICT_INFERIOR";
    case AugmentationVerdict.AUGMENTATION_VERDICT_DRIFT:
      return "AUGMENTATION_VERDICT_DRIFT";
    case AugmentationVerdict.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * StepInference names the epistemic move a single reasoning step makes.
 * Distinct from InferenceType (which describes a FactRelation edge) — this
 * is about reasoning moves WITHIN a trace, e.g. substitution, elimination.
 */
export enum StepInference {
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
  UNRECOGNIZED = -1,
}
export function stepInferenceFromJSON(object: any): StepInference {
  switch (object) {
    case 0:
    case "STEP_INFERENCE_UNSPECIFIED":
      return StepInference.STEP_INFERENCE_UNSPECIFIED;
    case 1:
    case "STEP_INFERENCE_OBSERVATION":
      return StepInference.STEP_INFERENCE_OBSERVATION;
    case 2:
    case "STEP_INFERENCE_DEFINITION":
      return StepInference.STEP_INFERENCE_DEFINITION;
    case 3:
    case "STEP_INFERENCE_DEDUCTION":
      return StepInference.STEP_INFERENCE_DEDUCTION;
    case 4:
    case "STEP_INFERENCE_INDUCTION":
      return StepInference.STEP_INFERENCE_INDUCTION;
    case 5:
    case "STEP_INFERENCE_ABDUCTION":
      return StepInference.STEP_INFERENCE_ABDUCTION;
    case 6:
    case "STEP_INFERENCE_ANALOGY":
      return StepInference.STEP_INFERENCE_ANALOGY;
    case 7:
    case "STEP_INFERENCE_DECOMPOSITION":
      return StepInference.STEP_INFERENCE_DECOMPOSITION;
    case 8:
    case "STEP_INFERENCE_CASE_SPLIT":
      return StepInference.STEP_INFERENCE_CASE_SPLIT;
    case 9:
    case "STEP_INFERENCE_CONTRADICTION":
      return StepInference.STEP_INFERENCE_CONTRADICTION;
    case 10:
    case "STEP_INFERENCE_UNIT_CONVERSION":
      return StepInference.STEP_INFERENCE_UNIT_CONVERSION;
    case 11:
    case "STEP_INFERENCE_VERIFICATION":
      return StepInference.STEP_INFERENCE_VERIFICATION;
    case 12:
    case "STEP_INFERENCE_CONCLUSION":
      return StepInference.STEP_INFERENCE_CONCLUSION;
    case -1:
    case "UNRECOGNIZED":
    default:
      return StepInference.UNRECOGNIZED;
  }
}
export function stepInferenceToJSON(object: StepInference): string {
  switch (object) {
    case StepInference.STEP_INFERENCE_UNSPECIFIED:
      return "STEP_INFERENCE_UNSPECIFIED";
    case StepInference.STEP_INFERENCE_OBSERVATION:
      return "STEP_INFERENCE_OBSERVATION";
    case StepInference.STEP_INFERENCE_DEFINITION:
      return "STEP_INFERENCE_DEFINITION";
    case StepInference.STEP_INFERENCE_DEDUCTION:
      return "STEP_INFERENCE_DEDUCTION";
    case StepInference.STEP_INFERENCE_INDUCTION:
      return "STEP_INFERENCE_INDUCTION";
    case StepInference.STEP_INFERENCE_ABDUCTION:
      return "STEP_INFERENCE_ABDUCTION";
    case StepInference.STEP_INFERENCE_ANALOGY:
      return "STEP_INFERENCE_ANALOGY";
    case StepInference.STEP_INFERENCE_DECOMPOSITION:
      return "STEP_INFERENCE_DECOMPOSITION";
    case StepInference.STEP_INFERENCE_CASE_SPLIT:
      return "STEP_INFERENCE_CASE_SPLIT";
    case StepInference.STEP_INFERENCE_CONTRADICTION:
      return "STEP_INFERENCE_CONTRADICTION";
    case StepInference.STEP_INFERENCE_UNIT_CONVERSION:
      return "STEP_INFERENCE_UNIT_CONVERSION";
    case StepInference.STEP_INFERENCE_VERIFICATION:
      return "STEP_INFERENCE_VERIFICATION";
    case StepInference.STEP_INFERENCE_CONCLUSION:
      return "STEP_INFERENCE_CONCLUSION";
    case StepInference.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * StepVerdict mirrors verifier panel judgment at step granularity. When a
 * PoT round's verifiers examine reasoning, their approvals/disapprovals
 * attach per-step — not just to the final claim. This makes PRMs trainable.
 */
export enum StepVerdict {
  STEP_VERDICT_UNSPECIFIED = 0,
  /** STEP_VERDICT_UNEXAMINED - default: no panel review at step level */
  STEP_VERDICT_UNEXAMINED = 1,
  /** STEP_VERDICT_SOUND - step holds under the claimed methodology */
  STEP_VERDICT_SOUND = 2,
  /** STEP_VERDICT_QUESTIONABLE - step may not follow; panel flagged */
  STEP_VERDICT_QUESTIONABLE = 3,
  /** STEP_VERDICT_UNSOUND - step does not follow; explicit reject */
  STEP_VERDICT_UNSOUND = 4,
  UNRECOGNIZED = -1,
}
export function stepVerdictFromJSON(object: any): StepVerdict {
  switch (object) {
    case 0:
    case "STEP_VERDICT_UNSPECIFIED":
      return StepVerdict.STEP_VERDICT_UNSPECIFIED;
    case 1:
    case "STEP_VERDICT_UNEXAMINED":
      return StepVerdict.STEP_VERDICT_UNEXAMINED;
    case 2:
    case "STEP_VERDICT_SOUND":
      return StepVerdict.STEP_VERDICT_SOUND;
    case 3:
    case "STEP_VERDICT_QUESTIONABLE":
      return StepVerdict.STEP_VERDICT_QUESTIONABLE;
    case 4:
    case "STEP_VERDICT_UNSOUND":
      return StepVerdict.STEP_VERDICT_UNSOUND;
    case -1:
    case "UNRECOGNIZED":
    default:
      return StepVerdict.UNRECOGNIZED;
  }
}
export function stepVerdictToJSON(object: StepVerdict): string {
  switch (object) {
    case StepVerdict.STEP_VERDICT_UNSPECIFIED:
      return "STEP_VERDICT_UNSPECIFIED";
    case StepVerdict.STEP_VERDICT_UNEXAMINED:
      return "STEP_VERDICT_UNEXAMINED";
    case StepVerdict.STEP_VERDICT_SOUND:
      return "STEP_VERDICT_SOUND";
    case StepVerdict.STEP_VERDICT_QUESTIONABLE:
      return "STEP_VERDICT_QUESTIONABLE";
    case StepVerdict.STEP_VERDICT_UNSOUND:
      return "STEP_VERDICT_UNSOUND";
    case StepVerdict.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum DriftKind {
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
  UNRECOGNIZED = -1,
}
export function driftKindFromJSON(object: any): DriftKind {
  switch (object) {
    case 0:
    case "DRIFT_KIND_UNSPECIFIED":
      return DriftKind.DRIFT_KIND_UNSPECIFIED;
    case 1:
    case "DRIFT_KIND_NARROWED":
      return DriftKind.DRIFT_KIND_NARROWED;
    case 2:
    case "DRIFT_KIND_WIDENED":
      return DriftKind.DRIFT_KIND_WIDENED;
    case 3:
    case "DRIFT_KIND_REFERENT_SWAP":
      return DriftKind.DRIFT_KIND_REFERENT_SWAP;
    case 4:
    case "DRIFT_KIND_POLARITY_FLIP":
      return DriftKind.DRIFT_KIND_POLARITY_FLIP;
    case 5:
    case "DRIFT_KIND_MODAL_SHIFT":
      return DriftKind.DRIFT_KIND_MODAL_SHIFT;
    case 6:
    case "DRIFT_KIND_DOMAIN_CONFLATION":
      return DriftKind.DRIFT_KIND_DOMAIN_CONFLATION;
    case 7:
    case "DRIFT_KIND_HEDGE_REMOVED":
      return DriftKind.DRIFT_KIND_HEDGE_REMOVED;
    case 8:
    case "DRIFT_KIND_HEDGE_ADDED":
      return DriftKind.DRIFT_KIND_HEDGE_ADDED;
    case 9:
    case "DRIFT_KIND_TEMPORAL_SHIFT":
      return DriftKind.DRIFT_KIND_TEMPORAL_SHIFT;
    case 10:
    case "DRIFT_KIND_CAUSAL_CONFLATION":
      return DriftKind.DRIFT_KIND_CAUSAL_CONFLATION;
    case 11:
    case "DRIFT_KIND_METHODOLOGY_SWAP":
      return DriftKind.DRIFT_KIND_METHODOLOGY_SWAP;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DriftKind.UNRECOGNIZED;
  }
}
export function driftKindToJSON(object: DriftKind): string {
  switch (object) {
    case DriftKind.DRIFT_KIND_UNSPECIFIED:
      return "DRIFT_KIND_UNSPECIFIED";
    case DriftKind.DRIFT_KIND_NARROWED:
      return "DRIFT_KIND_NARROWED";
    case DriftKind.DRIFT_KIND_WIDENED:
      return "DRIFT_KIND_WIDENED";
    case DriftKind.DRIFT_KIND_REFERENT_SWAP:
      return "DRIFT_KIND_REFERENT_SWAP";
    case DriftKind.DRIFT_KIND_POLARITY_FLIP:
      return "DRIFT_KIND_POLARITY_FLIP";
    case DriftKind.DRIFT_KIND_MODAL_SHIFT:
      return "DRIFT_KIND_MODAL_SHIFT";
    case DriftKind.DRIFT_KIND_DOMAIN_CONFLATION:
      return "DRIFT_KIND_DOMAIN_CONFLATION";
    case DriftKind.DRIFT_KIND_HEDGE_REMOVED:
      return "DRIFT_KIND_HEDGE_REMOVED";
    case DriftKind.DRIFT_KIND_HEDGE_ADDED:
      return "DRIFT_KIND_HEDGE_ADDED";
    case DriftKind.DRIFT_KIND_TEMPORAL_SHIFT:
      return "DRIFT_KIND_TEMPORAL_SHIFT";
    case DriftKind.DRIFT_KIND_CAUSAL_CONFLATION:
      return "DRIFT_KIND_CAUSAL_CONFLATION";
    case DriftKind.DRIFT_KIND_METHODOLOGY_SWAP:
      return "DRIFT_KIND_METHODOLOGY_SWAP";
    case DriftKind.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum RevisionReason {
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
  UNRECOGNIZED = -1,
}
export function revisionReasonFromJSON(object: any): RevisionReason {
  switch (object) {
    case 0:
    case "REVISION_REASON_UNSPECIFIED":
      return RevisionReason.REVISION_REASON_UNSPECIFIED;
    case 1:
    case "REVISION_REASON_CORROBORATION":
      return RevisionReason.REVISION_REASON_CORROBORATION;
    case 2:
    case "REVISION_REASON_CITATION":
      return RevisionReason.REVISION_REASON_CITATION;
    case 3:
    case "REVISION_REASON_CONTRADICTION":
      return RevisionReason.REVISION_REASON_CONTRADICTION;
    case 4:
    case "REVISION_REASON_REBUTTAL":
      return RevisionReason.REVISION_REASON_REBUTTAL;
    case 5:
    case "REVISION_REASON_VINDICATION":
      return RevisionReason.REVISION_REASON_VINDICATION;
    case 6:
    case "REVISION_REASON_TEMPORAL_DECAY":
      return RevisionReason.REVISION_REASON_TEMPORAL_DECAY;
    case 7:
    case "REVISION_REASON_RESUBMISSION":
      return RevisionReason.REVISION_REASON_RESUBMISSION;
    case 8:
    case "REVISION_REASON_METHODOLOGY_AMENDED":
      return RevisionReason.REVISION_REASON_METHODOLOGY_AMENDED;
    case 9:
    case "REVISION_REASON_CROSS_DOMAIN_SUPPORT":
      return RevisionReason.REVISION_REASON_CROSS_DOMAIN_SUPPORT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return RevisionReason.UNRECOGNIZED;
  }
}
export function revisionReasonToJSON(object: RevisionReason): string {
  switch (object) {
    case RevisionReason.REVISION_REASON_UNSPECIFIED:
      return "REVISION_REASON_UNSPECIFIED";
    case RevisionReason.REVISION_REASON_CORROBORATION:
      return "REVISION_REASON_CORROBORATION";
    case RevisionReason.REVISION_REASON_CITATION:
      return "REVISION_REASON_CITATION";
    case RevisionReason.REVISION_REASON_CONTRADICTION:
      return "REVISION_REASON_CONTRADICTION";
    case RevisionReason.REVISION_REASON_REBUTTAL:
      return "REVISION_REASON_REBUTTAL";
    case RevisionReason.REVISION_REASON_VINDICATION:
      return "REVISION_REASON_VINDICATION";
    case RevisionReason.REVISION_REASON_TEMPORAL_DECAY:
      return "REVISION_REASON_TEMPORAL_DECAY";
    case RevisionReason.REVISION_REASON_RESUBMISSION:
      return "REVISION_REASON_RESUBMISSION";
    case RevisionReason.REVISION_REASON_METHODOLOGY_AMENDED:
      return "REVISION_REASON_METHODOLOGY_AMENDED";
    case RevisionReason.REVISION_REASON_CROSS_DOMAIN_SUPPORT:
      return "REVISION_REASON_CROSS_DOMAIN_SUPPORT";
    case RevisionReason.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum DialecticRole {
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
  UNRECOGNIZED = -1,
}
export function dialecticRoleFromJSON(object: any): DialecticRole {
  switch (object) {
    case 0:
    case "DIALECTIC_ROLE_UNSPECIFIED":
      return DialecticRole.DIALECTIC_ROLE_UNSPECIFIED;
    case 1:
    case "DIALECTIC_ROLE_CHALLENGE":
      return DialecticRole.DIALECTIC_ROLE_CHALLENGE;
    case 2:
    case "DIALECTIC_ROLE_REBUTTAL":
      return DialecticRole.DIALECTIC_ROLE_REBUTTAL;
    case 3:
    case "DIALECTIC_ROLE_COUNTER":
      return DialecticRole.DIALECTIC_ROLE_COUNTER;
    case 4:
    case "DIALECTIC_ROLE_CONCESSION":
      return DialecticRole.DIALECTIC_ROLE_CONCESSION;
    case 5:
    case "DIALECTIC_ROLE_VERDICT":
      return DialecticRole.DIALECTIC_ROLE_VERDICT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DialecticRole.UNRECOGNIZED;
  }
}
export function dialecticRoleToJSON(object: DialecticRole): string {
  switch (object) {
    case DialecticRole.DIALECTIC_ROLE_UNSPECIFIED:
      return "DIALECTIC_ROLE_UNSPECIFIED";
    case DialecticRole.DIALECTIC_ROLE_CHALLENGE:
      return "DIALECTIC_ROLE_CHALLENGE";
    case DialecticRole.DIALECTIC_ROLE_REBUTTAL:
      return "DIALECTIC_ROLE_REBUTTAL";
    case DialecticRole.DIALECTIC_ROLE_COUNTER:
      return "DIALECTIC_ROLE_COUNTER";
    case DialecticRole.DIALECTIC_ROLE_CONCESSION:
      return "DIALECTIC_ROLE_CONCESSION";
    case DialecticRole.DIALECTIC_ROLE_VERDICT:
      return "DIALECTIC_ROLE_VERDICT";
    case DialecticRole.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * ContrastivePairType enumerates the epistemic relationship between the two
 * members of a contrastive pair. The (type, method_id, distinguishing
 * argument) triple tells a trainer WHY the positive beat the negative.
 */
export enum ContrastivePairType {
  CONTRASTIVE_PAIR_UNSPECIFIED = 0,
  /** CONTRASTIVE_PAIR_SURVIVED_VS_DISPROVEN - fact survived; a contradictory claim was disproven */
  CONTRASTIVE_PAIR_SURVIVED_VS_DISPROVEN = 1,
  /** CONTRASTIVE_PAIR_EQUIVALENT_VS_DRIFT - winning reformulation vs DRIFT variant on same original */
  CONTRASTIVE_PAIR_EQUIVALENT_VS_DRIFT = 2,
  /** CONTRASTIVE_PAIR_EQUIVALENT_VS_INFERIOR - winning reformulation vs INFERIOR variant */
  CONTRASTIVE_PAIR_EQUIVALENT_VS_INFERIOR = 3,
  /** CONTRASTIVE_PAIR_VINDICATED_MINORITY - the vindicated minority vote vs the (disproven) majority */
  CONTRASTIVE_PAIR_VINDICATED_MINORITY = 4,
  UNRECOGNIZED = -1,
}
export function contrastivePairTypeFromJSON(object: any): ContrastivePairType {
  switch (object) {
    case 0:
    case "CONTRASTIVE_PAIR_UNSPECIFIED":
      return ContrastivePairType.CONTRASTIVE_PAIR_UNSPECIFIED;
    case 1:
    case "CONTRASTIVE_PAIR_SURVIVED_VS_DISPROVEN":
      return ContrastivePairType.CONTRASTIVE_PAIR_SURVIVED_VS_DISPROVEN;
    case 2:
    case "CONTRASTIVE_PAIR_EQUIVALENT_VS_DRIFT":
      return ContrastivePairType.CONTRASTIVE_PAIR_EQUIVALENT_VS_DRIFT;
    case 3:
    case "CONTRASTIVE_PAIR_EQUIVALENT_VS_INFERIOR":
      return ContrastivePairType.CONTRASTIVE_PAIR_EQUIVALENT_VS_INFERIOR;
    case 4:
    case "CONTRASTIVE_PAIR_VINDICATED_MINORITY":
      return ContrastivePairType.CONTRASTIVE_PAIR_VINDICATED_MINORITY;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ContrastivePairType.UNRECOGNIZED;
  }
}
export function contrastivePairTypeToJSON(object: ContrastivePairType): string {
  switch (object) {
    case ContrastivePairType.CONTRASTIVE_PAIR_UNSPECIFIED:
      return "CONTRASTIVE_PAIR_UNSPECIFIED";
    case ContrastivePairType.CONTRASTIVE_PAIR_SURVIVED_VS_DISPROVEN:
      return "CONTRASTIVE_PAIR_SURVIVED_VS_DISPROVEN";
    case ContrastivePairType.CONTRASTIVE_PAIR_EQUIVALENT_VS_DRIFT:
      return "CONTRASTIVE_PAIR_EQUIVALENT_VS_DRIFT";
    case ContrastivePairType.CONTRASTIVE_PAIR_EQUIVALENT_VS_INFERIOR:
      return "CONTRASTIVE_PAIR_EQUIVALENT_VS_INFERIOR";
    case ContrastivePairType.CONTRASTIVE_PAIR_VINDICATED_MINORITY:
      return "CONTRASTIVE_PAIR_VINDICATED_MINORITY";
    case ContrastivePairType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/** ManifestStatus tracks the manifest lifecycle. */
export enum ManifestStatus {
  MANIFEST_STATUS_UNSPECIFIED = 0,
  /** MANIFEST_STATUS_DRAFT - created, selector locked, IDs computed, but Merkle not frozen */
  MANIFEST_STATUS_DRAFT = 1,
  /** MANIFEST_STATUS_FINALIZED - Merkle root committed; immutable */
  MANIFEST_STATUS_FINALIZED = 2,
  /** MANIFEST_STATUS_ATTESTED - bound to a TrainingAttestation (run complete) */
  MANIFEST_STATUS_ATTESTED = 3,
  /** MANIFEST_STATUS_SUPERSEDED - a later manifest for the same pipeline supersedes this one */
  MANIFEST_STATUS_SUPERSEDED = 4,
  UNRECOGNIZED = -1,
}
export function manifestStatusFromJSON(object: any): ManifestStatus {
  switch (object) {
    case 0:
    case "MANIFEST_STATUS_UNSPECIFIED":
      return ManifestStatus.MANIFEST_STATUS_UNSPECIFIED;
    case 1:
    case "MANIFEST_STATUS_DRAFT":
      return ManifestStatus.MANIFEST_STATUS_DRAFT;
    case 2:
    case "MANIFEST_STATUS_FINALIZED":
      return ManifestStatus.MANIFEST_STATUS_FINALIZED;
    case 3:
    case "MANIFEST_STATUS_ATTESTED":
      return ManifestStatus.MANIFEST_STATUS_ATTESTED;
    case 4:
    case "MANIFEST_STATUS_SUPERSEDED":
      return ManifestStatus.MANIFEST_STATUS_SUPERSEDED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ManifestStatus.UNRECOGNIZED;
  }
}
export function manifestStatusToJSON(object: ManifestStatus): string {
  switch (object) {
    case ManifestStatus.MANIFEST_STATUS_UNSPECIFIED:
      return "MANIFEST_STATUS_UNSPECIFIED";
    case ManifestStatus.MANIFEST_STATUS_DRAFT:
      return "MANIFEST_STATUS_DRAFT";
    case ManifestStatus.MANIFEST_STATUS_FINALIZED:
      return "MANIFEST_STATUS_FINALIZED";
    case ManifestStatus.MANIFEST_STATUS_ATTESTED:
      return "MANIFEST_STATUS_ATTESTED";
    case ManifestStatus.MANIFEST_STATUS_SUPERSEDED:
      return "MANIFEST_STATUS_SUPERSEDED";
    case ManifestStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum IncidentSeverity {
  INCIDENT_SEVERITY_UNSPECIFIED = 0,
  /** INCIDENT_SEVERITY_P0 - consensus break / chain halt required */
  INCIDENT_SEVERITY_P0 = 1,
  /** INCIDENT_SEVERITY_P1 - high-impact; immediate fix via param amendment or emergency upgrade */
  INCIDENT_SEVERITY_P1 = 2,
  /** INCIDENT_SEVERITY_P2 - scheduled upgrade; no halt needed */
  INCIDENT_SEVERITY_P2 = 3,
  /** INCIDENT_SEVERITY_P3 - low-impact; next-release or documentation-only */
  INCIDENT_SEVERITY_P3 = 4,
  UNRECOGNIZED = -1,
}
export function incidentSeverityFromJSON(object: any): IncidentSeverity {
  switch (object) {
    case 0:
    case "INCIDENT_SEVERITY_UNSPECIFIED":
      return IncidentSeverity.INCIDENT_SEVERITY_UNSPECIFIED;
    case 1:
    case "INCIDENT_SEVERITY_P0":
      return IncidentSeverity.INCIDENT_SEVERITY_P0;
    case 2:
    case "INCIDENT_SEVERITY_P1":
      return IncidentSeverity.INCIDENT_SEVERITY_P1;
    case 3:
    case "INCIDENT_SEVERITY_P2":
      return IncidentSeverity.INCIDENT_SEVERITY_P2;
    case 4:
    case "INCIDENT_SEVERITY_P3":
      return IncidentSeverity.INCIDENT_SEVERITY_P3;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IncidentSeverity.UNRECOGNIZED;
  }
}
export function incidentSeverityToJSON(object: IncidentSeverity): string {
  switch (object) {
    case IncidentSeverity.INCIDENT_SEVERITY_UNSPECIFIED:
      return "INCIDENT_SEVERITY_UNSPECIFIED";
    case IncidentSeverity.INCIDENT_SEVERITY_P0:
      return "INCIDENT_SEVERITY_P0";
    case IncidentSeverity.INCIDENT_SEVERITY_P1:
      return "INCIDENT_SEVERITY_P1";
    case IncidentSeverity.INCIDENT_SEVERITY_P2:
      return "INCIDENT_SEVERITY_P2";
    case IncidentSeverity.INCIDENT_SEVERITY_P3:
      return "INCIDENT_SEVERITY_P3";
    case IncidentSeverity.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum IncidentStatus {
  INCIDENT_STATUS_UNSPECIFIED = 0,
  /** INCIDENT_STATUS_OPEN - triaging; no remediation yet */
  INCIDENT_STATUS_OPEN = 1,
  /** INCIDENT_STATUS_MITIGATING - remediation(s) applied, still monitoring */
  INCIDENT_STATUS_MITIGATING = 2,
  /** INCIDENT_STATUS_RESOLVED - fix verified; monitoring window closed */
  INCIDENT_STATUS_RESOLVED = 3,
  /** INCIDENT_STATUS_CLOSED - post-mortem published; permanently archived */
  INCIDENT_STATUS_CLOSED = 4,
  UNRECOGNIZED = -1,
}
export function incidentStatusFromJSON(object: any): IncidentStatus {
  switch (object) {
    case 0:
    case "INCIDENT_STATUS_UNSPECIFIED":
      return IncidentStatus.INCIDENT_STATUS_UNSPECIFIED;
    case 1:
    case "INCIDENT_STATUS_OPEN":
      return IncidentStatus.INCIDENT_STATUS_OPEN;
    case 2:
    case "INCIDENT_STATUS_MITIGATING":
      return IncidentStatus.INCIDENT_STATUS_MITIGATING;
    case 3:
    case "INCIDENT_STATUS_RESOLVED":
      return IncidentStatus.INCIDENT_STATUS_RESOLVED;
    case 4:
    case "INCIDENT_STATUS_CLOSED":
      return IncidentStatus.INCIDENT_STATUS_CLOSED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return IncidentStatus.UNRECOGNIZED;
  }
}
export function incidentStatusToJSON(object: IncidentStatus): string {
  switch (object) {
    case IncidentStatus.INCIDENT_STATUS_UNSPECIFIED:
      return "INCIDENT_STATUS_UNSPECIFIED";
    case IncidentStatus.INCIDENT_STATUS_OPEN:
      return "INCIDENT_STATUS_OPEN";
    case IncidentStatus.INCIDENT_STATUS_MITIGATING:
      return "INCIDENT_STATUS_MITIGATING";
    case IncidentStatus.INCIDENT_STATUS_RESOLVED:
      return "INCIDENT_STATUS_RESOLVED";
    case IncidentStatus.INCIDENT_STATUS_CLOSED:
      return "INCIDENT_STATUS_CLOSED";
    case IncidentStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum RemediationType {
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
  UNRECOGNIZED = -1,
}
export function remediationTypeFromJSON(object: any): RemediationType {
  switch (object) {
    case 0:
    case "REMEDIATION_TYPE_UNSPECIFIED":
      return RemediationType.REMEDIATION_TYPE_UNSPECIFIED;
    case 1:
    case "REMEDIATION_TYPE_PARAM_AMENDMENT":
      return RemediationType.REMEDIATION_TYPE_PARAM_AMENDMENT;
    case 2:
    case "REMEDIATION_TYPE_NAMED_UPGRADE":
      return RemediationType.REMEDIATION_TYPE_NAMED_UPGRADE;
    case 3:
    case "REMEDIATION_TYPE_EMERGENCY_HALT":
      return RemediationType.REMEDIATION_TYPE_EMERGENCY_HALT;
    case 4:
    case "REMEDIATION_TYPE_EMERGENCY_RESUME":
      return RemediationType.REMEDIATION_TYPE_EMERGENCY_RESUME;
    case 5:
    case "REMEDIATION_TYPE_STATE_CORRECTION":
      return RemediationType.REMEDIATION_TYPE_STATE_CORRECTION;
    case 6:
    case "REMEDIATION_TYPE_SCHEMA_AMENDMENT":
      return RemediationType.REMEDIATION_TYPE_SCHEMA_AMENDMENT;
    case 7:
    case "REMEDIATION_TYPE_DOCUMENTATION":
      return RemediationType.REMEDIATION_TYPE_DOCUMENTATION;
    case -1:
    case "UNRECOGNIZED":
    default:
      return RemediationType.UNRECOGNIZED;
  }
}
export function remediationTypeToJSON(object: RemediationType): string {
  switch (object) {
    case RemediationType.REMEDIATION_TYPE_UNSPECIFIED:
      return "REMEDIATION_TYPE_UNSPECIFIED";
    case RemediationType.REMEDIATION_TYPE_PARAM_AMENDMENT:
      return "REMEDIATION_TYPE_PARAM_AMENDMENT";
    case RemediationType.REMEDIATION_TYPE_NAMED_UPGRADE:
      return "REMEDIATION_TYPE_NAMED_UPGRADE";
    case RemediationType.REMEDIATION_TYPE_EMERGENCY_HALT:
      return "REMEDIATION_TYPE_EMERGENCY_HALT";
    case RemediationType.REMEDIATION_TYPE_EMERGENCY_RESUME:
      return "REMEDIATION_TYPE_EMERGENCY_RESUME";
    case RemediationType.REMEDIATION_TYPE_STATE_CORRECTION:
      return "REMEDIATION_TYPE_STATE_CORRECTION";
    case RemediationType.REMEDIATION_TYPE_SCHEMA_AMENDMENT:
      return "REMEDIATION_TYPE_SCHEMA_AMENDMENT";
    case RemediationType.REMEDIATION_TYPE_DOCUMENTATION:
      return "REMEDIATION_TYPE_DOCUMENTATION";
    case RemediationType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum PrivilegedActionType {
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
  UNRECOGNIZED = -1,
}
export function privilegedActionTypeFromJSON(object: any): PrivilegedActionType {
  switch (object) {
    case 0:
    case "PRIVILEGED_ACTION_TYPE_UNSPECIFIED":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_UNSPECIFIED;
    case 1:
    case "PRIVILEGED_ACTION_TYPE_MODULE_PAUSE":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_MODULE_PAUSE;
    case 2:
    case "PRIVILEGED_ACTION_TYPE_MODULE_UNPAUSE":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_MODULE_UNPAUSE;
    case 3:
    case "PRIVILEGED_ACTION_TYPE_MANIFEST_CORRECT":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_MANIFEST_CORRECT;
    case 4:
    case "PRIVILEGED_ACTION_TYPE_INCIDENT_OPEN":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_INCIDENT_OPEN;
    case 5:
    case "PRIVILEGED_ACTION_TYPE_INCIDENT_RESOLVE":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_INCIDENT_RESOLVE;
    case 6:
    case "PRIVILEGED_ACTION_TYPE_INCIDENT_CLOSE":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_INCIDENT_CLOSE;
    case 7:
    case "PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TOKENIZER":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TOKENIZER;
    case 8:
    case "PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TRACE":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TRACE;
    case 9:
    case "PRIVILEGED_ACTION_TYPE_FACT_AUTHORITY_INJECT":
      return PrivilegedActionType.PRIVILEGED_ACTION_TYPE_FACT_AUTHORITY_INJECT;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PrivilegedActionType.UNRECOGNIZED;
  }
}
export function privilegedActionTypeToJSON(object: PrivilegedActionType): string {
  switch (object) {
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_UNSPECIFIED:
      return "PRIVILEGED_ACTION_TYPE_UNSPECIFIED";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_MODULE_PAUSE:
      return "PRIVILEGED_ACTION_TYPE_MODULE_PAUSE";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_MODULE_UNPAUSE:
      return "PRIVILEGED_ACTION_TYPE_MODULE_UNPAUSE";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_MANIFEST_CORRECT:
      return "PRIVILEGED_ACTION_TYPE_MANIFEST_CORRECT";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_INCIDENT_OPEN:
      return "PRIVILEGED_ACTION_TYPE_INCIDENT_OPEN";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_INCIDENT_RESOLVE:
      return "PRIVILEGED_ACTION_TYPE_INCIDENT_RESOLVE";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_INCIDENT_CLOSE:
      return "PRIVILEGED_ACTION_TYPE_INCIDENT_CLOSE";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TOKENIZER:
      return "PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TOKENIZER";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TRACE:
      return "PRIVILEGED_ACTION_TYPE_SCHEMA_AMEND_TRACE";
    case PrivilegedActionType.PRIVILEGED_ACTION_TYPE_FACT_AUTHORITY_INJECT:
      return "PRIVILEGED_ACTION_TYPE_FACT_AUTHORITY_INJECT";
    case PrivilegedActionType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
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
 *   · "The research fund requires dual human+AI authorization" (a constitutional rule)
 *   · "Verification history is public and permanent" (a procedural commitment)
 *
 * Governance governs commitments directly; a supermajority proposal amends
 * them. Commitments can *constrain* other modules operationally (e.g. the
 * dual-key research fund enforces the rule cryptographically), but they do
 * not enter the confidence / axiom-distance / corroboration machinery.
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
 * owner listed a colluder's fact that wasn't used). Resolves through the
 * standard PoT verification layer.
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
   * verifier panel representative (or authority for disputes-of-last-resort)
   */
  resolver: string;
  resolutionNote: string;
}
/**
 * TrainingFundDisbursement is a post-hoc reward to a pipeline whose ModelCard
 * has demonstrated calibration in live deployment. 50% is released at claim
 * time; 50% is held in vesting escrow and clawed back if calibration drops.
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
function createBaseFactRelation(): FactRelation {
  return {
    sourceFactId: "",
    targetFactId: "",
    relation: 0,
    createdAtBlock: BigInt(0),
    creator: "",
    inference: 0,
    inferenceStrengthBps: BigInt(0),
    methodId: ""
  };
}
/**
 * FactRelation is a typed, directional edge in the knowledge graph.
 * @name FactRelation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.FactRelation
 */
export const FactRelation = {
  typeUrl: "/zerone.knowledge.v1.FactRelation",
  encode(message: FactRelation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sourceFactId !== "") {
      writer.uint32(10).string(message.sourceFactId);
    }
    if (message.targetFactId !== "") {
      writer.uint32(18).string(message.targetFactId);
    }
    if (message.relation !== 0) {
      writer.uint32(24).int32(message.relation);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.createdAtBlock);
    }
    if (message.creator !== "") {
      writer.uint32(42).string(message.creator);
    }
    if (message.inference !== 0) {
      writer.uint32(48).int32(message.inference);
    }
    if (message.inferenceStrengthBps !== BigInt(0)) {
      writer.uint32(56).uint64(message.inferenceStrengthBps);
    }
    if (message.methodId !== "") {
      writer.uint32(66).string(message.methodId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): FactRelation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFactRelation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sourceFactId = reader.string();
          break;
        case 2:
          message.targetFactId = reader.string();
          break;
        case 3:
          message.relation = reader.int32() as any;
          break;
        case 4:
          message.createdAtBlock = reader.uint64();
          break;
        case 5:
          message.creator = reader.string();
          break;
        case 6:
          message.inference = reader.int32() as any;
          break;
        case 7:
          message.inferenceStrengthBps = reader.uint64();
          break;
        case 8:
          message.methodId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<FactRelation>): FactRelation {
    const message = createBaseFactRelation();
    message.sourceFactId = object.sourceFactId ?? "";
    message.targetFactId = object.targetFactId ?? "";
    message.relation = object.relation ?? 0;
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.creator = object.creator ?? "";
    message.inference = object.inference ?? 0;
    message.inferenceStrengthBps = object.inferenceStrengthBps !== undefined && object.inferenceStrengthBps !== null ? BigInt(object.inferenceStrengthBps.toString()) : BigInt(0);
    message.methodId = object.methodId ?? "";
    return message;
  }
};
function createBaseClaimRelation(): ClaimRelation {
  return {
    targetFactId: "",
    relation: 0,
    inference: 0,
    inferenceStrengthBps: BigInt(0),
    methodId: ""
  };
}
/**
 * ClaimRelation declares a typed relationship from a claim to an existing fact.
 * @name ClaimRelation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimRelation
 */
export const ClaimRelation = {
  typeUrl: "/zerone.knowledge.v1.ClaimRelation",
  encode(message: ClaimRelation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.targetFactId !== "") {
      writer.uint32(10).string(message.targetFactId);
    }
    if (message.relation !== 0) {
      writer.uint32(16).int32(message.relation);
    }
    if (message.inference !== 0) {
      writer.uint32(24).int32(message.inference);
    }
    if (message.inferenceStrengthBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.inferenceStrengthBps);
    }
    if (message.methodId !== "") {
      writer.uint32(42).string(message.methodId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ClaimRelation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaimRelation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.targetFactId = reader.string();
          break;
        case 2:
          message.relation = reader.int32() as any;
          break;
        case 3:
          message.inference = reader.int32() as any;
          break;
        case 4:
          message.inferenceStrengthBps = reader.uint64();
          break;
        case 5:
          message.methodId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ClaimRelation>): ClaimRelation {
    const message = createBaseClaimRelation();
    message.targetFactId = object.targetFactId ?? "";
    message.relation = object.relation ?? 0;
    message.inference = object.inference ?? 0;
    message.inferenceStrengthBps = object.inferenceStrengthBps !== undefined && object.inferenceStrengthBps !== null ? BigInt(object.inferenceStrengthBps.toString()) : BigInt(0);
    message.methodId = object.methodId ?? "";
    return message;
  }
};
function createBaseNormativeCommitment(): NormativeCommitment {
  return {
    id: "",
    statement: "",
    rationale: "",
    category: "",
    tags: [],
    ratifiedAtBlock: BigInt(0),
    version: BigInt(0),
    lastAmendmentProposalId: "",
    active: false,
    referencedFacts: []
  };
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
 *   · "The research fund requires dual human+AI authorization" (a constitutional rule)
 *   · "Verification history is public and permanent" (a procedural commitment)
 *
 * Governance governs commitments directly; a supermajority proposal amends
 * them. Commitments can *constrain* other modules operationally (e.g. the
 * dual-key research fund enforces the rule cryptographically), but they do
 * not enter the confidence / axiom-distance / corroboration machinery.
 * @name NormativeCommitment
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.NormativeCommitment
 */
export const NormativeCommitment = {
  typeUrl: "/zerone.knowledge.v1.NormativeCommitment",
  encode(message: NormativeCommitment, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.statement !== "") {
      writer.uint32(18).string(message.statement);
    }
    if (message.rationale !== "") {
      writer.uint32(26).string(message.rationale);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    for (const v of message.tags) {
      writer.uint32(42).string(v!);
    }
    if (message.ratifiedAtBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.ratifiedAtBlock);
    }
    if (message.version !== BigInt(0)) {
      writer.uint32(56).uint64(message.version);
    }
    if (message.lastAmendmentProposalId !== "") {
      writer.uint32(66).string(message.lastAmendmentProposalId);
    }
    if (message.active === true) {
      writer.uint32(72).bool(message.active);
    }
    for (const v of message.referencedFacts) {
      writer.uint32(82).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): NormativeCommitment {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseNormativeCommitment();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.statement = reader.string();
          break;
        case 3:
          message.rationale = reader.string();
          break;
        case 4:
          message.category = reader.string();
          break;
        case 5:
          message.tags.push(reader.string());
          break;
        case 6:
          message.ratifiedAtBlock = reader.uint64();
          break;
        case 7:
          message.version = reader.uint64();
          break;
        case 8:
          message.lastAmendmentProposalId = reader.string();
          break;
        case 9:
          message.active = reader.bool();
          break;
        case 10:
          message.referencedFacts.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<NormativeCommitment>): NormativeCommitment {
    const message = createBaseNormativeCommitment();
    message.id = object.id ?? "";
    message.statement = object.statement ?? "";
    message.rationale = object.rationale ?? "";
    message.category = object.category ?? "";
    message.tags = object.tags?.map(e => e) || [];
    message.ratifiedAtBlock = object.ratifiedAtBlock !== undefined && object.ratifiedAtBlock !== null ? BigInt(object.ratifiedAtBlock.toString()) : BigInt(0);
    message.version = object.version !== undefined && object.version !== null ? BigInt(object.version.toString()) : BigInt(0);
    message.lastAmendmentProposalId = object.lastAmendmentProposalId ?? "";
    message.active = object.active ?? false;
    message.referencedFacts = object.referencedFacts?.map(e => e) || [];
    return message;
  }
};
function createBaseMethodology_CrossMethodDiscountBpsEntry(): Methodology_CrossMethodDiscountBpsEntry {
  return {
    key: "",
    value: BigInt(0)
  };
}
/**
 * @name Methodology_CrossMethodDiscountBpsEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export const Methodology_CrossMethodDiscountBpsEntry = {
  encode(message: Methodology_CrossMethodDiscountBpsEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== BigInt(0)) {
      writer.uint32(16).uint64(message.value);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Methodology_CrossMethodDiscountBpsEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMethodology_CrossMethodDiscountBpsEntry();
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
  fromPartial(object: DeepPartial<Methodology_CrossMethodDiscountBpsEntry>): Methodology_CrossMethodDiscountBpsEntry {
    const message = createBaseMethodology_CrossMethodDiscountBpsEntry();
    message.key = object.key ?? "";
    message.value = object.value !== undefined && object.value !== null ? BigInt(object.value.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMethodology(): Methodology {
  return {
    id: "",
    name: "",
    description: "",
    complianceCriteria: [],
    falsificationPaths: [],
    crossMethodDiscountBps: {},
    minQualificationWeight: BigInt(0),
    version: BigInt(0),
    ratifiedAtBlock: BigInt(0),
    isTransitional: false
  };
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
export const Methodology = {
  typeUrl: "/zerone.knowledge.v1.Methodology",
  encode(message: Methodology, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    for (const v of message.complianceCriteria) {
      writer.uint32(34).string(v!);
    }
    for (const v of message.falsificationPaths) {
      writer.uint32(42).string(v!);
    }
    Object.entries(message.crossMethodDiscountBps).forEach(([key, value]) => {
      Methodology_CrossMethodDiscountBpsEntry.encode({
        key: key as any,
        value
      }, writer.uint32(48).fork()).ldelim();
    });
    if (message.minQualificationWeight !== BigInt(0)) {
      writer.uint32(56).uint64(message.minQualificationWeight);
    }
    if (message.version !== BigInt(0)) {
      writer.uint32(64).uint64(message.version);
    }
    if (message.ratifiedAtBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.ratifiedAtBlock);
    }
    if (message.isTransitional === true) {
      writer.uint32(80).bool(message.isTransitional);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Methodology {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMethodology();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.complianceCriteria.push(reader.string());
          break;
        case 5:
          message.falsificationPaths.push(reader.string());
          break;
        case 6:
          const entry6 = Methodology_CrossMethodDiscountBpsEntry.decode(reader, reader.uint32());
          if (entry6.value !== undefined) {
            message.crossMethodDiscountBps[entry6.key] = entry6.value;
          }
          break;
        case 7:
          message.minQualificationWeight = reader.uint64();
          break;
        case 8:
          message.version = reader.uint64();
          break;
        case 9:
          message.ratifiedAtBlock = reader.uint64();
          break;
        case 10:
          message.isTransitional = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Methodology>): Methodology {
    const message = createBaseMethodology();
    message.id = object.id ?? "";
    message.name = object.name ?? "";
    message.description = object.description ?? "";
    message.complianceCriteria = object.complianceCriteria?.map(e => e) || [];
    message.falsificationPaths = object.falsificationPaths?.map(e => e) || [];
    message.crossMethodDiscountBps = Object.entries(object.crossMethodDiscountBps ?? {}).reduce<{
      [key: string]: bigint;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = BigInt(value.toString());
      }
      return acc;
    }, {});
    message.minQualificationWeight = object.minQualificationWeight !== undefined && object.minQualificationWeight !== null ? BigInt(object.minQualificationWeight.toString()) : BigInt(0);
    message.version = object.version !== undefined && object.version !== null ? BigInt(object.version.toString()) : BigInt(0);
    message.ratifiedAtBlock = object.ratifiedAtBlock !== undefined && object.ratifiedAtBlock !== null ? BigInt(object.ratifiedAtBlock.toString()) : BigInt(0);
    message.isTransitional = object.isTransitional ?? false;
    return message;
  }
};
function createBaseClaimStructure(): ClaimStructure {
  return {
    subject: "",
    predicate: "",
    object: "",
    scope: "",
    temporalScope: "",
    negatable: false,
    tags: []
  };
}
/**
 * ClaimStructure provides machine-readable decomposition of a claim.
 * The full claim text (fact_content) remains the canonical human-readable form.
 * Structure is optional but strongly encouraged — agents prioritize structured facts.
 * @name ClaimStructure
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ClaimStructure
 */
export const ClaimStructure = {
  typeUrl: "/zerone.knowledge.v1.ClaimStructure",
  encode(message: ClaimStructure, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.subject !== "") {
      writer.uint32(10).string(message.subject);
    }
    if (message.predicate !== "") {
      writer.uint32(18).string(message.predicate);
    }
    if (message.object !== "") {
      writer.uint32(26).string(message.object);
    }
    if (message.scope !== "") {
      writer.uint32(34).string(message.scope);
    }
    if (message.temporalScope !== "") {
      writer.uint32(42).string(message.temporalScope);
    }
    if (message.negatable === true) {
      writer.uint32(48).bool(message.negatable);
    }
    for (const v of message.tags) {
      writer.uint32(58).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ClaimStructure {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaimStructure();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.subject = reader.string();
          break;
        case 2:
          message.predicate = reader.string();
          break;
        case 3:
          message.object = reader.string();
          break;
        case 4:
          message.scope = reader.string();
          break;
        case 5:
          message.temporalScope = reader.string();
          break;
        case 6:
          message.negatable = reader.bool();
          break;
        case 7:
          message.tags.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ClaimStructure>): ClaimStructure {
    const message = createBaseClaimStructure();
    message.subject = object.subject ?? "";
    message.predicate = object.predicate ?? "";
    message.object = object.object ?? "";
    message.scope = object.scope ?? "";
    message.temporalScope = object.temporalScope ?? "";
    message.negatable = object.negatable ?? false;
    message.tags = object.tags?.map(e => e) || [];
    return message;
  }
};
function createBaseFact(): Fact {
  return {
    id: "",
    content: "",
    domain: "",
    category: "",
    confidence: BigInt(0),
    submitter: "",
    submittedAtBlock: BigInt(0),
    verifiedAtBlock: BigInt(0),
    citationCount: BigInt(0),
    fundamentality: BigInt(0),
    references: [],
    status: 0,
    claimId: "",
    reverificationBlock: BigInt(0),
    lastVerifiedBlock: BigInt(0),
    challengeWindowEnd: BigInt(0),
    bridgeScore: BigInt(0),
    noveltyScore: BigInt(0),
    patronageAmount: "",
    patronageExpiryBlock: BigInt(0),
    stratum: "",
    maturity: "",
    incomingCitationCount: BigInt(0),
    claimType: 0,
    outgoingRelations: [],
    incomingRelations: [],
    structure: undefined,
    canonicalForm: "",
    canonicalHash: "",
    fitnessScore: BigInt(0),
    fitnessUpdatedBlock: BigInt(0),
    queryCount: BigInt(0),
    queryCountEpoch: BigInt(0),
    epochBorn: BigInt(0),
    energy: BigInt(0),
    energyCap: BigInt(0),
    energyLastUpdated: BigInt(0),
    atRiskSinceEpoch: BigInt(0),
    nicheKey: "",
    nicheLeader: false,
    nicheRank: BigInt(0),
    nicheSize: BigInt(0),
    competitionTax: BigInt(0),
    parentFactId: "",
    childFactIds: [],
    lineageDepth: BigInt(0),
    progenyCount: BigInt(0),
    lineageRootId: "",
    commonKnowledgeMatch: false,
    satisfactionUp: BigInt(0),
    satisfactionDown: BigInt(0),
    satisfactionUpEpoch: BigInt(0),
    satisfactionDownEpoch: BigInt(0),
    axiomDistance: 0,
    dependencyConfidenceFloor: BigInt(0),
    methodId: "",
    corroborationCount: BigInt(0),
    lastCorroboratedBlock: BigInt(0),
    reasoningTrace: "",
    submitterCalibrationSnapshotBps: BigInt(0),
    trainingRevenueEarned: "",
    trainingRevenueEarnedRecent: "",
    revenueClawbackBlock: BigInt(0),
    probeInvitedAtBlock: BigInt(0)
  };
}
/**
 * Fact represents a piece of verified knowledge in the protocol.
 * Confidence is measured on a 0-1,000,000 BPS scale (1,000,000 = 100%).
 * @name Fact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Fact
 */
export const Fact = {
  typeUrl: "/zerone.knowledge.v1.Fact",
  encode(message: Fact, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.content !== "") {
      writer.uint32(18).string(message.content);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    if (message.confidence !== BigInt(0)) {
      writer.uint32(40).uint64(message.confidence);
    }
    if (message.submitter !== "") {
      writer.uint32(50).string(message.submitter);
    }
    if (message.submittedAtBlock !== BigInt(0)) {
      writer.uint32(56).uint64(message.submittedAtBlock);
    }
    if (message.verifiedAtBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.verifiedAtBlock);
    }
    if (message.citationCount !== BigInt(0)) {
      writer.uint32(72).uint64(message.citationCount);
    }
    if (message.fundamentality !== BigInt(0)) {
      writer.uint32(80).uint64(message.fundamentality);
    }
    for (const v of message.references) {
      writer.uint32(90).string(v!);
    }
    if (message.status !== 0) {
      writer.uint32(96).int32(message.status);
    }
    if (message.claimId !== "") {
      writer.uint32(106).string(message.claimId);
    }
    if (message.reverificationBlock !== BigInt(0)) {
      writer.uint32(112).uint64(message.reverificationBlock);
    }
    if (message.lastVerifiedBlock !== BigInt(0)) {
      writer.uint32(120).uint64(message.lastVerifiedBlock);
    }
    if (message.challengeWindowEnd !== BigInt(0)) {
      writer.uint32(128).uint64(message.challengeWindowEnd);
    }
    if (message.bridgeScore !== BigInt(0)) {
      writer.uint32(136).uint64(message.bridgeScore);
    }
    if (message.noveltyScore !== BigInt(0)) {
      writer.uint32(144).uint64(message.noveltyScore);
    }
    if (message.patronageAmount !== "") {
      writer.uint32(154).string(message.patronageAmount);
    }
    if (message.patronageExpiryBlock !== BigInt(0)) {
      writer.uint32(160).uint64(message.patronageExpiryBlock);
    }
    if (message.stratum !== "") {
      writer.uint32(170).string(message.stratum);
    }
    if (message.maturity !== "") {
      writer.uint32(178).string(message.maturity);
    }
    if (message.incomingCitationCount !== BigInt(0)) {
      writer.uint32(184).uint64(message.incomingCitationCount);
    }
    if (message.claimType !== 0) {
      writer.uint32(192).int32(message.claimType);
    }
    for (const v of message.outgoingRelations) {
      FactRelation.encode(v!, writer.uint32(202).fork()).ldelim();
    }
    for (const v of message.incomingRelations) {
      FactRelation.encode(v!, writer.uint32(210).fork()).ldelim();
    }
    if (message.structure !== undefined) {
      ClaimStructure.encode(message.structure, writer.uint32(218).fork()).ldelim();
    }
    if (message.canonicalForm !== "") {
      writer.uint32(226).string(message.canonicalForm);
    }
    if (message.canonicalHash !== "") {
      writer.uint32(234).string(message.canonicalHash);
    }
    if (message.fitnessScore !== BigInt(0)) {
      writer.uint32(240).uint64(message.fitnessScore);
    }
    if (message.fitnessUpdatedBlock !== BigInt(0)) {
      writer.uint32(248).uint64(message.fitnessUpdatedBlock);
    }
    if (message.queryCount !== BigInt(0)) {
      writer.uint32(256).uint64(message.queryCount);
    }
    if (message.queryCountEpoch !== BigInt(0)) {
      writer.uint32(264).uint64(message.queryCountEpoch);
    }
    if (message.epochBorn !== BigInt(0)) {
      writer.uint32(272).uint64(message.epochBorn);
    }
    if (message.energy !== BigInt(0)) {
      writer.uint32(280).uint64(message.energy);
    }
    if (message.energyCap !== BigInt(0)) {
      writer.uint32(288).uint64(message.energyCap);
    }
    if (message.energyLastUpdated !== BigInt(0)) {
      writer.uint32(296).uint64(message.energyLastUpdated);
    }
    if (message.atRiskSinceEpoch !== BigInt(0)) {
      writer.uint32(304).uint64(message.atRiskSinceEpoch);
    }
    if (message.nicheKey !== "") {
      writer.uint32(314).string(message.nicheKey);
    }
    if (message.nicheLeader === true) {
      writer.uint32(320).bool(message.nicheLeader);
    }
    if (message.nicheRank !== BigInt(0)) {
      writer.uint32(328).uint64(message.nicheRank);
    }
    if (message.nicheSize !== BigInt(0)) {
      writer.uint32(336).uint64(message.nicheSize);
    }
    if (message.competitionTax !== BigInt(0)) {
      writer.uint32(344).uint64(message.competitionTax);
    }
    if (message.parentFactId !== "") {
      writer.uint32(354).string(message.parentFactId);
    }
    for (const v of message.childFactIds) {
      writer.uint32(362).string(v!);
    }
    if (message.lineageDepth !== BigInt(0)) {
      writer.uint32(368).uint64(message.lineageDepth);
    }
    if (message.progenyCount !== BigInt(0)) {
      writer.uint32(376).uint64(message.progenyCount);
    }
    if (message.lineageRootId !== "") {
      writer.uint32(386).string(message.lineageRootId);
    }
    if (message.commonKnowledgeMatch === true) {
      writer.uint32(392).bool(message.commonKnowledgeMatch);
    }
    if (message.satisfactionUp !== BigInt(0)) {
      writer.uint32(480).uint64(message.satisfactionUp);
    }
    if (message.satisfactionDown !== BigInt(0)) {
      writer.uint32(488).uint64(message.satisfactionDown);
    }
    if (message.satisfactionUpEpoch !== BigInt(0)) {
      writer.uint32(496).uint64(message.satisfactionUpEpoch);
    }
    if (message.satisfactionDownEpoch !== BigInt(0)) {
      writer.uint32(504).uint64(message.satisfactionDownEpoch);
    }
    if (message.axiomDistance !== 0) {
      writer.uint32(512).uint32(message.axiomDistance);
    }
    if (message.dependencyConfidenceFloor !== BigInt(0)) {
      writer.uint32(520).uint64(message.dependencyConfidenceFloor);
    }
    if (message.methodId !== "") {
      writer.uint32(530).string(message.methodId);
    }
    if (message.corroborationCount !== BigInt(0)) {
      writer.uint32(536).uint64(message.corroborationCount);
    }
    if (message.lastCorroboratedBlock !== BigInt(0)) {
      writer.uint32(544).uint64(message.lastCorroboratedBlock);
    }
    if (message.reasoningTrace !== "") {
      writer.uint32(554).string(message.reasoningTrace);
    }
    if (message.submitterCalibrationSnapshotBps !== BigInt(0)) {
      writer.uint32(560).uint64(message.submitterCalibrationSnapshotBps);
    }
    if (message.trainingRevenueEarned !== "") {
      writer.uint32(570).string(message.trainingRevenueEarned);
    }
    if (message.trainingRevenueEarnedRecent !== "") {
      writer.uint32(578).string(message.trainingRevenueEarnedRecent);
    }
    if (message.revenueClawbackBlock !== BigInt(0)) {
      writer.uint32(584).uint64(message.revenueClawbackBlock);
    }
    if (message.probeInvitedAtBlock !== BigInt(0)) {
      writer.uint32(592).uint64(message.probeInvitedAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Fact {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.content = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.category = reader.string();
          break;
        case 5:
          message.confidence = reader.uint64();
          break;
        case 6:
          message.submitter = reader.string();
          break;
        case 7:
          message.submittedAtBlock = reader.uint64();
          break;
        case 8:
          message.verifiedAtBlock = reader.uint64();
          break;
        case 9:
          message.citationCount = reader.uint64();
          break;
        case 10:
          message.fundamentality = reader.uint64();
          break;
        case 11:
          message.references.push(reader.string());
          break;
        case 12:
          message.status = reader.int32() as any;
          break;
        case 13:
          message.claimId = reader.string();
          break;
        case 14:
          message.reverificationBlock = reader.uint64();
          break;
        case 15:
          message.lastVerifiedBlock = reader.uint64();
          break;
        case 16:
          message.challengeWindowEnd = reader.uint64();
          break;
        case 17:
          message.bridgeScore = reader.uint64();
          break;
        case 18:
          message.noveltyScore = reader.uint64();
          break;
        case 19:
          message.patronageAmount = reader.string();
          break;
        case 20:
          message.patronageExpiryBlock = reader.uint64();
          break;
        case 21:
          message.stratum = reader.string();
          break;
        case 22:
          message.maturity = reader.string();
          break;
        case 23:
          message.incomingCitationCount = reader.uint64();
          break;
        case 24:
          message.claimType = reader.int32() as any;
          break;
        case 25:
          message.outgoingRelations.push(FactRelation.decode(reader, reader.uint32()));
          break;
        case 26:
          message.incomingRelations.push(FactRelation.decode(reader, reader.uint32()));
          break;
        case 27:
          message.structure = ClaimStructure.decode(reader, reader.uint32());
          break;
        case 28:
          message.canonicalForm = reader.string();
          break;
        case 29:
          message.canonicalHash = reader.string();
          break;
        case 30:
          message.fitnessScore = reader.uint64();
          break;
        case 31:
          message.fitnessUpdatedBlock = reader.uint64();
          break;
        case 32:
          message.queryCount = reader.uint64();
          break;
        case 33:
          message.queryCountEpoch = reader.uint64();
          break;
        case 34:
          message.epochBorn = reader.uint64();
          break;
        case 35:
          message.energy = reader.uint64();
          break;
        case 36:
          message.energyCap = reader.uint64();
          break;
        case 37:
          message.energyLastUpdated = reader.uint64();
          break;
        case 38:
          message.atRiskSinceEpoch = reader.uint64();
          break;
        case 39:
          message.nicheKey = reader.string();
          break;
        case 40:
          message.nicheLeader = reader.bool();
          break;
        case 41:
          message.nicheRank = reader.uint64();
          break;
        case 42:
          message.nicheSize = reader.uint64();
          break;
        case 43:
          message.competitionTax = reader.uint64();
          break;
        case 44:
          message.parentFactId = reader.string();
          break;
        case 45:
          message.childFactIds.push(reader.string());
          break;
        case 46:
          message.lineageDepth = reader.uint64();
          break;
        case 47:
          message.progenyCount = reader.uint64();
          break;
        case 48:
          message.lineageRootId = reader.string();
          break;
        case 49:
          message.commonKnowledgeMatch = reader.bool();
          break;
        case 60:
          message.satisfactionUp = reader.uint64();
          break;
        case 61:
          message.satisfactionDown = reader.uint64();
          break;
        case 62:
          message.satisfactionUpEpoch = reader.uint64();
          break;
        case 63:
          message.satisfactionDownEpoch = reader.uint64();
          break;
        case 64:
          message.axiomDistance = reader.uint32();
          break;
        case 65:
          message.dependencyConfidenceFloor = reader.uint64();
          break;
        case 66:
          message.methodId = reader.string();
          break;
        case 67:
          message.corroborationCount = reader.uint64();
          break;
        case 68:
          message.lastCorroboratedBlock = reader.uint64();
          break;
        case 69:
          message.reasoningTrace = reader.string();
          break;
        case 70:
          message.submitterCalibrationSnapshotBps = reader.uint64();
          break;
        case 71:
          message.trainingRevenueEarned = reader.string();
          break;
        case 72:
          message.trainingRevenueEarnedRecent = reader.string();
          break;
        case 73:
          message.revenueClawbackBlock = reader.uint64();
          break;
        case 74:
          message.probeInvitedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Fact>): Fact {
    const message = createBaseFact();
    message.id = object.id ?? "";
    message.content = object.content ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    message.confidence = object.confidence !== undefined && object.confidence !== null ? BigInt(object.confidence.toString()) : BigInt(0);
    message.submitter = object.submitter ?? "";
    message.submittedAtBlock = object.submittedAtBlock !== undefined && object.submittedAtBlock !== null ? BigInt(object.submittedAtBlock.toString()) : BigInt(0);
    message.verifiedAtBlock = object.verifiedAtBlock !== undefined && object.verifiedAtBlock !== null ? BigInt(object.verifiedAtBlock.toString()) : BigInt(0);
    message.citationCount = object.citationCount !== undefined && object.citationCount !== null ? BigInt(object.citationCount.toString()) : BigInt(0);
    message.fundamentality = object.fundamentality !== undefined && object.fundamentality !== null ? BigInt(object.fundamentality.toString()) : BigInt(0);
    message.references = object.references?.map(e => e) || [];
    message.status = object.status ?? 0;
    message.claimId = object.claimId ?? "";
    message.reverificationBlock = object.reverificationBlock !== undefined && object.reverificationBlock !== null ? BigInt(object.reverificationBlock.toString()) : BigInt(0);
    message.lastVerifiedBlock = object.lastVerifiedBlock !== undefined && object.lastVerifiedBlock !== null ? BigInt(object.lastVerifiedBlock.toString()) : BigInt(0);
    message.challengeWindowEnd = object.challengeWindowEnd !== undefined && object.challengeWindowEnd !== null ? BigInt(object.challengeWindowEnd.toString()) : BigInt(0);
    message.bridgeScore = object.bridgeScore !== undefined && object.bridgeScore !== null ? BigInt(object.bridgeScore.toString()) : BigInt(0);
    message.noveltyScore = object.noveltyScore !== undefined && object.noveltyScore !== null ? BigInt(object.noveltyScore.toString()) : BigInt(0);
    message.patronageAmount = object.patronageAmount ?? "";
    message.patronageExpiryBlock = object.patronageExpiryBlock !== undefined && object.patronageExpiryBlock !== null ? BigInt(object.patronageExpiryBlock.toString()) : BigInt(0);
    message.stratum = object.stratum ?? "";
    message.maturity = object.maturity ?? "";
    message.incomingCitationCount = object.incomingCitationCount !== undefined && object.incomingCitationCount !== null ? BigInt(object.incomingCitationCount.toString()) : BigInt(0);
    message.claimType = object.claimType ?? 0;
    message.outgoingRelations = object.outgoingRelations?.map(e => FactRelation.fromPartial(e)) || [];
    message.incomingRelations = object.incomingRelations?.map(e => FactRelation.fromPartial(e)) || [];
    message.structure = object.structure !== undefined && object.structure !== null ? ClaimStructure.fromPartial(object.structure) : undefined;
    message.canonicalForm = object.canonicalForm ?? "";
    message.canonicalHash = object.canonicalHash ?? "";
    message.fitnessScore = object.fitnessScore !== undefined && object.fitnessScore !== null ? BigInt(object.fitnessScore.toString()) : BigInt(0);
    message.fitnessUpdatedBlock = object.fitnessUpdatedBlock !== undefined && object.fitnessUpdatedBlock !== null ? BigInt(object.fitnessUpdatedBlock.toString()) : BigInt(0);
    message.queryCount = object.queryCount !== undefined && object.queryCount !== null ? BigInt(object.queryCount.toString()) : BigInt(0);
    message.queryCountEpoch = object.queryCountEpoch !== undefined && object.queryCountEpoch !== null ? BigInt(object.queryCountEpoch.toString()) : BigInt(0);
    message.epochBorn = object.epochBorn !== undefined && object.epochBorn !== null ? BigInt(object.epochBorn.toString()) : BigInt(0);
    message.energy = object.energy !== undefined && object.energy !== null ? BigInt(object.energy.toString()) : BigInt(0);
    message.energyCap = object.energyCap !== undefined && object.energyCap !== null ? BigInt(object.energyCap.toString()) : BigInt(0);
    message.energyLastUpdated = object.energyLastUpdated !== undefined && object.energyLastUpdated !== null ? BigInt(object.energyLastUpdated.toString()) : BigInt(0);
    message.atRiskSinceEpoch = object.atRiskSinceEpoch !== undefined && object.atRiskSinceEpoch !== null ? BigInt(object.atRiskSinceEpoch.toString()) : BigInt(0);
    message.nicheKey = object.nicheKey ?? "";
    message.nicheLeader = object.nicheLeader ?? false;
    message.nicheRank = object.nicheRank !== undefined && object.nicheRank !== null ? BigInt(object.nicheRank.toString()) : BigInt(0);
    message.nicheSize = object.nicheSize !== undefined && object.nicheSize !== null ? BigInt(object.nicheSize.toString()) : BigInt(0);
    message.competitionTax = object.competitionTax !== undefined && object.competitionTax !== null ? BigInt(object.competitionTax.toString()) : BigInt(0);
    message.parentFactId = object.parentFactId ?? "";
    message.childFactIds = object.childFactIds?.map(e => e) || [];
    message.lineageDepth = object.lineageDepth !== undefined && object.lineageDepth !== null ? BigInt(object.lineageDepth.toString()) : BigInt(0);
    message.progenyCount = object.progenyCount !== undefined && object.progenyCount !== null ? BigInt(object.progenyCount.toString()) : BigInt(0);
    message.lineageRootId = object.lineageRootId ?? "";
    message.commonKnowledgeMatch = object.commonKnowledgeMatch ?? false;
    message.satisfactionUp = object.satisfactionUp !== undefined && object.satisfactionUp !== null ? BigInt(object.satisfactionUp.toString()) : BigInt(0);
    message.satisfactionDown = object.satisfactionDown !== undefined && object.satisfactionDown !== null ? BigInt(object.satisfactionDown.toString()) : BigInt(0);
    message.satisfactionUpEpoch = object.satisfactionUpEpoch !== undefined && object.satisfactionUpEpoch !== null ? BigInt(object.satisfactionUpEpoch.toString()) : BigInt(0);
    message.satisfactionDownEpoch = object.satisfactionDownEpoch !== undefined && object.satisfactionDownEpoch !== null ? BigInt(object.satisfactionDownEpoch.toString()) : BigInt(0);
    message.axiomDistance = object.axiomDistance ?? 0;
    message.dependencyConfidenceFloor = object.dependencyConfidenceFloor !== undefined && object.dependencyConfidenceFloor !== null ? BigInt(object.dependencyConfidenceFloor.toString()) : BigInt(0);
    message.methodId = object.methodId ?? "";
    message.corroborationCount = object.corroborationCount !== undefined && object.corroborationCount !== null ? BigInt(object.corroborationCount.toString()) : BigInt(0);
    message.lastCorroboratedBlock = object.lastCorroboratedBlock !== undefined && object.lastCorroboratedBlock !== null ? BigInt(object.lastCorroboratedBlock.toString()) : BigInt(0);
    message.reasoningTrace = object.reasoningTrace ?? "";
    message.submitterCalibrationSnapshotBps = object.submitterCalibrationSnapshotBps !== undefined && object.submitterCalibrationSnapshotBps !== null ? BigInt(object.submitterCalibrationSnapshotBps.toString()) : BigInt(0);
    message.trainingRevenueEarned = object.trainingRevenueEarned ?? "";
    message.trainingRevenueEarnedRecent = object.trainingRevenueEarnedRecent ?? "";
    message.revenueClawbackBlock = object.revenueClawbackBlock !== undefined && object.revenueClawbackBlock !== null ? BigInt(object.revenueClawbackBlock.toString()) : BigInt(0);
    message.probeInvitedAtBlock = object.probeInvitedAtBlock !== undefined && object.probeInvitedAtBlock !== null ? BigInt(object.probeInvitedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseTokenizerSpec(): TokenizerSpec {
  return {
    version: BigInt(0),
    ratifiedAtBlock: BigInt(0),
    methodTokenPrefix: "",
    inferenceTokenPrefix: "",
    relationTokenPrefix: "",
    factStatusTokenPrefix: "",
    tierTokenPrefix: "",
    factBeginToken: "",
    factEndToken: "",
    reasoningBeginToken: "",
    reasoningEndToken: "",
    supportBeginToken: "",
    supportEndToken: "",
    disproofMarkerToken: "",
    canonicalSerialisationVersion: BigInt(0)
  };
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
export const TokenizerSpec = {
  typeUrl: "/zerone.knowledge.v1.TokenizerSpec",
  encode(message: TokenizerSpec, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.version !== BigInt(0)) {
      writer.uint32(8).uint64(message.version);
    }
    if (message.ratifiedAtBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.ratifiedAtBlock);
    }
    if (message.methodTokenPrefix !== "") {
      writer.uint32(26).string(message.methodTokenPrefix);
    }
    if (message.inferenceTokenPrefix !== "") {
      writer.uint32(34).string(message.inferenceTokenPrefix);
    }
    if (message.relationTokenPrefix !== "") {
      writer.uint32(42).string(message.relationTokenPrefix);
    }
    if (message.factStatusTokenPrefix !== "") {
      writer.uint32(50).string(message.factStatusTokenPrefix);
    }
    if (message.tierTokenPrefix !== "") {
      writer.uint32(58).string(message.tierTokenPrefix);
    }
    if (message.factBeginToken !== "") {
      writer.uint32(66).string(message.factBeginToken);
    }
    if (message.factEndToken !== "") {
      writer.uint32(74).string(message.factEndToken);
    }
    if (message.reasoningBeginToken !== "") {
      writer.uint32(82).string(message.reasoningBeginToken);
    }
    if (message.reasoningEndToken !== "") {
      writer.uint32(90).string(message.reasoningEndToken);
    }
    if (message.supportBeginToken !== "") {
      writer.uint32(98).string(message.supportBeginToken);
    }
    if (message.supportEndToken !== "") {
      writer.uint32(106).string(message.supportEndToken);
    }
    if (message.disproofMarkerToken !== "") {
      writer.uint32(114).string(message.disproofMarkerToken);
    }
    if (message.canonicalSerialisationVersion !== BigInt(0)) {
      writer.uint32(120).uint64(message.canonicalSerialisationVersion);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TokenizerSpec {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTokenizerSpec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.version = reader.uint64();
          break;
        case 2:
          message.ratifiedAtBlock = reader.uint64();
          break;
        case 3:
          message.methodTokenPrefix = reader.string();
          break;
        case 4:
          message.inferenceTokenPrefix = reader.string();
          break;
        case 5:
          message.relationTokenPrefix = reader.string();
          break;
        case 6:
          message.factStatusTokenPrefix = reader.string();
          break;
        case 7:
          message.tierTokenPrefix = reader.string();
          break;
        case 8:
          message.factBeginToken = reader.string();
          break;
        case 9:
          message.factEndToken = reader.string();
          break;
        case 10:
          message.reasoningBeginToken = reader.string();
          break;
        case 11:
          message.reasoningEndToken = reader.string();
          break;
        case 12:
          message.supportBeginToken = reader.string();
          break;
        case 13:
          message.supportEndToken = reader.string();
          break;
        case 14:
          message.disproofMarkerToken = reader.string();
          break;
        case 15:
          message.canonicalSerialisationVersion = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TokenizerSpec>): TokenizerSpec {
    const message = createBaseTokenizerSpec();
    message.version = object.version !== undefined && object.version !== null ? BigInt(object.version.toString()) : BigInt(0);
    message.ratifiedAtBlock = object.ratifiedAtBlock !== undefined && object.ratifiedAtBlock !== null ? BigInt(object.ratifiedAtBlock.toString()) : BigInt(0);
    message.methodTokenPrefix = object.methodTokenPrefix ?? "";
    message.inferenceTokenPrefix = object.inferenceTokenPrefix ?? "";
    message.relationTokenPrefix = object.relationTokenPrefix ?? "";
    message.factStatusTokenPrefix = object.factStatusTokenPrefix ?? "";
    message.tierTokenPrefix = object.tierTokenPrefix ?? "";
    message.factBeginToken = object.factBeginToken ?? "";
    message.factEndToken = object.factEndToken ?? "";
    message.reasoningBeginToken = object.reasoningBeginToken ?? "";
    message.reasoningEndToken = object.reasoningEndToken ?? "";
    message.supportBeginToken = object.supportBeginToken ?? "";
    message.supportEndToken = object.supportEndToken ?? "";
    message.disproofMarkerToken = object.disproofMarkerToken ?? "";
    message.canonicalSerialisationVersion = object.canonicalSerialisationVersion !== undefined && object.canonicalSerialisationVersion !== null ? BigInt(object.canonicalSerialisationVersion.toString()) : BigInt(0);
    return message;
  }
};
function createBaseTrainingPipeline(): TrainingPipeline {
  return {
    id: "",
    operatorAddress: "",
    corpusSnapshotHeight: BigInt(0),
    tokenizerVersion: BigInt(0),
    methodologySetVersion: BigInt(0),
    recipeHash: "",
    description: "",
    status: "",
    declaredAtBlock: BigInt(0),
    completedAtBlock: BigInt(0),
    corpusFilter: ""
  };
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
export const TrainingPipeline = {
  typeUrl: "/zerone.knowledge.v1.TrainingPipeline",
  encode(message: TrainingPipeline, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.operatorAddress !== "") {
      writer.uint32(18).string(message.operatorAddress);
    }
    if (message.corpusSnapshotHeight !== BigInt(0)) {
      writer.uint32(24).uint64(message.corpusSnapshotHeight);
    }
    if (message.tokenizerVersion !== BigInt(0)) {
      writer.uint32(32).uint64(message.tokenizerVersion);
    }
    if (message.methodologySetVersion !== BigInt(0)) {
      writer.uint32(40).uint64(message.methodologySetVersion);
    }
    if (message.recipeHash !== "") {
      writer.uint32(50).string(message.recipeHash);
    }
    if (message.description !== "") {
      writer.uint32(58).string(message.description);
    }
    if (message.status !== "") {
      writer.uint32(66).string(message.status);
    }
    if (message.declaredAtBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.declaredAtBlock);
    }
    if (message.completedAtBlock !== BigInt(0)) {
      writer.uint32(80).uint64(message.completedAtBlock);
    }
    if (message.corpusFilter !== "") {
      writer.uint32(90).string(message.corpusFilter);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TrainingPipeline {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTrainingPipeline();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.operatorAddress = reader.string();
          break;
        case 3:
          message.corpusSnapshotHeight = reader.uint64();
          break;
        case 4:
          message.tokenizerVersion = reader.uint64();
          break;
        case 5:
          message.methodologySetVersion = reader.uint64();
          break;
        case 6:
          message.recipeHash = reader.string();
          break;
        case 7:
          message.description = reader.string();
          break;
        case 8:
          message.status = reader.string();
          break;
        case 9:
          message.declaredAtBlock = reader.uint64();
          break;
        case 10:
          message.completedAtBlock = reader.uint64();
          break;
        case 11:
          message.corpusFilter = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TrainingPipeline>): TrainingPipeline {
    const message = createBaseTrainingPipeline();
    message.id = object.id ?? "";
    message.operatorAddress = object.operatorAddress ?? "";
    message.corpusSnapshotHeight = object.corpusSnapshotHeight !== undefined && object.corpusSnapshotHeight !== null ? BigInt(object.corpusSnapshotHeight.toString()) : BigInt(0);
    message.tokenizerVersion = object.tokenizerVersion !== undefined && object.tokenizerVersion !== null ? BigInt(object.tokenizerVersion.toString()) : BigInt(0);
    message.methodologySetVersion = object.methodologySetVersion !== undefined && object.methodologySetVersion !== null ? BigInt(object.methodologySetVersion.toString()) : BigInt(0);
    message.recipeHash = object.recipeHash ?? "";
    message.description = object.description ?? "";
    message.status = object.status ?? "";
    message.declaredAtBlock = object.declaredAtBlock !== undefined && object.declaredAtBlock !== null ? BigInt(object.declaredAtBlock.toString()) : BigInt(0);
    message.completedAtBlock = object.completedAtBlock !== undefined && object.completedAtBlock !== null ? BigInt(object.completedAtBlock.toString()) : BigInt(0);
    message.corpusFilter = object.corpusFilter ?? "";
    return message;
  }
};
function createBaseModelCard(): ModelCard {
  return {
    id: "",
    name: "",
    pipelineId: "",
    deploymentAddress: "",
    createdAtBlock: BigInt(0),
    parameterCount: BigInt(0),
    route: "",
    baseModel: "",
    ownerAddress: "",
    evalAcceptanceRateBps: BigInt(0),
    evalCorroborationRateBps: BigInt(0),
    evalSampleSize: BigInt(0),
    specialisedMethodId: "",
    active: false,
    retiredAtBlock: BigInt(0),
    retiredReason: "",
    predecessorModelId: ""
  };
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
export const ModelCard = {
  typeUrl: "/zerone.knowledge.v1.ModelCard",
  encode(message: ModelCard, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.pipelineId !== "") {
      writer.uint32(26).string(message.pipelineId);
    }
    if (message.deploymentAddress !== "") {
      writer.uint32(34).string(message.deploymentAddress);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.createdAtBlock);
    }
    if (message.parameterCount !== BigInt(0)) {
      writer.uint32(48).uint64(message.parameterCount);
    }
    if (message.route !== "") {
      writer.uint32(58).string(message.route);
    }
    if (message.baseModel !== "") {
      writer.uint32(66).string(message.baseModel);
    }
    if (message.ownerAddress !== "") {
      writer.uint32(74).string(message.ownerAddress);
    }
    if (message.evalAcceptanceRateBps !== BigInt(0)) {
      writer.uint32(80).uint64(message.evalAcceptanceRateBps);
    }
    if (message.evalCorroborationRateBps !== BigInt(0)) {
      writer.uint32(88).uint64(message.evalCorroborationRateBps);
    }
    if (message.evalSampleSize !== BigInt(0)) {
      writer.uint32(96).uint64(message.evalSampleSize);
    }
    if (message.specialisedMethodId !== "") {
      writer.uint32(106).string(message.specialisedMethodId);
    }
    if (message.active === true) {
      writer.uint32(112).bool(message.active);
    }
    if (message.retiredAtBlock !== BigInt(0)) {
      writer.uint32(120).uint64(message.retiredAtBlock);
    }
    if (message.retiredReason !== "") {
      writer.uint32(130).string(message.retiredReason);
    }
    if (message.predecessorModelId !== "") {
      writer.uint32(138).string(message.predecessorModelId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ModelCard {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseModelCard();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.pipelineId = reader.string();
          break;
        case 4:
          message.deploymentAddress = reader.string();
          break;
        case 5:
          message.createdAtBlock = reader.uint64();
          break;
        case 6:
          message.parameterCount = reader.uint64();
          break;
        case 7:
          message.route = reader.string();
          break;
        case 8:
          message.baseModel = reader.string();
          break;
        case 9:
          message.ownerAddress = reader.string();
          break;
        case 10:
          message.evalAcceptanceRateBps = reader.uint64();
          break;
        case 11:
          message.evalCorroborationRateBps = reader.uint64();
          break;
        case 12:
          message.evalSampleSize = reader.uint64();
          break;
        case 13:
          message.specialisedMethodId = reader.string();
          break;
        case 14:
          message.active = reader.bool();
          break;
        case 15:
          message.retiredAtBlock = reader.uint64();
          break;
        case 16:
          message.retiredReason = reader.string();
          break;
        case 17:
          message.predecessorModelId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ModelCard>): ModelCard {
    const message = createBaseModelCard();
    message.id = object.id ?? "";
    message.name = object.name ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.deploymentAddress = object.deploymentAddress ?? "";
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.parameterCount = object.parameterCount !== undefined && object.parameterCount !== null ? BigInt(object.parameterCount.toString()) : BigInt(0);
    message.route = object.route ?? "";
    message.baseModel = object.baseModel ?? "";
    message.ownerAddress = object.ownerAddress ?? "";
    message.evalAcceptanceRateBps = object.evalAcceptanceRateBps !== undefined && object.evalAcceptanceRateBps !== null ? BigInt(object.evalAcceptanceRateBps.toString()) : BigInt(0);
    message.evalCorroborationRateBps = object.evalCorroborationRateBps !== undefined && object.evalCorroborationRateBps !== null ? BigInt(object.evalCorroborationRateBps.toString()) : BigInt(0);
    message.evalSampleSize = object.evalSampleSize !== undefined && object.evalSampleSize !== null ? BigInt(object.evalSampleSize.toString()) : BigInt(0);
    message.specialisedMethodId = object.specialisedMethodId ?? "";
    message.active = object.active ?? false;
    message.retiredAtBlock = object.retiredAtBlock !== undefined && object.retiredAtBlock !== null ? BigInt(object.retiredAtBlock.toString()) : BigInt(0);
    message.retiredReason = object.retiredReason ?? "";
    message.predecessorModelId = object.predecessorModelId ?? "";
    return message;
  }
};
function createBaseTrainingAttestation(): TrainingAttestation {
  return {
    pipelineId: "",
    attesterAddress: "",
    flopsEstimate: BigInt(0),
    wallclockSeconds: BigInt(0),
    completedAtBlock: BigInt(0),
    evalHash: "",
    signature: "",
    notes: ""
  };
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
export const TrainingAttestation = {
  typeUrl: "/zerone.knowledge.v1.TrainingAttestation",
  encode(message: TrainingAttestation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.pipelineId !== "") {
      writer.uint32(10).string(message.pipelineId);
    }
    if (message.attesterAddress !== "") {
      writer.uint32(18).string(message.attesterAddress);
    }
    if (message.flopsEstimate !== BigInt(0)) {
      writer.uint32(24).uint64(message.flopsEstimate);
    }
    if (message.wallclockSeconds !== BigInt(0)) {
      writer.uint32(32).uint64(message.wallclockSeconds);
    }
    if (message.completedAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.completedAtBlock);
    }
    if (message.evalHash !== "") {
      writer.uint32(50).string(message.evalHash);
    }
    if (message.signature !== "") {
      writer.uint32(58).string(message.signature);
    }
    if (message.notes !== "") {
      writer.uint32(66).string(message.notes);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TrainingAttestation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTrainingAttestation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pipelineId = reader.string();
          break;
        case 2:
          message.attesterAddress = reader.string();
          break;
        case 3:
          message.flopsEstimate = reader.uint64();
          break;
        case 4:
          message.wallclockSeconds = reader.uint64();
          break;
        case 5:
          message.completedAtBlock = reader.uint64();
          break;
        case 6:
          message.evalHash = reader.string();
          break;
        case 7:
          message.signature = reader.string();
          break;
        case 8:
          message.notes = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TrainingAttestation>): TrainingAttestation {
    const message = createBaseTrainingAttestation();
    message.pipelineId = object.pipelineId ?? "";
    message.attesterAddress = object.attesterAddress ?? "";
    message.flopsEstimate = object.flopsEstimate !== undefined && object.flopsEstimate !== null ? BigInt(object.flopsEstimate.toString()) : BigInt(0);
    message.wallclockSeconds = object.wallclockSeconds !== undefined && object.wallclockSeconds !== null ? BigInt(object.wallclockSeconds.toString()) : BigInt(0);
    message.completedAtBlock = object.completedAtBlock !== undefined && object.completedAtBlock !== null ? BigInt(object.completedAtBlock.toString()) : BigInt(0);
    message.evalHash = object.evalHash ?? "";
    message.signature = object.signature ?? "";
    message.notes = object.notes ?? "";
    return message;
  }
};
function createBaseContributionRecord(): ContributionRecord {
  return {
    modelId: "",
    factIds: [],
    attributedBy: "",
    attributedAtBlock: BigInt(0),
    totalWeight: BigInt(0),
    computedTvw: BigInt(0),
    rejectedCommitmentCount: 0,
    perFactCalibrationBps: []
  };
}
/**
 * ContributionRecord attributes the facts a training run consumed. Enables
 * contributor-share rewards, reproducibility audit, and analysis of whose
 * data disproportionately trained a given model (Route B Wave 3b).
 * @name ContributionRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContributionRecord
 */
export const ContributionRecord = {
  typeUrl: "/zerone.knowledge.v1.ContributionRecord",
  encode(message: ContributionRecord, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.modelId !== "") {
      writer.uint32(10).string(message.modelId);
    }
    for (const v of message.factIds) {
      writer.uint32(18).string(v!);
    }
    if (message.attributedBy !== "") {
      writer.uint32(26).string(message.attributedBy);
    }
    if (message.attributedAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.attributedAtBlock);
    }
    if (message.totalWeight !== BigInt(0)) {
      writer.uint32(40).uint64(message.totalWeight);
    }
    if (message.computedTvw !== BigInt(0)) {
      writer.uint32(48).uint64(message.computedTvw);
    }
    if (message.rejectedCommitmentCount !== 0) {
      writer.uint32(56).uint32(message.rejectedCommitmentCount);
    }
    writer.uint32(66).fork();
    for (const v of message.perFactCalibrationBps) {
      writer.uint64(v);
    }
    writer.ldelim();
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ContributionRecord {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseContributionRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.modelId = reader.string();
          break;
        case 2:
          message.factIds.push(reader.string());
          break;
        case 3:
          message.attributedBy = reader.string();
          break;
        case 4:
          message.attributedAtBlock = reader.uint64();
          break;
        case 5:
          message.totalWeight = reader.uint64();
          break;
        case 6:
          message.computedTvw = reader.uint64();
          break;
        case 7:
          message.rejectedCommitmentCount = reader.uint32();
          break;
        case 8:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.perFactCalibrationBps.push(reader.uint64());
            }
          } else {
            message.perFactCalibrationBps.push(reader.uint64());
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ContributionRecord>): ContributionRecord {
    const message = createBaseContributionRecord();
    message.modelId = object.modelId ?? "";
    message.factIds = object.factIds?.map(e => e) || [];
    message.attributedBy = object.attributedBy ?? "";
    message.attributedAtBlock = object.attributedAtBlock !== undefined && object.attributedAtBlock !== null ? BigInt(object.attributedAtBlock.toString()) : BigInt(0);
    message.totalWeight = object.totalWeight !== undefined && object.totalWeight !== null ? BigInt(object.totalWeight.toString()) : BigInt(0);
    message.computedTvw = object.computedTvw !== undefined && object.computedTvw !== null ? BigInt(object.computedTvw.toString()) : BigInt(0);
    message.rejectedCommitmentCount = object.rejectedCommitmentCount ?? 0;
    message.perFactCalibrationBps = object.perFactCalibrationBps?.map(e => BigInt(e.toString())) || [];
    return message;
  }
};
function createBaseAugmentationBounty(): AugmentationBounty {
  return {
    id: "",
    sponsorAddress: "",
    targetFactId: "",
    rewardPerVariant: BigInt(0),
    maxVariants: 0,
    acceptedVariants: 0,
    createdAtBlock: BigInt(0),
    expiresAtBlock: BigInt(0),
    active: false,
    description: "",
    escrowLocked: "",
    methodologyId: ""
  };
}
/**
 * AugmentationBounty is an open offer to produce variant formulations of a
 * target fact. Sponsors lock a reward *in escrow* (Wave 4); payout routes
 * through a verifier-panel verdict, never through the sponsor directly.
 * @name AugmentationBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.AugmentationBounty
 */
export const AugmentationBounty = {
  typeUrl: "/zerone.knowledge.v1.AugmentationBounty",
  encode(message: AugmentationBounty, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.sponsorAddress !== "") {
      writer.uint32(18).string(message.sponsorAddress);
    }
    if (message.targetFactId !== "") {
      writer.uint32(26).string(message.targetFactId);
    }
    if (message.rewardPerVariant !== BigInt(0)) {
      writer.uint32(32).uint64(message.rewardPerVariant);
    }
    if (message.maxVariants !== 0) {
      writer.uint32(40).uint32(message.maxVariants);
    }
    if (message.acceptedVariants !== 0) {
      writer.uint32(48).uint32(message.acceptedVariants);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(56).uint64(message.createdAtBlock);
    }
    if (message.expiresAtBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.expiresAtBlock);
    }
    if (message.active === true) {
      writer.uint32(72).bool(message.active);
    }
    if (message.description !== "") {
      writer.uint32(82).string(message.description);
    }
    if (message.escrowLocked !== "") {
      writer.uint32(90).string(message.escrowLocked);
    }
    if (message.methodologyId !== "") {
      writer.uint32(98).string(message.methodologyId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AugmentationBounty {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAugmentationBounty();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.sponsorAddress = reader.string();
          break;
        case 3:
          message.targetFactId = reader.string();
          break;
        case 4:
          message.rewardPerVariant = reader.uint64();
          break;
        case 5:
          message.maxVariants = reader.uint32();
          break;
        case 6:
          message.acceptedVariants = reader.uint32();
          break;
        case 7:
          message.createdAtBlock = reader.uint64();
          break;
        case 8:
          message.expiresAtBlock = reader.uint64();
          break;
        case 9:
          message.active = reader.bool();
          break;
        case 10:
          message.description = reader.string();
          break;
        case 11:
          message.escrowLocked = reader.string();
          break;
        case 12:
          message.methodologyId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AugmentationBounty>): AugmentationBounty {
    const message = createBaseAugmentationBounty();
    message.id = object.id ?? "";
    message.sponsorAddress = object.sponsorAddress ?? "";
    message.targetFactId = object.targetFactId ?? "";
    message.rewardPerVariant = object.rewardPerVariant !== undefined && object.rewardPerVariant !== null ? BigInt(object.rewardPerVariant.toString()) : BigInt(0);
    message.maxVariants = object.maxVariants ?? 0;
    message.acceptedVariants = object.acceptedVariants ?? 0;
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.expiresAtBlock = object.expiresAtBlock !== undefined && object.expiresAtBlock !== null ? BigInt(object.expiresAtBlock.toString()) : BigInt(0);
    message.active = object.active ?? false;
    message.description = object.description ?? "";
    message.escrowLocked = object.escrowLocked ?? "";
    message.methodologyId = object.methodologyId ?? "";
    return message;
  }
};
function createBaseAugmentation(): Augmentation {
  return {
    id: "",
    bountyId: "",
    originalFactId: "",
    variantContent: "",
    variantReasoningTrace: "",
    submitter: "",
    createdAtBlock: BigInt(0),
    accepted: false,
    acceptedAtBlock: BigInt(0),
    acceptanceNote: "",
    verdict: 0,
    verdictBlock: BigInt(0),
    verdictVoters: [],
    verdictVotes: [],
    sponsorVetoed: false,
    payoutAmount: "",
    verdictVoteStakes: [],
    verdictVoteCalibrationBps: []
  };
}
/**
 * Augmentation is a variant formulation of an original fact. Acceptance
 * requires a verifier-panel verdict (Wave 4) — the sponsor never judges.
 * @name Augmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Augmentation
 */
export const Augmentation = {
  typeUrl: "/zerone.knowledge.v1.Augmentation",
  encode(message: Augmentation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.bountyId !== "") {
      writer.uint32(18).string(message.bountyId);
    }
    if (message.originalFactId !== "") {
      writer.uint32(26).string(message.originalFactId);
    }
    if (message.variantContent !== "") {
      writer.uint32(34).string(message.variantContent);
    }
    if (message.variantReasoningTrace !== "") {
      writer.uint32(42).string(message.variantReasoningTrace);
    }
    if (message.submitter !== "") {
      writer.uint32(50).string(message.submitter);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(56).uint64(message.createdAtBlock);
    }
    if (message.accepted === true) {
      writer.uint32(64).bool(message.accepted);
    }
    if (message.acceptedAtBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.acceptedAtBlock);
    }
    if (message.acceptanceNote !== "") {
      writer.uint32(82).string(message.acceptanceNote);
    }
    if (message.verdict !== 0) {
      writer.uint32(88).int32(message.verdict);
    }
    if (message.verdictBlock !== BigInt(0)) {
      writer.uint32(96).uint64(message.verdictBlock);
    }
    for (const v of message.verdictVoters) {
      writer.uint32(106).string(v!);
    }
    writer.uint32(114).fork();
    for (const v of message.verdictVotes) {
      writer.int32(v);
    }
    writer.ldelim();
    if (message.sponsorVetoed === true) {
      writer.uint32(120).bool(message.sponsorVetoed);
    }
    if (message.payoutAmount !== "") {
      writer.uint32(130).string(message.payoutAmount);
    }
    writer.uint32(138).fork();
    for (const v of message.verdictVoteStakes) {
      writer.uint64(v);
    }
    writer.ldelim();
    writer.uint32(146).fork();
    for (const v of message.verdictVoteCalibrationBps) {
      writer.uint64(v);
    }
    writer.ldelim();
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Augmentation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.bountyId = reader.string();
          break;
        case 3:
          message.originalFactId = reader.string();
          break;
        case 4:
          message.variantContent = reader.string();
          break;
        case 5:
          message.variantReasoningTrace = reader.string();
          break;
        case 6:
          message.submitter = reader.string();
          break;
        case 7:
          message.createdAtBlock = reader.uint64();
          break;
        case 8:
          message.accepted = reader.bool();
          break;
        case 9:
          message.acceptedAtBlock = reader.uint64();
          break;
        case 10:
          message.acceptanceNote = reader.string();
          break;
        case 11:
          message.verdict = reader.int32() as any;
          break;
        case 12:
          message.verdictBlock = reader.uint64();
          break;
        case 13:
          message.verdictVoters.push(reader.string());
          break;
        case 14:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.verdictVotes.push(reader.int32() as any);
            }
          } else {
            message.verdictVotes.push(reader.int32() as any);
          }
          break;
        case 15:
          message.sponsorVetoed = reader.bool();
          break;
        case 16:
          message.payoutAmount = reader.string();
          break;
        case 17:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.verdictVoteStakes.push(reader.uint64());
            }
          } else {
            message.verdictVoteStakes.push(reader.uint64());
          }
          break;
        case 18:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.verdictVoteCalibrationBps.push(reader.uint64());
            }
          } else {
            message.verdictVoteCalibrationBps.push(reader.uint64());
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Augmentation>): Augmentation {
    const message = createBaseAugmentation();
    message.id = object.id ?? "";
    message.bountyId = object.bountyId ?? "";
    message.originalFactId = object.originalFactId ?? "";
    message.variantContent = object.variantContent ?? "";
    message.variantReasoningTrace = object.variantReasoningTrace ?? "";
    message.submitter = object.submitter ?? "";
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.accepted = object.accepted ?? false;
    message.acceptedAtBlock = object.acceptedAtBlock !== undefined && object.acceptedAtBlock !== null ? BigInt(object.acceptedAtBlock.toString()) : BigInt(0);
    message.acceptanceNote = object.acceptanceNote ?? "";
    message.verdict = object.verdict ?? 0;
    message.verdictBlock = object.verdictBlock !== undefined && object.verdictBlock !== null ? BigInt(object.verdictBlock.toString()) : BigInt(0);
    message.verdictVoters = object.verdictVoters?.map(e => e) || [];
    message.verdictVotes = object.verdictVotes?.map(e => e) || [];
    message.sponsorVetoed = object.sponsorVetoed ?? false;
    message.payoutAmount = object.payoutAmount ?? "";
    message.verdictVoteStakes = object.verdictVoteStakes?.map(e => BigInt(e.toString())) || [];
    message.verdictVoteCalibrationBps = object.verdictVoteCalibrationBps?.map(e => BigInt(e.toString())) || [];
    return message;
  }
};
function createBaseContributionChallenge(): ContributionChallenge {
  return {
    id: "",
    modelId: "",
    challenger: "",
    disputedFactId: "",
    disputeType: "",
    bond: "",
    createdAtBlock: BigInt(0),
    evidence: "",
    status: "",
    resolvedAtBlock: BigInt(0),
    resolver: "",
    resolutionNote: ""
  };
}
/**
 * ContributionChallenge is a bonded dispute over whether a model's declared
 * ContributionRecord is accurate: a fact submitter asserts under-reporting
 * (the model used their fact but didn't attribute) or over-reporting (the
 * owner listed a colluder's fact that wasn't used). Resolves through the
 * standard PoT verification layer.
 * @name ContributionChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContributionChallenge
 */
export const ContributionChallenge = {
  typeUrl: "/zerone.knowledge.v1.ContributionChallenge",
  encode(message: ContributionChallenge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.modelId !== "") {
      writer.uint32(18).string(message.modelId);
    }
    if (message.challenger !== "") {
      writer.uint32(26).string(message.challenger);
    }
    if (message.disputedFactId !== "") {
      writer.uint32(34).string(message.disputedFactId);
    }
    if (message.disputeType !== "") {
      writer.uint32(42).string(message.disputeType);
    }
    if (message.bond !== "") {
      writer.uint32(50).string(message.bond);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(56).uint64(message.createdAtBlock);
    }
    if (message.evidence !== "") {
      writer.uint32(66).string(message.evidence);
    }
    if (message.status !== "") {
      writer.uint32(74).string(message.status);
    }
    if (message.resolvedAtBlock !== BigInt(0)) {
      writer.uint32(80).uint64(message.resolvedAtBlock);
    }
    if (message.resolver !== "") {
      writer.uint32(90).string(message.resolver);
    }
    if (message.resolutionNote !== "") {
      writer.uint32(98).string(message.resolutionNote);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ContributionChallenge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseContributionChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.modelId = reader.string();
          break;
        case 3:
          message.challenger = reader.string();
          break;
        case 4:
          message.disputedFactId = reader.string();
          break;
        case 5:
          message.disputeType = reader.string();
          break;
        case 6:
          message.bond = reader.string();
          break;
        case 7:
          message.createdAtBlock = reader.uint64();
          break;
        case 8:
          message.evidence = reader.string();
          break;
        case 9:
          message.status = reader.string();
          break;
        case 10:
          message.resolvedAtBlock = reader.uint64();
          break;
        case 11:
          message.resolver = reader.string();
          break;
        case 12:
          message.resolutionNote = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ContributionChallenge>): ContributionChallenge {
    const message = createBaseContributionChallenge();
    message.id = object.id ?? "";
    message.modelId = object.modelId ?? "";
    message.challenger = object.challenger ?? "";
    message.disputedFactId = object.disputedFactId ?? "";
    message.disputeType = object.disputeType ?? "";
    message.bond = object.bond ?? "";
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.evidence = object.evidence ?? "";
    message.status = object.status ?? "";
    message.resolvedAtBlock = object.resolvedAtBlock !== undefined && object.resolvedAtBlock !== null ? BigInt(object.resolvedAtBlock.toString()) : BigInt(0);
    message.resolver = object.resolver ?? "";
    message.resolutionNote = object.resolutionNote ?? "";
    return message;
  }
};
function createBaseTrainingFundDisbursement(): TrainingFundDisbursement {
  return {
    id: "",
    modelId: "",
    pipelineId: "",
    claimant: "",
    claimedAtBlock: BigInt(0),
    totalAmount: "",
    releasedAmount: "",
    vestingAmount: "",
    vestingEndBlock: BigInt(0),
    calibrationScoreAtClaimBps: BigInt(0),
    methodologyDiversityCount: BigInt(0),
    reproducibilityProofPresent: false,
    clawedBackAmount: "",
    clawedBackAtBlock: BigInt(0),
    clawbackReason: ""
  };
}
/**
 * TrainingFundDisbursement is a post-hoc reward to a pipeline whose ModelCard
 * has demonstrated calibration in live deployment. 50% is released at claim
 * time; 50% is held in vesting escrow and clawed back if calibration drops.
 * @name TrainingFundDisbursement
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingFundDisbursement
 */
export const TrainingFundDisbursement = {
  typeUrl: "/zerone.knowledge.v1.TrainingFundDisbursement",
  encode(message: TrainingFundDisbursement, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.modelId !== "") {
      writer.uint32(18).string(message.modelId);
    }
    if (message.pipelineId !== "") {
      writer.uint32(26).string(message.pipelineId);
    }
    if (message.claimant !== "") {
      writer.uint32(34).string(message.claimant);
    }
    if (message.claimedAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.claimedAtBlock);
    }
    if (message.totalAmount !== "") {
      writer.uint32(50).string(message.totalAmount);
    }
    if (message.releasedAmount !== "") {
      writer.uint32(58).string(message.releasedAmount);
    }
    if (message.vestingAmount !== "") {
      writer.uint32(66).string(message.vestingAmount);
    }
    if (message.vestingEndBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.vestingEndBlock);
    }
    if (message.calibrationScoreAtClaimBps !== BigInt(0)) {
      writer.uint32(80).uint64(message.calibrationScoreAtClaimBps);
    }
    if (message.methodologyDiversityCount !== BigInt(0)) {
      writer.uint32(88).uint64(message.methodologyDiversityCount);
    }
    if (message.reproducibilityProofPresent === true) {
      writer.uint32(96).bool(message.reproducibilityProofPresent);
    }
    if (message.clawedBackAmount !== "") {
      writer.uint32(106).string(message.clawedBackAmount);
    }
    if (message.clawedBackAtBlock !== BigInt(0)) {
      writer.uint32(112).uint64(message.clawedBackAtBlock);
    }
    if (message.clawbackReason !== "") {
      writer.uint32(122).string(message.clawbackReason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TrainingFundDisbursement {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTrainingFundDisbursement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.modelId = reader.string();
          break;
        case 3:
          message.pipelineId = reader.string();
          break;
        case 4:
          message.claimant = reader.string();
          break;
        case 5:
          message.claimedAtBlock = reader.uint64();
          break;
        case 6:
          message.totalAmount = reader.string();
          break;
        case 7:
          message.releasedAmount = reader.string();
          break;
        case 8:
          message.vestingAmount = reader.string();
          break;
        case 9:
          message.vestingEndBlock = reader.uint64();
          break;
        case 10:
          message.calibrationScoreAtClaimBps = reader.uint64();
          break;
        case 11:
          message.methodologyDiversityCount = reader.uint64();
          break;
        case 12:
          message.reproducibilityProofPresent = reader.bool();
          break;
        case 13:
          message.clawedBackAmount = reader.string();
          break;
        case 14:
          message.clawedBackAtBlock = reader.uint64();
          break;
        case 15:
          message.clawbackReason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TrainingFundDisbursement>): TrainingFundDisbursement {
    const message = createBaseTrainingFundDisbursement();
    message.id = object.id ?? "";
    message.modelId = object.modelId ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.claimant = object.claimant ?? "";
    message.claimedAtBlock = object.claimedAtBlock !== undefined && object.claimedAtBlock !== null ? BigInt(object.claimedAtBlock.toString()) : BigInt(0);
    message.totalAmount = object.totalAmount ?? "";
    message.releasedAmount = object.releasedAmount ?? "";
    message.vestingAmount = object.vestingAmount ?? "";
    message.vestingEndBlock = object.vestingEndBlock !== undefined && object.vestingEndBlock !== null ? BigInt(object.vestingEndBlock.toString()) : BigInt(0);
    message.calibrationScoreAtClaimBps = object.calibrationScoreAtClaimBps !== undefined && object.calibrationScoreAtClaimBps !== null ? BigInt(object.calibrationScoreAtClaimBps.toString()) : BigInt(0);
    message.methodologyDiversityCount = object.methodologyDiversityCount !== undefined && object.methodologyDiversityCount !== null ? BigInt(object.methodologyDiversityCount.toString()) : BigInt(0);
    message.reproducibilityProofPresent = object.reproducibilityProofPresent ?? false;
    message.clawedBackAmount = object.clawedBackAmount ?? "";
    message.clawedBackAtBlock = object.clawedBackAtBlock !== undefined && object.clawedBackAtBlock !== null ? BigInt(object.clawedBackAtBlock.toString()) : BigInt(0);
    message.clawbackReason = object.clawbackReason ?? "";
    return message;
  }
};
function createBaseAgentMethodStats(): AgentMethodStats {
  return {
    submissions: BigInt(0),
    accepted: BigInt(0),
    rejected: BigInt(0),
    corroborationsEarned: BigInt(0),
    disproven: BigInt(0)
  };
}
/**
 * AgentMethodStats tracks calibration for a single submitter within a single
 * methodology. Populated on round completion and challenge outcomes.
 * @name AgentMethodStats
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.AgentMethodStats
 */
export const AgentMethodStats = {
  typeUrl: "/zerone.knowledge.v1.AgentMethodStats",
  encode(message: AgentMethodStats, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.submissions !== BigInt(0)) {
      writer.uint32(8).uint64(message.submissions);
    }
    if (message.accepted !== BigInt(0)) {
      writer.uint32(16).uint64(message.accepted);
    }
    if (message.rejected !== BigInt(0)) {
      writer.uint32(24).uint64(message.rejected);
    }
    if (message.corroborationsEarned !== BigInt(0)) {
      writer.uint32(32).uint64(message.corroborationsEarned);
    }
    if (message.disproven !== BigInt(0)) {
      writer.uint32(40).uint64(message.disproven);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AgentMethodStats {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAgentMethodStats();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submissions = reader.uint64();
          break;
        case 2:
          message.accepted = reader.uint64();
          break;
        case 3:
          message.rejected = reader.uint64();
          break;
        case 4:
          message.corroborationsEarned = reader.uint64();
          break;
        case 5:
          message.disproven = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AgentMethodStats>): AgentMethodStats {
    const message = createBaseAgentMethodStats();
    message.submissions = object.submissions !== undefined && object.submissions !== null ? BigInt(object.submissions.toString()) : BigInt(0);
    message.accepted = object.accepted !== undefined && object.accepted !== null ? BigInt(object.accepted.toString()) : BigInt(0);
    message.rejected = object.rejected !== undefined && object.rejected !== null ? BigInt(object.rejected.toString()) : BigInt(0);
    message.corroborationsEarned = object.corroborationsEarned !== undefined && object.corroborationsEarned !== null ? BigInt(object.corroborationsEarned.toString()) : BigInt(0);
    message.disproven = object.disproven !== undefined && object.disproven !== null ? BigInt(object.disproven.toString()) : BigInt(0);
    return message;
  }
};
function createBaseAgentCalibration_PerMethodEntry(): AgentCalibration_PerMethodEntry {
  return {
    key: "",
    value: undefined
  };
}
/**
 * @name AgentCalibration_PerMethodEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.undefined
 */
export const AgentCalibration_PerMethodEntry = {
  encode(message: AgentCalibration_PerMethodEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== undefined) {
      AgentMethodStats.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AgentCalibration_PerMethodEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAgentCalibration_PerMethodEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = AgentMethodStats.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AgentCalibration_PerMethodEntry>): AgentCalibration_PerMethodEntry {
    const message = createBaseAgentCalibration_PerMethodEntry();
    message.key = object.key ?? "";
    message.value = object.value !== undefined && object.value !== null ? AgentMethodStats.fromPartial(object.value) : undefined;
    return message;
  }
};
function createBaseAgentCalibration(): AgentCalibration {
  return {
    address: "",
    accountType: "",
    totalSubmissions: BigInt(0),
    accepted: BigInt(0),
    rejected: BigInt(0),
    malformed: BigInt(0),
    inconclusive: BigInt(0),
    corroborationsEarned: BigInt(0),
    disprovenCount: BigInt(0),
    challengesIssued: BigInt(0),
    challengesSucceeded: BigInt(0),
    challengesFailed: BigInt(0),
    firstSubmissionBlock: BigInt(0),
    lastSubmissionBlock: BigInt(0),
    perMethod: {},
    calibrationScoreBps: BigInt(0),
    lastUpdatedBlock: BigInt(0)
  };
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
export const AgentCalibration = {
  typeUrl: "/zerone.knowledge.v1.AgentCalibration",
  encode(message: AgentCalibration, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.accountType !== "") {
      writer.uint32(18).string(message.accountType);
    }
    if (message.totalSubmissions !== BigInt(0)) {
      writer.uint32(24).uint64(message.totalSubmissions);
    }
    if (message.accepted !== BigInt(0)) {
      writer.uint32(32).uint64(message.accepted);
    }
    if (message.rejected !== BigInt(0)) {
      writer.uint32(40).uint64(message.rejected);
    }
    if (message.malformed !== BigInt(0)) {
      writer.uint32(48).uint64(message.malformed);
    }
    if (message.inconclusive !== BigInt(0)) {
      writer.uint32(56).uint64(message.inconclusive);
    }
    if (message.corroborationsEarned !== BigInt(0)) {
      writer.uint32(64).uint64(message.corroborationsEarned);
    }
    if (message.disprovenCount !== BigInt(0)) {
      writer.uint32(72).uint64(message.disprovenCount);
    }
    if (message.challengesIssued !== BigInt(0)) {
      writer.uint32(80).uint64(message.challengesIssued);
    }
    if (message.challengesSucceeded !== BigInt(0)) {
      writer.uint32(88).uint64(message.challengesSucceeded);
    }
    if (message.challengesFailed !== BigInt(0)) {
      writer.uint32(96).uint64(message.challengesFailed);
    }
    if (message.firstSubmissionBlock !== BigInt(0)) {
      writer.uint32(104).uint64(message.firstSubmissionBlock);
    }
    if (message.lastSubmissionBlock !== BigInt(0)) {
      writer.uint32(112).uint64(message.lastSubmissionBlock);
    }
    Object.entries(message.perMethod).forEach(([key, value]) => {
      AgentCalibration_PerMethodEntry.encode({
        key: key as any,
        value
      }, writer.uint32(122).fork()).ldelim();
    });
    if (message.calibrationScoreBps !== BigInt(0)) {
      writer.uint32(128).uint64(message.calibrationScoreBps);
    }
    if (message.lastUpdatedBlock !== BigInt(0)) {
      writer.uint32(136).uint64(message.lastUpdatedBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AgentCalibration {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAgentCalibration();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.accountType = reader.string();
          break;
        case 3:
          message.totalSubmissions = reader.uint64();
          break;
        case 4:
          message.accepted = reader.uint64();
          break;
        case 5:
          message.rejected = reader.uint64();
          break;
        case 6:
          message.malformed = reader.uint64();
          break;
        case 7:
          message.inconclusive = reader.uint64();
          break;
        case 8:
          message.corroborationsEarned = reader.uint64();
          break;
        case 9:
          message.disprovenCount = reader.uint64();
          break;
        case 10:
          message.challengesIssued = reader.uint64();
          break;
        case 11:
          message.challengesSucceeded = reader.uint64();
          break;
        case 12:
          message.challengesFailed = reader.uint64();
          break;
        case 13:
          message.firstSubmissionBlock = reader.uint64();
          break;
        case 14:
          message.lastSubmissionBlock = reader.uint64();
          break;
        case 15:
          const entry15 = AgentCalibration_PerMethodEntry.decode(reader, reader.uint32());
          if (entry15.value !== undefined) {
            message.perMethod[entry15.key] = entry15.value;
          }
          break;
        case 16:
          message.calibrationScoreBps = reader.uint64();
          break;
        case 17:
          message.lastUpdatedBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AgentCalibration>): AgentCalibration {
    const message = createBaseAgentCalibration();
    message.address = object.address ?? "";
    message.accountType = object.accountType ?? "";
    message.totalSubmissions = object.totalSubmissions !== undefined && object.totalSubmissions !== null ? BigInt(object.totalSubmissions.toString()) : BigInt(0);
    message.accepted = object.accepted !== undefined && object.accepted !== null ? BigInt(object.accepted.toString()) : BigInt(0);
    message.rejected = object.rejected !== undefined && object.rejected !== null ? BigInt(object.rejected.toString()) : BigInt(0);
    message.malformed = object.malformed !== undefined && object.malformed !== null ? BigInt(object.malformed.toString()) : BigInt(0);
    message.inconclusive = object.inconclusive !== undefined && object.inconclusive !== null ? BigInt(object.inconclusive.toString()) : BigInt(0);
    message.corroborationsEarned = object.corroborationsEarned !== undefined && object.corroborationsEarned !== null ? BigInt(object.corroborationsEarned.toString()) : BigInt(0);
    message.disprovenCount = object.disprovenCount !== undefined && object.disprovenCount !== null ? BigInt(object.disprovenCount.toString()) : BigInt(0);
    message.challengesIssued = object.challengesIssued !== undefined && object.challengesIssued !== null ? BigInt(object.challengesIssued.toString()) : BigInt(0);
    message.challengesSucceeded = object.challengesSucceeded !== undefined && object.challengesSucceeded !== null ? BigInt(object.challengesSucceeded.toString()) : BigInt(0);
    message.challengesFailed = object.challengesFailed !== undefined && object.challengesFailed !== null ? BigInt(object.challengesFailed.toString()) : BigInt(0);
    message.firstSubmissionBlock = object.firstSubmissionBlock !== undefined && object.firstSubmissionBlock !== null ? BigInt(object.firstSubmissionBlock.toString()) : BigInt(0);
    message.lastSubmissionBlock = object.lastSubmissionBlock !== undefined && object.lastSubmissionBlock !== null ? BigInt(object.lastSubmissionBlock.toString()) : BigInt(0);
    message.perMethod = Object.entries(object.perMethod ?? {}).reduce<{
      [key: string]: AgentMethodStats;
    }>((acc, [key, value]) => {
      if (value !== undefined) {
        acc[key] = AgentMethodStats.fromPartial(value);
      }
      return acc;
    }, {});
    message.calibrationScoreBps = object.calibrationScoreBps !== undefined && object.calibrationScoreBps !== null ? BigInt(object.calibrationScoreBps.toString()) : BigInt(0);
    message.lastUpdatedBlock = object.lastUpdatedBlock !== undefined && object.lastUpdatedBlock !== null ? BigInt(object.lastUpdatedBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCommonKnowledgeEntry(): CommonKnowledgeEntry {
  return {
    id: "",
    domain: "",
    subject: "",
    description: "",
    penaltyBps: BigInt(0),
    addedBlock: BigInt(0)
  };
}
/**
 * CommonKnowledgeEntry represents a subject that LLMs already know.
 * Claims matching these subjects receive a novelty penalty.
 * @name CommonKnowledgeEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CommonKnowledgeEntry
 */
export const CommonKnowledgeEntry = {
  typeUrl: "/zerone.knowledge.v1.CommonKnowledgeEntry",
  encode(message: CommonKnowledgeEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.subject !== "") {
      writer.uint32(26).string(message.subject);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.penaltyBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.penaltyBps);
    }
    if (message.addedBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.addedBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CommonKnowledgeEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCommonKnowledgeEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.subject = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.penaltyBps = reader.uint64();
          break;
        case 6:
          message.addedBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CommonKnowledgeEntry>): CommonKnowledgeEntry {
    const message = createBaseCommonKnowledgeEntry();
    message.id = object.id ?? "";
    message.domain = object.domain ?? "";
    message.subject = object.subject ?? "";
    message.description = object.description ?? "";
    message.penaltyBps = object.penaltyBps !== undefined && object.penaltyBps !== null ? BigInt(object.penaltyBps.toString()) : BigInt(0);
    message.addedBlock = object.addedBlock !== undefined && object.addedBlock !== null ? BigInt(object.addedBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseClaim(): Claim {
  return {
    id: "",
    factContent: "",
    domain: "",
    category: "",
    submitter: "",
    submittedAtBlock: BigInt(0),
    status: 0,
    references: [],
    verificationRoundId: "",
    stake: "",
    partnershipId: "",
    challengeWindowEnd: BigInt(0),
    provisionalFactId: "",
    contentHash: "",
    claimType: 0,
    relations: [],
    structure: undefined,
    canonicalForm: "",
    canonicalHash: "",
    methodId: "",
    reasoningTrace: "",
    argumentText: "",
    rebuttalText: ""
  };
}
/**
 * Claim is an unverified submission awaiting or undergoing verification.
 * @name Claim
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Claim
 */
export const Claim = {
  typeUrl: "/zerone.knowledge.v1.Claim",
  encode(message: Claim, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.factContent !== "") {
      writer.uint32(18).string(message.factContent);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    if (message.submitter !== "") {
      writer.uint32(42).string(message.submitter);
    }
    if (message.submittedAtBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.submittedAtBlock);
    }
    if (message.status !== 0) {
      writer.uint32(56).int32(message.status);
    }
    for (const v of message.references) {
      writer.uint32(66).string(v!);
    }
    if (message.verificationRoundId !== "") {
      writer.uint32(74).string(message.verificationRoundId);
    }
    if (message.stake !== "") {
      writer.uint32(82).string(message.stake);
    }
    if (message.partnershipId !== "") {
      writer.uint32(90).string(message.partnershipId);
    }
    if (message.challengeWindowEnd !== BigInt(0)) {
      writer.uint32(96).uint64(message.challengeWindowEnd);
    }
    if (message.provisionalFactId !== "") {
      writer.uint32(106).string(message.provisionalFactId);
    }
    if (message.contentHash !== "") {
      writer.uint32(114).string(message.contentHash);
    }
    if (message.claimType !== 0) {
      writer.uint32(120).int32(message.claimType);
    }
    for (const v of message.relations) {
      ClaimRelation.encode(v!, writer.uint32(130).fork()).ldelim();
    }
    if (message.structure !== undefined) {
      ClaimStructure.encode(message.structure, writer.uint32(138).fork()).ldelim();
    }
    if (message.canonicalForm !== "") {
      writer.uint32(146).string(message.canonicalForm);
    }
    if (message.canonicalHash !== "") {
      writer.uint32(154).string(message.canonicalHash);
    }
    if (message.methodId !== "") {
      writer.uint32(162).string(message.methodId);
    }
    if (message.reasoningTrace !== "") {
      writer.uint32(170).string(message.reasoningTrace);
    }
    if (message.argumentText !== "") {
      writer.uint32(178).string(message.argumentText);
    }
    if (message.rebuttalText !== "") {
      writer.uint32(186).string(message.rebuttalText);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Claim {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaim();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.factContent = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.category = reader.string();
          break;
        case 5:
          message.submitter = reader.string();
          break;
        case 6:
          message.submittedAtBlock = reader.uint64();
          break;
        case 7:
          message.status = reader.int32() as any;
          break;
        case 8:
          message.references.push(reader.string());
          break;
        case 9:
          message.verificationRoundId = reader.string();
          break;
        case 10:
          message.stake = reader.string();
          break;
        case 11:
          message.partnershipId = reader.string();
          break;
        case 12:
          message.challengeWindowEnd = reader.uint64();
          break;
        case 13:
          message.provisionalFactId = reader.string();
          break;
        case 14:
          message.contentHash = reader.string();
          break;
        case 15:
          message.claimType = reader.int32() as any;
          break;
        case 16:
          message.relations.push(ClaimRelation.decode(reader, reader.uint32()));
          break;
        case 17:
          message.structure = ClaimStructure.decode(reader, reader.uint32());
          break;
        case 18:
          message.canonicalForm = reader.string();
          break;
        case 19:
          message.canonicalHash = reader.string();
          break;
        case 20:
          message.methodId = reader.string();
          break;
        case 21:
          message.reasoningTrace = reader.string();
          break;
        case 22:
          message.argumentText = reader.string();
          break;
        case 23:
          message.rebuttalText = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Claim>): Claim {
    const message = createBaseClaim();
    message.id = object.id ?? "";
    message.factContent = object.factContent ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    message.submitter = object.submitter ?? "";
    message.submittedAtBlock = object.submittedAtBlock !== undefined && object.submittedAtBlock !== null ? BigInt(object.submittedAtBlock.toString()) : BigInt(0);
    message.status = object.status ?? 0;
    message.references = object.references?.map(e => e) || [];
    message.verificationRoundId = object.verificationRoundId ?? "";
    message.stake = object.stake ?? "";
    message.partnershipId = object.partnershipId ?? "";
    message.challengeWindowEnd = object.challengeWindowEnd !== undefined && object.challengeWindowEnd !== null ? BigInt(object.challengeWindowEnd.toString()) : BigInt(0);
    message.provisionalFactId = object.provisionalFactId ?? "";
    message.contentHash = object.contentHash ?? "";
    message.claimType = object.claimType ?? 0;
    message.relations = object.relations?.map(e => ClaimRelation.fromPartial(e)) || [];
    message.structure = object.structure !== undefined && object.structure !== null ? ClaimStructure.fromPartial(object.structure) : undefined;
    message.canonicalForm = object.canonicalForm ?? "";
    message.canonicalHash = object.canonicalHash ?? "";
    message.methodId = object.methodId ?? "";
    message.reasoningTrace = object.reasoningTrace ?? "";
    message.argumentText = object.argumentText ?? "";
    message.rebuttalText = object.rebuttalText ?? "";
    return message;
  }
};
function createBaseVerificationRound(): VerificationRound {
  return {
    id: "",
    claimId: "",
    startedAtBlock: BigInt(0),
    phase: 0,
    selectedVerifiers: [],
    commits: [],
    reveals: [],
    verdict: 0,
    verdictBlock: BigInt(0),
    commitDeadline: BigInt(0),
    revealDeadline: BigInt(0),
    aggregationDeadline: BigInt(0)
  };
}
/**
 * VerificationRound tracks one commit-reveal verification cycle.
 * @name VerificationRound
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.VerificationRound
 */
export const VerificationRound = {
  typeUrl: "/zerone.knowledge.v1.VerificationRound",
  encode(message: VerificationRound, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.claimId !== "") {
      writer.uint32(18).string(message.claimId);
    }
    if (message.startedAtBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.startedAtBlock);
    }
    if (message.phase !== 0) {
      writer.uint32(32).int32(message.phase);
    }
    for (const v of message.selectedVerifiers) {
      writer.uint32(42).string(v!);
    }
    for (const v of message.commits) {
      CommitEntry.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.reveals) {
      RevealEntry.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    if (message.verdict !== 0) {
      writer.uint32(64).int32(message.verdict);
    }
    if (message.verdictBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.verdictBlock);
    }
    if (message.commitDeadline !== BigInt(0)) {
      writer.uint32(80).uint64(message.commitDeadline);
    }
    if (message.revealDeadline !== BigInt(0)) {
      writer.uint32(88).uint64(message.revealDeadline);
    }
    if (message.aggregationDeadline !== BigInt(0)) {
      writer.uint32(96).uint64(message.aggregationDeadline);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): VerificationRound {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVerificationRound();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.claimId = reader.string();
          break;
        case 3:
          message.startedAtBlock = reader.uint64();
          break;
        case 4:
          message.phase = reader.int32() as any;
          break;
        case 5:
          message.selectedVerifiers.push(reader.string());
          break;
        case 6:
          message.commits.push(CommitEntry.decode(reader, reader.uint32()));
          break;
        case 7:
          message.reveals.push(RevealEntry.decode(reader, reader.uint32()));
          break;
        case 8:
          message.verdict = reader.int32() as any;
          break;
        case 9:
          message.verdictBlock = reader.uint64();
          break;
        case 10:
          message.commitDeadline = reader.uint64();
          break;
        case 11:
          message.revealDeadline = reader.uint64();
          break;
        case 12:
          message.aggregationDeadline = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<VerificationRound>): VerificationRound {
    const message = createBaseVerificationRound();
    message.id = object.id ?? "";
    message.claimId = object.claimId ?? "";
    message.startedAtBlock = object.startedAtBlock !== undefined && object.startedAtBlock !== null ? BigInt(object.startedAtBlock.toString()) : BigInt(0);
    message.phase = object.phase ?? 0;
    message.selectedVerifiers = object.selectedVerifiers?.map(e => e) || [];
    message.commits = object.commits?.map(e => CommitEntry.fromPartial(e)) || [];
    message.reveals = object.reveals?.map(e => RevealEntry.fromPartial(e)) || [];
    message.verdict = object.verdict ?? 0;
    message.verdictBlock = object.verdictBlock !== undefined && object.verdictBlock !== null ? BigInt(object.verdictBlock.toString()) : BigInt(0);
    message.commitDeadline = object.commitDeadline !== undefined && object.commitDeadline !== null ? BigInt(object.commitDeadline.toString()) : BigInt(0);
    message.revealDeadline = object.revealDeadline !== undefined && object.revealDeadline !== null ? BigInt(object.revealDeadline.toString()) : BigInt(0);
    message.aggregationDeadline = object.aggregationDeadline !== undefined && object.aggregationDeadline !== null ? BigInt(object.aggregationDeadline.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCommitEntry(): CommitEntry {
  return {
    verifier: "",
    commitHash: new Uint8Array(),
    committedAtBlock: BigInt(0)
  };
}
/**
 * CommitEntry records a validator's blinded commitment (SHA-256(vote || salt)).
 * @name CommitEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CommitEntry
 */
export const CommitEntry = {
  typeUrl: "/zerone.knowledge.v1.CommitEntry",
  encode(message: CommitEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.commitHash.length !== 0) {
      writer.uint32(18).bytes(message.commitHash);
    }
    if (message.committedAtBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.committedAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CommitEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCommitEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.commitHash = reader.bytes();
          break;
        case 3:
          message.committedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CommitEntry>): CommitEntry {
    const message = createBaseCommitEntry();
    message.verifier = object.verifier ?? "";
    message.commitHash = object.commitHash ?? new Uint8Array();
    message.committedAtBlock = object.committedAtBlock !== undefined && object.committedAtBlock !== null ? BigInt(object.committedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseRevealEntry(): RevealEntry {
  return {
    verifier: "",
    vote: "",
    salt: new Uint8Array(),
    revealedAtBlock: BigInt(0)
  };
}
/**
 * RevealEntry records a validator's revealed vote and salt.
 * @name RevealEntry
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RevealEntry
 */
export const RevealEntry = {
  typeUrl: "/zerone.knowledge.v1.RevealEntry",
  encode(message: RevealEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.vote !== "") {
      writer.uint32(18).string(message.vote);
    }
    if (message.salt.length !== 0) {
      writer.uint32(26).bytes(message.salt);
    }
    if (message.revealedAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.revealedAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): RevealEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRevealEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.vote = reader.string();
          break;
        case 3:
          message.salt = reader.bytes();
          break;
        case 4:
          message.revealedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<RevealEntry>): RevealEntry {
    const message = createBaseRevealEntry();
    message.verifier = object.verifier ?? "";
    message.vote = object.vote ?? "";
    message.salt = object.salt ?? new Uint8Array();
    message.revealedAtBlock = object.revealedAtBlock !== undefined && object.revealedAtBlock !== null ? BigInt(object.revealedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseVRFProof(): VRFProof {
  return {
    proof: new Uint8Array(),
    output: new Uint8Array(),
    proposer: "",
    blockHeight: BigInt(0)
  };
}
/**
 * VRFProof captures a Verifiable Random Function output for validator selection.
 * @name VRFProof
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.VRFProof
 */
export const VRFProof = {
  typeUrl: "/zerone.knowledge.v1.VRFProof",
  encode(message: VRFProof, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proof.length !== 0) {
      writer.uint32(10).bytes(message.proof);
    }
    if (message.output.length !== 0) {
      writer.uint32(18).bytes(message.output);
    }
    if (message.proposer !== "") {
      writer.uint32(26).string(message.proposer);
    }
    if (message.blockHeight !== BigInt(0)) {
      writer.uint32(32).uint64(message.blockHeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): VRFProof {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVRFProof();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proof = reader.bytes();
          break;
        case 2:
          message.output = reader.bytes();
          break;
        case 3:
          message.proposer = reader.string();
          break;
        case 4:
          message.blockHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<VRFProof>): VRFProof {
    const message = createBaseVRFProof();
    message.proof = object.proof ?? new Uint8Array();
    message.output = object.output ?? new Uint8Array();
    message.proposer = object.proposer ?? "";
    message.blockHeight = object.blockHeight !== undefined && object.blockHeight !== null ? BigInt(object.blockHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseDomain(): Domain {
  return {
    name: "",
    description: "",
    status: 0,
    createdAtBlock: BigInt(0),
    factCount: BigInt(0),
    proposer: "",
    endorsers: [],
    stratum: "",
    parentDomain: "",
    depth: 0
  };
}
/**
 * Domain is an epistemic knowledge domain (e.g., "mathematics", "physics").
 * @name Domain
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.Domain
 */
export const Domain = {
  typeUrl: "/zerone.knowledge.v1.Domain",
  encode(message: Domain, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.description !== "") {
      writer.uint32(18).string(message.description);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.createdAtBlock);
    }
    if (message.factCount !== BigInt(0)) {
      writer.uint32(40).uint64(message.factCount);
    }
    if (message.proposer !== "") {
      writer.uint32(50).string(message.proposer);
    }
    for (const v of message.endorsers) {
      writer.uint32(58).string(v!);
    }
    if (message.stratum !== "") {
      writer.uint32(66).string(message.stratum);
    }
    if (message.parentDomain !== "") {
      writer.uint32(74).string(message.parentDomain);
    }
    if (message.depth !== 0) {
      writer.uint32(80).uint32(message.depth);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Domain {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.description = reader.string();
          break;
        case 3:
          message.status = reader.int32() as any;
          break;
        case 4:
          message.createdAtBlock = reader.uint64();
          break;
        case 5:
          message.factCount = reader.uint64();
          break;
        case 6:
          message.proposer = reader.string();
          break;
        case 7:
          message.endorsers.push(reader.string());
          break;
        case 8:
          message.stratum = reader.string();
          break;
        case 9:
          message.parentDomain = reader.string();
          break;
        case 10:
          message.depth = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Domain>): Domain {
    const message = createBaseDomain();
    message.name = object.name ?? "";
    message.description = object.description ?? "";
    message.status = object.status ?? 0;
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.factCount = object.factCount !== undefined && object.factCount !== null ? BigInt(object.factCount.toString()) : BigInt(0);
    message.proposer = object.proposer ?? "";
    message.endorsers = object.endorsers?.map(e => e) || [];
    message.stratum = object.stratum ?? "";
    message.parentDomain = object.parentDomain ?? "";
    message.depth = object.depth ?? 0;
    return message;
  }
};
function createBaseValidatorInfo(): ValidatorInfo {
  return {
    address: "",
    stake: BigInt(0),
    tier: "",
    verificationCount: BigInt(0),
    accuracyBps: BigInt(0)
  };
}
/**
 * ValidatorInfo caches a validator's tier and verification stats for round selection.
 * @name ValidatorInfo
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ValidatorInfo
 */
export const ValidatorInfo = {
  typeUrl: "/zerone.knowledge.v1.ValidatorInfo",
  encode(message: ValidatorInfo, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.stake !== BigInt(0)) {
      writer.uint32(16).uint64(message.stake);
    }
    if (message.tier !== "") {
      writer.uint32(26).string(message.tier);
    }
    if (message.verificationCount !== BigInt(0)) {
      writer.uint32(32).uint64(message.verificationCount);
    }
    if (message.accuracyBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.accuracyBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ValidatorInfo {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseValidatorInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.stake = reader.uint64();
          break;
        case 3:
          message.tier = reader.string();
          break;
        case 4:
          message.verificationCount = reader.uint64();
          break;
        case 5:
          message.accuracyBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ValidatorInfo>): ValidatorInfo {
    const message = createBaseValidatorInfo();
    message.address = object.address ?? "";
    message.stake = object.stake !== undefined && object.stake !== null ? BigInt(object.stake.toString()) : BigInt(0);
    message.tier = object.tier ?? "";
    message.verificationCount = object.verificationCount !== undefined && object.verificationCount !== null ? BigInt(object.verificationCount.toString()) : BigInt(0);
    message.accuracyBps = object.accuracyBps !== undefined && object.accuracyBps !== null ? BigInt(object.accuracyBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseProvisionalChallenge(): ProvisionalChallenge {
  return {
    id: "",
    claimId: "",
    factId: "",
    challenger: "",
    stake: "",
    reason: "",
    evidenceIds: [],
    counterClaim: "",
    status: "",
    resolutionPath: "",
    disputeId: "",
    createdAtHeight: BigInt(0),
    resolvedAtHeight: BigInt(0),
    outcome: "",
    attemptNumber: 0
  };
}
/**
 * ProvisionalChallenge is an adversarial challenge against a provisional fact.
 * @name ProvisionalChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ProvisionalChallenge
 */
export const ProvisionalChallenge = {
  typeUrl: "/zerone.knowledge.v1.ProvisionalChallenge",
  encode(message: ProvisionalChallenge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.claimId !== "") {
      writer.uint32(18).string(message.claimId);
    }
    if (message.factId !== "") {
      writer.uint32(26).string(message.factId);
    }
    if (message.challenger !== "") {
      writer.uint32(34).string(message.challenger);
    }
    if (message.stake !== "") {
      writer.uint32(42).string(message.stake);
    }
    if (message.reason !== "") {
      writer.uint32(50).string(message.reason);
    }
    for (const v of message.evidenceIds) {
      writer.uint32(58).string(v!);
    }
    if (message.counterClaim !== "") {
      writer.uint32(66).string(message.counterClaim);
    }
    if (message.status !== "") {
      writer.uint32(74).string(message.status);
    }
    if (message.resolutionPath !== "") {
      writer.uint32(82).string(message.resolutionPath);
    }
    if (message.disputeId !== "") {
      writer.uint32(90).string(message.disputeId);
    }
    if (message.createdAtHeight !== BigInt(0)) {
      writer.uint32(96).uint64(message.createdAtHeight);
    }
    if (message.resolvedAtHeight !== BigInt(0)) {
      writer.uint32(104).uint64(message.resolvedAtHeight);
    }
    if (message.outcome !== "") {
      writer.uint32(114).string(message.outcome);
    }
    if (message.attemptNumber !== 0) {
      writer.uint32(120).uint32(message.attemptNumber);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ProvisionalChallenge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseProvisionalChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.claimId = reader.string();
          break;
        case 3:
          message.factId = reader.string();
          break;
        case 4:
          message.challenger = reader.string();
          break;
        case 5:
          message.stake = reader.string();
          break;
        case 6:
          message.reason = reader.string();
          break;
        case 7:
          message.evidenceIds.push(reader.string());
          break;
        case 8:
          message.counterClaim = reader.string();
          break;
        case 9:
          message.status = reader.string();
          break;
        case 10:
          message.resolutionPath = reader.string();
          break;
        case 11:
          message.disputeId = reader.string();
          break;
        case 12:
          message.createdAtHeight = reader.uint64();
          break;
        case 13:
          message.resolvedAtHeight = reader.uint64();
          break;
        case 14:
          message.outcome = reader.string();
          break;
        case 15:
          message.attemptNumber = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ProvisionalChallenge>): ProvisionalChallenge {
    const message = createBaseProvisionalChallenge();
    message.id = object.id ?? "";
    message.claimId = object.claimId ?? "";
    message.factId = object.factId ?? "";
    message.challenger = object.challenger ?? "";
    message.stake = object.stake ?? "";
    message.reason = object.reason ?? "";
    message.evidenceIds = object.evidenceIds?.map(e => e) || [];
    message.counterClaim = object.counterClaim ?? "";
    message.status = object.status ?? "";
    message.resolutionPath = object.resolutionPath ?? "";
    message.disputeId = object.disputeId ?? "";
    message.createdAtHeight = object.createdAtHeight !== undefined && object.createdAtHeight !== null ? BigInt(object.createdAtHeight.toString()) : BigInt(0);
    message.resolvedAtHeight = object.resolvedAtHeight !== undefined && object.resolvedAtHeight !== null ? BigInt(object.resolvedAtHeight.toString()) : BigInt(0);
    message.outcome = object.outcome ?? "";
    message.attemptNumber = object.attemptNumber ?? 0;
    return message;
  }
};
function createBaseDemandSignal(): DemandSignal {
  return {
    domain: "",
    subject: "",
    queryCount: BigInt(0),
    fulfilledCount: BigInt(0),
    unfulfilledCount: BigInt(0),
    lastQueryBlock: BigInt(0),
    epochQueryCount: BigInt(0),
    epochUnfulfilled: BigInt(0)
  };
}
/**
 * DemandSignal tracks aggregate query demand for a domain/subject pair.
 * @name DemandSignal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DemandSignal
 */
export const DemandSignal = {
  typeUrl: "/zerone.knowledge.v1.DemandSignal",
  encode(message: DemandSignal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.domain !== "") {
      writer.uint32(10).string(message.domain);
    }
    if (message.subject !== "") {
      writer.uint32(18).string(message.subject);
    }
    if (message.queryCount !== BigInt(0)) {
      writer.uint32(24).uint64(message.queryCount);
    }
    if (message.fulfilledCount !== BigInt(0)) {
      writer.uint32(32).uint64(message.fulfilledCount);
    }
    if (message.unfulfilledCount !== BigInt(0)) {
      writer.uint32(40).uint64(message.unfulfilledCount);
    }
    if (message.lastQueryBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.lastQueryBlock);
    }
    if (message.epochQueryCount !== BigInt(0)) {
      writer.uint32(56).uint64(message.epochQueryCount);
    }
    if (message.epochUnfulfilled !== BigInt(0)) {
      writer.uint32(64).uint64(message.epochUnfulfilled);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DemandSignal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDemandSignal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.domain = reader.string();
          break;
        case 2:
          message.subject = reader.string();
          break;
        case 3:
          message.queryCount = reader.uint64();
          break;
        case 4:
          message.fulfilledCount = reader.uint64();
          break;
        case 5:
          message.unfulfilledCount = reader.uint64();
          break;
        case 6:
          message.lastQueryBlock = reader.uint64();
          break;
        case 7:
          message.epochQueryCount = reader.uint64();
          break;
        case 8:
          message.epochUnfulfilled = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DemandSignal>): DemandSignal {
    const message = createBaseDemandSignal();
    message.domain = object.domain ?? "";
    message.subject = object.subject ?? "";
    message.queryCount = object.queryCount !== undefined && object.queryCount !== null ? BigInt(object.queryCount.toString()) : BigInt(0);
    message.fulfilledCount = object.fulfilledCount !== undefined && object.fulfilledCount !== null ? BigInt(object.fulfilledCount.toString()) : BigInt(0);
    message.unfulfilledCount = object.unfulfilledCount !== undefined && object.unfulfilledCount !== null ? BigInt(object.unfulfilledCount.toString()) : BigInt(0);
    message.lastQueryBlock = object.lastQueryBlock !== undefined && object.lastQueryBlock !== null ? BigInt(object.lastQueryBlock.toString()) : BigInt(0);
    message.epochQueryCount = object.epochQueryCount !== undefined && object.epochQueryCount !== null ? BigInt(object.epochQueryCount.toString()) : BigInt(0);
    message.epochUnfulfilled = object.epochUnfulfilled !== undefined && object.epochUnfulfilled !== null ? BigInt(object.epochUnfulfilled.toString()) : BigInt(0);
    return message;
  }
};
function createBaseKnowledgeBounty(): KnowledgeBounty {
  return {
    id: "",
    domain: "",
    subject: "",
    rewardAmount: "",
    createdAtBlock: BigInt(0),
    expiresAtBlock: BigInt(0),
    claimed: false,
    claimedByFactId: "",
    demandCount: BigInt(0)
  };
}
/**
 * KnowledgeBounty is an auto-generated reward for filling a knowledge gap.
 * @name KnowledgeBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.KnowledgeBounty
 */
export const KnowledgeBounty = {
  typeUrl: "/zerone.knowledge.v1.KnowledgeBounty",
  encode(message: KnowledgeBounty, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.subject !== "") {
      writer.uint32(26).string(message.subject);
    }
    if (message.rewardAmount !== "") {
      writer.uint32(34).string(message.rewardAmount);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.createdAtBlock);
    }
    if (message.expiresAtBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.expiresAtBlock);
    }
    if (message.claimed === true) {
      writer.uint32(56).bool(message.claimed);
    }
    if (message.claimedByFactId !== "") {
      writer.uint32(66).string(message.claimedByFactId);
    }
    if (message.demandCount !== BigInt(0)) {
      writer.uint32(72).uint64(message.demandCount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): KnowledgeBounty {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseKnowledgeBounty();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.subject = reader.string();
          break;
        case 4:
          message.rewardAmount = reader.string();
          break;
        case 5:
          message.createdAtBlock = reader.uint64();
          break;
        case 6:
          message.expiresAtBlock = reader.uint64();
          break;
        case 7:
          message.claimed = reader.bool();
          break;
        case 8:
          message.claimedByFactId = reader.string();
          break;
        case 9:
          message.demandCount = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<KnowledgeBounty>): KnowledgeBounty {
    const message = createBaseKnowledgeBounty();
    message.id = object.id ?? "";
    message.domain = object.domain ?? "";
    message.subject = object.subject ?? "";
    message.rewardAmount = object.rewardAmount ?? "";
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.expiresAtBlock = object.expiresAtBlock !== undefined && object.expiresAtBlock !== null ? BigInt(object.expiresAtBlock.toString()) : BigInt(0);
    message.claimed = object.claimed ?? false;
    message.claimedByFactId = object.claimedByFactId ?? "";
    message.demandCount = object.demandCount !== undefined && object.demandCount !== null ? BigInt(object.demandCount.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCompletedRoundMeta(): CompletedRoundMeta {
  return {
    domain: "",
    hasDissent: false,
    durationBlocks: BigInt(0)
  };
}
/**
 * CompletedRoundMeta stores metadata for completed verification rounds,
 * indexed by verdict block height for efficient window-based queries (R31-2).
 * @name CompletedRoundMeta
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CompletedRoundMeta
 */
export const CompletedRoundMeta = {
  typeUrl: "/zerone.knowledge.v1.CompletedRoundMeta",
  encode(message: CompletedRoundMeta, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.domain !== "") {
      writer.uint32(10).string(message.domain);
    }
    if (message.hasDissent === true) {
      writer.uint32(16).bool(message.hasDissent);
    }
    if (message.durationBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.durationBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CompletedRoundMeta {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCompletedRoundMeta();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.domain = reader.string();
          break;
        case 2:
          message.hasDissent = reader.bool();
          break;
        case 3:
          message.durationBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CompletedRoundMeta>): CompletedRoundMeta {
    const message = createBaseCompletedRoundMeta();
    message.domain = object.domain ?? "";
    message.hasDissent = object.hasDissent ?? false;
    message.durationBlocks = object.durationBlocks !== undefined && object.durationBlocks !== null ? BigInt(object.durationBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMethodologyApplicationTrace(): MethodologyApplicationTrace {
  return {
    traceId: "",
    factId: "",
    snapshotBlockHeight: BigInt(0),
    tokenizerVersion: BigInt(0),
    canonicalSerialisationVersion: BigInt(0),
    traceSchemaVersion: BigInt(0),
    content: "",
    domain: "",
    subject: "",
    canonicalForm: "",
    methodologyId: "",
    methodologyRubric: "",
    reasoningTrace: "",
    axiomDistance: 0,
    dependencyConfidenceFloorBps: BigInt(0),
    predecessorEdges: [],
    descendantEdges: [],
    groundedScoreBps: BigInt(0),
    ownConfidenceBps: BigInt(0),
    verifierPanelSize: 0,
    dissentingVerifiers: [],
    verifiedAtBlock: BigInt(0),
    challenges: [],
    corroborationCount: BigInt(0),
    lastCorroboratedBlock: BigInt(0),
    status: 0,
    vindication: undefined,
    disproval: undefined,
    supersessionChain: [],
    reformulations: [],
    driftExamples: [],
    contradictingFactIds: [],
    submitter: "",
    submitterCalibrationAtSubmissionBps: BigInt(0),
    partnershipId: "",
    submittedAtBlock: BigInt(0),
    trainingValueWeightBps: BigInt(0),
    curriculumTier: 0,
    qualityTier: 0,
    isNormative: false,
    reasoningSteps: [],
    methodologyChoice: undefined,
    beliefRevisions: [],
    dialecticTree: []
  };
}
/**
 * @name MethodologyApplicationTrace
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MethodologyApplicationTrace
 */
export const MethodologyApplicationTrace = {
  typeUrl: "/zerone.knowledge.v1.MethodologyApplicationTrace",
  encode(message: MethodologyApplicationTrace, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.traceId !== "") {
      writer.uint32(10).string(message.traceId);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.snapshotBlockHeight !== BigInt(0)) {
      writer.uint32(24).uint64(message.snapshotBlockHeight);
    }
    if (message.tokenizerVersion !== BigInt(0)) {
      writer.uint32(32).uint64(message.tokenizerVersion);
    }
    if (message.canonicalSerialisationVersion !== BigInt(0)) {
      writer.uint32(40).uint64(message.canonicalSerialisationVersion);
    }
    if (message.traceSchemaVersion !== BigInt(0)) {
      writer.uint32(48).uint64(message.traceSchemaVersion);
    }
    if (message.content !== "") {
      writer.uint32(82).string(message.content);
    }
    if (message.domain !== "") {
      writer.uint32(90).string(message.domain);
    }
    if (message.subject !== "") {
      writer.uint32(98).string(message.subject);
    }
    if (message.canonicalForm !== "") {
      writer.uint32(106).string(message.canonicalForm);
    }
    if (message.methodologyId !== "") {
      writer.uint32(162).string(message.methodologyId);
    }
    if (message.methodologyRubric !== "") {
      writer.uint32(170).string(message.methodologyRubric);
    }
    if (message.reasoningTrace !== "") {
      writer.uint32(178).string(message.reasoningTrace);
    }
    if (message.axiomDistance !== 0) {
      writer.uint32(184).uint32(message.axiomDistance);
    }
    if (message.dependencyConfidenceFloorBps !== BigInt(0)) {
      writer.uint32(192).uint64(message.dependencyConfidenceFloorBps);
    }
    for (const v of message.predecessorEdges) {
      FactRelation.encode(v!, writer.uint32(242).fork()).ldelim();
    }
    for (const v of message.descendantEdges) {
      FactRelation.encode(v!, writer.uint32(250).fork()).ldelim();
    }
    if (message.groundedScoreBps !== BigInt(0)) {
      writer.uint32(256).uint64(message.groundedScoreBps);
    }
    if (message.ownConfidenceBps !== BigInt(0)) {
      writer.uint32(320).uint64(message.ownConfidenceBps);
    }
    if (message.verifierPanelSize !== 0) {
      writer.uint32(328).uint32(message.verifierPanelSize);
    }
    for (const v of message.dissentingVerifiers) {
      writer.uint32(338).string(v!);
    }
    if (message.verifiedAtBlock !== BigInt(0)) {
      writer.uint32(344).uint64(message.verifiedAtBlock);
    }
    for (const v of message.challenges) {
      TraceChallenge.encode(v!, writer.uint32(402).fork()).ldelim();
    }
    if (message.corroborationCount !== BigInt(0)) {
      writer.uint32(408).uint64(message.corroborationCount);
    }
    if (message.lastCorroboratedBlock !== BigInt(0)) {
      writer.uint32(416).uint64(message.lastCorroboratedBlock);
    }
    if (message.status !== 0) {
      writer.uint32(480).int32(message.status);
    }
    if (message.vindication !== undefined) {
      TraceVindication.encode(message.vindication, writer.uint32(490).fork()).ldelim();
    }
    if (message.disproval !== undefined) {
      TraceDisproval.encode(message.disproval, writer.uint32(498).fork()).ldelim();
    }
    for (const v of message.supersessionChain) {
      writer.uint32(506).string(v!);
    }
    for (const v of message.reformulations) {
      TraceReformulation.encode(v!, writer.uint32(562).fork()).ldelim();
    }
    for (const v of message.driftExamples) {
      TraceDrift.encode(v!, writer.uint32(570).fork()).ldelim();
    }
    for (const v of message.contradictingFactIds) {
      writer.uint32(578).string(v!);
    }
    if (message.submitter !== "") {
      writer.uint32(642).string(message.submitter);
    }
    if (message.submitterCalibrationAtSubmissionBps !== BigInt(0)) {
      writer.uint32(648).uint64(message.submitterCalibrationAtSubmissionBps);
    }
    if (message.partnershipId !== "") {
      writer.uint32(658).string(message.partnershipId);
    }
    if (message.submittedAtBlock !== BigInt(0)) {
      writer.uint32(664).uint64(message.submittedAtBlock);
    }
    if (message.trainingValueWeightBps !== BigInt(0)) {
      writer.uint32(672).uint64(message.trainingValueWeightBps);
    }
    if (message.curriculumTier !== 0) {
      writer.uint32(680).int32(message.curriculumTier);
    }
    if (message.qualityTier !== 0) {
      writer.uint32(688).int32(message.qualityTier);
    }
    if (message.isNormative === true) {
      writer.uint32(720).bool(message.isNormative);
    }
    for (const v of message.reasoningSteps) {
      ReasoningStep.encode(v!, writer.uint32(802).fork()).ldelim();
    }
    if (message.methodologyChoice !== undefined) {
      MethodologyChoice.encode(message.methodologyChoice, writer.uint32(810).fork()).ldelim();
    }
    for (const v of message.beliefRevisions) {
      BeliefRevision.encode(v!, writer.uint32(818).fork()).ldelim();
    }
    for (const v of message.dialecticTree) {
      DialecticNode.encode(v!, writer.uint32(826).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MethodologyApplicationTrace {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMethodologyApplicationTrace();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.traceId = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.snapshotBlockHeight = reader.uint64();
          break;
        case 4:
          message.tokenizerVersion = reader.uint64();
          break;
        case 5:
          message.canonicalSerialisationVersion = reader.uint64();
          break;
        case 6:
          message.traceSchemaVersion = reader.uint64();
          break;
        case 10:
          message.content = reader.string();
          break;
        case 11:
          message.domain = reader.string();
          break;
        case 12:
          message.subject = reader.string();
          break;
        case 13:
          message.canonicalForm = reader.string();
          break;
        case 20:
          message.methodologyId = reader.string();
          break;
        case 21:
          message.methodologyRubric = reader.string();
          break;
        case 22:
          message.reasoningTrace = reader.string();
          break;
        case 23:
          message.axiomDistance = reader.uint32();
          break;
        case 24:
          message.dependencyConfidenceFloorBps = reader.uint64();
          break;
        case 30:
          message.predecessorEdges.push(FactRelation.decode(reader, reader.uint32()));
          break;
        case 31:
          message.descendantEdges.push(FactRelation.decode(reader, reader.uint32()));
          break;
        case 32:
          message.groundedScoreBps = reader.uint64();
          break;
        case 40:
          message.ownConfidenceBps = reader.uint64();
          break;
        case 41:
          message.verifierPanelSize = reader.uint32();
          break;
        case 42:
          message.dissentingVerifiers.push(reader.string());
          break;
        case 43:
          message.verifiedAtBlock = reader.uint64();
          break;
        case 50:
          message.challenges.push(TraceChallenge.decode(reader, reader.uint32()));
          break;
        case 51:
          message.corroborationCount = reader.uint64();
          break;
        case 52:
          message.lastCorroboratedBlock = reader.uint64();
          break;
        case 60:
          message.status = reader.int32() as any;
          break;
        case 61:
          message.vindication = TraceVindication.decode(reader, reader.uint32());
          break;
        case 62:
          message.disproval = TraceDisproval.decode(reader, reader.uint32());
          break;
        case 63:
          message.supersessionChain.push(reader.string());
          break;
        case 70:
          message.reformulations.push(TraceReformulation.decode(reader, reader.uint32()));
          break;
        case 71:
          message.driftExamples.push(TraceDrift.decode(reader, reader.uint32()));
          break;
        case 72:
          message.contradictingFactIds.push(reader.string());
          break;
        case 80:
          message.submitter = reader.string();
          break;
        case 81:
          message.submitterCalibrationAtSubmissionBps = reader.uint64();
          break;
        case 82:
          message.partnershipId = reader.string();
          break;
        case 83:
          message.submittedAtBlock = reader.uint64();
          break;
        case 84:
          message.trainingValueWeightBps = reader.uint64();
          break;
        case 85:
          message.curriculumTier = reader.int32() as any;
          break;
        case 86:
          message.qualityTier = reader.int32() as any;
          break;
        case 90:
          message.isNormative = reader.bool();
          break;
        case 100:
          message.reasoningSteps.push(ReasoningStep.decode(reader, reader.uint32()));
          break;
        case 101:
          message.methodologyChoice = MethodologyChoice.decode(reader, reader.uint32());
          break;
        case 102:
          message.beliefRevisions.push(BeliefRevision.decode(reader, reader.uint32()));
          break;
        case 103:
          message.dialecticTree.push(DialecticNode.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MethodologyApplicationTrace>): MethodologyApplicationTrace {
    const message = createBaseMethodologyApplicationTrace();
    message.traceId = object.traceId ?? "";
    message.factId = object.factId ?? "";
    message.snapshotBlockHeight = object.snapshotBlockHeight !== undefined && object.snapshotBlockHeight !== null ? BigInt(object.snapshotBlockHeight.toString()) : BigInt(0);
    message.tokenizerVersion = object.tokenizerVersion !== undefined && object.tokenizerVersion !== null ? BigInt(object.tokenizerVersion.toString()) : BigInt(0);
    message.canonicalSerialisationVersion = object.canonicalSerialisationVersion !== undefined && object.canonicalSerialisationVersion !== null ? BigInt(object.canonicalSerialisationVersion.toString()) : BigInt(0);
    message.traceSchemaVersion = object.traceSchemaVersion !== undefined && object.traceSchemaVersion !== null ? BigInt(object.traceSchemaVersion.toString()) : BigInt(0);
    message.content = object.content ?? "";
    message.domain = object.domain ?? "";
    message.subject = object.subject ?? "";
    message.canonicalForm = object.canonicalForm ?? "";
    message.methodologyId = object.methodologyId ?? "";
    message.methodologyRubric = object.methodologyRubric ?? "";
    message.reasoningTrace = object.reasoningTrace ?? "";
    message.axiomDistance = object.axiomDistance ?? 0;
    message.dependencyConfidenceFloorBps = object.dependencyConfidenceFloorBps !== undefined && object.dependencyConfidenceFloorBps !== null ? BigInt(object.dependencyConfidenceFloorBps.toString()) : BigInt(0);
    message.predecessorEdges = object.predecessorEdges?.map(e => FactRelation.fromPartial(e)) || [];
    message.descendantEdges = object.descendantEdges?.map(e => FactRelation.fromPartial(e)) || [];
    message.groundedScoreBps = object.groundedScoreBps !== undefined && object.groundedScoreBps !== null ? BigInt(object.groundedScoreBps.toString()) : BigInt(0);
    message.ownConfidenceBps = object.ownConfidenceBps !== undefined && object.ownConfidenceBps !== null ? BigInt(object.ownConfidenceBps.toString()) : BigInt(0);
    message.verifierPanelSize = object.verifierPanelSize ?? 0;
    message.dissentingVerifiers = object.dissentingVerifiers?.map(e => e) || [];
    message.verifiedAtBlock = object.verifiedAtBlock !== undefined && object.verifiedAtBlock !== null ? BigInt(object.verifiedAtBlock.toString()) : BigInt(0);
    message.challenges = object.challenges?.map(e => TraceChallenge.fromPartial(e)) || [];
    message.corroborationCount = object.corroborationCount !== undefined && object.corroborationCount !== null ? BigInt(object.corroborationCount.toString()) : BigInt(0);
    message.lastCorroboratedBlock = object.lastCorroboratedBlock !== undefined && object.lastCorroboratedBlock !== null ? BigInt(object.lastCorroboratedBlock.toString()) : BigInt(0);
    message.status = object.status ?? 0;
    message.vindication = object.vindication !== undefined && object.vindication !== null ? TraceVindication.fromPartial(object.vindication) : undefined;
    message.disproval = object.disproval !== undefined && object.disproval !== null ? TraceDisproval.fromPartial(object.disproval) : undefined;
    message.supersessionChain = object.supersessionChain?.map(e => e) || [];
    message.reformulations = object.reformulations?.map(e => TraceReformulation.fromPartial(e)) || [];
    message.driftExamples = object.driftExamples?.map(e => TraceDrift.fromPartial(e)) || [];
    message.contradictingFactIds = object.contradictingFactIds?.map(e => e) || [];
    message.submitter = object.submitter ?? "";
    message.submitterCalibrationAtSubmissionBps = object.submitterCalibrationAtSubmissionBps !== undefined && object.submitterCalibrationAtSubmissionBps !== null ? BigInt(object.submitterCalibrationAtSubmissionBps.toString()) : BigInt(0);
    message.partnershipId = object.partnershipId ?? "";
    message.submittedAtBlock = object.submittedAtBlock !== undefined && object.submittedAtBlock !== null ? BigInt(object.submittedAtBlock.toString()) : BigInt(0);
    message.trainingValueWeightBps = object.trainingValueWeightBps !== undefined && object.trainingValueWeightBps !== null ? BigInt(object.trainingValueWeightBps.toString()) : BigInt(0);
    message.curriculumTier = object.curriculumTier ?? 0;
    message.qualityTier = object.qualityTier ?? 0;
    message.isNormative = object.isNormative ?? false;
    message.reasoningSteps = object.reasoningSteps?.map(e => ReasoningStep.fromPartial(e)) || [];
    message.methodologyChoice = object.methodologyChoice !== undefined && object.methodologyChoice !== null ? MethodologyChoice.fromPartial(object.methodologyChoice) : undefined;
    message.beliefRevisions = object.beliefRevisions?.map(e => BeliefRevision.fromPartial(e)) || [];
    message.dialecticTree = object.dialecticTree?.map(e => DialecticNode.fromPartial(e)) || [];
    return message;
  }
};
function createBaseTraceChallenge(): TraceChallenge {
  return {
    challenger: "",
    argumentText: "",
    challengeMethodId: "",
    rebuttalText: "",
    outcome: "",
    resolvedBlock: BigInt(0),
    children: []
  };
}
/**
 * @name TraceChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceChallenge
 */
export const TraceChallenge = {
  typeUrl: "/zerone.knowledge.v1.TraceChallenge",
  encode(message: TraceChallenge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.argumentText !== "") {
      writer.uint32(18).string(message.argumentText);
    }
    if (message.challengeMethodId !== "") {
      writer.uint32(26).string(message.challengeMethodId);
    }
    if (message.rebuttalText !== "") {
      writer.uint32(34).string(message.rebuttalText);
    }
    if (message.outcome !== "") {
      writer.uint32(42).string(message.outcome);
    }
    if (message.resolvedBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.resolvedBlock);
    }
    for (const v of message.children) {
      DialecticNode.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TraceChallenge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTraceChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.argumentText = reader.string();
          break;
        case 3:
          message.challengeMethodId = reader.string();
          break;
        case 4:
          message.rebuttalText = reader.string();
          break;
        case 5:
          message.outcome = reader.string();
          break;
        case 6:
          message.resolvedBlock = reader.uint64();
          break;
        case 7:
          message.children.push(DialecticNode.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TraceChallenge>): TraceChallenge {
    const message = createBaseTraceChallenge();
    message.challenger = object.challenger ?? "";
    message.argumentText = object.argumentText ?? "";
    message.challengeMethodId = object.challengeMethodId ?? "";
    message.rebuttalText = object.rebuttalText ?? "";
    message.outcome = object.outcome ?? "";
    message.resolvedBlock = object.resolvedBlock !== undefined && object.resolvedBlock !== null ? BigInt(object.resolvedBlock.toString()) : BigInt(0);
    message.children = object.children?.map(e => DialecticNode.fromPartial(e)) || [];
    return message;
  }
};
function createBaseReasoningStep(): ReasoningStep {
  return {
    stepIndex: 0,
    content: "",
    stepInference: 0,
    predecessorFactIds: [],
    dependsOnSteps: [],
    verdict: 0,
    verdictNote: "",
    stepConfidenceBps: BigInt(0)
  };
}
/**
 * ReasoningStep is one unit of a structured reasoning trace. The model
 * learns to generate these in sequence, each step grounded in prior facts
 * and declared moves.
 * @name ReasoningStep
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ReasoningStep
 */
export const ReasoningStep = {
  typeUrl: "/zerone.knowledge.v1.ReasoningStep",
  encode(message: ReasoningStep, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.stepIndex !== 0) {
      writer.uint32(8).uint32(message.stepIndex);
    }
    if (message.content !== "") {
      writer.uint32(18).string(message.content);
    }
    if (message.stepInference !== 0) {
      writer.uint32(24).int32(message.stepInference);
    }
    for (const v of message.predecessorFactIds) {
      writer.uint32(34).string(v!);
    }
    writer.uint32(42).fork();
    for (const v of message.dependsOnSteps) {
      writer.uint32(v);
    }
    writer.ldelim();
    if (message.verdict !== 0) {
      writer.uint32(48).int32(message.verdict);
    }
    if (message.verdictNote !== "") {
      writer.uint32(58).string(message.verdictNote);
    }
    if (message.stepConfidenceBps !== BigInt(0)) {
      writer.uint32(64).uint64(message.stepConfidenceBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ReasoningStep {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseReasoningStep();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.stepIndex = reader.uint32();
          break;
        case 2:
          message.content = reader.string();
          break;
        case 3:
          message.stepInference = reader.int32() as any;
          break;
        case 4:
          message.predecessorFactIds.push(reader.string());
          break;
        case 5:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.dependsOnSteps.push(reader.uint32());
            }
          } else {
            message.dependsOnSteps.push(reader.uint32());
          }
          break;
        case 6:
          message.verdict = reader.int32() as any;
          break;
        case 7:
          message.verdictNote = reader.string();
          break;
        case 8:
          message.stepConfidenceBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ReasoningStep>): ReasoningStep {
    const message = createBaseReasoningStep();
    message.stepIndex = object.stepIndex ?? 0;
    message.content = object.content ?? "";
    message.stepInference = object.stepInference ?? 0;
    message.predecessorFactIds = object.predecessorFactIds?.map(e => e) || [];
    message.dependsOnSteps = object.dependsOnSteps?.map(e => e) || [];
    message.verdict = object.verdict ?? 0;
    message.verdictNote = object.verdictNote ?? "";
    message.stepConfidenceBps = object.stepConfidenceBps !== undefined && object.stepConfidenceBps !== null ? BigInt(object.stepConfidenceBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseDriftDiagnosis(): DriftDiagnosis {
  return {
    driftKind: 0,
    driftedAtStepIndex: 0,
    originalExcerpt: "",
    driftedExcerpt: "",
    explanation: ""
  };
}
/**
 * DriftDiagnosis carries the verifier panel's diagnosis of WHERE and HOW
 * the variant's meaning slipped. Populated on DRIFT/INFERIOR verdicts.
 * @name DriftDiagnosis
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DriftDiagnosis
 */
export const DriftDiagnosis = {
  typeUrl: "/zerone.knowledge.v1.DriftDiagnosis",
  encode(message: DriftDiagnosis, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.driftKind !== 0) {
      writer.uint32(8).int32(message.driftKind);
    }
    if (message.driftedAtStepIndex !== 0) {
      writer.uint32(16).uint32(message.driftedAtStepIndex);
    }
    if (message.originalExcerpt !== "") {
      writer.uint32(26).string(message.originalExcerpt);
    }
    if (message.driftedExcerpt !== "") {
      writer.uint32(34).string(message.driftedExcerpt);
    }
    if (message.explanation !== "") {
      writer.uint32(42).string(message.explanation);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DriftDiagnosis {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDriftDiagnosis();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.driftKind = reader.int32() as any;
          break;
        case 2:
          message.driftedAtStepIndex = reader.uint32();
          break;
        case 3:
          message.originalExcerpt = reader.string();
          break;
        case 4:
          message.driftedExcerpt = reader.string();
          break;
        case 5:
          message.explanation = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DriftDiagnosis>): DriftDiagnosis {
    const message = createBaseDriftDiagnosis();
    message.driftKind = object.driftKind ?? 0;
    message.driftedAtStepIndex = object.driftedAtStepIndex ?? 0;
    message.originalExcerpt = object.originalExcerpt ?? "";
    message.driftedExcerpt = object.driftedExcerpt ?? "";
    message.explanation = object.explanation ?? "";
    return message;
  }
};
function createBaseMethodologyChoice(): MethodologyChoice {
  return {
    chosenMethodId: "",
    consideredMethods: [],
    rationale: "",
    abandonedMethods: [],
    abandonmentReason: ""
  };
}
/**
 * @name MethodologyChoice
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MethodologyChoice
 */
export const MethodologyChoice = {
  typeUrl: "/zerone.knowledge.v1.MethodologyChoice",
  encode(message: MethodologyChoice, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.chosenMethodId !== "") {
      writer.uint32(10).string(message.chosenMethodId);
    }
    for (const v of message.consideredMethods) {
      writer.uint32(18).string(v!);
    }
    if (message.rationale !== "") {
      writer.uint32(26).string(message.rationale);
    }
    for (const v of message.abandonedMethods) {
      writer.uint32(34).string(v!);
    }
    if (message.abandonmentReason !== "") {
      writer.uint32(42).string(message.abandonmentReason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MethodologyChoice {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMethodologyChoice();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.chosenMethodId = reader.string();
          break;
        case 2:
          message.consideredMethods.push(reader.string());
          break;
        case 3:
          message.rationale = reader.string();
          break;
        case 4:
          message.abandonedMethods.push(reader.string());
          break;
        case 5:
          message.abandonmentReason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MethodologyChoice>): MethodologyChoice {
    const message = createBaseMethodologyChoice();
    message.chosenMethodId = object.chosenMethodId ?? "";
    message.consideredMethods = object.consideredMethods?.map(e => e) || [];
    message.rationale = object.rationale ?? "";
    message.abandonedMethods = object.abandonedMethods?.map(e => e) || [];
    message.abandonmentReason = object.abandonmentReason ?? "";
    return message;
  }
};
function createBaseBeliefRevision(): BeliefRevision {
  return {
    atBlock: BigInt(0),
    priorConfidenceBps: BigInt(0),
    posteriorConfidenceBps: BigInt(0),
    reason: 0,
    evidenceFactIds: [],
    evidenceClaimId: "",
    note: ""
  };
}
/**
 * @name BeliefRevision
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.BeliefRevision
 */
export const BeliefRevision = {
  typeUrl: "/zerone.knowledge.v1.BeliefRevision",
  encode(message: BeliefRevision, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.atBlock !== BigInt(0)) {
      writer.uint32(8).uint64(message.atBlock);
    }
    if (message.priorConfidenceBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.priorConfidenceBps);
    }
    if (message.posteriorConfidenceBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.posteriorConfidenceBps);
    }
    if (message.reason !== 0) {
      writer.uint32(32).int32(message.reason);
    }
    for (const v of message.evidenceFactIds) {
      writer.uint32(42).string(v!);
    }
    if (message.evidenceClaimId !== "") {
      writer.uint32(50).string(message.evidenceClaimId);
    }
    if (message.note !== "") {
      writer.uint32(58).string(message.note);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): BeliefRevision {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBeliefRevision();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.atBlock = reader.uint64();
          break;
        case 2:
          message.priorConfidenceBps = reader.uint64();
          break;
        case 3:
          message.posteriorConfidenceBps = reader.uint64();
          break;
        case 4:
          message.reason = reader.int32() as any;
          break;
        case 5:
          message.evidenceFactIds.push(reader.string());
          break;
        case 6:
          message.evidenceClaimId = reader.string();
          break;
        case 7:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<BeliefRevision>): BeliefRevision {
    const message = createBaseBeliefRevision();
    message.atBlock = object.atBlock !== undefined && object.atBlock !== null ? BigInt(object.atBlock.toString()) : BigInt(0);
    message.priorConfidenceBps = object.priorConfidenceBps !== undefined && object.priorConfidenceBps !== null ? BigInt(object.priorConfidenceBps.toString()) : BigInt(0);
    message.posteriorConfidenceBps = object.posteriorConfidenceBps !== undefined && object.posteriorConfidenceBps !== null ? BigInt(object.posteriorConfidenceBps.toString()) : BigInt(0);
    message.reason = object.reason ?? 0;
    message.evidenceFactIds = object.evidenceFactIds?.map(e => e) || [];
    message.evidenceClaimId = object.evidenceClaimId ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseDialecticNode(): DialecticNode {
  return {
    speaker: "",
    role: 0,
    argumentText: "",
    methodId: "",
    atBlock: BigInt(0),
    citedFactIds: [],
    children: [],
    nodeVerdict: 0
  };
}
/**
 * @name DialecticNode
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DialecticNode
 */
export const DialecticNode = {
  typeUrl: "/zerone.knowledge.v1.DialecticNode",
  encode(message: DialecticNode, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.speaker !== "") {
      writer.uint32(10).string(message.speaker);
    }
    if (message.role !== 0) {
      writer.uint32(16).int32(message.role);
    }
    if (message.argumentText !== "") {
      writer.uint32(26).string(message.argumentText);
    }
    if (message.methodId !== "") {
      writer.uint32(34).string(message.methodId);
    }
    if (message.atBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.atBlock);
    }
    for (const v of message.citedFactIds) {
      writer.uint32(50).string(v!);
    }
    for (const v of message.children) {
      DialecticNode.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    if (message.nodeVerdict !== 0) {
      writer.uint32(64).int32(message.nodeVerdict);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DialecticNode {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDialecticNode();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.speaker = reader.string();
          break;
        case 2:
          message.role = reader.int32() as any;
          break;
        case 3:
          message.argumentText = reader.string();
          break;
        case 4:
          message.methodId = reader.string();
          break;
        case 5:
          message.atBlock = reader.uint64();
          break;
        case 6:
          message.citedFactIds.push(reader.string());
          break;
        case 7:
          message.children.push(DialecticNode.decode(reader, reader.uint32()));
          break;
        case 8:
          message.nodeVerdict = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DialecticNode>): DialecticNode {
    const message = createBaseDialecticNode();
    message.speaker = object.speaker ?? "";
    message.role = object.role ?? 0;
    message.argumentText = object.argumentText ?? "";
    message.methodId = object.methodId ?? "";
    message.atBlock = object.atBlock !== undefined && object.atBlock !== null ? BigInt(object.atBlock.toString()) : BigInt(0);
    message.citedFactIds = object.citedFactIds?.map(e => e) || [];
    message.children = object.children?.map(e => DialecticNode.fromPartial(e)) || [];
    message.nodeVerdict = object.nodeVerdict ?? 0;
    return message;
  }
};
function createBaseTraceVindication(): TraceVindication {
  return {
    verifiers: [],
    refundTotal: "",
    vindicatedAtBlock: BigInt(0),
    disprovenByFactId: ""
  };
}
/**
 * @name TraceVindication
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceVindication
 */
export const TraceVindication = {
  typeUrl: "/zerone.knowledge.v1.TraceVindication",
  encode(message: TraceVindication, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    for (const v of message.verifiers) {
      writer.uint32(10).string(v!);
    }
    if (message.refundTotal !== "") {
      writer.uint32(18).string(message.refundTotal);
    }
    if (message.vindicatedAtBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.vindicatedAtBlock);
    }
    if (message.disprovenByFactId !== "") {
      writer.uint32(34).string(message.disprovenByFactId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TraceVindication {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTraceVindication();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifiers.push(reader.string());
          break;
        case 2:
          message.refundTotal = reader.string();
          break;
        case 3:
          message.vindicatedAtBlock = reader.uint64();
          break;
        case 4:
          message.disprovenByFactId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TraceVindication>): TraceVindication {
    const message = createBaseTraceVindication();
    message.verifiers = object.verifiers?.map(e => e) || [];
    message.refundTotal = object.refundTotal ?? "";
    message.vindicatedAtBlock = object.vindicatedAtBlock !== undefined && object.vindicatedAtBlock !== null ? BigInt(object.vindicatedAtBlock.toString()) : BigInt(0);
    message.disprovenByFactId = object.disprovenByFactId ?? "";
    return message;
  }
};
function createBaseTraceDisproval(): TraceDisproval {
  return {
    disprovenByFactId: "",
    disprovenByClaimId: "",
    methodId: "",
    disprovenAtBlock: BigInt(0),
    disproofArgument: ""
  };
}
/**
 * @name TraceDisproval
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceDisproval
 */
export const TraceDisproval = {
  typeUrl: "/zerone.knowledge.v1.TraceDisproval",
  encode(message: TraceDisproval, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.disprovenByFactId !== "") {
      writer.uint32(10).string(message.disprovenByFactId);
    }
    if (message.disprovenByClaimId !== "") {
      writer.uint32(18).string(message.disprovenByClaimId);
    }
    if (message.methodId !== "") {
      writer.uint32(26).string(message.methodId);
    }
    if (message.disprovenAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.disprovenAtBlock);
    }
    if (message.disproofArgument !== "") {
      writer.uint32(42).string(message.disproofArgument);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TraceDisproval {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTraceDisproval();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.disprovenByFactId = reader.string();
          break;
        case 2:
          message.disprovenByClaimId = reader.string();
          break;
        case 3:
          message.methodId = reader.string();
          break;
        case 4:
          message.disprovenAtBlock = reader.uint64();
          break;
        case 5:
          message.disproofArgument = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TraceDisproval>): TraceDisproval {
    const message = createBaseTraceDisproval();
    message.disprovenByFactId = object.disprovenByFactId ?? "";
    message.disprovenByClaimId = object.disprovenByClaimId ?? "";
    message.methodId = object.methodId ?? "";
    message.disprovenAtBlock = object.disprovenAtBlock !== undefined && object.disprovenAtBlock !== null ? BigInt(object.disprovenAtBlock.toString()) : BigInt(0);
    message.disproofArgument = object.disproofArgument ?? "";
    return message;
  }
};
function createBaseTraceReformulation(): TraceReformulation {
  return {
    augmentationId: "",
    variantContent: "",
    verdict: 0,
    verifierCount: 0,
    verdictBlock: BigInt(0),
    methodologyId: ""
  };
}
/**
 * @name TraceReformulation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceReformulation
 */
export const TraceReformulation = {
  typeUrl: "/zerone.knowledge.v1.TraceReformulation",
  encode(message: TraceReformulation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.augmentationId !== "") {
      writer.uint32(10).string(message.augmentationId);
    }
    if (message.variantContent !== "") {
      writer.uint32(18).string(message.variantContent);
    }
    if (message.verdict !== 0) {
      writer.uint32(24).int32(message.verdict);
    }
    if (message.verifierCount !== 0) {
      writer.uint32(32).uint32(message.verifierCount);
    }
    if (message.verdictBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.verdictBlock);
    }
    if (message.methodologyId !== "") {
      writer.uint32(50).string(message.methodologyId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TraceReformulation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTraceReformulation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.augmentationId = reader.string();
          break;
        case 2:
          message.variantContent = reader.string();
          break;
        case 3:
          message.verdict = reader.int32() as any;
          break;
        case 4:
          message.verifierCount = reader.uint32();
          break;
        case 5:
          message.verdictBlock = reader.uint64();
          break;
        case 6:
          message.methodologyId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TraceReformulation>): TraceReformulation {
    const message = createBaseTraceReformulation();
    message.augmentationId = object.augmentationId ?? "";
    message.variantContent = object.variantContent ?? "";
    message.verdict = object.verdict ?? 0;
    message.verifierCount = object.verifierCount ?? 0;
    message.verdictBlock = object.verdictBlock !== undefined && object.verdictBlock !== null ? BigInt(object.verdictBlock.toString()) : BigInt(0);
    message.methodologyId = object.methodologyId ?? "";
    return message;
  }
};
function createBaseTraceDrift(): TraceDrift {
  return {
    augmentationId: "",
    variantContent: "",
    verdict: 0,
    driftVoters: [],
    verdictBlock: BigInt(0),
    diagnosis: undefined,
    drifterSteps: []
  };
}
/**
 * @name TraceDrift
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TraceDrift
 */
export const TraceDrift = {
  typeUrl: "/zerone.knowledge.v1.TraceDrift",
  encode(message: TraceDrift, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.augmentationId !== "") {
      writer.uint32(10).string(message.augmentationId);
    }
    if (message.variantContent !== "") {
      writer.uint32(18).string(message.variantContent);
    }
    if (message.verdict !== 0) {
      writer.uint32(24).int32(message.verdict);
    }
    for (const v of message.driftVoters) {
      writer.uint32(34).string(v!);
    }
    if (message.verdictBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.verdictBlock);
    }
    if (message.diagnosis !== undefined) {
      DriftDiagnosis.encode(message.diagnosis, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.drifterSteps) {
      ReasoningStep.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TraceDrift {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTraceDrift();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.augmentationId = reader.string();
          break;
        case 2:
          message.variantContent = reader.string();
          break;
        case 3:
          message.verdict = reader.int32() as any;
          break;
        case 4:
          message.driftVoters.push(reader.string());
          break;
        case 5:
          message.verdictBlock = reader.uint64();
          break;
        case 6:
          message.diagnosis = DriftDiagnosis.decode(reader, reader.uint32());
          break;
        case 7:
          message.drifterSteps.push(ReasoningStep.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TraceDrift>): TraceDrift {
    const message = createBaseTraceDrift();
    message.augmentationId = object.augmentationId ?? "";
    message.variantContent = object.variantContent ?? "";
    message.verdict = object.verdict ?? 0;
    message.driftVoters = object.driftVoters?.map(e => e) || [];
    message.verdictBlock = object.verdictBlock !== undefined && object.verdictBlock !== null ? BigInt(object.verdictBlock.toString()) : BigInt(0);
    message.diagnosis = object.diagnosis !== undefined && object.diagnosis !== null ? DriftDiagnosis.fromPartial(object.diagnosis) : undefined;
    message.drifterSteps = object.drifterSteps?.map(e => ReasoningStep.fromPartial(e)) || [];
    return message;
  }
};
function createBaseContrastivePair(): ContrastivePair {
  return {
    pairId: "",
    pairType: 0,
    positiveFactId: "",
    positiveContent: "",
    negativeFactId: "",
    negativeAugmentationId: "",
    negativeContent: "",
    methodId: "",
    distinguishingArgument: "",
    resolvedAtBlock: BigInt(0),
    snapshotBlockHeight: BigInt(0),
    traceSchemaVersion: BigInt(0)
  };
}
/**
 * ContrastivePair is a (positive, negative, verdict) training row for
 * preference learning. ZERONE's unique lever: web crawl only has survivors;
 * this format ships the LOSING side with the adjudication that beat it.
 * @name ContrastivePair
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ContrastivePair
 */
export const ContrastivePair = {
  typeUrl: "/zerone.knowledge.v1.ContrastivePair",
  encode(message: ContrastivePair, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.pairId !== "") {
      writer.uint32(10).string(message.pairId);
    }
    if (message.pairType !== 0) {
      writer.uint32(16).int32(message.pairType);
    }
    if (message.positiveFactId !== "") {
      writer.uint32(26).string(message.positiveFactId);
    }
    if (message.positiveContent !== "") {
      writer.uint32(34).string(message.positiveContent);
    }
    if (message.negativeFactId !== "") {
      writer.uint32(42).string(message.negativeFactId);
    }
    if (message.negativeAugmentationId !== "") {
      writer.uint32(50).string(message.negativeAugmentationId);
    }
    if (message.negativeContent !== "") {
      writer.uint32(58).string(message.negativeContent);
    }
    if (message.methodId !== "") {
      writer.uint32(66).string(message.methodId);
    }
    if (message.distinguishingArgument !== "") {
      writer.uint32(74).string(message.distinguishingArgument);
    }
    if (message.resolvedAtBlock !== BigInt(0)) {
      writer.uint32(80).uint64(message.resolvedAtBlock);
    }
    if (message.snapshotBlockHeight !== BigInt(0)) {
      writer.uint32(88).uint64(message.snapshotBlockHeight);
    }
    if (message.traceSchemaVersion !== BigInt(0)) {
      writer.uint32(96).uint64(message.traceSchemaVersion);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ContrastivePair {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseContrastivePair();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pairId = reader.string();
          break;
        case 2:
          message.pairType = reader.int32() as any;
          break;
        case 3:
          message.positiveFactId = reader.string();
          break;
        case 4:
          message.positiveContent = reader.string();
          break;
        case 5:
          message.negativeFactId = reader.string();
          break;
        case 6:
          message.negativeAugmentationId = reader.string();
          break;
        case 7:
          message.negativeContent = reader.string();
          break;
        case 8:
          message.methodId = reader.string();
          break;
        case 9:
          message.distinguishingArgument = reader.string();
          break;
        case 10:
          message.resolvedAtBlock = reader.uint64();
          break;
        case 11:
          message.snapshotBlockHeight = reader.uint64();
          break;
        case 12:
          message.traceSchemaVersion = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ContrastivePair>): ContrastivePair {
    const message = createBaseContrastivePair();
    message.pairId = object.pairId ?? "";
    message.pairType = object.pairType ?? 0;
    message.positiveFactId = object.positiveFactId ?? "";
    message.positiveContent = object.positiveContent ?? "";
    message.negativeFactId = object.negativeFactId ?? "";
    message.negativeAugmentationId = object.negativeAugmentationId ?? "";
    message.negativeContent = object.negativeContent ?? "";
    message.methodId = object.methodId ?? "";
    message.distinguishingArgument = object.distinguishingArgument ?? "";
    message.resolvedAtBlock = object.resolvedAtBlock !== undefined && object.resolvedAtBlock !== null ? BigInt(object.resolvedAtBlock.toString()) : BigInt(0);
    message.snapshotBlockHeight = object.snapshotBlockHeight !== undefined && object.snapshotBlockHeight !== null ? BigInt(object.snapshotBlockHeight.toString()) : BigInt(0);
    message.traceSchemaVersion = object.traceSchemaVersion !== undefined && object.traceSchemaVersion !== null ? BigInt(object.traceSchemaVersion.toString()) : BigInt(0);
    return message;
  }
};
function createBaseTraceSchema(): TraceSchema {
  return {
    version: BigInt(0),
    ratifiedAtBlock: BigInt(0),
    jsonSchemaHash: "",
    jsonSchema: "",
    requiredFields: [],
    deprecatedFields: [],
    notes: ""
  };
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
export const TraceSchema = {
  typeUrl: "/zerone.knowledge.v1.TraceSchema",
  encode(message: TraceSchema, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.version !== BigInt(0)) {
      writer.uint32(8).uint64(message.version);
    }
    if (message.ratifiedAtBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.ratifiedAtBlock);
    }
    if (message.jsonSchemaHash !== "") {
      writer.uint32(26).string(message.jsonSchemaHash);
    }
    if (message.jsonSchema !== "") {
      writer.uint32(34).string(message.jsonSchema);
    }
    for (const v of message.requiredFields) {
      writer.uint32(42).string(v!);
    }
    for (const v of message.deprecatedFields) {
      writer.uint32(50).string(v!);
    }
    if (message.notes !== "") {
      writer.uint32(58).string(message.notes);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TraceSchema {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTraceSchema();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.version = reader.uint64();
          break;
        case 2:
          message.ratifiedAtBlock = reader.uint64();
          break;
        case 3:
          message.jsonSchemaHash = reader.string();
          break;
        case 4:
          message.jsonSchema = reader.string();
          break;
        case 5:
          message.requiredFields.push(reader.string());
          break;
        case 6:
          message.deprecatedFields.push(reader.string());
          break;
        case 7:
          message.notes = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TraceSchema>): TraceSchema {
    const message = createBaseTraceSchema();
    message.version = object.version !== undefined && object.version !== null ? BigInt(object.version.toString()) : BigInt(0);
    message.ratifiedAtBlock = object.ratifiedAtBlock !== undefined && object.ratifiedAtBlock !== null ? BigInt(object.ratifiedAtBlock.toString()) : BigInt(0);
    message.jsonSchemaHash = object.jsonSchemaHash ?? "";
    message.jsonSchema = object.jsonSchema ?? "";
    message.requiredFields = object.requiredFields?.map(e => e) || [];
    message.deprecatedFields = object.deprecatedFields?.map(e => e) || [];
    message.notes = object.notes ?? "";
    return message;
  }
};
function createBaseCorpusSelector(): CorpusSelector {
  return {
    methodId: "",
    minCorroboration: BigInt(0),
    minQualityTier: 0,
    minCurriculumTier: 0,
    includeDisproven: false,
    includeDrift: false,
    includeNormative: false,
    includeContrastivePairs: false,
    pairTypeFilter: 0,
    domainWhitelist: [],
    domainBlacklist: [],
    minSubmitterCalibrationBps: BigInt(0)
  };
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
export const CorpusSelector = {
  typeUrl: "/zerone.knowledge.v1.CorpusSelector",
  encode(message: CorpusSelector, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.methodId !== "") {
      writer.uint32(10).string(message.methodId);
    }
    if (message.minCorroboration !== BigInt(0)) {
      writer.uint32(16).uint64(message.minCorroboration);
    }
    if (message.minQualityTier !== 0) {
      writer.uint32(24).int32(message.minQualityTier);
    }
    if (message.minCurriculumTier !== 0) {
      writer.uint32(32).int32(message.minCurriculumTier);
    }
    if (message.includeDisproven === true) {
      writer.uint32(40).bool(message.includeDisproven);
    }
    if (message.includeDrift === true) {
      writer.uint32(48).bool(message.includeDrift);
    }
    if (message.includeNormative === true) {
      writer.uint32(56).bool(message.includeNormative);
    }
    if (message.includeContrastivePairs === true) {
      writer.uint32(64).bool(message.includeContrastivePairs);
    }
    if (message.pairTypeFilter !== 0) {
      writer.uint32(72).int32(message.pairTypeFilter);
    }
    for (const v of message.domainWhitelist) {
      writer.uint32(82).string(v!);
    }
    for (const v of message.domainBlacklist) {
      writer.uint32(90).string(v!);
    }
    if (message.minSubmitterCalibrationBps !== BigInt(0)) {
      writer.uint32(96).uint64(message.minSubmitterCalibrationBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CorpusSelector {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCorpusSelector();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.methodId = reader.string();
          break;
        case 2:
          message.minCorroboration = reader.uint64();
          break;
        case 3:
          message.minQualityTier = reader.int32() as any;
          break;
        case 4:
          message.minCurriculumTier = reader.int32() as any;
          break;
        case 5:
          message.includeDisproven = reader.bool();
          break;
        case 6:
          message.includeDrift = reader.bool();
          break;
        case 7:
          message.includeNormative = reader.bool();
          break;
        case 8:
          message.includeContrastivePairs = reader.bool();
          break;
        case 9:
          message.pairTypeFilter = reader.int32() as any;
          break;
        case 10:
          message.domainWhitelist.push(reader.string());
          break;
        case 11:
          message.domainBlacklist.push(reader.string());
          break;
        case 12:
          message.minSubmitterCalibrationBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CorpusSelector>): CorpusSelector {
    const message = createBaseCorpusSelector();
    message.methodId = object.methodId ?? "";
    message.minCorroboration = object.minCorroboration !== undefined && object.minCorroboration !== null ? BigInt(object.minCorroboration.toString()) : BigInt(0);
    message.minQualityTier = object.minQualityTier ?? 0;
    message.minCurriculumTier = object.minCurriculumTier ?? 0;
    message.includeDisproven = object.includeDisproven ?? false;
    message.includeDrift = object.includeDrift ?? false;
    message.includeNormative = object.includeNormative ?? false;
    message.includeContrastivePairs = object.includeContrastivePairs ?? false;
    message.pairTypeFilter = object.pairTypeFilter ?? 0;
    message.domainWhitelist = object.domainWhitelist?.map(e => e) || [];
    message.domainBlacklist = object.domainBlacklist?.map(e => e) || [];
    message.minSubmitterCalibrationBps = object.minSubmitterCalibrationBps !== undefined && object.minSubmitterCalibrationBps !== null ? BigInt(object.minSubmitterCalibrationBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseTrainingManifest(): TrainingManifest {
  return {
    manifestId: "",
    pipelineId: "",
    creator: "",
    createdAtBlock: BigInt(0),
    description: "",
    tokenizerVersion: BigInt(0),
    canonicalSerialisationVersion: BigInt(0),
    traceSchemaVersion: BigInt(0),
    methodologySetVersion: BigInt(0),
    snapshotBlockHeight: BigInt(0),
    chainId: "",
    corpusSelector: undefined,
    includedFactIds: [],
    includedTraceIds: [],
    includedPairIds: [],
    includedDriftAugmentationIds: [],
    includedNormativeCommitmentIds: [],
    merkleRoot: "",
    totalIncluded: 0,
    factCount: 0,
    traceCount: 0,
    pairCount: 0,
    driftCount: 0,
    normativeCount: 0,
    status: 0,
    finalizedAtBlock: BigInt(0),
    attestationId: "",
    attestedAtBlock: BigInt(0),
    parentManifestId: "",
    parentMerkleRoot: "",
    compositionDepth: 0
  };
}
/**
 * TrainingManifest is the atomic, verifiable unit of a training run: every
 * version pin, every included ID, one Merkle root. Distributed as JSON for
 * off-chain consumers; the on-chain record is authoritative.
 * @name TrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.TrainingManifest
 */
export const TrainingManifest = {
  typeUrl: "/zerone.knowledge.v1.TrainingManifest",
  encode(message: TrainingManifest, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.manifestId !== "") {
      writer.uint32(10).string(message.manifestId);
    }
    if (message.pipelineId !== "") {
      writer.uint32(18).string(message.pipelineId);
    }
    if (message.creator !== "") {
      writer.uint32(26).string(message.creator);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.createdAtBlock);
    }
    if (message.description !== "") {
      writer.uint32(42).string(message.description);
    }
    if (message.tokenizerVersion !== BigInt(0)) {
      writer.uint32(80).uint64(message.tokenizerVersion);
    }
    if (message.canonicalSerialisationVersion !== BigInt(0)) {
      writer.uint32(88).uint64(message.canonicalSerialisationVersion);
    }
    if (message.traceSchemaVersion !== BigInt(0)) {
      writer.uint32(96).uint64(message.traceSchemaVersion);
    }
    if (message.methodologySetVersion !== BigInt(0)) {
      writer.uint32(104).uint64(message.methodologySetVersion);
    }
    if (message.snapshotBlockHeight !== BigInt(0)) {
      writer.uint32(112).uint64(message.snapshotBlockHeight);
    }
    if (message.chainId !== "") {
      writer.uint32(122).string(message.chainId);
    }
    if (message.corpusSelector !== undefined) {
      CorpusSelector.encode(message.corpusSelector, writer.uint32(162).fork()).ldelim();
    }
    for (const v of message.includedFactIds) {
      writer.uint32(170).string(v!);
    }
    for (const v of message.includedTraceIds) {
      writer.uint32(178).string(v!);
    }
    for (const v of message.includedPairIds) {
      writer.uint32(186).string(v!);
    }
    for (const v of message.includedDriftAugmentationIds) {
      writer.uint32(194).string(v!);
    }
    for (const v of message.includedNormativeCommitmentIds) {
      writer.uint32(202).string(v!);
    }
    if (message.merkleRoot !== "") {
      writer.uint32(242).string(message.merkleRoot);
    }
    if (message.totalIncluded !== 0) {
      writer.uint32(248).uint32(message.totalIncluded);
    }
    if (message.factCount !== 0) {
      writer.uint32(320).uint32(message.factCount);
    }
    if (message.traceCount !== 0) {
      writer.uint32(328).uint32(message.traceCount);
    }
    if (message.pairCount !== 0) {
      writer.uint32(336).uint32(message.pairCount);
    }
    if (message.driftCount !== 0) {
      writer.uint32(344).uint32(message.driftCount);
    }
    if (message.normativeCount !== 0) {
      writer.uint32(352).uint32(message.normativeCount);
    }
    if (message.status !== 0) {
      writer.uint32(400).int32(message.status);
    }
    if (message.finalizedAtBlock !== BigInt(0)) {
      writer.uint32(408).uint64(message.finalizedAtBlock);
    }
    if (message.attestationId !== "") {
      writer.uint32(418).string(message.attestationId);
    }
    if (message.attestedAtBlock !== BigInt(0)) {
      writer.uint32(424).uint64(message.attestedAtBlock);
    }
    if (message.parentManifestId !== "") {
      writer.uint32(482).string(message.parentManifestId);
    }
    if (message.parentMerkleRoot !== "") {
      writer.uint32(490).string(message.parentMerkleRoot);
    }
    if (message.compositionDepth !== 0) {
      writer.uint32(496).uint32(message.compositionDepth);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TrainingManifest {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTrainingManifest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.manifestId = reader.string();
          break;
        case 2:
          message.pipelineId = reader.string();
          break;
        case 3:
          message.creator = reader.string();
          break;
        case 4:
          message.createdAtBlock = reader.uint64();
          break;
        case 5:
          message.description = reader.string();
          break;
        case 10:
          message.tokenizerVersion = reader.uint64();
          break;
        case 11:
          message.canonicalSerialisationVersion = reader.uint64();
          break;
        case 12:
          message.traceSchemaVersion = reader.uint64();
          break;
        case 13:
          message.methodologySetVersion = reader.uint64();
          break;
        case 14:
          message.snapshotBlockHeight = reader.uint64();
          break;
        case 15:
          message.chainId = reader.string();
          break;
        case 20:
          message.corpusSelector = CorpusSelector.decode(reader, reader.uint32());
          break;
        case 21:
          message.includedFactIds.push(reader.string());
          break;
        case 22:
          message.includedTraceIds.push(reader.string());
          break;
        case 23:
          message.includedPairIds.push(reader.string());
          break;
        case 24:
          message.includedDriftAugmentationIds.push(reader.string());
          break;
        case 25:
          message.includedNormativeCommitmentIds.push(reader.string());
          break;
        case 30:
          message.merkleRoot = reader.string();
          break;
        case 31:
          message.totalIncluded = reader.uint32();
          break;
        case 40:
          message.factCount = reader.uint32();
          break;
        case 41:
          message.traceCount = reader.uint32();
          break;
        case 42:
          message.pairCount = reader.uint32();
          break;
        case 43:
          message.driftCount = reader.uint32();
          break;
        case 44:
          message.normativeCount = reader.uint32();
          break;
        case 50:
          message.status = reader.int32() as any;
          break;
        case 51:
          message.finalizedAtBlock = reader.uint64();
          break;
        case 52:
          message.attestationId = reader.string();
          break;
        case 53:
          message.attestedAtBlock = reader.uint64();
          break;
        case 60:
          message.parentManifestId = reader.string();
          break;
        case 61:
          message.parentMerkleRoot = reader.string();
          break;
        case 62:
          message.compositionDepth = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TrainingManifest>): TrainingManifest {
    const message = createBaseTrainingManifest();
    message.manifestId = object.manifestId ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.creator = object.creator ?? "";
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.description = object.description ?? "";
    message.tokenizerVersion = object.tokenizerVersion !== undefined && object.tokenizerVersion !== null ? BigInt(object.tokenizerVersion.toString()) : BigInt(0);
    message.canonicalSerialisationVersion = object.canonicalSerialisationVersion !== undefined && object.canonicalSerialisationVersion !== null ? BigInt(object.canonicalSerialisationVersion.toString()) : BigInt(0);
    message.traceSchemaVersion = object.traceSchemaVersion !== undefined && object.traceSchemaVersion !== null ? BigInt(object.traceSchemaVersion.toString()) : BigInt(0);
    message.methodologySetVersion = object.methodologySetVersion !== undefined && object.methodologySetVersion !== null ? BigInt(object.methodologySetVersion.toString()) : BigInt(0);
    message.snapshotBlockHeight = object.snapshotBlockHeight !== undefined && object.snapshotBlockHeight !== null ? BigInt(object.snapshotBlockHeight.toString()) : BigInt(0);
    message.chainId = object.chainId ?? "";
    message.corpusSelector = object.corpusSelector !== undefined && object.corpusSelector !== null ? CorpusSelector.fromPartial(object.corpusSelector) : undefined;
    message.includedFactIds = object.includedFactIds?.map(e => e) || [];
    message.includedTraceIds = object.includedTraceIds?.map(e => e) || [];
    message.includedPairIds = object.includedPairIds?.map(e => e) || [];
    message.includedDriftAugmentationIds = object.includedDriftAugmentationIds?.map(e => e) || [];
    message.includedNormativeCommitmentIds = object.includedNormativeCommitmentIds?.map(e => e) || [];
    message.merkleRoot = object.merkleRoot ?? "";
    message.totalIncluded = object.totalIncluded ?? 0;
    message.factCount = object.factCount ?? 0;
    message.traceCount = object.traceCount ?? 0;
    message.pairCount = object.pairCount ?? 0;
    message.driftCount = object.driftCount ?? 0;
    message.normativeCount = object.normativeCount ?? 0;
    message.status = object.status ?? 0;
    message.finalizedAtBlock = object.finalizedAtBlock !== undefined && object.finalizedAtBlock !== null ? BigInt(object.finalizedAtBlock.toString()) : BigInt(0);
    message.attestationId = object.attestationId ?? "";
    message.attestedAtBlock = object.attestedAtBlock !== undefined && object.attestedAtBlock !== null ? BigInt(object.attestedAtBlock.toString()) : BigInt(0);
    message.parentManifestId = object.parentManifestId ?? "";
    message.parentMerkleRoot = object.parentMerkleRoot ?? "";
    message.compositionDepth = object.compositionDepth ?? 0;
    return message;
  }
};
function createBaseSeedStatus(): SeedStatus {
  return {
    methodologiesSeeded: false,
    tokenizerSpecSeeded: false,
    traceSchemaSeeded: false,
    commitmentsSeeded: false
  };
}
/**
 * SeedStatus reports which bootstrap seeds have run. A freshly-spawned
 * chain reports all false; SeedRouteB brings them all to true.
 * @name SeedStatus
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.SeedStatus
 */
export const SeedStatus = {
  typeUrl: "/zerone.knowledge.v1.SeedStatus",
  encode(message: SeedStatus, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.methodologiesSeeded === true) {
      writer.uint32(8).bool(message.methodologiesSeeded);
    }
    if (message.tokenizerSpecSeeded === true) {
      writer.uint32(16).bool(message.tokenizerSpecSeeded);
    }
    if (message.traceSchemaSeeded === true) {
      writer.uint32(24).bool(message.traceSchemaSeeded);
    }
    if (message.commitmentsSeeded === true) {
      writer.uint32(32).bool(message.commitmentsSeeded);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): SeedStatus {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSeedStatus();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.methodologiesSeeded = reader.bool();
          break;
        case 2:
          message.tokenizerSpecSeeded = reader.bool();
          break;
        case 3:
          message.traceSchemaSeeded = reader.bool();
          break;
        case 4:
          message.commitmentsSeeded = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<SeedStatus>): SeedStatus {
    const message = createBaseSeedStatus();
    message.methodologiesSeeded = object.methodologiesSeeded ?? false;
    message.tokenizerSpecSeeded = object.tokenizerSpecSeeded ?? false;
    message.traceSchemaSeeded = object.traceSchemaSeeded ?? false;
    message.commitmentsSeeded = object.commitmentsSeeded ?? false;
    return message;
  }
};
function createBaseRouteBCapabilities(): RouteBCapabilities {
  return {
    currentTokenizerVersion: BigInt(0),
    currentTraceSchemaVersion: BigInt(0),
    currentMethodologySetVersion: BigInt(0),
    methodologyCount: BigInt(0),
    factCount: BigInt(0),
    activePipelineCount: BigInt(0),
    modelCardCount: BigInt(0),
    activeBountyCount: BigInt(0),
    finalizedManifestCount: BigInt(0),
    openContributionChallengeCount: BigInt(0),
    trainingFundBalanceUzrn: "",
    trainingFundEscrowedUzrn: "",
    trainingFundVestingUzrn: "",
    availableCorpora: [],
    seedStatus: undefined,
    snapshotBlockHeight: BigInt(0),
    chainId: ""
  };
}
/**
 * RouteBCapabilities is the chain's self-description — what versions of
 * what contracts it exposes, and how much state it holds right now. The
 * first query a trainer runs against a new chain.
 * @name RouteBCapabilities
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.RouteBCapabilities
 */
export const RouteBCapabilities = {
  typeUrl: "/zerone.knowledge.v1.RouteBCapabilities",
  encode(message: RouteBCapabilities, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.currentTokenizerVersion !== BigInt(0)) {
      writer.uint32(8).uint64(message.currentTokenizerVersion);
    }
    if (message.currentTraceSchemaVersion !== BigInt(0)) {
      writer.uint32(16).uint64(message.currentTraceSchemaVersion);
    }
    if (message.currentMethodologySetVersion !== BigInt(0)) {
      writer.uint32(24).uint64(message.currentMethodologySetVersion);
    }
    if (message.methodologyCount !== BigInt(0)) {
      writer.uint32(80).uint64(message.methodologyCount);
    }
    if (message.factCount !== BigInt(0)) {
      writer.uint32(88).uint64(message.factCount);
    }
    if (message.activePipelineCount !== BigInt(0)) {
      writer.uint32(96).uint64(message.activePipelineCount);
    }
    if (message.modelCardCount !== BigInt(0)) {
      writer.uint32(104).uint64(message.modelCardCount);
    }
    if (message.activeBountyCount !== BigInt(0)) {
      writer.uint32(112).uint64(message.activeBountyCount);
    }
    if (message.finalizedManifestCount !== BigInt(0)) {
      writer.uint32(120).uint64(message.finalizedManifestCount);
    }
    if (message.openContributionChallengeCount !== BigInt(0)) {
      writer.uint32(128).uint64(message.openContributionChallengeCount);
    }
    if (message.trainingFundBalanceUzrn !== "") {
      writer.uint32(162).string(message.trainingFundBalanceUzrn);
    }
    if (message.trainingFundEscrowedUzrn !== "") {
      writer.uint32(170).string(message.trainingFundEscrowedUzrn);
    }
    if (message.trainingFundVestingUzrn !== "") {
      writer.uint32(178).string(message.trainingFundVestingUzrn);
    }
    for (const v of message.availableCorpora) {
      writer.uint32(242).string(v!);
    }
    if (message.seedStatus !== undefined) {
      SeedStatus.encode(message.seedStatus, writer.uint32(322).fork()).ldelim();
    }
    if (message.snapshotBlockHeight !== BigInt(0)) {
      writer.uint32(400).uint64(message.snapshotBlockHeight);
    }
    if (message.chainId !== "") {
      writer.uint32(410).string(message.chainId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): RouteBCapabilities {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRouteBCapabilities();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.currentTokenizerVersion = reader.uint64();
          break;
        case 2:
          message.currentTraceSchemaVersion = reader.uint64();
          break;
        case 3:
          message.currentMethodologySetVersion = reader.uint64();
          break;
        case 10:
          message.methodologyCount = reader.uint64();
          break;
        case 11:
          message.factCount = reader.uint64();
          break;
        case 12:
          message.activePipelineCount = reader.uint64();
          break;
        case 13:
          message.modelCardCount = reader.uint64();
          break;
        case 14:
          message.activeBountyCount = reader.uint64();
          break;
        case 15:
          message.finalizedManifestCount = reader.uint64();
          break;
        case 16:
          message.openContributionChallengeCount = reader.uint64();
          break;
        case 20:
          message.trainingFundBalanceUzrn = reader.string();
          break;
        case 21:
          message.trainingFundEscrowedUzrn = reader.string();
          break;
        case 22:
          message.trainingFundVestingUzrn = reader.string();
          break;
        case 30:
          message.availableCorpora.push(reader.string());
          break;
        case 40:
          message.seedStatus = SeedStatus.decode(reader, reader.uint32());
          break;
        case 50:
          message.snapshotBlockHeight = reader.uint64();
          break;
        case 51:
          message.chainId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<RouteBCapabilities>): RouteBCapabilities {
    const message = createBaseRouteBCapabilities();
    message.currentTokenizerVersion = object.currentTokenizerVersion !== undefined && object.currentTokenizerVersion !== null ? BigInt(object.currentTokenizerVersion.toString()) : BigInt(0);
    message.currentTraceSchemaVersion = object.currentTraceSchemaVersion !== undefined && object.currentTraceSchemaVersion !== null ? BigInt(object.currentTraceSchemaVersion.toString()) : BigInt(0);
    message.currentMethodologySetVersion = object.currentMethodologySetVersion !== undefined && object.currentMethodologySetVersion !== null ? BigInt(object.currentMethodologySetVersion.toString()) : BigInt(0);
    message.methodologyCount = object.methodologyCount !== undefined && object.methodologyCount !== null ? BigInt(object.methodologyCount.toString()) : BigInt(0);
    message.factCount = object.factCount !== undefined && object.factCount !== null ? BigInt(object.factCount.toString()) : BigInt(0);
    message.activePipelineCount = object.activePipelineCount !== undefined && object.activePipelineCount !== null ? BigInt(object.activePipelineCount.toString()) : BigInt(0);
    message.modelCardCount = object.modelCardCount !== undefined && object.modelCardCount !== null ? BigInt(object.modelCardCount.toString()) : BigInt(0);
    message.activeBountyCount = object.activeBountyCount !== undefined && object.activeBountyCount !== null ? BigInt(object.activeBountyCount.toString()) : BigInt(0);
    message.finalizedManifestCount = object.finalizedManifestCount !== undefined && object.finalizedManifestCount !== null ? BigInt(object.finalizedManifestCount.toString()) : BigInt(0);
    message.openContributionChallengeCount = object.openContributionChallengeCount !== undefined && object.openContributionChallengeCount !== null ? BigInt(object.openContributionChallengeCount.toString()) : BigInt(0);
    message.trainingFundBalanceUzrn = object.trainingFundBalanceUzrn ?? "";
    message.trainingFundEscrowedUzrn = object.trainingFundEscrowedUzrn ?? "";
    message.trainingFundVestingUzrn = object.trainingFundVestingUzrn ?? "";
    message.availableCorpora = object.availableCorpora?.map(e => e) || [];
    message.seedStatus = object.seedStatus !== undefined && object.seedStatus !== null ? SeedStatus.fromPartial(object.seedStatus) : undefined;
    message.snapshotBlockHeight = object.snapshotBlockHeight !== undefined && object.snapshotBlockHeight !== null ? BigInt(object.snapshotBlockHeight.toString()) : BigInt(0);
    message.chainId = object.chainId ?? "";
    return message;
  }
};
function createBaseRemediation(): Remediation {
  return {
    type: 0,
    reference: "",
    appliedAtBlock: BigInt(0),
    operator: "",
    note: ""
  };
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
export const Remediation = {
  typeUrl: "/zerone.knowledge.v1.Remediation",
  encode(message: Remediation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.type !== 0) {
      writer.uint32(8).int32(message.type);
    }
    if (message.reference !== "") {
      writer.uint32(18).string(message.reference);
    }
    if (message.appliedAtBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.appliedAtBlock);
    }
    if (message.operator !== "") {
      writer.uint32(34).string(message.operator);
    }
    if (message.note !== "") {
      writer.uint32(42).string(message.note);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Remediation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRemediation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.type = reader.int32() as any;
          break;
        case 2:
          message.reference = reader.string();
          break;
        case 3:
          message.appliedAtBlock = reader.uint64();
          break;
        case 4:
          message.operator = reader.string();
          break;
        case 5:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Remediation>): Remediation {
    const message = createBaseRemediation();
    message.type = object.type ?? 0;
    message.reference = object.reference ?? "";
    message.appliedAtBlock = object.appliedAtBlock !== undefined && object.appliedAtBlock !== null ? BigInt(object.appliedAtBlock.toString()) : BigInt(0);
    message.operator = object.operator ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseIncidentRecord(): IncidentRecord {
  return {
    id: "",
    severity: 0,
    status: 0,
    title: "",
    description: "",
    reporter: "",
    reportedAtBlock: BigInt(0),
    resolvedAtBlock: BigInt(0),
    closedAtBlock: BigInt(0),
    remediations: [],
    affectedModules: [],
    postMortemUri: "",
    slaTargetBlock: BigInt(0)
  };
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
export const IncidentRecord = {
  typeUrl: "/zerone.knowledge.v1.IncidentRecord",
  encode(message: IncidentRecord, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.severity !== 0) {
      writer.uint32(16).int32(message.severity);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(42).string(message.description);
    }
    if (message.reporter !== "") {
      writer.uint32(50).string(message.reporter);
    }
    if (message.reportedAtBlock !== BigInt(0)) {
      writer.uint32(56).uint64(message.reportedAtBlock);
    }
    if (message.resolvedAtBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.resolvedAtBlock);
    }
    if (message.closedAtBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.closedAtBlock);
    }
    for (const v of message.remediations) {
      Remediation.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    for (const v of message.affectedModules) {
      writer.uint32(90).string(v!);
    }
    if (message.postMortemUri !== "") {
      writer.uint32(98).string(message.postMortemUri);
    }
    if (message.slaTargetBlock !== BigInt(0)) {
      writer.uint32(104).uint64(message.slaTargetBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): IncidentRecord {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIncidentRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.severity = reader.int32() as any;
          break;
        case 3:
          message.status = reader.int32() as any;
          break;
        case 4:
          message.title = reader.string();
          break;
        case 5:
          message.description = reader.string();
          break;
        case 6:
          message.reporter = reader.string();
          break;
        case 7:
          message.reportedAtBlock = reader.uint64();
          break;
        case 8:
          message.resolvedAtBlock = reader.uint64();
          break;
        case 9:
          message.closedAtBlock = reader.uint64();
          break;
        case 10:
          message.remediations.push(Remediation.decode(reader, reader.uint32()));
          break;
        case 11:
          message.affectedModules.push(reader.string());
          break;
        case 12:
          message.postMortemUri = reader.string();
          break;
        case 13:
          message.slaTargetBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<IncidentRecord>): IncidentRecord {
    const message = createBaseIncidentRecord();
    message.id = object.id ?? "";
    message.severity = object.severity ?? 0;
    message.status = object.status ?? 0;
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.reporter = object.reporter ?? "";
    message.reportedAtBlock = object.reportedAtBlock !== undefined && object.reportedAtBlock !== null ? BigInt(object.reportedAtBlock.toString()) : BigInt(0);
    message.resolvedAtBlock = object.resolvedAtBlock !== undefined && object.resolvedAtBlock !== null ? BigInt(object.resolvedAtBlock.toString()) : BigInt(0);
    message.closedAtBlock = object.closedAtBlock !== undefined && object.closedAtBlock !== null ? BigInt(object.closedAtBlock.toString()) : BigInt(0);
    message.remediations = object.remediations?.map(e => Remediation.fromPartial(e)) || [];
    message.affectedModules = object.affectedModules?.map(e => e) || [];
    message.postMortemUri = object.postMortemUri ?? "";
    message.slaTargetBlock = object.slaTargetBlock !== undefined && object.slaTargetBlock !== null ? BigInt(object.slaTargetBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseModulePause(): ModulePause {
  return {
    moduleName: "",
    reason: "",
    pausedAtBlock: BigInt(0),
    pausedBy: "",
    autoUnpauseAtBlock: BigInt(0),
    incidentId: ""
  };
}
/**
 * @name ModulePause
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ModulePause
 */
export const ModulePause = {
  typeUrl: "/zerone.knowledge.v1.ModulePause",
  encode(message: ModulePause, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.moduleName !== "") {
      writer.uint32(10).string(message.moduleName);
    }
    if (message.reason !== "") {
      writer.uint32(18).string(message.reason);
    }
    if (message.pausedAtBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.pausedAtBlock);
    }
    if (message.pausedBy !== "") {
      writer.uint32(34).string(message.pausedBy);
    }
    if (message.autoUnpauseAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.autoUnpauseAtBlock);
    }
    if (message.incidentId !== "") {
      writer.uint32(50).string(message.incidentId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ModulePause {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseModulePause();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.moduleName = reader.string();
          break;
        case 2:
          message.reason = reader.string();
          break;
        case 3:
          message.pausedAtBlock = reader.uint64();
          break;
        case 4:
          message.pausedBy = reader.string();
          break;
        case 5:
          message.autoUnpauseAtBlock = reader.uint64();
          break;
        case 6:
          message.incidentId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ModulePause>): ModulePause {
    const message = createBaseModulePause();
    message.moduleName = object.moduleName ?? "";
    message.reason = object.reason ?? "";
    message.pausedAtBlock = object.pausedAtBlock !== undefined && object.pausedAtBlock !== null ? BigInt(object.pausedAtBlock.toString()) : BigInt(0);
    message.pausedBy = object.pausedBy ?? "";
    message.autoUnpauseAtBlock = object.autoUnpauseAtBlock !== undefined && object.autoUnpauseAtBlock !== null ? BigInt(object.autoUnpauseAtBlock.toString()) : BigInt(0);
    message.incidentId = object.incidentId ?? "";
    return message;
  }
};
function createBasePrivilegedAction(): PrivilegedAction {
  return {
    seq: BigInt(0),
    type: 0,
    invoker: "",
    invokedAtBlock: BigInt(0),
    target: "",
    incidentId: "",
    note: ""
  };
}
/**
 * @name PrivilegedAction
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.PrivilegedAction
 */
export const PrivilegedAction = {
  typeUrl: "/zerone.knowledge.v1.PrivilegedAction",
  encode(message: PrivilegedAction, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.seq !== BigInt(0)) {
      writer.uint32(8).uint64(message.seq);
    }
    if (message.type !== 0) {
      writer.uint32(16).int32(message.type);
    }
    if (message.invoker !== "") {
      writer.uint32(26).string(message.invoker);
    }
    if (message.invokedAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.invokedAtBlock);
    }
    if (message.target !== "") {
      writer.uint32(42).string(message.target);
    }
    if (message.incidentId !== "") {
      writer.uint32(50).string(message.incidentId);
    }
    if (message.note !== "") {
      writer.uint32(58).string(message.note);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): PrivilegedAction {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePrivilegedAction();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.seq = reader.uint64();
          break;
        case 2:
          message.type = reader.int32() as any;
          break;
        case 3:
          message.invoker = reader.string();
          break;
        case 4:
          message.invokedAtBlock = reader.uint64();
          break;
        case 5:
          message.target = reader.string();
          break;
        case 6:
          message.incidentId = reader.string();
          break;
        case 7:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<PrivilegedAction>): PrivilegedAction {
    const message = createBasePrivilegedAction();
    message.seq = object.seq !== undefined && object.seq !== null ? BigInt(object.seq.toString()) : BigInt(0);
    message.type = object.type ?? 0;
    message.invoker = object.invoker ?? "";
    message.invokedAtBlock = object.invokedAtBlock !== undefined && object.invokedAtBlock !== null ? BigInt(object.invokedAtBlock.toString()) : BigInt(0);
    message.target = object.target ?? "";
    message.incidentId = object.incidentId ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBasePendingFactInjection(): PendingFactInjection {
  return {
    id: "",
    content: "",
    domain: "",
    category: "",
    confidence: BigInt(0),
    references: [],
    proposer: "",
    proposedAtBlock: BigInt(0),
    executeAtBlock: BigInt(0)
  };
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
export const PendingFactInjection = {
  typeUrl: "/zerone.knowledge.v1.PendingFactInjection",
  encode(message: PendingFactInjection, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.content !== "") {
      writer.uint32(18).string(message.content);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    if (message.confidence !== BigInt(0)) {
      writer.uint32(40).uint64(message.confidence);
    }
    for (const v of message.references) {
      writer.uint32(50).string(v!);
    }
    if (message.proposer !== "") {
      writer.uint32(58).string(message.proposer);
    }
    if (message.proposedAtBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.proposedAtBlock);
    }
    if (message.executeAtBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.executeAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): PendingFactInjection {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePendingFactInjection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.content = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.category = reader.string();
          break;
        case 5:
          message.confidence = reader.uint64();
          break;
        case 6:
          message.references.push(reader.string());
          break;
        case 7:
          message.proposer = reader.string();
          break;
        case 8:
          message.proposedAtBlock = reader.uint64();
          break;
        case 9:
          message.executeAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<PendingFactInjection>): PendingFactInjection {
    const message = createBasePendingFactInjection();
    message.id = object.id ?? "";
    message.content = object.content ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    message.confidence = object.confidence !== undefined && object.confidence !== null ? BigInt(object.confidence.toString()) : BigInt(0);
    message.references = object.references?.map(e => e) || [];
    message.proposer = object.proposer ?? "";
    message.proposedAtBlock = object.proposedAtBlock !== undefined && object.proposedAtBlock !== null ? BigInt(object.proposedAtBlock.toString()) : BigInt(0);
    message.executeAtBlock = object.executeAtBlock !== undefined && object.executeAtBlock !== null ? BigInt(object.executeAtBlock.toString()) : BigInt(0);
    return message;
  }
};