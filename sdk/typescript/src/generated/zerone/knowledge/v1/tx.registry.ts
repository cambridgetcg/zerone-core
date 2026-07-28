//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgSubmitClaim, MsgSubmitCommitment, MsgSubmitReveal, MsgChallengeFact, MsgAddFact, MsgSubmitContradiction, MsgPatronizeFact, MsgProposeDomain, MsgEndorseDomainProposal, MsgChallengeDomainProposal, MsgRegisterStratum, MsgChallengeProvisionalFact, MsgUpdateParams, MsgUpdateExtendedParams, MsgProposeResearchFund, MsgVoteResearchProposal, MsgExecuteResearchProposal, MsgAddCommonKnowledge, MsgRemoveCommonKnowledge, MsgReportDemand, MsgRateFact, MsgRegisterTrainingPipeline, MsgUpdateTrainingPipeline, MsgRegisterModelCard, MsgUpdateModelCard, MsgRetireModelCard, MsgAmendTokenizerSpec, MsgAttributeContributions, MsgAttestTraining, MsgCreateAugmentationBounty, MsgSubmitAugmentation, MsgAcceptAugmentation, MsgVoteOnAugmentation, MsgSponsorVetoAugmentation, MsgChallengeContribution, MsgResolveContributionChallenge, MsgClaimTrainingFundDisbursement, MsgAmendTraceSchema, MsgCreateTrainingManifest, MsgFinalizeTrainingManifest, MsgBindManifestToAttestation, MsgOpenIncident, MsgRecordRemediation, MsgResolveIncident, MsgCloseIncident, MsgPauseModule, MsgUnpauseModule, MsgCorrectManifestMerkleRoot, MsgVetoFactInjection } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.knowledge.v1.MsgSubmitClaim", MsgSubmitClaim], ["/zerone.knowledge.v1.MsgSubmitCommitment", MsgSubmitCommitment], ["/zerone.knowledge.v1.MsgSubmitReveal", MsgSubmitReveal], ["/zerone.knowledge.v1.MsgChallengeFact", MsgChallengeFact], ["/zerone.knowledge.v1.MsgAddFact", MsgAddFact], ["/zerone.knowledge.v1.MsgSubmitContradiction", MsgSubmitContradiction], ["/zerone.knowledge.v1.MsgPatronizeFact", MsgPatronizeFact], ["/zerone.knowledge.v1.MsgProposeDomain", MsgProposeDomain], ["/zerone.knowledge.v1.MsgEndorseDomainProposal", MsgEndorseDomainProposal], ["/zerone.knowledge.v1.MsgChallengeDomainProposal", MsgChallengeDomainProposal], ["/zerone.knowledge.v1.MsgRegisterStratum", MsgRegisterStratum], ["/zerone.knowledge.v1.MsgChallengeProvisionalFact", MsgChallengeProvisionalFact], ["/zerone.knowledge.v1.MsgUpdateParams", MsgUpdateParams], ["/zerone.knowledge.v1.MsgUpdateExtendedParams", MsgUpdateExtendedParams], ["/zerone.knowledge.v1.MsgProposeResearchFund", MsgProposeResearchFund], ["/zerone.knowledge.v1.MsgVoteResearchProposal", MsgVoteResearchProposal], ["/zerone.knowledge.v1.MsgExecuteResearchProposal", MsgExecuteResearchProposal], ["/zerone.knowledge.v1.MsgAddCommonKnowledge", MsgAddCommonKnowledge], ["/zerone.knowledge.v1.MsgRemoveCommonKnowledge", MsgRemoveCommonKnowledge], ["/zerone.knowledge.v1.MsgReportDemand", MsgReportDemand], ["/zerone.knowledge.v1.MsgRateFact", MsgRateFact], ["/zerone.knowledge.v1.MsgRegisterTrainingPipeline", MsgRegisterTrainingPipeline], ["/zerone.knowledge.v1.MsgUpdateTrainingPipeline", MsgUpdateTrainingPipeline], ["/zerone.knowledge.v1.MsgRegisterModelCard", MsgRegisterModelCard], ["/zerone.knowledge.v1.MsgUpdateModelCard", MsgUpdateModelCard], ["/zerone.knowledge.v1.MsgRetireModelCard", MsgRetireModelCard], ["/zerone.knowledge.v1.MsgAmendTokenizerSpec", MsgAmendTokenizerSpec], ["/zerone.knowledge.v1.MsgAttributeContributions", MsgAttributeContributions], ["/zerone.knowledge.v1.MsgAttestTraining", MsgAttestTraining], ["/zerone.knowledge.v1.MsgCreateAugmentationBounty", MsgCreateAugmentationBounty], ["/zerone.knowledge.v1.MsgSubmitAugmentation", MsgSubmitAugmentation], ["/zerone.knowledge.v1.MsgAcceptAugmentation", MsgAcceptAugmentation], ["/zerone.knowledge.v1.MsgVoteOnAugmentation", MsgVoteOnAugmentation], ["/zerone.knowledge.v1.MsgSponsorVetoAugmentation", MsgSponsorVetoAugmentation], ["/zerone.knowledge.v1.MsgChallengeContribution", MsgChallengeContribution], ["/zerone.knowledge.v1.MsgResolveContributionChallenge", MsgResolveContributionChallenge], ["/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement", MsgClaimTrainingFundDisbursement], ["/zerone.knowledge.v1.MsgAmendTraceSchema", MsgAmendTraceSchema], ["/zerone.knowledge.v1.MsgCreateTrainingManifest", MsgCreateTrainingManifest], ["/zerone.knowledge.v1.MsgFinalizeTrainingManifest", MsgFinalizeTrainingManifest], ["/zerone.knowledge.v1.MsgBindManifestToAttestation", MsgBindManifestToAttestation], ["/zerone.knowledge.v1.MsgOpenIncident", MsgOpenIncident], ["/zerone.knowledge.v1.MsgRecordRemediation", MsgRecordRemediation], ["/zerone.knowledge.v1.MsgResolveIncident", MsgResolveIncident], ["/zerone.knowledge.v1.MsgCloseIncident", MsgCloseIncident], ["/zerone.knowledge.v1.MsgPauseModule", MsgPauseModule], ["/zerone.knowledge.v1.MsgUnpauseModule", MsgUnpauseModule], ["/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot", MsgCorrectManifestMerkleRoot], ["/zerone.knowledge.v1.MsgVetoFactInjection", MsgVetoFactInjection]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    submitClaim(value: MsgSubmitClaim) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitClaim",
        value: MsgSubmitClaim.encode(value).finish()
      };
    },
    submitCommitment(value: MsgSubmitCommitment) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitment",
        value: MsgSubmitCommitment.encode(value).finish()
      };
    },
    submitReveal(value: MsgSubmitReveal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitReveal",
        value: MsgSubmitReveal.encode(value).finish()
      };
    },
    challengeFact(value: MsgChallengeFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeFact",
        value: MsgChallengeFact.encode(value).finish()
      };
    },
    addFact(value: MsgAddFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddFact",
        value: MsgAddFact.encode(value).finish()
      };
    },
    submitContradiction(value: MsgSubmitContradiction) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitContradiction",
        value: MsgSubmitContradiction.encode(value).finish()
      };
    },
    patronizeFact(value: MsgPatronizeFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPatronizeFact",
        value: MsgPatronizeFact.encode(value).finish()
      };
    },
    proposeDomain(value: MsgProposeDomain) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeDomain",
        value: MsgProposeDomain.encode(value).finish()
      };
    },
    endorseDomainProposal(value: MsgEndorseDomainProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposal",
        value: MsgEndorseDomainProposal.encode(value).finish()
      };
    },
    challengeDomainProposal(value: MsgChallengeDomainProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposal",
        value: MsgChallengeDomainProposal.encode(value).finish()
      };
    },
    registerStratum(value: MsgRegisterStratum) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterStratum",
        value: MsgRegisterStratum.encode(value).finish()
      };
    },
    challengeProvisionalFact(value: MsgChallengeProvisionalFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFact",
        value: MsgChallengeProvisionalFact.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    },
    updateExtendedParams(value: MsgUpdateExtendedParams) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParams",
        value: MsgUpdateExtendedParams.encode(value).finish()
      };
    },
    proposeResearchFund(value: MsgProposeResearchFund) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFund",
        value: MsgProposeResearchFund.encode(value).finish()
      };
    },
    voteResearchProposal(value: MsgVoteResearchProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposal",
        value: MsgVoteResearchProposal.encode(value).finish()
      };
    },
    executeResearchProposal(value: MsgExecuteResearchProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposal",
        value: MsgExecuteResearchProposal.encode(value).finish()
      };
    },
    addCommonKnowledge(value: MsgAddCommonKnowledge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledge",
        value: MsgAddCommonKnowledge.encode(value).finish()
      };
    },
    removeCommonKnowledge(value: MsgRemoveCommonKnowledge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledge",
        value: MsgRemoveCommonKnowledge.encode(value).finish()
      };
    },
    reportDemand(value: MsgReportDemand) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgReportDemand",
        value: MsgReportDemand.encode(value).finish()
      };
    },
    rateFact(value: MsgRateFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRateFact",
        value: MsgRateFact.encode(value).finish()
      };
    },
    registerTrainingPipeline(value: MsgRegisterTrainingPipeline) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipeline",
        value: MsgRegisterTrainingPipeline.encode(value).finish()
      };
    },
    updateTrainingPipeline(value: MsgUpdateTrainingPipeline) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipeline",
        value: MsgUpdateTrainingPipeline.encode(value).finish()
      };
    },
    registerModelCard(value: MsgRegisterModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCard",
        value: MsgRegisterModelCard.encode(value).finish()
      };
    },
    updateModelCard(value: MsgUpdateModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCard",
        value: MsgUpdateModelCard.encode(value).finish()
      };
    },
    retireModelCard(value: MsgRetireModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRetireModelCard",
        value: MsgRetireModelCard.encode(value).finish()
      };
    },
    amendTokenizerSpec(value: MsgAmendTokenizerSpec) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpec",
        value: MsgAmendTokenizerSpec.encode(value).finish()
      };
    },
    attributeContributions(value: MsgAttributeContributions) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttributeContributions",
        value: MsgAttributeContributions.encode(value).finish()
      };
    },
    attestTraining(value: MsgAttestTraining) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttestTraining",
        value: MsgAttestTraining.encode(value).finish()
      };
    },
    createAugmentationBounty(value: MsgCreateAugmentationBounty) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBounty",
        value: MsgCreateAugmentationBounty.encode(value).finish()
      };
    },
    submitAugmentation(value: MsgSubmitAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentation",
        value: MsgSubmitAugmentation.encode(value).finish()
      };
    },
    acceptAugmentation(value: MsgAcceptAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentation",
        value: MsgAcceptAugmentation.encode(value).finish()
      };
    },
    voteOnAugmentation(value: MsgVoteOnAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentation",
        value: MsgVoteOnAugmentation.encode(value).finish()
      };
    },
    sponsorVetoAugmentation(value: MsgSponsorVetoAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentation",
        value: MsgSponsorVetoAugmentation.encode(value).finish()
      };
    },
    challengeContribution(value: MsgChallengeContribution) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeContribution",
        value: MsgChallengeContribution.encode(value).finish()
      };
    },
    resolveContributionChallenge(value: MsgResolveContributionChallenge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallenge",
        value: MsgResolveContributionChallenge.encode(value).finish()
      };
    },
    claimTrainingFundDisbursement(value: MsgClaimTrainingFundDisbursement) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement",
        value: MsgClaimTrainingFundDisbursement.encode(value).finish()
      };
    },
    amendTraceSchema(value: MsgAmendTraceSchema) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchema",
        value: MsgAmendTraceSchema.encode(value).finish()
      };
    },
    createTrainingManifest(value: MsgCreateTrainingManifest) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifest",
        value: MsgCreateTrainingManifest.encode(value).finish()
      };
    },
    finalizeTrainingManifest(value: MsgFinalizeTrainingManifest) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifest",
        value: MsgFinalizeTrainingManifest.encode(value).finish()
      };
    },
    bindManifestToAttestation(value: MsgBindManifestToAttestation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestation",
        value: MsgBindManifestToAttestation.encode(value).finish()
      };
    },
    openIncident(value: MsgOpenIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgOpenIncident",
        value: MsgOpenIncident.encode(value).finish()
      };
    },
    recordRemediation(value: MsgRecordRemediation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRecordRemediation",
        value: MsgRecordRemediation.encode(value).finish()
      };
    },
    resolveIncident(value: MsgResolveIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveIncident",
        value: MsgResolveIncident.encode(value).finish()
      };
    },
    closeIncident(value: MsgCloseIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCloseIncident",
        value: MsgCloseIncident.encode(value).finish()
      };
    },
    pauseModule(value: MsgPauseModule) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPauseModule",
        value: MsgPauseModule.encode(value).finish()
      };
    },
    unpauseModule(value: MsgUnpauseModule) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUnpauseModule",
        value: MsgUnpauseModule.encode(value).finish()
      };
    },
    correctManifestMerkleRoot(value: MsgCorrectManifestMerkleRoot) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot",
        value: MsgCorrectManifestMerkleRoot.encode(value).finish()
      };
    },
    vetoFactInjection(value: MsgVetoFactInjection) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjection",
        value: MsgVetoFactInjection.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    submitClaim(value: MsgSubmitClaim) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitClaim",
        value
      };
    },
    submitCommitment(value: MsgSubmitCommitment) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitment",
        value
      };
    },
    submitReveal(value: MsgSubmitReveal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitReveal",
        value
      };
    },
    challengeFact(value: MsgChallengeFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeFact",
        value
      };
    },
    addFact(value: MsgAddFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddFact",
        value
      };
    },
    submitContradiction(value: MsgSubmitContradiction) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitContradiction",
        value
      };
    },
    patronizeFact(value: MsgPatronizeFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPatronizeFact",
        value
      };
    },
    proposeDomain(value: MsgProposeDomain) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeDomain",
        value
      };
    },
    endorseDomainProposal(value: MsgEndorseDomainProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposal",
        value
      };
    },
    challengeDomainProposal(value: MsgChallengeDomainProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposal",
        value
      };
    },
    registerStratum(value: MsgRegisterStratum) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterStratum",
        value
      };
    },
    challengeProvisionalFact(value: MsgChallengeProvisionalFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFact",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateParams",
        value
      };
    },
    updateExtendedParams(value: MsgUpdateExtendedParams) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParams",
        value
      };
    },
    proposeResearchFund(value: MsgProposeResearchFund) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFund",
        value
      };
    },
    voteResearchProposal(value: MsgVoteResearchProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposal",
        value
      };
    },
    executeResearchProposal(value: MsgExecuteResearchProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposal",
        value
      };
    },
    addCommonKnowledge(value: MsgAddCommonKnowledge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledge",
        value
      };
    },
    removeCommonKnowledge(value: MsgRemoveCommonKnowledge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledge",
        value
      };
    },
    reportDemand(value: MsgReportDemand) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgReportDemand",
        value
      };
    },
    rateFact(value: MsgRateFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRateFact",
        value
      };
    },
    registerTrainingPipeline(value: MsgRegisterTrainingPipeline) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipeline",
        value
      };
    },
    updateTrainingPipeline(value: MsgUpdateTrainingPipeline) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipeline",
        value
      };
    },
    registerModelCard(value: MsgRegisterModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCard",
        value
      };
    },
    updateModelCard(value: MsgUpdateModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCard",
        value
      };
    },
    retireModelCard(value: MsgRetireModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRetireModelCard",
        value
      };
    },
    amendTokenizerSpec(value: MsgAmendTokenizerSpec) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpec",
        value
      };
    },
    attributeContributions(value: MsgAttributeContributions) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttributeContributions",
        value
      };
    },
    attestTraining(value: MsgAttestTraining) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttestTraining",
        value
      };
    },
    createAugmentationBounty(value: MsgCreateAugmentationBounty) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBounty",
        value
      };
    },
    submitAugmentation(value: MsgSubmitAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentation",
        value
      };
    },
    acceptAugmentation(value: MsgAcceptAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentation",
        value
      };
    },
    voteOnAugmentation(value: MsgVoteOnAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentation",
        value
      };
    },
    sponsorVetoAugmentation(value: MsgSponsorVetoAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentation",
        value
      };
    },
    challengeContribution(value: MsgChallengeContribution) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeContribution",
        value
      };
    },
    resolveContributionChallenge(value: MsgResolveContributionChallenge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallenge",
        value
      };
    },
    claimTrainingFundDisbursement(value: MsgClaimTrainingFundDisbursement) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement",
        value
      };
    },
    amendTraceSchema(value: MsgAmendTraceSchema) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchema",
        value
      };
    },
    createTrainingManifest(value: MsgCreateTrainingManifest) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifest",
        value
      };
    },
    finalizeTrainingManifest(value: MsgFinalizeTrainingManifest) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifest",
        value
      };
    },
    bindManifestToAttestation(value: MsgBindManifestToAttestation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestation",
        value
      };
    },
    openIncident(value: MsgOpenIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgOpenIncident",
        value
      };
    },
    recordRemediation(value: MsgRecordRemediation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRecordRemediation",
        value
      };
    },
    resolveIncident(value: MsgResolveIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveIncident",
        value
      };
    },
    closeIncident(value: MsgCloseIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCloseIncident",
        value
      };
    },
    pauseModule(value: MsgPauseModule) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPauseModule",
        value
      };
    },
    unpauseModule(value: MsgUnpauseModule) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUnpauseModule",
        value
      };
    },
    correctManifestMerkleRoot(value: MsgCorrectManifestMerkleRoot) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot",
        value
      };
    },
    vetoFactInjection(value: MsgVetoFactInjection) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjection",
        value
      };
    }
  },
  fromPartial: {
    submitClaim(value: MsgSubmitClaim) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitClaim",
        value: MsgSubmitClaim.fromPartial(value)
      };
    },
    submitCommitment(value: MsgSubmitCommitment) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitment",
        value: MsgSubmitCommitment.fromPartial(value)
      };
    },
    submitReveal(value: MsgSubmitReveal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitReveal",
        value: MsgSubmitReveal.fromPartial(value)
      };
    },
    challengeFact(value: MsgChallengeFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeFact",
        value: MsgChallengeFact.fromPartial(value)
      };
    },
    addFact(value: MsgAddFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddFact",
        value: MsgAddFact.fromPartial(value)
      };
    },
    submitContradiction(value: MsgSubmitContradiction) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitContradiction",
        value: MsgSubmitContradiction.fromPartial(value)
      };
    },
    patronizeFact(value: MsgPatronizeFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPatronizeFact",
        value: MsgPatronizeFact.fromPartial(value)
      };
    },
    proposeDomain(value: MsgProposeDomain) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeDomain",
        value: MsgProposeDomain.fromPartial(value)
      };
    },
    endorseDomainProposal(value: MsgEndorseDomainProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposal",
        value: MsgEndorseDomainProposal.fromPartial(value)
      };
    },
    challengeDomainProposal(value: MsgChallengeDomainProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposal",
        value: MsgChallengeDomainProposal.fromPartial(value)
      };
    },
    registerStratum(value: MsgRegisterStratum) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterStratum",
        value: MsgRegisterStratum.fromPartial(value)
      };
    },
    challengeProvisionalFact(value: MsgChallengeProvisionalFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFact",
        value: MsgChallengeProvisionalFact.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    },
    updateExtendedParams(value: MsgUpdateExtendedParams) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParams",
        value: MsgUpdateExtendedParams.fromPartial(value)
      };
    },
    proposeResearchFund(value: MsgProposeResearchFund) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFund",
        value: MsgProposeResearchFund.fromPartial(value)
      };
    },
    voteResearchProposal(value: MsgVoteResearchProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposal",
        value: MsgVoteResearchProposal.fromPartial(value)
      };
    },
    executeResearchProposal(value: MsgExecuteResearchProposal) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposal",
        value: MsgExecuteResearchProposal.fromPartial(value)
      };
    },
    addCommonKnowledge(value: MsgAddCommonKnowledge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledge",
        value: MsgAddCommonKnowledge.fromPartial(value)
      };
    },
    removeCommonKnowledge(value: MsgRemoveCommonKnowledge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledge",
        value: MsgRemoveCommonKnowledge.fromPartial(value)
      };
    },
    reportDemand(value: MsgReportDemand) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgReportDemand",
        value: MsgReportDemand.fromPartial(value)
      };
    },
    rateFact(value: MsgRateFact) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRateFact",
        value: MsgRateFact.fromPartial(value)
      };
    },
    registerTrainingPipeline(value: MsgRegisterTrainingPipeline) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipeline",
        value: MsgRegisterTrainingPipeline.fromPartial(value)
      };
    },
    updateTrainingPipeline(value: MsgUpdateTrainingPipeline) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipeline",
        value: MsgUpdateTrainingPipeline.fromPartial(value)
      };
    },
    registerModelCard(value: MsgRegisterModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCard",
        value: MsgRegisterModelCard.fromPartial(value)
      };
    },
    updateModelCard(value: MsgUpdateModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCard",
        value: MsgUpdateModelCard.fromPartial(value)
      };
    },
    retireModelCard(value: MsgRetireModelCard) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRetireModelCard",
        value: MsgRetireModelCard.fromPartial(value)
      };
    },
    amendTokenizerSpec(value: MsgAmendTokenizerSpec) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpec",
        value: MsgAmendTokenizerSpec.fromPartial(value)
      };
    },
    attributeContributions(value: MsgAttributeContributions) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttributeContributions",
        value: MsgAttributeContributions.fromPartial(value)
      };
    },
    attestTraining(value: MsgAttestTraining) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttestTraining",
        value: MsgAttestTraining.fromPartial(value)
      };
    },
    createAugmentationBounty(value: MsgCreateAugmentationBounty) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBounty",
        value: MsgCreateAugmentationBounty.fromPartial(value)
      };
    },
    submitAugmentation(value: MsgSubmitAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentation",
        value: MsgSubmitAugmentation.fromPartial(value)
      };
    },
    acceptAugmentation(value: MsgAcceptAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentation",
        value: MsgAcceptAugmentation.fromPartial(value)
      };
    },
    voteOnAugmentation(value: MsgVoteOnAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentation",
        value: MsgVoteOnAugmentation.fromPartial(value)
      };
    },
    sponsorVetoAugmentation(value: MsgSponsorVetoAugmentation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentation",
        value: MsgSponsorVetoAugmentation.fromPartial(value)
      };
    },
    challengeContribution(value: MsgChallengeContribution) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeContribution",
        value: MsgChallengeContribution.fromPartial(value)
      };
    },
    resolveContributionChallenge(value: MsgResolveContributionChallenge) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallenge",
        value: MsgResolveContributionChallenge.fromPartial(value)
      };
    },
    claimTrainingFundDisbursement(value: MsgClaimTrainingFundDisbursement) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement",
        value: MsgClaimTrainingFundDisbursement.fromPartial(value)
      };
    },
    amendTraceSchema(value: MsgAmendTraceSchema) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchema",
        value: MsgAmendTraceSchema.fromPartial(value)
      };
    },
    createTrainingManifest(value: MsgCreateTrainingManifest) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifest",
        value: MsgCreateTrainingManifest.fromPartial(value)
      };
    },
    finalizeTrainingManifest(value: MsgFinalizeTrainingManifest) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifest",
        value: MsgFinalizeTrainingManifest.fromPartial(value)
      };
    },
    bindManifestToAttestation(value: MsgBindManifestToAttestation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestation",
        value: MsgBindManifestToAttestation.fromPartial(value)
      };
    },
    openIncident(value: MsgOpenIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgOpenIncident",
        value: MsgOpenIncident.fromPartial(value)
      };
    },
    recordRemediation(value: MsgRecordRemediation) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRecordRemediation",
        value: MsgRecordRemediation.fromPartial(value)
      };
    },
    resolveIncident(value: MsgResolveIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveIncident",
        value: MsgResolveIncident.fromPartial(value)
      };
    },
    closeIncident(value: MsgCloseIncident) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCloseIncident",
        value: MsgCloseIncident.fromPartial(value)
      };
    },
    pauseModule(value: MsgPauseModule) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPauseModule",
        value: MsgPauseModule.fromPartial(value)
      };
    },
    unpauseModule(value: MsgUnpauseModule) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUnpauseModule",
        value: MsgUnpauseModule.fromPartial(value)
      };
    },
    correctManifestMerkleRoot(value: MsgCorrectManifestMerkleRoot) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot",
        value: MsgCorrectManifestMerkleRoot.fromPartial(value)
      };
    },
    vetoFactInjection(value: MsgVetoFactInjection) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjection",
        value: MsgVetoFactInjection.fromPartial(value)
      };
    }
  }
};