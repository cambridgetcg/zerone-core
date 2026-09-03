import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import {
  chmodSync,
  existsSync,
  lstatSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  loadOrCreatePersistedIdentityKey,
  loadPersistedIdentityKey,
  type PersistedIdentityKey,
} from "../src/account-registration.ts";
import {
  classifyRegisteredAccountQuery,
  parseRegisteredAccountQueryOutput,
  reconcileRegisteredAccount,
  type RegisteredZeroneAccount,
} from "../src/account-reconciliation.ts";

const identityPublic =
  "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a";
const identityPrivate =
  "302e020100300506032b6570042204209d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60";
const identity: PersistedIdentityKey = {
  key: "math-v1",
  public_hex: identityPublic,
  private_pkcs8_hex: identityPrivate,
};
const addresses = [
  "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z",
  "zrn1ur4eyeuuhrkfpcyhykfjsasftv9hn33smszt58",
  "zrn1q0ar3f2cswzlemcss4nu82cd40crftd9utnt0e",
  "zrn100mxrvv5chhhrj0yd9y4q8354z4edm42mukf5r",
  "zrn1qp7g4mas5wpv6gdgwjrvxl8xysa8jty993seas",
  "zrn1d59mrcrs4uanm58xrenckjyu6wrzf969h6vzdk",
] as const;

function operationalHash(publicHex = identityPublic): string {
  return createHash("sha256")
    .update(Buffer.from(publicHex, "hex"))
    .digest("hex");
}

function account(
  address = addresses[0],
  overrides: Partial<RegisteredZeroneAccount> = {},
): RegisteredZeroneAccount {
  return {
    address,
    did: `did:zrn:${identityPublic}`,
    public_key: identityPublic,
    account_type: "agent",
    operational_key_hash: operationalHash(),
    operational_public_key: identityPublic,
    operational_key_version: 1,
    ...overrides,
  };
}

function accountEnvelope(
  address = addresses[0],
  overrides: Partial<RegisteredZeroneAccount> = {},
): string {
  return JSON.stringify({
    account: {
      ...account(address, overrides),
      reputation_score: 500000,
      created_at_block: "1",
      last_active_block: "1",
      flags: { can_submit_claims: true, can_challenge: true },
      metadata: "",
    },
  });
}

describe("registered account parsing and reconciliation", () => {
  test("strict parsing accepts the complete canonical auth response", () => {
    expect(parseRegisteredAccountQueryOutput(accountEnvelope())).toEqual(account());
    expect(classifyRegisteredAccountQuery({
      exit_code: 0,
      stdout: `${accountEnvelope()}\n`,
      stderr: "",
    })).toEqual({ kind: "found", account: account() });
  });

  test("only the exact module NotFound diagnostic authorizes registration", () => {
    expect(classifyRegisteredAccountQuery({
      exit_code: 1,
      stdout: "",
      stderr: "Error: failed to query account: rpc error: code = Unknown desc = account not found: unknown request\nUsage:\n",
    })).toEqual({ kind: "not_found" });
    expect(classifyRegisteredAccountQuery({
      exit_code: 1,
      stdout: "",
      stderr: "Error: failed to query account: rpc error: code = NotFound desc = account not found\n",
    })).toEqual({ kind: "not_found" });

    for (const failure of [
      { exit_code: 1, stdout: "", stderr: "" },
      { exit_code: 1, stdout: "", stderr: "rpc error: connection refused\n" },
      { exit_code: 1, stdout: "{}", stderr: "Error: account not found\n" },
      { exit_code: 0, stdout: "not-json", stderr: "" },
      { exit_code: 0, stdout: accountEnvelope(), stderr: "warning\n" },
    ]) {
      expect(() => classifyRegisteredAccountQuery(failure)).toThrow();
    }
  });

  test("parser rejects partial identity and operational-key state", () => {
    expect(() => parseRegisteredAccountQueryOutput("{}"))
      .toThrow("response.account must be an object");
    expect(() => parseRegisteredAccountQueryOutput(accountEnvelope(addresses[0], {
      did: `did:zrn:${"00".repeat(32)}`,
    }))).toThrow("full DID derived");
    expect(() => parseRegisteredAccountQueryOutput(accountEnvelope(addresses[0], {
      operational_key_hash: "",
    }))).toThrow("non-empty string");
    expect(() => parseRegisteredAccountQueryOutput(accountEnvelope(addresses[0], {
      operational_key_hash: "00".repeat(32),
    }))).toThrow("not the SHA-256");
    expect(() => parseRegisteredAccountQueryOutput(accountEnvelope(addresses[0], {
      operational_key_version: 0,
    }))).toThrow("integer from 1");
    const stringVersion = JSON.parse(accountEnvelope()) as {
      account: Record<string, unknown>;
    };
    stringVersion.account.operational_key_version = "1";
    expect(() => parseRegisteredAccountQueryOutput(JSON.stringify(stringVersion)))
      .toThrow("integer from 1");
  });

  test("reconciliation proves address, DID, role, current key, hash, and keypair custody", () => {
    expect(() => reconcileRegisteredAccount(account(), {
      key: identity.key,
      address: addresses[0],
      accountType: "agent",
      identity,
    })).not.toThrow();

    expect(() => reconcileRegisteredAccount(account(addresses[1]), {
      key: identity.key,
      address: addresses[0],
      accountType: "agent",
      identity,
    })).toThrow("address is");
    expect(() => reconcileRegisteredAccount(account(addresses[0], {
      account_type: "human",
    }), {
      key: identity.key,
      address: addresses[0],
      accountType: "agent",
      identity,
    })).toThrow("account role is human");
    expect(() => reconcileRegisteredAccount(account(addresses[0], {
      operational_public_key: "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c",
      operational_key_hash: operationalHash(
        "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c",
      ),
      operational_key_version: 2,
    }), {
      key: identity.key,
      address: addresses[0],
      accountType: "agent",
      identity,
    })).toThrow("current operational key at version 2 is unavailable locally");
    expect(() => reconcileRegisteredAccount(account(), {
      key: identity.key,
      address: addresses[0],
      accountType: "agent",
      identity: { ...identity, private_pkcs8_hex: `${identityPrivate.slice(0, -2)}00` },
    })).toThrow();
  });
});

describe("identity custody directory durability", () => {
  test("creates each missing private component and securely reloads without creation", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-directory-chain-"));
    try {
      const first = join(scratch, "state", "identities");
      const identityPath = join(first, "math-v1.ed25519.json");
      const created = loadOrCreatePersistedIdentityKey(identityPath, "math-v1");
      expect(loadPersistedIdentityKey(identityPath, "math-v1")).toEqual(created);
      expect(lstatSync(join(scratch, "state")).mode & 0o777).toBe(0o700);
      expect(lstatSync(first).mode & 0o777).toBe(0o700);
      expect(lstatSync(identityPath).mode & 0o777).toBe(0o600);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  });

  test("load-only fails without creating missing custody", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-load-only-"));
    try {
      const missingParent = join(scratch, "must-not-exist");
      expect(() => loadPersistedIdentityKey(
        join(missingParent, "math-v1.ed25519.json"),
        "math-v1",
      )).toThrow("refusing to create custody for an existing account");
      expect(existsSync(missingParent)).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  });

  test("unsafe or symlinked existing ancestors fail before key generation", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-unsafe-directory-"));
    try {
      const permissive = join(scratch, "permissive");
      mkdirSync(permissive, { mode: 0o700 });
      chmodSync(permissive, 0o755);
      const beneathPermissive = join(permissive, "child", "identity.json");
      expect(() => loadOrCreatePersistedIdentityKey(beneathPermissive, "math-v1"))
        .toThrow("group/world permissions must be zero");
      expect(existsSync(join(permissive, "child"))).toBe(false);

      const realDirectory = join(scratch, "real");
      mkdirSync(realDirectory, { mode: 0o700 });
      const linkedDirectory = join(scratch, "linked");
      symlinkSync(realDirectory, linkedDirectory);
      expect(() => loadOrCreatePersistedIdentityKey(
        join(linkedDirectory, "identity.json"),
        "math-v1",
      )).toThrow("secure no-follow open failed");
      expect(existsSync(join(realDirectory, "identity.json"))).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  });
});

describe("setup account preflight", () => {
  const operatorKeys = ["math-sub", "math-v1", "math-v2", "math-v3", "math-v4"];

  function writeIdentities(directory: string): void {
    for (const key of operatorKeys) {
      writeFileSync(
        join(directory, `zerone-testnet-1.${key}.ed25519.json`),
        JSON.stringify({ ...identity, key }),
        { mode: 0o600 },
      );
    }
  }

  function writeFakeZeroned(path: string): void {
    writeFileSync(path, `#!/usr/bin/env bun
import { appendFileSync } from "node:fs";
const args = Bun.argv.slice(2);
const addresses = ${JSON.stringify({
  faucet: addresses[5],
  "math-sub": addresses[0],
  "math-v1": addresses[1],
  "math-v2": addresses[2],
  "math-v3": addresses[3],
  "math-v4": addresses[4],
})};
const publicHex = ${JSON.stringify(identityPublic)};
const operationalHash = ${JSON.stringify(operationalHash())};
const rotatedPublicHex = "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c";
const rotatedHash = ${JSON.stringify(operationalHash(
  "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c",
))};
if (args[0] === "keys" && args[1] === "show") {
  console.log(addresses[args[2]]);
} else if (args[0] === "query" && args[1] === "zerone_auth" && args[2] === "account") {
  if (process.env.FRONTIER_FAKE_QUERY === "ambiguous") {
    console.error("Error: failed to query account: rpc error: code = Unavailable desc = connection refused");
    process.exit(1);
  }
  if (process.env.FRONTIER_FAKE_QUERY === "first-notfound-second-ambiguous") {
    if (args[3] === addresses["math-sub"]) {
      console.error("Error: failed to query account: rpc error: code = Unknown desc = account not found: unknown request");
      process.exit(1);
    }
    if (args[3] === addresses["math-v1"]) {
      console.error("Error: failed to query account: rpc error: code = Unavailable desc = connection refused");
      process.exit(1);
    }
  }
  const wrongRole = process.env.FRONTIER_FAKE_QUERY === "wrong-role" ||
    (process.env.FRONTIER_FAKE_QUERY === "last-wrong-role" && args[3] === addresses["math-v4"]);
  const rotated = process.env.FRONTIER_FAKE_QUERY === "rotated";
  console.log(JSON.stringify({ account: {
    address: args[3],
    did: "did:zrn:" + publicHex,
    public_key: publicHex,
    account_type: wrongRole ? "human" : "agent",
    operational_key_hash: rotated ? rotatedHash : operationalHash,
    operational_public_key: rotated ? rotatedPublicHex : publicHex,
    operational_key_version: rotated ? 2 : 1,
    reputation_score: 500000,
    created_at_block: "1",
    last_active_block: "1",
    flags: { can_submit_claims: true, can_challenge: true },
    metadata: "",
  }}));
} else if (args[0] === "query" && args[1] === "bank" && args[2] === "balances") {
  console.log(JSON.stringify({ balances: [{ denom: "uzrn", amount: process.env.FRONTIER_FAKE_BALANCE ?? "200000000" }] }));
} else if (args[0] === "tx") {
  appendFileSync(process.env.FRONTIER_FAKE_LOG, JSON.stringify(args) + "\\n");
  console.error("unexpected transaction");
  process.exit(2);
} else {
  console.error("unexpected fake zeroned invocation: " + JSON.stringify(args));
  process.exit(2);
}
`, { mode: 0o700 });
  }

  function runSetup(
    scratch: string,
    queryMode: "match" | "wrong-role" | "last-wrong-role" | "rotated" | "ambiguous" | "first-notfound-second-ambiguous",
    balance = "200000000",
  ): ReturnType<typeof Bun.spawnSync> {
    const fakeZeroned = join(scratch, "zeroned");
    writeFakeZeroned(fakeZeroned);
    return Bun.spawnSync({
      cmd: [
        process.execPath,
        new URL("../intake.ts", import.meta.url).pathname,
        "setup",
        "--network",
        "zerone-testnet-1",
        "--live-ack=zerone-testnet-1",
      ],
      cwd: new URL("..", import.meta.url).pathname,
      env: {
        ...process.env,
        ZERONED_BIN: fakeZeroned,
        FRONTIER_FAKE_LOG: join(scratch, "broadcasts.jsonl"),
        FRONTIER_FAKE_QUERY: queryMode,
        FRONTIER_FAKE_BALANCE: balance,
        FRONTIER_INTAKE_STATE_DIR: join(scratch, "state"),
      },
      stdout: "pipe",
      stderr: "pipe",
    });
  }

  test("an existing registration is green only after complete local reconciliation", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-reconciled-"));
    try {
      const stateDirectory = join(scratch, "state");
      mkdirSync(stateDirectory, { mode: 0o700 });
      writeIdentities(stateDirectory);
      const result = runSetup(scratch, "match");
      expect(result.exitCode, result.stderr.toString()).toBe(0);
      expect(
        result.stdout.toString().match(/identity and current operational key reconciled/g),
      ).toHaveLength(5);
      expect(existsSync(join(scratch, "broadcasts.jsonl"))).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  }, 30_000);

  test("a registered mismatch fails before even a required funding broadcast", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-mismatch-"));
    try {
      const stateDirectory = join(scratch, "state");
      mkdirSync(stateDirectory, { mode: 0o700 });
      writeIdentities(stateDirectory);
      const result = runSetup(scratch, "wrong-role", "0");
      expect(result.exitCode).not.toBe(0);
      expect(result.stderr.toString()).toContain("account role is human, expected agent");
      expect(existsSync(join(scratch, "broadcasts.jsonl"))).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  }, 30_000);

  test("all identities preflight before the first funding transaction", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-all-preflight-"));
    try {
      const stateDirectory = join(scratch, "state");
      mkdirSync(stateDirectory, { mode: 0o700 });
      writeIdentities(stateDirectory);
      const result = runSetup(scratch, "last-wrong-role", "0");
      expect(result.exitCode).not.toBe(0);
      expect(
        result.stdout.toString().match(/identity and current operational key reconciled/g),
      ).toHaveLength(4);
      expect(result.stdout.toString()).not.toContain("funder faucet");
      expect(existsSync(join(scratch, "broadcasts.jsonl"))).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  }, 30_000);

  test("a rotated-away current operational key is explicitly unavailable", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-rotated-"));
    try {
      const stateDirectory = join(scratch, "state");
      mkdirSync(stateDirectory, { mode: 0o700 });
      writeIdentities(stateDirectory);
      const result = runSetup(scratch, "rotated", "0");
      expect(result.exitCode).not.toBe(0);
      expect(result.stderr.toString()).toContain(
        "current operational key at version 2 is unavailable locally",
      );
      expect(existsSync(join(scratch, "broadcasts.jsonl"))).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  }, 30_000);

  test("a found account with missing local custody creates nothing", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-missing-custody-"));
    try {
      const result = runSetup(scratch, "match", "0");
      expect(result.exitCode).not.toBe(0);
      expect(result.stderr.toString()).toContain(
        "refusing to create custody for an existing account",
      );
      expect(existsSync(join(scratch, "state"))).toBe(false);
      expect(existsSync(join(scratch, "broadcasts.jsonl"))).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  }, 30_000);

  test("an ambiguous account query creates no identity and broadcasts nothing", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-ambiguous-"));
    try {
      const result = runSetup(scratch, "ambiguous", "0");
      expect(result.exitCode).not.toBe(0);
      expect(result.stderr.toString()).toContain("failed ambiguously");
      expect(existsSync(join(scratch, "state"))).toBe(false);
      expect(existsSync(join(scratch, "broadcasts.jsonl"))).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  }, 30_000);

  test("a later ambiguous query cannot leave a key from an earlier NotFound", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-late-ambiguous-"));
    try {
      const result = runSetup(scratch, "first-notfound-second-ambiguous", "0");
      expect(result.exitCode).not.toBe(0);
      expect(result.stderr.toString()).toContain("math-v1");
      expect(result.stderr.toString()).toContain("failed ambiguously");
      expect(existsSync(join(scratch, "state"))).toBe(false);
      expect(existsSync(join(scratch, "broadcasts.jsonl"))).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  }, 30_000);
});
