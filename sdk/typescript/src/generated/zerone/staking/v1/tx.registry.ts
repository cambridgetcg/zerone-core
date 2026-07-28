//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgRegisterValidator, MsgDelegate, MsgUndelegate, MsgRedelegate, MsgUpdateValidatorStake, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.staking.v1.MsgRegisterValidator", MsgRegisterValidator], ["/zerone.staking.v1.MsgDelegate", MsgDelegate], ["/zerone.staking.v1.MsgUndelegate", MsgUndelegate], ["/zerone.staking.v1.MsgRedelegate", MsgRedelegate], ["/zerone.staking.v1.MsgUpdateValidatorStake", MsgUpdateValidatorStake], ["/zerone.staking.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    registerValidator(value: MsgRegisterValidator) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRegisterValidator",
        value: MsgRegisterValidator.encode(value).finish()
      };
    },
    delegate(value: MsgDelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgDelegate",
        value: MsgDelegate.encode(value).finish()
      };
    },
    undelegate(value: MsgUndelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUndelegate",
        value: MsgUndelegate.encode(value).finish()
      };
    },
    redelegate(value: MsgRedelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRedelegate",
        value: MsgRedelegate.encode(value).finish()
      };
    },
    updateValidatorStake(value: MsgUpdateValidatorStake) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStake",
        value: MsgUpdateValidatorStake.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    registerValidator(value: MsgRegisterValidator) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRegisterValidator",
        value
      };
    },
    delegate(value: MsgDelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgDelegate",
        value
      };
    },
    undelegate(value: MsgUndelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUndelegate",
        value
      };
    },
    redelegate(value: MsgRedelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRedelegate",
        value
      };
    },
    updateValidatorStake(value: MsgUpdateValidatorStake) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStake",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    registerValidator(value: MsgRegisterValidator) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRegisterValidator",
        value: MsgRegisterValidator.fromPartial(value)
      };
    },
    delegate(value: MsgDelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgDelegate",
        value: MsgDelegate.fromPartial(value)
      };
    },
    undelegate(value: MsgUndelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUndelegate",
        value: MsgUndelegate.fromPartial(value)
      };
    },
    redelegate(value: MsgRedelegate) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRedelegate",
        value: MsgRedelegate.fromPartial(value)
      };
    },
    updateValidatorStake(value: MsgUpdateValidatorStake) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStake",
        value: MsgUpdateValidatorStake.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};