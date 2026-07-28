//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
export enum ErrorType {
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
  UNRECOGNIZED = -1,
}
export function errorTypeFromJSON(object: any): ErrorType {
  switch (object) {
    case 0:
    case "ERROR_TYPE_UNSPECIFIED":
      return ErrorType.ERROR_TYPE_UNSPECIFIED;
    case 1:
    case "ERROR_TYPE_CATEGORICAL":
      return ErrorType.ERROR_TYPE_CATEGORICAL;
    case 2:
    case "ERROR_TYPE_METHODOLOGY":
      return ErrorType.ERROR_TYPE_METHODOLOGY;
    case 3:
    case "ERROR_TYPE_FACTUAL":
      return ErrorType.ERROR_TYPE_FACTUAL;
    case 4:
    case "ERROR_TYPE_REASONING":
      return ErrorType.ERROR_TYPE_REASONING;
    case 5:
    case "ERROR_TYPE_OMISSION":
      return ErrorType.ERROR_TYPE_OMISSION;
    case 6:
    case "ERROR_TYPE_CIRCULAR":
      return ErrorType.ERROR_TYPE_CIRCULAR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ErrorType.UNRECOGNIZED;
  }
}
export function errorTypeToJSON(object: ErrorType): string {
  switch (object) {
    case ErrorType.ERROR_TYPE_UNSPECIFIED:
      return "ERROR_TYPE_UNSPECIFIED";
    case ErrorType.ERROR_TYPE_CATEGORICAL:
      return "ERROR_TYPE_CATEGORICAL";
    case ErrorType.ERROR_TYPE_METHODOLOGY:
      return "ERROR_TYPE_METHODOLOGY";
    case ErrorType.ERROR_TYPE_FACTUAL:
      return "ERROR_TYPE_FACTUAL";
    case ErrorType.ERROR_TYPE_REASONING:
      return "ERROR_TYPE_REASONING";
    case ErrorType.ERROR_TYPE_OMISSION:
      return "ERROR_TYPE_OMISSION";
    case ErrorType.ERROR_TYPE_CIRCULAR:
      return "ERROR_TYPE_CIRCULAR";
    case ErrorType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum CounterexampleStatus {
  COUNTEREXAMPLE_STATUS_UNSPECIFIED = 0,
  COUNTEREXAMPLE_STATUS_PROPOSED = 1,
  COUNTEREXAMPLE_STATUS_VALIDATED = 2,
  COUNTEREXAMPLE_STATUS_REJECTED = 3,
  UNRECOGNIZED = -1,
}
export function counterexampleStatusFromJSON(object: any): CounterexampleStatus {
  switch (object) {
    case 0:
    case "COUNTEREXAMPLE_STATUS_UNSPECIFIED":
      return CounterexampleStatus.COUNTEREXAMPLE_STATUS_UNSPECIFIED;
    case 1:
    case "COUNTEREXAMPLE_STATUS_PROPOSED":
      return CounterexampleStatus.COUNTEREXAMPLE_STATUS_PROPOSED;
    case 2:
    case "COUNTEREXAMPLE_STATUS_VALIDATED":
      return CounterexampleStatus.COUNTEREXAMPLE_STATUS_VALIDATED;
    case 3:
    case "COUNTEREXAMPLE_STATUS_REJECTED":
      return CounterexampleStatus.COUNTEREXAMPLE_STATUS_REJECTED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return CounterexampleStatus.UNRECOGNIZED;
  }
}
export function counterexampleStatusToJSON(object: CounterexampleStatus): string {
  switch (object) {
    case CounterexampleStatus.COUNTEREXAMPLE_STATUS_UNSPECIFIED:
      return "COUNTEREXAMPLE_STATUS_UNSPECIFIED";
    case CounterexampleStatus.COUNTEREXAMPLE_STATUS_PROPOSED:
      return "COUNTEREXAMPLE_STATUS_PROPOSED";
    case CounterexampleStatus.COUNTEREXAMPLE_STATUS_VALIDATED:
      return "COUNTEREXAMPLE_STATUS_VALIDATED";
    case CounterexampleStatus.COUNTEREXAMPLE_STATUS_REJECTED:
      return "COUNTEREXAMPLE_STATUS_REJECTED";
    case CounterexampleStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
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
function createBaseCounterexample(): Counterexample {
  return {
    id: "",
    factId: "",
    author: "",
    wrongClaim: "",
    reasoning: "",
    errorType: 0,
    violatedMethodologyIds: [],
    submittedAtBlock: BigInt(0),
    status: 0,
    validations: 0,
    rejections: 0,
    resolvedAtBlock: BigInt(0),
    bond: ""
  };
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
export const Counterexample = {
  typeUrl: "/zerone.counterexamples.v1.Counterexample",
  encode(message: Counterexample, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.author !== "") {
      writer.uint32(26).string(message.author);
    }
    if (message.wrongClaim !== "") {
      writer.uint32(34).string(message.wrongClaim);
    }
    if (message.reasoning !== "") {
      writer.uint32(42).string(message.reasoning);
    }
    if (message.errorType !== 0) {
      writer.uint32(48).int32(message.errorType);
    }
    for (const v of message.violatedMethodologyIds) {
      writer.uint32(58).string(v!);
    }
    if (message.submittedAtBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.submittedAtBlock);
    }
    if (message.status !== 0) {
      writer.uint32(72).int32(message.status);
    }
    if (message.validations !== 0) {
      writer.uint32(80).uint32(message.validations);
    }
    if (message.rejections !== 0) {
      writer.uint32(88).uint32(message.rejections);
    }
    if (message.resolvedAtBlock !== BigInt(0)) {
      writer.uint32(96).uint64(message.resolvedAtBlock);
    }
    if (message.bond !== "") {
      writer.uint32(106).string(message.bond);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Counterexample {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCounterexample();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.author = reader.string();
          break;
        case 4:
          message.wrongClaim = reader.string();
          break;
        case 5:
          message.reasoning = reader.string();
          break;
        case 6:
          message.errorType = reader.int32() as any;
          break;
        case 7:
          message.violatedMethodologyIds.push(reader.string());
          break;
        case 8:
          message.submittedAtBlock = reader.uint64();
          break;
        case 9:
          message.status = reader.int32() as any;
          break;
        case 10:
          message.validations = reader.uint32();
          break;
        case 11:
          message.rejections = reader.uint32();
          break;
        case 12:
          message.resolvedAtBlock = reader.uint64();
          break;
        case 13:
          message.bond = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Counterexample>): Counterexample {
    const message = createBaseCounterexample();
    message.id = object.id ?? "";
    message.factId = object.factId ?? "";
    message.author = object.author ?? "";
    message.wrongClaim = object.wrongClaim ?? "";
    message.reasoning = object.reasoning ?? "";
    message.errorType = object.errorType ?? 0;
    message.violatedMethodologyIds = object.violatedMethodologyIds?.map(e => e) || [];
    message.submittedAtBlock = object.submittedAtBlock !== undefined && object.submittedAtBlock !== null ? BigInt(object.submittedAtBlock.toString()) : BigInt(0);
    message.status = object.status ?? 0;
    message.validations = object.validations ?? 0;
    message.rejections = object.rejections ?? 0;
    message.resolvedAtBlock = object.resolvedAtBlock !== undefined && object.resolvedAtBlock !== null ? BigInt(object.resolvedAtBlock.toString()) : BigInt(0);
    message.bond = object.bond ?? "";
    return message;
  }
};
function createBaseValidation(): Validation {
  return {
    id: BigInt(0),
    counterexampleId: "",
    validator: "",
    affirm: false,
    reason: "",
    submittedAtBlock: BigInt(0)
  };
}
/**
 * Validation is one validator's vote on a Counterexample. The chain
 * records every vote (commitment 10: forward-only audit).
 * @name Validation
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.Validation
 */
export const Validation = {
  typeUrl: "/zerone.counterexamples.v1.Validation",
  encode(message: Validation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== BigInt(0)) {
      writer.uint32(8).uint64(message.id);
    }
    if (message.counterexampleId !== "") {
      writer.uint32(18).string(message.counterexampleId);
    }
    if (message.validator !== "") {
      writer.uint32(26).string(message.validator);
    }
    if (message.affirm === true) {
      writer.uint32(32).bool(message.affirm);
    }
    if (message.reason !== "") {
      writer.uint32(42).string(message.reason);
    }
    if (message.submittedAtBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.submittedAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Validation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseValidation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.uint64();
          break;
        case 2:
          message.counterexampleId = reader.string();
          break;
        case 3:
          message.validator = reader.string();
          break;
        case 4:
          message.affirm = reader.bool();
          break;
        case 5:
          message.reason = reader.string();
          break;
        case 6:
          message.submittedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Validation>): Validation {
    const message = createBaseValidation();
    message.id = object.id !== undefined && object.id !== null ? BigInt(object.id.toString()) : BigInt(0);
    message.counterexampleId = object.counterexampleId ?? "";
    message.validator = object.validator ?? "";
    message.affirm = object.affirm ?? false;
    message.reason = object.reason ?? "";
    message.submittedAtBlock = object.submittedAtBlock !== undefined && object.submittedAtBlock !== null ? BigInt(object.submittedAtBlock.toString()) : BigInt(0);
    return message;
  }
};