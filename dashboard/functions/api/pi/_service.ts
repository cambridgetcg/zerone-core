import {
  fromBech32,
  toBech32,
} from "@cosmjs/encoding";

import {
  constantTimeEqual,
  hashOpaque,
  isOpaqueToken,
  keyedHash,
  opaqueToken,
} from "./_crypto";
import {
  D1PiRepository,
  piPepperKeysetFingerprint,
} from "./_store";
import {
  parsePiStdSignature,
  verifyAdr36Signature,
  walletProofHash,
} from "./_wallet-proof";
import type {
  PiBinding,
  PiChallenge,
  PiEnv,
  PiPagesContext,
  PiRuntime,
  PiSession,
  PiSessionEnvelope,
  PiSubjectAlias,
} from "./_types";

export {
  PI_BEARER_PENDING_TTL_MS,
  PI_CHALLENGE_RATE_WINDOW_MS,
} from "./_types";
import { PI_CHALLENGE_RATE_WINDOW_MS } from "./_types";

export type PiEndpoint =
  | "authorize"
  | "session"
  | "me"
  | "logout"
  | "challenge"
  | "bind"
  | "data";

type JsonRecord = Record<string, unknown>;

interface PiPepperKey {
  readonly version: number;
  readonly pepper: string;
}

interface PiConfig {
  readonly clientId: string;
  readonly origin: string;
  readonly redirectUri: string;
  readonly subjectPeppers: readonly PiPepperKey[];
  readonly walletProofEnabled: boolean;
}

interface AuthenticatedSession {
  readonly rawToken: string;
  readonly tokenHash: string;
  readonly session: PiSession;
  readonly csrfToken: string;
  readonly pepperKey: PiPepperKey;
}

interface PiProfile {
  readonly uid: string;
  readonly username: string;
}

type ProfileResult =
  | { readonly kind: "ok"; readonly profile: PiProfile }
  | { readonly kind: "unauthorized" }
  | { readonly kind: "unavailable" };

const PI_AUTHORIZE_URL = "https://accounts.pinet.com/oauth/authorize";
const PI_ME_URL = "https://api.minepi.com/v2/me";
const CHAIN_ID = "zerone-1";
const OAUTH_COOKIE = "__Host-zrn-pi-oauth";
const SESSION_COOKIE = "__Host-zrn-pi-session";
const OAUTH_TTL_MS = 10 * 60 * 1_000;
const SESSION_TTL_MS = 8 * 60 * 60 * 1_000;
const SESSION_IDLE_MS = 30 * 60 * 1_000;
const CHALLENGE_TTL_MS = 5 * 60 * 1_000;
const MAX_CHALLENGES_PER_WINDOW = 5;
const MAX_REQUEST_BODY_BYTES = 8_192;
const MAX_PROFILE_BODY_BYTES = 16_384;
const PI_UPSTREAM_TIMEOUT_MS = 8_000;
const WALLET_LINK_CONSENT_VERSION = "pi-wallet-link-v1";
const DATA_DELETION_CONFIRMATION = "delete-pi-pilot-data-v1";
const MAX_SUBJECT_PEPPER_BYTES = 1_024;
const MAX_SUBJECT_PEPPER_KEYS = 8;

export const PI_SUBJECT_DELETION_GUARD_MS = 12 * 60 * 1_000;

const API_HEADERS = {
  "Cache-Control": "no-store",
  "Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
  "Content-Type": "application/json; charset=utf-8",
  "Referrer-Policy": "no-referrer",
  "X-Content-Type-Options": "nosniff",
};

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasOnlyKeys(value: JsonRecord, allowed: readonly string[]): boolean {
  const allowedKeys = new Set(allowed);
  return Object.keys(value).every((key) => allowedKeys.has(key));
}

function responseHeaders(): Headers {
  return new Headers(API_HEADERS);
}

function jsonResponse(
  body: unknown,
  status = 200,
  mutateHeaders?: (headers: Headers) => void,
): Response {
  const headers = responseHeaders();
  mutateHeaders?.(headers);
  return new Response(JSON.stringify(body), { status, headers });
}

function jsonError(
  message: string,
  status: number,
  mutateHeaders?: (headers: Headers) => void,
): Response {
  return jsonResponse({ error: message }, status, mutateHeaders);
}

function methodNotAllowed(allowed: readonly string[]): Response {
  return jsonError("Unsupported Pi pilot method", 405, (headers) => {
    headers.set("Allow", allowed.join(", "));
  });
}

function setCookie(
  name: string,
  value: string,
  maximumAgeSeconds: number,
): string {
  return `${name}=${value}; Path=/; Max-Age=${maximumAgeSeconds}; Secure; HttpOnly; SameSite=Lax`;
}

function clearCookie(name: string): string {
  return `${name}=; Path=/; Max-Age=0; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Secure; HttpOnly; SameSite=Lax`;
}

interface CookieResult {
  readonly present: boolean;
  readonly valid: boolean;
  readonly value?: string;
}

function readOpaqueCookie(request: Request, name: string): CookieResult {
  const header = request.headers.get("Cookie");
  if (!header) return { present: false, valid: true };
  const values: string[] = [];
  for (const part of header.split(";")) {
    const separator = part.indexOf("=");
    if (separator < 0) continue;
    if (part.slice(0, separator).trim() !== name) continue;
    values.push(part.slice(separator + 1).trim());
  }
  if (values.length === 0) return { present: false, valid: true };
  const value = values[0];
  return values.length === 1 && value !== undefined && isOpaqueToken(value)
    ? { present: true, valid: true, value }
    : { present: true, valid: false };
}

function safeUsername(value: unknown): value is string {
  return (
    typeof value === "string" &&
    value.length >= 1 &&
    value.length <= 64 &&
    !/[\u0000-\u001f\u007f]/u.test(value) &&
    !/[\ud800-\udfff]/u.test(value)
  );
}

function safeUid(value: unknown): value is string {
  return (
    typeof value === "string" &&
    value.length >= 1 &&
    value.length <= 256 &&
    !/[\u0000-\u001f\u007f]/u.test(value) &&
    !/[\ud800-\udfff]/u.test(value)
  );
}

function validSubjectPepper(value: unknown): value is string {
  if (typeof value !== "string") return false;
  const bytes = new TextEncoder().encode(value).length;
  return bytes >= 32 && bytes <= MAX_SUBJECT_PEPPER_BYTES;
}

function parseActivePepperVersion(value: string | undefined): number | null {
  if (value === undefined) return 1;
  if (!/^[1-9]\d*$/u.test(value)) return null;
  const version = Number(value);
  return Number.isSafeInteger(version) &&
    version <= 2_147_483_647 &&
    String(version) === value
    ? version
    : null;
}

function parseSubjectPeppers(env: PiEnv): readonly PiPepperKey[] | null {
  const activeVersion = parseActivePepperVersion(
    env.PI_SUBJECT_PEPPER_VERSION,
  );
  if (activeVersion === null || !validSubjectPepper(env.PI_SUBJECT_PEPPER)) {
    return null;
  }

  let previousValue: unknown = [];
  if (env.PI_SUBJECT_PEPPER_PREVIOUS !== undefined) {
    try {
      previousValue = JSON.parse(env.PI_SUBJECT_PEPPER_PREVIOUS) as unknown;
    } catch {
      return null;
    }
  }
  if (
    !Array.isArray(previousValue) ||
    previousValue.length >= MAX_SUBJECT_PEPPER_KEYS
  ) {
    return null;
  }

  const previous: PiPepperKey[] = [];
  let lastVersion = 0;
  for (const value of previousValue) {
    if (
      !isRecord(value) ||
      !hasOnlyKeys(value, ["version", "pepper"]) ||
      typeof value.version !== "number" ||
      !Number.isSafeInteger(value.version) ||
      value.version <= lastVersion ||
      value.version >= activeVersion ||
      value.version > 2_147_483_647 ||
      !validSubjectPepper(value.pepper)
    ) {
      return null;
    }
    previous.push({ version: value.version, pepper: value.pepper });
    lastVersion = value.version;
  }
  const subjectPeppers = [
    { version: activeVersion, pepper: env.PI_SUBJECT_PEPPER },
    ...previous,
  ];
  if (
    new Set(subjectPeppers.map((key) => key.pepper)).size !==
    subjectPeppers.length
  ) {
    return null;
  }
  return subjectPeppers;
}

function parseConfig(env: PiEnv): PiConfig | null {
  const clientId = env.PI_CLIENT_ID;
  const configuredOrigin = env.PI_PUBLIC_ORIGIN;
  const subjectPeppers = parseSubjectPeppers(env);
  if (
    env.PI_BEARER_SHA_CLEAN_START_CONFIRMED !== "true" ||
    typeof clientId !== "string" ||
    clientId.length < 1 ||
    clientId.length > 256 ||
    !/^[\x21-\x7e]+$/u.test(clientId) ||
    typeof configuredOrigin !== "string" ||
    subjectPeppers === null
  ) {
    return null;
  }

  let origin: URL;
  try {
    origin = new URL(configuredOrigin);
  } catch {
    return null;
  }
  const loopback =
    origin.hostname === "localhost" ||
    origin.hostname === "127.0.0.1" ||
    origin.hostname === "[::1]";
  if (
    configuredOrigin !== origin.origin ||
    origin.username !== "" ||
    origin.password !== "" ||
    origin.pathname !== "/" ||
    origin.search !== "" ||
    origin.hash !== "" ||
    (origin.protocol !== "https:" && !(loopback && origin.protocol === "http:"))
  ) {
    return null;
  }

  return {
    clientId,
    origin: origin.origin,
    redirectUri: new URL("/pi/callback/", origin.origin).toString(),
    subjectPeppers,
    walletProofEnabled: env.PI_WALLET_PROOF_ENABLED === "true",
  };
}

function requestIsAtOrigin(request: Request, config: PiConfig): boolean {
  try {
    return new URL(request.url).origin === config.origin;
  } catch {
    return false;
  }
}

function mutationOriginIsValid(request: Request, config: PiConfig): boolean {
  if (!requestIsAtOrigin(request, config)) return false;
  if (request.headers.get("Origin") !== config.origin) return false;
  const fetchSite = request.headers.get("Sec-Fetch-Site");
  return fetchSite === null || fetchSite === "same-origin";
}

function authorizationOriginIsValid(
  request: Request,
  config: PiConfig,
): boolean {
  return (
    requestIsAtOrigin(request, config) &&
    request.headers.get("Origin") === config.origin &&
    request.headers.get("Sec-Fetch-Site") === "same-origin"
  );
}

function queryIsEmpty(request: Request): boolean {
  try {
    return new URL(request.url).search === "";
  } catch {
    return false;
  }
}

type JsonBodyResult =
  | { readonly ok: true; readonly value: JsonRecord }
  | { readonly ok: false };

async function readJsonBody(request: Request): Promise<JsonBodyResult> {
  const contentType = request.headers.get("Content-Type");
  if (
    contentType === null ||
    contentType.split(";", 1)[0]?.trim().toLowerCase() !== "application/json" ||
    ![null, "identity"].includes(request.headers.get("Content-Encoding"))
  ) {
    return { ok: false };
  }
  const declaredLength = request.headers.get("Content-Length");
  if (
    declaredLength !== null &&
    (!/^\d+$/u.test(declaredLength) ||
      Number(declaredLength) > MAX_REQUEST_BODY_BYTES)
  ) {
    return { ok: false };
  }

  const reader = request.body?.getReader();
  if (!reader) return { ok: false };
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAX_REQUEST_BODY_BYTES) {
        await reader.cancel();
        return { ok: false };
      }
      chunks.push(value);
    }
  } catch {
    return { ok: false };
  }

  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  try {
    const parsed = JSON.parse(
      new TextDecoder("utf-8", { fatal: true }).decode(bytes),
    ) as unknown;
    return isRecord(parsed) ? { ok: true, value: parsed } : { ok: false };
  } catch {
    return { ok: false };
  }
}

async function readLimitedText(
  response: Response,
  maximumBytes: number,
): Promise<string | null> {
  const declaredLength = response.headers.get("Content-Length");
  if (
    declaredLength !== null &&
    (!/^\d+$/u.test(declaredLength) || Number(declaredLength) > maximumBytes)
  ) {
    return null;
  }
  const reader = response.body?.getReader();
  if (!reader) return "";
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > maximumBytes) {
        await reader.cancel();
        return null;
      }
      chunks.push(value);
    }
  } catch {
    return null;
  }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    return null;
  }
}

function profileFromJson(value: unknown): PiProfile | null {
  if (
    !isRecord(value) ||
    Object.hasOwn(value, "wallet_address") ||
    !safeUid(value.uid) ||
    !safeUsername(value.username)
  ) {
    return null;
  }
  if (value.credentials !== undefined) {
    if (!isRecord(value.credentials) || !Array.isArray(value.credentials.scopes)) {
      return null;
    }
    const scopes = value.credentials.scopes;
    if (
      scopes.length !== 1 ||
      scopes[0] !== "username"
    ) {
      return null;
    }
  }
  return { uid: value.uid, username: value.username };
}

async function verifyPiProfile(
  accessToken: string,
  runtime: PiRuntime,
): Promise<ProfileResult> {
  let upstream: Response;
  try {
    upstream = await runtime.fetch(PI_ME_URL, {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${accessToken}`,
      },
      redirect: "manual",
      signal: AbortSignal.timeout(PI_UPSTREAM_TIMEOUT_MS),
    });
  } catch {
    return { kind: "unavailable" };
  }
  if (upstream.status === 401 || upstream.status === 403) {
    return { kind: "unauthorized" };
  }
  if (!upstream.ok || upstream.status !== 200) {
    return { kind: "unavailable" };
  }
  const contentType = upstream.headers.get("Content-Type");
  if (
    contentType === null ||
    contentType.split(";", 1)[0]?.trim().toLowerCase() !== "application/json"
  ) {
    return { kind: "unavailable" };
  }
  const text = await readLimitedText(upstream, MAX_PROFILE_BODY_BYTES);
  if (text === null) return { kind: "unavailable" };
  try {
    const profile = profileFromJson(JSON.parse(text) as unknown);
    return profile ? { kind: "ok", profile } : { kind: "unavailable" };
  } catch {
    return { kind: "unavailable" };
  }
}

function activePepper(config: PiConfig): PiPepperKey {
  const active = config.subjectPeppers[0];
  if (!active) throw new Error("Active Pi subject pepper is missing");
  return active;
}

function pepperForVersion(
  config: PiConfig,
  version: number,
): PiPepperKey | null {
  return config.subjectPeppers.find((key) => key.version === version) ?? null;
}

async function pepperConfigurationIsCurrent(
  config: PiConfig,
  runtime: PiRuntime,
): Promise<string | null> {
  const active = activePepper(config);
  const pins = await Promise.all(
    config.subjectPeppers.map(async (key) => ({
      version: key.version,
      fingerprint: await keyedHash(
        key.pepper,
        `zerone-pi-pepper-key-fingerprint-v1\u0000${config.origin}`,
      ),
    })),
  );
  const current = await runtime.repository.ensurePepperConfiguration(
    active.version,
    pins,
    runtime.now(),
  );
  return current ? piPepperKeysetFingerprint(active.version, pins) : null;
}

function bearerReplayCommitment(accessToken: string): string {
  return hashOpaque(`zerone-pi-bearer-replay-v1\u0000${accessToken}`);
}

async function subjectAliases(
  config: PiConfig,
  uid: string,
): Promise<readonly PiSubjectAlias[]> {
  return Promise.all(
    config.subjectPeppers.map(async (key) => ({
      pepperVersion: key.version,
      aliasHash: await keyedHash(
        key.pepper,
        `zerone-pi-subject-v1\u0000${uid}`,
      ),
    })),
  );
}

async function csrfToken(
  pepperKey: PiPepperKey,
  rawSessionToken: string,
): Promise<string> {
  return keyedHash(
    pepperKey.pepper,
    `zerone-pi-csrf-v1\u0000${rawSessionToken}`,
  );
}

async function authenticate(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
): Promise<AuthenticatedSession | null> {
  const cookie = readOpaqueCookie(request, SESSION_COOKIE);
  if (!cookie.valid || !cookie.value) return null;
  const tokenHash = hashOpaque(cookie.value);
  const now = runtime.now();
  const session = await runtime.repository.getSession(
    tokenHash,
    now,
    now - SESSION_IDLE_MS,
  );
  if (!session) return null;
  const pepperKey = pepperForVersion(config, session.pepperVersion);
  if (!pepperKey) return null;
  return {
    rawToken: cookie.value,
    tokenHash,
    session,
    csrfToken: await csrfToken(pepperKey, cookie.value),
    pepperKey,
  };
}

async function authenticateMutation(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
): Promise<AuthenticatedSession | null> {
  if (!mutationOriginIsValid(request, config)) return null;
  const authenticated = await authenticate(request, config, runtime);
  const presented = request.headers.get("X-Zerone-CSRF");
  return authenticated &&
    presented !== null &&
    isOpaqueToken(presented) &&
    constantTimeEqual(presented, authenticated.csrfToken)
    ? authenticated
    : null;
}

async function rotatedAuthentication(
  authenticated: AuthenticatedSession,
  config: PiConfig,
  runtime: PiRuntime,
  now: number,
): Promise<AuthenticatedSession | null> {
  if (authenticated.session.expiresAt <= now) return null;
  const pepperKey = activePepper(config);
  const rawToken = opaqueToken(runtime.randomBytes(32));
  const tokenHash = hashOpaque(rawToken);
  const session: PiSession = {
    tokenHash,
    subjectHash: authenticated.session.subjectHash,
    username: authenticated.session.username,
    pepperVersion: pepperKey.version,
    lastSeenAt: now,
    expiresAt: authenticated.session.expiresAt,
  };
  return {
    rawToken,
    tokenHash,
    session,
    csrfToken: await csrfToken(pepperKey, rawToken),
    pepperKey,
  };
}

function remainingSessionSeconds(session: PiSession, now: number): number {
  return Math.max(0, Math.floor((session.expiresAt - now) / 1_000));
}

function envelope(
  config: PiConfig,
  authenticated?: AuthenticatedSession,
  binding?: PiBinding | null,
): PiSessionEnvelope {
  if (!authenticated) {
    return {
      enabled: true,
      walletProofEnabled: config.walletProofEnabled,
      authenticated: false,
    };
  }
  return {
    enabled: true,
    walletProofEnabled: config.walletProofEnabled,
    authenticated: true,
    username: authenticated.session.username,
    csrfToken: authenticated.csrfToken,
    ...(binding
      ? {
          linked: {
            address: binding.address,
            accountId: binding.accountId,
          },
        }
      : {}),
  };
}

function disabledEnvelope(): PiSessionEnvelope {
  return {
    enabled: false,
    walletProofEnabled: false,
    authenticated: false,
  };
}

function disabledResponse(endpoint: PiEndpoint, request: Request): Response {
  if (endpoint === "me" && request.method === "GET") {
    return jsonResponse(disabledEnvelope());
  }
  return jsonError("Pi pilot is unavailable", 404);
}

async function authorize(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
): Promise<Response> {
  if (request.method !== "POST") return methodNotAllowed(["POST"]);
  if (!authorizationOriginIsValid(request, config)) {
    return jsonError("Pi authorization origin was not accepted", 403);
  }
  if (!queryIsEmpty(request)) {
    return jsonError("Invalid Pi authorization request", 400);
  }
  const body = await readJsonBody(request);
  if (!body.ok || !hasOnlyKeys(body.value, [])) {
    return jsonError("Invalid Pi authorization request", 400);
  }
  const now = runtime.now();
  const state = opaqueToken(runtime.randomBytes(32));
  const browserTransaction = opaqueToken(runtime.randomBytes(32));
  await runtime.repository.createOAuthFlow(
    hashOpaque(state),
    hashOpaque(browserTransaction),
    now,
    now + OAUTH_TTL_MS,
  );
  const target = new URL(PI_AUTHORIZE_URL);
  target.searchParams.set("response_type", "token");
  target.searchParams.set("client_id", config.clientId);
  target.searchParams.set("redirect_uri", config.redirectUri);
  target.searchParams.set("scope", "username");
  target.searchParams.set("state", state);

  return jsonResponse(
    { authorizationUrl: target.toString() },
    200,
    (headers) => {
      headers.append(
        "Set-Cookie",
        setCookie(OAUTH_COOKIE, browserTransaction, OAUTH_TTL_MS / 1_000),
      );
    },
  );
}

function validAccessToken(value: unknown): value is string {
  return (
    typeof value === "string" &&
    value.length >= 16 &&
    value.length <= 4_096 &&
    /^[\x21-\x7e]+$/u.test(value)
  );
}

async function createSession(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
  keysetFingerprint: string,
): Promise<Response> {
  if (request.method !== "POST") return methodNotAllowed(["POST"]);
  if (!mutationOriginIsValid(request, config) || !queryIsEmpty(request)) {
    return jsonError("Invalid Pi session request", 403);
  }
  const body = await readJsonBody(request);
  if (
    !body.ok ||
    !hasOnlyKeys(body.value, ["accessToken", "state"]) ||
    !validAccessToken(body.value.accessToken) ||
    typeof body.value.state !== "string" ||
    !isOpaqueToken(body.value.state)
  ) {
    return jsonError("Invalid Pi session request", 400);
  }
  const oauthCookie = readOpaqueCookie(request, OAUTH_COOKIE);
  if (!oauthCookie.valid || !oauthCookie.value) {
    return jsonError("Pi authorization state was not accepted", 400);
  }

  const now = runtime.now();
  const activeKey = activePepper(config);
  const stateHash = hashOpaque(body.value.state);
  const browserHash = hashOpaque(oauthCookie.value);
  const replayCommitment = bearerReplayCommitment(body.value.accessToken);
  const consumed = await runtime.repository.consumeOAuthFlow(
    stateHash,
    browserHash,
    replayCommitment,
    activeKey.version,
    now,
  );
  if (consumed === null) {
    return jsonError("Pi authorization state was not accepted", 400);
  }
  const clearOAuth = (headers: Headers) => {
    headers.append("Set-Cookie", clearCookie(OAUTH_COOKIE));
  };

  const profile = await verifyPiProfile(body.value.accessToken, runtime);
  if (profile.kind === "unauthorized") {
    await runtime.repository.rejectBearerClaim(
      replayCommitment,
      stateHash,
      browserHash,
    );
    return jsonError("Pi sign-in was not accepted", 401, clearOAuth);
  }
  if (profile.kind === "unavailable") {
    return jsonError("Pi identity is temporarily unavailable", 502, clearOAuth);
  }

  const promoted = await runtime.repository.promoteBearerClaim(
    replayCommitment,
    stateHash,
    browserHash,
  );
  if (!promoted) {
    return jsonError("Pi authorization must be restarted", 409, clearOAuth);
  }

  const createdAt = runtime.now();
  const aliases = await subjectAliases(config, profile.profile.uid);
  const canonicalSubject = await runtime.repository.resolveSubject(aliases);
  if (!canonicalSubject) {
    return jsonError("Pi authorization must be restarted", 409, clearOAuth);
  }
  const rawSessionToken = opaqueToken(runtime.randomBytes(32));
  const session: PiSession = {
    tokenHash: hashOpaque(rawSessionToken),
    subjectHash: canonicalSubject,
    username: profile.profile.username,
    pepperVersion: activeKey.version,
    lastSeenAt: createdAt,
    expiresAt: createdAt + SESSION_TTL_MS,
  };
  const replaced = await runtime.repository.replaceSubjectSessions(
    session,
    aliases,
    consumed,
    createdAt,
    keysetFingerprint,
  );
  if (!replaced) {
    return jsonError("Pi authorization must be restarted", 409, clearOAuth);
  }
  const authenticated: AuthenticatedSession = {
    rawToken: rawSessionToken,
    tokenHash: session.tokenHash,
    session,
    csrfToken: await csrfToken(activeKey, rawSessionToken),
    pepperKey: activeKey,
  };

  return jsonResponse(envelope(config, authenticated), 200, (headers) => {
    clearOAuth(headers);
    headers.append(
      "Set-Cookie",
      setCookie(SESSION_COOKIE, rawSessionToken, SESSION_TTL_MS / 1_000),
    );
  });
}

async function getMe(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
): Promise<Response> {
  if (request.method !== "GET") return methodNotAllowed(["GET"]);
  if (!requestIsAtOrigin(request, config) || !queryIsEmpty(request)) {
    return jsonError("Invalid Pi session request", 400);
  }
  const cookie = readOpaqueCookie(request, SESSION_COOKIE);
  const authenticated = await authenticate(request, config, runtime);
  if (!authenticated) {
    return jsonResponse(envelope(config), 200, (headers) => {
      if (cookie.present) headers.append("Set-Cookie", clearCookie(SESSION_COOKIE));
    });
  }
  const binding = await runtime.repository.getBinding(
    authenticated.session.subjectHash,
  );
  return jsonResponse(envelope(config, authenticated, binding));
}

async function logout(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
): Promise<Response> {
  if (request.method !== "POST") return methodNotAllowed(["POST"]);
  if (!queryIsEmpty(request)) return jsonError("Invalid Pi logout request", 400);
  const body = await readJsonBody(request);
  if (!body.ok || !hasOnlyKeys(body.value, [])) {
    return jsonError("Invalid Pi logout request", 400);
  }
  const authenticated = await authenticateMutation(request, config, runtime);
  if (!authenticated) return jsonError("Pi session was not accepted", 401);
  await runtime.repository.revokeSession(
    authenticated.tokenHash,
    runtime.now(),
  );
  return jsonResponse(envelope(config), 200, (headers) => {
    headers.append("Set-Cookie", clearCookie(SESSION_COOKIE));
  });
}

async function deletePilotData(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
  keysetFingerprint: string,
): Promise<Response> {
  if (request.method !== "DELETE") return methodNotAllowed(["DELETE"]);
  if (!queryIsEmpty(request)) {
    return jsonError("Invalid Pi data-deletion request", 400);
  }
  const body = await readJsonBody(request);
  if (
    !body.ok ||
    !hasOnlyKeys(body.value, ["confirmation"]) ||
    body.value.confirmation !== DATA_DELETION_CONFIRMATION
  ) {
    return jsonError("Invalid Pi data-deletion request", 400);
  }
  const authenticated = await authenticateMutation(request, config, runtime);
  if (!authenticated) return jsonError("Pi session was not accepted", 401);

  const now = runtime.now();
  const deleted = await runtime.repository.deleteSubject(
    authenticated.session.subjectHash,
    now,
    now + PI_SUBJECT_DELETION_GUARD_MS,
    activePepper(config).version,
    keysetFingerprint,
    hashOpaque(
      `zerone-pi-deletion-operation-v1\u0000${opaqueToken(runtime.randomBytes(32))}`,
    ),
  );
  if (!deleted) {
    return jsonError("Pi pilot data could not be deleted", 409);
  }
  return jsonResponse(envelope(config), 200, (headers) => {
    headers.append("Set-Cookie", clearCookie(SESSION_COOKIE));
    headers.append("Set-Cookie", clearCookie(OAUTH_COOKIE));
  });
}

function zeroneAccount(value: unknown): { address: string; accountId: string } | null {
  if (typeof value !== "string" || value.length !== 42) return null;
  try {
    const decoded = fromBech32(value);
    if (
      decoded.prefix !== "zrn" ||
      decoded.data.length !== 20 ||
      toBech32(decoded.prefix, decoded.data) !== value
    ) {
      return null;
    }
    return {
      address: value,
      accountId: `cosmos:${CHAIN_ID}:${value}`,
    };
  } catch {
    return null;
  }
}

function bindingMessage(
  config: PiConfig,
  address: string,
  accountId: string,
  nonce: string,
  challengeId: string,
  sessionBinding: string,
  issuedAt: number,
  expiresAt: number,
): string {
  const origin = new URL(config.origin);
  return [
    "ZERONE off-chain Pi account binding",
    "",
    "Version: 1",
    "Audience: zerone-pi-wallet-link",
    `Domain: ${origin.host}`,
    `URI: ${new URL("/", config.origin).toString()}`,
    `Chain ID: ${CHAIN_ID}`,
    `Account ID: ${accountId}`,
    `Address: ${address}`,
    `Nonce: ${nonce}`,
    `Request ID: ${challengeId}`,
    `Session Binding: ${sessionBinding}`,
    `Issued At: ${new Date(issuedAt).toISOString()}`,
    `Expiration Time: ${new Date(expiresAt).toISOString()}`,
    `Consent Version: ${WALLET_LINK_CONSENT_VERSION}`,
    "Purpose: Bind an off-chain Zerone address-control proof to the current Pi account session.",
    "Authorization: No transaction, payment, transfer, bridge, reward, delegation, or on-chain permission.",
  ].join("\n");
}

async function createChallenge(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
): Promise<Response> {
  if (request.method !== "POST") return methodNotAllowed(["POST"]);
  if (!config.walletProofEnabled) {
    return jsonError("Zerone wallet proof is unavailable", 404);
  }
  if (!queryIsEmpty(request)) {
    return jsonError("Invalid wallet challenge request", 400);
  }
  const body = await readJsonBody(request);
  if (!body.ok || !hasOnlyKeys(body.value, ["address"])) {
    return jsonError("Invalid wallet challenge request", 400);
  }
  const account = zeroneAccount(body.value.address);
  if (!account) return jsonError("Invalid wallet challenge request", 400);
  const authenticated = await authenticateMutation(request, config, runtime);
  if (!authenticated) return jsonError("Pi session was not accepted", 401);
  if (
    await runtime.repository.getBinding(authenticated.session.subjectHash)
  ) {
    return jsonError("A Zerone wallet proof is already linked", 409);
  }

  const now = runtime.now();
  const expiresAt = now + CHALLENGE_TTL_MS;
  const challengeId = opaqueToken(runtime.randomBytes(32));
  const sessionBinding = await keyedHash(
    authenticated.pepperKey.pepper,
    [
      "zerone-pi-wallet-session-binding-v1",
      authenticated.session.subjectHash,
      authenticated.tokenHash,
      challengeId,
    ].join("\u0000"),
  );
  const challenge: PiChallenge = {
    idHash: hashOpaque(challengeId),
    sessionHash: authenticated.tokenHash,
    subjectHash: authenticated.session.subjectHash,
    address: account.address,
    accountId: account.accountId,
    message: bindingMessage(
      config,
      account.address,
      account.accountId,
      opaqueToken(runtime.randomBytes(32)),
      challengeId,
      sessionBinding,
      now,
      expiresAt,
    ),
    createdAt: now,
    expiresAt,
  };
  const created = await runtime.repository.createChallenge(
    challenge,
    now - PI_CHALLENGE_RATE_WINDOW_MS,
    MAX_CHALLENGES_PER_WINDOW,
    now - SESSION_IDLE_MS,
  );
  if (!created) {
    if (
      await runtime.repository.getBinding(authenticated.session.subjectHash)
    ) {
      return jsonError("A Zerone wallet proof is already linked", 409);
    }
    return jsonError("Too many wallet challenge requests", 429);
  }
  return jsonResponse({
    challengeId,
    message: challenge.message,
    expiresAt: new Date(expiresAt).toISOString(),
  });
}

async function bind(
  request: Request,
  config: PiConfig,
  runtime: PiRuntime,
): Promise<Response> {
  if (request.method !== "POST" && request.method !== "DELETE") {
    return methodNotAllowed(["POST", "DELETE"]);
  }
  if (!queryIsEmpty(request)) {
    return jsonError("Invalid wallet binding request", 400);
  }
  const body = await readJsonBody(request);
  if (!body.ok) return jsonError("Invalid wallet binding request", 400);
  const authenticated = await authenticateMutation(request, config, runtime);
  if (!authenticated) return jsonError("Pi session was not accepted", 401);

  if (request.method === "DELETE") {
    if (!hasOnlyKeys(body.value, [])) {
      return jsonError("Invalid wallet binding request", 400);
    }
    const now = runtime.now();
    const rotated = await rotatedAuthentication(
      authenticated,
      config,
      runtime,
      now,
    );
    if (
      !rotated ||
      !(await runtime.repository.deleteBinding(
        authenticated.session.subjectHash,
        authenticated.tokenHash,
        rotated.session,
        now - SESSION_IDLE_MS,
        now,
      ))
    ) {
      return jsonError("Wallet binding could not be removed", 409);
    }
    return jsonResponse(envelope(config, rotated), 200, (headers) => {
      headers.append(
        "Set-Cookie",
        setCookie(
          SESSION_COOKIE,
          rotated.rawToken,
          remainingSessionSeconds(rotated.session, now),
        ),
      );
    });
  }

  if (!config.walletProofEnabled) {
    return jsonError("Zerone wallet proof is unavailable", 404);
  }
  if (
    !hasOnlyKeys(body.value, ["challengeId", "signature"]) ||
    typeof body.value.challengeId !== "string" ||
    !isOpaqueToken(body.value.challengeId)
  ) {
    return jsonError("Invalid wallet binding request", 400);
  }
  const signature = parsePiStdSignature(body.value.signature);
  if (!signature) return jsonError("Invalid wallet binding request", 400);

  const challengeHash = hashOpaque(body.value.challengeId);
  const challengeReadAt = runtime.now();
  const challenge = await runtime.repository.getChallenge(
    challengeHash,
    authenticated.tokenHash,
    authenticated.session.subjectHash,
    challengeReadAt,
    challengeReadAt - SESSION_IDLE_MS,
  );
  if (!challenge) {
    return jsonError("Wallet challenge was not accepted", 409);
  }
  if (!verifyAdr36Signature(challenge.address, challenge.message, signature)) {
    return jsonError("Wallet proof was not accepted", 400);
  }
  const now = runtime.now();
  const rotated = await rotatedAuthentication(
    authenticated,
    config,
    runtime,
    now,
  );
  if (!rotated) {
    return jsonError("Wallet binding could not be completed", 409);
  }
  const binding = await runtime.repository.bindChallenge(
    challengeHash,
    authenticated.tokenHash,
    authenticated.session.subjectHash,
    walletProofHash(challenge.message, signature),
    WALLET_LINK_CONSENT_VERSION,
    rotated.session,
    now - SESSION_IDLE_MS,
    now,
  );
  if (!binding) {
    return jsonError("Wallet binding could not be completed", 409);
  }
  return jsonResponse(envelope(config, rotated, binding), 200, (headers) => {
    headers.append(
      "Set-Cookie",
      setCookie(
        SESSION_COOKIE,
        rotated.rawToken,
        remainingSessionSeconds(rotated.session, now),
      ),
    );
  });
}

export async function handlePiRequest(
  endpoint: PiEndpoint,
  request: Request,
  env: PiEnv,
  runtime: PiRuntime,
): Promise<Response> {
  const enabled = env.PI_PILOT_ENABLED === "true";
  if (!enabled) return disabledResponse(endpoint, request);
  const config = parseConfig(env);
  if (!config) return jsonError("Pi pilot is temporarily unavailable", 503);
  const keysetFingerprint = await pepperConfigurationIsCurrent(config, runtime);
  if (keysetFingerprint === null) {
    return jsonError("Pi pilot is temporarily unavailable", 503);
  }

  switch (endpoint) {
    case "authorize":
      return authorize(request, config, runtime);
    case "session":
      return createSession(request, config, runtime, keysetFingerprint);
    case "me":
      return getMe(request, config, runtime);
    case "logout":
      return logout(request, config, runtime);
    case "challenge":
      return createChallenge(request, config, runtime);
    case "bind":
      return bind(request, config, runtime);
    case "data":
      return deletePilotData(request, config, runtime, keysetFingerprint);
  }
}

function runtimeFromEnv(env: PiEnv): PiRuntime {
  if (!env.PI_AUTH_DB || typeof env.PI_AUTH_DB.prepare !== "function") {
    throw new Error("PI_AUTH_DB is unavailable");
  }
  return {
    fetch: globalThis.fetch.bind(globalThis),
    now: () => Date.now(),
    randomBytes(length) {
      const bytes = new Uint8Array(length);
      globalThis.crypto.getRandomValues(bytes);
      return bytes;
    },
    repository: new D1PiRepository(env.PI_AUTH_DB),
  };
}

export async function runPiEndpoint(
  endpoint: PiEndpoint,
  context: PiPagesContext,
): Promise<Response> {
  try {
    if (context.env.PI_PILOT_ENABLED !== "true") {
      return disabledResponse(endpoint, context.request);
    }
    return await handlePiRequest(
      endpoint,
      context.request,
      context.env,
      runtimeFromEnv(context.env),
    );
  } catch {
    return jsonError("Pi pilot is temporarily unavailable", 503);
  }
}
