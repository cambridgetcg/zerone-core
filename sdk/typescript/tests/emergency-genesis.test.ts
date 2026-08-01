import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { GenesisState } from "../src/generated/zerone/emergency/v1/genesis";

describe("emergency genesis anti-abuse state", () => {
  it("round-trips fields 7 through 12 without losing incident authority", () => {
    const original = GenesisState.fromPartial({
      status: "halted",
      guardianProposalCounts: [
        {
          guardian: "zrn1guardian000000000000000000000000000000",
          count: 9_007_199_254_740_993n,
        },
      ],
      epochProposalCount: 9_007_199_254_740_995n,
      lastProposalBlock: 18_446_744_073_709_551_614n,
      lastHaltEscalationBlock: 18_446_744_073_709_551_615n,
      quarantineReleaseBlock: 9_007_199_254_740_997n,
      recoveryAuthorization: {
        haltCeremonyId: "halt-1",
        authorizationCeremonyId: "recovery-auth-1",
        sdkGovProposalId: 9_007_199_254_740_999n,
        actionSha256: "11".repeat(32),
        recoveryManifestSha256: "22".repeat(32),
        authorizedAtBlock: 9_007_199_254_741_001n,
        upgradePlanSha256: "33".repeat(32),
        terminalAtBlock: 9_007_199_254_741_003n,
        outcome: "passed",
        authorizedSubmitter:
          "zrn100mxrvv5chhhrj0yd9y4q8354z4edm42mukf5r",
        actionType: "software_upgrade",
        generation: 9_007_199_254_741_005n,
      },
    });

    const encoded = GenesisState.encode(original).finish();
    const decoded = GenesisState.decode(encoded);

    assert.deepEqual(decoded.guardianProposalCounts, original.guardianProposalCounts);
    assert.equal(decoded.epochProposalCount, original.epochProposalCount);
    assert.equal(decoded.lastProposalBlock, original.lastProposalBlock);
    assert.equal(
      decoded.lastHaltEscalationBlock,
      original.lastHaltEscalationBlock,
    );
    assert.equal(
      decoded.quarantineReleaseBlock,
      original.quarantineReleaseBlock,
    );
    assert.deepEqual(
      decoded.recoveryAuthorization,
      original.recoveryAuthorization,
    );
    assert.deepEqual(GenesisState.encode(decoded).finish(), encoded);
  });
});
