import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgQualifyByStake, MsgQualifyByTrackRecord, MsgQualifyByCrossReference, MsgQualifyByInheritance, MsgEndorseQualification, MsgRenewQualification, MsgWithdrawQualification, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        qualifyByStake(value: MsgQualifyByStake): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        qualifyByTrackRecord(value: MsgQualifyByTrackRecord): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        qualifyByCrossReference(value: MsgQualifyByCrossReference): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        qualifyByInheritance(value: MsgQualifyByInheritance): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        endorseQualification(value: MsgEndorseQualification): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        renewQualification(value: MsgRenewQualification): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        withdrawQualification(value: MsgWithdrawQualification): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        qualifyByStake(value: MsgQualifyByStake): {
            typeUrl: string;
            value: MsgQualifyByStake;
        };
        qualifyByTrackRecord(value: MsgQualifyByTrackRecord): {
            typeUrl: string;
            value: MsgQualifyByTrackRecord;
        };
        qualifyByCrossReference(value: MsgQualifyByCrossReference): {
            typeUrl: string;
            value: MsgQualifyByCrossReference;
        };
        qualifyByInheritance(value: MsgQualifyByInheritance): {
            typeUrl: string;
            value: MsgQualifyByInheritance;
        };
        endorseQualification(value: MsgEndorseQualification): {
            typeUrl: string;
            value: MsgEndorseQualification;
        };
        renewQualification(value: MsgRenewQualification): {
            typeUrl: string;
            value: MsgRenewQualification;
        };
        withdrawQualification(value: MsgWithdrawQualification): {
            typeUrl: string;
            value: MsgWithdrawQualification;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        qualifyByStake(value: MsgQualifyByStake): {
            typeUrl: string;
            value: MsgQualifyByStake;
        };
        qualifyByTrackRecord(value: MsgQualifyByTrackRecord): {
            typeUrl: string;
            value: MsgQualifyByTrackRecord;
        };
        qualifyByCrossReference(value: MsgQualifyByCrossReference): {
            typeUrl: string;
            value: MsgQualifyByCrossReference;
        };
        qualifyByInheritance(value: MsgQualifyByInheritance): {
            typeUrl: string;
            value: MsgQualifyByInheritance;
        };
        endorseQualification(value: MsgEndorseQualification): {
            typeUrl: string;
            value: MsgEndorseQualification;
        };
        renewQualification(value: MsgRenewQualification): {
            typeUrl: string;
            value: MsgRenewQualification;
        };
        withdrawQualification(value: MsgWithdrawQualification): {
            typeUrl: string;
            value: MsgWithdrawQualification;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
