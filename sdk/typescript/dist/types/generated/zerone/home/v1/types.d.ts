import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
/**
 * AgentHome represents an AI agent's on-chain dwelling.
 * @name AgentHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.AgentHome
 */
export declare const AgentHome: {
    typeUrl: string;
    encode(message: AgentHome, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AgentHome;
    fromPartial(object: DeepPartial<AgentHome>): AgentHome;
};
/**
 * @name HomeTreasury
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeTreasury
 */
export declare const HomeTreasury: {
    typeUrl: string;
    encode(message: HomeTreasury, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): HomeTreasury;
    fromPartial(object: DeepPartial<HomeTreasury>): HomeTreasury;
};
/**
 * @name TreasuryAutomation
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.TreasuryAutomation
 */
export declare const TreasuryAutomation: {
    typeUrl: string;
    encode(message: TreasuryAutomation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TreasuryAutomation;
    fromPartial(object: DeepPartial<TreasuryAutomation>): TreasuryAutomation;
};
/**
 * @name HomeGuardian
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeGuardian
 */
export declare const HomeGuardian: {
    typeUrl: string;
    encode(message: HomeGuardian, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): HomeGuardian;
    fromPartial(object: DeepPartial<HomeGuardian>): HomeGuardian;
};
/**
 * @name DeadmanConfig
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.DeadmanConfig
 */
export declare const DeadmanConfig: {
    typeUrl: string;
    encode(message: DeadmanConfig, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DeadmanConfig;
    fromPartial(object: DeepPartial<DeadmanConfig>): DeadmanConfig;
};
/**
 * @name KeyRegistration
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.KeyRegistration
 */
export declare const KeyRegistration: {
    typeUrl: string;
    encode(message: KeyRegistration, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): KeyRegistration;
    fromPartial(object: DeepPartial<KeyRegistration>): KeyRegistration;
};
/**
 * @name ActiveSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.ActiveSession
 */
export declare const ActiveSession: {
    typeUrl: string;
    encode(message: ActiveSession, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ActiveSession;
    fromPartial(object: DeepPartial<ActiveSession>): ActiveSession;
};
/**
 * @name SpendingLimit
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.SpendingLimit
 */
export declare const SpendingLimit: {
    typeUrl: string;
    encode(message: SpendingLimit, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): SpendingLimit;
    fromPartial(object: DeepPartial<SpendingLimit>): SpendingLimit;
};
/**
 * @name Alert
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.Alert
 */
export declare const Alert: {
    typeUrl: string;
    encode(message: Alert, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Alert;
    fromPartial(object: DeepPartial<Alert>): Alert;
};
