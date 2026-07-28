//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgUpdateParams, MsgActivate } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.alignment.v1.MsgUpdateParams", MsgUpdateParams], ["/zerone.alignment.v1.MsgActivate", MsgActivate]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    },
    activate(value: MsgActivate) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgActivate",
        value: MsgActivate.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgUpdateParams",
        value
      };
    },
    activate(value: MsgActivate) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgActivate",
        value
      };
    }
  },
  fromPartial: {
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    },
    activate(value: MsgActivate) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgActivate",
        value: MsgActivate.fromPartial(value)
      };
    }
  }
};