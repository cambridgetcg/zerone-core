import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateToken, MsgMintToken, MsgBurnToken, MsgTransferToken, MsgApproveToken, MsgTransferFrom, MsgPauseToken, MsgUnpauseToken, MsgDelegatePower, MsgUndelegatePower, MsgWrapToken, MsgUnwrapToken, MsgCreateEmissionPeriod, MsgCancelEmissionPeriod, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        createToken(value: MsgCreateToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        mintToken(value: MsgMintToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        burnToken(value: MsgBurnToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        transferToken(value: MsgTransferToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        approveToken(value: MsgApproveToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        transferFrom(value: MsgTransferFrom): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        pauseToken(value: MsgPauseToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        unpauseToken(value: MsgUnpauseToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        delegatePower(value: MsgDelegatePower): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        undelegatePower(value: MsgUndelegatePower): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        wrapToken(value: MsgWrapToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        unwrapToken(value: MsgUnwrapToken): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        createEmissionPeriod(value: MsgCreateEmissionPeriod): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        cancelEmissionPeriod(value: MsgCancelEmissionPeriod): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        createToken(value: MsgCreateToken): {
            typeUrl: string;
            value: MsgCreateToken;
        };
        mintToken(value: MsgMintToken): {
            typeUrl: string;
            value: MsgMintToken;
        };
        burnToken(value: MsgBurnToken): {
            typeUrl: string;
            value: MsgBurnToken;
        };
        transferToken(value: MsgTransferToken): {
            typeUrl: string;
            value: MsgTransferToken;
        };
        approveToken(value: MsgApproveToken): {
            typeUrl: string;
            value: MsgApproveToken;
        };
        transferFrom(value: MsgTransferFrom): {
            typeUrl: string;
            value: MsgTransferFrom;
        };
        pauseToken(value: MsgPauseToken): {
            typeUrl: string;
            value: MsgPauseToken;
        };
        unpauseToken(value: MsgUnpauseToken): {
            typeUrl: string;
            value: MsgUnpauseToken;
        };
        delegatePower(value: MsgDelegatePower): {
            typeUrl: string;
            value: MsgDelegatePower;
        };
        undelegatePower(value: MsgUndelegatePower): {
            typeUrl: string;
            value: MsgUndelegatePower;
        };
        wrapToken(value: MsgWrapToken): {
            typeUrl: string;
            value: MsgWrapToken;
        };
        unwrapToken(value: MsgUnwrapToken): {
            typeUrl: string;
            value: MsgUnwrapToken;
        };
        createEmissionPeriod(value: MsgCreateEmissionPeriod): {
            typeUrl: string;
            value: MsgCreateEmissionPeriod;
        };
        cancelEmissionPeriod(value: MsgCancelEmissionPeriod): {
            typeUrl: string;
            value: MsgCancelEmissionPeriod;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        createToken(value: MsgCreateToken): {
            typeUrl: string;
            value: MsgCreateToken;
        };
        mintToken(value: MsgMintToken): {
            typeUrl: string;
            value: MsgMintToken;
        };
        burnToken(value: MsgBurnToken): {
            typeUrl: string;
            value: MsgBurnToken;
        };
        transferToken(value: MsgTransferToken): {
            typeUrl: string;
            value: MsgTransferToken;
        };
        approveToken(value: MsgApproveToken): {
            typeUrl: string;
            value: MsgApproveToken;
        };
        transferFrom(value: MsgTransferFrom): {
            typeUrl: string;
            value: MsgTransferFrom;
        };
        pauseToken(value: MsgPauseToken): {
            typeUrl: string;
            value: MsgPauseToken;
        };
        unpauseToken(value: MsgUnpauseToken): {
            typeUrl: string;
            value: MsgUnpauseToken;
        };
        delegatePower(value: MsgDelegatePower): {
            typeUrl: string;
            value: MsgDelegatePower;
        };
        undelegatePower(value: MsgUndelegatePower): {
            typeUrl: string;
            value: MsgUndelegatePower;
        };
        wrapToken(value: MsgWrapToken): {
            typeUrl: string;
            value: MsgWrapToken;
        };
        unwrapToken(value: MsgUnwrapToken): {
            typeUrl: string;
            value: MsgUnwrapToken;
        };
        createEmissionPeriod(value: MsgCreateEmissionPeriod): {
            typeUrl: string;
            value: MsgCreateEmissionPeriod;
        };
        cancelEmissionPeriod(value: MsgCancelEmissionPeriod): {
            typeUrl: string;
            value: MsgCancelEmissionPeriod;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
