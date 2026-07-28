//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateBountyOrder, MsgFulfillBounty, MsgCancelBountyOrder } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.sponsorship.v1.MsgCreateBountyOrder", MsgCreateBountyOrder], ["/zerone.sponsorship.v1.MsgFulfillBounty", MsgFulfillBounty], ["/zerone.sponsorship.v1.MsgCancelBountyOrder", MsgCancelBountyOrder]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    createBountyOrder(value: MsgCreateBountyOrder) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrder",
        value: MsgCreateBountyOrder.encode(value).finish()
      };
    },
    fulfillBounty(value: MsgFulfillBounty) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgFulfillBounty",
        value: MsgFulfillBounty.encode(value).finish()
      };
    },
    cancelBountyOrder(value: MsgCancelBountyOrder) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrder",
        value: MsgCancelBountyOrder.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createBountyOrder(value: MsgCreateBountyOrder) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrder",
        value
      };
    },
    fulfillBounty(value: MsgFulfillBounty) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgFulfillBounty",
        value
      };
    },
    cancelBountyOrder(value: MsgCancelBountyOrder) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrder",
        value
      };
    }
  },
  fromPartial: {
    createBountyOrder(value: MsgCreateBountyOrder) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrder",
        value: MsgCreateBountyOrder.fromPartial(value)
      };
    },
    fulfillBounty(value: MsgFulfillBounty) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgFulfillBounty",
        value: MsgFulfillBounty.fromPartial(value)
      };
    },
    cancelBountyOrder(value: MsgCancelBountyOrder) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrder",
        value: MsgCancelBountyOrder.fromPartial(value)
      };
    }
  }
};