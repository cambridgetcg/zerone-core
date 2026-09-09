import { defineConfig, type Plugin } from "vite";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import type { IncomingMessage, ServerResponse } from "node:http";
import { NETWORK_PROFILE } from "./network-profile";
import { observerPage } from "./observer-page";
import { loadNodeGuideProfile } from "./node-guide-build";
import { nodeGuidePage } from "./node-guide-page";
import { proxyRequest } from "./functions/api/_proxy";

const MAINNET_RPC = "http://169.155.55.44:26657";
const MAINNET_REST = "http://169.155.55.44:1317";
const DASHBOARD_ROOT = fileURLToPath(new URL(".", import.meta.url));
const legacy = NETWORK_PROFILE.mode === "legacy";

// Static documentation names its local and public networks explicitly. It
// neither selects a dashboard profile nor adds a gateway or wallet control.
function nodeGuidePlugin(): Plugin {
  let profile: ReturnType<typeof loadNodeGuideProfile> | undefined;
  const guide = () => profile ??= loadNodeGuideProfile(resolve(DASHBOARD_ROOT, ".."));
  const json = () => `${JSON.stringify(guide(), null, 2)}\n`;
  return {
    name: "source-pinned-node-guide",
    transformIndexHtml: {
      order: "pre",
      handler: (html, context) => context.filename === resolve(DASHBOARD_ROOT, "nodes/index.html")
        ? nodeGuidePage(guide()) : html,
    },
    generateBundle() {
      this.emitFile({ type: "asset", fileName: "nodes/guide.json", source: json() });
    },
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        if (request.url?.split("?")[0] !== "/nodes/guide.json") { next(); return; }
        if (request.method !== "GET" && request.method !== "HEAD") {
          response.writeHead(405, { Allow: "GET, HEAD" }); response.end(); return;
        }
        try {
          const body = json();
          response.writeHead(200, { "Content-Type": "application/json; charset=utf-8", "Cache-Control": "no-cache" });
          response.end(request.method === "HEAD" ? undefined : body);
        } catch (error) {
          response.writeHead(503, { "Content-Type": "text/plain; charset=utf-8" });
          response.end(error instanceof Error ? error.message : "Node guide unavailable");
        }
      });
    },
  };
}

// Beta development and local build previews use the same read-only edge, not
// Vite's unrestricted legacy relay. No production server is started here.
function observerPlugin(): Plugin {
  const cache = new Map<string, { expires: number; response: Response }>();
  const middleware = async (request: IncomingMessage, response: ServerResponse, next: () => void): Promise<void> => {
    if (legacy || !request.url?.startsWith("/api/")) { next(); return; }
    try {
      const url = new URL(request.url, "http://localhost");
      const match = /^\/api\/(rpc|rest)(?:\/(.*))?$/.exec(url.pathname);
      const result = match ? await proxyRequest({
        request: new Request(url, { method: request.method }), params: { path: match[2] }, waitUntil: (promise) => { void promise; },
      }, match[1] as "rpc" | "rest", {
        fetch: globalThis.fetch.bind(globalThis), upstreams: { rpc: "", rest: "" }, profile: NETWORK_PROFILE,
        cache: {
          async match(key) { const hit = cache.get(key.url); return hit && hit.expires > Date.now() ? hit.response.clone() : undefined; },
          async put(key, value) {
            if (cache.size >= 32) cache.delete(cache.keys().next().value!);
            cache.set(key.url, { expires: Date.now() + 3000, response: value.clone() });
          },
        },
      }) : Response.json({ error: "Only observer query routes are enabled" }, { status: 403 });
      response.writeHead(result.status, Object.fromEntries(result.headers));
      response.end(Buffer.from(await result.arrayBuffer()));
    } catch { response.writeHead(502); response.end("Observer unavailable"); }
  };
  return {
    name: "release-bound-observer",
    transformIndexHtml: { order: "pre", handler: (html, context) => !legacy && (context.path === "/" || context.path === "/index.html" || context.filename === resolve(DASHBOARD_ROOT, "index.html")) ? observerPage(NETWORK_PROFILE) : html },
    configureServer(server) { server.middlewares.use((...args) => { void middleware(...args); }); },
    configurePreviewServer(server) { server.middlewares.use((...args) => { void middleware(...args); }); },
  };
}

export default defineConfig({
  appType: "mpa",
  plugins: [observerPlugin(), nodeGuidePlugin()],
  resolve: { preserveSymlinks: true },
  server: {
    host: "127.0.0.1", port: 4173,
    proxy: legacy ? {
      "/api/rpc": { target: MAINNET_RPC, changeOrigin: true, rewrite: (path) => path.replace(/^\/api\/rpc/, "") || "/" },
      "/api/rest": { target: MAINNET_REST, changeOrigin: true, rewrite: (path) => path.replace(/^\/api\/rest/, "") || "/" },
    } : {},
  },
  build: {
    target: "es2022", sourcemap: false,
    rollupOptions: { input: legacy ? {
      dashboard: resolve(DASHBOARD_ROOT, "index.html"), piCallback: resolve(DASHBOARD_ROOT, "pi/callback/index.html"),
      nodes: resolve(DASHBOARD_ROOT, "nodes/index.html"),
    } : { dashboard: resolve(DASHBOARD_ROOT, "index.html"), nodes: resolve(DASHBOARD_ROOT, "nodes/index.html") } },
  },
});
