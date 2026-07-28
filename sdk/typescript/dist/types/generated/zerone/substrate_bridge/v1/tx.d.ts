import { AdapterRegistration } from "./adapter.js";
import { SubstrateLink } from "./substrate_link.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * @name MsgRegisterAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgRegisterAdapter
 */
export interface MsgRegisterAdapter {
    authority: string;
    adapter?: AdapterRegistration;
}
/**
 * @name MsgRegisterAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgRegisterAdapterResponse
 */
export interface MsgRegisterAdapterResponse {
}
/**
 * @name MsgSuspendAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSuspendAdapter
 */
export interface MsgSuspendAdapter {
    authority: string;
    adapterId: string;
    reason: string;
}
/**
 * @name MsgSuspendAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSuspendAdapterResponse
 */
export interface MsgSuspendAdapterResponse {
}
/**
 * @name MsgTombstoneAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgTombstoneAdapter
 */
export interface MsgTombstoneAdapter {
    authority: string;
    adapterId: string;
    reason: string;
}
/**
 * @name MsgTombstoneAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgTombstoneAdapterResponse
 */
export interface MsgTombstoneAdapterResponse {
}
/**
 * @name MsgSubmitExternalAttestation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSubmitExternalAttestation
 */
export interface MsgSubmitExternalAttestation {
    submitter: string;
    adapterId: string;
    workClassId: string;
    link?: SubstrateLink;
    bondUzrn: string;
}
/**
 * @name MsgSubmitExternalAttestationResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSubmitExternalAttestationResponse
 */
export interface MsgSubmitExternalAttestationResponse {
    attestationId: string;
}
/**
 * @name MsgRegisterAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgRegisterAdapter
 */
export declare const MsgRegisterAdapter: {
    typeUrl: string;
    encode(message: MsgRegisterAdapter, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterAdapter;
    fromPartial(object: DeepPartial<MsgRegisterAdapter>): MsgRegisterAdapter;
};
/**
 * @name MsgRegisterAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgRegisterAdapterResponse
 */
export declare const MsgRegisterAdapterResponse: {
    typeUrl: string;
    encode(_: MsgRegisterAdapterResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterAdapterResponse;
    fromPartial(_: DeepPartial<MsgRegisterAdapterResponse>): MsgRegisterAdapterResponse;
};
/**
 * @name MsgSuspendAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSuspendAdapter
 */
export declare const MsgSuspendAdapter: {
    typeUrl: string;
    encode(message: MsgSuspendAdapter, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSuspendAdapter;
    fromPartial(object: DeepPartial<MsgSuspendAdapter>): MsgSuspendAdapter;
};
/**
 * @name MsgSuspendAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSuspendAdapterResponse
 */
export declare const MsgSuspendAdapterResponse: {
    typeUrl: string;
    encode(_: MsgSuspendAdapterResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSuspendAdapterResponse;
    fromPartial(_: DeepPartial<MsgSuspendAdapterResponse>): MsgSuspendAdapterResponse;
};
/**
 * @name MsgTombstoneAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgTombstoneAdapter
 */
export declare const MsgTombstoneAdapter: {
    typeUrl: string;
    encode(message: MsgTombstoneAdapter, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgTombstoneAdapter;
    fromPartial(object: DeepPartial<MsgTombstoneAdapter>): MsgTombstoneAdapter;
};
/**
 * @name MsgTombstoneAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgTombstoneAdapterResponse
 */
export declare const MsgTombstoneAdapterResponse: {
    typeUrl: string;
    encode(_: MsgTombstoneAdapterResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgTombstoneAdapterResponse;
    fromPartial(_: DeepPartial<MsgTombstoneAdapterResponse>): MsgTombstoneAdapterResponse;
};
/**
 * @name MsgSubmitExternalAttestation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSubmitExternalAttestation
 */
export declare const MsgSubmitExternalAttestation: {
    typeUrl: string;
    encode(message: MsgSubmitExternalAttestation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitExternalAttestation;
    fromPartial(object: DeepPartial<MsgSubmitExternalAttestation>): MsgSubmitExternalAttestation;
};
/**
 * @name MsgSubmitExternalAttestationResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSubmitExternalAttestationResponse
 */
export declare const MsgSubmitExternalAttestationResponse: {
    typeUrl: string;
    encode(message: MsgSubmitExternalAttestationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitExternalAttestationResponse;
    fromPartial(object: DeepPartial<MsgSubmitExternalAttestationResponse>): MsgSubmitExternalAttestationResponse;
};
