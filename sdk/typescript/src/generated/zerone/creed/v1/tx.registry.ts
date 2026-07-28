//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgAnchorPin, MsgUpdateParams, MsgUpdateCouncilMember } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.creed.v1.MsgAnchorPin", MsgAnchorPin], ["/zerone.creed.v1.MsgUpdateParams", MsgUpdateParams], ["/zerone.creed.v1.MsgUpdateCouncilMember", MsgUpdateCouncilMember]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    anchorPin(value: MsgAnchorPin) {
      return {
        typeUrl: "/zerone.creed.v1.MsgAnchorPin",
        value: MsgAnchorPin.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    },
    updateCouncilMember(value: MsgUpdateCouncilMember) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMember",
        value: MsgUpdateCouncilMember.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    anchorPin(value: MsgAnchorPin) {
      return {
        typeUrl: "/zerone.creed.v1.MsgAnchorPin",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateParams",
        value
      };
    },
    updateCouncilMember(value: MsgUpdateCouncilMember) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMember",
        value
      };
    }
  },
  fromPartial: {
    anchorPin(value: MsgAnchorPin) {
      return {
        typeUrl: "/zerone.creed.v1.MsgAnchorPin",
        value: MsgAnchorPin.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    },
    updateCouncilMember(value: MsgUpdateCouncilMember) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMember",
        value: MsgUpdateCouncilMember.fromPartial(value)
      };
    }
  }
};