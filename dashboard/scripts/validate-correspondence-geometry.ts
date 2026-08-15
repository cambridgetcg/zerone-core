import { createHash } from "node:crypto";
import { lstatSync, readFileSync, realpathSync } from "node:fs";
import { isAbsolute, relative, resolve } from "node:path";
import { pathToFileURL } from "node:url";

import {
  CORRESPONDENCE_GEOMETRY_SHA256,
  CorrespondenceGeometryDataError,
  parseCorrespondenceGeometryJson,
} from "../src/correspondence-geometry";

export interface CorrespondenceGeometryValidationSummary {
  manifestSha256: string;
  sourceBindingCount: number;
  physicsSourceCount: number;
  epistemicLaneCount: number;
  relationKindCount: number;
  dimensionCount: number;
  correspondenceCount: number;
  dualityCandidateCount: number;
  acceptedDualityCount: number;
  releaseEffectCount: number;
}

function digest(bytes: string | Uint8Array): string {
  return createHash("sha256").update(bytes).digest("hex");
}

function fail(message: string): never {
  throw new CorrespondenceGeometryDataError(`correspondence geometry: ${message}`);
}

function decodeUtf8(bytes: Uint8Array, label: string): string {
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    fail(`${label} is not valid UTF-8`);
  }
}

/**
 * Verify the sealed manifest and the exact bytes of every local source binding.
 * Physics URLs are provenance labels only and are deliberately never fetched.
 */
export function validateCorrespondenceGeometryRaw(
  raw: string,
  sourceBytes: ReadonlyMap<string, string | Uint8Array>,
): CorrespondenceGeometryValidationSummary {
  const manifestSha256 = digest(raw);
  if (manifestSha256 !== CORRESPONDENCE_GEOMETRY_SHA256) {
    fail(
      `document digest differs from the reviewed runtime pin (expected ${CORRESPONDENCE_GEOMETRY_SHA256}, got ${manifestSha256})`,
    );
  }

  const geometry = parseCorrespondenceGeometryJson(raw);
  for (const binding of geometry.sourceBindings) {
    const bytes = sourceBytes.get(binding.path);
    if (bytes === undefined) fail(`source set is missing ${binding.path}`);
    if (digest(bytes) !== binding.sha256) {
      fail(`source bytes drifted for ${binding.path}; SHA-256 mismatch`);
    }
  }
  if (sourceBytes.size !== geometry.sourceBindings.length) {
    fail("source set contains an unreviewed or duplicate path");
  }
  for (const path of sourceBytes.keys()) {
    if (!geometry.sourceBindings.some((binding) => binding.path === path)) {
      fail(`source set contains unreviewed path ${path}`);
    }
  }

  const dualityCandidateCount = geometry.correspondences.filter(
    (correspondence) => correspondence.relationKind === "DUALITY_CANDIDATE",
  ).length;
  const releaseEffectCount = Object.values(geometry.releaseBoundary).filter(
    (effect) => effect !== false,
  ).length;
  const summary: CorrespondenceGeometryValidationSummary = {
    manifestSha256,
    sourceBindingCount: geometry.sourceBindings.length,
    physicsSourceCount: geometry.physicsSources.length,
    epistemicLaneCount: geometry.epistemicLanes.length,
    relationKindCount: geometry.relationKinds.length,
    dimensionCount: geometry.dimensions.length,
    correspondenceCount: geometry.correspondences.length,
    dualityCandidateCount,
    acceptedDualityCount: geometry.dualityGate.acceptedCandidateCount,
    releaseEffectCount,
  };

  const expected = {
    sourceBindingCount: 7,
    physicsSourceCount: 5,
    epistemicLaneCount: 5,
    relationKindCount: 4,
    dimensionCount: 4,
    correspondenceCount: 7,
    dualityCandidateCount: 0,
    acceptedDualityCount: 0,
    releaseEffectCount: 0,
  } as const;
  for (const [key, value] of Object.entries(expected) as Array<
    [keyof typeof expected, number]
  >) {
    if (summary[key] !== value) fail(`reviewed v0 shape changed at ${key}`);
  }
  return summary;
}

function safeBoundSource(repositoryRoot: string, repositoryPath: string): string {
  let root: string;
  try {
    root = realpathSync(repositoryRoot);
  } catch {
    fail(`repository root is unavailable: ${repositoryRoot}`);
  }
  const candidate = resolve(root, repositoryPath);
  const lexicalRelative = relative(root, candidate);
  if (
    lexicalRelative === "" ||
    lexicalRelative === ".." ||
    lexicalRelative.startsWith(`..${process.platform === "win32" ? "\\" : "/"}`) ||
    isAbsolute(lexicalRelative)
  ) {
    fail(`source path escapes the repository: ${repositoryPath}`);
  }
  let stat;
  try {
    stat = lstatSync(candidate);
  } catch {
    fail(`source file is unavailable: ${repositoryPath}`);
  }
  if (stat.isSymbolicLink() || !stat.isFile()) {
    fail(`source must be a regular non-symlink file: ${repositoryPath}`);
  }
  const actual = realpathSync(candidate);
  const resolvedRelative = relative(root, actual);
  if (
    resolvedRelative === ".." ||
    resolvedRelative.startsWith(`..${process.platform === "win32" ? "\\" : "/"}`) ||
    isAbsolute(resolvedRelative)
  ) {
    fail(`resolved source path escapes the repository: ${repositoryPath}`);
  }
  return actual;
}

function runCli(): void {
  if (process.argv.length !== 4) {
    console.error(
      "usage: tsx scripts/validate-correspondence-geometry.ts CORRESPONDENCE_JSON REPOSITORY_ROOT",
    );
    process.exitCode = 2;
    return;
  }
  const artifactPath = process.argv[2];
  const repositoryRoot = process.argv[3];
  if (artifactPath === undefined || repositoryRoot === undefined) {
    process.exitCode = 2;
    return;
  }

  try {
    const raw = decodeUtf8(
      readFileSync(resolve(artifactPath)),
      "manifest",
    );
    const geometry = parseCorrespondenceGeometryJson(raw);
    const sourceBytes = new Map<string, Uint8Array>();
    for (const binding of geometry.sourceBindings) {
      sourceBytes.set(
        binding.path,
        readFileSync(safeBoundSource(repositoryRoot, binding.path)),
      );
    }
    const summary = validateCorrespondenceGeometryRaw(raw, sourceBytes);
    console.log(
      `correspondence geometry: PASS (${summary.epistemicLaneCount} lanes, ${summary.relationKindCount} relation kinds, ${summary.dimensionCount} dimensions, ${summary.correspondenceCount} mappings, ${summary.dualityCandidateCount} duality candidates, ${summary.acceptedDualityCount} equivalences, ${summary.releaseEffectCount} enabled effects, ${summary.sourceBindingCount} local source bindings, ${summary.physicsSourceCount} offline physics citations)`,
    );
  } catch (error) {
    console.error(
      `correspondence geometry: FAIL: ${error instanceof Error ? error.message : "unknown error"}`,
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
