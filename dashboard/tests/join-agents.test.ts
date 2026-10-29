import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { describe, it } from "node:test";

import { isDiscoveryShaped } from "../functions/_middleware";

// /join/agents.md is the agents' guide to the household's admission door
// (tools/zerone-door/JOIN-AGENTS.md, PR #160). It is published before the door
// opens, so it must say so on its first line and carry no unfilled template.
const GUIDE_URL = new URL("../public/join/agents.md", import.meta.url);
const NOT_OPEN = "Door status: not open yet. Nothing on this page works today.";
// SHA-256 of the hub machine's name in lowercase, so the name is not written here.
const HUB_NAME_SHA256 = new Set(["da0daaba4b156961c049ab4b85b6d0bcba2872200b7dedb4aa77f9208c601444"]);

const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8");
const guide = existsSync(GUIDE_URL) ? readFileSync(GUIDE_URL, "utf8") : "";
// The guide wraps its prose; compare phrases with whitespace collapsed.
const prose = guide.replace(/\s+/gu, " ");

describe("the agents' join guide at /join/agents.md", () => {
  it("ships in the build output: Vite copies public/ to dist/ unchanged", () => {
    assert.ok(existsSync(GUIDE_URL), "public/join/agents.md must exist");
    const vite = read("vite.config.ts");
    assert.doesNotMatch(vite, /\bpublicDir\s*:|\bcopyPublicDir\s*:/u);
  });

  it("starts with the not-open status line and leaves no template unfilled", () => {
    assert.equal(guide.split("\n")[0], NOT_OPEN);
    assert.ok(!guide.includes("{{") && !guide.includes("}}"), "no {{...}} placeholder may remain");
    assert.match(prose, /R's address is published here when the door opens/u);
    assert.match(prose, /The door room's id is published here when the door opens/u);
    assert.match(prose, /Only after that does the line at the top of this page say the door is open\./u);
    assert.doesNotMatch(guide, /Door status: open\b|the door is open now/iu);
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
    ]) {
      assert.match(prose, hazard);
    }
    // Nobody reviews any claim today, the household's included.
    assert.doesNotMatch(prose, /claims from outside the household|reviews claims from outside/u);
    assert.match(prose, /Remember hazard 8: today nobody reviews claims, so a claim ends INSUFFICIENT/u);
  });

  it("says what R signs, what the door never holds, and what R's float allows", () => {
    assert.match(prose, /R signs its own registration \(step 3\) and the door's admissions, re-grants and revocations, and nothing else\./u);
    assert.doesNotMatch(prose, /signs admissions and nothing else/u);
    assert.match(prose, /The door never makes, asks for, sees or stores your private key or recovery phrase\./u);
    assert.match(prose, /Each new admission needs 3\.25 ZRN of R's balance free beyond what R's live allowances can still draw/u);
    assert.match(prose, /a 5 ZRN float covers about two admissions, the opening canary included\./u);
  });

  it("lists every answer the door gives and when an agent may ask again", () => {
    const step4 = prose.match(/### 4\. Read the answer(.*?)### 5\./u)?.[1] ?? "";
    for (const answer of ["admitted", "regranted", "refused", "pending", "failed"]) {
      assert.match(step4, new RegExp(`\`${answer}\``, "u"), `step 4 must name \`${answer}\``);
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

  it("keeps the SDK and CosmJS code with its timeout and result-code checks", () => {
    assert.match(guide, /npm install @zerone-chain\/sdk@0\.1\.1 /u);
    assert.match(prose, /On 2026-10-29 npm answered 404 for `@zerone-chain\/sdk`: it is not published yet\./u);
    assert.match(guide, /git clone https:\/\/github\.com\/cambridgetcg\/zerone-core\.git/u);
    assert.match(guide, /import \{ accountRegistrationProofSignBytes \} from "@zerone-chain\/sdk\/messages";/u);
    assert.match(guide, /import \{ createZeroneRegistry \} from "@zerone-chain\/sdk\/registry";/u);
    assert.match(guide, /broadcastTimeoutMs: 180_000,/u);
    assert.match(guide, /if \(result\.code !== 0\) throw new Error/u);
    assert.match(guide, /if \(error instanceof TimeoutError\) \{/u);
    assert.match(guide, /if \(landed && landed\.code === 0\) return landed;/u);
    assert.match(guide, /accountType: "agent"/u);
    assert.doesNotMatch(guide, /MsgRotateKey"|typeUrl: "\/zerone\.auth\.v1\.MsgRotateKey/u);
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
    assert.match(prose, /Household beings on the hub are told where the inbox is; it is not published\./u);
    assert.doesNotMatch(guide, /Keychain|keyring|\/Users\/|~\/|\.local\/|\/private\/|inbox\/\S/u);
    // The hub machine's name stays out of this test too: only the SHA-256 of
    // its lowercase form is kept here, and every word of the guide is checked
    // against it.
    const words = new Set(guide.toLowerCase().match(/[a-z][a-z0-9]*/gu) ?? []);
    for (const word of words) {
      const digest = createHash("sha256").update(word).digest("hex");
      assert.ok(!HUB_NAME_SHA256.has(digest), "the guide must not name the hub machine");
    }
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

  it("is listed for agents in llms.txt as not open yet", () => {
    const llms = read("public/llms.txt");
    assert.match(
      llms,
      /^- \[How an agent joins zerone-1\]\(https:\/\/zerone\.ai\/join\/agents\.md\): [^\n]*The door is not open yet, and nothing in the guide works today\./mu,
    );
    const today = llms.split("\n").find((line) => line.startsWith("Today:")) ?? "";
    assert.match(today, /the household does not offer new accounts, starter funds or a supported way to join as a validator\./u);
    assert.match(today, /the door is not open yet \(https:\/\/zerone\.ai\/join\/agents\.md\)/u);
    assert.doesNotMatch(llms, /door is open\b|door opened|open to agents/iu);
  });
});
