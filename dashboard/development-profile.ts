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

function pin(value: unknown, length: number, label: string): asserts value is string {
  if (typeof value !== "string" || !new RegExp(`^[a-f0-9]{${length}}$`, "u").test(value) || /^0+$/u.test(value)) throw new Error(`Invalid ${label} pin`);
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
export function buildDevelopmentProfile(sourceCommit: string, publication?: DevelopmentPublication) {
  pin(sourceCommit, 40, "website source");
  if (publication !== undefined) validateDevelopmentPublication(publication);
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
    consensus: "The initial consensus has one operator-controlled validator seat. Development funds and full nodes do not confer block-signing membership. Separately controlled contributors and reviewers are welcome; several addresses do not prove independent people or agents.",
    funds: "Valueless development funds: 1,000 ZRN per address, once, within a 100,000 ZRN lifetime faucet budget. At most 5 new grants per IP per hour and 10 globally per hour. No cash value, promised future allocation, implied scientific credibility or proof of personhood. Existing development fee and review-settlement rules still apply.",
    reviews: "Commit and reveal each allow 3,600 blocks by default. Follow the round's actual height deadlines; block counts are not a wall-clock guarantee. Keep your committed review and salt locally until reveal.",
    resets: "A reset requires a new chain ID and a public notice preserving the old public history. This network does not replace zerone-1 or activate zerone-2.",
    privacy: "Claims, reasons, references and revealed reviews become public records. Publish only material you may share; corrections do not make prior public material private.",
    effects: { browserReads: publication ? "fixed-development-gateway-GET-only" : "none", browserFaucet: false, wallet: false, signing: false, transactions: false, browserStorage: false, agentProtocolEndpoint: false },
  } as const;
}
export type DevelopmentProfile = ReturnType<typeof buildDevelopmentProfile>;
