#!/usr/bin/env bun

import { createHash } from "node:crypto";
import {
  chmodSync,
  closeSync,
  copyFileSync,
  existsSync,
  fsyncSync,
  lstatSync,
  mkdirSync,
  mkdtempSync,
  openSync,
  readFileSync,
  readdirSync,
  renameSync,
  rmSync,
  statSync,
  utimesSync,
  writeFileSync,
} from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import {
  FRONTIER_INTAKE_BUN_VERSION,
  requireFrontierIntakeBunVersion,
} from "../src/runtime.ts";

const SCHEMA = "zerone.frontier-intake-macos-artifact.v1";
const ACL_PROTOCOL = "zerone-darwin-acl-v1";
const PACKAGE_DIRECTORY = "frontier-intake-macos";
const ARCHIVE_NAME = `${PACKAGE_DIRECTORY}.tar`;
const MANIFEST_NAME = `${PACKAGE_DIRECTORY}.manifest.json`;
const CHECKSUM_NAME = `${ARCHIVE_NAME}.sha256`;
const FIXED_MTIME = new Date("2000-01-01T00:00:00.000Z");
const FRONTIER_BUILD_RELATIVE = "tools/frontier-intake/build";
const BUNDLE_ARCHIVE_PATH = `${PACKAGE_DIRECTORY}/${FRONTIER_BUILD_RELATIVE}/frontier-intake.js`;
const HELPER_ARCHIVE_PATH = `${PACKAGE_DIRECTORY}/${FRONTIER_BUILD_RELATIVE}/darwin-acl-check`;

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const repositoryRoot = resolve(scriptDirectory, "../../..");
const frontierBuild = join(repositoryRoot, "tools/frontier-intake/build");
const releaseBuild = join(repositoryRoot, "build");
const bundleSource = join(frontierBuild, "frontier-intake.js");
const helperSource = join(frontierBuild, "darwin-acl-check");
const archivePath = join(releaseBuild, ARCHIVE_NAME);
const manifestPath = join(releaseBuild, MANIFEST_NAME);
const checksumPath = join(releaseBuild, CHECKSUM_NAME);

type ArtifactFile = {
  mode: "0444" | "0555";
  path: string;
  sha256: string;
  size: number;
};

type ArtifactManifest = {
  acl_helper_protocol: typeof ACL_PROTOCOL;
  archive_root: typeof PACKAGE_DIRECTORY;
  bun_version: string;
  files: [ArtifactFile, ArtifactFile, ArtifactFile];
  minimum_macos: "12.0";
  provenance_requirement: string;
  schema: typeof SCHEMA;
  source_commit: string;
};

function fail(message: string): never {
  throw new Error(`frontier-intake macOS artifact: ${message}`);
}

function command(
  argv: string[],
  options: { cwd?: string; stdin?: number } = {},
): string {
  const result = Bun.spawnSync({
    cmd: argv,
    cwd: options.cwd ?? repositoryRoot,
    env: {
      ...process.env,
      COPYFILE_DISABLE: "1",
      LANG: "C",
      LC_ALL: "C",
    },
    stdin: options.stdin,
    stdout: "pipe",
    stderr: "pipe",
  });
  if (result.exitCode !== 0) {
    const detail = result.stderr.toString().trim();
    fail(`${argv.join(" ")} failed${detail ? `: ${detail}` : ""}`);
  }
  return result.stdout.toString();
}

function sha256(path: string): string {
  return createHash("sha256").update(readFileSync(path)).digest("hex");
}

function regularFile(path: string, label: string): void {
  const metadata = lstatSync(path);
  if (!metadata.isFile() || metadata.isSymbolicLink()) {
    fail(`${label} must be a regular file`);
  }
  if (metadata.nlink !== 1) {
    fail(`${label} must have exactly one hard link`);
  }
}

function exactMode(path: string, expected: number, label: string): void {
  const actual = statSync(path).mode & 0o7777;
  if (actual !== expected) {
    fail(
      `${label} mode must be ${expected.toString(8).padStart(4, "0")}, got ${actual.toString(8).padStart(4, "0")}`,
    );
  }
}

function helperProtocol(helperPath: string, targetPath = helperPath): void {
  const descriptor = openSync(targetPath, "r");
  try {
    const result = Bun.spawnSync({
      cmd: [helperPath],
      env: { ...process.env, LANG: "C", LC_ALL: "C" },
      stdin: descriptor,
      stdout: "pipe",
      stderr: "pipe",
      timeout: 10_000,
    });
    if (
      result.exitCode !== 0 ||
      result.stdout.toString() !== `${ACL_PROTOCOL} clear\n` ||
      result.stderr.byteLength !== 0
    ) {
      fail("ACL helper did not return its exact clear self-inspection result");
    }
  } finally {
    closeSync(descriptor);
  }
}

function validateHelper(path: string, directory?: string): void {
  regularFile(path, "ACL helper");
  exactMode(path, 0o555, "ACL helper");
  command(["/usr/bin/lipo", path, "-verify_arch", "arm64", "x86_64"]);
  command(["/usr/bin/codesign", "--verify", "--strict", path]);
  const identifier = Bun.spawnSync({
    cmd: ["/usr/bin/codesign", "-d", "--verbose=4", path],
    env: { ...process.env, LANG: "C", LC_ALL: "C" },
    stdout: "pipe",
    stderr: "pipe",
  });
  if (
    identifier.exitCode !== 0 ||
    !identifier.stderr.toString().split("\n").includes(
      "Identifier=org.zerone.darwin-acl-check",
    ) ||
    !identifier.stderr.toString().includes("flags=0x2(adhoc)")
  ) {
    fail("ACL helper lacks the expected ad-hoc signature and identifier");
  }
  helperProtocol(path);
  if (directory) helperProtocol(path, directory);
}

function stableJson(value: unknown): string {
  function sort(input: unknown): unknown {
    if (Array.isArray(input)) return input.map(sort);
    if (input !== null && typeof input === "object") {
      return Object.fromEntries(
        Object.entries(input as Record<string, unknown>)
          .sort(([left], [right]) => (left < right ? -1 : left > right ? 1 : 0))
          .map(([key, child]) => [key, sort(child)]),
      );
    }
    return input;
  }
  return `${JSON.stringify(sort(value))}\n`;
}

function describeFile(
  path: string,
  archiveRelativePath: string,
  mode: ArtifactFile["mode"],
): ArtifactFile {
  const metadata = statSync(path);
  return {
    mode,
    path: archiveRelativePath,
    sha256: sha256(path),
    size: metadata.size,
  };
}

function registerSource(): { path: string; archivePath: string } {
  const directory = join(repositoryRoot, "docs/research");
  const names = readdirSync(directory)
    .filter((name) => /^math-breakthroughs-.*\.v.*\.json$/.test(name))
    .sort();
  const name = names.at(-1);
  if (!name) fail("canonical math breakthrough register is missing");
  return {
    path: join(directory, name),
    archivePath: `${PACKAGE_DIRECTORY}/docs/research/${name}`,
  };
}

function sourceCommit(): string {
  const value =
    process.env.SOURCE_COMMIT ??
    command(["/usr/bin/git", "rev-parse", "HEAD"]).trim();
  if (!/^[0-9a-f]{40}$/.test(value)) {
    fail("SOURCE_COMMIT must be an exact lowercase 40-hex Git commit");
  }
  return value;
}

function syncFile(path: string): void {
  const descriptor = openSync(path, "r");
  try {
    fsyncSync(descriptor);
  } finally {
    closeSync(descriptor);
  }
}

function publish(source: string, destination: string): void {
  syncFile(source);
  renameSync(source, destination);
}

function removeScratch(path: string): void {
  const packagePath = join(path, PACKAGE_DIRECTORY);
  for (const directory of [
    join(packagePath, "docs/research"),
    join(packagePath, "docs"),
    join(packagePath, FRONTIER_BUILD_RELATIVE),
    join(packagePath, "tools/frontier-intake"),
    join(packagePath, "tools"),
    packagePath,
  ]) {
    if (existsSync(directory)) chmodSync(directory, 0o755);
  }
  rmSync(path, { force: true, recursive: true });
}

function assertExactKeys(
  value: unknown,
  expected: string[],
  label: string,
): asserts value is Record<string, unknown> {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(`${label} must be an object`);
  }
  const actual = Object.keys(value as Record<string, unknown>).sort();
  const wanted = [...expected].sort();
  if (actual.length !== wanted.length || actual.some((key, i) => key !== wanted[i])) {
    fail(`${label} has unexpected fields`);
  }
}

function parseManifest(bytes: string): ArtifactManifest {
  let value: unknown;
  try {
    value = JSON.parse(bytes);
  } catch {
    fail("manifest is not valid JSON");
  }
  assertExactKeys(
    value,
    [
      "acl_helper_protocol",
      "archive_root",
      "bun_version",
      "files",
      "minimum_macos",
      "provenance_requirement",
      "schema",
      "source_commit",
    ],
    "manifest",
  );
  if (
    value.schema !== SCHEMA ||
    value.acl_helper_protocol !== ACL_PROTOCOL ||
    value.archive_root !== PACKAGE_DIRECTORY ||
    value.minimum_macos !== "12.0" ||
    value.provenance_requirement !==
      "verify an authorized distributor signature over the archive before execution" ||
    value.bun_version !== FRONTIER_INTAKE_BUN_VERSION ||
    typeof value.source_commit !== "string" ||
    !/^[0-9a-f]{40}$/.test(value.source_commit) ||
    !Array.isArray(value.files) ||
    value.files.length !== 3
  ) {
    fail("manifest contract is invalid");
  }
  const expectedPaths = [
    registerSource().archivePath,
    HELPER_ARCHIVE_PATH,
    BUNDLE_ARCHIVE_PATH,
  ];
  const expectedModes = ["0444", "0555", "0555"];
  value.files.forEach((entry, index) => {
    assertExactKeys(entry, ["mode", "path", "sha256", "size"], `files[${index}]`);
    if (
      entry.mode !== expectedModes[index] ||
      entry.path !== expectedPaths[index] ||
      typeof entry.sha256 !== "string" ||
      !/^[0-9a-f]{64}$/.test(entry.sha256) ||
      typeof entry.size !== "number" ||
      !Number.isSafeInteger(entry.size) ||
      entry.size <= 0
    ) {
      fail(`manifest files[${index}] contract is invalid`);
    }
  });
  return value as ArtifactManifest;
}

function verifyPublished(expectedCommit?: string): ArtifactManifest {
  regularFile(archivePath, "archive");
  regularFile(manifestPath, "detached manifest");
  regularFile(checksumPath, "checksum");

  const checksum = readFileSync(checksumPath, "utf8");
  const escapedArchiveName = ARCHIVE_NAME.replace(
    /[.*+?^${}()|[\]\\]/g,
    "\\$&",
  );
  const match = checksum.match(
    new RegExp(`^([0-9a-f]{64})  ${escapedArchiveName}\\n$`),
  );
  if (!match || match[1] !== sha256(archivePath)) {
    fail("archive checksum is malformed or does not match");
  }

  const register = registerSource();
  const expectedEntries = [
    `${PACKAGE_DIRECTORY}/`,
    `${PACKAGE_DIRECTORY}/MANIFEST.json`,
    `${PACKAGE_DIRECTORY}/docs/`,
    `${PACKAGE_DIRECTORY}/docs/research/`,
    register.archivePath,
    `${PACKAGE_DIRECTORY}/tools/`,
    `${PACKAGE_DIRECTORY}/tools/frontier-intake/`,
    `${PACKAGE_DIRECTORY}/${FRONTIER_BUILD_RELATIVE}/`,
    BUNDLE_ARCHIVE_PATH,
    HELPER_ARCHIVE_PATH,
  ];
  const entries = command(["/usr/bin/tar", "-tf", archivePath])
    .trimEnd()
    .split("\n");
  if (
    entries.length !== expectedEntries.length ||
    entries.some((entry, index) => entry !== expectedEntries[index])
  ) {
    fail("archive has an unexpected path, order, or entry count");
  }
  const expectedTypesAndModes = [
    "dr-xr-xr-x",
    "-r--r--r--",
    "dr-xr-xr-x",
    "dr-xr-xr-x",
    "-r--r--r--",
    "dr-xr-xr-x",
    "dr-xr-xr-x",
    "dr-xr-xr-x",
    "-r-xr-xr-x",
    "-r-xr-xr-x",
  ];
  const verboseEntries = command(["/usr/bin/tar", "-tvf", archivePath])
    .trimEnd()
    .split("\n");
  if (
    verboseEntries.length !== expectedEntries.length ||
    verboseEntries.some(
      (entry, index) =>
        entry.slice(0, 10) !== expectedTypesAndModes[index] ||
        !entry.endsWith(` ${expectedEntries[index]}`),
    )
  ) {
    fail("archive member type or mode is unsafe");
  }

  const scratch = mkdtempSync(join(releaseBuild, ".frontier-intake-verify."));
  try {
    command(["/usr/bin/tar", "-xf", archivePath, "-C", scratch]);
    const packagePath = join(scratch, PACKAGE_DIRECTORY);
    const internalManifestPath = join(packagePath, "MANIFEST.json");
    const internalBytes = readFileSync(internalManifestPath, "utf8");
    if (internalBytes !== readFileSync(manifestPath, "utf8")) {
      fail("detached manifest is not byte-identical to the archived manifest");
    }
    const manifest = parseManifest(internalBytes);
    if (stableJson(manifest) !== internalBytes) {
      fail("manifest is not in deterministic sorted-key encoding");
    }
    if (expectedCommit && manifest.source_commit !== expectedCommit) {
      fail("manifest source commit does not match the required commit");
    }
    exactMode(packagePath, 0o555, "archive root");
    exactMode(internalManifestPath, 0o444, "archived manifest");
    for (const entry of manifest.files) {
      const extractedPath = join(scratch, entry.path);
      regularFile(extractedPath, entry.path);
      exactMode(extractedPath, Number.parseInt(entry.mode, 8), entry.path);
      const metadata = statSync(extractedPath);
      if (metadata.size !== entry.size || sha256(extractedPath) !== entry.sha256) {
        fail(`${entry.path} does not match its manifest digest and size`);
      }
    }
    const helperPath = join(scratch, HELPER_ARCHIVE_PATH);
    validateHelper(helperPath, dirname(helperPath));
    return manifest;
  } finally {
    removeScratch(scratch);
  }
}

function build(): void {
  if (process.platform !== "darwin") {
    fail("building the universal helper package requires macOS");
  }
  mkdirSync(releaseBuild, { mode: 0o755, recursive: true });
  regularFile(bundleSource, "Bun bundle");
  regularFile(helperSource, "ACL helper");
  const register = registerSource();
  regularFile(register.path, "canonical register");
  validateHelper(helperSource, frontierBuild);

  const scratch = mkdtempSync(join(releaseBuild, ".frontier-intake-package."));
  try {
    const packagePath = join(scratch, PACKAGE_DIRECTORY);
    const stagedFrontierBuild = join(packagePath, FRONTIER_BUILD_RELATIVE);
    const stagedResearch = join(packagePath, "docs/research");
    mkdirSync(stagedFrontierBuild, { mode: 0o755, recursive: true });
    mkdirSync(stagedResearch, { mode: 0o755, recursive: true });
    const stagedBundle = join(stagedFrontierBuild, "frontier-intake.js");
    const stagedHelper = join(stagedFrontierBuild, "darwin-acl-check");
    const stagedRegister = join(stagedResearch, register.archivePath.split("/").at(-1)!);
    const stagedManifest = join(packagePath, "MANIFEST.json");
    copyFileSync(bundleSource, stagedBundle);
    copyFileSync(helperSource, stagedHelper);
    copyFileSync(register.path, stagedRegister);
    chmodSync(stagedBundle, 0o555);
    chmodSync(stagedHelper, 0o555);
    chmodSync(stagedRegister, 0o444);
    utimesSync(stagedBundle, FIXED_MTIME, FIXED_MTIME);
    utimesSync(stagedHelper, FIXED_MTIME, FIXED_MTIME);
    utimesSync(stagedRegister, FIXED_MTIME, FIXED_MTIME);

    const manifest: ArtifactManifest = {
      acl_helper_protocol: ACL_PROTOCOL,
      archive_root: PACKAGE_DIRECTORY,
      bun_version: Bun.version,
      files: [
        describeFile(stagedRegister, register.archivePath, "0444"),
        describeFile(
          stagedHelper,
          HELPER_ARCHIVE_PATH,
          "0555",
        ),
        describeFile(
          stagedBundle,
          BUNDLE_ARCHIVE_PATH,
          "0555",
        ),
      ],
      minimum_macos: "12.0",
      provenance_requirement:
        "verify an authorized distributor signature over the archive before execution",
      schema: SCHEMA,
      source_commit: sourceCommit(),
    };
    const manifestBytes = stableJson(manifest);
    writeFileSync(stagedManifest, manifestBytes, { mode: 0o444 });
    chmodSync(stagedManifest, 0o444);
    utimesSync(stagedManifest, FIXED_MTIME, FIXED_MTIME);
    const artifactDirectories = [
      packagePath,
      join(packagePath, "docs"),
      stagedResearch,
      join(packagePath, "tools"),
      join(packagePath, "tools/frontier-intake"),
      stagedFrontierBuild,
    ];
    for (const directory of artifactDirectories) {
      utimesSync(directory, FIXED_MTIME, FIXED_MTIME);
      chmodSync(directory, 0o555);
    }

    const stagedArchive = join(scratch, ARCHIVE_NAME);
    command([
      "/usr/bin/tar",
      "-c",
      "--format",
      "ustar",
      "--no-recursion",
      "--uid",
      "0",
      "--gid",
      "0",
      "--uname",
      "root",
      "--gname",
      "wheel",
      "--no-xattrs",
      "--no-acls",
      "--no-fflags",
      "--no-mac-metadata",
      "-f",
      stagedArchive,
      "-C",
      scratch,
      `${PACKAGE_DIRECTORY}/`,
      `${PACKAGE_DIRECTORY}/MANIFEST.json`,
      `${PACKAGE_DIRECTORY}/docs/`,
      `${PACKAGE_DIRECTORY}/docs/research/`,
      register.archivePath,
      `${PACKAGE_DIRECTORY}/tools/`,
      `${PACKAGE_DIRECTORY}/tools/frontier-intake/`,
      `${PACKAGE_DIRECTORY}/${FRONTIER_BUILD_RELATIVE}/`,
      BUNDLE_ARCHIVE_PATH,
      HELPER_ARCHIVE_PATH,
    ]);

    const stagedDetachedManifest = join(scratch, MANIFEST_NAME);
    const stagedChecksum = join(scratch, CHECKSUM_NAME);
    copyFileSync(stagedManifest, stagedDetachedManifest);
    chmodSync(stagedDetachedManifest, 0o444);
    writeFileSync(
      stagedChecksum,
      `${sha256(stagedArchive)}  ${ARCHIVE_NAME}\n`,
      { mode: 0o444 },
    );
    chmodSync(stagedChecksum, 0o444);

    // Sidecars land first. The archive rename is the publication boundary: a
    // concurrent reader can see either the old complete set, the new complete
    // set, or a checksum mismatch and must fail closed; it cannot accept mixed
    // payload bytes.
    publish(stagedDetachedManifest, manifestPath);
    publish(stagedChecksum, checksumPath);
    publish(stagedArchive, archivePath);
    const buildDescriptor = openSync(releaseBuild, "r");
    try {
      fsyncSync(buildDescriptor);
    } finally {
      closeSync(buildDescriptor);
    }
  } finally {
    removeScratch(scratch);
  }

  const manifest = verifyPublished(sourceCommit());
  console.log(
    `${archivePath}\nsha256 ${sha256(archivePath)}\nsource ${manifest.source_commit}`,
  );
}

function main(): void {
  requireFrontierIntakeBunVersion();
  const operation = process.argv[2] ?? "build";
  if (operation === "build") {
    build();
    return;
  }
  if (operation === "verify") {
    if (process.platform !== "darwin") {
      fail("verifying the universal helper package requires macOS");
    }
    const expectedCommit = process.argv[3];
    if (expectedCommit && !/^[0-9a-f]{40}$/.test(expectedCommit)) {
      fail("expected source commit must be lowercase 40-hex");
    }
    const manifest = verifyPublished(expectedCommit);
    console.log(
      `${archivePath}\nsha256 ${sha256(archivePath)}\nsource ${manifest.source_commit}`,
    );
    return;
  }
  fail("usage: macos-artifact.ts [build|verify [expected-source-commit]]");
}

if (import.meta.main) {
  try {
    main();
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exit(1);
  }
}
