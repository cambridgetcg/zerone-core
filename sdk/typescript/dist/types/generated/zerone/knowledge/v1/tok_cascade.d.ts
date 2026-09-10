import { FactStatus } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * CascadeReplaySelector walks the disproval-graph from a DISPROVEN root.
 * Returns the disproven fact + first-hop cascaded descendants + every
 * CascadeEvent the chain recorded when the disproof landed. Optionally
 * includes vindication records and supersession-chain pointers.
 *
 * TC4 (the graph carries its disprovals) is the doctrinal binding.
 * Plan 1's RootedSubtree/AncestorCone deliberately filter out
 * CONTRADICTS|SUPERSEDES edges; CascadeReplay deliberately includes them.
 * The two selectors compose: a trainer that wants both the support-graph
 * and the disproval-graph for a fact issues two BundleToK calls.
 * @name CascadeReplaySelector
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CascadeReplaySelector
 */
export interface CascadeReplaySelector {
    /**
     * must reference a fact with Status=DISPROVEN
     */
    disprovenFactId: string;
    /**
     * cascade depth; 0 = chain default 1; cap 3
     */
    maxDepth: number;
    /**
     * ship VindicationRecord entries
     */
    includeVindications: boolean;
    /**
     * walk SUPERSEDES chain (8-deep cap)
     */
    includeSupersessions: boolean;
    /**
     * ship StatusTransition timeline per node
     */
    includeStatusHistory: boolean;
}
/**
 * CascadeEvent is the persistent record of a single descendant status flip
 * during a falsification cascade. The chain emits a `falsification_cascade`
 * voice event when this fires; the record is what makes TC4 bundle-able.
 *
 * One CascadeEvent is written per descendant per disproof. Forward-only
 * (commitment 10): records are append-only, never amended in place.
 * @name CascadeEvent
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CascadeEvent
 */
export interface CascadeEvent {
    /**
     * monotonic per disproven_fact_id
     */
    seq: bigint;
    /**
     * root of the cascade
     */
    disprovenFactId: string;
    /**
     * direct descendant whose status flipped
     */
    descendantFactId: string;
    /**
     * challenge that triggered the cascade
     */
    challengeClaimId: string;
    /**
     * SUPPORTS|REQUIRES|REFINES|GENERALIZES|CITES
     */
    edgeRelation: string;
    /**
     * VERIFIED|ACTIVE|AT_RISK
     */
    priorStatus: FactStatus;
    /**
     * CONTESTED (current cascade always)
     */
    newStatus: FactStatus;
    /**
     * height the cascade fired
     */
    blockHeight: bigint;
}
/**
 * StatusTransition is the persistent record of a single status change on
 * a fact. Forward-only: once written, never modified. The full history of
 * a fact's status is reconstructable by iterating these records by factID.
 *
 * TC4: "Fact.status is preserved in graph manifests with full transition
 * history." This is the record that makes "full transition history" real.
 * @name StatusTransition
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.StatusTransition
 */
export interface StatusTransition {
    /**
     * monotonic per fact_id
     */
    seq: bigint;
    factId: string;
    /**
     * status before this transition
     */
    priorStatus: FactStatus;
    /**
     * status after this transition
     */
    newStatus: FactStatus;
    blockHeight: bigint;
    /**
     * "verification" | "challenge_disproven" | "cascade" | "supersession"
     */
    causeEventType: string;
    /**
     * round_id | challenge_claim_id | disproven_fact_id | superseding_fact_id
     */
    causeId: string;
}
/**
 * ToKVindicationRecord is the proto mirror of the existing Go struct
 * VindicationRecord in x/knowledge/types/vindication.go. The keeper
 * writes/reads JSON for backward compatibility but converts to/from this
 * proto for bundle assembly (TC4). Named with a ToK prefix to avoid a
 * collision with the pre-existing Go struct in the same package.
 * @name ToKVindicationRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ToKVindicationRecord
 */
export interface ToKVindicationRecord {
    verifier: string;
    factId: string;
    refundAmount: string;
    bonusAmount: string;
    vindicatedAt: bigint;
    disprovenBy: string;
    roundId: string;
}
/**
 * CascadeReplaySelector walks the disproval-graph from a DISPROVEN root.
 * Returns the disproven fact + first-hop cascaded descendants + every
 * CascadeEvent the chain recorded when the disproof landed. Optionally
 * includes vindication records and supersession-chain pointers.
 *
 * TC4 (the graph carries its disprovals) is the doctrinal binding.
 * Plan 1's RootedSubtree/AncestorCone deliberately filter out
 * CONTRADICTS|SUPERSEDES edges; CascadeReplay deliberately includes them.
 * The two selectors compose: a trainer that wants both the support-graph
 * and the disproval-graph for a fact issues two BundleToK calls.
 * @name CascadeReplaySelector
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CascadeReplaySelector
 */
export declare const CascadeReplaySelector: {
    typeUrl: string;
    encode(message: CascadeReplaySelector, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CascadeReplaySelector;
    fromPartial(object: DeepPartial<CascadeReplaySelector>): CascadeReplaySelector;
};
/**
 * CascadeEvent is the persistent record of a single descendant status flip
 * during a falsification cascade. The chain emits a `falsification_cascade`
 * voice event when this fires; the record is what makes TC4 bundle-able.
 *
 * One CascadeEvent is written per descendant per disproof. Forward-only
 * (commitment 10): records are append-only, never amended in place.
 * @name CascadeEvent
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.CascadeEvent
 */
export declare const CascadeEvent: {
    typeUrl: string;
    encode(message: CascadeEvent, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CascadeEvent;
    fromPartial(object: DeepPartial<CascadeEvent>): CascadeEvent;
};
/**
 * StatusTransition is the persistent record of a single status change on
 * a fact. Forward-only: once written, never modified. The full history of
 * a fact's status is reconstructable by iterating these records by factID.
 *
 * TC4: "Fact.status is preserved in graph manifests with full transition
 * history." This is the record that makes "full transition history" real.
 * @name StatusTransition
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.StatusTransition
 */
export declare const StatusTransition: {
    typeUrl: string;
    encode(message: StatusTransition, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): StatusTransition;
    fromPartial(object: DeepPartial<StatusTransition>): StatusTransition;
};
/**
 * ToKVindicationRecord is the proto mirror of the existing Go struct
 * VindicationRecord in x/knowledge/types/vindication.go. The keeper
 * writes/reads JSON for backward compatibility but converts to/from this
 * proto for bundle assembly (TC4). Named with a ToK prefix to avoid a
 * collision with the pre-existing Go struct in the same package.
 * @name ToKVindicationRecord
 * @package zerone.knowledge.v1
 * @see proto type: zerone.knowledge.v1.ToKVindicationRecord
 */
export declare const ToKVindicationRecord: {
    typeUrl: string;
    encode(message: ToKVindicationRecord, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ToKVindicationRecord;
    fromPartial(object: DeepPartial<ToKVindicationRecord>): ToKVindicationRecord;
};
