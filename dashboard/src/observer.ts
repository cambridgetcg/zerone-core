import "./styles.css";
import "./observer.css";
import { NETWORK_PROFILE } from "./config";
import { createObserverClient } from "./api";
import { observerReadinessPresentation } from "./onboarding";
import { observerReadinessAtAge, type ObserverBlock, type ObserverSnapshot } from "./observer-api";

const byId = <T extends HTMLElement>(id: string): T => {
  const node = document.getElementById(id);
  if (!node) throw new Error(`Missing #${id}`);
  return node as T;
};
const text = (id: string, value: string): void => { byId(id).textContent = value; };
const client = createObserverClient(NETWORK_PROFILE, location.origin);
const refreshButton = byId<HTMLButtonElement>("observer-refresh");
const submit = byId<HTMLButtonElement>("lookup-submit");
const input = byId<HTMLInputElement>("lookup-value");
const kind = byId<HTMLSelectElement>("lookup-kind");
let busy = false;
let lookupBusy = false;
let snapshot: ObserverSnapshot | null = null;
let timer: ReturnType<typeof setTimeout> | undefined;

// Keep native amounts exact, including zero; never round through Number.
function displayUzrn(value: string): string {
  const integer = BigInt(value);
  const remainder = String(integer % 1_000_000n).padStart(6, "0").replace(/0+$/, "");
  return `${(integer / 1_000_000n).toLocaleString("en-GB")}${remainder ? `.${remainder}` : ""}`;
}
function renderBlocks(blocks: ObserverBlock[], unavailable = "Recent blocks unavailable; no zero count inferred"): void {
  const rows = byId("block-rows");
  rows.replaceChildren();
  for (const block of blocks) {
    const row = document.createElement("tr");
    for (const value of [String(block.height), block.time, String(block.transactionCount), block.hash]) {
      const cell = document.createElement("td");
      cell.textContent = value;
      row.append(cell);
    }
    rows.append(row);
  }
  if (!blocks.length) {
    const row = document.createElement("tr");
    const cell = document.createElement("td");
    cell.colSpan = 4;
    cell.textContent = unavailable;
    row.append(cell);
    rows.append(row);
  }
}
function statusLabel(): void {
  if (!snapshot) return;
  const readiness = observerReadinessAtAge(snapshot.readiness, Date.now() - Date.parse(snapshot.block.time));
  const presentation = observerReadinessPresentation(readiness);
  text("network-pill-label", presentation.label);
  text("hero-state", readiness === "archive" ? "Frozen archive" : readiness);
  text("hero-block-age", snapshot.block.time);
  text("observer-status", `${presentation.detail} ${snapshot.issues.join(" ")}`);
  byId("network-pill").dataset.state = readiness === "ready" || readiness === "archive" ? "online" : "offline";
}
function scheduleRefresh(): void {
  clearTimeout(timer);
  timer = setTimeout(() => {
    if (!document.hidden) void refresh();
    else scheduleRefresh();
  }, 20_000);
}
async function refresh(): Promise<void> {
  if (busy || lookupBusy) { scheduleRefresh(); return; }
  clearTimeout(timer);
  busy = true;
  refreshButton.disabled = true;
  submit.disabled = true;
  text("observer-status", "Checking configured identity and block…");
  try {
    snapshot = await client.snapshot();
    text("hero-height", String(snapshot.block.height));
    text("supply-value", snapshot.supplyUzrn === null ? "Unknown" : displayUzrn(snapshot.supplyUzrn));
    text("validator-value", snapshot.validators === null ? "Unknown" : String(snapshot.validators));
    try { renderBlocks(await client.recentBlocks()); }
    catch { renderBlocks([]); snapshot.issues.push("Recent block reads unavailable; no completeness claim."); }
    statusLabel();
    submit.disabled = false;
  } catch (error) {
    snapshot = null;
    text("network-pill-label", "Record unavailable");
    byId("network-pill").dataset.state = "offline";
    for (const id of ["hero-height", "hero-state", "supply-value", "validator-value"]) text(id, "Unknown");
    text("hero-block-age", "No current identity check");
    text("observer-status", error instanceof Error ? error.message : "Observer unavailable");
    text("lookup-result", "Lookup disabled until network identity is checked. Previous reads are not current evidence.");
    renderBlocks([]);
  } finally {
    busy = false;
    refreshButton.disabled = false;
    // One timer per completed refresh; hidden tabs make no polling request.
    scheduleRefresh();
  }
}
byId<HTMLFormElement>("observer-lookup").addEventListener("submit", async (event) => {
  event.preventDefault();
  if (busy || lookupBusy || !snapshot) return;
  lookupBusy = true;
  submit.disabled = true;
  refreshButton.disabled = true;
  try {
    const value = input.value.trim();
    if (kind.value === "address") {
      const balance = await client.balance(value);
      text("lookup-result", `${value} · ${displayUzrn(balance)} ZRN (${balance} uzrn). Latest native balance, not migration eligibility or an entitlement.`);
    } else if (kind.value === "transaction") {
      const tx = await client.transaction(value);
      text("lookup-result", `${tx.hash} · height ${tx.height} · execution code ${tx.code} (${tx.code === 0 ? "success reported" : "failure reported"}). Gateway report, not a cryptographic inclusion proof.`);
    } else {
      if (!/^[1-9]\d{0,9}$/.test(value)) throw new Error("Enter a positive block height");
      const block = await client.block(Number(value));
      text("lookup-result", `Block ${block.height} · ${block.time} · ${block.transactionCount} transactions · hash ${block.hash} · header app hash ${block.appHash}`);
    }
  } catch (error) { text("lookup-result", error instanceof Error ? error.message : "Record unavailable; not zero"); }
  finally { lookupBusy = false; submit.disabled = false; refreshButton.disabled = false; }
});
refreshButton.addEventListener("click", () => void refresh());
document.addEventListener("visibilitychange", () => {
  if (!document.hidden && NETWORK_PROFILE.mode !== "unconfigured") void refresh();
});
if (NETWORK_PROFILE.mode !== "unconfigured" && NETWORK_PROFILE.mode !== "legacy") {
  input.disabled = false;
  kind.disabled = false;
  void refresh();
  setInterval(statusLabel, 1_000);
} else {
  text("observer-status", "Observer unconfigured. No network request, no legacy fallback.");
}
