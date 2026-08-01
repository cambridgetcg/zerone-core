import { cosmosChainId, zeroneRegistryTypes } from "@zerone-chain/sdk";
import {
  LIQUIDITY_FEE_SCALE,
  discloseLiquiditySwapFee,
  minimumOutputForSlippage,
} from "@zerone-chain/sdk/liquidity";
import {
  auth,
  authMessages,
} from "@zerone-chain/sdk/messages";
import { createZeroneRegistry } from "@zerone-chain/sdk/registry";
import {
  IN_TOTO_STATEMENT_V1_TYPE,
} from "@zerone-chain/sdk/provenance";
import { defaultRegistryTypes } from "@cosmjs/stargate";

const chainId = cosmosChainId("zerone-1");
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
const feeDisclosure = discloseLiquiditySwapFee({
  tokenInDenom: "uzrn",
  feeAmount: "30",
  protocolFeeMillionths: 0n,
});

void [
  chainId,
  encoded,
  registry,
  zeroneRegistryTypes,
  IN_TOTO_STATEMENT_V1_TYPE,
  LIQUIDITY_FEE_SCALE,
  feeDisclosure,
  minimumOutputForSlippage("100", 10_000n),
];
