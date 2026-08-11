import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { describe, it } from "node:test";
import {
  AUTHORITY_GEOMETRY_ENDPOINT,
  AUTHORITY_GEOMETRY_MAX_BYTES,
  AUTHORITY_GEOMETRY_SHA256,
  assertAuthorityGeometryTargetGate,
  assessAuthorityGeometry,
  fetchAuthorityGeometry,
  parseAuthorityGeometry,
  parseAuthorityGeometryJson,
  renderAuthorityGeometry,
} from "../src/authority-geometry";
import {
  validateAuthorityGeometryRaw,
  validateAuthorityGeometrySourceAnchors,
} from "../scripts/validate-authority-geometry";

type MutableDocument = Record<string, any>;

const manifestUrl = new URL(
  "../public/standards/authority-geometry.v1.json",
  import.meta.url,
);
const manifestPath = fileURLToPath(manifestUrl);
const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const canonicalRaw = readFileSync(manifestUrl, "utf8");
const canonicalDocument = JSON.parse(canonicalRaw) as MutableDocument;
const sourceRaw = readFileSync(
  new URL("../src/authority-geometry.ts", import.meta.url),
  "utf8",
);

function copyManifest(): MutableDocument {
  return structuredClone(canonicalDocument) as MutableDocument;
}

function edgeById(document: MutableDocument, id: string): MutableDocument {
  const edge = document.edges.find((candidate: MutableDocument) => candidate.id === id);
  assert.ok(edge, `missing edge fixture ${id}`);
  return edge as MutableDocument;
}

function responseAt(
  body: BodyInit,
  url = "https://zerone.ai/standards/authority-geometry.v1.json",
  init: ResponseInit = {},
  redirected = false,
): Response {
  const response = new Response(body, {
    ...init,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...init.headers,
    },
  });
  Object.defineProperty(response, "url", { value: url });
  Object.defineProperty(response, "redirected", { value: redirected });
  return response;
}

class TestDomNode {
  readonly tagName: string;
  className = "";
  textContent: string | null = null;
  readonly children: TestDomNode[] = [];
  readonly attributes = new Map<string, string>();
  href = "";
  target = "";
  rel = "";
  type = "";
  ariaBusy = "false";

  constructor(tagName: string) {
    this.tagName = tagName.toUpperCase();
  }

  append(...nodes: TestDomNode[]): void {
    this.children.push(...nodes);
  }

  replaceChildren(...nodes: TestDomNode[]): void {
    this.children.splice(0, this.children.length, ...nodes);
    this.textContent = null;
  }

  setAttribute(name: string, value: string): void {
    this.attributes.set(name, value);
  }

  addEventListener(): void {
    // Renderer tests do not dispatch events.
  }
}

function testNodeText(node: TestDomNode): string {
  return `${node.textContent ?? ""}${node.children.map(testNodeText).join("")}`;
}

function testNodes(
  root: TestDomNode,
  predicate: (node: TestDomNode) => boolean,
): TestDomNode[] {
  const matches = predicate(root) ? [root] : [];
  for (const child of root.children) matches.push(...testNodes(child, predicate));
  return matches;
}

function withTestDocument<T>(run: () => T): T {
  const original = Object.getOwnPropertyDescriptor(globalThis, "document");
  const testDocument = {
    createElement: (tagName: string) => new TestDomNode(tagName),
    createTextNode: (text: string) => {
      const node = new TestDomNode("#text");
      node.textContent = text;
      return node;
    },
  };
  Object.defineProperty(globalThis, "document", {
    configurable: true,
    value: testDocument,
  });
  try {
    return run();
  } finally {
    if (original === undefined) delete (globalThis as { document?: unknown }).document;
    else Object.defineProperty(globalThis, "document", original);
  }
}

describe("Authority Geometry v1", () => {
  it("pins the exact canonical bytes and parses the source-only observatory", () => {
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      AUTHORITY_GEOMETRY_SHA256,
    );
    const geometry = parseAuthorityGeometryJson(canonicalRaw);
    assert.equal(geometry.schema, "zerone.authority-geometry/v1");
    assert.equal(geometry.status, "SOURCE_OBSERVATORY_ONLY");
    assert.equal(geometry.currentTruth.liveEvidence, "NOT_ESTABLISHED_BY_THIS_ARTIFACT");
    assert.equal(geometry.capabilities.length, 12);
    assert.equal(geometry.nodes.length, 18);
    assert.equal(geometry.edges.length, 26);
    assert.equal(geometry.currentFindings.length, 7);
    assert.equal(geometry.sourceAnchors.length, 17);
  });

  it("computes the current NO-GO assessment instead of promoting a green report", () => {
    const assessment = assessAuthorityGeometry(
      parseAuthorityGeometryJson(canonicalRaw),
    );
    assert.deepEqual(assessment, {
      overall: "NO_GO",
      currentFindingCount: 7,
      currentEdgeCount: 12,
      targetEdgeCount: 14,
      dualSurfaceCapabilityCount: 5,
      targetForbiddenPathCount: 0,
      staticAuthoritySurfacesPassing: 1,
      staticAuthoritySurfacesTotal: 7,
      h4GatesEvidenced: 0,
      h4GatesTotal: 24,
      h5GatesEvidenced: 0,
      h5GatesTotal: 14,
      targetGateSatisfied: false,
    });
  });

  it("keeps every release effect false and every activation gate unevidenced", () => {
    const geometry = parseAuthorityGeometryJson(canonicalRaw);
    assert.ok(Object.values(geometry.releaseBoundary).every((value) => value === false));
    assert.equal(geometry.activationGates.h4.length, 24);
    assert.equal(geometry.activationGates.h5.length, 14);
    assert.ok(
      [...geometry.activationGates.h4, ...geometry.activationGates.h5].every(
        (gate) => gate.status === "NOT_EVIDENCED",
      ),
    );
  });

  it("assigns every capability to exactly one matching target writer", () => {
    const geometry = parseAuthorityGeometryJson(canonicalRaw);
    for (const capability of geometry.capabilities) {
      const owners = geometry.nodes.filter((node) =>
        node.targetCapabilities.includes(capability.id),
      );
      assert.deepEqual(owners.map((node) => node.id), [capability.targetWriter]);
    }
  });

  it("refuses the explicit target gate on current source", () => {
    assert.throws(
      () => assertAuthorityGeometryTargetGate(parseAuthorityGeometryJson(canonicalRaw)),
      /target gate REFUSED: 1\/7 static surfaces pass, H4 0\/24, H5 0\/14/,
    );
  });

  it("rejects unknown top-level fields and duplicate JSON keys", () => {
    const extra = copyManifest();
    extra.runtimeAuthority = true;
    assert.throws(() => parseAuthorityGeometry(extra), /expected exactly/);

    const duplicate = canonicalRaw.replace(
      '"schema": "zerone.authority-geometry\/v1",',
      '"schema": "zerone.authority-geometry/v1",\n  "schema": "zerone.authority-geometry/v1",',
    );
    assert.throws(
      () => parseAuthorityGeometryJson(duplicate),
      /duplicate JSON object key/,
    );
  });

  it("rejects malformed, excessively nested, and oversized JSON", () => {
    assert.throws(
      () => parseAuthorityGeometryJson("{"),
      /contains malformed JSON/,
    );
    const nested = `${"[".repeat(65)}0${"]".repeat(65)}`;
    assert.throws(() => parseAuthorityGeometryJson(nested), /nesting exceeds 64/);
    assert.throws(
      () => parseAuthorityGeometryJson(`"${"x".repeat(AUTHORITY_GEOMETRY_MAX_BYTES)}"`),
      /exceeds 65536 UTF-8 bytes/,
    );
  });

  it("rejects a second or mismatched target writer", () => {
    const secondWriter = copyManifest();
    secondWriter.nodes[2].targetCapabilities = ["consensus-stake"];
    assert.throws(
      () => parseAuthorityGeometry(secondWriter),
      /exactly one target writer/,
    );

    const mismatchedWriter = copyManifest();
    mismatchedWriter.capabilities[0].targetWriter = "custom-staking";
    assert.throws(
      () => parseAuthorityGeometry(mismatchedWriter),
      /matching node ownership/,
    );
  });

  it("rejects unresolved graph identifiers", () => {
    const unresolvedNode = copyManifest();
    unresolvedNode.edges[0].to = "missing-node";
    assert.throws(() => parseAuthorityGeometry(unresolvedNode), /expected one of/);

    const unresolvedAnchor = copyManifest();
    unresolvedAnchor.edges[0].evidence = ["missing-anchor"];
    assert.throws(
      () => parseAuthorityGeometry(unresolvedAnchor),
      /unresolved source anchor/,
    );
  });

  it("keeps current-source edges away from target-only modules", () => {
    const document = copyManifest();
    document.edges[0].to = "controller";
    assert.throws(
      () => parseAuthorityGeometry(document),
      /current-source edge reaches a target-only node/,
    );
  });

  it("keeps current observation and accepted target evidence separate", () => {
    const targetWithCurrentEvidence = copyManifest();
    edgeById(targetWithCurrentEvidence, "target-auth-to-controller").evidence = [
      "app-wiring",
    ];
    assert.throws(
      () => parseAuthorityGeometry(targetWithCurrentEvidence),
      /accepted-target edges must be sourced only to the accepted design/,
    );

    const currentWithDesignEvidence = copyManifest();
    currentWithDesignEvidence.edges[0].evidence = ["authoritative-state-design"];
    assert.throws(
      () => parseAuthorityGeometry(currentWithDesignEvidence),
      /current-source evidence cannot be replaced/,
    );
  });

  it("fails closed if any release effect turns on", () => {
    for (const key of Object.keys(canonicalDocument.releaseBoundary)) {
      const document = copyManifest();
      document.releaseBoundary[key] = true;
      assert.throws(
        () => parseAuthorityGeometry(document),
        new RegExp(`releaseBoundary\\.${key}`),
      );
    }
  });

  it("fails closed if current source is relabelled as unified or live-evidenced", () => {
    const unified = copyManifest();
    unified.currentTruth.sourceAuthorityUnified = true;
    assert.throws(() => parseAuthorityGeometry(unified), /sourceAuthorityUnified/);

    const live = copyManifest();
    live.currentTruth.liveEvidence = "ESTABLISHED";
    assert.throws(() => parseAuthorityGeometry(live), /liveEvidence/);
  });

  it("requires exactly 24 H4 and 14 H5 gates, all closed", () => {
    const missing = copyManifest();
    missing.activationGates.h4.pop();
    assert.throws(() => parseAuthorityGeometry(missing), /exactly 24 closed gates/);

    const promoted = copyManifest();
    promoted.activationGates.h5[0].status = "EVIDENCED";
    assert.throws(() => parseAuthorityGeometry(promoted), /must remain "NOT_EVIDENCED"/);
  });

  it("computes and refuses a forbidden accepted-target wealth path", () => {
    const document = copyManifest();
    const edge = edgeById(document, "target-auth-to-controller");
    edge.from = "sdk-staking";
    edge.to = "sdk-gov";
    assert.throws(
      () => parseAuthorityGeometry(document),
      /accepted target must contain zero forbidden paths/,
    );

    const falseCounter = copyManifest();
    falseCounter.forbiddenInfluence[0].acceptedTargetPathCount = 1;
    assert.throws(() => parseAuthorityGeometry(falseCounter), /must remain 0/);

    const narrowedKarmaBoundary = copyManifest();
    narrowedKarmaBoundary.forbiddenInfluence[3].targets = [
      "sdk-staking",
      "sdk-gov",
      "electorate",
      "qualification",
      "knowledge",
    ];
    assert.throws(
      () => parseAuthorityGeometry(narrowedKarmaBoundary),
      /forbiddenInfluence\[3\]\.targets: must preserve exact identifiers/,
    );
  });

  it("pins target scope and effect so semantic relabelling cannot hide influence", () => {
    const relabelled = copyManifest();
    edgeById(relabelled, "target-controller-to-electorate").effect =
      "REFERENCE_RELATION";
    assert.throws(
      () => parseAuthorityGeometry(relabelled),
      /must remain "CONTROL_RELATION"/,
    );

    const removedFromTarget = copyManifest();
    edgeById(removedFromTarget, "target-sdk-policy-to-ontology").scope =
      "CURRENT_SOURCE";
    assert.throws(
      () => parseAuthorityGeometry(removedFromTarget),
      /must remain "ACCEPTED_TARGET"/,
    );

    const geometry = parseAuthorityGeometryJson(canonicalRaw);
    assert.deepEqual(
      geometry.edges
        .filter((edge) =>
          ["REFERENCE_RELATION", "EVIDENCE_RELATION", "RETIREMENT_RELATION"].includes(
            edge.effect,
          ),
        )
        .map((edge) => edge.id),
      [
        "target-ontology-to-qualification",
        "target-ontology-to-knowledge",
        "target-knowledge-to-profile",
        "target-custom-stake-to-legacy-exit",
        "target-knowledge-domain-to-ontology",
      ],
    );
  });

  it("keeps static findings and gate statuses mutually consistent", () => {
    const failWithoutFinding = copyManifest();
    failWithoutFinding.staticAuthorityGate.surfaceChecks[1].findingIds = [];
    assert.throws(
      () => parseAuthorityGeometry(failWithoutFinding),
      /FAIL must name findings/,
    );

    const unaccounted = copyManifest();
    unaccounted.staticAuthorityGate.surfaceChecks[6].findingIds = [
      "dual-staking-ledgers",
    ];
    assert.throws(
      () => parseAuthorityGeometry(unaccounted),
      /must account for every current finding/,
    );
  });

  it("rejects assessment counters that differ from computed graph state", () => {
    const document = copyManifest();
    document.releaseAssessment.staticAuthoritySurfacesPassing = 2;
    assert.throws(
      () => parseAuthorityGeometry(document),
      /staticAuthoritySurfacesPassing/,
    );
  });

  it("verifies every exact repository source hash and required or forbidden snippet", () => {
    const geometry = parseAuthorityGeometryJson(canonicalRaw);
    const summary = validateAuthorityGeometrySourceAnchors(
      geometry,
      repositoryRoot,
    );
    assert.equal(summary.sourceAnchorCount, 17);
    assert.ok(summary.requiredSnippetCount >= 30);
    assert.equal(summary.forbiddenSnippetCount, 1);

    const full = validateAuthorityGeometryRaw(canonicalRaw, { repositoryRoot });
    assert.equal(full.manifestSha256, AUTHORITY_GEOMETRY_SHA256);
    assert.equal(full.overall, "NO_GO");
  });

  it("rejects a structurally valid but false source-anchor digest", () => {
    const document = copyManifest();
    document.sourceAnchors[1].sha256 = "0".repeat(64);
    const raw = JSON.stringify(document);
    assert.throws(
      () =>
        validateAuthorityGeometryRaw(raw, {
          repositoryRoot,
          requireCanonicalDigest: false,
        }),
      /app-wiring: app\/app\.go SHA-256 mismatch/,
    );
  });

  it("rejects every declared source anchor that no graph fact references", () => {
    const document = copyManifest();
    for (const finding of document.currentFindings) {
      finding.evidence = finding.evidence.map((id: string) =>
        id === "ontology-writer" ? "app-wiring" : id,
      );
    }
    assert.throws(
      () => parseAuthorityGeometry(document),
      /sourceAnchors\.ontology-writer: source anchor is not referenced by a graph fact/,
    );
  });

  it("uses a bounded same-origin no-store request and accepts the exact digest", async () => {
    let requestInput: RequestInfo | URL | undefined;
    let requestInit: RequestInit | undefined;
    const fetcher = (async (input: RequestInfo | URL, init?: RequestInit) => {
      requestInput = input;
      requestInit = init;
      return responseAt(canonicalRaw);
    }) as typeof fetch;
    const geometry = await fetchAuthorityGeometry({
      fetcher,
      baseUrl: "https://zerone.ai/explorer",
    });
    assert.equal(geometry.releaseAssessment.overall, "NO_GO");
    assert.equal(requestInput, AUTHORITY_GEOMETRY_ENDPOINT);
    assert.equal(requestInit?.cache, "no-store");
    assert.equal(requestInit?.credentials, "same-origin");
    assert.equal(requestInit?.redirect, "error");
    assert.ok(requestInit?.signal instanceof AbortSignal);
    assert.deepEqual(requestInit?.headers, { Accept: "application/json" });
  });

  it("refuses redirects, cross-origin final URLs, digest drift, and non-JSON", async () => {
    await assert.rejects(
      fetchAuthorityGeometry({
        fetcher: (async () => responseAt(canonicalRaw, undefined, {}, true)) as typeof fetch,
        baseUrl: "https://zerone.ai/",
      }),
      /response was redirected/,
    );
    await assert.rejects(
      fetchAuthorityGeometry({
        fetcher: (async () =>
          responseAt(
            canonicalRaw,
            "https://attacker.example/standards/authority-geometry.v1.json",
          )) as typeof fetch,
        baseUrl: "https://zerone.ai/",
      }),
      /left its canonical same-origin path/,
    );
    await assert.rejects(
      fetchAuthorityGeometry({
        fetcher: (async () =>
          responseAt(canonicalRaw.replace("Authority Geometry", "Authority geometry"))) as typeof fetch,
        baseUrl: "https://zerone.ai/",
      }),
      /did not match the reviewed canonical digest/,
    );
    await assert.rejects(
      fetchAuthorityGeometry({
        fetcher: (async () =>
          responseAt(canonicalRaw, undefined, {
            headers: { "content-type": "text/plain" },
          })) as typeof fetch,
        baseUrl: "https://zerone.ai/",
      }),
      /returned non-JSON content/,
    );
  });

  it("accepts only exact JSON media types followed by valid parameters", async () => {
    for (const contentType of [
      "application/json",
      "application/json; charset=utf-8",
      "application/problem+json",
      'application/vnd.zerone.geometry+json; charset="utf-8"',
    ]) {
      const geometry = await fetchAuthorityGeometry({
        fetcher: (async () =>
          responseAt(canonicalRaw, undefined, {
            headers: { "content-type": contentType },
          })) as typeof fetch,
        baseUrl: "https://zerone.ai/",
      });
      assert.equal(geometry.schema, "zerone.authority-geometry/v1");
    }

    for (const contentType of [
      "application/json+evil",
      "application/json/evil",
      "application/+json",
      "application/json; garbage",
      "text/json",
    ]) {
      await assert.rejects(
        fetchAuthorityGeometry({
          fetcher: (async () =>
            responseAt(canonicalRaw, undefined, {
              headers: { "content-type": contentType },
            })) as typeof fetch,
          baseUrl: "https://zerone.ai/",
        }),
        /returned non-JSON content/,
      );
    }
  });

  it("aborts and cancels every response body refused before streaming", async () => {
    const cases = [
      {
        label: "HTTP status",
        pattern: /returned HTTP 503/,
        url: "https://zerone.ai/standards/authority-geometry.v1.json",
        init: { status: 503 },
        redirected: false,
      },
      {
        label: "redirect flag",
        pattern: /response was redirected/,
        url: "https://zerone.ai/standards/authority-geometry.v1.json",
        init: {},
        redirected: true,
      },
      {
        label: "final URL",
        pattern: /left its canonical same-origin path/,
        url: "https://attacker.example/standards/authority-geometry.v1.json",
        init: {},
        redirected: false,
      },
      {
        label: "media type",
        pattern: /returned non-JSON content/,
        url: "https://zerone.ai/standards/authority-geometry.v1.json",
        init: { headers: { "content-type": "text/plain" } },
        redirected: false,
      },
      {
        label: "declared size",
        pattern: /exceeded its size limit/,
        url: "https://zerone.ai/standards/authority-geometry.v1.json",
        init: {
          headers: {
            "content-length": String(AUTHORITY_GEOMETRY_MAX_BYTES + 1),
          },
        },
        redirected: false,
      },
    ] as const;

    for (const refusal of cases) {
      let cancelled = false;
      let signal: AbortSignal | undefined;
      const body = new ReadableStream<Uint8Array>({
        cancel: () => {
          cancelled = true;
        },
      });
      const response = responseAt(
        body,
        refusal.url,
        refusal.init,
        refusal.redirected,
      );
      await assert.rejects(
        fetchAuthorityGeometry({
          fetcher: (async (_input, init) => {
            signal = init?.signal as AbortSignal | undefined;
            return response;
          }) as typeof fetch,
          baseUrl: "https://zerone.ai/",
        }),
        refusal.pattern,
        refusal.label,
      );
      await Promise.resolve();
      assert.equal(signal?.aborted, true, `${refusal.label} did not abort`);
      assert.equal(cancelled, true, `${refusal.label} did not cancel its body`);
    }
  });

  it("refuses declared or streamed responses beyond the byte cap", async () => {
    await assert.rejects(
      fetchAuthorityGeometry({
        fetcher: (async () =>
          responseAt(canonicalRaw, undefined, {
            headers: {
              "content-length": String(AUTHORITY_GEOMETRY_MAX_BYTES + 1),
            },
          })) as typeof fetch,
        baseUrl: "https://zerone.ai/",
      }),
      /exceeded its size limit/,
    );
    await assert.rejects(
      fetchAuthorityGeometry({
        fetcher: (async () =>
          responseAt("x".repeat(AUTHORITY_GEOMETRY_MAX_BYTES + 1))) as typeof fetch,
        baseUrl: "https://zerone.ai/",
      }),
      /exceeded its size limit/,
    );
  });

  it("enforces its deadline even if an injected fetch or body stream ignores abort", async () => {
    await assert.rejects(
      fetchAuthorityGeometry({
        fetcher: (() => new Promise<Response>(() => undefined)) as typeof fetch,
        baseUrl: "https://zerone.ai/",
        timeoutMs: 10,
      }),
      /request timed out/,
    );

    const neverEndingBody = new ReadableStream<Uint8Array>({
      pull: () => new Promise<void>(() => undefined),
    });
    await assert.rejects(
      fetchAuthorityGeometry({
        fetcher: (async () => responseAt(neverEndingBody)) as typeof fetch,
        baseUrl: "https://zerone.ai/",
        timeoutMs: 10,
      }),
      /request timed out/,
    );
  });

  it("exits nonzero when the CLI is explicitly asked for the target gate", () => {
    const tsx = fileURLToPath(new URL("../node_modules/.bin/tsx", import.meta.url));
    const cli = fileURLToPath(
      new URL("../scripts/validate-authority-geometry.ts", import.meta.url),
    );
    const result = spawnSync(tsx, [cli, manifestPath, "--target-gate"], {
      cwd: fileURLToPath(new URL("../", import.meta.url)),
      encoding: "utf8",
    });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /target gate REFUSED/);
    assert.match(result.stderr, /current source remains NO-GO/);
  });

  it("keeps the renderer DOM-safe and exposes the complete CSS contract", () => {
    assert.doesNotMatch(
      sourceRaw,
      /\.innerHTML\b|\.outerHTML\b|insertAdjacentHTML|document\.write|DOMParser/,
    );
    for (const className of [
      "authority-geometry-shell",
      "authority-geometry-status",
      "authority-geometry-metrics",
      "authority-geometry-planes",
      "authority-geometry-plane",
      "authority-geometry-principles",
      "authority-geometry-findings",
      "authority-geometry-table-wrap",
      "authority-geometry-table",
      "authority-geometry-footer",
      "authority-geometry-error",
      "authority-geometry-loading",
    ]) {
      assert.ok(sourceRaw.includes(`"${className}"`), `missing ${className}`);
    }
    assert.match(sourceRaw, /createElement/);
    assert.match(sourceRaw, /textContent/);
    assert.match(sourceRaw, /target = "_blank"/);
    assert.match(sourceRaw, /rel = "noreferrer"/);
  });

  it("renders both the seven-surface gate and every relationship as accessible text", () => {
    const geometry = parseAuthorityGeometryJson(canonicalRaw);
    const shell = withTestDocument(
      () => renderAuthorityGeometry(geometry) as unknown as TestDomNode,
    );
    const tables = testNodes(
      shell,
      (node) =>
        node.tagName === "TABLE" && node.className === "authority-geometry-table",
    );
    assert.equal(tables.length, 2);

    const surfaceTable = tables.find((table) =>
      testNodeText(table).includes("Static authority release-gate assessment"),
    );
    const relationshipTable = tables.find((table) =>
      testNodeText(table).includes("Every classified authority relationship"),
    );
    assert.ok(surfaceTable);
    assert.ok(relationshipTable);

    const surfaceBody = surfaceTable.children.find((node) => node.tagName === "TBODY");
    assert.equal(surfaceBody?.children.length, 7);

    const relationshipBody = relationshipTable.children.find(
      (node) => node.tagName === "TBODY",
    );
    assert.equal(relationshipBody?.children.length, geometry.edges.length);
    for (const [index, edge] of geometry.edges.entries()) {
      const row: TestDomNode | undefined = relationshipBody?.children[index];
      assert.ok(row, `missing relationship row ${edge.id}`);
      const text = testNodeText(row);
      for (const expected of [
        edge.id,
        edge.from,
        edge.to,
        edge.relationship,
        edge.scope,
        edge.effect,
        ...edge.evidence,
        edge.protections.consentBoundary,
        edge.protections.challengeRoute,
        edge.protections.repairRoute,
        edge.protections.exitRoute,
      ]) {
        assert.ok(text.includes(expected), `${edge.id} omitted ${expected}`);
      }
      assert.equal(row.children[0]?.attributes.get("scope"), "row");
    }
    const columnHeaders = testNodes(
      relationshipTable,
      (node) => node.tagName === "TH" && node.attributes.get("scope") === "col",
    );
    assert.equal(columnHeaders.length, 11);
  });
});
