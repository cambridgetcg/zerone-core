import {
  accountRegistrationProofSignBytes,
  cosmosChainId,
  keyRotationAcceptanceSignBytes,
  keyRotationAuthorizationSignBytes,
  zeroneRegistryTypes,
} from "@zerone-chain/sdk";
import { asZeroneMemoryCid } from "@zerone-chain/sdk/cid";
import { defineZeroneNetwork } from "@zerone-chain/sdk/caip";
import {
  makeBoundedFeeGrant,
  makeRevokeFeeGrant,
  makeSponsoredFee,
} from "@zerone-chain/sdk/feegrant";
import {
  LIQUIDITY_FEE_SCALE,
  discloseLiquiditySwapFee,
  minimumOutputForSlippage,
} from "@zerone-chain/sdk/liquidity";
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
const identityPublicKeyHex =
  "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a";
const identityPublicKey = Uint8Array.from(
  identityPublicKeyHex.match(/../g) ?? [],
  (byte) => Number.parseInt(byte, 16),
);
const registerAccount: auth.MsgRegisterAccount = {
  sender: "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z",
  did: `did:zrn:${identityPublicKeyHex}`,
  publicKey: identityPublicKeyHex,
  accountType: "human",
  operationalKeyHash: "",
  metadata: "",
  identityProofSignature: new Uint8Array(64),
};
const registrationProof = accountRegistrationProofSignBytes({
  chainId: "zerone-1",
  sender: registerAccount.sender,
  did: registerAccount.did,
  identityPublicKey,
  accountType: "human",
  metadata: registerAccount.metadata,
});
const replacementPublicKeyHex =
  "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c";
const replacementPublicKey = Uint8Array.from(
  replacementPublicKeyHex.match(/../g) ?? [],
  (byte) => Number.parseInt(byte, 16),
);
const rotationContext = {
  chainId: "zerone-1",
  sender: registerAccount.sender,
  currentKeyVersion: 1,
  newOperationalKey: replacementPublicKey,
  authorizationExpiresAtUnix: 1n,
};
const rotationAuthorization = keyRotationAuthorizationSignBytes(rotationContext);
const rotationAcceptance = keyRotationAcceptanceSignBytes(rotationContext);
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
const feeDisclosure = discloseLiquiditySwapFee({
  tokenInDenom: "uzrn",
  feeAmount: "30",
  protocolFeeMillionths: 0n,
});

void [
  chainId,
  memoryCid,
  registrationProof,
  rotationAuthorization,
  rotationAcceptance,
  encoded,
  registry,
  zeroneRegistryTypes,
  grant,
  revoke,
  signingFee,
  IN_TOTO_STATEMENT_V1_TYPE,
  LIQUIDITY_FEE_SCALE,
  feeDisclosure,
  minimumOutputForSlippage("100", 10_000n),
];
