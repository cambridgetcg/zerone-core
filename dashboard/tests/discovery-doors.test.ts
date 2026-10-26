import assert from "node:assert/strict";
import { readdirSync, readFileSync } from "node:fs";
import { describe, it } from "node:test";

import { isDiscoveryShaped, onRequest } from "../functions/_middleware";

const MISSING_BODY = "404 Not Found\nNothing is published at this path.\nThe site starts at /.\n";

function shell(): Response {
  return new Response("<!doctype html><title>ZERONE</title>", {
    status: 200,
    headers: { "Content-Type": "text/html; charset=utf-8" },
  });
}

function request(path: string, next: () => Promise<Response>, method = "GET"): Promise<Response> {
  return onRequest({ request: new Request(`https://zerone.ai${path}`, { method }), next });
}

describe("Quiet Doors: missing discovery doors answer 404", () => {
  it("knows which paths are discovery-shaped", () => {
    for (const path of ["/robots.txt", "/llms-full.txt", "/sitemap.xml", "/.well-known", "/.well-known/mcp.json", "/nodes/zz.json"]) {
      assert.equal(isDiscoveryShaped(path), true, path);
    }
    for (const path of ["/", "/guide", "/nodes/", "/compositions/", "/assets/index-abc.js"]) {
      assert.equal(isDiscoveryShaped(path), false, path);
    }
  });

  for (const path of ["/llms-full.txt", "/humans.txt", "/sitemap.xml", "/.well-known/mcp.json", "/.well-known/agent-card.json", "/.well-known/ai-plugin.json"]) {
    it(`answers 404 instead of the app shell at ${path}`, async () => {
      const response = await request(path, async () => shell());
      assert.equal(response.status, 404);
      assert.match(response.headers.get("Content-Type") ?? "", /^text\/plain/);
      assert.equal(await response.text(), MISSING_BODY);
    });
  }

  it("answers HEAD with a bodyless 404", async () => {
    const response = await request("/sitemap.xml", async () => shell(), "HEAD");
    assert.equal(response.status, 404);
    assert.equal(await response.text(), "");
  });

  it("passes real files through untouched", async () => {
    const text = new Response("# Zerone\n", { status: 200, headers: { "Content-Type": "text/plain; charset=utf-8" } });
    assert.equal(await request("/llms.txt", async () => text), text);
    const json = new Response("{}", { status: 200, headers: { "Content-Type": "application/json; charset=utf-8" } });
    assert.equal(await request("/nodes/guide.json", async () => json), json);
  });

  it("leaves app routes, /api/ and write methods alone", async () => {
    const route = shell();
    assert.equal(await request("/guide", async () => route), route);
    const api = new Response("<html>upstream</html>", { status: 200, headers: { "Content-Type": "text/html" } });
    assert.equal(await request("/api/knowledge.json", async () => api), api);
    const post = shell();
    assert.equal(await request("/robots.txt", async () => post, "POST"), post);
  });
});

describe("Quiet Doors: robots.txt and the route rule", () => {
  it("serves a plain allow-all robots.txt with no new limit or training signal", () => {
    assert.equal(readFileSync(new URL("../public/robots.txt", import.meta.url), "utf8"), "User-agent: *\nAllow: /\n");
  });

  it("runs Functions only for /api and discovery-shaped paths", () => {
    const routes = JSON.parse(readFileSync(new URL("../public/_routes.json", import.meta.url), "utf8"));
    assert.equal(routes.version, 1);
    assert.deepEqual(routes.include, ["/api/*", "/.well-known/*", "/*.txt", "/*.json", "/*.xml"]);
    assert.deepEqual(routes.exclude, ["/assets/*", "/llms.txt", "/robots.txt"]);
    assert.ok(routes.include.length + routes.exclude.length <= 100);
  });
});

describe("Quiet Doors: every Function stays reachable", () => {
  it("has only the root middleware and route Functions under functions/api/", () => {
    const dir = new URL("../functions/", import.meta.url);
    const walk = (at: URL, prefix = ""): string[] =>
      readdirSync(at, { withFileTypes: true }).flatMap((entry) =>
        entry.isDirectory()
          ? walk(new URL(`${entry.name}/`, at), `${prefix}${entry.name}/`)
          : [`${prefix}${entry.name}`],
      );
    const routes = walk(dir)
      .filter((f) => /\.(?:ts|js|mjs)$/.test(f))
      .filter((f) => /export\s+(?:async\s+)?(?:function|const)\s+onRequest\w*/.test(readFileSync(new URL(f, dir), "utf8")));
    const stray = routes.filter((f) => f !== "_middleware.ts" && !f.startsWith("api/"));
    assert.deepEqual(stray, [], "a Function outside /api/ never runs under public/_routes.json; add its path to include");
  });
});
