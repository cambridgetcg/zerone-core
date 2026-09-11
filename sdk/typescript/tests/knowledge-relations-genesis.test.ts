import assert from "node:assert/strict";
import { test } from "node:test";
import { GenesisState } from "../src/generated/zerone/knowledge/v1/genesis";
import {
  InferenceType,
  RelationType,
} from "../src/generated/zerone/knowledge/v1/types";

// The SDK exposes binary codecs and fromPartial, not a protobuf-JSON parser.
// This tests the JSON-safe partial-object transport used by a caller, with
// uint64 values represented as decimal strings rather than JS numbers.
function jsonPartial(message: GenesisState) {
  return JSON.parse(
    JSON.stringify(message, (_key, value) =>
      typeof value === "bigint" ? value.toString() : value,
    ),
  );
}

test("knowledge genesis distinguishes absent and explicitly empty relation inventories", () => {
  const absent = GenesisState.fromPartial({});
  const empty = GenesisState.fromPartial({ factRelationState: { relations: [] } });
  const absentWire = GenesisState.encode(absent).finish();
  const emptyWire = GenesisState.encode(empty).finish();

  assert.equal(GenesisState.decode(new Uint8Array()).factRelationState, undefined);
  assert.equal(GenesisState.decode(absentWire).factRelationState, undefined);
  assert.deepEqual(GenesisState.decode(emptyWire).factRelationState, { relations: [] });
  assert.notDeepEqual(emptyWire, absentWire);
  // Field 68, wire type 2, zero-length nested message has explicit presence.
  assert.deepEqual(
    GenesisState.decode(Uint8Array.from([0xa2, 0x04, 0x00])).factRelationState,
    { relations: [] },
  );

  const absentJson = jsonPartial(absent);
  const emptyJson = jsonPartial(empty);
  assert.equal(Object.hasOwn(absentJson, "factRelationState"), false);
  assert.deepEqual(emptyJson.factRelationState, { relations: [] });
  assert.equal(GenesisState.fromPartial(absentJson).factRelationState, undefined);
  assert.deepEqual(GenesisState.fromPartial(emptyJson).factRelationState, { relations: [] });
});

test("knowledge relation genesis preserves full metadata through binary and JSON-safe partials", () => {
  const original = GenesisState.fromPartial({
    factRelationState: {
      relations: [{
        sourceFactId: "fact-source",
        targetFactId: "fact-target",
        relation: RelationType.RELATION_TYPE_SUPPORTS,
        createdAtBlock: 18_446_744_073_709_551_615n,
        creator: "historical-creator",
        inference: InferenceType.INFERENCE_TYPE_DEDUCTIVE,
        inferenceStrengthBps: 654_321n,
        methodId: "historical-method",
      }],
    },
  });
  const wire = GenesisState.encode(original).finish();
  const decoded = GenesisState.decode(wire);
  assert.deepEqual(decoded.factRelationState, original.factRelationState);
  assert.deepEqual(GenesisState.encode(decoded).finish(), wire);

  const json = jsonPartial(decoded);
  assert.equal(json.factRelationState.relations[0].createdAtBlock, "18446744073709551615");
  assert.equal(json.factRelationState.relations[0].inferenceStrengthBps, "654321");
  const restored = GenesisState.fromPartial(json);
  assert.deepEqual(restored.factRelationState, original.factRelationState);
  assert.deepEqual(GenesisState.encode(restored).finish(), wire);
});
