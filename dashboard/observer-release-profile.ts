const REPOSITORY = "https://github.com/cambridgetcg/zerone-core";
const FINGERPRINT = "09327B031F8FF2C2EE49B18F2234027FC5B68C19";

export interface ObserverPublication {
  schema: "zerone.observer-publication/v1";
  releaseId: string;
  toolingCommit: string;
  executableBaseCommit: string;
  executableSha256: string;
  manifestSha256: string;
  archiveSha256: string;
  receiptSha256: string;
  expiresAt: string;
  checkpointHeight: number;
  checkpointHash: string;
}

export function buildObserverRelease(value: ObserverPublication) {
  const fields = ["schema", "releaseId", "toolingCommit", "executableBaseCommit", "executableSha256",
    "manifestSha256", "archiveSha256", "receiptSha256", "expiresAt", "checkpointHeight", "checkpointHash"];
  if (value === null || typeof value !== "object" || Array.isArray(value) || Object.keys(value).sort().join() !== fields.sort().join() ||
      value.schema !== "zerone.observer-publication/v1" || typeof value.releaseId !== "string" ||
      !/^zerone-1-observer-[a-zA-Z0-9][a-zA-Z0-9._-]{0,95}$/u.test(value.releaseId)) {
    throw new Error("Invalid observer publication contract");
  }
  for (const key of ["toolingCommit", "executableBaseCommit"] as const) {
    if (typeof value[key] !== "string" || !/^[a-f0-9]{40}$/u.test(value[key]) || /^0+$/u.test(value[key])) throw new Error(`Invalid ${key}`);
  }
  for (const key of ["executableSha256", "manifestSha256", "archiveSha256", "receiptSha256"] as const) {
    if (typeof value[key] !== "string" || !/^[a-f0-9]{64}$/u.test(value[key]) || /^0+$/u.test(value[key])) throw new Error(`Invalid ${key}`);
  }
  if (typeof value.checkpointHash !== "string" || !/^[A-F0-9]{64}$/u.test(value.checkpointHash) ||
      /^0+$/u.test(value.checkpointHash) || !Number.isSafeInteger(value.checkpointHeight) ||
      value.checkpointHeight <= 0 || typeof value.expiresAt !== "string" ||
      !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/u.test(value.expiresAt) ||
      !Number.isFinite(Date.parse(value.expiresAt)) ||
      new Date(value.expiresAt).toISOString() !== value.expiresAt.replace(/Z$/u, ".000Z")) {
    throw new Error("Invalid observer checkpoint/expiry");
  }
  const releaseUrl = `${REPOSITORY}/releases/tag/${value.releaseId}`;
  const archiveUrl = `${REPOSITORY}/releases/download/${value.releaseId}/observer-release.tar.gz`;
  const receiptUrl = `${REPOSITORY}/releases/download/${value.releaseId}/REHEARSAL-RECEIPT.json`;
  const source = `${REPOSITORY}/blob/${value.toolingCommit}/deploy/legacy-observer`;
  return {
    ...value,
    releaseUrl,
    archiveUrl,
    guideUrl: `${source}/README.md`,
    buildGuideUrl: `${source}/README-build.md`,
    receiptUrl,
    signatureFingerprint: FINGERPRINT,
    chainId: "zerone-1",
    role: "observer",
    validatorPower: 0,
    platform: "linux/amd64",
    executableProvenance: "Legacy application source plus the dependency patch authenticated inside the release archive.",
    executableSourceStatus: "reproduced-observer-patch",
    executableBaseIsUnpatchedSource: true,
    sameExecutableAsLiveValidator: false,
    expiryScope: "bootstrap-only",
    checkpointTrust: "Operator-selected sole-validator checkpoint; the RPC aliases share one upstream.",
    independentTrustAnchor: false,
    completeHistoricalReplay: false,
    transactionAdmission: false,
    validatorAdmission: false,
    maintenance: "Experimental legacy runtime: SDK 0.50 is end of life and Go 1.25 is outside upstream support. Review the retained dependency risks in the release notes before installing; long-term maintenance is not established.",
    prerequisites: "Linux amd64, Python 3.11+, gpgv, local Docker, 2 available CPUs, 6 GiB available memory and at least 20 GiB free disk for an initial trial.",
    verifyCommand: [
      `git clone ${REPOSITORY}.git zerone-observer-tools`,
      "cd zerone-observer-tools",
      `git checkout --detach ${value.toolingCommit}`,
      `curl --fail --location --connect-timeout 15 --max-time 300 --proto '=https' --proto-redir '=https' '${archiveUrl}' --output ../observer-release.tar.gz`,
      `python3 -I -c 'import hashlib; from pathlib import Path; assert hashlib.sha256(Path("../observer-release.tar.gz").read_bytes()).hexdigest() == "${value.archiveSha256}", "Archive digest mismatch"'`,
      'python3 -I -B deploy/legacy-observer/package.py unpack --archive "$PWD/../observer-release.tar.gz" --output "$HOME/zerone-observer-package" --gpgv "$(command -v gpgv)"',
      `curl --fail --location --connect-timeout 15 --max-time 60 --proto '=https' --proto-redir '=https' '${receiptUrl}' --output ../REHEARSAL-RECEIPT.json`,
      `curl --fail --location --connect-timeout 15 --max-time 60 --proto '=https' --proto-redir '=https' '${receiptUrl}.sig' --output ../REHEARSAL-RECEIPT.json.sig`,
      `python3 -I -c 'import hashlib; from pathlib import Path; assert hashlib.sha256(Path("../REHEARSAL-RECEIPT.json").read_bytes()).hexdigest() == "${value.receiptSha256}", "Rehearsal receipt digest mismatch"'`,
      'python3 -I -B deploy/legacy-observer/verify-rehearsal.py --bundle "$HOME/zerone-observer-package" --archive "$PWD/../observer-release.tar.gz" --receipt "$PWD/../REHEARSAL-RECEIPT.json" --signature "$PWD/../REHEARSAL-RECEIPT.json.sig" --gpgv "$(command -v gpgv)"',
    ].join(" &&\n"),
    initCommand: 'python3 -I -B "$HOME/zerone-observer-package/observer.py" init --home "$HOME/zerone-observer-home"',
    startCommand: 'python3 -I -B "$HOME/zerone-observer-package/observer.py" start --home "$HOME/zerone-observer-home"',
    statusCommand: 'python3 -I -B "$HOME/zerone-observer-package/observer.py" status --home "$HOME/zerone-observer-home"',
    stop: "Press Ctrl-C in the start terminal; keep the package and home. Repeat start to resume the same observer.",
    completion: "Wait for FOLLOWING: the fresh zero-power observer has synced and advanced with recent blocks. RPC listens at http://127.0.0.1:27657.",
    expiry: "New bootstrap is refused after the signed expiry. Obtain a refreshed signed package then. A successfully synced home can resume with its original verified package.",
  };
}
