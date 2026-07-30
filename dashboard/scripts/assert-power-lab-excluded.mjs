import { existsSync, lstatSync, readdirSync, readFileSync } from "node:fs";
import { resolve, relative } from "node:path";

const LAB_SCHEMA = "zerone.local-power-reward-lab/v1";
const LAB_MASTHEAD = "LOCAL FIXTURE · NON-AUTHORITATIVE";
const distArgument = process.argv[2] ?? "dist";
const distRoot = resolve(process.cwd(), distArgument);

if (!existsSync(distRoot)) {
  throw new Error(`production build directory does not exist: ${distRoot}`);
}

const files = [];
const visit = (directory) => {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = resolve(directory, entry.name);
    const stat = lstatSync(path);
    if (stat.isSymbolicLink()) {
      throw new Error(
        `production build contains an unexpected symlink: ${relative(distRoot, path)}`,
      );
    }
    if (entry.isDirectory()) {
      visit(path);
      continue;
    }
    if (entry.isFile()) files.push(path);
  }
};
visit(distRoot);

const forbiddenNames = new Set(["power-lab.html", "shadow.generated.json"]);
const forbiddenMarkers = [LAB_SCHEMA, LAB_MASTHEAD].map((value) =>
  Buffer.from(value),
);

for (const path of files) {
  const name = relative(distRoot, path);
  if (forbiddenNames.has(name) || name.split("/").some((part) => forbiddenNames.has(part))) {
    throw new Error(`local power-lab artifact was published: ${name}`);
  }
  const contents = readFileSync(path);
  for (const marker of forbiddenMarkers) {
    if (contents.indexOf(marker) !== -1) {
      throw new Error(`local power-lab marker was published in ${name}`);
    }
  }
}

console.log(
  `Power-lab exclusion OK: scanned ${files.length} production build files under ${distRoot}`,
);
