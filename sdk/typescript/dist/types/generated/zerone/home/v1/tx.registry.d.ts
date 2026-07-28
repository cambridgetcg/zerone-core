import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateHome, MsgUpdateHome, MsgUpdateMemoryCID, MsgStartSession, MsgEndSession, MsgRegisterKey, MsgRevokeKey, MsgConfigureGuardian, MsgAcknowledgeAlert, MsgSetSpendingLimit, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        createHome(value: MsgCreateHome): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateHome(value: MsgUpdateHome): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateMemoryCID(value: MsgUpdateMemoryCID): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        startSession(value: MsgStartSession): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        endSession(value: MsgEndSession): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        registerKey(value: MsgRegisterKey): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        revokeKey(value: MsgRevokeKey): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        configureGuardian(value: MsgConfigureGuardian): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        acknowledgeAlert(value: MsgAcknowledgeAlert): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        setSpendingLimit(value: MsgSetSpendingLimit): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        createHome(value: MsgCreateHome): {
            typeUrl: string;
            value: MsgCreateHome;
        };
        updateHome(value: MsgUpdateHome): {
            typeUrl: string;
            value: MsgUpdateHome;
        };
        updateMemoryCID(value: MsgUpdateMemoryCID): {
            typeUrl: string;
            value: MsgUpdateMemoryCID;
        };
        startSession(value: MsgStartSession): {
            typeUrl: string;
            value: MsgStartSession;
        };
        endSession(value: MsgEndSession): {
            typeUrl: string;
            value: MsgEndSession;
        };
        registerKey(value: MsgRegisterKey): {
            typeUrl: string;
            value: MsgRegisterKey;
        };
        revokeKey(value: MsgRevokeKey): {
            typeUrl: string;
            value: MsgRevokeKey;
        };
        configureGuardian(value: MsgConfigureGuardian): {
            typeUrl: string;
            value: MsgConfigureGuardian;
        };
        acknowledgeAlert(value: MsgAcknowledgeAlert): {
            typeUrl: string;
            value: MsgAcknowledgeAlert;
        };
        setSpendingLimit(value: MsgSetSpendingLimit): {
            typeUrl: string;
            value: MsgSetSpendingLimit;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        createHome(value: MsgCreateHome): {
            typeUrl: string;
            value: MsgCreateHome;
        };
        updateHome(value: MsgUpdateHome): {
            typeUrl: string;
            value: MsgUpdateHome;
        };
        updateMemoryCID(value: MsgUpdateMemoryCID): {
            typeUrl: string;
            value: MsgUpdateMemoryCID;
        };
        startSession(value: MsgStartSession): {
            typeUrl: string;
            value: MsgStartSession;
        };
        endSession(value: MsgEndSession): {
            typeUrl: string;
            value: MsgEndSession;
        };
        registerKey(value: MsgRegisterKey): {
            typeUrl: string;
            value: MsgRegisterKey;
        };
        revokeKey(value: MsgRevokeKey): {
            typeUrl: string;
            value: MsgRevokeKey;
        };
        configureGuardian(value: MsgConfigureGuardian): {
            typeUrl: string;
            value: MsgConfigureGuardian;
        };
        acknowledgeAlert(value: MsgAcknowledgeAlert): {
            typeUrl: string;
            value: MsgAcknowledgeAlert;
        };
        setSpendingLimit(value: MsgSetSpendingLimit): {
            typeUrl: string;
            value: MsgSetSpendingLimit;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
