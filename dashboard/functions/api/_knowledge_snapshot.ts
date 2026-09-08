// Finite publication product. This module performs no network or filesystem
// writes; an authorized, source-matched publisher supplies the validated page.
import { cachedProjection, KNOWLEDGE_OUTPUT_MAX_BYTES, type KnowledgeGeometrySnapshot } from "./_knowledge";

const encoder = new TextEncoder();
export const SNAPSHOT_SCHEMA = "zerone.knowledge-public-snapshot/v1" as const;
export const TOPOLOGY_SCOPE = "TOK_ROOT/v1 over projected node IDs and all returned canonical relation/inference edges; not a ToK selector root; excludes content, method, strength, time, history and source metadata" as const;

async function sha256(bytes: Uint8Array): Promise<Uint8Array> {
  return new Uint8Array(await crypto.subtle.digest("SHA-256", bytes as BufferSource));
}
function hex(bytes: Uint8Array): string { return [...bytes].map(b=>b.toString(16).padStart(2,"0")).join(""); }
function concatenate(parts: Uint8Array[]): Uint8Array {
  const out = new Uint8Array(parts.reduce((sum,p)=>sum+p.length,0));
  let offset=0;
  for(const part of parts){out.set(part,offset);offset+=part.length;}
  return out;
}
function field(text: string): Uint8Array {
  const value=encoder.encode(text), length=new Uint8Array(8);
  new DataView(length.buffer).setBigUint64(0,BigInt(value.length),false);
  return concatenate([length,value]);
}

// Same length-prefix and domain tags as keeper.ComputeToKSnapshotRoot. The
// scope is intentionally NOT the support-only selector's edge set.
export async function projectionTopologyRoot(snapshot: KnowledgeGeometrySnapshot): Promise<string> {
  const nodes=[...snapshot.facts].map(f=>f.id).sort();
  const compare=(a:string,b:string)=>a<b?-1:a>b?1:0;
  const edges=[...snapshot.relations].sort((a,b)=>compare(a.sourceFactId,b.sourceFactId)||compare(a.targetFactId,b.targetFactId)||compare(a.relation,b.relation)||compare(a.inference,b.inference));
  const nodeHash=await sha256(concatenate([field("TOK_NODES"),...nodes.map(field)]));
  const edgeHash=await sha256(concatenate([field("TOK_EDGES"),...edges.flatMap(e=>[e.sourceFactId,e.targetFactId,e.relation,e.inference].map(field))]));
  return hex(await sha256(concatenate([field("TOK_ROOT"),nodeHash,edgeHash])));
}

export interface SnapshotPublication {
  schema: "zerone.knowledge-publication-metadata/v1";
  chainId: string;
  blockHeight: string;
  statusHeight: string;
  blockTime: string;
  observedAt: string;
  restOrigin: string;
  rpcOrigin: string;
  sourceCommit: string;
  projectionSha256: string;
  provenance: "operator-asserted actual-query projection; not chain, release, custody or authority proof";
}

// With no publication metadata this remains an inspectable draft, NOT input the
// concrete serving tool will admit. CLI publication additionally enforces age,
// fixed operator origins, immutable storage and exact content-addressed heads.
export async function buildFiniteKnowledgeSnapshot(projection: KnowledgeGeometrySnapshot, observedAt: string, publication?: SnapshotPublication): Promise<{ id: string; body: string }> {
  // Capture a detached normalized value before any asynchronous digest work.
  const payload=JSON.stringify(projection);
  const snapshot=cachedProjection(payload);
  if (!snapshot) throw new Error("Invalid finite geometry projection");
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/.test(observedAt) || new Date(observedAt).toISOString()!==observedAt) throw new Error("Invalid observation time");
  const metadata = publication === undefined ? undefined : structuredClone(publication);
  const payloadDigest = hex(await sha256(encoder.encode(payload)));
  if (metadata !== undefined) {
    const keys = ["schema", "chainId", "blockHeight", "statusHeight", "blockTime", "observedAt", "restOrigin", "rpcOrigin", "sourceCommit", "projectionSha256", "provenance"];
    if (Object.keys(metadata).length !== keys.length || keys.some(key => !Object.hasOwn(metadata, key)) ||
      metadata.schema !== "zerone.knowledge-publication-metadata/v1" ||
      metadata.chainId !== snapshot.source.chainId || metadata.blockHeight !== snapshot.source.blockHeight ||
      metadata.statusHeight !== snapshot.source.statusHeight || snapshot.source.catchingUp ||
      metadata.observedAt !== observedAt || metadata.projectionSha256 !== payloadDigest ||
      !/^[a-f0-9]{40}$/.test(metadata.sourceCommit) ||
      metadata.provenance !== "operator-asserted actual-query projection; not chain, release, custody or authority proof" ||
      !Number.isFinite(Date.parse(metadata.blockTime)) ||
      Date.parse(observedAt) - Date.parse(metadata.blockTime) > 30_000 ||
      Date.parse(metadata.blockTime) - Date.parse(observedAt) > 10_000 ||
      encoder.encode(JSON.stringify(metadata)).length > 4096) throw new Error("Invalid publication metadata");
    for (const origin of [metadata.restOrigin, metadata.rpcOrigin]) {
      const url = new URL(origin);
      if (!["http:", "https:"].includes(url.protocol) || url.origin !== origin || url.username || url.password || url.search || url.hash) throw new Error("Invalid fixed publication origin");
    }
  }
  const body=JSON.stringify({
    schema:SNAPSHOT_SCHEMA,
    chainId:snapshot.source.chainId,
    blockHeight:snapshot.source.blockHeight,
    observedAt,
    freshness:{kind:"observation-only",statusHeight:snapshot.source.statusHeight,catchingUp:snapshot.source.catchingUp,liveStatus:"UNKNOWN"},
    coverage:{kind:"bounded-page",wholeGraph:"NOT_CLAIMED",canonicalRelations:"source-asserted",history:"NOT_INCLUDED",truncated:snapshot.source.truncated,nodes:snapshot.facts.length,edges:snapshot.relations.length},
    topologyRoot:await projectionTopologyRoot(snapshot),
    topologyRootScope:TOPOLOGY_SCOPE,
    payloadDigest,
    payloadDigestScope:"SHA-256 of exact embedded UTF-8 compact JSON payload bytes, including projected metadata; not a chain proof",
    ...(metadata === undefined ? {} : { publication: metadata }),
    payload:snapshot,
  });
  if(encoder.encode(body).length>KNOWLEDGE_OUTPUT_MAX_BYTES) throw new Error("Immutable snapshot exceeds 256 KiB");
  return {id:hex(await sha256(encoder.encode(body))),body};
}
