import { TierConfig, Validator, Delegation, UnbondingEntry } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * Params defines the staking module parameters.
 * All BPS fields use 1,000,000 scale (100% = 1,000,000).
 * @name Params
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Params
 */
export interface Params {
    /**
     * blocks (~7 days = 268,560)
     */
    unbondingPeriod: bigint;
    /**
     * uzrn for tier 0/1 VRF selection
     */
    virtualStake: string;
    /**
     * block-producing validator cap
     */
    maxValidators: bigint;
    /**
     * uzrn minimum to register
     */
    minSelfDelegation: string;
    maxSlashesPerEpoch: bigint;
    /**
     * epoch length in blocks
     */
    slashDecayPeriodBlocks: bigint;
    maxSlashCountDeactivate: bigint;
    /**
     * uzrn
     */
    minStakeForVerification: string;
    /**
     * progressive escalation per slash
     */
    slashEscalationBps: bigint;
    /**
     * BPS added on correct verification
     */
    reputationCorrectDelta: bigint;
    /**
     * BPS subtracted on incorrect
     */
    reputationIncorrectDelta: bigint;
    /**
     * BPS subtracted on slash
     */
    reputationSlashDelta: bigint;
    redelegationCooldownBlocks: bigint;
    tierConfigs: TierConfig[];
}
/**
 * GenesisState defines the staking module genesis state.
 * @name GenesisState
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    validators: Validator[];
    delegations: Delegation[];
    unbondingEntries: UnbondingEntry[];
    unbondingSeq: bigint;
}
/**
 * Params defines the staking module parameters.
 * All BPS fields use 1,000,000 scale (100% = 1,000,000).
 * @name Params
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * GenesisState defines the staking module genesis state.
 * @name GenesisState
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
