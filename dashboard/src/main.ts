import "./styles.css";
import {
  getNetworkSnapshot,
  getRecentBlocks,
  microToDisplay,
  type LiquidityParams,
  type LiquidityPool,
  type LiquidityPoolRegistry,
  type NetworkSnapshot,
  type RecentBlock,
} from "./api";
import {
  CHAIN_ID,
  FEEGRANT_SPONSORSHIP_ENABLED,
  HARD_CAP_ZRN,
} from "./config";
import { initialiseConstructiveTree } from "./constructive-tree";
import { initialiseLifeGarden } from "./life-garden";
import { initialiseQuantumSeason } from "./quantum-season";
import type { FeeGrantAllowance } from "./feegrant";
import { initialiseMathFrontier } from "./math-frontier";
import type { WalletState } from "./wallet";

const byId = <T extends HTMLElement>(id: string): T => {
  const element = document.getElementById(id);
  if (!element) throw new Error(`Missing #${id}`);
  return element as T;
};

const networkPill = byId<HTMLDivElement>("network-pill");
const networkPillLabel = byId<HTMLSpanElement>("network-pill-label");
const heroHeight = byId<HTMLElement>("hero-height");
const heroBlockAge = byId<HTMLSpanElement>("hero-block-age");
const heroState = byId<HTMLElement>("hero-state");
const supplyValue = byId<HTMLElement>("supply-value");
const supplyProgress = byId<HTMLProgressElement>("supply-progress");
const validatorValue = byId<HTMLElement>("validator-value");
const validatorNote = byId<HTMLParagraphElement>("validator-note");
const peerValue = byId<HTMLElement>("peer-value");
const peerNote = byId<HTMLParagraphElement>("peer-note");
const poolValue = byId<HTMLElement>("pool-value");
const poolNote = byId<HTMLParagraphElement>("pool-note");
const custodyCopy = byId<HTMLParagraphElement>("custody-copy");
const poolHeading = byId<HTMLElement>("pool-heading");
const poolContent = byId<HTMLDivElement>("pool-content");
const blockRows = byId<HTMLTableSectionElement>("block-rows");
const walletDisconnected = byId<HTMLDivElement>("wallet-disconnected");
const walletConnected = byId<HTMLDivElement>("wallet-connected");
const walletBalance = byId<HTMLSpanElement>("wallet-balance");
const walletAddress = byId<HTMLElement>("wallet-address");
const walletFootnote = byId<HTMLParagraphElement>("wallet-footnote");
const feeGrantSummary = byId<HTMLElement>("feegrant-summary");
const feeGrantOpenButton = byId<HTMLButtonElement>("feegrant-open");
const copyAddressButton = byId<HTMLButtonElement>("copy-address");
const sendOpenButton = byId<HTMLButtonElement>("send-open");
const sendDialog = byId<HTMLDialogElement>("send-dialog");
const sendForm = byId<HTMLFormElement>("send-form");
const sendError = byId<HTMLParagraphElement>("send-error");
const sendSubmit = byId<HTMLButtonElement>("send-submit");
const sendFeePayer = byId<HTMLSelectElement>("send-fee-payer");
const sendNoticeCopy = byId<HTMLParagraphElement>("send-notice-copy");
const feeGrantDialog = byId<HTMLDialogElement>("feegrant-dialog");
const feeGrantForm = byId<HTMLFormElement>("feegrant-form");
const feeGrantError = byId<HTMLParagraphElement>("feegrant-error");
const feeGrantSubmit = byId<HTMLButtonElement>("feegrant-submit");
const feeGrantIncoming = byId<HTMLDivElement>("feegrant-incoming");
const feeGrantRevokeForm = byId<HTMLFormElement>("feegrant-revoke-form");
const feeGrantRevokeSubmit = byId<HTMLButtonElement>(
  "feegrant-revoke-submit",
);
const feeGrantActivation = byId<HTMLParagraphElement>("feegrant-activation");
const constructiveTreeRoot = byId<HTMLElement>("constructive-tree-root");
const quantumSeasonRoot = byId<HTMLElement>("quantum-season-root");
const mathFrontierRoot = byId<HTMLElement>("math-frontier-root");
const lifeGardenRoot = byId<HTMLElement>("life-garden-root");
const piPilotSection = byId<HTMLElement>("contribute");
const toast = byId<HTMLDivElement>("toast");

let snapshot: NetworkSnapshot | null = null;
let connectedWallet: WalletState | null = null;
let networkRefreshRunning = false;
let walletConnectRunning = false;
let toastTimer: number | undefined;
let poolsSignature = "";
let paramsSignature = "";
let sendPending = false;
let feeGrantPending = false;
let walletEpoch = 0;

const loadWallet = () => import("./wallet");
const BANK_SEND_FEE_UZRN = 200_000n;
const PI_PILOT_ENABLED =
  import.meta.env.VITE_PI_PILOT_ENABLED === "true";
const PI_WALLET_PROOF_ENABLED =
  import.meta.env.VITE_PI_WALLET_PROOF_ENABLED === "true";
const PI_CONSTRUCTIVE_COMPASS_ENABLED =
  import.meta.env.VITE_PI_CONSTRUCTIVE_COMPASS_ENABLED === "true";

function formatHeight(height: number): string {
  return new Intl.NumberFormat("en-GB").format(height);
}

function formatCount(value: string): string {
  return BigInt(value).toLocaleString("en-GB");
}

function timeAgo(value: string): string {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) return "unknown";
  const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1_000));
  if (seconds < 5) return "just now";
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h ago`;
}

function shortValue(value: string, start = 9, end = 7): string {
  if (value.length <= start + end + 1) return value;
  return `${value.slice(0, start)}…${value.slice(-end)}`;
}

function percentFromMillionScale(value: number): string {
  const percent = value / 10_000;
  return `${new Intl.NumberFormat("en-GB", { maximumFractionDigits: 4 }).format(percent)}%`;
}

function readableError(error: unknown): string {
  if (!(error instanceof Error)) return "Something went wrong. Please try again.";
  if ("transactionHash" in error || "txId" in error) return error.message;
  if (/rejected|denied/i.test(error.message)) return "The request was declined in Keplr.";
  if (/failed to fetch|networkerror|load failed/i.test(error.message)) {
    return "The mainnet connection is unavailable right now.";
  }
  return error.message.replace(/^Error:\s*/i, "");
}

function showToast(message: string, tone: "success" | "error" = "success"): void {
  window.clearTimeout(toastTimer);
  toast.textContent = message;
  toast.dataset.tone = tone;
  toast.hidden = false;
  requestAnimationFrame(() => toast.classList.add("is-visible"));
  toastTimer = window.setTimeout(() => {
    toast.classList.remove("is-visible");
    window.setTimeout(() => {
      toast.hidden = true;
    }, 220);
  }, 4_500);
}

function setNetworkState(state: "online" | "offline" | "loading", label: string): void {
  if (networkPill.dataset.state !== state) networkPill.dataset.state = state;
  if (networkPillLabel.textContent !== label) networkPillLabel.textContent = label;
}

function updateMetric(element: HTMLElement, value: string | null): void {
  element.textContent = value ?? "—";
  element.classList.toggle("is-unavailable", value === null);
}

function renderLiquidityParams(params: LiquidityParams | null): void {
  byId("swap-fee").textContent = params
    ? percentFromMillionScale(params.defaultSwapFeeBps)
    : "Unavailable";
  byId("protocol-fee").textContent = params
    ? params.protocolFeePolicy === "LP_ONLY_NO_PROTOCOL_SKIM"
      ? "0% · LPs keep all swap fees"
      : `${percentFromMillionScale(params.protocolFeeBps)} · legacy pre-H1 ZRN-fee skim`
    : "Unavailable";
  byId("minimum-liquidity").textContent = params
    ? `${microToDisplay(params.minInitialLiquidity, 0)} ZRN`
    : "Unavailable";
  byId("max-pools").textContent = params
    ? params.maxPools === 0
      ? "Disabled · pre-H1"
      : `${formatHeight(params.maxPools)} open`
    : "Unavailable";
  byId("minimum-reserve").textContent = params
    ? `${BigInt(params.minReserve).toLocaleString("en-GB")} base ${params.minReserve === "1" ? "unit" : "units"}`
    : "Unavailable";
  byId("billing-oracle").textContent = params
    ? params.billingOracleEnabled
      ? "Candidates configured · live gates apply"
      : "Disabled · fail-closed"
    : "Unavailable";
  byId("billing-quotes").textContent = params
    ? params.billingQuoteDenoms.length > 0
      ? `Allowlist: ${params.billingQuoteDenoms.join(", ")}`
      : "Quote allowlist is empty"
    : "Parameter query unavailable";
  const poolDenoms = byId("pool-denoms");
  poolDenoms.textContent = params
    ? params.allowedPoolDenoms.length > 0
      ? params.allowedPoolDenoms.join(", ")
      : "None pending · fail-closed"
    : "Unavailable";
  poolDenoms.title = params?.allowedPoolDenoms.join(", ") ?? "";
  const poolCreators = byId("pool-creators");
  poolCreators.textContent = params
    ? params.poolCreators.length > 0
      ? `${params.poolCreators.length} approved`
      : "None · fail-closed"
    : "Unavailable";
  const creatorAllowlist = byId("pool-creators-list");
  creatorAllowlist.textContent = params
    ? params.poolCreators.length > 0
      ? params.poolCreators.join(", ")
      : "Creator allowlist is empty"
    : "Parameter query unavailable";
  creatorAllowlist.title = params?.poolCreators.join(", ") ?? "";
  byId("oracle-truth").textContent = params
    ? `The ${formatHeight(params.twapWindowBlocks)}-block TWAP setting is the retention and default-query window; each response's window_used proves the actual served span. Billing price discovery ${params.billingOracleEnabled ? "has allowlisted quote-denom candidates, but serves one only from an ACTIVE pool when both denoms are send-enabled, reserves meet the floor, and a complete configured TWAP exists." : "selects no pool while the quote allowlist is empty."} Pool creation ${params.poolCreationEnabled ? "requires a pending one-shot counter-denom grant and an allowlisted creator; successful creation consumes that denom grant." : "is fail-closed while either no one-shot denom grant is pending or the creator allowlist is empty."}`
    : "TWAP and billing-oracle state cannot be inferred while the parameter query is unavailable.";
}

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  text?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function poolReserve(amount: string, denom: string): string {
  return denom === "uzrn" ? `${microToDisplay(amount)} ZRN` : `${amount} ${denom}`;
}

function poolStatusPresentation(pool: LiquidityPool): {
  label: string;
  className: string;
} {
  if (pool.locked) {
    return { label: `${pool.status.replaceAll("_", " ")} · busy`, className: "locked" };
  }
  switch (pool.status) {
    case "ACTIVE":
      return { label: "Active", className: "active" };
    case "SWAPS_PAUSED":
      return { label: "Swaps paused", className: "paused" };
    case "EXIT_ONLY":
      return { label: "Exit only", className: "exit-only" };
    case "CLOSED":
      return { label: "Closed tombstone", className: "closed" };
    case "PRE_V4":
      return { label: "Pre-v4 · status absent", className: "unknown" };
  }
}

function renderPoolRegistry(registry: LiquidityPoolRegistry | null): void {
  poolContent.replaceChildren();

  if (registry === null) {
    poolHeading.textContent = "Registry unavailable";
    const state = element("div", "pool-empty error-state");
    state.append(
      element("div", "empty-symbol", "↯"),
      element("h3", "", "The pool registry could not be read."),
      element("p", "", "No value has been assumed. Refresh when the mainnet endpoint returns."),
    );
    poolContent.append(state);
    return;
  }

  const { pools } = registry;
  if (pools.length === 0) {
    poolHeading.textContent = "No pools on zerone-1";
    const state = element("div", "pool-empty");
    state.append(
      element("div", "empty-symbol", "0"),
      element("h3", "", "No admitted creator has opened a pool yet."),
      element(
        "p",
        "",
        "There is currently no on-chain ZRN exchange rate, swap route, TVL, or APY. Governance first grants a counter-denom for one creation and admits a creator through Params; that allowlisted creator then signs and funds pool creation. A successful creation consumes the denom grant.",
      ),
    );
    const notes = element("div", "empty-notes");
    [
      "AMM module live",
      "Governance grants one-shot denom admission and controls pool status",
      "Allowlisted creators sign and fund creation",
      "Successful creation consumes its denom grant",
      "LP exits remain permissionless",
      "Billing oracle requires allowlist plus live eligibility gates",
      "Closed pools remain readable tombstones",
    ].forEach((note) => notes.append(element("span", "", note)));
    state.append(notes);
    poolContent.append(state);
    return;
  }

  const activePools = pools.filter((pool) => pool.status === "ACTIVE").length;
  const preV4Pools = pools.filter((pool) => pool.status === "PRE_V4").length;
  if (!registry.complete) {
    poolHeading.textContent = `${pools.length} of ${formatCount(registry.total)} shown · ${activePools} active among shown`;
  } else {
    poolHeading.textContent =
      preV4Pools > 0
        ? `${formatCount(registry.total)} registered · ${preV4Pools} without lifecycle metadata`
        : `${formatCount(registry.total)} registered · ${activePools} active`;
  }
  const list = element("div", "pool-list");
  pools.forEach((pool) => {
    const row = element("article", "pool-row");
    const pair = element("div", "pool-pair");
    pair.append(
      element("span", "pair-mark", "01"),
      element("strong", "", `${pool.denomA} / ${pool.denomB}`),
      element("small", "", shortValue(pool.id, 10, 6)),
    );
    const reserves = element("div", "pool-reserves");
    reserves.append(
      element("span", "", poolReserve(pool.reserveA, pool.denomA)),
      element("span", "", poolReserve(pool.reserveB, pool.denomB)),
    );
    const fee = element("div", "pool-fee");
    fee.append(
      element("span", "", "Swap fee"),
      element("strong", "", percentFromMillionScale(pool.swapFeeBps)),
    );
    const presentation = poolStatusPresentation(pool);
    const status = element(
      "span",
      `pool-status ${presentation.className}`,
      presentation.label,
    );
    if (pool.closedAtBlock !== null) {
      status.title = `Closed at block ${formatHeight(pool.closedAtBlock)}`;
    }
    row.append(pair, reserves, fee, status);
    list.append(row);
  });
  poolContent.append(list);
}

function renderBlocks(blocks: RecentBlock[]): void {
  blockRows.replaceChildren();
  if (blocks.length === 0) {
    const row = element("tr");
    const cell = element("td", "table-empty", "Recent blocks are unavailable.");
    cell.colSpan = 4;
    row.append(cell);
    blockRows.append(row);
    return;
  }

  blocks.forEach((block) => {
    const row = element("tr");
    const height = element("td");
    const heightCode = element("code", "height-code", `#${formatHeight(block.height)}`);
    height.append(heightCode);
    const age = element("td", "", timeAgo(block.time));
    age.dataset.timestamp = block.time;
    const transactions = element("td", "", String(block.transactionCount));
    const hash = element("td");
    hash.append(element("code", "hash-code", shortValue(block.hash, 12, 8)));
    row.append(height, age, transactions, hash);
    blockRows.append(row);
  });
}

function updateSnapshot(next: NetworkSnapshot): void {
  const previous = snapshot;
  snapshot = next;
  const blockAgeMs = Date.now() - Date.parse(next.blockTime);
  const fresh = blockAgeMs >= 0 && blockAgeMs <= 30_000;
  const regressed = previous !== null && next.height < previous.height;
  const healthy = next.chainId === CHAIN_ID && !next.catchingUp && fresh && !regressed;
  const stateLabel = regressed
    ? "Height regression"
    : !fresh
      ? "Mainnet stale"
      : next.catchingUp
        ? "Node syncing"
        : `${CHAIN_ID} live`;
  setNetworkState(
    healthy ? "online" : regressed || !fresh ? "offline" : "loading",
    stateLabel,
  );
  heroHeight.textContent = formatHeight(next.height);
  heroBlockAge.textContent = `sealed ${timeAgo(next.blockTime)}`;
  heroState.textContent = healthy
    ? "Producing"
    : regressed
      ? "Verify chain"
      : fresh
        ? "Syncing"
        : "Stalled";
  if (regressed) {
    showToast(
      "Chain height moved backwards. Verify the mainnet trust state before acting.",
      "error",
    );
  }

  if (next.supplyUzrn === null) {
    updateMetric(supplyValue, null);
    supplyProgress.value = 0;
  } else {
    const issued = Number(next.supplyUzrn) / 1_000_000;
    updateMetric(
      supplyValue,
      new Intl.NumberFormat("en-GB", { maximumFractionDigits: 6 }).format(issued),
    );
    supplyValue.title = `${microToDisplay(next.supplyUzrn)} ZRN issued`;
    const percentage = Math.min(100, (issued / HARD_CAP_ZRN) * 100);
    supplyProgress.value = percentage;
    supplyProgress.title = `${percentage.toFixed(6)}% issued`;
  }

  updateMetric(validatorValue, next.validators === null ? null : String(next.validators));
  validatorNote.textContent =
    next.validators === null
      ? "Consensus set unavailable"
      : next.validators === 1
        ? "One block producer · custodial"
        : `${next.validators} active block producers`;

  updateMetric(peerValue, next.peers === null ? null : String(next.peers));
  peerNote.textContent =
    next.peers === null
      ? "Peer view unavailable"
      : next.peers === 0
        ? "No peers visible from this RPC node"
        : `${next.peers} ${next.peers === 1 ? "peer" : "peers"} connected to this RPC node`;

  const poolRegistry = next.poolRegistry;
  const pools = poolRegistry?.pools ?? null;
  updateMetric(
    poolValue,
    poolRegistry === null ? null : formatCount(poolRegistry.total),
  );
  const activePoolCount =
    pools?.filter((pool) => pool.status === "ACTIVE").length ?? 0;
  const restrictedPoolCount =
    pools?.filter(
      (pool) =>
        pool.status === "SWAPS_PAUSED" ||
        pool.status === "EXIT_ONLY",
    ).length ?? 0;
  const closedPoolCount =
    pools?.filter((pool) => pool.status === "CLOSED").length ?? 0;
  const preV4PoolCount =
    pools?.filter((pool) => pool.status === "PRE_V4").length ?? 0;
  poolNote.textContent =
    poolRegistry === null
      ? "Pool registry unavailable"
      : poolRegistry.pools.length === 0
        ? "No on-chain market yet"
        : !poolRegistry.complete
          ? `${poolRegistry.pools.length} records shown of ${formatCount(poolRegistry.total)} · display cap ${poolRegistry.recordCap}`
        : preV4PoolCount > 0
          ? `${preV4PoolCount} ${preV4PoolCount === 1 ? "record has" : "records have"} no lifecycle metadata`
          : `${activePoolCount} active · ${restrictedPoolCount} restricted · ${closedPoolCount} closed`;

  if (next.validators !== null && next.peers !== null) {
    custodyCopy.textContent = `The unsealed custodial launch has ${next.validators} consensus ${next.validators === 1 ? "validator" : "validators"}; the public RPC node currently sees ${next.peers} connected ${next.peers === 1 ? "peer" : "peers"}. Block production and governance are not distributed yet, and the operator retains the disclosed ability to halt, revert, or re-genesis.`;
  }

  const nextParamsSignature = JSON.stringify(next.liquidityParams);
  if (nextParamsSignature !== paramsSignature) {
    paramsSignature = nextParamsSignature;
    renderLiquidityParams(next.liquidityParams);
  }
  const nextPoolsSignature = JSON.stringify(next.poolRegistry);
  if (nextPoolsSignature !== poolsSignature) {
    poolsSignature = nextPoolsSignature;
    renderPoolRegistry(next.poolRegistry);
  }
}

async function refreshNetwork(showFailure = true): Promise<void> {
  if (networkRefreshRunning) return;
  networkRefreshRunning = true;
  try {
    const next = await getNetworkSnapshot();
    updateSnapshot(next);
    try {
      renderBlocks(await getRecentBlocks(next.height, 6));
    } catch {
      renderBlocks([]);
    }
  } catch (error) {
    setNetworkState("offline", "Mainnet unavailable");
    heroState.textContent = snapshot ? "Stale snapshot" : "Unavailable";
    heroBlockAge.textContent = snapshot
      ? `last sealed ${timeAgo(snapshot.blockTime)}`
      : "No fresh block received";
    if (!snapshot) {
      updateMetric(supplyValue, null);
      updateMetric(validatorValue, null);
      updateMetric(peerValue, null);
      updateMetric(poolValue, null);
      validatorNote.textContent = "Consensus set unavailable";
      peerNote.textContent = "Peer view unavailable";
      poolNote.textContent = "Pool registry unavailable";
      paramsSignature = "null";
      poolsSignature = "null";
      renderLiquidityParams(null);
      renderPoolRegistry(null);
      renderBlocks([]);
    }
    if (showFailure) showToast(readableError(error), "error");
  } finally {
    networkRefreshRunning = false;
  }
}

function nativeGrantAmount(
  coins: FeeGrantAllowance["spendLimit"],
): string | null {
  if (coins === null) return null;
  return coins.find((entry) => entry.denom === "uzrn")?.amount ?? "0";
}

function feeGrantDetail(grant: FeeGrantAllowance): string {
  if (!grant.supported) {
    return `Unsupported allowance type ${grant.typeUrl}. It can be inspected and revoked, but this interface will not spend it.`;
  }
  const spendLimit = nativeGrantAmount(grant.spendLimit);
  const periodLimit = nativeGrantAmount(
    grant.periodReset !== null &&
      Date.parse(grant.periodReset) <= Date.now()
      ? grant.periodSpendLimit
      : grant.periodCanSpend,
  );
  const messages =
    grant.allowedMessages === null
      ? "all message types"
      : grant.allowedMessages
          .map((message) =>
            message === "/cosmos.bank.v1beta1.MsgSend"
              ? "bank sends"
              : message === "/zerone.claiming_pot.v1.MsgClaim"
                ? "claiming-pot claims"
                : message,
          )
          .join(", ");
  const expiration =
    grant.expiration === null
      ? "no expiry"
      : `expires ${new Intl.DateTimeFormat("en-GB", {
          dateStyle: "medium",
          timeStyle: "short",
        }).format(new Date(grant.expiration))}`;
  const limit =
    spendLimit === null
      ? "no overall fee cap"
      : `${microToDisplay(spendLimit)} ZRN overall fee cap`;
  const period =
    periodLimit === null
      ? ""
      : ` · ${microToDisplay(periodLimit)} ZRN left this period`;
  return `${limit}${period} · ${expiration} · ${messages}`;
}

function renderFeeGrantList(
  container: HTMLElement,
  grants: FeeGrantAllowance[] | null,
): void {
  container.replaceChildren();
  if (grants === null) {
    container.append(
      element(
        "p",
        "feegrant-empty",
        "This feegrant query is temporarily unavailable. No grants are assumed.",
      ),
    );
    return;
  }
  if (grants.length === 0) {
    container.append(
      element(
        "p",
        "feegrant-empty",
        "No accounts currently sponsor this wallet.",
      ),
    );
    return;
  }
  grants.forEach((grant) => {
    const row = element("article", "feegrant-row");
    const addressCode = element(
      "code",
      "",
      shortValue(grant.granter, 13, 9),
    );
    addressCode.title = grant.granter;
    row.append(addressCode);
    row.append(element("p", "", feeGrantDetail(grant)));
    container.append(row);
  });
}

function renderFeeGrantManager(wallet: WalletState): void {
  renderFeeGrantList(
    feeGrantIncoming,
    wallet.incomingFeeGrants,
  );
  feeGrantRevokeSubmit.disabled =
    feeGrantPending || wallet.frozen === true;
  feeGrantRevokeSubmit.textContent =
    wallet.frozen === true
      ? "Account frozen"
      : feeGrantPending
        ? "Waiting for Keplr…"
        : "Review revoke in Keplr";
  feeGrantActivation.hidden = FEEGRANT_SPONSORSHIP_ENABLED;
  feeGrantSubmit.disabled =
    feeGrantPending ||
    wallet.frozen === true ||
    !FEEGRANT_SPONSORSHIP_ENABLED;
  feeGrantSubmit.textContent =
    wallet.frozen === true
      ? "Account frozen"
      : !FEEGRANT_SPONSORSHIP_ENABLED
        ? "Awaiting validator guard"
      : feeGrantPending
        ? "Waiting for Keplr…"
        : "Review grant in Keplr";
}

function renderWallet(wallet: WalletState): void {
  walletDisconnected.hidden = true;
  walletConnected.hidden = false;
  walletBalance.textContent = microToDisplay(wallet.balanceUzrn);
  walletBalance.title = `${microToDisplay(wallet.balanceUzrn)} ZRN`;
  walletAddress.textContent = shortValue(wallet.address, 13, 9);
  walletAddress.title = wallet.accountId;
  copyAddressButton.dataset.address = wallet.address;
  const incoming =
    wallet.incomingFeeGrants === null
      ? "received unavailable"
      : `${wallet.incomingFeeGrants.length} received`;
  feeGrantSummary.textContent = `${incoming} · exact revoke only`;
  const identity = [wallet.name, wallet.accountId];
  if (wallet.did) identity.push(wallet.did);
  if (wallet.accountType) identity.push(`${wallet.accountType} account`);
  if (wallet.frozen !== undefined) {
    identity.push(wallet.frozen ? "Frozen on-chain" : "Active on-chain");
  }
  walletFootnote.textContent = `${identity.join(" · ")} · Balance read directly from ${CHAIN_ID}. Passport-issued accounts began as shared custody because the onboarding operator retained a copy of the key.`;
  sendOpenButton.disabled = wallet.frozen === true;
  sendOpenButton.textContent = wallet.frozen === true ? "Account frozen" : "Send ZRN";
  sendOpenButton.title =
    wallet.frozen === true
      ? "This account is frozen on-chain and cannot send ZRN."
      : "";
  feeGrantOpenButton.disabled = false;

  document.querySelectorAll<HTMLButtonElement>(".wallet-connect").forEach((button) => {
    button.textContent = shortValue(wallet.address, 8, 5);
    button.title = wallet.address;
    button.disabled = false;
  });
}

function renderWalletDisconnected(): void {
  walletDisconnected.hidden = false;
  walletConnected.hidden = true;
  walletAddress.removeAttribute("title");
  delete copyAddressButton.dataset.address;
  sendOpenButton.disabled = false;
  sendOpenButton.textContent = "Send ZRN";
  sendOpenButton.removeAttribute("title");
  feeGrantSummary.textContent = "Not loaded";
  feeGrantOpenButton.disabled = true;
  document.querySelectorAll<HTMLButtonElement>(".wallet-connect").forEach((button) => {
    button.disabled = false;
    button.removeAttribute("title");
    button.textContent = button.classList.contains("compact")
      ? "Connect wallet"
      : button.closest(".hero")
        ? "Open your wallet"
        : "Connect Keplr";
  });
}

async function handleWalletConnect(): Promise<void> {
  if (connectedWallet) {
    byId("wallet").scrollIntoView({ behavior: "smooth", block: "start" });
    return;
  }
  if (walletConnectRunning) return;
  const requestedEpoch = walletEpoch;
  walletConnectRunning = true;
  const buttons = document.querySelectorAll<HTMLButtonElement>(".wallet-connect");
  buttons.forEach((button) => {
    button.disabled = true;
    button.textContent = "Connecting…";
  });
  try {
    const wallet = await (await loadWallet()).connectWallet();
    if (requestedEpoch !== walletEpoch) return;
    connectedWallet = wallet;
    renderWallet(wallet);
    showToast("Wallet connected to zerone-1.");
  } catch (error) {
    if (requestedEpoch !== walletEpoch) return;
    renderWalletDisconnected();
    showToast(readableError(error), "error");
  } finally {
    walletConnectRunning = false;
    if (requestedEpoch !== walletEpoch && connectedWallet === null) {
      void handleWalletConnect();
    }
  }
}

async function handleWalletRefresh(): Promise<void> {
  if (!connectedWallet) return;
  const requestedWallet = connectedWallet;
  const requestedEpoch = walletEpoch;
  const button = byId<HTMLButtonElement>("wallet-refresh");
  button.disabled = true;
  button.textContent = "Refreshing…";
  try {
    const refreshed = await (await loadWallet()).refreshWallet(requestedWallet);
    if (
      requestedEpoch !== walletEpoch ||
      connectedWallet?.address !== requestedWallet.address
    ) {
      return;
    }
    connectedWallet = refreshed;
    renderWallet(refreshed);
    if (feeGrantDialog.open) renderFeeGrantManager(refreshed);
    showToast("Balance refreshed.");
  } catch (error) {
    if (
      requestedEpoch !== walletEpoch ||
      connectedWallet?.address !== requestedWallet.address
    ) {
      return;
    }
    showToast(readableError(error), "error");
  } finally {
    button.disabled = false;
    button.textContent = "Refresh";
  }
}

function updateSendFeeCopy(): void {
  const sponsor = sendFeePayer.value;
  sendNoticeCopy.textContent = sponsor
    ? `Fee sponsor: ${shortValue(sponsor, 13, 9)}. The exact on-chain grant will be checked again before Keplr opens; your wallet still signs and sends its own ZRN.`
    : "Network fee: 0.20 ZRN (200,000 gas × 1 uzrn). Keplr will show the exact recipient, amount, and fee before anything is signed.";
}

async function populateSendFeePayers(wallet: WalletState): Promise<void> {
  const ownBalance = element(
    "option",
    "",
    "Pay 0.20 ZRN from this wallet",
  );
  ownBalance.value = "";
  sendFeePayer.replaceChildren(ownBalance);
  if (!FEEGRANT_SPONSORSHIP_ENABLED) {
    sendFeePayer.disabled = true;
    sendNoticeCopy.textContent =
      "Sponsored spending remains disabled until the live validator enforces frozen fee granters before fee deduction. This send will use 0.20 ZRN from your wallet.";
    return;
  }
  const grants = wallet.incomingFeeGrants;
  if (grants !== null) {
    const { BANK_SEND_TYPE_URL, feeGrantAllowsMessage } =
      await import("./feegrant");
    grants
      .filter(
        (grant) =>
          grant.grantee === wallet.address &&
          feeGrantAllowsMessage(
            grant,
            BANK_SEND_TYPE_URL,
            BANK_SEND_FEE_UZRN,
          ),
      )
      .forEach((grant) => {
        const option = element(
          "option",
          "",
          `Use sponsor ${shortValue(grant.granter, 13, 9)}`,
        );
        option.value = grant.granter;
        option.title = grant.granter;
        sendFeePayer.append(option);
      });
  }
  sendFeePayer.disabled = sendFeePayer.options.length === 1;
  updateSendFeeCopy();
}

async function openSendDialog(): Promise<void> {
  if (connectedWallet?.frozen === true) {
    showToast("This account is frozen on-chain and cannot send ZRN.", "error");
    return;
  }
  if (!connectedWallet) return;
  sendError.hidden = true;
  sendError.textContent = "";
  sendDialog.showModal();
  try {
    await populateSendFeePayers(connectedWallet);
  } catch {
    sendFeePayer.disabled = true;
    sendNoticeCopy.textContent =
      "Fee sponsorship could not be verified. This send will use 0.20 ZRN from your wallet.";
  }
  window.setTimeout(() => byId<HTMLInputElement>("send-recipient").focus(), 0);
}

async function handleSend(event: SubmitEvent): Promise<void> {
  event.preventDefault();
  if (!connectedWallet) return;
  sendError.hidden = true;
  sendSubmit.disabled = true;
  sendSubmit.textContent = "Waiting for Keplr…";
  sendPending = true;
  byId<HTMLButtonElement>("send-close").disabled = true;

  const recipient = byId<HTMLInputElement>("send-recipient").value.trim();
  const amount = byId<HTMLInputElement>("send-amount").value.trim();
  const memo = byId<HTMLInputElement>("send-memo").value;

  try {
    const result = await (await loadWallet()).sendZrn(
      connectedWallet,
      recipient,
      amount,
      memo,
      FEEGRANT_SPONSORSHIP_ENABLED
        ? sendFeePayer.value || undefined
        : undefined,
    );
    sendDialog.close();
    sendForm.reset();
    showToast(`Sent. Tx ${shortValue(result.transactionHash, 10, 8)}`);
    window.setTimeout(() => void handleWalletRefresh(), 2_500);
    window.setTimeout(() => void refreshNetwork(false), 2_500);
  } catch (error) {
    const txHash =
      typeof error === "object" && error !== null && "transactionHash" in error
        ? String(error.transactionHash)
        : typeof error === "object" && error !== null && "txId" in error
          ? String(error.txId)
          : "";
    if (txHash) {
      sendDialog.close();
      sendForm.reset();
      showToast(
        `${readableError(error)} Tx ${shortValue(txHash, 10, 8)}. Verify its final result before retrying.`,
        "error",
      );
      window.setTimeout(() => void handleWalletRefresh(), 2_500);
    } else if (sendDialog.open) {
      sendError.textContent = readableError(error);
      sendError.hidden = false;
    } else {
      showToast(readableError(error), "error");
    }
  } finally {
    sendPending = false;
    byId<HTMLButtonElement>("send-close").disabled = false;
    sendSubmit.disabled = false;
    sendSubmit.textContent = "Review in Keplr";
  }
}

function transactionHashFromError(error: unknown): string {
  return typeof error === "object" && error !== null && "transactionHash" in error
    ? String(error.transactionHash)
    : typeof error === "object" && error !== null && "txId" in error
      ? String(error.txId)
      : "";
}

function openFeeGrantDialog(): void {
  if (!connectedWallet) return;
  feeGrantError.hidden = true;
  feeGrantError.textContent = "";
  renderFeeGrantManager(connectedWallet);
  feeGrantDialog.showModal();
  window.setTimeout(
    () => byId<HTMLInputElement>("feegrant-grantee").focus(),
    0,
  );
}

function setFeeGrantPending(pending: boolean): void {
  feeGrantPending = pending;
  byId<HTMLButtonElement>("feegrant-close").disabled = pending;
  if (connectedWallet) renderFeeGrantManager(connectedWallet);
}

async function refreshFeeGrantState(): Promise<void> {
  if (!connectedWallet) return;
  const requestedWallet = connectedWallet;
  const requestedEpoch = walletEpoch;
  const refreshed = await (await loadWallet()).refreshWallet(requestedWallet);
  if (
    requestedEpoch !== walletEpoch ||
    connectedWallet?.address !== requestedWallet.address
  ) {
    return;
  }
  connectedWallet = refreshed;
  renderWallet(refreshed);
  if (feeGrantDialog.open) renderFeeGrantManager(refreshed);
}

async function handleFeeGrant(event: SubmitEvent): Promise<void> {
  event.preventDefault();
  if (!connectedWallet || feeGrantPending) return;
  feeGrantError.hidden = true;
  setFeeGrantPending(true);

  const daysValue = byId<HTMLInputElement>("feegrant-days").value.trim();
  const days = /^\d+$/.test(daysValue) ? Number(daysValue) : 0;
  const allowedMessages: string[] = [];
  if (byId<HTMLInputElement>("feegrant-bank-send").checked) {
    allowedMessages.push("/cosmos.bank.v1beta1.MsgSend");
  }
  if (byId<HTMLInputElement>("feegrant-claim").checked) {
    allowedMessages.push("/zerone.claiming_pot.v1.MsgClaim");
  }

  try {
    if (!FEEGRANT_SPONSORSHIP_ENABLED) {
      throw new Error(
        "Fee grant creation is waiting for the live validator freeze guard.",
      );
    }
    if (!Number.isSafeInteger(days) || days < 1 || days > 30) {
      throw new Error("Expiration must be a whole number from 1 to 30 days.");
    }
    const result = await (await loadWallet()).grantFeeAllowance(
      connectedWallet,
      {
        grantee: byId<HTMLInputElement>("feegrant-grantee").value.trim(),
        spendLimitZrn: byId<HTMLInputElement>("feegrant-limit").value.trim(),
        expiration: new Date(Date.now() + days * 24 * 60 * 60 * 1_000),
        allowedMessages,
      },
    );
    feeGrantForm.reset();
    try {
      await refreshFeeGrantState();
      showToast(
        `Fee grant recorded. Tx ${shortValue(result.transactionHash, 10, 8)}`,
      );
    } catch {
      showToast(
        `Fee grant recorded. Tx ${shortValue(result.transactionHash, 10, 8)}. Refresh to update the list.`,
      );
    }
  } catch (error) {
    const txHash = transactionHashFromError(error);
    if (txHash) {
      feeGrantDialog.close();
      showToast(
        `${readableError(error)} Tx ${shortValue(txHash, 10, 8)}. Verify its final result before retrying.`,
        "error",
      );
    } else {
      feeGrantError.textContent = readableError(error);
      feeGrantError.hidden = false;
    }
  } finally {
    setFeeGrantPending(false);
  }
}

async function handleFeeGrantRevoke(event: SubmitEvent): Promise<void> {
  event.preventDefault();
  if (!connectedWallet || feeGrantPending) return;
  const grantee = byId<HTMLInputElement>(
    "feegrant-revoke-grantee",
  ).value.trim();
  if (
    !window.confirm(
      `Revoke fee sponsorship for ${grantee}? Keplr will show the on-chain transaction before signing.`,
    )
  ) {
    return;
  }

  feeGrantError.hidden = true;
  setFeeGrantPending(true);
  try {
    const result = await (await loadWallet()).revokeFeeAllowance(
      connectedWallet,
      grantee,
    );
    feeGrantRevokeForm.reset();
    try {
      await refreshFeeGrantState();
      showToast(
        `Fee grant revoked. Tx ${shortValue(result.transactionHash, 10, 8)}`,
      );
    } catch {
      showToast(
        `Fee grant revoked. Tx ${shortValue(result.transactionHash, 10, 8)}. Refresh to update the list.`,
      );
    }
  } catch (error) {
    const txHash = transactionHashFromError(error);
    if (txHash) {
      feeGrantDialog.close();
      showToast(
        `${readableError(error)} Tx ${shortValue(txHash, 10, 8)}. Verify its final result before retrying.`,
        "error",
      );
    } else {
      feeGrantError.textContent = readableError(error);
      feeGrantError.hidden = false;
    }
  } finally {
    setFeeGrantPending(false);
  }
}

function initialiseReveal(): void {
  const items = document.querySelectorAll<HTMLElement>(".reveal");
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
    items.forEach((item) => item.classList.add("is-visible"));
    return;
  }
  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add("is-visible");
          observer.unobserve(entry.target);
        }
      });
    },
    { threshold: 0.08 },
  );
  items.forEach((item) => observer.observe(item));
}

async function initialisePiPilotIfEnabled(): Promise<void> {
  if (!PI_PILOT_ENABLED) return;
  try {
    const { initialisePiPilot } = await import("./pi-ui");
    await initialisePiPilot({
      walletProofEnabled: PI_WALLET_PROOF_ENABLED,
      constructiveCompass: PI_CONSTRUCTIVE_COMPASS_ENABLED
        ? constructiveTreeReady.then((constructiveTree) =>
            constructiveTree
              ? {
                  resolveCapability: (id: string) =>
                    constructiveTree.resolveCapability(id),
                  openCapability: (id: string) =>
                    constructiveTree.openCapability(id),
                }
              : null,
          )
        : null,
      getWallet: () => connectedWallet,
      connectWallet: async () => {
        await handleWalletConnect();
        return connectedWallet;
      },
      signWalletProof: async (wallet, message) =>
        (await loadWallet()).signWalletControlProof(wallet, message),
      notify: showToast,
    });
  } catch {
    showToast(
      "The optional Pi pilot is unavailable. The dashboard and Keplr remain available.",
      "error",
    );
  }
}

document.querySelectorAll<HTMLButtonElement>(".wallet-connect").forEach((button) => {
  button.addEventListener("click", () => void handleWalletConnect());
});
byId("wallet-refresh").addEventListener("click", () => void handleWalletRefresh());
byId("pools-refresh").addEventListener("click", () => void refreshNetwork());
copyAddressButton.addEventListener("click", async () => {
  const address = copyAddressButton.dataset.address;
  if (!address) return;
  try {
    await navigator.clipboard.writeText(address);
    showToast("Address copied.");
  } catch {
    showToast("Copy is unavailable. Select the address in Keplr instead.", "error");
  }
});
sendOpenButton.addEventListener("click", () => void openSendDialog());
sendFeePayer.addEventListener("change", updateSendFeeCopy);
byId("send-close").addEventListener("click", () => {
  if (!sendPending) sendDialog.close();
});
sendDialog.addEventListener("click", (event) => {
  if (event.target === sendDialog && !sendPending) sendDialog.close();
});
sendDialog.addEventListener("cancel", (event) => {
  if (sendPending) event.preventDefault();
});
sendForm.addEventListener("submit", (event) => void handleSend(event));
feeGrantOpenButton.addEventListener("click", openFeeGrantDialog);
byId("feegrant-close").addEventListener("click", () => {
  if (!feeGrantPending) feeGrantDialog.close();
});
feeGrantDialog.addEventListener("click", (event) => {
  if (event.target === feeGrantDialog && !feeGrantPending) {
    feeGrantDialog.close();
  }
});
feeGrantDialog.addEventListener("cancel", (event) => {
  if (feeGrantPending) event.preventDefault();
});
feeGrantForm.addEventListener("submit", (event) => void handleFeeGrant(event));
feeGrantRevokeForm.addEventListener(
  "submit",
  (event) => void handleFeeGrantRevoke(event),
);
window.addEventListener("keplr_keystorechange", () => {
  if (!connectedWallet && !walletConnectRunning) return;
  walletEpoch += 1;
  if (sendDialog.open && !sendPending) sendDialog.close();
  if (feeGrantDialog.open && !feeGrantPending) feeGrantDialog.close();
  connectedWallet = null;
  renderWalletDisconnected();
  void handleWalletConnect();
});

initialiseReveal();
const constructiveTreeReady = initialiseConstructiveTree(constructiveTreeRoot);
const quantumSeasonReady = initialiseQuantumSeason(quantumSeasonRoot);
const mathFrontierReady = initialiseMathFrontier(mathFrontierRoot);
const lifeGardenReady = initialiseLifeGarden(lifeGardenRoot);
const piPilotReady = initialisePiPilotIfEnabled();
const initialNetworkReady = refreshNetwork(false);
const alignInitialHash = (): void => {
  if (
    window.location.hash !== "#skills" &&
    window.location.hash !== "#math-frontier" &&
    window.location.hash !== "#life"
  ) {
    return;
  }
  window.requestAnimationFrame(() => {
    const target =
      window.location.hash === "#math-frontier"
        ? mathFrontierRoot.closest<HTMLElement>("#math-frontier")
        : window.location.hash === "#life"
          ? lifeGardenRoot.closest<HTMLElement>("#life")
          : constructiveTreeRoot.closest<HTMLElement>("#skills");
    target?.scrollIntoView({ block: "start", behavior: "instant" });
  });
};
const alignPiHash = (): void => {
  if (window.location.hash !== "#contribute" || piPilotSection.hidden) return;
  window.requestAnimationFrame(() => {
    piPilotSection.scrollIntoView({ block: "start", behavior: "instant" });
  });
};
void Promise.allSettled([
  constructiveTreeReady,
  quantumSeasonReady,
  mathFrontierReady,
  lifeGardenReady,
]).then(alignInitialHash);
void Promise.allSettled([
  constructiveTreeReady,
  quantumSeasonReady,
  mathFrontierReady,
  lifeGardenReady,
  initialNetworkReady,
]).then(alignInitialHash);
void piPilotReady.then(alignPiHash);
window.setInterval(() => {
  if (!document.hidden) void refreshNetwork(false);
}, 20_000);
window.setInterval(() => {
  document.querySelectorAll<HTMLElement>("[data-timestamp]").forEach((node) => {
    node.textContent = timeAgo(node.dataset.timestamp ?? "");
  });
  if (snapshot) {
    const prefix = networkPill.dataset.state === "online" ? "sealed" : "last sealed";
    heroBlockAge.textContent = `${prefix} ${timeAgo(snapshot.blockTime)}`;
  }
}, 1_000);
