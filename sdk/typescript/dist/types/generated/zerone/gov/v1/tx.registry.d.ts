import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgSubmitLIP, MsgStakeLIP, MsgAdvanceLIPStage, MsgCastVote, MsgWithdrawLIP, MsgUpdateParams, MsgSubmitResearchSpend, MsgVoteResearchSpend, MsgSetResearchVoters, MsgAttachUpgradePlan, MsgAttachCreedAmendmentPin, MsgNominateSeatElection, MsgAcceptSeatNomination, MsgVoteSeatElection, MsgDomainFormationFreeze } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        submitLIP(value: MsgSubmitLIP): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        stakeLIP(value: MsgStakeLIP): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        advanceLIPStage(value: MsgAdvanceLIPStage): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        castVote(value: MsgCastVote): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        withdrawLIP(value: MsgWithdrawLIP): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        submitResearchSpend(value: MsgSubmitResearchSpend): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        voteResearchSpend(value: MsgVoteResearchSpend): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        setResearchVoters(value: MsgSetResearchVoters): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        attachUpgradePlan(value: MsgAttachUpgradePlan): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        attachCreedAmendmentPin(value: MsgAttachCreedAmendmentPin): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        nominateSeatElection(value: MsgNominateSeatElection): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        acceptSeatNomination(value: MsgAcceptSeatNomination): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        voteSeatElection(value: MsgVoteSeatElection): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        domainFormationFreeze(value: MsgDomainFormationFreeze): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        submitLIP(value: MsgSubmitLIP): {
            typeUrl: string;
            value: MsgSubmitLIP;
        };
        stakeLIP(value: MsgStakeLIP): {
            typeUrl: string;
            value: MsgStakeLIP;
        };
        advanceLIPStage(value: MsgAdvanceLIPStage): {
            typeUrl: string;
            value: MsgAdvanceLIPStage;
        };
        castVote(value: MsgCastVote): {
            typeUrl: string;
            value: MsgCastVote;
        };
        withdrawLIP(value: MsgWithdrawLIP): {
            typeUrl: string;
            value: MsgWithdrawLIP;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
        submitResearchSpend(value: MsgSubmitResearchSpend): {
            typeUrl: string;
            value: MsgSubmitResearchSpend;
        };
        voteResearchSpend(value: MsgVoteResearchSpend): {
            typeUrl: string;
            value: MsgVoteResearchSpend;
        };
        setResearchVoters(value: MsgSetResearchVoters): {
            typeUrl: string;
            value: MsgSetResearchVoters;
        };
        attachUpgradePlan(value: MsgAttachUpgradePlan): {
            typeUrl: string;
            value: MsgAttachUpgradePlan;
        };
        attachCreedAmendmentPin(value: MsgAttachCreedAmendmentPin): {
            typeUrl: string;
            value: MsgAttachCreedAmendmentPin;
        };
        nominateSeatElection(value: MsgNominateSeatElection): {
            typeUrl: string;
            value: MsgNominateSeatElection;
        };
        acceptSeatNomination(value: MsgAcceptSeatNomination): {
            typeUrl: string;
            value: MsgAcceptSeatNomination;
        };
        voteSeatElection(value: MsgVoteSeatElection): {
            typeUrl: string;
            value: MsgVoteSeatElection;
        };
        domainFormationFreeze(value: MsgDomainFormationFreeze): {
            typeUrl: string;
            value: MsgDomainFormationFreeze;
        };
    };
    fromPartial: {
        submitLIP(value: MsgSubmitLIP): {
            typeUrl: string;
            value: MsgSubmitLIP;
        };
        stakeLIP(value: MsgStakeLIP): {
            typeUrl: string;
            value: MsgStakeLIP;
        };
        advanceLIPStage(value: MsgAdvanceLIPStage): {
            typeUrl: string;
            value: MsgAdvanceLIPStage;
        };
        castVote(value: MsgCastVote): {
            typeUrl: string;
            value: MsgCastVote;
        };
        withdrawLIP(value: MsgWithdrawLIP): {
            typeUrl: string;
            value: MsgWithdrawLIP;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
        submitResearchSpend(value: MsgSubmitResearchSpend): {
            typeUrl: string;
            value: MsgSubmitResearchSpend;
        };
        voteResearchSpend(value: MsgVoteResearchSpend): {
            typeUrl: string;
            value: MsgVoteResearchSpend;
        };
        setResearchVoters(value: MsgSetResearchVoters): {
            typeUrl: string;
            value: MsgSetResearchVoters;
        };
        attachUpgradePlan(value: MsgAttachUpgradePlan): {
            typeUrl: string;
            value: MsgAttachUpgradePlan;
        };
        attachCreedAmendmentPin(value: MsgAttachCreedAmendmentPin): {
            typeUrl: string;
            value: MsgAttachCreedAmendmentPin;
        };
        nominateSeatElection(value: MsgNominateSeatElection): {
            typeUrl: string;
            value: MsgNominateSeatElection;
        };
        acceptSeatNomination(value: MsgAcceptSeatNomination): {
            typeUrl: string;
            value: MsgAcceptSeatNomination;
        };
        voteSeatElection(value: MsgVoteSeatElection): {
            typeUrl: string;
            value: MsgVoteSeatElection;
        };
        domainFormationFreeze(value: MsgDomainFormationFreeze): {
            typeUrl: string;
            value: MsgDomainFormationFreeze;
        };
    };
};
