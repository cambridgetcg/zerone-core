import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { BinaryReader, BinaryWriter } from "../src/generated/binary";
import { MsgRotateKey } from "../src/generated/zerone/auth/v1/tx";

const U64 = (1n << 64n) - 1n;

// Independent reference: protobuf base-128 varint over the two's-complement
// 64-bit pattern, built from bigint arithmetic only.
function referenceVarint(value: bigint): string {
  let remaining = BigInt.asUintN(64, value);
  const bytes: number[] = [];
  do {
    let byte = Number(remaining & 0x7fn);
    remaining >>= 7n;
    if (remaining !== 0n) byte |= 0x80;
    bytes.push(byte);
  } while (remaining !== 0n);
  return Buffer.from(bytes).toString("hex");
}

function referenceZigZag(value: bigint): bigint {
  return BigInt.asUintN(64, (value << 1n) ^ (value >> 63n));
}

function written(write: (writer: BinaryWriter) => BinaryWriter): string {
  return Buffer.from(write(new BinaryWriter()).finish()).toString("hex");
}

const unsignedBoundaries = [
  0n,
  1n,
  127n,
  128n,
  (1n << 31n) - 1n,
  1n << 31n, // first value the unpatched writer truncated
  3_000_000_000n,
  4_102_444_800n, // 2100-01-01T00:00:00Z as Unix seconds
  (1n << 32n) - 1n, // last value the unpatched writer truncated
  1n << 32n,
  (1n << 53n) + 1n,
  (1n << 63n) - 1n,
];

const signedBoundaries = [
  ...unsignedBoundaries,
  -1n,
  -(1n << 31n),
  -(1n << 31n) - 1n,
  -(1n << 32n),
  -(1n << 63n),
];

describe("generated 64-bit varint writer", () => {
  for (const value of signedBoundaries) {
    it(`int64 ${value} matches the reference encoding and round-trips`, () => {
      const hex = written((writer) => writer.int64(value));
      assert.equal(hex, referenceVarint(value));
      const reader = new BinaryReader(Buffer.from(hex, "hex"));
      assert.equal(reader.int64(), value);
    });

    it(`sint64 ${value} matches the reference zigzag encoding and round-trips`, () => {
      const hex = written((writer) => writer.sint64(value));
      assert.equal(hex, referenceVarint(referenceZigZag(value)));
      const reader = new BinaryReader(Buffer.from(hex, "hex"));
      assert.equal(reader.sint64(), value);
    });
  }

  for (const value of [...unsignedBoundaries, 1n << 63n, U64]) {
    it(`uint64 ${value} matches the reference encoding and round-trips`, () => {
      const hex = written((writer) => writer.uint64(value));
      assert.equal(hex, referenceVarint(value));
      const reader = new BinaryReader(Buffer.from(hex, "hex"));
      assert.equal(reader.uint64(), value);
    });
  }

  it("keeps a post-2038 rotation expiry intact through MsgRotateKey", () => {
    const authorizationExpiresAtUnix = 4_102_444_800n;
    const encoded = MsgRotateKey.encode(MsgRotateKey.fromPartial({
      sender: "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf",
      authorizationExpiresAtUnix,
    })).finish();
    assert.equal(
      MsgRotateKey.decode(encoded).authorizationExpiresAtUnix,
      authorizationExpiresAtUnix,
    );
    assert.ok(
      Buffer.from(encoded).toString("hex").endsWith(
        `20${referenceVarint(authorizationExpiresAtUnix)}`,
      ),
    );
  });
});
