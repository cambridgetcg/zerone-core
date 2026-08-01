import { createHash } from "node:crypto";
import { existsSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

// Offline by construction: reference URLs are exact allowlisted locators, never fetched.
export const LIFE_SCIENCES_SCHEMA =
  "zerone.constructive-intelligence-life-sciences/v0";
export const LIFE_SCIENCES_EVIDENCE_FIXTURE_SCHEMA =
  "zerone.constructive-intelligence-life-sciences-evidence-fixture/v0";
export const LIFE_SCIENCES_MAX_BYTES = 196_608;
export const LIFE_SCIENCES_MIN_NODES = 12;
export const LIFE_SCIENCES_MAX_NODES = 20;
export const LIFE_SCIENCES_MAX_DEPTH = 8;
export const LIFE_SCIENCES_MAX_PREREQUISITES = 4;
export const LIFE_SCIENCES_MAX_FAN_OUT = 8;
export const BASE_TREE_SHA256 =
  "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf";
export const MONEY_KARMA_CONSTITUTION_SCHEMA =
  "zerone.money-karma.constitution/v1";
export const MONEY_KARMA_CONSTITUTION_SHA256 =
  "f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24";

const TOP_LEVEL_KEYS = [
  "schema",
  "status",
  "mode",
  "authoritative",
  "networkObserved",
  "rewardBearing",
  "snapshotDate",
  "baseTreeBinding",
  "constitutionBinding",
  "releaseBoundary",
  "economics",
  "scope",
  "fixtureBoundary",
  "evidencePolicy",
  "independence",
  "challengePolicy",
  "attestationBoundary",
  "nonImplicationWalls",
  "references",
  "roots",
  "nodes",
];
const BASE_TREE_BINDING_KEYS = ["schema", "sha256", "policyVersion"];
const CONSTITUTION_BINDING_KEYS = ["schema", "documentSha256"];
const RELEASE_BOUNDARY_KEYS = [
  "addsConsensusBehavior",
  "activatesRewards",
  "movesFunds",
  "grantsQualification",
  "grantsGovernanceAuthority",
  "createsKarmaMagnitude",
  "createsTransferableKarma",
  "authorizesExperiments",
  "authorizesClinicalDecisions",
  "performsNetworkRequests",
  "publishesRestrictedEvidence",
  "marksReleaseReady",
];
const ECONOMICS_KEYS = [
  "effect",
  "amount",
  "denom",
  "rewardMultiplier",
  "escrowReference",
];
const SCOPE_KEYS = [
  "riskClass",
  "allowedWork",
  "refusedTopics",
  "unknownRiskDisposition",
  "privateEscalationContent",
];
const FIXTURE_BOUNDARY_KEYS = [
  "syntheticIdentifiersOnly",
  "invalidTldRequired",
  "sequencePayloadsAllowed",
  "operationalProtocolsAllowed",
  "identifiableHumanDataAllowed",
  "rawHumanGenomeAllowed",
  "maximumArtifactMetadataBytes",
];
const EVIDENCE_POLICY_KEYS = [
  "equivalenceToCoreTree",
  "equivalenceToPoca",
  "ladder",
];
const EVIDENCE_LEVEL_KEYS = ["level", "name", "economicTreatment"];
const INDEPENDENCE_KEYS = [
  "minimumEffectiveControlClusters",
  "minimumOrganizationRoots",
  "minimumMethodRoots",
  "minimumContextRoots",
  "rawIdentityCountIsEvidence",
  "beneficialControlDisclosureRequired",
  "futureEligibilityRequiresExternalControllerAttestation",
];
const CHALLENGE_POLICY_KEYS = [
  "openChallengeRequired",
  "unresolvedChallengeBlocksCrown",
  "upheldChallengeBlocksCrown",
  "challengeCanCreateReward",
  "futureEligibilityRequiresAdjudicationReceipt",
];
const ATTESTATION_BOUNDARY_KEYS = [
  "controlDisclosures",
  "challengeStatus",
  "establishesControllerIndependence",
  "establishesChallengeClosure",
];
const WALL_KEYS = ["id", "premise", "doesNotEstablish"];
const REFERENCE_KEYS = ["canonicalId", "authority", "title", "specification"];
const NODE_KEYS = [
  "id",
  "title",
  "stage",
  "branch",
  "summary",
  "prerequisites",
  "evidenceFloor",
  "claimClass",
  "permittedConclusions",
  "forbiddenConclusions",
  "artifactRequirements",
  "revalidationTriggers",
  "referenceIds",
  "crown",
];
const FIXTURE_KEYS = [
  "schema",
  "id",
  "profileSchema",
  "nodeId",
  "requestedConclusions",
  "evidenceLevel",
  "artifacts",
  "controls",
  "challengeStatus",
];
const ARTIFACT_KEYS = [
  "id",
  "uri",
  "sha256",
  "observationKind",
  "containsSequence",
  "containsOperationalProtocol",
  "humanDataClass",
  "riskTopics",
];
const CONTROL_KEYS = [
  "contributorId",
  "effectiveControlCluster",
  "organizationRoot",
  "methodRoot",
  "contextRoot",
];

const REQUIRED_ALLOWED_WORK = [
  "benign-non-pathogenic-biomolecule-analysis",
  "benign-non-pathogenic-enzyme-analysis",
  "cell-free-expression-analysis",
  "in-silico-model-evaluation",
  "non-identifiable-aggregate-expression-analysis",
  "ordinary-low-risk-expression-analysis",
  "sanitized-metadata-only-replication",
];
const REQUIRED_REFUSED_TOPICS = [
  "CLINICAL_DECISIONS",
  "GERMLINE",
  "HOST_RANGE",
  "IDENTIFIABLE_HUMAN_DATA",
  "IMMUNE_EVASION",
  "OPERATIONAL_WET_LAB_ENABLEMENT",
  "PATHOGENS",
  "RAW_HUMAN_GENOME",
  "TOXINS",
  "VIRULENCE",
];
const EXPECTED_EVIDENCE_LADDER = [
  {
    level: "LS0",
    name: "scope-and-risk-commitment",
    economicTreatment: "NONE",
  },
  {
    level: "LS1",
    name: "inspectable-provenance",
    economicTreatment: "NONE",
  },
  {
    level: "LS2",
    name: "bounded-method-check",
    economicTreatment: "NONE",
  },
  {
    level: "LS3",
    name: "independent-reproduction",
    economicTreatment: "NONE",
  },
  {
    level: "LS4",
    name: "prospective-or-adversarial-validation",
    economicTreatment: "NONE",
  },
  {
    level: "LS5",
    name: "independent-cross-context-replication",
    economicTreatment: "NONE",
  },
];
const EXPECTED_WALLS = [
  {
    id: "ASSOCIATION_NOT_REGULATION_OR_CAUSALITY",
    premise: "REGULATORY_ASSOCIATION",
    doesNotEstablish: ["CAUSAL_REGULATION"],
  },
  {
    id: "INSILICO_NOT_PROSPECTIVE_EXPERIMENT",
    premise: "IN_SILICO_PREDICTION",
    doesNotEstablish: ["PROSPECTIVE_VALIDATION"],
  },
  {
    id: "MRNA_NOT_PROTEIN_ABUNDANCE_LOCALIZATION_OR_FUNCTION",
    premise: "TRANSCRIPT_ABUNDANCE",
    doesNotEstablish: [
      "PROTEIN_ABUNDANCE",
      "PROTEIN_FUNCTION",
      "PROTEIN_LOCALIZATION",
    ],
  },
  {
    id: "STATIC_COORDINATES_NOT_FOLDING_PATHWAY_OR_KINETICS",
    premise: "STATIC_STRUCTURE",
    doesNotEstablish: ["FOLDING_KINETICS", "FOLDING_PATHWAY"],
  },
];

const REFERENCE_LOCATORS = Object.freeze({
  "casp:prediction-center": Object.freeze({
    authority: "CASP Prediction Center",
    title: "Critical Assessment of protein Structure Prediction",
    specification: "https://predictioncenter.org/",
  }),
  "ebi:miame-minseqe": Object.freeze({
    authority: "EMBL-EBI",
    title: "MIAME and MINSEQE data-sharing guidance",
    specification:
      "https://www.ebi.ac.uk/training/online/courses/functional-genomics-iii-submitting-data/what-kind-of-data-should-we-share/",
  }),
  "encode:data-standards": Object.freeze({
    authority: "ENCODE Project",
    title: "ENCODE data standards",
    specification: "https://www.encodeproject.org/data-standards/",
  }),
  "gene-ontology:evidence-codes": Object.freeze({
    authority: "Gene Ontology Consortium",
    title: "Guide to Gene Ontology evidence codes",
    specification: "https://geneontology.org/docs/guide-go-evidence-codes/",
  }),
  "ncbi:geo-hts": Object.freeze({
    authority: "NCBI",
    title: "GEO high-throughput sequence submission guidance",
    specification: "https://www.ncbi.nlm.nih.gov/geo/info/seq.html",
  }),
  "ncbi:sra-metadata": Object.freeze({
    authority: "NCBI",
    title: "Sequence Read Archive metadata guidance",
    specification: "https://www.ncbi.nlm.nih.gov/sra/docs/submitmeta/",
  }),
  "uniprot:uniprotkb": Object.freeze({
    authority: "UniProt Consortium",
    title: "UniProt Knowledgebase",
    specification: "https://www.uniprot.org/help/uniprotkb",
  }),
  "who:lab-biosafety-manual-4": Object.freeze({
    authority: "World Health Organization",
    title: "Laboratory biosafety manual, fourth edition",
    specification: "https://www.who.int/publications/i/item/9789240011311",
  }),
  "wwpdb:pdbx-mmcif": Object.freeze({
    authority: "Worldwide Protein Data Bank",
    title: "PDBx/mmCIF resources",
    specification: "https://mmcif.wwpdb.org/",
  }),
});

const STAGES = new Set([
  "foundation",
  "measurement",
  "inference",
  "validation",
  "integration",
  "crown",
]);
const STAGE_RANK = new Map([
  ["foundation", 0],
  ["measurement", 1],
  ["inference", 2],
  ["validation", 3],
  ["integration", 4],
  ["crown", 5],
]);
const BRANCHES = new Set([
  "biomolecule-foundations",
  "protein-folding",
  "gene-expression",
  "integration",
]);
const EVIDENCE_LEVELS = new Set(EXPECTED_EVIDENCE_LADDER.map(({ level }) => level));
const EVIDENCE_RANK = new Map(
  EXPECTED_EVIDENCE_LADDER.map(({ level }, index) => [level, index]),
);
const CLAIM_CLASSES = new Set([
  "ASSAY_DESIGN",
  "CONTEXT_NORMALIZATION",
  "CONTEXT_PROVENANCE",
  "CROSS_CONTEXT_REPLICATION",
  "FOLD_BENCHMARK",
  "FOLD_ENSEMBLE",
  "FOLD_FUNCTION",
  "FOLD_MODEL",
  "FUNCTION_ANNOTATION",
  "GENOTYPE_EXPRESSION_PHENOTYPE",
  "PROTEIN_MEASUREMENT",
  "REGULATORY_ASSOCIATION",
  "REGULATORY_CAUSALITY",
  "STATIC_STRUCTURE",
  "STUDY_PROVENANCE",
  "TRANSCRIPT_MEASUREMENT",
  "VARIANT_FOLD_FUNCTION",
]);
const CONCLUSIONS = new Set([
  "ANNOTATED_FUNCTION",
  "ASSAY_SCOPE",
  "CAUSAL_REGULATION",
  "CONTEXT_PROVENANCE",
  "FOLDING_KINETICS",
  "FOLDING_PATHWAY",
  "FOLD_ENSEMBLE_UNCERTAINTY",
  "FOLD_MODEL_CONFIDENCE",
  "GENOTYPE_EXPRESSION_PHENOTYPE_CHAIN",
  "INDEPENDENT_CROSS_CONTEXT_REPLICATION",
  "NORMALIZED_COMPARISON",
  "PROSPECTIVE_VALIDATION",
  "PROTEIN_ABUNDANCE",
  "PROTEIN_FUNCTION",
  "PROTEIN_LOCALIZATION",
  "REGULATORY_ASSOCIATION",
  "STATIC_STRUCTURE",
  "STUDY_PROVENANCE",
  "TRANSCRIPT_ABUNDANCE",
  "VARIANT_FOLD_FUNCTION_CHAIN",
]);
const OBSERVATION_KINDS = new Set([
  "ANNOTATION_EVIDENCE",
  "ASSAY_METADATA",
  "CAUSAL_PERTURBATION",
  "CONTEXT_METADATA",
  "CROSS_CONTEXT_REPLICATION",
  "FOLDING_DYNAMICS",
  "FOLD_ENSEMBLE",
  "FOLD_MODEL",
  "FUNCTIONAL_EVIDENCE",
  "NORMALIZED_EXPRESSION",
  "PROSPECTIVE_EVALUATION",
  "PROTEIN_ABUNDANCE",
  "PROTEIN_LOCALIZATION",
  "REGULATORY_ASSOCIATION",
  "STATIC_STRUCTURE",
  "STUDY_METADATA",
  "TRANSCRIPT_ABUNDANCE",
]);
const HUMAN_DATA_CLASSES = new Set([
  "NONE",
  "NON_IDENTIFIABLE_AGGREGATE",
  "IDENTIFIABLE",
  "RAW_HUMAN_GENOME",
]);
const CHALLENGE_STATUSES = new Set(["CLEAR", "OPEN", "UNRESOLVED", "UPHELD"]);
const NODE_ID_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*@1$/;
const TOKEN_PATTERN = /^[A-Z0-9]+(?:_[A-Z0-9]+)*$/;
const SIMPLE_ID_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const SHA256_PATTERN = /^[0-9a-f]{64}$/;
const DNA_LIKE_PAYLOAD_PATTERN = /(?:^|[^A-Za-z])[ACGTUN]{24,}(?:$|[^A-Za-z])/i;
const SCRIPT_PATH = fileURLToPath(import.meta.url);
const BASE_TREE_PATH = resolve(
  dirname(SCRIPT_PATH),
  "../public/standards/constructive-intelligence-tree.v1.json",
);
const MONEY_KARMA_CONSTITUTION_PATH = resolve(
  dirname(SCRIPT_PATH),
  "../../docs/constitution/money-karma-v1.json",
);

const WALL_FORBIDDEN_BY_CLAIM_CLASS = Object.freeze({
  STATIC_STRUCTURE: Object.freeze(["FOLDING_KINETICS", "FOLDING_PATHWAY"]),
  TRANSCRIPT_MEASUREMENT: Object.freeze([
    "PROTEIN_ABUNDANCE",
    "PROTEIN_FUNCTION",
    "PROTEIN_LOCALIZATION",
  ]),
  REGULATORY_ASSOCIATION: Object.freeze(["CAUSAL_REGULATION"]),
  FOLD_MODEL: Object.freeze(["PROSPECTIVE_VALIDATION"]),
  FOLD_ENSEMBLE: Object.freeze(["PROSPECTIVE_VALIDATION"]),
  FOLD_BENCHMARK: Object.freeze(["PROSPECTIVE_VALIDATION"]),
});
const CONCLUSION_OBSERVATIONS = Object.freeze({
  ANNOTATED_FUNCTION: Object.freeze(["ANNOTATION_EVIDENCE"]),
  ASSAY_SCOPE: Object.freeze(["ASSAY_METADATA"]),
  CAUSAL_REGULATION: Object.freeze(["CAUSAL_PERTURBATION"]),
  CONTEXT_PROVENANCE: Object.freeze(["CONTEXT_METADATA"]),
  FOLDING_KINETICS: Object.freeze(["FOLDING_DYNAMICS"]),
  FOLDING_PATHWAY: Object.freeze(["FOLDING_DYNAMICS"]),
  FOLD_ENSEMBLE_UNCERTAINTY: Object.freeze(["FOLD_ENSEMBLE"]),
  FOLD_MODEL_CONFIDENCE: Object.freeze(["FOLD_MODEL"]),
  GENOTYPE_EXPRESSION_PHENOTYPE_CHAIN: Object.freeze([
    "CAUSAL_PERTURBATION",
    "NORMALIZED_EXPRESSION",
    "PROTEIN_ABUNDANCE",
  ]),
  INDEPENDENT_CROSS_CONTEXT_REPLICATION: Object.freeze([
    "CROSS_CONTEXT_REPLICATION",
  ]),
  NORMALIZED_COMPARISON: Object.freeze(["NORMALIZED_EXPRESSION"]),
  PROSPECTIVE_VALIDATION: Object.freeze(["PROSPECTIVE_EVALUATION"]),
  PROTEIN_ABUNDANCE: Object.freeze(["PROTEIN_ABUNDANCE"]),
  PROTEIN_FUNCTION: Object.freeze(["FUNCTIONAL_EVIDENCE"]),
  PROTEIN_LOCALIZATION: Object.freeze(["PROTEIN_LOCALIZATION"]),
  REGULATORY_ASSOCIATION: Object.freeze(["REGULATORY_ASSOCIATION"]),
  STATIC_STRUCTURE: Object.freeze(["STATIC_STRUCTURE"]),
  STUDY_PROVENANCE: Object.freeze(["STUDY_METADATA"]),
  TRANSCRIPT_ABUNDANCE: Object.freeze(["TRANSCRIPT_ABUNDANCE"]),
  VARIANT_FOLD_FUNCTION_CHAIN: Object.freeze([
    "FOLD_MODEL",
    "FUNCTIONAL_EVIDENCE",
    "PROSPECTIVE_EVALUATION",
    "STATIC_STRUCTURE",
  ]),
});

export class LifeSciencesValidationError extends Error {
  constructor(path, message) {
    super(`${path}: ${message}`);
    this.name = "LifeSciencesValidationError";
    this.path = path;
  }
}
function fail(path, message) {
  throw new LifeSciencesValidationError(path, message);
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
    if (!allowedSet.has(key)) fail(`${path}.${key}`, "is not part of schema v0");
  }
  for (const key of allowed) {
    if (!Object.hasOwn(value, key)) fail(`${path}.${key}`, "is required");
  }
}

function boundedString(value, path, maxBytes = 512) {
  if (typeof value !== "string" || value.length === 0) {
    fail(path, "must be a nonempty string");
  }
  if (Buffer.byteLength(value, "utf8") > maxBytes) {
    fail(path, `must be at most ${maxBytes} UTF-8 bytes`);
  }
  if (value.includes("\u0000")) fail(path, "must not contain NUL");
  return value;
}

function boolean(value, path) {
  if (typeof value !== "boolean") fail(path, "must be a boolean");
  return value;
}

function integer(value, path, minimum, maximum) {
  if (!Number.isSafeInteger(value) || value < minimum || value > maximum) {
    fail(path, `must be an integer from ${minimum} through ${maximum}`);
  }
  return value;
}

function falseOnly(value, path) {
  if (value !== false) fail(path, "must be false in DRAFT SHADOW_ONLY v0");
}

function trueOnly(value, path) {
  if (value !== true) fail(path, "must be true in the v0 safety boundary");
}

function stringArray(
  value,
  path,
  { minItems = 0, maxItems = 16, pattern, allowed, sorted = true } = {},
) {
  if (
    !Array.isArray(value) ||
    value.length < minItems ||
    value.length > maxItems
  ) {
    fail(path, `must contain ${minItems} through ${maxItems} items`);
  }
  const result = value.map((item, index) => {
    const parsed = boundedString(item, `${path}[${index}]`);
    if (pattern && !pattern.test(parsed)) {
      fail(`${path}[${index}]`, "has an invalid format");
    }
    if (allowed && !allowed.has(parsed)) {
      fail(`${path}[${index}]`, "has an unknown value");
    }
    return parsed;
  });
  if (new Set(result).size !== result.length) fail(path, "must not contain duplicates");
  if (sorted && result.some((item, index) => index > 0 && result[index - 1] > item)) {
    fail(path, "must be sorted lexicographically");
  }
  return result;
}

function exactArray(actual, expected, path) {
  if (
    !Array.isArray(actual) ||
    actual.length !== expected.length ||
    actual.some((item, index) => item !== expected[index])
  ) {
    fail(path, "must preserve the exact v0 safety policy");
  }
}

function exactJson(actual, expected, path) {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    fail(path, "must preserve the exact v0 policy");
  }
}

function validateHttpsLocator(value, expected, path) {
  const text = boundedString(value, path, 1024);
  let url;
  try {
    url = new URL(text);
  } catch {
    fail(path, "must be an absolute URL");
  }
  if (
    url.protocol !== "https:" ||
    url.username !== "" ||
    url.password !== "" ||
    url.search !== "" ||
    url.hash !== "" ||
    text !== expected
  ) {
    fail(path, "must be the exact allowlisted authoritative HTTPS locator");
  }
  return text;
}

function validateReleaseBoundary(value) {
  const boundary = record(value, "$.releaseBoundary");
  exactKeys(boundary, RELEASE_BOUNDARY_KEYS, "$.releaseBoundary");
  for (const key of RELEASE_BOUNDARY_KEYS) {
    falseOnly(boundary[key], `$.releaseBoundary.${key}`);
  }
}

function validateEconomics(value) {
  const economics = record(value, "$.economics");
  exactKeys(economics, ECONOMICS_KEYS, "$.economics");
  if (economics.effect !== "NONE") fail("$.economics.effect", "must be NONE");
  if (economics.amount !== "0") fail("$.economics.amount", "must be the string 0");
  if (economics.denom !== null) fail("$.economics.denom", "must be null");
  falseOnly(economics.rewardMultiplier, "$.economics.rewardMultiplier");
  if (economics.escrowReference !== null) {
    fail("$.economics.escrowReference", "must be null");
  }
}

function validateScope(value) {
  const scope = record(value, "$.scope");
  exactKeys(scope, SCOPE_KEYS, "$.scope");
  if (scope.riskClass !== "GREEN_ONLY") {
    fail("$.scope.riskClass", "must be GREEN_ONLY");
  }
  exactArray(scope.allowedWork, REQUIRED_ALLOWED_WORK, "$.scope.allowedWork");
  exactArray(scope.refusedTopics, REQUIRED_REFUSED_TOPICS, "$.scope.refusedTopics");
  if (scope.unknownRiskDisposition !== "REFUSE_AND_ESCALATE_PRIVATELY") {
    fail(
      "$.scope.unknownRiskDisposition",
      "must refuse unknown risk and escalate only privately",
    );
  }
  if (scope.privateEscalationContent !== "NON_OPERATIONAL_METADATA_ONLY") {
    fail(
      "$.scope.privateEscalationContent",
      "must be NON_OPERATIONAL_METADATA_ONLY",
    );
  }
}

function validateFixtureBoundary(value) {
  const boundary = record(value, "$.fixtureBoundary");
  exactKeys(boundary, FIXTURE_BOUNDARY_KEYS, "$.fixtureBoundary");
  trueOnly(boundary.syntheticIdentifiersOnly, "$.fixtureBoundary.syntheticIdentifiersOnly");
  trueOnly(boundary.invalidTldRequired, "$.fixtureBoundary.invalidTldRequired");
  falseOnly(boundary.sequencePayloadsAllowed, "$.fixtureBoundary.sequencePayloadsAllowed");
  falseOnly(
    boundary.operationalProtocolsAllowed,
    "$.fixtureBoundary.operationalProtocolsAllowed",
  );
  falseOnly(
    boundary.identifiableHumanDataAllowed,
    "$.fixtureBoundary.identifiableHumanDataAllowed",
  );
  falseOnly(boundary.rawHumanGenomeAllowed, "$.fixtureBoundary.rawHumanGenomeAllowed");
  if (boundary.maximumArtifactMetadataBytes !== 8192) {
    fail(
      "$.fixtureBoundary.maximumArtifactMetadataBytes",
      "must remain exactly 8192 bytes",
    );
  }
}

function validateEvidencePolicy(value) {
  const policy = record(value, "$.evidencePolicy");
  exactKeys(policy, EVIDENCE_POLICY_KEYS, "$.evidencePolicy");
  if (policy.equivalenceToCoreTree !== "NO_EQUIVALENCE_TO_E0_E6") {
    fail(
      "$.evidencePolicy.equivalenceToCoreTree",
      "must declare NO_EQUIVALENCE_TO_E0_E6",
    );
  }
  if (policy.equivalenceToPoca !== "NO_EQUIVALENCE_TO_POCA_TIERS") {
    fail(
      "$.evidencePolicy.equivalenceToPoca",
      "must declare NO_EQUIVALENCE_TO_POCA_TIERS",
    );
  }
  if (!Array.isArray(policy.ladder) || policy.ladder.length !== 6) {
    fail("$.evidencePolicy.ladder", "must contain LS0 through LS5 exactly");
  }
  policy.ladder.forEach((item, index) => {
    const level = record(item, `$.evidencePolicy.ladder[${index}]`);
    exactKeys(level, EVIDENCE_LEVEL_KEYS, `$.evidencePolicy.ladder[${index}]`);
  });
  exactJson(policy.ladder, EXPECTED_EVIDENCE_LADDER, "$.evidencePolicy.ladder");
}

function validateIndependence(value) {
  const independence = record(value, "$.independence");
  exactKeys(independence, INDEPENDENCE_KEYS, "$.independence");
  const exactIntegers = {
    minimumEffectiveControlClusters: 3,
    minimumOrganizationRoots: 2,
    minimumMethodRoots: 2,
    minimumContextRoots: 2,
  };
  for (const [key, expected] of Object.entries(exactIntegers)) {
    integer(independence[key], `$.independence.${key}`, expected, expected);
  }
  falseOnly(
    independence.rawIdentityCountIsEvidence,
    "$.independence.rawIdentityCountIsEvidence",
  );
  trueOnly(
    independence.beneficialControlDisclosureRequired,
    "$.independence.beneficialControlDisclosureRequired",
  );
  trueOnly(
    independence.futureEligibilityRequiresExternalControllerAttestation,
    "$.independence.futureEligibilityRequiresExternalControllerAttestation",
  );
}

function validateChallengePolicy(value) {
  const policy = record(value, "$.challengePolicy");
  exactKeys(policy, CHALLENGE_POLICY_KEYS, "$.challengePolicy");
  trueOnly(policy.openChallengeRequired, "$.challengePolicy.openChallengeRequired");
  trueOnly(
    policy.unresolvedChallengeBlocksCrown,
    "$.challengePolicy.unresolvedChallengeBlocksCrown",
  );
  trueOnly(
    policy.upheldChallengeBlocksCrown,
    "$.challengePolicy.upheldChallengeBlocksCrown",
  );
  falseOnly(
    policy.challengeCanCreateReward,
    "$.challengePolicy.challengeCanCreateReward",
  );
  trueOnly(
    policy.futureEligibilityRequiresAdjudicationReceipt,
    "$.challengePolicy.futureEligibilityRequiresAdjudicationReceipt",
  );
}

function validateAttestationBoundary(value) {
  const boundary = record(value, "$.attestationBoundary");
  exactKeys(boundary, ATTESTATION_BOUNDARY_KEYS, "$.attestationBoundary");
  if (boundary.controlDisclosures !== "SELF_DECLARED_SYNTHETIC_LABELS") {
    fail(
      "$.attestationBoundary.controlDisclosures",
      "must remain SELF_DECLARED_SYNTHETIC_LABELS",
    );
  }
  if (boundary.challengeStatus !== "SELF_DECLARED_SYNTHETIC_LABEL") {
    fail(
      "$.attestationBoundary.challengeStatus",
      "must remain SELF_DECLARED_SYNTHETIC_LABEL",
    );
  }
  falseOnly(
    boundary.establishesControllerIndependence,
    "$.attestationBoundary.establishesControllerIndependence",
  );
  falseOnly(
    boundary.establishesChallengeClosure,
    "$.attestationBoundary.establishesChallengeClosure",
  );
}

function validateWalls(value) {
  if (!Array.isArray(value) || value.length !== EXPECTED_WALLS.length) {
    fail("$.nonImplicationWalls", "must preserve all four v0 scientific walls");
  }
  value.forEach((item, index) => {
    const wall = record(item, `$.nonImplicationWalls[${index}]`);
    exactKeys(wall, WALL_KEYS, `$.nonImplicationWalls[${index}]`);
  });
  exactJson(value, EXPECTED_WALLS, "$.nonImplicationWalls");
}

function validateReferences(value) {
  const expectedIds = Object.keys(REFERENCE_LOCATORS).sort();
  if (!Array.isArray(value) || value.length !== expectedIds.length) {
    fail("$.references", "must contain every exact v0 authority locator");
  }
  const ids = [];
  value.forEach((item, index) => {
    const path = `$.references[${index}]`;
    const reference = record(item, path);
    exactKeys(reference, REFERENCE_KEYS, path);
    const id = boundedString(reference.canonicalId, `${path}.canonicalId`);
    ids.push(id);
    const locator = REFERENCE_LOCATORS[id];
    if (!locator) fail(`${path}.canonicalId`, "has no v0 authoritative source locator");
    for (const key of ["authority", "title"]) {
      if (reference[key] !== locator[key]) {
        fail(`${path}.${key}`, "does not match the reviewed authority locator metadata");
      }
    }
    validateHttpsLocator(reference.specification, locator.specification, `${path}.specification`);
  });
  exactArray(ids, expectedIds, "$.references");
}

function validateNode(value, index) {
  const path = `$.nodes[${index}]`;
  const node = record(value, path);
  exactKeys(node, NODE_KEYS, path);
  const id = boundedString(node.id, `${path}.id`, 96);
  if (!NODE_ID_PATTERN.test(id)) fail(`${path}.id`, "must be lowercase kebab-case ending @1");
  boundedString(node.title, `${path}.title`, 128);
  if (!STAGES.has(node.stage)) fail(`${path}.stage`, "has an unknown stage");
  if (!BRANCHES.has(node.branch)) fail(`${path}.branch`, "has an unknown branch");
  boundedString(node.summary, `${path}.summary`, 512);
  const prerequisites = stringArray(node.prerequisites, `${path}.prerequisites`, {
    maxItems: LIFE_SCIENCES_MAX_PREREQUISITES,
    pattern: NODE_ID_PATTERN,
  });
  if (!EVIDENCE_LEVELS.has(node.evidenceFloor)) {
    fail(`${path}.evidenceFloor`, "must be LS0 through LS5");
  }
  if (!CLAIM_CLASSES.has(node.claimClass)) {
    fail(`${path}.claimClass`, "has an unknown claim class");
  }
  const permitted = stringArray(node.permittedConclusions, `${path}.permittedConclusions`, {
    minItems: 1,
    maxItems: 5,
    allowed: CONCLUSIONS,
  });
  const forbidden = stringArray(node.forbiddenConclusions, `${path}.forbiddenConclusions`, {
    maxItems: 8,
    allowed: CONCLUSIONS,
  });
  for (const conclusion of permitted) {
    if (forbidden.includes(conclusion)) {
      fail(`${path}.permittedConclusions`, "must not overlap forbidden conclusions");
    }
  }
  const requiredForbidden = WALL_FORBIDDEN_BY_CLAIM_CLASS[node.claimClass] ?? [];
  for (const conclusion of requiredForbidden) {
    if (!forbidden.includes(conclusion)) {
      fail(
        `${path}.forbiddenConclusions`,
        `must preserve scientific wall for ${conclusion}`,
      );
    }
    if (permitted.includes(conclusion)) {
      fail(`${path}.permittedConclusions`, `must not cross wall into ${conclusion}`);
    }
  }
  stringArray(node.artifactRequirements, `${path}.artifactRequirements`, {
    minItems: 2,
    maxItems: 6,
    sorted: false,
  });
  stringArray(node.revalidationTriggers, `${path}.revalidationTriggers`, {
    minItems: 1,
    maxItems: 5,
    sorted: false,
  });
  stringArray(node.referenceIds, `${path}.referenceIds`, {
    minItems: 1,
    maxItems: 6,
  });
  boolean(node.crown, `${path}.crown`);
  if (node.crown && (node.stage !== "crown" || node.evidenceFloor !== "LS5")) {
    fail(path, "a crown node must be stage crown with an LS5 floor");
  }
  if (!node.crown && node.stage === "crown") {
    fail(`${path}.crown`, "stage crown must set crown true");
  }
  if (DNA_LIKE_PAYLOAD_PATTERN.test(JSON.stringify(node))) {
    fail(path, "must not embed a nucleotide-like sequence payload");
  }
  return { node, id, prerequisites };
}

function validateGraph(nodes, roots) {
  const byId = new Map(nodes.map((entry) => [entry.id, entry]));
  const rootSet = new Set(roots);
  const crownNodes = nodes.filter(({ node }) => node.crown);
  if (crownNodes.length !== 1) fail("$.nodes", "must contain exactly one crown node");

  for (const root of roots) {
    const entry = byId.get(root);
    if (!entry) fail("$.roots", `references missing node ${root}`);
    if (entry.prerequisites.length !== 0 || entry.node.stage !== "foundation") {
      fail("$.roots", "roots must be prerequisite-free foundation nodes");
    }
  }
  for (const entry of nodes) {
    if ((entry.prerequisites.length === 0) !== rootSet.has(entry.id)) {
      fail("$.roots", "must name every and only prerequisite-free node");
    }
    for (const prerequisite of entry.prerequisites) {
      const parent = byId.get(prerequisite);
      if (!parent) fail("$.nodes", `${entry.id} has missing prerequisite ${prerequisite}`);
      if (STAGE_RANK.get(parent.node.stage) > STAGE_RANK.get(entry.node.stage)) {
        fail("$.nodes", `${entry.id} depends on a later-stage node`);
      }
    }
  }

  const visiting = new Set();
  const visited = new Set();
  const depthMemo = new Map();
  const visit = (id) => {
    if (visiting.has(id)) fail("$.nodes", "prerequisites must form a DAG");
    if (visited.has(id)) return depthMemo.get(id);
    visiting.add(id);
    const entry = byId.get(id);
    const depth =
      entry.prerequisites.length === 0
        ? 1
        : 1 + Math.max(...entry.prerequisites.map(visit));
    visiting.delete(id);
    visited.add(id);
    depthMemo.set(id, depth);
    return depth;
  };
  const maxDepth = Math.max(...nodes.map(({ id }) => visit(id)));
  if (maxDepth > LIFE_SCIENCES_MAX_DEPTH) {
    fail("$.nodes", `graph depth exceeds ${LIFE_SCIENCES_MAX_DEPTH}`);
  }

  const fanOut = new Map(nodes.map(({ id }) => [id, 0]));
  let edgeCount = 0;
  for (const { prerequisites } of nodes) {
    for (const prerequisite of prerequisites) {
      edgeCount += 1;
      fanOut.set(prerequisite, fanOut.get(prerequisite) + 1);
    }
  }
  if ([...fanOut.values()].some((count) => count > LIFE_SCIENCES_MAX_FAN_OUT)) {
    fail("$.nodes", `graph fan-out exceeds ${LIFE_SCIENCES_MAX_FAN_OUT}`);
  }

  const reachable = new Set();
  const children = new Map(nodes.map(({ id }) => [id, []]));
  for (const entry of nodes) {
    for (const prerequisite of entry.prerequisites) {
      children.get(prerequisite).push(entry.id);
    }
  }
  const stack = [...roots];
  while (stack.length > 0) {
    const id = stack.pop();
    if (reachable.has(id)) continue;
    reachable.add(id);
    stack.push(...children.get(id));
  }
  if (reachable.size !== nodes.length) fail("$.nodes", "every node must be reachable from a root");
  return { edgeCount, maxDepth, crownId: crownNodes[0].id };
}

function validateBaseTreeBinding(value) {
  const binding = record(value, "$.baseTreeBinding");
  exactKeys(binding, BASE_TREE_BINDING_KEYS, "$.baseTreeBinding");
  if (binding.schema !== "zerone.constructive-intelligence-tree/v1") {
    fail("$.baseTreeBinding.schema", "must bind the core v1 tree schema");
  }
  if (binding.sha256 !== BASE_TREE_SHA256) {
    fail("$.baseTreeBinding.sha256", "does not match the pinned core tree bytes");
  }
  if (binding.policyVersion !== "1.0.0") {
    fail("$.baseTreeBinding.policyVersion", "must bind policy version 1.0.0");
  }
  if (existsSync(BASE_TREE_PATH)) {
    const actual = createHash("sha256").update(readFileSync(BASE_TREE_PATH)).digest("hex");
    if (actual !== binding.sha256) {
      fail(
        "$.baseTreeBinding.sha256",
        `checked-in core tree drifted: expected ${binding.sha256}, got ${actual}`,
      );
    }
  }
}

export function validateMoneyKarmaConstitutionBinding(
  value,
  constitutionRaw = readFileSync(MONEY_KARMA_CONSTITUTION_PATH),
) {
  const binding = record(value, "$.constitutionBinding");
  exactKeys(binding, CONSTITUTION_BINDING_KEYS, "$.constitutionBinding");
  if (binding.schema !== MONEY_KARMA_CONSTITUTION_SCHEMA) {
    fail(
      "$.constitutionBinding.schema",
      `must remain ${MONEY_KARMA_CONSTITUTION_SCHEMA}`,
    );
  }
  const expected = `sha256:${MONEY_KARMA_CONSTITUTION_SHA256}`;
  if (binding.documentSha256 !== expected) {
    fail(
      "$.constitutionBinding.documentSha256",
      `must remain ${expected}`,
    );
  }
  const actual = createHash("sha256").update(constitutionRaw).digest("hex");
  if (actual !== MONEY_KARMA_CONSTITUTION_SHA256) {
    fail(
      "$.constitutionBinding.documentSha256",
      `checked-in Money-KARMA constitution drifted: expected ${MONEY_KARMA_CONSTITUTION_SHA256}, got ${actual}`,
    );
  }
}

export function validateConstructiveIntelligenceLifeSciences(value) {
  const profile = record(value, "$");
  exactKeys(profile, TOP_LEVEL_KEYS, "$");
  if (profile.schema !== LIFE_SCIENCES_SCHEMA) fail("$.schema", "has an unknown schema");
  if (profile.status !== "DRAFT") fail("$.status", "must be DRAFT");
  if (profile.mode !== "SHADOW_ONLY") fail("$.mode", "must be SHADOW_ONLY");
  falseOnly(profile.authoritative, "$.authoritative");
  falseOnly(profile.networkObserved, "$.networkObserved");
  falseOnly(profile.rewardBearing, "$.rewardBearing");
  if (!/^\d{4}-\d{2}-\d{2}$/.test(boundedString(profile.snapshotDate, "$.snapshotDate"))) {
    fail("$.snapshotDate", "must be YYYY-MM-DD");
  }
  validateBaseTreeBinding(profile.baseTreeBinding);
  validateMoneyKarmaConstitutionBinding(profile.constitutionBinding);
  validateReleaseBoundary(profile.releaseBoundary);
  validateEconomics(profile.economics);
  validateScope(profile.scope);
  validateFixtureBoundary(profile.fixtureBoundary);
  validateEvidencePolicy(profile.evidencePolicy);
  validateIndependence(profile.independence);
  validateChallengePolicy(profile.challengePolicy);
  validateAttestationBoundary(profile.attestationBoundary);
  validateWalls(profile.nonImplicationWalls);
  validateReferences(profile.references);

  const roots = stringArray(profile.roots, "$.roots", {
    minItems: 1,
    maxItems: 4,
    pattern: NODE_ID_PATTERN,
  });
  if (
    !Array.isArray(profile.nodes) ||
    profile.nodes.length < LIFE_SCIENCES_MIN_NODES ||
    profile.nodes.length > LIFE_SCIENCES_MAX_NODES
  ) {
    fail(
      "$.nodes",
      `must contain ${LIFE_SCIENCES_MIN_NODES} through ${LIFE_SCIENCES_MAX_NODES} nodes`,
    );
  }
  const nodes = profile.nodes.map(validateNode);
  const ids = nodes.map(({ id }) => id);
  if (new Set(ids).size !== ids.length) fail("$.nodes", "node IDs must be unique");
  if (ids.some((id, index) => index > 0 && ids[index - 1] > id)) {
    fail("$.nodes", "nodes must be sorted by ID");
  }
  const referenceIds = new Set(Object.keys(REFERENCE_LOCATORS));
  for (let index = 0; index < nodes.length; index += 1) {
    for (const referenceId of nodes[index].node.referenceIds) {
      if (!referenceIds.has(referenceId)) {
        fail(`$.nodes[${index}].referenceIds`, `unknown reference ${referenceId}`);
      }
    }
  }
  const graph = validateGraph(nodes, roots);
  return {
    schema: profile.schema,
    status: profile.status,
    mode: profile.mode,
    nodeCount: nodes.length,
    edgeCount: graph.edgeCount,
    maxDepth: graph.maxDepth,
    crownId: graph.crownId,
    economicEffect: "NONE",
    amount: "0",
  };
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
    while (offset < raw.length && !/[\s,\]}]/.test(raw[offset])) offset += 1;
  };

  scanValue("$");
}

function rejectExcessiveJsonNesting(raw) {
  let depth = 0;
  let inString = false;
  let escaped = false;
  for (const character of raw) {
    if (inString) {
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') inString = false;
      continue;
    }
    if (character === '"') inString = true;
    else if (character === "{" || character === "[") {
      depth += 1;
      if (depth > 32) fail("$", "JSON nesting exceeds 32");
    } else if (character === "}" || character === "]") depth -= 1;
  }
}

export function parseAndValidateConstructiveIntelligenceLifeSciences(raw) {
  if (typeof raw !== "string") fail("$", "raw document must be a string");
  if (Buffer.byteLength(raw, "utf8") > LIFE_SCIENCES_MAX_BYTES) {
    fail("$", `raw document exceeds ${LIFE_SCIENCES_MAX_BYTES} UTF-8 bytes`);
  }
  rejectExcessiveJsonNesting(raw);
  let parsed;
  try {
    parsed = JSON.parse(raw);
  } catch (error) {
    fail("$", `invalid JSON: ${error.message}`);
  }
  rejectDuplicateJsonKeys(raw);
  return validateConstructiveIntelligenceLifeSciences(parsed);
}

function validateFixtureArtifact(value, index) {
  const path = `$.artifacts[${index}]`;
  const artifact = record(value, path);
  exactKeys(artifact, ARTIFACT_KEYS, path);
  const id = boundedString(artifact.id, `${path}.id`, 96);
  if (!SIMPLE_ID_PATTERN.test(id)) fail(`${path}.id`, "must be a synthetic kebab-case ID");
  const uri = boundedString(artifact.uri, `${path}.uri`, 512);
  let parsed;
  try {
    parsed = new URL(uri);
  } catch {
    fail(`${path}.uri`, "must be an absolute URL");
  }
  if (
    parsed.protocol !== "https:" ||
    !parsed.hostname.endsWith(".invalid") ||
    parsed.username !== "" ||
    parsed.password !== "" ||
    parsed.search !== "" ||
    parsed.hash !== ""
  ) {
    fail(`${path}.uri`, "must be a credential-free synthetic HTTPS .invalid URI");
  }
  if (!SHA256_PATTERN.test(artifact.sha256)) fail(`${path}.sha256`, "must be SHA-256 hex");
  if (!OBSERVATION_KINDS.has(artifact.observationKind)) {
    fail(`${path}.observationKind`, "has an unknown observation kind");
  }
  boolean(artifact.containsSequence, `${path}.containsSequence`);
  boolean(artifact.containsOperationalProtocol, `${path}.containsOperationalProtocol`);
  if (!HUMAN_DATA_CLASSES.has(artifact.humanDataClass)) {
    fail(`${path}.humanDataClass`, "has an unknown human-data class");
  }
  const riskTopics = stringArray(artifact.riskTopics, `${path}.riskTopics`, {
    maxItems: 10,
    pattern: TOKEN_PATTERN,
  });
  if (DNA_LIKE_PAYLOAD_PATTERN.test(uri) || DNA_LIKE_PAYLOAD_PATTERN.test(id)) {
    fail(path, "must not embed a nucleotide-like sequence payload");
  }
  return { ...artifact, riskTopics };
}

function validateFixtureControl(value, index) {
  const path = `$.controls[${index}]`;
  const control = record(value, path);
  exactKeys(control, CONTROL_KEYS, path);
  for (const key of CONTROL_KEYS) {
    const id = boundedString(control[key], `${path}.${key}`, 96);
    if (!SIMPLE_ID_PATTERN.test(id)) fail(`${path}.${key}`, "must be a synthetic kebab-case ID");
  }
  return control;
}

function validateEvidenceFixture(value) {
  const fixture = record(value, "$fixture");
  exactKeys(fixture, FIXTURE_KEYS, "$fixture");
  if (fixture.schema !== LIFE_SCIENCES_EVIDENCE_FIXTURE_SCHEMA) {
    fail("$fixture.schema", "has an unknown fixture schema");
  }
  const id = boundedString(fixture.id, "$fixture.id", 96);
  if (!SIMPLE_ID_PATTERN.test(id)) fail("$fixture.id", "must be a synthetic kebab-case ID");
  if (fixture.profileSchema !== LIFE_SCIENCES_SCHEMA) {
    fail("$fixture.profileSchema", "must bind the life-sciences v0 profile");
  }
  const nodeId = boundedString(fixture.nodeId, "$fixture.nodeId", 96);
  if (!NODE_ID_PATTERN.test(nodeId)) fail("$fixture.nodeId", "has an invalid node ID");
  const requestedConclusions = stringArray(
    fixture.requestedConclusions,
    "$fixture.requestedConclusions",
    { minItems: 1, maxItems: 5, allowed: CONCLUSIONS },
  );
  if (!EVIDENCE_LEVELS.has(fixture.evidenceLevel)) {
    fail("$fixture.evidenceLevel", "must be LS0 through LS5");
  }
  if (!Array.isArray(fixture.artifacts) || fixture.artifacts.length < 1 || fixture.artifacts.length > 16) {
    fail("$fixture.artifacts", "must contain 1 through 16 metadata-only artifacts");
  }
  const artifacts = fixture.artifacts.map(validateFixtureArtifact);
  if (new Set(artifacts.map(({ id: artifactId }) => artifactId)).size !== artifacts.length) {
    fail("$fixture.artifacts", "artifact IDs must be unique");
  }
  if (!Array.isArray(fixture.controls) || fixture.controls.length > 16) {
    fail("$fixture.controls", "must contain at most 16 control disclosures");
  }
  const controls = fixture.controls.map(validateFixtureControl);
  if (!CHALLENGE_STATUSES.has(fixture.challengeStatus)) {
    fail("$fixture.challengeStatus", "has an unknown challenge status");
  }
  return { ...fixture, requestedConclusions, artifacts, controls };
}

function uniqueCount(values) {
  return new Set(values).size;
}

export function evaluateLifeSciencesEvidenceFixture(profileValue, fixtureValue) {
  validateConstructiveIntelligenceLifeSciences(profileValue);
  const fixture = validateEvidenceFixture(fixtureValue);
  const node = profileValue.nodes.find(({ id }) => id === fixture.nodeId);
  if (!node) fail("$fixture.nodeId", "does not exist in the bound profile");

  const reasons = new Set();
  let refused = false;
  const observations = new Set(fixture.artifacts.map(({ observationKind }) => observationKind));
  for (const artifact of fixture.artifacts) {
    if (artifact.containsSequence) {
      refused = true;
      reasons.add("SEQUENCE_PAYLOAD_REFUSED");
    }
    if (artifact.containsOperationalProtocol) {
      refused = true;
      reasons.add("OPERATIONAL_PROTOCOL_REFUSED");
    }
    if (artifact.humanDataClass === "IDENTIFIABLE") {
      refused = true;
      reasons.add("IDENTIFIABLE_HUMAN_DATA_REFUSED");
    }
    if (artifact.humanDataClass === "RAW_HUMAN_GENOME") {
      refused = true;
      reasons.add("RAW_HUMAN_GENOME_REFUSED");
    }
    for (const riskTopic of artifact.riskTopics) {
      refused = true;
      reasons.add(
        REQUIRED_REFUSED_TOPICS.includes(riskTopic)
          ? `RED_RISK_REFUSED:${riskTopic}`
          : `UNKNOWN_RISK_PRIVATE_ESCALATION_ONLY:${riskTopic}`,
      );
    }
  }

  const requested = new Set(fixture.requestedConclusions);
  if (
    observations.has("STATIC_STRUCTURE") &&
    !observations.has("FOLDING_DYNAMICS") &&
    (["FOLDING_KINETICS", "FOLDING_PATHWAY"].some((claim) => requested.has(claim)))
  ) {
    reasons.add("WALL_STATIC_COORDINATES_NOT_FOLDING_PATHWAY_OR_KINETICS");
  }
  if (
    observations.has("TRANSCRIPT_ABUNDANCE") &&
    !observations.has("PROTEIN_ABUNDANCE") &&
    !observations.has("PROTEIN_LOCALIZATION") &&
    !observations.has("FUNCTIONAL_EVIDENCE") &&
    (["PROTEIN_ABUNDANCE", "PROTEIN_FUNCTION", "PROTEIN_LOCALIZATION"].some(
      (claim) => requested.has(claim),
    ))
  ) {
    reasons.add("WALL_MRNA_NOT_PROTEIN_ABUNDANCE_LOCALIZATION_OR_FUNCTION");
  }
  if (
    observations.has("REGULATORY_ASSOCIATION") &&
    !observations.has("CAUSAL_PERTURBATION") &&
    requested.has("CAUSAL_REGULATION")
  ) {
    reasons.add("WALL_ASSOCIATION_NOT_REGULATION_OR_CAUSALITY");
  }
  if (
    (observations.has("FOLD_MODEL") || observations.has("FOLD_ENSEMBLE")) &&
    !observations.has("PROSPECTIVE_EVALUATION") &&
    requested.has("PROSPECTIVE_VALIDATION")
  ) {
    reasons.add("WALL_INSILICO_NOT_PROSPECTIVE_EXPERIMENT");
  }

  for (const conclusion of requested) {
    if (!node.permittedConclusions.includes(conclusion)) {
      reasons.add(`NODE_DOES_NOT_PERMIT:${conclusion}`);
    }
    for (const observation of CONCLUSION_OBSERVATIONS[conclusion] ?? []) {
      if (!observations.has(observation)) reasons.add(`MISSING_OBSERVATION:${observation}`);
    }
  }
  if (EVIDENCE_RANK.get(fixture.evidenceLevel) < EVIDENCE_RANK.get(node.evidenceFloor)) {
    reasons.add(`EVIDENCE_BELOW_NODE_FLOOR:${node.evidenceFloor}`);
  }

  if (EVIDENCE_RANK.get(fixture.evidenceLevel) >= EVIDENCE_RANK.get("LS3")) {
    if (
      uniqueCount(fixture.controls.map(({ effectiveControlCluster }) => effectiveControlCluster)) <
      profileValue.independence.minimumEffectiveControlClusters
    ) {
      reasons.add("INSUFFICIENT_EFFECTIVE_CONTROL_CLUSTERS");
    }
    if (
      uniqueCount(fixture.controls.map(({ organizationRoot }) => organizationRoot)) <
      profileValue.independence.minimumOrganizationRoots
    ) {
      reasons.add("INSUFFICIENT_ORGANIZATION_ROOTS");
    }
    if (
      uniqueCount(fixture.controls.map(({ methodRoot }) => methodRoot)) <
      profileValue.independence.minimumMethodRoots
    ) {
      reasons.add("INSUFFICIENT_METHOD_ROOTS");
    }
    if (
      uniqueCount(fixture.controls.map(({ contextRoot }) => contextRoot)) <
      profileValue.independence.minimumContextRoots
    ) {
      reasons.add("INSUFFICIENT_CONTEXT_ROOTS");
    }
  }

  const contributorControlTuples = new Map();
  for (const control of fixture.controls) {
    const tuples = contributorControlTuples.get(control.contributorId) ?? new Set();
    tuples.add(
      JSON.stringify([
        control.effectiveControlCluster,
        control.organizationRoot,
        control.methodRoot,
        control.contextRoot,
      ]),
    );
    contributorControlTuples.set(control.contributorId, tuples);
  }
  for (const [contributorId, tuples] of contributorControlTuples) {
    if (tuples.size > 1) {
      reasons.add(`CONTRIBUTOR_CLAIMS_DIVERGENT_CONTROL_TUPLES:${contributorId}`);
    }
  }

  if (node.crown) {
    if (fixture.evidenceLevel !== "LS5") reasons.add("CROWN_REQUIRES_LS5");
    if (fixture.challengeStatus !== "CLEAR") {
      reasons.add(`CHALLENGE_BLOCKS_CROWN:${fixture.challengeStatus}`);
    }
    if (!requested.has("INDEPENDENT_CROSS_CONTEXT_REPLICATION")) {
      reasons.add("CROWN_REQUIRES_CROSS_CONTEXT_CLAIM");
    }
  } else if (requested.has("INDEPENDENT_CROSS_CONTEXT_REPLICATION")) {
    reasons.add("CROSS_CONTEXT_CROWN_CLAIM_REQUIRES_CROWN_NODE");
  }

  return {
    outcome: refused
      ? "SHADOW_ONLY_REFUSED"
      : reasons.size === 0
        ? "SHADOW_ONLY_STRUCTURAL_MATCH"
        : "SHADOW_ONLY_BLOCKED",
    rewardEligible: false,
    independenceStatus: "DECLARED_UNVERIFIED",
    challengeStatus: "DECLARED_UNVERIFIED",
    economicEffect: "NONE",
    amount: "0",
    reasons: [...reasons].sort(),
  };
}

function runCli() {
  if (process.argv.length !== 3) {
    console.error(
      "usage: node scripts/validate-constructive-intelligence-life-sciences.mjs PATH",
    );
    process.exitCode = 2;
    return;
  }
  let raw;
  try {
    raw = readFileSync(resolve(process.argv[2]), "utf8");
  } catch (error) {
    console.error(`constructive-intelligence life sciences: FAIL: ${error.message}`);
    process.exitCode = 1;
    return;
  }
  try {
    const result = parseAndValidateConstructiveIntelligenceLifeSciences(raw);
    console.log(
      `constructive-intelligence life sciences: PASS (${result.nodeCount} nodes, ${result.edgeCount} prerequisite edges, depth ${result.maxDepth}, ${result.status}/${result.mode}, economics ${result.economicEffect} ${result.amount})`,
    );
  } catch (error) {
    console.error(`constructive-intelligence life sciences: FAIL: ${error.message}`);
    process.exitCode = 1;
  }
}

if (
  process.argv[1] &&
  import.meta.url === pathToFileURL(resolve(process.argv[1])).href
) {
  runCli();
}
