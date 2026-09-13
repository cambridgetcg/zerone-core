#!/usr/bin/env python3
"""Read-only loopback viewer for the separately validated local claim workflow."""

import base64
import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import re
import threading


MAX_RESPONSE_BYTES = 8 * 1024 * 1024
MAX_RECORDS = 100
_PENDING_FIELDS = {"claim_id", "round_id", "actor", "status", "commit_deadline", "reveal_deadline"}
_SNAPSHOT_FIELDS = {"schema", "chain_id", "height", "local_only", "notice", "claims", "transactions", "pending_reviews"}
_TRANSACTION_FIELDS = {"id", "action", "actor", "status", "txhash", "height", "code", "gas_wanted", "gas_used",
                       "claim_id", "round_id", "tx_bytes", "review_fee_uzrn", "collateral_uzrn"}

STYLE = """
:root{color-scheme:light;--ink:#172823;--muted:#566961;--line:#cddbd2;--paper:#f5f7f0;--accent:#23674d}
*{box-sizing:border-box}body{margin:0;background:var(--paper);color:var(--ink);font:16px/1.6 system-ui,sans-serif}
header,main,footer{max-width:1120px;margin:auto;padding:28px}header{padding-top:44px}h1{font-size:clamp(2rem,5vw,3.5rem);line-height:1.1;margin:12px 0}h2{font-size:1.45rem;margin:0 0 14px}h3{font-size:1.12rem;margin:0 0 10px}h4{margin:16px 0 8px}
p{margin:8px 0}a{color:var(--accent)}button{font:inherit;font-weight:650;background:var(--accent);color:white;border:0;border-radius:8px;padding:10px 17px;cursor:pointer}button:disabled{opacity:.6;cursor:wait}:focus-visible{outline:3px solid #b15e21;outline-offset:4px}
.eyebrow{text-transform:uppercase;letter-spacing:.12em;font-size:.76rem;font-weight:700;color:var(--accent)}.notice{border-left:4px solid #b57937;padding:10px 16px;background:#fff7e9}.muted,.empty{color:var(--muted)}.small{font-size:.88rem}.bar{display:flex;gap:18px;align-items:center;flex-wrap:wrap;margin:22px 0 8px}.status{padding:10px 14px;border:1px solid var(--line);border-radius:8px;background:white}.status[data-tone=error]{border-color:#b14f34;color:#802a16;background:#fff0e9}
main{padding-top:0}.section{margin:26px 0 36px;scroll-margin-top:16px}.card{background:white;border:1px solid var(--line);border-radius:12px;padding:22px;margin:14px 0}.subcard{background:#f7faf5;border:1px solid var(--line);border-radius:8px;padding:16px;margin:12px 0}.badge{display:inline-block;padding:2px 9px;border-radius:5px;background:#e5eee4;font-weight:650;font-size:.84rem;margin-bottom:10px}.identifier,code{font:13px/1.6 ui-monospace,monospace;overflow-wrap:anywhere;word-break:break-word}.identifier{display:block;color:var(--muted);margin:4px 0 12px}.content{font-size:1.16rem;white-space:pre-wrap;overflow-wrap:anywhere}.text{white-space:pre-wrap;overflow-wrap:anywhere}dl{display:grid;grid-template-columns:minmax(110px,180px) minmax(0,1fr);gap:5px 16px;margin:12px 0}dt{color:var(--muted);font-size:.88rem}dd{margin:0;overflow-wrap:anywhere;white-space:pre-wrap}ul{padding-left:22px}li{overflow-wrap:anywhere}summary{cursor:pointer;font-weight:650;padding:10px 0}details{margin-top:12px}pre{white-space:pre-wrap;overflow-wrap:anywhere;font-size:12px;max-height:520px;overflow:auto;background:#edf1e8;padding:16px;border-radius:8px}.skip{position:absolute;left:16px;top:-100px}.skip:focus{top:8px;background:white;padding:8px}.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.grid .subcard{margin:0}footer{border-top:1px solid var(--line);color:var(--muted)}[hidden]{display:none!important}
@media(max-width:650px){header,main,footer{padding:20px}.card{padding:16px}.grid{grid-template-columns:1fr}dl{grid-template-columns:1fr;gap:0}dd{margin-bottom:8px}header{padding-top:34px}}
"""

SCRIPT = r"""
'use strict';
const byId=id=>document.getElementById(id);
const list=value=>Array.isArray(value)?value:[];
const value=(input,fallback='Not retained')=>input===undefined||input===null||input===''?fallback:String(input);
function el(tag,text,className){const node=document.createElement(tag);if(text!==undefined)node.textContent=text;if(className)node.className=className;return node;}
const ENUMS={
  claim:['CLAIM_STATUS_',['UNSPECIFIED','PENDING','PENDING_EVALUATION','EVALUATED','PROVISIONAL','IN_VERIFICATION','ACCEPTED','REJECTED','CHALLENGED','EXPIRED','INSUFFICIENT','CONTESTED','MALFORMED']],
  phase:['VERIFICATION_PHASE_',['UNSPECIFIED','COMMIT','REVEAL','AGGREGATION','COMPLETE','EXPIRED']],
  verdict:['VERDICT_',['UNSPECIFIED','ACCEPT','REJECT','INCONCLUSIVE','MALFORMED']],
  fact:['FACT_STATUS_',['UNSPECIFIED','PENDING','PROVISIONAL','VERIFIED','ACTIVE','CONTESTED','CHALLENGED','SUPERSEDED','EXPIRED','DISPROVEN','REVOKED','AT_RISK','PRUNED']],
  relation:['RELATION_TYPE_',['UNSPECIFIED','SUPPORTS','CONTRADICTS','REQUIRES','REFINES','GENERALIZES','SUPERSEDES','CITES','REFORMULATES']],
  inference:['INFERENCE_TYPE_',['UNSPECIFIED','DEDUCTIVE','INDUCTIVE','ABDUCTIVE','EMPIRICAL','ANALOGICAL','CITATION']]
};
function label(kind,input){
  if(input===undefined||input===null||input==='')return 'Not recorded';
  const [prefix,names]=ENUMS[kind],raw=String(input);
  const name=/^(0|[1-9][0-9]*)$/.test(raw)?names[Number(raw)]:raw.startsWith(prefix)?raw.slice(prefix.length):undefined;
  return name!==undefined&&names.includes(name)?name.replaceAll('_',' '):`Unknown ${kind} value (${raw})`;
}
function field(parent,title,text){parent.append(el('dt',title),el('dd',value(text)));}
function paragraph(parent,title,text){parent.append(el('h4',title),el('p',value(text),'text'));}
function items(parent,title,rows,empty){parent.append(el('h4',title));if(!rows.length){parent.append(el('p',empty,'empty small'));return;}const ul=el('ul');for(const row of rows)ul.append(el('li',value(row)));parent.append(ul);}
function details(title){const node=el('details');node.append(el('summary',title));return node;}
function money(amount){if(typeof amount!=='string'||!/^(0|[1-9][0-9]*)$/.test(amount)||amount.length>20||BigInt(amount)>18446744073709551615n)throw new Error('Malformed recorded uzrn amount');return amount+' uzrn';}
function funding(claim){const body=details('Funding terms and allocations'),terms=claim.funding_terms;
  if(!terms){body.append(el('p','No prospective funding terms retained. Historical message-specific rules apply; absence is not a zero balance or refund promise.'));return body;}
  if(terms.policy_version!==1)throw new Error('Unsupported funding policy');
  const kind=({1:'Ordinary review fee',2:'Challenge deposit',CLAIM_FUNDING_KIND_REVIEW_FEE:'Ordinary review fee',CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT:'Challenge deposit'})[terms.kind];if(!kind)throw new Error('Unsupported funding kind');
  body.append(el('p','Funding policy 1 · '+kind));const meta=el('dl');for(const [key,title]of[['paid_amount','Recorded admission amount'],['review_budget','Review budget'],['refundable_amount','Refundable allocation'],['retained_fee','Retained non-refundable fee']])field(meta,title,money(terms[key]));
  if(BigInt(terms.paid_amount)!==BigInt(terms.review_budget)+BigInt(terms.refundable_amount)+BigInt(terms.retained_fee))throw new Error('Funding allocations do not sum to the recorded amount');
  body.append(meta,el('p','Network fees are separate. Allocations are instructions; the round records show whether payment or refund was recorded.','muted small'));return body;
}
function renderRound(round){
  const card=el('section',undefined,'subcard');card.append(el('h4','Review round'),el('code',value(round.id),'identifier'),el('span',label('phase',round.phase),'badge'));
  const meta=el('dl');field(meta,'Verdict',label('verdict',round.verdict));field(meta,'Verdict block',round.verdict_block);field(meta,'Commit deadline',round.commit_deadline);field(meta,'Reveal deadline',round.reveal_deadline);field(meta,'Aggregation deadline',round.aggregation_deadline);field(meta,'Commitment scheme',round.commitment_scheme??0);field(meta,'Review policy',round.review_policy_version??0);card.append(meta);
  card.append(el('p','Deadlines are block heights. A recorded verdict is a review outcome, not a truth certificate.','muted small'));
  const commits=list(round.commits),reveals=list(round.reveals);
  items(card,'Published commitments',commits.map(c=>`${value(c.verifier)} · block ${value(c.committed_at_block)} · hash ${value(c.commit_hash)}`),'No commitments retained.');
  card.append(el('h4',`Published reviews (${reveals.length})`));
  if(!reveals.length)card.append(el('p','No revealed reviews retained. Private review preparations are not displayed.','empty'));
  for(const review of reveals){
    const body=el('div',undefined,'subcard');body.append(el('strong',value(review.vote,'Vote not retained')),el('code',value(review.verifier),'identifier'));
    const meta=el('dl');field(meta,'Revealed at block',review.revealed_at_block);
    field(meta,'Declared confidence',Number(round.commitment_scheme)===2?value(review.confidence, '0')+' / 1000000':'Not retained for a legacy review');body.append(meta);
    const att=review.attestation;paragraph(body,'Reason / work described',att?.reason);paragraph(body,'Scope',att?.scope);paragraph(body,'Method',att?.method_id);items(body,'Evidence references',list(att?.evidence_ids),'No evidence references retained.');card.append(body);
  }
  const plan=round.verifier_reward_settlement;
  if(plan){const payment=details('Reviewer payment record');payment.append(el('p',String(plan.paid_at_block??'0')==='0'?'Recorded obligation awaiting transfer.':'Recorded transferred at block '+plan.paid_at_block+'.'));payment.append(el('p','Obligation created at block '+value(plan.created_at_block)));
    payment.append(el('p','This state observation is not an independent bank or transaction proof. Amounts are uzrn.','muted small'));
    items(payment,'Recorded recipients',list(plan.payments).map(p=>`${value(p.verifier)} · amount ${money(p.amount)} · withheld ${money(p.withheld??'0')}`),'No payments listed.');card.append(payment);}else card.append(el('p','No reviewer settlement record returned; absence alone does not establish that nothing is owed.','muted small'));
  const refund=round.claim_refund_settlement;if(refund){const body=details('Claim refund record');body.append(el('p',String(refund.paid_at_block??'0')==='0'?'Recorded obligation awaiting transfer.':'Recorded transferred at block '+refund.paid_at_block+'.'));const meta=el('dl');field(meta,'Recipient',refund.recipient);field(meta,'Amount',money(refund.amount));field(meta,'Obligation created at block',refund.created_at_block);body.append(meta,el('p','Retained state observation, not an independent bank-transfer proof.','muted small'));card.append(body);}else card.append(el('p','No refund settlement record returned; absence alone does not establish that nothing is owed.','muted small'));
  return card;
}
function renderFact(entry){
  const body=el('div',undefined,'subcard'),fact=entry.fact;
  if(!fact){body.append(el('p','Fact row is missing.','empty'));return body;}
  body.append(el('code',value(fact.id),'identifier'),el('span',label('fact',fact.status),'badge'),el('p',value(fact.content),'text'));
  const meta=el('dl');field(meta,'Submitted by',fact.submitter);field(meta,'Submitted at block',fact.submitted_at_block);field(meta,'Verified at block',fact.verified_at_block);body.append(meta);
  for(const [title,key]of[['Outgoing canonical relations','outgoing_relations'],['Incoming canonical relations','incoming_relations']]){
    items(body,title,list(entry[key]).map(r=>`${value(r.source_fact_id)} → ${value(r.target_fact_id)} · ${label('relation',r.relation)} · ${label('inference',r.inference)} · declared strength ${value(r.inference_strength_bps)} bps · method ${value(r.method_id)} · creator ${value(r.creator)} · block ${value(r.created_at_block)}`),'No canonical relations returned in this direction.');}
  items(body,'Retained status changes',list(entry.status_transitions).map(t=>`Block ${value(t.block_height)} · ${label('fact',t.prior_status)} → ${label('fact',t.new_status)} · ${value(t.cause_event_type)} · ${value(t.cause_id)}`),'No status changes retained; absence does not prove the status never changed.');return body;
}
function renderRecord(record,title){
  const card=el('article',undefined,'card');card.append(el('h3',title),el('code',value(record?.claim_id),'identifier'));
  if(!record){card.append(el('p','Record missing.','empty'));return card;}
  const claim=record.claim;
  if(!claim)card.append(el('p','Claim row missing. Surviving rounds or facts can still be shown below.','notice'));
  else{
    card.append(el('span',label('claim',claim.status),'badge'),el('p',value(claim.fact_content),'content'));
    const meta=el('dl');field(meta,'Author / submitter',claim.submitter);field(meta,'Submitted at block',claim.submitted_at_block);field(meta,'Domain / category',`${value(claim.domain)} / ${value(claim.category)}`);field(meta,'Method',claim.method_id);field(meta,'Review policy',claim.review_policy_version??0);field(meta,'Recorded stake field',claim.stake===undefined?'Not retained':`${claim.stake} uzrn`);card.append(meta);
    card.append(el('p','The stake field has message-specific fee or deposit rules. It is not a payment or refund receipt.','muted small'),funding(claim));
    paragraph(card,'Reasoning',claim.reasoning_trace);paragraph(card,'Challenge / contradiction reason',claim.argument_text);
    if(claim.counter_claim)paragraph(card,'Counterclaim',claim.counter_claim);if(claim.rebuttal_text)paragraph(card,'Retained rebuttal',claim.rebuttal_text);
    items(card,'References',list(claim.references),'No references retained.');items(card,'Challenge evidence',list(claim.evidence_ids),'No challenge evidence retained.');
  }
  if(list(record.missing_round_ids).length)items(card,'Missing referenced rounds',record.missing_round_ids,'');
  card.append(el('h4','Retained review rounds'));
  if(!list(record.rounds).length)card.append(el('p','No review rounds retained.','empty'));
  for(const round of list(record.rounds))card.append(renderRound(round));
  const facts=details(`Derived Facts (${list(record.facts).length})`);if(!list(record.facts).length)facts.append(el('p','No derived Facts returned. A pending or rejected claim can still be a recorded contribution.','empty'));
  for(const fact of list(record.facts))facts.append(renderFact(fact));card.append(facts);return card;
}
function render(snapshot){
  byId('chain').textContent=`${snapshot.chain_id} · observed block ${snapshot.height}`;byId('notice').textContent=snapshot.notice;
  const claims=byId('claims');claims.replaceChildren();
  if(!snapshot.claims.length)claims.append(el('p','No workflow claims yet. Submit a claim with the local workflow CLI, then refresh.','empty'));
  for(const item of snapshot.claims){const history=item.history;const card=renderRecord(history.record,'Claim');
    const related=details(`Directly related claims (${list(history.related_claims).length})`);
    related.append(el('p','These are stored direct links, not inferred causation or an exhaustive graph.','muted small'));
    if(!list(history.related_claims).length)related.append(el('p','No direct related claims returned.','empty'));
    for(const relation of list(history.related_claims)){items(related,'Stored links',list(relation.links).map(l=>`${value(l.field)} → ${value(l.target_id)}`),'No links retained.');related.append(renderRecord(relation.record,'Related claim / challenge'));}
    card.append(related);claims.append(card);}
  const pending=byId('pending');pending.replaceChildren();
  if(!snapshot.pending_reviews.length)pending.append(el('p','No local review preparations pending.','empty'));
  for(const review of snapshot.pending_reviews){const card=el('div',undefined,'subcard');card.append(el('strong',value(review.actor)),el('p',value(review.status)));const meta=el('dl');for(const key of ['claim_id','round_id','commit_deadline','reveal_deadline'])field(meta,key.replaceAll('_',' '),review[key]);card.append(meta);pending.append(card);}
  const transactions=byId('transactions');transactions.replaceChildren();
  if(!snapshot.transactions.length)transactions.append(el('p','No transaction attempts recorded by this workflow.','empty'));
  for(const tx of snapshot.transactions){const card=el('article',undefined,'subcard');const included=/^[1-9][0-9]*$/.test(String(tx.height??''));const code=String(tx.code??'');const state=included?(code==='0'?'Included · execution succeeded':code!==''?'Included · execution failed':'Included · execution result unavailable'):tx.status==='rejected'?'CheckTx refused · no committed inclusion recorded':'No committed inclusion recorded';
    card.append(el('h3',`${value(tx.action,'Transaction')} · ${value(tx.actor,'actor not recorded')}`),el('span',state,'badge'));const meta=el('dl');field(meta,'Workflow receipt status',tx.status);field(meta,'Transaction hash',tx.txhash);field(meta,'Included block',tx.height);field(meta,included?'Execution code':'CheckTx code / result',tx.code);field(meta,'Gas budget / wanted',tx.gas_wanted);field(meta,'Gas used',tx.gas_used);field(meta,'Signed transaction size',tx.tx_bytes===undefined?undefined:`${tx.tx_bytes} bytes`);
    if(tx.review_fee_uzrn!==undefined)field(meta,'Declared review fee',`${tx.review_fee_uzrn} uzrn`);if(tx.collateral_uzrn!==undefined)field(meta,'Declared challenge collateral',`${tx.collateral_uzrn} uzrn`);card.append(meta);transactions.append(card);}
  byId('raw').textContent=JSON.stringify(snapshot,null,2);
}
let refreshing=false,lastSuccess=null;
async function refresh(){
  if(refreshing)return;refreshing=true;byId('refresh').disabled=true;
  try{const response=await fetch('/api/history',{cache:'no-store',redirect:'error',signal:AbortSignal.timeout(15000)});if(!response.ok)throw new Error('History request failed');const snapshot=await response.json();if(snapshot.schema!=='zerone-claim-workflow-v1'||snapshot.local_only!==true)throw new Error('Wrong snapshot');render(snapshot);lastSuccess=new Date();byId('status').dataset.tone='ok';byId('status').textContent=`Updated ${lastSuccess.toLocaleTimeString()}. Node observation; not a state proof.`;}
  catch{byId('status').dataset.tone='error';byId('status').textContent=lastSuccess?`Refresh failed. Displaying stale data from ${lastSuccess.toLocaleTimeString()}; current state is unknown.`:'History unavailable. No current claim state has been loaded. Check the local workflow CLI and node.';}
  finally{refreshing=false;byId('refresh').disabled=false;}
}
byId('refresh').addEventListener('click',refresh);refresh();setInterval(()=>{if(!document.hidden)refresh();},10000);
"""

PAGE = ("""<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Zerone · Local claim history</title><style>""" + STYLE + """</style></head><body>
<a class="skip" href="#main">Skip to claim history</a><header><div class="eyebrow">Zerone · local development</div><h1>A claim, and what followed.</h1><p>Read the contribution, the reasons people recorded, and the changes that followed.</p>
<p class="notice">Fresh local development accounts share one operator. They are not independent scientific endorsements. This is not public zerone-1, and these test funds have no value.</p><p id="chain" class="identifier">Waiting for the local chain</p><p id="notice" class="muted"></p><div class="bar"><button id="refresh" type="button">Refresh history</button><span class="small muted">Refreshes every 10 seconds while visible.</span></div><p id="status" class="status" role="status" aria-live="polite">Loading history…</p><noscript><p class="notice">This viewer needs JavaScript to display a snapshot. Use the local CLI’s history command, or read <a href="/api/history">the history JSON</a>.</p></noscript></header>
<main id="main"><section class="section" aria-labelledby="claims-title"><h2 id="claims-title">Claims and review outcomes</h2><p class="muted">A committed contribution can remain pending. Acceptance records an assessment; it does not certify truth or a payment.</p><div id="claims"></div></section>
<section class="section" aria-labelledby="pending-title"><h2 id="pending-title">Local review preparations</h2><p class="muted">Only actor, round and deadline summaries appear here. Unrevealed votes, reasons and salts stay private.</p><div id="pending" class="grid"></div></section>
<section class="section" aria-labelledby="transactions-title"><h2 id="transactions-title">Transaction inclusion and execution</h2><p class="muted">Inclusion and code 0 concern transaction execution, separately from the claim’s review outcome. These are observed local receipts, not independently verified inclusion proofs. A height-0 response does not establish committed inclusion.</p><div id="transactions" class="grid"></div></section>
<details class="section"><summary>Inspect the full observed snapshot</summary><pre id="raw">No snapshot loaded.</pre></details></main><footer>Read-only viewer · no wallet, signing, transaction or evidence upload controls. Evidence references are displayed as text; their contents are not fetched.</footer><script>""" + SCRIPT + """</script></body></html>""").encode()


def _csp_hash(text):
    return base64.b64encode(hashlib.sha256(text.encode()).digest()).decode()


CSP = ("default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'; "
       "connect-src 'self'; script-src 'sha256-" + _csp_hash(SCRIPT) + "'; "
       "style-src 'sha256-" + _csp_hash(STYLE) + "'")


def _height(value):
    if isinstance(value, bool) or not isinstance(value, (str, int)):
        raise ValueError("invalid height")
    text = str(value)
    if not re.fullmatch(r"0|[1-9][0-9]{0,19}", text) or int(text) > 2**64 - 1:
        raise ValueError("invalid height")
    if isinstance(value, int) and value > 2**53 - 1:
        raise ValueError("large heights must be decimal strings")
    return text


def _snapshot_bytes(snapshot):
    if not isinstance(snapshot, dict) or set(snapshot) != _SNAPSHOT_FIELDS:
        raise ValueError("unexpected snapshot fields")
    if snapshot["schema"] != "zerone-claim-workflow-v1" or snapshot["local_only"] is not True:
        raise ValueError("not a local workflow")
    chain = snapshot["chain_id"]
    if not isinstance(chain, str) or not re.fullmatch(r"zerone-local-[A-Za-z0-9._-]{1,100}", chain):
        raise ValueError("unexpected chain")
    height = _height(snapshot["height"])
    if not isinstance(snapshot["notice"], str):
        raise ValueError("missing notice")
    for key in ("claims", "transactions", "pending_reviews"):
        if not isinstance(snapshot[key], list) or len(snapshot[key]) > MAX_RECORDS:
            raise ValueError("workflow limit exceeded")
    ids = set()
    for item in snapshot["claims"]:
        if not isinstance(item, dict) or set(item) != {"claim_id", "history"}:
            raise ValueError("invalid claim wrapper")
        claim_id, history = item["claim_id"], item["history"]
        if not isinstance(claim_id, str) or not 0 < len(claim_id.encode()) <= 256 or claim_id in ids:
            raise ValueError("invalid claim identity")
        ids.add(claim_id)
        if not isinstance(history, dict) or history.get("chain_id") != chain or _height(history.get("block_height")) != height:
            raise ValueError("mixed history context")
        if not isinstance(history.get("record"), dict) or history["record"].get("claim_id") != claim_id:
            raise ValueError("mixed claim identity")
    for review in snapshot["pending_reviews"]:
        if not isinstance(review, dict) or set(review) != _PENDING_FIELDS:
            raise ValueError("unexpected private review fields")
        for key in ("claim_id", "round_id", "actor", "status"):
            if not isinstance(review[key], str):
                raise ValueError("invalid review summary")
        _height(review["commit_deadline"])
        _height(review["reveal_deadline"])
    for transaction in snapshot["transactions"]:
        if not isinstance(transaction, dict) or set(transaction) - _TRANSACTION_FIELDS:
            raise ValueError("invalid transaction")
        if any(isinstance(value, bool) or not isinstance(value, (str, int)) for value in transaction.values()):
            raise ValueError("transaction receipt fields must be public scalar metadata")
        if "height" in transaction:
            _height(transaction["height"])
    raw = bytearray()
    encoder = json.JSONEncoder(ensure_ascii=True, allow_nan=False, separators=(",", ":"))
    for chunk in encoder.iterencode(snapshot):
        encoded = chunk.encode()
        if len(raw) + len(encoded) > MAX_RESPONSE_BYTES:
            raise ValueError("snapshot exceeds viewer limit")
        raw.extend(encoded)
    return bytes(raw)


def _make_server(snapshot_callback, port):
    if not callable(snapshot_callback) or isinstance(port, bool) or not isinstance(port, int) or not 0 <= port <= 65535:
        raise ValueError("a snapshot callback and port from 0 to 65535 are required")
    callback_lock = threading.Lock()

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_args):
            pass  # Do not turn request data or callback exceptions into logs.

        def send_error(self, code, message=None, explain=None):
            self._send(code, b'{"error":"Request refused."}')

        def _send(self, status, body, content_type="application/json; charset=utf-8"):
            self.send_response(status)
            for key, value in {"Content-Type": content_type, "Content-Length": str(len(body)),
                               "Cache-Control": "no-store", "Content-Security-Policy": CSP,
                               "X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY",
                               "Referrer-Policy": "no-referrer", "Cross-Origin-Resource-Policy": "same-origin",
                               "Cross-Origin-Opener-Policy": "same-origin", "Connection": "close"}.items():
                self.send_header(key, value)
            self.end_headers()
            self.close_connection = True
            if getattr(self, "command", None) != "HEAD":
                try:
                    self.wfile.write(body)
                except (BrokenPipeError, ConnectionResetError):
                    pass

        def _request(self):
            host = f"127.0.0.1:{self.server.server_port}"
            origin = f"http://{host}"
            if self.headers.get_all("Host", []) != [host] or self.headers.get_all("Origin", []) not in ([], [origin]):
                self._send(403, b'{"error":"Foreign host or origin refused."}')
                return
            if self.headers.get("Sec-Fetch-Site") == "cross-site":
                self._send(403, b'{"error":"Cross-site request refused."}')
                return
            if self.headers.get_all("Transfer-Encoding", []) or self.headers.get_all("Content-Length", []) not in ([], ["0"]):
                self._send(400, b'{"error":"Request bodies are not accepted."}')
                return
            if self.command != "GET":
                self._send(405, b'{"error":"Read-only viewer; GET is required."}')
                return
            # BaseHTTPRequestHandler normalizes a leading //; require the original target.
            target = self.requestline.split()[1]
            if target == "/":
                self._send(200, PAGE, "text/html; charset=utf-8")
            elif target == "/api/history":
                if not callback_lock.acquire(blocking=False):
                    self._send(503, b'{"error":"History refresh is already running."}')
                    return
                try:
                    body = _snapshot_bytes(snapshot_callback())
                except Exception:
                    self._send(503, b'{"error":"History unavailable. Check the local workflow and node."}')
                else:
                    self._send(200, body)
                finally:
                    callback_lock.release()
            else:
                self._send(404, b'{"error":"Unknown viewer path."}')

        do_GET = do_HEAD = do_POST = do_PUT = do_PATCH = do_DELETE = do_OPTIONS = do_TRACE = do_CONNECT = _request

    class Server(ThreadingHTTPServer):
        daemon_threads = True

        def get_request(self):
            connection, address = super().get_request()
            connection.settimeout(3)
            return connection, address

    return Server(("127.0.0.1", port), Handler)


def serve(snapshot_callback, port):
    """Serve a validated local snapshot until Ctrl-C; no application writes."""
    with _make_server(snapshot_callback, port) as server:
        print(f"Local claim history: http://127.0.0.1:{server.server_port}/", flush=True)
        try:
            server.serve_forever(poll_interval=0.25)
        except KeyboardInterrupt:
            pass
