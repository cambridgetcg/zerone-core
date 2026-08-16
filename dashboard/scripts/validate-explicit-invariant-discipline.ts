import { createHash } from "node:crypto";
import { lstatSync, readFileSync, realpathSync } from "node:fs";
import { isAbsolute, relative, resolve } from "node:path";
import { pathToFileURL } from "node:url";

import {
  EXPLICIT_INVARIANT_DISCIPLINE_SHA256,
  ExplicitInvariantDisciplineDataError,
  parseExplicitInvariantDisciplineJson,
} from "../src/explicit-invariant-discipline";

export interface ExplicitInvariantDisciplineValidationSummary {
  manifestSha256: string;
  sourceBindingCount: number;
  primarySourceCount: number;
  integrationTargetCount: number;
  notImplementedIntegrationCount: number;
  recordCount: number;
  familyResultCount: number;
  noGoResultCount: number;
  conditionalUniquenessCount: number;
  constraintWitnessCount: number;
  falsifierCount: number;
  counterexampleCount: number;
  boundaryTermCount: number;
  proposedTransferCount: number;
  notRunLocalTestCount: number;
  nonTransferCount: number;
  releaseEffectCount: number;
}

function digest(bytes: string | Uint8Array): string {
  return createHash("sha256").update(bytes).digest("hex");
}

function fail(message: string): never {
  throw new ExplicitInvariantDisciplineDataError(
    `explicit invariant discipline: ${message}`,
  );
}

function decodeUtf8(bytes: Uint8Array, label: string): string {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    fail(`${label} is not valid UTF-8`);
  }
}

function verifyManifestSeal(raw: string | Uint8Array): string {
  const manifestSha256 = digest(raw);
  if (manifestSha256 !== EXPLICIT_INVARIANT_DISCIPLINE_SHA256) {
    fail(
      `document digest differs from the reviewed runtime pin (expected ${EXPLICIT_INVARIANT_DISCIPLINE_SHA256}, got ${manifestSha256})`,
    );
  }
  return manifestSha256;
}

/**
 * Verify the sealed manifest and the exact bytes of every reviewed local
 * source binding. Primary-source URLs are inert provenance locators and are
 * deliberately never fetched by this validator.
 */
export function validateExplicitInvariantDisciplineRaw(
  raw: string,
  sourceBytes: ReadonlyMap<string, string | Uint8Array>,
): ExplicitInvariantDisciplineValidationSummary {
  const manifestSha256 = verifyManifestSeal(raw);

  const discipline = parseExplicitInvariantDisciplineJson(raw);
  for (const binding of discipline.sourceBindings) {
    const bytes = sourceBytes.get(binding.path);
    if (bytes === undefined) fail(`source set is missing ${binding.path}`);
    if (digest(bytes) !== binding.rawSha256) {
      fail(`source bytes drifted for ${binding.path}; SHA-256 mismatch`);
    }
  }
  if (sourceBytes.size !== discipline.sourceBindings.length) {
    fail("source set contains an unreviewed or duplicate path");
  }
  const reviewedPaths = new Set(
    discipline.sourceBindings.map((binding) => binding.path),
  );
  for (const sourcePath of sourceBytes.keys()) {
    if (!reviewedPaths.has(sourcePath)) {
      fail(`source set contains unreviewed path ${sourcePath}`);
    }
  }

  const familyResultCount = discipline.records.filter(
    (record) => record.sourceResult.result.kind === "FAMILY",
  ).length;
  const noGoResultCount = discipline.records.filter(
    (record) => record.sourceResult.result.kind === "NO_GO",
  ).length;
  const conditionalUniquenessCount = discipline.records.filter(
    (record) => record.sourceResult.result.kind === "CONDITIONAL_UNIQUENESS",
  ).length;
  const constraintWitnessCount = discipline.records.reduce(
    (count, record) => count + record.sourceResult.constraintWitnesses.length,
    0,
  );
  const falsifierCount = discipline.records.reduce(
    (count, record) => count + record.sourceResult.falsifiers.length,
    0,
  );
  const counterexampleCount = discipline.records.reduce(
    (count, record) => count + record.sourceResult.counterexamples.length,
    0,
  );
  const boundaryTermCount = discipline.records.reduce(
    (count, record) => count + record.sourceResult.boundaryTerms.length,
    0,
  );
  const proposedTransferCount = discipline.records.filter(
    (record) => record.zeroneTransfer.assessment === "PROPOSED",
  ).length;
  const notRunLocalTestCount = discipline.records.filter(
    (record) => record.zeroneTransfer.localTest.status === "NOT_RUN",
  ).length;
  const nonTransferCount = discipline.records.reduce(
    (count, record) => count + record.zeroneTransfer.nonTransfers.length,
    0,
  );
  const notImplementedIntegrationCount = discipline.integrationTargets.filter(
    (target) => target.status === "NOT_IMPLEMENTED",
  ).length;
  const releaseEffectCount = Object.values(discipline.releaseBoundary).filter(
    (effect) => effect !== false,
  ).length;

  const summary: ExplicitInvariantDisciplineValidationSummary = {
    manifestSha256,
    sourceBindingCount: discipline.sourceBindings.length,
    primarySourceCount: discipline.primarySources.length,
    integrationTargetCount: discipline.integrationTargets.length,
    notImplementedIntegrationCount,
    recordCount: discipline.records.length,
    familyResultCount,
    noGoResultCount,
    conditionalUniquenessCount,
    constraintWitnessCount,
    falsifierCount,
    counterexampleCount,
    boundaryTermCount,
    proposedTransferCount,
    notRunLocalTestCount,
    nonTransferCount,
    releaseEffectCount,
  };

  const expected = {
    sourceBindingCount: 4,
    primarySourceCount: 4,
    integrationTargetCount: 5,
    notImplementedIntegrationCount: 1,
    recordCount: 4,
    familyResultCount: 2,
    noGoResultCount: 0,
    conditionalUniquenessCount: 2,
    constraintWitnessCount: 8,
    falsifierCount: 4,
    counterexampleCount: 4,
    boundaryTermCount: 4,
    proposedTransferCount: 4,
    notRunLocalTestCount: 4,
    nonTransferCount: 16,
    releaseEffectCount: 0,
  } as const;
  for (const [key, value] of Object.entries(expected) as Array<
    [keyof typeof expected, number]
  >) {
    if (summary[key] !== value) fail(`reviewed v1 shape changed at ${key}`);
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
    fail(`repository root must be a directory reached without a terminal symlink: ${repositoryRoot}`);
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

function safeRegularFile(
  root: string,
  candidatePath: string,
  repositoryRelative: boolean,
  label: string,
): string {
  const candidate = repositoryRelative
    ? resolve(root, candidatePath)
    : resolve(candidatePath);
  if (escapes(root, candidate)) {
    fail(`${label} path escapes the repository: ${candidatePath}`);
  }
  refuseSymlinkComponents(root, candidate, label);
  let stat;
  try {
    stat = lstatSync(candidate);
  } catch {
    fail(`${label} file is unavailable: ${candidatePath}`);
  }
  if (stat.isSymbolicLink() || !stat.isFile()) {
    fail(`${label} must be a regular non-symlink file: ${candidatePath}`);
  }
  let actual: string;
  try {
    actual = realpathSync(candidate);
  } catch {
    fail(`${label} file is unavailable: ${candidatePath}`);
  }
  if (escapes(root, actual)) {
    fail(`resolved ${label} path escapes the repository: ${candidatePath}`);
  }
  return actual;
}

function runCli(): void {
  if (process.argv.length !== 4) {
    console.error(
      "usage: tsx scripts/validate-explicit-invariant-discipline.ts EXPLICIT_INVARIANT_JSON REPOSITORY_ROOT",
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
    const manifestPath = safeRegularFile(
      repositoryRoot,
      artifactPath,
      false,
      "manifest",
    );
    const manifestBytes = readFileSync(manifestPath);
    // Authenticate the original file bytes before decoding or using any
    // manifest-provided source path. TextDecoder otherwise strips a UTF-8 BOM.
    verifyManifestSeal(manifestBytes);
    const raw = decodeUtf8(manifestBytes, "manifest");
    const discipline = parseExplicitInvariantDisciplineJson(raw);
    const sourceBytes = new Map<string, Uint8Array>();
    for (const binding of discipline.sourceBindings) {
      sourceBytes.set(
        binding.path,
        readFileSync(
          safeRegularFile(repositoryRoot, binding.path, true, "source"),
        ),
      );
    }
    const summary = validateExplicitInvariantDisciplineRaw(raw, sourceBytes);
    console.log(
      `explicit invariant discipline: PASS (${summary.recordCount} records: ${summary.familyResultCount} families, ${summary.noGoResultCount} no-go results, ${summary.conditionalUniquenessCount} conditional uniqueness results; ${summary.constraintWitnessCount} constraint witnesses, ${summary.falsifierCount} falsifiers, ${summary.counterexampleCount} counterexamples, ${summary.boundaryTermCount} boundary terms; ${summary.proposedTransferCount} proposed Zerone transfers, ${summary.notRunLocalTestCount} local tests not run, ${summary.nonTransferCount} non-transfer walls; ${summary.integrationTargetCount} integration targets including ${summary.notImplementedIntegrationCount} not implemented; ${summary.releaseEffectCount} enabled effects; ${summary.sourceBindingCount} local source bindings, ${summary.primarySourceCount} offline primary citations)`,
    );
  } catch (error) {
    console.error(
      `explicit invariant discipline: FAIL: ${error instanceof Error ? error.message : "unknown error"}`,
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
