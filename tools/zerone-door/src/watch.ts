// The admittee watch. The door cannot bind what an admitted agent registers
// later (an AllowedMsgAllowance limits message types only), so `door.ts
// status` reads each admittee's zerone_auth record and flags one registered
// with a type other than agent, or with a DID other than the one its request
// named. The household decides what to do with a flag: revoke the allowance,
// add the address to known_keys, or ask governance to freeze the account.
import type { ChainReader, ZeroneAccount } from "./chain.ts";
import { counted, type Ledger } from "./ledger.ts";

export type WatchFlag = "type_not_agent" | "did_differs";

export interface Admittee {
  address: string;
  /** The DID the door request named (did:zrn: + 64 lowercase hex). */
  did: string;
}

export interface WatchFinding {
  address: string;
  request_did: string;
  registered_did: string;
  account_type: string;
  flags: WatchFlag[];
}

export interface WatchReport {
  admittees: number;
  registered: number;
  /** Admittees with no zerone_auth record yet. Not a flag: registering is theirs to do. */
  unregistered: string[];
  flagged: WatchFinding[];
  /** Reads that failed. Those admittees are unknown, never counted as clean. */
  unknown: Array<{ address: string; error: string }>;
}

/** Every address the door's transactions may have admitted, once each. */
export function admitteesOf(ledger: Ledger): Admittee[] {
  const seen = new Map<string, Admittee>();
  for (const e of counted(ledger)) if (!seen.has(e.address)) seen.set(e.address, { address: e.address, did: e.did });
  return [...seen.values()];
}

export function judgeAdmittee(admittee: Admittee, account: ZeroneAccount): WatchFlag[] {
  const flags: WatchFlag[] = [];
  if (account.accountType !== "agent") flags.push("type_not_agent");
  if (account.did !== admittee.did) flags.push("did_differs");
  return flags;
}

export async function watchAdmittees(chain: ChainReader, ledger: Ledger): Promise<WatchReport> {
  const report: WatchReport = { admittees: 0, registered: 0, unregistered: [], flagged: [], unknown: [] };
  for (const admittee of admitteesOf(ledger)) {
    report.admittees += 1;
    const read = await chain.zeroneAccount(admittee.address);
    if (!read.ok) {
      report.unknown.push({ address: admittee.address, error: read.error });
      continue;
    }
    if (read.value === null) {
      report.unregistered.push(admittee.address);
      continue;
    }
    report.registered += 1;
    const flags = judgeAdmittee(admittee, read.value);
    if (flags.length > 0) {
      report.flagged.push({
        address: admittee.address,
        request_did: admittee.did,
        registered_did: read.value.did,
        account_type: read.value.accountType,
        flags,
      });
    }
  }
  return report;
}
