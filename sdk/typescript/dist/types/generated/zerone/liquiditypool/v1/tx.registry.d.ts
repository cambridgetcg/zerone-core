import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreatePool, MsgSwap, MsgAddLiquidity, MsgRemoveLiquidity, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        createPool(value: MsgCreatePool): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        swap(value: MsgSwap): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        addLiquidity(value: MsgAddLiquidity): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        removeLiquidity(value: MsgRemoveLiquidity): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        createPool(value: MsgCreatePool): {
            typeUrl: string;
            value: MsgCreatePool;
        };
        swap(value: MsgSwap): {
            typeUrl: string;
            value: MsgSwap;
        };
        addLiquidity(value: MsgAddLiquidity): {
            typeUrl: string;
            value: MsgAddLiquidity;
        };
        removeLiquidity(value: MsgRemoveLiquidity): {
            typeUrl: string;
            value: MsgRemoveLiquidity;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        createPool(value: MsgCreatePool): {
            typeUrl: string;
            value: MsgCreatePool;
        };
        swap(value: MsgSwap): {
            typeUrl: string;
            value: MsgSwap;
        };
        addLiquidity(value: MsgAddLiquidity): {
            typeUrl: string;
            value: MsgAddLiquidity;
        };
        removeLiquidity(value: MsgRemoveLiquidity): {
            typeUrl: string;
            value: MsgRemoveLiquidity;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
