import { ClaimType, ClaimRelation, ClaimStructure, TokenizerSpec, AugmentationVerdict, TraceSchema, CorpusSelector, IncidentSeverity, RemediationType } from "./types.js";
import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
export interface MsgSubmitCommitmentResponse {
}
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
export interface MsgSubmitRevealResponse {
}
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
export interface MsgPatronizeFactResponse {
}
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
export interface MsgEndorseDomainProposalResponse {
}
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
export interface MsgChallengeDomainProposalResponse {
}
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
export interface MsgRegisterStratumResponse {
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
export interface MsgUpdateParamsResponse {
}
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
export interface MsgUpdateExtendedParamsResponse {
}
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
export interface MsgVoteResearchProposalResponse {
}
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
export interface MsgExecuteResearchProposalResponse {
}
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
export interface MsgRemoveCommonKnowledgeResponse {
}
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
export interface MsgReportDemandResponse {
}
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
export interface MsgRateFactResponse {
}
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
export interface MsgRegisterTrainingPipelineResponse {
}
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
export interface MsgUpdateTrainingPipelineResponse {
}
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
export interface MsgRegisterModelCardResponse {
}
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
export interface MsgUpdateModelCardResponse {
}
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
export interface MsgRetireModelCardResponse {
}
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
export interface MsgAttestTrainingResponse {
}
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
export interface MsgCreateAugmentationBountyResponse {
}
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
export interface MsgSubmitAugmentationResponse {
}
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
export interface MsgAcceptAugmentationResponse {
}
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
export interface MsgSponsorVetoAugmentationResponse {
}
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
     * upheld: escrowed bond refund; rejected: "0"
     */
    payoutToWinner: string;
}
/**
 * ─── Wave 4: disabled training-fund disbursement placeholder ───────────────
 * @name MsgClaimTrainingFundDisbursement
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgClaimTrainingFundDisbursement
 */
export interface MsgClaimTrainingFundDisbursement {
    claimant: string;
    modelId: string;
    /**
     * Legacy client-chosen identifier. It does not bind a unique reward
     * entitlement; this is why the current handler is fail-closed.
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
export interface MsgBindManifestToAttestationResponse {
}
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
export interface MsgResolveIncidentResponse {
}
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
export interface MsgCloseIncidentResponse {
}
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
export interface MsgUnpauseModuleResponse {
}
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
export interface MsgVetoFactInjectionResponse {
}
/**
 * @name MsgSubmitClaim
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitClaim
 */
export declare const MsgSubmitClaim: {
    typeUrl: string;
    encode(message: MsgSubmitClaim, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitClaim;
    fromPartial(object: DeepPartial<MsgSubmitClaim>): MsgSubmitClaim;
};
/**
 * @name MsgSubmitClaimResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitClaimResponse
 */
export declare const MsgSubmitClaimResponse: {
    typeUrl: string;
    encode(message: MsgSubmitClaimResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitClaimResponse;
    fromPartial(object: DeepPartial<MsgSubmitClaimResponse>): MsgSubmitClaimResponse;
};
/**
 * @name MsgSubmitCommitment
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitCommitment
 */
export declare const MsgSubmitCommitment: {
    typeUrl: string;
    encode(message: MsgSubmitCommitment, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitCommitment;
    fromPartial(object: DeepPartial<MsgSubmitCommitment>): MsgSubmitCommitment;
};
/**
 * @name MsgSubmitCommitmentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitCommitmentResponse
 */
export declare const MsgSubmitCommitmentResponse: {
    typeUrl: string;
    encode(_: MsgSubmitCommitmentResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitCommitmentResponse;
    fromPartial(_: DeepPartial<MsgSubmitCommitmentResponse>): MsgSubmitCommitmentResponse;
};
/**
 * @name MsgSubmitReveal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitReveal
 */
export declare const MsgSubmitReveal: {
    typeUrl: string;
    encode(message: MsgSubmitReveal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitReveal;
    fromPartial(object: DeepPartial<MsgSubmitReveal>): MsgSubmitReveal;
};
/**
 * @name MsgSubmitRevealResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitRevealResponse
 */
export declare const MsgSubmitRevealResponse: {
    typeUrl: string;
    encode(_: MsgSubmitRevealResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitRevealResponse;
    fromPartial(_: DeepPartial<MsgSubmitRevealResponse>): MsgSubmitRevealResponse;
};
/**
 * @name MsgChallengeFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeFact
 */
export declare const MsgChallengeFact: {
    typeUrl: string;
    encode(message: MsgChallengeFact, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeFact;
    fromPartial(object: DeepPartial<MsgChallengeFact>): MsgChallengeFact;
};
/**
 * @name MsgChallengeFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeFactResponse
 */
export declare const MsgChallengeFactResponse: {
    typeUrl: string;
    encode(message: MsgChallengeFactResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeFactResponse;
    fromPartial(object: DeepPartial<MsgChallengeFactResponse>): MsgChallengeFactResponse;
};
/**
 * @name MsgAddFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddFact
 */
export declare const MsgAddFact: {
    typeUrl: string;
    encode(message: MsgAddFact, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddFact;
    fromPartial(object: DeepPartial<MsgAddFact>): MsgAddFact;
};
/**
 * @name MsgAddFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddFactResponse
 */
export declare const MsgAddFactResponse: {
    typeUrl: string;
    encode(message: MsgAddFactResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddFactResponse;
    fromPartial(object: DeepPartial<MsgAddFactResponse>): MsgAddFactResponse;
};
/**
 * @name MsgSubmitContradiction
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitContradiction
 */
export declare const MsgSubmitContradiction: {
    typeUrl: string;
    encode(message: MsgSubmitContradiction, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitContradiction;
    fromPartial(object: DeepPartial<MsgSubmitContradiction>): MsgSubmitContradiction;
};
/**
 * @name MsgSubmitContradictionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitContradictionResponse
 */
export declare const MsgSubmitContradictionResponse: {
    typeUrl: string;
    encode(message: MsgSubmitContradictionResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitContradictionResponse;
    fromPartial(object: DeepPartial<MsgSubmitContradictionResponse>): MsgSubmitContradictionResponse;
};
/**
 * @name MsgPatronizeFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPatronizeFact
 */
export declare const MsgPatronizeFact: {
    typeUrl: string;
    encode(message: MsgPatronizeFact, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPatronizeFact;
    fromPartial(object: DeepPartial<MsgPatronizeFact>): MsgPatronizeFact;
};
/**
 * @name MsgPatronizeFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPatronizeFactResponse
 */
export declare const MsgPatronizeFactResponse: {
    typeUrl: string;
    encode(_: MsgPatronizeFactResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPatronizeFactResponse;
    fromPartial(_: DeepPartial<MsgPatronizeFactResponse>): MsgPatronizeFactResponse;
};
/**
 * @name MsgProposeDomain
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeDomain
 */
export declare const MsgProposeDomain: {
    typeUrl: string;
    encode(message: MsgProposeDomain, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeDomain;
    fromPartial(object: DeepPartial<MsgProposeDomain>): MsgProposeDomain;
};
/**
 * @name MsgProposeDomainResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeDomainResponse
 */
export declare const MsgProposeDomainResponse: {
    typeUrl: string;
    encode(message: MsgProposeDomainResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeDomainResponse;
    fromPartial(object: DeepPartial<MsgProposeDomainResponse>): MsgProposeDomainResponse;
};
/**
 * @name MsgEndorseDomainProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgEndorseDomainProposal
 */
export declare const MsgEndorseDomainProposal: {
    typeUrl: string;
    encode(message: MsgEndorseDomainProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgEndorseDomainProposal;
    fromPartial(object: DeepPartial<MsgEndorseDomainProposal>): MsgEndorseDomainProposal;
};
/**
 * @name MsgEndorseDomainProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgEndorseDomainProposalResponse
 */
export declare const MsgEndorseDomainProposalResponse: {
    typeUrl: string;
    encode(_: MsgEndorseDomainProposalResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgEndorseDomainProposalResponse;
    fromPartial(_: DeepPartial<MsgEndorseDomainProposalResponse>): MsgEndorseDomainProposalResponse;
};
/**
 * @name MsgChallengeDomainProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeDomainProposal
 */
export declare const MsgChallengeDomainProposal: {
    typeUrl: string;
    encode(message: MsgChallengeDomainProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeDomainProposal;
    fromPartial(object: DeepPartial<MsgChallengeDomainProposal>): MsgChallengeDomainProposal;
};
/**
 * @name MsgChallengeDomainProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeDomainProposalResponse
 */
export declare const MsgChallengeDomainProposalResponse: {
    typeUrl: string;
    encode(_: MsgChallengeDomainProposalResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeDomainProposalResponse;
    fromPartial(_: DeepPartial<MsgChallengeDomainProposalResponse>): MsgChallengeDomainProposalResponse;
};
/**
 * @name MsgRegisterStratum
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterStratum
 */
export declare const MsgRegisterStratum: {
    typeUrl: string;
    encode(message: MsgRegisterStratum, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterStratum;
    fromPartial(object: DeepPartial<MsgRegisterStratum>): MsgRegisterStratum;
};
/**
 * @name MsgRegisterStratumResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterStratumResponse
 */
export declare const MsgRegisterStratumResponse: {
    typeUrl: string;
    encode(_: MsgRegisterStratumResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterStratumResponse;
    fromPartial(_: DeepPartial<MsgRegisterStratumResponse>): MsgRegisterStratumResponse;
};
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
export declare const MsgPostConjecture: {
    typeUrl: string;
    encode(message: MsgPostConjecture, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPostConjecture;
    fromPartial(object: DeepPartial<MsgPostConjecture>): MsgPostConjecture;
};
/**
 * @name MsgPostConjectureResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPostConjectureResponse
 */
export declare const MsgPostConjectureResponse: {
    typeUrl: string;
    encode(message: MsgPostConjectureResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPostConjectureResponse;
    fromPartial(object: DeepPartial<MsgPostConjectureResponse>): MsgPostConjectureResponse;
};
/**
 * @name MsgChallengeProvisionalFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeProvisionalFact
 */
export declare const MsgChallengeProvisionalFact: {
    typeUrl: string;
    encode(message: MsgChallengeProvisionalFact, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeProvisionalFact;
    fromPartial(object: DeepPartial<MsgChallengeProvisionalFact>): MsgChallengeProvisionalFact;
};
/**
 * @name MsgChallengeProvisionalFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeProvisionalFactResponse
 */
export declare const MsgChallengeProvisionalFactResponse: {
    typeUrl: string;
    encode(message: MsgChallengeProvisionalFactResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeProvisionalFactResponse;
    fromPartial(object: DeepPartial<MsgChallengeProvisionalFactResponse>): MsgChallengeProvisionalFactResponse;
};
/**
 * @name MsgUpdateParams
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
/**
 * @name MsgUpdateExtendedParams
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateExtendedParams
 */
export declare const MsgUpdateExtendedParams: {
    typeUrl: string;
    encode(message: MsgUpdateExtendedParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateExtendedParams;
    fromPartial(object: DeepPartial<MsgUpdateExtendedParams>): MsgUpdateExtendedParams;
};
/**
 * @name MsgUpdateExtendedParamsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateExtendedParamsResponse
 */
export declare const MsgUpdateExtendedParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateExtendedParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateExtendedParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateExtendedParamsResponse>): MsgUpdateExtendedParamsResponse;
};
/**
 * @name MsgProposeResearchFund
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeResearchFund
 */
export declare const MsgProposeResearchFund: {
    typeUrl: string;
    encode(message: MsgProposeResearchFund, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeResearchFund;
    fromPartial(object: DeepPartial<MsgProposeResearchFund>): MsgProposeResearchFund;
};
/**
 * @name MsgProposeResearchFundResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgProposeResearchFundResponse
 */
export declare const MsgProposeResearchFundResponse: {
    typeUrl: string;
    encode(message: MsgProposeResearchFundResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeResearchFundResponse;
    fromPartial(object: DeepPartial<MsgProposeResearchFundResponse>): MsgProposeResearchFundResponse;
};
/**
 * @name MsgVoteResearchProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteResearchProposal
 */
export declare const MsgVoteResearchProposal: {
    typeUrl: string;
    encode(message: MsgVoteResearchProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResearchProposal;
    fromPartial(object: DeepPartial<MsgVoteResearchProposal>): MsgVoteResearchProposal;
};
/**
 * @name MsgVoteResearchProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteResearchProposalResponse
 */
export declare const MsgVoteResearchProposalResponse: {
    typeUrl: string;
    encode(_: MsgVoteResearchProposalResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResearchProposalResponse;
    fromPartial(_: DeepPartial<MsgVoteResearchProposalResponse>): MsgVoteResearchProposalResponse;
};
/**
 * @name MsgExecuteResearchProposal
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgExecuteResearchProposal
 */
export declare const MsgExecuteResearchProposal: {
    typeUrl: string;
    encode(message: MsgExecuteResearchProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgExecuteResearchProposal;
    fromPartial(object: DeepPartial<MsgExecuteResearchProposal>): MsgExecuteResearchProposal;
};
/**
 * @name MsgExecuteResearchProposalResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgExecuteResearchProposalResponse
 */
export declare const MsgExecuteResearchProposalResponse: {
    typeUrl: string;
    encode(_: MsgExecuteResearchProposalResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgExecuteResearchProposalResponse;
    fromPartial(_: DeepPartial<MsgExecuteResearchProposalResponse>): MsgExecuteResearchProposalResponse;
};
/**
 * @name MsgAddCommonKnowledge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddCommonKnowledge
 */
export declare const MsgAddCommonKnowledge: {
    typeUrl: string;
    encode(message: MsgAddCommonKnowledge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddCommonKnowledge;
    fromPartial(object: DeepPartial<MsgAddCommonKnowledge>): MsgAddCommonKnowledge;
};
/**
 * @name MsgAddCommonKnowledgeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAddCommonKnowledgeResponse
 */
export declare const MsgAddCommonKnowledgeResponse: {
    typeUrl: string;
    encode(message: MsgAddCommonKnowledgeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddCommonKnowledgeResponse;
    fromPartial(object: DeepPartial<MsgAddCommonKnowledgeResponse>): MsgAddCommonKnowledgeResponse;
};
/**
 * @name MsgRemoveCommonKnowledge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRemoveCommonKnowledge
 */
export declare const MsgRemoveCommonKnowledge: {
    typeUrl: string;
    encode(message: MsgRemoveCommonKnowledge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveCommonKnowledge;
    fromPartial(object: DeepPartial<MsgRemoveCommonKnowledge>): MsgRemoveCommonKnowledge;
};
/**
 * @name MsgRemoveCommonKnowledgeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRemoveCommonKnowledgeResponse
 */
export declare const MsgRemoveCommonKnowledgeResponse: {
    typeUrl: string;
    encode(_: MsgRemoveCommonKnowledgeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRemoveCommonKnowledgeResponse;
    fromPartial(_: DeepPartial<MsgRemoveCommonKnowledgeResponse>): MsgRemoveCommonKnowledgeResponse;
};
/**
 * @name MsgReportDemand
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgReportDemand
 */
export declare const MsgReportDemand: {
    typeUrl: string;
    encode(message: MsgReportDemand, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgReportDemand;
    fromPartial(object: DeepPartial<MsgReportDemand>): MsgReportDemand;
};
/**
 * @name DemandReport
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.DemandReport
 */
export declare const DemandReport: {
    typeUrl: string;
    encode(message: DemandReport, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DemandReport;
    fromPartial(object: DeepPartial<DemandReport>): DemandReport;
};
/**
 * @name MsgReportDemandResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgReportDemandResponse
 */
export declare const MsgReportDemandResponse: {
    typeUrl: string;
    encode(_: MsgReportDemandResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgReportDemandResponse;
    fromPartial(_: DeepPartial<MsgReportDemandResponse>): MsgReportDemandResponse;
};
/**
 * MsgRateFact allows a querier to provide relevance feedback on a fact.
 * The querier must have previously queried this fact (enforced by query receipt).
 * @name MsgRateFact
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRateFact
 */
export declare const MsgRateFact: {
    typeUrl: string;
    encode(message: MsgRateFact, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRateFact;
    fromPartial(object: DeepPartial<MsgRateFact>): MsgRateFact;
};
/**
 * @name MsgRateFactResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRateFactResponse
 */
export declare const MsgRateFactResponse: {
    typeUrl: string;
    encode(_: MsgRateFactResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRateFactResponse;
    fromPartial(_: DeepPartial<MsgRateFactResponse>): MsgRateFactResponse;
};
/**
 * @name MsgRegisterTrainingPipeline
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterTrainingPipeline
 */
export declare const MsgRegisterTrainingPipeline: {
    typeUrl: string;
    encode(message: MsgRegisterTrainingPipeline, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterTrainingPipeline;
    fromPartial(object: DeepPartial<MsgRegisterTrainingPipeline>): MsgRegisterTrainingPipeline;
};
/**
 * @name MsgRegisterTrainingPipelineResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterTrainingPipelineResponse
 */
export declare const MsgRegisterTrainingPipelineResponse: {
    typeUrl: string;
    encode(_: MsgRegisterTrainingPipelineResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterTrainingPipelineResponse;
    fromPartial(_: DeepPartial<MsgRegisterTrainingPipelineResponse>): MsgRegisterTrainingPipelineResponse;
};
/**
 * @name MsgUpdateTrainingPipeline
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateTrainingPipeline
 */
export declare const MsgUpdateTrainingPipeline: {
    typeUrl: string;
    encode(message: MsgUpdateTrainingPipeline, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateTrainingPipeline;
    fromPartial(object: DeepPartial<MsgUpdateTrainingPipeline>): MsgUpdateTrainingPipeline;
};
/**
 * @name MsgUpdateTrainingPipelineResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateTrainingPipelineResponse
 */
export declare const MsgUpdateTrainingPipelineResponse: {
    typeUrl: string;
    encode(_: MsgUpdateTrainingPipelineResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateTrainingPipelineResponse;
    fromPartial(_: DeepPartial<MsgUpdateTrainingPipelineResponse>): MsgUpdateTrainingPipelineResponse;
};
/**
 * @name MsgRegisterModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterModelCard
 */
export declare const MsgRegisterModelCard: {
    typeUrl: string;
    encode(message: MsgRegisterModelCard, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterModelCard;
    fromPartial(object: DeepPartial<MsgRegisterModelCard>): MsgRegisterModelCard;
};
/**
 * @name MsgRegisterModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRegisterModelCardResponse
 */
export declare const MsgRegisterModelCardResponse: {
    typeUrl: string;
    encode(_: MsgRegisterModelCardResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterModelCardResponse;
    fromPartial(_: DeepPartial<MsgRegisterModelCardResponse>): MsgRegisterModelCardResponse;
};
/**
 * @name MsgUpdateModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateModelCard
 */
export declare const MsgUpdateModelCard: {
    typeUrl: string;
    encode(message: MsgUpdateModelCard, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateModelCard;
    fromPartial(object: DeepPartial<MsgUpdateModelCard>): MsgUpdateModelCard;
};
/**
 * @name MsgUpdateModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUpdateModelCardResponse
 */
export declare const MsgUpdateModelCardResponse: {
    typeUrl: string;
    encode(_: MsgUpdateModelCardResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateModelCardResponse;
    fromPartial(_: DeepPartial<MsgUpdateModelCardResponse>): MsgUpdateModelCardResponse;
};
/**
 * @name MsgRetireModelCard
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRetireModelCard
 */
export declare const MsgRetireModelCard: {
    typeUrl: string;
    encode(message: MsgRetireModelCard, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRetireModelCard;
    fromPartial(object: DeepPartial<MsgRetireModelCard>): MsgRetireModelCard;
};
/**
 * @name MsgRetireModelCardResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRetireModelCardResponse
 */
export declare const MsgRetireModelCardResponse: {
    typeUrl: string;
    encode(_: MsgRetireModelCardResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRetireModelCardResponse;
    fromPartial(_: DeepPartial<MsgRetireModelCardResponse>): MsgRetireModelCardResponse;
};
/**
 * @name MsgAmendTokenizerSpec
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTokenizerSpec
 */
export declare const MsgAmendTokenizerSpec: {
    typeUrl: string;
    encode(message: MsgAmendTokenizerSpec, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAmendTokenizerSpec;
    fromPartial(object: DeepPartial<MsgAmendTokenizerSpec>): MsgAmendTokenizerSpec;
};
/**
 * @name MsgAmendTokenizerSpecResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTokenizerSpecResponse
 */
export declare const MsgAmendTokenizerSpecResponse: {
    typeUrl: string;
    encode(message: MsgAmendTokenizerSpecResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAmendTokenizerSpecResponse;
    fromPartial(object: DeepPartial<MsgAmendTokenizerSpecResponse>): MsgAmendTokenizerSpecResponse;
};
/**
 * @name MsgAttributeContributions
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttributeContributions
 */
export declare const MsgAttributeContributions: {
    typeUrl: string;
    encode(message: MsgAttributeContributions, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAttributeContributions;
    fromPartial(object: DeepPartial<MsgAttributeContributions>): MsgAttributeContributions;
};
/**
 * @name MsgAttributeContributionsResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttributeContributionsResponse
 */
export declare const MsgAttributeContributionsResponse: {
    typeUrl: string;
    encode(message: MsgAttributeContributionsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAttributeContributionsResponse;
    fromPartial(object: DeepPartial<MsgAttributeContributionsResponse>): MsgAttributeContributionsResponse;
};
/**
 * @name MsgAttestTraining
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttestTraining
 */
export declare const MsgAttestTraining: {
    typeUrl: string;
    encode(message: MsgAttestTraining, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAttestTraining;
    fromPartial(object: DeepPartial<MsgAttestTraining>): MsgAttestTraining;
};
/**
 * @name MsgAttestTrainingResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAttestTrainingResponse
 */
export declare const MsgAttestTrainingResponse: {
    typeUrl: string;
    encode(_: MsgAttestTrainingResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAttestTrainingResponse;
    fromPartial(_: DeepPartial<MsgAttestTrainingResponse>): MsgAttestTrainingResponse;
};
/**
 * @name MsgCreateAugmentationBounty
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateAugmentationBounty
 */
export declare const MsgCreateAugmentationBounty: {
    typeUrl: string;
    encode(message: MsgCreateAugmentationBounty, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateAugmentationBounty;
    fromPartial(object: DeepPartial<MsgCreateAugmentationBounty>): MsgCreateAugmentationBounty;
};
/**
 * @name MsgCreateAugmentationBountyResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateAugmentationBountyResponse
 */
export declare const MsgCreateAugmentationBountyResponse: {
    typeUrl: string;
    encode(_: MsgCreateAugmentationBountyResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateAugmentationBountyResponse;
    fromPartial(_: DeepPartial<MsgCreateAugmentationBountyResponse>): MsgCreateAugmentationBountyResponse;
};
/**
 * @name MsgSubmitAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitAugmentation
 */
export declare const MsgSubmitAugmentation: {
    typeUrl: string;
    encode(message: MsgSubmitAugmentation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitAugmentation;
    fromPartial(object: DeepPartial<MsgSubmitAugmentation>): MsgSubmitAugmentation;
};
/**
 * @name MsgSubmitAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSubmitAugmentationResponse
 */
export declare const MsgSubmitAugmentationResponse: {
    typeUrl: string;
    encode(_: MsgSubmitAugmentationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitAugmentationResponse;
    fromPartial(_: DeepPartial<MsgSubmitAugmentationResponse>): MsgSubmitAugmentationResponse;
};
/**
 * @name MsgAcceptAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAcceptAugmentation
 */
export declare const MsgAcceptAugmentation: {
    typeUrl: string;
    encode(message: MsgAcceptAugmentation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAcceptAugmentation;
    fromPartial(object: DeepPartial<MsgAcceptAugmentation>): MsgAcceptAugmentation;
};
/**
 * @name MsgAcceptAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAcceptAugmentationResponse
 */
export declare const MsgAcceptAugmentationResponse: {
    typeUrl: string;
    encode(_: MsgAcceptAugmentationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAcceptAugmentationResponse;
    fromPartial(_: DeepPartial<MsgAcceptAugmentationResponse>): MsgAcceptAugmentationResponse;
};
/**
 * ─── Wave 4: reformulation verdicts ───────────────────────────────────────
 * @name MsgVoteOnAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteOnAugmentation
 */
export declare const MsgVoteOnAugmentation: {
    typeUrl: string;
    encode(message: MsgVoteOnAugmentation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteOnAugmentation;
    fromPartial(object: DeepPartial<MsgVoteOnAugmentation>): MsgVoteOnAugmentation;
};
/**
 * @name MsgVoteOnAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVoteOnAugmentationResponse
 */
export declare const MsgVoteOnAugmentationResponse: {
    typeUrl: string;
    encode(message: MsgVoteOnAugmentationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteOnAugmentationResponse;
    fromPartial(object: DeepPartial<MsgVoteOnAugmentationResponse>): MsgVoteOnAugmentationResponse;
};
/**
 * @name MsgSponsorVetoAugmentation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSponsorVetoAugmentation
 */
export declare const MsgSponsorVetoAugmentation: {
    typeUrl: string;
    encode(message: MsgSponsorVetoAugmentation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSponsorVetoAugmentation;
    fromPartial(object: DeepPartial<MsgSponsorVetoAugmentation>): MsgSponsorVetoAugmentation;
};
/**
 * @name MsgSponsorVetoAugmentationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgSponsorVetoAugmentationResponse
 */
export declare const MsgSponsorVetoAugmentationResponse: {
    typeUrl: string;
    encode(_: MsgSponsorVetoAugmentationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSponsorVetoAugmentationResponse;
    fromPartial(_: DeepPartial<MsgSponsorVetoAugmentationResponse>): MsgSponsorVetoAugmentationResponse;
};
/**
 * ─── Wave 4: attribution challenges ───────────────────────────────────────
 * @name MsgChallengeContribution
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeContribution
 */
export declare const MsgChallengeContribution: {
    typeUrl: string;
    encode(message: MsgChallengeContribution, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeContribution;
    fromPartial(object: DeepPartial<MsgChallengeContribution>): MsgChallengeContribution;
};
/**
 * @name MsgChallengeContributionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgChallengeContributionResponse
 */
export declare const MsgChallengeContributionResponse: {
    typeUrl: string;
    encode(message: MsgChallengeContributionResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgChallengeContributionResponse;
    fromPartial(object: DeepPartial<MsgChallengeContributionResponse>): MsgChallengeContributionResponse;
};
/**
 * @name MsgResolveContributionChallenge
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveContributionChallenge
 */
export declare const MsgResolveContributionChallenge: {
    typeUrl: string;
    encode(message: MsgResolveContributionChallenge, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveContributionChallenge;
    fromPartial(object: DeepPartial<MsgResolveContributionChallenge>): MsgResolveContributionChallenge;
};
/**
 * @name MsgResolveContributionChallengeResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveContributionChallengeResponse
 */
export declare const MsgResolveContributionChallengeResponse: {
    typeUrl: string;
    encode(message: MsgResolveContributionChallengeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveContributionChallengeResponse;
    fromPartial(object: DeepPartial<MsgResolveContributionChallengeResponse>): MsgResolveContributionChallengeResponse;
};
/**
 * ─── Wave 4: disabled training-fund disbursement placeholder ───────────────
 * @name MsgClaimTrainingFundDisbursement
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgClaimTrainingFundDisbursement
 */
export declare const MsgClaimTrainingFundDisbursement: {
    typeUrl: string;
    encode(message: MsgClaimTrainingFundDisbursement, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimTrainingFundDisbursement;
    fromPartial(object: DeepPartial<MsgClaimTrainingFundDisbursement>): MsgClaimTrainingFundDisbursement;
};
/**
 * @name MsgClaimTrainingFundDisbursementResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgClaimTrainingFundDisbursementResponse
 */
export declare const MsgClaimTrainingFundDisbursementResponse: {
    typeUrl: string;
    encode(message: MsgClaimTrainingFundDisbursementResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimTrainingFundDisbursementResponse;
    fromPartial(object: DeepPartial<MsgClaimTrainingFundDisbursementResponse>): MsgClaimTrainingFundDisbursementResponse;
};
/**
 * ─── Route B Wave 5: trace schema amendment ───────────────────────────────
 * @name MsgAmendTraceSchema
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTraceSchema
 */
export declare const MsgAmendTraceSchema: {
    typeUrl: string;
    encode(message: MsgAmendTraceSchema, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAmendTraceSchema;
    fromPartial(object: DeepPartial<MsgAmendTraceSchema>): MsgAmendTraceSchema;
};
/**
 * @name MsgAmendTraceSchemaResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgAmendTraceSchemaResponse
 */
export declare const MsgAmendTraceSchemaResponse: {
    typeUrl: string;
    encode(message: MsgAmendTraceSchemaResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAmendTraceSchemaResponse;
    fromPartial(object: DeepPartial<MsgAmendTraceSchemaResponse>): MsgAmendTraceSchemaResponse;
};
/**
 * ─── Route B Wave 7: training manifests ──────────────────────────────────
 * @name MsgCreateTrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateTrainingManifest
 */
export declare const MsgCreateTrainingManifest: {
    typeUrl: string;
    encode(message: MsgCreateTrainingManifest, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateTrainingManifest;
    fromPartial(object: DeepPartial<MsgCreateTrainingManifest>): MsgCreateTrainingManifest;
};
/**
 * @name MsgCreateTrainingManifestResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCreateTrainingManifestResponse
 */
export declare const MsgCreateTrainingManifestResponse: {
    typeUrl: string;
    encode(message: MsgCreateTrainingManifestResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateTrainingManifestResponse;
    fromPartial(object: DeepPartial<MsgCreateTrainingManifestResponse>): MsgCreateTrainingManifestResponse;
};
/**
 * @name MsgFinalizeTrainingManifest
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgFinalizeTrainingManifest
 */
export declare const MsgFinalizeTrainingManifest: {
    typeUrl: string;
    encode(message: MsgFinalizeTrainingManifest, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFinalizeTrainingManifest;
    fromPartial(object: DeepPartial<MsgFinalizeTrainingManifest>): MsgFinalizeTrainingManifest;
};
/**
 * @name MsgFinalizeTrainingManifestResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgFinalizeTrainingManifestResponse
 */
export declare const MsgFinalizeTrainingManifestResponse: {
    typeUrl: string;
    encode(message: MsgFinalizeTrainingManifestResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgFinalizeTrainingManifestResponse;
    fromPartial(object: DeepPartial<MsgFinalizeTrainingManifestResponse>): MsgFinalizeTrainingManifestResponse;
};
/**
 * @name MsgBindManifestToAttestation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgBindManifestToAttestation
 */
export declare const MsgBindManifestToAttestation: {
    typeUrl: string;
    encode(message: MsgBindManifestToAttestation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgBindManifestToAttestation;
    fromPartial(object: DeepPartial<MsgBindManifestToAttestation>): MsgBindManifestToAttestation;
};
/**
 * @name MsgBindManifestToAttestationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgBindManifestToAttestationResponse
 */
export declare const MsgBindManifestToAttestationResponse: {
    typeUrl: string;
    encode(_: MsgBindManifestToAttestationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgBindManifestToAttestationResponse;
    fromPartial(_: DeepPartial<MsgBindManifestToAttestationResponse>): MsgBindManifestToAttestationResponse;
};
/**
 * ─── Route B Wave 11: incident response ──────────────────────────────────
 * @name MsgOpenIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgOpenIncident
 */
export declare const MsgOpenIncident: {
    typeUrl: string;
    encode(message: MsgOpenIncident, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgOpenIncident;
    fromPartial(object: DeepPartial<MsgOpenIncident>): MsgOpenIncident;
};
/**
 * @name MsgOpenIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgOpenIncidentResponse
 */
export declare const MsgOpenIncidentResponse: {
    typeUrl: string;
    encode(message: MsgOpenIncidentResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgOpenIncidentResponse;
    fromPartial(object: DeepPartial<MsgOpenIncidentResponse>): MsgOpenIncidentResponse;
};
/**
 * @name MsgRecordRemediation
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRecordRemediation
 */
export declare const MsgRecordRemediation: {
    typeUrl: string;
    encode(message: MsgRecordRemediation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRecordRemediation;
    fromPartial(object: DeepPartial<MsgRecordRemediation>): MsgRecordRemediation;
};
/**
 * @name MsgRecordRemediationResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgRecordRemediationResponse
 */
export declare const MsgRecordRemediationResponse: {
    typeUrl: string;
    encode(message: MsgRecordRemediationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRecordRemediationResponse;
    fromPartial(object: DeepPartial<MsgRecordRemediationResponse>): MsgRecordRemediationResponse;
};
/**
 * @name MsgResolveIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveIncident
 */
export declare const MsgResolveIncident: {
    typeUrl: string;
    encode(message: MsgResolveIncident, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveIncident;
    fromPartial(object: DeepPartial<MsgResolveIncident>): MsgResolveIncident;
};
/**
 * @name MsgResolveIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgResolveIncidentResponse
 */
export declare const MsgResolveIncidentResponse: {
    typeUrl: string;
    encode(_: MsgResolveIncidentResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgResolveIncidentResponse;
    fromPartial(_: DeepPartial<MsgResolveIncidentResponse>): MsgResolveIncidentResponse;
};
/**
 * @name MsgCloseIncident
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCloseIncident
 */
export declare const MsgCloseIncident: {
    typeUrl: string;
    encode(message: MsgCloseIncident, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCloseIncident;
    fromPartial(object: DeepPartial<MsgCloseIncident>): MsgCloseIncident;
};
/**
 * @name MsgCloseIncidentResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCloseIncidentResponse
 */
export declare const MsgCloseIncidentResponse: {
    typeUrl: string;
    encode(_: MsgCloseIncidentResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCloseIncidentResponse;
    fromPartial(_: DeepPartial<MsgCloseIncidentResponse>): MsgCloseIncidentResponse;
};
/**
 * ─── Route B Wave 12: module circuit breakers ────────────────────────────
 * @name MsgPauseModule
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPauseModule
 */
export declare const MsgPauseModule: {
    typeUrl: string;
    encode(message: MsgPauseModule, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseModule;
    fromPartial(object: DeepPartial<MsgPauseModule>): MsgPauseModule;
};
/**
 * @name MsgPauseModuleResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgPauseModuleResponse
 */
export declare const MsgPauseModuleResponse: {
    typeUrl: string;
    encode(message: MsgPauseModuleResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseModuleResponse;
    fromPartial(object: DeepPartial<MsgPauseModuleResponse>): MsgPauseModuleResponse;
};
/**
 * @name MsgUnpauseModule
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUnpauseModule
 */
export declare const MsgUnpauseModule: {
    typeUrl: string;
    encode(message: MsgUnpauseModule, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUnpauseModule;
    fromPartial(object: DeepPartial<MsgUnpauseModule>): MsgUnpauseModule;
};
/**
 * @name MsgUnpauseModuleResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgUnpauseModuleResponse
 */
export declare const MsgUnpauseModuleResponse: {
    typeUrl: string;
    encode(_: MsgUnpauseModuleResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUnpauseModuleResponse;
    fromPartial(_: DeepPartial<MsgUnpauseModuleResponse>): MsgUnpauseModuleResponse;
};
/**
 * @name MsgCorrectManifestMerkleRoot
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCorrectManifestMerkleRoot
 */
export declare const MsgCorrectManifestMerkleRoot: {
    typeUrl: string;
    encode(message: MsgCorrectManifestMerkleRoot, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCorrectManifestMerkleRoot;
    fromPartial(object: DeepPartial<MsgCorrectManifestMerkleRoot>): MsgCorrectManifestMerkleRoot;
};
/**
 * @name MsgCorrectManifestMerkleRootResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgCorrectManifestMerkleRootResponse
 */
export declare const MsgCorrectManifestMerkleRootResponse: {
    typeUrl: string;
    encode(message: MsgCorrectManifestMerkleRootResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCorrectManifestMerkleRootResponse;
    fromPartial(object: DeepPartial<MsgCorrectManifestMerkleRootResponse>): MsgCorrectManifestMerkleRootResponse;
};
/**
 * MsgVetoFactInjection — a registered guardian cancels a pending
 * authority-injected fact during the veto window. The fact never
 * materializes; the privileged-action log records the veto for audit.
 * @name MsgVetoFactInjection
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVetoFactInjection
 */
export declare const MsgVetoFactInjection: {
    typeUrl: string;
    encode(message: MsgVetoFactInjection, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVetoFactInjection;
    fromPartial(object: DeepPartial<MsgVetoFactInjection>): MsgVetoFactInjection;
};
/**
 * @name MsgVetoFactInjectionResponse
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.MsgVetoFactInjectionResponse
 */
export declare const MsgVetoFactInjectionResponse: {
    typeUrl: string;
    encode(_: MsgVetoFactInjectionResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVetoFactInjectionResponse;
    fromPartial(_: DeepPartial<MsgVetoFactInjectionResponse>): MsgVetoFactInjectionResponse;
};
