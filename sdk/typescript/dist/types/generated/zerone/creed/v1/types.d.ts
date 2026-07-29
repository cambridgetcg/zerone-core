import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * PinnedCreed is an on-chain anchor for a proposed canonical
 * docs/TRUTH_SEEKING.md version. Repository CI independently binds that file
 * to .creed-hash; it does not query a running network or prove that its
 * current pin matches the build. The per-commitment registry makes an
 * on-chain version's numbered structure queryable and append-only.
 *
 * See docs/TRUTH_SEEKING.md commitments 6 and 10:
 *   - 6 (no unilateral injection): direct authority anchoring can be
 *     disabled, but the published zerone-1 setting leaves it enabled.
 *   - 10 (forward-only audit): pin records are append-only by
 *     monotonic version; prior creed versions remain queryable as
 *     the chain's history of self-amendment.
 * @name PinnedCreed
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.PinnedCreed
 */
export interface PinnedCreed {
    /**
     * Monotonic, starting at 1 at genesis. version=N+1 archives
     * version=N; both remain queryable. The current pin is whichever
     * version has highest number.
     */
    version: number;
    /**
     * sha256 of normalized docs/TRUTH_SEEKING.md (CR characters
     * stripped; no other normalization applied so structural changes
     * remain visible).
     */
    canonicalHash: Uint8Array;
    /**
     * Block at which this version became canonical.
     */
    pinnedAtHeight: bigint;
    /**
     * Optional source LIP id. Empty for genesis and also possible for later
     * authority-direct pins while direct_anchor_enabled=true; this field alone
     * is not proof of governance passage.
     */
    pinnedViaLip: string;
    /**
     * Per-commitment registry at this version. Each entry names a
     * commitment by number and tracks when it entered the creed and
     * (if archived) when it left. The registry is the structural
     * protection against interpretation drift within the pin. A source LIP is
     * required only by the governance path/configuration that adopted it.
     */
    commitments: CommitmentEntry[];
}
/**
 * CreedCouncilMember is a registry/query surface intended for a future
 * AI-side voter pool for Creed Amendment LIPs.
 *
 * docs/TRUTH_SEEKING.md commitment 19 (the creed is governance-
 * gated): current ordinary LIP tally does NOT route or enforce separate
 * human/AI quorum. Membership and weight are scaffolded inputs only; source
 * must not treat this registry as an active pass condition.
 * @name CreedCouncilMember
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.CreedCouncilMember
 */
export interface CreedCouncilMember {
    /**
     * Bech32 address of the council seat. At launch, this is a
     * home address from x/home; future iterations may admit any
     * qualified agent address.
     */
    address: string;
    /**
     * Block at which this seat was admitted. Audit trail:
     * capability-gated admissions can be reconstructed from this
     * height paired with the agent_understanding state at that
     * height.
     */
    admittedAtHeight: bigint;
    /**
     * LIP id that authorized this admission. Empty for genesis-
     * installed seats (which are recorded in GenesisState directly).
     */
    admittedViaLip: string;
    /**
     * Voting weight in basis points. At launch all genesis-installed
     * seats carry equal weight (1_000_000 / N_seats); capability-
     * gated future admissions derive weight from agent_understanding
     * domain coverage and accuracy.
     */
    votingWeightBps: bigint;
    /**
     * True if the seat is currently active. Inactive seats remain
     * in the registry as historical record (commitment 10: forward-
     * only audit) but their voting_weight_bps is treated as 0 in
     * tallies. Setting active=false is the structural form of
     * archival.
     */
    active: boolean;
    /**
     * Optional admission basis label. Examples: "genesis",
     * "capability_gated:agent_understanding>0.7".
     */
    admissionBasis: string;
}
/**
 * CommitmentEntry is one numbered commitment's anchor on chain.
 * The entry binds a commitment number to its name and the LIP
 * that introduced or last amended it. A future amendment that
 * changes the meaning of commitment N must produce a new
 * PinnedCreed where the corresponding entry's introduced_via_lip
 * is the amending LIP — even if the commitment_number stays the
 * same.
 * @name CommitmentEntry
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.CommitmentEntry
 */
export interface CommitmentEntry {
    /**
     * Stable identifier across versions. 1..N.
     */
    number: number;
    /**
     * Short title. Repository invariants compare the canonical source registry
     * with TRUTH_SEEKING.md headings; runtime validates structure and stores this
     * value but cannot independently read the source document.
     */
    name: string;
    /**
     * Block at which this commitment first entered the creed (or was
     * last materially amended).
     */
    introducedAtHeight: bigint;
    /**
     * LIP id that introduced or last amended this commitment.
     * Empty for genesis-installed commitments.
     */
    introducedViaLip: string;
    /**
     * True iff this commitment has been formally archived (no longer
     * load-bearing). Archived commitments remain in the registry as
     * historical record; new code MUST NOT cite an archived number.
     */
    archived: boolean;
    /**
     * Block at which this commitment was archived. 0 if not archived.
     */
    archivedAtHeight: bigint;
}
/**
 * PinnedCreed is an on-chain anchor for a proposed canonical
 * docs/TRUTH_SEEKING.md version. Repository CI independently binds that file
 * to .creed-hash; it does not query a running network or prove that its
 * current pin matches the build. The per-commitment registry makes an
 * on-chain version's numbered structure queryable and append-only.
 *
 * See docs/TRUTH_SEEKING.md commitments 6 and 10:
 *   - 6 (no unilateral injection): direct authority anchoring can be
 *     disabled, but the published zerone-1 setting leaves it enabled.
 *   - 10 (forward-only audit): pin records are append-only by
 *     monotonic version; prior creed versions remain queryable as
 *     the chain's history of self-amendment.
 * @name PinnedCreed
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.PinnedCreed
 */
export declare const PinnedCreed: {
    typeUrl: string;
    encode(message: PinnedCreed, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): PinnedCreed;
    fromPartial(object: DeepPartial<PinnedCreed>): PinnedCreed;
};
/**
 * CreedCouncilMember is a registry/query surface intended for a future
 * AI-side voter pool for Creed Amendment LIPs.
 *
 * docs/TRUTH_SEEKING.md commitment 19 (the creed is governance-
 * gated): current ordinary LIP tally does NOT route or enforce separate
 * human/AI quorum. Membership and weight are scaffolded inputs only; source
 * must not treat this registry as an active pass condition.
 * @name CreedCouncilMember
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.CreedCouncilMember
 */
export declare const CreedCouncilMember: {
    typeUrl: string;
    encode(message: CreedCouncilMember, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CreedCouncilMember;
    fromPartial(object: DeepPartial<CreedCouncilMember>): CreedCouncilMember;
};
/**
 * CommitmentEntry is one numbered commitment's anchor on chain.
 * The entry binds a commitment number to its name and the LIP
 * that introduced or last amended it. A future amendment that
 * changes the meaning of commitment N must produce a new
 * PinnedCreed where the corresponding entry's introduced_via_lip
 * is the amending LIP — even if the commitment_number stays the
 * same.
 * @name CommitmentEntry
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.CommitmentEntry
 */
export declare const CommitmentEntry: {
    typeUrl: string;
    encode(message: CommitmentEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CommitmentEntry;
    fromPartial(object: DeepPartial<CommitmentEntry>): CommitmentEntry;
};
