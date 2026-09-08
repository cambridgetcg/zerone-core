// Deterministic local test-vector generator, not a publisher or network observer.
// Run with the repository's lockfile-pinned dashboard tsx. No network requests.
import { writeFileSync, mkdirSync } from "node:fs";
import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";
import { buildFiniteKnowledgeSnapshot, type SnapshotPublication } from "../../dashboard/functions/api/_knowledge_snapshot";
import { type KnowledgeGeometrySnapshot } from "../../dashboard/functions/api/_knowledge";

async function main(): Promise<void> {
  const payload: KnowledgeGeometrySnapshot = {
    schema: "zerone.knowledge-geometry-snapshot/v0",
    source: { chainId: "zerone-1", blockHeight: "1000", statusHeight: "1000", catchingUp: false, queryPath: "/zerone/knowledge/v1/facts?pagination.limit=100", queryTracked: false, writes: false, completeness: "NOT_CLAIMED", upstreamRecords: 1, returnedRecords: 1, truncated: false },
    facts: [{ id: "fact-a", content: "<A bounded record> — synthetic fixture", domain: "general", category: "", status: "FACT_STATUS_VERIFIED", claimType: "CLAIM_TYPE_ASSERTION", confidence: 0, verifiedAtBlock: "900", lastVerifiedBlock: "900", energy: 0, energyCap: 0, fitnessScore: 0, methodId: "" }],
    relations: [{ sourceFactId: "fact-a", targetFactId: "external", relation: "RELATION_TYPE_CITES", inference: "INFERENCE_TYPE_CITATION", inferenceStrengthBps: 0, createdAtBlock: "900", methodId: "" }],
  };
  const publication: SnapshotPublication = {
    schema: "zerone.knowledge-publication-metadata/v1", chainId: "zerone-1", blockHeight: "1000", statusHeight: "1000",
    blockTime: "2026-09-08T00:00:00.000Z", observedAt: "2026-09-08T00:00:00.000Z", restOrigin: "http://127.0.0.1:1317", rpcOrigin: "http://127.0.0.1:26657", sourceCommit: "a".repeat(40),
    projectionSha256: createHash("sha256").update(JSON.stringify(payload)).digest("hex"),
    provenance: "operator-asserted actual-query projection; not chain, release, custody or authority proof",
  };
  const snapshot = await buildFiniteKnowledgeSnapshot(payload, publication.observedAt, publication);
  const directory = new URL("./testdata/", import.meta.url);
  mkdirSync(directory, { recursive: true });
  writeFileSync(new URL("typescript-publication.json", directory), snapshot.body, { flag: "w" });
  console.log(`${snapshot.id} ${fileURLToPath(new URL("typescript-publication.json", directory))}`);
}
void main();
