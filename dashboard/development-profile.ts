export const DEVELOPMENT_GATEWAY = "https://zerone-dev-1.fly.dev";
export const DEVELOPMENT_CHAIN = "zerone-dev-1";
const REPOSITORY = "https://github.com/cambridgetcg/zerone-core";

export interface DevelopmentPublication {
  schema: "zerone.development-publication/v1";
  chain_id: "zerone-dev-1";
  gateway: "https://zerone-dev-1.fly.dev";
  descriptor_sha256: string;
  genesis_sha256: string;
  rpc_genesis_sha256: string;
  runtime_source_commit: string;
  binary_sha256: string;
  verified_at: string;
}

/** A dated proposed upgrade notice, separate from the live participant publication. */
export interface DevelopmentUpgradeProposal {
  status: "proposed";
  target_source_commit: string;
  height: string;
  proposal_id: string;
  recorded_at: string;
  observed_height: string;
  packet_url: string;
  packet_sha256: string;
  release_record_url: string;
  release_record_sha256: string;
}
// Filled only from the operator's verified release and actual proposal receipt.
// Null publishes no upgrade notice or placeholder pins.
export const DEVELOPMENT_UPGRADE_PROPOSAL: DevelopmentUpgradeProposal | null = {
  "status": "proposed",
  "target_source_commit": "5542b221864ca0080e143fa82abbef4cd2064f71",
  "height": "300000",
  "proposal_id": "1",
  "recorded_at": "2026-09-13T01:12:41Z",
  "observed_height": "44135",
  "packet_url": "https://github.com/cambridgetcg/zerone-core/releases/download/zerone-dev-1-5542b221864c/UPGRADE.json",
  "packet_sha256": "04f0415a361396e28397155ecb3c0b13d578475ebf1b86c3e2e2b5d533006f75",
  "release_record_url": "https://github.com/cambridgetcg/zerone-core/releases/download/zerone-dev-1-5542b221864c/UPGRADE-RELEASE.json",
  "release_record_sha256": "be85503652e9c467e074970c7169c80d5864a2a34fc620db7ca83d0b435d1d2b"
};

export function validateDevelopmentUpgradeProposal(value: DevelopmentUpgradeProposal): void {
  const fields = ["status", "target_source_commit", "height", "proposal_id", "recorded_at", "observed_height", "packet_url", "packet_sha256", "release_record_url", "release_record_sha256"];
  if (!value || typeof value !== "object" || Array.isArray(value) || Object.keys(value).length !== fields.length || fields.some((key) => !Object.hasOwn(value, key))) throw new Error("Invalid development upgrade notice fields");
  if (value.status !== "proposed") throw new Error("An upgrade proposal cannot announce activation");
  pin(value.target_source_commit, 40, "proposed source");
  pin(value.packet_sha256, 64, "upgrade packet"); pin(value.release_record_sha256, 64, "upgrade release record");
  for (const key of ["height", "proposal_id", "observed_height"] as const) if (typeof value[key] !== "string" || !/^[1-9][0-9]*$/u.test(value[key]) || value[key].length > 19 || BigInt(value[key]) > 9223372036854775807n) throw new Error("Invalid proposal ID or observed/proposed height");
  if (typeof value.recorded_at !== "string" || !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/u.test(value.recorded_at) || new Date(value.recorded_at).toISOString() !== value.recorded_at.replace("Z", ".000Z")) throw new Error("Invalid proposal observation date");
  let tag: string | undefined;
  for (const [key, name] of [["packet_url", "UPGRADE.json"], ["release_record_url", "UPGRADE-RELEASE.json"]] as const) {
    const url = new URL(value[key]);
    const match = /^\/cambridgetcg\/zerone-core\/releases\/download\/([A-Za-z0-9._-]+)\/([^/]+)$/u.exec(url.pathname);
    if (url.origin !== "https://github.com" || url.username || url.password || url.search || url.hash || !match || match[1] !== `zerone-dev-1-${value.target_source_commit.slice(0, 12)}` || match[2] !== name || (tag && tag !== match[1])) throw new Error("Upgrade links require one exact Zerone release");
    tag = match[1];
  }
}

function pin(value: unknown, length: number, label: string): asserts value is string {
  if (typeof value !== "string" || !new RegExp(`^[a-f0-9]{${length}}$`, "u").test(value) || /^0+$/u.test(value)) throw new Error(`Invalid ${label} pin`);
}

function packageLinks(sourceCommit: string, runtimeCommit: string) {
  const tag = `zerone-dev-1-${runtimeCommit.slice(0, 12)}`;
  const recordPath = "deploy/networks/zerone-dev-1/release.json";
  return {
    tag,
    releaseUrl: `${REPOSITORY}/releases/tag/${tag}`,
    recordUrl: `${REPOSITORY}/blob/${sourceCommit}/${recordPath}`,
    recordRawUrl: `https://raw.githubusercontent.com/cambridgetcg/zerone-core/${sourceCommit}/${recordPath}`,
    sourceScope: "Package source is the runtime commit; the release record and instructions are pinned to this website's source commit.",
    verification: "Verify the archive SHA-256 against the source-pinned release record before extracting or running it. That record also identifies each platform's binary hash. A checksum identifies bytes; it is not a security audit or independent operator trust.",
    artifacts: [
      { platform: "linux-amd64", label: "Linux · x86-64" },
      { platform: "darwin-arm64", label: "macOS · Apple Silicon" },
    ].map(({ platform, label }) => {
      const name = `${tag}-${platform}.tar.gz`;
      return { platform, label, name, url: `${REPOSITORY}/releases/download/${tag}/${name}` };
    }),
  };
}

/** The website links to a committed external checksum record, not a hash supplied by its archive. */
export function validateDevelopmentRelease(value: unknown, publication: DevelopmentPublication): void {
  validateDevelopmentPublication(publication);
  const object = (item: unknown, fields: string[]): Record<string, unknown> => {
    if (!item || typeof item !== "object" || Array.isArray(item) || Object.keys(item).length !== fields.length || fields.some((key) => !Object.hasOwn(item, key))) throw new Error("Invalid development release fields");
    return item as Record<string, unknown>;
  };
  const release = object(value, ["schema", "chain_id", "source_commit", "source_tree", "release_tag", "release_url", "runtime_image", "descriptor_sha256", "genesis_sha256", "rpc_genesis_sha256", "runtime_binary_sha256", "artifacts", "bootstrap_consensus", "funds", "private_keys_in_packages"]);
  const expected = packageLinks(publication.runtime_source_commit, publication.runtime_source_commit);
  if (release.schema !== "zerone-development-release/v1" || release.chain_id !== DEVELOPMENT_CHAIN || release.source_commit !== publication.runtime_source_commit || release.release_tag !== expected.tag || release.release_url !== expected.releaseUrl || release.bootstrap_consensus !== "single-operator" || release.funds !== "valueless-development-only" || release.private_keys_in_packages !== false) throw new Error("Development release identity or scope mismatch");
  pin(release.source_tree, 40, "runtime source tree");
  if (typeof release.runtime_image !== "string" || !/^registry\.fly\.io\/zerone-dev-1@sha256:[a-f0-9]{64}$/u.test(release.runtime_image)) throw new Error("Development release requires an immutable runtime image");
  for (const field of ["descriptor_sha256", "genesis_sha256", "rpc_genesis_sha256"] as const) if (release[field] !== publication[field]) throw new Error(`Development release ${field} mismatch`);
  if (release.runtime_binary_sha256 !== publication.binary_sha256) throw new Error("Development runtime binary mismatch");
  if (!Array.isArray(release.artifacts) || release.artifacts.length !== 2) throw new Error("Both compatible development packages are required");
  const seen = new Set<string>();
  for (const item of release.artifacts) {
    const artifact = object(item, ["name", "platform", "url", "sha256", "bytes", "binary_sha256"]);
    const link = expected.artifacts.find((candidate) => candidate.platform === artifact.platform);
    if (!link || seen.has(link.platform) || artifact.name !== link.name || artifact.url !== link.url) throw new Error("Unexpected or duplicate development package");
    seen.add(link.platform);
    pin(artifact.sha256, 64, "archive"); pin(artifact.binary_sha256, 64, "package binary");
    if (typeof artifact.bytes !== "number" || !Number.isSafeInteger(artifact.bytes) || artifact.bytes <= 0 || artifact.bytes > 256 * 1024 * 1024) throw new Error("Development archive exceeds the supported download bound");
    if (link.platform === "linux-amd64" && artifact.binary_sha256 !== publication.binary_sha256) throw new Error("Linux package differs from the deployed runtime binary");
  }
}

export function validateDevelopmentPublication(value: DevelopmentPublication): void {
  const fields = ["schema", "chain_id", "gateway", "descriptor_sha256", "genesis_sha256", "rpc_genesis_sha256", "runtime_source_commit", "binary_sha256", "verified_at"];
  if (!value || typeof value !== "object" || Array.isArray(value) || Object.keys(value).length !== fields.length || fields.some((key) => !Object.hasOwn(value, key))) throw new Error("Invalid development publication fields");
  if (value.schema !== "zerone.development-publication/v1" || value.chain_id !== DEVELOPMENT_CHAIN || value.gateway !== DEVELOPMENT_GATEWAY) throw new Error("Wrong development publication network");
  for (const key of ["descriptor_sha256", "genesis_sha256", "rpc_genesis_sha256", "binary_sha256"] as const) pin(value[key], 64, key);
  pin(value.runtime_source_commit, 40, "runtime source");
  if (typeof value.verified_at !== "string" || !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/u.test(value.verified_at) || new Date(value.verified_at).toISOString() !== value.verified_at.replace("Z", ".000Z")) throw new Error("Invalid verification date");
}

/** Build-time publication evidence, never a current network-health assertion. */
export function buildDevelopmentProfile(sourceCommit: string, publication?: DevelopmentPublication, proposal: DevelopmentUpgradeProposal | null = publication ? DEVELOPMENT_UPGRADE_PROPOSAL : null) {
  pin(sourceCommit, 40, "website source");
  if (publication !== undefined) validateDevelopmentPublication(publication);
  if (proposal !== null) { validateDevelopmentUpgradeProposal(proposal); if (!publication || proposal.target_source_commit === publication.runtime_source_commit) throw new Error("Proposed target requires a distinct live predecessor publication"); }
  const source = (path: string) => `${REPOSITORY}/blob/${sourceCommit}/${path}`;
  return {
    schema: "zerone.development-guide/v1",
    chainId: DEVELOPMENT_CHAIN,
    title: "Work on a claim together",
    availability: publication ? "operator-verified-publication" : "verification-pending",
    status: publication ? `Deployment checked ${publication.verified_at}. Availability can change; reads below make a new endpoint observation.` : "Deployment verification pending. This page does not yet offer network reads or announce open participation.",
    source: { commit: sourceCommit, guide: source("docs/SHARED-DEVELOPMENT.md"), client: source("scripts/shared-claims.py"), scope: "Website documentation and participant client source; deployed binary source is identified separately." },
    endpoints: { descriptor: `${DEVELOPMENT_GATEWAY}/network.json`, genesis: `${DEVELOPMENT_GATEWAY}/genesis.json`, gateway: DEVELOPMENT_GATEWAY },
    publication: publication ?? null,
    packages: publication ? packageLinks(sourceCommit, publication.runtime_source_commit) : null,
    upgradeProposal: proposal ? {
      ...proposal,
      name: "knowledge-fund-settlement-v1",
      label: "Proposed · activation not verified here",
      releaseUrl: proposal.packet_url.replace(/\/download\/([^/]+)\/UPGRADE\.json$/u, "/tag/$1"),
      specUrl: source("docs/specs/knowledge-fund-settlement-v1.md"),
      runtimeGuideUrl: source("deploy/networks/zerone-dev-1/README.md") + "#stage-a-knowledge-10-to-11-upgrade",
      effect: "This dated proposal notice does not verify activation. The original compatible participant packages remain available below. An applied upgrade and successor descriptor require separate observation and publication.",
      participantContinuity: "Existing participant homes remain usable with their original verified package, binary, descriptor and client files. Existing commitments keep their original deadlines and can be revealed with that client. Older binaries omit the new funding fields; the current history reader can display them when the chain returns them.",
      followerContinuity: "Fresh full nodes need both verified packages: use the predecessor helper to join the original genesis, then stage both binaries with the successor helper. Replay uses version 10 until the exact upgrade boundary and version 11 afterward. No genesis reset or identity replacement is part of this path.",
      funding: "After actual activation, new admissions retain explicit funding terms. Ordinary claims and conjectures allocate 55% to review and retain the remainder; challenges allocate 55% to review and make the remainder refundable for every terminal verdict. With no eligible reveals the review budget is refundable too. A refundable amount is not evidence of payment. Network fees are separate, and old claims keep their original rules.",
    } : null,
    exampleClaim: publication ? {
      claimId: "8db0bccfb17b9a5ad1cf2389b79abd38",
      url: "/development/?claim=8db0bccfb17b9a5ad1cf2389b79abd38",
      includedAtHeight: "432",
      observedOn: "2026-09-12",
      label: "Read the operator-owned example",
      scope: "An operator-owned prime-polynomial exercise was included at height 432 on 12 September 2026. At that check it had no reviews or challenges. This is a setup exercise, with no claim of acceptance or independent endorsement; open it for a new endpoint observation.",
    } : null,
    consensus: "The initial consensus has one operator-controlled validator seat. Development funds and full nodes do not confer block-signing membership. Separately controlled contributors and reviewers are welcome; several addresses do not prove independent people or agents.",
    funds: "Valueless development funds: 1,000 ZRN per address, once, within a 100,000 ZRN lifetime faucet budget. At most 5 new grants per IP per hour and 10 globally per hour. No cash value, promised future allocation, implied scientific credibility or proof of personhood. Existing development fee and review-settlement rules still apply.",
    reviews: "Commit and reveal each allow 3,600 blocks by default. Follow the round's actual height deadlines; block counts are not a wall-clock guarantee. Keep your committed review and salt locally until reveal.",
    resets: "A reset requires a new chain ID and a public notice preserving the old public history. This network does not replace zerone-1 or activate zerone-2.",
    privacy: "Claims, reasons, references and revealed reviews become public records. Publish only material you may share; corrections do not make prior public material private.",
    effects: { browserReads: publication ? "fixed-development-gateway-GET-only" : "none", browserFaucet: false, wallet: false, signing: false, transactions: false, browserStorage: false, agentProtocolEndpoint: false },
  } as const;
}
export type DevelopmentProfile = ReturnType<typeof buildDevelopmentProfile>;
