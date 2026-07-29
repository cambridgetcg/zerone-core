import type { EncodeObject } from "@cosmjs/proto-signing";
import type { Coin } from "cosmjs-types/cosmos/base/v1beta1/coin";
import {
  AllowedMsgAllowance,
  BasicAllowance,
} from "cosmjs-types/cosmos/feegrant/v1beta1/feegrant";
import {
  MsgGrantAllowance,
  MsgRevokeAllowance,
} from "cosmjs-types/cosmos/feegrant/v1beta1/tx";
import type { Timestamp } from "cosmjs-types/google/protobuf/timestamp";
import { zeroneAccountId, type ZeroneNetwork } from "./caip";

export type FeeGrantErrorCode =
  | "SELF_GRANT"
  | "EMPTY_SPEND_LIMIT"
  | "INVALID_COIN"
  | "DUPLICATE_DENOM"
  | "INVALID_EXPIRATION"
  | "EXPIRED_ALLOWANCE"
  | "EMPTY_ALLOWED_MESSAGES"
  | "INVALID_MESSAGE_TYPE_URL"
  | "DUPLICATE_MESSAGE_TYPE_URL"
  | "SENSITIVE_MESSAGE_TYPE_URL"
  | "INVALID_GAS";

export class FeeGrantError extends Error {
  readonly code: FeeGrantErrorCode;

  constructor(code: FeeGrantErrorCode, message: string) {
    super(message);
    this.name = "FeeGrantError";
    this.code = code;
  }
}

export interface FeeGrantCoin {
  readonly denom: string;
  readonly amount: string;
}

export interface FeeGrantParties {
  readonly network: ZeroneNetwork;
  readonly granter: string;
  readonly grantee: string;
}

export interface BoundedFeeGrantInput extends FeeGrantParties {
  /**
   * A required, finite lifetime budget. Empty limits are deliberately rejected
   * because Cosmos SDK interprets them as unlimited.
   */
  readonly spendLimit: readonly FeeGrantCoin[];
  /** A required wall-clock expiry. */
  readonly expiration: Date;
  /**
   * Exact protobuf message type URLs that the sponsor agrees to pay for.
   * Wildcards and privileged control-plane messages are rejected.
   */
  readonly allowedMessageTypeUrls: readonly string[];
}

export interface SponsoredFeeInput {
  readonly network: ZeroneNetwork;
  readonly granter: string;
  readonly amount: readonly FeeGrantCoin[];
  readonly gas: string;
}

/**
 * Structural counterpart of CosmJS's StdFee with a required fee granter.
 * Keeping this structural avoids making @cosmjs/amino a runtime dependency.
 */
export interface SponsoredStdFee {
  readonly amount: readonly FeeGrantCoin[];
  readonly gas: string;
  readonly granter: string;
}

const DENOM_PATTERN = /^[a-zA-Z][a-zA-Z0-9/:._-]{2,127}$/;
const POSITIVE_INTEGER_PATTERN = /^[1-9][0-9]*$/;
const MESSAGE_TYPE_URL_PATTERN =
  /^\/[A-Za-z][A-Za-z0-9_]*(?:\.[A-Za-z][A-Za-z0-9_]*)+\.Msg[A-Za-z0-9_]+$/;
const MAX_UINT64 = (1n << 64n) - 1n;
const MAX_SDK_INT = (1n << 256n) - 1n;
const MIN_PROTOBUF_TIMESTAMP_MILLISECONDS = -62_135_596_800_000;
const MAX_PROTOBUF_TIMESTAMP_MILLISECONDS = 253_402_300_799_999;

function validateParties(input: FeeGrantParties): void {
  zeroneAccountId(input.network, input.granter);
  zeroneAccountId(input.network, input.grantee);
  if (input.granter === input.grantee) {
    throw new FeeGrantError(
      "SELF_GRANT",
      "Fee allowance granter and grantee must be different Zerone accounts",
    );
  }
}

function validateAndSortCoins(
  coins: readonly FeeGrantCoin[],
  emptyCode: "EMPTY_SPEND_LIMIT" | "INVALID_COIN",
): Coin[] {
  if (coins.length === 0) {
    throw new FeeGrantError(
      emptyCode,
      emptyCode === "EMPTY_SPEND_LIMIT"
        ? "Fee allowance spend limit must not be empty or unlimited"
        : "Sponsored fee amount must contain at least one positive coin",
    );
  }

  const seenDenoms = new Set<string>();
  const validated = coins.map(({ denom, amount }): Coin => {
    if (!DENOM_PATTERN.test(denom)) {
      throw new FeeGrantError(
        "INVALID_COIN",
        `Invalid Cosmos SDK coin denomination: ${denom}`,
      );
    }
    if (
      amount.length > 78 ||
      !POSITIVE_INTEGER_PATTERN.test(amount) ||
      BigInt(amount) > MAX_SDK_INT
    ) {
      throw new FeeGrantError(
        "INVALID_COIN",
        `Coin amount for ${denom} must be a canonical positive SDK integer`,
      );
    }
    if (seenDenoms.has(denom)) {
      throw new FeeGrantError(
        "DUPLICATE_DENOM",
        `Duplicate fee denomination: ${denom}`,
      );
    }
    seenDenoms.add(denom);
    return { denom, amount };
  });

  return validated.sort((left, right) =>
    left.denom < right.denom ? -1 : left.denom > right.denom ? 1 : 0,
  );
}

function dateToTimestamp(expiration: Date): Timestamp {
  const now = new Date();
  if (!(expiration instanceof Date)) {
    throw new FeeGrantError(
      "INVALID_EXPIRATION",
      "Fee allowance expiration and current time must be Date values",
    );
  }
  const expirationMilliseconds = expiration.getTime();
  const nowMilliseconds = now.getTime();
  if (
    !Number.isFinite(expirationMilliseconds) ||
    !Number.isFinite(nowMilliseconds) ||
    expirationMilliseconds < MIN_PROTOBUF_TIMESTAMP_MILLISECONDS ||
    expirationMilliseconds > MAX_PROTOBUF_TIMESTAMP_MILLISECONDS ||
    nowMilliseconds < MIN_PROTOBUF_TIMESTAMP_MILLISECONDS ||
    nowMilliseconds > MAX_PROTOBUF_TIMESTAMP_MILLISECONDS
  ) {
    throw new FeeGrantError(
      "INVALID_EXPIRATION",
      "Fee allowance expiration and current time must be valid protobuf timestamps",
    );
  }
  if (expirationMilliseconds <= nowMilliseconds) {
    throw new FeeGrantError(
      "EXPIRED_ALLOWANCE",
      "Fee allowance expiration must be in the future",
    );
  }

  return {
    seconds: BigInt(Math.floor(expirationMilliseconds / 1_000)),
    nanos:
      (expirationMilliseconds -
        Math.floor(expirationMilliseconds / 1_000) * 1_000) *
      1_000_000,
  };
}

function encodeUnsignedVarint(value: bigint): Uint8Array {
  if (value < 0n) {
    throw new RangeError("Cannot encode a negative unsigned protobuf varint");
  }
  const bytes: number[] = [];
  let remaining = value;
  do {
    const current = Number(remaining & 0x7fn);
    remaining >>= 7n;
    bytes.push(remaining === 0n ? current : current | 0x80);
  } while (remaining !== 0n);
  return Uint8Array.from(bytes);
}

function encodeTimestamp(timestamp: Timestamp): Uint8Array {
  const seconds = encodeUnsignedVarint(
    BigInt.asUintN(64, timestamp.seconds),
  );
  const nanos = encodeUnsignedVarint(BigInt(timestamp.nanos));
  const bytes = new Uint8Array(
    1 + seconds.length + (timestamp.nanos === 0 ? 0 : 1 + nanos.length),
  );
  let offset = 0;
  bytes[offset] = 0x08;
  offset += 1;
  bytes.set(seconds, offset);
  offset += seconds.length;
  if (timestamp.nanos !== 0) {
    bytes[offset] = 0x10;
    offset += 1;
    bytes.set(nanos, offset);
  }
  return bytes;
}

function encodeBasicAllowance(allowance: BasicAllowance): Uint8Array {
  // cosmjs-types 0.11's BinaryWriter underallocates some positive int64
  // varints above 2^31-1. Write the Timestamp field explicitly so valid
  // post-2038 expirations cannot be silently truncated.
  const writer = BasicAllowance.encode({
    spendLimit: allowance.spendLimit,
    expiration: undefined,
  });
  if (allowance.expiration !== undefined) {
    writer.uint32(0x12).bytes(encodeTimestamp(allowance.expiration));
  }
  return writer.finish();
}

function isSensitiveMessageTypeUrl(typeUrl: string): boolean {
  const normalized = typeUrl.toLowerCase();
  const modulePath = normalized.slice(1, normalized.lastIndexOf(".msg"));
  const messageName = normalized.slice(normalized.lastIndexOf(".msg") + 4);
  const moduleSegments = modulePath.split(".");

  if (
    moduleSegments.some((segment) =>
      ["emergency", "upgrade", "gov", "governance", "params", "admin"].includes(
        segment,
      ),
    )
  ) {
    return true;
  }

  return [
    "freeze",
    "unfreeze",
    "rotate",
    "params",
    "admin",
    "authority",
  ].some((fragment) => messageName.includes(fragment));
}

function validateAllowedMessages(typeUrls: readonly string[]): string[] {
  if (typeUrls.length === 0) {
    throw new FeeGrantError(
      "EMPTY_ALLOWED_MESSAGES",
      "Fee allowance must name at least one exact message type URL",
    );
  }

  const seen = new Set<string>();
  const validated = typeUrls.map((typeUrl) => {
    if (!MESSAGE_TYPE_URL_PATTERN.test(typeUrl)) {
      throw new FeeGrantError(
        "INVALID_MESSAGE_TYPE_URL",
        `Invalid exact protobuf message type URL: ${typeUrl}`,
      );
    }
    if (seen.has(typeUrl)) {
      throw new FeeGrantError(
        "DUPLICATE_MESSAGE_TYPE_URL",
        `Duplicate allowed message type URL: ${typeUrl}`,
      );
    }
    if (isSensitiveMessageTypeUrl(typeUrl)) {
      throw new FeeGrantError(
        "SENSITIVE_MESSAGE_TYPE_URL",
        `Refusing to sponsor sensitive message type: ${typeUrl}`,
      );
    }
    seen.add(typeUrl);
    return typeUrl;
  });

  return validated.sort((left, right) =>
    left < right ? -1 : left > right ? 1 : 0,
  );
}

/**
 * Builds a finite Cosmos SDK fee allowance for Zerone onboarding.
 *
 * The BasicAllowance is always wrapped in AllowedMsgAllowance, so neither the
 * spend budget, lifetime, nor permitted message set can be omitted.
 */
export function makeBoundedFeeGrant(input: BoundedFeeGrantInput): EncodeObject {
  validateParties(input);
  const spendLimit = validateAndSortCoins(
    input.spendLimit,
    "EMPTY_SPEND_LIMIT",
  );
  const expiration = dateToTimestamp(input.expiration);
  const allowedMessages = validateAllowedMessages(
    input.allowedMessageTypeUrls,
  );

  const basicAllowance = BasicAllowance.fromPartial({
    spendLimit,
    expiration,
  });
  const allowedMsgAllowance = AllowedMsgAllowance.fromPartial({
    allowance: {
      typeUrl: BasicAllowance.typeUrl,
      value: encodeBasicAllowance(basicAllowance),
    },
    allowedMessages,
  });

  return {
    typeUrl: MsgGrantAllowance.typeUrl,
    value: MsgGrantAllowance.fromPartial({
      granter: input.granter,
      grantee: input.grantee,
      allowance: {
        typeUrl: AllowedMsgAllowance.typeUrl,
        value: AllowedMsgAllowance.encode(allowedMsgAllowance).finish(),
      },
    }),
  };
}

/** Builds a standard Cosmos SDK message that revokes one Zerone fee grant. */
export function makeRevokeFeeGrant(input: FeeGrantParties): EncodeObject {
  validateParties(input);
  return {
    typeUrl: MsgRevokeAllowance.typeUrl,
    value: MsgRevokeAllowance.fromPartial({
      granter: input.granter,
      grantee: input.grantee,
    }),
  };
}

/**
 * Adds a validated Zerone fee granter to a concrete positive CosmJS fee.
 *
 * This does not query the chain or prove that a matching allowance exists.
 */
export function makeSponsoredFee(input: SponsoredFeeInput): SponsoredStdFee {
  zeroneAccountId(input.network, input.granter);
  if (
    input.gas.length > 20 ||
    !POSITIVE_INTEGER_PATTERN.test(input.gas) ||
    BigInt(input.gas) > MAX_UINT64
  ) {
    throw new FeeGrantError(
      "INVALID_GAS",
      "Sponsored fee gas must be a canonical positive uint64",
    );
  }

  return {
    amount: validateAndSortCoins(input.amount, "INVALID_COIN"),
    gas: input.gas,
    granter: input.granter,
  };
}
