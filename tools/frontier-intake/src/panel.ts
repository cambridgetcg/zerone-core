import { createHash, randomBytes } from "node:crypto";

// The proven ceremony recipe (scripts/first-truth-ceremony.sh:175):
//   sha256("ZRN.commit.v1:<round>:<vote>:<confidence>:<salt-hex>")
// The hospitable CLI computes confidence 0; the reveal message carries only
// (vote, salt), so 0 is the only reveal-compatible confidence here.
export function commitHash(roundId: string, vote: string, saltHex: string): string {
  return createHash("sha256")
    .update(`ZRN.commit.v1:${roundId}:${vote}:0:${saltHex}`, "utf8")
    .digest("hex");
}

export function newSalt(): string {
  return randomBytes(16).toString("hex");
}

export interface PhaseParams {
  commit_blocks: number;
  reveal_blocks: number;
  aggregation_blocks: number;
}

export interface VerifierProgress {
  salt: string;
  commit_tx?: string;
  reveal_tx?: string;
}

export interface RoundState {
  entry_id: string;
  claim_id: string;
  round_id: string;
  submit_height: number;
  submit_tx: string;
  vote: "accept" | "reject";
  verifiers: Record<string, VerifierProgress>;
  stage: "submitted" | "pending_inclusion" | "resolved_accepted" | "resolved_other" | "missed_window" | "abandoned" | "error";
  fact_status?: string;
  fact_id?: string;
  confidence?: string;
  query_failures?: number;
  notes: string[];
}

// x/knowledge ClaimStatus enum (proto/zerone/knowledge/v1/types.proto), as the
// live CLI emits it numerically. 1-5 are in-flight; 6 is the goal.
export function classifyClaimStatus(raw: string): "accepted" | "terminal_other" | "in_flight" | "unknown" {
  const value = raw.trim();
  if (/^6$/.test(value) || /ACCEPT/i.test(value)) return "accepted";
  if (/^(7|9|10|11|12)$/.test(value) || /REJECT|EXPIRE|INSUFFICIENT|CONTEST|MALFORM/i.test(value)) return "terminal_other";
  if (/^[1-5]$/.test(value) || /PENDING|COMMIT|REVEAL|EVALUAT|VERIF/i.test(value)) return "in_flight";
  return "unknown";
}

export type PanelAction =
  | { kind: "commit"; verifier: string; hash: string }
  | { kind: "reveal"; verifier: string; salt: string }
  | { kind: "check_resolution" }
  | { kind: "missed_window"; detail: string }
  | { kind: "wait"; until_height: number; why: string };

/**
 * Pure phase logic: given a round's state, the current height, and the phase
 * params, decide what to do next. The chain flips phases in BeginBlocker at
 * height >= deadline, so the real wall is deadline-1; the margin scales with
 * how many serial txs remain (each waits ~2 blocks for inclusion).
 */
export function decideActions(
  round: RoundState,
  height: number,
  params: PhaseParams,
  blocksPerTx = 2,
): PanelAction[] {
  if (round.stage !== "submitted") return [];
  const commitEnd = round.submit_height + params.commit_blocks;
  const revealEnd = commitEnd + params.reveal_blocks;
  const aggregationEnd = revealEnd + params.aggregation_blocks;
  const actions: PanelAction[] = [];
  const names = Object.keys(round.verifiers);
  const uncommitted = names.filter(name => !round.verifiers[name]?.commit_tx);
  const unrevealed = names.filter(name => round.verifiers[name]?.commit_tx && !round.verifiers[name]?.reveal_tx);
  const margin = 2 + blocksPerTx * Math.max(uncommitted.length, unrevealed.length);

  if (uncommitted.length > 0) {
    if (height < commitEnd - margin) {
      for (const verifier of uncommitted) {
        const progress = round.verifiers[verifier] as VerifierProgress;
        actions.push({ kind: "commit", verifier, hash: commitHash(round.round_id, round.vote, progress.salt) });
      }
      return actions;
    }
    return [{ kind: "missed_window", detail: `commit window closed at ${commitEnd} with ${uncommitted.length} uncommitted` }];
  }
  if (unrevealed.length > 0) {
    if (height <= commitEnd) {
      return [{ kind: "wait", until_height: commitEnd + 1, why: "reveal phase not yet open" }];
    }
    if (height < revealEnd - margin) {
      for (const verifier of unrevealed) {
        const progress = round.verifiers[verifier] as VerifierProgress;
        actions.push({ kind: "reveal", verifier, salt: progress.salt });
      }
      return actions;
    }
    return [{ kind: "missed_window", detail: `reveal window closed at ${revealEnd} with ${unrevealed.length} unrevealed` }];
  }
  if (height > aggregationEnd + margin) {
    return [{ kind: "check_resolution" }];
  }
  return [{ kind: "wait", until_height: aggregationEnd + margin + 1, why: "aggregation pending" }];
}
