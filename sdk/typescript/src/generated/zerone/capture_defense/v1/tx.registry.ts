//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgRecordVerification, MsgAnalyzeDomain, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.capture_defense.v1.MsgRecordVerification", MsgRecordVerification], ["/zerone.capture_defense.v1.MsgAnalyzeDomain", MsgAnalyzeDomain], ["/zerone.capture_defense.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    recordVerification(value: MsgRecordVerification) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgRecordVerification",
        value: MsgRecordVerification.encode(value).finish()
      };
    },
    analyzeDomain(value: MsgAnalyzeDomain) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomain",
        value: MsgAnalyzeDomain.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    recordVerification(value: MsgRecordVerification) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgRecordVerification",
        value
      };
    },
    analyzeDomain(value: MsgAnalyzeDomain) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomain",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    recordVerification(value: MsgRecordVerification) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgRecordVerification",
        value: MsgRecordVerification.fromPartial(value)
      };
    },
    analyzeDomain(value: MsgAnalyzeDomain) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomain",
        value: MsgAnalyzeDomain.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};