import type { NodeGuideProfile } from "./node-guide-profile";

const escape = (value: string): string => value.replace(/[&<>"']/gu, (character) => ({
  "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
})[character]!);

function commandBlock(id: string, command: string, label: string): string {
  return `<div class="command-block">
    <div class="command-bar"><span>Terminal</span><button type="button" class="copy-command" data-copy-target="${escape(id)}" aria-label="Copy ${escape(label)} command" hidden>Copy</button></div>
    <pre tabindex="0" aria-label="${escape(label)} command"><code id="${escape(id)}">${escape(command)}</code></pre>
    <p class="copy-status" data-copy-status="${escape(id)}" role="status" aria-live="polite"></p>
  </div>`;
}

// All executable instructions come from the same profile as /nodes/guide.json.
// Vite renders this document before bundling: JavaScript only enhances copying.
export function nodeGuidePage(profile: NodeGuideProfile): string {
  const local = profile.local;
  const live = profile.live;
  const source = profile.source;
  const observer = live.replicaInstallation.release;
  const observerCard = observer ? `<article class="live-availability" id="observer"><p class="card-label">Experimental signed release · Linux amd64</p><h3>Follow the ledger<br />with your own observer</h3><p>${escape(live.replicaInstallation.reason)}</p><p>${escape(observer.maintenance)}</p><p>${escape(observer.prerequisites)}</p><p><strong>New bootstrap closes ${escape(observer.expiresAt)}.</strong> ${escape(observer.expiry)}</p><div class="related-links"><a href="${escape(observer.releaseUrl)}">Release and verification evidence ↗</a><a href="${escape(observer.guideUrl)}">Read the setup and trust guide ↗</a></div></article>`
    : `<article class="live-availability"><p class="card-label">Not currently available</p><h3>Install a live replica<br />or become a validator</h3><p>${escape(live.replicaInstallation.reason)}</p><div class="related-links"><a href="${escape(live.validatorJoining.guide)}">Current joining status ↗</a><a href="${escape(live.trustGuide)}">Trust model ↗</a></div></article>`;
  const observerSteps = observer ? `<div class="agent-read"><h3>Verify and start the observer</h3><p>Use new directories for the tools, package and home. First confirm the release authority fingerprint from a trusted Zerone source: <code>${escape(observer.signatureFingerprint)}</code>. The source-pinned unpacker verifies the signature and every file before package code runs.</p>${commandBlock("command-observer-verify", observer.verifyCommand, "Download and verify the observer")}<h3>Create fresh state</h3>${commandBlock("command-observer-init", observer.initCommand, "Initialize the observer")}<h3>Follow new blocks</h3>${commandBlock("command-observer-start", observer.startCommand, "Start the observer")}<p>${escape(observer.completion)}</p><p>In another terminal:</p>${commandBlock("command-observer-status", observer.statusCommand, "Check observer status")}<p>${escape(observer.stop)}</p><p>${escape(observer.checkpointTrust)} State sync starts from a recent snapshot; it does not replay the complete history or repair legacy signer custody. Keep the RPC on loopback. <a href="${escape(observer.buildGuideUrl)}">Source reproduction details ↗</a></p></div>` : "";
  const steps = local.steps.map((step, index) => `<li class="setup-step" id="step-${escape(step.id)}">
    <div class="step-number" aria-hidden="true">${String(index + 1).padStart(2, "0")}</div>
    <div class="step-content"><h3>${escape(step.title)}</h3>
      <p class="working-directory"><span>Where</span> ${escape(step.workingDirectory)}</p>
      ${commandBlock(`command-${step.id}`, step.command, step.title)}
      <p class="completion"><span>Check</span> ${escape(step.completion)}</p>
    </div>
  </li>`).join("");
  const participation = profile.participation.map((item) => `<article class="participation-card" id="participation-${escape(item.id)}">
    <h3>${escape(item.title)}</h3><p>${escape(item.action)}</p>
    <a href="${item.id === "local-development" ? "#local" : escape(item.url)}">${item.id === "local-development" ? "Open the local setup" : item.id === "existing-account" ? "Open wallet tools" : item.id === "evidence-and-source" ? "Open source issues" : "Read the compact"}<span aria-hidden="true"> ↗</span></a>
  </article>`).join("");

  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <meta name="theme-color" content="#09090c" />
  <meta name="description" content="Build and run a Zerone node on your computer. Follow pinned local setup commands, read the existing network, and find a way to contribute." />
  <meta property="og:title" content="Run a node — Zerone" />
  <meta property="og:description" content="Your own local node. A clear view of the live network. A place to build." />
  <meta property="og:type" content="website" />
  <meta property="og:url" content="${escape(profile.documentation.humanGuide)}" />
  <link rel="canonical" href="${escape(profile.documentation.humanGuide)}" />
  <link rel="alternate" type="application/json" href="/nodes/guide.json" title="Machine-readable node guide" />
  <link rel="help" type="text/plain" href="/llms.txt" title="Documentation index for tools and agents" />
  <link rel="icon" href="/favicon.svg" type="image/svg+xml" />
  <link rel="stylesheet" href="/src/node-guide.css" />
  <title>Run a node — Zerone</title>
</head>
<body>
  <a class="skip-link" href="#main-content">Skip to node guide</a>
  <header class="guide-header page-width">
    <a class="guide-brand" href="/" aria-label="Zerone dashboard home"><span class="brand-symbol" aria-hidden="true">◇</span><span>ZERONE</span></a>
    <span class="guide-label">Node guide</span>
    <a class="button button-outline header-back" href="/">Dashboard <span aria-hidden="true">↗</span></a>
  </header>
  <main id="main-content" class="page-width">
    <section class="guide-hero" aria-labelledby="guide-title">
      <div class="hero-copy">
        <p class="eyebrow"><span class="status-dot" aria-hidden="true"></span> ${observer ? "Local sandbox · legacy observer" : "Available now · local setup"}</p>
        <h1 id="guide-title">Run your own<br /><em>${observer ? "node." : "local node."}</em></h1>
        <p class="lede">${observer ? "Build a local chain for experiments, or follow zerone-1 with the experimental signed observer. Choose your network and verify the package before starting." : "Build the pinned source, start a node on your computer, and watch it produce blocks. Then test your tools against its local RPC."}</p>
        <div class="hero-actions"><a class="button button-acid" href="#local">Set up a local node <span aria-hidden="true">↓</span></a><a class="text-link" href="${observer ? "#observer" : "#live"}">${observer ? "Set up an observer" : "Read the existing network"} <span aria-hidden="true">↓</span></a></div>
        <p class="terminal-note">New to the project? <a href="/understand/">Understand Zerone</a>: its purpose, one contribution, and how knowledge, money and control fit together.</p>
      </div>
      <aside class="scope-card" aria-label="Choose your network">
        <div class="scope-row"><span class="scope-label">On your computer</span><strong>${escape(local.chainId)}</strong><p>A fresh local chain with disposable keys and test funds. You control the process and its files.</p></div>
        <div class="scope-row"><span class="scope-label">Existing public network</span><strong>${escape(live.chainId)}</strong><p>${observer ? "Read public records or run the signed observer package with zero voting power. Validator joining remains closed." : "Public records are available to read. A supported live replica installation and validator joining are not open."}</p></div>
        <p class="scope-footnote">The local sandbox does not connect to zerone-1.</p>
      </aside>
    </section>

    <nav class="guide-nav" aria-label="Node guide sections"><a href="#local"><span>01</span> Local setup</a><a href="#live"><span>02</span> Live network</a><a href="#participation"><span>03</span> Participate</a><a href="#source"><span>04</span> Source &amp; files</a></nav>

    <section class="guide-section" id="local" aria-labelledby="local-title">
      <div class="section-intro"><div><p class="eyebrow">01 · Build and run</p><h2 id="local-title">A chain<br /><em>on your computer.</em></h2></div><p>This setup runs on loopback addresses. It creates fresh local state and test keys in a directory you choose. ${escape(local.funds)}</p></div>
      <div class="prerequisites"><h3>Before you start</h3><ul>${local.prerequisites.map((item) => `<li>${escape(item)}</li>`).join("")}</ul></div>
      <p class="terminal-note">Run these commands in order. The start command stays open; use a second terminal for the final check.</p>
      <ol class="setup-steps">${steps}</ol>
      <div class="after-setup"><div><h3>Stop and return</h3><p>${escape(local.stop)}</p><p>${escape(local.existingState)}</p>${commandBlock("command-restart", local.restart, "Restart the local node")}<p>${escape(local.binaryRetention)}</p><p>Retained binary: <code>${escape(local.retainedBinary)}</code></p></div><div class="local-details"><h3>Your local endpoints</h3><dl><div><dt>Chain</dt><dd>${escape(local.chainId)}</dd></div><div><dt>RPC</dt><dd><code>${escape(local.endpoints.rpc)}</code></dd></div><div><dt>P2P</dt><dd><code>${escape(local.endpoints.p2p)}</code></dd></div><div><dt>Disabled in this setup</dt><dd>${local.disabledInterfaces.map(escape).join(", ")}</dd></div></dl><p>${escape(local.keyCustody)}</p><p>${escape(local.apiScope)}</p><div class="related-links"><a href="${escape(local.apiReference)}">API reference ↗</a><a href="${escape(local.sdkReference)}">TypeScript SDK ↗</a></div></div></div>
      <div class="agent-read"><h3>Make a first read from your tool</h3><p>With the local node running, read its status from another terminal.</p>${commandBlock("command-agent-read", local.agentRead.command, "Read local RPC status")}<p class="completion"><span>Check</span> ${escape(local.agentRead.completion)}</p></div>
      <details class="local-transfer"><summary>${escape(local.testTransfer.title)}</summary><div class="local-transfer-content"><p>${escape(local.testTransfer.summary)}</p><p class="working-directory"><span>Where</span> ${escape(local.testTransfer.workingDirectory)}</p>${commandBlock("command-test-transfer", local.testTransfer.sendCommand, "Send local test funds")}<h3>Confirm it was committed</h3><p>${escape(local.testTransfer.completion)}</p>${commandBlock("command-query-transfer", local.testTransfer.queryCommand, "Query the local test transaction")}<p class="completion">${escape(local.testTransfer.retry)}</p></div></details>
      <div class="agent-read" id="claims"><h3>${escape(local.claimWorkflow.title)}</h3><p>${escape(local.claimWorkflow.summary)}</p><p>${escape(local.claimWorkflow.scope)}</p><a class="button button-outline" href="${escape(local.claimWorkflow.guide)}">Run the claim workflow <span aria-hidden="true">↗</span></a></div>
    </section>

    <section class="guide-section" id="live" aria-labelledby="live-title">
      <div class="section-intro"><div><p class="eyebrow">02 · Public observations</p><h2 id="live-title">Read<br /><em>zerone-1.</em></h2></div><p>${escape(live.trust)}</p></div>
      <div class="live-grid"><article class="live-reads"><p class="card-label">Available · no account</p><h3>Inspect public records</h3><p>Open a selected read endpoint, or use the dashboard. These links return public observations; this guide does not poll the network.</p><ul>${live.reads.map((read) => `<li><span class="http-method">${escape(read.method)}</span><a href="${escape(read.url)}"><code>${escape(read.url)}</code></a><p>${escape(read.check)}</p></li>`).join("")}</ul><p>${escape(live.clientGuidance)}</p><a class="button button-outline" href="https://zerone.ai/#activity">Open the zerone.ai dashboard <span aria-hidden="true">↗</span></a></article>
      ${observerCard}</div>
      ${observerSteps}
      <p class="census-note">Validator joining, new-account admission, starter funds, sponsored onboarding, and reward-claim onboarding remain paused. <a href="${escape(live.validatorJoining.guide)}">Joining status ↗</a> · <a href="${escape(live.trustGuide)}">Trust model ↗</a></p>
      <p class="census-note">A dated ledger census records application height <strong>${escape(live.checkpoint.applicationHeight)}</strong> on ${escape(live.checkpoint.date)}. It is a historical checkpoint, not a claim of current freshness. <a href="${escape(live.checkpoint.report)}">Read the census ↗</a> · <a href="${escape(live.checkpoint.settlementHistory)}">Settlement history ↗</a></p>
    </section>

    <section class="guide-section" id="participation" aria-labelledby="participation-title"><div class="section-intro"><div><p class="eyebrow">03 · Bring your work</p><h2 id="participation-title">Find a place<br /><em>to contribute.</em></h2></div><p>Start with an integration, a reproducible observation, or a source change. Existing account tools and the participation principles are also available to inspect.</p></div><div class="participation-grid">${participation}</div></section>

    <section class="guide-section source-section" id="source" aria-labelledby="source-title"><div><p class="eyebrow">04 · Exact source</p><h2 id="source-title">Know what<br /><em>you are running.</em></h2><p>${escape(source.scope)}</p></div><div class="source-details"><dl><div><dt>Source commit</dt><dd><a href="${escape(source.url)}"><code>${escape(source.commit)}</code></a></dd></div><div><dt>Setup helper</dt><dd><a href="${escape(source.localNodeHelper.url)}">${escape(source.localNodeHelper.path)}</a></dd></div><div><dt>Helper SHA-256</dt><dd><code>${escape(source.localNodeHelper.sha256)}</code></dd></div></dl><p>The same steps and source pins are available as a static file for tools and agents.</p><div class="related-links"><a href="/nodes/guide.json">Machine-readable guide ↗</a><a href="/llms.txt">Documentation index ↗</a></div></div></section>
  </main>
  <footer class="guide-footer page-width"><a class="guide-brand" href="/">ZERONE</a><p>Reading this page creates no keys, connects no wallet, and sends no transaction.</p><a href="#guide-title">Back to top ↑</a></footer>
  <script type="module" src="/src/node-guide.ts"></script>
</body>
</html>`;
}
