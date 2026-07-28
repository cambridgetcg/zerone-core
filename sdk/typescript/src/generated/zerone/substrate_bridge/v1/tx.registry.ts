//@ts-nocheck
import { GeneratedType, Registry } from "@cosmjs/proto-signing";
import { MsgRegisterAdapter, MsgSuspendAdapter, MsgTombstoneAdapter, MsgSubmitExternalAttestation } from "./tx";
export const registry: ReadonlyArray<[string, GeneratedType]> = [["/zerone.substrate_bridge.v1.MsgRegisterAdapter", MsgRegisterAdapter], ["/zerone.substrate_bridge.v1.MsgSuspendAdapter", MsgSuspendAdapter], ["/zerone.substrate_bridge.v1.MsgTombstoneAdapter", MsgTombstoneAdapter], ["/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation", MsgSubmitExternalAttestation]];
export const load = (protoRegistry: Registry) => {
  registry.forEach(([typeUrl, mod]) => {
    protoRegistry.register(typeUrl, mod);
  });
};
export const MessageComposer = {
  encoded: {
    registerAdapter(value: MsgRegisterAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
        value: MsgRegisterAdapter.encode(value).finish()
      };
    },
    suspendAdapter(value: MsgSuspendAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
        value: MsgSuspendAdapter.encode(value).finish()
      };
    },
    tombstoneAdapter(value: MsgTombstoneAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapter",
        value: MsgTombstoneAdapter.encode(value).finish()
      };
    },
    submitExternalAttestation(value: MsgSubmitExternalAttestation) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
        value: MsgSubmitExternalAttestation.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    registerAdapter(value: MsgRegisterAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
        value
      };
    },
    suspendAdapter(value: MsgSuspendAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
        value
      };
    },
    tombstoneAdapter(value: MsgTombstoneAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapter",
        value
      };
    },
    submitExternalAttestation(value: MsgSubmitExternalAttestation) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
        value
      };
    }
  },
  fromPartial: {
    registerAdapter(value: MsgRegisterAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
        value: MsgRegisterAdapter.fromPartial(value)
      };
    },
    suspendAdapter(value: MsgSuspendAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
        value: MsgSuspendAdapter.fromPartial(value)
      };
    },
    tombstoneAdapter(value: MsgTombstoneAdapter) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapter",
        value: MsgTombstoneAdapter.fromPartial(value)
      };
    },
    submitExternalAttestation(value: MsgSubmitExternalAttestation) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
        value: MsgSubmitExternalAttestation.fromPartial(value)
      };
    }
  }
};