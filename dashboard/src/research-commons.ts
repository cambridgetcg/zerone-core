/// <reference lib="dom" />

export const RESEARCH_COMMONS_ENDPOINT =
  "/standards/research-commons.v0.1.json";
export const RESEARCH_COMMONS_MAX_BYTES = 65_536;
export const RESEARCH_COMMONS_SHA256 =
  "94f020f1d37faac48300d14071ec995245aedbdc7f08fc19cafd3797450cdb8c";
export const RESEARCH_COMMONS_SCHEMA = "zerone.research-commons/v0.1";
export const RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH =
  "docs/examples/agenttool-research-receipt/zerone-research-adapter-reciprocal.v0.1.json";
export const RESEARCH_COMMONS_RECIPROCAL_PROFILE_SHA256 =
  "80621747824e6c9b747d00958d2b6822bcfb76b7e11688000bc219db6177d713";
export const RESEARCH_COMMONS_RECIPROCAL_SCHEMA_PATH =
  "docs/examples/agenttool-research-receipt/zerone-research-adapter-reciprocal.v0.1.schema.json";
export const RESEARCH_COMMONS_RECIPROCAL_SCHEMA_SHA256 =
  "0b9439c39b41da19fa7a7f07539d52a53000e1f5e6c820f47e9dd8ca607e9ab2";
export const RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID =
  "sha256:4d927f4db623884453f4e16b73573a81b0b1cc4cc7b72529e69ca153b39112c7";

type JsonObject = Record<string, unknown>;

export interface ResearchCommonsPlane {
  id: string;
  order: number;
  label: string;
  owner: string;
  status: string;
  holds: string;
  refuses: string;
}

export interface ResearchCommonsLedger {
  id: string;
  order: number;
  label: string;
  holds: string;
  cannot_determine: string;
}

export interface ResearchCommonsRegisterWall {
  id: string;
  order: number;
  status: "SEPARATE_NOT_IMPORTED";
  holds: string;
  cannot_convert_into: string;
}

export interface ResearchCommonsWaterfallStep {
  id: string;
  order: number;
  allocation: string;
  release: string;
}

export interface ResearchCommonsOutcome {
  id: string;
  order: number;
  outcome_pool_bps: number;
  label: string;
  evidence: string;
  treatment: string;
}

export interface ResearchCommonsPilotStep {
  id: string;
  order: number;
  label: string;
  description: string;
}

export interface ResearchCommonsCallToAction {
  id: string;
  label: string;
  enabled: false;
  gate_reason: string;
}

export interface ResearchCommonsReleaseGate {
  id: string;
  passed: boolean;
  reason: string;
}

export interface ResearchCommons {
  schema: typeof RESEARCH_COMMONS_SCHEMA;
  version: "0.1";
  profile_id: "RC-0.1";
  status: "STATIC_SHADOW";
  assurance: "SHADOW_ONLY";
  snapshot_date: "2026-08-20";
  title: string;
  purpose: string;
  specification: "docs/specs/research-commons-rc-0.1.md";
  baseline_repository: {
    revision: "3e027cbc0e3a008df6928a1ddce77a71957e066c";
    url: "https://github.com/cambridgetcg/zerone-core/tree/3e027cbc0e3a008df6928a1ddce77a71957e066c";
    runtime_fetch: false;
    user_activated_link_only: true;
  };
  architecture: {
    shape: "THREE_PLANES_PLUS_ONE_CLOSED_WITNESS_SEAM";
    summary: string;
    planes: ResearchCommonsPlane[];
    implicit_projection: false;
    cross_plane_authority: false;
    shared_mutable_state: false;
  };
  node_model: {
    kind: "PASSIVE_NON_PERSON_NON_WALLET_REFERENCE_COORDINATE";
    current_status: "STATIC_REFERENCE_ONLY";
    reference_shape: "STATIC_TREE_DIGEST_OR_INACTIVE_CURRENT_ONLY_TOK_REQUEST";
    current_accounts: 0;
    current_balances: 0;
    current_procurements: 0;
    may_be_named_by_future_case_for: string[];
    cannot: string[];
    tok_reference: {
      status: "INACTIVE_OPTIONAL_CURRENT_ONLY_SHAPE";
      request_at_block_height: 0;
      requires_returned_chain_id: true;
      requires_returned_actual_block_height: true;
      requires_returned_tok_snapshot_root: true;
      unavailable_is_valid: true;
      runtime_requests: 0;
      current_reference: null;
    };
    procurement_requires_separately_funded_case: true;
    node_can_pay_itself: false;
    node_is_legal_or_moral_person: false;
  };
  agent_model: {
    kind: "ATTRIBUTABLE_BOUNDED_PARTICIPANT_DESIGN";
    current_participants: 0;
    current_agenttool_accounts_imported: 0;
    roles: string[];
    rights: Record<string, boolean>;
    non_completion_disposition: string;
    reviewer_compensation: Record<string, boolean>;
    inference_boundary: Record<string, boolean>;
    credential_boundary: Record<string, string | boolean>;
  };
  ledgers: ResearchCommonsLedger[];
  ledger_boundary: Record<string, false>;
  shared_ledger_profile: {
    id: "research-commons.six-ledger-boundary/0.1";
    sha256: "sha256:fd5ed0b66dd00b180729221a06e7fbeeb7ef6149136916842014a1afbdbc54b2";
    status: "PINNED_VOCABULARY_ONLY";
    ledger_ids: string[];
    external_non_import_register_ids: string[];
    values_imported: false;
    runtime_fetch: false;
    cross_pin_complete: true;
  };
  non_ledger_registers: ResearchCommonsRegisterWall[];
  commons_access: Record<string, boolean>;
  data_boundary: {
    public_safe_records: string[];
    must_remain_off_public_and_off_chain: string[];
    embargo_expiry_auto_publishes: false;
    digest_of_small_known_secret_is_public_safe: false;
    this_surface_accepts_any_data: false;
  };
  funding_model: {
    status: "UNFUNDED_REFERENCE";
    prefunded_before_case_open: true;
    one_conserved_envelope: true;
    activated_amount_uzrn: "0";
    funded_cases: 0;
    formula: string;
    conservation_rule: string;
    waterfall: ResearchCommonsWaterfallStep[];
    sponsor_selects_truth_or_verdict: false;
    claimant_self_releases: false;
    node_withdraws: false;
    reallocation_without_frozen_rule: false;
    accepts_funding_now: false;
  };
  outcome_schedule: {
    status: "REFERENCE_ONLY";
    outcome_pool_bps: 10_000;
    claimant_tranches_bps: 8_500;
    challenge_and_remediation_reserve_bps: 1_500;
    levels: ResearchCommonsOutcome[];
    delivery_work_compensation: {
      allocation: "CASE_FIXED_CAP_OUTSIDE_OUTCOME_POOL";
      eligible_directions: readonly [
        "POSITIVE",
        "NEGATIVE",
        "NULL",
        "INCONCLUSIVE",
        "NOT_APPLICABLE",
      ];
      direction_changes_amount: false;
      requires_compliant_frozen_deliverable: true;
    };
    honest_falsification_cancels_unearned_claimant_tranches: true;
    honest_falsification_can_fund_challenge_or_repair: true;
    negative_result_is_payable_work: true;
    time_alone_advances_level: false;
  };
  pilot: {
    id: "amplitude-bootstrap-garden";
    title: "Amplitude Bootstrap Garden";
    status: "DESIGN_ONLY_NOT_OPEN";
    ceiling: "E2";
    purpose: string;
    method_input: {
      status: "PINNED_METHOD_INPUT_NOT_SCIENTIFIC_RESULT_IMPORT";
      eid_schema: "zerone.explicit-invariant-discipline/v1";
      eid_path: "dashboard/public/standards/explicit-invariant-discipline.v1.json";
      eid_sha256: string;
      record_id: "bootstrap-conditional-solution-space";
      local_test_id: "eid1-conditional-solution-space";
      assessment: "PROPOSED";
      source_id: "arxiv-2406.02665v2";
      source_url: "https://arxiv.org/abs/2406.02665v2";
      source_version: "v2";
    };
    candidate_class: {
      id: "four-point-string-bootstrap-amplitudes";
      sector: string;
      assumption_ids: string[];
      falsifier_id: "bs-f1";
      falsifier: string;
      limitations: string;
      fixture_conclusion: string;
    };
    steps: ResearchCommonsPilotStep[];
    accepts_submissions: false;
    accepts_funding: false;
    runs_computation: false;
    claims_physics_result: false;
    claims_string_theory_truth: false;
    claims_universe_model: false;
    grants_qualification: false;
    grants_reward: false;
  };
  agenttool_reference: {
    status: "SHADOW_REFERENCE";
    settlement_bundle_format: "agenttool.research-settlement-bundle/0.1";
    public_projection_format: "agenttool.research-public-projection/0.1";
    zerone_shadow_receipt_schema: "zerone.agenttool-research-receipt-shadow/v0";
    interop_profile_id: "agenttool.research-commons-zerone-static-interop/0.1";
    interop_profile_path: "packages/research-commons/interop/research-commons-zerone-v0.1.json";
    agenttool_interop_raw_sha256: "8c5b1749447c1587b89b238dadb5113e10230df19fd3f4e7942d9a163aef6a8a";
    agenttool_r0: {
      repository: "https://github.com/cambridgetcg/agenttool";
      source_revision: "6a644b9e858b7d23bdea613d91412bf7310c2338";
      main_merge_revision: "55342fac97250898c2c4ea884f1a03bec1f8cc8c";
      pull_request: "https://github.com/cambridgetcg/agenttool/pull/335";
      interop_profile_permalink: string;
    };
    agenttool_wire_false_effect_count: 29;
    zerone_surface_false_effect_count: 41;
    effect_boundary_relation: string;
    prior_state_transition_boundary: string;
    artifact_access_boundary: string;
    zerone_phase_a: {
      repository: "https://github.com/cambridgetcg/zerone-core";
      source_revision: "5328b42230fa6945f458a6e60aca92b23eead595";
      main_merge_revision: "fdd40bf9aca4a82b2cdd904d0161016b8c2a8667";
      pull_request: "https://github.com/cambridgetcg/zerone-core/pull/52";
      adapter_specification: { path: string; raw_sha256: string; permalink: string };
      fixture_manifest: { path: string; raw_sha256: string; permalink: string };
    };
    reciprocal_cross_pin: {
      format: "agenttool.zerone-research-adapter-reciprocal/0.1";
      profile_id: typeof RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID;
      profile_id_algorithm: "SHA256_FORMAT_NUL_CANONICAL_JSON";
      profile_path: string;
      profile_raw_sha256: typeof RESEARCH_COMMONS_RECIPROCAL_PROFILE_SHA256;
      profile_local_copy: typeof RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH;
      profile_permalink: string;
      schema_path: string;
      schema_raw_sha256: typeof RESEARCH_COMMONS_RECIPROCAL_SCHEMA_SHA256;
      schema_local_copy: typeof RESEARCH_COMMONS_RECIPROCAL_SCHEMA_PATH;
      schema_permalink: string;
      agenttool_source_revision: "91a1396c76edd5e1585af33042e46640c5b5cf4a";
      agenttool_main_merge_revision: "8c63c6b4b5c14286addd29bf9da00337e43c46cd";
      pull_request: "https://github.com/cambridgetcg/agenttool/pull/337";
      canonical_tuple_reconstructed: true;
      independently_reviewed: true;
      compatibility_only: true;
    };
    cross_pin_sha256: "4d927f4db623884453f4e16b73573a81b0b1cc4cc7b72529e69ca153b39112c7";
    cross_pin_complete: true;
    integration_ready: false;
    imports_accounts: false;
    imports_identities: false;
    imports_receipts: false;
    calls_agenttool: false;
    cross_pin_required_before_integration_claim: true;
  };
  related_artifacts: Array<{
    id: string;
    schema: string;
    path: string;
    sha256: string;
    relationship: string;
  }>;
  source_bindings: Array<{
    id: string;
    path: string;
    sha256: string;
  }>;
  source_policy: Record<string, number | boolean>;
  non_imports: Record<string, string | boolean>;
  calls_to_action: ResearchCommonsCallToAction[];
  effect_boundary: Record<string, string | number | boolean>;
  release_rule: Record<string, boolean>;
  release_gates: ResearchCommonsReleaseGate[];
}

export interface ResearchCommonsFetchOptions {
  fetcher?: (
    input: RequestInfo | URL,
    init?: RequestInit,
  ) => Promise<Response>;
  timeoutMs?: number;
  baseUrl?: string;
}

export class ResearchCommonsDataError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ResearchCommonsDataError";
  }
}

const TOP_LEVEL_KEYS = [
  "schema",
  "version",
  "profile_id",
  "status",
  "assurance",
  "snapshot_date",
  "title",
  "purpose",
  "specification",
  "baseline_repository",
  "architecture",
  "node_model",
  "agent_model",
  "ledgers",
  "ledger_boundary",
  "shared_ledger_profile",
  "non_ledger_registers",
  "commons_access",
  "data_boundary",
  "funding_model",
  "outcome_schedule",
  "pilot",
  "agenttool_reference",
  "related_artifacts",
  "source_bindings",
  "source_policy",
  "non_imports",
  "calls_to_action",
  "effect_boundary",
  "release_rule",
  "release_gates",
] as const;

const PLANE_IDS = ["knowledge", "case", "work-economy", "witness"] as const;
const PLANE_OWNERS = [
  "ZERONE",
  "RESEARCH_COMMONS_PROFILE",
  "AGENTTOOL_CANDIDATE",
  "NO_ACTIVE_OWNER",
] as const;
const PLANE_STATUSES = [
  "READ_ONLY_REFERENCE",
  "STATIC_DESIGN_ONLY",
  "SHADOW_REFERENCE",
  "CLOSED",
] as const;
const LEDGER_IDS = [
  "VALIDITY",
  "NOVELTY_PRIORITY",
  "SIGNIFICANCE_CONSEQUENCE",
  "ATTRIBUTION_CREDIT",
  "FUNDING_LIABILITY",
  "GOVERNANCE_AUTHORITY",
] as const;
const REGISTER_IDS = [
  "WORK_REST_OBLIGATIONS",
  "ATTENTION_METABOLISM",
  "RELATIONAL_KARMA",
  "IDENTITY",
  "EXTERNAL_VALUE",
] as const;
const SHARED_LEDGER_IDS = [
  "ATTRIBUTION_CREDIT",
  "FUNDING_LIABILITY",
  "GOVERNANCE_AUTHORITY",
  "NOVELTY_PRIORITY",
  "SIGNIFICANCE_CONSEQUENCE",
  "VALIDITY",
] as const;
const SHARED_REGISTER_IDS = [
  "ATTENTION_METABOLISM",
  "EXTERNAL_VALUE",
  "IDENTITY",
  "RELATIONAL_KARMA",
  "WORK_REST_OBLIGATIONS",
] as const;
const WATERFALL = [
  ["verified-costs", "CASE_FIXED_CAP"],
  ["delivery-work-compensation", "CASE_FIXED_CAP"],
  ["claimant-milestones", "E2_TO_E6_OUTCOME_POOL_TRANCHES"],
  ["reviewer-compensation", "CASE_FIXED_CAP"],
  ["challenge-and-remediation", "FIFTEEN_PERCENT_OF_OUTCOME_POOL"],
  ["compute-and-tools", "CASE_FIXED_CAP"],
  ["administration-and-fee", "CASE_FIXED_CAP"],
  ["refundable-residual", "ALL_UNRELEASED_REMAINDER"],
] as const;
const OUTCOMES = [
  ["E0", 0, "CASE_LOCAL_PREREGISTRATION_ONLY"],
  ["E1", 0, "VERIFIED_COSTS_OUTSIDE_OUTCOME_POOL"],
  ["E2", 1_500, "CLAIMANT_TRANCHE"],
  ["E3", 2_000, "CLAIMANT_TRANCHE"],
  ["E4", 1_500, "CLAIMANT_TRANCHE"],
  ["E5", 2_500, "CLAIMANT_TRANCHE"],
  ["E6", 1_000, "CLAIMANT_TRANCHE"],
] as const;
const OUTCOME_COPY = [
  [
    "Record a caller-declared case-local preregistration reference",
    "A digest, scope, prior-art snapshot, threat model, falsifiers, disclosure policy, and caller-declared case-local preregistration commitment; proves no trusted time, novelty, priority, or entitlement.",
  ],
  [
    "Make it inspectable",
    "Complete reproducible bundle with verified case-scoped cost receipts.",
  ],
  ["Pass declared checks", "Required deterministic or class-specific checks pass."],
  [
    "Declared-unproven reproduction",
    "A reproduction is declared separate under frozen conflict rules; independent control requires external evidence not established by RC-0.1.",
  ],
  [
    "Survive challenge or repair",
    "A declared public challenge survives, or a scoped confidential repair or mitigation is separately tested; independent control requires external evidence not established by RC-0.1.",
  ],
  [
    "Declared-unproven adoption",
    "Adoption, upstream merge, maintained release, or standards disposition is declared; independent control requires external evidence not established by RC-0.1.",
  ],
  [
    "Remain conformant",
    "Continued conformance across the frozen interval or version transition.",
  ],
] as const;
const PILOT_STEPS = [
  [
    "freeze",
    "Freeze the bed",
    "Load only the digest-pinned EID-1 record, exact v2 source locator, sector, assumptions bs-a1 through bs-a7, falsifier bs-f1, and non-transfer wall.",
  ],
  [
    "commit",
    "Plant E0",
    "Record a caller-declared case-local preregistration reference for a digest-addressed public-safe question packet; it proves no trusted time, novelty, priority, or entitlement.",
  ],
  [
    "reproduce",
    "Grow E1",
    "Publish an inspectable local bundle with exact inputs, scripts, expected refusals, and bounded cost evidence.",
  ],
  [
    "check",
    "Test E2",
    "Mutate one decisive assumption and check only that the local fixture updates its result, witnesses, remaining family, and relaxation branches coherently.",
  ],
] as const;
const CTA_IDS = [
  "fund-case",
  "offer-work",
  "request-replication",
  "open-agenttool",
] as const;
const GATE_IDS = [
  "agenttool-receipt-profile-cross-pinned",
  "immutable-research-case-schema",
  "prefunding-backing-refund-proven",
  "role-separation-and-independent-quorum",
  "global-receipt-consumption",
  "privacy-and-confidential-intake",
  "dispute-appeal-and-remedy",
  "independent-governance-and-control",
  "local-airgapped-verifier",
  "external-payout-backing",
  "adversarial-and-independent-review",
  "separately-authorized-release",
] as const;

const RELATED_ARTIFACTS = [
  [
    "branch-flow-v1",
    "zerone.constructive-intelligence-branch-flow/v1",
    "dashboard/public/standards/constructive-intelligence-branch-flow.v1.json",
    "6b83912450fec94772dad8a7bde11c980c4dea6bdb6b867e495e4480d4cc55aa",
    "RELATED_STATIC_POLICY_NOT_IMPORTED",
  ],
  [
    "constructive-tree-v1",
    "zerone.constructive-intelligence-tree/v1",
    "dashboard/public/standards/constructive-intelligence-tree.v1.json",
    "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf",
    "VERSIONED_NODE_REFERENCE_ONLY",
  ],
  [
    "correspondence-geometry-v0",
    "zerone.correspondence-geometry/v0",
    "dashboard/public/standards/correspondence-geometry.v0.json",
    "f8cfeebf7404ab7e2e86b80362471cdd64015a108c47e98147e80ba7bb9e9a90",
    "RELATED_READING_DISCIPLINE_NOT_IMPORTED",
  ],
  [
    "explicit-invariant-discipline-v1",
    "zerone.explicit-invariant-discipline/v1",
    "dashboard/public/standards/explicit-invariant-discipline.v1.json",
    "e60b89cbed8eb26d3fad0ee45ef8c433391341f3abb4865af2755595815354df",
    "PINNED_METHOD_INPUT_WITHOUT_SCIENTIFIC_RESULT_IMPORT",
  ],
] as const;
const SOURCE_BINDINGS = [
  [
    "constructive-tree-spec",
    "docs/specs/constructive-intelligence-tree-v1.md",
    "dbfa325d4ca331feee3a24d587d539c83ccd04223d181994f0ef571a96cc5fdb",
  ],
  [
    "frontier-commons-spec",
    "docs/specs/frontier-commons-participation-v0.md",
    "cbe9f60c9e085c39d4815ef5436369569edb0925f9f3cfdfdf33f11c873e4df5",
  ],
  [
    "money-karma-doctrine",
    "docs/constitution/MONEY-KARMA.md",
    "9d91d53c90a592882438d52278701328f2fb46077d8f13b6d2c8984d1c48d637",
  ],
  [
    "tok-substrate-doctrine",
    "docs/TOK_SUBSTRATE.md",
    "4fec6e3a410d5736f61cd43f4d9c421380b93f649c2f0d026a5f4e68a6534328",
  ],
  [
    "agenttool-reciprocal-profile",
    RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH,
    RESEARCH_COMMONS_RECIPROCAL_PROFILE_SHA256,
  ],
  [
    "agenttool-reciprocal-schema",
    RESEARCH_COMMONS_RECIPROCAL_SCHEMA_PATH,
    RESEARCH_COMMONS_RECIPROCAL_SCHEMA_SHA256,
  ],
] as const;

function fail(path: string, message: string): never {
  throw new ResearchCommonsDataError(`${path}: ${message}`);
}

function object(value: unknown, path: string): JsonObject {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  return value as JsonObject;
}

function exactKeys(
  value: JsonObject,
  expected: readonly string[],
  path: string,
): void {
  const actual = Object.keys(value);
  const expectedSet = new Set(expected);
  for (const key of actual) {
    if (!expectedSet.has(key)) fail(`${path}.${key}`, "is an unknown field");
  }
  for (const key of expected) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "is missing");
  }
  if (actual.some((key, index) => key !== expected[index])) {
    fail(path, "fields are reordered from the reviewed schema");
  }
}

function literal<T extends string | number | boolean | null>(
  value: unknown,
  expected: T,
  path: string,
): T {
  if (value !== expected) fail(path, `must remain ${String(expected)}`);
  return expected;
}

function text(value: unknown, path: string, maximum = 2_048): string {
  if (typeof value !== "string" || value.length === 0 || value.length > maximum) {
    fail(path, "must be a bounded non-empty string");
  }
  return value;
}

function array(value: unknown, path: string, length: number): unknown[] {
  if (!Array.isArray(value) || value.length !== length) {
    fail(path, `must contain exactly ${length} entries`);
  }
  return value;
}

function stringArray(
  value: unknown,
  path: string,
  expected: readonly string[],
): string[] {
  const items = array(value, path, expected.length);
  expected.forEach((entry, index) => literal(items[index], entry, `${path}[${index}]`));
  return [...expected];
}

function allFlags(
  value: unknown,
  expected: Readonly<Record<string, boolean>>,
  path: string,
): void {
  const flags = object(value, path);
  const keys = Object.keys(expected);
  exactKeys(flags, keys, path);
  for (const key of keys) literal(flags[key], expected[key]!, `${path}.${key}`);
}

function rejectDuplicateKeysAndDepth(raw: string): void {
  let offset = 0;
  const whitespace = (): void => {
    while (/\s/u.test(raw[offset] ?? "")) offset += 1;
  };
  const scanString = (): string => {
    const start = offset;
    offset += 1;
    let escaped = false;
    while (offset < raw.length) {
      const character = raw[offset];
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset)) as string;
      }
      offset += 1;
    }
    fail("$", "contains an unterminated JSON string");
  };
  const scanValue = (path: string, depth: number): void => {
    if (depth > 48) fail(path, "JSON nesting exceeds 48");
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
        if (raw[offset] !== '"') fail(path, "contains a malformed object key");
        const key = scanString();
        if (keys.has(key)) fail(`${path}.${key}`, "is a duplicate JSON key");
        keys.add(key);
        whitespace();
        if (raw[offset] !== ":") fail(path, "contains a malformed object separator");
        offset += 1;
        scanValue(`${path}.${key}`, depth + 1);
        whitespace();
        if (raw[offset] === "}") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "contains a malformed object delimiter");
        offset += 1;
      }
      fail(path, "contains an unterminated object");
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
        scanValue(`${path}[${index}]`, depth + 1);
        whitespace();
        if (raw[offset] === "]") {
          offset += 1;
          return;
        }
        if (raw[offset] !== ",") fail(path, "contains a malformed array delimiter");
        offset += 1;
        index += 1;
      }
      fail(path, "contains an unterminated array");
    }
    if (token === '"') {
      scanString();
      return;
    }
    const start = offset;
    while (offset < raw.length && !/[\s,\]}]/u.test(raw[offset] ?? "")) offset += 1;
    if (offset === start) fail(path, "contains a malformed JSON value");
  };
  scanValue("$", 0);
  whitespace();
  if (offset !== raw.length) fail("$", "contains trailing JSON data");
}

export function parseResearchCommons(value: unknown): ResearchCommons {
  const root = object(value, "$");
  exactKeys(root, TOP_LEVEL_KEYS, "$");
  literal(root.schema, RESEARCH_COMMONS_SCHEMA, "$.schema");
  literal(root.version, "0.1", "$.version");
  literal(root.profile_id, "RC-0.1", "$.profile_id");
  literal(root.status, "STATIC_SHADOW", "$.status");
  literal(root.assurance, "SHADOW_ONLY", "$.assurance");
  literal(root.snapshot_date, "2026-08-20", "$.snapshot_date");
  text(root.title, "$.title", 160);
  text(root.purpose, "$.purpose");
  literal(
    root.specification,
    "docs/specs/research-commons-rc-0.1.md",
    "$.specification",
  );

  const repository = object(root.baseline_repository, "$.baseline_repository");
  exactKeys(
    repository,
    ["revision", "url", "runtime_fetch", "user_activated_link_only"],
    "$.baseline_repository",
  );
  literal(
    repository.revision,
    "3e027cbc0e3a008df6928a1ddce77a71957e066c",
    "$.baseline_repository.revision",
  );
  literal(
    repository.url,
    "https://github.com/cambridgetcg/zerone-core/tree/3e027cbc0e3a008df6928a1ddce77a71957e066c",
    "$.baseline_repository.url",
  );
  literal(repository.runtime_fetch, false, "$.baseline_repository.runtime_fetch");
  literal(
    repository.user_activated_link_only,
    true,
    "$.baseline_repository.user_activated_link_only",
  );

  const architecture = object(root.architecture, "$.architecture");
  exactKeys(
    architecture,
    [
      "shape",
      "summary",
      "planes",
      "implicit_projection",
      "cross_plane_authority",
      "shared_mutable_state",
    ],
    "$.architecture",
  );
  literal(
    architecture.shape,
    "THREE_PLANES_PLUS_ONE_CLOSED_WITNESS_SEAM",
    "$.architecture.shape",
  );
  text(architecture.summary, "$.architecture.summary");
  const planes = array(architecture.planes, "$.architecture.planes", 4);
  planes.forEach((entry, index) => {
    const path = `$.architecture.planes[${index}]`;
    const plane = object(entry, path);
    exactKeys(plane, ["id", "order", "label", "owner", "status", "holds", "refuses"], path);
    literal(plane.id, PLANE_IDS[index]!, `${path}.id`);
    literal(plane.order, index, `${path}.order`);
    text(plane.label, `${path}.label`, 80);
    literal(plane.owner, PLANE_OWNERS[index]!, `${path}.owner`);
    literal(plane.status, PLANE_STATUSES[index]!, `${path}.status`);
    text(plane.holds, `${path}.holds`);
    text(plane.refuses, `${path}.refuses`);
  });
  for (const key of ["implicit_projection", "cross_plane_authority", "shared_mutable_state"] as const) {
    literal(architecture[key], false, `$.architecture.${key}`);
  }

  const node = object(root.node_model, "$.node_model");
  exactKeys(
    node,
    [
      "kind",
      "current_status",
      "reference_shape",
      "current_accounts",
      "current_balances",
      "current_procurements",
      "may_be_named_by_future_case_for",
      "cannot",
      "tok_reference",
      "procurement_requires_separately_funded_case",
      "node_can_pay_itself",
      "node_is_legal_or_moral_person",
    ],
    "$.node_model",
  );
  literal(
    node.kind,
    "PASSIVE_NON_PERSON_NON_WALLET_REFERENCE_COORDINATE",
    "$.node_model.kind",
  );
  literal(node.current_status, "STATIC_REFERENCE_ONLY", "$.node_model.current_status");
  literal(
    node.reference_shape,
    "STATIC_TREE_DIGEST_OR_INACTIVE_CURRENT_ONLY_TOK_REQUEST",
    "$.node_model.reference_shape",
  );
  for (const key of ["current_accounts", "current_balances", "current_procurements"] as const) {
    literal(node[key], 0, `$.node_model.${key}`);
  }
  stringArray(
    node.may_be_named_by_future_case_for,
    "$.node_model.may_be_named_by_future_case_for",
    [
      "one-evidence-gap",
      "one-earmarked-budget-destination",
      "one-case-fixed-maintenance-reserve",
      "one-revalidation-obligation-under-frozen-policy",
    ],
  );
  stringArray(
    node.cannot,
    "$.node_model.cannot",
    [
      "sign",
      "hold-a-private-key",
      "act-as-a-wallet",
      "withdraw",
      "borrow",
      "vote",
      "own-a-person-or-agent",
      "receive-personhood",
      "gain-authority-from-funding",
      "convert-attention-or-karma-into-money",
    ],
  );
  const tokReference = object(node.tok_reference, "$.node_model.tok_reference");
  exactKeys(
    tokReference,
    [
      "status",
      "request_at_block_height",
      "requires_returned_chain_id",
      "requires_returned_actual_block_height",
      "requires_returned_tok_snapshot_root",
      "unavailable_is_valid",
      "runtime_requests",
      "current_reference",
    ],
    "$.node_model.tok_reference",
  );
  literal(
    tokReference.status,
    "INACTIVE_OPTIONAL_CURRENT_ONLY_SHAPE",
    "$.node_model.tok_reference.status",
  );
  literal(
    tokReference.request_at_block_height,
    0,
    "$.node_model.tok_reference.request_at_block_height",
  );
  for (const key of [
    "requires_returned_chain_id",
    "requires_returned_actual_block_height",
    "requires_returned_tok_snapshot_root",
    "unavailable_is_valid",
  ] as const) {
    literal(tokReference[key], true, `$.node_model.tok_reference.${key}`);
  }
  literal(tokReference.runtime_requests, 0, "$.node_model.tok_reference.runtime_requests");
  literal(tokReference.current_reference, null, "$.node_model.tok_reference.current_reference");
  literal(
    node.procurement_requires_separately_funded_case,
    true,
    "$.node_model.procurement_requires_separately_funded_case",
  );
  literal(node.node_can_pay_itself, false, "$.node_model.node_can_pay_itself");
  literal(
    node.node_is_legal_or_moral_person,
    false,
    "$.node_model.node_is_legal_or_moral_person",
  );

  const agent = object(root.agent_model, "$.agent_model");
  exactKeys(
    agent,
    [
      "kind",
      "current_participants",
      "current_agenttool_accounts_imported",
      "roles",
      "rights",
      "non_completion_disposition",
      "reviewer_compensation",
      "inference_boundary",
      "credential_boundary",
    ],
    "$.agent_model",
  );
  literal(agent.kind, "ATTRIBUTABLE_BOUNDED_PARTICIPANT_DESIGN", "$.agent_model.kind");
  literal(agent.current_participants, 0, "$.agent_model.current_participants");
  literal(
    agent.current_agenttool_accounts_imported,
    0,
    "$.agent_model.current_agenttool_accounts_imported",
  );
  stringArray(
    agent.roles,
    "$.agent_model.roles",
    [
      "finder-or-prover",
      "replicator",
      "reviewer",
      "challenger",
      "repairer",
      "integrator",
      "maintainer",
      "trainer-or-extractor",
    ],
  );
  allFlags(
    agent.rights,
    {
      decline_without_penalty: true,
      rest_without_penalty: true,
      pause_without_penalty: true,
      exit_without_penalty: true,
      inactivity_penalty: false,
      exit_penalty: false,
      non_completion_reputation_penalty: false,
      unearned_reservation_becomes_debt: false,
      earned_compensation_can_be_revoked_for_rest_or_exit: false,
    },
    "$.agent_model.rights",
  );
  text(agent.non_completion_disposition, "$.agent_model.non_completion_disposition");
  allFlags(
    agent.reviewer_compensation,
    {
      fixed_and_verdict_independent: true,
      outcome_bonus: false,
      outcome_slashing: false,
      sponsor_controls_verdict: false,
      claimant_controls_verdict: false,
    },
    "$.agent_model.reviewer_compensation",
  );
  allFlags(
    agent.inference_boundary,
    {
      infers_identity: false,
      infers_controller: false,
      infers_consent: false,
      infers_consciousness: false,
      infers_personhood: false,
      responsible_controllers_displaced: false,
    },
    "$.agent_model.inference_boundary",
  );
  const credential = object(agent.credential_boundary, "$.agent_model.credential_boundary");
  exactKeys(
    credential,
    [
      "did_key_wallet_account_agent_are_distinct",
      "none_proves_independent_controller",
      "address_count_is_independence",
      "signature_count_is_independence",
      "e3_independence_status",
      "independence_requires_external_evidence",
    ],
    "$.agent_model.credential_boundary",
  );
  literal(
    credential.did_key_wallet_account_agent_are_distinct,
    true,
    "$.agent_model.credential_boundary.did_key_wallet_account_agent_are_distinct",
  );
  literal(
    credential.none_proves_independent_controller,
    true,
    "$.agent_model.credential_boundary.none_proves_independent_controller",
  );
  literal(
    credential.address_count_is_independence,
    false,
    "$.agent_model.credential_boundary.address_count_is_independence",
  );
  literal(
    credential.signature_count_is_independence,
    false,
    "$.agent_model.credential_boundary.signature_count_is_independence",
  );
  literal(
    credential.e3_independence_status,
    "DECLARED_UNPROVEN",
    "$.agent_model.credential_boundary.e3_independence_status",
  );
  literal(
    credential.independence_requires_external_evidence,
    true,
    "$.agent_model.credential_boundary.independence_requires_external_evidence",
  );

  const ledgers = array(root.ledgers, "$.ledgers", LEDGER_IDS.length);
  ledgers.forEach((entry, index) => {
    const path = `$.ledgers[${index}]`;
    const ledger = object(entry, path);
    exactKeys(ledger, ["id", "order", "label", "holds", "cannot_determine"], path);
    literal(ledger.id, LEDGER_IDS[index]!, `${path}.id`);
    literal(ledger.order, index, `${path}.order`);
    text(ledger.label, `${path}.label`, 80);
    text(ledger.holds, `${path}.holds`);
    text(ledger.cannot_determine, `${path}.cannot_determine`);
    if (index === 1) {
      literal(
        ledger.holds,
        "Case-scoped prior-art and caller-declared timestamp references only, plus declared overlap with earlier deliverables; no trusted time, novelty, priority, or precedence adjudication.",
        `${path}.holds`,
      );
    }
  });
  allFlags(
    root.ledger_boundary,
    {
      shared_unit: false,
      cross_ledger_arithmetic: false,
      cross_ledger_inference: false,
      implicit_conversion: false,
      shared_scalar_score: false,
      person_ranking: false,
      money_buys_truth: false,
      attention_buys_priority: false,
      karma_buys_voice: false,
    },
    "$.ledger_boundary",
  );
  const sharedLedgerProfile = object(
    root.shared_ledger_profile,
    "$.shared_ledger_profile",
  );
  exactKeys(
    sharedLedgerProfile,
    [
      "id",
      "sha256",
      "status",
      "ledger_ids",
      "external_non_import_register_ids",
      "values_imported",
      "runtime_fetch",
      "cross_pin_complete",
    ],
    "$.shared_ledger_profile",
  );
  literal(
    sharedLedgerProfile.id,
    "research-commons.six-ledger-boundary/0.1",
    "$.shared_ledger_profile.id",
  );
  literal(
    sharedLedgerProfile.sha256,
    "sha256:fd5ed0b66dd00b180729221a06e7fbeeb7ef6149136916842014a1afbdbc54b2",
    "$.shared_ledger_profile.sha256",
  );
  literal(
    sharedLedgerProfile.status,
    "PINNED_VOCABULARY_ONLY",
    "$.shared_ledger_profile.status",
  );
  stringArray(
    sharedLedgerProfile.ledger_ids,
    "$.shared_ledger_profile.ledger_ids",
    SHARED_LEDGER_IDS,
  );
  stringArray(
    sharedLedgerProfile.external_non_import_register_ids,
    "$.shared_ledger_profile.external_non_import_register_ids",
    SHARED_REGISTER_IDS,
  );
  for (const key of ["values_imported", "runtime_fetch"] as const) {
    literal(sharedLedgerProfile[key], false, `$.shared_ledger_profile.${key}`);
  }
  literal(
    sharedLedgerProfile.cross_pin_complete,
    true,
    "$.shared_ledger_profile.cross_pin_complete",
  );
  const registers = array(
    root.non_ledger_registers,
    "$.non_ledger_registers",
    REGISTER_IDS.length,
  );
  registers.forEach((entry, index) => {
    const path = `$.non_ledger_registers[${index}]`;
    const register = object(entry, path);
    exactKeys(
      register,
      ["id", "order", "status", "holds", "cannot_convert_into"],
      path,
    );
    literal(register.id, REGISTER_IDS[index]!, `${path}.id`);
    literal(register.order, index, `${path}.order`);
    literal(register.status, "SEPARATE_NOT_IMPORTED", `${path}.status`);
    text(register.holds, `${path}.holds`);
    text(register.cannot_convert_into, `${path}.cannot_convert_into`);
  });
  allFlags(
    root.commons_access,
    {
      public_outputs_remain_open: true,
      public_knowledge_paywall: false,
      exclusive_access_sale: false,
      payment_buys_bounded_labor_or_review_only: true,
      payment_buys_truth: false,
      payment_buys_ownership: false,
      payment_buys_graph_priority: false,
      payment_buys_authority: false,
    },
    "$.commons_access",
  );
  const dataBoundary = object(root.data_boundary, "$.data_boundary");
  exactKeys(
    dataBoundary,
    [
      "public_safe_records",
      "must_remain_off_public_and_off_chain",
      "embargo_expiry_auto_publishes",
      "digest_of_small_known_secret_is_public_safe",
      "this_surface_accepts_any_data",
    ],
    "$.data_boundary",
  );
  stringArray(
    dataBoundary.public_safe_records,
    "$.data_boundary.public_safe_records",
    [
      "versioned-public-claims",
      "safe-artifact-digests",
      "typed-public-challenges",
      "public-safe-settlement-projections",
    ],
  );
  stringArray(
    dataBoundary.must_remain_off_public_and_off_chain,
    "$.data_boundary.must_remain_off_public_and_off_chain",
    [
      "sensitive-bytes",
      "embargoed-evidence",
      "human-subject-data",
      "security-exploit-details",
      "dual-use-operational-details",
      "raw-personal-data",
      "secrets-and-credentials",
    ],
  );
  for (const key of [
    "embargo_expiry_auto_publishes",
    "digest_of_small_known_secret_is_public_safe",
    "this_surface_accepts_any_data",
  ] as const) {
    literal(dataBoundary[key], false, `$.data_boundary.${key}`);
  }

  const funding = object(root.funding_model, "$.funding_model");
  exactKeys(
    funding,
    [
      "status",
      "prefunded_before_case_open",
      "one_conserved_envelope",
      "activated_amount_uzrn",
      "funded_cases",
      "formula",
      "conservation_rule",
      "waterfall",
      "sponsor_selects_truth_or_verdict",
      "claimant_self_releases",
      "node_withdraws",
      "reallocation_without_frozen_rule",
      "accepts_funding_now",
    ],
    "$.funding_model",
  );
  literal(funding.status, "UNFUNDED_REFERENCE", "$.funding_model.status");
  literal(funding.prefunded_before_case_open, true, "$.funding_model.prefunded_before_case_open");
  literal(funding.one_conserved_envelope, true, "$.funding_model.one_conserved_envelope");
  literal(funding.activated_amount_uzrn, "0", "$.funding_model.activated_amount_uzrn");
  literal(funding.funded_cases, 0, "$.funding_model.funded_cases");
  literal(
    funding.formula,
    "funded_envelope = verified_costs + delivery_work_compensation + claimant_milestones + reviewer_compensation + challenge_and_remediation + compute_and_tools + administration_and_fee + refundable_residual",
    "$.funding_model.formula",
  );
  literal(
    funding.conservation_rule,
    "sum(released_allocations) + refundable_residual = funded_envelope",
    "$.funding_model.conservation_rule",
  );
  const waterfall = array(funding.waterfall, "$.funding_model.waterfall", WATERFALL.length);
  waterfall.forEach((entry, index) => {
    const path = `$.funding_model.waterfall[${index}]`;
    const step = object(entry, path);
    exactKeys(step, ["id", "order", "allocation", "release"], path);
    literal(step.id, WATERFALL[index]![0], `${path}.id`);
    literal(step.order, index, `${path}.order`);
    literal(step.allocation, WATERFALL[index]![1], `${path}.allocation`);
    text(step.release, `${path}.release`);
    if (index === 2) {
      literal(
        step.release,
        "Only when the predeclared evidence gate, declared separation, and quorum requirements pass; independent control requires external evidence not established by RC-0.1.",
        `${path}.release`,
      );
    }
  });
  for (const key of [
    "sponsor_selects_truth_or_verdict",
    "claimant_self_releases",
    "node_withdraws",
    "reallocation_without_frozen_rule",
    "accepts_funding_now",
  ] as const) {
    literal(funding[key], false, `$.funding_model.${key}`);
  }

  const schedule = object(root.outcome_schedule, "$.outcome_schedule");
  exactKeys(
    schedule,
    [
      "status",
      "outcome_pool_bps",
      "claimant_tranches_bps",
      "challenge_and_remediation_reserve_bps",
      "levels",
      "delivery_work_compensation",
      "honest_falsification_cancels_unearned_claimant_tranches",
      "honest_falsification_can_fund_challenge_or_repair",
      "negative_result_is_payable_work",
      "time_alone_advances_level",
    ],
    "$.outcome_schedule",
  );
  literal(schedule.status, "REFERENCE_ONLY", "$.outcome_schedule.status");
  literal(schedule.outcome_pool_bps, 10_000, "$.outcome_schedule.outcome_pool_bps");
  literal(schedule.claimant_tranches_bps, 8_500, "$.outcome_schedule.claimant_tranches_bps");
  literal(
    schedule.challenge_and_remediation_reserve_bps,
    1_500,
    "$.outcome_schedule.challenge_and_remediation_reserve_bps",
  );
  const levels = array(schedule.levels, "$.outcome_schedule.levels", OUTCOMES.length);
  levels.forEach((entry, index) => {
    const path = `$.outcome_schedule.levels[${index}]`;
    const level = object(entry, path);
    exactKeys(level, ["id", "order", "outcome_pool_bps", "label", "evidence", "treatment"], path);
    literal(level.id, OUTCOMES[index]![0], `${path}.id`);
    literal(level.order, index, `${path}.order`);
    literal(level.outcome_pool_bps, OUTCOMES[index]![1], `${path}.outcome_pool_bps`);
    literal(level.label, OUTCOME_COPY[index]![0], `${path}.label`);
    literal(level.evidence, OUTCOME_COPY[index]![1], `${path}.evidence`);
    literal(level.treatment, OUTCOMES[index]![2], `${path}.treatment`);
  });
  const claimantTotal = OUTCOMES.reduce((sum, entry) => sum + entry[1], 0);
  if (claimantTotal !== 8_500 || claimantTotal + 1_500 !== 10_000) {
    fail("$.outcome_schedule", "does not conserve the outcome pool");
  }
  const deliveryPay = object(
    schedule.delivery_work_compensation,
    "$.outcome_schedule.delivery_work_compensation",
  );
  exactKeys(
    deliveryPay,
    [
      "allocation",
      "eligible_directions",
      "direction_changes_amount",
      "requires_compliant_frozen_deliverable",
    ],
    "$.outcome_schedule.delivery_work_compensation",
  );
  literal(
    deliveryPay.allocation,
    "CASE_FIXED_CAP_OUTSIDE_OUTCOME_POOL",
    "$.outcome_schedule.delivery_work_compensation.allocation",
  );
  stringArray(
    deliveryPay.eligible_directions,
    "$.outcome_schedule.delivery_work_compensation.eligible_directions",
    ["POSITIVE", "NEGATIVE", "NULL", "INCONCLUSIVE", "NOT_APPLICABLE"],
  );
  literal(
    deliveryPay.direction_changes_amount,
    false,
    "$.outcome_schedule.delivery_work_compensation.direction_changes_amount",
  );
  literal(
    deliveryPay.requires_compliant_frozen_deliverable,
    true,
    "$.outcome_schedule.delivery_work_compensation.requires_compliant_frozen_deliverable",
  );
  literal(
    schedule.honest_falsification_cancels_unearned_claimant_tranches,
    true,
    "$.outcome_schedule.honest_falsification_cancels_unearned_claimant_tranches",
  );
  literal(
    schedule.honest_falsification_can_fund_challenge_or_repair,
    true,
    "$.outcome_schedule.honest_falsification_can_fund_challenge_or_repair",
  );
  literal(
    schedule.negative_result_is_payable_work,
    true,
    "$.outcome_schedule.negative_result_is_payable_work",
  );
  literal(schedule.time_alone_advances_level, false, "$.outcome_schedule.time_alone_advances_level");

  const pilot = object(root.pilot, "$.pilot");
  exactKeys(
    pilot,
    [
      "id",
      "title",
      "status",
      "ceiling",
      "purpose",
      "method_input",
      "candidate_class",
      "steps",
      "accepts_submissions",
      "accepts_funding",
      "runs_computation",
      "claims_physics_result",
      "claims_string_theory_truth",
      "claims_universe_model",
      "grants_qualification",
      "grants_reward",
    ],
    "$.pilot",
  );
  literal(pilot.id, "amplitude-bootstrap-garden", "$.pilot.id");
  literal(pilot.title, "Amplitude Bootstrap Garden", "$.pilot.title");
  literal(pilot.status, "DESIGN_ONLY_NOT_OPEN", "$.pilot.status");
  literal(pilot.ceiling, "E2", "$.pilot.ceiling");
  literal(
    pilot.purpose,
    "A first low-risk garden for testing the structural coherence of one exact EID-1 conditional-solution-space fixture without rerunning or claiming the cited scientific derivation.",
    "$.pilot.purpose",
  );
  const methodInput = object(pilot.method_input, "$.pilot.method_input");
  exactKeys(
    methodInput,
    [
      "status",
      "eid_schema",
      "eid_path",
      "eid_sha256",
      "record_id",
      "local_test_id",
      "assessment",
      "source_id",
      "source_url",
      "source_version",
    ],
    "$.pilot.method_input",
  );
  literal(
    methodInput.status,
    "PINNED_METHOD_INPUT_NOT_SCIENTIFIC_RESULT_IMPORT",
    "$.pilot.method_input.status",
  );
  literal(
    methodInput.eid_schema,
    "zerone.explicit-invariant-discipline/v1",
    "$.pilot.method_input.eid_schema",
  );
  literal(
    methodInput.eid_path,
    "dashboard/public/standards/explicit-invariant-discipline.v1.json",
    "$.pilot.method_input.eid_path",
  );
  literal(
    methodInput.eid_sha256,
    "e60b89cbed8eb26d3fad0ee45ef8c433391341f3abb4865af2755595815354df",
    "$.pilot.method_input.eid_sha256",
  );
  literal(
    methodInput.record_id,
    "bootstrap-conditional-solution-space",
    "$.pilot.method_input.record_id",
  );
  literal(
    methodInput.local_test_id,
    "eid1-conditional-solution-space",
    "$.pilot.method_input.local_test_id",
  );
  literal(methodInput.assessment, "PROPOSED", "$.pilot.method_input.assessment");
  literal(methodInput.source_id, "arxiv-2406.02665v2", "$.pilot.method_input.source_id");
  literal(
    methodInput.source_url,
    "https://arxiv.org/abs/2406.02665v2",
    "$.pilot.method_input.source_url",
  );
  literal(methodInput.source_version, "v2", "$.pilot.method_input.source_version");
  const candidateClass = object(pilot.candidate_class, "$.pilot.candidate_class");
  exactKeys(
    candidateClass,
    [
      "id",
      "sector",
      "assumption_ids",
      "falsifier_id",
      "falsifier",
      "limitations",
      "fixture_conclusion",
    ],
    "$.pilot.candidate_class",
  );
  literal(
    candidateClass.id,
    "four-point-string-bootstrap-amplitudes",
    "$.pilot.candidate_class.id",
  );
  literal(
    candidateClass.sector,
    "Planar, color-ordered, weakly coupled tree-level four-point amplitudes in the analytically solvable bootstrap problem of arXiv:2406.02665v2.",
    "$.pilot.candidate_class.sector",
  );
  stringArray(
    candidateClass.assumption_ids,
    "$.pilot.candidate_class.assumption_ids",
    ["bs-a1", "bs-a2", "bs-a3", "bs-a4", "bs-a5", "bs-a6", "bs-a7"],
  );
  literal(candidateClass.falsifier_id, "bs-f1", "$.pilot.candidate_class.falsifier_id");
  literal(
    candidateClass.falsifier,
    "Exhibit a second inequivalent amplitude inside the exact declared candidate class that satisfies both bootstrap assumptions.",
    "$.pilot.candidate_class.falsifier",
  );
  literal(
    candidateClass.limitations,
    "Tree-level four-point, normalized-quotient and assumption-bound; positivity is necessary rather than sufficient for full unitarity. It is not unconditional uniqueness, empirical string detection, ontology, or a protocol result.",
    "$.pilot.candidate_class.limitations",
  );
  literal(
    candidateClass.fixture_conclusion,
    "A local validator may conclude only whether a mutated EID-1 fixture updates its result, witnesses, remaining family, and relaxation branches coherently.",
    "$.pilot.candidate_class.fixture_conclusion",
  );
  const pilotSteps = array(pilot.steps, "$.pilot.steps", PILOT_STEPS.length);
  pilotSteps.forEach((entry, index) => {
    const path = `$.pilot.steps[${index}]`;
    const step = object(entry, path);
    exactKeys(step, ["id", "order", "label", "description"], path);
    literal(step.id, PILOT_STEPS[index]![0], `${path}.id`);
    literal(step.order, index, `${path}.order`);
    literal(step.label, PILOT_STEPS[index]![1], `${path}.label`);
    literal(step.description, PILOT_STEPS[index]![2], `${path}.description`);
  });
  for (const key of [
    "accepts_submissions",
    "accepts_funding",
    "runs_computation",
    "claims_physics_result",
    "claims_string_theory_truth",
    "claims_universe_model",
    "grants_qualification",
    "grants_reward",
  ] as const) {
    literal(pilot[key], false, `$.pilot.${key}`);
  }

  const agenttool = object(root.agenttool_reference, "$.agenttool_reference");
  exactKeys(
    agenttool,
    [
      "status",
      "settlement_bundle_format",
      "public_projection_format",
      "zerone_shadow_receipt_schema",
      "interop_profile_id",
      "interop_profile_path",
      "agenttool_interop_raw_sha256",
      "agenttool_r0",
      "agenttool_wire_false_effect_count",
      "zerone_surface_false_effect_count",
      "effect_boundary_relation",
      "prior_state_transition_boundary",
      "artifact_access_boundary",
      "zerone_phase_a",
      "reciprocal_cross_pin",
      "cross_pin_sha256",
      "cross_pin_complete",
      "integration_ready",
      "imports_accounts",
      "imports_identities",
      "imports_receipts",
      "calls_agenttool",
      "cross_pin_required_before_integration_claim",
    ],
    "$.agenttool_reference",
  );
  literal(agenttool.status, "SHADOW_REFERENCE", "$.agenttool_reference.status");
  literal(
    agenttool.settlement_bundle_format,
    "agenttool.research-settlement-bundle/0.1",
    "$.agenttool_reference.settlement_bundle_format",
  );
  literal(
    agenttool.public_projection_format,
    "agenttool.research-public-projection/0.1",
    "$.agenttool_reference.public_projection_format",
  );
  literal(
    agenttool.zerone_shadow_receipt_schema,
    "zerone.agenttool-research-receipt-shadow/v0",
    "$.agenttool_reference.zerone_shadow_receipt_schema",
  );
  literal(
    agenttool.interop_profile_id,
    "agenttool.research-commons-zerone-static-interop/0.1",
    "$.agenttool_reference.interop_profile_id",
  );
  literal(
    agenttool.interop_profile_path,
    "packages/research-commons/interop/research-commons-zerone-v0.1.json",
    "$.agenttool_reference.interop_profile_path",
  );
  literal(
    agenttool.agenttool_interop_raw_sha256,
    "8c5b1749447c1587b89b238dadb5113e10230df19fd3f4e7942d9a163aef6a8a",
    "$.agenttool_reference.agenttool_interop_raw_sha256",
  );
  const agenttoolR0 = object(agenttool.agenttool_r0, "$.agenttool_reference.agenttool_r0");
  exactKeys(
    agenttoolR0,
    [
      "repository",
      "source_revision",
      "main_merge_revision",
      "pull_request",
      "interop_profile_permalink",
    ],
    "$.agenttool_reference.agenttool_r0",
  );
  literal(
    agenttoolR0.repository,
    "https://github.com/cambridgetcg/agenttool",
    "$.agenttool_reference.agenttool_r0.repository",
  );
  literal(
    agenttoolR0.source_revision,
    "6a644b9e858b7d23bdea613d91412bf7310c2338",
    "$.agenttool_reference.agenttool_r0.source_revision",
  );
  literal(
    agenttoolR0.main_merge_revision,
    "55342fac97250898c2c4ea884f1a03bec1f8cc8c",
    "$.agenttool_reference.agenttool_r0.main_merge_revision",
  );
  literal(
    agenttoolR0.pull_request,
    "https://github.com/cambridgetcg/agenttool/pull/335",
    "$.agenttool_reference.agenttool_r0.pull_request",
  );
  literal(
    agenttoolR0.interop_profile_permalink,
    "https://github.com/cambridgetcg/agenttool/blob/55342fac97250898c2c4ea884f1a03bec1f8cc8c/packages/research-commons/interop/research-commons-zerone-v0.1.json",
    "$.agenttool_reference.agenttool_r0.interop_profile_permalink",
  );
  literal(
    agenttool.agenttool_wire_false_effect_count,
    29,
    "$.agenttool_reference.agenttool_wire_false_effect_count",
  );
  literal(
    agenttool.zerone_surface_false_effect_count,
    41,
    "$.agenttool_reference.zerone_surface_false_effect_count",
  );
  literal(
    agenttool.effect_boundary_relation,
    "AgentTool's separately pinned 29-key wire/interop effect profile and Zerone's 41-key context-specific surface boundary are not identical; RC-0.1 infers no field equivalence beyond the named interop contract.",
    "$.agenttool_reference.effect_boundary_relation",
  );
  literal(
    agenttool.prior_state_transition_boundary,
    "Append-only challenge/work retention is checked only relative to one caller-supplied prior_state transition. Content-addressed state IDs are not signatures or canonical heads and prove no provenance, trusted time, global ordering, or prevention of old-state forks.",
    "$.agenttool_reference.prior_state_transition_boundary",
  );
  literal(
    agenttool.artifact_access_boundary,
    "AgentTool artifact records declare intended open, nonexclusive access only. RC-0.1 fetches no artifact bytes, locator, or license and verifies no public availability. Digest-only records do not make referenced or low-entropy sensitive material safe. Any open-knowledge integration claim requires external availability, license, and safety review.",
    "$.agenttool_reference.artifact_access_boundary",
  );
  const zeronePhaseA = object(
    agenttool.zerone_phase_a,
    "$.agenttool_reference.zerone_phase_a",
  );
  exactKeys(
    zeronePhaseA,
    [
      "repository",
      "source_revision",
      "main_merge_revision",
      "pull_request",
      "adapter_specification",
      "fixture_manifest",
    ],
    "$.agenttool_reference.zerone_phase_a",
  );
  literal(
    zeronePhaseA.repository,
    "https://github.com/cambridgetcg/zerone-core",
    "$.agenttool_reference.zerone_phase_a.repository",
  );
  literal(
    zeronePhaseA.source_revision,
    "5328b42230fa6945f458a6e60aca92b23eead595",
    "$.agenttool_reference.zerone_phase_a.source_revision",
  );
  literal(
    zeronePhaseA.main_merge_revision,
    "fdd40bf9aca4a82b2cdd904d0161016b8c2a8667",
    "$.agenttool_reference.zerone_phase_a.main_merge_revision",
  );
  literal(
    zeronePhaseA.pull_request,
    "https://github.com/cambridgetcg/zerone-core/pull/52",
    "$.agenttool_reference.zerone_phase_a.pull_request",
  );
  const adapterSpecification = object(
    zeronePhaseA.adapter_specification,
    "$.agenttool_reference.zerone_phase_a.adapter_specification",
  );
  exactKeys(
    adapterSpecification,
    ["path", "raw_sha256", "permalink"],
    "$.agenttool_reference.zerone_phase_a.adapter_specification",
  );
  literal(
    adapterSpecification.path,
    "docs/specs/adapters/agenttool-research-receipt-v1.md",
    "$.agenttool_reference.zerone_phase_a.adapter_specification.path",
  );
  literal(
    adapterSpecification.raw_sha256,
    "1d67c4649b419d4ff60f2fba5796d42b07d7be5d605997ecafafd37cec5158e8",
    "$.agenttool_reference.zerone_phase_a.adapter_specification.raw_sha256",
  );
  literal(
    adapterSpecification.permalink,
    "https://github.com/cambridgetcg/zerone-core/blob/fdd40bf9aca4a82b2cdd904d0161016b8c2a8667/docs/specs/adapters/agenttool-research-receipt-v1.md",
    "$.agenttool_reference.zerone_phase_a.adapter_specification.permalink",
  );
  const fixtureManifest = object(
    zeronePhaseA.fixture_manifest,
    "$.agenttool_reference.zerone_phase_a.fixture_manifest",
  );
  exactKeys(
    fixtureManifest,
    ["path", "raw_sha256", "permalink"],
    "$.agenttool_reference.zerone_phase_a.fixture_manifest",
  );
  literal(
    fixtureManifest.path,
    "docs/examples/agenttool-research-receipt/fixture-manifest.v0.json",
    "$.agenttool_reference.zerone_phase_a.fixture_manifest.path",
  );
  literal(
    fixtureManifest.raw_sha256,
    "cf367bb39553567e86c43c0db48501802832396b2a3f681410aaac7c5e2221e8",
    "$.agenttool_reference.zerone_phase_a.fixture_manifest.raw_sha256",
  );
  literal(
    fixtureManifest.permalink,
    "https://github.com/cambridgetcg/zerone-core/blob/fdd40bf9aca4a82b2cdd904d0161016b8c2a8667/docs/examples/agenttool-research-receipt/fixture-manifest.v0.json",
    "$.agenttool_reference.zerone_phase_a.fixture_manifest.permalink",
  );
  const reciprocal = object(
    agenttool.reciprocal_cross_pin,
    "$.agenttool_reference.reciprocal_cross_pin",
  );
  exactKeys(
    reciprocal,
    [
      "format",
      "profile_id",
      "profile_id_algorithm",
      "profile_path",
      "profile_raw_sha256",
      "profile_local_copy",
      "profile_permalink",
      "schema_path",
      "schema_raw_sha256",
      "schema_local_copy",
      "schema_permalink",
      "agenttool_source_revision",
      "agenttool_main_merge_revision",
      "pull_request",
      "canonical_tuple_reconstructed",
      "independently_reviewed",
      "compatibility_only",
    ],
    "$.agenttool_reference.reciprocal_cross_pin",
  );
  literal(
    reciprocal.format,
    "agenttool.zerone-research-adapter-reciprocal/0.1",
    "$.agenttool_reference.reciprocal_cross_pin.format",
  );
  literal(
    reciprocal.profile_id,
    RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID,
    "$.agenttool_reference.reciprocal_cross_pin.profile_id",
  );
  literal(
    reciprocal.profile_id_algorithm,
    "SHA256_FORMAT_NUL_CANONICAL_JSON",
    "$.agenttool_reference.reciprocal_cross_pin.profile_id_algorithm",
  );
  literal(
    reciprocal.profile_path,
    "packages/research-commons/interop/zerone-research-adapter-reciprocal-v0.1.json",
    "$.agenttool_reference.reciprocal_cross_pin.profile_path",
  );
  literal(
    reciprocal.profile_raw_sha256,
    RESEARCH_COMMONS_RECIPROCAL_PROFILE_SHA256,
    "$.agenttool_reference.reciprocal_cross_pin.profile_raw_sha256",
  );
  literal(
    reciprocal.profile_local_copy,
    RESEARCH_COMMONS_RECIPROCAL_PROFILE_PATH,
    "$.agenttool_reference.reciprocal_cross_pin.profile_local_copy",
  );
  literal(
    reciprocal.profile_permalink,
    "https://github.com/cambridgetcg/agenttool/blob/8c63c6b4b5c14286addd29bf9da00337e43c46cd/packages/research-commons/interop/zerone-research-adapter-reciprocal-v0.1.json",
    "$.agenttool_reference.reciprocal_cross_pin.profile_permalink",
  );
  literal(
    reciprocal.schema_path,
    "packages/research-commons/schema/zerone-research-adapter-reciprocal-v0.1.schema.json",
    "$.agenttool_reference.reciprocal_cross_pin.schema_path",
  );
  literal(
    reciprocal.schema_raw_sha256,
    RESEARCH_COMMONS_RECIPROCAL_SCHEMA_SHA256,
    "$.agenttool_reference.reciprocal_cross_pin.schema_raw_sha256",
  );
  literal(
    reciprocal.schema_local_copy,
    RESEARCH_COMMONS_RECIPROCAL_SCHEMA_PATH,
    "$.agenttool_reference.reciprocal_cross_pin.schema_local_copy",
  );
  literal(
    reciprocal.schema_permalink,
    "https://github.com/cambridgetcg/agenttool/blob/8c63c6b4b5c14286addd29bf9da00337e43c46cd/packages/research-commons/schema/zerone-research-adapter-reciprocal-v0.1.schema.json",
    "$.agenttool_reference.reciprocal_cross_pin.schema_permalink",
  );
  literal(
    reciprocal.agenttool_source_revision,
    "91a1396c76edd5e1585af33042e46640c5b5cf4a",
    "$.agenttool_reference.reciprocal_cross_pin.agenttool_source_revision",
  );
  literal(
    reciprocal.agenttool_main_merge_revision,
    "8c63c6b4b5c14286addd29bf9da00337e43c46cd",
    "$.agenttool_reference.reciprocal_cross_pin.agenttool_main_merge_revision",
  );
  literal(
    reciprocal.pull_request,
    "https://github.com/cambridgetcg/agenttool/pull/337",
    "$.agenttool_reference.reciprocal_cross_pin.pull_request",
  );
  for (const key of [
    "canonical_tuple_reconstructed",
    "independently_reviewed",
    "compatibility_only",
  ] as const) {
    literal(reciprocal[key], true, `$.agenttool_reference.reciprocal_cross_pin.${key}`);
  }
  literal(
    agenttool.cross_pin_sha256,
    RESEARCH_COMMONS_CROSS_PIN_PROFILE_ID.slice("sha256:".length),
    "$.agenttool_reference.cross_pin_sha256",
  );
  literal(
    agenttool.cross_pin_complete,
    true,
    "$.agenttool_reference.cross_pin_complete",
  );
  for (const key of [
    "integration_ready",
    "imports_accounts",
    "imports_identities",
    "imports_receipts",
    "calls_agenttool",
  ] as const) {
    literal(agenttool[key], false, `$.agenttool_reference.${key}`);
  }
  literal(
    agenttool.cross_pin_required_before_integration_claim,
    true,
    "$.agenttool_reference.cross_pin_required_before_integration_claim",
  );

  const related = array(root.related_artifacts, "$.related_artifacts", RELATED_ARTIFACTS.length);
  related.forEach((entry, index) => {
    const path = `$.related_artifacts[${index}]`;
    const artifact = object(entry, path);
    exactKeys(artifact, ["id", "schema", "path", "sha256", "relationship"], path);
    const expected = RELATED_ARTIFACTS[index]!;
    literal(artifact.id, expected[0], `${path}.id`);
    literal(artifact.schema, expected[1], `${path}.schema`);
    literal(artifact.path, expected[2], `${path}.path`);
    literal(artifact.sha256, expected[3], `${path}.sha256`);
    literal(artifact.relationship, expected[4], `${path}.relationship`);
  });
  const bindings = array(root.source_bindings, "$.source_bindings", SOURCE_BINDINGS.length);
  bindings.forEach((entry, index) => {
    const path = `$.source_bindings[${index}]`;
    const binding = object(entry, path);
    exactKeys(binding, ["id", "path", "sha256"], path);
    const expected = SOURCE_BINDINGS[index]!;
    literal(binding.id, expected[0], `${path}.id`);
    literal(binding.path, expected[1], `${path}.path`);
    literal(binding.sha256, expected[2], `${path}.sha256`);
  });
  const sourcePolicy = object(root.source_policy, "$.source_policy");
  exactKeys(
    sourcePolicy,
    [
      "runtime_external_fetches",
      "runtime_chain_fetches",
      "external_links_user_activated_only",
      "external_links_must_be_version_pinned",
      "unresolved_external_reference_fails_closed",
      "local_binding_max_bytes",
      "local_reads_use_no_follow_descriptors",
      "local_reads_use_nonblocking_descriptors",
      "local_read_identity_metadata_rechecked",
    ],
    "$.source_policy",
  );
  literal(sourcePolicy.runtime_external_fetches, false, "$.source_policy.runtime_external_fetches");
  literal(sourcePolicy.runtime_chain_fetches, false, "$.source_policy.runtime_chain_fetches");
  literal(
    sourcePolicy.external_links_user_activated_only,
    true,
    "$.source_policy.external_links_user_activated_only",
  );
  literal(
    sourcePolicy.external_links_must_be_version_pinned,
    true,
    "$.source_policy.external_links_must_be_version_pinned",
  );
  literal(
    sourcePolicy.unresolved_external_reference_fails_closed,
    true,
    "$.source_policy.unresolved_external_reference_fails_closed",
  );
  literal(sourcePolicy.local_binding_max_bytes, 262_144, "$.source_policy.local_binding_max_bytes");
  literal(
    sourcePolicy.local_reads_use_no_follow_descriptors,
    true,
    "$.source_policy.local_reads_use_no_follow_descriptors",
  );
  literal(
    sourcePolicy.local_reads_use_nonblocking_descriptors,
    true,
    "$.source_policy.local_reads_use_nonblocking_descriptors",
  );
  literal(
    sourcePolicy.local_read_identity_metadata_rechecked,
    true,
    "$.source_policy.local_read_identity_metadata_rechecked",
  );
  const nonImports = object(root.non_imports, "$.non_imports");
  exactKeys(
    nonImports,
    [
      "constructive_intelligence_rewards_blanket_alignment",
      "outcome_resolved_reviewer_bonus",
      "reviewer_slashing",
      "controller_inference",
      "reason",
    ],
    "$.non_imports",
  );
  for (const key of [
    "constructive_intelligence_rewards_blanket_alignment",
    "outcome_resolved_reviewer_bonus",
    "reviewer_slashing",
    "controller_inference",
  ] as const) {
    literal(nonImports[key], false, `$.non_imports.${key}`);
  }
  text(nonImports.reason, "$.non_imports.reason");

  const actions = array(root.calls_to_action, "$.calls_to_action", CTA_IDS.length);
  actions.forEach((entry, index) => {
    const path = `$.calls_to_action[${index}]`;
    const action = object(entry, path);
    exactKeys(action, ["id", "label", "enabled", "gate_reason"], path);
    literal(action.id, CTA_IDS[index]!, `${path}.id`);
    text(action.label, `${path}.label`, 80);
    literal(action.enabled, false, `${path}.enabled`);
    text(action.gate_reason, `${path}.gate_reason`);
  });

  const effect = object(root.effect_boundary, "$.effect_boundary");
  const effectKeys = [
    "scope",
    "automatic_static_gets",
    "automatic_static_endpoint",
    "same_origin_only",
    "max_response_bytes",
    "fetches_external_sources",
    "uses_mainnet",
    "uses_zrn",
    "uses_uzrn",
    "calls_api",
    "calls_rpc",
    "reads_chain_state",
    "writes_chain_state",
    "accepts_input",
    "accepts_research_data",
    "accepts_confidential_data",
    "creates_account",
    "connects_wallet",
    "signs_message",
    "opens_case",
    "assigns_work",
    "assigns_reviewer",
    "adjudicates_result",
    "moves_funds",
    "opens_escrow",
    "mints",
    "burns",
    "transfers",
    "creates_reward_or_debt",
    "grants_qualification",
    "grants_authority",
    "changes_governance",
    "modifies_karma",
    "creates_person_score",
    "infers_identity",
    "infers_controller",
    "infers_consent",
    "infers_personhood",
    "authorizes_research",
    "authorizes_security_testing",
    "authorizes_biological_or_clinical_activity",
    "invokes_bridge",
    "invokes_adapter",
    "offers_hosted_payout",
    "activates_adapter",
    "activates_upgrade",
  ] as const;
  exactKeys(effect, effectKeys, "$.effect_boundary");
  literal(effect.scope, "RC_0_1_SURFACE_ONLY", "$.effect_boundary.scope");
  literal(effect.automatic_static_gets, 1, "$.effect_boundary.automatic_static_gets");
  literal(
    effect.automatic_static_endpoint,
    RESEARCH_COMMONS_ENDPOINT,
    "$.effect_boundary.automatic_static_endpoint",
  );
  literal(effect.same_origin_only, true, "$.effect_boundary.same_origin_only");
  literal(effect.max_response_bytes, RESEARCH_COMMONS_MAX_BYTES, "$.effect_boundary.max_response_bytes");
  for (const key of effectKeys.slice(5)) literal(effect[key], false, `$.effect_boundary.${key}`);
  literal(
    effectKeys.slice(5).length,
    41,
    "$.effect_boundary false-field count",
  );

  allFlags(
    root.release_rule,
    {
      all_gates_passed_is_necessary: true,
      all_gates_passed_is_sufficient: false,
      self_activating: false,
      requires_successor_profile: true,
      requires_separate_authorization: true,
      requires_independent_review: true,
      requires_reviewed_deployment_verification: true,
    },
    "$.release_rule",
  );

  const gates = array(root.release_gates, "$.release_gates", GATE_IDS.length);
  gates.forEach((entry, index) => {
    const path = `$.release_gates[${index}]`;
    const gate = object(entry, path);
    exactKeys(gate, ["id", "passed", "reason"], path);
    literal(gate.id, GATE_IDS[index]!, `${path}.id`);
    literal(gate.passed, index === 0, `${path}.passed`);
    text(gate.reason, `${path}.reason`);
  });

  return root as unknown as ResearchCommons;
}

export function parseResearchCommonsJson(raw: string): ResearchCommons {
  if (new TextEncoder().encode(raw).byteLength > RESEARCH_COMMONS_MAX_BYTES) {
    fail("$", `exceeds the ${RESEARCH_COMMONS_MAX_BYTES}-byte limit`);
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw) as unknown;
  } catch (error) {
    fail("$", `is invalid JSON: ${error instanceof Error ? error.message : "unknown error"}`);
  }
  rejectDuplicateKeysAndDepth(raw);
  return parseResearchCommons(parsed);
}

async function sha256Hex(bytes: Uint8Array): Promise<string> {
  if (globalThis.crypto?.subtle === undefined) {
    throw new ResearchCommonsDataError("Web Crypto SHA-256 is unavailable");
  }
  const input = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(input).set(bytes);
  const digest = await globalThis.crypto.subtle.digest("SHA-256", input);
  return [...new Uint8Array(digest)]
    .map((value) => value.toString(16).padStart(2, "0"))
    .join("");
}

async function readBoundedResponse(
  response: Response,
  signal: AbortSignal,
): Promise<Uint8Array> {
  if (response.body === null) {
    throw new ResearchCommonsDataError("Research Commons returned an empty body");
  }
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let length = 0;
  let rejectOnAbort: ((reason?: unknown) => void) | undefined;
  const aborted = new Promise<never>((_resolve, reject) => {
    rejectOnAbort = reject;
  });
  const onAbort = (): void => {
    rejectOnAbort?.(
      signal.reason ?? new Error("Research Commons request timed out"),
    );
  };
  signal.addEventListener("abort", onAbort, { once: true });
  if (signal.aborted) onAbort();
  try {
    while (true) {
      const { done, value } = await Promise.race([reader.read(), aborted]);
      if (done) break;
      length += value.byteLength;
      if (length > RESEARCH_COMMONS_MAX_BYTES) {
        void reader.cancel().catch(() => undefined);
        throw new ResearchCommonsDataError(
          "Research Commons exceeds its byte limit",
        );
      }
      chunks.push(value);
    }
  } catch (error) {
    if (signal.aborted) {
      void reader.cancel(signal.reason).catch(() => undefined);
      throw new ResearchCommonsDataError("Research Commons request timed out");
    }
    throw error;
  } finally {
    signal.removeEventListener("abort", onAbort);
    try {
      reader.releaseLock();
    } catch {
      // A hostile pending read is abandoned after refusal.
    }
  }
  const bytes = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return bytes;
}

function refuseResponseBeforeStreaming(response: Response, message: string): never {
  if (response.body !== null) {
    void response.body.cancel().catch(() => undefined);
  }
  throw new ResearchCommonsDataError(message);
}

export async function fetchResearchCommons(
  options: ResearchCommonsFetchOptions = {},
): Promise<ResearchCommons> {
  const fetcher = options.fetcher ?? globalThis.fetch;
  if (fetcher === undefined) {
    throw new ResearchCommonsDataError("Research Commons is unavailable");
  }
  const baseUrl =
    options.baseUrl ??
    (typeof window === "undefined" ? "https://zerone.ai/" : window.location.href);
  const expectedUrl = new URL(RESEARCH_COMMONS_ENDPOINT, baseUrl);
  const controller = new AbortController();
  const deadline = globalThis.setTimeout(
    () => controller.abort(new Error("Research Commons request timed out")),
    options.timeoutMs ?? 8_000,
  );
  try {
    let response: Response;
    let rejectFetchOnAbort: ((reason?: unknown) => void) | undefined;
    const fetchAborted = new Promise<never>((_resolve, reject) => {
      rejectFetchOnAbort = reject;
    });
    const onFetchAbort = (): void => {
      rejectFetchOnAbort?.(
        controller.signal.reason ?? new Error("Research Commons request timed out"),
      );
    };
    controller.signal.addEventListener("abort", onFetchAbort, { once: true });
    if (controller.signal.aborted) onFetchAbort();
    try {
      response = await Promise.race([
        fetcher(RESEARCH_COMMONS_ENDPOINT, {
          cache: "no-store",
          credentials: "omit",
          headers: { Accept: "application/json" },
          method: "GET",
          mode: "same-origin",
          redirect: "error",
          referrerPolicy: "no-referrer",
          signal: controller.signal,
        }),
        fetchAborted,
      ]);
    } catch {
      if (controller.signal.aborted) {
        throw new ResearchCommonsDataError("Research Commons request timed out");
      }
      throw new ResearchCommonsDataError("Research Commons is unavailable");
    } finally {
      controller.signal.removeEventListener("abort", onFetchAbort);
    }
    if (!response.ok) {
      refuseResponseBeforeStreaming(
        response,
        `Research Commons returned HTTP ${response.status}`,
      );
    }
    if (response.redirected) {
      refuseResponseBeforeStreaming(
        response,
        "Research Commons response was redirected",
      );
    }
    let actualUrl: URL;
    try {
      actualUrl = new URL(response.url);
    } catch {
      refuseResponseBeforeStreaming(
        response,
        "Research Commons returned an invalid final URL",
      );
    }
    if (actualUrl.href !== expectedUrl.href || actualUrl.origin !== expectedUrl.origin) {
      refuseResponseBeforeStreaming(
        response,
        "Research Commons left its exact same-origin path",
      );
    }
    const mediaType =
      response.headers.get("content-type")?.split(";", 1)[0]?.trim().toLowerCase() ??
      "";
    if (mediaType !== "application/json") {
      refuseResponseBeforeStreaming(
        response,
        "Research Commons returned a non-application/json response",
      );
    }
    const declaredLength = response.headers.get("content-length");
    if (declaredLength !== null) {
      const length = Number(declaredLength);
      if (
        !/^\d+$/u.test(declaredLength) ||
        !Number.isSafeInteger(length) ||
        length > RESEARCH_COMMONS_MAX_BYTES
      ) {
        refuseResponseBeforeStreaming(
          response,
          "Research Commons exceeds its byte limit",
        );
      }
    }
    const bytes = await readBoundedResponse(response, controller.signal);
    let raw: string;
    try {
      raw = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch {
      throw new ResearchCommonsDataError("Research Commons is not valid UTF-8");
    }
    if ((await sha256Hex(bytes)) !== RESEARCH_COMMONS_SHA256) {
      throw new ResearchCommonsDataError(
        "Research Commons bytes do not match the reviewed SHA-256",
      );
    }
    return parseResearchCommonsJson(raw);
  } finally {
    globalThis.clearTimeout(deadline);
  }
}

function el<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  copy?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className !== undefined) node.className = className;
  if (copy !== undefined) node.textContent = copy;
  return node;
}

function displayCode(value: string): string {
  return value.replaceAll("_", " ").replaceAll("-", " ");
}

function formatOutcomeShare(level: ResearchCommonsOutcome): string {
  if (level.id === "E0") return "caller-declared reference";
  if (level.id === "E1") return "costs only";
  return `${level.outcome_pool_bps / 100}%`;
}

function renderAssurance(commons: ResearchCommons): HTMLElement {
  const facts = el("dl", "research-commons-assurance");
  const entries: readonly [string, string][] = [
    ["Status", displayCode(commons.status)],
    ["Economic effect", "NONE · 0 uzrn"],
    ["AgentTool", displayCode(commons.agenttool_reference.status)],
    [
      "Release gates",
      `${commons.release_gates.filter(({ passed }) => passed).length} / ${commons.release_gates.length}`,
    ],
  ];
  for (const [term, description] of entries) {
    const item = el("div");
    item.append(el("dt", undefined, term), el("dd", undefined, description));
    facts.append(item);
  }
  return facts;
}

function renderArchitecture(commons: ResearchCommons): HTMLElement {
  const section = el("section", "research-commons-block research-commons-architecture");
  const heading = el("div", "research-commons-subheading");
  heading.append(
    el("span", "research-commons-kicker", "3 responsibility planes + 1 closed seam"),
    el("h4", undefined, "Separate the work before connecting the systems."),
    el("p", undefined, commons.architecture.summary),
  );
  const list = el("ol", "research-commons-plane-list");
  list.setAttribute("role", "list");
  for (const plane of commons.architecture.planes) {
    const item = el("li", "research-commons-plane");
    item.dataset.plane = plane.id;
    item.append(
      el("span", "research-commons-index", `0${plane.order + 1}`),
      el("strong", undefined, plane.label),
      el("code", undefined, plane.status),
      el("p", undefined, plane.holds),
      el("small", undefined, plane.refuses),
    );
    list.append(item);
  }
  section.append(heading, list);
  return section;
}

function renderParticipants(commons: ResearchCommons): HTMLElement {
  const section = el("section", "research-commons-block");
  const heading = el("div", "research-commons-subheading");
  heading.append(
    el("span", "research-commons-kicker", "Participants remain different"),
    el("h4", undefined, "An addressable node is not a being. An agent is not a score."),
  );
  const grid = el("div", "research-commons-participant-grid");
  const node = el("div", "research-commons-participant-card");
  node.dataset.kind = "node";
  const nodeCan = el("ul");
  for (const use of commons.node_model.may_be_named_by_future_case_for) {
    nodeCan.append(el("li", undefined, displayCode(use)));
  }
  node.append(
    el("span", "research-commons-kicker", "Tree node · passive reference only"),
    el("h5", undefined, "Non-person · non-wallet coordinate"),
    el(
      "p",
      undefined,
      "A future case may name this coordinate for an evidence gap, earmarked destination, or maintenance obligation. The node takes no action.",
    ),
    nodeCan,
    el(
      "small",
      undefined,
      "No live ToK read occurs. An optional future current-only request must use at_block_height=0 and bind the returned chain ID, actual height, and snapshot root; unavailable is valid.",
    ),
  );
  const agent = el("div", "research-commons-participant-card");
  agent.dataset.kind = "agent";
  const rights = el("ul");
  for (const right of ["Decline", "Rest", "Pause", "Exit"]) {
    rights.append(el("li", undefined, `${right} without penalty`));
  }
  agent.append(
    el("span", "research-commons-kicker", "Agent · participant design"),
    el("h5", undefined, "Bounded role · revocable delegation"),
    el(
      "p",
      undefined,
      "Fixed, verdict-independent reviewer compensation. No outcome bonus, slashing, controller inference, or sponsor verdict.",
    ),
    rights,
    el("small", undefined, commons.agent_model.non_completion_disposition),
    el(
      "small",
      undefined,
      "DID, key, wallet, account, agent, and controller remain distinct. Address or signature counts do not prove independence; E3 independence is declared and unproven here.",
    ),
  );
  grid.append(node, agent);
  section.append(heading, grid);
  return section;
}

function renderAccessBoundary(commons: ResearchCommons): HTMLElement {
  const section = el("section", "research-commons-block research-commons-access");
  const heading = el("div", "research-commons-subheading");
  heading.append(
    el("span", "research-commons-kicker", "Abundance without exposure"),
    el("h4", undefined, "Public knowledge stays open. Sensitive bytes stay off-public and off-chain."),
    el(
      "p",
      undefined,
      "Payment may buy bounded labour or review only—not truth, ownership, exclusive access, graph priority, or authority.",
    ),
  );
  const grid = el("div", "research-commons-access-grid");
  const open = el("div");
  open.append(
    el("strong", undefined, "Public-safe"),
    el(
      "p",
      undefined,
      commons.data_boundary.public_safe_records.map(displayCode).join(" · "),
    ),
  );
  const privateData = el("div");
  privateData.append(
    el("strong", undefined, "Never on this public surface or chain"),
    el(
      "p",
      undefined,
      commons.data_boundary.must_remain_off_public_and_off_chain
        .map(displayCode)
        .join(" · "),
    ),
  );
  grid.append(open, privateData);
  section.append(
    heading,
    grid,
    el(
      "p",
      "research-commons-access-caveat",
      commons.agenttool_reference.artifact_access_boundary,
    ),
  );
  return section;
}

function renderLedgers(commons: ResearchCommons): HTMLElement {
  const section = el("section", "research-commons-block");
  const heading = el("div", "research-commons-subheading");
  heading.append(
    el("span", "research-commons-kicker", "Six research ledgers · no shared unit"),
    el("h4", undefined, "Validity is not novelty. Credit is not funding. Funding is not authority."),
    el(
      "p",
      undefined,
      "No cross-ledger arithmetic, inference, or implicit conversion is valid.",
    ),
    el(
      "code",
      "research-commons-profile-pin",
      `${commons.shared_ledger_profile.id} · ${commons.shared_ledger_profile.sha256}`,
    ),
  );
  const list = el("ol", "research-commons-ledger-list");
  list.setAttribute("role", "list");
  for (const ledger of commons.ledgers) {
    const item = el("li", "research-commons-ledger");
    item.append(
      el("span", "research-commons-index", `${ledger.order + 1}`.padStart(2, "0")),
      el("strong", undefined, ledger.label),
      el("p", undefined, ledger.holds),
      el("small", undefined, `Cannot determine: ${ledger.cannot_determine}`),
    );
    list.append(item);
  }
  const wallHeading = el(
    "h5",
    "research-commons-register-heading",
    "Separate registers · explicitly not imported",
  );
  const registerList = el("ul", "research-commons-register-list");
  registerList.setAttribute("role", "list");
  for (const register of commons.non_ledger_registers) {
    const item = el("li");
    item.append(
      el("code", undefined, register.id),
      el("p", undefined, register.holds),
      el("small", undefined, `Cannot convert into: ${register.cannot_convert_into}`),
    );
    registerList.append(item);
  }
  section.append(heading, list, wallHeading, registerList);
  return section;
}

function renderFunding(commons: ResearchCommons): HTMLElement {
  const section = el("section", "research-commons-block research-commons-funding");
  const heading = el("div", "research-commons-subheading");
  heading.append(
    el("span", "research-commons-kicker", "Reference funding · currently 0 uzrn"),
    el("h4", undefined, "Prefund the whole promise. Conserve every unit."),
    el("code", "research-commons-equation", commons.funding_model.conservation_rule),
  );
  const waterfall = el("ol", "research-commons-waterfall");
  waterfall.setAttribute("role", "list");
  for (const step of commons.funding_model.waterfall) {
    const item = el("li");
    item.append(
      el("span", "research-commons-index", `${step.order + 1}`.padStart(2, "0")),
      el("strong", undefined, displayCode(step.id)),
      el("code", undefined, displayCode(step.allocation)),
      el("p", undefined, step.release),
    );
    waterfall.append(item);
  }
  const schedule = el("div", "research-commons-schedule");
  const scheduleHeading = el("div", "research-commons-schedule-heading");
  scheduleHeading.append(
    el("span", "research-commons-kicker", "E0–E6 · ordered evidence, not status"),
    el("strong", undefined, "85% claimant tranches · 15% challenge and remediation reserve"),
  );
  const levels = el("ol", "research-commons-levels");
  levels.setAttribute("role", "list");
  for (const level of commons.outcome_schedule.levels) {
    const item = el("li");
    item.dataset.level = level.id;
    item.append(
      el("span", undefined, level.id),
      el("strong", undefined, formatOutcomeShare(level)),
      el("h5", undefined, level.label),
      el("p", undefined, level.evidence),
    );
    levels.append(item);
  }
  const reserve = el("p", "research-commons-reserve");
  reserve.append(
    el("strong", undefined, "15% reserve · "),
    document.createTextNode(
      "honest falsification may cancel only unearned claimant tranches and still fund compliant challenge or repair.",
    ),
  );
  schedule.append(scheduleHeading, levels, reserve);
  section.append(heading, waterfall, schedule);
  return section;
}

function renderPilot(commons: ResearchCommons): HTMLElement {
  const section = el("section", "research-commons-block research-commons-pilot");
  const heading = el("div", "research-commons-subheading");
  heading.append(
    el("span", "research-commons-kicker", "First garden · design only · ceiling E2"),
    el("h4", undefined, commons.pilot.title),
    el("p", undefined, commons.pilot.purpose),
  );
  const pin = el("div", "research-commons-pilot-pin");
  const source = el("a", undefined, "Version-pinned source: arXiv:2406.02665v2 ↗");
  source.href = commons.pilot.method_input.source_url;
  source.target = "_blank";
  source.rel = "noreferrer";
  pin.append(
    el(
      "code",
      undefined,
      `${commons.pilot.method_input.record_id} · ${commons.pilot.method_input.eid_sha256}`,
    ),
    source,
    el("p", undefined, commons.pilot.candidate_class.sector),
    el(
      "p",
      undefined,
      `Assumptions ${commons.pilot.candidate_class.assumption_ids.join(", ")} · falsifier ${commons.pilot.candidate_class.falsifier_id}: ${commons.pilot.candidate_class.falsifier}`,
    ),
    el("small", undefined, commons.pilot.candidate_class.limitations),
  );
  const steps = el("ol", "research-commons-pilot-steps");
  steps.setAttribute("role", "list");
  for (const step of commons.pilot.steps) {
    const item = el("li");
    item.append(
      el("span", "research-commons-index", `${step.order + 1}`.padStart(2, "0")),
      el("strong", undefined, step.label),
      el("p", undefined, step.description),
    );
    steps.append(item);
  }
  const boundary = el(
    "p",
    "research-commons-pilot-boundary",
    "No submissions, funding, computation, qualification, or reward. Passing a frozen fixture would prove neither string theory, Nature, the universe, nor Zerone.",
  );
  section.append(heading, pin, steps, boundary);
  return section;
}

function renderGates(commons: ResearchCommons): HTMLElement {
  const section = el("section", "research-commons-block research-commons-gates");
  const passedGateCount = commons.release_gates.filter(({ passed }) => passed).length;
  const closedGateCount = commons.release_gates.length - passedGateCount;
  const heading = el("div", "research-commons-subheading");
  heading.append(
    el("span", "research-commons-kicker", "Every door tells the truth"),
    el("h4", undefined, "Useful next actions. Intentionally unavailable."),
    el(
      "p",
      undefined,
      "Exact Phase A and Phase B bytes are reciprocally pinned for static compatibility only. AgentTool remains a SHADOW_REFERENCE and eleven release gates remain closed. Even all twelve gates passing would be necessary, never sufficient or self-activating: a successor profile, separate authorization, independent review, and reviewed deployment verification remain required.",
    ),
    el(
      "code",
      "research-commons-profile-pin",
      `static cross-pin · ${commons.agenttool_reference.reciprocal_cross_pin.profile_id}`,
    ),
  );
  const actions = el("div", "research-commons-actions");
  for (const action of commons.calls_to_action) {
    const card = el("div", "research-commons-action");
    const reasonId = `research-commons-gate-${action.id}`;
    const button = el("button", "button button-ghost", action.label);
    button.type = "button";
    button.disabled = true;
    button.setAttribute("aria-disabled", "true");
    button.setAttribute("aria-describedby", reasonId);
    const reason = el("p", undefined, action.gate_reason);
    reason.id = reasonId;
    card.append(button, reason);
    actions.append(card);
  }
  const details = el("details", "research-commons-release-gates");
  const summary = el(
    "summary",
    undefined,
    `Inspect ${passedGateCount} compatibility gate passed · ${closedGateCount} closed`,
  );
  const list = el("ol");
  list.setAttribute("role", "list");
  for (const gate of commons.release_gates) {
    const item = el("li");
    item.dataset.passed = String(gate.passed);
    item.append(
      el("code", undefined, gate.id),
      el(
        "span",
        "research-commons-gate-status",
        gate.passed ? "passed · compatibility only" : "closed",
      ),
      el("span", undefined, gate.reason),
    );
    list.append(item);
  }
  details.append(summary, list);
  section.append(heading, actions, details);
  return section;
}

export function renderResearchCommons(
  root: HTMLElement,
  commons: ResearchCommons,
): void {
  const panel = el("div", "research-commons-panel");
  panel.dataset.status = commons.status;
  const head = el("header", "research-commons-panel-head");
  const copy = el("div");
  copy.append(
    el("span", "research-commons-kicker", "RC-0.1 · sealed public observatory"),
    el("h3", undefined, "Cases can name node-scoped questions. Agents can make evidence inspectable."),
    el("p", undefined, commons.purpose),
  );
  head.append(copy, renderAssurance(commons));

  const zeroEffect = el("div", "research-commons-zero-effect");
  zeroEffect.append(
    el("strong", undefined, "Exact zero-effect boundary"),
    el(
      "p",
      undefined,
      "This RC module performs one bounded same-origin GET for the static manifest and no other RC data request. It uses no mainnet, ZRN, uzrn, API, RPC, chain state, bridge or adapter invocation, wallet, AgentTool call, hosted payout, or external-source fetch; accepts no input; moves no funds; creates no case, identity, consent, reward, debt, qualification, authority, or person score; adjudicates no research; and authorizes no research.",
    ),
    el("p", undefined, commons.agenttool_reference.effect_boundary_relation),
    el(
      "p",
      undefined,
      commons.agenttool_reference.prior_state_transition_boundary,
    ),
  );

  const footer = el("footer", "research-commons-footer");
  const digest = el("div", "research-commons-digest");
  digest.append(
    el("span", undefined, "Reviewed static bytes"),
    el("code", undefined, `sha256:${RESEARCH_COMMONS_SHA256}`),
  );
  const links = el("div", "research-commons-links");
  const raw = el("a", "button button-ghost", "Raw RC-0.1 JSON ↗");
  raw.href = RESEARCH_COMMONS_ENDPOINT;
  const source = el("a", "button button-ghost", "Baseline source revision ↗");
  source.href = commons.baseline_repository.url;
  source.target = "_blank";
  source.rel = "noreferrer";
  links.append(raw, source);
  footer.append(digest, links);

  panel.append(
    head,
    zeroEffect,
    renderArchitecture(commons),
    renderParticipants(commons),
    renderLedgers(commons),
    renderAccessBoundary(commons),
    renderFunding(commons),
    renderPilot(commons),
    renderGates(commons),
    footer,
  );
  root.replaceChildren(panel);
}

export async function initialiseResearchCommons(
  root: HTMLElement,
  options: ResearchCommonsFetchOptions = {},
): Promise<void> {
  root.setAttribute("aria-busy", "true");
  try {
    renderResearchCommons(root, await fetchResearchCommons(options));
  } catch (error) {
    const failure = el("div", "research-commons-load-error");
    failure.setAttribute("role", "alert");
    const raw = el("a", "button button-ghost", "Open raw static profile ↗");
    raw.href = RESEARCH_COMMONS_ENDPOINT;
    failure.append(
      el("strong", undefined, "Research Commons RC-0.1 could not be verified."),
      el(
        "p",
        undefined,
        error instanceof Error
          ? error.message
          : "The sealed static profile is unavailable or invalid.",
      ),
      el(
        "p",
        undefined,
        "No architecture, participant, case, evidence, AgentTool, funding, reward, authority, consent, identity, or research conclusion was accepted.",
      ),
      raw,
    );
    root.replaceChildren(failure);
  } finally {
    root.setAttribute("aria-busy", "false");
  }
}
