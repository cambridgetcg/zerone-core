// secp256k1 ECDSA verification for ADR-036 request signatures. Bun's
// node:crypto (BoringSSL) has no secp256k1 group, and this tool has no npm
// dependencies, so verification is done here with BigInt arithmetic. It only
// ever handles public values (keys, signatures, digests): nothing secret
// passes through it, so constant-time behaviour is not required.
import { createHash } from "node:crypto";

export const P = 2n ** 256n - 2n ** 32n - 977n;
export const N = 0xfffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141n;
const HALF_N = N >> 1n;
const GX = 0x79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798n;
const GY = 0x483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8n;

type Jacobian = [bigint, bigint, bigint];
const INFINITY: Jacobian = [0n, 1n, 0n];

function mod(a: bigint, m: bigint = P): bigint {
  const r = a % m;
  return r >= 0n ? r : r + m;
}

function powMod(base: bigint, exp: bigint, m: bigint): bigint {
  let result = 1n;
  let b = mod(base, m);
  let e = exp;
  while (e > 0n) {
    if (e & 1n) result = (result * b) % m;
    b = (b * b) % m;
    e >>= 1n;
  }
  return result;
}

function invert(a: bigint, m: bigint): bigint {
  // Extended Euclid; a is never 0 here (callers check ranges first).
  let [oldR, r] = [mod(a, m), m];
  let [oldS, s] = [1n, 0n];
  while (r !== 0n) {
    const q = oldR / r;
    [oldR, r] = [r, oldR - q * r];
    [oldS, s] = [s, oldS - q * s];
  }
  if (oldR !== 1n) throw new Error("secp256k1: value has no inverse");
  return mod(oldS, m);
}

function double(p: Jacobian): Jacobian {
  const [x, y, z] = p;
  if (z === 0n || y === 0n) return INFINITY;
  const ysq = mod(y * y);
  const s = mod(4n * x * ysq);
  const m = mod(3n * x * x);
  const nx = mod(m * m - 2n * s);
  const ny = mod(m * (s - nx) - 8n * ysq * ysq);
  const nz = mod(2n * y * z);
  return [nx, ny, nz];
}

function add(p: Jacobian, q: Jacobian): Jacobian {
  if (p[2] === 0n) return q;
  if (q[2] === 0n) return p;
  const [x1, y1, z1] = p;
  const [x2, y2, z2] = q;
  const z1z1 = mod(z1 * z1);
  const z2z2 = mod(z2 * z2);
  const u1 = mod(x1 * z2z2);
  const u2 = mod(x2 * z1z1);
  const s1 = mod(y1 * z2 * z2z2);
  const s2 = mod(y2 * z1 * z1z1);
  if (u1 === u2) {
    if (s1 !== s2) return INFINITY;
    return double(p);
  }
  const h = mod(u2 - u1);
  const r = mod(s2 - s1);
  const hh = mod(h * h);
  const hhh = mod(h * hh);
  const v = mod(u1 * hh);
  const nx = mod(r * r - hhh - 2n * v);
  const ny = mod(r * (v - nx) - s1 * hhh);
  const nz = mod(h * z1 * z2);
  return [nx, ny, nz];
}

function multiply(point: Jacobian, scalar: bigint): Jacobian {
  let result = INFINITY;
  let addend = point;
  let k = scalar;
  while (k > 0n) {
    if (k & 1n) result = add(result, addend);
    addend = double(addend);
    k >>= 1n;
  }
  return result;
}

function toAffine(p: Jacobian): [bigint, bigint] | null {
  if (p[2] === 0n) return null;
  const zInv = invert(p[2], P);
  const zInv2 = mod(zInv * zInv);
  return [mod(p[0] * zInv2), mod(p[1] * zInv2 * zInv)];
}

function bytesToBigInt(bytes: Uint8Array): bigint {
  return bytes.length === 0 ? 0n : BigInt(`0x${Buffer.from(bytes).toString("hex")}`);
}

function bigIntTo32(value: bigint): Uint8Array {
  return Uint8Array.from(Buffer.from(value.toString(16).padStart(64, "0"), "hex"));
}

/** Decompresses a 33-byte SEC1 public key, refusing anything off the curve. */
export function decompressPublicKey(compressed: Uint8Array): [bigint, bigint] | null {
  if (compressed.length !== 33) return null;
  const prefix = compressed[0];
  if (prefix !== 0x02 && prefix !== 0x03) return null;
  const x = bytesToBigInt(compressed.subarray(1));
  if (x >= P) return null;
  const rhs = mod(x * x * x + 7n);
  let y = powMod(rhs, (P + 1n) / 4n, P);
  if (mod(y * y) !== rhs) return null;
  if ((y & 1n) !== BigInt(prefix & 1)) y = P - y;
  return [x, y];
}

/** Cosmos SDK semantics: 64-byte r||s, 0 < r,s < n, and low-S (s <= n/2). */
export function verifySecp256k1(digest: Uint8Array, signature: Uint8Array, compressedPublicKey: Uint8Array): boolean {
  if (digest.length !== 32 || signature.length !== 64) return false;
  const point = decompressPublicKey(compressedPublicKey);
  if (point === null) return false;
  const r = bytesToBigInt(signature.subarray(0, 32));
  const s = bytesToBigInt(signature.subarray(32));
  if (r <= 0n || r >= N || s <= 0n || s >= N) return false;
  if (s > HALF_N) return false;
  const e = bytesToBigInt(digest);
  const w = invert(s, N);
  const u1 = mod(e * w, N);
  const u2 = mod(r * w, N);
  const sum = add(multiply([GX, GY, 1n], u1), multiply([point[0], point[1], 1n], u2));
  const affine = toAffine(sum);
  if (affine === null) return false;
  return mod(affine[0], N) === r;
}

/** Cosmos account address bytes: RIPEMD-160(SHA-256(compressed public key)). */
export function accountAddressBytes(compressedPublicKey: Uint8Array): Uint8Array {
  const sha = createHash("sha256").update(compressedPublicKey).digest();
  return Uint8Array.from(createHash("ripemd160").update(sha).digest());
}

// ── test support: deterministic signing for fixtures only. The door never
//    signs with secp256k1 itself; R signs through the pinned zeroned CLI. ──

export function publicKeyFromPrivate(privateKey: Uint8Array): Uint8Array {
  const d = bytesToBigInt(privateKey);
  if (d <= 0n || d >= N) throw new Error("secp256k1: private key out of range");
  const affine = toAffine(multiply([GX, GY, 1n], d));
  if (affine === null) throw new Error("secp256k1: bad private key");
  const out = new Uint8Array(33);
  out[0] = (affine[1] & 1n) === 1n ? 0x03 : 0x02;
  out.set(bigIntTo32(affine[0]), 1);
  return out;
}

/** Low-S ECDSA with a caller-chosen nonce. Test fixtures only. */
export function signSecp256k1ForTest(digest: Uint8Array, privateKey: Uint8Array, nonce: bigint): Uint8Array {
  const d = bytesToBigInt(privateKey);
  const k = mod(nonce, N);
  if (k === 0n) throw new Error("secp256k1: zero nonce");
  const affine = toAffine(multiply([GX, GY, 1n], k));
  if (affine === null) throw new Error("secp256k1: bad nonce");
  const r = mod(affine[0], N);
  let s = mod(invert(k, N) * (bytesToBigInt(digest) + r * d), N);
  if (s > HALF_N) s = N - s;
  const out = new Uint8Array(64);
  out.set(bigIntTo32(r), 0);
  out.set(bigIntTo32(s), 32);
  return out;
}

export function highSVariantForTest(signature: Uint8Array): Uint8Array {
  const s = bytesToBigInt(signature.subarray(32));
  const out = new Uint8Array(signature);
  out.set(bigIntTo32(N - s), 32);
  return out;
}
