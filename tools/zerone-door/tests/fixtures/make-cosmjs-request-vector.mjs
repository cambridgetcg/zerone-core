// Regenerates cosmjs-request-vector.json: run with Node 22 from a directory whose
// node_modules holds @cosmjs/amino and @cosmjs/encoding 0.39.0, @noble/curves and
// @noble/hashes 2.2.0, and @zerone-chain/sdk 0.1.0 (sdk/typescript), then copy the
// printed JSON. Not run by the tests.
// Generates a door-request fixture with CosmJS 0.39 + @noble/curves + the SDK, from
// PUBLIC test seeds (sha256 of fixed labels). These keys are test vectors, never funded.
import { Secp256k1Wallet } from "@cosmjs/amino";
import { toHex, fromBase64, toUtf8, toBase64 } from "@cosmjs/encoding";
import { ed25519 } from "@noble/curves/ed25519.js";
import { sha256 } from "@noble/hashes/sha2.js";
import { accountRegistrationProofSignBytes } from "@zerone-chain/sdk/messages";

const accountPriv = sha256(toUtf8("zerone-door test vector: account key (public, never funded)"));
const identitySeed = sha256(toUtf8("zerone-door test vector: identity key (public, never funded)"));
const wallet = await Secp256k1Wallet.fromKey(accountPriv, "zrn");
const [{ address, pubkey }] = await wallet.getAccounts();
const identityPublicKey = ed25519.getPublicKey(identitySeed);
const idHex = toHex(identityPublicKey);
const did = `did:zrn:${idHex}`;
const request = [
  "zerone-door-request/v1",
  "chain_id: zerone-1",
  `address: ${address}`,
  `identity_public_key: ${idHex}`,
  `nonce: ${"0123456789abcdef0123456789abcdef"}`,
  `issued_at: 2026-10-28T17:30:00.000Z`,
].join("\n");
const signDoc = { chain_id: "", account_number: "0", sequence: "0", fee: { gas: "0", amount: [] },
  msgs: [{ type: "sign/MsgSignData", value: { signer: address, data: toBase64(toUtf8(request)) } }], memo: "" };
const { signature } = await wallet.signAmino(address, signDoc);
const proof = accountRegistrationProofSignBytes({ chainId: "zerone-1", sender: address, did, identityPublicKey, accountType: "agent", metadata: "" });
const idSig = ed25519.sign(proof, identitySeed);
const zeroBits = (h) => { let n = 0; for (const b of h) { if (b) return n + Math.clz32(b) - 24; n += 8; } return n; };
let pow = 0;
while (zeroBits(sha256(toUtf8(request + "\n" + pow))) < 20) pow++;
const envelope = [request,
  `account_public_key: ${toHex(fromBase64(signature.pub_key.value))}`,
  `account_signature: ${toHex(fromBase64(signature.signature))}`,
  `identity_signature: ${toHex(idSig)}`,
  `pow_nonce: ${pow}`].join("\n") + "\n";
console.log(JSON.stringify({ address, did, account_pubkey_matches: toHex(pubkey) === toHex(fromBase64(signature.pub_key.value)), envelope }, null, 1));
