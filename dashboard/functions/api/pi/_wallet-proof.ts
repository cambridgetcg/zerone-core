import {
  decodeSignature,
  makeSignDoc,
  pubkeyToAddress,
  serializeSignDoc,
} from "@cosmjs/amino";
import {
  Secp256k1,
  Secp256k1Signature,
  sha256,
} from "@cosmjs/crypto";
import {
  toBase64,
  toUtf8,
} from "@cosmjs/encoding";

import { toBase64Url } from "./_crypto";
import type { PiStdSignature } from "./_types";

type JsonRecord = Record<string, unknown>;

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasOnlyKeys(value: JsonRecord, allowed: readonly string[]): boolean {
  const allowedKeys = new Set(allowed);
  return Object.keys(value).every((key) => allowedKeys.has(key));
}

export function parsePiStdSignature(value: unknown): PiStdSignature | null {
  if (!isRecord(value) || !hasOnlyKeys(value, ["pub_key", "signature"])) {
    return null;
  }
  if (
    !isRecord(value.pub_key) ||
    !hasOnlyKeys(value.pub_key, ["type", "value"]) ||
    value.pub_key.type !== "tendermint/PubKeySecp256k1" ||
    typeof value.pub_key.value !== "string" ||
    value.pub_key.value.length > 64 ||
    typeof value.signature !== "string" ||
    value.signature.length > 128
  ) {
    return null;
  }
  return {
    pub_key: {
      type: "tendermint/PubKeySecp256k1",
      value: value.pub_key.value,
    },
    signature: value.signature,
  };
}

export function adr36SignBytes(address: string, message: string): Uint8Array {
  const signDoc = makeSignDoc(
    [
      {
        type: "sign/MsgSignData",
        value: {
          signer: address,
          data: toBase64(toUtf8(message)),
        },
      },
    ],
    { amount: [], gas: "0" },
    "",
    "",
    "0",
    "0",
  );
  return serializeSignDoc(signDoc);
}

export function verifyAdr36Signature(
  address: string,
  message: string,
  signature: PiStdSignature,
): boolean {
  try {
    if (signature.pub_key.type !== "tendermint/PubKeySecp256k1") {
      return false;
    }
    const decoded = decodeSignature(signature);
    if (
      decoded.pubkey.length !== 33 ||
      (decoded.pubkey[0] !== 0x02 && decoded.pubkey[0] !== 0x03) ||
      decoded.signature.length !== 64 ||
      toBase64(decoded.pubkey) !== signature.pub_key.value ||
      toBase64(decoded.signature) !== signature.signature ||
      pubkeyToAddress(signature.pub_key, "zrn") !== address
    ) {
      return false;
    }
    return Secp256k1.verifySignature(
      Secp256k1Signature.fromFixedLength(decoded.signature),
      sha256(adr36SignBytes(address, message)),
      decoded.pubkey,
    );
  } catch {
    return false;
  }
}

export function walletProofHash(
  message: string,
  signature: PiStdSignature,
): string {
  const canonical = [
    "zerone-pi-wallet-proof-v1",
    message,
    signature.pub_key.type,
    signature.pub_key.value,
    signature.signature,
  ].join("\u0000");
  return toBase64Url(sha256(toUtf8(canonical)));
}
