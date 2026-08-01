import { CID } from "multiformats/cid";

declare const canonicalCidV1Brand: unique symbol;
declare const zeroneMemoryCidBrand: unique symbol;

export type CanonicalCidV1 = string & {
  readonly [canonicalCidV1Brand]: true;
};
export type ZeroneMemoryCid = CanonicalCidV1 & {
  readonly [zeroneMemoryCidBrand]: true;
};

export type CidErrorCode =
  | "EMPTY_CID"
  | "CID_TOO_LONG"
  | "INVALID_CID"
  | "UNSUPPORTED_CID_VERSION"
  | "NON_CANONICAL_CID";

export class CidError extends Error {
  readonly code: CidErrorCode;

  constructor(code: CidErrorCode, message: string) {
    super(message);
    this.name = "CidError";
    this.code = code;
  }
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
export function parseCanonicalCidV1(value: string): CidV1Details {
  if (value.length === 0) {
    throw new CidError("EMPTY_CID", "CID is required");
  }

  let parsed: CID;
  try {
    parsed = CID.parse(value);
  } catch {
    throw new CidError("INVALID_CID", "Invalid CID");
  }
  if (parsed.version !== 1) {
    throw new CidError(
      "UNSUPPORTED_CID_VERSION",
      `CID version must be 1, got ${parsed.version}`,
    );
  }

  const canonical = parsed.toString();
  if (value !== canonical) {
    throw new CidError(
      "NON_CANONICAL_CID",
      `CIDv1 must use Zerone's lowercase-base32 representation ${canonical}`,
    );
  }

  return {
    cid: value as CanonicalCidV1,
    codec: parsed.code,
    multihashCode: parsed.multihash.code,
    digestBytes: parsed.multihash.digest.byteLength,
  };
}

/**
 * Applies the current x/home text-size limit after CIDv1 parsing. Validator
 * consensus still accepts legacy opaque values. Callers must opt into this
 * helper before constructing a newly signed MsgUpdateMemoryCID transaction.
 */
export function asZeroneMemoryCid(value: string): ZeroneMemoryCid {
  if (new TextEncoder().encode(value).byteLength > 256) {
    throw new CidError(
      "CID_TOO_LONG",
      "Zerone memory CID exceeds the on-chain 256-byte limit",
    );
  }
  return parseCanonicalCidV1(value).cid as ZeroneMemoryCid;
}
