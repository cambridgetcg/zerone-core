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
  PI_BEARER_PENDING_TTL_MS,
  PI_SUBJECT_DELETION_GUARD_MS,
  runPiEndpoint,
} from "../functions/api/pi/_service";
import {
  D1PiRepository,
  piPepperKeysetFingerprint,
} from "../functions/api/pi/_store";
import {
  adr36SignBytes,
  parsePiStdSignature,
} from "../functions/api/pi/_wallet-proof";
import { PI_CHALLENGE_RATE_WINDOW_MS } from "../functions/api/pi/_types";
import type {
  PiBinding,
  PiChallenge,
  PiD1Database,
  PiD1PreparedStatement,
  PiD1Result,
  PiEnv,
  PiPepperPin,
  PiRepository,
  PiRetentionPolicy,
  PiRetentionResult,
  PiRuntime,
  PiSession,
  PiSessionEnvelope,
  PiStdSignature,
  PiSubjectAlias,
} from "../functions/api/pi/_types";

const ORIGIN = "https://zerone.ai";
const ENV: PiEnv = {
  PI_BEARER_SHA_CLEAN_START_CONFIRMED: "true",
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
  readonly deletionEpoch: number;
  consumed: boolean;
}

class MemoryPiRepository implements PiRepository {
  readonly oauthFlows = new Map<string, OAuthFlow>();
  readonly bearerFingerprints = new Set<string>();
  readonly pendingBearerClaims = new Map<
    string,
    { readonly stateHash: string; readonly browserHash: string }
  >();
  readonly sessions = new Map<string, PiSession>();
  readonly challenges = new Map<string, PiChallenge>();
  readonly challengeRateEvents = new Map<
    string,
    { readonly subjectHash: string; readonly createdAt: number }
  >();
  readonly challengeUses = new Set<string>();
  readonly bindings = new Map<string, PiBinding>();
  readonly addressSubjects = new Map<string, string>();
  readonly revoked = new Set<string>();
  readonly pepperPins = new Map<number, string>();
  readonly subjectAliases = new Map<string, string>();
  readonly deletionGuards = new Map<
    string,
    {
      readonly deletedAt: number;
      readonly expiresAt: number;
      readonly deletionEpoch: number;
      readonly operationHash: string;
    }
  >();
  readonly bindingChallengeHashes = new Map<string, string>();
  activePepperVersion: number | null = null;
  configuredKeysetFingerprint: string | null = null;
  deletionEpoch = 0;

  private aliasKey(alias: PiSubjectAlias): string {
    return `${alias.pepperVersion}:${alias.aliasHash}`;
  }

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
      deletionEpoch: this.deletionEpoch,
      consumed: false,
    });
  }

  async consumeOAuthFlow(
    stateHash: string,
    browserHash: string,
    bearerReplayCommitment: string,
    activePepperVersion: number,
    now: number,
  ): Promise<number | null> {
    const flow = this.oauthFlows.get(stateHash);
    if (
      !flow ||
      flow.browserHash !== browserHash ||
      flow.consumed ||
      flow.expiresAt <= now ||
      this.activePepperVersion !== activePepperVersion ||
      this.bearerFingerprints.has(bearerReplayCommitment) ||
      this.pendingBearerClaims.has(bearerReplayCommitment)
    ) {
      return null;
    }
    flow.consumed = true;
    this.pendingBearerClaims.set(bearerReplayCommitment, {
      stateHash,
      browserHash,
    });
    return flow.deletionEpoch;
  }

  async promoteBearerClaim(
    bearerReplayCommitment: string,
    stateHash: string,
    browserHash: string,
  ): Promise<boolean> {
    const pending = this.pendingBearerClaims.get(bearerReplayCommitment);
    if (
      !pending ||
      pending.stateHash !== stateHash ||
      pending.browserHash !== browserHash ||
      this.bearerFingerprints.has(bearerReplayCommitment)
    ) {
      return false;
    }
    this.bearerFingerprints.add(bearerReplayCommitment);
    this.pendingBearerClaims.delete(bearerReplayCommitment);
    return true;
  }

  async rejectBearerClaim(
    bearerReplayCommitment: string,
    stateHash: string,
    browserHash: string,
  ): Promise<void> {
    const pending = this.pendingBearerClaims.get(bearerReplayCommitment);
    if (
      pending?.stateHash === stateHash &&
      pending.browserHash === browserHash
    ) {
      this.pendingBearerClaims.delete(bearerReplayCommitment);
    }
  }

  async ensurePepperConfiguration(
    activeVersion: number,
    pins: readonly PiPepperPin[],
    now: number,
  ): Promise<boolean> {
    const keysetFingerprint = piPepperKeysetFingerprint(activeVersion, pins);
    if (
      pins[0]?.version !== activeVersion ||
      (this.activePepperVersion !== null &&
        activeVersion < this.activePepperVersion) ||
      (this.configuredKeysetFingerprint !== null &&
        this.configuredKeysetFingerprint !== keysetFingerprint &&
        [...this.deletionGuards.values()].some(
          (guard) => guard.expiresAt > now,
        ))
    ) {
      return false;
    }
    const configuredVersions = new Set(pins.map((pin) => pin.version));
    const durableSubjects = new Set([
      ...[...this.sessions.values()].map((session) => session.subjectHash),
      ...[...this.challenges.values()].map(
        (challenge) => challenge.subjectHash,
      ),
      ...this.bindings.keys(),
    ]);
    for (const subjectHash of durableSubjects) {
      const hasConfiguredAlias = [...this.subjectAliases.entries()].some(
        ([key, canonical]) =>
          canonical === subjectHash &&
          configuredVersions.has(Number(key.split(":", 1)[0])),
      );
      if (!hasConfiguredAlias) return false;
    }
    for (const pin of pins) {
      const existing = this.pepperPins.get(pin.version);
      if (existing !== undefined && existing !== pin.fingerprint) return false;
      for (const [version, fingerprint] of this.pepperPins) {
        if (version !== pin.version && fingerprint === pin.fingerprint) {
          return false;
        }
      }
    }
    for (const pin of pins) this.pepperPins.set(pin.version, pin.fingerprint);
    this.activePepperVersion = activeVersion;
    this.configuredKeysetFingerprint = keysetFingerprint;
    return true;
  }

  async resolveSubject(
    aliases: readonly PiSubjectAlias[],
  ): Promise<string | null> {
    const existing = new Set(
      aliases
        .map((alias) => this.subjectAliases.get(this.aliasKey(alias)))
        .filter((subject): subject is string => subject !== undefined),
    );
    if (existing.size > 1) return null;
    return existing.values().next().value ?? aliases[0]?.aliasHash ?? null;
  }

  async replaceSubjectSessions(
    session: PiSession,
    aliases: readonly PiSubjectAlias[],
    oauthDeletionEpoch: number,
    createdAt: number,
    expectedKeysetFingerprint: string,
  ): Promise<boolean> {
    if (
      this.activePepperVersion !== session.pepperVersion ||
      this.configuredKeysetFingerprint !== expectedKeysetFingerprint ||
      this.sessions.has(session.tokenHash) ||
      (await this.resolveSubject(aliases)) !== session.subjectHash
    ) {
      return false;
    }
    for (const alias of aliases) {
      const canonical = this.subjectAliases.get(this.aliasKey(alias));
      if (canonical !== undefined && canonical !== session.subjectHash) {
        return false;
      }
    }
    const guardHashes = new Set([
      session.subjectHash,
      ...aliases.map((alias) => alias.aliasHash),
    ]);
    for (const hash of guardHashes) {
      const guard = this.deletionGuards.get(hash);
      if (
        guard &&
        guard.deletionEpoch > oauthDeletionEpoch &&
        guard.expiresAt > createdAt
      ) {
        return false;
      }
    }
    for (const alias of aliases) {
      this.subjectAliases.set(this.aliasKey(alias), session.subjectHash);
    }
    this.sessions.set(session.tokenHash, session);
    for (const candidate of this.sessions.values()) {
      if (
        candidate.subjectHash === session.subjectHash &&
        candidate.tokenHash !== session.tokenHash
      ) {
        this.revoked.add(candidate.tokenHash);
      }
    }
    return true;
  }

  async createSession(session: PiSession): Promise<void> {
    if (this.sessions.has(session.tokenHash)) throw new Error("duplicate session");
    this.sessions.set(session.tokenHash, session);
    if (session.pepperVersion === 1) {
      this.subjectAliases.set(`1:${session.subjectHash}`, session.subjectHash);
    }
  }

  async getSession(
    tokenHash: string,
    now: number,
    idleSince: number,
  ): Promise<PiSession | null> {
    const session = this.sessions.get(tokenHash);
    if (
      !session ||
      session.expiresAt <= now ||
      session.lastSeenAt <= idleSince ||
      this.revoked.has(tokenHash)
    ) {
      return null;
    }
    const touched = {
      ...session,
      lastSeenAt: Math.max(session.lastSeenAt, now),
    };
    this.sessions.set(tokenHash, touched);
    return touched;
  }

  async revokeSession(tokenHash: string): Promise<void> {
    this.revoked.add(tokenHash);
  }

  async createChallenge(
    challenge: PiChallenge,
    recentSince: number,
    maximumRecent: number,
    idleSince: number,
  ): Promise<boolean> {
    const recent = [...this.challengeRateEvents.values()].filter(
      (candidate) =>
        candidate.subjectHash === challenge.subjectHash &&
        candidate.createdAt > recentSince,
    ).length;
    const session = this.sessions.get(challenge.sessionHash);
    if (
      recent >= maximumRecent ||
      this.challenges.has(challenge.idHash) ||
      this.challengeRateEvents.has(challenge.idHash) ||
      this.bindings.has(challenge.subjectHash) ||
      !session ||
      session.subjectHash !== challenge.subjectHash ||
      session.expiresAt <= challenge.createdAt ||
      session.lastSeenAt <= idleSince ||
      this.revoked.has(challenge.sessionHash)
    ) {
      return false;
    }
    this.challenges.set(challenge.idHash, challenge);
    this.challengeRateEvents.set(challenge.idHash, {
      subjectHash: challenge.subjectHash,
      createdAt: challenge.createdAt,
    });
    return true;
  }

  async getChallenge(
    idHash: string,
    sessionHash: string,
    subjectHash: string,
    now: number,
    idleSince: number,
  ): Promise<PiChallenge | null> {
    const challenge = this.challenges.get(idHash);
    const session = this.sessions.get(sessionHash);
    return challenge &&
      session &&
      challenge.sessionHash === sessionHash &&
      challenge.subjectHash === subjectHash &&
      challenge.expiresAt > now &&
      session.expiresAt > now &&
      session.lastSeenAt > idleSince &&
      !this.revoked.has(sessionHash) &&
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
    rotatedSession: PiSession,
    idleSince: number,
    now: number,
  ): Promise<PiBinding | null> {
    const challenge = await this.getChallenge(
      idHash,
      sessionHash,
      subjectHash,
      now,
      idleSince,
    );
    if (
      !challenge ||
      this.activePepperVersion !== rotatedSession.pepperVersion ||
      rotatedSession.subjectHash !== subjectHash ||
      this.sessions.has(rotatedSession.tokenHash) ||
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
    this.bindingChallengeHashes.set(subjectHash, idHash);
    this.addressSubjects.set(challenge.address, subjectHash);
    for (const candidate of [...this.challenges.values()]) {
      if (candidate.subjectHash !== subjectHash) continue;
      this.challengeUses.add(candidate.idHash);
      this.challenges.delete(candidate.idHash);
    }
    this.sessions.set(rotatedSession.tokenHash, rotatedSession);
    for (const candidate of this.sessions.values()) {
      if (
        candidate.subjectHash === subjectHash &&
        candidate.tokenHash !== rotatedSession.tokenHash
      ) {
        this.revoked.add(candidate.tokenHash);
      }
    }
    assert.equal(proofHash.length, 43);
    return binding;
  }

  async getBinding(subjectHash: string): Promise<PiBinding | null> {
    return this.bindings.get(subjectHash) ?? null;
  }

  async deleteBinding(
    subjectHash: string,
    currentSessionHash: string,
    rotatedSession: PiSession,
    idleSince: number,
    now: number,
  ): Promise<boolean> {
    const current = this.sessions.get(currentSessionHash);
    const binding = this.bindings.get(subjectHash);
    if (
      !current ||
      !binding ||
      current.subjectHash !== subjectHash ||
      current.expiresAt <= now ||
      current.lastSeenAt <= idleSince ||
      this.revoked.has(currentSessionHash) ||
      this.activePepperVersion !== rotatedSession.pepperVersion ||
      rotatedSession.subjectHash !== subjectHash ||
      this.sessions.has(rotatedSession.tokenHash)
    ) {
      return false;
    }
    this.addressSubjects.delete(binding.address);
    this.bindings.delete(subjectHash);
    this.bindingChallengeHashes.delete(subjectHash);
    for (const candidate of [...this.challenges.values()]) {
      if (candidate.subjectHash !== subjectHash) continue;
      this.challenges.delete(candidate.idHash);
    }
    this.sessions.set(rotatedSession.tokenHash, rotatedSession);
    for (const candidate of this.sessions.values()) {
      if (
        candidate.subjectHash === subjectHash &&
        candidate.tokenHash !== rotatedSession.tokenHash
      ) {
        this.revoked.add(candidate.tokenHash);
      }
    }
    return true;
  }

  async deleteSubject(
    subjectHash: string,
    now: number,
    guardExpiresAt: number,
    expectedActivePepperVersion: number,
    expectedKeysetFingerprint: string,
    operationHash: string,
  ): Promise<boolean> {
    if (
      guardExpiresAt <= now ||
      this.activePepperVersion !== expectedActivePepperVersion ||
      this.configuredKeysetFingerprint !== expectedKeysetFingerprint ||
      operationHash.length !== 43
    ) {
      return false;
    }
    const deletionEpoch = this.deletionEpoch + 1;
    this.deletionEpoch = deletionEpoch;
    const aliasHashes = [...this.subjectAliases.entries()]
      .filter(([, canonical]) => canonical === subjectHash)
      .map(([key]) => key.slice(key.indexOf(":") + 1));
    for (const hash of new Set([subjectHash, ...aliasHashes])) {
      this.deletionGuards.set(hash, {
        deletedAt: now,
        expiresAt: guardExpiresAt,
        deletionEpoch,
        operationHash,
      });
    }
    for (const challenge of [...this.challenges.values()]) {
      if (challenge.subjectHash !== subjectHash) continue;
      this.challengeUses.delete(challenge.idHash);
      this.challenges.delete(challenge.idHash);
    }
    for (const [challengeHash, event] of this.challengeRateEvents) {
      if (event.subjectHash === subjectHash) {
        this.challengeRateEvents.delete(challengeHash);
      }
    }
    const directUse = this.bindingChallengeHashes.get(subjectHash);
    if (directUse) this.challengeUses.delete(directUse);
    const binding = this.bindings.get(subjectHash);
    if (binding) this.addressSubjects.delete(binding.address);
    this.bindings.delete(subjectHash);
    this.bindingChallengeHashes.delete(subjectHash);
    for (const [tokenHash, session] of this.sessions) {
      if (session.subjectHash !== subjectHash) continue;
      this.sessions.delete(tokenHash);
      this.revoked.delete(tokenHash);
    }
    for (const [key, canonical] of this.subjectAliases) {
      if (canonical === subjectHash) this.subjectAliases.delete(key);
    }
    return true;
  }

  async cleanupExpired(
    _policy: PiRetentionPolicy,
  ): Promise<PiRetentionResult> {
    return {
      oauthFlowsDeleted: 0,
      bearerPendingClaimsDeleted: 0,
      challengesDeleted: 0,
      challengeRateEventsDeleted: 0,
      challengeUsesDeleted: 0,
      sessionsDeleted: 0,
      bindingsDeleted: 0,
      subjectAliasesDeleted: 0,
      deletionGuardsDeleted: 0,
    };
  }
}

function memoryPepperConfiguration(
  repository: MemoryPiRepository,
): readonly [number, string] {
  if (
    repository.activePepperVersion === null ||
    repository.configuredKeysetFingerprint === null
  ) {
    assert.fail("expected an active in-memory pepper configuration");
  }
  return [
    repository.activePepperVersion,
    repository.configuredKeysetFingerprint,
  ];
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

function authorizationRequest(
  url = `${ORIGIN}/api/pi/authorize`,
  headers: HeadersInit = {},
  body = {},
): Request {
  return new Request(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Origin: ORIGIN,
      "Sec-Fetch-Site": "same-origin",
      ...Object.fromEntries(new Headers(headers)),
    },
    body: JSON.stringify(body),
  });
}

async function startOAuth(
  testHarness: Harness,
  env: PiEnv = ENV,
): Promise<OAuthStart> {
  const response = await handlePiRequest(
    "authorize",
    authorizationRequest(),
    env,
    testHarness.runtime,
  );
  assert.equal(response.status, 200);
  const body = (await response.clone().json()) as JsonRecord;
  if (typeof body.authorizationUrl !== "string") {
    assert.fail("authorize response did not include authorizationUrl");
  }
  const target = new URL(body.authorizationUrl);
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

async function signIn(
  testHarness: Harness,
  env: PiEnv = ENV,
  accessToken = ACCESS_TOKEN,
): Promise<SignedIn> {
  const flow = await startOAuth(testHarness, env);
  const response = await handlePiRequest(
    "session",
    sessionRequest(flow, accessToken),
    env,
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

test("enabled edge requires explicit clean-start bearer provenance confirmation", async () => {
  for (const confirmation of [undefined, "false", "TRUE", " true"] as const) {
    const testHarness = harness();
    const response = await handlePiRequest(
      "me",
      new Request(`${ORIGIN}/api/pi/me`),
      { ...ENV, PI_BEARER_SHA_CLEAN_START_CONFIRMED: confirmation },
      testHarness.runtime,
    );
    assert.equal(response.status, 503);
    assert.equal(testHarness.repository.activePepperVersion, null);
  }
});

test("authorize uses fixed Pi OAuth parameters and a distinct browser transaction", async () => {
  const testHarness = harness();
  const response = await handlePiRequest(
    "authorize",
    authorizationRequest(),
    ENV,
    testHarness.runtime,
  );

  assert.equal(response.status, 200);
  assert.equal(response.headers.get("Location"), null);
  assert.match(response.headers.get("Content-Type") ?? "", /^application\/json/u);
  const body = (await response.clone().json()) as JsonRecord;
  assert.equal(typeof body.authorizationUrl, "string");
  const target = new URL(String(body.authorizationUrl));
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

test("authorize rejects cross-site, headerless, and legacy GET initiation before creating state", async () => {
  const testHarness = harness();
  const requests = [
    new Request(`${ORIGIN}/api/pi/authorize`, {
      headers: {
        Origin: ORIGIN,
        "Sec-Fetch-Site": "same-origin",
      },
    }),
    new Request(`${ORIGIN}/api/pi/authorize`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Origin: "https://attacker.invalid",
        "Sec-Fetch-Site": "cross-site",
      },
      body: "{}",
    }),
    new Request(`${ORIGIN}/api/pi/authorize`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Origin: ORIGIN,
        "Sec-Fetch-Site": "cross-site",
      },
      body: "{}",
    }),
    new Request(`${ORIGIN}/api/pi/authorize`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Sec-Fetch-Site": "same-origin",
      },
      body: "{}",
    }),
    new Request(`${ORIGIN}/api/pi/authorize`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Origin: ORIGIN,
      },
      body: "{}",
    }),
  ];

  for (const [index, request] of requests.entries()) {
    const response = await handlePiRequest(
      "authorize",
      request,
      ENV,
      testHarness.runtime,
    );
    assert.equal(response.status, index === 0 ? 405 : 403);
    assert.equal(response.headers.get("Set-Cookie"), null);
    assert.equal(response.headers.get("Location"), null);
  }
  assert.equal(testHarness.repository.oauthFlows.size, 0);
});

test("authorize rejects query parameters and non-empty JSON before creating state", async () => {
  const testHarness = harness();
  const withQuery = await handlePiRequest(
    "authorize",
    authorizationRequest(`${ORIGIN}/api/pi/authorize?next=pi`),
    ENV,
    testHarness.runtime,
  );
  const withBody = await handlePiRequest(
    "authorize",
    authorizationRequest(`${ORIGIN}/api/pi/authorize`, {}, { next: "pi" }),
    ENV,
    testHarness.runtime,
  );

  assert.equal(withQuery.status, 400);
  assert.equal(withBody.status, 400);
  assert.equal(withQuery.headers.get("Set-Cookie"), null);
  assert.equal(withBody.headers.get("Set-Cookie"), null);
  assert.equal(testHarness.repository.oauthFlows.size, 0);
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
  assert.equal(
    testHarness.repository.bearerFingerprints.has(
      hashOpaque(`zerone-pi-bearer-replay-v1\u0000${ACCESS_TOKEN}`),
    ),
    true,
  );
  assert.equal(testHarness.repository.pendingBearerClaims.size, 0);
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
  assert.equal(redirectHarness.repository.bearerFingerprints.size, 0);
  assert.equal(redirectHarness.repository.pendingBearerClaims.size, 1);

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

  const unauthorizedHarness = harness();
  unauthorizedHarness.setFetch(async () => new Response(null, { status: 401 }));
  const unauthorizedFlow = await startOAuth(unauthorizedHarness);
  const unauthorizedResponse = await handlePiRequest(
    "session",
    sessionRequest(unauthorizedFlow, "random-invalid-access-token-123"),
    ENV,
    unauthorizedHarness.runtime,
  );
  assert.equal(unauthorizedResponse.status, 401);
  assert.equal(unauthorizedHarness.repository.bearerFingerprints.size, 0);
  assert.equal(unauthorizedHarness.repository.pendingBearerClaims.size, 0);
  const unauthorizedRetryFlow = await startOAuth(unauthorizedHarness);
  const unauthorizedRetry = await handlePiRequest(
    "session",
    sessionRequest(
      unauthorizedRetryFlow,
      "random-invalid-access-token-123",
    ),
    ENV,
    unauthorizedHarness.runtime,
  );
  assert.equal(unauthorizedRetry.status, 401);
  assert.equal(unauthorizedHarness.calls.length, 2);
  assert.equal(unauthorizedHarness.repository.bearerFingerprints.size, 0);
  assert.equal(unauthorizedHarness.repository.pendingBearerClaims.size, 0);

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

  const extraField = await handlePiRequest(
    "data",
    authenticatedRequest(
      "/api/pi/data",
      "DELETE",
      signedIn,
      {
        confirmation: "delete-pi-pilot-data-v1",
        unexpected: true,
      },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(extraField.status, 400);

  const queryString = await handlePiRequest(
    "data",
    authenticatedRequest(
      "/api/pi/data?unexpected=1",
      "DELETE",
      signedIn,
      { confirmation: "delete-pi-pilot-data-v1" },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(queryString.status, 400);

  const wrongMethod = await handlePiRequest(
    "data",
    authenticatedRequest(
      "/api/pi/data",
      "POST",
      signedIn,
      { confirmation: "delete-pi-pilot-data-v1" },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(wrongMethod.status, 405);
  assert.equal(wrongMethod.headers.get("Allow"), "DELETE");

  const badCsrf = await handlePiRequest(
    "data",
    authenticatedRequest(
      "/api/pi/data",
      "DELETE",
      signedIn,
      { confirmation: "delete-pi-pilot-data-v1" },
      { "X-Zerone-CSRF": "Z".repeat(43) },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(badCsrf.status, 401);
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

  const deleteSubject = testHarness.repository.deleteSubject.bind(
    testHarness.repository,
  );
  testHarness.repository.deleteSubject = async () => false;
  const rejectedDeletion = await handlePiRequest(
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
  assert.equal(rejectedDeletion.status, 409);
  assert.equal(rejectedDeletion.headers.get("Set-Cookie"), null);
  assert.equal(testHarness.repository.sessions.has(sessionHash), true);
  assert.equal(
    testHarness.repository.deletionGuards.has(session.subjectHash),
    false,
  );
  testHarness.repository.deleteSubject = deleteSubject;

  const deletionAt = testHarness.runtime.now();
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
  const deletionGuard = testHarness.repository.deletionGuards.get(
    session.subjectHash,
  );
  assert.ok(deletionGuard);
  assert.deepEqual(
    {
      deletedAt: deletionGuard.deletedAt,
      expiresAt: deletionGuard.expiresAt,
      deletionEpoch: deletionGuard.deletionEpoch,
    },
    {
      deletedAt: deletionAt,
      expiresAt: deletionAt + PI_SUBJECT_DELETION_GUARD_MS,
      deletionEpoch: 1,
    },
  );
  assert.equal(deletionGuard.operationHash.length, 43);
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
    pepperVersion: 1,
    lastSeenAt: initialNow,
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
  const linkedSignedIn: SignedIn = {
    response: bound,
    sessionCookie: cookieValue(bound, "__Host-zrn-pi-session"),
    envelope: boundEnvelope,
  };
  assert.notEqual(linkedSignedIn.sessionCookie, signedIn.sessionCookie);
  assert.equal(testHarness.repository.challenges.size, 0);

  const challengeWhileBound = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      linkedSignedIn,
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
  assert.equal(replay.status, 401);

  const unlinked = await handlePiRequest(
    "bind",
    authenticatedRequest("/api/pi/bind", "DELETE", linkedSignedIn),
    ENV,
    testHarness.runtime,
  );
  assert.equal(unlinked.status, 200);
  const unlinkedEnvelope = (await unlinked.json()) as PiSessionEnvelope;
  assert.equal(unlinkedEnvelope.linked, undefined);
  const unlinkedSignedIn: SignedIn = {
    response: unlinked,
    sessionCookie: cookieValue(unlinked, "__Host-zrn-pi-session"),
    envelope: unlinkedEnvelope,
  };
  assert.notEqual(
    unlinkedSignedIn.sessionCookie,
    linkedSignedIn.sessionCookie,
  );
  const unlinkedSessionHash = hashOpaque(unlinkedSignedIn.sessionCookie);
  const unlinkedSession = testHarness.repository.sessions.get(
    unlinkedSessionHash,
  );
  if (!unlinkedSession) assert.fail("expected rotated unlinked session");
  const redundantUnlink = await handlePiRequest(
    "bind",
    authenticatedRequest("/api/pi/bind", "DELETE", unlinkedSignedIn),
    ENV,
    testHarness.runtime,
  );
  assert.equal(redundantUnlink.status, 409);
  assert.equal(redundantUnlink.headers.get("Set-Cookie"), null);
  assert.equal(
    testHarness.repository.revoked.has(unlinkedSessionHash),
    false,
  );
  assert.equal(
    [...testHarness.repository.sessions.values()].filter(
      (session) =>
        session.subjectHash === unlinkedSession.subjectHash &&
        !testHarness.repository.revoked.has(session.tokenHash),
    ).length,
    1,
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
  assert.equal(replayAfterUnlink.status, 401);

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
  assert.equal(staleReplayAfterUnlink.status, 401);

  const expiringChallengeResponse = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      unlinkedSignedIn,
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
  for (let count = 0; count < 2; count += 1) {
    const acceptedAcrossRotation = await handlePiRequest(
      "challenge",
      authenticatedRequest(
        "/api/pi/challenge",
        "POST",
        unlinkedSignedIn,
        { address: wallet.address },
      ),
      ENV,
      testHarness.runtime,
    );
    assert.equal(acceptedAcrossRotation.status, 200);
  }
  const limitedAcrossRotation = await handlePiRequest(
    "challenge",
    authenticatedRequest(
      "/api/pi/challenge",
      "POST",
      unlinkedSignedIn,
      { address: wallet.address },
    ),
    ENV,
    testHarness.runtime,
  );
  assert.equal(limitedAcrossRotation.status, 429);
  testHarness.setNow(initialNow + 5 * 60 * 1_000 + 1);
  const expired = await handlePiRequest(
    "bind",
    authenticatedRequest(
      "/api/pi/bind",
      "POST",
      unlinkedSignedIn,
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

test("reauth rotates the subject session family and enforces idle plus absolute expiry", async () => {
  const startedAt = 5_000_000;
  const testHarness = harness(startedAt);
  const first = await signIn(testHarness);
  assert.equal(first.response.status, 200);

  testHarness.setNow(startedAt + 1_000);
  const second = await signIn(
    testHarness,
    ENV,
    "pi-access-token-reauth-123456789",
  );
  assert.equal(second.response.status, 200);
  assert.notEqual(second.sessionCookie, first.sessionCookie);
  assert.equal(
    [...testHarness.repository.sessions.values()].filter(
      (session) => !testHarness.repository.revoked.has(session.tokenHash),
    ).length,
    1,
  );

  const stale = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: { Cookie: `__Host-zrn-pi-session=${first.sessionCookie}` },
    }),
    ENV,
    testHarness.runtime,
  );
  assert.equal(((await stale.json()) as PiSessionEnvelope).authenticated, false);

  const activeTokenHash = hashOpaque(second.sessionCookie);
  const absoluteExpiry = testHarness.repository.sessions.get(
    activeTokenHash,
  )?.expiresAt;
  assert.ok(absoluteExpiry);
  const forwardTouch = startedAt + 10 * 60 * 1_000;
  testHarness.setNow(forwardTouch);
  const touched = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: { Cookie: `__Host-zrn-pi-session=${second.sessionCookie}` },
    }),
    ENV,
    testHarness.runtime,
  );
  assert.equal(((await touched.json()) as PiSessionEnvelope).authenticated, true);
  assert.equal(
    testHarness.repository.sessions.get(activeTokenHash)?.lastSeenAt,
    forwardTouch,
  );

  testHarness.setNow(forwardTouch - 5_000);
  const outOfOrderTouch = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: { Cookie: `__Host-zrn-pi-session=${second.sessionCookie}` },
    }),
    ENV,
    testHarness.runtime,
  );
  assert.equal(
    ((await outOfOrderTouch.json()) as PiSessionEnvelope).authenticated,
    true,
  );
  assert.equal(
    testHarness.repository.sessions.get(activeTokenHash)?.lastSeenAt,
    forwardTouch,
  );

  testHarness.setNow(forwardTouch + 30 * 60 * 1_000);
  const idleBoundary = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: { Cookie: `__Host-zrn-pi-session=${second.sessionCookie}` },
    }),
    ENV,
    testHarness.runtime,
  );
  assert.equal(
    ((await idleBoundary.json()) as PiSessionEnvelope).authenticated,
    false,
  );

  const absoluteHarness = harness(startedAt);
  const absolute = await signIn(absoluteHarness);
  absoluteHarness.setNow(startedAt + 8 * 60 * 60 * 1_000);
  const expired = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: { Cookie: `__Host-zrn-pi-session=${absolute.sessionCookie}` },
    }),
    ENV,
    absoluteHarness.runtime,
  );
  assert.equal(((await expired.json()) as PiSessionEnvelope).authenticated, false);
});

test("pepper rotation preserves canonical identity and fails closed on retirement mistakes", async () => {
  const testHarness = harness(6_000_000);
  const v1 = await signIn(testHarness);
  const v1Subject = testHarness.repository.sessions.get(
    hashOpaque(v1.sessionCookie),
  )?.subjectHash;
  assert.ok(v1Subject);

  const v2Overlap: PiEnv = {
    ...ENV,
    PI_SUBJECT_PEPPER: "q".repeat(64),
    PI_SUBJECT_PEPPER_VERSION: "2",
    PI_SUBJECT_PEPPER_PREVIOUS: JSON.stringify([
      { version: 1, pepper: ENV.PI_SUBJECT_PEPPER },
    ]),
  };
  const overlapMe = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: { Cookie: `__Host-zrn-pi-session=${v1.sessionCookie}` },
    }),
    v2Overlap,
    testHarness.runtime,
  );
  assert.equal(
    ((await overlapMe.json()) as PiSessionEnvelope).authenticated,
    true,
  );

  testHarness.setNow(6_000_100);
  const v2 = await signIn(
    testHarness,
    v2Overlap,
    "pi-access-token-v2-123456789",
  );
  assert.equal(v2.response.status, 200);
  const v2Session = testHarness.repository.sessions.get(
    hashOpaque(v2.sessionCookie),
  );
  assert.equal(v2Session?.pepperVersion, 2);
  assert.equal(v2Session?.subjectHash, v1Subject);

  const v2Only: PiEnv = {
    ...v2Overlap,
    PI_SUBJECT_PEPPER_PREVIOUS: undefined,
  };
  const afterRetirement = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`, {
      headers: { Cookie: `__Host-zrn-pi-session=${v2.sessionCookie}` },
    }),
    v2Only,
    testHarness.runtime,
  );
  assert.equal(
    ((await afterRetirement.json()) as PiSessionEnvelope).authenticated,
    true,
  );

  const mismatchedKey = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`),
    { ...v2Only, PI_SUBJECT_PEPPER: "x".repeat(64) },
    testHarness.runtime,
  );
  assert.equal(mismatchedKey.status, 503);
  const downgrade = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`),
    ENV,
    testHarness.runtime,
  );
  assert.equal(downgrade.status, 503);

  const v1OnlyHarness = harness(6_500_000);
  await signIn(v1OnlyHarness);
  const prematureRetirement = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`),
    v2Only,
    v1OnlyHarness.runtime,
  );
  assert.equal(prematureRetirement.status, 503);

  const malformedPrevious = await handlePiRequest(
    "me",
    new Request(`${ORIGIN}/api/pi/me`),
    { ...ENV, PI_SUBJECT_PEPPER_PREVIOUS: "{}" },
    harness().runtime,
  );
  assert.equal(malformedPrevious.status, 503);
});

test("a paused v1 session cannot mint after the D1 pepper floor advances", async () => {
  const testHarness = harness(7_000_000);
  const flow = await startOAuth(testHarness);
  let markFetchStarted: (() => void) | undefined;
  const fetchStarted = new Promise<void>((resolve) => {
    markFetchStarted = resolve;
  });
  let releaseFetch: (() => void) | undefined;
  const fetchGate = new Promise<void>((resolve) => {
    releaseFetch = resolve;
  });
  testHarness.setFetch(async () => {
    markFetchStarted?.();
    await fetchGate;
    return Response.json({
      uid: RAW_UID,
      username: "pioneer",
      credentials: { scopes: ["username"] },
    });
  });
  const pending = handlePiRequest(
    "session",
    sessionRequest(flow),
    ENV,
    testHarness.runtime,
  );
  await fetchStarted;

  const v2Overlap: PiEnv = {
    ...ENV,
    PI_SUBJECT_PEPPER: "q".repeat(64),
    PI_SUBJECT_PEPPER_VERSION: "2",
    PI_SUBJECT_PEPPER_PREVIOUS: JSON.stringify([
      { version: 1, pepper: ENV.PI_SUBJECT_PEPPER },
    ]),
  };
  const rollout = await handlePiRequest(
    "authorize",
    authorizationRequest(),
    v2Overlap,
    testHarness.runtime,
  );
  assert.equal(rollout.status, 200);
  releaseFetch?.();
  const staleResult = await pending;
  assert.equal(staleResult.status, 409);
  assert.equal(testHarness.repository.sessions.size, 0);
  assert.equal(testHarness.repository.subjectAliases.size, 0);
  assert.equal(testHarness.repository.pendingBearerClaims.size, 0);
  assert.equal(testHarness.repository.bearerFingerprints.size, 1);
});

test("subject deletion epochs order pending /v2/me races at equal timestamps", async () => {
  const base = 8_000_000;
  const testHarness = harness(base);
  const initial = await signIn(testHarness);
  const subjectHash = testHarness.repository.sessions.get(
    hashOpaque(initial.sessionCookie),
  )?.subjectHash;
  assert.ok(subjectHash);

  const deletionAt = base + 100;
  testHarness.setNow(deletionAt);
  const flow = await startOAuth(testHarness);
  let markFetchStarted: (() => void) | undefined;
  const fetchStarted = new Promise<void>((resolve) => {
    markFetchStarted = resolve;
  });
  let releaseFetch: (() => void) | undefined;
  const fetchGate = new Promise<void>((resolve) => {
    releaseFetch = resolve;
  });
  testHarness.setFetch(async () => {
    markFetchStarted?.();
    await fetchGate;
    return Response.json({
      uid: RAW_UID,
      username: "pioneer",
      credentials: { scopes: ["username"] },
    });
  });
  const pending = handlePiRequest(
    "session",
    sessionRequest(flow, "pi-access-token-pending-delete-123"),
    ENV,
    testHarness.runtime,
  );
  await fetchStarted;
  testHarness.setNow(deletionAt);
  assert.equal(
    await testHarness.repository.deleteSubject(
      subjectHash,
      deletionAt,
      deletionAt + PI_SUBJECT_DELETION_GUARD_MS,
      ...memoryPepperConfiguration(testHarness.repository),
      hashOpaque("memory-delete-operation-pending"),
    ),
    true,
  );
  releaseFetch?.();
  assert.equal((await pending).status, 409);
  assert.equal(testHarness.repository.sessions.size, 0);
  assert.equal(testHarness.repository.subjectAliases.size, 0);
  assert.equal(testHarness.repository.pendingBearerClaims.size, 0);
  assert.equal(testHarness.repository.bearerFingerprints.size, 2);
  assert.deepEqual(testHarness.repository.deletionGuards.get(subjectHash), {
    deletedAt: deletionAt,
    expiresAt: deletionAt + 12 * 60 * 1_000,
    deletionEpoch: 1,
    operationHash: hashOpaque("memory-delete-operation-pending"),
  });

  testHarness.setNow(deletionAt + 1);
  const fresh = await signIn(
    testHarness,
    ENV,
    "pi-access-token-after-delete-123456",
  );
  assert.equal(fresh.response.status, 200);
  assert.equal(
    await testHarness.repository.deleteSubject(
      subjectHash,
      deletionAt + 2,
      deletionAt + 2 + PI_SUBJECT_DELETION_GUARD_MS,
      ...memoryPepperConfiguration(testHarness.repository),
      hashOpaque("memory-delete-operation-after-restore"),
    ),
    true,
  );
  assert.equal(testHarness.repository.sessions.size, 0);

  const equalHarness = harness(base);
  const equalInitial = await signIn(equalHarness);
  const equalSubject = equalHarness.repository.sessions.get(
    hashOpaque(equalInitial.sessionCookie),
  )?.subjectHash;
  assert.ok(equalSubject);
  equalHarness.setNow(deletionAt);
  await equalHarness.repository.deleteSubject(
    equalSubject,
    deletionAt,
    deletionAt + PI_SUBJECT_DELETION_GUARD_MS,
    ...memoryPepperConfiguration(equalHarness.repository),
    hashOpaque("memory-delete-operation-equal-timestamp"),
  );
  const equalFlow = await startOAuth(equalHarness);
  const equalTimestamp = await handlePiRequest(
    "session",
    sessionRequest(equalFlow, "pi-access-token-equal-delete-12345"),
    ENV,
    equalHarness.runtime,
  );
  assert.equal(equalTimestamp.status, 200);
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
    const statement = this.database.prepare(this.query);
    const before = Number(
      this.database.prepare(`SELECT total_changes() AS count`).get()?.count ??
        0,
    );
    let results: T[] = [];
    if (statement.columns().length > 0) {
      results = statement.all(...(this.values as never[])) as T[];
    } else {
      statement.run(...(this.values as never[]));
    }
    const after = Number(
      this.database.prepare(`SELECT total_changes() AS count`).get()?.count ??
        before,
    );
    return {
      success: true,
      results,
      meta: { changes: after - before },
    };
  }
}

class SqliteD1Database implements PiD1Database {
  lastBatchResults: readonly PiD1Result<unknown>[] = [];

  constructor(
    readonly database: DatabaseSync,
    private readonly failBatchAt?: number,
  ) {}

  prepare(query: string): PiD1PreparedStatement {
    return new SqliteD1Statement(this.database, query);
  }

  async batch<T = Record<string, unknown>>(
    statements: readonly PiD1PreparedStatement[],
  ): Promise<readonly PiD1Result<T>[]> {
    this.database.exec("BEGIN IMMEDIATE");
    try {
      const results: PiD1Result<T>[] = [];
      for (const [index, statement] of statements.entries()) {
        if (index === this.failBatchAt) {
          throw new Error("injected D1 batch failure");
        }
        results.push(await statement.run<T>());
      }
      this.database.exec("COMMIT");
      this.lastBatchResults = results as readonly PiD1Result<unknown>[];
      return results;
    } catch (error) {
      this.database.exec("ROLLBACK");
      throw error;
    }
  }
}

function lifecycleDatabase(): {
  readonly database: DatabaseSync;
  readonly adapter: SqliteD1Database;
  readonly repository: D1PiRepository;
} {
  const database = new DatabaseSync(":memory:");
  for (const migration of [
    "../migrations/0001_pi_identity.sql",
    "../migrations/0002_pi_identity_lifecycle.sql",
  ]) {
    database.exec(
      readFileSync(new URL(migration, import.meta.url), "utf8"),
    );
  }
  const adapter = new SqliteD1Database(database);
  return {
    database,
    adapter,
    repository: new D1PiRepository(adapter),
  };
}

function repositorySession(
  label: string,
  subjectHash: string,
  now: number,
  pepperVersion = 1,
): PiSession {
  return {
    tokenHash: hashOpaque(`session-${label}`),
    subjectHash,
    username: label,
    pepperVersion,
    lastSeenAt: now,
    expiresAt: now + 8 * 60 * 60 * 1_000,
  };
}

describe("D1 migration and atomic constraints", () => {
  it("atomically reserves state, browser transaction, and SHA bearer commitment once", async () => {
    const database = new DatabaseSync(":memory:");
    database.exec(
      readFileSync(
        new URL("../migrations/0001_pi_identity.sql", import.meta.url),
        "utf8",
      ),
    );
    database.exec(
      readFileSync(
        new URL("../migrations/0002_pi_identity_lifecycle.sql", import.meta.url),
        "utf8",
      ),
    );
    const adapter = new SqliteD1Database(database);
    const repository = new D1PiRepository(adapter);
    const now = 10_000;
    assert.equal(
      await repository.ensurePepperConfiguration(
        1,
        [{ version: 1, fingerprint: hashOpaque("pepper-one") }],
        now,
      ),
      true,
    );
    const stateOne = hashOpaque("state-one");
    const browserOne = hashOpaque("browser-one");
    const bearerOne = hashOpaque("keyed-bearer-one");
    await repository.createOAuthFlow(stateOne, browserOne, now, now + 1_000);

    const bearerTwo = hashOpaque("keyed-bearer-two");
    const firstClaim = await repository.consumeOAuthFlow(
        stateOne,
        browserOne,
        bearerOne,
        1,
        now + 1,
      );
    const consumeResult = adapter.lastBatchResults[0];
    assert.equal(consumeResult?.results?.length, 1);
    assert.ok(
      (consumeResult?.meta?.changes ?? 0) >
        (consumeResult?.results?.length ?? 0),
      "OAuth consume trigger changes must exceed its RETURNING rows",
    );
    const secondClaim = await repository.consumeOAuthFlow(
        stateOne,
        browserOne,
        bearerTwo,
        1,
        now + 1,
      );
    assert.equal(firstClaim, 0);
    assert.equal(secondClaim, null);
    assert.equal(
      await repository.promoteBearerClaim(
        bearerOne,
        stateOne,
        browserOne,
      ),
      true,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_bearer_replay_fingerprints
           WHERE fingerprint_scheme = 'sha256-v1'`,
        )
        .get()?.count,
      1,
    );
    const claimedBearer = bearerOne;
    assert.equal(
      await repository.consumeOAuthFlow(
        stateOne,
        browserOne,
        hashOpaque("other-bearer"),
        1,
        now + 2,
      ),
      null,
    );

    const stateTwo = hashOpaque("state-two");
    const browserTwo = hashOpaque("browser-two");
    await repository.createOAuthFlow(stateTwo, browserTwo, now, now + 1_000);
    assert.equal(
      await repository.consumeOAuthFlow(
        stateTwo,
        browserTwo,
        claimedBearer,
        1,
        now + 3,
      ),
      null,
    );
    assert.equal(
      await repository.consumeOAuthFlow(
        stateTwo,
        browserTwo,
        hashOpaque("keyed-bearer-three"),
        1,
        now + 4,
      ),
      0,
    );
    database.close();
  });

  it("keeps reservations ABA-safe and expires only unverified pending claims at two minutes", async () => {
    const { database, repository } = lifecycleDatabase();
    const now = 15_000;
    await repository.ensurePepperConfiguration(
      1,
      [{ version: 1, fingerprint: hashOpaque("reservation-pin") }],
      now,
    );
    const commitment = hashOpaque("sha-commitment-aba");
    const stateOne = hashOpaque("aba-state-one");
    const browserOne = hashOpaque("aba-browser-one");
    const stateTwo = hashOpaque("aba-state-two");
    const browserTwo = hashOpaque("aba-browser-two");
    await repository.createOAuthFlow(stateOne, browserOne, now, now + 10_000);
    await repository.createOAuthFlow(stateTwo, browserTwo, now, now + 10_000);
    assert.equal(
      await repository.consumeOAuthFlow(
        stateOne,
        browserOne,
        commitment,
        1,
        now,
      ),
      0,
    );
    assert.equal(
      await repository.consumeOAuthFlow(
        stateTwo,
        browserTwo,
        commitment,
        1,
        now,
      ),
      null,
    );
    await repository.rejectBearerClaim(commitment, stateTwo, browserTwo);
    assert.equal(
      database
        .prepare(`SELECT COUNT(*) AS count FROM pi_bearer_claims`)
        .get()?.count,
      1,
    );
    await repository.rejectBearerClaim(commitment, stateOne, browserOne);
    assert.equal(
      database
        .prepare(`SELECT COUNT(*) AS count FROM pi_bearer_claims`)
        .get()?.count,
      0,
    );
    assert.equal(
      await repository.consumeOAuthFlow(
        stateTwo,
        browserTwo,
        commitment,
        1,
        now + 1,
      ),
      0,
    );
    await repository.rejectBearerClaim(commitment, stateOne, browserOne);
    assert.equal(
      database
        .prepare(`SELECT state_hash FROM pi_bearer_claims`)
        .get()?.state_hash,
      stateTwo,
    );
    assert.equal(
      await repository.promoteBearerClaim(
        commitment,
        stateTwo,
        browserTwo,
      ),
      true,
    );

    const pendingCommitment = hashOpaque("sha-commitment-timeout");
    const pendingState = hashOpaque("pending-state");
    const pendingBrowser = hashOpaque("pending-browser");
    await repository.createOAuthFlow(
      pendingState,
      pendingBrowser,
      now + 2,
      now + 20_000,
    );
    assert.equal(
      await repository.consumeOAuthFlow(
        pendingState,
        pendingBrowser,
        pendingCommitment,
        1,
        now + 2,
      ),
      0,
    );
    const beforeTimeout = await repository.cleanupExpired({
      now: now + 2 + PI_BEARER_PENDING_TTL_MS - 1,
      pendingBefore: now + 1,
      idleBefore: now,
      retainedBefore: 0,
      maximumRowsPerTable: 100,
    });
    assert.equal(beforeTimeout.bearerPendingClaimsDeleted, 0);
    const atTimeout = await repository.cleanupExpired({
      now: now + 2 + PI_BEARER_PENDING_TTL_MS,
      pendingBefore: now + 2,
      idleBefore: now,
      retainedBefore: 0,
      maximumRowsPerTable: 100,
    });
    assert.equal(atTimeout.bearerPendingClaimsDeleted, 1);
    assert.equal(
      await repository.promoteBearerClaim(
        pendingCommitment,
        pendingState,
        pendingBrowser,
      ),
      false,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_bearer_replay_fingerprints
           WHERE fingerprint_scheme = 'sha256-v1'`,
        )
        .get()?.count,
      1,
    );
    const rollbackCommitment = hashOpaque("rollback-commitment");
    const d1 = new SqliteD1Database(database);
    await assert.rejects(
      d1.batch([
        d1
          .prepare(
            `INSERT INTO pi_bearer_replay_fingerprints
               (fingerprint_scheme, fingerprint, first_claimed_at)
             VALUES ('sha256-v1', ?, ?)`,
          )
          .bind(rollbackCommitment, now),
        d1
          .prepare(
            `INSERT INTO pi_bearer_replay_fingerprints
               (fingerprint_scheme, fingerprint, first_claimed_at)
             VALUES ('sha256-v1', 'too-short', ?)`,
          )
          .bind(now),
      ]),
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_bearer_replay_fingerprints
           WHERE fingerprint = ?`,
        )
        .get(rollbackCommitment)?.count,
      0,
    );
    database.close();
  });

  it("latches pre-migration bearer evidence and rejects new legacy claims", async () => {
    const database = new DatabaseSync(":memory:");
    database.exec(
      readFileSync(
        new URL("../migrations/0001_pi_identity.sql", import.meta.url),
        "utf8",
      ),
    );
    const now = 30_000;
    const legacyState = hashOpaque("legacy-state");
    const legacyBrowser = hashOpaque("legacy-browser");
    database
      .prepare(
        `INSERT INTO pi_oauth_flows
           (state_hash, browser_hash, created_at, expires_at)
         VALUES (?, ?, ?, ?)`,
      )
      .run(legacyState, legacyBrowser, now, now + 10_000);
    database
      .prepare(
        `INSERT INTO pi_bearer_claims
           (fingerprint, state_hash, browser_hash, claimed_at)
         VALUES (?, ?, ?, ?)`,
      )
      .run(hashOpaque("legacy-keyed-claim"), legacyState, legacyBrowser, now);
    database.exec(
      readFileSync(
        new URL("../migrations/0002_pi_identity_lifecycle.sql", import.meta.url),
        "utf8",
      ),
    );
    const repository = new D1PiRepository(new SqliteD1Database(database));
    assert.equal(
      database
        .prepare(`SELECT COUNT(*) AS count FROM pi_bearer_legacy_state`)
        .get()?.count,
      1,
    );
    assert.equal(
      await repository.ensurePepperConfiguration(
        1,
        [{ version: 1, fingerprint: hashOpaque("legacy-pin") }],
        now + 1,
      ),
      false,
    );
    const cleanup = await repository.cleanupExpired({
      now: now + 1_000_000,
      pendingBefore: now + 1_000_000 - PI_BEARER_PENDING_TTL_MS,
      idleBefore: now + 1_000_000,
      retainedBefore: now + 1_000_000,
      maximumRowsPerTable: 100,
    });
    assert.equal(cleanup.bearerPendingClaimsDeleted, 0);
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_bearer_claims
           WHERE fingerprint_scheme = 'legacy-keyed-v1'`,
        )
        .get()?.count,
      1,
    );

    const newState = hashOpaque("new-legacy-state");
    const newBrowser = hashOpaque("new-legacy-browser");
    database
      .prepare(
        `INSERT INTO pi_oauth_flows
           (state_hash, browser_hash, created_at, expires_at)
         VALUES (?, ?, ?, ?)`,
      )
      .run(newState, newBrowser, now + 2, now + 20_000);
    assert.throws(
      () =>
        database
          .prepare(
            `INSERT INTO pi_bearer_claims
               (fingerprint, state_hash, browser_hash, claimed_at)
             VALUES (?, ?, ?, ?)`,
          )
          .run(
            hashOpaque("new-legacy-keyed-claim"),
            newState,
            newBrowser,
            now + 2,
          ),
      /legacy Pi bearer claim scheme is closed/u,
    );
    database.close();
  });

  it("pins monotonic keys, preserves rolling v1 inserts, and enforces the atomic floor", async () => {
    const { database, repository } = lifecycleDatabase();
    const now = 20_000;
    const pinOne = { version: 1, fingerprint: hashOpaque("pepper-pin-one") };
    const pinTwo = { version: 2, fingerprint: hashOpaque("pepper-pin-two") };
    const v2OverlapFingerprint = piPepperKeysetFingerprint(2, [
      pinTwo,
      pinOne,
    ]);
    const v2OnlyFingerprint = piPepperKeysetFingerprint(2, [pinTwo]);
    assert.equal(
      await repository.ensurePepperConfiguration(1, [pinOne], now),
      true,
    );

    const legacySubject = hashOpaque("legacy-subject");
    const legacyToken = hashOpaque("legacy-token");
    database
      .prepare(
        `INSERT INTO pi_sessions
           (token_hash, subject_hash, username, created_at, expires_at)
         VALUES (?, ?, 'legacy', ?, ?)`,
      )
      .run(legacyToken, legacySubject, now, now + 10_000);
    assert.equal(
      database
        .prepare(
          `SELECT last_seen_at FROM pi_sessions WHERE token_hash = ?`,
        )
        .get(legacyToken)?.last_seen_at,
      now,
    );
    assert.equal(
      database
        .prepare(
          `SELECT canonical_subject_hash
           FROM pi_subject_aliases
           WHERE pepper_version = 1 AND alias_hash = ?`,
        )
        .get(legacySubject)?.canonical_subject_hash,
      legacySubject,
    );

    assert.equal(
      await repository.ensurePepperConfiguration(
        2,
        [pinTwo, pinOne],
        now + 1,
      ),
      true,
    );
    const v2Alias = hashOpaque("legacy-subject-v2-alias");
    assert.equal(
      await repository.resolveSubject([
        { pepperVersion: 2, aliasHash: v2Alias },
        { pepperVersion: 1, aliasHash: legacySubject },
      ]),
      legacySubject,
    );
    const v2Session = repositorySession(
      "legacy-v2",
      legacySubject,
      now + 2,
      2,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        v2Session,
        [
          { pepperVersion: 2, aliasHash: v2Alias },
          { pepperVersion: 1, aliasHash: legacySubject },
        ],
        0,
        now + 2,
        v2OverlapFingerprint,
      ),
      true,
    );
    assert.equal(
      await repository.ensurePepperConfiguration(2, [pinTwo], now + 3),
      true,
    );
    assert.equal(
      await repository.resolveSubject([
        { pepperVersion: 2, aliasHash: v2Alias },
      ]),
      legacySubject,
    );

    assert.throws(
      () =>
        database
          .prepare(
            `INSERT INTO pi_sessions
               (token_hash, subject_hash, username, created_at, expires_at)
             VALUES (?, ?, 'stale', ?, ?)`,
          )
          .run(
            hashOpaque("stale-default-v1-token"),
            hashOpaque("stale-default-v1-subject"),
            now + 4,
            now + 10_000,
          ),
      /pepper floor mismatch/u,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("stale-v1", legacySubject, now + 4, 1),
        [{ pepperVersion: 1, aliasHash: legacySubject }],
        0,
        now + 4,
        v2OnlyFingerprint,
      ),
      false,
    );
    assert.equal(
      await repository.ensurePepperConfiguration(1, [pinOne], now + 4),
      false,
    );
    assert.equal(
      await repository.ensurePepperConfiguration(
        2,
        [{ version: 2, fingerprint: hashOpaque("wrong-pin-two") }],
        now + 4,
      ),
      false,
    );

    const forward = await repository.getSession(
      v2Session.tokenHash,
      now + 100,
      0,
    );
    assert.equal(forward?.lastSeenAt, now + 100);
    const outOfOrder = await repository.getSession(
      v2Session.tokenHash,
      now + 50,
      0,
    );
    assert.equal(outOfOrder?.lastSeenAt, now + 100);
    database.close();
  });

  it("rotates real D1 bind/unlink families and rolls back conflicting provisional sessions", async () => {
    const { database, adapter, repository } = lifecycleDatabase();
    const now = 35_000;
    const bindingPin = {
      version: 1,
      fingerprint: hashOpaque("binding-pin"),
    };
    const bindingKeysetFingerprint = piPepperKeysetFingerprint(1, [
      bindingPin,
    ]);
    await repository.ensurePepperConfiguration(
      1,
      [bindingPin],
      now,
    );
    const subjectOne = hashOpaque("binding-subject-one");
    const subjectTwo = hashOpaque("binding-subject-two");
    const sessionOne = repositorySession("binding-one", subjectOne, now);
    const sessionTwo = repositorySession("binding-two", subjectTwo, now);
    assert.equal(
      await repository.replaceSubjectSessions(
        sessionOne,
        [{ pepperVersion: 1, aliasHash: subjectOne }],
        0,
        now,
        bindingKeysetFingerprint,
      ),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        sessionTwo,
        [{ pepperVersion: 1, aliasHash: subjectTwo }],
        0,
        now,
        bindingKeysetFingerprint,
      ),
      true,
    );

    const address = `zrn1${"r".repeat(38)}`;
    const challenge: PiChallenge = {
      idHash: hashOpaque("binding-challenge"),
      sessionHash: sessionOne.tokenHash,
      subjectHash: subjectOne,
      address,
      accountId: `cosmos:zerone-1:${address}`,
      message: "binding challenge",
      createdAt: now + 1,
      expiresAt: now + 1_000,
    };
    const superseded: PiChallenge = {
      ...challenge,
      idHash: hashOpaque("binding-superseded"),
      message: "superseded challenge",
    };
    assert.equal(
      await repository.createChallenge(challenge, now - 1_000, 5, now - 1_000),
      true,
    );
    assert.equal(
      await repository.createChallenge(
        superseded,
        now - 1_000,
        5,
        now - 1_000,
      ),
      true,
    );
    const linkedSession = repositorySession("binding-linked", subjectOne, now + 2);
    assert.ok(
      await repository.bindChallenge(
        challenge.idHash,
        sessionOne.tokenHash,
        subjectOne,
        hashOpaque("binding-proof"),
        "pi-wallet-link-v1",
        linkedSession,
        now - 1_000,
        now + 2,
      ),
    );
    const bindingResult = adapter.lastBatchResults[1];
    assert.equal(bindingResult?.results?.length, 1);
    assert.ok(
      (bindingResult?.meta?.changes ?? 0) >
        (bindingResult?.results?.length ?? 0),
      "binding trigger changes must exceed its RETURNING rows",
    );
    assert.equal(
      await repository.bindChallenge(
        challenge.idHash,
        sessionOne.tokenHash,
        subjectOne,
        hashOpaque("binding-proof-replay"),
        "pi-wallet-link-v1",
        repositorySession("binding-replay", subjectOne, now + 3),
        now - 1_000,
        now + 3,
      ),
      null,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count FROM pi_sessions
           WHERE subject_hash = ? AND revoked_at IS NULL`,
        )
        .get(subjectOne)?.count,
      1,
    );
    assert.deepEqual(
      database
        .prepare(
          `SELECT disposition FROM pi_wallet_challenge_uses
           ORDER BY disposition`,
        )
        .all()
        .map((row) => row.disposition),
      ["bound", "superseded"],
    );

    const unlinkedSession = repositorySession(
      "binding-unlinked",
      subjectOne,
      now + 4,
    );
    assert.equal(
      await repository.deleteBinding(
        subjectOne,
        linkedSession.tokenHash,
        unlinkedSession,
        now - 1_000,
        now + 4,
      ),
      true,
    );
    assert.equal(await repository.getBinding(subjectOne), null);
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_wallet_challenge_uses
           WHERE disposition = 'unlinked'`,
        )
        .get()?.count,
      0,
    );
    assert.equal(
      database
        .prepare(
          `SELECT token_hash FROM pi_sessions
           WHERE subject_hash = ? AND revoked_at IS NULL`,
        )
        .get(subjectOne)?.token_hash,
      unlinkedSession.tokenHash,
    );
    const redundantUnlinkSession = repositorySession(
      "binding-redundant-unlink",
      subjectOne,
      now + 5,
    );
    assert.equal(
      await repository.deleteBinding(
        subjectOne,
        unlinkedSession.tokenHash,
        redundantUnlinkSession,
        now - 1_000,
        now + 5,
      ),
      false,
    );
    assert.equal(
      database
        .prepare(
          `SELECT token_hash FROM pi_sessions
           WHERE subject_hash = ? AND revoked_at IS NULL`,
        )
        .get(subjectOne)?.token_hash,
      unlinkedSession.tokenHash,
    );
    assert.equal(
      database
        .prepare(`SELECT COUNT(*) AS count FROM pi_sessions WHERE token_hash = ?`)
        .get(redundantUnlinkSession.tokenHash)?.count,
      0,
    );

    const reconsentChallenge: PiChallenge = {
      ...challenge,
      idHash: hashOpaque("binding-reconsent"),
      sessionHash: unlinkedSession.tokenHash,
      message: "binding reconsent challenge",
      createdAt: now + 5,
    };
    assert.equal(
      await repository.createChallenge(
        reconsentChallenge,
        now - 1_000,
        5,
        now - 1_000,
      ),
      true,
    );
    for (const [label, message] of [
      ["binding-rate-four", "binding rate four"],
      ["binding-rate-five", "binding rate five"],
    ] as const) {
      assert.equal(
        await repository.createChallenge(
          {
            ...reconsentChallenge,
            idHash: hashOpaque(label),
            message,
          },
          now - 1_000,
          5,
          now - 1_000,
        ),
        true,
      );
    }
    assert.equal(
      await repository.createChallenge(
        {
          ...reconsentChallenge,
          idHash: hashOpaque("binding-rate-six"),
          message: "binding rate six",
        },
        now - 1_000,
        5,
        now - 1_000,
      ),
      false,
    );
    const reconsentedSession = repositorySession(
      "binding-reconsented",
      subjectOne,
      now + 6,
    );
    assert.ok(
      await repository.bindChallenge(
        reconsentChallenge.idHash,
        unlinkedSession.tokenHash,
        subjectOne,
        hashOpaque("binding-reconsent-proof"),
        "pi-wallet-link-v1",
        reconsentedSession,
        now - 1_000,
        now + 6,
      ),
    );

    const conflictingChallenge: PiChallenge = {
      ...challenge,
      idHash: hashOpaque("binding-conflict"),
      sessionHash: sessionTwo.tokenHash,
      subjectHash: subjectTwo,
      message: "conflicting address challenge",
      createdAt: now + 7,
    };
    assert.equal(
      await repository.createChallenge(
        conflictingChallenge,
        now - 1_000,
        5,
        now - 1_000,
      ),
      true,
    );
    const conflictingFresh = repositorySession(
      "binding-conflict-fresh",
      subjectTwo,
      now + 8,
    );
    assert.equal(
      await repository.bindChallenge(
        conflictingChallenge.idHash,
        sessionTwo.tokenHash,
        subjectTwo,
        hashOpaque("binding-conflict-proof"),
        "pi-wallet-link-v1",
        conflictingFresh,
        now - 1_000,
        now + 8,
      ),
      null,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count FROM pi_sessions
           WHERE token_hash = ?`,
        )
        .get(conflictingFresh.tokenHash)?.count,
      0,
    );
    assert.equal(
      database
        .prepare(
          `SELECT token_hash FROM pi_sessions
           WHERE subject_hash = ? AND revoked_at IS NULL`,
        )
        .get(subjectTwo)?.token_hash,
      sessionTwo.tokenHash,
    );
    database.close();
  });

  it("deletes a canonical subject atomically and guards stale OAuth resurrection", async () => {
    const { database, repository } = lifecycleDatabase();
    const now = 40_000;
    const pin = { version: 1, fingerprint: hashOpaque("delete-pin-one") };
    const keysetFingerprint = piPepperKeysetFingerprint(1, [pin]);
    await repository.ensurePepperConfiguration(1, [pin], now);
    const subjectHash = hashOpaque("delete-subject");
    const alias = { pepperVersion: 1, aliasHash: subjectHash };
    const session = repositorySession("delete", subjectHash, now);
    assert.equal(
      await repository.replaceSubjectSessions(
        session,
        [alias],
        0,
        now,
        keysetFingerprint,
      ),
      true,
    );

    const address = `zrn1${"q".repeat(38)}`;
    const challenge: PiChallenge = {
      idHash: hashOpaque("delete-challenge"),
      sessionHash: session.tokenHash,
      subjectHash,
      address,
      accountId: `cosmos:zerone-1:${address}`,
      message: "delete challenge",
      createdAt: now + 1,
      expiresAt: now + 1_000,
    };
    assert.equal(
      await repository.createChallenge(challenge, now - 1_000, 5, now - 1_000),
      true,
    );
    const rotated = repositorySession("delete-bound", subjectHash, now + 2);
    assert.ok(
      await repository.bindChallenge(
        challenge.idHash,
        session.tokenHash,
        subjectHash,
        hashOpaque("delete-proof"),
        "pi-wallet-link-v1",
        rotated,
        now - 1_000,
        now + 2,
      ),
    );

    const deletedAt = now + 3;
    const guardExpiry = deletedAt + PI_SUBJECT_DELETION_GUARD_MS;
    const failingRepository = new D1PiRepository(
      new SqliteD1Database(database, 3),
    );
    await assert.rejects(
      failingRepository.deleteSubject(
        subjectHash,
        deletedAt,
        guardExpiry,
        1,
        keysetFingerprint,
        hashOpaque("delete-operation-injected-failure"),
      ),
      /injected D1 batch failure/u,
    );
    assert.equal(
      database
        .prepare(`SELECT COUNT(*) AS count FROM pi_subject_deletion_guards`)
        .get()?.count,
      0,
    );
    assert.equal(await repository.getBinding(subjectHash) !== null, true);
    assert.equal(
      await repository.deleteSubject(
        subjectHash,
        deletedAt,
        guardExpiry,
        1,
        keysetFingerprint,
        hashOpaque("delete-operation-atomic-success"),
      ),
      true,
    );
    for (const table of [
      "pi_sessions",
      "pi_wallet_challenges",
      "pi_wallet_bindings",
      "pi_wallet_challenge_uses",
      "pi_subject_aliases",
    ]) {
      assert.equal(
        database.prepare(`SELECT COUNT(*) AS count FROM ${table}`).get()?.count,
        0,
      );
    }

    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("stale-delete", subjectHash, deletedAt + 1),
        [alias],
        0,
        deletedAt + 1,
        keysetFingerprint,
      ),
      false,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("post-delete-epoch", subjectHash, deletedAt + 1),
        [alias],
        1,
        deletedAt + 1,
        keysetFingerprint,
      ),
      true,
    );
    const fresh = repositorySession("fresh-delete", subjectHash, deletedAt + 2);
    assert.equal(
      await repository.replaceSubjectSessions(
        fresh,
        [alias],
        1,
        deletedAt + 2,
        keysetFingerprint,
      ),
      true,
    );

    const beforeBoundary = await repository.cleanupExpired({
      now: guardExpiry - 1,
      pendingBefore: guardExpiry - 1 - PI_BEARER_PENDING_TTL_MS,
      idleBefore: guardExpiry - 1,
      retainedBefore: deletedAt,
      maximumRowsPerTable: 100,
    });
    assert.equal(beforeBoundary.deletionGuardsDeleted, 0);
    const atBoundary = await repository.cleanupExpired({
      now: guardExpiry,
      pendingBefore: guardExpiry - PI_BEARER_PENDING_TTL_MS,
      idleBefore: guardExpiry,
      retainedBefore: deletedAt,
      maximumRowsPerTable: 100,
    });
    assert.equal(atBoundary.deletionGuardsDeleted, 1);
    database.close();
  });

  it("orders migrated and fresh OAuth flows by the global deletion epoch", async () => {
    const database = new DatabaseSync(":memory:");
    database.exec(
      readFileSync(
        new URL("../migrations/0001_pi_identity.sql", import.meta.url),
        "utf8",
      ),
    );
    const now = 50_000;
    const deletedAt = now + 1;
    const migratedState = hashOpaque("migrated-epoch-state");
    const migratedBrowser = hashOpaque("migrated-epoch-browser");
    database
      .prepare(
        `INSERT INTO pi_oauth_flows
           (state_hash, browser_hash, created_at, expires_at)
         VALUES (?, ?, ?, ?)`,
      )
      .run(migratedState, migratedBrowser, deletedAt, deletedAt + 10_000);
    database.exec(
      readFileSync(
        new URL("../migrations/0002_pi_identity_lifecycle.sql", import.meta.url),
        "utf8",
      ),
    );
    assert.equal(
      database
        .prepare(`SELECT deletion_epoch FROM pi_oauth_flows WHERE state_hash = ?`)
        .get(migratedState)?.deletion_epoch,
      0,
    );

    const repository = new D1PiRepository(new SqliteD1Database(database));
    const pin = { version: 1, fingerprint: hashOpaque("epoch-order-pin") };
    const keysetFingerprint = piPepperKeysetFingerprint(1, [pin]);
    const subjectHash = hashOpaque("epoch-order-subject");
    const alias = { pepperVersion: 1, aliasHash: subjectHash };
    assert.equal(
      await repository.ensurePepperConfiguration(1, [pin], now),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("epoch-order-initial", subjectHash, now),
        [alias],
        0,
        now,
        keysetFingerprint,
      ),
      true,
    );
    assert.equal(
      await repository.deleteSubject(
        subjectHash,
        deletedAt,
        deletedAt + PI_SUBJECT_DELETION_GUARD_MS,
        1,
        keysetFingerprint,
        hashOpaque("delete-operation-epoch-order"),
      ),
      true,
    );
    assert.equal(
      database
        .prepare(`SELECT current_epoch FROM pi_identity_deletion_epoch`)
        .get()?.current_epoch,
      1,
    );

    const migratedCommitment = hashOpaque("migrated-epoch-commitment");
    assert.equal(
      await repository.consumeOAuthFlow(
        migratedState,
        migratedBrowser,
        migratedCommitment,
        1,
        deletedAt,
      ),
      0,
    );
    assert.equal(
      await repository.promoteBearerClaim(
        migratedCommitment,
        migratedState,
        migratedBrowser,
      ),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("epoch-order-migrated", subjectHash, deletedAt),
        [alias],
        0,
        deletedAt,
        keysetFingerprint,
      ),
      false,
    );

    const freshState = hashOpaque("fresh-epoch-state");
    const freshBrowser = hashOpaque("fresh-epoch-browser");
    await repository.createOAuthFlow(
      freshState,
      freshBrowser,
      deletedAt,
      deletedAt + 10_000,
    );
    const freshCommitment = hashOpaque("fresh-epoch-commitment");
    assert.equal(
      await repository.consumeOAuthFlow(
        freshState,
        freshBrowser,
        freshCommitment,
        1,
        deletedAt,
      ),
      1,
    );
    assert.equal(
      await repository.promoteBearerClaim(
        freshCommitment,
        freshState,
        freshBrowser,
      ),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("epoch-order-fresh", subjectHash, deletedAt),
        [alias],
        1,
        deletedAt,
        keysetFingerprint,
      ),
      true,
    );
    database.close();
  });

  it("advances deletion epochs despite an inverted clock and preserves alias guards", async () => {
    const { database, repository } = lifecycleDatabase();
    const now = 80_000;
    const v1Pin = { version: 1, fingerprint: hashOpaque("epoch-v1-pin") };
    const v2Pin = { version: 2, fingerprint: hashOpaque("epoch-v2-pin") };
    const pins = [v2Pin, v1Pin] as const;
    const keysetFingerprint = piPepperKeysetFingerprint(2, pins);
    const v1Subject = hashOpaque("epoch-subject-v1");
    const v2Subject = hashOpaque("epoch-subject-v2");
    const aliases = [
      { pepperVersion: 2, aliasHash: v2Subject },
      { pepperVersion: 1, aliasHash: v1Subject },
    ] as const;
    assert.equal(
      await repository.ensurePepperConfiguration(2, pins, now),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("epoch-inverted-initial", v2Subject, now, 2),
        aliases,
        0,
        now,
        keysetFingerprint,
      ),
      true,
    );

    const firstDeletedAt = now + 10_000;
    const firstGuardExpiry =
      firstDeletedAt + PI_SUBJECT_DELETION_GUARD_MS;
    assert.equal(
      await repository.deleteSubject(
        v2Subject,
        firstDeletedAt,
        firstGuardExpiry,
        2,
        keysetFingerprint,
        hashOpaque("delete-operation-inverted-first"),
      ),
      true,
    );
    const betweenState = hashOpaque("epoch-between-state");
    const betweenBrowser = hashOpaque("epoch-between-browser");
    await repository.createOAuthFlow(
      betweenState,
      betweenBrowser,
      firstDeletedAt,
      firstGuardExpiry,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession(
          "epoch-inverted-recreated",
          v2Subject,
          firstDeletedAt + 1,
          2,
        ),
        aliases,
        1,
        firstDeletedAt + 1,
        keysetFingerprint,
      ),
      true,
    );

    const secondDeletedAt = firstDeletedAt - 1_000;
    const secondGuardExpiry = firstGuardExpiry - 1_000;
    assert.equal(
      await repository.deleteSubject(
        v2Subject,
        secondDeletedAt,
        secondGuardExpiry,
        2,
        keysetFingerprint,
        hashOpaque("delete-operation-inverted-second"),
      ),
      true,
    );
    const guards = database
      .prepare(
        `SELECT subject_hash, deleted_at, expires_at, deletion_epoch
         FROM pi_subject_deletion_guards
         ORDER BY subject_hash`,
      )
      .all() as Array<Record<string, unknown>>;
    assert.equal(guards.length, 2);
    for (const guard of guards) {
      assert.equal(guard.deleted_at, firstDeletedAt);
      assert.equal(guard.expires_at, firstGuardExpiry);
      assert.equal(guard.deletion_epoch, 2);
    }
    assert.deepEqual(
      new Set(guards.map((guard) => guard.subject_hash)),
      new Set([v1Subject, v2Subject]),
    );

    const betweenCommitment = hashOpaque("epoch-between-commitment");
    assert.equal(
      await repository.consumeOAuthFlow(
        betweenState,
        betweenBrowser,
        betweenCommitment,
        2,
        firstDeletedAt + 2,
      ),
      1,
    );
    assert.equal(
      await repository.promoteBearerClaim(
        betweenCommitment,
        betweenState,
        betweenBrowser,
      ),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession(
          "epoch-inverted-stale-between",
          v2Subject,
          firstDeletedAt + 2,
          2,
        ),
        aliases,
        1,
        firstDeletedAt + 2,
        keysetFingerprint,
      ),
      false,
    );
    database.close();
  });

  it("freezes the configured pepper keyset while a deletion guard is live", async () => {
    const { database, repository } = lifecycleDatabase();
    const now = 60_000;
    const v1Pepper = "p".repeat(64);
    const v2Pepper = "q".repeat(64);
    const v1Pin = {
      version: 1,
      fingerprint: await keyedHash(
        v1Pepper,
        `zerone-pi-pepper-key-fingerprint-v1\u0000${ORIGIN}`,
      ),
    };
    const v2Pin = {
      version: 2,
      fingerprint: await keyedHash(
        v2Pepper,
        `zerone-pi-pepper-key-fingerprint-v1\u0000${ORIGIN}`,
      ),
    };
    const v1Subject = await keyedHash(
      v1Pepper,
      `zerone-pi-subject-v1\u0000${RAW_UID}`,
    );
    const v2Subject = await keyedHash(
      v2Pepper,
      `zerone-pi-subject-v1\u0000${RAW_UID}`,
    );
    const v1Alias = { pepperVersion: 1, aliasHash: v1Subject };
    const v2Alias = { pepperVersion: 2, aliasHash: v2Subject };

    assert.equal(
      await repository.ensurePepperConfiguration(1, [v1Pin], now),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("delete-keyset-v1", v1Subject, now),
        [v1Alias],
        0,
        now,
        piPepperKeysetFingerprint(1, [v1Pin]),
      ),
      true,
    );
    assert.equal(
      await repository.ensurePepperConfiguration(
        2,
        [v2Pin, v1Pin],
        now + 1,
      ),
      true,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_subject_aliases
           WHERE pepper_version = 1
             AND alias_hash = ?`,
        )
        .get(v1Subject)?.count,
      1,
    );

    const deletedAt = now + 2;
    const guardExpiry = deletedAt + PI_SUBJECT_DELETION_GUARD_MS;
    assert.equal(
      await repository.deleteSubject(
        v1Subject,
        deletedAt,
        guardExpiry,
        1,
        piPepperKeysetFingerprint(1, [v1Pin]),
        hashOpaque("delete-operation-config-first-stale"),
      ),
      false,
    );
    assert.equal(
      database
        .prepare(`SELECT COUNT(*) AS count FROM pi_subject_deletion_guards`)
        .get()?.count,
      0,
    );
    assert.equal(
      database.prepare(`SELECT COUNT(*) AS count FROM pi_sessions`).get()
        ?.count,
      1,
    );
    assert.equal(
      database.prepare(`SELECT COUNT(*) AS count FROM pi_subject_aliases`).get()
        ?.count,
      1,
    );
    assert.equal(
      await repository.deleteSubject(
        v1Subject,
        deletedAt,
        guardExpiry,
        2,
        piPepperKeysetFingerprint(2, [v2Pin, v1Pin]),
        hashOpaque("delete-operation-config-first-current"),
      ),
      true,
    );
    assert.equal(
      await repository.ensurePepperConfiguration(
        2,
        [v2Pin],
        deletedAt + 1,
      ),
      false,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession(
          "delete-keyset-stale-v2-only",
          v2Subject,
          deletedAt + 1,
          2,
        ),
        [v2Alias],
        0,
        deletedAt + 1,
        piPepperKeysetFingerprint(2, [v2Pin]),
      ),
      false,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession(
          "delete-keyset-stale-overlap",
          v2Subject,
          deletedAt + 1,
          2,
        ),
        [v2Alias, v1Alias],
        0,
        deletedAt + 1,
        piPepperKeysetFingerprint(2, [v2Pin, v1Pin]),
      ),
      false,
    );
    const staleState = "S".repeat(43);
    const staleBrowser = "B".repeat(43);
    await repository.createOAuthFlow(
      hashOpaque(staleState),
      hashOpaque(staleBrowser),
      deletedAt - 1,
      guardExpiry,
    );
    let fetchCalls = 0;
    const v2OnlyResponse = await handlePiRequest(
      "session",
      new Request(`${ORIGIN}/api/pi/session`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Cookie: `__Host-zrn-pi-oauth=${staleBrowser}`,
          Origin: ORIGIN,
          "Sec-Fetch-Site": "same-origin",
        },
        body: JSON.stringify({
          accessToken: "pi-access-token-stale-v2-only-12345",
          state: staleState,
        }),
      }),
      {
        ...ENV,
        PI_SUBJECT_PEPPER: v2Pepper,
        PI_SUBJECT_PEPPER_VERSION: "2",
        PI_SUBJECT_PEPPER_PREVIOUS: "[]",
      },
      {
        repository,
        now: () => deletedAt + 1,
        randomBytes: (length) => new Uint8Array(length),
        fetch: async () => {
          fetchCalls += 1;
          return Response.json({ uid: RAW_UID, username: "pioneer" });
        },
      },
    );
    assert.equal(v2OnlyResponse.status, 503);
    assert.equal(fetchCalls, 0);
    assert.equal(
      database.prepare(`SELECT COUNT(*) AS count FROM pi_sessions`).get()
        ?.count,
      0,
    );

    const cleanup = await repository.cleanupExpired({
      now: guardExpiry,
      pendingBefore: guardExpiry - PI_BEARER_PENDING_TTL_MS,
      idleBefore: guardExpiry,
      retainedBefore: deletedAt,
      maximumRowsPerTable: 100,
    });
    assert.equal(cleanup.deletionGuardsDeleted, 1);
    assert.equal(
      await repository.ensurePepperConfiguration(2, [v2Pin], guardExpiry),
      true,
    );
    database.close();
  });

  it("keeps restored subject data when a stale-keyset deletion precondition fails", async () => {
    const { database, repository } = lifecycleDatabase();
    const now = 500_000;
    const v1Pin = {
      version: 1,
      fingerprint: hashOpaque("stale-delete-v1-pin"),
    };
    const v2Pin = {
      version: 2,
      fingerprint: hashOpaque("stale-delete-v2-pin"),
    };
    const v1Keyset = piPepperKeysetFingerprint(1, [v1Pin]);
    const v2Keyset = piPepperKeysetFingerprint(2, [v2Pin, v1Pin]);
    const subjectHash = hashOpaque("stale-delete-restored-subject");
    const alias = { pepperVersion: 1, aliasHash: subjectHash };

    assert.equal(
      await repository.ensurePepperConfiguration(1, [v1Pin], now),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        repositorySession("stale-delete-initial", subjectHash, now),
        [alias],
        0,
        now,
        v1Keyset,
      ),
      true,
    );

    const deletedAt = now + 1;
    const guardExpiry = deletedAt + PI_SUBJECT_DELETION_GUARD_MS;
    const originalOperation = hashOpaque("stale-delete-original-operation");
    assert.equal(
      await repository.deleteSubject(
        subjectHash,
        deletedAt,
        guardExpiry,
        1,
        v1Keyset,
        originalOperation,
      ),
      true,
    );

    const restoredAt = guardExpiry + 1;
    const restoredSession = repositorySession(
      "stale-delete-restored",
      subjectHash,
      restoredAt,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        restoredSession,
        [alias],
        1,
        restoredAt,
        v1Keyset,
      ),
      true,
    );
    const address = `zrn1${"t".repeat(38)}`;
    assert.equal(
      await repository.createChallenge(
        {
          idHash: hashOpaque("stale-delete-restored-challenge"),
          sessionHash: restoredSession.tokenHash,
          subjectHash,
          address,
          accountId: `cosmos:zerone-1:${address}`,
          message: "restored challenge",
          createdAt: restoredAt,
          expiresAt: restoredAt + 60_000,
        },
        restoredAt - PI_CHALLENGE_RATE_WINDOW_MS,
        5,
        restoredAt - 1_000,
      ),
      true,
    );
    assert.equal(
      await repository.ensurePepperConfiguration(
        2,
        [v2Pin, v1Pin],
        restoredAt + 1,
      ),
      true,
    );

    assert.equal(
      await repository.deleteSubject(
        subjectHash,
        restoredAt + 2,
        restoredAt + 2 + PI_SUBJECT_DELETION_GUARD_MS,
        1,
        v1Keyset,
        hashOpaque("stale-delete-rejected-operation"),
      ),
      false,
    );
    assert.deepEqual(
      {
        ...(database
          .prepare(
            `SELECT deleted_at, expires_at, deletion_epoch, operation_hash
             FROM pi_subject_deletion_guards
             WHERE subject_hash = ?`,
          )
          .get(subjectHash) ?? {}),
      },
      {
        deleted_at: deletedAt,
        expires_at: guardExpiry,
        deletion_epoch: 1,
        operation_hash: originalOperation,
      },
    );
    assert.equal(
      database.prepare(`SELECT current_epoch FROM pi_identity_deletion_epoch`)
        .get()?.current_epoch,
      1,
    );
    assert.deepEqual(
      {
        ...(database
          .prepare(
            `SELECT active_version, keyset_fingerprint FROM pi_pepper_state`,
          )
          .get() ?? {}),
      },
      { active_version: 2, keyset_fingerprint: v2Keyset },
    );
    for (const table of [
      "pi_sessions",
      "pi_wallet_challenges",
      "pi_wallet_challenge_rate_events",
      "pi_subject_aliases",
    ]) {
      assert.equal(
        database.prepare(`SELECT COUNT(*) AS count FROM ${table}`).get()?.count,
        1,
        `${table} must survive the rejected stale-keyset deletion`,
      );
    }
    database.close();
  });

  it("aligns challenge quota and cleanup at the exact 60-second boundary", async () => {
    const { database, repository } = lifecycleDatabase();
    const now = 2_000_000;
    const recentSince = now - PI_CHALLENGE_RATE_WINDOW_MS;
    const pin = {
      version: 1,
      fingerprint: hashOpaque("challenge-boundary-pin"),
    };
    const keysetFingerprint = piPepperKeysetFingerprint(1, [pin]);
    const subjectHash = hashOpaque("challenge-boundary-subject");
    const session = repositorySession("challenge-boundary", subjectHash, now);
    assert.equal(
      await repository.ensurePepperConfiguration(1, [pin], now),
      true,
    );
    assert.equal(
      await repository.replaceSubjectSessions(
        session,
        [{ pepperVersion: 1, aliasHash: subjectHash }],
        0,
        now,
        keysetFingerprint,
      ),
      true,
    );
    database
      .prepare(
        `INSERT INTO pi_wallet_challenge_rate_events
           (subject_hash, challenge_hash, created_at)
         VALUES (?, ?, ?)`,
      )
      .run(subjectHash, hashOpaque("rate-exact-boundary"), recentSince);
    for (let index = 0; index < 4; index += 1) {
      database
        .prepare(
          `INSERT INTO pi_wallet_challenge_rate_events
             (subject_hash, challenge_hash, created_at)
           VALUES (?, ?, ?)`,
        )
        .run(
          subjectHash,
          hashOpaque(`rate-just-newer-${index}`),
          recentSince + 1,
        );
    }

    const address = `zrn1${"s".repeat(38)}`;
    const challenge = (label: string): PiChallenge => ({
      idHash: hashOpaque(`challenge-${label}`),
      sessionHash: session.tokenHash,
      subjectHash,
      address,
      accountId: `cosmos:zerone-1:${address}`,
      message: label,
      createdAt: now,
      expiresAt: now + 5 * 60 * 1_000,
    });
    assert.equal(
      await repository.createChallenge(
        challenge("boundary-accepted"),
        recentSince,
        5,
        recentSince,
      ),
      true,
    );
    assert.equal(
      await repository.createChallenge(
        challenge("boundary-limited"),
        recentSince,
        5,
        recentSince,
      ),
      false,
    );

    const cleanup = await repository.cleanupExpired({
      now,
      pendingBefore: now - PI_BEARER_PENDING_TTL_MS,
      idleBefore: recentSince,
      retainedBefore: now,
      maximumRowsPerTable: 100,
    });
    assert.equal(cleanup.challengeRateEventsDeleted, 1);
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_wallet_challenge_rate_events
           WHERE subject_hash = ? AND created_at = ?`,
        )
        .get(subjectHash, recentSince)?.count,
      0,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_wallet_challenge_rate_events
           WHERE subject_hash = ? AND created_at > ?`,
        )
        .get(subjectHash, recentSince)?.count,
      5,
    );
    database.close();
  });

  it("runs bounded finite retention at inclusive operator cutoffs", async () => {
    const { database, repository } = lifecycleDatabase();
    const now = 10_000_000;
    const retainedBefore = 1_000_000;
    await repository.ensurePepperConfiguration(
      1,
      [{ version: 1, fingerprint: hashOpaque("retention-pin") }],
      now,
    );

    database
      .prepare(
        `INSERT INTO pi_bearer_replay_fingerprints
           (fingerprint_scheme, fingerprint, first_claimed_at)
         VALUES ('sha256-v1', ?, ?), ('sha256-v1', ?, ?)`,
      )
      .run(
        hashOpaque("replay-at-boundary"),
        retainedBefore,
        hashOpaque("replay-after-boundary"),
        retainedBefore + 1,
      );
    for (const [label, disposition, consumedAt] of [
      ["old-superseded", "superseded", retainedBefore],
      ["old-unlinked", "unlinked", retainedBefore - 1],
      ["fresh-superseded", "superseded", retainedBefore + 1],
    ] as const) {
      database
        .prepare(
          `INSERT INTO pi_wallet_challenge_uses
             (challenge_hash, proof_hash, disposition, consumed_at)
           VALUES (?, NULL, ?, ?)`,
        )
        .run(hashOpaque(label), disposition, consumedAt);
    }

    const insertBinding = (
      label: string,
      subjectHash: string,
      lastSeenAt: number,
    ): void => {
      const tokenHash = hashOpaque(`retention-session-${label}`);
      database
        .prepare(
          `INSERT INTO pi_sessions
             (token_hash, subject_hash, username, created_at, expires_at,
              pepper_version, last_seen_at)
           VALUES (?, ?, ?, ?, ?, 1, ?)`,
        )
        .run(
          tokenHash,
          subjectHash,
          label,
          retainedBefore - 100,
          now + 10_000,
          lastSeenAt,
        );
      const challengeHash = hashOpaque(`retention-challenge-${label}`);
      const proofHash = hashOpaque(`retention-proof-${label}`);
      const address = `zrn1${label[0]?.repeat(38)}`;
      database
        .prepare(
          `INSERT INTO pi_wallet_bindings
             (subject_hash, address, account_id, challenge_hash, proof_hash,
              consent_version, bound_at)
           VALUES (?, ?, ?, ?, ?, 'pi-wallet-link-v1', ?)`,
        )
        .run(
          subjectHash,
          address,
          `cosmos:zerone-1:${address}`,
          challengeHash,
          proofHash,
          retainedBefore,
        );
    };

    const recentSubject = hashOpaque("retention-recent-subject");
    const boundarySubject = hashOpaque("retention-boundary-subject");
    insertBinding("recent", recentSubject, retainedBefore + 1);
    insertBinding("boundary", boundarySubject, retainedBefore);
    database
      .prepare(
        `INSERT INTO pi_subject_deletion_guards
           (subject_hash, deleted_at, expires_at, deletion_epoch,
            operation_hash)
         VALUES (?, 1, ?, 1, ?), (?, 1, ?, 1, ?)`,
      )
      .run(
        hashOpaque("guard-at-boundary"),
        now,
        hashOpaque("guard-operation-at-boundary"),
        hashOpaque("guard-after-boundary"),
        now + 1,
        hashOpaque("guard-operation-after-boundary"),
      );

    const result = await repository.cleanupExpired({
      now,
      pendingBefore: now - PI_BEARER_PENDING_TTL_MS,
      idleBefore: now - 30 * 60 * 1_000,
      retainedBefore,
      maximumRowsPerTable: 100,
    });
    assert.equal(result.sessionsDeleted, 1);
    assert.equal(result.bindingsDeleted, 1);
    assert.equal(result.challengeUsesDeleted, 3);
    assert.equal(result.deletionGuardsDeleted, 1);
    assert.equal(await repository.getBinding(recentSubject) !== null, true);
    assert.equal(await repository.getBinding(boundarySubject), null);
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_bearer_replay_fingerprints
           WHERE fingerprint_scheme = 'sha256-v1'`,
        )
        .get()?.count,
      2,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_bearer_replay_fingerprints
           WHERE first_claimed_at = ?`,
        )
        .get(retainedBefore)?.count,
      1,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_bearer_replay_fingerprints
           WHERE first_claimed_at = ?`,
        )
        .get(retainedBefore + 1)?.count,
      1,
    );
    assert.equal(
      database
        .prepare(
          `SELECT COUNT(*) AS count
           FROM pi_wallet_challenge_uses
           WHERE consumed_at = ?`,
        )
        .get(retainedBefore + 1)?.count,
      1,
    );
    database.close();
  });

});
