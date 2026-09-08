// Explicit, loopback-only SYNTHETIC fixture for manual observer browser checks.
// Does not contact a chain, satisfy gateway readiness, or represent production.
import { createServer } from "node:http";
import { syntheticProfile, syntheticResponse, APP_HASH, BLOCK_HASH } from "../tests/observer-fixtures";
import { validObserverQuery } from "../functions/api/_proxy";

const scenario = process.argv.find((arg) => arg.startsWith("--scenario="))?.split("=")[1] ?? "healthy";
if (!["healthy", "stale", "future", "wrong-chain", "archive", "unavailable", "missing-tx", "unknown-supply"].includes(scenario)) throw new Error("Unknown synthetic scenario");
const archive = scenario === "archive";
const profile = syntheticProfile();
if (archive) profile.checkpoint = { height: 10, blockHash: BLOCK_HASH, appHash: APP_HASH };
if (process.argv.includes("--profile")) {
  process.stdout.write(`${JSON.stringify(profile, null, 2)}\n`);
} else {
  const started = Date.now() - 1000;
  const blockHash = (height: number): string => height === 10 ? BLOCK_HASH : height.toString(16).padStart(64, "0");
  const server = createServer((request, response) => {
    const url = new URL(request.url ?? "/", "http://127.0.0.1:4180");
    const kind = /^\/(cosmos|zerone)\//.test(url.pathname) ? "rest" : "rpc";
    response.setHeader("Content-Type", "application/json");
    response.setHeader("Cache-Control", "no-store");
    if (!["GET", "HEAD"].includes(request.method ?? "") || !validObserverQuery(kind, url.pathname.slice(1), url.search)) {
      response.writeHead(403); response.end('{"error":"Synthetic read-only allowlist"}'); return;
    }
    if (scenario === "unavailable" || (scenario === "unknown-supply" && url.pathname.includes("supply"))) {
      response.writeHead(503); response.end('{"error":"Synthetic unavailable"}'); return;
    }
    if (scenario === "missing-tx" && url.pathname === "/tx") {
      response.writeHead(404); response.end('{"error":"Synthetic missing or unavailable tx"}'); return;
    }
    const latest = archive ? 10 : 10 + Math.floor((Date.now() - started) / 30_000);
    const wanted = url.pathname === "/block" ? Number(url.searchParams.get("height")) : latest;
    if (wanted > latest) { response.writeHead(404); response.end('{"error":"Synthetic height not found"}'); return; }
    const shift = scenario === "stale" ? -300_000 : scenario === "future" ? 300_000 : archive ? -86_400_000 : 0;
    const timestamp = new Date(started + (wanted - 10) * 30_000 + shift).toISOString();
    const body = syntheticResponse(`${url.pathname}${url.search}`, started + 1000, archive);
    if (url.pathname === "/status") {
      const result = body.result as { node_info: { network: string }; sync_info: Record<string, unknown> };
      Object.assign(result.sync_info, { latest_block_height: String(latest), latest_block_time: timestamp, latest_block_hash: blockHash(latest) });
      if (scenario === "wrong-chain") result.node_info.network = "not-the-configured-chain";
    }
    if (url.pathname === "/block") {
      const result = body.result as { block_id: { hash: string }; block: { header: { time: string } } };
      result.block_id.hash = blockHash(wanted); result.block.header.time = timestamp;
    }
    response.end(request.method === "HEAD" ? undefined : JSON.stringify(body));
  });
  server.listen(4180, "127.0.0.1", () => {
    process.stdout.write(`SYNTHETIC ONLY: http://127.0.0.1:4180 (${scenario}); Ctrl-C stops. No upstream network or production authority.\n`);
  });
}
