// Building, checking, signing and broadcasting R's transactions through the
// pinned zeroned CLI. Unsigned transactions are built with --generate-only
// and a memory keyring (no key is read); signing reads R's file keyring with
// a passphrase passed on stdin, never on argv.
import { createHash } from "node:crypto";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import {
  ADMIT_GAS,
  ALLOWANCE_DAYS,
  ALLOWANCE_PERIOD_LIMIT_UZRN,
  ALLOWANCE_PERIOD_SECONDS,
  ALLOWANCE_SPEND_LIMIT_UZRN,
  ALLOWED_MESSAGES,
  CHAIN_ID,
  MSG_ADD_BOOTSTRAP_ENTRY,
  MSG_GRANT_ALLOWANCE,
  MSG_REVOKE_ALLOWANCE,
  REGRANT_GAS,
  TX_TIMEOUT_BLOCKS,
} from "./policy.ts";

export interface RunResult {
  code: number;
  stdout: string;
  stderr: string;
}

/** Runs the pinned CLI with an exact argv (no shell). stdin, when given, is
 * written to the child and never logged. The caller's buffer is handed to the
 * runtime as it is (no copy here), so the caller can zero it afterwards. */
export type ZeronedRunner = (args: string[], stdin?: Uint8Array) => RunResult;

export function makeZeronedRunner(binary: string): ZeronedRunner {
  return (args, stdin) => {
    const proc = Bun.spawnSync({
      cmd: [binary, ...args],
      stdin: stdin ?? "ignore",
      stdout: "pipe",
      stderr: "pipe",
      env: { HOME: process.env.HOME ?? "/nonexistent", PATH: "/usr/bin:/bin:/usr/sbin:/sbin" },
    });
    return { code: proc.exitCode ?? -1, stdout: proc.stdout.toString(), stderr: proc.stderr.toString() };
  };
}

export function sha256File(path: string): string {
  return createHash("sha256").update(readFileSync(path)).digest("hex");
}

export function rfc3339Seconds(date: Date): string {
  return date.toISOString().replace(/\.\d{3}Z$/, "Z");
}

export function grantExpiration(now: Date): string {
  return rfc3339Seconds(new Date(now.getTime() + ALLOWANCE_DAYS * 86_400_000));
}

export interface UnsignedTx {
  body: { messages: any[]; memo: string; timeout_height: string; extension_options: unknown[]; non_critical_extension_options: unknown[] };
  auth_info: { signer_infos: unknown[]; fee: { amount: Array<{ denom: string; amount: string }>; gas_limit: string; payer: string; granter: string }; tip?: unknown };
  signatures: string[];
}

function generate(run: ZeronedRunner, workHome: string, args: string[], timeoutHeight: number): UnsignedTx {
  const r = run([
    "tx", ...args,
    "--generate-only",
    "--chain-id", CHAIN_ID,
    "--keyring-backend", "memory",
    "--home", workHome,
    "--gas", "200000",
    "--fees", "200000uzrn",
    "--timeout-height", String(timeoutHeight),
    "--output", "json",
  ]);
  if (r.code !== 0) throw new Error(`zeroned refused to build ${args.slice(0, 2).join(" ")}: ${r.stderr.trim().slice(0, 300)}`);
  return JSON.parse(r.stdout) as UnsignedTx;
}

function grantArgs(registrar: string, address: string, expiration: string): string[] {
  return [
    "feegrant", "grant", registrar, address,
    "--spend-limit", `${ALLOWANCE_SPEND_LIMIT_UZRN}uzrn`,
    "--period", String(ALLOWANCE_PERIOD_SECONDS),
    "--period-limit", `${ALLOWANCE_PERIOD_LIMIT_UZRN}uzrn`,
    "--expiration", expiration,
    "--allowed-messages", ALLOWED_MESSAGES.join(","),
  ];
}

function withFee(tx: UnsignedTx, messages: any[], gas: number): UnsignedTx {
  return {
    ...tx,
    body: { ...tx.body, messages, memo: "zerone-door/v1" },
    auth_info: {
      ...tx.auth_info,
      signer_infos: [],
      fee: { amount: [{ denom: "uzrn", amount: String(gas) }], gas_limit: String(gas), payer: "", granter: "" },
    },
    signatures: [],
  };
}

/** The last height that may run a transaction built when the node was at
 * `height` (see TX_TIMEOUT_BLOCKS). */
export function timeoutHeightFor(height: number): number {
  return height + TX_TIMEOUT_BLOCKS;
}

/** One admission: AddBootstrapEntry [address] + GrantAllowance R -> address,
 * with timeout_height `timeoutHeight` (the body comes from the first part). */
export function buildAdmissionTx(run: ZeronedRunner, workHome: string, registrar: string, address: string, expiration: string, timeoutHeight: number): UnsignedTx {
  const add = generate(run, workHome, ["claiming_pot", "add-bootstrap-entry", address, "--from", registrar], timeoutHeight);
  const grant = generate(run, workHome, grantArgs(registrar, address, expiration), timeoutHeight);
  return withFee(add, [...add.body.messages, ...grant.body.messages], ADMIT_GAS);
}

/** A re-grant for an admittee whose seed is unclaimed and whose allowance
 * lapsed: optionally revoke the expired, unpruned grant, then grant afresh. */
export function buildRegrantTx(run: ZeronedRunner, workHome: string, registrar: string, address: string, expiration: string, revokeFirst: boolean, timeoutHeight: number): UnsignedTx {
  const grant = generate(run, workHome, grantArgs(registrar, address, expiration), timeoutHeight);
  const messages = [...grant.body.messages];
  if (revokeFirst) {
    const revoke = generate(run, workHome, ["feegrant", "revoke", registrar, address, "--from", registrar], timeoutHeight);
    messages.unshift(...revoke.body.messages);
  }
  return withFee(grant, messages, REGRANT_GAS);
}

function coinIs(coins: unknown, amount: number): boolean {
  return Array.isArray(coins) && coins.length === 1 && coins[0]?.denom === "uzrn" && coins[0]?.amount === String(amount);
}

/** Checks the CLI's output is exactly what the door means to sign. */
export function checkBuiltTx(tx: UnsignedTx, want: { kind: "admit" | "regrant"; registrar: string; address: string; expiration: string; revokeFirst: boolean; timeoutHeight: number }): string[] {
  const problems: string[] = [];
  // Without its timeout height a lost broadcast could land at any later time,
  // and the door could never prove it gone.
  if (!(want.timeoutHeight > 0) || tx.body.timeout_height !== String(want.timeoutHeight)) problems.push("timeout height");
  const msgs = tx.body.messages;
  const expectTypes = want.kind === "admit"
    ? [MSG_ADD_BOOTSTRAP_ENTRY, MSG_GRANT_ALLOWANCE]
    : want.revokeFirst ? [MSG_REVOKE_ALLOWANCE, MSG_GRANT_ALLOWANCE] : [MSG_GRANT_ALLOWANCE];
  const types = msgs.map((m) => m?.["@type"]);
  if (JSON.stringify(types) !== JSON.stringify(expectTypes)) problems.push(`messages ${JSON.stringify(types)}`);
  for (const m of msgs) {
    if (m?.["@type"] === MSG_ADD_BOOTSTRAP_ENTRY) {
      if (m.authority !== want.registrar) problems.push("bootstrap authority is not R");
      if (JSON.stringify(m.addresses) !== JSON.stringify([want.address])) problems.push("bootstrap addresses");
    }
    if (m?.["@type"] === MSG_REVOKE_ALLOWANCE) {
      if (m.granter !== want.registrar || m.grantee !== want.address) problems.push("revoke parties");
    }
    if (m?.["@type"] === MSG_GRANT_ALLOWANCE) {
      if (m.granter !== want.registrar || m.grantee !== want.address) problems.push("grant parties");
      const outer = m.allowance ?? {};
      const inner = outer.allowance ?? {};
      const basic = inner.basic ?? {};
      if (outer["@type"] !== "/cosmos.feegrant.v1beta1.AllowedMsgAllowance") problems.push("allowance is not AllowedMsgAllowance");
      if (JSON.stringify(outer.allowed_messages) !== JSON.stringify(ALLOWED_MESSAGES)) problems.push("allowed messages");
      if (inner["@type"] !== "/cosmos.feegrant.v1beta1.PeriodicAllowance") problems.push("inner allowance is not PeriodicAllowance");
      if (!coinIs(basic.spend_limit, ALLOWANCE_SPEND_LIMIT_UZRN)) problems.push("spend limit");
      if (basic.expiration !== want.expiration) problems.push("expiration");
      if (inner.period !== `${ALLOWANCE_PERIOD_SECONDS}s`) problems.push("period");
      if (!coinIs(inner.period_spend_limit, ALLOWANCE_PERIOD_LIMIT_UZRN)) problems.push("period limit");
      if (!coinIs(inner.period_can_spend, ALLOWANCE_PERIOD_LIMIT_UZRN)) problems.push("period can-spend");
    }
  }
  const gas = want.kind === "admit" ? ADMIT_GAS : REGRANT_GAS;
  const fee = tx.auth_info.fee;
  if (fee.gas_limit !== String(gas) || !coinIs(fee.amount, gas) || fee.payer !== "" || fee.granter !== "") problems.push("fee");
  return problems;
}

export interface SignerInfoInput {
  pubKeyB64: string;
  sequence: string;
}

/** The unsigned tx with R's public key and live sequence and an empty
 * signature: the shape Simulate needs. No private key is involved. */
export function withPublicSigner(tx: UnsignedTx, signer: SignerInfoInput): UnsignedTx {
  return {
    ...tx,
    auth_info: {
      ...tx.auth_info,
      signer_infos: [{
        public_key: { "@type": "/cosmos.crypto.secp256k1.PubKey", key: signer.pubKeyB64 },
        mode_info: { single: { mode: "SIGN_MODE_DIRECT" } },
        sequence: signer.sequence,
      }],
    },
    signatures: [""],
  };
}

export function encodeTx(run: ZeronedRunner, workHome: string, tx: UnsignedTx, name: string): Uint8Array {
  mkdirSync(workHome, { recursive: true, mode: 0o700 });
  const file = join(workHome, `${name}.json`);
  writeFileSync(file, JSON.stringify(tx), { mode: 0o600 });
  const r = run(["tx", "encode", file, "--home", workHome]);
  if (r.code !== 0) throw new Error(`zeroned could not encode the tx: ${r.stderr.trim().slice(0, 300)}`);
  return Uint8Array.from(Buffer.from(r.stdout.trim(), "base64"));
}

export function txHashOf(bytes: Uint8Array): string {
  return createHash("sha256").update(bytes).digest("hex").toUpperCase();
}

export interface SignInput {
  keyName: string;
  keyringDir: string;
  accountNumber: string;
  sequence: string;
  passphrase: Uint8Array;
}

/** Signs offline with R's file keyring. The passphrase goes to stdin only. */
export function signTx(run: ZeronedRunner, workHome: string, tx: UnsignedTx, input: SignInput): UnsignedTx {
  mkdirSync(workHome, { recursive: true, mode: 0o700 });
  const unsigned = join(workHome, "admit.unsigned.json");
  const signed = join(workHome, "admit.signed.json");
  writeFileSync(unsigned, JSON.stringify(tx), { mode: 0o600 });
  const stdin = new Uint8Array(input.passphrase.length + 1);
  stdin.set(input.passphrase, 0);
  stdin[input.passphrase.length] = 0x0a;
  try {
    const r = run([
      "tx", "sign", unsigned,
      "--from", input.keyName,
      "--keyring-backend", "file",
      "--keyring-dir", input.keyringDir,
      "--chain-id", CHAIN_ID,
      "--offline",
      "--account-number", input.accountNumber,
      "--sequence", input.sequence,
      "--sign-mode", "direct",
      "--output-document", signed,
      "--home", workHome,
    ], stdin);
    if (r.code !== 0) throw new Error(`signing failed: ${r.stderr.replace(/passphrase[^\n]*/gi, "passphrase prompt").trim().slice(0, 300)}`);
  } finally {
    stdin.fill(0);
  }
  return JSON.parse(readFileSync(signed, "utf8")) as UnsignedTx;
}

/** The signed tx must carry exactly the body and fee that were simulated,
 * R's on-chain public key and the expected sequence. */
export function checkSignedTx(signed: UnsignedTx, simulated: UnsignedTx, signer: SignerInfoInput): string[] {
  const problems: string[] = [];
  if (JSON.stringify(signed.body) !== JSON.stringify(simulated.body)) problems.push("body changed in signing");
  if (JSON.stringify(signed.auth_info.fee) !== JSON.stringify(simulated.auth_info.fee)) problems.push("fee changed in signing");
  const info = signed.auth_info.signer_infos as any[];
  if (!Array.isArray(info) || info.length !== 1) problems.push("signer count");
  else {
    if (info[0]?.public_key?.key !== signer.pubKeyB64) problems.push("signed by a key other than R's on-chain key");
    if (String(info[0]?.sequence) !== signer.sequence) problems.push("sequence");
  }
  if (!Array.isArray(signed.signatures) || signed.signatures.length !== 1 || !signed.signatures[0]) problems.push("signature missing");
  return problems;
}

export interface BroadcastResult {
  txHash: string;
  code: number;
  log: string;
}

export function broadcastTx(run: ZeronedRunner, workHome: string, signed: UnsignedTx, node: string): BroadcastResult {
  const file = join(workHome, "admit.signed.broadcast.json");
  writeFileSync(file, JSON.stringify(signed), { mode: 0o600 });
  const r = run(["tx", "broadcast", file, "--node", node, "--broadcast-mode", "sync", "--output", "json", "--home", workHome]);
  const start = r.stdout.indexOf("{");
  if (start === -1) throw new Error(`broadcast gave no answer: ${r.stderr.trim().slice(0, 300)}`);
  const parsed = JSON.parse(r.stdout.slice(start));
  if (typeof parsed.txhash !== "string") throw new Error("broadcast answer has no txhash");
  return { txHash: parsed.txhash.toUpperCase(), code: Number(parsed.code ?? 0), log: String(parsed.raw_log ?? "") };
}

export function tcpNode(rpc: string): string {
  return rpc.startsWith("http://") ? `tcp://${rpc.slice("http://".length)}` : rpc;
}
