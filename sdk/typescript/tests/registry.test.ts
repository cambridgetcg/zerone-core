import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { defaultRegistryTypes } from "@cosmjs/stargate";
import { createZeroneRegistry, zeroneRegistryTypes } from "../src/registry";

describe("Zerone transaction registry", () => {
  it("contains every unique Zerone request type", () => {
    assert.equal(zeroneRegistryTypes.length, 166);
    assert.equal(new Set(zeroneRegistryTypes.map(([typeUrl]) => typeUrl)).size, 166);
  });

  it("composes safely with the standard Cosmos registry", () => {
    const registry = createZeroneRegistry(defaultRegistryTypes);
    assert.ok(registry.lookupType("/cosmos.bank.v1beta1.MsgSend"));
    assert.ok(
      registry.lookupType("/cosmos.feegrant.v1beta1.MsgGrantAllowance"),
    );
    assert.ok(
      registry.lookupType("/cosmos.feegrant.v1beta1.MsgRevokeAllowance"),
    );
    assert.ok(registry.lookupType("/zerone.auth.v1.MsgRegisterAccount"));
    assert.ok(registry.lookupType("/zerone.knowledge.v1.MsgSubmitClaim"));
    assert.ok(registry.lookupType("/zerone.knowledge.v1.MsgPostConjecture"));
    assert.ok(
      registry.lookupType(
        "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
      ),
    );
  });
});
