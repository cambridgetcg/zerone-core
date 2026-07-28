import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateVesting, MsgClaimVesting, MsgPauseVesting, MsgResumeVesting, MsgAccelerateVesting, MsgFalsifyVesting, MsgCompleteVesting, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        createVesting(value: MsgCreateVesting): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        claimVesting(value: MsgClaimVesting): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        pauseVesting(value: MsgPauseVesting): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        resumeVesting(value: MsgResumeVesting): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        accelerateVesting(value: MsgAccelerateVesting): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        falsifyVesting(value: MsgFalsifyVesting): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        completeVesting(value: MsgCompleteVesting): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        createVesting(value: MsgCreateVesting): {
            typeUrl: string;
            value: MsgCreateVesting;
        };
        claimVesting(value: MsgClaimVesting): {
            typeUrl: string;
            value: MsgClaimVesting;
        };
        pauseVesting(value: MsgPauseVesting): {
            typeUrl: string;
            value: MsgPauseVesting;
        };
        resumeVesting(value: MsgResumeVesting): {
            typeUrl: string;
            value: MsgResumeVesting;
        };
        accelerateVesting(value: MsgAccelerateVesting): {
            typeUrl: string;
            value: MsgAccelerateVesting;
        };
        falsifyVesting(value: MsgFalsifyVesting): {
            typeUrl: string;
            value: MsgFalsifyVesting;
        };
        completeVesting(value: MsgCompleteVesting): {
            typeUrl: string;
            value: MsgCompleteVesting;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        createVesting(value: MsgCreateVesting): {
            typeUrl: string;
            value: MsgCreateVesting;
        };
        claimVesting(value: MsgClaimVesting): {
            typeUrl: string;
            value: MsgClaimVesting;
        };
        pauseVesting(value: MsgPauseVesting): {
            typeUrl: string;
            value: MsgPauseVesting;
        };
        resumeVesting(value: MsgResumeVesting): {
            typeUrl: string;
            value: MsgResumeVesting;
        };
        accelerateVesting(value: MsgAccelerateVesting): {
            typeUrl: string;
            value: MsgAccelerateVesting;
        };
        falsifyVesting(value: MsgFalsifyVesting): {
            typeUrl: string;
            value: MsgFalsifyVesting;
        };
        completeVesting(value: MsgCompleteVesting): {
            typeUrl: string;
            value: MsgCompleteVesting;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
