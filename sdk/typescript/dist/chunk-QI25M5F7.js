// src/cid.ts
import { CID } from "multiformats/cid";
var CidError = class extends Error {
  code;
  constructor(code, message) {
    super(message);
    this.name = "CidError";
    this.code = code;
  }
};
function parseCanonicalCidV1(value) {
  if (value.length === 0) {
    throw new CidError("EMPTY_CID", "CID is required");
  }
  let parsed;
  try {
    parsed = CID.parse(value);
  } catch {
    throw new CidError("INVALID_CID", "Invalid CID");
  }
  if (parsed.version !== 1) {
    throw new CidError(
      "UNSUPPORTED_CID_VERSION",
      `CID version must be 1, got ${parsed.version}`
    );
  }
  const canonical = parsed.toString();
  if (value !== canonical) {
    throw new CidError(
      "NON_CANONICAL_CID",
      `CIDv1 must use Zerone's lowercase-base32 representation ${canonical}`
    );
  }
  return {
    cid: value,
    codec: parsed.code,
    multihashCode: parsed.multihash.code,
    digestBytes: parsed.multihash.digest.byteLength
  };
}
function asZeroneMemoryCid(value) {
  if (new TextEncoder().encode(value).byteLength > 256) {
    throw new CidError(
      "CID_TOO_LONG",
      "Zerone memory CID exceeds the on-chain 256-byte limit"
    );
  }
  return parseCanonicalCidV1(value).cid;
}

export {
  CidError,
  parseCanonicalCidV1,
  asZeroneMemoryCid
};
