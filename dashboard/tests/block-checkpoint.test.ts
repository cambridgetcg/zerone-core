import assert from "node:assert/strict";
import { afterEach, describe, it } from "node:test";
import { IncompleteBlockReplyError } from "../src/api";
import {
  AT_LIMIT_MESSAGE,
  KEPT_BLOCKS_KEY,
  KEPT_BLOCKS_LIMIT,
  STORAGE_BLOCKED_MESSAGE,
  UNREADABLE_MESSAGE,
  assessKeptBlock,
  checkSummary,
  describeAssessment,
  describeView,
  forgetFromPage,
  forgetKeptBlock,
  formatKeeperDate,
  keepBlock,
  keepIntoPage,
  keepResultMessage,
  keeperNotice,
  keptBlockFrom,
  keptBlocksDownload,
  parseKeptBlocks,
  readKeptBlocks,
  readKeptHeight,
  writeKeptBlocks,
  type KeptBlock,
  type KeptBlockAssessment,
  type KeptBlockReading,
  type KeptBlocksRead,
} from "../src/block-checkpoint";

const HASH = "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08";
const OTHER_HASH = "2C26B46B68FFC68FF99B453C1D30413413422D706483BFA0F98A5E886266E7AE";
const CHECKED_AT = "2026-11-10T09:12:00.000Z";

const KEPT: KeptBlock = {
  chainId: "zerone-1",
  height: 1_437_954,
  hash: HASH,
  blockTime: "2026-11-09T22:29:50.123456789Z",
  keptAt: "2026-11-09T22:31:00.000Z",
};

function reading(overrides: Partial<KeptBlockReading> = {}): KeptBlockReading {
  return {
    block: { chainId: "zerone-1", height: KEPT.height, hash: HASH },
    readFailed: false,
    latestHeight: KEPT.height + 120,
    catchingUp: false,
    earliestHeight: 1,
    ...overrides,
  };
}

function wording(assessment: KeptBlockAssessment): string {
  return describeAssessment(KEPT, assessment, CHECKED_AT, "UTC");
}

const windowDescriptor = Object.getOwnPropertyDescriptor(globalThis, "window");

function setWindow(value: unknown): void {
  Object.defineProperty(globalThis, "window", { configurable: true, value });
}

function memoryStorage(): Storage & { values: Map<string, string> } {
  const values = new Map<string, string>();
  return {
    values,
    get length() {
      return values.size;
    },
    clear: () => values.clear(),
    getItem: (key: string) => values.get(key) ?? null,
    key: (index: number) => [...values.keys()][index] ?? null,
    removeItem: (key: string) => {
      values.delete(key);
    },
    setItem: (key: string, value: string) => {
      values.set(key, String(value));
    },
  };
}

afterEach(() => {
  if (windowDescriptor) Object.defineProperty(globalThis, "window", windowDescriptor);
  else delete (globalThis as { window?: unknown }).window;
});

describe("block keeper", () => {
  it("accepts only a well-formed zerone-1 kept block and refuses extra fields, bad hex, bad heights and other chains", () => {
    assert.deepEqual(keptBlockFrom({ ...KEPT }), KEPT);
    assert.deepEqual(parseKeptBlocks(JSON.stringify([KEPT])), { readable: true, blocks: [KEPT] });
    assert.deepEqual(parseKeptBlocks(null), { readable: true, blocks: [] });

    const refused: Array<[string, unknown]> = [
      ["extra field", { ...KEPT, note: "x" }],
      ["missing field", { chainId: KEPT.chainId, height: KEPT.height, hash: KEPT.hash, blockTime: KEPT.blockTime }],
      ["short hash", { ...KEPT, hash: HASH.slice(1) }],
      ["non-hex hash", { ...KEPT, hash: `${HASH.slice(1)}G` }],
      ["lower-case hash", { ...KEPT, hash: HASH.toLowerCase() }],
      ["height zero", { ...KEPT, height: 0 }],
      ["negative height", { ...KEPT, height: -5 }],
      ["fractional height", { ...KEPT, height: 1.5 }],
      ["height as text", { ...KEPT, height: "1437954" }],
      ["height above what the edge reads", { ...KEPT, height: 1_000_000_001 }],
      ["another chain", { ...KEPT, chainId: "zerone-local-1" }],
      ["no chain", { ...KEPT, chainId: "" }],
      ["unreadable block time", { ...KEPT, blockTime: "yesterday" }],
      ["impossible block time", { ...KEPT, blockTime: "2026-13-40T00:00:00Z" }],
      ["kept time with an offset", { ...KEPT, keptAt: "2026-11-09T22:31:00+01:00" }],
      ["an array", [KEPT]],
      ["null", null],
    ];
    for (const [name, value] of refused) {
      assert.equal(keptBlockFrom(value), null, name);
      assert.deepEqual(parseKeptBlocks(JSON.stringify([value])), { readable: false }, name);
    }
    for (const raw of [
      "not json",
      "{}",
      '"text"',
      JSON.stringify([KEPT, { ...KEPT, keptAt: "2026-11-09T22:32:00.000Z" }]),
      `[{"__proto__":{},${JSON.stringify(KEPT).slice(1)}]`,
    ]) {
      assert.deepEqual(parseKeptBlocks(raw), { readable: false }, raw);
    }
    assert.deepEqual(parseKeptBlocks(42), { readable: false });
  });

  it("keeps at most five blocks and refuses a sixth", () => {
    assert.equal(KEPT_BLOCKS_LIMIT, 5);
    let blocks: KeptBlock[] = [];
    for (let index = 0; index < KEPT_BLOCKS_LIMIT; index += 1) {
      const result = keepBlock(
        blocks,
        { height: 100 + index, hash: HASH.toLowerCase(), time: KEPT.blockTime },
        new Date(Date.UTC(2026, 10, 9, 22, 31 + index)),
      );
      assert.equal(result.kind, "kept");
      if (result.kind !== "kept") return;
      assert.equal(result.block.hash, HASH, "a kept hash is stored in uppercase");
      assert.equal(keepResultMessage(result), `Kept block #${100 + index}.`);
      blocks = result.blocks;
    }
    assert.equal(blocks.length, 5);

    const sixth = keepBlock(blocks, { height: 200, hash: HASH, time: KEPT.blockTime }, new Date());
    assert.deepEqual(sixth, { kind: "full" });
    assert.equal(keepResultMessage(sixth), "You already keep five blocks. Forget one to keep another.");
    assert.equal(AT_LIMIT_MESSAGE, keepResultMessage(sixth));

    const again = keepBlock(blocks, { height: 102, hash: HASH, time: KEPT.blockTime }, new Date());
    assert.deepEqual(again, { kind: "already-kept", height: 102 });
    assert.equal(keepResultMessage(again), "Block #102 is already kept.");

    const sixEntries = [...blocks, { ...KEPT, height: 999 }];
    assert.deepEqual(parseKeptBlocks(JSON.stringify(sixEntries)), { readable: false });

    const fewer = forgetKeptBlock(blocks, 100);
    assert.deepEqual(fewer.map((block) => block.height), [101, 102, 103, 104]);
    assert.equal(keepBlock(fewer, { height: 200, hash: HASH, time: KEPT.blockTime }, new Date()).kind, "kept");

    assert.deepEqual(
      keepBlock([], { height: 7, hash: HASH, time: "not a time" }, new Date()),
      { kind: "incomplete" },
    );
  });

  it("reports still the same only for an equal hash at the same height and chain, ignoring hex case", () => {
    assert.deepEqual(assessKeptBlock(KEPT, reading()), { state: "same" });
    assert.deepEqual(
      assessKeptBlock(KEPT, reading({ block: { chainId: "zerone-1", height: KEPT.height, hash: HASH.toLowerCase() } })),
      { state: "same" },
    );
    // A stale status that still shows a lower tip does not override a block that came back equal.
    assert.deepEqual(assessKeptBlock(KEPT, reading({ latestHeight: KEPT.height - 1 })), { state: "same" });
    for (const block of [
      { chainId: "zerone-2", height: KEPT.height, hash: HASH },
      { chainId: "zerone-1", height: KEPT.height + 1, hash: HASH },
    ]) {
      assert.deepEqual(assessKeptBlock(KEPT, reading({ block })), { state: "unknown", reason: "incomplete" });
    }
    // Like DIFFERENT, it says what this site returned, not what the chain holds.
    assert.equal(
      wording({ state: "same" }),
      "Still the same. On 10 November 2026, 09:12 UTC, this site returned block #1,437,954 with the hash you kept on 9 November 2026, 22:31 UTC.",
    );
    // Times name their zone, because the copy and the download give UTC.
    assert.equal(formatKeeperDate("2026-11-09T14:05:00Z", "UTC"), "9 November 2026, 14:05 UTC");
    assert.equal(formatKeeperDate("2026-11-09T14:05:00Z", "Asia/Tokyo"), "9 November 2026, 23:05 GMT+9");
    assert.equal(formatKeeperDate(KEPT.blockTime, "UTC"), "9 November 2026, 22:29 UTC");
  });

  it("reports DIFFERENT for a changed hash, and for a missing block only when the read failed, the tip is lower and the node is not catching up", () => {
    const changed = assessKeptBlock(
      KEPT,
      reading({ block: { chainId: "zerone-1", height: KEPT.height, hash: OTHER_HASH } }),
    );
    assert.deepEqual(changed, { state: "different", reason: "hash" });
    assert.equal(
      wording(changed),
      "DIFFERENT. Block #1,437,954 now has another hash than the one you kept on 9 November 2026, 22:31 UTC. The record this site reads has changed at or before this block since then. Checked on 10 November 2026, 09:12 UTC.",
    );

    const tipBelow = KEPT.height - 400;
    const missing = assessKeptBlock(
      KEPT,
      reading({ block: null, readFailed: true, latestHeight: tipBelow, catchingUp: false }),
    );
    assert.deepEqual(missing, { state: "different", reason: "missing", latestHeight: tipBelow });
    assert.equal(
      wording(missing),
      "DIFFERENT. Block #1,437,954 is missing: the node’s newest block is now #1,437,554, lower than the block you kept on 9 November 2026, 22:31 UTC. Checked on 10 November 2026, 09:12 UTC.",
    );

    // Each of the three conditions is needed; without any one, it is not DIFFERENT.
    const notMissing: Array<[string, Partial<KeptBlockReading>, string]> = [
      ["the read answered, but incompletely", { block: null, readFailed: false, latestHeight: tipBelow }, "incomplete"],
      ["the tip is not lower", { block: null, readFailed: true, latestHeight: KEPT.height }, "no-answer"],
      ["the node is catching up", { block: null, readFailed: true, latestHeight: tipBelow, catchingUp: true }, "catching-up"],
      ["the status did not say", { block: null, readFailed: true, latestHeight: tipBelow, catchingUp: null }, "no-answer"],
      ["there was no status", { block: null, readFailed: true, latestHeight: null, catchingUp: null }, "no-answer"],
    ];
    for (const [name, overrides, reason] of notMissing) {
      assert.deepEqual(assessKeptBlock(KEPT, reading(overrides)), { state: "unknown", reason }, name);
    }
  });

  it("reports could not check for a failed read, a pruned height, a malformed reply or a node catching up, and never as the same", () => {
    const cases: Array<[string, Partial<KeptBlockReading>, string, string]> = [
      ["failed read", { block: null, readFailed: true }, "no-answer", "The node gave no usable answer."],
      ["pruned height", { block: null, readFailed: true, earliestHeight: KEPT.height + 1 }, "pruned", "The node no longer serves this block."],
      ["malformed reply", { block: null, readFailed: false }, "incomplete", "The reply was not a complete record of this block."],
      ["reply with a bad hash", { block: { chainId: "zerone-1", height: KEPT.height, hash: "not a hash" } }, "incomplete", "The reply was not a complete record of this block."],
      ["contradictory reading", { readFailed: true }, "incomplete", "The reply was not a complete record of this block."],
      ["catching up, with an equal hash", { catchingUp: true }, "catching-up", "The node is catching up."],
      ["no status, with an equal hash", { latestHeight: null, catchingUp: null }, "no-answer", "The node gave no usable answer."],
    ];
    for (const [name, overrides, reason, sentence] of cases) {
      const assessment = assessKeptBlock(KEPT, reading(overrides));
      assert.notEqual(assessment.state, "same", name);
      assert.deepEqual(assessment, { state: "unknown", reason }, name);
      assert.equal(
        wording(assessment),
        `Could not check. ${sentence} This does not mean the block is unchanged. Tried on 10 November 2026, 09:12 UTC.`,
        name,
      );
    }
  });

  it("says a block kept while saving fails lasts for this visit only", () => {
    assert.equal(
      describeView(KEPT, { kind: "kept" }, "saved", "UTC"),
      "Kept on 9 November 2026, 22:31 UTC. The page asks the node for this block again when you come back, or when you press Check again.",
    );
    assert.equal(
      describeView(KEPT, { kind: "kept" }, "unavailable", "UTC"),
      "Kept on 9 November 2026, 22:31 UTC, for this visit only. Press Check again to ask the node for it.",
    );
    assert.doesNotMatch(describeView(KEPT, { kind: "kept" }, "unavailable", "UTC"), /when you come back/u);
    assert.equal(
      describeView(KEPT, { kind: "checked", assessment: { state: "same" }, checkedAt: CHECKED_AT }, "unavailable", "UTC"),
      wording({ state: "same" }),
    );
  });

  it("keeps working when browser storage is missing or throws on access", () => {
    const unavailable = { blocks: [], storage: "unavailable", unreadable: false };

    // No window at all, as in this test runner.
    assert.deepEqual(readKeptBlocks(), unavailable);
    assert.equal(writeKeptBlocks([KEPT]), "unavailable");

    // Reading the property itself throws, as when site data is blocked.
    setWindow({
      get localStorage(): Storage {
        throw new Error("SecurityError");
      },
    });
    assert.deepEqual(readKeptBlocks(), unavailable);
    assert.equal(writeKeptBlocks([KEPT]), "unavailable");

    setWindow({ localStorage: null });
    assert.deepEqual(readKeptBlocks(), unavailable);
    assert.equal(writeKeptBlocks([KEPT]), "unavailable");

    const full = memoryStorage();
    full.setItem = () => {
      throw new Error("QuotaExceededError");
    };
    setWindow({ localStorage: full });
    assert.equal(writeKeptBlocks([KEPT]), "unavailable");

    // Working storage: exactly the five fields, under the one key.
    const storage = memoryStorage();
    setWindow({ localStorage: storage });
    assert.deepEqual(readKeptBlocks(), { blocks: [], storage: "saved", unreadable: false });
    assert.equal(writeKeptBlocks([{ ...KEPT, extra: "dropped" } as KeptBlock]), "saved");
    assert.deepEqual([...storage.values.keys()], [KEPT_BLOCKS_KEY]);
    assert.equal(KEPT_BLOCKS_KEY, "zerone.kept-blocks.v1");
    assert.deepEqual(JSON.parse(storage.values.get(KEPT_BLOCKS_KEY) ?? ""), [KEPT]);
    assert.deepEqual(readKeptBlocks(), { blocks: [KEPT], storage: "saved", unreadable: false });
    assert.equal(writeKeptBlocks([]), "saved");
    assert.equal(storage.values.size, 0, "forgetting the last block leaves nothing behind");

    // Unreadable data is reported, not silently dropped.
    storage.setItem(KEPT_BLOCKS_KEY, "{not json");
    assert.deepEqual(readKeptBlocks(), { blocks: [], storage: "saved", unreadable: true });
    assert.equal(storage.values.get(KEPT_BLOCKS_KEY), "{not json", "reading never rewrites storage");

    assert.equal(
      keeperNotice({ message: "", storage: "unavailable", unreadable: false }),
      "This browser is not letting the page save anything, so kept blocks last only until you leave. Copy or download them.",
    );
    assert.equal(STORAGE_BLOCKED_MESSAGE, keeperNotice({ message: "", storage: "unavailable", unreadable: false }));
    assert.equal(keeperNotice({ message: "", storage: "saved", unreadable: true }), UNREADABLE_MESSAGE);
    assert.equal(
      keeperNotice({ message: AT_LIMIT_MESSAGE, storage: "unavailable", unreadable: false }),
      `${AT_LIMIT_MESSAGE} ${STORAGE_BLOCKED_MESSAGE}`,
    );
    assert.equal(keeperNotice({ message: "", storage: "saved", unreadable: false }), "");
  });

  it("sums up a finished check in one status line", () => {
    assert.equal(checkSummary([]), "");
    assert.equal(checkSummary([{ state: "same" }]), "Checked 1 kept block: 1 still the same.");
    assert.equal(
      checkSummary([
        { state: "same" },
        { state: "unknown", reason: "no-answer" },
        { state: "different", reason: "hash" },
        { state: "same" },
        { state: "different", reason: "missing", latestHeight: 5 },
      ]),
      "Checked 5 kept blocks: 2 still the same, 2 DIFFERENT, 1 could not be checked.",
    );
    assert.equal(
      checkSummary([{ state: "unknown", reason: "catching-up" }, { state: "unknown", reason: "pruned" }]),
      "Checked 2 kept blocks: 2 could not be checked.",
    );
    assert.equal(keepResultMessage({ kind: "kept", blocks: [KEPT], block: KEPT }), "Kept block #1,437,954.");
  });

  it("keeps every block kept on this visit when storage reads but refuses writes", () => {
    const block = (height: number) => ({ height, hash: HASH, time: KEPT.blockTime });
    const heights = (page: KeptBlocksRead) => page.blocks.map((kept) => kept.height);

    // getItem works and setItem throws, as when the site's storage is full.
    const full = memoryStorage();
    full.setItem = () => {
      throw new Error("QuotaExceededError");
    };
    setWindow({ localStorage: full });
    let page = readKeptBlocks();
    assert.deepEqual(page, { blocks: [], storage: "saved", unreadable: false });
    for (const height of [100, 106, 112]) {
      const next = keepIntoPage(page, block(height), new Date(Date.UTC(2026, 10, 9, 22, 31)));
      assert.equal(next.result.kind, "kept", String(height));
      page = next.page;
      assert.equal(page.storage, "unavailable");
    }
    assert.deepEqual(heights(page), [100, 106, 112], "each Keep adds to the list, never replaces it");
    assert.equal(keeperNotice({ message: "", ...page }), STORAGE_BLOCKED_MESSAGE);
    assert.deepEqual(keepIntoPage(page, block(106), new Date()).result, { kind: "already-kept", height: 106 });
    assert.deepEqual(heights(keepIntoPage(page, block(106), new Date()).page), [100, 106, 112]);

    page = forgetFromPage(page, 112);
    assert.deepEqual(heights(page), [100, 106], "Forget removes one block, not the unsaved ones");
    assert.equal(page.storage, "unavailable");
    // Forgetting the last one removes the key, which works, so saving resumes.
    page = forgetFromPage(forgetFromPage(page, 100), 106);
    assert.deepEqual(page, { blocks: [], storage: "saved", unreadable: false });
    assert.equal(full.values.size, 0);

    // Storage already held a block from an earlier visit; a block that fails to save is not lost.
    const earlier = memoryStorage();
    earlier.setItem(KEPT_BLOCKS_KEY, JSON.stringify([KEPT]));
    earlier.setItem = () => {
      throw new Error("QuotaExceededError");
    };
    setWindow({ localStorage: earlier });
    page = readKeptBlocks();
    page = keepIntoPage(page, block(200), new Date()).page;
    page = keepIntoPage(page, block(201), new Date()).page;
    assert.deepEqual(heights(page), [KEPT.height, 200, 201]);
  });

  it("starts a Keep or Forget from what another tab saved while this page's writes land", () => {
    const storage = memoryStorage();
    setWindow({ localStorage: storage });
    let page = keepIntoPage(readKeptBlocks(), { height: 100, hash: HASH, time: KEPT.blockTime }, new Date()).page;
    assert.equal(page.storage, "saved");
    // Another tab keeps a block of its own.
    storage.setItem(KEPT_BLOCKS_KEY, JSON.stringify([...page.blocks, KEPT]));
    page = keepIntoPage(page, { height: 300, hash: HASH, time: KEPT.blockTime }, new Date()).page;
    assert.deepEqual(page.blocks.map((kept) => kept.height), [100, KEPT.height, 300]);
    assert.deepEqual(parseKeptBlocks(storage.values.get(KEPT_BLOCKS_KEY)), { readable: true, blocks: page.blocks });

    // Unreadable data under the key is replaced by the first write that lands.
    storage.setItem(KEPT_BLOCKS_KEY, "{not json");
    page = readKeptBlocks();
    assert.deepEqual(page, { blocks: [], storage: "saved", unreadable: true });
    page = keepIntoPage(page, { height: 7, hash: HASH, time: KEPT.blockTime }, new Date()).page;
    assert.equal(page.unreadable, false);
    assert.deepEqual(page.blocks.map((kept) => kept.height), [7]);
  });

  it("counts a reply that is not this block as an answer, and anything else that throws as a failed read", async () => {
    const observed = { chainId: "zerone-1", height: KEPT.height, hash: HASH };
    assert.deepEqual(await readKeptHeight(KEPT.height, async () => observed), { block: observed, readFailed: false });
    const throws = (error: unknown) => async (): Promise<never> => {
      throw error;
    };
    const timeout = new Error("The operation was aborted due to timeout");
    timeout.name = "TimeoutError";
    const cases: Array<[string, unknown, boolean]> = [
      ["a reply that is not exactly this block", new IncompleteBlockReplyError("another chain"), false],
      ["HTTP 500", new Error("Mainnet returned HTTP 500"), true],
      ["no answer", new TypeError("Failed to fetch"), true],
      ["a timeout", timeout, true],
      ["an unreadable body", new Error("Mainnet returned malformed JSON."), true],
      ["something that is not an Error", { name: "IncompleteBlockReplyError" }, true],
    ];
    for (const [name, error, readFailed] of cases) {
      assert.deepEqual(await readKeptHeight(KEPT.height, throws(error)), { block: null, readFailed }, name);
    }
  });

  it("writes a download naming chain, height, hash, block time, kept time and the raw lookup link", () => {
    const second: KeptBlock = { ...KEPT, height: 1, hash: OTHER_HASH, keptAt: "2026-11-09T22:40:00.000Z" };
    const text = keptBlocksDownload([KEPT, second], "https://zerone.ai");
    assert.ok(text.endsWith("\n"));
    assert.deepEqual(JSON.parse(text), [
      {
        chain_id: "zerone-1",
        height: 1_437_954,
        hash: HASH,
        block_time: "2026-11-09T22:29:50.123456789Z",
        kept_at: "2026-11-09T22:31:00.000Z",
        look_up: "https://zerone.ai/api/rpc/block?height=1437954",
      },
      {
        chain_id: "zerone-1",
        height: 1,
        hash: OTHER_HASH,
        block_time: "2026-11-09T22:29:50.123456789Z",
        kept_at: "2026-11-09T22:40:00.000Z",
        look_up: "https://zerone.ai/api/rpc/block?height=1",
      },
    ]);
    assert.equal(keptBlocksDownload([], "https://zerone.ai"), "[]\n");
  });
});
