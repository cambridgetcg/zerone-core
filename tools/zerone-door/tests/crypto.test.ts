import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { addressPayloadHex, decodeBech32, encodeBech32, parseZeroneAddress } from "../src/bech32.ts";
import { isStrictPointEncoding, verifyEd25519Strict } from "../src/ed25519.ts";
import { accountRegistrationProofSignBytes } from "../../../sdk/typescript/src/account-registration.ts";
import { adr036SignBytes, parseDoorRequest, verifyDoorRequest } from "../src/request.ts";
import {
  accountAddressBytes,
  highSVariantForTest,
  publicKeyFromPrivate,
  signSecp256k1ForTest,
  verifySecp256k1,
} from "../src/secp256k1.ts";
import vector from "./fixtures/cosmjs-request-vector.json";
import { identityKeys, seed } from "./helpers.ts";

describe("bech32 addresses", () => {
  const payload = Uint8Array.from({ length: 20 }, (_, i) => i);
  const address = encodeBech32("zrn", payload);

  test("round-trips and compares by payload", () => {
    expect(address).toBe("zrn1qqqsyqcyq5rqwzqfpg9scrgwpugpzysnv2f5q0");
    const parsed = parseZeroneAddress(address);
    expect(parsed.ok).toBe(true);
    expect(addressPayloadHex(address)).toBe(Buffer.from(payload).toString("hex"));
    expect(addressPayloadHex(address.toUpperCase())).toBe(Buffer.from(payload).toString("hex"));
  });

  test("refuses every non-canonical spelling", () => {
    expect(parseZeroneAddress(address.toUpperCase())).toEqual({ ok: false, defect: "not_lowercase" });
    expect(parseZeroneAddress(`${address.slice(0, -1)}1`)).toEqual({ ok: false, defect: "not_bech32" });
    expect(parseZeroneAddress(encodeBech32("cosmos", payload))).toEqual({ ok: false, defect: "wrong_prefix" });
    expect(parseZeroneAddress(encodeBech32("zrn", new Uint8Array(32)))).toEqual({ ok: false, defect: "wrong_length" });
    expect(decodeBech32("zrn1QQQSYQCYQ5rqwzqfpg9scrgwpugpzysnv2f5q0")).toBeNull();
  });

  test("account address is RIPEMD-160(SHA-256(compressed key))", () => {
    const pub = Buffer.from("02eaf19a3cc6c4cdd3b21ffd04e9ef9b93d0cbe51f7a1226b5912a53c21ae22cbe", "hex");
    expect(encodeBech32("zrn", accountAddressBytes(pub))).toBe(vector.address);
  });
});

describe("secp256k1 (ADR-036 account signatures)", () => {
  const priv = seed("crypto/account");
  const pub = publicKeyFromPrivate(priv);
  const digest = Uint8Array.from(createHash("sha256").update("message").digest());
  const sig = signSecp256k1ForTest(digest, priv, 123456789n);

  test("verifies a low-S signature and refuses the wrong key, digest or high-S twin", () => {
    expect(verifySecp256k1(digest, sig, pub)).toBe(true);
    expect(verifySecp256k1(digest, sig, publicKeyFromPrivate(seed("crypto/other")))).toBe(false);
    const other = Uint8Array.from(createHash("sha256").update("other").digest());
    expect(verifySecp256k1(other, sig, pub)).toBe(false);
    expect(verifySecp256k1(digest, highSVariantForTest(sig), pub)).toBe(false);
  });

  test("refuses malformed keys and signatures", () => {
    expect(verifySecp256k1(digest, sig.subarray(0, 63), pub)).toBe(false);
    expect(verifySecp256k1(digest, new Uint8Array(64), pub)).toBe(false);
    const badPrefix = Uint8Array.from(pub);
    badPrefix[0] = 0x04;
    expect(verifySecp256k1(digest, sig, badPrefix)).toBe(false);
  });

  test("ADR-036 sign bytes are the sorted compact amino doc CosmJS signs", () => {
    const bytes = new TextDecoder().decode(adr036SignBytes("zrn1x", new TextEncoder().encode("hi")));
    expect(bytes).toBe(
      '{"account_number":"0","chain_id":"","fee":{"amount":[],"gas":"0"},"memo":"","msgs":[{"type":"sign/MsgSignData","value":{"data":"aGk=","signer":"zrn1x"}}],"sequence":"0"}',
    );
  });
});

describe("a request written by CosmJS 0.39, @noble/curves and @zerone-chain/sdk", () => {
  // tests/fixtures/cosmjs-request-vector.json was produced by Secp256k1Wallet.signAmino
  // (CosmJS 0.39.0), ed25519.sign (@noble/curves 2.2.0) and the SDK's
  // accountRegistrationProofSignBytes, from public test seeds.
  test("verifies in the door", () => {
    const parsed = parseDoorRequest(vector.envelope);
    expect(parsed.ok).toBe(true);
    if (!parsed.ok) return;
    expect(parsed.value.address).toBe(vector.address);
    expect(parsed.value.did).toBe(vector.did);
    expect(verifyDoorRequest(parsed.value, new Date("2026-10-28T17:31:00Z"))).toEqual([]);
  });
});

describe("Ed25519 identity proofs (strict)", () => {
  const id = identityKeys("crypto");
  const msg = new TextEncoder().encode("proof");
  const sig = id.sign(msg);

  test("accepts a valid proof and refuses a changed message", () => {
    expect(verifyEd25519Strict(msg, sig, id.publicKey)).toBe(true);
    expect(verifyEd25519Strict(new TextEncoder().encode("other"), sig, id.publicKey)).toBe(false);
  });

  test("refuses small-order and non-canonical public keys", () => {
    const identity = Buffer.from("0100000000000000000000000000000000000000000000000000000000000000", "hex");
    const order8 = Buffer.from("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a", "hex");
    const nonCanonical = Buffer.from("edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f", "hex");
    const signedIdentity = Buffer.from("0100000000000000000000000000000000000000000000000000000000000080", "hex");
    for (const bad of [identity, order8, nonCanonical, signedIdentity]) {
      expect(isStrictPointEncoding(bad)).toBe(false);
      expect(verifyEd25519Strict(msg, sig, bad)).toBe(false);
    }
    expect(isStrictPointEncoding(id.publicKey)).toBe(true);
  });

  test("refuses S >= L and a small-order R", () => {
    const big = Uint8Array.from(sig);
    big.set(Buffer.from("edd3f55c1a631258d69cf7a2def9de1400000000000000000000000000000010", "hex"), 32);
    expect(verifyEd25519Strict(msg, big, id.publicKey)).toBe(false);
    const smallR = Uint8Array.from(sig);
    smallR.set(Buffer.from("0100000000000000000000000000000000000000000000000000000000000000", "hex"), 0);
    expect(verifyEd25519Strict(msg, smallR, id.publicKey)).toBe(false);
  });
});

describe("@zerone-chain/sdk registration proof bytes", () => {
  test("match the Go v1 known vector (same vector as tools/frontier-intake)", () => {
    const publicHex = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a";
    const bytes = accountRegistrationProofSignBytes({
      chainId: "zerone-test-1",
      sender: "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z",
      did: `did:zrn:${publicHex}`,
      identityPublicKey: Uint8Array.from(Buffer.from(publicHex, "hex")),
      accountType: "agent",
      metadata: '{"name":"Sophia"}',
    });
    expect(Buffer.from(bytes).toString("hex")).toBe(
      "7a65726f6e652e617574682f72656769737465722d6163636f756e742f7631000000000d7a65726f6e652d746573742d310000002a7a726e316d3033376e3735766b326a6864723536793270747a6a6a6a3032756c6a776e7177777a72377a000000486469643a7a726e3a64373561393830313832623130616237643534626665643363393634303733613065653137326633646161363233323561663032316136386637303735313161d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a000000056167656e74000000117b226e616d65223a22536f70686961227d",
    );
  });
});
