import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/** BountyStatus enumerates the lifecycle states of a BountyOrder. */
export declare enum BountyStatus {
    BOUNTY_STATUS_UNSPECIFIED = 0,
    /** BOUNTY_STATUS_ACTIVE - accepting fulfillments */
    BOUNTY_STATUS_ACTIVE = 1,
    /** BOUNTY_STATUS_FULFILLED - target_count reached */
    BOUNTY_STATUS_FULFILLED = 2,
    /** BOUNTY_STATUS_EXPIRED - currentBlock >= end_block, set by BeginBlocker */
    BOUNTY_STATUS_EXPIRED = 3,
    /** BOUNTY_STATUS_CANCELED - sponsor reclaimed remaining */
    BOUNTY_STATUS_CANCELED = 4,
    UNRECOGNIZED = -1
}
export declare function bountyStatusFromJSON(object: any): BountyStatus;
export declare function bountyStatusToJSON(object: BountyStatus): string;
/**
 * BountyOrder is a sponsor's commitment of escrowed funds against a
 * typed bounty for verified work in a specific domain.
 * @name BountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.BountyOrder
 */
export interface BountyOrder {
    id: string;
    sponsor: string;
    domain: string;
    pricePerArtifact: string;
    targetCount: number;
    fulfilledCount: number;
    escrowRemaining: string;
    startBlock: bigint;
    endBlock: bigint;
    status: BountyStatus;
}
/**
 * BountyFulfillment records a single payout from a bounty to a worker
 * (the submitter of the fulfilling fact).
 * @name BountyFulfillment
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.BountyFulfillment
 */
export interface BountyFulfillment {
    bountyId: string;
    factId: string;
    worker: string;
    amountPaid: string;
    fulfilledAtBlock: bigint;
}
/**
 * Params is the governance-tunable configuration.
 * @name Params
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.Params
 */
export interface Params {
    minTargetCount: number;
    minDurationBlocks: bigint;
    maxActiveBountiesPerSponsor: number;
}
/**
 * BountyOrder is a sponsor's commitment of escrowed funds against a
 * typed bounty for verified work in a specific domain.
 * @name BountyOrder
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.BountyOrder
 */
export declare const BountyOrder: {
    typeUrl: string;
    encode(message: BountyOrder, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): BountyOrder;
    fromPartial(object: DeepPartial<BountyOrder>): BountyOrder;
};
/**
 * BountyFulfillment records a single payout from a bounty to a worker
 * (the submitter of the fulfilling fact).
 * @name BountyFulfillment
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.BountyFulfillment
 */
export declare const BountyFulfillment: {
    typeUrl: string;
    encode(message: BountyFulfillment, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): BountyFulfillment;
    fromPartial(object: DeepPartial<BountyFulfillment>): BountyFulfillment;
};
/**
 * Params is the governance-tunable configuration.
 * @name Params
 * @package zerone.sponsorship.v1
 * @see proto type: zerone.sponsorship.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
