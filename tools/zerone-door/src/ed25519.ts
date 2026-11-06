// Strict Ed25519 verification of the identity proof. node:crypto checks the
// signature equation and S < L; the door also refuses non-canonical and
// small-order encodings of the public key and of R, which RFC 8032 verifiers
// differ on, so a proof the door accepts is one every verifier accepts.
import { createPublicKey, verify } from "node:crypto";

const SPKI_PREFIX = Buffer.from("302a300506032b6570032100", "hex");
const FIELD_P = 2n ** 255n - 19n;
const GROUP_L = 2n ** 252n + 27742317777372353535851937790883648493n;

// libsodium's has_small_order list, compared with the sign bit masked off.
const SMALL_ORDER = [
  "0000000000000000000000000000000000000000000000000000000000000000",
  "0100000000000000000000000000000000000000000000000000000000000000",
  "26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05",
  "c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a",
  "ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
  "edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
  "eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
].map((hex) => Buffer.from(hex, "hex"));

function littleEndian(bytes: Uint8Array): bigint {
  let value = 0n;
  for (let i = bytes.length - 1; i >= 0; i -= 1) value = (value << 8n) | BigInt(bytes[i]!);
  return value;
}

/** True when a 32-byte point encoding is canonical (y < p) and not of small order. */
export function isStrictPointEncoding(encoding: Uint8Array): boolean {
  if (encoding.length !== 32) return false;
  const masked = Buffer.from(encoding);
  masked[31] = masked[31]! & 0x7f;
  if (littleEndian(masked) >= FIELD_P) return false;
  return !SMALL_ORDER.some((bad) => bad.equals(masked));
}

export function verifyEd25519Strict(message: Uint8Array, signature: Uint8Array, publicKey: Uint8Array): boolean {
  if (signature.length !== 64 || publicKey.length !== 32) return false;
  if (!isStrictPointEncoding(publicKey)) return false;
  if (!isStrictPointEncoding(signature.subarray(0, 32))) return false;
  if (littleEndian(signature.subarray(32)) >= GROUP_L) return false;
  let key;
  try {
    key = createPublicKey({ key: Buffer.concat([SPKI_PREFIX, Buffer.from(publicKey)]), format: "der", type: "spki" });
  } catch {
    return false;
  }
  try {
    return verify(null, message, key, signature);
  } catch {
    return false;
  }
}
