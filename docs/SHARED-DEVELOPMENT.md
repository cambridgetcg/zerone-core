# Shared development: zerone-dev-1

Use this separate development network to submit a claim, have someone else inspect it, and leave a reasoned review or counterexample. Each participant keeps their own keys and decides what to sign. The initial consensus has one operator-controlled validator seat; different account addresses do not establish independent controllers or scientific endorsement.

**Check [the development page](https://zerone.ai/development/) first.** Until it contains a dated verified publication, deployment and admission are pending. This source guide alone does not announce a live network. The page's [machine-readable profile](https://zerone.ai/development/guide.json) carries the publication pins when available. An operator verification date is not a current health guarantee.

This is **zerone-dev-1**, separate from the legacy **zerone-1**, local `zerone-local-*` sandboxes, and the unactivated production successor **zerone-2**. Existing zerone-1 admission and wallet gates are unchanged. Development funds have no cash value, promised future allocation or implied scientific credibility; existing development fee and review-settlement rules still apply. A reset must use a new chain ID and publish a notice preserving old public history; an existing participant home must not be rebound silently.

## 1. Verify the source and network packet

You need Python 3.11+ and a terminal on Linux x86-64 or macOS Apple Silicon for the compatible packages. Building from source additionally needs Git, Go 1.25.14 and the platform C build tools. Use a new dedicated home. This client uses an **unencrypted SDK test keyring** protected by local file permissions; it is for valueless development funds. Keep the entire home private and out of shared folders, source control and public backups. Onboarding also creates a separate Ed25519 identity file; retaining only the signing key is not a complete backup.

Read the exact publication and package record through [the development page](https://zerone.ai/development/#packages). Its `release.json` link is pinned to the website's full source commit, outside the downloaded archive. Confirm that source through a channel you trust. A hash supplied only inside the same untrusted archive does not authenticate it. Package source and the deployed server use the record's `source_commit`; the website and these instructions can have a later commit. Checksums identify bytes, not a security audit or independent operator trust.

Choose **one** of the following paths, then initialize your participant. If there is no verified publication and compatible package record, stop before requesting funds or signing transactions.

### Compatible package

Open the source-pinned `release.json` from the development page. Choose the `linux-amd64` or `darwin-arm64` entry in `artifacts` and copy its `url`, `sha256` and `binary_sha256` below. Also copy the top-level `descriptor_sha256` and `source_commit`. The GitHub prerelease tag is `zerone-dev-1-` followed by the first 12 characters of that runtime commit; a tag alone is not an integrity check.

```sh
ARCHIVE_URL='REPLACE_WITH_SELECTED_ARTIFACT_URL'
ARCHIVE_SHA256='REPLACE_WITH_SELECTED_ARTIFACT_SHA256'
LOCAL_BINARY_SHA256='REPLACE_WITH_SELECTED_ARTIFACT_BINARY_SHA256'
DESCRIPTOR_SHA256='REPLACE_WITH_VERIFIED_DESCRIPTOR_SHA256'
RUNTIME_COMMIT='REPLACE_WITH_FULL_VERIFIED_RUNTIME_SOURCE_COMMIT'
curl --fail --location --max-time 180 --max-filesize 268435456 \
  "$ARCHIVE_URL" -o zerone-dev-package.tar.gz &&
python3 -I -c 'import hashlib,re,sys; actual=hashlib.sha256(open(sys.argv[1],"rb").read()).hexdigest(); ok=bool(re.fullmatch("[0-9a-f]{64}",sys.argv[2])) and actual==sys.argv[2]; print("Archive SHA-256: PASS" if ok else "Archive SHA-256: FAIL"); sys.exit(0 if ok else 1)' zerone-dev-package.tar.gz "$ARCHIVE_SHA256" &&
mkdir -m 700 zerone-dev-package &&
tar -xzf zerone-dev-package.tar.gz -C zerone-dev-package &&
cd zerone-dev-package
```

Do not continue after a failed check. Extraction requires a **new** directory. The package has no enclosing directory and contains `bin/zeroned`, the three client scripts, `runtime/runtime.py`, `network.json`, `genesis.json`, `SOURCE.json`, `SHA256SUMS` and its README. The macOS package also includes `bin/darwin-acl-check`. It contains no participant, faucet or validator private keys. Leave the files together and keep the Darwin helper beside the binary.

```sh
CLIENT_SCRIPT="$PWD/scripts/shared-claims.py"
RUNTIME_HELPER="$PWD/runtime/runtime.py"
RUNTIME_BINARY="$PWD/bin/zeroned"
DESCRIPTOR_FILE="$PWD/network.json"
```

Use the package's source receipt to inspect what was built. GitHub's automatic “Source code” downloads are repository snapshots, not the prebuilt participant packages.

### Build from source instead

Set `CLIENT_COMMIT` to the full website/client source pin and `RUNTIME_COMMIT` to the descriptor's `source_commit`. Do not substitute a moving branch name. The client checks the binary's embedded runtime commit and SDK version before creating keys.

```sh
CLIENT_COMMIT='REPLACE_WITH_FULL_VERIFIED_CLIENT_SOURCE_COMMIT'
RUNTIME_COMMIT='REPLACE_WITH_FULL_VERIFIED_RUNTIME_SOURCE_COMMIT'
DESCRIPTOR_SHA256='REPLACE_WITH_VERIFIED_DESCRIPTOR_SHA256'
git clone https://github.com/cambridgetcg/zerone-core.git zerone-dev-client
cd zerone-dev-client
git checkout --detach "$CLIENT_COMMIT"
git worktree add --detach ../zerone-dev-runtime "$RUNTIME_COMMIT"
make -C ../zerone-dev-runtime build
curl --fail --max-time 20 --max-filesize 1048576 \
  https://zerone-dev-1.fly.dev/network.json -o network.json
CLIENT_SCRIPT="$PWD/scripts/shared-claims.py"
RUNTIME_HELPER="$PWD/../zerone-dev-runtime/deploy/networks/zerone-dev-1/runtime.py"
RUNTIME_BINARY="$PWD/../zerone-dev-runtime/build/zeroned"
DESCRIPTOR_FILE="$PWD/network.json"
LOCAL_BINARY_SHA256=$(python3 -I -c 'import hashlib,sys; print(hashlib.sha256(open(sys.argv[1],"rb").read()).hexdigest())' "$RUNTIME_BINARY")
```

`make build` also creates `build/darwin-acl-check` on macOS. Keep it beside the binary. Use all three client source files from the client checkout: `scripts/shared-claims.py`, `scripts/claim-workflow.py` and `scripts/local-node.py`. Pinning your own build's hash protects subsequent use from unnoticed drift; it is not an independent reproduction or security audit. Its hash may differ from the released or deployed Linux binary because the platform or build differs.

### Initialize after either path

The public packet has two distinct genesis hashes: `genesis_sha256` authenticates the [exact genesis file](https://zerone-dev-1.fly.dev/genesis.json), while `rpc_genesis_sha256` authenticates the gateway's canonical RPC genesis serialization. The [network descriptor](https://zerone-dev-1.fly.dev/network.json) binds both. Initialization verifies the descriptor's external pin, the binary pin, chain identity and both genesis representations before creating keys.

```sh
PARTICIPANT_HOME="$HOME/zerone-dev-participant"
python3 -I "$CLIENT_SCRIPT" init \
  --home "$PARTICIPANT_HOME" \
  --descriptor "$DESCRIPTOR_FILE" --descriptor-sha256 "$DESCRIPTOR_SHA256" \
  --binary "$RUNTIME_BINARY" --binary-sha256 "$LOCAL_BINARY_SHA256"
```

Save the returned public address. The helper creates one participant signing account, not consensus validator keys. It refuses an existing home and does not replace or import identities. It binds the home to the descriptor, genesis, local binary and client files. Do not use `--allow-loopback-test` for the public network; that flag is for isolated test descriptors.

## 2. Request development funds and register

```sh
python3 -I "$CLIENT_SCRIPT" fund --home "$PARTICIPANT_HOME"
python3 -I "$CLIENT_SCRIPT" onboard --home "$PARTICIPANT_HOME" --type human
```

An agent uses `--type agent`. This is a self-selected account type, not identity verification or proof of an independent controller. Signing and identity key files remain local; neither the website nor the faucet needs their contents.

The faucet offers **1,000 development ZRN once per address**, subject to a **100,000 ZRN lifetime budget**, at most **5 new grants per IP per hour** and **10 globally per hour**. Budget and rate controls limit spending; they do not measure uniqueness or worth. There is no automatic quota reset. A queued or uncertain faucet receipt is not a confirmed transfer. Repeating the funding request checks its existing transaction before any exact-byte replay; it does not promise another grant.

Every signed client transaction declares **2,000,000 gas** and pays **2,000,000 uzrn (2 development ZRN)** as its network fee. The ordinary **non-refundable review fee** is queried separately and can change with the chain parameters and pacing. The reviewer admission check currently requires a balance of at least **100,000,000 uzrn (100 development ZRN)** after the transaction fee is deducted; this is neither a locked bond nor evidence of expertise or independence. A counterclaim has separate collateral settlement rules.

Ordinary claims from the same participant also have a cooldown: the deployed base is **50 blocks**, and network pacing or domain pressure can increase it. Completing a review round does not clear that cooldown. A fresh submission can therefore be refused even after an earlier claim has finished review.

Check the public receipt's `status`, `height`, `code` and transaction hash. Successful CheckTx admission is not inclusion. Require a positive committed height and execution code zero, then read the account or claim state. Preserve an uncertain receipt and use the explicit retry command rather than making a new transaction to guess what happened.

## 3. Submit one bounded claim

State what you checked, which method you used, and where your claim might fail. Publish only evidence you may share. Claim content, reasoning and later revealed reviews are public; a correction does not make earlier material private.

This is an **illustrative submission command**, not an assertion that this example has been submitted, reviewed or accepted:

```sh
python3 -I "$CLIENT_SCRIPT" submit --home "$PARTICIPANT_HOME" \
  --content 'My date parser accepts every valid Gregorian leap day.' \
  --domain mathematics --category empirical --method M-COMPUTATIONAL \
  --reasoning 'Tested 2000-02-29 and 2024-02-29. Century boundaries other than 2000 remain unchecked; the implementation and test inputs must be shared before review.'
```

Use your own real content and accessible references. The example intentionally leaves an important gap: 1900 is not a Gregorian leap year. A few passing cases do not establish a universal claim. The chain stores attributed assertions and review outcomes; it does not supply the outside-world evidence or execute a scientific truth test.

The receipt gives a claim ID after the transaction is committed. Share it with another controller, for example:

```text
https://zerone.ai/development/?claim=YOUR_32_CHARACTER_CLAIM_ID
```

The public reader accepts generated 32-character lowercase hexadecimal claim IDs. It fetches only the requested retained history, not a list of pending work. A claim normally enters verification directly; an endpoint listing only `PENDING` claims would not be a complete review queue.

Once the development page publishes its verification record, you can inspect [the operator-owned prime-polynomial exercise](https://zerone.ai/development/?claim=8db0bccfb17b9a5ad1cf2389b79abd38). It was included at height **432** on **12 September 2026**, with no reviews or challenges at that check. This setup exercise is not a claim of acceptance or independent endorsement. The page makes a new endpoint observation when opened; later records may differ.

## 4. Review from your own participant home

Another participant initializes, funds and registers their **own** home using steps 1–2. Inspect the claim and evidence first:

```sh
CLAIM_ID='REPLACE_WITH_EXACT_CLAIM_ID'
python3 -I "$CLIENT_SCRIPT" history --home "$PARTICIPANT_HOME" --claim "$CLAIM_ID"
python3 -I "$CLIENT_SCRIPT" commit --home "$PARTICIPANT_HOME" \
  --claim "$CLAIM_ID" --vote reject --confidence 800000 \
  --method M-COMPUTATIONAL \
  --reason 'The stated tests do not cover century years; I cannot support the universal claim from these observations.' \
  --scope 'Review of the listed examples and stated limit only; I have not executed the parser.'
python3 -I "$CLIENT_SCRIPT" reveal --home "$PARTICIPANT_HOME" \
  --claim "$CLAIM_ID" --wait-for-phase
```

The vote is `accept`, `reject` or `malformed`. Confidence is the reviewer's declared parts per million, not measured accuracy. Optional `--evidence REF` may repeat. The commitment binds the signer, original chain, round and complete signed review fields. The local client saves the reason, evidence, scope and salt privately before signing; copying someone else's commitment does not let your signer reveal their preimage successfully. Preserve your home until reveal completes.

The default commit and reveal windows are **3,600 blocks each**. At one second per block that is about an hour per phase, but use the actual `commit_deadline` and `reveal_deadline` returned by the round. Stalls and pacing changes alter elapsed time. `--wait-for-phase` waits at most 7,200 seconds; a timeout does not cancel your commitment or authorize a late reveal. Return while the phase is still open. The page shows the endpoint's phase/deadlines; it does not keep your private pending-review journal.

New review policy uses equal account votes and recorded fees for valid on-time reveals, including dissent; it does not turn majority agreement into qualification or universal credibility. Account counts still do not prove separate controllers. Insufficient participation or nondecisive votes can produce an inconclusive outcome. Neither acceptance nor a settlement record establishes a proven claim or independently verified payment.

## 5. Leave a counterexample and follow the retained history

A contradiction names an actual **derived fact ID**, not just a claim ID. Obtain it from the history response after a fact exists. Explain your evidence and its scope:

```sh
FACT_ID='REPLACE_WITH_DERIVED_FACT_ID'
python3 -I "$CLIENT_SCRIPT" challenge --home "$PARTICIPANT_HOME" \
  --fact "$FACT_ID" \
  --content 'The parser accepts 1900-02-29, which is not a valid Gregorian date.' \
  --reason 'Counterexample from the supplied implementation and test input; identify the exact version and attach a reproducible result.' \
  --evidence 'REPLACE_WITH_PUBLIC_REPRODUCTION_REFERENCE'
```

This example also needs real evidence before use. Default challenge collateral is **11,000,000 uzrn (11 development ZRN)**, separate from the 2 ZRN transaction fee; `--stake` changes the explicit collateral amount. A counterclaim gets its own review. Its acceptance retains a literal contradiction relation and may leave the original fact `CONTESTED`; it does not imply an automatic final `DISPROVEN` status.

```sh
python3 -I "$CLIENT_SCRIPT" history --home "$PARTICIPANT_HOME" --claim "$CLAIM_ID"
python3 -I "$CLIENT_SCRIPT" watch --home "$PARTICIPANT_HOME" --claim "$CLAIM_ID"
python3 -I "$CLIENT_SCRIPT" snapshot --home "$PARTICIPANT_HOME"
python3 -I "$CLIENT_SCRIPT" retry --home "$PARTICIPANT_HOME" --tx 'EXACT_SAVED_TRANSACTION_HASH'
```

`retry` queries first and may replay only the saved exact signed bytes. It does not silently create another signature or intent. `snapshot` lists this client's tracked work, not all network activity. The website shows the requested claim, retained rounds and revealed attestations, derived facts with canonical relation metadata, retained status transitions, and directly related claim records. Missing rows remain explicit. The observed chain and height do not constitute a Merkle proof, verified transaction inclusion, a complete archive or independent operator trust. Evidence references are displayed as text; the browser does not fetch them.

## Run your own full node

You can follow the same development chain from its exact genesis and inspect your own application state. Use `runtime/runtime.py` from the verified package, or `deploy/networks/zerone-dev-1/runtime.py` from the exact runtime source checkout; do not use the legacy zerone-1 observer package or the local single-node sandbox to join this chain. Initial full nodes have zero consensus voting power. Neither development funds nor a full node confer block-signing membership; this guide does not open validator admission or prove reviewer independence. Its RPC is loopback-only on port 26657, P2P listens on port 26656, and REST/gRPC are disabled. Choose a host where these ports are free. Preserve the full-node home when stopping or restarting.

The network descriptor identifies the persistent peer and genesis. A full node replaying from genesis reduces reliance on the public query endpoint, while the initial one-operator consensus remains the same trust boundary. The participant client pins its designated gateway; do not edit an existing participant home to redirect it to an arbitrary node.

From the selected package or client checkout above, keep `RUNTIME_HELPER` and `RUNTIME_BINARY` set for that path. Take `GENESIS_SHA256` and `PEER` from the already verified descriptor. `PEER` includes the pinned node ID and address; do not substitute a random discovery endpoint.

```sh
FULL_HOME="$HOME/zerone-dev-full-node"
GENESIS_SHA256='REPLACE_WITH_VERIFIED_GENESIS_FILE_SHA256'
PEER='REPLACE_WITH_VERIFIED_NODE_ID_AT_HOST_PORT'
curl --fail --max-time 30 --max-filesize 8388608 \
  https://zerone-dev-1.fly.dev/genesis.json -o zerone-dev-genesis.json
python3 -I -B "$RUNTIME_HELPER" join \
  --home "$FULL_HOME" --binary "$RUNTIME_BINARY" --source-commit "$RUNTIME_COMMIT" \
  --genesis zerone-dev-genesis.json --genesis-sha256 "$GENESIS_SHA256" \
  --peer "$PEER" --reference-rpc https://zerone-dev-1.fly.dev
python3 -I -B "$RUNTIME_HELPER" run --home "$FULL_HOME" --binary "$RUNTIME_BINARY"
```

Leave `run` in the foreground. In another terminal, with the same checkout and variables:

```sh
python3 -I -B "$RUNTIME_HELPER" status --home "$FULL_HOME" --binary "$RUNTIME_BINARY"
```

Wait for synchronization, zero reported voting power and a matching common block ID/header app hash against the reference node. These status comparisons are node observations, not an independent signature proof. Stop with Ctrl-C and wait for the process to exit cleanly; rerun the same `run` command to resume. Never copy another validator's signing key or use two processes with the same node home. See the [runtime guide](../deploy/networks/zerone-dev-1/README.md) for the exact interfaces and trust limits.
