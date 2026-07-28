import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const declarationsRoot = resolve(packageRoot, "dist/types");
const javascriptExtension = /\.(?:[cm]?js|json|node)$/;
const relativeSpecifier =
  /(\b(?:from|import)\s*(?:\(\s*)?)(["'])(\.{1,2}\/[^"']+)\2/g;

function declarationFilesBelow(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) return declarationFilesBelow(path);
    return entry.isFile() && entry.name.endsWith(".d.ts") ? [path] : [];
  });
}

let rewrittenSpecifiers = 0;

for (const path of declarationFilesBelow(declarationsRoot)) {
  const declaration = readFileSync(path, "utf8");
  const rewritten = declaration.replace(
    relativeSpecifier,
    (match, prefix, quote, specifier) => {
      if (javascriptExtension.test(specifier)) return match;
      rewrittenSpecifiers += 1;
      return `${prefix}${quote}${specifier}.js${quote}`;
    },
  );

  if (rewritten !== declaration) {
    writeFileSync(path, rewritten);
  }
}

if (rewrittenSpecifiers === 0) {
  throw new Error("No relative declaration specifiers required rewriting");
}

console.log(
  `rewrote ${rewrittenSpecifiers} relative declaration specifiers for NodeNext`,
);
