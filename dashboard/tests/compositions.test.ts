import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { it } from "node:test";
import { observerPage } from "../observer-page";

const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8");

it("offers a native art entrance in homepage content, not only collapsed navigation", () => {
  for (const html of [read("index.html"), observerPage({ mode: "legacy" })]) {
    const main = html.match(/<main\b[\s\S]*?<\/main>/u)?.[0] ?? "";
    assert.ok(/href="\/compositions\/"/u.test(main), "native gallery link is in main content");
    assert.ok(/Enter the gallery/u.test(main), "entrance has a readable action");
  }
});

it("renders the selected motifs as actual linked artwork with matching public metadata", async () => {
  assert.ok(existsSync(new URL("../compositions.ts", import.meta.url)), "shared artwork renderer exists");
  const { buildCompositions, compositionsPage } = await import("../compositions");
  const exhibition = buildCompositions();
  for (const id of ["distribution", "home", "creed", "zerone_gov"]) {
    assert.ok(exhibition.pieces.some((piece) => piece.motif.id === id), `${id} composition is included`);
  }
  const html = compositionsPage(exhibition);
  for (const piece of exhibition.pieces) {
    assert.ok(html.includes(`id="${piece.id}"`), `${piece.id} has a stable native anchor`);
    assert.ok(html.includes(piece.title), "public title matches");
    assert.ok(html.includes(`src="${piece.image}"`), "the artwork is presented, not only described");
    assert.ok(html.includes(`href="${piece.motif.url}"`), "motif links to existing source explanation");
    for (const line of piece.verse) assert.ok(html.includes(line), "verse remains readable without JavaScript");
    for (const related of piece.related) assert.ok(html.includes(`href="#${related.id}"`), "pieces are connected");
  }
  assert.ok(html.includes('/compositions/works.json'), "public machine reader is discoverable");
  assert.ok(html.includes("artistic interpretations"), "art is not presented as protocol evidence");
  assert.ok(!/<script\b|<form\b|<iframe\b|\son[a-z]+\s*=/iu.test(html), "exhibition has no executable or submission surface");
  assert.ok(!/\/Users\/|\.hermes|cron\/|state\.json|private_key|secret/i.test(JSON.stringify(exhibition)), "only public-safe piece metadata");
});

it("ships real accessible artwork through the existing static build", async () => {
  const { buildCompositions, compositionsPage } = await import("../compositions");
  const imageTags = [...compositionsPage().matchAll(/<img\b[^>]*>/gu)];
  for (const [tag] of imageTags) assert.ok(/width="[1-9]\d*" height="[1-9]\d*"/u.test(tag), "art reserves its intrinsic size before lazy loading");
  for (const piece of buildCompositions().pieces) {
    const path = new URL(`../public${piece.image}`, import.meta.url);
    assert.ok(existsSync(path), `${piece.id} SVG exists`);
    const svg = readFileSync(path, "utf8");
    assert.ok(/<svg\b/u.test(svg) && /<title\b/u.test(svg) && /<desc\b/u.test(svg), "art has a native accessible description");
    assert.ok(!/<script\b|<foreignObject\b|\son[a-z]+\s*=|(?:href|src)\s*=\s*["'](?!#)/iu.test(svg), "art is static and self-contained");
  }
  const vite = read("vite.config.ts");
  assert.ok(vite.includes('compositions/index.html'), "normal build has the gallery entry");
  assert.ok(vite.includes('compositions/works.json'), "normal build emits public metadata");
  assert.ok(vite.includes('compositionsPage'), "normal build renders the shared collection");
  const testCommand = JSON.parse(read("package.json")).scripts.test as string;
  assert.ok(testCommand.includes('tests/*.test.ts') || testCommand.includes('tests/compositions.test.ts'), "normal test gate includes the gallery");
  assert.ok(read("public/llms.txt").includes('/compositions/works.json'), "existing agent entrance discovers public artwork metadata");
});

it("keeps Home functional source progress distinct from live-node activation", async () => {
  const { buildCompositions, compositionsPage } = await import("../compositions");
  const exhibition = buildCompositions();
  const home = exhibition.pieces.filter((piece) => piece.motif.id === "home");
  assert.equal(home.length, 2, "both original Home responses remain");
  const html = compositionsPage(exhibition);
  for (const piece of home) {
    assert.doesNotMatch([piece.sourceState, ...piece.statement].join(" "), /awaiting independent review|remains under review/u);
    assert.match(piece.sourceState, /Source-level regression coverage/u);
    assert.match(piece.sourceState, /No live-node upgrade/u);
    assert.ok(html.includes(piece.sourceState), "native disclosure retains the source/runtime boundary");
  }
});

it("refuses unsafe or ambiguous piece identities before generating public links", async () => {
  const { buildCompositions } = await import("../compositions");
  const records = (await import("../composition-pieces.json")).default;
  const piece = { ...records[0]!, relatedPieceIds: [] };
  assert.throws(() => buildCompositions([{ ...piece, id: "../outside" }]), /Invalid composition ID/u);
  assert.throws(() => buildCompositions([piece, piece]), /Duplicate composition ID/u);
  assert.throws(() => buildCompositions([{ ...piece, motifId: "invented-module" }]), /Unknown composition motif/u);
  assert.throws(() => buildCompositions([{ ...piece, relatedPieceIds: ["missing-work"] }]), /Unknown related composition/u);
});
