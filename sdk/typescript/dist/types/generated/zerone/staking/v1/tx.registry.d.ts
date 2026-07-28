import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgRegisterValidator, MsgDelegate, MsgUndelegate, MsgRedelegate, MsgUpdateValidatorStake, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        registerValidator(value: MsgRegisterValidator): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        delegate(value: MsgDelegate): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        undelegate(value: MsgUndelegate): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        redelegate(value: MsgRedelegate): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateValidatorStake(value: MsgUpdateValidatorStake): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        registerValidator(value: MsgRegisterValidator): {
            typeUrl: string;
            value: MsgRegisterValidator;
        };
        delegate(value: MsgDelegate): {
            typeUrl: string;
            value: MsgDelegate;
        };
        undelegate(value: MsgUndelegate): {
            typeUrl: string;
            value: MsgUndelegate;
        };
        redelegate(value: MsgRedelegate): {
            typeUrl: string;
            value: MsgRedelegate;
        };
        updateValidatorStake(value: MsgUpdateValidatorStake): {
            typeUrl: string;
            value: MsgUpdateValidatorStake;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        registerValidator(value: MsgRegisterValidator): {
            typeUrl: string;
            value: MsgRegisterValidator;
        };
        delegate(value: MsgDelegate): {
            typeUrl: string;
            value: MsgDelegate;
        };
        undelegate(value: MsgUndelegate): {
            typeUrl: string;
            value: MsgUndelegate;
        };
        redelegate(value: MsgRedelegate): {
            typeUrl: string;
            value: MsgRedelegate;
        };
        updateValidatorStake(value: MsgUpdateValidatorStake): {
            typeUrl: string;
            value: MsgUpdateValidatorStake;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
