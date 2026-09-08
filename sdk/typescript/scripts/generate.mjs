import { execFileSync } from "node:child_process";
import { createRequire } from "node:module";
import {
  readdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import telescopeModule from "@hyperweb/telescope";
import {
  generatedOutputDigest,
  generatedSourceDigest,
} from "./proto-digest.mjs";

const expectedBufVersion = "1.65.0";
const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(packageRoot, "../..");
const protoExport = resolve(packageRoot, ".proto-export");
const generated = resolve(packageRoot, "src/generated");
const telescope =
  typeof telescopeModule === "function" ? telescopeModule : telescopeModule.default;

function filesBelow(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) return filesBelow(path);
    return entry.isFile() ? [path] : [];
  });
}

function normalizeGeneratedTypeScript(directory) {
  for (const path of filesBelow(directory)) {
    if (!path.endsWith(".ts")) continue;
    const source = readFileSync(path, "utf8");
    const normalized = source
      .replace(/\r\n?/g, "\n")
      .replace(/[ \t]+$/gm, "");
    if (normalized !== source) {
      writeFileSync(path, normalized);
    }
  }
}

if (typeof telescope !== "function") {
  throw new TypeError("@hyperweb/telescope did not expose its generator");
}
if (dirname(protoExport) !== packageRoot || dirname(generated) !== resolve(packageRoot, "src")) {
  throw new Error("Refusing to replace an unexpected code-generation path");
}

const bufVersion = execFileSync("buf", ["--version"], {
  encoding: "utf8",
}).trim();
if (bufVersion !== expectedBufVersion) {
  throw new Error(
    `Expected buf ${expectedBufVersion}, received ${bufVersion || "no version"}`,
  );
}

rmSync(protoExport, { recursive: true, force: true });
rmSync(generated, { recursive: true, force: true });
execFileSync("buf", ["export", resolve(repoRoot, "proto"), "--output", protoExport], {
  stdio: "inherit",
});

// Telescope 2.2.4 / @cosmology/ast 2.2.0 derives a map's OUTER tag
// from its scalar VALUE type (e.g. knowledge Params tag140 becomes wire0).
// Protobuf maps are embedded entry messages and must always use wire2.
// Correct the pinned generator's map-only AST template, not generated files or
// node_modules on disk. Keep this in the source digest and fail closed if the
// dependency changes; SDK and committed-Go transport tests cover the wire.
const require = createRequire(import.meta.url);
if (require("@hyperweb/telescope/package.json").version !== "2.2.4" ||
    require("@cosmology/ast/package.json").version !== "2.2.0") {
  throw new Error("Re-review the map-wire correction for the new generator version");
}
const mapTemplates = require("@cosmology/ast/encoding/proto/encode/utils.js").types;
const originalMapTemplate = mapTemplates.keyHash;
if (typeof originalMapTemplate !== "function") throw new Error("Missing pinned map AST template");
mapTemplates.keyHash = (tag, ...args) => {
  if (!Number.isSafeInteger(tag) || tag < 8 || tag > 0xffffffff) {
    throw new Error(`Invalid generated map tag: ${tag}`);
  }
  return originalMapTemplate(Math.floor(tag / 8) * 8 + 2, ...args);
};

await telescope({
  protoDirs: [protoExport],
  outPath: generated,
  options: {
    useInterchainJs: false,
    useSDKTypes: false,
    interfaces: { enabled: false },
    prototypes: {
      enabled: true,
      includes: { protos: ["zerone/**/tx.proto"] },
      enableMessageComposer: true,
      enableRegistryLoader: true,
      methods: {
        encode: true,
        decode: true,
        fromJSON: false,
        toJSON: false,
        fromPartial: true,
        toSDK: false,
        fromSDK: false,
        fromSDKJSON: false,
        toAmino: false,
        fromAmino: false,
        toProto: false,
        fromProto: false,
      },
      typingsFormat: {
        num64: "bigint",
        useExact: false,
        useDeepPartial: true,
        useTelescopeGeneratedType: false,
      },
    },
    bundle: { enabled: false },
    tsDisable: { disableAll: true },
    aminoEncoding: {
      enabled: true,
      useLegacyInlineEncoding: true,
    },
    lcdClients: { enabled: false },
    rpcClients: { enabled: false },
    stargateClients: { enabled: false },
    helperFunctions: { enabled: false },
    mcpServer: { enabled: false },
  },
});

rmSync(resolve(generated, "google"), { recursive: true, force: true });
normalizeGeneratedTypeScript(generated);
writeFileSync(
  resolve(generated, "SOURCE_SHA256"),
  `${generatedSourceDigest(repoRoot, packageRoot)}\n`,
);
writeFileSync(
  resolve(generated, "GENERATED_SHA256"),
  `${generatedOutputDigest(packageRoot)}\n`,
);
rmSync(protoExport, { recursive: true, force: true });
