//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgProposeHalt, MsgVoteHalt, MsgProposeRevert, MsgVoteRevert, MsgProposeResume, MsgVoteResume, MsgProposeRecoveryAuthorization, MsgVoteRecoveryAuthorization, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.emergency.v1.MsgProposeHalt", MsgProposeHalt], ["/zerone.emergency.v1.MsgVoteHalt", MsgVoteHalt], ["/zerone.emergency.v1.MsgProposeRevert", MsgProposeRevert], ["/zerone.emergency.v1.MsgVoteRevert", MsgVoteRevert], ["/zerone.emergency.v1.MsgProposeResume", MsgProposeResume], ["/zerone.emergency.v1.MsgVoteResume", MsgVoteResume], ["/zerone.emergency.v1.MsgProposeRecoveryAuthorization", MsgProposeRecoveryAuthorization], ["/zerone.emergency.v1.MsgVoteRecoveryAuthorization", MsgVoteRecoveryAuthorization], ["/zerone.emergency.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    proposeHalt(value: MsgProposeHalt) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeHalt",
        value: MsgProposeHalt.encode(value).finish()
      };
    },
    voteHalt(value: MsgVoteHalt) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteHalt",
        value: MsgVoteHalt.encode(value).finish()
      };
    },
    proposeRevert(value: MsgProposeRevert) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRevert",
        value: MsgProposeRevert.encode(value).finish()
      };
    },
    voteRevert(value: MsgVoteRevert) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRevert",
        value: MsgVoteRevert.encode(value).finish()
      };
    },
    proposeResume(value: MsgProposeResume) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeResume",
        value: MsgProposeResume.encode(value).finish()
      };
    },
    voteResume(value: MsgVoteResume) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteResume",
        value: MsgVoteResume.encode(value).finish()
      };
    },
    proposeRecoveryAuthorization(value: MsgProposeRecoveryAuthorization) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRecoveryAuthorization",
        value: MsgProposeRecoveryAuthorization.encode(value).finish()
      };
    },
    voteRecoveryAuthorization(value: MsgVoteRecoveryAuthorization) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRecoveryAuthorization",
        value: MsgVoteRecoveryAuthorization.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    proposeHalt(value: MsgProposeHalt) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeHalt",
        value
      };
    },
    voteHalt(value: MsgVoteHalt) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteHalt",
        value
      };
    },
    proposeRevert(value: MsgProposeRevert) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRevert",
        value
      };
    },
    voteRevert(value: MsgVoteRevert) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRevert",
        value
      };
    },
    proposeResume(value: MsgProposeResume) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeResume",
        value
      };
    },
    voteResume(value: MsgVoteResume) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteResume",
        value
      };
    },
    proposeRecoveryAuthorization(value: MsgProposeRecoveryAuthorization) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRecoveryAuthorization",
        value
      };
    },
    voteRecoveryAuthorization(value: MsgVoteRecoveryAuthorization) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRecoveryAuthorization",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    proposeHalt(value: MsgProposeHalt) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeHalt",
        value: MsgProposeHalt.fromPartial(value)
      };
    },
    voteHalt(value: MsgVoteHalt) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteHalt",
        value: MsgVoteHalt.fromPartial(value)
      };
    },
    proposeRevert(value: MsgProposeRevert) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRevert",
        value: MsgProposeRevert.fromPartial(value)
      };
    },
    voteRevert(value: MsgVoteRevert) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRevert",
        value: MsgVoteRevert.fromPartial(value)
      };
    },
    proposeResume(value: MsgProposeResume) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeResume",
        value: MsgProposeResume.fromPartial(value)
      };
    },
    voteResume(value: MsgVoteResume) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteResume",
        value: MsgVoteResume.fromPartial(value)
      };
    },
    proposeRecoveryAuthorization(value: MsgProposeRecoveryAuthorization) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRecoveryAuthorization",
        value: MsgProposeRecoveryAuthorization.fromPartial(value)
      };
    },
    voteRecoveryAuthorization(value: MsgVoteRecoveryAuthorization) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRecoveryAuthorization",
        value: MsgVoteRecoveryAuthorization.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};