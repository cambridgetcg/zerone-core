# Disposable seed localnet fixture

This is **NON-FINAL local source-pipeline evidence**, never production readiness,
release acceptance, legacy-runtime compatibility, or four-validator safety. It
creates one fresh SDK/Comet validator; custom Zerone staking remains unchanged.
It does not build binaries, fetch dependencies, discover credentials, select a
live profile, or modify production genesis. Default unit tests use fake processes.

## Interface

From any directory, explicitly supply every executable/source input and a runner.
The following is an interface example, not a request to execute it.

```sh
/absolute/zerone/scripts/seed-localnet-fixture.sh \
  --ack-disposable-localnet \
  --authorize-test-admission \
  --authorize-test-feegrant \
  --zeroned /absolute/candidate/zeroned \
  --helper /absolute/candidate/zerone-seed-io \
  --runtime /absolute/candidate/zeroned \
  --source-manifest /absolute/public-source-manifest.json \
  --source-commit EXACT_40_CHARACTER_LOWERCASE_GIT_REVISION \
  --timeout-seconds 600 \
  -- /absolute/pinned/bun /absolute/local-seed-runner.ts
```

All three acknowledgement flags are required. They authorize **only this
fresh disposable fixture's** initialization, one registrar admission, and one
sponsor feegrant. No runner, relative executable path, missing artifact, unsafe
file, or missing acknowledgement means refusal before allocating chain state.
There is no environment opt-in, default binary, implicit runner, production
endpoint, build fallback, or default keyring. `--retain-test-state` is the sole
explicit retention flag; both successful and failed test homes otherwise disappear
following child shutdown and successful public-evidence preservation.

Prerequisites: Bash, Python 3.9+, `lsof`, the already-built candidate `zeroned`,
matching `zerone-seed-io/0.1`, an explicit **native chain runtime** executable
(normally the same candidate `zeroned`, not Bun), and an executable local runner. Paths must be absolute, canonical/symlink-free, regular and
single-linked; artifact files must be owned by this user or root and not writable
by group/others. The source manifest must be canonical public JSON, at most
256 KiB, with at most one final newline. Supply **no secrets** in it or arguments.
Its exact bytes are pinned; the supplied Git revision is not independently
attested as the binary's source. The fixture hashes the binary, helper, runtime,
source manifest and runner executable before execution and rechecks them before
PASS. An interpreter's arguments/imported source are not thereby build-attested;
include their public source references in the supplied manifest.

## Disposable initialization and operator effects

* Allocate fresh marked `0700` directories under the canonical system `/tmp`
  directory, independent of ambient `TMPDIR`. Create separate node/operator,
  claimant-keyring and runner homes, never import an existing key.
* Create real disposable secp256k1 test-keyring keys for validator, registrar,
  sponsor and claimant. Discard **both** key-generation output streams. No
  mnemonic or private-key export is invoked or saved to logs.
* Fund only validator, registrar and sponsor in the **test** bank/auth genesis,
  each with `2,000,000,000 uzrn`. Stake `1,000,000,000 uzrn` through one SDK
  `/cosmos.staking.v1beta1.MsgCreateValidator` gentx. Explicitly derive its
  `account_number` from auth genesis, use sequence zero, and attach a positive
  `2,000,000 uzrn` fee and `2,000,000` gas. These are disposable fixture numbers,
  not recommended production allocations or budgets.
* Install only this test registrar in test claiming-pot params, with a one-entry
  `222,000 uzrn` lifetime commitment and daily admission cap one. Start from empty
  pots/claims. Prove the claimant is absent from bank/auth genesis and query its
  absent Cosmos account and zero bank balance after startup. No Zerone DID
  registration is performed or required.
* Allocate three distinct ports by simultaneously binding three loopback sockets;
  recheck availability immediately before launch. Configure only IPv4 loopback
  P2P, RPC and gRPC. Disable seeds, persistent peers, PEX, state sync, RPC unsafe
  mode/pprof, REST, gRPC-web, oracle and telemetry/Prometheus. Require `lsof` on
  the exact child PID to show exactly those three loopback listeners.
* Registrar admission creates the address-bound pot, **not** the Cosmos account.
  Query that the account is still absent. Sponsor signs a finite
  `10,000,000 uzrn`, expiring BasicAllowance wrapped in AllowedMsgAllowance with
  **only** `/zerone.claiming_pot.v1.MsgClaim`. Both operator transactions are
  independently compared byte-for-byte with the native helper's construction-only
  message results. Wait for actual inclusion, never rebroadcast automatically.
* Query allowance shape/expiry and account materialization: found, sequence zero,
  no registered public key, still zero bank balance. Wait past the pot's one-block
  vesting end before calling the runner.

The pinned consensus gas price is **1 uzrn per gas**, not a one-uzrn total fee.
The unmapped native Claim floor is **22,222 gas**, not measured total gas. The
runner must simulate its exact granter-bearing Claim, measure total gas and stay
within the fixture's `2,000,000` gas/fee ceilings. Full outstanding sponsor grant
exposure is `10,000,000 uzrn`; the separate setup budget is `4,000,000 uzrn`.
The default grant expires 720 seconds after contract creation (configured total
timeout plus 120 seconds); policy expiry is 30 seconds earlier. No production
economics or consensus rule is changed.

## Runner contract

The runner is supplied after the literal `--`; no shell evaluation is used. It
must remain foreground, reap its own helper/provider subprocesses on termination,
not daemonize, use only fixture paths/endpoints, and never touch outside keys,
chains or networks. This is a cooperative test runner, **not an OS sandbox for
hostile code**. The fixture signals only direct child handles it actually launched;
it deliberately does not discover or kill arbitrary descendants by process name,
argv, process group, `pkill`, or process-list matching.

The environment is rebuilt, not inherited. It contains only an owned HOME,
TMPDIR and XDG directories, fixed system PATH, C locale, NO_COLOR, and:

| Environment variable | Meaning |
|---|---|
| `SEED_FIXTURE_JSON` | Owned canonical `zerone.seed-local-runner/1` manifest |
| `SEED_RESULT_JSON` | Exact initially absent owned runner receipt path |
| `SEED_TEST_WORK_DIR` | Private runner work directory |
| `SEED_NON_FINAL` | Literal `1` |

`fixture.json` contains RPC/gRPC, unique `seed-local-*` chain reference, genesis,
CLI/helper/runtime paths and digests, source manifest/revision, claimant and sponsor
addresses, exact profile/policy/node JSON paths, helper trust file, and the explicit
`{home, backend:"test", key_name:"claimant"}` keyring handle. It contains no secret
bytes. Use the supplied profile and policy unchanged; native helper calls are
`HELPER COMMAND --trust-file PATH --disposable-test`, canonical request on stdin.
The helper enforces configured-node typed ABCI reads, exact trusted artifacts,
local transport, and **latest-only simulation with empty mempool and unchanged
latest height around the call**. Its `grpc_address` is explicit configuration,
not a fallback transport. The fixture's own direct ABCI allowlist contains only
bank Balance and auth ModuleAccountByName; Claim/pot/allowance/account inspection
uses the native helper's bounded read allowlist. POST JSON-RPC encodes ABCI `data`
as plain hexadecimal (`HexBytes`, no `0x`) and transaction lookup `hash` as standard
base64 (`[]byte`). URI/GET-style `0x` arguments are not the POST JSON representation.
The first real-daemon run exposed this fixture framing defect; the regression is
in `test_json_rpc_binary_encodings_not_uri_encodings`.

The runner owns actual Wallet record/key preparation, inspect/prepare, exact
simulation, atomic durable reservation, separately invoked external sign/verify,
single submit and confirmed reconciliation. A transient pre-sign simulation
failure is not permission to repeat signing. Unknown signing/submission outcomes
remain sticky; the fixture never resubmits or signs a replay automatically.

After reconciliation and an actual **CLI replay refusal**, write a small (at most
4 KiB), `0600`, regular, single-linked public JSON file at `SEED_RESULT_JSON`. Exactly
these four members are allowed:

```json
{"claimed_tx_hash":"UPPERCASE_64_HEX_TRANSACTION_HASH","claimed_address":"THE_FIXTURE_CLAIMANT_ADDRESS","actual_amount":"222000","replay_refused":true}
```

The displayed placeholder values must be replaced by actual public test results.
A callback exit status zero alone is insufficient. The fixture independently
requires the supplied hash to match queried transaction bytes and canonical block
inclusion, exactly one correctly addressed native Claim, the exact sponsor and no
payer, valid gas/fee floor and ceiling, a matching native claim at that transaction
height, a depleted `222000` pot, claimant sequence one and bank balance `222000`,
exact allowance consumption, exact sponsor debit and positive measured gas.
The native helper performs native claim/pot queries separately from the runner.
The fixture verifies full issuance for this deliberately ample-headroom genesis;
it is **not** a supply-clipping experiment. Native account/state observations are
through the configured local node, not portable ICS23 proofs or a quorum proof.
`replay_refused` remains explicitly **runner-reported CLI evidence**, distinct
from independent native post-state verification. Native replay can be tested by a
separately scoped runner; this fixture never signs a second attempt.

## Failure, evidence and cleanup

The chain/runner deadline defaults to 600 seconds, configurable from 30 to 1800.
Individual commands are bounded to 30 seconds, helper requests to 10 seconds,
RPC requests to 3 seconds, and height/transaction waits to 60 seconds within the
overall deadline. Each subprocess output stream is capped at 256 KiB in memory;
excess output fails closed. Output buffers are never printed or copied to public
artifacts. Key-generation and genesis command streams are discarded entirely.
The raw daemon/runner logs are intentionally **not retained**, even on failure;
public diagnostics expose static failing stage, bounded error code, exit code and
output byte counts. Detailed private debugging requires a separately authorized
change, not an automatic log export. No signed transaction bytes appear in the
public evidence bundle.

INT/TERM become failure statuses 130/143. Cleanup uses exact `Popen` handles,
TERM then a finite grace before KILL, and bounded wait/reap; it never signals an
already-reaped PID. The ordinary daemon/command grace remains **3 seconds**.
Only the explicitly launched runner gets **120 seconds**: its active compiled
CLI has a 90-second ceiling, with 30 seconds of teardown margin. The supplied
runner configures native/provider calls at 10 seconds; its longest CLI chain,
reserve-sign, has five sequential calls (inspect, simulate, record seal, sign,
verify), at most 50 seconds plus local work. Its initial record creation likewise
has five sequential 10-second provider calls. TERM sets a stop flag; it does not
cancel the CLI/helper. The next child launch is refused after the active call is
reaped. An outer supervisor must budget at least the overall deadline plus this
runner grace, ordinary daemon grace and pipe/reap margin (600 + 120 + 30 seconds
for the standard invocation). A supplied runner must honor these budgets; this
is not hostile-descendant containment.

A forcibly killed or signal-terminated runner is **cleanup uncertain**, even if
its direct PID is reaped and all three ports are closed. This is sticky across
repeated cleanup calls: preserve owned test state, expose a cleanup error, and
never delete it or report PASS without cooperative descendant-reaping evidence.
Port closure and direct-child reaping are separately reported observations, not
proxies for descendant cleanup. Additional teardown time is bounded per live child.
Check all three ports after shutdown without killing a foreign process that might
have taken one. Cleanup errors turn an otherwise successful run into failure,
while an original failure/status is preserved. SIGKILL of the fixture, machine
loss or a hostile runner are outside trap guarantees.

The fixture writes bounded public `observations.json` with UNFINALIZED cleanup
status **before** deleting any state, then final `evidence.json` in a separately
fresh marked `seed-local-evidence-*` directory. Final stdout is one small JSON
object locating that evidence and any explicitly retained state. Removal requires
matching directory device/inode, ownership/mode and exact marker token, and uses
symlink-resistant directory deletion. Never remove an unverified directory. If
ownership verification, evidence preservation or child cleanup fails, fail and
leave the state untouched with an explicit cleanup error; this is not successful
retention or successful cleanup. Public evidence survives normal deletion.

## Local unit validation and reuse

```sh
/absolute/zerone/scripts/seed-localnet-fixture_test.sh
```

The tests extract this script's embedded Python without running its main function,
exercise real fake child processes and temporary loopback socket availability,
and run one shell-entry fake initialization failure. No real daemon, keyring,
Go build, or external network is used. Tests cover explicit opt-in, source/path
checks, bounded stdin/output/timeouts, foreign-process survival, exact child reap,
marker/inode/symlink-safe cleanup, failure preservation, evidence-before-delete,
closed runner receipt, and independently rejected false post-state/fee/credit
reports. Passing these tests does **not** claim the real chain journey passed.
The separate orchestrator must invoke one actual fixture after all candidate
binaries and the concrete durable runner are ready.

Source reuse: `scripts/local-consensus-rehearsal.sh` supplies initialization,
loopback configuration and gentx/query patterns, **not** its ps/argv-based cleanup
or old fee-less gentx assumptions. The positive fee/account-number correction is
grounded in `app/ante_zerone.go` and SDK gentx signing. Native semantics are in
`x/claiming_pot/types/types.go`, `keeper/msg_server.go`, `client/cli/tx.go`,
`proto/zerone/claiming_pot/v1/{tx,state,query}.proto`, and pinned
`cosmossdk.io/x/feegrant@v0.2.0/{client/cli/tx.go,keeper/keeper.go}`.
Helper/profile contracts come from `tools/zerone-seed-io/{main,files,native,node}.go`
and the separately developed AgentTool `packages/wallet-zerone/src/bootstrap/`
and `bin/zerone-seed/`. Those files are read-only dependencies of this fixture.
