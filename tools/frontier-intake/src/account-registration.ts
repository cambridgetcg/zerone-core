import {
  createPrivateKey,
  createPublicKey,
  generateKeyPairSync,
  sign as signEd25519,
  type KeyObject,
} from "node:crypto";
import {
  closeSync,
  constants,
  fchmodSync,
  fstatSync,
  fsyncSync,
  mkdirSync,
  openSync,
  realpathSync,
  readSync,
  writeSync,
  type Stats,
} from "node:fs";
import { basename, dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const ACCOUNT_REGISTRATION_PROOF_DOMAIN =
  "zerone.auth/register-account/v1" as const;

const ED25519_PUBLIC_KEY_BYTES = 32;
const ED25519_SIGNATURE_BYTES = 64;
const ED25519_SPKI_PREFIX = Buffer.from("302a300506032b6570032100", "hex");
const BECH32_CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l";
const MAX_IDENTITY_FILE_BYTES = 4 * 1024;
const IDENTITY_FILE_MODE = 0o600;
const PRIVATE_DIRECTORY_MODE = 0o700;
const COLLISION_RETRIES = 100;
const COLLISION_RETRY_MS = 10;
const DARWIN_ACL_HELPER_PATH = fileURLToPath(
  new URL("../build/darwin-acl-check", import.meta.url),
);
const DARWIN_ACL_HELPER_MODE = 0o555;
const MAX_DARWIN_ACL_HELPER_BYTES = 1024 * 1024;
const DARWIN_ACL_INSPECTION_LIMIT_MS = 10_000;
const DARWIN_ACL_CLEAR = "zerone-darwin-acl-v1 clear\n";
const DARWIN_ACL_PRESENT = "zerone-darwin-acl-v1 present\n";
const DARWIN_ACL_PRESENT_EXIT = 10;
const MACH_O_FAT_MAGICS = new Set(["cafebabe", "cafebabf"]);

interface ValidatedDarwinAclHelper {
  readonly fd: number;
  readonly stat: Stats;
  readonly directoryFd: number;
  readonly directoryStat: Stats;
}

class IdentityFileContentError extends Error {}

interface DirectoryIdentity {
  readonly dev: number;
  readonly ino: number;
}

interface SecuredIdentityFilePath {
  readonly path: string;
  readonly parent: DirectoryIdentity;
}

export interface PersistedIdentityKey {
  readonly key: string;
  readonly public_hex: string;
  readonly private_pkcs8_hex: string;
}

export interface RegistrationProofInput {
  readonly chainId: string;
  readonly sender: string;
  readonly publicHex: string;
  readonly accountType: "agent" | "human" | "contract" | "system";
  readonly metadata: string;
}

export interface RegistrationCommandInput {
  readonly chainId: string;
  readonly sender: string;
  readonly identity: PersistedIdentityKey;
  readonly accountType: "agent" | "human" | "contract" | "system";
  readonly metadata: string;
}

/** Parse the identity file written by frontier-intake and reject partial keys. */
export function parsePersistedIdentityKey(
  json: string,
  expectedKey: string,
): PersistedIdentityKey {
  let value: unknown;
  try {
    value = JSON.parse(json);
  } catch {
    throw new Error(`identity file for ${expectedKey} is not valid JSON`);
  }
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`identity file for ${expectedKey} must contain an object`);
  }
  const candidate = value as Record<string, unknown>;
  if (candidate.key !== expectedKey) {
    throw new Error(`identity file key does not match ${expectedKey}`);
  }
  if (typeof candidate.public_hex !== "string") {
    throw new Error(`identity file for ${expectedKey} has no public_hex`);
  }
  if (typeof candidate.private_pkcs8_hex !== "string") {
    throw new Error(`identity file for ${expectedKey} has no private_pkcs8_hex`);
  }
  return {
    key: expectedKey,
    public_hex: candidate.public_hex,
    private_pkcs8_hex: candidate.private_pkcs8_hex,
  };
}

/**
 * Load an existing identity or create one exactly once. Every file access uses
 * O_NOFOLLOW and descriptor metadata; the exclusive-create loser reopens the
 * winner through the same validation path instead of overwriting it.
 */
export function loadOrCreatePersistedIdentityKey(
  path: string,
  key: string,
): PersistedIdentityKey {
  const secured = secureIdentityFilePath(path, true);
  const securePath = secured.path;
  const parent = dirname(securePath);

  try {
    return readPersistedIdentityKey(securePath, key, secured.parent);
  } catch (error) {
    // Another exclusive creator may already have published the directory entry
    // but not completed its single write yet.
    if (error instanceof IdentityFileContentError) {
      return readIdentityAfterExclusiveCollision(securePath, key, secured.parent);
    }
    if (filesystemErrorCode(error) !== "ENOENT") throw error;
  }

  // The complete directory chain is descriptor-validated and durable before
  // private key generation. Recheck its identity around generation so a
  // rename/swap cannot silently redirect the subsequent exclusive create.
  assertPrivateDirectoryIdentity(parent, secured.parent, true);
  const identity = generatePersistedIdentityKey(key);
  assertPrivateDirectoryIdentity(parent, secured.parent, false);
  const encoded = Buffer.from(JSON.stringify(identity), "utf8");
  if (encoded.length > MAX_IDENTITY_FILE_BYTES) {
    throw new Error("generated identity file exceeds its size bound");
  }

  let collided = false;
  withBoundPrivateDirectory(parent, secured.parent, parentFd => {
    let fd: number;
    try {
      // The process cwd is bound to the already-validated parent inode for
      // this synchronous section. A rename/replace of its pathname therefore
      // cannot redirect the relative exclusive create to another directory.
      fd = openSync(
        basename(securePath),
        constants.O_RDWR |
          constants.O_CREAT |
          constants.O_EXCL |
          requiredNoFollowFlag(),
        IDENTITY_FILE_MODE,
      );
    } catch (error) {
      if (filesystemErrorCode(error) === "EEXIST") {
        collided = true;
        return;
      }
      throw new Error(
        `could not exclusively create identity file ${securePath}: ${filesystemErrorCode(error) ?? "unknown error"}`,
      );
    }

    try {
      // Enforce 0600 even under an unusually restrictive or permissive umask.
      fchmodSync(fd, IDENTITY_FILE_MODE);
      validateIdentityFileStat(fstatSync(fd), securePath, true);
      validateNoDarwinExtendedAcl(fd, securePath, "file");
      writeAll(fd, encoded);
      fsyncSync(fd);
      fsyncSync(parentFd);
    } finally {
      closeSync(fd);
    }
  });
  if (collided) {
    return readIdentityAfterExclusiveCollision(securePath, key, secured.parent);
  }
  assertPrivateDirectoryIdentity(parent, secured.parent, true);
  return identity;
}

/**
 * Securely load an already-persisted identity without creating a directory,
 * file, or replacement key. This is the only valid path for reconciling an
 * account that is already registered on-chain.
 */
export function loadPersistedIdentityKey(
  path: string,
  key: string,
): PersistedIdentityKey {
  const secured = secureIdentityFilePath(path, false);
  return readPersistedIdentityKey(secured.path, key, secured.parent);
}

/**
 * Encode the exact v1 bytes accepted by x/auth AccountRegistrationProofSignBytes.
 * Every string is UTF-8 and prefixed by an unsigned 32-bit big-endian length.
 */
export function accountRegistrationProofSignBytes(
  input: RegistrationProofInput,
): Buffer {
  if (
    input.chainId.length === 0 ||
    input.chainId.trim() !== input.chainId ||
    input.chainId.startsWith("\u0085") ||
    input.chainId.endsWith("\u0085")
  ) {
    throw new Error("chain ID must be non-empty without surrounding whitespace");
  }
  validateCanonicalZeroneAddress(input.sender);
  if (!/^[0-9a-f]{64}$/.test(input.publicHex)) {
    throw new Error("identity public key must be 64 lowercase hex characters");
  }
  if (!["agent", "human", "contract", "system"].includes(input.accountType)) {
    throw new Error("unsupported Zerone account type");
  }

  const publicKey = Buffer.from(input.publicHex, "hex");
  const did = `did:zrn:${input.publicHex}`;
  return Buffer.concat([
    Buffer.from(ACCOUNT_REGISTRATION_PROOF_DOMAIN, "utf8"),
    Buffer.from([0]),
    lengthPrefixedUtf8(input.chainId, "chain ID"),
    lengthPrefixedUtf8(input.sender, "sender"),
    lengthPrefixedUtf8(did, "DID"),
    publicKey,
    lengthPrefixedUtf8(input.accountType, "account type"),
    lengthPrefixedUtf8(input.metadata, "metadata"),
  ]);
}

/**
 * Build the zeroned subcommand with a proof made by the persisted PKCS#8 key.
 * The returned arguments after register-account are DID, key, type, proof.
 */
export function accountRegistrationCommandArgs(
  input: RegistrationCommandInput,
): string[] {
  const privateKey = importEd25519PrivateKey(input.identity.private_pkcs8_hex);
  const derivedPublicHex = exportEd25519PublicHex(privateKey);
  if (input.identity.public_hex !== derivedPublicHex) {
    throw new Error("persisted identity public key does not match its private key");
  }

  const proofBytes = accountRegistrationProofSignBytes({
    chainId: input.chainId,
    sender: input.sender,
    publicHex: derivedPublicHex,
    accountType: input.accountType,
    metadata: input.metadata,
  });
  const signature = signEd25519(null, proofBytes, privateKey);
  if (signature.length !== ED25519_SIGNATURE_BYTES) {
    throw new Error("Ed25519 registration proof must be 64 bytes");
  }

  return [
    "zerone_auth",
    "register-account",
    `did:zrn:${derivedPublicHex}`,
    derivedPublicHex,
    input.accountType,
    signature.toString("hex"),
  ];
}

function importEd25519PrivateKey(privatePkcs8Hex: string): KeyObject {
  if (!/^(?:[0-9a-f]{2})+$/.test(privatePkcs8Hex)) {
    throw new Error("persisted identity private key must be lowercase PKCS#8 hex");
  }
  let privateKey: KeyObject;
  try {
    privateKey = createPrivateKey({
      key: Buffer.from(privatePkcs8Hex, "hex"),
      format: "der",
      type: "pkcs8",
    });
  } catch {
    throw new Error("persisted identity private key is not valid PKCS#8 DER");
  }
  if (privateKey.asymmetricKeyType !== "ed25519") {
    throw new Error("persisted identity private key is not Ed25519");
  }
  return privateKey;
}

function generatePersistedIdentityKey(key: string): PersistedIdentityKey {
  const pair = generateKeyPairSync("ed25519");
  const publicHex = (pair.publicKey.export({ format: "der", type: "spki" }) as Buffer)
    .subarray(-ED25519_PUBLIC_KEY_BYTES)
    .toString("hex");
  const privatePkcs8Hex = (pair.privateKey.export({ format: "der", type: "pkcs8" }) as Buffer)
    .toString("hex");
  return { key, public_hex: publicHex, private_pkcs8_hex: privatePkcs8Hex };
}

function readPersistedIdentityKey(
  path: string,
  key: string,
  parentIdentity: DirectoryIdentity,
): PersistedIdentityKey {
  const parent = dirname(path);
  return withBoundPrivateDirectory(parent, parentIdentity, parentFd => {
    let fd: number;
    try {
      fd = openSync(
        basename(path),
        constants.O_RDONLY | constants.O_NONBLOCK | requiredNoFollowFlag(),
      );
    } catch (error) {
      if (filesystemErrorCode(error) === "ENOENT") throw error;
      throw new Error(
        `refusing identity file ${path}: secure no-follow open failed (${filesystemErrorCode(error) ?? "unknown error"})`,
      );
    }

    let identity: PersistedIdentityKey;
    try {
      const before = fstatSync(fd);
      validateIdentityFileStat(before, path, false);
      validateNoDarwinExtendedAcl(fd, path, "file");
      if (before.size === 0) {
        throw new IdentityFileContentError(`identity file ${path} is empty`);
      }

      const encoded = Buffer.alloc(before.size);
      let offset = 0;
      while (offset < encoded.length) {
        const count = readSync(fd, encoded, offset, encoded.length - offset, null);
        if (count === 0) {
          throw new IdentityFileContentError(`identity file ${path} changed while being read`);
        }
        offset += count;
      }
      if (readSync(fd, Buffer.alloc(1), 0, 1, null) !== 0) {
        throw new IdentityFileContentError(`identity file ${path} grew while being read`);
      }
      const after = fstatSync(fd);
      if (after.size !== before.size || after.dev !== before.dev || after.ino !== before.ino) {
        throw new IdentityFileContentError(`identity file ${path} changed while being read`);
      }

      try {
        identity = parsePersistedIdentityKey(encoded.toString("utf8"), key);
        assertPersistedIdentityKeyCustody(identity);
      } catch (error) {
        throw new IdentityFileContentError(
          error instanceof Error ? error.message : `identity file ${path} is invalid`,
        );
      }
      // A process that observes a just-created file makes both data and the
      // directory entry durable, even if it raced the creator's own fsync.
      fsyncSync(fd);
      fsyncSync(parentFd);
    } finally {
      closeSync(fd);
    }
    return identity;
  });
}

function readIdentityAfterExclusiveCollision(
  path: string,
  key: string,
  parentIdentity: DirectoryIdentity,
): PersistedIdentityKey {
  let lastError: unknown;
  for (let attempt = 0; attempt < COLLISION_RETRIES; attempt += 1) {
    try {
      return readPersistedIdentityKey(path, key, parentIdentity);
    } catch (error) {
      lastError = error;
      if (
        !(error instanceof IdentityFileContentError) &&
        filesystemErrorCode(error) !== "ENOENT"
      ) {
        throw error;
      }
      synchronousPause(COLLISION_RETRY_MS);
    }
  }
  throw lastError instanceof Error
    ? lastError
    : new Error(`identity file ${path} did not become readable after exclusive-create collision`);
}

function validateIdentityFileStat(stat: Stats, path: string, requireExactMode: boolean): void {
  if (!stat.isFile()) {
    throw new Error(`refusing identity file ${path}: path is not a regular file`);
  }
  if (typeof process.geteuid !== "function" || stat.uid !== process.geteuid()) {
    throw new Error(`refusing identity file ${path}: file is not owned by the current user`);
  }
  const mode = stat.mode & 0o777;
  if ((mode & 0o077) !== 0) {
    throw new Error(`refusing identity file ${path}: group/world permissions must be zero`);
  }
  if (requireExactMode && mode !== IDENTITY_FILE_MODE) {
    throw new Error(`refusing identity file ${path}: newly created file is not mode 0600`);
  }
  if (stat.size > MAX_IDENTITY_FILE_BYTES) {
    throw new Error(
      `refusing identity file ${path}: file exceeds ${MAX_IDENTITY_FILE_BYTES} bytes`,
    );
  }
}

/**
 * Resolve the deepest existing directory once, bind it to a descriptor, then
 * create each missing component separately. Each new directory and its
 * containing directory are fsynced before this function returns, so callers
 * cannot generate or broadcast with a merely cached custody path.
 */
function secureIdentityFilePath(
  inputPath: string,
  createParent: boolean,
): SecuredIdentityFilePath {
  if (inputPath.length === 0 || inputPath.endsWith("/")) {
    throw new Error("identity file path must name a file");
  }
  const absolutePath = resolve(inputPath);
  const fileName = basename(absolutePath);
  if (fileName === "" || fileName === "." || fileName === "..") {
    throw new Error("identity file path must have a safe basename");
  }
  const securedParent = securePrivateDirectory(dirname(absolutePath), createParent);
  return {
    path: join(securedParent.path, fileName),
    parent: securedParent.identity,
  };
}

function securePrivateDirectory(
  requestedPath: string,
  create: boolean,
): { path: string; identity: DirectoryIdentity } {
  let probe = resolve(requestedPath);
  const missingComponents: string[] = [];
  let ancestorFd: number;

  for (;;) {
    try {
      ancestorFd = openDirectoryNoFollow(probe);
      break;
    } catch (error) {
      if (filesystemErrorCode(error) !== "ENOENT") {
        throw directoryOpenError(probe, error);
      }
      const containing = dirname(probe);
      if (containing === probe) {
        throw new Error(`refusing identity directory ${requestedPath}: no existing ancestor`);
      }
      missingComponents.unshift(basename(probe));
      probe = containing;
    }
  }

  let currentFd = ancestorFd;
  try {
    validatePrivateDirectoryDescriptor(currentFd, probe, false);

    let canonicalPath: string;
    try {
      canonicalPath = realpathSync(probe);
    } catch (error) {
      throw new Error(
        `refusing identity directory ${probe}: could not bind its canonical path (${filesystemErrorCode(error) ?? "unknown error"})`,
      );
    }
    const canonicalFd = openDirectoryNoFollowOrThrow(canonicalPath);
    try {
      assertSameDirectory(
        fstatSync(currentFd),
        fstatSync(canonicalFd),
        probe,
        "changed while its canonical path was resolved",
      );
      validatePrivateDirectoryDescriptor(canonicalFd, canonicalPath, false);
    } catch (error) {
      closeSync(canonicalFd);
      throw error;
    }
    closeSync(currentFd);
    currentFd = canonicalFd;

    if (!create && missingComponents.length > 0) {
      throw new Error(
        `required identity directory ${requestedPath} does not exist; refusing to create custody for an existing account`,
      );
    }

    for (const component of missingComponents) {
      const parentIdentity = directoryIdentity(fstatSync(currentFd));
      assertPrivateDirectoryIdentity(canonicalPath, parentIdentity, false);
      const nextPath = join(canonicalPath, component);
      let created = false;
      try {
        // Intentionally non-recursive: durability and validation happen at
        // every path edge before the next edge is attempted.
        mkdirSync(nextPath, { mode: PRIVATE_DIRECTORY_MODE });
        created = true;
      } catch (error) {
        if (filesystemErrorCode(error) !== "EEXIST") {
          throw new Error(
            `could not create private identity directory ${nextPath}: ${filesystemErrorCode(error) ?? "unknown error"}`,
          );
        }
      }

      let childFd = openDirectoryNoFollowOrThrow(nextPath);
      try {
        if (created) fchmodSync(childFd, PRIVATE_DIRECTORY_MODE);
        validatePrivateDirectoryDescriptor(childFd, nextPath, created);
        assertPrivateDirectoryIdentity(canonicalPath, parentIdentity, false);

        let childCanonicalPath: string;
        try {
          childCanonicalPath = realpathSync(nextPath);
        } catch (error) {
          throw new Error(
            `refusing identity directory ${nextPath}: could not resolve newly created component (${filesystemErrorCode(error) ?? "unknown error"})`,
          );
        }
        if (childCanonicalPath !== nextPath) {
          throw new Error(
            `refusing identity directory ${nextPath}: component resolved through a symbolic link`,
          );
        }
        const childIdentity = directoryIdentity(fstatSync(childFd));
        assertPrivateDirectoryIdentity(nextPath, childIdentity, false);

        // Persist both the directory inode and its entry in the containing
        // directory before any key material can be generated.
        fsyncSync(childFd);
        fsyncSync(currentFd);

        closeSync(currentFd);
        currentFd = childFd;
        childFd = -1;
        canonicalPath = nextPath;
      } finally {
        if (childFd >= 0) closeSync(childFd);
      }
    }

    const finalIdentity = directoryIdentity(fstatSync(currentFd));
    validatePrivateDirectoryDescriptor(currentFd, canonicalPath, false);
    fsyncSync(currentFd);
    assertPrivateDirectoryIdentity(canonicalPath, finalIdentity, true);
    return {
      path: canonicalPath,
      identity: finalIdentity,
    };
  } finally {
    closeSync(currentFd);
  }
}

function openDirectoryNoFollow(path: string): number {
  return openSync(
    path,
    constants.O_RDONLY |
      constants.O_NONBLOCK |
      requiredDirectoryFlag() |
      requiredNoFollowFlag(),
  );
}

function openDirectoryNoFollowOrThrow(path: string): number {
  try {
    return openDirectoryNoFollow(path);
  } catch (error) {
    throw directoryOpenError(path, error);
  }
}

function directoryOpenError(path: string, error: unknown): Error {
  return new Error(
    `refusing identity directory ${path}: secure no-follow open failed (${filesystemErrorCode(error) ?? "unknown error"})`,
  );
}

function validatePrivateDirectoryDescriptor(
  fd: number,
  path: string,
  requireExactMode: boolean,
): void {
  const stat = fstatSync(fd);
  if (!stat.isDirectory()) {
    throw new Error(`refusing identity directory ${path}: path is not a directory`);
  }
  if (typeof process.geteuid !== "function" || stat.uid !== process.geteuid()) {
    throw new Error(`refusing identity directory ${path}: directory is not owned by the current user`);
  }
  const mode = stat.mode & 0o777;
  if ((mode & 0o077) !== 0) {
    throw new Error(`refusing identity directory ${path}: group/world permissions must be zero`);
  }
  if (requireExactMode && mode !== PRIVATE_DIRECTORY_MODE) {
    throw new Error(`refusing identity directory ${path}: newly created directory is not mode 0700`);
  }
  validateNoDarwinExtendedAcl(fd, path, "directory");
}

function assertPrivateDirectoryIdentity(
  path: string,
  expected: DirectoryIdentity,
  sync: boolean,
): void {
  const fd = openDirectoryNoFollowOrThrow(path);
  try {
    validatePrivateDirectoryDescriptor(fd, path, false);
    const actual = fstatSync(fd);
    if (actual.dev !== expected.dev || actual.ino !== expected.ino) {
      throw new Error(`refusing identity directory ${path}: directory changed during custody operation`);
    }
    if (sync) fsyncSync(fd);
  } finally {
    closeSync(fd);
  }
}

/**
 * Run a synchronous file operation relative to a cwd that has been checked
 * against the expected directory inode. POSIX keeps cwd bound to that inode
 * across renames, giving Bun/Node a descriptor-relative equivalent for the
 * short section where their fs APIs do not expose openat(2).
 */
function withBoundPrivateDirectory<T>(
  path: string,
  expected: DirectoryIdentity,
  operation: (directoryFd: number) => T,
): T {
  const previousPath = realpathSync(".");
  const previousFd = openDirectoryNoFollowOrThrow(".");
  const previousIdentity = directoryIdentity(fstatSync(previousFd));
  let boundFd: number | undefined;
  try {
    process.chdir(path);
    boundFd = openDirectoryNoFollowOrThrow(".");
    validatePrivateDirectoryDescriptor(boundFd, path, false);
    const bound = fstatSync(boundFd);
    if (bound.dev !== expected.dev || bound.ino !== expected.ino) {
      throw new Error(
        `refusing identity directory ${path}: cwd binding observed a raced directory`,
      );
    }
    return operation(boundFd);
  } finally {
    if (boundFd !== undefined) closeSync(boundFd);
    process.chdir(previousPath);
    const restoredFd = openDirectoryNoFollowOrThrow(".");
    try {
      assertSameDirectory(
        fstatSync(restoredFd),
        fstatSync(previousFd),
        previousPath,
        "original working directory changed during custody operation",
      );
      const restored = fstatSync(restoredFd);
      if (
        restored.dev !== previousIdentity.dev ||
        restored.ino !== previousIdentity.ino
      ) {
        throw new Error(
          `refusing identity operation: could not restore original working directory ${previousPath}`,
        );
      }
    } finally {
      closeSync(restoredFd);
      closeSync(previousFd);
    }
    assertPrivateDirectoryIdentity(path, expected, false);
  }
}

function assertSameDirectory(
  left: Stats,
  right: Stats,
  path: string,
  reason: string,
): void {
  if (left.dev !== right.dev || left.ino !== right.ino) {
    throw new Error(`refusing identity directory ${path}: ${reason}`);
  }
}

function directoryIdentity(stat: Stats): DirectoryIdentity {
  return { dev: stat.dev, ino: stat.ino };
}

/**
 * macOS NFSv4 ACLs are independent of the POSIX mode bits: a 0600 file can
 * still grant another principal read access, and fchmod(2) does not remove
 * those entries. Inspect the already-open object, never its pathname, so a
 * concurrent rename or symlink swap cannot change what was authorized.
 *
 * Linux POSIX ACL grants are constrained by the group-class mask updated by
 * chmod(2), so the existing zero group/world mode check remains authoritative
 * there. Any Darwin inspection error other than "no extended ACL" is fatal;
 * every other platform is unsupported and therefore also fails closed.
 */
function validateNoDarwinExtendedAcl(
  fd: number,
  path: string,
  kind: "file" | "directory",
): void {
  if (process.platform === "linux") return;
  if (process.platform !== "darwin") {
    throw new Error(
      `refusing identity ${kind} ${path}: ACL policy is unsupported on ${process.platform}`,
    );
  }

  let helper: ValidatedDarwinAclHelper;
  try {
    helper = openValidatedDarwinAclHelper();
  } catch (error) {
    throw new Error(
      `refusing identity ${kind} ${path}: descriptor ACL inspection is unavailable or unsafe (${error instanceof Error ? error.message : "unknown error"}); run make frontier-intake-darwin-acl-helper`,
    );
  }

  let targetHasExtendedAcl: boolean;
  const inspectionDeadline = performance.now() + DARWIN_ACL_INSPECTION_LIMIT_MS;
  try {
    // The release-distributed helper is the native bootstrap boundary (and
    // must be cryptographically bound by the distributor). Before it authorizes
    // a target, prove that neither its inode nor directory carries this ACL.
    if (inspectDarwinAclDescriptor(helper.fd, inspectionDeadline)) {
      throw new Error("ACL helper has an extended ACL");
    }
    assertDarwinAclHelperUnchanged(helper.stat, helper.directoryStat);
    if (inspectDarwinAclDescriptor(helper.directoryFd, inspectionDeadline)) {
      throw new Error("ACL helper directory has an extended ACL");
    }
    assertDarwinAclHelperUnchanged(helper.stat, helper.directoryStat);
    targetHasExtendedAcl = inspectDarwinAclDescriptor(fd, inspectionDeadline);
    assertDarwinAclHelperUnchanged(helper.stat, helper.directoryStat);
  } catch (error) {
    throw new Error(
      `refusing identity ${kind} ${path}: descriptor ACL inspection failed (${error instanceof Error ? error.message : "unknown error"})`,
    );
  } finally {
    closeSync(helper.fd);
    closeSync(helper.directoryFd);
  }

  if (targetHasExtendedAcl) {
    throw new Error(
      `refusing identity ${kind} ${path}: extended ACLs are not allowed`,
    );
  }
}

function inspectDarwinAclDescriptor(fd: number, deadline: number): boolean {
  const remainingMs = Math.ceil(deadline - performance.now());
  if (remainingMs <= 0) {
    throw new Error("descriptor ACL inspection timed out");
  }
  const result = Bun.spawnSync({
    cmd: [DARWIN_ACL_HELPER_PATH],
    // The helper calls acl_get_fd_np(0, ACL_TYPE_EXTENDED). Passing the
    // already-open object as fd 0 is descriptor-bound and preserves the
    // parent's fd in Bun; no /dev/fd pathname is inspected.
    stdin: fd,
    stdout: "pipe",
    stderr: "pipe",
    env: {},
    timeout: remainingMs,
  });
  const stdout = result.stdout.toString("utf8");
  const stderr = result.stderr.toString("utf8");
  if (
    result.exitCode === 0 &&
    result.signalCode === undefined &&
    stdout === DARWIN_ACL_CLEAR &&
    stderr === ""
  ) {
    return false;
  }
  if (
    result.exitCode === DARWIN_ACL_PRESENT_EXIT &&
    result.signalCode === undefined &&
    stdout === DARWIN_ACL_PRESENT &&
    stderr === ""
  ) {
    return true;
  }
  throw new Error(
    `descriptor ACL inspection returned an invalid result (exit ${String(result.exitCode)}, signal ${String(result.signalCode)})`,
  );
}

function openValidatedDarwinAclHelper(): ValidatedDarwinAclHelper {
  const directoryPath = dirname(DARWIN_ACL_HELPER_PATH);
  let directoryFd: number;
  try {
    directoryFd = openDirectoryNoFollow(directoryPath);
  } catch (error) {
    throw new Error(
      `secure no-follow helper directory open failed (${filesystemErrorCode(error) ?? "unknown error"})`,
    );
  }

  let directoryStat: Stats;
  try {
    directoryStat = fstatSync(directoryFd);
    validateDarwinAclHelperDirectoryStat(directoryStat);
  } catch (error) {
    closeSync(directoryFd);
    throw error;
  }

  let fd: number;
  try {
    fd = openSync(
      DARWIN_ACL_HELPER_PATH,
      constants.O_RDONLY | constants.O_NONBLOCK | requiredNoFollowFlag(),
    );
  } catch (error) {
    closeSync(directoryFd);
    throw new Error(
      `secure no-follow helper open failed (${filesystemErrorCode(error) ?? "unknown error"})`,
    );
  }

  try {
    const stat = fstatSync(fd);
    validateDarwinAclHelperStat(stat);
    const magic = Buffer.alloc(4);
    if (
      readSync(fd, magic, 0, magic.length, 0) !== magic.length ||
      !MACH_O_FAT_MAGICS.has(magic.toString("hex"))
    ) {
      throw new Error("helper is not a universal Mach-O executable");
    }
    assertDarwinAclHelperDirectoryUnchanged(directoryStat);
    return { fd, stat, directoryFd, directoryStat };
  } catch (error) {
    closeSync(fd);
    closeSync(directoryFd);
    throw error;
  }
}

function validateDarwinAclHelperDirectoryStat(stat: Stats): void {
  if (!stat.isDirectory()) {
    throw new Error("helper directory is not a directory");
  }
  const effectiveUid = typeof process.geteuid === "function" ? process.geteuid() : -1;
  if (stat.uid !== 0 && stat.uid !== effectiveUid) {
    throw new Error("helper directory is not owned by root or the current user");
  }
  if ((stat.mode & 0o022) !== 0) {
    throw new Error("helper directory must not be group/world writable");
  }
}

function validateDarwinAclHelperStat(stat: Stats): void {
  if (!stat.isFile()) {
    throw new Error("helper is not a regular file");
  }
  const effectiveUid = typeof process.geteuid === "function" ? process.geteuid() : -1;
  if (stat.uid !== 0 && stat.uid !== effectiveUid) {
    throw new Error("helper is not owned by root or the current user");
  }
  if ((stat.mode & 0o7777) !== DARWIN_ACL_HELPER_MODE) {
    throw new Error("helper must have exact mode 0555 with no special bits");
  }
  if (stat.nlink !== 1) {
    throw new Error("helper must have exactly one hard link");
  }
  if (stat.size <= 0 || stat.size > MAX_DARWIN_ACL_HELPER_BYTES) {
    throw new Error("helper size is outside the trusted bound");
  }
}

function assertDarwinAclHelperUnchanged(
  before: Stats,
  directoryBefore: Stats,
): void {
  assertDarwinAclHelperDirectoryUnchanged(directoryBefore);
  let afterFd: number;
  try {
    afterFd = openSync(
      DARWIN_ACL_HELPER_PATH,
      constants.O_RDONLY | constants.O_NONBLOCK | requiredNoFollowFlag(),
    );
  } catch (error) {
    throw new Error(
      `helper could not be reopened after inspection (${filesystemErrorCode(error) ?? "unknown error"})`,
    );
  }
  try {
    const after = fstatSync(afterFd);
    validateDarwinAclHelperStat(after);
    if (
      after.dev !== before.dev ||
      after.ino !== before.ino ||
      after.size !== before.size ||
      after.mtimeMs !== before.mtimeMs ||
      after.ctimeMs !== before.ctimeMs
    ) {
      throw new Error("helper changed while descriptor ACL inspection ran");
    }
    assertDarwinAclHelperDirectoryUnchanged(directoryBefore);
  } finally {
    closeSync(afterFd);
  }
}

function assertDarwinAclHelperDirectoryUnchanged(before: Stats): void {
  const path = dirname(DARWIN_ACL_HELPER_PATH);
  let fd: number;
  try {
    fd = openDirectoryNoFollow(path);
  } catch (error) {
    throw new Error(
      `helper directory could not be reopened (${filesystemErrorCode(error) ?? "unknown error"})`,
    );
  }
  try {
    const after = fstatSync(fd);
    validateDarwinAclHelperDirectoryStat(after);
    if (
      after.dev !== before.dev ||
      after.ino !== before.ino ||
      after.mtimeMs !== before.mtimeMs ||
      after.ctimeMs !== before.ctimeMs
    ) {
      throw new Error("helper directory changed while descriptor ACL inspection ran");
    }
  } finally {
    closeSync(fd);
  }
}

export function assertPersistedIdentityKeyCustody(
  identity: PersistedIdentityKey,
): void {
  const privateKey = importEd25519PrivateKey(identity.private_pkcs8_hex);
  if (identity.public_hex !== exportEd25519PublicHex(privateKey)) {
    throw new Error("persisted identity public key does not match its private key");
  }
}

function writeAll(fd: number, value: Buffer): void {
  let offset = 0;
  while (offset < value.length) {
    const count = writeSync(fd, value, offset, value.length - offset, null);
    if (count <= 0) throw new Error("could not completely write identity file");
    offset += count;
  }
}

function requiredNoFollowFlag(): number {
  if (typeof constants.O_NOFOLLOW !== "number") {
    throw new Error("this platform cannot safely open identity files without following symlinks");
  }
  return constants.O_NOFOLLOW;
}

function requiredDirectoryFlag(): number {
  if (typeof constants.O_DIRECTORY !== "number") {
    throw new Error("this platform cannot securely open the identity directory");
  }
  return constants.O_DIRECTORY;
}

function filesystemErrorCode(error: unknown): string | undefined {
  if (!error || typeof error !== "object" || !("code" in error)) return undefined;
  const code = (error as { code?: unknown }).code;
  return typeof code === "string" ? code : undefined;
}

function synchronousPause(milliseconds: number): void {
  const signal = new Int32Array(new SharedArrayBuffer(4));
  Atomics.wait(signal, 0, 0, milliseconds);
}

function exportEd25519PublicHex(privateKey: KeyObject): string {
  const spki = createPublicKey(privateKey).export({ format: "der", type: "spki" });
  if (
    spki.length !== ED25519_SPKI_PREFIX.length + ED25519_PUBLIC_KEY_BYTES ||
    !spki.subarray(0, ED25519_SPKI_PREFIX.length).equals(ED25519_SPKI_PREFIX)
  ) {
    throw new Error("could not derive a canonical Ed25519 public key");
  }
  return spki.subarray(ED25519_SPKI_PREFIX.length).toString("hex");
}

function lengthPrefixedUtf8(value: string, label: string): Buffer {
  const encoded = Buffer.from(value, "utf8");
  if (encoded.toString("utf8") !== value) {
    throw new Error(`${label} must contain well-formed Unicode`);
  }
  if (encoded.length > 0xffff_ffff) {
    throw new Error(`${label} exceeds the uint32 length bound`);
  }
  const length = Buffer.allocUnsafe(4);
  length.writeUInt32BE(encoded.length);
  return Buffer.concat([length, encoded]);
}

export function validateCanonicalZeroneAddress(address: string): void {
  if (address !== address.toLowerCase() || address.length > 90) {
    throw new Error("sender must be a canonical lowercase Zerone address");
  }
  const separator = address.lastIndexOf("1");
  if (separator !== 3 || address.slice(0, separator) !== "zrn") {
    throw new Error("sender must use the zrn Bech32 prefix");
  }
  const words = [...address.slice(separator + 1)].map(character =>
    BECH32_CHARSET.indexOf(character));
  if (words.length < 6 || words.some(word => word < 0)) {
    throw new Error("sender must be a valid Bech32 address");
  }
  if (bech32Polymod([...bech32HrpExpand("zrn"), ...words]) !== 1) {
    throw new Error("sender has an invalid Bech32 checksum");
  }
  const decoded = convertBits(words.slice(0, -6), 5, 8);
  if (decoded.length !== 20) {
    throw new Error("sender must decode to a 20-byte account address");
  }
}

function bech32HrpExpand(hrp: string): number[] {
  return [
    ...[...hrp].map(character => character.charCodeAt(0) >> 5),
    0,
    ...[...hrp].map(character => character.charCodeAt(0) & 31),
  ];
}

function bech32Polymod(values: number[]): number {
  const generators = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3];
  let checksum = 1;
  for (const value of values) {
    const top = checksum >>> 25;
    checksum = ((checksum & 0x1ffffff) << 5) ^ value;
    for (let bit = 0; bit < generators.length; bit += 1) {
      if ((top >>> bit) & 1) checksum ^= generators[bit] as number;
    }
  }
  return checksum >>> 0;
}

function convertBits(values: number[], fromBits: number, toBits: number): number[] {
  let accumulator = 0;
  let bits = 0;
  const result: number[] = [];
  const maxOutput = (1 << toBits) - 1;
  for (const value of values) {
    accumulator = (accumulator << fromBits) | value;
    bits += fromBits;
    while (bits >= toBits) {
      bits -= toBits;
      result.push((accumulator >>> bits) & maxOutput);
    }
  }
  if (bits >= fromBits || ((accumulator << (toBits - bits)) & maxOutput) !== 0) {
    throw new Error("sender has invalid Bech32 padding");
  }
  return result;
}
