import type { EncodeObject } from "@cosmjs/proto-signing";
import {
  defineZeroneNetwork,
  zeroneAccountId,
} from "@zerone-chain/sdk/caip";
import {
  AllowedMsgAllowance,
  BasicAllowance,
} from "cosmjs-types/cosmos/feegrant/v1beta1/feegrant";
import {
  MsgGrantAllowance,
  MsgRevokeAllowance,
} from "cosmjs-types/cosmos/feegrant/v1beta1/tx";

export const BANK_SEND_TYPE_URL = "/cosmos.bank.v1beta1.MsgSend";
export const CLAIM_TYPE_URL = "/zerone.claiming_pot.v1.MsgClaim";
export const GRANTABLE_MESSAGE_TYPES = [
  BANK_SEND_TYPE_URL,
  CLAIM_TYPE_URL,
] as const;
export const MAX_GRANT_UZRN = 100_000_000n;
export const MAX_GRANT_DURATION_MS = 30 * 24 * 60 * 60 * 1_000;

const BASIC_ALLOWANCE_TYPE =
  "/cosmos.feegrant.v1beta1.BasicAllowance";
const PERIODIC_ALLOWANCE_TYPE =
  "/cosmos.feegrant.v1beta1.PeriodicAllowance";
const ALLOWED_MSG_ALLOWANCE_TYPE =
  "/cosmos.feegrant.v1beta1.AllowedMsgAllowance";
const MAX_ALLOWANCES_PER_PAGE = 50;
const MAX_COINS = 20;
const MAX_ALLOWED_MESSAGES = 64;
const NETWORK = defineZeroneNetwork("zerone-1");
const TYPE_URL = /^\/[A-Za-z][A-Za-z0-9_.]{0,198}$/;
const DENOM = /^[a-z][a-z0-9/:._-]{0,127}$/;
const RFC3339 =
  /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?Z$/;
const BASE64 =
  /^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/;

export interface FeeGrantCoin {
  denom: string;
  amount: string;
}

export interface FeeGrantAllowance {
  granter: string;
  grantee: string;
  typeUrl: string;
  supported: boolean;
  spendLimit: readonly FeeGrantCoin[] | null;
  periodSpendLimit: readonly FeeGrantCoin[] | null;
  periodCanSpend: readonly FeeGrantCoin[] | null;
  expiration: string | null;
  periodReset: string | null;
  allowedMessages: readonly string[] | null;
}

export interface FeeGrantPage {
  allowances: FeeGrantAllowance[];
  nextKey: string | null;
  total: string | null;
}

interface ParsedAllowance {
  typeUrl: string;
  supported: boolean;
  spendLimit: readonly FeeGrantCoin[] | null;
  periodSpendLimit: readonly FeeGrantCoin[] | null;
  periodCanSpend: readonly FeeGrantCoin[] | null;
  expiration: string | null;
  periodReset: string | null;
  allowedMessages: readonly string[] | null;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function isMissingFeeGrantError(value: unknown): boolean {
  if (!isRecord(value)) return false;
  const keys = Object.keys(value).sort();
  return (
    keys.length === 3 &&
    keys[0] === "code" &&
    keys[1] === "details" &&
    keys[2] === "message" &&
    value.code === 13 &&
    value.message === "fee-grant not found: not found" &&
    Array.isArray(value.details) &&
    value.details.length === 0
  );
}

function recordField(
  record: Record<string, unknown>,
  snakeCase: string,
  camelCase: string,
): unknown {
  return record[snakeCase] ?? record[camelCase];
}

function accountAddress(value: unknown): string {
  if (typeof value !== "string") {
    throw new Error("Fee grant contained a non-string account address.");
  }
  try {
    zeroneAccountId(NETWORK, value);
  } catch {
    throw new Error("Fee grant contained an invalid Zerone account address.");
  }
  return value;
}

function coins(value: unknown, field: string): readonly FeeGrantCoin[] {
  if (!Array.isArray(value) || value.length > MAX_COINS) {
    throw new Error(`Fee grant ${field} was not a bounded coin list.`);
  }
  const seen = new Set<string>();
  return value.map((item) => {
    if (!isRecord(item)) {
      throw new Error(`Fee grant ${field} contained a malformed coin.`);
    }
    const denom = item.denom;
    const amount = item.amount;
    if (
      typeof denom !== "string" ||
      !DENOM.test(denom) ||
      typeof amount !== "string" ||
      amount.length > 78 ||
      !/^(?:0|[1-9]\d*)$/.test(amount) ||
      seen.has(denom)
    ) {
      throw new Error(`Fee grant ${field} contained a malformed coin.`);
    }
    seen.add(denom);
    return { denom, amount };
  });
}

function timestamp(value: unknown, field: string): string | null {
  if (value === null || value === undefined) return null;
  if (typeof value !== "string" || value.length > 40) {
    throw new Error(`Fee grant ${field} was not a valid UTC timestamp.`);
  }
  const match = RFC3339.exec(value);
  if (!match) {
    throw new Error(`Fee grant ${field} was not a valid UTC timestamp.`);
  }
  const year = Number(match[1] ?? 0);
  const month = Number(match[2] ?? 0);
  const day = Number(match[3] ?? 0);
  const hour = Number(match[4] ?? -1);
  const minute = Number(match[5] ?? -1);
  const second = Number(match[6] ?? -1);
  const leapYear =
    year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const daysInMonth = [
    31,
    leapYear ? 29 : 28,
    31,
    30,
    31,
    30,
    31,
    31,
    30,
    31,
    30,
    31,
  ][month - 1];
  if (
    year < 1 ||
    daysInMonth === undefined ||
    day < 1 ||
    day > daysInMonth ||
    hour > 23 ||
    minute > 59 ||
    second > 59 ||
    !Number.isFinite(Date.parse(value))
  ) {
    throw new Error(`Fee grant ${field} was not a valid UTC timestamp.`);
  }
  return value;
}

function messageTypes(value: unknown): readonly string[] {
  if (
    !Array.isArray(value) ||
    value.length === 0 ||
    value.length > MAX_ALLOWED_MESSAGES
  ) {
    throw new Error("Fee grant contained an invalid allowed-message list.");
  }
  const seen = new Set<string>();
  return value.map((item) => {
    if (
      typeof item !== "string" ||
      !TYPE_URL.test(item) ||
      seen.has(item)
    ) {
      throw new Error("Fee grant contained an invalid allowed-message list.");
    }
    seen.add(item);
    return item;
  });
}

function intersectMessages(
  outer: readonly string[],
  inner: readonly string[] | null,
): readonly string[] {
  if (inner === null) return outer;
  const innerSet = new Set(inner);
  return outer.filter((message) => innerSet.has(message));
}

function parseBasicAllowance(
  raw: Record<string, unknown>,
  typeUrl = BASIC_ALLOWANCE_TYPE,
): ParsedAllowance {
  const spendLimit = coins(
    recordField(raw, "spend_limit", "spendLimit") ?? [],
    "spend limit",
  );
  return {
    typeUrl,
    supported: true,
    spendLimit: spendLimit.length === 0 ? null : spendLimit,
    periodSpendLimit: null,
    periodCanSpend: null,
    expiration: timestamp(raw.expiration, "expiration"),
    periodReset: null,
    allowedMessages: null,
  };
}

function parseAllowance(raw: unknown, depth = 0): ParsedAllowance {
  if (!isRecord(raw) || depth > 4) {
    throw new Error("Fee grant allowance nesting was malformed or too deep.");
  }
  const typeUrl = raw["@type"];
  if (
    typeof typeUrl !== "string" ||
    typeUrl.length > 200 ||
    !TYPE_URL.test(typeUrl)
  ) {
    throw new Error("Fee grant allowance type was malformed.");
  }

  if (typeUrl === BASIC_ALLOWANCE_TYPE) {
    return parseBasicAllowance(raw);
  }

  if (typeUrl === PERIODIC_ALLOWANCE_TYPE) {
    const basicRaw = raw.basic;
    if (!isRecord(basicRaw)) {
      throw new Error("Periodic fee grant did not contain a basic allowance.");
    }
    const basic = parseBasicAllowance(basicRaw, typeUrl);
    const periodCanSpend = coins(
      recordField(raw, "period_can_spend", "periodCanSpend"),
      "period remaining",
    );
    const periodSpendLimit = coins(
      recordField(raw, "period_spend_limit", "periodSpendLimit"),
      "period spend limit",
    );
    const period = raw.period;
    if (
      typeof period !== "string" ||
      period.length > 40 ||
      !/^(?:0|[1-9]\d*)(?:\.\d{1,9})?s$/.test(period) ||
      periodSpendLimit.length === 0
    ) {
      throw new Error("Periodic fee grant contained an invalid period.");
    }
    return {
      ...basic,
      periodSpendLimit,
      periodCanSpend,
      periodReset: timestamp(
        recordField(raw, "period_reset", "periodReset"),
        "period reset",
      ),
    };
  }

  if (typeUrl === ALLOWED_MSG_ALLOWANCE_TYPE) {
    const inner = parseAllowance(raw.allowance, depth + 1);
    const allowedMessages = messageTypes(
      recordField(raw, "allowed_messages", "allowedMessages"),
    );
    return {
      ...inner,
      typeUrl,
      allowedMessages: intersectMessages(
        allowedMessages,
        inner.allowedMessages,
      ),
    };
  }

  return {
    typeUrl,
    supported: false,
    spendLimit: null,
    periodSpendLimit: null,
    periodCanSpend: null,
    expiration: null,
    periodReset: null,
    allowedMessages: null,
  };
}

function paginationKey(value: unknown): string | null {
  if (value === null || value === undefined || value === "") return null;
  if (
    typeof value !== "string" ||
    value.length > 344 ||
    !BASE64.test(value)
  ) {
    throw new Error("Fee grant pagination key was malformed.");
  }
  return value;
}

export function parseFeeGrantPage(value: unknown): FeeGrantPage {
  if (!isRecord(value) || !Array.isArray(value.allowances)) {
    throw new Error("Fee grant response was incomplete.");
  }
  if (value.allowances.length > MAX_ALLOWANCES_PER_PAGE) {
    throw new Error("Fee grant response exceeded the page limit.");
  }
  const allowances = value.allowances.map((grant) => {
    if (!isRecord(grant)) {
      throw new Error("Fee grant response contained a malformed grant.");
    }
    const parsed = parseAllowance(grant.allowance);
    return {
      granter: accountAddress(grant.granter),
      grantee: accountAddress(grant.grantee),
      ...parsed,
    };
  });

  const pagination = value.pagination;
  if (pagination !== null && pagination !== undefined && !isRecord(pagination)) {
    throw new Error("Fee grant pagination was malformed.");
  }
  const total = pagination?.total;
  if (
    total !== null &&
    total !== undefined &&
    (typeof total !== "string" ||
      total.length > 20 ||
      !/^(?:0|[1-9]\d*)$/.test(total))
  ) {
    throw new Error("Fee grant pagination total was malformed.");
  }
  return {
    allowances,
    nextKey: paginationKey(
      pagination?.next_key ?? pagination?.nextKey,
    ),
    total: typeof total === "string" ? total : null,
  };
}

function uzrnAmount(coinsValue: readonly FeeGrantCoin[] | null): bigint | null {
  if (coinsValue === null) return null;
  const native = coinsValue.find((coin) => coin.denom === "uzrn");
  return native ? BigInt(native.amount) : 0n;
}

export function feeGrantAllowsMessage(
  grant: FeeGrantAllowance,
  messageType: string,
  requiredFeeUzrn: bigint,
  now = Date.now(),
): boolean {
  if (!grant.supported || requiredFeeUzrn <= 0n) return false;
  if (
    grant.expiration !== null &&
    Date.parse(grant.expiration) <= now
  ) {
    return false;
  }
  if (
    grant.allowedMessages !== null &&
    !grant.allowedMessages.includes(messageType)
  ) {
    return false;
  }
  const overall = uzrnAmount(grant.spendLimit);
  if (overall !== null && overall < requiredFeeUzrn) return false;
  const periodCoins =
    grant.periodReset !== null &&
    Date.parse(grant.periodReset) <= now
      ? grant.periodSpendLimit
      : grant.periodCanSpend;
  const period = uzrnAmount(periodCoins);
  if (period !== null && period < requiredFeeUzrn) return false;
  return true;
}

export function displayZrnToMicro(value: string): bigint {
  const normalized = value.trim();
  if (!/^\d+(?:\.\d{0,6})?$/.test(normalized)) {
    throw new Error("Use no more than 6 decimal places.");
  }
  const [whole = "0", fraction = ""] = normalized.split(".");
  const amount =
    BigInt(whole) * 1_000_000n +
    BigInt(fraction.padEnd(6, "0"));
  if (amount <= 0n) throw new Error("Spend limit must be greater than zero.");
  if (amount > MAX_GRANT_UZRN) {
    throw new Error("Fee grants are capped at 100 ZRN in this interface.");
  }
  return amount;
}

function validateGrantMessageTypes(
  values: readonly string[],
): readonly string[] {
  const allowed = new Set<string>(GRANTABLE_MESSAGE_TYPES);
  const unique = [...new Set(values)];
  if (
    unique.length === 0 ||
    unique.length !== values.length ||
    unique.some((value) => !allowed.has(value))
  ) {
    throw new Error("Choose at least one supported message type.");
  }
  return unique;
}

function expirationTimestamp(
  expiration: Date,
  now = Date.now(),
): { seconds: bigint; nanos: number } {
  const milliseconds = expiration.getTime();
  if (
    !Number.isFinite(milliseconds) ||
    milliseconds <= now + 60_000 ||
    milliseconds > now + MAX_GRANT_DURATION_MS
  ) {
    throw new Error("Expiration must be between one minute and 30 days away.");
  }
  return {
    seconds: BigInt(Math.floor(milliseconds / 1_000)),
    nanos: Math.floor(milliseconds % 1_000) * 1_000_000,
  };
}

function encodeUnsignedVarint(value: bigint): Uint8Array {
  if (value < 0n) {
    throw new RangeError("Cannot encode a negative unsigned protobuf varint.");
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

function encodeTimestamp(timestamp: {
  seconds: bigint;
  nanos: number;
}): Uint8Array {
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
  // cosmjs-types 0.11 corrupts some positive int64 varints above 2^31-1
  // through signed low-word handling. Encode Timestamp explicitly.
  const writer = BasicAllowance.encode({
    spendLimit: allowance.spendLimit,
    expiration: undefined,
  });
  if (allowance.expiration !== undefined) {
    writer.uint32(0x12).bytes(encodeTimestamp(allowance.expiration));
  }
  return writer.finish();
}

export function createBoundedGrantMessage(input: {
  granter: string;
  grantee: string;
  spendLimitZrn: string;
  expiration: Date;
  allowedMessages: readonly string[];
  now?: number;
}): EncodeObject {
  const granter = accountAddress(input.granter);
  const grantee = accountAddress(input.grantee);
  if (granter === grantee) {
    throw new Error("A fee grant must name a different grantee.");
  }
  const amount = displayZrnToMicro(input.spendLimitZrn);
  const allowedMessages = validateGrantMessageTypes(input.allowedMessages);
  const expiration = expirationTimestamp(
    input.expiration,
    input.now ?? Date.now(),
  );
  const basic = BasicAllowance.fromPartial({
    spendLimit: [{ denom: "uzrn", amount: amount.toString() }],
    expiration,
  });
  const basicAny = {
    typeUrl: BASIC_ALLOWANCE_TYPE,
    value: encodeBasicAllowance(basic),
  };
  const restricted = AllowedMsgAllowance.fromPartial({
    allowance: basicAny,
    allowedMessages: [...allowedMessages],
  });
  const allowance = {
    typeUrl: ALLOWED_MSG_ALLOWANCE_TYPE,
    value: AllowedMsgAllowance.encode(restricted).finish(),
  };
  return {
    typeUrl: MsgGrantAllowance.typeUrl,
    value: MsgGrantAllowance.fromPartial({ granter, grantee, allowance }),
  };
}

export function createRevokeGrantMessage(
  granterValue: string,
  granteeValue: string,
): EncodeObject {
  const granter = accountAddress(granterValue);
  const grantee = accountAddress(granteeValue);
  if (granter === grantee) {
    throw new Error("A fee grant must name a different grantee.");
  }
  return {
    typeUrl: MsgRevokeAllowance.typeUrl,
    value: MsgRevokeAllowance.fromPartial({ granter, grantee }),
  };
}
