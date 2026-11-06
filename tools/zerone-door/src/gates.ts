// Every admission gate. Facts are read first (gatherGlobalFacts,
// gatherSubjectFacts); evaluateGates is pure, so each gate has a unit test.
import { lstatSync } from "node:fs";
import { addressPayloadHex } from "./bech32.ts";
import type { AuthAccount, ChainReader, ChainVersion, Coin, Grant, NodeStatus, Pot, PotParams, Read, WindowState, ZeroneAccount } from "./chain.ts";
import type { Allowlist, ChainVersionPin } from "./config.ts";
import { counted, HUB_CHANNEL_KEY, utcDay, type ChannelRef, type Ledger, type LedgerEntry } from "./ledger.ts";
import {
  ADMISSION_CEILING_UZRN,
  BOOTSTRAP_WINDOW_BLOCKS,
  CHAIN_STALE_SECONDS,
  CHANNEL_DAILY_CAP_DEFAULT,
  DOOR_DAILY_CAP,
  DOOR_WINDOW_CAP,
  REGISTRAR_RESERVE_UZRN,
  REGRANT_CEILING_UZRN,
  UNCLAIMED_LIVE_SEED_CAP,
} from "./policy.ts";
import type { DoorRequest, RequestReason } from "./request.ts";

export type GateReason =
  | RequestReason
  | "request_seen"
  | "channel_not_allowed"
  | "lobby_sender_not_allowed"
  | "lobby_sender_not_verified"
  | "halt_file"
  | "chain_read_failed"
  | "chain_stale"
  | "chain_id_mismatch_node"
  | "chain_version_changed"
  | "upgrade_plan_pending"
  | "registrar_not_r"
  | "registrar_not_registered"
  | "registrar_frozen"
  | "registrar_no_public_key"
  | "registrar_has_unbounded_grant"
  | "registrar_balance_low"
  | "door_busy"
  | "door_daily_cap"
  | "channel_daily_cap"
  | "door_window_cap"
  | "chain_window_full"
  | "unclaimed_seed_cap"
  | "known_key"
  | "already_admitted"
  | "has_balance"
  | "has_account"
  | "has_zerone_auth_record"
  | "has_pot"
  | "has_grant"
  | "did_registered";

export interface GlobalFacts {
  status: NodeStatus;
  params: PotParams;
  window: WindowState;
  upgradePending: boolean;
  chainVersion: ChainVersion;
  registrarSpendable: bigint;
  registrarAuth: AuthAccount | null;
  registrarZerone: ZeroneAccount | null;
  registrarIssued: Grant[];
  pots: Pot[];
}

export interface SubjectFacts {
  balances: Coin[];
  auth: AuthAccount | null;
  zerone: ZeroneAccount | null;
  potLower: Pot | null;
  potUpper: Pot | null;
  grantsToAddress: Grant[];
  /** DID spellings found registered: lowercase/uppercase x 64/32 hex. */
  didRegistered: string[];
}

export type Channel = ChannelRef & { verified?: boolean; currentMember?: boolean };

export interface MembershipVerdict {
  reasons: GateReason[];
  channelKey: string;
  channelCap: number;
}

export function channelKeyOf(channel: ChannelRef): string {
  return channel.kind === "hub_local" ? HUB_CHANNEL_KEY : `lobby:${channel.identity_id}`;
}

/** Eligibility: a household allowlist decides who may ask; the lobby's
 * door-signed roster proves that a remote request really came from that desk. */
export function checkMembership(channel: Channel, allowlist: Allowlist): MembershipVerdict {
  const channelKey = channelKeyOf(channel);
  if (channel.kind === "hub_local") {
    if (!allowlist.hub_local.enabled) return { reasons: ["channel_not_allowed"], channelKey, channelCap: 0 };
    return { reasons: [], channelKey, channelCap: allowlist.hub_local.daily_cap ?? CHANNEL_DAILY_CAP_DEFAULT };
  }
  const reasons: GateReason[] = [];
  if (channel.verified !== true || channel.currentMember !== true) reasons.push("lobby_sender_not_verified");
  const desk = allowlist.lobby_desks.find((d) => d.identity_id === channel.identity_id);
  if (!desk) reasons.push("lobby_sender_not_allowed");
  return { reasons, channelKey, channelCap: desk?.daily_cap ?? CHANNEL_DAILY_CAP_DEFAULT };
}

export function haltFilesPresent(paths: readonly string[]): string[] {
  return paths.filter((p) => {
    try {
      lstatSync(p);
      return true;
    } catch {
      return false;
    }
  });
}

export function didSpellings(identityPublicKeyHex: string): string[] {
  const lower = identityPublicKeyHex.toLowerCase();
  const upper = identityPublicKeyHex.toUpperCase();
  return [`did:zrn:${lower}`, `did:zrn:${upper}`, `did:zrn:${lower.slice(0, 32)}`, `did:zrn:${upper.slice(0, 32)}`];
}

function unwrap<T>(r: Read<T>, errors: string[]): T | undefined {
  if (r.ok) return r.value;
  errors.push(r.error);
  return undefined;
}

export async function gatherGlobalFacts(chain: ChainReader, registrar: string): Promise<{ facts: GlobalFacts | null; errors: string[] }> {
  const errors: string[] = [];
  const status = unwrap(await chain.status(), errors);
  const params = unwrap(await chain.potParams(), errors);
  const window = unwrap(await chain.windowState(), errors);
  const upgradePending = unwrap(await chain.upgradePlanPending(), errors);
  const chainVersion = unwrap(await chain.chainVersion(), errors);
  const registrarSpendable = unwrap(await chain.spendableUzrn(registrar), errors);
  const registrarAuth = unwrap(await chain.authAccount(registrar), errors);
  const registrarZerone = unwrap(await chain.zeroneAccount(registrar), errors);
  const registrarIssued = unwrap(await chain.grantsByGranter(registrar), errors);
  const pots = unwrap(await chain.allPots(), errors);
  if (errors.length > 0) return { facts: null, errors };
  return {
    facts: {
      status: status!,
      params: params!,
      window: window!,
      upgradePending: upgradePending!,
      chainVersion: chainVersion!,
      registrarSpendable: registrarSpendable!,
      registrarAuth: registrarAuth ?? null,
      registrarZerone: registrarZerone ?? null,
      registrarIssued: registrarIssued!,
      pots: pots!,
    },
    errors,
  };
}

export async function gatherSubjectFacts(chain: ChainReader, req: DoorRequest): Promise<{ facts: SubjectFacts | null; errors: string[] }> {
  const errors: string[] = [];
  const address = req.address;
  const balances = unwrap(await chain.balances(address), errors);
  const auth = unwrap(await chain.authAccount(address), errors);
  const zerone = unwrap(await chain.zeroneAccount(address), errors);
  const potLower = unwrap(await chain.pot(`bootstrap-${address}`), errors);
  const potUpper = unwrap(await chain.pot(`bootstrap-${address.toUpperCase()}`), errors);
  const grantsToAddress = unwrap(await chain.grantsByGrantee(address), errors);
  const didRegistered: string[] = [];
  for (const did of didSpellings(req.identityPublicKeyHex)) {
    const hit = unwrap(await chain.accountByDid(did), errors);
    if (hit) didRegistered.push(did);
  }
  if (errors.length > 0) return { facts: null, errors };
  return {
    facts: {
      balances: balances!,
      auth: auth ?? null,
      zerone: zerone ?? null,
      potLower: potLower ?? null,
      potUpper: potUpper ?? null,
      grantsToAddress: grantsToAddress!,
      didRegistered,
    },
    errors,
  };
}

export function grantIsLive(grant: Grant, blockTime: Date): boolean {
  const exp = grant.allowance.expiration;
  return exp === null || Date.parse(exp) > blockTime.getTime();
}

export function isUnclaimedBootstrap(pot: Pot): boolean {
  return pot.id.startsWith("bootstrap-") && pot.status === "POT_STATUS_ACTIVE" && pot.claimed < pot.total;
}

export interface SeedCounts {
  /** Unclaimed seeds whose allowance from R is still live: the gate's count. */
  unclaimedLive: number;
  /** Unclaimed seeds with no live allowance from R: an alert, never a stop. */
  unclaimedLapsed: string[];
}

export function seedCounts(global: GlobalFacts, registrar: string): SeedCounts {
  const blockTime = new Date(global.status.time);
  const live = new Set(
    global.registrarIssued.filter((g) => g.granter === registrar && grantIsLive(g, blockTime)).map((g) => g.grantee),
  );
  let unclaimedLive = 0;
  const unclaimedLapsed: string[] = [];
  for (const pot of global.pots) {
    if (!isUnclaimedBootstrap(pot)) continue;
    const owner = pot.id.slice("bootstrap-".length);
    if (live.has(owner)) unclaimedLive += 1;
    else unclaimedLapsed.push(owner);
  }
  return { unclaimedLive, unclaimedLapsed };
}

/** Remaining spend R has promised through live allowances, or null when one
 * is unbounded (the door never issues one; refuse rather than guess). */
export function registrarLiability(global: GlobalFacts): bigint | null {
  const blockTime = new Date(global.status.time);
  let total = 0n;
  for (const g of global.registrarIssued) {
    if (!grantIsLive(g, blockTime)) continue;
    if (g.allowance.spendLimitUzrn === null) return null;
    total += g.allowance.spendLimitUzrn;
  }
  return total;
}

export function currentWindow(height: number): number {
  return Math.floor(height / BOOTSTRAP_WINDOW_BLOCKS);
}

/** True when the node runs the chain version the door was measured on. */
export function chainVersionMatches(observed: ChainVersion, pin: ChainVersionPin): boolean {
  return observed.lastUpgrade?.name === pin.last_upgrade && observed.moduleVersionsSha256 === pin.module_versions_sha256;
}

export interface GateInput {
  request: DoorRequest;
  requestReasons: RequestReason[];
  membership: MembershipVerdict;
  registrar: string;
  chainVersionPin: ChainVersionPin;
  knownKeys: readonly string[];
  ledger: Ledger;
  global: GlobalFacts;
  subject: SubjectFacts;
  haltFiles: readonly string[];
  now: Date;
  chainId: string;
}

export interface GateVerdict {
  kind: "admit" | "regrant" | "refuse";
  reasons: GateReason[];
  /** Set when a re-grant must first revoke an expired, unpruned grant. */
  revokeFirst: boolean;
}

function samePayload(a: string, b: string): boolean {
  const pa = addressPayloadHex(a);
  return pa !== null && pa === addressPayloadHex(b);
}

function priorAdmissions(ledger: Ledger, req: DoorRequest, payload: string): LedgerEntry[] {
  return counted(ledger).filter((e) =>
    e.address_payload === payload || e.identity_public_key === req.identityPublicKeyHex || e.did === req.did);
}

export function evaluateGates(input: GateInput): GateVerdict {
  const { request: req, global, subject, ledger, registrar } = input;
  const reasons = new Set<GateReason>([...input.requestReasons, ...input.membership.reasons]);
  if (ledger.seen[req.requestId] !== undefined) reasons.add("request_seen");

  // ── the door and the chain must be in a state to admit anyone ──
  if (input.haltFiles.length > 0) reasons.add("halt_file");
  const blockTime = new Date(global.status.time);
  if ((input.now.getTime() - blockTime.getTime()) / 1000 > CHAIN_STALE_SECONDS) reasons.add("chain_stale");
  if (global.status.network !== input.chainId) reasons.add("chain_id_mismatch_node");
  if (!chainVersionMatches(global.chainVersion, input.chainVersionPin)) reasons.add("chain_version_changed");
  if (global.upgradePending) reasons.add("upgrade_plan_pending");
  if (global.params.registrar !== registrar) reasons.add("registrar_not_r");
  if (global.registrarZerone === null) reasons.add("registrar_not_registered");
  else if (global.registrarZerone.frozen) reasons.add("registrar_frozen");
  if (global.registrarAuth === null || global.registrarAuth.pubKeyB64 === null) reasons.add("registrar_no_public_key");

  // ── who is asking: fresh account, or an earlier admittee whose seed lapsed ──
  const payload = addressPayloadHex(req.address) ?? "";
  const prior = payload ? priorAdmissions(ledger, req, payload) : [];
  let kind: "admit" | "regrant" = "admit";
  let revokeFirst = false;
  const rGrant = global.registrarIssued.find((g) => g.granter === registrar && g.grantee === req.address) ?? null;
  if (prior.length > 0) {
    const sameAgent = prior.every((e) => e.address_payload === payload && e.identity_public_key === req.identityPublicKeyHex);
    const alreadyRegranted = prior.some((e) => e.kind === "regrant");
    const admitted = prior.find((e) => e.kind === "admit" && e.status === "included");
    const potUnclaimed = subject.potLower !== null && isUnclaimedBootstrap(subject.potLower);
    const grantLive = rGrant !== null && grantIsLive(rGrant, blockTime);
    // A re-grant answers a grant that ran out by time, never one that was
    // spent or taken away. The grant the door recorded must have expired at
    // block time: a grant missing before then was drained to zero (feegrant
    // deletes it) or revoked. And it must have gone unused: only transactions
    // the agent signs can draw on it, so an account sequence above 0 means the
    // agent spent from it, which leaves no other trace once the grant is pruned.
    const expiredByTime = admitted !== undefined && Date.parse(admitted.grant_expiration) <= blockTime.getTime();
    const unused = subject.auth === null || subject.auth.sequence === "0";
    if (sameAgent && !alreadyRegranted && admitted && potUnclaimed && !grantLive && expiredByTime && unused) {
      kind = "regrant";
      revokeFirst = rGrant !== null;
    } else {
      reasons.add("already_admitted");
    }
  }

  // ── the subject: never a household key, never an account with history ──
  const known = [registrar, ...input.knownKeys];
  if (known.some((k) => samePayload(k, req.address))) reasons.add("known_key");
  if (subject.balances.some((c) => /^\d+$/.test(c.amount) && BigInt(c.amount) > 0n)) reasons.add("has_balance");
  if (subject.zerone !== null) reasons.add("has_zerone_auth_record");
  if (subject.didRegistered.length > 0) reasons.add("did_registered");
  if (kind === "admit") {
    if (subject.auth !== null) reasons.add("has_account");
    if (subject.potLower !== null || subject.potUpper !== null) reasons.add("has_pot");
    if (subject.grantsToAddress.length > 0) reasons.add("has_grant");
  } else {
    if (subject.potUpper !== null) reasons.add("has_pot");
    if (subject.grantsToAddress.some((g) => g.granter !== registrar)) reasons.add("has_grant");
  }

  // R signs one transaction at a time: while an earlier one is unconfirmed,
  // its sequence on chain is stale and a second signature would collide.
  if (ledger.entries.some((e) => e.status === "pending_inclusion")) reasons.add("door_busy");

  // ── capacity: caps, unclaimed seeds and what R can still pay ──
  const today = utcDay(input.now);
  const countedEntries = counted(ledger);
  if (countedEntries.filter((e) => e.utc_day === today).length >= DOOR_DAILY_CAP) reasons.add("door_daily_cap");
  if (input.membership.reasons.length === 0
    && countedEntries.filter((e) => e.utc_day === today && e.channel_key === input.membership.channelKey).length
      >= input.membership.channelCap) reasons.add("channel_daily_cap");
  const window = currentWindow(global.status.height);
  if (kind === "admit") {
    if (countedEntries.filter((e) => e.kind === "admit" && e.chain_window === window).length >= DOOR_WINDOW_CAP) {
      reasons.add("door_window_cap");
    }
    const used = global.window.storedWindow === window ? global.window.storedCount : 0;
    if (used + 1 > global.params.windowCap) reasons.add("chain_window_full");
  }
  if (seedCounts(global, registrar).unclaimedLive + 1 > UNCLAIMED_LIVE_SEED_CAP) reasons.add("unclaimed_seed_cap");
  const liability = registrarLiability(global);
  if (liability === null) {
    reasons.add("registrar_has_unbounded_grant");
  } else {
    const ceiling = BigInt(kind === "admit" ? ADMISSION_CEILING_UZRN : REGRANT_CEILING_UZRN);
    if (global.registrarSpendable < liability + ceiling + BigInt(REGISTRAR_RESERVE_UZRN)) reasons.add("registrar_balance_low");
  }

  const list = [...reasons];
  return { kind: list.length === 0 ? kind : "refuse", reasons: list, revokeFirst };
}
