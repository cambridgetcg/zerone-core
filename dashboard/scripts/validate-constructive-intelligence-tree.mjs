import { createHash } from "node:crypto";
import { existsSync, readFileSync, realpathSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

// This validator is deliberately offline: URLs are syntax and source-policy
// pins, never network fetches or truth oracles.
export const CONSTRUCTIVE_TREE_SCHEMA =
  "zerone.constructive-intelligence-tree/v1";
export const CONSTRUCTIVE_TREE_MAX_BYTES = 262_144;
export const CONSTRUCTIVE_TREE_MAX_DEPTH = 8;
export const CONSTRUCTIVE_TREE_MAX_FAN_OUT = 16;

const TOP_LEVEL_KEYS = [
  "schema",
  "authoritative",
  "networkObserved",
  "rewardBearing",
  "snapshotDate",
  "policyVersion",
  "releaseBoundary",
  "policy",
  "roots",
  "nodes",
];
const RELEASE_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "authorizesSecurityTesting",
  "assertsProtocolSecurity",
  "performsNetworkRequests",
  "publishesConfidentialEvidence",
];
const POLICY_KEYS = [
  "artifactEdgeTypes",
  "breakthroughRecognition",
  "funding",
  "independence",
  "disclosure",
  "milestones",
  "challengeReserveBps",
];
const BREAKTHROUGH_KEYS = [
  "authorSelected",
  "minimumEvidenceLevel",
  "requiresPriorArtDelta",
  "requiresAdoptionOrDescendantImpact",
];
const FUNDING_KEYS = [
  "skillUnlockCreatesReward",
  "externalWorkDefault",
  "protocolIssuanceGate",
  "timeAloneUnlocksEvidence",
];
const INDEPENDENCE_KEYS = [
  "minimumEffectiveClusters",
  "minimumOrganizationRoots",
  "minimumImplementationRoots",
  "minimumExecutionEnvironments",
  "assignmentAfterArtifactFreeze",
  "reviewPayOutcomeIndependent",
  "rawAddressCountIsEvidence",
];
const DISCLOSURE_KEYS = [
  "safetyIsHardGate",
  "unknownSecurityImpactEscalatesTo",
  "publicExploitPlaintextAllowed",
  "vendorHasPayoutVeto",
];
const MILESTONE_KEYS = ["level", "name", "rewardBps", "treatment"];
const NODE_KEYS = [
  "id",
  "title",
  "stage",
  "domain",
  "summary",
  "prerequisites",
  "attainmentEvidence",
  "rewardEligibility",
  "defaultDisclosureLane",
  "artifactRequirements",
  "revalidationTriggers",
  "standards",
  "repositoryReferences",
  "acceptance",
];
const STANDARD_KEYS = [
  "canonicalId",
  "authority",
  "title",
  "revision",
  "authorityStatus",
  "normalizedMaturity",
  "specification",
  "statusCheckedAt",
  "reviewAfter",
];
const ACCEPTANCE_KEYS = [
  "targetEvidence",
  "scopeBounds",
  "scopeHash",
  "coverageTargets",
  "minimumEffectiveClusters",
  "minimumOrganizationRoots",
  "minimumImplementationRoots",
  "minimumExecutionEnvironments",
  "adoptionReceiptTypes",
  "privateEscalationRequired",
  "prepublicationTriageRequired",
];
const COVERAGE_TARGET_KEYS = [
  "id",
  "minimumEffectiveClusters",
  "minimumOrganizationRoots",
  "minimumImplementationRoots",
  "minimumExecutionEnvironments",
  "minimumCases",
  "requiresCheckerOrCorpusDigest",
];

const REQUIRED_ARTIFACT_EDGE_TYPES = [
  "ATTACKS",
  "DEPLOYS",
  "DISPROVES",
  "IMPLEMENTS",
  "MAINTAINS",
  "PROVES",
  "REPAIRS",
  "REPLICATES",
  "SUPERSEDES",
];
const EXPECTED_MILESTONES = [
  {
    level: "E0",
    name: "committed",
    rewardBps: 0,
    treatment: "precedence-only",
  },
  {
    level: "E1",
    name: "inspectable",
    rewardBps: 0,
    treatment: "verified-cost-only",
  },
  {
    level: "E2",
    name: "class-verified",
    rewardBps: 1500,
    treatment: "milestone",
  },
  {
    level: "E3",
    name: "independently-reproduced",
    rewardBps: 2000,
    treatment: "milestone",
  },
  {
    level: "E4",
    name: "adversarially-survived-or-fix-tested",
    rewardBps: 1500,
    treatment: "milestone",
  },
  {
    level: "E5",
    name: "independently-adopted",
    rewardBps: 2500,
    treatment: "milestone",
  },
  {
    level: "E6",
    name: "maintained",
    rewardBps: 1000,
    treatment: "milestone",
  },
];

const STAGES = new Set([
  "foundation",
  "primitive",
  "assurance",
  "protocol",
  "quest",
]);
const STAGE_RANK = new Map([
  ["foundation", 0],
  ["primitive", 1],
  ["assurance", 2],
  ["protocol", 3],
  ["quest", 4],
]);
const DOMAINS = new Set([
  "assurance",
  "cryptography",
  "mathematics",
  "protocols",
  "quests",
  "security",
  "systems",
]);
const STAGE_DOMAINS = Object.freeze({
  foundation: new Set(["mathematics", "security", "systems"]),
  primitive: new Set(["cryptography"]),
  assurance: new Set(["assurance"]),
  protocol: new Set(["protocols"]),
  quest: new Set(["quests"]),
});
const EVIDENCE_LEVELS = new Set(["E0", "E1", "E2", "E3", "E4", "E5", "E6"]);
const DISCLOSURE_LANES = new Set([
  "open-construction",
  "private-coordinated-repair",
  "controlled-operations",
]);
const REWARD_ELIGIBILITIES = new Set([
  "qualification-only",
  "sponsor-milestones",
]);
const MATURITIES = new Set([
  "final",
  "published",
  "recommendation",
  "approved",
  "project-specification",
  "maintained-policy",
  "candidate-recommendation",
  "draft",
  "eol",
]);
const SPONSOR_FORBIDDEN_MATURITIES = new Set([
  "draft",
  "candidate-recommendation",
]);
const STANDARD_HOSTS = new Set([
  "certcc.github.io",
  "docs.cosmos.network",
  "doi.org",
  "github.com",
  "openid.net",
  "slsa.dev",
  "www.rfc-editor.org",
  "www.w3.org",
]);
const STANDARD_CANONICAL_PINS = Object.freeze({
  "certcc:guide-to-cvd:2022": Object.freeze({
    authority: "CERT/CC",
    title: "CERT Guide to Coordinated Vulnerability Disclosure",
    revision: "2022",
    normalizedMaturity: "maintained-policy",
    specification: "https://certcc.github.io/CERT-Guide-to-CVD/",
  }),
  "cosmos:release-family:2025.1": Object.freeze({
    authority: "Cosmos SDK",
    title: "Cosmos release family 2025.1",
    revision: "status snapshot 2026-07-29",
    normalizedMaturity: "maintained-policy",
    specification: "https://docs.cosmos.network/sdk/latest/release-family",
  }),
  "cosmos:release-family:2026.1": Object.freeze({
    authority: "Cosmos SDK",
    title: "Cosmos release family 2026.1",
    revision: "status snapshot 2026-07-29",
    normalizedMaturity: "maintained-policy",
    specification: "https://docs.cosmos.network/sdk/latest/release-family",
  }),
  "ietf:rfc:8446": Object.freeze({
    authority: "IETF",
    title: "The Transport Layer Security (TLS) Protocol Version 1.3",
    revision: "2018-08",
    normalizedMaturity: "published",
    specification: "https://www.rfc-editor.org/rfc/rfc8446.html",
  }),
  "ietf:rfc:9000": Object.freeze({
    authority: "IETF",
    title: "QUIC: A UDP-Based Multiplexed and Secure Transport",
    revision: "2021-05",
    normalizedMaturity: "published",
    specification: "https://www.rfc-editor.org/rfc/rfc9000.html",
  }),
  "ietf:rfc:9001": Object.freeze({
    authority: "IETF",
    title: "Using TLS to Secure QUIC",
    revision: "2021-05",
    normalizedMaturity: "published",
    specification: "https://www.rfc-editor.org/rfc/rfc9001.html",
  }),
  "ietf:rfc:9420": Object.freeze({
    authority: "IETF",
    title: "The Messaging Layer Security (MLS) Protocol",
    revision: "2023-07",
    normalizedMaturity: "published",
    specification: "https://www.rfc-editor.org/rfc/rfc9420.html",
  }),
  "ietf:rfc:9700": Object.freeze({
    authority: "IETF",
    title: "Best Current Practice for OAuth 2.0 Security",
    revision: "2025-01",
    normalizedMaturity: "published",
    specification: "https://www.rfc-editor.org/rfc/rfc9700.html",
  }),
  "ietf:rfc:9846": Object.freeze({
    authority: "IETF",
    title: "The Transport Layer Security (TLS) Protocol Version 1.3",
    revision: "2026-07",
    normalizedMaturity: "published",
    specification: "https://www.rfc-editor.org/rfc/rfc9846.html",
  }),
  "in-toto:attestation:1.2.0": Object.freeze({
    authority: "in-toto",
    title: "in-toto Attestation Framework",
    revision: "v1.2.0 (commit df02077bf97218a8860a5c534eff1f1381f56984)",
    normalizedMaturity: "project-specification",
    specification:
      "https://github.com/in-toto/attestation/tree/df02077bf97218a8860a5c534eff1f1381f56984",
  }),
  "irtf:rfc:9180": Object.freeze({
    authority: "IRTF CFRG",
    title: "Hybrid Public Key Encryption",
    revision: "2022-02",
    normalizedMaturity: "published",
    specification: "https://www.rfc-editor.org/rfc/rfc9180.html",
  }),
  "nist:fips:203:2024": Object.freeze({
    authority: "NIST",
    title: "Module-Lattice-Based Key-Encapsulation Mechanism Standard",
    revision: "2024-08-13",
    normalizedMaturity: "final",
    specification: "https://doi.org/10.6028/NIST.FIPS.203",
  }),
  "nist:fips:204:2024": Object.freeze({
    authority: "NIST",
    title: "Module-Lattice-Based Digital Signature Standard",
    revision: "2024-08-13",
    normalizedMaturity: "final",
    specification: "https://doi.org/10.6028/NIST.FIPS.204",
  }),
  "nist:fips:205:2024": Object.freeze({
    authority: "NIST",
    title: "Stateless Hash-Based Digital Signature Standard",
    revision: "2024-08-13",
    normalizedMaturity: "final",
    specification: "https://doi.org/10.6028/NIST.FIPS.205",
  }),
  "nist:sp:800-227:2025": Object.freeze({
    authority: "NIST",
    title: "Recommendations for Key-Encapsulation Mechanisms",
    revision: "2025-09-18",
    normalizedMaturity: "final",
    specification: "https://doi.org/10.6028/NIST.SP.800-227",
  }),
  "oidf:fapi2-attacker-model:20250222": Object.freeze({
    authority: "OpenID Foundation",
    title: "FAPI 2.0 Attacker Model",
    revision: "Final, 2025-02-22",
    normalizedMaturity: "final",
    specification: "https://openid.net/specs/fapi-attacker-model-2_0-final.html",
  }),
  "oidf:fapi2-security-profile:20250222": Object.freeze({
    authority: "OpenID Foundation",
    title: "FAPI 2.0 Security Profile",
    revision: "Final, 2025-02-22",
    normalizedMaturity: "final",
    specification: "https://openid.net/specs/fapi-security-profile-2_0-final.html",
  }),
  "sigstore:bundle:v0.3": Object.freeze({
    authority: "Sigstore",
    title: "Sigstore bundle media type v0.3",
    revision:
      "protobuf-specs v0.5.1 (commit 3001afe9102b15b04ca1b91efccd613976bdf514)",
    normalizedMaturity: "project-specification",
    specification:
      "https://github.com/sigstore/protobuf-specs/blob/3001afe9102b15b04ca1b91efccd613976bdf514/protos/sigstore_bundle.proto",
  }),
  "slsa:spec:1.2": Object.freeze({
    authority: "SLSA",
    title: "Supply-chain Levels for Software Artifacts specification",
    revision: "v1.2",
    normalizedMaturity: "approved",
    specification: "https://slsa.dev/spec/v1.2/",
  }),
  "w3c:webauthn:2:20210408": Object.freeze({
    authority: "W3C",
    title:
      "Web Authentication: An API for accessing Public Key Credentials Level 2",
    revision: "2021-04-08",
    normalizedMaturity: "recommendation",
    specification: "https://www.w3.org/TR/2021/REC-webauthn-2-20210408/",
  }),
});
const STANDARD_REVIEW_MAX_DAYS = Object.freeze({
  "cosmos:release-family:2025.1": 7,
  "cosmos:release-family:2026.1": 7,
  "ietf:rfc:8446": 30,
  "ietf:rfc:9846": 30,
  "in-toto:attestation:1.2.0": 7,
  "irtf:rfc:9180": 7,
  "nist:fips:203:2024": 30,
  "nist:fips:204:2024": 30,
  "sigstore:bundle:v0.3": 7,
  "slsa:spec:1.2": 7,
});
const MATURITY_STATUS_MARKERS = Object.freeze({
  final: ["final"],
  published: ["proposed standard", "best current practice", "informational rfc"],
  recommendation: ["recommendation"],
  approved: ["approved"],
  "project-specification": ["project"],
  "maintained-policy": ["maintained"],
  "candidate-recommendation": ["candidate recommendation"],
  draft: ["draft"],
  eol: ["eol", "end of life"],
});

const NODE_ID_PATTERN =
  /^[a-z0-9]+(?:-[a-z0-9]+)*@[a-z0-9]+(?:[.-][a-z0-9]+)*$/;
const CANONICAL_STANDARD_ID_PATTERN = /^[a-z0-9][a-z0-9.:/+@_-]*$/;
const POLICY_VERSION_PATTERN = /^[1-9][0-9]*\.[0-9]+\.[0-9]+$/;
const REPOSITORY_PATH_PATTERN =
  /^(?!\/)(?!.*(?:^|\/)\.\.(?:\/|$))[A-Za-z0-9._/-]+$/;
const ISO_DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const SHA256_PATTERN = /^[0-9a-f]{64}$/;
const COVERAGE_TARGET_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const ADOPTION_RECEIPT_TYPES = new Set([
  "maintained-fixture",
  "maintained-release",
  "standards-disposition",
  "upstream-merge",
]);
const REVIEWED_V1_NORMATIVE_DIGEST =
  "43f65d91d700c9ed7a874f0a34520fc815d51d89a67255aa75f7e8be4ecd7a9a";
const REVIEWED_QUEST_TEMPLATE_DIGESTS = Object.freeze({
  "quest-mls-state-invariants@1":
    "e5832b2f0ed1445467016bae386cc97e54c37e0326a984aaabefb56440e77039",
  "quest-pqc-cross-library-conformance@1":
    "4aa921879f3c81c27db9ba19f2f258d544927586060738098b2436ccb0ba6a00",
  "quest-tls-rfc9846-keyshare-reuse@1":
    "bcefb7c2d177c79d135722bf38a689d122fe564eb39ebec873b0020dacb46206",
});

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const REPOSITORY_ROOT = resolve(dirname(SCRIPT_PATH), "../..");
const REAL_REPOSITORY_ROOT = realpathSync(REPOSITORY_ROOT);

export class ConstructiveTreeValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "ConstructiveTreeValidationError";
    this.path = path;
  }
}

function fail(path, message) {
  throw new ConstructiveTreeValidationError(path, message);
}

function record(value, path) {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    fail(path, "must be an object");
  }
  return value;
}

function exactKeys(value, allowed, path) {
  const allowedSet = new Set(allowed);
  for (const key of Object.keys(value)) {
    if (!allowedSet.has(key)) fail(`${path}.${key}`, "is not part of schema v1");
  }
  for (const key of allowed) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "is required");
  }
}

function boundedString(value, path, maxBytes = 1024) {
  if (typeof value !== "string" || value.length === 0) {
    fail(path, "must be a nonempty string");
  }
  if (Buffer.byteLength(value, "utf8") > maxBytes) {
    fail(path, `must be at most ${maxBytes} UTF-8 bytes`);
  }
  return value;
}

function canonicalJson(value) {
  if (Array.isArray(value)) {
    return `[${value.map((item) => canonicalJson(item)).join(",")}]`;
  }
  if (typeof value === "object" && value !== null) {
    return `{${Object.keys(value)
      .sort()
      .map((key) => `${JSON.stringify(key)}:${canonicalJson(value[key])}`)
      .join(",")}}`;
  }
  return JSON.stringify(value);
}

function normativeStandardProjection(standard) {
  const {
    authorityStatus: _authorityStatus,
    statusCheckedAt: _statusCheckedAt,
    reviewAfter: _reviewAfter,
    ...normative
  } = standard;
  return normative;
}

function normativeNodeProjection(node) {
  return {
    ...node,
    standards: node.standards.map(normativeStandardProjection),
  };
}

function normativeTreeProjection(tree) {
  return {
    schema: tree.schema,
    policyVersion: tree.policyVersion,
    policy: tree.policy,
    roots: tree.roots,
    nodes: tree.nodes.map(normativeNodeProjection),
  };
}

function exactBoolean(value, expected, path) {
  if (value !== expected) fail(path, `must be ${expected}`);
}

function boundedInteger(value, path, minimum, maximum) {
  if (!Number.isSafeInteger(value) || value < minimum || value > maximum) {
    fail(path, `must be an integer from ${minimum} through ${maximum}`);
  }
  return value;
}

function enumValue(value, allowed, path) {
  const parsed = boundedString(value, path, 128);
  if (!allowed.has(parsed)) fail(path, "has an unknown value");
  return parsed;
}

function stringArray(
  value,
  path,
  { minimum = 0, maximum = 32, maxBytes = 1024, pattern, sorted = false } = {},
) {
  if (
    !Array.isArray(value) ||
    value.length < minimum ||
    value.length > maximum
  ) {
    fail(path, `must contain ${minimum} through ${maximum} items`);
  }
  const parsed = value.map((item, index) => {
    const text = boundedString(item, `${path}[${index}]`, maxBytes);
    if (pattern && !pattern.test(text)) {
      fail(`${path}[${index}]`, "has an invalid format");
    }
    return text;
  });
  if (new Set(parsed).size !== parsed.length) fail(path, "must not contain duplicates");
  if (sorted && parsed.some((item, index) => index > 0 && parsed[index - 1] > item)) {
    fail(path, "must be sorted lexicographically");
  }
  return parsed;
}

function parseIsoDate(value, path) {
  const text = boundedString(value, path, 10);
  if (!ISO_DATE_PATTERN.test(text)) fail(path, "must use YYYY-MM-DD");
  const timestamp = Date.parse(`${text}T00:00:00Z`);
  if (!Number.isFinite(timestamp) || new Date(timestamp).toISOString().slice(0, 10) !== text) {
    fail(path, "must be a real calendar date");
  }
  return text;
}

function parseHttpsUrl(value, path) {
  const text = boundedString(value, path, 512);
  let parsed;
  try {
    parsed = new URL(text);
  } catch {
    fail(path, "must be an absolute URL");
  }
  if (
    parsed.protocol !== "https:" ||
    !STANDARD_HOSTS.has(parsed.hostname) ||
    parsed.username !== "" ||
    parsed.password !== "" ||
    parsed.search !== "" ||
    parsed.hash !== ""
  ) {
    fail(
      path,
      "must be an allowlisted authoritative HTTPS source without credentials, query parameters, or a fragment",
    );
  }
  if (
    parsed.pathname.includes("/latest/") &&
    !parsed.hostname.endsWith("cosmos.network")
  ) {
    fail(path, "must pin an immutable edition rather than a mutable latest path");
  }
  return parsed;
}

function validateRepositoryReferences(value, path) {
  const references = stringArray(value, path, {
    minimum: 1,
    maximum: 16,
    maxBytes: 256,
    pattern: REPOSITORY_PATH_PATTERN,
    sorted: true,
  });
  for (const [index, reference] of references.entries()) {
    const target = resolve(REPOSITORY_ROOT, reference);
    const relativeTarget = relative(REPOSITORY_ROOT, target);
    if (
      relativeTarget === "" ||
      relativeTarget.startsWith("..") ||
      !existsSync(target)
    ) {
      fail(`${path}[${index}]`, "must resolve to an existing repository path");
    }
    const realTarget = realpathSync(target);
    const realRelativeTarget = relative(REAL_REPOSITORY_ROOT, realTarget);
    if (
      realRelativeTarget === "" ||
      realRelativeTarget.startsWith("..")
    ) {
      fail(`${path}[${index}]`, "must not escape through a repository symlink");
    }
  }
  return references;
}

function validateStandard(value, nodePath, standardIndex, snapshotDate) {
  const path = `${nodePath}.standards[${standardIndex}]`;
  const standard = record(value, path);
  exactKeys(standard, STANDARD_KEYS, path);

  const canonicalId = boundedString(standard.canonicalId, `${path}.canonicalId`, 128);
  if (!CANONICAL_STANDARD_ID_PATTERN.test(canonicalId)) {
    fail(`${path}.canonicalId`, "has an invalid canonical identifier");
  }
  const authority = boundedString(standard.authority, `${path}.authority`, 96);
  boundedString(standard.title, `${path}.title`, 256);
  boundedString(standard.revision, `${path}.revision`, 128);
  const authorityStatus = boundedString(
    standard.authorityStatus,
    `${path}.authorityStatus`,
    256,
  );
  const maturity = enumValue(
    standard.normalizedMaturity,
    MATURITIES,
    `${path}.normalizedMaturity`,
  );
  const specification = parseHttpsUrl(
    standard.specification,
    `${path}.specification`,
  );
  const checked = parseIsoDate(standard.statusCheckedAt, `${path}.statusCheckedAt`);
  const review = parseIsoDate(standard.reviewAfter, `${path}.reviewAfter`);

  if (checked !== snapshotDate) {
    fail(`${path}.statusCheckedAt`, "must equal the tree snapshot date");
  }
  if (review < snapshotDate || review < checked) {
    fail(`${path}.reviewAfter`, "must not be stale at the tree snapshot");
  }
  if (
    !MATURITY_STATUS_MARKERS[maturity].some((marker) =>
      authorityStatus.toLowerCase().includes(marker),
    )
  ) {
    fail(
      `${path}.authorityStatus`,
      `contradicts normalized maturity ${maturity}`,
    );
  }
  const lowerAuthorityStatus = authorityStatus.toLowerCase();
  if (
    maturity !== "draft" &&
    (/\binternet-draft\b/.test(lowerAuthorityStatus) ||
      /\bdraft\b/.test(lowerAuthorityStatus))
  ) {
    fail(`${path}.authorityStatus`, "contains a conflicting draft status");
  }
  if (
    maturity !== "candidate-recommendation" &&
    lowerAuthorityStatus.includes("candidate recommendation")
  ) {
    fail(
      `${path}.authorityStatus`,
      "contains a conflicting candidate-recommendation status",
    );
  }
  if (
    maturity !== "eol" &&
    (/\beol\b/.test(lowerAuthorityStatus) ||
      lowerAuthorityStatus.includes("end of life"))
  ) {
    fail(`${path}.authorityStatus`, "contains a conflicting end-of-life status");
  }
  if (maturity === "final" && lowerAuthorityStatus.includes("not final")) {
    fail(`${path}.authorityStatus`, "contains a conflicting not-final status");
  }
  const canonicalPin = STANDARD_CANONICAL_PINS[canonicalId];
  if (!canonicalPin) {
    fail(`${path}.canonicalId`, "has no exact v1 canonical pin");
  }
  for (const [field, expected] of Object.entries(canonicalPin)) {
    if (standard[field] !== expected) {
      fail(
        `${path}.${field}`,
        `must match the exact ${canonicalId} v1 pin`,
      );
    }
  }
  const checkedTime = Date.parse(`${checked}T00:00:00Z`);
  const reviewTime = Date.parse(`${review}T00:00:00Z`);
  const reviewDays = (reviewTime - checkedTime) / 86_400_000;
  const maximumReviewDays = STANDARD_REVIEW_MAX_DAYS[canonicalId] ?? 90;
  if (reviewDays > maximumReviewDays) {
    fail(
      `${path}.reviewAfter`,
      `exceeds the ${maximumReviewDays}-day v1 review horizon for ${canonicalId}`,
    );
  }
  const pathSegments = specification.pathname.split("/").filter(Boolean);
  if (
    pathSegments.includes("main") ||
    pathSegments.includes("master") ||
    pathSegments.includes("HEAD")
  ) {
    fail(`${path}.specification`, "must pin an immutable edition rather than a mutable branch");
  }
  if (
    specification.pathname.includes("/latest/") &&
    maturity !== "maintained-policy"
  ) {
    fail(`${path}.specification`, "mutable latest sources require maintained-policy status");
  }

  const fingerprint = JSON.stringify(
    Object.fromEntries(STANDARD_KEYS.map((key) => [key, standard[key]])),
  );
  return { canonicalId, maturity, fingerprint };
}

function validateAcceptance(value, nodePath, policy) {
  if (value === null) return null;
  const path = `${nodePath}.acceptance`;
  const acceptance = record(value, path);
  exactKeys(acceptance, ACCEPTANCE_KEYS, path);
  const targetEvidence = enumValue(
    acceptance.targetEvidence,
    EVIDENCE_LEVELS,
    `${path}.targetEvidence`,
  );
  if (targetEvidence !== "E5" && targetEvidence !== "E6") {
    fail(`${path}.targetEvidence`, "must target adoption or maintenance evidence");
  }
  const scopeBounds = stringArray(acceptance.scopeBounds, `${path}.scopeBounds`, {
    minimum: 1,
    maximum: 32,
    maxBytes: 256,
    sorted: true,
  });
  const scopeHash = boundedString(acceptance.scopeHash, `${path}.scopeHash`, 64);
  if (!SHA256_PATTERN.test(scopeHash)) {
    fail(`${path}.scopeHash`, "must be a lowercase SHA-256 digest");
  }
  const expectedScopeHash = createHash("sha256")
    .update(JSON.stringify(scopeBounds))
    .digest("hex");
  if (scopeHash !== expectedScopeHash) {
    fail(`${path}.scopeHash`, "does not commit to the exact scope bounds");
  }
  if (
    !Array.isArray(acceptance.coverageTargets) ||
    acceptance.coverageTargets.length < 1 ||
    acceptance.coverageTargets.length > 16
  ) {
    fail(`${path}.coverageTargets`, "must contain 1 through 16 coverage targets");
  }
  const coverageIds = [];
  for (const [index, value] of acceptance.coverageTargets.entries()) {
    const targetPath = `${path}.coverageTargets[${index}]`;
    const target = record(value, targetPath);
    exactKeys(target, COVERAGE_TARGET_KEYS, targetPath);
    const id = boundedString(target.id, `${targetPath}.id`, 128);
    if (!COVERAGE_TARGET_PATTERN.test(id)) {
      fail(`${targetPath}.id`, "must be a kebab-case coverage identifier");
    }
    coverageIds.push(id);
    const effectiveClusters = boundedInteger(
      target.minimumEffectiveClusters,
      `${targetPath}.minimumEffectiveClusters`,
      1,
      16,
    );
    const organizationRoots = boundedInteger(
      target.minimumOrganizationRoots,
      `${targetPath}.minimumOrganizationRoots`,
      1,
      16,
    );
    const implementationRoots = boundedInteger(
      target.minimumImplementationRoots,
      `${targetPath}.minimumImplementationRoots`,
      1,
      16,
    );
    const executionEnvironments = boundedInteger(
      target.minimumExecutionEnvironments,
      `${targetPath}.minimumExecutionEnvironments`,
      1,
      16,
    );
    boundedInteger(
      target.minimumCases,
      `${targetPath}.minimumCases`,
      1,
      1_000_000,
    );
    exactBoolean(
      target.requiresCheckerOrCorpusDigest,
      true,
      `${targetPath}.requiresCheckerOrCorpusDigest`,
    );
    if (effectiveClusters < policy.minimumEffectiveClusters) {
      fail(
        `${targetPath}.minimumEffectiveClusters`,
        "must meet the policy floor",
      );
    }
    if (organizationRoots < policy.minimumOrganizationRoots) {
      fail(
        `${targetPath}.minimumOrganizationRoots`,
        "must meet the policy floor",
      );
    }
    if (implementationRoots < policy.minimumImplementationRoots) {
      fail(
        `${targetPath}.minimumImplementationRoots`,
        "must meet the policy floor",
      );
    }
    if (executionEnvironments < policy.minimumExecutionEnvironments) {
      fail(
        `${targetPath}.minimumExecutionEnvironments`,
        "must meet the policy floor",
      );
    }
  }
  if (new Set(coverageIds).size !== coverageIds.length) {
    fail(`${path}.coverageTargets`, "must not contain duplicate IDs");
  }
  if (coverageIds.some((id, index) => index > 0 && coverageIds[index - 1] > id)) {
    fail(`${path}.coverageTargets`, "must be sorted by ID");
  }
  const effectiveClusters = boundedInteger(
    acceptance.minimumEffectiveClusters,
    `${path}.minimumEffectiveClusters`,
    1,
    16,
  );
  const organizationRoots = boundedInteger(
    acceptance.minimumOrganizationRoots,
    `${path}.minimumOrganizationRoots`,
    1,
    16,
  );
  const implementationRoots = boundedInteger(
    acceptance.minimumImplementationRoots,
    `${path}.minimumImplementationRoots`,
    1,
    16,
  );
  const executionEnvironments = boundedInteger(
    acceptance.minimumExecutionEnvironments,
    `${path}.minimumExecutionEnvironments`,
    1,
    16,
  );
  if (effectiveClusters < policy.minimumEffectiveClusters) {
    fail(`${path}.minimumEffectiveClusters`, "must meet the policy floor");
  }
  if (organizationRoots < policy.minimumOrganizationRoots) {
    fail(`${path}.minimumOrganizationRoots`, "must meet the policy floor");
  }
  if (implementationRoots < policy.minimumImplementationRoots) {
    fail(`${path}.minimumImplementationRoots`, "must meet the policy floor");
  }
  if (executionEnvironments < policy.minimumExecutionEnvironments) {
    fail(`${path}.minimumExecutionEnvironments`, "must meet the policy floor");
  }
  const adoptionReceiptTypes = stringArray(
    acceptance.adoptionReceiptTypes,
    `${path}.adoptionReceiptTypes`,
    { minimum: 1, maximum: 4, maxBytes: 64, sorted: true },
  );
  for (const [index, receiptType] of adoptionReceiptTypes.entries()) {
    if (!ADOPTION_RECEIPT_TYPES.has(receiptType)) {
      fail(`${path}.adoptionReceiptTypes[${index}]`, "has an unknown value");
    }
  }
  exactBoolean(
    acceptance.privateEscalationRequired,
    true,
    `${path}.privateEscalationRequired`,
  );
  exactBoolean(
    acceptance.prepublicationTriageRequired,
    true,
    `${path}.prepublicationTriageRequired`,
  );
  return acceptance;
}

function validateNode(value, index, snapshotDate, policy) {
  const path = `$.nodes[${index}]`;
  const node = record(value, path);
  exactKeys(node, NODE_KEYS, path);

  const id = boundedString(node.id, `${path}.id`, 128);
  if (!NODE_ID_PATTERN.test(id)) {
    fail(`${path}.id`, "must be a lowercase kebab-case identifier with an @version suffix");
  }
  if (id.includes("breakthrough")) {
    fail(`${path}.id`, "breakthrough is derived evidence, not a node kind");
  }
  boundedString(node.title, `${path}.title`, 160);
  boundedString(node.summary, `${path}.summary`, 512);
  const stage = enumValue(node.stage, STAGES, `${path}.stage`);
  const domain = enumValue(node.domain, DOMAINS, `${path}.domain`);
  if (!STAGE_DOMAINS[stage].has(domain)) {
    fail(`${path}.domain`, `is not valid for stage ${stage}`);
  }
  const prerequisites = stringArray(node.prerequisites, `${path}.prerequisites`, {
    maximum: 16,
    maxBytes: 128,
    pattern: NODE_ID_PATTERN,
    sorted: true,
  });
  const attainmentEvidence = enumValue(
    node.attainmentEvidence,
    EVIDENCE_LEVELS,
    `${path}.attainmentEvidence`,
  );
  const rewardEligibility = enumValue(
    node.rewardEligibility,
    REWARD_ELIGIBILITIES,
    `${path}.rewardEligibility`,
  );
  enumValue(
    node.defaultDisclosureLane,
    DISCLOSURE_LANES,
    `${path}.defaultDisclosureLane`,
  );
  stringArray(node.artifactRequirements, `${path}.artifactRequirements`, {
    minimum: 1,
    maximum: 16,
    maxBytes: 768,
  });
  stringArray(node.revalidationTriggers, `${path}.revalidationTriggers`, {
    minimum: 1,
    maximum: 16,
    maxBytes: 768,
  });

  if (!Array.isArray(node.standards) || node.standards.length > 8) {
    fail(`${path}.standards`, "must contain at most eight standards");
  }
  if (stage === "quest") {
    const forbiddenIndex = node.standards.findIndex(
      (standard) =>
        typeof standard === "object" &&
        standard !== null &&
        !Array.isArray(standard) &&
        SPONSOR_FORBIDDEN_MATURITIES.has(standard.normalizedMaturity),
    );
    if (forbiddenIndex !== -1) {
      fail(
        `${path}.standards[${forbiddenIndex}].normalizedMaturity`,
        "sponsor quests cannot directly target draft or candidate specifications",
      );
    }
  }
  const standards = node.standards.map((standard, standardIndex) =>
    validateStandard(standard, path, standardIndex, snapshotDate),
  );
  const canonicalIds = standards.map((standard) => standard.canonicalId);
  if (new Set(canonicalIds).size !== canonicalIds.length) {
    fail(`${path}.standards`, "must not contain duplicate canonical IDs");
  }
  if (canonicalIds.some((id, standardIndex) => standardIndex > 0 && canonicalIds[standardIndex - 1] > id)) {
    fail(`${path}.standards`, "must be sorted by canonical ID");
  }

  validateRepositoryReferences(node.repositoryReferences, `${path}.repositoryReferences`);
  const acceptance = validateAcceptance(node.acceptance, path, policy);

  if (stage === "quest") {
    if (rewardEligibility !== "sponsor-milestones") {
      fail(`${path}.rewardEligibility`, "quest nodes must use sponsor milestones");
    }
    if (attainmentEvidence !== "E5" && attainmentEvidence !== "E6") {
      fail(`${path}.attainmentEvidence`, "quest nodes must require adoption or maintenance evidence");
    }
    if (acceptance === null) {
      fail(`${path}.acceptance`, "quest nodes require bounded acceptance");
    }
    if (acceptance.targetEvidence !== attainmentEvidence) {
      fail(
        `${path}.acceptance.targetEvidence`,
        "must equal the quest attainment evidence",
      );
    }
    if (standards.length === 0) {
      fail(`${path}.standards`, "quest nodes must pin at least one standard");
    }
    if (standards.some(({ maturity }) => SPONSOR_FORBIDDEN_MATURITIES.has(maturity))) {
      fail(`${path}.standards`, "sponsor quests cannot directly target draft or candidate specifications");
    }
  } else {
    if (rewardEligibility !== "qualification-only") {
      fail(`${path}.rewardEligibility`, "non-quest nodes cannot be reward eligible");
    }
    if (acceptance !== null) {
      fail(`${path}.acceptance`, "non-quest nodes must not define quest acceptance");
    }
  }

  const normativeDigest = createHash("sha256")
    .update(canonicalJson(normativeNodeProjection(node)))
    .digest("hex");
  return { id, stage, prerequisites, standards, normativeDigest };
}

function validatePolicy(value) {
  const path = "$.policy";
  const policy = record(value, path);
  exactKeys(policy, POLICY_KEYS, path);

  const edgeTypes = stringArray(
    policy.artifactEdgeTypes,
    `${path}.artifactEdgeTypes`,
    { minimum: 1, maximum: 32, maxBytes: 32, sorted: true },
  );
  if (
    edgeTypes.length !== REQUIRED_ARTIFACT_EDGE_TYPES.length ||
    edgeTypes.some((edge, index) => edge !== REQUIRED_ARTIFACT_EDGE_TYPES[index])
  ) {
    fail(`${path}.artifactEdgeTypes`, "must contain the complete v1 edge set");
  }

  const breakthrough = record(
    policy.breakthroughRecognition,
    `${path}.breakthroughRecognition`,
  );
  exactKeys(breakthrough, BREAKTHROUGH_KEYS, `${path}.breakthroughRecognition`);
  exactBoolean(
    breakthrough.authorSelected,
    false,
    `${path}.breakthroughRecognition.authorSelected`,
  );
  if (breakthrough.minimumEvidenceLevel !== "E3") {
    fail(
      `${path}.breakthroughRecognition.minimumEvidenceLevel`,
      "must equal E3",
    );
  }
  exactBoolean(
    breakthrough.requiresPriorArtDelta,
    true,
    `${path}.breakthroughRecognition.requiresPriorArtDelta`,
  );
  exactBoolean(
    breakthrough.requiresAdoptionOrDescendantImpact,
    true,
    `${path}.breakthroughRecognition.requiresAdoptionOrDescendantImpact`,
  );

  const funding = record(policy.funding, `${path}.funding`);
  exactKeys(funding, FUNDING_KEYS, `${path}.funding`);
  exactBoolean(
    funding.skillUnlockCreatesReward,
    false,
    `${path}.funding.skillUnlockCreatesReward`,
  );
  if (funding.externalWorkDefault !== "sponsor-escrow") {
    fail(`${path}.funding.externalWorkDefault`, "must equal sponsor-escrow");
  }
  if (funding.protocolIssuanceGate !== "recursive-useful-work-only") {
    fail(
      `${path}.funding.protocolIssuanceGate`,
      "must equal recursive-useful-work-only",
    );
  }
  exactBoolean(
    funding.timeAloneUnlocksEvidence,
    false,
    `${path}.funding.timeAloneUnlocksEvidence`,
  );

  const independence = record(policy.independence, `${path}.independence`);
  exactKeys(independence, INDEPENDENCE_KEYS, `${path}.independence`);
  const minimumEffectiveClusters = boundedInteger(
    independence.minimumEffectiveClusters,
    `${path}.independence.minimumEffectiveClusters`,
    3,
    16,
  );
  const minimumOrganizationRoots = boundedInteger(
    independence.minimumOrganizationRoots,
    `${path}.independence.minimumOrganizationRoots`,
    2,
    16,
  );
  const minimumImplementationRoots = boundedInteger(
    independence.minimumImplementationRoots,
    `${path}.independence.minimumImplementationRoots`,
    2,
    16,
  );
  const minimumExecutionEnvironments = boundedInteger(
    independence.minimumExecutionEnvironments,
    `${path}.independence.minimumExecutionEnvironments`,
    2,
    16,
  );
  exactBoolean(
    independence.assignmentAfterArtifactFreeze,
    true,
    `${path}.independence.assignmentAfterArtifactFreeze`,
  );
  exactBoolean(
    independence.reviewPayOutcomeIndependent,
    true,
    `${path}.independence.reviewPayOutcomeIndependent`,
  );
  exactBoolean(
    independence.rawAddressCountIsEvidence,
    false,
    `${path}.independence.rawAddressCountIsEvidence`,
  );

  const disclosure = record(policy.disclosure, `${path}.disclosure`);
  exactKeys(disclosure, DISCLOSURE_KEYS, `${path}.disclosure`);
  exactBoolean(
    disclosure.safetyIsHardGate,
    true,
    `${path}.disclosure.safetyIsHardGate`,
  );
  if (
    disclosure.unknownSecurityImpactEscalatesTo !==
    "private-coordinated-repair"
  ) {
    fail(
      `${path}.disclosure.unknownSecurityImpactEscalatesTo`,
      "must equal private-coordinated-repair",
    );
  }
  exactBoolean(
    disclosure.publicExploitPlaintextAllowed,
    false,
    `${path}.disclosure.publicExploitPlaintextAllowed`,
  );
  exactBoolean(
    disclosure.vendorHasPayoutVeto,
    false,
    `${path}.disclosure.vendorHasPayoutVeto`,
  );

  if (
    !Array.isArray(policy.milestones) ||
    policy.milestones.length !== EXPECTED_MILESTONES.length
  ) {
    fail(`${path}.milestones`, "must contain the seven v1 evidence levels");
  }
  let rewardBps = 0;
  for (const [index, expected] of EXPECTED_MILESTONES.entries()) {
    const milestonePath = `${path}.milestones[${index}]`;
    const milestone = record(policy.milestones[index], milestonePath);
    exactKeys(milestone, MILESTONE_KEYS, milestonePath);
    for (const key of MILESTONE_KEYS) {
      if (milestone[key] !== expected[key]) {
        fail(`${milestonePath}.${key}`, `must equal ${JSON.stringify(expected[key])}`);
      }
    }
    rewardBps += milestone.rewardBps;
  }
  const challengeReserveBps = boundedInteger(
    policy.challengeReserveBps,
    `${path}.challengeReserveBps`,
    0,
    10_000,
  );
  if (rewardBps + challengeReserveBps !== 10_000) {
    fail(path, "milestone tranches plus challenge reserve must equal 10000 bps");
  }
  if (challengeReserveBps !== 1500) {
    fail(`${path}.challengeReserveBps`, "must preserve the v1 1500 bps reserve");
  }

  return {
    minimumEffectiveClusters,
    minimumOrganizationRoots,
    minimumImplementationRoots,
    minimumExecutionEnvironments,
  };
}

function validateGraph(nodes, roots) {
  const nodeById = new Map(nodes.map((node) => [node.id, node]));
  for (const node of nodes) {
    for (const prerequisite of node.prerequisites) {
      if (prerequisite === node.id) {
        fail("$.nodes", `${node.id} cannot require itself`);
      }
      const required = nodeById.get(prerequisite);
      if (!required) {
        fail("$.nodes", `${node.id} has missing prerequisite ${prerequisite}`);
      }
      if (STAGE_RANK.get(required.stage) > STAGE_RANK.get(node.stage)) {
        fail(
          "$.nodes",
          `${node.id} cannot require later-stage node ${prerequisite}`,
        );
      }
    }
  }

  const color = new Map();
  const stack = [];
  const visit = (id) => {
    const state = color.get(id) ?? 0;
    if (state === 1) {
      const start = stack.indexOf(id);
      const cycle = [...stack.slice(start), id].join(" -> ");
      fail("$.nodes", `prerequisite cycle detected: ${cycle}`);
    }
    if (state === 2) return;
    color.set(id, 1);
    stack.push(id);
    for (const prerequisite of nodeById.get(id).prerequisites) visit(prerequisite);
    stack.pop();
    color.set(id, 2);
  };
  for (const node of nodes) visit(node.id);

  const zeroPrerequisiteIds = nodes
    .filter((node) => node.prerequisites.length === 0)
    .map((node) => node.id);
  if (
    roots.length !== zeroPrerequisiteIds.length ||
    roots.some((root, index) => root !== zeroPrerequisiteIds[index])
  ) {
    fail("$.roots", "must equal the sorted set of zero-prerequisite nodes");
  }
  for (const root of roots) {
    if (!nodeById.has(root)) fail("$.roots", `unknown root ${root}`);
  }

  const dependents = new Map(nodes.map((node) => [node.id, []]));
  let edgeCount = 0;
  for (const node of nodes) {
    for (const prerequisite of node.prerequisites) {
      dependents.get(prerequisite).push(node.id);
      edgeCount += 1;
    }
  }
  for (const [id, children] of dependents) {
    if (children.length > CONSTRUCTIVE_TREE_MAX_FAN_OUT) {
      fail(
        "$.nodes",
        `${id} exceeds the v1 fan-out limit of ${CONSTRUCTIVE_TREE_MAX_FAN_OUT}`,
      );
    }
  }

  const depthMemo = new Map();
  const depthOf = (id) => {
    if (depthMemo.has(id)) return depthMemo.get(id);
    const prerequisites = nodeById.get(id).prerequisites;
    const depth =
      prerequisites.length === 0
        ? 1
        : 1 + Math.max(...prerequisites.map(depthOf));
    depthMemo.set(id, depth);
    return depth;
  };
  const maxDepth = Math.max(...nodes.map((node) => depthOf(node.id)));
  if (maxDepth > CONSTRUCTIVE_TREE_MAX_DEPTH) {
    fail(
      "$.nodes",
      `graph depth ${maxDepth} exceeds the v1 limit of ${CONSTRUCTIVE_TREE_MAX_DEPTH}`,
    );
  }

  const reached = new Set();
  const queue = [...roots];
  while (queue.length > 0) {
    const id = queue.shift();
    if (reached.has(id)) continue;
    reached.add(id);
    queue.push(...dependents.get(id));
  }
  if (reached.size !== nodes.length) {
    const missing = nodes.find((node) => !reached.has(node.id));
    fail("$.nodes", `node ${missing.id} is unreachable from declared roots`);
  }
  return { edgeCount, maxDepth };
}

export function validateConstructiveIntelligenceTree(value) {
  const tree = record(value, "$");
  exactKeys(tree, TOP_LEVEL_KEYS, "$");
  if (tree.schema !== CONSTRUCTIVE_TREE_SCHEMA) {
    fail("$.schema", `must equal ${CONSTRUCTIVE_TREE_SCHEMA}`);
  }
  exactBoolean(tree.authoritative, false, "$.authoritative");
  exactBoolean(tree.networkObserved, false, "$.networkObserved");
  exactBoolean(tree.rewardBearing, false, "$.rewardBearing");
  const snapshotDate = parseIsoDate(tree.snapshotDate, "$.snapshotDate");
  const policyVersion = boundedString(tree.policyVersion, "$.policyVersion", 32);
  if (!POLICY_VERSION_PATTERN.test(policyVersion) || policyVersion !== "1.0.0") {
    fail("$.policyVersion", "must equal 1.0.0 for schema v1");
  }

  const boundary = record(tree.releaseBoundary, "$.releaseBoundary");
  exactKeys(boundary, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) {
    exactBoolean(boundary[key], false, `$.releaseBoundary.${key}`);
  }

  const policy = validatePolicy(tree.policy);
  const roots = stringArray(tree.roots, "$.roots", {
    minimum: 1,
    maximum: 16,
    maxBytes: 128,
    pattern: NODE_ID_PATTERN,
    sorted: true,
  });
  if (!Array.isArray(tree.nodes) || tree.nodes.length < 1 || tree.nodes.length > 64) {
    fail("$.nodes", "must contain 1 through 64 nodes");
  }
  const nodes = tree.nodes.map((node, index) =>
    validateNode(node, index, snapshotDate, policy),
  );
  const ids = nodes.map((node) => node.id);
  if (new Set(ids).size !== ids.length) fail("$.nodes", "must not contain duplicate IDs");
  if (ids.some((id, index) => index > 0 && ids[index - 1] > id)) {
    fail("$.nodes", "must be sorted by ID");
  }
  const standardFingerprints = new Map();
  for (const node of nodes) {
    for (const standard of node.standards) {
      const previous = standardFingerprints.get(standard.canonicalId);
      if (previous !== undefined && previous !== standard.fingerprint) {
        fail(
          "$.nodes",
          `canonical standard ${standard.canonicalId} has conflicting metadata`,
        );
      }
      standardFingerprints.set(standard.canonicalId, standard.fingerprint);
    }
  }
  const { edgeCount, maxDepth } = validateGraph(nodes, roots);

  const questIds = nodes
    .filter((node) => node.stage === "quest")
    .map((node) => node.id);
  const requiredQuestIds = Object.keys(REVIEWED_QUEST_TEMPLATE_DIGESTS);
  if (
    questIds.length !== requiredQuestIds.length ||
    questIds.some((id, index) => id !== requiredQuestIds[index])
  ) {
    fail("$.nodes", "must contain exactly the three reviewed v1 quest templates");
  }
  for (const quest of nodes.filter((node) => node.stage === "quest")) {
    if (quest.normativeDigest !== REVIEWED_QUEST_TEMPLATE_DIGESTS[quest.id]) {
      fail(
        "$.nodes",
        `${quest.id} differs from its reviewed v1 template; use a new versioned ID`,
      );
    }
  }
  const normativeDigest = createHash("sha256")
    .update(canonicalJson(normativeTreeProjection(tree)))
    .digest("hex");
  if (normativeDigest !== REVIEWED_V1_NORMATIVE_DIGEST) {
    fail(
      "$",
      "normative v1 policy or capability content changed; use reviewed versioned identifiers",
    );
  }

  return {
    schema: tree.schema,
    policyVersion,
    snapshotDate,
    nodeCount: nodes.length,
    edgeCount,
    maxDepth,
    questCount: nodes.filter((node) => node.stage === "quest").length,
    normativeDigest,
  };
}

export function validateConstructiveIntelligenceTreeForUse(value, asOfDate) {
  const result = validateConstructiveIntelligenceTree(value);
  const checkedForUseAt = parseIsoDate(asOfDate, "$.checkedForUseAt");
  if (value.snapshotDate > checkedForUseAt) {
    fail(
      "$.snapshotDate",
      `cannot be after active-use date ${checkedForUseAt}`,
    );
  }
  for (const [nodeIndex, node] of value.nodes.entries()) {
    for (const [standardIndex, standard] of node.standards.entries()) {
      if (standard.reviewAfter < checkedForUseAt) {
        fail(
          `$.nodes[${nodeIndex}].standards[${standardIndex}].reviewAfter`,
          `expired before active-use date ${checkedForUseAt}; revalidate the authority snapshot`,
        );
      }
    }
  }
  return { ...result, checkedForUseAt };
}

function rejectDuplicateJsonKeys(raw) {
  let offset = 0;

  const whitespace = () => {
    while (/\s/.test(raw[offset] ?? "")) offset += 1;
  };

  const scanString = () => {
    const start = offset;
    offset += 1;
    while (offset < raw.length) {
      if (raw[offset] === "\\") {
        offset += 2;
        continue;
      }
      if (raw[offset] === '"') {
        offset += 1;
        return JSON.parse(raw.slice(start, offset));
      }
      offset += 1;
    }
    fail("$", "unterminated JSON string");
  };

  const scanValue = (path) => {
    whitespace();
    const token = raw[offset];
    if (token === "{") {
      offset += 1;
      whitespace();
      const keys = new Set();
      if (raw[offset] === "}") {
        offset += 1;
        return;
      }
      while (offset < raw.length) {
        whitespace();
        const key = scanString();
        const keyPath = `${path}.${key}`;
        if (keys.has(key)) fail(keyPath, "duplicate JSON object key");
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
    while (
      offset < raw.length &&
      !/[\s,\]}]/.test(raw[offset])
    ) {
      offset += 1;
    }
  };

  scanValue("$");
}

function rejectExcessiveJsonNesting(raw) {
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
      if (depth > 64) fail("$", "JSON nesting exceeds the v1 limit of 64");
    } else if (character === "}" || character === "]") {
      depth -= 1;
    }
  }
}

export function parseAndValidateConstructiveIntelligenceTree(raw) {
  if (typeof raw !== "string") fail("$", "raw document must be a string");
  if (Buffer.byteLength(raw, "utf8") > CONSTRUCTIVE_TREE_MAX_BYTES) {
    fail("$", `raw document exceeds ${CONSTRUCTIVE_TREE_MAX_BYTES} UTF-8 bytes`);
  }
  rejectExcessiveJsonNesting(raw);
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    fail("$", `invalid JSON: ${error.message}`);
  }
  rejectDuplicateJsonKeys(raw);
  return validateConstructiveIntelligenceTree(parsed);
}

function runCli() {
  if (process.argv.length !== 3) {
    console.error(
      "usage: node scripts/validate-constructive-intelligence-tree.mjs PATH",
    );
    process.exitCode = 2;
    return;
  }
  let raw;
  try {
    raw = readFileSync(resolve(process.argv[2]), "utf8");
  } catch (error) {
    console.error(`constructive-intelligence tree: FAIL: ${error.message}`);
    process.exitCode = 1;
    return;
  }
  try {
    const result = parseAndValidateConstructiveIntelligenceTree(raw);
    console.log(
      `constructive-intelligence tree: PASS (${result.nodeCount} nodes, ${result.edgeCount} prerequisite edges, depth ${result.maxDepth}, ${result.questCount} quests)`,
    );
  } catch (error) {
    console.error(`constructive-intelligence tree: FAIL: ${error.message}`);
    process.exitCode = 1;
  }
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runCli();
}
