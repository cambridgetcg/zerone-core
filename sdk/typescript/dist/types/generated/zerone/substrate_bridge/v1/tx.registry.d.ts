import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgRegisterAdapter, MsgSuspendAdapter, MsgTombstoneAdapter, MsgSubmitExternalAttestation } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        registerAdapter(value: MsgRegisterAdapter): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        suspendAdapter(value: MsgSuspendAdapter): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        tombstoneAdapter(value: MsgTombstoneAdapter): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        submitExternalAttestation(value: MsgSubmitExternalAttestation): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        registerAdapter(value: MsgRegisterAdapter): {
            typeUrl: string;
            value: MsgRegisterAdapter;
        };
        suspendAdapter(value: MsgSuspendAdapter): {
            typeUrl: string;
            value: MsgSuspendAdapter;
        };
        tombstoneAdapter(value: MsgTombstoneAdapter): {
            typeUrl: string;
            value: MsgTombstoneAdapter;
        };
        submitExternalAttestation(value: MsgSubmitExternalAttestation): {
            typeUrl: string;
            value: MsgSubmitExternalAttestation;
        };
    };
    fromPartial: {
        registerAdapter(value: MsgRegisterAdapter): {
            typeUrl: string;
            value: MsgRegisterAdapter;
        };
        suspendAdapter(value: MsgSuspendAdapter): {
            typeUrl: string;
            value: MsgSuspendAdapter;
        };
        tombstoneAdapter(value: MsgTombstoneAdapter): {
            typeUrl: string;
            value: MsgTombstoneAdapter;
        };
        submitExternalAttestation(value: MsgSubmitExternalAttestation): {
            typeUrl: string;
            value: MsgSubmitExternalAttestation;
        };
    };
};
