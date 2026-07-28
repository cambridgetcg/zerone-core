//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateVesting, MsgClaimVesting, MsgPauseVesting, MsgResumeVesting, MsgAccelerateVesting, MsgFalsifyVesting, MsgCompleteVesting, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.vesting_rewards.v1.MsgCreateVesting", MsgCreateVesting], ["/zerone.vesting_rewards.v1.MsgClaimVesting", MsgClaimVesting], ["/zerone.vesting_rewards.v1.MsgPauseVesting", MsgPauseVesting], ["/zerone.vesting_rewards.v1.MsgResumeVesting", MsgResumeVesting], ["/zerone.vesting_rewards.v1.MsgAccelerateVesting", MsgAccelerateVesting], ["/zerone.vesting_rewards.v1.MsgFalsifyVesting", MsgFalsifyVesting], ["/zerone.vesting_rewards.v1.MsgCompleteVesting", MsgCompleteVesting], ["/zerone.vesting_rewards.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    createVesting(value: MsgCreateVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVesting",
        value: MsgCreateVesting.encode(value).finish()
      };
    },
    claimVesting(value: MsgClaimVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVesting",
        value: MsgClaimVesting.encode(value).finish()
      };
    },
    pauseVesting(value: MsgPauseVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVesting",
        value: MsgPauseVesting.encode(value).finish()
      };
    },
    resumeVesting(value: MsgResumeVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVesting",
        value: MsgResumeVesting.encode(value).finish()
      };
    },
    accelerateVesting(value: MsgAccelerateVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVesting",
        value: MsgAccelerateVesting.encode(value).finish()
      };
    },
    falsifyVesting(value: MsgFalsifyVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVesting",
        value: MsgFalsifyVesting.encode(value).finish()
      };
    },
    completeVesting(value: MsgCompleteVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVesting",
        value: MsgCompleteVesting.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createVesting(value: MsgCreateVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVesting",
        value
      };
    },
    claimVesting(value: MsgClaimVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVesting",
        value
      };
    },
    pauseVesting(value: MsgPauseVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVesting",
        value
      };
    },
    resumeVesting(value: MsgResumeVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVesting",
        value
      };
    },
    accelerateVesting(value: MsgAccelerateVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVesting",
        value
      };
    },
    falsifyVesting(value: MsgFalsifyVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVesting",
        value
      };
    },
    completeVesting(value: MsgCompleteVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVesting",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createVesting(value: MsgCreateVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVesting",
        value: MsgCreateVesting.fromPartial(value)
      };
    },
    claimVesting(value: MsgClaimVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVesting",
        value: MsgClaimVesting.fromPartial(value)
      };
    },
    pauseVesting(value: MsgPauseVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVesting",
        value: MsgPauseVesting.fromPartial(value)
      };
    },
    resumeVesting(value: MsgResumeVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVesting",
        value: MsgResumeVesting.fromPartial(value)
      };
    },
    accelerateVesting(value: MsgAccelerateVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVesting",
        value: MsgAccelerateVesting.fromPartial(value)
      };
    },
    falsifyVesting(value: MsgFalsifyVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVesting",
        value: MsgFalsifyVesting.fromPartial(value)
      };
    },
    completeVesting(value: MsgCompleteVesting) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVesting",
        value: MsgCompleteVesting.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};