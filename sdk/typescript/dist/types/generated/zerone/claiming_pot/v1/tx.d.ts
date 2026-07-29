import { VestingSchedule, EligibilityCriteria, Params } from "./state.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * MsgCreatePot is a legacy authority-gated general native-issuance surface.
 * The keeper charges ceil(total_amount / 222,000 uzrn) against the same
 * lifetime budget used by bootstrap admissions. Prefer MsgAddBootstrapEntry
 * for the fixed, one-address participation seed.
 * @name MsgCreatePot
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgCreatePot
 */
export interface MsgCreatePot {
    authority: string;
    name: string;
    totalAmount: string;
    schedule?: VestingSchedule;
    eligibility?: EligibilityCriteria;
}
/**
 * @name MsgCreatePotResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgCreatePotResponse
 */
export interface MsgCreatePotResponse {
    potId: string;
}
/**
 * @name MsgClaim
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgClaim
 */
export interface MsgClaim {
    claimant: string;
    potId: string;
}
/**
 * @name MsgClaimResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgClaimResponse
 */
export interface MsgClaimResponse {
    amount: string;
}
/**
 * @name MsgUpdatePotParams
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgUpdatePotParams
 */
export interface MsgUpdatePotParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdatePotParamsResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgUpdatePotParamsResponse
 */
export interface MsgUpdatePotParamsResponse {
}
/**
 * MsgAddBootstrapEntry adds one or more bootstrap pots to the claiming_pot
 * module after genesis. Authority-gated — either the governance account or
 * configured BootstrapRegistrar may sign. Each address gets a
 * single-claimant ClaimingPot sized
 * PerAgentBootstrapUzrn (0.222 ZRN) at the current block height, instant
 * vest, ACTIVE status, ID = BootstrapPotIDPrefix + addr.
 *
 * Idempotent semantics: addresses with an existing bootstrap pot are
 * silently skipped (counted in skipped_count). Re-running the same request
 * does not double-mint or double-create.
 *
 * Doctrine: commitment 20 (issuance follows participation) extended from
 * "at genesis" to continuously, bounded authority-gated admission. Registrar
 * batches are rate/cap bounded and the registrar is governance-revocable.
 * @name MsgAddBootstrapEntry
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgAddBootstrapEntry
 */
export interface MsgAddBootstrapEntry {
    authority: string;
    addresses: string[];
}
/**
 * @name MsgAddBootstrapEntryResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgAddBootstrapEntryResponse
 */
export interface MsgAddBootstrapEntryResponse {
    addedCount: number;
    skippedCount: number;
}
/**
 * MsgCreatePot is a legacy authority-gated general native-issuance surface.
 * The keeper charges ceil(total_amount / 222,000 uzrn) against the same
 * lifetime budget used by bootstrap admissions. Prefer MsgAddBootstrapEntry
 * for the fixed, one-address participation seed.
 * @name MsgCreatePot
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgCreatePot
 */
export declare const MsgCreatePot: {
    typeUrl: string;
    encode(message: MsgCreatePot, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreatePot;
    fromPartial(object: DeepPartial<MsgCreatePot>): MsgCreatePot;
};
/**
 * @name MsgCreatePotResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgCreatePotResponse
 */
export declare const MsgCreatePotResponse: {
    typeUrl: string;
    encode(message: MsgCreatePotResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCreatePotResponse;
    fromPartial(object: DeepPartial<MsgCreatePotResponse>): MsgCreatePotResponse;
};
/**
 * @name MsgClaim
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgClaim
 */
export declare const MsgClaim: {
    typeUrl: string;
    encode(message: MsgClaim, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgClaim;
    fromPartial(object: DeepPartial<MsgClaim>): MsgClaim;
};
/**
 * @name MsgClaimResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgClaimResponse
 */
export declare const MsgClaimResponse: {
    typeUrl: string;
    encode(message: MsgClaimResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimResponse;
    fromPartial(object: DeepPartial<MsgClaimResponse>): MsgClaimResponse;
};
/**
 * @name MsgUpdatePotParams
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgUpdatePotParams
 */
export declare const MsgUpdatePotParams: {
    typeUrl: string;
    encode(message: MsgUpdatePotParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdatePotParams;
    fromPartial(object: DeepPartial<MsgUpdatePotParams>): MsgUpdatePotParams;
};
/**
 * @name MsgUpdatePotParamsResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgUpdatePotParamsResponse
 */
export declare const MsgUpdatePotParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdatePotParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdatePotParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdatePotParamsResponse>): MsgUpdatePotParamsResponse;
};
/**
 * MsgAddBootstrapEntry adds one or more bootstrap pots to the claiming_pot
 * module after genesis. Authority-gated — either the governance account or
 * configured BootstrapRegistrar may sign. Each address gets a
 * single-claimant ClaimingPot sized
 * PerAgentBootstrapUzrn (0.222 ZRN) at the current block height, instant
 * vest, ACTIVE status, ID = BootstrapPotIDPrefix + addr.
 *
 * Idempotent semantics: addresses with an existing bootstrap pot are
 * silently skipped (counted in skipped_count). Re-running the same request
 * does not double-mint or double-create.
 *
 * Doctrine: commitment 20 (issuance follows participation) extended from
 * "at genesis" to continuously, bounded authority-gated admission. Registrar
 * batches are rate/cap bounded and the registrar is governance-revocable.
 * @name MsgAddBootstrapEntry
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgAddBootstrapEntry
 */
export declare const MsgAddBootstrapEntry: {
    typeUrl: string;
    encode(message: MsgAddBootstrapEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddBootstrapEntry;
    fromPartial(object: DeepPartial<MsgAddBootstrapEntry>): MsgAddBootstrapEntry;
};
/**
 * @name MsgAddBootstrapEntryResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgAddBootstrapEntryResponse
 */
export declare const MsgAddBootstrapEntryResponse: {
    typeUrl: string;
    encode(message: MsgAddBootstrapEntryResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAddBootstrapEntryResponse;
    fromPartial(object: DeepPartial<MsgAddBootstrapEntryResponse>): MsgAddBootstrapEntryResponse;
};
