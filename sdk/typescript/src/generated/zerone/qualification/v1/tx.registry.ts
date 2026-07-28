//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgQualifyByStake, MsgQualifyByTrackRecord, MsgQualifyByCrossReference, MsgQualifyByInheritance, MsgEndorseQualification, MsgRenewQualification, MsgWithdrawQualification, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.qualification.v1.MsgQualifyByStake", MsgQualifyByStake], ["/zerone.qualification.v1.MsgQualifyByTrackRecord", MsgQualifyByTrackRecord], ["/zerone.qualification.v1.MsgQualifyByCrossReference", MsgQualifyByCrossReference], ["/zerone.qualification.v1.MsgQualifyByInheritance", MsgQualifyByInheritance], ["/zerone.qualification.v1.MsgEndorseQualification", MsgEndorseQualification], ["/zerone.qualification.v1.MsgRenewQualification", MsgRenewQualification], ["/zerone.qualification.v1.MsgWithdrawQualification", MsgWithdrawQualification], ["/zerone.qualification.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    qualifyByStake(value: MsgQualifyByStake) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByStake",
        value: MsgQualifyByStake.encode(value).finish()
      };
    },
    qualifyByTrackRecord(value: MsgQualifyByTrackRecord) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecord",
        value: MsgQualifyByTrackRecord.encode(value).finish()
      };
    },
    qualifyByCrossReference(value: MsgQualifyByCrossReference) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReference",
        value: MsgQualifyByCrossReference.encode(value).finish()
      };
    },
    qualifyByInheritance(value: MsgQualifyByInheritance) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritance",
        value: MsgQualifyByInheritance.encode(value).finish()
      };
    },
    endorseQualification(value: MsgEndorseQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgEndorseQualification",
        value: MsgEndorseQualification.encode(value).finish()
      };
    },
    renewQualification(value: MsgRenewQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgRenewQualification",
        value: MsgRenewQualification.encode(value).finish()
      };
    },
    withdrawQualification(value: MsgWithdrawQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgWithdrawQualification",
        value: MsgWithdrawQualification.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    qualifyByStake(value: MsgQualifyByStake) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByStake",
        value
      };
    },
    qualifyByTrackRecord(value: MsgQualifyByTrackRecord) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecord",
        value
      };
    },
    qualifyByCrossReference(value: MsgQualifyByCrossReference) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReference",
        value
      };
    },
    qualifyByInheritance(value: MsgQualifyByInheritance) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritance",
        value
      };
    },
    endorseQualification(value: MsgEndorseQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgEndorseQualification",
        value
      };
    },
    renewQualification(value: MsgRenewQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgRenewQualification",
        value
      };
    },
    withdrawQualification(value: MsgWithdrawQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgWithdrawQualification",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    qualifyByStake(value: MsgQualifyByStake) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByStake",
        value: MsgQualifyByStake.fromPartial(value)
      };
    },
    qualifyByTrackRecord(value: MsgQualifyByTrackRecord) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecord",
        value: MsgQualifyByTrackRecord.fromPartial(value)
      };
    },
    qualifyByCrossReference(value: MsgQualifyByCrossReference) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReference",
        value: MsgQualifyByCrossReference.fromPartial(value)
      };
    },
    qualifyByInheritance(value: MsgQualifyByInheritance) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritance",
        value: MsgQualifyByInheritance.fromPartial(value)
      };
    },
    endorseQualification(value: MsgEndorseQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgEndorseQualification",
        value: MsgEndorseQualification.fromPartial(value)
      };
    },
    renewQualification(value: MsgRenewQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgRenewQualification",
        value: MsgRenewQualification.fromPartial(value)
      };
    },
    withdrawQualification(value: MsgWithdrawQualification) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgWithdrawQualification",
        value: MsgWithdrawQualification.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};