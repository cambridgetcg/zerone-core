import { DomainQualification, QualificationEndorsement } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name GenesisState
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    qualifications: DomainQualification[];
    endorsements: QualificationEndorsement[];
    nextEndorsementId: bigint;
}
/**
 * @name Params
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.Params
 */
export interface Params {
    /**
     * Stake pathway
     */
    minStakeAmount: string;
    /**
     * blocks to lock stake
     */
    stakeLockPeriod: bigint;
    /**
     * Track record pathway
     */
    minVerifications: bigint;
    /**
     * minimum accuracy (1M scale)
     */
    minAccuracyBps: bigint;
    /**
     * minimum domain reputation (from CaptureDefense)
     */
    minReputationScore: bigint;
    /**
     * Qualification settings
     */
    qualificationPeriod: bigint;
    /**
     * blocks for probationary period
     */
    probationPeriod: bigint;
    /**
     * blocks before expiry where renewal allowed
     */
    renewalWindow: bigint;
    /**
     * max endorsements per qualification
     */
    maxEndorsements: number;
    /**
     * Cross-reference
     */
    crossRefMinWeight: bigint;
    /**
     * discount applied to inherited weight (1M scale)
     */
    crossRefWeightDiscountBps: bigint;
    /**
     * Inheritance
     */
    inheritanceWeightDiscountBps: bigint;
    /**
     * Endorsement diversity guard (L3). Maximum allowed fraction of shared
     * domain qualifications between endorser and endorsee, measured as
     * shared / min(|endorser_domains|, |endorsee_domains|). Prevents rings
     * where a small group of heavily-overlapped validators repeatedly endorses
     * each other. "0" disables the guard.
     */
    endorsementMaxOverlapBps: bigint;
    /**
     * ─── Wave 16: accuracy-based decay ────────────────────────────────────
     * Skill is current, not historical. Once a qualification accumulates
     * enough samples (decay_min_samples), the BeginBlocker checks accuracy
     * every decay_check_interval_blocks and transitions state when it
     * crosses thresholds. ACTIVE → PROBATIONARY when accuracy slips below
     * decay_probation_bps; PROBATIONARY → SUSPENDED when below
     * decay_suspension_bps; PROBATIONARY → ACTIVE when accuracy recovers
     * above decay_recovery_bps. Without these thresholds qualification is
     * a one-time grant, not an ongoing competency assessment.
     */
    decayCheckIntervalBlocks: bigint;
    /**
     * default: 20 — minimum verifications before decay applies
     */
    decayMinSamples: bigint;
    /**
     * default: 600,000 (60%) — ACTIVE → PROBATIONARY
     */
    decayProbationBps: bigint;
    /**
     * default: 400,000 (40%) — PROBATIONARY → SUSPENDED
     */
    decaySuspensionBps: bigint;
    /**
     * default: 750,000 (75%) — PROBATIONARY → ACTIVE
     */
    decayRecoveryBps: bigint;
}
/**
 * @name GenesisState
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * @name Params
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
