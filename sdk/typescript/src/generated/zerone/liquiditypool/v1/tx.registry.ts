//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgCreatePool, MsgSwap, MsgAddLiquidity, MsgRemoveLiquidity, MsgUpdateParams } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.liquiditypool.v1.MsgCreatePool", MsgCreatePool], ["/zerone.liquiditypool.v1.MsgSwap", MsgSwap], ["/zerone.liquiditypool.v1.MsgAddLiquidity", MsgAddLiquidity], ["/zerone.liquiditypool.v1.MsgRemoveLiquidity", MsgRemoveLiquidity], ["/zerone.liquiditypool.v1.MsgUpdateParams", MsgUpdateParams]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    createPool(value: MsgCreatePool) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgCreatePool",
        value: MsgCreatePool.encode(value).finish()
      };
    },
    swap(value: MsgSwap) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgSwap",
        value: MsgSwap.encode(value).finish()
      };
    },
    addLiquidity(value: MsgAddLiquidity) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidity",
        value: MsgAddLiquidity.encode(value).finish()
      };
    },
    removeLiquidity(value: MsgRemoveLiquidity) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidity",
        value: MsgRemoveLiquidity.encode(value).finish()
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createPool(value: MsgCreatePool) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgCreatePool",
        value
      };
    },
    swap(value: MsgSwap) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgSwap",
        value
      };
    },
    addLiquidity(value: MsgAddLiquidity) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidity",
        value
      };
    },
    removeLiquidity(value: MsgRemoveLiquidity) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidity",
        value
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createPool(value: MsgCreatePool) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgCreatePool",
        value: MsgCreatePool.fromPartial(value)
      };
    },
    swap(value: MsgSwap) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgSwap",
        value: MsgSwap.fromPartial(value)
      };
    },
    addLiquidity(value: MsgAddLiquidity) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidity",
        value: MsgAddLiquidity.fromPartial(value)
      };
    },
    removeLiquidity(value: MsgRemoveLiquidity) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidity",
        value: MsgRemoveLiquidity.fromPartial(value)
      };
    },
    updateParams(value: MsgUpdateParams) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    }
  }
};