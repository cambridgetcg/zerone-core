import {
  registry,
  registry10,
  registry11,
  registry12,
  registry13,
  registry14,
  registry15,
  registry16,
  registry17,
  registry18,
  registry19,
  registry2,
  registry20,
  registry3,
  registry4,
  registry5,
  registry6,
  registry7,
  registry8,
  registry9
} from "./chunk-PQV3XR6M.js";

// src/registry.ts
import { Registry } from "@cosmjs/proto-signing";
var zeroneRegistryTypes = [
  ...registry,
  ...registry2,
  ...registry3,
  ...registry4,
  ...registry5,
  ...registry6,
  ...registry7,
  ...registry8,
  ...registry9,
  ...registry10,
  ...registry11,
  ...registry12,
  ...registry13,
  ...registry14,
  ...registry15,
  ...registry16,
  ...registry17,
  ...registry18,
  ...registry19,
  ...registry20
];
function registerZeroneMessages(registry21) {
  for (const [typeUrl, codec] of zeroneRegistryTypes) {
    registry21.register(typeUrl, codec);
  }
  return registry21;
}
function createZeroneRegistry(baseTypes) {
  return registerZeroneMessages(new Registry(baseTypes));
}

export {
  zeroneRegistryTypes,
  registerZeroneMessages,
  createZeroneRegistry
};
