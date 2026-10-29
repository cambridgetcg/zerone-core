import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { it } from "node:test";
import { fileURLToPath } from "node:url";
import { buildDevelopmentProfile, type DevelopmentPublication } from "../development-profile";
import { developmentPage } from "../development-page";
import publication from "../development-publication.json";
import { buildNodeGuideProfile } from "../node-guide-profile";
import { buildUnderstandGuide, understandPage } from "../understand-guide";

it("discloses public package access separately from dated runtime publication", () => {
  const value = buildDevelopmentProfile("a1".repeat(20), publication as DevelopmentPublication);
  assert.equal(value.repositoryAccess, "public");
  assert.match(value.accessNote, /download without a GitHub account/u);
  const html = developmentPage(value);
  assert.ok(html.includes('id="repository-access"'));
  assert.ok(html.includes(value.accessNote));
  assert.ok(html.indexOf('id="repository-access"') < html.indexOf('id="packages"'));
  assert.equal(value.availability, "operator-verified-publication");
  assert.equal(value.publication?.runtime_source_commit, publication.runtime_source_commit);
  assert.equal(value.effects.signing, false);
});

const sourceCommit = "a1".repeat(20);
const guide = () => buildUnderstandGuide(buildNodeGuideProfile({ sourceCommit, helperSha256: "b2".repeat(32) }));

it("keeps public guide and repository navigator module metadata consistent", () => {
  const census = JSON.parse(execFileSync("python3", [
    "-B", fileURLToPath(new URL("../../tools/source-map.py", import.meta.url)), "export",
  ], { encoding: "utf8", maxBuffer: 16 * 1024 * 1024 })) as { modules: Record<string, unknown>[] };
  const sourceModules = new Map(census.modules.map((module) => [module.id, module]));
  const modules = guide().modules.modules;
  assert.deepEqual(new Set(sourceModules.keys()), new Set(modules.map((module) => module.id)));
  for (const module of modules) {
    const source = sourceModules.get(module.id)!;
    for (const key of ["id", "label", "family", "group", "purpose", "owns", "boundary", "connections", "tags"] as const) {
      assert.deepEqual(source[key], module[key], `${module.id}.${key}`);
    }
    assert.equal(source.sourcePath, module.source.path);
  }
});

it("connects source modules and development status in the same static human and machine guide", () => {
  const value = guide();
  assert.ok(value.modules.development, "one shared development map is missing");
  const html = understandPage(value);
  assert.ok(html.includes('id="development-map"'));
  assert.ok(html.includes('href="#development-map"'));
  assert.match(value.modules.development.scope, /not a live/u);
  for (const track of value.modules.development.tracks) {
    assert.ok(html.includes(`id="development-${track.id}"`));
    assert.ok(html.includes(`data-status="${track.status}"`));
    assert.ok(html.includes(track.title));
    assert.ok(html.includes(`href="${track.link.url}"`));
    assert.ok(track.purpose && track.next && track.boundary);
  }
  assert.doesNotMatch(html, /<(?:script|form|input|textarea|iframe)\b|\son[a-z]+\s*=/iu);
  assert.ok(Object.values(value.effects).every((effect) => effect === false));
});

it("keeps current staking, governance holds and deposit custody distinct", () => {
  const byId = new Map(guide().modules.modules.map((module) => [module.id, module]));
  const connection = (from: string, to: string) => byId.get(from)?.connections.find((edge) => edge.module === to);
  assert.ok(connection("staking", "gov"), "SDK stake supplies current SDK voting weight");
  assert.ok(connection("gov", "staking"));
  assert.match(connection("gov", "zerone_gov")?.why ?? "", /post-incident hold/u);
  assert.ok(connection("zerone_gov", "zerone_staking"));
  assert.match(byId.get("zerone_gov")?.boundary ?? "", /LIP resolution does not refund deposits/u);
  assert.doesNotMatch(connection("zerone_gov", "bank")?.why ?? "", /settle/u);
  assert.match(connection("zerone_gov", "bank")?.why ?? "", /custody|balances/u);
  assert.ok(connection("zerone_gov", "vesting_rewards"));
  assert.equal(connection("zerone_staking", "auth"), undefined, "unused constructor attachment is not account access");
});

it("tags modules without renaming runtime IDs and discloses public source access", () => {
  const value = guide();
  const html = understandPage(value);
  assert.equal(value.source.repositoryAccess, "public");
  assert.match(html, /Source links open without a GitHub account/u);
  for (const module of value.modules.modules) {
    assert.deepEqual(module.tags, ["registered-source", module.family, module.group]);
    assert.ok(module.source.path && module.source.url.endsWith(module.source.path));
    assert.ok(html.includes(`data-module="${module.id}"`));
    assert.ok(html.includes(`data-tags="${module.tags.join(" ")}"`));
  }
  const byId = new Map(value.modules.modules.map((module) => [module.id, module]));
  for (const id of ["auth", "zerone_auth", "staking", "zerone_staking", "gov", "zerone_gov"]) assert.ok(byId.has(id));
  assert.equal(byId.get("zerone_staking")?.source.path, "x/staking/keeper/keeper.go");
});
