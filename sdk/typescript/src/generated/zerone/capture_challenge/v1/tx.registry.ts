//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgSubmitChallenge, MsgAddEvidence, MsgResolveChallenge, MsgFundBountyPool, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.capture_challenge.v1.MsgSubmitChallenge", MsgSubmitChallenge], ["/zerone.capture_challenge.v1.MsgAddEvidence", MsgAddEvidence], ["/zerone.capture_challenge.v1.MsgResolveChallenge", MsgResolveChallenge], ["/zerone.capture_challenge.v1.MsgFundBountyPool", MsgFundBountyPool], ["/zerone.capture_challenge.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    submitChallenge(value: MsgSubmitChallenge) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallenge",
        value: MsgSubmitChallenge.encode(value).finish()
      };
    },
    addEvidence(value: MsgAddEvidence) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidence",
        value: MsgAddEvidence.encode(value).finish()
      };
    },
    resolveChallenge(value: MsgResolveChallenge) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallenge",
        value: MsgResolveChallenge.encode(value).finish()
      };
    },
    fundBountyPool(value: MsgFundBountyPool) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPool",
        value: MsgFundBountyPool.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    submitChallenge(value: MsgSubmitChallenge) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallenge",
        value
      };
    },
    addEvidence(value: MsgAddEvidence) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidence",
        value
      };
    },
    resolveChallenge(value: MsgResolveChallenge) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallenge",
        value
      };
    },
    fundBountyPool(value: MsgFundBountyPool) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPool",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    submitChallenge(value: MsgSubmitChallenge) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallenge",
        value: MsgSubmitChallenge.fromPartial(value)
      };
    },
    addEvidence(value: MsgAddEvidence) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidence",
        value: MsgAddEvidence.fromPartial(value)
      };
    },
    resolveChallenge(value: MsgResolveChallenge) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallenge",
        value: MsgResolveChallenge.fromPartial(value)
      };
    },
    fundBountyPool(value: MsgFundBountyPool) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPool",
        value: MsgFundBountyPool.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};