import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgRecordVerification, MsgAnalyzeDomain, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        recordVerification(value: MsgRecordVerification): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        analyzeDomain(value: MsgAnalyzeDomain): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        recordVerification(value: MsgRecordVerification): {
            typeUrl: string;
            value: MsgRecordVerification;
        };
        analyzeDomain(value: MsgAnalyzeDomain): {
            typeUrl: string;
            value: MsgAnalyzeDomain;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        recordVerification(value: MsgRecordVerification): {
            typeUrl: string;
            value: MsgRecordVerification;
        };
        analyzeDomain(value: MsgAnalyzeDomain): {
            typeUrl: string;
            value: MsgAnalyzeDomain;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
