declare const canonicalCidV1Brand: unique symbol;
declare const zeroneMemoryCidBrand: unique symbol;
export type CanonicalCidV1 = string & {
    readonly [canonicalCidV1Brand]: true;
};
export type ZeroneMemoryCid = CanonicalCidV1 & {
    readonly [zeroneMemoryCidBrand]: true;
};
export type CidErrorCode = "EMPTY_CID" | "CID_TOO_LONG" | "INVALID_CID" | "UNSUPPORTED_CID_VERSION" | "NON_CANONICAL_CID";
export declare class CidError extends Error {
    readonly code: CidErrorCode;
    constructor(code: CidErrorCode, message: string);
}
export interface CidV1Details {
    readonly cid: CanonicalCidV1;
    readonly codec: number;
    readonly multihashCode: number;
    readonly digestBytes: number;
}
/**
 * Parses a CIDv1 and requires Zerone's chosen lowercase-base32 text form.
 *
 * CID permits other multibase strings for the same identifier. This validates
 * identifier structure only; it does not hash referenced bytes, establish
 * availability or authenticity, or choose a codec or multihash policy.
 */
export declare function parseCanonicalCidV1(value: string): CidV1Details;
/**
 * Applies the current x/home text-size limit after CIDv1 parsing. Validator
 * consensus still accepts legacy opaque values. Callers must opt into this
 * helper before constructing a newly signed MsgUpdateMemoryCID transaction.
 */
export declare function asZeroneMemoryCid(value: string): ZeroneMemoryCid;
export {};
