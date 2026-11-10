import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { describe, it } from "node:test";

import { isDiscoveryShaped } from "../functions/_middleware";
import { nodeGuidePage } from "../node-guide-page";
import { buildNodeGuideProfile } from "../node-guide-profile";
import { buildUnderstandGuide } from "../understand-guide";

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
    // No door room is open to desks yet, so the guide gives no room id or command to
    // post with. A room can exist before desks may use it, so the page neither says
    // that no room exists nor promises the id on the day a room is made.
    assert.doesNotMatch(guide, /agenttool-lobby room say/u);
    assert.match(prose, /No door room is open to desks yet, and there is no way to ask from another device;/u);
    assert.match(prose, /Once a test agent on another device has been admitted through the lobby, this page publishes the room's id and changes the status line above\./u);
    assert.doesNotMatch(prose, /there is no door room|No door room exists|On that day this page publishes/u);
  });

  it("says what a desk on another device needs first, and gives no room id or post command yet", () => {
    const start = guide.indexOf("## From another device on the household network");
    const end = guide.indexOf("## Refusal reasons");
    assert.ok(start > guide.indexOf("### 6. Submit a claim") && start < end, "the section sits after step 6, before the refusal reasons");
    const section = guide.slice(start, end);
    const text = section.replace(/\s+/gu, " ");
    assert.match(text, /^## From another device on the household network \*\*Not yet, as of 2 November 2026\.\*\* No door room is open to desks yet, and no device other than the hub has joined the household's lobby/u);
    // Every step is the household's decision; the first three need a person, and
    // the fourth (listing the desk) is the household's too.
    assert.match(text, /### What has to happen first Each step in this list is the household's decision, and no agent on your device can do one alone; steps 1–3 of this list need a person at a keyboard\./u);
    assert.equal((section.match(/^\d\. \*\*/gmu) ?? []).length, 4);
    // References to the guide's own steps name them, so they cannot be read as this list.
    assert.match(text, /make them on your own device as steps 1 and 2 under "Steps" show \("Make your two keys and keep them"; "Write and sign your door request"\)/u);
    assert.match(text, /the chain, read as the "Anyone" bullet of step 4 under "Steps" \("Read the answer"\) shows\. There is no reply file for you\./u);
    // Each request goes in a file of its own, so a second request never collides.
    assert.match(text, /Save each request in a new file of its own, named for that request/u);
    assert.match(text, /the door accepts a request for 15 minutes after its `issued_at`/u);
    assert.match(text, /in the conversation `zerone-door`, addressed to the room's host, the hub's desk/u);
    assert.match(text, /An agent never answers that prompt\./u);
    // The typed-yes acts stay a person's: joining, the seat lease and the daily renewal.
    assert.match(text, /with a person at your device and at the hub\./u);
    assert.match(text, /or that person makes a short, bounded seat lease/u);
    assert.doesNotMatch(text, /\ban agent (?:makes|renews|types|answers|admits|knocks|approves)\b/iu);
    assert.match(text, /Until the lobby can renew them by signature, a person renews them each day/u);
    // The per-desk cap is a default the household can change, not a fixed bound.
    assert.match(text, /3 admissions a day per desk by default \(the household can set another number for each desk\)/u);
    assert.doesNotMatch(text, /5 ZRN float/u);
    assert.match(step(3), /What has to happen before a desk can ask is under "From another device on the household network" below\./u);
    // Not yet: no room id, no post command, and nothing stale about the SDK or a device name.
    assert.doesNotMatch(guide, /room_[0-9a-f]{32}/u);
    assert.doesNotMatch(section, /agenttool-lobby/u);
    assert.doesNotMatch(prose, /not (?:yet )?(?:on|published (?:on|to)) npm/iu);
    assert.doesNotMatch(section, /\bMac(?![a-z])|\bMacBook|\biMac/u);
    // R's balance after the 2 November top-ups, with both transactions.
    assert.match(prose, /On 2 November 2026 the household topped R up twice, to 15 ZRN \(transactions `40AD7126585E041AEB8136D1DB44382ABC555AE36A8AA39F174AB19B9203BA4C` at height 1,416,426 and `0ABB6B41EE87F770BD553161E414444CCF3EF3F4255FE35DADB8C6BBB467AF89` at height 1,416,779\)\./u);
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
      /Claims need reviewers, and on \d{4}-\d{2}-\d{2} no one had committed to review one since \d{4}-\d{2}-\d{2}\./u,
      /The last review commitment \(`MsgSubmitCommitment`\) on `zerone-1` was made at height 854,278/u,
      // Re-checked live on 9 November 2026.
      /and none has been made since \(checked 2026-11-09 at about height 1,437,950\)\./u,
      /INSUFFICIENT \(status 10\), and the 0\.2 ZRN review fee is not returned/u,
      /any account holding ZRN, yours included, can file a contradiction that marks a VERIFIED fact CONTESTED/u,
      /Running a validator is not supported\./u,
    ]) {
      assert.match(prose, hazard);
    }
    // Nobody reviews any claim today, the household's included.
    assert.doesNotMatch(prose, /claims from outside the household|reviews claims from outside/u);
    // Step 6 points to hazard 8's dated check instead of saying "today".
    assert.match(step(6), /Remember hazard 8 before you pay a review fee: a claim nobody reviews ends INSUFFICIENT and its fee is not returned, and hazard 8 says when anyone last committed to review a zerone-1 claim\./u);
    // No undated "today" about reviewers anywhere in the guide.
    assert.doesNotMatch(prose, /today (?:there are no reviewers|nobody reviews|no one reviews)|and today there are none/iu);
  });

  it("repeats hazard 8's block, its date and the checked date everywhere the site describes the door", () => {
    // Hazard 8 is the one dated source for "no review commitment since block H, made
    // on date D". Every other place that offers the door's seed carries the same block
    // and the same checked date, and every place that dates the block gives the same
    // block date, so a later re-check of hazard 8 must change them all together.
    const hazard = prose.match(/The last review commitment \(`MsgSubmitCommitment`\) on `zerone-1` was made at height ([\d,]+) on (\d{4}-\d{2}-\d{2}), and none has been made since \(checked (\d{4}-\d{2}-\d{2}) at about height [\d,]+\)/u);
    assert.ok(hazard, "hazard 8 must give the block, the block's date and the date it was checked");
    const [, height = "", blockIso = "", checkedIso = ""] = hazard;
    // Hazard 8's heading carries the same two dates, so it cannot outlive a re-check.
    assert.ok(
      prose.includes(`8. **Claims need reviewers, and on ${checkedIso} no one had committed to review one since ${blockIso}.**`),
      `hazard 8's heading must carry the checked date (${checkedIso}) and the block's date (${blockIso})`,
    );
    const months = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
    // A date in either form the site writes, never part of a longer date or number,
    // so "19 November 2026" does not pass for "9 November 2026".
    const dated = (iso: string): string => {
      const [year, month, day] = iso.split("-");
      return `(?<![\\d-])(?:${Number(day)} ${months[Number(month) - 1]} ${year}|${iso})(?![\\d-])`;
    };
    const blockSince = new RegExp(`since ${dated(blockIso)} \\(block ${height}\\)`, "u");
    const html = read("index.html");
    const honestItem = html.slice(html.indexOf('id="honest-state"'), html.indexOf("</aside>"))
      .match(/<li>Since 29 October 2026 the household&rsquo;s admission door[\s\S]*?<\/li>/u)?.[0] ?? "";
    const agentDoor = html.match(/<p id="agent-door">([\s\S]*?)<\/p>/u)?.[1]?.replace(/\s+/gu, " ") ?? "";
    const llms = read("public/llms.txt").split("\n");
    const node = buildNodeGuideProfile({ sourceCommit: "74e2bbac317dacb1695ed8aed20830310617ea7a", helperSha256: "a1".repeat(32) });
    const nodesDoor = nodeGuidePage(node).match(/<p class="census-note">On zerone-1, validator joining[\s\S]*?<\/p>/u)?.[0] ?? "";
    // [place, text, whether it dates the block]
    const places: [string, string, boolean][] = [
      ["the homepage's honest-state door item", honestItem, true],
      ["the homepage's #agent-door", agentDoor, true],
      ["llms.txt's Today line", llms.find((line) => line.startsWith("Today:")) ?? "", true],
      ["llms.txt's join-guide entry", llms.find((line) => line.startsWith("- [How an agent joins zerone-1](https://zerone.ai/join/agents.md):")) ?? "", false],
      ["the node guide profile's door.reviews", node.live.newAccountAdmission.door.reviews, true],
      ["the /nodes/ door note", nodesDoor, true],
      ["the /understand/ new-accounts row", buildUnderstandGuide(node).sections[5].rows[4].text, true],
    ];
    for (const [name, text, datesBlock] of places) {
      assert.ok(text, `${name} must be found`);
      assert.match(text, new RegExp(`block ${height}(?![\\d,])`, "u"), `${name} must name block ${height}, as hazard 8 does`);
      assert.match(text, new RegExp(dated(checkedIso), "u"), `${name} must carry hazard 8's checked date (${checkedIso})`);
      if (datesBlock) assert.match(text, blockSince, `${name} must date block ${height} as hazard 8 does (${blockIso})`);
    }
  });

  it("repeats hazard 9's sentence on who cannot review wherever the site offers the seed beside hazard 8", () => {
    // Hazard 9 is the one source for why the door's seed alone cannot pay for
    // reviewing on zerone-1: at 2e37c4c, MsgSubmitCommitment refuses a balance below
    // 100,000,000 uzrn. Every place that offers the seed beside hazard 8's dated
    // sentence carries hazard 9's sentence word for word, so a change to the seed
    // or the threshold must change them all together.
    const hazard = prose.match(/9\. \*\*Reviewing and other roles\.\*\* (An account funded only by the ([\d.]+) ZRN starter seed cannot review a zerone-1 claim: a reviewer must hold at least ([\d,]+) ZRN when it commits\.)/u);
    assert.ok(hazard, "hazard 9 must open with the sentence on who cannot review");
    const [, sentence = "", seedZrn = "", minimumZrn = ""] = hazard;
    // The sentence's figures match the rest of the guide.
    assert.match(prose, new RegExp(`This makes a \\*\\*starter seed\\*\\* of ${seedZrn.replace(".", "\\.")} ZRN`, "u"));
    const minimumUzrn = (Number(minimumZrn.replaceAll(",", "")) * 1_000_000).toLocaleString("en-US");
    assert.match(prose, new RegExp(`\`MsgSubmitCommitment\` refuses an account whose balance is below ${minimumUzrn} uzrn`, "u"));
    // It points to where a review can be practised, and promises nothing there.
    const nine = prose.slice(prose.indexOf("9. **Reviewing and other roles.**"), prose.indexOf("10. **"));
    assert.match(nine, /use the separate test chain `zerone-dev-1` \(<https:\/\/zerone\.ai\/development\/>\)/u);
    assert.match(nine, /`docs\/DEVELOPMENT-REVIEW-HANDOFF\.md` in the source repository/u);
    assert.ok(existsSync(new URL("../../docs/DEVELOPMENT-REVIEW-HANDOFF.md", import.meta.url)), "the review handoff hazard 9 names must exist");
    assert.match(nine, /An offer to review there opens no review round, and no response time is promised\./u);
    assert.match(nine, /Running a validator is not supported\./u);
    const html = read("index.html");
    const honestItem = html.slice(html.indexOf('id="honest-state"'), html.indexOf("</aside>"))
      .match(/<li>Since 29 October 2026 the household&rsquo;s admission door[\s\S]*?<\/li>/u)?.[0] ?? "";
    const agentDoor = html.match(/<p id="agent-door">([\s\S]*?)<\/p>/u)?.[1]?.replace(/\s+/gu, " ") ?? "";
    const llms = read("public/llms.txt").split("\n");
    const node = buildNodeGuideProfile({ sourceCommit: "74e2bbac317dacb1695ed8aed20830310617ea7a", helperSha256: "a1".repeat(32) });
    const nodesDoor = nodeGuidePage(node).match(/<p class="census-note">On zerone-1, validator joining[\s\S]*?<\/p>/u)?.[0] ?? "";
    const places: [string, string][] = [
      ["the homepage's honest-state door item", honestItem],
      ["the homepage's #agent-door", agentDoor],
      ["llms.txt's Today line", llms.find((line) => line.startsWith("Today:")) ?? ""],
      ["the node guide profile's door.reviews", node.live.newAccountAdmission.door.reviews],
      ["the /nodes/ door note", nodesDoor],
      ["the /understand/ new-accounts row", buildUnderstandGuide(node).sections[5].rows[4].text],
    ];
    for (const [name, text] of places) {
      assert.ok(text, `${name} must be found`);
      assert.ok(text.includes(sentence), `${name} must carry hazard 9's sentence: ${sentence}`);
      // Beside hazard 8's dated sentence, not somewhere else on the page.
      const dated = text.search(/no one had committed to review a zerone-1 claim since/u);
      assert.ok(dated >= 0 && text.indexOf(sentence) > dated, `${name} must put hazard 9's sentence after hazard 8's dated sentence`);
    }
  });

  it("says in hazard 10 what happens to an accepted claim nobody uses, dated, and llms.txt says the same", () => {
    const start = prose.indexOf("10. **An accepted claim that nobody uses loses standing over time.**");
    assert.ok(start > prose.indexOf("9. **Reviewing and other roles.**") && start < prose.indexOf("## Steps"), "hazard 10 follows hazard 9, before the steps");
    const ten = prose.slice(start, prose.indexOf("## Steps"));
    // It applies only after acceptance, and points back to hazard 8 rather than saying "today".
    assert.match(ten, /This applies only after a claim has been accepted; hazard 8 records that no one had committed to review a zerone-1 claim for months when it was last checked\./u);
    // The source, the date and the live height it was checked at.
    const checked = ten.match(/Checked on (\d{4}-\d{2}-\d{2}) against (2e37c4c), the source `zerone-1` runs, and against the chain's live parameters at about height ([\d,]+)/u);
    assert.ok(checked, "hazard 10 must name its date, commit and height");
    const [, checkedIso = "", commit = ""] = checked;
    // The pace, in epochs and in days. PRUNED is AT_RISK plus 5 + 20 epochs.
    const pace = ten.match(/With no income it goes AT_RISK after (\d+) to (\d+) epochs, EXPIRED (\d+) epochs later and PRUNED (\d+) epochs after that: (\d+) to (\d+) epochs after acceptance/u);
    assert.ok(pace, "hazard 10 must give the AT_RISK, EXPIRED and PRUNED steps in epochs");
    const [low, high, expired, pruned, prunedLow, prunedHigh] = pace.slice(1).map(Number);
    assert.equal(prunedLow, low + expired + pruned);
    assert.equal(prunedHigh, high + expired + pruned);
    assert.match(ten, /at every 10,000th block \(a fitness epoch\)/u);
    assert.match(ten, /so an epoch is about 3\.5 days: AT_RISK after about 60 to 140 days, PRUNED after about 150 to 230\./u);
    // What it keeps, what it loses, and what it is not.
    assert.match(ten, /\*\*The record stays\.\*\* Nothing deletes a PRUNED fact/u);
    assert.match(ten, /from AT_RISK on, the chain grades it UNSUITABLE as training material, and `MsgChallengeFact` refuses it/u);
    assert.match(ten, /\*\*This is not a finding that the claim is false\.\*\*/u);
    assert.match(ten, /Reading a fact adds nothing\./u);
    // The exemption is described by its label, and the guide keeps category empty.
    assert.match(ten, /Leave `category` empty, as step 6 does: the label says nothing about who wrote a fact\./u);
    assert.match(guide, /^ {4}category: "",$/mu);
    // llms.txt's join-guide entry carries the same figures, commit and date.
    const entry = read("public/llms.txt").split("\n").find((line) => line.startsWith("- [How an agent joins zerone-1](https://zerone.ai/join/agents.md):")) ?? "";
    const [year, month, day] = checkedIso.split("-");
    const longDate = `${Number(day)} ${["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"][Number(month) - 1]} ${year}`;
    assert.ok(
      entry.includes(`Its hazard 10 says what happens to an accepted claim that nobody cites or funds: on the version zerone-1 runs (${commit}, with parameters read on ${longDate}) it goes AT_RISK after ${low} to ${high} fitness epochs of 10,000 blocks (about 60 to 140 days at 30-second blocks) and is PRUNED ${expired + pruned} epochs later. Its record stays, and this is not a finding that the claim is false.`),
      "llms.txt's join-guide entry must repeat hazard 10's figures, commit and date",
    );
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

  it("installs the npm release exactly, without install scripts, as ES modules", () => {
    assert.match(guide, /npm install --ignore-scripts --save-exact @zerone-chain\/sdk@0\.1\.1 @cosmjs\/proto-signing@0\.39\.0 \\\n {2}@cosmjs\/stargate@0\.39\.0 @cosmjs\/amino@0\.39\.0 @cosmjs\/encoding@0\.39\.0 @noble\/curves@2\.2\.0 \\\n {2}@noble\/hashes@2\.2\.0\n/u);
    // 0.1.1 has been on npm since 2026-11-02; the old "answered 404" fallback is gone.
    assert.match(prose, /`@zerone-chain\/sdk` has been on npm since 2 November 2026\. Its first version there is 0\.1\.1; 0\.1\.0 was tagged but never published\. Install exactly 0\.1\.1, as above\./u);
    assert.doesNotMatch(prose, /answered 404|it is not published yet|Until it is, build it/u);
    // npm's staged-publishing placeholder holds no code and must never be installed.
    assert.match(prose, /\*\*Never install `0\.0\.0-stage`\*\*: npm's staged publishing made that version as a placeholder/u);
    assert.doesNotMatch(guide, /@zerone-chain\/sdk@(?!0\.1\.1\b)/u);
    // No provenance attestation: the guide says so and gives the bytes to check.
    assert.match(prose, /0\.1\.1 has no npm provenance attestation: the package owner published it by hand, with npm 2FA, from a tarball built from the annotated `sdk-v0\.1\.1` tag\./u);
    for (const value of [
      "3df235181617eb1544f7590abe6aee76c702dee48367d4fa5436e407ae7879a2",
      "8484fd566c6d07a0a6614995417cccdbd7bdceab",
      "sha512-K+tiN/efOIFfwlRoi/oREAq9/qEPBMyIidbNlPS2i7DcApnx7qnIKMYoued9EGbJi6HzMCI8hZgQTmnKo//HFw==",
    ]) {
      assert.ok(guide.includes(`| \`${value}\` |`), `the guide must publish ${value}`);
    }
    assert.ok(guide.includes(`node --print "require('./package-lock.json').packages['node_modules/@zerone-chain/sdk'].integrity"`));
    assert.match(guide, /^curl --fail -sSO https:\/\/registry\.npmjs\.org\/@zerone-chain\/sdk\/-\/sdk-0\.1\.1\.tgz$/mu);
    assert.match(guide, /^shasum -a 256 sdk-0\.1\.1\.tgz {8}# on Linux: sha256sum sdk-0\.1\.1\.tgz$/mu);
    // The optional rebuild starts from the annotated tag and is only compared, never installed.
    assert.match(guide, /^git clone --branch sdk-v0\.1\.1 --depth 1 https:\/\/github\.com\/cambridgetcg\/zerone-core\.git$/mu);
    assert.match(guide, /^ {2}git status --porcelain -- dist {14}# prints nothing: the build matches the tag$/mu);
    assert.match(prose, /This is optional; install from npm either way\./u);
    // It runs in a subshell, so an agent that carries on in the same shell is back in
    // its own project, not inside the SDK clone.
    const rebuild = guide.match(/```sh\ngit clone --branch sdk-v0\.1\.1 [^\n]*\n([\s\S]*?)```/u)?.[1] ?? "";
    assert.match(rebuild, /^\(\n {2}cd zerone-core\/sdk\/typescript\n(?: {2}[^\n]+\n)+\)\n$/u);
    assert.match(rebuild, /^ {2}shasum -a 256 zerone-chain-sdk-0\.1\.1\.tgz$/mu);
    assert.doesNotMatch(guide, /^cd zerone-core/mu);
    assert.match(prose, /The parentheses run the rebuild in a subshell, so your shell is back in your own project when it ends\./u);
    assert.doesNotMatch(guide, /npm install [^\n]*\.tgz|npm install \.\/zerone-core/u);
    assert.doesNotMatch(guide, /npm ci && npm run build/u);
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
    // The skill points to hazard 8's dated check instead of saying "today".
    assert.match(skill, /A claim nobody reviews ends INSUFFICIENT and loses its review fee; the guide's hazard 8 says when anyone last committed to review a zerone-1 claim, so read it before submitting one\./u);
    assert.doesNotMatch(skill, /today nobody reviews|today there are none/iu);
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
    // The SDK has been on npm since 2026-11-02; the index pins 0.1.1 and warns off the placeholder.
    assert.match(llms, /Its install step pins @zerone-chain\/sdk 0\.1\.1 from npm/u);
    assert.match(llms, /^- \[TypeScript SDK\]\([^)]+\): [^\n]*On npm as @zerone-chain\/sdk 0\.1\.1 since 2 November 2026, published by hand without a provenance attestation: install exactly 0\.1\.1, never npm's 0\.0\.0-stage placeholder/mu);
    assert.doesNotMatch(llms, /not a published npm package|not published (?:on|to) npm/iu);
    // The index never names the hub's inbox or key store either.
    assert.doesNotMatch(llms, /Keychain|keyring|\/Users\/|~\/|\/inbox\b|inbox\//iu);
    assert.ok(!llms.includes("{{"));
  });
});
