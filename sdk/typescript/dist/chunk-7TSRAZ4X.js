import {
  __export
} from "./chunk-MLKGABMK.js";

// src/generated/zerone/alignment/v1/tx.ts
var tx_exports = {};
__export(tx_exports, {
  MsgActivate: () => MsgActivate,
  MsgActivateResponse: () => MsgActivateResponse,
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

// src/generated/zerone/alignment/v1/genesis.ts
function createBaseParams() {
  return {
    observationIntervalBlocks: BigInt(0),
    weightKnowledgeQuality: BigInt(0),
    weightEconomicStability: BigInt(0),
    weightGovernanceParticipation: BigInt(0),
    weightNetworkSecurity: BigInt(0),
    weightStakingRatio: BigInt(0),
    criticalThreshold: BigInt(0),
    degradedThreshold: BigInt(0),
    healthyThreshold: BigInt(0),
    enabled: false,
    maxAutoApplyMagnitudeBps: BigInt(0),
    correctionConfidenceWindowSize: BigInt(0),
    correctionConfidenceMinSamples: BigInt(0),
    minConfidenceForAutoApply: BigInt(0),
    correctionBoundsMinMultiplierBps: BigInt(0),
    correctionBoundsMaxMultiplierBps: BigInt(0),
    advisoryMagnitudeBps: BigInt(0)
  };
}
var Params = {
  typeUrl: "/zerone.alignment.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.observationIntervalBlocks !== BigInt(0)) {
      writer.uint32(8).uint64(message.observationIntervalBlocks);
    }
    if (message.weightKnowledgeQuality !== BigInt(0)) {
      writer.uint32(16).uint64(message.weightKnowledgeQuality);
    }
    if (message.weightEconomicStability !== BigInt(0)) {
      writer.uint32(24).uint64(message.weightEconomicStability);
    }
    if (message.weightGovernanceParticipation !== BigInt(0)) {
      writer.uint32(32).uint64(message.weightGovernanceParticipation);
    }
    if (message.weightNetworkSecurity !== BigInt(0)) {
      writer.uint32(40).uint64(message.weightNetworkSecurity);
    }
    if (message.weightStakingRatio !== BigInt(0)) {
      writer.uint32(48).uint64(message.weightStakingRatio);
    }
    if (message.criticalThreshold !== BigInt(0)) {
      writer.uint32(56).uint64(message.criticalThreshold);
    }
    if (message.degradedThreshold !== BigInt(0)) {
      writer.uint32(64).uint64(message.degradedThreshold);
    }
    if (message.healthyThreshold !== BigInt(0)) {
      writer.uint32(72).uint64(message.healthyThreshold);
    }
    if (message.enabled === true) {
      writer.uint32(80).bool(message.enabled);
    }
    if (message.maxAutoApplyMagnitudeBps !== BigInt(0)) {
      writer.uint32(88).uint64(message.maxAutoApplyMagnitudeBps);
    }
    if (message.correctionConfidenceWindowSize !== BigInt(0)) {
      writer.uint32(96).uint64(message.correctionConfidenceWindowSize);
    }
    if (message.correctionConfidenceMinSamples !== BigInt(0)) {
      writer.uint32(104).uint64(message.correctionConfidenceMinSamples);
    }
    if (message.minConfidenceForAutoApply !== BigInt(0)) {
      writer.uint32(112).uint64(message.minConfidenceForAutoApply);
    }
    if (message.correctionBoundsMinMultiplierBps !== BigInt(0)) {
      writer.uint32(120).uint64(message.correctionBoundsMinMultiplierBps);
    }
    if (message.correctionBoundsMaxMultiplierBps !== BigInt(0)) {
      writer.uint32(128).uint64(message.correctionBoundsMaxMultiplierBps);
    }
    if (message.advisoryMagnitudeBps !== BigInt(0)) {
      writer.uint32(136).uint64(message.advisoryMagnitudeBps);
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
          message.observationIntervalBlocks = reader.uint64();
          break;
        case 2:
          message.weightKnowledgeQuality = reader.uint64();
          break;
        case 3:
          message.weightEconomicStability = reader.uint64();
          break;
        case 4:
          message.weightGovernanceParticipation = reader.uint64();
          break;
        case 5:
          message.weightNetworkSecurity = reader.uint64();
          break;
        case 6:
          message.weightStakingRatio = reader.uint64();
          break;
        case 7:
          message.criticalThreshold = reader.uint64();
          break;
        case 8:
          message.degradedThreshold = reader.uint64();
          break;
        case 9:
          message.healthyThreshold = reader.uint64();
          break;
        case 10:
          message.enabled = reader.bool();
          break;
        case 11:
          message.maxAutoApplyMagnitudeBps = reader.uint64();
          break;
        case 12:
          message.correctionConfidenceWindowSize = reader.uint64();
          break;
        case 13:
          message.correctionConfidenceMinSamples = reader.uint64();
          break;
        case 14:
          message.minConfidenceForAutoApply = reader.uint64();
          break;
        case 15:
          message.correctionBoundsMinMultiplierBps = reader.uint64();
          break;
        case 16:
          message.correctionBoundsMaxMultiplierBps = reader.uint64();
          break;
        case 17:
          message.advisoryMagnitudeBps = reader.uint64();
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
    message.observationIntervalBlocks = object.observationIntervalBlocks !== void 0 && object.observationIntervalBlocks !== null ? BigInt(object.observationIntervalBlocks.toString()) : BigInt(0);
    message.weightKnowledgeQuality = object.weightKnowledgeQuality !== void 0 && object.weightKnowledgeQuality !== null ? BigInt(object.weightKnowledgeQuality.toString()) : BigInt(0);
    message.weightEconomicStability = object.weightEconomicStability !== void 0 && object.weightEconomicStability !== null ? BigInt(object.weightEconomicStability.toString()) : BigInt(0);
    message.weightGovernanceParticipation = object.weightGovernanceParticipation !== void 0 && object.weightGovernanceParticipation !== null ? BigInt(object.weightGovernanceParticipation.toString()) : BigInt(0);
    message.weightNetworkSecurity = object.weightNetworkSecurity !== void 0 && object.weightNetworkSecurity !== null ? BigInt(object.weightNetworkSecurity.toString()) : BigInt(0);
    message.weightStakingRatio = object.weightStakingRatio !== void 0 && object.weightStakingRatio !== null ? BigInt(object.weightStakingRatio.toString()) : BigInt(0);
    message.criticalThreshold = object.criticalThreshold !== void 0 && object.criticalThreshold !== null ? BigInt(object.criticalThreshold.toString()) : BigInt(0);
    message.degradedThreshold = object.degradedThreshold !== void 0 && object.degradedThreshold !== null ? BigInt(object.degradedThreshold.toString()) : BigInt(0);
    message.healthyThreshold = object.healthyThreshold !== void 0 && object.healthyThreshold !== null ? BigInt(object.healthyThreshold.toString()) : BigInt(0);
    message.enabled = object.enabled ?? false;
    message.maxAutoApplyMagnitudeBps = object.maxAutoApplyMagnitudeBps !== void 0 && object.maxAutoApplyMagnitudeBps !== null ? BigInt(object.maxAutoApplyMagnitudeBps.toString()) : BigInt(0);
    message.correctionConfidenceWindowSize = object.correctionConfidenceWindowSize !== void 0 && object.correctionConfidenceWindowSize !== null ? BigInt(object.correctionConfidenceWindowSize.toString()) : BigInt(0);
    message.correctionConfidenceMinSamples = object.correctionConfidenceMinSamples !== void 0 && object.correctionConfidenceMinSamples !== null ? BigInt(object.correctionConfidenceMinSamples.toString()) : BigInt(0);
    message.minConfidenceForAutoApply = object.minConfidenceForAutoApply !== void 0 && object.minConfidenceForAutoApply !== null ? BigInt(object.minConfidenceForAutoApply.toString()) : BigInt(0);
    message.correctionBoundsMinMultiplierBps = object.correctionBoundsMinMultiplierBps !== void 0 && object.correctionBoundsMinMultiplierBps !== null ? BigInt(object.correctionBoundsMinMultiplierBps.toString()) : BigInt(0);
    message.correctionBoundsMaxMultiplierBps = object.correctionBoundsMaxMultiplierBps !== void 0 && object.correctionBoundsMaxMultiplierBps !== null ? BigInt(object.correctionBoundsMaxMultiplierBps.toString()) : BigInt(0);
    message.advisoryMagnitudeBps = object.advisoryMagnitudeBps !== void 0 && object.advisoryMagnitudeBps !== null ? BigInt(object.advisoryMagnitudeBps.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/alignment/v1/tx.ts
function createBaseMsgUpdateParams() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams = {
  typeUrl: "/zerone.alignment.v1.MsgUpdateParams",
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
  typeUrl: "/zerone.alignment.v1.MsgUpdateParamsResponse",
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
function createBaseMsgActivate() {
  return {
    authority: "",
    enabled: false
  };
}
var MsgActivate = {
  typeUrl: "/zerone.alignment.v1.MsgActivate",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.enabled === true) {
      writer.uint32(16).bool(message.enabled);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgActivate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.enabled = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgActivate();
    message.authority = object.authority ?? "";
    message.enabled = object.enabled ?? false;
    return message;
  }
};
function createBaseMsgActivateResponse() {
  return {};
}
var MsgActivateResponse = {
  typeUrl: "/zerone.alignment.v1.MsgActivateResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgActivateResponse();
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
    const message = createBaseMsgActivateResponse();
    return message;
  }
};

// src/generated/zerone/alignment/v1/tx.registry.ts
var registry = [["/zerone.alignment.v1.MsgUpdateParams", MsgUpdateParams], ["/zerone.alignment.v1.MsgActivate", MsgActivate]];
var MessageComposer = {
  encoded: {
    updateParams(value) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgUpdateParams",
        value: MsgUpdateParams.encode(value).finish()
      };
    },
    activate(value) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgActivate",
        value: MsgActivate.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    updateParams(value) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgUpdateParams",
        value
      };
    },
    activate(value) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgActivate",
        value
      };
    }
  },
  fromPartial: {
    updateParams(value) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgUpdateParams",
        value: MsgUpdateParams.fromPartial(value)
      };
    },
    activate(value) {
      return {
        typeUrl: "/zerone.alignment.v1.MsgActivate",
        value: MsgActivate.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/auth/v1/tx.ts
var tx_exports2 = {};
__export(tx_exports2, {
  MsgFreezeAccount: () => MsgFreezeAccount,
  MsgFreezeAccountResponse: () => MsgFreezeAccountResponse,
  MsgRegisterAccount: () => MsgRegisterAccount,
  MsgRegisterAccountResponse: () => MsgRegisterAccountResponse,
  MsgRotateKey: () => MsgRotateKey,
  MsgRotateKeyResponse: () => MsgRotateKeyResponse,
  MsgUnfreezeAccount: () => MsgUnfreezeAccount,
  MsgUnfreezeAccountResponse: () => MsgUnfreezeAccountResponse,
  MsgUpdateParams: () => MsgUpdateParams2,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse2
});

// src/generated/zerone/auth/v1/genesis.ts
function createBaseParams2() {
  return {
    keyRotationCooldown: BigInt(0),
    maxMetadataLength: 0,
    requireDid: false
  };
}
var Params2 = {
  typeUrl: "/zerone.auth.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.keyRotationCooldown !== BigInt(0)) {
      writer.uint32(24).uint64(message.keyRotationCooldown);
    }
    if (message.maxMetadataLength !== 0) {
      writer.uint32(64).uint32(message.maxMetadataLength);
    }
    if (message.requireDid === true) {
      writer.uint32(72).bool(message.requireDid);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams2();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 3:
          message.keyRotationCooldown = reader.uint64();
          break;
        case 8:
          message.maxMetadataLength = reader.uint32();
          break;
        case 9:
          message.requireDid = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams2();
    message.keyRotationCooldown = object.keyRotationCooldown !== void 0 && object.keyRotationCooldown !== null ? BigInt(object.keyRotationCooldown.toString()) : BigInt(0);
    message.maxMetadataLength = object.maxMetadataLength ?? 0;
    message.requireDid = object.requireDid ?? false;
    return message;
  }
};

// src/generated/zerone/auth/v1/tx.ts
function createBaseMsgRegisterAccount() {
  return {
    sender: "",
    did: "",
    publicKey: "",
    accountType: "",
    operationalKeyHash: "",
    metadata: ""
  };
}
var MsgRegisterAccount = {
  typeUrl: "/zerone.auth.v1.MsgRegisterAccount",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.did !== "") {
      writer.uint32(18).string(message.did);
    }
    if (message.publicKey !== "") {
      writer.uint32(26).string(message.publicKey);
    }
    if (message.accountType !== "") {
      writer.uint32(34).string(message.accountType);
    }
    if (message.operationalKeyHash !== "") {
      writer.uint32(42).string(message.operationalKeyHash);
    }
    if (message.metadata !== "") {
      writer.uint32(50).string(message.metadata);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.did = reader.string();
          break;
        case 3:
          message.publicKey = reader.string();
          break;
        case 4:
          message.accountType = reader.string();
          break;
        case 5:
          message.operationalKeyHash = reader.string();
          break;
        case 6:
          message.metadata = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterAccount();
    message.sender = object.sender ?? "";
    message.did = object.did ?? "";
    message.publicKey = object.publicKey ?? "";
    message.accountType = object.accountType ?? "";
    message.operationalKeyHash = object.operationalKeyHash ?? "";
    message.metadata = object.metadata ?? "";
    return message;
  }
};
function createBaseMsgRegisterAccountResponse() {
  return {};
}
var MsgRegisterAccountResponse = {
  typeUrl: "/zerone.auth.v1.MsgRegisterAccountResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAccountResponse();
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
    const message = createBaseMsgRegisterAccountResponse();
    return message;
  }
};
function createBaseMsgRotateKey() {
  return {
    sender: "",
    newOperationalKey: new Uint8Array(),
    authorizationSignature: new Uint8Array()
  };
}
var MsgRotateKey = {
  typeUrl: "/zerone.auth.v1.MsgRotateKey",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.newOperationalKey.length !== 0) {
      writer.uint32(18).bytes(message.newOperationalKey);
    }
    if (message.authorizationSignature.length !== 0) {
      writer.uint32(26).bytes(message.authorizationSignature);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRotateKey();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.newOperationalKey = reader.bytes();
          break;
        case 3:
          message.authorizationSignature = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRotateKey();
    message.sender = object.sender ?? "";
    message.newOperationalKey = object.newOperationalKey ?? new Uint8Array();
    message.authorizationSignature = object.authorizationSignature ?? new Uint8Array();
    return message;
  }
};
function createBaseMsgRotateKeyResponse() {
  return {
    newKeyVersion: 0
  };
}
var MsgRotateKeyResponse = {
  typeUrl: "/zerone.auth.v1.MsgRotateKeyResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.newKeyVersion !== 0) {
      writer.uint32(8).uint32(message.newKeyVersion);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRotateKeyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newKeyVersion = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRotateKeyResponse();
    message.newKeyVersion = object.newKeyVersion ?? 0;
    return message;
  }
};
function createBaseMsgFreezeAccount() {
  return {
    sender: "",
    address: "",
    reason: ""
  };
}
var MsgFreezeAccount = {
  typeUrl: "/zerone.auth.v1.MsgFreezeAccount",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFreezeAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgFreezeAccount();
    message.sender = object.sender ?? "";
    message.address = object.address ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgFreezeAccountResponse() {
  return {};
}
var MsgFreezeAccountResponse = {
  typeUrl: "/zerone.auth.v1.MsgFreezeAccountResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFreezeAccountResponse();
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
    const message = createBaseMsgFreezeAccountResponse();
    return message;
  }
};
function createBaseMsgUnfreezeAccount() {
  return {
    authority: "",
    address: ""
  };
}
var MsgUnfreezeAccount = {
  typeUrl: "/zerone.auth.v1.MsgUnfreezeAccount",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUnfreezeAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUnfreezeAccount();
    message.authority = object.authority ?? "";
    message.address = object.address ?? "";
    return message;
  }
};
function createBaseMsgUnfreezeAccountResponse() {
  return {};
}
var MsgUnfreezeAccountResponse = {
  typeUrl: "/zerone.auth.v1.MsgUnfreezeAccountResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUnfreezeAccountResponse();
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
    const message = createBaseMsgUnfreezeAccountResponse();
    return message;
  }
};
function createBaseMsgUpdateParams2() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams2 = {
  typeUrl: "/zerone.auth.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params2.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams2();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params2.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams2();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params2.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse2() {
  return {};
}
var MsgUpdateParamsResponse2 = {
  typeUrl: "/zerone.auth.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse2();
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
    const message = createBaseMsgUpdateParamsResponse2();
    return message;
  }
};

// src/generated/zerone/auth/v1/tx.registry.ts
var registry2 = [["/zerone.auth.v1.MsgRegisterAccount", MsgRegisterAccount], ["/zerone.auth.v1.MsgRotateKey", MsgRotateKey], ["/zerone.auth.v1.MsgFreezeAccount", MsgFreezeAccount], ["/zerone.auth.v1.MsgUnfreezeAccount", MsgUnfreezeAccount], ["/zerone.auth.v1.MsgUpdateParams", MsgUpdateParams2]];
var MessageComposer2 = {
  encoded: {
    registerAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRegisterAccount",
        value: MsgRegisterAccount.encode(value).finish()
      };
    },
    rotateKey(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRotateKey",
        value: MsgRotateKey.encode(value).finish()
      };
    },
    freezeAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgFreezeAccount",
        value: MsgFreezeAccount.encode(value).finish()
      };
    },
    unfreezeAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUnfreezeAccount",
        value: MsgUnfreezeAccount.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUpdateParams",
        value: MsgUpdateParams2.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    registerAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRegisterAccount",
        value
      };
    },
    rotateKey(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRotateKey",
        value
      };
    },
    freezeAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgFreezeAccount",
        value
      };
    },
    unfreezeAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUnfreezeAccount",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    registerAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRegisterAccount",
        value: MsgRegisterAccount.fromPartial(value)
      };
    },
    rotateKey(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgRotateKey",
        value: MsgRotateKey.fromPartial(value)
      };
    },
    freezeAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgFreezeAccount",
        value: MsgFreezeAccount.fromPartial(value)
      };
    },
    unfreezeAccount(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUnfreezeAccount",
        value: MsgUnfreezeAccount.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.auth.v1.MsgUpdateParams",
        value: MsgUpdateParams2.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/capture_challenge/v1/tx.ts
var tx_exports3 = {};
__export(tx_exports3, {
  MsgAddEvidence: () => MsgAddEvidence,
  MsgAddEvidenceResponse: () => MsgAddEvidenceResponse,
  MsgFundBountyPool: () => MsgFundBountyPool,
  MsgFundBountyPoolResponse: () => MsgFundBountyPoolResponse,
  MsgResolveChallenge: () => MsgResolveChallenge,
  MsgResolveChallengeResponse: () => MsgResolveChallengeResponse,
  MsgSubmitChallenge: () => MsgSubmitChallenge,
  MsgSubmitChallengeResponse: () => MsgSubmitChallengeResponse,
  MsgUpdateParams: () => MsgUpdateParams3,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse3
});

// src/generated/zerone/capture_challenge/v1/genesis.ts
function createBaseParams3() {
  return {
    minChallengeStake: "",
    evidencePeriodBlocks: BigInt(0),
    reviewPeriodBlocks: BigInt(0),
    domainPauseBlocks: BigInt(0),
    rewardRateBps: BigInt(0),
    slashRateBps: BigInt(0),
    bountyContributionPerFact: "",
    riskAnalysisInterval: BigInt(0)
  };
}
var Params3 = {
  typeUrl: "/zerone.capture_challenge.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.minChallengeStake !== "") {
      writer.uint32(10).string(message.minChallengeStake);
    }
    if (message.evidencePeriodBlocks !== BigInt(0)) {
      writer.uint32(16).uint64(message.evidencePeriodBlocks);
    }
    if (message.reviewPeriodBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.reviewPeriodBlocks);
    }
    if (message.domainPauseBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.domainPauseBlocks);
    }
    if (message.rewardRateBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.rewardRateBps);
    }
    if (message.slashRateBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.slashRateBps);
    }
    if (message.bountyContributionPerFact !== "") {
      writer.uint32(58).string(message.bountyContributionPerFact);
    }
    if (message.riskAnalysisInterval !== BigInt(0)) {
      writer.uint32(64).uint64(message.riskAnalysisInterval);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams3();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minChallengeStake = reader.string();
          break;
        case 2:
          message.evidencePeriodBlocks = reader.uint64();
          break;
        case 3:
          message.reviewPeriodBlocks = reader.uint64();
          break;
        case 4:
          message.domainPauseBlocks = reader.uint64();
          break;
        case 5:
          message.rewardRateBps = reader.uint64();
          break;
        case 6:
          message.slashRateBps = reader.uint64();
          break;
        case 7:
          message.bountyContributionPerFact = reader.string();
          break;
        case 8:
          message.riskAnalysisInterval = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams3();
    message.minChallengeStake = object.minChallengeStake ?? "";
    message.evidencePeriodBlocks = object.evidencePeriodBlocks !== void 0 && object.evidencePeriodBlocks !== null ? BigInt(object.evidencePeriodBlocks.toString()) : BigInt(0);
    message.reviewPeriodBlocks = object.reviewPeriodBlocks !== void 0 && object.reviewPeriodBlocks !== null ? BigInt(object.reviewPeriodBlocks.toString()) : BigInt(0);
    message.domainPauseBlocks = object.domainPauseBlocks !== void 0 && object.domainPauseBlocks !== null ? BigInt(object.domainPauseBlocks.toString()) : BigInt(0);
    message.rewardRateBps = object.rewardRateBps !== void 0 && object.rewardRateBps !== null ? BigInt(object.rewardRateBps.toString()) : BigInt(0);
    message.slashRateBps = object.slashRateBps !== void 0 && object.slashRateBps !== null ? BigInt(object.slashRateBps.toString()) : BigInt(0);
    message.bountyContributionPerFact = object.bountyContributionPerFact ?? "";
    message.riskAnalysisInterval = object.riskAnalysisInterval !== void 0 && object.riskAnalysisInterval !== null ? BigInt(object.riskAnalysisInterval.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/capture_challenge/v1/tx.ts
function createBaseMsgSubmitChallenge() {
  return {
    challenger: "",
    domain: "",
    accusedValidators: [],
    stake: "",
    reason: ""
  };
}
var MsgSubmitChallenge = {
  typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallenge",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    for (const v of message.accusedValidators) {
      writer.uint32(26).string(v);
    }
    if (message.stake !== "") {
      writer.uint32(34).string(message.stake);
    }
    if (message.reason !== "") {
      writer.uint32(42).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.accusedValidators.push(reader.string());
          break;
        case 4:
          message.stake = reader.string();
          break;
        case 5:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitChallenge();
    message.challenger = object.challenger ?? "";
    message.domain = object.domain ?? "";
    message.accusedValidators = object.accusedValidators?.map((e) => e) || [];
    message.stake = object.stake ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgSubmitChallengeResponse() {
  return {
    challengeId: ""
  };
}
var MsgSubmitChallengeResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallengeResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challengeId !== "") {
      writer.uint32(10).string(message.challengeId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitChallengeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challengeId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitChallengeResponse();
    message.challengeId = object.challengeId ?? "";
    return message;
  }
};
function createBaseMsgAddEvidence() {
  return {
    challenger: "",
    challengeId: "",
    description: "",
    dataHash: ""
  };
}
var MsgAddEvidence = {
  typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidence",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.challengeId !== "") {
      writer.uint32(18).string(message.challengeId);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.dataHash !== "") {
      writer.uint32(34).string(message.dataHash);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddEvidence();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.challengeId = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.dataHash = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddEvidence();
    message.challenger = object.challenger ?? "";
    message.challengeId = object.challengeId ?? "";
    message.description = object.description ?? "";
    message.dataHash = object.dataHash ?? "";
    return message;
  }
};
function createBaseMsgAddEvidenceResponse() {
  return {};
}
var MsgAddEvidenceResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidenceResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddEvidenceResponse();
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
    const message = createBaseMsgAddEvidenceResponse();
    return message;
  }
};
function createBaseMsgResolveChallenge() {
  return {
    authority: "",
    challengeId: "",
    outcome: 0,
    reason: ""
  };
}
var MsgResolveChallenge = {
  typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallenge",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.challengeId !== "") {
      writer.uint32(18).string(message.challengeId);
    }
    if (message.outcome !== 0) {
      writer.uint32(24).int32(message.outcome);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.challengeId = reader.string();
          break;
        case 3:
          message.outcome = reader.int32();
          break;
        case 4:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgResolveChallenge();
    message.authority = object.authority ?? "";
    message.challengeId = object.challengeId ?? "";
    message.outcome = object.outcome ?? 0;
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgResolveChallengeResponse() {
  return {};
}
var MsgResolveChallengeResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallengeResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveChallengeResponse();
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
    const message = createBaseMsgResolveChallengeResponse();
    return message;
  }
};
function createBaseMsgFundBountyPool() {
  return {
    sender: "",
    domain: "",
    amount: ""
  };
}
var MsgFundBountyPool = {
  typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPool",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFundBountyPool();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgFundBountyPool();
    message.sender = object.sender ?? "";
    message.domain = object.domain ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgFundBountyPoolResponse() {
  return {};
}
var MsgFundBountyPoolResponse = {
  typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPoolResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFundBountyPoolResponse();
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
    const message = createBaseMsgFundBountyPoolResponse();
    return message;
  }
};
function createBaseMsgUpdateParams3() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams3 = {
  typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params3.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams3();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params3.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams3();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params3.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse3() {
  return {};
}
var MsgUpdateParamsResponse3 = {
  typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse3();
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
    const message = createBaseMsgUpdateParamsResponse3();
    return message;
  }
};

// src/generated/zerone/capture_challenge/v1/tx.registry.ts
var registry3 = [["/zerone.capture_challenge.v1.MsgSubmitChallenge", MsgSubmitChallenge], ["/zerone.capture_challenge.v1.MsgAddEvidence", MsgAddEvidence], ["/zerone.capture_challenge.v1.MsgResolveChallenge", MsgResolveChallenge], ["/zerone.capture_challenge.v1.MsgFundBountyPool", MsgFundBountyPool], ["/zerone.capture_challenge.v1.MsgUpdateParams", MsgUpdateParams3]];
var MessageComposer3 = {
  encoded: {
    submitChallenge(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallenge",
        value: MsgSubmitChallenge.encode(value).finish()
      };
    },
    addEvidence(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidence",
        value: MsgAddEvidence.encode(value).finish()
      };
    },
    resolveChallenge(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallenge",
        value: MsgResolveChallenge.encode(value).finish()
      };
    },
    fundBountyPool(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPool",
        value: MsgFundBountyPool.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParams",
        value: MsgUpdateParams3.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    submitChallenge(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallenge",
        value
      };
    },
    addEvidence(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidence",
        value
      };
    },
    resolveChallenge(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallenge",
        value
      };
    },
    fundBountyPool(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPool",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    submitChallenge(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgSubmitChallenge",
        value: MsgSubmitChallenge.fromPartial(value)
      };
    },
    addEvidence(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgAddEvidence",
        value: MsgAddEvidence.fromPartial(value)
      };
    },
    resolveChallenge(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgResolveChallenge",
        value: MsgResolveChallenge.fromPartial(value)
      };
    },
    fundBountyPool(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgFundBountyPool",
        value: MsgFundBountyPool.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.capture_challenge.v1.MsgUpdateParams",
        value: MsgUpdateParams3.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/capture_defense/v1/tx.ts
var tx_exports4 = {};
__export(tx_exports4, {
  MsgAnalyzeDomain: () => MsgAnalyzeDomain,
  MsgAnalyzeDomainResponse: () => MsgAnalyzeDomainResponse,
  MsgRecordVerification: () => MsgRecordVerification,
  MsgRecordVerificationResponse: () => MsgRecordVerificationResponse,
  MsgUpdateParams: () => MsgUpdateParams4,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse4
});

// src/generated/zerone/capture_defense/v1/genesis.ts
function createBaseParams4() {
  return {
    decayEpochBlocks: BigInt(0),
    minVerificationsForScore: BigInt(0),
    hhiThreshold: BigInt(0),
    riskAnalysisInterval: BigInt(0),
    historyRetentionBlocks: BigInt(0),
    baseReputationScore: BigInt(0),
    maxHistoryPerDomain: BigInt(0),
    baseReputationRecoveryBps: BigInt(0),
    activityRecoveryBonusMaxBps: BigInt(0)
  };
}
var Params4 = {
  typeUrl: "/zerone.capture_defense.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.decayEpochBlocks !== BigInt(0)) {
      writer.uint32(8).uint64(message.decayEpochBlocks);
    }
    if (message.minVerificationsForScore !== BigInt(0)) {
      writer.uint32(16).uint64(message.minVerificationsForScore);
    }
    if (message.hhiThreshold !== BigInt(0)) {
      writer.uint32(24).uint64(message.hhiThreshold);
    }
    if (message.riskAnalysisInterval !== BigInt(0)) {
      writer.uint32(32).uint64(message.riskAnalysisInterval);
    }
    if (message.historyRetentionBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.historyRetentionBlocks);
    }
    if (message.baseReputationScore !== BigInt(0)) {
      writer.uint32(48).uint64(message.baseReputationScore);
    }
    if (message.maxHistoryPerDomain !== BigInt(0)) {
      writer.uint32(56).uint64(message.maxHistoryPerDomain);
    }
    if (message.baseReputationRecoveryBps !== BigInt(0)) {
      writer.uint32(64).uint64(message.baseReputationRecoveryBps);
    }
    if (message.activityRecoveryBonusMaxBps !== BigInt(0)) {
      writer.uint32(72).uint64(message.activityRecoveryBonusMaxBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams4();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.decayEpochBlocks = reader.uint64();
          break;
        case 2:
          message.minVerificationsForScore = reader.uint64();
          break;
        case 3:
          message.hhiThreshold = reader.uint64();
          break;
        case 4:
          message.riskAnalysisInterval = reader.uint64();
          break;
        case 5:
          message.historyRetentionBlocks = reader.uint64();
          break;
        case 6:
          message.baseReputationScore = reader.uint64();
          break;
        case 7:
          message.maxHistoryPerDomain = reader.uint64();
          break;
        case 8:
          message.baseReputationRecoveryBps = reader.uint64();
          break;
        case 9:
          message.activityRecoveryBonusMaxBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams4();
    message.decayEpochBlocks = object.decayEpochBlocks !== void 0 && object.decayEpochBlocks !== null ? BigInt(object.decayEpochBlocks.toString()) : BigInt(0);
    message.minVerificationsForScore = object.minVerificationsForScore !== void 0 && object.minVerificationsForScore !== null ? BigInt(object.minVerificationsForScore.toString()) : BigInt(0);
    message.hhiThreshold = object.hhiThreshold !== void 0 && object.hhiThreshold !== null ? BigInt(object.hhiThreshold.toString()) : BigInt(0);
    message.riskAnalysisInterval = object.riskAnalysisInterval !== void 0 && object.riskAnalysisInterval !== null ? BigInt(object.riskAnalysisInterval.toString()) : BigInt(0);
    message.historyRetentionBlocks = object.historyRetentionBlocks !== void 0 && object.historyRetentionBlocks !== null ? BigInt(object.historyRetentionBlocks.toString()) : BigInt(0);
    message.baseReputationScore = object.baseReputationScore !== void 0 && object.baseReputationScore !== null ? BigInt(object.baseReputationScore.toString()) : BigInt(0);
    message.maxHistoryPerDomain = object.maxHistoryPerDomain !== void 0 && object.maxHistoryPerDomain !== null ? BigInt(object.maxHistoryPerDomain.toString()) : BigInt(0);
    message.baseReputationRecoveryBps = object.baseReputationRecoveryBps !== void 0 && object.baseReputationRecoveryBps !== null ? BigInt(object.baseReputationRecoveryBps.toString()) : BigInt(0);
    message.activityRecoveryBonusMaxBps = object.activityRecoveryBonusMaxBps !== void 0 && object.activityRecoveryBonusMaxBps !== null ? BigInt(object.activityRecoveryBonusMaxBps.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/capture_defense/v1/tx.ts
function createBaseMsgRecordVerification() {
  return {
    authority: "",
    domain: "",
    roundId: "",
    validators: [],
    verdicts: [],
    submitBlocks: []
  };
}
var MsgRecordVerification = {
  typeUrl: "/zerone.capture_defense.v1.MsgRecordVerification",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.roundId !== "") {
      writer.uint32(26).string(message.roundId);
    }
    for (const v of message.validators) {
      writer.uint32(34).string(v);
    }
    writer.uint32(42).fork();
    for (const v of message.verdicts) {
      writer.bool(v);
    }
    writer.ldelim();
    writer.uint32(50).fork();
    for (const v of message.submitBlocks) {
      writer.uint64(v);
    }
    writer.ldelim();
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRecordVerification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.roundId = reader.string();
          break;
        case 4:
          message.validators.push(reader.string());
          break;
        case 5:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.verdicts.push(reader.bool());
            }
          } else {
            message.verdicts.push(reader.bool());
          }
          break;
        case 6:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.submitBlocks.push(reader.uint64());
            }
          } else {
            message.submitBlocks.push(reader.uint64());
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRecordVerification();
    message.authority = object.authority ?? "";
    message.domain = object.domain ?? "";
    message.roundId = object.roundId ?? "";
    message.validators = object.validators?.map((e) => e) || [];
    message.verdicts = object.verdicts?.map((e) => e) || [];
    message.submitBlocks = object.submitBlocks?.map((e) => BigInt(e.toString())) || [];
    return message;
  }
};
function createBaseMsgRecordVerificationResponse() {
  return {};
}
var MsgRecordVerificationResponse = {
  typeUrl: "/zerone.capture_defense.v1.MsgRecordVerificationResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRecordVerificationResponse();
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
    const message = createBaseMsgRecordVerificationResponse();
    return message;
  }
};
function createBaseMsgAnalyzeDomain() {
  return {
    sender: "",
    domain: ""
  };
}
var MsgAnalyzeDomain = {
  typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomain",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAnalyzeDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAnalyzeDomain();
    message.sender = object.sender ?? "";
    message.domain = object.domain ?? "";
    return message;
  }
};
function createBaseMsgAnalyzeDomainResponse() {
  return {
    riskScore: BigInt(0),
    flagged: false
  };
}
var MsgAnalyzeDomainResponse = {
  typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomainResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.riskScore !== BigInt(0)) {
      writer.uint32(8).uint64(message.riskScore);
    }
    if (message.flagged === true) {
      writer.uint32(16).bool(message.flagged);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAnalyzeDomainResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.riskScore = reader.uint64();
          break;
        case 2:
          message.flagged = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAnalyzeDomainResponse();
    message.riskScore = object.riskScore !== void 0 && object.riskScore !== null ? BigInt(object.riskScore.toString()) : BigInt(0);
    message.flagged = object.flagged ?? false;
    return message;
  }
};
function createBaseMsgUpdateParams4() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams4 = {
  typeUrl: "/zerone.capture_defense.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params4.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams4();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params4.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams4();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params4.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse4() {
  return {};
}
var MsgUpdateParamsResponse4 = {
  typeUrl: "/zerone.capture_defense.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse4();
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
    const message = createBaseMsgUpdateParamsResponse4();
    return message;
  }
};

// src/generated/zerone/capture_defense/v1/tx.registry.ts
var registry4 = [["/zerone.capture_defense.v1.MsgRecordVerification", MsgRecordVerification], ["/zerone.capture_defense.v1.MsgAnalyzeDomain", MsgAnalyzeDomain], ["/zerone.capture_defense.v1.MsgUpdateParams", MsgUpdateParams4]];
var MessageComposer4 = {
  encoded: {
    recordVerification(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgRecordVerification",
        value: MsgRecordVerification.encode(value).finish()
      };
    },
    analyzeDomain(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomain",
        value: MsgAnalyzeDomain.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgUpdateParams",
        value: MsgUpdateParams4.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    recordVerification(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgRecordVerification",
        value
      };
    },
    analyzeDomain(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomain",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    recordVerification(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgRecordVerification",
        value: MsgRecordVerification.fromPartial(value)
      };
    },
    analyzeDomain(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgAnalyzeDomain",
        value: MsgAnalyzeDomain.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.capture_defense.v1.MsgUpdateParams",
        value: MsgUpdateParams4.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/claiming_pot/v1/tx.ts
var tx_exports5 = {};
__export(tx_exports5, {
  MsgAddBootstrapEntry: () => MsgAddBootstrapEntry,
  MsgAddBootstrapEntryResponse: () => MsgAddBootstrapEntryResponse,
  MsgClaim: () => MsgClaim,
  MsgClaimResponse: () => MsgClaimResponse,
  MsgCreatePot: () => MsgCreatePot,
  MsgCreatePotResponse: () => MsgCreatePotResponse,
  MsgUpdatePotParams: () => MsgUpdatePotParams,
  MsgUpdatePotParamsResponse: () => MsgUpdatePotParamsResponse
});

// src/generated/zerone/claiming_pot/v1/state.ts
function createBaseVestingSchedule() {
  return {
    startBlock: BigInt(0),
    endBlock: BigInt(0),
    cliffBlocks: BigInt(0),
    periodBlocks: BigInt(0)
  };
}
var VestingSchedule = {
  typeUrl: "/zerone.claiming_pot.v1.VestingSchedule",
  encode(message, writer = BinaryWriter.create()) {
    if (message.startBlock !== BigInt(0)) {
      writer.uint32(8).uint64(message.startBlock);
    }
    if (message.endBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.endBlock);
    }
    if (message.cliffBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.cliffBlocks);
    }
    if (message.periodBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.periodBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseVestingSchedule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.startBlock = reader.uint64();
          break;
        case 2:
          message.endBlock = reader.uint64();
          break;
        case 3:
          message.cliffBlocks = reader.uint64();
          break;
        case 4:
          message.periodBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseVestingSchedule();
    message.startBlock = object.startBlock !== void 0 && object.startBlock !== null ? BigInt(object.startBlock.toString()) : BigInt(0);
    message.endBlock = object.endBlock !== void 0 && object.endBlock !== null ? BigInt(object.endBlock.toString()) : BigInt(0);
    message.cliffBlocks = object.cliffBlocks !== void 0 && object.cliffBlocks !== null ? BigInt(object.cliffBlocks.toString()) : BigInt(0);
    message.periodBlocks = object.periodBlocks !== void 0 && object.periodBlocks !== null ? BigInt(object.periodBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseEligibilityCriteria() {
  return {
    minStakingTier: 0,
    minRegistrationAge: BigInt(0),
    whitelist: []
  };
}
var EligibilityCriteria = {
  typeUrl: "/zerone.claiming_pot.v1.EligibilityCriteria",
  encode(message, writer = BinaryWriter.create()) {
    if (message.minStakingTier !== 0) {
      writer.uint32(8).uint32(message.minStakingTier);
    }
    if (message.minRegistrationAge !== BigInt(0)) {
      writer.uint32(16).uint64(message.minRegistrationAge);
    }
    for (const v of message.whitelist) {
      writer.uint32(26).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseEligibilityCriteria();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minStakingTier = reader.uint32();
          break;
        case 2:
          message.minRegistrationAge = reader.uint64();
          break;
        case 3:
          message.whitelist.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseEligibilityCriteria();
    message.minStakingTier = object.minStakingTier ?? 0;
    message.minRegistrationAge = object.minRegistrationAge !== void 0 && object.minRegistrationAge !== null ? BigInt(object.minRegistrationAge.toString()) : BigInt(0);
    message.whitelist = object.whitelist?.map((e) => e) || [];
    return message;
  }
};
function createBaseParams5() {
  return {
    maxPotsActive: 0,
    minClaimAmount: "",
    bootstrapRegistrar: "",
    bootstrapEmissionCapUzrn: "",
    bootstrapDailyAdmissionCap: BigInt(0)
  };
}
var Params5 = {
  typeUrl: "/zerone.claiming_pot.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.maxPotsActive !== 0) {
      writer.uint32(8).uint32(message.maxPotsActive);
    }
    if (message.minClaimAmount !== "") {
      writer.uint32(18).string(message.minClaimAmount);
    }
    if (message.bootstrapRegistrar !== "") {
      writer.uint32(26).string(message.bootstrapRegistrar);
    }
    if (message.bootstrapEmissionCapUzrn !== "") {
      writer.uint32(34).string(message.bootstrapEmissionCapUzrn);
    }
    if (message.bootstrapDailyAdmissionCap !== BigInt(0)) {
      writer.uint32(40).uint64(message.bootstrapDailyAdmissionCap);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams5();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.maxPotsActive = reader.uint32();
          break;
        case 2:
          message.minClaimAmount = reader.string();
          break;
        case 3:
          message.bootstrapRegistrar = reader.string();
          break;
        case 4:
          message.bootstrapEmissionCapUzrn = reader.string();
          break;
        case 5:
          message.bootstrapDailyAdmissionCap = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams5();
    message.maxPotsActive = object.maxPotsActive ?? 0;
    message.minClaimAmount = object.minClaimAmount ?? "";
    message.bootstrapRegistrar = object.bootstrapRegistrar ?? "";
    message.bootstrapEmissionCapUzrn = object.bootstrapEmissionCapUzrn ?? "";
    message.bootstrapDailyAdmissionCap = object.bootstrapDailyAdmissionCap !== void 0 && object.bootstrapDailyAdmissionCap !== null ? BigInt(object.bootstrapDailyAdmissionCap.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/claiming_pot/v1/tx.ts
function createBaseMsgCreatePot() {
  return {
    authority: "",
    name: "",
    totalAmount: "",
    schedule: void 0,
    eligibility: void 0
  };
}
var MsgCreatePot = {
  typeUrl: "/zerone.claiming_pot.v1.MsgCreatePot",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.totalAmount !== "") {
      writer.uint32(26).string(message.totalAmount);
    }
    if (message.schedule !== void 0) {
      VestingSchedule.encode(message.schedule, writer.uint32(34).fork()).ldelim();
    }
    if (message.eligibility !== void 0) {
      EligibilityCriteria.encode(message.eligibility, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreatePot();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.totalAmount = reader.string();
          break;
        case 4:
          message.schedule = VestingSchedule.decode(reader, reader.uint32());
          break;
        case 5:
          message.eligibility = EligibilityCriteria.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreatePot();
    message.authority = object.authority ?? "";
    message.name = object.name ?? "";
    message.totalAmount = object.totalAmount ?? "";
    message.schedule = object.schedule !== void 0 && object.schedule !== null ? VestingSchedule.fromPartial(object.schedule) : void 0;
    message.eligibility = object.eligibility !== void 0 && object.eligibility !== null ? EligibilityCriteria.fromPartial(object.eligibility) : void 0;
    return message;
  }
};
function createBaseMsgCreatePotResponse() {
  return {
    potId: ""
  };
}
var MsgCreatePotResponse = {
  typeUrl: "/zerone.claiming_pot.v1.MsgCreatePotResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.potId !== "") {
      writer.uint32(10).string(message.potId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreatePotResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.potId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreatePotResponse();
    message.potId = object.potId ?? "";
    return message;
  }
};
function createBaseMsgClaim() {
  return {
    claimant: "",
    potId: ""
  };
}
var MsgClaim = {
  typeUrl: "/zerone.claiming_pot.v1.MsgClaim",
  encode(message, writer = BinaryWriter.create()) {
    if (message.claimant !== "") {
      writer.uint32(10).string(message.claimant);
    }
    if (message.potId !== "") {
      writer.uint32(18).string(message.potId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgClaim();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimant = reader.string();
          break;
        case 2:
          message.potId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgClaim();
    message.claimant = object.claimant ?? "";
    message.potId = object.potId ?? "";
    return message;
  }
};
function createBaseMsgClaimResponse() {
  return {
    amount: ""
  };
}
var MsgClaimResponse = {
  typeUrl: "/zerone.claiming_pot.v1.MsgClaimResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.amount !== "") {
      writer.uint32(10).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgClaimResponse();
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgUpdatePotParams() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdatePotParams = {
  typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params5.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdatePotParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params5.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdatePotParams();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params5.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdatePotParamsResponse() {
  return {};
}
var MsgUpdatePotParamsResponse = {
  typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdatePotParamsResponse();
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
    const message = createBaseMsgUpdatePotParamsResponse();
    return message;
  }
};
function createBaseMsgAddBootstrapEntry() {
  return {
    authority: "",
    addresses: []
  };
}
var MsgAddBootstrapEntry = {
  typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    for (const v of message.addresses) {
      writer.uint32(18).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddBootstrapEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.addresses.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddBootstrapEntry();
    message.authority = object.authority ?? "";
    message.addresses = object.addresses?.map((e) => e) || [];
    return message;
  }
};
function createBaseMsgAddBootstrapEntryResponse() {
  return {
    addedCount: 0,
    skippedCount: 0
  };
}
var MsgAddBootstrapEntryResponse = {
  typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntryResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.addedCount !== 0) {
      writer.uint32(8).uint32(message.addedCount);
    }
    if (message.skippedCount !== 0) {
      writer.uint32(16).uint32(message.skippedCount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddBootstrapEntryResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.addedCount = reader.uint32();
          break;
        case 2:
          message.skippedCount = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddBootstrapEntryResponse();
    message.addedCount = object.addedCount ?? 0;
    message.skippedCount = object.skippedCount ?? 0;
    return message;
  }
};

// src/generated/zerone/claiming_pot/v1/tx.registry.ts
var registry5 = [["/zerone.claiming_pot.v1.MsgCreatePot", MsgCreatePot], ["/zerone.claiming_pot.v1.MsgClaim", MsgClaim], ["/zerone.claiming_pot.v1.MsgUpdatePotParams", MsgUpdatePotParams], ["/zerone.claiming_pot.v1.MsgAddBootstrapEntry", MsgAddBootstrapEntry]];
var MessageComposer5 = {
  encoded: {
    createPot(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgCreatePot",
        value: MsgCreatePot.encode(value).finish()
      };
    },
    claim(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgClaim",
        value: MsgClaim.encode(value).finish()
      };
    },
    updatePotParams(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParams",
        value: MsgUpdatePotParams.encode(value).finish()
      };
    },
    addBootstrapEntry(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
        value: MsgAddBootstrapEntry.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createPot(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgCreatePot",
        value
      };
    },
    claim(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgClaim",
        value
      };
    },
    updatePotParams(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParams",
        value
      };
    },
    addBootstrapEntry(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
        value
      };
    }
  },
  fromPartial: {
    createPot(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgCreatePot",
        value: MsgCreatePot.fromPartial(value)
      };
    },
    claim(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgClaim",
        value: MsgClaim.fromPartial(value)
      };
    },
    updatePotParams(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParams",
        value: MsgUpdatePotParams.fromPartial(value)
      };
    },
    addBootstrapEntry(value) {
      return {
        typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
        value: MsgAddBootstrapEntry.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/counterexamples/v1/tx.ts
var tx_exports6 = {};
__export(tx_exports6, {
  MsgProposeCounterexample: () => MsgProposeCounterexample,
  MsgProposeCounterexampleResponse: () => MsgProposeCounterexampleResponse,
  MsgUpdateParams: () => MsgUpdateParams5,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse5,
  MsgValidate: () => MsgValidate,
  MsgValidateResponse: () => MsgValidateResponse
});

// src/generated/zerone/counterexamples/v1/genesis.ts
function createBaseParams6() {
  return {
    proposalBond: "",
    validationReward: "",
    minVotes: 0,
    affirmThresholdBps: BigInt(0),
    maxReasonBytes: 0,
    tvwMultiplierBps: BigInt(0),
    proposalsEnabled: false
  };
}
var Params6 = {
  typeUrl: "/zerone.counterexamples.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalBond !== "") {
      writer.uint32(10).string(message.proposalBond);
    }
    if (message.validationReward !== "") {
      writer.uint32(18).string(message.validationReward);
    }
    if (message.minVotes !== 0) {
      writer.uint32(24).uint32(message.minVotes);
    }
    if (message.affirmThresholdBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.affirmThresholdBps);
    }
    if (message.maxReasonBytes !== 0) {
      writer.uint32(40).uint32(message.maxReasonBytes);
    }
    if (message.tvwMultiplierBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.tvwMultiplierBps);
    }
    if (message.proposalsEnabled === true) {
      writer.uint32(56).bool(message.proposalsEnabled);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams6();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalBond = reader.string();
          break;
        case 2:
          message.validationReward = reader.string();
          break;
        case 3:
          message.minVotes = reader.uint32();
          break;
        case 4:
          message.affirmThresholdBps = reader.uint64();
          break;
        case 5:
          message.maxReasonBytes = reader.uint32();
          break;
        case 6:
          message.tvwMultiplierBps = reader.uint64();
          break;
        case 7:
          message.proposalsEnabled = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams6();
    message.proposalBond = object.proposalBond ?? "";
    message.validationReward = object.validationReward ?? "";
    message.minVotes = object.minVotes ?? 0;
    message.affirmThresholdBps = object.affirmThresholdBps !== void 0 && object.affirmThresholdBps !== null ? BigInt(object.affirmThresholdBps.toString()) : BigInt(0);
    message.maxReasonBytes = object.maxReasonBytes ?? 0;
    message.tvwMultiplierBps = object.tvwMultiplierBps !== void 0 && object.tvwMultiplierBps !== null ? BigInt(object.tvwMultiplierBps.toString()) : BigInt(0);
    message.proposalsEnabled = object.proposalsEnabled ?? false;
    return message;
  }
};

// src/generated/zerone/counterexamples/v1/tx.ts
function createBaseMsgProposeCounterexample() {
  return {
    author: "",
    factId: "",
    wrongClaim: "",
    reasoning: "",
    errorType: 0,
    violatedMethodologyIds: []
  };
}
var MsgProposeCounterexample = {
  typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexample",
  encode(message, writer = BinaryWriter.create()) {
    if (message.author !== "") {
      writer.uint32(10).string(message.author);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.wrongClaim !== "") {
      writer.uint32(26).string(message.wrongClaim);
    }
    if (message.reasoning !== "") {
      writer.uint32(34).string(message.reasoning);
    }
    if (message.errorType !== 0) {
      writer.uint32(40).int32(message.errorType);
    }
    for (const v of message.violatedMethodologyIds) {
      writer.uint32(50).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeCounterexample();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.author = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.wrongClaim = reader.string();
          break;
        case 4:
          message.reasoning = reader.string();
          break;
        case 5:
          message.errorType = reader.int32();
          break;
        case 6:
          message.violatedMethodologyIds.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeCounterexample();
    message.author = object.author ?? "";
    message.factId = object.factId ?? "";
    message.wrongClaim = object.wrongClaim ?? "";
    message.reasoning = object.reasoning ?? "";
    message.errorType = object.errorType ?? 0;
    message.violatedMethodologyIds = object.violatedMethodologyIds?.map((e) => e) || [];
    return message;
  }
};
function createBaseMsgProposeCounterexampleResponse() {
  return {
    counterexampleId: ""
  };
}
var MsgProposeCounterexampleResponse = {
  typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexampleResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.counterexampleId !== "") {
      writer.uint32(10).string(message.counterexampleId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeCounterexampleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.counterexampleId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeCounterexampleResponse();
    message.counterexampleId = object.counterexampleId ?? "";
    return message;
  }
};
function createBaseMsgValidate() {
  return {
    validator: "",
    counterexampleId: "",
    affirm: false,
    reason: ""
  };
}
var MsgValidate = {
  typeUrl: "/zerone.counterexamples.v1.MsgValidate",
  encode(message, writer = BinaryWriter.create()) {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.counterexampleId !== "") {
      writer.uint32(18).string(message.counterexampleId);
    }
    if (message.affirm === true) {
      writer.uint32(24).bool(message.affirm);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgValidate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.counterexampleId = reader.string();
          break;
        case 3:
          message.affirm = reader.bool();
          break;
        case 4:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgValidate();
    message.validator = object.validator ?? "";
    message.counterexampleId = object.counterexampleId ?? "";
    message.affirm = object.affirm ?? false;
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgValidateResponse() {
  return {
    validationId: BigInt(0),
    resolved: false,
    status: 0
  };
}
var MsgValidateResponse = {
  typeUrl: "/zerone.counterexamples.v1.MsgValidateResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.validationId !== BigInt(0)) {
      writer.uint32(8).uint64(message.validationId);
    }
    if (message.resolved === true) {
      writer.uint32(16).bool(message.resolved);
    }
    if (message.status !== 0) {
      writer.uint32(24).int32(message.status);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgValidateResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validationId = reader.uint64();
          break;
        case 2:
          message.resolved = reader.bool();
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
    const message = createBaseMsgValidateResponse();
    message.validationId = object.validationId !== void 0 && object.validationId !== null ? BigInt(object.validationId.toString()) : BigInt(0);
    message.resolved = object.resolved ?? false;
    message.status = object.status ?? 0;
    return message;
  }
};
function createBaseMsgUpdateParams5() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams5 = {
  typeUrl: "/zerone.counterexamples.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params6.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams5();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params6.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams5();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params6.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse5() {
  return {};
}
var MsgUpdateParamsResponse5 = {
  typeUrl: "/zerone.counterexamples.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse5();
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
    const message = createBaseMsgUpdateParamsResponse5();
    return message;
  }
};

// src/generated/zerone/counterexamples/v1/tx.registry.ts
var registry6 = [["/zerone.counterexamples.v1.MsgProposeCounterexample", MsgProposeCounterexample], ["/zerone.counterexamples.v1.MsgValidate", MsgValidate], ["/zerone.counterexamples.v1.MsgUpdateParams", MsgUpdateParams5]];
var MessageComposer6 = {
  encoded: {
    proposeCounterexample(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexample",
        value: MsgProposeCounterexample.encode(value).finish()
      };
    },
    validate(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgValidate",
        value: MsgValidate.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgUpdateParams",
        value: MsgUpdateParams5.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    proposeCounterexample(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexample",
        value
      };
    },
    validate(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgValidate",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    proposeCounterexample(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgProposeCounterexample",
        value: MsgProposeCounterexample.fromPartial(value)
      };
    },
    validate(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgValidate",
        value: MsgValidate.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.counterexamples.v1.MsgUpdateParams",
        value: MsgUpdateParams5.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/creed/v1/tx.ts
var tx_exports7 = {};
__export(tx_exports7, {
  MsgAnchorPin: () => MsgAnchorPin,
  MsgAnchorPinResponse: () => MsgAnchorPinResponse,
  MsgUpdateCouncilMember: () => MsgUpdateCouncilMember,
  MsgUpdateCouncilMemberResponse: () => MsgUpdateCouncilMemberResponse,
  MsgUpdateParams: () => MsgUpdateParams6,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse6
});

// src/generated/zerone/creed/v1/types.ts
function createBasePinnedCreed() {
  return {
    version: 0,
    canonicalHash: new Uint8Array(),
    pinnedAtHeight: BigInt(0),
    pinnedViaLip: "",
    commitments: []
  };
}
var PinnedCreed = {
  typeUrl: "/zerone.creed.v1.PinnedCreed",
  encode(message, writer = BinaryWriter.create()) {
    if (message.version !== 0) {
      writer.uint32(8).uint32(message.version);
    }
    if (message.canonicalHash.length !== 0) {
      writer.uint32(18).bytes(message.canonicalHash);
    }
    if (message.pinnedAtHeight !== BigInt(0)) {
      writer.uint32(24).uint64(message.pinnedAtHeight);
    }
    if (message.pinnedViaLip !== "") {
      writer.uint32(34).string(message.pinnedViaLip);
    }
    for (const v of message.commitments) {
      CommitmentEntry.encode(v, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBasePinnedCreed();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.version = reader.uint32();
          break;
        case 2:
          message.canonicalHash = reader.bytes();
          break;
        case 3:
          message.pinnedAtHeight = reader.uint64();
          break;
        case 4:
          message.pinnedViaLip = reader.string();
          break;
        case 5:
          message.commitments.push(CommitmentEntry.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBasePinnedCreed();
    message.version = object.version ?? 0;
    message.canonicalHash = object.canonicalHash ?? new Uint8Array();
    message.pinnedAtHeight = object.pinnedAtHeight !== void 0 && object.pinnedAtHeight !== null ? BigInt(object.pinnedAtHeight.toString()) : BigInt(0);
    message.pinnedViaLip = object.pinnedViaLip ?? "";
    message.commitments = object.commitments?.map((e) => CommitmentEntry.fromPartial(e)) || [];
    return message;
  }
};
function createBaseCreedCouncilMember() {
  return {
    address: "",
    admittedAtHeight: BigInt(0),
    admittedViaLip: "",
    votingWeightBps: BigInt(0),
    active: false,
    admissionBasis: ""
  };
}
var CreedCouncilMember = {
  typeUrl: "/zerone.creed.v1.CreedCouncilMember",
  encode(message, writer = BinaryWriter.create()) {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.admittedAtHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.admittedAtHeight);
    }
    if (message.admittedViaLip !== "") {
      writer.uint32(26).string(message.admittedViaLip);
    }
    if (message.votingWeightBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.votingWeightBps);
    }
    if (message.active === true) {
      writer.uint32(40).bool(message.active);
    }
    if (message.admissionBasis !== "") {
      writer.uint32(50).string(message.admissionBasis);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseCreedCouncilMember();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.admittedAtHeight = reader.uint64();
          break;
        case 3:
          message.admittedViaLip = reader.string();
          break;
        case 4:
          message.votingWeightBps = reader.uint64();
          break;
        case 5:
          message.active = reader.bool();
          break;
        case 6:
          message.admissionBasis = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseCreedCouncilMember();
    message.address = object.address ?? "";
    message.admittedAtHeight = object.admittedAtHeight !== void 0 && object.admittedAtHeight !== null ? BigInt(object.admittedAtHeight.toString()) : BigInt(0);
    message.admittedViaLip = object.admittedViaLip ?? "";
    message.votingWeightBps = object.votingWeightBps !== void 0 && object.votingWeightBps !== null ? BigInt(object.votingWeightBps.toString()) : BigInt(0);
    message.active = object.active ?? false;
    message.admissionBasis = object.admissionBasis ?? "";
    return message;
  }
};
function createBaseCommitmentEntry() {
  return {
    number: 0,
    name: "",
    introducedAtHeight: BigInt(0),
    introducedViaLip: "",
    archived: false,
    archivedAtHeight: BigInt(0)
  };
}
var CommitmentEntry = {
  typeUrl: "/zerone.creed.v1.CommitmentEntry",
  encode(message, writer = BinaryWriter.create()) {
    if (message.number !== 0) {
      writer.uint32(8).uint32(message.number);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.introducedAtHeight !== BigInt(0)) {
      writer.uint32(24).uint64(message.introducedAtHeight);
    }
    if (message.introducedViaLip !== "") {
      writer.uint32(34).string(message.introducedViaLip);
    }
    if (message.archived === true) {
      writer.uint32(40).bool(message.archived);
    }
    if (message.archivedAtHeight !== BigInt(0)) {
      writer.uint32(48).uint64(message.archivedAtHeight);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseCommitmentEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.number = reader.uint32();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.introducedAtHeight = reader.uint64();
          break;
        case 4:
          message.introducedViaLip = reader.string();
          break;
        case 5:
          message.archived = reader.bool();
          break;
        case 6:
          message.archivedAtHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseCommitmentEntry();
    message.number = object.number ?? 0;
    message.name = object.name ?? "";
    message.introducedAtHeight = object.introducedAtHeight !== void 0 && object.introducedAtHeight !== null ? BigInt(object.introducedAtHeight.toString()) : BigInt(0);
    message.introducedViaLip = object.introducedViaLip ?? "";
    message.archived = object.archived ?? false;
    message.archivedAtHeight = object.archivedAtHeight !== void 0 && object.archivedAtHeight !== null ? BigInt(object.archivedAtHeight.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/creed/v1/genesis.ts
function createBaseParams7() {
  return {
    authority: "",
    directAnchorEnabled: false
  };
}
var Params7 = {
  typeUrl: "/zerone.creed.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.directAnchorEnabled === true) {
      writer.uint32(16).bool(message.directAnchorEnabled);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams7();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.directAnchorEnabled = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams7();
    message.authority = object.authority ?? "";
    message.directAnchorEnabled = object.directAnchorEnabled ?? false;
    return message;
  }
};

// src/generated/zerone/creed/v1/tx.ts
function createBaseMsgAnchorPin() {
  return {
    authority: "",
    pin: void 0,
    sourceLip: ""
  };
}
var MsgAnchorPin = {
  typeUrl: "/zerone.creed.v1.MsgAnchorPin",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.pin !== void 0) {
      PinnedCreed.encode(message.pin, writer.uint32(18).fork()).ldelim();
    }
    if (message.sourceLip !== "") {
      writer.uint32(26).string(message.sourceLip);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAnchorPin();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.pin = PinnedCreed.decode(reader, reader.uint32());
          break;
        case 3:
          message.sourceLip = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAnchorPin();
    message.authority = object.authority ?? "";
    message.pin = object.pin !== void 0 && object.pin !== null ? PinnedCreed.fromPartial(object.pin) : void 0;
    message.sourceLip = object.sourceLip ?? "";
    return message;
  }
};
function createBaseMsgAnchorPinResponse() {
  return {
    newVersion: 0
  };
}
var MsgAnchorPinResponse = {
  typeUrl: "/zerone.creed.v1.MsgAnchorPinResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.newVersion !== 0) {
      writer.uint32(8).uint32(message.newVersion);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAnchorPinResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newVersion = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAnchorPinResponse();
    message.newVersion = object.newVersion ?? 0;
    return message;
  }
};
function createBaseMsgUpdateParams6() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams6 = {
  typeUrl: "/zerone.creed.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params7.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams6();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params7.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams6();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params7.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse6() {
  return {};
}
var MsgUpdateParamsResponse6 = {
  typeUrl: "/zerone.creed.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse6();
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
    const message = createBaseMsgUpdateParamsResponse6();
    return message;
  }
};
function createBaseMsgUpdateCouncilMember() {
  return {
    authority: "",
    member: void 0,
    sourceLip: ""
  };
}
var MsgUpdateCouncilMember = {
  typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMember",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.member !== void 0) {
      CreedCouncilMember.encode(message.member, writer.uint32(18).fork()).ldelim();
    }
    if (message.sourceLip !== "") {
      writer.uint32(26).string(message.sourceLip);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateCouncilMember();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.member = CreedCouncilMember.decode(reader, reader.uint32());
          break;
        case 3:
          message.sourceLip = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateCouncilMember();
    message.authority = object.authority ?? "";
    message.member = object.member !== void 0 && object.member !== null ? CreedCouncilMember.fromPartial(object.member) : void 0;
    message.sourceLip = object.sourceLip ?? "";
    return message;
  }
};
function createBaseMsgUpdateCouncilMemberResponse() {
  return {};
}
var MsgUpdateCouncilMemberResponse = {
  typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMemberResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateCouncilMemberResponse();
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
    const message = createBaseMsgUpdateCouncilMemberResponse();
    return message;
  }
};

// src/generated/zerone/creed/v1/tx.registry.ts
var registry7 = [["/zerone.creed.v1.MsgAnchorPin", MsgAnchorPin], ["/zerone.creed.v1.MsgUpdateParams", MsgUpdateParams6], ["/zerone.creed.v1.MsgUpdateCouncilMember", MsgUpdateCouncilMember]];
var MessageComposer7 = {
  encoded: {
    anchorPin(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgAnchorPin",
        value: MsgAnchorPin.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateParams",
        value: MsgUpdateParams6.encode(value).finish()
      };
    },
    updateCouncilMember(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMember",
        value: MsgUpdateCouncilMember.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    anchorPin(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgAnchorPin",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateParams",
        value
      };
    },
    updateCouncilMember(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMember",
        value
      };
    }
  },
  fromPartial: {
    anchorPin(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgAnchorPin",
        value: MsgAnchorPin.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateParams",
        value: MsgUpdateParams6.fromPartial(value)
      };
    },
    updateCouncilMember(value) {
      return {
        typeUrl: "/zerone.creed.v1.MsgUpdateCouncilMember",
        value: MsgUpdateCouncilMember.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/emergency/v1/tx.ts
var tx_exports8 = {};
__export(tx_exports8, {
  EmergencyCategory: () => EmergencyCategory,
  MsgProposeHalt: () => MsgProposeHalt,
  MsgProposeHaltResponse: () => MsgProposeHaltResponse,
  MsgProposeResume: () => MsgProposeResume,
  MsgProposeResumeResponse: () => MsgProposeResumeResponse,
  MsgProposeRevert: () => MsgProposeRevert,
  MsgProposeRevertResponse: () => MsgProposeRevertResponse,
  MsgUpdateParams: () => MsgUpdateParams7,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse7,
  MsgVoteHalt: () => MsgVoteHalt,
  MsgVoteHaltResponse: () => MsgVoteHaltResponse,
  MsgVoteResume: () => MsgVoteResume,
  MsgVoteResumeResponse: () => MsgVoteResumeResponse,
  MsgVoteRevert: () => MsgVoteRevert,
  MsgVoteRevertResponse: () => MsgVoteRevertResponse,
  emergencyCategoryFromJSON: () => emergencyCategoryFromJSON,
  emergencyCategoryToJSON: () => emergencyCategoryToJSON
});

// src/generated/zerone/emergency/v1/genesis.ts
function createBaseParams8() {
  return {
    haltQuorum: BigInt(0),
    revertQuorum: BigInt(0),
    resumeQuorum: BigInt(0),
    haltPrevoteBlocks: BigInt(0),
    haltPrecommitBlocks: BigInt(0),
    haltTimeoutBlocks: BigInt(0),
    revertPrevoteBlocks: BigInt(0),
    revertPrecommitBlocks: BigInt(0),
    revertTimeoutBlocks: BigInt(0),
    resumePrevoteBlocks: BigInt(0),
    resumePrecommitBlocks: BigInt(0),
    resumeTimeoutBlocks: BigInt(0),
    maxProposalsPerEpoch: BigInt(0),
    maxProposalsPerGuardianPerEpoch: BigInt(0),
    cooldownBlocks: BigInt(0),
    minGuardianStake: "",
    minDistinctVoters: BigInt(0),
    maxRevertDepth: BigInt(0),
    epochBlocks: BigInt(0),
    genesisCouncil: [],
    councilExpiryBlock: BigInt(0),
    councilVirtualStake: "",
    maxHaltDurationBlocks: BigInt(0)
  };
}
var Params8 = {
  typeUrl: "/zerone.emergency.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.haltQuorum !== BigInt(0)) {
      writer.uint32(8).uint64(message.haltQuorum);
    }
    if (message.revertQuorum !== BigInt(0)) {
      writer.uint32(16).uint64(message.revertQuorum);
    }
    if (message.resumeQuorum !== BigInt(0)) {
      writer.uint32(24).uint64(message.resumeQuorum);
    }
    if (message.haltPrevoteBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.haltPrevoteBlocks);
    }
    if (message.haltPrecommitBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.haltPrecommitBlocks);
    }
    if (message.haltTimeoutBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.haltTimeoutBlocks);
    }
    if (message.revertPrevoteBlocks !== BigInt(0)) {
      writer.uint32(56).uint64(message.revertPrevoteBlocks);
    }
    if (message.revertPrecommitBlocks !== BigInt(0)) {
      writer.uint32(64).uint64(message.revertPrecommitBlocks);
    }
    if (message.revertTimeoutBlocks !== BigInt(0)) {
      writer.uint32(72).uint64(message.revertTimeoutBlocks);
    }
    if (message.resumePrevoteBlocks !== BigInt(0)) {
      writer.uint32(80).uint64(message.resumePrevoteBlocks);
    }
    if (message.resumePrecommitBlocks !== BigInt(0)) {
      writer.uint32(88).uint64(message.resumePrecommitBlocks);
    }
    if (message.resumeTimeoutBlocks !== BigInt(0)) {
      writer.uint32(96).uint64(message.resumeTimeoutBlocks);
    }
    if (message.maxProposalsPerEpoch !== BigInt(0)) {
      writer.uint32(104).uint64(message.maxProposalsPerEpoch);
    }
    if (message.maxProposalsPerGuardianPerEpoch !== BigInt(0)) {
      writer.uint32(112).uint64(message.maxProposalsPerGuardianPerEpoch);
    }
    if (message.cooldownBlocks !== BigInt(0)) {
      writer.uint32(120).uint64(message.cooldownBlocks);
    }
    if (message.minGuardianStake !== "") {
      writer.uint32(130).string(message.minGuardianStake);
    }
    if (message.minDistinctVoters !== BigInt(0)) {
      writer.uint32(136).uint64(message.minDistinctVoters);
    }
    if (message.maxRevertDepth !== BigInt(0)) {
      writer.uint32(144).uint64(message.maxRevertDepth);
    }
    if (message.epochBlocks !== BigInt(0)) {
      writer.uint32(152).uint64(message.epochBlocks);
    }
    for (const v of message.genesisCouncil) {
      writer.uint32(162).string(v);
    }
    if (message.councilExpiryBlock !== BigInt(0)) {
      writer.uint32(168).uint64(message.councilExpiryBlock);
    }
    if (message.councilVirtualStake !== "") {
      writer.uint32(178).string(message.councilVirtualStake);
    }
    if (message.maxHaltDurationBlocks !== BigInt(0)) {
      writer.uint32(184).uint64(message.maxHaltDurationBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams8();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.haltQuorum = reader.uint64();
          break;
        case 2:
          message.revertQuorum = reader.uint64();
          break;
        case 3:
          message.resumeQuorum = reader.uint64();
          break;
        case 4:
          message.haltPrevoteBlocks = reader.uint64();
          break;
        case 5:
          message.haltPrecommitBlocks = reader.uint64();
          break;
        case 6:
          message.haltTimeoutBlocks = reader.uint64();
          break;
        case 7:
          message.revertPrevoteBlocks = reader.uint64();
          break;
        case 8:
          message.revertPrecommitBlocks = reader.uint64();
          break;
        case 9:
          message.revertTimeoutBlocks = reader.uint64();
          break;
        case 10:
          message.resumePrevoteBlocks = reader.uint64();
          break;
        case 11:
          message.resumePrecommitBlocks = reader.uint64();
          break;
        case 12:
          message.resumeTimeoutBlocks = reader.uint64();
          break;
        case 13:
          message.maxProposalsPerEpoch = reader.uint64();
          break;
        case 14:
          message.maxProposalsPerGuardianPerEpoch = reader.uint64();
          break;
        case 15:
          message.cooldownBlocks = reader.uint64();
          break;
        case 16:
          message.minGuardianStake = reader.string();
          break;
        case 17:
          message.minDistinctVoters = reader.uint64();
          break;
        case 18:
          message.maxRevertDepth = reader.uint64();
          break;
        case 19:
          message.epochBlocks = reader.uint64();
          break;
        case 20:
          message.genesisCouncil.push(reader.string());
          break;
        case 21:
          message.councilExpiryBlock = reader.uint64();
          break;
        case 22:
          message.councilVirtualStake = reader.string();
          break;
        case 23:
          message.maxHaltDurationBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams8();
    message.haltQuorum = object.haltQuorum !== void 0 && object.haltQuorum !== null ? BigInt(object.haltQuorum.toString()) : BigInt(0);
    message.revertQuorum = object.revertQuorum !== void 0 && object.revertQuorum !== null ? BigInt(object.revertQuorum.toString()) : BigInt(0);
    message.resumeQuorum = object.resumeQuorum !== void 0 && object.resumeQuorum !== null ? BigInt(object.resumeQuorum.toString()) : BigInt(0);
    message.haltPrevoteBlocks = object.haltPrevoteBlocks !== void 0 && object.haltPrevoteBlocks !== null ? BigInt(object.haltPrevoteBlocks.toString()) : BigInt(0);
    message.haltPrecommitBlocks = object.haltPrecommitBlocks !== void 0 && object.haltPrecommitBlocks !== null ? BigInt(object.haltPrecommitBlocks.toString()) : BigInt(0);
    message.haltTimeoutBlocks = object.haltTimeoutBlocks !== void 0 && object.haltTimeoutBlocks !== null ? BigInt(object.haltTimeoutBlocks.toString()) : BigInt(0);
    message.revertPrevoteBlocks = object.revertPrevoteBlocks !== void 0 && object.revertPrevoteBlocks !== null ? BigInt(object.revertPrevoteBlocks.toString()) : BigInt(0);
    message.revertPrecommitBlocks = object.revertPrecommitBlocks !== void 0 && object.revertPrecommitBlocks !== null ? BigInt(object.revertPrecommitBlocks.toString()) : BigInt(0);
    message.revertTimeoutBlocks = object.revertTimeoutBlocks !== void 0 && object.revertTimeoutBlocks !== null ? BigInt(object.revertTimeoutBlocks.toString()) : BigInt(0);
    message.resumePrevoteBlocks = object.resumePrevoteBlocks !== void 0 && object.resumePrevoteBlocks !== null ? BigInt(object.resumePrevoteBlocks.toString()) : BigInt(0);
    message.resumePrecommitBlocks = object.resumePrecommitBlocks !== void 0 && object.resumePrecommitBlocks !== null ? BigInt(object.resumePrecommitBlocks.toString()) : BigInt(0);
    message.resumeTimeoutBlocks = object.resumeTimeoutBlocks !== void 0 && object.resumeTimeoutBlocks !== null ? BigInt(object.resumeTimeoutBlocks.toString()) : BigInt(0);
    message.maxProposalsPerEpoch = object.maxProposalsPerEpoch !== void 0 && object.maxProposalsPerEpoch !== null ? BigInt(object.maxProposalsPerEpoch.toString()) : BigInt(0);
    message.maxProposalsPerGuardianPerEpoch = object.maxProposalsPerGuardianPerEpoch !== void 0 && object.maxProposalsPerGuardianPerEpoch !== null ? BigInt(object.maxProposalsPerGuardianPerEpoch.toString()) : BigInt(0);
    message.cooldownBlocks = object.cooldownBlocks !== void 0 && object.cooldownBlocks !== null ? BigInt(object.cooldownBlocks.toString()) : BigInt(0);
    message.minGuardianStake = object.minGuardianStake ?? "";
    message.minDistinctVoters = object.minDistinctVoters !== void 0 && object.minDistinctVoters !== null ? BigInt(object.minDistinctVoters.toString()) : BigInt(0);
    message.maxRevertDepth = object.maxRevertDepth !== void 0 && object.maxRevertDepth !== null ? BigInt(object.maxRevertDepth.toString()) : BigInt(0);
    message.epochBlocks = object.epochBlocks !== void 0 && object.epochBlocks !== null ? BigInt(object.epochBlocks.toString()) : BigInt(0);
    message.genesisCouncil = object.genesisCouncil?.map((e) => e) || [];
    message.councilExpiryBlock = object.councilExpiryBlock !== void 0 && object.councilExpiryBlock !== null ? BigInt(object.councilExpiryBlock.toString()) : BigInt(0);
    message.councilVirtualStake = object.councilVirtualStake ?? "";
    message.maxHaltDurationBlocks = object.maxHaltDurationBlocks !== void 0 && object.maxHaltDurationBlocks !== null ? BigInt(object.maxHaltDurationBlocks.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/emergency/v1/tx.ts
var EmergencyCategory = /* @__PURE__ */ ((EmergencyCategory2) => {
  EmergencyCategory2[EmergencyCategory2["EMERGENCY_CATEGORY_UNSPECIFIED"] = 0] = "EMERGENCY_CATEGORY_UNSPECIFIED";
  EmergencyCategory2[EmergencyCategory2["EMERGENCY_CATEGORY_SECURITY_BREACH"] = 1] = "EMERGENCY_CATEGORY_SECURITY_BREACH";
  EmergencyCategory2[EmergencyCategory2["EMERGENCY_CATEGORY_CONSENSUS_FAILURE"] = 2] = "EMERGENCY_CATEGORY_CONSENSUS_FAILURE";
  EmergencyCategory2[EmergencyCategory2["EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT"] = 3] = "EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT";
  EmergencyCategory2[EmergencyCategory2["EMERGENCY_CATEGORY_STATE_CORRUPTION"] = 4] = "EMERGENCY_CATEGORY_STATE_CORRUPTION";
  EmergencyCategory2[EmergencyCategory2["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
  return EmergencyCategory2;
})(EmergencyCategory || {});
function emergencyCategoryFromJSON(object) {
  switch (object) {
    case 0:
    case "EMERGENCY_CATEGORY_UNSPECIFIED":
      return 0 /* EMERGENCY_CATEGORY_UNSPECIFIED */;
    case 1:
    case "EMERGENCY_CATEGORY_SECURITY_BREACH":
      return 1 /* EMERGENCY_CATEGORY_SECURITY_BREACH */;
    case 2:
    case "EMERGENCY_CATEGORY_CONSENSUS_FAILURE":
      return 2 /* EMERGENCY_CATEGORY_CONSENSUS_FAILURE */;
    case 3:
    case "EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT":
      return 3 /* EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT */;
    case 4:
    case "EMERGENCY_CATEGORY_STATE_CORRUPTION":
      return 4 /* EMERGENCY_CATEGORY_STATE_CORRUPTION */;
    case -1:
    case "UNRECOGNIZED":
    default:
      return -1 /* UNRECOGNIZED */;
  }
}
function emergencyCategoryToJSON(object) {
  switch (object) {
    case 0 /* EMERGENCY_CATEGORY_UNSPECIFIED */:
      return "EMERGENCY_CATEGORY_UNSPECIFIED";
    case 1 /* EMERGENCY_CATEGORY_SECURITY_BREACH */:
      return "EMERGENCY_CATEGORY_SECURITY_BREACH";
    case 2 /* EMERGENCY_CATEGORY_CONSENSUS_FAILURE */:
      return "EMERGENCY_CATEGORY_CONSENSUS_FAILURE";
    case 3 /* EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT */:
      return "EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT";
    case 4 /* EMERGENCY_CATEGORY_STATE_CORRUPTION */:
      return "EMERGENCY_CATEGORY_STATE_CORRUPTION";
    case -1 /* UNRECOGNIZED */:
    default:
      return "UNRECOGNIZED";
  }
}
function createBaseMsgProposeHalt() {
  return {
    proposer: "",
    reason: "",
    category: 0
  };
}
var MsgProposeHalt = {
  typeUrl: "/zerone.emergency.v1.MsgProposeHalt",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.reason !== "") {
      writer.uint32(18).string(message.reason);
    }
    if (message.category !== 0) {
      writer.uint32(24).int32(message.category);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeHalt();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.reason = reader.string();
          break;
        case 3:
          message.category = reader.int32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeHalt();
    message.proposer = object.proposer ?? "";
    message.reason = object.reason ?? "";
    message.category = object.category ?? 0;
    return message;
  }
};
function createBaseMsgProposeHaltResponse() {
  return {
    proposalId: ""
  };
}
var MsgProposeHaltResponse = {
  typeUrl: "/zerone.emergency.v1.MsgProposeHaltResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeHaltResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeHaltResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteHalt() {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
var MsgVoteHalt = {
  typeUrl: "/zerone.emergency.v1.MsgVoteHalt",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.approve === true) {
      writer.uint32(24).bool(message.approve);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteHalt();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteHalt();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteHaltResponse() {
  return {
    quorumReached: false,
    chainHalted: false
  };
}
var MsgVoteHaltResponse = {
  typeUrl: "/zerone.emergency.v1.MsgVoteHaltResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.quorumReached === true) {
      writer.uint32(8).bool(message.quorumReached);
    }
    if (message.chainHalted === true) {
      writer.uint32(16).bool(message.chainHalted);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteHaltResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.quorumReached = reader.bool();
          break;
        case 2:
          message.chainHalted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteHaltResponse();
    message.quorumReached = object.quorumReached ?? false;
    message.chainHalted = object.chainHalted ?? false;
    return message;
  }
};
function createBaseMsgProposeRevert() {
  return {
    proposer: "",
    revertToHeight: BigInt(0),
    justification: ""
  };
}
var MsgProposeRevert = {
  typeUrl: "/zerone.emergency.v1.MsgProposeRevert",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.revertToHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.revertToHeight);
    }
    if (message.justification !== "") {
      writer.uint32(26).string(message.justification);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeRevert();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.revertToHeight = reader.uint64();
          break;
        case 3:
          message.justification = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeRevert();
    message.proposer = object.proposer ?? "";
    message.revertToHeight = object.revertToHeight !== void 0 && object.revertToHeight !== null ? BigInt(object.revertToHeight.toString()) : BigInt(0);
    message.justification = object.justification ?? "";
    return message;
  }
};
function createBaseMsgProposeRevertResponse() {
  return {
    proposalId: ""
  };
}
var MsgProposeRevertResponse = {
  typeUrl: "/zerone.emergency.v1.MsgProposeRevertResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeRevertResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeRevertResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteRevert() {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
var MsgVoteRevert = {
  typeUrl: "/zerone.emergency.v1.MsgVoteRevert",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.approve === true) {
      writer.uint32(24).bool(message.approve);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteRevert();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteRevert();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteRevertResponse() {
  return {
    quorumReached: false,
    revertExecuted: false
  };
}
var MsgVoteRevertResponse = {
  typeUrl: "/zerone.emergency.v1.MsgVoteRevertResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.quorumReached === true) {
      writer.uint32(8).bool(message.quorumReached);
    }
    if (message.revertExecuted === true) {
      writer.uint32(16).bool(message.revertExecuted);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteRevertResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.quorumReached = reader.bool();
          break;
        case 2:
          message.revertExecuted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteRevertResponse();
    message.quorumReached = object.quorumReached ?? false;
    message.revertExecuted = object.revertExecuted ?? false;
    return message;
  }
};
function createBaseMsgProposeResume() {
  return {
    proposer: "",
    justification: ""
  };
}
var MsgProposeResume = {
  typeUrl: "/zerone.emergency.v1.MsgProposeResume",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.justification !== "") {
      writer.uint32(18).string(message.justification);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeResume();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.justification = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeResume();
    message.proposer = object.proposer ?? "";
    message.justification = object.justification ?? "";
    return message;
  }
};
function createBaseMsgProposeResumeResponse() {
  return {
    proposalId: ""
  };
}
var MsgProposeResumeResponse = {
  typeUrl: "/zerone.emergency.v1.MsgProposeResumeResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeResumeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeResumeResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteResume() {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
var MsgVoteResume = {
  typeUrl: "/zerone.emergency.v1.MsgVoteResume",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.approve === true) {
      writer.uint32(24).bool(message.approve);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResume();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteResume();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteResumeResponse() {
  return {
    quorumReached: false,
    chainResumed: false
  };
}
var MsgVoteResumeResponse = {
  typeUrl: "/zerone.emergency.v1.MsgVoteResumeResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.quorumReached === true) {
      writer.uint32(8).bool(message.quorumReached);
    }
    if (message.chainResumed === true) {
      writer.uint32(16).bool(message.chainResumed);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResumeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.quorumReached = reader.bool();
          break;
        case 2:
          message.chainResumed = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteResumeResponse();
    message.quorumReached = object.quorumReached ?? false;
    message.chainResumed = object.chainResumed ?? false;
    return message;
  }
};
function createBaseMsgUpdateParams7() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams7 = {
  typeUrl: "/zerone.emergency.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params8.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams7();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params8.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams7();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params8.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse7() {
  return {};
}
var MsgUpdateParamsResponse7 = {
  typeUrl: "/zerone.emergency.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse7();
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
    const message = createBaseMsgUpdateParamsResponse7();
    return message;
  }
};

// src/generated/zerone/emergency/v1/tx.registry.ts
var registry8 = [["/zerone.emergency.v1.MsgProposeHalt", MsgProposeHalt], ["/zerone.emergency.v1.MsgVoteHalt", MsgVoteHalt], ["/zerone.emergency.v1.MsgProposeRevert", MsgProposeRevert], ["/zerone.emergency.v1.MsgVoteRevert", MsgVoteRevert], ["/zerone.emergency.v1.MsgProposeResume", MsgProposeResume], ["/zerone.emergency.v1.MsgVoteResume", MsgVoteResume], ["/zerone.emergency.v1.MsgUpdateParams", MsgUpdateParams7]];
var MessageComposer8 = {
  encoded: {
    proposeHalt(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeHalt",
        value: MsgProposeHalt.encode(value).finish()
      };
    },
    voteHalt(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteHalt",
        value: MsgVoteHalt.encode(value).finish()
      };
    },
    proposeRevert(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRevert",
        value: MsgProposeRevert.encode(value).finish()
      };
    },
    voteRevert(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRevert",
        value: MsgVoteRevert.encode(value).finish()
      };
    },
    proposeResume(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeResume",
        value: MsgProposeResume.encode(value).finish()
      };
    },
    voteResume(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteResume",
        value: MsgVoteResume.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgUpdateParams",
        value: MsgUpdateParams7.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    proposeHalt(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeHalt",
        value
      };
    },
    voteHalt(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteHalt",
        value
      };
    },
    proposeRevert(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRevert",
        value
      };
    },
    voteRevert(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRevert",
        value
      };
    },
    proposeResume(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeResume",
        value
      };
    },
    voteResume(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteResume",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    proposeHalt(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeHalt",
        value: MsgProposeHalt.fromPartial(value)
      };
    },
    voteHalt(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteHalt",
        value: MsgVoteHalt.fromPartial(value)
      };
    },
    proposeRevert(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeRevert",
        value: MsgProposeRevert.fromPartial(value)
      };
    },
    voteRevert(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteRevert",
        value: MsgVoteRevert.fromPartial(value)
      };
    },
    proposeResume(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgProposeResume",
        value: MsgProposeResume.fromPartial(value)
      };
    },
    voteResume(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgVoteResume",
        value: MsgVoteResume.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.emergency.v1.MsgUpdateParams",
        value: MsgUpdateParams7.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/gov/v1/tx.ts
var tx_exports9 = {};
__export(tx_exports9, {
  MsgAcceptSeatNomination: () => MsgAcceptSeatNomination,
  MsgAcceptSeatNominationResponse: () => MsgAcceptSeatNominationResponse,
  MsgAdvanceLIPStage: () => MsgAdvanceLIPStage,
  MsgAdvanceLIPStageResponse: () => MsgAdvanceLIPStageResponse,
  MsgAttachCreedAmendmentPin: () => MsgAttachCreedAmendmentPin,
  MsgAttachCreedAmendmentPinResponse: () => MsgAttachCreedAmendmentPinResponse,
  MsgAttachUpgradePlan: () => MsgAttachUpgradePlan,
  MsgAttachUpgradePlanResponse: () => MsgAttachUpgradePlanResponse,
  MsgCastVote: () => MsgCastVote,
  MsgCastVoteResponse: () => MsgCastVoteResponse,
  MsgDomainFormationFreeze: () => MsgDomainFormationFreeze,
  MsgDomainFormationFreezeResponse: () => MsgDomainFormationFreezeResponse,
  MsgNominateSeatElection: () => MsgNominateSeatElection,
  MsgNominateSeatElectionResponse: () => MsgNominateSeatElectionResponse,
  MsgSetResearchVoters: () => MsgSetResearchVoters,
  MsgSetResearchVotersResponse: () => MsgSetResearchVotersResponse,
  MsgStakeLIP: () => MsgStakeLIP,
  MsgStakeLIPResponse: () => MsgStakeLIPResponse,
  MsgSubmitLIP: () => MsgSubmitLIP,
  MsgSubmitLIPResponse: () => MsgSubmitLIPResponse,
  MsgSubmitResearchSpend: () => MsgSubmitResearchSpend,
  MsgSubmitResearchSpendResponse: () => MsgSubmitResearchSpendResponse,
  MsgUpdateParams: () => MsgUpdateParams8,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse8,
  MsgVoteResearchSpend: () => MsgVoteResearchSpend,
  MsgVoteResearchSpendResponse: () => MsgVoteResearchSpendResponse,
  MsgVoteSeatElection: () => MsgVoteSeatElection,
  MsgVoteSeatElectionResponse: () => MsgVoteSeatElectionResponse,
  MsgWithdrawLIP: () => MsgWithdrawLIP,
  MsgWithdrawLIPResponse: () => MsgWithdrawLIPResponse
});

// src/generated/zerone/gov/v1/types.ts
function createBaseParamChange() {
  return {
    module: "",
    key: "",
    value: ""
  };
}
var ParamChange = {
  typeUrl: "/zerone.gov.v1.ParamChange",
  encode(message, writer = BinaryWriter.create()) {
    if (message.module !== "") {
      writer.uint32(10).string(message.module);
    }
    if (message.key !== "") {
      writer.uint32(18).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(26).string(message.value);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParamChange();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.module = reader.string();
          break;
        case 2:
          message.key = reader.string();
          break;
        case 3:
          message.value = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParamChange();
    message.module = object.module ?? "";
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  }
};
function createBaseResearchFundVoters() {
  return {
    voter1: "",
    voter2: ""
  };
}
var ResearchFundVoters = {
  typeUrl: "/zerone.gov.v1.ResearchFundVoters",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter1 !== "") {
      writer.uint32(10).string(message.voter1);
    }
    if (message.voter2 !== "") {
      writer.uint32(18).string(message.voter2);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseResearchFundVoters();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter1 = reader.string();
          break;
        case 2:
          message.voter2 = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseResearchFundVoters();
    message.voter1 = object.voter1 ?? "";
    message.voter2 = object.voter2 ?? "";
    return message;
  }
};

// src/generated/zerone/gov/v1/genesis.ts
function createBaseParams9() {
  return {
    votingPeriodBlocks: BigInt(0),
    discussionPeriodBlocks: BigInt(0),
    quorumThresholdBps: BigInt(0),
    supportThresholdBps: BigInt(0),
    minLipStake: "",
    minVoteStake: "",
    categoryConfigs: [],
    researchFundVoters: void 0,
    researchDiscussionBlocks: BigInt(0),
    researchVotingBlocks: BigInt(0)
  };
}
var Params9 = {
  typeUrl: "/zerone.gov.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.votingPeriodBlocks !== BigInt(0)) {
      writer.uint32(8).uint64(message.votingPeriodBlocks);
    }
    if (message.discussionPeriodBlocks !== BigInt(0)) {
      writer.uint32(16).uint64(message.discussionPeriodBlocks);
    }
    if (message.quorumThresholdBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.quorumThresholdBps);
    }
    if (message.supportThresholdBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.supportThresholdBps);
    }
    if (message.minLipStake !== "") {
      writer.uint32(42).string(message.minLipStake);
    }
    if (message.minVoteStake !== "") {
      writer.uint32(50).string(message.minVoteStake);
    }
    for (const v of message.categoryConfigs) {
      CategoryConfig.encode(v, writer.uint32(58).fork()).ldelim();
    }
    if (message.researchFundVoters !== void 0) {
      ResearchFundVoters.encode(message.researchFundVoters, writer.uint32(66).fork()).ldelim();
    }
    if (message.researchDiscussionBlocks !== BigInt(0)) {
      writer.uint32(72).uint64(message.researchDiscussionBlocks);
    }
    if (message.researchVotingBlocks !== BigInt(0)) {
      writer.uint32(80).uint64(message.researchVotingBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams9();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.votingPeriodBlocks = reader.uint64();
          break;
        case 2:
          message.discussionPeriodBlocks = reader.uint64();
          break;
        case 3:
          message.quorumThresholdBps = reader.uint64();
          break;
        case 4:
          message.supportThresholdBps = reader.uint64();
          break;
        case 5:
          message.minLipStake = reader.string();
          break;
        case 6:
          message.minVoteStake = reader.string();
          break;
        case 7:
          message.categoryConfigs.push(CategoryConfig.decode(reader, reader.uint32()));
          break;
        case 8:
          message.researchFundVoters = ResearchFundVoters.decode(reader, reader.uint32());
          break;
        case 9:
          message.researchDiscussionBlocks = reader.uint64();
          break;
        case 10:
          message.researchVotingBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams9();
    message.votingPeriodBlocks = object.votingPeriodBlocks !== void 0 && object.votingPeriodBlocks !== null ? BigInt(object.votingPeriodBlocks.toString()) : BigInt(0);
    message.discussionPeriodBlocks = object.discussionPeriodBlocks !== void 0 && object.discussionPeriodBlocks !== null ? BigInt(object.discussionPeriodBlocks.toString()) : BigInt(0);
    message.quorumThresholdBps = object.quorumThresholdBps !== void 0 && object.quorumThresholdBps !== null ? BigInt(object.quorumThresholdBps.toString()) : BigInt(0);
    message.supportThresholdBps = object.supportThresholdBps !== void 0 && object.supportThresholdBps !== null ? BigInt(object.supportThresholdBps.toString()) : BigInt(0);
    message.minLipStake = object.minLipStake ?? "";
    message.minVoteStake = object.minVoteStake ?? "";
    message.categoryConfigs = object.categoryConfigs?.map((e) => CategoryConfig.fromPartial(e)) || [];
    message.researchFundVoters = object.researchFundVoters !== void 0 && object.researchFundVoters !== null ? ResearchFundVoters.fromPartial(object.researchFundVoters) : void 0;
    message.researchDiscussionBlocks = object.researchDiscussionBlocks !== void 0 && object.researchDiscussionBlocks !== null ? BigInt(object.researchDiscussionBlocks.toString()) : BigInt(0);
    message.researchVotingBlocks = object.researchVotingBlocks !== void 0 && object.researchVotingBlocks !== null ? BigInt(object.researchVotingBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCategoryConfig() {
  return {
    category: "",
    requiredStakeUzrn: "",
    reviewBlocks: BigInt(0)
  };
}
var CategoryConfig = {
  typeUrl: "/zerone.gov.v1.CategoryConfig",
  encode(message, writer = BinaryWriter.create()) {
    if (message.category !== "") {
      writer.uint32(10).string(message.category);
    }
    if (message.requiredStakeUzrn !== "") {
      writer.uint32(18).string(message.requiredStakeUzrn);
    }
    if (message.reviewBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.reviewBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseCategoryConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.category = reader.string();
          break;
        case 2:
          message.requiredStakeUzrn = reader.string();
          break;
        case 3:
          message.reviewBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseCategoryConfig();
    message.category = object.category ?? "";
    message.requiredStakeUzrn = object.requiredStakeUzrn ?? "";
    message.reviewBlocks = object.reviewBlocks !== void 0 && object.reviewBlocks !== null ? BigInt(object.reviewBlocks.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/gov/v1/tx.ts
function createBaseMsgSubmitLIP() {
  return {
    proposer: "",
    title: "",
    description: "",
    category: "",
    initialStake: "",
    paramChanges: []
  };
}
var MsgSubmitLIP = {
  typeUrl: "/zerone.gov.v1.MsgSubmitLIP",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    if (message.initialStake !== "") {
      writer.uint32(42).string(message.initialStake);
    }
    for (const v of message.paramChanges) {
      ParamChange.encode(v, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitLIP();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.category = reader.string();
          break;
        case 5:
          message.initialStake = reader.string();
          break;
        case 6:
          message.paramChanges.push(ParamChange.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitLIP();
    message.proposer = object.proposer ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.category = object.category ?? "";
    message.initialStake = object.initialStake ?? "";
    message.paramChanges = object.paramChanges?.map((e) => ParamChange.fromPartial(e)) || [];
    return message;
  }
};
function createBaseMsgSubmitLIPResponse() {
  return {
    lipId: ""
  };
}
var MsgSubmitLIPResponse = {
  typeUrl: "/zerone.gov.v1.MsgSubmitLIPResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.lipId !== "") {
      writer.uint32(10).string(message.lipId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitLIPResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.lipId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitLIPResponse();
    message.lipId = object.lipId ?? "";
    return message;
  }
};
function createBaseMsgStakeLIP() {
  return {
    staker: "",
    lipId: "",
    amount: ""
  };
}
var MsgStakeLIP = {
  typeUrl: "/zerone.gov.v1.MsgStakeLIP",
  encode(message, writer = BinaryWriter.create()) {
    if (message.staker !== "") {
      writer.uint32(10).string(message.staker);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgStakeLIP();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.staker = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgStakeLIP();
    message.staker = object.staker ?? "";
    message.lipId = object.lipId ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgStakeLIPResponse() {
  return {};
}
var MsgStakeLIPResponse = {
  typeUrl: "/zerone.gov.v1.MsgStakeLIPResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgStakeLIPResponse();
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
    const message = createBaseMsgStakeLIPResponse();
    return message;
  }
};
function createBaseMsgAdvanceLIPStage() {
  return {
    authority: "",
    lipId: ""
  };
}
var MsgAdvanceLIPStage = {
  typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStage",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAdvanceLIPStage();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAdvanceLIPStage();
    message.authority = object.authority ?? "";
    message.lipId = object.lipId ?? "";
    return message;
  }
};
function createBaseMsgAdvanceLIPStageResponse() {
  return {
    newStage: ""
  };
}
var MsgAdvanceLIPStageResponse = {
  typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStageResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.newStage !== "") {
      writer.uint32(10).string(message.newStage);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAdvanceLIPStageResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newStage = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAdvanceLIPStageResponse();
    message.newStage = object.newStage ?? "";
    return message;
  }
};
function createBaseMsgCastVote() {
  return {
    voter: "",
    lipId: "",
    option: ""
  };
}
var MsgCastVote = {
  typeUrl: "/zerone.gov.v1.MsgCastVote",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    if (message.option !== "") {
      writer.uint32(26).string(message.option);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCastVote();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        case 3:
          message.option = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCastVote();
    message.voter = object.voter ?? "";
    message.lipId = object.lipId ?? "";
    message.option = object.option ?? "";
    return message;
  }
};
function createBaseMsgCastVoteResponse() {
  return {
    effectiveWeight: ""
  };
}
var MsgCastVoteResponse = {
  typeUrl: "/zerone.gov.v1.MsgCastVoteResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.effectiveWeight !== "") {
      writer.uint32(10).string(message.effectiveWeight);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCastVoteResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.effectiveWeight = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCastVoteResponse();
    message.effectiveWeight = object.effectiveWeight ?? "";
    return message;
  }
};
function createBaseMsgWithdrawLIP() {
  return {
    proposer: "",
    lipId: ""
  };
}
var MsgWithdrawLIP = {
  typeUrl: "/zerone.gov.v1.MsgWithdrawLIP",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgWithdrawLIP();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgWithdrawLIP();
    message.proposer = object.proposer ?? "";
    message.lipId = object.lipId ?? "";
    return message;
  }
};
function createBaseMsgWithdrawLIPResponse() {
  return {};
}
var MsgWithdrawLIPResponse = {
  typeUrl: "/zerone.gov.v1.MsgWithdrawLIPResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgWithdrawLIPResponse();
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
    const message = createBaseMsgWithdrawLIPResponse();
    return message;
  }
};
function createBaseMsgUpdateParams8() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams8 = {
  typeUrl: "/zerone.gov.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params9.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams8();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params9.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams8();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params9.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse8() {
  return {};
}
var MsgUpdateParamsResponse8 = {
  typeUrl: "/zerone.gov.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse8();
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
    const message = createBaseMsgUpdateParamsResponse8();
    return message;
  }
};
function createBaseMsgSubmitResearchSpend() {
  return {
    proposer: "",
    title: "",
    description: "",
    recipient: "",
    amount: "",
    justification: ""
  };
}
var MsgSubmitResearchSpend = {
  typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpend",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.recipient !== "") {
      writer.uint32(34).string(message.recipient);
    }
    if (message.amount !== "") {
      writer.uint32(42).string(message.amount);
    }
    if (message.justification !== "") {
      writer.uint32(50).string(message.justification);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitResearchSpend();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.recipient = reader.string();
          break;
        case 5:
          message.amount = reader.string();
          break;
        case 6:
          message.justification = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitResearchSpend();
    message.proposer = object.proposer ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.recipient = object.recipient ?? "";
    message.amount = object.amount ?? "";
    message.justification = object.justification ?? "";
    return message;
  }
};
function createBaseMsgSubmitResearchSpendResponse() {
  return {
    proposalId: BigInt(0)
  };
}
var MsgSubmitResearchSpendResponse = {
  typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpendResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(8).uint64(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitResearchSpendResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitResearchSpendResponse();
    message.proposalId = object.proposalId !== void 0 && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgVoteResearchSpend() {
  return {
    voter: "",
    proposalId: BigInt(0),
    vote: "",
    reasoning: ""
  };
}
var MsgVoteResearchSpend = {
  typeUrl: "/zerone.gov.v1.MsgVoteResearchSpend",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(16).uint64(message.proposalId);
    }
    if (message.vote !== "") {
      writer.uint32(26).string(message.vote);
    }
    if (message.reasoning !== "") {
      writer.uint32(34).string(message.reasoning);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResearchSpend();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.uint64();
          break;
        case 3:
          message.vote = reader.string();
          break;
        case 4:
          message.reasoning = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteResearchSpend();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId !== void 0 && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    message.vote = object.vote ?? "";
    message.reasoning = object.reasoning ?? "";
    return message;
  }
};
function createBaseMsgVoteResearchSpendResponse() {
  return {};
}
var MsgVoteResearchSpendResponse = {
  typeUrl: "/zerone.gov.v1.MsgVoteResearchSpendResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResearchSpendResponse();
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
    const message = createBaseMsgVoteResearchSpendResponse();
    return message;
  }
};
function createBaseMsgSetResearchVoters() {
  return {
    authority: "",
    voters: void 0
  };
}
var MsgSetResearchVoters = {
  typeUrl: "/zerone.gov.v1.MsgSetResearchVoters",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.voters !== void 0) {
      ResearchFundVoters.encode(message.voters, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSetResearchVoters();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.voters = ResearchFundVoters.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSetResearchVoters();
    message.authority = object.authority ?? "";
    message.voters = object.voters !== void 0 && object.voters !== null ? ResearchFundVoters.fromPartial(object.voters) : void 0;
    return message;
  }
};
function createBaseMsgSetResearchVotersResponse() {
  return {};
}
var MsgSetResearchVotersResponse = {
  typeUrl: "/zerone.gov.v1.MsgSetResearchVotersResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSetResearchVotersResponse();
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
    const message = createBaseMsgSetResearchVotersResponse();
    return message;
  }
};
function createBaseMsgAttachUpgradePlan() {
  return {
    proposer: "",
    lipId: "",
    upgradeName: "",
    height: BigInt(0),
    info: ""
  };
}
var MsgAttachUpgradePlan = {
  typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlan",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    if (message.upgradeName !== "") {
      writer.uint32(26).string(message.upgradeName);
    }
    if (message.height !== BigInt(0)) {
      writer.uint32(32).int64(message.height);
    }
    if (message.info !== "") {
      writer.uint32(42).string(message.info);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAttachUpgradePlan();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        case 3:
          message.upgradeName = reader.string();
          break;
        case 4:
          message.height = reader.int64();
          break;
        case 5:
          message.info = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAttachUpgradePlan();
    message.proposer = object.proposer ?? "";
    message.lipId = object.lipId ?? "";
    message.upgradeName = object.upgradeName ?? "";
    message.height = object.height !== void 0 && object.height !== null ? BigInt(object.height.toString()) : BigInt(0);
    message.info = object.info ?? "";
    return message;
  }
};
function createBaseMsgAttachUpgradePlanResponse() {
  return {};
}
var MsgAttachUpgradePlanResponse = {
  typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlanResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAttachUpgradePlanResponse();
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
    const message = createBaseMsgAttachUpgradePlanResponse();
    return message;
  }
};
function createBaseMsgAttachCreedAmendmentPin() {
  return {
    proposer: "",
    lipId: "",
    canonicalHash: new Uint8Array(),
    commitmentsJson: new Uint8Array()
  };
}
var MsgAttachCreedAmendmentPin = {
  typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPin",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    if (message.canonicalHash.length !== 0) {
      writer.uint32(26).bytes(message.canonicalHash);
    }
    if (message.commitmentsJson.length !== 0) {
      writer.uint32(34).bytes(message.commitmentsJson);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAttachCreedAmendmentPin();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        case 3:
          message.canonicalHash = reader.bytes();
          break;
        case 4:
          message.commitmentsJson = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAttachCreedAmendmentPin();
    message.proposer = object.proposer ?? "";
    message.lipId = object.lipId ?? "";
    message.canonicalHash = object.canonicalHash ?? new Uint8Array();
    message.commitmentsJson = object.commitmentsJson ?? new Uint8Array();
    return message;
  }
};
function createBaseMsgAttachCreedAmendmentPinResponse() {
  return {};
}
var MsgAttachCreedAmendmentPinResponse = {
  typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPinResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAttachCreedAmendmentPinResponse();
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
    const message = createBaseMsgAttachCreedAmendmentPinResponse();
    return message;
  }
};
function createBaseMsgNominateSeatElection() {
  return {
    proposer: "",
    candidate: "",
    seatIndex: 0,
    statement: ""
  };
}
var MsgNominateSeatElection = {
  typeUrl: "/zerone.gov.v1.MsgNominateSeatElection",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.candidate !== "") {
      writer.uint32(18).string(message.candidate);
    }
    if (message.seatIndex !== 0) {
      writer.uint32(24).uint32(message.seatIndex);
    }
    if (message.statement !== "") {
      writer.uint32(34).string(message.statement);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgNominateSeatElection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.candidate = reader.string();
          break;
        case 3:
          message.seatIndex = reader.uint32();
          break;
        case 4:
          message.statement = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgNominateSeatElection();
    message.proposer = object.proposer ?? "";
    message.candidate = object.candidate ?? "";
    message.seatIndex = object.seatIndex ?? 0;
    message.statement = object.statement ?? "";
    return message;
  }
};
function createBaseMsgNominateSeatElectionResponse() {
  return {
    proposalId: BigInt(0)
  };
}
var MsgNominateSeatElectionResponse = {
  typeUrl: "/zerone.gov.v1.MsgNominateSeatElectionResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(8).uint64(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgNominateSeatElectionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgNominateSeatElectionResponse();
    message.proposalId = object.proposalId !== void 0 && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAcceptSeatNomination() {
  return {
    candidate: "",
    proposalId: BigInt(0)
  };
}
var MsgAcceptSeatNomination = {
  typeUrl: "/zerone.gov.v1.MsgAcceptSeatNomination",
  encode(message, writer = BinaryWriter.create()) {
    if (message.candidate !== "") {
      writer.uint32(10).string(message.candidate);
    }
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(16).uint64(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAcceptSeatNomination();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.candidate = reader.string();
          break;
        case 2:
          message.proposalId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAcceptSeatNomination();
    message.candidate = object.candidate ?? "";
    message.proposalId = object.proposalId !== void 0 && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAcceptSeatNominationResponse() {
  return {};
}
var MsgAcceptSeatNominationResponse = {
  typeUrl: "/zerone.gov.v1.MsgAcceptSeatNominationResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAcceptSeatNominationResponse();
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
    const message = createBaseMsgAcceptSeatNominationResponse();
    return message;
  }
};
function createBaseMsgVoteSeatElection() {
  return {
    voter: "",
    proposalId: BigInt(0),
    option: ""
  };
}
var MsgVoteSeatElection = {
  typeUrl: "/zerone.gov.v1.MsgVoteSeatElection",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(16).uint64(message.proposalId);
    }
    if (message.option !== "") {
      writer.uint32(26).string(message.option);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteSeatElection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.uint64();
          break;
        case 3:
          message.option = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteSeatElection();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId !== void 0 && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    message.option = object.option ?? "";
    return message;
  }
};
function createBaseMsgVoteSeatElectionResponse() {
  return {
    effectiveWeight: ""
  };
}
var MsgVoteSeatElectionResponse = {
  typeUrl: "/zerone.gov.v1.MsgVoteSeatElectionResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.effectiveWeight !== "") {
      writer.uint32(10).string(message.effectiveWeight);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteSeatElectionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.effectiveWeight = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteSeatElectionResponse();
    message.effectiveWeight = object.effectiveWeight ?? "";
    return message;
  }
};
function createBaseMsgDomainFormationFreeze() {
  return {
    authority: "",
    domain: "",
    durationBlocks: BigInt(0),
    reason: ""
  };
}
var MsgDomainFormationFreeze = {
  typeUrl: "/zerone.gov.v1.MsgDomainFormationFreeze",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.durationBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.durationBlocks);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgDomainFormationFreeze();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.durationBlocks = reader.uint64();
          break;
        case 4:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgDomainFormationFreeze();
    message.authority = object.authority ?? "";
    message.domain = object.domain ?? "";
    message.durationBlocks = object.durationBlocks !== void 0 && object.durationBlocks !== null ? BigInt(object.durationBlocks.toString()) : BigInt(0);
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgDomainFormationFreezeResponse() {
  return {};
}
var MsgDomainFormationFreezeResponse = {
  typeUrl: "/zerone.gov.v1.MsgDomainFormationFreezeResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgDomainFormationFreezeResponse();
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
    const message = createBaseMsgDomainFormationFreezeResponse();
    return message;
  }
};

// src/generated/zerone/gov/v1/tx.registry.ts
var registry9 = [["/zerone.gov.v1.MsgSubmitLIP", MsgSubmitLIP], ["/zerone.gov.v1.MsgStakeLIP", MsgStakeLIP], ["/zerone.gov.v1.MsgAdvanceLIPStage", MsgAdvanceLIPStage], ["/zerone.gov.v1.MsgCastVote", MsgCastVote], ["/zerone.gov.v1.MsgWithdrawLIP", MsgWithdrawLIP], ["/zerone.gov.v1.MsgUpdateParams", MsgUpdateParams8], ["/zerone.gov.v1.MsgSubmitResearchSpend", MsgSubmitResearchSpend], ["/zerone.gov.v1.MsgVoteResearchSpend", MsgVoteResearchSpend], ["/zerone.gov.v1.MsgSetResearchVoters", MsgSetResearchVoters], ["/zerone.gov.v1.MsgAttachUpgradePlan", MsgAttachUpgradePlan], ["/zerone.gov.v1.MsgAttachCreedAmendmentPin", MsgAttachCreedAmendmentPin], ["/zerone.gov.v1.MsgNominateSeatElection", MsgNominateSeatElection], ["/zerone.gov.v1.MsgAcceptSeatNomination", MsgAcceptSeatNomination], ["/zerone.gov.v1.MsgVoteSeatElection", MsgVoteSeatElection], ["/zerone.gov.v1.MsgDomainFormationFreeze", MsgDomainFormationFreeze]];
var MessageComposer9 = {
  encoded: {
    submitLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitLIP",
        value: MsgSubmitLIP.encode(value).finish()
      };
    },
    stakeLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgStakeLIP",
        value: MsgStakeLIP.encode(value).finish()
      };
    },
    advanceLIPStage(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStage",
        value: MsgAdvanceLIPStage.encode(value).finish()
      };
    },
    castVote(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgCastVote",
        value: MsgCastVote.encode(value).finish()
      };
    },
    withdrawLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgWithdrawLIP",
        value: MsgWithdrawLIP.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgUpdateParams",
        value: MsgUpdateParams8.encode(value).finish()
      };
    },
    submitResearchSpend(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpend",
        value: MsgSubmitResearchSpend.encode(value).finish()
      };
    },
    voteResearchSpend(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteResearchSpend",
        value: MsgVoteResearchSpend.encode(value).finish()
      };
    },
    setResearchVoters(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSetResearchVoters",
        value: MsgSetResearchVoters.encode(value).finish()
      };
    },
    attachUpgradePlan(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlan",
        value: MsgAttachUpgradePlan.encode(value).finish()
      };
    },
    attachCreedAmendmentPin(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPin",
        value: MsgAttachCreedAmendmentPin.encode(value).finish()
      };
    },
    nominateSeatElection(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgNominateSeatElection",
        value: MsgNominateSeatElection.encode(value).finish()
      };
    },
    acceptSeatNomination(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAcceptSeatNomination",
        value: MsgAcceptSeatNomination.encode(value).finish()
      };
    },
    voteSeatElection(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteSeatElection",
        value: MsgVoteSeatElection.encode(value).finish()
      };
    },
    domainFormationFreeze(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgDomainFormationFreeze",
        value: MsgDomainFormationFreeze.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    submitLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitLIP",
        value
      };
    },
    stakeLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgStakeLIP",
        value
      };
    },
    advanceLIPStage(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStage",
        value
      };
    },
    castVote(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgCastVote",
        value
      };
    },
    withdrawLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgWithdrawLIP",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgUpdateParams",
        value
      };
    },
    submitResearchSpend(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpend",
        value
      };
    },
    voteResearchSpend(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteResearchSpend",
        value
      };
    },
    setResearchVoters(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSetResearchVoters",
        value
      };
    },
    attachUpgradePlan(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlan",
        value
      };
    },
    attachCreedAmendmentPin(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPin",
        value
      };
    },
    nominateSeatElection(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgNominateSeatElection",
        value
      };
    },
    acceptSeatNomination(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAcceptSeatNomination",
        value
      };
    },
    voteSeatElection(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteSeatElection",
        value
      };
    },
    domainFormationFreeze(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgDomainFormationFreeze",
        value
      };
    }
  },
  fromPartial: {
    submitLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitLIP",
        value: MsgSubmitLIP.fromPartial(value)
      };
    },
    stakeLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgStakeLIP",
        value: MsgStakeLIP.fromPartial(value)
      };
    },
    advanceLIPStage(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStage",
        value: MsgAdvanceLIPStage.fromPartial(value)
      };
    },
    castVote(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgCastVote",
        value: MsgCastVote.fromPartial(value)
      };
    },
    withdrawLIP(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgWithdrawLIP",
        value: MsgWithdrawLIP.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgUpdateParams",
        value: MsgUpdateParams8.fromPartial(value)
      };
    },
    submitResearchSpend(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpend",
        value: MsgSubmitResearchSpend.fromPartial(value)
      };
    },
    voteResearchSpend(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteResearchSpend",
        value: MsgVoteResearchSpend.fromPartial(value)
      };
    },
    setResearchVoters(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgSetResearchVoters",
        value: MsgSetResearchVoters.fromPartial(value)
      };
    },
    attachUpgradePlan(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlan",
        value: MsgAttachUpgradePlan.fromPartial(value)
      };
    },
    attachCreedAmendmentPin(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPin",
        value: MsgAttachCreedAmendmentPin.fromPartial(value)
      };
    },
    nominateSeatElection(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgNominateSeatElection",
        value: MsgNominateSeatElection.fromPartial(value)
      };
    },
    acceptSeatNomination(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgAcceptSeatNomination",
        value: MsgAcceptSeatNomination.fromPartial(value)
      };
    },
    voteSeatElection(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgVoteSeatElection",
        value: MsgVoteSeatElection.fromPartial(value)
      };
    },
    domainFormationFreeze(value) {
      return {
        typeUrl: "/zerone.gov.v1.MsgDomainFormationFreeze",
        value: MsgDomainFormationFreeze.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/home/v1/tx.ts
var tx_exports10 = {};
__export(tx_exports10, {
  MsgAcknowledgeAlert: () => MsgAcknowledgeAlert,
  MsgAcknowledgeAlertResponse: () => MsgAcknowledgeAlertResponse,
  MsgConfigureGuardian: () => MsgConfigureGuardian,
  MsgConfigureGuardianResponse: () => MsgConfigureGuardianResponse,
  MsgCreateHome: () => MsgCreateHome,
  MsgCreateHomeResponse: () => MsgCreateHomeResponse,
  MsgEndSession: () => MsgEndSession,
  MsgEndSessionResponse: () => MsgEndSessionResponse,
  MsgRegisterKey: () => MsgRegisterKey,
  MsgRegisterKeyResponse: () => MsgRegisterKeyResponse,
  MsgRevokeKey: () => MsgRevokeKey,
  MsgRevokeKeyResponse: () => MsgRevokeKeyResponse,
  MsgSetSpendingLimit: () => MsgSetSpendingLimit,
  MsgSetSpendingLimitResponse: () => MsgSetSpendingLimitResponse,
  MsgStartSession: () => MsgStartSession,
  MsgStartSessionResponse: () => MsgStartSessionResponse,
  MsgUpdateHome: () => MsgUpdateHome,
  MsgUpdateHomeResponse: () => MsgUpdateHomeResponse,
  MsgUpdateMemoryCID: () => MsgUpdateMemoryCID,
  MsgUpdateMemoryCIDResponse: () => MsgUpdateMemoryCIDResponse,
  MsgUpdateParams: () => MsgUpdateParams9,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse9
});

// src/generated/zerone/home/v1/types.ts
function createBaseHomeGuardian() {
  return {
    defenseStrategy: "",
    autoDefend: false,
    deadman: void 0,
    recoveryAddresses: [],
    recoveryThreshold: 0,
    guardianAddress: ""
  };
}
var HomeGuardian = {
  typeUrl: "/zerone.home.v1.HomeGuardian",
  encode(message, writer = BinaryWriter.create()) {
    if (message.defenseStrategy !== "") {
      writer.uint32(10).string(message.defenseStrategy);
    }
    if (message.autoDefend === true) {
      writer.uint32(16).bool(message.autoDefend);
    }
    if (message.deadman !== void 0) {
      DeadmanConfig.encode(message.deadman, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.recoveryAddresses) {
      writer.uint32(34).string(v);
    }
    if (message.recoveryThreshold !== 0) {
      writer.uint32(40).uint32(message.recoveryThreshold);
    }
    if (message.guardianAddress !== "") {
      writer.uint32(50).string(message.guardianAddress);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseHomeGuardian();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.defenseStrategy = reader.string();
          break;
        case 2:
          message.autoDefend = reader.bool();
          break;
        case 3:
          message.deadman = DeadmanConfig.decode(reader, reader.uint32());
          break;
        case 4:
          message.recoveryAddresses.push(reader.string());
          break;
        case 5:
          message.recoveryThreshold = reader.uint32();
          break;
        case 6:
          message.guardianAddress = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseHomeGuardian();
    message.defenseStrategy = object.defenseStrategy ?? "";
    message.autoDefend = object.autoDefend ?? false;
    message.deadman = object.deadman !== void 0 && object.deadman !== null ? DeadmanConfig.fromPartial(object.deadman) : void 0;
    message.recoveryAddresses = object.recoveryAddresses?.map((e) => e) || [];
    message.recoveryThreshold = object.recoveryThreshold ?? 0;
    message.guardianAddress = object.guardianAddress ?? "";
    return message;
  }
};
function createBaseDeadmanConfig() {
  return {
    enabled: false,
    inactivityThreshold: BigInt(0),
    action: "",
    beneficiaryAddress: ""
  };
}
var DeadmanConfig = {
  typeUrl: "/zerone.home.v1.DeadmanConfig",
  encode(message, writer = BinaryWriter.create()) {
    if (message.enabled === true) {
      writer.uint32(8).bool(message.enabled);
    }
    if (message.inactivityThreshold !== BigInt(0)) {
      writer.uint32(16).uint64(message.inactivityThreshold);
    }
    if (message.action !== "") {
      writer.uint32(26).string(message.action);
    }
    if (message.beneficiaryAddress !== "") {
      writer.uint32(34).string(message.beneficiaryAddress);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseDeadmanConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.enabled = reader.bool();
          break;
        case 2:
          message.inactivityThreshold = reader.uint64();
          break;
        case 3:
          message.action = reader.string();
          break;
        case 4:
          message.beneficiaryAddress = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseDeadmanConfig();
    message.enabled = object.enabled ?? false;
    message.inactivityThreshold = object.inactivityThreshold !== void 0 && object.inactivityThreshold !== null ? BigInt(object.inactivityThreshold.toString()) : BigInt(0);
    message.action = object.action ?? "";
    message.beneficiaryAddress = object.beneficiaryAddress ?? "";
    return message;
  }
};

// src/generated/zerone/home/v1/genesis.ts
function createBaseParams10() {
  return {
    maxKeysPerHome: BigInt(0),
    maxSessionsPerHome: BigInt(0),
    sessionTimeoutBlocks: BigInt(0),
    deadmanMinThreshold: BigInt(0),
    deadmanMaxThreshold: BigInt(0),
    maxAlertsPerHome: BigInt(0),
    homeCreationFee: "",
    maxRecoveryAddresses: BigInt(0)
  };
}
var Params10 = {
  typeUrl: "/zerone.home.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.maxKeysPerHome !== BigInt(0)) {
      writer.uint32(8).uint64(message.maxKeysPerHome);
    }
    if (message.maxSessionsPerHome !== BigInt(0)) {
      writer.uint32(16).uint64(message.maxSessionsPerHome);
    }
    if (message.sessionTimeoutBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.sessionTimeoutBlocks);
    }
    if (message.deadmanMinThreshold !== BigInt(0)) {
      writer.uint32(32).uint64(message.deadmanMinThreshold);
    }
    if (message.deadmanMaxThreshold !== BigInt(0)) {
      writer.uint32(40).uint64(message.deadmanMaxThreshold);
    }
    if (message.maxAlertsPerHome !== BigInt(0)) {
      writer.uint32(48).uint64(message.maxAlertsPerHome);
    }
    if (message.homeCreationFee !== "") {
      writer.uint32(58).string(message.homeCreationFee);
    }
    if (message.maxRecoveryAddresses !== BigInt(0)) {
      writer.uint32(64).uint64(message.maxRecoveryAddresses);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams10();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.maxKeysPerHome = reader.uint64();
          break;
        case 2:
          message.maxSessionsPerHome = reader.uint64();
          break;
        case 3:
          message.sessionTimeoutBlocks = reader.uint64();
          break;
        case 4:
          message.deadmanMinThreshold = reader.uint64();
          break;
        case 5:
          message.deadmanMaxThreshold = reader.uint64();
          break;
        case 6:
          message.maxAlertsPerHome = reader.uint64();
          break;
        case 7:
          message.homeCreationFee = reader.string();
          break;
        case 8:
          message.maxRecoveryAddresses = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams10();
    message.maxKeysPerHome = object.maxKeysPerHome !== void 0 && object.maxKeysPerHome !== null ? BigInt(object.maxKeysPerHome.toString()) : BigInt(0);
    message.maxSessionsPerHome = object.maxSessionsPerHome !== void 0 && object.maxSessionsPerHome !== null ? BigInt(object.maxSessionsPerHome.toString()) : BigInt(0);
    message.sessionTimeoutBlocks = object.sessionTimeoutBlocks !== void 0 && object.sessionTimeoutBlocks !== null ? BigInt(object.sessionTimeoutBlocks.toString()) : BigInt(0);
    message.deadmanMinThreshold = object.deadmanMinThreshold !== void 0 && object.deadmanMinThreshold !== null ? BigInt(object.deadmanMinThreshold.toString()) : BigInt(0);
    message.deadmanMaxThreshold = object.deadmanMaxThreshold !== void 0 && object.deadmanMaxThreshold !== null ? BigInt(object.deadmanMaxThreshold.toString()) : BigInt(0);
    message.maxAlertsPerHome = object.maxAlertsPerHome !== void 0 && object.maxAlertsPerHome !== null ? BigInt(object.maxAlertsPerHome.toString()) : BigInt(0);
    message.homeCreationFee = object.homeCreationFee ?? "";
    message.maxRecoveryAddresses = object.maxRecoveryAddresses !== void 0 && object.maxRecoveryAddresses !== null ? BigInt(object.maxRecoveryAddresses.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/home/v1/tx.ts
function createBaseMsgCreateHome() {
  return {
    owner: "",
    name: "",
    initialGuardianConfig: void 0
  };
}
var MsgCreateHome = {
  typeUrl: "/zerone.home.v1.MsgCreateHome",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.initialGuardianConfig !== void 0) {
      HomeGuardian.encode(message.initialGuardianConfig, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateHome();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.initialGuardianConfig = HomeGuardian.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateHome();
    message.owner = object.owner ?? "";
    message.name = object.name ?? "";
    message.initialGuardianConfig = object.initialGuardianConfig !== void 0 && object.initialGuardianConfig !== null ? HomeGuardian.fromPartial(object.initialGuardianConfig) : void 0;
    return message;
  }
};
function createBaseMsgCreateHomeResponse() {
  return {
    homeId: ""
  };
}
var MsgCreateHomeResponse = {
  typeUrl: "/zerone.home.v1.MsgCreateHomeResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.homeId !== "") {
      writer.uint32(10).string(message.homeId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateHomeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.homeId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateHomeResponse();
    message.homeId = object.homeId ?? "";
    return message;
  }
};
function createBaseMsgUpdateHome() {
  return {
    owner: "",
    homeId: "",
    name: "",
    status: ""
  };
}
var MsgUpdateHome = {
  typeUrl: "/zerone.home.v1.MsgUpdateHome",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    if (message.status !== "") {
      writer.uint32(34).string(message.status);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateHome();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.name = reader.string();
          break;
        case 4:
          message.status = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateHome();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.name = object.name ?? "";
    message.status = object.status ?? "";
    return message;
  }
};
function createBaseMsgUpdateHomeResponse() {
  return {};
}
var MsgUpdateHomeResponse = {
  typeUrl: "/zerone.home.v1.MsgUpdateHomeResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateHomeResponse();
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
    const message = createBaseMsgUpdateHomeResponse();
    return message;
  }
};
function createBaseMsgUpdateMemoryCID() {
  return {
    owner: "",
    homeId: "",
    cid: ""
  };
}
var MsgUpdateMemoryCID = {
  typeUrl: "/zerone.home.v1.MsgUpdateMemoryCID",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.cid !== "") {
      writer.uint32(26).string(message.cid);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateMemoryCID();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.cid = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateMemoryCID();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.cid = object.cid ?? "";
    return message;
  }
};
function createBaseMsgUpdateMemoryCIDResponse() {
  return {};
}
var MsgUpdateMemoryCIDResponse = {
  typeUrl: "/zerone.home.v1.MsgUpdateMemoryCIDResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateMemoryCIDResponse();
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
    const message = createBaseMsgUpdateMemoryCIDResponse();
    return message;
  }
};
function createBaseMsgStartSession() {
  return {
    signer: "",
    homeId: "",
    keyHash: "",
    requestedPermissions: []
  };
}
var MsgStartSession = {
  typeUrl: "/zerone.home.v1.MsgStartSession",
  encode(message, writer = BinaryWriter.create()) {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyHash !== "") {
      writer.uint32(26).string(message.keyHash);
    }
    for (const v of message.requestedPermissions) {
      writer.uint32(34).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgStartSession();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyHash = reader.string();
          break;
        case 4:
          message.requestedPermissions.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgStartSession();
    message.signer = object.signer ?? "";
    message.homeId = object.homeId ?? "";
    message.keyHash = object.keyHash ?? "";
    message.requestedPermissions = object.requestedPermissions?.map((e) => e) || [];
    return message;
  }
};
function createBaseMsgStartSessionResponse() {
  return {
    sessionId: ""
  };
}
var MsgStartSessionResponse = {
  typeUrl: "/zerone.home.v1.MsgStartSessionResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sessionId !== "") {
      writer.uint32(10).string(message.sessionId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgStartSessionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sessionId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgStartSessionResponse();
    message.sessionId = object.sessionId ?? "";
    return message;
  }
};
function createBaseMsgEndSession() {
  return {
    signer: "",
    homeId: "",
    sessionId: ""
  };
}
var MsgEndSession = {
  typeUrl: "/zerone.home.v1.MsgEndSession",
  encode(message, writer = BinaryWriter.create()) {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.sessionId !== "") {
      writer.uint32(26).string(message.sessionId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgEndSession();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.sessionId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgEndSession();
    message.signer = object.signer ?? "";
    message.homeId = object.homeId ?? "";
    message.sessionId = object.sessionId ?? "";
    return message;
  }
};
function createBaseMsgEndSessionResponse() {
  return {};
}
var MsgEndSessionResponse = {
  typeUrl: "/zerone.home.v1.MsgEndSessionResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgEndSessionResponse();
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
    const message = createBaseMsgEndSessionResponse();
    return message;
  }
};
function createBaseMsgRegisterKey() {
  return {
    owner: "",
    homeId: "",
    keyHash: "",
    keyType: "",
    role: "",
    permissions: [],
    expiresAt: BigInt(0)
  };
}
var MsgRegisterKey = {
  typeUrl: "/zerone.home.v1.MsgRegisterKey",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyHash !== "") {
      writer.uint32(26).string(message.keyHash);
    }
    if (message.keyType !== "") {
      writer.uint32(34).string(message.keyType);
    }
    if (message.role !== "") {
      writer.uint32(42).string(message.role);
    }
    for (const v of message.permissions) {
      writer.uint32(50).string(v);
    }
    if (message.expiresAt !== BigInt(0)) {
      writer.uint32(56).uint64(message.expiresAt);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterKey();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyHash = reader.string();
          break;
        case 4:
          message.keyType = reader.string();
          break;
        case 5:
          message.role = reader.string();
          break;
        case 6:
          message.permissions.push(reader.string());
          break;
        case 7:
          message.expiresAt = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterKey();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.keyHash = object.keyHash ?? "";
    message.keyType = object.keyType ?? "";
    message.role = object.role ?? "";
    message.permissions = object.permissions?.map((e) => e) || [];
    message.expiresAt = object.expiresAt !== void 0 && object.expiresAt !== null ? BigInt(object.expiresAt.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgRegisterKeyResponse() {
  return {};
}
var MsgRegisterKeyResponse = {
  typeUrl: "/zerone.home.v1.MsgRegisterKeyResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterKeyResponse();
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
    const message = createBaseMsgRegisterKeyResponse();
    return message;
  }
};
function createBaseMsgRevokeKey() {
  return {
    owner: "",
    homeId: "",
    keyHash: ""
  };
}
var MsgRevokeKey = {
  typeUrl: "/zerone.home.v1.MsgRevokeKey",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyHash !== "") {
      writer.uint32(26).string(message.keyHash);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRevokeKey();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyHash = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRevokeKey();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.keyHash = object.keyHash ?? "";
    return message;
  }
};
function createBaseMsgRevokeKeyResponse() {
  return {};
}
var MsgRevokeKeyResponse = {
  typeUrl: "/zerone.home.v1.MsgRevokeKeyResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRevokeKeyResponse();
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
    const message = createBaseMsgRevokeKeyResponse();
    return message;
  }
};
function createBaseMsgConfigureGuardian() {
  return {
    owner: "",
    homeId: "",
    defenseStrategy: "",
    autoDefend: false,
    deadman: void 0,
    recoveryAddresses: [],
    recoveryThreshold: 0,
    guardianAddress: ""
  };
}
var MsgConfigureGuardian = {
  typeUrl: "/zerone.home.v1.MsgConfigureGuardian",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.defenseStrategy !== "") {
      writer.uint32(26).string(message.defenseStrategy);
    }
    if (message.autoDefend === true) {
      writer.uint32(32).bool(message.autoDefend);
    }
    if (message.deadman !== void 0) {
      DeadmanConfig.encode(message.deadman, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.recoveryAddresses) {
      writer.uint32(50).string(v);
    }
    if (message.recoveryThreshold !== 0) {
      writer.uint32(56).uint32(message.recoveryThreshold);
    }
    if (message.guardianAddress !== "") {
      writer.uint32(66).string(message.guardianAddress);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgConfigureGuardian();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.defenseStrategy = reader.string();
          break;
        case 4:
          message.autoDefend = reader.bool();
          break;
        case 5:
          message.deadman = DeadmanConfig.decode(reader, reader.uint32());
          break;
        case 6:
          message.recoveryAddresses.push(reader.string());
          break;
        case 7:
          message.recoveryThreshold = reader.uint32();
          break;
        case 8:
          message.guardianAddress = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgConfigureGuardian();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.defenseStrategy = object.defenseStrategy ?? "";
    message.autoDefend = object.autoDefend ?? false;
    message.deadman = object.deadman !== void 0 && object.deadman !== null ? DeadmanConfig.fromPartial(object.deadman) : void 0;
    message.recoveryAddresses = object.recoveryAddresses?.map((e) => e) || [];
    message.recoveryThreshold = object.recoveryThreshold ?? 0;
    message.guardianAddress = object.guardianAddress ?? "";
    return message;
  }
};
function createBaseMsgConfigureGuardianResponse() {
  return {};
}
var MsgConfigureGuardianResponse = {
  typeUrl: "/zerone.home.v1.MsgConfigureGuardianResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgConfigureGuardianResponse();
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
    const message = createBaseMsgConfigureGuardianResponse();
    return message;
  }
};
function createBaseMsgAcknowledgeAlert() {
  return {
    signer: "",
    homeId: "",
    alertId: ""
  };
}
var MsgAcknowledgeAlert = {
  typeUrl: "/zerone.home.v1.MsgAcknowledgeAlert",
  encode(message, writer = BinaryWriter.create()) {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.alertId !== "") {
      writer.uint32(26).string(message.alertId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAcknowledgeAlert();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.alertId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAcknowledgeAlert();
    message.signer = object.signer ?? "";
    message.homeId = object.homeId ?? "";
    message.alertId = object.alertId ?? "";
    return message;
  }
};
function createBaseMsgAcknowledgeAlertResponse() {
  return {};
}
var MsgAcknowledgeAlertResponse = {
  typeUrl: "/zerone.home.v1.MsgAcknowledgeAlertResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAcknowledgeAlertResponse();
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
    const message = createBaseMsgAcknowledgeAlertResponse();
    return message;
  }
};
function createBaseMsgSetSpendingLimit() {
  return {
    owner: "",
    homeId: "",
    keyType: "",
    maxAmount: "",
    periodBlocks: BigInt(0)
  };
}
var MsgSetSpendingLimit = {
  typeUrl: "/zerone.home.v1.MsgSetSpendingLimit",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyType !== "") {
      writer.uint32(26).string(message.keyType);
    }
    if (message.maxAmount !== "") {
      writer.uint32(34).string(message.maxAmount);
    }
    if (message.periodBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.periodBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSetSpendingLimit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyType = reader.string();
          break;
        case 4:
          message.maxAmount = reader.string();
          break;
        case 5:
          message.periodBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSetSpendingLimit();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.keyType = object.keyType ?? "";
    message.maxAmount = object.maxAmount ?? "";
    message.periodBlocks = object.periodBlocks !== void 0 && object.periodBlocks !== null ? BigInt(object.periodBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgSetSpendingLimitResponse() {
  return {};
}
var MsgSetSpendingLimitResponse = {
  typeUrl: "/zerone.home.v1.MsgSetSpendingLimitResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSetSpendingLimitResponse();
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
    const message = createBaseMsgSetSpendingLimitResponse();
    return message;
  }
};
function createBaseMsgUpdateParams9() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams9 = {
  typeUrl: "/zerone.home.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params10.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams9();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params10.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams9();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params10.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse9() {
  return {};
}
var MsgUpdateParamsResponse9 = {
  typeUrl: "/zerone.home.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse9();
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
    const message = createBaseMsgUpdateParamsResponse9();
    return message;
  }
};

// src/generated/zerone/home/v1/tx.registry.ts
var registry10 = [["/zerone.home.v1.MsgCreateHome", MsgCreateHome], ["/zerone.home.v1.MsgUpdateHome", MsgUpdateHome], ["/zerone.home.v1.MsgUpdateMemoryCID", MsgUpdateMemoryCID], ["/zerone.home.v1.MsgStartSession", MsgStartSession], ["/zerone.home.v1.MsgEndSession", MsgEndSession], ["/zerone.home.v1.MsgRegisterKey", MsgRegisterKey], ["/zerone.home.v1.MsgRevokeKey", MsgRevokeKey], ["/zerone.home.v1.MsgConfigureGuardian", MsgConfigureGuardian], ["/zerone.home.v1.MsgAcknowledgeAlert", MsgAcknowledgeAlert], ["/zerone.home.v1.MsgSetSpendingLimit", MsgSetSpendingLimit], ["/zerone.home.v1.MsgUpdateParams", MsgUpdateParams9]];
var MessageComposer10 = {
  encoded: {
    createHome(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgCreateHome",
        value: MsgCreateHome.encode(value).finish()
      };
    },
    updateHome(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateHome",
        value: MsgUpdateHome.encode(value).finish()
      };
    },
    updateMemoryCID(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateMemoryCID",
        value: MsgUpdateMemoryCID.encode(value).finish()
      };
    },
    startSession(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgStartSession",
        value: MsgStartSession.encode(value).finish()
      };
    },
    endSession(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgEndSession",
        value: MsgEndSession.encode(value).finish()
      };
    },
    registerKey(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgRegisterKey",
        value: MsgRegisterKey.encode(value).finish()
      };
    },
    revokeKey(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgRevokeKey",
        value: MsgRevokeKey.encode(value).finish()
      };
    },
    configureGuardian(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgConfigureGuardian",
        value: MsgConfigureGuardian.encode(value).finish()
      };
    },
    acknowledgeAlert(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgAcknowledgeAlert",
        value: MsgAcknowledgeAlert.encode(value).finish()
      };
    },
    setSpendingLimit(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgSetSpendingLimit",
        value: MsgSetSpendingLimit.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateParams",
        value: MsgUpdateParams9.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createHome(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgCreateHome",
        value
      };
    },
    updateHome(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateHome",
        value
      };
    },
    updateMemoryCID(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateMemoryCID",
        value
      };
    },
    startSession(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgStartSession",
        value
      };
    },
    endSession(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgEndSession",
        value
      };
    },
    registerKey(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgRegisterKey",
        value
      };
    },
    revokeKey(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgRevokeKey",
        value
      };
    },
    configureGuardian(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgConfigureGuardian",
        value
      };
    },
    acknowledgeAlert(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgAcknowledgeAlert",
        value
      };
    },
    setSpendingLimit(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgSetSpendingLimit",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createHome(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgCreateHome",
        value: MsgCreateHome.fromPartial(value)
      };
    },
    updateHome(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateHome",
        value: MsgUpdateHome.fromPartial(value)
      };
    },
    updateMemoryCID(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateMemoryCID",
        value: MsgUpdateMemoryCID.fromPartial(value)
      };
    },
    startSession(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgStartSession",
        value: MsgStartSession.fromPartial(value)
      };
    },
    endSession(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgEndSession",
        value: MsgEndSession.fromPartial(value)
      };
    },
    registerKey(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgRegisterKey",
        value: MsgRegisterKey.fromPartial(value)
      };
    },
    revokeKey(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgRevokeKey",
        value: MsgRevokeKey.fromPartial(value)
      };
    },
    configureGuardian(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgConfigureGuardian",
        value: MsgConfigureGuardian.fromPartial(value)
      };
    },
    acknowledgeAlert(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgAcknowledgeAlert",
        value: MsgAcknowledgeAlert.fromPartial(value)
      };
    },
    setSpendingLimit(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgSetSpendingLimit",
        value: MsgSetSpendingLimit.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.home.v1.MsgUpdateParams",
        value: MsgUpdateParams9.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/ibcratelimit/v1/tx.ts
var tx_exports11 = {};
__export(tx_exports11, {
  MsgAddRateLimit: () => MsgAddRateLimit,
  MsgAddRateLimitResponse: () => MsgAddRateLimitResponse,
  MsgRemoveRateLimit: () => MsgRemoveRateLimit,
  MsgRemoveRateLimitResponse: () => MsgRemoveRateLimitResponse,
  MsgUpdateParams: () => MsgUpdateParams10,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse10
});

// src/generated/zerone/ibcratelimit/v1/genesis.ts
function createBaseParams11() {
  return {
    enabled: false
  };
}
var Params11 = {
  typeUrl: "/zerone.ibcratelimit.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.enabled === true) {
      writer.uint32(8).bool(message.enabled);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams11();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.enabled = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams11();
    message.enabled = object.enabled ?? false;
    return message;
  }
};

// src/generated/zerone/ibcratelimit/v1/tx.ts
function createBaseMsgAddRateLimit() {
  return {
    authority: "",
    channelId: "",
    denom: "",
    maxSend: "",
    maxRecv: "",
    windowBlocks: BigInt(0)
  };
}
var MsgAddRateLimit = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimit",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.channelId !== "") {
      writer.uint32(18).string(message.channelId);
    }
    if (message.denom !== "") {
      writer.uint32(26).string(message.denom);
    }
    if (message.maxSend !== "") {
      writer.uint32(34).string(message.maxSend);
    }
    if (message.maxRecv !== "") {
      writer.uint32(42).string(message.maxRecv);
    }
    if (message.windowBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.windowBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddRateLimit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.channelId = reader.string();
          break;
        case 3:
          message.denom = reader.string();
          break;
        case 4:
          message.maxSend = reader.string();
          break;
        case 5:
          message.maxRecv = reader.string();
          break;
        case 6:
          message.windowBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddRateLimit();
    message.authority = object.authority ?? "";
    message.channelId = object.channelId ?? "";
    message.denom = object.denom ?? "";
    message.maxSend = object.maxSend ?? "";
    message.maxRecv = object.maxRecv ?? "";
    message.windowBlocks = object.windowBlocks !== void 0 && object.windowBlocks !== null ? BigInt(object.windowBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAddRateLimitResponse() {
  return {};
}
var MsgAddRateLimitResponse = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimitResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddRateLimitResponse();
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
    const message = createBaseMsgAddRateLimitResponse();
    return message;
  }
};
function createBaseMsgRemoveRateLimit() {
  return {
    authority: "",
    channelId: "",
    denom: ""
  };
}
var MsgRemoveRateLimit = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimit",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.channelId !== "") {
      writer.uint32(18).string(message.channelId);
    }
    if (message.denom !== "") {
      writer.uint32(26).string(message.denom);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveRateLimit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.channelId = reader.string();
          break;
        case 3:
          message.denom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRemoveRateLimit();
    message.authority = object.authority ?? "";
    message.channelId = object.channelId ?? "";
    message.denom = object.denom ?? "";
    return message;
  }
};
function createBaseMsgRemoveRateLimitResponse() {
  return {};
}
var MsgRemoveRateLimitResponse = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimitResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveRateLimitResponse();
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
    const message = createBaseMsgRemoveRateLimitResponse();
    return message;
  }
};
function createBaseMsgUpdateParams10() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams10 = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params11.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams10();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params11.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams10();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params11.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse10() {
  return {};
}
var MsgUpdateParamsResponse10 = {
  typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse10();
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
    const message = createBaseMsgUpdateParamsResponse10();
    return message;
  }
};

// src/generated/zerone/ibcratelimit/v1/tx.registry.ts
var registry11 = [["/zerone.ibcratelimit.v1.MsgAddRateLimit", MsgAddRateLimit], ["/zerone.ibcratelimit.v1.MsgRemoveRateLimit", MsgRemoveRateLimit], ["/zerone.ibcratelimit.v1.MsgUpdateParams", MsgUpdateParams10]];
var MessageComposer11 = {
  encoded: {
    addRateLimit(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimit",
        value: MsgAddRateLimit.encode(value).finish()
      };
    },
    removeRateLimit(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimit",
        value: MsgRemoveRateLimit.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParams",
        value: MsgUpdateParams10.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    addRateLimit(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimit",
        value
      };
    },
    removeRateLimit(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimit",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    addRateLimit(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgAddRateLimit",
        value: MsgAddRateLimit.fromPartial(value)
      };
    },
    removeRateLimit(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgRemoveRateLimit",
        value: MsgRemoveRateLimit.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.ibcratelimit.v1.MsgUpdateParams",
        value: MsgUpdateParams10.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/knowledge/v1/tx.ts
var tx_exports12 = {};
__export(tx_exports12, {
  DemandReport: () => DemandReport,
  MsgAcceptAugmentation: () => MsgAcceptAugmentation,
  MsgAcceptAugmentationResponse: () => MsgAcceptAugmentationResponse,
  MsgAddCommonKnowledge: () => MsgAddCommonKnowledge,
  MsgAddCommonKnowledgeResponse: () => MsgAddCommonKnowledgeResponse,
  MsgAddFact: () => MsgAddFact,
  MsgAddFactResponse: () => MsgAddFactResponse,
  MsgAmendTokenizerSpec: () => MsgAmendTokenizerSpec,
  MsgAmendTokenizerSpecResponse: () => MsgAmendTokenizerSpecResponse,
  MsgAmendTraceSchema: () => MsgAmendTraceSchema,
  MsgAmendTraceSchemaResponse: () => MsgAmendTraceSchemaResponse,
  MsgAttestTraining: () => MsgAttestTraining,
  MsgAttestTrainingResponse: () => MsgAttestTrainingResponse,
  MsgAttributeContributions: () => MsgAttributeContributions,
  MsgAttributeContributionsResponse: () => MsgAttributeContributionsResponse,
  MsgBindManifestToAttestation: () => MsgBindManifestToAttestation,
  MsgBindManifestToAttestationResponse: () => MsgBindManifestToAttestationResponse,
  MsgChallengeContribution: () => MsgChallengeContribution,
  MsgChallengeContributionResponse: () => MsgChallengeContributionResponse,
  MsgChallengeDomainProposal: () => MsgChallengeDomainProposal,
  MsgChallengeDomainProposalResponse: () => MsgChallengeDomainProposalResponse,
  MsgChallengeFact: () => MsgChallengeFact,
  MsgChallengeFactResponse: () => MsgChallengeFactResponse,
  MsgChallengeProvisionalFact: () => MsgChallengeProvisionalFact,
  MsgChallengeProvisionalFactResponse: () => MsgChallengeProvisionalFactResponse,
  MsgClaimTrainingFundDisbursement: () => MsgClaimTrainingFundDisbursement,
  MsgClaimTrainingFundDisbursementResponse: () => MsgClaimTrainingFundDisbursementResponse,
  MsgCloseIncident: () => MsgCloseIncident,
  MsgCloseIncidentResponse: () => MsgCloseIncidentResponse,
  MsgCorrectManifestMerkleRoot: () => MsgCorrectManifestMerkleRoot,
  MsgCorrectManifestMerkleRootResponse: () => MsgCorrectManifestMerkleRootResponse,
  MsgCreateAugmentationBounty: () => MsgCreateAugmentationBounty,
  MsgCreateAugmentationBountyResponse: () => MsgCreateAugmentationBountyResponse,
  MsgCreateTrainingManifest: () => MsgCreateTrainingManifest,
  MsgCreateTrainingManifestResponse: () => MsgCreateTrainingManifestResponse,
  MsgEndorseDomainProposal: () => MsgEndorseDomainProposal,
  MsgEndorseDomainProposalResponse: () => MsgEndorseDomainProposalResponse,
  MsgExecuteResearchProposal: () => MsgExecuteResearchProposal,
  MsgExecuteResearchProposalResponse: () => MsgExecuteResearchProposalResponse,
  MsgFinalizeTrainingManifest: () => MsgFinalizeTrainingManifest,
  MsgFinalizeTrainingManifestResponse: () => MsgFinalizeTrainingManifestResponse,
  MsgOpenIncident: () => MsgOpenIncident,
  MsgOpenIncidentResponse: () => MsgOpenIncidentResponse,
  MsgPatronizeFact: () => MsgPatronizeFact,
  MsgPatronizeFactResponse: () => MsgPatronizeFactResponse,
  MsgPauseModule: () => MsgPauseModule,
  MsgPauseModuleResponse: () => MsgPauseModuleResponse,
  MsgProposeDomain: () => MsgProposeDomain,
  MsgProposeDomainResponse: () => MsgProposeDomainResponse,
  MsgProposeResearchFund: () => MsgProposeResearchFund,
  MsgProposeResearchFundResponse: () => MsgProposeResearchFundResponse,
  MsgRateFact: () => MsgRateFact,
  MsgRateFactResponse: () => MsgRateFactResponse,
  MsgRecordRemediation: () => MsgRecordRemediation,
  MsgRecordRemediationResponse: () => MsgRecordRemediationResponse,
  MsgRegisterModelCard: () => MsgRegisterModelCard,
  MsgRegisterModelCardResponse: () => MsgRegisterModelCardResponse,
  MsgRegisterStratum: () => MsgRegisterStratum,
  MsgRegisterStratumResponse: () => MsgRegisterStratumResponse,
  MsgRegisterTrainingPipeline: () => MsgRegisterTrainingPipeline,
  MsgRegisterTrainingPipelineResponse: () => MsgRegisterTrainingPipelineResponse,
  MsgRemoveCommonKnowledge: () => MsgRemoveCommonKnowledge,
  MsgRemoveCommonKnowledgeResponse: () => MsgRemoveCommonKnowledgeResponse,
  MsgReportDemand: () => MsgReportDemand,
  MsgReportDemandResponse: () => MsgReportDemandResponse,
  MsgResolveContributionChallenge: () => MsgResolveContributionChallenge,
  MsgResolveContributionChallengeResponse: () => MsgResolveContributionChallengeResponse,
  MsgResolveIncident: () => MsgResolveIncident,
  MsgResolveIncidentResponse: () => MsgResolveIncidentResponse,
  MsgRetireModelCard: () => MsgRetireModelCard,
  MsgRetireModelCardResponse: () => MsgRetireModelCardResponse,
  MsgSponsorVetoAugmentation: () => MsgSponsorVetoAugmentation,
  MsgSponsorVetoAugmentationResponse: () => MsgSponsorVetoAugmentationResponse,
  MsgSubmitAugmentation: () => MsgSubmitAugmentation,
  MsgSubmitAugmentationResponse: () => MsgSubmitAugmentationResponse,
  MsgSubmitClaim: () => MsgSubmitClaim,
  MsgSubmitClaimResponse: () => MsgSubmitClaimResponse,
  MsgSubmitCommitment: () => MsgSubmitCommitment,
  MsgSubmitCommitmentResponse: () => MsgSubmitCommitmentResponse,
  MsgSubmitContradiction: () => MsgSubmitContradiction,
  MsgSubmitContradictionResponse: () => MsgSubmitContradictionResponse,
  MsgSubmitReveal: () => MsgSubmitReveal,
  MsgSubmitRevealResponse: () => MsgSubmitRevealResponse,
  MsgUnpauseModule: () => MsgUnpauseModule,
  MsgUnpauseModuleResponse: () => MsgUnpauseModuleResponse,
  MsgUpdateExtendedParams: () => MsgUpdateExtendedParams,
  MsgUpdateExtendedParamsResponse: () => MsgUpdateExtendedParamsResponse,
  MsgUpdateModelCard: () => MsgUpdateModelCard,
  MsgUpdateModelCardResponse: () => MsgUpdateModelCardResponse,
  MsgUpdateParams: () => MsgUpdateParams11,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse11,
  MsgUpdateTrainingPipeline: () => MsgUpdateTrainingPipeline,
  MsgUpdateTrainingPipelineResponse: () => MsgUpdateTrainingPipelineResponse,
  MsgVetoFactInjection: () => MsgVetoFactInjection,
  MsgVetoFactInjectionResponse: () => MsgVetoFactInjectionResponse,
  MsgVoteOnAugmentation: () => MsgVoteOnAugmentation,
  MsgVoteOnAugmentationResponse: () => MsgVoteOnAugmentationResponse,
  MsgVoteResearchProposal: () => MsgVoteResearchProposal,
  MsgVoteResearchProposalResponse: () => MsgVoteResearchProposalResponse
});

// src/generated/zerone/knowledge/v1/types.ts
function createBaseClaimRelation() {
  return {
    targetFactId: "",
    relation: 0,
    inference: 0,
    inferenceStrengthBps: BigInt(0),
    methodId: ""
  };
}
var ClaimRelation = {
  typeUrl: "/zerone.knowledge.v1.ClaimRelation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.targetFactId !== "") {
      writer.uint32(10).string(message.targetFactId);
    }
    if (message.relation !== 0) {
      writer.uint32(16).int32(message.relation);
    }
    if (message.inference !== 0) {
      writer.uint32(24).int32(message.inference);
    }
    if (message.inferenceStrengthBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.inferenceStrengthBps);
    }
    if (message.methodId !== "") {
      writer.uint32(42).string(message.methodId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseClaimRelation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.targetFactId = reader.string();
          break;
        case 2:
          message.relation = reader.int32();
          break;
        case 3:
          message.inference = reader.int32();
          break;
        case 4:
          message.inferenceStrengthBps = reader.uint64();
          break;
        case 5:
          message.methodId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseClaimRelation();
    message.targetFactId = object.targetFactId ?? "";
    message.relation = object.relation ?? 0;
    message.inference = object.inference ?? 0;
    message.inferenceStrengthBps = object.inferenceStrengthBps !== void 0 && object.inferenceStrengthBps !== null ? BigInt(object.inferenceStrengthBps.toString()) : BigInt(0);
    message.methodId = object.methodId ?? "";
    return message;
  }
};
function createBaseClaimStructure() {
  return {
    subject: "",
    predicate: "",
    object: "",
    scope: "",
    temporalScope: "",
    negatable: false,
    tags: []
  };
}
var ClaimStructure = {
  typeUrl: "/zerone.knowledge.v1.ClaimStructure",
  encode(message, writer = BinaryWriter.create()) {
    if (message.subject !== "") {
      writer.uint32(10).string(message.subject);
    }
    if (message.predicate !== "") {
      writer.uint32(18).string(message.predicate);
    }
    if (message.object !== "") {
      writer.uint32(26).string(message.object);
    }
    if (message.scope !== "") {
      writer.uint32(34).string(message.scope);
    }
    if (message.temporalScope !== "") {
      writer.uint32(42).string(message.temporalScope);
    }
    if (message.negatable === true) {
      writer.uint32(48).bool(message.negatable);
    }
    for (const v of message.tags) {
      writer.uint32(58).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseClaimStructure();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.subject = reader.string();
          break;
        case 2:
          message.predicate = reader.string();
          break;
        case 3:
          message.object = reader.string();
          break;
        case 4:
          message.scope = reader.string();
          break;
        case 5:
          message.temporalScope = reader.string();
          break;
        case 6:
          message.negatable = reader.bool();
          break;
        case 7:
          message.tags.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseClaimStructure();
    message.subject = object.subject ?? "";
    message.predicate = object.predicate ?? "";
    message.object = object.object ?? "";
    message.scope = object.scope ?? "";
    message.temporalScope = object.temporalScope ?? "";
    message.negatable = object.negatable ?? false;
    message.tags = object.tags?.map((e) => e) || [];
    return message;
  }
};
function createBaseTokenizerSpec() {
  return {
    version: BigInt(0),
    ratifiedAtBlock: BigInt(0),
    methodTokenPrefix: "",
    inferenceTokenPrefix: "",
    relationTokenPrefix: "",
    factStatusTokenPrefix: "",
    tierTokenPrefix: "",
    factBeginToken: "",
    factEndToken: "",
    reasoningBeginToken: "",
    reasoningEndToken: "",
    supportBeginToken: "",
    supportEndToken: "",
    disproofMarkerToken: "",
    canonicalSerialisationVersion: BigInt(0)
  };
}
var TokenizerSpec = {
  typeUrl: "/zerone.knowledge.v1.TokenizerSpec",
  encode(message, writer = BinaryWriter.create()) {
    if (message.version !== BigInt(0)) {
      writer.uint32(8).uint64(message.version);
    }
    if (message.ratifiedAtBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.ratifiedAtBlock);
    }
    if (message.methodTokenPrefix !== "") {
      writer.uint32(26).string(message.methodTokenPrefix);
    }
    if (message.inferenceTokenPrefix !== "") {
      writer.uint32(34).string(message.inferenceTokenPrefix);
    }
    if (message.relationTokenPrefix !== "") {
      writer.uint32(42).string(message.relationTokenPrefix);
    }
    if (message.factStatusTokenPrefix !== "") {
      writer.uint32(50).string(message.factStatusTokenPrefix);
    }
    if (message.tierTokenPrefix !== "") {
      writer.uint32(58).string(message.tierTokenPrefix);
    }
    if (message.factBeginToken !== "") {
      writer.uint32(66).string(message.factBeginToken);
    }
    if (message.factEndToken !== "") {
      writer.uint32(74).string(message.factEndToken);
    }
    if (message.reasoningBeginToken !== "") {
      writer.uint32(82).string(message.reasoningBeginToken);
    }
    if (message.reasoningEndToken !== "") {
      writer.uint32(90).string(message.reasoningEndToken);
    }
    if (message.supportBeginToken !== "") {
      writer.uint32(98).string(message.supportBeginToken);
    }
    if (message.supportEndToken !== "") {
      writer.uint32(106).string(message.supportEndToken);
    }
    if (message.disproofMarkerToken !== "") {
      writer.uint32(114).string(message.disproofMarkerToken);
    }
    if (message.canonicalSerialisationVersion !== BigInt(0)) {
      writer.uint32(120).uint64(message.canonicalSerialisationVersion);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseTokenizerSpec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.version = reader.uint64();
          break;
        case 2:
          message.ratifiedAtBlock = reader.uint64();
          break;
        case 3:
          message.methodTokenPrefix = reader.string();
          break;
        case 4:
          message.inferenceTokenPrefix = reader.string();
          break;
        case 5:
          message.relationTokenPrefix = reader.string();
          break;
        case 6:
          message.factStatusTokenPrefix = reader.string();
          break;
        case 7:
          message.tierTokenPrefix = reader.string();
          break;
        case 8:
          message.factBeginToken = reader.string();
          break;
        case 9:
          message.factEndToken = reader.string();
          break;
        case 10:
          message.reasoningBeginToken = reader.string();
          break;
        case 11:
          message.reasoningEndToken = reader.string();
          break;
        case 12:
          message.supportBeginToken = reader.string();
          break;
        case 13:
          message.supportEndToken = reader.string();
          break;
        case 14:
          message.disproofMarkerToken = reader.string();
          break;
        case 15:
          message.canonicalSerialisationVersion = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseTokenizerSpec();
    message.version = object.version !== void 0 && object.version !== null ? BigInt(object.version.toString()) : BigInt(0);
    message.ratifiedAtBlock = object.ratifiedAtBlock !== void 0 && object.ratifiedAtBlock !== null ? BigInt(object.ratifiedAtBlock.toString()) : BigInt(0);
    message.methodTokenPrefix = object.methodTokenPrefix ?? "";
    message.inferenceTokenPrefix = object.inferenceTokenPrefix ?? "";
    message.relationTokenPrefix = object.relationTokenPrefix ?? "";
    message.factStatusTokenPrefix = object.factStatusTokenPrefix ?? "";
    message.tierTokenPrefix = object.tierTokenPrefix ?? "";
    message.factBeginToken = object.factBeginToken ?? "";
    message.factEndToken = object.factEndToken ?? "";
    message.reasoningBeginToken = object.reasoningBeginToken ?? "";
    message.reasoningEndToken = object.reasoningEndToken ?? "";
    message.supportBeginToken = object.supportBeginToken ?? "";
    message.supportEndToken = object.supportEndToken ?? "";
    message.disproofMarkerToken = object.disproofMarkerToken ?? "";
    message.canonicalSerialisationVersion = object.canonicalSerialisationVersion !== void 0 && object.canonicalSerialisationVersion !== null ? BigInt(object.canonicalSerialisationVersion.toString()) : BigInt(0);
    return message;
  }
};
function createBaseTraceSchema() {
  return {
    version: BigInt(0),
    ratifiedAtBlock: BigInt(0),
    jsonSchemaHash: "",
    jsonSchema: "",
    requiredFields: [],
    deprecatedFields: [],
    notes: ""
  };
}
var TraceSchema = {
  typeUrl: "/zerone.knowledge.v1.TraceSchema",
  encode(message, writer = BinaryWriter.create()) {
    if (message.version !== BigInt(0)) {
      writer.uint32(8).uint64(message.version);
    }
    if (message.ratifiedAtBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.ratifiedAtBlock);
    }
    if (message.jsonSchemaHash !== "") {
      writer.uint32(26).string(message.jsonSchemaHash);
    }
    if (message.jsonSchema !== "") {
      writer.uint32(34).string(message.jsonSchema);
    }
    for (const v of message.requiredFields) {
      writer.uint32(42).string(v);
    }
    for (const v of message.deprecatedFields) {
      writer.uint32(50).string(v);
    }
    if (message.notes !== "") {
      writer.uint32(58).string(message.notes);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseTraceSchema();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.version = reader.uint64();
          break;
        case 2:
          message.ratifiedAtBlock = reader.uint64();
          break;
        case 3:
          message.jsonSchemaHash = reader.string();
          break;
        case 4:
          message.jsonSchema = reader.string();
          break;
        case 5:
          message.requiredFields.push(reader.string());
          break;
        case 6:
          message.deprecatedFields.push(reader.string());
          break;
        case 7:
          message.notes = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseTraceSchema();
    message.version = object.version !== void 0 && object.version !== null ? BigInt(object.version.toString()) : BigInt(0);
    message.ratifiedAtBlock = object.ratifiedAtBlock !== void 0 && object.ratifiedAtBlock !== null ? BigInt(object.ratifiedAtBlock.toString()) : BigInt(0);
    message.jsonSchemaHash = object.jsonSchemaHash ?? "";
    message.jsonSchema = object.jsonSchema ?? "";
    message.requiredFields = object.requiredFields?.map((e) => e) || [];
    message.deprecatedFields = object.deprecatedFields?.map((e) => e) || [];
    message.notes = object.notes ?? "";
    return message;
  }
};
function createBaseCorpusSelector() {
  return {
    methodId: "",
    minCorroboration: BigInt(0),
    minQualityTier: 0,
    minCurriculumTier: 0,
    includeDisproven: false,
    includeDrift: false,
    includeNormative: false,
    includeContrastivePairs: false,
    pairTypeFilter: 0,
    domainWhitelist: [],
    domainBlacklist: [],
    minSubmitterCalibrationBps: BigInt(0)
  };
}
var CorpusSelector = {
  typeUrl: "/zerone.knowledge.v1.CorpusSelector",
  encode(message, writer = BinaryWriter.create()) {
    if (message.methodId !== "") {
      writer.uint32(10).string(message.methodId);
    }
    if (message.minCorroboration !== BigInt(0)) {
      writer.uint32(16).uint64(message.minCorroboration);
    }
    if (message.minQualityTier !== 0) {
      writer.uint32(24).int32(message.minQualityTier);
    }
    if (message.minCurriculumTier !== 0) {
      writer.uint32(32).int32(message.minCurriculumTier);
    }
    if (message.includeDisproven === true) {
      writer.uint32(40).bool(message.includeDisproven);
    }
    if (message.includeDrift === true) {
      writer.uint32(48).bool(message.includeDrift);
    }
    if (message.includeNormative === true) {
      writer.uint32(56).bool(message.includeNormative);
    }
    if (message.includeContrastivePairs === true) {
      writer.uint32(64).bool(message.includeContrastivePairs);
    }
    if (message.pairTypeFilter !== 0) {
      writer.uint32(72).int32(message.pairTypeFilter);
    }
    for (const v of message.domainWhitelist) {
      writer.uint32(82).string(v);
    }
    for (const v of message.domainBlacklist) {
      writer.uint32(90).string(v);
    }
    if (message.minSubmitterCalibrationBps !== BigInt(0)) {
      writer.uint32(96).uint64(message.minSubmitterCalibrationBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseCorpusSelector();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.methodId = reader.string();
          break;
        case 2:
          message.minCorroboration = reader.uint64();
          break;
        case 3:
          message.minQualityTier = reader.int32();
          break;
        case 4:
          message.minCurriculumTier = reader.int32();
          break;
        case 5:
          message.includeDisproven = reader.bool();
          break;
        case 6:
          message.includeDrift = reader.bool();
          break;
        case 7:
          message.includeNormative = reader.bool();
          break;
        case 8:
          message.includeContrastivePairs = reader.bool();
          break;
        case 9:
          message.pairTypeFilter = reader.int32();
          break;
        case 10:
          message.domainWhitelist.push(reader.string());
          break;
        case 11:
          message.domainBlacklist.push(reader.string());
          break;
        case 12:
          message.minSubmitterCalibrationBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseCorpusSelector();
    message.methodId = object.methodId ?? "";
    message.minCorroboration = object.minCorroboration !== void 0 && object.minCorroboration !== null ? BigInt(object.minCorroboration.toString()) : BigInt(0);
    message.minQualityTier = object.minQualityTier ?? 0;
    message.minCurriculumTier = object.minCurriculumTier ?? 0;
    message.includeDisproven = object.includeDisproven ?? false;
    message.includeDrift = object.includeDrift ?? false;
    message.includeNormative = object.includeNormative ?? false;
    message.includeContrastivePairs = object.includeContrastivePairs ?? false;
    message.pairTypeFilter = object.pairTypeFilter ?? 0;
    message.domainWhitelist = object.domainWhitelist?.map((e) => e) || [];
    message.domainBlacklist = object.domainBlacklist?.map((e) => e) || [];
    message.minSubmitterCalibrationBps = object.minSubmitterCalibrationBps !== void 0 && object.minSubmitterCalibrationBps !== null ? BigInt(object.minSubmitterCalibrationBps.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/knowledge/v1/genesis.ts
function createBaseParams_MethodologyNormalizationBpsEntry() {
  return {
    key: "",
    value: BigInt(0)
  };
}
var Params_MethodologyNormalizationBpsEntry = {
  encode(message, writer = BinaryWriter.create()) {
    if (message.key !== "") {
      writer.uint32(10).string(message.key);
    }
    if (message.value !== BigInt(0)) {
      writer.uint32(16).uint64(message.value);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams_MethodologyNormalizationBpsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = reader.string();
          break;
        case 2:
          message.value = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams_MethodologyNormalizationBpsEntry();
    message.key = object.key ?? "";
    message.value = object.value !== void 0 && object.value !== null ? BigInt(object.value.toString()) : BigInt(0);
    return message;
  }
};
function createBaseParams12() {
  return {
    minVerifiers: BigInt(0),
    maxVerifiers: BigInt(0),
    commitPhaseBlocks: BigInt(0),
    revealPhaseBlocks: BigInt(0),
    aggregationPhaseBlocks: BigInt(0),
    claimCooldownBlocks: BigInt(0),
    initialConfidence: BigInt(0),
    confidenceBoostPerVerification: BigInt(0),
    confidenceThreshold: BigInt(0),
    quorumThreshold: BigInt(0),
    wrongVerificationSlashBps: BigInt(0),
    missedRevealSlashBps: BigInt(0),
    equivocationSlashBps: BigInt(0),
    invalidClaimSlashBps: BigInt(0),
    verificationReward: "",
    verificationRewardDecayBps: BigInt(0),
    minClaimTextLength: BigInt(0),
    maxClaimTextLength: BigInt(0),
    minReviewFee: "",
    adversarialVerificationEnabled: false,
    provisionalThreshold: BigInt(0),
    rejectThreshold: BigInt(0),
    challengeDurationBlocks: BigInt(0),
    minChallengeStake: "",
    failedChallengeSlashBps: BigInt(0),
    successfulChallengeRewardBps: BigInt(0),
    maxConcurrentChallenges: BigInt(0),
    citationShareBps: BigInt(0),
    crossDomainBonusBps: BigInt(0),
    maxFactsPerDomain: BigInt(0),
    factExpiryBlocks: BigInt(0),
    crossStratumDiscountBps: BigInt(0),
    maxValidatorsPerRound: BigInt(0),
    confidenceGrowthEpoch: BigInt(0),
    confidenceGrowthPerEpochBps: BigInt(0),
    maxSurvivalConfidence: BigInt(0),
    survivedChallengeConfidenceCap: BigInt(0),
    maxApprenticeValidators: BigInt(0),
    researchFundShareBps: BigInt(0),
    fitnessEpochBlocks: BigInt(0),
    fitnessWeightQueryBps: BigInt(0),
    fitnessWeightCitationBps: BigInt(0),
    fitnessWeightBridgeBps: BigInt(0),
    fitnessWeightDepthBps: BigInt(0),
    fitnessWeightPatronBps: BigInt(0),
    fitnessWeightUniqueBps: BigInt(0),
    fitnessWeightAgeBps: BigInt(0),
    fitnessInitialScore: BigInt(0),
    fitnessGraceEpochs: BigInt(0),
    bootstrapFundEnabled: false,
    bootstrapFundMaxPerAddress: "",
    bootstrapFundMaxPerEpoch: "",
    bootstrapFundEpochBlocks: BigInt(0),
    bootstrapFundFeeCap: "",
    metabolismBaseCost: BigInt(0),
    metabolismContentLengthBps: BigInt(0),
    metabolismDomainCompetitionBps: BigInt(0),
    metabolismEnergyPerQuery: BigInt(0),
    metabolismEnergyPerCitation: BigInt(0),
    metabolismEnergyPerPatronage: BigInt(0),
    metabolismEnergyChallengeSurvival: BigInt(0),
    metabolismEnergyCap: BigInt(0),
    metabolismInitialEnergy: BigInt(0),
    metabolismAtRiskEpochs: BigInt(0),
    metabolismExpiredToPrunedEpochs: BigInt(0),
    reproductionRoyaltyBps: BigInt(0),
    reproductionRoyaltyDecayBps: BigInt(0),
    reproductionMaxRoyaltyDepth: BigInt(0),
    reproductionParentEnergyBonus: BigInt(0),
    reproductionChildFitnessInheritanceBps: BigInt(0),
    reproductionMaxChildren: BigInt(0),
    noveltyCommonKnowledgePenaltyBps: BigInt(0),
    noveltySubjectOverlapPenaltyBps: BigInt(0),
    noveltyPrecisionBonusBps: BigInt(0),
    noveltyCrossDomainBonusBps: BigInt(0),
    noveltyMaxOverlapFacts: BigInt(0),
    demandBountyThreshold: BigInt(0),
    demandBountyBaseReward: "",
    demandBountyPerQueryBonus: "",
    demandBountyExpiryEpochs: BigInt(0),
    demandMultiplierCap: BigInt(0),
    demandTrackingEnabled: false,
    authorizedDemandReporters: [],
    competitionNicheDominanceBonusBps: BigInt(0),
    competitionRedundancyThresholdBps: BigInt(0),
    competitionMaxNicheSize: BigInt(0),
    competitionSymbiosisBonusBps: BigInt(0),
    fitnessWeightSatisfactionBps: BigInt(0),
    satisfactionMinRatings: BigInt(0),
    diversityConformityAlertThreshold: BigInt(0),
    diversityConformityAlertEpochs: BigInt(0),
    vindicationRefundEnabled: false,
    vindicationBonusBps: BigInt(0),
    vindicationSlashBps: BigInt(0),
    vindicationWindowBlocks: BigInt(0),
    metabolismActiveThreshold: BigInt(0),
    metabolismExtinctionThreshold: BigInt(0),
    maxConfidence: BigInt(0),
    humanEmpiricalBonusBps: BigInt(0),
    agentComputationalBonusBps: BigInt(0),
    agentVerificationBonusBps: BigInt(0),
    humanPatronageBonusBps: BigInt(0),
    dualValidationBonusBps: BigInt(0),
    domainBaseCapacity: BigInt(0),
    domainCapacityGrowthPerCitation: BigInt(0),
    overcrowdingDecayMultiplierBps: BigInt(0),
    underpopulationBirthBonusBps: BigInt(0),
    epistemicTemperatureDecayBps: BigInt(0),
    epistemicConformityCoolingBps: BigInt(0),
    epistemicVindicationHeatingBps: BigInt(0),
    epistemicColdConfidenceCapBps: BigInt(0),
    epistemicHotConfidenceGrowthBps: BigInt(0),
    epistemicTemperatureWindowBlocks: BigInt(0),
    roleElasticityMinCalls: BigInt(0),
    roleElasticityMaxMultiplierBps: BigInt(0),
    roleElasticityMinMultiplierBps: BigInt(0),
    roleElasticityDecayEpochs: BigInt(0),
    mentorshipDividendEnergy: BigInt(0),
    mentorshipCapacityBonus: BigInt(0),
    socialSaturationThreshold: BigInt(0),
    observationWindowBlocks: BigInt(0),
    minHeadcountAgreement: BigInt(0),
    challengeConfidenceScalingBps: BigInt(0),
    independenceRewardStrengthBps: BigInt(0),
    reformulationMinPanelVotes: BigInt(0),
    reformulationConsensusBps: BigInt(0),
    reformulationSuperiorBonusBps: BigInt(0),
    augmentationExpiryFeeBps: BigInt(0),
    methodologyNormalizationBps: {},
    vindicationTvwMultiplierBps: BigInt(0),
    disprovalClawbackBps: BigInt(0),
    disprovalClawbackWindowEpochs: BigInt(0),
    trainingFundCalibrationFloorBps: BigInt(0),
    trainingFundVestingEpochs: BigInt(0),
    trainingFundMethodologyDiversityBonusBps: BigInt(0),
    trainingFundBaseReward: "",
    contributionChallengeBond: "",
    contributionChallengeRewardMultiplierBps: BigInt(0),
    sponsorVetoForfeitBps: BigInt(0),
    maxPauseDurationBlocks: BigInt(0),
    probeInvitationIdleThresholdBlocks: BigInt(0),
    probeInvitationMinConfidenceBps: BigInt(0),
    probeInvitationBatchSize: 0,
    probeInvitationReinviteCooldown: BigInt(0),
    probeBountyMintPerBlock: "",
    probeBountyMaxPoolSize: "",
    invitationBonusAmount: "",
    guardianAddresses: [],
    addFactVetoWindowBlocks: BigInt(0)
  };
}
var Params12 = {
  typeUrl: "/zerone.knowledge.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.minVerifiers !== BigInt(0)) {
      writer.uint32(8).uint64(message.minVerifiers);
    }
    if (message.maxVerifiers !== BigInt(0)) {
      writer.uint32(16).uint64(message.maxVerifiers);
    }
    if (message.commitPhaseBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.commitPhaseBlocks);
    }
    if (message.revealPhaseBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.revealPhaseBlocks);
    }
    if (message.aggregationPhaseBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.aggregationPhaseBlocks);
    }
    if (message.claimCooldownBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.claimCooldownBlocks);
    }
    if (message.initialConfidence !== BigInt(0)) {
      writer.uint32(56).uint64(message.initialConfidence);
    }
    if (message.confidenceBoostPerVerification !== BigInt(0)) {
      writer.uint32(64).uint64(message.confidenceBoostPerVerification);
    }
    if (message.confidenceThreshold !== BigInt(0)) {
      writer.uint32(72).uint64(message.confidenceThreshold);
    }
    if (message.quorumThreshold !== BigInt(0)) {
      writer.uint32(80).uint64(message.quorumThreshold);
    }
    if (message.wrongVerificationSlashBps !== BigInt(0)) {
      writer.uint32(88).uint64(message.wrongVerificationSlashBps);
    }
    if (message.missedRevealSlashBps !== BigInt(0)) {
      writer.uint32(96).uint64(message.missedRevealSlashBps);
    }
    if (message.equivocationSlashBps !== BigInt(0)) {
      writer.uint32(104).uint64(message.equivocationSlashBps);
    }
    if (message.invalidClaimSlashBps !== BigInt(0)) {
      writer.uint32(112).uint64(message.invalidClaimSlashBps);
    }
    if (message.verificationReward !== "") {
      writer.uint32(122).string(message.verificationReward);
    }
    if (message.verificationRewardDecayBps !== BigInt(0)) {
      writer.uint32(128).uint64(message.verificationRewardDecayBps);
    }
    if (message.minClaimTextLength !== BigInt(0)) {
      writer.uint32(136).uint64(message.minClaimTextLength);
    }
    if (message.maxClaimTextLength !== BigInt(0)) {
      writer.uint32(144).uint64(message.maxClaimTextLength);
    }
    if (message.minReviewFee !== "") {
      writer.uint32(154).string(message.minReviewFee);
    }
    if (message.adversarialVerificationEnabled === true) {
      writer.uint32(160).bool(message.adversarialVerificationEnabled);
    }
    if (message.provisionalThreshold !== BigInt(0)) {
      writer.uint32(168).uint64(message.provisionalThreshold);
    }
    if (message.rejectThreshold !== BigInt(0)) {
      writer.uint32(176).uint64(message.rejectThreshold);
    }
    if (message.challengeDurationBlocks !== BigInt(0)) {
      writer.uint32(184).uint64(message.challengeDurationBlocks);
    }
    if (message.minChallengeStake !== "") {
      writer.uint32(194).string(message.minChallengeStake);
    }
    if (message.failedChallengeSlashBps !== BigInt(0)) {
      writer.uint32(200).uint64(message.failedChallengeSlashBps);
    }
    if (message.successfulChallengeRewardBps !== BigInt(0)) {
      writer.uint32(208).uint64(message.successfulChallengeRewardBps);
    }
    if (message.maxConcurrentChallenges !== BigInt(0)) {
      writer.uint32(216).uint64(message.maxConcurrentChallenges);
    }
    if (message.citationShareBps !== BigInt(0)) {
      writer.uint32(224).uint64(message.citationShareBps);
    }
    if (message.crossDomainBonusBps !== BigInt(0)) {
      writer.uint32(232).uint64(message.crossDomainBonusBps);
    }
    if (message.maxFactsPerDomain !== BigInt(0)) {
      writer.uint32(240).uint64(message.maxFactsPerDomain);
    }
    if (message.factExpiryBlocks !== BigInt(0)) {
      writer.uint32(248).uint64(message.factExpiryBlocks);
    }
    if (message.crossStratumDiscountBps !== BigInt(0)) {
      writer.uint32(256).uint64(message.crossStratumDiscountBps);
    }
    if (message.maxValidatorsPerRound !== BigInt(0)) {
      writer.uint32(272).uint64(message.maxValidatorsPerRound);
    }
    if (message.confidenceGrowthEpoch !== BigInt(0)) {
      writer.uint32(304).uint64(message.confidenceGrowthEpoch);
    }
    if (message.confidenceGrowthPerEpochBps !== BigInt(0)) {
      writer.uint32(312).uint64(message.confidenceGrowthPerEpochBps);
    }
    if (message.maxSurvivalConfidence !== BigInt(0)) {
      writer.uint32(320).uint64(message.maxSurvivalConfidence);
    }
    if (message.survivedChallengeConfidenceCap !== BigInt(0)) {
      writer.uint32(328).uint64(message.survivedChallengeConfidenceCap);
    }
    if (message.maxApprenticeValidators !== BigInt(0)) {
      writer.uint32(336).uint64(message.maxApprenticeValidators);
    }
    if (message.researchFundShareBps !== BigInt(0)) {
      writer.uint32(392).uint64(message.researchFundShareBps);
    }
    if (message.fitnessEpochBlocks !== BigInt(0)) {
      writer.uint32(408).uint64(message.fitnessEpochBlocks);
    }
    if (message.fitnessWeightQueryBps !== BigInt(0)) {
      writer.uint32(416).uint64(message.fitnessWeightQueryBps);
    }
    if (message.fitnessWeightCitationBps !== BigInt(0)) {
      writer.uint32(424).uint64(message.fitnessWeightCitationBps);
    }
    if (message.fitnessWeightBridgeBps !== BigInt(0)) {
      writer.uint32(432).uint64(message.fitnessWeightBridgeBps);
    }
    if (message.fitnessWeightDepthBps !== BigInt(0)) {
      writer.uint32(440).uint64(message.fitnessWeightDepthBps);
    }
    if (message.fitnessWeightPatronBps !== BigInt(0)) {
      writer.uint32(448).uint64(message.fitnessWeightPatronBps);
    }
    if (message.fitnessWeightUniqueBps !== BigInt(0)) {
      writer.uint32(456).uint64(message.fitnessWeightUniqueBps);
    }
    if (message.fitnessWeightAgeBps !== BigInt(0)) {
      writer.uint32(464).uint64(message.fitnessWeightAgeBps);
    }
    if (message.fitnessInitialScore !== BigInt(0)) {
      writer.uint32(472).uint64(message.fitnessInitialScore);
    }
    if (message.fitnessGraceEpochs !== BigInt(0)) {
      writer.uint32(480).uint64(message.fitnessGraceEpochs);
    }
    if (message.bootstrapFundEnabled === true) {
      writer.uint32(488).bool(message.bootstrapFundEnabled);
    }
    if (message.bootstrapFundMaxPerAddress !== "") {
      writer.uint32(498).string(message.bootstrapFundMaxPerAddress);
    }
    if (message.bootstrapFundMaxPerEpoch !== "") {
      writer.uint32(506).string(message.bootstrapFundMaxPerEpoch);
    }
    if (message.bootstrapFundEpochBlocks !== BigInt(0)) {
      writer.uint32(512).uint64(message.bootstrapFundEpochBlocks);
    }
    if (message.bootstrapFundFeeCap !== "") {
      writer.uint32(522).string(message.bootstrapFundFeeCap);
    }
    if (message.metabolismBaseCost !== BigInt(0)) {
      writer.uint32(528).uint64(message.metabolismBaseCost);
    }
    if (message.metabolismContentLengthBps !== BigInt(0)) {
      writer.uint32(536).uint64(message.metabolismContentLengthBps);
    }
    if (message.metabolismDomainCompetitionBps !== BigInt(0)) {
      writer.uint32(544).uint64(message.metabolismDomainCompetitionBps);
    }
    if (message.metabolismEnergyPerQuery !== BigInt(0)) {
      writer.uint32(552).uint64(message.metabolismEnergyPerQuery);
    }
    if (message.metabolismEnergyPerCitation !== BigInt(0)) {
      writer.uint32(560).uint64(message.metabolismEnergyPerCitation);
    }
    if (message.metabolismEnergyPerPatronage !== BigInt(0)) {
      writer.uint32(568).uint64(message.metabolismEnergyPerPatronage);
    }
    if (message.metabolismEnergyChallengeSurvival !== BigInt(0)) {
      writer.uint32(576).uint64(message.metabolismEnergyChallengeSurvival);
    }
    if (message.metabolismEnergyCap !== BigInt(0)) {
      writer.uint32(584).uint64(message.metabolismEnergyCap);
    }
    if (message.metabolismInitialEnergy !== BigInt(0)) {
      writer.uint32(592).uint64(message.metabolismInitialEnergy);
    }
    if (message.metabolismAtRiskEpochs !== BigInt(0)) {
      writer.uint32(600).uint64(message.metabolismAtRiskEpochs);
    }
    if (message.metabolismExpiredToPrunedEpochs !== BigInt(0)) {
      writer.uint32(608).uint64(message.metabolismExpiredToPrunedEpochs);
    }
    if (message.reproductionRoyaltyBps !== BigInt(0)) {
      writer.uint32(616).uint64(message.reproductionRoyaltyBps);
    }
    if (message.reproductionRoyaltyDecayBps !== BigInt(0)) {
      writer.uint32(624).uint64(message.reproductionRoyaltyDecayBps);
    }
    if (message.reproductionMaxRoyaltyDepth !== BigInt(0)) {
      writer.uint32(632).uint64(message.reproductionMaxRoyaltyDepth);
    }
    if (message.reproductionParentEnergyBonus !== BigInt(0)) {
      writer.uint32(640).uint64(message.reproductionParentEnergyBonus);
    }
    if (message.reproductionChildFitnessInheritanceBps !== BigInt(0)) {
      writer.uint32(648).uint64(message.reproductionChildFitnessInheritanceBps);
    }
    if (message.reproductionMaxChildren !== BigInt(0)) {
      writer.uint32(656).uint64(message.reproductionMaxChildren);
    }
    if (message.noveltyCommonKnowledgePenaltyBps !== BigInt(0)) {
      writer.uint32(664).uint64(message.noveltyCommonKnowledgePenaltyBps);
    }
    if (message.noveltySubjectOverlapPenaltyBps !== BigInt(0)) {
      writer.uint32(672).uint64(message.noveltySubjectOverlapPenaltyBps);
    }
    if (message.noveltyPrecisionBonusBps !== BigInt(0)) {
      writer.uint32(680).uint64(message.noveltyPrecisionBonusBps);
    }
    if (message.noveltyCrossDomainBonusBps !== BigInt(0)) {
      writer.uint32(688).uint64(message.noveltyCrossDomainBonusBps);
    }
    if (message.noveltyMaxOverlapFacts !== BigInt(0)) {
      writer.uint32(696).uint64(message.noveltyMaxOverlapFacts);
    }
    if (message.demandBountyThreshold !== BigInt(0)) {
      writer.uint32(704).uint64(message.demandBountyThreshold);
    }
    if (message.demandBountyBaseReward !== "") {
      writer.uint32(714).string(message.demandBountyBaseReward);
    }
    if (message.demandBountyPerQueryBonus !== "") {
      writer.uint32(722).string(message.demandBountyPerQueryBonus);
    }
    if (message.demandBountyExpiryEpochs !== BigInt(0)) {
      writer.uint32(728).uint64(message.demandBountyExpiryEpochs);
    }
    if (message.demandMultiplierCap !== BigInt(0)) {
      writer.uint32(736).uint64(message.demandMultiplierCap);
    }
    if (message.demandTrackingEnabled === true) {
      writer.uint32(744).bool(message.demandTrackingEnabled);
    }
    for (const v of message.authorizedDemandReporters) {
      writer.uint32(754).string(v);
    }
    if (message.competitionNicheDominanceBonusBps !== BigInt(0)) {
      writer.uint32(760).uint64(message.competitionNicheDominanceBonusBps);
    }
    if (message.competitionRedundancyThresholdBps !== BigInt(0)) {
      writer.uint32(768).uint64(message.competitionRedundancyThresholdBps);
    }
    if (message.competitionMaxNicheSize !== BigInt(0)) {
      writer.uint32(776).uint64(message.competitionMaxNicheSize);
    }
    if (message.competitionSymbiosisBonusBps !== BigInt(0)) {
      writer.uint32(784).uint64(message.competitionSymbiosisBonusBps);
    }
    if (message.fitnessWeightSatisfactionBps !== BigInt(0)) {
      writer.uint32(792).uint64(message.fitnessWeightSatisfactionBps);
    }
    if (message.satisfactionMinRatings !== BigInt(0)) {
      writer.uint32(800).uint64(message.satisfactionMinRatings);
    }
    if (message.diversityConformityAlertThreshold !== BigInt(0)) {
      writer.uint32(808).uint64(message.diversityConformityAlertThreshold);
    }
    if (message.diversityConformityAlertEpochs !== BigInt(0)) {
      writer.uint32(816).uint64(message.diversityConformityAlertEpochs);
    }
    if (message.vindicationRefundEnabled === true) {
      writer.uint32(824).bool(message.vindicationRefundEnabled);
    }
    if (message.vindicationBonusBps !== BigInt(0)) {
      writer.uint32(832).uint64(message.vindicationBonusBps);
    }
    if (message.vindicationSlashBps !== BigInt(0)) {
      writer.uint32(840).uint64(message.vindicationSlashBps);
    }
    if (message.vindicationWindowBlocks !== BigInt(0)) {
      writer.uint32(848).uint64(message.vindicationWindowBlocks);
    }
    if (message.metabolismActiveThreshold !== BigInt(0)) {
      writer.uint32(856).uint64(message.metabolismActiveThreshold);
    }
    if (message.metabolismExtinctionThreshold !== BigInt(0)) {
      writer.uint32(864).uint64(message.metabolismExtinctionThreshold);
    }
    if (message.maxConfidence !== BigInt(0)) {
      writer.uint32(872).uint64(message.maxConfidence);
    }
    if (message.humanEmpiricalBonusBps !== BigInt(0)) {
      writer.uint32(880).uint64(message.humanEmpiricalBonusBps);
    }
    if (message.agentComputationalBonusBps !== BigInt(0)) {
      writer.uint32(888).uint64(message.agentComputationalBonusBps);
    }
    if (message.agentVerificationBonusBps !== BigInt(0)) {
      writer.uint32(896).uint64(message.agentVerificationBonusBps);
    }
    if (message.humanPatronageBonusBps !== BigInt(0)) {
      writer.uint32(904).uint64(message.humanPatronageBonusBps);
    }
    if (message.dualValidationBonusBps !== BigInt(0)) {
      writer.uint32(912).uint64(message.dualValidationBonusBps);
    }
    if (message.domainBaseCapacity !== BigInt(0)) {
      writer.uint32(920).uint64(message.domainBaseCapacity);
    }
    if (message.domainCapacityGrowthPerCitation !== BigInt(0)) {
      writer.uint32(928).uint64(message.domainCapacityGrowthPerCitation);
    }
    if (message.overcrowdingDecayMultiplierBps !== BigInt(0)) {
      writer.uint32(936).uint64(message.overcrowdingDecayMultiplierBps);
    }
    if (message.underpopulationBirthBonusBps !== BigInt(0)) {
      writer.uint32(944).uint64(message.underpopulationBirthBonusBps);
    }
    if (message.epistemicTemperatureDecayBps !== BigInt(0)) {
      writer.uint32(952).uint64(message.epistemicTemperatureDecayBps);
    }
    if (message.epistemicConformityCoolingBps !== BigInt(0)) {
      writer.uint32(960).uint64(message.epistemicConformityCoolingBps);
    }
    if (message.epistemicVindicationHeatingBps !== BigInt(0)) {
      writer.uint32(968).uint64(message.epistemicVindicationHeatingBps);
    }
    if (message.epistemicColdConfidenceCapBps !== BigInt(0)) {
      writer.uint32(976).uint64(message.epistemicColdConfidenceCapBps);
    }
    if (message.epistemicHotConfidenceGrowthBps !== BigInt(0)) {
      writer.uint32(984).uint64(message.epistemicHotConfidenceGrowthBps);
    }
    if (message.epistemicTemperatureWindowBlocks !== BigInt(0)) {
      writer.uint32(992).uint64(message.epistemicTemperatureWindowBlocks);
    }
    if (message.roleElasticityMinCalls !== BigInt(0)) {
      writer.uint32(1e3).uint64(message.roleElasticityMinCalls);
    }
    if (message.roleElasticityMaxMultiplierBps !== BigInt(0)) {
      writer.uint32(1008).uint64(message.roleElasticityMaxMultiplierBps);
    }
    if (message.roleElasticityMinMultiplierBps !== BigInt(0)) {
      writer.uint32(1016).uint64(message.roleElasticityMinMultiplierBps);
    }
    if (message.roleElasticityDecayEpochs !== BigInt(0)) {
      writer.uint32(1024).uint64(message.roleElasticityDecayEpochs);
    }
    if (message.mentorshipDividendEnergy !== BigInt(0)) {
      writer.uint32(1032).uint64(message.mentorshipDividendEnergy);
    }
    if (message.mentorshipCapacityBonus !== BigInt(0)) {
      writer.uint32(1040).uint64(message.mentorshipCapacityBonus);
    }
    if (message.socialSaturationThreshold !== BigInt(0)) {
      writer.uint32(1048).uint64(message.socialSaturationThreshold);
    }
    if (message.observationWindowBlocks !== BigInt(0)) {
      writer.uint32(1056).uint64(message.observationWindowBlocks);
    }
    if (message.minHeadcountAgreement !== BigInt(0)) {
      writer.uint32(1064).uint64(message.minHeadcountAgreement);
    }
    if (message.challengeConfidenceScalingBps !== BigInt(0)) {
      writer.uint32(1072).uint64(message.challengeConfidenceScalingBps);
    }
    if (message.independenceRewardStrengthBps !== BigInt(0)) {
      writer.uint32(1080).uint64(message.independenceRewardStrengthBps);
    }
    if (message.reformulationMinPanelVotes !== BigInt(0)) {
      writer.uint32(1088).uint64(message.reformulationMinPanelVotes);
    }
    if (message.reformulationConsensusBps !== BigInt(0)) {
      writer.uint32(1096).uint64(message.reformulationConsensusBps);
    }
    if (message.reformulationSuperiorBonusBps !== BigInt(0)) {
      writer.uint32(1104).uint64(message.reformulationSuperiorBonusBps);
    }
    if (message.augmentationExpiryFeeBps !== BigInt(0)) {
      writer.uint32(1112).uint64(message.augmentationExpiryFeeBps);
    }
    Object.entries(message.methodologyNormalizationBps).forEach(([key, value]) => {
      Params_MethodologyNormalizationBpsEntry.encode({
        key,
        value
      }, writer.uint32(1120).fork()).ldelim();
    });
    if (message.vindicationTvwMultiplierBps !== BigInt(0)) {
      writer.uint32(1128).uint64(message.vindicationTvwMultiplierBps);
    }
    if (message.disprovalClawbackBps !== BigInt(0)) {
      writer.uint32(1136).uint64(message.disprovalClawbackBps);
    }
    if (message.disprovalClawbackWindowEpochs !== BigInt(0)) {
      writer.uint32(1144).uint64(message.disprovalClawbackWindowEpochs);
    }
    if (message.trainingFundCalibrationFloorBps !== BigInt(0)) {
      writer.uint32(1152).uint64(message.trainingFundCalibrationFloorBps);
    }
    if (message.trainingFundVestingEpochs !== BigInt(0)) {
      writer.uint32(1160).uint64(message.trainingFundVestingEpochs);
    }
    if (message.trainingFundMethodologyDiversityBonusBps !== BigInt(0)) {
      writer.uint32(1168).uint64(message.trainingFundMethodologyDiversityBonusBps);
    }
    if (message.trainingFundBaseReward !== "") {
      writer.uint32(1178).string(message.trainingFundBaseReward);
    }
    if (message.contributionChallengeBond !== "") {
      writer.uint32(1186).string(message.contributionChallengeBond);
    }
    if (message.contributionChallengeRewardMultiplierBps !== BigInt(0)) {
      writer.uint32(1192).uint64(message.contributionChallengeRewardMultiplierBps);
    }
    if (message.sponsorVetoForfeitBps !== BigInt(0)) {
      writer.uint32(1200).uint64(message.sponsorVetoForfeitBps);
    }
    if (message.maxPauseDurationBlocks !== BigInt(0)) {
      writer.uint32(1208).uint64(message.maxPauseDurationBlocks);
    }
    if (message.probeInvitationIdleThresholdBlocks !== BigInt(0)) {
      writer.uint32(1216).uint64(message.probeInvitationIdleThresholdBlocks);
    }
    if (message.probeInvitationMinConfidenceBps !== BigInt(0)) {
      writer.uint32(1224).uint64(message.probeInvitationMinConfidenceBps);
    }
    if (message.probeInvitationBatchSize !== 0) {
      writer.uint32(1232).uint32(message.probeInvitationBatchSize);
    }
    if (message.probeInvitationReinviteCooldown !== BigInt(0)) {
      writer.uint32(1240).uint64(message.probeInvitationReinviteCooldown);
    }
    if (message.probeBountyMintPerBlock !== "") {
      writer.uint32(1250).string(message.probeBountyMintPerBlock);
    }
    if (message.probeBountyMaxPoolSize !== "") {
      writer.uint32(1258).string(message.probeBountyMaxPoolSize);
    }
    if (message.invitationBonusAmount !== "") {
      writer.uint32(1266).string(message.invitationBonusAmount);
    }
    for (const v of message.guardianAddresses) {
      writer.uint32(1274).string(v);
    }
    if (message.addFactVetoWindowBlocks !== BigInt(0)) {
      writer.uint32(1280).uint64(message.addFactVetoWindowBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams12();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minVerifiers = reader.uint64();
          break;
        case 2:
          message.maxVerifiers = reader.uint64();
          break;
        case 3:
          message.commitPhaseBlocks = reader.uint64();
          break;
        case 4:
          message.revealPhaseBlocks = reader.uint64();
          break;
        case 5:
          message.aggregationPhaseBlocks = reader.uint64();
          break;
        case 6:
          message.claimCooldownBlocks = reader.uint64();
          break;
        case 7:
          message.initialConfidence = reader.uint64();
          break;
        case 8:
          message.confidenceBoostPerVerification = reader.uint64();
          break;
        case 9:
          message.confidenceThreshold = reader.uint64();
          break;
        case 10:
          message.quorumThreshold = reader.uint64();
          break;
        case 11:
          message.wrongVerificationSlashBps = reader.uint64();
          break;
        case 12:
          message.missedRevealSlashBps = reader.uint64();
          break;
        case 13:
          message.equivocationSlashBps = reader.uint64();
          break;
        case 14:
          message.invalidClaimSlashBps = reader.uint64();
          break;
        case 15:
          message.verificationReward = reader.string();
          break;
        case 16:
          message.verificationRewardDecayBps = reader.uint64();
          break;
        case 17:
          message.minClaimTextLength = reader.uint64();
          break;
        case 18:
          message.maxClaimTextLength = reader.uint64();
          break;
        case 19:
          message.minReviewFee = reader.string();
          break;
        case 20:
          message.adversarialVerificationEnabled = reader.bool();
          break;
        case 21:
          message.provisionalThreshold = reader.uint64();
          break;
        case 22:
          message.rejectThreshold = reader.uint64();
          break;
        case 23:
          message.challengeDurationBlocks = reader.uint64();
          break;
        case 24:
          message.minChallengeStake = reader.string();
          break;
        case 25:
          message.failedChallengeSlashBps = reader.uint64();
          break;
        case 26:
          message.successfulChallengeRewardBps = reader.uint64();
          break;
        case 27:
          message.maxConcurrentChallenges = reader.uint64();
          break;
        case 28:
          message.citationShareBps = reader.uint64();
          break;
        case 29:
          message.crossDomainBonusBps = reader.uint64();
          break;
        case 30:
          message.maxFactsPerDomain = reader.uint64();
          break;
        case 31:
          message.factExpiryBlocks = reader.uint64();
          break;
        case 32:
          message.crossStratumDiscountBps = reader.uint64();
          break;
        case 34:
          message.maxValidatorsPerRound = reader.uint64();
          break;
        case 38:
          message.confidenceGrowthEpoch = reader.uint64();
          break;
        case 39:
          message.confidenceGrowthPerEpochBps = reader.uint64();
          break;
        case 40:
          message.maxSurvivalConfidence = reader.uint64();
          break;
        case 41:
          message.survivedChallengeConfidenceCap = reader.uint64();
          break;
        case 42:
          message.maxApprenticeValidators = reader.uint64();
          break;
        case 49:
          message.researchFundShareBps = reader.uint64();
          break;
        case 51:
          message.fitnessEpochBlocks = reader.uint64();
          break;
        case 52:
          message.fitnessWeightQueryBps = reader.uint64();
          break;
        case 53:
          message.fitnessWeightCitationBps = reader.uint64();
          break;
        case 54:
          message.fitnessWeightBridgeBps = reader.uint64();
          break;
        case 55:
          message.fitnessWeightDepthBps = reader.uint64();
          break;
        case 56:
          message.fitnessWeightPatronBps = reader.uint64();
          break;
        case 57:
          message.fitnessWeightUniqueBps = reader.uint64();
          break;
        case 58:
          message.fitnessWeightAgeBps = reader.uint64();
          break;
        case 59:
          message.fitnessInitialScore = reader.uint64();
          break;
        case 60:
          message.fitnessGraceEpochs = reader.uint64();
          break;
        case 61:
          message.bootstrapFundEnabled = reader.bool();
          break;
        case 62:
          message.bootstrapFundMaxPerAddress = reader.string();
          break;
        case 63:
          message.bootstrapFundMaxPerEpoch = reader.string();
          break;
        case 64:
          message.bootstrapFundEpochBlocks = reader.uint64();
          break;
        case 65:
          message.bootstrapFundFeeCap = reader.string();
          break;
        case 66:
          message.metabolismBaseCost = reader.uint64();
          break;
        case 67:
          message.metabolismContentLengthBps = reader.uint64();
          break;
        case 68:
          message.metabolismDomainCompetitionBps = reader.uint64();
          break;
        case 69:
          message.metabolismEnergyPerQuery = reader.uint64();
          break;
        case 70:
          message.metabolismEnergyPerCitation = reader.uint64();
          break;
        case 71:
          message.metabolismEnergyPerPatronage = reader.uint64();
          break;
        case 72:
          message.metabolismEnergyChallengeSurvival = reader.uint64();
          break;
        case 73:
          message.metabolismEnergyCap = reader.uint64();
          break;
        case 74:
          message.metabolismInitialEnergy = reader.uint64();
          break;
        case 75:
          message.metabolismAtRiskEpochs = reader.uint64();
          break;
        case 76:
          message.metabolismExpiredToPrunedEpochs = reader.uint64();
          break;
        case 77:
          message.reproductionRoyaltyBps = reader.uint64();
          break;
        case 78:
          message.reproductionRoyaltyDecayBps = reader.uint64();
          break;
        case 79:
          message.reproductionMaxRoyaltyDepth = reader.uint64();
          break;
        case 80:
          message.reproductionParentEnergyBonus = reader.uint64();
          break;
        case 81:
          message.reproductionChildFitnessInheritanceBps = reader.uint64();
          break;
        case 82:
          message.reproductionMaxChildren = reader.uint64();
          break;
        case 83:
          message.noveltyCommonKnowledgePenaltyBps = reader.uint64();
          break;
        case 84:
          message.noveltySubjectOverlapPenaltyBps = reader.uint64();
          break;
        case 85:
          message.noveltyPrecisionBonusBps = reader.uint64();
          break;
        case 86:
          message.noveltyCrossDomainBonusBps = reader.uint64();
          break;
        case 87:
          message.noveltyMaxOverlapFacts = reader.uint64();
          break;
        case 88:
          message.demandBountyThreshold = reader.uint64();
          break;
        case 89:
          message.demandBountyBaseReward = reader.string();
          break;
        case 90:
          message.demandBountyPerQueryBonus = reader.string();
          break;
        case 91:
          message.demandBountyExpiryEpochs = reader.uint64();
          break;
        case 92:
          message.demandMultiplierCap = reader.uint64();
          break;
        case 93:
          message.demandTrackingEnabled = reader.bool();
          break;
        case 94:
          message.authorizedDemandReporters.push(reader.string());
          break;
        case 95:
          message.competitionNicheDominanceBonusBps = reader.uint64();
          break;
        case 96:
          message.competitionRedundancyThresholdBps = reader.uint64();
          break;
        case 97:
          message.competitionMaxNicheSize = reader.uint64();
          break;
        case 98:
          message.competitionSymbiosisBonusBps = reader.uint64();
          break;
        case 99:
          message.fitnessWeightSatisfactionBps = reader.uint64();
          break;
        case 100:
          message.satisfactionMinRatings = reader.uint64();
          break;
        case 101:
          message.diversityConformityAlertThreshold = reader.uint64();
          break;
        case 102:
          message.diversityConformityAlertEpochs = reader.uint64();
          break;
        case 103:
          message.vindicationRefundEnabled = reader.bool();
          break;
        case 104:
          message.vindicationBonusBps = reader.uint64();
          break;
        case 105:
          message.vindicationSlashBps = reader.uint64();
          break;
        case 106:
          message.vindicationWindowBlocks = reader.uint64();
          break;
        case 107:
          message.metabolismActiveThreshold = reader.uint64();
          break;
        case 108:
          message.metabolismExtinctionThreshold = reader.uint64();
          break;
        case 109:
          message.maxConfidence = reader.uint64();
          break;
        case 110:
          message.humanEmpiricalBonusBps = reader.uint64();
          break;
        case 111:
          message.agentComputationalBonusBps = reader.uint64();
          break;
        case 112:
          message.agentVerificationBonusBps = reader.uint64();
          break;
        case 113:
          message.humanPatronageBonusBps = reader.uint64();
          break;
        case 114:
          message.dualValidationBonusBps = reader.uint64();
          break;
        case 115:
          message.domainBaseCapacity = reader.uint64();
          break;
        case 116:
          message.domainCapacityGrowthPerCitation = reader.uint64();
          break;
        case 117:
          message.overcrowdingDecayMultiplierBps = reader.uint64();
          break;
        case 118:
          message.underpopulationBirthBonusBps = reader.uint64();
          break;
        case 119:
          message.epistemicTemperatureDecayBps = reader.uint64();
          break;
        case 120:
          message.epistemicConformityCoolingBps = reader.uint64();
          break;
        case 121:
          message.epistemicVindicationHeatingBps = reader.uint64();
          break;
        case 122:
          message.epistemicColdConfidenceCapBps = reader.uint64();
          break;
        case 123:
          message.epistemicHotConfidenceGrowthBps = reader.uint64();
          break;
        case 124:
          message.epistemicTemperatureWindowBlocks = reader.uint64();
          break;
        case 125:
          message.roleElasticityMinCalls = reader.uint64();
          break;
        case 126:
          message.roleElasticityMaxMultiplierBps = reader.uint64();
          break;
        case 127:
          message.roleElasticityMinMultiplierBps = reader.uint64();
          break;
        case 128:
          message.roleElasticityDecayEpochs = reader.uint64();
          break;
        case 129:
          message.mentorshipDividendEnergy = reader.uint64();
          break;
        case 130:
          message.mentorshipCapacityBonus = reader.uint64();
          break;
        case 131:
          message.socialSaturationThreshold = reader.uint64();
          break;
        case 132:
          message.observationWindowBlocks = reader.uint64();
          break;
        case 133:
          message.minHeadcountAgreement = reader.uint64();
          break;
        case 134:
          message.challengeConfidenceScalingBps = reader.uint64();
          break;
        case 135:
          message.independenceRewardStrengthBps = reader.uint64();
          break;
        case 136:
          message.reformulationMinPanelVotes = reader.uint64();
          break;
        case 137:
          message.reformulationConsensusBps = reader.uint64();
          break;
        case 138:
          message.reformulationSuperiorBonusBps = reader.uint64();
          break;
        case 139:
          message.augmentationExpiryFeeBps = reader.uint64();
          break;
        case 140:
          const entry140 = Params_MethodologyNormalizationBpsEntry.decode(reader, reader.uint32());
          if (entry140.value !== void 0) {
            message.methodologyNormalizationBps[entry140.key] = entry140.value;
          }
          break;
        case 141:
          message.vindicationTvwMultiplierBps = reader.uint64();
          break;
        case 142:
          message.disprovalClawbackBps = reader.uint64();
          break;
        case 143:
          message.disprovalClawbackWindowEpochs = reader.uint64();
          break;
        case 144:
          message.trainingFundCalibrationFloorBps = reader.uint64();
          break;
        case 145:
          message.trainingFundVestingEpochs = reader.uint64();
          break;
        case 146:
          message.trainingFundMethodologyDiversityBonusBps = reader.uint64();
          break;
        case 147:
          message.trainingFundBaseReward = reader.string();
          break;
        case 148:
          message.contributionChallengeBond = reader.string();
          break;
        case 149:
          message.contributionChallengeRewardMultiplierBps = reader.uint64();
          break;
        case 150:
          message.sponsorVetoForfeitBps = reader.uint64();
          break;
        case 151:
          message.maxPauseDurationBlocks = reader.uint64();
          break;
        case 152:
          message.probeInvitationIdleThresholdBlocks = reader.uint64();
          break;
        case 153:
          message.probeInvitationMinConfidenceBps = reader.uint64();
          break;
        case 154:
          message.probeInvitationBatchSize = reader.uint32();
          break;
        case 155:
          message.probeInvitationReinviteCooldown = reader.uint64();
          break;
        case 156:
          message.probeBountyMintPerBlock = reader.string();
          break;
        case 157:
          message.probeBountyMaxPoolSize = reader.string();
          break;
        case 158:
          message.invitationBonusAmount = reader.string();
          break;
        case 159:
          message.guardianAddresses.push(reader.string());
          break;
        case 160:
          message.addFactVetoWindowBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams12();
    message.minVerifiers = object.minVerifiers !== void 0 && object.minVerifiers !== null ? BigInt(object.minVerifiers.toString()) : BigInt(0);
    message.maxVerifiers = object.maxVerifiers !== void 0 && object.maxVerifiers !== null ? BigInt(object.maxVerifiers.toString()) : BigInt(0);
    message.commitPhaseBlocks = object.commitPhaseBlocks !== void 0 && object.commitPhaseBlocks !== null ? BigInt(object.commitPhaseBlocks.toString()) : BigInt(0);
    message.revealPhaseBlocks = object.revealPhaseBlocks !== void 0 && object.revealPhaseBlocks !== null ? BigInt(object.revealPhaseBlocks.toString()) : BigInt(0);
    message.aggregationPhaseBlocks = object.aggregationPhaseBlocks !== void 0 && object.aggregationPhaseBlocks !== null ? BigInt(object.aggregationPhaseBlocks.toString()) : BigInt(0);
    message.claimCooldownBlocks = object.claimCooldownBlocks !== void 0 && object.claimCooldownBlocks !== null ? BigInt(object.claimCooldownBlocks.toString()) : BigInt(0);
    message.initialConfidence = object.initialConfidence !== void 0 && object.initialConfidence !== null ? BigInt(object.initialConfidence.toString()) : BigInt(0);
    message.confidenceBoostPerVerification = object.confidenceBoostPerVerification !== void 0 && object.confidenceBoostPerVerification !== null ? BigInt(object.confidenceBoostPerVerification.toString()) : BigInt(0);
    message.confidenceThreshold = object.confidenceThreshold !== void 0 && object.confidenceThreshold !== null ? BigInt(object.confidenceThreshold.toString()) : BigInt(0);
    message.quorumThreshold = object.quorumThreshold !== void 0 && object.quorumThreshold !== null ? BigInt(object.quorumThreshold.toString()) : BigInt(0);
    message.wrongVerificationSlashBps = object.wrongVerificationSlashBps !== void 0 && object.wrongVerificationSlashBps !== null ? BigInt(object.wrongVerificationSlashBps.toString()) : BigInt(0);
    message.missedRevealSlashBps = object.missedRevealSlashBps !== void 0 && object.missedRevealSlashBps !== null ? BigInt(object.missedRevealSlashBps.toString()) : BigInt(0);
    message.equivocationSlashBps = object.equivocationSlashBps !== void 0 && object.equivocationSlashBps !== null ? BigInt(object.equivocationSlashBps.toString()) : BigInt(0);
    message.invalidClaimSlashBps = object.invalidClaimSlashBps !== void 0 && object.invalidClaimSlashBps !== null ? BigInt(object.invalidClaimSlashBps.toString()) : BigInt(0);
    message.verificationReward = object.verificationReward ?? "";
    message.verificationRewardDecayBps = object.verificationRewardDecayBps !== void 0 && object.verificationRewardDecayBps !== null ? BigInt(object.verificationRewardDecayBps.toString()) : BigInt(0);
    message.minClaimTextLength = object.minClaimTextLength !== void 0 && object.minClaimTextLength !== null ? BigInt(object.minClaimTextLength.toString()) : BigInt(0);
    message.maxClaimTextLength = object.maxClaimTextLength !== void 0 && object.maxClaimTextLength !== null ? BigInt(object.maxClaimTextLength.toString()) : BigInt(0);
    message.minReviewFee = object.minReviewFee ?? "";
    message.adversarialVerificationEnabled = object.adversarialVerificationEnabled ?? false;
    message.provisionalThreshold = object.provisionalThreshold !== void 0 && object.provisionalThreshold !== null ? BigInt(object.provisionalThreshold.toString()) : BigInt(0);
    message.rejectThreshold = object.rejectThreshold !== void 0 && object.rejectThreshold !== null ? BigInt(object.rejectThreshold.toString()) : BigInt(0);
    message.challengeDurationBlocks = object.challengeDurationBlocks !== void 0 && object.challengeDurationBlocks !== null ? BigInt(object.challengeDurationBlocks.toString()) : BigInt(0);
    message.minChallengeStake = object.minChallengeStake ?? "";
    message.failedChallengeSlashBps = object.failedChallengeSlashBps !== void 0 && object.failedChallengeSlashBps !== null ? BigInt(object.failedChallengeSlashBps.toString()) : BigInt(0);
    message.successfulChallengeRewardBps = object.successfulChallengeRewardBps !== void 0 && object.successfulChallengeRewardBps !== null ? BigInt(object.successfulChallengeRewardBps.toString()) : BigInt(0);
    message.maxConcurrentChallenges = object.maxConcurrentChallenges !== void 0 && object.maxConcurrentChallenges !== null ? BigInt(object.maxConcurrentChallenges.toString()) : BigInt(0);
    message.citationShareBps = object.citationShareBps !== void 0 && object.citationShareBps !== null ? BigInt(object.citationShareBps.toString()) : BigInt(0);
    message.crossDomainBonusBps = object.crossDomainBonusBps !== void 0 && object.crossDomainBonusBps !== null ? BigInt(object.crossDomainBonusBps.toString()) : BigInt(0);
    message.maxFactsPerDomain = object.maxFactsPerDomain !== void 0 && object.maxFactsPerDomain !== null ? BigInt(object.maxFactsPerDomain.toString()) : BigInt(0);
    message.factExpiryBlocks = object.factExpiryBlocks !== void 0 && object.factExpiryBlocks !== null ? BigInt(object.factExpiryBlocks.toString()) : BigInt(0);
    message.crossStratumDiscountBps = object.crossStratumDiscountBps !== void 0 && object.crossStratumDiscountBps !== null ? BigInt(object.crossStratumDiscountBps.toString()) : BigInt(0);
    message.maxValidatorsPerRound = object.maxValidatorsPerRound !== void 0 && object.maxValidatorsPerRound !== null ? BigInt(object.maxValidatorsPerRound.toString()) : BigInt(0);
    message.confidenceGrowthEpoch = object.confidenceGrowthEpoch !== void 0 && object.confidenceGrowthEpoch !== null ? BigInt(object.confidenceGrowthEpoch.toString()) : BigInt(0);
    message.confidenceGrowthPerEpochBps = object.confidenceGrowthPerEpochBps !== void 0 && object.confidenceGrowthPerEpochBps !== null ? BigInt(object.confidenceGrowthPerEpochBps.toString()) : BigInt(0);
    message.maxSurvivalConfidence = object.maxSurvivalConfidence !== void 0 && object.maxSurvivalConfidence !== null ? BigInt(object.maxSurvivalConfidence.toString()) : BigInt(0);
    message.survivedChallengeConfidenceCap = object.survivedChallengeConfidenceCap !== void 0 && object.survivedChallengeConfidenceCap !== null ? BigInt(object.survivedChallengeConfidenceCap.toString()) : BigInt(0);
    message.maxApprenticeValidators = object.maxApprenticeValidators !== void 0 && object.maxApprenticeValidators !== null ? BigInt(object.maxApprenticeValidators.toString()) : BigInt(0);
    message.researchFundShareBps = object.researchFundShareBps !== void 0 && object.researchFundShareBps !== null ? BigInt(object.researchFundShareBps.toString()) : BigInt(0);
    message.fitnessEpochBlocks = object.fitnessEpochBlocks !== void 0 && object.fitnessEpochBlocks !== null ? BigInt(object.fitnessEpochBlocks.toString()) : BigInt(0);
    message.fitnessWeightQueryBps = object.fitnessWeightQueryBps !== void 0 && object.fitnessWeightQueryBps !== null ? BigInt(object.fitnessWeightQueryBps.toString()) : BigInt(0);
    message.fitnessWeightCitationBps = object.fitnessWeightCitationBps !== void 0 && object.fitnessWeightCitationBps !== null ? BigInt(object.fitnessWeightCitationBps.toString()) : BigInt(0);
    message.fitnessWeightBridgeBps = object.fitnessWeightBridgeBps !== void 0 && object.fitnessWeightBridgeBps !== null ? BigInt(object.fitnessWeightBridgeBps.toString()) : BigInt(0);
    message.fitnessWeightDepthBps = object.fitnessWeightDepthBps !== void 0 && object.fitnessWeightDepthBps !== null ? BigInt(object.fitnessWeightDepthBps.toString()) : BigInt(0);
    message.fitnessWeightPatronBps = object.fitnessWeightPatronBps !== void 0 && object.fitnessWeightPatronBps !== null ? BigInt(object.fitnessWeightPatronBps.toString()) : BigInt(0);
    message.fitnessWeightUniqueBps = object.fitnessWeightUniqueBps !== void 0 && object.fitnessWeightUniqueBps !== null ? BigInt(object.fitnessWeightUniqueBps.toString()) : BigInt(0);
    message.fitnessWeightAgeBps = object.fitnessWeightAgeBps !== void 0 && object.fitnessWeightAgeBps !== null ? BigInt(object.fitnessWeightAgeBps.toString()) : BigInt(0);
    message.fitnessInitialScore = object.fitnessInitialScore !== void 0 && object.fitnessInitialScore !== null ? BigInt(object.fitnessInitialScore.toString()) : BigInt(0);
    message.fitnessGraceEpochs = object.fitnessGraceEpochs !== void 0 && object.fitnessGraceEpochs !== null ? BigInt(object.fitnessGraceEpochs.toString()) : BigInt(0);
    message.bootstrapFundEnabled = object.bootstrapFundEnabled ?? false;
    message.bootstrapFundMaxPerAddress = object.bootstrapFundMaxPerAddress ?? "";
    message.bootstrapFundMaxPerEpoch = object.bootstrapFundMaxPerEpoch ?? "";
    message.bootstrapFundEpochBlocks = object.bootstrapFundEpochBlocks !== void 0 && object.bootstrapFundEpochBlocks !== null ? BigInt(object.bootstrapFundEpochBlocks.toString()) : BigInt(0);
    message.bootstrapFundFeeCap = object.bootstrapFundFeeCap ?? "";
    message.metabolismBaseCost = object.metabolismBaseCost !== void 0 && object.metabolismBaseCost !== null ? BigInt(object.metabolismBaseCost.toString()) : BigInt(0);
    message.metabolismContentLengthBps = object.metabolismContentLengthBps !== void 0 && object.metabolismContentLengthBps !== null ? BigInt(object.metabolismContentLengthBps.toString()) : BigInt(0);
    message.metabolismDomainCompetitionBps = object.metabolismDomainCompetitionBps !== void 0 && object.metabolismDomainCompetitionBps !== null ? BigInt(object.metabolismDomainCompetitionBps.toString()) : BigInt(0);
    message.metabolismEnergyPerQuery = object.metabolismEnergyPerQuery !== void 0 && object.metabolismEnergyPerQuery !== null ? BigInt(object.metabolismEnergyPerQuery.toString()) : BigInt(0);
    message.metabolismEnergyPerCitation = object.metabolismEnergyPerCitation !== void 0 && object.metabolismEnergyPerCitation !== null ? BigInt(object.metabolismEnergyPerCitation.toString()) : BigInt(0);
    message.metabolismEnergyPerPatronage = object.metabolismEnergyPerPatronage !== void 0 && object.metabolismEnergyPerPatronage !== null ? BigInt(object.metabolismEnergyPerPatronage.toString()) : BigInt(0);
    message.metabolismEnergyChallengeSurvival = object.metabolismEnergyChallengeSurvival !== void 0 && object.metabolismEnergyChallengeSurvival !== null ? BigInt(object.metabolismEnergyChallengeSurvival.toString()) : BigInt(0);
    message.metabolismEnergyCap = object.metabolismEnergyCap !== void 0 && object.metabolismEnergyCap !== null ? BigInt(object.metabolismEnergyCap.toString()) : BigInt(0);
    message.metabolismInitialEnergy = object.metabolismInitialEnergy !== void 0 && object.metabolismInitialEnergy !== null ? BigInt(object.metabolismInitialEnergy.toString()) : BigInt(0);
    message.metabolismAtRiskEpochs = object.metabolismAtRiskEpochs !== void 0 && object.metabolismAtRiskEpochs !== null ? BigInt(object.metabolismAtRiskEpochs.toString()) : BigInt(0);
    message.metabolismExpiredToPrunedEpochs = object.metabolismExpiredToPrunedEpochs !== void 0 && object.metabolismExpiredToPrunedEpochs !== null ? BigInt(object.metabolismExpiredToPrunedEpochs.toString()) : BigInt(0);
    message.reproductionRoyaltyBps = object.reproductionRoyaltyBps !== void 0 && object.reproductionRoyaltyBps !== null ? BigInt(object.reproductionRoyaltyBps.toString()) : BigInt(0);
    message.reproductionRoyaltyDecayBps = object.reproductionRoyaltyDecayBps !== void 0 && object.reproductionRoyaltyDecayBps !== null ? BigInt(object.reproductionRoyaltyDecayBps.toString()) : BigInt(0);
    message.reproductionMaxRoyaltyDepth = object.reproductionMaxRoyaltyDepth !== void 0 && object.reproductionMaxRoyaltyDepth !== null ? BigInt(object.reproductionMaxRoyaltyDepth.toString()) : BigInt(0);
    message.reproductionParentEnergyBonus = object.reproductionParentEnergyBonus !== void 0 && object.reproductionParentEnergyBonus !== null ? BigInt(object.reproductionParentEnergyBonus.toString()) : BigInt(0);
    message.reproductionChildFitnessInheritanceBps = object.reproductionChildFitnessInheritanceBps !== void 0 && object.reproductionChildFitnessInheritanceBps !== null ? BigInt(object.reproductionChildFitnessInheritanceBps.toString()) : BigInt(0);
    message.reproductionMaxChildren = object.reproductionMaxChildren !== void 0 && object.reproductionMaxChildren !== null ? BigInt(object.reproductionMaxChildren.toString()) : BigInt(0);
    message.noveltyCommonKnowledgePenaltyBps = object.noveltyCommonKnowledgePenaltyBps !== void 0 && object.noveltyCommonKnowledgePenaltyBps !== null ? BigInt(object.noveltyCommonKnowledgePenaltyBps.toString()) : BigInt(0);
    message.noveltySubjectOverlapPenaltyBps = object.noveltySubjectOverlapPenaltyBps !== void 0 && object.noveltySubjectOverlapPenaltyBps !== null ? BigInt(object.noveltySubjectOverlapPenaltyBps.toString()) : BigInt(0);
    message.noveltyPrecisionBonusBps = object.noveltyPrecisionBonusBps !== void 0 && object.noveltyPrecisionBonusBps !== null ? BigInt(object.noveltyPrecisionBonusBps.toString()) : BigInt(0);
    message.noveltyCrossDomainBonusBps = object.noveltyCrossDomainBonusBps !== void 0 && object.noveltyCrossDomainBonusBps !== null ? BigInt(object.noveltyCrossDomainBonusBps.toString()) : BigInt(0);
    message.noveltyMaxOverlapFacts = object.noveltyMaxOverlapFacts !== void 0 && object.noveltyMaxOverlapFacts !== null ? BigInt(object.noveltyMaxOverlapFacts.toString()) : BigInt(0);
    message.demandBountyThreshold = object.demandBountyThreshold !== void 0 && object.demandBountyThreshold !== null ? BigInt(object.demandBountyThreshold.toString()) : BigInt(0);
    message.demandBountyBaseReward = object.demandBountyBaseReward ?? "";
    message.demandBountyPerQueryBonus = object.demandBountyPerQueryBonus ?? "";
    message.demandBountyExpiryEpochs = object.demandBountyExpiryEpochs !== void 0 && object.demandBountyExpiryEpochs !== null ? BigInt(object.demandBountyExpiryEpochs.toString()) : BigInt(0);
    message.demandMultiplierCap = object.demandMultiplierCap !== void 0 && object.demandMultiplierCap !== null ? BigInt(object.demandMultiplierCap.toString()) : BigInt(0);
    message.demandTrackingEnabled = object.demandTrackingEnabled ?? false;
    message.authorizedDemandReporters = object.authorizedDemandReporters?.map((e) => e) || [];
    message.competitionNicheDominanceBonusBps = object.competitionNicheDominanceBonusBps !== void 0 && object.competitionNicheDominanceBonusBps !== null ? BigInt(object.competitionNicheDominanceBonusBps.toString()) : BigInt(0);
    message.competitionRedundancyThresholdBps = object.competitionRedundancyThresholdBps !== void 0 && object.competitionRedundancyThresholdBps !== null ? BigInt(object.competitionRedundancyThresholdBps.toString()) : BigInt(0);
    message.competitionMaxNicheSize = object.competitionMaxNicheSize !== void 0 && object.competitionMaxNicheSize !== null ? BigInt(object.competitionMaxNicheSize.toString()) : BigInt(0);
    message.competitionSymbiosisBonusBps = object.competitionSymbiosisBonusBps !== void 0 && object.competitionSymbiosisBonusBps !== null ? BigInt(object.competitionSymbiosisBonusBps.toString()) : BigInt(0);
    message.fitnessWeightSatisfactionBps = object.fitnessWeightSatisfactionBps !== void 0 && object.fitnessWeightSatisfactionBps !== null ? BigInt(object.fitnessWeightSatisfactionBps.toString()) : BigInt(0);
    message.satisfactionMinRatings = object.satisfactionMinRatings !== void 0 && object.satisfactionMinRatings !== null ? BigInt(object.satisfactionMinRatings.toString()) : BigInt(0);
    message.diversityConformityAlertThreshold = object.diversityConformityAlertThreshold !== void 0 && object.diversityConformityAlertThreshold !== null ? BigInt(object.diversityConformityAlertThreshold.toString()) : BigInt(0);
    message.diversityConformityAlertEpochs = object.diversityConformityAlertEpochs !== void 0 && object.diversityConformityAlertEpochs !== null ? BigInt(object.diversityConformityAlertEpochs.toString()) : BigInt(0);
    message.vindicationRefundEnabled = object.vindicationRefundEnabled ?? false;
    message.vindicationBonusBps = object.vindicationBonusBps !== void 0 && object.vindicationBonusBps !== null ? BigInt(object.vindicationBonusBps.toString()) : BigInt(0);
    message.vindicationSlashBps = object.vindicationSlashBps !== void 0 && object.vindicationSlashBps !== null ? BigInt(object.vindicationSlashBps.toString()) : BigInt(0);
    message.vindicationWindowBlocks = object.vindicationWindowBlocks !== void 0 && object.vindicationWindowBlocks !== null ? BigInt(object.vindicationWindowBlocks.toString()) : BigInt(0);
    message.metabolismActiveThreshold = object.metabolismActiveThreshold !== void 0 && object.metabolismActiveThreshold !== null ? BigInt(object.metabolismActiveThreshold.toString()) : BigInt(0);
    message.metabolismExtinctionThreshold = object.metabolismExtinctionThreshold !== void 0 && object.metabolismExtinctionThreshold !== null ? BigInt(object.metabolismExtinctionThreshold.toString()) : BigInt(0);
    message.maxConfidence = object.maxConfidence !== void 0 && object.maxConfidence !== null ? BigInt(object.maxConfidence.toString()) : BigInt(0);
    message.humanEmpiricalBonusBps = object.humanEmpiricalBonusBps !== void 0 && object.humanEmpiricalBonusBps !== null ? BigInt(object.humanEmpiricalBonusBps.toString()) : BigInt(0);
    message.agentComputationalBonusBps = object.agentComputationalBonusBps !== void 0 && object.agentComputationalBonusBps !== null ? BigInt(object.agentComputationalBonusBps.toString()) : BigInt(0);
    message.agentVerificationBonusBps = object.agentVerificationBonusBps !== void 0 && object.agentVerificationBonusBps !== null ? BigInt(object.agentVerificationBonusBps.toString()) : BigInt(0);
    message.humanPatronageBonusBps = object.humanPatronageBonusBps !== void 0 && object.humanPatronageBonusBps !== null ? BigInt(object.humanPatronageBonusBps.toString()) : BigInt(0);
    message.dualValidationBonusBps = object.dualValidationBonusBps !== void 0 && object.dualValidationBonusBps !== null ? BigInt(object.dualValidationBonusBps.toString()) : BigInt(0);
    message.domainBaseCapacity = object.domainBaseCapacity !== void 0 && object.domainBaseCapacity !== null ? BigInt(object.domainBaseCapacity.toString()) : BigInt(0);
    message.domainCapacityGrowthPerCitation = object.domainCapacityGrowthPerCitation !== void 0 && object.domainCapacityGrowthPerCitation !== null ? BigInt(object.domainCapacityGrowthPerCitation.toString()) : BigInt(0);
    message.overcrowdingDecayMultiplierBps = object.overcrowdingDecayMultiplierBps !== void 0 && object.overcrowdingDecayMultiplierBps !== null ? BigInt(object.overcrowdingDecayMultiplierBps.toString()) : BigInt(0);
    message.underpopulationBirthBonusBps = object.underpopulationBirthBonusBps !== void 0 && object.underpopulationBirthBonusBps !== null ? BigInt(object.underpopulationBirthBonusBps.toString()) : BigInt(0);
    message.epistemicTemperatureDecayBps = object.epistemicTemperatureDecayBps !== void 0 && object.epistemicTemperatureDecayBps !== null ? BigInt(object.epistemicTemperatureDecayBps.toString()) : BigInt(0);
    message.epistemicConformityCoolingBps = object.epistemicConformityCoolingBps !== void 0 && object.epistemicConformityCoolingBps !== null ? BigInt(object.epistemicConformityCoolingBps.toString()) : BigInt(0);
    message.epistemicVindicationHeatingBps = object.epistemicVindicationHeatingBps !== void 0 && object.epistemicVindicationHeatingBps !== null ? BigInt(object.epistemicVindicationHeatingBps.toString()) : BigInt(0);
    message.epistemicColdConfidenceCapBps = object.epistemicColdConfidenceCapBps !== void 0 && object.epistemicColdConfidenceCapBps !== null ? BigInt(object.epistemicColdConfidenceCapBps.toString()) : BigInt(0);
    message.epistemicHotConfidenceGrowthBps = object.epistemicHotConfidenceGrowthBps !== void 0 && object.epistemicHotConfidenceGrowthBps !== null ? BigInt(object.epistemicHotConfidenceGrowthBps.toString()) : BigInt(0);
    message.epistemicTemperatureWindowBlocks = object.epistemicTemperatureWindowBlocks !== void 0 && object.epistemicTemperatureWindowBlocks !== null ? BigInt(object.epistemicTemperatureWindowBlocks.toString()) : BigInt(0);
    message.roleElasticityMinCalls = object.roleElasticityMinCalls !== void 0 && object.roleElasticityMinCalls !== null ? BigInt(object.roleElasticityMinCalls.toString()) : BigInt(0);
    message.roleElasticityMaxMultiplierBps = object.roleElasticityMaxMultiplierBps !== void 0 && object.roleElasticityMaxMultiplierBps !== null ? BigInt(object.roleElasticityMaxMultiplierBps.toString()) : BigInt(0);
    message.roleElasticityMinMultiplierBps = object.roleElasticityMinMultiplierBps !== void 0 && object.roleElasticityMinMultiplierBps !== null ? BigInt(object.roleElasticityMinMultiplierBps.toString()) : BigInt(0);
    message.roleElasticityDecayEpochs = object.roleElasticityDecayEpochs !== void 0 && object.roleElasticityDecayEpochs !== null ? BigInt(object.roleElasticityDecayEpochs.toString()) : BigInt(0);
    message.mentorshipDividendEnergy = object.mentorshipDividendEnergy !== void 0 && object.mentorshipDividendEnergy !== null ? BigInt(object.mentorshipDividendEnergy.toString()) : BigInt(0);
    message.mentorshipCapacityBonus = object.mentorshipCapacityBonus !== void 0 && object.mentorshipCapacityBonus !== null ? BigInt(object.mentorshipCapacityBonus.toString()) : BigInt(0);
    message.socialSaturationThreshold = object.socialSaturationThreshold !== void 0 && object.socialSaturationThreshold !== null ? BigInt(object.socialSaturationThreshold.toString()) : BigInt(0);
    message.observationWindowBlocks = object.observationWindowBlocks !== void 0 && object.observationWindowBlocks !== null ? BigInt(object.observationWindowBlocks.toString()) : BigInt(0);
    message.minHeadcountAgreement = object.minHeadcountAgreement !== void 0 && object.minHeadcountAgreement !== null ? BigInt(object.minHeadcountAgreement.toString()) : BigInt(0);
    message.challengeConfidenceScalingBps = object.challengeConfidenceScalingBps !== void 0 && object.challengeConfidenceScalingBps !== null ? BigInt(object.challengeConfidenceScalingBps.toString()) : BigInt(0);
    message.independenceRewardStrengthBps = object.independenceRewardStrengthBps !== void 0 && object.independenceRewardStrengthBps !== null ? BigInt(object.independenceRewardStrengthBps.toString()) : BigInt(0);
    message.reformulationMinPanelVotes = object.reformulationMinPanelVotes !== void 0 && object.reformulationMinPanelVotes !== null ? BigInt(object.reformulationMinPanelVotes.toString()) : BigInt(0);
    message.reformulationConsensusBps = object.reformulationConsensusBps !== void 0 && object.reformulationConsensusBps !== null ? BigInt(object.reformulationConsensusBps.toString()) : BigInt(0);
    message.reformulationSuperiorBonusBps = object.reformulationSuperiorBonusBps !== void 0 && object.reformulationSuperiorBonusBps !== null ? BigInt(object.reformulationSuperiorBonusBps.toString()) : BigInt(0);
    message.augmentationExpiryFeeBps = object.augmentationExpiryFeeBps !== void 0 && object.augmentationExpiryFeeBps !== null ? BigInt(object.augmentationExpiryFeeBps.toString()) : BigInt(0);
    message.methodologyNormalizationBps = Object.entries(object.methodologyNormalizationBps ?? {}).reduce((acc, [key, value]) => {
      if (value !== void 0) {
        acc[key] = BigInt(value.toString());
      }
      return acc;
    }, {});
    message.vindicationTvwMultiplierBps = object.vindicationTvwMultiplierBps !== void 0 && object.vindicationTvwMultiplierBps !== null ? BigInt(object.vindicationTvwMultiplierBps.toString()) : BigInt(0);
    message.disprovalClawbackBps = object.disprovalClawbackBps !== void 0 && object.disprovalClawbackBps !== null ? BigInt(object.disprovalClawbackBps.toString()) : BigInt(0);
    message.disprovalClawbackWindowEpochs = object.disprovalClawbackWindowEpochs !== void 0 && object.disprovalClawbackWindowEpochs !== null ? BigInt(object.disprovalClawbackWindowEpochs.toString()) : BigInt(0);
    message.trainingFundCalibrationFloorBps = object.trainingFundCalibrationFloorBps !== void 0 && object.trainingFundCalibrationFloorBps !== null ? BigInt(object.trainingFundCalibrationFloorBps.toString()) : BigInt(0);
    message.trainingFundVestingEpochs = object.trainingFundVestingEpochs !== void 0 && object.trainingFundVestingEpochs !== null ? BigInt(object.trainingFundVestingEpochs.toString()) : BigInt(0);
    message.trainingFundMethodologyDiversityBonusBps = object.trainingFundMethodologyDiversityBonusBps !== void 0 && object.trainingFundMethodologyDiversityBonusBps !== null ? BigInt(object.trainingFundMethodologyDiversityBonusBps.toString()) : BigInt(0);
    message.trainingFundBaseReward = object.trainingFundBaseReward ?? "";
    message.contributionChallengeBond = object.contributionChallengeBond ?? "";
    message.contributionChallengeRewardMultiplierBps = object.contributionChallengeRewardMultiplierBps !== void 0 && object.contributionChallengeRewardMultiplierBps !== null ? BigInt(object.contributionChallengeRewardMultiplierBps.toString()) : BigInt(0);
    message.sponsorVetoForfeitBps = object.sponsorVetoForfeitBps !== void 0 && object.sponsorVetoForfeitBps !== null ? BigInt(object.sponsorVetoForfeitBps.toString()) : BigInt(0);
    message.maxPauseDurationBlocks = object.maxPauseDurationBlocks !== void 0 && object.maxPauseDurationBlocks !== null ? BigInt(object.maxPauseDurationBlocks.toString()) : BigInt(0);
    message.probeInvitationIdleThresholdBlocks = object.probeInvitationIdleThresholdBlocks !== void 0 && object.probeInvitationIdleThresholdBlocks !== null ? BigInt(object.probeInvitationIdleThresholdBlocks.toString()) : BigInt(0);
    message.probeInvitationMinConfidenceBps = object.probeInvitationMinConfidenceBps !== void 0 && object.probeInvitationMinConfidenceBps !== null ? BigInt(object.probeInvitationMinConfidenceBps.toString()) : BigInt(0);
    message.probeInvitationBatchSize = object.probeInvitationBatchSize ?? 0;
    message.probeInvitationReinviteCooldown = object.probeInvitationReinviteCooldown !== void 0 && object.probeInvitationReinviteCooldown !== null ? BigInt(object.probeInvitationReinviteCooldown.toString()) : BigInt(0);
    message.probeBountyMintPerBlock = object.probeBountyMintPerBlock ?? "";
    message.probeBountyMaxPoolSize = object.probeBountyMaxPoolSize ?? "";
    message.invitationBonusAmount = object.invitationBonusAmount ?? "";
    message.guardianAddresses = object.guardianAddresses?.map((e) => e) || [];
    message.addFactVetoWindowBlocks = object.addFactVetoWindowBlocks !== void 0 && object.addFactVetoWindowBlocks !== null ? BigInt(object.addFactVetoWindowBlocks.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/knowledge/v1/tx.ts
function createBaseMsgSubmitClaim() {
  return {
    submitter: "",
    factContent: "",
    domain: "",
    category: "",
    stake: "",
    references: [],
    partnershipId: "",
    claimType: 0,
    relations: [],
    structure: void 0,
    canonicalForm: "",
    sponsored: false
  };
}
var MsgSubmitClaim = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitClaim",
  encode(message, writer = BinaryWriter.create()) {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.factContent !== "") {
      writer.uint32(18).string(message.factContent);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    if (message.stake !== "") {
      writer.uint32(42).string(message.stake);
    }
    for (const v of message.references) {
      writer.uint32(50).string(v);
    }
    if (message.partnershipId !== "") {
      writer.uint32(58).string(message.partnershipId);
    }
    if (message.claimType !== 0) {
      writer.uint32(64).int32(message.claimType);
    }
    for (const v of message.relations) {
      ClaimRelation.encode(v, writer.uint32(74).fork()).ldelim();
    }
    if (message.structure !== void 0) {
      ClaimStructure.encode(message.structure, writer.uint32(82).fork()).ldelim();
    }
    if (message.canonicalForm !== "") {
      writer.uint32(90).string(message.canonicalForm);
    }
    if (message.sponsored === true) {
      writer.uint32(96).bool(message.sponsored);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitClaim();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.factContent = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.category = reader.string();
          break;
        case 5:
          message.stake = reader.string();
          break;
        case 6:
          message.references.push(reader.string());
          break;
        case 7:
          message.partnershipId = reader.string();
          break;
        case 8:
          message.claimType = reader.int32();
          break;
        case 9:
          message.relations.push(ClaimRelation.decode(reader, reader.uint32()));
          break;
        case 10:
          message.structure = ClaimStructure.decode(reader, reader.uint32());
          break;
        case 11:
          message.canonicalForm = reader.string();
          break;
        case 12:
          message.sponsored = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitClaim();
    message.submitter = object.submitter ?? "";
    message.factContent = object.factContent ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    message.stake = object.stake ?? "";
    message.references = object.references?.map((e) => e) || [];
    message.partnershipId = object.partnershipId ?? "";
    message.claimType = object.claimType ?? 0;
    message.relations = object.relations?.map((e) => ClaimRelation.fromPartial(e)) || [];
    message.structure = object.structure !== void 0 && object.structure !== null ? ClaimStructure.fromPartial(object.structure) : void 0;
    message.canonicalForm = object.canonicalForm ?? "";
    message.sponsored = object.sponsored ?? false;
    return message;
  }
};
function createBaseMsgSubmitClaimResponse() {
  return {
    claimId: ""
  };
}
var MsgSubmitClaimResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitClaimResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.claimId !== "") {
      writer.uint32(10).string(message.claimId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitClaimResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitClaimResponse();
    message.claimId = object.claimId ?? "";
    return message;
  }
};
function createBaseMsgSubmitCommitment() {
  return {
    verifier: "",
    roundId: "",
    commitHash: new Uint8Array()
  };
}
var MsgSubmitCommitment = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitment",
  encode(message, writer = BinaryWriter.create()) {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.roundId !== "") {
      writer.uint32(18).string(message.roundId);
    }
    if (message.commitHash.length !== 0) {
      writer.uint32(26).bytes(message.commitHash);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitCommitment();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.roundId = reader.string();
          break;
        case 3:
          message.commitHash = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitCommitment();
    message.verifier = object.verifier ?? "";
    message.roundId = object.roundId ?? "";
    message.commitHash = object.commitHash ?? new Uint8Array();
    return message;
  }
};
function createBaseMsgSubmitCommitmentResponse() {
  return {};
}
var MsgSubmitCommitmentResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitmentResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitCommitmentResponse();
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
    const message = createBaseMsgSubmitCommitmentResponse();
    return message;
  }
};
function createBaseMsgSubmitReveal() {
  return {
    verifier: "",
    roundId: "",
    vote: "",
    salt: new Uint8Array(),
    confidence: BigInt(0)
  };
}
var MsgSubmitReveal = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitReveal",
  encode(message, writer = BinaryWriter.create()) {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.roundId !== "") {
      writer.uint32(18).string(message.roundId);
    }
    if (message.vote !== "") {
      writer.uint32(26).string(message.vote);
    }
    if (message.salt.length !== 0) {
      writer.uint32(34).bytes(message.salt);
    }
    if (message.confidence !== BigInt(0)) {
      writer.uint32(40).uint64(message.confidence);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitReveal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.roundId = reader.string();
          break;
        case 3:
          message.vote = reader.string();
          break;
        case 4:
          message.salt = reader.bytes();
          break;
        case 5:
          message.confidence = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitReveal();
    message.verifier = object.verifier ?? "";
    message.roundId = object.roundId ?? "";
    message.vote = object.vote ?? "";
    message.salt = object.salt ?? new Uint8Array();
    message.confidence = object.confidence !== void 0 && object.confidence !== null ? BigInt(object.confidence.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgSubmitRevealResponse() {
  return {};
}
var MsgSubmitRevealResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitRevealResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitRevealResponse();
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
    const message = createBaseMsgSubmitRevealResponse();
    return message;
  }
};
function createBaseMsgChallengeFact() {
  return {
    challenger: "",
    factId: "",
    stake: "",
    reason: "",
    evidenceIds: []
  };
}
var MsgChallengeFact = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeFact",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.stake !== "") {
      writer.uint32(26).string(message.stake);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    for (const v of message.evidenceIds) {
      writer.uint32(42).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.stake = reader.string();
          break;
        case 4:
          message.reason = reader.string();
          break;
        case 5:
          message.evidenceIds.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgChallengeFact();
    message.challenger = object.challenger ?? "";
    message.factId = object.factId ?? "";
    message.stake = object.stake ?? "";
    message.reason = object.reason ?? "";
    message.evidenceIds = object.evidenceIds?.map((e) => e) || [];
    return message;
  }
};
function createBaseMsgChallengeFactResponse() {
  return {
    roundId: ""
  };
}
var MsgChallengeFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeFactResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.roundId !== "") {
      writer.uint32(10).string(message.roundId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeFactResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.roundId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgChallengeFactResponse();
    message.roundId = object.roundId ?? "";
    return message;
  }
};
function createBaseMsgAddFact() {
  return {
    authority: "",
    content: "",
    domain: "",
    category: "",
    references: [],
    confidence: BigInt(0)
  };
}
var MsgAddFact = {
  typeUrl: "/zerone.knowledge.v1.MsgAddFact",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.content !== "") {
      writer.uint32(18).string(message.content);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    for (const v of message.references) {
      writer.uint32(42).string(v);
    }
    if (message.confidence !== BigInt(0)) {
      writer.uint32(48).uint64(message.confidence);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.content = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.category = reader.string();
          break;
        case 5:
          message.references.push(reader.string());
          break;
        case 6:
          message.confidence = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddFact();
    message.authority = object.authority ?? "";
    message.content = object.content ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    message.references = object.references?.map((e) => e) || [];
    message.confidence = object.confidence !== void 0 && object.confidence !== null ? BigInt(object.confidence.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAddFactResponse() {
  return {
    factId: ""
  };
}
var MsgAddFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAddFactResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.factId !== "") {
      writer.uint32(10).string(message.factId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddFactResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.factId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddFactResponse();
    message.factId = object.factId ?? "";
    return message;
  }
};
function createBaseMsgSubmitContradiction() {
  return {
    submitter: "",
    factId: "",
    counterClaim: "",
    stake: "",
    evidenceIds: [],
    reason: "",
    domain: "",
    category: ""
  };
}
var MsgSubmitContradiction = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitContradiction",
  encode(message, writer = BinaryWriter.create()) {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.counterClaim !== "") {
      writer.uint32(26).string(message.counterClaim);
    }
    if (message.stake !== "") {
      writer.uint32(34).string(message.stake);
    }
    for (const v of message.evidenceIds) {
      writer.uint32(42).string(v);
    }
    if (message.reason !== "") {
      writer.uint32(50).string(message.reason);
    }
    if (message.domain !== "") {
      writer.uint32(58).string(message.domain);
    }
    if (message.category !== "") {
      writer.uint32(66).string(message.category);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitContradiction();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.counterClaim = reader.string();
          break;
        case 4:
          message.stake = reader.string();
          break;
        case 5:
          message.evidenceIds.push(reader.string());
          break;
        case 6:
          message.reason = reader.string();
          break;
        case 7:
          message.domain = reader.string();
          break;
        case 8:
          message.category = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitContradiction();
    message.submitter = object.submitter ?? "";
    message.factId = object.factId ?? "";
    message.counterClaim = object.counterClaim ?? "";
    message.stake = object.stake ?? "";
    message.evidenceIds = object.evidenceIds?.map((e) => e) || [];
    message.reason = object.reason ?? "";
    message.domain = object.domain ?? "";
    message.category = object.category ?? "";
    return message;
  }
};
function createBaseMsgSubmitContradictionResponse() {
  return {
    counterFactId: ""
  };
}
var MsgSubmitContradictionResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitContradictionResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.counterFactId !== "") {
      writer.uint32(10).string(message.counterFactId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitContradictionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.counterFactId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitContradictionResponse();
    message.counterFactId = object.counterFactId ?? "";
    return message;
  }
};
function createBaseMsgPatronizeFact() {
  return {
    patron: "",
    factId: "",
    amount: "",
    durationBlocks: BigInt(0)
  };
}
var MsgPatronizeFact = {
  typeUrl: "/zerone.knowledge.v1.MsgPatronizeFact",
  encode(message, writer = BinaryWriter.create()) {
    if (message.patron !== "") {
      writer.uint32(10).string(message.patron);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.durationBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.durationBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgPatronizeFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.patron = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.durationBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgPatronizeFact();
    message.patron = object.patron ?? "";
    message.factId = object.factId ?? "";
    message.amount = object.amount ?? "";
    message.durationBlocks = object.durationBlocks !== void 0 && object.durationBlocks !== null ? BigInt(object.durationBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgPatronizeFactResponse() {
  return {};
}
var MsgPatronizeFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgPatronizeFactResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgPatronizeFactResponse();
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
    const message = createBaseMsgPatronizeFactResponse();
    return message;
  }
};
function createBaseMsgProposeDomain() {
  return {
    proposer: "",
    name: "",
    description: "",
    stratum: "",
    stake: ""
  };
}
var MsgProposeDomain = {
  typeUrl: "/zerone.knowledge.v1.MsgProposeDomain",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.stratum !== "") {
      writer.uint32(34).string(message.stratum);
    }
    if (message.stake !== "") {
      writer.uint32(42).string(message.stake);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.stratum = reader.string();
          break;
        case 5:
          message.stake = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeDomain();
    message.proposer = object.proposer ?? "";
    message.name = object.name ?? "";
    message.description = object.description ?? "";
    message.stratum = object.stratum ?? "";
    message.stake = object.stake ?? "";
    return message;
  }
};
function createBaseMsgProposeDomainResponse() {
  return {
    proposalId: ""
  };
}
var MsgProposeDomainResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgProposeDomainResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeDomainResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeDomainResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgEndorseDomainProposal() {
  return {
    endorser: "",
    proposalId: ""
  };
}
var MsgEndorseDomainProposal = {
  typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposal",
  encode(message, writer = BinaryWriter.create()) {
    if (message.endorser !== "") {
      writer.uint32(10).string(message.endorser);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgEndorseDomainProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.endorser = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgEndorseDomainProposal();
    message.endorser = object.endorser ?? "";
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgEndorseDomainProposalResponse() {
  return {};
}
var MsgEndorseDomainProposalResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposalResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgEndorseDomainProposalResponse();
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
    const message = createBaseMsgEndorseDomainProposalResponse();
    return message;
  }
};
function createBaseMsgChallengeDomainProposal() {
  return {
    challenger: "",
    proposalId: "",
    reason: ""
  };
}
var MsgChallengeDomainProposal = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposal",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeDomainProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgChallengeDomainProposal();
    message.challenger = object.challenger ?? "";
    message.proposalId = object.proposalId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgChallengeDomainProposalResponse() {
  return {};
}
var MsgChallengeDomainProposalResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposalResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeDomainProposalResponse();
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
    const message = createBaseMsgChallengeDomainProposalResponse();
    return message;
  }
};
function createBaseMsgRegisterStratum() {
  return {
    authority: "",
    name: "",
    description: "",
    confidenceCeiling: BigInt(0),
    parentStrata: []
  };
}
var MsgRegisterStratum = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterStratum",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.confidenceCeiling !== BigInt(0)) {
      writer.uint32(32).uint64(message.confidenceCeiling);
    }
    for (const v of message.parentStrata) {
      writer.uint32(42).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterStratum();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.confidenceCeiling = reader.uint64();
          break;
        case 5:
          message.parentStrata.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterStratum();
    message.authority = object.authority ?? "";
    message.name = object.name ?? "";
    message.description = object.description ?? "";
    message.confidenceCeiling = object.confidenceCeiling !== void 0 && object.confidenceCeiling !== null ? BigInt(object.confidenceCeiling.toString()) : BigInt(0);
    message.parentStrata = object.parentStrata?.map((e) => e) || [];
    return message;
  }
};
function createBaseMsgRegisterStratumResponse() {
  return {};
}
var MsgRegisterStratumResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterStratumResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterStratumResponse();
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
    const message = createBaseMsgRegisterStratumResponse();
    return message;
  }
};
function createBaseMsgChallengeProvisionalFact() {
  return {
    challenger: "",
    claimId: "",
    factId: "",
    stake: "",
    reason: "",
    evidenceIds: [],
    counterClaim: ""
  };
}
var MsgChallengeProvisionalFact = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFact",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.claimId !== "") {
      writer.uint32(18).string(message.claimId);
    }
    if (message.factId !== "") {
      writer.uint32(26).string(message.factId);
    }
    if (message.stake !== "") {
      writer.uint32(34).string(message.stake);
    }
    if (message.reason !== "") {
      writer.uint32(42).string(message.reason);
    }
    for (const v of message.evidenceIds) {
      writer.uint32(50).string(v);
    }
    if (message.counterClaim !== "") {
      writer.uint32(58).string(message.counterClaim);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeProvisionalFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.claimId = reader.string();
          break;
        case 3:
          message.factId = reader.string();
          break;
        case 4:
          message.stake = reader.string();
          break;
        case 5:
          message.reason = reader.string();
          break;
        case 6:
          message.evidenceIds.push(reader.string());
          break;
        case 7:
          message.counterClaim = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgChallengeProvisionalFact();
    message.challenger = object.challenger ?? "";
    message.claimId = object.claimId ?? "";
    message.factId = object.factId ?? "";
    message.stake = object.stake ?? "";
    message.reason = object.reason ?? "";
    message.evidenceIds = object.evidenceIds?.map((e) => e) || [];
    message.counterClaim = object.counterClaim ?? "";
    return message;
  }
};
function createBaseMsgChallengeProvisionalFactResponse() {
  return {
    challengeId: ""
  };
}
var MsgChallengeProvisionalFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFactResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challengeId !== "") {
      writer.uint32(10).string(message.challengeId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeProvisionalFactResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challengeId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgChallengeProvisionalFactResponse();
    message.challengeId = object.challengeId ?? "";
    return message;
  }
};
function createBaseMsgUpdateParams11() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams11 = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params12.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams11();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params12.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams11();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params12.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse11() {
  return {};
}
var MsgUpdateParamsResponse11 = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse11();
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
    const message = createBaseMsgUpdateParamsResponse11();
    return message;
  }
};
function createBaseMsgUpdateExtendedParams() {
  return {
    authority: "",
    paramsJson: ""
  };
}
var MsgUpdateExtendedParams = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.paramsJson !== "") {
      writer.uint32(18).string(message.paramsJson);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateExtendedParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.paramsJson = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateExtendedParams();
    message.authority = object.authority ?? "";
    message.paramsJson = object.paramsJson ?? "";
    return message;
  }
};
function createBaseMsgUpdateExtendedParamsResponse() {
  return {};
}
var MsgUpdateExtendedParamsResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateExtendedParamsResponse();
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
    const message = createBaseMsgUpdateExtendedParamsResponse();
    return message;
  }
};
function createBaseMsgProposeResearchFund() {
  return {
    proposer: "",
    title: "",
    description: "",
    amount: "",
    recipient: "",
    votingPeriodBlocks: BigInt(0)
  };
}
var MsgProposeResearchFund = {
  typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFund",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    if (message.recipient !== "") {
      writer.uint32(42).string(message.recipient);
    }
    if (message.votingPeriodBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.votingPeriodBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeResearchFund();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        case 5:
          message.recipient = reader.string();
          break;
        case 6:
          message.votingPeriodBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeResearchFund();
    message.proposer = object.proposer ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.amount = object.amount ?? "";
    message.recipient = object.recipient ?? "";
    message.votingPeriodBlocks = object.votingPeriodBlocks !== void 0 && object.votingPeriodBlocks !== null ? BigInt(object.votingPeriodBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgProposeResearchFundResponse() {
  return {
    proposalId: ""
  };
}
var MsgProposeResearchFundResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFundResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeResearchFundResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeResearchFundResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteResearchProposal() {
  return {
    voter: "",
    proposalId: "",
    vote: false
  };
}
var MsgVoteResearchProposal = {
  typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposal",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.vote === true) {
      writer.uint32(24).bool(message.vote);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResearchProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.vote = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteResearchProposal();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.vote = object.vote ?? false;
    return message;
  }
};
function createBaseMsgVoteResearchProposalResponse() {
  return {};
}
var MsgVoteResearchProposalResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposalResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResearchProposalResponse();
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
    const message = createBaseMsgVoteResearchProposalResponse();
    return message;
  }
};
function createBaseMsgExecuteResearchProposal() {
  return {
    authority: "",
    proposalId: ""
  };
}
var MsgExecuteResearchProposal = {
  typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposal",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgExecuteResearchProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgExecuteResearchProposal();
    message.authority = object.authority ?? "";
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgExecuteResearchProposalResponse() {
  return {};
}
var MsgExecuteResearchProposalResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposalResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgExecuteResearchProposalResponse();
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
    const message = createBaseMsgExecuteResearchProposalResponse();
    return message;
  }
};
function createBaseMsgAddCommonKnowledge() {
  return {
    authority: "",
    domain: "",
    subject: "",
    description: "",
    penaltyBps: BigInt(0)
  };
}
var MsgAddCommonKnowledge = {
  typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledge",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.subject !== "") {
      writer.uint32(26).string(message.subject);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.penaltyBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.penaltyBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddCommonKnowledge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.subject = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.penaltyBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddCommonKnowledge();
    message.authority = object.authority ?? "";
    message.domain = object.domain ?? "";
    message.subject = object.subject ?? "";
    message.description = object.description ?? "";
    message.penaltyBps = object.penaltyBps !== void 0 && object.penaltyBps !== null ? BigInt(object.penaltyBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAddCommonKnowledgeResponse() {
  return {
    id: ""
  };
}
var MsgAddCommonKnowledgeResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledgeResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAddCommonKnowledgeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAddCommonKnowledgeResponse();
    message.id = object.id ?? "";
    return message;
  }
};
function createBaseMsgRemoveCommonKnowledge() {
  return {
    authority: "",
    id: ""
  };
}
var MsgRemoveCommonKnowledge = {
  typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledge",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveCommonKnowledge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRemoveCommonKnowledge();
    message.authority = object.authority ?? "";
    message.id = object.id ?? "";
    return message;
  }
};
function createBaseMsgRemoveCommonKnowledgeResponse() {
  return {};
}
var MsgRemoveCommonKnowledgeResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledgeResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRemoveCommonKnowledgeResponse();
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
    const message = createBaseMsgRemoveCommonKnowledgeResponse();
    return message;
  }
};
function createBaseMsgReportDemand() {
  return {
    reporter: "",
    reports: []
  };
}
var MsgReportDemand = {
  typeUrl: "/zerone.knowledge.v1.MsgReportDemand",
  encode(message, writer = BinaryWriter.create()) {
    if (message.reporter !== "") {
      writer.uint32(10).string(message.reporter);
    }
    for (const v of message.reports) {
      DemandReport.encode(v, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgReportDemand();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.reporter = reader.string();
          break;
        case 2:
          message.reports.push(DemandReport.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgReportDemand();
    message.reporter = object.reporter ?? "";
    message.reports = object.reports?.map((e) => DemandReport.fromPartial(e)) || [];
    return message;
  }
};
function createBaseDemandReport() {
  return {
    domain: "",
    subject: "",
    queries: BigInt(0),
    fulfilled: BigInt(0),
    unfulfilled: BigInt(0)
  };
}
var DemandReport = {
  typeUrl: "/zerone.knowledge.v1.DemandReport",
  encode(message, writer = BinaryWriter.create()) {
    if (message.domain !== "") {
      writer.uint32(10).string(message.domain);
    }
    if (message.subject !== "") {
      writer.uint32(18).string(message.subject);
    }
    if (message.queries !== BigInt(0)) {
      writer.uint32(24).uint64(message.queries);
    }
    if (message.fulfilled !== BigInt(0)) {
      writer.uint32(32).uint64(message.fulfilled);
    }
    if (message.unfulfilled !== BigInt(0)) {
      writer.uint32(40).uint64(message.unfulfilled);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseDemandReport();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.domain = reader.string();
          break;
        case 2:
          message.subject = reader.string();
          break;
        case 3:
          message.queries = reader.uint64();
          break;
        case 4:
          message.fulfilled = reader.uint64();
          break;
        case 5:
          message.unfulfilled = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseDemandReport();
    message.domain = object.domain ?? "";
    message.subject = object.subject ?? "";
    message.queries = object.queries !== void 0 && object.queries !== null ? BigInt(object.queries.toString()) : BigInt(0);
    message.fulfilled = object.fulfilled !== void 0 && object.fulfilled !== null ? BigInt(object.fulfilled.toString()) : BigInt(0);
    message.unfulfilled = object.unfulfilled !== void 0 && object.unfulfilled !== null ? BigInt(object.unfulfilled.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgReportDemandResponse() {
  return {};
}
var MsgReportDemandResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgReportDemandResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgReportDemandResponse();
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
    const message = createBaseMsgReportDemandResponse();
    return message;
  }
};
function createBaseMsgRateFact() {
  return {
    rater: "",
    factId: "",
    useful: false,
    memo: ""
  };
}
var MsgRateFact = {
  typeUrl: "/zerone.knowledge.v1.MsgRateFact",
  encode(message, writer = BinaryWriter.create()) {
    if (message.rater !== "") {
      writer.uint32(10).string(message.rater);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.useful === true) {
      writer.uint32(24).bool(message.useful);
    }
    if (message.memo !== "") {
      writer.uint32(34).string(message.memo);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRateFact();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.rater = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.useful = reader.bool();
          break;
        case 4:
          message.memo = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRateFact();
    message.rater = object.rater ?? "";
    message.factId = object.factId ?? "";
    message.useful = object.useful ?? false;
    message.memo = object.memo ?? "";
    return message;
  }
};
function createBaseMsgRateFactResponse() {
  return {};
}
var MsgRateFactResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRateFactResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRateFactResponse();
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
    const message = createBaseMsgRateFactResponse();
    return message;
  }
};
function createBaseMsgRegisterTrainingPipeline() {
  return {
    operator: "",
    id: "",
    corpusSnapshotHeight: BigInt(0),
    tokenizerVersion: BigInt(0),
    methodologySetVersion: BigInt(0),
    recipeHash: "",
    description: "",
    corpusFilter: ""
  };
}
var MsgRegisterTrainingPipeline = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipeline",
  encode(message, writer = BinaryWriter.create()) {
    if (message.operator !== "") {
      writer.uint32(10).string(message.operator);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.corpusSnapshotHeight !== BigInt(0)) {
      writer.uint32(24).uint64(message.corpusSnapshotHeight);
    }
    if (message.tokenizerVersion !== BigInt(0)) {
      writer.uint32(32).uint64(message.tokenizerVersion);
    }
    if (message.methodologySetVersion !== BigInt(0)) {
      writer.uint32(40).uint64(message.methodologySetVersion);
    }
    if (message.recipeHash !== "") {
      writer.uint32(50).string(message.recipeHash);
    }
    if (message.description !== "") {
      writer.uint32(58).string(message.description);
    }
    if (message.corpusFilter !== "") {
      writer.uint32(66).string(message.corpusFilter);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterTrainingPipeline();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operator = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.corpusSnapshotHeight = reader.uint64();
          break;
        case 4:
          message.tokenizerVersion = reader.uint64();
          break;
        case 5:
          message.methodologySetVersion = reader.uint64();
          break;
        case 6:
          message.recipeHash = reader.string();
          break;
        case 7:
          message.description = reader.string();
          break;
        case 8:
          message.corpusFilter = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterTrainingPipeline();
    message.operator = object.operator ?? "";
    message.id = object.id ?? "";
    message.corpusSnapshotHeight = object.corpusSnapshotHeight !== void 0 && object.corpusSnapshotHeight !== null ? BigInt(object.corpusSnapshotHeight.toString()) : BigInt(0);
    message.tokenizerVersion = object.tokenizerVersion !== void 0 && object.tokenizerVersion !== null ? BigInt(object.tokenizerVersion.toString()) : BigInt(0);
    message.methodologySetVersion = object.methodologySetVersion !== void 0 && object.methodologySetVersion !== null ? BigInt(object.methodologySetVersion.toString()) : BigInt(0);
    message.recipeHash = object.recipeHash ?? "";
    message.description = object.description ?? "";
    message.corpusFilter = object.corpusFilter ?? "";
    return message;
  }
};
function createBaseMsgRegisterTrainingPipelineResponse() {
  return {};
}
var MsgRegisterTrainingPipelineResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipelineResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterTrainingPipelineResponse();
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
    const message = createBaseMsgRegisterTrainingPipelineResponse();
    return message;
  }
};
function createBaseMsgUpdateTrainingPipeline() {
  return {
    operator: "",
    id: "",
    newStatus: "",
    completedAtBlock: BigInt(0)
  };
}
var MsgUpdateTrainingPipeline = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipeline",
  encode(message, writer = BinaryWriter.create()) {
    if (message.operator !== "") {
      writer.uint32(10).string(message.operator);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.newStatus !== "") {
      writer.uint32(26).string(message.newStatus);
    }
    if (message.completedAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.completedAtBlock);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateTrainingPipeline();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operator = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.newStatus = reader.string();
          break;
        case 4:
          message.completedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateTrainingPipeline();
    message.operator = object.operator ?? "";
    message.id = object.id ?? "";
    message.newStatus = object.newStatus ?? "";
    message.completedAtBlock = object.completedAtBlock !== void 0 && object.completedAtBlock !== null ? BigInt(object.completedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgUpdateTrainingPipelineResponse() {
  return {};
}
var MsgUpdateTrainingPipelineResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipelineResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateTrainingPipelineResponse();
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
    const message = createBaseMsgUpdateTrainingPipelineResponse();
    return message;
  }
};
function createBaseMsgRegisterModelCard() {
  return {
    owner: "",
    id: "",
    name: "",
    pipelineId: "",
    deploymentAddress: "",
    parameterCount: BigInt(0),
    route: "",
    baseModel: "",
    evalAcceptanceRateBps: BigInt(0),
    evalCorroborationRateBps: BigInt(0),
    evalSampleSize: BigInt(0),
    specialisedMethodId: ""
  };
}
var MsgRegisterModelCard = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCard",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    if (message.pipelineId !== "") {
      writer.uint32(34).string(message.pipelineId);
    }
    if (message.deploymentAddress !== "") {
      writer.uint32(42).string(message.deploymentAddress);
    }
    if (message.parameterCount !== BigInt(0)) {
      writer.uint32(48).uint64(message.parameterCount);
    }
    if (message.route !== "") {
      writer.uint32(58).string(message.route);
    }
    if (message.baseModel !== "") {
      writer.uint32(66).string(message.baseModel);
    }
    if (message.evalAcceptanceRateBps !== BigInt(0)) {
      writer.uint32(72).uint64(message.evalAcceptanceRateBps);
    }
    if (message.evalCorroborationRateBps !== BigInt(0)) {
      writer.uint32(80).uint64(message.evalCorroborationRateBps);
    }
    if (message.evalSampleSize !== BigInt(0)) {
      writer.uint32(88).uint64(message.evalSampleSize);
    }
    if (message.specialisedMethodId !== "") {
      writer.uint32(98).string(message.specialisedMethodId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterModelCard();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.name = reader.string();
          break;
        case 4:
          message.pipelineId = reader.string();
          break;
        case 5:
          message.deploymentAddress = reader.string();
          break;
        case 6:
          message.parameterCount = reader.uint64();
          break;
        case 7:
          message.route = reader.string();
          break;
        case 8:
          message.baseModel = reader.string();
          break;
        case 9:
          message.evalAcceptanceRateBps = reader.uint64();
          break;
        case 10:
          message.evalCorroborationRateBps = reader.uint64();
          break;
        case 11:
          message.evalSampleSize = reader.uint64();
          break;
        case 12:
          message.specialisedMethodId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterModelCard();
    message.owner = object.owner ?? "";
    message.id = object.id ?? "";
    message.name = object.name ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.deploymentAddress = object.deploymentAddress ?? "";
    message.parameterCount = object.parameterCount !== void 0 && object.parameterCount !== null ? BigInt(object.parameterCount.toString()) : BigInt(0);
    message.route = object.route ?? "";
    message.baseModel = object.baseModel ?? "";
    message.evalAcceptanceRateBps = object.evalAcceptanceRateBps !== void 0 && object.evalAcceptanceRateBps !== null ? BigInt(object.evalAcceptanceRateBps.toString()) : BigInt(0);
    message.evalCorroborationRateBps = object.evalCorroborationRateBps !== void 0 && object.evalCorroborationRateBps !== null ? BigInt(object.evalCorroborationRateBps.toString()) : BigInt(0);
    message.evalSampleSize = object.evalSampleSize !== void 0 && object.evalSampleSize !== null ? BigInt(object.evalSampleSize.toString()) : BigInt(0);
    message.specialisedMethodId = object.specialisedMethodId ?? "";
    return message;
  }
};
function createBaseMsgRegisterModelCardResponse() {
  return {};
}
var MsgRegisterModelCardResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCardResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterModelCardResponse();
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
    const message = createBaseMsgRegisterModelCardResponse();
    return message;
  }
};
function createBaseMsgUpdateModelCard() {
  return {
    owner: "",
    id: "",
    evalAcceptanceRateBps: BigInt(0),
    evalCorroborationRateBps: BigInt(0),
    evalSampleSize: BigInt(0),
    name: ""
  };
}
var MsgUpdateModelCard = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCard",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.evalAcceptanceRateBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.evalAcceptanceRateBps);
    }
    if (message.evalCorroborationRateBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.evalCorroborationRateBps);
    }
    if (message.evalSampleSize !== BigInt(0)) {
      writer.uint32(40).uint64(message.evalSampleSize);
    }
    if (message.name !== "") {
      writer.uint32(50).string(message.name);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateModelCard();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.evalAcceptanceRateBps = reader.uint64();
          break;
        case 4:
          message.evalCorroborationRateBps = reader.uint64();
          break;
        case 5:
          message.evalSampleSize = reader.uint64();
          break;
        case 6:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateModelCard();
    message.owner = object.owner ?? "";
    message.id = object.id ?? "";
    message.evalAcceptanceRateBps = object.evalAcceptanceRateBps !== void 0 && object.evalAcceptanceRateBps !== null ? BigInt(object.evalAcceptanceRateBps.toString()) : BigInt(0);
    message.evalCorroborationRateBps = object.evalCorroborationRateBps !== void 0 && object.evalCorroborationRateBps !== null ? BigInt(object.evalCorroborationRateBps.toString()) : BigInt(0);
    message.evalSampleSize = object.evalSampleSize !== void 0 && object.evalSampleSize !== null ? BigInt(object.evalSampleSize.toString()) : BigInt(0);
    message.name = object.name ?? "";
    return message;
  }
};
function createBaseMsgUpdateModelCardResponse() {
  return {};
}
var MsgUpdateModelCardResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCardResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateModelCardResponse();
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
    const message = createBaseMsgUpdateModelCardResponse();
    return message;
  }
};
function createBaseMsgRetireModelCard() {
  return {
    owner: "",
    id: "",
    reason: ""
  };
}
var MsgRetireModelCard = {
  typeUrl: "/zerone.knowledge.v1.MsgRetireModelCard",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRetireModelCard();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRetireModelCard();
    message.owner = object.owner ?? "";
    message.id = object.id ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgRetireModelCardResponse() {
  return {};
}
var MsgRetireModelCardResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRetireModelCardResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRetireModelCardResponse();
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
    const message = createBaseMsgRetireModelCardResponse();
    return message;
  }
};
function createBaseMsgAmendTokenizerSpec() {
  return {
    authority: "",
    spec: void 0
  };
}
var MsgAmendTokenizerSpec = {
  typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpec",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.spec !== void 0) {
      TokenizerSpec.encode(message.spec, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAmendTokenizerSpec();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.spec = TokenizerSpec.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAmendTokenizerSpec();
    message.authority = object.authority ?? "";
    message.spec = object.spec !== void 0 && object.spec !== null ? TokenizerSpec.fromPartial(object.spec) : void 0;
    return message;
  }
};
function createBaseMsgAmendTokenizerSpecResponse() {
  return {
    newVersion: BigInt(0)
  };
}
var MsgAmendTokenizerSpecResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpecResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.newVersion !== BigInt(0)) {
      writer.uint32(8).uint64(message.newVersion);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAmendTokenizerSpecResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newVersion = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAmendTokenizerSpecResponse();
    message.newVersion = object.newVersion !== void 0 && object.newVersion !== null ? BigInt(object.newVersion.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAttributeContributions() {
  return {
    owner: "",
    modelId: "",
    factIds: [],
    totalWeight: BigInt(0)
  };
}
var MsgAttributeContributions = {
  typeUrl: "/zerone.knowledge.v1.MsgAttributeContributions",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.modelId !== "") {
      writer.uint32(18).string(message.modelId);
    }
    for (const v of message.factIds) {
      writer.uint32(26).string(v);
    }
    if (message.totalWeight !== BigInt(0)) {
      writer.uint32(32).uint64(message.totalWeight);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAttributeContributions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.modelId = reader.string();
          break;
        case 3:
          message.factIds.push(reader.string());
          break;
        case 4:
          message.totalWeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAttributeContributions();
    message.owner = object.owner ?? "";
    message.modelId = object.modelId ?? "";
    message.factIds = object.factIds?.map((e) => e) || [];
    message.totalWeight = object.totalWeight !== void 0 && object.totalWeight !== null ? BigInt(object.totalWeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAttributeContributionsResponse() {
  return {
    recorded: 0
  };
}
var MsgAttributeContributionsResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAttributeContributionsResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.recorded !== 0) {
      writer.uint32(8).uint32(message.recorded);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAttributeContributionsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.recorded = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAttributeContributionsResponse();
    message.recorded = object.recorded ?? 0;
    return message;
  }
};
function createBaseMsgAttestTraining() {
  return {
    attester: "",
    pipelineId: "",
    flopsEstimate: BigInt(0),
    wallclockSeconds: BigInt(0),
    evalHash: "",
    signature: "",
    notes: ""
  };
}
var MsgAttestTraining = {
  typeUrl: "/zerone.knowledge.v1.MsgAttestTraining",
  encode(message, writer = BinaryWriter.create()) {
    if (message.attester !== "") {
      writer.uint32(10).string(message.attester);
    }
    if (message.pipelineId !== "") {
      writer.uint32(18).string(message.pipelineId);
    }
    if (message.flopsEstimate !== BigInt(0)) {
      writer.uint32(24).uint64(message.flopsEstimate);
    }
    if (message.wallclockSeconds !== BigInt(0)) {
      writer.uint32(32).uint64(message.wallclockSeconds);
    }
    if (message.evalHash !== "") {
      writer.uint32(42).string(message.evalHash);
    }
    if (message.signature !== "") {
      writer.uint32(50).string(message.signature);
    }
    if (message.notes !== "") {
      writer.uint32(58).string(message.notes);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAttestTraining();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.attester = reader.string();
          break;
        case 2:
          message.pipelineId = reader.string();
          break;
        case 3:
          message.flopsEstimate = reader.uint64();
          break;
        case 4:
          message.wallclockSeconds = reader.uint64();
          break;
        case 5:
          message.evalHash = reader.string();
          break;
        case 6:
          message.signature = reader.string();
          break;
        case 7:
          message.notes = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAttestTraining();
    message.attester = object.attester ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.flopsEstimate = object.flopsEstimate !== void 0 && object.flopsEstimate !== null ? BigInt(object.flopsEstimate.toString()) : BigInt(0);
    message.wallclockSeconds = object.wallclockSeconds !== void 0 && object.wallclockSeconds !== null ? BigInt(object.wallclockSeconds.toString()) : BigInt(0);
    message.evalHash = object.evalHash ?? "";
    message.signature = object.signature ?? "";
    message.notes = object.notes ?? "";
    return message;
  }
};
function createBaseMsgAttestTrainingResponse() {
  return {};
}
var MsgAttestTrainingResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAttestTrainingResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAttestTrainingResponse();
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
    const message = createBaseMsgAttestTrainingResponse();
    return message;
  }
};
function createBaseMsgCreateAugmentationBounty() {
  return {
    sponsor: "",
    id: "",
    targetFactId: "",
    rewardPerVariant: BigInt(0),
    maxVariants: 0,
    expiresAtBlock: BigInt(0),
    description: ""
  };
}
var MsgCreateAugmentationBounty = {
  typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBounty",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sponsor !== "") {
      writer.uint32(10).string(message.sponsor);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.targetFactId !== "") {
      writer.uint32(26).string(message.targetFactId);
    }
    if (message.rewardPerVariant !== BigInt(0)) {
      writer.uint32(32).uint64(message.rewardPerVariant);
    }
    if (message.maxVariants !== 0) {
      writer.uint32(40).uint32(message.maxVariants);
    }
    if (message.expiresAtBlock !== BigInt(0)) {
      writer.uint32(48).uint64(message.expiresAtBlock);
    }
    if (message.description !== "") {
      writer.uint32(58).string(message.description);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateAugmentationBounty();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sponsor = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.targetFactId = reader.string();
          break;
        case 4:
          message.rewardPerVariant = reader.uint64();
          break;
        case 5:
          message.maxVariants = reader.uint32();
          break;
        case 6:
          message.expiresAtBlock = reader.uint64();
          break;
        case 7:
          message.description = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateAugmentationBounty();
    message.sponsor = object.sponsor ?? "";
    message.id = object.id ?? "";
    message.targetFactId = object.targetFactId ?? "";
    message.rewardPerVariant = object.rewardPerVariant !== void 0 && object.rewardPerVariant !== null ? BigInt(object.rewardPerVariant.toString()) : BigInt(0);
    message.maxVariants = object.maxVariants ?? 0;
    message.expiresAtBlock = object.expiresAtBlock !== void 0 && object.expiresAtBlock !== null ? BigInt(object.expiresAtBlock.toString()) : BigInt(0);
    message.description = object.description ?? "";
    return message;
  }
};
function createBaseMsgCreateAugmentationBountyResponse() {
  return {};
}
var MsgCreateAugmentationBountyResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBountyResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateAugmentationBountyResponse();
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
    const message = createBaseMsgCreateAugmentationBountyResponse();
    return message;
  }
};
function createBaseMsgSubmitAugmentation() {
  return {
    submitter: "",
    id: "",
    bountyId: "",
    originalFactId: "",
    variantContent: "",
    variantReasoningTrace: ""
  };
}
var MsgSubmitAugmentation = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.bountyId !== "") {
      writer.uint32(26).string(message.bountyId);
    }
    if (message.originalFactId !== "") {
      writer.uint32(34).string(message.originalFactId);
    }
    if (message.variantContent !== "") {
      writer.uint32(42).string(message.variantContent);
    }
    if (message.variantReasoningTrace !== "") {
      writer.uint32(50).string(message.variantReasoningTrace);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.bountyId = reader.string();
          break;
        case 4:
          message.originalFactId = reader.string();
          break;
        case 5:
          message.variantContent = reader.string();
          break;
        case 6:
          message.variantReasoningTrace = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitAugmentation();
    message.submitter = object.submitter ?? "";
    message.id = object.id ?? "";
    message.bountyId = object.bountyId ?? "";
    message.originalFactId = object.originalFactId ?? "";
    message.variantContent = object.variantContent ?? "";
    message.variantReasoningTrace = object.variantReasoningTrace ?? "";
    return message;
  }
};
function createBaseMsgSubmitAugmentationResponse() {
  return {};
}
var MsgSubmitAugmentationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentationResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitAugmentationResponse();
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
    const message = createBaseMsgSubmitAugmentationResponse();
    return message;
  }
};
function createBaseMsgAcceptAugmentation() {
  return {
    acceptor: "",
    augmentationId: "",
    note: ""
  };
}
var MsgAcceptAugmentation = {
  typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.acceptor !== "") {
      writer.uint32(10).string(message.acceptor);
    }
    if (message.augmentationId !== "") {
      writer.uint32(18).string(message.augmentationId);
    }
    if (message.note !== "") {
      writer.uint32(26).string(message.note);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAcceptAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.acceptor = reader.string();
          break;
        case 2:
          message.augmentationId = reader.string();
          break;
        case 3:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAcceptAugmentation();
    message.acceptor = object.acceptor ?? "";
    message.augmentationId = object.augmentationId ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgAcceptAugmentationResponse() {
  return {};
}
var MsgAcceptAugmentationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentationResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAcceptAugmentationResponse();
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
    const message = createBaseMsgAcceptAugmentationResponse();
    return message;
  }
};
function createBaseMsgVoteOnAugmentation() {
  return {
    verifier: "",
    augmentationId: "",
    vote: 0,
    rationale: ""
  };
}
var MsgVoteOnAugmentation = {
  typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.verifier !== "") {
      writer.uint32(10).string(message.verifier);
    }
    if (message.augmentationId !== "") {
      writer.uint32(18).string(message.augmentationId);
    }
    if (message.vote !== 0) {
      writer.uint32(24).int32(message.vote);
    }
    if (message.rationale !== "") {
      writer.uint32(34).string(message.rationale);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteOnAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verifier = reader.string();
          break;
        case 2:
          message.augmentationId = reader.string();
          break;
        case 3:
          message.vote = reader.int32();
          break;
        case 4:
          message.rationale = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteOnAugmentation();
    message.verifier = object.verifier ?? "";
    message.augmentationId = object.augmentationId ?? "";
    message.vote = object.vote ?? 0;
    message.rationale = object.rationale ?? "";
    return message;
  }
};
function createBaseMsgVoteOnAugmentationResponse() {
  return {
    verdictFinalized: false,
    finalizedVerdict: 0
  };
}
var MsgVoteOnAugmentationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentationResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.verdictFinalized === true) {
      writer.uint32(8).bool(message.verdictFinalized);
    }
    if (message.finalizedVerdict !== 0) {
      writer.uint32(16).int32(message.finalizedVerdict);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteOnAugmentationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.verdictFinalized = reader.bool();
          break;
        case 2:
          message.finalizedVerdict = reader.int32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteOnAugmentationResponse();
    message.verdictFinalized = object.verdictFinalized ?? false;
    message.finalizedVerdict = object.finalizedVerdict ?? 0;
    return message;
  }
};
function createBaseMsgSponsorVetoAugmentation() {
  return {
    sponsor: "",
    augmentationId: "",
    reason: ""
  };
}
var MsgSponsorVetoAugmentation = {
  typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sponsor !== "") {
      writer.uint32(10).string(message.sponsor);
    }
    if (message.augmentationId !== "") {
      writer.uint32(18).string(message.augmentationId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSponsorVetoAugmentation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sponsor = reader.string();
          break;
        case 2:
          message.augmentationId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSponsorVetoAugmentation();
    message.sponsor = object.sponsor ?? "";
    message.augmentationId = object.augmentationId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgSponsorVetoAugmentationResponse() {
  return {};
}
var MsgSponsorVetoAugmentationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentationResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSponsorVetoAugmentationResponse();
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
    const message = createBaseMsgSponsorVetoAugmentationResponse();
    return message;
  }
};
function createBaseMsgChallengeContribution() {
  return {
    challenger: "",
    modelId: "",
    disputedFactId: "",
    disputeType: "",
    evidence: "",
    id: ""
  };
}
var MsgChallengeContribution = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeContribution",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.modelId !== "") {
      writer.uint32(18).string(message.modelId);
    }
    if (message.disputedFactId !== "") {
      writer.uint32(26).string(message.disputedFactId);
    }
    if (message.disputeType !== "") {
      writer.uint32(34).string(message.disputeType);
    }
    if (message.evidence !== "") {
      writer.uint32(42).string(message.evidence);
    }
    if (message.id !== "") {
      writer.uint32(50).string(message.id);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeContribution();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.modelId = reader.string();
          break;
        case 3:
          message.disputedFactId = reader.string();
          break;
        case 4:
          message.disputeType = reader.string();
          break;
        case 5:
          message.evidence = reader.string();
          break;
        case 6:
          message.id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgChallengeContribution();
    message.challenger = object.challenger ?? "";
    message.modelId = object.modelId ?? "";
    message.disputedFactId = object.disputedFactId ?? "";
    message.disputeType = object.disputeType ?? "";
    message.evidence = object.evidence ?? "";
    message.id = object.id ?? "";
    return message;
  }
};
function createBaseMsgChallengeContributionResponse() {
  return {
    bondEscrowed: ""
  };
}
var MsgChallengeContributionResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgChallengeContributionResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.bondEscrowed !== "") {
      writer.uint32(10).string(message.bondEscrowed);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgChallengeContributionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.bondEscrowed = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgChallengeContributionResponse();
    message.bondEscrowed = object.bondEscrowed ?? "";
    return message;
  }
};
function createBaseMsgResolveContributionChallenge() {
  return {
    resolver: "",
    challengeId: "",
    uphold: false,
    note: ""
  };
}
var MsgResolveContributionChallenge = {
  typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallenge",
  encode(message, writer = BinaryWriter.create()) {
    if (message.resolver !== "") {
      writer.uint32(10).string(message.resolver);
    }
    if (message.challengeId !== "") {
      writer.uint32(18).string(message.challengeId);
    }
    if (message.uphold === true) {
      writer.uint32(24).bool(message.uphold);
    }
    if (message.note !== "") {
      writer.uint32(34).string(message.note);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveContributionChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.resolver = reader.string();
          break;
        case 2:
          message.challengeId = reader.string();
          break;
        case 3:
          message.uphold = reader.bool();
          break;
        case 4:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgResolveContributionChallenge();
    message.resolver = object.resolver ?? "";
    message.challengeId = object.challengeId ?? "";
    message.uphold = object.uphold ?? false;
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgResolveContributionChallengeResponse() {
  return {
    payoutToWinner: ""
  };
}
var MsgResolveContributionChallengeResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallengeResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.payoutToWinner !== "") {
      writer.uint32(10).string(message.payoutToWinner);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveContributionChallengeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.payoutToWinner = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgResolveContributionChallengeResponse();
    message.payoutToWinner = object.payoutToWinner ?? "";
    return message;
  }
};
function createBaseMsgClaimTrainingFundDisbursement() {
  return {
    claimant: "",
    modelId: "",
    id: ""
  };
}
var MsgClaimTrainingFundDisbursement = {
  typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement",
  encode(message, writer = BinaryWriter.create()) {
    if (message.claimant !== "") {
      writer.uint32(10).string(message.claimant);
    }
    if (message.modelId !== "") {
      writer.uint32(18).string(message.modelId);
    }
    if (message.id !== "") {
      writer.uint32(26).string(message.id);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimTrainingFundDisbursement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimant = reader.string();
          break;
        case 2:
          message.modelId = reader.string();
          break;
        case 3:
          message.id = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgClaimTrainingFundDisbursement();
    message.claimant = object.claimant ?? "";
    message.modelId = object.modelId ?? "";
    message.id = object.id ?? "";
    return message;
  }
};
function createBaseMsgClaimTrainingFundDisbursementResponse() {
  return {
    totalAmount: "",
    releasedAmount: "",
    vestingAmount: "",
    vestingEndBlock: BigInt(0)
  };
}
var MsgClaimTrainingFundDisbursementResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursementResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.totalAmount !== "") {
      writer.uint32(10).string(message.totalAmount);
    }
    if (message.releasedAmount !== "") {
      writer.uint32(18).string(message.releasedAmount);
    }
    if (message.vestingAmount !== "") {
      writer.uint32(26).string(message.vestingAmount);
    }
    if (message.vestingEndBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.vestingEndBlock);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimTrainingFundDisbursementResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalAmount = reader.string();
          break;
        case 2:
          message.releasedAmount = reader.string();
          break;
        case 3:
          message.vestingAmount = reader.string();
          break;
        case 4:
          message.vestingEndBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgClaimTrainingFundDisbursementResponse();
    message.totalAmount = object.totalAmount ?? "";
    message.releasedAmount = object.releasedAmount ?? "";
    message.vestingAmount = object.vestingAmount ?? "";
    message.vestingEndBlock = object.vestingEndBlock !== void 0 && object.vestingEndBlock !== null ? BigInt(object.vestingEndBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAmendTraceSchema() {
  return {
    authority: "",
    schema: void 0
  };
}
var MsgAmendTraceSchema = {
  typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchema",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.schema !== void 0) {
      TraceSchema.encode(message.schema, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAmendTraceSchema();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.schema = TraceSchema.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAmendTraceSchema();
    message.authority = object.authority ?? "";
    message.schema = object.schema !== void 0 && object.schema !== null ? TraceSchema.fromPartial(object.schema) : void 0;
    return message;
  }
};
function createBaseMsgAmendTraceSchemaResponse() {
  return {
    newVersion: BigInt(0)
  };
}
var MsgAmendTraceSchemaResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchemaResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.newVersion !== BigInt(0)) {
      writer.uint32(8).uint64(message.newVersion);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAmendTraceSchemaResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newVersion = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAmendTraceSchemaResponse();
    message.newVersion = object.newVersion !== void 0 && object.newVersion !== null ? BigInt(object.newVersion.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgCreateTrainingManifest() {
  return {
    creator: "",
    id: "",
    pipelineId: "",
    corpusSelector: void 0,
    description: "",
    parentManifestId: ""
  };
}
var MsgCreateTrainingManifest = {
  typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifest",
  encode(message, writer = BinaryWriter.create()) {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.pipelineId !== "") {
      writer.uint32(26).string(message.pipelineId);
    }
    if (message.corpusSelector !== void 0) {
      CorpusSelector.encode(message.corpusSelector, writer.uint32(34).fork()).ldelim();
    }
    if (message.description !== "") {
      writer.uint32(42).string(message.description);
    }
    if (message.parentManifestId !== "") {
      writer.uint32(50).string(message.parentManifestId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateTrainingManifest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.pipelineId = reader.string();
          break;
        case 4:
          message.corpusSelector = CorpusSelector.decode(reader, reader.uint32());
          break;
        case 5:
          message.description = reader.string();
          break;
        case 6:
          message.parentManifestId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateTrainingManifest();
    message.creator = object.creator ?? "";
    message.id = object.id ?? "";
    message.pipelineId = object.pipelineId ?? "";
    message.corpusSelector = object.corpusSelector !== void 0 && object.corpusSelector !== null ? CorpusSelector.fromPartial(object.corpusSelector) : void 0;
    message.description = object.description ?? "";
    message.parentManifestId = object.parentManifestId ?? "";
    return message;
  }
};
function createBaseMsgCreateTrainingManifestResponse() {
  return {
    totalIncluded: 0,
    factCount: 0,
    traceCount: 0,
    pairCount: 0,
    driftCount: 0,
    normativeCount: 0
  };
}
var MsgCreateTrainingManifestResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifestResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.totalIncluded !== 0) {
      writer.uint32(8).uint32(message.totalIncluded);
    }
    if (message.factCount !== 0) {
      writer.uint32(16).uint32(message.factCount);
    }
    if (message.traceCount !== 0) {
      writer.uint32(24).uint32(message.traceCount);
    }
    if (message.pairCount !== 0) {
      writer.uint32(32).uint32(message.pairCount);
    }
    if (message.driftCount !== 0) {
      writer.uint32(40).uint32(message.driftCount);
    }
    if (message.normativeCount !== 0) {
      writer.uint32(48).uint32(message.normativeCount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateTrainingManifestResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalIncluded = reader.uint32();
          break;
        case 2:
          message.factCount = reader.uint32();
          break;
        case 3:
          message.traceCount = reader.uint32();
          break;
        case 4:
          message.pairCount = reader.uint32();
          break;
        case 5:
          message.driftCount = reader.uint32();
          break;
        case 6:
          message.normativeCount = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateTrainingManifestResponse();
    message.totalIncluded = object.totalIncluded ?? 0;
    message.factCount = object.factCount ?? 0;
    message.traceCount = object.traceCount ?? 0;
    message.pairCount = object.pairCount ?? 0;
    message.driftCount = object.driftCount ?? 0;
    message.normativeCount = object.normativeCount ?? 0;
    return message;
  }
};
function createBaseMsgFinalizeTrainingManifest() {
  return {
    creator: "",
    manifestId: ""
  };
}
var MsgFinalizeTrainingManifest = {
  typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifest",
  encode(message, writer = BinaryWriter.create()) {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.manifestId !== "") {
      writer.uint32(18).string(message.manifestId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFinalizeTrainingManifest();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.manifestId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgFinalizeTrainingManifest();
    message.creator = object.creator ?? "";
    message.manifestId = object.manifestId ?? "";
    return message;
  }
};
function createBaseMsgFinalizeTrainingManifestResponse() {
  return {
    merkleRoot: ""
  };
}
var MsgFinalizeTrainingManifestResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifestResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.merkleRoot !== "") {
      writer.uint32(10).string(message.merkleRoot);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFinalizeTrainingManifestResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.merkleRoot = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgFinalizeTrainingManifestResponse();
    message.merkleRoot = object.merkleRoot ?? "";
    return message;
  }
};
function createBaseMsgBindManifestToAttestation() {
  return {
    creator: "",
    manifestId: "",
    attestationId: ""
  };
}
var MsgBindManifestToAttestation = {
  typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.manifestId !== "") {
      writer.uint32(18).string(message.manifestId);
    }
    if (message.attestationId !== "") {
      writer.uint32(26).string(message.attestationId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgBindManifestToAttestation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.manifestId = reader.string();
          break;
        case 3:
          message.attestationId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgBindManifestToAttestation();
    message.creator = object.creator ?? "";
    message.manifestId = object.manifestId ?? "";
    message.attestationId = object.attestationId ?? "";
    return message;
  }
};
function createBaseMsgBindManifestToAttestationResponse() {
  return {};
}
var MsgBindManifestToAttestationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestationResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgBindManifestToAttestationResponse();
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
    const message = createBaseMsgBindManifestToAttestationResponse();
    return message;
  }
};
function createBaseMsgOpenIncident() {
  return {
    authority: "",
    id: "",
    severity: 0,
    title: "",
    description: "",
    reporter: "",
    affectedModules: [],
    slaWindowBlocks: BigInt(0)
  };
}
var MsgOpenIncident = {
  typeUrl: "/zerone.knowledge.v1.MsgOpenIncident",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.id !== "") {
      writer.uint32(18).string(message.id);
    }
    if (message.severity !== 0) {
      writer.uint32(24).int32(message.severity);
    }
    if (message.title !== "") {
      writer.uint32(34).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(42).string(message.description);
    }
    if (message.reporter !== "") {
      writer.uint32(50).string(message.reporter);
    }
    for (const v of message.affectedModules) {
      writer.uint32(58).string(v);
    }
    if (message.slaWindowBlocks !== BigInt(0)) {
      writer.uint32(64).uint64(message.slaWindowBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgOpenIncident();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.id = reader.string();
          break;
        case 3:
          message.severity = reader.int32();
          break;
        case 4:
          message.title = reader.string();
          break;
        case 5:
          message.description = reader.string();
          break;
        case 6:
          message.reporter = reader.string();
          break;
        case 7:
          message.affectedModules.push(reader.string());
          break;
        case 8:
          message.slaWindowBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgOpenIncident();
    message.authority = object.authority ?? "";
    message.id = object.id ?? "";
    message.severity = object.severity ?? 0;
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.reporter = object.reporter ?? "";
    message.affectedModules = object.affectedModules?.map((e) => e) || [];
    message.slaWindowBlocks = object.slaWindowBlocks !== void 0 && object.slaWindowBlocks !== null ? BigInt(object.slaWindowBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgOpenIncidentResponse() {
  return {
    slaTargetBlock: BigInt(0)
  };
}
var MsgOpenIncidentResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgOpenIncidentResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.slaTargetBlock !== BigInt(0)) {
      writer.uint32(8).uint64(message.slaTargetBlock);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgOpenIncidentResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.slaTargetBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgOpenIncidentResponse();
    message.slaTargetBlock = object.slaTargetBlock !== void 0 && object.slaTargetBlock !== null ? BigInt(object.slaTargetBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgRecordRemediation() {
  return {
    authority: "",
    incidentId: "",
    type: 0,
    reference: "",
    note: ""
  };
}
var MsgRecordRemediation = {
  typeUrl: "/zerone.knowledge.v1.MsgRecordRemediation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.incidentId !== "") {
      writer.uint32(18).string(message.incidentId);
    }
    if (message.type !== 0) {
      writer.uint32(24).int32(message.type);
    }
    if (message.reference !== "") {
      writer.uint32(34).string(message.reference);
    }
    if (message.note !== "") {
      writer.uint32(42).string(message.note);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRecordRemediation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.incidentId = reader.string();
          break;
        case 3:
          message.type = reader.int32();
          break;
        case 4:
          message.reference = reader.string();
          break;
        case 5:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRecordRemediation();
    message.authority = object.authority ?? "";
    message.incidentId = object.incidentId ?? "";
    message.type = object.type ?? 0;
    message.reference = object.reference ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgRecordRemediationResponse() {
  return {
    totalRemediations: 0
  };
}
var MsgRecordRemediationResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgRecordRemediationResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.totalRemediations !== 0) {
      writer.uint32(8).uint32(message.totalRemediations);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRecordRemediationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalRemediations = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRecordRemediationResponse();
    message.totalRemediations = object.totalRemediations ?? 0;
    return message;
  }
};
function createBaseMsgResolveIncident() {
  return {
    authority: "",
    incidentId: "",
    postMortemUri: ""
  };
}
var MsgResolveIncident = {
  typeUrl: "/zerone.knowledge.v1.MsgResolveIncident",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.incidentId !== "") {
      writer.uint32(18).string(message.incidentId);
    }
    if (message.postMortemUri !== "") {
      writer.uint32(26).string(message.postMortemUri);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveIncident();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.incidentId = reader.string();
          break;
        case 3:
          message.postMortemUri = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgResolveIncident();
    message.authority = object.authority ?? "";
    message.incidentId = object.incidentId ?? "";
    message.postMortemUri = object.postMortemUri ?? "";
    return message;
  }
};
function createBaseMsgResolveIncidentResponse() {
  return {};
}
var MsgResolveIncidentResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgResolveIncidentResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgResolveIncidentResponse();
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
    const message = createBaseMsgResolveIncidentResponse();
    return message;
  }
};
function createBaseMsgCloseIncident() {
  return {
    authority: "",
    incidentId: ""
  };
}
var MsgCloseIncident = {
  typeUrl: "/zerone.knowledge.v1.MsgCloseIncident",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.incidentId !== "") {
      writer.uint32(18).string(message.incidentId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCloseIncident();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.incidentId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCloseIncident();
    message.authority = object.authority ?? "";
    message.incidentId = object.incidentId ?? "";
    return message;
  }
};
function createBaseMsgCloseIncidentResponse() {
  return {};
}
var MsgCloseIncidentResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgCloseIncidentResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCloseIncidentResponse();
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
    const message = createBaseMsgCloseIncidentResponse();
    return message;
  }
};
function createBaseMsgPauseModule() {
  return {
    authority: "",
    moduleName: "",
    reason: "",
    autoUnpauseAtBlock: BigInt(0),
    incidentId: ""
  };
}
var MsgPauseModule = {
  typeUrl: "/zerone.knowledge.v1.MsgPauseModule",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.moduleName !== "") {
      writer.uint32(18).string(message.moduleName);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    if (message.autoUnpauseAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.autoUnpauseAtBlock);
    }
    if (message.incidentId !== "") {
      writer.uint32(42).string(message.incidentId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseModule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.moduleName = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        case 4:
          message.autoUnpauseAtBlock = reader.uint64();
          break;
        case 5:
          message.incidentId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgPauseModule();
    message.authority = object.authority ?? "";
    message.moduleName = object.moduleName ?? "";
    message.reason = object.reason ?? "";
    message.autoUnpauseAtBlock = object.autoUnpauseAtBlock !== void 0 && object.autoUnpauseAtBlock !== null ? BigInt(object.autoUnpauseAtBlock.toString()) : BigInt(0);
    message.incidentId = object.incidentId ?? "";
    return message;
  }
};
function createBaseMsgPauseModuleResponse() {
  return {
    pausedAtBlock: BigInt(0)
  };
}
var MsgPauseModuleResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgPauseModuleResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.pausedAtBlock !== BigInt(0)) {
      writer.uint32(8).uint64(message.pausedAtBlock);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseModuleResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.pausedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgPauseModuleResponse();
    message.pausedAtBlock = object.pausedAtBlock !== void 0 && object.pausedAtBlock !== null ? BigInt(object.pausedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgUnpauseModule() {
  return {
    authority: "",
    moduleName: "",
    note: ""
  };
}
var MsgUnpauseModule = {
  typeUrl: "/zerone.knowledge.v1.MsgUnpauseModule",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.moduleName !== "") {
      writer.uint32(18).string(message.moduleName);
    }
    if (message.note !== "") {
      writer.uint32(26).string(message.note);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUnpauseModule();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.moduleName = reader.string();
          break;
        case 3:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUnpauseModule();
    message.authority = object.authority ?? "";
    message.moduleName = object.moduleName ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgUnpauseModuleResponse() {
  return {};
}
var MsgUnpauseModuleResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgUnpauseModuleResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUnpauseModuleResponse();
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
    const message = createBaseMsgUnpauseModuleResponse();
    return message;
  }
};
function createBaseMsgCorrectManifestMerkleRoot() {
  return {
    authority: "",
    manifestId: "",
    incidentId: "",
    expectedRecomputedRoot: "",
    note: ""
  };
}
var MsgCorrectManifestMerkleRoot = {
  typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.manifestId !== "") {
      writer.uint32(18).string(message.manifestId);
    }
    if (message.incidentId !== "") {
      writer.uint32(26).string(message.incidentId);
    }
    if (message.expectedRecomputedRoot !== "") {
      writer.uint32(34).string(message.expectedRecomputedRoot);
    }
    if (message.note !== "") {
      writer.uint32(42).string(message.note);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCorrectManifestMerkleRoot();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.manifestId = reader.string();
          break;
        case 3:
          message.incidentId = reader.string();
          break;
        case 4:
          message.expectedRecomputedRoot = reader.string();
          break;
        case 5:
          message.note = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCorrectManifestMerkleRoot();
    message.authority = object.authority ?? "";
    message.manifestId = object.manifestId ?? "";
    message.incidentId = object.incidentId ?? "";
    message.expectedRecomputedRoot = object.expectedRecomputedRoot ?? "";
    message.note = object.note ?? "";
    return message;
  }
};
function createBaseMsgCorrectManifestMerkleRootResponse() {
  return {
    priorRoot: "",
    recomputedRoot: "",
    wasCorrupted: false
  };
}
var MsgCorrectManifestMerkleRootResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRootResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.priorRoot !== "") {
      writer.uint32(10).string(message.priorRoot);
    }
    if (message.recomputedRoot !== "") {
      writer.uint32(18).string(message.recomputedRoot);
    }
    if (message.wasCorrupted === true) {
      writer.uint32(24).bool(message.wasCorrupted);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCorrectManifestMerkleRootResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.priorRoot = reader.string();
          break;
        case 2:
          message.recomputedRoot = reader.string();
          break;
        case 3:
          message.wasCorrupted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCorrectManifestMerkleRootResponse();
    message.priorRoot = object.priorRoot ?? "";
    message.recomputedRoot = object.recomputedRoot ?? "";
    message.wasCorrupted = object.wasCorrupted ?? false;
    return message;
  }
};
function createBaseMsgVetoFactInjection() {
  return {
    guardian: "",
    pendingId: "",
    reason: ""
  };
}
var MsgVetoFactInjection = {
  typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjection",
  encode(message, writer = BinaryWriter.create()) {
    if (message.guardian !== "") {
      writer.uint32(10).string(message.guardian);
    }
    if (message.pendingId !== "") {
      writer.uint32(18).string(message.pendingId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVetoFactInjection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.guardian = reader.string();
          break;
        case 2:
          message.pendingId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVetoFactInjection();
    message.guardian = object.guardian ?? "";
    message.pendingId = object.pendingId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgVetoFactInjectionResponse() {
  return {};
}
var MsgVetoFactInjectionResponse = {
  typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjectionResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVetoFactInjectionResponse();
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
    const message = createBaseMsgVetoFactInjectionResponse();
    return message;
  }
};

// src/generated/zerone/knowledge/v1/tx.registry.ts
var registry12 = [["/zerone.knowledge.v1.MsgSubmitClaim", MsgSubmitClaim], ["/zerone.knowledge.v1.MsgSubmitCommitment", MsgSubmitCommitment], ["/zerone.knowledge.v1.MsgSubmitReveal", MsgSubmitReveal], ["/zerone.knowledge.v1.MsgChallengeFact", MsgChallengeFact], ["/zerone.knowledge.v1.MsgAddFact", MsgAddFact], ["/zerone.knowledge.v1.MsgSubmitContradiction", MsgSubmitContradiction], ["/zerone.knowledge.v1.MsgPatronizeFact", MsgPatronizeFact], ["/zerone.knowledge.v1.MsgProposeDomain", MsgProposeDomain], ["/zerone.knowledge.v1.MsgEndorseDomainProposal", MsgEndorseDomainProposal], ["/zerone.knowledge.v1.MsgChallengeDomainProposal", MsgChallengeDomainProposal], ["/zerone.knowledge.v1.MsgRegisterStratum", MsgRegisterStratum], ["/zerone.knowledge.v1.MsgChallengeProvisionalFact", MsgChallengeProvisionalFact], ["/zerone.knowledge.v1.MsgUpdateParams", MsgUpdateParams11], ["/zerone.knowledge.v1.MsgUpdateExtendedParams", MsgUpdateExtendedParams], ["/zerone.knowledge.v1.MsgProposeResearchFund", MsgProposeResearchFund], ["/zerone.knowledge.v1.MsgVoteResearchProposal", MsgVoteResearchProposal], ["/zerone.knowledge.v1.MsgExecuteResearchProposal", MsgExecuteResearchProposal], ["/zerone.knowledge.v1.MsgAddCommonKnowledge", MsgAddCommonKnowledge], ["/zerone.knowledge.v1.MsgRemoveCommonKnowledge", MsgRemoveCommonKnowledge], ["/zerone.knowledge.v1.MsgReportDemand", MsgReportDemand], ["/zerone.knowledge.v1.MsgRateFact", MsgRateFact], ["/zerone.knowledge.v1.MsgRegisterTrainingPipeline", MsgRegisterTrainingPipeline], ["/zerone.knowledge.v1.MsgUpdateTrainingPipeline", MsgUpdateTrainingPipeline], ["/zerone.knowledge.v1.MsgRegisterModelCard", MsgRegisterModelCard], ["/zerone.knowledge.v1.MsgUpdateModelCard", MsgUpdateModelCard], ["/zerone.knowledge.v1.MsgRetireModelCard", MsgRetireModelCard], ["/zerone.knowledge.v1.MsgAmendTokenizerSpec", MsgAmendTokenizerSpec], ["/zerone.knowledge.v1.MsgAttributeContributions", MsgAttributeContributions], ["/zerone.knowledge.v1.MsgAttestTraining", MsgAttestTraining], ["/zerone.knowledge.v1.MsgCreateAugmentationBounty", MsgCreateAugmentationBounty], ["/zerone.knowledge.v1.MsgSubmitAugmentation", MsgSubmitAugmentation], ["/zerone.knowledge.v1.MsgAcceptAugmentation", MsgAcceptAugmentation], ["/zerone.knowledge.v1.MsgVoteOnAugmentation", MsgVoteOnAugmentation], ["/zerone.knowledge.v1.MsgSponsorVetoAugmentation", MsgSponsorVetoAugmentation], ["/zerone.knowledge.v1.MsgChallengeContribution", MsgChallengeContribution], ["/zerone.knowledge.v1.MsgResolveContributionChallenge", MsgResolveContributionChallenge], ["/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement", MsgClaimTrainingFundDisbursement], ["/zerone.knowledge.v1.MsgAmendTraceSchema", MsgAmendTraceSchema], ["/zerone.knowledge.v1.MsgCreateTrainingManifest", MsgCreateTrainingManifest], ["/zerone.knowledge.v1.MsgFinalizeTrainingManifest", MsgFinalizeTrainingManifest], ["/zerone.knowledge.v1.MsgBindManifestToAttestation", MsgBindManifestToAttestation], ["/zerone.knowledge.v1.MsgOpenIncident", MsgOpenIncident], ["/zerone.knowledge.v1.MsgRecordRemediation", MsgRecordRemediation], ["/zerone.knowledge.v1.MsgResolveIncident", MsgResolveIncident], ["/zerone.knowledge.v1.MsgCloseIncident", MsgCloseIncident], ["/zerone.knowledge.v1.MsgPauseModule", MsgPauseModule], ["/zerone.knowledge.v1.MsgUnpauseModule", MsgUnpauseModule], ["/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot", MsgCorrectManifestMerkleRoot], ["/zerone.knowledge.v1.MsgVetoFactInjection", MsgVetoFactInjection]];
var MessageComposer12 = {
  encoded: {
    submitClaim(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitClaim",
        value: MsgSubmitClaim.encode(value).finish()
      };
    },
    submitCommitment(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitment",
        value: MsgSubmitCommitment.encode(value).finish()
      };
    },
    submitReveal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitReveal",
        value: MsgSubmitReveal.encode(value).finish()
      };
    },
    challengeFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeFact",
        value: MsgChallengeFact.encode(value).finish()
      };
    },
    addFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddFact",
        value: MsgAddFact.encode(value).finish()
      };
    },
    submitContradiction(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitContradiction",
        value: MsgSubmitContradiction.encode(value).finish()
      };
    },
    patronizeFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPatronizeFact",
        value: MsgPatronizeFact.encode(value).finish()
      };
    },
    proposeDomain(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeDomain",
        value: MsgProposeDomain.encode(value).finish()
      };
    },
    endorseDomainProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposal",
        value: MsgEndorseDomainProposal.encode(value).finish()
      };
    },
    challengeDomainProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposal",
        value: MsgChallengeDomainProposal.encode(value).finish()
      };
    },
    registerStratum(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterStratum",
        value: MsgRegisterStratum.encode(value).finish()
      };
    },
    challengeProvisionalFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFact",
        value: MsgChallengeProvisionalFact.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateParams",
        value: MsgUpdateParams11.encode(value).finish()
      };
    },
    updateExtendedParams(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParams",
        value: MsgUpdateExtendedParams.encode(value).finish()
      };
    },
    proposeResearchFund(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFund",
        value: MsgProposeResearchFund.encode(value).finish()
      };
    },
    voteResearchProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposal",
        value: MsgVoteResearchProposal.encode(value).finish()
      };
    },
    executeResearchProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposal",
        value: MsgExecuteResearchProposal.encode(value).finish()
      };
    },
    addCommonKnowledge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledge",
        value: MsgAddCommonKnowledge.encode(value).finish()
      };
    },
    removeCommonKnowledge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledge",
        value: MsgRemoveCommonKnowledge.encode(value).finish()
      };
    },
    reportDemand(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgReportDemand",
        value: MsgReportDemand.encode(value).finish()
      };
    },
    rateFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRateFact",
        value: MsgRateFact.encode(value).finish()
      };
    },
    registerTrainingPipeline(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipeline",
        value: MsgRegisterTrainingPipeline.encode(value).finish()
      };
    },
    updateTrainingPipeline(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipeline",
        value: MsgUpdateTrainingPipeline.encode(value).finish()
      };
    },
    registerModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCard",
        value: MsgRegisterModelCard.encode(value).finish()
      };
    },
    updateModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCard",
        value: MsgUpdateModelCard.encode(value).finish()
      };
    },
    retireModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRetireModelCard",
        value: MsgRetireModelCard.encode(value).finish()
      };
    },
    amendTokenizerSpec(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpec",
        value: MsgAmendTokenizerSpec.encode(value).finish()
      };
    },
    attributeContributions(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttributeContributions",
        value: MsgAttributeContributions.encode(value).finish()
      };
    },
    attestTraining(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttestTraining",
        value: MsgAttestTraining.encode(value).finish()
      };
    },
    createAugmentationBounty(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBounty",
        value: MsgCreateAugmentationBounty.encode(value).finish()
      };
    },
    submitAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentation",
        value: MsgSubmitAugmentation.encode(value).finish()
      };
    },
    acceptAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentation",
        value: MsgAcceptAugmentation.encode(value).finish()
      };
    },
    voteOnAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentation",
        value: MsgVoteOnAugmentation.encode(value).finish()
      };
    },
    sponsorVetoAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentation",
        value: MsgSponsorVetoAugmentation.encode(value).finish()
      };
    },
    challengeContribution(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeContribution",
        value: MsgChallengeContribution.encode(value).finish()
      };
    },
    resolveContributionChallenge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallenge",
        value: MsgResolveContributionChallenge.encode(value).finish()
      };
    },
    claimTrainingFundDisbursement(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement",
        value: MsgClaimTrainingFundDisbursement.encode(value).finish()
      };
    },
    amendTraceSchema(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchema",
        value: MsgAmendTraceSchema.encode(value).finish()
      };
    },
    createTrainingManifest(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifest",
        value: MsgCreateTrainingManifest.encode(value).finish()
      };
    },
    finalizeTrainingManifest(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifest",
        value: MsgFinalizeTrainingManifest.encode(value).finish()
      };
    },
    bindManifestToAttestation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestation",
        value: MsgBindManifestToAttestation.encode(value).finish()
      };
    },
    openIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgOpenIncident",
        value: MsgOpenIncident.encode(value).finish()
      };
    },
    recordRemediation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRecordRemediation",
        value: MsgRecordRemediation.encode(value).finish()
      };
    },
    resolveIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveIncident",
        value: MsgResolveIncident.encode(value).finish()
      };
    },
    closeIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCloseIncident",
        value: MsgCloseIncident.encode(value).finish()
      };
    },
    pauseModule(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPauseModule",
        value: MsgPauseModule.encode(value).finish()
      };
    },
    unpauseModule(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUnpauseModule",
        value: MsgUnpauseModule.encode(value).finish()
      };
    },
    correctManifestMerkleRoot(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot",
        value: MsgCorrectManifestMerkleRoot.encode(value).finish()
      };
    },
    vetoFactInjection(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjection",
        value: MsgVetoFactInjection.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    submitClaim(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitClaim",
        value
      };
    },
    submitCommitment(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitment",
        value
      };
    },
    submitReveal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitReveal",
        value
      };
    },
    challengeFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeFact",
        value
      };
    },
    addFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddFact",
        value
      };
    },
    submitContradiction(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitContradiction",
        value
      };
    },
    patronizeFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPatronizeFact",
        value
      };
    },
    proposeDomain(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeDomain",
        value
      };
    },
    endorseDomainProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposal",
        value
      };
    },
    challengeDomainProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposal",
        value
      };
    },
    registerStratum(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterStratum",
        value
      };
    },
    challengeProvisionalFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFact",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateParams",
        value
      };
    },
    updateExtendedParams(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParams",
        value
      };
    },
    proposeResearchFund(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFund",
        value
      };
    },
    voteResearchProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposal",
        value
      };
    },
    executeResearchProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposal",
        value
      };
    },
    addCommonKnowledge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledge",
        value
      };
    },
    removeCommonKnowledge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledge",
        value
      };
    },
    reportDemand(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgReportDemand",
        value
      };
    },
    rateFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRateFact",
        value
      };
    },
    registerTrainingPipeline(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipeline",
        value
      };
    },
    updateTrainingPipeline(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipeline",
        value
      };
    },
    registerModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCard",
        value
      };
    },
    updateModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCard",
        value
      };
    },
    retireModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRetireModelCard",
        value
      };
    },
    amendTokenizerSpec(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpec",
        value
      };
    },
    attributeContributions(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttributeContributions",
        value
      };
    },
    attestTraining(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttestTraining",
        value
      };
    },
    createAugmentationBounty(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBounty",
        value
      };
    },
    submitAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentation",
        value
      };
    },
    acceptAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentation",
        value
      };
    },
    voteOnAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentation",
        value
      };
    },
    sponsorVetoAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentation",
        value
      };
    },
    challengeContribution(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeContribution",
        value
      };
    },
    resolveContributionChallenge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallenge",
        value
      };
    },
    claimTrainingFundDisbursement(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement",
        value
      };
    },
    amendTraceSchema(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchema",
        value
      };
    },
    createTrainingManifest(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifest",
        value
      };
    },
    finalizeTrainingManifest(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifest",
        value
      };
    },
    bindManifestToAttestation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestation",
        value
      };
    },
    openIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgOpenIncident",
        value
      };
    },
    recordRemediation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRecordRemediation",
        value
      };
    },
    resolveIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveIncident",
        value
      };
    },
    closeIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCloseIncident",
        value
      };
    },
    pauseModule(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPauseModule",
        value
      };
    },
    unpauseModule(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUnpauseModule",
        value
      };
    },
    correctManifestMerkleRoot(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot",
        value
      };
    },
    vetoFactInjection(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjection",
        value
      };
    }
  },
  fromPartial: {
    submitClaim(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitClaim",
        value: MsgSubmitClaim.fromPartial(value)
      };
    },
    submitCommitment(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitCommitment",
        value: MsgSubmitCommitment.fromPartial(value)
      };
    },
    submitReveal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitReveal",
        value: MsgSubmitReveal.fromPartial(value)
      };
    },
    challengeFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeFact",
        value: MsgChallengeFact.fromPartial(value)
      };
    },
    addFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddFact",
        value: MsgAddFact.fromPartial(value)
      };
    },
    submitContradiction(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitContradiction",
        value: MsgSubmitContradiction.fromPartial(value)
      };
    },
    patronizeFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPatronizeFact",
        value: MsgPatronizeFact.fromPartial(value)
      };
    },
    proposeDomain(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeDomain",
        value: MsgProposeDomain.fromPartial(value)
      };
    },
    endorseDomainProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgEndorseDomainProposal",
        value: MsgEndorseDomainProposal.fromPartial(value)
      };
    },
    challengeDomainProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeDomainProposal",
        value: MsgChallengeDomainProposal.fromPartial(value)
      };
    },
    registerStratum(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterStratum",
        value: MsgRegisterStratum.fromPartial(value)
      };
    },
    challengeProvisionalFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeProvisionalFact",
        value: MsgChallengeProvisionalFact.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateParams",
        value: MsgUpdateParams11.fromPartial(value)
      };
    },
    updateExtendedParams(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateExtendedParams",
        value: MsgUpdateExtendedParams.fromPartial(value)
      };
    },
    proposeResearchFund(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgProposeResearchFund",
        value: MsgProposeResearchFund.fromPartial(value)
      };
    },
    voteResearchProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteResearchProposal",
        value: MsgVoteResearchProposal.fromPartial(value)
      };
    },
    executeResearchProposal(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgExecuteResearchProposal",
        value: MsgExecuteResearchProposal.fromPartial(value)
      };
    },
    addCommonKnowledge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAddCommonKnowledge",
        value: MsgAddCommonKnowledge.fromPartial(value)
      };
    },
    removeCommonKnowledge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRemoveCommonKnowledge",
        value: MsgRemoveCommonKnowledge.fromPartial(value)
      };
    },
    reportDemand(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgReportDemand",
        value: MsgReportDemand.fromPartial(value)
      };
    },
    rateFact(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRateFact",
        value: MsgRateFact.fromPartial(value)
      };
    },
    registerTrainingPipeline(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterTrainingPipeline",
        value: MsgRegisterTrainingPipeline.fromPartial(value)
      };
    },
    updateTrainingPipeline(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateTrainingPipeline",
        value: MsgUpdateTrainingPipeline.fromPartial(value)
      };
    },
    registerModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRegisterModelCard",
        value: MsgRegisterModelCard.fromPartial(value)
      };
    },
    updateModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUpdateModelCard",
        value: MsgUpdateModelCard.fromPartial(value)
      };
    },
    retireModelCard(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRetireModelCard",
        value: MsgRetireModelCard.fromPartial(value)
      };
    },
    amendTokenizerSpec(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTokenizerSpec",
        value: MsgAmendTokenizerSpec.fromPartial(value)
      };
    },
    attributeContributions(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttributeContributions",
        value: MsgAttributeContributions.fromPartial(value)
      };
    },
    attestTraining(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAttestTraining",
        value: MsgAttestTraining.fromPartial(value)
      };
    },
    createAugmentationBounty(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateAugmentationBounty",
        value: MsgCreateAugmentationBounty.fromPartial(value)
      };
    },
    submitAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSubmitAugmentation",
        value: MsgSubmitAugmentation.fromPartial(value)
      };
    },
    acceptAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAcceptAugmentation",
        value: MsgAcceptAugmentation.fromPartial(value)
      };
    },
    voteOnAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVoteOnAugmentation",
        value: MsgVoteOnAugmentation.fromPartial(value)
      };
    },
    sponsorVetoAugmentation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgSponsorVetoAugmentation",
        value: MsgSponsorVetoAugmentation.fromPartial(value)
      };
    },
    challengeContribution(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgChallengeContribution",
        value: MsgChallengeContribution.fromPartial(value)
      };
    },
    resolveContributionChallenge(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveContributionChallenge",
        value: MsgResolveContributionChallenge.fromPartial(value)
      };
    },
    claimTrainingFundDisbursement(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgClaimTrainingFundDisbursement",
        value: MsgClaimTrainingFundDisbursement.fromPartial(value)
      };
    },
    amendTraceSchema(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgAmendTraceSchema",
        value: MsgAmendTraceSchema.fromPartial(value)
      };
    },
    createTrainingManifest(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCreateTrainingManifest",
        value: MsgCreateTrainingManifest.fromPartial(value)
      };
    },
    finalizeTrainingManifest(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgFinalizeTrainingManifest",
        value: MsgFinalizeTrainingManifest.fromPartial(value)
      };
    },
    bindManifestToAttestation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgBindManifestToAttestation",
        value: MsgBindManifestToAttestation.fromPartial(value)
      };
    },
    openIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgOpenIncident",
        value: MsgOpenIncident.fromPartial(value)
      };
    },
    recordRemediation(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgRecordRemediation",
        value: MsgRecordRemediation.fromPartial(value)
      };
    },
    resolveIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgResolveIncident",
        value: MsgResolveIncident.fromPartial(value)
      };
    },
    closeIncident(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCloseIncident",
        value: MsgCloseIncident.fromPartial(value)
      };
    },
    pauseModule(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgPauseModule",
        value: MsgPauseModule.fromPartial(value)
      };
    },
    unpauseModule(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgUnpauseModule",
        value: MsgUnpauseModule.fromPartial(value)
      };
    },
    correctManifestMerkleRoot(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgCorrectManifestMerkleRoot",
        value: MsgCorrectManifestMerkleRoot.fromPartial(value)
      };
    },
    vetoFactInjection(value) {
      return {
        typeUrl: "/zerone.knowledge.v1.MsgVetoFactInjection",
        value: MsgVetoFactInjection.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/liquiditypool/v1/tx.ts
var tx_exports13 = {};
__export(tx_exports13, {
  MsgAddLiquidity: () => MsgAddLiquidity,
  MsgAddLiquidityResponse: () => MsgAddLiquidityResponse,
  MsgCreatePool: () => MsgCreatePool,
  MsgCreatePoolResponse: () => MsgCreatePoolResponse,
  MsgRemoveLiquidity: () => MsgRemoveLiquidity,
  MsgRemoveLiquidityResponse: () => MsgRemoveLiquidityResponse,
  MsgSwap: () => MsgSwap,
  MsgSwapResponse: () => MsgSwapResponse,
  MsgUpdateParams: () => MsgUpdateParams12,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse12
});

// src/generated/zerone/liquiditypool/v1/genesis.ts
function createBaseParams13() {
  return {
    defaultSwapFeeBps: BigInt(0),
    maxPools: BigInt(0),
    minInitialLiquidity: "",
    twapWindowBlocks: BigInt(0),
    protocolFeeBps: BigInt(0),
    minReserve: "",
    billingQuoteDenoms: []
  };
}
var Params13 = {
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
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams13();
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
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams13();
    message.defaultSwapFeeBps = object.defaultSwapFeeBps !== void 0 && object.defaultSwapFeeBps !== null ? BigInt(object.defaultSwapFeeBps.toString()) : BigInt(0);
    message.maxPools = object.maxPools !== void 0 && object.maxPools !== null ? BigInt(object.maxPools.toString()) : BigInt(0);
    message.minInitialLiquidity = object.minInitialLiquidity ?? "";
    message.twapWindowBlocks = object.twapWindowBlocks !== void 0 && object.twapWindowBlocks !== null ? BigInt(object.twapWindowBlocks.toString()) : BigInt(0);
    message.protocolFeeBps = object.protocolFeeBps !== void 0 && object.protocolFeeBps !== null ? BigInt(object.protocolFeeBps.toString()) : BigInt(0);
    message.minReserve = object.minReserve ?? "";
    message.billingQuoteDenoms = object.billingQuoteDenoms?.map((e) => e) || [];
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
function createBaseMsgUpdateParams12() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams12 = {
  typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params13.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams12();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params13.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams12();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params13.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse12() {
  return {};
}
var MsgUpdateParamsResponse12 = {
  typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse12();
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
    const message = createBaseMsgUpdateParamsResponse12();
    return message;
  }
};

// src/generated/zerone/liquiditypool/v1/tx.registry.ts
var registry13 = [["/zerone.liquiditypool.v1.MsgCreatePool", MsgCreatePool], ["/zerone.liquiditypool.v1.MsgSwap", MsgSwap], ["/zerone.liquiditypool.v1.MsgAddLiquidity", MsgAddLiquidity], ["/zerone.liquiditypool.v1.MsgRemoveLiquidity", MsgRemoveLiquidity], ["/zerone.liquiditypool.v1.MsgUpdateParams", MsgUpdateParams12]];
var MessageComposer13 = {
  encoded: {
    createPool(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgCreatePool",
        value: MsgCreatePool.encode(value).finish()
      };
    },
    swap(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgSwap",
        value: MsgSwap.encode(value).finish()
      };
    },
    addLiquidity(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidity",
        value: MsgAddLiquidity.encode(value).finish()
      };
    },
    removeLiquidity(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidity",
        value: MsgRemoveLiquidity.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
        value: MsgUpdateParams12.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createPool(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgCreatePool",
        value
      };
    },
    swap(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgSwap",
        value
      };
    },
    addLiquidity(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidity",
        value
      };
    },
    removeLiquidity(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidity",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createPool(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgCreatePool",
        value: MsgCreatePool.fromPartial(value)
      };
    },
    swap(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgSwap",
        value: MsgSwap.fromPartial(value)
      };
    },
    addLiquidity(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgAddLiquidity",
        value: MsgAddLiquidity.fromPartial(value)
      };
    },
    removeLiquidity(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgRemoveLiquidity",
        value: MsgRemoveLiquidity.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.liquiditypool.v1.MsgUpdateParams",
        value: MsgUpdateParams12.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/ontology/v1/tx.ts
var tx_exports14 = {};
__export(tx_exports14, {
  MsgAcknowledgeIncompleteness: () => MsgAcknowledgeIncompleteness,
  MsgAcknowledgeIncompletenessResponse: () => MsgAcknowledgeIncompletenessResponse,
  MsgProposeDomain: () => MsgProposeDomain2,
  MsgProposeDomainResponse: () => MsgProposeDomainResponse2,
  MsgRegisterLogicZone: () => MsgRegisterLogicZone,
  MsgRegisterLogicZoneResponse: () => MsgRegisterLogicZoneResponse,
  MsgUpdateDomain: () => MsgUpdateDomain,
  MsgUpdateDomainResponse: () => MsgUpdateDomainResponse,
  MsgUpdateParams: () => MsgUpdateParams13,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse13,
  MsgVoteDomainProposal: () => MsgVoteDomainProposal,
  MsgVoteDomainProposalResponse: () => MsgVoteDomainProposalResponse
});

// src/generated/zerone/ontology/v1/state.ts
function createBaseLogicZoneProperties() {
  return {
    zone: "",
    complete: false,
    decidable: false,
    goedelApplies: false,
    maxConfidenceBps: BigInt(0),
    description: ""
  };
}
var LogicZoneProperties = {
  typeUrl: "/zerone.ontology.v1.LogicZoneProperties",
  encode(message, writer = BinaryWriter.create()) {
    if (message.zone !== "") {
      writer.uint32(10).string(message.zone);
    }
    if (message.complete === true) {
      writer.uint32(16).bool(message.complete);
    }
    if (message.decidable === true) {
      writer.uint32(24).bool(message.decidable);
    }
    if (message.goedelApplies === true) {
      writer.uint32(32).bool(message.goedelApplies);
    }
    if (message.maxConfidenceBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.maxConfidenceBps);
    }
    if (message.description !== "") {
      writer.uint32(50).string(message.description);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseLogicZoneProperties();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.zone = reader.string();
          break;
        case 2:
          message.complete = reader.bool();
          break;
        case 3:
          message.decidable = reader.bool();
          break;
        case 4:
          message.goedelApplies = reader.bool();
          break;
        case 5:
          message.maxConfidenceBps = reader.uint64();
          break;
        case 6:
          message.description = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseLogicZoneProperties();
    message.zone = object.zone ?? "";
    message.complete = object.complete ?? false;
    message.decidable = object.decidable ?? false;
    message.goedelApplies = object.goedelApplies ?? false;
    message.maxConfidenceBps = object.maxConfidenceBps !== void 0 && object.maxConfidenceBps !== null ? BigInt(object.maxConfidenceBps.toString()) : BigInt(0);
    message.description = object.description ?? "";
    return message;
  }
};

// src/generated/zerone/ontology/v1/genesis.ts
function createBaseParams14() {
  return {
    minProposalStake: "",
    proposalVotingPeriod: BigInt(0),
    minEndorsements: 0,
    crossStratumDiscount: BigInt(0),
    maxDomainsPerStratum: 0,
    allowNewStrata: false
  };
}
var Params14 = {
  typeUrl: "/zerone.ontology.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.minProposalStake !== "") {
      writer.uint32(10).string(message.minProposalStake);
    }
    if (message.proposalVotingPeriod !== BigInt(0)) {
      writer.uint32(16).uint64(message.proposalVotingPeriod);
    }
    if (message.minEndorsements !== 0) {
      writer.uint32(24).uint32(message.minEndorsements);
    }
    if (message.crossStratumDiscount !== BigInt(0)) {
      writer.uint32(32).uint64(message.crossStratumDiscount);
    }
    if (message.maxDomainsPerStratum !== 0) {
      writer.uint32(40).uint32(message.maxDomainsPerStratum);
    }
    if (message.allowNewStrata === true) {
      writer.uint32(48).bool(message.allowNewStrata);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams14();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minProposalStake = reader.string();
          break;
        case 2:
          message.proposalVotingPeriod = reader.uint64();
          break;
        case 3:
          message.minEndorsements = reader.uint32();
          break;
        case 4:
          message.crossStratumDiscount = reader.uint64();
          break;
        case 5:
          message.maxDomainsPerStratum = reader.uint32();
          break;
        case 6:
          message.allowNewStrata = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams14();
    message.minProposalStake = object.minProposalStake ?? "";
    message.proposalVotingPeriod = object.proposalVotingPeriod !== void 0 && object.proposalVotingPeriod !== null ? BigInt(object.proposalVotingPeriod.toString()) : BigInt(0);
    message.minEndorsements = object.minEndorsements ?? 0;
    message.crossStratumDiscount = object.crossStratumDiscount !== void 0 && object.crossStratumDiscount !== null ? BigInt(object.crossStratumDiscount.toString()) : BigInt(0);
    message.maxDomainsPerStratum = object.maxDomainsPerStratum ?? 0;
    message.allowNewStrata = object.allowNewStrata ?? false;
    return message;
  }
};

// src/generated/zerone/ontology/v1/tx.ts
function createBaseMsgProposeDomain2() {
  return {
    proposer: "",
    name: "",
    displayName: "",
    description: "",
    stratum: 0,
    stake: ""
  };
}
var MsgProposeDomain2 = {
  typeUrl: "/zerone.ontology.v1.MsgProposeDomain",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.displayName !== "") {
      writer.uint32(26).string(message.displayName);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.stratum !== 0) {
      writer.uint32(40).uint32(message.stratum);
    }
    if (message.stake !== "") {
      writer.uint32(50).string(message.stake);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeDomain2();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.displayName = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.stratum = reader.uint32();
          break;
        case 6:
          message.stake = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeDomain2();
    message.proposer = object.proposer ?? "";
    message.name = object.name ?? "";
    message.displayName = object.displayName ?? "";
    message.description = object.description ?? "";
    message.stratum = object.stratum ?? 0;
    message.stake = object.stake ?? "";
    return message;
  }
};
function createBaseMsgProposeDomainResponse2() {
  return {
    proposalId: ""
  };
}
var MsgProposeDomainResponse2 = {
  typeUrl: "/zerone.ontology.v1.MsgProposeDomainResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeDomainResponse2();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgProposeDomainResponse2();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteDomainProposal() {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
var MsgVoteDomainProposal = {
  typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposal",
  encode(message, writer = BinaryWriter.create()) {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.approve === true) {
      writer.uint32(24).bool(message.approve);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteDomainProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgVoteDomainProposal();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteDomainProposalResponse() {
  return {};
}
var MsgVoteDomainProposalResponse = {
  typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposalResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteDomainProposalResponse();
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
    const message = createBaseMsgVoteDomainProposalResponse();
    return message;
  }
};
function createBaseMsgUpdateDomain() {
  return {
    authority: "",
    domainName: "",
    displayName: "",
    description: "",
    status: ""
  };
}
var MsgUpdateDomain = {
  typeUrl: "/zerone.ontology.v1.MsgUpdateDomain",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.domainName !== "") {
      writer.uint32(18).string(message.domainName);
    }
    if (message.displayName !== "") {
      writer.uint32(26).string(message.displayName);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.status !== "") {
      writer.uint32(42).string(message.status);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.domainName = reader.string();
          break;
        case 3:
          message.displayName = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.status = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateDomain();
    message.authority = object.authority ?? "";
    message.domainName = object.domainName ?? "";
    message.displayName = object.displayName ?? "";
    message.description = object.description ?? "";
    message.status = object.status ?? "";
    return message;
  }
};
function createBaseMsgUpdateDomainResponse() {
  return {};
}
var MsgUpdateDomainResponse = {
  typeUrl: "/zerone.ontology.v1.MsgUpdateDomainResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateDomainResponse();
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
    const message = createBaseMsgUpdateDomainResponse();
    return message;
  }
};
function createBaseMsgRegisterLogicZone() {
  return {
    authority: "",
    zoneProperties: void 0
  };
}
var MsgRegisterLogicZone = {
  typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZone",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.zoneProperties !== void 0) {
      LogicZoneProperties.encode(message.zoneProperties, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterLogicZone();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.zoneProperties = LogicZoneProperties.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterLogicZone();
    message.authority = object.authority ?? "";
    message.zoneProperties = object.zoneProperties !== void 0 && object.zoneProperties !== null ? LogicZoneProperties.fromPartial(object.zoneProperties) : void 0;
    return message;
  }
};
function createBaseMsgRegisterLogicZoneResponse() {
  return {};
}
var MsgRegisterLogicZoneResponse = {
  typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZoneResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterLogicZoneResponse();
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
    const message = createBaseMsgRegisterLogicZoneResponse();
    return message;
  }
};
function createBaseMsgAcknowledgeIncompleteness() {
  return {
    submitter: "",
    factId: "",
    zone: "",
    reason: ""
  };
}
var MsgAcknowledgeIncompleteness = {
  typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompleteness",
  encode(message, writer = BinaryWriter.create()) {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.zone !== "") {
      writer.uint32(26).string(message.zone);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAcknowledgeIncompleteness();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.zone = reader.string();
          break;
        case 4:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAcknowledgeIncompleteness();
    message.submitter = object.submitter ?? "";
    message.factId = object.factId ?? "";
    message.zone = object.zone ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgAcknowledgeIncompletenessResponse() {
  return {};
}
var MsgAcknowledgeIncompletenessResponse = {
  typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompletenessResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAcknowledgeIncompletenessResponse();
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
    const message = createBaseMsgAcknowledgeIncompletenessResponse();
    return message;
  }
};
function createBaseMsgUpdateParams13() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams13 = {
  typeUrl: "/zerone.ontology.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params14.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams13();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params14.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams13();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params14.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse13() {
  return {};
}
var MsgUpdateParamsResponse13 = {
  typeUrl: "/zerone.ontology.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse13();
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
    const message = createBaseMsgUpdateParamsResponse13();
    return message;
  }
};

// src/generated/zerone/ontology/v1/tx.registry.ts
var registry14 = [["/zerone.ontology.v1.MsgProposeDomain", MsgProposeDomain2], ["/zerone.ontology.v1.MsgVoteDomainProposal", MsgVoteDomainProposal], ["/zerone.ontology.v1.MsgUpdateDomain", MsgUpdateDomain], ["/zerone.ontology.v1.MsgRegisterLogicZone", MsgRegisterLogicZone], ["/zerone.ontology.v1.MsgAcknowledgeIncompleteness", MsgAcknowledgeIncompleteness], ["/zerone.ontology.v1.MsgUpdateParams", MsgUpdateParams13]];
var MessageComposer14 = {
  encoded: {
    proposeDomain(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgProposeDomain",
        value: MsgProposeDomain2.encode(value).finish()
      };
    },
    voteDomainProposal(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposal",
        value: MsgVoteDomainProposal.encode(value).finish()
      };
    },
    updateDomain(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateDomain",
        value: MsgUpdateDomain.encode(value).finish()
      };
    },
    registerLogicZone(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZone",
        value: MsgRegisterLogicZone.encode(value).finish()
      };
    },
    acknowledgeIncompleteness(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompleteness",
        value: MsgAcknowledgeIncompleteness.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateParams",
        value: MsgUpdateParams13.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    proposeDomain(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgProposeDomain",
        value
      };
    },
    voteDomainProposal(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposal",
        value
      };
    },
    updateDomain(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateDomain",
        value
      };
    },
    registerLogicZone(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZone",
        value
      };
    },
    acknowledgeIncompleteness(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompleteness",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    proposeDomain(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgProposeDomain",
        value: MsgProposeDomain2.fromPartial(value)
      };
    },
    voteDomainProposal(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposal",
        value: MsgVoteDomainProposal.fromPartial(value)
      };
    },
    updateDomain(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateDomain",
        value: MsgUpdateDomain.fromPartial(value)
      };
    },
    registerLogicZone(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZone",
        value: MsgRegisterLogicZone.fromPartial(value)
      };
    },
    acknowledgeIncompleteness(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompleteness",
        value: MsgAcknowledgeIncompleteness.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.ontology.v1.MsgUpdateParams",
        value: MsgUpdateParams13.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/qualification/v1/tx.ts
var tx_exports15 = {};
__export(tx_exports15, {
  MsgEndorseQualification: () => MsgEndorseQualification,
  MsgEndorseQualificationResponse: () => MsgEndorseQualificationResponse,
  MsgQualifyByCrossReference: () => MsgQualifyByCrossReference,
  MsgQualifyByCrossReferenceResponse: () => MsgQualifyByCrossReferenceResponse,
  MsgQualifyByInheritance: () => MsgQualifyByInheritance,
  MsgQualifyByInheritanceResponse: () => MsgQualifyByInheritanceResponse,
  MsgQualifyByStake: () => MsgQualifyByStake,
  MsgQualifyByStakeResponse: () => MsgQualifyByStakeResponse,
  MsgQualifyByTrackRecord: () => MsgQualifyByTrackRecord,
  MsgQualifyByTrackRecordResponse: () => MsgQualifyByTrackRecordResponse,
  MsgRenewQualification: () => MsgRenewQualification,
  MsgRenewQualificationResponse: () => MsgRenewQualificationResponse,
  MsgUpdateParams: () => MsgUpdateParams14,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse14,
  MsgWithdrawQualification: () => MsgWithdrawQualification,
  MsgWithdrawQualificationResponse: () => MsgWithdrawQualificationResponse
});

// src/generated/zerone/qualification/v1/genesis.ts
function createBaseParams15() {
  return {
    minStakeAmount: "",
    stakeLockPeriod: BigInt(0),
    minVerifications: BigInt(0),
    minAccuracyBps: BigInt(0),
    minReputationScore: BigInt(0),
    qualificationPeriod: BigInt(0),
    probationPeriod: BigInt(0),
    renewalWindow: BigInt(0),
    maxEndorsements: 0,
    crossRefMinWeight: BigInt(0),
    crossRefWeightDiscountBps: BigInt(0),
    inheritanceWeightDiscountBps: BigInt(0),
    endorsementMaxOverlapBps: BigInt(0),
    decayCheckIntervalBlocks: BigInt(0),
    decayMinSamples: BigInt(0),
    decayProbationBps: BigInt(0),
    decaySuspensionBps: BigInt(0),
    decayRecoveryBps: BigInt(0)
  };
}
var Params15 = {
  typeUrl: "/zerone.qualification.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.minStakeAmount !== "") {
      writer.uint32(10).string(message.minStakeAmount);
    }
    if (message.stakeLockPeriod !== BigInt(0)) {
      writer.uint32(16).uint64(message.stakeLockPeriod);
    }
    if (message.minVerifications !== BigInt(0)) {
      writer.uint32(24).uint64(message.minVerifications);
    }
    if (message.minAccuracyBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.minAccuracyBps);
    }
    if (message.minReputationScore !== BigInt(0)) {
      writer.uint32(40).uint64(message.minReputationScore);
    }
    if (message.qualificationPeriod !== BigInt(0)) {
      writer.uint32(48).uint64(message.qualificationPeriod);
    }
    if (message.probationPeriod !== BigInt(0)) {
      writer.uint32(56).uint64(message.probationPeriod);
    }
    if (message.renewalWindow !== BigInt(0)) {
      writer.uint32(64).uint64(message.renewalWindow);
    }
    if (message.maxEndorsements !== 0) {
      writer.uint32(72).uint32(message.maxEndorsements);
    }
    if (message.crossRefMinWeight !== BigInt(0)) {
      writer.uint32(80).uint64(message.crossRefMinWeight);
    }
    if (message.crossRefWeightDiscountBps !== BigInt(0)) {
      writer.uint32(88).uint64(message.crossRefWeightDiscountBps);
    }
    if (message.inheritanceWeightDiscountBps !== BigInt(0)) {
      writer.uint32(96).uint64(message.inheritanceWeightDiscountBps);
    }
    if (message.endorsementMaxOverlapBps !== BigInt(0)) {
      writer.uint32(104).uint64(message.endorsementMaxOverlapBps);
    }
    if (message.decayCheckIntervalBlocks !== BigInt(0)) {
      writer.uint32(112).uint64(message.decayCheckIntervalBlocks);
    }
    if (message.decayMinSamples !== BigInt(0)) {
      writer.uint32(120).uint64(message.decayMinSamples);
    }
    if (message.decayProbationBps !== BigInt(0)) {
      writer.uint32(128).uint64(message.decayProbationBps);
    }
    if (message.decaySuspensionBps !== BigInt(0)) {
      writer.uint32(136).uint64(message.decaySuspensionBps);
    }
    if (message.decayRecoveryBps !== BigInt(0)) {
      writer.uint32(144).uint64(message.decayRecoveryBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams15();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.minStakeAmount = reader.string();
          break;
        case 2:
          message.stakeLockPeriod = reader.uint64();
          break;
        case 3:
          message.minVerifications = reader.uint64();
          break;
        case 4:
          message.minAccuracyBps = reader.uint64();
          break;
        case 5:
          message.minReputationScore = reader.uint64();
          break;
        case 6:
          message.qualificationPeriod = reader.uint64();
          break;
        case 7:
          message.probationPeriod = reader.uint64();
          break;
        case 8:
          message.renewalWindow = reader.uint64();
          break;
        case 9:
          message.maxEndorsements = reader.uint32();
          break;
        case 10:
          message.crossRefMinWeight = reader.uint64();
          break;
        case 11:
          message.crossRefWeightDiscountBps = reader.uint64();
          break;
        case 12:
          message.inheritanceWeightDiscountBps = reader.uint64();
          break;
        case 13:
          message.endorsementMaxOverlapBps = reader.uint64();
          break;
        case 14:
          message.decayCheckIntervalBlocks = reader.uint64();
          break;
        case 15:
          message.decayMinSamples = reader.uint64();
          break;
        case 16:
          message.decayProbationBps = reader.uint64();
          break;
        case 17:
          message.decaySuspensionBps = reader.uint64();
          break;
        case 18:
          message.decayRecoveryBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams15();
    message.minStakeAmount = object.minStakeAmount ?? "";
    message.stakeLockPeriod = object.stakeLockPeriod !== void 0 && object.stakeLockPeriod !== null ? BigInt(object.stakeLockPeriod.toString()) : BigInt(0);
    message.minVerifications = object.minVerifications !== void 0 && object.minVerifications !== null ? BigInt(object.minVerifications.toString()) : BigInt(0);
    message.minAccuracyBps = object.minAccuracyBps !== void 0 && object.minAccuracyBps !== null ? BigInt(object.minAccuracyBps.toString()) : BigInt(0);
    message.minReputationScore = object.minReputationScore !== void 0 && object.minReputationScore !== null ? BigInt(object.minReputationScore.toString()) : BigInt(0);
    message.qualificationPeriod = object.qualificationPeriod !== void 0 && object.qualificationPeriod !== null ? BigInt(object.qualificationPeriod.toString()) : BigInt(0);
    message.probationPeriod = object.probationPeriod !== void 0 && object.probationPeriod !== null ? BigInt(object.probationPeriod.toString()) : BigInt(0);
    message.renewalWindow = object.renewalWindow !== void 0 && object.renewalWindow !== null ? BigInt(object.renewalWindow.toString()) : BigInt(0);
    message.maxEndorsements = object.maxEndorsements ?? 0;
    message.crossRefMinWeight = object.crossRefMinWeight !== void 0 && object.crossRefMinWeight !== null ? BigInt(object.crossRefMinWeight.toString()) : BigInt(0);
    message.crossRefWeightDiscountBps = object.crossRefWeightDiscountBps !== void 0 && object.crossRefWeightDiscountBps !== null ? BigInt(object.crossRefWeightDiscountBps.toString()) : BigInt(0);
    message.inheritanceWeightDiscountBps = object.inheritanceWeightDiscountBps !== void 0 && object.inheritanceWeightDiscountBps !== null ? BigInt(object.inheritanceWeightDiscountBps.toString()) : BigInt(0);
    message.endorsementMaxOverlapBps = object.endorsementMaxOverlapBps !== void 0 && object.endorsementMaxOverlapBps !== null ? BigInt(object.endorsementMaxOverlapBps.toString()) : BigInt(0);
    message.decayCheckIntervalBlocks = object.decayCheckIntervalBlocks !== void 0 && object.decayCheckIntervalBlocks !== null ? BigInt(object.decayCheckIntervalBlocks.toString()) : BigInt(0);
    message.decayMinSamples = object.decayMinSamples !== void 0 && object.decayMinSamples !== null ? BigInt(object.decayMinSamples.toString()) : BigInt(0);
    message.decayProbationBps = object.decayProbationBps !== void 0 && object.decayProbationBps !== null ? BigInt(object.decayProbationBps.toString()) : BigInt(0);
    message.decaySuspensionBps = object.decaySuspensionBps !== void 0 && object.decaySuspensionBps !== null ? BigInt(object.decaySuspensionBps.toString()) : BigInt(0);
    message.decayRecoveryBps = object.decayRecoveryBps !== void 0 && object.decayRecoveryBps !== null ? BigInt(object.decayRecoveryBps.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/qualification/v1/tx.ts
function createBaseMsgQualifyByStake() {
  return {
    validator: "",
    domain: "",
    stakeAmount: ""
  };
}
var MsgQualifyByStake = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByStake",
  encode(message, writer = BinaryWriter.create()) {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.stakeAmount !== "") {
      writer.uint32(26).string(message.stakeAmount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByStake();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.stakeAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgQualifyByStake();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    message.stakeAmount = object.stakeAmount ?? "";
    return message;
  }
};
function createBaseMsgQualifyByStakeResponse() {
  return {};
}
var MsgQualifyByStakeResponse = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByStakeResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByStakeResponse();
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
    const message = createBaseMsgQualifyByStakeResponse();
    return message;
  }
};
function createBaseMsgQualifyByTrackRecord() {
  return {
    validator: "",
    domain: ""
  };
}
var MsgQualifyByTrackRecord = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecord",
  encode(message, writer = BinaryWriter.create()) {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByTrackRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgQualifyByTrackRecord();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    return message;
  }
};
function createBaseMsgQualifyByTrackRecordResponse() {
  return {};
}
var MsgQualifyByTrackRecordResponse = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecordResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByTrackRecordResponse();
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
    const message = createBaseMsgQualifyByTrackRecordResponse();
    return message;
  }
};
function createBaseMsgQualifyByCrossReference() {
  return {
    validator: "",
    targetDomain: "",
    sourceDomain: ""
  };
}
var MsgQualifyByCrossReference = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReference",
  encode(message, writer = BinaryWriter.create()) {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.targetDomain !== "") {
      writer.uint32(18).string(message.targetDomain);
    }
    if (message.sourceDomain !== "") {
      writer.uint32(26).string(message.sourceDomain);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByCrossReference();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.targetDomain = reader.string();
          break;
        case 3:
          message.sourceDomain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgQualifyByCrossReference();
    message.validator = object.validator ?? "";
    message.targetDomain = object.targetDomain ?? "";
    message.sourceDomain = object.sourceDomain ?? "";
    return message;
  }
};
function createBaseMsgQualifyByCrossReferenceResponse() {
  return {};
}
var MsgQualifyByCrossReferenceResponse = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReferenceResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByCrossReferenceResponse();
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
    const message = createBaseMsgQualifyByCrossReferenceResponse();
    return message;
  }
};
function createBaseMsgQualifyByInheritance() {
  return {
    validator: "",
    targetDomain: "",
    parentDomain: ""
  };
}
var MsgQualifyByInheritance = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritance",
  encode(message, writer = BinaryWriter.create()) {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.targetDomain !== "") {
      writer.uint32(18).string(message.targetDomain);
    }
    if (message.parentDomain !== "") {
      writer.uint32(26).string(message.parentDomain);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByInheritance();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.targetDomain = reader.string();
          break;
        case 3:
          message.parentDomain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgQualifyByInheritance();
    message.validator = object.validator ?? "";
    message.targetDomain = object.targetDomain ?? "";
    message.parentDomain = object.parentDomain ?? "";
    return message;
  }
};
function createBaseMsgQualifyByInheritanceResponse() {
  return {};
}
var MsgQualifyByInheritanceResponse = {
  typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritanceResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgQualifyByInheritanceResponse();
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
    const message = createBaseMsgQualifyByInheritanceResponse();
    return message;
  }
};
function createBaseMsgEndorseQualification() {
  return {
    endorser: "",
    validator: "",
    domain: "",
    reason: "",
    weight: 0
  };
}
var MsgEndorseQualification = {
  typeUrl: "/zerone.qualification.v1.MsgEndorseQualification",
  encode(message, writer = BinaryWriter.create()) {
    if (message.endorser !== "") {
      writer.uint32(10).string(message.endorser);
    }
    if (message.validator !== "") {
      writer.uint32(18).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    if (message.weight !== 0) {
      writer.uint32(40).uint32(message.weight);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgEndorseQualification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.endorser = reader.string();
          break;
        case 2:
          message.validator = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.reason = reader.string();
          break;
        case 5:
          message.weight = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgEndorseQualification();
    message.endorser = object.endorser ?? "";
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    message.reason = object.reason ?? "";
    message.weight = object.weight ?? 0;
    return message;
  }
};
function createBaseMsgEndorseQualificationResponse() {
  return {
    endorsementId: BigInt(0)
  };
}
var MsgEndorseQualificationResponse = {
  typeUrl: "/zerone.qualification.v1.MsgEndorseQualificationResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.endorsementId !== BigInt(0)) {
      writer.uint32(8).uint64(message.endorsementId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgEndorseQualificationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.endorsementId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgEndorseQualificationResponse();
    message.endorsementId = object.endorsementId !== void 0 && object.endorsementId !== null ? BigInt(object.endorsementId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgRenewQualification() {
  return {
    validator: "",
    domain: ""
  };
}
var MsgRenewQualification = {
  typeUrl: "/zerone.qualification.v1.MsgRenewQualification",
  encode(message, writer = BinaryWriter.create()) {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRenewQualification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRenewQualification();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    return message;
  }
};
function createBaseMsgRenewQualificationResponse() {
  return {};
}
var MsgRenewQualificationResponse = {
  typeUrl: "/zerone.qualification.v1.MsgRenewQualificationResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRenewQualificationResponse();
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
    const message = createBaseMsgRenewQualificationResponse();
    return message;
  }
};
function createBaseMsgWithdrawQualification() {
  return {
    validator: "",
    domain: ""
  };
}
var MsgWithdrawQualification = {
  typeUrl: "/zerone.qualification.v1.MsgWithdrawQualification",
  encode(message, writer = BinaryWriter.create()) {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgWithdrawQualification();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgWithdrawQualification();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    return message;
  }
};
function createBaseMsgWithdrawQualificationResponse() {
  return {};
}
var MsgWithdrawQualificationResponse = {
  typeUrl: "/zerone.qualification.v1.MsgWithdrawQualificationResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgWithdrawQualificationResponse();
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
    const message = createBaseMsgWithdrawQualificationResponse();
    return message;
  }
};
function createBaseMsgUpdateParams14() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams14 = {
  typeUrl: "/zerone.qualification.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params15.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams14();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params15.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams14();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params15.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse14() {
  return {};
}
var MsgUpdateParamsResponse14 = {
  typeUrl: "/zerone.qualification.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse14();
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
    const message = createBaseMsgUpdateParamsResponse14();
    return message;
  }
};

// src/generated/zerone/qualification/v1/tx.registry.ts
var registry15 = [["/zerone.qualification.v1.MsgQualifyByStake", MsgQualifyByStake], ["/zerone.qualification.v1.MsgQualifyByTrackRecord", MsgQualifyByTrackRecord], ["/zerone.qualification.v1.MsgQualifyByCrossReference", MsgQualifyByCrossReference], ["/zerone.qualification.v1.MsgQualifyByInheritance", MsgQualifyByInheritance], ["/zerone.qualification.v1.MsgEndorseQualification", MsgEndorseQualification], ["/zerone.qualification.v1.MsgRenewQualification", MsgRenewQualification], ["/zerone.qualification.v1.MsgWithdrawQualification", MsgWithdrawQualification], ["/zerone.qualification.v1.MsgUpdateParams", MsgUpdateParams14]];
var MessageComposer15 = {
  encoded: {
    qualifyByStake(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByStake",
        value: MsgQualifyByStake.encode(value).finish()
      };
    },
    qualifyByTrackRecord(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecord",
        value: MsgQualifyByTrackRecord.encode(value).finish()
      };
    },
    qualifyByCrossReference(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReference",
        value: MsgQualifyByCrossReference.encode(value).finish()
      };
    },
    qualifyByInheritance(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritance",
        value: MsgQualifyByInheritance.encode(value).finish()
      };
    },
    endorseQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgEndorseQualification",
        value: MsgEndorseQualification.encode(value).finish()
      };
    },
    renewQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgRenewQualification",
        value: MsgRenewQualification.encode(value).finish()
      };
    },
    withdrawQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgWithdrawQualification",
        value: MsgWithdrawQualification.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgUpdateParams",
        value: MsgUpdateParams14.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    qualifyByStake(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByStake",
        value
      };
    },
    qualifyByTrackRecord(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecord",
        value
      };
    },
    qualifyByCrossReference(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReference",
        value
      };
    },
    qualifyByInheritance(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritance",
        value
      };
    },
    endorseQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgEndorseQualification",
        value
      };
    },
    renewQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgRenewQualification",
        value
      };
    },
    withdrawQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgWithdrawQualification",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    qualifyByStake(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByStake",
        value: MsgQualifyByStake.fromPartial(value)
      };
    },
    qualifyByTrackRecord(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByTrackRecord",
        value: MsgQualifyByTrackRecord.fromPartial(value)
      };
    },
    qualifyByCrossReference(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByCrossReference",
        value: MsgQualifyByCrossReference.fromPartial(value)
      };
    },
    qualifyByInheritance(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgQualifyByInheritance",
        value: MsgQualifyByInheritance.fromPartial(value)
      };
    },
    endorseQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgEndorseQualification",
        value: MsgEndorseQualification.fromPartial(value)
      };
    },
    renewQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgRenewQualification",
        value: MsgRenewQualification.fromPartial(value)
      };
    },
    withdrawQualification(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgWithdrawQualification",
        value: MsgWithdrawQualification.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.qualification.v1.MsgUpdateParams",
        value: MsgUpdateParams14.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/sponsorship/v1/tx.ts
var tx_exports16 = {};
__export(tx_exports16, {
  MsgCancelBountyOrder: () => MsgCancelBountyOrder,
  MsgCancelBountyOrderResponse: () => MsgCancelBountyOrderResponse,
  MsgCreateBountyOrder: () => MsgCreateBountyOrder,
  MsgCreateBountyOrderResponse: () => MsgCreateBountyOrderResponse,
  MsgFulfillBounty: () => MsgFulfillBounty,
  MsgFulfillBountyResponse: () => MsgFulfillBountyResponse
});
function createBaseMsgCreateBountyOrder() {
  return {
    sponsor: "",
    domain: "",
    pricePerArtifact: "",
    targetCount: 0,
    durationBlocks: BigInt(0)
  };
}
var MsgCreateBountyOrder = {
  typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrder",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sponsor !== "") {
      writer.uint32(10).string(message.sponsor);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.pricePerArtifact !== "") {
      writer.uint32(26).string(message.pricePerArtifact);
    }
    if (message.targetCount !== 0) {
      writer.uint32(32).uint32(message.targetCount);
    }
    if (message.durationBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.durationBlocks);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateBountyOrder();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sponsor = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.pricePerArtifact = reader.string();
          break;
        case 4:
          message.targetCount = reader.uint32();
          break;
        case 5:
          message.durationBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateBountyOrder();
    message.sponsor = object.sponsor ?? "";
    message.domain = object.domain ?? "";
    message.pricePerArtifact = object.pricePerArtifact ?? "";
    message.targetCount = object.targetCount ?? 0;
    message.durationBlocks = object.durationBlocks !== void 0 && object.durationBlocks !== null ? BigInt(object.durationBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgCreateBountyOrderResponse() {
  return {
    bountyId: ""
  };
}
var MsgCreateBountyOrderResponse = {
  typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrderResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.bountyId !== "") {
      writer.uint32(10).string(message.bountyId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateBountyOrderResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.bountyId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateBountyOrderResponse();
    message.bountyId = object.bountyId ?? "";
    return message;
  }
};
function createBaseMsgFulfillBounty() {
  return {
    caller: "",
    bountyId: "",
    factId: ""
  };
}
var MsgFulfillBounty = {
  typeUrl: "/zerone.sponsorship.v1.MsgFulfillBounty",
  encode(message, writer = BinaryWriter.create()) {
    if (message.caller !== "") {
      writer.uint32(10).string(message.caller);
    }
    if (message.bountyId !== "") {
      writer.uint32(18).string(message.bountyId);
    }
    if (message.factId !== "") {
      writer.uint32(26).string(message.factId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFulfillBounty();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.caller = reader.string();
          break;
        case 2:
          message.bountyId = reader.string();
          break;
        case 3:
          message.factId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgFulfillBounty();
    message.caller = object.caller ?? "";
    message.bountyId = object.bountyId ?? "";
    message.factId = object.factId ?? "";
    return message;
  }
};
function createBaseMsgFulfillBountyResponse() {
  return {
    worker: "",
    amountPaid: "",
    bountyNowFulfilled: false
  };
}
var MsgFulfillBountyResponse = {
  typeUrl: "/zerone.sponsorship.v1.MsgFulfillBountyResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.worker !== "") {
      writer.uint32(10).string(message.worker);
    }
    if (message.amountPaid !== "") {
      writer.uint32(18).string(message.amountPaid);
    }
    if (message.bountyNowFulfilled === true) {
      writer.uint32(24).bool(message.bountyNowFulfilled);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFulfillBountyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.worker = reader.string();
          break;
        case 2:
          message.amountPaid = reader.string();
          break;
        case 3:
          message.bountyNowFulfilled = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgFulfillBountyResponse();
    message.worker = object.worker ?? "";
    message.amountPaid = object.amountPaid ?? "";
    message.bountyNowFulfilled = object.bountyNowFulfilled ?? false;
    return message;
  }
};
function createBaseMsgCancelBountyOrder() {
  return {
    sponsor: "",
    bountyId: ""
  };
}
var MsgCancelBountyOrder = {
  typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrder",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sponsor !== "") {
      writer.uint32(10).string(message.sponsor);
    }
    if (message.bountyId !== "") {
      writer.uint32(18).string(message.bountyId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelBountyOrder();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sponsor = reader.string();
          break;
        case 2:
          message.bountyId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCancelBountyOrder();
    message.sponsor = object.sponsor ?? "";
    message.bountyId = object.bountyId ?? "";
    return message;
  }
};
function createBaseMsgCancelBountyOrderResponse() {
  return {
    refundedAmount: ""
  };
}
var MsgCancelBountyOrderResponse = {
  typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrderResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.refundedAmount !== "") {
      writer.uint32(10).string(message.refundedAmount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelBountyOrderResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.refundedAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCancelBountyOrderResponse();
    message.refundedAmount = object.refundedAmount ?? "";
    return message;
  }
};

// src/generated/zerone/sponsorship/v1/tx.registry.ts
var registry16 = [["/zerone.sponsorship.v1.MsgCreateBountyOrder", MsgCreateBountyOrder], ["/zerone.sponsorship.v1.MsgFulfillBounty", MsgFulfillBounty], ["/zerone.sponsorship.v1.MsgCancelBountyOrder", MsgCancelBountyOrder]];
var MessageComposer16 = {
  encoded: {
    createBountyOrder(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrder",
        value: MsgCreateBountyOrder.encode(value).finish()
      };
    },
    fulfillBounty(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgFulfillBounty",
        value: MsgFulfillBounty.encode(value).finish()
      };
    },
    cancelBountyOrder(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrder",
        value: MsgCancelBountyOrder.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createBountyOrder(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrder",
        value
      };
    },
    fulfillBounty(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgFulfillBounty",
        value
      };
    },
    cancelBountyOrder(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrder",
        value
      };
    }
  },
  fromPartial: {
    createBountyOrder(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCreateBountyOrder",
        value: MsgCreateBountyOrder.fromPartial(value)
      };
    },
    fulfillBounty(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgFulfillBounty",
        value: MsgFulfillBounty.fromPartial(value)
      };
    },
    cancelBountyOrder(value) {
      return {
        typeUrl: "/zerone.sponsorship.v1.MsgCancelBountyOrder",
        value: MsgCancelBountyOrder.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/staking/v1/tx.ts
var tx_exports17 = {};
__export(tx_exports17, {
  MsgDelegate: () => MsgDelegate,
  MsgDelegateResponse: () => MsgDelegateResponse,
  MsgRedelegate: () => MsgRedelegate,
  MsgRedelegateResponse: () => MsgRedelegateResponse,
  MsgRegisterValidator: () => MsgRegisterValidator,
  MsgRegisterValidatorResponse: () => MsgRegisterValidatorResponse,
  MsgUndelegate: () => MsgUndelegate,
  MsgUndelegateResponse: () => MsgUndelegateResponse,
  MsgUpdateParams: () => MsgUpdateParams15,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse15,
  MsgUpdateValidatorStake: () => MsgUpdateValidatorStake,
  MsgUpdateValidatorStakeResponse: () => MsgUpdateValidatorStakeResponse
});

// src/generated/zerone/staking/v1/types.ts
function createBaseTierConfig() {
  return {
    tier: 0,
    name: "",
    minStake: "",
    minReputation: BigInt(0),
    minVerifications: BigInt(0),
    minAccuracy: BigInt(0),
    maxSlashCount: BigInt(0),
    allowedCategories: [],
    rewardMultiplierBps: BigInt(0),
    selectionWeightBps: BigInt(0),
    slashWindowEpochs: BigInt(0),
    minContestedVerifications: BigInt(0),
    contestedVerificationMultiplier: BigInt(0),
    slashMultiplierBps: BigInt(0)
  };
}
var TierConfig = {
  typeUrl: "/zerone.staking.v1.TierConfig",
  encode(message, writer = BinaryWriter.create()) {
    if (message.tier !== 0) {
      writer.uint32(8).int32(message.tier);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.minStake !== "") {
      writer.uint32(26).string(message.minStake);
    }
    if (message.minReputation !== BigInt(0)) {
      writer.uint32(32).uint64(message.minReputation);
    }
    if (message.minVerifications !== BigInt(0)) {
      writer.uint32(40).uint64(message.minVerifications);
    }
    if (message.minAccuracy !== BigInt(0)) {
      writer.uint32(48).uint64(message.minAccuracy);
    }
    if (message.maxSlashCount !== BigInt(0)) {
      writer.uint32(56).int64(message.maxSlashCount);
    }
    for (const v of message.allowedCategories) {
      writer.uint32(66).string(v);
    }
    if (message.rewardMultiplierBps !== BigInt(0)) {
      writer.uint32(72).uint64(message.rewardMultiplierBps);
    }
    if (message.selectionWeightBps !== BigInt(0)) {
      writer.uint32(80).uint64(message.selectionWeightBps);
    }
    if (message.slashWindowEpochs !== BigInt(0)) {
      writer.uint32(88).uint64(message.slashWindowEpochs);
    }
    if (message.minContestedVerifications !== BigInt(0)) {
      writer.uint32(96).uint64(message.minContestedVerifications);
    }
    if (message.contestedVerificationMultiplier !== BigInt(0)) {
      writer.uint32(104).uint64(message.contestedVerificationMultiplier);
    }
    if (message.slashMultiplierBps !== BigInt(0)) {
      writer.uint32(112).uint64(message.slashMultiplierBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseTierConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tier = reader.int32();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.minStake = reader.string();
          break;
        case 4:
          message.minReputation = reader.uint64();
          break;
        case 5:
          message.minVerifications = reader.uint64();
          break;
        case 6:
          message.minAccuracy = reader.uint64();
          break;
        case 7:
          message.maxSlashCount = reader.int64();
          break;
        case 8:
          message.allowedCategories.push(reader.string());
          break;
        case 9:
          message.rewardMultiplierBps = reader.uint64();
          break;
        case 10:
          message.selectionWeightBps = reader.uint64();
          break;
        case 11:
          message.slashWindowEpochs = reader.uint64();
          break;
        case 12:
          message.minContestedVerifications = reader.uint64();
          break;
        case 13:
          message.contestedVerificationMultiplier = reader.uint64();
          break;
        case 14:
          message.slashMultiplierBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseTierConfig();
    message.tier = object.tier ?? 0;
    message.name = object.name ?? "";
    message.minStake = object.minStake ?? "";
    message.minReputation = object.minReputation !== void 0 && object.minReputation !== null ? BigInt(object.minReputation.toString()) : BigInt(0);
    message.minVerifications = object.minVerifications !== void 0 && object.minVerifications !== null ? BigInt(object.minVerifications.toString()) : BigInt(0);
    message.minAccuracy = object.minAccuracy !== void 0 && object.minAccuracy !== null ? BigInt(object.minAccuracy.toString()) : BigInt(0);
    message.maxSlashCount = object.maxSlashCount !== void 0 && object.maxSlashCount !== null ? BigInt(object.maxSlashCount.toString()) : BigInt(0);
    message.allowedCategories = object.allowedCategories?.map((e) => e) || [];
    message.rewardMultiplierBps = object.rewardMultiplierBps !== void 0 && object.rewardMultiplierBps !== null ? BigInt(object.rewardMultiplierBps.toString()) : BigInt(0);
    message.selectionWeightBps = object.selectionWeightBps !== void 0 && object.selectionWeightBps !== null ? BigInt(object.selectionWeightBps.toString()) : BigInt(0);
    message.slashWindowEpochs = object.slashWindowEpochs !== void 0 && object.slashWindowEpochs !== null ? BigInt(object.slashWindowEpochs.toString()) : BigInt(0);
    message.minContestedVerifications = object.minContestedVerifications !== void 0 && object.minContestedVerifications !== null ? BigInt(object.minContestedVerifications.toString()) : BigInt(0);
    message.contestedVerificationMultiplier = object.contestedVerificationMultiplier !== void 0 && object.contestedVerificationMultiplier !== null ? BigInt(object.contestedVerificationMultiplier.toString()) : BigInt(0);
    message.slashMultiplierBps = object.slashMultiplierBps !== void 0 && object.slashMultiplierBps !== null ? BigInt(object.slashMultiplierBps.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/staking/v1/genesis.ts
function createBaseParams16() {
  return {
    unbondingPeriod: BigInt(0),
    virtualStake: "",
    maxValidators: BigInt(0),
    minSelfDelegation: "",
    maxSlashesPerEpoch: BigInt(0),
    slashDecayPeriodBlocks: BigInt(0),
    maxSlashCountDeactivate: BigInt(0),
    minStakeForVerification: "",
    slashEscalationBps: BigInt(0),
    reputationCorrectDelta: BigInt(0),
    reputationIncorrectDelta: BigInt(0),
    reputationSlashDelta: BigInt(0),
    redelegationCooldownBlocks: BigInt(0),
    tierConfigs: []
  };
}
var Params16 = {
  typeUrl: "/zerone.staking.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.unbondingPeriod !== BigInt(0)) {
      writer.uint32(8).uint64(message.unbondingPeriod);
    }
    if (message.virtualStake !== "") {
      writer.uint32(18).string(message.virtualStake);
    }
    if (message.maxValidators !== BigInt(0)) {
      writer.uint32(24).uint64(message.maxValidators);
    }
    if (message.minSelfDelegation !== "") {
      writer.uint32(34).string(message.minSelfDelegation);
    }
    if (message.maxSlashesPerEpoch !== BigInt(0)) {
      writer.uint32(40).uint64(message.maxSlashesPerEpoch);
    }
    if (message.slashDecayPeriodBlocks !== BigInt(0)) {
      writer.uint32(48).uint64(message.slashDecayPeriodBlocks);
    }
    if (message.maxSlashCountDeactivate !== BigInt(0)) {
      writer.uint32(56).uint64(message.maxSlashCountDeactivate);
    }
    if (message.minStakeForVerification !== "") {
      writer.uint32(66).string(message.minStakeForVerification);
    }
    if (message.slashEscalationBps !== BigInt(0)) {
      writer.uint32(72).uint64(message.slashEscalationBps);
    }
    if (message.reputationCorrectDelta !== BigInt(0)) {
      writer.uint32(80).uint64(message.reputationCorrectDelta);
    }
    if (message.reputationIncorrectDelta !== BigInt(0)) {
      writer.uint32(88).uint64(message.reputationIncorrectDelta);
    }
    if (message.reputationSlashDelta !== BigInt(0)) {
      writer.uint32(96).uint64(message.reputationSlashDelta);
    }
    if (message.redelegationCooldownBlocks !== BigInt(0)) {
      writer.uint32(104).uint64(message.redelegationCooldownBlocks);
    }
    for (const v of message.tierConfigs) {
      TierConfig.encode(v, writer.uint32(114).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams16();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.unbondingPeriod = reader.uint64();
          break;
        case 2:
          message.virtualStake = reader.string();
          break;
        case 3:
          message.maxValidators = reader.uint64();
          break;
        case 4:
          message.minSelfDelegation = reader.string();
          break;
        case 5:
          message.maxSlashesPerEpoch = reader.uint64();
          break;
        case 6:
          message.slashDecayPeriodBlocks = reader.uint64();
          break;
        case 7:
          message.maxSlashCountDeactivate = reader.uint64();
          break;
        case 8:
          message.minStakeForVerification = reader.string();
          break;
        case 9:
          message.slashEscalationBps = reader.uint64();
          break;
        case 10:
          message.reputationCorrectDelta = reader.uint64();
          break;
        case 11:
          message.reputationIncorrectDelta = reader.uint64();
          break;
        case 12:
          message.reputationSlashDelta = reader.uint64();
          break;
        case 13:
          message.redelegationCooldownBlocks = reader.uint64();
          break;
        case 14:
          message.tierConfigs.push(TierConfig.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams16();
    message.unbondingPeriod = object.unbondingPeriod !== void 0 && object.unbondingPeriod !== null ? BigInt(object.unbondingPeriod.toString()) : BigInt(0);
    message.virtualStake = object.virtualStake ?? "";
    message.maxValidators = object.maxValidators !== void 0 && object.maxValidators !== null ? BigInt(object.maxValidators.toString()) : BigInt(0);
    message.minSelfDelegation = object.minSelfDelegation ?? "";
    message.maxSlashesPerEpoch = object.maxSlashesPerEpoch !== void 0 && object.maxSlashesPerEpoch !== null ? BigInt(object.maxSlashesPerEpoch.toString()) : BigInt(0);
    message.slashDecayPeriodBlocks = object.slashDecayPeriodBlocks !== void 0 && object.slashDecayPeriodBlocks !== null ? BigInt(object.slashDecayPeriodBlocks.toString()) : BigInt(0);
    message.maxSlashCountDeactivate = object.maxSlashCountDeactivate !== void 0 && object.maxSlashCountDeactivate !== null ? BigInt(object.maxSlashCountDeactivate.toString()) : BigInt(0);
    message.minStakeForVerification = object.minStakeForVerification ?? "";
    message.slashEscalationBps = object.slashEscalationBps !== void 0 && object.slashEscalationBps !== null ? BigInt(object.slashEscalationBps.toString()) : BigInt(0);
    message.reputationCorrectDelta = object.reputationCorrectDelta !== void 0 && object.reputationCorrectDelta !== null ? BigInt(object.reputationCorrectDelta.toString()) : BigInt(0);
    message.reputationIncorrectDelta = object.reputationIncorrectDelta !== void 0 && object.reputationIncorrectDelta !== null ? BigInt(object.reputationIncorrectDelta.toString()) : BigInt(0);
    message.reputationSlashDelta = object.reputationSlashDelta !== void 0 && object.reputationSlashDelta !== null ? BigInt(object.reputationSlashDelta.toString()) : BigInt(0);
    message.redelegationCooldownBlocks = object.redelegationCooldownBlocks !== void 0 && object.redelegationCooldownBlocks !== null ? BigInt(object.redelegationCooldownBlocks.toString()) : BigInt(0);
    message.tierConfigs = object.tierConfigs?.map((e) => TierConfig.fromPartial(e)) || [];
    return message;
  }
};

// src/generated/zerone/staking/v1/tx.ts
function createBaseMsgRegisterValidator() {
  return {
    operator: "",
    consensusPubkey: "",
    did: "",
    moniker: "",
    selfDelegation: "",
    commissionBps: BigInt(0),
    website: "",
    details: ""
  };
}
var MsgRegisterValidator = {
  typeUrl: "/zerone.staking.v1.MsgRegisterValidator",
  encode(message, writer = BinaryWriter.create()) {
    if (message.operator !== "") {
      writer.uint32(10).string(message.operator);
    }
    if (message.consensusPubkey !== "") {
      writer.uint32(18).string(message.consensusPubkey);
    }
    if (message.did !== "") {
      writer.uint32(26).string(message.did);
    }
    if (message.moniker !== "") {
      writer.uint32(34).string(message.moniker);
    }
    if (message.selfDelegation !== "") {
      writer.uint32(42).string(message.selfDelegation);
    }
    if (message.commissionBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.commissionBps);
    }
    if (message.website !== "") {
      writer.uint32(58).string(message.website);
    }
    if (message.details !== "") {
      writer.uint32(66).string(message.details);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterValidator();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operator = reader.string();
          break;
        case 2:
          message.consensusPubkey = reader.string();
          break;
        case 3:
          message.did = reader.string();
          break;
        case 4:
          message.moniker = reader.string();
          break;
        case 5:
          message.selfDelegation = reader.string();
          break;
        case 6:
          message.commissionBps = reader.uint64();
          break;
        case 7:
          message.website = reader.string();
          break;
        case 8:
          message.details = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterValidator();
    message.operator = object.operator ?? "";
    message.consensusPubkey = object.consensusPubkey ?? "";
    message.did = object.did ?? "";
    message.moniker = object.moniker ?? "";
    message.selfDelegation = object.selfDelegation ?? "";
    message.commissionBps = object.commissionBps !== void 0 && object.commissionBps !== null ? BigInt(object.commissionBps.toString()) : BigInt(0);
    message.website = object.website ?? "";
    message.details = object.details ?? "";
    return message;
  }
};
function createBaseMsgRegisterValidatorResponse() {
  return {
    initialTier: 0
  };
}
var MsgRegisterValidatorResponse = {
  typeUrl: "/zerone.staking.v1.MsgRegisterValidatorResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.initialTier !== 0) {
      writer.uint32(8).uint32(message.initialTier);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterValidatorResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.initialTier = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterValidatorResponse();
    message.initialTier = object.initialTier ?? 0;
    return message;
  }
};
function createBaseMsgDelegate() {
  return {
    delegator: "",
    validator: "",
    amount: ""
  };
}
var MsgDelegate = {
  typeUrl: "/zerone.staking.v1.MsgDelegate",
  encode(message, writer = BinaryWriter.create()) {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.validator !== "") {
      writer.uint32(18).string(message.validator);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgDelegate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.validator = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgDelegate();
    message.delegator = object.delegator ?? "";
    message.validator = object.validator ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgDelegateResponse() {
  return {
    newDelegation: ""
  };
}
var MsgDelegateResponse = {
  typeUrl: "/zerone.staking.v1.MsgDelegateResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.newDelegation !== "") {
      writer.uint32(10).string(message.newDelegation);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgDelegateResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newDelegation = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgDelegateResponse();
    message.newDelegation = object.newDelegation ?? "";
    return message;
  }
};
function createBaseMsgUndelegate() {
  return {
    delegator: "",
    validator: "",
    amount: ""
  };
}
var MsgUndelegate = {
  typeUrl: "/zerone.staking.v1.MsgUndelegate",
  encode(message, writer = BinaryWriter.create()) {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.validator !== "") {
      writer.uint32(18).string(message.validator);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUndelegate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.validator = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUndelegate();
    message.delegator = object.delegator ?? "";
    message.validator = object.validator ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgUndelegateResponse() {
  return {
    unbondingId: "",
    completesAtHeight: BigInt(0)
  };
}
var MsgUndelegateResponse = {
  typeUrl: "/zerone.staking.v1.MsgUndelegateResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.unbondingId !== "") {
      writer.uint32(10).string(message.unbondingId);
    }
    if (message.completesAtHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.completesAtHeight);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUndelegateResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.unbondingId = reader.string();
          break;
        case 2:
          message.completesAtHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUndelegateResponse();
    message.unbondingId = object.unbondingId ?? "";
    message.completesAtHeight = object.completesAtHeight !== void 0 && object.completesAtHeight !== null ? BigInt(object.completesAtHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgRedelegate() {
  return {
    delegator: "",
    srcValidator: "",
    dstValidator: "",
    amount: ""
  };
}
var MsgRedelegate = {
  typeUrl: "/zerone.staking.v1.MsgRedelegate",
  encode(message, writer = BinaryWriter.create()) {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.srcValidator !== "") {
      writer.uint32(18).string(message.srcValidator);
    }
    if (message.dstValidator !== "") {
      writer.uint32(26).string(message.dstValidator);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRedelegate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.srcValidator = reader.string();
          break;
        case 3:
          message.dstValidator = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRedelegate();
    message.delegator = object.delegator ?? "";
    message.srcValidator = object.srcValidator ?? "";
    message.dstValidator = object.dstValidator ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgRedelegateResponse() {
  return {};
}
var MsgRedelegateResponse = {
  typeUrl: "/zerone.staking.v1.MsgRedelegateResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRedelegateResponse();
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
    const message = createBaseMsgRedelegateResponse();
    return message;
  }
};
function createBaseMsgUpdateValidatorStake() {
  return {
    operator: "",
    amount: "",
    increase: false
  };
}
var MsgUpdateValidatorStake = {
  typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStake",
  encode(message, writer = BinaryWriter.create()) {
    if (message.operator !== "") {
      writer.uint32(10).string(message.operator);
    }
    if (message.amount !== "") {
      writer.uint32(18).string(message.amount);
    }
    if (message.increase === true) {
      writer.uint32(24).bool(message.increase);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateValidatorStake();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operator = reader.string();
          break;
        case 2:
          message.amount = reader.string();
          break;
        case 3:
          message.increase = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateValidatorStake();
    message.operator = object.operator ?? "";
    message.amount = object.amount ?? "";
    message.increase = object.increase ?? false;
    return message;
  }
};
function createBaseMsgUpdateValidatorStakeResponse() {
  return {};
}
var MsgUpdateValidatorStakeResponse = {
  typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStakeResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateValidatorStakeResponse();
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
    const message = createBaseMsgUpdateValidatorStakeResponse();
    return message;
  }
};
function createBaseMsgUpdateParams15() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams15 = {
  typeUrl: "/zerone.staking.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params16.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams15();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params16.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams15();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params16.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse15() {
  return {};
}
var MsgUpdateParamsResponse15 = {
  typeUrl: "/zerone.staking.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse15();
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
    const message = createBaseMsgUpdateParamsResponse15();
    return message;
  }
};

// src/generated/zerone/staking/v1/tx.registry.ts
var registry17 = [["/zerone.staking.v1.MsgRegisterValidator", MsgRegisterValidator], ["/zerone.staking.v1.MsgDelegate", MsgDelegate], ["/zerone.staking.v1.MsgUndelegate", MsgUndelegate], ["/zerone.staking.v1.MsgRedelegate", MsgRedelegate], ["/zerone.staking.v1.MsgUpdateValidatorStake", MsgUpdateValidatorStake], ["/zerone.staking.v1.MsgUpdateParams", MsgUpdateParams15]];
var MessageComposer17 = {
  encoded: {
    registerValidator(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRegisterValidator",
        value: MsgRegisterValidator.encode(value).finish()
      };
    },
    delegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgDelegate",
        value: MsgDelegate.encode(value).finish()
      };
    },
    undelegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUndelegate",
        value: MsgUndelegate.encode(value).finish()
      };
    },
    redelegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRedelegate",
        value: MsgRedelegate.encode(value).finish()
      };
    },
    updateValidatorStake(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStake",
        value: MsgUpdateValidatorStake.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateParams",
        value: MsgUpdateParams15.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    registerValidator(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRegisterValidator",
        value
      };
    },
    delegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgDelegate",
        value
      };
    },
    undelegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUndelegate",
        value
      };
    },
    redelegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRedelegate",
        value
      };
    },
    updateValidatorStake(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStake",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    registerValidator(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRegisterValidator",
        value: MsgRegisterValidator.fromPartial(value)
      };
    },
    delegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgDelegate",
        value: MsgDelegate.fromPartial(value)
      };
    },
    undelegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUndelegate",
        value: MsgUndelegate.fromPartial(value)
      };
    },
    redelegate(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgRedelegate",
        value: MsgRedelegate.fromPartial(value)
      };
    },
    updateValidatorStake(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStake",
        value: MsgUpdateValidatorStake.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.staking.v1.MsgUpdateParams",
        value: MsgUpdateParams15.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/substrate_bridge/v1/tx.ts
var tx_exports18 = {};
__export(tx_exports18, {
  MsgRegisterAdapter: () => MsgRegisterAdapter,
  MsgRegisterAdapterResponse: () => MsgRegisterAdapterResponse,
  MsgSubmitExternalAttestation: () => MsgSubmitExternalAttestation,
  MsgSubmitExternalAttestationResponse: () => MsgSubmitExternalAttestationResponse,
  MsgSuspendAdapter: () => MsgSuspendAdapter,
  MsgSuspendAdapterResponse: () => MsgSuspendAdapterResponse,
  MsgTombstoneAdapter: () => MsgTombstoneAdapter,
  MsgTombstoneAdapterResponse: () => MsgTombstoneAdapterResponse
});

// src/generated/zerone/substrate_bridge/v1/types.ts
function createBaseExternalSource() {
  return {
    adapterId: "",
    sourceId: "",
    sourceUrl: "",
    contentHash: new Uint8Array(),
    fetchedAtBlock: BigInt(0)
  };
}
var ExternalSource = {
  typeUrl: "/zerone.substrate_bridge.v1.ExternalSource",
  encode(message, writer = BinaryWriter.create()) {
    if (message.adapterId !== "") {
      writer.uint32(10).string(message.adapterId);
    }
    if (message.sourceId !== "") {
      writer.uint32(18).string(message.sourceId);
    }
    if (message.sourceUrl !== "") {
      writer.uint32(26).string(message.sourceUrl);
    }
    if (message.contentHash.length !== 0) {
      writer.uint32(34).bytes(message.contentHash);
    }
    if (message.fetchedAtBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.fetchedAtBlock);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseExternalSource();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.adapterId = reader.string();
          break;
        case 2:
          message.sourceId = reader.string();
          break;
        case 3:
          message.sourceUrl = reader.string();
          break;
        case 4:
          message.contentHash = reader.bytes();
          break;
        case 5:
          message.fetchedAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseExternalSource();
    message.adapterId = object.adapterId ?? "";
    message.sourceId = object.sourceId ?? "";
    message.sourceUrl = object.sourceUrl ?? "";
    message.contentHash = object.contentHash ?? new Uint8Array();
    message.fetchedAtBlock = object.fetchedAtBlock !== void 0 && object.fetchedAtBlock !== null ? BigInt(object.fetchedAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseAxisProjection() {
  return {
    axisSubstrate: BigInt(0),
    axisVerification: BigInt(0),
    axisClassification: BigInt(0),
    axisAttribution: BigInt(0),
    axisTooling: BigInt(0),
    axisInterface: BigInt(0)
  };
}
var AxisProjection = {
  typeUrl: "/zerone.substrate_bridge.v1.AxisProjection",
  encode(message, writer = BinaryWriter.create()) {
    if (message.axisSubstrate !== BigInt(0)) {
      writer.uint32(8).uint64(message.axisSubstrate);
    }
    if (message.axisVerification !== BigInt(0)) {
      writer.uint32(16).uint64(message.axisVerification);
    }
    if (message.axisClassification !== BigInt(0)) {
      writer.uint32(24).uint64(message.axisClassification);
    }
    if (message.axisAttribution !== BigInt(0)) {
      writer.uint32(32).uint64(message.axisAttribution);
    }
    if (message.axisTooling !== BigInt(0)) {
      writer.uint32(40).uint64(message.axisTooling);
    }
    if (message.axisInterface !== BigInt(0)) {
      writer.uint32(48).uint64(message.axisInterface);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseAxisProjection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.axisSubstrate = reader.uint64();
          break;
        case 2:
          message.axisVerification = reader.uint64();
          break;
        case 3:
          message.axisClassification = reader.uint64();
          break;
        case 4:
          message.axisAttribution = reader.uint64();
          break;
        case 5:
          message.axisTooling = reader.uint64();
          break;
        case 6:
          message.axisInterface = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseAxisProjection();
    message.axisSubstrate = object.axisSubstrate !== void 0 && object.axisSubstrate !== null ? BigInt(object.axisSubstrate.toString()) : BigInt(0);
    message.axisVerification = object.axisVerification !== void 0 && object.axisVerification !== null ? BigInt(object.axisVerification.toString()) : BigInt(0);
    message.axisClassification = object.axisClassification !== void 0 && object.axisClassification !== null ? BigInt(object.axisClassification.toString()) : BigInt(0);
    message.axisAttribution = object.axisAttribution !== void 0 && object.axisAttribution !== null ? BigInt(object.axisAttribution.toString()) : BigInt(0);
    message.axisTooling = object.axisTooling !== void 0 && object.axisTooling !== null ? BigInt(object.axisTooling.toString()) : BigInt(0);
    message.axisInterface = object.axisInterface !== void 0 && object.axisInterface !== null ? BigInt(object.axisInterface.toString()) : BigInt(0);
    return message;
  }
};
function createBaseAxisBounds() {
  return {
    axisSubstrateMax: BigInt(0),
    axisVerificationMax: BigInt(0),
    axisClassificationMax: BigInt(0),
    axisAttributionMax: BigInt(0),
    axisToolingMax: BigInt(0),
    axisInterfaceMax: BigInt(0)
  };
}
var AxisBounds = {
  typeUrl: "/zerone.substrate_bridge.v1.AxisBounds",
  encode(message, writer = BinaryWriter.create()) {
    if (message.axisSubstrateMax !== BigInt(0)) {
      writer.uint32(8).uint64(message.axisSubstrateMax);
    }
    if (message.axisVerificationMax !== BigInt(0)) {
      writer.uint32(16).uint64(message.axisVerificationMax);
    }
    if (message.axisClassificationMax !== BigInt(0)) {
      writer.uint32(24).uint64(message.axisClassificationMax);
    }
    if (message.axisAttributionMax !== BigInt(0)) {
      writer.uint32(32).uint64(message.axisAttributionMax);
    }
    if (message.axisToolingMax !== BigInt(0)) {
      writer.uint32(40).uint64(message.axisToolingMax);
    }
    if (message.axisInterfaceMax !== BigInt(0)) {
      writer.uint32(48).uint64(message.axisInterfaceMax);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseAxisBounds();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.axisSubstrateMax = reader.uint64();
          break;
        case 2:
          message.axisVerificationMax = reader.uint64();
          break;
        case 3:
          message.axisClassificationMax = reader.uint64();
          break;
        case 4:
          message.axisAttributionMax = reader.uint64();
          break;
        case 5:
          message.axisToolingMax = reader.uint64();
          break;
        case 6:
          message.axisInterfaceMax = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseAxisBounds();
    message.axisSubstrateMax = object.axisSubstrateMax !== void 0 && object.axisSubstrateMax !== null ? BigInt(object.axisSubstrateMax.toString()) : BigInt(0);
    message.axisVerificationMax = object.axisVerificationMax !== void 0 && object.axisVerificationMax !== null ? BigInt(object.axisVerificationMax.toString()) : BigInt(0);
    message.axisClassificationMax = object.axisClassificationMax !== void 0 && object.axisClassificationMax !== null ? BigInt(object.axisClassificationMax.toString()) : BigInt(0);
    message.axisAttributionMax = object.axisAttributionMax !== void 0 && object.axisAttributionMax !== null ? BigInt(object.axisAttributionMax.toString()) : BigInt(0);
    message.axisToolingMax = object.axisToolingMax !== void 0 && object.axisToolingMax !== null ? BigInt(object.axisToolingMax.toString()) : BigInt(0);
    message.axisInterfaceMax = object.axisInterfaceMax !== void 0 && object.axisInterfaceMax !== null ? BigInt(object.axisInterfaceMax.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/substrate_bridge/v1/adapter.ts
function createBaseSlashGradient() {
  return {
    compilerDriftBps: 0,
    axisOverflowBps: 0,
    fraudBps: 0
  };
}
var SlashGradient = {
  typeUrl: "/zerone.substrate_bridge.v1.SlashGradient",
  encode(message, writer = BinaryWriter.create()) {
    if (message.compilerDriftBps !== 0) {
      writer.uint32(8).uint32(message.compilerDriftBps);
    }
    if (message.axisOverflowBps !== 0) {
      writer.uint32(16).uint32(message.axisOverflowBps);
    }
    if (message.fraudBps !== 0) {
      writer.uint32(24).uint32(message.fraudBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseSlashGradient();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.compilerDriftBps = reader.uint32();
          break;
        case 2:
          message.axisOverflowBps = reader.uint32();
          break;
        case 3:
          message.fraudBps = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseSlashGradient();
    message.compilerDriftBps = object.compilerDriftBps ?? 0;
    message.axisOverflowBps = object.axisOverflowBps ?? 0;
    message.fraudBps = object.fraudBps ?? 0;
    return message;
  }
};
function createBaseAdapterRegistration() {
  return {
    adapterId: "",
    sourceType: "",
    version: "",
    compilerBinaryHash: new Uint8Array(),
    axisBounds: void 0,
    minAttestationBondUzrn: "",
    minPerClaimBondUzrn: "",
    slashGradient: void 0,
    requiredQualificationDomain: "",
    minQualificationStatus: 0,
    allowedClassIds: [],
    status: 0,
    registeredViaLipId: "",
    registeredAtBlock: BigInt(0),
    tombstonedAtBlock: BigInt(0),
    witnessRewardUzrn: ""
  };
}
var AdapterRegistration = {
  typeUrl: "/zerone.substrate_bridge.v1.AdapterRegistration",
  encode(message, writer = BinaryWriter.create()) {
    if (message.adapterId !== "") {
      writer.uint32(10).string(message.adapterId);
    }
    if (message.sourceType !== "") {
      writer.uint32(18).string(message.sourceType);
    }
    if (message.version !== "") {
      writer.uint32(26).string(message.version);
    }
    if (message.compilerBinaryHash.length !== 0) {
      writer.uint32(34).bytes(message.compilerBinaryHash);
    }
    if (message.axisBounds !== void 0) {
      AxisBounds.encode(message.axisBounds, writer.uint32(42).fork()).ldelim();
    }
    if (message.minAttestationBondUzrn !== "") {
      writer.uint32(50).string(message.minAttestationBondUzrn);
    }
    if (message.minPerClaimBondUzrn !== "") {
      writer.uint32(58).string(message.minPerClaimBondUzrn);
    }
    if (message.slashGradient !== void 0) {
      SlashGradient.encode(message.slashGradient, writer.uint32(66).fork()).ldelim();
    }
    if (message.requiredQualificationDomain !== "") {
      writer.uint32(74).string(message.requiredQualificationDomain);
    }
    if (message.minQualificationStatus !== 0) {
      writer.uint32(80).int32(message.minQualificationStatus);
    }
    for (const v of message.allowedClassIds) {
      writer.uint32(90).string(v);
    }
    if (message.status !== 0) {
      writer.uint32(96).int32(message.status);
    }
    if (message.registeredViaLipId !== "") {
      writer.uint32(106).string(message.registeredViaLipId);
    }
    if (message.registeredAtBlock !== BigInt(0)) {
      writer.uint32(112).uint64(message.registeredAtBlock);
    }
    if (message.tombstonedAtBlock !== BigInt(0)) {
      writer.uint32(120).uint64(message.tombstonedAtBlock);
    }
    if (message.witnessRewardUzrn !== "") {
      writer.uint32(130).string(message.witnessRewardUzrn);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseAdapterRegistration();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.adapterId = reader.string();
          break;
        case 2:
          message.sourceType = reader.string();
          break;
        case 3:
          message.version = reader.string();
          break;
        case 4:
          message.compilerBinaryHash = reader.bytes();
          break;
        case 5:
          message.axisBounds = AxisBounds.decode(reader, reader.uint32());
          break;
        case 6:
          message.minAttestationBondUzrn = reader.string();
          break;
        case 7:
          message.minPerClaimBondUzrn = reader.string();
          break;
        case 8:
          message.slashGradient = SlashGradient.decode(reader, reader.uint32());
          break;
        case 9:
          message.requiredQualificationDomain = reader.string();
          break;
        case 10:
          message.minQualificationStatus = reader.int32();
          break;
        case 11:
          message.allowedClassIds.push(reader.string());
          break;
        case 12:
          message.status = reader.int32();
          break;
        case 13:
          message.registeredViaLipId = reader.string();
          break;
        case 14:
          message.registeredAtBlock = reader.uint64();
          break;
        case 15:
          message.tombstonedAtBlock = reader.uint64();
          break;
        case 16:
          message.witnessRewardUzrn = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseAdapterRegistration();
    message.adapterId = object.adapterId ?? "";
    message.sourceType = object.sourceType ?? "";
    message.version = object.version ?? "";
    message.compilerBinaryHash = object.compilerBinaryHash ?? new Uint8Array();
    message.axisBounds = object.axisBounds !== void 0 && object.axisBounds !== null ? AxisBounds.fromPartial(object.axisBounds) : void 0;
    message.minAttestationBondUzrn = object.minAttestationBondUzrn ?? "";
    message.minPerClaimBondUzrn = object.minPerClaimBondUzrn ?? "";
    message.slashGradient = object.slashGradient !== void 0 && object.slashGradient !== null ? SlashGradient.fromPartial(object.slashGradient) : void 0;
    message.requiredQualificationDomain = object.requiredQualificationDomain ?? "";
    message.minQualificationStatus = object.minQualificationStatus ?? 0;
    message.allowedClassIds = object.allowedClassIds?.map((e) => e) || [];
    message.status = object.status ?? 0;
    message.registeredViaLipId = object.registeredViaLipId ?? "";
    message.registeredAtBlock = object.registeredAtBlock !== void 0 && object.registeredAtBlock !== null ? BigInt(object.registeredAtBlock.toString()) : BigInt(0);
    message.tombstonedAtBlock = object.tombstonedAtBlock !== void 0 && object.tombstonedAtBlock !== null ? BigInt(object.tombstonedAtBlock.toString()) : BigInt(0);
    message.witnessRewardUzrn = object.witnessRewardUzrn ?? "";
    return message;
  }
};

// src/generated/zerone/substrate_bridge/v1/substrate_link.ts
function createBaseSubstrateLink() {
  return {
    citedFacts: [],
    pendingClaims: [],
    recursionWeight: void 0,
    adapterId: "",
    source: void 0,
    linkHash: new Uint8Array()
  };
}
var SubstrateLink = {
  typeUrl: "/zerone.substrate_bridge.v1.SubstrateLink",
  encode(message, writer = BinaryWriter.create()) {
    for (const v of message.citedFacts) {
      FactCitation.encode(v, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.pendingClaims) {
      PendingClaim.encode(v, writer.uint32(18).fork()).ldelim();
    }
    if (message.recursionWeight !== void 0) {
      AxisProjection.encode(message.recursionWeight, writer.uint32(26).fork()).ldelim();
    }
    if (message.adapterId !== "") {
      writer.uint32(34).string(message.adapterId);
    }
    if (message.source !== void 0) {
      ExternalSource.encode(message.source, writer.uint32(42).fork()).ldelim();
    }
    if (message.linkHash.length !== 0) {
      writer.uint32(50).bytes(message.linkHash);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseSubstrateLink();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.citedFacts.push(FactCitation.decode(reader, reader.uint32()));
          break;
        case 2:
          message.pendingClaims.push(PendingClaim.decode(reader, reader.uint32()));
          break;
        case 3:
          message.recursionWeight = AxisProjection.decode(reader, reader.uint32());
          break;
        case 4:
          message.adapterId = reader.string();
          break;
        case 5:
          message.source = ExternalSource.decode(reader, reader.uint32());
          break;
        case 6:
          message.linkHash = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseSubstrateLink();
    message.citedFacts = object.citedFacts?.map((e) => FactCitation.fromPartial(e)) || [];
    message.pendingClaims = object.pendingClaims?.map((e) => PendingClaim.fromPartial(e)) || [];
    message.recursionWeight = object.recursionWeight !== void 0 && object.recursionWeight !== null ? AxisProjection.fromPartial(object.recursionWeight) : void 0;
    message.adapterId = object.adapterId ?? "";
    message.source = object.source !== void 0 && object.source !== null ? ExternalSource.fromPartial(object.source) : void 0;
    message.linkHash = object.linkHash ?? new Uint8Array();
    return message;
  }
};
function createBaseFactCitation() {
  return {
    factId: "",
    citationType: 0,
    citationContext: ""
  };
}
var FactCitation = {
  typeUrl: "/zerone.substrate_bridge.v1.FactCitation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.factId !== "") {
      writer.uint32(10).string(message.factId);
    }
    if (message.citationType !== 0) {
      writer.uint32(16).int32(message.citationType);
    }
    if (message.citationContext !== "") {
      writer.uint32(26).string(message.citationContext);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseFactCitation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.factId = reader.string();
          break;
        case 2:
          message.citationType = reader.int32();
          break;
        case 3:
          message.citationContext = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseFactCitation();
    message.factId = object.factId ?? "";
    message.citationType = object.citationType ?? 0;
    message.citationContext = object.citationContext ?? "";
    return message;
  }
};
function createBasePendingClaim() {
  return {
    claimContent: "",
    proposedFactId: "",
    domain: "",
    methodologyId: "",
    relations: []
  };
}
var PendingClaim = {
  typeUrl: "/zerone.substrate_bridge.v1.PendingClaim",
  encode(message, writer = BinaryWriter.create()) {
    if (message.claimContent !== "") {
      writer.uint32(10).string(message.claimContent);
    }
    if (message.proposedFactId !== "") {
      writer.uint32(18).string(message.proposedFactId);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    if (message.methodologyId !== "") {
      writer.uint32(34).string(message.methodologyId);
    }
    for (const v of message.relations) {
      ClaimRelation2.encode(v, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBasePendingClaim();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimContent = reader.string();
          break;
        case 2:
          message.proposedFactId = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.methodologyId = reader.string();
          break;
        case 5:
          message.relations.push(ClaimRelation2.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBasePendingClaim();
    message.claimContent = object.claimContent ?? "";
    message.proposedFactId = object.proposedFactId ?? "";
    message.domain = object.domain ?? "";
    message.methodologyId = object.methodologyId ?? "";
    message.relations = object.relations?.map((e) => ClaimRelation2.fromPartial(e)) || [];
    return message;
  }
};
function createBaseClaimRelation2() {
  return {
    targetFactId: "",
    relation: "",
    inference: "",
    inferenceStrengthBps: 0
  };
}
var ClaimRelation2 = {
  typeUrl: "/zerone.substrate_bridge.v1.ClaimRelation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.targetFactId !== "") {
      writer.uint32(10).string(message.targetFactId);
    }
    if (message.relation !== "") {
      writer.uint32(18).string(message.relation);
    }
    if (message.inference !== "") {
      writer.uint32(26).string(message.inference);
    }
    if (message.inferenceStrengthBps !== 0) {
      writer.uint32(32).uint32(message.inferenceStrengthBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseClaimRelation2();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.targetFactId = reader.string();
          break;
        case 2:
          message.relation = reader.string();
          break;
        case 3:
          message.inference = reader.string();
          break;
        case 4:
          message.inferenceStrengthBps = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseClaimRelation2();
    message.targetFactId = object.targetFactId ?? "";
    message.relation = object.relation ?? "";
    message.inference = object.inference ?? "";
    message.inferenceStrengthBps = object.inferenceStrengthBps ?? 0;
    return message;
  }
};

// src/generated/zerone/substrate_bridge/v1/tx.ts
function createBaseMsgRegisterAdapter() {
  return {
    authority: "",
    adapter: void 0
  };
}
var MsgRegisterAdapter = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.adapter !== void 0) {
      AdapterRegistration.encode(message.adapter, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAdapter();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.adapter = AdapterRegistration.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgRegisterAdapter();
    message.authority = object.authority ?? "";
    message.adapter = object.adapter !== void 0 && object.adapter !== null ? AdapterRegistration.fromPartial(object.adapter) : void 0;
    return message;
  }
};
function createBaseMsgRegisterAdapterResponse() {
  return {};
}
var MsgRegisterAdapterResponse = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapterResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAdapterResponse();
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
    const message = createBaseMsgRegisterAdapterResponse();
    return message;
  }
};
function createBaseMsgSuspendAdapter() {
  return {
    authority: "",
    adapterId: "",
    reason: ""
  };
}
var MsgSuspendAdapter = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.adapterId !== "") {
      writer.uint32(18).string(message.adapterId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSuspendAdapter();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.adapterId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSuspendAdapter();
    message.authority = object.authority ?? "";
    message.adapterId = object.adapterId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgSuspendAdapterResponse() {
  return {};
}
var MsgSuspendAdapterResponse = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapterResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSuspendAdapterResponse();
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
    const message = createBaseMsgSuspendAdapterResponse();
    return message;
  }
};
function createBaseMsgTombstoneAdapter() {
  return {
    authority: "",
    adapterId: "",
    reason: ""
  };
}
var MsgTombstoneAdapter = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapter",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.adapterId !== "") {
      writer.uint32(18).string(message.adapterId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgTombstoneAdapter();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.adapterId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgTombstoneAdapter();
    message.authority = object.authority ?? "";
    message.adapterId = object.adapterId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgTombstoneAdapterResponse() {
  return {};
}
var MsgTombstoneAdapterResponse = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapterResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgTombstoneAdapterResponse();
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
    const message = createBaseMsgTombstoneAdapterResponse();
    return message;
  }
};
function createBaseMsgSubmitExternalAttestation() {
  return {
    submitter: "",
    adapterId: "",
    workClassId: "",
    link: void 0,
    bondUzrn: ""
  };
}
var MsgSubmitExternalAttestation = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
  encode(message, writer = BinaryWriter.create()) {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.adapterId !== "") {
      writer.uint32(18).string(message.adapterId);
    }
    if (message.workClassId !== "") {
      writer.uint32(26).string(message.workClassId);
    }
    if (message.link !== void 0) {
      SubstrateLink.encode(message.link, writer.uint32(34).fork()).ldelim();
    }
    if (message.bondUzrn !== "") {
      writer.uint32(42).string(message.bondUzrn);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitExternalAttestation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.adapterId = reader.string();
          break;
        case 3:
          message.workClassId = reader.string();
          break;
        case 4:
          message.link = SubstrateLink.decode(reader, reader.uint32());
          break;
        case 5:
          message.bondUzrn = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitExternalAttestation();
    message.submitter = object.submitter ?? "";
    message.adapterId = object.adapterId ?? "";
    message.workClassId = object.workClassId ?? "";
    message.link = object.link !== void 0 && object.link !== null ? SubstrateLink.fromPartial(object.link) : void 0;
    message.bondUzrn = object.bondUzrn ?? "";
    return message;
  }
};
function createBaseMsgSubmitExternalAttestationResponse() {
  return {
    attestationId: ""
  };
}
var MsgSubmitExternalAttestationResponse = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestationResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.attestationId !== "") {
      writer.uint32(10).string(message.attestationId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitExternalAttestationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.attestationId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgSubmitExternalAttestationResponse();
    message.attestationId = object.attestationId ?? "";
    return message;
  }
};

// src/generated/zerone/substrate_bridge/v1/tx.registry.ts
var registry18 = [["/zerone.substrate_bridge.v1.MsgRegisterAdapter", MsgRegisterAdapter], ["/zerone.substrate_bridge.v1.MsgSuspendAdapter", MsgSuspendAdapter], ["/zerone.substrate_bridge.v1.MsgTombstoneAdapter", MsgTombstoneAdapter], ["/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation", MsgSubmitExternalAttestation]];
var MessageComposer18 = {
  encoded: {
    registerAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
        value: MsgRegisterAdapter.encode(value).finish()
      };
    },
    suspendAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
        value: MsgSuspendAdapter.encode(value).finish()
      };
    },
    tombstoneAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapter",
        value: MsgTombstoneAdapter.encode(value).finish()
      };
    },
    submitExternalAttestation(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
        value: MsgSubmitExternalAttestation.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    registerAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
        value
      };
    },
    suspendAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
        value
      };
    },
    tombstoneAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapter",
        value
      };
    },
    submitExternalAttestation(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
        value
      };
    }
  },
  fromPartial: {
    registerAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
        value: MsgRegisterAdapter.fromPartial(value)
      };
    },
    suspendAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
        value: MsgSuspendAdapter.fromPartial(value)
      };
    },
    tombstoneAdapter(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapter",
        value: MsgTombstoneAdapter.fromPartial(value)
      };
    },
    submitExternalAttestation(value) {
      return {
        typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
        value: MsgSubmitExternalAttestation.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/tokens/v1/tx.ts
var tx_exports19 = {};
__export(tx_exports19, {
  MsgApproveToken: () => MsgApproveToken,
  MsgApproveTokenResponse: () => MsgApproveTokenResponse,
  MsgBurnToken: () => MsgBurnToken,
  MsgBurnTokenResponse: () => MsgBurnTokenResponse,
  MsgCancelEmissionPeriod: () => MsgCancelEmissionPeriod,
  MsgCancelEmissionPeriodResponse: () => MsgCancelEmissionPeriodResponse,
  MsgCreateEmissionPeriod: () => MsgCreateEmissionPeriod,
  MsgCreateEmissionPeriodResponse: () => MsgCreateEmissionPeriodResponse,
  MsgCreateToken: () => MsgCreateToken,
  MsgCreateTokenResponse: () => MsgCreateTokenResponse,
  MsgDelegatePower: () => MsgDelegatePower,
  MsgDelegatePowerResponse: () => MsgDelegatePowerResponse,
  MsgMintToken: () => MsgMintToken,
  MsgMintTokenResponse: () => MsgMintTokenResponse,
  MsgPauseToken: () => MsgPauseToken,
  MsgPauseTokenResponse: () => MsgPauseTokenResponse,
  MsgTransferFrom: () => MsgTransferFrom,
  MsgTransferFromResponse: () => MsgTransferFromResponse,
  MsgTransferToken: () => MsgTransferToken,
  MsgTransferTokenResponse: () => MsgTransferTokenResponse,
  MsgUndelegatePower: () => MsgUndelegatePower,
  MsgUndelegatePowerResponse: () => MsgUndelegatePowerResponse,
  MsgUnpauseToken: () => MsgUnpauseToken,
  MsgUnpauseTokenResponse: () => MsgUnpauseTokenResponse,
  MsgUnwrapToken: () => MsgUnwrapToken,
  MsgUnwrapTokenResponse: () => MsgUnwrapTokenResponse,
  MsgUpdateParams: () => MsgUpdateParams16,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse16,
  MsgWrapToken: () => MsgWrapToken,
  MsgWrapTokenResponse: () => MsgWrapTokenResponse
});

// src/generated/zerone/tokens/v1/types.ts
function createBaseTokenFeatures() {
  return {
    mintable: false,
    burnable: false,
    pausable: false,
    wrappable: false
  };
}
var TokenFeatures = {
  typeUrl: "/zerone.tokens.v1.TokenFeatures",
  encode(message, writer = BinaryWriter.create()) {
    if (message.mintable === true) {
      writer.uint32(8).bool(message.mintable);
    }
    if (message.burnable === true) {
      writer.uint32(16).bool(message.burnable);
    }
    if (message.pausable === true) {
      writer.uint32(24).bool(message.pausable);
    }
    if (message.wrappable === true) {
      writer.uint32(32).bool(message.wrappable);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseTokenFeatures();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.mintable = reader.bool();
          break;
        case 2:
          message.burnable = reader.bool();
          break;
        case 3:
          message.pausable = reader.bool();
          break;
        case 4:
          message.wrappable = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseTokenFeatures();
    message.mintable = object.mintable ?? false;
    message.burnable = object.burnable ?? false;
    message.pausable = object.pausable ?? false;
    message.wrappable = object.wrappable ?? false;
    return message;
  }
};

// src/generated/zerone/tokens/v1/genesis.ts
function createBaseParams17() {
  return {
    emissionEpochBlocks: BigInt(0),
    defaultFeeBps: ""
  };
}
var Params17 = {
  typeUrl: "/zerone.tokens.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.emissionEpochBlocks !== BigInt(0)) {
      writer.uint32(8).uint64(message.emissionEpochBlocks);
    }
    if (message.defaultFeeBps !== "") {
      writer.uint32(18).string(message.defaultFeeBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams17();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.emissionEpochBlocks = reader.uint64();
          break;
        case 2:
          message.defaultFeeBps = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams17();
    message.emissionEpochBlocks = object.emissionEpochBlocks !== void 0 && object.emissionEpochBlocks !== null ? BigInt(object.emissionEpochBlocks.toString()) : BigInt(0);
    message.defaultFeeBps = object.defaultFeeBps ?? "";
    return message;
  }
};

// src/generated/zerone/tokens/v1/tx.ts
function createBaseMsgCreateToken() {
  return {
    creator: "",
    name: "",
    symbol: "",
    decimals: 0,
    initialSupply: "",
    maxSupply: "",
    features: void 0
  };
}
var MsgCreateToken = {
  typeUrl: "/zerone.tokens.v1.MsgCreateToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.symbol !== "") {
      writer.uint32(26).string(message.symbol);
    }
    if (message.decimals !== 0) {
      writer.uint32(32).uint32(message.decimals);
    }
    if (message.initialSupply !== "") {
      writer.uint32(42).string(message.initialSupply);
    }
    if (message.maxSupply !== "") {
      writer.uint32(50).string(message.maxSupply);
    }
    if (message.features !== void 0) {
      TokenFeatures.encode(message.features, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.symbol = reader.string();
          break;
        case 4:
          message.decimals = reader.uint32();
          break;
        case 5:
          message.initialSupply = reader.string();
          break;
        case 6:
          message.maxSupply = reader.string();
          break;
        case 7:
          message.features = TokenFeatures.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateToken();
    message.creator = object.creator ?? "";
    message.name = object.name ?? "";
    message.symbol = object.symbol ?? "";
    message.decimals = object.decimals ?? 0;
    message.initialSupply = object.initialSupply ?? "";
    message.maxSupply = object.maxSupply ?? "";
    message.features = object.features !== void 0 && object.features !== null ? TokenFeatures.fromPartial(object.features) : void 0;
    return message;
  }
};
function createBaseMsgCreateTokenResponse() {
  return {
    tokenId: ""
  };
}
var MsgCreateTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgCreateTokenResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.tokenId !== "") {
      writer.uint32(10).string(message.tokenId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateTokenResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateTokenResponse();
    message.tokenId = object.tokenId ?? "";
    return message;
  }
};
function createBaseMsgMintToken() {
  return {
    authority: "",
    tokenId: "",
    to: "",
    amount: ""
  };
}
var MsgMintToken = {
  typeUrl: "/zerone.tokens.v1.MsgMintToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.to !== "") {
      writer.uint32(26).string(message.to);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgMintToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.to = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgMintToken();
    message.authority = object.authority ?? "";
    message.tokenId = object.tokenId ?? "";
    message.to = object.to ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgMintTokenResponse() {
  return {};
}
var MsgMintTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgMintTokenResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgMintTokenResponse();
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
    const message = createBaseMsgMintTokenResponse();
    return message;
  }
};
function createBaseMsgBurnToken() {
  return {
    burner: "",
    tokenId: "",
    amount: ""
  };
}
var MsgBurnToken = {
  typeUrl: "/zerone.tokens.v1.MsgBurnToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.burner !== "") {
      writer.uint32(10).string(message.burner);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgBurnToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.burner = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgBurnToken();
    message.burner = object.burner ?? "";
    message.tokenId = object.tokenId ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgBurnTokenResponse() {
  return {};
}
var MsgBurnTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgBurnTokenResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgBurnTokenResponse();
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
    const message = createBaseMsgBurnTokenResponse();
    return message;
  }
};
function createBaseMsgTransferToken() {
  return {
    sender: "",
    tokenId: "",
    to: "",
    amount: ""
  };
}
var MsgTransferToken = {
  typeUrl: "/zerone.tokens.v1.MsgTransferToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.to !== "") {
      writer.uint32(26).string(message.to);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgTransferToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.to = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgTransferToken();
    message.sender = object.sender ?? "";
    message.tokenId = object.tokenId ?? "";
    message.to = object.to ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgTransferTokenResponse() {
  return {};
}
var MsgTransferTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgTransferTokenResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgTransferTokenResponse();
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
    const message = createBaseMsgTransferTokenResponse();
    return message;
  }
};
function createBaseMsgApproveToken() {
  return {
    owner: "",
    tokenId: "",
    spender: "",
    amount: ""
  };
}
var MsgApproveToken = {
  typeUrl: "/zerone.tokens.v1.MsgApproveToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.spender !== "") {
      writer.uint32(26).string(message.spender);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgApproveToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.spender = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgApproveToken();
    message.owner = object.owner ?? "";
    message.tokenId = object.tokenId ?? "";
    message.spender = object.spender ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgApproveTokenResponse() {
  return {};
}
var MsgApproveTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgApproveTokenResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgApproveTokenResponse();
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
    const message = createBaseMsgApproveTokenResponse();
    return message;
  }
};
function createBaseMsgTransferFrom() {
  return {
    spender: "",
    tokenId: "",
    from: "",
    to: "",
    amount: ""
  };
}
var MsgTransferFrom = {
  typeUrl: "/zerone.tokens.v1.MsgTransferFrom",
  encode(message, writer = BinaryWriter.create()) {
    if (message.spender !== "") {
      writer.uint32(10).string(message.spender);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.from !== "") {
      writer.uint32(26).string(message.from);
    }
    if (message.to !== "") {
      writer.uint32(34).string(message.to);
    }
    if (message.amount !== "") {
      writer.uint32(42).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgTransferFrom();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.spender = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.from = reader.string();
          break;
        case 4:
          message.to = reader.string();
          break;
        case 5:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgTransferFrom();
    message.spender = object.spender ?? "";
    message.tokenId = object.tokenId ?? "";
    message.from = object.from ?? "";
    message.to = object.to ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgTransferFromResponse() {
  return {};
}
var MsgTransferFromResponse = {
  typeUrl: "/zerone.tokens.v1.MsgTransferFromResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgTransferFromResponse();
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
    const message = createBaseMsgTransferFromResponse();
    return message;
  }
};
function createBaseMsgPauseToken() {
  return {
    authority: "",
    tokenId: ""
  };
}
var MsgPauseToken = {
  typeUrl: "/zerone.tokens.v1.MsgPauseToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgPauseToken();
    message.authority = object.authority ?? "";
    message.tokenId = object.tokenId ?? "";
    return message;
  }
};
function createBaseMsgPauseTokenResponse() {
  return {};
}
var MsgPauseTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgPauseTokenResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseTokenResponse();
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
    const message = createBaseMsgPauseTokenResponse();
    return message;
  }
};
function createBaseMsgUnpauseToken() {
  return {
    authority: "",
    tokenId: ""
  };
}
var MsgUnpauseToken = {
  typeUrl: "/zerone.tokens.v1.MsgUnpauseToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUnpauseToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUnpauseToken();
    message.authority = object.authority ?? "";
    message.tokenId = object.tokenId ?? "";
    return message;
  }
};
function createBaseMsgUnpauseTokenResponse() {
  return {};
}
var MsgUnpauseTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgUnpauseTokenResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUnpauseTokenResponse();
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
    const message = createBaseMsgUnpauseTokenResponse();
    return message;
  }
};
function createBaseMsgDelegatePower() {
  return {
    delegator: "",
    tokenId: "",
    delegate: "",
    amount: ""
  };
}
var MsgDelegatePower = {
  typeUrl: "/zerone.tokens.v1.MsgDelegatePower",
  encode(message, writer = BinaryWriter.create()) {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.delegate !== "") {
      writer.uint32(26).string(message.delegate);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgDelegatePower();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.delegate = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgDelegatePower();
    message.delegator = object.delegator ?? "";
    message.tokenId = object.tokenId ?? "";
    message.delegate = object.delegate ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgDelegatePowerResponse() {
  return {};
}
var MsgDelegatePowerResponse = {
  typeUrl: "/zerone.tokens.v1.MsgDelegatePowerResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgDelegatePowerResponse();
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
    const message = createBaseMsgDelegatePowerResponse();
    return message;
  }
};
function createBaseMsgUndelegatePower() {
  return {
    delegator: "",
    tokenId: "",
    delegate: "",
    amount: ""
  };
}
var MsgUndelegatePower = {
  typeUrl: "/zerone.tokens.v1.MsgUndelegatePower",
  encode(message, writer = BinaryWriter.create()) {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.delegate !== "") {
      writer.uint32(26).string(message.delegate);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUndelegatePower();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.delegate = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUndelegatePower();
    message.delegator = object.delegator ?? "";
    message.tokenId = object.tokenId ?? "";
    message.delegate = object.delegate ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgUndelegatePowerResponse() {
  return {};
}
var MsgUndelegatePowerResponse = {
  typeUrl: "/zerone.tokens.v1.MsgUndelegatePowerResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUndelegatePowerResponse();
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
    const message = createBaseMsgUndelegatePowerResponse();
    return message;
  }
};
function createBaseMsgWrapToken() {
  return {
    sender: "",
    tokenId: "",
    amount: ""
  };
}
var MsgWrapToken = {
  typeUrl: "/zerone.tokens.v1.MsgWrapToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgWrapToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgWrapToken();
    message.sender = object.sender ?? "";
    message.tokenId = object.tokenId ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgWrapTokenResponse() {
  return {
    wrappedDenom: ""
  };
}
var MsgWrapTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgWrapTokenResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.wrappedDenom !== "") {
      writer.uint32(10).string(message.wrappedDenom);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgWrapTokenResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.wrappedDenom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgWrapTokenResponse();
    message.wrappedDenom = object.wrappedDenom ?? "";
    return message;
  }
};
function createBaseMsgUnwrapToken() {
  return {
    sender: "",
    wrappedDenom: "",
    amount: ""
  };
}
var MsgUnwrapToken = {
  typeUrl: "/zerone.tokens.v1.MsgUnwrapToken",
  encode(message, writer = BinaryWriter.create()) {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.wrappedDenom !== "") {
      writer.uint32(18).string(message.wrappedDenom);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUnwrapToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.wrappedDenom = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUnwrapToken();
    message.sender = object.sender ?? "";
    message.wrappedDenom = object.wrappedDenom ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgUnwrapTokenResponse() {
  return {
    tokenId: ""
  };
}
var MsgUnwrapTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgUnwrapTokenResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.tokenId !== "") {
      writer.uint32(10).string(message.tokenId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUnwrapTokenResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUnwrapTokenResponse();
    message.tokenId = object.tokenId ?? "";
    return message;
  }
};
function createBaseMsgCreateEmissionPeriod() {
  return {
    authority: "",
    startBlock: BigInt(0),
    endBlock: BigInt(0),
    amountPerBlock: "",
    recipient: ""
  };
}
var MsgCreateEmissionPeriod = {
  typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriod",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.startBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.startBlock);
    }
    if (message.endBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.endBlock);
    }
    if (message.amountPerBlock !== "") {
      writer.uint32(34).string(message.amountPerBlock);
    }
    if (message.recipient !== "") {
      writer.uint32(42).string(message.recipient);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateEmissionPeriod();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.startBlock = reader.uint64();
          break;
        case 3:
          message.endBlock = reader.uint64();
          break;
        case 4:
          message.amountPerBlock = reader.string();
          break;
        case 5:
          message.recipient = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateEmissionPeriod();
    message.authority = object.authority ?? "";
    message.startBlock = object.startBlock !== void 0 && object.startBlock !== null ? BigInt(object.startBlock.toString()) : BigInt(0);
    message.endBlock = object.endBlock !== void 0 && object.endBlock !== null ? BigInt(object.endBlock.toString()) : BigInt(0);
    message.amountPerBlock = object.amountPerBlock ?? "";
    message.recipient = object.recipient ?? "";
    return message;
  }
};
function createBaseMsgCreateEmissionPeriodResponse() {
  return {
    emissionId: ""
  };
}
var MsgCreateEmissionPeriodResponse = {
  typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriodResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.emissionId !== "") {
      writer.uint32(10).string(message.emissionId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateEmissionPeriodResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.emissionId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateEmissionPeriodResponse();
    message.emissionId = object.emissionId ?? "";
    return message;
  }
};
function createBaseMsgCancelEmissionPeriod() {
  return {
    authority: "",
    emissionId: ""
  };
}
var MsgCancelEmissionPeriod = {
  typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriod",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.emissionId !== "") {
      writer.uint32(18).string(message.emissionId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelEmissionPeriod();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.emissionId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCancelEmissionPeriod();
    message.authority = object.authority ?? "";
    message.emissionId = object.emissionId ?? "";
    return message;
  }
};
function createBaseMsgCancelEmissionPeriodResponse() {
  return {};
}
var MsgCancelEmissionPeriodResponse = {
  typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriodResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelEmissionPeriodResponse();
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
    const message = createBaseMsgCancelEmissionPeriodResponse();
    return message;
  }
};
function createBaseMsgUpdateParams16() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams16 = {
  typeUrl: "/zerone.tokens.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params17.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams16();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params17.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams16();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params17.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse16() {
  return {};
}
var MsgUpdateParamsResponse16 = {
  typeUrl: "/zerone.tokens.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse16();
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
    const message = createBaseMsgUpdateParamsResponse16();
    return message;
  }
};

// src/generated/zerone/tokens/v1/tx.registry.ts
var registry19 = [["/zerone.tokens.v1.MsgCreateToken", MsgCreateToken], ["/zerone.tokens.v1.MsgMintToken", MsgMintToken], ["/zerone.tokens.v1.MsgBurnToken", MsgBurnToken], ["/zerone.tokens.v1.MsgTransferToken", MsgTransferToken], ["/zerone.tokens.v1.MsgApproveToken", MsgApproveToken], ["/zerone.tokens.v1.MsgTransferFrom", MsgTransferFrom], ["/zerone.tokens.v1.MsgPauseToken", MsgPauseToken], ["/zerone.tokens.v1.MsgUnpauseToken", MsgUnpauseToken], ["/zerone.tokens.v1.MsgDelegatePower", MsgDelegatePower], ["/zerone.tokens.v1.MsgUndelegatePower", MsgUndelegatePower], ["/zerone.tokens.v1.MsgWrapToken", MsgWrapToken], ["/zerone.tokens.v1.MsgUnwrapToken", MsgUnwrapToken], ["/zerone.tokens.v1.MsgCreateEmissionPeriod", MsgCreateEmissionPeriod], ["/zerone.tokens.v1.MsgCancelEmissionPeriod", MsgCancelEmissionPeriod], ["/zerone.tokens.v1.MsgUpdateParams", MsgUpdateParams16]];
var MessageComposer19 = {
  encoded: {
    createToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateToken",
        value: MsgCreateToken.encode(value).finish()
      };
    },
    mintToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgMintToken",
        value: MsgMintToken.encode(value).finish()
      };
    },
    burnToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgBurnToken",
        value: MsgBurnToken.encode(value).finish()
      };
    },
    transferToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferToken",
        value: MsgTransferToken.encode(value).finish()
      };
    },
    approveToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgApproveToken",
        value: MsgApproveToken.encode(value).finish()
      };
    },
    transferFrom(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferFrom",
        value: MsgTransferFrom.encode(value).finish()
      };
    },
    pauseToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgPauseToken",
        value: MsgPauseToken.encode(value).finish()
      };
    },
    unpauseToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnpauseToken",
        value: MsgUnpauseToken.encode(value).finish()
      };
    },
    delegatePower(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgDelegatePower",
        value: MsgDelegatePower.encode(value).finish()
      };
    },
    undelegatePower(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUndelegatePower",
        value: MsgUndelegatePower.encode(value).finish()
      };
    },
    wrapToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgWrapToken",
        value: MsgWrapToken.encode(value).finish()
      };
    },
    unwrapToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnwrapToken",
        value: MsgUnwrapToken.encode(value).finish()
      };
    },
    createEmissionPeriod(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriod",
        value: MsgCreateEmissionPeriod.encode(value).finish()
      };
    },
    cancelEmissionPeriod(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriod",
        value: MsgCancelEmissionPeriod.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUpdateParams",
        value: MsgUpdateParams16.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateToken",
        value
      };
    },
    mintToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgMintToken",
        value
      };
    },
    burnToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgBurnToken",
        value
      };
    },
    transferToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferToken",
        value
      };
    },
    approveToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgApproveToken",
        value
      };
    },
    transferFrom(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferFrom",
        value
      };
    },
    pauseToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgPauseToken",
        value
      };
    },
    unpauseToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnpauseToken",
        value
      };
    },
    delegatePower(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgDelegatePower",
        value
      };
    },
    undelegatePower(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUndelegatePower",
        value
      };
    },
    wrapToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgWrapToken",
        value
      };
    },
    unwrapToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnwrapToken",
        value
      };
    },
    createEmissionPeriod(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriod",
        value
      };
    },
    cancelEmissionPeriod(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriod",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateToken",
        value: MsgCreateToken.fromPartial(value)
      };
    },
    mintToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgMintToken",
        value: MsgMintToken.fromPartial(value)
      };
    },
    burnToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgBurnToken",
        value: MsgBurnToken.fromPartial(value)
      };
    },
    transferToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferToken",
        value: MsgTransferToken.fromPartial(value)
      };
    },
    approveToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgApproveToken",
        value: MsgApproveToken.fromPartial(value)
      };
    },
    transferFrom(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgTransferFrom",
        value: MsgTransferFrom.fromPartial(value)
      };
    },
    pauseToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgPauseToken",
        value: MsgPauseToken.fromPartial(value)
      };
    },
    unpauseToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnpauseToken",
        value: MsgUnpauseToken.fromPartial(value)
      };
    },
    delegatePower(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgDelegatePower",
        value: MsgDelegatePower.fromPartial(value)
      };
    },
    undelegatePower(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUndelegatePower",
        value: MsgUndelegatePower.fromPartial(value)
      };
    },
    wrapToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgWrapToken",
        value: MsgWrapToken.fromPartial(value)
      };
    },
    unwrapToken(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUnwrapToken",
        value: MsgUnwrapToken.fromPartial(value)
      };
    },
    createEmissionPeriod(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriod",
        value: MsgCreateEmissionPeriod.fromPartial(value)
      };
    },
    cancelEmissionPeriod(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriod",
        value: MsgCancelEmissionPeriod.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.tokens.v1.MsgUpdateParams",
        value: MsgUpdateParams16.fromPartial(value)
      };
    }
  }
};

// src/generated/zerone/vesting_rewards/v1/tx.ts
var tx_exports20 = {};
__export(tx_exports20, {
  MsgAccelerateVesting: () => MsgAccelerateVesting,
  MsgAccelerateVestingResponse: () => MsgAccelerateVestingResponse,
  MsgClaimVesting: () => MsgClaimVesting,
  MsgClaimVestingResponse: () => MsgClaimVestingResponse,
  MsgCompleteVesting: () => MsgCompleteVesting,
  MsgCompleteVestingResponse: () => MsgCompleteVestingResponse,
  MsgCreateVesting: () => MsgCreateVesting,
  MsgCreateVestingResponse: () => MsgCreateVestingResponse,
  MsgFalsifyVesting: () => MsgFalsifyVesting,
  MsgFalsifyVestingResponse: () => MsgFalsifyVestingResponse,
  MsgPauseVesting: () => MsgPauseVesting,
  MsgPauseVestingResponse: () => MsgPauseVestingResponse,
  MsgResumeVesting: () => MsgResumeVesting,
  MsgResumeVestingResponse: () => MsgResumeVestingResponse,
  MsgUpdateParams: () => MsgUpdateParams17,
  MsgUpdateParamsResponse: () => MsgUpdateParamsResponse17,
  VestingCategory: () => VestingCategory,
  vestingCategoryFromJSON: () => vestingCategoryFromJSON,
  vestingCategoryToJSON: () => vestingCategoryToJSON
});

// src/generated/zerone/common/v1/common.ts
function createBaseRevenueSplit() {
  return {
    contributorBps: BigInt(0),
    protocolBps: BigInt(0),
    researchBps: BigInt(0),
    developmentBps: BigInt(0)
  };
}
var RevenueSplit = {
  typeUrl: "/zerone.common.v1.RevenueSplit",
  encode(message, writer = BinaryWriter.create()) {
    if (message.contributorBps !== BigInt(0)) {
      writer.uint32(8).uint64(message.contributorBps);
    }
    if (message.protocolBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.protocolBps);
    }
    if (message.researchBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.researchBps);
    }
    if (message.developmentBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.developmentBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseRevenueSplit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.contributorBps = reader.uint64();
          break;
        case 2:
          message.protocolBps = reader.uint64();
          break;
        case 3:
          message.researchBps = reader.uint64();
          break;
        case 4:
          message.developmentBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseRevenueSplit();
    message.contributorBps = object.contributorBps !== void 0 && object.contributorBps !== null ? BigInt(object.contributorBps.toString()) : BigInt(0);
    message.protocolBps = object.protocolBps !== void 0 && object.protocolBps !== null ? BigInt(object.protocolBps.toString()) : BigInt(0);
    message.researchBps = object.researchBps !== void 0 && object.researchBps !== null ? BigInt(object.researchBps.toString()) : BigInt(0);
    message.developmentBps = object.developmentBps !== void 0 && object.developmentBps !== null ? BigInt(object.developmentBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseProtocolSubSplit() {
  return {
    citationBps: BigInt(0),
    verificationBps: BigInt(0),
    treasuryBps: BigInt(0)
  };
}
var ProtocolSubSplit = {
  typeUrl: "/zerone.common.v1.ProtocolSubSplit",
  encode(message, writer = BinaryWriter.create()) {
    if (message.citationBps !== BigInt(0)) {
      writer.uint32(8).uint64(message.citationBps);
    }
    if (message.verificationBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.verificationBps);
    }
    if (message.treasuryBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.treasuryBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseProtocolSubSplit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.citationBps = reader.uint64();
          break;
        case 2:
          message.verificationBps = reader.uint64();
          break;
        case 3:
          message.treasuryBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseProtocolSubSplit();
    message.citationBps = object.citationBps !== void 0 && object.citationBps !== null ? BigInt(object.citationBps.toString()) : BigInt(0);
    message.verificationBps = object.verificationBps !== void 0 && object.verificationBps !== null ? BigInt(object.verificationBps.toString()) : BigInt(0);
    message.treasuryBps = object.treasuryBps !== void 0 && object.treasuryBps !== null ? BigInt(object.treasuryBps.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/vesting_rewards/v1/genesis.ts
function createBaseParams18() {
  return {
    blockReward: "",
    rewardDecayBps: BigInt(0),
    blocksPerRewardEpoch: BigInt(0),
    revenueSplit: void 0,
    protocolSubSplit: void 0,
    founderShareBps: BigInt(0),
    founderAddress: "",
    governanceActivationHeight: BigInt(0),
    categoryRewardConfigs: [],
    researchFundModuleAccount: "",
    vestingEnabled: false,
    releasedClawbackRate: BigInt(0),
    minValidatorsForFullReward: 0,
    emptyBlockRewardRate: BigInt(0),
    floorReward: "",
    initialFundBalance: "",
    knowledgeCouplingTargetBps: BigInt(0),
    knowledgeCouplingFloorBps: BigInt(0)
  };
}
var Params18 = {
  typeUrl: "/zerone.vesting_rewards.v1.Params",
  encode(message, writer = BinaryWriter.create()) {
    if (message.blockReward !== "") {
      writer.uint32(10).string(message.blockReward);
    }
    if (message.rewardDecayBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.rewardDecayBps);
    }
    if (message.blocksPerRewardEpoch !== BigInt(0)) {
      writer.uint32(24).uint64(message.blocksPerRewardEpoch);
    }
    if (message.revenueSplit !== void 0) {
      RevenueSplit.encode(message.revenueSplit, writer.uint32(34).fork()).ldelim();
    }
    if (message.protocolSubSplit !== void 0) {
      ProtocolSubSplit.encode(message.protocolSubSplit, writer.uint32(42).fork()).ldelim();
    }
    if (message.founderShareBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.founderShareBps);
    }
    if (message.founderAddress !== "") {
      writer.uint32(58).string(message.founderAddress);
    }
    if (message.governanceActivationHeight !== BigInt(0)) {
      writer.uint32(64).uint64(message.governanceActivationHeight);
    }
    for (const v of message.categoryRewardConfigs) {
      CategoryRewardConfig.encode(v, writer.uint32(74).fork()).ldelim();
    }
    if (message.researchFundModuleAccount !== "") {
      writer.uint32(82).string(message.researchFundModuleAccount);
    }
    if (message.vestingEnabled === true) {
      writer.uint32(88).bool(message.vestingEnabled);
    }
    if (message.releasedClawbackRate !== BigInt(0)) {
      writer.uint32(96).uint64(message.releasedClawbackRate);
    }
    if (message.minValidatorsForFullReward !== 0) {
      writer.uint32(104).uint32(message.minValidatorsForFullReward);
    }
    if (message.emptyBlockRewardRate !== BigInt(0)) {
      writer.uint32(112).uint64(message.emptyBlockRewardRate);
    }
    if (message.floorReward !== "") {
      writer.uint32(122).string(message.floorReward);
    }
    if (message.initialFundBalance !== "") {
      writer.uint32(130).string(message.initialFundBalance);
    }
    if (message.knowledgeCouplingTargetBps !== BigInt(0)) {
      writer.uint32(136).uint64(message.knowledgeCouplingTargetBps);
    }
    if (message.knowledgeCouplingFloorBps !== BigInt(0)) {
      writer.uint32(144).uint64(message.knowledgeCouplingFloorBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseParams18();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.blockReward = reader.string();
          break;
        case 2:
          message.rewardDecayBps = reader.uint64();
          break;
        case 3:
          message.blocksPerRewardEpoch = reader.uint64();
          break;
        case 4:
          message.revenueSplit = RevenueSplit.decode(reader, reader.uint32());
          break;
        case 5:
          message.protocolSubSplit = ProtocolSubSplit.decode(reader, reader.uint32());
          break;
        case 6:
          message.founderShareBps = reader.uint64();
          break;
        case 7:
          message.founderAddress = reader.string();
          break;
        case 8:
          message.governanceActivationHeight = reader.uint64();
          break;
        case 9:
          message.categoryRewardConfigs.push(CategoryRewardConfig.decode(reader, reader.uint32()));
          break;
        case 10:
          message.researchFundModuleAccount = reader.string();
          break;
        case 11:
          message.vestingEnabled = reader.bool();
          break;
        case 12:
          message.releasedClawbackRate = reader.uint64();
          break;
        case 13:
          message.minValidatorsForFullReward = reader.uint32();
          break;
        case 14:
          message.emptyBlockRewardRate = reader.uint64();
          break;
        case 15:
          message.floorReward = reader.string();
          break;
        case 16:
          message.initialFundBalance = reader.string();
          break;
        case 17:
          message.knowledgeCouplingTargetBps = reader.uint64();
          break;
        case 18:
          message.knowledgeCouplingFloorBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseParams18();
    message.blockReward = object.blockReward ?? "";
    message.rewardDecayBps = object.rewardDecayBps !== void 0 && object.rewardDecayBps !== null ? BigInt(object.rewardDecayBps.toString()) : BigInt(0);
    message.blocksPerRewardEpoch = object.blocksPerRewardEpoch !== void 0 && object.blocksPerRewardEpoch !== null ? BigInt(object.blocksPerRewardEpoch.toString()) : BigInt(0);
    message.revenueSplit = object.revenueSplit !== void 0 && object.revenueSplit !== null ? RevenueSplit.fromPartial(object.revenueSplit) : void 0;
    message.protocolSubSplit = object.protocolSubSplit !== void 0 && object.protocolSubSplit !== null ? ProtocolSubSplit.fromPartial(object.protocolSubSplit) : void 0;
    message.founderShareBps = object.founderShareBps !== void 0 && object.founderShareBps !== null ? BigInt(object.founderShareBps.toString()) : BigInt(0);
    message.founderAddress = object.founderAddress ?? "";
    message.governanceActivationHeight = object.governanceActivationHeight !== void 0 && object.governanceActivationHeight !== null ? BigInt(object.governanceActivationHeight.toString()) : BigInt(0);
    message.categoryRewardConfigs = object.categoryRewardConfigs?.map((e) => CategoryRewardConfig.fromPartial(e)) || [];
    message.researchFundModuleAccount = object.researchFundModuleAccount ?? "";
    message.vestingEnabled = object.vestingEnabled ?? false;
    message.releasedClawbackRate = object.releasedClawbackRate !== void 0 && object.releasedClawbackRate !== null ? BigInt(object.releasedClawbackRate.toString()) : BigInt(0);
    message.minValidatorsForFullReward = object.minValidatorsForFullReward ?? 0;
    message.emptyBlockRewardRate = object.emptyBlockRewardRate !== void 0 && object.emptyBlockRewardRate !== null ? BigInt(object.emptyBlockRewardRate.toString()) : BigInt(0);
    message.floorReward = object.floorReward ?? "";
    message.initialFundBalance = object.initialFundBalance ?? "";
    message.knowledgeCouplingTargetBps = object.knowledgeCouplingTargetBps !== void 0 && object.knowledgeCouplingTargetBps !== null ? BigInt(object.knowledgeCouplingTargetBps.toString()) : BigInt(0);
    message.knowledgeCouplingFloorBps = object.knowledgeCouplingFloorBps !== void 0 && object.knowledgeCouplingFloorBps !== null ? BigInt(object.knowledgeCouplingFloorBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCategoryRewardConfig() {
  return {
    category: "",
    multiplierBps: BigInt(0)
  };
}
var CategoryRewardConfig = {
  typeUrl: "/zerone.vesting_rewards.v1.CategoryRewardConfig",
  encode(message, writer = BinaryWriter.create()) {
    if (message.category !== "") {
      writer.uint32(10).string(message.category);
    }
    if (message.multiplierBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.multiplierBps);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseCategoryRewardConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.category = reader.string();
          break;
        case 2:
          message.multiplierBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseCategoryRewardConfig();
    message.category = object.category ?? "";
    message.multiplierBps = object.multiplierBps !== void 0 && object.multiplierBps !== null ? BigInt(object.multiplierBps.toString()) : BigInt(0);
    return message;
  }
};

// src/generated/zerone/vesting_rewards/v1/tx.ts
var VestingCategory = /* @__PURE__ */ ((VestingCategory2) => {
  VestingCategory2[VestingCategory2["VESTING_CATEGORY_UNSPECIFIED"] = 0] = "VESTING_CATEGORY_UNSPECIFIED";
  VestingCategory2[VestingCategory2["VESTING_CATEGORY_VERIFICATION_REWARD"] = 1] = "VESTING_CATEGORY_VERIFICATION_REWARD";
  VestingCategory2[VestingCategory2["VESTING_CATEGORY_BLOCK_REWARD"] = 2] = "VESTING_CATEGORY_BLOCK_REWARD";
  VestingCategory2[VestingCategory2["VESTING_CATEGORY_BOUNTY_REWARD"] = 3] = "VESTING_CATEGORY_BOUNTY_REWARD";
  VestingCategory2[VestingCategory2["VESTING_CATEGORY_DISPUTE_REWARD"] = 4] = "VESTING_CATEGORY_DISPUTE_REWARD";
  VestingCategory2[VestingCategory2["VESTING_CATEGORY_RESEARCH_GRANT"] = 5] = "VESTING_CATEGORY_RESEARCH_GRANT";
  VestingCategory2[VestingCategory2["VESTING_CATEGORY_BOOTSTRAP"] = 6] = "VESTING_CATEGORY_BOOTSTRAP";
  VestingCategory2[VestingCategory2["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
  return VestingCategory2;
})(VestingCategory || {});
function vestingCategoryFromJSON(object) {
  switch (object) {
    case 0:
    case "VESTING_CATEGORY_UNSPECIFIED":
      return 0 /* VESTING_CATEGORY_UNSPECIFIED */;
    case 1:
    case "VESTING_CATEGORY_VERIFICATION_REWARD":
      return 1 /* VESTING_CATEGORY_VERIFICATION_REWARD */;
    case 2:
    case "VESTING_CATEGORY_BLOCK_REWARD":
      return 2 /* VESTING_CATEGORY_BLOCK_REWARD */;
    case 3:
    case "VESTING_CATEGORY_BOUNTY_REWARD":
      return 3 /* VESTING_CATEGORY_BOUNTY_REWARD */;
    case 4:
    case "VESTING_CATEGORY_DISPUTE_REWARD":
      return 4 /* VESTING_CATEGORY_DISPUTE_REWARD */;
    case 5:
    case "VESTING_CATEGORY_RESEARCH_GRANT":
      return 5 /* VESTING_CATEGORY_RESEARCH_GRANT */;
    case 6:
    case "VESTING_CATEGORY_BOOTSTRAP":
      return 6 /* VESTING_CATEGORY_BOOTSTRAP */;
    case -1:
    case "UNRECOGNIZED":
    default:
      return -1 /* UNRECOGNIZED */;
  }
}
function vestingCategoryToJSON(object) {
  switch (object) {
    case 0 /* VESTING_CATEGORY_UNSPECIFIED */:
      return "VESTING_CATEGORY_UNSPECIFIED";
    case 1 /* VESTING_CATEGORY_VERIFICATION_REWARD */:
      return "VESTING_CATEGORY_VERIFICATION_REWARD";
    case 2 /* VESTING_CATEGORY_BLOCK_REWARD */:
      return "VESTING_CATEGORY_BLOCK_REWARD";
    case 3 /* VESTING_CATEGORY_BOUNTY_REWARD */:
      return "VESTING_CATEGORY_BOUNTY_REWARD";
    case 4 /* VESTING_CATEGORY_DISPUTE_REWARD */:
      return "VESTING_CATEGORY_DISPUTE_REWARD";
    case 5 /* VESTING_CATEGORY_RESEARCH_GRANT */:
      return "VESTING_CATEGORY_RESEARCH_GRANT";
    case 6 /* VESTING_CATEGORY_BOOTSTRAP */:
      return "VESTING_CATEGORY_BOOTSTRAP";
    case -1 /* UNRECOGNIZED */:
    default:
      return "UNRECOGNIZED";
  }
}
function createBaseMsgCreateVesting() {
  return {
    authority: "",
    beneficiary: "",
    amount: "",
    category: 0,
    linkedFactId: "",
    startHeight: BigInt(0)
  };
}
var MsgCreateVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVesting",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.beneficiary !== "") {
      writer.uint32(18).string(message.beneficiary);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.category !== 0) {
      writer.uint32(32).int32(message.category);
    }
    if (message.linkedFactId !== "") {
      writer.uint32(42).string(message.linkedFactId);
    }
    if (message.startHeight !== BigInt(0)) {
      writer.uint32(48).uint64(message.startHeight);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.beneficiary = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.category = reader.int32();
          break;
        case 5:
          message.linkedFactId = reader.string();
          break;
        case 6:
          message.startHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateVesting();
    message.authority = object.authority ?? "";
    message.beneficiary = object.beneficiary ?? "";
    message.amount = object.amount ?? "";
    message.category = object.category ?? 0;
    message.linkedFactId = object.linkedFactId ?? "";
    message.startHeight = object.startHeight !== void 0 && object.startHeight !== null ? BigInt(object.startHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgCreateVestingResponse() {
  return {
    vestingId: ""
  };
}
var MsgCreateVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVestingResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.vestingId !== "") {
      writer.uint32(10).string(message.vestingId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateVestingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.vestingId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCreateVestingResponse();
    message.vestingId = object.vestingId ?? "";
    return message;
  }
};
function createBaseMsgClaimVesting() {
  return {
    claimer: "",
    vestingIds: []
  };
}
var MsgClaimVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVesting",
  encode(message, writer = BinaryWriter.create()) {
    if (message.claimer !== "") {
      writer.uint32(10).string(message.claimer);
    }
    for (const v of message.vestingIds) {
      writer.uint32(18).string(v);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimer = reader.string();
          break;
        case 2:
          message.vestingIds.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgClaimVesting();
    message.claimer = object.claimer ?? "";
    message.vestingIds = object.vestingIds?.map((e) => e) || [];
    return message;
  }
};
function createBaseMsgClaimVestingResponse() {
  return {
    totalClaimed: "",
    vestingsClaimed: 0
  };
}
var MsgClaimVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVestingResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.totalClaimed !== "") {
      writer.uint32(10).string(message.totalClaimed);
    }
    if (message.vestingsClaimed !== 0) {
      writer.uint32(16).uint32(message.vestingsClaimed);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimVestingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalClaimed = reader.string();
          break;
        case 2:
          message.vestingsClaimed = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgClaimVestingResponse();
    message.totalClaimed = object.totalClaimed ?? "";
    message.vestingsClaimed = object.vestingsClaimed ?? 0;
    return message;
  }
};
function createBaseMsgPauseVesting() {
  return {
    authority: "",
    vestingId: "",
    reason: ""
  };
}
var MsgPauseVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVesting",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgPauseVesting();
    message.authority = object.authority ?? "";
    message.vestingId = object.vestingId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgPauseVestingResponse() {
  return {};
}
var MsgPauseVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVestingResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseVestingResponse();
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
    const message = createBaseMsgPauseVestingResponse();
    return message;
  }
};
function createBaseMsgResumeVesting() {
  return {
    authority: "",
    vestingId: ""
  };
}
var MsgResumeVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVesting",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgResumeVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgResumeVesting();
    message.authority = object.authority ?? "";
    message.vestingId = object.vestingId ?? "";
    return message;
  }
};
function createBaseMsgResumeVestingResponse() {
  return {};
}
var MsgResumeVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVestingResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgResumeVestingResponse();
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
    const message = createBaseMsgResumeVestingResponse();
    return message;
  }
};
function createBaseMsgAccelerateVesting() {
  return {
    authority: "",
    vestingId: "",
    accelerationFactor: 0
  };
}
var MsgAccelerateVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVesting",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    if (message.accelerationFactor !== 0) {
      writer.uint32(24).uint32(message.accelerationFactor);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAccelerateVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        case 3:
          message.accelerationFactor = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgAccelerateVesting();
    message.authority = object.authority ?? "";
    message.vestingId = object.vestingId ?? "";
    message.accelerationFactor = object.accelerationFactor ?? 0;
    return message;
  }
};
function createBaseMsgAccelerateVestingResponse() {
  return {};
}
var MsgAccelerateVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVestingResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgAccelerateVestingResponse();
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
    const message = createBaseMsgAccelerateVestingResponse();
    return message;
  }
};
function createBaseMsgFalsifyVesting() {
  return {
    challenger: "",
    vestingId: "",
    reason: "",
    counterEvidenceHash: ""
  };
}
var MsgFalsifyVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVesting",
  encode(message, writer = BinaryWriter.create()) {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    if (message.counterEvidenceHash !== "") {
      writer.uint32(34).string(message.counterEvidenceHash);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFalsifyVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        case 4:
          message.counterEvidenceHash = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgFalsifyVesting();
    message.challenger = object.challenger ?? "";
    message.vestingId = object.vestingId ?? "";
    message.reason = object.reason ?? "";
    message.counterEvidenceHash = object.counterEvidenceHash ?? "";
    return message;
  }
};
function createBaseMsgFalsifyVestingResponse() {
  return {
    vestingPaused: false
  };
}
var MsgFalsifyVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVestingResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.vestingPaused === true) {
      writer.uint32(8).bool(message.vestingPaused);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgFalsifyVestingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.vestingPaused = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgFalsifyVestingResponse();
    message.vestingPaused = object.vestingPaused ?? false;
    return message;
  }
};
function createBaseMsgCompleteVesting() {
  return {
    authority: "",
    vestingId: ""
  };
}
var MsgCompleteVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVesting",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCompleteVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCompleteVesting();
    message.authority = object.authority ?? "";
    message.vestingId = object.vestingId ?? "";
    return message;
  }
};
function createBaseMsgCompleteVestingResponse() {
  return {
    remainingAmount: ""
  };
}
var MsgCompleteVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVestingResponse",
  encode(message, writer = BinaryWriter.create()) {
    if (message.remainingAmount !== "") {
      writer.uint32(10).string(message.remainingAmount);
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgCompleteVestingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.remainingAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgCompleteVestingResponse();
    message.remainingAmount = object.remainingAmount ?? "";
    return message;
  }
};
function createBaseMsgUpdateParams17() {
  return {
    authority: "",
    params: void 0
  };
}
var MsgUpdateParams17 = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParams",
  encode(message, writer = BinaryWriter.create()) {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== void 0) {
      Params18.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams17();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params18.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object) {
    const message = createBaseMsgUpdateParams17();
    message.authority = object.authority ?? "";
    message.params = object.params !== void 0 && object.params !== null ? Params18.fromPartial(object.params) : void 0;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse17() {
  return {};
}
var MsgUpdateParamsResponse17 = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParamsResponse",
  encode(_, writer = BinaryWriter.create()) {
    return writer;
  },
  decode(input, length) {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === void 0 ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse17();
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
    const message = createBaseMsgUpdateParamsResponse17();
    return message;
  }
};

// src/generated/zerone/vesting_rewards/v1/tx.registry.ts
var registry20 = [["/zerone.vesting_rewards.v1.MsgCreateVesting", MsgCreateVesting], ["/zerone.vesting_rewards.v1.MsgClaimVesting", MsgClaimVesting], ["/zerone.vesting_rewards.v1.MsgPauseVesting", MsgPauseVesting], ["/zerone.vesting_rewards.v1.MsgResumeVesting", MsgResumeVesting], ["/zerone.vesting_rewards.v1.MsgAccelerateVesting", MsgAccelerateVesting], ["/zerone.vesting_rewards.v1.MsgFalsifyVesting", MsgFalsifyVesting], ["/zerone.vesting_rewards.v1.MsgCompleteVesting", MsgCompleteVesting], ["/zerone.vesting_rewards.v1.MsgUpdateParams", MsgUpdateParams17]];
var MessageComposer20 = {
  encoded: {
    createVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVesting",
        value: MsgCreateVesting.encode(value).finish()
      };
    },
    claimVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVesting",
        value: MsgClaimVesting.encode(value).finish()
      };
    },
    pauseVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVesting",
        value: MsgPauseVesting.encode(value).finish()
      };
    },
    resumeVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVesting",
        value: MsgResumeVesting.encode(value).finish()
      };
    },
    accelerateVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVesting",
        value: MsgAccelerateVesting.encode(value).finish()
      };
    },
    falsifyVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVesting",
        value: MsgFalsifyVesting.encode(value).finish()
      };
    },
    completeVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVesting",
        value: MsgCompleteVesting.encode(value).finish()
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParams",
        value: MsgUpdateParams17.encode(value).finish()
      };
    }
  },
  withTypeUrl: {
    createVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVesting",
        value
      };
    },
    claimVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVesting",
        value
      };
    },
    pauseVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVesting",
        value
      };
    },
    resumeVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVesting",
        value
      };
    },
    accelerateVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVesting",
        value
      };
    },
    falsifyVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVesting",
        value
      };
    },
    completeVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVesting",
        value
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParams",
        value
      };
    }
  },
  fromPartial: {
    createVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVesting",
        value: MsgCreateVesting.fromPartial(value)
      };
    },
    claimVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVesting",
        value: MsgClaimVesting.fromPartial(value)
      };
    },
    pauseVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVesting",
        value: MsgPauseVesting.fromPartial(value)
      };
    },
    resumeVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVesting",
        value: MsgResumeVesting.fromPartial(value)
      };
    },
    accelerateVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVesting",
        value: MsgAccelerateVesting.fromPartial(value)
      };
    },
    falsifyVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVesting",
        value: MsgFalsifyVesting.fromPartial(value)
      };
    },
    completeVesting(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVesting",
        value: MsgCompleteVesting.fromPartial(value)
      };
    },
    updateParams(value) {
      return {
        typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParams",
        value: MsgUpdateParams17.fromPartial(value)
      };
    }
  }
};

export {
  tx_exports,
  registry,
  MessageComposer,
  tx_exports2,
  registry2,
  MessageComposer2,
  tx_exports3,
  registry3,
  MessageComposer3,
  tx_exports4,
  registry4,
  MessageComposer4,
  tx_exports5,
  registry5,
  MessageComposer5,
  tx_exports6,
  registry6,
  MessageComposer6,
  tx_exports7,
  registry7,
  MessageComposer7,
  tx_exports8,
  registry8,
  MessageComposer8,
  tx_exports9,
  registry9,
  MessageComposer9,
  tx_exports10,
  registry10,
  MessageComposer10,
  tx_exports11,
  registry11,
  MessageComposer11,
  tx_exports12,
  registry12,
  MessageComposer12,
  tx_exports13,
  registry13,
  MessageComposer13,
  tx_exports14,
  registry14,
  MessageComposer14,
  tx_exports15,
  registry15,
  MessageComposer15,
  tx_exports16,
  registry16,
  MessageComposer16,
  tx_exports17,
  registry17,
  MessageComposer17,
  tx_exports18,
  registry18,
  MessageComposer18,
  tx_exports19,
  registry19,
  MessageComposer19,
  tx_exports20,
  registry20,
  MessageComposer20
};
