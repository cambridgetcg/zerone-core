import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgAnchorPin, MsgUpdateParams, MsgUpdateCouncilMember } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        anchorPin(value: MsgAnchorPin): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateCouncilMember(value: MsgUpdateCouncilMember): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        anchorPin(value: MsgAnchorPin): {
            typeUrl: string;
            value: MsgAnchorPin;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
        updateCouncilMember(value: MsgUpdateCouncilMember): {
            typeUrl: string;
            value: MsgUpdateCouncilMember;
        };
    };
    fromPartial: {
        anchorPin(value: MsgAnchorPin): {
            typeUrl: string;
            value: MsgAnchorPin;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
        updateCouncilMember(value: MsgUpdateCouncilMember): {
            typeUrl: string;
            value: MsgUpdateCouncilMember;
        };
    };
};
