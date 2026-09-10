import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { researchDestination } from "../src/research-links";

const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8");
const home = read("index.html");
const library = read("research/index.html");
const main = read("src/main.ts");
const runtime = read("src/research.ts");
const oldLinks = [...home.matchAll(/<a\b[^>]*\bid="([^"]+)" data-research-destination href="([^"]+)"/gu)];

describe("optional research library", () => {
  it("preserves every moved fragment and provides exact native fallback links", () => {
    const ids = oldLinks.map((match) => match[1]!);
    assert.equal(ids.length, 102);
    // Complete moved ID set captured from the reviewed predecessor632f372f.
    assert.equal(createHash("sha256").update([...ids].sort().join("\n")).digest("hex"),
      "09684f4f456fb77b1e047ff75b0460b8c731c663a3e27c9b0e9ec398c94f9483");
    for (const [, id, href] of oldLinks) {
      assert.equal(href, `/research/#${id}`);
      assert.equal([...home.matchAll(/\bid="([^"]+)"/gu)].filter((match) => match[1] === id).length, 1);
      assert.equal([...library.matchAll(/\bid="([^"]+)"/gu)].filter((match) => match[1] === id).length, 1);
    }
    assert.match(read("src/styles.css"), /\.research-index \.research-legacy-target:target\s*\{[^}]*display: inline-flex/u);
  });

  it("routes only explicitly retained same-fragment research links", () => {
    const root = (id: string, href: string, retained = true) => ({
      getElementById: (query: string) => query !== id ? null : {
        hasAttribute: (name: string) => retained && name === "data-research-destination",
        getAttribute: (name: string) => name === "href" ? href : null,
      } as unknown as HTMLElement,
    });
    for (const [, id, href] of oldLinks) assert.equal(researchDestination(`#${id}`, root(id!, href!)), href);
    for (const destination of ["https://attacker.example/", "//attacker.example/#skills", "/research/#other", "/research/?account=secret#skills", "javascript:alert(1)"]) {
      assert.equal(researchDestination("#skills", root("skills", destination)), null);
    }
    assert.equal(researchDestination("#skills", root("skills", "/research/#skills", false)), null);
    for (const hash of ["", "skills", "#unknown", "#wallet", "#understanding", "#" + "a".repeat(201)]) {
      assert.equal(researchDestination(hash, root("skills", "/research/#skills")), null);
    }
  });

  it("keeps specialist startup outside the core entrypoint and keeps core controls at home", () => {
    const specialists = ["branch-flow", "constructive-tree", "research-commons", "life-sciences-tree", "quantum-season", "math-frontier", "fold-to-fire", "life-garden", "relational-topology", "correspondence-geometry", "explicit-invariant-discipline"];
    for (const module of specialists) {
      assert.ok(runtime.includes(`from "./${module}"`), module);
      assert.ok(!main.includes(`from "./${module}"`), module);
    }
    for (const id of ["onboarding", "wallet", "activity", "network", "authority-geometry-root", "knowledge-geometry-root", "frontier-participation-root"]) {
      assert.ok(home.includes(`id="${id}"`), id);
      assert.ok(!library.includes(`id="${id}"`), id);
    }
    assert.doesNotMatch(runtime, /from "\.\/(?:api|wallet|pi-ui|onboarding|config)"|fetch\(|setInterval\(/u);
    assert.doesNotMatch(library, /wallet-connect|id="(?:send-form|feegrant-form|contribute|pi-constructive-compass)"|\/src\/main\.ts/u);
    assert.match(library, /href="https:\/\/zerone\.ai\/#understanding"/u);
  });

  it("loads the optional Pi tree only behind both existing flags and keeps selection in the page", () => {
    const initializer = main.slice(main.indexOf("async function initialisePiPilotIfEnabled"), main.indexOf('\ndocument.querySelectorAll<HTMLButtonElement>(".wallet-connect")'));
    assert.ok(initializer.indexOf("if (!PI_PILOT_ENABLED) return;") < initializer.indexOf('import("./constructive-tree")'));
    assert.match(initializer, /const constructiveTreeReady = PI_CONSTRUCTIVE_COMPASS_ENABLED\s*\? import\("\.\/constructive-tree"\)/u);
    assert.match(initializer, /constructiveTree\.resolveCapability\(id\)/u);
    assert.match(initializer, /constructiveTree\.openCapability\(id\)/u);
    assert.doesNotMatch(initializer, /location\.|history\.|Storage|\/research\//u);
    assert.doesNotMatch(home, /<div[^>]*id="(?:constructive-tree-root|pi-constructive-tree-root)"/u);
  });
});
