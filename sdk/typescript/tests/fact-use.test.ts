import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { DirectSecp256k1Wallet } from "@cosmjs/proto-signing";
import { SigningStargateClient, defaultRegistryTypes } from "@cosmjs/stargate";
import { TxBody, TxRaw } from "cosmjs-types/cosmos/tx/v1beta1/tx";
import { knowledge, knowledgeMessages } from "../src/messages";
import { createZeroneRegistry } from "../src/registry";

// Deliberately public deterministic LOCAL-ONLY test key. Never used on a network.
const localTestKey = new Uint8Array(32).fill(7);

describe("generated signed self-report messages", () => {
  it("encodes exact fields via the public composer and registered codec", () => {
    const registry = createZeroneRegistry(defaultRegistryTypes);
    const value: knowledge.MsgReportFactUse = { consumer: "local-fixture", factId: "f" };
    const message = knowledgeMessages.withTypeUrl.reportFactUse(value);
    assert.equal(message.typeUrl, "/zerone.knowledge.v1.MsgReportFactUse");
    assert.equal(registry.lookupType(message.typeUrl), knowledge.MsgReportFactUse);
    const bytes = registry.encode(message);
    assert.equal(Buffer.from(bytes).toString("hex"), "0a0d6c6f63616c2d66697874757265120166");
    assert.deepEqual(knowledge.MsgReportFactUse.decode(bytes), value);
    assert.throws(() => registry.encode({ typeUrl: "/zerone.knowledge.v1.MsgNotAContract", value }), /Unregistered type url/);
  });

  it("encodes scalar-valued Params maps as length-delimited entry messages", () => {
    const registry = createZeroneRegistry(defaultRegistryTypes);
    const value = knowledge.MsgUpdateParams.fromPartial({ params: { methodologyNormalizationBps: { x: 1n } } });
    const bytes = registry.encode(knowledgeMessages.withTypeUrl.updateParams(value));
    // Outer Params=2, map field=140/wire2, entry key=1/string, value=2/varint.
    assert.equal(Buffer.from(bytes).toString("hex"), "1208e208050a01781001");
    assert.deepEqual(knowledge.MsgUpdateParams.decode(bytes).params?.methodologyNormalizationBps, { x: 1n });
  });

  it("uses the existing wallet's SIGN_MODE_DIRECT and preserves rating wire fields", async () => {
    const wallet = await DirectSecp256k1Wallet.fromKey(localTestKey, "zrn");
    const [account] = await wallet.getAccounts();
    assert.ok(account);
    const registry = createZeroneRegistry(defaultRegistryTypes);
    const client = await SigningStargateClient.offline(wallet, { registry });
    const messages = [
      knowledgeMessages.withTypeUrl.reportFactUse({ consumer: account.address, factId: "f" }),
      knowledgeMessages.withTypeUrl.rateFact({ rater: account.address, factId: "f", useful: false, memo: "public local fixture" }),
    ];
    const signed = await client.sign(account.address, messages, { amount: [{ denom: "uzrn", amount: "400000" }], gas: "400000" }, "local-only", {
      accountNumber: 2n, sequence: 3, chainId: "tok-feedback-local-only",
    });
    assert.equal(signed.signatures[0]?.length, 64);
    const decoded = TxRaw.decode(TxRaw.encode(signed).finish());
    const body = TxBody.decode(decoded.bodyBytes);
    assert.equal(body.messages.length, 2);
    assert.deepEqual(knowledge.MsgReportFactUse.decode(body.messages[0]!.value), messages[0]!.value);
    assert.deepEqual(knowledge.MsgRateFact.decode(body.messages[1]!.value), messages[1]!.value);
    // Signature verification, account policy, fees and commitment are checked
    // by TestToKFeedbackSignedTransport, not inferred from offline signing.
  });
});
