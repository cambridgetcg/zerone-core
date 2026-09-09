import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { buildNodeGuideProfile } from "./node-guide-profile";
import { buildObserverRelease, type ObserverPublication } from "./observer-release-profile";

function parseObserverPublication(raw: Buffer): ObserverPublication {
  if (raw.length > 16 * 1024) throw new Error("Observer publication metadata exceeds 16 KiB");
  const text = raw.toString("utf8");
  const value = JSON.parse(text) as ObserverPublication;
  // This contract is flat and scalar. Validate that shape first, then scan
  // complete JSON string tokens followed by a colon, including escaped keys.
  // JSON.parse alone would silently replace earlier duplicate fields.
  buildObserverRelease(value);
  const seen = new Set<string>();
  for (const match of text.matchAll(/"(?:\\[\s\S]|[^"\\])*"/gu)) {
    if (!/^\s*:/u.test(text.slice(match.index! + match[0].length))) continue;
    const key = JSON.parse(match[0]) as string;
    if (seen.has(key)) throw new Error("Observer publication contains duplicate JSON keys");
    seen.add(key);
  }
  return value;
}

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
  const publicationPath = "dashboard/observer-release.json";
  let observerPublication: ObserverPublication | undefined;
  const committedPublication = git("ls-tree", "--name-only", sourceCommit, "--", publicationPath).length > 0;
  const workingPublication = existsSync(resolve(repositoryRoot, publicationPath));
  if (committedPublication !== workingPublication) {
    throw new Error("Observer publication metadata must match its presence at HEAD; commit the reviewed publication change before building.");
  }
  if (committedPublication) {
    const committed = git("show", `${sourceCommit}:${publicationPath}`);
    if (!committed.equals(readFileSync(resolve(repositoryRoot, publicationPath)))) {
      throw new Error("Observer publication metadata differs from HEAD; commit the verified release pins before building.");
    }
    observerPublication = parseObserverPublication(committed);
  }
  return buildNodeGuideProfile({
    sourceCommit,
    helperSha256: createHash("sha256").update(committedHelper).digest("hex"),
    observerPublication,
  });
}
