/// <reference lib="dom" />

export const FOLD_TO_FIRE_ENDPOINT =
  "/standards/constructive-intelligence-fold-to-fire.v0.json";
export const FOLD_TO_FIRE_MAX_BYTES = 65_536;
export const FOLD_TO_FIRE_SHA256 =
  "3fb78beaec220b4f62219a120ea33f46cfbe5ca1e76286929ae7b1120ccf4033";

const TOP_LEVEL_KEYS = [
  "schema",
  "status",
  "mode",
  "title",
  "snapshotDate",
  "authoritative",
  "networkObserved",
  "sourceBindings",
  "releaseBoundary",
  "economics",
  "modelBoundary",
  "enumeration",
  "frontierProblem",
  "weightedBridge",
  "nonImplicationWalls",
  "sources",
] as const;

const RELEASE_BOUNDARY_KEYS = [
  "changesConsensus",
  "writesNetworkState",
  "performsNetworkRequests",
  "activatesRewards",
  "movesFunds",
  "createsKarmaEvent",
  "createsKarmaMagnitude",
  "grantsQualification",
  "grantsGovernance",
  "grantsAuthority",
  "ranksPersons",
  "authorizesExperiment",
  "authorizesWetLab",
  "authorizesClinicalDecision",
  "containsSequencePayload",
  "claimsProteinPrediction",
  "claimsCatalystDesign",
] as const;

const EXPECTED_ROWS = [
  [3, ["7", "2"], ["0", "2"], "9", "2", "2/9"],
  [5, ["41", "22", "8"], ["0", "0", "6"], "71", "6", "6/71"],
  [7, ["235", "184", "86", "38"], ["0", "4", "0", "24"], "543", "28", "28/543"],
  [9, ["1331", "1344", "850", "346", "196"], ["0", "10", "40", "0", "90"], "4067", "140", "20/581"],
  [11, ["7485", "9244", "6900", "3888", "1606", "888", "62"], ["0", "54", "120", "240", "0", "306", "24"], "30073", "744", "744/30073"],
  [13, ["41867", "60884", "52934", "33472", "19076", "7444", "3978", "720"], ["0", "252", "672", "770", "1092", "112", "966", "252"], "220375", "4116", "4116/220375"],
  [15, ["233157", "389792", "383628", "276892", "169214", "91128", "37466", "17324", "5410", "138"], ["0", "1232", "3264", "4496", "3904", "4672", "1408", "2976", "1504", "48"], "1604149", "23504", "23504/1604149"],
] as const;

const EXPECTED_SOURCE_URLS = [
  "https://doi.org/10.1214/14-AOP993",
  "https://arxiv.org/abs/1504.05286",
  "https://doi.org/10.1088/0305-4470/37/21/002",
  "https://doi.org/10.1088/1751-8121/ac943a",
  "https://doi.org/10.1088/0305-4470/31/20/010",
  "https://doi.org/10.1073/pnas.0803405105",
  "https://doi.org/10.1038/329268a0",
  "https://doi.org/10.1038/365185a0",
  "https://doi.org/10.1038/s41589-018-0013-8",
] as const;

const EXPECTED_BINDING_DIGESTS = [
  [
    "constructive-intelligence-math-frontier-v0",
    "4fdcd54c35c69c26a28c385275688351ee2a9131702e81bacf100de8d7612456",
    "b6260de31969a56e601a5a81f1b4f7c1c68fcd34f9aa68b35ce2701c3f012503",
  ],
  [
    "constructive-intelligence-life-sciences-v0",
    "64dc2c5b2e21dfc9697d173317254ce651dede8661993ece7b380b7e1421496e",
    "a208ea9e30a16ccfbb74f3f19298a5d3f93d7f87273b0b5aa10bf72e0e708822",
  ],
  [
    "money-karma-v1",
    "f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24",
    "a41286c936d3ab83d1cbd782b119cf3b434518ba80859edfe76f0de184143b7b",
  ],
] as const;

interface FoldToFireRow {
  stepCount: number;
  allByContacts: string[];
  activeByContacts: string[];
  totalWalks: string;
  activeWalks: string;
  closingFraction: string;
}

interface FoldToFireSource {
  id: string;
  title: string;
  url: string;
  role: string;
}

export interface FoldToFireProfile {
  schema: "zerone.constructive-intelligence-fold-to-fire/v0";
  title: string;
  snapshotDate: string;
  sourceBindings: Array<{
    id: string;
    rawSha256: string;
    canonicalSha256: string;
  }>;
  releaseBoundary: Record<(typeof RELEASE_BOUNDARY_KEYS)[number], false>;
  modelBoundary: {
    walkConvention: string;
    contactDefinition: string;
    activeDefinition: string;
    proteinInterpretation: string;
  };
  enumeration: {
    solver: string;
    maximumExactSteps: number;
    evidenceRole: string;
    rows: FoldToFireRow[];
  };
  frontierProblem: {
    status: string;
    statement: string;
    exponent: string;
    evidenceStatus: string;
    computationDoesNotProve: true;
    weightedBridgeDoesNotEstablish: true;
  };
  weightedBridge: {
    status: string;
    partitionFunction: string;
    activePartitionFunction: string;
    effectiveFlux: string;
    interpretation: string;
    qNotEqualOneExponentTransfer: string;
    noveltyAuditRequired: true;
    notClaimedAsEstablishedOpenProblem: true;
  };
  nonImplicationWalls: Array<{
    id: string;
    premise: string;
    doesNotEstablish: string[];
  }>;
  sources: FoldToFireSource[];
}

export interface FoldToFireFetchOptions {
  fetcher?: typeof fetch;
  timeoutMs?: number;
  baseUrl?: string;
}

export class FoldToFireDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "FoldToFireDataError";
  }
}

function fail(path: string, message: string): never {
  throw new FoldToFireDataError(`${path}: ${message}`);
}

function object(value: unknown, path: string): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  return value as Record<string, unknown>;
}

function exactKeys(
  value: Record<string, unknown>,
  expected: readonly string[],
  path: string,
): void {
  const actual = Object.keys(value);
  if (
    actual.length !== expected.length ||
    actual.some((key) => !expected.includes(key))
  ) {
    fail(path, "contains missing or unknown fields");
  }
}

function literal<T extends string | number | boolean | null>(
  value: unknown,
  expected: T,
  path: string,
): T {
  if (value !== expected) fail(path, `must equal ${String(expected)}`);
  return expected;
}

function string(value: unknown, path: string): string {
  if (typeof value !== "string" || value.length === 0 || value.length > 4096) {
    fail(path, "must be a bounded nonempty string");
  }
  return value;
}

function stringArray(value: unknown, path: string): string[] {
  if (!Array.isArray(value) || value.length === 0 || value.length > 32) {
    fail(path, "must be a bounded nonempty array");
  }
  return value.map((item, index) => string(item, `${path}[${index}]`));
}

function exactJson(value: unknown, expected: unknown, path: string): void {
  if (JSON.stringify(value) !== JSON.stringify(expected)) {
    fail(path, "does not match the reviewed v0 contract");
  }
}

function parseRow(value: unknown, index: number): FoldToFireRow {
  const path = `$.enumeration.rows[${index}]`;
  const row = object(value, path);
  exactKeys(
    row,
    [
      "stepCount",
      "allByContacts",
      "activeByContacts",
      "totalWalks",
      "activeWalks",
      "closingFraction",
    ],
    path,
  );
  exactJson(
    [
      row.stepCount,
      row.allByContacts,
      row.activeByContacts,
      row.totalWalks,
      row.activeWalks,
      row.closingFraction,
    ],
    EXPECTED_ROWS[index],
    path,
  );
  return {
    stepCount: row.stepCount as number,
    allByContacts: stringArray(row.allByContacts, `${path}.allByContacts`),
    activeByContacts: stringArray(
      row.activeByContacts,
      `${path}.activeByContacts`,
    ),
    totalWalks: string(row.totalWalks, `${path}.totalWalks`),
    activeWalks: string(row.activeWalks, `${path}.activeWalks`),
    closingFraction: string(row.closingFraction, `${path}.closingFraction`),
  };
}

export function parseFoldToFire(value: unknown): FoldToFireProfile {
  const profile = object(value, "$");
  exactKeys(profile, TOP_LEVEL_KEYS, "$");
  literal(
    profile.schema,
    "zerone.constructive-intelligence-fold-to-fire/v0",
    "$.schema",
  );
  literal(profile.status, "SEALED_STATIC_PROFILE", "$.status");
  literal(profile.mode, "READ_ONLY_ZERO_EFFECT", "$.mode");
  literal(profile.authoritative, false, "$.authoritative");
  literal(profile.networkObserved, false, "$.networkObserved");

  const bindings = profile.sourceBindings;
  if (!Array.isArray(bindings) || bindings.length !== 3) {
    fail("$.sourceBindings", "must contain three immutable bindings");
  }
  const parsedBindings = bindings.map((value, index) => {
    const binding = object(value, `$.sourceBindings[${index}]`);
    exactKeys(
      binding,
      ["id", "path", "schema", "rawSha256", "canonicalSha256", "boundary"],
      `$.sourceBindings[${index}]`,
    );
    exactJson(
      [binding.id, binding.rawSha256, binding.canonicalSha256],
      EXPECTED_BINDING_DIGESTS[index],
      `$.sourceBindings[${index}]`,
    );
    return {
      id: string(binding.id, `$.sourceBindings[${index}].id`),
      rawSha256: string(
        binding.rawSha256,
        `$.sourceBindings[${index}].rawSha256`,
      ),
      canonicalSha256: string(
        binding.canonicalSha256,
        `$.sourceBindings[${index}].canonicalSha256`,
      ),
    };
  });

  const release = object(profile.releaseBoundary, "$.releaseBoundary");
  exactKeys(release, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) {
    literal(release[key], false, `$.releaseBoundary.${key}`);
  }
  const economics = object(profile.economics, "$.economics");
  exactKeys(
    economics,
    ["effect", "amount", "denom", "rewardMultiplier", "escrowReference"],
    "$.economics",
  );
  exactJson(
    economics,
    {
      effect: "NONE",
      amount: "0",
      denom: null,
      rewardMultiplier: false,
      escrowReference: null,
    },
    "$.economics",
  );

  const model = object(profile.modelBoundary, "$.modelBoundary");
  exactKeys(
    model,
    [
      "lattice",
      "walkConvention",
      "contactDefinition",
      "activeDefinition",
      "rotationTreatment",
      "reflectionTreatment",
      "minimumActiveSteps",
      "proteinInterpretation",
    ],
    "$.modelBoundary",
  );
  literal(model.lattice, "Z2_SQUARE_NEAREST_NEIGHBOUR", "$.modelBoundary.lattice");
  literal(model.minimumActiveSteps, 3, "$.modelBoundary.minimumActiveSteps");
  literal(
    model.rotationTreatment,
    "ROTATIONS_QUOTIENTED_BY_FIXING_FIRST_STEP_EAST",
    "$.modelBoundary.rotationTreatment",
  );
  literal(
    model.reflectionTreatment,
    "REFLECTIONS_REMAIN_DISTINCT",
    "$.modelBoundary.reflectionTreatment",
  );
  literal(
    model.proteinInterpretation,
    "ABSTRACT_POLYMER_GEOMETRY_ANALOGY_NOT_ATOMIC_PROTEIN_MODEL",
    "$.modelBoundary.proteinInterpretation",
  );

  const enumeration = object(profile.enumeration, "$.enumeration");
  exactKeys(
    enumeration,
    ["solver", "maximumExactSteps", "evidenceRole", "rows"],
    "$.enumeration",
  );
  literal(enumeration.solver, "fold-to-fire-exact-dfs/v0", "$.enumeration.solver");
  literal(enumeration.maximumExactSteps, 15, "$.enumeration.maximumExactSteps");
  literal(
    enumeration.evidenceRole,
    "EXACT_FINITE_ENUMERATION_NOT_ASYMPTOTIC_PROOF",
    "$.enumeration.evidenceRole",
  );
  if (!Array.isArray(enumeration.rows) || enumeration.rows.length !== 7) {
    fail("$.enumeration.rows", "must contain seven exact rows");
  }
  const rows = enumeration.rows.map(parseRow);

  const frontier = object(profile.frontierProblem, "$.frontierProblem");
  exactKeys(
    frontier,
    [
      "status",
      "domain",
      "statement",
      "exponent",
      "quantifier",
      "interpretation",
      "evidenceStatus",
      "computationDoesNotProve",
      "weightedBridgeDoesNotEstablish",
    ],
    "$.frontierProblem",
  );
  literal(frontier.status, "ESTABLISHED_OPEN_CONJECTURE", "$.frontierProblem.status");
  literal(frontier.exponent, "59/32", "$.frontierProblem.exponent");
  literal(frontier.evidenceStatus, "LITERATURE_CONJECTURE_NOT_PROVED", "$.frontierProblem.evidenceStatus");
  literal(frontier.computationDoesNotProve, true, "$.frontierProblem.computationDoesNotProve");
  literal(frontier.weightedBridgeDoesNotEstablish, true, "$.frontierProblem.weightedBridgeDoesNotEstablish");

  const weighted = object(profile.weightedBridge, "$.weightedBridge");
  exactKeys(
    weighted,
    [
      "status",
      "domain",
      "qDomain",
      "partitionFunction",
      "activePartitionFunction",
      "effectiveFlux",
      "interpretation",
      "qNotEqualOneExponentTransfer",
      "noveltyAuditRequired",
      "notClaimedAsEstablishedOpenProblem",
    ],
    "$.weightedBridge",
  );
  literal(weighted.status, "BESPOKE_RESEARCH_BRIDGE", "$.weightedBridge.status");
  literal(weighted.qDomain, "q>0", "$.weightedBridge.qDomain");
  literal(weighted.qNotEqualOneExponentTransfer, "OUT_OF_SCOPE_NOT_CLAIMED", "$.weightedBridge.qNotEqualOneExponentTransfer");
  literal(weighted.noveltyAuditRequired, true, "$.weightedBridge.noveltyAuditRequired");
  literal(weighted.notClaimedAsEstablishedOpenProblem, true, "$.weightedBridge.notClaimedAsEstablishedOpenProblem");

  if (!Array.isArray(profile.nonImplicationWalls) || profile.nonImplicationWalls.length !== 7) {
    fail("$.nonImplicationWalls", "must contain seven boundary walls");
  }
  const walls = profile.nonImplicationWalls.map((value, index) => {
    const wall = object(value, `$.nonImplicationWalls[${index}]`);
    exactKeys(wall, ["id", "premise", "doesNotEstablish"], `$.nonImplicationWalls[${index}]`);
    return {
      id: string(wall.id, `$.nonImplicationWalls[${index}].id`),
      premise: string(wall.premise, `$.nonImplicationWalls[${index}].premise`),
      doesNotEstablish: stringArray(wall.doesNotEstablish, `$.nonImplicationWalls[${index}].doesNotEstablish`),
    };
  });
  const wallIds = walls.map(({ id }) => id);
  exactJson(
    wallIds,
    [
      "FINITE_ENUMERATION_NOT_ASYMPTOTIC_PROOF",
      "UNWEIGHTED_EXPONENT_NOT_WEIGHTED_EXPONENT",
      "LATTICE_POLYMER_NOT_ATOMIC_PROTEIN",
      "ACTIVE_CONTACT_NOT_ENZYME_CATALYSIS",
      "GEOMETRY_FACTOR_NOT_CHEMICAL_RATE",
      "FOLDING_NOT_FOLDING_CATALYST",
      "SCIENCE_NOT_KARMA_AUTHORITY",
    ],
    "$.nonImplicationWalls",
  );

  if (!Array.isArray(profile.sources) || profile.sources.length !== 9) {
    fail("$.sources", "must contain nine reviewed locators");
  }
  const sources = profile.sources.map((value, index) => {
    const source = object(value, `$.sources[${index}]`);
    const expectedUrl = EXPECTED_SOURCE_URLS[index];
    if (expectedUrl === undefined) fail(`$.sources[${index}]`, "has no reviewed URL");
    exactKeys(source, ["id", "title", "url", "role"], `$.sources[${index}]`);
    literal(source.url, expectedUrl, `$.sources[${index}].url`);
    return {
      id: string(source.id, `$.sources[${index}].id`),
      title: string(source.title, `$.sources[${index}].title`),
      url: expectedUrl,
      role: string(source.role, `$.sources[${index}].role`),
    };
  });

  return {
    schema: "zerone.constructive-intelligence-fold-to-fire/v0",
    title: string(profile.title, "$.title"),
    snapshotDate: literal(profile.snapshotDate, "2026-08-13", "$.snapshotDate"),
    sourceBindings: parsedBindings,
    releaseBoundary: release as FoldToFireProfile["releaseBoundary"],
    modelBoundary: {
      walkConvention: string(model.walkConvention, "$.modelBoundary.walkConvention"),
      contactDefinition: string(model.contactDefinition, "$.modelBoundary.contactDefinition"),
      activeDefinition: string(model.activeDefinition, "$.modelBoundary.activeDefinition"),
      proteinInterpretation: "ABSTRACT_POLYMER_GEOMETRY_ANALOGY_NOT_ATOMIC_PROTEIN_MODEL",
    },
    enumeration: {
      solver: "fold-to-fire-exact-dfs/v0",
      maximumExactSteps: 15,
      evidenceRole: "EXACT_FINITE_ENUMERATION_NOT_ASYMPTOTIC_PROOF",
      rows,
    },
    frontierProblem: {
      status: "ESTABLISHED_OPEN_CONJECTURE",
      statement: string(frontier.statement, "$.frontierProblem.statement"),
      exponent: "59/32",
      evidenceStatus: "LITERATURE_CONJECTURE_NOT_PROVED",
      computationDoesNotProve: true,
      weightedBridgeDoesNotEstablish: true,
    },
    weightedBridge: {
      status: "BESPOKE_RESEARCH_BRIDGE",
      partitionFunction: string(weighted.partitionFunction, "$.weightedBridge.partitionFunction"),
      activePartitionFunction: string(weighted.activePartitionFunction, "$.weightedBridge.activePartitionFunction"),
      effectiveFlux: string(weighted.effectiveFlux, "$.weightedBridge.effectiveFlux"),
      interpretation: string(weighted.interpretation, "$.weightedBridge.interpretation"),
      qNotEqualOneExponentTransfer: "OUT_OF_SCOPE_NOT_CLAIMED",
      noveltyAuditRequired: true,
      notClaimedAsEstablishedOpenProblem: true,
    },
    nonImplicationWalls: walls,
    sources,
  };
}

export function parseFoldToFireJson(raw: string): FoldToFireProfile {
  if (new TextEncoder().encode(raw).byteLength > FOLD_TO_FIRE_MAX_BYTES) {
    fail("$", `document exceeds ${FOLD_TO_FIRE_MAX_BYTES} UTF-8 bytes`);
  }
  let value: unknown;
  try {
    value = JSON.parse(raw) as unknown;
  } catch {
    fail("$", "malformed JSON");
  }
  return parseFoldToFire(value);
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) throw new FoldToFireDataError("Fold-to-Fire returned an empty body");
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let length = 0;
  const abort = new Promise<never>((_resolve, reject) => {
    const refuse = (): void => reject(signal.reason ?? new DOMException("timed out", "TimeoutError"));
    signal.addEventListener("abort", refuse, { once: true });
    if (signal.aborted) refuse();
  });
  try {
    while (true) {
      const result = await Promise.race([reader.read(), abort]);
      if (result.done) break;
      length += result.value.byteLength;
      if (length > FOLD_TO_FIRE_MAX_BYTES) {
        void reader.cancel().catch(() => {
          // Refusal does not wait for a hostile stream cancellation.
        });
        throw new FoldToFireDataError("Fold-to-Fire exceeds its size limit");
      }
      chunks.push(result.value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => {
        // The deadline wins even if stream cancellation rejects.
      });
      throw new FoldToFireDataError("Fold-to-Fire request timed out");
    }
    throw error;
  } finally {
    try {
      reader.releaseLock();
    } catch {
      // A hostile pending stream is abandoned after refusal.
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
    throw new FoldToFireDataError("Fold-to-Fire digest verification is unavailable");
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", input);
  return [...new Uint8Array(digest)]
    .map((value) => value.toString(16).padStart(2, "0"))
    .join("");
}

function assertCanonicalResponseUrl(response: Response, baseUrl?: string): void {
  if (response.redirected) throw new FoldToFireDataError("Fold-to-Fire response was redirected");
  if (baseUrl === undefined) return;
  let expected: URL;
  let actual: URL;
  try {
    expected = new URL(FOLD_TO_FIRE_ENDPOINT, baseUrl);
    actual = new URL(response.url);
  } catch {
    throw new FoldToFireDataError("Fold-to-Fire returned an invalid final URL");
  }
  if (actual.href !== expected.href) {
    throw new FoldToFireDataError("Fold-to-Fire left its canonical same-origin path");
  }
}

export async function fetchFoldToFire(
  options: FoldToFireFetchOptions = {},
): Promise<FoldToFireProfile> {
  const fetcher = options.fetcher ?? fetch;
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(
    () => controller.abort(new DOMException("timed out", "TimeoutError")),
    options.timeoutMs ?? 8_000,
  );
  const baseUrl = options.baseUrl ??
    (typeof window === "undefined" ? undefined : window.location.href);
  const aborted = new Promise<never>((_resolve, reject) => {
    const refuse = (): void => reject(
      controller.signal.reason ?? new DOMException("timed out", "TimeoutError"),
    );
    controller.signal.addEventListener("abort", refuse, { once: true });
    if (controller.signal.aborted) refuse();
  });
  try {
    let response: Response;
    try {
      response = await Promise.race([
        fetcher(FOLD_TO_FIRE_ENDPOINT, {
          cache: "no-store",
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          redirect: "error",
          signal: controller.signal,
        }),
        aborted,
      ]);
    } catch (error) {
      if (controller.signal.aborted) throw new FoldToFireDataError("Fold-to-Fire request timed out");
      throw new FoldToFireDataError("Fold-to-Fire is unavailable");
    }
    if (!response.ok) throw new FoldToFireDataError(`Fold-to-Fire returned HTTP ${response.status}`);
    const contentType = response.headers.get("content-type");
    if (
      contentType === null ||
      !/^application\/json(?:\s*;\s*charset\s*=\s*(?:utf-8|"utf-8"))?\s*$/i.test(
        contentType,
      )
    ) {
      throw new FoldToFireDataError("Fold-to-Fire returned a non-JSON response");
    }
    const declaredLength = response.headers.get("content-length");
    if (declaredLength !== null && (!/^\d+$/.test(declaredLength) || Number(declaredLength) > FOLD_TO_FIRE_MAX_BYTES)) {
      throw new FoldToFireDataError("Fold-to-Fire exceeded its size limit");
    }
    assertCanonicalResponseUrl(response, baseUrl);
    const bytes = await readBoundedResponse(response, controller.signal);
    if ((await sha256Hex(bytes)) !== FOLD_TO_FIRE_SHA256) {
      throw new FoldToFireDataError("Fold-to-Fire did not match the reviewed digest");
    }
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new FoldToFireDataError("Fold-to-Fire was not valid UTF-8");
    }
    return parseFoldToFireJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  text?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className !== undefined) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function renderFoldToFire(root: HTMLElement, profile: FoldToFireProfile): void {
  const lab = element("div", "fold-to-fire-lab");
  const boundaries = element("div", "fold-to-fire-boundaries");
  for (const [label, value, description, kind] of [
    ["Finite baseline", "Exact through n = 15", "BigInt enumeration; not asymptotic proof.", ""],
    ["Open target", "59/32 remains conjectural", "The established q = 1 closing problem.", " is-open"],
    ["Weighted bridge", "Bespoke · audit required", "No q ≠ 1 exponent transfer is claimed.", " is-bespoke"],
    ["Effects", "Read only · zero", "No KARMA, reward, authority, wallet, or network write.", ""],
  ] as const) {
    const card = element("div", `fold-to-fire-boundary${kind}`);
    card.append(
      element("span", undefined, label),
      element("strong", undefined, value),
      element("p", undefined, description),
    );
    boundaries.append(card);
  }

  const modelGrid = element("div", "fold-to-fire-model-grid");
  const model = element("section", "fold-to-fire-card");
  model.append(
    element("span", "fold-to-fire-card-kicker", "Declared toy model"),
    element("h4", undefined, "Geometry first; chemistry remains separate."),
    element("p", undefined, "The walk is an interacting square-lattice homopolymer. Endpoint adjacency is active only by definition; it is not an enzyme active site."),
  );
  const equations = element("div", "fold-to-fire-equations");
  for (const equation of [
    profile.weightedBridge.partitionFunction,
    profile.weightedBridge.activePartitionFunction,
    profile.weightedBridge.effectiveFlux,
    profile.frontierProblem.statement,
  ]) equations.append(element("code", undefined, equation));
  model.append(equations);
  const conventions = element("ul", "fold-to-fire-convention-list");
  for (const item of [
    profile.modelBoundary.walkConvention,
    profile.modelBoundary.contactDefinition,
    profile.modelBoundary.activeDefinition,
    "J = κA/Z is a declared rapid-equilibrium, unit-occupancy simplification—not a physical flux derivation.",
  ]) conventions.append(element("li", undefined, item));
  model.append(conventions);

  const tablePanel = element("section", "fold-to-fire-table-panel");
  tablePanel.append(
    element("span", "fold-to-fire-card-kicker", profile.enumeration.evidenceRole),
    element("h4", undefined, "Exact fixed-first-step counts at q = 1"),
    element("p", undefined, "Rotations are fixed by the first East step; reflections remain distinct. Coefficients stay available in the sealed profile."),
  );
  const wrap = element("div", "fold-to-fire-table-wrap");
  const table = element("table", "fold-to-fire-table");
  const caption = element("caption", undefined, "Exact stored integers—not a fitted exponent");
  const head = element("thead");
  const headRow = element("tr");
  for (const heading of ["Steps n", "Z_n(1)", "A_n(1)", "A_n/Z_n"]) headRow.append(element("th", undefined, heading));
  for (const cell of headRow.children) cell.setAttribute("scope", "col");
  head.append(headRow);
  const body = element("tbody");
  for (const row of profile.enumeration.rows) {
    const tr = element("tr");
    const step = element("th", undefined, String(row.stepCount));
    step.setAttribute("scope", "row");
    tr.append(
      step,
      element("td", undefined, BigInt(row.totalWalks).toLocaleString("en-GB")),
      element("td", undefined, BigInt(row.activeWalks).toLocaleString("en-GB")),
      element("td", undefined, row.closingFraction),
    );
    body.append(tr);
  }
  table.append(caption, head, body);
  wrap.append(table);
  tablePanel.append(wrap);
  modelGrid.append(model, tablePanel);

  const challenge = element("section", "fold-to-fire-challenge");
  challenge.append(
    element("span", "fold-to-fire-card-kicker", "Family theorem hunt"),
    element("h4", undefined, "Build evidence without confusing it for proof."),
    element("p", undefined, "Here c_n counts unrestricted rooted n-step walks and p_m counts m-edge polygons up to translation; A_n(1)/Z_n(1) = 2(n+1)p_(n+1)/c_n."),
  );
  const ladder = element("ol");
  for (const item of [
    "Derive Z_3(q) = 7 + 2q and A_3(q) = 2q by hand.",
    "Reproduce the seven coefficient rows with a genuinely independent implementation.",
    "Prove a new finite identity, injection, monotonicity statement, or rigorous bound.",
    "Compare it against polygon-joining and endpoint-delocalization methods.",
    "Only then attempt progress toward the q = 1 exponent 59/32—or disprove the prediction.",
  ]) ladder.append(element("li", undefined, item));
  challenge.append(ladder);

  const boundaryGrid = element("div", "fold-to-fire-boundary-grid");
  for (const [title, copy, kind] of [
    ["Established open problem", "The q = 1 closing exponent is predicted, not proved. Seven exact rows cannot settle an asymptotic limit.", ""],
    ["Bespoke interpretive bridge", "Contact weighting has prior art. Fold-to-Fire and J = κA/Z are bounded framing; novelty audit remains required.", " is-bespoke"],
  ] as const) {
    const note = element("p", `fold-to-fire-boundary-note${kind}`);
    note.append(element("strong", undefined, `${title}. `), document.createTextNode(copy));
    boundaryGrid.append(note);
  }

  const sources = element("section", "fold-to-fire-sources");
  sources.append(
    element("span", "fold-to-fire-card-kicker", `${profile.sources.length} pinned source locators`),
    element("h4", undefined, "Math, folding, and foldases stay in their own lanes."),
  );
  const sourceList = element("ul");
  for (const source of profile.sources) {
    const item = element("li");
    const link = element("a", "fold-to-fire-source-link", source.title);
    link.href = source.url;
    link.target = "_blank";
    link.rel = "noreferrer";
    item.append(link, element("span", "fold-to-fire-source-meta", source.role));
    sourceList.append(item);
  }
  sources.append(sourceList);
  const facts = element("p", "fold-to-fire-facts");
  facts.append(
    element("strong", undefined, `Profile ${FOLD_TO_FIRE_SHA256.slice(0, 12)}…`),
    document.createTextNode(" · no sequence · no wet lab · no diagnosis · no reward · no KARMA event · no network write"),
  );
  sources.append(facts);

  lab.append(boundaries, modelGrid, challenge, boundaryGrid, sources);
  root.replaceChildren(lab);
  root.setAttribute("aria-busy", "false");
}

export async function initialiseFoldToFire(
  root: HTMLElement,
  options: FoldToFireFetchOptions = {},
): Promise<void> {
  const load = async (): Promise<void> => {
    root.setAttribute("aria-busy", "true");
    try {
      renderFoldToFire(root, await fetchFoldToFire(options));
      if (window.location.hash === "#fold-to-fire") {
        requestAnimationFrame(() => root.closest<HTMLElement>("#fold-to-fire")?.scrollIntoView({ block: "start", behavior: "instant" }));
      }
    } catch (error) {
      root.setAttribute("aria-busy", "false");
      const state = element("div", "fold-to-fire-load-error");
      state.setAttribute("role", "alert");
      state.append(
        element("strong", undefined, "The Fold-to-Fire challenge could not be loaded."),
        element("p", undefined, error instanceof Error ? error.message : "The response was unavailable or invalid."),
      );
      const actions = element("div", "fold-to-fire-load-actions");
      const retry = element("button", "button button-primary", "Try again");
      retry.type = "button";
      retry.addEventListener("click", () => void load());
      const raw = element("a", "button button-ghost", "Open raw profile");
      raw.href = FOLD_TO_FIRE_ENDPOINT;
      raw.target = "_blank";
      raw.rel = "noreferrer";
      actions.append(retry, raw);
      state.append(actions);
      root.replaceChildren(state);
    }
  };
  await load();
}
