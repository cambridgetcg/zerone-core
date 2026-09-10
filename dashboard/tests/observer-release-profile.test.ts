import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { it } from "node:test";
import { buildObserverRelease, type ObserverPublication } from "../observer-release-profile";
import { buildNodeGuideProfile } from "../node-guide-profile";
import { nodeGuidePage } from "../node-guide-page";

const publication: ObserverPublication = {
  schema: "zerone.observer-publication/v1",
  releaseId: "zerone-1-observer-test",
  toolingCommit: "a1".repeat(20),
  executableBaseCommit: "b2".repeat(20),
  executableSha256: "c3".repeat(32),
  manifestSha256: "d4".repeat(32),
  archiveSha256: "e5".repeat(32),
  receiptSha256: "f6".repeat(32),
  expiresAt: "2026-09-16T19:19:13Z",
  checkpointHeight: 1262000,
  checkpointHash: "A7".repeat(32),
};

it("binds an observer recipe to immutable signed artifacts and preserves closed admission", () => {
  const profile = buildNodeGuideProfile({ sourceCommit: "a8".repeat(20), helperSha256: "b9".repeat(32), observerPublication: publication });
  const observer = profile.live.replicaInstallation.release!;
  assert.equal(profile.live.replicaInstallation.availability, "signed-legacy-observer");
  assert.equal(observer.chainId, "zerone-1");
  assert.equal(observer.validatorPower, 0);
  assert.equal(observer.validatorAdmission, false);
  assert.equal(observer.transactionAdmission, false);
  assert.equal(observer.independentTrustAnchor, false);
  assert.equal(observer.completeHistoricalReplay, false);
  assert.equal(observer.executableSourceStatus, "reproduced-observer-patch");
  assert.equal(observer.executableBaseIsUnpatchedSource, true);
  assert.equal(observer.sameExecutableAsLiveValidator, false);
  assert.notEqual(profile.source.commit, observer.executableBaseCommit);
  assert.equal(profile.live.validatorJoining.availability, "not-open");
  assert.equal(profile.live.newAccountAdmission.availability, "paused");
  assert.match(observer.verifyCommand, new RegExp(`checkout --detach ${publication.toolingCommit}`, "u"));
  assert.ok(observer.verifyCommand.includes(publication.archiveSha256));
  assert.match(observer.verifyCommand, /package\.py unpack/u);
  assert.match(observer.verifyCommand, /verify-rehearsal\.py --bundle/u);
  assert.ok(observer.verifyCommand.includes(publication.receiptSha256));
  assert.ok(observer.verifyCommand.includes(`${observer.receiptUrl}.sig`));
  for (const line of observer.verifyCommand.split("\n")) {
    if (line.startsWith("python3")) assert.match(line, /^python3 -I /u);
  }
  assert.doesNotMatch(observer.verifyCommand, /curl[^\n]*\|\s*(?:sh|bash)/u);
  const html = nodeGuidePage(profile);
  assert.ok(html.includes(publication.expiresAt));
  assert.ok(html.includes(observer.releaseUrl));
  assert.ok(html.includes("command-observer-verify"));
  assert.ok(html.includes("command-observer-start"));
  assert.ok(html.includes("zero voting power"));
  assert.doesNotMatch(html, /supported live replica installation and validator joining are not open/u);
});

it("refuses moving source pins, unsafe release names, added capabilities and malformed checkpoints", () => {
  for (const change of [
    { toolingCommit: "main" }, { archiveSha256: "0".repeat(64) },
    { releaseId: "zerone-1-observer-x'; touch /tmp/no" },
    { releaseId: "zerone-1-observer-.bad" }, { releaseId: `zerone-1-observer-${"a".repeat(97)}` },
    { checkpointHeight: -1 }, { checkpointHash: "a7".repeat(32) },
    { checkpointHeight: "1262000" }, { checkpointHash: "0".repeat(64) },
    { expiresAt: "later" }, { expiresAt: "2026-02-30T12:00:00Z" }, { validatorAdmission: true },
    { toolingCommit: [publication.toolingCommit] }, { archiveSha256: [publication.archiveSha256] },
  ]) {
    assert.throws(() => buildObserverRelease({ ...publication, ...change } as ObserverPublication));
  }
});

it("retains the unpublished state when no verified publication record exists", () => {
  const profile = buildNodeGuideProfile({ sourceCommit: "a8".repeat(20), helperSha256: "b9".repeat(32) });
  assert.equal(profile.live.replicaInstallation.release, null);
  assert.equal(profile.live.replicaInstallation.availability, "not-published");
  assert.doesNotMatch(nodeGuidePage(profile), /command-observer-start/u);
  for (const malformed of [null, false, 0, "", []]) {
    assert.throws(() => buildNodeGuideProfile({ sourceCommit: "a8".repeat(20), helperSha256: "b9".repeat(32),
      observerPublication: malformed as unknown as ObserverPublication }));
  }
});

it("stops the copied verification recipe immediately if source acquisition fails", () => {
  const directory = mkdtempSync(join(tmpdir(), "zerone-observer-command-test-"));
  try {
    writeFileSync(join(directory, "git"), "#!/bin/sh\nexit 68\n", { mode: 0o700 });
    writeFileSync(join(directory, "curl"), "#!/bin/sh\ntouch unexpected-download\n", { mode: 0o700 });
    const observer = buildObserverRelease(publication);
    assert.throws(() => execFileSync("/bin/sh", ["-c", observer.verifyCommand], {
      cwd: directory, env: { PATH: `${directory}:/usr/bin:/bin` }, stdio: "pipe",
    }), (error: unknown) => error !== null && typeof error === "object" && "status" in error && error.status === 68);
    assert.equal(existsSync(join(directory, "unexpected-download")), false);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});
