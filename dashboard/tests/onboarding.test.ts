import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  BLOCK_FRESHNESS_WINDOW_MS,
  assessNetworkReadiness,
  onboardingPresentation,
  type OnboardingState,
} from "../src/onboarding";

const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
const css = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");
const main = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");

function elementById(source: string, tagName: string, id: string): string {
  const idIndex = source.indexOf(`id="${id}"`);
  assert.ok(idIndex >= 0, `missing #${id}`);
  const start = source.lastIndexOf(`<${tagName}`, idIndex);
  const close = `</${tagName}>`;
  const end = source.indexOf(close, idIndex);
  assert.ok(start >= 0 && end > idIndex, `missing complete <${tagName}>#${id}`);
  return source.slice(start, end + close.length);
}

function visibleText(fragment: string): string {
  return fragment
    .replace(/<[^>]+>/gu, " ")
    .replace(/&amp;/gu, "&")
    .replace(/&lt;/gu, "<")
    .replace(/&gt;/gu, ">")
    .replace(/\s+/gu, " ")
    .trim();
}

const onboarding = elementById(html, "section", "onboarding");
const onboardingStart = html.indexOf(onboarding);
const onboardingEnd = onboardingStart + onboarding.length;

describe("production onboarding surface", () => {
  it("is the first primary action and follows the honest-state disclosure", () => {
    const truthTitle = html.indexOf('id="truth-banner-title"');
    const truthEnd = html.indexOf("</aside>", truthTitle) + "</aside>".length;
    const readingPathStart = html.lastIndexOf(
      "<nav",
      html.indexOf('id="reading-path"'),
    );
    const walletStart = html.lastIndexOf(
      "<section",
      html.indexOf('id="wallet"'),
    );

    assert.ok(truthEnd < onboardingStart);
    assert.ok(onboardingEnd < readingPathStart);
    assert.ok(readingPathStart < walletStart);
    assert.match(html.slice(truthEnd, onboardingStart), /^\s*$/u);
    assert.match(html.slice(onboardingEnd, readingPathStart), /^\s*$/u);
    assert.equal(html.match(/href="#onboarding"/gu)?.length, 3);
    assert.match(
      html,
      /<a class="button button-primary" href="#onboarding">\s*Start here/u,
    );
    assert.doesNotMatch(
      html.slice(0, onboardingStart),
      /class="[^"]*wallet-connect/u,
    );
  });

  it("offers one zero-account route, one existing-account route, and an honest pause", () => {
    const text = visibleText(onboarding);
    assert.match(text, /Explore without joining/u);
    assert.match(text, /Completion: you reach the dashboard with zero credentials requested/u);
    assert.match(text, /Bring an existing account/u);
    assert.match(
      text,
      /Requires JavaScript and Keplr\. Connection is not account creation, admission, identity proof, or funding/u,
    );
    assert.match(text, /Need a new account\? Paused in this release|Paused in this release Need a new account\?/u);
    assert.match(text, /No signup, key creation, starter funds, or waitlist is active/u);
    assert.match(text, /Do not send money, secrets, identity documents, or seed phrases/u);
    assert.match(text, /never creates keys,[\s\S]*or broadcasts a transaction/u);

    assert.equal(onboarding.match(/<button\b/gu)?.length, 1);
    assert.match(
      onboarding,
      /<button[\s\S]*class="button button-ghost wallet-connect"[\s\S]*data-disconnected-label="Connect existing account"[\s\S]*data-js-required="wallet"[\s\S]*disabled/u,
    );
    assert.doesNotMatch(onboarding, /<(?:form|input|select|textarea|dialog)\b/iu);
    assert.doesNotMatch(onboarding, /\b(?:data-endpoint|data-fetch|data-secret|data-create)\s*=/iu);
    assert.doesNotMatch(onboarding, /\/api\//u);
  });

  it("keeps the release boundary available without JavaScript", () => {
    assert.ok(
      onboardingEnd <
        html.indexOf('<script type="module" src="/src/main.ts"></script>'),
    );
    assert.match(onboarding, /id="onboarding-network-state">Checking mainnet</u);
    assert.match(onboarding, /id="onboarding-wallet-state">Optional</u);
    assert.match(onboarding, /data-onboarding-status="admission" data-tone="paused"/u);
    assert.match(
      onboarding,
      /deploy\/mainnet\/JOIN\.md#onboarding-and-transaction-lanes-are-paused/u,
    );
    assert.doesNotMatch(onboarding, /<noscript\b/iu);
    assert.doesNotMatch(onboarding, /<script\b|\son[a-z]+\s*=/iu);
  });

  it("wires persistent network and wallet states without activating an admission lane", () => {
    assert.match(main, /initialiseOnboarding\(byId<HTMLElement>\("onboarding"\)\)/u);
    assert.match(main, /onboarding\.setNetwork\(readiness\)/u);
    assert.match(main, /onboarding\.setNetwork\("unavailable"\)/u);
    assert.match(main, /onboarding\.setWallet\(\{ state: "connecting" \}\)/u);
    assert.match(main, /onboarding\.setWallet\(\{ state: "error", message \}\)/u);
    assert.match(
      main,
      /\.wallet-connect"\)\.forEach\(\(button\) => \{[\s\S]*button\.addEventListener\("click"[\s\S]*button\.disabled = false;/u,
    );
    assert.doesNotMatch(main, /(?:create|register|issue|fund)Onboarding/u);
  });

  it("keeps both wallet controls honest until their handlers exist", () => {
    const walletButtons = [...html.matchAll(/<button[\s\S]*?<\/button>/gu)].filter(
      ([button]) => button.includes("wallet-connect"),
    );

    assert.equal(walletButtons.length, 2);
    for (const [button] of walletButtons) {
      assert.match(button, /data-js-required="wallet"/u);
      assert.match(button, /\sdisabled(?:\s|>)/u);
    }
    assert.equal(html.match(/Requires JavaScript and (?:Keplr|the Keplr extension)\./gu)?.length, 2);

    const registrationStart = main.indexOf(
      'document.querySelectorAll<HTMLButtonElement>(".wallet-connect").forEach((button) => {',
    );
    const registrationEnd = main.indexOf("});", registrationStart);
    const registration = main.slice(registrationStart, registrationEnd);
    assert.ok(registrationStart >= 0);
    assert.ok(registrationEnd > registrationStart);
    assert.ok(
      registration.indexOf('button.addEventListener("click"') <
        registration.indexOf("button.disabled = false;"),
    );
  });

  it("uses a three-to-one responsive layout with visible status tones", () => {
    assert.match(
      css,
      /\.onboarding-choices\s*\{[^}]*grid-template-columns:\s*repeat\(3,\s*minmax\(0,\s*1fr\)\)/su,
    );
    assert.match(
      css,
      /@media \(max-width: 820px\)[\s\S]*\.onboarding-readiness,\s*\.onboarding-choices\s*\{[^}]*grid-template-columns:\s*1fr/su,
    );
    for (const tone of ["ready", "checking", "error", "paused"]) {
      assert.match(css, new RegExp(`data-tone="${tone}"`, "u"));
    }
    assert.match(
      css,
      /@media \(max-width: 1040px\)[\s\S]*\.topbar-actions\s*\{[^}]*grid-column:\s*2;[^}]*grid-row:\s*1/su,
    );
  });
});

describe("onboarding state model", () => {
  it("allows two observed 30-second block intervals plus polling latency", () => {
    assert.equal(BLOCK_FRESHNESS_WINDOW_MS, 75_000);
    const base = {
      catchingUp: false,
      chainMatches: true,
      regressed: false,
    };
    assert.equal(
      assessNetworkReadiness({ ...base, blockAgeMs: BLOCK_FRESHNESS_WINDOW_MS }),
      "ready",
    );
    assert.equal(
      assessNetworkReadiness({
        ...base,
        blockAgeMs: BLOCK_FRESHNESS_WINDOW_MS + 1,
      }),
      "stale",
    );
    assert.equal(
      assessNetworkReadiness({ ...base, blockAgeMs: 1, catchingUp: true }),
      "syncing",
    );
    assert.equal(
      assessNetworkReadiness({ ...base, blockAgeMs: 1, chainMatches: false }),
      "stale",
    );
    assert.equal(
      assessNetworkReadiness({ ...base, blockAgeMs: 1, regressed: true }),
      "stale",
    );
  });

  it("presents every first-run network state honestly", () => {
    const expected = {
      checking: ["Checking mainnet", "checking"],
      ready: ["Ready to explore", "ready"],
      syncing: ["Node is syncing", "checking"],
      stale: ["Live check needs attention", "error"],
      unavailable: ["Live check unavailable", "error"],
    } as const;

    for (const [network, [label, tone]] of Object.entries(expected)) {
      const view = onboardingPresentation({
        network: network as OnboardingState["network"],
        wallet: { state: "disconnected" },
      });
      assert.equal(view.network.label, label);
      assert.equal(view.network.tone, tone);
    }
  });

  it("keeps wallet decline/failure recoverable and connection explicit", () => {
    assert.deepEqual(
      onboardingPresentation({
        network: "ready",
        wallet: { state: "disconnected" },
      }).wallet,
      {
        label: "Optional",
        detail: "Explore first, or connect an existing account.",
        tone: "neutral",
      },
    );
    assert.equal(
      onboardingPresentation({
        network: "ready",
        wallet: { state: "connecting" },
      }).wallet.label,
      "Waiting for Keplr",
    );
    assert.deepEqual(
      onboardingPresentation({
        network: "ready",
        wallet: { state: "error", message: "The request was declined in Keplr." },
      }).wallet,
      {
        label: "Connection not completed",
        detail: "The request was declined in Keplr.",
        tone: "error",
      },
    );
    assert.deepEqual(
      onboardingPresentation({
        network: "ready",
        wallet: {
          state: "connected",
          address: "zrn1abcdefghijklmnopqrstuv",
          balanceZrn: "12.5",
        },
      }).wallet,
      {
        label: "Existing account connected",
        detail: "zrn1abcdef…qrstuv · balance: 12.5 ZRN.",
        tone: "ready",
      },
    );
  });
});
