// The door's fixed policy. Every number here is a reviewed constant, changed
// only by a pull request, with the measurement or reason beside it. Gas was
// read on zerone-1 (2e37c4c lineage, SDK v0.50.15) on 2026-10-28; the fee floor
// there is fee >= gas_limit x 1 uzrn, and a fee is charged on the gas LIMIT.

export const CHAIN_ID = "zerone-1" as const;
export const DENOM = "uzrn" as const;

export const MSG_POT_CLAIM = "/zerone.claiming_pot.v1.MsgClaim";
export const MSG_REGISTER = "/zerone.auth.v1.MsgRegisterAccount";
export const MSG_SUBMIT_CLAIM = "/zerone.knowledge.v1.MsgSubmitClaim";
export const MSG_ADD_BOOTSTRAP_ENTRY = "/zerone.claiming_pot.v1.MsgAddBootstrapEntry";
export const MSG_GRANT_ALLOWANCE = "/cosmos.feegrant.v1beta1.MsgGrantAllowance";
export const MSG_REVOKE_ALLOWANCE = "/cosmos.feegrant.v1beta1.MsgRevokeAllowance";

/** The only messages a door allowance pays for, in this order. */
export const ALLOWED_MESSAGES = [MSG_POT_CLAIM, MSG_REGISTER, MSG_SUBMIT_CLAIM] as const;

// ── what the agent's own transactions cost (the guide sets these limits) ──
/** Pot MsgClaim. Fee-granted seed claims used 128,378 and 142,289 gas (July). */
export const AGENT_POT_CLAIM_GAS = 170_000;
/** MsgRegisterAccount. Simulated 86,946 gas; the fee-grant read and update and
 * the decorators Simulate skips add about 30,000. */
export const AGENT_REGISTER_GAS = 130_000;
/** MsgSubmitClaim. An SDK-built claim simulated 244,357 gas; the household's
 * delivered claims used 279,929-351,830. A claim that needs more than this is
 * "heavy" and is never paid by a door allowance. */
export const AGENT_SUBMIT_CLAIM_GAS = 400_000;

// ── the allowance: AllowedMsgAllowance over a PeriodicAllowance ──
/** Period spend cap. It is also the largest fee any single transaction can
 * draw, so no sponsored transaction can pay for a claim heavier than
 * AGENT_SUBMIT_CLAIM_GAS. Seed claim + registration (300,000) fit in one period. */
export const ALLOWANCE_PERIOD_LIMIT_UZRN = 400_000;
/** 15 minutes: the first claim waits at most one period after registering. */
export const ALLOWANCE_PERIOD_SECONDS = 900;
/** Total: seed claim + registration + one claim + one retry of that claim
 * (170,000 + 130,000 + 400,000 + 400,000). The 0.222 ZRN seed covers exactly
 * one 0.2 ZRN review fee, which a fee allowance can never pay. */
export const ALLOWANCE_SPEND_LIMIT_UZRN =
  AGENT_POT_CLAIM_GAS + AGENT_REGISTER_GAS + 2 * AGENT_SUBMIT_CLAIM_GAS;
export const ALLOWANCE_DAYS = 14;

// ── what R's own transactions cost ──
/** AddBootstrapEntry + GrantAllowance (this allowance shape and memo)
 * simulated 110,906 gas at h1402838 with the live registrar as a
 * public-key-only stand-in (tests/live-e2e.test.ts); July's delivered
 * admissions ran about 15,000 above their Simulate. */
export const ADMIT_GAS = 150_000;
/** Re-grant: GrantAllowance alone simulated 88,429 at h1402839. An expired
 * grant the feegrant EndBlocker has not pruned yet is revoked first in the
 * same transaction; that variant cannot be simulated until such a grant
 * exists, so it gets more room. */
export const REGRANT_GAS = 140_000;
/** Simulate skips the ante decorators; refuse when sim + margin > limit. */
export const SIMULATE_MARGIN_GAS = 20_000;

// ── what R can lose, and the caps that bound it ──
/** R must keep this much beyond every live allowance it has issued. */
export const REGISTRAR_RESERVE_UZRN = 2_000_000;
/** Worst case per admission: R's fee plus a fully spent allowance (1.25 ZRN). */
export const ADMISSION_CEILING_UZRN = ADMIT_GAS + ALLOWANCE_SPEND_LIMIT_UZRN;
export const REGRANT_CEILING_UZRN = REGRANT_GAS + ALLOWANCE_SPEND_LIMIT_UZRN;
/** Door admissions per chain admission window (34,272 blocks, about 11.9 days).
 * 20 x 1.25 ZRN + 2 ZRN reserve = 27 ZRN, inside R's ~30 ZRN float. */
export const DOOR_WINDOW_CAP = 20;
/** Door admissions and re-grants per UTC day, across every channel. */
export const DOOR_DAILY_CAP = 10;
/** Per allowlisted lobby desk or the Studio channel, per UTC day, unless the
 * allowlist names a smaller number. */
export const CHANNEL_DAILY_CAP_DEFAULT = 3;
/** Unclaimed seeds whose allowance from R is still live. Bootstrap pots never
 * expire and are walked every BeginBlock, so this stays small. Seeds whose
 * allowance lapsed are not counted (they cost R nothing); status lists them. */
export const UNCLAIMED_LIVE_SEED_CAP = 10;
export const SEED_UZRN = 222_000;
export const BOOTSTRAP_WINDOW_BLOCKS = 34_272;

// ── requests ──
export const POW_BITS = 20;
export const REQUEST_MAX_AGE_SECONDS = 15 * 60;
export const REQUEST_MAX_FUTURE_SECONDS = 2 * 60;
export const REQUEST_MAX_BYTES = 2_048;

// ── liveness ──
/** The chain is treated as halted when its newest block is older than this. */
export const CHAIN_STALE_SECONDS = 10 * 60;
/** At 30-second blocks, ten blocks. CosmJS's 60 s default is too short. */
export const INCLUSION_TIMEOUT_MS = 300_000;
export const READ_TIMEOUT_MS = 30_000;

export function allowanceInvariantsHold(): boolean {
  return AGENT_POT_CLAIM_GAS + AGENT_REGISTER_GAS <= ALLOWANCE_PERIOD_LIMIT_UZRN
    && AGENT_SUBMIT_CLAIM_GAS <= ALLOWANCE_PERIOD_LIMIT_UZRN
    && ALLOWANCE_PERIOD_LIMIT_UZRN <= ALLOWANCE_SPEND_LIMIT_UZRN
    && ALLOWANCE_PERIOD_SECONDS < ALLOWANCE_DAYS * 86_400
    && DOOR_WINDOW_CAP * ADMISSION_CEILING_UZRN + REGISTRAR_RESERVE_UZRN <= 30_000_000;
}
