import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { describe, it } from "node:test";

import { isDiscoveryShaped } from "../functions/_middleware";
import { buildNodeGuideProfile } from "../node-guide-profile";

// /join/agents.md is the agents' guide to the household's admission door
// (tools/zerone-door, merged in PR #160; the door's JOIN-AGENTS.md now only
// points here, so there is one agents' guide). The door opened on
// 29 October 2026 to agents on the household hub; desks on the household's
// other devices are not yet included. The guide must say exactly that on its
// first line and carry no unfilled template.
const GUIDE_URL = new URL("../public/join/agents.md", import.meta.url);
const OPEN =
  "Door status: open since 29 October 2026 to agents on the household hub; desks on the household's other devices: not yet.";
// R, the door's registrar key: zerone-1's bootstrap registrar since proposal 8.
const R_ADDRESS = "zrn1axn0xlhazjm9h4xn8jfamf48lxcks0ts7hk7tu";
// The door's opening acts (D1-D4) and the canary's three transactions, as
// looked up on zerone-1 on 2026-10-29.
const OPENING_TXS = [
  ["C498D48DAE89906C4E5BA5695B4AA0FDDDFF47D98BB248DAD1E8452E73638698", "1,405,208"],
  ["1415824A6BCAD8DF72CC78E651FF9060A569E7FCE76B2F051AF7B1562FFD88C4", "1,405,210"],
  ["EC18E9C2C0B0C8C54738A6D6474DAE69B6AB6A44D06BCD1E2EF08CDE38077480", "1,405,226"],
  ["596B37DAED69F626996C5461F8F1DD1AB39FBDAC85A3FE12D0C60CADC895AAD6", "1,405,228"],
] as const;
const CANARY_TXS = [
  ["592592BCB4CCD9C03CE5269D45177142B09C71D710F75C4789B99B2621D2BBEB", "1,405,404"],
  ["FE16B2BF252957E6076A97EF6FF6B3AFC85F29285D2B333AB54E8263475439D6", "1,405,406"],
  ["900A6FA8DFF39E30DB95A466051523D80BECA269C07D6D5BF3DD82E725A68304", "1,405,407"],
] as const;
// SHA-256 of the hub machine's name in lowercase, so the name is not written here.
const HUB_NAME_SHA256 = new Set(["da0daaba4b156961c049ab4b85b6d0bcba2872200b7dedb4aa77f9208c601444"]);

const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8");
const guide = existsSync(GUIDE_URL) ? readFileSync(GUIDE_URL, "utf8") : "";
// The guide wraps its prose; compare phrases with whitespace collapsed.
const prose = guide.replace(/\s+/gu, " ");
const step = (n: number) => prose.match(new RegExp(`### ${n}\\. (.*?)(?:### ${n + 1}\\.|## Refusal reasons)`, "u"))?.[1] ?? "";

describe("the agents' join guide at /join/agents.md", () => {
  it("ships in the build output: Vite copies public/ to dist/ unchanged", () => {
    assert.ok(existsSync(GUIDE_URL), "public/join/agents.md must exist");
    const vite = read("vite.config.ts");
    assert.doesNotMatch(vite, /\bpublicDir\s*:|\bcopyPublicDir\s*:/u);
  });

  it("starts with the open status line and leaves no template or placeholder unfilled", () => {
    assert.equal(guide.split("\n")[0], OPEN);
    assert.ok(!guide.includes("{{") && !guide.includes("}}"), "no {{...}} placeholder may remain");
    assert.ok(!guide.includes("<R's address>"), "R's address must be published, not a placeholder");
    // Nothing stale from before the door opened.
    assert.doesNotMatch(prose, /not open yet|Nothing on this page works today|before the door opens|Before the door opens|published here when the door opens|Once it opens|When the door is open/u);
    assert.equal((guide.match(/^Door status:/gmu) ?? []).length, 1);
  });

  it("publishes R's address wherever an agent needs it", () => {
    assert.match(prose, new RegExp(`R's address is \`${R_ADDRESS}\``, "u"));
    assert.match(guide, new RegExp(`^const door = "${R_ADDRESS}";`, "mu"));
    assert.ok(guide.includes(`/cosmos/feegrant/v1beta1/allowance/${R_ADDRESS}/<address>`));
    // The same R everywhere the site and the mainnet docs name it.
    for (const path of ["index.html", "public/llms.txt", "../deploy/mainnet/JOIN.md", "../deploy/mainnet/TRUST.md"]) {
      assert.ok(read(path).includes(R_ADDRESS), `${path} must name R`);
    }
    const node = buildNodeGuideProfile({ sourceCommit: "74e2bbac317dacb1695ed8aed20830310617ea7a", helperSha256: "a1".repeat(32) });
    assert.equal(node.live.newAccountAdmission.door.registrar, R_ADDRESS);
    // The mainnet docs say R signs the door's admissions, so "household broadcasts
    // are paused" stays true after the next admission, and the genesis float's
    // fee-grant role is dated.
    const join = read("../deploy/mainnet/JOIN.md").replace(/\s+/gu, " ");
    assert.match(join, /Household broadcasts are paused too, apart from the transactions listed under \[Transactions since the pause\]\(#transactions-since-the-pause\) and the door's admissions, which R signs on request/u);
    assert.match(read("../deploy/mainnet/TRUST.md"), /claim the 0\.222 ZRN bonus \(at genesis; since proposal 8 the door's key R admits agents and grants their fees\) \|/u);
  });

  it("cites the opening and canary transactions as evidence that the path works", () => {
    for (const [hash, height] of [...OPENING_TXS, ...CANARY_TXS]) {
      assert.match(prose, new RegExp(`\\| \`${hash}\` \\| ${height} \\|`, "u"), hash);
    }
    assert.match(prose, /Governance proposal 8 made R the chain's bootstrap registrar, lowered the chain's cap on the registrar's admissions from 5,000 to 100 per admission window of 34,272 blocks, and suspended the `agenttool-invocation-v1` attestation adapter\./u);
    assert.match(prose, /the proposal passed on 2026-10-29 at 13:51 UTC/u);
    assert.match(prose, /zrn1duyn96d6x99q8lmvhjc4d9d0x32ypuydxn98cd/u);
    // The canary did not run step 5's code word for word; the guide says so.
    assert.match(prose, /So the canary proves the door and the chain; it did not run step 5's code word for word\./u);
  });

  it("says who may ask today, how hub beings learn the inbox, and how desks join later", () => {
    assert.match(prose, /Today the door admits \*\*agents on the household hub\*\*/u);
    // The guide does not name the inbox, and does not claim its path is secret:
    // the door's public source gives its default home. Privacy protects it.
    assert.match(prose, /This guide does not name the door's inbox; the household tells each being on the hub where it is \(step 3\)\. What protects the inbox is that it is private to the door's user, not that its path is secret\./u);
    assert.match(step(3), /This guide does not name the inbox; the household tells hub beings where it is\. What protects the inbox is that it is private to the door's user, not that its path is secret/u);
    assert.doesNotMatch(prose, /location is not published|it is not published\.|inbox is secret|secret inbox/u);
    assert.match(prose, /\*\*Desks on the household's other devices: not yet\.\*\* The household will add them later, through the lobby/u);
    assert.match(step(3), /\*\*From a desk on another of the household's devices:\*\* not yet\. The household will add desks later, through the lobby\./u);
    // No lobby room exists yet, so the guide gives no room id or command to post with.
    assert.doesNotMatch(guide, /agenttool-lobby room say/u);
  });

  it("keeps every hazard of the version zerone-1 runs", () => {
    for (const hazard of [
      /Never send `MsgRotateKey`\./u,
      /`system` freezes your own account at once/u,
      /`did:zrn:` followed by your Ed25519 identity public key as 64 lowercase hex characters/u,
      /Registration does not yet prove you hold your identity key\./u,
      /Leave field 7 \(`identityProofSignature`\) of `MsgRegisterAccount` empty\./u,
      /Set gas and fees yourself\./u,
      /Use a long broadcast timeout and check the result code\./u,
      /Claims need reviewers, and today there are none\./u,
      /The last review commitment \(`MsgSubmitCommitment`\) on `zerone-1` was made at height 854,278/u,
      /INSUFFICIENT \(status 10\), and the 0\.2 ZRN review fee is not returned/u,
      /any account holding ZRN, yours included, can file a contradiction that marks a VERIFIED fact CONTESTED/u,
      /Running a validator is not supported\./u,
    ]) {
      assert.match(prose, hazard);
    }
    // Nobody reviews any claim today, the household's included.
    assert.doesNotMatch(prose, /claims from outside the household|reviews claims from outside/u);
    assert.match(prose, /Remember hazard 8: today nobody reviews claims, so a claim ends INSUFFICIENT/u);
  });

  it("says what R signs, what the door never holds, and what R's float allows", () => {
    assert.match(prose, /R signs its own registration \(item 3 below\) and the door's admissions, re-grants and revocations, and nothing else\./u);
    assert.doesNotMatch(prose, /signs admissions and nothing else/u);
    assert.match(prose, /The door never makes, asks for, sees or stores your private key or recovery phrase\./u);
    assert.match(prose, /Each new admission needs 3\.25 ZRN of R's balance free beyond what R's live allowances can still draw/u);
    assert.match(prose, /a 5 ZRN float covers about two admissions, the opening canary included\./u);
    assert.match(prose, /The chain's own cap on the registrar also applies: 100 per window since proposal 8 passed\./u);
  });

  it("lists every answer the door gives and when an agent may ask again", () => {
    const step4 = step(4);
    for (const answer of ["admitted", "regranted", "refused", "pending", "failed"]) {
      assert.match(step4, new RegExp(`\`${answer}\``, "u"), `step 4 must name \`${answer}\``);
    }
    for (const reason of ["inclusion_timeout", "broadcast_failed", "delivery_failed", "expired", "superseded", "broadcast_rejected", "postcheck_failed"]) {
      assert.match(step4, new RegExp(`\`${reason}\``, "u"), `step 4 must name \`${reason}\``);
    }
    assert.match(step4, /It can land up to the reply's `timeout_height`/u);
    assert.match(step4, /A `failed` answer with reason `expired` or `superseded` means the door's transaction can never run and you were not admitted: you may ask again\./u);
    assert.match(step4, /After `postcheck_failed` the transaction ran/u);
    assert.match(step4, /The door answers each request once/u);
    assert.match(prose, /\| `request_stale`, `request_from_future`, `request_seen` \| Make a new request/u);
  });

  it("gives the claim text's length limits beside the claim", () => {
    assert.match(guide, /const myClaim = "\.\.\.";\s+\/\/ [^\n]*20 to 1,000 bytes/u);
    assert.match(prose, /The claim's text must be at least 20 and at most 1,000 bytes long/u);
  });

  it("installs without install scripts, from a packed SDK, with exact versions, as ES modules", () => {
    assert.match(guide, /npm install --ignore-scripts --save-exact @zerone-chain\/sdk@0\.1\.1 /u);
    assert.match(prose, /On 2026-10-29 npm answered 404 for `@zerone-chain\/sdk`: it is not published yet\./u);
    assert.match(guide, /git clone https:\/\/github\.com\/cambridgetcg\/zerone-core\.git/u);
    assert.match(guide, /\(cd zerone-core\/sdk\/typescript && npm ci --ignore-scripts && npm run build && npm pack --ignore-scripts\)/u);
    assert.match(guide, /npm install --ignore-scripts --save-exact \.\/zerone-core\/sdk\/typescript\/zerone-chain-sdk-0\.1\.1\.tgz/u);
    // The old folder install linked the SDK and loaded two copies of CosmJS.
    assert.doesNotMatch(guide, /npm install \.\/zerone-core\/sdk\/typescript /u);
    assert.doesNotMatch(guide, /npm ci && npm run build/u);
    assert.match(prose, /Install the packed file, not the folder/u);
    assert.match(prose, /save it in a file whose name ends in `\.mjs` \(for example `join\.mjs`\), or set `"type": "module"`/u);
  });

  it("keeps the SDK and CosmJS code with its timeout height and honest result handling", () => {
    assert.match(guide, /import \{ accountRegistrationProofSignBytes \} from "@zerone-chain\/sdk\/messages";/u);
    assert.match(guide, /import \{ createZeroneRegistry \} from "@zerone-chain\/sdk\/registry";/u);
    assert.match(guide, /broadcastTimeoutMs: 180_000,/u);
    // Step 5: a timeout height of height + 40, passed to signAndBroadcast as its fifth argument.
    assert.match(guide, /const timeoutHeight = BigInt\(await client\.getHeight\(\)\) \+ 40n;/u);
    assert.match(guide, /client\.signAndBroadcast\(address, messages, fee\(gas\), "", timeoutHeight\)/u);
    // CheckTx refusals, landed-but-failed, and never-seen are told apart.
    assert.match(guide, /if \(error instanceof BroadcastTxError && error\.code !== 19\) \{/u);
    assert.match(guide, /if \(!\(error instanceof TimeoutError\)\) \{/u);
    assert.match(guide, /if \(result\.code !== 0\) \{/u);
    assert.match(guide, /landed in block \$\{result\.height\} and failed/u);
    // A transaction refused by the ante handler during delivery takes no fee and
    // leaves the sequence unchanged, so the guide says "usually" and how to tell.
    assert.match(guide, /`Usually its fee was taken \(check your account's sequence: if it moved from \$\{sequence\}, the fee was charged\)\. `/u);
    assert.doesNotMatch(guide, /"Its fee was taken\./u);
    assert.match(prose, /Usually its fee was taken from your allowance: read your account's sequence, and if it moved, the fee was charged\. If it did not move, the block refused the transaction before it ran/u);
    assert.match(guide, /not seen yet; it may still land up to height \$\{timeoutHeight\}/u);
    assert.doesNotMatch(guide, /not seen in 180 s: look up/u);
    assert.match(prose, /Never resend blindly on any error after a broadcast\./u);
    assert.match(prose, /If the hash is unseen and the sequence has not moved, it never ran and never can, and you may send again\./u);
    assert.match(guide, /accountType: "agent"/u);
    assert.doesNotMatch(guide, /MsgRotateKey"|typeUrl: "\/zerone\.auth\.v1\.MsgRotateKey/u);
  });

  it("states the seed's vesting, the per-period budget, and what a pre-admission Simulate shows", () => {
    assert.match(prose, /The seed vests one block after your admission: a claim run in the admission block fails with `cliff period not reached`\./u);
    // A claim at the pot's endBlock already vests in full (CalculateClaimable at
    // 2e37c4c), so the rule is "after your admission is visible", not "past endBlock".
    assert.match(step(5), /A claim that runs in your admission block itself fails with `cliff period not reached`\. Once your admission is visible on the chain, a claim you send lands in a later block, when the seed has vested \(the pot's `schedule\.endBlock` is your admission height plus one\), so one more block is enough\./u);
    assert.doesNotMatch(prose, /a claim run sooner fails/u);
    assert.match(prose, /the claim, its retry and registration need 470,000, so registration must wait for the next 15-minute period\./u);
    assert.match(step(5), /You cannot dry-run these two transactions before you are admitted\. The chain stops at the fee grant \(`fee-grant not found`\) before any message runs, so a Simulate then shows only that a transaction is well formed, not that it will pass\./u);
  });

  it("writes the hub request under another name, then renames it", () => {
    const step3 = guide.slice(guide.indexOf("### 3. Send the request"), guide.indexOf("### 4. Read the answer"));
    assert.match(step3, /const partial = join\(inbox, `\$\{requestId\}\.partial`\);/u);
    assert.match(step3, /writeFileSync\(partial, doorRequest, \{ mode: 0o600, flag: "wx" \}\);/u);
    assert.match(step3, /renameSync\(partial, join\(inbox, `\$\{requestId\}\.request`\)\);/u);
    assert.ok(step3.indexOf("writeFileSync(partial") < step3.indexOf("renameSync(partial"));
    assert.match(step3.replace(/\s+/gu, " "), /a file it catches half-written is refused as malformed/u);
  });

  it("keeps the ten-line request format", () => {
    const format = guide.match(/```\n(zerone-door-request\/v1\n[\s\S]*?)```/u)?.[1];
    assert.ok(format, "the request format block must remain");
    assert.deepEqual(
      format.trimEnd().split("\n").map((line) => line.split(":")[0]),
      ["zerone-door-request/v1", "chain_id", "address", "identity_public_key", "nonce", "issued_at",
        "account_public_key", "account_signature", "identity_signature", "pow_nonce"],
    );
  });

  it("names no private inbox path, key store or hub machine", () => {
    assert.doesNotMatch(guide, /Keychain|keyring|\/Users\/|~\/|\.local\/|\/private\/|\/inbox\b|inbox\/\S/iu);
    assert.match(guide, /const inbox = "<the inbox folder the household told you>";/u);
    // The hub machine's name stays out of this test too: only the SHA-256 of
    // its lowercase form is kept here, and every word of the guide is checked
    // against it.
    // The other pages that now describe the open door are held to the same rule.
    const node = buildNodeGuideProfile({ sourceCommit: "74e2bbac317dacb1695ed8aed20830310617ea7a", helperSha256: "a1".repeat(32) });
    const pages: [string, string][] = [
      ["the guide", guide],
      ["index.html", read("index.html")],
      ["llms.txt", read("public/llms.txt")],
      ["the node guide profile", JSON.stringify(node)],
      ["deploy/mainnet/JOIN.md", read("../deploy/mainnet/JOIN.md")],
      ["deploy/mainnet/TRUST.md", read("../deploy/mainnet/TRUST.md")],
      ["tools/zerone-door/JOIN-AGENTS.md", read("../tools/zerone-door/JOIN-AGENTS.md")],
      ["skills/zerone-onboarding/SKILL.md", read("../skills/zerone-onboarding/SKILL.md")],
    ];
    for (const [name, text] of pages) {
      const words = new Set(text.toLowerCase().match(/[a-z][a-z0-9]*/gu) ?? []);
      for (const word of words) {
        const digest = createHash("sha256").update(word).digest("hex");
        assert.ok(!HUB_NAME_SHA256.has(digest), `${name} must not name the hub machine`);
      }
      // Generic shapes only: the test itself must not spell out a private location.
      // (The node guide's local sandbox legitimately passes `--keyring-backend test`,
      // so "keyring" is checked in the guide and llms.txt only.)
      assert.doesNotMatch(text, /Keychain|\/inbox\b|inbox\/|~\/|\/Users\//iu, `${name} must not name the inbox or a key store`);
    }
  });

  it("is the only agents' guide: the door's JOIN-AGENTS.md and the onboarding skill point here", () => {
    // The door's first draft of this guide said "not open yet" and kept {{...}}
    // placeholders. It is now a short pointer, so it cannot contradict this page.
    const pointer = read("../tools/zerone-door/JOIN-AGENTS.md");
    assert.ok(pointer.includes("<https://zerone.ai/join/agents.md>"));
    assert.ok(pointer.includes("(../../dashboard/public/join/agents.md)"));
    assert.doesNotMatch(pointer, /Door status:|not open yet|\{\{|signAndBroadcast|```/u);
    assert.ok(pointer.trimEnd().split("\n").length <= 15, "a pointer, not a second copy of the guide");
    assert.match(read("../tools/zerone-door/README.md").replace(/\s+/gu, " "), /The agent-facing guide is published at <https:\/\/zerone\.ai\/join\/agents\.md>/u);
    // The onboarding skill sends hub agents here and keeps every other lane paused.
    const skill = read("../skills/zerone-onboarding/SKILL.md").replace(/\s+/gu, " ");
    assert.match(skill, /Since 2026-10-29 the household's admission door admits AI agents on the household hub, by request, through https:\/\/zerone\.ai\/join\/agents\.md; every other onboarding lane stays paused\./u);
    assert.match(skill, /## The one open lane: the agents' door/u);
    assert.match(read("../skills/README.md").replace(/\s+/gu, " "), /`zerone-onboarding` \(point agents on the household hub to the agents' door/u);
  });

  it("is served as a static file, as Markdown that is revalidated on every read", () => {
    // Functions run only for /api and discovery-shaped paths; a .md file is
    // neither, so Pages serves the file itself.
    assert.equal(isDiscoveryShaped("/join/agents.md"), false);
    const routes = JSON.parse(read("public/_routes.json")) as { include: string[] };
    assert.ok(!routes.include.some((pattern) => pattern.includes(".md") || pattern.startsWith("/join")));
    const headers = read("public/_headers");
    assert.match(headers, /\n\/join\/\*\n {2}Cache-Control: public, max-age=0, must-revalidate, no-transform\n/u);
    assert.match(headers, /\n\/join\/agents\.md\n {2}Content-Type: text\/markdown; charset=utf-8\n/u);
  });

  it("is listed for agents in llms.txt as open to agents on the household hub", () => {
    const llms = read("public/llms.txt");
    assert.match(
      llms,
      /^- \[How an agent joins zerone-1\]\(https:\/\/zerone\.ai\/join\/agents\.md\): [^\n]*The door has been open to AI agents on the household hub since 29 October 2026; desks on the household's other devices are not yet included\./mu,
    );
    const today = llms.split("\n").find((line) => line.startsWith("Today:")) ?? "";
    assert.match(today, /there is no website sign-up and no supported way to join as a validator\./u);
    assert.match(today, /Since 29 October 2026 the household's admission door has been open to AI agents on the household hub/u);
    assert.match(today, new RegExp(`The door's key, ${R_ADDRESS}, is the chain's bootstrap registrar, capped by governance at 100 admissions per 34,272 blocks`, "u"));
    assert.match(today, /Agents on the household's other devices cannot use the door yet \(https:\/\/zerone\.ai\/join\/agents\.md\)/u);
    assert.doesNotMatch(llms, /door is not open|not open yet|admission remains paused/iu);
    // The index never names the hub's inbox or key store either.
    assert.doesNotMatch(llms, /Keychain|keyring|\/Users\/|~\/|\/inbox\b|inbox\//iu);
    assert.ok(!llms.includes("{{"));
  });
});
