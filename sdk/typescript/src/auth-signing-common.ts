import { fromBech32 } from "@cosmjs/encoding";

export const U32_MAX = 0xffff_ffff;

const BIP173_MAX_LENGTH = 90;

export function encodeChainId(chainId: string): Uint8Array {
  const encoded = encodeText(chainId, "chain ID");
  if (
    chainId.length === 0 ||
    chainId.trim() !== chainId ||
    chainId.startsWith("\u0085") ||
    chainId.endsWith("\u0085")
  ) {
    throw new RangeError(
      "chain ID must be non-empty without surrounding whitespace",
    );
  }
  return encoded;
}

export function encodeText(value: string, label: string): Uint8Array {
  if (hasIllFormedUtf16(value)) {
    throw new RangeError(`${label} must contain well-formed Unicode`);
  }
  const encoded = new TextEncoder().encode(value);
  if (encoded.length > U32_MAX) {
    throw new RangeError(`${label} exceeds the uint32 length bound`);
  }
  return encoded;
}

export function validateZeroneAddress(address: string): void {
  if (address !== address.toLowerCase()) {
    throw new RangeError("sender must be a canonical lowercase Zerone address");
  }
  let decoded: ReturnType<typeof fromBech32>;
  try {
    decoded = fromBech32(address, BIP173_MAX_LENGTH);
  } catch {
    throw new RangeError("sender must be a valid Zerone Bech32 address");
  }
  if (decoded.prefix !== "zrn" || decoded.data.length !== 20) {
    throw new RangeError("sender must be a 20-byte zrn account address");
  }
}

export function writeUint32(
  output: Uint8Array,
  offset: number,
  value: number,
): number {
  new DataView(output.buffer, output.byteOffset, output.byteLength).setUint32(
    offset,
    value,
    false,
  );
  return offset + 4;
}

function hasIllFormedUtf16(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const codeUnit = value.charCodeAt(index);
    if (codeUnit >= 0xd800 && codeUnit <= 0xdbff) {
      const next = value.charCodeAt(index + 1);
      if (!(next >= 0xdc00 && next <= 0xdfff)) return true;
      index += 1;
    } else if (codeUnit >= 0xdc00 && codeUnit <= 0xdfff) {
      return true;
    }
  }
  return false;
}
