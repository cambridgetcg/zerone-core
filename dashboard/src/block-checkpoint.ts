// The block keeper on the legacy homepage. A visitor keeps a block's height and
// full hash in this browser; on a later visit the page asks the node for that
// height again and says whether the hash is the same. Both readings come from
// the household's own node through this site, so this catches a later change
// to that block, or to any block before it, in what this site returns. It
// cannot catch a node that gave a false block in the first place, or one that
// tells the same story every time.
//
// Only a visitor's explicit "Keep" stores anything. Every touch of browser
// storage sits inside try/catch, and the page works when storage is blocked.
// Text enters the page only through createElement and textContent.

export const KEPT_BLOCKS_KEY = "zerone.kept-blocks.v1";
export const KEPT_BLOCKS_LIMIT = 5;
export const KEPT_CHAIN_ID = "zerone-1";
export const KEPT_BLOCKS_FILE_NAME = "zerone-1-kept-blocks.json";

// The site's edge reads no height above this (functions/api/_proxy.ts).
const MAX_KEPT_HEIGHT = 1_000_000_000;
const KEPT_HASH = /^[A-F0-9]{64}$/u;
const ANY_CASE_HASH = /^[A-F0-9]{64}$/iu;
const TIMESTAMP = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const KEPT_BLOCK_KEYS = ["blockTime", "chainId", "hash", "height", "keptAt"];
// The site's edge caches the status read for a few seconds (max-age=2,
// s-maxage=3 in functions/api/_proxy.ts), so a check just after a Keep can get
// a newest height lower than the block just kept. One minute is a wide margin
// over that cache.
const JUST_KEPT_MS = 60_000;

export const STORAGE_BLOCKED_MESSAGE =
  "This browser is not letting the page save anything, so kept blocks last only until you leave. Copy or download them.";
export const AT_LIMIT_MESSAGE =
  "You already keep five blocks. Forget one to keep another.";
export const UNREADABLE_MESSAGE =
  "This browser holds kept blocks that this page cannot read, so none are shown. Keeping a new block replaces them.";

export interface KeptBlock {
  chainId: typeof KEPT_CHAIN_ID;
  height: number;
  /** 64 hex characters, uppercase. */
  hash: string;
  /** The block's own time, as the node gave it (RFC 3339, UTC). */
  blockTime: string;
  /** When the visitor kept it (ISO 8601, UTC). */
  keptAt: string;
}

/** A block as the node returned it on a later read. */
export interface ObservedBlock {
  chainId: string;
  height: number;
  hash: string;
}

export type UnknownReason =
  | "no-answer"
  | "pruned"
  | "catching-up"
  | "incomplete"
  | "just-kept";

export type KeptBlockAssessment =
  | { state: "same" }
  | { state: "different"; reason: "hash" }
  | { state: "different"; reason: "missing"; latestHeight: number }
  | { state: "unknown"; reason: UnknownReason };

export interface KeptBlockReading {
  /** The block the node returned for the kept height, or null. */
  block: ObservedBlock | null;
  /** True when the request itself failed: no answer, an HTTP error, an unreadable body. */
  readFailed: boolean;
  /** From the status read in the same refresh; null when that read failed. */
  latestHeight: number | null;
  catchingUp: boolean | null;
  earliestHeight?: number | null;
  /** When this check ran (ISO 8601, UTC). */
  checkedAt: string;
}

export type KeptStorage = "saved" | "unavailable";

export interface KeptBlocksRead {
  blocks: KeptBlock[];
  storage: KeptStorage;
  /** Storage held something under the key that failed the schema. */
  unreadable: boolean;
}

export type KeptBlockView =
  | { kind: "waiting" }
  | { kind: "kept" }
  | { kind: "checking" }
  | { kind: "checked"; assessment: KeptBlockAssessment; checkedAt: string };

export type KeepResult =
  | { kind: "kept"; blocks: KeptBlock[]; block: KeptBlock }
  | { kind: "already-kept"; height: number }
  | { kind: "full" }
  | { kind: "incomplete" };

function timestampMs(value: string): number {
  // Node times carry up to nine fractional digits; parse at millisecond
  // precision so every browser reads them the same way.
  return Date.parse(value.replace(/(\.\d{3})\d+Z$/u, "$1Z"));
}

function isTimestamp(value: unknown): value is string {
  return (
    typeof value === "string" &&
    value.length <= 40 &&
    TIMESTAMP.test(value) &&
    Number.isFinite(timestampMs(value))
  );
}

/** One stored entry, accepted only with exactly the five known fields. */
export function keptBlockFrom(value: unknown): KeptBlock | null {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return null;
  }
  const keys = Object.keys(value).sort();
  if (
    keys.length !== KEPT_BLOCK_KEYS.length ||
    keys.some((key, index) => key !== KEPT_BLOCK_KEYS[index])
  ) {
    return null;
  }
  const { chainId, height, hash, blockTime, keptAt } = value as Record<
    string,
    unknown
  >;
  if (
    chainId !== KEPT_CHAIN_ID ||
    typeof height !== "number" ||
    !Number.isSafeInteger(height) ||
    height < 1 ||
    height > MAX_KEPT_HEIGHT ||
    typeof hash !== "string" ||
    !KEPT_HASH.test(hash) ||
    !isTimestamp(blockTime) ||
    !isTimestamp(keptAt)
  ) {
    return null;
  }
  return { chainId, height, hash, blockTime, keptAt };
}

/**
 * Reads what storage holds under KEPT_BLOCKS_KEY. Nothing stored is an empty
 * list; anything that fails the schema is reported as unreadable rather than
 * silently dropped.
 */
export function parseKeptBlocks(
  raw: unknown,
): { readable: true; blocks: KeptBlock[] } | { readable: false } {
  if (raw === null || raw === undefined) return { readable: true, blocks: [] };
  if (typeof raw !== "string") return { readable: false };
  let value: unknown;
  try {
    value = JSON.parse(raw);
  } catch {
    return { readable: false };
  }
  if (!Array.isArray(value) || value.length > KEPT_BLOCKS_LIMIT) {
    return { readable: false };
  }
  const blocks: KeptBlock[] = [];
  for (const entry of value) {
    const block = keptBlockFrom(entry);
    if (block === null || blocks.some((kept) => kept.height === block.height)) {
      return { readable: false };
    }
    blocks.push(block);
  }
  return { readable: true, blocks };
}

export function readKeptBlocks(): KeptBlocksRead {
  let raw: unknown;
  try {
    raw = window.localStorage.getItem(KEPT_BLOCKS_KEY);
  } catch {
    return { blocks: [], storage: "unavailable", unreadable: false };
  }
  const parsed = parseKeptBlocks(raw);
  return parsed.readable
    ? { blocks: parsed.blocks, storage: "saved", unreadable: false }
    : { blocks: [], storage: "saved", unreadable: true };
}

export function writeKeptBlocks(blocks: readonly KeptBlock[]): KeptStorage {
  const entries = blocks.map(({ chainId, height, hash, blockTime, keptAt }) => ({
    chainId,
    height,
    hash,
    blockTime,
    keptAt,
  }));
  try {
    if (entries.length === 0) {
      window.localStorage.removeItem(KEPT_BLOCKS_KEY);
    } else {
      window.localStorage.setItem(KEPT_BLOCKS_KEY, JSON.stringify(entries));
    }
    return "saved";
  } catch {
    return "unavailable";
  }
}

export function keepBlock(
  blocks: readonly KeptBlock[],
  block: { height: number; hash: string; time: string },
  keptAt: Date,
): KeepResult {
  if (blocks.some((kept) => kept.height === block.height)) {
    return { kind: "already-kept", height: block.height };
  }
  if (blocks.length >= KEPT_BLOCKS_LIMIT) return { kind: "full" };
  const kept = keptBlockFrom({
    chainId: KEPT_CHAIN_ID,
    height: block.height,
    hash: block.hash.toUpperCase(),
    blockTime: block.time,
    keptAt: keptAt.toISOString(),
  });
  if (kept === null) return { kind: "incomplete" };
  return { kind: "kept", blocks: [...blocks, kept], block: kept };
}

export function forgetKeptBlock(
  blocks: readonly KeptBlock[],
  height: number,
): KeptBlock[] {
  return blocks.filter((kept) => kept.height !== height);
}

/**
 * The list a Keep or Forget starts from. Another tab may have kept or
 * forgotten blocks since this page loaded, so readable storage wins while this
 * page's writes are landing. Once a write has failed, blocks kept only in this
 * page exist nowhere else, so the page's own list wins until a write lands.
 */
export function startingKeptBlocks(page: KeptBlocksRead): KeptBlock[] {
  if (page.storage === "unavailable") return [...page.blocks];
  const stored = readKeptBlocks();
  return stored.storage === "saved" && !stored.unreadable
    ? stored.blocks
    : [...page.blocks];
}

function savedPage(blocks: KeptBlock[], page: KeptBlocksRead): KeptBlocksRead {
  const storage = writeKeptBlocks(blocks);
  // A write that landed replaced whatever unreadable data storage held.
  return { blocks, storage, unreadable: storage === "saved" ? false : page.unreadable };
}

/** Keeps a block into the page's list and saves the list when storage allows. */
export function keepIntoPage(
  page: KeptBlocksRead,
  block: { height: number; hash: string; time: string },
  keptAt: Date,
): { page: KeptBlocksRead; result: KeepResult } {
  const current = startingKeptBlocks(page);
  const result = keepBlock(current, block, keptAt);
  if (result.kind !== "kept") return { page: { ...page, blocks: current }, result };
  return { page: savedPage(result.blocks, page), result };
}

/** Forgets a height from the page's list and saves the list when storage allows. */
export function forgetFromPage(page: KeptBlocksRead, height: number): KeptBlocksRead {
  return savedPage(forgetKeptBlock(startingKeptBlocks(page), height), page);
}

/**
 * Reads one kept height through `read` (getBlockAt on the page). A reply that
 * was not exactly this block (api.ts throws IncompleteBlockReplyError) is an
 * answer, not a failed read. Anything else that throws, such as no answer, an
 * HTTP error, a timeout or an unreadable body, is a failed read.
 */
export async function readKeptHeight(
  height: number,
  read: (height: number) => Promise<ObservedBlock>,
): Promise<Pick<KeptBlockReading, "block" | "readFailed">> {
  try {
    return { block: await read(height), readFailed: false };
  } catch (error) {
    const incomplete =
      error instanceof Error && error.name === "IncompleteBlockReplyError";
    return { block: null, readFailed: !incomplete };
  }
}

/**
 * Compares a kept block with a later reading. "same" needs the same chain,
 * the same height and an equal hash. A missing block is DIFFERENT only when
 * the read of that height failed, the same refresh's status reports a lower
 * newest height from a node that is not catching up, and the block was kept a
 * minute or more before the check. A block kept less than a minute before is
 * "unknown" instead, because the edge may serve a status reading from before
 * it was kept. A check time before the kept time means the two clock readings
 * cannot be compared, and does not soften DIFFERENT. Everything else that is
 * not a clean answer is "unknown", never "same".
 */
export function assessKeptBlock(
  kept: KeptBlock,
  reading: KeptBlockReading,
): KeptBlockAssessment {
  const { block, readFailed, latestHeight, catchingUp } = reading;
  const earliestHeight = reading.earliestHeight ?? null;
  if (catchingUp === true) return { state: "unknown", reason: "catching-up" };
  if (
    catchingUp !== false ||
    latestHeight === null ||
    !Number.isSafeInteger(latestHeight)
  ) {
    return { state: "unknown", reason: "no-answer" };
  }
  if (block === null) {
    if (!readFailed) return { state: "unknown", reason: "incomplete" };
    if (latestHeight < kept.height) {
      const age = timestampMs(reading.checkedAt) - timestampMs(kept.keptAt);
      if (Number.isFinite(age) && age >= 0 && age < JUST_KEPT_MS) {
        return { state: "unknown", reason: "just-kept" };
      }
      return { state: "different", reason: "missing", latestHeight };
    }
    if (earliestHeight !== null && kept.height < earliestHeight) {
      return { state: "unknown", reason: "pruned" };
    }
    return { state: "unknown", reason: "no-answer" };
  }
  if (
    readFailed ||
    block.chainId !== kept.chainId ||
    block.height !== kept.height ||
    !ANY_CASE_HASH.test(block.hash)
  ) {
    return { state: "unknown", reason: "incomplete" };
  }
  return block.hash.toUpperCase() === kept.hash.toUpperCase()
    ? { state: "same" }
    : { state: "different", reason: "hash" };
}

function formatHeight(height: number): string {
  return new Intl.NumberFormat("en-GB").format(height);
}

/**
 * "9 November 2026, 14:05 GMT", in the visitor's time zone unless one is
 * given. The zone is named because the copy and the download give UTC.
 */
export function formatKeeperDate(value: string, timeZone?: string): string {
  const ms = timestampMs(value);
  if (!Number.isFinite(ms)) return value;
  const date = new Date(ms);
  const day = new Intl.DateTimeFormat("en-GB", {
    day: "numeric",
    month: "long",
    year: "numeric",
    timeZone,
  }).format(date);
  const time = new Intl.DateTimeFormat("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZone,
    timeZoneName: "short",
  }).format(date);
  return `${day}, ${time}`;
}

const UNKNOWN_REASONS: Record<UnknownReason, string> = {
  // Also an HTTP error reply or a failed status read, not only silence.
  "no-answer": "The node gave no usable answer.",
  pruned: "The node no longer serves this block.",
  "catching-up": "The node is catching up.",
  // Also a reply naming another chain or another height.
  incomplete: "The reply was not a complete record of this block.",
  "just-kept":
    "The node’s newest block reads lower than this block, which you kept less than a minute ago. Check again in a minute.",
};

export function describeAssessment(
  kept: KeptBlock,
  assessment: KeptBlockAssessment,
  checkedAt: string,
  timeZone?: string,
): string {
  const height = `#${formatHeight(kept.height)}`;
  const keptOn = formatKeeperDate(kept.keptAt, timeZone);
  const checkedOn = formatKeeperDate(checkedAt, timeZone);
  if (assessment.state === "same") {
    // Only this site's reading is known, as in the DIFFERENT sentence below.
    return `Still the same. On ${checkedOn}, this site returned block ${height} with the hash you kept on ${keptOn}.`;
  }
  if (assessment.state === "different" && assessment.reason === "hash") {
    return `DIFFERENT. Block ${height} now has another hash than the one you kept on ${keptOn}. The record this site reads has changed at or before this block since then. Checked on ${checkedOn}.`;
  }
  if (assessment.state === "different") {
    return `DIFFERENT. Block ${height} is missing: the node’s newest block is now #${formatHeight(assessment.latestHeight)}, lower than the block you kept on ${keptOn}. Checked on ${checkedOn}.`;
  }
  return `Could not check. ${UNKNOWN_REASONS[assessment.reason]} This does not mean the block is unchanged. Tried on ${checkedOn}.`;
}

export function describeView(
  kept: KeptBlock,
  view: KeptBlockView,
  storage: KeptStorage,
  timeZone?: string,
): string {
  switch (view.kind) {
    case "waiting":
      return "Not checked yet on this visit.";
    case "kept":
      // When the last save failed, a block kept now is gone after this visit.
      return storage === "saved"
        ? `Kept on ${formatKeeperDate(kept.keptAt, timeZone)}. The page asks the node for this block again when you come back, or when you press Check again.`
        : `Kept on ${formatKeeperDate(kept.keptAt, timeZone)}, for this visit only. Press Check again to ask the node for it.`;
    case "checking":
      return `Asking the node for block #${formatHeight(kept.height)}.`;
    case "checked":
      return describeAssessment(kept, view.assessment, view.checkedAt, timeZone);
  }
}

export function keepResultMessage(result: KeepResult): string {
  switch (result.kind) {
    case "kept":
      return `Kept block #${formatHeight(result.block.height)}.`;
    case "already-kept":
      return `Block #${formatHeight(result.height)} is already kept.`;
    case "full":
      return AT_LIMIT_MESSAGE;
    case "incomplete":
      return "The newest block could not be kept: its record from the node was incomplete.";
  }
}

/**
 * One line for the status after a check, so a screen reader hears the check
 * once instead of every kept block's progress.
 */
export function checkSummary(
  assessments: readonly KeptBlockAssessment[],
): string {
  if (assessments.length === 0) return "";
  const count = (state: KeptBlockAssessment["state"]) =>
    assessments.filter((assessment) => assessment.state === state).length;
  const parts = (
    [
      [count("same"), "still the same"],
      [count("different"), "DIFFERENT"],
      [count("unknown"), "could not be checked"],
    ] as const
  )
    .filter(([total]) => total > 0)
    .map(([total, label]) => `${total} ${label}`);
  const blocks = assessments.length === 1 ? "block" : "blocks";
  return `Checked ${assessments.length} kept ${blocks}: ${parts.join(", ")}.`;
}

export function keeperNotice(state: {
  message: string;
  storage: KeptStorage;
  unreadable: boolean;
}): string {
  const parts = [state.message];
  if (state.storage === "unavailable") parts.push(STORAGE_BLOCKED_MESSAGE);
  else if (state.unreadable) parts.push(UNREADABLE_MESSAGE);
  return parts.filter((part) => part.length > 0).join(" ");
}

/** The JSON a visitor copies or downloads: one entry per kept block. */
export function keptBlocksDownload(
  blocks: readonly KeptBlock[],
  origin: string,
): string {
  const entries = blocks.map((kept) => ({
    chain_id: kept.chainId,
    height: kept.height,
    hash: kept.hash,
    block_time: kept.blockTime,
    kept_at: kept.keptAt,
    look_up: `${origin}/api/rpc/block?height=${kept.height}`,
  }));
  return `${JSON.stringify(entries, null, 2)}\n`;
}

export function saveKeptBlocksFile(text: string): void {
  const url = URL.createObjectURL(new Blob([text], { type: "application/json" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = KEPT_BLOCKS_FILE_NAME;
  link.click();
  window.setTimeout(() => URL.revokeObjectURL(url), 1_000);
}

function keptItem(
  kept: KeptBlock,
  onForget: (height: number) => void,
): HTMLLIElement {
  const item = document.createElement("li");
  item.dataset.height = String(kept.height);
  item.dataset.keptAt = kept.keptAt;
  const head = document.createElement("div");
  head.className = "kept-block-head";
  const height = document.createElement("code");
  height.className = "height-code";
  height.textContent = `#${formatHeight(kept.height)}`;
  const times = document.createElement("span");
  times.className = "kept-block-times";
  times.textContent = `Block time ${formatKeeperDate(kept.blockTime)} · kept ${formatKeeperDate(kept.keptAt)}`;
  head.append(height, times);
  const hash = document.createElement("code");
  hash.className = "kept-block-hash";
  hash.textContent = kept.hash;
  const result = document.createElement("p");
  result.className = "kept-block-result";
  const forget = document.createElement("button");
  forget.className = "button button-ghost compact";
  forget.type = "button";
  forget.textContent = "Forget";
  forget.setAttribute("aria-label", `Forget block #${formatHeight(kept.height)}`);
  forget.addEventListener("click", () => onForget(kept.height));
  item.append(head, hash, result, forget);
  return item;
}

/**
 * Shows the kept blocks. Items are reused by height, so a result arriving
 * while a visitor is on a Forget button does not move their focus.
 */
export function renderKeptBlocks(
  list: HTMLElement,
  blocks: readonly KeptBlock[],
  views: ReadonlyMap<number, KeptBlockView>,
  storage: KeptStorage,
  onForget: (height: number) => void,
): void {
  const existing = new Map<string, HTMLLIElement>();
  for (const child of Array.from(list.children)) {
    const item = child as HTMLLIElement;
    existing.set(`${item.dataset.height}@${item.dataset.keptAt}`, item);
  }
  const items = blocks.map((kept) => {
    const item =
      existing.get(`${kept.height}@${kept.keptAt}`) ?? keptItem(kept, onForget);
    const view = views.get(kept.height) ?? { kind: "waiting" };
    item.dataset.state = view.kind === "checked" ? view.assessment.state : "pending";
    const result = item.querySelector<HTMLElement>(".kept-block-result");
    const text = describeView(kept, view, storage);
    if (result && result.textContent !== text) result.textContent = text;
    return item;
  });
  const unchanged =
    items.length === list.children.length &&
    items.every((item, index) => list.children[index] === item);
  if (!unchanged) list.replaceChildren(...items);
}
