import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  RELATIONAL_TOPOLOGY_ENDPOINT,
  RELATIONAL_TOPOLOGY_MAX_BYTES,
  RELATIONAL_TOPOLOGY_SHA256,
  fetchRelationalTopology,
  hasRelationalFlowPath,
  initialiseRelationalTopology,
  parseRelationalTopology,
  parseRelationalTopologyJson,
  renderRelationalTopology,
} from "../src/relational-topology";
import { validateRelationalTopologyRaw } from "../scripts/validate-relational-topology";

type MutableDocument = Record<string, any>;

const canonicalRaw = readFileSync(
  new URL("../public/standards/relational-topology.v0.json", import.meta.url),
  "utf8",
);
const authoritativeStateRaw = readFileSync(
  new URL("../../docs/AUTHORITATIVE-STATE.md", import.meta.url),
  "utf8",
);
const moneyKarmaRaw = readFileSync(
  new URL("../../docs/constitution/money-karma-v1.json", import.meta.url),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as MutableDocument;

function copy(): MutableDocument {
  return structuredClone(canonical) as MutableDocument;
}

function byId(
  values: MutableDocument[],
  id: string,
): MutableDocument {
  const found = values.find((value) => value.id === id);
  assert.ok(found, `missing canonical fixture ${id}`);
  return found;
}

function sourceBytes(
  overrides: ReadonlyMap<string, string> = new Map(),
): ReadonlyMap<string, string> {
  return new Map([
    [
      "docs/AUTHORITATIVE-STATE.md",
      overrides.get("docs/AUTHORITATIVE-STATE.md") ?? authoritativeStateRaw,
    ],
    [
      "docs/constitution/money-karma-v1.json",
      overrides.get("docs/constitution/money-karma-v1.json") ?? moneyKarmaRaw,
    ],
  ]);
}

function jsonResponse(
  body: BodyInit | null = canonicalRaw,
  init: ResponseInit = {},
  url = `https://zerone.ai${RELATIONAL_TOPOLOGY_ENDPOINT}`,
): Response {
  const headers = new Headers(init.headers);
  if (!headers.has("content-type")) {
    headers.set("content-type", "application/json; charset=utf-8");
  }
  const response = new Response(body, { ...init, headers });
  Object.defineProperty(response, "url", { value: url });
  return response;
}

async function settlesWithin<T>(promise: Promise<T>, milliseconds: number): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      promise,
      new Promise<never>((_resolve, reject) => {
        timer = setTimeout(
          () => reject(new Error(`promise did not settle within ${milliseconds}ms`)),
          milliseconds,
        );
      }),
    ]);
  } finally {
    if (timer !== undefined) clearTimeout(timer);
  }
}

class FakeClassList {
  constructor(private readonly element: FakeElement) {}

  toggle(token: string, force?: boolean): boolean {
    const names = new Set(this.element.className.split(/\s+/u).filter(Boolean));
    const present = force ?? !names.has(token);
    if (present) names.add(token);
    else names.delete(token);
    this.element.className = [...names].join(" ");
    return present;
  }
}

class FakeElement {
  readonly children: FakeElement[] = [];
  readonly attributes = new Map<string, string>();
  readonly dataset: Record<string, string> = {};
  readonly listeners = new Map<string, Array<() => void>>();
  readonly classList = new FakeClassList(this);
  className = "";
  textContent: string | null = null;
  href = "";
  target = "";
  rel = "";
  type = "";
  tabIndex = -1;
  hidden = false;

  constructor(readonly tagName: string) {}

  set innerHTML(_value: string) {
    throw new Error("renderer must not use innerHTML");
  }

  get firstElementChild(): FakeElement | null {
    return this.children[0] ?? null;
  }

  append(...nodes: FakeElement[]): void {
    this.children.push(...nodes);
  }

  replaceChildren(...nodes: FakeElement[]): void {
    this.children.splice(0, this.children.length, ...nodes);
  }

  setAttribute(name: string, value: string): void {
    this.attributes.set(name, value);
  }

  getAttribute(name: string): string | null {
    return this.attributes.get(name) ?? null;
  }

  addEventListener(name: string, listener: () => void): void {
    const listeners = this.listeners.get(name) ?? [];
    listeners.push(listener);
    this.listeners.set(name, listeners);
  }

  click(): void {
    for (const listener of this.listeners.get("click") ?? []) listener();
  }
}

function descendants(node: FakeElement): FakeElement[] {
  return [node, ...node.children.flatMap(descendants)];
}

function flattenedText(node: FakeElement): string {
  return [node.textContent ?? "", ...node.children.map(flattenedText)].join(" ");
}

async function withFakeDocument<T>(run: () => T | Promise<T>): Promise<T> {
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, "document");
  Object.defineProperty(globalThis, "document", {
    configurable: true,
    value: {
      createElement(tag: string) {
        return new FakeElement(tag);
      },
    },
  });
  try {
    return await run();
  } finally {
    if (descriptor === undefined) delete (globalThis as { document?: unknown }).document;
    else Object.defineProperty(globalThis, "document", descriptor);
  }
}

describe("relational topology v0", () => {
  it("pins and validates the reviewed source-only graph", () => {
    const topology = parseRelationalTopologyJson(canonicalRaw);
    assert.equal(topology.schema, "zerone.relational-topology/v0");
    assert.equal(topology.principles.length, 6);
    assert.equal(topology.planes.length, 4);
    assert.equal(topology.nodes.length, 17);
    assert.equal(topology.edges.length, 30);
    assert.equal(topology.currentConflicts.length, 6);
    assert.equal(topology.forbiddenPaths.length, 5);
    assert.equal(topology.status.authoritative, false);
    assert.equal(topology.status.currentNetworkStateClaimed, false);
    assert.equal(topology.status.consensusRuntimeDeployed, false);
    assert.equal(topology.status.networkActivated, false);
    assert.ok(Object.values(topology.releaseBoundary).every((value) => value === false));
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      RELATIONAL_TOPOLOGY_SHA256,
    );
    assert.ok(Buffer.byteLength(canonicalRaw, "utf8") < RELATIONAL_TOPOLOGY_MAX_BYTES);
  });

  it("keeps declared ethics distinct from source-derived claims", () => {
    const topology = parseRelationalTopologyJson(canonicalRaw);
    const voluntary = byId(topology.principles as unknown as MutableDocument[], "voluntary-relation");
    assert.equal(voluntary.basis, "DECLARED_V0_PRINCIPLE");
    assert.deepEqual(voluntary.sourceRefs, []);
    assert.ok(
      topology.principles
        .filter((principle) => principle.basis === "SOURCE_DERIVED")
        .every((principle) => principle.sourceRefs.length > 0),
    );
    assert.equal(
      topology.edges.find((edge) => edge.id === "knowledge-karma-projection")?.status,
      "DESIGN_INFERENCE",
    );
  });

  it("preserves distinct custody, handler, emergency, and conflict surfaces", () => {
    const topology = parseRelationalTopologyJson(canonicalRaw);
    assert.ok(topology.nodes.some((node) => node.id === "research-fund-account"));
    assert.ok(
      topology.nodes.some((node) => node.id === "vesting-rewards-disbursement"),
    );
    assert.equal(
      topology.nodes.find((node) => node.id === "x-emergency")?.implementation,
      "TARGET_SEMANTICS_NOT_ACTIVATED",
    );
    assert.deepEqual(
      topology.currentConflicts.map((conflict) => conflict.id),
      [
        "custom-governance-writer",
        "custom-staking-writer",
        "knowledge-direct-fact-writer",
        "knowledge-domain-writer",
        "knowledge-pause-correction-writer",
        "legacy-research-disbursement-writer",
      ],
    );
  });

  it("validates exact topology shape and both source bytes", () => {
    const summary = validateRelationalTopologyRaw(canonicalRaw, sourceBytes());
    assert.deepEqual(summary, {
      principleCount: 6,
      planeCount: 4,
      nodeCount: 17,
      edgeCount: 30,
      conflictCount: 6,
      forbiddenPathCount: 5,
      sourcePinCount: 2,
      digest: RELATIONAL_TOPOLOGY_SHA256,
    });

    assert.throws(
      () =>
        validateRelationalTopologyRaw(
          canonicalRaw,
          sourceBytes(
            new Map([["docs/AUTHORITATIVE-STATE.md", `${authoritativeStateRaw}\nDRIFT`]]),
          ),
        ),
      /source bytes drifted/,
    );
    assert.throws(
      () => validateRelationalTopologyRaw(`${canonicalRaw}\n`, sourceBytes()),
      /document digest differs/,
    );
  });

  it("rejects unknown fields, duplicate keys, unsafe pins, and excessive input", () => {
    const unknown = copy();
    unknown.unreviewed = true;
    assert.throws(() => parseRelationalTopology(unknown), /unknown or missing fields/);

    const duplicate = canonicalRaw.replace(
      '"schema": "zerone.relational-topology\/v0",',
      '"schema": "zerone.relational-topology/v0",\n  "schema": "zerone.relational-topology/v0",',
    );
    assert.notEqual(duplicate, canonicalRaw);
    assert.throws(() => parseRelationalTopologyJson(duplicate), /duplicate JSON key/);

    const unsafePin = copy();
    unsafePin.sourcePins[0].path = "docs/../secrets";
    assert.throws(() => parseRelationalTopology(unsafePin), /safe docs\/ repository path/);

    assert.throws(
      () => parseRelationalTopologyJson(`"${"x".repeat(RELATIONAL_TOPOLOGY_MAX_BYTES)}"`),
      /byte limit/,
    );
  });

  it("rejects any activation or authority effect", () => {
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const document = copy();
      document.releaseBoundary[key] = true;
      assert.throws(() => parseRelationalTopology(document), new RegExp(key));
    }
    for (const key of [
      "authoritative",
      "currentNetworkStateClaimed",
      "consensusRuntimeDeployed",
      "networkActivated",
    ]) {
      const document = copy();
      document.status[key] = true;
      assert.throws(() => parseRelationalTopology(document), new RegExp(key));
    }
  });

  it("rejects unresolved, self, duplicate-ownership, and unordered graph records", () => {
    const unresolved = copy();
    unresolved.edges[0].to = "missing-node";
    assert.throws(() => parseRelationalTopology(unresolved), /unknown id missing-node/);

    const selfEdge = copy();
    selfEdge.edges[0].to = selfEdge.edges[0].from;
    assert.throws(() => parseRelationalTopology(selfEdge), /self-edges/);

    const duplicateOwner = copy();
    byId(duplicateOwner.nodes, "research-fund-account").owns = ["account-balances"];
    assert.throws(() => parseRelationalTopology(duplicateOwner), /owned by both/);

    const unordered = copy();
    [unordered.nodes[0], unordered.nodes[1]] = [unordered.nodes[1], unordered.nodes[0]];
    assert.throws(() => parseRelationalTopology(unordered), /strictly sorted by id/);
  });

  it("enforces typed flow locality, provenance, and authority origins", () => {
    const economicEscape = copy();
    byId(economicEscape.edges, "staking-panel-status").flows = ["ECONOMIC"];
    assert.throws(() => parseRelationalTopology(economicEscape), /remain inside the economy plane/);

    const falseAuthority = copy();
    byId(falseAuthority.edges, "panel-knowledge-verdict").flows = ["AUTHORITY"];
    assert.throws(() => parseRelationalTopology(falseAuthority), /AUTHORITY must originate/);

    const falseEmergency = copy();
    byId(falseEmergency.edges, "governance-electorate-policy").flows = ["EMERGENCY"];
    assert.throws(() => parseRelationalTopology(falseEmergency), /EMERGENCY must originate/);

    const borrowedProvenance = copy();
    byId(borrowedProvenance.principles, "voluntary-relation").sourceRefs = [
      "authoritative-state",
    ];
    assert.throws(() => parseRelationalTopology(borrowedProvenance), /must not borrow/);
  });

  it("rejects authority cycles and declared same-flow forbidden routes", () => {
    const authorityCycle = copy();
    byId(authorityCycle.edges, "emergency-governance-cancellation").flows = [
      "AUTHORITY",
    ];
    assert.throws(() => parseRelationalTopology(authorityCycle), /AUTHORITY flow contains a cycle/);

    const forbiddenRoute = copy();
    byId(forbiddenRoute.edges, "qualification-panel-eligibility").flows = [
      "AUTHORITY",
    ];
    assert.throws(
      () => parseRelationalTopology(forbiddenRoute),
      /AUTHORITY flow reaches verification-panel from sdk-gov/,
    );

    const topology = parseRelationalTopologyJson(canonicalRaw);
    assert.equal(
      hasRelationalFlowPath(topology.edges, "AUTHORITY", "sdk-gov", "x-qualification"),
      true,
    );
    assert.equal(
      hasRelationalFlowPath(topology.edges, "AUTHORITY", "sdk-gov", "verification-panel"),
      false,
    );
  });

  it("fetches only exact same-origin, digest-pinned JSON", async () => {
    const calls: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
    const topology = await fetchRelationalTopology({
      baseUrl: "https://zerone.ai/#relations",
      fetcher: async (input, init) => {
        calls.push({ input, init });
        return jsonResponse();
      },
    });
    assert.equal(topology.nodes.length, 17);
    assert.equal(calls.length, 1);
    assert.equal(calls[0]?.input, RELATIONAL_TOPOLOGY_ENDPOINT);
    assert.equal(calls[0]?.init?.cache, "no-store");
    assert.equal(calls[0]?.init?.credentials, "omit");
    assert.equal(calls[0]?.init?.redirect, "error");
    assert.equal(calls[0]?.init?.referrerPolicy, "no-referrer");

    await assert.rejects(
      fetchRelationalTopology({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => jsonResponse(`${canonicalRaw}\n`),
      }),
      /do not match the reviewed SHA-256/,
    );
    await assert.rejects(
      fetchRelationalTopology({
        baseUrl: "https://zerone.ai/",
        fetcher: async () =>
          jsonResponse(canonicalRaw, {}, "https://example.org/relational-topology.v0.json"),
      }),
      /exact same-origin path/,
    );
    await assert.rejects(
      fetchRelationalTopology({
        baseUrl: "https://zerone.ai/",
        fetcher: async () =>
          jsonResponse(canonicalRaw, { headers: { "content-type": "text/html" } }),
      }),
      /non-application\/json/,
    );
    await assert.rejects(
      fetchRelationalTopology({
        baseUrl: "https://zerone.ai/",
        fetcher: async () =>
          jsonResponse(canonicalRaw, {
            headers: { "content-length": String(RELATIONAL_TOPOLOGY_MAX_BYTES + 1) },
          }),
      }),
      /byte limit/,
    );
  });

  it("times out a fetch that ignores AbortSignal", async () => {
    await assert.rejects(
      settlesWithin(
        fetchRelationalTopology({
          baseUrl: "https://zerone.ai/",
          timeoutMs: 5,
          fetcher: async () => await new Promise<Response>(() => undefined),
        }),
        250,
      ),
      /timed out/,
    );
  });

  it("renders an accessible synchronized graph with a complete text fallback", async () => {
    const topology = parseRelationalTopologyJson(canonicalRaw);
    await withFakeDocument(() => {
      const root = new FakeElement("div");
      renderRelationalTopology(root as unknown as HTMLElement, topology);
      const all = descendants(root);
      const graph = all.find(
        (node) => node.className === "relational-topology-graph",
      );
      assert.equal(graph?.getAttribute("role"), "region");
      assert.equal(graph?.tabIndex, 0);
      assert.equal(root.getAttribute("aria-busy"), "false");
      assert.equal(
        all.filter((node) => node.className === "relational-topology-node").length,
        17,
      );
      assert.equal(
        all.filter((node) => node.className === "relational-topology-edge").length,
        30,
      );
      assert.equal(
        all.filter((node) => node.className === "relational-topology-principle").length,
        6,
      );
      assert.equal(
        all.filter((node) => node.className === "relational-topology-conflict").length,
        6,
      );
      const nodeButton = all.find(
        (node) =>
          node.className === "relational-topology-node" &&
          flattenedText(node).includes("SDK governance"),
      );
      assert.equal(nodeButton?.tagName, "button");
      nodeButton?.click();
      assert.equal(nodeButton?.getAttribute("aria-pressed"), "true");
      const visibleEdges = all.filter(
        (node) => node.tagName === "li" && !node.hidden,
      );
      assert.ok(visibleEdges.length > 0 && visibleEdges.length < 30);
      const copyText = flattenedText(root);
      assert.match(copyText, /STATIC PROJECTION · TARGET NOT LIVE/);
      assert.match(copyText, /Connection without capture/);
      assert.match(copyText, /Known source conflicts remain visible/);
      assert.match(copyText, /same-flow reachability guards/);
      assert.match(copyText, /no consensus, network observation, authority/i);
      assert.ok(
        all.some(
          (node) => node.tagName === "a" && node.href === RELATIONAL_TOPOLOGY_ENDPOINT,
        ),
      );
    });

    const source = readFileSync(new URL("../src/relational-topology.ts", import.meta.url), "utf8");
    const main = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");
    const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
    assert.doesNotMatch(source, /\.innerHTML\s*=/u);
    assert.match(
      main,
      /initialiseRelationalTopology\(\s*relationalTopologyRoot,?\s*\)/,
    );
    assert.match(html, /href="#relations"/);
    assert.match(html, /id="relational-topology-root"/);

    await withFakeDocument(async () => {
      const descriptor = Object.getOwnPropertyDescriptor(globalThis, "fetch");
      Object.defineProperty(globalThis, "fetch", {
        configurable: true,
        value: async () => jsonResponse("unavailable", { status: 503 }),
      });
      try {
        const root = new FakeElement("div");
        await initialiseRelationalTopology(root as unknown as HTMLElement);
        assert.equal(root.getAttribute("aria-busy"), "false");
        const all = descendants(root);
        assert.ok(all.some((node) => node.getAttribute("role") === "alert"));
        assert.ok(
          all.some(
            (node) => node.tagName === "a" && node.href === RELATIONAL_TOPOLOGY_ENDPOINT,
          ),
        );
      } finally {
        if (descriptor === undefined) delete (globalThis as { fetch?: unknown }).fetch;
        else Object.defineProperty(globalThis, "fetch", descriptor);
      }
    });
  });
});
