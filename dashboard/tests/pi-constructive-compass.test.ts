import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import {
  buildConstructiveTreeIndex,
  parseConstructiveIntelligenceTreeJson,
} from "../src/constructive-tree";
import {
  PI_CONSTRUCTIVE_COMPASS_PATHS,
  PiConstructiveCompassError,
  resolvePiConstructiveCompassPaths,
} from "../src/pi-constructive-compass";

const DASHBOARD_ROOT = fileURLToPath(new URL("..", import.meta.url));
const readDashboardFile = (path: string): string =>
  readFileSync(new URL(path, `${new URL("..", import.meta.url).href}/`), "utf8");
const canonicalRaw = readFileSync(
  `${DASHBOARD_ROOT}/public/standards/constructive-intelligence-tree.v1.json`,
  "utf8",
);

describe("Pi Constructive Compass path manifest", () => {
  const tree = parseConstructiveIntelligenceTreeJson(canonicalRaw);
  const index = buildConstructiveTreeIndex(tree);
  const resolveCanonical = (id: string) => {
    const capability = index.byId.get(id);
    return capability ? { id: capability.id, title: capability.title } : null;
  };

  it("resolves three fixed trails and nine unique canonical capabilities", () => {
    const paths = resolvePiConstructiveCompassPaths(resolveCanonical);
    assert.equal(paths.length, 3);
    assert.deepEqual(
      paths.map((path) => path.id),
      ["follow-question", "guard-edges", "make-exact"],
    );
    assert.ok(paths.every((path) => path.capabilities.length === 3));

    const capabilities = paths.flatMap((path) => path.capabilities);
    assert.equal(capabilities.length, 9);
    assert.equal(new Set(capabilities.map((capability) => capability.id)).size, 9);
    capabilities.forEach((capability) => {
      const canonical = index.byId.get(capability.id);
      assert.ok(canonical);
      assert.equal(capability.title, canonical.title);
      assert.notEqual(canonical.stage, "quest");
      assert.equal(canonical.rewardEligibility, "qualification-only");
      assert.equal(canonical.acceptance, null);
    });
    assert.ok(Object.isFrozen(paths));
    assert.ok(paths.every(Object.isFrozen));
  });

  it("fails the entire manifest closed on missing or mismatched tree data", () => {
    const firstCapability = PI_CONSTRUCTIVE_COMPASS_PATHS[0].capabilityIds[0];
    assert.throws(
      () =>
        resolvePiConstructiveCompassPaths((id) =>
          id === firstCapability ? null : resolveCanonical(id),
        ),
      new RegExp(`Canonical constructive capability unavailable: ${firstCapability}`),
    );
    assert.throws(
      () =>
        resolvePiConstructiveCompassPaths((id) => {
          const capability = resolveCanonical(id);
          return capability ? { ...capability, id: `${id}-drifted` } : null;
        }),
      PiConstructiveCompassError,
    );
  });
});

describe("Pi Constructive Compass privacy and activation boundary", () => {
  const compassSource = readDashboardFile("src/pi-constructive-compass.ts");
  const piUiSource = readDashboardFile("src/pi-ui.ts");
  const mainSource = readDashboardFile("src/main.ts");
  const treeSource = readDashboardFile("src/constructive-tree.ts");
  const html = readDashboardFile("index.html");

  it("is independently default-off and available only inside authenticated Pi UI", () => {
    assert.match(
      mainSource,
      /VITE_PI_CONSTRUCTIVE_COMPASS_ENABLED === "true"/,
    );
    assert.match(
      mainSource,
      /PI_CONSTRUCTIVE_COMPASS_ENABLED\s*\? constructiveTreeReady\.then/,
    );
    assert.match(
      piUiSource,
      /if \(!compassOptions \|\| !session\.authenticated\) return/,
    );
    assert.match(
      piUiSource,
      /constructiveCompass\?\.setAuthenticated\(session\.authenticated\)/,
    );
    assert.match(html, /id="pi-constructive-compass"[\s\S]*?hidden/);
  });

  it("keeps selection in page memory and clears it when authentication ends", () => {
    assert.doesNotMatch(
      compassSource,
      /localStorage|sessionStorage|indexedDB|caches\.|fetch\(|XMLHttpRequest|sendBeacon/,
    );
    assert.doesNotMatch(
      compassSource,
      /wallet|address|signature|transaction|payment|reward/i,
    );
    assert.match(compassSource, /form\.reset\(\)/);
    assert.match(compassSource, /if \(!authenticated\) reset\(\)/);
    assert.match(compassSource, /trail\.replaceChildren\(\)/);
    assert.match(
      compassSource,
      /This control did not send the choice to Pi or Zerone servers/,
    );
  });

  it("requires an explicit accessible choice and page-only consent", () => {
    assert.match(html, /<fieldset aria-describedby="pi-compass-privacy">/);
    assert.match(html, /<legend>Pick one temporary learning intention<\/legend>/);
    assert.equal(
      html.match(/name="pi-compass-path"/g)?.length,
      PI_CONSTRUCTIVE_COMPASS_PATHS.length,
    );
    assert.match(
      html,
      /id="pi-compass-consent" type="checkbox" required/,
    );
    assert.match(html, /id="pi-compass-status" role="status" aria-live="polite"/);
    assert.match(html, /id="pi-compass-result"[\s\S]*?tabindex="-1"[\s\S]*?hidden/);
    for (const claim of [
      "not who you are",
      "This control does not send the choice",
      "no KYC or unique-human",
      "no identity, profile, evaluation, evidence",
      "qualification, reward, payment, wallet action",
      "chain write",
    ]) {
      assert.match(html, new RegExp(claim));
    }
  });

  it("opens only capabilities resolved by the loaded public explorer", () => {
    assert.match(
      treeSource,
      /resolveCapability\(id: string\): ConstructiveTreeCapabilityReference \| null/,
    );
    assert.match(treeSource, /if \(!index\.byId\.has\(id\)\) return false/);
    assert.match(mainSource, /constructiveTree\.resolveCapability\(id\)/);
    assert.match(mainSource, /constructiveTree\.openCapability\(id\)/);
    assert.doesNotMatch(compassSource, /window\.location|history\.|document\.cookie/);
  });
});
