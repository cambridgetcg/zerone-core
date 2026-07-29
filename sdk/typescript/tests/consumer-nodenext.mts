import { cosmosChainId, zeroneRegistryTypes } from "@zerone-chain/sdk";
import { asZeroneMemoryCid } from "@zerone-chain/sdk/cid";
import { defineZeroneNetwork } from "@zerone-chain/sdk/caip";
import {
  makeBoundedFeeGrant,
  makeRevokeFeeGrant,
  makeSponsoredFee,
} from "@zerone-chain/sdk/feegrant";
import {
  auth,
  authMessages,
} from "@zerone-chain/sdk/messages";
import {
  IN_TOTO_STATEMENT_V1_TYPE,
} from "@zerone-chain/sdk/provenance";
import { createZeroneRegistry } from "@zerone-chain/sdk/registry";
import {
  defaultRegistryTypes,
  type SigningStargateClient,
} from "@cosmjs/stargate";

const chainId = cosmosChainId("zerone-1");
const network = defineZeroneNetwork("zerone-1");
const granter = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";
const grantee = "zrn1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5s75sh2";
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
const grant = makeBoundedFeeGrant({
  network,
  granter,
  grantee,
  spendLimit: [{ denom: "uzrn", amount: "100000" }],
  expiration: new Date("2099-01-01T00:00:00Z"),
  allowedMessageTypeUrls: ["/zerone.claiming_pot.v1.MsgClaim"],
});
const revoke = makeRevokeFeeGrant({ network, granter, grantee });
const sponsoredFee = makeSponsoredFee({
  network,
  granter,
  amount: [{ denom: "uzrn", amount: "2500" }],
  gas: "200000",
});
const signingFee: Parameters<SigningStargateClient["signAndBroadcast"]>[2] =
  sponsoredFee;

void [
  chainId,
  memoryCid,
  encoded,
  registry,
  zeroneRegistryTypes,
  grant,
  revoke,
  signingFee,
  IN_TOTO_STATEMENT_V1_TYPE,
];
