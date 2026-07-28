import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * RateLimit defines a per-(channel, denom) flow cap with a sliding window.
 * @name RateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.RateLimit
 */
export interface RateLimit {
    channelId: string;
    denom: string;
    /**
     * bigint string (uzrn)
     */
    maxSend: string;
    /**
     * bigint string (uzrn)
     */
    maxRecv: string;
    windowBlocks: bigint;
    /**
     * bigint string — accumulated in current window
     */
    currentSend: string;
    /**
     * bigint string — accumulated in current window
     */
    currentRecv: string;
    /**
     * block height at which the current window began
     */
    windowStart: bigint;
}
/**
 * PacketFlow records an outbound packet for quota reversal on timeout/error ack.
 * @name PacketFlow
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.PacketFlow
 */
export interface PacketFlow {
    channelId: string;
    sequence: bigint;
    denom: string;
    /**
     * bigint string
     */
    amount: string;
}
/**
 * RateLimit defines a per-(channel, denom) flow cap with a sliding window.
 * @name RateLimit
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.RateLimit
 */
export declare const RateLimit: {
    typeUrl: string;
    encode(message: RateLimit, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): RateLimit;
    fromPartial(object: DeepPartial<RateLimit>): RateLimit;
};
/**
 * PacketFlow records an outbound packet for quota reversal on timeout/error ack.
 * @name PacketFlow
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.PacketFlow
 */
export declare const PacketFlow: {
    typeUrl: string;
    encode(message: PacketFlow, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): PacketFlow;
    fromPartial(object: DeepPartial<PacketFlow>): PacketFlow;
};
