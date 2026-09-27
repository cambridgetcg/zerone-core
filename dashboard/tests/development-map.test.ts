import assert from "node:assert/strict";
import { it } from "node:test";
import { buildDevelopmentProfile, type DevelopmentPublication } from "../development-profile";
import { developmentPage } from "../development-page";
import publication from "../development-publication.json";
import { buildNodeGuideProfile } from "../node-guide-profile";
import { buildUnderstandGuide, understandPage } from "../understand-guide";

it("discloses package access restrictions separately from dated runtime publication", () => {
  const value = buildDevelopmentProfile("a1".repeat(20), publication as DevelopmentPublication);
  assert.equal(value.repositoryAccess, "restricted");
  assert.match(value.accessNote, /not anonymously downloadable/u);
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

it("tags modules without renaming runtime IDs and discloses restricted source access", () => {
  const value = guide();
  const html = understandPage(value);
  assert.equal(value.source.repositoryAccess, "restricted");
  assert.match(html, /Source links require repository access/u);
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
