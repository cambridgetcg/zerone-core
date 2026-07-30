import {
  __export
} from "./chunk-MLKGABMK.js";

// src/generated/zerone/liquiditypool/v1/tx.ts
var tx_exports = {};
__export(tx_exports, {
  MsgAddLiquidity: () => MsgAddLiquidity,
  MsgAddLiquidityResponse: () => MsgAddLiquidityResponse,
  MsgCreatePool: () => MsgCreatePool,
  MsgCreatePoolResponse: () => MsgCreatePoolResponse,
  MsgRemoveLiquidity: () => MsgRemoveLiquidity,
  MsgRemoveLiquidityResponse: () => MsgRemoveLiquidityResponse,
  MsgSetPoolStatus: () => MsgSetPoolStatus,
  MsgSetPoolStatusResponse: () => MsgSetPoolStatusResponse,
  MsgSwap: () => MsgSwap,
  MsgSwapResponse: () => MsgSwapResponse,
  MsgUpdateParams: () => MsgUpdateParams,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse
});

// src/generated/utf8.ts
function utf8Length(str) {
  let len = 0, c = 0;
  for (let i = 0; i < str.length; ++i) {
    c = str.charCodeAt(i);
    if (c < 128) len += 1;
    else if (c < 2048) len += 2;
    else if ((c & 64512) === 55296 && (str.charCodeAt(i + 1) & 64512) === 56320) {
      ++i;
      len += 4;
    } else len += 3;
  }
  return len;
}
function utf8Read(buffer, start, end) {
  const len = end - start;
  if (len < 1) return "";
  const chunk = [];
  let parts = [], i = 0, t;
  while (start < end) {
    t = buffer[start++];
    if (t < 128) chunk[i++] = t;
    else if (t > 191 && t < 224)
      chunk[i++] = (t & 31) << 6 | buffer[start++] & 63;
    else if (t > 239 && t < 365) {
      t = ((t & 7) << 18 | (buffer[start++] & 63) << 12 | (buffer[start++] & 63) << 6 | buffer[start++] & 63) - 65536;
      chunk[i++] = 55296 + (t >> 10);
      chunk[i++] = 56320 + (t & 1023);
    } else
      chunk[i++] = (t & 15) << 12 | (buffer[start++] & 63) << 6 | buffer[start++] & 63;
    if (i > 8191) {
      (parts || (parts = [])).push(String.fromCharCode(...chunk));
      i = 0;
    }
  }
  if (parts) {
    if (i) parts.push(String.fromCharCode(...chunk.slice(0, i)));
    return parts.join("");
  }
  return String.fromCharCode(...chunk.slice(0, i));
}
function utf8Write(str, buffer, offset) {
  const start = offset;
  let c1, c2;
  for (let i = 0; i < str.length; ++i) {
    c1 = str.charCodeAt(i);
    if (c1 < 128) {
      buffer[offset++] = c1;
    } else if (c1 < 2048) {
      buffer[offset++] = c1 >> 6 | 192;
      buffer[offset++] = c1 & 63 | 128;
    } else if ((c1 & 64512) === 55296 && ((c2 = str.charCodeAt(i + 1)) & 64512) === 56320) {
      c1 = 65536 + ((c1 & 1023) << 10) + (c2 & 1023);
      ++i;
      buffer[offset++] = c1 >> 18 | 240;
      buffer[offset++] = c1 >> 12 & 63 | 128;
      buffer[offset++] = c1 >> 6 & 63 | 128;
      buffer[offset++] = c1 & 63 | 128;
    } else {
      buffer[offset++] = c1 >> 12 | 224;
      buffer[offset++] = c1 >> 6 & 63 | 128;
      buffer[offset++] = c1 & 63 | 128;
    }
  }
  return offset - start;
}

// src/generated/varint.ts
function varint64read() {
  let lowBits = 0;
  let highBits = 0;
  for (let shift = 0; shift < 28; shift += 7) {
    let b = this.buf[this.pos++];
    lowBits |= (b & 127) << shift;
    if ((b & 128) == 0) {
      this.assertBounds();
      return [lowBits, highBits];
    }
  }
  let middleByte = this.buf[this.pos++];
  lowBits |= (middleByte & 15) << 28;
  highBits = (middleByte & 112) >> 4;
  if ((middleByte & 128) == 0) {
    this.assertBounds();
    return [lowBits, highBits];
  }
  for (let shift = 3; shift <= 31; shift += 7) {
    let b = this.buf[this.pos++];
    highBits |= (b & 127) << shift;
    if ((b & 128) == 0) {
      this.assertBounds();
      return [lowBits, highBits];
    }
  }
  throw new Error("invalid varint");
}
var TWO_PWR_32_DBL = 4294967296;
function int64FromString(dec) {
  const minus = dec[0] === "-";
  if (minus) {
    dec = dec.slice(1);
  }
  const base = 1e6;
  let lowBits = 0;
  let highBits = 0;
  function add1e6digit(begin, end) {
    const digit1e6 = Number(dec.slice(begin, end));
    highBits *= base;
    lowBits = lowBits * base + digit1e6;
    if (lowBits >= TWO_PWR_32_DBL) {
      highBits = highBits + (lowBits / TWO_PWR_32_DBL | 0);
      lowBits = lowBits % TWO_PWR_32_DBL;
    }
  }
  add1e6digit(-24, -18);
  add1e6digit(-18, -12);
  add1e6digit(-12, -6);
  add1e6digit(-6);
  return minus ? negate(lowBits, highBits) : newBits(lowBits, highBits);
}
function int64ToString(lo, hi) {
  let bits = newBits(lo, hi);
  const negative = bits.hi & 2147483648;
  if (negative) {
    bits = negate(bits.lo, bits.hi);
  }
  const result = uInt64ToString(bits.lo, bits.hi);
  return negative ? "-" + result : result;
}
function uInt64ToString(lo, hi) {
  ({ lo, hi } = toUnsigned(lo, hi));
  if (hi <= 2097151) {
    return String(TWO_PWR_32_DBL * hi + lo);
  }
  const low = lo & 16777215;
  const mid = (lo >>> 24 | hi << 8) & 16777215;
  const high = hi >> 16 & 65535;
  let digitA = low + mid * 6777216 + high * 6710656;
  let digitB = mid + high * 8147497;
  let digitC = high * 2;
  const base = 1e7;
  if (digitA >= base) {
    digitB += Math.floor(digitA / base);
    digitA %= base;
  }
  if (digitB >= base) {
    digitC += Math.floor(digitB / base);
    digitB %= base;
  }
  return digitC.toString() + decimalFrom1e7WithLeadingZeros(digitB) + decimalFrom1e7WithLeadingZeros(digitA);
}
function toUnsigned(lo, hi) {
  return { lo: lo >>> 0, hi: hi >>> 0 };
}
function newBits(lo, hi) {
  return { lo: lo | 0, hi: hi | 0 };
}
function negate(lowBits, highBits) {
  highBits = ~highBits;
  if (lowBits) {
    lowBits = ~lowBits + 1;
  } else {
    highBits += 1;
  }
  return newBits(lowBits, highBits);
}
var decimalFrom1e7WithLeadingZeros = (digit1e7) => {
  const partial = String(digit1e7);
  return "0000000".slice(partial.length) + partial;
};
function varint32read() {
  let b = this.buf[this.pos++];
  let result = b & 127;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 7;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 14;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 21;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 15) << 28;
  for (let readBytes = 5; (b & 128) !== 0 && readBytes < 10; readBytes++)
    b = this.buf[this.pos++];
  if ((b & 128) != 0) throw new Error("invalid varint");
  this.assertBounds();
  return result >>> 0;
}
function zzEncode(lo, hi) {
  let mask = hi >> 31;
  hi = ((hi << 1 | lo >>> 31) ^ mask) >>> 0;
  lo = (lo << 1 ^ mask) >>> 0;
  return [lo, hi];
}
function zzDecode(lo, hi) {
  let mask = -(lo & 1);
  lo = ((lo >>> 1 | hi << 31) ^ mask) >>> 0;
  hi = (hi >>> 1 ^ mask) >>> 0;
  return [lo, hi];
}
function readUInt32(buf, pos) {
  return (buf[pos] | buf[pos + 1] << 8 | buf[pos + 2] << 16) + buf[pos + 3] * 16777216;
}
function readInt32(buf, pos) {
  return (buf[pos] | buf[pos + 1] << 8 | buf[pos + 2] << 16) + (buf[pos + 3] << 24);
}
function writeVarint32(val, buf, pos) {
  while (val > 127) {
    buf[pos++] = val & 127 | 128;
    val >>>= 7;
  }
  buf[pos] = val;
}
function writeVarint64(val, buf, pos) {
  while (val.hi) {
    buf[pos++] = val.lo & 127 | 128;
    val.lo = (val.lo >>> 7 | val.hi << 25) >>> 0;
    val.hi >>>= 7;
  }
  while (val.lo > 127) {
    buf[pos++] = val.lo & 127 | 128;
    val.lo = val.lo >>> 7;
  }
  buf[pos++] = val.lo;
}
function int64Length(lo, hi) {
  let part0 = lo, part1 = (lo >>> 28 | hi << 4) >>> 0, part2 = hi >>> 24;
  return part2 === 0 ? part1 === 0 ? part0 < 16384 ? part0 < 128 ? 1 : 2 : part0 < 2097152 ? 3 : 4 : part1 < 16384 ? part1 < 128 ? 5 : 6 : part1 < 2097152 ? 7 : 8 : part2 < 128 ? 9 : 10;
}
function writeFixed32(val, buf, pos) {
  buf[pos] = val & 255;
  buf[pos + 1] = val >>> 8 & 255;
  buf[pos + 2] = val >>> 16 & 255;
  buf[pos + 3] = val >>> 24;
}
function writeByte(val, buf, pos) {
  buf[pos] = val & 255;
}

// src/generated/binary.ts
var BinaryReader = class {
  buf;
  pos;
  type;
  len;
  assertBounds() {
    if (this.pos > this.len) throw new RangeError("premature EOF");
  }
  constructor(buf) {
    this.buf = buf ? new Uint8Array(buf) : new Uint8Array(0);
    this.pos = 0;
    this.type = 0;
    this.len = this.buf.length;
  }
  tag() {
    const tag = this.uint32(), fieldNo = tag >>> 3, wireType = tag & 7;
    if (fieldNo <= 0 || wireType < 0 || wireType > 5)
      throw new Error(
        "illegal tag: field no " + fieldNo + " wire type " + wireType
      );
    return [fieldNo, wireType, tag];
  }
  skip(length) {
    if (typeof length === "number") {
      if (this.pos + length > this.len) throw indexOutOfRange(this, length);
      this.pos += length;
    } else {
      do {
        if (this.pos >= this.len) throw indexOutOfRange(this);
      } while (this.buf[this.pos++] & 128);
    }
    return this;
  }
  skipType(wireType) {
    switch (wireType) {
      case 0 /* Varint */:
        this.skip();
        break;
      case 1 /* Fixed64 */:
        this.skip(8);
        break;
      case 2 /* Bytes */:
        this.skip(this.uint32());
        break;
      case 3:
        while ((wireType = this.uint32() & 7) !== 4) {
          this.skipType(wireType);
        }
        break;
      case 5 /* Fixed32 */:
        this.skip(4);
        break;
      /* istanbul ignore next */
      default:
        throw Error("invalid wire type " + wireType + " at offset " + this.pos);
    }
    return this;
  }
  uint32() {
    return varint32read.bind(this)();
  }
  int32() {
    return this.uint32() | 0;
  }
  sint32() {
    const num = this.uint32();
    return num % 2 === 1 ? (num + 1) / -2 : num / 2;
  }
  fixed32() {
    const val = readUInt32(this.buf, this.pos);
    this.pos += 4;
    return val;
  }
  sfixed32() {
    const val = readInt32(this.buf, this.pos);
    this.pos += 4;
    return val;
  }
  int64() {
    const [lo, hi] = varint64read.bind(this)();
    return BigInt(int64ToString(lo, hi));
  }
  uint64() {
    const [lo, hi] = varint64read.bind(this)();
    return BigInt(uInt64ToString(lo, hi));
  }
  sint64() {
    let [lo, hi] = varint64read.bind(this)();
    [lo, hi] = zzDecode(lo, hi);
    return BigInt(int64ToString(lo, hi));
  }
  fixed64() {
    const lo = this.sfixed32();
    const hi = this.sfixed32();
    return BigInt(uInt64ToString(lo, hi));
  }
  sfixed64() {
    const lo = this.sfixed32();
    const hi = this.sfixed32();
    return BigInt(int64ToString(lo, hi));
  }
  float() {
    throw new Error("float not supported");
  }
  double() {
    throw new Error("double not supported");
  }
  bool() {
    const [lo, hi] = varint64read.bind(this)();
    return lo !== 0 || hi !== 0;
  }
  bytes() {
    const len = this.uint32(), start = this.pos;
    this.pos += len;
    this.assertBounds();
    return this.buf.subarray(start, start + len);
  }
  string() {
    const bytes = this.bytes();
    return utf8Read(bytes, 0, bytes.length);
  }
};
var Op = class {
  fn;
  len;
  val;
  next;
  constructor(fn, len, val) {
    this.fn = fn;
    this.len = len;
    this.val = val;
  }
  proceed(buf, pos) {
    if (this.fn) {
      this.fn(this.val, buf, pos);
    }
  }
};
var State = class {
  head;
  tail;
  len;
  next;
  constructor(writer) {
    this.head = writer.head;
    this.tail = writer.tail;
    this.len = writer.len;
    this.next = writer.states;
  }
};
var BinaryWriter = class _BinaryWriter {
  len = 0;
  head;
  tail;
  states;
  constructor() {
    this.head = new Op(null, 0, 0);
    this.tail = this.head;
    this.states = null;
  }
  static create() {
    return new _BinaryWriter();
  }
  static alloc(size) {
    if (typeof Uint8Array !== "undefined") {
      return pool(
        (size2) => new Uint8Array(size2),
        Uint8Array.prototype.subarray
      )(size);
    } else {
      return new Array(size);
    }
  }
  _push(fn, len, val) {
    this.tail = this.tail.next = new Op(fn, len, val);
    this.len += len;
    return this;
  }
  finish() {
    let head = this.head.next, pos = 0;
    const buf = _BinaryWriter.alloc(this.len);
    while (head) {
      head.proceed(buf, pos);
      pos += head.len;
      head = head.next;
    }
    return buf;
  }
  fork() {
    this.states = new State(this);
    this.head = this.tail = new Op(null, 0, 0);
    this.len = 0;
    return this;
  }
  reset() {
    if (this.states) {
      this.head = this.states.head;
      this.tail = this.states.tail;
      this.len = this.states.len;
      this.states = this.states.next;
    } else {
      this.head = this.tail = new Op(null, 0, 0);
      this.len = 0;
    }
    return this;
  }
  ldelim() {
    const head = this.head, tail = this.tail, len = this.len;
    this.reset().uint32(len);
    if (len) {
      this.tail.next = head.next;
      this.tail = tail;
      this.len += len;
    }
    return this;
  }
  tag(fieldNo, type) {
    return this.uint32((fieldNo << 3 | type) >>> 0);
  }
  uint32(value) {
    this.len += (this.tail = this.tail.next = new Op(
      writeVarint32,
      (value = value >>> 0) < 128 ? 1 : value < 16384 ? 2 : value < 2097152 ? 3 : value < 268435456 ? 4 : 5,
      value
    )).len;
    return this;
  }
  int32(value) {
    return value < 0 ? this._push(writeVarint64, 10, int64FromString(value.toString())) : this.uint32(value);
  }
  sint32(value) {
    return this.uint32((value << 1 ^ value >> 31) >>> 0);
  }
  int64(value) {
    const { lo, hi } = int64FromString(value.toString());
    return this._push(writeVarint64, int64Length(lo, hi), { lo, hi });
  }
  // uint64 is the same with int64
  uint64 = _BinaryWriter.prototype.int64;
  sint64(value) {
    let { lo, hi } = int64FromString(value.toString());
    [lo, hi] = zzEncode(lo, hi);
    return this._push(writeVarint64, int64Length(lo, hi), { lo, hi });
  }
  fixed64(value) {
    const { lo, hi } = int64FromString(value.toString());
    return this._push(writeFixed32, 4, lo)._push(writeFixed32, 4, hi);
  }
  // sfixed64 is the same with fixed64
  sfixed64 = _BinaryWriter.prototype.fixed64;
  bool(value) {
    return this._push(writeByte, 1, value ? 1 : 0);
  }
  fixed32(value) {
    return this._push(writeFixed32, 4, value >>> 0);
  }
  // sfixed32 is the same with fixed32
  sfixed32 = _BinaryWriter.prototype.fixed32;
  float(value) {
    throw new Error("float not supported" + value);
  }
  double(value) {
    throw new Error("double not supported" + value);
  }
  bytes(value) {
    const len = value.length >>> 0;
    if (!len) return this._push(writeByte, 1, 0);
    return this.uint32(len)._push(writeBytes, len, value);
  }
  string(value) {
    const len = utf8Length(value);
    return len ? this.uint32(len)._push(utf8Write, len, value) : this._push(writeByte, 1, 0);
  }
};
function writeBytes(val, buf, pos) {
  if (typeof Uint8Array !== "undefined") {
    buf.set(val, pos);
  } else {
    for (let i = 0; i < val.length; ++i) buf[pos + i] = val[i];
  }
}
function pool(alloc, slice, size) {
  const SIZE = size || 8192;
  const MAX = SIZE >>> 1;
  let slab = null;
  let offset = SIZE;
  return function pool_alloc(size2) {
    if (size2 < 1 || size2 > MAX) return alloc(size2);
    if (offset + size2 > SIZE) {
      slab = alloc(SIZE);
      offset = 0;
    }
    const buf = slice.call(slab, offset, offset += size2);
    if (offset & 7)
      offset = (offset | 7) + 1;
    return buf;
  };
}
function indexOutOfRange(reader, writeLength) {
  return RangeError(
    "index out of range: " + reader.pos + " + " + (writeLength || 1) + " > " + reader.len
  );
}

// src/generated/zerone/liquiditypool/v1/genesis.ts
function createBaseParams() {
  return {
    defaultSwapFeeBps: BigInt(0),
    maxPools: BigInt(0),
    minInitialLiquidity: "",
    twapWindowBlocks: BigInt(0),
    protocolFeeBps: BigInt(0),
    minReserve: "",
    billingQuoteDenoms: [],
    allowedPoolDenoms: [],
    poolCreators: []
  };
}
var Params = {
  typeUrl: "/zerone.liquiditypool.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.defaultSwapFeeBps !== BigInt(0)) {
      writer.uint32(8).uint64(message.defaultSwapFeeBps);
    }
    if (message.maxPools !== BigInt(0)) {
      writer.uint32(16).uint64(message.maxPools);
    }
    if (message.minInitialLiquidity !== "") {
      writer.uint32(26).string(message.minInitialLiquidity);
    }
    if (message.twapWindowBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.twapWindowBlocks);
    }
    if (message.protocolFeeBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.protocolFeeBps);
    }
    if (message.minReserve !== "") {
      writer.uint32(50).string(message.minReserve);
    }
    for (const v of message.billingQuoteDenoms) {
      writer.uint32(58).string(v);
    }
    for (const v of message.allowedPoolDenoms) {
      writer.uint32(66).string(v);
    }
    for (const v of message.poolCreators) {
      writer.uint32(74).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.defaultSwapFeeBps = reader.uint64();
          break;
        case 2:
          message.maxPools = reader.uint64();
          break;
        case 3:
          message.minInitialLiquidity = reader.string();
          break;
        case 4:
          message.twapWindowBlocks = reader.uint64();
          break;
        case 5:
          message.protocolFeeBps = reader.uint64();
          break;
        case 6:
          message.minReserve = reader.string();
          break;
        case 7:
          message.billingQuoteDenoms.push(reader.string());
          break;
        case 8:
          message.allowedPoolDenoms.push(reader.string());
          break;
        case 9:
          message.poolCreators.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams();
    message.defaultSwapFeeBps = object.defaultSwapFeeBps !== void 0 && object.defaultSwapFeeBps !== null ? BigInt(object.defaultSwapFeeBps.toString()) : BigInt(0);
    message.maxPools = object.maxPools !== void 0 && object.maxPools !== null ? BigInt(object.maxPools.toString()) : BigInt(0);
    message.minInitialLiquidity = object.minInitialLiquidity ?? "";
    message.twapWindowBlocks = object.twapWindowBlocks !== void 0 && object.twapWindowBlocks !== null ? BigInt(object.twapWindowBlocks.toString()) : BigInt(0);
    message.protocolFeeBps = object.protocolFeeBps !== void 0 && object.protocolFeeBps !== null ? BigInt(object.protocolFeeBps.toString()) : BigInt(0);
    message.minReserve = object.minReserve ?? "";
    message.billingQuoteDenoms = object.billingQuoteDenoms?.map((e) => e) || [];
    message.allowedPoolDenoms = object.allowedPoolDenoms?.map((e) => e) || [];
    message.poolCreators = object.poolCreators?.map((e) => e) || [];
    return message;
  }
};

// src/generated/zerone/liquiditypool/v1/tx.ts
function createBaseMsgCreatePool() {
  return {
    creator: "",
    denomA: "",
    denomB: "",
    amountA: "",
    amountB: "",
    swapFeeBps: BigInt(0)
  };
}
var MsgCreatePool = {
  typeUrl: "/zerone.liquiditypool.v1.MsgCreatePool",
  encode(message, writer = BinaryWriter.create()) {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.denomA !== "") {
      writer.uint32(18).string(message.denomA);
    }
    if (message.denomB !== "") {
      writer.uint32(26).string(message.denomB);
    }
    if (message.amountA !== "") {
      writer.uint32(34).string(message.amountA);
    }
    if (message.amountB !== "") {
      writer.uint32(42).string(message.amountB);
    }
    if (message.swapFeeBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.swapFeeBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreatePool();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.denomA = reader.string();
          break;
        case 3:
          message.denomB = reader.string();
          break;
        case 4:
          message.amountA = reader.string();
          break;
        case 5:
          message.amountB = reader.string();
          break;
        case 6:
          message.swapFeeBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreatePool();
    message.creator = object.creator ?? "";
    message.denomA = object.denomA ?? "";
    message.denomB = object.denomB ?? "";
    message.amountA = object.amountA ?? "";
    message.amountB = object.amountB ?? "";
    message.swapFeeBps = object.swapFeeBps !== void 0 && object.swapFeeBps !== null ? BigInt(object.swapFeeBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgCreatePoolResponse() {
  return {
    poolId: ""
  };
}
var MsgCreatePoolResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgCreatePoolResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.poolId !== "") {
      writer.uint32(10).string(message.poolId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreatePoolResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.poolId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreatePoolResponse();
    message.poolId = object.poolId ?? "";
    return message;
  }
};
function createBaseMsgSwap() {
  return {
    sender: "",
    poolId: "",
    tokenInDenom: "",
    tokenInAmount: "",
    minTokenOut: ""
  };
}
var MsgSwap = {
  typeUrl: "/zerone.liquiditypool.v1.MsgSwap",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.poolId !== "") {
      writer.uint32(18).string(message.poolId);
    }
    if (message.tokenInDenom !== "") {
      writer.uint32(26).string(message.tokenInDenom);
    }
    if (message.tokenInAmount !== "") {
      writer.uint32(34).string(message.tokenInAmount);
    }
    if (message.minTokenOut !== "") {
      writer.uint32(42).string(message.minTokenOut);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSwap();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.poolId = reader.string();
          break;
        case 3:
          message.tokenInDenom = reader.string();
          break;
        case 4:
          message.tokenInAmount = reader.string();
          break;
        case 5:
          message.minTokenOut = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSwap();
    message.sender = object.sender ?? "";
    message.poolId = object.poolId ?? "";
    message.tokenInDenom = object.tokenInDenom ?? "";
    message.tokenInAmount = object.tokenInAmount ?? "";
    message.minTokenOut = object.minTokenOut ?? "";
    return message;
  }
};
function createBaseMsgSwapResponse() {
  return {
    tokenOutAmount: "",
    feeAmount: ""
  };
}
var MsgSwapResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgSwapResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.tokenOutAmount !== "") {
      writer.uint32(10).string(message.tokenOutAmount);
    }
    if (message.feeAmount !== "") {
      writer.uint32(18).string(message.feeAmount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSwapResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenOutAmount = reader.string();
          break;
        case 2:
          message.feeAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSwapResponse();
    message.tokenOutAmount = object.tokenOutAmount ?? "";
    message.feeAmount = object.feeAmount ?? "";
    return message;
  }
};
function createBaseMsgAddLiquidity() {
  return {
    sender: "",
    poolId: "",
    amountA: "",
    amountB: "",
    minLpTokens: ""
  };
}
var MsgAddLiquidity = {
  typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidity",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.poolId !== "") {
      writer.uint32(18).string(message.poolId);
    }
    if (message.amountA !== "") {
      writer.uint32(26).string(message.amountA);
    }
    if (message.amountB !== "") {
      writer.uint32(34).string(message.amountB);
    }
    if (message.minLpTokens !== "") {
      writer.uint32(42).string(message.minLpTokens);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddLiquidity();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.poolId = reader.string();
          break;
        case 3:
          message.amountA = reader.string();
          break;
        case 4:
          message.amountB = reader.string();
          break;
        case 5:
          message.minLpTokens = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddLiquidity();
    message.sender = object.sender ?? "";
    message.poolId = object.poolId ?? "";
    message.amountA = object.amountA ?? "";
    message.amountB = object.amountB ?? "";
    message.minLpTokens = object.minLpTokens ?? "";
    return message;
  }
};
function createBaseMsgAddLiquidityResponse() {
  return {
    lpTokensMinted: "",
    actualA: "",
    actualB: ""
  };
}
var MsgAddLiquidityResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidityResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.lpTokensMinted !== "") {
      writer.uint32(10).string(message.lpTokensMinted);
    }
    if (message.actualA !== "") {
      writer.uint32(18).string(message.actualA);
    }
    if (message.actualB !== "") {
      writer.uint32(26).string(message.actualB);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddLiquidityResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.lpTokensMinted = reader.string();
          break;
        case 2:
          message.actualA = reader.string();
          break;
        case 3:
          message.actualB = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddLiquidityResponse();
    message.lpTokensMinted = object.lpTokensMinted ?? "";
    message.actualA = object.actualA ?? "";
    message.actualB = object.actualB ?? "";
    return message;
  }
};
function createBaseMsgRemoveLiquidity() {
  return {
    sender: "",
    poolId: "",
    lpTokens: "",
    minAmountA: "",
    minAmountB: ""
  };
}
var MsgRemoveLiquidity = {
  typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidity",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.poolId !== "") {
      writer.uint32(18).string(message.poolId);
    }
    if (message.lpTokens !== "") {
      writer.uint32(26).string(message.lpTokens);
    }
    if (message.minAmountA !== "") {
      writer.uint32(34).string(message.minAmountA);
    }
    if (message.minAmountB !== "") {
      writer.uint32(42).string(message.minAmountB);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveLiquidity();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.poolId = reader.string();
          break;
        case 3:
          message.lpTokens = reader.string();
          break;
        case 4:
          message.minAmountA = reader.string();
          break;
        case 5:
          message.minAmountB = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRemoveLiquidity();
    message.sender = object.sender ?? "";
    message.poolId = object.poolId ?? "";
    message.lpTokens = object.lpTokens ?? "";
    message.minAmountA = object.minAmountA ?? "";
    message.minAmountB = object.minAmountB ?? "";
    return message;
  }
};
function createBaseMsgRemoveLiquidityResponse() {
  return {
    amountA: "",
    amountB: ""
  };
}
var MsgRemoveLiquidityResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidityResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.amountA !== "") {
      writer.uint32(10).string(message.amountA);
    }
    if (message.amountB !== "") {
      writer.uint32(18).string(message.amountB);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveLiquidityResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.amountA = reader.string();
          break;
        case 2:
          message.amountB = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRemoveLiquidityResponse();
    message.amountA = object.amountA ?? "";
    message.amountB = object.amountB ?? "";
    return message;
  }
};
function createBaseMsgUpdateParams() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams = {
  typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse() {
  return {};
}
var MsgUpdateParamsResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_) {
    const message = createBaseMsgUpdateParamsResponse();
    return message;
  }
};
function createBaseMsgSetPoolStatus() {
  return {
    authority: "",
    poolId: "",
    status: 0
  };
}
var MsgSetPoolStatus = {
  typeUrl: "/zerone.liquiditypool.v1.MsgSetPoolStatus",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.poolId !== "") {
      writer.uint32(18).string(message.poolId);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSetPoolStatus();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.poolId = reader.string();
          break;
        case 3:
          message.status = reader.int32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSetPoolStatus();
    message.authority = object.authority ?? "";
    message.poolId = object.poolId ?? "";
    message.status = object.status ?? 0;
    return message;
  }
};
function createBaseMsgSetPoolStatusResponse() {
  return {};
}
var MsgSetPoolStatusResponse = {
  typeUrl: "/zerone.liquiditypool.v1.MsgSetPoolStatusResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSetPoolStatusResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_) {
    const message = createBaseMsgSetPoolStatusResponse();
    return message;
  }
};

export {
  BinaryReader,
  BinaryWriter,
  MsgCreatePool,
  MsgSwap,
  MsgAddLiquidity,
  MsgRemoveLiquidity,
  MsgUpdateParams,
  MsgSetPoolStatus,
  tx_exports
};
