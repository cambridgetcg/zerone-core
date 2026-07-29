import {
  zeroneAccountId
} from "./chunk-YKTHEBLN.js";

// src/feegrant.ts
import {
  AllowedMsgAllowance,
  BasicAllowance
} from "cosmjs-types/cosmos/feegrant/v1beta1/feegrant";
import {
  MsgGrantAllowance,
  MsgRevokeAllowance
} from "cosmjs-types/cosmos/feegrant/v1beta1/tx";
var FeeGrantError = class extends Error {
  code;
  constructor(code, message) {
    super(message);
    this.name = "FeeGrantError";
    this.code = code;
  }
};
var DENOM_PATTERN = /^[a-zA-Z][a-zA-Z0-9/:._-]{2,127}$/;
var POSITIVE_INTEGER_PATTERN = /^[1-9][0-9]*$/;
var MESSAGE_TYPE_URL_PATTERN = /^\/[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z][A-Za-z0-9_]*)+\.Msg[A-Za-z0-9_]+$/;
var MAX_UINT64 = (1n << 64n) - 1n;
var MAX_SDK_INT = (1n << 256n) - 1n;
var MIN_PROTOBUF_TIMESTAMP_MILLISECONDS = -621355968e5;
var MAX_PROTOBUF_TIMESTAMP_MILLISECONDS = 253402300799999;
function validateParties(input) {
  zeroneAccountId(input.network, input.granter);
  zeroneAccountId(input.network, input.grantee);
  if (input.granter === input.grantee) {
    throw new FeeGrantError(
      "SELF_GRANT",
      "Fee allowance granter and grantee must be different Zerone accounts"
    );
  }
}
function validateAndSortCoins(coins, emptyCode) {
  if (coins.length === 0) {
    throw new FeeGrantError(
      emptyCode,
      emptyCode === "EMPTY_SPEND_LIMIT" ? "Fee allowance spend limit must not be empty or unlimited" : "Sponsored fee amount must contain at least one positive coin"
    );
  }
  const seenDenoms = /* @__PURE__ */ new Set();
  const validated = coins.map(({ denom, amount }) => {
    if (!DENOM_PATTERN.test(denom)) {
      throw new FeeGrantError(
        "INVALID_COIN",
        `Invalid Cosmos SDK coin denomination: ${denom}`
      );
    }
    if (amount.length > 78 || !POSITIVE_INTEGER_PATTERN.test(amount) || BigInt(amount) > MAX_SDK_INT) {
      throw new FeeGrantError(
        "INVALID_COIN",
        `Coin amount for ${denom} must be a canonical positive SDK integer`
      );
    }
    if (seenDenoms.has(denom)) {
      throw new FeeGrantError(
        "DUPLICATE_DENOM",
        `Duplicate fee denomination: ${denom}`
      );
    }
    seenDenoms.add(denom);
    return { denom, amount };
  });
  return validated.sort(
    (left, right) => left.denom < right.denom ? -1 : left.denom > right.denom ? 1 : 0
  );
}
function dateToTimestamp(expiration) {
  const now = /* @__PURE__ */ new Date();
  if (!(expiration instanceof Date)) {
    throw new FeeGrantError(
      "INVALID_EXPIRATION",
      "Fee allowance expiration and current time must be Date values"
    );
  }
  const expirationMilliseconds = expiration.getTime();
  const nowMilliseconds = now.getTime();
  if (!Number.isFinite(expirationMilliseconds) || !Number.isFinite(nowMilliseconds) || expirationMilliseconds < MIN_PROTOBUF_TIMESTAMP_MILLISECONDS || expirationMilliseconds > MAX_PROTOBUF_TIMESTAMP_MILLISECONDS || nowMilliseconds < MIN_PROTOBUF_TIMESTAMP_MILLISECONDS || nowMilliseconds > MAX_PROTOBUF_TIMESTAMP_MILLISECONDS) {
    throw new FeeGrantError(
      "INVALID_EXPIRATION",
      "Fee allowance expiration and current time must be valid protobuf timestamps"
    );
  }
  if (expirationMilliseconds <= nowMilliseconds) {
    throw new FeeGrantError(
      "EXPIRED_ALLOWANCE",
      "Fee allowance expiration must be in the future"
    );
  }
  return {
    seconds: BigInt(Math.floor(expirationMilliseconds / 1e3)),
    nanos: (expirationMilliseconds - Math.floor(expirationMilliseconds / 1e3) * 1e3) * 1e6
  };
}
function encodeUnsignedVarint(value) {
  if (value < 0n) {
    throw new RangeError("Cannot encode a negative unsigned protobuf varint");
  }
  const bytes = [];
  let remaining = value;
  do {
    const current = Number(remaining & 0x7fn);
    remaining >>= 7n;
    bytes.push(remaining === 0n ? current : current | 128);
  } while (remaining !== 0n);
  return Uint8Array.from(bytes);
}
function encodeTimestamp(timestamp) {
  const seconds = encodeUnsignedVarint(
    BigInt.asUintN(64, timestamp.seconds)
  );
  const nanos = encodeUnsignedVarint(BigInt(timestamp.nanos));
  const bytes = new Uint8Array(
    1 + seconds.length + (timestamp.nanos === 0 ? 0 : 1 + nanos.length)
  );
  let offset = 0;
  bytes[offset] = 8;
  offset += 1;
  bytes.set(seconds, offset);
  offset += seconds.length;
  if (timestamp.nanos !== 0) {
    bytes[offset] = 16;
    offset += 1;
    bytes.set(nanos, offset);
  }
  return bytes;
}
function encodeBasicAllowance(allowance) {
  const writer = BasicAllowance.encode({
    spendLimit: allowance.spendLimit,
    expiration: void 0
  });
  if (allowance.expiration !== void 0) {
    writer.uint32(18).bytes(encodeTimestamp(allowance.expiration));
  }
  return writer.finish();
}
function isSensitiveMessageTypeUrl(typeUrl) {
  const normalized = typeUrl.toLowerCase();
  const modulePath = normalized.slice(1, normalized.lastIndexOf(".msg"));
  const messageName = normalized.slice(normalized.lastIndexOf(".msg") + 4);
  const moduleSegments = modulePath.split(".");
  if (moduleSegments.some(
    (segment) => ["emergency", "upgrade", "gov", "governance", "params", "admin"].includes(
      segment
    )
  )) {
    return true;
  }
  return [
    "freeze",
    "unfreeze",
    "rotate",
    "params",
    "admin",
    "authority"
  ].some((fragment) => messageName.includes(fragment));
}
function validateAllowedMessages(typeUrls) {
  if (typeUrls.length === 0) {
    throw new FeeGrantError(
      "EMPTY_ALLOWED_MESSAGES",
      "Fee allowance must name at least one exact message type URL"
    );
  }
  const seen = /* @__PURE__ */ new Set();
  const validated = typeUrls.map((typeUrl) => {
    if (!MESSAGE_TYPE_URL_PATTERN.test(typeUrl)) {
      throw new FeeGrantError(
        "INVALID_MESSAGE_TYPE_URL",
        `Invalid exact protobuf message type URL: ${typeUrl}`
      );
    }
    if (seen.has(typeUrl)) {
      throw new FeeGrantError(
        "DUPLICATE_MESSAGE_TYPE_URL",
        `Duplicate allowed message type URL: ${typeUrl}`
      );
    }
    if (isSensitiveMessageTypeUrl(typeUrl)) {
      throw new FeeGrantError(
        "SENSITIVE_MESSAGE_TYPE_URL",
        `Refusing to sponsor sensitive message type: ${typeUrl}`
      );
    }
    seen.add(typeUrl);
    return typeUrl;
  });
  return validated.sort(
    (left, right) => left < right ? -1 : left > right ? 1 : 0
  );
}
function makeBoundedFeeGrant(input) {
  validateParties(input);
  const spendLimit = validateAndSortCoins(
    input.spendLimit,
    "EMPTY_SPEND_LIMIT"
  );
  const expiration = dateToTimestamp(input.expiration);
  const allowedMessages = validateAllowedMessages(
    input.allowedMessageTypeUrls
  );
  const basicAllowance = BasicAllowance.fromPartial({
    spendLimit,
    expiration
  });
  const allowedMsgAllowance = AllowedMsgAllowance.fromPartial({
    allowance: {
      typeUrl: BasicAllowance.typeUrl,
      value: encodeBasicAllowance(basicAllowance)
    },
    allowedMessages
  });
  return {
    typeUrl: MsgGrantAllowance.typeUrl,
    value: MsgGrantAllowance.fromPartial({
      granter: input.granter,
      grantee: input.grantee,
      allowance: {
        typeUrl: AllowedMsgAllowance.typeUrl,
        value: AllowedMsgAllowance.encode(allowedMsgAllowance).finish()
      }
    })
  };
}
function makeRevokeFeeGrant(input) {
  validateParties(input);
  return {
    typeUrl: MsgRevokeAllowance.typeUrl,
    value: MsgRevokeAllowance.fromPartial({
      granter: input.granter,
      grantee: input.grantee
    })
  };
}
function makeSponsoredFee(input) {
  zeroneAccountId(input.network, input.granter);
  if (input.gas.length > 20 || !POSITIVE_INTEGER_PATTERN.test(input.gas) || BigInt(input.gas) > MAX_UINT64) {
    throw new FeeGrantError(
      "INVALID_GAS",
      "Sponsored fee gas must be a canonical positive uint64"
    );
  }
  return {
    amount: validateAndSortCoins(input.amount, "INVALID_COIN"),
    gas: input.gas,
    granter: input.granter
  };
}

export {
  FeeGrantError,
  makeBoundedFeeGrant,
  makeRevokeFeeGrant,
  makeSponsoredFee
};
