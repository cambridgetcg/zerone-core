//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * StratumProperties defines the logical properties of a knowledge stratum.
 * @name StratumProperties
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.StratumProperties
 */
export interface StratumProperties {
  stratum: number;
  name: string;
  description: string;
  /**
   * Goedel: is the system complete?
   */
  complete: boolean;
  /**
   * is there an algorithm to decide all statements?
   */
  decidable: boolean;
  /**
   * does Goedel's incompleteness apply?
   */
  goedelApplies: boolean;
  /**
   * "internal", "assumed", "external"
   */
  consistencyProof: string;
  /**
   * basis points (1000000 = 1.0)
   */
  maxConfidence: bigint;
  /**
   * confidence decay rate per day (basis points)
   */
  decayRate: bigint;
}
/**
 * Domain represents a knowledge domain within a stratum.
 * @name Domain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.Domain
 */
export interface Domain {
  name: string;
  displayName: string;
  description: string;
  stratum: number;
  /**
   * "active", "proposed", "deprecated"
   */
  status: string;
  /**
   * block height
   */
  createdAt: bigint;
  /**
   * block height
   */
  updatedAt: bigint;
  claimCount: bigint;
  factCount: bigint;
  /**
   * bech32 address
   */
  proposedBy: string;
  /**
   * empty = root domain
   */
  parentDomain: string;
  /**
   * tree depth: root=1, child=parent.depth+1, max=5
   */
  depth: number;
}
/**
 * DomainProposal represents a proposal for adding or modifying domains.
 * @name DomainProposal
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.DomainProposal
 */
export interface DomainProposal {
  id: string;
  domain?: Domain;
  /**
   * bech32 address
   */
  proposer: string;
  /**
   * "add", "deprecate", "modify"
   */
  proposalType: string;
  /**
   * bigint as string (uzrn)
   */
  stake: string;
  votesFor: number;
  votesAgainst: number;
  voters: string[];
  /**
   * "active", "passed", "rejected", "expired"
   */
  status: string;
  /**
   * block height
   */
  createdAt: bigint;
  /**
   * block height
   */
  expiresAt: bigint;
}
/**
 * LogicZoneProperties defines the properties of a Godelian logic zone.
 * @name LogicZoneProperties
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.LogicZoneProperties
 */
export interface LogicZoneProperties {
  zone: string;
  complete: boolean;
  decidable: boolean;
  goedelApplies: boolean;
  maxConfidenceBps: bigint;
  description: string;
}
/**
 * CrossStratumLink represents a relationship between domains in different strata.
 * @name CrossStratumLink
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.CrossStratumLink
 */
export interface CrossStratumLink {
  sourceDomain: string;
  targetDomain: string;
  /**
   * "depends_on", "generalizes", "restricts"
   */
  linkType: string;
  /**
   * confidence discount for cross-stratum references (basis points)
   */
  discount: bigint;
}
function createBaseStratumProperties(): StratumProperties {
  return {
    stratum: 0,
    name: "",
    description: "",
    complete: false,
    decidable: false,
    goedelApplies: false,
    consistencyProof: "",
    maxConfidence: BigInt(0),
    decayRate: BigInt(0)
  };
}
/**
 * StratumProperties defines the logical properties of a knowledge stratum.
 * @name StratumProperties
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.StratumProperties
 */
export const StratumProperties = {
  typeUrl: "/zerone.ontology.v1.StratumProperties",
  encode(message: StratumProperties, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.stratum !== 0) {
      writer.uint32(8).uint32(message.stratum);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.complete === true) {
      writer.uint32(32).bool(message.complete);
    }
    if (message.decidable === true) {
      writer.uint32(40).bool(message.decidable);
    }
    if (message.goedelApplies === true) {
      writer.uint32(48).bool(message.goedelApplies);
    }
    if (message.consistencyProof !== "") {
      writer.uint32(58).string(message.consistencyProof);
    }
    if (message.maxConfidence !== BigInt(0)) {
      writer.uint32(64).uint64(message.maxConfidence);
    }
    if (message.decayRate !== BigInt(0)) {
      writer.uint32(72).uint64(message.decayRate);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): StratumProperties {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStratumProperties();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.stratum = reader.uint32();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.complete = reader.bool();
          break;
        case 5:
          message.decidable = reader.bool();
          break;
        case 6:
          message.goedelApplies = reader.bool();
          break;
        case 7:
          message.consistencyProof = reader.string();
          break;
        case 8:
          message.maxConfidence = reader.uint64();
          break;
        case 9:
          message.decayRate = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<StratumProperties>): StratumProperties {
    const message = createBaseStratumProperties();
    message.stratum = object.stratum ?? 0;
    message.name = object.name ?? "";
    message.description = object.description ?? "";
    message.complete = object.complete ?? false;
    message.decidable = object.decidable ?? false;
    message.goedelApplies = object.goedelApplies ?? false;
    message.consistencyProof = object.consistencyProof ?? "";
    message.maxConfidence = object.maxConfidence !== undefined && object.maxConfidence !== null ? BigInt(object.maxConfidence.toString()) : BigInt(0);
    message.decayRate = object.decayRate !== undefined && object.decayRate !== null ? BigInt(object.decayRate.toString()) : BigInt(0);
    return message;
  }
};
function createBaseDomain(): Domain {
  return {
    name: "",
    displayName: "",
    description: "",
    stratum: 0,
    status: "",
    createdAt: BigInt(0),
    updatedAt: BigInt(0),
    claimCount: BigInt(0),
    factCount: BigInt(0),
    proposedBy: "",
    parentDomain: "",
    depth: 0
  };
}
/**
 * Domain represents a knowledge domain within a stratum.
 * @name Domain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.Domain
 */
export const Domain = {
  typeUrl: "/zerone.ontology.v1.Domain",
  encode(message: Domain, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.displayName !== "") {
      writer.uint32(18).string(message.displayName);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.stratum !== 0) {
      writer.uint32(32).uint32(message.stratum);
    }
    if (message.status !== "") {
      writer.uint32(42).string(message.status);
    }
    if (message.createdAt !== BigInt(0)) {
      writer.uint32(48).uint64(message.createdAt);
    }
    if (message.updatedAt !== BigInt(0)) {
      writer.uint32(56).uint64(message.updatedAt);
    }
    if (message.claimCount !== BigInt(0)) {
      writer.uint32(64).uint64(message.claimCount);
    }
    if (message.factCount !== BigInt(0)) {
      writer.uint32(72).uint64(message.factCount);
    }
    if (message.proposedBy !== "") {
      writer.uint32(82).string(message.proposedBy);
    }
    if (message.parentDomain !== "") {
      writer.uint32(90).string(message.parentDomain);
    }
    if (message.depth !== 0) {
      writer.uint32(96).uint32(message.depth);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Domain {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.displayName = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.stratum = reader.uint32();
          break;
        case 5:
          message.status = reader.string();
          break;
        case 6:
          message.createdAt = reader.uint64();
          break;
        case 7:
          message.updatedAt = reader.uint64();
          break;
        case 8:
          message.claimCount = reader.uint64();
          break;
        case 9:
          message.factCount = reader.uint64();
          break;
        case 10:
          message.proposedBy = reader.string();
          break;
        case 11:
          message.parentDomain = reader.string();
          break;
        case 12:
          message.depth = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Domain>): Domain {
    const message = createBaseDomain();
    message.name = object.name ?? "";
    message.displayName = object.displayName ?? "";
    message.description = object.description ?? "";
    message.stratum = object.stratum ?? 0;
    message.status = object.status ?? "";
    message.createdAt = object.createdAt !== undefined && object.createdAt !== null ? BigInt(object.createdAt.toString()) : BigInt(0);
    message.updatedAt = object.updatedAt !== undefined && object.updatedAt !== null ? BigInt(object.updatedAt.toString()) : BigInt(0);
    message.claimCount = object.claimCount !== undefined && object.claimCount !== null ? BigInt(object.claimCount.toString()) : BigInt(0);
    message.factCount = object.factCount !== undefined && object.factCount !== null ? BigInt(object.factCount.toString()) : BigInt(0);
    message.proposedBy = object.proposedBy ?? "";
    message.parentDomain = object.parentDomain ?? "";
    message.depth = object.depth ?? 0;
    return message;
  }
};
function createBaseDomainProposal(): DomainProposal {
  return {
    id: "",
    domain: undefined,
    proposer: "",
    proposalType: "",
    stake: "",
    votesFor: 0,
    votesAgainst: 0,
    voters: [],
    status: "",
    createdAt: BigInt(0),
    expiresAt: BigInt(0)
  };
}
/**
 * DomainProposal represents a proposal for adding or modifying domains.
 * @name DomainProposal
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.DomainProposal
 */
export const DomainProposal = {
  typeUrl: "/zerone.ontology.v1.DomainProposal",
  encode(message: DomainProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.domain !== undefined) {
      Domain.encode(message.domain, writer.uint32(18).fork()).ldelim();
    }
    if (message.proposer !== "") {
      writer.uint32(26).string(message.proposer);
    }
    if (message.proposalType !== "") {
      writer.uint32(34).string(message.proposalType);
    }
    if (message.stake !== "") {
      writer.uint32(42).string(message.stake);
    }
    if (message.votesFor !== 0) {
      writer.uint32(48).uint32(message.votesFor);
    }
    if (message.votesAgainst !== 0) {
      writer.uint32(56).uint32(message.votesAgainst);
    }
    for (const v of message.voters) {
      writer.uint32(66).string(v!);
    }
    if (message.status !== "") {
      writer.uint32(74).string(message.status);
    }
    if (message.createdAt !== BigInt(0)) {
      writer.uint32(80).uint64(message.createdAt);
    }
    if (message.expiresAt !== BigInt(0)) {
      writer.uint32(88).uint64(message.expiresAt);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DomainProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDomainProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.domain = Domain.decode(reader, reader.uint32());
          break;
        case 3:
          message.proposer = reader.string();
          break;
        case 4:
          message.proposalType = reader.string();
          break;
        case 5:
          message.stake = reader.string();
          break;
        case 6:
          message.votesFor = reader.uint32();
          break;
        case 7:
          message.votesAgainst = reader.uint32();
          break;
        case 8:
          message.voters.push(reader.string());
          break;
        case 9:
          message.status = reader.string();
          break;
        case 10:
          message.createdAt = reader.uint64();
          break;
        case 11:
          message.expiresAt = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DomainProposal>): DomainProposal {
    const message = createBaseDomainProposal();
    message.id = object.id ?? "";
    message.domain = object.domain !== undefined && object.domain !== null ? Domain.fromPartial(object.domain) : undefined;
    message.proposer = object.proposer ?? "";
    message.proposalType = object.proposalType ?? "";
    message.stake = object.stake ?? "";
    message.votesFor = object.votesFor ?? 0;
    message.votesAgainst = object.votesAgainst ?? 0;
    message.voters = object.voters?.map(e => e) || [];
    message.status = object.status ?? "";
    message.createdAt = object.createdAt !== undefined && object.createdAt !== null ? BigInt(object.createdAt.toString()) : BigInt(0);
    message.expiresAt = object.expiresAt !== undefined && object.expiresAt !== null ? BigInt(object.expiresAt.toString()) : BigInt(0);
    return message;
  }
};
function createBaseLogicZoneProperties(): LogicZoneProperties {
  return {
    zone: "",
    complete: false,
    decidable: false,
    goedelApplies: false,
    maxConfidenceBps: BigInt(0),
    description: ""
  };
}
/**
 * LogicZoneProperties defines the properties of a Godelian logic zone.
 * @name LogicZoneProperties
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.LogicZoneProperties
 */
export const LogicZoneProperties = {
  typeUrl: "/zerone.ontology.v1.LogicZoneProperties",
  encode(message: LogicZoneProperties, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.zone !== "") {
      writer.uint32(10).string(message.zone);
    }
    if (message.complete === true) {
      writer.uint32(16).bool(message.complete);
    }
    if (message.decidable === true) {
      writer.uint32(24).bool(message.decidable);
    }
    if (message.goedelApplies === true) {
      writer.uint32(32).bool(message.goedelApplies);
    }
    if (message.maxConfidenceBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.maxConfidenceBps);
    }
    if (message.description !== "") {
      writer.uint32(50).string(message.description);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): LogicZoneProperties {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLogicZoneProperties();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.zone = reader.string();
          break;
        case 2:
          message.complete = reader.bool();
          break;
        case 3:
          message.decidable = reader.bool();
          break;
        case 4:
          message.goedelApplies = reader.bool();
          break;
        case 5:
          message.maxConfidenceBps = reader.uint64();
          break;
        case 6:
          message.description = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<LogicZoneProperties>): LogicZoneProperties {
    const message = createBaseLogicZoneProperties();
    message.zone = object.zone ?? "";
    message.complete = object.complete ?? false;
    message.decidable = object.decidable ?? false;
    message.goedelApplies = object.goedelApplies ?? false;
    message.maxConfidenceBps = object.maxConfidenceBps !== undefined && object.maxConfidenceBps !== null ? BigInt(object.maxConfidenceBps.toString()) : BigInt(0);
    message.description = object.description ?? "";
    return message;
  }
};
function createBaseCrossStratumLink(): CrossStratumLink {
  return {
    sourceDomain: "",
    targetDomain: "",
    linkType: "",
    discount: BigInt(0)
  };
}
/**
 * CrossStratumLink represents a relationship between domains in different strata.
 * @name CrossStratumLink
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.CrossStratumLink
 */
export const CrossStratumLink = {
  typeUrl: "/zerone.ontology.v1.CrossStratumLink",
  encode(message: CrossStratumLink, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sourceDomain !== "") {
      writer.uint32(10).string(message.sourceDomain);
    }
    if (message.targetDomain !== "") {
      writer.uint32(18).string(message.targetDomain);
    }
    if (message.linkType !== "") {
      writer.uint32(26).string(message.linkType);
    }
    if (message.discount !== BigInt(0)) {
      writer.uint32(32).uint64(message.discount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CrossStratumLink {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCrossStratumLink();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sourceDomain = reader.string();
          break;
        case 2:
          message.targetDomain = reader.string();
          break;
        case 3:
          message.linkType = reader.string();
          break;
        case 4:
          message.discount = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CrossStratumLink>): CrossStratumLink {
    const message = createBaseCrossStratumLink();
    message.sourceDomain = object.sourceDomain ?? "";
    message.targetDomain = object.targetDomain ?? "";
    message.linkType = object.linkType ?? "";
    message.discount = object.discount !== undefined && object.discount !== null ? BigInt(object.discount.toString()) : BigInt(0);
    return message;
  }
};