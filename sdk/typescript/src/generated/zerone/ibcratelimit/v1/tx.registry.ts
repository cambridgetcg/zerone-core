//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgAddRateLimit, MsgRemoveRateLimit, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.ibcratelimit.v1.MsgAddRateLimit", MsgAddRateLimit], ["/zerone.ibcratelimit.v1.MsgRemoveRateLimit", MsgRemoveRateLimit], ["/zerone.ibcratelimit.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    addRateLimit(value: MsgAddRateLimit) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimit",
        value: MsgAddRateLimit.encode(value).finish()
      };
    },
    removeRateLimit(value: MsgRemoveRateLimit) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimit",
        value: MsgRemoveRateLimit.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    addRateLimit(value: MsgAddRateLimit) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimit",
        value
      };
    },
    removeRateLimit(value: MsgRemoveRateLimit) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimit",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    addRateLimit(value: MsgAddRateLimit) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimit",
        value: MsgAddRateLimit.fromPartial(value)
      };
    },
    removeRateLimit(value: MsgRemoveRateLimit) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimit",
        value: MsgRemoveRateLimit.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};