const MAX_FRAGMENT_LENGTH = 16_384;
const MAX_ACCESS_TOKEN_LENGTH = 4_096;
const MAX_ERROR_DESCRIPTION_LENGTH = 512;
const MAX_ERROR_URI_LENGTH = 1_024;

const SUCCESS_KEYS = new Set([
  "access_token",
  "expires_in",
  "state",
  "token_type",
]);
const ERROR_KEYS = new Set([
  "error",
  "error_description",
  "error_uri",
  "state",
]);

export type PiCallbackFragment =
  | {
      kind: "success";
      accessToken: string;
      state: string;
    }
  | {
      kind: "error";
      errorCode: string;
    };

export class PiFragmentError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "PiFragmentError";
  }
}

function fail(message: string): never {
  throw new PiFragmentError(message);
}

function decodePart(value: string): string {
  if (/%(?![A-Fa-f0-9]{2})/.test(value)) {
    fail("The Pi callback contains invalid encoding.");
  }
  try {
    return decodeURIComponent(value.replace(/\+/g, " "));
  } catch {
    return fail("The Pi callback contains invalid encoding.");
  }
}

function parseParameters(fragment: string): Map<string, string> {
  if (
    fragment.length === 0 ||
    fragment.length > MAX_FRAGMENT_LENGTH ||
    fragment[0] !== "#"
  ) {
    fail("The Pi callback fragment is missing or too large.");
  }

  const body = fragment.slice(1);
  if (body.length === 0) fail("The Pi callback fragment is empty.");

  const segments = body.split("&");
  if (segments.length > 8) fail("The Pi callback contains too many fields.");

  const fields = new Map<string, string>();
  for (const segment of segments) {
    const separator = segment.indexOf("=");
    if (separator <= 0) fail("The Pi callback contains a malformed field.");
    const key = decodePart(segment.slice(0, separator));
    const value = decodePart(segment.slice(separator + 1));
    if (!/^[a-z_]+$/.test(key)) {
      fail("The Pi callback contains an invalid field name.");
    }
    if (fields.has(key)) fail("The Pi callback contains a duplicate field.");
    fields.set(key, value);
  }
  return fields;
}

function requireState(fields: ReadonlyMap<string, string>): string {
  const state = fields.get("state");
  if (
    state === undefined ||
    !/^[A-Za-z0-9_-]{43}$/.test(state)
  ) {
    fail("The Pi callback state is missing or invalid.");
  }
  return state;
}

function rejectUnknownFields(
  fields: ReadonlyMap<string, string>,
  allowed: ReadonlySet<string>,
): void {
  for (const key of fields.keys()) {
    if (!allowed.has(key)) fail("The Pi callback contains an unexpected field.");
  }
}

function validateOptionalErrorMetadata(fields: ReadonlyMap<string, string>): void {
  const description = fields.get("error_description");
  if (
    description !== undefined &&
    (description.length === 0 ||
      description.length > MAX_ERROR_DESCRIPTION_LENGTH ||
      /[\u0000-\u001f\u007f]/.test(description))
  ) {
    fail("The Pi callback error description is invalid.");
  }

  const errorUri = fields.get("error_uri");
  if (
    errorUri !== undefined &&
    (errorUri.length === 0 ||
      errorUri.length > MAX_ERROR_URI_LENGTH ||
      /[\u0000-\u0020\u007f]/.test(errorUri))
  ) {
    fail("The Pi callback error URI is invalid.");
  }
}

export function parsePiCallbackFragment(fragment: string): PiCallbackFragment {
  const fields = parseParameters(fragment);
  requireState(fields);

  if (fields.has("error")) {
    rejectUnknownFields(fields, ERROR_KEYS);
    if (fields.has("access_token")) {
      fail("The Pi callback mixes success and error fields.");
    }
    const errorCode = fields.get("error");
    if (
      errorCode === undefined ||
      errorCode.length === 0 ||
      errorCode.length > 128 ||
      !/^[A-Za-z0-9._-]+$/.test(errorCode)
    ) {
      fail("The Pi callback error code is invalid.");
    }
    validateOptionalErrorMetadata(fields);
    return { kind: "error", errorCode };
  }

  rejectUnknownFields(fields, SUCCESS_KEYS);
  if (
    fields.has("error_description") ||
    fields.has("error_uri")
  ) {
    fail("The Pi callback contains error metadata without an error.");
  }

  const accessToken = fields.get("access_token");
  if (
    accessToken === undefined ||
    accessToken.length < 16 ||
    accessToken.length > MAX_ACCESS_TOKEN_LENGTH ||
    !/^[\x21-\x7e]+$/.test(accessToken)
  ) {
    fail("The Pi access token is missing or invalid.");
  }

  const tokenType = fields.get("token_type");
  if (tokenType !== undefined && tokenType !== "Bearer") {
    fail("The Pi callback returned an unsupported token type.");
  }

  const expiresIn = fields.get("expires_in");
  if (
    expiresIn !== undefined &&
    (!/^[1-9]\d*$/.test(expiresIn) ||
      Number(expiresIn) > 86_400)
  ) {
    fail("The Pi callback returned an invalid expiry.");
  }

  return {
    kind: "success",
    accessToken,
    state: requireState(fields),
  };
}
