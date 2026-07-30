import assert from "node:assert/strict";
import test from "node:test";

import {
  encodeSecp256k1Pubkey,
  encodeSecp256k1Signature,
  pubkeyToAddress,
} from "@cosmjs/amino";
import {
  Secp256k1,
  sha256,
} from "@cosmjs/crypto";

import {
  adr36SignBytes,
  parsePiStdSignature,
  verifyAdr36Signature,
  walletProofHash,
} from "../functions/api/pi/_wallet-proof";
import type { PiStdSignature } from "../functions/api/pi/_types";

const MESSAGE = "ZERONE Pi proof deterministic vector";
const ADDRESS = "zrn1w508d6qejxtdg4y5r3zarvary0c5xw7k4p057w";
const SIGNATURE: PiStdSignature = {
  pub_key: {
    type: "tendermint/PubKeySecp256k1",
    value: "Anm+Zn753LusVaBilc6HCwcCm/zbLc4o2VnygVsW+BeY",
  },
  signature:
    "BUrTlzA7hJzQU5NH5tTCF8EIqHkYJNuxXc28G3UWO9pvquSMdIqMDcRP9bNd/lIO/6zc55z+O2+dvVQXRVVWXA==",
};
const CANONICAL_SIGN_DOC =
  "{\"account_number\":\"0\",\"chain_id\":\"\",\"fee\":{\"amount\":[],\"gas\":\"0\"},\"memo\":\"\",\"msgs\":[{\"type\":\"sign/MsgSignData\",\"value\":{\"data\":\"WkVST05FIFBpIHByb29mIGRldGVybWluaXN0aWMgdmVjdG9y\",\"signer\":\"zrn1w508d6qejxtdg4y5r3zarvary0c5xw7k4p057w\"}}],\"sequence\":\"0\"}";

test("ADR-36 bytes and secp256k1 signature match a deterministic vector", () => {
  assert.equal(
    new TextDecoder().decode(adr36SignBytes(ADDRESS, MESSAGE)),
    CANONICAL_SIGN_DOC,
  );
  assert.equal(verifyAdr36Signature(ADDRESS, MESSAGE, SIGNATURE), true);

  const privateKey = new Uint8Array(32);
  privateKey[31] = 1;
  const publicKey = Secp256k1.compressPubkey(
    Secp256k1.makeKeypair(privateKey).pubkey,
  );
  assert.equal(
    pubkeyToAddress(encodeSecp256k1Pubkey(publicKey), "zrn"),
    ADDRESS,
  );
  const created = Secp256k1.createSignature(
    sha256(adr36SignBytes(ADDRESS, MESSAGE)),
    privateKey,
  );
  const fixedLength = new Uint8Array(64);
  fixedLength.set(created.r(32));
  fixedLength.set(created.s(32), 32);
  assert.deepEqual(
    encodeSecp256k1Signature(publicKey, fixedLength),
    SIGNATURE,
  );
});

test("ADR-36 verification rejects altered message, signer, and signature", () => {
  assert.equal(
    verifyAdr36Signature(ADDRESS, `${MESSAGE}.`, SIGNATURE),
    false,
  );
  assert.equal(
    verifyAdr36Signature(
      "zrn1qyqszqgpqyqszqgpqyqszqgpqyqszqgpnvv46q",
      MESSAGE,
      SIGNATURE,
    ),
    false,
  );
  assert.equal(
    verifyAdr36Signature(ADDRESS, MESSAGE, {
      ...SIGNATURE,
      signature: `${SIGNATURE.signature.slice(0, -2)}AA`,
    }),
    false,
  );
  assert.equal(
    verifyAdr36Signature(ADDRESS, MESSAGE, {
      ...SIGNATURE,
      pub_key: {
        ...SIGNATURE.pub_key,
        value: `${SIGNATURE.pub_key.value}=`,
      },
    }),
    false,
  );
});

test("signature parsing is exact and bounded", () => {
  assert.deepEqual(parsePiStdSignature(SIGNATURE), SIGNATURE);
  assert.equal(parsePiStdSignature(null), null);
  assert.equal(parsePiStdSignature([]), null);
  assert.equal(
    parsePiStdSignature({ ...SIGNATURE, unexpected: true }),
    null,
  );
  assert.equal(
    parsePiStdSignature({
      ...SIGNATURE,
      pub_key: { ...SIGNATURE.pub_key, unexpected: true },
    }),
    null,
  );
  assert.equal(
    parsePiStdSignature({
      ...SIGNATURE,
      pub_key: {
        ...SIGNATURE.pub_key,
        type: "tendermint/PubKeyEd25519",
      },
    }),
    null,
  );
  assert.equal(
    parsePiStdSignature({
      ...SIGNATURE,
      signature: "A".repeat(129),
    }),
    null,
  );
});

test("wallet proof fingerprints are stable, bounded, and transcript-specific", () => {
  const first = walletProofHash(MESSAGE, SIGNATURE);
  assert.match(first, /^[A-Za-z0-9_-]{43}$/u);
  assert.equal(walletProofHash(MESSAGE, SIGNATURE), first);
  assert.notEqual(walletProofHash(`${MESSAGE}.`, SIGNATURE), first);
  assert.notEqual(
    walletProofHash(MESSAGE, {
      ...SIGNATURE,
      signature: `${SIGNATURE.signature.slice(0, -2)}AA`,
    }),
    first,
  );
});
