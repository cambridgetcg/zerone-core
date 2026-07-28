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
/**
 * SlashGradient mirrors M1's graduated slashing — different failure
 * modes carry different bps slash weights. Values stored at adapter
 * registration and applied at attestation rejection paths.
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
 * AdapterRegistration is the gov-approved metadata for one adapter.
 * Adapter is a recipe (binary hash + bounds + slash); no operator role.
 * Anyone who runs the registered binary AND submits an attestation
 * earns via the UW formula.
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
