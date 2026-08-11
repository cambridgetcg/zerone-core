import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  BRANCH_FLOW_ENDPOINT,
  BRANCH_FLOW_MAX_BYTES,
  BRANCH_FLOW_REFERENCE_POLICY_DIGEST,
  BRANCH_FLOW_SHA256,
  fetchBranchFlow,
  parseBranchFlow,
  parseBranchFlowJson,
} from "../src/branch-flow";

type MutableBranchFlow = Record<string, any> & {
  reference_policy: Record<string, any> & {
    decay: Array<Record<string, any>>;
  };
  milestone_boundary: Record<string, any> & {
    milestones: Array<Record<string, any>>;
  };
  allocation_invariants: Record<string, any>;
  authority_boundary: Record<string, any>;
  release_boundary: Record<string, any>;
  release_gates: Array<Record<string, any>>;
};

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-branch-flow.v1.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as MutableBranchFlow;
const runtimeSource = readFileSync(
  new URL("../src/branch-flow.ts", import.meta.url),
  "utf8",
);
const mainSource = readFileSync(
  new URL("../src/main.ts", import.meta.url),
  "utf8",
);
const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");

function copyFlow(): MutableBranchFlow {
  return structuredClone(canonical) as MutableBranchFlow;
}

function jsonResponse(
  body = canonicalRaw,
  init: ResponseInit = {},
): Response {
  return new Response(body, {
    ...init,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...init.headers,
    },
  });
}

describe("Branch Flow v1 runtime boundary", () => {
  it("parses the reviewed funded-cluster shadow profile", () => {
    const flow = parseBranchFlowJson(canonicalRaw);
    assert.equal(
      flow.schema,
      "zerone.constructive-intelligence-branch-flow/v1",
    );
    assert.equal(flow.assurance, "SHADOW_ONLY");
    assert.equal(flow.economicEffect, "NONE");
    assert.equal(flow.movesFunds, false);
    assert.equal(flow.integrationReady, false);
    assert.equal(flow.releaseAmountUzrn, "0");
    assert.equal(flow.referencePolicy.directPpm, 600_000);
    assert.equal(flow.referencePolicy.upstreamPpm, 100_000);
    assert.equal(flow.referencePolicy.downstreamPpm, 300_000);
    assert.equal(flow.referencePolicy.baseCommonsPpm, 0);
    assert.equal(
      flow.referencePolicy.policyDigest,
      BRANCH_FLOW_REFERENCE_POLICY_DIGEST,
    );
    assert.equal(flow.releaseGates.length, 17);
    assert.ok(flow.releaseGates.every(({ passed }) => !passed));
  });

  it("pins absolute decay, the E5 fixture, receipt use, and no-TC6 boundary", () => {
    const flow = parseBranchFlowJson(canonicalRaw);
    for (const decay of flow.referencePolicy.decay) {
      assert.equal(decay.continuationPpm, 500_000);
      assert.equal(decay.maxDepth, 5);
      assert.deepEqual(decay.depthWeightsPpm, [
        500_000,
        250_000,
        125_000,
        62_500,
        31_250,
      ]);
      assert.equal(decay.tailPpm, 31_250);
    }
    const invariants = flow.allocationInvariants;
    assert.equal(invariants.fundedClusterIsEconomicSubject, true);
    assert.equal(invariants.breakthroughIsAllocationInput, false);
    assert.equal(invariants.breakthroughCreatesPrize, false);
    assert.equal(invariants.referenceMilestone, "E5");
    assert.equal(invariants.preE5DownstreamPpm, 0);
    assert.equal(invariants.preE5DescendantImpactsAllowed, false);
    assert.equal(
      invariants.descendantImpactShareFormula,
      "floor(S*m/max(S,sum_m_per_semantic_descendant))",
    );
    assert.equal(
      invariants.acceptedReceiptUse,
      "CONSUME_ON_SUCCESSFUL_EVALUATION",
    );
    assert.equal(invariants.zeroProjectionConsumesAcceptedReceipt, true);
    assert.equal(invariants.invalidRequestConsumesReceipt, false);
    assert.equal(invariants.fundedControllerDescendantCreditEligible, false);
    assert.equal(
      invariants.mixedControlDescendantIndependentCreditsRemainEvaluable,
      true,
    );
    assert.equal(
      invariants.hiddenOrCorrelatedControlRequiresExternalAdjudication,
      true,
    );
    assert.equal(invariants.tc6TrainingRevenue.implemented, false);
  });

  it("fails closed on economic, authority, policy, milestone, and unknown-field drift", () => {
    for (const key of [
      "moves_funds",
      "integration_ready",
      "authoritative",
      "network_observed",
      "reward_bearing",
    ]) {
      const changed = copyFlow();
      changed[key] = true;
      assert.throws(() => parseBranchFlow(changed), new RegExp(key));
    }
    const split = copyFlow();
    split.reference_policy.direct_ppm += 1;
    assert.throws(() => parseBranchFlow(split), /reference_policy\.direct_ppm/);

    const relativeDepth = copyFlow();
    relativeDepth.reference_policy.absolute_depth_buckets = false;
    assert.throws(
      () => parseBranchFlow(relativeDepth),
      /absolute_depth_buckets/,
    );

    const preE5Impact = copyFlow();
    preE5Impact.allocation_invariants.pre_e5_descendant_impacts_allowed = true;
    assert.throws(
      () => parseBranchFlow(preE5Impact),
      /pre_e5_descendant_impacts_allowed/,
    );

    const fundedControllerSelfCredit = copyFlow();
    fundedControllerSelfCredit.allocation_invariants.funded_controller_descendant_credit_eligible =
      true;
    assert.throws(
      () => parseBranchFlow(fundedControllerSelfCredit),
      /funded_controller_descendant_credit_eligible/,
    );

    const eraseIndependentCredit = copyFlow();
    eraseIndependentCredit.allocation_invariants.mixed_control_descendant_independent_credits_remain_evaluable =
      false;
    assert.throws(
      () => parseBranchFlow(eraseIndependentCredit),
      /mixed_control_descendant_independent_credits_remain_evaluable/,
    );

    const inferHiddenControl = copyFlow();
    inferHiddenControl.allocation_invariants.hidden_or_correlated_control_requires_external_adjudication =
      false;
    assert.throws(
      () => parseBranchFlow(inferHiddenControl),
      /hidden_or_correlated_control_requires_external_adjudication/,
    );

    const release = copyFlow();
    release.release_boundary.moves_funds = true;
    assert.throws(() => parseBranchFlow(release), /release_boundary\.moves_funds/);

    const authority = copyFlow();
    authority.authority_boundary.selects_winners = true;
    assert.throws(
      () => parseBranchFlow(authority),
      /authority_boundary\.selects_winners/,
    );

    const milestone = copyFlow();
    milestone.milestone_boundary.milestones[3]!.outcome_pool_bps += 1;
    assert.throws(
      () => parseBranchFlow(milestone),
      /milestones\[3\]\.outcome_pool_bps/,
    );

    const unknown = copyFlow();
    unknown.royalty = true;
    assert.throws(() => parseBranchFlow(unknown), /royalty: unknown field/);
  });

  it("rejects malformed and oversized JSON before presentation", () => {
    assert.throws(() => parseBranchFlowJson("{"), /malformed JSON/);
    assert.throws(
      () => parseBranchFlowJson(" ".repeat(BRANCH_FLOW_MAX_BYTES + 1)),
      /exceeds/,
    );
  });
});

describe("Branch Flow v1 bounded static fetch", () => {
  it("pins the reviewed document bytes", () => {
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      BRANCH_FLOW_SHA256,
    );
  });

  it("uses the versioned endpoint with no-store, same-origin credentials, redirect refusal, and a deadline", async () => {
    let inputSeen: RequestInfo | URL | undefined;
    let initSeen: RequestInit | undefined;
    const flow = await fetchBranchFlow({
      fetcher: async (input, init) => {
        inputSeen = input;
        initSeen = init;
        return jsonResponse();
      },
    });
    assert.equal(inputSeen, BRANCH_FLOW_ENDPOINT);
    assert.equal(initSeen?.cache, "no-store");
    assert.equal(initSeen?.credentials, "same-origin");
    assert.equal(initSeen?.redirect, "error");
    assert.equal(
      new Headers(initSeen?.headers).get("accept"),
      "application/json",
    );
    assert.ok(initSeen?.signal instanceof AbortSignal);
    assert.equal(flow.status, "SHADOW_ONLY");
  });

  it("rejects content drift and any final URL outside the exact same-origin path", async () => {
    await assert.rejects(
      fetchBranchFlow({
        fetcher: async () =>
          jsonResponse(canonicalRaw.replace("600000", "600001")),
      }),
      /reviewed canonical digest/,
    );

    const redirected = jsonResponse();
    Object.defineProperty(redirected, "url", {
      value:
        "https://attacker.example/standards/constructive-intelligence-branch-flow.v1.json",
    });
    await assert.rejects(
      fetchBranchFlow({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => redirected,
      }),
      /canonical same-origin path/,
    );

    const queryDrift = jsonResponse();
    Object.defineProperty(queryDrift, "url", {
      value:
        "https://zerone.ai/standards/constructive-intelligence-branch-flow.v1.json?unreviewed=1",
    });
    await assert.rejects(
      fetchBranchFlow({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => queryDrift,
      }),
      /canonical same-origin path/,
    );
  });

  it("rejects HTTP, media-type, declared-size, and streamed-size failures", async () => {
    await assert.rejects(
      fetchBranchFlow({
        fetcher: async () => jsonResponse("{}", { status: 503 }),
      }),
      /HTTP 503/,
    );
    await assert.rejects(
      fetchBranchFlow({
        fetcher: async () =>
          new Response(canonicalRaw, {
            headers: { "content-type": "text/html" },
          }),
      }),
      /explicit JSON/,
    );
    await assert.rejects(
      fetchBranchFlow({
        fetcher: async () => new Response(canonicalRaw),
      }),
      /explicit JSON/,
    );
    await assert.rejects(
      fetchBranchFlow({
        fetcher: async () =>
          jsonResponse("{}", {
            headers: {
              "content-length": `${BRANCH_FLOW_MAX_BYTES + 1}`,
            },
          }),
      }),
      /size limit/,
    );
    await assert.rejects(
      fetchBranchFlow({
        fetcher: async () =>
          jsonResponse(" ".repeat(BRANCH_FLOW_MAX_BYTES + 1)),
      }),
      /exceeds/,
    );
  });

  it("times out a stalled request and can recover on a later call", async () => {
    await assert.rejects(
      fetchBranchFlow({
        timeoutMs: 5,
        fetcher: async (_input, init) =>
          await new Promise<Response>((_resolve, reject) => {
            const signal = init?.signal;
            assert.ok(signal);
            signal.addEventListener(
              "abort",
              () => reject(signal.reason),
              { once: true },
            );
          }),
      }),
      /timed out/,
    );

    let attempt = 0;
    const fetcher = async (): Promise<Response> => {
      attempt += 1;
      return attempt === 1
        ? jsonResponse("{}", { status: 502 })
        : jsonResponse();
    };
    await assert.rejects(fetchBranchFlow({ fetcher }), /HTTP 502/);
    const recovered = await fetchBranchFlow({ fetcher });
    assert.equal(recovered.assurance, "SHADOW_ONLY");
    assert.equal(attempt, 2);
  });

  it("keeps the deadline when an injected fetcher ignores AbortSignal", async () => {
    const started = Date.now();
    await assert.rejects(
      fetchBranchFlow({
        timeoutMs: 5,
        fetcher: async () => await new Promise<Response>(() => undefined),
      }),
      /timed out/,
    );
    assert.ok(Date.now() - started < 1_000);
  });
});

describe("Branch Flow v1 read-only page wiring", () => {
  it("mounts one small Skills-section surface with direct raw access", () => {
    assert.equal(html.match(/id="branch-flow"/g)?.length, 1);
    assert.equal(html.match(/id="branch-flow-root"/g)?.length, 1);
    assert.match(html, /One envelope\. Three directions\. No royalties\./);
    assert.match(
      html,
      /href="\/standards\/constructive-intelligence-branch-flow\.v1\.json"/,
    );
    assert.ok(
      html.indexOf('id="constructive-tree-root"') <
        html.indexOf('id="branch-flow"'),
    );
    assert.ok(
      html.indexOf('id="branch-flow"') <
        html.indexOf('id="life-sciences-tree-root"'),
    );
  });

  it("uses text-node rendering and no chain, wallet, form, or HTML injection path", () => {
    assert.doesNotMatch(runtimeSource, /innerHTML|insertAdjacentHTML/);
    assert.doesNotMatch(runtimeSource, /\/api\/|wallet|keplr|<form/i);
    assert.match(runtimeSource, /Activated amount \(static\)/);
    assert.match(runtimeSource, /Breakthrough is retrospective only/);
    assert.match(runtimeSource, /releaseGates\.filter/);
    assert.match(runtimeSource, /flow\.releaseGates\.length/);
  });

  it("is explicitly initialized by the dashboard entrypoint", () => {
    assert.match(
      mainSource,
      /import \{ initialiseBranchFlow \} from "\.\/branch-flow"/,
    );
    assert.match(
      mainSource,
      /initialiseBranchFlow\(branchFlowRoot\)/,
    );
    assert.match(mainSource, /byId<HTMLElement>\("branch-flow-root"\)/);
    assert.match(mainSource, /window\.location\.hash !== "#branch-flow"/);
    assert.match(
      mainSource,
      /branchFlowRoot\.closest<HTMLElement>\("#branch-flow"\)/,
    );
  });
});
