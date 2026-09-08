import assert from "node:assert/strict";
import test from "node:test";
import {createHash} from "node:crypto";
import {readFileSync} from "node:fs";
import {buildFiniteKnowledgeSnapshot, type SnapshotPublication} from "../functions/api/_knowledge_snapshot";
import {KNOWLEDGE_SCHEMA,KNOWLEDGE_FACTS_QUERY_PATH,knowledgeMainnet,type KnowledgeGeometrySnapshot} from "../functions/api/_knowledge";

function projection(): KnowledgeGeometrySnapshot{return {
 schema:KNOWLEDGE_SCHEMA,
 source:{chainId:"zerone-1",blockHeight:"1000",statusHeight:"1000",catchingUp:false,queryPath:KNOWLEDGE_FACTS_QUERY_PATH,queryTracked:false,writes:false,completeness:"NOT_CLAIMED",upstreamRecords:1,returnedRecords:1,truncated:false},
 facts:[{id:"root",content:"A bounded record",domain:"general",category:"",status:"FACT_STATUS_VERIFIED",claimType:"CLAIM_TYPE_ASSERTION",confidence:0,verifiedAtBlock:"900",lastVerifiedBlock:"900",energy:0,energyCap:0,fitnessScore:0,methodId:""}],relations:[],
};}
const hash=(text:string)=>createHash("sha256").update(text).digest("hex");

test("immutable snapshot digest commits payload metadata separately from topology",async()=>{
 const input=projection();const first=await buildFiniteKnowledgeSnapshot(input,"2026-09-08T00:00:00.000Z");
 const a=JSON.parse(first.body);
 assert.equal(first.id,hash(first.body));assert.equal(a.payloadDigest,hash(JSON.stringify(a.payload)));
 assert.equal(a.coverage.history,"NOT_INCLUDED");assert.equal(a.freshness.liveStatus,"UNKNOWN");
 input.facts[0]!.content="Corrected content; same topology";
 const second=await buildFiniteKnowledgeSnapshot(input,"2026-09-08T00:00:00.000Z");const b=JSON.parse(second.body);
 assert.equal(a.topologyRoot,b.topologyRoot);assert.notEqual(a.payloadDigest,b.payloadDigest);assert.notEqual(first.id,second.id);
 input.relations=[{sourceFactId:"root",targetFactId:"external",relation:"RELATION_TYPE_CITES",inference:"INFERENCE_TYPE_CITATION",inferenceStrengthBps:0,createdAtBlock:"900",methodId:""}];
 const c=JSON.parse((await buildFiniteKnowledgeSnapshot(input,"2026-09-08T00:00:00.000Z")).body);
 assert.notEqual(b.topologyRoot,c.topologyRoot);
 input.relations[0]!.inferenceStrengthBps=1;
 const d=JSON.parse((await buildFiniteKnowledgeSnapshot(input,"2026-09-08T00:00:00.000Z")).body);
 assert.equal(c.topologyRoot,d.topologyRoot);assert.notEqual(c.payloadDigest,d.payloadDigest);
});

test("snapshot unknown fields and invalid observation times are refused",async()=>{
 await assert.rejects(buildFiniteKnowledgeSnapshot(projection(),"yesterday"));
 const input=projection();input.source.blockHeight="0";
 await assert.rejects(buildFiniteKnowledgeSnapshot(input,"2026-09-08T00:00:00.000Z"));
});

test("Go and TypeScript share exact immutable publication bytes",async()=>{
 const body=readFileSync(new URL("../../deploy/knowledge-read-facade/testdata/typescript-publication.json",import.meta.url),"utf8");
 const input=JSON.parse(body) as {payload:KnowledgeGeometrySnapshot; publication:SnapshotPublication; observedAt:string};
 const rebuilt=await buildFiniteKnowledgeSnapshot(input.payload,input.observedAt,input.publication);
 assert.equal(rebuilt.body,body);
 assert.equal(rebuilt.id,"bbd7749824ac776397ed1ca0fbc5c12287fef25069da14b2c346dec05b08bb1c");
 for(const change of [{blockHeight:"999"},{projectionSha256:"0".repeat(64)},{sourceCommit:"main"},{restOrigin:"http://127.0.0.1:1/?url=bad"},{blockTime:"2025-01-01T00:00:00.000Z"},{unexpected:true}]){
  await assert.rejects(buildFiniteKnowledgeSnapshot(input.payload,input.observedAt,{...input.publication,...change}));
 }
});

test("public dashboard fails closed without immutable publication binding",async()=>{
 const response=await knowledgeMainnet({request:new Request("https://dashboard.invalid/api/knowledge"),waitUntil:()=>{throw new Error("unexpected publication")}});
 assert.equal(response.status,503);assert.equal(response.headers.get("Cache-Control"),"no-store");
});
