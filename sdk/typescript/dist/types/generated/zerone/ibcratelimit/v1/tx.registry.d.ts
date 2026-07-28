import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgAddRateLimit, MsgRemoveRateLimit, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        addRateLimit(value: MsgAddRateLimit): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        removeRateLimit(value: MsgRemoveRateLimit): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        addRateLimit(value: MsgAddRateLimit): {
            typeUrl: string;
            value: MsgAddRateLimit;
        };
        removeRateLimit(value: MsgRemoveRateLimit): {
            typeUrl: string;
            value: MsgRemoveRateLimit;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        addRateLimit(value: MsgAddRateLimit): {
            typeUrl: string;
            value: MsgAddRateLimit;
        };
        removeRateLimit(value: MsgRemoveRateLimit): {
            typeUrl: string;
            value: MsgRemoveRateLimit;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
