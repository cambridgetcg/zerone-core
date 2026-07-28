import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgSubmitClaim, MsgSubmitCommitment, MsgSubmitReveal, MsgChallengeFact, MsgAddFact, MsgSubmitContradiction, MsgPatronizeFact, MsgProposeDomain, MsgEndorseDomainProposal, MsgChallengeDomainProposal, MsgRegisterStratum, MsgChallengeProvisionalFact, MsgUpdateParams, MsgUpdateExtendedParams, MsgProposeResearchFund, MsgVoteResearchProposal, MsgExecuteResearchProposal, MsgAddCommonKnowledge, MsgRemoveCommonKnowledge, MsgReportDemand, MsgRateFact, MsgRegisterTrainingPipeline, MsgUpdateTrainingPipeline, MsgRegisterModelCard, MsgUpdateModelCard, MsgRetireModelCard, MsgAmendTokenizerSpec, MsgAttributeContributions, MsgAttestTraining, MsgCreateAugmentationBounty, MsgSubmitAugmentation, MsgAcceptAugmentation, MsgVoteOnAugmentation, MsgSponsorVetoAugmentation, MsgChallengeContribution, MsgResolveContributionChallenge, MsgClaimTrainingFundDisbursement, MsgAmendTraceSchema, MsgCreateTrainingManifest, MsgFinalizeTrainingManifest, MsgBindManifestToAttestation, MsgOpenIncident, MsgRecordRemediation, MsgResolveIncident, MsgCloseIncident, MsgPauseModule, MsgUnpauseModule, MsgCorrectManifestMerkleRoot, MsgVetoFactInjection } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        submitClaim(value: MsgSubmitClaim): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        submitCommitment(value: MsgSubmitCommitment): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        submitReveal(value: MsgSubmitReveal): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        challengeFact(value: MsgChallengeFact): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        addFact(value: MsgAddFact): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        submitContradiction(value: MsgSubmitContradiction): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        patronizeFact(value: MsgPatronizeFact): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        proposeDomain(value: MsgProposeDomain): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        endorseDomainProposal(value: MsgEndorseDomainProposal): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        challengeDomainProposal(value: MsgChallengeDomainProposal): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        registerStratum(value: MsgRegisterStratum): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        challengeProvisionalFact(value: MsgChallengeProvisionalFact): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateExtendedParams(value: MsgUpdateExtendedParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        proposeResearchFund(value: MsgProposeResearchFund): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        voteResearchProposal(value: MsgVoteResearchProposal): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        executeResearchProposal(value: MsgExecuteResearchProposal): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        addCommonKnowledge(value: MsgAddCommonKnowledge): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        removeCommonKnowledge(value: MsgRemoveCommonKnowledge): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        reportDemand(value: MsgReportDemand): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        rateFact(value: MsgRateFact): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        registerTrainingPipeline(value: MsgRegisterTrainingPipeline): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateTrainingPipeline(value: MsgUpdateTrainingPipeline): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        registerModelCard(value: MsgRegisterModelCard): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateModelCard(value: MsgUpdateModelCard): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        retireModelCard(value: MsgRetireModelCard): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        amendTokenizerSpec(value: MsgAmendTokenizerSpec): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        attributeContributions(value: MsgAttributeContributions): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        attestTraining(value: MsgAttestTraining): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        createAugmentationBounty(value: MsgCreateAugmentationBounty): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        submitAugmentation(value: MsgSubmitAugmentation): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        acceptAugmentation(value: MsgAcceptAugmentation): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        voteOnAugmentation(value: MsgVoteOnAugmentation): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        sponsorVetoAugmentation(value: MsgSponsorVetoAugmentation): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        challengeContribution(value: MsgChallengeContribution): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        resolveContributionChallenge(value: MsgResolveContributionChallenge): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        claimTrainingFundDisbursement(value: MsgClaimTrainingFundDisbursement): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        amendTraceSchema(value: MsgAmendTraceSchema): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        createTrainingManifest(value: MsgCreateTrainingManifest): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        finalizeTrainingManifest(value: MsgFinalizeTrainingManifest): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        bindManifestToAttestation(value: MsgBindManifestToAttestation): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        openIncident(value: MsgOpenIncident): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        recordRemediation(value: MsgRecordRemediation): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        resolveIncident(value: MsgResolveIncident): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        closeIncident(value: MsgCloseIncident): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        pauseModule(value: MsgPauseModule): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        unpauseModule(value: MsgUnpauseModule): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        correctManifestMerkleRoot(value: MsgCorrectManifestMerkleRoot): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        vetoFactInjection(value: MsgVetoFactInjection): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        submitClaim(value: MsgSubmitClaim): {
            typeUrl: string;
            value: MsgSubmitClaim;
        };
        submitCommitment(value: MsgSubmitCommitment): {
            typeUrl: string;
            value: MsgSubmitCommitment;
        };
        submitReveal(value: MsgSubmitReveal): {
            typeUrl: string;
            value: MsgSubmitReveal;
        };
        challengeFact(value: MsgChallengeFact): {
            typeUrl: string;
            value: MsgChallengeFact;
        };
        addFact(value: MsgAddFact): {
            typeUrl: string;
            value: MsgAddFact;
        };
        submitContradiction(value: MsgSubmitContradiction): {
            typeUrl: string;
            value: MsgSubmitContradiction;
        };
        patronizeFact(value: MsgPatronizeFact): {
            typeUrl: string;
            value: MsgPatronizeFact;
        };
        proposeDomain(value: MsgProposeDomain): {
            typeUrl: string;
            value: MsgProposeDomain;
        };
        endorseDomainProposal(value: MsgEndorseDomainProposal): {
            typeUrl: string;
            value: MsgEndorseDomainProposal;
        };
        challengeDomainProposal(value: MsgChallengeDomainProposal): {
            typeUrl: string;
            value: MsgChallengeDomainProposal;
        };
        registerStratum(value: MsgRegisterStratum): {
            typeUrl: string;
            value: MsgRegisterStratum;
        };
        challengeProvisionalFact(value: MsgChallengeProvisionalFact): {
            typeUrl: string;
            value: MsgChallengeProvisionalFact;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
        updateExtendedParams(value: MsgUpdateExtendedParams): {
            typeUrl: string;
            value: MsgUpdateExtendedParams;
        };
        proposeResearchFund(value: MsgProposeResearchFund): {
            typeUrl: string;
            value: MsgProposeResearchFund;
        };
        voteResearchProposal(value: MsgVoteResearchProposal): {
            typeUrl: string;
            value: MsgVoteResearchProposal;
        };
        executeResearchProposal(value: MsgExecuteResearchProposal): {
            typeUrl: string;
            value: MsgExecuteResearchProposal;
        };
        addCommonKnowledge(value: MsgAddCommonKnowledge): {
            typeUrl: string;
            value: MsgAddCommonKnowledge;
        };
        removeCommonKnowledge(value: MsgRemoveCommonKnowledge): {
            typeUrl: string;
            value: MsgRemoveCommonKnowledge;
        };
        reportDemand(value: MsgReportDemand): {
            typeUrl: string;
            value: MsgReportDemand;
        };
        rateFact(value: MsgRateFact): {
            typeUrl: string;
            value: MsgRateFact;
        };
        registerTrainingPipeline(value: MsgRegisterTrainingPipeline): {
            typeUrl: string;
            value: MsgRegisterTrainingPipeline;
        };
        updateTrainingPipeline(value: MsgUpdateTrainingPipeline): {
            typeUrl: string;
            value: MsgUpdateTrainingPipeline;
        };
        registerModelCard(value: MsgRegisterModelCard): {
            typeUrl: string;
            value: MsgRegisterModelCard;
        };
        updateModelCard(value: MsgUpdateModelCard): {
            typeUrl: string;
            value: MsgUpdateModelCard;
        };
        retireModelCard(value: MsgRetireModelCard): {
            typeUrl: string;
            value: MsgRetireModelCard;
        };
        amendTokenizerSpec(value: MsgAmendTokenizerSpec): {
            typeUrl: string;
            value: MsgAmendTokenizerSpec;
        };
        attributeContributions(value: MsgAttributeContributions): {
            typeUrl: string;
            value: MsgAttributeContributions;
        };
        attestTraining(value: MsgAttestTraining): {
            typeUrl: string;
            value: MsgAttestTraining;
        };
        createAugmentationBounty(value: MsgCreateAugmentationBounty): {
            typeUrl: string;
            value: MsgCreateAugmentationBounty;
        };
        submitAugmentation(value: MsgSubmitAugmentation): {
            typeUrl: string;
            value: MsgSubmitAugmentation;
        };
        acceptAugmentation(value: MsgAcceptAugmentation): {
            typeUrl: string;
            value: MsgAcceptAugmentation;
        };
        voteOnAugmentation(value: MsgVoteOnAugmentation): {
            typeUrl: string;
            value: MsgVoteOnAugmentation;
        };
        sponsorVetoAugmentation(value: MsgSponsorVetoAugmentation): {
            typeUrl: string;
            value: MsgSponsorVetoAugmentation;
        };
        challengeContribution(value: MsgChallengeContribution): {
            typeUrl: string;
            value: MsgChallengeContribution;
        };
        resolveContributionChallenge(value: MsgResolveContributionChallenge): {
            typeUrl: string;
            value: MsgResolveContributionChallenge;
        };
        claimTrainingFundDisbursement(value: MsgClaimTrainingFundDisbursement): {
            typeUrl: string;
            value: MsgClaimTrainingFundDisbursement;
        };
        amendTraceSchema(value: MsgAmendTraceSchema): {
            typeUrl: string;
            value: MsgAmendTraceSchema;
        };
        createTrainingManifest(value: MsgCreateTrainingManifest): {
            typeUrl: string;
            value: MsgCreateTrainingManifest;
        };
        finalizeTrainingManifest(value: MsgFinalizeTrainingManifest): {
            typeUrl: string;
            value: MsgFinalizeTrainingManifest;
        };
        bindManifestToAttestation(value: MsgBindManifestToAttestation): {
            typeUrl: string;
            value: MsgBindManifestToAttestation;
        };
        openIncident(value: MsgOpenIncident): {
            typeUrl: string;
            value: MsgOpenIncident;
        };
        recordRemediation(value: MsgRecordRemediation): {
            typeUrl: string;
            value: MsgRecordRemediation;
        };
        resolveIncident(value: MsgResolveIncident): {
            typeUrl: string;
            value: MsgResolveIncident;
        };
        closeIncident(value: MsgCloseIncident): {
            typeUrl: string;
            value: MsgCloseIncident;
        };
        pauseModule(value: MsgPauseModule): {
            typeUrl: string;
            value: MsgPauseModule;
        };
        unpauseModule(value: MsgUnpauseModule): {
            typeUrl: string;
            value: MsgUnpauseModule;
        };
        correctManifestMerkleRoot(value: MsgCorrectManifestMerkleRoot): {
            typeUrl: string;
            value: MsgCorrectManifestMerkleRoot;
        };
        vetoFactInjection(value: MsgVetoFactInjection): {
            typeUrl: string;
            value: MsgVetoFactInjection;
        };
    };
    fromPartial: {
        submitClaim(value: MsgSubmitClaim): {
            typeUrl: string;
            value: MsgSubmitClaim;
        };
        submitCommitment(value: MsgSubmitCommitment): {
            typeUrl: string;
            value: MsgSubmitCommitment;
        };
        submitReveal(value: MsgSubmitReveal): {
            typeUrl: string;
            value: MsgSubmitReveal;
        };
        challengeFact(value: MsgChallengeFact): {
            typeUrl: string;
            value: MsgChallengeFact;
        };
        addFact(value: MsgAddFact): {
            typeUrl: string;
            value: MsgAddFact;
        };
        submitContradiction(value: MsgSubmitContradiction): {
            typeUrl: string;
            value: MsgSubmitContradiction;
        };
        patronizeFact(value: MsgPatronizeFact): {
            typeUrl: string;
            value: MsgPatronizeFact;
        };
        proposeDomain(value: MsgProposeDomain): {
            typeUrl: string;
            value: MsgProposeDomain;
        };
        endorseDomainProposal(value: MsgEndorseDomainProposal): {
            typeUrl: string;
            value: MsgEndorseDomainProposal;
        };
        challengeDomainProposal(value: MsgChallengeDomainProposal): {
            typeUrl: string;
            value: MsgChallengeDomainProposal;
        };
        registerStratum(value: MsgRegisterStratum): {
            typeUrl: string;
            value: MsgRegisterStratum;
        };
        challengeProvisionalFact(value: MsgChallengeProvisionalFact): {
            typeUrl: string;
            value: MsgChallengeProvisionalFact;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
        updateExtendedParams(value: MsgUpdateExtendedParams): {
            typeUrl: string;
            value: MsgUpdateExtendedParams;
        };
        proposeResearchFund(value: MsgProposeResearchFund): {
            typeUrl: string;
            value: MsgProposeResearchFund;
        };
        voteResearchProposal(value: MsgVoteResearchProposal): {
            typeUrl: string;
            value: MsgVoteResearchProposal;
        };
        executeResearchProposal(value: MsgExecuteResearchProposal): {
            typeUrl: string;
            value: MsgExecuteResearchProposal;
        };
        addCommonKnowledge(value: MsgAddCommonKnowledge): {
            typeUrl: string;
            value: MsgAddCommonKnowledge;
        };
        removeCommonKnowledge(value: MsgRemoveCommonKnowledge): {
            typeUrl: string;
            value: MsgRemoveCommonKnowledge;
        };
        reportDemand(value: MsgReportDemand): {
            typeUrl: string;
            value: MsgReportDemand;
        };
        rateFact(value: MsgRateFact): {
            typeUrl: string;
            value: MsgRateFact;
        };
        registerTrainingPipeline(value: MsgRegisterTrainingPipeline): {
            typeUrl: string;
            value: MsgRegisterTrainingPipeline;
        };
        updateTrainingPipeline(value: MsgUpdateTrainingPipeline): {
            typeUrl: string;
            value: MsgUpdateTrainingPipeline;
        };
        registerModelCard(value: MsgRegisterModelCard): {
            typeUrl: string;
            value: MsgRegisterModelCard;
        };
        updateModelCard(value: MsgUpdateModelCard): {
            typeUrl: string;
            value: MsgUpdateModelCard;
        };
        retireModelCard(value: MsgRetireModelCard): {
            typeUrl: string;
            value: MsgRetireModelCard;
        };
        amendTokenizerSpec(value: MsgAmendTokenizerSpec): {
            typeUrl: string;
            value: MsgAmendTokenizerSpec;
        };
        attributeContributions(value: MsgAttributeContributions): {
            typeUrl: string;
            value: MsgAttributeContributions;
        };
        attestTraining(value: MsgAttestTraining): {
            typeUrl: string;
            value: MsgAttestTraining;
        };
        createAugmentationBounty(value: MsgCreateAugmentationBounty): {
            typeUrl: string;
            value: MsgCreateAugmentationBounty;
        };
        submitAugmentation(value: MsgSubmitAugmentation): {
            typeUrl: string;
            value: MsgSubmitAugmentation;
        };
        acceptAugmentation(value: MsgAcceptAugmentation): {
            typeUrl: string;
            value: MsgAcceptAugmentation;
        };
        voteOnAugmentation(value: MsgVoteOnAugmentation): {
            typeUrl: string;
            value: MsgVoteOnAugmentation;
        };
        sponsorVetoAugmentation(value: MsgSponsorVetoAugmentation): {
            typeUrl: string;
            value: MsgSponsorVetoAugmentation;
        };
        challengeContribution(value: MsgChallengeContribution): {
            typeUrl: string;
            value: MsgChallengeContribution;
        };
        resolveContributionChallenge(value: MsgResolveContributionChallenge): {
            typeUrl: string;
            value: MsgResolveContributionChallenge;
        };
        claimTrainingFundDisbursement(value: MsgClaimTrainingFundDisbursement): {
            typeUrl: string;
            value: MsgClaimTrainingFundDisbursement;
        };
        amendTraceSchema(value: MsgAmendTraceSchema): {
            typeUrl: string;
            value: MsgAmendTraceSchema;
        };
        createTrainingManifest(value: MsgCreateTrainingManifest): {
            typeUrl: string;
            value: MsgCreateTrainingManifest;
        };
        finalizeTrainingManifest(value: MsgFinalizeTrainingManifest): {
            typeUrl: string;
            value: MsgFinalizeTrainingManifest;
        };
        bindManifestToAttestation(value: MsgBindManifestToAttestation): {
            typeUrl: string;
            value: MsgBindManifestToAttestation;
        };
        openIncident(value: MsgOpenIncident): {
            typeUrl: string;
            value: MsgOpenIncident;
        };
        recordRemediation(value: MsgRecordRemediation): {
            typeUrl: string;
            value: MsgRecordRemediation;
        };
        resolveIncident(value: MsgResolveIncident): {
            typeUrl: string;
            value: MsgResolveIncident;
        };
        closeIncident(value: MsgCloseIncident): {
            typeUrl: string;
            value: MsgCloseIncident;
        };
        pauseModule(value: MsgPauseModule): {
            typeUrl: string;
            value: MsgPauseModule;
        };
        unpauseModule(value: MsgUnpauseModule): {
            typeUrl: string;
            value: MsgUnpauseModule;
        };
        correctManifestMerkleRoot(value: MsgCorrectManifestMerkleRoot): {
            typeUrl: string;
            value: MsgCorrectManifestMerkleRoot;
        };
        vetoFactInjection(value: MsgVetoFactInjection): {
            typeUrl: string;
            value: MsgVetoFactInjection;
        };
    };
};
