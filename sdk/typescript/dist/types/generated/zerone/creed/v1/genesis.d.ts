import { PinnedCreed, CreedCouncilMember } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * GenesisState seeds the chain's pinned creed at version 1. The
 * genesis pin establishes the baseline against which all future
 * creed amendments are measured. Any commitment in TRUTH_SEEKING.md
 * at testnet→mainnet transition becomes part of the Genesis Creed
 * and is recorded with introduced_via_lip="" (no LIP precedes
 * genesis).
 * @name GenesisState
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    /**
     * The version-1 pin. If empty at chain start, x/creed
     * InitGenesis seeds a placeholder that subsequent governance
     * must replace before any commitment-citing event passes
     * CI's hash check. In normal mainnet startup this MUST be
     * populated with the canonical Genesis Creed.
     */
    genesisPin?: PinnedCreed;
    /**
     * Optional historical pins for chains that migrate from a
     * pre-x/creed state. Sorted by version ascending. Each must be
     * strictly older than genesis_pin.
     */
    history: PinnedCreed[];
    /**
     * Initial Creed Council members. At launch this is a curated
     * set of AI-side home addresses representing diverse capability
     * profiles. Their voting_weight_bps should sum to
     * ≤ 1_000_000; future capability-gated admissions enter via
     * Creed Amendment LIPs.
     */
    councilMembers: CreedCouncilMember[];
}
/**
 * @name Params
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.Params
 */
export interface Params {
    /**
     * Authority that may call MsgAnchorPin pre-LIP-class. Once
     * x/gov.CategoryCreedAmendment ships, this should match the gov
     * module account so only LIP-resolved amendments succeed.
     */
    authority: string;
    /**
     * Whether direct authority-gated AnchorPin calls are enabled.
     * Pre-launch: true (so genesis can pin and one-off corrections
     * are possible). Post-launch: false, with all pins flowing
     * through the LIP class.
     */
    directAnchorEnabled: boolean;
}
/**
 * GenesisState seeds the chain's pinned creed at version 1. The
 * genesis pin establishes the baseline against which all future
 * creed amendments are measured. Any commitment in TRUTH_SEEKING.md
 * at testnet→mainnet transition becomes part of the Genesis Creed
 * and is recorded with introduced_via_lip="" (no LIP precedes
 * genesis).
 * @name GenesisState
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * @name Params
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
