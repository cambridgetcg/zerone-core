import { createHash } from "node:crypto";
import { readdirSync, readFileSync } from "node:fs";
import { relative, resolve, sep } from "node:path";

function filesBelow(directory, include) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) return filesBelow(path, include);
    return entry.isFile() && include(entry.name) ? [path] : [];
  });
}

function portableRelative(root, path) {
  return relative(root, path).split(sep).join("/");
}

function normalizedText(path) {
  return readFileSync(path, "utf8").replace(/\r\n?/g, "\n");
}

function digestFiles(root, paths) {
  const hash = createHash("sha256");
  const inputs = paths
    .map((path) => ({ path, relativePath: portableRelative(root, path) }))
    .sort((left, right) => {
      if (left.relativePath < right.relativePath) return -1;
      if (left.relativePath > right.relativePath) return 1;
      return 0;
    });

  for (const input of inputs) {
    hash.update(input.relativePath);
    hash.update("\0");
    hash.update(normalizedText(input.path));
    hash.update("\0");
  }

  return hash.digest("hex");
}

export function generatedSourceDigest(repoRoot, packageRoot) {
  const inputs = [
    ...filesBelow(resolve(repoRoot, "proto/zerone"), (name) =>
      name.endsWith(".proto"),
    ),
    resolve(repoRoot, "proto/buf.lock"),
    resolve(repoRoot, "proto/buf.yaml"),
    resolve(packageRoot, "package-lock.json"),
    resolve(packageRoot, "scripts/generate.mjs"),
    resolve(packageRoot, "scripts/proto-digest.mjs"),
  ];

  return digestFiles(repoRoot, inputs);
}

export function generatedOutputDigest(packageRoot) {
  const generatedRoot = resolve(packageRoot, "src/generated");
  const markers = new Set(["GENERATED_SHA256", "SOURCE_SHA256"]);
  const outputs = filesBelow(
    generatedRoot,
    (name) => !markers.has(name),
  );

  return digestFiles(generatedRoot, outputs);
}
