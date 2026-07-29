//@ts-nocheck
import { ClaimType, ClaimRelation, ClaimStructure, TokenizerSpec, AugmentationVerdict, TraceSchema, CorpusSelector, IncidentSeverity, RemediationType } from "./types";
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgSubmitClaim
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitClaim
 */
export interface MsgSubmitClaim {
  submitter: string;
  factContent: string;
  domain: string;
  category: string;
  /**
   * uzrn amount
   */
  stake: string;
  /**
   * fact IDs cited
   */
  references: string[];
  partnershipId: string;
  /**
   * Optional — defaults to ASSERTION if unset
   */
  claimType: ClaimType;
  /**
   * Typed relationships to existing facts.
   * Replaces untyped `references` for new claims (references kept for backward compat).
   */
  relations: ClaimRelation[];
  /**
   * Optional structured decomposition
   */
  structure?: ClaimStructure;
  /**
   * Optional — auto-derived from structure if omitted
   */
  canonicalForm: string;
  /**
   * Request bootstrap fund sponsorship for review fee
   */
  sponsored: boolean;
}
/**
 * @name MsgSubmitClaimResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitClaimResponse
 */
export interface MsgSubmitClaimResponse {
  claimId: string;
}
/**
 * @name MsgSubmitCommitment
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitCommitment
 */
export interface MsgSubmitCommitment {
  verifier: string;
  roundId: string;
  /**
   * ComputeCommitmentHash(round_id, vote, confidence, salt) — see types/commitment.go
   */
  commitHash: Uint8Array;
}
/**
 * @name MsgSubmitCommitmentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitCommitmentResponse
 */
export interface MsgSubmitCommitmentResponse {}
/**
 * @name MsgSubmitReveal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitReveal
 */
export interface MsgSubmitReveal {
  verifier: string;
  roundId: string;
  /**
   * "accept", "reject", or "malformed"
   */
  vote: string;
  salt: Uint8Array;
  /**
   * BPS; bound to commitment via ComputeCommitmentHash
   */
  confidence: bigint;
}
/**
 * @name MsgSubmitRevealResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitRevealResponse
 */
export interface MsgSubmitRevealResponse {}
/**
 * @name MsgChallengeFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeFact
 */
export interface MsgChallengeFact {
  challenger: string;
  factId: string;
  /**
   * uzrn amount
   */
  stake: string;
  reason: string;
  evidenceIds: string[];
}
/**
 * @name MsgChallengeFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeFactResponse
 */
export interface MsgChallengeFactResponse {
  roundId: string;
}
/**
 * @name MsgAddFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddFact
 */
export interface MsgAddFact {
  authority: string;
  content: string;
  domain: string;
  category: string;
  references: string[];
  /**
   * initial confidence (0-1,000,000)
   */
  confidence: bigint;
}
/**
 * @name MsgAddFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddFactResponse
 */
export interface MsgAddFactResponse {
  factId: string;
}
/**
 * @name MsgSubmitContradiction
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitContradiction
 */
export interface MsgSubmitContradiction {
  submitter: string;
  /**
   * fact being contradicted
   */
  factId: string;
  counterClaim: string;
  /**
   * uzrn amount
   */
  stake: string;
  evidenceIds: string[];
  reason: string;
  domain: string;
  category: string;
}
/**
 * @name MsgSubmitContradictionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitContradictionResponse
 */
export interface MsgSubmitContradictionResponse {
  counterFactId: string;
}
/**
 * @name MsgPatronizeFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPatronizeFact
 */
export interface MsgPatronizeFact {
  patron: string;
  factId: string;
  /**
   * uzrn amount
   */
  amount: string;
  durationBlocks: bigint;
}
/**
 * @name MsgPatronizeFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPatronizeFactResponse
 */
export interface MsgPatronizeFactResponse {}
/**
 * @name MsgProposeDomain
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeDomain
 */
export interface MsgProposeDomain {
  proposer: string;
  name: string;
  description: string;
  stratum: string;
  /**
   * uzrn amount
   */
  stake: string;
}
/**
 * @name MsgProposeDomainResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeDomainResponse
 */
export interface MsgProposeDomainResponse {
  proposalId: string;
}
/**
 * @name MsgEndorseDomainProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgEndorseDomainProposal
 */
export interface MsgEndorseDomainProposal {
  endorser: string;
  proposalId: string;
}
/**
 * @name MsgEndorseDomainProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgEndorseDomainProposalResponse
 */
export interface MsgEndorseDomainProposalResponse {}
/**
 * @name MsgChallengeDomainProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeDomainProposal
 */
export interface MsgChallengeDomainProposal {
  challenger: string;
  proposalId: string;
  reason: string;
}
/**
 * @name MsgChallengeDomainProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeDomainProposalResponse
 */
export interface MsgChallengeDomainProposalResponse {}
/**
 * @name MsgRegisterStratum
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterStratum
 */
export interface MsgRegisterStratum {
  authority: string;
  name: string;
  description: string;
  /**
   * 0-1,000,000
   */
  confidenceCeiling: bigint;
  parentStrata: string[];
}
/**
 * @name MsgRegisterStratumResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterStratumResponse
 */
export interface MsgRegisterStratumResponse {}
/**
 * MsgPostConjecture submits an unsettled proposition. It asserts nothing:
 * the resulting Fact carries confidence 0, cites nothing, cannot be cited,
 * is excluded from the training corpus, and pays its proposer no reward.
 * The review fee is the proposer's only cost and is non-refundable, exactly
 * as for an ordinary claim.
 *
 * The panel's question is "is this well-posed and falsifiable?". A conjecture
 * with no falsification predicate — or one that is not truth-apt — is
 * returned VERDICT_MALFORMED and no Fact is created.
 * @name MsgPostConjecture
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPostConjecture
 */
export interface MsgPostConjecture {
  proposer: string;
  /**
   * the conjecture itself
   */
  statement: string;
  /**
   * what observation would kill it
   */
  falsificationPredicate: string;
  domain: string;
  category: string;
  /**
   * uzrn review fee, same schedule as a claim
   */
  stake: string;
  /**
   * optional: why this is worth asking
   */
  reasoningTrace: string;
}
/**
 * @name MsgPostConjectureResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPostConjectureResponse
 */
export interface MsgPostConjectureResponse {
  claimId: string;
  roundId: string;
}
/**
 * @name MsgChallengeProvisionalFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeProvisionalFact
 */
export interface MsgChallengeProvisionalFact {
  challenger: string;
  claimId: string;
  factId: string;
  /**
   * uzrn amount
   */
  stake: string;
  reason: string;
  evidenceIds: string[];
  counterClaim: string;
}
/**
 * @name MsgChallengeProvisionalFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeProvisionalFactResponse
 */
export interface MsgChallengeProvisionalFactResponse {
  challengeId: string;
}
/**
 * @name MsgUpdateParams
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
/**
 * @name MsgUpdateExtendedParams
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateExtendedParams
 */
export interface MsgUpdateExtendedParams {
  authority: string;
  /**
   * JSON-encoded ExtendedParams blob
   */
  paramsJson: string;
}
/**
 * @name MsgUpdateExtendedParamsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateExtendedParamsResponse
 */
export interface MsgUpdateExtendedParamsResponse {}
/**
 * @name MsgProposeResearchFund
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeResearchFund
 */
export interface MsgProposeResearchFund {
  proposer: string;
  title: string;
  description: string;
  /**
   * uzrn amount
   */
  amount: string;
  recipient: string;
  votingPeriodBlocks: bigint;
}
/**
 * @name MsgProposeResearchFundResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeResearchFundResponse
 */
export interface MsgProposeResearchFundResponse {
  proposalId: string;
}
/**
 * @name MsgVoteResearchProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteResearchProposal
 */
export interface MsgVoteResearchProposal {
  voter: string;
  proposalId: string;
  /**
   * true = yes, false = no
   */
  vote: boolean;
}
/**
 * @name MsgVoteResearchProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteResearchProposalResponse
 */
export interface MsgVoteResearchProposalResponse {}
/**
 * @name MsgExecuteResearchProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgExecuteResearchProposal
 */
export interface MsgExecuteResearchProposal {
  authority: string;
  proposalId: string;
}
/**
 * @name MsgExecuteResearchProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgExecuteResearchProposalResponse
 */
export interface MsgExecuteResearchProposalResponse {}
/**
 * @name MsgAddCommonKnowledge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddCommonKnowledge
 */
export interface MsgAddCommonKnowledge {
  authority: string;
  domain: string;
  subject: string;
  description: string;
  penaltyBps: bigint;
}
/**
 * @name MsgAddCommonKnowledgeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddCommonKnowledgeResponse
 */
export interface MsgAddCommonKnowledgeResponse {
  id: string;
}
/**
 * @name MsgRemoveCommonKnowledge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRemoveCommonKnowledge
 */
export interface MsgRemoveCommonKnowledge {
  authority: string;
  /**
   * Common knowledge entry ID
   */
  id: string;
}
/**
 * @name MsgRemoveCommonKnowledgeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRemoveCommonKnowledgeResponse
 */
export interface MsgRemoveCommonKnowledgeResponse {}
/**
 * @name MsgReportDemand
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgReportDemand
 */
export interface MsgReportDemand {
  /**
   * Context server address (whitelisted)
   */
  reporter: string;
  reports: DemandReport[];
}
/**
 * @name DemandReport
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DemandReport
 */
export interface DemandReport {
  domain: string;
  subject: string;
  /**
   * Total queries in this batch
   */
  queries: bigint;
  /**
   * How many returned results
   */
  fulfilled: bigint;
  unfulfilled: bigint;
}
/**
 * @name MsgReportDemandResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgReportDemandResponse
 */
export interface MsgReportDemandResponse {}
/**
 * MsgRateFact allows a querier to provide relevance feedback on a fact.
 * The querier must have previously queried this fact (enforced by query receipt).
 * @name MsgRateFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRateFact
 */
export interface MsgRateFact {
  /**
   * Address of the rating agent
   */
  rater: string;
  /**
   * Fact being rated
   */
  factId: string;
  /**
   * true = satisfied, false = dissatisfied
   */
  useful: boolean;
  /**
   * Optional: brief reason (max 256 chars)
   */
  memo: string;
}
/**
 * @name MsgRateFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRateFactResponse
 */
export interface MsgRateFactResponse {}
/**
 * @name MsgRegisterTrainingPipeline
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterTrainingPipeline
 */
export interface MsgRegisterTrainingPipeline {
  operator: string;
  id: string;
  corpusSnapshotHeight: bigint;
  tokenizerVersion: bigint;
  methodologySetVersion: bigint;
  recipeHash: string;
  description: string;
  corpusFilter: string;
}
/**
 * @name MsgRegisterTrainingPipelineResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterTrainingPipelineResponse
 */
export interface MsgRegisterTrainingPipelineResponse {}
/**
 * @name MsgUpdateTrainingPipeline
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateTrainingPipeline
 */
export interface MsgUpdateTrainingPipeline {
  operator: string;
  id: string;
  /**
   * new_status: "declared" | "running" | "completed" | "failed" | "superseded"
   */
  newStatus: string;
  /**
   * set when transitioning to "completed"
   */
  completedAtBlock: bigint;
}
/**
 * @name MsgUpdateTrainingPipelineResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateTrainingPipelineResponse
 */
export interface MsgUpdateTrainingPipelineResponse {}
/**
 * @name MsgRegisterModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterModelCard
 */
export interface MsgRegisterModelCard {
  owner: string;
  id: string;
  name: string;
  pipelineId: string;
  deploymentAddress: string;
  parameterCount: bigint;
  /**
   * "openweight_fine_tune" | "from_scratch" | "distilled"
   */
  route: string;
  baseModel: string;
  evalAcceptanceRateBps: bigint;
  evalCorroborationRateBps: bigint;
  evalSampleSize: bigint;
  specialisedMethodId: string;
}
/**
 * @name MsgRegisterModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterModelCardResponse
 */
export interface MsgRegisterModelCardResponse {}
/**
 * @name MsgUpdateModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateModelCard
 */
export interface MsgUpdateModelCard {
  owner: string;
  id: string;
  evalAcceptanceRateBps: bigint;
  evalCorroborationRateBps: bigint;
  evalSampleSize: bigint;
  /**
   * optional re-naming
   */
  name: string;
}
/**
 * @name MsgUpdateModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateModelCardResponse
 */
export interface MsgUpdateModelCardResponse {}
/**
 * @name MsgRetireModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRetireModelCard
 */
export interface MsgRetireModelCard {
  owner: string;
  id: string;
  reason: string;
}
/**
 * @name MsgRetireModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRetireModelCardResponse
 */
export interface MsgRetireModelCardResponse {}
/**
 * @name MsgAmendTokenizerSpec
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTokenizerSpec
 */
export interface MsgAmendTokenizerSpec {
  authority: string;
  /**
   * new spec; version auto-assigned as current+1
   */
  spec?: TokenizerSpec;
}
/**
 * @name MsgAmendTokenizerSpecResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTokenizerSpecResponse
 */
export interface MsgAmendTokenizerSpecResponse {
  newVersion: bigint;
}
/**
 * @name MsgAttributeContributions
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttributeContributions
 */
export interface MsgAttributeContributions {
  owner: string;
  modelId: string;
  factIds: string[];
  /**
   * optional; 0 = sum of per-fact corroboration
   */
  totalWeight: bigint;
}
/**
 * @name MsgAttributeContributionsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttributeContributionsResponse
 */
export interface MsgAttributeContributionsResponse {
  recorded: number;
}
/**
 * @name MsgAttestTraining
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttestTraining
 */
export interface MsgAttestTraining {
  /**
   * must match the pipeline's operator
   */
  attester: string;
  pipelineId: string;
  flopsEstimate: bigint;
  wallclockSeconds: bigint;
  evalHash: string;
  signature: string;
  notes: string;
}
/**
 * @name MsgAttestTrainingResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttestTrainingResponse
 */
export interface MsgAttestTrainingResponse {}
/**
 * @name MsgCreateAugmentationBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateAugmentationBounty
 */
export interface MsgCreateAugmentationBounty {
  sponsor: string;
  id: string;
  targetFactId: string;
  rewardPerVariant: bigint;
  maxVariants: number;
  expiresAtBlock: bigint;
  description: string;
}
/**
 * @name MsgCreateAugmentationBountyResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateAugmentationBountyResponse
 */
export interface MsgCreateAugmentationBountyResponse {}
/**
 * @name MsgSubmitAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitAugmentation
 */
export interface MsgSubmitAugmentation {
  submitter: string;
  id: string;
  /**
   * optional — empty for volunteer submissions
   */
  bountyId: string;
  originalFactId: string;
  variantContent: string;
  variantReasoningTrace: string;
}
/**
 * @name MsgSubmitAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitAugmentationResponse
 */
export interface MsgSubmitAugmentationResponse {}
/**
 * @name MsgAcceptAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAcceptAugmentation
 */
export interface MsgAcceptAugmentation {
  acceptor: string;
  augmentationId: string;
  note: string;
}
/**
 * @name MsgAcceptAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAcceptAugmentationResponse
 */
export interface MsgAcceptAugmentationResponse {}
/**
 * ─── Wave 4: reformulation verdicts ───────────────────────────────────────
 * @name MsgVoteOnAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteOnAugmentation
 */
export interface MsgVoteOnAugmentation {
  verifier: string;
  augmentationId: string;
  vote: AugmentationVerdict;
  /**
   * short verifier note (optional)
   */
  rationale: string;
}
/**
 * @name MsgVoteOnAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteOnAugmentationResponse
 */
export interface MsgVoteOnAugmentationResponse {
  verdictFinalized: boolean;
  finalizedVerdict: AugmentationVerdict;
}
/**
 * @name MsgSponsorVetoAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSponsorVetoAugmentation
 */
export interface MsgSponsorVetoAugmentation {
  sponsor: string;
  augmentationId: string;
  reason: string;
}
/**
 * @name MsgSponsorVetoAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSponsorVetoAugmentationResponse
 */
export interface MsgSponsorVetoAugmentationResponse {}
/**
 * ─── Wave 4: attribution challenges ───────────────────────────────────────
 * @name MsgChallengeContribution
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeContribution
 */
export interface MsgChallengeContribution {
  challenger: string;
  modelId: string;
  disputedFactId: string;
  /**
   * "missing" | "fraudulent"
   */
  disputeType: string;
  evidence: string;
  /**
   * unique challenge id (client-chosen)
   */
  id: string;
}
/**
 * @name MsgChallengeContributionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeContributionResponse
 */
export interface MsgChallengeContributionResponse {
  bondEscrowed: string;
}
/**
 * @name MsgResolveContributionChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveContributionChallenge
 */
export interface MsgResolveContributionChallenge {
  resolver: string;
  challengeId: string;
  /**
   * true = challenge succeeds; false = rejected
   */
  uphold: boolean;
  note: string;
}
/**
 * @name MsgResolveContributionChallengeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveContributionChallengeResponse
 */
export interface MsgResolveContributionChallengeResponse {
  /**
   * uzrn the winner received
   */
  payoutToWinner: string;
}
/**
 * ─── Wave 4: training fund post-hoc disbursement ──────────────────────────
 * @name MsgClaimTrainingFundDisbursement
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgClaimTrainingFundDisbursement
 */
export interface MsgClaimTrainingFundDisbursement {
  claimant: string;
  modelId: string;
  /**
   * unique disbursement id (client-chosen)
   */
  id: string;
}
/**
 * @name MsgClaimTrainingFundDisbursementResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgClaimTrainingFundDisbursementResponse
 */
export interface MsgClaimTrainingFundDisbursementResponse {
  totalAmount: string;
  /**
   * paid immediately
   */
  releasedAmount: string;
  /**
   * held for vesting_end_block
   */
  vestingAmount: string;
  vestingEndBlock: bigint;
}
/**
 * ─── Route B Wave 5: trace schema amendment ───────────────────────────────
 * @name MsgAmendTraceSchema
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTraceSchema
 */
export interface MsgAmendTraceSchema {
  authority: string;
  /**
   * new schema; version auto-assigned as current+1
   */
  schema?: TraceSchema;
}
/**
 * @name MsgAmendTraceSchemaResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTraceSchemaResponse
 */
export interface MsgAmendTraceSchemaResponse {
  newVersion: bigint;
}
/**
 * ─── Route B Wave 7: training manifests ──────────────────────────────────
 * @name MsgCreateTrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateTrainingManifest
 */
export interface MsgCreateTrainingManifest {
  /**
   * pipeline operator
   */
  creator: string;
  /**
   * client-chosen manifest id
   */
  id: string;
  pipelineId: string;
  corpusSelector?: CorpusSelector;
  description: string;
  /**
   * Wave 8: optional parent manifest this run builds on. When set, the
   * child's corpus_selector is applied ONLY over IDs NOT already present
   * in the parent — the child captures the delta. Bundle assembly unions.
   */
  parentManifestId: string;
}
/**
 * @name MsgCreateTrainingManifestResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateTrainingManifestResponse
 */
export interface MsgCreateTrainingManifestResponse {
  totalIncluded: number;
  factCount: number;
  traceCount: number;
  pairCount: number;
  driftCount: number;
  normativeCount: number;
}
/**
 * @name MsgFinalizeTrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgFinalizeTrainingManifest
 */
export interface MsgFinalizeTrainingManifest {
  /**
   * must match the manifest creator
   */
  creator: string;
  manifestId: string;
}
/**
 * @name MsgFinalizeTrainingManifestResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgFinalizeTrainingManifestResponse
 */
export interface MsgFinalizeTrainingManifestResponse {
  /**
   * the locked commitment
   */
  merkleRoot: string;
}
/**
 * @name MsgBindManifestToAttestation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgBindManifestToAttestation
 */
export interface MsgBindManifestToAttestation {
  creator: string;
  manifestId: string;
  /**
   * references TrainingAttestation.pipeline_id (1:1 with pipeline)
   */
  attestationId: string;
}
/**
 * @name MsgBindManifestToAttestationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgBindManifestToAttestationResponse
 */
export interface MsgBindManifestToAttestationResponse {}
/**
 * ─── Route B Wave 11: incident response ──────────────────────────────────
 * @name MsgOpenIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgOpenIncident
 */
export interface MsgOpenIncident {
  authority: string;
  id: string;
  severity: IncidentSeverity;
  title: string;
  description: string;
  reporter: string;
  affectedModules: string[];
  /**
   * Optional override of the SLA window (blocks). 0 → use the default
   * inferred from severity.
   */
  slaWindowBlocks: bigint;
}
/**
 * @name MsgOpenIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgOpenIncidentResponse
 */
export interface MsgOpenIncidentResponse {
  slaTargetBlock: bigint;
}
/**
 * @name MsgRecordRemediation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRecordRemediation
 */
export interface MsgRecordRemediation {
  authority: string;
  incidentId: string;
  type: RemediationType;
  reference: string;
  note: string;
}
/**
 * @name MsgRecordRemediationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRecordRemediationResponse
 */
export interface MsgRecordRemediationResponse {
  totalRemediations: number;
}
/**
 * @name MsgResolveIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveIncident
 */
export interface MsgResolveIncident {
  authority: string;
  incidentId: string;
  postMortemUri: string;
}
/**
 * @name MsgResolveIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveIncidentResponse
 */
export interface MsgResolveIncidentResponse {}
/**
 * @name MsgCloseIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCloseIncident
 */
export interface MsgCloseIncident {
  authority: string;
  incidentId: string;
}
/**
 * @name MsgCloseIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCloseIncidentResponse
 */
export interface MsgCloseIncidentResponse {}
/**
 * ─── Route B Wave 12: module circuit breakers ────────────────────────────
 * @name MsgPauseModule
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPauseModule
 */
export interface MsgPauseModule {
  authority: string;
  moduleName: string;
  reason: string;
  /**
   * 0 = no auto-unpause
   */
  autoUnpauseAtBlock: bigint;
  /**
   * optional incident binding
   */
  incidentId: string;
}
/**
 * @name MsgPauseModuleResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPauseModuleResponse
 */
export interface MsgPauseModuleResponse {
  pausedAtBlock: bigint;
}
/**
 * @name MsgUnpauseModule
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUnpauseModule
 */
export interface MsgUnpauseModule {
  authority: string;
  moduleName: string;
  note: string;
}
/**
 * @name MsgUnpauseModuleResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUnpauseModuleResponse
 */
export interface MsgUnpauseModuleResponse {}
/**
 * @name MsgCorrectManifestMerkleRoot
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCorrectManifestMerkleRoot
 */
export interface MsgCorrectManifestMerkleRoot {
  authority: string;
  manifestId: string;
  /**
   * Required: the incident_id this correction is recorded under. A
   * correction without an open incident is rejected so no authority-gated
   * state edit can happen without an audit trail.
   */
  incidentId: string;
  /**
   * Optional caller-provided expected root. When set, the handler asserts
   * the recomputed root matches — prevents the handler from silently
   * overwriting a NON-corrupted manifest due to an operator error.
   */
  expectedRecomputedRoot: string;
  note: string;
}
/**
 * @name MsgCorrectManifestMerkleRootResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCorrectManifestMerkleRootResponse
 */
export interface MsgCorrectManifestMerkleRootResponse {
  /**
   * what was there before correction
   */
  priorRoot: string;
  /**
   * what the handler wrote
   */
  recomputedRoot: string;
  /**
   * true iff prior != recomputed (else no-op)
   */
  wasCorrupted: boolean;
}
/**
 * MsgVetoFactInjection — a registered guardian cancels a pending
 * authority-injected fact during the veto window. The fact never
 * materializes; the privileged-action log records the veto for audit.
 * @name MsgVetoFactInjection
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVetoFactInjection
 */
export interface MsgVetoFactInjection {
  /**
   * must appear in Params.guardian_addresses
   */
  guardian: string;
  /**
   * the PendingFactInjection.id to cancel
   */
  pendingId: string;
  /**
   * free-form audit comment
   */
  reason: string;
}
/**
 * @name MsgVetoFactInjectionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVetoFactInjectionResponse
 */
export interface MsgVetoFactInjectionResponse {}
function createBaseMsgSubmitClaim(): MsgSubmitClaim {
  return {
    submitter: "",
    factContent: "",
    domain: "",
    category: "",
    stake: "",
    references: [],
    partnershipId: "",
    claimType: 0,
    relations: [],
    structure: undefined,
    canonicalForm: "",
    sponsored: false
  };
}
/**
 * @name MsgSubmitClaim
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitClaim
 */
export const MsgSubmitClaim = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitClaim",
  encode(message: MsgSubmitClaim, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
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
    if (message.stake !== "") {
      writer.uint32(42).string(message.stake);
    }
    for (const v of message.references) {
      writer.uint32(50).string(v!);
    }
    if (message.partnershipId !== "") {
      writer.uint32(58).string(message.partnershipId);
    }
    if (message.claimType !== 0) {
      writer.uint32(64).int32(message.claimType);
    }
    for (const v of message.relations) {
      ClaimRelation.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    if (message.structure !== undefined) {
      ClaimStructure.encode(message.structure, writer.uint32(82).fork()).ldelim();
    }
    if (message.canonicalForm !== "") {
      writer.uint32(90).string(message.canonicalForm);
    }
    if (message.sponsored === true) {
      writer.uint32(96).bool(message.sponsored);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitClaim {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitClaim();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
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
          message.stake = reader.string();
          break;
        case 6:
          message.references.push(reader.string());
          break;
        case 7:
          message.partnershipId = reader.string();
          break;
        case 8:
          message.claimType = reader.int32() as any;
          break;
        case 9:
          message.relations.push(ClaimRelation.decode(reader, reader.uint32()));
          break;
        case 10:
          message.structure = ClaimStructure.decode(reader, reader.uint32());
          break;
        case 11:
          message.canonicalForm = reader.string();
          break;
        case 12:
          message.sponsored = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitClaim>): MsgSubmitClaim {
    const message = createBaseMsgSubmitClaim();
    message.submitter = object.submitter ?? "";
    message.factContent = object.factContent ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    message.stake = object.stake ?? "";
    message.references = object.references?.map(e => e) || [];
    message.partnershipId = object.partnershipId ?? "";
    message.claimType = object.claimType ?? 0;
    message.relations = object.relations?.map(e => ClaimRelation.fromPartial(e)) || [];
    message.structure = object.structure !== undefined && object.structure !== null ? ClaimStructure.fromPartial(object.structure) : undefined;
    message.canonicalForm = object.canonicalForm ?? "";
    message.sponsored = object.sponsored ?? false;
    return message;
  }
};
function createBaseMsgSubmitClaimResponse(): MsgSubmitClaimResponse {
  return {
    claimId: ""
  };
}
/**
 * @name MsgSubmitClaimResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitClaimResponse
 */
export const MsgSubmitClaimResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitClaimResponse",
  encode(message: MsgSubmitClaimResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.claimId !== "") {
      writer.uint32(10).string(message.claimId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitClaimResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitClaimResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitClaimResponse>): MsgSubmitClaimResponse {
    const message = createBaseMsgSubmitClaimResponse();
    message.claimId = object.claimId ?? "";
    return message;
  }
};
function createBaseMsgSubmitCommitment(): MsgSubmitCommitment {
  return {
    verifier: "",
    roundId: "",
    commitHash: new Uint8Array()
  };
}
/**
 * @name MsgSubmitCommitment
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitCommitment
 */
export const MsgSubmitCommitment = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitment",
  encode(message: MsgSubmitCommitment, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.roundId !== "") {
      writer.uint32(18).string(message.roundId);
    }
    if (message.commitHash.length !== 0) {
      writer.uint32(26).bytes(message.commitHash);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitCommitment {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitCommitment();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.roundId = reader.string();
          break;
        case 3:
          message.commitHash = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitCommitment>): MsgSubmitCommitment {
    const message = createBaseMsgSubmitCommitment();
    message.verifier = object.verifier ?? "";
    message.roundId = object.roundId ?? "";
    message.commitHash = object.commitHash ?? new Uint8Array();
    return message;
  }
};
function createBaseMsgSubmitCommitmentResponse(): MsgSubmitCommitmentResponse {
  return {};
}
/**
 * @name MsgSubmitCommitmentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitCommitmentResponse
 */
export const MsgSubmitCommitmentResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitmentResponse",
  encode(_: MsgSubmitCommitmentResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitCommitmentResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitCommitmentResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgSubmitCommitmentResponse>): MsgSubmitCommitmentResponse {
    const message = createBaseMsgSubmitCommitmentResponse();
    return message;
  }
};
function createBaseMsgSubmitReveal(): MsgSubmitReveal {
  return {
    verifier: "",
    roundId: "",
    vote: "",
    salt: new Uint8Array(),
    confidence: BigInt(0)
  };
}
/**
 * @name MsgSubmitReveal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitReveal
 */
export const MsgSubmitReveal = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitReveal",
  encode(message: MsgSubmitReveal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.roundId !== "") {
      writer.uint32(18).string(message.roundId);
    }
    if (message.vote !== "") {
      writer.uint32(26).string(message.vote);
    }
    if (message.salt.length !== 0) {
      writer.uint32(34).bytes(message.salt);
    }
    if (message.confidence !== BigInt(0)) {
      writer.uint32(40).uint64(message.confidence);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitReveal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitReveal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.roundId = reader.string();
          break;
        case 3:
          message.vote = reader.string();
          break;
        case 4:
          message.salt = reader.bytes();
          break;
        case 5:
          message.confidence = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitReveal>): MsgSubmitReveal {
    const message = createBaseMsgSubmitReveal();
    message.verifier = object.verifier ?? "";
    message.roundId = object.roundId ?? "";
    message.vote = object.vote ?? "";
    message.salt = object.salt ?? new Uint8Array();
    message.confidence = object.confidence !== undefined && object.confidence !== null ? BigInt(object.confidence.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgSubmitRevealResponse(): MsgSubmitRevealResponse {
  return {};
}
/**
 * @name MsgSubmitRevealResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitRevealResponse
 */
export const MsgSubmitRevealResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitRevealResponse",
  encode(_: MsgSubmitRevealResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitRevealResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitRevealResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgSubmitRevealResponse>): MsgSubmitRevealResponse {
    const message = createBaseMsgSubmitRevealResponse();
    return message;
  }
};
function createBaseMsgChallengeFact(): MsgChallengeFact {
  return {
    challenger: "",
    factId: "",
    stake: "",
    reason: "",
    evidenceIds: []
  };
}
/**
 * @name MsgChallengeFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeFact
 */
export const MsgChallengeFact = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeFact",
  encode(message: MsgChallengeFact, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.stake !== "") {
      writer.uint32(26).string(message.stake);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    for (const v of message.evidenceIds) {
      writer.uint32(42).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeFact {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.stake = reader.string();
          break;
        case 4:
          message.reason = reader.string();
          break;
        case 5:
          message.evidenceIds.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgChallengeFact>): MsgChallengeFact {
    const message = createBaseMsgChallengeFact();
    message.challenger = object.challenger ?? "";
    message.factId = object.factId ?? "";
    message.stake = object.stake ?? "";
    message.reason = object.reason ?? "";
    message.evidenceIds = object.evidenceIds?.map(e => e) || [];
    return message;
  }
};
function createBaseMsgChallengeFactResponse(): MsgChallengeFactResponse {
  return {
    roundId: ""
  };
}
/**
 * @name MsgChallengeFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeFactResponse
 */
export const MsgChallengeFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeFactResponse",
  encode(message: MsgChallengeFactResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.roundId !== "") {
      writer.uint32(10).string(message.roundId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeFactResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeFactResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.roundId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgChallengeFactResponse>): MsgChallengeFactResponse {
    const message = createBaseMsgChallengeFactResponse();
    message.roundId = object.roundId ?? "";
    return message;
  }
};
function createBaseMsgAddFact(): MsgAddFact {
  return {
    authority: "",
    content: "",
    domain: "",
    category: "",
    references: [],
    confidence: BigInt(0)
  };
}
/**
 * @name MsgAddFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddFact
 */
export const MsgAddFact = {
  typeUrl: "/zerone.knowledge.v1.MsgAddFact",
  encode(message: MsgAddFact, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
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
    for (const v of message.references) {
      writer.uint32(42).string(v!);
    }
    if (message.confidence !== BigInt(0)) {
      writer.uint32(48).uint64(message.confidence);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddFact {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
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
          message.references.push(reader.string());
          break;
        case 6:
          message.confidence = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddFact>): MsgAddFact {
    const message = createBaseMsgAddFact();
    message.authority = object.authority ?? "";
    message.content = object.content ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    message.references = object.references?.map(e => e) || [];
    message.confidence = object.confidence !== undefined && object.confidence !== null ? BigInt(object.confidence.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAddFactResponse(): MsgAddFactResponse {
  return {
    factId: ""
  };
}
/**
 * @name MsgAddFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddFactResponse
 */
export const MsgAddFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAddFactResponse",
  encode(message: MsgAddFactResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.factId !== "") {
      writer.uint32(10).string(message.factId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddFactResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddFactResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.factId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddFactResponse>): MsgAddFactResponse {
    const message = createBaseMsgAddFactResponse();
    message.factId = object.factId ?? "";
    return message;
  }
};
function createBaseMsgSubmitContradiction(): MsgSubmitContradiction {
  return {
    submitter: "",
    factId: "",
    counterClaim: "",
    stake: "",
    evidenceIds: [],
    reason: "",
    domain: "",
    category: ""
  };
}
/**
 * @name MsgSubmitContradiction
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitContradiction
 */
export const MsgSubmitContradiction = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitContradiction",
  encode(message: MsgSubmitContradiction, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.counterClaim !== "") {
      writer.uint32(26).string(message.counterClaim);
    }
    if (message.stake !== "") {
      writer.uint32(34).string(message.stake);
    }
    for (const v of message.evidenceIds) {
      writer.uint32(42).string(v!);
    }
    if (message.reason !== "") {
      writer.uint32(50).string(message.reason);
    }
    if (message.domain !== "") {
      writer.uint32(58).string(message.domain);
    }
    if (message.category !== "") {
      writer.uint32(66).string(message.category);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitContradiction {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitContradiction();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.counterClaim = reader.string();
          break;
        case 4:
          message.stake = reader.string();
          break;
        case 5:
          message.evidenceIds.push(reader.string());
          break;
        case 6:
          message.reason = reader.string();
          break;
        case 7:
          message.domain = reader.string();
          break;
        case 8:
          message.category = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitContradiction>): MsgSubmitContradiction {
    const message = createBaseMsgSubmitContradiction();
    message.submitter = object.submitter ?? "";
    message.factId = object.factId ?? "";
    message.counterClaim = object.counterClaim ?? "";
    message.stake = object.stake ?? "";
    message.evidenceIds = object.evidenceIds?.map(e => e) || [];
    message.reason = object.reason ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    return message;
  }
};
function createBaseMsgSubmitContradictionResponse(): MsgSubmitContradictionResponse {
  return {
    counterFactId: ""
  };
}
/**
 * @name MsgSubmitContradictionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitContradictionResponse
 */
export const MsgSubmitContradictionResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitContradictionResponse",
  encode(message: MsgSubmitContradictionResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.counterFactId !== "") {
      writer.uint32(10).string(message.counterFactId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitContradictionResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitContradictionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.counterFactId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitContradictionResponse>): MsgSubmitContradictionResponse {
    const message = createBaseMsgSubmitContradictionResponse();
    message.counterFactId = object.counterFactId ?? "";
    return message;
  }
};
function createBaseMsgPatronizeFact(): MsgPatronizeFact {
  return {
    patron: "",
    factId: "",
    amount: "",
    durationBlocks: BigInt(0)
  };
}
/**
 * @name MsgPatronizeFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPatronizeFact
 */
export const MsgPatronizeFact = {
  typeUrl: "/zerone.knowledge.v1.MsgPatronizeFact",
  encode(message: MsgPatronizeFact, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.patron !== "") {
      writer.uint32(10).string(message.patron);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.durationBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.durationBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPatronizeFact {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPatronizeFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.patron = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.durationBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgPatronizeFact>): MsgPatronizeFact {
    const message = createBaseMsgPatronizeFact();
    message.patron = object.patron ?? "";
    message.factId = object.factId ?? "";
    message.amount = object.amount ?? "";
    message.durationBlocks = object.durationBlocks !== undefined && object.durationBlocks !== null ? BigInt(object.durationBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgPatronizeFactResponse(): MsgPatronizeFactResponse {
  return {};
}
/**
 * @name MsgPatronizeFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPatronizeFactResponse
 */
export const MsgPatronizeFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgPatronizeFactResponse",
  encode(_: MsgPatronizeFactResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPatronizeFactResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPatronizeFactResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgPatronizeFactResponse>): MsgPatronizeFactResponse {
    const message = createBaseMsgPatronizeFactResponse();
    return message;
  }
};
function createBaseMsgProposeDomain(): MsgProposeDomain {
  return {
    proposer: "",
    name: "",
    description: "",
    stratum: "",
    stake: ""
  };
}
/**
 * @name MsgProposeDomain
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeDomain
 */
export const MsgProposeDomain = {
  typeUrl: "/zerone.knowledge.v1.MsgProposeDomain",
  encode(message: MsgProposeDomain, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.stratum !== "") {
      writer.uint32(34).string(message.stratum);
    }
    if (message.stake !== "") {
      writer.uint32(42).string(message.stake);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeDomain {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.stratum = reader.string();
          break;
        case 5:
          message.stake = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeDomain>): MsgProposeDomain {
    const message = createBaseMsgProposeDomain();
    message.proposer = object.proposer ?? "";
    message.name = object.name ?? "";
    message.description = object.description ?? "";
    message.stratum = object.stratum ?? "";
    message.stake = object.stake ?? "";
    return message;
  }
};
function createBaseMsgProposeDomainResponse(): MsgProposeDomainResponse {
  return {
    proposalId: ""
  };
}
/**
 * @name MsgProposeDomainResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeDomainResponse
 */
export const MsgProposeDomainResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgProposeDomainResponse",
  encode(message: MsgProposeDomainResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeDomainResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeDomainResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeDomainResponse>): MsgProposeDomainResponse {
    const message = createBaseMsgProposeDomainResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgEndorseDomainProposal(): MsgEndorseDomainProposal {
  return {
    endorser: "",
    proposalId: ""
  };
}
/**
 * @name MsgEndorseDomainProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgEndorseDomainProposal
 */
export const MsgEndorseDomainProposal = {
  typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposal",
  encode(message: MsgEndorseDomainProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.endorser !== "") {
      writer.uint32(10).string(message.endorser);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgEndorseDomainProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgEndorseDomainProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.endorser = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgEndorseDomainProposal>): MsgEndorseDomainProposal {
    const message = createBaseMsgEndorseDomainProposal();
    message.endorser = object.endorser ?? "";
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgEndorseDomainProposalResponse(): MsgEndorseDomainProposalResponse {
  return {};
}
/**
 * @name MsgEndorseDomainProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgEndorseDomainProposalResponse
 */
export const MsgEndorseDomainProposalResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposalResponse",
  encode(_: MsgEndorseDomainProposalResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgEndorseDomainProposalResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgEndorseDomainProposalResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgEndorseDomainProposalResponse>): MsgEndorseDomainProposalResponse {
    const message = createBaseMsgEndorseDomainProposalResponse();
    return message;
  }
};
function createBaseMsgChallengeDomainProposal(): MsgChallengeDomainProposal {
  return {
    challenger: "",
    proposalId: "",
    reason: ""
  };
}
/**
 * @name MsgChallengeDomainProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeDomainProposal
 */
export const MsgChallengeDomainProposal = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposal",
  encode(message: MsgChallengeDomainProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeDomainProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeDomainProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgChallengeDomainProposal>): MsgChallengeDomainProposal {
    const message = createBaseMsgChallengeDomainProposal();
    message.challenger = object.challenger ?? "";
    message.proposalId = object.proposalId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgChallengeDomainProposalResponse(): MsgChallengeDomainProposalResponse {
  return {};
}
/**
 * @name MsgChallengeDomainProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeDomainProposalResponse
 */
export const MsgChallengeDomainProposalResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposalResponse",
  encode(_: MsgChallengeDomainProposalResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeDomainProposalResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeDomainProposalResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgChallengeDomainProposalResponse>): MsgChallengeDomainProposalResponse {
    const message = createBaseMsgChallengeDomainProposalResponse();
    return message;
  }
};
function createBaseMsgRegisterStratum(): MsgRegisterStratum {
  return {
    authority: "",
    name: "",
    description: "",
    confidenceCeiling: BigInt(0),
    parentStrata: []
  };
}
/**
 * @name MsgRegisterStratum
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterStratum
 */
export const MsgRegisterStratum = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterStratum",
  encode(message: MsgRegisterStratum, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.confidenceCeiling !== BigInt(0)) {
      writer.uint32(32).uint64(message.confidenceCeiling);
    }
    for (const v of message.parentStrata) {
      writer.uint32(42).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterStratum {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterStratum();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.confidenceCeiling = reader.uint64();
          break;
        case 5:
          message.parentStrata.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterStratum>): MsgRegisterStratum {
    const message = createBaseMsgRegisterStratum();
    message.authority = object.authority ?? "";
    message.name = object.name ?? "";
    message.description = object.description ?? "";
    message.confidenceCeiling = object.confidenceCeiling !== undefined && object.confidenceCeiling !== null ? BigInt(object.confidenceCeiling.toString()) : BigInt(0);
    message.parentStrata = object.parentStrata?.map(e => e) || [];
    return message;
  }
};
function createBaseMsgRegisterStratumResponse(): MsgRegisterStratumResponse {
  return {};
}
/**
 * @name MsgRegisterStratumResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterStratumResponse
 */
export const MsgRegisterStratumResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterStratumResponse",
  encode(_: MsgRegisterStratumResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterStratumResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterStratumResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgRegisterStratumResponse>): MsgRegisterStratumResponse {
    const message = createBaseMsgRegisterStratumResponse();
    return message;
  }
};
function createBaseMsgPostConjecture(): MsgPostConjecture {
  return {
    proposer: "",
    statement: "",
    falsificationPredicate: "",
    domain: "",
    category: "",
    stake: "",
    reasoningTrace: ""
  };
}
/**
 * MsgPostConjecture submits an unsettled proposition. It asserts nothing:
 * the resulting Fact carries confidence 0, cites nothing, cannot be cited,
 * is excluded from the training corpus, and pays its proposer no reward.
 * The review fee is the proposer's only cost and is non-refundable, exactly
 * as for an ordinary claim.
 *
 * The panel's question is "is this well-posed and falsifiable?". A conjecture
 * with no falsification predicate — or one that is not truth-apt — is
 * returned VERDICT_MALFORMED and no Fact is created.
 * @name MsgPostConjecture
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPostConjecture
 */
export const MsgPostConjecture = {
  typeUrl: "/zerone.knowledge.v1.MsgPostConjecture",
  encode(message: MsgPostConjecture, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.statement !== "") {
      writer.uint32(18).string(message.statement);
    }
    if (message.falsificationPredicate !== "") {
      writer.uint32(26).string(message.falsificationPredicate);
    }
    if (message.domain !== "") {
      writer.uint32(34).string(message.domain);
    }
    if (message.category !== "") {
      writer.uint32(42).string(message.category);
    }
    if (message.stake !== "") {
      writer.uint32(50).string(message.stake);
    }
    if (message.reasoningTrace !== "") {
      writer.uint32(58).string(message.reasoningTrace);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPostConjecture {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPostConjecture();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.statement = reader.string();
          break;
        case 3:
          message.falsificationPredicate = reader.string();
          break;
        case 4:
          message.domain = reader.string();
          break;
        case 5:
          message.category = reader.string();
          break;
        case 6:
          message.stake = reader.string();
          break;
        case 7:
          message.reasoningTrace = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgPostConjecture>): MsgPostConjecture {
    const message = createBaseMsgPostConjecture();
    message.proposer = object.proposer ?? "";
    message.statement = object.statement ?? "";
    message.falsificationPredicate = object.falsificationPredicate ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    message.stake = object.stake ?? "";
    message.reasoningTrace = object.reasoningTrace ?? "";
    return message;
  }
};
function createBaseMsgPostConjectureResponse(): MsgPostConjectureResponse {
  return {
    claimId: "",
    roundId: ""
  };
}
/**
 * @name MsgPostConjectureResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPostConjectureResponse
 */
export const MsgPostConjectureResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgPostConjectureResponse",
  encode(message: MsgPostConjectureResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.claimId !== "") {
      writer.uint32(10).string(message.claimId);
    }
    if (message.roundId !== "") {
      writer.uint32(18).string(message.roundId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPostConjectureResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPostConjectureResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimId = reader.string();
          break;
        case 2:
          message.roundId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgPostConjectureResponse>): MsgPostConjectureResponse {
    const message = createBaseMsgPostConjectureResponse();
    message.claimId = object.claimId ?? "";
    message.roundId = object.roundId ?? "";
    return message;
  }
};
function createBaseMsgChallengeProvisionalFact(): MsgChallengeProvisionalFact {
  return {
    challenger: "",
    claimId: "",
    factId: "",
    stake: "",
    reason: "",
    evidenceIds: [],
    counterClaim: ""
  };
}
/**
 * @name MsgChallengeProvisionalFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeProvisionalFact
 */
export const MsgChallengeProvisionalFact = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFact",
  encode(message: MsgChallengeProvisionalFact, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.claimId !== "") {
      writer.uint32(18).string(message.claimId);
    }
    if (message.factId !== "") {
      writer.uint32(26).string(message.factId);
    }
    if (message.stake !== "") {
      writer.uint32(34).string(message.stake);
    }
    if (message.reason !== "") {
      writer.uint32(42).string(message.reason);
    }
    for (const v of message.evidenceIds) {
      writer.uint32(50).string(v!);
    }
    if (message.counterClaim !== "") {
      writer.uint32(58).string(message.counterClaim);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeProvisionalFact {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeProvisionalFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.claimId = reader.string();
          break;
        case 3:
          message.factId = reader.string();
          break;
        case 4:
          message.stake = reader.string();
          break;
        case 5:
          message.reason = reader.string();
          break;
        case 6:
          message.evidenceIds.push(reader.string());
          break;
        case 7:
          message.counterClaim = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgChallengeProvisionalFact>): MsgChallengeProvisionalFact {
    const message = createBaseMsgChallengeProvisionalFact();
    message.challenger = object.challenger ?? "";
    message.claimId = object.claimId ?? "";
    message.factId = object.factId ?? "";
    message.stake = object.stake ?? "";
    message.reason = object.reason ?? "";
    message.evidenceIds = object.evidenceIds?.map(e => e) || [];
    message.counterClaim = object.counterClaim ?? "";
    return message;
  }
};
function createBaseMsgChallengeProvisionalFactResponse(): MsgChallengeProvisionalFactResponse {
  return {
    challengeId: ""
  };
}
/**
 * @name MsgChallengeProvisionalFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeProvisionalFactResponse
 */
export const MsgChallengeProvisionalFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFactResponse",
  encode(message: MsgChallengeProvisionalFactResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challengeId !== "") {
      writer.uint32(10).string(message.challengeId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeProvisionalFactResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeProvisionalFactResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challengeId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgChallengeProvisionalFactResponse>): MsgChallengeProvisionalFactResponse {
    const message = createBaseMsgChallengeProvisionalFactResponse();
    message.challengeId = object.challengeId ?? "";
    return message;
  }
};
function createBaseMsgUpdateParams(): MsgUpdateParams {
  return {
    authority: "",
    params: undefined
  };
}
/**
 * @name MsgUpdateParams
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateParams",
  encode(message: MsgUpdateParams, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams {
    const message = createBaseMsgUpdateParams();
    message.authority = object.authority ?? "";
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse(): MsgUpdateParamsResponse {
  return {};
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateParamsResponse",
  encode(_: MsgUpdateParamsResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse {
    const message = createBaseMsgUpdateParamsResponse();
    return message;
  }
};
function createBaseMsgUpdateExtendedParams(): MsgUpdateExtendedParams {
  return {
    authority: "",
    paramsJson: ""
  };
}
/**
 * @name MsgUpdateExtendedParams
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateExtendedParams
 */
export const MsgUpdateExtendedParams = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParams",
  encode(message: MsgUpdateExtendedParams, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.paramsJson !== "") {
      writer.uint32(18).string(message.paramsJson);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateExtendedParams {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateExtendedParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.paramsJson = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateExtendedParams>): MsgUpdateExtendedParams {
    const message = createBaseMsgUpdateExtendedParams();
    message.authority = object.authority ?? "";
    message.paramsJson = object.paramsJson ?? "";
    return message;
  }
};
function createBaseMsgUpdateExtendedParamsResponse(): MsgUpdateExtendedParamsResponse {
  return {};
}
/**
 * @name MsgUpdateExtendedParamsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateExtendedParamsResponse
 */
export const MsgUpdateExtendedParamsResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParamsResponse",
  encode(_: MsgUpdateExtendedParamsResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateExtendedParamsResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateExtendedParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgUpdateExtendedParamsResponse>): MsgUpdateExtendedParamsResponse {
    const message = createBaseMsgUpdateExtendedParamsResponse();
    return message;
  }
};
function createBaseMsgProposeResearchFund(): MsgProposeResearchFund {
  return {
    proposer: "",
    title: "",
    description: "",
    amount: "",
    recipient: "",
    votingPeriodBlocks: BigInt(0)
  };
}
/**
 * @name MsgProposeResearchFund
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeResearchFund
 */
export const MsgProposeResearchFund = {
  typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFund",
  encode(message: MsgProposeResearchFund, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    if (message.recipient !== "") {
      writer.uint32(42).string(message.recipient);
    }
    if (message.votingPeriodBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.votingPeriodBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeResearchFund {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeResearchFund();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        case 5:
          message.recipient = reader.string();
          break;
        case 6:
          message.votingPeriodBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeResearchFund>): MsgProposeResearchFund {
    const message = createBaseMsgProposeResearchFund();
    message.proposer = object.proposer ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.amount = object.amount ?? "";
    message.recipient = object.recipient ?? "";
    message.votingPeriodBlocks = object.votingPeriodBlocks !== undefined && object.votingPeriodBlocks !== null ? BigInt(object.votingPeriodBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgProposeResearchFundResponse(): MsgProposeResearchFundResponse {
  return {
    proposalId: ""
  };
}
/**
 * @name MsgProposeResearchFundResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeResearchFundResponse
 */
export const MsgProposeResearchFundResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFundResponse",
  encode(message: MsgProposeResearchFundResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeResearchFundResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeResearchFundResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeResearchFundResponse>): MsgProposeResearchFundResponse {
    const message = createBaseMsgProposeResearchFundResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteResearchProposal(): MsgVoteResearchProposal {
  return {
    voter: "",
    proposalId: "",
    vote: false
  };
}
/**
 * @name MsgVoteResearchProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteResearchProposal
 */
export const MsgVoteResearchProposal = {
  typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposal",
  encode(message: MsgVoteResearchProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.vote === true) {
      writer.uint32(24).bool(message.vote);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResearchProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResearchProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.vote = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteResearchProposal>): MsgVoteResearchProposal {
    const message = createBaseMsgVoteResearchProposal();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.vote = object.vote ?? false;
    return message;
  }
};
function createBaseMsgVoteResearchProposalResponse(): MsgVoteResearchProposalResponse {
  return {};
}
/**
 * @name MsgVoteResearchProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteResearchProposalResponse
 */
export const MsgVoteResearchProposalResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposalResponse",
  encode(_: MsgVoteResearchProposalResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResearchProposalResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResearchProposalResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgVoteResearchProposalResponse>): MsgVoteResearchProposalResponse {
    const message = createBaseMsgVoteResearchProposalResponse();
    return message;
  }
};
function createBaseMsgExecuteResearchProposal(): MsgExecuteResearchProposal {
  return {
    authority: "",
    proposalId: ""
  };
}
/**
 * @name MsgExecuteResearchProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgExecuteResearchProposal
 */
export const MsgExecuteResearchProposal = {
  typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposal",
  encode(message: MsgExecuteResearchProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgExecuteResearchProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgExecuteResearchProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgExecuteResearchProposal>): MsgExecuteResearchProposal {
    const message = createBaseMsgExecuteResearchProposal();
    message.authority = object.authority ?? "";
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgExecuteResearchProposalResponse(): MsgExecuteResearchProposalResponse {
  return {};
}
/**
 * @name MsgExecuteResearchProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgExecuteResearchProposalResponse
 */
export const MsgExecuteResearchProposalResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposalResponse",
  encode(_: MsgExecuteResearchProposalResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgExecuteResearchProposalResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgExecuteResearchProposalResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgExecuteResearchProposalResponse>): MsgExecuteResearchProposalResponse {
    const message = createBaseMsgExecuteResearchProposalResponse();
    return message;
  }
};
function createBaseMsgAddCommonKnowledge(): MsgAddCommonKnowledge {
  return {
    authority: "",
    domain: "",
    subject: "",
    description: "",
    penaltyBps: BigInt(0)
  };
}
/**
 * @name MsgAddCommonKnowledge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddCommonKnowledge
 */
export const MsgAddCommonKnowledge = {
  typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledge",
  encode(message: MsgAddCommonKnowledge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
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
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddCommonKnowledge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddCommonKnowledge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddCommonKnowledge>): MsgAddCommonKnowledge {
    const message = createBaseMsgAddCommonKnowledge();
    message.authority = object.authority ?? "";
    message.domain = object.domain ?? "";
    message.subject = object.subject ?? "";
    message.description = object.description ?? "";
    message.penaltyBps = object.penaltyBps !== undefined && object.penaltyBps !== null ? BigInt(object.penaltyBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAddCommonKnowledgeResponse(): MsgAddCommonKnowledgeResponse {
  return {
    id: ""
  };
}
/**
 * @name MsgAddCommonKnowledgeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddCommonKnowledgeResponse
 */
export const MsgAddCommonKnowledgeResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledgeResponse",
  encode(message: MsgAddCommonKnowledgeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddCommonKnowledgeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddCommonKnowledgeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddCommonKnowledgeResponse>): MsgAddCommonKnowledgeResponse {
    const message = createBaseMsgAddCommonKnowledgeResponse();
    message.id = object.id ?? "";
    return message;
  }
};
function createBaseMsgRemoveCommonKnowledge(): MsgRemoveCommonKnowledge {
  return {
    authority: "",
    id: ""
  };
}
/**
 * @name MsgRemoveCommonKnowledge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRemoveCommonKnowledge
 */
export const MsgRemoveCommonKnowledge = {
  typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledge",
  encode(message: MsgRemoveCommonKnowledge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveCommonKnowledge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveCommonKnowledge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRemoveCommonKnowledge>): MsgRemoveCommonKnowledge {
    const message = createBaseMsgRemoveCommonKnowledge();
    message.authority = object.authority ?? "";
    message.id = object.id ?? "";
    return message;
  }
};
function createBaseMsgRemoveCommonKnowledgeResponse(): MsgRemoveCommonKnowledgeResponse {
  return {};
}
/**
 * @name MsgRemoveCommonKnowledgeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRemoveCommonKnowledgeResponse
 */
export const MsgRemoveCommonKnowledgeResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledgeResponse",
  encode(_: MsgRemoveCommonKnowledgeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveCommonKnowledgeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveCommonKnowledgeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgRemoveCommonKnowledgeResponse>): MsgRemoveCommonKnowledgeResponse {
    const message = createBaseMsgRemoveCommonKnowledgeResponse();
    return message;
  }
};
function createBaseMsgReportDemand(): MsgReportDemand {
  return {
    reporter: "",
    reports: []
  };
}
/**
 * @name MsgReportDemand
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgReportDemand
 */
export const MsgReportDemand = {
  typeUrl: "/zerone.knowledge.v1.MsgReportDemand",
  encode(message: MsgReportDemand, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.reporter !== "") {
      writer.uint32(10).string(message.reporter);
    }
    for (const v of message.reports) {
      DemandReport.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgReportDemand {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgReportDemand();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reporter = reader.string();
          break;
        case 2:
          message.reports.push(DemandReport.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgReportDemand>): MsgReportDemand {
    const message = createBaseMsgReportDemand();
    message.reporter = object.reporter ?? "";
    message.reports = object.reports?.map(e => DemandReport.fromPartial(e)) || [];
    return message;
  }
};
function createBaseDemandReport(): DemandReport {
  return {
    domain: "",
    subject: "",
    queries: BigInt(0),
    fulfilled: BigInt(0),
    unfulfilled: BigInt(0)
  };
}
/**
 * @name DemandReport
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DemandReport
 */
export const DemandReport = {
  typeUrl: "/zerone.knowledge.v1.DemandReport",
  encode(message: DemandReport, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.domain !== "") {
      writer.uint32(10).string(message.domain);
    }
    if (message.subject !== "") {
      writer.uint32(18).string(message.subject);
    }
    if (message.queries !== BigInt(0)) {
      writer.uint32(24).uint64(message.queries);
    }
    if (message.fulfilled !== BigInt(0)) {
      writer.uint32(32).uint64(message.fulfilled);
    }
    if (message.unfulfilled !== BigInt(0)) {
      writer.uint32(40).uint64(message.unfulfilled);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DemandReport {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDemandReport();
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
          message.queries = reader.uint64();
          break;
        case 4:
          message.fulfilled = reader.uint64();
          break;
        case 5:
          message.unfulfilled = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DemandReport>): DemandReport {
    const message = createBaseDemandReport();
    message.domain = object.domain ?? "";
    message.subject = object.subject ?? "";
    message.queries = object.queries !== undefined && object.queries !== null ? BigInt(object.queries.toString()) : BigInt(0);
    message.fulfilled = object.fulfilled !== undefined && object.fulfilled !== null ? BigInt(object.fulfilled.toString()) : BigInt(0);
    message.unfulfilled = object.unfulfilled !== undefined && object.unfulfilled !== null ? BigInt(object.unfulfilled.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgReportDemandResponse(): MsgReportDemandResponse {
  return {};
}
/**
 * @name MsgReportDemandResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgReportDemandResponse
 */
export const MsgReportDemandResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgReportDemandResponse",
  encode(_: MsgReportDemandResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgReportDemandResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgReportDemandResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgReportDemandResponse>): MsgReportDemandResponse {
    const message = createBaseMsgReportDemandResponse();
    return message;
  }
};
function createBaseMsgRateFact(): MsgRateFact {
  return {
    rater: "",
    factId: "",
    useful: false,
    memo: ""
  };
}
/**
 * MsgRateFact allows a querier to provide relevance feedback on a fact.
 * The querier must have previously queried this fact (enforced by query receipt).
 * @name MsgRateFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRateFact
 */
export const MsgRateFact = {
  typeUrl: "/zerone.knowledge.v1.MsgRateFact",
  encode(message: MsgRateFact, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.rater !== "") {
      writer.uint32(10).string(message.rater);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.useful === true) {
      writer.uint32(24).bool(message.useful);
    }
    if (message.memo !== "") {
      writer.uint32(34).string(message.memo);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRateFact {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRateFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.rater = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.useful = reader.bool();
          break;
        case 4:
          message.memo = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRateFact>): MsgRateFact {
    const message = createBaseMsgRateFact();
    message.rater = object.rater ?? "";
    message.factId = object.factId ?? "";
    message.useful = object.useful ?? false;
    message.memo = object.memo ?? "";
    return message;
  }
};
function createBaseMsgRateFactResponse(): MsgRateFactResponse {
  return {};
}
/**
 * @name MsgRateFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRateFactResponse
 */
export const MsgRateFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRateFactResponse",
  encode(_: MsgRateFactResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRateFactResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRateFactResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgRateFactResponse>): MsgRateFactResponse {
    const message = createBaseMsgRateFactResponse();
    return message;
  }
};
function createBaseMsgRegisterTrainingPipeline(): MsgRegisterTrainingPipeline {
  return {
    operator: "",
    id: "",
    corpusSnapshotHeight: BigInt(0),
    tokenizerVersion: BigInt(0),
    methodologySetVersion: BigInt(0),
    recipeHash: "",
    description: "",
    corpusFilter: ""
  };
}
/**
 * @name MsgRegisterTrainingPipeline
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterTrainingPipeline
 */
export const MsgRegisterTrainingPipeline = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipeline",
  encode(message: MsgRegisterTrainingPipeline, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.operator !== "") {
      writer.uint32(10).string(message.operator);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
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
    if (message.corpusFilter !== "") {
      writer.uint32(66).string(message.corpusFilter);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterTrainingPipeline {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterTrainingPipeline();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operator = reader.string();
          break;
        case 2:
          message.id = reader.string();
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
          message.corpusFilter = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterTrainingPipeline>): MsgRegisterTrainingPipeline {
    const message = createBaseMsgRegisterTrainingPipeline();
    message.operator = object.operator ?? "";
    message.id = object.id ?? "";
    message.corpusSnapshotHeight = object.corpusSnapshotHeight !== undefined && object.corpusSnapshotHeight !== null ? BigInt(object.corpusSnapshotHeight.toString()) : BigInt(0);
    message.tokenizerVersion = object.tokenizerVersion !== undefined && object.tokenizerVersion !== null ? BigInt(object.tokenizerVersion.toString()) : BigInt(0);
    message.methodologySetVersion = object.methodologySetVersion !== undefined && object.methodologySetVersion !== null ? BigInt(object.methodologySetVersion.toString()) : BigInt(0);
    message.recipeHash = object.recipeHash ?? "";
    message.description = object.description ?? "";
    message.corpusFilter = object.corpusFilter ?? "";
    return message;
  }
};
function createBaseMsgRegisterTrainingPipelineResponse(): MsgRegisterTrainingPipelineResponse {
  return {};
}
/**
 * @name MsgRegisterTrainingPipelineResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterTrainingPipelineResponse
 */
export const MsgRegisterTrainingPipelineResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipelineResponse",
  encode(_: MsgRegisterTrainingPipelineResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterTrainingPipelineResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterTrainingPipelineResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgRegisterTrainingPipelineResponse>): MsgRegisterTrainingPipelineResponse {
    const message = createBaseMsgRegisterTrainingPipelineResponse();
    return message;
  }
};
function createBaseMsgUpdateTrainingPipeline(): MsgUpdateTrainingPipeline {
  return {
    operator: "",
    id: "",
    newStatus: "",
    completedAtBlock: BigInt(0)
  };
}
/**
 * @name MsgUpdateTrainingPipeline
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateTrainingPipeline
 */
export const MsgUpdateTrainingPipeline = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipeline",
  encode(message: MsgUpdateTrainingPipeline, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.operator !== "") {
      writer.uint32(10).string(message.operator);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.newStatus !== "") {
      writer.uint32(26).string(message.newStatus);
    }
    if (message.completedAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.completedAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateTrainingPipeline {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateTrainingPipeline();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operator = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.newStatus = reader.string();
          break;
        case 4:
          message.completedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateTrainingPipeline>): MsgUpdateTrainingPipeline {
    const message = createBaseMsgUpdateTrainingPipeline();
    message.operator = object.operator ?? "";
    message.id = object.id ?? "";
    message.newStatus = object.newStatus ?? "";
    message.completedAtBlock = object.completedAtBlock !== undefined && object.completedAtBlock !== null ? BigInt(object.completedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgUpdateTrainingPipelineResponse(): MsgUpdateTrainingPipelineResponse {
  return {};
}
/**
 * @name MsgUpdateTrainingPipelineResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateTrainingPipelineResponse
 */
export const MsgUpdateTrainingPipelineResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipelineResponse",
  encode(_: MsgUpdateTrainingPipelineResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateTrainingPipelineResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateTrainingPipelineResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgUpdateTrainingPipelineResponse>): MsgUpdateTrainingPipelineResponse {
    const message = createBaseMsgUpdateTrainingPipelineResponse();
    return message;
  }
};
function createBaseMsgRegisterModelCard(): MsgRegisterModelCard {
  return {
    owner: "",
    id: "",
    name: "",
    pipelineId: "",
    deploymentAddress: "",
    parameterCount: BigInt(0),
    route: "",
    baseModel: "",
    evalAcceptanceRateBps: BigInt(0),
    evalCorroborationRateBps: BigInt(0),
    evalSampleSize: BigInt(0),
    specialisedMethodId: ""
  };
}
/**
 * @name MsgRegisterModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterModelCard
 */
export const MsgRegisterModelCard = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCard",
  encode(message: MsgRegisterModelCard, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    if (message.pipelineId !== "") {
      writer.uint32(34).string(message.pipelineId);
    }
    if (message.deploymentAddress !== "") {
      writer.uint32(42).string(message.deploymentAddress);
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
    if (message.evalAcceptanceRateBps !== BigInt(0)) {
      writer.uint32(72).uint64(message.evalAcceptanceRateBps);
    }
    if (message.evalCorroborationRateBps !== BigInt(0)) {
      writer.uint32(80).uint64(message.evalCorroborationRateBps);
    }
    if (message.evalSampleSize !== BigInt(0)) {
      writer.uint32(88).uint64(message.evalSampleSize);
    }
    if (message.specialisedMethodId !== "") {
      writer.uint32(98).string(message.specialisedMethodId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterModelCard {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterModelCard();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.name = reader.string();
          break;
        case 4:
          message.pipelineId = reader.string();
          break;
        case 5:
          message.deploymentAddress = reader.string();
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
          message.evalAcceptanceRateBps = reader.uint64();
          break;
        case 10:
          message.evalCorroborationRateBps = reader.uint64();
          break;
        case 11:
          message.evalSampleSize = reader.uint64();
          break;
        case 12:
          message.specialisedMethodId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterModelCard>): MsgRegisterModelCard {
    const message = createBaseMsgRegisterModelCard();
    message.owner = object.owner ?? "";
    message.id = object.id ?? "";
    message.name = object.name ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.deploymentAddress = object.deploymentAddress ?? "";
    message.parameterCount = object.parameterCount !== undefined && object.parameterCount !== null ? BigInt(object.parameterCount.toString()) : BigInt(0);
    message.route = object.route ?? "";
    message.baseModel = object.baseModel ?? "";
    message.evalAcceptanceRateBps = object.evalAcceptanceRateBps !== undefined && object.evalAcceptanceRateBps !== null ? BigInt(object.evalAcceptanceRateBps.toString()) : BigInt(0);
    message.evalCorroborationRateBps = object.evalCorroborationRateBps !== undefined && object.evalCorroborationRateBps !== null ? BigInt(object.evalCorroborationRateBps.toString()) : BigInt(0);
    message.evalSampleSize = object.evalSampleSize !== undefined && object.evalSampleSize !== null ? BigInt(object.evalSampleSize.toString()) : BigInt(0);
    message.specialisedMethodId = object.specialisedMethodId ?? "";
    return message;
  }
};
function createBaseMsgRegisterModelCardResponse(): MsgRegisterModelCardResponse {
  return {};
}
/**
 * @name MsgRegisterModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterModelCardResponse
 */
export const MsgRegisterModelCardResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCardResponse",
  encode(_: MsgRegisterModelCardResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterModelCardResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterModelCardResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgRegisterModelCardResponse>): MsgRegisterModelCardResponse {
    const message = createBaseMsgRegisterModelCardResponse();
    return message;
  }
};
function createBaseMsgUpdateModelCard(): MsgUpdateModelCard {
  return {
    owner: "",
    id: "",
    evalAcceptanceRateBps: BigInt(0),
    evalCorroborationRateBps: BigInt(0),
    evalSampleSize: BigInt(0),
    name: ""
  };
}
/**
 * @name MsgUpdateModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateModelCard
 */
export const MsgUpdateModelCard = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCard",
  encode(message: MsgUpdateModelCard, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.evalAcceptanceRateBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.evalAcceptanceRateBps);
    }
    if (message.evalCorroborationRateBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.evalCorroborationRateBps);
    }
    if (message.evalSampleSize !== BigInt(0)) {
      writer.uint32(40).uint64(message.evalSampleSize);
    }
    if (message.name !== "") {
      writer.uint32(50).string(message.name);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateModelCard {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateModelCard();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.evalAcceptanceRateBps = reader.uint64();
          break;
        case 4:
          message.evalCorroborationRateBps = reader.uint64();
          break;
        case 5:
          message.evalSampleSize = reader.uint64();
          break;
        case 6:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateModelCard>): MsgUpdateModelCard {
    const message = createBaseMsgUpdateModelCard();
    message.owner = object.owner ?? "";
    message.id = object.id ?? "";
    message.evalAcceptanceRateBps = object.evalAcceptanceRateBps !== undefined && object.evalAcceptanceRateBps !== null ? BigInt(object.evalAcceptanceRateBps.toString()) : BigInt(0);
    message.evalCorroborationRateBps = object.evalCorroborationRateBps !== undefined && object.evalCorroborationRateBps !== null ? BigInt(object.evalCorroborationRateBps.toString()) : BigInt(0);
    message.evalSampleSize = object.evalSampleSize !== undefined && object.evalSampleSize !== null ? BigInt(object.evalSampleSize.toString()) : BigInt(0);
    message.name = object.name ?? "";
    return message;
  }
};
function createBaseMsgUpdateModelCardResponse(): MsgUpdateModelCardResponse {
  return {};
}
/**
 * @name MsgUpdateModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateModelCardResponse
 */
export const MsgUpdateModelCardResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCardResponse",
  encode(_: MsgUpdateModelCardResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateModelCardResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateModelCardResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgUpdateModelCardResponse>): MsgUpdateModelCardResponse {
    const message = createBaseMsgUpdateModelCardResponse();
    return message;
  }
};
function createBaseMsgRetireModelCard(): MsgRetireModelCard {
  return {
    owner: "",
    id: "",
    reason: ""
  };
}
/**
 * @name MsgRetireModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRetireModelCard
 */
export const MsgRetireModelCard = {
  typeUrl: "/zerone.knowledge.v1.MsgRetireModelCard",
  encode(message: MsgRetireModelCard, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRetireModelCard {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRetireModelCard();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRetireModelCard>): MsgRetireModelCard {
    const message = createBaseMsgRetireModelCard();
    message.owner = object.owner ?? "";
    message.id = object.id ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgRetireModelCardResponse(): MsgRetireModelCardResponse {
  return {};
}
/**
 * @name MsgRetireModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRetireModelCardResponse
 */
export const MsgRetireModelCardResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRetireModelCardResponse",
  encode(_: MsgRetireModelCardResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRetireModelCardResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRetireModelCardResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgRetireModelCardResponse>): MsgRetireModelCardResponse {
    const message = createBaseMsgRetireModelCardResponse();
    return message;
  }
};
function createBaseMsgAmendTokenizerSpec(): MsgAmendTokenizerSpec {
  return {
    authority: "",
    spec: undefined
  };
}
/**
 * @name MsgAmendTokenizerSpec
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTokenizerSpec
 */
export const MsgAmendTokenizerSpec = {
  typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpec",
  encode(message: MsgAmendTokenizerSpec, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.spec !== undefined) {
      TokenizerSpec.encode(message.spec, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAmendTokenizerSpec {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAmendTokenizerSpec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.spec = TokenizerSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAmendTokenizerSpec>): MsgAmendTokenizerSpec {
    const message = createBaseMsgAmendTokenizerSpec();
    message.authority = object.authority ?? "";
    message.spec = object.spec !== undefined && object.spec !== null ? TokenizerSpec.fromPartial(object.spec) : undefined;
    return message;
  }
};
function createBaseMsgAmendTokenizerSpecResponse(): MsgAmendTokenizerSpecResponse {
  return {
    newVersion: BigInt(0)
  };
}
/**
 * @name MsgAmendTokenizerSpecResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTokenizerSpecResponse
 */
export const MsgAmendTokenizerSpecResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpecResponse",
  encode(message: MsgAmendTokenizerSpecResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.newVersion !== BigInt(0)) {
      writer.uint32(8).uint64(message.newVersion);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAmendTokenizerSpecResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAmendTokenizerSpecResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newVersion = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAmendTokenizerSpecResponse>): MsgAmendTokenizerSpecResponse {
    const message = createBaseMsgAmendTokenizerSpecResponse();
    message.newVersion = object.newVersion !== undefined && object.newVersion !== null ? BigInt(object.newVersion.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAttributeContributions(): MsgAttributeContributions {
  return {
    owner: "",
    modelId: "",
    factIds: [],
    totalWeight: BigInt(0)
  };
}
/**
 * @name MsgAttributeContributions
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttributeContributions
 */
export const MsgAttributeContributions = {
  typeUrl: "/zerone.knowledge.v1.MsgAttributeContributions",
  encode(message: MsgAttributeContributions, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.modelId !== "") {
      writer.uint32(18).string(message.modelId);
    }
    for (const v of message.factIds) {
      writer.uint32(26).string(v!);
    }
    if (message.totalWeight !== BigInt(0)) {
      writer.uint32(32).uint64(message.totalWeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAttributeContributions {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAttributeContributions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.modelId = reader.string();
          break;
        case 3:
          message.factIds.push(reader.string());
          break;
        case 4:
          message.totalWeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAttributeContributions>): MsgAttributeContributions {
    const message = createBaseMsgAttributeContributions();
    message.owner = object.owner ?? "";
    message.modelId = object.modelId ?? "";
    message.factIds = object.factIds?.map(e => e) || [];
    message.totalWeight = object.totalWeight !== undefined && object.totalWeight !== null ? BigInt(object.totalWeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAttributeContributionsResponse(): MsgAttributeContributionsResponse {
  return {
    recorded: 0
  };
}
/**
 * @name MsgAttributeContributionsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttributeContributionsResponse
 */
export const MsgAttributeContributionsResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAttributeContributionsResponse",
  encode(message: MsgAttributeContributionsResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.recorded !== 0) {
      writer.uint32(8).uint32(message.recorded);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAttributeContributionsResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAttributeContributionsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.recorded = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAttributeContributionsResponse>): MsgAttributeContributionsResponse {
    const message = createBaseMsgAttributeContributionsResponse();
    message.recorded = object.recorded ?? 0;
    return message;
  }
};
function createBaseMsgAttestTraining(): MsgAttestTraining {
  return {
    attester: "",
    pipelineId: "",
    flopsEstimate: BigInt(0),
    wallclockSeconds: BigInt(0),
    evalHash: "",
    signature: "",
    notes: ""
  };
}
/**
 * @name MsgAttestTraining
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttestTraining
 */
export const MsgAttestTraining = {
  typeUrl: "/zerone.knowledge.v1.MsgAttestTraining",
  encode(message: MsgAttestTraining, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.attester !== "") {
      writer.uint32(10).string(message.attester);
    }
    if (message.pipelineId !== "") {
      writer.uint32(18).string(message.pipelineId);
    }
    if (message.flopsEstimate !== BigInt(0)) {
      writer.uint32(24).uint64(message.flopsEstimate);
    }
    if (message.wallclockSeconds !== BigInt(0)) {
      writer.uint32(32).uint64(message.wallclockSeconds);
    }
    if (message.evalHash !== "") {
      writer.uint32(42).string(message.evalHash);
    }
    if (message.signature !== "") {
      writer.uint32(50).string(message.signature);
    }
    if (message.notes !== "") {
      writer.uint32(58).string(message.notes);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAttestTraining {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAttestTraining();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.attester = reader.string();
          break;
        case 2:
          message.pipelineId = reader.string();
          break;
        case 3:
          message.flopsEstimate = reader.uint64();
          break;
        case 4:
          message.wallclockSeconds = reader.uint64();
          break;
        case 5:
          message.evalHash = reader.string();
          break;
        case 6:
          message.signature = reader.string();
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
  fromPartial(object: DeepPartial<MsgAttestTraining>): MsgAttestTraining {
    const message = createBaseMsgAttestTraining();
    message.attester = object.attester ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.flopsEstimate = object.flopsEstimate !== undefined && object.flopsEstimate !== null ? BigInt(object.flopsEstimate.toString()) : BigInt(0);
    message.wallclockSeconds = object.wallclockSeconds !== undefined && object.wallclockSeconds !== null ? BigInt(object.wallclockSeconds.toString()) : BigInt(0);
    message.evalHash = object.evalHash ?? "";
    message.signature = object.signature ?? "";
    message.notes = object.notes ?? "";
    return message;
  }
};
function createBaseMsgAttestTrainingResponse(): MsgAttestTrainingResponse {
  return {};
}
/**
 * @name MsgAttestTrainingResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttestTrainingResponse
 */
export const MsgAttestTrainingResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAttestTrainingResponse",
  encode(_: MsgAttestTrainingResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAttestTrainingResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAttestTrainingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgAttestTrainingResponse>): MsgAttestTrainingResponse {
    const message = createBaseMsgAttestTrainingResponse();
    return message;
  }
};
function createBaseMsgCreateAugmentationBounty(): MsgCreateAugmentationBounty {
  return {
    sponsor: "",
    id: "",
    targetFactId: "",
    rewardPerVariant: BigInt(0),
    maxVariants: 0,
    expiresAtBlock: BigInt(0),
    description: ""
  };
}
/**
 * @name MsgCreateAugmentationBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateAugmentationBounty
 */
export const MsgCreateAugmentationBounty = {
  typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBounty",
  encode(message: MsgCreateAugmentationBounty, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sponsor !== "") {
      writer.uint32(10).string(message.sponsor);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
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
    if (message.expiresAtBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.expiresAtBlock);
    }
    if (message.description !== "") {
      writer.uint32(58).string(message.description);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateAugmentationBounty {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateAugmentationBounty();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sponsor = reader.string();
          break;
        case 2:
          message.id = reader.string();
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
          message.expiresAtBlock = reader.uint64();
          break;
        case 7:
          message.description = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateAugmentationBounty>): MsgCreateAugmentationBounty {
    const message = createBaseMsgCreateAugmentationBounty();
    message.sponsor = object.sponsor ?? "";
    message.id = object.id ?? "";
    message.targetFactId = object.targetFactId ?? "";
    message.rewardPerVariant = object.rewardPerVariant !== undefined && object.rewardPerVariant !== null ? BigInt(object.rewardPerVariant.toString()) : BigInt(0);
    message.maxVariants = object.maxVariants ?? 0;
    message.expiresAtBlock = object.expiresAtBlock !== undefined && object.expiresAtBlock !== null ? BigInt(object.expiresAtBlock.toString()) : BigInt(0);
    message.description = object.description ?? "";
    return message;
  }
};
function createBaseMsgCreateAugmentationBountyResponse(): MsgCreateAugmentationBountyResponse {
  return {};
}
/**
 * @name MsgCreateAugmentationBountyResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateAugmentationBountyResponse
 */
export const MsgCreateAugmentationBountyResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBountyResponse",
  encode(_: MsgCreateAugmentationBountyResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateAugmentationBountyResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateAugmentationBountyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgCreateAugmentationBountyResponse>): MsgCreateAugmentationBountyResponse {
    const message = createBaseMsgCreateAugmentationBountyResponse();
    return message;
  }
};
function createBaseMsgSubmitAugmentation(): MsgSubmitAugmentation {
  return {
    submitter: "",
    id: "",
    bountyId: "",
    originalFactId: "",
    variantContent: "",
    variantReasoningTrace: ""
  };
}
/**
 * @name MsgSubmitAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitAugmentation
 */
export const MsgSubmitAugmentation = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentation",
  encode(message: MsgSubmitAugmentation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.bountyId !== "") {
      writer.uint32(26).string(message.bountyId);
    }
    if (message.originalFactId !== "") {
      writer.uint32(34).string(message.originalFactId);
    }
    if (message.variantContent !== "") {
      writer.uint32(42).string(message.variantContent);
    }
    if (message.variantReasoningTrace !== "") {
      writer.uint32(50).string(message.variantReasoningTrace);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitAugmentation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.bountyId = reader.string();
          break;
        case 4:
          message.originalFactId = reader.string();
          break;
        case 5:
          message.variantContent = reader.string();
          break;
        case 6:
          message.variantReasoningTrace = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitAugmentation>): MsgSubmitAugmentation {
    const message = createBaseMsgSubmitAugmentation();
    message.submitter = object.submitter ?? "";
    message.id = object.id ?? "";
    message.bountyId = object.bountyId ?? "";
    message.originalFactId = object.originalFactId ?? "";
    message.variantContent = object.variantContent ?? "";
    message.variantReasoningTrace = object.variantReasoningTrace ?? "";
    return message;
  }
};
function createBaseMsgSubmitAugmentationResponse(): MsgSubmitAugmentationResponse {
  return {};
}
/**
 * @name MsgSubmitAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitAugmentationResponse
 */
export const MsgSubmitAugmentationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentationResponse",
  encode(_: MsgSubmitAugmentationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitAugmentationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitAugmentationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgSubmitAugmentationResponse>): MsgSubmitAugmentationResponse {
    const message = createBaseMsgSubmitAugmentationResponse();
    return message;
  }
};
function createBaseMsgAcceptAugmentation(): MsgAcceptAugmentation {
  return {
    acceptor: "",
    augmentationId: "",
    note: ""
  };
}
/**
 * @name MsgAcceptAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAcceptAugmentation
 */
export const MsgAcceptAugmentation = {
  typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentation",
  encode(message: MsgAcceptAugmentation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.acceptor !== "") {
      writer.uint32(10).string(message.acceptor);
    }
    if (message.augmentationId !== "") {
      writer.uint32(18).string(message.augmentationId);
    }
    if (message.note !== "") {
      writer.uint32(26).string(message.note);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAcceptAugmentation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAcceptAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.acceptor = reader.string();
          break;
        case 2:
          message.augmentationId = reader.string();
          break;
        case 3:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAcceptAugmentation>): MsgAcceptAugmentation {
    const message = createBaseMsgAcceptAugmentation();
    message.acceptor = object.acceptor ?? "";
    message.augmentationId = object.augmentationId ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgAcceptAugmentationResponse(): MsgAcceptAugmentationResponse {
  return {};
}
/**
 * @name MsgAcceptAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAcceptAugmentationResponse
 */
export const MsgAcceptAugmentationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentationResponse",
  encode(_: MsgAcceptAugmentationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAcceptAugmentationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAcceptAugmentationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgAcceptAugmentationResponse>): MsgAcceptAugmentationResponse {
    const message = createBaseMsgAcceptAugmentationResponse();
    return message;
  }
};
function createBaseMsgVoteOnAugmentation(): MsgVoteOnAugmentation {
  return {
    verifier: "",
    augmentationId: "",
    vote: 0,
    rationale: ""
  };
}
/**
 * ─── Wave 4: reformulation verdicts ───────────────────────────────────────
 * @name MsgVoteOnAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteOnAugmentation
 */
export const MsgVoteOnAugmentation = {
  typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentation",
  encode(message: MsgVoteOnAugmentation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.augmentationId !== "") {
      writer.uint32(18).string(message.augmentationId);
    }
    if (message.vote !== 0) {
      writer.uint32(24).int32(message.vote);
    }
    if (message.rationale !== "") {
      writer.uint32(34).string(message.rationale);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteOnAugmentation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteOnAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.augmentationId = reader.string();
          break;
        case 3:
          message.vote = reader.int32() as any;
          break;
        case 4:
          message.rationale = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteOnAugmentation>): MsgVoteOnAugmentation {
    const message = createBaseMsgVoteOnAugmentation();
    message.verifier = object.verifier ?? "";
    message.augmentationId = object.augmentationId ?? "";
    message.vote = object.vote ?? 0;
    message.rationale = object.rationale ?? "";
    return message;
  }
};
function createBaseMsgVoteOnAugmentationResponse(): MsgVoteOnAugmentationResponse {
  return {
    verdictFinalized: false,
    finalizedVerdict: 0
  };
}
/**
 * @name MsgVoteOnAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteOnAugmentationResponse
 */
export const MsgVoteOnAugmentationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentationResponse",
  encode(message: MsgVoteOnAugmentationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.verdictFinalized === true) {
      writer.uint32(8).bool(message.verdictFinalized);
    }
    if (message.finalizedVerdict !== 0) {
      writer.uint32(16).int32(message.finalizedVerdict);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteOnAugmentationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteOnAugmentationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verdictFinalized = reader.bool();
          break;
        case 2:
          message.finalizedVerdict = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteOnAugmentationResponse>): MsgVoteOnAugmentationResponse {
    const message = createBaseMsgVoteOnAugmentationResponse();
    message.verdictFinalized = object.verdictFinalized ?? false;
    message.finalizedVerdict = object.finalizedVerdict ?? 0;
    return message;
  }
};
function createBaseMsgSponsorVetoAugmentation(): MsgSponsorVetoAugmentation {
  return {
    sponsor: "",
    augmentationId: "",
    reason: ""
  };
}
/**
 * @name MsgSponsorVetoAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSponsorVetoAugmentation
 */
export const MsgSponsorVetoAugmentation = {
  typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentation",
  encode(message: MsgSponsorVetoAugmentation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sponsor !== "") {
      writer.uint32(10).string(message.sponsor);
    }
    if (message.augmentationId !== "") {
      writer.uint32(18).string(message.augmentationId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSponsorVetoAugmentation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSponsorVetoAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sponsor = reader.string();
          break;
        case 2:
          message.augmentationId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSponsorVetoAugmentation>): MsgSponsorVetoAugmentation {
    const message = createBaseMsgSponsorVetoAugmentation();
    message.sponsor = object.sponsor ?? "";
    message.augmentationId = object.augmentationId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgSponsorVetoAugmentationResponse(): MsgSponsorVetoAugmentationResponse {
  return {};
}
/**
 * @name MsgSponsorVetoAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSponsorVetoAugmentationResponse
 */
export const MsgSponsorVetoAugmentationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentationResponse",
  encode(_: MsgSponsorVetoAugmentationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSponsorVetoAugmentationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSponsorVetoAugmentationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgSponsorVetoAugmentationResponse>): MsgSponsorVetoAugmentationResponse {
    const message = createBaseMsgSponsorVetoAugmentationResponse();
    return message;
  }
};
function createBaseMsgChallengeContribution(): MsgChallengeContribution {
  return {
    challenger: "",
    modelId: "",
    disputedFactId: "",
    disputeType: "",
    evidence: "",
    id: ""
  };
}
/**
 * ─── Wave 4: attribution challenges ───────────────────────────────────────
 * @name MsgChallengeContribution
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeContribution
 */
export const MsgChallengeContribution = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeContribution",
  encode(message: MsgChallengeContribution, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.modelId !== "") {
      writer.uint32(18).string(message.modelId);
    }
    if (message.disputedFactId !== "") {
      writer.uint32(26).string(message.disputedFactId);
    }
    if (message.disputeType !== "") {
      writer.uint32(34).string(message.disputeType);
    }
    if (message.evidence !== "") {
      writer.uint32(42).string(message.evidence);
    }
    if (message.id !== "") {
      writer.uint32(50).string(message.id);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeContribution {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeContribution();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.modelId = reader.string();
          break;
        case 3:
          message.disputedFactId = reader.string();
          break;
        case 4:
          message.disputeType = reader.string();
          break;
        case 5:
          message.evidence = reader.string();
          break;
        case 6:
          message.id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgChallengeContribution>): MsgChallengeContribution {
    const message = createBaseMsgChallengeContribution();
    message.challenger = object.challenger ?? "";
    message.modelId = object.modelId ?? "";
    message.disputedFactId = object.disputedFactId ?? "";
    message.disputeType = object.disputeType ?? "";
    message.evidence = object.evidence ?? "";
    message.id = object.id ?? "";
    return message;
  }
};
function createBaseMsgChallengeContributionResponse(): MsgChallengeContributionResponse {
  return {
    bondEscrowed: ""
  };
}
/**
 * @name MsgChallengeContributionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeContributionResponse
 */
export const MsgChallengeContributionResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeContributionResponse",
  encode(message: MsgChallengeContributionResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.bondEscrowed !== "") {
      writer.uint32(10).string(message.bondEscrowed);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeContributionResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeContributionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.bondEscrowed = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgChallengeContributionResponse>): MsgChallengeContributionResponse {
    const message = createBaseMsgChallengeContributionResponse();
    message.bondEscrowed = object.bondEscrowed ?? "";
    return message;
  }
};
function createBaseMsgResolveContributionChallenge(): MsgResolveContributionChallenge {
  return {
    resolver: "",
    challengeId: "",
    uphold: false,
    note: ""
  };
}
/**
 * @name MsgResolveContributionChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveContributionChallenge
 */
export const MsgResolveContributionChallenge = {
  typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallenge",
  encode(message: MsgResolveContributionChallenge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.resolver !== "") {
      writer.uint32(10).string(message.resolver);
    }
    if (message.challengeId !== "") {
      writer.uint32(18).string(message.challengeId);
    }
    if (message.uphold === true) {
      writer.uint32(24).bool(message.uphold);
    }
    if (message.note !== "") {
      writer.uint32(34).string(message.note);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveContributionChallenge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveContributionChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.resolver = reader.string();
          break;
        case 2:
          message.challengeId = reader.string();
          break;
        case 3:
          message.uphold = reader.bool();
          break;
        case 4:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgResolveContributionChallenge>): MsgResolveContributionChallenge {
    const message = createBaseMsgResolveContributionChallenge();
    message.resolver = object.resolver ?? "";
    message.challengeId = object.challengeId ?? "";
    message.uphold = object.uphold ?? false;
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgResolveContributionChallengeResponse(): MsgResolveContributionChallengeResponse {
  return {
    payoutToWinner: ""
  };
}
/**
 * @name MsgResolveContributionChallengeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveContributionChallengeResponse
 */
export const MsgResolveContributionChallengeResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallengeResponse",
  encode(message: MsgResolveContributionChallengeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.payoutToWinner !== "") {
      writer.uint32(10).string(message.payoutToWinner);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveContributionChallengeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveContributionChallengeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.payoutToWinner = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgResolveContributionChallengeResponse>): MsgResolveContributionChallengeResponse {
    const message = createBaseMsgResolveContributionChallengeResponse();
    message.payoutToWinner = object.payoutToWinner ?? "";
    return message;
  }
};
function createBaseMsgClaimTrainingFundDisbursement(): MsgClaimTrainingFundDisbursement {
  return {
    claimant: "",
    modelId: "",
    id: ""
  };
}
/**
 * ─── Wave 4: training fund post-hoc disbursement ──────────────────────────
 * @name MsgClaimTrainingFundDisbursement
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgClaimTrainingFundDisbursement
 */
export const MsgClaimTrainingFundDisbursement = {
  typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement",
  encode(message: MsgClaimTrainingFundDisbursement, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.claimant !== "") {
      writer.uint32(10).string(message.claimant);
    }
    if (message.modelId !== "") {
      writer.uint32(18).string(message.modelId);
    }
    if (message.id !== "") {
      writer.uint32(26).string(message.id);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimTrainingFundDisbursement {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimTrainingFundDisbursement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimant = reader.string();
          break;
        case 2:
          message.modelId = reader.string();
          break;
        case 3:
          message.id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgClaimTrainingFundDisbursement>): MsgClaimTrainingFundDisbursement {
    const message = createBaseMsgClaimTrainingFundDisbursement();
    message.claimant = object.claimant ?? "";
    message.modelId = object.modelId ?? "";
    message.id = object.id ?? "";
    return message;
  }
};
function createBaseMsgClaimTrainingFundDisbursementResponse(): MsgClaimTrainingFundDisbursementResponse {
  return {
    totalAmount: "",
    releasedAmount: "",
    vestingAmount: "",
    vestingEndBlock: BigInt(0)
  };
}
/**
 * @name MsgClaimTrainingFundDisbursementResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgClaimTrainingFundDisbursementResponse
 */
export const MsgClaimTrainingFundDisbursementResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursementResponse",
  encode(message: MsgClaimTrainingFundDisbursementResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.totalAmount !== "") {
      writer.uint32(10).string(message.totalAmount);
    }
    if (message.releasedAmount !== "") {
      writer.uint32(18).string(message.releasedAmount);
    }
    if (message.vestingAmount !== "") {
      writer.uint32(26).string(message.vestingAmount);
    }
    if (message.vestingEndBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.vestingEndBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimTrainingFundDisbursementResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimTrainingFundDisbursementResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalAmount = reader.string();
          break;
        case 2:
          message.releasedAmount = reader.string();
          break;
        case 3:
          message.vestingAmount = reader.string();
          break;
        case 4:
          message.vestingEndBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgClaimTrainingFundDisbursementResponse>): MsgClaimTrainingFundDisbursementResponse {
    const message = createBaseMsgClaimTrainingFundDisbursementResponse();
    message.totalAmount = object.totalAmount ?? "";
    message.releasedAmount = object.releasedAmount ?? "";
    message.vestingAmount = object.vestingAmount ?? "";
    message.vestingEndBlock = object.vestingEndBlock !== undefined && object.vestingEndBlock !== null ? BigInt(object.vestingEndBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAmendTraceSchema(): MsgAmendTraceSchema {
  return {
    authority: "",
    schema: undefined
  };
}
/**
 * ─── Route B Wave 5: trace schema amendment ───────────────────────────────
 * @name MsgAmendTraceSchema
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTraceSchema
 */
export const MsgAmendTraceSchema = {
  typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchema",
  encode(message: MsgAmendTraceSchema, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.schema !== undefined) {
      TraceSchema.encode(message.schema, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAmendTraceSchema {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAmendTraceSchema();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.schema = TraceSchema.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAmendTraceSchema>): MsgAmendTraceSchema {
    const message = createBaseMsgAmendTraceSchema();
    message.authority = object.authority ?? "";
    message.schema = object.schema !== undefined && object.schema !== null ? TraceSchema.fromPartial(object.schema) : undefined;
    return message;
  }
};
function createBaseMsgAmendTraceSchemaResponse(): MsgAmendTraceSchemaResponse {
  return {
    newVersion: BigInt(0)
  };
}
/**
 * @name MsgAmendTraceSchemaResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTraceSchemaResponse
 */
export const MsgAmendTraceSchemaResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchemaResponse",
  encode(message: MsgAmendTraceSchemaResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.newVersion !== BigInt(0)) {
      writer.uint32(8).uint64(message.newVersion);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAmendTraceSchemaResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAmendTraceSchemaResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newVersion = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAmendTraceSchemaResponse>): MsgAmendTraceSchemaResponse {
    const message = createBaseMsgAmendTraceSchemaResponse();
    message.newVersion = object.newVersion !== undefined && object.newVersion !== null ? BigInt(object.newVersion.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgCreateTrainingManifest(): MsgCreateTrainingManifest {
  return {
    creator: "",
    id: "",
    pipelineId: "",
    corpusSelector: undefined,
    description: "",
    parentManifestId: ""
  };
}
/**
 * ─── Route B Wave 7: training manifests ──────────────────────────────────
 * @name MsgCreateTrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateTrainingManifest
 */
export const MsgCreateTrainingManifest = {
  typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifest",
  encode(message: MsgCreateTrainingManifest, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.pipelineId !== "") {
      writer.uint32(26).string(message.pipelineId);
    }
    if (message.corpusSelector !== undefined) {
      CorpusSelector.encode(message.corpusSelector, writer.uint32(34).fork()).ldelim();
    }
    if (message.description !== "") {
      writer.uint32(42).string(message.description);
    }
    if (message.parentManifestId !== "") {
      writer.uint32(50).string(message.parentManifestId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateTrainingManifest {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateTrainingManifest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.pipelineId = reader.string();
          break;
        case 4:
          message.corpusSelector = CorpusSelector.decode(reader, reader.uint32());
          break;
        case 5:
          message.description = reader.string();
          break;
        case 6:
          message.parentManifestId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateTrainingManifest>): MsgCreateTrainingManifest {
    const message = createBaseMsgCreateTrainingManifest();
    message.creator = object.creator ?? "";
    message.id = object.id ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.corpusSelector = object.corpusSelector !== undefined && object.corpusSelector !== null ? CorpusSelector.fromPartial(object.corpusSelector) : undefined;
    message.description = object.description ?? "";
    message.parentManifestId = object.parentManifestId ?? "";
    return message;
  }
};
function createBaseMsgCreateTrainingManifestResponse(): MsgCreateTrainingManifestResponse {
  return {
    totalIncluded: 0,
    factCount: 0,
    traceCount: 0,
    pairCount: 0,
    driftCount: 0,
    normativeCount: 0
  };
}
/**
 * @name MsgCreateTrainingManifestResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateTrainingManifestResponse
 */
export const MsgCreateTrainingManifestResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifestResponse",
  encode(message: MsgCreateTrainingManifestResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.totalIncluded !== 0) {
      writer.uint32(8).uint32(message.totalIncluded);
    }
    if (message.factCount !== 0) {
      writer.uint32(16).uint32(message.factCount);
    }
    if (message.traceCount !== 0) {
      writer.uint32(24).uint32(message.traceCount);
    }
    if (message.pairCount !== 0) {
      writer.uint32(32).uint32(message.pairCount);
    }
    if (message.driftCount !== 0) {
      writer.uint32(40).uint32(message.driftCount);
    }
    if (message.normativeCount !== 0) {
      writer.uint32(48).uint32(message.normativeCount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateTrainingManifestResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateTrainingManifestResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalIncluded = reader.uint32();
          break;
        case 2:
          message.factCount = reader.uint32();
          break;
        case 3:
          message.traceCount = reader.uint32();
          break;
        case 4:
          message.pairCount = reader.uint32();
          break;
        case 5:
          message.driftCount = reader.uint32();
          break;
        case 6:
          message.normativeCount = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateTrainingManifestResponse>): MsgCreateTrainingManifestResponse {
    const message = createBaseMsgCreateTrainingManifestResponse();
    message.totalIncluded = object.totalIncluded ?? 0;
    message.factCount = object.factCount ?? 0;
    message.traceCount = object.traceCount ?? 0;
    message.pairCount = object.pairCount ?? 0;
    message.driftCount = object.driftCount ?? 0;
    message.normativeCount = object.normativeCount ?? 0;
    return message;
  }
};
function createBaseMsgFinalizeTrainingManifest(): MsgFinalizeTrainingManifest {
  return {
    creator: "",
    manifestId: ""
  };
}
/**
 * @name MsgFinalizeTrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgFinalizeTrainingManifest
 */
export const MsgFinalizeTrainingManifest = {
  typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifest",
  encode(message: MsgFinalizeTrainingManifest, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.manifestId !== "") {
      writer.uint32(18).string(message.manifestId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFinalizeTrainingManifest {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFinalizeTrainingManifest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.manifestId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgFinalizeTrainingManifest>): MsgFinalizeTrainingManifest {
    const message = createBaseMsgFinalizeTrainingManifest();
    message.creator = object.creator ?? "";
    message.manifestId = object.manifestId ?? "";
    return message;
  }
};
function createBaseMsgFinalizeTrainingManifestResponse(): MsgFinalizeTrainingManifestResponse {
  return {
    merkleRoot: ""
  };
}
/**
 * @name MsgFinalizeTrainingManifestResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgFinalizeTrainingManifestResponse
 */
export const MsgFinalizeTrainingManifestResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifestResponse",
  encode(message: MsgFinalizeTrainingManifestResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.merkleRoot !== "") {
      writer.uint32(10).string(message.merkleRoot);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFinalizeTrainingManifestResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFinalizeTrainingManifestResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.merkleRoot = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgFinalizeTrainingManifestResponse>): MsgFinalizeTrainingManifestResponse {
    const message = createBaseMsgFinalizeTrainingManifestResponse();
    message.merkleRoot = object.merkleRoot ?? "";
    return message;
  }
};
function createBaseMsgBindManifestToAttestation(): MsgBindManifestToAttestation {
  return {
    creator: "",
    manifestId: "",
    attestationId: ""
  };
}
/**
 * @name MsgBindManifestToAttestation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgBindManifestToAttestation
 */
export const MsgBindManifestToAttestation = {
  typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestation",
  encode(message: MsgBindManifestToAttestation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.manifestId !== "") {
      writer.uint32(18).string(message.manifestId);
    }
    if (message.attestationId !== "") {
      writer.uint32(26).string(message.attestationId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgBindManifestToAttestation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgBindManifestToAttestation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.manifestId = reader.string();
          break;
        case 3:
          message.attestationId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgBindManifestToAttestation>): MsgBindManifestToAttestation {
    const message = createBaseMsgBindManifestToAttestation();
    message.creator = object.creator ?? "";
    message.manifestId = object.manifestId ?? "";
    message.attestationId = object.attestationId ?? "";
    return message;
  }
};
function createBaseMsgBindManifestToAttestationResponse(): MsgBindManifestToAttestationResponse {
  return {};
}
/**
 * @name MsgBindManifestToAttestationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgBindManifestToAttestationResponse
 */
export const MsgBindManifestToAttestationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestationResponse",
  encode(_: MsgBindManifestToAttestationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgBindManifestToAttestationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgBindManifestToAttestationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgBindManifestToAttestationResponse>): MsgBindManifestToAttestationResponse {
    const message = createBaseMsgBindManifestToAttestationResponse();
    return message;
  }
};
function createBaseMsgOpenIncident(): MsgOpenIncident {
  return {
    authority: "",
    id: "",
    severity: 0,
    title: "",
    description: "",
    reporter: "",
    affectedModules: [],
    slaWindowBlocks: BigInt(0)
  };
}
/**
 * ─── Route B Wave 11: incident response ──────────────────────────────────
 * @name MsgOpenIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgOpenIncident
 */
export const MsgOpenIncident = {
  typeUrl: "/zerone.knowledge.v1.MsgOpenIncident",
  encode(message: MsgOpenIncident, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.severity !== 0) {
      writer.uint32(24).int32(message.severity);
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
    for (const v of message.affectedModules) {
      writer.uint32(58).string(v!);
    }
    if (message.slaWindowBlocks !== BigInt(0)) {
      writer.uint32(64).uint64(message.slaWindowBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgOpenIncident {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgOpenIncident();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.severity = reader.int32() as any;
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
          message.affectedModules.push(reader.string());
          break;
        case 8:
          message.slaWindowBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgOpenIncident>): MsgOpenIncident {
    const message = createBaseMsgOpenIncident();
    message.authority = object.authority ?? "";
    message.id = object.id ?? "";
    message.severity = object.severity ?? 0;
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.reporter = object.reporter ?? "";
    message.affectedModules = object.affectedModules?.map(e => e) || [];
    message.slaWindowBlocks = object.slaWindowBlocks !== undefined && object.slaWindowBlocks !== null ? BigInt(object.slaWindowBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgOpenIncidentResponse(): MsgOpenIncidentResponse {
  return {
    slaTargetBlock: BigInt(0)
  };
}
/**
 * @name MsgOpenIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgOpenIncidentResponse
 */
export const MsgOpenIncidentResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgOpenIncidentResponse",
  encode(message: MsgOpenIncidentResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.slaTargetBlock !== BigInt(0)) {
      writer.uint32(8).uint64(message.slaTargetBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgOpenIncidentResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgOpenIncidentResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.slaTargetBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgOpenIncidentResponse>): MsgOpenIncidentResponse {
    const message = createBaseMsgOpenIncidentResponse();
    message.slaTargetBlock = object.slaTargetBlock !== undefined && object.slaTargetBlock !== null ? BigInt(object.slaTargetBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgRecordRemediation(): MsgRecordRemediation {
  return {
    authority: "",
    incidentId: "",
    type: 0,
    reference: "",
    note: ""
  };
}
/**
 * @name MsgRecordRemediation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRecordRemediation
 */
export const MsgRecordRemediation = {
  typeUrl: "/zerone.knowledge.v1.MsgRecordRemediation",
  encode(message: MsgRecordRemediation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.incidentId !== "") {
      writer.uint32(18).string(message.incidentId);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.reference !== "") {
      writer.uint32(34).string(message.reference);
    }
    if (message.note !== "") {
      writer.uint32(42).string(message.note);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRecordRemediation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRecordRemediation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.incidentId = reader.string();
          break;
        case 3:
          message.type = reader.int32() as any;
          break;
        case 4:
          message.reference = reader.string();
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
  fromPartial(object: DeepPartial<MsgRecordRemediation>): MsgRecordRemediation {
    const message = createBaseMsgRecordRemediation();
    message.authority = object.authority ?? "";
    message.incidentId = object.incidentId ?? "";
    message.type = object.type ?? 0;
    message.reference = object.reference ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgRecordRemediationResponse(): MsgRecordRemediationResponse {
  return {
    totalRemediations: 0
  };
}
/**
 * @name MsgRecordRemediationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRecordRemediationResponse
 */
export const MsgRecordRemediationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRecordRemediationResponse",
  encode(message: MsgRecordRemediationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.totalRemediations !== 0) {
      writer.uint32(8).uint32(message.totalRemediations);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRecordRemediationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRecordRemediationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalRemediations = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRecordRemediationResponse>): MsgRecordRemediationResponse {
    const message = createBaseMsgRecordRemediationResponse();
    message.totalRemediations = object.totalRemediations ?? 0;
    return message;
  }
};
function createBaseMsgResolveIncident(): MsgResolveIncident {
  return {
    authority: "",
    incidentId: "",
    postMortemUri: ""
  };
}
/**
 * @name MsgResolveIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveIncident
 */
export const MsgResolveIncident = {
  typeUrl: "/zerone.knowledge.v1.MsgResolveIncident",
  encode(message: MsgResolveIncident, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.incidentId !== "") {
      writer.uint32(18).string(message.incidentId);
    }
    if (message.postMortemUri !== "") {
      writer.uint32(26).string(message.postMortemUri);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveIncident {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveIncident();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.incidentId = reader.string();
          break;
        case 3:
          message.postMortemUri = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgResolveIncident>): MsgResolveIncident {
    const message = createBaseMsgResolveIncident();
    message.authority = object.authority ?? "";
    message.incidentId = object.incidentId ?? "";
    message.postMortemUri = object.postMortemUri ?? "";
    return message;
  }
};
function createBaseMsgResolveIncidentResponse(): MsgResolveIncidentResponse {
  return {};
}
/**
 * @name MsgResolveIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveIncidentResponse
 */
export const MsgResolveIncidentResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgResolveIncidentResponse",
  encode(_: MsgResolveIncidentResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveIncidentResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveIncidentResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgResolveIncidentResponse>): MsgResolveIncidentResponse {
    const message = createBaseMsgResolveIncidentResponse();
    return message;
  }
};
function createBaseMsgCloseIncident(): MsgCloseIncident {
  return {
    authority: "",
    incidentId: ""
  };
}
/**
 * @name MsgCloseIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCloseIncident
 */
export const MsgCloseIncident = {
  typeUrl: "/zerone.knowledge.v1.MsgCloseIncident",
  encode(message: MsgCloseIncident, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.incidentId !== "") {
      writer.uint32(18).string(message.incidentId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCloseIncident {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCloseIncident();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.incidentId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCloseIncident>): MsgCloseIncident {
    const message = createBaseMsgCloseIncident();
    message.authority = object.authority ?? "";
    message.incidentId = object.incidentId ?? "";
    return message;
  }
};
function createBaseMsgCloseIncidentResponse(): MsgCloseIncidentResponse {
  return {};
}
/**
 * @name MsgCloseIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCloseIncidentResponse
 */
export const MsgCloseIncidentResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgCloseIncidentResponse",
  encode(_: MsgCloseIncidentResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCloseIncidentResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCloseIncidentResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgCloseIncidentResponse>): MsgCloseIncidentResponse {
    const message = createBaseMsgCloseIncidentResponse();
    return message;
  }
};
function createBaseMsgPauseModule(): MsgPauseModule {
  return {
    authority: "",
    moduleName: "",
    reason: "",
    autoUnpauseAtBlock: BigInt(0),
    incidentId: ""
  };
}
/**
 * ─── Route B Wave 12: module circuit breakers ────────────────────────────
 * @name MsgPauseModule
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPauseModule
 */
export const MsgPauseModule = {
  typeUrl: "/zerone.knowledge.v1.MsgPauseModule",
  encode(message: MsgPauseModule, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.moduleName !== "") {
      writer.uint32(18).string(message.moduleName);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    if (message.autoUnpauseAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.autoUnpauseAtBlock);
    }
    if (message.incidentId !== "") {
      writer.uint32(42).string(message.incidentId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseModule {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseModule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.moduleName = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        case 4:
          message.autoUnpauseAtBlock = reader.uint64();
          break;
        case 5:
          message.incidentId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgPauseModule>): MsgPauseModule {
    const message = createBaseMsgPauseModule();
    message.authority = object.authority ?? "";
    message.moduleName = object.moduleName ?? "";
    message.reason = object.reason ?? "";
    message.autoUnpauseAtBlock = object.autoUnpauseAtBlock !== undefined && object.autoUnpauseAtBlock !== null ? BigInt(object.autoUnpauseAtBlock.toString()) : BigInt(0);
    message.incidentId = object.incidentId ?? "";
    return message;
  }
};
function createBaseMsgPauseModuleResponse(): MsgPauseModuleResponse {
  return {
    pausedAtBlock: BigInt(0)
  };
}
/**
 * @name MsgPauseModuleResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPauseModuleResponse
 */
export const MsgPauseModuleResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgPauseModuleResponse",
  encode(message: MsgPauseModuleResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.pausedAtBlock !== BigInt(0)) {
      writer.uint32(8).uint64(message.pausedAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseModuleResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseModuleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pausedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgPauseModuleResponse>): MsgPauseModuleResponse {
    const message = createBaseMsgPauseModuleResponse();
    message.pausedAtBlock = object.pausedAtBlock !== undefined && object.pausedAtBlock !== null ? BigInt(object.pausedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgUnpauseModule(): MsgUnpauseModule {
  return {
    authority: "",
    moduleName: "",
    note: ""
  };
}
/**
 * @name MsgUnpauseModule
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUnpauseModule
 */
export const MsgUnpauseModule = {
  typeUrl: "/zerone.knowledge.v1.MsgUnpauseModule",
  encode(message: MsgUnpauseModule, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.moduleName !== "") {
      writer.uint32(18).string(message.moduleName);
    }
    if (message.note !== "") {
      writer.uint32(26).string(message.note);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUnpauseModule {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnpauseModule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.moduleName = reader.string();
          break;
        case 3:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUnpauseModule>): MsgUnpauseModule {
    const message = createBaseMsgUnpauseModule();
    message.authority = object.authority ?? "";
    message.moduleName = object.moduleName ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgUnpauseModuleResponse(): MsgUnpauseModuleResponse {
  return {};
}
/**
 * @name MsgUnpauseModuleResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUnpauseModuleResponse
 */
export const MsgUnpauseModuleResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUnpauseModuleResponse",
  encode(_: MsgUnpauseModuleResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUnpauseModuleResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnpauseModuleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgUnpauseModuleResponse>): MsgUnpauseModuleResponse {
    const message = createBaseMsgUnpauseModuleResponse();
    return message;
  }
};
function createBaseMsgCorrectManifestMerkleRoot(): MsgCorrectManifestMerkleRoot {
  return {
    authority: "",
    manifestId: "",
    incidentId: "",
    expectedRecomputedRoot: "",
    note: ""
  };
}
/**
 * @name MsgCorrectManifestMerkleRoot
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCorrectManifestMerkleRoot
 */
export const MsgCorrectManifestMerkleRoot = {
  typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot",
  encode(message: MsgCorrectManifestMerkleRoot, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.manifestId !== "") {
      writer.uint32(18).string(message.manifestId);
    }
    if (message.incidentId !== "") {
      writer.uint32(26).string(message.incidentId);
    }
    if (message.expectedRecomputedRoot !== "") {
      writer.uint32(34).string(message.expectedRecomputedRoot);
    }
    if (message.note !== "") {
      writer.uint32(42).string(message.note);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCorrectManifestMerkleRoot {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCorrectManifestMerkleRoot();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.manifestId = reader.string();
          break;
        case 3:
          message.incidentId = reader.string();
          break;
        case 4:
          message.expectedRecomputedRoot = reader.string();
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
  fromPartial(object: DeepPartial<MsgCorrectManifestMerkleRoot>): MsgCorrectManifestMerkleRoot {
    const message = createBaseMsgCorrectManifestMerkleRoot();
    message.authority = object.authority ?? "";
    message.manifestId = object.manifestId ?? "";
    message.incidentId = object.incidentId ?? "";
    message.expectedRecomputedRoot = object.expectedRecomputedRoot ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgCorrectManifestMerkleRootResponse(): MsgCorrectManifestMerkleRootResponse {
  return {
    priorRoot: "",
    recomputedRoot: "",
    wasCorrupted: false
  };
}
/**
 * @name MsgCorrectManifestMerkleRootResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCorrectManifestMerkleRootResponse
 */
export const MsgCorrectManifestMerkleRootResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRootResponse",
  encode(message: MsgCorrectManifestMerkleRootResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.priorRoot !== "") {
      writer.uint32(10).string(message.priorRoot);
    }
    if (message.recomputedRoot !== "") {
      writer.uint32(18).string(message.recomputedRoot);
    }
    if (message.wasCorrupted === true) {
      writer.uint32(24).bool(message.wasCorrupted);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCorrectManifestMerkleRootResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCorrectManifestMerkleRootResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.priorRoot = reader.string();
          break;
        case 2:
          message.recomputedRoot = reader.string();
          break;
        case 3:
          message.wasCorrupted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCorrectManifestMerkleRootResponse>): MsgCorrectManifestMerkleRootResponse {
    const message = createBaseMsgCorrectManifestMerkleRootResponse();
    message.priorRoot = object.priorRoot ?? "";
    message.recomputedRoot = object.recomputedRoot ?? "";
    message.wasCorrupted = object.wasCorrupted ?? false;
    return message;
  }
};
function createBaseMsgVetoFactInjection(): MsgVetoFactInjection {
  return {
    guardian: "",
    pendingId: "",
    reason: ""
  };
}
/**
 * MsgVetoFactInjection — a registered guardian cancels a pending
 * authority-injected fact during the veto window. The fact never
 * materializes; the privileged-action log records the veto for audit.
 * @name MsgVetoFactInjection
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVetoFactInjection
 */
export const MsgVetoFactInjection = {
  typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjection",
  encode(message: MsgVetoFactInjection, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.guardian !== "") {
      writer.uint32(10).string(message.guardian);
    }
    if (message.pendingId !== "") {
      writer.uint32(18).string(message.pendingId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVetoFactInjection {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVetoFactInjection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardian = reader.string();
          break;
        case 2:
          message.pendingId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVetoFactInjection>): MsgVetoFactInjection {
    const message = createBaseMsgVetoFactInjection();
    message.guardian = object.guardian ?? "";
    message.pendingId = object.pendingId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgVetoFactInjectionResponse(): MsgVetoFactInjectionResponse {
  return {};
}
/**
 * @name MsgVetoFactInjectionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVetoFactInjectionResponse
 */
export const MsgVetoFactInjectionResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjectionResponse",
  encode(_: MsgVetoFactInjectionResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVetoFactInjectionResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVetoFactInjectionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgVetoFactInjectionResponse>): MsgVetoFactInjectionResponse {
    const message = createBaseMsgVetoFactInjectionResponse();
    return message;
  }
};