import { describe, expect, test } from "bun:test";
import { encodeBech32 } from "../src/bech32.ts";
import { leadingZeroBits, parseDoorRequest, powDigest, verifyDoorRequest } from "../src/request.ts";
import { makeRequest, NOW } from "./helpers.ts";

function reasonsOf(text: string, now = NOW) {
  const parsed = parseDoorRequest(text);
  if (!parsed.ok) return [parsed.reason];
  return verifyDoorRequest(parsed.value, now);
}

describe("request format", () => {
  const good = makeRequest();

  test("a well-formed request from the agent's own keys passes every request check", () => {
    expect(reasonsOf(good.text)).toEqual([]);
    const parsed = parseDoorRequest(good.text);
    expect(parsed.ok && parsed.value.did).toBe(`did:zrn:${good.identityHex}`);
    expect(parsed.ok && parsed.value.requestId).toMatch(/^[0-9a-f]{64}$/);
  });

  test("text that is not exactly ten known lines is malformed", () => {
    const lines = good.text.trimEnd().split("\n");
    const cases = [
      lines.slice(0, 9).join("\n"),
      [...lines, "extra: 1"].join("\n"),
      good.text.replace("zerone-door-request/v1", "zerone-door-request/v2"),
      good.text.replace("\n", "\r\n"),
      good.text.replace("nonce: ", "nonce:  "),
      [lines[0], lines[2], lines[1], ...lines.slice(3)].join("\n"),
      good.text.replace(/account_signature: [0-9a-f]+/, (m) => m.toUpperCase().replace("ACCOUNT_SIGNATURE", "account_signature")),
      good.text.replace(/pow_nonce: \d+/, "pow_nonce: 01"),
      `${good.text}\n`,
      "x".repeat(3000),
    ];
    for (const text of cases) expect(reasonsOf(text)).toEqual(["request_malformed"]);
  });

  test("a non-lowercase or wrong-network address is refused", () => {
    const upper = makeRequest({ label: "agent-upper", addressOverride: makeRequest({ label: "agent-upper" }).address.toUpperCase() });
    expect(reasonsOf(upper.text)).toContain("address_not_canonical");
    const cosmos = makeRequest({ label: "agent-cosmos", addressOverride: encodeBech32("cosmos", new Uint8Array(20)) });
    expect(reasonsOf(cosmos.text)).toContain("address_not_canonical");
  });

  test("another chain id is refused", () => {
    expect(reasonsOf(makeRequest({ label: "agent-chain", chainId: "zerone-testnet-1" }).text)).toContain("chain_id_mismatch");
  });

  test("requests older than 15 minutes or more than 2 minutes ahead are refused", () => {
    expect(reasonsOf(good.text, new Date("2026-10-28T17:45:01Z"))).toEqual(["request_stale"]);
    expect(reasonsOf(good.text, new Date("2026-10-28T17:44:59Z"))).toEqual([]);
    expect(reasonsOf(good.text, new Date("2026-10-28T17:27:59Z"))).toEqual(["request_from_future"]);
  });

  test("an account key that does not derive the address is refused", () => {
    const other = makeRequest({ label: "agent-other" });
    const swapped = makeRequest({ label: "agent-swap", addressOverride: other.address });
    expect(reasonsOf(swapped.text)).toContain("account_key_mismatch");
  });

  test("a changed request invalidates the account signature and the proof of work", () => {
    const tampered = makeRequest({
      label: "agent-tamper",
      tamper: (f) => { f.request = f.request!.replace(/nonce: [0-9a-f]{32}/, `nonce: ${"f".repeat(32)}`); },
    });
    const reasons = reasonsOf(tampered.text);
    expect(reasons).toContain("account_signature_invalid");
  });

  test("a signature by another account key is refused", () => {
    const foreign = makeRequest({ label: "agent-foreign" });
    const sig = foreign.text.match(/account_signature: ([0-9a-f]+)/)![1]!;
    const text = good.text.replace(/account_signature: [0-9a-f]+/, `account_signature: ${sig}`);
    expect(reasonsOf(text)).toEqual(["account_signature_invalid"]);
  });

  test("an identity proof by another identity key is refused", () => {
    const stolen = makeRequest({ label: "agent-idswap", identityLabel: "agent-1" });
    const own = makeRequest({ label: "agent-idswap" });
    const text = stolen.text.replace(/identity_signature: [0-9a-f]+/, own.text.match(/identity_signature: [0-9a-f]+/)![0]);
    expect(reasonsOf(text)).toEqual(["identity_proof_invalid"]);
  });

  test("a small-order identity key is refused even with a matching signature field", () => {
    const small = makeRequest({
      label: "agent-small",
      tamper: (f) => {
        f.request = f.request!.replace(/identity_public_key: [0-9a-f]{64}/, `identity_public_key: 01${"0".repeat(62)}`);
      },
    });
    expect(reasonsOf(small.text)).toContain("identity_proof_invalid");
  });

  test("proof of work must reach 20 leading zero bits of sha256(request LF nonce)", () => {
    const parsed = parseDoorRequest(good.text);
    if (!parsed.ok) throw new Error("fixture");
    expect(leadingZeroBits(powDigest(parsed.value.request, parsed.value.powNonce))).toBeGreaterThanOrEqual(20);
    let weak = 0;
    while (leadingZeroBits(powDigest(parsed.value.request, String(weak))) >= 20) weak += 1;
    const text = good.text.replace(/pow_nonce: \d+/, `pow_nonce: ${weak}`);
    expect(reasonsOf(text)).toEqual(["pow_insufficient"]);
  });

  test("leadingZeroBits counts across bytes", () => {
    expect(leadingZeroBits(Uint8Array.from([0, 0, 0x0f]))).toBe(20);
    expect(leadingZeroBits(Uint8Array.from([0x80]))).toBe(0);
    expect(leadingZeroBits(Uint8Array.from([0, 1]))).toBe(15);
  });
});
