// constructive-rewards is a deterministic pre-consensus mechanism simulator.
//
//	go run ./tools/constructive-rewards -mode report
//	go run ./tools/constructive-rewards -mode sweep
//	go run ./tools/constructive-rewards -mode shadow
//	go run ./tools/constructive-rewards -mode release # expected to fail closed
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type cliConfig struct {
	mode          string
	format        string
	budget        float64
	alpha         float64
	controllerCap float64
}

func parseFlags(args []string, stderr io.Writer) (cliConfig, error) {
	defaults := DefaultParams()
	flags := flag.NewFlagSet("constructive-rewards", flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := cliConfig{}
	flags.StringVar(&config.mode, "mode", "report", "report, sweep, model, shadow, or release")
	flags.StringVar(&config.format, "format", "text", "text or json")
	flags.Float64Var(&config.budget, "budget", defaults.Budget, "illustrative epoch budget")
	flags.Float64Var(&config.alpha, "alpha", defaults.Alpha, "scarcity-allocation concavity in (0,1]")
	flags.Float64Var(
		&config.controllerCap,
		"controller-cap",
		defaults.ControllerCapShare,
		"maximum cumulative direct share per controller within one cluster lifetime",
	)
	if err := flags.Parse(args); err != nil {
		return cliConfig{}, err
	}
	if flags.NArg() != 0 {
		return cliConfig{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	switch config.mode {
	case "report", "sweep", "model", "shadow", "release":
	default:
		return cliConfig{}, fmt.Errorf("unknown mode %q", config.mode)
	}
	switch config.format {
	case "text", "json":
	default:
		return cliConfig{}, fmt.Errorf("unknown format %q", config.format)
	}
	return config, nil
}

func printSweep(out io.Writer, report SimulationReport) {
	fmt.Fprintln(out, "alpha  cap   aggregate cluster-cap newcomer commons  unfunded   budget-error  pass")
	for _, row := range report.Sweep {
		fmt.Fprintf(
			out,
			"%.2f   %.2f  %.4f    %.4f      %.4f   %.4f   %9.3f  %.3g  %t\n",
			row.Alpha,
			row.ControllerCapShare,
			row.LargestAggregateDirectShare,
			row.MaxClusterCapUtilization,
			row.NewcomerDirectShare,
			row.CommonsShare,
			row.UnfundedDemand,
			row.BudgetError,
			row.Passed,
		)
	}
}

func printGates(out io.Writer, report SimulationReport, class string) {
	for _, gate := range report.ReleaseGates {
		if class != "" && gate.Class != class {
			continue
		}
		status := "PASS"
		if !gate.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(out, "[%s] %-34s %s — %s\n", status, gate.Name, gate.Class, gate.Detail)
	}
}

func printReport(out io.Writer, report SimulationReport) {
	fmt.Fprintln(out, "ZERONE CONSTRUCTIVE-INTELLIGENCE REWARD SIMULATION")
	fmt.Fprintln(out, "pre-consensus model; no chain state read or changed")
	fmt.Fprintln(out)
	fmt.Fprintf(
		out,
		"epoch budget: %.3f | direct: %.3f | commons: %.3f | unallocated: %.3f | unfunded demand: %.3f\n",
		report.IllustrativeEpoch.Budget,
		report.IllustrativeEpoch.DirectTotal,
		report.IllustrativeEpoch.CommonsTotal,
		report.IllustrativeEpoch.Unallocated,
		report.IllustrativeEpoch.UnfundedDemand,
	)
	fmt.Fprintf(
		out,
		"sqrt-stake whale share: one address %.4f -> 100 aliases %.4f (%.2fx)\n",
		report.Attacks.WhaleSingleAddressShare,
		report.Attacks.WhaleHundredAliasShare,
		report.Attacks.WhaleAliasGain,
	)
	fmt.Fprintf(
		out,
		"salami share: naive 1 artifact %.4f -> 100 artifacts %.4f; clustered funded %.3f -> %.3f, direct %.3f -> %.3f\n",
		report.Attacks.NaiveOneArtifactShare,
		report.Attacks.NaiveHundredArtifactShare,
		report.Attacks.PreclusteredOneArtifactFunded,
		report.Attacks.PreclusteredHundredFunded,
		report.Attacks.PreclusteredOneArtifactDirect,
		report.Attacks.PreclusteredHundredDirect,
	)
	fmt.Fprintf(
		out,
		"correlation: 100 reviewers at rho=.2 => n_eff %.6f; 100 linked aliases => %.6f\n",
		report.Attacks.MonocultureEffectiveCount,
		report.Attacks.HundredAliasEffectiveCount,
	)
	fmt.Fprintf(
		out,
		"power illusion: address-level effective count %.3f; controller-level %.3f\n",
		report.Attacks.ObservedAddressEffectivePower,
		report.Attacks.ControllerEffectivePower,
	)
	fmt.Fprintf(out, "weakest multi-surface effective power: %.3f\n", report.WeakestEffectivePower)
	fmt.Fprintln(out)
	printGates(out, report, "")
	fmt.Fprintln(out)
	fmt.Fprintf(
		out,
		"model checks passed: %t | integration ready: %t\n",
		report.ModelChecksPassed,
		report.IntegrationReady,
	)
}

func printShadowReport(out io.Writer, report ShadowScenarioReport) {
	fmt.Fprintln(out, "ZERONE CONSTRUCTIVE-CAPACITY SHADOW LEDGER")
	fmt.Fprintln(out, "local counterfactual; non-authoritative; no network observation or settlement")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "event                 A    Z    L    Q    X    R")
	for _, step := range report.Trace {
		snapshot := step.Snapshot
		fmt.Fprintf(
			out,
			"%-20s %4d %4d %4d %4d %4d %4d\n",
			step.Event,
			snapshot.Accrued,
			snapshot.Funded,
			snapshot.Live,
			snapshot.Quarantined,
			snapshot.Extinguished,
			snapshot.ReplacementUsed,
		)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(
		out,
		"exact checks passed: %t | settlement: %d ZRN | integration ready: %t\n",
		report.Checks.Passed,
		report.SettlementZRN,
		report.IntegrationReady,
	)
}

func run(args []string, stdout, stderr io.Writer) int {
	config, err := parseFlags(args, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "constructive-rewards: %v\n", err)
		return 2
	}
	if config.mode == "shadow" {
		report, err := RunShadowScenario()
		if err != nil {
			fmt.Fprintf(stderr, "constructive-rewards: %v\n", err)
			return 2
		}
		if config.format == "json" {
			encoder := json.NewEncoder(stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(report); err != nil {
				fmt.Fprintf(stderr, "constructive-rewards: encode shadow report: %v\n", err)
				return 2
			}
		} else {
			printShadowReport(stdout, report)
		}
		if !report.Checks.Passed {
			return 1
		}
		return 0
	}
	params := DefaultParams()
	params.Budget = config.budget
	params.Alpha = config.alpha
	params.ControllerCapShare = config.controllerCap
	report, err := RunSimulation(params)
	if err != nil {
		fmt.Fprintf(stderr, "constructive-rewards: %v\n", err)
		return 2
	}
	if config.format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "constructive-rewards: encode report: %v\n", err)
			return 2
		}
	} else {
		switch config.mode {
		case "sweep":
			printSweep(stdout, report)
		case "model":
			printGates(stdout, report, "model")
		case "release":
			printGates(stdout, report, "")
			fmt.Fprintf(
				stdout,
				"\nmodel checks passed: %t | integration ready: %t\n",
				report.ModelChecksPassed,
				report.IntegrationReady,
			)
		default:
			printReport(stdout, report)
		}
	}

	if !report.ModelChecksPassed {
		return 1
	}
	if config.mode == "release" && !report.IntegrationReady {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
