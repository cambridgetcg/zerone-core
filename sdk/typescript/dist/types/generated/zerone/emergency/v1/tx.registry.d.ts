import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgProposeHalt, MsgVoteHalt, MsgProposeRevert, MsgVoteRevert, MsgProposeResume, MsgVoteResume, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        proposeHalt(value: MsgProposeHalt): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        voteHalt(value: MsgVoteHalt): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        proposeRevert(value: MsgProposeRevert): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        voteRevert(value: MsgVoteRevert): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        proposeResume(value: MsgProposeResume): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        voteResume(value: MsgVoteResume): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        proposeHalt(value: MsgProposeHalt): {
            typeUrl: string;
            value: MsgProposeHalt;
        };
        voteHalt(value: MsgVoteHalt): {
            typeUrl: string;
            value: MsgVoteHalt;
        };
        proposeRevert(value: MsgProposeRevert): {
            typeUrl: string;
            value: MsgProposeRevert;
        };
        voteRevert(value: MsgVoteRevert): {
            typeUrl: string;
            value: MsgVoteRevert;
        };
        proposeResume(value: MsgProposeResume): {
            typeUrl: string;
            value: MsgProposeResume;
        };
        voteResume(value: MsgVoteResume): {
            typeUrl: string;
            value: MsgVoteResume;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        proposeHalt(value: MsgProposeHalt): {
            typeUrl: string;
            value: MsgProposeHalt;
        };
        voteHalt(value: MsgVoteHalt): {
            typeUrl: string;
            value: MsgVoteHalt;
        };
        proposeRevert(value: MsgProposeRevert): {
            typeUrl: string;
            value: MsgProposeRevert;
        };
        voteRevert(value: MsgVoteRevert): {
            typeUrl: string;
            value: MsgVoteRevert;
        };
        proposeResume(value: MsgProposeResume): {
            typeUrl: string;
            value: MsgProposeResume;
        };
        voteResume(value: MsgVoteResume): {
            typeUrl: string;
            value: MsgVoteResume;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
