import { execFileSync } from "node:child_process";
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
