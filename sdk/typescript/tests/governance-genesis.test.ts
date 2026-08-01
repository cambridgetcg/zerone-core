import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { GenesisState } from "../src/generated/zerone/gov/v1/genesis";

describe("custom governance emergency transition hold", () => {
  it("round-trips the bounded incident lineage without bigint loss", () => {
    const original = GenesisState.fromPartial({
      emergencyTransitionHold: {
        incidentId: "halt-first",
        activatedAtBlock: 9_007_199_254_740_993n,
        latestIncidentId: "halt-latest",
        incidentCount: 18_446_744_073_709_551_615n,
        incidentLineageSha256: Uint8Array.from(
          Array.from({ length: 32 }, (_, index) => index),
        ),
      },
    });

    const encoded = GenesisState.encode(original).finish();
    const decoded = GenesisState.decode(encoded);

    assert.deepEqual(
      decoded.emergencyTransitionHold,
      original.emergencyTransitionHold,
    );
    assert.deepEqual(GenesisState.encode(decoded).finish(), encoded);
  });
});
