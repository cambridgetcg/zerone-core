//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgSubmitLIP, MsgStakeLIP, MsgAdvanceLIPStage, MsgCastVote, MsgWithdrawLIP, MsgUpdateParams, MsgSubmitResearchSpend, MsgVoteResearchSpend, MsgSetResearchVoters, MsgAttachUpgradePlan, MsgAttachCreedAmendmentPin, MsgNominateSeatElection, MsgAcceptSeatNomination, MsgVoteSeatElection, MsgDomainFormationFreeze } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.gov.v1.MsgSubmitLIP", MsgSubmitLIP], ["/zerone.gov.v1.MsgStakeLIP", MsgStakeLIP], ["/zerone.gov.v1.MsgAdvanceLIPStage", MsgAdvanceLIPStage], ["/zerone.gov.v1.MsgCastVote", MsgCastVote], ["/zerone.gov.v1.MsgWithdrawLIP", MsgWithdrawLIP], ["/zerone.gov.v1.MsgUpdateParams", MsgUpdateParams], ["/zerone.gov.v1.MsgSubmitResearchSpend", MsgSubmitResearchSpend], ["/zerone.gov.v1.MsgVoteResearchSpend", MsgVoteResearchSpend], ["/zerone.gov.v1.MsgSetResearchVoters", MsgSetResearchVoters], ["/zerone.gov.v1.MsgAttachUpgradePlan", MsgAttachUpgradePlan], ["/zerone.gov.v1.MsgAttachCreedAmendmentPin", MsgAttachCreedAmendmentPin], ["/zerone.gov.v1.MsgNominateSeatElection", MsgNominateSeatElection], ["/zerone.gov.v1.MsgAcceptSeatNomination", MsgAcceptSeatNomination], ["/zerone.gov.v1.MsgVoteSeatElection", MsgVoteSeatElection], ["/zerone.gov.v1.MsgDomainFormationFreeze", MsgDomainFormationFreeze]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    submitLIP(value: MsgSubmitLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitLIP",
        value: MsgSubmitLIP.encode(value).finish()
      };
    },
    stakeLIP(value: MsgStakeLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgStakeLIP",
        value: MsgStakeLIP.encode(value).finish()
      };
    },
    advanceLIPStage(value: MsgAdvanceLIPStage) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStage",
        value: MsgAdvanceLIPStage.encode(value).finish()
      };
    },
    castVote(value: MsgCastVote) {
      return {
        typeUrl: "/zerone.gov.v1.MsgCastVote",
        value: MsgCastVote.encode(value).finish()
      };
    },
    withdrawLIP(value: MsgWithdrawLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgWithdrawLIP",
        value: MsgWithdrawLIP.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.gov.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    },
    submitResearchSpend(value: MsgSubmitResearchSpend) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpend",
        value: MsgSubmitResearchSpend.encode(value).finish()
      };
    },
    voteResearchSpend(value: MsgVoteResearchSpend) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteResearchSpend",
        value: MsgVoteResearchSpend.encode(value).finish()
      };
    },
    setResearchVoters(value: MsgSetResearchVoters) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSetResearchVoters",
        value: MsgSetResearchVoters.encode(value).finish()
      };
    },
    attachUpgradePlan(value: MsgAttachUpgradePlan) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlan",
        value: MsgAttachUpgradePlan.encode(value).finish()
      };
    },
    attachCreedAmendmentPin(value: MsgAttachCreedAmendmentPin) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPin",
        value: MsgAttachCreedAmendmentPin.encode(value).finish()
      };
    },
    nominateSeatElection(value: MsgNominateSeatElection) {
      return {
        typeUrl: "/zerone.gov.v1.MsgNominateSeatElection",
        value: MsgNominateSeatElection.encode(value).finish()
      };
    },
    acceptSeatNomination(value: MsgAcceptSeatNomination) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAcceptSeatNomination",
        value: MsgAcceptSeatNomination.encode(value).finish()
      };
    },
    voteSeatElection(value: MsgVoteSeatElection) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteSeatElection",
        value: MsgVoteSeatElection.encode(value).finish()
      };
    },
    domainFormationFreeze(value: MsgDomainFormationFreeze) {
      return {
        typeUrl: "/zerone.gov.v1.MsgDomainFormationFreeze",
        value: MsgDomainFormationFreeze.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    submitLIP(value: MsgSubmitLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitLIP",
        value
      };
    },
    stakeLIP(value: MsgStakeLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgStakeLIP",
        value
      };
    },
    advanceLIPStage(value: MsgAdvanceLIPStage) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStage",
        value
      };
    },
    castVote(value: MsgCastVote) {
      return {
        typeUrl: "/zerone.gov.v1.MsgCastVote",
        value
      };
    },
    withdrawLIP(value: MsgWithdrawLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgWithdrawLIP",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.gov.v1.MsgUpdateParams",
        value
      };
    },
    submitResearchSpend(value: MsgSubmitResearchSpend) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpend",
        value
      };
    },
    voteResearchSpend(value: MsgVoteResearchSpend) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteResearchSpend",
        value
      };
    },
    setResearchVoters(value: MsgSetResearchVoters) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSetResearchVoters",
        value
      };
    },
    attachUpgradePlan(value: MsgAttachUpgradePlan) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlan",
        value
      };
    },
    attachCreedAmendmentPin(value: MsgAttachCreedAmendmentPin) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPin",
        value
      };
    },
    nominateSeatElection(value: MsgNominateSeatElection) {
      return {
        typeUrl: "/zerone.gov.v1.MsgNominateSeatElection",
        value
      };
    },
    acceptSeatNomination(value: MsgAcceptSeatNomination) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAcceptSeatNomination",
        value
      };
    },
    voteSeatElection(value: MsgVoteSeatElection) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteSeatElection",
        value
      };
    },
    domainFormationFreeze(value: MsgDomainFormationFreeze) {
      return {
        typeUrl: "/zerone.gov.v1.MsgDomainFormationFreeze",
        value
      };
    }
  },
  fromPartial: {
    submitLIP(value: MsgSubmitLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitLIP",
        value: MsgSubmitLIP.fromPartial(value)
      };
    },
    stakeLIP(value: MsgStakeLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgStakeLIP",
        value: MsgStakeLIP.fromPartial(value)
      };
    },
    advanceLIPStage(value: MsgAdvanceLIPStage) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStage",
        value: MsgAdvanceLIPStage.fromPartial(value)
      };
    },
    castVote(value: MsgCastVote) {
      return {
        typeUrl: "/zerone.gov.v1.MsgCastVote",
        value: MsgCastVote.fromPartial(value)
      };
    },
    withdrawLIP(value: MsgWithdrawLIP) {
      return {
        typeUrl: "/zerone.gov.v1.MsgWithdrawLIP",
        value: MsgWithdrawLIP.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.gov.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    },
    submitResearchSpend(value: MsgSubmitResearchSpend) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpend",
        value: MsgSubmitResearchSpend.fromPartial(value)
      };
    },
    voteResearchSpend(value: MsgVoteResearchSpend) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteResearchSpend",
        value: MsgVoteResearchSpend.fromPartial(value)
      };
    },
    setResearchVoters(value: MsgSetResearchVoters) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSetResearchVoters",
        value: MsgSetResearchVoters.fromPartial(value)
      };
    },
    attachUpgradePlan(value: MsgAttachUpgradePlan) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlan",
        value: MsgAttachUpgradePlan.fromPartial(value)
      };
    },
    attachCreedAmendmentPin(value: MsgAttachCreedAmendmentPin) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPin",
        value: MsgAttachCreedAmendmentPin.fromPartial(value)
      };
    },
    nominateSeatElection(value: MsgNominateSeatElection) {
      return {
        typeUrl: "/zerone.gov.v1.MsgNominateSeatElection",
        value: MsgNominateSeatElection.fromPartial(value)
      };
    },
    acceptSeatNomination(value: MsgAcceptSeatNomination) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAcceptSeatNomination",
        value: MsgAcceptSeatNomination.fromPartial(value)
      };
    },
    voteSeatElection(value: MsgVoteSeatElection) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteSeatElection",
        value: MsgVoteSeatElection.fromPartial(value)
      };
    },
    domainFormationFreeze(value: MsgDomainFormationFreeze) {
      return {
        typeUrl: "/zerone.gov.v1.MsgDomainFormationFreeze",
        value: MsgDomainFormationFreeze.fromPartial(value)
      };
    }
  }
};