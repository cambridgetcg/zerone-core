import { readdirSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(packageRoot, "../..");
const protoRoot = resolve(repoRoot, "proto/zerone");
const generatedRoot = resolve(packageRoot, "src/generated/zerone");

const modules = readdirSync(protoRoot, { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
  .sort();

const expected = new Set();
const actual = new Set();

for (const moduleName of modules) {
  const txProto = resolve(protoRoot, moduleName, "v1/tx.proto");
  let proto;
  try {
    proto = readFileSync(txProto, "utf8");
  } catch {
    continue;
  }

  const packageName = proto.match(/^\s*package\s+([^;]+);/m)?.[1];
  if (!packageName) throw new Error(`Missing package declaration in ${txProto}`);

  for (const match of proto.matchAll(/^\s*rpc\s+\w+\s*\(\s*(Msg\w+)\s*\)/gm)) {
    expected.add(`/${packageName}.${match[1]}`);
  }

  const registryPath = resolve(generatedRoot, moduleName, "v1/tx.registry.ts");
  const registry = readFileSync(registryPath, "utf8");
  for (const match of registry.matchAll(/"\/(zerone\.[^"]+\.Msg[^"]+)"/g)) {
    actual.add(`/${match[1]}`);
  }
}

const missing = [...expected].filter((typeUrl) => !actual.has(typeUrl)).sort();
const extra = [...actual].filter((typeUrl) => !expected.has(typeUrl)).sort();

if (missing.length || extra.length || actual.size !== expected.size) {
  throw new Error(
    [
      `Registry mismatch: ${actual.size} generated, ${expected.size} expected`,
      missing.length ? `Missing: ${missing.join(", ")}` : "",
      extra.length ? `Extra: ${extra.join(", ")}` : "",
    ]
      .filter(Boolean)
      .join("\n"),
  );
}

function assertDocumentedCounts(paths, label, expectedCount, expression) {
  for (const path of paths) {
    const document = readFileSync(resolve(repoRoot, path), "utf8");
    const counts = [...document.matchAll(expression)].map((match) =>
      Number.parseInt(match[1], 10),
    );
    if (counts.length === 0) {
      throw new Error(`${path} does not declare its ${label} inventory`);
    }
    const mismatched = counts.filter((count) => count !== expectedCount);
    if (mismatched.length > 0) {
      throw new Error(
        `${path} declares ${label} count ${mismatched.join(", ")}; generated inventory is ${expectedCount}`,
      );
    }
  }
}

const requestInventoryDocuments = [
  "README.md",
  "CHANGELOG.md",
  "STATE.md",
  "docs/API.md",
  "docs/ROADMAP.md",
  "docs/standards/OPEN_CRYPTO_SDK.md",
];
assertDocumentedCounts(
  requestInventoryDocuments,
  "transaction request",
  actual.size,
  /\b(\d+)\b[ \t]*(?:\r?\n[ \t]*)?(?:protobuf[ \t]+Msg[ \t]+)?request(?:[ \t]+message)?(?:[ \t]+type)?s\b/g,
);

const swaggerPath = resolve(repoRoot, "docs/swagger-ui/swagger.json");
const swagger = JSON.parse(readFileSync(swaggerPath, "utf8"));
if (
  swagger === null ||
  typeof swagger !== "object" ||
  Array.isArray(swagger.paths) ||
  swagger.paths === null ||
  typeof swagger.paths !== "object" ||
  Array.isArray(swagger.definitions) ||
  swagger.definitions === null ||
  typeof swagger.definitions !== "object"
) {
  throw new Error(`${swaggerPath} does not contain Swagger 2 paths and definitions objects`);
}
const swaggerPathCount = Object.keys(swagger.paths).length;
const swaggerDefinitionCount = Object.keys(swagger.definitions).length;
const swaggerInventoryDocuments = [
  "README.md",
  "CHANGELOG.md",
  "docs/API.md",
  "docs/ROADMAP.md",
  "docs/standards/OPEN_CRYPTO_SDK.md",
];
assertDocumentedCounts(
  swaggerInventoryDocuments,
  "Swagger path",
  swaggerPathCount,
  /\b(\d+)\s+(?:REST\s+)?paths\b/g,
);
assertDocumentedCounts(
  swaggerInventoryDocuments,
  "Swagger definition",
  swaggerDefinitionCount,
  /\b(\d+)\s+(?:schema\s+)?definitions\b/g,
);

console.log(
  `${actual.size} Zerone transaction request types registered; Swagger inventory has ${swaggerPathCount} paths and ${swaggerDefinitionCount} definitions`,
);
