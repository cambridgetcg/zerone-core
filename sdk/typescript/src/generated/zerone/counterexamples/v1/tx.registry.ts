//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgProposeCounterexample, MsgValidate, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.counterexamples.v1.MsgProposeCounterexample", MsgProposeCounterexample], ["/zerone.counterexamples.v1.MsgValidate", MsgValidate], ["/zerone.counterexamples.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    proposeCounterexample(value: MsgProposeCounterexample) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexample",
        value: MsgProposeCounterexample.encode(value).finish()
      };
    },
    validate(value: MsgValidate) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgValidate",
        value: MsgValidate.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    proposeCounterexample(value: MsgProposeCounterexample) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexample",
        value
      };
    },
    validate(value: MsgValidate) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgValidate",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    proposeCounterexample(value: MsgProposeCounterexample) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexample",
        value: MsgProposeCounterexample.fromPartial(value)
      };
    },
    validate(value: MsgValidate) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgValidate",
        value: MsgValidate.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};