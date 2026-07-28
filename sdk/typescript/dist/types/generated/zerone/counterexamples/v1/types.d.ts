import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
export declare enum ErrorType {
    ERROR_TYPE_UNSPECIFIED = 0,
    /**
     * ERROR_TYPE_CATEGORICAL - Confuses categories (e.g. treats a normative claim as empirical,
     * or applies one stratum's confidence to another).
     */
    ERROR_TYPE_CATEGORICAL = 1,
    /**
     * ERROR_TYPE_METHODOLOGY - Would require a methodology that does not yield this conclusion
     * (e.g. claims an ECOLOGICAL conclusion from a LEGACY method).
     */
    ERROR_TYPE_METHODOLOGY = 2,
    /**
     * ERROR_TYPE_FACTUAL - Contradicts established facts (would conflict with a higher-
     * confidence fact already in the corpus).
     */
    ERROR_TYPE_FACTUAL = 3,
    /**
     * ERROR_TYPE_REASONING - Premises are valid but the inference is invalid (non sequitur,
     * affirming the consequent, etc.).
     */
    ERROR_TYPE_REASONING = 4,
    /**
     * ERROR_TYPE_OMISSION - Ignores critical context that the fact addresses (e.g. omits
     * boundary conditions, scope qualifiers, or cited counter-evidence).
     */
    ERROR_TYPE_OMISSION = 5,
    /** ERROR_TYPE_CIRCULAR - Begs the question — the reasoning presupposes its conclusion. */
    ERROR_TYPE_CIRCULAR = 6,
    UNRECOGNIZED = -1
}
export declare function errorTypeFromJSON(object: any): ErrorType;
export declare function errorTypeToJSON(object: ErrorType): string;
export declare enum CounterexampleStatus {
    COUNTEREXAMPLE_STATUS_UNSPECIFIED = 0,
    COUNTEREXAMPLE_STATUS_PROPOSED = 1,
    COUNTEREXAMPLE_STATUS_VALIDATED = 2,
    COUNTEREXAMPLE_STATUS_REJECTED = 3,
    UNRECOGNIZED = -1
}
export declare function counterexampleStatusFromJSON(object: any): CounterexampleStatus;
export declare function counterexampleStatusToJSON(object: CounterexampleStatus): string;
/**
 * Counterexample is a structured "wrong-but-plausible alternative" paired
 * with a Fact. The alignment-by-structure principle: a model trained on
 * (fact, counterexample) pairs learns the discriminator, not just the
 * predictor — distinguishing right from wrong is the cognitive primitive
 * that resists manipulation.
 *
 * See docs/TRUTH_SEEKING.md commitment 15.
 * @name Counterexample
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.Counterexample
 */
export interface Counterexample {
    /**
     * Chain-assigned monotonic id. Format: "ce-<n>".
     */
    id: string;
    /**
     * The fact this counterexample is paired with. Must reference an
     * existing fact in x/knowledge.
     */
    factId: string;
    /**
     * Bech32 address of whoever proposed the counterexample.
     */
    author: string;
    /**
     * The wrong claim — a plausible but mistaken alternative to the
     * fact's content. The training signal is "this would seem to follow
     * but is wrong because <reasoning>."
     */
    wrongClaim: string;
    /**
     * Why the wrong_claim is wrong. The reasoning is what teaches the
     * model the discriminator.
     */
    reasoning: string;
    /**
     * Category of error. The enumeration produces structured curriculum
     * for training pipelines (e.g., "show me 100 CATEGORICAL errors").
     */
    errorType: ErrorType;
    /**
     * Methodology IDs whose output, if mis-applied, would yield the
     * wrong_claim. Optional but encouraged — connects the counterexample
     * to commitment 1 (methodology over statement).
     */
    violatedMethodologyIds: string[];
    /**
     * Block when proposed.
     */
    submittedAtBlock: bigint;
    /**
     * Status driven by validations and rejections crossing thresholds.
     */
    status: CounterexampleStatus;
    /**
     * Tally fields. Validations affirm "yes this IS a counterexample
     * and the reasoning is right." Rejections deny it.
     */
    validations: number;
    rejections: number;
    /**
     * Block when status was last changed (PROPOSED → VALIDATED/REJECTED).
     */
    resolvedAtBlock: bigint;
    /**
     * uzrn bond locked on submission. Returned (plus reward) on
     * VALIDATED; burned on REJECTED.
     */
    bond: string;
}
/**
 * Validation is one validator's vote on a Counterexample. The chain
 * records every vote (commitment 10: forward-only audit).
 * @name Validation
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.Validation
 */
export interface Validation {
    id: bigint;
    counterexampleId: string;
    /**
     * bech32
     */
    validator: string;
    /**
     * True = "this IS a good counterexample"; false = "no, this is
     * actually right or the reasoning is wrong."
     */
    affirm: boolean;
    /**
     * Free-text rationale. Bounded by params.max_reason_bytes.
     */
    reason: string;
    submittedAtBlock: bigint;
}
/**
 * Counterexample is a structured "wrong-but-plausible alternative" paired
 * with a Fact. The alignment-by-structure principle: a model trained on
 * (fact, counterexample) pairs learns the discriminator, not just the
 * predictor — distinguishing right from wrong is the cognitive primitive
 * that resists manipulation.
 *
 * See docs/TRUTH_SEEKING.md commitment 15.
 * @name Counterexample
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.Counterexample
 */
export declare const Counterexample: {
    typeUrl: string;
    encode(message: Counterexample, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Counterexample;
    fromPartial(object: DeepPartial<Counterexample>): Counterexample;
};
/**
 * Validation is one validator's vote on a Counterexample. The chain
 * records every vote (commitment 10: forward-only audit).
 * @name Validation
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.Validation
 */
export declare const Validation: {
    typeUrl: string;
    encode(message: Validation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Validation;
    fromPartial(object: DeepPartial<Validation>): Validation;
};
