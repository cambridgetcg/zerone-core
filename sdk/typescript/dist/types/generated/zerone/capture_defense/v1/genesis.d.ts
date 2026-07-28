import { GlobalReputation, StratumReputation, DomainReputation, CaptureMetrics, CrossStratumRequirement } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name Params
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.Params
 */
export interface Params {
    /**
     * half-life for reputation decay
     */
    decayEpochBlocks: bigint;
    /**
     * min verifications before domain rep is used
     */
    minVerificationsForScore: bigint;
    /**
     * HHI above which domain is flagged
     */
    hhiThreshold: bigint;
    /**
     * blocks between auto-analysis
     */
    riskAnalysisInterval: bigint;
    /**
     * how long verification history is kept
     */
    historyRetentionBlocks: bigint;
    /**
     * floor for decayed scores
     */
    baseReputationScore: bigint;
    /**
     * max rounds kept per domain
     */
    maxHistoryPerDomain: bigint;
    /**
     * base recovery rate per decay epoch (default: 50,000 = 5%)
     */
    baseReputationRecoveryBps: bigint;
    /**
     * max acceleration from verification activity (default: 500,000 = 50%)
     */
    activityRecoveryBonusMaxBps: bigint;
}
/**
 * @name GenesisState
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    globalReputations: GlobalReputation[];
    stratumReputations: StratumReputation[];
    domainReputations: DomainReputation[];
    captureMetrics: CaptureMetrics[];
    crossStratumRequirements: CrossStratumRequirement[];
}
/**
 * @name Params
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * @name GenesisState
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
