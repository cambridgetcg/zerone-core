import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  KNOWLEDGE_GEOMETRY_ENDPOINT,
  KNOWLEDGE_GEOMETRY_MAX_PLANE_SIZE_PX,
  KNOWLEDGE_GEOMETRY_MAX_BYTES,
  KNOWLEDGE_GEOMETRY_MIN_PLANE_SIZE_PX,
  KNOWLEDGE_GEOMETRY_TARGET_SEPARATION_PX,
  KnowledgeGeometryDataError,
  buildKnowledgeGeometryLayout,
  fetchKnowledgeGeometry,
  initialiseKnowledgeGeometry,
  knowledgeGeometryPlaneSize,
  parseKnowledgeGeometry,
  parseKnowledgeGeometryJson,
  renderKnowledgeGeometry,
  type KnowledgeGeometryFact,
  type KnowledgeGeometryRelation,
  type KnowledgeGeometrySnapshot,
} from "../src/knowledge-geometry";

const SNAPSHOT_SCHEMA = "zerone.knowledge-geometry-snapshot/v0";
const QUERY_PATH =
  "/zerone/knowledge/v1/facts?pagination.limit=100&pagination.count_total=true";
const MAX_UINT64 = "18446744073709551615";

type MutableSnapshot = Record<string, any> & {
  source: Record<string, any>;
  facts: Array<Record<string, any>>;
  relations: Array<Record<string, any>>;
};

function fact(
  id: string,
  domain: string,
  overrides: Partial<KnowledgeGeometryFact> = {},
): KnowledgeGeometryFact {
  return {
    id,
    content: `Recorded proposition ${id}`,
    domain,
    category: "protocol observation",
    status: "FACT_STATUS_VERIFIED",
    claimType: "CLAIM_TYPE_OBSERVATION",
    confidence: 875_000,
    verifiedAtBlock: "40",
    lastVerifiedBlock: "41",
    energy: 500_000,
    energyCap: 1_000_000,
    fitnessScore: 625_000,
    methodId: "doctrine_authorship",
    ...overrides,
  };
}

function relation(
  overrides: Partial<KnowledgeGeometryRelation> = {},
): KnowledgeGeometryRelation {
  return {
    sourceFactId: "fact.alpha.00000001",
    targetFactId: "fact.beta.00000002",
    relation: "RELATION_TYPE_SUPPORTS",
    inference: "INFERENCE_TYPE_CITATION",
    inferenceStrengthBps: 750_000,
    createdAtBlock: "40",
    methodId: "",
    ...overrides,
  };
}

function snapshot(): KnowledgeGeometrySnapshot {
  const value = {
    schema: SNAPSHOT_SCHEMA,
    source: {
      chainId: "zerone-1",
      blockHeight: "42",
      statusHeight: "43",
      catchingUp: false,
      queryPath: QUERY_PATH,
      queryTracked: false,
      writes: false,
      completeness: "NOT_CLAIMED",
      upstreamRecords: 2,
      returnedRecords: 2,
      truncated: false,
    },
    facts: [
      fact("fact.alpha.00000001", "Protocol & Safety", {
        content: "  Exact source wording remains exact.  ",
        verifiedAtBlock: "0",
        lastVerifiedBlock: "0",
        energy: 900_000,
        energyCap: 0,
        methodId: "",
      }),
      fact("fact.beta.00000002", "", {
        category: "",
      }),
    ],
    relations: [relation({ createdAtBlock: "0" })],
  };
  return value as unknown as KnowledgeGeometrySnapshot;
}

function copy(): MutableSnapshot {
  return structuredClone(snapshot()) as unknown as MutableSnapshot;
}

function rawSnapshot(): string {
  return JSON.stringify(snapshot());
}

function jsonResponse(
  body: BodyInit | null = rawSnapshot(),
  init: ResponseInit = {},
  url = "",
): Response {
  const headers = new Headers(init.headers);
  if (!headers.has("content-type")) {
    headers.set("content-type", "application/json; charset=utf-8");
  }
  const response = new Response(body, { ...init, headers });
  Object.defineProperty(response, "url", { configurable: true, value: url });
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

describe("knowledge geometry projection parser", () => {
  it("accepts the exact read-only projection and preserves legacy zero metadata", () => {
    const parsed = parseKnowledgeGeometry(snapshot());

    assert.equal(parsed.schema, SNAPSHOT_SCHEMA);
    assert.equal(parsed.source.queryPath, QUERY_PATH);
    assert.equal(parsed.source.queryTracked, false);
    assert.equal(parsed.source.writes, false);
    assert.equal(parsed.source.completeness, "NOT_CLAIMED");
    assert.equal(parsed.facts[0]?.content, "  Exact source wording remains exact.  ");
    assert.equal(parsed.facts[0]?.verifiedAtBlock, "0");
    assert.equal(parsed.facts[0]?.lastVerifiedBlock, "0");
    assert.equal(parsed.facts[0]?.energyCap, 0);
    assert.equal(parsed.facts[0]?.methodId, "");
    assert.equal(parsed.facts[1]?.domain, "");
    assert.equal(parsed.facts[1]?.category, "");
    assert.equal(parsed.relations[0]?.createdAtBlock, "0");
    assert.equal(parsed.relations[0]?.methodId, "");
  });

  it("rejects decoded duplicate keys before JSON.parse can overwrite them", () => {
    const raw = rawSnapshot();
    const duplicateRoot = raw.replace(
      `{"schema":"${SNAPSHOT_SCHEMA}"`,
      `{"\\u0073chema":"${SNAPSHOT_SCHEMA}","schema":"${SNAPSHOT_SCHEMA}"`,
    );
    assert.throws(
      () => parseKnowledgeGeometryJson(duplicateRoot),
      /\$\.schema: is a duplicate JSON key/,
    );

    const duplicateNested = raw.replace(
      `"chainId":"zerone-1"`,
      `"chainId":"zerone-1","\\u0063hainId":"zerone-1"`,
    );
    assert.throws(
      () => parseKnowledgeGeometryJson(duplicateNested),
      /\$\.source\.chainId: is a duplicate JSON key/,
    );
  });

  it("enforces the UTF-8 document boundary, including its exact edge", () => {
    const raw = rawSnapshot();
    const rawBytes = new TextEncoder().encode(raw).byteLength;
    const atLimit = raw + " ".repeat(KNOWLEDGE_GEOMETRY_MAX_BYTES - rawBytes);
    assert.equal(new TextEncoder().encode(atLimit).byteLength, KNOWLEDGE_GEOMETRY_MAX_BYTES);
    assert.equal(parseKnowledgeGeometryJson(atLimit).facts.length, 2);

    assert.throws(
      () => parseKnowledgeGeometryJson(`${atLimit} `),
      new RegExp(`exceeds ${KNOWLEDGE_GEOMETRY_MAX_BYTES} bytes`),
    );

    const multibyteOverflow = JSON.stringify(
      "é".repeat(Math.floor(KNOWLEDGE_GEOMETRY_MAX_BYTES / 2)),
    );
    assert.ok(multibyteOverflow.length < KNOWLEDGE_GEOMETRY_MAX_BYTES);
    assert.ok(
      new TextEncoder().encode(multibyteOverflow).byteLength >
        KNOWLEDGE_GEOMETRY_MAX_BYTES,
    );
    assert.throws(
      () => parseKnowledgeGeometryJson(multibyteOverflow),
      /exceeds/,
    );
  });

  it("pins the source identity, read-only boundary, counts, and truncation claim", () => {
    const mutations: Array<{
      name: string;
      change(value: MutableSnapshot): void;
      error: RegExp;
    }> = [
      {
        name: "chain",
        change: (value) => { value.source.chainId = "zerone-testnet-1"; },
        error: /source\.chainId/,
      },
      {
        name: "query path",
        change: (value) => { value.source.queryPath = "/zerone/knowledge/v1/facts"; },
        error: /source\.queryPath/,
      },
      {
        name: "query tracking",
        change: (value) => { value.source.queryTracked = true; },
        error: /source\.queryTracked/,
      },
      {
        name: "writes",
        change: (value) => { value.source.writes = true; },
        error: /source\.writes/,
      },
      {
        name: "completeness",
        change: (value) => { value.source.completeness = "COMPLETE"; },
        error: /source\.completeness/,
      },
      {
        name: "returned count",
        change: (value) => {
          value.source.returnedRecords = 1;
          value.source.truncated = true;
        },
        error: /facts.*length.*returnedRecords/,
      },
      {
        name: "upstream count",
        change: (value) => { value.source.upstreamRecords = 1; },
        error: /returnedRecords.*upstreamRecords/,
      },
      {
        name: "missing truncation",
        change: (value) => {
          value.source.upstreamRecords = 3;
          value.source.truncated = false;
        },
        error: /source\.truncated/,
      },
    ];

    for (const mutation of mutations) {
      const value = copy();
      mutation.change(value);
      assert.throws(
        () => parseKnowledgeGeometry(value),
        mutation.error,
        mutation.name,
      );
    }

    const honestlyTruncated = copy();
    honestlyTruncated.source.upstreamRecords = 3;
    honestlyTruncated.source.truncated = true;
    assert.equal(parseKnowledgeGeometry(honestlyTruncated).source.truncated, true);

    const relationOrPaginationTruncated = copy();
    relationOrPaginationTruncated.source.truncated = true;
    assert.equal(
      parseKnowledgeGeometry(relationOrPaginationTruncated).source.truncated,
      true,
    );

    const unknown = copy();
    unknown.source.authoritative = true;
    assert.throws(() => parseKnowledgeGeometry(unknown), /unknown.*authoritative/);
  });

  it("accepts either source-height order within 128 and bounds every nonzero metadata height by both", () => {
    for (const [blockHeight, statusHeight] of [
      ["1000", "1128"],
      ["1128", "1000"],
    ] as const) {
      const value = copy();
      value.source.blockHeight = blockHeight;
      value.source.statusHeight = statusHeight;
      value.facts[0]!.verifiedAtBlock = "0";
      value.facts[0]!.lastVerifiedBlock = "0";
      value.facts[1]!.verifiedAtBlock = "999";
      value.facts[1]!.lastVerifiedBlock = "1000";
      value.relations[0]!.createdAtBlock = "1000";
      assert.equal(parseKnowledgeGeometry(value).source.blockHeight, blockHeight);
    }

    const tooFar = copy();
    tooFar.source.blockHeight = "1000";
    tooFar.source.statusHeight = "1129";
    assert.throws(() => parseKnowledgeGeometry(tooFar), /height.*128|128.*height/);

    for (const field of ["verifiedAtBlock", "lastVerifiedBlock"] as const) {
      const future = copy();
      future.source.blockHeight = "100";
      future.source.statusHeight = "101";
      future.facts[0]![field] = "101";
      assert.throws(
        () => parseKnowledgeGeometry(future),
        new RegExp(`facts\\[0\\].*height|facts\\[0\\]\\.${field}`),
      );
    }

    const futureRelation = copy();
    futureRelation.source.blockHeight = "100";
    futureRelation.source.statusHeight = "101";
    futureRelation.relations[0]!.createdAtBlock = "101";
    assert.throws(
      () => parseKnowledgeGeometry(futureRelation),
      /relations\[0\]\.createdAtBlock|relations\[0\].*height/,
    );

    const inconsistent = copy();
    inconsistent.facts[1]!.verifiedAtBlock = "41";
    inconsistent.facts[1]!.lastVerifiedBlock = "40";
    assert.throws(() => parseKnowledgeGeometry(inconsistent), /facts\[1\].*height/);
  });

  it("uses canonical uint64 strings while retaining zero as legacy absence metadata", () => {
    const maximum = copy();
    maximum.source.blockHeight = MAX_UINT64;
    maximum.source.statusHeight = MAX_UINT64;
    maximum.facts[0]!.verifiedAtBlock = "0";
    maximum.facts[0]!.lastVerifiedBlock = "0";
    maximum.facts[1]!.verifiedAtBlock = MAX_UINT64;
    maximum.facts[1]!.lastVerifiedBlock = MAX_UINT64;
    maximum.relations[0]!.createdAtBlock = MAX_UINT64;
    assert.equal(
      parseKnowledgeGeometry(maximum).relations[0]?.createdAtBlock,
      MAX_UINT64,
    );

    const zeroSource = copy();
    zeroSource.source.blockHeight = "0";
    assert.throws(() => parseKnowledgeGeometry(zeroSource), /positive uint64 height/);

    const leadingZero = copy();
    leadingZero.facts[1]!.verifiedAtBlock = "040";
    assert.throws(() => parseKnowledgeGeometry(leadingZero), /invalid format/);

    const overflow = copy();
    overflow.relations[0]!.createdAtBlock = "18446744073709551616";
    assert.throws(() => parseKnowledgeGeometry(overflow), /canonical uint64/);
  });

  it("preserves safe text and method IDs by UTF-8 byte length", () => {
    const exact = copy();
    exact.facts[0]!.content = "é".repeat(8_192);
    exact.facts[0]!.domain = "é".repeat(64);
    exact.facts[0]!.category = " Mixed case / punctuation: exact ";
    exact.facts[0]!.methodId = "m".repeat(128);
    exact.relations[0]!.methodId = "";
    const parsed = parseKnowledgeGeometry(exact);
    assert.equal(parsed.facts[0]?.content, exact.facts[0]!.content);
    assert.equal(parsed.facts[0]?.domain, exact.facts[0]!.domain);
    assert.equal(parsed.facts[0]?.category, exact.facts[0]!.category);

    const contentOverflow = copy();
    contentOverflow.facts[0]!.content = `${"é".repeat(8_192)}a`;
    assert.throws(() => parseKnowledgeGeometry(contentOverflow), /facts\[0\]\.content/);

    const labelOverflow = copy();
    labelOverflow.facts[0]!.domain = `${"é".repeat(64)}a`;
    assert.throws(() => parseKnowledgeGeometry(labelOverflow), /facts\[0\]\.domain/);

    for (const unsafe of ["unsafe\u0000text", "unsafe\u202etext"]) {
      const value = copy();
      value.facts[0]!.content = unsafe;
      assert.throws(() => parseKnowledgeGeometry(value), /facts\[0\]\.content/);
    }

    const emptyContent = copy();
    emptyContent.facts[0]!.content = "";
    assert.throws(() => parseKnowledgeGeometry(emptyContent), /facts\[0\]\.content/);

    for (const location of ["fact", "relation"] as const) {
      const invalid = copy();
      if (location === "fact") invalid.facts[0]!.methodId = "method/unsafe";
      else invalid.relations[0]!.methodId = "method/unsafe";
      assert.throws(() => parseKnowledgeGeometry(invalid), /methodId/);
    }
  });

  it("enforces safe unique fact IDs and directional relation-pair authority", () => {
    for (const id of ["", " leading", "fact/unsafe", `f${"x".repeat(128)}`]) {
      const invalid = copy();
      invalid.facts[0]!.id = id;
      assert.throws(() => parseKnowledgeGeometry(invalid), /facts\[0\]\.id/);
    }

    const duplicateFact = copy();
    duplicateFact.facts[1]!.id = duplicateFact.facts[0]!.id;
    assert.throws(() => parseKnowledgeGeometry(duplicateFact), /facts\[1\]\.id.*unique/);

    const detached = copy();
    detached.relations[0]!.sourceFactId = "outside.source";
    detached.relations[0]!.targetFactId = "outside.target";
    assert.throws(() => parseKnowledgeGeometry(detached), /touch at least one returned fact/);

    const truncatedEndpoint = copy();
    truncatedEndpoint.relations[0]!.targetFactId = "outside.returned.window";
    assert.equal(parseKnowledgeGeometry(truncatedEndpoint).relations.length, 1);

    const duplicatePair = copy();
    duplicatePair.relations.push({
      ...duplicatePair.relations[0]!,
      relation: "RELATION_TYPE_CONTRADICTS",
      inferenceStrengthBps: 1,
      createdAtBlock: "41",
    });
    assert.throws(() => parseKnowledgeGeometry(duplicatePair), /relations\[1\].*duplic/);
  });

  it("caps ordinary dense arrays and every bounded metric", () => {
    const sparse = copy();
    sparse.facts = new Array(2);
    sparse.facts[0] = fact("fact.alpha.00000001", "alpha");
    assert.throws(() => parseKnowledgeGeometry(sparse), /facts.*dense.*slot 1/);

    const maximumFacts = copy();
    maximumFacts.source.upstreamRecords = 128;
    maximumFacts.source.returnedRecords = 128;
    maximumFacts.facts = Array.from({ length: 128 }, (_unused, index) =>
      fact(`fact.${String(index).padStart(3, "0")}`, "domain"),
    );
    maximumFacts.relations = [];
    assert.equal(parseKnowledgeGeometry(maximumFacts).facts.length, 128);

    const tooManyFacts = copy();
    tooManyFacts.source.upstreamRecords = 129;
    tooManyFacts.source.returnedRecords = 128;
    tooManyFacts.source.truncated = true;
    tooManyFacts.facts = Array.from({ length: 129 }, (_unused, index) =>
      fact(`fact.${String(index).padStart(3, "0")}`, "domain"),
    );
    assert.throws(() => parseKnowledgeGeometry(tooManyFacts), /facts.*at most 128/);

    const maximumRelations = copy();
    maximumRelations.relations = Array.from({ length: 512 }, (_unused, index) =>
      relation({ targetFactId: `outside.${String(index).padStart(3, "0")}` }),
    );
    assert.equal(parseKnowledgeGeometry(maximumRelations).relations.length, 512);

    const tooManyRelations = copy();
    tooManyRelations.relations = Array.from({ length: 513 }, (_unused, index) =>
      relation({
        targetFactId: `outside.${String(index).padStart(3, "0")}`,
      }),
    );
    assert.throws(
      () => parseKnowledgeGeometry(tooManyRelations),
      /relations.*at most 512/,
    );

    for (const [section, field] of [
      ["facts", "confidence"],
      ["facts", "energy"],
      ["facts", "energyCap"],
      ["facts", "fitnessScore"],
      ["relations", "inferenceStrengthBps"],
    ] as const) {
      const above = copy();
      above[section][0]![field] = 1_000_001;
      assert.throws(() => parseKnowledgeGeometry(above), /between 0 and 1000000/);

      const fractional = copy();
      fractional[section][0]![field] = 1.5;
      assert.throws(() => parseKnowledgeGeometry(fractional), /integer/);
    }
  });

  it("rejects non-plain shapes, unknown fields, malformed JSON, and trailing data", () => {
    assert.throws(
      () => parseKnowledgeGeometry(new (class Snapshot {})()),
      /plain object/,
    );

    const unknown = copy();
    unknown.facts[0]!.popularity = 1;
    assert.throws(() => parseKnowledgeGeometry(unknown), /unknown.*popularity/);

    assert.throws(
      () => parseKnowledgeGeometryJson("{not-json}"),
      KnowledgeGeometryDataError,
    );
    assert.throws(
      () => parseKnowledgeGeometryJson(`${rawSnapshot()} true`),
      /trailing JSON data/,
    );
  });
});

describe("knowledge geometry deterministic layout", () => {
  it("is identical under shuffled input and uses raw code-unit ordering", () => {
    const facts = [
      fact("Z.fact", "z-domain"),
      fact("a.fact", "z-domain"),
      fact("B.fact", "A-domain"),
      fact("b.fact", "A-domain"),
    ];
    const originalIds = facts.map(({ id }) => id);
    const first = buildKnowledgeGeometryLayout(facts);
    const shuffled = buildKnowledgeGeometryLayout([
      facts[2]!,
      facts[0]!,
      facts[3]!,
      facts[1]!,
    ]);

    assert.deepEqual(shuffled, first);
    assert.deepEqual(facts.map(({ id }) => id), originalIds);
    assert.deepEqual(first.domains.map(({ id }) => id), ["A-domain", "z-domain"]);
    assert.deepEqual(first.points.map(({ factId }) => factId), [
      "B.fact",
      "b.fact",
      "Z.fact",
      "a.fact",
    ]);
    assert.ok(first.points.every(({ x }) => x >= 3 && x <= 97));
    assert.ok(first.points.every(({ y }) => y >= 5 && y <= 95));
  });

  it("does not turn confidence, energy, fitness, content, or input mutation into geometry", () => {
    const facts = [
      fact("fact.low", "one", {
        confidence: 0,
        energy: 0,
        fitnessScore: 0,
      }),
      fact("fact.high", "one", {
        confidence: 1_000_000,
        energy: 1_000_000,
        fitnessScore: 1_000_000,
      }),
    ];
    const changedMetrics = facts.map((entry) => ({
      ...entry,
      content: `changed ${entry.id}`,
      confidence: 123,
      energy: 456,
      energyCap: 789,
      fitnessScore: 999,
    }));

    const expected = buildKnowledgeGeometryLayout(facts);
    assert.deepEqual(buildKnowledgeGeometryLayout(changedMetrics), expected);
    assert.deepEqual(expected.domains, [
      { id: "one", index: 0, x: 50, y: 50, factCount: 2 },
    ]);
    assert.equal(expected.points[0]?.x, 50);
    assert.equal(expected.points[0]?.y, 50);
  });

  it("keeps the current live domain distribution at least 24 pixels apart on the minimum plane", () => {
    const liveDistribution = [
      ["mathematics", 23],
      ["doctrine_truth_seeking", 20],
      ["doctrine_useful_work", 14],
      ["doctrine_strange_loop", 7],
      ["doctrine_tok", 6],
      ["general", 2],
    ] as const;
    const facts = liveDistribution.flatMap(([domain, count]) =>
      Array.from({ length: count }, (_unused, index) =>
        fact(`${domain}.${String(index).padStart(2, "0")}`, domain),
      ),
    );
    const { points } = buildKnowledgeGeometryLayout(facts);
    const planeSize = knowledgeGeometryPlaneSize(points);
    let minimumPixels = Number.POSITIVE_INFINITY;
    for (let left = 0; left < points.length; left += 1) {
      for (let right = left + 1; right < points.length; right += 1) {
        const first = points[left]!;
        const second = points[right]!;
        minimumPixels = Math.min(
          minimumPixels,
          Math.hypot(
            ((first.x - second.x) * planeSize) / 100,
            ((first.y - second.y) * planeSize) / 100,
          ),
        );
      }
    }

    assert.equal(planeSize, KNOWLEDGE_GEOMETRY_MIN_PLANE_SIZE_PX);
    assert.equal(points.length, 72);
    assert.ok(
      minimumPixels >= KNOWLEDGE_GEOMETRY_TARGET_SEPARATION_PX,
      `minimum record-node separation was ${minimumPixels.toFixed(2)}px`,
    );
  });

  it("expands a maximum single-domain layout enough to preserve equal pointer access", () => {
    const facts = Array.from({ length: 128 }, (_unused, index) =>
      fact(`fact.${String(index).padStart(3, "0")}`, "domain"),
    );
    const { points } = buildKnowledgeGeometryLayout(facts);
    const planeSize = knowledgeGeometryPlaneSize(points);
    let minimumPixels = Number.POSITIVE_INFINITY;
    for (let left = 0; left < points.length; left += 1) {
      for (let right = left + 1; right < points.length; right += 1) {
        minimumPixels = Math.min(
          minimumPixels,
          (Math.hypot(
            points[left]!.x - points[right]!.x,
            points[left]!.y - points[right]!.y,
          ) *
            planeSize) /
            100,
        );
      }
    }

    assert.equal(points.length, 128);
    assert.equal(planeSize, 6_335);
    assert.ok(planeSize <= KNOWLEDGE_GEOMETRY_MAX_PLANE_SIZE_PX);
    assert.ok(
      minimumPixels >= KNOWLEDGE_GEOMETRY_TARGET_SEPARATION_PX,
      `minimum record-node separation was ${minimumPixels.toFixed(2)}px`,
    );
  });

  it("refuses non-finite, coincident, and adversarially close layouts", () => {
    const point = (factId: string, x: number, y: number) => ({
      factId,
      domain: "domain",
      domainIndex: 0,
      x,
      y,
    });

    assert.throws(
      () => knowledgeGeometryPlaneSize([point("one", Number.NaN, 0)]),
      KnowledgeGeometryDataError,
    );
    assert.throws(
      () => knowledgeGeometryPlaneSize([point("one", 1, 1), point("two", 1, 1)]),
      /must not coincide/,
    );
    assert.throws(
      () =>
        knowledgeGeometryPlaneSize([
          point("one", 1, 1),
          point("two", 1.1, 1),
        ]),
      new RegExp(`larger than ${KNOWLEDGE_GEOMETRY_MAX_PLANE_SIZE_PX}px`),
    );
  });
});

describe("knowledge geometry bounded fetch", () => {
  it("requests the exact endpoint with GET, JSON acceptance, manual redirects, and a deadline", async () => {
    let inputSeen: RequestInfo | URL | undefined;
    let initSeen: RequestInit | undefined;
    const parsed = await fetchKnowledgeGeometry({
      baseUrl: "https://zerone.example/dashboard?ignored=true",
      fetcher: async (input, init) => {
        inputSeen = input;
        initSeen = init;
        return jsonResponse(
          rawSnapshot(),
          {},
          `https://zerone.example${KNOWLEDGE_GEOMETRY_ENDPOINT}`,
        );
      },
    });

    assert.ok(inputSeen instanceof URL);
    assert.equal(inputSeen.href, `https://zerone.example${KNOWLEDGE_GEOMETRY_ENDPOINT}`);
    assert.equal(initSeen?.method, "GET");
    assert.equal(new Headers(initSeen?.headers).get("accept"), "application/json");
    assert.equal(initSeen?.redirect, "manual");
    assert.ok(initSeen?.signal instanceof AbortSignal);
    assert.equal(parsed.facts.length, 2);
  });

  it("refuses redirects and any final URL drift, including fragments", async () => {
    const redirected = jsonResponse();
    Object.defineProperty(redirected, "redirected", { value: true });
    await assert.rejects(
      fetchKnowledgeGeometry({
        baseUrl: "https://zerone.example/",
        fetcher: async () => redirected,
      }),
      /redirects are refused/,
    );

    for (const url of [
      "https://attacker.example/api/knowledge",
      "https://zerone.example/api/other",
      "https://zerone.example/api/knowledge?unbounded=true",
      "https://zerone.example/api/knowledge#unreviewed",
    ]) {
      await assert.rejects(
        fetchKnowledgeGeometry({
          baseUrl: "https://zerone.example/",
          fetcher: async () => jsonResponse(rawSnapshot(), {}, url),
        }),
        /response URL does not match/,
        url,
      );
    }
  });

  it("rejects HTTP failures and anything but application/json", async () => {
    await assert.rejects(
      fetchKnowledgeGeometry({
        fetcher: async () => jsonResponse("{}", { status: 503 }),
      }),
      /HTTP 503/,
    );
    await assert.rejects(
      fetchKnowledgeGeometry({
        fetcher: async () =>
          new Response(rawSnapshot(), { headers: { "content-type": "text/html" } }),
      }),
      /must use application\/json/,
    );
    await assert.rejects(
      fetchKnowledgeGeometry({
        fetcher: async () => new Response(rawSnapshot()),
      }),
      /must use application\/json/,
    );
  });

  it("enforces declared and streamed byte limits and fatal UTF-8 decoding", async () => {
    await assert.rejects(
      fetchKnowledgeGeometry({
        fetcher: async () =>
          jsonResponse("{}", {
            headers: {
              "content-length": String(KNOWLEDGE_GEOMETRY_MAX_BYTES + 1),
            },
          }),
      }),
      /response exceeds/,
    );
    await assert.rejects(
      fetchKnowledgeGeometry({
        fetcher: async () =>
          jsonResponse("{}", { headers: { "content-length": "01" } }),
      }),
      /response exceeds/,
    );

    const oversized = new Uint8Array(KNOWLEDGE_GEOMETRY_MAX_BYTES + 1).fill(0x20);
    await assert.rejects(
      fetchKnowledgeGeometry({
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
      /response exceeds/,
    );

    await assert.rejects(
      fetchKnowledgeGeometry({
        fetcher: async () => jsonResponse(new Uint8Array([0xff])),
      }),
      /not valid UTF-8/,
    );

    const raw = rawSnapshot();
    const atLimit = raw +
      " ".repeat(
        KNOWLEDGE_GEOMETRY_MAX_BYTES - new TextEncoder().encode(raw).byteLength,
      );
    const accepted = await fetchKnowledgeGeometry({
      fetcher: async () =>
        jsonResponse(atLimit, {
          headers: { "content-length": String(KNOWLEDGE_GEOMETRY_MAX_BYTES) },
        }),
    });
    assert.equal(accepted.facts.length, 2);
  });

  it("bounds a stalled request using the supplied request signal", async () => {
    let signalSeen: AbortSignal | undefined;
    await assert.rejects(
      fetchKnowledgeGeometry({
        timeoutMs: 10,
        fetcher: async (_input, init) =>
          await new Promise<Response>((_resolve, reject) => {
            signalSeen = init?.signal ?? undefined;
            assert.ok(signalSeen);
            signalSeen.addEventListener("abort", () => reject(signalSeen?.reason), {
              once: true,
            });
          }),
      }),
      /projection request timed out/,
    );
    assert.equal(signalSeen?.aborted, true);
  });

  it("times out and cancels a stalled response body", async () => {
    let cancelled = false;
    await assert.rejects(
      fetchKnowledgeGeometry({
        timeoutMs: 10,
        fetcher: async () =>
          jsonResponse(
            new ReadableStream<Uint8Array>({
              start(controller) {
                controller.enqueue(new TextEncoder().encode('{"schema":'));
              },
              cancel() {
                cancelled = true;
              },
            }),
          ),
      }),
      /projection request timed out/,
    );
    assert.equal(cancelled, true);
  });

  it("refuses overflow without awaiting hostile stream cancellation", async () => {
    const oversized = new Uint8Array(KNOWLEDGE_GEOMETRY_MAX_BYTES + 1).fill(0x20);
    await assert.rejects(
      settlesWithin(
        fetchKnowledgeGeometry({
          fetcher: async () =>
            jsonResponse(
              new ReadableStream<Uint8Array>({
                start(controller) {
                  controller.enqueue(oversized);
                },
                cancel: async () => await new Promise<void>(() => {}),
              }),
            ),
        }),
        100,
      ),
      /response exceeds/,
    );
  });
});

class FakeStyle {
  readonly properties = new Map<string, string>();

  setProperty(name: string, value: string): void {
    this.properties.set(name, value);
  }
}

class FakeElement {
  readonly children: FakeElement[] = [];
  readonly attributes = new Map<string, string>();
  readonly dataset: Record<string, string> = {};
  readonly style = new FakeStyle();
  readonly listeners = new Map<
    string,
    Array<(event: Record<string, unknown>) => unknown>
  >();
  className = "";
  type = "";
  placeholder = "";
  autocomplete = "";
  value = "";
  selectedIndex = 0;
  tabIndex = 0;
  disabled = false;
  hidden = false;
  focused = false;
  scrolledIntoView = false;
  title = "";
  href = "";
  private text: string | null = null;

  constructor(readonly tagName: string) {}

  get options(): FakeElement[] {
    return this.children.filter(({ tagName }) => tagName === "option");
  }

  get textContent(): string | null {
    return this.text;
  }

  set textContent(value: string | null) {
    this.text = value;
    this.children.splice(0);
  }

  set innerHTML(_value: string) {
    throw new Error("knowledge geometry renderer must never use innerHTML");
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

  focus(_options?: FocusOptions): void {
    this.focused = true;
  }

  scrollIntoView(_options?: ScrollIntoViewOptions): void {
    this.scrolledIntoView = true;
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
      createElementNS(_namespace: string, tag: string) {
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

describe("knowledge geometry renderer", () => {
  it("renders hostile record copy as text and keeps every fact node visually equal", async () => {
    await withFakeDocument(async () => {
      const value = snapshot();
      const hostile = '<img src=x onerror="globalThis.pwned=true"> & understood?';
      value.facts[0]!.content = hostile;
      value.facts[0]!.confidence = 0;
      value.facts[0]!.energy = 0;
      value.facts[0]!.fitnessScore = 0;
      value.facts[1]!.confidence = 1_000_000;
      value.facts[1]!.energy = 1_000_000;
      value.facts[1]!.energyCap = 1_000_000;
      value.facts[1]!.fitnessScore = 1_000_000;

      const root = new FakeElement("div");
      renderKnowledgeGeometry(root as unknown as HTMLElement, value);

      assert.equal(root.getAttribute("aria-busy"), "false");
      const all = descendants(root);
      assert.equal(all.filter(({ tagName }) => tagName === "img").length, 0);
      assert.equal(all.filter(({ tagName }) => tagName === "script").length, 0);
      const content = all.find(
        ({ className }) => className === "knowledge-geometry-fact-content",
      );
      assert.ok(content);
      assert.equal(content.textContent, hostile);
      assert.ok(flattenedText(root).includes(hostile));

      const mapScroll = all.find(
        ({ className }) => className === "knowledge-geometry-map-scroll",
      );
      const map = all.find(
        ({ className }) => className === "knowledge-geometry-map",
      );
      assert.ok(mapScroll);
      assert.ok(map);
      assert.equal(
        map.style.properties.get("--kg-plane-size"),
        `${KNOWLEDGE_GEOMETRY_MIN_PLANE_SIZE_PX}px`,
      );
      assert.equal(
        map.getAttribute("aria-describedby"),
        "knowledge-geometry-result-status",
      );
      const resultStatus = all.find(
        ({ className }) => className === "knowledge-geometry-result-status",
      );
      assert.ok(resultStatus);
      assert.equal(resultStatus.getAttribute("role"), "status");
      assert.equal(resultStatus.getAttribute("aria-live"), "polite");
      assert.match(resultStatus.textContent ?? "", /2 of 2 returned records visible/);

      const nodes = all.filter(
        ({ className }) => className === "knowledge-geometry-node",
      );
      assert.equal(nodes.length, 2);
      for (const node of nodes) {
        assert.equal(node.tagName, "button");
        assert.equal(node.type, "button");
        assert.equal(node.textContent, null);
        assert.equal(node.children.length, 0);
        assert.equal(
          node.getAttribute("aria-controls"),
          "knowledge-geometry-inspector",
        );
        assert.deepEqual([...node.style.properties.keys()].sort(), ["--kg-x", "--kg-y"]);
        assert.deepEqual(Object.keys(node.dataset).sort(), [
          "domain",
          "domainIndex",
          "factId",
          "status",
        ]);
      }
      assert.ok(
        nodes.every(
          (node) =>
            ![...node.style.properties.keys()].some((name) =>
              /size|scale|radius|confidence|energy|fitness/i.test(name),
          ),
        ),
      );
      assert.deepEqual(nodes.map(({ tabIndex }) => tabIndex).sort(), [-1, 0]);

      const initiallyRoving = nodes.find(({ tabIndex }) => tabIndex === 0);
      assert.ok(initiallyRoving);
      let defaultPrevented = false;
      await initiallyRoving.dispatch("keydown", {
        key: "ArrowRight",
        preventDefault() {
          defaultPrevented = true;
        },
      });
      assert.equal(defaultPrevented, true);
      const movedTo = nodes.find(({ tabIndex }) => tabIndex === 0);
      assert.ok(movedTo);
      assert.notEqual(movedTo, initiallyRoving);
      assert.equal(movedTo.focused, true);
      assert.equal(movedTo.scrolledIntoView, true);
      assert.equal(movedTo.getAttribute("aria-pressed"), "true");

      await movedTo.dispatch("keydown", {
        key: "Home",
        preventDefault() {},
      });
      assert.equal(nodes[0]?.tabIndex, 0);
      await nodes[0]!.dispatch("keydown", {
        key: "End",
        preventDefault() {},
      });
      assert.equal(nodes.at(-1)?.tabIndex, 0);

      const search = all.find(
        ({ className }) => className === "knowledge-geometry-search",
      );
      const inspector = all.find(
        ({ className }) => className === "knowledge-geometry-inspector",
      );
      assert.ok(search);
      assert.ok(inspector);
      search.value = "no returned record has this text";
      await search.dispatch("input");
      assert.ok(nodes.every(({ hidden }) => hidden));
      assert.ok(nodes.every(({ tabIndex }) => tabIndex === -1));
      assert.match(resultStatus.textContent ?? "", /0 of 2 returned records visible/);
      assert.match(flattenedText(inspector), /No records in this view/);

      search.value = "";
      await search.dispatch("input");
      assert.equal(nodes.filter(({ tabIndex }) => tabIndex === 0).length, 1);
    });
  });

  it("announces a refused snapshot and retries the bounded read in place", async () => {
    await withFakeDocument(async () => {
      let attempts = 0;
      const root = new FakeElement("div");
      await initialiseKnowledgeGeometry(root as unknown as HTMLElement, {
        baseUrl: "https://zerone.example/",
        fetcher: async () => {
          attempts += 1;
          return attempts === 1
            ? jsonResponse("{}", { status: 503 })
            : jsonResponse(rawSnapshot());
        },
      });

      const error = descendants(root).find(
        ({ className }) => className === "knowledge-geometry-error",
      );
      assert.ok(error);
      assert.equal(error.getAttribute("role"), "alert");
      assert.equal(error.getAttribute("aria-live"), "assertive");
      assert.equal(error.getAttribute("aria-atomic"), "true");
      const retry = descendants(error).find(
        ({ textContent }) => textContent === "Retry bounded read",
      );
      assert.ok(retry);

      await retry.dispatch("click");

      assert.equal(attempts, 2);
      assert.equal(root.getAttribute("aria-busy"), "false");
      assert.ok(
        descendants(root).some(
          ({ className }) => className === "knowledge-geometry-shell",
        ),
      );
    });
  });
});

describe("knowledge geometry dashboard integration", () => {
  it("keeps the static no-JS fallback settled and marks only a live read busy", () => {
    const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
    const runtime = readFileSync(
      new URL("../src/knowledge-geometry.ts", import.meta.url),
      "utf8",
    );

    assert.match(
      html,
      /class="knowledge-geometry-root"\s+id="knowledge-geometry-root"\s*>/,
    );
    assert.doesNotMatch(
      html,
      /id="knowledge-geometry-root"\s+aria-busy="true"/,
    );
    assert.match(runtime, /root\.setAttribute\("aria-busy", "true"\);/);
  });

  it("waits for all initial surfaces and aligns a direct hash only once", () => {
    const source = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");
    assert.match(source, /let initialHashInputsSettled = false;/);
    assert.match(source, /let initialHashAligned = false;/);
    assert.match(
      source,
      /if \(!initialHashInputsSettled \|\| initialHashAligned\) return;/,
    );
    assert.equal(source.match(/initialHashAligned = true;/g)?.length, 1);
    assert.match(
      source,
      /Promise\.allSettled\(\[\s*knowledgeGeometryReady,[\s\S]*?initialNetworkReady,\s*\]\)\.then\(\(\) => \{\s*initialHashInputsSettled = true;\s*alignInitialHash\(\);/,
    );
  });
});
