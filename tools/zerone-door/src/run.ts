// One pass of the door, as launchd runs it every two minutes: settle what a
// previous live run left pending, take Studio-local and lobby requests, decide
// each, answer, and remember where the lobby read stopped.
import { processItem, readLedger, reconcilePending, replyText, writeLedger, type DoorContext } from "./door.ts";
import { readLobby, replyInLobby, takeSpool, type IntakeItem, type LobbyRunner } from "./intake.ts";
import type { LobbyCursor } from "./ledger.ts";
import { withLock } from "./ledger.ts";

export interface RunDeps {
  /** Runs the agenttool-lobby CLI; null when the config has no lobby. */
  lobbyRun: LobbyRunner | null;
}

/** Refusals the door does not answer in the room: they come from senders it
 * cannot hold to a request (malformed, stale or unverified). */
const QUIET_LOBBY_REASONS = new Set(["request_malformed", "request_stale", "request_from_future", "lobby_sender_not_verified"]);

export async function runOnce(ctx: DoorContext, deps: RunDeps): Promise<void> {
  await withLock(ctx.paths.lock, async () => {
    await reconcilePending(ctx);
    const items: IntakeItem[] = [];
    try {
      items.push(...takeSpool(ctx.paths.inbox, ctx.paths.processed, ctx.now()));
    } catch (error) {
      ctx.log({ event: "spool_unavailable", detail: error instanceof Error ? error.message : String(error) });
    }
    let lobbyCursor: LobbyCursor | null = null;
    const lobby = ctx.config.lobby;
    if (lobby && deps.lobbyRun) {
      try {
        const read = readLobby(deps.lobbyRun, lobby, readLedger(ctx).lobby_cursor);
        items.push(...read.items);
        lobbyCursor = read.cursor;
        if (read.seasonChanged) {
          ctx.log({ event: "lobby_season_changed", from: read.seasonChanged.from, to: read.seasonChanged.to, stale_cursor: read.seasonChanged.stale_cursor });
        }
        if (read.unverified > 0) ctx.log({ event: "lobby_unverified_skipped", count: read.unverified });
      } catch (error) {
        ctx.log({ event: "lobby_unavailable", detail: error instanceof Error ? error.message : String(error) });
      }
    }
    for (const item of items) {
      const outcome = await processItem(ctx, item);
      if (item.channel.kind !== "lobby" || !outcome.answer || !outcome.reply || !lobby || !deps.lobbyRun || !lobby.reply_lease) continue;
      // A dry run signs nothing, so it says nothing in the room: an agent
      // reading "dry_run_ok" there could take it for an admission.
      if (!ctx.live) continue;
      if (outcome.reply.reasons.some((r) => QUIET_LOBBY_REASONS.has(r))) continue;
      const sent = replyInLobby(deps.lobbyRun, lobby, item.channel.event_id, outcome.reply.request_id, replyText(outcome.reply));
      ctx.log({ event: "lobby_reply", request_id: outcome.reply.request_id, ok: sent.ok, detail: sent.detail });
    }
    if (lobbyCursor !== null) {
      const ledger = readLedger(ctx);
      ledger.lobby_cursor = lobbyCursor;
      writeLedger(ctx, ledger);
    }
  });
}
