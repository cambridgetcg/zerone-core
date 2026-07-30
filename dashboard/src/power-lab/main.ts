import "./styles.css";
import rawFixture from "./shadow.generated.json";
import {
  LEDGER_LANE_IDS,
  POWER_SURFACE_LABELS,
  formatModelUnits,
  formatPercent,
  mathNodeLevels,
  parsePowerLabFixture,
  type CapacityShadowStep,
  type Gate,
  type PowerLabFixture,
  type PowerSurface,
  type ShadowCluster,
} from "./model";

const LOCAL_HOSTS = new Set(["127.0.0.1", "localhost", "::1", "[::1]"]);
const LAB_BOUNDARY =
  "LOCAL FIXTURE · NON-AUTHORITATIVE · NOT NETWORK-OBSERVED · 0 ZRN · RELEASE CLOSED";

function labRoot(): HTMLElement {
  const found = document.getElementById("lab-root");
  if (!found) throw new Error("Missing #lab-root");
  return found;
}

const root = labRoot();

function element<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  text?: string,
): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function sectionHeading(
  number: string,
  kicker: string,
  title: string,
  description: string,
): HTMLDivElement {
  const heading = element("div", "section-heading");
  const eyebrow = element("p", "eyebrow");
  eyebrow.append(element("span", "", number), document.createTextNode(kicker));
  heading.append(
    eyebrow,
    element("h2", "", title),
    element("p", "section-copy", description),
  );
  return heading;
}

function metric(label: string, value: string): HTMLDivElement {
  const row = element("div");
  row.append(element("dt", "", label), element("dd", "", value));
  return row;
}

function renderFailure(message: string): void {
  root.replaceChildren();
  const failure = element("section", "lab-failure");
  failure.append(
    element("p", "eyebrow", "FAIL-CLOSED FIXTURE"),
    element("h1", "", "The local lab did not open."),
    element("p", "", message),
    element(
      "p",
      "zero-value",
      "No chain state was read. No reward exists. Settlement remains 0 ZRN.",
    ),
  );
  root.append(failure);
}

function renderHero(fixture: PowerLabFixture): HTMLElement {
  const hero = element("header", "lab-hero");
  const copy = element("div", "hero-copy");
  const eyebrow = element("p", "eyebrow");
  eyebrow.append(
    element("span", "", "技能樹 / 00"),
    document.createTextNode("Power and reward research"),
  );
  copy.append(
    eyebrow,
    element("h1", "", "Power garden."),
    element(
      "p",
      "hero-lede",
      "A synthetic stress bench for seeing where authority can accumulate—and where a counterfactual reward ledger still has no right to become money.",
    ),
  );
  const pins = element("div", "hero-pins");
  [
    `Tree policy ${fixture.sources.tree.policyVersion}`,
    `${fixture.capability.nodeCount} capability nodes`,
    `${fixture.powerStress.surfaces.length} separate power surfaces`,
  ].forEach((text) => pins.append(element("span", "", text)));
  copy.append(pins);

  const fuse = element("aside", "release-fuse");
  fuse.setAttribute("aria-label", "Release fuse");
  fuse.append(
    element("p", "card-kicker", "Hard release fuse"),
    element("strong", "fuse-value", "0 ZRN"),
    element("p", "fuse-state", "Settlement disabled"),
  );
  const gateGrid = element("dl", "fuse-gates");
  gateGrid.append(
    metric("Bounded model checks", `${fixture.release.modelPassCount} pass`),
    metric(
      "Production integrations",
      `${fixture.release.integrationFailCount} closed`,
    ),
    metric("Chain state reads", "none"),
    metric("Claimable balance", "none"),
  );
  fuse.append(gateGrid, element("p", "boundary-copy", LAB_BOUNDARY));
  hero.append(copy, fuse);
  return hero;
}

function renderCapability(fixture: PowerLabFixture): HTMLElement {
  const section = element("section", "lab-section capability-section");
  section.setAttribute("aria-labelledby", "capability-title");
  const heading = sectionHeading(
    "01",
    "Capability context",
    "Math is a prerequisite map, not a payout ladder.",
    "These four canonical mathematics nodes constrain later work. They do not certify a person, unlock a claim, or advance the shadow epoch below.",
  );
  heading.querySelector("h2")?.setAttribute("id", "capability-title");
  section.append(heading);

  const levels = element("div", "math-levels");
  mathNodeLevels(fixture.capability.mathNodes).forEach((nodes, depth) => {
    const level = element("div", "math-level");
    level.append(element("p", "level-label", `Prerequisite depth ${depth}`));
    const cards = element("div", "math-level-cards");
    nodes.forEach((node) => {
      const card = element("article", "math-node");
      card.append(
        element("span", "node-mark", depth === 0 ? "ROOT" : `D${depth}`),
        element("h3", "", node.title),
        element("code", "", node.id),
        element(
          "p",
          "node-prerequisite",
          node.prerequisites.length === 0
            ? "No prerequisites"
            : `Requires ${node.prerequisites.join(" + ")}`,
        ),
        element("strong", "qualification-only", "Qualification only · 0 ZRN"),
      );
      cards.append(card);
    });
    level.append(cards);
    levels.append(level);
  });
  section.append(levels);

  const questNote = element("aside", "context-note");
  questNote.append(
    element("strong", "", `${fixture.capability.questCount} bounded quests exist in tree v1.`),
    element(
      "p",
      "",
      "Their sponsor-milestone eligibility is curriculum metadata, not an active sponsor, protocol issuance path, or entitlement.",
    ),
  );
  section.append(questNote);
  return section;
}

function renderPowerSurface(
  surface: PowerSurface,
  fixture: PowerLabFixture,
): HTMLElement {
  const card = element(
    "article",
    `surface-card ${surface.passesIllustrativeFloor ? "floor-met" : "floor-missed"}`,
  );
  const header = element("div", "surface-header");
  const seed = element("span", "surface-seed");
  seed.setAttribute("aria-hidden", "true");
  const title = element("div");
  title.append(
    element("p", "surface-id", surface.id),
    element("h3", "", POWER_SURFACE_LABELS[surface.id]),
  );
  header.append(seed, title);

  const status = element(
    "p",
    "surface-status",
    surface.passesIllustrativeFloor
      ? "Meets this illustrative floor"
      : "Below this illustrative floor",
  );
  const progressLabel = `${POWER_SURFACE_LABELS[surface.id]} effective controller count ${surface.effectiveCount.toFixed(3)}; illustrative floor ${fixture.powerStress.minimumEffectiveCount.toFixed(3)}`;
  const progress = element("progress", "power-stem");
  progress.max = fixture.powerStress.minimumEffectiveCount * 2;
  progress.value = Math.min(progress.max, surface.effectiveCount);
  progress.setAttribute("aria-label", progressLabel);

  const metrics = element("dl", "surface-metrics");
  metrics.append(
    metric("Effective count", surface.effectiveCount.toFixed(3)),
    metric("HHI", surface.hhi.toFixed(3)),
    metric(
      `Nakamoto count @ ${formatPercent(surface.coalitionThreshold)}`,
      String(surface.nakamotoCount),
    ),
    metric("Largest controller", formatPercent(surface.largestShare)),
  );
  card.append(header, status, progress, metrics);
  return card;
}

function renderPowerGarden(fixture: PowerLabFixture): HTMLElement {
  const section = element("section", "lab-section power-section");
  section.setAttribute("aria-labelledby", "power-title");
  const heading = sectionHeading(
    "02",
    "Twelve surfaces",
    "No healthy average can hide a captured path.",
    "This is a synthetic captured-policy stress snapshot. It is not live, has no time series, and is deliberately not the power input used by the illustrative allocation later on this page.",
  );
  heading.querySelector("h2")?.setAttribute("id", "power-title");
  section.append(heading);

  const separation = element("aside", "scenario-separation");
  separation.append(
    element("strong", "", "Scenario boundary"),
    element(
      "p",
      "",
      "Power garden = captured-policy stress. Shadow epoch = balanced-policy allocation. Side-by-side does not mean causally linked.",
    ),
  );
  section.append(separation);

  const garden = element("div", "power-garden");
  fixture.powerStress.surfaces.forEach((surface) =>
    garden.append(renderPowerSurface(surface, fixture)),
  );
  section.append(garden);

  const missing = element("div", "missing-observations");
  [
    ["Time series", fixture.powerStress.history],
    ["Controller uncertainty", fixture.powerStress.uncertainty],
    ["Joint-control path cut", fixture.powerStress.jointPathCut],
  ].forEach(([label, value]) => {
    const item = element("article");
    item.append(
      element("span", "", "UNAVAILABLE"),
      element("h3", "", String(label)),
      element(
        "p",
        "",
        value === null
          ? "A single synthetic snapshot cannot supply this required production view."
          : "Unexpected fixture state.",
      ),
    );
    missing.append(item);
  });
  section.append(missing);
  return section;
}

const LEDGER_LABELS: Record<(typeof LEDGER_LANE_IDS)[number], string> = {
  validity: "Validity",
  "novelty-priority": "Novelty & priority",
  "significance-consequence": "Significance & consequence",
  "attribution-credit": "Attribution & credit",
  "funding-liability": "Funding & liability",
  "governance-authority": "Governance & authority",
};

function renderLedgers(fixture: PowerLabFixture): HTMLElement {
  const section = element("section", "lab-section ledger-section");
  section.setAttribute("aria-labelledby", "ledger-title");
  const heading = sectionHeading(
    "03",
    "Six separate ledgers",
    "Different questions stay different.",
    "The fixture refuses to compress truth, novelty, consequence, credit, funding and authority into one score. Most lanes remain explicitly partial or absent.",
  );
  heading.querySelector("h2")?.setAttribute("id", "ledger-title");
  section.append(heading);

  const lanes = element("ol", "ledger-lanes");
  fixture.ledgerLanes.forEach((lane, index) => {
    const item = element("li", "ledger-lane");
    item.append(
      element("span", "lane-number", String(index + 1).padStart(2, "0")),
      element("h3", "", LEDGER_LABELS[lane.id]),
      element("code", "lane-status", lane.status),
      element("p", "", lane.detail),
    );
    lanes.append(item);
  });
  section.append(lanes);
  return section;
}

function renderCluster(cluster: ShadowCluster): HTMLElement {
  const card = element("article", "cluster-card");
  const header = element("div", "cluster-header");
  const identity = element("div");
  identity.append(
    element("p", "card-kicker", "Synthetic semantic cluster"),
    element("h3", "", cluster.id),
    element(
      "p",
      "cluster-artifacts",
      `${cluster.artifactCount} audit-only artifact IDs`,
    ),
  );
  const settlement = element("div", "cluster-settlement");
  settlement.append(
    element("span", "", "Settlement"),
    element("strong", "", "0 ZRN"),
    element("small", "", "blocked"),
  );
  header.append(identity, settlement);

  const allocation = element("div", "allocation-track");
  const allocationLabel = `Counterfactual allocation ${formatModelUnits(cluster.counterfactualFunded)} of eligible demand ${formatModelUnits(cluster.eligibleDemand)}`;
  const progress = element("progress");
  progress.max = cluster.eligibleDemand;
  progress.value = cluster.counterfactualFunded;
  progress.setAttribute("aria-label", allocationLabel);
  allocation.append(
    element("span", "", "Counterfactual allocation / eligible demand"),
    progress,
  );

  const measures = element("dl", "cluster-measures");
  measures.append(
    metric("Economic high-water", formatPercent(cluster.highWater)),
    metric("Lifetime cap", formatModelUnits(cluster.lifetimeCap)),
    metric("New gross accrual", formatModelUnits(cluster.newGrossAccrual)),
    metric("Model allocation", formatModelUnits(cluster.counterfactualFunded)),
    metric("Unfunded demand", formatModelUnits(cluster.unfundedDemand)),
    metric(
      "Direct / commons",
      `${formatModelUnits(cluster.direct)} / ${formatModelUnits(cluster.commons)}`,
    ),
  );

  const absent = element("ul", "cluster-absent");
  [
    ["Tree receipt", cluster.canonicalTreeReceipt],
    ["E0–E6 milestone", cluster.evidenceMilestone],
    ["Extinguished-to-date", cluster.extinguishedToDate],
    ["Expiry lots", cluster.eligibilityLots],
  ].forEach(([label, value]) => {
    absent.append(
      element(
        "li",
        "",
        `${String(label)}: ${value === null ? "not modelled" : "unexpected"}`,
      ),
    );
  });
  card.append(header, allocation, measures, absent);
  return card;
}

function renderShadowEpoch(fixture: PowerLabFixture): HTMLElement {
  const section = element("section", "lab-section epoch-section");
  section.setAttribute("aria-labelledby", "epoch-title");
  const heading = sectionHeading(
    "04",
    "Shadow allocation",
    "Arithmetic can be visible without becoming value.",
    "This first synthetic epoch uses balanced policy surfaces and model units. “Allocated” below means a counterfactual calculator output—not escrow, a liability, a transferable credit, or money owed.",
  );
  heading.querySelector("h2")?.setAttribute("id", "epoch-title");
  section.append(heading);

  const totals = element("dl", "epoch-totals");
  totals.append(
    metric("Model budget", formatModelUnits(fixture.shadowEpoch.budget)),
    metric("Direct", formatModelUnits(fixture.shadowEpoch.direct)),
    metric("Commons", formatModelUnits(fixture.shadowEpoch.commons)),
    metric("Unallocated", formatModelUnits(fixture.shadowEpoch.unallocated)),
    metric(
      "Unfunded demand",
      formatModelUnits(fixture.shadowEpoch.unfundedDemand),
    ),
    metric("Transferable settlement", "0 ZRN"),
  );
  section.append(totals);

  const clusters = element("div", "cluster-grid");
  fixture.shadowEpoch.clusters.forEach((cluster) =>
    clusters.append(renderCluster(cluster)),
  );
  section.append(clusters);

  const backlogWarning = element("aside", "backlog-warning");
  backlogWarning.append(
    element("strong", "", "Unfunded demand is not debt."),
    element(
      "p",
      "",
      "This floating-point allocation has no eligibility-lot scheduler or collateralized reserve. The exact capacity machine below is a separate counterfactual and does not turn this backlog into a claim.",
    ),
  );
  section.append(backlogWarning);
  return section;
}

const CAPACITY_EVENT_LABELS: Record<CapacityShadowStep["event"], string> = {
  accrue: "Ordinary accrual",
  fund: "Partial funding",
  "final-invalidation": "Final invalidation",
  reattribute: "Clean successor",
};

function renderCapacityStep(step: CapacityShadowStep): HTMLElement {
  const card = element("article", "capacity-step");
  card.append(
    element("p", "card-kicker", `Epoch ${step.epoch} · exact integer`),
    element("h3", "", CAPACITY_EVENT_LABELS[step.event]),
    element("code", "capacity-event", step.event),
  );

  const partition = element("div", "capacity-partition");
  partition.setAttribute(
    "aria-label",
    `Funded ${step.funded}, live ${step.live}, quarantined ${step.quarantined}, extinguished ${step.extinguished}, accrued ${step.accrued} model units`,
  );
  for (const [kind, value] of [
    ["funded", step.funded],
    ["live", step.live],
    ["quarantined", step.quarantined],
    ["extinguished", step.extinguished],
  ] as const) {
    if (value === 0) continue;
    const segment = element("span", `capacity-${kind}`);
    segment.style.flexGrow = String(value);
    segment.title = `${kind}: ${value} model units`;
    partition.append(segment);
  }

  const measures = element("dl", "capacity-measures");
  measures.append(
    metric("A · accrued", `${step.accrued} MU`),
    metric("Z · funded", `${step.funded} MU`),
    metric("L · live", `${step.live} MU`),
    metric("Y · quarantine", `${step.quarantined} MU`),
    metric("X · extinguished", `${step.extinguished} MU`),
    metric("R · reattributed", `${step.replacementUsed} MU`),
  );
  card.append(
    partition,
    measures,
    element("p", "capacity-deadline", `Inherited deadline · epoch ${step.deadline}`),
  );
  return card;
}

function renderCapacityShadow(fixture: PowerLabFixture): HTMLElement {
  const section = element("section", "lab-section capacity-section");
  section.setAttribute("aria-labelledby", "capacity-title");
  const heading = sectionHeading(
    "05",
    "One-shot quarantine machine",
    "Invalid work cannot mint a second cap.",
    "This separate exact-integer trace moves only still-unpaid capacity. A raw challenge changes nothing; final invalidation quarantines a bounded slice; a clean successor receives only supported headroom and inherits the original deadline.",
  );
  heading.querySelector("h2")?.setAttribute("id", "capacity-title");
  section.append(heading);

  const identity = element("aside", "capacity-identity");
  identity.append(
    element("code", "", "A = Z + L + Y + X"),
    element("code", "", "R + Y ≤ R̄"),
    element(
      "p",
      "",
      "A, Z, X and R never decrease. Z and X are terminal. Reattribution changes neither accrued, funded nor extinguished capacity.",
    ),
  );
  section.append(identity);

  const trace = element("div", "capacity-trace");
  fixture.capacityShadow.trace.forEach((step) =>
    trace.append(renderCapacityStep(step)),
  );
  section.append(trace);

  const boundary = element("aside", "capacity-boundary");
  boundary.append(
    element("strong", "", "Accounting proof, not authority"),
    element(
      "p",
      "",
      "The fixed 100 → 30 funded → 60 quarantined + 10 extinguished → 50 live successor vector passes exact conservation. Receipts, adjudication, identity, escrow and settlement remain external and closed.",
    ),
    element("span", "", "0 ZRN · integration false"),
  );
  section.append(boundary);
  return section;
}

function renderGateList(title: string, gates: Gate[], open: boolean): HTMLElement {
  const details = element("details", "gate-group");
  details.open = open;
  const summary = element("summary");
  summary.append(
    element("span", "", title),
    element("strong", "", `${gates.length}`),
  );
  details.append(summary);
  const list = element("ul", "gate-list");
  gates.forEach((gate) => {
    const item = element("li", gate.passed ? "gate-pass" : "gate-closed");
    item.append(
      element("span", "gate-mark", gate.passed ? "PASS" : "CLOSED"),
      element("div", "gate-copy"),
    );
    const copy = item.querySelector<HTMLDivElement>(".gate-copy");
    copy?.append(
      element("strong", "", gate.name),
      element("p", "", gate.detail),
    );
    list.append(item);
  });
  details.append(list);
  return details;
}

function renderRelease(fixture: PowerLabFixture): HTMLElement {
  const section = element("section", "lab-section release-section");
  section.setAttribute("aria-labelledby", "release-title");
  const heading = sectionHeading(
    "06",
    "Fail-closed release",
    "Passing the model is not passing production.",
    "The bounded arithmetic checks pass. Every required chain integration remains closed, so the only truthful settlement is zero.",
  );
  heading.querySelector("h2")?.setAttribute("id", "release-title");
  section.append(heading);

  const verdict = element("div", "release-verdict");
  verdict.append(
    element("div", "verdict-zero", "0 ZRN"),
    element("div", "verdict-copy"),
  );
  verdict.querySelector<HTMLDivElement>(".verdict-copy")?.append(
    element("strong", "", "No release. No claim. No chain action."),
    element(
      "p",
      "",
      `${fixture.release.modelPassCount} model checks pass; ${fixture.release.integrationFailCount} production integrations fail closed.`,
    ),
  );
  section.append(verdict);

  const modelGates = fixture.release.gates.filter(
    (gate) => gate.class === "model",
  );
  const integrationGates = fixture.release.gates.filter(
    (gate) => gate.class === "integration",
  );
  const groups = element("div", "gate-groups");
  groups.append(
    renderGateList("Bounded model checks", modelGates, false),
    renderGateList("Closed production integrations", integrationGates, true),
  );
  section.append(groups);
  return section;
}

function renderFooter(fixture: PowerLabFixture): HTMLElement {
  const footer = element("footer", "lab-footer");
  footer.append(
    element("strong", "", "Fixture provenance"),
    element(
      "p",
      "",
      `Tree ${fixture.sources.tree.schema} · policy ${fixture.sources.tree.policyVersion} · snapshot ${fixture.sources.tree.snapshotDate}`,
    ),
    element(
      "code",
      "",
      `tree sha256:${fixture.sources.tree.sha256.slice(0, 16)}…`,
    ),
    element(
      "code",
      "",
      `simulation sha256:${fixture.sources.simulation.sha256.slice(0, 16)}…`,
    ),
    element(
      "code",
      "",
      `shadow ledger sha256:${fixture.sources.shadowLedger.sha256.slice(0, 16)}…`,
    ),
    element(
      "p",
      "footer-boundary",
      "Checked-in deterministic fixture. No API, wallet, local storage, telemetry, transaction or network observation.",
    ),
  );
  return footer;
}

function renderLab(fixture: PowerLabFixture): void {
  root.replaceChildren(
    renderHero(fixture),
    renderCapability(fixture),
    renderPowerGarden(fixture),
    renderLedgers(fixture),
    renderShadowEpoch(fixture),
    renderCapacityShadow(fixture),
    renderRelease(fixture),
    renderFooter(fixture),
  );
}

function boot(): void {
  if (!LOCAL_HOSTS.has(window.location.hostname)) {
    renderFailure(
      "This research surface is deliberately unavailable outside loopback. Run `npm run lab` from dashboard/.",
    );
    return;
  }
  try {
    renderLab(parsePowerLabFixture(rawFixture));
  } catch (error) {
    const message =
      error instanceof Error
        ? error.message
        : "The checked-in fixture failed validation.";
    renderFailure(message);
  }
}

boot();
