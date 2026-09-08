import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { readFileSync } from "node:fs";
import { parseNetworkProfile, observerLabel } from "../network-profile";
import { observerPage } from "../observer-page";
import { syntheticProfile } from "./observer-fixtures";
import { apiMiddleware } from "../functions/api/_middleware";

describe("release-bound network profile", () => {
  it("preserves legacy only when explicitly selected", () => {
    assert.deepEqual(parseNetworkProfile({ mode: "legacy" }), { mode: "legacy" });
    for (const value of [null, {}, { mode: "beta-active" }, { mode: "legacy", gatewayOrigin: "https://override.example.org" }]) {
      assert.equal(parseNetworkProfile(value).mode, "unconfigured");
    }
  });
  it("accepts separately marked synthetic, active and archive contracts", () => {
    for (const mode of ["preview", "beta-active", "beta-archive"] as const) {
      const value = parseNetworkProfile(syntheticProfile(mode));
      assert.equal(value.mode, mode);
    }
    assert.match(observerLabel(syntheticProfile()), /NOT PRODUCTION/);
  });
  it("rejects incomplete identities, authority assertions and unsafe coordinates", () => {
    const profile = syntheticProfile("beta-active");
    for (const change of [
      { chainId: "" }, { genesisSha256: "none" }, { releaseCommit: "main" },
      { releaseManifestSha256: "" }, { operatorVerification: undefined },
      { operatorVerification: { verifiedAt: "yesterday" } },
      { operatorVerification: { ...profile.operatorVerification, verifiedAt: "2026-02-31T07:00:00Z" } },
      { gatewayOrigin: "https://gateway..example.org" }, { walletEnabled: true },
      { gatewayOrigin: "http://gateway.example.org" }, { gatewayOrigin: "https://127.0.0.1" },
      { gatewayOrigin: "https://node.internal" }, { gatewayOrigin: "https://user:password@gateway.example.org" },
      { gatewayOrigin: "https://gateway.example.org/relay" }, { gatewayOrigin: "https://gateway.example.org?url=anything" },
      { gatewayOrigin: "https://gateway.example.org#anything" }, { releaseUrl: "javascript:alert(1)" },
      { checkpoint: { height: 10, blockHash: "a".repeat(64), appHash: "b".repeat(64) } },
    ]) assert.equal(parseNetworkProfile({ ...profile, ...change }).mode, "unconfigured", JSON.stringify(change));
    const archive = syntheticProfile("beta-archive");
    assert.equal(parseNetworkProfile({ ...archive, checkpoint: undefined }).mode, "unconfigured");
    assert.equal(parseNetworkProfile({ ...archive, checkpoint: { ...archive.checkpoint, height: 0 } }).mode, "unconfigured");
    assert.equal(parseNetworkProfile({ ...syntheticProfile(), gatewayOrigin: "https://gateway.example.org" }).mode, "unconfigured");
  });
  it("renders usable no-JS observer limits without legacy control or live claims", () => {
    for (const profile of [syntheticProfile(), syntheticProfile("beta-active"), syntheticProfile("beta-archive"), parseNetworkProfile({ mode: "beta-active" })]) {
      const html = observerPage(profile);
      assert.match(html, /<noscript>[\s\S]*no chain check has run/);
      assert.match(html, /not.*migration.*entitlement/i);
      assert.match(html, /operator-supplied references, not verified/);
      assert.match(html, /src="\/src\/observer.ts"/);
      assert.doesNotMatch(html, /wallet-connect|send-dialog|feegrant-form|\/src\/main.ts|Mainnet is live\./);
      assert.match(html, /id="lookup-submit" disabled/);
      assert.match(html, /id="lookup-value"[^>]*disabled/);
    }
  });
  it("discloses no selected network or requests without claiming a beta launch", () => {
    const html = observerPage(parseNetworkProfile({}));
    assert.match(html, /No network selected\. No network requests are made\./);
    assert.doesNotMatch(html, /Public custodial beta|one operator-controlled validator|f=0/);
    assert.match(html, /id="validator-value">Unknown</);
    assert.match(html, /id="observer-refresh" disabled/);
  });
  it("blocks alternate legacy API and Pi routes before dispatch for every observer profile", async () => {
    for (const profile of [syntheticProfile(), syntheticProfile("beta-active"), syntheticProfile("beta-archive"), parseNetworkProfile({})]) {
      for (const path of ["knowledge", "knowledge%2Fprivate", "%6bnowledge", "pi/session", "pi/bind", "unknown"]) {
        let nextCalls = 0;
        const response = await apiMiddleware({ request: new Request(`https://dashboard.invalid/api/${path}`, { method: "POST" }), waitUntil() {}, async next() { nextCalls += 1; return Response.json({}); } }, profile);
        assert.equal(response.status, 403); assert.equal(nextCalls, 0);
      }
    }
  });
  it("selects the observer before bundle entry resolution and guards stale legacy entry", () => {
    const vite = readFileSync(new URL("../vite.config.ts", import.meta.url), "utf8");
    const main = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");
    const middleware = readFileSync(new URL("../functions/api/_middleware.ts", import.meta.url), "utf8");
    assert.match(vite, /order: "pre"/);
    assert.match(vite, /proxy: legacy \?/);
    assert.match(vite, /configurePreviewServer/);
    assert.ok(main.indexOf('if (NETWORK_PROFILE.mode !== "legacy") throw') < main.indexOf('const byId'));
    assert.match(middleware, /profile.mode !== "legacy"/);
    assert.match(middleware, /Only observer query routes are enabled/);
  });
});
