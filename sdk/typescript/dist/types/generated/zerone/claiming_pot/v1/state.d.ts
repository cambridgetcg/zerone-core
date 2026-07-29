import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
export declare enum PotStatus {
    POT_STATUS_UNSPECIFIED = 0,
    POT_STATUS_ACTIVE = 1,
    POT_STATUS_DEPLETED = 2,
    POT_STATUS_EXPIRED = 3,
    UNRECOGNIZED = -1
}
export declare function potStatusFromJSON(object: any): PotStatus;
export declare function potStatusToJSON(object: PotStatus): string;
/**
 * @name ClaimingPot
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.ClaimingPot
 */
export interface ClaimingPot {
    id: string;
    name: string;
    /**
     * uzrn
     */
    totalAmount: string;
    /**
     * uzrn
     */
    claimedAmount: string;
    schedule?: VestingSchedule;
    eligibility?: EligibilityCriteria;
    createdAtBlock: bigint;
    status: PotStatus;
}
/**
 * @name VestingSchedule
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.VestingSchedule
 */
export interface VestingSchedule {
    startBlock: bigint;
    endBlock: bigint;
    /**
     * blocks after start before any vesting
     */
    cliffBlocks: bigint;
    /**
     * vesting period granularity
     */
    periodBlocks: bigint;
}
/**
 * @name Claim
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.Claim
 */
export interface Claim {
    potId: string;
    /**
     * bech32
     */
    claimant: string;
    /**
     * uzrn
     */
    amount: string;
    /**
     * block height
     */
    claimedAt: bigint;
}
/**
 * @name EligibilityCriteria
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.EligibilityCriteria
 */
export interface EligibilityCriteria {
    /**
     * minimum validator tier required
     */
    minStakingTier: number;
    /**
     * minimum blocks since registration
     */
    minRegistrationAge: bigint;
    /**
     * bech32 addresses (empty = open to all qualified)
     */
    whitelist: string[];
}
/**
 * @name Params
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.Params
 */
export interface Params {
    /**
     * default: 10
     */
    maxPotsActive: number;
    /**
     * default: "1000" uzrn
     */
    minClaimAmount: string;
    /**
     * bootstrap_registrar is an optional bech32 account (e.g. the agenttool
     * 2-of-3 ops multisig) accepted alongside the gov authority as the signer
     * of MsgAddBootstrapEntry. Empty string (default) disables the registrar
     * pathway — governance remains the only admitter. Revocation is a single
     * param change setting this back to "".
     */
    bootstrapRegistrar: string;
    /**
     * bootstrap_emission_cap_uzrn is the shared lifetime commitment cap for
     * bootstrap entries and legacy general pots. Every pot is charged
     * ceil(total_amount / 222,000) fixed-size units. Applies to governance and
     * registrar admissions.
     */
    bootstrapEmissionCapUzrn: string;
    /**
     * bootstrap_daily_admission_cap is the maximum number of registrar
     * admissions per 34,272-block window (~1 day). Gov-authority admissions
     * bypass this window (they remain bounded by the emission cap and the
     * governance process itself).
     */
    bootstrapDailyAdmissionCap: bigint;
}
/**
 * @name ClaimingPot
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.ClaimingPot
 */
export declare const ClaimingPot: {
    typeUrl: string;
    encode(message: ClaimingPot, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClaimingPot;
    fromPartial(object: DeepPartial<ClaimingPot>): ClaimingPot;
};
/**
 * @name VestingSchedule
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.VestingSchedule
 */
export declare const VestingSchedule: {
    typeUrl: string;
    encode(message: VestingSchedule, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): VestingSchedule;
    fromPartial(object: DeepPartial<VestingSchedule>): VestingSchedule;
};
/**
 * @name Claim
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.Claim
 */
export declare const Claim: {
    typeUrl: string;
    encode(message: Claim, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Claim;
    fromPartial(object: DeepPartial<Claim>): Claim;
};
/**
 * @name EligibilityCriteria
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.EligibilityCriteria
 */
export declare const EligibilityCriteria: {
    typeUrl: string;
    encode(message: EligibilityCriteria, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EligibilityCriteria;
    fromPartial(object: DeepPartial<EligibilityCriteria>): EligibilityCriteria;
};
/**
 * @name Params
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
