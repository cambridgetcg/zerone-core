import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgSubmitChallenge, MsgAddEvidence, MsgResolveChallenge, MsgFundBountyPool, MsgUpdateParams } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        submitChallenge(value: MsgSubmitChallenge): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        addEvidence(value: MsgAddEvidence): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        resolveChallenge(value: MsgResolveChallenge): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        fundBountyPool(value: MsgFundBountyPool): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        submitChallenge(value: MsgSubmitChallenge): {
            typeUrl: string;
            value: MsgSubmitChallenge;
        };
        addEvidence(value: MsgAddEvidence): {
            typeUrl: string;
            value: MsgAddEvidence;
        };
        resolveChallenge(value: MsgResolveChallenge): {
            typeUrl: string;
            value: MsgResolveChallenge;
        };
        fundBountyPool(value: MsgFundBountyPool): {
            typeUrl: string;
            value: MsgFundBountyPool;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
    fromPartial: {
        submitChallenge(value: MsgSubmitChallenge): {
            typeUrl: string;
            value: MsgSubmitChallenge;
        };
        addEvidence(value: MsgAddEvidence): {
            typeUrl: string;
            value: MsgAddEvidence;
        };
        resolveChallenge(value: MsgResolveChallenge): {
            typeUrl: string;
            value: MsgResolveChallenge;
        };
        fundBountyPool(value: MsgFundBountyPool): {
            typeUrl: string;
            value: MsgFundBountyPool;
        };
        updateParams(value: MsgUpdateParams): {
            typeUrl: string;
            value: MsgUpdateParams;
        };
    };
};
