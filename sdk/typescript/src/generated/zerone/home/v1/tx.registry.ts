//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateHome, MsgUpdateHome, MsgUpdateMemoryCID, MsgStartSession, MsgEndSession, MsgRegisterKey, MsgRevokeKey, MsgConfigureGuardian, MsgAcknowledgeAlert, MsgSetSpendingLimit, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.home.v1.MsgCreateHome", MsgCreateHome], ["/zerone.home.v1.MsgUpdateHome", MsgUpdateHome], ["/zerone.home.v1.MsgUpdateMemoryCID", MsgUpdateMemoryCID], ["/zerone.home.v1.MsgStartSession", MsgStartSession], ["/zerone.home.v1.MsgEndSession", MsgEndSession], ["/zerone.home.v1.MsgRegisterKey", MsgRegisterKey], ["/zerone.home.v1.MsgRevokeKey", MsgRevokeKey], ["/zerone.home.v1.MsgConfigureGuardian", MsgConfigureGuardian], ["/zerone.home.v1.MsgAcknowledgeAlert", MsgAcknowledgeAlert], ["/zerone.home.v1.MsgSetSpendingLimit", MsgSetSpendingLimit], ["/zerone.home.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    createHome(value: MsgCreateHome) {
      return {
        typeUrl: "/zerone.home.v1.MsgCreateHome",
        value: MsgCreateHome.encode(value).finish()
      };
    },
    updateHome(value: MsgUpdateHome) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateHome",
        value: MsgUpdateHome.encode(value).finish()
      };
    },
    updateMemoryCID(value: MsgUpdateMemoryCID) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateMemoryCID",
        value: MsgUpdateMemoryCID.encode(value).finish()
      };
    },
    startSession(value: MsgStartSession) {
      return {
        typeUrl: "/zerone.home.v1.MsgStartSession",
        value: MsgStartSession.encode(value).finish()
      };
    },
    endSession(value: MsgEndSession) {
      return {
        typeUrl: "/zerone.home.v1.MsgEndSession",
        value: MsgEndSession.encode(value).finish()
      };
    },
    registerKey(value: MsgRegisterKey) {
      return {
        typeUrl: "/zerone.home.v1.MsgRegisterKey",
        value: MsgRegisterKey.encode(value).finish()
      };
    },
    revokeKey(value: MsgRevokeKey) {
      return {
        typeUrl: "/zerone.home.v1.MsgRevokeKey",
        value: MsgRevokeKey.encode(value).finish()
      };
    },
    configureGuardian(value: MsgConfigureGuardian) {
      return {
        typeUrl: "/zerone.home.v1.MsgConfigureGuardian",
        value: MsgConfigureGuardian.encode(value).finish()
      };
    },
    acknowledgeAlert(value: MsgAcknowledgeAlert) {
      return {
        typeUrl: "/zerone.home.v1.MsgAcknowledgeAlert",
        value: MsgAcknowledgeAlert.encode(value).finish()
      };
    },
    setSpendingLimit(value: MsgSetSpendingLimit) {
      return {
        typeUrl: "/zerone.home.v1.MsgSetSpendingLimit",
        value: MsgSetSpendingLimit.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createHome(value: MsgCreateHome) {
      return {
        typeUrl: "/zerone.home.v1.MsgCreateHome",
        value
      };
    },
    updateHome(value: MsgUpdateHome) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateHome",
        value
      };
    },
    updateMemoryCID(value: MsgUpdateMemoryCID) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateMemoryCID",
        value
      };
    },
    startSession(value: MsgStartSession) {
      return {
        typeUrl: "/zerone.home.v1.MsgStartSession",
        value
      };
    },
    endSession(value: MsgEndSession) {
      return {
        typeUrl: "/zerone.home.v1.MsgEndSession",
        value
      };
    },
    registerKey(value: MsgRegisterKey) {
      return {
        typeUrl: "/zerone.home.v1.MsgRegisterKey",
        value
      };
    },
    revokeKey(value: MsgRevokeKey) {
      return {
        typeUrl: "/zerone.home.v1.MsgRevokeKey",
        value
      };
    },
    configureGuardian(value: MsgConfigureGuardian) {
      return {
        typeUrl: "/zerone.home.v1.MsgConfigureGuardian",
        value
      };
    },
    acknowledgeAlert(value: MsgAcknowledgeAlert) {
      return {
        typeUrl: "/zerone.home.v1.MsgAcknowledgeAlert",
        value
      };
    },
    setSpendingLimit(value: MsgSetSpendingLimit) {
      return {
        typeUrl: "/zerone.home.v1.MsgSetSpendingLimit",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createHome(value: MsgCreateHome) {
      return {
        typeUrl: "/zerone.home.v1.MsgCreateHome",
        value: MsgCreateHome.fromPartial(value)
      };
    },
    updateHome(value: MsgUpdateHome) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateHome",
        value: MsgUpdateHome.fromPartial(value)
      };
    },
    updateMemoryCID(value: MsgUpdateMemoryCID) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateMemoryCID",
        value: MsgUpdateMemoryCID.fromPartial(value)
      };
    },
    startSession(value: MsgStartSession) {
      return {
        typeUrl: "/zerone.home.v1.MsgStartSession",
        value: MsgStartSession.fromPartial(value)
      };
    },
    endSession(value: MsgEndSession) {
      return {
        typeUrl: "/zerone.home.v1.MsgEndSession",
        value: MsgEndSession.fromPartial(value)
      };
    },
    registerKey(value: MsgRegisterKey) {
      return {
        typeUrl: "/zerone.home.v1.MsgRegisterKey",
        value: MsgRegisterKey.fromPartial(value)
      };
    },
    revokeKey(value: MsgRevokeKey) {
      return {
        typeUrl: "/zerone.home.v1.MsgRevokeKey",
        value: MsgRevokeKey.fromPartial(value)
      };
    },
    configureGuardian(value: MsgConfigureGuardian) {
      return {
        typeUrl: "/zerone.home.v1.MsgConfigureGuardian",
        value: MsgConfigureGuardian.fromPartial(value)
      };
    },
    acknowledgeAlert(value: MsgAcknowledgeAlert) {
      return {
        typeUrl: "/zerone.home.v1.MsgAcknowledgeAlert",
        value: MsgAcknowledgeAlert.fromPartial(value)
      };
    },
    setSpendingLimit(value: MsgSetSpendingLimit) {
      return {
        typeUrl: "/zerone.home.v1.MsgSetSpendingLimit",
        value: MsgSetSpendingLimit.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};