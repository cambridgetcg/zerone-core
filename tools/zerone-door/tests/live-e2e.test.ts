// Keyless end-to-end dry run against live zerone-1. Skipped unless
// ZERONE_DOOR_LIVE_E2E=1 and ZERONE_DOOR_ZERONED names the pinned CLI.
//
// It makes a door request from public test seeds (a throwaway address that
// has never existed on chain), runs every gate against live state with the
// CURRENT on-chain registrar as a public-key-only stand-in for R (its address
// and its on-chain public key; no private key exists here), builds the real
// admission with --generate-only and a memory keyring, and Simulates it. It
// signs nothing and broadcasts nothing. Its only writes are a temp directory.
// The chain version pin is the committed one in config.example.json, so this
// also checks that zerone-1 still runs the version the door was measured on.
import { describe, expect, test } from "bun:test";
import { createHash, randomBytes } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { ChainReader } from "../src/chain.ts";
import { doorPaths, expandHome } from "../src/config.ts";
import { processItem, type DoorContext } from "../src/door.ts";
import { ADMIT_GAS, REGRANT_GAS, SIMULATE_MARGIN_GAS } from "../src/policy.ts";
import { adr036SignBytes, envelopeText, registrationProofBytes, requestText, solvePow } from "../src/request.ts";
import { publicKeyFromPrivate, signSecp256k1ForTest } from "../src/secp256k1.ts";
import { buildRegrantTx, encodeTx, grantExpiration, makeZeronedRunner, sha256File, withPublicSigner } from "../src/tx.ts";
import { addressFor, identityKeys, makeDoorHome, seed } from "./helpers.ts";

const enabled = process.env.ZERONE_DOOR_LIVE_E2E === "1" && !!process.env.ZERONE_DOOR_ZERONED;
const RPC = "http://169.155.55.44:26657";
const REST = "http://169.155.55.44:1317";

function liveRequest(now: Date): { text: string; address: string } {
  const label = "live-e2e-throwaway";
  const priv = seed(`${label}/account`);
  const address = addressFor(label);
  const id = identityKeys(label);
  const request = requestText({
    chainId: "zerone-1",
    address,
    identityPublicKeyHex: Buffer.from(id.publicKey).toString("hex"),
    nonceHex: randomBytes(16).toString("hex"),
    issuedAt: now.toISOString(),
  });
  const digest = Uint8Array.from(createHash("sha256").update(adr036SignBytes(address, new TextEncoder().encode(request))).digest());
  const k = BigInt(`0x${randomBytes(32).toString("hex")}`);
  return {
    address,
    text: envelopeText(request, {
      accountPublicKeyHex: Buffer.from(publicKeyFromPrivate(priv)).toString("hex"),
      accountSignatureHex: Buffer.from(signSecp256k1ForTest(digest, priv, k)).toString("hex"),
      identitySignatureHex: Buffer.from(id.sign(registrationProofBytes("zerone-1", address, id.publicKey))).toString("hex"),
      powNonce: solvePow(request),
    }),
  };
}

describe.skipIf(!enabled)("live zerone-1, keyless dry run", () => {
  test("every gate passes for a throwaway agent and the real admission simulates", async () => {
    const zeronedPath = process.env.ZERONE_DOOR_ZERONED!;
    const chain = new ChainReader({ rpc: RPC, rest: REST });
    const params = await chain.potParams();
    if (!params.ok) throw new Error(params.error);
    const standIn = params.value.registrar;
    const { configPath, config } = makeDoorHome();
    const committedPin = JSON.parse(readFileSync(join(import.meta.dir, "..", "config.example.json"), "utf8")).chain_version;
    const cfg = {
      ...config,
      rpc: RPC,
      rest: REST,
      zeroned: { path: zeronedPath, sha256: sha256File(zeronedPath) },
      chain_version: committedPin,
      registrar: { ...config.registrar, address: standIn },
      known_keys: [],
    };
    writeFileSync(configPath, JSON.stringify(cfg), { mode: 0o600 });
    const logs: Array<Record<string, unknown>> = [];
    const ctx: DoorContext = {
      config: cfg,
      paths: { ...doorPaths(configPath), haltFiles: [join(cfg.home, "HALT"), expandHome("~/.local/state/zerone/rung-window.freeze")] },
      chain,
      zeroned: makeZeronedRunner(zeronedPath),
      passphrase: () => { throw new Error("a dry run never asks for a passphrase"); },
      live: false,
      now: () => new Date(),
      log: (e) => logs.push(e),
    };
    const version = await chain.chainVersion();
    console.log(JSON.stringify({ chain_version_live: version, chain_version_pinned: committedPin }));
    const req = liveRequest(new Date());
    const { reply } = await processItem(ctx, { text: req.text, channel: { kind: "studio_local", file: "live-e2e.request" } });
    console.log(JSON.stringify({ stand_in_registrar: standIn, throwaway: req.address, reply }, null, 2));
    expect(reply?.decision).toBe("dry_run_ok");
    expect(reply!.simulated_gas! + SIMULATE_MARGIN_GAS).toBeLessThanOrEqual(ADMIT_GAS);

    // Re-grant shape (grant only): measured for REGRANT_GAS. The revoke-first
    // variant cannot be simulated until a grant exists to revoke.
    const auth = await chain.authAccount(standIn);
    if (!auth.ok || !auth.value?.pubKeyB64) throw new Error("stand-in has no public key");
    const work = join(cfg.home, "state", "work", "regrant");
    const regrant = buildRegrantTx(ctx.zeroned, work, standIn, req.address, grantExpiration(new Date()), false);
    const sim = await chain.simulate(encodeTx(ctx.zeroned, work, withPublicSigner(regrant, { pubKeyB64: auth.value.pubKeyB64, sequence: auth.value.sequence }), "regrant"));
    console.log(JSON.stringify({ regrant_simulated: sim }));
    expect(sim.ok).toBe(true);
    if (sim.ok) expect(sim.value.gasUsed + SIMULATE_MARGIN_GAS).toBeLessThanOrEqual(REGRANT_GAS);
  }, 180_000);
});
