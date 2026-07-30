import { copyFileSync, rmSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { build } from "esbuild";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const outdir = resolve(packageRoot, "dist");

if (dirname(outdir) !== packageRoot) {
  throw new Error(`Refusing to clean unexpected output path: ${outdir}`);
}

rmSync(outdir, { recursive: true, force: true });

await build({
  absWorkingDir: packageRoot,
  entryPoints: {
    index: "src/index.ts",
    caip: "src/caip.ts",
    liquidity: "src/liquidity.ts",
    messages: "src/messages.ts",
    provenance: "src/provenance.ts",
    registry: "src/registry.ts",
  },
  outdir,
  bundle: true,
  splitting: true,
  format: "esm",
  platform: "neutral",
  target: "es2022",
  external: [
    "@cosmjs/encoding",
    "@cosmjs/proto-signing",
    "@noble/hashes/*",
  ],
  legalComments: "none",
  sourcemap: false,
});

for (const digest of ["SOURCE_SHA256", "GENERATED_SHA256"]) {
  copyFileSync(
    resolve(packageRoot, "src/generated", digest),
    resolve(outdir, digest),
  );
}
