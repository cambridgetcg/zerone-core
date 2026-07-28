//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreatePot, MsgClaim, MsgUpdatePotParams, MsgAddBootstrapEntry } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.claiming_pot.v1.MsgCreatePot", MsgCreatePot], ["/zerone.claiming_pot.v1.MsgClaim", MsgClaim], ["/zerone.claiming_pot.v1.MsgUpdatePotParams", MsgUpdatePotParams], ["/zerone.claiming_pot.v1.MsgAddBootstrapEntry", MsgAddBootstrapEntry]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    createPot(value: MsgCreatePot) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgCreatePot",
        value: MsgCreatePot.encode(value).finish()
      };
    },
    claim(value: MsgClaim) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgClaim",
        value: MsgClaim.encode(value).finish()
      };
    },
    updatePotParams(value: MsgUpdatePotParams) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParams",
        value: MsgUpdatePotParams.encode(value).finish()
      };
    },
    addBootstrapEntry(value: MsgAddBootstrapEntry) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
        value: MsgAddBootstrapEntry.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createPot(value: MsgCreatePot) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgCreatePot",
        value
      };
    },
    claim(value: MsgClaim) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgClaim",
        value
      };
    },
    updatePotParams(value: MsgUpdatePotParams) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParams",
        value
      };
    },
    addBootstrapEntry(value: MsgAddBootstrapEntry) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
        value
      };
    }
  },
  fromPartial: {
    createPot(value: MsgCreatePot) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgCreatePot",
        value: MsgCreatePot.fromPartial(value)
      };
    },
    claim(value: MsgClaim) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgClaim",
        value: MsgClaim.fromPartial(value)
      };
    },
    updatePotParams(value: MsgUpdatePotParams) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParams",
        value: MsgUpdatePotParams.fromPartial(value)
      };
    },
    addBootstrapEntry(value: MsgAddBootstrapEntry) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
        value: MsgAddBootstrapEntry.fromPartial(value)
      };
    }
  }
};