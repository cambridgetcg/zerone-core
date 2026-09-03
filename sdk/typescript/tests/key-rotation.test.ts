import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  KEY_ROTATION_ACCEPTANCE_DOMAIN,
  KEY_ROTATION_AUTHORIZATION_DOMAIN,
  KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS,
  keyRotationAcceptanceSignBytes,
  keyRotationAuthorizationSignBytes,
} from "../src/key-rotation";

const sender = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z";

describe("operational-key rotation authorization", () => {
  it("matches the Go known-answer encoding", () => {
    const key = Uint8Array.from({ length: 32 }, (_, index) => index);
    const transition = {
      chainId: "zerone-test-1",
      sender,
      currentKeyVersion: 7,
      newOperationalKey: key,
      authorizationExpiresAtUnix: 1788436800n,
    };
    const authorization = keyRotationAuthorizationSignBytes(transition);
    const acceptance = keyRotationAcceptanceSignBytes(transition);
    assert.equal(
      Buffer.from(authorization).toString("hex"),
      "7a65726f6e652e617574682f726f746174652d6b65792f7631000000000d7a65726f6e652d746573742d310000002a7a726e316d3033376e3735766b326a6864723536793270747a6a6a6a3032756c6a776e7177777a72377a00000007000000006a996140000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
    );
    assert.equal(
      Buffer.from(acceptance).toString("hex"),
      "7a65726f6e652e617574682f6163636570742d6b65792f7631000000000d7a65726f6e652d746573742d310000002a7a726e316d3033376e3735766b326a6864723536793270747a6a6a6a3032756c6a776e7177777a72377a00000007000000006a996140000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
    );
    assert.notDeepEqual(authorization, acceptance);
    assert.equal(KEY_ROTATION_AUTHORIZATION_DOMAIN, "zerone.auth/rotate-key/v1");
    assert.equal(KEY_ROTATION_ACCEPTANCE_DOMAIN, "zerone.auth/accept-key/v1");
    assert.equal(KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS, 600n);
  });

  it("rejects ambiguous and out-of-range fields", () => {
    const valid = {
      chainId: "zerone-test-1",
      sender,
      currentKeyVersion: 1,
      newOperationalKey: new Uint8Array(32),
      authorizationExpiresAtUnix: 1n,
    };
    for (const candidate of [
      { ...valid, chainId: "" },
      { ...valid, chainId: " zerone-test-1" },
      { ...valid, chainId: "zerone-test-1\u0085" },
      { ...valid, chainId: "bad\ud800" },
      { ...valid, sender: "invalid" },
      { ...valid, currentKeyVersion: 0 },
      { ...valid, currentKeyVersion: 0x1_0000_0000 },
      { ...valid, newOperationalKey: new Uint8Array(31) },
      { ...valid, authorizationExpiresAtUnix: 0n },
      { ...valid, authorizationExpiresAtUnix: 0x8000_0000_0000_0000n },
    ]) {
      assert.throws(() => keyRotationAuthorizationSignBytes(candidate));
    }
  });
});
