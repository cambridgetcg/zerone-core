import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { base32 } from "multiformats/bases/base32";
import { base58btc } from "multiformats/bases/base58";
import { CID } from "multiformats/cid";
import {
  CidError,
  asZeroneMemoryCid,
  parseCanonicalCidV1,
} from "../src/cid";

const CANONICAL_CID_V1 =
  "bafzbeigai3eoy2ccc7ybwjfz5r3rdxqrinwi4rwytly24tdbh6yk7zslrm";
const CID_V0 = "QmYwAPJzv5CZsnAzt8auVZRnZrmPaUe2rLCsShjUEiB2yR";

interface CidVector {
  readonly name: string;
  readonly value: string;
  readonly parse: boolean;
  readonly memory: boolean;
}

const sharedVectors = JSON.parse(
  readFileSync(
    new URL("../../../testdata/cid-v1-vectors.json", import.meta.url),
    "utf8",
  ),
) as CidVector[];

function encodeVarint(value: number): number[] {
  const bytes: number[] = [];
  let remaining = value;
  while (remaining >= 0x80) {
    bytes.push((remaining & 0x7f) | 0x80);
    remaining >>>= 7;
  }
  bytes.push(remaining);
  return bytes;
}

function identityCid(digestBytes: number): string {
  return base32.encode(
    Uint8Array.from([
      1,
      0x55,
      0,
      ...encodeVarint(digestBytes),
      ...new Array<number>(digestBytes).fill(0x61),
    ]),
  );
}

function assertCidError(
  operation: () => unknown,
  code: InstanceType<typeof CidError>["code"],
): void {
  assert.throws(
    operation,
    (error: unknown) => error instanceof CidError && error.code === code,
  );
}

describe("CIDv1 preflight", () => {
  it("returns stable codec and multihash details", () => {
    const details = parseCanonicalCidV1(CANONICAL_CID_V1);
    assert.equal(details.cid, CANONICAL_CID_V1);
    assert.equal(details.codec, 0x72);
    assert.equal(details.multihashCode, 0x12);
    assert.equal(details.digestBytes, 32);
    assert.equal(asZeroneMemoryCid(CANONICAL_CID_V1), CANONICAL_CID_V1);
  });

  it("rejects CIDv0 and malformed input", () => {
    assertCidError(() => parseCanonicalCidV1(""), "EMPTY_CID");
    assertCidError(() => parseCanonicalCidV1("not-a-cid"), "INVALID_CID");
    assertCidError(
      () => parseCanonicalCidV1(CID_V0),
      "UNSUPPORTED_CID_VERSION",
    );
  });

  it("rejects valid but noncanonical CIDv1 text", () => {
    const alternate = CID.parse(CANONICAL_CID_V1).toString(base58btc);
    assert.notEqual(alternate, CANONICAL_CID_V1);
    assertCidError(
      () => parseCanonicalCidV1(alternate),
      "NON_CANONICAL_CID",
    );
  });

  it("applies the x/home byte limit separately from CID parsing", () => {
    assertCidError(
      () => asZeroneMemoryCid(`b${"a".repeat(256)}`),
      "CID_TOO_LONG",
    );

    const atLimit = identityCid(154);
    const overLimit = identityCid(155);
    assert.equal(new TextEncoder().encode(atLimit).byteLength, 256);
    assert.equal(new TextEncoder().encode(overLimit).byteLength, 257);
    assert.equal(asZeroneMemoryCid(atLimit), atLimit);
    assert.equal(parseCanonicalCidV1(overLimit).cid, overLimit);
    assertCidError(() => asZeroneMemoryCid(overLimit), "CID_TOO_LONG");
  });

  it("matches the shared Go/TypeScript adversarial corpus", () => {
    for (const vector of sharedVectors) {
      let parsed = false;
      try {
        parseCanonicalCidV1(vector.value);
        parsed = true;
      } catch (error) {
        assert.ok(error instanceof CidError, vector.name);
      }
      assert.equal(parsed, vector.parse, `${vector.name}: parse`);

      let memory = false;
      try {
        asZeroneMemoryCid(vector.value);
        memory = true;
      } catch (error) {
        assert.ok(error instanceof CidError, vector.name);
      }
      assert.equal(memory, vector.memory, `${vector.name}: memory`);
    }
  });
});
