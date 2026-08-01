import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

import {
  PiFragmentError,
  parsePiCallbackFragment,
} from "../src/pi-fragment";

const TEST_DIRECTORY = dirname(fileURLToPath(import.meta.url));
const DASHBOARD_ROOT = resolve(TEST_DIRECTORY, "..");
const CALLBACK_HTML_PATH = resolve(
  DASHBOARD_ROOT,
  "pi/callback/index.html",
);
const CALLBACK_SOURCE_PATH = resolve(
  DASHBOARD_ROOT,
  "src/pi-callback.ts",
);
const PI_TRANSPORT_PATH = resolve(DASHBOARD_ROOT, "src/pi.ts");
const MAIN_SOURCE_PATH = resolve(DASHBOARD_ROOT, "src/main.ts");
const MAIN_HTML_PATH = resolve(DASHBOARD_ROOT, "index.html");
const HEADERS_PATH = resolve(DASHBOARD_ROOT, "public/_headers");
const STATE = "s".repeat(43);
const ACCESS_TOKEN = "pi-token.with_safe-characters_1234567890";
const ADDRESS = "zrn100mxrvv5chhhrj0yd9y4q8354z4edm42mukf5r";
const OTHER_ADDRESS = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";

interface TestWalletState {
  name: string;
  address: string;
  accountId: string;
  balanceUzrn: string;
  incomingFeeGrants: null;
}

interface TestWalletSignature {
  pub_key: {
    type: "tendermint/PubKeySecp256k1";
    value: string;
  };
  signature: string;
}

interface WalletProofModule {
  signWalletControlProof(
    wallet: TestWalletState,
    message: string,
  ): Promise<TestWalletSignature>;
}

const dynamicImport = new Function(
  "url",
  "return import(url)",
) as (url: string) => Promise<WalletProofModule>;
const walletModuleUrl = new URL("../src/wallet.ts", import.meta.url).href;

function loadWalletProofModule(): Promise<WalletProofModule> {
  return dynamicImport(walletModuleUrl);
}

function wallet(address = ADDRESS): TestWalletState {
  return {
    name: "Test wallet",
    address,
    accountId: `cosmos:zerone-1:${address}`,
    balanceUzrn: "0",
    incomingFeeGrants: null,
  };
}

function signature(): TestWalletSignature {
  return {
    pub_key: {
      type: "tendermint/PubKeySecp256k1",
      value: Buffer.alloc(33, 2).toString("base64"),
    },
    signature: Buffer.alloc(64, 3).toString("base64"),
  };
}

function installWindow(keplr: Record<string, unknown>): void {
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      location: { origin: "https://dashboard.invalid" },
      keplr,
      getOfflineSigner: () => ({}),
    },
    writable: true,
  });
}

describe("Pi callback fragment parsing", () => {
  it("accepts one bounded bearer response and returns only the required fields", () => {
    assert.deepEqual(
      parsePiCallbackFragment(
        `#access_token=${ACCESS_TOKEN}&token_type=Bearer&expires_in=3600&state=${STATE}`,
      ),
      {
        kind: "success",
        accessToken: ACCESS_TOKEN,
        state: STATE,
      },
    );
  });

  it("reduces an OAuth denial to a non-sensitive error code", () => {
    assert.deepEqual(
      parsePiCallbackFragment(
        `#error=access_denied&error_description=User%20declined&state=${STATE}`,
      ),
      { kind: "error", errorCode: "access_denied" },
    );
  });

  it("rejects duplicate, mixed, unknown, malformed, and incomplete fields", () => {
    const invalid = [
      `#access_token=${ACCESS_TOKEN}&access_token=second&state=${STATE}`,
      `#access_token=${ACCESS_TOKEN}&state=${STATE}&state=second`,
      `#access_token=${ACCESS_TOKEN}&state=${STATE}&scope=username`,
      `#access_token=${ACCESS_TOKEN}&error=denied&state=${STATE}`,
      `#access_token=${ACCESS_TOKEN}&token_type=bearer&state=${STATE}`,
      `#access_token=bad%ZZtoken&state=${STATE}`,
      `#access_token=${ACCESS_TOKEN}`,
      `#state=${STATE}`,
      "",
    ];
    invalid.forEach((fragment) => {
      assert.throws(
        () => parsePiCallbackFragment(fragment),
        PiFragmentError,
      );
    });
  });

  it("never includes a rejected bearer value in parser errors", () => {
    const bearer = "private-token-that-must-not-appear";
    try {
      parsePiCallbackFragment(
        `#access_token=${bearer}&access_token=duplicate&state=${STATE}`,
      );
      assert.fail("expected duplicate bearer rejection");
    } catch (error) {
      assert.ok(error instanceof PiFragmentError);
      assert.doesNotMatch(error.message, new RegExp(bearer));
    }
  });
});

describe("Pi callback page hardening", () => {
  it("erases the fragment in the first executable head action", () => {
    const html = readFileSync(CALLBACK_HTML_PATH, "utf8");
    const scriptMatch = html.match(/<script>([^<]+)<\/script>/);
    assert.ok(scriptMatch);
    const bootstrap = scriptMatch[1];
    assert.ok(bootstrap);

    const bootstrapIndex = html.indexOf(`<script>${bootstrap}</script>`);
    assert.equal(bootstrapIndex, html.indexOf("<script>"));
    assert.ok(bootstrapIndex < html.indexOf("<link rel=\"stylesheet\""));
    assert.ok(bootstrapIndex < html.indexOf("<body>"));
    assert.ok(
      bootstrap.indexOf("location.hash") <
        bootstrap.indexOf("history.replaceState"),
    );
    assert.ok(
      bootstrap.indexOf("history.replaceState") <
        bootstrap.indexOf("Object.defineProperty"),
    );
    assert.match(bootstrap, /__takeZeronePiCallbackFragment/);
    assert.doesNotMatch(
      bootstrap,
      /console|localStorage|sessionStorage|document\.|innerHTML/,
    );
  });

  it("pins the exact bootstrap hash in both enforced CSP policies", () => {
    const html = readFileSync(CALLBACK_HTML_PATH, "utf8");
    const headers = readFileSync(HEADERS_PATH, "utf8");
    const scriptMatch = html.match(/<script>([^<]+)<\/script>/);
    assert.ok(scriptMatch?.[1]);
    const hash = createHash("sha256")
      .update(scriptMatch[1])
      .digest("base64");
    const directive = `'sha256-${hash}'`;

    assert.equal(headers.split(directive).length - 1, 2);
    const callbackStart = headers.indexOf("/pi/callback/*");
    const callbackEnd = headers.indexOf("\n/assets/*", callbackStart);
    const callbackHeaders = headers.slice(callbackStart, callbackEnd);
    assert.match(callbackHeaders, /Cache-Control: no-store, max-age=0, no-transform/);
    assert.match(callbackHeaders, /Referrer-Policy: no-referrer/);
    assert.match(
      callbackHeaders,
      /default-src 'none'; script-src 'self' 'sha256-/,
    );
    assert.match(callbackHeaders, /connect-src 'self'/);
    assert.doesNotMatch(callbackHeaders, /cloudflare|unsafe-inline/);
  });

  it("refuses automatic HTML transformations without weakening cache policy", () => {
    const headers = readFileSync(HEADERS_PATH, "utf8");
    const rootStart = headers.indexOf("\n/\n");
    const rootEnd = headers.indexOf("\n/index.html", rootStart);
    const rootHeaders = headers.slice(rootStart, rootEnd);
    const indexStart = rootEnd;
    const indexEnd = headers.indexOf("\n/pi/callback/*", indexStart);
    const indexHeaders = headers.slice(indexStart, indexEnd);
    const assetsStart = headers.indexOf("/assets/*");
    const assetHeaders = headers.slice(assetsStart);

    assert.match(
      rootHeaders,
      /Cache-Control: public, max-age=0, must-revalidate, no-transform/,
    );
    assert.match(
      indexHeaders,
      /Cache-Control: public, max-age=0, must-revalidate, no-transform/,
    );
    assert.match(
      assetHeaders,
      /Cache-Control: public, max-age=31536000, immutable/,
    );
    assert.doesNotMatch(assetHeaders, /no-transform/);
  });

  it("keeps bearer material out of storage, logging, and rendered copy", () => {
    const html = readFileSync(CALLBACK_HTML_PATH, "utf8");
    const source = readFileSync(CALLBACK_SOURCE_PATH, "utf8");
    const transport = readFileSync(PI_TRANSPORT_PATH, "utf8");
    assert.doesNotMatch(html, /https?:\/\//);
    assert.doesNotMatch(
      `${source}\n${transport}`,
      /console\.|localStorage|sessionStorage|innerHTML/,
    );
    assert.match(source, /delete window\.__takeZeronePiCallbackFragment/);
    assert.doesNotMatch(source, /:\s*window\.location\.hash/);
    assert.doesNotMatch(source, /^\s*import\s/m);
    assert.match(transport, /url\.origin !== window\.location\.origin/);
    assert.match(transport, /credentials: "same-origin"/);
    assert.match(transport, /redirect: "error"/);
    assert.match(transport, /piRequest\("\/api\/pi\/session"/);
    assert.doesNotMatch(transport, /accessToken.*(?:searchParams|localStorage)/);
  });

  it("keeps both pilot phases independent and default-off", () => {
    const source = readFileSync(MAIN_SOURCE_PATH, "utf8");
    const html = readFileSync(MAIN_HTML_PATH, "utf8");
    assert.match(
      source,
      /VITE_PI_PILOT_ENABLED === "true"/,
    );
    assert.match(
      source,
      /VITE_PI_WALLET_PROOF_ENABLED === "true"/,
    );
    assert.match(source, /if \(!PI_PILOT_ENABLED\) return/);
    assert.match(source, /const piPilotReady = initialisePiPilotIfEnabled\(\)/);
    assert.match(source, /window\.location\.hash !== "#contribute" \|\| piPilotSection\.hidden/);
    assert.match(source, /piPilotReady\.then\(alignPiHash\)/);
    assert.match(
      source,
      /initialiseConstructiveTree\(constructiveTreeRoot\)/,
    );
    assert.match(source, /initialiseQuantumSeason\(quantumSeasonRoot\)/);
    assert.match(source, /initialiseMathFrontier\(mathFrontierRoot\)/);
    assert.match(source, /initialiseLifeGarden\(lifeGardenRoot\)/);
    assert.match(source, /initialNetworkReady = refreshNetwork\(false\)/);
    assert.ok(html.indexOf('id="skills"') < html.indexOf('id="life"'));
    assert.ok(html.indexOf('id="life"') < html.indexOf('id="contribute"'));
    assert.match(html, /href="#skills"[^>]*>Browse without signing in/);
    assert.match(html, /<span>08<\/span> Optional account pilot/);
  });

  it("requires informed confirmation before subject-linked pilot deletion", () => {
    const source = readFileSync(PI_TRANSPORT_PATH, "utf8");
    const ui = readFileSync(resolve(DASHBOARD_ROOT, "src/pi-ui.ts"), "utf8");
    const html = readFileSync(MAIN_HTML_PATH, "utf8");
    assert.match(source, /piRequest\("\/api\/pi\/data"/u);
    assert.match(source, /confirmation: "delete-pi-pilot-data-v1"/u);
    assert.match(ui, /deletePiPilotData\(session\.csrfToken\)/u);
    assert.match(html, /id="pi-data-dialog"/u);
    assert.match(html, /id="pi-data-check" type="checkbox" required/u);
    assert.match(html, /subject guard blocks older in-flight authorization[\s\S]*?row may remain until operator cleanup/u);
    assert.match(html, /Promoted markers[\s\S]*?may be retained indefinitely/u);
    assert.match(html, /30 days on a[\s\S]*?Workers Paid plan[\s\S]*?seven days on a Workers Free plan/u);
    assert.match(html, /currently authenticated app-specific/u);
    assert.match(html, /not my Pi account/u);
    assert.match(html, /does not revoke or alter Pi, Keplr/u);
    assert.match(html, /id="pi-data-title" tabindex="-1"/u);
    assert.match(html, /id="pi-data-status" role="status" aria-live="polite"/u);
    assert.match(ui, /dataForm\.setAttribute\("aria-busy", "true"\)/u);
    assert.match(ui, /dataTitle\.focus\(\)/u);
    assert.match(ui, /dataDialog\.close\(\);\s*consentOpenButton\.focus\(\)/u);
    assert.doesNotMatch(ui, /cancel[\s\S]{0,100}preventDefault/u);
  });
});

describe("Keplr text-only Zerone wallet proof", () => {
  it("rechecks the exact address before and after signArbitrary", async () => {
    const calls: string[] = [];
    installWindow({
      async enable() {
        calls.push("enable");
      },
      async getKey() {
        calls.push("getKey");
        return { name: "Test wallet", bech32Address: ADDRESS };
      },
      async signArbitrary(
        chainId: string,
        signer: string,
        message: string,
      ) {
        calls.push(`sign:${chainId}:${signer}:${message}`);
        return signature();
      },
    });
    const { signWalletControlProof } = await loadWalletProofModule();

    assert.deepEqual(
      await signWalletControlProof(wallet(), "bounded challenge"),
      signature(),
    );
    assert.deepEqual(calls, [
      "enable",
      "getKey",
      `sign:zerone-1:${ADDRESS}:bounded challenge`,
      "getKey",
    ]);
  });

  it("rejects a Keplr account change after signing", async () => {
    let keyReads = 0;
    installWindow({
      async enable() {},
      async getKey() {
        keyReads += 1;
        return {
          name: "Test wallet",
          bech32Address: keyReads === 1 ? ADDRESS : OTHER_ADDRESS,
        };
      },
      async signArbitrary() {
        return signature();
      },
    });
    const { signWalletControlProof } = await loadWalletProofModule();

    await assert.rejects(
      () => signWalletControlProof(wallet(), "bounded challenge"),
      /Keplr changed accounts/,
    );
  });

  it("has no transaction fallback when arbitrary signing is unavailable", async () => {
    let transactionFallbackCalled = false;
    installWindow({
      async enable() {},
      async getKey() {
        return { name: "Test wallet", bech32Address: ADDRESS };
      },
      async signAndBroadcast() {
        transactionFallbackCalled = true;
      },
    });
    const { signWalletControlProof } = await loadWalletProofModule();

    await assert.rejects(
      () => signWalletControlProof(wallet(), "bounded challenge"),
      /No transaction fallback/,
    );
    assert.equal(transactionFallbackCalled, false);
  });
});
