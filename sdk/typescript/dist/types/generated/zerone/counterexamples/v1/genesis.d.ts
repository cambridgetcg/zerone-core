import { Counterexample, Validation } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name GenesisState
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    counterexamples: Counterexample[];
    validations: Validation[];
    nextCounterexampleSeq: bigint;
    nextValidationId: bigint;
}
/**
 * @name Params
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.Params
 */
export interface Params {
    /**
     * uzrn bond locked when proposing a counterexample. Returned on
     * VALIDATED, burned on REJECTED. Default: 1 ZRN.
     */
    proposalBond: string;
    /**
     * uzrn reward paid from the protocol treasury to the proposer when
     * a counterexample is VALIDATED. Default: 0.5 ZRN. Note this
     * exceeds bond at the margin so the chain ECONOMICALLY ENCOURAGES
     * counterexample contribution — alignment-by-structure is a public
     * good the chain pays for.
     */
    validationReward: string;
    /**
     * Minimum total votes (validations + rejections) before the chain
     * will resolve a counterexample. Prevents premature resolution on
     * a single vote.
     */
    minVotes: number;
    /**
     * Validation succeeds if affirm_votes >= total_votes *
     * affirm_threshold_bps / 1_000_000. Default: 666,000 (66.6%).
     * Below that, counterexample is REJECTED.
     */
    affirmThresholdBps: bigint;
    /**
     * Maximum bytes for reasoning, wrong_claim, and validation reason
     * text fields.
     */
    maxReasonBytes: number;
    /**
     * Per-fact TVW multiplier (in BPS) granted when a fact has at
     * least one VALIDATED counterexample. Read by x/knowledge during
     * ComputeTrainingValueWeight.
     * Default: 1,200,000 (1.2x — facts with counterexamples earn 20%
     * more training-data value than facts without).
     */
    tvwMultiplierBps: bigint;
    /**
     * Whether new counterexample proposals are accepted. Governance
     * can pause without affecting existing counterexamples.
     */
    proposalsEnabled: boolean;
}
/**
 * @name GenesisState
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * @name Params
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
