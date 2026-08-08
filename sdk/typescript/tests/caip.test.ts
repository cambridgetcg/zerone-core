import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { toBech32 } from "@cosmjs/encoding";
import {
  CaipError,
  asExistingZeroneDid,
  cosmosChainId,
  defineZeroneNetwork,
  formatCaip10,
  formatCaip2,
  parseCaip2,
  parseCaip10,
  parseCosmosChainId,
  zeroneAccountId,
  type ZeroneNetwork,
} from "../src/caip";

const MAINNET_ADDRESS = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";

function assertCaipError(
  operation: () => unknown,
  code: InstanceType<typeof CaipError>["code"],
): void {
  assert.throws(
    operation,
    (error: unknown) => error instanceof CaipError && error.code === code,
  );
}

describe("CAIP syntax", () => {
  it("round-trips case-sensitive CAIP-2 and CAIP-10 identifiers", () => {
    assert.equal(
      parseCaip2("cosmos:Binance-Chain-Tigris").reference,
      "Binance-Chain-Tigris",
    );
    const chain = parseCaip2("cosmos:cosmoshub-3");
    const account = formatCaip10(chain.id, "cosmos1t2uflqwqe0fsj0shcfkrvpukewcw40yjj6hdc0");
    assert.equal(parseCaip10(account).chainId, chain.id);
  });

  it("enforces the exact CAIP component length boundaries", () => {
    assert.equal(formatCaip2("abc", "x"), "abc:x");
    assert.equal(formatCaip2("abcdefgh", "x".repeat(32)), `abcdefgh:${"x".repeat(32)}`);
    assertCaipError(() => formatCaip2("ab", "x"), "INVALID_CAIP2");
    assertCaipError(() => formatCaip2("abcdefghi", "x"), "INVALID_CAIP2");
    assertCaipError(() => formatCaip2("abc", ""), "INVALID_CAIP2");
    assertCaipError(() => formatCaip2("abc", "x".repeat(33)), "INVALID_CAIP2");

    const chain = formatCaip2("cosmos", "zerone-1");
    assert.equal(formatCaip10(chain, "a".repeat(128)), `${chain}:${"a".repeat(128)}`);
    assertCaipError(
      () => formatCaip10(chain, "a".repeat(129)),
      "INVALID_CAIP10",
    );
  });

  it("accepts only the generic CAIP-10 account character set", () => {
    const chain = formatCaip2("cosmos", "zerone-1");
    assert.equal(formatCaip10(chain, "a"), `${chain}:a`);
    assert.equal(formatCaip10(chain, "A0-.%20"), `${chain}:A0-.%20`);
    for (const forbidden of ["_", ":", "/", "\\"]) {
      assertCaipError(
        () => formatCaip10(chain, `account${forbidden}part`),
        "INVALID_CAIP10",
      );
    }
  });

  it("rejects trailing newlines and forbidden account characters", () => {
    assertCaipError(() => parseCaip2("cosmos:zerone-1\n"), "INVALID_CAIP2");
    assertCaipError(
      () => parseCaip10("cosmos:zerone-1:did:zrn:abc"),
      "INVALID_CAIP10",
    );
    assertCaipError(
      () => parseCaip10("cosmos:zerone-1:a/b"),
      "INVALID_CAIP10",
    );
  });
});

describe("Cosmos chain references", () => {
  it("uses direct references at the profile boundaries", () => {
    assert.equal(cosmosChainId("zerone-1"), "cosmos:zerone-1");
    assert.equal(cosmosChainId("x"), "cosmos:x");
    assert.equal(cosmosChainId("x".repeat(32)), `cosmos:${"x".repeat(32)}`);
    assert.equal(cosmosChainId("hashed"), "cosmos:hashed");
    assert.equal(cosmosChainId("hash-"), "cosmos:hash-");
  });

  it("hashes reserved, long, and non-ASCII Tendermint chain IDs", () => {
    assert.equal(cosmosChainId("hashed-"), "cosmos:hashed-c904589232422def");
    assert.equal(
      cosmosChainId("123456789012345678901234567890123456789012345678"),
      "cosmos:hashed-0204c92a0388779d",
    );
    assert.equal(cosmosChainId("wonderland🧝‍♂️"), "cosmos:hashed-843d2fc87f40eeb9");
    assert.equal(cosmosChainId(" "), "cosmos:hashed-36a9e7f1c95b82ff");
    assert.match(cosmosChainId("x".repeat(33)), /^cosmos:hashed-[0-9a-f]{16}$/);
    assert.match(cosmosChainId("chain_id"), /^cosmos:hashed-[0-9a-f]{16}$/);
    assert.match(cosmosChainId("hashed-anything"), /^cosmos:hashed-[0-9a-f]{16}$/);
  });

  it("distinguishes generic CAIP syntax from the Cosmos namespace profile", () => {
    assert.equal(parseCaip2("cosmos:foo_bar").reference, "foo_bar");
    assertCaipError(
      () => parseCosmosChainId("cosmos:foo_bar"),
      "INVALID_COSMOS_REFERENCE",
    );
    assertCaipError(() => parseCosmosChainId("eip155:1"), "NOT_COSMOS");
    assertCaipError(() => cosmosChainId(""), "INVALID_COSMOS_REFERENCE");
    assert.equal(
      parseCosmosChainId("cosmos:hashed-0123456789abcdef"),
      "cosmos:hashed-0123456789abcdef",
    );
    assertCaipError(
      () => parseCosmosChainId("cosmos:hashed-123"),
      "INVALID_COSMOS_REFERENCE",
    );
    assertCaipError(
      () => parseCosmosChainId("cosmos:hashed-0123456789ABCDEf"),
      "INVALID_COSMOS_REFERENCE",
    );
  });

  it("rejects ill-formed UTF-16 instead of hashing replacement characters", () => {
    assertCaipError(() => cosmosChainId("\ud800"), "INVALID_COSMOS_REFERENCE");
    assertCaipError(() => cosmosChainId("\udc00"), "INVALID_COSMOS_REFERENCE");
    assertCaipError(() => cosmosChainId("a\ud800b"), "INVALID_COSMOS_REFERENCE");
  });
});

describe("Zerone identity references", () => {
  it("formats a checksum-validated 20-byte zrn account as CAIP-10", () => {
    const network = defineZeroneNetwork("zerone-1");
    assert.equal(network.chainId, "cosmos:zerone-1");
    assert.equal(
      zeroneAccountId(network, MAINNET_ADDRESS),
      `cosmos:zerone-1:${MAINNET_ADDRESS}`,
    );
  });

  it("requires an explicitly declared Zerone network", () => {
    if (false) {
      // @ts-expect-error A generic Cosmos chain is not a trusted Zerone network.
      zeroneAccountId(cosmosChainId("cosmoshub-4"), MAINNET_ADDRESS);
    }
  });

  it("rejects a forged or internally inconsistent network descriptor", () => {
    const network = defineZeroneNetwork("zerone-1");
    const tampered = {
      ...network,
      rawChainId: "cosmoshub-4",
    } as unknown as ZeroneNetwork;
    assertCaipError(
      () => zeroneAccountId(tampered, MAINNET_ADDRESS),
      "INVALID_COSMOS_REFERENCE",
    );
  });

  it("rejects malformed, noncanonical, wrong-prefix, and wrong-length addresses", () => {
    const network = defineZeroneNetwork("zerone-1");
    assertCaipError(
      () => zeroneAccountId(network, `${MAINNET_ADDRESS.slice(0, -1)}q`),
      "INVALID_BECH32",
    );
    assertCaipError(
      () => zeroneAccountId(network, MAINNET_ADDRESS.toUpperCase()),
      "NON_CANONICAL_ADDRESS",
    );
    assertCaipError(
      () =>
        zeroneAccountId(
          network,
          "zrnvaloper1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqdkemrh",
        ),
      "WRONG_HRP",
    );
    assertCaipError(
      () => zeroneAccountId(network, toBech32("zrn", new Uint8Array(32))),
      "WRONG_ACCOUNT_LENGTH",
    );
    assertCaipError(
      () => zeroneAccountId(network, toBech32("zrn", new Uint8Array(64), 200)),
      "INVALID_BECH32",
    );
  });

  it("keeps did:zrn separate from the CAIP account address", () => {
    const did = asExistingZeroneDid("did:zrn:abcdef0123456789abcdef0123456789");
    assert.equal(did, "did:zrn:abcdef0123456789abcdef0123456789");
    assert.equal(
      asExistingZeroneDid(
        "did:zrn:ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
      ),
      "did:zrn:ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789",
    );
    assertCaipError(() => asExistingZeroneDid("did:zrn:short"), "INVALID_DID_ZRN");
    assertCaipError(
      () => asExistingZeroneDid(`did:zrn:${"g".repeat(32)}`),
      "INVALID_DID_ZRN",
    );
    const encodedDid = `did%3Azrn%3A${"a".repeat(32)}`;
    assert.equal(
      formatCaip10(formatCaip2("cosmos", "zerone-1"), encodedDid),
      `cosmos:zerone-1:${encodedDid}`,
    );
    assertCaipError(() => asExistingZeroneDid(encodedDid), "INVALID_DID_ZRN");
    assertCaipError(
      () =>
        zeroneAccountId(
          defineZeroneNetwork("zerone-1"),
          "did:zrn:abcdef0123456789abcdef0123456789",
        ),
      "INVALID_BECH32",
    );
  });
});
