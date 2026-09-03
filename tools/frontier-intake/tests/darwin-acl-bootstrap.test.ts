import { expect, test } from "bun:test";
import {
  chmodSync,
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

test.skipIf(process.platform !== "darwin")(
  "Frontier rejects ACLs on its native inspector and inspector directory",
  async () => {
    const scratch = mkdtempSync(join(tmpdir(), "frontier-acl-bootstrap-"));
    try {
      const sourceDirectory = join(scratch, "src");
      const buildDirectory = join(scratch, "build");
      mkdirSync(sourceDirectory, { mode: 0o700 });
      mkdirSync(buildDirectory, { mode: 0o700 });
      const bundle = await Bun.build({
        entrypoints: [fileURLToPath(new URL("../src/account-registration.ts", import.meta.url))],
        outdir: sourceDirectory,
        target: "bun",
        naming: "account-registration.js",
      });
      expect(bundle.success, bundle.logs.map(log => log.message).join("\n")).toBe(true);

      const helper = join(buildDirectory, "darwin-acl-check");
      copyFileSync(
        fileURLToPath(new URL("../build/darwin-acl-check", import.meta.url)),
        helper,
      );
      chmodSync(helper, 0o555);
      const identityPath = join(scratch, "state", "identity.json");
      const runner = join(scratch, "run.ts");
      writeFileSync(runner, `
import { loadOrCreatePersistedIdentityKey } from ${JSON.stringify(pathToFileURL(join(sourceDirectory, "account-registration.js")).href)};
try {
  loadOrCreatePersistedIdentityKey(${JSON.stringify(identityPath)}, "bootstrap-test");
  process.exit(0);
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(2);
}
`, { mode: 0o700 });

      const addSpecialMode = Bun.spawnSync({
        cmd: ["/bin/chmod", "4555", helper],
        stdout: "pipe",
        stderr: "pipe",
      });
      expect(addSpecialMode.exitCode, addSpecialMode.stderr.toString()).toBe(0);
      const specialModeRejected = Bun.spawnSync({
        cmd: [process.execPath, runner],
        stdout: "pipe",
        stderr: "pipe",
      });
      expect(specialModeRejected.exitCode).toBe(2);
      expect(specialModeRejected.stderr.toString()).toContain(
        "exact mode 0555 with no special bits",
      );
      expect(existsSync(identityPath)).toBe(false);
      chmodSync(helper, 0o555);

      const addHelperAcl = Bun.spawnSync({
        cmd: ["/bin/chmod", "+a", "everyone allow read", helper],
        stdout: "pipe",
        stderr: "pipe",
      });
      expect(addHelperAcl.exitCode, addHelperAcl.stderr.toString()).toBe(0);
      const helperRejected = Bun.spawnSync({
        cmd: [process.execPath, runner],
        stdout: "pipe",
        stderr: "pipe",
      });
      expect(helperRejected.exitCode).toBe(2);
      expect(helperRejected.stderr.toString()).toContain("ACL helper has an extended ACL");
      expect(existsSync(identityPath)).toBe(false);

      const clearHelperAcl = Bun.spawnSync({
        cmd: ["/bin/chmod", "-N", helper],
        stdout: "pipe",
        stderr: "pipe",
      });
      expect(clearHelperAcl.exitCode, clearHelperAcl.stderr.toString()).toBe(0);
      const addDirectoryAcl = Bun.spawnSync({
        cmd: ["/bin/chmod", "+a", "everyone allow list,search,read", buildDirectory],
        stdout: "pipe",
        stderr: "pipe",
      });
      expect(addDirectoryAcl.exitCode, addDirectoryAcl.stderr.toString()).toBe(0);
      const directoryRejected = Bun.spawnSync({
        cmd: [process.execPath, runner],
        stdout: "pipe",
        stderr: "pipe",
      });
      expect(directoryRejected.exitCode).toBe(2);
      expect(directoryRejected.stderr.toString()).toContain(
        "ACL helper directory has an extended ACL",
      );
      expect(existsSync(identityPath)).toBe(false);
    } finally {
      rmSync(scratch, { recursive: true, force: true });
    }
  },
  15_000,
);
