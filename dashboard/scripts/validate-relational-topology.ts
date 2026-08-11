import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

import {
  RELATIONAL_TOPOLOGY_SHA256,
  RelationalTopologyDataError,
  parseRelationalTopologyJson,
} from "../src/relational-topology";

const EXPECTED_SHAPE = {
  principleCount: 6,
  planeCount: 4,
  nodeCount: 17,
  edgeCount: 30,
  conflictCount: 6,
  forbiddenPathCount: 5,
} as const;

function digest(raw: string): string {
  return createHash("sha256").update(raw).digest("hex");
}

function fail(message: string): never {
  throw new RelationalTopologyDataError(`relational topology: ${message}`);
}

export interface RelationalTopologyValidationSummary {
  principleCount: number;
  planeCount: number;
  nodeCount: number;
  edgeCount: number;
  conflictCount: number;
  forbiddenPathCount: number;
  sourcePinCount: number;
  digest: string;
}

/**
 * Validate the exact public artifact plus the bytes of every source it pins.
 * The static projection is deliberately never accepted as an authority input.
 */
export function validateRelationalTopologyRaw(
  raw: string,
  sourceBytes: ReadonlyMap<string, string>,
): RelationalTopologyValidationSummary {
  const actualDigest = digest(raw);
  if (actualDigest !== RELATIONAL_TOPOLOGY_SHA256) {
    fail("document digest differs from the reviewed runtime pin");
  }

  const topology = parseRelationalTopologyJson(raw);
  for (const source of topology.sourcePins) {
    const bytes = sourceBytes.get(source.path);
    if (bytes === undefined) fail(`missing pinned source bytes for ${source.path}`);
    if (digest(bytes) !== source.sha256) {
      fail(`pinned source bytes drifted for ${source.path}`);
    }
  }
  if (sourceBytes.size !== topology.sourcePins.length) {
    fail("source input set contains an unreviewed or duplicate path");
  }

  const summary: RelationalTopologyValidationSummary = {
    principleCount: topology.principles.length,
    planeCount: topology.planes.length,
    nodeCount: topology.nodes.length,
    edgeCount: topology.edges.length,
    conflictCount: topology.currentConflicts.length,
    forbiddenPathCount: topology.forbiddenPaths.length,
    sourcePinCount: topology.sourcePins.length,
    digest: actualDigest,
  };
  for (const [key, expected] of Object.entries(EXPECTED_SHAPE) as Array<
    [keyof typeof EXPECTED_SHAPE, number]
  >) {
    if (summary[key] !== expected) {
      fail(`reviewed v0 shape changed at ${key}`);
    }
  }
  if (summary.sourcePinCount !== 2) fail("reviewed source pin set changed");
  return summary;
}

function runCli(): void {
  if (process.argv.length !== 5) {
    console.error(
      "usage: tsx scripts/validate-relational-topology.ts TOPOLOGY_JSON AUTHORITATIVE_STATE MONEY_KARMA",
    );
    process.exitCode = 2;
    return;
  }
  const topologyPath = process.argv[2];
  const authoritativeStatePath = process.argv[3];
  const moneyKarmaPath = process.argv[4];
  if (
    topologyPath === undefined ||
    authoritativeStatePath === undefined ||
    moneyKarmaPath === undefined
  ) {
    process.exitCode = 2;
    return;
  }

  try {
    const summary = validateRelationalTopologyRaw(
      readFileSync(resolve(topologyPath), "utf8"),
      new Map([
        [
          "docs/AUTHORITATIVE-STATE.md",
          readFileSync(resolve(authoritativeStatePath), "utf8"),
        ],
        [
          "docs/constitution/money-karma-v1.json",
          readFileSync(resolve(moneyKarmaPath), "utf8"),
        ],
      ]),
    );
    console.log(
      `relational topology: PASS (${summary.principleCount} principles, ${summary.planeCount} planes, ${summary.nodeCount} nodes, ${summary.edgeCount} edges, ${summary.conflictCount} known conflicts, ${summary.forbiddenPathCount} same-flow guards, ${summary.sourcePinCount} source pins)`,
    );
  } catch (error) {
    console.error(
      `relational topology: FAIL: ${error instanceof Error ? error.message : "unknown error"}`,
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
