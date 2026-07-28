//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgProposeDomain, MsgVoteDomainProposal, MsgUpdateDomain, MsgRegisterLogicZone, MsgAcknowledgeIncompleteness, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.ontology.v1.MsgProposeDomain", MsgProposeDomain], ["/zerone.ontology.v1.MsgVoteDomainProposal", MsgVoteDomainProposal], ["/zerone.ontology.v1.MsgUpdateDomain", MsgUpdateDomain], ["/zerone.ontology.v1.MsgRegisterLogicZone", MsgRegisterLogicZone], ["/zerone.ontology.v1.MsgAcknowledgeIncompleteness", MsgAcknowledgeIncompleteness], ["/zerone.ontology.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    proposeDomain(value: MsgProposeDomain) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgProposeDomain",
        value: MsgProposeDomain.encode(value).finish()
      };
    },
    voteDomainProposal(value: MsgVoteDomainProposal) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposal",
        value: MsgVoteDomainProposal.encode(value).finish()
      };
    },
    updateDomain(value: MsgUpdateDomain) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateDomain",
        value: MsgUpdateDomain.encode(value).finish()
      };
    },
    registerLogicZone(value: MsgRegisterLogicZone) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZone",
        value: MsgRegisterLogicZone.encode(value).finish()
      };
    },
    acknowledgeIncompleteness(value: MsgAcknowledgeIncompleteness) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompleteness",
        value: MsgAcknowledgeIncompleteness.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    proposeDomain(value: MsgProposeDomain) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgProposeDomain",
        value
      };
    },
    voteDomainProposal(value: MsgVoteDomainProposal) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposal",
        value
      };
    },
    updateDomain(value: MsgUpdateDomain) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateDomain",
        value
      };
    },
    registerLogicZone(value: MsgRegisterLogicZone) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZone",
        value
      };
    },
    acknowledgeIncompleteness(value: MsgAcknowledgeIncompleteness) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompleteness",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    proposeDomain(value: MsgProposeDomain) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgProposeDomain",
        value: MsgProposeDomain.fromPartial(value)
      };
    },
    voteDomainProposal(value: MsgVoteDomainProposal) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposal",
        value: MsgVoteDomainProposal.fromPartial(value)
      };
    },
    updateDomain(value: MsgUpdateDomain) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateDomain",
        value: MsgUpdateDomain.fromPartial(value)
      };
    },
    registerLogicZone(value: MsgRegisterLogicZone) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZone",
        value: MsgRegisterLogicZone.fromPartial(value)
      };
    },
    acknowledgeIncompleteness(value: MsgAcknowledgeIncompleteness) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompleteness",
        value: MsgAcknowledgeIncompleteness.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};