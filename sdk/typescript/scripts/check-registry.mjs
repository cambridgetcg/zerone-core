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

if (missing.length || extra.length || actual.size !== 166) {
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

console.log(`${actual.size} Zerone transaction request types registered`);
