import { cosmosChainId, zeroneRegistryTypes } from "@zerone-chain/sdk";
import { asZeroneMemoryCid } from "@zerone-chain/sdk/cid";
import {
  auth,
  authMessages,
} from "@zerone-chain/sdk/messages";
import { createZeroneRegistry } from "@zerone-chain/sdk/registry";
import { defaultRegistryTypes } from "@cosmjs/stargate";

const chainId = cosmosChainId("zerone-1");
const memoryCid = asZeroneMemoryCid(
  "bafzbeigai3eoy2ccc7ybwjfz5r3rdxqrinwi4rwytly24tdbh6yk7zslrm",
);
const registerAccount: auth.MsgRegisterAccount = {
  sender: "zrn1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
  did: `did:zrn:${"00".repeat(32)}`,
  publicKey: "",
  accountType: "human",
  operationalKeyHash: "",
  metadata: "",
};
const encoded = authMessages.encoded.registerAccount(registerAccount);
const registry = createZeroneRegistry(defaultRegistryTypes);

void [chainId, memoryCid, encoded, registry, zeroneRegistryTypes];
