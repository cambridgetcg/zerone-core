import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { DatabaseSync } from "node:sqlite";
import test, { describe, it } from "node:test";

import {
  encodeSecp256k1Pubkey,
  encodeSecp256k1Signature,
  pubkeyToAddress,
} from "@cosmjs/amino";
import {
  Secp256k1,
  sha256,
} from "@cosmjs/crypto";
import { toBech32 } from "@cosmjs/encoding";

import {
  hashOpaque,
  keyedHash,
} from "../functions/api/pi/_crypto";
import {
  handlePiRequest,
  runPiEndpoint,
} from "../functions/api/pi/_service";
import { D1PiRepository } from "../functions/api/pi/_store";
import {
  adr36SignBytes,
  parsePiStdSignature,
} from "../functions/api/pi/_wallet-proof";
import type {
  PiBinding,
  PiChallenge,
  PiD1Database,
  PiD1PreparedStatement,
  PiD1Result,
  PiEnv,
  PiRepository,
  PiRuntime,
  PiSession,
  PiSessionEnvelope,
  PiStdSignature,
} from "../functions/api/pi/_types";

const ORIGIN = "https://zerone.ai";
const ENV: PiEnv = {
  PI_CLIENT_ID: "pi-client-id",
  PI_PILOT_ENABLED: "true",
  PI_PUBLIC_ORIGIN: ORIGIN,
  PI_SUBJECT_PEPPER: "p".repeat(64),
  PI_WALLET_PROOF_ENABLED: "true",
};
const ACCESS_TOKEN = "pi-access-token-123456789";
const RAW_UID = "pi-app-scoped-user-123";

interface OAuthFlow {
  readonly browserHash: string;
  readonly createdAt: number;
  readonly expiresAt: number;
  consumed: boolean;
}

class MemoryPiRepository implements PiRepository {
  readonly oauthFlows = new Map<string, OAuthFlow>();
  readonly bearerFingerprints = new Set<string>();
  readonly sessions = new Map<string, PiSession>();
  readonly challenges = new Map<string, PiChallenge>();
  readonly challengeUses = new Set<string>();
  readonly bindings = new Map<string, PiBinding>();
  readonly addressSubjects = new Map<string, string>();
  readonly revoked = new Set<string>();

  async createOAuthFlow(
    stateHash: string,
    browserHash: string,
    createdAt: number,
    expiresAt: number,
  ): Promise<void> {
    if (this.oauthFlows.has(stateHash)) throw new Error("duplicate state");
    this.oauthFlows.set(stateHash, {
      browserHash,
      createdAt,
      expiresAt,
      consumed: false,
    });
  }

  async consumeOAuthFlow(
    stateHash: string,
    browserHash: string,
    bearerFingerprint: string,
    now: number,
  ): Promise<boolean> {
    const flow = this.oauthFlows.get(stateHash);
    if (
      !flow ||
      flow.browserHash !== browserHash ||
      flow.consumed ||
      flow.expiresAt <= now ||
      this.bearerFingerprints.has(bearerFingerprint)
    ) {
      return false;
    }
    flow.consumed = true;
    this.bearerFingerprints.add(bearerFingerprint);
    return true;
  }

  async createSession(session: PiSession): Promise<void> {
    if (this.sessions.has(session.tokenHash)) throw new Error("duplicate session");
    this.sessions.set(session.tokenHash, session);
  }

  async getSession(tokenHash: string, now: number): Promise<PiSession | null> {
    const session = this.sessions.get(tokenHash);
    return session && session.expiresAt > now && !this.revoked.has(tokenHash)
      ? session
      : null;
  }

  async revokeSession(tokenHash: string): Promise<void> {
    this.revoked.add(tokenHash);
  }

  async createChallenge(
    challenge: PiChallenge,
    recentSince: number,
    maximumRecent: number,
  ): Promise<boolean> {
    const activeSession = await this.getSession(
      challenge.sessionHash,
      challenge.createdAt,
    );
    const recent = [...this.challenges.values()].filter(
      (candidate) =>
        candidate.sessionHash === challenge.sessionHash &&
        candidate.createdAt >= recentSince,
    ).length;
    if (
      recent >= maximumRecent ||
      this.challenges.has(challenge.idHash) ||
      activeSession?.subjectHash !== challenge.subjectHash ||
      this.bindings.has(challenge.subjectHash)
    ) {
      return false;
    }
    this.challenges.set(challenge.idHash, challenge);
    return true;
  }

  async getChallenge(
    idHash: string,
    sessionHash: string,
    subjectHash: string,
    now: number,
  ): Promise<PiChallenge | null> {
    const challenge = this.challenges.get(idHash);
    const session = await this.getSession(sessionHash, now);
    return challenge &&
      session &&
      challenge.sessionHash === sessionHash &&
      challenge.subjectHash === subjectHash &&
      challenge.expiresAt > now &&
      !this.challengeUses.has(idHash)
      ? challenge
      : null;
  }

  async bindChallenge(
    idHash: string,
    sessionHash: string,
    subjectHash: string,
    proofHash: string,
    consentVersion: string,
    now: number,
  ): Promise<PiBinding | null> {
    const challenge = await this.getChallenge(
      idHash,
      sessionHash,
      subjectHash,
      now,
    );
    if (
      !challenge ||
      this.bindings.has(subjectHash) ||
      this.addressSubjects.has(challenge.address)
    ) {
      return null;
    }
    const binding: PiBinding = {
      subjectHash,
      address: challenge.address,
      accountId: challenge.accountId,
      consentVersion,
      boundAt: now,
    };
    this.bindings.set(subjectHash, binding);
    this.addressSubjects.set(challenge.address, subjectHash);
    for (const candidate of [...this.challenges.values()]) {
      if (candidate.subjectHash !== subjectHash) continue;
      this.challengeUses.add(candidate.idHash);
      this.challenges.delete(candidate.idHash);
    }
    assert.equal(proofHash.length, 43);
    return binding;
  }

  async getBinding(subjectHash: string): Promise<PiBinding | null> {
    return this.bindings.get(subjectHash) ?? null;
  }

  async deleteBinding(subjectHash: string): Promise<void> {
    const binding = this.bindings.get(subjectHash);
    if (binding) this.addressSubjects.delete(binding.address);
    this.bindings.delete(subjectHash);
    for (const candidate of [...this.challenges.values()]) {
      if (candidate.subjectHash !== subjectHash) continue;
      this.challengeUses.add(candidate.idHash);
      this.challenges.delete(candidate.idHash);
    }
  }

  async deleteSubject(subjectHash: string): Promise<void> {
    const binding = this.bindings.get(subjectHash);
    if (binding) this.addressSubjects.delete(binding.address);
    this.bindings.delete(subjectHash);
    for (const [tokenHash, session] of this.sessions) {
      if (session.subjectHash !== subjectHash) continue;
      this.sessions.delete(tokenHash);
      this.revoked.delete(tokenHash);
    }
    for (const candidate of [...this.challenges.values()]) {
      if (candidate.subjectHash !== subjectHash) continue;
      this.challengeUses.delete(candidate.idHash);
      this.challenges.delete(candidate.idHash);
    }
  }
}

interface FetchCall {
  readonly input: string | URL | Request;
  readonly init?: RequestInit;
}

interface Harness {
  readonly repository: MemoryPiRepository;
  readonly calls: FetchCall[];
  readonly runtime: PiRuntime;
  setNow(value: number): void;
  setFetch(
    implementation: (input: string | URL | Request, init?: RequestInit) => Promise<Response>,
  ): void;
}

function harness(now = 2_000_000): Harness {
  const repository = new MemoryPiRepository();
  const calls: FetchCall[] = [];
  let currentNow = now;
  let randomCounter = 0;
  let fetchImplementation: PiRuntime["fetch"] = async (): Promise<Response> =>
    Response.json(
      {
        uid: RAW_UID,
        username: "pioneer",
        credentials: {
          scopes: ["username"],
          valid_until: {
            timestamp: now + 60_000,
            iso8601: new Date(now + 60_000).toISOString(),
          },
        },
      },
    );
  const runtime: PiRuntime = {
    repository,
    now: () => currentNow,
    randomBytes(length) {
      randomCounter += 1;
      return new Uint8Array(length).fill(randomCounter);
    },
    async fetch(input, init) {
      calls.push({ input, init });
      return fetchImplementation(input, init);
    },
  };
  return {
    repository,
    calls,
    runtime,
    setNow(value) {
      currentNow = value;
    },
    setFetch(implementation) {
      fetchImplementation = implementation;
    },
  };
}

function cookieValue(response: Response, name: string): string {
  const header = response.headers.get("Set-Cookie") ?? "";
  const match = new RegExp(`${name}=([^;,]+)`, "u").exec(header);
  if (!match?.[1]) assert.fail(`Missing ${name} cookie in ${header}`);
  return match[1];
}

interface OAuthStart {
  readonly state: string;
  readonly browserCookie: string;
}

async function startOAuth(testHarness: Harness): Promise<OAuthStart> {
  const response = await handlePiRequest(
    "authorize",
    new Request(`${ORIGIN}/api/pi/authorize`),
    ENV,
    testHarness.runtime,
  );
  assert.equal(response.status, 302);
  const location = response.headers.get("Location");
  if (!location) assert.fail("authorize response did not include Location");
  const target = new URL(location);
  return {
    state: target.searchParams.get("state") ?? "",
    browserCookie: cookieValue(response, "__Host-zrn-pi-oauth"),
  };
}

function sessionRequest(
  flow: OAuthStart,
  accessToken = ACCESS_TOKEN,
  extraHeaders?: HeadersInit,
): Request {
  return new Request(`${ORIGIN}/api/pi/session`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Cookie: `__Host-zrn-pi-oauth=${flow.browserCookie}`,
      Origin: ORIGIN,
      "Sec-Fetch-Site": "same-origin",
      ...Object.fromEntries(new Headers(extraHeaders)),
    },
    body: JSON.stringify({ accessToken, state: flow.state }),
  });
}

interface SignedIn {
  readonly response: Response;
  readonly sessionCookie: string;
  readonly envelope: PiSessionEnvelope;
}

async function signIn(testHarness: Harness): Promise<SignedIn> {
  const flow = await startOAuth(testHarness);
  const response = await handlePiRequest(
    "session",
    sessionRequest(flow),
    ENV,
    testHarness.runtime,
  );
  const envelope = (await response.clone().json()) as PiSessionEnvelope;
  return {
    response,
    sessionCookie: cookieValue(response, "__Host-zrn-pi-session"),
    envelope,
  };
}

function authenticatedRequest(
  path: string,
  method: "POST" | "DELETE",
  session: SignedIn,
  body: JsonRecord = {},
  overrides?: HeadersInit,
): Request {
  return new Request(`${ORIGIN}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      Cookie: `__Host-zrn-pi-session=${session.sessionCookie}`,
      Origin: ORIGIN,
      "Sec-Fetch-Site": "same-origin",
      "X-Zerone-CSRF": session.envelope.csrfToken ?? "",
      ...Object.fromEntries(new Headers(overrides)),
    },
    body: JSON.stringify(body),
  });
}

function deterministicWallet(privateKeyTail = 1): {
  readonly address: string;
  readonly sign: (message: string) => PiStdSignature;
} {
  const privateKey = new Uint8Array(32);
  privateKey[31] = privateKeyTail;
  const publicKey = Secp256k1.compressPubkey(
    Secp256k1.makeKeypair(privateKey).pubkey,
  );
  const address = pubkeyToAddress(
    encodeSecp256k1Pubkey(publicKey),
    "zrn",
  );
  return {
    address,
    sign(message) {
      const signature = Secp256k1.createSignature(
        sha256(adr36SignBytes(address, message)),
        privateKey,
      );
      const fixedLength = new Uint8Array(64);
      fixedLength.set(signature.r(32));
      fixedLength.set(signature.s(32), 32);
      const parsed = parsePiStdSignature(
        encodeSecp256k1Signature(publicKey, fixedLength),
      );
      if (!parsed) assert.fail("deterministic signature was not accepted");
      return parsed;
    },
  };
}

type JsonRecord = Record<string, unknown>;

test("disabled edge returns its envelope without requiring config or D1", async () => {
  const response = await runPiEndpoint("me", {
    request: new Request(`${ORIGIN}/api/pi/me`),
    env: {},
  });

  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), {
    enabled: false,
    walletProofEnabled: false,
    authenticated: false,
  });
  assert.equal(response.headers.get("Cache-Control"), "no-store");
  assert.equal(response.headers.get("Access-Control-Allow-Origin"), null);

  const authorizeResponse = await runPiEndpoint("authorize", {
    request: new Request(`${ORIGIN}/api/pi/authorize`),
    env: {},
  });
  assert.equal(authorizeResponse.status, 404);
});

test("authorize uses fixed Pi OAuth parameters and a distinct browser transaction", async () => {
  const testHarness = harness();
  const response = await handlePiRequest(
    "authorize",
    new Request(`${ORIGIN}/api/pi/authorize`),
    ENV,
    testHarness.runtime,
  );

  assert.equal(response.status, 302);
  const target = new URL(response.headers.get("Location") ?? "");
  assert.equal(target.origin, "https://accounts.pinet.com");
  assert.equal(target.pathname, "/oauth/authorize");
  assert.deepEqual(
    [...target.searchParams.keys()].sort(),
    ["client_id", "redirect_uri", "response_type", "scope", "state"].sort(),
  );
  assert.equal(target.searchParams.get("response_type"), "token");
  assert.equal(target.searchParams.get("client_id"), "pi-client-id");
  assert.equal(target.searchParams.get("redirect_uri"), `${ORIGIN}/pi/callback/`);
  assert.equal(target.searchParams.get("scope"), "username");
  assert.equal(target.searchParams.has("payments"), false);
  assert.equal(target.searchParams.has("wallet_address"), false);

  const state = target.searchParams.get("state") ?? "";
  const browser = cookieValue(response, "__Host-zrn-pi-oauth");
  assert.notEqual(state, browser);
  assert.match(response.headers.get("Set-Cookie") ?? "", /Secure/u);
  assert.match(response.headers.get("Set-Cookie") ?? "", /HttpOnly/u);
  assert.match(response.headers.get("Set-Cookie") ?? "", /SameSite=Lax/u);
  assert.equal(testHarness.repository.oauthFlows.get(hashOpaque(state))?.browserHash, hashOpaque(browser));
  assert.equal(response.headers.get("Access-Control-Allow-Origin"), null);
});

test("session atomically consumes state, browser transaction, and bearer before fixed /v2/me", async () => {
  const testHarness = harness();
  const flow = await startOAuth(testHarness);
  const request = sessionRequest(flow);
  const response = await handlePiRequest(
    "session",
    request,
    ENV,
    testHarness.runtime,
  );

  assert.equal(response.status, 200);
  assert.equal(testHarness.calls.length, 1);
  const call = testHarness.calls[0];
  assert.equal(String(call?.input), "https://api.minepi.com/v2/me");
  assert.equal(call?.init?.method, "GET");
  assert.equal(call?.init?.redirect, "manual");
  assert.ok(call?.init?.signal);
  const upstreamHeaders = new Headers(call?.init?.headers);
  assert.equal(upstreamHeaders.get("Authorization"), `Bearer ${ACCESS_TOKEN}`);
  assert.equal(upstreamHeaders.get("Accept"), "application/json");

  const bodyText = await response.clone().text();
  assert.equal(bodyText.includes(ACCESS_TOKEN), false);
  assert.equal(bodyText.includes(RAW_UID), false);
  const body = JSON.parse(bodyText) as PiSessionEnvelope;
  assert.equal(body.authenticated, true);
  assert.equal(body.username, "pioneer");
  assert.match(body.csrfToken ?? "", /^[A-Za-z0-9_-]{43}$/u);
  const sessionCookie = cookieValue(response, "__Host-zrn-pi-session");
  assert.notEqual(sessionCookie, ACCESS_TOKEN);
  assert.match(response.headers.get("Set-Cookie") ?? "", /Secure/u);
  assert.match(response.headers.get("Set-Cookie") ?? "", /HttpOnly/u);
  assert.equal(testHarness.repository.bearerFingerprints.size, 1);
  assert.equal(testHarness.repository.bearerFingerprints.has(ACCESS_TOKEN), false);
  const persisted = JSON.stringify([
    ...testHarness.repository.sessions.values(),
    ...testHarness.repository.bearerFingerprints,
  ]);
  assert.equal(persisted.includes(ACCESS_TOKEN), false);
  assert.equal(persisted.includes(RAW_UID), false);

  const replay = await handlePiRequest(
    "session",
    sessionRequest(flow),
    ENV,
    testHarness.runtime,
  );
  assert.equal(replay.status, 400);
  assert.equal(testHarness.calls.length, 1);

  const freshFlow = await startOAuth(testHarness);
  const bearerReplay = await handlePiRequest(
    "session",
    sessionRequest(freshFlow),
    ENV,
    testHarness.runtime,
  );
  assert.equal(bearerReplay.status, 400);
  assert.equal(testHarness.calls.length, 1);
});

test("session rejects a mismatched browser transaction and never calls Pi", async () => {
  const testHarness = harness();
  const first = await startOAuth(testHarness);
  const second = await startOAuth(testHarness);
  const response = await handlePiRequest(
    "session",
    sessionRequest({
      state: first.state,
      browserCookie: second.browserCookie,
    }),
    ENV,
    testHarness.runtime,
  );

  assert.equal(response.status, 400);
  assert.equal(testHarness.calls.length, 0);
  assert.equal(
    testHarness.repository.oauthFlows.get(hashOpaque(first.state))?.consumed,
    false,
  );
});

test("session rejects expired OAuth state before calling Pi", async () => {
  const testHarness = harness();
  const flow = await startOAuth(testHarness);
  testHarness.setNow(testHarness.runtime.now() + 10 * 60 * 1_000);

  const response = await handlePiRequest(
    "session",
    sessionRequest(flow),
    ENV,
    testHarness.runtime,
  );

  assert.equal(response.status, 400);
  assert.equal(testHarness.calls.length, 0);
  assert.equal(
    testHarness.repository.oauthFlows.get(hashOpaque(flow.state))?.consumed,
    false,
  );
});

test("Pi upstream redirects and oversized responses fail closed", async () => {
  const oversizedRequestHarness = harness();
  const oversizedRequestFlow = await startOAuth(oversizedRequestHarness);
  let streamedBytes = 0;
  let streamCancelled = false;
  const oversizedRequestBody = new ReadableStream<Uint8Array>({
    pull(controller) {
      const remaining = 8_193 - streamedBytes;
      if (remaining <= 0) {
        controller.close();
        return;
      }
      const chunkLength = Math.min(4_096, remaining);
      streamedBytes += chunkLength;
      controller.enqueue(new Uint8Array(chunkLength).fill(0x78));
    },
    cancel() {
      streamCancelled = true;
    },
  });
  const oversizedRequest = new Request(`${ORIGIN}/api/pi/session`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Cookie: `__Host-zrn-pi-oauth=${oversizedRequestFlow.browserCookie}`,
      Origin: ORIGIN,
      "Sec-Fetch-Site": "same-origin",
    },
    body: oversizedRequestBody,
    duplex: "half",
  } as RequestInit & { duplex: "half" });
  const oversizedRequestResponse = await handlePiRequest(
    "session",
    oversizedRequest,
    ENV,
    oversizedRequestHarness.runtime,
  );
  assert.equal(oversizedRequestResponse.status, 400);
  assert.equal(streamCancelled, true);
  assert.equal(streamedBytes, 8_193);
  assert.equal(oversizedRequestHarness.calls.length, 0);
  assert.equal(
    oversizedRequestHarness.repository.oauthFlows.get(
      hashOpaque(oversizedRequestFlow.state),
    )?.consumed,
    false,
  );

  const redirectHarness = harness();
  redirectHarness.setFetch(async () =>
    new Response(null, {
      status: 302,
      headers: { Location: "https://attacker.invalid/token" },
    }),
  );
  const redirectFlow = await startOAuth(redirectHarness);
  const redirectResponse = await handlePiRequest(
    "session",
    sessionRequest(redirectFlow),
    ENV,
    redirectHarness.runtime,
  );
  assert.equal(redirectResponse.status, 502);
  assert.equal(redirectHarness.calls[0]?.init?.redirect, "manual");

  const oversizedHarness = harness();
  oversizedHarness.setFetch(async () =>
    new Response("x".repeat(17_000), {
      headers: { "Content-Type": "application/json" },
    }),
  );
  const oversizedFlow = await startOAuth(oversizedHarness);
  const oversizedResponse = await handlePiRequest(
    "session",
    sessionRequest(oversizedFlow, "different-access-token-1234"),
    ENV,
    oversizedHarness.runtime,
  );
  assert.equal(oversizedResponse.status, 502);
  assert.equal(oversizedResponse.headers.get("Cache-Control"), "no-store");

  const broadScopeHarness = harness();
  broadScopeHarness.setFetch(async () =>
    Response.json({
      uid: RAW_UID,
      username: "pioneer",
      credentials: {
        scopes: ["username", "wallet_address"],
      },
    }),
  );
  const broadScopeFlow = await startOAuth(broadScopeHarness);
  const broadScopeResponse = await handlePiRequest(
    "session",
    sessionRequest(broadScopeFlow, "broad-scope-access-token"),
    ENV,
    broadScopeHarness.runtime,
  );
  assert.equal(broadScopeResponse.status, 502);
  assert.equal(broadScopeHarness.repository.sessions.size, 0);

  const walletFieldHarness = harness();
  walletFieldHarness.setFetch(async () =>
    Response.json({
      uid: RAW_UID,
      username: "pioneer",
      wallet_address: "GFAKEPIWALLETADDRESS",
    }),
  );
  const walletFieldFlow = await startOAuth(walletFieldHarness);
  const walletFieldResponse = await handlePiRequest(
    "session",
    sessionRequest(walletFieldFlow, "wallet-field-access-token"),
    ENV,
    walletFieldHarness.runtime,
  );
  assert.equal(walletFieldResponse.status, 502);
  assert.equal(walletFieldHarness.repository.sessions.size, 0);

  const benignFieldHarness = harness();
  benignFieldHarness.setFetch(async () =>
    Response.json({
      uid: RAW_UID,
      username: "pioneer",
      api_version: "future-compatible-metadata",
    }),
  );
  const benignFieldFlow = await startOAuth(benignFieldHarness);
  const benignFieldResponse = await handlePiRequest(
    "session",
    sessionRequest(benignFieldFlow, "benign-field-access-token"),
    ENV,
    benignFieldHarness.runtime,
  );
  assert.equal(benignFieldResponse.status, 200);
  assert.equal(benignFieldHarness.repository.sessions.size, 1);
});

test("me exposes sanitized state and mutations require exact Origin plus session CSRF", async () => {
  const testHarness = harness();
  const signedIn = await signIn(testHarness);
  assert.equal(signedIn.response.status, 200);

  const me = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: {
        Cookie: `__Host-zrn-pi-session=${signedIn.sessionCookie}`,
      },
    }),
    ENV,
    testHarness.runtime,
  );
  const meBody = (await me.json()) as PiSessionEnvelope;
  assert.deepEqual(meBody, signedIn.envelope);
  assert.equal(JSON.stringify(meBody).includes(RAW_UID), false);

  const badOrigin = await handlePiRequest(
    "logout",
    authenticatedRequest("/api/pi/logout", "POST", signedIn, {}, {
      Origin: "https://attacker.invalid",
    }),
    ENV,
    testHarness.runtime,
  );
  assert.equal(badOrigin.status, 401);

  const badCsrf = await handlePiRequest(
    "logout",
    authenticatedRequest("/api/pi/logout", "POST", signedIn, {}, {
      "X-Zerone-CSRF": "A".repeat(43),
    }),
    ENV,
    testHarness.runtime,
  );
  assert.equal(badCsrf.status, 401);

  const logout = await handlePiRequest(
    "logout",
    authenticatedRequest("/api/pi/logout", "POST", signedIn),
    ENV,
    testHarness.runtime,
  );
  assert.equal(logout.status, 200);
  assert.deepEqual(await logout.json(), {
    enabled: true,
    walletProofEnabled: true,
    authenticated: false,
  });
  assert.match(logout.headers.get("Set-Cookie") ?? "", /Max-Age=0/u);
});

test("data deletion requires explicit confirmation and atomically removes subject-linked pilot state", async () => {
  const testHarness = harness();
  const signedIn = await signIn(testHarness);
  const sessionHash = hashOpaque(signedIn.sessionCookie);
  const session = testHarness.repository.sessions.get(sessionHash);
  if (!session) assert.fail("expected authenticated session");

  const secondTokenHash = hashOpaque("second-session-for-subject");
  await testHarness.repository.createSession({
    ...session,
    tokenHash: secondTokenHash,
  });
  const address = toBech32("zrn", new Uint8Array(20).fill(8));
  testHarness.repository.bindings.set(session.subjectHash, {
    subjectHash: session.subjectHash,
    address,
    accountId: `cosmos:zerone-1:${address}`,
    consentVersion: "pi-wallet-link-v1",
    boundAt: testHarness.runtime.now(),
  });
  testHarness.repository.addressSubjects.set(address, session.subjectHash);
  const outstandingChallenge: PiChallenge = {
    idHash: hashOpaque("privacy-delete-challenge"),
    sessionHash,
    subjectHash: session.subjectHash,
    address,
    accountId: `cosmos:zerone-1:${address}`,
    message: "privacy deletion challenge",
    createdAt: testHarness.runtime.now(),
    expiresAt: testHarness.runtime.now() + 60_000,
  };
  testHarness.repository.challenges.set(
    outstandingChallenge.idHash,
    outstandingChallenge,
  );

  const missingConfirmation = await handlePiRequest(
    "data",
    authenticatedRequest("/api/pi/data", "DELETE", signedIn),
    ENV,
    testHarness.runtime,
  );
  assert.equal(missingConfirmation.status, 400);
  assert.equal(testHarness.repository.sessions.has(sessionHash), true);

  const badOrigin = await handlePiRequest(
    "data",
    authenticatedRequest(
      "/api/pi/data",
      "DELETE",
      signedIn,
      { confirmation: "delete-pi-pilot-data-v1" },
      { Origin: "https://attacker.invalid" },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(badOrigin.status, 401);
  assert.equal(testHarness.repository.sessions.has(sessionHash), true);

  const deleted = await handlePiRequest(
    "data",
    authenticatedRequest(
      "/api/pi/data",
      "DELETE",
      signedIn,
      { confirmation: "delete-pi-pilot-data-v1" },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(deleted.status, 200);
  assert.deepEqual(await deleted.json(), {
    enabled: true,
    walletProofEnabled: true,
    authenticated: false,
  });
  const clearedCookies = deleted.headers.get("Set-Cookie") ?? "";
  assert.match(clearedCookies, /__Host-zrn-pi-session=.*Max-Age=0/u);
  assert.match(clearedCookies, /__Host-zrn-pi-oauth=.*Max-Age=0/u);
  assert.equal(
    [...testHarness.repository.sessions.values()].some(
      (candidate) => candidate.subjectHash === session.subjectHash,
    ),
    false,
  );
  assert.equal(testHarness.repository.bindings.has(session.subjectHash), false);
  assert.equal(testHarness.repository.addressSubjects.has(address), false);
  assert.equal(
    [...testHarness.repository.challenges.values()].some(
      (candidate) => candidate.subjectHash === session.subjectHash,
    ),
    false,
  );
  assert.equal(testHarness.repository.sessions.has(secondTokenHash), false);

  const staleSession = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: {
        Cookie: `__Host-zrn-pi-session=${signedIn.sessionCookie}`,
      },
    }),
    ENV,
    testHarness.runtime,
  );
  assert.equal((await staleSession.json() as PiSessionEnvelope).authenticated, false);
});

test("wallet challenge is optional, session-bound, rate-limited, and explicitly off-chain", async () => {
  const testHarness = harness();
  const signedIn = await signIn(testHarness);
  const address = toBech32("zrn", new Uint8Array(20).fill(7));
  const fetchesBeforeChallenge = testHarness.calls.length;

  for (const invalidAddress of [
    address.toUpperCase(),
    toBech32("cosmos", new Uint8Array(20).fill(7)),
    `${address.slice(0, -1)}${address.endsWith("q") ? "p" : "q"}`,
  ]) {
    const rejected = await handlePiRequest(
      "challenge",
      authenticatedRequest(
        "/api/pi/challenge",
        "POST",
        signedIn,
        { address: invalidAddress },
      ),
      ENV,
      testHarness.runtime,
    );
    assert.equal(rejected.status, 400);
  }
  assert.equal(testHarness.repository.challenges.size, 0);

  const response = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      signedIn,
      { address },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(response.status, 200);
  const body = (await response.json()) as {
    challengeId: string;
    message: string;
    expiresAt: string;
  };
  assert.match(body.challengeId, /^[A-Za-z0-9_-]{43}$/u);
  assert.match(body.message, /Version: 1/u);
  assert.match(body.message, /Audience: zerone-pi-wallet-link/u);
  assert.match(body.message, /Chain ID: zerone-1/u);
  assert.match(body.message, /Consent Version: pi-wallet-link-v1/u);
  assert.match(body.message, /No transaction, payment, transfer, bridge, reward/u);
  assert.equal(body.message.includes(RAW_UID), false);
  const sessionHash = hashOpaque(signedIn.sessionCookie);
  const session = testHarness.repository.sessions.get(sessionHash);
  if (!session) assert.fail("expected authenticated session");
  const expectedBinding = await keyedHash(
    ENV.PI_SUBJECT_PEPPER ?? "",
    [
      "zerone-pi-wallet-session-binding-v1",
      session.subjectHash,
      sessionHash,
      body.challengeId,
    ].join("\u0000"),
  );
  assert.match(body.message, new RegExp(`Session Binding: ${expectedBinding}`, "u"));
  assert.equal(testHarness.calls.length, fetchesBeforeChallenge);

  for (let count = 1; count < 5; count += 1) {
    const accepted = await handlePiRequest(
      "challenge",
      authenticatedRequest(
        "/api/pi/challenge",
        "POST",
        signedIn,
        { address },
      ),
      ENV,
      testHarness.runtime,
    );
    assert.equal(accepted.status, 200);
  }
  const limited = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      signedIn,
      { address },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(limited.status, 429);
});

test("bind verifies ADR-36 server-side and permanently rejects challenge replay", async () => {
  const testHarness = harness();
  const initialNow = testHarness.runtime.now();
  const signedIn = await signIn(testHarness);
  const wallet = deterministicWallet();
  const challengeResponse = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      signedIn,
      { address: wallet.address },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(challengeResponse.status, 200);
  const challenge = (await challengeResponse.json()) as {
    challengeId: string;
    message: string;
  };
  const signature = wallet.sign(challenge.message);
  const staleChallengeResponse = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      signedIn,
      { address: wallet.address },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(staleChallengeResponse.status, 200);
  const staleChallenge = (await staleChallengeResponse.json()) as {
    challengeId: string;
    message: string;
  };
  const staleSignature = wallet.sign(staleChallenge.message);
  const wrongTranscript = challenge.message.replace(
    "Chain ID: zerone-1",
    "Chain ID: attacker-1",
  );
  assert.notEqual(wrongTranscript, challenge.message);
  const wrongTranscriptProof = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      signedIn,
      {
        challengeId: challenge.challengeId,
        signature: wallet.sign(wrongTranscript),
      },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(wrongTranscriptProof.status, 400);

  const wrongWalletProof = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      signedIn,
      {
        challengeId: challenge.challengeId,
        signature: deterministicWallet(2).sign(challenge.message),
      },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(wrongWalletProof.status, 400);

  const invalidSignature: PiStdSignature = {
    ...signature,
    signature: `${signature.signature.slice(0, -2)}AA`,
  };

  const invalid = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      signedIn,
      { challengeId: challenge.challengeId, signature: invalidSignature },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(invalid.status, 400);

  const otherToken = "B".repeat(43);
  const otherSession: PiSession = {
    tokenHash: hashOpaque(otherToken),
    subjectHash: hashOpaque("other-subject"),
    username: "other",
    expiresAt: initialNow + 60 * 60 * 1_000,
  };
  await testHarness.repository.createSession(otherSession);
  const otherSignedIn: SignedIn = {
    response: new Response(),
    sessionCookie: otherToken,
    envelope: {
      enabled: true,
      walletProofEnabled: true,
      authenticated: true,
      username: otherSession.username,
      csrfToken: await keyedHash(
        ENV.PI_SUBJECT_PEPPER ?? "",
        `zerone-pi-csrf-v1\u0000${otherToken}`,
      ),
    },
  };
  const wrongSession = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      otherSignedIn,
      { challengeId: challenge.challengeId, signature },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(wrongSession.status, 409);

  const bound = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      signedIn,
      { challengeId: challenge.challengeId, signature },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(bound.status, 200);
  const boundEnvelope = (await bound.json()) as PiSessionEnvelope;
  assert.deepEqual(boundEnvelope.linked, {
    address: wallet.address,
    accountId: `cosmos:zerone-1:${wallet.address}`,
  });
  assert.equal(testHarness.repository.challenges.size, 0);

  const challengeWhileBound = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      signedIn,
      { address: wallet.address },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(challengeWhileBound.status, 409);

  const replay = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      signedIn,
      { challengeId: challenge.challengeId, signature },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(replay.status, 409);

  const unlinked = await handlePiRequest(
    "bind",
    authenticatedRequest("/api/pi/bind", "DELETE", signedIn),
    ENV,
    testHarness.runtime,
  );
  assert.equal(unlinked.status, 200);
  assert.equal(
    ((await unlinked.json()) as PiSessionEnvelope).linked,
    undefined,
  );

  const replayAfterUnlink = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      signedIn,
      { challengeId: challenge.challengeId, signature },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(replayAfterUnlink.status, 409);

  const staleReplayAfterUnlink = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      signedIn,
      {
        challengeId: staleChallenge.challengeId,
        signature: staleSignature,
      },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(staleReplayAfterUnlink.status, 409);

  const expiringChallengeResponse = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      signedIn,
      { address: wallet.address },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(expiringChallengeResponse.status, 200);
  const expiringChallenge = (await expiringChallengeResponse.json()) as {
    challengeId: string;
    message: string;
  };
  testHarness.setNow(initialNow + 5 * 60 * 1_000 + 1);
  const expired = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      signedIn,
      {
        challengeId: expiringChallenge.challengeId,
        signature: wallet.sign(expiringChallenge.message),
      },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(expired.status, 409);
});

class SqliteD1Statement implements PiD1PreparedStatement {
  constructor(
    private readonly database: DatabaseSync,
    private readonly query: string,
    private readonly values: readonly unknown[] = [],
  ) {}

  bind(...values: unknown[]): PiD1PreparedStatement {
    return new SqliteD1Statement(this.database, this.query, values);
  }

  async first<T = Record<string, unknown>>(columnName?: string): Promise<T | null> {
    const row = this.database
      .prepare(this.query)
      .get(...(this.values as never[])) as Record<string, unknown> | undefined;
    if (!row) return null;
    return (columnName === undefined ? row : row[columnName]) as T;
  }

  async all<T = Record<string, unknown>>(): Promise<PiD1Result<T>> {
    const rows = this.database
      .prepare(this.query)
      .all(...(this.values as never[])) as T[];
    return { success: true, results: rows };
  }

  async run<T = Record<string, unknown>>(): Promise<PiD1Result<T>> {
    const result = this.database
      .prepare(this.query)
      .run(...(this.values as never[]));
    return {
      success: true,
      results: [],
      meta: { changes: Number(result.changes) },
    };
  }
}

class SqliteD1Database implements PiD1Database {
  constructor(readonly database: DatabaseSync) {}

  prepare(query: string): PiD1PreparedStatement {
    return new SqliteD1Statement(this.database, query);
  }

  async batch<T = Record<string, unknown>>(
    statements: readonly PiD1PreparedStatement[],
  ): Promise<readonly PiD1Result<T>[]> {
    this.database.exec("BEGIN IMMEDIATE");
    try {
      const results: PiD1Result<T>[] = [];
      for (const statement of statements) {
        results.push(await statement.run<T>());
      }
      this.database.exec("COMMIT");
      return results;
    } catch (error) {
      this.database.exec("ROLLBACK");
      throw error;
    }
  }
}

describe("D1 migration and atomic constraints", () => {
  it("atomically claims state, browser transaction, and keyed bearer once", async () => {
    const database = new DatabaseSync(":memory:");
    database.exec(
      readFileSync(
        new URL("../migrations/0001_pi_identity.sql", import.meta.url),
        "utf8",
      ),
    );
    const repository = new D1PiRepository(new SqliteD1Database(database));
    const now = 10_000;
    const stateOne = hashOpaque("state-one");
    const browserOne = hashOpaque("browser-one");
    const bearerOne = hashOpaque("keyed-bearer-one");
    await repository.createOAuthFlow(stateOne, browserOne, now, now + 1_000);

    const bearerTwo = hashOpaque("keyed-bearer-two");
    const concurrentClaims = await Promise.all([
      repository.consumeOAuthFlow(
        stateOne,
        browserOne,
        bearerOne,
        now + 1,
      ),
      repository.consumeOAuthFlow(
        stateOne,
        browserOne,
        bearerTwo,
        now + 1,
      ),
    ]);
    assert.equal(concurrentClaims.filter(Boolean).length, 1);
    const claimedBearer = concurrentClaims[0] ? bearerOne : bearerTwo;
    assert.equal(
      await repository.consumeOAuthFlow(
        stateOne,
        browserOne,
        hashOpaque("other-bearer"),
        now + 2,
      ),
      false,
    );

    const stateTwo = hashOpaque("state-two");
    const browserTwo = hashOpaque("browser-two");
    await repository.createOAuthFlow(stateTwo, browserTwo, now, now + 1_000);
    assert.equal(
      await repository.consumeOAuthFlow(
        stateTwo,
        browserTwo,
        claimedBearer,
        now + 3,
      ),
      false,
    );
    assert.equal(
      await repository.consumeOAuthFlow(
        stateTwo,
        browserTwo,
        hashOpaque("keyed-bearer-three"),
        now + 4,
      ),
      true,
    );
    database.close();
  });

  it("preserves challenge use across unlink and enforces one active 1:1 binding", async () => {
    const database = new DatabaseSync(":memory:");
    database.exec(
      readFileSync(
        new URL("../migrations/0001_pi_identity.sql", import.meta.url),
        "utf8",
      ),
    );
    const repository = new D1PiRepository(new SqliteD1Database(database));
    const now = 50_000;
    const address = `zrn1${"q".repeat(38)}`;
    const accountId = `cosmos:zerone-1:${address}`;
    const sessionOne: PiSession = {
      tokenHash: hashOpaque("session-one"),
      subjectHash: hashOpaque("subject-one"),
      username: "one",
      expiresAt: now + 10_000,
    };
    const sessionTwo: PiSession = {
      tokenHash: hashOpaque("session-two"),
      subjectHash: hashOpaque("subject-two"),
      username: "two",
      expiresAt: now + 10_000,
    };
    await repository.createSession(sessionOne, now);
    await repository.createSession(sessionTwo, now);

    const challengeOne: PiChallenge = {
      idHash: hashOpaque("challenge-one"),
      sessionHash: sessionOne.tokenHash,
      subjectHash: sessionOne.subjectHash,
      address,
      accountId,
      message: "challenge one",
      createdAt: now,
      expiresAt: now + 1_000,
    };
    assert.equal(
      await repository.createChallenge(challengeOne, now - 1_000, 5),
      true,
    );
    const staleChallenge: PiChallenge = {
      ...challengeOne,
      idHash: hashOpaque("challenge-stale"),
      message: "challenge stale",
    };
    assert.equal(
      await repository.createChallenge(staleChallenge, now - 1_000, 5),
      true,
    );
    const concurrentBindings = await Promise.all([
      repository.bindChallenge(
        challengeOne.idHash,
        sessionOne.tokenHash,
        sessionOne.subjectHash,
        hashOpaque("proof-one-a"),
        "pi-wallet-link-v1",
        now + 1,
      ),
      repository.bindChallenge(
        challengeOne.idHash,
        sessionOne.tokenHash,
        sessionOne.subjectHash,
        hashOpaque("proof-one-b"),
        "pi-wallet-link-v1",
        now + 1,
      ),
    ]);
    assert.equal(
      concurrentBindings.filter((binding) => binding !== null).length,
      1,
    );
    const firstBinding = concurrentBindings.find(
      (binding) => binding !== null,
    );
    assert.equal(firstBinding?.address, address);
    assert.equal(firstBinding?.consentVersion, "pi-wallet-link-v1");
    assert.equal(
      await repository.bindChallenge(
        challengeOne.idHash,
        sessionOne.tokenHash,
        sessionOne.subjectHash,
        hashOpaque("proof-one-replay"),
        "pi-wallet-link-v1",
        now + 2,
      ),
      null,
    );

    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_wallet_challenges
           WHERE subject_hash = ?`,
        )
        .get(sessionOne.subjectHash)?.count,
      0,
    );
    assert.deepEqual(
      database
        .prepare(
          `SELECT challenge_hash, disposition
           FROM pi_wallet_challenge_uses
           ORDER BY challenge_hash`,
        )
        .all()
        .map((row) => ({
          challenge_hash: row.challenge_hash,
          disposition: row.disposition,
        })),
      [
        {
          challenge_hash: challengeOne.idHash,
          disposition: "bound",
        },
        {
          challenge_hash: staleChallenge.idHash,
          disposition: "superseded",
        },
      ].sort((left, right) =>
        left.challenge_hash.localeCompare(right.challenge_hash),
      ),
    );
    await repository.deleteBinding(sessionOne.subjectHash, now + 2);
    assert.equal(
      await repository.bindChallenge(
        challengeOne.idHash,
        sessionOne.tokenHash,
        sessionOne.subjectHash,
        hashOpaque("proof-one-after-unlink"),
        "pi-wallet-link-v1",
        now + 3,
      ),
      null,
    );
    assert.equal(
      await repository.bindChallenge(
        staleChallenge.idHash,
        sessionOne.tokenHash,
        sessionOne.subjectHash,
        hashOpaque("proof-stale-after-unlink"),
        "pi-wallet-link-v1",
        now + 3,
      ),
      null,
    );

    const reconsentChallenge: PiChallenge = {
      ...challengeOne,
      idHash: hashOpaque("challenge-reconsent"),
      message: "challenge reconsent",
    };
    assert.equal(
      await repository.createChallenge(reconsentChallenge, now - 1_000, 5),
      true,
    );
    assert.equal(
      (
        await repository.bindChallenge(
          reconsentChallenge.idHash,
          sessionOne.tokenHash,
          sessionOne.subjectHash,
          hashOpaque("proof-reconsent"),
          "pi-wallet-link-v1",
          now + 4,
        )
      )?.address,
      address,
    );
    await repository.deleteBinding(sessionOne.subjectHash, now + 5);

    const challengeTwo: PiChallenge = {
      ...challengeOne,
      idHash: hashOpaque("challenge-two"),
      sessionHash: sessionTwo.tokenHash,
      subjectHash: sessionTwo.subjectHash,
      message: "challenge two",
    };
    assert.equal(
      await repository.createChallenge(challengeTwo, now - 1_000, 5),
      true,
    );
    assert.equal(
      (
        await repository.bindChallenge(
          challengeTwo.idHash,
          sessionTwo.tokenHash,
          sessionTwo.subjectHash,
          hashOpaque("proof-two"),
          "pi-wallet-link-v1",
          now + 5,
        )
      )?.subjectHash,
      sessionTwo.subjectHash,
    );

    const challengeThree: PiChallenge = {
      ...challengeOne,
      idHash: hashOpaque("challenge-three"),
      message: "challenge three",
    };
    assert.equal(
      await repository.createChallenge(challengeThree, now - 1_000, 5),
      true,
    );
    assert.equal(
      await repository.bindChallenge(
        challengeThree.idHash,
        sessionOne.tokenHash,
        sessionOne.subjectHash,
        hashOpaque("proof-three"),
        "pi-wallet-link-v1",
        now + 6,
      ),
      null,
    );
    database.close();
  });

  it("deletes one subject's sessions, binding, challenges, and linked replay row without touching another subject", async () => {
    const database = new DatabaseSync(":memory:");
    database.exec(
      readFileSync(
        new URL("../migrations/0001_pi_identity.sql", import.meta.url),
        "utf8",
      ),
    );
    const repository = new D1PiRepository(new SqliteD1Database(database));
    const now = 80_000;
    const subjectOne = hashOpaque("privacy-subject-one");
    const subjectTwo = hashOpaque("privacy-subject-two");
    const sessionOne: PiSession = {
      tokenHash: hashOpaque("privacy-session-one"),
      subjectHash: subjectOne,
      username: "one",
      expiresAt: now + 10_000,
    };
    const sessionOneAgain: PiSession = {
      ...sessionOne,
      tokenHash: hashOpaque("privacy-session-one-again"),
    };
    const sessionTwo: PiSession = {
      tokenHash: hashOpaque("privacy-session-two"),
      subjectHash: subjectTwo,
      username: "two",
      expiresAt: now + 10_000,
    };
    await repository.createSession(sessionOne, now);
    await repository.createSession(sessionOneAgain, now);
    await repository.createSession(sessionTwo, now);

    const addressOne = `zrn1${"q".repeat(38)}`;
    const challengeOne: PiChallenge = {
      idHash: hashOpaque("privacy-bound-challenge"),
      sessionHash: sessionOne.tokenHash,
      subjectHash: subjectOne,
      address: addressOne,
      accountId: `cosmos:zerone-1:${addressOne}`,
      message: "privacy bound challenge",
      createdAt: now,
      expiresAt: now + 1_000,
    };
    assert.equal(
      await repository.createChallenge(challengeOne, now - 1_000, 5),
      true,
    );
    assert.ok(
      await repository.bindChallenge(
        challengeOne.idHash,
        sessionOne.tokenHash,
        subjectOne,
        hashOpaque("privacy-proof"),
        "pi-wallet-link-v1",
        now + 1,
      ),
    );

    const outstandingHash = hashOpaque("privacy-outstanding-challenge");
    database
      .prepare(
        `INSERT INTO pi_wallet_challenges
           (id_hash, session_hash, subject_hash, address, account_id, message,
            created_at, expires_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
      )
      .run(
        outstandingHash,
        sessionOneAgain.tokenHash,
        subjectOne,
        addressOne,
        `cosmos:zerone-1:${addressOne}`,
        "privacy outstanding challenge",
        now,
        now + 1_000,
      );

    const addressTwo = `zrn1${"p".repeat(38)}`;
    const challengeTwo: PiChallenge = {
      ...challengeOne,
      idHash: hashOpaque("privacy-other-challenge"),
      sessionHash: sessionTwo.tokenHash,
      subjectHash: subjectTwo,
      address: addressTwo,
      accountId: `cosmos:zerone-1:${addressTwo}`,
      message: "privacy other challenge",
    };
    assert.equal(
      await repository.createChallenge(challengeTwo, now - 1_000, 5),
      true,
    );

    await repository.deleteSubject(subjectOne);

    assert.equal(
      database
        .prepare("SELECT COUNT(*) AS count FROM pi_sessions WHERE subject_hash = ?")
        .get(subjectOne)?.count,
      0,
    );
    assert.equal(
      database
        .prepare("SELECT COUNT(*) AS count FROM pi_wallet_bindings WHERE subject_hash = ?")
        .get(subjectOne)?.count,
      0,
    );
    assert.equal(
      database
        .prepare("SELECT COUNT(*) AS count FROM pi_wallet_challenges WHERE subject_hash = ?")
        .get(subjectOne)?.count,
      0,
    );
    assert.equal(
      database
        .prepare("SELECT COUNT(*) AS count FROM pi_wallet_challenge_uses WHERE challenge_hash = ?")
        .get(challengeOne.idHash)?.count,
      0,
    );
    assert.equal(
      database
        .prepare("SELECT COUNT(*) AS count FROM pi_sessions WHERE subject_hash = ?")
        .get(subjectTwo)?.count,
      1,
    );
    assert.equal(
      database
        .prepare("SELECT COUNT(*) AS count FROM pi_wallet_challenges WHERE subject_hash = ?")
        .get(subjectTwo)?.count,
      1,
    );
    assert.equal(
      await repository.createChallenge(
        {
          ...challengeOne,
          idHash: hashOpaque("privacy-post-delete-challenge"),
          message: "privacy post-delete challenge",
          createdAt: now + 2,
        },
        now - 1_000,
        5,
      ),
      false,
    );
    database.close();
  });
});
