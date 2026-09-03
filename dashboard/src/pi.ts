const PI_API_PREFIX = "/api/pi/";
const PI_AUTHORIZATION_ORIGIN = "https://accounts.pinet.com";
const PI_AUTHORIZATION_PATH = "/oauth/authorize";
const MAX_JSON_RESPONSE_BYTES = 16_384;
const PI_REQUEST_TIMEOUT_MS = 12_000;
const ZERONE_ADDRESS = /^zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}$/;
const BASE64URL_VALUE = /^[A-Za-z0-9_-]+$/;

type JsonRecord = Record<string, unknown>;

export interface PiLinkedWalletProof {
  address: string;
  accountId: string;
}

export interface PiSession {
  enabled: boolean;
  walletProofEnabled: boolean;
  authenticated: boolean;
  username?: string;
  csrfToken?: string;
  linked?: PiLinkedWalletProof;
}

export interface PiWalletChallenge {
  challengeId: string;
  message: string;
  expiresAt: string;
}

export interface PiWalletProofSignature {
  pub_key: {
    type: "tendermint/PubKeySecp256k1";
    value: string;
  };
  signature: string;
}

export class PiTransportError extends Error {
  readonly status: number | null;

  constructor(message: string, status: number | null = null) {
    super(message);
    this.name = "PiTransportError";
    this.status = status;
  }
}

function isRecord(value: unknown): value is JsonRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasOnlyKeys(value: JsonRecord, allowed: readonly string[]): boolean {
  const allowedSet = new Set(allowed);
  return Object.keys(value).every((key) => allowedSet.has(key));
}

function boundedText(
  value: unknown,
  minimum: number,
  maximum: number,
): value is string {
  return (
    typeof value === "string" &&
    value.length >= minimum &&
    value.length <= maximum &&
    !/[\u0000-\u001f\u007f]/.test(value)
  );
}

function parseLinkedWalletProof(value: unknown): PiLinkedWalletProof | undefined {
  if (value === undefined) return undefined;
  if (
    !isRecord(value) ||
    !hasOnlyKeys(value, ["address", "accountId"]) ||
    typeof value.address !== "string" ||
    !ZERONE_ADDRESS.test(value.address) ||
    value.accountId !== `cosmos:zerone-1:${value.address}`
  ) {
    throw new PiTransportError("The Pi pilot returned an invalid wallet proof.");
  }
  return {
    address: value.address,
    accountId: value.accountId,
  };
}

function parsePiSession(value: unknown): PiSession {
  if (
    !isRecord(value) ||
    !hasOnlyKeys(value, [
      "enabled",
      "walletProofEnabled",
      "authenticated",
      "username",
      "csrfToken",
      "linked",
    ]) ||
    typeof value.enabled !== "boolean" ||
    typeof value.walletProofEnabled !== "boolean" ||
    typeof value.authenticated !== "boolean"
  ) {
    throw new PiTransportError("The Pi pilot returned an invalid session.");
  }

  const linked = parseLinkedWalletProof(value.linked);
  if (!value.authenticated) {
    if (
      value.username !== undefined ||
      value.csrfToken !== undefined ||
      linked !== undefined
    ) {
      throw new PiTransportError("The Pi pilot returned an invalid session.");
    }
    return {
      enabled: value.enabled,
      walletProofEnabled: value.walletProofEnabled,
      authenticated: false,
    };
  }

  if (
    !boundedText(value.username, 1, 128) ||
    typeof value.csrfToken !== "string" ||
    value.csrfToken.length < 16 ||
    value.csrfToken.length > 256 ||
    !BASE64URL_VALUE.test(value.csrfToken)
  ) {
    throw new PiTransportError("The Pi pilot returned an invalid session.");
  }

  return {
    enabled: value.enabled,
    walletProofEnabled: value.walletProofEnabled,
    authenticated: true,
    username: value.username,
    csrfToken: value.csrfToken,
    ...(linked ? { linked } : {}),
  };
}

function parseWalletChallenge(value: unknown): PiWalletChallenge {
  if (
    !isRecord(value) ||
    !hasOnlyKeys(value, ["challengeId", "message", "expiresAt"]) ||
    typeof value.challengeId !== "string" ||
    value.challengeId.length < 16 ||
    value.challengeId.length > 128 ||
    !BASE64URL_VALUE.test(value.challengeId.replaceAll("-", "")) ||
    typeof value.message !== "string" ||
    value.message.length === 0 ||
    value.message.length > 4_096 ||
    /[\u0000\u007f]/.test(value.message) ||
    typeof value.expiresAt !== "string" ||
    value.expiresAt.length > 64 ||
    !Number.isFinite(Date.parse(value.expiresAt))
  ) {
    throw new PiTransportError("The Pi pilot returned an invalid wallet challenge.");
  }
  return {
    challengeId: value.challengeId,
    message: value.message,
    expiresAt: value.expiresAt,
  };
}

function parsePiAuthorizationUrl(value: unknown): URL {
  if (
    !isRecord(value) ||
    !hasOnlyKeys(value, ["authorizationUrl"]) ||
    typeof value.authorizationUrl !== "string" ||
    value.authorizationUrl.length > 4_096
  ) {
    throw new PiTransportError("The Pi authorization response was invalid.");
  }

  let target: URL;
  try {
    target = new URL(value.authorizationUrl);
  } catch {
    throw new PiTransportError("The Pi authorization response was invalid.");
  }
  const expectedKeys = [
    "client_id",
    "redirect_uri",
    "response_type",
    "scope",
    "state",
  ];
  const actualKeys = [...target.searchParams.keys()].sort();
  if (
    target.origin !== PI_AUTHORIZATION_ORIGIN ||
    target.pathname !== PI_AUTHORIZATION_PATH ||
    target.username !== "" ||
    target.password !== "" ||
    target.hash !== "" ||
    actualKeys.length !== expectedKeys.length ||
    actualKeys.some((key, index) => key !== expectedKeys[index]) ||
    !boundedText(target.searchParams.get("client_id"), 1, 256) ||
    target.searchParams.get("redirect_uri") !==
      `${window.location.origin}/pi/callback/` ||
    target.searchParams.get("response_type") !== "token" ||
    target.searchParams.get("scope") !== "username" ||
    !/^[A-Za-z0-9_-]{43}$/u.test(target.searchParams.get("state") ?? "")
  ) {
    throw new PiTransportError("The Pi authorization response was invalid.");
  }
  return target;
}

function sameOriginPiUrl(path: string): URL {
  const url = new URL(path, window.location.origin);
  if (
    url.origin !== window.location.origin ||
    !url.pathname.startsWith(PI_API_PREFIX) ||
    url.search !== "" ||
    url.hash !== ""
  ) {
    throw new PiTransportError("The Pi pilot endpoint is invalid.");
  }
  return url;
}

async function readLimitedResponseText(response: Response): Promise<string> {
  const declaredLength = response.headers.get("content-length");
  if (
    declaredLength !== null &&
    (!/^\d+$/.test(declaredLength) ||
      Number(declaredLength) > MAX_JSON_RESPONSE_BYTES)
  ) {
    throw new PiTransportError("The Pi pilot response was too large.");
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
      if (total > MAX_JSON_RESPONSE_BYTES) {
        await reader.cancel();
        throw new PiTransportError("The Pi pilot response was too large.");
      }
      chunks.push(value);
    }
  } catch (error) {
    if (error instanceof PiTransportError) throw error;
    throw new PiTransportError("The Pi pilot response could not be read.");
  }

  const bytes = new Uint8Array(total);
  let offset = 0;
  chunks.forEach((chunk) => {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  });
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    throw new PiTransportError("The Pi pilot returned invalid text.");
  }
}

async function readJsonResponse(response: Response): Promise<unknown> {
  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.toLowerCase().startsWith("application/json")) {
    throw new PiTransportError("The Pi pilot returned an invalid response.");
  }
  const text = await readLimitedResponseText(response);
  try {
    return JSON.parse(text) as unknown;
  } catch {
    throw new PiTransportError("The Pi pilot returned invalid JSON.");
  }
}

async function piRequest(
  path: string,
  init: {
    method?: "GET" | "POST" | "DELETE";
    body?: JsonRecord;
    csrfToken?: string;
  } = {},
): Promise<unknown> {
  const method = init.method ?? "GET";
  const headers = new Headers({ Accept: "application/json" });
  let body: string | undefined;
  if (method !== "GET") {
    headers.set("Content-Type", "application/json");
    body = JSON.stringify(init.body ?? {});
  }
  if (init.csrfToken) headers.set("X-Zerone-CSRF", init.csrfToken);

  let response: Response;
  try {
    response = await fetch(sameOriginPiUrl(path), {
      method,
      headers,
      ...(body === undefined ? {} : { body }),
      credentials: "same-origin",
      cache: "no-store",
      redirect: "error",
      referrerPolicy: "no-referrer",
      signal: AbortSignal.timeout(PI_REQUEST_TIMEOUT_MS),
    });
  } catch {
    throw new PiTransportError("The Pi pilot is unavailable right now.");
  }
  if (!response.ok) {
    throw new PiTransportError(
      "The Pi pilot request was not accepted.",
      response.status,
    );
  }
  return readJsonResponse(response);
}

export async function beginPiSignIn(): Promise<void> {
  const target = parsePiAuthorizationUrl(
    await piRequest("/api/pi/authorize", {
      method: "POST",
      body: {},
    }),
  );
  window.location.assign(target);
}

export async function finishPiSignIn(
  accessToken: string,
  state: string,
): Promise<PiSession> {
  const session = parsePiSession(
    await piRequest("/api/pi/session", {
      method: "POST",
      body: { accessToken, state },
    }),
  );
  if (!session.enabled || !session.authenticated) {
    throw new PiTransportError(
      "The Pi pilot did not create an authenticated session.",
    );
  }
  return session;
}

export async function getPiSession(): Promise<PiSession> {
  return parsePiSession(await piRequest("/api/pi/me"));
}

export async function endPiSession(csrfToken: string): Promise<PiSession> {
  const session = parsePiSession(
    await piRequest("/api/pi/logout", {
      method: "POST",
      body: {},
      csrfToken,
    }),
  );
  if (!session.enabled || session.authenticated) {
    throw new PiTransportError("The Pi pilot did not end the session.");
  }
  return session;
}

export async function deletePiPilotData(
  csrfToken: string,
): Promise<PiSession> {
  const session = parsePiSession(
    await piRequest("/api/pi/data", {
      method: "DELETE",
      body: { confirmation: "delete-pi-pilot-data-v1" },
      csrfToken,
    }),
  );
  if (!session.enabled || session.authenticated) {
    throw new PiTransportError("The Pi pilot data was not deleted.");
  }
  return session;
}

export async function requestWalletChallenge(
  address: string,
  csrfToken: string,
): Promise<PiWalletChallenge> {
  if (!ZERONE_ADDRESS.test(address)) {
    throw new PiTransportError("The Zerone wallet address is invalid.");
  }
  return parseWalletChallenge(
    await piRequest("/api/pi/challenge", {
      method: "POST",
      body: { address },
      csrfToken,
    }),
  );
}

export async function bindWalletProof(
  challengeId: string,
  signature: PiWalletProofSignature,
  csrfToken: string,
): Promise<PiSession> {
  const session = parsePiSession(
    await piRequest("/api/pi/bind", {
      method: "POST",
      body: { challengeId, signature },
      csrfToken,
    }),
  );
  if (!session.enabled || !session.authenticated || !session.linked) {
    throw new PiTransportError(
      "The Pi pilot did not record the wallet proof.",
    );
  }
  return session;
}

export async function removeWalletProof(
  csrfToken: string,
): Promise<PiSession> {
  const session = parsePiSession(
    await piRequest("/api/pi/bind", {
      method: "DELETE",
      body: {},
      csrfToken,
    }),
  );
  if (!session.enabled || !session.authenticated || session.linked) {
    throw new PiTransportError(
      "The Pi pilot did not remove the wallet proof.",
    );
  }
  return session;
}
