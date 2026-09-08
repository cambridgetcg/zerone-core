package deploy_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Jobs run on independent checkouts. A passing SDK job cannot provide packages
// to make test, nor can a Go job populate the dashboard runner's offline cache.
type prerequisiteWorkflow struct {
	Env  map[string]string          `yaml:"env"`
	Jobs map[string]prerequisiteJob `yaml:"jobs"`
}

type prerequisiteJob struct {
	If              string             `yaml:"if"`
	ContinueOnError bool               `yaml:"continue-on-error"`
	Env             map[string]string  `yaml:"env"`
	Steps           []prerequisiteStep `yaml:"steps"`
}

type prerequisiteStep struct {
	Name             string            `yaml:"name"`
	Uses             string            `yaml:"uses"`
	Run              string            `yaml:"run"`
	WorkingDirectory string            `yaml:"working-directory"`
	If               string            `yaml:"if"`
	ContinueOnError  bool              `yaml:"continue-on-error"`
	With             map[string]string `yaml:"with"`
	Env              map[string]string `yaml:"env"`
}

func readPrerequisiteWorkflow(t *testing.T) prerequisiteWorkflow {
	t.Helper()
	data, err := os.ReadFile("../.github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	var workflow prerequisiteWorkflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	return workflow
}

func requiredPrerequisiteSteps(t *testing.T, workflow prerequisiteWorkflow, jobID string, names ...string) []prerequisiteStep {
	t.Helper()
	job, ok := workflow.Jobs[jobID]
	if !ok || job.If != "" || job.ContinueOnError {
		t.Fatalf("%s must be an unconditional, failure-propagating job", jobID)
	}
	var result []prerequisiteStep
	last := -1
	for _, name := range names {
		found := -1
		for i, step := range job.Steps {
			if step.Name == name {
				if found >= 0 {
					t.Fatalf("%s has duplicate prerequisite %q", jobID, name)
				}
				found = i
			}
		}
		if found <= last {
			t.Fatalf("%s prerequisite %q is missing or out of order", jobID, name)
		}
		step := job.Steps[found]
		if step.If != "" || step.ContinueOnError {
			t.Fatalf("%s prerequisite %q must not skip or ignore failure", jobID, name)
		}
		result = append(result, step)
		last = found
	}
	return result
}

func requirePinnedPrerequisiteTool(t *testing.T, workflow prerequisiteWorkflow, jobID, tool, versionKey, envKey string) {
	t.Helper()
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(workflow.Env[envKey]) {
		t.Fatalf("%s must pin an exact toolchain release", envKey)
	}
	job := workflow.Jobs[jobID]
	for _, step := range job.Steps {
		if strings.HasPrefix(step.Uses, "actions/setup-"+tool+"@") {
			if !regexp.MustCompile(`^actions/setup-`+tool+`@[0-9a-f]{40}$`).MatchString(step.Uses) ||
				step.With[versionKey] != "${{ env."+envKey+" }}" || step.If != "" || step.ContinueOnError {
				t.Fatalf("%s must unconditionally install pinned %s before any run steps", jobID, tool)
			}
			return
		}
		if step.Run != "" {
			t.Fatalf("%s runs a command before provisioning %s", jobID, tool)
		}
	}
	t.Fatalf("%s has no setup-%s action", jobID, tool)
}

func TestCICrossLanguagePrerequisites(t *testing.T) {
	workflow := readPrerequisiteWorkflow(t)
	for _, jobID := range []string{"test", "typescript-sdk"} {
		t.Run(jobID, func(t *testing.T) {
			requirePinnedPrerequisiteTool(t, workflow, jobID, "go", "go-version", "GO_VERSION")
			requirePinnedPrerequisiteTool(t, workflow, jobID, "node", "node-version", "NODE_VERSION")
			job := workflow.Jobs[jobID]
			if job.Env["GOTOOLCHAIN"] != "local" || !strings.Contains(job.Env["GOFLAGS"], "-mod=readonly") {
				t.Fatal("cross-language jobs must keep the selected Go and read-only module graph")
			}
		})
	}
	t.Run("signed-transport", func(t *testing.T) {
		checkout := workflow.Jobs["test"].Steps[0]
		if !strings.HasPrefix(checkout.Uses, "actions/checkout@") || checkout.With["fetch-depth"] != "0" || checkout.With["persist-credentials"] != "false" {
			t.Fatal("offline historical regressions need the pinned ancestor object, without persisted credentials")
		}
		steps := requiredPrerequisiteSteps(t, workflow, "test", "Use declared npm release", "Install SDK dependencies", "Verify SDK codecs and rebuild distribution", "Verify committed SDK distribution", "Test")
		if !strings.Contains(steps[0].Run, "npm@11.9.0") || steps[1].Run != "npm ci" || steps[1].WorkingDirectory != "sdk/typescript" {
			t.Fatal("Go signing tests require the lockfile SDK install, including CosmJS dev dependencies")
		}
		if steps[2].WorkingDirectory != "sdk/typescript" || steps[2].Run != "npm run check:generated\nnpm run check:registry\nnpm run build\n" {
			t.Fatal("Go signing tests must use a verified SDK build")
		}
		if steps[3].Run != `test -z "$(git status --porcelain --untracked-files=all -- sdk/typescript/dist)"` || steps[4].Run != "make test" {
			t.Fatal("committed distribution verification and the full Go suite must remain mandatory")
		}
	})
	t.Run("canonical-wire", func(t *testing.T) {
		steps := requiredPrerequisiteSteps(t, workflow, "typescript-sdk", "Download and verify Go modules for canonical-wire test", "Prime canonical-wire test build cache", "Build dashboard")
		if steps[0].WorkingDirectory != "" || !strings.Contains(steps[0].Run, "go mod download") || !strings.Contains(steps[0].Run, "go mod verify") {
			t.Fatal("dashboard runner must download and verify its own root Go module graph")
		}
		if steps[1].Run != "go test -mod=readonly -p 2 ./x/knowledge/keeper -run '^TestReadCanonicalProjectionWire$' -count=1" ||
			steps[1].Env["GOPROXY"] != "off" || steps[1].Env["GOSUMDB"] != "off" {
			t.Fatal("cold compilation must complete offline before the dashboard subprocess timeout begins")
		}
		if steps[2].Run != "npm run build" || steps[2].WorkingDirectory != "dashboard" {
			t.Fatal("full dashboard build/check/test chain must remain mandatory")
		}
	})
}

// Exercise the actual workflow shell, with only network/tool commands stubbed.
// Failed downloads exhaust three attempts; verification never runs on failure,
// and a checksum failure is not converted into a successful prerequisite step.
func TestCICanonicalModuleDownloadFailurePropagation(t *testing.T) {
	workflow := readPrerequisiteWorkflow(t)
	step := requiredPrerequisiteSteps(t, workflow, "typescript-sdk", "Download and verify Go modules for canonical-wire test")[0]
	for _, tc := range []struct {
		name       string
		failures   int
		verifyExit int
		wantOK     bool
		wantCalls  string
	}{
		{"cold-success", 0, 0, true, "mod download\nmod verify\n"},
		{"transient-download", 2, 0, true, "mod download\nsleep 5\nmod download\nsleep 10\nmod download\nmod verify\n"},
		{"download-exhausted", 3, 0, false, "mod download\nsleep 5\nmod download\nsleep 10\nmod download\n"},
		{"verification-failed", 0, 1, false, "mod download\nmod verify\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stub := `#!/bin/sh
printf '%s\n' "$*" >> "$CALLS"
if [ "$*" = 'mod download' ]; then
  count=0
  if [ -f "$COUNTER" ]; then read -r count < "$COUNTER"; fi
  count=$((count + 1))
  printf '%s\n' "$count" > "$COUNTER"
  [ "$count" -gt "$FAILURES" ]
elif [ "$*" = 'mod verify' ]; then
  exit "$VERIFY_EXIT"
else
  exit 99
fi
`
			for name, body := range map[string]string{"go": stub, "sleep": "#!/bin/sh\nprintf 'sleep %s\\n' \"$*\" >> \"$CALLS\"\n"} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			calls := filepath.Join(dir, "calls")
			command := exec.Command("bash", "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", step.Run)
			command.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "CALLS="+calls, "COUNTER="+filepath.Join(dir, "counter"), fmt.Sprintf("FAILURES=%d", tc.failures), fmt.Sprintf("VERIFY_EXIT=%d", tc.verifyExit))
			output, err := command.CombinedOutput()
			if (err == nil) != tc.wantOK {
				t.Fatalf("prerequisite exit = %v, want success %t: %s", err, tc.wantOK, output)
			}
			actual, err := os.ReadFile(calls)
			if err != nil || string(actual) != tc.wantCalls {
				t.Fatalf("calls = %q (error %v), want %q", actual, err, tc.wantCalls)
			}
		})
	}
}
