import { observerLabel, type NetworkProfile } from "./network-profile";

const escape = (value: string): string => value.replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[char]!);

// The observer keeps the existing shell, hero, metric cards and block table.
// Render at build time as well as in development: no-JS users must never see
// legacy live claims or wallet controls while a beta profile is selected.
export function observerPage(profile: NetworkProfile): string {
  const configured = profile.mode !== "legacy" && profile.mode !== "unconfigured";
  const title = escape(observerLabel(profile));
  const chain = configured ? escape(profile.chainId) : "Unconfigured";
  const details = configured ? [
    ["Chain identity", profile.chainId], ["Gateway (operator approved)", profile.gatewayOrigin],
    ["Genesis SHA-256 (operator supplied, not queried)", profile.genesisSha256],
    ["Release commit", profile.releaseCommit], ["Release manifest SHA-256", profile.releaseManifestSha256],
    ["OPEN decision SHA-256 (operator verification assertion)", profile.operatorVerification.openDecisionSha256],
    ["Operator verification timestamp", profile.operatorVerification.verifiedAt],
    ...(profile.checkpoint ? [["Fixed archive height", String(profile.checkpoint.height)], ["Fixed block hash", profile.checkpoint.blockHash], ["Fixed header app hash", profile.checkpoint.appHash]] : []),
  ].map(([label, value]) => `<dt>${escape(label!)}</dt><dd><code>${escape(value!)}</code></dd>`).join("") : "<dt>Configuration</dt><dd>No network request is permitted. Supply a validated profile; legacy is never substituted.</dd>";
  return `<!doctype html>
<html lang="en"><head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" />
<meta name="theme-color" content="#09090c" /><meta name="description" content="Read-only observer. Configuration is not launch authority; no wallet, broadcast or rewards." />
<link rel="icon" href="/favicon.svg" type="image/svg+xml" /><title>ZERONE — ${title}</title></head>
<body><a class="skip-link" href="#main-content">Skip to observer</a><div class="app-shell">
<header class="topbar"><a class="brand" href="#overview"><span class="brand-mark" aria-hidden="true"><i></i><i></i></span><span>ZERONE</span></a>
<nav class="topnav" aria-label="Primary navigation"><a href="#onboarding">Scope</a><a href="#activity">Blocks</a><a href="#lookup">Lookup</a><a href="#release">Release evidence</a></nav>
<div class="network-pill" id="network-pill" data-state="loading" aria-live="polite"><span id="network-pill-label">Not checked</span></div></header>
<main id="main-content">
<section class="hero section" id="overview" aria-labelledby="hero-title"><div class="hero-copy"><div class="eyebrow">${title}</div><h1 id="hero-title">Read the record.<br /><em>Keep its limits.</em></h1>
<p class="hero-lede">${chain}. No account, wallet connection or credentials needed. The configured identity is not evidence that this network is live.</p><a class="button button-primary" href="#release">Inspect release evidence</a></div>
<div class="hero-orbit" aria-label="Observed block summary"><div class="orbit-ring ring-one"></div><div class="orbit-ring ring-two"></div><div class="orbit-core"><span class="orbit-kicker">Observed block</span><strong id="hero-height">—</strong><span id="hero-block-age">Not checked</span></div><div class="orbit-note orbit-note-bottom"><span>State</span><strong id="hero-state">Unknown</strong></div></div></section>
<aside class="truth-banner section" id="onboarding"><div><p class="eyebrow">${title}</p><h2>${!configured ? "No network selected. No network requests are made." : profile.mode === "preview" ? "Synthetic / local only. No production evidence." : profile.mode === "beta-archive" ? "Intentionally frozen predecessor. Not a stalled active chain." : "Public custodial beta; one operator-controlled validator; f=0."}</h2>
<p>This is the configured release scope, not a decentralisation or availability verdict. The validator count below is a separate observation. Configuration syntax and matching hashes do not authenticate an operator, a release, signatures or OPEN authority.</p>
<p>Hosted onboarding, wallet connection, sends, sponsorship, claims, migration payouts, IBC, liquidity and rewards are disabled here. Already-funded native users must use their own independently verified node. Registration does not supply gas, funds or validator membership.</p>
<p>Historical nominal 1:1 opt-in allocation is a policy direction, not an address balance entitlement. Eligibility, reserves and payout authority are not established by this observer.</p>
<noscript><p>JavaScript is off: no chain check has run. All values remain unknown. The release references and capability limits on this page remain readable.</p></noscript></div></aside>
<section class="metrics section" aria-label="Observed network metrics"><article class="metric-card"><div class="metric-label">Native supply</div><strong id="supply-value">Unknown</strong><p>ZRN · latest query, not a migration entitlement</p></article><article class="metric-card"><div class="metric-label">Validators</div><strong id="validator-value">Unknown</strong><p>Observed consensus count, not independent control</p></article><article class="metric-card"><div class="metric-label">Node peers</div><strong>Unavailable</strong><p>Private net_info is not queried</p></article><article class="metric-card"><div class="metric-label">Capabilities</div><strong>Read-only</strong><p>No hosted transaction or reward lane</p></article></section>
<section class="section" id="activity"><div class="section-heading"><h2>Recent blocks</h2><button class="button button-ghost" id="observer-refresh" disabled>Refresh record</button></div><p id="observer-status" role="status">Not checked. Requires JavaScript and the configured query gateway.</p><p>At most four sequential point reads; cached locally. Data is a gateway observation, not a light-client proof.</p><div class="block-table-wrap"><table class="block-table"><thead><tr><th>Height</th><th>Time (UTC)</th><th>Transactions</th><th>Block hash</th></tr></thead><tbody id="block-rows"><tr><td colspan="4">Unknown — no request yet</td></tr></tbody></table></div></section>
<section class="section onboarding" id="lookup"><h2>Inspect a known record</h2><p>Native address, block height, or known transaction hash only. No address history search. A missing or unavailable transaction is not proof it never existed.</p>
<form id="observer-lookup" class="ci-tree-controls"><label>Record type <select id="lookup-kind" disabled><option value="block">Block height</option><option value="transaction">Known transaction hash</option><option value="address">Native address</option></select></label><label>Public identifier <input id="lookup-value" maxlength="128" required autocomplete="off" disabled /></label><button class="button button-ghost" id="lookup-submit" disabled>Inspect record</button></form><p id="lookup-result" role="status">No lookup performed. Never enter secrets or seed phrases.</p></section>
<section class="section onboarding" id="release"><h2>Release binding and limits</h2><dl class="observer-evidence">${details}</dl>
<p>The gateway checks its origin. This client compares chain ID, point-block identity and status; archives additionally require their fixed height, block hash and header app hash. Genesis and release authority are operator-supplied references, not verified by these queries. Supply and validator reads may be from later heights, not an atomic snapshot.</p>
${configured && profile.mode !== "preview" ? `<p><a href="${escape(profile.releaseUrl)}" rel="noreferrer">Operator-supplied immutable release bundle</a> · <a href="https://github.com/cambridgetcg/zerone-core/blob/${escape(profile.releaseCommit)}/deploy/mainnet/JOIN.md" rel="noreferrer">Release-pinned observer / own-node instructions</a></p>` : "<p>No production release bundle is linked from this preview or unconfigured page.</p>"}
<p>Use only the reviewed artifact and genesis pinned by that release, verify its signatures and independent trust anchors, and follow its exact observer/full-node packet. Do not install moving main, infer validator admission, or use preview fixtures for production.</p>
<p>Knowledge and standards remain source-only material. They are not represented here as live successor facts, qualifications or Z2 rewards.</p></section>
</main><footer class="footer section"><p>Read-only observation. No transaction, allocation or activation authority.</p></footer></div><script type="module" src="/src/observer.ts"></script></body></html>`;
}
