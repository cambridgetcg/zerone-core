import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgProposeDomain, MsgVoteDomainProposal, MsgUpdateDomain, MsgRegisterLogicZone, MsgAcknowledgeIncompleteness, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        proposeDomain(value: MsgProposeDomain): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        voteDomainProposal(value: MsgVoteDomainProposal): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateDomain(value: MsgUpdateDomain): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        registerLogicZone(value: MsgRegisterLogicZone): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        acknowledgeIncompleteness(value: MsgAcknowledgeIncompleteness): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        proposeDomain(value: MsgProposeDomain): {
            typeUrl: string;
            value: MsgProposeDomain;
        };
        voteDomainProposal(value: MsgVoteDomainProposal): {
            typeUrl: string;
            value: MsgVoteDomainProposal;
        };
        updateDomain(value: MsgUpdateDomain): {
            typeUrl: string;
            value: MsgUpdateDomain;
        };
        registerLogicZone(value: MsgRegisterLogicZone): {
            typeUrl: string;
            value: MsgRegisterLogicZone;
        };
        acknowledgeIncompleteness(value: MsgAcknowledgeIncompleteness): {
            typeUrl: string;
            value: MsgAcknowledgeIncompleteness;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        proposeDomain(value: MsgProposeDomain): {
            typeUrl: string;
            value: MsgProposeDomain;
        };
        voteDomainProposal(value: MsgVoteDomainProposal): {
            typeUrl: string;
            value: MsgVoteDomainProposal;
        };
        updateDomain(value: MsgUpdateDomain): {
            typeUrl: string;
            value: MsgUpdateDomain;
        };
        registerLogicZone(value: MsgRegisterLogicZone): {
            typeUrl: string;
            value: MsgRegisterLogicZone;
        };
        acknowledgeIncompleteness(value: MsgAcknowledgeIncompleteness): {
            typeUrl: string;
            value: MsgAcknowledgeIncompleteness;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
