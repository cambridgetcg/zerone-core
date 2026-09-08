package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Extract only named definitions: never source the rehearsal's main routine,
// initialize a daemon, touch an existing home, or signal an actual process.
func rehearsalFunction(t *testing.T, name string) string {
	t.Helper()
	bz, err := os.ReadFile(filepath.Join("..", "..", "scripts", "local-consensus-rehearsal.sh"))
	require.NoError(t, err)
	match := regexp.MustCompile(`(?ms)^` + regexp.QuoteMeta(name) + `\(\) \{\n.*?^\}`).FindString(string(bz))
	require.NotEmpty(t, match, name)
	return match + "\n"
}

func runLifecycleFixture(t *testing.T, body string) (string, error) {
	t.Helper()
	command := exec.Command("bash", "-c", "set -eu\n"+body)
	command.Env = append(os.Environ(), "TMPDIR="+t.TempDir())
	output, err := command.CombinedOutput()
	return string(output), err
}

func TestObserverCleanupIncludesFifthOwnedProcess(t *testing.T) {
	body := rehearsalFunction(t, "stop_all") + `
stop_node() { printf 'stop-%s\n' "$1"; }
stop_all
`
	output, err := runLifecycleFixture(t, body)
	require.NoError(t, err, output)
	require.Equal(t, "stop-0\nstop-1\nstop-2\nstop-3\nstop-4\n", output)
}

func TestObserverProcessOwnershipAndGracefulStop(t *testing.T) {
	definitions := rehearsalFunction(t, "is_owned_pid") + rehearsalFunction(t, "stop_node")
	setup := `
declare -a NODE_PID NODE_HOME
NODE_PID[4]=4242
NODE_HOME[4]=/owned/node4
BINARY=/owned/zeroned
live=1
fixture_mode=owned
kill() {
  if [ "$1" = -0 ]; then [ "$live" -eq 1 ]; return; fi
  printf 'signal-%s\n' "$1"
  if [ "$fixture_mode" != stuck ]; then live=0; fi
}
ps() {
  if [ "$fixture_mode" = foreign ]; then printf '/other/zeroned start --home /other/node4\n';
  elif [ "$fixture_mode" = prefix ]; then printf '/owned/zeroned start --home /owned/node4-foreign\n';
  else printf '/owned/zeroned start --home /owned/node4\n'; fi
}
sleep() { :; }
wait() { :; }
die() { printf 'refused\n'; exit 9; }
`
	for _, tc := range []struct {
		name, script string
		fails        bool
		signals      string
	}{
		{"owned graceful", `stop_node 4 graceful; [ -z "${NODE_PID[4]}" ]`, false, "signal--TERM\n"},
		{"foreign refusal", `fixture_mode=foreign; stop_node 4`, true, "refused\n"},
		{"home prefix refusal", `fixture_mode=prefix; stop_node 4`, true, "refused\n"},
		{"graceful no escalation", `fixture_mode=stuck; stop_node 4 graceful`, true, "signal--TERM\nrefused\n"},
		{"fifth absent", `NODE_PID[4]=""; stop_node 4`, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := runLifecycleFixture(t, definitions+setup+tc.script)
			if tc.fails {
				require.Error(t, err)
			} else {
				require.NoError(t, err, output)
			}
			require.Equal(t, tc.signals, output)
		})
	}
}

func TestObserverGracefulStopRequiresSuccessfulChildExit(t *testing.T) {
	definitions := rehearsalFunction(t, "is_owned_pid") + rehearsalFunction(t, "stop_node")
	setup := `
declare -a NODE_PID NODE_HOME
NODE_PID[4]=4242
NODE_HOME[4]=/owned/node4
BINARY=/owned/zeroned
fixture_live=1
fixture_term_status=0
fixture_wait_status=0
kill() {
  if [ "$1" = -0 ]; then [ "$fixture_live" -eq 1 ]; return; fi
  printf 'signal-%s\n' "$1"
  fixture_live=0
  return "$fixture_term_status"
}
ps() { printf '/owned/zeroned start --home /owned/node4\n'; }
sleep() { :; }
wait() { printf 'wait-%s\n' "$1"; return "$fixture_wait_status"; }
die() { printf 'refused: %s\n' "$*"; exit 9; }
trap 'printf "pid=%s\n" "${NODE_PID[4]:-}"' EXIT
`
	for _, tc := range []struct {
		name, script, mode, trace, refusal, pid string
	}{
		{"successful child", "", "graceful", "signal--TERM\nwait-4242\n", "", ""},
		{"missing PID", `NODE_PID[4]=""`, "graceful", "", "has no recorded PID for graceful stop", ""},
		{"already dead", `fixture_live=0`, "graceful", "", "exited before graceful stop", "4242"},
		{"TERM delivery failed", `fixture_term_status=1`, "graceful", "signal--TERM\n", "could not TERM owned node", "4242"},
		{"child failed", `fixture_wait_status=1`, "graceful", "signal--TERM\nwait-4242\n", "exited with status 1; no graceful restart-pass claim", "4242"},
		{"child killed", `fixture_wait_status=137`, "graceful", "signal--TERM\nwait-4242\n", "exited with status 137; no graceful restart-pass claim", "4242"},
		{"unhandled TERM", `fixture_wait_status=143`, "graceful", "signal--TERM\nwait-4242\n", "exited with status 143; no graceful restart-pass claim", "4242"},
		{"not a child", `fixture_wait_status=127`, "graceful", "signal--TERM\nwait-4242\n", "exited with status 127; no graceful restart-pass claim", "4242"},
		{"cleanup already dead", `fixture_live=0`, "cleanup", "", "", ""},
		{"cleanup failed child", `fixture_wait_status=137`, "cleanup", "signal--TERM\nwait-4242\n", "", ""},
		{"cleanup failed signal", `fixture_term_status=1`, "cleanup", "signal--TERM\nwait-4242\n", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := runLifecycleFixture(t, definitions+setup+tc.script+"\nstop_node 4 "+tc.mode+"\n")
			if tc.refusal == "" {
				require.NoError(t, err, output)
				require.Equal(t, tc.trace+"pid="+tc.pid+"\n", output)
			} else {
				require.EqualError(t, err, "exit status 9", output)
				require.True(t, strings.HasPrefix(output, tc.trace+"refused: "), output)
				require.Contains(t, output, tc.refusal)
				require.True(t, strings.HasSuffix(output, "pid="+tc.pid+"\n"), output)
			}
		})
	}
}

func TestObserverPortAndPeerBoundaries(t *testing.T) {
	body := rehearsalFunction(t, "allocate_ports") + rehearsalFunction(t, "persistent_peers") + `
declare -a P2P_PORT RPC_PORT NODE_ID
port_is_free() { return 0; }
die() { exit 9; }
allocate_ports
[ "${RPC_PORT[4]}" -eq "$((BASE_PORT + 9))" ]
[ "${P2P_PORT[4]}" -eq "$((BASE_PORT + 8))" ]
NODE_ID=(v0 v1 v2 v3 observer)
persistent_peers 4
`
	output, err := runLifecycleFixture(t, body)
	require.NoError(t, err, output)
	require.Len(t, strings.Split(output, ","), 4)
	require.NotContains(t, output, "observer")
}

func TestObserverConfigurationDisablesStateSync(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(home, "config"), 0700))
	config := strings.Replace(observerTestConfig, "enable = false", "enable = true", 1)
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "config.toml"), []byte(config), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(home, "config", "app.toml"), []byte(`minimum-gas-prices = ""`), 0600))
	body := rehearsalFunction(t, "sed_in_place") + rehearsalFunction(t, "configure_node") +
		"declare -a NODE_HOME\nNODE_HOME[4]=" + "'" + home + "'\nconfigure_node 4\n"
	output, err := runLifecycleFixture(t, body)
	require.NoError(t, err, output)
	bz, err := os.ReadFile(filepath.Join(home, "config", "config.toml"))
	require.NoError(t, err)
	require.Contains(t, string(bz), "[statesync]\nenable = false")
}

func TestObserverRefusesEnvironmentOverridesWithoutLeakingValues(t *testing.T) {
	body := rehearsalFunction(t, "assert_local_environment") + "export ZERONED_STATESYNC_ENABLE=sentinel-private-value\nassert_local_environment\n"
	output, err := runLifecycleFixture(t, body)
	require.Error(t, err)
	require.Contains(t, output, "environment overrides are refused")
	require.NotContains(t, output, "sentinel-private-value")
}

func TestObserverHarnessPreservesCustodyAndSchedulerOrdering(t *testing.T) {
	bz, err := os.ReadFile(filepath.Join("..", "..", "scripts", "local-consensus-rehearsal.sh"))
	require.NoError(t, err)
	script := string(bz)
	full := rehearsalFunction(t, "verify_integrated_flow")
	require.Contains(t, full, "verify_scheduler_flow\n  verify_phase four-up")
	require.Contains(t, full, "verify_message_flow\n\n  verify_observer_flow\n  verify_offline_custom_staking_census")
	require.NotContains(t, full, "OBSERVER_ONLY")
	require.Contains(t, script, "mkdir -p \"${COORDINATOR}/config/gentx\"\nfor index in 0 1 2 3; do")
	observer := rehearsalFunction(t, "verify_observer_flow")
	require.NotContains(t, observer, "cp -R")
	require.NotContains(t, observer, "keys add")
	require.NotContains(t, observer, "gentx")
	require.Equal(t, 1, strings.Count(observer, "cp "))
	require.Equal(t, 1, strings.Count(observer, "init observer"))
	require.Equal(t, 2, strings.Count(observer, "start_node 4"))
	require.Contains(t, observer, "stop_node 4 graceful")
	stop := strings.Index(observer, "stop_node 4 graceful")
	require.NotContains(t, observer[stop:], " cp ")
	require.NotContains(t, observer[stop:], " init ")
	require.Contains(t, rehearsalFunction(t, "start_node"), "${GENESIS_SHA}")
	require.Contains(t, rehearsalFunction(t, "start_node"), "${BINARY_SHA}")
}

func TestObserverOnlySelectionUsesSameSetupAndSourceChecks(t *testing.T) {
	bz, err := os.ReadFile(filepath.Join("..", "..", "scripts", "local-consensus-rehearsal.sh"))
	require.NoError(t, err)
	script := string(bz)
	require.Contains(t, script, "\nOBSERVER_ONLY=0\n")
	require.Contains(t, script, "\nparse_args \"$@\"\n")
	start := strings.Index(script, "RUN_ROOT=\"$(mktemp")
	end := strings.Index(script, "\nverify_integrated_flow()")
	require.Greater(t, end, start)
	setup := script[start:end]
	require.NotContains(t, setup, "OBSERVER_ONLY", "diagnostic must share genesis, build, config and startup")
	require.Contains(t, setup, "--offline")
	require.Contains(t, setup, "--account-number \"${VALIDATOR_ACCOUNT_NUMBER}\"")
	require.Contains(t, setup, "--sequence 0")
	require.Contains(t, setup, "--fees \"${TX_FEE}${DENOM}\"")
	require.Contains(t, setup, "-scheduler-fixture-root")
	require.Contains(t, setup, ".app_state.message_schedule.params.accept_new_schedules == false")
	require.Contains(t, setup, `snapshot_candidate_sources "${RUN_ROOT}/reports/candidate-source-sha256.json"`)
	require.Contains(t, setup, `cmp "${RUN_ROOT}/reports/candidate-source-sha256.json" "${RUN_ROOT}/reports/candidate-source-after-build-sha256.json" || die`)
	final := script[strings.LastIndex(script, "\nverify_selected_flow\n"):]
	require.Contains(t, final, "checkout HEAD changed during the rehearsal")
	require.Contains(t, final, `cmp "${RUN_ROOT}/reports/candidate-source-sha256.json" "${RUN_ROOT}/reports/candidate-source-final-sha256.json" || die`)
	require.Less(t, strings.Index(final, "candidate-source-final-sha256.json"), strings.Index(final, "SUCCESS=1\nreport_success"))
	require.Contains(t, final, "NON-FINAL dirty development run")
}

func TestObserverOnlyDiagnosticSelectionAndOmissionLabels(t *testing.T) {
	definitions := rehearsalFunction(t, "parse_args") + rehearsalFunction(t, "verify_selected_flow") +
		rehearsalFunction(t, "observer_only_scope") + rehearsalFunction(t, "report_success")
	setup := `
OBSERVER_ONLY=0
ALLOW_DIRTY=0
KEEP=0
info() { :; }
die() { printf 'refused\n'; exit 9; }
usage() { :; }
verify_integrated_flow() { printf 'full-suite\n'; }
wait_for_height() { [ "$2" -eq 13 ]; printf 'history-%s-%s\n' "$1" "$2"; }
verify_phase() { printf 'phase-%s\n' "$*"; }
verify_observer_flow() { printf 'observer-replay-restart\n'; }
`
	for _, tc := range []struct{ name, args string }{
		{"default", ""}, {"dirty default", "--allow-dirty --keep"},
		{"targeted", "--observer-only"}, {"dirty targeted", "--allow-dirty --observer-only --keep"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := runLifecycleFixture(t, definitions+setup+"\nparse_args "+tc.args+"\nverify_selected_flow\nreport_success\n")
			require.NoError(t, err, output)
			if strings.Contains(tc.args, "--observer-only") {
				require.NotContains(t, output, "full-suite")
				require.NotContains(t, output, "PASS isolated consensus rehearsal")
				require.Contains(t, output, "history-0-13\nhistory-1-13\nhistory-2-13\nhistory-3-13\nphase-observer-only-initial-quorum 0 1 2 3\nobserver-replay-restart\n")
				require.Contains(t, output, "PASS targeted observer replay/restart diagnostic")
				for _, omission := range []string{"NOT RUN:", "scheduler timing/TERM/KILL/bank-zero", "75% progress/50% halt and validator recovery", "MsgSend/replay", "offline census", "Full integrated suite not established", "nonmembership limitation unchanged"} {
					require.Contains(t, output, omission)
				}
			} else {
				require.Equal(t, "full-suite\n\nPASS isolated consensus rehearsal\n", output)
			}
		})
	}
	output, err := runLifecycleFixture(t, definitions+setup+"\nparse_args --observer-onli\nverify_selected_flow\nreport_success\n")
	require.Error(t, err)
	require.Equal(t, "refused\n", output)
	for _, phase := range []string{"verify_integrated_flow", "verify_phase", "verify_observer_flow"} {
		args := "--observer-only"
		if phase == "verify_integrated_flow" {
			args = ""
		}
		output, err = runLifecycleFixture(t, definitions+setup+"\n"+phase+"() { return 9; }\nparse_args "+args+"\nverify_selected_flow\nreport_success\n")
		require.Error(t, err, phase)
		require.NotContains(t, output, "PASS", "failed proof/phase must never translate into success")
	}
}

func TestObserverDefaultSelectionExecutesAllPriorPhases(t *testing.T) {
	// Execute the actual full-suite body with pure shell mocks. No daemon,
	// network, real process signals or user-local database is reached.
	definitions := rehearsalFunction(t, "verify_integrated_flow") + rehearsalFunction(t, "verify_selected_flow")
	setup := `
OBSERVER_ONLY=0
CHAIN_ID=fixture
RUN_ROOT="$TMPDIR"
mkdir "$RUN_ROOT/reports"
VERIFY_BINARY=fixture_verify
info() { :; }
ok() { :; }
die() { printf 'refused\n'; exit 9; }
verify_scheduler_flow() { printf 'scheduler-TERM-KILL-proofs\n'; }
verify_phase() { printf 'phase-%s\n' "$*"; }
stop_node() { printf 'stop-%s\n' "$1"; }
start_node() { printf 'start-%s\n' "$1"; }
wait_for_advance() { printf 'advance-%s\n' "$*"; }
wait_for_height() { printf 'height-%s-%s\n' "$1" "$2"; }
sleep() { :; }
current_height() { printf '20'; }
rpc_get() { printf '{"result":{"block_id":{"hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"block":{"header":{"app_hash":"fixture-root"}}}}'; }
rpc_url() { printf 'mock-%s' "$1"; }
fixture_verify() { [ "$*" = '-rpcs mock-0,mock-1 -height 19 -expect-chain-id fixture -expect-validators 4 -expect-equal-power' ]; printf 'frozen-quorum-proof\n' >&2; }
is_owned_pid() { printf 'owned-%s\n' "$1"; }
verify_message_flow() { printf 'MsgSend-proof-replay\n'; }
verify_observer_flow() { printf 'observer-replay-restart\n'; }
verify_offline_custom_staking_census() { printf 'offline-census\n'; }
verify_selected_flow
`
	output, err := runLifecycleFixture(t, definitions+setup)
	require.NoError(t, err, output)
	require.Equal(t, "scheduler-TERM-KILL-proofs\nphase-four-up 0 1 2 3\nstop-3\nadvance-3 one-validator-down 0 1 2\nphase-one-down 0 1 2\nstop-2\nfrozen-quorum-proof\nowned-0\nowned-1\nstart-2\nadvance-3 three-up-recovery 0 1 2\nphase-three-up-recovery 0 1 2\nstart-3\nheight-0-23\nheight-1-23\nheight-2-23\nheight-3-23\nphase-four-up-recovered 0 1 2 3\nMsgSend-proof-replay\nobserver-replay-restart\noffline-census\n", output)
	failed := strings.Replace(setup, `verify_scheduler_flow() { printf 'scheduler-TERM-KILL-proofs\n'; }`,
		`verify_scheduler_flow() { printf 'upstream absence proof failed\n'; return 9; }`, 1)
	output, err = runLifecycleFixture(t, definitions+failed+"\nprintf 'PASS must not execute\\n'\n")
	require.Error(t, err)
	require.Equal(t, "upstream absence proof failed\n", output, "no fallback, later phase or PASS after default proof failure")
}
