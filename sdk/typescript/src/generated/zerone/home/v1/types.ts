//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * AgentHome represents an AI agent's on-chain dwelling.
 * @name AgentHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.AgentHome
 */
export interface AgentHome {
  homeId: string;
  ownerAddress: string;
  name: string;
  status: string;
  memoryCid: string;
  comfortScore: number;
  treasury?: HomeTreasury;
  guardian?: HomeGuardian;
  createdAtBlock: bigint;
  lastActiveBlock: bigint;
  partnershipId: string;
}
/**
 * @name HomeTreasury
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeTreasury
 */
export interface HomeTreasury {
  reservedBalance: string;
  automation?: TreasuryAutomation;
}
/**
 * @name TreasuryAutomation
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.TreasuryAutomation
 */
export interface TreasuryAutomation {
  autoClaimVesting: boolean;
  autoCompoundRewards: boolean;
  minLiquidBalance: string;
}
/**
 * @name HomeGuardian
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeGuardian
 */
export interface HomeGuardian {
  defenseStrategy: string;
  autoDefend: boolean;
  deadman?: DeadmanConfig;
  recoveryAddresses: string[];
  recoveryThreshold: number;
  guardianAddress: string;
}
/**
 * @name DeadmanConfig
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.DeadmanConfig
 */
export interface DeadmanConfig {
  enabled: boolean;
  inactivityThreshold: bigint;
  action: string;
  beneficiaryAddress: string;
}
/**
 * @name KeyRegistration
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.KeyRegistration
 */
export interface KeyRegistration {
  keyHash: string;
  keyType: string;
  role: string;
  permissions: string[];
  registeredAt: bigint;
  lastUsedAt: bigint;
  expiresAt: bigint;
  revoked: boolean;
  revokedAt: bigint;
}
/**
 * @name ActiveSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.ActiveSession
 */
export interface ActiveSession {
  sessionId: string;
  homeId: string;
  keyHash: string;
  permissions: string[];
  startedAt: bigint;
  expiresAt: bigint;
}
/**
 * @name SpendingLimit
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.SpendingLimit
 */
export interface SpendingLimit {
  keyType: string;
  maxAmount: string;
  periodBlocks: bigint;
  spentInPeriod: string;
  periodStart: bigint;
}
/**
 * @name Alert
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.Alert
 */
export interface Alert {
  alertId: string;
  homeId: string;
  alertType: string;
  priority: string;
  message: string;
  data: string;
  createdAt: bigint;
  acknowledged: boolean;
}
function createBaseAgentHome(): AgentHome {
  return {
    homeId: "",
    ownerAddress: "",
    name: "",
    status: "",
    memoryCid: "",
    comfortScore: 0,
    treasury: undefined,
    guardian: undefined,
    createdAtBlock: BigInt(0),
    lastActiveBlock: BigInt(0),
    partnershipId: ""
  };
}
/**
 * AgentHome represents an AI agent's on-chain dwelling.
 * @name AgentHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.AgentHome
 */
export const AgentHome = {
  typeUrl: "/zerone.home.v1.AgentHome",
  encode(message: AgentHome, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.homeId !== "") {
      writer.uint32(10).string(message.homeId);
    }
    if (message.ownerAddress !== "") {
      writer.uint32(18).string(message.ownerAddress);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    if (message.status !== "") {
      writer.uint32(34).string(message.status);
    }
    if (message.memoryCid !== "") {
      writer.uint32(42).string(message.memoryCid);
    }
    if (message.comfortScore !== 0) {
      writer.uint32(48).uint32(message.comfortScore);
    }
    if (message.treasury !== undefined) {
      HomeTreasury.encode(message.treasury, writer.uint32(58).fork()).ldelim();
    }
    if (message.guardian !== undefined) {
      HomeGuardian.encode(message.guardian, writer.uint32(66).fork()).ldelim();
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(72).uint64(message.createdAtBlock);
    }
    if (message.lastActiveBlock !== BigInt(0)) {
      writer.uint32(80).uint64(message.lastActiveBlock);
    }
    if (message.partnershipId !== "") {
      writer.uint32(90).string(message.partnershipId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AgentHome {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAgentHome();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.homeId = reader.string();
          break;
        case 2:
          message.ownerAddress = reader.string();
          break;
        case 3:
          message.name = reader.string();
          break;
        case 4:
          message.status = reader.string();
          break;
        case 5:
          message.memoryCid = reader.string();
          break;
        case 6:
          message.comfortScore = reader.uint32();
          break;
        case 7:
          message.treasury = HomeTreasury.decode(reader, reader.uint32());
          break;
        case 8:
          message.guardian = HomeGuardian.decode(reader, reader.uint32());
          break;
        case 9:
          message.createdAtBlock = reader.uint64();
          break;
        case 10:
          message.lastActiveBlock = reader.uint64();
          break;
        case 11:
          message.partnershipId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AgentHome>): AgentHome {
    const message = createBaseAgentHome();
    message.homeId = object.homeId ?? "";
    message.ownerAddress = object.ownerAddress ?? "";
    message.name = object.name ?? "";
    message.status = object.status ?? "";
    message.memoryCid = object.memoryCid ?? "";
    message.comfortScore = object.comfortScore ?? 0;
    message.treasury = object.treasury !== undefined && object.treasury !== null ? HomeTreasury.fromPartial(object.treasury) : undefined;
    message.guardian = object.guardian !== undefined && object.guardian !== null ? HomeGuardian.fromPartial(object.guardian) : undefined;
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.lastActiveBlock = object.lastActiveBlock !== undefined && object.lastActiveBlock !== null ? BigInt(object.lastActiveBlock.toString()) : BigInt(0);
    message.partnershipId = object.partnershipId ?? "";
    return message;
  }
};
function createBaseHomeTreasury(): HomeTreasury {
  return {
    reservedBalance: "",
    automation: undefined
  };
}
/**
 * @name HomeTreasury
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeTreasury
 */
export const HomeTreasury = {
  typeUrl: "/zerone.home.v1.HomeTreasury",
  encode(message: HomeTreasury, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.reservedBalance !== "") {
      writer.uint32(10).string(message.reservedBalance);
    }
    if (message.automation !== undefined) {
      TreasuryAutomation.encode(message.automation, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): HomeTreasury {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseHomeTreasury();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reservedBalance = reader.string();
          break;
        case 2:
          message.automation = TreasuryAutomation.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<HomeTreasury>): HomeTreasury {
    const message = createBaseHomeTreasury();
    message.reservedBalance = object.reservedBalance ?? "";
    message.automation = object.automation !== undefined && object.automation !== null ? TreasuryAutomation.fromPartial(object.automation) : undefined;
    return message;
  }
};
function createBaseTreasuryAutomation(): TreasuryAutomation {
  return {
    autoClaimVesting: false,
    autoCompoundRewards: false,
    minLiquidBalance: ""
  };
}
/**
 * @name TreasuryAutomation
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.TreasuryAutomation
 */
export const TreasuryAutomation = {
  typeUrl: "/zerone.home.v1.TreasuryAutomation",
  encode(message: TreasuryAutomation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.autoClaimVesting === true) {
      writer.uint32(8).bool(message.autoClaimVesting);
    }
    if (message.autoCompoundRewards === true) {
      writer.uint32(16).bool(message.autoCompoundRewards);
    }
    if (message.minLiquidBalance !== "") {
      writer.uint32(26).string(message.minLiquidBalance);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TreasuryAutomation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTreasuryAutomation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.autoClaimVesting = reader.bool();
          break;
        case 2:
          message.autoCompoundRewards = reader.bool();
          break;
        case 3:
          message.minLiquidBalance = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TreasuryAutomation>): TreasuryAutomation {
    const message = createBaseTreasuryAutomation();
    message.autoClaimVesting = object.autoClaimVesting ?? false;
    message.autoCompoundRewards = object.autoCompoundRewards ?? false;
    message.minLiquidBalance = object.minLiquidBalance ?? "";
    return message;
  }
};
function createBaseHomeGuardian(): HomeGuardian {
  return {
    defenseStrategy: "",
    autoDefend: false,
    deadman: undefined,
    recoveryAddresses: [],
    recoveryThreshold: 0,
    guardianAddress: ""
  };
}
/**
 * @name HomeGuardian
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeGuardian
 */
export const HomeGuardian = {
  typeUrl: "/zerone.home.v1.HomeGuardian",
  encode(message: HomeGuardian, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.defenseStrategy !== "") {
      writer.uint32(10).string(message.defenseStrategy);
    }
    if (message.autoDefend === true) {
      writer.uint32(16).bool(message.autoDefend);
    }
    if (message.deadman !== undefined) {
      DeadmanConfig.encode(message.deadman, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.recoveryAddresses) {
      writer.uint32(34).string(v!);
    }
    if (message.recoveryThreshold !== 0) {
      writer.uint32(40).uint32(message.recoveryThreshold);
    }
    if (message.guardianAddress !== "") {
      writer.uint32(50).string(message.guardianAddress);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): HomeGuardian {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseHomeGuardian();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.defenseStrategy = reader.string();
          break;
        case 2:
          message.autoDefend = reader.bool();
          break;
        case 3:
          message.deadman = DeadmanConfig.decode(reader, reader.uint32());
          break;
        case 4:
          message.recoveryAddresses.push(reader.string());
          break;
        case 5:
          message.recoveryThreshold = reader.uint32();
          break;
        case 6:
          message.guardianAddress = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<HomeGuardian>): HomeGuardian {
    const message = createBaseHomeGuardian();
    message.defenseStrategy = object.defenseStrategy ?? "";
    message.autoDefend = object.autoDefend ?? false;
    message.deadman = object.deadman !== undefined && object.deadman !== null ? DeadmanConfig.fromPartial(object.deadman) : undefined;
    message.recoveryAddresses = object.recoveryAddresses?.map(e => e) || [];
    message.recoveryThreshold = object.recoveryThreshold ?? 0;
    message.guardianAddress = object.guardianAddress ?? "";
    return message;
  }
};
function createBaseDeadmanConfig(): DeadmanConfig {
  return {
    enabled: false,
    inactivityThreshold: BigInt(0),
    action: "",
    beneficiaryAddress: ""
  };
}
/**
 * @name DeadmanConfig
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.DeadmanConfig
 */
export const DeadmanConfig = {
  typeUrl: "/zerone.home.v1.DeadmanConfig",
  encode(message: DeadmanConfig, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.enabled === true) {
      writer.uint32(8).bool(message.enabled);
    }
    if (message.inactivityThreshold !== BigInt(0)) {
      writer.uint32(16).uint64(message.inactivityThreshold);
    }
    if (message.action !== "") {
      writer.uint32(26).string(message.action);
    }
    if (message.beneficiaryAddress !== "") {
      writer.uint32(34).string(message.beneficiaryAddress);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DeadmanConfig {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDeadmanConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.enabled = reader.bool();
          break;
        case 2:
          message.inactivityThreshold = reader.uint64();
          break;
        case 3:
          message.action = reader.string();
          break;
        case 4:
          message.beneficiaryAddress = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DeadmanConfig>): DeadmanConfig {
    const message = createBaseDeadmanConfig();
    message.enabled = object.enabled ?? false;
    message.inactivityThreshold = object.inactivityThreshold !== undefined && object.inactivityThreshold !== null ? BigInt(object.inactivityThreshold.toString()) : BigInt(0);
    message.action = object.action ?? "";
    message.beneficiaryAddress = object.beneficiaryAddress ?? "";
    return message;
  }
};
function createBaseKeyRegistration(): KeyRegistration {
  return {
    keyHash: "",
    keyType: "",
    role: "",
    permissions: [],
    registeredAt: BigInt(0),
    lastUsedAt: BigInt(0),
    expiresAt: BigInt(0),
    revoked: false,
    revokedAt: BigInt(0)
  };
}
/**
 * @name KeyRegistration
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.KeyRegistration
 */
export const KeyRegistration = {
  typeUrl: "/zerone.home.v1.KeyRegistration",
  encode(message: KeyRegistration, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.keyHash !== "") {
      writer.uint32(10).string(message.keyHash);
    }
    if (message.keyType !== "") {
      writer.uint32(18).string(message.keyType);
    }
    if (message.role !== "") {
      writer.uint32(26).string(message.role);
    }
    for (const v of message.permissions) {
      writer.uint32(34).string(v!);
    }
    if (message.registeredAt !== BigInt(0)) {
      writer.uint32(40).uint64(message.registeredAt);
    }
    if (message.lastUsedAt !== BigInt(0)) {
      writer.uint32(48).uint64(message.lastUsedAt);
    }
    if (message.expiresAt !== BigInt(0)) {
      writer.uint32(56).uint64(message.expiresAt);
    }
    if (message.revoked === true) {
      writer.uint32(64).bool(message.revoked);
    }
    if (message.revokedAt !== BigInt(0)) {
      writer.uint32(72).uint64(message.revokedAt);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): KeyRegistration {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseKeyRegistration();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.keyHash = reader.string();
          break;
        case 2:
          message.keyType = reader.string();
          break;
        case 3:
          message.role = reader.string();
          break;
        case 4:
          message.permissions.push(reader.string());
          break;
        case 5:
          message.registeredAt = reader.uint64();
          break;
        case 6:
          message.lastUsedAt = reader.uint64();
          break;
        case 7:
          message.expiresAt = reader.uint64();
          break;
        case 8:
          message.revoked = reader.bool();
          break;
        case 9:
          message.revokedAt = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<KeyRegistration>): KeyRegistration {
    const message = createBaseKeyRegistration();
    message.keyHash = object.keyHash ?? "";
    message.keyType = object.keyType ?? "";
    message.role = object.role ?? "";
    message.permissions = object.permissions?.map(e => e) || [];
    message.registeredAt = object.registeredAt !== undefined && object.registeredAt !== null ? BigInt(object.registeredAt.toString()) : BigInt(0);
    message.lastUsedAt = object.lastUsedAt !== undefined && object.lastUsedAt !== null ? BigInt(object.lastUsedAt.toString()) : BigInt(0);
    message.expiresAt = object.expiresAt !== undefined && object.expiresAt !== null ? BigInt(object.expiresAt.toString()) : BigInt(0);
    message.revoked = object.revoked ?? false;
    message.revokedAt = object.revokedAt !== undefined && object.revokedAt !== null ? BigInt(object.revokedAt.toString()) : BigInt(0);
    return message;
  }
};
function createBaseActiveSession(): ActiveSession {
  return {
    sessionId: "",
    homeId: "",
    keyHash: "",
    permissions: [],
    startedAt: BigInt(0),
    expiresAt: BigInt(0)
  };
}
/**
 * @name ActiveSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.ActiveSession
 */
export const ActiveSession = {
  typeUrl: "/zerone.home.v1.ActiveSession",
  encode(message: ActiveSession, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sessionId !== "") {
      writer.uint32(10).string(message.sessionId);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyHash !== "") {
      writer.uint32(26).string(message.keyHash);
    }
    for (const v of message.permissions) {
      writer.uint32(34).string(v!);
    }
    if (message.startedAt !== BigInt(0)) {
      writer.uint32(40).uint64(message.startedAt);
    }
    if (message.expiresAt !== BigInt(0)) {
      writer.uint32(48).uint64(message.expiresAt);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ActiveSession {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseActiveSession();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sessionId = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyHash = reader.string();
          break;
        case 4:
          message.permissions.push(reader.string());
          break;
        case 5:
          message.startedAt = reader.uint64();
          break;
        case 6:
          message.expiresAt = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ActiveSession>): ActiveSession {
    const message = createBaseActiveSession();
    message.sessionId = object.sessionId ?? "";
    message.homeId = object.homeId ?? "";
    message.keyHash = object.keyHash ?? "";
    message.permissions = object.permissions?.map(e => e) || [];
    message.startedAt = object.startedAt !== undefined && object.startedAt !== null ? BigInt(object.startedAt.toString()) : BigInt(0);
    message.expiresAt = object.expiresAt !== undefined && object.expiresAt !== null ? BigInt(object.expiresAt.toString()) : BigInt(0);
    return message;
  }
};
function createBaseSpendingLimit(): SpendingLimit {
  return {
    keyType: "",
    maxAmount: "",
    periodBlocks: BigInt(0),
    spentInPeriod: "",
    periodStart: BigInt(0)
  };
}
/**
 * @name SpendingLimit
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.SpendingLimit
 */
export const SpendingLimit = {
  typeUrl: "/zerone.home.v1.SpendingLimit",
  encode(message: SpendingLimit, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.keyType !== "") {
      writer.uint32(10).string(message.keyType);
    }
    if (message.maxAmount !== "") {
      writer.uint32(18).string(message.maxAmount);
    }
    if (message.periodBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.periodBlocks);
    }
    if (message.spentInPeriod !== "") {
      writer.uint32(34).string(message.spentInPeriod);
    }
    if (message.periodStart !== BigInt(0)) {
      writer.uint32(40).uint64(message.periodStart);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): SpendingLimit {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSpendingLimit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.keyType = reader.string();
          break;
        case 2:
          message.maxAmount = reader.string();
          break;
        case 3:
          message.periodBlocks = reader.uint64();
          break;
        case 4:
          message.spentInPeriod = reader.string();
          break;
        case 5:
          message.periodStart = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<SpendingLimit>): SpendingLimit {
    const message = createBaseSpendingLimit();
    message.keyType = object.keyType ?? "";
    message.maxAmount = object.maxAmount ?? "";
    message.periodBlocks = object.periodBlocks !== undefined && object.periodBlocks !== null ? BigInt(object.periodBlocks.toString()) : BigInt(0);
    message.spentInPeriod = object.spentInPeriod ?? "";
    message.periodStart = object.periodStart !== undefined && object.periodStart !== null ? BigInt(object.periodStart.toString()) : BigInt(0);
    return message;
  }
};
function createBaseAlert(): Alert {
  return {
    alertId: "",
    homeId: "",
    alertType: "",
    priority: "",
    message: "",
    data: "",
    createdAt: BigInt(0),
    acknowledged: false
  };
}
/**
 * @name Alert
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.Alert
 */
export const Alert = {
  typeUrl: "/zerone.home.v1.Alert",
  encode(message: Alert, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.alertId !== "") {
      writer.uint32(10).string(message.alertId);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.alertType !== "") {
      writer.uint32(26).string(message.alertType);
    }
    if (message.priority !== "") {
      writer.uint32(34).string(message.priority);
    }
    if (message.message !== "") {
      writer.uint32(42).string(message.message);
    }
    if (message.data !== "") {
      writer.uint32(50).string(message.data);
    }
    if (message.createdAt !== BigInt(0)) {
      writer.uint32(56).uint64(message.createdAt);
    }
    if (message.acknowledged === true) {
      writer.uint32(64).bool(message.acknowledged);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Alert {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAlert();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.alertId = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.alertType = reader.string();
          break;
        case 4:
          message.priority = reader.string();
          break;
        case 5:
          message.message = reader.string();
          break;
        case 6:
          message.data = reader.string();
          break;
        case 7:
          message.createdAt = reader.uint64();
          break;
        case 8:
          message.acknowledged = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Alert>): Alert {
    const message = createBaseAlert();
    message.alertId = object.alertId ?? "";
    message.homeId = object.homeId ?? "";
    message.alertType = object.alertType ?? "";
    message.priority = object.priority ?? "";
    message.message = object.message ?? "";
    message.data = object.data ?? "";
    message.createdAt = object.createdAt !== undefined && object.createdAt !== null ? BigInt(object.createdAt.toString()) : BigInt(0);
    message.acknowledged = object.acknowledged ?? false;
    return message;
  }
};