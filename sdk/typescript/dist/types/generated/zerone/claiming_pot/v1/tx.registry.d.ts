import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreatePot, MsgClaim, MsgUpdatePotParams, MsgAddBootstrapEntry } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        createPot(value: MsgCreatePot): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        claim(value: MsgClaim): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updatePotParams(value: MsgUpdatePotParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        addBootstrapEntry(value: MsgAddBootstrapEntry): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        createPot(value: MsgCreatePot): {
            typeUrl: string;
            value: MsgCreatePot;
        };
        claim(value: MsgClaim): {
            typeUrl: string;
            value: MsgClaim;
        };
        updatePotParams(value: MsgUpdatePotParams): {
            typeUrl: string;
            value: MsgUpdatePotParams;
        };
        addBootstrapEntry(value: MsgAddBootstrapEntry): {
            typeUrl: string;
            value: MsgAddBootstrapEntry;
        };
    };
    fromPartial: {
        createPot(value: MsgCreatePot): {
            typeUrl: string;
            value: MsgCreatePot;
        };
        claim(value: MsgClaim): {
            typeUrl: string;
            value: MsgClaim;
        };
        updatePotParams(value: MsgUpdatePotParams): {
            typeUrl: string;
            value: MsgUpdatePotParams;
        };
        addBootstrapEntry(value: MsgAddBootstrapEntry): {
            typeUrl: string;
            value: MsgAddBootstrapEntry;
        };
    };
};
