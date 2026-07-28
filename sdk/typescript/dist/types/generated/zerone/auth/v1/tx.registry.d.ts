import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgRegisterAccount, MsgRotateKey, MsgFreezeAccount, MsgUnfreezeAccount, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        registerAccount(value: MsgRegisterAccount): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        rotateKey(value: MsgRotateKey): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        freezeAccount(value: MsgFreezeAccount): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        unfreezeAccount(value: MsgUnfreezeAccount): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        registerAccount(value: MsgRegisterAccount): {
            typeUrl: string;
            value: MsgRegisterAccount;
        };
        rotateKey(value: MsgRotateKey): {
            typeUrl: string;
            value: MsgRotateKey;
        };
        freezeAccount(value: MsgFreezeAccount): {
            typeUrl: string;
            value: MsgFreezeAccount;
        };
        unfreezeAccount(value: MsgUnfreezeAccount): {
            typeUrl: string;
            value: MsgUnfreezeAccount;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        registerAccount(value: MsgRegisterAccount): {
            typeUrl: string;
            value: MsgRegisterAccount;
        };
        rotateKey(value: MsgRotateKey): {
            typeUrl: string;
            value: MsgRotateKey;
        };
        freezeAccount(value: MsgFreezeAccount): {
            typeUrl: string;
            value: MsgFreezeAccount;
        };
        unfreezeAccount(value: MsgUnfreezeAccount): {
            typeUrl: string;
            value: MsgUnfreezeAccount;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
