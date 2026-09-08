import configuredProfile from "./network-profile.json";

export interface Checkpoint {
  height: number;
  blockHash: string;
  appHash: string;
}

export interface ObserverProfile {
  mode: "beta-active" | "beta-archive" | "preview";
  chainId: string;
  gatewayOrigin: string;
  genesisSha256: string;
  releaseCommit: string;
  releaseManifestSha256: string;
  releaseUrl: string;
  // An operator assertion that external verification already occurred, NOT a
  // signature verifier or an OPEN authorization supplied by this application.
  operatorVerification: {
    openDecisionSha256: string;
    verifiedAt: string;
  };
  checkpoint?: Checkpoint;
}

export type NetworkProfile = ObserverProfile | { mode: "legacy" } |
  { mode: "unconfigured"; error: string };

const HASH = /^[a-f0-9]{64}$/;
function record(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
function keys(value: Record<string, unknown>, allowed: string[]): void {
  if (Object.keys(value).some((key) => !allowed.includes(key))) {
    throw new Error("Profile contains unsupported fields");
  }
}
function requireValue(condition: unknown, message: string): asserts condition {
  if (!condition) throw new Error(message);
}
function publicHttps(value: unknown): URL {
  requireValue(typeof value === "string" && value.length <= 2048, "Expected public HTTPS URL");
  const url = new URL(value);
  requireValue(url.protocol === "https:" && !url.username && !url.password &&
    !url.hash && !url.search && !url.port &&
    url.hostname.length <= 253 && /^[a-z0-9.-]+\.[a-z]{2,}$/i.test(url.hostname) &&
    url.hostname.split(".").every((label) => /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/i.test(label)) &&
    !/(?:^|\.)(?:localhost|local|internal|invalid|test|example)$/.test(url.hostname),
  "Expected credential-free public HTTPS URL (no query, fragment, port or private host)");
  return url;
}

export function parseNetworkProfile(value: unknown): NetworkProfile {
  try {
    requireValue(record(value), "Profile must be an object");
    if (value.mode === "legacy") {
      keys(value, ["mode"]);
      return { mode: "legacy" };
    }
    requireValue(value.mode === "beta-active" || value.mode === "beta-archive" || value.mode === "preview", "Select an explicit network profile");
    keys(value, ["mode", "chainId", "gatewayOrigin", "genesisSha256", "releaseCommit", "releaseManifestSha256", "releaseUrl", "operatorVerification", "checkpoint"]);
    requireValue(typeof value.chainId === "string" && /^[a-zA-Z0-9][a-zA-Z0-9._-]{0,49}$/.test(value.chainId), "Invalid chain identity");
    for (const key of ["genesisSha256", "releaseManifestSha256"]) {
      requireValue(typeof value[key] === "string" && HASH.test(value[key]), `Missing or invalid ${key}`);
    }
    requireValue(typeof value.releaseCommit === "string" && /^[a-f0-9]{40}$/.test(value.releaseCommit), "Expected immutable release commit");
    requireValue(typeof value.gatewayOrigin === "string", "Missing approved query gateway origin");
    if (value.mode === "preview") {
      const gateway = new URL(value.gatewayOrigin);
      requireValue(gateway.protocol === "http:" && ["localhost", "127.0.0.1", "[::1]"].includes(gateway.hostname) && !gateway.username && !gateway.password && value.gatewayOrigin === gateway.origin,
        "Preview requires an explicit loopback HTTP query gateway");
      requireValue(value.releaseUrl === "https://example.invalid/synthetic-release", "Preview requires the synthetic release marker");
    } else {
      const gateway = publicHttps(value.gatewayOrigin);
      requireValue(value.gatewayOrigin === gateway.origin, "Gateway must be one exact approved HTTPS origin");
      publicHttps(value.releaseUrl);
    }
    requireValue(record(value.operatorVerification), "Operator-supplied verified release information is required");
    keys(value.operatorVerification, ["openDecisionSha256", "verifiedAt"]);
    requireValue(typeof value.operatorVerification.openDecisionSha256 === "string" && HASH.test(value.operatorVerification.openDecisionSha256), "Missing OPEN decision digest");
    requireValue(typeof value.operatorVerification.verifiedAt === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/.test(value.operatorVerification.verifiedAt) && Number.isFinite(Date.parse(value.operatorVerification.verifiedAt)), "Missing verification timestamp");
    requireValue(new Date(value.operatorVerification.verifiedAt).toISOString() === value.operatorVerification.verifiedAt.replace("Z", ".000Z"), "Invalid verification calendar date");
    if (value.mode === "beta-archive" || value.checkpoint !== undefined) {
      requireValue(value.mode !== "beta-active" && record(value.checkpoint), "Archive requires its fixed checkpoint");
      keys(value.checkpoint, ["height", "blockHash", "appHash"]);
      requireValue(Number.isSafeInteger(value.checkpoint.height) && Number(value.checkpoint.height) > 0 && Number(value.checkpoint.height) <= 1_000_000_000, "Invalid checkpoint height");
      for (const key of ["blockHash", "appHash"]) {
        requireValue(typeof value.checkpoint[key] === "string" && HASH.test(value.checkpoint[key]), "Invalid checkpoint hash");
      }
    }
    return structuredClone(value) as unknown as ObserverProfile;
  } catch (error) {
    return { mode: "unconfigured", error: error instanceof Error ? error.message : "Invalid profile" };
  }
}

export const NETWORK_PROFILE = parseNetworkProfile(configuredProfile);
export function observerLabel(profile: NetworkProfile): string {
  switch (profile.mode) {
    case "legacy": return "Legacy dashboard · zerone-1";
    case "beta-active": return "Public custodial beta · active observer";
    case "beta-archive": return "Frozen predecessor archive · read-only";
    case "preview": return "LOCAL / SYNTHETIC PREVIEW · NOT PRODUCTION";
    case "unconfigured": return "Observer unconfigured · network access disabled";
  }
}
