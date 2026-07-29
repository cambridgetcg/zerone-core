//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
export enum PotStatus {
  POT_STATUS_UNSPECIFIED = 0,
  POT_STATUS_ACTIVE = 1,
  POT_STATUS_DEPLETED = 2,
  POT_STATUS_EXPIRED = 3,
  UNRECOGNIZED = -1,
}
export function potStatusFromJSON(object: any): PotStatus {
  switch (object) {
    case 0:
    case "POT_STATUS_UNSPECIFIED":
      return PotStatus.POT_STATUS_UNSPECIFIED;
    case 1:
    case "POT_STATUS_ACTIVE":
      return PotStatus.POT_STATUS_ACTIVE;
    case 2:
    case "POT_STATUS_DEPLETED":
      return PotStatus.POT_STATUS_DEPLETED;
    case 3:
    case "POT_STATUS_EXPIRED":
      return PotStatus.POT_STATUS_EXPIRED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return PotStatus.UNRECOGNIZED;
  }
}
export function potStatusToJSON(object: PotStatus): string {
  switch (object) {
    case PotStatus.POT_STATUS_UNSPECIFIED:
      return "POT_STATUS_UNSPECIFIED";
    case PotStatus.POT_STATUS_ACTIVE:
      return "POT_STATUS_ACTIVE";
    case PotStatus.POT_STATUS_DEPLETED:
      return "POT_STATUS_DEPLETED";
    case PotStatus.POT_STATUS_EXPIRED:
      return "POT_STATUS_EXPIRED";
    case PotStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
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
function createBaseClaimingPot(): ClaimingPot {
  return {
    id: "",
    name: "",
    totalAmount: "",
    claimedAmount: "",
    schedule: undefined,
    eligibility: undefined,
    createdAtBlock: BigInt(0),
    status: 0
  };
}
/**
 * @name ClaimingPot
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.ClaimingPot
 */
export const ClaimingPot = {
  typeUrl: "/zerone.claiming_pot.v1.ClaimingPot",
  encode(message: ClaimingPot, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.totalAmount !== "") {
      writer.uint32(26).string(message.totalAmount);
    }
    if (message.claimedAmount !== "") {
      writer.uint32(34).string(message.claimedAmount);
    }
    if (message.schedule !== undefined) {
      VestingSchedule.encode(message.schedule, writer.uint32(42).fork()).ldelim();
    }
    if (message.eligibility !== undefined) {
      EligibilityCriteria.encode(message.eligibility, writer.uint32(50).fork()).ldelim();
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(56).uint64(message.createdAtBlock);
    }
    if (message.status !== 0) {
      writer.uint32(64).int32(message.status);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ClaimingPot {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaimingPot();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.totalAmount = reader.string();
          break;
        case 4:
          message.claimedAmount = reader.string();
          break;
        case 5:
          message.schedule = VestingSchedule.decode(reader, reader.uint32());
          break;
        case 6:
          message.eligibility = EligibilityCriteria.decode(reader, reader.uint32());
          break;
        case 7:
          message.createdAtBlock = reader.uint64();
          break;
        case 8:
          message.status = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ClaimingPot>): ClaimingPot {
    const message = createBaseClaimingPot();
    message.id = object.id ?? "";
    message.name = object.name ?? "";
    message.totalAmount = object.totalAmount ?? "";
    message.claimedAmount = object.claimedAmount ?? "";
    message.schedule = object.schedule !== undefined && object.schedule !== null ? VestingSchedule.fromPartial(object.schedule) : undefined;
    message.eligibility = object.eligibility !== undefined && object.eligibility !== null ? EligibilityCriteria.fromPartial(object.eligibility) : undefined;
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.status = object.status ?? 0;
    return message;
  }
};
function createBaseVestingSchedule(): VestingSchedule {
  return {
    startBlock: BigInt(0),
    endBlock: BigInt(0),
    cliffBlocks: BigInt(0),
    periodBlocks: BigInt(0)
  };
}
/**
 * @name VestingSchedule
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.VestingSchedule
 */
export const VestingSchedule = {
  typeUrl: "/zerone.claiming_pot.v1.VestingSchedule",
  encode(message: VestingSchedule, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.startBlock !== BigInt(0)) {
      writer.uint32(8).uint64(message.startBlock);
    }
    if (message.endBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.endBlock);
    }
    if (message.cliffBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.cliffBlocks);
    }
    if (message.periodBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.periodBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): VestingSchedule {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVestingSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.startBlock = reader.uint64();
          break;
        case 2:
          message.endBlock = reader.uint64();
          break;
        case 3:
          message.cliffBlocks = reader.uint64();
          break;
        case 4:
          message.periodBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<VestingSchedule>): VestingSchedule {
    const message = createBaseVestingSchedule();
    message.startBlock = object.startBlock !== undefined && object.startBlock !== null ? BigInt(object.startBlock.toString()) : BigInt(0);
    message.endBlock = object.endBlock !== undefined && object.endBlock !== null ? BigInt(object.endBlock.toString()) : BigInt(0);
    message.cliffBlocks = object.cliffBlocks !== undefined && object.cliffBlocks !== null ? BigInt(object.cliffBlocks.toString()) : BigInt(0);
    message.periodBlocks = object.periodBlocks !== undefined && object.periodBlocks !== null ? BigInt(object.periodBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseClaim(): Claim {
  return {
    potId: "",
    claimant: "",
    amount: "",
    claimedAt: BigInt(0)
  };
}
/**
 * @name Claim
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.Claim
 */
export const Claim = {
  typeUrl: "/zerone.claiming_pot.v1.Claim",
  encode(message: Claim, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.potId !== "") {
      writer.uint32(10).string(message.potId);
    }
    if (message.claimant !== "") {
      writer.uint32(18).string(message.claimant);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.claimedAt !== BigInt(0)) {
      writer.uint32(32).uint64(message.claimedAt);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Claim {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseClaim();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.potId = reader.string();
          break;
        case 2:
          message.claimant = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.claimedAt = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Claim>): Claim {
    const message = createBaseClaim();
    message.potId = object.potId ?? "";
    message.claimant = object.claimant ?? "";
    message.amount = object.amount ?? "";
    message.claimedAt = object.claimedAt !== undefined && object.claimedAt !== null ? BigInt(object.claimedAt.toString()) : BigInt(0);
    return message;
  }
};
function createBaseEligibilityCriteria(): EligibilityCriteria {
  return {
    minStakingTier: 0,
    minRegistrationAge: BigInt(0),
    whitelist: []
  };
}
/**
 * @name EligibilityCriteria
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.EligibilityCriteria
 */
export const EligibilityCriteria = {
  typeUrl: "/zerone.claiming_pot.v1.EligibilityCriteria",
  encode(message: EligibilityCriteria, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.minStakingTier !== 0) {
      writer.uint32(8).uint32(message.minStakingTier);
    }
    if (message.minRegistrationAge !== BigInt(0)) {
      writer.uint32(16).uint64(message.minRegistrationAge);
    }
    for (const v of message.whitelist) {
      writer.uint32(26).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EligibilityCriteria {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEligibilityCriteria();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minStakingTier = reader.uint32();
          break;
        case 2:
          message.minRegistrationAge = reader.uint64();
          break;
        case 3:
          message.whitelist.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EligibilityCriteria>): EligibilityCriteria {
    const message = createBaseEligibilityCriteria();
    message.minStakingTier = object.minStakingTier ?? 0;
    message.minRegistrationAge = object.minRegistrationAge !== undefined && object.minRegistrationAge !== null ? BigInt(object.minRegistrationAge.toString()) : BigInt(0);
    message.whitelist = object.whitelist?.map(e => e) || [];
    return message;
  }
};
function createBaseParams(): Params {
  return {
    maxPotsActive: 0,
    minClaimAmount: "",
    bootstrapRegistrar: "",
    bootstrapEmissionCapUzrn: "",
    bootstrapDailyAdmissionCap: BigInt(0)
  };
}
/**
 * @name Params
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.claiming_pot.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.maxPotsActive !== 0) {
      writer.uint32(8).uint32(message.maxPotsActive);
    }
    if (message.minClaimAmount !== "") {
      writer.uint32(18).string(message.minClaimAmount);
    }
    if (message.bootstrapRegistrar !== "") {
      writer.uint32(26).string(message.bootstrapRegistrar);
    }
    if (message.bootstrapEmissionCapUzrn !== "") {
      writer.uint32(34).string(message.bootstrapEmissionCapUzrn);
    }
    if (message.bootstrapDailyAdmissionCap !== BigInt(0)) {
      writer.uint32(40).uint64(message.bootstrapDailyAdmissionCap);
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
          message.maxPotsActive = reader.uint32();
          break;
        case 2:
          message.minClaimAmount = reader.string();
          break;
        case 3:
          message.bootstrapRegistrar = reader.string();
          break;
        case 4:
          message.bootstrapEmissionCapUzrn = reader.string();
          break;
        case 5:
          message.bootstrapDailyAdmissionCap = reader.uint64();
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
    message.maxPotsActive = object.maxPotsActive ?? 0;
    message.minClaimAmount = object.minClaimAmount ?? "";
    message.bootstrapRegistrar = object.bootstrapRegistrar ?? "";
    message.bootstrapEmissionCapUzrn = object.bootstrapEmissionCapUzrn ?? "";
    message.bootstrapDailyAdmissionCap = object.bootstrapDailyAdmissionCap !== undefined && object.bootstrapDailyAdmissionCap !== null ? BigInt(object.bootstrapDailyAdmissionCap.toString()) : BigInt(0);
    return message;
  }
};