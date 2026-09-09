import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { buildNodeGuideProfile } from "./node-guide-profile";

// An install recipe must point to bytes that actually exist at its source pin.
// The deployment workflow additionally requires a clean, merged checkout.
export function loadNodeGuideProfile(repositoryRoot: string) {
  const git = (...args: string[]) => execFileSync("git", args, {
    cwd: repositoryRoot, stdio: ["ignore", "pipe", "pipe"],
  });
  const sourceCommit = git("rev-parse", "HEAD").toString("utf8").trim();
  const helperPath = "scripts/local-node.py";
  let committedHelper: Buffer;
  try {
    committedHelper = git("show", `${sourceCommit}:${helperPath}`);
  } catch {
    throw new Error("Node guide requires scripts/local-node.py committed at HEAD; commit the reviewed helper before building the guide.");
  }
  if (!committedHelper.equals(readFileSync(resolve(repositoryRoot, helperPath)))) {
    throw new Error("Node guide helper differs from HEAD; commit the reviewed helper before building the guide.");
  }
  return buildNodeGuideProfile({
    sourceCommit,
    helperSha256: createHash("sha256").update(committedHelper).digest("hex"),
  });
}
