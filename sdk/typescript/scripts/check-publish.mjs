import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { execFileSync } from "node:child_process";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const temporaryRoot = mkdtempSync(join(tmpdir(), "zerone-sdk-publish-"));
const packDirectory = join(temporaryRoot, "pack");
const consumerDirectory = join(temporaryRoot, "consumer");
const consumerSourceDirectory = join(consumerDirectory, "src");
const packageJson = JSON.parse(
  readFileSync(join(packageRoot, "package.json"), "utf8"),
);
const typescriptCompiler = join(
  packageRoot,
  "node_modules",
  "typescript",
  "bin",
  "tsc",
);

const consumerSource = `
import * as root from "@zerone-chain/sdk";
import * as caip from "@zerone-chain/sdk/caip";
import * as cid from "@zerone-chain/sdk/cid";
import * as feegrant from "@zerone-chain/sdk/feegrant";
import * as messages from "@zerone-chain/sdk/messages";
import * as registry from "@zerone-chain/sdk/registry";

function assert(condition: unknown, message: string): asserts condition {
  if (!condition) {
    throw new Error(message);
  }
}

const network: caip.ZeroneNetwork = caip.defineZeroneNetwork("zerone-1");
const registerAccount: messages.auth.MsgRegisterAccount = {
  sender: "zrn1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a",
  did: \`did:zrn:\${"00".repeat(32)}\`,
  publicKey: "",
  accountType: "human",
  operationalKeyHash: "",
  metadata: "",
};
const encoded = messages.authMessages.encoded.registerAccount(registerAccount);
const registered = registry.createZeroneRegistry([]);
const memoryCid =
  "bafzbeigai3eoy2ccc7ybwjfz5r3rdxqrinwi4rwytly24tdbh6yk7zslrm";
const validatedMemoryCid = cid.asZeroneMemoryCid(memoryCid);
const feeGrant = feegrant.makeBoundedFeeGrant({
  network,
  granter: "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf",
  grantee: "zrn1qypqxpq9qcrsszg2pvxq6rs0zqg3yyc5s75sh2",
  spendLimit: [{ denom: "uzrn", amount: "100000" }],
  expiration: new Date("2099-01-01T00:00:00Z"),
  allowedMessageTypeUrls: ["/zerone.claiming_pot.v1.MsgClaim"],
});
const sponsoredFee = feegrant.makeSponsoredFee({
  network,
  granter: "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf",
  amount: [{ denom: "uzrn", amount: "2500" }],
  gas: "200000",
});

assert(root.cosmosChainId("zerone-1") === "cosmos:zerone-1", "root export failed");
assert(network.chainId === "cosmos:zerone-1", "caip export failed");
assert(validatedMemoryCid === memoryCid, "cid export failed");
assert(
  feeGrant.typeUrl === "/cosmos.feegrant.v1beta1.MsgGrantAllowance",
  "feegrant export failed",
);
assert(
  sponsoredFee.granter === "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf" &&
    sponsoredFee.gas === "200000",
  "sponsored fee export failed",
);
assert(
  encoded.typeUrl === "/zerone.auth.v1.MsgRegisterAccount" &&
    encoded.value instanceof Uint8Array,
  "messages export failed",
);
assert(registry.zeroneRegistryTypes.length > 0, "registry export is empty");
assert(
  registered.lookupType("/zerone.auth.v1.MsgRegisterAccount") !== undefined,
  "registry export failed",
);
assert(
  root.zeroneRegistryTypes.length === registry.zeroneRegistryTypes.length,
  "root registry re-export failed",
);
assert(
  root.ZERONE_ONBOARDING_MESSAGE_TYPE_URLS[0] ===
    feegrant.ZERONE_ONBOARDING_MESSAGE_TYPE_URLS[0],
  "root feegrant re-export failed",
);
`;

try {
  mkdirSync(packDirectory);
  mkdirSync(consumerSourceDirectory, { recursive: true });

  execFileSync(
    "npm",
    ["pack", "--pack-destination", packDirectory],
    {
      cwd: packageRoot,
      env: {
        ...process.env,
        npm_config_audit: "false",
        npm_config_fund: "false",
        npm_config_update_notifier: "false",
      },
      stdio: "inherit",
    },
  );

  const tarballs = readdirSync(packDirectory).filter((file) =>
    file.endsWith(".tgz"),
  );
  if (tarballs.length !== 1) {
    throw new Error(
      `Expected npm pack to create exactly one tarball, found ${tarballs.length}`,
    );
  }
  const tarball = join(packDirectory, tarballs[0]);

  writeFileSync(
    join(consumerDirectory, "package.json"),
    `${JSON.stringify(
      {
        name: "zerone-sdk-publish-consumer",
        version: "0.0.0",
        private: true,
        type: "module",
        engines: packageJson.engines,
      },
      null,
      2,
    )}\n`,
  );
  writeFileSync(
    join(consumerDirectory, "tsconfig.json"),
    `${JSON.stringify(
      {
        compilerOptions: {
          target: "ES2022",
          module: "NodeNext",
          moduleResolution: "NodeNext",
          strict: true,
          noUncheckedIndexedAccess: true,
          exactOptionalPropertyTypes: true,
          skipLibCheck: false,
          rootDir: "src",
          outDir: "dist",
        },
        include: ["src/**/*.ts"],
      },
      null,
      2,
    )}\n`,
  );
  writeFileSync(join(consumerSourceDirectory, "consumer.ts"), consumerSource);

  execFileSync(
    "npm",
    [
      "install",
      "--ignore-scripts",
      "--no-audit",
      "--no-fund",
      "--no-package-lock",
      "--save-exact",
      tarball,
    ],
    { cwd: consumerDirectory, stdio: "inherit" },
  );
  execFileSync(
    process.execPath,
    [typescriptCompiler, "--project", join(consumerDirectory, "tsconfig.json")],
    { cwd: consumerDirectory, stdio: "inherit" },
  );
  execFileSync(
    process.execPath,
    [join(consumerDirectory, "dist", "consumer.js")],
    { cwd: consumerDirectory, stdio: "inherit" },
  );

  console.log(
    `Validated ${packageJson.name}@${packageJson.version} packed exports in a strict NodeNext consumer.`,
  );
} finally {
  rmSync(temporaryRoot, { recursive: true, force: true });
}
