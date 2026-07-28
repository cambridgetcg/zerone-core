//@ts-nocheck
import { AxisBounds } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * AdapterStatus governs registration lifecycle. ACTIVE accepts new
 * attestations; SUSPENDED refuses new but in-flight settle;
 * TOMBSTONED is permanent retirement (commitment 10 forward-only).
 */
export enum AdapterStatus {
  ADAPTER_STATUS_UNSPECIFIED = 0,
  ADAPTER_STATUS_ACTIVE = 1,
  ADAPTER_STATUS_SUSPENDED = 2,
  ADAPTER_STATUS_TOMBSTONED = 3,
  UNRECOGNIZED = -1,
}
export function adapterStatusFromJSON(object: any): AdapterStatus {
  switch (object) {
    case 0:
    case "ADAPTER_STATUS_UNSPECIFIED":
      return AdapterStatus.ADAPTER_STATUS_UNSPECIFIED;
    case 1:
    case "ADAPTER_STATUS_ACTIVE":
      return AdapterStatus.ADAPTER_STATUS_ACTIVE;
    case 2:
    case "ADAPTER_STATUS_SUSPENDED":
      return AdapterStatus.ADAPTER_STATUS_SUSPENDED;
    case 3:
    case "ADAPTER_STATUS_TOMBSTONED":
      return AdapterStatus.ADAPTER_STATUS_TOMBSTONED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return AdapterStatus.UNRECOGNIZED;
  }
}
export function adapterStatusToJSON(object: AdapterStatus): string {
  switch (object) {
    case AdapterStatus.ADAPTER_STATUS_UNSPECIFIED:
      return "ADAPTER_STATUS_UNSPECIFIED";
    case AdapterStatus.ADAPTER_STATUS_ACTIVE:
      return "ADAPTER_STATUS_ACTIVE";
    case AdapterStatus.ADAPTER_STATUS_SUSPENDED:
      return "ADAPTER_STATUS_SUSPENDED";
    case AdapterStatus.ADAPTER_STATUS_TOMBSTONED:
      return "ADAPTER_STATUS_TOMBSTONED";
    case AdapterStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * QualificationStatus mirrors x/qualification's status enum so an
 * adapter can specify the minimum status a submitter must hold in the
 * required domain. Imported here as a uint32 for proto isolation;
 * keeper resolves to x/qualification's enum at query time.
 */
export enum QualificationStatus {
  QUALIFICATION_STATUS_UNSPECIFIED = 0,
  QUALIFICATION_STATUS_PROBATIONARY = 1,
  QUALIFICATION_STATUS_ACTIVE = 2,
  QUALIFICATION_STATUS_DISTINGUISHED = 3,
  UNRECOGNIZED = -1,
}
export function qualificationStatusFromJSON(object: any): QualificationStatus {
  switch (object) {
    case 0:
    case "QUALIFICATION_STATUS_UNSPECIFIED":
      return QualificationStatus.QUALIFICATION_STATUS_UNSPECIFIED;
    case 1:
    case "QUALIFICATION_STATUS_PROBATIONARY":
      return QualificationStatus.QUALIFICATION_STATUS_PROBATIONARY;
    case 2:
    case "QUALIFICATION_STATUS_ACTIVE":
      return QualificationStatus.QUALIFICATION_STATUS_ACTIVE;
    case 3:
    case "QUALIFICATION_STATUS_DISTINGUISHED":
      return QualificationStatus.QUALIFICATION_STATUS_DISTINGUISHED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return QualificationStatus.UNRECOGNIZED;
  }
}
export function qualificationStatusToJSON(object: QualificationStatus): string {
  switch (object) {
    case QualificationStatus.QUALIFICATION_STATUS_UNSPECIFIED:
      return "QUALIFICATION_STATUS_UNSPECIFIED";
    case QualificationStatus.QUALIFICATION_STATUS_PROBATIONARY:
      return "QUALIFICATION_STATUS_PROBATIONARY";
    case QualificationStatus.QUALIFICATION_STATUS_ACTIVE:
      return "QUALIFICATION_STATUS_ACTIVE";
    case QualificationStatus.QUALIFICATION_STATUS_DISTINGUISHED:
      return "QUALIFICATION_STATUS_DISTINGUISHED";
    case QualificationStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * SlashGradient mirrors M1's graduated slashing — different failure
 * modes carry different bps slash weights. Values stored at adapter
 * registration and applied at attestation rejection paths.
 * @name SlashGradient
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.SlashGradient
 */
export interface SlashGradient {
  /**
   * adapter-binary mismatch — typically 10000 (full)
   */
  compilerDriftBps: number;
  /**
   * axis claim exceeds bounds — typically pro-rata
   */
  axisOverflowBps: number;
  /**
   * > rejection threshold reached — typically 10000
   */
  fraudBps: number;
}
/**
 * AdapterRegistration is the gov-approved metadata for one adapter.
 * Adapter is a recipe (binary hash + bounds + slash); no operator role.
 * Anyone who runs the registered binary AND submits an attestation
 * earns via the UW formula.
 * @name AdapterRegistration
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AdapterRegistration
 */
export interface AdapterRegistration {
  /**
   * canonical, gov-approved (e.g. "wikipedia-en-v1")
   */
  adapterId: string;
  /**
   * "wikipedia" | "arxiv" | "ibc_packet" | etc.
   */
  sourceType: string;
  /**
   * semver
   */
  version: string;
  /**
   * determinism guarantee
   */
  compilerBinaryHash: Uint8Array;
  axisBounds?: AxisBounds;
  minAttestationBondUzrn: string;
  minPerClaimBondUzrn: string;
  slashGradient?: SlashGradient;
  requiredQualificationDomain: string;
  minQualificationStatus: QualificationStatus;
  /**
   * empty = any class allowed
   */
  allowedClassIds: string[];
  status: AdapterStatus;
  registeredViaLipId: string;
  registeredAtBlock: bigint;
  /**
   * 0 if not tombstoned
   */
  tombstonedAtBlock: bigint;
  /**
   * Witness reward: uzrn minted (cap-gated) per witness-only attestation
   * settled through this adapter. Escrowed for the challenge window before
   * release — tombstoning the adapter inside the window cancels unpaid
   * rewards (issuance follows survival, not acceptance). Empty or "0"
   * means witness-only attestations return their bond and pay nothing.
   */
  witnessRewardUzrn: string;
}
function createBaseSlashGradient(): SlashGradient {
  return {
    compilerDriftBps: 0,
    axisOverflowBps: 0,
    fraudBps: 0
  };
}
/**
 * SlashGradient mirrors M1's graduated slashing — different failure
 * modes carry different bps slash weights. Values stored at adapter
 * registration and applied at attestation rejection paths.
 * @name SlashGradient
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.SlashGradient
 */
export const SlashGradient = {
  typeUrl: "/zerone.substrate_bridge.v1.SlashGradient",
  encode(message: SlashGradient, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.compilerDriftBps !== 0) {
      writer.uint32(8).uint32(message.compilerDriftBps);
    }
    if (message.axisOverflowBps !== 0) {
      writer.uint32(16).uint32(message.axisOverflowBps);
    }
    if (message.fraudBps !== 0) {
      writer.uint32(24).uint32(message.fraudBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): SlashGradient {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSlashGradient();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.compilerDriftBps = reader.uint32();
          break;
        case 2:
          message.axisOverflowBps = reader.uint32();
          break;
        case 3:
          message.fraudBps = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<SlashGradient>): SlashGradient {
    const message = createBaseSlashGradient();
    message.compilerDriftBps = object.compilerDriftBps ?? 0;
    message.axisOverflowBps = object.axisOverflowBps ?? 0;
    message.fraudBps = object.fraudBps ?? 0;
    return message;
  }
};
function createBaseAdapterRegistration(): AdapterRegistration {
  return {
    adapterId: "",
    sourceType: "",
    version: "",
    compilerBinaryHash: new Uint8Array(),
    axisBounds: undefined,
    minAttestationBondUzrn: "",
    minPerClaimBondUzrn: "",
    slashGradient: undefined,
    requiredQualificationDomain: "",
    minQualificationStatus: 0,
    allowedClassIds: [],
    status: 0,
    registeredViaLipId: "",
    registeredAtBlock: BigInt(0),
    tombstonedAtBlock: BigInt(0),
    witnessRewardUzrn: ""
  };
}
/**
 * AdapterRegistration is the gov-approved metadata for one adapter.
 * Adapter is a recipe (binary hash + bounds + slash); no operator role.
 * Anyone who runs the registered binary AND submits an attestation
 * earns via the UW formula.
 * @name AdapterRegistration
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AdapterRegistration
 */
export const AdapterRegistration = {
  typeUrl: "/zerone.substrate_bridge.v1.AdapterRegistration",
  encode(message: AdapterRegistration, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.adapterId !== "") {
      writer.uint32(10).string(message.adapterId);
    }
    if (message.sourceType !== "") {
      writer.uint32(18).string(message.sourceType);
    }
    if (message.version !== "") {
      writer.uint32(26).string(message.version);
    }
    if (message.compilerBinaryHash.length !== 0) {
      writer.uint32(34).bytes(message.compilerBinaryHash);
    }
    if (message.axisBounds !== undefined) {
      AxisBounds.encode(message.axisBounds, writer.uint32(42).fork()).ldelim();
    }
    if (message.minAttestationBondUzrn !== "") {
      writer.uint32(50).string(message.minAttestationBondUzrn);
    }
    if (message.minPerClaimBondUzrn !== "") {
      writer.uint32(58).string(message.minPerClaimBondUzrn);
    }
    if (message.slashGradient !== undefined) {
      SlashGradient.encode(message.slashGradient, writer.uint32(66).fork()).ldelim();
    }
    if (message.requiredQualificationDomain !== "") {
      writer.uint32(74).string(message.requiredQualificationDomain);
    }
    if (message.minQualificationStatus !== 0) {
      writer.uint32(80).int32(message.minQualificationStatus);
    }
    for (const v of message.allowedClassIds) {
      writer.uint32(90).string(v!);
    }
    if (message.status !== 0) {
      writer.uint32(96).int32(message.status);
    }
    if (message.registeredViaLipId !== "") {
      writer.uint32(106).string(message.registeredViaLipId);
    }
    if (message.registeredAtBlock !== BigInt(0)) {
      writer.uint32(112).uint64(message.registeredAtBlock);
    }
    if (message.tombstonedAtBlock !== BigInt(0)) {
      writer.uint32(120).uint64(message.tombstonedAtBlock);
    }
    if (message.witnessRewardUzrn !== "") {
      writer.uint32(130).string(message.witnessRewardUzrn);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AdapterRegistration {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAdapterRegistration();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.adapterId = reader.string();
          break;
        case 2:
          message.sourceType = reader.string();
          break;
        case 3:
          message.version = reader.string();
          break;
        case 4:
          message.compilerBinaryHash = reader.bytes();
          break;
        case 5:
          message.axisBounds = AxisBounds.decode(reader, reader.uint32());
          break;
        case 6:
          message.minAttestationBondUzrn = reader.string();
          break;
        case 7:
          message.minPerClaimBondUzrn = reader.string();
          break;
        case 8:
          message.slashGradient = SlashGradient.decode(reader, reader.uint32());
          break;
        case 9:
          message.requiredQualificationDomain = reader.string();
          break;
        case 10:
          message.minQualificationStatus = reader.int32() as any;
          break;
        case 11:
          message.allowedClassIds.push(reader.string());
          break;
        case 12:
          message.status = reader.int32() as any;
          break;
        case 13:
          message.registeredViaLipId = reader.string();
          break;
        case 14:
          message.registeredAtBlock = reader.uint64();
          break;
        case 15:
          message.tombstonedAtBlock = reader.uint64();
          break;
        case 16:
          message.witnessRewardUzrn = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AdapterRegistration>): AdapterRegistration {
    const message = createBaseAdapterRegistration();
    message.adapterId = object.adapterId ?? "";
    message.sourceType = object.sourceType ?? "";
    message.version = object.version ?? "";
    message.compilerBinaryHash = object.compilerBinaryHash ?? new Uint8Array();
    message.axisBounds = object.axisBounds !== undefined && object.axisBounds !== null ? AxisBounds.fromPartial(object.axisBounds) : undefined;
    message.minAttestationBondUzrn = object.minAttestationBondUzrn ?? "";
    message.minPerClaimBondUzrn = object.minPerClaimBondUzrn ?? "";
    message.slashGradient = object.slashGradient !== undefined && object.slashGradient !== null ? SlashGradient.fromPartial(object.slashGradient) : undefined;
    message.requiredQualificationDomain = object.requiredQualificationDomain ?? "";
    message.minQualificationStatus = object.minQualificationStatus ?? 0;
    message.allowedClassIds = object.allowedClassIds?.map(e => e) || [];
    message.status = object.status ?? 0;
    message.registeredViaLipId = object.registeredViaLipId ?? "";
    message.registeredAtBlock = object.registeredAtBlock !== undefined && object.registeredAtBlock !== null ? BigInt(object.registeredAtBlock.toString()) : BigInt(0);
    message.tombstonedAtBlock = object.tombstonedAtBlock !== undefined && object.tombstonedAtBlock !== null ? BigInt(object.tombstonedAtBlock.toString()) : BigInt(0);
    message.witnessRewardUzrn = object.witnessRewardUzrn ?? "";
    return message;
  }
};