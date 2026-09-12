import {
  COSMOS_AMOUNT_MAX,
  COSMOS_UINT64_MAX,
  LIQUIDITY_FEE_SCALE,
  LIQUIDITY_LEGACY_PROTOCOL_FEE_DESTINATION_MODULE,
  LIQUIDITY_POOL_STATUS,
  LiquidityClientError,
  MSG_CREATE_POOL_TYPE_URL,
  MSG_SUBMIT_PROPOSAL_TYPE_URL,
  MSG_SWAP_TYPE_URL,
  MSG_UPDATE_LIQUIDITY_PARAMS_TYPE_URL,
  ZERONE_MAX_POOL_RECORDS,
  ZERONE_MAX_SWAP_FEE,
  ZeroneLiquidityRestClient,
  createExactInSwapPlan,
  createLiquidityAdmissionProposal,
  createLiquidityAdmissionUpdateMessage,
  createPoolMessage,
  discloseLiquiditySwapFee,
  minimumOutputForSlippage,
  parseCanonicalPositiveAmount,
  quoteConstantProductExactIn,
  timeoutHeightAfter,
  withTimeoutHeight
} from "./chunk-4HDWG77G.js";
import {
  CidError,
  asZeroneMemoryCid,
  parseCanonicalCidV1
} from "./chunk-QI25M5F7.js";
import {
  FeeGrantError,
  ZERONE_ONBOARDING_MESSAGE_TYPE_URLS,
  makeBoundedFeeGrant,
  makeRevokeFeeGrant,
  makeSponsoredFee
} from "./chunk-2SQQL3WA.js";
import {
  CaipError,
  asExistingZeroneDid,
  cosmosChainId,
  defineZeroneNetwork,
  formatCaip10,
  formatCaip2,
  parseCaip10,
  parseCaip2,
  parseCosmosChainId,
  zeroneAccountId
} from "./chunk-HSUZCRNJ.js";
import {
  ACCOUNT_REGISTRATION_PROOF_DOMAIN,
  KEY_ROTATION_ACCEPTANCE_DOMAIN,
  KEY_ROTATION_AUTHORIZATION_DOMAIN,
  KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS,
  accountRegistrationProofSignBytes,
  computeReviewCommitmentV2,
  keyRotationAcceptanceSignBytes,
  keyRotationAuthorizationSignBytes,
  makeReviewRevealV2
} from "./chunk-KUI5J3XY.js";
import {
  IN_TOTO_STATEMENT_V1_TYPE,
  ProvenanceParseError,
  ZERONE_PROVENANCE_LIMITS,
  ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE,
  parseUnsignedZeroneInTotoStatement
} from "./chunk-FKXK63RS.js";
import {
  createZeroneRegistry,
  registerZeroneMessages,
  zeroneRegistryTypes
} from "./chunk-CGU5T5DU.js";
import {
  Claim,
  Fact,
  FactRelation,
  StatusTransition,
  VerificationRound
} from "./chunk-HXIVKHSQ.js";
import {
  BinaryReader,
  BinaryWriter
} from "./chunk-CXBAXZI7.js";
import "./chunk-MLKGABMK.js";

// src/generated/zerone/knowledge/v1/claim_history.ts
function createBaseQueryClaimHistoryRequest() {
  return {
    id: "",
    atBlockHeight: BigInt(0)
  };
}
var QueryClaimHistoryRequest = {
  typeUrl: "/zerone.knowledge.v1.QueryClaimHistoryRequest",
  encode(message, writer = BinaryWriter.create()) {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.atBlockHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.atBlockHeight);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseQueryClaimHistoryRequest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.atBlockHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseQueryClaimHistoryRequest();
    message.id = object.id ?? "";
    message.atBlockHeight = object.atBlockHeight !== void 0 && object.atBlockHeight !== null ? BigInt(object.atBlockHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseQueryClaimHistoryResponse() {
  return {
    chainId: "",
    blockHeight: BigInt(0),
    record: void 0,
    relatedClaims: []
  };
}
var QueryClaimHistoryResponse = {
  typeUrl: "/zerone.knowledge.v1.QueryClaimHistoryResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.chainId !== "") {
      writer.uint32(10).string(message.chainId);
    }
    if (message.blockHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.blockHeight);
    }
    if (message.record !== void 0) {
      ClaimHistoryRecord.encode(message.record, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.relatedClaims) {
      RelatedClaimHistory.encode(v, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseQueryClaimHistoryResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.chainId = reader.string();
          break;
        case 2:
          message.blockHeight = reader.uint64();
          break;
        case 3:
          message.record = ClaimHistoryRecord.decode(reader, reader.uint32());
          break;
        case 4:
          message.relatedClaims.push(RelatedClaimHistory.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseQueryClaimHistoryResponse();
    message.chainId = object.chainId ?? "";
    message.blockHeight = object.blockHeight !== void 0 && object.blockHeight !== null ? BigInt(object.blockHeight.toString()) : BigInt(0);
    message.record = object.record !== void 0 && object.record !== null ? ClaimHistoryRecord.fromPartial(object.record) : void 0;
    message.relatedClaims = object.relatedClaims?.map((e) => RelatedClaimHistory.fromPartial(e)) || [];
    return message;
  }
};
function createBaseClaimHistoryRecord() {
  return {
    claimId: "",
    claim: void 0,
    rounds: [],
    facts: [],
    missingRoundIds: []
  };
}
var ClaimHistoryRecord = {
  typeUrl: "/zerone.knowledge.v1.ClaimHistoryRecord",
  encode(message, writer = BinaryWriter.create()) {
    if (message.claimId !== "") {
      writer.uint32(10).string(message.claimId);
    }
    if (message.claim !== void 0) {
      Claim.encode(message.claim, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.rounds) {
      VerificationRound.encode(v, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.facts) {
      ClaimHistoryFact.encode(v, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.missingRoundIds) {
      writer.uint32(42).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseClaimHistoryRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimId = reader.string();
          break;
        case 2:
          message.claim = Claim.decode(reader, reader.uint32());
          break;
        case 3:
          message.rounds.push(VerificationRound.decode(reader, reader.uint32()));
          break;
        case 4:
          message.facts.push(ClaimHistoryFact.decode(reader, reader.uint32()));
          break;
        case 5:
          message.missingRoundIds.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseClaimHistoryRecord();
    message.claimId = object.claimId ?? "";
    message.claim = object.claim !== void 0 && object.claim !== null ? Claim.fromPartial(object.claim) : void 0;
    message.rounds = object.rounds?.map((e) => VerificationRound.fromPartial(e)) || [];
    message.facts = object.facts?.map((e) => ClaimHistoryFact.fromPartial(e)) || [];
    message.missingRoundIds = object.missingRoundIds?.map((e) => e) || [];
    return message;
  }
};
function createBaseClaimHistoryFact() {
  return {
    fact: void 0,
    outgoingRelations: [],
    incomingRelations: [],
    statusTransitions: []
  };
}
var ClaimHistoryFact = {
  typeUrl: "/zerone.knowledge.v1.ClaimHistoryFact",
  encode(message, writer = BinaryWriter.create()) {
    if (message.fact !== void 0) {
      Fact.encode(message.fact, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.outgoingRelations) {
      FactRelation.encode(v, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.incomingRelations) {
      FactRelation.encode(v, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.statusTransitions) {
      StatusTransition.encode(v, writer.uint32(34).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseClaimHistoryFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.fact = Fact.decode(reader, reader.uint32());
          break;
        case 2:
          message.outgoingRelations.push(FactRelation.decode(reader, reader.uint32()));
          break;
        case 3:
          message.incomingRelations.push(FactRelation.decode(reader, reader.uint32()));
          break;
        case 4:
          message.statusTransitions.push(StatusTransition.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseClaimHistoryFact();
    message.fact = object.fact !== void 0 && object.fact !== null ? Fact.fromPartial(object.fact) : void 0;
    message.outgoingRelations = object.outgoingRelations?.map((e) => FactRelation.fromPartial(e)) || [];
    message.incomingRelations = object.incomingRelations?.map((e) => FactRelation.fromPartial(e)) || [];
    message.statusTransitions = object.statusTransitions?.map((e) => StatusTransition.fromPartial(e)) || [];
    return message;
  }
};
function createBaseRelatedClaimHistory() {
  return {
    record: void 0,
    links: []
  };
}
var RelatedClaimHistory = {
  typeUrl: "/zerone.knowledge.v1.RelatedClaimHistory",
  encode(message, writer = BinaryWriter.create()) {
    if (message.record !== void 0) {
      ClaimHistoryRecord.encode(message.record, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.links) {
      ClaimHistoryLink.encode(v, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseRelatedClaimHistory();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.record = ClaimHistoryRecord.decode(reader, reader.uint32());
          break;
        case 2:
          message.links.push(ClaimHistoryLink.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseRelatedClaimHistory();
    message.record = object.record !== void 0 && object.record !== null ? ClaimHistoryRecord.fromPartial(object.record) : void 0;
    message.links = object.links?.map((e) => ClaimHistoryLink.fromPartial(e)) || [];
    return message;
  }
};
function createBaseClaimHistoryLink() {
  return {
    field: "",
    targetId: ""
  };
}
var ClaimHistoryLink = {
  typeUrl: "/zerone.knowledge.v1.ClaimHistoryLink",
  encode(message, writer = BinaryWriter.create()) {
    if (message.field !== "") {
      writer.uint32(10).string(message.field);
    }
    if (message.targetId !== "") {
      writer.uint32(18).string(message.targetId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseClaimHistoryLink();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.field = reader.string();
          break;
        case 2:
          message.targetId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseClaimHistoryLink();
    message.field = object.field ?? "";
    message.targetId = object.targetId ?? "";
    return message;
  }
};

// src/claim-history.ts
var maximumBytes = 8 * 1024 * 1024;
var maximumHeight = (1n << 64n) - 1n;
var utf8 = new TextEncoder();
var strictUtf8 = new TextDecoder("utf-8", { fatal: true });
function requireValue(condition, message) {
  if (!condition) throw new Error(`ClaimHistory: ${message}`);
}
function identifier(value, field) {
  requireValue(typeof value === "string", `${field} must be a string`);
  const bytes = utf8.encode(value);
  requireValue(
    bytes.length > 0 && bytes.length <= 256 && strictUtf8.decode(bytes) === value,
    `${field} must be nonempty valid UTF-8 of at most 256 bytes`
  );
}
function validateRecord(record) {
  requireValue(record && record.claimId !== "", "missing record identity");
  if (record.claim) requireValue(record.claim.id === record.claimId, "embedded claim identity mismatch");
  const rounds = /* @__PURE__ */ new Set();
  for (const round of record.rounds) {
    requireValue(
      round.id !== "" && !rounds.has(round.id) && round.claimId === record.claimId,
      "duplicate or inconsistent round identity"
    );
    rounds.add(round.id);
  }
  const missing = /* @__PURE__ */ new Set();
  for (const id of record.missingRoundIds) {
    requireValue(id !== "" && !missing.has(id) && !rounds.has(id), "inconsistent missing round identity");
    missing.add(id);
  }
  const facts = /* @__PURE__ */ new Set();
  for (const row of record.facts) {
    const fact = row.fact;
    requireValue(
      fact && fact.id !== "" && !facts.has(fact.id) && fact.claimId === record.claimId,
      "duplicate or inconsistent fact identity"
    );
    facts.add(fact.id);
    requireValue(
      row.outgoingRelations.every((edge) => edge.sourceFactId === fact.id),
      "outgoing relation identity mismatch"
    );
    requireValue(
      row.incomingRelations.every((edge) => edge.targetFactId === fact.id),
      "incoming relation identity mismatch"
    );
    requireValue(
      row.statusTransitions.every((transition) => transition.factId === fact.id),
      "status transition identity mismatch"
    );
  }
}
async function queryClaimHistory(rpc, claimId, options = {}) {
  identifier(claimId, "claim ID");
  requireValue(rpc && typeof rpc.request === "function", "a protobuf RPC transport is required");
  const expectedChainId = options.expectedChainId;
  if (expectedChainId !== void 0) identifier(expectedChainId, "expected chain ID");
  const height = options.expectedHeight;
  requireValue(
    height === void 0 || typeof height === "bigint" && height > 0n && height <= maximumHeight,
    "expected height must be a positive uint64 bigint"
  );
  const limit = options.maximumResponseBytes ?? maximumBytes;
  requireValue(
    Number.isInteger(limit) && limit >= 1 && limit <= maximumBytes,
    "maximum response bytes must be between 1 and 8388608"
  );
  const request = QueryClaimHistoryRequest.fromPartial({ id: claimId, atBlockHeight: height ?? 0n });
  const bytes = await rpc.request(
    "zerone.knowledge.v1.Query",
    "ClaimHistory",
    QueryClaimHistoryRequest.encode(request).finish()
  );
  requireValue(
    bytes instanceof Uint8Array && bytes.byteLength > 0 && bytes.byteLength <= limit,
    "response is empty, invalid, or exceeds the byte limit"
  );
  const response = QueryClaimHistoryResponse.decode(bytes);
  const encoded = QueryClaimHistoryResponse.encode(response).finish();
  requireValue(
    encoded.length === bytes.length && encoded.every((byte, index) => byte === bytes[index]),
    "unsupported or noncanonical protobuf response"
  );
  identifier(response.chainId, "response chain ID");
  requireValue(response.blockHeight >= 0n && response.blockHeight <= maximumHeight, "invalid response height");
  if (height !== void 0) requireValue(response.blockHeight === height, "response height mismatch");
  if (expectedChainId !== void 0) {
    requireValue(response.chainId === expectedChainId, "response chain ID mismatch");
  }
  validateRecord(response.record);
  requireValue(response.record.claimId === claimId, "requested root claim identity mismatch");
  const relatedIds = /* @__PURE__ */ new Set([claimId]);
  const linkFields = /* @__PURE__ */ new Set(["provisional_fact_id", "challenged_claim_id", "relations.contradicts"]);
  for (const related of response.relatedClaims) {
    validateRecord(related.record);
    requireValue(!relatedIds.has(related.record.claimId), "duplicate related claim identity");
    relatedIds.add(related.record.claimId);
    requireValue(
      related.links.length > 0 && related.links.every((link) => linkFields.has(link.field) && link.targetId !== ""),
      "missing or unsupported related-claim link"
    );
  }
  return response;
}
export {
  ACCOUNT_REGISTRATION_PROOF_DOMAIN,
  COSMOS_AMOUNT_MAX,
  COSMOS_UINT64_MAX,
  CaipError,
  CidError,
  FeeGrantError,
  IN_TOTO_STATEMENT_V1_TYPE,
  KEY_ROTATION_ACCEPTANCE_DOMAIN,
  KEY_ROTATION_AUTHORIZATION_DOMAIN,
  KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS,
  LIQUIDITY_FEE_SCALE,
  LIQUIDITY_LEGACY_PROTOCOL_FEE_DESTINATION_MODULE,
  LIQUIDITY_POOL_STATUS,
  LiquidityClientError,
  MSG_CREATE_POOL_TYPE_URL,
  MSG_SUBMIT_PROPOSAL_TYPE_URL,
  MSG_SWAP_TYPE_URL,
  MSG_UPDATE_LIQUIDITY_PARAMS_TYPE_URL,
  ProvenanceParseError,
  ZERONE_MAX_POOL_RECORDS,
  ZERONE_MAX_SWAP_FEE,
  ZERONE_ONBOARDING_MESSAGE_TYPE_URLS,
  ZERONE_PROVENANCE_LIMITS,
  ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE,
  ZeroneLiquidityRestClient,
  accountRegistrationProofSignBytes,
  asExistingZeroneDid,
  asZeroneMemoryCid,
  computeReviewCommitmentV2,
  cosmosChainId,
  createExactInSwapPlan,
  createLiquidityAdmissionProposal,
  createLiquidityAdmissionUpdateMessage,
  createPoolMessage,
  createZeroneRegistry,
  defineZeroneNetwork,
  discloseLiquiditySwapFee,
  formatCaip10,
  formatCaip2,
  keyRotationAcceptanceSignBytes,
  keyRotationAuthorizationSignBytes,
  makeBoundedFeeGrant,
  makeReviewRevealV2,
  makeRevokeFeeGrant,
  makeSponsoredFee,
  minimumOutputForSlippage,
  parseCaip10,
  parseCaip2,
  parseCanonicalCidV1,
  parseCanonicalPositiveAmount,
  parseCosmosChainId,
  parseUnsignedZeroneInTotoStatement,
  queryClaimHistory,
  quoteConstantProductExactIn,
  registerZeroneMessages,
  timeoutHeightAfter,
  withTimeoutHeight,
  zeroneAccountId,
  zeroneRegistryTypes
};
