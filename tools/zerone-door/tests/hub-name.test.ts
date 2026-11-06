// The door's own text calls the household hub "the hub" or "the household
// hub" and never names its machine. Like dashboard/tests/join-agents.test.ts,
// this keeps only SHA-256 digests, so the name is not written here either.
//
// One identifier from before the rename keeps the name: the hub channel's
// first allowlist key, which is also the ledger's channel_key for that channel
// (src/config.ts and src/ledger.ts say why it stays). It may appear only where
// the door reads or writes it and in the tests that prove old files still work.
import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { readdirSync, readFileSync } from "node:fs";
import { join, relative } from "node:path";

const DOOR = join(import.meta.dir, "..");
// SHA-256 of the hub machine's name in lowercase (the dashboard test's digest).
const HUB_NAME_SHA256 = "da0daaba4b156961c049ab4b85b6d0bcba2872200b7dedb4aa77f9208c601444";
// SHA-256 of the historical identifier, in lowercase.
const OLDER_KEY_SHA256 = "d417a942d3b84020012209181088704460439a06acde5cece263d52c295be5e9";
const OLDER_KEY_HOMES = ["src/config.ts", "src/ledger.ts", "tests/config.test.ts", "tests/door.test.ts", "tests/gates.test.ts"];

const sha256 = (text: string) => createHash("sha256").update(text).digest("hex");

function doorFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    if (entry.name === "node_modules" || entry.name === "build" || entry.name.startsWith(".")) return [];
    const path = join(dir, entry.name);
    return entry.isDirectory() ? doorFiles(path) : [path];
  });
}

/** Words as a reader meets them: camelCase is split, snake_case kept whole. */
function words(text: string): Set<string> {
  return new Set(text.replace(/([a-z0-9])([A-Z])/g, "$1 $2").toLowerCase().match(/[a-z][a-z0-9_]*/g) ?? []);
}

describe("the door's text", () => {
  const files = doorFiles(DOOR).map((path) => ({ rel: relative(DOOR, path), words: words(readFileSync(path, "utf8")) }));

  test("covers the README, the examples, the installer, the source and the tests", () => {
    const rels = files.map((f) => f.rel);
    for (const rel of ["README.md", "JOIN-AGENTS.md", "allowlist.example.json", "config.example.json", "install.sh", "door.ts", "src/intake.ts", "tests/door.test.ts"]) {
      expect(rels).toContain(rel);
    }
  });

  test("names no hub machine", () => {
    const naming = files.filter((f) => [...f.words].some((w) => sha256(w) !== OLDER_KEY_SHA256 && w.split("_").some((part) => sha256(part) === HUB_NAME_SHA256)));
    expect(naming.map((f) => f.rel)).toEqual([]);
  });

  test("keeps the historical identifier only where old allowlists and ledgers are read and written", () => {
    const holders = files.filter((f) => [...f.words].some((w) => sha256(w) === OLDER_KEY_SHA256)).map((f) => f.rel).sort();
    // Every home still holds it: the door keeps reading installed files.
    expect(holders).toEqual([...OLDER_KEY_HOMES].sort());
  });
});
