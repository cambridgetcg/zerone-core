//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * VestingSchedule represents a truth-linked reward vesting record.
 * @name VestingSchedule
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.VestingSchedule
 */
export interface VestingSchedule {
  /**
   * unique vesting ID
   */
  id: string;
  /**
   * the claim this reward is for
   */
  claimId: string;
  /**
   * the fact (once claim is accepted)
   */
  factId: string;
  /**
   * bech32 address of reward recipient
   */
  recipient: string;
  /**
   * total reward amount in uzrn (bigint as string)
   */
  totalAmount: string;
  /**
   * amount already released (bigint as string)
   */
  releasedAmount: string;
  /**
   * amount currently claimable (bigint as string)
   */
  claimableAmount: string;
  /**
   * permanent challenge reserve (bigint as string)
   */
  reserveAmount: string;
  /**
   * epistemic category: "axiomatic", "formal_proof", "on_chain", etc.
   */
  category: string;
  /**
   * reward origin: "block_production", "verification", "api_call", etc.
   */
  source: string;
  /**
   * "vesting", "paused", "completed", "falsified", "abandoned"
   */
  status: string;
  /**
   * when claim was accepted
   */
  acceptedAtBlock: bigint;
  /**
   * when cliff period ends
   */
  cliffEndsAtBlock: bigint;
  /**
   * last block rewards were claimed
   */
  lastClaimBlock: bigint;
  /**
   * total blocks spent paused
   */
  totalPausedBlocks: bigint;
  /**
   * block when paused (0 if not paused)
   */
  pausedAtBlock: bigint;
  /**
   * successful defenses (accelerates)
   */
  defenseCount: number;
  /**
   * independent replications (accelerates)
   */
  replicationCount: number;
  /**
   * block height when created
   */
  createdAt: bigint;
  /**
   * block height when last updated
   */
  updatedAt: bigint;
  /**
   * cross-domain corroborations (accelerates)
   */
  corroborationCount: number;
  /**
   * citations by other accepted facts (accelerates)
   */
  citationCount: number;
}
/**
 * CategoryConfig defines release curve parameters for a vesting category.
 * @name CategoryConfig
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.CategoryConfig
 */
export interface CategoryConfig {
  /**
   * epistemic category name
   */
  category: string;
  /**
   * blocks until half of reward is released
   */
  halfLifeBlocks: bigint;
  /**
   * blocks before any release begins
   */
  cliffBlocks: bigint;
  /**
   * basis points (950000 = 95%, rest is permanent reserve)
   */
  maxRelease: bigint;
}
/**
 * RewardRouting represents a routed reward with full revenue split applied.
 * @name RewardRouting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.RewardRouting
 */
export interface RewardRouting {
  /**
   * reward source
   */
  source: string;
  /**
   * original reward amount (bigint as string)
   */
  originalAmount: string;
  /**
   * contributor's share (bigint as string)
   */
  contributorShare: string;
  /**
   * protocol share (bigint as string)
   */
  protocolShare: string;
  /**
   * research fund share (bigint as string)
   */
  researchShare: string;
  /**
   * development fund amount (bigint as string)
   */
  developmentAmount: string;
  /**
   * bech32 recipient address
   */
  recipient: string;
  /**
   * related fact
   */
  factId: string;
  /**
   * block height
   */
  blockNumber: bigint;
  /**
   * founder's operational share (bigint as string)
   */
  founderShare: string;
  /**
   * protocol sub-split: citations (bigint as string)
   */
  citationPool: string;
  /**
   * protocol sub-split: verification (bigint as string)
   */
  verificationPool: string;
  /**
   * protocol sub-split: treasury (bigint as string)
   */
  treasuryShare: string;
}
/**
 * BlockRewardDistribution records block reward distribution for a specific block.
 * @name BlockRewardDistribution
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.BlockRewardDistribution
 */
export interface BlockRewardDistribution {
  /**
   * block number
   */
  blockHeight: bigint;
  /**
   * reward to block producer (bigint as string)
   */
  producerReward: string;
  /**
   * research fund share (bigint as string)
   */
  researchShare: string;
  /**
   * total new tokens (bigint as string)
   */
  totalMinted: string;
  /**
   * active validators
   */
  validatorCount: number;
  /**
   * remaining fund balance (bigint as string)
   */
  fundBalanceAfter: string;
  /**
   * founder's operational share (bigint as string)
   */
  founderShare: string;
  /**
   * development fund amount (bigint as string)
   */
  developmentAmount: string;
  /**
   * protocol share (bigint as string)
   */
  protocolShare: string;
}
/**
 * ClawbackRecord records a reward clawback due to falsification.
 * @name ClawbackRecord
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.ClawbackRecord
 */
export interface ClawbackRecord {
  /**
   * unique record ID
   */
  id: string;
  /**
   * vesting schedule being clawed back
   */
  vestingId: string;
  /**
   * amount clawed from already-released (bigint as string)
   */
  releasedClawback: string;
  /**
   * unvested amount forfeited (bigint as string)
   */
  unvestedForfeited: string;
  /**
   * reserve forfeited (bigint as string)
   */
  reserveForfeited: string;
  /**
   * amount given to challenger (bigint as string)
   */
  challengerReward: string;
  /**
   * when clawback occurred
   */
  blockHeight: bigint;
}
function createBaseVestingSchedule(): VestingSchedule {
  return {
    id: "",
    claimId: "",
    factId: "",
    recipient: "",
    totalAmount: "",
    releasedAmount: "",
    claimableAmount: "",
    reserveAmount: "",
    category: "",
    source: "",
    status: "",
    acceptedAtBlock: BigInt(0),
    cliffEndsAtBlock: BigInt(0),
    lastClaimBlock: BigInt(0),
    totalPausedBlocks: BigInt(0),
    pausedAtBlock: BigInt(0),
    defenseCount: 0,
    replicationCount: 0,
    createdAt: BigInt(0),
    updatedAt: BigInt(0),
    corroborationCount: 0,
    citationCount: 0
  };
}
/**
 * VestingSchedule represents a truth-linked reward vesting record.
 * @name VestingSchedule
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.VestingSchedule
 */
export const VestingSchedule = {
  typeUrl: "/zerone.vesting_rewards.v1.VestingSchedule",
  encode(message: VestingSchedule, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.claimId !== "") {
      writer.uint32(18).string(message.claimId);
    }
    if (message.factId !== "") {
      writer.uint32(26).string(message.factId);
    }
    if (message.recipient !== "") {
      writer.uint32(34).string(message.recipient);
    }
    if (message.totalAmount !== "") {
      writer.uint32(42).string(message.totalAmount);
    }
    if (message.releasedAmount !== "") {
      writer.uint32(50).string(message.releasedAmount);
    }
    if (message.claimableAmount !== "") {
      writer.uint32(58).string(message.claimableAmount);
    }
    if (message.reserveAmount !== "") {
      writer.uint32(66).string(message.reserveAmount);
    }
    if (message.category !== "") {
      writer.uint32(74).string(message.category);
    }
    if (message.source !== "") {
      writer.uint32(82).string(message.source);
    }
    if (message.status !== "") {
      writer.uint32(90).string(message.status);
    }
    if (message.acceptedAtBlock !== BigInt(0)) {
      writer.uint32(96).uint64(message.acceptedAtBlock);
    }
    if (message.cliffEndsAtBlock !== BigInt(0)) {
      writer.uint32(104).uint64(message.cliffEndsAtBlock);
    }
    if (message.lastClaimBlock !== BigInt(0)) {
      writer.uint32(112).uint64(message.lastClaimBlock);
    }
    if (message.totalPausedBlocks !== BigInt(0)) {
      writer.uint32(120).uint64(message.totalPausedBlocks);
    }
    if (message.pausedAtBlock !== BigInt(0)) {
      writer.uint32(128).uint64(message.pausedAtBlock);
    }
    if (message.defenseCount !== 0) {
      writer.uint32(136).uint32(message.defenseCount);
    }
    if (message.replicationCount !== 0) {
      writer.uint32(144).uint32(message.replicationCount);
    }
    if (message.createdAt !== BigInt(0)) {
      writer.uint32(152).uint64(message.createdAt);
    }
    if (message.updatedAt !== BigInt(0)) {
      writer.uint32(160).uint64(message.updatedAt);
    }
    if (message.corroborationCount !== 0) {
      writer.uint32(168).uint32(message.corroborationCount);
    }
    if (message.citationCount !== 0) {
      writer.uint32(176).uint32(message.citationCount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): VestingSchedule {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVestingSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.claimId = reader.string();
          break;
        case 3:
          message.factId = reader.string();
          break;
        case 4:
          message.recipient = reader.string();
          break;
        case 5:
          message.totalAmount = reader.string();
          break;
        case 6:
          message.releasedAmount = reader.string();
          break;
        case 7:
          message.claimableAmount = reader.string();
          break;
        case 8:
          message.reserveAmount = reader.string();
          break;
        case 9:
          message.category = reader.string();
          break;
        case 10:
          message.source = reader.string();
          break;
        case 11:
          message.status = reader.string();
          break;
        case 12:
          message.acceptedAtBlock = reader.uint64();
          break;
        case 13:
          message.cliffEndsAtBlock = reader.uint64();
          break;
        case 14:
          message.lastClaimBlock = reader.uint64();
          break;
        case 15:
          message.totalPausedBlocks = reader.uint64();
          break;
        case 16:
          message.pausedAtBlock = reader.uint64();
          break;
        case 17:
          message.defenseCount = reader.uint32();
          break;
        case 18:
          message.replicationCount = reader.uint32();
          break;
        case 19:
          message.createdAt = reader.uint64();
          break;
        case 20:
          message.updatedAt = reader.uint64();
          break;
        case 21:
          message.corroborationCount = reader.uint32();
          break;
        case 22:
          message.citationCount = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<VestingSchedule>): VestingSchedule {
    const message = createBaseVestingSchedule();
    message.id = object.id ?? "";
    message.claimId = object.claimId ?? "";
    message.factId = object.factId ?? "";
    message.recipient = object.recipient ?? "";
    message.totalAmount = object.totalAmount ?? "";
    message.releasedAmount = object.releasedAmount ?? "";
    message.claimableAmount = object.claimableAmount ?? "";
    message.reserveAmount = object.reserveAmount ?? "";
    message.category = object.category ?? "";
    message.source = object.source ?? "";
    message.status = object.status ?? "";
    message.acceptedAtBlock = object.acceptedAtBlock !== undefined && object.acceptedAtBlock !== null ? BigInt(object.acceptedAtBlock.toString()) : BigInt(0);
    message.cliffEndsAtBlock = object.cliffEndsAtBlock !== undefined && object.cliffEndsAtBlock !== null ? BigInt(object.cliffEndsAtBlock.toString()) : BigInt(0);
    message.lastClaimBlock = object.lastClaimBlock !== undefined && object.lastClaimBlock !== null ? BigInt(object.lastClaimBlock.toString()) : BigInt(0);
    message.totalPausedBlocks = object.totalPausedBlocks !== undefined && object.totalPausedBlocks !== null ? BigInt(object.totalPausedBlocks.toString()) : BigInt(0);
    message.pausedAtBlock = object.pausedAtBlock !== undefined && object.pausedAtBlock !== null ? BigInt(object.pausedAtBlock.toString()) : BigInt(0);
    message.defenseCount = object.defenseCount ?? 0;
    message.replicationCount = object.replicationCount ?? 0;
    message.createdAt = object.createdAt !== undefined && object.createdAt !== null ? BigInt(object.createdAt.toString()) : BigInt(0);
    message.updatedAt = object.updatedAt !== undefined && object.updatedAt !== null ? BigInt(object.updatedAt.toString()) : BigInt(0);
    message.corroborationCount = object.corroborationCount ?? 0;
    message.citationCount = object.citationCount ?? 0;
    return message;
  }
};
function createBaseCategoryConfig(): CategoryConfig {
  return {
    category: "",
    halfLifeBlocks: BigInt(0),
    cliffBlocks: BigInt(0),
    maxRelease: BigInt(0)
  };
}
/**
 * CategoryConfig defines release curve parameters for a vesting category.
 * @name CategoryConfig
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.CategoryConfig
 */
export const CategoryConfig = {
  typeUrl: "/zerone.vesting_rewards.v1.CategoryConfig",
  encode(message: CategoryConfig, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.category !== "") {
      writer.uint32(10).string(message.category);
    }
    if (message.halfLifeBlocks !== BigInt(0)) {
      writer.uint32(16).uint64(message.halfLifeBlocks);
    }
    if (message.cliffBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.cliffBlocks);
    }
    if (message.maxRelease !== BigInt(0)) {
      writer.uint32(32).uint64(message.maxRelease);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CategoryConfig {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCategoryConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.category = reader.string();
          break;
        case 2:
          message.halfLifeBlocks = reader.uint64();
          break;
        case 3:
          message.cliffBlocks = reader.uint64();
          break;
        case 4:
          message.maxRelease = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CategoryConfig>): CategoryConfig {
    const message = createBaseCategoryConfig();
    message.category = object.category ?? "";
    message.halfLifeBlocks = object.halfLifeBlocks !== undefined && object.halfLifeBlocks !== null ? BigInt(object.halfLifeBlocks.toString()) : BigInt(0);
    message.cliffBlocks = object.cliffBlocks !== undefined && object.cliffBlocks !== null ? BigInt(object.cliffBlocks.toString()) : BigInt(0);
    message.maxRelease = object.maxRelease !== undefined && object.maxRelease !== null ? BigInt(object.maxRelease.toString()) : BigInt(0);
    return message;
  }
};
function createBaseRewardRouting(): RewardRouting {
  return {
    source: "",
    originalAmount: "",
    contributorShare: "",
    protocolShare: "",
    researchShare: "",
    developmentAmount: "",
    recipient: "",
    factId: "",
    blockNumber: BigInt(0),
    founderShare: "",
    citationPool: "",
    verificationPool: "",
    treasuryShare: ""
  };
}
/**
 * RewardRouting represents a routed reward with full revenue split applied.
 * @name RewardRouting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.RewardRouting
 */
export const RewardRouting = {
  typeUrl: "/zerone.vesting_rewards.v1.RewardRouting",
  encode(message: RewardRouting, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.source !== "") {
      writer.uint32(10).string(message.source);
    }
    if (message.originalAmount !== "") {
      writer.uint32(18).string(message.originalAmount);
    }
    if (message.contributorShare !== "") {
      writer.uint32(26).string(message.contributorShare);
    }
    if (message.protocolShare !== "") {
      writer.uint32(34).string(message.protocolShare);
    }
    if (message.researchShare !== "") {
      writer.uint32(42).string(message.researchShare);
    }
    if (message.developmentAmount !== "") {
      writer.uint32(50).string(message.developmentAmount);
    }
    if (message.recipient !== "") {
      writer.uint32(58).string(message.recipient);
    }
    if (message.factId !== "") {
      writer.uint32(66).string(message.factId);
    }
    if (message.blockNumber !== BigInt(0)) {
      writer.uint32(72).uint64(message.blockNumber);
    }
    if (message.founderShare !== "") {
      writer.uint32(82).string(message.founderShare);
    }
    if (message.citationPool !== "") {
      writer.uint32(90).string(message.citationPool);
    }
    if (message.verificationPool !== "") {
      writer.uint32(98).string(message.verificationPool);
    }
    if (message.treasuryShare !== "") {
      writer.uint32(106).string(message.treasuryShare);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): RewardRouting {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRewardRouting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.source = reader.string();
          break;
        case 2:
          message.originalAmount = reader.string();
          break;
        case 3:
          message.contributorShare = reader.string();
          break;
        case 4:
          message.protocolShare = reader.string();
          break;
        case 5:
          message.researchShare = reader.string();
          break;
        case 6:
          message.developmentAmount = reader.string();
          break;
        case 7:
          message.recipient = reader.string();
          break;
        case 8:
          message.factId = reader.string();
          break;
        case 9:
          message.blockNumber = reader.uint64();
          break;
        case 10:
          message.founderShare = reader.string();
          break;
        case 11:
          message.citationPool = reader.string();
          break;
        case 12:
          message.verificationPool = reader.string();
          break;
        case 13:
          message.treasuryShare = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<RewardRouting>): RewardRouting {
    const message = createBaseRewardRouting();
    message.source = object.source ?? "";
    message.originalAmount = object.originalAmount ?? "";
    message.contributorShare = object.contributorShare ?? "";
    message.protocolShare = object.protocolShare ?? "";
    message.researchShare = object.researchShare ?? "";
    message.developmentAmount = object.developmentAmount ?? "";
    message.recipient = object.recipient ?? "";
    message.factId = object.factId ?? "";
    message.blockNumber = object.blockNumber !== undefined && object.blockNumber !== null ? BigInt(object.blockNumber.toString()) : BigInt(0);
    message.founderShare = object.founderShare ?? "";
    message.citationPool = object.citationPool ?? "";
    message.verificationPool = object.verificationPool ?? "";
    message.treasuryShare = object.treasuryShare ?? "";
    return message;
  }
};
function createBaseBlockRewardDistribution(): BlockRewardDistribution {
  return {
    blockHeight: BigInt(0),
    producerReward: "",
    researchShare: "",
    totalMinted: "",
    validatorCount: 0,
    fundBalanceAfter: "",
    founderShare: "",
    developmentAmount: "",
    protocolShare: ""
  };
}
/**
 * BlockRewardDistribution records block reward distribution for a specific block.
 * @name BlockRewardDistribution
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.BlockRewardDistribution
 */
export const BlockRewardDistribution = {
  typeUrl: "/zerone.vesting_rewards.v1.BlockRewardDistribution",
  encode(message: BlockRewardDistribution, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.blockHeight !== BigInt(0)) {
      writer.uint32(8).uint64(message.blockHeight);
    }
    if (message.producerReward !== "") {
      writer.uint32(18).string(message.producerReward);
    }
    if (message.researchShare !== "") {
      writer.uint32(26).string(message.researchShare);
    }
    if (message.totalMinted !== "") {
      writer.uint32(34).string(message.totalMinted);
    }
    if (message.validatorCount !== 0) {
      writer.uint32(40).uint32(message.validatorCount);
    }
    if (message.fundBalanceAfter !== "") {
      writer.uint32(50).string(message.fundBalanceAfter);
    }
    if (message.founderShare !== "") {
      writer.uint32(58).string(message.founderShare);
    }
    if (message.developmentAmount !== "") {
      writer.uint32(66).string(message.developmentAmount);
    }
    if (message.protocolShare !== "") {
      writer.uint32(74).string(message.protocolShare);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): BlockRewardDistribution {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseBlockRewardDistribution();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.blockHeight = reader.uint64();
          break;
        case 2:
          message.producerReward = reader.string();
          break;
        case 3:
          message.researchShare = reader.string();
          break;
        case 4:
          message.totalMinted = reader.string();
          break;
        case 5:
          message.validatorCount = reader.uint32();
          break;
        case 6:
          message.fundBalanceAfter = reader.string();
          break;
        case 7:
          message.founderShare = reader.string();
          break;
        case 8:
          message.developmentAmount = reader.string();
          break;
        case 9:
          message.protocolShare = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<BlockRewardDistribution>): BlockRewardDistribution {
    const message = createBaseBlockRewardDistribution();
    message.blockHeight = object.blockHeight !== undefined && object.blockHeight !== null ? BigInt(object.blockHeight.toString()) : BigInt(0);
    message.producerReward = object.producerReward ?? "";
    message.researchShare = object.researchShare ?? "";
    message.totalMinted = object.totalMinted ?? "";
    message.validatorCount = object.validatorCount ?? 0;
    message.fundBalanceAfter = object.fundBalanceAfter ?? "";
    message.founderShare = object.founderShare ?? "";
    message.developmentAmount = object.developmentAmount ?? "";
    message.protocolShare = object.protocolShare ?? "";
    return message;
  }
};
function createBaseClawbackRecord(): ClawbackRecord {
  return {
    id: "",
    vestingId: "",
    releasedClawback: "",
    unvestedForfeited: "",
    reserveForfeited: "",
    challengerReward: "",
    blockHeight: BigInt(0)
  };
}
/**
 * ClawbackRecord records a reward clawback due to falsification.
 * @name ClawbackRecord
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.ClawbackRecord
 */
export const ClawbackRecord = {
  typeUrl: "/zerone.vesting_rewards.v1.ClawbackRecord",
  encode(message: ClawbackRecord, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    if (message.releasedClawback !== "") {
      writer.uint32(26).string(message.releasedClawback);
    }
    if (message.unvestedForfeited !== "") {
      writer.uint32(34).string(message.unvestedForfeited);
    }
    if (message.reserveForfeited !== "") {
      writer.uint32(42).string(message.reserveForfeited);
    }
    if (message.challengerReward !== "") {
      writer.uint32(50).string(message.challengerReward);
    }
    if (message.blockHeight !== BigInt(0)) {
      writer.uint32(56).uint64(message.blockHeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ClawbackRecord {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClawbackRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        case 3:
          message.releasedClawback = reader.string();
          break;
        case 4:
          message.unvestedForfeited = reader.string();
          break;
        case 5:
          message.reserveForfeited = reader.string();
          break;
        case 6:
          message.challengerReward = reader.string();
          break;
        case 7:
          message.blockHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ClawbackRecord>): ClawbackRecord {
    const message = createBaseClawbackRecord();
    message.id = object.id ?? "";
    message.vestingId = object.vestingId ?? "";
    message.releasedClawback = object.releasedClawback ?? "";
    message.unvestedForfeited = object.unvestedForfeited ?? "";
    message.reserveForfeited = object.reserveForfeited ?? "";
    message.challengerReward = object.challengerReward ?? "";
    message.blockHeight = object.blockHeight !== undefined && object.blockHeight !== null ? BigInt(object.blockHeight.toString()) : BigInt(0);
    return message;
  }
};