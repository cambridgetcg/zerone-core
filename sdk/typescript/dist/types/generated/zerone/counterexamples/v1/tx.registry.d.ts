import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgProposeCounterexample, MsgValidate, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        proposeCounterexample(value: MsgProposeCounterexample): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        validate(value: MsgValidate): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        proposeCounterexample(value: MsgProposeCounterexample): {
            typeUrl: string;
            value: MsgProposeCounterexample;
        };
        validate(value: MsgValidate): {
            typeUrl: string;
            value: MsgValidate;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        proposeCounterexample(value: MsgProposeCounterexample): {
            typeUrl: string;
            value: MsgProposeCounterexample;
        };
        validate(value: MsgValidate): {
            typeUrl: string;
            value: MsgValidate;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
