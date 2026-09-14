import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createHash } from "node:crypto";
import { describe, it } from "node:test";
import { validateExport, selectSnapshot } from "../src/research-review-journal";

const root = new URL("../public/research/checkpoints/2026-09-14-demimetric-v1/", import.meta.url);
const read = (path: string) => new Uint8Array(readFileSync(new URL(path, root)));
const sha256 = (bytes: Uint8Array) => createHash("sha256").update(bytes).digest("hex");

describe("Published research checkpoint evidence", () => {
  it("keeps the exact original 81-entry journal compatible with local browser replay", async () => {
    const bytes = read("journal.json");
    assert.equal(await sha256(bytes), "86c8163599c0098aa43b4f3599a81b02a03557553fc595b67b026fdfe108704e");
    const journal = await validateExport(bytes);
    assert.equal(journal.header.collection_id, "ce6e799a-6713-41a9-b95f-57c50a223074");
    assert.equal(journal.entries.length, 81);
    assert.equal(journal.entries.at(-1)!.sha256, "23baf5b0a2e7f4a48fe2b86775ae2ef92e607f2d793270ff10219e61d2563847");
    const before = selectSnapshot(journal, 64);
    assert.ok(!before.entries.some(e => e.record.id.startsWith("local:restricted-repair:")));
    assert.ok(selectSnapshot(journal, 81).entries.some(e => e.record.id === "local:restricted-repair:strong-convergence-finding"));
  });

  it("joins each included artifact to its commitment and labels the private original receipt as a derivative", async () => {
    const decode = (path: string) => JSON.parse(new TextDecoder().decode(read(path)));
    const manifest = decode("MANIFEST.json") as { files: Array<{ path: string; bytes: number; sha256: string }> };
    assert.equal(await sha256(read("MANIFEST.json")), "dd80f7ca6ba15682301ee8d80c447d24eee79d80465a72f825572541ecad5995");
    assert.equal(manifest.files.length, 48);
    for (const file of manifest.files) {
      assert.ok(!file.path.startsWith("/") && !file.path.split("/").includes(".."));
      const bytes = read(file.path);
      assert.equal(bytes.byteLength, file.bytes, file.path);
      assert.equal(await sha256(bytes), file.sha256, file.path);
    }
    const index = decode("EVIDENCE-INDEX.json") as { entries: Array<{ status: string; path?: string; sha256: string | null; derivative?: { path: string; sha256: string } }> };
    const withheld = index.entries.filter(e => e.status === "original_withheld_private_paths");
    assert.equal(withheld.length, 1);
    assert.equal(withheld[0]!.sha256, "7906d600a1494b5697cdce84a17741fe999ff1a76408b631d534f5214eeef523");
    assert.equal(await sha256(read(withheld[0]!.derivative!.path)), withheld[0]!.derivative!.sha256);
    assert.notEqual(withheld[0]!.sha256, withheld[0]!.derivative!.sha256);
    for (const entry of index.entries.filter(e => e.status === "included_exact_bytes")) {
      assert.equal(await sha256(read(entry.path!)), entry.sha256);
    }
  });
});
