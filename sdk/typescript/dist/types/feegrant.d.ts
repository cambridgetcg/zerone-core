import type { EncodeObject } from "@cosmjs/proto-signing";
import { type ZeroneNetwork } from "./caip.js";
export type FeeGrantErrorCode = "SELF_GRANT" | "EMPTY_SPEND_LIMIT" | "INVALID_COIN" | "DUPLICATE_DENOM" | "INVALID_EXPIRATION" | "EXPIRED_ALLOWANCE" | "EMPTY_ALLOWED_MESSAGES" | "INVALID_MESSAGE_TYPE_URL" | "DUPLICATE_MESSAGE_TYPE_URL" | "UNAPPROVED_MESSAGE_TYPE_URL" | "INVALID_GAS";
export declare class FeeGrantError extends Error {
    readonly code: FeeGrantErrorCode;
    constructor(code: FeeGrantErrorCode, message: string);
}
export interface FeeGrantCoin {
    readonly denom: string;
    readonly amount: string;
}
export interface FeeGrantParties {
    readonly network: ZeroneNetwork;
    readonly granter: string;
    readonly grantee: string;
}
export interface BoundedFeeGrantInput extends FeeGrantParties {
    /**
     * A required, finite lifetime budget. Empty limits are deliberately rejected
     * because Cosmos SDK interprets them as unlimited.
     */
    readonly spendLimit: readonly FeeGrantCoin[];
    /** A required wall-clock expiry. */
    readonly expiration: Date;
    /**
     * Exact protobuf message type URLs that the sponsor agrees to pay for.
     * Wildcards and types outside the reviewed onboarding allowlist are rejected.
     */
    readonly allowedMessageTypeUrls: readonly string[];
}
export interface SponsoredFeeInput {
    readonly network: ZeroneNetwork;
    readonly granter: string;
    readonly amount: readonly FeeGrantCoin[];
    readonly gas: string;
}
/**
 * Structural counterpart of CosmJS's StdFee with a required fee granter.
 * Keeping this structural avoids making @cosmjs/amino a runtime dependency.
 */
export interface SponsoredStdFee {
    readonly amount: readonly FeeGrantCoin[];
    readonly gas: string;
    readonly granter: string;
}
/**
 * Message types reviewed for sponsor-funded Zerone onboarding.
 *
 * This is an allowlist, not a pattern: a newly added message remains rejected
 * until the SDK policy and the message's signer/authority behavior are audited.
 */
export declare const ZERONE_ONBOARDING_MESSAGE_TYPE_URLS: readonly ["/zerone.claiming_pot.v1.MsgClaim"];
/**
 * Builds a finite Cosmos SDK fee allowance for Zerone onboarding.
 *
 * The BasicAllowance is always wrapped in AllowedMsgAllowance, so neither the
 * spend budget, lifetime, nor permitted message set can be omitted.
 */
export declare function makeBoundedFeeGrant(input: BoundedFeeGrantInput): EncodeObject;
/** Builds a standard Cosmos SDK message that revokes one Zerone fee grant. */
export declare function makeRevokeFeeGrant(input: FeeGrantParties): EncodeObject;
/**
 * Adds a validated Zerone fee granter to a concrete positive CosmJS fee.
 *
 * This does not query the chain or prove that a matching allowance exists.
 */
export declare function makeSponsoredFee(input: SponsoredFeeInput): SponsoredStdFee;
