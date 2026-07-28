//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreateToken, MsgMintToken, MsgBurnToken, MsgTransferToken, MsgApproveToken, MsgTransferFrom, MsgPauseToken, MsgUnpauseToken, MsgDelegatePower, MsgUndelegatePower, MsgWrapToken, MsgUnwrapToken, MsgCreateEmissionPeriod, MsgCancelEmissionPeriod, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.tokens.v1.MsgCreateToken", MsgCreateToken], ["/zerone.tokens.v1.MsgMintToken", MsgMintToken], ["/zerone.tokens.v1.MsgBurnToken", MsgBurnToken], ["/zerone.tokens.v1.MsgTransferToken", MsgTransferToken], ["/zerone.tokens.v1.MsgApproveToken", MsgApproveToken], ["/zerone.tokens.v1.MsgTransferFrom", MsgTransferFrom], ["/zerone.tokens.v1.MsgPauseToken", MsgPauseToken], ["/zerone.tokens.v1.MsgUnpauseToken", MsgUnpauseToken], ["/zerone.tokens.v1.MsgDelegatePower", MsgDelegatePower], ["/zerone.tokens.v1.MsgUndelegatePower", MsgUndelegatePower], ["/zerone.tokens.v1.MsgWrapToken", MsgWrapToken], ["/zerone.tokens.v1.MsgUnwrapToken", MsgUnwrapToken], ["/zerone.tokens.v1.MsgCreateEmissionPeriod", MsgCreateEmissionPeriod], ["/zerone.tokens.v1.MsgCancelEmissionPeriod", MsgCancelEmissionPeriod], ["/zerone.tokens.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    createToken(value: MsgCreateToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateToken",
        value: MsgCreateToken.encode(value).finish()
      };
    },
    mintToken(value: MsgMintToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgMintToken",
        value: MsgMintToken.encode(value).finish()
      };
    },
    burnToken(value: MsgBurnToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgBurnToken",
        value: MsgBurnToken.encode(value).finish()
      };
    },
    transferToken(value: MsgTransferToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferToken",
        value: MsgTransferToken.encode(value).finish()
      };
    },
    approveToken(value: MsgApproveToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgApproveToken",
        value: MsgApproveToken.encode(value).finish()
      };
    },
    transferFrom(value: MsgTransferFrom) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferFrom",
        value: MsgTransferFrom.encode(value).finish()
      };
    },
    pauseToken(value: MsgPauseToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgPauseToken",
        value: MsgPauseToken.encode(value).finish()
      };
    },
    unpauseToken(value: MsgUnpauseToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnpauseToken",
        value: MsgUnpauseToken.encode(value).finish()
      };
    },
    delegatePower(value: MsgDelegatePower) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgDelegatePower",
        value: MsgDelegatePower.encode(value).finish()
      };
    },
    undelegatePower(value: MsgUndelegatePower) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUndelegatePower",
        value: MsgUndelegatePower.encode(value).finish()
      };
    },
    wrapToken(value: MsgWrapToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgWrapToken",
        value: MsgWrapToken.encode(value).finish()
      };
    },
    unwrapToken(value: MsgUnwrapToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnwrapToken",
        value: MsgUnwrapToken.encode(value).finish()
      };
    },
    createEmissionPeriod(value: MsgCreateEmissionPeriod) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriod",
        value: MsgCreateEmissionPeriod.encode(value).finish()
      };
    },
    cancelEmissionPeriod(value: MsgCancelEmissionPeriod) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriod",
        value: MsgCancelEmissionPeriod.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createToken(value: MsgCreateToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateToken",
        value
      };
    },
    mintToken(value: MsgMintToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgMintToken",
        value
      };
    },
    burnToken(value: MsgBurnToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgBurnToken",
        value
      };
    },
    transferToken(value: MsgTransferToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferToken",
        value
      };
    },
    approveToken(value: MsgApproveToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgApproveToken",
        value
      };
    },
    transferFrom(value: MsgTransferFrom) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferFrom",
        value
      };
    },
    pauseToken(value: MsgPauseToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgPauseToken",
        value
      };
    },
    unpauseToken(value: MsgUnpauseToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnpauseToken",
        value
      };
    },
    delegatePower(value: MsgDelegatePower) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgDelegatePower",
        value
      };
    },
    undelegatePower(value: MsgUndelegatePower) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUndelegatePower",
        value
      };
    },
    wrapToken(value: MsgWrapToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgWrapToken",
        value
      };
    },
    unwrapToken(value: MsgUnwrapToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnwrapToken",
        value
      };
    },
    createEmissionPeriod(value: MsgCreateEmissionPeriod) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriod",
        value
      };
    },
    cancelEmissionPeriod(value: MsgCancelEmissionPeriod) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriod",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createToken(value: MsgCreateToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateToken",
        value: MsgCreateToken.fromPartial(value)
      };
    },
    mintToken(value: MsgMintToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgMintToken",
        value: MsgMintToken.fromPartial(value)
      };
    },
    burnToken(value: MsgBurnToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgBurnToken",
        value: MsgBurnToken.fromPartial(value)
      };
    },
    transferToken(value: MsgTransferToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferToken",
        value: MsgTransferToken.fromPartial(value)
      };
    },
    approveToken(value: MsgApproveToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgApproveToken",
        value: MsgApproveToken.fromPartial(value)
      };
    },
    transferFrom(value: MsgTransferFrom) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferFrom",
        value: MsgTransferFrom.fromPartial(value)
      };
    },
    pauseToken(value: MsgPauseToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgPauseToken",
        value: MsgPauseToken.fromPartial(value)
      };
    },
    unpauseToken(value: MsgUnpauseToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnpauseToken",
        value: MsgUnpauseToken.fromPartial(value)
      };
    },
    delegatePower(value: MsgDelegatePower) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgDelegatePower",
        value: MsgDelegatePower.fromPartial(value)
      };
    },
    undelegatePower(value: MsgUndelegatePower) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUndelegatePower",
        value: MsgUndelegatePower.fromPartial(value)
      };
    },
    wrapToken(value: MsgWrapToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgWrapToken",
        value: MsgWrapToken.fromPartial(value)
      };
    },
    unwrapToken(value: MsgUnwrapToken) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnwrapToken",
        value: MsgUnwrapToken.fromPartial(value)
      };
    },
    createEmissionPeriod(value: MsgCreateEmissionPeriod) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriod",
        value: MsgCreateEmissionPeriod.fromPartial(value)
      };
    },
    cancelEmissionPeriod(value: MsgCancelEmissionPeriod) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriod",
        value: MsgCancelEmissionPeriod.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};