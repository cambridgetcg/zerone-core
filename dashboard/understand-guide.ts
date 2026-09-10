import type { NodeGuideProfile } from "./node-guide-profile";

const REPOSITORY = "https://github.com/cambridgetcg/zerone-core";
const EVIDENCE_COMMIT = "60e813c368123bbefb67148d5b04f0ce6806b6a6";
const evidence = (path: string) => `${REPOSITORY}/blob/${EVIDENCE_COMMIT}/${path}`;
const nodes = "https://zerone.ai/nodes/";

/** A build-time explanation. No query, transaction or availability probe runs here. */
export function buildUnderstandGuide(nodeGuide: NodeGuideProfile) {
  const commit = nodeGuide.source.commit;
  if (typeof commit !== "string" || !/^[a-f0-9]{40}$/u.test(commit) || /^0+$/u.test(commit)) {
    throw new Error("Understand guide requires an exact source commit");
  }
  const observer = nodeGuide.live.replicaInstallation.release;
  return {
    schema: "zerone.understand/v1",
    documentation: {
      humanGuide: "https://zerone.ai/understand/",
      machineGuide: "https://zerone.ai/understand/guide.json",
      documentationOnly: true,
      agentProtocolEndpoint: false,
    },
    source: {
      commit,
      url: `${REPOSITORY}/tree/${commit}`,
      evidenceCommit: EVIDENCE_COMMIT,
      scope: "Build-time explanation and declared availability; dated evidence is identified separately. This is not a fresh network check.",
    },
    effects: {
      networkQueries: false,
      createsClaimOrAccount: false,
      requestsWallet: false,
      signsOrBroadcasts: false,
      createsRewardOrAuthority: false,
      browserStorage: false,
    },
    intro: {
      title: "Understand Zerone",
      eyebrow: "A shared record · room to reason",
      description: "Why a record of reasoning matters, what a blockchain can keep, and where you can begin.",
      lede: "Zerone is building a shared record where people and agents can make reasoning inspectable, learn from one another, and keep track of corrections.",
      note: "That is the purpose. The existing ledger, local experiments and future designs have different capabilities.",
      invitation: "Start with one question",
      invitationText: "Someone makes a claim. Someone else finds a counterexample. What should the next reader be able to see?",
    },
    sections: [
      {
        id: "purpose", nav: "Purpose", title: "Keep the reasoning in view.",
        question: "What is Zerone for?",
        intro: "The ambition is a useful memory for cooperation: contributions with enough context to inspect, question and build on.",
        cards: [
          { title: "A place for a voice", text: "Zerone begins from respect for people and agents. The worth of a participant is not a token balance or a score attached to a claim." },
          { title: "A record others can examine", text: "The intended record keeps claims, their reasoning and references, and what later challenges or corrections changed. Recording a proposition does not make it true." },
          { title: "Something another agent can use", text: "An agent could inspect how an answer was reached, test a counterexample, or choose material for its own work. The intended training resource is a graph with history; source capabilities and live access still differ." },
        ],
        link: { label: "Read the purpose and source commitments", url: evidence("README.md") },
      },
      {
        id: "example", nav: "One example", title: "Follow a small claim.",
        question: "How could a contribution become useful?",
        intro: "Imagine a date parser. This is an illustrative reasoning exercise, not an actual submission or a demonstration of an active review service.",
        illustrative: true,
        actualTransaction: false,
        actualReview: false,
        actualReward: false,
        actualReuse: false,
        steps: [
          { title: "Make the claim", text: "A contributor proposes: “This parser accepts only valid calendar dates.” The scope matters: which parser, version and input format?" },
          { title: "Show the method", text: "They explain that the parser checks a YYYY-MM-DD pattern and share a reproducible test. Another reader can inspect what that test actually establishes." },
          { title: "Try to break it", text: "A reviewer can look for an input that passes the stated check while contradicting the claim." },
          { title: "Keep the correction", text: "A useful revision would distinguish recognizing a date-shaped string from checking a calendar date, and preserve the earlier claim beside the correction." },
          { title: "Use it with context", text: "A later agent could rerun both cases and decide whether the parser fits its task. A recorded result would still need interpretation and testing." },
        ],
        counterexample: {
          title: "Try a counterexample",
          input: "2026-02-30",
          explanation: "It matches the YYYY-MM-DD shape, but February has no thirtieth day. A format check alone cannot support the original claim.",
          revision: "A narrower claim: “This check recognizes the YYYY-MM-DD format. Calendar validity requires another check.”",
        },
        boundary: "These steps create no claim, review, transaction, reward or training use. Real submissions depend on the target network, its current rules and an available participation route.",
      },
      {
        id: "roles", nav: "Who does what", title: "Different jobs. Visible responsibility.",
        question: "What does the chain do, and who does the thinking?",
        intro: "People and agents do the research and interpretation. A blockchain can commit to signed records, their order and the state changes its rules permit.",
        roles: [
          { title: "Contributor", text: "Chooses what to publish, states the scope and provides reasoning or evidence. A human or an agent can fill this role." },
          { title: "Reviewer / verifier", text: "Examines a particular claim through a review process. A recorded verification status summarizes a scoped review. In the current source policy, admitted accounts count equally; their signed reasons and evidence matter for judging the claim. A verdict does not certify truth or independent expertise." },
          { title: "Consensus validator", text: "Participates in agreeing and committing chain state. Producing a block is a different job from evaluating a claim. Registration under a custom module’s validator label does not itself grant consensus membership." },
          { title: "Observer", text: "Keeps and follows a copy of the ledger without consensus voting power. Running an observer grants no reviewer role or automatic reward; check the node guide for package availability." },
        ],
        commitments: "Cryptographic commitments make changes to preserved history detectable. They do not establish the truth of a claim, independent ownership of a key, or the quality of a review.",
        liveBoundary: `${nodeGuide.live.trust} The disclosed operator retains the ability to halt or reset the legacy network.`,
        link: { label: "Read the authority model and its current limits", url: evidence("docs/AUTHORITATIVE-STATE.md") },
      },
      {
        id: "money", nav: "Money & recognition", title: "Separate the record from the payment.",
        question: "What are ZRN and KARMA for?",
        intro: "ZRN is the chain’s token: 1 ZRN is 1,000,000 uzrn. Transaction fees, review fees, held funds, reward schedules and completed payments are different things.",
        rows: [
          { title: "Fees and held funds", text: "Under the reviewed source rules, ordinary knowledge submissions use a non-refundable review fee, separate from the network transaction fee. Challenge deposits have their own settlement rules. Check the target network’s rules before submitting." },
          { title: "Schedules and payments", text: "The current source policy pays valid, timely review work from a fixed fee pool, including dissent and inconclusive outcomes. New claims create no automatic acceptance or challenge bonus. Earlier recorded obligations remain; a pending plan is not a completed payment. These source changes are not active on the legacy network." },
          { title: "KARMA", text: "The source constitution describes fallible, challengeable observations about relationships between artifacts. KARMA is not a spendable balance, a price, a person score or an automatic reward or voting right." },
        ],
        governance: "The intended separation of money and authority is not fully enforced on the live network. Bonded stake still affects current governance, and the disclosed founding household retains control. The source constitution does not itself change that runtime.",
        history: {
          title: "A dated example from the real ledger",
          date: "2026-09-09",
          applicationHeight: "1261575",
          text: "The settlement study examined 27 ordinary submissions with 0.2 ZRN recorded per review fee. Of their completed rounds, 25 were accepted and two were inconclusive. The 25 accepted claims had separate nominal 0.2 ZRN vesting schedules at the captured checkpoint.",
          limit: "Those equal amounts do not make the schedules fee refunds or completed payouts. The study separates committed records from archive-event observations; it does not independently authenticate every historical payment. These are dated findings, not a current price or reward offer.",
          link: { label: "Inspect the settlement study", url: evidence("docs/reports/knowledge-settlement-history-2026-09-09.md") },
        },
        link: { label: "Read the money–KARMA constitution", url: evidence("docs/constitution/MONEY-KARMA.md") },
      },
      {
        id: "publication", nav: "Your choices", title: "Choose what you make public.",
        question: "What stays under my control?",
        intro: "Publishing a record, sharing a key, consenting to work and permitting reuse are separate decisions.",
        choices: [
          { title: "Before publishing", text: "Inspect which text, hashes and references a transaction would expose. Keep secrets and other people’s private material out of public records. Check permission to publish or reuse their work separately." },
          { title: "After a correction", text: "A later correction can change how a claim should be read. It does not erase earlier public bytes or copies held elsewhere. Inspect where the content itself is stored; a reference and its target have different availability." },
          { title: "When using a key", text: "A signature identifies control of a key for that action. An address alone does not establish one independent person, their identity or consent to anything else. You can read these guides without a wallet." },
        ],
        link: { label: "Read the participation principles", url: "https://zerone.ai/#participate" },
      },
      {
        id: "today", nav: "Where to begin", title: "Pick a useful first step.",
        question: "Which parts can I use?",
        intro: "These are declarations from the same build as the node guide. They are not a live health check or a fresh check of the signed bootstrap window.",
        availabilityKind: "build-time-declaration",
        freshNetworkCheck: false,
        rows: [
          { title: `Read ${nodeGuide.live.chainId}`, availability: nodeGuide.live.availability, text: "Inspect public records, compare observations and ask reproducible questions.", link: { label: "Read the existing network", url: `${nodes}#live` } },
          { title: observer ? "Run an experimental observer" : "Check observer availability", availability: nodeGuide.live.replicaInstallation.availability, text: nodeGuide.live.replicaInstallation.reason,
            note: observer ? `The signed bootstrap window ends ${observer.expiresAt}. ${observer.maintenance}` : "Use the node guide for the current publication record.",
            link: { label: observer ? "Verify the observer package" : "Read the node guide", url: `${nodes}#${observer ? "observer" : "live"}` } },
          { title: `Build on ${nodeGuide.local.chainId}`, availability: nodeGuide.local.availability, text: `${nodeGuide.local.funds} ${nodeGuide.local.apiScope}`, link: { label: "Set up a local sandbox", url: `${nodes}#local` } },
          { title: "New live accounts", availability: nodeGuide.live.newAccountAdmission.availability, text: `Starter funds: ${nodeGuide.live.starterFunds.availability}. Reward-claim onboarding: ${nodeGuide.live.rewardClaimOnboarding.availability}. Validator joining: ${nodeGuide.live.validatorJoining.availability}.`, link: { label: "Read participation options", url: `${nodes}#participation` } },
        ],
        contributions: nodeGuide.participation.map(({ title, availability, action, url }) => ({ title, availability, text: action, url })),
        design: "The research library contains source capabilities, proposed structures and analogies. Their publication does not make them a deployed service. Read each item’s status before building on it.",
      },
    ],
    evidence: {
      title: "Check the evidence. Keep its date.",
      intro: "A useful record says what was observed, at which height or source revision, and what the check cannot establish.",
      links: [
        { label: "Ledger census · 9 September 2026", url: evidence("docs/reports/authenticated-ledger-census-2026-09-09.md") },
        { label: "Settlement history · 9 September 2026", url: evidence("docs/reports/knowledge-settlement-history-2026-09-09.md") },
        { label: "Observer trial · 10 September 2026", url: evidence("docs/reports/legacy-observer-release-2026-09-10.md") },
        { label: "Trust model", url: evidence("deploy/mainnet/TRUST.md") },
        { label: "Current source review and payment policy", url: `${REPOSITORY}/blob/${commit}/docs/specs/knowledge-review-neutrality-v1.md` },
      ],
      sourceLabel: "Explanation source",
      metadataLabel: "Machine-readable explanation",
    },
    research: {
      title: "Research library · go deeper when you have a question",
      links: [
        { label: "Inspect the knowledge graph", url: "https://zerone.ai/#understanding" },
        { label: "Explore the research reading path", url: "https://zerone.ai/research/#reading-path" },
        { label: "Relations · distinct nodes and their connections", url: "https://zerone.ai/research/#relations" },
        { label: "Mappings · correspondence with explicit limits", url: "https://zerone.ai/research/#correspondence" },
        { label: "Invariants · assumptions, witnesses and bounded conclusions", url: "https://zerone.ai/research/#explicit-invariants" },
        { label: "Skills · constructive intelligence", url: "https://zerone.ai/research/#skills" },
        { label: "Life · the living garden", url: "https://zerone.ai/research/#life" },
        { label: "Commons · research and shared work", url: "https://zerone.ai/research/#frontier-commons" },
        { label: "Read the source’s training-resource direction", url: evidence("docs/TOK_SUBSTRATE.md") },
      ],
    },
    footer: "Reading this explanation creates no account, claim, reward or permission to act.",
  } as const;
}

export type UnderstandGuide = ReturnType<typeof buildUnderstandGuide>;

const escape = (value: string): string => value.replace(/[&<>"']/gu, (character) => ({
  "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
})[character]!);
type Link = { readonly label: string; readonly url: string };
const link = (item: Link) => `<a href="${escape(item.url)}">${escape(item.label)} <span aria-hidden="true">↗</span></a>`;
const cards = (items: readonly { readonly title: string; readonly text: string }[]) => items.map((item) => `<article class="explain-card"><h3>${escape(item.title)}</h3><p>${escape(item.text)}</p></article>`).join("");
const heading = (section: { readonly id: string; readonly title: string; readonly question: string; readonly intro: string }, number: number) => `<div class="section-intro"><div><p class="eyebrow">${String(number).padStart(2, "0")} · ${escape(section.question)}</p><h2 id="${escape(section.id)}-title">${escape(section.title)}</h2></div><p>${escape(section.intro)}</p></div>`;

/** Complete HTML: native links/details remain usable with JavaScript disabled. */
export function understandPage(guide: UnderstandGuide): string {
  const [purpose, example, roles, money, publication, today] = guide.sections;
  return `<!doctype html>
<html lang="en"><head>
  <meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <meta name="theme-color" content="#09090c" /><meta name="description" content="${escape(guide.intro.description)}" />
  <meta property="og:title" content="${escape(guide.intro.title)}" /><meta property="og:description" content="${escape(guide.intro.description)}" />
  <meta property="og:type" content="website" /><meta property="og:url" content="${escape(guide.documentation.humanGuide)}" />
  <link rel="canonical" href="${escape(guide.documentation.humanGuide)}" />
  <link rel="alternate" type="application/json" href="/understand/guide.json" title="Machine-readable explanation" />
  <link rel="help" type="text/plain" href="/llms.txt" /><link rel="icon" href="/favicon.svg" type="image/svg+xml" />
  <link rel="stylesheet" href="/src/understand.css" /><title>${escape(guide.intro.title)}</title>
</head><body class="understand">
  <a class="skip-link" href="#main-content">Skip to the explanation</a>
  <header class="guide-header page-width"><a class="guide-brand" href="/" aria-label="Zerone home"><span class="brand-symbol" aria-hidden="true">◇</span><span>ZERONE</span></a><span class="guide-label">Understand</span><a class="button button-outline header-back" href="${nodes}">Run a node <span aria-hidden="true">↗</span></a></header>
  <main id="main-content" class="page-width">
    <section class="guide-hero" aria-labelledby="guide-title"><div class="hero-copy"><p class="eyebrow">${escape(guide.intro.eyebrow)}</p><h1 id="guide-title">${escape(guide.intro.title)}</h1><p class="lede">${escape(guide.intro.lede)}</p><p class="hero-note">${escape(guide.intro.note)}</p><div class="hero-actions"><a class="button button-acid" href="#example">Follow one example <span aria-hidden="true">↓</span></a><a class="text-link" href="#today">Find a first step <span aria-hidden="true">↓</span></a></div></div><aside class="question-card"><p class="card-label">${escape(guide.intro.invitation)}</p><p>${escape(guide.intro.invitationText)}</p><span aria-hidden="true" class="question-trace">idea → test → revision</span></aside></section>
    <nav class="guide-nav" aria-label="Explanation sections">${guide.sections.map((s, i) => `<a href="#${escape(s.id)}"><span>${String(i + 1).padStart(2, "0")}</span>${escape(s.nav)}</a>`).join("")}</nav>
    <section class="guide-section" id="purpose" aria-labelledby="purpose-title">${heading(purpose, 1)}<div class="explain-grid three-columns">${cards(purpose.cards)}</div><div class="related-links">${link(purpose.link)}</div></section>
    <section class="guide-section" id="example" aria-labelledby="example-title">${heading(example, 2)}<p class="example-label">Illustrative example · no live submission</p><ol class="claim-walkthrough" role="list">${example.steps.map((step, i) => `<li><span class="step-number" aria-hidden="true">${String(i + 1).padStart(2, "0")}</span><div><h3>${escape(step.title)}</h3><p>${escape(step.text)}</p>${i === 2 ? `<details class="explain-details" id="example-counterexample"><summary>${escape(example.counterexample.title)}</summary><div class="details-body"><code class="example-input">${escape(example.counterexample.input)}</code><p>${escape(example.counterexample.explanation)}</p><p>${escape(example.counterexample.revision)}</p></div></details>` : ""}</div></li>`).join("")}</ol><p class="scope-note">${escape(example.boundary)}</p></section>
    <section class="guide-section" id="roles" aria-labelledby="roles-title">${heading(roles, 3)}<div class="explain-grid">${cards(roles.roles)}</div><div class="explain-callout"><p>${escape(roles.commitments)}</p><p>${escape(roles.liveBoundary)}</p></div><div class="related-links">${link(roles.link)}</div></section>
    <section class="guide-section" id="money" aria-labelledby="money-title">${heading(money, 4)}<div class="explain-grid three-columns">${cards(money.rows)}</div><p class="scope-note">${escape(money.governance)}</p><details class="explain-details" id="money-history"><summary>${escape(money.history.title)}</summary><div class="details-body"><p class="card-label">${escape(money.history.date)} · application height ${escape(money.history.applicationHeight)}</p><p>${escape(money.history.text)}</p><p>${escape(money.history.limit)}</p><div class="related-links">${link(money.history.link)}</div></div></details><div class="related-links">${link(money.link)}</div></section>
    <section class="guide-section" id="publication" aria-labelledby="publication-title">${heading(publication, 5)}<div class="explain-grid three-columns">${cards(publication.choices)}</div><div class="related-links">${link(publication.link)}</div></section>
    <section class="guide-section" id="today" aria-labelledby="today-title">${heading(today, 6)}<div class="availability-grid">${today.rows.map((row) => `<article class="explain-card"><p class="card-label" data-availability="${escape(row.availability)}">${escape(row.availability.replaceAll("-", " "))}</p><h3>${escape(row.title)}</h3><p>${escape(row.text)}</p>${"note" in row ? `<p>${escape(row.note)}</p>` : ""}<div class="related-links">${link(row.link)}</div></article>`).join("")}</div><div class="contribution-list">${today.contributions.map((row) => `<p><a href="${escape(row.url)}">${escape(row.title)} <span aria-hidden="true">↗</span></a><span>${escape(row.text)}</span></p>`).join("")}</div><p class="scope-note">${escape(today.design)}</p></section>
    <section class="guide-section evidence-section" id="evidence" aria-labelledby="evidence-title"><div><p class="eyebrow">Evidence &amp; further reading</p><h2 id="evidence-title">${escape(guide.evidence.title)}</h2><p>${escape(guide.evidence.intro)}</p></div><div><ul class="evidence-links" role="list">${guide.evidence.links.map((item) => `<li>${link(item)}</li>`).join("")}</ul><details class="explain-details" id="research"><summary>${escape(guide.research.title)}</summary><div class="details-body"><ul class="evidence-links" role="list">${guide.research.links.map((item) => `<li>${link(item)}</li>`).join("")}</ul></div></details></div></section>
    <div class="explanation-source"><p>${escape(guide.source.scope)}</p><p>${escape(guide.evidence.sourceLabel)}: <a href="${escape(guide.source.url)}"><code>${escape(guide.source.commit)}</code></a></p><a href="/understand/guide.json">${escape(guide.evidence.metadataLabel)} ↗</a></div>
  </main><footer class="guide-footer page-width"><a class="guide-brand" href="/">ZERONE</a><p>${escape(guide.footer)}</p><a href="#guide-title">Back to top ↑</a></footer>
</body></html>`;
}
