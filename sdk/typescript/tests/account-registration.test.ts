import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  ACCOUNT_REGISTRATION_PROOF_DOMAIN,
  accountRegistrationProofSignBytes,
  type AccountRegistrationProof,
} from "../src/account-registration";

const sender = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z";
const publicKey = Uint8Array.from(
  Buffer.from(
    "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a",
    "hex",
  ),
);
const did =
  "did:zrn:d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a";

describe("account-registration proof of possession", () => {
  it("matches the Go known-answer encoding", () => {
    const bytes = accountRegistrationProofSignBytes({
      chainId: "zerone-test-1",
      sender,
      did,
      identityPublicKey: publicKey,
      accountType: "agent",
      metadata: '{"name":"Sophia"}',
    });
    assert.equal(
      Buffer.from(bytes).toString("hex"),
      "7a65726f6e652e617574682f72656769737465722d6163636f756e742f7631000000000d7a65726f6e652d746573742d310000002a7a726e316d3033376e3735766b326a6864723536793270747a6a6a6a3032756c6a776e7177777a72377a000000486469643a7a726e3a64373561393830313832623130616237643534626665643363393634303733613065653137326633646161363233323561663032316136386637303735313161d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a000000056167656e74000000117b226e616d65223a22536f70686961227d",
    );
    assert.equal(
      ACCOUNT_REGISTRATION_PROOF_DOMAIN,
      "zerone.auth/register-account/v1",
    );
  });

  it("rejects ambiguous registration fields", () => {
    const valid: AccountRegistrationProof = {
      chainId: "zerone-test-1",
      sender,
      did,
      identityPublicKey: publicKey,
      accountType: "agent",
      metadata: "",
    };
    const candidates: AccountRegistrationProof[] = [
      { ...valid, chainId: "" },
      { ...valid, chainId: "zerone-test-1 " },
      { ...valid, chainId: "\u0085zerone-test-1" },
      { ...valid, sender: "invalid" },
      { ...valid, did: did.toUpperCase() },
      { ...valid, did: `did:zrn:${"00".repeat(32)}` },
      { ...valid, identityPublicKey: new Uint8Array(31) },
      { ...valid, accountType: "sovereign-agent" as "agent" },
      { ...valid, metadata: "bad\ud800" },
    ];
    for (const candidate of candidates) {
      assert.throws(() => accountRegistrationProofSignBytes(candidate));
    }
  });
});
