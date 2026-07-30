import { sha256 } from "@cosmjs/crypto";
import { toBase64, toUtf8 } from "@cosmjs/encoding";

const OPAQUE_TOKEN = /^[A-Za-z0-9_-]{43}$/;

export function toBase64Url(bytes: Uint8Array): string {
  return toBase64(bytes)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/u, "");
}

export function opaqueToken(bytes: Uint8Array): string {
  if (bytes.length !== 32) {
    throw new Error("Opaque tokens require exactly 32 random bytes");
  }
  return toBase64Url(bytes);
}

export function isOpaqueToken(value: string): boolean {
  return OPAQUE_TOKEN.test(value);
}

export function hashOpaque(value: string): string {
  return toBase64Url(sha256(toUtf8(value)));
}

export async function keyedHash(secret: string, value: string): Promise<string> {
  const key = await globalThis.crypto.subtle.importKey(
    "raw",
    toUtf8(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const digest = await globalThis.crypto.subtle.sign(
    "HMAC",
    key,
    toUtf8(value),
  );
  return toBase64Url(new Uint8Array(digest));
}

export function constantTimeEqual(left: string, right: string): boolean {
  const leftBytes = toUtf8(left);
  const rightBytes = toUtf8(right);
  const maximum = Math.max(leftBytes.length, rightBytes.length);
  let difference = leftBytes.length ^ rightBytes.length;
  for (let index = 0; index < maximum; index += 1) {
    difference |= (leftBytes[index] ?? 0) ^ (rightBytes[index] ?? 0);
  }
  return difference === 0;
}
