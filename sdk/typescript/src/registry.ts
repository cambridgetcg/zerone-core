import { Registry, type GeneratedType } from "@cosmjs/proto-signing";
import { registry as alignmentRegistry } from "./generated/zerone/alignment/v1/tx.registry";
import { registry as authRegistry } from "./generated/zerone/auth/v1/tx.registry";
import { registry as captureChallengeRegistry } from "./generated/zerone/capture_challenge/v1/tx.registry";
import { registry as captureDefenseRegistry } from "./generated/zerone/capture_defense/v1/tx.registry";
import { registry as claimingPotRegistry } from "./generated/zerone/claiming_pot/v1/tx.registry";
import { registry as counterexamplesRegistry } from "./generated/zerone/counterexamples/v1/tx.registry";
import { registry as creedRegistry } from "./generated/zerone/creed/v1/tx.registry";
import { registry as emergencyRegistry } from "./generated/zerone/emergency/v1/tx.registry";
import { registry as governanceRegistry } from "./generated/zerone/gov/v1/tx.registry";
import { registry as homeRegistry } from "./generated/zerone/home/v1/tx.registry";
import { registry as ibcRateLimitRegistry } from "./generated/zerone/ibcratelimit/v1/tx.registry";
import { registry as knowledgeRegistry } from "./generated/zerone/knowledge/v1/tx.registry";
import { registry as liquidityPoolRegistry } from "./generated/zerone/liquiditypool/v1/tx.registry";
import { registry as ontologyRegistry } from "./generated/zerone/ontology/v1/tx.registry";
import { registry as qualificationRegistry } from "./generated/zerone/qualification/v1/tx.registry";
import { registry as scheduleRegistry } from "./generated/zerone/schedule/v2/tx.registry";
import { registry as sponsorshipRegistry } from "./generated/zerone/sponsorship/v1/tx.registry";
import { registry as stakingRegistry } from "./generated/zerone/staking/v1/tx.registry";
import { registry as substrateBridgeRegistry } from "./generated/zerone/substrate_bridge/v1/tx.registry";
import { registry as tokensRegistry } from "./generated/zerone/tokens/v1/tx.registry";
import { registry as vestingRewardsRegistry } from "./generated/zerone/vesting_rewards/v1/tx.registry";

export const zeroneRegistryTypes: ReadonlyArray<[string, GeneratedType]> = [
  ...alignmentRegistry,
  ...authRegistry,
  ...captureChallengeRegistry,
  ...captureDefenseRegistry,
  ...claimingPotRegistry,
  ...counterexamplesRegistry,
  ...creedRegistry,
  ...emergencyRegistry,
  ...governanceRegistry,
  ...homeRegistry,
  ...ibcRateLimitRegistry,
  ...knowledgeRegistry,
  ...liquidityPoolRegistry,
  ...ontologyRegistry,
  ...qualificationRegistry,
  ...scheduleRegistry,
  ...sponsorshipRegistry,
  ...stakingRegistry,
  ...substrateBridgeRegistry,
  ...tokensRegistry,
  ...vestingRewardsRegistry,
];

export function registerZeroneMessages(registry: Registry): Registry {
  for (const [typeUrl, codec] of zeroneRegistryTypes) {
    registry.register(typeUrl, codec);
  }
  return registry;
}

export function createZeroneRegistry(
  baseTypes: ReadonlyArray<[string, GeneratedType]>,
): Registry {
  return registerZeroneMessages(new Registry(baseTypes));
}
