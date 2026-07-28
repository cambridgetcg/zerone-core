import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateBountyOrder, MsgFulfillBounty, MsgCancelBountyOrder } from "./tx.js";
export declare const registry: ReadonlyArray<[string, GeneratedType]>;
export declare const load: (protoRegistry: Registry) => void;
export declare const MessageComposer: {
    encoded: {
        createBountyOrder(value: MsgCreateBountyOrder): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        fulfillBounty(value: MsgFulfillBounty): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
        cancelBountyOrder(value: MsgCancelBountyOrder): {
            typeUrl: string;
            value: Uint8Array<ArrayBufferLike>;
        };
    };
    withTypeUrl: {
        createBountyOrder(value: MsgCreateBountyOrder): {
            typeUrl: string;
            value: MsgCreateBountyOrder;
        };
        fulfillBounty(value: MsgFulfillBounty): {
            typeUrl: string;
            value: MsgFulfillBounty;
        };
        cancelBountyOrder(value: MsgCancelBountyOrder): {
            typeUrl: string;
            value: MsgCancelBountyOrder;
        };
    };
    fromPartial: {
        createBountyOrder(value: MsgCreateBountyOrder): {
            typeUrl: string;
            value: MsgCreateBountyOrder;
        };
        fulfillBounty(value: MsgFulfillBounty): {
            typeUrl: string;
            value: MsgFulfillBounty;
        };
        cancelBountyOrder(value: MsgCancelBountyOrder): {
            typeUrl: string;
            value: MsgCancelBountyOrder;
        };
    };
};
