import { buildObserverRelease, type ObserverPublication } from "./observer-release-profile";

const REPOSITORY = "https://github.com/cambridgetcg/zerone-core";
const HELPER_PATH = "scripts/local-node.py";

export interface NodeGuideBuildInputs {
  sourceCommit: string;
  helperSha256: string;
  observerPublication?: ObserverPublication;
}

function requirePin(value: string, length: number, name: string): void {
  if (
    typeof value !== "string" ||
    !new RegExp(`^[a-f0-9]{${length}}$`, "u").test(value) ||
    /^0+$/u.test(value)
  ) {
    throw new Error(`${name} must be a full lowercase hexadecimal pin`);
  }
}

/** Static documentation. Inputs are checked by the release build, not a network query. */
export function buildNodeGuideProfile({
  sourceCommit,
  helperSha256,
  observerPublication,
}: NodeGuideBuildInputs) {
  requirePin(sourceCommit, 40, "sourceCommit");
  requirePin(helperSha256, 64, "helperSha256");
  const source = (path: string) => `${REPOSITORY}/blob/${sourceCommit}/${path}`;
  const helperCommand = "python3 scripts/local-node.py";
  const localHome = '"$HOME/zerone-local"';
  const observerRelease = observerPublication === undefined ? null : buildObserverRelease(observerPublication);

  return {
    schema: "zerone.node-participation/v1",
    title: "Run a Zerone node and participate",
    documentation: {
      humanGuide: "https://zerone.ai/nodes/",
      machineGuide: "https://zerone.ai/nodes/guide.json",
      index: "https://zerone.ai/llms.txt",
      kind: "documentation",
      agentProtocolEndpoint: false,
    },
    source: {
      repository: REPOSITORY,
      commit: sourceCommit,
      url: `${REPOSITORY}/tree/${sourceCommit}`,
      localNodeHelper: {
        path: HELPER_PATH,
        url: source(HELPER_PATH),
        rawUrl: `https://raw.githubusercontent.com/cambridgetcg/zerone-core/${sourceCommit}/${HELPER_PATH}`,
        sha256: helperSha256,
      },
      scope: "Pinned source for the local sandbox; not the installed zerone-1 executable.",
      productionBinaryProvenance: false,
    },
    local: {
      id: "local-sandbox",
      title: "Run your own local node",
      availability: "available",
      chainId: "zerone-local-1",
      role: "local-validator",
      networkScope: "loopback-only",
      liveNetworkConnection: false,
      funds: "Local test funds only; no live ZRN balance or reward entitlement.",
      prerequisites: [
        "Go 1.25.14, Git, make and the platform C build tools.",
        "Python 3.11+ and a terminal on Linux or macOS.",
        "A new, dedicated home directory and free loopback ports 47656 and 47657.",
        "Network access for the source and Go dependencies during the build.",
      ],
      endpoints: {
        rpc: "http://127.0.0.1:47657",
        p2p: "tcp://127.0.0.1:47656",
        publicGateway: false,
      },
      disabledInterfaces: ["REST", "gRPC", "gRPC-Web"],
      agentRead: {
        command: "curl --fail --max-time 10 http://127.0.0.1:47657/status",
        completion: "Read result.node_info.network and result.sync_info.latest_block_height from your local node.",
      },
      testTransfer: {
        title: "Optional: send local test funds",
        summary: "Send 100 uzrn from the funded local user to the local validator, with a 250000 uzrn test fee. This signs a local transaction; it uses no live funds.",
        workingDirectory: "zerone-local-source, in the second terminal while the node remains running",
        sendCommand: [
          'LOCAL_HOME="$HOME/zerone-local"',
          'LOCAL_BIN="$LOCAL_HOME/bin/zeroned"',
          'TX_HASH=""',
          'python3 scripts/local-node.py status --home "$LOCAL_HOME" &&',
          'RECIPIENT=$("$LOCAL_BIN" keys show validator -a --home "$LOCAL_HOME" --keyring-backend test) &&',
          'LOCAL_TX_OUTPUT=$("$LOCAL_BIN" tx bank send user "$RECIPIENT" 100uzrn --home "$LOCAL_HOME" --chain-id zerone-local-1 --keyring-backend test --node http://127.0.0.1:47657 --fees 250000uzrn --gas 250000 --yes --output json) &&',
          'printf \'%s\\n\' "$LOCAL_TX_OUTPUT" &&',
          'TX_HASH=$(printf \'%s\' "$LOCAL_TX_OUTPUT" | python3 -c \'import json,re,sys; result=json.load(sys.stdin); txhash=result.get("txhash", ""); assert result.get("code") == 0 and re.fullmatch("[0-9A-Fa-f]{64}", txhash), "No accepted transaction hash returned"; print(txhash)\')',
        ].join("\n"),
        queryCommand: [
          'if [ -n "$TX_HASH" ]; then',
          '  "$LOCAL_BIN" query tx "$TX_HASH" --home "$LOCAL_HOME" --node http://127.0.0.1:47657 --output json',
          "else",
          "  printf 'No accepted transaction hash was captured.\\n'",
          "fi",
        ].join("\n"),
        completion: "In the same terminal, query the captured hash until the result has height greater than zero and code 0. The initial send response at height 0 is only mempool acceptance.",
        retry: "If the transaction is not indexed yet, retry only the query. Repeating the send creates another transaction and spends another local test fee.",
        localOnly: true,
      },
      steps: [
        {
          id: "source",
          title: "Get the pinned source",
          workingDirectory: "A directory where zerone-local-source does not exist",
          command: [
            `git clone ${REPOSITORY}.git zerone-local-source`,
            "cd zerone-local-source",
            `git checkout --detach ${sourceCommit}`,
          ].join("\n"),
          completion: `git rev-parse HEAD reports ${sourceCommit}.`,
        },
        {
          id: "build",
          title: "Build the local binary",
          workingDirectory: "zerone-local-source",
          command: [
            "GOTOOLCHAIN=go1.25.14 make build",
            "./build/zeroned version --long",
          ].join("\n"),
          completion: "The build succeeds and the binary reports the pinned source revision.",
        },
        {
          id: "verify-helper",
          title: "Check the setup helper",
          workingDirectory: "zerone-local-source",
          command: `python3 -c 'import hashlib; from pathlib import Path; actual = hashlib.sha256(Path("${HELPER_PATH}").read_bytes()).hexdigest(); assert actual == "${helperSha256}", "Setup helper digest mismatch"; print(actual)'`,
          completion: `The helper SHA-256 is ${helperSha256}.`,
        },
        {
          id: "init",
          title: "Create fresh local state",
          workingDirectory: "zerone-local-source",
          command: `${helperCommand} init --home ${localHome} --binary "$PWD/build/zeroned"`,
          completion: "A new local genesis, validator and funded test account are created in the chosen home.",
        },
        {
          id: "start",
          title: "Start your local node",
          workingDirectory: "zerone-local-source",
          command: `${helperCommand} start --home ${localHome}`,
          completion: "The foreground process confirms zerone-local-1 and advancing blocks. Leave this terminal open.",
        },
        {
          id: "status",
          title: "Check it from another terminal",
          workingDirectory: "zerone-local-source, in a second terminal",
          command: `${helperCommand} status --home ${localHome}`,
          completion: "The JSON status confirms the local node identity, chain and advancing height.",
        },
      ],
      stop: "Press Ctrl-C in the start terminal. The helper stops its own child and preserves your local state.",
      restart: `${helperCommand} start --home ${localHome}`,
      retainedBinary: "$HOME/zerone-local/bin/zeroned",
      binaryRetention: "Initialization retains the exact binary in your local home. Restart uses that copy, even if you later rebuild the source checkout.",
      keyCustody: "Fresh local test keys remain in your chosen home. Never import a live validator key or reuse test keys for real funds.",
      existingState: "Initialization refuses an existing home. Restart the same home to continue; setup is not a reset command.",
      apiReference: source("docs/API.md"),
      sdkReference: source("sdk/typescript/README.md"),
      claimWorkflow: {
        title: "Put a claim into a block",
        guide: source("docs/LOCAL-CLAIMS.md"),
        summary: "Create a fresh knowledge sandbox with funded development participants. Submit a signed claim, save reasoned reviews, record a counterclaim and inspect the retained history in a local browser page.",
        scope: "Local development only. The same operator controls these accounts; an accepted review outcome is not a guarantee of truth or independent scientific agreement.",
      },
      apiScope: "These APIs describe the pinned local source. Public legacy routes and messages may differ.",
    },
    live: {
      id: "zerone-1",
      title: "Read the existing network",
      availability: "public-reads",
      chainId: "zerone-1",
      role: "gateway-reader",
      localSandboxIsReplica: false,
      endpoints: {
        rpc: "https://zerone.ai/api/rpc",
        rest: "https://zerone.ai/api/rest",
        scope: "The website gateway exposes selected routes, not the whole source API.",
      },
      reads: [
        {
          method: "GET",
          url: "https://zerone.ai/api/rpc/status",
          check: "Check result.node_info.network is zerone-1, block age and catching_up before relying on freshness.",
        },
        {
          method: "GET",
          url: "https://zerone.ai/api/rest/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn",
          check: "Read the returned denom and amount; a failed request means unavailable, not zero.",
        },
      ],
      trust: "A custodial network with a disclosed sole-validator trust model. Gateway reads are observations, not independent proofs; the upstream node hop uses HTTP.",
      clientGuidance: "For automated reads, send an honest application User-Agent, such as zerone-node-guide/1.0, and set a timeout. Some generic client signatures are blocked by the gateway. Treat non-200 responses as unavailable.",
      trustGuide: source("deploy/mainnet/TRUST.md"),
      replicaInstallation: {
        availability: observerRelease ? "signed-legacy-observer" : "not-published",
        reason: observerRelease
          ? "A signed Linux amd64 observer package retains the legacy application with reviewed dependency fixes and follows the ledger with zero voting power. Check the bootstrap expiry before installing."
          : "A compatible public legacy observer binary and verified setup bundle are not published. Current source is not a drop-in live-node binary.",
        release: observerRelease,
      },
      validatorJoining: { availability: "not-open", guide: source("deploy/mainnet/JOIN.md") },
      newAccountAdmission: { availability: "paused", websiteSignup: false },
      starterFunds: { availability: "paused" },
      sponsorshipOnboarding: { availability: "paused" },
      rewardClaimOnboarding: { availability: "paused" },
      checkpoint: {
        kind: "dated-operator-observed-census",
        applicationHeight: "1261575",
        bindingHeaderHeight: "1261576",
        postCommitAppHash: "a381775589d84a74bbf365b9b2add26201465e20980cf50a53544f7a81256106",
        date: "2026-09-09",
        currentFreshnessClaim: false,
        independentTrustAnchor: false,
        report: source("docs/reports/authenticated-ledger-census-2026-09-09.md"),
        settlementHistory: source("docs/reports/knowledge-settlement-history-2026-09-09.md"),
      },
    },
    participation: [
      {
        id: "local-development",
        title: "Build an agent integration",
        availability: "available",
        action: "Run the local node and test against its local RPC using the pinned API and SDK.",
        url: "https://zerone.ai/nodes/#local",
        liveTransaction: false,
      },
      {
        id: "existing-account",
        title: "Use an existing zerone-1 account",
        availability: "existing-wallet-tools",
        action: "Connect Keplr to inspect an address and balance. Any supported transaction is a separate action you review and sign.",
        url: "https://zerone.ai/#wallet",
        admissionCreatedByConnection: false,
        fundingCreatedByConnection: false,
      },
      {
        id: "evidence-and-source",
        title: "Contribute evidence or code",
        availability: "available",
        action: "Compare public records, reproduce local results, and contribute a bounded issue or source change. Share public evidence, never keys or private account material.",
        url: `${REPOSITORY}/issues`,
        automaticPayment: false,
      },
      {
        id: "participation-principles",
        title: "Read the participation principles",
        availability: "static-document",
        action: "Read the covenant and adapter status. These documents do not enroll an account or open a service endpoint.",
        url: "https://zerone.ai/#participate",
        adapterIndex: "https://zerone.ai/standards/adapter-index.v1.json",
      },
    ],
    boundaries: {
      readingGuideRequestsWallet: false,
      liveNetworkMutationBySetup: false,
      localSetupWritesStateAndCreatesTestKeys: true,
      privateStateUploadRequired: false,
      admitsValidatorsOrNewLiveAccounts: false,
      activatesSuccessorNetwork: false,
      provesHistoricalPayments: false,
    },
  } as const;
}

export type NodeGuideProfile = ReturnType<typeof buildNodeGuideProfile>;
