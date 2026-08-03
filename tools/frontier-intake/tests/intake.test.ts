import { describe, expect, test } from "bun:test";
import { defaultRegisterPath } from "../src/config.ts";
import { commitHash, decideActions, newSalt, type RoundState } from "../src/panel.ts";
import { admissibleEntries, firstBatch, loadRegister, validateRegister } from "../src/register.ts";

describe("register", () => {
  const register = loadRegister(defaultRegisterPath());

  test("the checked-in register validates cleanly", () => {
    expect(validateRegister(register)).toEqual([]);
  });

  test("admission partitions are sane", () => {
    const asserts = admissibleEntries(register);
    expect(asserts.length).toBeGreaterThanOrEqual(20);
    const batch = firstBatch(register);
    expect(batch.length).toBe(9);
    expect(batch[0]?.id).toBe("pfr-marton-2023");
    for (const entry of register.entries) {
      expect(["assert", "conjecture", "withhold"]).toContain(entry.chain.intent);
      if (entry.chain.intent === "conjecture") expect(entry.status).toBe("open_conjecture");
      if (entry.status === "preprint_under_review" || entry.status === "announced_unverified") {
        expect(entry.chain.intent).toBe("withhold");
      }
    }
  });

  test("every statement fits the live chain limits", () => {
    for (const entry of register.entries) {
      expect(entry.statement.length).toBeGreaterThanOrEqual(20);
      expect(entry.statement.length).toBeLessThanOrEqual(1000);
    }
  });

  test("refuted entries assert the refutation, never the refuted statement", () => {
    for (const entry of register.entries) {
      if (entry.status === "refuted" && entry.chain.intent === "assert") {
        expect(entry.chain.category).toBe("refutation");
        expect(/false|refut|disproof|disprove|counterexample/i.test(entry.statement)).toBe(true);
      }
    }
  });

  test("validator catches broken entries", () => {
    const broken = structuredClone(register);
    const entry = broken.entries[0];
    if (!entry) throw new Error("register empty");
    entry.statement = "too short";
    entry.chain.domain = "physics";
    const issues = validateRegister(broken);
    expect(issues.some(issue => issue.problem.includes("statement length"))).toBe(true);
    expect(issues.some(issue => issue.problem.includes("not mathematics"))).toBe(true);
  });
});

describe("panel", () => {
  test("commit hash matches the ceremony recipe", () => {
    // printf 'ZRN.commit.v1:round-1:accept:0:aabb' | shasum -a 256
    const expected = Bun.spawnSync({
      cmd: ["sh", "-c", "printf 'ZRN.commit.v1:round-1:accept:0:aabb' | shasum -a 256 | cut -d' ' -f1"],
      stdout: "pipe",
    }).stdout.toString().trim();
    expect(commitHash("round-1", "accept", "aabb")).toBe(expected);
  });

  const params = { commit_blocks: 200, reveal_blocks: 200, aggregation_blocks: 50 };
  function round(overrides: Partial<RoundState>): RoundState {
    return {
      entry_id: "e",
      claim_id: "c",
      round_id: "r",
      submit_height: 1000,
      submit_tx: "T",
      vote: "accept",
      verifiers: { v1: { salt: newSalt() }, v2: { salt: newSalt() } },
      stage: "submitted",
      notes: [],
      ...overrides,
    };
  }

  test("commits inside the commit window", () => {
    const actions = decideActions(round({}), 1010, params);
    expect(actions.every(action => action.kind === "commit")).toBe(true);
    expect(actions).toHaveLength(2);
  });

  test("waits for reveal phase after all commits", () => {
    const committed = round({
      verifiers: { v1: { salt: "a", commit_tx: "x" }, v2: { salt: "b", commit_tx: "y" } },
    });
    const during = decideActions(committed, 1100, params);
    expect(during[0]?.kind).toBe("wait");
    const revealPhase = decideActions(committed, 1250, params);
    expect(revealPhase.every(action => action.kind === "reveal")).toBe(true);
  });

  test("reports a missed commit window instead of pretending", () => {
    const actions = decideActions(round({}), 1199, params);
    expect(actions[0]?.kind).toBe("missed_window");
  });

  test("checks resolution only after aggregation margin", () => {
    const done = round({
      verifiers: { v1: { salt: "a", commit_tx: "x", reveal_tx: "p" }, v2: { salt: "b", commit_tx: "y", reveal_tx: "q" } },
    });
    expect(decideActions(done, 1440, params)[0]?.kind).toBe("wait");
    expect(decideActions(done, 1460, params)[0]?.kind).toBe("check_resolution");
  });

  test("resolved rounds require no actions", () => {
    expect(decideActions(round({ stage: "resolved_accepted" }), 2000, params)).toEqual([]);
  });
});
