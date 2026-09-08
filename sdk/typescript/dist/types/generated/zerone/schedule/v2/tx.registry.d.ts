import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateSchedule, MsgUpdateSchedule, MsgCancelSchedule, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        createSchedule(value: MsgCreateSchedule): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateSchedule(value: MsgUpdateSchedule): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        cancelSchedule(value: MsgCancelSchedule): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        createSchedule(value: MsgCreateSchedule): {
            typeUrl: string;
            value: MsgCreateSchedule;
        };
        updateSchedule(value: MsgUpdateSchedule): {
            typeUrl: string;
            value: MsgUpdateSchedule;
        };
        cancelSchedule(value: MsgCancelSchedule): {
            typeUrl: string;
            value: MsgCancelSchedule;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        createSchedule(value: MsgCreateSchedule): {
            typeUrl: string;
            value: MsgCreateSchedule;
        };
        updateSchedule(value: MsgUpdateSchedule): {
            typeUrl: string;
            value: MsgUpdateSchedule;
        };
        cancelSchedule(value: MsgCancelSchedule): {
            typeUrl: string;
            value: MsgCancelSchedule;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
