// Local-only test driver. Never accepts a private key, key path, RPC endpoint,
// mnemonic or production chain ID. Uses the already-existing SDK wallet flow.
import { createRequire } from "node:module";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { knowledgeMessages } from "../../sdk/typescript/dist/messages.js";
import { createZeroneRegistry } from "../../sdk/typescript/dist/registry.js";
const require = createRequire(new URL("../../sdk/typescript/package.json", import.meta.url));
const { DirectSecp256k1Wallet } = require("@cosmjs/proto-signing");
const { SigningStargateClient, defaultRegistryTypes } = require("@cosmjs/stargate");
const { TxRaw } = require("cosmjs-types/cosmos/tx/v1beta1/tx");
const { MsgSubmitProposal, MsgVote } = require("cosmjs-types/cosmos/gov/v1/tx");

const input = JSON.parse(readFileSync(0, "utf8"));
if (input.chainId !== "tok-feedback-local-only" || ![7, 8, 9, 10, 11, 12, 13].includes(input.actor)) {
  throw new Error("Only fixed local test identities and chain are supported");
}
const wallet = await DirectSecp256k1Wallet.fromKey(new Uint8Array(32).fill(input.actor), "zrn");
const [account] = await wallet.getAccounts();
const registry = createZeroneRegistry(defaultRegistryTypes);
// SDK versions may omit gov/v1 from the base registry; use its real codecs.
registry.register("/cosmos.gov.v1.MsgSubmitProposal", MsgSubmitProposal);
registry.register("/cosmos.gov.v1.MsgVote", MsgVote);
const client = await SigningStargateClient.offline(wallet, { registry });
let message;
switch (input.operation) {
  case "claim":
    message = knowledgeMessages.withTypeUrl.submitClaim({ submitter: account.address,
      factContent: input.content, domain: "local_feedback", category: "empirical", stake: input.stake,
      claimType: input.claimType ?? 1,
      relations: (input.relations ?? []).map(r => ({ ...r, inferenceStrengthBps: BigInt(r.inferenceStrengthBps ?? 0) })),
      references: [] });
    break;
  case "commit":
    message = knowledgeMessages.withTypeUrl.submitCommitment({ verifier: account.address,
      roundId: input.roundId, commitHash: Buffer.from(input.commitHash, "base64") });
    break;
  case "reveal":
    message = knowledgeMessages.withTypeUrl.submitReveal({ verifier: account.address,
      roundId: input.roundId, vote: input.vote, confidence: BigInt(input.confidence), salt: Buffer.from(input.salt, "base64") });
    break;
  case "challenge":
    message = knowledgeMessages.withTypeUrl.challengeFact({ challenger: account.address,
      factId: input.factId, stake: input.stake, reason: input.reason, evidenceIds: input.evidenceIds });
    break;
  case "challenge-provisional":
    message = knowledgeMessages.withTypeUrl.challengeProvisionalFact({ challenger: account.address,
      factId: input.factId, stake: input.stake, reason: input.reason, evidenceIds: input.evidenceIds });
    break;
  case "report":
  case "report-twice":
    message = knowledgeMessages.withTypeUrl.reportFactUse({ consumer: input.consumer ?? account.address, factId: input.factId });
    if (input.operation === "report-twice") message = [message, message];
    break;
  case "rate":
    message = knowledgeMessages.withTypeUrl.rateFact({ rater: input.consumer ?? account.address, factId: input.factId, useful: input.useful ?? true, memo: input.memo ?? "local public fixture" });
    break;
  case "proposal":
    // Decode with the generated codec, then re-encode using the public SDK
    // composer: the nested authority message is never an opaque fallback.
    const codec = registry.lookupType("/zerone.knowledge.v1.MsgUpdateParams");
    const value = codec.decode(Buffer.from(input.paramsMessage, "base64"));
    message = { typeUrl: "/cosmos.gov.v1.MsgSubmitProposal", value: MsgSubmitProposal.fromPartial({
      messages: [registry.encodeAsAny(knowledgeMessages.withTypeUrl.updateParams(value))],
      initialDeposit: [{ denom: "uzrn", amount: "1" }], proposer: account.address,
      title: "Local feedback parameter regression", summary: "Local fixture only", metadata: "local-only",
    }) };
    break;
  case "vote":
    message = { typeUrl: "/cosmos.gov.v1.MsgVote", value: MsgVote.fromPartial({ proposalId: BigInt(input.proposalId), voter: account.address, option: 1, metadata: "local-only" }) };
    break;
  default: throw new Error("Unknown local test operation");
}
const signed = await client.sign(account.address, Array.isArray(message) ? message : [message], {
  amount: [{ denom: "uzrn", amount: String(input.gas) }], gas: String(input.gas),
}, "local-only feedback acceptance", {
  chainId: input.chainId, accountNumber: BigInt(input.accountNumber), sequence: input.sequence,
});
const bytes = TxRaw.encode(signed).finish();
process.stdout.write(JSON.stringify({
  address: account.address, tx: Buffer.from(bytes).toString("base64"),
  sha256: createHash("sha256").update(bytes).digest("hex"),
}));
