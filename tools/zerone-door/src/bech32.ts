// BIP-173 bech32 for zrn1 account addresses. The door compares addresses by
// their 20-byte payload, never by spelling, and accepts only the canonical
// lowercase spelling as input (pot IDs on zerone-1 are raw address strings, so
// "bootstrap-ZRN1..." is a different pot from "bootstrap-zrn1...").

const CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l";
const GENERATOR = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];

export const ZERONE_HRP = "zrn";

function polymod(values: number[]): number {
  let chk = 1;
  for (const value of values) {
    const top = chk >>> 25;
    chk = ((chk & 0x1ffffff) << 5) ^ value;
    for (let i = 0; i < 5; i += 1) {
      if ((top >>> i) & 1) chk ^= GENERATOR[i]!;
    }
  }
  return chk >>> 0;
}

function hrpExpand(hrp: string): number[] {
  const out: number[] = [];
  for (const ch of hrp) out.push(ch.charCodeAt(0) >> 5);
  out.push(0);
  for (const ch of hrp) out.push(ch.charCodeAt(0) & 31);
  return out;
}

function convertBits(data: readonly number[], from: number, to: number, pad: boolean): number[] | null {
  let acc = 0;
  let bits = 0;
  const out: number[] = [];
  const maxv = (1 << to) - 1;
  for (const value of data) {
    if (value < 0 || value >> from !== 0) return null;
    acc = (acc << from) | value;
    bits += from;
    while (bits >= to) {
      bits -= to;
      out.push((acc >> bits) & maxv);
    }
  }
  if (pad) {
    if (bits > 0) out.push((acc << (to - bits)) & maxv);
  } else if (bits >= from || ((acc << (to - bits)) & maxv) !== 0) {
    return null;
  }
  return out;
}

export function encodeBech32(hrp: string, payload: Uint8Array): string {
  const data = convertBits([...payload], 8, 5, true);
  if (data === null) throw new Error("bech32: cannot convert payload");
  const values = [...hrpExpand(hrp), ...data, 0, 0, 0, 0, 0, 0];
  const mod = polymod(values) ^ 1;
  const checksum: number[] = [];
  for (let i = 0; i < 6; i += 1) checksum.push((mod >>> (5 * (5 - i))) & 31);
  return `${hrp}1${[...data, ...checksum].map((d) => CHARSET[d]).join("")}`;
}

export interface DecodedBech32 {
  hrp: string;
  payload: Uint8Array;
}

/** Decodes any valid bech32 string (either case, never mixed). Returns null on
 * any defect. Callers that need the canonical spelling use parseZeroneAddress. */
export function decodeBech32(input: string): DecodedBech32 | null {
  if (input.length < 8 || input.length > 90) return null;
  const lower = input.toLowerCase();
  if (input !== lower && input !== input.toUpperCase()) return null;
  const sep = lower.lastIndexOf("1");
  if (sep < 1 || sep + 7 > lower.length) return null;
  const hrp = lower.slice(0, sep);
  for (const ch of hrp) {
    const code = ch.charCodeAt(0);
    if (code < 33 || code > 126) return null;
  }
  const data: number[] = [];
  for (const ch of lower.slice(sep + 1)) {
    const index = CHARSET.indexOf(ch);
    if (index === -1) return null;
    data.push(index);
  }
  if (polymod([...hrpExpand(hrp), ...data]) !== 1) return null;
  const bytes = convertBits(data.slice(0, -6), 5, 8, false);
  if (bytes === null) return null;
  return { hrp, payload: Uint8Array.from(bytes) };
}

export type AddressDefect =
  | "not_lowercase"
  | "not_bech32"
  | "wrong_prefix"
  | "wrong_length"
  | "not_canonical";

export type ParsedAddress =
  | { ok: true; address: string; payload: Uint8Array; payloadHex: string }
  | { ok: false; defect: AddressDefect };

/** Accepts only a canonical lowercase zrn1 address of a 20-byte account. */
export function parseZeroneAddress(input: string): ParsedAddress {
  if (input !== input.toLowerCase()) return { ok: false, defect: "not_lowercase" };
  const decoded = decodeBech32(input);
  if (decoded === null) return { ok: false, defect: "not_bech32" };
  if (decoded.hrp !== ZERONE_HRP) return { ok: false, defect: "wrong_prefix" };
  if (decoded.payload.length !== 20) return { ok: false, defect: "wrong_length" };
  if (encodeBech32(ZERONE_HRP, decoded.payload) !== input) return { ok: false, defect: "not_canonical" };
  return { ok: true, address: input, payload: decoded.payload, payloadHex: Buffer.from(decoded.payload).toString("hex") };
}

/** The 20-byte payload of any valid zrn address spelling, as lowercase hex, or
 * null. Used to compare a request with household and ledger addresses. */
export function addressPayloadHex(input: string): string | null {
  const decoded = decodeBech32(input);
  if (decoded === null || decoded.hrp !== ZERONE_HRP || decoded.payload.length !== 20) return null;
  return Buffer.from(decoded.payload).toString("hex");
}
