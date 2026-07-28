import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import {
  generatedOutputDigest,
  generatedSourceDigest,
} from "./proto-digest.mjs";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(packageRoot, "../..");
const sourceDigestPath = resolve(packageRoot, "src/generated/SOURCE_SHA256");
const outputDigestPath = resolve(packageRoot, "src/generated/GENERATED_SHA256");
const expectedSource = readFileSync(sourceDigestPath, "utf8").trim();
const actualSource = generatedSourceDigest(repoRoot, packageRoot);

if (actualSource !== expectedSource) {
  throw new Error(
    `Generated codec inputs are stale: expected ${expectedSource}, got ${actualSource}. Run npm run generate.`,
  );
}

const expectedOutput = readFileSync(outputDigestPath, "utf8").trim();
const actualOutput = generatedOutputDigest(packageRoot);

if (actualOutput !== expectedOutput) {
  throw new Error(
    `Generated codec outputs were modified: expected ${expectedOutput}, got ${actualOutput}. Run npm run generate.`,
  );
}

console.log(
  `generated codecs are current (input ${actualSource.slice(0, 12)}, output ${actualOutput.slice(0, 12)})`,
);
