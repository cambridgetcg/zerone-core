import { CaptureChallenge, DomainBountyPool } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name Params
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.Params
 */
export interface Params {
    /**
     * minimum uzrn to stake a challenge
     */
    minChallengeStake: string;
    /**
     * blocks allowed for evidence submission
     */
    evidencePeriodBlocks: bigint;
    /**
     * blocks allowed for governance review
     */
    reviewPeriodBlocks: bigint;
    /**
     * blocks domain is paused on challenge
     */
    domainPauseBlocks: bigint;
    /**
     * reward from bounty pool (BPS of pool)
     */
    rewardRateBps: bigint;
    /**
     * slash of accused validators (BPS of stake)
     */
    slashRateBps: bigint;
    /**
     * uzrn per fact creation auto-funded
     */
    bountyContributionPerFact: string;
    /**
     * blocks between auto risk analysis
     */
    riskAnalysisInterval: bigint;
}
/**
 * @name GenesisState
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    challenges: CaptureChallenge[];
    bountyPools: DomainBountyPool[];
}
/**
 * @name Params
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * @name GenesisState
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
