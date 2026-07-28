//@ts-nocheck
import { StratumProperties, Domain, DomainProposal, CrossStratumLink } from "./state";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * GenesisState defines the ontology module's genesis state.
 * @name GenesisState
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  strata: StratumProperties[];
  domains: Domain[];
  proposals: DomainProposal[];
  crossLinks: CrossStratumLink[];
}
/**
 * Params defines the ontology module parameters.
 * @name Params
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.Params
 */
export interface Params {
  /**
   * bigint as string (uzrn)
   */
  minProposalStake: string;
  /**
   * blocks
   */
  proposalVotingPeriod: bigint;
  minEndorsements: number;
  /**
   * basis points
   */
  crossStratumDiscount: bigint;
  maxDomainsPerStratum: number;
  allowNewStrata: boolean;
}
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    strata: [],
    domains: [],
    proposals: [],
    crossLinks: []
  };
}
/**
 * GenesisState defines the ontology module's genesis state.
 * @name GenesisState
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.ontology.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.strata) {
      StratumProperties.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.domains) {
      Domain.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.proposals) {
      DomainProposal.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.crossLinks) {
      CrossStratumLink.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.strata.push(StratumProperties.decode(reader, reader.uint32()));
          break;
        case 3:
          message.domains.push(Domain.decode(reader, reader.uint32()));
          break;
        case 4:
          message.proposals.push(DomainProposal.decode(reader, reader.uint32()));
          break;
        case 5:
          message.crossLinks.push(CrossStratumLink.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = createBaseGenesisState();
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    message.strata = object.strata?.map(e => StratumProperties.fromPartial(e)) || [];
    message.domains = object.domains?.map(e => Domain.fromPartial(e)) || [];
    message.proposals = object.proposals?.map(e => DomainProposal.fromPartial(e)) || [];
    message.crossLinks = object.crossLinks?.map(e => CrossStratumLink.fromPartial(e)) || [];
    return message;
  }
};
function createBaseParams(): Params {
  return {
    minProposalStake: "",
    proposalVotingPeriod: BigInt(0),
    minEndorsements: 0,
    crossStratumDiscount: BigInt(0),
    maxDomainsPerStratum: 0,
    allowNewStrata: false
  };
}
/**
 * Params defines the ontology module parameters.
 * @name Params
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.ontology.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.minProposalStake !== "") {
      writer.uint32(10).string(message.minProposalStake);
    }
    if (message.proposalVotingPeriod !== BigInt(0)) {
      writer.uint32(16).uint64(message.proposalVotingPeriod);
    }
    if (message.minEndorsements !== 0) {
      writer.uint32(24).uint32(message.minEndorsements);
    }
    if (message.crossStratumDiscount !== BigInt(0)) {
      writer.uint32(32).uint64(message.crossStratumDiscount);
    }
    if (message.maxDomainsPerStratum !== 0) {
      writer.uint32(40).uint32(message.maxDomainsPerStratum);
    }
    if (message.allowNewStrata === true) {
      writer.uint32(48).bool(message.allowNewStrata);
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
          message.minProposalStake = reader.string();
          break;
        case 2:
          message.proposalVotingPeriod = reader.uint64();
          break;
        case 3:
          message.minEndorsements = reader.uint32();
          break;
        case 4:
          message.crossStratumDiscount = reader.uint64();
          break;
        case 5:
          message.maxDomainsPerStratum = reader.uint32();
          break;
        case 6:
          message.allowNewStrata = reader.bool();
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
    message.minProposalStake = object.minProposalStake ?? "";
    message.proposalVotingPeriod = object.proposalVotingPeriod !== undefined && object.proposalVotingPeriod !== null ? BigInt(object.proposalVotingPeriod.toString()) : BigInt(0);
    message.minEndorsements = object.minEndorsements ?? 0;
    message.crossStratumDiscount = object.crossStratumDiscount !== undefined && object.crossStratumDiscount !== null ? BigInt(object.crossStratumDiscount.toString()) : BigInt(0);
    message.maxDomainsPerStratum = object.maxDomainsPerStratum ?? 0;
    message.allowNewStrata = object.allowNewStrata ?? false;
    return message;
  }
};