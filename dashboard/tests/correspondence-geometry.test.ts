import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  CORRESPONDENCE_GEOMETRY_ENDPOINT,
  CORRESPONDENCE_GEOMETRY_MAX_BYTES,
  CORRESPONDENCE_GEOMETRY_SHA256,
  fetchCorrespondenceGeometry,
  initialiseCorrespondenceGeometry,
  parseCorrespondenceGeometry,
  parseCorrespondenceGeometryJson,
  renderCorrespondenceGeometry,
} from "../src/correspondence-geometry";
import { validateCorrespondenceGeometryRaw } from "../scripts/validate-correspondence-geometry";

type MutableDocument = Record<string, any>;

const canonicalRaw = readFileSync(
  new URL("../public/standards/correspondence-geometry.v0.json", import.meta.url),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as MutableDocument;

function copy(): MutableDocument {
  return structuredClone(canonical) as MutableDocument;
}

function byId(values: MutableDocument[], id: string): MutableDocument {
  const found = values.find((value) => value.id === id);
  assert.ok(found, `missing canonical fixture ${id}`);
  return found;
}

function pinnedSourceBytes(
  overrides: ReadonlyMap<string, string> = new Map(),
): ReadonlyMap<string, string> {
  return new Map(
    canonical.sourceBindings.map((binding: MutableDocument) => [
      binding.path as string,
      overrides.get(binding.path as string) ??
        readFileSync(new URL(`../../${binding.path as string}`, import.meta.url), "utf8"),
    ]),
  );
}

function jsonResponse(
  body: BodyInit | null = canonicalRaw,
  init: ResponseInit = {},
  url = `https://zerone.ai${CORRESPONDENCE_GEOMETRY_ENDPOINT}`,
  redirected = false,
): Response {
  const headers = new Headers(init.headers);
  if (!headers.has("content-type")) {
    headers.set("content-type", "application/json; charset=utf-8");
  }
  const response = new Response(body, { ...init, headers });
  Object.defineProperty(response, "url", { value: url });
  Object.defineProperty(response, "redirected", { value: redirected });
  return response;
}

async function settlesWithin<T>(
  promise: Promise<T>,
  milliseconds: number,
): Promise<T> {
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

  add(...tokens: string[]): void {
    const names = new Set(this.element.className.split(/\s+/u).filter(Boolean));
    for (const token of tokens) names.add(token);
    this.element.className = [...names].join(" ");
  }

  remove(...tokens: string[]): void {
    const names = new Set(this.element.className.split(/\s+/u).filter(Boolean));
    for (const token of tokens) names.delete(token);
    this.element.className = [...names].join(" ");
  }

  toggle(token: string, force?: boolean): boolean {
    const names = new Set(this.element.className.split(/\s+/u).filter(Boolean));
    const present = force ?? !names.has(token);
    if (present) names.add(token);
    else names.delete(token);
    this.element.className = [...names].join(" ");
    return present;
  }

  contains(token: string): boolean {
    return this.element.className.split(/\s+/u).includes(token);
  }
}

class FakeElement {
  readonly children: FakeElement[] = [];
  readonly attributes = new Map<string, string>();
  readonly dataset: Record<string, string> = {};
  readonly listeners = new Map<
    string,
    Array<(event: Record<string, unknown>) => unknown>
  >();
  readonly classList = new FakeClassList(this);
  className = "";
  href = "";
  target = "";
  rel = "";
  type = "";
  title = "";
  hidden = false;
  disabled = false;
  tabIndex = 0;
  private text: string | null = null;

  constructor(readonly tagName: string) {}

  get textContent(): string | null {
    return this.text;
  }

  set textContent(value: string | null) {
    this.text = value;
    this.children.splice(0);
  }

  set innerHTML(_value: string) {
    throw new Error("correspondence renderer must never use innerHTML");
  }

  get firstElementChild(): FakeElement | null {
    return this.children[0] ?? null;
  }

  append(...nodes: FakeElement[]): void {
    this.children.push(...nodes);
  }

  replaceChildren(...nodes: FakeElement[]): void {
    this.children.splice(0, this.children.length, ...nodes);
    this.text = null;
  }

  setAttribute(name: string, value: string): void {
    this.attributes.set(name, value);
  }

  getAttribute(name: string): string | null {
    return this.attributes.get(name) ?? null;
  }

  removeAttribute(name: string): void {
    this.attributes.delete(name);
  }

  addEventListener(
    name: string,
    listener: (event: Record<string, unknown>) => unknown,
  ): void {
    const listeners = this.listeners.get(name) ?? [];
    listeners.push(listener);
    this.listeners.set(name, listeners);
  }

  async dispatch(
    name: string,
    event: Record<string, unknown> = {},
  ): Promise<void> {
    for (const listener of this.listeners.get(name) ?? []) {
      await listener(event);
    }
  }

  async click(): Promise<void> {
    await this.dispatch("click");
  }
}

function descendants(node: FakeElement): FakeElement[] {
  return [node, ...node.children.flatMap(descendants)];
}

function flattenedText(node: FakeElement): string {
  return [node.textContent ?? "", ...node.children.map(flattenedText)].join(" ");
}

function hasClass(node: FakeElement, className: string): boolean {
  return node.className.split(/\s+/u).includes(className);
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
    if (descriptor === undefined) {
      delete (globalThis as { document?: unknown }).document;
    } else {
      Object.defineProperty(globalThis, "document", descriptor);
    }
  }
}

describe("correspondence geometry v0 parser and seal", () => {
  it("pins the exact zero-effect atlas, source set, and reviewed counts", () => {
    const geometry = parseCorrespondenceGeometryJson(canonicalRaw);

    assert.equal(geometry.schema, "zerone.correspondence-geometry/v0");
    assert.equal(geometry.version, 0);
    assert.equal(geometry.status, "READ_ONLY_ZERO_EFFECT");
    assert.equal(geometry.epistemicLanes.length, 5);
    assert.equal(geometry.relationKinds.length, 4);
    assert.equal(geometry.dimensions.length, 4);
    assert.equal(geometry.correspondences.length, 7);
    assert.equal(geometry.physicsSources.length, 5);
    assert.equal(geometry.sourceBindings.length, 7);
    assert.equal(geometry.energyFirewall.lanes.length, 6);
    assert.equal(geometry.dualityGate.reviewedCandidateCount, 0);
    assert.equal(geometry.dualityGate.acceptedCandidateCount, 0);
    assert.equal(
      geometry.correspondences.filter(
        ({ relationKind }) => relationKind === "DUALITY_CANDIDATE",
      ).length,
      0,
    );
    assert.ok(Object.values(geometry.releaseBoundary).every((value) => value === false));
    assert.ok(
      geometry.energyFirewall.lanes.every(({ mayConvertTo }) => mayConvertTo.length === 0),
    );
    assert.deepEqual(
      geometry.physicsSources.map(({ id, title, url }) => [id, title, url]),
      [
        [
          "ads-cft",
          "The Large N Limit of Superconformal Field Theories and Supergravity",
          "https://arxiv.org/abs/hep-th/9711200v3",
        ],
        [
          "compactification",
          "Strong Coupling Expansion Of Calabi-Yau Compactification",
          "https://arxiv.org/abs/hep-th/9602070v2",
        ],
        [
          "entanglement-spacetime",
          "Building up spacetime with quantum entanglement",
          "https://arxiv.org/abs/1005.3035v2",
        ],
        [
          "holographic-entanglement",
          "Holographic Derivation of Entanglement Entropy from AdS/CFT",
          "https://arxiv.org/abs/hep-th/0603001v2",
        ],
        [
          "string-duality",
          "String Theory Dynamics in Various Dimensions",
          "https://arxiv.org/abs/hep-th/9503124v2",
        ],
      ],
    );
    const physicsSourceIds = new Set(geometry.physicsSources.map(({ id }) => id));
    assert.ok(
      geometry.correspondences
        .filter(({ sourceLane }) => sourceLane === "PHYSICS_MATH")
        .every(({ sourceRefs }) => sourceRefs.some((id) => physicsSourceIds.has(id))),
    );
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      CORRESPONDENCE_GEOMETRY_SHA256,
    );
    assert.ok(Buffer.byteLength(canonicalRaw, "utf8") < CORRESPONDENCE_GEOMETRY_MAX_BYTES);

    const actualPins = new Map(
      [...pinnedSourceBytes()].map(([path, bytes]) => [
        path,
        createHash("sha256").update(bytes).digest("hex"),
      ]),
    );
    assert.deepEqual(
      geometry.sourceBindings.map(({ path, sha256 }) => [path, sha256]),
      [...actualPins],
    );
  });

  it("validates the sealed bytes and every local source without network access", () => {
    const summary = validateCorrespondenceGeometryRaw(
      canonicalRaw,
      pinnedSourceBytes(),
    );

    assert.equal(summary.epistemicLaneCount, 5);
    assert.equal(summary.relationKindCount, 4);
    assert.equal(summary.dimensionCount, 4);
    assert.equal(summary.correspondenceCount, 7);
    assert.equal(summary.dualityCandidateCount, 0);
    assert.equal(summary.acceptedDualityCount, 0);
    assert.equal(summary.releaseEffectCount, 0);
    assert.equal(summary.sourceBindingCount, 7);
    assert.equal(summary.physicsSourceCount, 5);
    assert.equal(summary.manifestSha256, CORRESPONDENCE_GEOMETRY_SHA256);

    const first = canonical.sourceBindings[0] as MutableDocument;
    assert.throws(
      () =>
        validateCorrespondenceGeometryRaw(
          canonicalRaw,
          pinnedSourceBytes(new Map([[first.path as string, "drifted source bytes"]])),
        ),
      /source.*drift|SHA-256|digest/i,
    );
    assert.throws(
      () =>
        validateCorrespondenceGeometryRaw(
          canonicalRaw,
          new Map([...pinnedSourceBytes(), ["docs/unreviewed.md", "unreviewed"]]),
        ),
      /unreviewed|source.*set|count/i,
    );
    assert.throws(
      () => validateCorrespondenceGeometryRaw(`${canonicalRaw}\n`, pinnedSourceBytes()),
      /document digest|SHA-256|runtime pin/i,
    );
  });

  it("rejects unknown fields at every reviewed shape and decoded duplicate keys", () => {
    const mutations: Array<(document: MutableDocument) => void> = [
      (document) => { document.unreviewed = true; },
      (document) => { document.sourceBindings[0].unreviewed = true; },
      (document) => { document.physicsSources[0].unreviewed = true; },
      (document) => { document.epistemicLanes[0].unreviewed = true; },
      (document) => { document.relationKinds[0].unreviewed = true; },
      (document) => { document.dimensions[0].unreviewed = true; },
      (document) => { document.correspondences[0].unreviewed = true; },
      (document) => { document.correspondences[0].background.unreviewed = true; },
      (document) => { document.correspondences[0].preservedInvariants[0].unreviewed = true; },
      (document) => { document.correspondences[0].informationLosses[0].unreviewed = true; },
      (document) => { document.correspondences[0].counterexamples[0].unreviewed = true; },
      (document) => {
        byId(
          document.correspondences,
          "fact-energy-to-metabolism-language",
        ).roundTripTests[0].unreviewed = true;
      },
      (document) => { document.correspondences[0].nonTransfers[0].unreviewed = true; },
      (document) => { document.energyFirewall.unreviewed = true; },
      (document) => { document.energyFirewall.lanes[0].unreviewed = true; },
      (document) => { document.dualityGate.unreviewed = true; },
      (document) => { document.releaseBoundary.unreviewed = true; },
    ];
    for (const mutate of mutations) {
      const document = copy();
      mutate(document);
      assert.throws(() => parseCorrespondenceGeometry(document), /unknown|fields|keys/i);
    }

    const duplicate = canonicalRaw.replace(
      '"schema": "zerone.correspondence-geometry/v0",',
      '"\\u0073chema": "zerone.correspondence-geometry/v0",\n  "schema": "zerone.correspondence-geometry/v0",',
    );
    assert.notEqual(duplicate, canonicalRaw);
    assert.throws(
      () => parseCorrespondenceGeometryJson(duplicate),
      /duplicate JSON.*key|duplicate.*schema/i,
    );
  });

  it("bounds UTF-8 document bytes and JSON nesting depth", () => {
    const bytes = new TextEncoder().encode(canonicalRaw).byteLength;
    const atLimit = canonicalRaw + " ".repeat(CORRESPONDENCE_GEOMETRY_MAX_BYTES - bytes);
    assert.equal(new TextEncoder().encode(atLimit).byteLength, CORRESPONDENCE_GEOMETRY_MAX_BYTES);
    assert.equal(parseCorrespondenceGeometryJson(atLimit).correspondences.length, 7);
    assert.throws(
      () => parseCorrespondenceGeometryJson(`${atLimit} `),
      /byte limit|exceeds|too large/i,
    );

    const tooDeep = `${"[".repeat(100)}0${"]".repeat(100)}`;
    assert.throws(
      () => parseCorrespondenceGeometryJson(tooDeep),
      /depth|nesting/i,
    );
  });

  it("keeps identifiers, references, and reviewed arrays deterministic", () => {
    const unknownLane = copy();
    unknownLane.correspondences[0].sourceLane = "UNREVIEWED";
    assert.throws(() => parseCorrespondenceGeometry(unknownLane), /sourceLane|unknown/i);

    const unknownDimension = copy();
    unknownDimension.correspondences[0].dimension = "metaverse";
    assert.throws(() => parseCorrespondenceGeometry(unknownDimension), /dimension|unknown/i);

    const unknownSource = copy();
    unknownSource.correspondences[0].sourceRefs = ["missing-source"];
    assert.throws(() => parseCorrespondenceGeometry(unknownSource), /sourceRefs|unknown/i);

    const physicsWithoutPaper = copy();
    const physicsIds = new Set(
      physicsWithoutPaper.physicsSources.map(({ id }: MutableDocument) => id),
    );
    const physicsMapping = byId(
      physicsWithoutPaper.correspondences,
      "duality-to-bounded-reformulation",
    );
    physicsMapping.sourceRefs = physicsMapping.sourceRefs.filter(
      (id: string) => !physicsIds.has(id),
    );
    assert.throws(
      () => parseCorrespondenceGeometry(physicsWithoutPaper),
      /PHYSICS_MATH.*physics|physics.*sourceRef|primary.*source/i,
    );

    const unsafeSourcePath = copy();
    unsafeSourcePath.sourceBindings[0].path = "docs/../private.txt";
    assert.throws(
      () => parseCorrespondenceGeometry(unsafeSourcePath),
      /repository path|safe.*path|path.*safe/i,
    );

    const duplicate = copy();
    duplicate.correspondences[1].id = duplicate.correspondences[0].id;
    assert.throws(() => parseCorrespondenceGeometry(duplicate), /duplicate|unique|sorted/i);

    const unordered = copy();
    [unordered.correspondences[0], unordered.correspondences[1]] = [
      unordered.correspondences[1],
      unordered.correspondences[0],
    ];
    assert.throws(() => parseCorrespondenceGeometry(unordered), /sorted|order/i);
  });

  it("rejects every attempted release effect", () => {
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const document = copy();
      document.releaseBoundary[key] = true;
      assert.throws(
        () => parseCorrespondenceGeometry(document),
        new RegExp(key),
      );
    }
  });
});

describe("correspondence relation and resource gates", () => {
  it("keeps analogies one-way and outside equivalence", () => {
    const inverse = copy();
    const analogy = byId(
      inverse.correspondences,
      "duality-to-bounded-reformulation",
    );
    analogy.inverseMap = "silently reverse the resemblance";
    assert.throws(
      () => parseCorrespondenceGeometry(inverse),
      /ANALOGY.*inverseMap|inverseMap.*ANALOGY/i,
    );

    const equivalence = copy();
    byId(
      equivalence.correspondences,
      "duality-to-bounded-reformulation",
    ).equivalenceScope = "all meanings";
    assert.throws(
      () => parseCorrespondenceGeometry(equivalence),
      /ANALOGY.*equivalenceScope|equivalenceScope.*ANALOGY/i,
    );

    const falseRoundTrip = copy();
    byId(
      falseRoundTrip.correspondences,
      "duality-to-bounded-reformulation",
    ).roundTripTests = [
      structuredClone(
        byId(
          falseRoundTrip.correspondences,
          "fact-energy-to-metabolism-language",
        ).roundTripTests[0],
      ),
    ];
    assert.throws(
      () => parseCorrespondenceGeometry(falseRoundTrip),
      /ANALOGY.*round.trip|round.trip.*ANALOGY|one-way/i,
    );
  });

  it("requires translations to expose an inverse and both unrun proposed round trips", () => {
    const noInverse = copy();
    byId(
      noInverse.correspondences,
      "fact-energy-to-metabolism-language",
    ).inverseMap = null;
    assert.throws(
      () => parseCorrespondenceGeometry(noInverse),
      /TRANSLATION.*inverseMap|inverseMap.*TRANSLATION/i,
    );

    const oneDirection = copy();
    byId(
      oneDirection.correspondences,
      "fact-energy-to-metabolism-language",
    ).roundTripTests.pop();
    assert.throws(
      () => parseCorrespondenceGeometry(oneDirection),
      /TRANSLATION.*round.trip|both.*direction|TARGET_SOURCE_TARGET/i,
    );

    const prematurePass = copy();
    byId(
      prematurePass.correspondences,
      "fact-energy-to-metabolism-language",
    ).roundTripTests[0].observed = "PASS";
    assert.throws(
      () => parseCorrespondenceGeometry(prematurePass),
      /PROPOSED.*NOT_RUN|NOT_RUN.*PROPOSED/i,
    );
  });

  it("requires a projection to name its in-scope information loss", () => {
    const document = copy();
    byId(
      document.correspondences,
      "religious-witness-to-public-projection",
    ).informationLosses[0].scope = "OUTSIDE_TRANSFER_SCOPE";
    assert.throws(
      () => parseCorrespondenceGeometry(document),
      /PROJECTION.*IN_SCOPE|IN_SCOPE.*PROJECTION/i,
    );

    const falseRoundTrip = copy();
    byId(
      falseRoundTrip.correspondences,
      "religious-witness-to-public-projection",
    ).roundTripTests = [
      structuredClone(
        byId(
          falseRoundTrip.correspondences,
          "fact-energy-to-metabolism-language",
        ).roundTripTests[0],
      ),
    ];
    assert.throws(
      () => parseCorrespondenceGeometry(falseRoundTrip),
      /PROJECTION.*round.trip|round.trip.*PROJECTION|one-way/i,
    );
  });

  it("enforces every duality-candidate gate before equivalence can be accepted", () => {
    const promoted = (): MutableDocument => {
      const document = copy();
      const candidate = byId(
        document.correspondences,
        "fact-energy-to-metabolism-language",
      );
      candidate.relationKind = "DUALITY_CANDIDATE";
      candidate.assessment = "TESTED_WITHIN_SCOPE";
      candidate.equivalenceScope = "exact versioned field-name vocabulary only";
      candidate.preservedInvariants[0].mode = "EXACT";
      for (const roundTrip of candidate.roundTripTests) {
        roundTrip.observed = "PASS";
      }
      document.dualityGate.label = "EQUIVALENCE_CANDIDATE_ACCEPTED";
      document.dualityGate.reviewedCandidateCount = 1;
      document.dualityGate.acceptedCandidateCount = 1;
      return document;
    };

    assert.doesNotThrow(() => parseCorrespondenceGeometry(promoted()));

    const mutations: Array<{
      mutate(candidate: MutableDocument): void;
      error: RegExp;
    }> = [
      {
        mutate: (candidate) => { candidate.equivalenceScope = null; },
        error: /equivalenceScope/i,
      },
      {
        mutate: (candidate) => { candidate.inverseMap = null; },
        error: /inverseMap/i,
      },
      {
        mutate: (candidate) => { candidate.preservedInvariants[0].mode = "STRUCTURAL"; },
        error: /EXACT|TOLERANCED|invariant/i,
      },
      {
        mutate: (candidate) => { candidate.informationLosses[0].scope = "IN_SCOPE"; },
        error: /IN_SCOPE.*loss|loss.*IN_SCOPE/i,
      },
      {
        mutate: (candidate) => { candidate.roundTripTests[0].observed = "NOT_RUN"; },
        error: /round.trip|observed PASS/i,
      },
      {
        mutate: (candidate) => { candidate.roundTripTests.pop(); },
        error: /both.*direction|TARGET_SOURCE_TARGET|round.trip/i,
      },
    ];
    for (const { mutate, error } of mutations) {
      const document = promoted();
      mutate(byId(document.correspondences, "fact-energy-to-metabolism-language"));
      assert.throws(() => parseCorrespondenceGeometry(document), error);
    }
  });

  it("keeps all six energy registers non-convertible and non-authoritative", () => {
    for (const key of [
      "implicitConversion",
      "truthFromResource",
      "authorityFromResource",
      "personWorthFromResource",
      "restPenalty",
    ]) {
      const document = copy();
      document.energyFirewall[key] = true;
      assert.throws(() => parseCorrespondenceGeometry(document), new RegExp(key));
    }

    const conversion = copy();
    byId(conversion.energyFirewall.lanes, "PHYSICAL_ENERGY").mayConvertTo = [
      "PROTOCOL_METABOLISM",
    ];
    assert.throws(
      () => parseCorrespondenceGeometry(conversion),
      /mayConvertTo|conversion/i,
    );

    const registerCollapse = copy();
    byId(registerCollapse.energyFirewall.lanes, "ECONOMIC").measureOrRegister =
      byId(
        registerCollapse.energyFirewall.lanes,
        "PHYSICAL_ENERGY",
      ).measureOrRegister;
    assert.throws(
      () => parseCorrespondenceGeometry(registerCollapse),
      /distinct.*measure|measure.*register|register.*distinct/i,
    );
  });
});

describe("correspondence geometry bounded fetch", () => {
  it("makes exactly one bounded request and accepts only the exact same-origin artifact", async () => {
    const calls: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
    const geometry = await fetchCorrespondenceGeometry({
      baseUrl: "https://zerone.ai/dashboard?ignored=true#correspondence",
      fetcher: async (input, init) => {
        calls.push({ input, init });
        return jsonResponse();
      },
    });

    assert.equal(geometry.correspondences.length, 7);
    assert.equal(calls.length, 1);
    assert.equal(String(calls[0]?.input), CORRESPONDENCE_GEOMETRY_ENDPOINT);
    assert.equal(new Headers(calls[0]?.init?.headers).get("accept"), "application/json");
    assert.equal(calls[0]?.init?.cache, "no-store");
    assert.equal(calls[0]?.init?.credentials, "omit");
    assert.ok(["error", "manual"].includes(calls[0]?.init?.redirect ?? ""));
    assert.equal(calls[0]?.init?.referrerPolicy, "no-referrer");
    assert.ok(calls[0]?.init?.signal instanceof AbortSignal);

    await assert.rejects(
      fetchCorrespondenceGeometry({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => jsonResponse(canonicalRaw, {}, undefined, true),
      }),
      /redirect/i,
    );
    for (const url of [
      "https://attacker.example/standards/correspondence-geometry.v0.json",
      "https://zerone.ai/standards/other.json",
      `https://zerone.ai${CORRESPONDENCE_GEOMETRY_ENDPOINT}?unreviewed=1`,
      `https://zerone.ai${CORRESPONDENCE_GEOMETRY_ENDPOINT}#unreviewed`,
    ]) {
      await assert.rejects(
        fetchCorrespondenceGeometry({
          baseUrl: "https://zerone.ai/",
          fetcher: async () => jsonResponse(canonicalRaw, {}, url),
        }),
        /same-origin|exact.*path|response URL/i,
        url,
      );
    }
  });

  it("rejects HTTP, media-type, digest, declared-size, streamed-size, and UTF-8 drift", async () => {
    await assert.rejects(
      fetchCorrespondenceGeometry({
        fetcher: async () => jsonResponse("{}", { status: 503 }),
      }),
      /HTTP 503/i,
    );
    await assert.rejects(
      fetchCorrespondenceGeometry({
        fetcher: async () =>
          jsonResponse(canonicalRaw, { headers: { "content-type": "text/html" } }),
      }),
      /application\/json|content.type/i,
    );
    await assert.rejects(
      fetchCorrespondenceGeometry({
        fetcher: async () => jsonResponse(`${canonicalRaw}\n`),
      }),
      /SHA-256|digest|reviewed/i,
    );
    await assert.rejects(
      fetchCorrespondenceGeometry({
        fetcher: async () =>
          jsonResponse("{}", {
            headers: {
              "content-length": String(CORRESPONDENCE_GEOMETRY_MAX_BYTES + 1),
            },
          }),
      }),
      /byte limit|exceeds|too large/i,
    );

    const oversized = new Uint8Array(CORRESPONDENCE_GEOMETRY_MAX_BYTES + 1).fill(0x20);
    await assert.rejects(
      fetchCorrespondenceGeometry({
        fetcher: async () =>
          jsonResponse(
            new ReadableStream<Uint8Array>({
              start(controller) {
                controller.enqueue(oversized);
                controller.close();
              },
            }),
          ),
      }),
      /byte limit|exceeds|too large/i,
    );
    await assert.rejects(
      fetchCorrespondenceGeometry({
        fetcher: async () => jsonResponse(new Uint8Array([0xff])),
      }),
      /UTF-8/i,
    );
  });

  it("settles when a fetcher ignores its AbortSignal", async () => {
    await assert.rejects(
      settlesWithin(
        fetchCorrespondenceGeometry({
          timeoutMs: 5,
          fetcher: async () => await new Promise<Response>(() => undefined),
        }),
        250,
      ),
      /timed out|timeout/i,
    );
  });

  it("settles when a response body never yields bytes", async () => {
    await assert.rejects(
      settlesWithin(
        fetchCorrespondenceGeometry({
          timeoutMs: 5,
          fetcher: async () =>
            jsonResponse(
              new ReadableStream<Uint8Array>({
                start() {
                  // Deliberately never enqueue or close.
                },
              }),
            ),
        }),
        250,
      ),
      /timed out|timeout/i,
    );
  });
});

describe("correspondence geometry renderer and dashboard integration", () => {
  it("renders data only as text and filters cards without losing mapping detail", async () => {
    await withFakeDocument(async () => {
      const geometry = parseCorrespondenceGeometryJson(canonicalRaw);
      const hostile = '<img src=x onerror="globalThis.pwned=true"> & witness';
      const firstMapping = geometry.correspondences[0];
      assert.ok(firstMapping);
      firstMapping.title = hostile;
      const root = new FakeElement("div");
      renderCorrespondenceGeometry(root as unknown as HTMLElement, geometry);

      assert.equal(root.getAttribute("aria-busy"), "false");
      const all = descendants(root);
      assert.equal(all.filter(({ tagName }) => tagName === "img").length, 0);
      assert.equal(all.filter(({ tagName }) => tagName === "script").length, 0);
      assert.ok(flattenedText(root).includes(hostile));
      assert.equal(
        all.filter((node) => hasClass(node, "correspondence-geometry-lane")).length,
        5,
      );
      const cards = all.filter(
        (node) => hasClass(node, "correspondence-geometry-mapping"),
      );
      assert.equal(cards.length, 7);
      assert.equal(cards.filter(({ hidden }) => !hidden).length, 7);
      assert.match(flattenedText(root), /Release boundary · every value false/i);
      assert.match(flattenedText(root), /No implicit conversion/i);
      assert.match(flattenedText(root), /does not endorse|do not endorse/i);
      assert.ok(
        all.some(
          ({ tagName, href }) =>
            tagName === "a" && href === CORRESPONDENCE_GEOMETRY_ENDPOINT,
        ),
      );

      const filters = all.filter(
        (node) => hasClass(node, "correspondence-geometry-filter"),
      );
      assert.equal(filters.length, 5);
      const religion = filters.find((filter) => /religion/i.test(flattenedText(filter)));
      assert.ok(religion);
      assert.equal(religion.getAttribute("aria-pressed"), "false");
      await religion.click();
      assert.equal(religion.getAttribute("aria-pressed"), "true");
      assert.equal(cards.filter(({ hidden }) => !hidden).length, 1);
      assert.ok(
        cards
          .filter(({ hidden }) => !hidden)
          .every(({ dataset }) => dataset.dimension === "religion"),
      );

      const allFilter = filters.find((filter) =>
        /^all(?: dimensions)?$/iu.test(flattenedText(filter).trim()),
      );
      assert.ok(allFilter);
      await allFilter.click();
      assert.equal(cards.filter(({ hidden }) => !hidden).length, 7);
    });

    const runtime = readFileSync(
      new URL("../src/correspondence-geometry.ts", import.meta.url),
      "utf8",
    );
    assert.doesNotMatch(
      runtime,
      /\.innerHTML\b|\.outerHTML\b|insertAdjacentHTML|document\.write|DOMParser/u,
    );
  });

  it("renders a complete static refusal when the bounded read fails", async () => {
    await withFakeDocument(async () => {
      const descriptor = Object.getOwnPropertyDescriptor(globalThis, "fetch");
      Object.defineProperty(globalThis, "fetch", {
        configurable: true,
        value: async () => jsonResponse("unavailable", { status: 503 }),
      });
      try {
        const root = new FakeElement("div");
        await initialiseCorrespondenceGeometry(root as unknown as HTMLElement);

        assert.equal(root.getAttribute("aria-busy"), "false");
        const all = descendants(root);
        assert.ok(all.some((node) => node.getAttribute("role") === "alert"));
        assert.ok(
          all.some(
            ({ tagName, href }) =>
              tagName === "a" && href === CORRESPONDENCE_GEOMETRY_ENDPOINT,
          ),
        );
        assert.match(flattenedText(root), /read-only|no effect|raw|refus/i);
      } finally {
        if (descriptor === undefined) {
          delete (globalThis as { fetch?: unknown }).fetch;
        } else {
          Object.defineProperty(globalThis, "fetch", descriptor);
        }
      }
    });
  });

  it("ships a substantive no-JS summary and wires direct-hash alignment", () => {
    const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
    const main = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");
    const noScriptMarker = '<div class="correspondence-geometry-noscript">';
    const noScriptStart = html.indexOf(noScriptMarker);
    const noScriptEnd = html.indexOf("</noscript>", noScriptStart);
    const noScript =
      noScriptStart >= 0 && noScriptEnd > noScriptStart
        ? html.slice(noScriptStart, noScriptEnd)
        : undefined;

    assert.ok(noScript, "missing complete Correspondence Geometry no-JS account");
    assert.equal(
      noScript.match(/class="correspondence-geometry-mapping"/gu)?.length,
      7,
    );
    assert.match(noScript, /Five lanes remain distinct/);
    assert.match(noScript, /Six energy-language lanes\. No implicit conversion/);
    assert.match(noScript, /Zero equivalences claimed/);
    assert.match(noScript, /Twenty disabled effects/);
    assert.match(noScript, /performsNetworkWrites<\/code><b>false/);
    assert.match(noScript, /href="\/standards\/correspondence-geometry\.v0\.json"/);
    for (const { url } of canonical.physicsSources as MutableDocument[]) {
      assert.ok(noScript.includes(`href="${url as string}"`), url as string);
    }
    assert.match(noScript, /Strong coupling expansion of Calabi.Yau compactification/i);
    assert.match(html, /href="#correspondence">Mappings<\/a>/);
    assert.match(html, /id="correspondence-geometry-root"/);

    assert.match(
      main,
      /initialiseCorrespondenceGeometry\(\s*correspondenceGeometryRoot,?\s*\)/,
    );
    assert.match(main, /window\.location\.hash !== "#correspondence"/);
    assert.match(
      main,
      /window\.location\.hash === "#correspondence"[\s\S]*?#correspondence/,
    );
    assert.match(
      main,
      /Promise\.allSettled\(\[[\s\S]*?correspondenceGeometryReady,[\s\S]*?initialNetworkReady,[\s\S]*?\]\)\.then\(\(\) => \{\s*initialHashInputsSettled = true;/,
    );
  });
});
