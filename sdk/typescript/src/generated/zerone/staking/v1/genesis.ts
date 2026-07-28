//@ts-nocheck
import { TierConfig, Validator, Delegation, UnbondingEntry } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
function createBaseParams(): Params {
  return {
    unbondingPeriod: BigInt(0),
    virtualStake: "",
    maxValidators: BigInt(0),
    minSelfDelegation: "",
    maxSlashesPerEpoch: BigInt(0),
    slashDecayPeriodBlocks: BigInt(0),
    maxSlashCountDeactivate: BigInt(0),
    minStakeForVerification: "",
    slashEscalationBps: BigInt(0),
    reputationCorrectDelta: BigInt(0),
    reputationIncorrectDelta: BigInt(0),
    reputationSlashDelta: BigInt(0),
    redelegationCooldownBlocks: BigInt(0),
    tierConfigs: []
  };
}
/**
 * Params defines the staking module parameters.
 * All BPS fields use 1,000,000 scale (100% = 1,000,000).
 * @name Params
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.staking.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.unbondingPeriod !== BigInt(0)) {
      writer.uint32(8).uint64(message.unbondingPeriod);
    }
    if (message.virtualStake !== "") {
      writer.uint32(18).string(message.virtualStake);
    }
    if (message.maxValidators !== BigInt(0)) {
      writer.uint32(24).uint64(message.maxValidators);
    }
    if (message.minSelfDelegation !== "") {
      writer.uint32(34).string(message.minSelfDelegation);
    }
    if (message.maxSlashesPerEpoch !== BigInt(0)) {
      writer.uint32(40).uint64(message.maxSlashesPerEpoch);
    }
    if (message.slashDecayPeriodBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.slashDecayPeriodBlocks);
    }
    if (message.maxSlashCountDeactivate !== BigInt(0)) {
      writer.uint32(56).uint64(message.maxSlashCountDeactivate);
    }
    if (message.minStakeForVerification !== "") {
      writer.uint32(66).string(message.minStakeForVerification);
    }
    if (message.slashEscalationBps !== BigInt(0)) {
      writer.uint32(72).uint64(message.slashEscalationBps);
    }
    if (message.reputationCorrectDelta !== BigInt(0)) {
      writer.uint32(80).uint64(message.reputationCorrectDelta);
    }
    if (message.reputationIncorrectDelta !== BigInt(0)) {
      writer.uint32(88).uint64(message.reputationIncorrectDelta);
    }
    if (message.reputationSlashDelta !== BigInt(0)) {
      writer.uint32(96).uint64(message.reputationSlashDelta);
    }
    if (message.redelegationCooldownBlocks !== BigInt(0)) {
      writer.uint32(104).uint64(message.redelegationCooldownBlocks);
    }
    for (const v of message.tierConfigs) {
      TierConfig.encode(v!, writer.uint32(114).fork()).ldelim();
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
          message.unbondingPeriod = reader.uint64();
          break;
        case 2:
          message.virtualStake = reader.string();
          break;
        case 3:
          message.maxValidators = reader.uint64();
          break;
        case 4:
          message.minSelfDelegation = reader.string();
          break;
        case 5:
          message.maxSlashesPerEpoch = reader.uint64();
          break;
        case 6:
          message.slashDecayPeriodBlocks = reader.uint64();
          break;
        case 7:
          message.maxSlashCountDeactivate = reader.uint64();
          break;
        case 8:
          message.minStakeForVerification = reader.string();
          break;
        case 9:
          message.slashEscalationBps = reader.uint64();
          break;
        case 10:
          message.reputationCorrectDelta = reader.uint64();
          break;
        case 11:
          message.reputationIncorrectDelta = reader.uint64();
          break;
        case 12:
          message.reputationSlashDelta = reader.uint64();
          break;
        case 13:
          message.redelegationCooldownBlocks = reader.uint64();
          break;
        case 14:
          message.tierConfigs.push(TierConfig.decode(reader, reader.uint32()));
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
    message.unbondingPeriod = object.unbondingPeriod !== undefined && object.unbondingPeriod !== null ? BigInt(object.unbondingPeriod.toString()) : BigInt(0);
    message.virtualStake = object.virtualStake ?? "";
    message.maxValidators = object.maxValidators !== undefined && object.maxValidators !== null ? BigInt(object.maxValidators.toString()) : BigInt(0);
    message.minSelfDelegation = object.minSelfDelegation ?? "";
    message.maxSlashesPerEpoch = object.maxSlashesPerEpoch !== undefined && object.maxSlashesPerEpoch !== null ? BigInt(object.maxSlashesPerEpoch.toString()) : BigInt(0);
    message.slashDecayPeriodBlocks = object.slashDecayPeriodBlocks !== undefined && object.slashDecayPeriodBlocks !== null ? BigInt(object.slashDecayPeriodBlocks.toString()) : BigInt(0);
    message.maxSlashCountDeactivate = object.maxSlashCountDeactivate !== undefined && object.maxSlashCountDeactivate !== null ? BigInt(object.maxSlashCountDeactivate.toString()) : BigInt(0);
    message.minStakeForVerification = object.minStakeForVerification ?? "";
    message.slashEscalationBps = object.slashEscalationBps !== undefined && object.slashEscalationBps !== null ? BigInt(object.slashEscalationBps.toString()) : BigInt(0);
    message.reputationCorrectDelta = object.reputationCorrectDelta !== undefined && object.reputationCorrectDelta !== null ? BigInt(object.reputationCorrectDelta.toString()) : BigInt(0);
    message.reputationIncorrectDelta = object.reputationIncorrectDelta !== undefined && object.reputationIncorrectDelta !== null ? BigInt(object.reputationIncorrectDelta.toString()) : BigInt(0);
    message.reputationSlashDelta = object.reputationSlashDelta !== undefined && object.reputationSlashDelta !== null ? BigInt(object.reputationSlashDelta.toString()) : BigInt(0);
    message.redelegationCooldownBlocks = object.redelegationCooldownBlocks !== undefined && object.redelegationCooldownBlocks !== null ? BigInt(object.redelegationCooldownBlocks.toString()) : BigInt(0);
    message.tierConfigs = object.tierConfigs?.map(e => TierConfig.fromPartial(e)) || [];
    return message;
  }
};
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    validators: [],
    delegations: [],
    unbondingEntries: [],
    unbondingSeq: BigInt(0)
  };
}
/**
 * GenesisState defines the staking module genesis state.
 * @name GenesisState
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.staking.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.validators) {
      Validator.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.delegations) {
      Delegation.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.unbondingEntries) {
      UnbondingEntry.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    if (message.unbondingSeq !== BigInt(0)) {
      writer.uint32(40).uint64(message.unbondingSeq);
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
          message.validators.push(Validator.decode(reader, reader.uint32()));
          break;
        case 3:
          message.delegations.push(Delegation.decode(reader, reader.uint32()));
          break;
        case 4:
          message.unbondingEntries.push(UnbondingEntry.decode(reader, reader.uint32()));
          break;
        case 5:
          message.unbondingSeq = reader.uint64();
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
    message.validators = object.validators?.map(e => Validator.fromPartial(e)) || [];
    message.delegations = object.delegations?.map(e => Delegation.fromPartial(e)) || [];
    message.unbondingEntries = object.unbondingEntries?.map(e => UnbondingEntry.fromPartial(e)) || [];
    message.unbondingSeq = object.unbondingSeq !== undefined && object.unbondingSeq !== null ? BigInt(object.unbondingSeq.toString()) : BigInt(0);
    return message;
  }
};