/// <reference lib="dom" />

export const LIFE_SCIENCES_TREE_ENDPOINT =
  "/standards/constructive-intelligence-life-sciences.v0.json";
export const LIFE_SCIENCES_TREE_SHA256 =
  "64dc2c5b2e21dfc9697d173317254ce651dede8661993ece7b380b7e1421496e";
export const LIFE_SCIENCES_TREE_MAX_BYTES = 196_608;

const SCHEMA = "zerone.constructive-intelligence-life-sciences/v0";
const BASE_TREE_SHA256 =
  "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf";
const MONEY_KARMA_CONSTITUTION_SCHEMA = "zerone.money-karma.constitution/v1";
const MONEY_KARMA_CONSTITUTION_DOCUMENT_SHA256 =
  "sha256:f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24";
const STAGES = [
  "foundation",
  "measurement",
  "inference",
  "validation",
  "integration",
  "crown",
] as const;
const BRANCHES = [
  "biomolecule-foundations",
  "protein-folding",
  "gene-expression",
  "integration",
] as const;
const EVIDENCE_LEVELS = ["LS0", "LS1", "LS2", "LS3", "LS4", "LS5"] as const;

const TOP_LEVEL_KEYS = [
  "schema",
  "status",
  "mode",
  "authoritative",
  "networkObserved",
  "rewardBearing",
  "snapshotDate",
  "baseTreeBinding",
  "constitutionBinding",
  "releaseBoundary",
  "economics",
  "scope",
  "fixtureBoundary",
  "evidencePolicy",
  "independence",
  "challengePolicy",
  "attestationBoundary",
  "nonImplicationWalls",
  "references",
  "roots",
  "nodes",
] as const;
const BASE_TREE_BINDING_KEYS = ["schema", "sha256", "policyVersion"] as const;
const CONSTITUTION_BINDING_KEYS = ["schema", "documentSha256"] as const;
const RELEASE_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "grantsGovernanceAuthority",
  "createsKarmaMagnitude",
  "createsTransferableKarma",
  "authorizesExperiments",
  "authorizesClinicalDecisions",
  "performsNetworkRequests",
  "publishesRestrictedEvidence",
  "marksReleaseReady",
] as const;
const ECONOMICS_KEYS = [
  "effect",
  "amount",
  "denom",
  "rewardMultiplier",
  "escrowReference",
] as const;
const SCOPE_KEYS = [
  "riskClass",
  "allowedWork",
  "refusedTopics",
  "unknownRiskDisposition",
  "privateEscalationContent",
] as const;
const FIXTURE_BOUNDARY_KEYS = [
  "syntheticIdentifiersOnly",
  "invalidTldRequired",
  "sequencePayloadsAllowed",
  "operationalProtocolsAllowed",
  "identifiableHumanDataAllowed",
  "rawHumanGenomeAllowed",
  "maximumArtifactMetadataBytes",
] as const;
const EVIDENCE_POLICY_KEYS = [
  "equivalenceToCoreTree",
  "equivalenceToPoca",
  "ladder",
] as const;
const EVIDENCE_LEVEL_KEYS = ["level", "name", "economicTreatment"] as const;
const INDEPENDENCE_KEYS = [
  "minimumEffectiveControlClusters",
  "minimumOrganizationRoots",
  "minimumMethodRoots",
  "minimumContextRoots",
  "rawIdentityCountIsEvidence",
  "beneficialControlDisclosureRequired",
  "futureEligibilityRequiresExternalControllerAttestation",
] as const;
const CHALLENGE_POLICY_KEYS = [
  "openChallengeRequired",
  "unresolvedChallengeBlocksCrown",
  "upheldChallengeBlocksCrown",
  "challengeCanCreateReward",
  "futureEligibilityRequiresAdjudicationReceipt",
] as const;
const ATTESTATION_BOUNDARY_KEYS = [
  "controlDisclosures",
  "challengeStatus",
  "establishesControllerIndependence",
  "establishesChallengeClosure",
] as const;
const NON_IMPLICATION_WALL_KEYS = ["id", "premise", "doesNotEstablish"] as const;
const REFERENCE_KEYS = ["canonicalId", "authority", "title", "specification"] as const;
const NODE_KEYS = [
  "id",
  "title",
  "stage",
  "branch",
  "summary",
  "prerequisites",
  "evidenceFloor",
  "claimClass",
  "permittedConclusions",
  "forbiddenConclusions",
  "artifactRequirements",
  "revalidationTriggers",
  "referenceIds",
  "crown",
] as const;

type Stage = (typeof STAGES)[number];
type Branch = (typeof BRANCHES)[number];
type EvidenceLevel = (typeof EVIDENCE_LEVELS)[number];

export interface LifeSciencesNode {
  id: string;
  title: string;
  stage: Stage;
  branch: Branch;
  summary: string;
  prerequisites: string[];
  evidenceFloor: EvidenceLevel;
  claimClass: string;
  permittedConclusions: string[];
  forbiddenConclusions: string[];
  crown: boolean;
}
interface NonImplicationWall {
  id: string;
  premise: string;
  doesNotEstablish: string[];
}

export interface LifeSciencesOverlay {
  schema: typeof SCHEMA;
  status: "DRAFT";
  mode: "SHADOW_ONLY";
  snapshotDate: string;
  baseTreeSha256: string;
  constitutionDocumentSha256: typeof MONEY_KARMA_CONSTITUTION_DOCUMENT_SHA256;
  riskClass: "GREEN_ONLY";
  refusedTopics: string[];
  minimumEffectiveControlClusters: number;
  independenceStatus: "DECLARED_UNVERIFIED";
  challengeStatus: "DECLARED_UNVERIFIED";
  nonImplicationWalls: NonImplicationWall[];
  nodes: LifeSciencesNode[];
}

export interface LifeSciencesTreeFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class LifeSciencesTreeError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "LifeSciencesTreeError";
  }
}

function fail(path: string, message: string): never {
  throw new LifeSciencesTreeError(`${path}: ${message}`);
}

function object(value: unknown, path: string): Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  return value as Record<string, unknown>;
}

function exactKeys(
  value: Record<string, unknown>,
  expected: readonly string[],
  path: string,
): void {
  const expectedSet = new Set(expected);
  for (const key of Object.keys(value)) {
    if (!expectedSet.has(key)) fail(`${path}.${key}`, "unknown field");
  }
  for (const key of expected) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "required field is missing");
  }
}

function text(value: unknown, path: string, max = 1_024): string {
  if (typeof value !== "string" || value.length === 0 || value.length > max) {
    fail(path, `must be a non-empty string no longer than ${max} characters`);
  }
  return value;
}

function stringArray(value: unknown, path: string, max = 64): string[] {
  if (!Array.isArray(value) || value.length > max) fail(path, "must be a bounded array");
  return value.map((item, index) => text(item, `${path}[${index}]`, 512));
}

function exactFalse(value: unknown, path: string): false {
  if (value !== false) fail(path, "must remain false");
  return false;
}

function exactTrue(value: unknown, path: string): true {
  if (value !== true) fail(path, "must remain true");
  return true;
}

function member<T extends readonly string[]>(
  value: unknown,
  allowed: T,
  path: string,
): T[number] {
  if (typeof value !== "string" || !allowed.includes(value as T[number])) {
    fail(path, `must be one of ${allowed.join(", ")}`);
  }
  return value as T[number];
}

export function parseLifeSciencesOverlay(value: unknown): LifeSciencesOverlay {
  const root = object(value, "$");
  exactKeys(root, TOP_LEVEL_KEYS, "$");
  if (root.schema !== SCHEMA) fail("$.schema", "unsupported schema");
  if (root.status !== "DRAFT") fail("$.status", "must remain DRAFT");
  if (root.mode !== "SHADOW_ONLY") fail("$.mode", "must remain SHADOW_ONLY");
  exactFalse(root.authoritative, "$.authoritative");
  exactFalse(root.networkObserved, "$.networkObserved");
  exactFalse(root.rewardBearing, "$.rewardBearing");

  const binding = object(root.baseTreeBinding, "$.baseTreeBinding");
  exactKeys(binding, BASE_TREE_BINDING_KEYS, "$.baseTreeBinding");
  if (binding.sha256 !== BASE_TREE_SHA256) {
    fail("$.baseTreeBinding.sha256", "core-tree binding drifted");
  }

  const constitutionBinding = object(root.constitutionBinding, "$.constitutionBinding");
  exactKeys(
    constitutionBinding,
    CONSTITUTION_BINDING_KEYS,
    "$.constitutionBinding",
  );
  if (constitutionBinding.schema !== MONEY_KARMA_CONSTITUTION_SCHEMA) {
    fail("$.constitutionBinding.schema", "Money-KARMA constitution schema drifted");
  }
  if (
    constitutionBinding.documentSha256 !== MONEY_KARMA_CONSTITUTION_DOCUMENT_SHA256
  ) {
    fail("$.constitutionBinding.documentSha256", "Money-KARMA constitution pin drifted");
  }

  const boundary = object(root.releaseBoundary, "$.releaseBoundary");
  exactKeys(boundary, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of [
    "addsConsensusBehavior",
    "activatesRewards",
    "movesFunds",
    "grantsQualification",
    "grantsGovernanceAuthority",
    "createsKarmaMagnitude",
    "createsTransferableKarma",
    "authorizesExperiments",
    "authorizesClinicalDecisions",
    "performsNetworkRequests",
    "publishesRestrictedEvidence",
    "marksReleaseReady",
  ]) {
    exactFalse(boundary[key], `$.releaseBoundary.${key}`);
  }

  const economics = object(root.economics, "$.economics");
  exactKeys(economics, ECONOMICS_KEYS, "$.economics");
  if (
    economics.effect !== "NONE" ||
    economics.amount !== "0" ||
    economics.denom !== null ||
    economics.escrowReference !== null
  ) {
    fail("$.economics", "must remain NONE/0 with no denomination or escrow");
  }
  exactFalse(economics.rewardMultiplier, "$.economics.rewardMultiplier");

  const scope = object(root.scope, "$.scope");
  exactKeys(scope, SCOPE_KEYS, "$.scope");
  if (scope.riskClass !== "GREEN_ONLY") fail("$.scope.riskClass", "must remain GREEN_ONLY");
  const refusedTopics = stringArray(scope.refusedTopics, "$.scope.refusedTopics", 32);
  for (const topic of [
    "PATHOGENS",
    "TOXINS",
    "VIRULENCE",
    "IMMUNE_EVASION",
    "HOST_RANGE",
    "GERMLINE",
    "CLINICAL_DECISIONS",
    "IDENTIFIABLE_HUMAN_DATA",
    "RAW_HUMAN_GENOME",
    "OPERATIONAL_WET_LAB_ENABLEMENT",
  ]) {
    if (!refusedTopics.includes(topic)) fail("$.scope.refusedTopics", `missing ${topic}`);
  }

  const fixtureBoundary = object(root.fixtureBoundary, "$.fixtureBoundary");
  exactKeys(fixtureBoundary, FIXTURE_BOUNDARY_KEYS, "$.fixtureBoundary");

  const evidencePolicy = object(root.evidencePolicy, "$.evidencePolicy");
  exactKeys(evidencePolicy, EVIDENCE_POLICY_KEYS, "$.evidencePolicy");
  if (!Array.isArray(evidencePolicy.ladder)) {
    fail("$.evidencePolicy.ladder", "must be an array");
  }
  evidencePolicy.ladder.forEach((item, index) => {
    const level = object(item, `$.evidencePolicy.ladder[${index}]`);
    exactKeys(level, EVIDENCE_LEVEL_KEYS, `$.evidencePolicy.ladder[${index}]`);
  });
  if (
    evidencePolicy.equivalenceToCoreTree !== "NO_EQUIVALENCE_TO_E0_E6" ||
    evidencePolicy.equivalenceToPoca !== "NO_EQUIVALENCE_TO_POCA_TIERS"
  ) {
    fail("$.evidencePolicy", "LS levels must remain non-equivalent to core and PoCA tiers");
  }

  const independence = object(root.independence, "$.independence");
  exactKeys(independence, INDEPENDENCE_KEYS, "$.independence");
  if (independence.minimumEffectiveControlClusters !== 3) {
    fail("$.independence.minimumEffectiveControlClusters", "must remain 3");
  }
  exactFalse(independence.rawIdentityCountIsEvidence, "$.independence.rawIdentityCountIsEvidence");
  exactTrue(
    independence.futureEligibilityRequiresExternalControllerAttestation,
    "$.independence.futureEligibilityRequiresExternalControllerAttestation",
  );

  const challengePolicy = object(root.challengePolicy, "$.challengePolicy");
  exactKeys(challengePolicy, CHALLENGE_POLICY_KEYS, "$.challengePolicy");
  exactTrue(
    challengePolicy.futureEligibilityRequiresAdjudicationReceipt,
    "$.challengePolicy.futureEligibilityRequiresAdjudicationReceipt",
  );

  const attestationBoundary = object(root.attestationBoundary, "$.attestationBoundary");
  exactKeys(
    attestationBoundary,
    ATTESTATION_BOUNDARY_KEYS,
    "$.attestationBoundary",
  );
  if (attestationBoundary.controlDisclosures !== "SELF_DECLARED_SYNTHETIC_LABELS") {
    fail(
      "$.attestationBoundary.controlDisclosures",
      "must remain SELF_DECLARED_SYNTHETIC_LABELS",
    );
  }
  if (attestationBoundary.challengeStatus !== "SELF_DECLARED_SYNTHETIC_LABEL") {
    fail(
      "$.attestationBoundary.challengeStatus",
      "must remain SELF_DECLARED_SYNTHETIC_LABEL",
    );
  }
  exactFalse(
    attestationBoundary.establishesControllerIndependence,
    "$.attestationBoundary.establishesControllerIndependence",
  );
  exactFalse(
    attestationBoundary.establishesChallengeClosure,
    "$.attestationBoundary.establishesChallengeClosure",
  );

  if (!Array.isArray(root.nonImplicationWalls) || root.nonImplicationWalls.length !== 4) {
    fail("$.nonImplicationWalls", "must contain the four scientific inference walls");
  }
  const nonImplicationWalls = root.nonImplicationWalls.map((item, index) => {
    const wall = object(item, `$.nonImplicationWalls[${index}]`);
    exactKeys(
      wall,
      NON_IMPLICATION_WALL_KEYS,
      `$.nonImplicationWalls[${index}]`,
    );
    return {
      id: text(wall.id, `$.nonImplicationWalls[${index}].id`, 128),
      premise: text(wall.premise, `$.nonImplicationWalls[${index}].premise`, 128),
      doesNotEstablish: stringArray(
        wall.doesNotEstablish,
        `$.nonImplicationWalls[${index}].doesNotEstablish`,
        8,
      ),
    };
  });

  if (!Array.isArray(root.references)) fail("$.references", "must be an array");
  root.references.forEach((item, index) => {
    const reference = object(item, `$.references[${index}]`);
    exactKeys(reference, REFERENCE_KEYS, `$.references[${index}]`);
  });

  if (!Array.isArray(root.nodes) || root.nodes.length < 12 || root.nodes.length > 20) {
    fail("$.nodes", "must contain 12 through 20 nodes");
  }
  const nodes = root.nodes.map((item, index): LifeSciencesNode => {
    const node = object(item, `$.nodes[${index}]`);
    exactKeys(node, NODE_KEYS, `$.nodes[${index}]`);
    if (typeof node.crown !== "boolean") fail(`$.nodes[${index}].crown`, "must be boolean");
    return {
      id: text(node.id, `$.nodes[${index}].id`, 128),
      title: text(node.title, `$.nodes[${index}].title`, 256),
      stage: member(node.stage, STAGES, `$.nodes[${index}].stage`),
      branch: member(node.branch, BRANCHES, `$.nodes[${index}].branch`),
      summary: text(node.summary, `$.nodes[${index}].summary`, 1_024),
      prerequisites: stringArray(node.prerequisites, `$.nodes[${index}].prerequisites`, 4),
      evidenceFloor: member(
        node.evidenceFloor,
        EVIDENCE_LEVELS,
        `$.nodes[${index}].evidenceFloor`,
      ),
      claimClass: text(node.claimClass, `$.nodes[${index}].claimClass`, 128),
      permittedConclusions: stringArray(
        node.permittedConclusions,
        `$.nodes[${index}].permittedConclusions`,
        16,
      ),
      forbiddenConclusions: stringArray(
        node.forbiddenConclusions,
        `$.nodes[${index}].forbiddenConclusions`,
        16,
      ),
      crown: node.crown,
    };
  });

  const byId = new Map<string, LifeSciencesNode>();
  for (const node of nodes) {
    if (byId.has(node.id)) fail("$.nodes", `duplicate node ${node.id}`);
    byId.set(node.id, node);
  }
  for (const node of nodes) {
    for (const prerequisite of node.prerequisites) {
      if (!byId.has(prerequisite)) fail("$.nodes", `${node.id} has dangling prerequisite ${prerequisite}`);
    }
  }
  if (nodes.filter((node) => node.crown).length !== 1) {
    fail("$.nodes", "must contain exactly one crown");
  }

  return {
    schema: SCHEMA,
    status: "DRAFT",
    mode: "SHADOW_ONLY",
    snapshotDate: text(root.snapshotDate, "$.snapshotDate", 10),
    baseTreeSha256: BASE_TREE_SHA256,
    constitutionDocumentSha256: MONEY_KARMA_CONSTITUTION_DOCUMENT_SHA256,
    riskClass: "GREEN_ONLY",
    refusedTopics,
    minimumEffectiveControlClusters: 3,
    independenceStatus: "DECLARED_UNVERIFIED",
    challengeStatus: "DECLARED_UNVERIFIED",
    nonImplicationWalls,
    nodes,
  };
}

function hex(bytes: ArrayBuffer): string {
  return Array.from(new Uint8Array(bytes), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new LifeSciencesTreeError(
      "Life-science overlay returned an empty response body",
    );
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let length = 0;
  let rejectOnAbort: ((reason?: unknown) => void) | undefined;
  const aborted = new Promise<never>((_resolve, reject) => {
    rejectOnAbort = reject;
  });
  const onAbort = (): void => {
    rejectOnAbort?.(
      signal.reason ??
        new DOMException("Life-science overlay request timed out", "TimeoutError"),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      length += value.byteLength;
      if (length > LIFE_SCIENCES_TREE_MAX_BYTES) {
        void reader.cancel().catch(() => {
          // Refusal must not wait for a hostile stream to accept cancellation.
        });
        throw new LifeSciencesTreeError(
          "Life-science overlay exceeds its byte limit",
        );
      }
      chunks.push(value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => {
        // The deadline wins even if cancellation stalls or rejects.
      });
      throw new LifeSciencesTreeError(
        "Life-science overlay request timed out",
      );
    }
    throw error;
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A still-pending hostile read is abandoned after refusal.
    }
  }
  const bytes = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return bytes;
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new LifeSciencesTreeError(
      "Life-science overlay digest verification is unavailable",
    );
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  return hex(await globalThis.crypto.subtle.digest("SHA-256", input));
}

function assertCanonicalResponseUrl(
  response: Response,
  baseUrl?: string,
): void {
  if (response.redirected) {
    throw new LifeSciencesTreeError(
      "Life-science overlay response was redirected",
    );
  }
  if (baseUrl === undefined) return;
  let expected: URL;
  let actual: URL;
  try {
    expected = new URL(LIFE_SCIENCES_TREE_ENDPOINT, baseUrl);
    actual = new URL(response.url);
  } catch {
    throw new LifeSciencesTreeError(
      "Life-science overlay returned an invalid final URL",
    );
  }
  if (
    actual.origin !== expected.origin ||
    actual.pathname !== expected.pathname ||
    actual.search !== expected.search ||
    actual.hash !== expected.hash
  ) {
    throw new LifeSciencesTreeError(
      "Life-science overlay left its canonical same-origin path",
    );
  }
}

function rejectDuplicateJsonKeys(raw: string): void {
  let offset = 0;
  const whitespace = (): void => {
    while (/\s/.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = (): string => {
    const start = offset;
    offset += 1;
    let escaped = false;
    while (offset < raw.length) {
      const character = raw[offset];
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset)) as string;
      }
      offset += 1;
    }
    fail("$", "unterminated JSON string");
  };
  const scanValue = (path: string, depth = 0): void => {
    if (depth > 64) fail(path, "JSON nesting exceeds 64");
    whitespace();
    const token = raw[offset];
    if (token === "{") {
      offset += 1;
      whitespace();
      const keys = new Set<string>();
      if (raw[offset] === "}") {
        offset += 1;
        return;
      }
      while (offset < raw.length) {
        whitespace();
        if (raw[offset] !== '"') fail(path, "malformed object key");
        const key = scanString();
        if (keys.has(key)) fail(`${path}.${key}`, "duplicate JSON object key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "malformed object separator");
        offset += 1;
        scanValue(`${path}.${key}`, depth + 1);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "malformed object delimiter");
        offset += 1;
      }
      fail(path, "unterminated object");
    }
    if (token === "[") {
      offset += 1;
      whitespace();
      if (raw[offset] === "]") {
        offset += 1;
        return;
      }
      let index = 0;
      while (offset < raw.length) {
        scanValue(`${path}[${index}]`, depth + 1);
        whitespace();
        if (raw[offset] === "]") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "malformed array delimiter");
        offset += 1;
        index += 1;
      }
      fail(path, "unterminated array");
    }
    if (token === '"') {
      scanString();
      return;
    }
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset] ?? "")) offset += 1;
  };
  scanValue("$");
}

export function parseLifeSciencesOverlayJson(raw: string): LifeSciencesOverlay {
  if (new TextEncoder().encode(raw).byteLength > LIFE_SCIENCES_TREE_MAX_BYTES) {
    fail("$", `document exceeds ${LIFE_SCIENCES_TREE_MAX_BYTES} UTF-8 bytes`);
  }
  try {
    const parsed = JSON.parse(raw) as unknown;
    rejectDuplicateJsonKeys(raw);
    return parseLifeSciencesOverlay(parsed);
  } catch (error) {
    if (error instanceof LifeSciencesTreeError) throw error;
    fail("$", "malformed JSON");
  }
}

export async function fetchLifeSciencesOverlay(
  options: LifeSciencesTreeFetchOptions = {},
): Promise<LifeSciencesOverlay> {
  const fetcher = options.fetcher ?? globalThis.fetch;
  if (fetcher === undefined) {
    throw new LifeSciencesTreeError("Life-science overlay is unavailable");
  }
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(() => {
    controller.abort(
      new DOMException("Life-science overlay request timed out", "TimeoutError"),
    );
  }, options.timeoutMs ?? 8_000);
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? undefined : window.location.href);
  try {
    let response: Response;
    let rejectFetchOnAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchOnAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchOnAbort?.(
        controller.signal.reason ??
          new DOMException("Life-science overlay request timed out", "TimeoutError"),
      );
    };
    controller.signal.addEventListener("abort", onFetchAbort, { once: true });
    if (controller.signal.aborted) onFetchAbort();
    try {
      response = await Promise.race([
        fetcher(LIFE_SCIENCES_TREE_ENDPOINT, {
          cache: "no-store",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          redirect: "error",
          signal: controller.signal,
        }),
        fetchAborted,
      ]);
    } catch (error) {
      if (
        controller.signal.aborted ||
        (error instanceof DOMException &&
          (error.name === "AbortError" || error.name === "TimeoutError"))
      ) {
        throw new LifeSciencesTreeError(
          "Life-science overlay request timed out",
        );
      }
      throw new LifeSciencesTreeError("Life-science overlay is unavailable");
    } finally {
      controller.signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok) {
      throw new LifeSciencesTreeError(
        `Life-science overlay returned HTTP ${response.status}`,
      );
    }
    assertCanonicalResponseUrl(response, baseUrl);
    const contentType = response.headers.get("content-type");
    if (contentType === null || !/\bjson\b/i.test(contentType)) {
      throw new LifeSciencesTreeError(
        "Life-science overlay returned a non-JSON response",
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (declaredLength !== null) {
      const length = Number(declaredLength);
      if (
        !/^\d+$/.test(declaredLength) ||
        !Number.isSafeInteger(length) ||
        length > LIFE_SCIENCES_TREE_MAX_BYTES
      ) {
        throw new LifeSciencesTreeError(
          "Life-science overlay exceeds its byte limit",
        );
      }
    }
    const bytes = await readBoundedResponse(response, controller.signal);
    if ((await sha256Hex(bytes)) !== LIFE_SCIENCES_TREE_SHA256) {
      throw new LifeSciencesTreeError(
        "Life-science overlay bytes do not match the reviewed SHA-256",
      );
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new LifeSciencesTreeError(
        "Life-science overlay is not valid UTF-8",
      );
    }
    return parseLifeSciencesOverlayJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  copy?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (copy !== undefined) node.textContent = copy;
  return node;
}

function label(value: string): string {
  return value
    .toLowerCase()
    .split(/[-_]/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function renderOverlay(root: HTMLElement, overlay: LifeSciencesOverlay): void {
  const shell = el("div", "ls-tree");
  const facts = el("div", "ls-tree-facts");
  for (const [name, value] of [
    ["Profile", `${overlay.status} · ${overlay.mode.replaceAll("_", " ")}`],
    ["Graph", `${overlay.nodes.length} nodes · 4 branches`],
    ["Evidence", "LS0–LS5 · no tier equivalence"],
    ["Economics", "NONE · amount 0 · no denomination"],
  ]) {
    const item = el("div");
    item.append(el("span", undefined, name), el("strong", undefined, value));
    facts.append(item);
  }

  const boundary = el("div", "ls-tree-boundary");
  boundary.append(
    el("strong", undefined, "GREEN-only shadow map"),
    el(
      "p",
      undefined,
      `No experiments, clinical decisions, qualification, governance, rewards, or funds. Control roots and challenge state are self-declared synthetic labels, not verified independence or closure. A structural crown match would still never be reward-eligible.`,
    ),
  );

  const walls = el("div", "ls-tree-walls");
  overlay.nonImplicationWalls.forEach((wall) => {
    const card = el("article");
    card.append(
      el("span", undefined, label(wall.premise)),
      el("strong", undefined, "≠"),
      el("p", undefined, wall.doesNotEstablish.map(label).join(" or ")),
    );
    walls.append(card);
  });

  const map = el("div", "ls-tree-map");
  for (const branch of BRANCHES) {
    const section = el("section", `ls-tree-branch branch-${branch}`);
    const branchNodes = overlay.nodes.filter((node) => node.branch === branch);
    section.append(
      el("span", "card-kicker", label(branch)),
      el("h3", undefined, branch === "integration" ? "Cross-domain bosses" : label(branch)),
    );
    const list = el("ol");
    branchNodes.forEach((node) => {
      const item = el("li");
      const details = el("details", node.crown ? "is-crown" : undefined);
      const summary = el("summary");
      const meta = el("span", "ls-tree-node-meta");
      meta.append(
        el("span", undefined, node.evidenceFloor),
        el("span", undefined, label(node.stage)),
      );
      summary.append(meta, el("strong", undefined, node.title));
      const body = el("div", "ls-tree-node-body");
      body.append(el("p", undefined, node.summary));
      if (node.prerequisites.length > 0) {
        body.append(
          el(
            "small",
            undefined,
            `Requires: ${node.prerequisites.map((id) => overlay.nodes.find((candidate) => candidate.id === id)?.title ?? id).join(" · ")}`,
          ),
        );
      }
      if (node.forbiddenConclusions.length > 0) {
        body.append(
          el(
            "small",
            "ls-tree-refusal",
            `Does not establish: ${node.forbiddenConclusions.map(label).join(" · ")}`,
          ),
        );
      }
      details.append(summary, body);
      item.append(details);
      list.append(item);
    });
    section.append(list);
    map.append(section);
  }

  const footer = el("div", "ls-tree-footer");
  const raw = el("a", "button button-ghost", "Open profile JSON ↗");
  raw.href = LIFE_SCIENCES_TREE_ENDPOINT;
  raw.target = "_blank";
  raw.rel = "noreferrer";
  footer.append(
    el(
      "p",
      undefined,
      `Static snapshot ${overlay.snapshotDate} · base tree ${overlay.baseTreeSha256.slice(0, 12)}… · not scientific truth or payment state.`,
    ),
    raw,
  );

  shell.append(facts, boundary, walls, map, footer);
  root.replaceChildren(shell);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseLifeSciencesTree(root: HTMLElement): Promise<void> {
  root.setAttribute("aria-busy", "true");
  try {
    renderOverlay(root, await fetchLifeSciencesOverlay());
  } catch (error) {
    const failure = el("div", "ci-tree-load-error");
    failure.setAttribute("role", "alert");
    failure.append(
      el("strong", undefined, "The life-science overlay could not be verified."),
      el("p", undefined, error instanceof Error ? error.message : "The static profile is unavailable."),
    );
    root.replaceChildren(failure);
    root.setAttribute("aria-busy", "false");
  }
}
