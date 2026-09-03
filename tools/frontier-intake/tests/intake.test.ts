import { describe, expect, test } from "bun:test";
import { createPrivateKey, verify } from "node:crypto";
import {
  chmodSync,
  closeSync,
  constants,
  existsSync,
  lstatSync,
  mkdirSync,
  mkdtempSync,
  openSync,
  readFileSync,
  readdirSync,
  renameSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import {
  ACCOUNT_REGISTRATION_PROOF_DOMAIN,
  DARWIN_ACL_CAPTURE_PARENT,
  DARWIN_ACL_CAPTURE_PREFIX,
  accountRegistrationCommandArgs,
  accountRegistrationProofSignBytes,
  decodeDarwinAclInspectionResult,
  loadOrCreatePersistedIdentityKey,
  parsePersistedIdentityKey,
  type PersistedIdentityKey,
} from "../src/account-registration.ts";
import { defaultRegisterPath } from "../src/config.ts";
import { commitHash, decideActions, newSalt, type RoundState } from "../src/panel.ts";
import { admissibleEntries, firstBatch, loadRegister, validateRegister } from "../src/register.ts";
import {
  FRONTIER_INTAKE_BUN_VERSION,
  requireFrontierIntakeBunVersion,
} from "../src/runtime.ts";

const registrationSender = "zrn1m037n75vk2jhdr56y2ptzjjj02uljwnqwwzr7z";
const registrationPublicHex = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a";
const registrationPrivatePkcs8Hex = "302e020100300506032b6570042204209d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60";
const registrationIdentity: PersistedIdentityKey = {
  key: "math-v1",
  public_hex: registrationPublicHex,
  private_pkcs8_hex: registrationPrivatePkcs8Hex,
};

const inheritedEveryoneAcl =
  "everyone allow list,search,read,file_inherit,directory_inherit";
const darwinAclHelperPath = fileURLToPath(
  new URL("../build/darwin-acl-check", import.meta.url),
);

function darwinAclCaptureArtifacts(): string[] {
  const processPrefix = `${DARWIN_ACL_CAPTURE_PREFIX}${process.pid}-`;
  return readdirSync(DARWIN_ACL_CAPTURE_PARENT)
    .filter(name => name.startsWith(processPrefix))
    .sort();
}

function inheritedEveryoneAclToolingAvailable(): boolean {
  if (process.platform !== "darwin") return false;
  const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-acl-probe-"));
  try {
    const result = Bun.spawnSync({
      cmd: ["/bin/chmod", "+a", inheritedEveryoneAcl, scratch],
      stdout: "pipe",
      stderr: "pipe",
    });
    return result.exitCode === 0;
  } finally {
    rmSync(scratch, { recursive: true, force: true });
  }
}

const hasInheritedEveryoneAclTooling = inheritedEveryoneAclToolingAvailable();

describe("zerone_auth registration", () => {
  test("production CLI requires the exact audited Bun runtime", () => {
    expect(FRONTIER_INTAKE_BUN_VERSION).toBe("1.3.5");
    expect(() => requireFrontierIntakeBunVersion("1.3.5")).not.toThrow();
    expect(() => requireFrontierIntakeBunVersion("1.3.6")).toThrow(
      "frontier-intake requires Bun 1.3.5, got 1.3.6",
    );
  });

  test("Darwin ACL helper protocol accepts only exact authenticated tuples", () => {
    expect(decodeDarwinAclInspectionResult(
      0,
      undefined,
      "zerone-darwin-acl-v1 clear\n",
      "",
    )).toBe(false);
    expect(decodeDarwinAclInspectionResult(
      10,
      undefined,
      "zerone-darwin-acl-v1 present\n",
      "",
    )).toBe(true);

    for (const tuple of [
      [0, undefined, "", ""],
      [0, undefined, "zerone-darwin-acl-v1 clear\nextra", ""],
      [0, undefined, "zerone-darwin-acl-v1 clear\n", "warning"],
      [0, "SIGKILL", "zerone-darwin-acl-v1 clear\n", ""],
      [70, undefined, "", "inspection failed"],
    ] as const) {
      expect(() => decodeDarwinAclInspectionResult(...tuple)).toThrow(
        "descriptor ACL inspection returned an invalid result",
      );
    }
  });

  test("proof bytes and PKCS#8 signature match the Go v1 known vector", () => {
    const proofBytes = accountRegistrationProofSignBytes({
      chainId: "zerone-test-1",
      sender: registrationSender,
      publicHex: registrationPublicHex,
      accountType: "agent",
      metadata: '{"name":"Sophia"}',
    });
    expect(ACCOUNT_REGISTRATION_PROOF_DOMAIN).toBe("zerone.auth/register-account/v1");
    expect(proofBytes.toString("hex")).toBe(
      "7a65726f6e652e617574682f72656769737465722d6163636f756e742f7631000000000d7a65726f6e652d746573742d310000002a7a726e316d3033376e3735766b326a6864723536793270747a6a6a6a3032756c6a776e7177777a72377a000000486469643a7a726e3a64373561393830313832623130616237643534626665643363393634303733613065653137326633646161363233323561663032316136386637303735313161d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a000000056167656e74000000117b226e616d65223a22536f70686961227d",
    );

    const args = accountRegistrationCommandArgs({
      chainId: "zerone-test-1",
      sender: registrationSender,
      identity: registrationIdentity,
      accountType: "agent",
      metadata: '{"name":"Sophia"}',
    });
    expect(args.slice(0, 5)).toEqual([
      "zerone_auth",
      "register-account",
      `did:zrn:${registrationPublicHex}`,
      registrationPublicHex,
      "agent",
    ]);
    expect(args[5]).toBe(
      "cea58f50222ca9d26ccca52ed5338c61fe1c04be4d313693b63cadce7e6dd51c7f38179de28ea7309894e190c2d50b2bc3f5de7757e57fc1113856b508efbe06",
    );
    const privateKey = createPrivateKey({
      key: Buffer.from(registrationPrivatePkcs8Hex, "hex"),
      format: "der",
      type: "pkcs8",
    });
    expect(verify(null, proofBytes, privateKey, Buffer.from(args[5] as string, "hex"))).toBe(true);
  });

  test("persisted identity parsing and key matching fail closed", () => {
    expect(parsePersistedIdentityKey(JSON.stringify(registrationIdentity), "math-v1"))
      .toEqual(registrationIdentity);
    expect(() => parsePersistedIdentityKey(JSON.stringify(registrationIdentity), "math-v2"))
      .toThrow("identity file key does not match math-v2");
    expect(() => accountRegistrationCommandArgs({
      chainId: "zerone-test-1",
      sender: registrationSender,
      identity: { ...registrationIdentity, public_hex: "00".repeat(32) },
      accountType: "agent",
      metadata: "",
    })).toThrow("public key does not match its private key");
  });

  test("secure reuse preserves an existing private identity without replacement", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-existing-"));
    try {
      const identityPath = join(scratch, "math-v1.ed25519.json");
      const encoded = JSON.stringify(registrationIdentity);
      writeFileSync(identityPath, encoded, { mode: 0o600 });
      const before = lstatSync(identityPath);

      expect(loadOrCreatePersistedIdentityKey(identityPath, "math-v1"))
        .toEqual(registrationIdentity);
      expect(loadOrCreatePersistedIdentityKey(identityPath, "math-v1"))
        .toEqual(registrationIdentity);

      const after = lstatSync(identityPath);
      expect(after.dev).toBe(before.dev);
      expect(after.ino).toBe(before.ino);
      expect(readFileSync(identityPath, "utf8")).toBe(encoded);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  });

  test("secure reuse rejects symlinks, permissive modes, non-files, and oversized files", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-invalid-key-"));
    try {
      const target = join(scratch, "target.ed25519.json");
      writeFileSync(target, JSON.stringify(registrationIdentity), { mode: 0o600 });
      const link = join(scratch, "linked.ed25519.json");
      symlinkSync(target, link);
      expect(() => loadOrCreatePersistedIdentityKey(link, "math-v1"))
        .toThrow("secure no-follow open failed");

      const permissive = join(scratch, "permissive.ed25519.json");
      writeFileSync(permissive, JSON.stringify(registrationIdentity), { mode: 0o600 });
      chmodSync(permissive, 0o640);
      expect(() => loadOrCreatePersistedIdentityKey(permissive, "math-v1"))
        .toThrow("group/world permissions must be zero");

      const directory = join(scratch, "directory.ed25519.json");
      mkdirSync(directory, { mode: 0o700 });
      expect(() => loadOrCreatePersistedIdentityKey(directory, "math-v1"))
        .toThrow("path is not a regular file");

      const oversized = join(scratch, "oversized.ed25519.json");
      writeFileSync(oversized, Buffer.alloc(4097), { mode: 0o600 });
      expect(() => loadOrCreatePersistedIdentityKey(oversized, "math-v1"))
        .toThrow("file exceeds 4096 bytes");
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  });

  test.skipIf(!hasInheritedEveryoneAclTooling)(
    "a mode-0600 identity file with a macOS read ACL is rejected",
    () => {
      const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-file-acl-"));
      try {
        const clearParentAcl = Bun.spawnSync({
          cmd: ["/bin/chmod", "-N", scratch],
          stdout: "pipe",
          stderr: "pipe",
        });
        expect(clearParentAcl.exitCode, clearParentAcl.stderr.toString()).toBe(0);

        const identityPath = join(scratch, "math-v1.ed25519.json");
        const encoded = JSON.stringify(registrationIdentity);
        writeFileSync(identityPath, encoded, { mode: 0o600 });
        const addFileAcl = Bun.spawnSync({
          cmd: ["/bin/chmod", "+a", "everyone allow read", identityPath],
          stdout: "pipe",
          stderr: "pipe",
        });
        expect(addFileAcl.exitCode, addFileAcl.stderr.toString()).toBe(0);
        expect(lstatSync(identityPath).mode & 0o777).toBe(0o600);

        const capturesBefore = darwinAclCaptureArtifacts();
        expect(() => loadOrCreatePersistedIdentityKey(identityPath, "math-v1"))
          .toThrow("extended ACLs are not allowed");
        expect(darwinAclCaptureArtifacts()).toEqual(capturesBefore);
        expect(readFileSync(identityPath, "utf8")).toBe(encoded);
      } finally {
        rmSync(scratch, { recursive: true, force: true });
      }
    },
  );

  test.skipIf(!hasInheritedEveryoneAclTooling)(
    "the macOS ACL helper follows the inherited descriptor, not a replaced path",
    () => {
      const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-acl-fd-"));
      let fd: number | undefined;
      try {
        const identityPath = join(scratch, "identity.json");
        const movedIdentityPath = join(scratch, "identity-with-acl.json");
        writeFileSync(identityPath, JSON.stringify(registrationIdentity), { mode: 0o600 });
        const addFileAcl = Bun.spawnSync({
          cmd: ["/bin/chmod", "+a", "everyone allow read", identityPath],
          stdout: "pipe",
          stderr: "pipe",
        });
        expect(addFileAcl.exitCode, addFileAcl.stderr.toString()).toBe(0);

        fd = openSync(identityPath, constants.O_RDONLY | constants.O_NOFOLLOW);
        renameSync(identityPath, movedIdentityPath);
        writeFileSync(identityPath, JSON.stringify(registrationIdentity), { mode: 0o600 });

        const result = Bun.spawnSync({
          cmd: [darwinAclHelperPath],
          stdin: fd,
          stdout: "pipe",
          stderr: "pipe",
          env: {},
          timeout: 10_000,
        });
        expect(result.exitCode).toBe(10);
        expect(result.signalCode).toBeUndefined();
        expect(result.stdout.toString()).toBe("zerone-darwin-acl-v1 present\n");
        expect(result.stderr.toString()).toBe("");
      } finally {
        if (fd !== undefined) closeSync(fd);
        rmSync(scratch, { recursive: true, force: true });
      }
    },
    15_000,
  );

  test.skipIf(process.platform !== "darwin")(
    "a bundled intake module fails closed when its ACL helper is missing or unsafe",
    async () => {
      const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-acl-helper-"));
      try {
        const bundledSourceDirectory = join(scratch, "src");
        const bundle = await Bun.build({
          entrypoints: [fileURLToPath(new URL("../src/account-registration.ts", import.meta.url))],
          outdir: bundledSourceDirectory,
          target: "bun",
          naming: "account-registration.js",
        });
        expect(bundle.success, bundle.logs.map(log => log.message).join("\n")).toBe(true);
        const bundledModule = join(bundledSourceDirectory, "account-registration.js");
        const identityPath = join(scratch, "state", "identity.json");
        const runner = join(scratch, "run.ts");
        writeFileSync(runner, `
import { loadOrCreatePersistedIdentityKey } from ${JSON.stringify(pathToFileURL(bundledModule).href)};
try {
  loadOrCreatePersistedIdentityKey(${JSON.stringify(identityPath)}, "math-v1");
  console.log("unexpected success");
  process.exit(0);
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(2);
}
`, { mode: 0o700 });

        const missing = Bun.spawnSync({
          cmd: [process.execPath, runner],
          stdout: "pipe",
          stderr: "pipe",
        });
        expect(missing.exitCode).toBe(2);
        expect(missing.stderr.toString()).toContain(
          "descriptor ACL inspection is unavailable or unsafe",
        );
        expect(missing.stderr.toString()).toContain(
          "run make frontier-intake-darwin-acl-helper",
        );
        expect(existsSync(identityPath)).toBe(false);

        const isolatedBuildDirectory = join(scratch, "build");
        mkdirSync(isolatedBuildDirectory, { mode: 0o700 });
        const unsafeHelper = join(isolatedBuildDirectory, "darwin-acl-check");
        writeFileSync(unsafeHelper, "not a Mach-O executable", { mode: 0o555 });
        const unsafe = Bun.spawnSync({
          cmd: [process.execPath, runner],
          stdout: "pipe",
          stderr: "pipe",
        });
        expect(unsafe.exitCode).toBe(2);
        expect(unsafe.stderr.toString()).toContain(
          "descriptor ACL inspection is unavailable or unsafe",
        );
        expect(unsafe.stderr.toString()).toContain(
          "helper is not a universal Mach-O executable",
        );
        expect(existsSync(identityPath)).toBe(false);
      } finally {
        rmSync(scratch, { recursive: true, force: true });
      }
    },
    15_000,
  );

  test.skipIf(!hasInheritedEveryoneAclTooling)(
    "an inherited macOS ACL blocks setup before key exposure or broadcast",
    () => {
      const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-inherited-acl-"));
      try {
        const stateDirectory = join(scratch, "state");
        const fakeZeroned = join(scratch, "zeroned");
        const broadcastLog = join(scratch, "broadcasts.jsonl");
        writeFileSync(fakeZeroned, `#!/usr/bin/env bun
import { appendFileSync } from "node:fs";
const args = Bun.argv.slice(2);
if (args[0] === "keys" && args[1] === "show") {
  console.log(${JSON.stringify(registrationSender)});
} else if (args[0] === "query" && args[1] === "bank" && args[2] === "balances") {
  console.log(JSON.stringify({ balances: [{ denom: "uzrn", amount: "200000000" }] }));
} else if (args[0] === "query" && args[1] === "zerone_auth" && args[2] === "account") {
  console.error("Error: failed to query account: rpc error: code = Unknown desc = account not found: unknown request");
  process.exit(1);
} else if (args[0] === "tx") {
  appendFileSync(process.env.FRONTIER_FAKE_LOG, JSON.stringify(args) + "\\n");
  console.log(JSON.stringify({ txhash: "UNEXPECTED", code: 0 }));
} else {
  console.error("unexpected fake zeroned invocation: " + JSON.stringify(args));
  process.exit(2);
}
`);
        chmodSync(fakeZeroned, 0o700);

        const aclResult = Bun.spawnSync({
          cmd: ["/bin/chmod", "+a", inheritedEveryoneAcl, scratch],
          stdout: "pipe",
          stderr: "pipe",
        });
        expect(aclResult.exitCode, aclResult.stderr.toString()).toBe(0);

        const intakePath = new URL("../intake.ts", import.meta.url).pathname;
        const run = Bun.spawnSync({
          cmd: [
            process.execPath,
            intakePath,
            "setup",
            "--network",
            "zerone-testnet-1",
            "--live-ack=zerone-testnet-1",
          ],
          cwd: new URL("..", import.meta.url).pathname,
          env: {
            ...process.env,
            ZERONED_BIN: fakeZeroned,
            FRONTIER_FAKE_LOG: broadcastLog,
            FRONTIER_INTAKE_STATE_DIR: stateDirectory,
          },
          stdout: "pipe",
          stderr: "pipe",
        });

        expect(run.exitCode).not.toBe(0);
        expect(run.stderr.toString()).toContain("extended ACLs are not allowed");
        for (const key of ["math-sub", "math-v1", "math-v2", "math-v3", "math-v4"]) {
          expect(existsSync(join(
            stateDirectory,
            `zerone-testnet-1.${key}.ed25519.json`,
          ))).toBe(false);
        }
        expect(existsSync(broadcastLog)).toBe(false);
      } finally {
        rmSync(scratch, { recursive: true, force: true });
      }
    },
    15_000,
  );

  test("exclusive concurrent creators converge on one durable identity", async () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-concurrent-key-"));
    try {
      const runner = join(scratch, "create-identity.ts");
      const identityPath = join(scratch, "race.ed25519.json");
      const helperUrl = new URL("../src/account-registration.ts", import.meta.url).href;
      writeFileSync(runner, `import { loadOrCreatePersistedIdentityKey } from ${JSON.stringify(helperUrl)};
process.umask(0o777);
const identity = loadOrCreatePersistedIdentityKey(Bun.argv[2], "race-key");
console.log(JSON.stringify(identity));
`, { mode: 0o700 });

      const children = Array.from({ length: 4 }, () => Bun.spawn({
        cmd: [process.execPath, runner, identityPath],
        stdout: "pipe",
        stderr: "pipe",
      }));
      const results = await Promise.all(children.map(async child => {
        const stdout = new Response(child.stdout).text();
        const stderr = new Response(child.stderr).text();
        const exitCode = await child.exited;
        return { exitCode, stdout: await stdout, stderr: await stderr };
      }));
      for (const result of results) {
        expect(result.exitCode, result.stderr).toBe(0);
      }

      const identities = results.map(result =>
        parsePersistedIdentityKey(result.stdout.trim(), "race-key"));
      expect(new Set(identities.map(identity => JSON.stringify(identity))).size).toBe(1);
      expect(loadOrCreatePersistedIdentityKey(identityPath, "race-key"))
        .toEqual(identities[0] as PersistedIdentityKey);
      expect(lstatSync(identityPath).mode & 0o777).toBe(0o600);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  });

  test("setup passes a valid identity proof as register-account's fourth argument", () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-intake-proof-"));
    try {
      const fakeZeroned = join(scratch, "zeroned");
      const invocationLog = join(scratch, "registration-invocations.jsonl");
      for (const key of ["math-sub", "math-v1", "math-v2", "math-v3", "math-v4"]) {
        writeFileSync(
          join(scratch, `zerone-testnet-1.${key}.ed25519.json`),
          JSON.stringify({ ...registrationIdentity, key }),
          { mode: 0o600 },
        );
      }
      writeFileSync(fakeZeroned, `#!/usr/bin/env bun
import { createHash } from "node:crypto";
import { appendFileSync, existsSync, readFileSync, unlinkSync, writeFileSync } from "node:fs";
const args = Bun.argv.slice(2);
if (args[0] === "keys" && args[1] === "show") {
  console.log(${JSON.stringify(registrationSender)});
} else if (args[0] === "query" && args[1] === "bank" && args[2] === "balances") {
  console.log(JSON.stringify({ balances: [{ denom: "uzrn", amount: "200000000" }] }));
} else if (args[0] === "query" && args[1] === "zerone_auth" && args[2] === "account") {
  if (existsSync(process.env.FRONTIER_FAKE_ACCOUNT)) {
    const account = JSON.parse(readFileSync(process.env.FRONTIER_FAKE_ACCOUNT, "utf8"));
    unlinkSync(process.env.FRONTIER_FAKE_ACCOUNT);
    console.log(JSON.stringify({ account }));
    process.exit(0);
  }
  console.error("Error: failed to query account: rpc error: code = Unknown desc = account not found: unknown request");
  process.exit(1);
} else if (args[0] === "tx" && args[1] === "zerone_auth" && args[2] === "register-account") {
  appendFileSync(process.env.FRONTIER_FAKE_LOG, JSON.stringify(args) + "\\n");
  const publicHex = args[4];
  writeFileSync(process.env.FRONTIER_FAKE_ACCOUNT, JSON.stringify({
    address: ${JSON.stringify(registrationSender)},
    did: args[3],
    public_key: publicHex,
    account_type: args[5],
    operational_key_hash: createHash("sha256").update(Buffer.from(publicHex, "hex")).digest("hex"),
    operational_public_key: publicHex,
    operational_key_version: 1,
  }));
  console.log(JSON.stringify({ txhash: "ABCDEF", code: 0 }));
} else if (args[0] === "query" && args[1] === "tx") {
  console.log(JSON.stringify({ height: "1", code: 0, events: [] }));
} else {
  console.error("unexpected fake zeroned invocation: " + JSON.stringify(args));
  process.exit(2);
}
`);
      chmodSync(fakeZeroned, 0o700);

      const intakePath = new URL("../intake.ts", import.meta.url).pathname;
      const run = Bun.spawnSync({
        cmd: [process.execPath, intakePath, "setup", "--network", "zerone-testnet-1", "--live-ack=zerone-testnet-1"],
        cwd: new URL("..", import.meta.url).pathname,
        env: {
          ...process.env,
          ZERONED_BIN: fakeZeroned,
          FRONTIER_FAKE_LOG: invocationLog,
          FRONTIER_FAKE_ACCOUNT: join(scratch, "registered-account.json"),
          FRONTIER_INTAKE_STATE_DIR: scratch,
        },
        stdout: "pipe",
        stderr: "pipe",
      });
      expect(run.exitCode, run.stderr.toString()).toBe(0);

      const invocations = readFileSync(invocationLog, "utf8").trim().split("\n")
        .map(line => JSON.parse(line) as string[]);
      expect(invocations).toHaveLength(5);
      for (const invocation of invocations) {
        const registerIndex = invocation.indexOf("register-account");
        const did = invocation[registerIndex + 1] as string;
        const publicHex = invocation[registerIndex + 2] as string;
        const accountType = invocation[registerIndex + 3] as "agent";
        const signatureHex = invocation[registerIndex + 4] as string;
        expect(invocation[registerIndex + 5]).toBe("--from");
        expect(did).toBe(`did:zrn:${publicHex}`);
        expect(signatureHex).toMatch(/^[0-9a-f]{128}$/);

        const proofBytes = accountRegistrationProofSignBytes({
          chainId: "zerone-testnet-1",
          sender: registrationSender,
          publicHex,
          accountType,
          metadata: "",
        });
        const identityFile = invocation[registerIndex + 6] as string;
        const persistedPath = join(scratch, `zerone-testnet-1.${identityFile}.ed25519.json`);
        const persisted = parsePersistedIdentityKey(readFileSync(persistedPath, "utf8"), identityFile);
        const privateKey = createPrivateKey({
          key: Buffer.from(persisted.private_pkcs8_hex, "hex"),
          format: "der",
          type: "pkcs8",
        });
        expect(verify(null, proofBytes, privateKey, Buffer.from(signatureHex, "hex"))).toBe(true);
      }
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  }, 15_000);
});

describe("register", () => {
  const register = loadRegister(defaultRegisterPath());

  test("the checked-in register validates cleanly", () => {
    expect(validateRegister(register)).toEqual([]);
  });

  test("admission partitions are sane", () => {
    const asserts = admissibleEntries(register);
    expect(asserts.length).toBeGreaterThanOrEqual(20);
    const batch = firstBatch(register);
    expect(batch.length).toBe(9);
    expect(batch[0]?.id).toBe("pfr-marton-2023");
    for (const entry of register.entries) {
      expect(["assert", "conjecture", "withhold"]).toContain(entry.chain.intent);
      if (entry.chain.intent === "conjecture") expect(entry.status).toBe("open_conjecture");
      if (entry.status === "preprint_under_review" || entry.status === "announced_unverified") {
        expect(entry.chain.intent).toBe("withhold");
      }
    }
  });

  test("every statement fits the live chain limits", () => {
    for (const entry of register.entries) {
      expect(entry.statement.length).toBeGreaterThanOrEqual(20);
      expect(entry.statement.length).toBeLessThanOrEqual(1000);
    }
  });

  test("refuted entries assert the refutation, never the refuted statement", () => {
    for (const entry of register.entries) {
      if (entry.status === "refuted" && entry.chain.intent === "assert") {
        expect(entry.chain.category).toBe("refutation");
        expect(/false|refut|disproof|disprove|counterexample/i.test(entry.statement)).toBe(true);
      }
    }
  });

  test("relations are validated: vocabulary, target shape, justification, contradicts ban", () => {
    const broken = structuredClone(register);
    const entry = broken.entries.find(candidate => candidate.chain.intent === "assert");
    if (!entry) throw new Error("no assert entry");
    entry.chain.relations = [
      { relation: "contradicts", target: "a".repeat(32), justification: "x" },
      { relation: "cites", target: "not-a-fact-id", justification: "x" },
      { relation: "cites", target: "entry:nonexistent-entry", justification: "x" },
      { relation: "cites", target: "b".repeat(32), justification: "" },
    ];
    const issues = validateRegister(broken);
    expect(issues.some(issue => issue.problem.includes("contradicts is deliberately excluded"))).toBe(true);
    expect(issues.some(issue => issue.problem.includes("neither a 32-hex fact id"))).toBe(true);
    expect(issues.some(issue => issue.problem.includes("unknown register entry"))).toBe(true);
    expect(issues.some(issue => issue.problem.includes("no justification"))).toBe(true);
    expect(issues.some(issue => issue.problem.includes("more than 3 relations"))).toBe(true);
    entry.chain.relations = [{ relation: "cites", target: "c".repeat(32), justification: "the paper builds on it" }];
    expect(validateRegister(broken)).toEqual([]);
  });

  test("validator catches broken entries", () => {
    const broken = structuredClone(register);
    const entry = broken.entries[0];
    if (!entry) throw new Error("register empty");
    entry.statement = "too short";
    entry.chain.domain = "physics";
    const issues = validateRegister(broken);
    expect(issues.some(issue => issue.problem.includes("statement length"))).toBe(true);
    expect(issues.some(issue => issue.problem.includes("not mathematics"))).toBe(true);
  });
});

describe("panel", () => {
  test("commit hash matches the ceremony recipe", () => {
    // printf 'ZRN.commit.v1:round-1:accept:0:aabb' | shasum -a 256
    expect(commitHash("round-1", "accept", "aabb")).toBe(
      "a5b5ed5bb99aeaaaec1fbc3d2e2beda96d066e7a61d3cfe47d9f2bbdea3af85a",
    );
  });

  const params = { commit_blocks: 200, reveal_blocks: 200, aggregation_blocks: 50 };
  function round(overrides: Partial<RoundState>): RoundState {
    return {
      entry_id: "e",
      claim_id: "c",
      round_id: "r",
      submit_height: 1000,
      submit_tx: "T",
      vote: "accept",
      verifiers: { v1: { salt: newSalt() }, v2: { salt: newSalt() } },
      stage: "submitted",
      notes: [],
      ...overrides,
    };
  }

  test("commits inside the commit window", () => {
    const actions = decideActions(round({}), 1010, params);
    expect(actions.every(action => action.kind === "commit")).toBe(true);
    expect(actions).toHaveLength(2);
  });

  test("waits for reveal phase after all commits", () => {
    const committed = round({
      verifiers: { v1: { salt: "a", commit_tx: "x" }, v2: { salt: "b", commit_tx: "y" } },
    });
    const during = decideActions(committed, 1100, params);
    expect(during[0]?.kind).toBe("wait");
    const revealPhase = decideActions(committed, 1250, params);
    expect(revealPhase.every(action => action.kind === "reveal")).toBe(true);
  });

  test("reports a missed commit window instead of pretending", () => {
    const actions = decideActions(round({}), 1199, params);
    expect(actions[0]?.kind).toBe("missed_window");
  });

  test("checks resolution only after aggregation margin", () => {
    const done = round({
      verifiers: { v1: { salt: "a", commit_tx: "x", reveal_tx: "p" }, v2: { salt: "b", commit_tx: "y", reveal_tx: "q" } },
    });
    expect(decideActions(done, 1440, params)[0]?.kind).toBe("wait");
    expect(decideActions(done, 1460, params)[0]?.kind).toBe("check_resolution");
  });

  test("resolved rounds require no actions", () => {
    expect(decideActions(round({ stage: "resolved_accepted" }), 2000, params)).toEqual([]);
  });
});
