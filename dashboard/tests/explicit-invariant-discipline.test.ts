import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  realpathSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

import {
  EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT,
  EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES,
  EXPLICIT_INVARIANT_DISCIPLINE_SHA256,
  fetchExplicitInvariantDiscipline,
  initialiseExplicitInvariantDiscipline,
  parseExplicitInvariantDisciplineJson,
  renderExplicitInvariantDiscipline,
} from "../src/explicit-invariant-discipline";
import { validateExplicitInvariantDisciplineRaw } from "../scripts/validate-explicit-invariant-discipline";

type MutableDocument = Record<string, any>;

const REVIEWED_SHA256 =
  "e60b89cbed8eb26d3fad0ee45ef8c433391341f3abb4865af2755595815354df";
const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/explicit-invariant-discipline.v1.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as MutableDocument;
const validatorCli = fileURLToPath(
  new URL("../scripts/validate-explicit-invariant-discipline.ts", import.meta.url),
);
const tsxCli = fileURLToPath(new URL("../node_modules/.bin/tsx", import.meta.url));

function runValidatorCli(manifestPath: string, repositoryRoot: string) {
  return spawnSync(tsxCli, [validatorCli, manifestPath, repositoryRoot], {
    encoding: "utf8",
    maxBuffer: 1_048_576,
    timeout: 5_000,
  });
}

function copy(): MutableDocument {
  return structuredClone(canonical) as MutableDocument;
}

function parseMutation(document: MutableDocument): unknown {
  return parseExplicitInvariantDisciplineJson(JSON.stringify(document));
}

function sourceBytes(
  overrides: ReadonlyMap<string, string | Uint8Array> = new Map(),
): ReadonlyMap<string, string | Uint8Array> {
  return new Map(
    canonical.sourceBindings.map((binding: MutableDocument) => [
      binding.path as string,
      overrides.get(binding.path as string) ??
        readFileSync(new URL(`../../${binding.path as string}`, import.meta.url)),
    ]),
  );
}

function jsonResponse(
  body: BodyInit | null = canonicalRaw,
  init: ResponseInit = {},
  url = `https://zerone.ai${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}`,
  redirected = false,
): Response {
  const headers = new Headers(init.headers);
  if (!headers.has("content-type")) {
    headers.set("content-type", "application/json; charset=utf-8");
  }
  const response = new Response(body, { ...init, headers });
  Object.defineProperty(response, "url", { configurable: true, value: url });
  Object.defineProperty(response, "redirected", {
    configurable: true,
    value: redirected,
  });
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

class FakeStyle {
  readonly values = new Map<string, string>();

  setProperty(name: string, value: string): void {
    this.values.set(name, value);
  }

  removeProperty(name: string): string {
    const previous = this.values.get(name) ?? "";
    this.values.delete(name);
    return previous;
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
  readonly style = new FakeStyle();
  parentElement: FakeElement | null = null;
  className = "";
  href = "";
  target = "";
  rel = "";
  type = "";
  title = "";
  id = "";
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
    throw new Error("explicit-invariant renderer must never use innerHTML");
  }

  set outerHTML(_value: string) {
    throw new Error("explicit-invariant renderer must never use outerHTML");
  }

  get firstElementChild(): FakeElement | null {
    return this.children[0] ?? null;
  }

  append(...nodes: Array<FakeElement | string>): void {
    for (const node of nodes) {
      const child =
        typeof node === "string"
          ? Object.assign(new FakeElement("#text"), { textContent: node })
          : node;
      child.parentElement = this;
      this.children.push(child);
    }
  }

  appendChild<T extends FakeElement>(node: T): T {
    this.append(node);
    return node;
  }

  replaceChildren(...nodes: Array<FakeElement | string>): void {
    for (const child of this.children) child.parentElement = null;
    this.children.splice(0);
    this.text = null;
    this.append(...nodes);
  }

  removeChild<T extends FakeElement>(node: T): T {
    const index = this.children.indexOf(node);
    if (index >= 0) this.children.splice(index, 1);
    node.parentElement = null;
    return node;
  }

  setAttribute(name: string, value: string): void {
    this.attributes.set(name, value);
    if (name === "class") this.className = value;
    if (name === "id") this.id = value;
    if (name.startsWith("data-")) {
      const key = name
        .slice(5)
        .replace(/-([a-z])/gu, (_match, letter: string) => letter.toUpperCase());
      this.dataset[key] = value;
    }
  }

  getAttribute(name: string): string | null {
    if (name === "class") return this.className || null;
    if (name === "id") return this.id || null;
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
      await listener({ currentTarget: this, target: this, ...event });
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
      createTextNode(value: string) {
        const node = new FakeElement("#text");
        node.textContent = value;
        return node;
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

describe("explicit invariant discipline v1 seal and parser", () => {
  it("pins the literal seal, reviewed counts, four result records, and zero effects", () => {
    const discipline = parseExplicitInvariantDisciplineJson(canonicalRaw);

    assert.equal(discipline.schema, "zerone.explicit-invariant-discipline/v1");
    assert.equal(discipline.version, 1);
    assert.equal(discipline.snapshotDate, "2026-08-16");
    assert.equal(discipline.status, "SEALED_STATIC_PROFILE");
    assert.equal(discipline.records.length, 4);
    assert.equal(discipline.primarySources.length, 4);
    assert.equal(discipline.sourceBindings.length, 4);
    assert.equal(discipline.integrationTargets.length, 5);
    assert.equal(Object.keys(discipline.releaseBoundary).length, 24);
    assert.equal(
      discipline.records.reduce(
        (count, record) => count + record.sourceResult.assumptions.length,
        0,
      ),
      30,
    );
    assert.equal(
      discipline.records.reduce(
        (count, record) => count + record.sourceResult.invariants.length,
        0,
      ),
      10,
    );
    assert.equal(
      discipline.records.reduce(
        (count, record) => count + record.sourceResult.constraintWitnesses.length,
        0,
      ),
      8,
    );
    assert.equal(
      discipline.records.reduce(
        (count, record) =>
          count + record.sourceResult.remainingFamily.relaxationBranches.length,
        0,
      ),
      18,
    );
    assert.equal(
      discipline.records.reduce(
        (count, record) => count + record.zeroneTransfer.nonTransfers.length,
        0,
      ),
      16,
    );
    assert.equal(discipline.browserBoundary.staticReadCount, 1);
    assert.equal(discipline.browserBoundary.externalFetchCount, 0);
    assert.equal(discipline.browserBoundary.sameOriginOnly, true);
    assert.ok(
      Object.values(discipline.releaseBoundary).every((value) => value === false),
    );
    assert.equal(
      discipline.releaseBoundary.automaticProtocolOrAuthorityAction,
      false,
    );
    assert.equal("automaticAction" in discipline.releaseBoundary, false);
    assert.deepEqual(
      discipline.records.map((record) => [
        record.id,
        record.sourceResult.result.kind,
        record.sourceResult.remainingFamily.cardinality,
        record.zeroneTransfer.localTest.status,
      ]),
      [
        ["boundary-probing-zero-input-robustness", "FAMILY", "MANY", "NOT_RUN"],
        [
          "factorization-declared-dependency-integrity",
          "CONDITIONAL_UNIQUENESS",
          "ONE",
          "NOT_RUN",
        ],
        [
          "bootstrap-conditional-solution-space",
          "CONDITIONAL_UNIQUENESS",
          "ONE",
          "NOT_RUN",
        ],
        ["witness-projection-publication-integrity", "FAMILY", "MANY", "NOT_RUN"],
      ],
    );
    assert.deepEqual(
      discipline.integrationTargets.map(({ id, status }) => [id, status]),
      [
        ["cg-0", "CURRENT_STATIC_REFERENCE"],
        ["knowledge-boundary", "CURRENT_STATIC_REFERENCE"],
        ["m-analogical", "CURRENT_STATIC_REFERENCE"],
        ["math-proofcraft-1", "CURRENT_STATIC_REFERENCE"],
        ["tok-per-fact-explicit-invariants", "NOT_IMPLEMENTED"],
      ],
    );
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      REVIEWED_SHA256,
    );
    assert.equal(EXPLICIT_INVARIANT_DISCIPLINE_SHA256, REVIEWED_SHA256);
    assert.ok(
      new TextEncoder().encode(canonicalRaw).byteLength <
        EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES,
    );
  });

  it("pins exact primary-source authorship, version dates, and arXiv locators", () => {
    const discipline = parseExplicitInvariantDisciplineJson(canonicalRaw);
    assert.deepEqual(
      discipline.primarySources.map(
        ({ id, title, authors, locator, version, versionDate }) => [
          id,
          title,
          authors,
          locator,
          version,
          versionDate,
        ],
      ),
      [
        [
          "arxiv-1412.4095v1",
          "Effective Field Theories from Soft Limits",
          ["Clifford Cheung", "Karol Kampf", "Jiri Novotny", "Jaroslav Trnka"],
          "https://arxiv.org/abs/1412.4095v1",
          "v1",
          "2014-12-12T19:32:50Z",
        ],
        [
          "arxiv-1509.03309v1",
          "On-Shell Recursion Relations for Effective Field Theories",
          [
            "Clifford Cheung",
            "Karol Kampf",
            "Jiri Novotny",
            "Chia-Hsien Shen",
            "Jaroslav Trnka",
          ],
          "https://arxiv.org/abs/1509.03309v1",
          "v1",
          "2015-09-10T20:03:45Z",
        ],
        [
          "arxiv-2406.02665v2",
          "Bootstrap Principle for the Spectrum and Scattering of Strings",
          ["Clifford Cheung", "Aaron Hillman", "Grant N. Remmen"],
          "https://arxiv.org/abs/2406.02665v2",
          "v2",
          "2025-09-09T17:09:29Z",
        ],
        [
          "arxiv-2508.09246v2",
          "Strings from Almost Nothing",
          [
            "Clifford Cheung",
            "Grant N. Remmen",
            "Francesco Sciotti",
            "Michele Tarquini",
          ],
          "https://arxiv.org/abs/2508.09246v2",
          "v2",
          "2026-06-24T15:33:06Z",
        ],
      ],
    );
  });

  it("validates the exact offline source set and refuses source or manifest drift", () => {
    const expectedBindings = [
      [
        "constructive-intelligence-tree",
        "dashboard/public/standards/constructive-intelligence-tree.v1.json",
        "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf",
      ],
      [
        "correspondence-geometry",
        "dashboard/public/standards/correspondence-geometry.v0.json",
        "f8cfeebf7404ab7e2e86b80362471cdd64015a108c47e98147e80ba7bb9e9a90",
      ],
      [
        "knowledge-methodologies",
        "x/knowledge/types/methodologies.go",
        "fa16ac33e7f2c10a19ed76541af6c2378edb79683578f2cec6f1a0563ebec386",
      ],
      [
        "knowledge-types",
        "proto/zerone/knowledge/v1/types.proto",
        "7b2b301c80711587a55ae03216728ec1f6f5bf981035106d26ac1fa4923d8ced",
      ],
    ];
    assert.deepEqual(
      canonical.sourceBindings.map(({ id, path, rawSha256 }: MutableDocument) => [
        id,
        path,
        rawSha256,
      ]),
      expectedBindings,
    );
    for (const [path, bytes] of sourceBytes()) {
      const binding = canonical.sourceBindings.find(
        (candidate: MutableDocument) => candidate.path === path,
      );
      assert.ok(binding);
      assert.equal(createHash("sha256").update(bytes).digest("hex"), binding.rawSha256);
    }

    const fetchDescriptor = Object.getOwnPropertyDescriptor(globalThis, "fetch");
    Object.defineProperty(globalThis, "fetch", {
      configurable: true,
      value: () => {
        throw new Error("offline validator attempted a network read");
      },
    });
    try {
      assert.deepEqual(
        validateExplicitInvariantDisciplineRaw(canonicalRaw, sourceBytes()),
        {
          manifestSha256: REVIEWED_SHA256,
          sourceBindingCount: 4,
          primarySourceCount: 4,
          integrationTargetCount: 5,
          notImplementedIntegrationCount: 1,
          recordCount: 4,
          familyResultCount: 2,
          noGoResultCount: 0,
          conditionalUniquenessCount: 2,
          constraintWitnessCount: 8,
          falsifierCount: 4,
          counterexampleCount: 4,
          boundaryTermCount: 4,
          proposedTransferCount: 4,
          notRunLocalTestCount: 4,
          nonTransferCount: 16,
          releaseEffectCount: 0,
        },
      );
    } finally {
      if (fetchDescriptor === undefined) {
        delete (globalThis as { fetch?: unknown }).fetch;
      } else {
        Object.defineProperty(globalThis, "fetch", fetchDescriptor);
      }
    }

    const first = canonical.sourceBindings[0] as MutableDocument;
    assert.throws(
      () =>
        validateExplicitInvariantDisciplineRaw(
          canonicalRaw,
          sourceBytes(new Map([[first.path as string, "drifted source bytes"]])),
        ),
      /source.*drift|SHA-256|digest/i,
    );
    assert.throws(
      () =>
        validateExplicitInvariantDisciplineRaw(
          canonicalRaw,
          new Map([...sourceBytes(), ["docs/unreviewed.md", "unreviewed"]]),
        ),
      /unreviewed|extra|source.*set|count/i,
    );
    const missingSource = new Map(sourceBytes());
    missingSource.delete(first.path as string);
    assert.throws(
      () => validateExplicitInvariantDisciplineRaw(canonicalRaw, missingSource),
      /missing|source.*set|count/i,
    );
    assert.throws(
      () =>
        validateExplicitInvariantDisciplineRaw(`${canonicalRaw}\n`, sourceBytes()),
      /manifest|document.*digest|SHA-256|runtime pin/i,
    );
  });

  it("authenticates original CLI bytes before a UTF-8 BOM can be stripped", () => {
    const repositoryRoot = realpathSync(
      mkdtempSync(join(tmpdir(), "zerone-eid-bom-")),
    );
    try {
      const manifestPath = join(repositoryRoot, "manifest.json");
      writeFileSync(
        manifestPath,
        Buffer.concat([
          Buffer.from([0xef, 0xbb, 0xbf]),
          Buffer.from(canonicalRaw, "utf8"),
        ]),
      );

      const result = runValidatorCli(manifestPath, repositoryRoot);
      assert.equal(result.signal, null);
      assert.equal(result.status, 1);
      assert.match(result.stderr, /document digest.*reviewed runtime pin/i);
      assert.doesNotMatch(result.stdout, /PASS/u);
    } finally {
      rmSync(repositoryRoot, { force: true, recursive: true });
    }
  });

  it(
    "refuses a manifest reached through an intermediate directory symlink",
    { skip: process.platform === "win32" },
    () => {
      const repositoryRoot = realpathSync(
        mkdtempSync(join(tmpdir(), "zerone-eid-symlink-")),
      );
      try {
        const realDirectory = join(repositoryRoot, "real");
        const linkedDirectory = join(repositoryRoot, "linked");
        mkdirSync(realDirectory);
        writeFileSync(join(realDirectory, "manifest.json"), canonicalRaw, "utf8");
        symlinkSync(realDirectory, linkedDirectory, "dir");

        const result = runValidatorCli(
          join(linkedDirectory, "manifest.json"),
          repositoryRoot,
        );
        assert.equal(result.signal, null);
        assert.equal(result.status, 1);
        assert.match(
          result.stderr,
          /manifest path contains a symbolic link/i,
        );
        assert.doesNotMatch(result.stdout, /PASS/u);
      } finally {
        rmSync(repositoryRoot, { force: true, recursive: true });
      }
    },
  );

  it("rejects unknown fields at every reviewed shape and missing required fields", () => {
    const mutations: Array<(document: MutableDocument) => void> = [
      (document) => { document.unreviewed = true; },
      (document) => { document.canonicalVocabulary.unreviewed = true; },
      (document) => { document.sourceBindings[0].unreviewed = true; },
      (document) => { document.primarySources[0].unreviewed = true; },
      (document) => { document.integrationTargets[0].unreviewed = true; },
      (document) => { document.records[0].unreviewed = true; },
      (document) => { document.records[0].sourceResult.unreviewed = true; },
      (document) => { document.records[0].sourceResult.candidateClass.unreviewed = true; },
      (document) => { document.records[0].sourceResult.regime.unreviewed = true; },
      (document) => { document.records[0].sourceResult.assumptions[0].unreviewed = true; },
      (document) => { document.records[0].sourceResult.invariants[0].unreviewed = true; },
      (document) => {
        document.records[0].sourceResult.constraintWitnesses[0].unreviewed = true;
      },
      (document) => { document.records[0].sourceResult.falsifiers[0].unreviewed = true; },
      (document) => { document.records[0].sourceResult.counterexamples[0].unreviewed = true; },
      (document) => { document.records[0].sourceResult.result.unreviewed = true; },
      (document) => { document.records[0].sourceResult.remainingFamily.unreviewed = true; },
      (document) => {
        document.records[0].sourceResult.remainingFamily.relaxationBranches[0].unreviewed =
          true;
      },
      (document) => { document.records[0].sourceResult.boundaryTerms[0].unreviewed = true; },
      (document) => { document.records[0].zeroneTransfer.unreviewed = true; },
      (document) => { document.records[0].zeroneTransfer.localTest.unreviewed = true; },
      (document) => { document.browserBoundary.unreviewed = true; },
      (document) => { document.releaseBoundary.unreviewed = false; },
    ];
    for (const mutate of mutations) {
      const document = copy();
      mutate(document);
      assert.throws(() => parseMutation(document), /unknown|missing|fields|keys/i);
    }

    const missing = copy();
    delete missing.records[0].sourceResult.allowedConclusion;
    assert.throws(() => parseMutation(missing), /missing|fields|keys/i);
  });

  it("rejects decoded duplicate keys, excessive nesting, and byte overflow", () => {
    const duplicate = canonicalRaw.replace(
      '"schema": "zerone.explicit-invariant-discipline/v1",',
      '"\\u0073chema": "zerone.explicit-invariant-discipline/v1",\n  "schema": "zerone.explicit-invariant-discipline/v1",',
    );
    assert.notEqual(duplicate, canonicalRaw);
    assert.throws(
      () => parseExplicitInvariantDisciplineJson(duplicate),
      /duplicate JSON.*key|duplicate.*schema/i,
    );

    const tooDeep = `${"[".repeat(60)}0${"]".repeat(60)}`;
    assert.throws(
      () => parseExplicitInvariantDisciplineJson(tooDeep),
      /depth|nesting/i,
    );

    const byteLength = new TextEncoder().encode(canonicalRaw).byteLength;
    const atLimit =
      canonicalRaw +
      " ".repeat(EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES - byteLength);
    assert.equal(
      new TextEncoder().encode(atLimit).byteLength,
      EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES,
    );
    assert.equal(parseExplicitInvariantDisciplineJson(atLimit).records.length, 4);
    assert.throws(
      () => parseExplicitInvariantDisciplineJson(`${atLimit} `),
      /byte limit|exceeds|too large/i,
    );

    const reordered = copy();
    const schema = reordered.schema;
    delete reordered.schema;
    reordered.schema = schema;
    assert.throws(() => parseMutation(reordered), /reordered fields/i);
  });

  it("rejects source metadata drift, reordered authors, duplicate IDs, and unsafe pins", () => {
    const mutations: Array<(document: MutableDocument) => void> = [
      (document) => { document.primarySources[0].title = "A convenient paraphrase"; },
      (document) => { document.primarySources[0].version = "v2"; },
      (document) => {
        document.primarySources[0].locator = "https://arxiv.org/abs/1412.4095";
      },
      (document) => {
        document.primarySources[0].versionDate = "2014-12-13T19:32:50Z";
      },
      (document) => {
        [document.primarySources[0].authors[0], document.primarySources[0].authors[1]] =
          [document.primarySources[0].authors[1], document.primarySources[0].authors[0]];
      },
      (document) => { document.records[1].id = document.records[0].id; },
      (document) => { document.sourceBindings[1].id = document.sourceBindings[0].id; },
      (document) => { document.sourceBindings[0].path = "docs/../private.txt"; },
      (document) => { document.sourceBindings[0].rawSha256 = "A".repeat(64); },
    ];
    for (const mutate of mutations) {
      const document = copy();
      mutate(document);
      assert.throws(
        () => parseMutation(document),
        /source|reviewed|author|version|locator|timestamp|duplicate|sorted|path|SHA-256/i,
      );
    }
  });

  it("rejects predecessor, future, and unsealed profile identities", () => {
    const mutations: Array<(document: MutableDocument) => void> = [
      (document) => { document.schema = "zerone.explicit-invariant-discipline/v0"; },
      (document) => { document.version = 2; },
      (document) => { document.snapshotDate = "2026-08-15"; },
      (document) => { document.status = "DRAFT"; },
    ];
    for (const mutate of mutations) {
      const document = copy();
      mutate(document);
      assert.throws(
        () => parseMutation(document),
        /schema|version|snapshot|status|remain|must be/i,
      );
    }
  });
});

describe("explicit invariant cross-reference and result gates", () => {
  it("rejects unresolved source, integration, invariant, witness, and result references", () => {
    const mutations: Array<(document: MutableDocument) => void> = [
      (document) => { document.integrationTargets[0].sourceRefs = ["missing-source"]; },
      (document) => {
        document.records[0].sourceResult.assumptions[0].sourceRefs = ["missing-source"];
      },
      (document) => {
        document.records[0].sourceResult.invariants[0].witnessIds = ["missing-witness"];
      },
      (document) => {
        document.records[0].sourceResult.constraintWitnesses[0].targetRefs = [
          "missing-target",
        ];
      },
      (document) => {
        document.records[0].sourceResult.constraintWitnesses[0].artifactRefs = [
          "missing-source",
        ];
      },
      (document) => {
        document.records[0].sourceResult.falsifiers[0].targetRefs = ["missing-target"];
      },
      (document) => {
        document.records[0].sourceResult.falsifiers[0].witnessRef =
          "missing-witness";
      },
      (document) => {
        document.records[0].sourceResult.counterexamples[0].targetRefs = [
          "missing-target",
        ];
      },
      (document) => {
        document.records[2].sourceResult.counterexamples[0].relaxationBranchRefs = [
          "missing-branch",
        ];
      },
      (document) => {
        document.records[0].sourceResult.remainingFamily.relaxationBranches[0].relaxedAssumptionIds =
          ["missing-assumption"];
      },
      (document) => {
        document.records[0].sourceResult.boundaryTerms[0].underAssumptionIds = [
          "missing-assumption",
        ];
      },
      (document) => {
        document.records[0].sourceResult.boundaryTerms[0].underInvariantIds = [
          "missing-invariant",
        ];
      },
      (document) => {
        document.records[0].sourceResult.boundaryTerms[0].witnessIds = [
          "missing-witness",
        ];
      },
      (document) => {
        document.records[0].sourceResult.result.underAssumptionIds = [
          "missing-assumption",
        ];
      },
      (document) => {
        document.records[0].sourceResult.result.underInvariantIds = [
          "missing-invariant",
        ];
      },
      (document) => {
        document.records[0].sourceResult.result.witnessIds = ["missing-witness"];
      },
      (document) => { document.records[0].zeroneTransfer.zeroneRefs = ["missing-target"]; },
    ];
    for (const mutate of mutations) {
      const document = copy();
      mutate(document);
      assert.throws(
        () => parseMutation(document),
        /unresolved|unknown|reference|target/i,
      );
    }

    const crossCategoryCollision = copy();
    crossCategoryCollision.records[0].sourceResult.falsifiers[0].id =
      crossCategoryCollision.records[0].sourceResult.assumptions[0].id;
    assert.throws(
      () => parseMutation(crossCategoryCollision),
      /collides|duplicate|unique/i,
    );
  });

  it("enforces typed assumptions, invariant tolerance, and positive witness status", () => {
    const unknownAssumption = copy();
    unknownAssumption.records[0].sourceResult.assumptions[0].kind = "RHETORICAL";
    assert.throws(() => parseMutation(unknownAssumption), /unsupported|assumption|kind/i);

    const exactWithTolerance = copy();
    exactWithTolerance.records[1].sourceResult.invariants[0].tolerance = "1e-9";
    assert.throws(() => parseMutation(exactWithTolerance), /tolerance|TOLERANCED/i);

    const tolerancedWithoutTolerance = copy();
    tolerancedWithoutTolerance.records[0].sourceResult.invariants[0].mode =
      "TOLERANCED";
    assert.throws(
      () => parseMutation(tolerancedWithoutTolerance),
      /tolerance|TOLERANCED/i,
    );

    for (const outcome of ["FAIL", "NOT_RUN", "INCONCLUSIVE"]) {
      const document = copy();
      document.records[2].sourceResult.constraintWitnesses[0].outcome = outcome;
      assert.throws(
        () => parseMutation(document),
        /witness|PASS|result|positive|uniqueness/i,
      );
    }
  });

  it("keeps family, no-go, and conditional-uniqueness claims inside their boundaries", () => {
    const openCandidate = copy();
    openCandidate.records[2].sourceResult.candidateClass.completeness = "OPEN";
    assert.throws(
      () => parseMutation(openCandidate),
      /CONDITIONAL_UNIQUENESS|candidate|complete|OPEN/i,
    );

    const manySolutions = copy();
    manySolutions.records[2].sourceResult.remainingFamily.cardinality = "MANY";
    assert.throws(
      () => parseMutation(manySolutions),
      /CONDITIONAL_UNIQUENESS|cardinality|ONE|family/i,
    );

    const familyOfOne = copy();
    familyOfOne.records[0].sourceResult.remainingFamily.cardinality = "ONE";
    assert.throws(
      () => parseMutation(familyOfOne),
      /FAMILY|cardinality|MANY|UNKNOWN/i,
    );

    const noGoWithMembers = copy();
    noGoWithMembers.records[0].sourceResult.result.kind = "NO_GO";
    assert.throws(
      () => parseMutation(noGoWithMembers),
      /NO_GO|cardinality|NONE|knownMembers/i,
    );

    const noGoFixture = (): MutableDocument => {
      const document = copy();
      const source = document.records[0].sourceResult;
      source.result.kind = "NO_GO";
      source.remainingFamily.cardinality = "NONE";
      source.remainingFamily.parameters = [];
      source.remainingFamily.knownMembers = [];
      source.constraintWitnesses[0].kind = "COMPLETENESS";
      source.constraintWitnesses[0].targetRefs.push(source.candidateClass.id);
      return document;
    };

    const openNoGo = noGoFixture();
    openNoGo.records[0].sourceResult.candidateClass.completeness = "OPEN";
    assert.throws(
      () => parseMutation(openNoGo),
      /OPEN cannot support NO_GO|NO_GO.*OPEN/i,
    );

    const noGoWithoutExclusion = noGoFixture();
    noGoWithoutExclusion.records[0].sourceResult.constraintWitnesses[1].kind =
      "PRESERVATION";
    assert.throws(
      () => parseMutation(noGoWithoutExclusion),
      /NO_GO requires.*exclusion and completeness witnesses/i,
    );

    const noGoWithoutCompleteness = noGoFixture();
    noGoWithoutCompleteness.records[0].sourceResult.constraintWitnesses[0].kind =
      "PRESERVATION";
    assert.throws(
      () => parseMutation(noGoWithoutCompleteness),
      /NO_GO requires.*exclusion and completeness witnesses/i,
    );

    const missingAssumption = copy();
    missingAssumption.records[2].sourceResult.result.underAssumptionIds.pop();
    assert.throws(
      () => parseMutation(missingAssumption),
      /assumption|result|complete|all/i,
    );

    const missingInvariant = copy();
    missingInvariant.records[2].sourceResult.result.underInvariantIds.pop();
    assert.throws(
      () => parseMutation(missingInvariant),
      /invariant|result|complete|all/i,
    );

    const noCompletenessWitness = copy();
    noCompletenessWitness.records[2].sourceResult.constraintWitnesses[0].kind =
      "PRESERVATION";
    assert.throws(
      () => parseMutation(noCompletenessWitness),
      /CONDITIONAL_UNIQUENESS|completeness witness/i,
    );

    const noRelaxationBranch = copy();
    noRelaxationBranch.records[2].sourceResult.remainingFamily.relaxationBranches = [];
    assert.throws(
      () => parseMutation(noRelaxationBranch),
      /relaxationBranches|must not be empty|uniqueness/i,
    );

    const twoSurvivors = copy();
    twoSurvivors.records[2].sourceResult.remainingFamily.knownMembers.push(
      "A second admitted solution",
    );
    assert.throws(
      () => parseMutation(twoSurvivors),
      /CONDITIONAL_UNIQUENESS|exactly one|knownMembers|surviving/i,
    );

    const unresolvedBoundary = copy();
    unresolvedBoundary.records[2].sourceResult.boundaryTerms[0].treatment = "UNKNOWN";
    assert.throws(
      () => parseMutation(unresolvedBoundary),
      /boundary|UNKNOWN|bound witness|CONDITIONAL_UNIQUENESS/i,
    );

    const unrelatedBoundAndTolerance = copy();
    const unrelatedSource = unrelatedBoundAndTolerance.records[2].sourceResult;
    unrelatedSource.boundaryTerms[0].treatment = "UNKNOWN";
    unrelatedSource.invariants[0].mode = "TOLERANCED";
    unrelatedSource.invariants[0].tolerance = "absolute error <= 1e-12";
    assert.throws(
      () => parseMutation(unrelatedBoundAndTolerance),
      /boundary|bound witness|toleranced invariant/i,
    );

    const unrelatedRelaxation = copy();
    const unrelatedBranch =
      unrelatedRelaxation.records[2].sourceResult.remainingFamily.relaxationBranches[0];
    unrelatedBranch.statement =
      "Database latency and CSS colors define another nonempty branch.";
    unrelatedBranch.relaxedAssumptionIds = ["bs-a2"];
    assert.throws(
      () => parseMutation(unrelatedRelaxation),
      /RELAXES_ASSUMPTION|relaxationBranches|counterexample|branch/i,
    );
  });

  it("refuses a triggered falsifier or in-scope counterexample under a positive result", () => {
    const triggered = copy();
    triggered.records[2].sourceResult.falsifiers[0].status = "TRIGGERED";
    assert.throws(
      () => parseMutation(triggered),
      /TRIGGERED|falsifier|positive|result/i,
    );

    const breakingCounterexample = copy();
    breakingCounterexample.records[2].sourceResult.counterexamples[0].disposition =
      "IN_SCOPE_BREAKS_RESULT";
    assert.throws(
      () => parseMutation(breakingCounterexample),
      /counterexample|IN_SCOPE_BREAKS_RESULT|positive|result/i,
    );
  });

  it("keeps every Zerone transfer proposed, one-way, locally unrun, and non-transferring", () => {
    for (const record of canonical.records as MutableDocument[]) {
      assert.equal(record.sourceResult.claimOwner, "SOURCE_AUTHORS");
      assert.equal(record.zeroneTransfer.claimOwner, "ZERONE");
      assert.equal(record.zeroneTransfer.relationKind, "METHODOLOGICAL_ANALOGY");
      assert.equal(record.zeroneTransfer.assessment, "PROPOSED");
      assert.equal(record.zeroneTransfer.localTest.status, "NOT_RUN");
      assert.ok(record.zeroneTransfer.nonTransfers.length >= 4);
    }

    const mutations: Array<(document: MutableDocument) => void> = [
      (document) => { document.records[0].sourceResult.claimOwner = "ZERONE"; },
      (document) => { document.records[0].zeroneTransfer.claimOwner = "SOURCE_AUTHORS"; },
      (document) => { document.records[0].zeroneTransfer.relationKind = "EQUIVALENCE"; },
      (document) => { document.records[0].zeroneTransfer.assessment = "ACCEPTED"; },
      (document) => { document.records[0].zeroneTransfer.localTest.status = "PASS"; },
      (document) => { document.records[0].zeroneTransfer.nonTransfers = []; },
      (document) => { document.records[0].zeroneTransfer.preservedDiscipline = []; },
      (document) => { document.records[0].zeroneTransfer.inverseMap = "reverse it"; },
      (document) => { document.records[0].zeroneTransfer.equivalenceScope = "everything"; },
    ];
    for (const mutate of mutations) {
      const document = copy();
      mutate(document);
      assert.throws(
        () => parseMutation(document),
        /claimOwner|METHODOLOGICAL_ANALOGY|PROPOSED|NOT_RUN|nonTransfers|must not be empty|unknown|fields/i,
      );
    }
  });

  it("rejects every attempted release effect", () => {
    assert.equal(Object.keys(canonical.releaseBoundary).length, 24);
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const document = copy();
      document.releaseBoundary[key] = true;
      assert.throws(
        () => parseMutation(document),
        new RegExp(`${key}|remain false`, "i"),
      );
    }
  });
});

describe("explicit invariant discipline bounded fetch", () => {
  it("makes exactly one restrictive request for the exact same-origin artifact", async () => {
    const calls: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
    const discipline = await fetchExplicitInvariantDiscipline({
      baseUrl: "https://zerone.ai/dashboard?ignored=true#explicit-invariants",
      fetcher: async (input, init) => {
        calls.push({ input, init });
        return jsonResponse();
      },
    });

    assert.equal(discipline.records.length, 4);
    assert.equal(calls.length, 1);
    assert.equal(
      String(calls[0]?.input),
      `https://zerone.ai${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}`,
    );
    assert.equal(new Headers(calls[0]?.init?.headers).get("accept"), "application/json");
    assert.equal(calls[0]?.init?.cache, "no-store");
    assert.equal(calls[0]?.init?.credentials, "omit");
    assert.equal(calls[0]?.init?.redirect, "error");
    assert.equal(calls[0]?.init?.referrerPolicy, "no-referrer");
    assert.ok(calls[0]?.init?.signal instanceof AbortSignal);
  });

  it("rejects redirects and every final URL except the exact origin and path", async () => {
    await assert.rejects(
      fetchExplicitInvariantDiscipline({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => jsonResponse(canonicalRaw, {}, undefined, true),
      }),
      /redirect|exact same-origin path/i,
    );

    for (const url of [
      "https://attacker.example/standards/explicit-invariant-discipline.v1.json",
      "https://zerone.ai/standards/other.json",
      `https://zerone.ai${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}?unreviewed=1`,
      `https://zerone.ai${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}#unreviewed`,
      `https://zerone.ai:444${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}`,
    ]) {
      await assert.rejects(
        fetchExplicitInvariantDiscipline({
          baseUrl: "https://zerone.ai/",
          fetcher: async () => jsonResponse(canonicalRaw, {}, url),
        }),
        /same-origin|canonical|exact.*path|response URL/i,
        url,
      );
    }
  });

  it("rejects HTTP, media-type, digest, declared-size, streamed-size, and UTF-8 drift", async () => {
    await assert.rejects(
      fetchExplicitInvariantDiscipline({
        fetcher: async () => jsonResponse("{}", { status: 503 }),
      }),
      /HTTP 503/i,
    );
    for (const contentType of [
      "text/html",
      "text/json",
      "application/jsonp",
      "application/json; garbage",
    ]) {
      await assert.rejects(
        fetchExplicitInvariantDiscipline({
          fetcher: async () =>
            jsonResponse(canonicalRaw, {
              headers: { "content-type": contentType },
            }),
        }),
        /application\/json|JSON content|content.type|media/i,
      );
    }
    await assert.rejects(
      fetchExplicitInvariantDiscipline({
        fetcher: async () => jsonResponse(`${canonicalRaw}\n`),
      }),
      /SHA-256|digest|reviewed/i,
    );
    await assert.rejects(
      fetchExplicitInvariantDiscipline({
        fetcher: async () =>
          jsonResponse("{}", {
            headers: {
              "content-length": String(
                EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES + 1,
              ),
            },
          }),
      }),
      /byte limit|exceeds|too large|size limit/i,
    );

    const oversized = new Uint8Array(
      EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES + 1,
    ).fill(0x20);
    await assert.rejects(
      fetchExplicitInvariantDiscipline({
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
      /byte limit|exceeds|too large|size limit/i,
    );
    await assert.rejects(
      fetchExplicitInvariantDiscipline({
        fetcher: async () => jsonResponse(new Uint8Array([0xff])),
      }),
      /UTF-8/i,
    );
  });

  it("aborts and cancels response bodies refused before streaming", async () => {
    const cases = [
      {
        label: "HTTP status",
        pattern: /HTTP 503/i,
        url: `https://zerone.ai${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}`,
        init: { status: 503 },
        redirected: false,
      },
      {
        label: "redirect flag",
        pattern: /redirect|exact same-origin path/i,
        url: `https://zerone.ai${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}`,
        init: {},
        redirected: true,
      },
      {
        label: "final URL",
        pattern: /same-origin|canonical|exact.*path|response URL/i,
        url: `https://attacker.example${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}`,
        init: {},
        redirected: false,
      },
      {
        label: "media type",
        pattern: /application\/json|JSON content|content.type|media/i,
        url: `https://zerone.ai${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}`,
        init: { headers: { "content-type": "text/plain" } },
        redirected: false,
      },
      {
        label: "declared size",
        pattern: /byte limit|exceeds|too large|size limit/i,
        url: `https://zerone.ai${EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT}`,
        init: {
          headers: {
            "content-length": String(
              EXPLICIT_INVARIANT_DISCIPLINE_MAX_BYTES + 1,
            ),
          },
        },
        redirected: false,
      },
    ] as const;

    for (const refusal of cases) {
      let cancelled = false;
      let signal: AbortSignal | undefined;
      const body = new ReadableStream<Uint8Array>({
        cancel() {
          cancelled = true;
        },
      });
      await assert.rejects(
        fetchExplicitInvariantDiscipline({
          baseUrl: "https://zerone.ai/",
          fetcher: async (_input, init) => {
            signal = init?.signal as AbortSignal | undefined;
            return jsonResponse(
              body,
              refusal.init,
              refusal.url,
              refusal.redirected,
            );
          },
        }),
        refusal.pattern,
        refusal.label,
      );
      await Promise.resolve();
      assert.equal(signal?.aborted, true, `${refusal.label} did not abort`);
      assert.equal(cancelled, true, `${refusal.label} did not cancel its body`);
    }
  });

  it("times out even when fetch or the response stream ignores AbortSignal", async () => {
    let stalledBodyCancelled = false;
    await assert.rejects(
      settlesWithin(
        fetchExplicitInvariantDiscipline({
          timeoutMs: 5,
          fetcher: async () => await new Promise<Response>(() => undefined),
        }),
        250,
      ),
      /timed out|timeout/i,
    );

    await assert.rejects(
      settlesWithin(
        fetchExplicitInvariantDiscipline({
          timeoutMs: 5,
          fetcher: async () =>
            jsonResponse(
              new ReadableStream<Uint8Array>({
                start() {
                  // Deliberately never enqueue or close.
                },
                cancel() {
                  stalledBodyCancelled = true;
                },
              }),
            ),
        }),
        250,
      ),
      /timed out|timeout/i,
    );
    await Promise.resolve();
    assert.equal(stalledBodyCancelled, true, "timed-out body was not cancelled");
  });
});

describe("explicit invariant renderer and dashboard integration", () => {
  it("renders hostile data only as text and filters by actual result kind", async () => {
    await withFakeDocument(async () => {
      const discipline = parseExplicitInvariantDisciplineJson(canonicalRaw);
      const hostile = '<img src=x onerror="globalThis.pwned=true"> & witness';
      discipline.records[0]!.title = hostile;
      const root = new FakeElement("div");
      renderExplicitInvariantDiscipline(
        root as unknown as HTMLElement,
        discipline,
      );

      assert.equal(root.getAttribute("aria-busy"), "false");
      const all = descendants(root);
      assert.equal(all.filter(({ tagName }) => tagName === "img").length, 0);
      assert.equal(all.filter(({ tagName }) => tagName === "script").length, 0);
      assert.ok(flattenedText(root).includes(hostile));
      const cards = all.filter((node) =>
        hasClass(node, "explicit-invariant-record"),
      );
      assert.equal(cards.length, 4);
      assert.equal(cards.filter(({ hidden }) => !hidden).length, 4);
      assert.ok(cards.every(({ dataset }) => Boolean(dataset.resultKind)));
      assert.ok(
        all.some(
          (node) =>
            node.tagName === "h3" &&
            node.textContent === "Twenty-four disabled effects",
        ),
      );
      assert.equal(
        all.filter(
          (node) => node.tagName === "code" && node.textContent === "false",
        ).length,
        24,
      );
      assert.match(
        flattenedText(root),
        /Source results and Zerone transfers remain separate|source result summaries.*author/i,
      );
      assert.match(flattenedText(root), /ZERONE/i);
      assert.match(
        flattenedText(root),
        /must not transfer|does not transfer|non.transfer/i,
      );
      const filters = all.filter((node) =>
        hasClass(node, "explicit-invariant-filter"),
      );
      assert.equal(filters.length, 3);
      assert.deepEqual(
        filters.map(({ dataset }) => dataset.resultKind).sort(),
        ["ALL", "CONDITIONAL_UNIQUENESS", "FAMILY"],
      );
      const conditional = filters.find((filter) =>
        /conditional/i.test(flattenedText(filter)),
      );
      assert.ok(conditional);
      assert.equal(conditional.getAttribute("aria-pressed"), "false");
      await conditional.click();
      assert.equal(conditional.getAttribute("aria-pressed"), "true");
      assert.equal(cards.filter(({ hidden }) => !hidden).length, 2);
      assert.ok(
        cards
          .filter(({ hidden }) => !hidden)
          .every(({ dataset }) => dataset.resultKind === "CONDITIONAL_UNIQUENESS"),
      );

      const allFilter = filters.find((filter) =>
        /^all(?: results?)?$/iu.test(flattenedText(filter).trim()),
      );
      assert.ok(allFilter);
      await allFilter.click();
      assert.equal(cards.filter(({ hidden }) => !hidden).length, 4);
    });

    const runtime = readFileSync(
      new URL("../src/explicit-invariant-discipline.ts", import.meta.url),
      "utf8",
    );
    assert.doesNotMatch(
      runtime,
      /\.innerHTML\b|\.outerHTML\b|insertAdjacentHTML|document\.write|DOMParser/u,
    );
  });

  it("renders a complete static fallback when the bounded read fails", async () => {
    await withFakeDocument(async () => {
      const descriptor = Object.getOwnPropertyDescriptor(globalThis, "fetch");
      Object.defineProperty(globalThis, "fetch", {
        configurable: true,
        value: async () => jsonResponse("unavailable", { status: 503 }),
      });
      try {
        const root = new FakeElement("div");
        await initialiseExplicitInvariantDiscipline(
          root as unknown as HTMLElement,
        );

        assert.equal(root.getAttribute("aria-busy"), "false");
        const all = descendants(root);
        assert.ok(all.some((node) => node.getAttribute("role") === "alert"));
        assert.ok(
          all.some(
            ({ tagName, href }) =>
              tagName === "a" && href === EXPLICIT_INVARIANT_DISCIPLINE_ENDPOINT,
          ),
        );
        assert.match(flattenedText(root), /read-only|no effect|raw|refus|static/i);
      } finally {
        if (descriptor === undefined) {
          delete (globalThis as { fetch?: unknown }).fetch;
        } else {
          Object.defineProperty(globalThis, "fetch", descriptor);
        }
      }
    });
  });

  it("ships all four no-JS records and wires navigation, root, and direct hash", () => {
    const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
    const main = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");
    const marker = '<div class="explicit-invariant-noscript">';
    const start = html.indexOf(marker);
    const end = html.indexOf("</noscript>", start);
    const noScript = start >= 0 && end > start ? html.slice(start, end) : undefined;

    assert.ok(noScript, "missing complete EID-1 no-JavaScript account");
    assert.equal(noScript.match(/data-record-id=/gu)?.length, 4);
    for (const record of canonical.records as MutableDocument[]) {
      assert.ok(noScript.includes(`data-record-id="${record.id as string}"`));
      assert.ok(noScript.includes(record.sourceResult.result.kind as string));
      assert.ok(noScript.includes(record.zeroneTransfer.localTest.testId as string));
    }
    for (const source of canonical.primarySources as MutableDocument[]) {
      assert.ok(noScript.includes(`href="${source.locator as string}"`));
    }
    assert.match(noScript, /four proposed analogies/i);
    assert.match(noScript, /Source result first\. Zerone test second\./i);
    assert.match(noScript, /does not transfer/i);
    assert.match(
      noScript,
      /href="\/standards\/explicit-invariant-discipline\.v1\.json"/,
    );
    assert.match(html, /href="#explicit-invariants">Invariants<\/a>/);
    assert.match(html, /id="explicit-invariant-discipline-root"/);

    assert.match(
      main,
      /initialiseExplicitInvariantDiscipline\(\s*explicitInvariantDisciplineRoot,?\s*\)/,
    );
    assert.match(main, /window\.location\.hash !== "#explicit-invariants"/);
    assert.match(
      main,
      /window\.location\.hash === "#explicit-invariants"[\s\S]*?#explicit-invariants/,
    );
    assert.match(
      main,
      /Promise\.allSettled\(\[[\s\S]*?explicitInvariantDisciplineReady,[\s\S]*?initialNetworkReady,[\s\S]*?\]\)\.then\(\(\) => \{\s*initialHashInputsSettled = true;/,
    );
  });
});
