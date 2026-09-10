import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { computeReviewCommitmentV2, makeReviewRevealV2, type ReviewCommitmentV2 } from "../src/review-commitment";
import { MsgSubmitReveal, MsgSubmitClaim, MsgChallengeProvisionalFact } from "../src/generated/zerone/knowledge/v1/tx";
import { Claim } from "../src/generated/zerone/knowledge/v1/types";
const fixture = JSON.parse(readFileSync(new URL("./fixtures/review-commitment-v2.json", import.meta.url), "utf8"));
function review(): ReviewCommitmentV2 { return { chainId:fixture.chainId,roundId:fixture.roundId,verifier:fixture.verifier,vote:fixture.vote,confidence:BigInt(fixture.confidence),salt:Uint8Array.from(Buffer.from(fixture.saltHex,"hex")),attestation:structuredClone(fixture.attestation) }; }
test("scheme-2 independently encoded fixture agrees with Go and binds every signed field",()=>{
 const original=review();const expected=fixture.sha256;
 assert.equal(Buffer.from(computeReviewCommitmentV2(original)).toString("hex"),expected);
 const changes:((r:ReviewCommitmentV2)=>void)[]=[r=>r.chainId+="-other",r=>r.roundId+="-other",r=>r.vote="reject",r=>r.confidence--,r=>r.salt[0]=r.salt[0]!+1,r=>r.attestation.methodId+="x",r=>r.attestation.reason+=" ",r=>r.attestation.scope+=" ",r=>r.attestation.evidenceIds[0]+="x",r=>r.attestation.evidenceIds.reverse()];
 for(const change of changes){const r=review();change(r);assert.notEqual(Buffer.from(computeReviewCommitmentV2(r)).toString("hex"),expected);}
});
test("scheme-2 rejects malformed or unbound attestation fields",()=>{
 const changes:((r:ReviewCommitmentV2)=>void)[]=[r=>r.confidence=1000001n,r=>r.salt=new Uint8Array(15),r=>r.attestation.reason="\u0085 \t",r=>r.attestation.reason="x".repeat(4097),r=>r.attestation.reason="\ud800",r=>r.attestation.evidenceIds=["a","a"],r=>r.attestation.evidenceIds=Array(17).fill("a"),r=>Object.assign(r.attestation,{unbound:"x"}),r=>r.verifier=r.verifier.toUpperCase()];
 for(const change of changes){const r=review();change(r);assert.throws(()=>computeReviewCommitmentV2(r));}
});
test("reasoned reveal and signed claim/challenge fields survive protobuf codecs exactly",()=>{
 const r=review();const msg=makeReviewRevealV2(r);const restored=MsgSubmitReveal.decode(MsgSubmitReveal.encode(msg).finish());assert.deepEqual(restored,msg);
 r.salt[0]=r.salt[0]!+1;r.attestation.evidenceIds[0]="mutated";assert.notDeepEqual(r.salt,msg.salt);assert.notDeepEqual(r.attestation.evidenceIds,msg.attestation!.evidenceIds);
 const claim=MsgSubmitClaim.fromPartial({methodId:"M-COMPUTATIONAL",reasoningTrace:"Exact reason α",relations:[{methodId:"M-FORMAL"}]});assert.deepEqual(MsgSubmitClaim.decode(MsgSubmitClaim.encode(claim).finish()),claim);
 const challenge=MsgChallengeProvisionalFact.fromPartial({claimId:"original",reason:"Counterexample",evidenceIds:["evidence"],counterClaim:"Counter claim",methodId:"M-COMPUTATIONAL"});assert.deepEqual(MsgChallengeProvisionalFact.decode(MsgChallengeProvisionalFact.encode(challenge).finish()),challenge);
 const stored=Claim.fromPartial({challengedClaimId:"original",evidenceIds:["evidence"],counterClaim:"Counter claim",argumentText:"Counterexample",methodId:"M-COMPUTATIONAL"});assert.deepEqual(Claim.decode(Claim.encode(stored).finish()),stored);
});
