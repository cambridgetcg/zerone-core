import assert from "node:assert/strict";
import { test } from "node:test";
import { GenesisState } from "../src/generated/zerone/knowledge/v1/genesis";

test("knowledge genesis preserves all pending reward fields without number rounding", () => {
  const original = GenesisState.fromPartial({
    survivalPendingRewards: [
      {
        claimId: "claim-1",
        factId: "fact-1",
        recipient: "historical-recipient",
        amount: "184467440737095516160000",
        category: "historical-category",
        partnershipId: "historical-partnership",
        deadline: 18446744073709551615n,
      },
      {
        claimId: "claim-2",
        factId: "fact-2",
        recipient: "other-recipient",
        amount: "+200000",
        category: "",
        deadline: 0n,
      },
    ],
  });
  const encoded = GenesisState.encode(original).finish();
  const decoded = GenesisState.decode(encoded);
  assert.deepEqual(decoded.survivalPendingRewards, original.survivalPendingRewards);
  assert.deepEqual(GenesisState.encode(decoded).finish(), encoded);
});

test("old knowledge genesis bytes do not fabricate pending rewards", () => {
  assert.deepEqual(GenesisState.decode(new Uint8Array()).survivalPendingRewards, []);
});
