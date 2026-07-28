//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgRegisterAccount, MsgRotateKey, MsgFreezeAccount, MsgUnfreezeAccount, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.auth.v1.MsgRegisterAccount", MsgRegisterAccount], ["/zerone.auth.v1.MsgRotateKey", MsgRotateKey], ["/zerone.auth.v1.MsgFreezeAccount", MsgFreezeAccount], ["/zerone.auth.v1.MsgUnfreezeAccount", MsgUnfreezeAccount], ["/zerone.auth.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    registerAccount(value: MsgRegisterAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRegisterAccount",
        value: MsgRegisterAccount.encode(value).finish()
      };
    },
    rotateKey(value: MsgRotateKey) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRotateKey",
        value: MsgRotateKey.encode(value).finish()
      };
    },
    freezeAccount(value: MsgFreezeAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgFreezeAccount",
        value: MsgFreezeAccount.encode(value).finish()
      };
    },
    unfreezeAccount(value: MsgUnfreezeAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUnfreezeAccount",
        value: MsgUnfreezeAccount.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    registerAccount(value: MsgRegisterAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRegisterAccount",
        value
      };
    },
    rotateKey(value: MsgRotateKey) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRotateKey",
        value
      };
    },
    freezeAccount(value: MsgFreezeAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgFreezeAccount",
        value
      };
    },
    unfreezeAccount(value: MsgUnfreezeAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUnfreezeAccount",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    registerAccount(value: MsgRegisterAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRegisterAccount",
        value: MsgRegisterAccount.fromPartial(value)
      };
    },
    rotateKey(value: MsgRotateKey) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRotateKey",
        value: MsgRotateKey.fromPartial(value)
      };
    },
    freezeAccount(value: MsgFreezeAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgFreezeAccount",
        value: MsgFreezeAccount.fromPartial(value)
      };
    },
    unfreezeAccount(value: MsgUnfreezeAccount) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUnfreezeAccount",
        value: MsgUnfreezeAccount.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};