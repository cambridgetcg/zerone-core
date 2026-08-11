import { createHash } from "node:crypto";
import {
  lstatSync,
  readFileSync,
  realpathSync,
} from "node:fs";
import {
  dirname,
  isAbsolute,
  relative,
  resolve,
} from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import {
  AUTHORITY_GEOMETRY_SHA256,
  AuthorityGeometryDataError,
  assertAuthorityGeometryTargetGate,
  assessAuthorityGeometry,
  parseAuthorityGeometryJson,
  type AuthorityGeometry,
  type AuthorityGeometryAssessment,
} from "../src/authority-geometry";

const DEFAULT_REPOSITORY_ROOT = resolve(
  dirname(fileURLToPath(import.meta.url)),
  "../..",
);

export interface AuthorityGeometryValidationSummary
  extends AuthorityGeometryAssessment {
  manifestSha256: string;
  sourceAnchorCount: number;
  requiredSnippetCount: number;
  forbiddenSnippetCount: number;
}

export interface AuthorityGeometryValidationOptions {
  repositoryRoot?: string;
  requireCanonicalDigest?: boolean;
  verifySourceAnchors?: boolean;
}

function digest(bytes: string | Uint8Array): string {
  return createHash("sha256").update(bytes).digest("hex");
}

function safeAnchoredFile(
  repositoryRoot: string,
  repositoryPath: string,
): string {
  let root: string;
  try {
    root = realpathSync(repositoryRoot);
  } catch {
    throw new AuthorityGeometryDataError(
      `source anchors: repository root is unavailable: ${repositoryRoot}`,
    );
  }
  const candidate = resolve(root, repositoryPath);
  const lexicalRelative = relative(root, candidate);
  if (
    lexicalRelative === "" ||
    lexicalRelative === ".." ||
    lexicalRelative.startsWith(`..${process.platform === "win32" ? "\\" : "/"}`) ||
    isAbsolute(lexicalRelative)
  ) {
    throw new AuthorityGeometryDataError(
      `source anchor ${repositoryPath}: path escapes the repository`,
    );
  }
  let stat;
  try {
    stat = lstatSync(candidate);
  } catch {
    throw new AuthorityGeometryDataError(
      `source anchor ${repositoryPath}: file is unavailable`,
    );
  }
  if (stat.isSymbolicLink() || !stat.isFile()) {
    throw new AuthorityGeometryDataError(
      `source anchor ${repositoryPath}: expected a regular non-symlink file`,
    );
  }
  const actual = realpathSync(candidate);
  const resolvedRelative = relative(root, actual);
  if (
    resolvedRelative === ".." ||
    resolvedRelative.startsWith(`..${process.platform === "win32" ? "\\" : "/"}`) ||
    isAbsolute(resolvedRelative)
  ) {
    throw new AuthorityGeometryDataError(
      `source anchor ${repositoryPath}: resolved path escapes the repository`,
    );
  }
  return actual;
}

export function validateAuthorityGeometrySourceAnchors(
  geometry: AuthorityGeometry,
  repositoryRoot = DEFAULT_REPOSITORY_ROOT,
): {
  sourceAnchorCount: number;
  requiredSnippetCount: number;
  forbiddenSnippetCount: number;
} {
  let requiredSnippetCount = 0;
  let forbiddenSnippetCount = 0;
  for (const anchor of geometry.sourceAnchors) {
    const filename = safeAnchoredFile(repositoryRoot, anchor.path);
    const bytes = readFileSync(filename);
    const actualDigest = digest(bytes);
    if (actualDigest !== anchor.sha256) {
      throw new AuthorityGeometryDataError(
        `source anchor ${anchor.id}: ${anchor.path} SHA-256 mismatch (expected ${anchor.sha256}, got ${actualDigest})`,
      );
    }
    const source = bytes.toString("utf8");
    if (!Buffer.from(source, "utf8").equals(bytes)) {
      throw new AuthorityGeometryDataError(
        `source anchor ${anchor.id}: ${anchor.path} is not valid UTF-8`,
      );
    }
    for (const snippet of anchor.requiredSnippets) {
      requiredSnippetCount += 1;
      if (!source.includes(snippet)) {
        throw new AuthorityGeometryDataError(
          `source anchor ${anchor.id}: missing required snippet ${JSON.stringify(snippet)}`,
        );
      }
    }
    for (const snippet of anchor.forbiddenSnippets) {
      forbiddenSnippetCount += 1;
      if (source.includes(snippet)) {
        throw new AuthorityGeometryDataError(
          `source anchor ${anchor.id}: found forbidden snippet ${JSON.stringify(snippet)}`,
        );
      }
    }
  }
  return {
    sourceAnchorCount: geometry.sourceAnchors.length,
    requiredSnippetCount,
    forbiddenSnippetCount,
  };
}

export function validateAuthorityGeometryRaw(
  raw: string,
  options: AuthorityGeometryValidationOptions = {},
): AuthorityGeometryValidationSummary {
  const manifestSha256 = digest(raw);
  if (
    options.requireCanonicalDigest !== false &&
    manifestSha256 !== AUTHORITY_GEOMETRY_SHA256
  ) {
    throw new AuthorityGeometryDataError(
      `manifest SHA-256 mismatch (expected ${AUTHORITY_GEOMETRY_SHA256}, got ${manifestSha256})`,
    );
  }
  const geometry = parseAuthorityGeometryJson(raw);
  const assessment = assessAuthorityGeometry(geometry);
  const anchors =
    options.verifySourceAnchors === false
      ? {
          sourceAnchorCount: geometry.sourceAnchors.length,
          requiredSnippetCount: geometry.sourceAnchors.reduce(
            (total, anchor) => total + anchor.requiredSnippets.length,
            0,
          ),
          forbiddenSnippetCount: geometry.sourceAnchors.reduce(
            (total, anchor) => total + anchor.forbiddenSnippets.length,
            0,
          ),
        }
      : validateAuthorityGeometrySourceAnchors(
          geometry,
          options.repositoryRoot ?? DEFAULT_REPOSITORY_ROOT,
        );
  return { manifestSha256, ...anchors, ...assessment };
}

interface CliOptions {
  artifactPath: string;
  targetGate: boolean;
}

function parseCliOptions(arguments_: readonly string[]): CliOptions {
  let targetGate = false;
  const positional: string[] = [];
  for (const argument of arguments_) {
    if (argument === "--target-gate") targetGate = true;
    else if (argument.startsWith("--")) {
      throw new AuthorityGeometryDataError(`unknown option: ${argument}`);
    } else positional.push(argument);
  }
  if (positional.length !== 1 || positional[0] === undefined) {
    throw new AuthorityGeometryDataError(
      "usage: validate-authority-geometry.ts [--target-gate] <authority-geometry.v1.json>",
    );
  }
  return { artifactPath: positional[0], targetGate };
}

function runCli(): void {
  try {
    const options = parseCliOptions(process.argv.slice(2));
    const artifactPath = resolve(options.artifactPath);
    const raw = readFileSync(artifactPath, "utf8");
    const summary = validateAuthorityGeometryRaw(raw);
    const geometry = parseAuthorityGeometryJson(raw);
    if (options.targetGate) assertAuthorityGeometryTargetGate(geometry);
    process.stdout.write(
      `authority geometry: REPORT PASS; activation ${summary.overall}; ` +
        `static ${summary.staticAuthoritySurfacesPassing}/${summary.staticAuthoritySurfacesTotal}; ` +
        `H4 ${summary.h4GatesEvidenced}/${summary.h4GatesTotal}; ` +
        `H5 ${summary.h5GatesEvidenced}/${summary.h5GatesTotal}; ` +
        `${summary.sourceAnchorCount} source anchors; sha256 ${summary.manifestSha256}\n`,
    );
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    process.stderr.write(`authority geometry: ${message}\n`);
    process.exitCode = 1;
  }
}

if (
  process.argv[1] !== undefined &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runCli();
}
