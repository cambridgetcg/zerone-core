import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
import {
  EPIGENETICS_GARDEN_SHA256,
  KARMA_FOUNDATION_SHA256,
  LifeGardenDataError,
  STATIC_STANDARD_MAX_BYTES,
  parseEpigeneticsCapabilityGardenJson,
  parseKarmaFoundationJson,
  type EpigeneticsCapabilityGarden,
  type KarmaFoundation,
} from "../src/life-garden";

const EXPECTED_GROUPED_PREREQUISITES = [
  {
    nodeId: "analysis-batch-confounding@1",
    allOf: ["integrity-statistics-reproducibility@1"],
    atLeast: {
      count: 1,
      of: [
        "assay-atac-cuttag@1",
        "assay-bisulfite-methylation@1",
        "assay-rna-seq@1",
      ],
    },
  },
  {
    nodeId: "analysis-multi-omic-integration@1",
    allOf: ["analysis-batch-confounding@1"],
    atLeast: {
      count: 2,
      of: [
        "assay-atac-cuttag@1",
        "assay-bisulfite-methylation@1",
        "assay-rna-seq@1",
      ],
    },
  },
  {
    nodeId: "assay-single-cell-multiome@1",
    allOf: ["foundation-cell-identity@1"],
    atLeast: {
      count: 2,
      of: [
        "assay-atac-cuttag@1",
        "assay-bisulfite-methylation@1",
        "assay-rna-seq@1",
      ],
    },
  },
  {
    nodeId: "intervention-epigenome-editing@1",
    allOf: ["analysis-causal-graphs@1"],
    atLeast: {
      count: 1,
      of: ["assay-atac-cuttag@1", "assay-bisulfite-methylation@1"],
    },
  },
] as const;

function fail(label: string, message: string): never {
  throw new LifeGardenDataError(`${label}: ${message}`);
}

function rejectExcessiveJsonNesting(raw: string, label: string): void {
  let depth = 0;
  let inString = false;
  let escaped = false;
  for (const character of raw) {
    if (inString) {
      if (escaped) {
        escaped = false;
      } else if (character === "\\") {
        escaped = true;
      } else if (character === '"') {
        inString = false;
      }
      continue;
    }
    if (character === '"') {
      inString = true;
    } else if (character === "{" || character === "[") {
      depth += 1;
      if (depth > 64) fail(label, "JSON nesting exceeds the v1 limit of 64");
    } else if (character === "}" || character === "]") {
      depth -= 1;
    }
  }
}

function rejectDuplicateJsonKeys(raw: string, label: string): void {
  let offset = 0;
  const whitespace = (): void => {
    while (/\s/.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = (): string => {
    const start = offset;
    offset += 1;
    while (offset < raw.length) {
      if (raw[offset] === "\\") {
        offset += 2;
        continue;
      }
      if (raw[offset] === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset)) as string;
      }
      offset += 1;
    }
    return fail(label, "unterminated JSON string");
  };
  const scanValue = (path: string): void => {
    whitespace();
    const token = raw[offset];
    if (token === "{") {
      offset += 1;
      whitespace();
      const keys = new Set<string>();
      if (raw[offset] === "}") {
        offset += 1;
        return;
      }
      while (offset < raw.length) {
        whitespace();
        const key = scanString();
        const keyPath = `${path}.${key}`;
        if (keys.has(key)) fail(label, `${keyPath}: duplicate JSON object key`);
        keys.add(key);
        whitespace();
        offset += 1;
        scanValue(keyPath);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        offset += 1;
      }
      return;
    }
    if (token === "[") {
      offset += 1;
      whitespace();
      if (raw[offset] === "]") {
        offset += 1;
        return;
      }
      let index = 0;
      while (offset < raw.length) {
        scanValue(`${path}[${index}]`);
        whitespace();
        if (raw[offset] === "]") {
          offset += 1;
          return;
        }
        offset += 1;
        index += 1;
      }
      return;
    }
    if (token === '"') {
      scanString();
      return;
    }
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset] ?? "")) {
      offset += 1;
    }
  };
  scanValue("$");
}

function preflight(raw: string, label: string): void {
  if (Buffer.byteLength(raw, "utf8") > STATIC_STANDARD_MAX_BYTES) {
    fail(label, `document exceeds ${STATIC_STANDARD_MAX_BYTES} UTF-8 bytes`);
  }
  try {
    JSON.parse(raw);
  } catch (error) {
    fail(label, `invalid JSON: ${error instanceof Error ? error.message : "unknown"}`);
  }
  rejectExcessiveJsonNesting(raw, label);
  rejectDuplicateJsonKeys(raw, label);
}

function digest(raw: string): string {
  return createHash("sha256").update(raw).digest("hex");
}

export interface GardenValidationSummary {
  nodeCount: number;
  edgeCount: number;
  questCount: number;
  maximumDepth: number;
  maximumFanOut: number;
  digest: string;
}

export function validateGardenRaw(raw: string): GardenValidationSummary {
  preflight(raw, "epigenetics garden");
  const actualDigest = digest(raw);
  if (actualDigest !== EPIGENETICS_GARDEN_SHA256) {
    fail("epigenetics garden", "document digest differs from the reviewed runtime pin");
  }
  const garden: EpigeneticsCapabilityGarden =
    parseEpigeneticsCapabilityGardenJson(raw);
  if (
    JSON.stringify(garden.prerequisiteSemantics.grouped) !==
    JSON.stringify(EXPECTED_GROUPED_PREREQUISITES)
  ) {
    fail("epigenetics garden", "reviewed v1 grouped-prerequisite shape changed");
  }
  for (const quest of garden.nodes.filter((node) => node.kind === "quest")) {
    if (quest.acceptance === null) {
      fail("epigenetics garden", `${quest.id} lacks bounded acceptance`);
    }
    const scopeHash = createHash("sha256")
      .update(quest.acceptance.scopeBounds.join("\n"))
      .digest("hex");
    if (scopeHash !== quest.acceptance.scopeHash) {
      fail("epigenetics garden", `${quest.id} scope hash is not canonical`);
    }
  }
  const byId = new Map(garden.nodes.map((node) => [node.id, node]));
  const depthMemo = new Map<string, number>();
  const depth = (id: string): number => {
    const cached = depthMemo.get(id);
    if (cached !== undefined) return cached;
    const node = byId.get(id);
    if (!node) fail("epigenetics garden", `missing node ${id}`);
    const result =
      node.prerequisites.length === 0
        ? 1
        : 1 + Math.max(...node.prerequisites.map(depth));
    depthMemo.set(id, result);
    return result;
  };
  const fanOut = new Map(garden.nodes.map((node) => [node.id, 0]));
  for (const node of garden.nodes) {
    for (const prerequisite of node.prerequisites) {
      fanOut.set(prerequisite, (fanOut.get(prerequisite) ?? 0) + 1);
    }
  }
  const summary = {
    nodeCount: garden.nodes.length,
    edgeCount: garden.nodes.reduce(
      (total, node) => total + node.prerequisites.length,
      0,
    ),
    questCount: garden.nodes.filter((node) => node.kind === "quest").length,
    maximumDepth: Math.max(...garden.nodes.map((node) => depth(node.id))),
    maximumFanOut: Math.max(...fanOut.values()),
    digest: actualDigest,
  };
  if (
    summary.nodeCount !== 25 ||
    summary.edgeCount !== 58 ||
    summary.questCount !== 3
  ) {
    fail("epigenetics garden", "reviewed v1 graph shape changed");
  }
  if (summary.maximumDepth > 10) {
    fail("epigenetics garden", "graph depth exceeds the reviewed limit of 10");
  }
  if (summary.maximumFanOut > 12) {
    fail("epigenetics garden", "graph fan-out exceeds the reviewed limit of 12");
  }
  return summary;
}

export interface KarmaValidationSummary {
  eventKindCount: number;
  invariantCount: number;
  prohibitedUseCount: number;
  closedGateCount: number;
  digest: string;
}

export function validateKarmaRaw(raw: string): KarmaValidationSummary {
  preflight(raw, "KARMA foundation");
  const actualDigest = digest(raw);
  if (actualDigest !== KARMA_FOUNDATION_SHA256) {
    fail("KARMA foundation", "document digest differs from the reviewed runtime pin");
  }
  const karma: KarmaFoundation = parseKarmaFoundationJson(raw);
  const summary = {
    eventKindCount: karma.eventVocabulary.length,
    invariantCount: karma.invariants.length,
    prohibitedUseCount: karma.prohibitedUses.length,
    closedGateCount: karma.futureGovernanceGates.filter((gate) => !gate.passed)
      .length,
    digest: actualDigest,
  };
  if (
    summary.eventKindCount !== 7 ||
    summary.invariantCount !== 9 ||
    summary.prohibitedUseCount !== 9 ||
    summary.closedGateCount !== 8
  ) {
    fail("KARMA foundation", "reviewed v1 constitutional shape changed");
  }
  return summary;
}

function runCli(): void {
  if (process.argv.length !== 4) {
    console.error(
      "usage: tsx scripts/validate-life-garden.ts GARDEN_JSON KARMA_JSON",
    );
    process.exitCode = 2;
    return;
  }
  const gardenPath = process.argv[2];
  const karmaPath = process.argv[3];
  if (gardenPath === undefined || karmaPath === undefined) {
    process.exitCode = 2;
    return;
  }
  try {
    const garden = validateGardenRaw(readFileSync(resolve(gardenPath), "utf8"));
    const karma = validateKarmaRaw(readFileSync(resolve(karmaPath), "utf8"));
    console.log(
      `life garden: PASS (${garden.nodeCount} nodes, ${garden.edgeCount} edges, depth ${garden.maximumDepth}, fan-out ${garden.maximumFanOut}, ${garden.questCount} quests; KARMA ${karma.eventKindCount} event kinds, ${karma.invariantCount} invariants, ${karma.closedGateCount} gates closed)`,
    );
  } catch (error) {
    console.error(
      `life garden: FAIL: ${error instanceof Error ? error.message : "unknown error"}`,
    );
    process.exitCode = 1;
  }
}

if (
  process.argv[1] !== undefined &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runCli();
}
