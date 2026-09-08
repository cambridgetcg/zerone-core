import assert from "node:assert/strict";
import test from "node:test";
import {execFileSync} from "node:child_process";
import {fileURLToPath} from "node:url";
import {knowledgeRequest, type KnowledgeGeometrySnapshot} from "../functions/api/_knowledge";
import {buildFiniteKnowledgeSnapshot} from "../functions/api/_knowledge_snapshot";

// Not a fixture-only parser test: construct canonical keeper state, invoke the
// actual Facts query, then consume its generated protobuf JSON in the adapter.
test("Facts query canonical relation storage reaches dashboard and finite snapshot",async()=>{
 const root=fileURLToPath(new URL("../../",import.meta.url));
 const go=process.env.TOK_TEST_GO ?? "go";
 const output=execFileSync(go,["-C",root,"test","-mod=readonly","-p","2","./x/knowledge/keeper","-run","^TestReadCanonicalProjectionWire$","-count=1","-v"],{encoding:"utf8",timeout:120000,maxBuffer:1024*1024,env:{...process.env,GOTOOLCHAIN:"local",GOPROXY:"off",GOSUMDB:"off",GOTELEMETRY:"off",TOK_READ_WIRE:"1"}});
 const wire=output.match(/^TOK_READ_WIRE=([A-Za-z0-9+/=]+)$/m)?.[1];
 assert.ok(wire,"actual Go query did not produce its wire result");
 const raw=Buffer.from(wire,"base64").toString("utf8");
 let reads=0;
 const response=await knowledgeRequest({request:new Request("https://dashboard.invalid/api/knowledge"),waitUntil:()=>{}},{
  cache:{match:async()=>undefined,put:async()=>{}},upstreams:{rest:"https://local-test.invalid",rpc:"https://local-test.invalid"},
  fetch:async input=>{
   reads++;
   const target=new URL(String(input));
   if(target.pathname==="/status")return new Response(JSON.stringify({result:{node_info:{network:"zerone-1"},sync_info:{latest_block_height:"1000",catching_up:false}}}),{headers:{"Content-Type":"application/json"}});
   assert.equal(target.search,"?pagination.limit=100");
   return new Response(raw,{headers:{"Content-Type":"application/json","X-Cosmos-Block-Height":"1000"}});
  },
 });
 assert.equal(response.status,200,await response.clone().text());
 const snapshot=await response.json() as KnowledgeGeometrySnapshot;
 assert.equal(reads,2);assert.ok(snapshot.facts.some(f=>f.id==="fact-a"));assert.ok(snapshot.facts.some(f=>f.id==="fact-b"));
 assert.deepEqual(snapshot.relations.find(r=>r.sourceFactId==="fact-b"&&r.targetFactId==="fact-a"),{sourceFactId:"fact-b",targetFactId:"fact-a",relation:"RELATION_TYPE_SUPPORTS",inference:"INFERENCE_TYPE_EMPIRICAL",inferenceStrengthBps:750000,createdAtBlock:"900",methodId:"M-EMPIRICAL"});
 const published=await buildFiniteKnowledgeSnapshot(snapshot,"2026-09-08T00:00:00.000Z");
 const envelope=JSON.parse(published.body);
 assert.equal(envelope.topologyRoot,output.match(/^TOK_PROJECTION_ROOT=([0-9a-f]{64})$/m)?.[1]);
 assert.equal(envelope.chainId,"zerone-1");assert.equal(envelope.blockHeight,"1000");
 assert.equal(envelope.coverage.wholeGraph,"NOT_CLAIMED");assert.equal(envelope.freshness.liveStatus,"UNKNOWN");
});
