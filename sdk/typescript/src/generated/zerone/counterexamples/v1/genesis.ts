//@ts-nocheck
import { Counterexample, Validation } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    counterexamples: [],
    validations: [],
    nextCounterexampleSeq: BigInt(0),
    nextValidationId: BigInt(0)
  };
}
/**
 * @name GenesisState
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.counterexamples.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.counterexamples) {
      Counterexample.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.validations) {
      Validation.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.nextCounterexampleSeq !== BigInt(0)) {
      writer.uint32(32).uint64(message.nextCounterexampleSeq);
    }
    if (message.nextValidationId !== BigInt(0)) {
      writer.uint32(40).uint64(message.nextValidationId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.counterexamples.push(Counterexample.decode(reader, reader.uint32()));
          break;
        case 3:
          message.validations.push(Validation.decode(reader, reader.uint32()));
          break;
        case 4:
          message.nextCounterexampleSeq = reader.uint64();
          break;
        case 5:
          message.nextValidationId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = createBaseGenesisState();
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    message.counterexamples = object.counterexamples?.map(e => Counterexample.fromPartial(e)) || [];
    message.validations = object.validations?.map(e => Validation.fromPartial(e)) || [];
    message.nextCounterexampleSeq = object.nextCounterexampleSeq !== undefined && object.nextCounterexampleSeq !== null ? BigInt(object.nextCounterexampleSeq.toString()) : BigInt(0);
    message.nextValidationId = object.nextValidationId !== undefined && object.nextValidationId !== null ? BigInt(object.nextValidationId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseParams(): Params {
  return {
    proposalBond: "",
    validationReward: "",
    minVotes: 0,
    affirmThresholdBps: BigInt(0),
    maxReasonBytes: 0,
    tvwMultiplierBps: BigInt(0),
    proposalsEnabled: false
  };
}
/**
 * @name Params
 * @package zerone.counterexamples.v1
 * @see proto type: zerone.counterexamples.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.counterexamples.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalBond !== "") {
      writer.uint32(10).string(message.proposalBond);
    }
    if (message.validationReward !== "") {
      writer.uint32(18).string(message.validationReward);
    }
    if (message.minVotes !== 0) {
      writer.uint32(24).uint32(message.minVotes);
    }
    if (message.affirmThresholdBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.affirmThresholdBps);
    }
    if (message.maxReasonBytes !== 0) {
      writer.uint32(40).uint32(message.maxReasonBytes);
    }
    if (message.tvwMultiplierBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.tvwMultiplierBps);
    }
    if (message.proposalsEnabled === true) {
      writer.uint32(56).bool(message.proposalsEnabled);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Params {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalBond = reader.string();
          break;
        case 2:
          message.validationReward = reader.string();
          break;
        case 3:
          message.minVotes = reader.uint32();
          break;
        case 4:
          message.affirmThresholdBps = reader.uint64();
          break;
        case 5:
          message.maxReasonBytes = reader.uint32();
          break;
        case 6:
          message.tvwMultiplierBps = reader.uint64();
          break;
        case 7:
          message.proposalsEnabled = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Params>): Params {
    const message = createBaseParams();
    message.proposalBond = object.proposalBond ?? "";
    message.validationReward = object.validationReward ?? "";
    message.minVotes = object.minVotes ?? 0;
    message.affirmThresholdBps = object.affirmThresholdBps !== undefined && object.affirmThresholdBps !== null ? BigInt(object.affirmThresholdBps.toString()) : BigInt(0);
    message.maxReasonBytes = object.maxReasonBytes ?? 0;
    message.tvwMultiplierBps = object.tvwMultiplierBps !== undefined && object.tvwMultiplierBps !== null ? BigInt(object.tvwMultiplierBps.toString()) : BigInt(0);
    message.proposalsEnabled = object.proposalsEnabled ?? false;
    return message;
  }
};