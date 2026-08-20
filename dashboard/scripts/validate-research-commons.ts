import { createHash } from "node:crypto";
import {
  closeSync,
  constants,
  fstatSync,
  lstatSync,
  openSync,
  readSync,
  realpathSync,
} from "node:fs";
import type { Stats } from "node:fs";
import { isAbsolute, relative, resolve } from "node:path";
import { pathToFileURL } from "node:url";

import {
  RESEARCH_COMMONS_MAX_BYTES,
  RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID,
  RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH,
  RESEARCH_COMMONS_SHA256,
  ResearchCommonsDataError,
  parseResearchCommonsJson,
} from "../src/research-commons";

export const RESEARCH_COMMONS_LOCAL_SOURCE_MAX_BYTES = 262_144;

export interface ResearchCommonsValidationSummary {
  manifestSha256: string;
  planeCount: number;
  ledgerCount: number;
  externalRegisterCount: number;
  waterfallStepCount: number;
  outcomeLevelCount: number;
  pilotStepCount: number;
  relatedArtifactCount: number;
  sourceBindingCount: number;
  enabledActionCount: number;
  passedReleaseGateCount: number;
  activatedAmountUzrn: string;
  currentParticipantCount: number;
  automaticStaticGetCount: number;
  agentToolStatus: string;
  reciprocalProfileId: string;
}

function digest(bytes: string | Uint8Array): string {
  return createHash("sha256").update(bytes).digest("hex");
}

function fail(message: string): never {
  throw new ResearchCommonsDataError(`research commons: ${message}`);
}

function decodeUtf8(bytes: Uint8Array, label: string): string {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    fail(`${label} is not valid UTF-8`);
  }
}

type JsonValue = null | boolean | number | string | JsonValue[] | { [key: string]: JsonValue };

function jsonObject(value: unknown, label: string): Record<string, JsonValue> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(`${label} must be an object`);
  }
  return value as Record<string, JsonValue>;
}

function exactObjectKeys(
  value: Record<string, JsonValue>,
  expected: readonly string[],
  label: string,
): void {
  const actual = Object.keys(value);
  if (
    actual.length !== expected.length ||
    actual.some((key, index) => key !== expected[index])
  ) {
    fail(`${label} fields differ from the frozen reciprocal profile`);
  }
}

function compareCodePointKeys(left: string, right: string): number {
  const leftPoints = Array.from(left, (character) => character.codePointAt(0)!);
  const rightPoints = Array.from(right, (character) => character.codePointAt(0)!);
  const shared = Math.min(leftPoints.length, rightPoints.length);
  for (let index = 0; index < shared; index += 1) {
    const difference = leftPoints[index]! - rightPoints[index]!;
    if (difference !== 0) return difference;
  }
  return leftPoints.length - rightPoints.length;
}

function canonicalJson(value: JsonValue): string {
  if (value === null || typeof value === "boolean" || typeof value === "string") {
    return JSON.stringify(value);
  }
  if (typeof value === "number") {
    if (!Number.isFinite(value)) fail("reciprocal profile contains a non-finite number");
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) return `[${value.map(canonicalJson).join(",")}]`;
  const members = Object.keys(value)
    .sort(compareCodePointKeys)
    .map((key) => `${JSON.stringify(key)}:${canonicalJson(value[key]!)}`);
  return `{${members.join(",")}}`;
}

function literalJson(
  actual: JsonValue | undefined,
  expected: JsonValue,
  label: string,
): void {
  if (actual !== expected) fail(`${label} differs from the frozen reciprocal contract`);
}

function verifyReciprocalProfile(
  commons: ReturnType<typeof parseResearchCommonsJson>,
  boundBytes: ReadonlyMap<string, string | Uint8Array>,
): string {
  const source = boundBytes.get(RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH);
  if (source === undefined) fail("reciprocal profile source binding is missing");
  const raw = typeof source === "string" ? source : decodeUtf8(source, "reciprocal profile");
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw) as unknown;
  } catch {
    fail("reciprocal profile is not valid JSON");
  }
  const envelope = jsonObject(parsed, "reciprocal profile envelope");
  exactObjectKeys(envelope, ["profile", "profile_id"], "reciprocal profile envelope");
  const profile = jsonObject(envelope.profile, "reciprocal profile body");
  exactObjectKeys(
    profile,
    [
      "_format",
      "agenttool_formats",
      "authority_transfer",
      "canonicalization",
      "effects",
      "integration_ready",
      "integration_status",
      "original_static_interop",
      "pin_stage",
      "profile_id_algorithm",
      "six_ledger_boundary",
      "tree",
      "zerone_phase_a",
    ],
    "reciprocal profile body",
  );
  literalJson(
    profile._format,
    "agenttool.zerone-research-adapter-reciprocal/0.1",
    "reciprocal profile format",
  );
  literalJson(
    profile.canonicalization,
    "RECURSIVE_UNICODE_CODE_POINT_KEYS_COMPACT_JSON",
    "reciprocal canonicalization",
  );
  literalJson(
    profile.profile_id_algorithm,
    "SHA256_FORMAT_NUL_CANONICAL_JSON",
    "reciprocal profile ID algorithm",
  );
  literalJson(
    profile.pin_stage,
    "PHASE_B_AGENTTOOL_PINS_ZERONE_PHASE_A",
    "reciprocal pin stage",
  );
  literalJson(profile.authority_transfer, false, "reciprocal authority transfer");
  literalJson(profile.integration_ready, false, "reciprocal integration readiness");
  literalJson(
    profile.integration_status,
    "SHADOW_ONLY_NO_LIVE_INTEGRATION",
    "reciprocal integration status",
  );

  const formats = jsonObject(profile.agenttool_formats, "reciprocal AgentTool formats");
  exactObjectKeys(formats, ["public_projection", "settlement_bundle"], "AgentTool formats");
  literalJson(
    formats.public_projection,
    commons.agenttool_reference.public_projection_format,
    "reciprocal public projection format",
  );
  literalJson(
    formats.settlement_bundle,
    commons.agenttool_reference.settlement_bundle_format,
    "reciprocal settlement bundle format",
  );

  const effects = jsonObject(profile.effects, "reciprocal effect boundary");
  if (Object.keys(effects).length !== commons.agenttool_reference.agenttool_wire_false_effect_count) {
    fail("reciprocal effect boundary does not contain exactly 29 fields");
  }
  for (const [key, value] of Object.entries(effects)) {
    if (value !== false) fail(`reciprocal effect ${key} must remain false`);
  }

  const original = jsonObject(profile.original_static_interop, "original static interop pin");
  exactObjectKeys(original, ["format", "path", "raw_sha256"], "original static interop pin");
  literalJson(original.format, commons.agenttool_reference.interop_profile_id, "static interop format");
  literalJson(original.path, commons.agenttool_reference.interop_profile_path, "static interop path");
  literalJson(
    original.raw_sha256,
    commons.agenttool_reference.agenttool_interop_raw_sha256,
    "static interop digest",
  );

  const ledgers = jsonObject(profile.six_ledger_boundary, "reciprocal six-ledger pin");
  exactObjectKeys(ledgers, ["profile_digest", "profile_id"], "reciprocal six-ledger pin");
  literalJson(ledgers.profile_id, commons.shared_ledger_profile.id, "six-ledger profile ID");
  literalJson(ledgers.profile_digest, commons.shared_ledger_profile.sha256, "six-ledger digest");

  const tree = jsonObject(profile.tree, "reciprocal Tree pin");
  exactObjectKeys(
    tree,
    ["node_digest", "node_id", "raw_sha256", "schema"],
    "reciprocal Tree pin",
  );
  literalJson(tree.schema, commons.related_artifacts[1]!.schema, "reciprocal Tree schema");
  literalJson(
    tree.raw_sha256,
    `sha256:${commons.related_artifacts[1]!.sha256}`,
    "reciprocal Tree raw digest",
  );
  literalJson(tree.node_id, "math-proofcraft@1", "reciprocal Tree node ID");
  literalJson(
    tree.node_digest,
    "sha256:d8f364772611a214aaf5f671c630a5fa00daa3558330bfaf5e85efe7c5a1d0e2",
    "reciprocal Tree node digest",
  );

  const phaseA = jsonObject(profile.zerone_phase_a, "reciprocal Zerone Phase A pin");
  exactObjectKeys(
    phaseA,
    [
      "adapter_spec",
      "fixture_manifest",
      "main_merge_revision",
      "pull_request",
      "repository",
      "source_revision",
      "status",
    ],
    "reciprocal Zerone Phase A pin",
  );
  literalJson(
    phaseA.source_revision,
    commons.agenttool_reference.zerone_phase_a.source_revision,
    "reciprocal Zerone source revision",
  );
  literalJson(
    phaseA.main_merge_revision,
    commons.agenttool_reference.zerone_phase_a.main_merge_revision,
    "reciprocal Zerone main merge revision",
  );
  literalJson(
    phaseA.repository,
    commons.agenttool_reference.zerone_phase_a.repository,
    "reciprocal Zerone repository",
  );
  literalJson(
    phaseA.pull_request,
    commons.agenttool_reference.zerone_phase_a.pull_request,
    "reciprocal Zerone pull request",
  );
  literalJson(phaseA.status, "PHASE_A_STATIC_FIXTURE_ONLY", "reciprocal Zerone status");
  const phaseAAdapter = jsonObject(phaseA.adapter_spec, "reciprocal Zerone adapter pin");
  exactObjectKeys(
    phaseAAdapter,
    ["adapter_version", "path", "raw_sha256", "receipt_schema"],
    "reciprocal Zerone adapter pin",
  );
  literalJson(
    phaseAAdapter.adapter_version,
    "agenttool-research-receipt/v1",
    "reciprocal adapter version",
  );
  literalJson(
    phaseAAdapter.receipt_schema,
    commons.agenttool_reference.zerone_shadow_receipt_schema,
    "reciprocal receipt schema",
  );
  literalJson(
    phaseAAdapter.path,
    commons.agenttool_reference.zerone_phase_a.adapter_specification.path,
    "reciprocal adapter specification path",
  );
  literalJson(
    phaseAAdapter.raw_sha256,
    commons.agenttool_reference.zerone_phase_a.adapter_specification.raw_sha256,
    "reciprocal adapter specification digest",
  );
  const phaseAFixture = jsonObject(
    phaseA.fixture_manifest,
    "reciprocal Zerone fixture-manifest pin",
  );
  exactObjectKeys(
    phaseAFixture,
    ["format", "path", "raw_sha256"],
    "reciprocal Zerone fixture-manifest pin",
  );
  literalJson(
    phaseAFixture.format,
    "zerone.agenttool-research-fixture-set/0.1",
    "reciprocal fixture-manifest format",
  );
  literalJson(
    phaseAFixture.path,
    commons.agenttool_reference.zerone_phase_a.fixture_manifest.path,
    "reciprocal fixture-manifest path",
  );
  literalJson(
    phaseAFixture.raw_sha256,
    commons.agenttool_reference.zerone_phase_a.fixture_manifest.raw_sha256,
    "reciprocal fixture-manifest digest",
  );

  const format = profile._format as string;
  const reconstructed = `sha256:${digest(`${format}\u0000${canonicalJson(profile)}`)}`;
  literalJson(envelope.profile_id, reconstructed, "reciprocal reconstructed profile ID");
  if (reconstructed !== commons.agenttool_reference.reciprocal_cross_pin.profile_id) {
    fail("manifest and reconstructed reciprocal profile IDs differ");
  }
  if (
    reconstructed !== RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID ||
    reconstructed.slice("sha256:".length) !== commons.agenttool_reference.cross_pin_sha256
  ) {
    fail("reciprocal canonical tuple differs from the reviewed cross-pin digest");
  }
  return reconstructed;
}

function verifyManifestSeal(raw: string | Uint8Array): string {
  const manifestSha256 = digest(raw);
  if (manifestSha256 !== RESEARCH_COMMONS_SHA256) {
    fail(
      `document digest differs from the reviewed runtime pin (expected ${RESEARCH_COMMONS_SHA256}, got ${manifestSha256})`,
    );
  }
  return manifestSha256;
}

/**
 * Validate the exact RC-0.1 bytes and every local provenance binding. External
 * paper URLs remain inert locators and are deliberately never fetched.
 */
export function validateResearchCommonsRaw(
  raw: string,
  boundBytes: ReadonlyMap<string, string | Uint8Array>,
): ResearchCommonsValidationSummary {
  const manifestSha256 = verifyManifestSeal(raw);
  const commons = parseResearchCommonsJson(raw);
  const bindings = [
    ...commons.related_artifacts.map(({ path, sha256 }) => ({ path, sha256 })),
    ...commons.source_bindings.map(({ path, sha256 }) => ({ path, sha256 })),
  ];
  const expectedPaths = new Set(bindings.map(({ path }) => path));
  if (expectedPaths.size !== bindings.length) {
    fail("local provenance bindings contain a duplicate path");
  }
  for (const binding of bindings) {
    const bytes = boundBytes.get(binding.path);
    if (bytes === undefined) fail(`bound source set is missing ${binding.path}`);
    if (digest(bytes) !== binding.sha256) {
      fail(`bound source bytes drifted for ${binding.path}; SHA-256 mismatch`);
    }
  }
  if (boundBytes.size !== bindings.length) {
    fail("bound source set contains an unreviewed or duplicate path");
  }
  for (const path of boundBytes.keys()) {
    if (!expectedPaths.has(path)) fail(`bound source set contains unreviewed path ${path}`);
  }
  const reciprocalProfileId = verifyReciprocalProfile(commons, boundBytes);

  const summary: ResearchCommonsValidationSummary = {
    manifestSha256,
    planeCount: commons.architecture.planes.length,
    ledgerCount: commons.ledgers.length,
    externalRegisterCount: commons.non_ledger_registers.length,
    waterfallStepCount: commons.funding_model.waterfall.length,
    outcomeLevelCount: commons.outcome_schedule.levels.length,
    pilotStepCount: commons.pilot.steps.length,
    relatedArtifactCount: commons.related_artifacts.length,
    sourceBindingCount: commons.source_bindings.length,
    enabledActionCount: commons.calls_to_action.filter(({ enabled }) => enabled).length,
    passedReleaseGateCount: commons.release_gates.filter(({ passed }) => passed).length,
    activatedAmountUzrn: commons.funding_model.activated_amount_uzrn,
    currentParticipantCount: commons.agent_model.current_participants,
    automaticStaticGetCount: Number(commons.effect_boundary.automatic_static_gets),
    agentToolStatus: commons.agenttool_reference.status,
    reciprocalProfileId,
  };
  const expected = {
    planeCount: 4,
    ledgerCount: 6,
    externalRegisterCount: 5,
    waterfallStepCount: 8,
    outcomeLevelCount: 7,
    pilotStepCount: 4,
    relatedArtifactCount: 4,
    sourceBindingCount: 6,
    enabledActionCount: 0,
    passedReleaseGateCount: 1,
    activatedAmountUzrn: "0",
    currentParticipantCount: 0,
    automaticStaticGetCount: 1,
    agentToolStatus: "SHADOW_REFERENCE",
    reciprocalProfileId: RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID,
  } as const;
  for (const [key, value] of Object.entries(expected) as Array<
    [keyof typeof expected, number | string]
  >) {
    if (summary[key] !== value) fail(`reviewed RC-0.1 shape changed at ${key}`);
  }
  return summary;
}

function escapes(root: string, candidate: string): boolean {
  const relativePath = relative(root, candidate);
  return (
    relativePath === "" ||
    relativePath === ".." ||
    relativePath.startsWith(`..${process.platform === "win32" ? "\\" : "/"}`) ||
    isAbsolute(relativePath)
  );
}

function safeRepositoryRoot(repositoryRoot: string): string {
  const candidate = resolve(repositoryRoot);
  let stat;
  try {
    stat = lstatSync(candidate);
  } catch {
    fail(`repository root is unavailable: ${repositoryRoot}`);
  }
  if (stat.isSymbolicLink() || !stat.isDirectory()) {
    fail(
      `repository root must be a directory reached without a terminal symlink: ${repositoryRoot}`,
    );
  }
  try {
    return realpathSync(candidate);
  } catch {
    fail(`repository root is unavailable: ${repositoryRoot}`);
  }
}

function refuseSymlinkComponents(
  root: string,
  candidate: string,
  label: string,
): void {
  const relativePath = relative(root, candidate);
  let cursor = root;
  for (const component of relativePath.split(/[\\/]+/u)) {
    if (component.length === 0) continue;
    cursor = resolve(cursor, component);
    let stat;
    try {
      stat = lstatSync(cursor);
    } catch {
      fail(`${label} file is unavailable: ${candidate}`);
    }
    if (stat.isSymbolicLink()) {
      fail(`${label} path contains a symbolic link: ${candidate}`);
    }
  }
}

function sameFileMetadata(before: Stats, after: Stats): boolean {
  return (
    before.dev === after.dev &&
    before.ino === after.ino &&
    before.mode === after.mode &&
    before.size === after.size &&
    before.mtimeMs === after.mtimeMs &&
    before.ctimeMs === after.ctimeMs
  );
}

/**
 * Read through one no-follow, non-blocking descriptor and verify that both the
 * descriptor and pathname still identify the same unchanged regular file.
 * `afterOpenForTest` exists solely to make path-swap regression tests
 * deterministic; the CLI never supplies it.
 */
export function readBoundedRegularFile(
  root: string,
  candidatePath: string,
  repositoryRelative: boolean,
  label: string,
  maximumBytes: number,
  afterOpenForTest?: () => void,
): Uint8Array {
  if (
    !Number.isSafeInteger(maximumBytes) ||
    maximumBytes < 1 ||
    maximumBytes > RESEARCH_COMMONS_LOCAL_SOURCE_MAX_BYTES
  ) {
    fail(`${label} byte limit is invalid`);
  }
  const candidate = repositoryRelative
    ? resolve(root, candidatePath)
    : resolve(candidatePath);
  if (escapes(root, candidate)) {
    fail(`${label} path escapes the repository: ${candidatePath}`);
  }
  refuseSymlinkComponents(root, candidate, label);
  let descriptor: number;
  try {
    descriptor = openSync(
      candidate,
      constants.O_RDONLY | constants.O_NOFOLLOW | constants.O_NONBLOCK,
    );
  } catch {
    fail(`${label} must open as a no-follow regular file: ${candidatePath}`);
  }
  try {
    const before = fstatSync(descriptor);
    if (!before.isFile()) {
      fail(`${label} must be a regular non-symlink file: ${candidatePath}`);
    }
    if (!Number.isSafeInteger(before.size) || before.size > maximumBytes) {
      fail(`${label} exceeds its ${maximumBytes}-byte limit: ${candidatePath}`);
    }
    let actual: string;
    try {
      actual = realpathSync(candidate);
    } catch {
      fail(`${label} file became unavailable: ${candidatePath}`);
    }
    if (escapes(root, actual)) {
      fail(`resolved ${label} path escapes the repository: ${candidatePath}`);
    }

    afterOpenForTest?.();

    const bytes = Buffer.allocUnsafe(maximumBytes + 1);
    let length = 0;
    while (length <= maximumBytes) {
      const count = readSync(
        descriptor,
        bytes,
        length,
        Math.min(65_536, maximumBytes + 1 - length),
        null,
      );
      if (count === 0) break;
      length += count;
    }
    if (length > maximumBytes) {
      fail(`${label} exceeds its ${maximumBytes}-byte limit: ${candidatePath}`);
    }
    const after = fstatSync(descriptor);
    if (!sameFileMetadata(before, after) || length !== after.size) {
      fail(`${label} changed while it was being read: ${candidatePath}`);
    }

    let pathStat: Stats;
    let finalActual: string;
    try {
      pathStat = lstatSync(candidate);
      finalActual = realpathSync(candidate);
    } catch {
      fail(`${label} path changed while it was being read: ${candidatePath}`);
    }
    if (
      pathStat.isSymbolicLink() ||
      !pathStat.isFile() ||
      !sameFileMetadata(pathStat, after) ||
      escapes(root, finalActual)
    ) {
      fail(`${label} path changed while it was being read: ${candidatePath}`);
    }
    return bytes.subarray(0, length);
  } finally {
    closeSync(descriptor);
  }
}

function runCli(): void {
  if (process.argv.length !== 4) {
    console.error(
      "usage: tsx scripts/validate-research-commons.ts RESEARCH_COMMONS_JSON REPOSITORY_ROOT",
    );
    process.exitCode = 2;
    return;
  }
  const artifactPath = process.argv[2];
  const repositoryRootArgument = process.argv[3];
  if (artifactPath === undefined || repositoryRootArgument === undefined) {
    process.exitCode = 2;
    return;
  }

  try {
    const repositoryRoot = safeRepositoryRoot(repositoryRootArgument);
    const manifestBytes = readBoundedRegularFile(
      repositoryRoot,
      artifactPath,
      false,
      "manifest",
      RESEARCH_COMMONS_MAX_BYTES,
    );
    // Authenticate original bytes before decoding; TextDecoder strips a BOM.
    verifyManifestSeal(manifestBytes);
    const raw = decodeUtf8(manifestBytes, "manifest");
    const commons = parseResearchCommonsJson(raw);
    const boundBytes = new Map<string, Uint8Array>();
    for (const binding of [
      ...commons.related_artifacts,
      ...commons.source_bindings,
    ]) {
      boundBytes.set(
        binding.path,
        readBoundedRegularFile(
          repositoryRoot,
          binding.path,
          true,
          "source",
          RESEARCH_COMMONS_LOCAL_SOURCE_MAX_BYTES,
        ),
      );
    }
    const summary = validateResearchCommonsRaw(raw, boundBytes);
    console.log(
      `research commons: PASS (${summary.planeCount} planes, ${summary.ledgerCount} separate ledgers, ${summary.externalRegisterCount} external non-import registers, ${summary.waterfallStepCount} waterfall steps, ${summary.outcomeLevelCount} E-levels, ${summary.pilotStepCount} pilot steps, ${summary.enabledActionCount} enabled actions, ${summary.passedReleaseGateCount} passed gates, ${summary.activatedAmountUzrn} uzrn, ${summary.currentParticipantCount} participants, ${summary.automaticStaticGetCount} automatic static GET, AgentTool ${summary.agentToolStatus}, cross-pin ${summary.reciprocalProfileId}, ${summary.relatedArtifactCount} related artifacts, ${summary.sourceBindingCount} source bindings)`,
    );
  } catch (error) {
    console.error(
      `research commons: FAIL: ${error instanceof Error ? error.message : "unknown error"}`,
    );
    process.exitCode = 1;
  }
}

if (
  process.argv[1] !== undefined &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runCli();
}
