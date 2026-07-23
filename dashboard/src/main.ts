import "./styles.css";
import {
  getNetworkSnapshot,
  getRecentBlocks,
  microToDisplay,
  type LiquidityParams,
  type LiquidityPool,
  type NetworkSnapshot,
  type RecentBlock,
} from "./api";
import { CHAIN_ID, HARD_CAP_ZRN } from "./config";
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
const copyAddressButton = byId<HTMLButtonElement>("copy-address");
const sendDialog = byId<HTMLDialogElement>("send-dialog");
const sendForm = byId<HTMLFormElement>("send-form");
const sendError = byId<HTMLParagraphElement>("send-error");
const sendSubmit = byId<HTMLButtonElement>("send-submit");
const toast = byId<HTMLDivElement>("toast");

let snapshot: NetworkSnapshot | null = null;
let connectedWallet: WalletState | null = null;
let networkRefreshRunning = false;
let walletConnectRunning = false;
let toastTimer: number | undefined;
let poolsSignature = "";
let paramsSignature = "";
let sendPending = false;

const loadWallet = () => import("./wallet");

function formatHeight(height: number): string {
  return new Intl.NumberFormat("en-GB").format(height);
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
    ? `${percentFromMillionScale(params.protocolFeeBps)} · ZRN-in fee`
    : "Unavailable";
  byId("minimum-liquidity").textContent = params
    ? `${microToDisplay(params.minInitialLiquidity, 0)} ZRN`
    : "Unavailable";
  byId("twap-window").textContent = params
    ? `${formatHeight(params.twapWindowBlocks)} blocks`
    : "Unavailable";
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

function renderPoolRegistry(pools: LiquidityPool[] | null): void {
  poolContent.replaceChildren();

  if (pools === null) {
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

  if (pools.length === 0) {
    poolHeading.textContent = "No pools on zerone-1";
    const state = element("div", "pool-empty");
    state.append(
      element("div", "empty-symbol", "0"),
      element("h3", "", "No governance-approved pools exist yet."),
      element(
        "p",
        "",
        "There is currently no on-chain ZRN exchange rate, swap route, TVL, or APY. The first pool can only be created by governance.",
      ),
    );
    const notes = element("div", "empty-notes");
    [
      "AMM module live",
      "Hardening applied at block 44,636",
      "Creation is governance-only",
    ].forEach((note) => notes.append(element("span", "", note)));
    state.append(notes);
    poolContent.append(state);
    return;
  }

  poolHeading.textContent = `${pools.length} live ${pools.length === 1 ? "pool" : "pools"}`;
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
    const status = element("span", `pool-status ${pool.locked ? "locked" : "ready"}`);
    status.textContent = pool.locked ? "Busy" : "Readable";
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

  updateMetric(poolValue, next.pools === null ? null : String(next.pools.length));
  poolNote.textContent =
    next.pools === null
      ? "Pool registry unavailable"
      : next.pools.length === 0
        ? "No on-chain market yet"
        : `${next.pools.length} governance-approved ${next.pools.length === 1 ? "pool" : "pools"}`;

  if (next.validators !== null && next.peers !== null) {
    custodyCopy.textContent = `The unsealed custodial launch has ${next.validators} consensus ${next.validators === 1 ? "validator" : "validators"}; the public RPC node currently sees ${next.peers} connected ${next.peers === 1 ? "peer" : "peers"}. Block production and governance are not distributed yet, and the operator retains the disclosed ability to halt, revert, or re-genesis.`;
  }

  const nextParamsSignature = JSON.stringify(next.liquidityParams);
  if (nextParamsSignature !== paramsSignature) {
    paramsSignature = nextParamsSignature;
    renderLiquidityParams(next.liquidityParams);
  }
  const nextPoolsSignature = JSON.stringify(next.pools);
  if (nextPoolsSignature !== poolsSignature) {
    poolsSignature = nextPoolsSignature;
    renderPoolRegistry(next.pools);
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

function renderWallet(wallet: WalletState): void {
  walletDisconnected.hidden = true;
  walletConnected.hidden = false;
  walletBalance.textContent = microToDisplay(wallet.balanceUzrn);
  walletBalance.title = `${microToDisplay(wallet.balanceUzrn)} ZRN`;
  walletAddress.textContent = shortValue(wallet.address, 13, 9);
  copyAddressButton.dataset.address = wallet.address;
  walletFootnote.textContent = `${wallet.name} · Balance read directly from ${CHAIN_ID}. Passport-issued accounts began as shared custody because the onboarding operator retained a copy of the key.`;

  document.querySelectorAll<HTMLButtonElement>(".wallet-connect").forEach((button) => {
    button.textContent = shortValue(wallet.address, 8, 5);
    button.title = wallet.address;
    button.disabled = false;
  });
}

function renderWalletDisconnected(): void {
  walletDisconnected.hidden = false;
  walletConnected.hidden = true;
  delete copyAddressButton.dataset.address;
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
  walletConnectRunning = true;
  const buttons = document.querySelectorAll<HTMLButtonElement>(".wallet-connect");
  buttons.forEach((button) => {
    button.disabled = true;
    button.textContent = "Connecting…";
  });
  try {
    connectedWallet = await (await loadWallet()).connectWallet();
    renderWallet(connectedWallet);
    showToast("Wallet connected to zerone-1.");
  } catch (error) {
    renderWalletDisconnected();
    showToast(readableError(error), "error");
  } finally {
    walletConnectRunning = false;
  }
}

async function handleWalletRefresh(): Promise<void> {
  if (!connectedWallet) return;
  const button = byId<HTMLButtonElement>("wallet-refresh");
  button.disabled = true;
  button.textContent = "Refreshing…";
  try {
    connectedWallet = await (await loadWallet()).refreshWallet(connectedWallet);
    renderWallet(connectedWallet);
    showToast("Balance refreshed.");
  } catch (error) {
    showToast(readableError(error), "error");
  } finally {
    button.disabled = false;
    button.textContent = "Refresh";
  }
}

function openSendDialog(): void {
  sendError.hidden = true;
  sendError.textContent = "";
  sendDialog.showModal();
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
byId("send-open").addEventListener("click", openSendDialog);
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
window.addEventListener("keplr_keystorechange", () => {
  if (!connectedWallet) return;
  connectedWallet = null;
  renderWalletDisconnected();
  void handleWalletConnect();
});

initialiseReveal();
void refreshNetwork(false);
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
