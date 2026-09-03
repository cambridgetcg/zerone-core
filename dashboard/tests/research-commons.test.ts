import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  realpathSync,
  renameSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";

import {
  RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID,
  RESEARCH_COMMONS_ENDPOINT,
  RESEARCH_COMMONS_MAX_BYTES,
  RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH,
  RESEARCH_COMMONS_RECIPROCAL_PROFILE_SHA256,
  RESEARCH_COMMONS_RECIPROCAL_SCHEMA_PATH,
  RESEARCH_COMMONS_RECIPROCAL_SCHEMA_SHA256,
  RESEARCH_COMMONS_SHA256,
  fetchResearchCommons,
  initialiseResearchCommons,
  parseResearchCommonsJson,
  renderResearchCommons,
} from "../src/research-commons";
import {
  readBoundedRegularFile,
  validateResearchCommonsRaw,
} from "../scripts/validate-research-commons";

type MutableDocument = Record<string, any>;

const canonicalRaw = readFileSync(
  new URL("../public/standards/research-commons.v0.1.json", import.meta.url),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as MutableDocument;
const indexSource = readFileSync(new URL("../index.html", import.meta.url), "utf8");
const mainSource = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");
const runtimeSource = readFileSync(
  new URL("../src/research-commons.ts", import.meta.url),
  "utf8",
);
const validatorSource = readFileSync(
  new URL("../scripts/validate-research-commons.ts", import.meta.url),
  "utf8",
);
const stylesSource = readFileSync(
  new URL("../src/styles.css", import.meta.url),
  "utf8",
);
const headersSource = readFileSync(new URL("../public/_headers", import.meta.url), "utf8");
const repositoryRoot = realpathSync(fileURLToPath(new URL("../../", import.meta.url)));
const validatorCli = fileURLToPath(
  new URL("../scripts/validate-research-commons.ts", import.meta.url),
);
const tsxCli = fileURLToPath(new URL("../node_modules/.bin/tsx", import.meta.url));

function copy(): MutableDocument {
  return structuredClone(canonical) as MutableDocument;
}

function parseMutation(document: MutableDocument): unknown {
  return parseResearchCommonsJson(JSON.stringify(document));
}

function localBoundBytes(): ReadonlyMap<string, Uint8Array> {
  return new Map(
    [...canonical.related_artifacts, ...canonical.source_bindings].map(
      (binding: MutableDocument) => [
        binding.path as string,
        readFileSync(join(repositoryRoot, binding.path as string)),
      ],
    ),
  );
}

function jsonResponse(
  body: BodyInit | null = canonicalRaw,
  init: ResponseInit = {},
  url = `https://zerone.ai${RESEARCH_COMMONS_ENDPOINT}`,
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

class FakeElement {
  readonly children: FakeElement[] = [];
  readonly attributes = new Map<string, string>();
  readonly dataset: Record<string, string> = {};
  className = "";
  id = "";
  href = "";
  target = "";
  rel = "";
  type = "";
  disabled = false;
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
    throw new Error("Research Commons renderer must never use innerHTML");
  }

  set outerHTML(_value: string) {
    throw new Error("Research Commons renderer must never use outerHTML");
  }

  append(...nodes: Array<FakeElement | string>): void {
    for (const node of nodes) {
      if (typeof node === "string") {
        const text = new FakeElement("#text");
        text.textContent = node;
        this.children.push(text);
      } else {
        this.children.push(node);
      }
    }
  }

  replaceChildren(...nodes: Array<FakeElement | string>): void {
    this.children.splice(0);
    this.text = null;
    this.append(...nodes);
  }

  setAttribute(name: string, value: string): void {
    this.attributes.set(name, value);
    if (name === "class") this.className = value;
    if (name === "id") this.id = value;
  }

  getAttribute(name: string): string | null {
    if (name === "class") return this.className || null;
    if (name === "id") return this.id || null;
    return this.attributes.get(name) ?? null;
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

describe("Research Commons RC-0.1 manifest and offline validator", () => {
  it("pins the closed participant, ledger, funding, pilot, and effect boundaries", () => {
    const commons = parseResearchCommonsJson(canonicalRaw);
    assert.equal(commons.schema, "zerone.research-commons/v0.1");
    assert.equal(commons.status, "STATIC_SHADOW");
    assert.equal(commons.assurance, "SHADOW_ONLY");
    assert.equal(commons.architecture.planes.length, 4);
    assert.equal(
      commons.node_model.kind,
      "PASSIVE_NON_PERSON_NON_WALLET_REFERENCE_COORDINATE",
    );
    assert.equal(commons.node_model.tok_reference.request_at_block_height, 0);
    assert.equal(commons.node_model.tok_reference.runtime_requests, 0);
    assert.equal(commons.node_model.tok_reference.current_reference, null);
    assert.equal(commons.agent_model.current_participants, 0);
    assert.deepEqual(commons.ledgers.map(({ id }) => id), [
      "VALIDITY",
      "NOVELTY_PRIORITY",
      "SIGNIFICANCE_CONSEQUENCE",
      "ATTRIBUTION_CREDIT",
      "FUNDING_LIABILITY",
      "GOVERNANCE_AUTHORITY",
    ]);
    assert.deepEqual(commons.shared_ledger_profile.external_non_import_register_ids, [
      "ATTENTION_METABOLISM",
      "EXTERNAL_VALUE",
      "IDENTITY",
      "RELATIONAL_KARMA",
      "WORK_REST_OBLIGATIONS",
    ]);
    assert.equal(
      commons.shared_ledger_profile.sha256,
      "sha256:fd5ed0b66dd00b180729221a06e7fbeeb7ef6149136916842014a1afbdbc54b2",
    );
    assert.equal(commons.shared_ledger_profile.cross_pin_complete, true);
    assert.equal(commons.funding_model.activated_amount_uzrn, "0");
    assert.equal(commons.funding_model.waterfall.length, 8);
    assert.equal(commons.outcome_schedule.levels.length, 7);
    assert.equal(
      commons.outcome_schedule.levels[0]?.treatment,
      "CASE_LOCAL_PREREGISTRATION_ONLY",
    );
    assert.match(
      commons.outcome_schedule.levels[0]?.evidence ?? "",
      /caller-declared.*no trusted time, novelty, priority, or entitlement/u,
    );
    assert.deepEqual(
      commons.outcome_schedule.delivery_work_compensation.eligible_directions,
      ["POSITIVE", "NEGATIVE", "NULL", "INCONCLUSIVE", "NOT_APPLICABLE"],
    );
    assert.equal(commons.outcome_schedule.levels[3]?.label, "Declared-unproven reproduction");
    assert.equal(commons.outcome_schedule.levels[5]?.label, "Declared-unproven adoption");
    assert.equal(commons.pilot.method_input.record_id, "bootstrap-conditional-solution-space");
    assert.equal(commons.pilot.method_input.source_url, "https://arxiv.org/abs/2406.02665v2");
    assert.deepEqual(commons.pilot.candidate_class.assumption_ids, [
      "bs-a1",
      "bs-a2",
      "bs-a3",
      "bs-a4",
      "bs-a5",
      "bs-a6",
      "bs-a7",
    ]);
    assert.equal(commons.agenttool_reference.status, "SHADOW_REFERENCE");
    assert.equal(
      commons.agenttool_reference.interop_profile_id,
      "agenttool.research-commons-zerone-static-interop/0.1",
    );
    assert.equal(
      commons.agenttool_reference.agenttool_r0.main_merge_revision,
      "55342fac97250898c2c4ea884f1a03bec1f8cc8c",
    );
    assert.equal(
      commons.agenttool_reference.zerone_phase_a.source_revision,
      "5328b42230fa6945f458a6e60aca92b23eead595",
    );
    assert.equal(
      commons.agenttool_reference.zerone_phase_a.main_merge_revision,
      "fdd40bf9aca4a82b2cdd904d0161016b8c2a8667",
    );
    assert.equal(
      commons.agenttool_reference.reciprocal_cross_pin.agenttool_source_revision,
      "91a1396c76edd5e1585af33042e46640c5b5cf4a",
    );
    assert.equal(
      commons.agenttool_reference.reciprocal_cross_pin.agenttool_main_merge_revision,
      "8c63c6b4b5c14286addd29bf9da00337e43c46cd",
    );
    assert.equal(
      commons.agenttool_reference.reciprocal_cross_pin.profile_id,
      RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID,
    );
    assert.equal(
      commons.agenttool_reference.reciprocal_cross_pin.profile_local_copy,
      RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH,
    );
    assert.equal(
      commons.agenttool_reference.reciprocal_cross_pin.profile_raw_sha256,
      RESEARCH_COMMONS_RECIPROCAL_PROFILE_SHA256,
    );
    assert.equal(
      commons.agenttool_reference.reciprocal_cross_pin.schema_local_copy,
      RESEARCH_COMMONS_RECIPROCAL_SCHEMA_PATH,
    );
    assert.equal(
      commons.agenttool_reference.reciprocal_cross_pin.schema_raw_sha256,
      RESEARCH_COMMONS_RECIPROCAL_SCHEMA_SHA256,
    );
    assert.equal(commons.agenttool_reference.cross_pin_complete, true);
    assert.equal(
      commons.agenttool_reference.cross_pin_sha256,
      RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID.slice("sha256:".length),
    );
    assert.equal(commons.agenttool_reference.agenttool_wire_false_effect_count, 29);
    assert.equal(commons.agenttool_reference.zerone_surface_false_effect_count, 41);
    assert.match(
      commons.agenttool_reference.effect_boundary_relation,
      /not identical.*no field equivalence/u,
    );
    assert.match(
      commons.agenttool_reference.prior_state_transition_boundary,
      /caller-supplied prior_state.*not signatures or canonical heads.*no provenance, trusted time, global ordering, or prevention of old-state forks/u,
    );
    assert.match(
      commons.agenttool_reference.artifact_access_boundary,
      /intended open, nonexclusive access only.*fetches no artifact bytes, locator, or license.*verifies no public availability.*low-entropy sensitive material safe.*external availability, license, and safety review/u,
    );
    assert.equal(commons.agenttool_reference.integration_ready, false);
    assert.ok(commons.calls_to_action.every(({ enabled }) => !enabled));
    assert.equal(commons.release_gates[0]?.passed, true);
    assert.ok(commons.release_gates.slice(1).every(({ passed }) => !passed));
    assert.equal(commons.release_rule.all_gates_passed_is_sufficient, false);
    assert.equal(commons.release_rule.self_activating, false);
    assert.equal(commons.effect_boundary.automatic_static_gets, 1);
    assert.equal(commons.effect_boundary.uses_mainnet, false);
    assert.equal(commons.effect_boundary.invokes_bridge, false);
    assert.equal(commons.effect_boundary.invokes_adapter, false);
    assert.equal(commons.effect_boundary.offers_hosted_payout, false);
    assert.equal(commons.effect_boundary.adjudicates_result, false);
    assert.equal(
      Object.values(commons.effect_boundary).filter((value) => value === false).length,
      41,
    );
    assert.equal(commons.source_policy.local_binding_max_bytes, 262_144);
    assert.equal(commons.source_policy.local_reads_use_no_follow_descriptors, true);
    assert.equal(new TextEncoder().encode(canonicalRaw).byteLength < RESEARCH_COMMONS_MAX_BYTES, true);
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      RESEARCH_COMMONS_SHA256,
    );
  });

  it("rejects unknown, reordered, duplicate, over-deep, and semantic boundary drift", () => {
    const mutations: Array<(document: MutableDocument) => void> = [
      (document) => { document.unknown = false; },
      (document) => { document.node_model.kind = "ECONOMIC_CELL"; },
      (document) => { document.node_model.tok_reference.request_at_block_height = 1; },
      (document) => { document.agent_model.rights.inactivity_penalty = true; },
      (document) => { document.shared_ledger_profile.ledger_ids[0] = "VALIDITY"; },
      (document) => {
        document.shared_ledger_profile.external_non_import_register_ids[0] =
          "ATTENTION_METABOLISM_CHANGED";
      },
      (document) => { document.shared_ledger_profile.cross_pin_complete = false; },
      (document) => {
        document.outcome_schedule.delivery_work_compensation.eligible_directions.pop();
      },
      (document) => { document.outcome_schedule.levels[0].treatment = "CASE_LOCAL_REGISTRATION"; },
      (document) => { document.outcome_schedule.levels[3].label = "Independent"; },
      (document) => { document.agenttool_reference.integration_ready = true; },
      (document) => {
        document.agenttool_reference.reciprocal_cross_pin.profile_id = "sha256:" + "0".repeat(64);
      },
      (document) => {
        document.agenttool_reference.reciprocal_cross_pin.agenttool_main_merge_revision = "main";
      },
      (document) => { document.agenttool_reference.cross_pin_complete = false; },
      (document) => { document.calls_to_action[0].enabled = true; },
      (document) => { document.release_gates[0].passed = false; },
      (document) => { document.release_gates[1].passed = true; },
      (document) => { document.release_rule.all_gates_passed_is_sufficient = true; },
      (document) => { document.effect_boundary.uses_mainnet = true; },
    ];
    for (const mutate of mutations) {
      const document = copy();
      mutate(document);
      assert.throws(() => parseMutation(document), /must|remain|exactly|entry|field/i);
    }

    const reordered = copy();
    const schema = reordered.schema;
    delete reordered.schema;
    reordered.schema = schema;
    assert.throws(() => parseMutation(reordered), /reordered/i);

    const duplicate = canonicalRaw.replace(
      '"schema": "zerone.research-commons/v0.1",',
      '"\\u0073chema": "zerone.research-commons/v0.1",\n  "schema": "zerone.research-commons/v0.1",',
    );
    assert.throws(() => parseResearchCommonsJson(duplicate), /duplicate/i);
    assert.throws(
      () => parseResearchCommonsJson(`${"[".repeat(60)}0${"]".repeat(60)}`),
      /nesting|depth/i,
    );
    assert.throws(
      () => parseResearchCommonsJson(`${canonicalRaw}${" ".repeat(RESEARCH_COMMONS_MAX_BYTES)}`),
      /byte|exceeds/i,
    );
  });

  it("validates the seal and exact local provenance set without network access", () => {
    const fetchDescriptor = Object.getOwnPropertyDescriptor(globalThis, "fetch");
    Object.defineProperty(globalThis, "fetch", {
      configurable: true,
      value: () => {
        throw new Error("offline RC validator attempted a network read");
      },
    });
    try {
      const summary = validateResearchCommonsRaw(canonicalRaw, localBoundBytes());
      assert.equal(summary.relatedArtifactCount, 4);
      assert.equal(summary.sourceBindingCount, 6);
      assert.equal(summary.enabledActionCount, 0);
      assert.equal(summary.passedReleaseGateCount, 1);
      assert.equal(summary.automaticStaticGetCount, 1);
      assert.equal(summary.reciprocalProfileId, RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID);
    } finally {
      if (fetchDescriptor === undefined) {
        delete (globalThis as { fetch?: unknown }).fetch;
      } else {
        Object.defineProperty(globalThis, "fetch", fetchDescriptor);
      }
    }

    const bytes = new Map(localBoundBytes());
    const firstPath = canonical.related_artifacts[0].path as string;
    bytes.set(firstPath, Buffer.from("drift"));
    assert.throws(
      () => validateResearchCommonsRaw(canonicalRaw, bytes),
      /drift|SHA-256/i,
    );
    assert.throws(
      () => validateResearchCommonsRaw(`${canonicalRaw}\n`, localBoundBytes()),
      /digest|runtime pin/i,
    );
    const reciprocalDrift = new Map(localBoundBytes());
    reciprocalDrift.set(RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH, Buffer.from("{}\n"));
    assert.throws(
      () => validateResearchCommonsRaw(canonicalRaw, reciprocalDrift),
      /drift|SHA-256/i,
    );
  });

  it("runs the bounded offline validator CLI against the repository", () => {
    const result = spawnSync(
      tsxCli,
      [
        validatorCli,
        join(repositoryRoot, "dashboard/public/standards/research-commons.v0.1.json"),
        repositoryRoot,
      ],
      { encoding: "utf8", maxBuffer: 1_048_576, timeout: 5_000 },
    );
    assert.equal(result.signal, null);
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /research commons: PASS/u);
    assert.match(result.stdout, /cross-pin sha256:4d927f4db623884453f4e16b73573a81/u);
  });

  it(
    "refuses terminal/intermediate symlinks, special files, oversize files, and path swaps",
    { skip: process.platform === "win32" },
    () => {
      assert.match(
        validatorSource,
        /constants\.O_RDONLY \| constants\.O_NOFOLLOW \| constants\.O_NONBLOCK/u,
      );
      assert.equal((validatorSource.match(/fstatSync\(descriptor\)/gu) ?? []).length, 2);
      assert.doesNotMatch(validatorSource, /readFileSync\(/u);
      const root = realpathSync(mkdtempSync(join(tmpdir(), "zerone-rc-read-")));
      try {
        writeFileSync(join(root, "real.json"), "{}", "utf8");
        symlinkSync(join(root, "real.json"), join(root, "terminal.json"));
        assert.throws(
          () => readBoundedRegularFile(root, "terminal.json", true, "source", 64),
          /symbolic|no-follow/i,
        );

        mkdirSync(join(root, "real-dir"));
        writeFileSync(join(root, "real-dir", "source.json"), "{}", "utf8");
        symlinkSync(join(root, "real-dir"), join(root, "linked-dir"), "dir");
        assert.throws(
          () =>
            readBoundedRegularFile(
              root,
              "linked-dir/source.json",
              true,
              "source",
              64,
            ),
          /symbolic/i,
        );

        assert.throws(
          () => readBoundedRegularFile("/dev", "/dev/null", false, "source", 64),
          /regular/i,
        );

        writeFileSync(join(root, "large.json"), "x".repeat(65), "utf8");
        assert.throws(
          () => readBoundedRegularFile(root, "large.json", true, "source", 64),
          /byte limit|exceeds/i,
        );

        writeFileSync(join(root, "swap.json"), "original", "utf8");
        writeFileSync(join(root, "replacement.json"), "replaced", "utf8");
        assert.throws(
          () =>
            readBoundedRegularFile(
              root,
              "swap.json",
              true,
              "source",
              64,
              () => {
                renameSync(join(root, "swap.json"), join(root, "old.json"));
                renameSync(join(root, "replacement.json"), join(root, "swap.json"));
              },
            ),
          /path changed|changed while/i,
        );
      } finally {
        rmSync(root, { force: true, recursive: true });
      }
    },
  );
});

describe("Research Commons bounded browser read and renderer", () => {
  it("makes exactly one restrictive same-origin GET and accepts the reviewed bytes", async () => {
    let calls = 0;
    const commons = await fetchResearchCommons({
      baseUrl: "https://zerone.ai/#research-commons",
      fetcher: async (input, init) => {
        calls += 1;
        assert.equal(input, RESEARCH_COMMONS_ENDPOINT);
        assert.equal(init?.cache, "no-store");
        assert.equal(init?.credentials, "omit");
        assert.equal(init?.method, "GET");
        assert.equal(init?.mode, "same-origin");
        assert.equal(init?.redirect, "error");
        assert.equal(init?.referrerPolicy, "no-referrer");
        assert.equal((init?.headers as Record<string, string>).Accept, "application/json");
        return jsonResponse();
      },
    });
    assert.equal(calls, 1);
    assert.equal(commons.profile_id, "RC-0.1");
  });

  it("refuses redirect, wrong path, wrong media, oversize, and content drift", async () => {
    const attempts = [
      () => fetchResearchCommons({ fetcher: async () => jsonResponse(canonicalRaw, {}, undefined, true) }),
      () =>
        fetchResearchCommons({
          fetcher: async () => jsonResponse(canonicalRaw, {}, "https://zerone.ai/standards/other.json"),
        }),
      () =>
        fetchResearchCommons({
          fetcher: async () => jsonResponse(canonicalRaw, { headers: { "content-type": "text/plain" } }),
        }),
      () =>
        fetchResearchCommons({
          fetcher: async () =>
            jsonResponse(canonicalRaw, {
              headers: { "content-length": String(RESEARCH_COMMONS_MAX_BYTES + 1) },
            }),
        }),
      () => fetchResearchCommons({ fetcher: async () => jsonResponse(`${canonicalRaw}\n`) }),
    ];
    for (const attempt of attempts) await assert.rejects(attempt());
  });

  it("cancels every response body refused before streaming", async () => {
    let cancellations = 0;
    const trackedResponse = (
      init: ResponseInit = {},
      url = `https://zerone.ai${RESEARCH_COMMONS_ENDPOINT}`,
      redirected = false,
    ): Response =>
      jsonResponse(
        new ReadableStream<Uint8Array>({
          cancel: () => {
            cancellations += 1;
          },
        }),
        init,
        url,
        redirected,
      );
    const attempts = [
      () => fetchResearchCommons({ fetcher: async () => trackedResponse({ status: 500 }) }),
      () => fetchResearchCommons({ fetcher: async () => trackedResponse({}, undefined, true) }),
      () =>
        fetchResearchCommons({
          fetcher: async () => trackedResponse({}, "https://zerone.ai/standards/other.json"),
        }),
      () =>
        fetchResearchCommons({
          fetcher: async () => trackedResponse({ headers: { "content-type": "text/plain" } }),
        }),
      () =>
        fetchResearchCommons({
          fetcher: async () =>
            trackedResponse({
              headers: { "content-length": String(RESEARCH_COMMONS_MAX_BYTES + 1) },
            }),
        }),
    ];
    for (const attempt of attempts) await assert.rejects(attempt());
    assert.equal(cancellations, attempts.length);
  });

  it("renders as text, restores list semantics, and settles aria-busy on success", async () => {
    await withFakeDocument(async () => {
      const root = new FakeElement("div");
      let calls = 0;
      const ready = initialiseResearchCommons(root as unknown as HTMLElement, {
        baseUrl: "https://zerone.ai/#research-commons",
        fetcher: async () => {
          calls += 1;
          return jsonResponse();
        },
      });
      assert.equal(root.getAttribute("aria-busy"), "true");
      await ready;
      assert.equal(calls, 1);
      assert.equal(root.getAttribute("aria-busy"), "false");
      const renderedText = flattenedText(root);
      assert.match(renderedText, /Cases can name node-scoped questions/u);
      assert.match(renderedText, /1 compatibility gate passed.*11 closed/u);
      assert.match(renderedText, /static cross-pin.*sha256:4d927f4d/u);
      assert.equal(renderedText.split(canonical.architecture.summary).length - 1, 1);
      const markerlessLists = descendants(root).filter((node) =>
        [
          "research-commons-plane-list",
          "research-commons-ledger-list",
          "research-commons-register-list",
          "research-commons-waterfall",
          "research-commons-levels",
          "research-commons-pilot-steps",
        ].includes(node.className),
      );
      assert.equal(markerlessLists.length, 6);
      assert.ok(markerlessLists.every((node) => node.getAttribute("role") === "list"));
    });
  });

  it("settles aria-busy, shows only the refusal, and never retries on failure", async () => {
    await withFakeDocument(async () => {
      const root = new FakeElement("div");
      let calls = 0;
      await initialiseResearchCommons(root as unknown as HTMLElement, {
        fetcher: async () => {
          calls += 1;
          throw new Error("offline");
        },
      });
      assert.equal(calls, 1);
      assert.equal(root.getAttribute("aria-busy"), "false");
      assert.equal(root.children[0]?.getAttribute("role"), "alert");
      assert.match(flattenedText(root), /could not be verified/u);
      assert.doesNotMatch(flattenedText(root), /Prefund the whole promise/u);
    });
  });

  it("treats hostile copy as literal text and contains no HTML string renderer", async () => {
    await withFakeDocument(() => {
      const document = copy();
      document.purpose = '<img src=x onerror="globalThis.compromised=true">';
      const commons = parseMutation(document);
      const root = new FakeElement("div");
      renderResearchCommons(root as unknown as HTMLElement, commons as never);
      const renderedText = flattenedText(root);
      assert.match(renderedText, /<img src=x onerror=/u);
      assert.equal(renderedText.split(canonical.architecture.summary).length - 1, 1);
      assert.equal((globalThis as { compromised?: boolean }).compromised, undefined);
    });
    assert.doesNotMatch(runtimeSource, /\.innerHTML\s*=/u);
    assert.doesNotMatch(runtimeSource, /insertAdjacentHTML|DOMParser/u);
  });
});

describe("Research Commons static fallback, a11y, and hash wiring", () => {
  it("ships a substantive no-JS mirror with exact IDs, disabled actions, and list roles", () => {
    const start = indexSource.indexOf('id="research-commons"');
    const end = indexSource.indexOf('class="life-sciences-overlay"', start);
    assert.ok(start >= 0 && end > start);
    const section = indexSource.slice(start, end);
    for (const id of [
      ...canonical.ledgers.map((entry: MutableDocument) => entry.id as string),
      ...canonical.non_ledger_registers.map((entry: MutableDocument) => entry.id as string),
    ]) {
      assert.match(section, new RegExp(id, "u"));
    }
    assert.match(section, /research-commons\.six-ledger-boundary\/0\.1/u);
    assert.match(section, /fd5ed0b66dd00b180729221a06e7fbeeb7ef6149136916842014a1afbdbc54b2/u);
    assert.match(section, /<dt>Release gates<\/dt><dd>1 \/ 12<\/dd>/u);
    assert.match(section, /one compatibility gate has passed and eleven release gates remain closed/iu);
    assert.match(section, /4d927f4db623884453f4e16b73573a81b0b1cc4cc7b72529e69ca153b39112c7/u);
    assert.match(section, /static format cross-pin transfers no authority/u);
    assert.match(section, /NOT_APPLICABLE/u);
    assert.match(
      section,
      /caller-declared case-local preregistration reference[\s\S]*no trusted time, novelty, priority, or entitlement/u,
    );
    assert.match(section, /Declared-unproven reproduction/u);
    assert.match(section, /Declared-unproven adoption/u);
    assert.match(section, /29-key wire\/interop effect profile/u);
    assert.match(section, /41-key context-specific surface boundary/u);
    assert.match(
      section,
      /caller-supplied <code>prior_state<\/code> transition[\s\S]*prove no provenance,[\s\S]*trusted time, global ordering, or prevention of old-state forks/u,
    );
    assert.match(
      section,
      /artifact records declare intended open, nonexclusive access[\s\S]*fetches no artifact bytes, locator, or license[\s\S]*low-entropy sensitive material safe/u,
    );
    const noScriptStyle = indexSource.match(
      /<noscript><style>(#research-commons-root\{display:none\})<\/style><\/noscript>/u,
    );
    assert.ok(noScriptStyle?.[1]);
    const styleHash = createHash("sha256").update(noScriptStyle[1]).digest("base64");
    assert.ok(headersSource.includes(`style-src 'self' 'sha256-${styleHash}'`));
    assert.match(section, /no mainnet,\s*ZRN, uzrn, API, RPC, chain state/u);
    assert.match(section, /bridge or adapter invocation/u);
    assert.match(section, /hosted payout/u);
    assert.match(section, /adjudicates no research/u);
    assert.equal((section.match(/<(?:ol|ul)[^>]+role="list"/gu) ?? []).length, 6);
    assert.equal((section.match(/<button[^>]+disabled/gu) ?? []).length, 4);
  });

  it("calls one initializer and aligns an initial cold research-commons hash", () => {
    assert.equal((mainSource.match(/initialiseResearchCommons\(/gu) ?? []).length, 1);
    assert.match(
      mainSource,
      /const researchCommonsReady = initialiseResearchCommons\(researchCommonsRoot\);/u,
    );
    assert.match(mainSource, /window\.location\.hash !== "#research-commons"/u);
    assert.match(
      mainSource,
      /window\.location\.hash === "#research-commons"[\s\S]*researchCommonsRoot\.closest<HTMLElement>\(\s*"#research-commons",?\s*\)/u,
    );
    assert.match(
      mainSource,
      /Promise\.allSettled\(\[[\s\S]*researchCommonsReady,[\s\S]*initialNetworkReady,[\s\S]*\]\)\.then\(\(\) => \{\s*initialHashInputsSettled = true;\s*alignInitialHash\(\);/u,
    );
    assert.equal((indexSource.match(/id="research-commons-root"/gu) ?? []).length, 1);
    assert.match(
      indexSource,
      /id="research-commons-root"\s+aria-busy="true"/u,
    );
  });

  it("keeps markerless lists semantic and all RC rem microcopy at 0.72rem or larger", () => {
    for (const className of [
      "research-commons-plane-list",
      "research-commons-ledger-list",
      "research-commons-register-list",
      "research-commons-waterfall",
      "research-commons-levels",
      "research-commons-pilot-steps",
    ]) {
      const classOffset = runtimeSource.indexOf(`"${className}"`);
      const roleOffset = runtimeSource.indexOf(
        'setAttribute("role", "list")',
        classOffset,
      );
      assert.ok(classOffset >= 0 && roleOffset > classOffset && roleOffset - classOffset < 180);
    }
    const start = stylesSource.indexOf("  .research-commons-overlay {");
    const end = stylesSource.indexOf("  .life-sciences-overlay {", start);
    assert.ok(start >= 0 && end > start);
    const rcStyles = stylesSource.slice(start, end);
    assert.match(
      rcStyles,
      /body:has\(\.research-commons-noscript\) \.research-commons-root\s*\{\s*display: none;/u,
    );
    const remSizes = [...rcStyles.matchAll(/font-size:\s*([0-9.]+)rem/gu)].map(
      (match) => Number(match[1]),
    );
    assert.ok(remSizes.length > 10);
    assert.ok(remSizes.every((size) => size >= 0.72), String(remSizes));
    for (const width of [1100, 820, 620]) {
      assert.match(rcStyles, new RegExp(`@media \\(max-width: ${width}px\\)`, "u"));
    }
  });
});
