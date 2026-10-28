import { describe, expect, test } from "bun:test";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { addressPayloadHex } from "../src/bech32.ts";
import type { Allowlist } from "../src/config.ts";
import {
  checkMembership,
  didSpellings,
  evaluateGates,
  haltFilesPresent,
  registrarLiability,
  seedCounts,
  type GateInput,
} from "../src/gates.ts";
import type { LedgerEntry } from "../src/ledger.ts";
import {
  ADMISSION_CEILING_UZRN,
  DOOR_DAILY_CAP,
  DOOR_WINDOW_CAP,
  REGISTRAR_RESERVE_UZRN,
  UNCLAIMED_LIVE_SEED_CAP,
} from "../src/policy.ts";
import { parseDoorRequest, type DoorRequest } from "../src/request.ts";
import {
  addressFor,
  baseGlobal,
  baseSubject,
  grantFromR,
  HOUSEHOLD,
  ledgerWith,
  makeRequest,
  NOW,
  R,
  TEST_PIN,
  unclaimedPot,
} from "./helpers.ts";

const made = makeRequest();
const req = (() => {
  const p = parseDoorRequest(made.text);
  if (!p.ok) throw new Error("fixture");
  return p.value;
})() as DoorRequest;

const allowAll: Allowlist = { schema: "zerone-door-allowlist/v1", studio_local: { enabled: true }, lobby_desks: [] };

function input(over: Partial<GateInput> = {}): GateInput {
  return {
    request: req,
    requestReasons: [],
    membership: checkMembership({ kind: "studio_local", file: "a.request" }, allowAll),
    registrar: R,
    chainVersionPin: TEST_PIN,
    knownKeys: [HOUSEHOLD],
    ledger: ledgerWith(),
    global: baseGlobal(),
    subject: baseSubject(),
    haltFiles: [],
    now: NOW,
    chainId: "zerone-1",
    ...over,
  };
}

function entry(over: Partial<LedgerEntry> = {}): LedgerEntry {
  return {
    request_id: "e".repeat(64),
    kind: "admit",
    address: addressFor("someone-else"),
    address_payload: addressPayloadHex(addressFor("someone-else"))!,
    identity_public_key: "11".repeat(32),
    did: `did:zrn:${"11".repeat(32)}`,
    channel_key: "lobby:00000000-0000-4000-8000-000000000000",
    tx_hash: "A".repeat(64),
    status: "included",
    chain_window: Math.floor(1_402_800 / 34_272),
    utc_day: "2026-10-28",
    at: "2026-10-28T10:00:00.000Z",
    grant_expiration: "2026-11-11T10:00:00Z",
    ...over,
  };
}

function only(over: Partial<GateInput>) {
  return evaluateGates(input(over)).reasons;
}

describe("the baseline", () => {
  test("a fresh agent on an allowed channel with R in order is admitted", () => {
    expect(evaluateGates(input())).toEqual({ kind: "admit", reasons: [], revokeFirst: false });
  });
});

describe("system gates", () => {
  test("a HALT or rung-window.freeze file stops everything", () => {
    expect(only({ haltFiles: ["/x/HALT"] })).toEqual(["halt_file"]);
    const dir = mkdtempSync(join(tmpdir(), "zerone-door-halt-"));
    writeFileSync(join(dir, "rung-window.freeze"), "");
    expect(haltFilesPresent([join(dir, "HALT"), join(dir, "rung-window.freeze")])).toEqual([join(dir, "rung-window.freeze")]);
  });

  test("a pending upgrade plan stops admissions", () => {
    expect(only({ global: baseGlobal({ upgradePending: true }) })).toEqual(["upgrade_plan_pending"]);
  });

  test("a chain version other than the pinned one stops admissions until a person re-pins", () => {
    const pinned = baseGlobal().chainVersion;
    const upgraded = { ...pinned, lastUpgrade: { name: "h1-v1", height: 1_500_000 } };
    expect(only({ global: baseGlobal({ chainVersion: upgraded }) })).toEqual(["chain_version_changed"]);
    const migrated = { ...pinned, moduleVersionsSha256: "0".repeat(64) };
    expect(only({ global: baseGlobal({ chainVersion: migrated }) })).toEqual(["chain_version_changed"]);
    expect(only({ global: baseGlobal({ chainVersion: { ...pinned, lastUpgrade: null } }) })).toEqual(["chain_version_changed"]);
    // Re-pinning to what the chain now runs opens the door again.
    expect(only({ global: baseGlobal({ chainVersion: upgraded }), chainVersionPin: { ...TEST_PIN, last_upgrade: "h1-v1" } })).toEqual([]);
  });

  test("the on-chain registrar must be R", () => {
    expect(only({ global: baseGlobal({ params: { registrar: HOUSEHOLD, windowCap: 25 } }) })).toEqual(["registrar_not_r"]);
  });

  test("R must be registered, not frozen, and have a public key on chain", () => {
    expect(only({ global: baseGlobal({ registrarZerone: null }) })).toEqual(["registrar_not_registered"]);
    const frozen = baseGlobal();
    frozen.registrarZerone = { ...frozen.registrarZerone!, frozen: true };
    expect(only({ global: frozen })).toEqual(["registrar_frozen"]);
    expect(only({ global: baseGlobal({ registrarAuth: null }) })).toEqual(["registrar_no_public_key"]);
  });

  test("a halted chain or another network refuses", () => {
    expect(only({ global: baseGlobal({ status: { network: "zerone-1", height: 1_402_800, time: "2026-10-28T17:00:00Z" } }) }))
      .toEqual(["chain_stale"]);
    expect(only({ global: baseGlobal({ status: { network: "zerone-2", height: 1_402_800, time: "2026-10-28T17:30:50Z" } }) }))
      .toEqual(["chain_id_mismatch_node"]);
  });
});

describe("membership", () => {
  const desk = "3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d";
  const lobby = { kind: "lobby" as const, room_id: `room_${"a1".repeat(16)}`, identity_id: desk, fingerprint: "f", event_id: "ev" };

  test("Studio-local requests follow the allowlist's switch", () => {
    expect(checkMembership({ kind: "studio_local", file: "x" }, { ...allowAll, studio_local: { enabled: false } }).reasons)
      .toEqual(["channel_not_allowed"]);
  });

  test("a lobby request needs a door-roster-verified current member that the allowlist names", () => {
    const listed: Allowlist = { ...allowAll, lobby_desks: [{ identity_id: desk, daily_cap: 2 }] };
    expect(checkMembership({ ...lobby, verified: true, currentMember: true }, listed)).toEqual({
      reasons: [], channelKey: `lobby:${desk}`, channelCap: 2,
    });
    expect(checkMembership({ ...lobby, verified: true, currentMember: true }, allowAll).reasons).toEqual(["lobby_sender_not_allowed"]);
    expect(checkMembership({ ...lobby, verified: false, currentMember: true }, listed).reasons).toEqual(["lobby_sender_not_verified"]);
    expect(checkMembership({ ...lobby, verified: true, currentMember: false }, listed).reasons).toEqual(["lobby_sender_not_verified"]);
  });

  test("membership refusals carry through the gates", () => {
    const m = checkMembership({ kind: "studio_local", file: "x" }, { ...allowAll, studio_local: { enabled: false } });
    expect(only({ membership: m })).toEqual(["channel_not_allowed"]);
  });
});

describe("the subject: not a known key, no history", () => {
  test("a household, retiring or known key is refused, compared by payload", () => {
    expect(only({ knownKeys: [made.address] })).toEqual(["known_key"]);
    expect(only({ knownKeys: [made.address.toUpperCase()] })).toEqual(["known_key"]);
    expect(only({ registrar: made.address, global: baseGlobal({ params: { registrar: made.address, windowCap: 25 } }) }))
      .toContain("known_key");
  });

  test("any balance, account, zerone_auth record, pot or grant is refused", () => {
    expect(only({ subject: baseSubject({ balances: [{ denom: "uzrn", amount: "1" }] }) })).toEqual(["has_balance"]);
    expect(only({ subject: baseSubject({ balances: [{ denom: "ibc/ABC", amount: "5" }] }) })).toEqual(["has_balance"]);
    expect(only({ subject: baseSubject({ auth: { accountNumber: "9", sequence: "0", pubKeyType: null, pubKeyB64: null } }) }))
      .toEqual(["has_account"]);
    expect(only({ subject: baseSubject({ zerone: { address: made.address, did: "did:zrn:x", accountType: "agent", frozen: false } }) }))
      .toEqual(["has_zerone_auth_record"]);
    expect(only({ subject: baseSubject({ potLower: unclaimedPot(made.address) }) })).toEqual(["has_pot"]);
    expect(only({ subject: baseSubject({ potUpper: unclaimedPot(made.address.toUpperCase()) }) })).toEqual(["has_pot"]);
    expect(only({ subject: baseSubject({ grantsToAddress: [{ ...grantFromR(made.address), granter: HOUSEHOLD }] }) }))
      .toEqual(["has_grant"]);
  });

  test("a DID registered in any spelling is refused", () => {
    const spellings = didSpellings(made.identityHex);
    expect(spellings).toEqual([
      `did:zrn:${made.identityHex}`,
      `did:zrn:${made.identityHex.toUpperCase()}`,
      `did:zrn:${made.identityHex.slice(0, 32)}`,
      `did:zrn:${made.identityHex.toUpperCase().slice(0, 32)}`,
    ]);
    for (const did of spellings) expect(only({ subject: baseSubject({ didRegistered: [did] }) })).toEqual(["did_registered"]);
  });

  test("an address, identity key or DID the door admitted before is refused", () => {
    const byAddress = entry({ address: made.address, address_payload: addressPayloadHex(made.address)!, identity_public_key: "22".repeat(32), did: `did:zrn:${"22".repeat(32)}` });
    expect(only({ ledger: ledgerWith({ entries: [byAddress] }) })).toEqual(["already_admitted"]);
    const byIdentity = entry({ identity_public_key: made.identityHex, did: `did:zrn:${made.identityHex}` });
    expect(only({ ledger: ledgerWith({ entries: [byIdentity] }) })).toEqual(["already_admitted"]);
    const rejected = entry({ address: made.address, address_payload: addressPayloadHex(made.address)!, status: "rejected" });
    expect(only({ ledger: ledgerWith({ entries: [rejected] }) })).toEqual([]);
  });

  test("a request already seen is a replay", () => {
    expect(only({ ledger: ledgerWith({ seen: { [req.requestId]: "2026-10-28T17:00:00Z" } }) })).toEqual(["request_seen"]);
  });
});

describe("re-grant of an expired, unclaimed admittee, once", () => {
  const admitted = entry({
    address: made.address,
    address_payload: addressPayloadHex(made.address)!,
    identity_public_key: made.identityHex,
    did: `did:zrn:${made.identityHex}`,
    utc_day: "2026-10-01",
    chain_window: 1,
    at: "2026-10-01T00:00:00.000Z",
    // Fourteen days after admission: before the block time of 2026-10-28.
    grant_expiration: "2026-10-15T00:00:00Z",
  });
  const lapsedSubject = baseSubject({
    auth: { accountNumber: "99", sequence: "0", pubKeyType: null, pubKeyB64: null },
    potLower: unclaimedPot(made.address),
  });
  // Admitted today: its grant runs until 2026-11-11.
  const admittedToday = { ...admitted, utc_day: "2026-10-28", at: "2026-10-28T10:00:00.000Z", grant_expiration: "2026-11-11T10:00:00Z" };
  const spentSubject = { ...lapsedSubject, auth: { accountNumber: "99", sequence: "3", pubKeyType: null, pubKeyB64: null } };

  test("an admittee whose seed is unclaimed and whose allowance lapsed (and was pruned) gets a re-grant", () => {
    expect(evaluateGates(input({ ledger: ledgerWith({ entries: [admitted] }), subject: lapsedSubject })))
      .toEqual({ kind: "regrant", reasons: [], revokeFirst: false });
  });

  test("an expired grant that is still stored is revoked in the same transaction", () => {
    const expired = grantFromR(made.address, "2026-10-20T00:00:00Z");
    const verdict = evaluateGates(input({
      ledger: ledgerWith({ entries: [admitted] }),
      subject: { ...lapsedSubject, grantsToAddress: [expired] },
      global: baseGlobal({ registrarIssued: [expired] }),
    }));
    expect(verdict).toEqual({ kind: "regrant", reasons: [], revokeFirst: true });
  });

  test("only once, only while unclaimed, only while the allowance is not live, only for the same keys", () => {
    const regranted = { ...admitted, request_id: "f".repeat(64), kind: "regrant" as const };
    // A refused admittee also has an account and a pot: every reason is listed.
    expect(only({ ledger: ledgerWith({ entries: [admitted, regranted] }), subject: lapsedSubject })).toContain("already_admitted");
    const claimed = { ...lapsedSubject, potLower: { ...unclaimedPot(made.address), status: "POT_STATUS_DEPLETED", claimed: 222_000n } };
    expect(only({ ledger: ledgerWith({ entries: [admitted] }), subject: claimed })).toContain("already_admitted");
    const live = grantFromR(made.address);
    expect(only({ ledger: ledgerWith({ entries: [admitted] }), subject: lapsedSubject, global: baseGlobal({ registrarIssued: [live] }) }))
      .toContain("already_admitted");
    const otherIdentity = { ...admitted, identity_public_key: "33".repeat(32), did: `did:zrn:${"33".repeat(32)}` };
    expect(only({ ledger: ledgerWith({ entries: [otherIdentity] }), subject: lapsedSubject })).toContain("already_admitted");
  });

  test("a grant drained to zero before its expiry is not a lapse: refused", () => {
    // feegrant deletes a grant spent to zero, so none is on chain; the agent
    // signed the transactions that spent it (sequence 3).
    const verdict = evaluateGates(input({ ledger: ledgerWith({ entries: [admittedToday] }), subject: spentSubject }));
    expect(verdict.kind).toBe("refuse");
    expect(verdict.reasons).toContain("already_admitted");
  });

  test("a grant revoked before its expiry is not a lapse: refused", () => {
    // A revoke deletes the grant too; this agent never used it (sequence 0).
    const verdict = evaluateGates(input({ ledger: ledgerWith({ entries: [admittedToday] }), subject: lapsedSubject }));
    expect(verdict.kind).toBe("refuse");
    expect(verdict.reasons).toContain("already_admitted");
  });

  test("a grant drained and then past its expiry is still not a lapse: refused", () => {
    const verdict = evaluateGates(input({ ledger: ledgerWith({ entries: [admitted] }), subject: spentSubject }));
    expect(verdict.kind).toBe("refuse");
    expect(verdict.reasons).toContain("already_admitted");
  });
});

describe("capacity", () => {
  test("the door's daily cap counts every channel", () => {
    const today = Array.from({ length: DOOR_DAILY_CAP }, (_, i) => entry({ request_id: String(i).padStart(64, "0"), channel_key: `lobby:${i}` }));
    expect(only({ ledger: ledgerWith({ entries: today }) })).toEqual(["door_daily_cap"]);
  });

  test("each channel has its own daily cap", () => {
    const mine = Array.from({ length: 3 }, (_, i) => entry({ request_id: String(i).padStart(64, "0"), channel_key: "studio_local" }));
    expect(only({ ledger: ledgerWith({ entries: mine }) })).toEqual(["channel_daily_cap"]);
  });

  test("the door's window cap counts admissions in this chain window", () => {
    const window = Array.from({ length: DOOR_WINDOW_CAP }, (_, i) => entry({ request_id: String(i).padStart(64, "0"), utc_day: "2026-10-20" }));
    expect(only({ ledger: ledgerWith({ entries: window }) })).toEqual(["door_window_cap"]);
  });

  test("the chain window must have room (the stored count resets with each window)", () => {
    const current = Math.floor(1_402_800 / 34_272);
    expect(only({ global: baseGlobal({ window: { height: 1_402_800, storedWindow: current, storedCount: 25 } }) })).toEqual(["chain_window_full"]);
    expect(only({ global: baseGlobal({ window: { height: 1_402_800, storedWindow: current - 1, storedCount: 25 } }) })).toEqual([]);
  });

  test("only unclaimed seeds with a live allowance from R count toward the seed cap", () => {
    const live = Array.from({ length: UNCLAIMED_LIVE_SEED_CAP }, (_, i) => addressFor(`live-${i}`));
    const lapsed = Array.from({ length: 30 }, (_, i) => addressFor(`lapsed-${i}`));
    const global = baseGlobal({
      pots: [...live, ...lapsed].map((a) => unclaimedPot(a)),
      registrarIssued: [
        ...live.map((a) => grantFromR(a, "2026-11-11T00:00:00Z", 1n)),
        ...lapsed.map((a) => grantFromR(a, "2026-10-01T00:00:00Z", 1n)),
      ],
    });
    expect(seedCounts(global, R)).toEqual({ unclaimedLive: UNCLAIMED_LIVE_SEED_CAP, unclaimedLapsed: lapsed });
    expect(only({ global })).toEqual(["unclaimed_seed_cap"]);
    const fewer = { ...global, registrarIssued: global.registrarIssued.slice(1) };
    expect(only({ global: fewer })).toEqual([]);
  });

  test("R must cover every live allowance, this admission and a 2 ZRN reserve", () => {
    const outstanding = [grantFromR(addressFor("a"), "2026-11-11T00:00:00Z", 1_000_000n), grantFromR(addressFor("b"), "2026-10-01T00:00:00Z", 9_000_000n)];
    expect(registrarLiability(baseGlobal({ registrarIssued: outstanding }))).toBe(1_000_000n);
    const exact = BigInt(1_000_000 + ADMISSION_CEILING_UZRN + REGISTRAR_RESERVE_UZRN);
    expect(only({ global: baseGlobal({ registrarIssued: outstanding, registrarSpendable: exact }) })).toEqual([]);
    expect(only({ global: baseGlobal({ registrarIssued: outstanding, registrarSpendable: exact - 1n }) })).toEqual(["registrar_balance_low"]);
  });

  test("an unbounded allowance from R stops the door", () => {
    const unbounded = grantFromR(addressFor("u"));
    unbounded.allowance = { ...unbounded.allowance, spendLimitUzrn: null };
    expect(only({ global: baseGlobal({ registrarIssued: [unbounded] }) })).toEqual(["registrar_has_unbounded_grant"]);
  });
});

describe("one transaction at a time", () => {
  test("an unconfirmed earlier admission makes the door wait", () => {
    expect(only({ ledger: ledgerWith({ entries: [entry({ status: "pending_inclusion", utc_day: "2026-10-27" })] }) })).toEqual(["door_busy"]);
  });
});

describe("reasons accumulate", () => {
  test("a refusal lists every reason that applies", () => {
    const reasons = only({
      haltFiles: ["/x/HALT"],
      subject: baseSubject({ balances: [{ denom: "uzrn", amount: "5" }], didRegistered: [`did:zrn:${made.identityHex}`] }),
      global: baseGlobal({ upgradePending: true }),
    });
    expect(reasons.sort()).toEqual(["did_registered", "halt_file", "has_balance", "upgrade_plan_pending"]);
  });
});
