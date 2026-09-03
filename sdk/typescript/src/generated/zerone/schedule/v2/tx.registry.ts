//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateSchedule, MsgUpdateSchedule, MsgCancelSchedule, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.schedule.v2.MsgCreateSchedule", MsgCreateSchedule], ["/zerone.schedule.v2.MsgUpdateSchedule", MsgUpdateSchedule], ["/zerone.schedule.v2.MsgCancelSchedule", MsgCancelSchedule], ["/zerone.schedule.v2.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    createSchedule(value: MsgCreateSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgCreateSchedule",
        value: MsgCreateSchedule.encode(value).finish()
      };
    },
    updateSchedule(value: MsgUpdateSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgUpdateSchedule",
        value: MsgUpdateSchedule.encode(value).finish()
      };
    },
    cancelSchedule(value: MsgCancelSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgCancelSchedule",
        value: MsgCancelSchedule.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createSchedule(value: MsgCreateSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgCreateSchedule",
        value
      };
    },
    updateSchedule(value: MsgUpdateSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgUpdateSchedule",
        value
      };
    },
    cancelSchedule(value: MsgCancelSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgCancelSchedule",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createSchedule(value: MsgCreateSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgCreateSchedule",
        value: MsgCreateSchedule.fromPartial(value)
      };
    },
    updateSchedule(value: MsgUpdateSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgUpdateSchedule",
        value: MsgUpdateSchedule.fromPartial(value)
      };
    },
    cancelSchedule(value: MsgCancelSchedule) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgCancelSchedule",
        value: MsgCancelSchedule.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.schedule.v2.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};