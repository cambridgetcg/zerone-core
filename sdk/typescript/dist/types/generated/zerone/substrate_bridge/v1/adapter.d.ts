import { AxisBounds } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * AdapterStatus governs registration lifecycle. ACTIVE accepts new
 * attestations; SUSPENDED refuses new but in-flight settle;
 * TOMBSTONED is permanent retirement (commitment 10 forward-only).
 */
export declare enum AdapterStatus {
    ADAPTER_STATUS_UNSPECIFIED = 0,
    ADAPTER_STATUS_ACTIVE = 1,
    ADAPTER_STATUS_SUSPENDED = 2,
    ADAPTER_STATUS_TOMBSTONED = 3,
    UNRECOGNIZED = -1
}
export declare function adapterStatusFromJSON(object: any): AdapterStatus;
export declare function adapterStatusToJSON(object: AdapterStatus): string;
/**
 * QualificationStatus mirrors x/qualification's status enum so an
 * adapter can specify the minimum status a submitter must hold in the
 * required domain. Imported here as a uint32 for proto isolation;
 * keeper resolves to x/qualification's enum at query time.
 */
export declare enum QualificationStatus {
    QUALIFICATION_STATUS_UNSPECIFIED = 0,
    QUALIFICATION_STATUS_PROBATIONARY = 1,
    QUALIFICATION_STATUS_ACTIVE = 2,
    QUALIFICATION_STATUS_DISTINGUISHED = 3,
    UNRECOGNIZED = -1
}
export declare function qualificationStatusFromJSON(object: any): QualificationStatus;
export declare function qualificationStatusToJSON(object: QualificationStatus): string;
/**
 * SlashGradient is retained adapter metadata for a future graduated-slashing
 * implementation. Current pre-escrow validation rejects compiler/bounds
 * failures and a settled rejection burns the full bond; these values are not
 * read by consensus.
 * @name SlashGradient
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.SlashGradient
 */
export interface SlashGradient {
    /**
     * inert metadata; 10,000 scale
     */
    compilerDriftBps: number;
    /**
     * inert metadata; 10,000 scale
     */
    axisOverflowBps: number;
    /**
     * inert metadata; 10,000 scale
     */
    fraudBps: number;
}
/**
 * AdapterRegistration is the metadata for one genesis-seeded or
 * gov-authority-registered adapter. CategoryAdapterRegistration LIP payload
 * dispatch is not wired yet; registered_via_lip_id is therefore provenance,
 * not proof that current LIP execution installed the adapter. Adapter is a
 * recipe (binary hash + bounds + slash), not an operator role.
 * @name AdapterRegistration
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AdapterRegistration
 */
export interface AdapterRegistration {
    /**
     * canonical ID (e.g. "wikipedia-en-v1")
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
     * Off-chain compiler provenance metadata. Runtime does not fetch or execute
     * the binary and does not compare this hash during attestation validation.
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
    /**
     * empty for genesis/direct authority registration
     */
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
/**
 * SlashGradient is retained adapter metadata for a future graduated-slashing
 * implementation. Current pre-escrow validation rejects compiler/bounds
 * failures and a settled rejection burns the full bond; these values are not
 * read by consensus.
 * @name SlashGradient
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.SlashGradient
 */
export declare const SlashGradient: {
    typeUrl: string;
    encode(message: SlashGradient, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): SlashGradient;
    fromPartial(object: DeepPartial<SlashGradient>): SlashGradient;
};
/**
 * AdapterRegistration is the metadata for one genesis-seeded or
 * gov-authority-registered adapter. CategoryAdapterRegistration LIP payload
 * dispatch is not wired yet; registered_via_lip_id is therefore provenance,
 * not proof that current LIP execution installed the adapter. Adapter is a
 * recipe (binary hash + bounds + slash), not an operator role.
 * @name AdapterRegistration
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AdapterRegistration
 */
export declare const AdapterRegistration: {
    typeUrl: string;
    encode(message: AdapterRegistration, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AdapterRegistration;
    fromPartial(object: DeepPartial<AdapterRegistration>): AdapterRegistration;
};
