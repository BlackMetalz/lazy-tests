package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kienlt/lazy-tests/internal/engine"
	"github.com/kienlt/lazy-tests/internal/report"
	"github.com/kienlt/lazy-tests/internal/scenario"
)

type App struct {
	stdout io.Writer
	stderr io.Writer
	eng    *engine.Engine
	now    func() time.Time
}

func New(stdout, stderr io.Writer) *App {
	return &App{
		stdout: stdout,
		stderr: stderr,
		eng:    engine.New(),
		now:    time.Now,
	}
}

func (a *App) Run(args []string) int {
	if len(args) == 0 {
		a.printRootUsage()
		return 1
	}

	switch args[0] {
	case "run":
		return a.runCommand(args[1:])
	case "validate":
		return a.validateCommand(args[1:])
	case "list":
		return a.listCommand(args[1:])
	case "help", "-h", "--help":
		a.printRootUsage()
		return 0
	default:
		fmt.Fprintf(a.stderr, "unknown command %q\n\n", args[0])
		a.printRootUsage()
		return 1
	}
}

func (a *App) runCommand(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var file string
	var outDir string
	var duration string
	var connections int
	var connectRate int
	var targetHost string
	var targetPort int
	var dryRun bool
	var unsafePublic bool

	fs.StringVar(&file, "f", "", "scenario yaml file")
	fs.StringVar(&file, "file", "", "scenario yaml file")
	fs.StringVar(&outDir, "out", "", "report output directory")
	fs.StringVar(&duration, "duration", "", "override workload duration (e.g. 30s)")
	fs.IntVar(&connections, "connections", 0, "override workload connections")
	fs.IntVar(&connectRate, "connect-rate-per-sec", 0, "override connect rate per second")
	fs.StringVar(&targetHost, "target-host", "", "override target host")
	fs.IntVar(&targetPort, "target-port", 0, "override target port")
	fs.BoolVar(&dryRun, "dry-run", false, "validate and print execution plan without opening sockets")
	fs.BoolVar(&unsafePublic, "unsafe-public-target", false, "allow public target hosts")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if strings.TrimSpace(file) == "" {
		fmt.Fprintln(a.stderr, "run requires -f <scenario.yaml>")
		return 1
	}

	sc, err := scenario.Load(file)
	if err != nil {
		fmt.Fprintf(a.stderr, "load scenario: %v\n", err)
		return 1
	}

	overrides := scenario.Overrides{
		Host:              targetHost,
		Port:              targetPort,
		Duration:          duration,
		Connections:       connections,
		ConnectRatePerSec: connectRate,
		OutDir:            outDir,
	}
	if err := sc.ApplyOverrides(overrides); err != nil {
		fmt.Fprintf(a.stderr, "apply overrides: %v\n", err)
		return 1
	}

	fmt.Fprintf(
		a.stdout,
		"starting run scenario=%s protocol=%s target=%s:%d pattern=%s duration=%s connections=%d rate=%d\n",
		sc.Name,
		sc.Protocol,
		sc.Target.Host,
		sc.Target.Port,
		sc.Workload.Pattern,
		sc.Workload.Duration,
		sc.Workload.Connections,
		sc.Workload.ConnectRatePerSec,
	)

	result, err := a.eng.Run(context.Background(), *sc, engine.RunOptions{
		DryRun:             dryRun,
		UnsafePublicTarget: unsafePublic,
	})
	if err != nil {
		fmt.Fprintf(a.stderr, "run failed: %v\n", err)
		return 1
	}

	if dryRun {
		a.printDryRunPlan(*sc)
		return 0
	}

	files, err := report.Write(result, sc.Output.ReportDir, a.now())
	if err != nil {
		fmt.Fprintf(a.stderr, "write report: %v\n", err)
		return 1
	}

	a.printRunSummary(result, files)
	if !result.Assertions.Passed {
		return 2
	}
	return 0
}

func (a *App) validateCommand(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(a.stderr)

	var file string
	fs.StringVar(&file, "f", "", "scenario yaml file")
	fs.StringVar(&file, "file", "", "scenario yaml file")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if strings.TrimSpace(file) == "" {
		fmt.Fprintln(a.stderr, "validate requires -f <scenario.yaml>")
		return 1
	}

	sc, err := scenario.Load(file)
	if err != nil {
		fmt.Fprintf(a.stderr, "invalid scenario: %v\n", err)
		return 1
	}

	if err := sc.Validate(); err != nil {
		fmt.Fprintf(a.stderr, "invalid scenario: %v\n", err)
		return 1
	}

	fmt.Fprintf(a.stdout, "scenario %q is valid\n", sc.Name)
	return 0
}

func (a *App) listCommand(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.stderr, "list requires a resource. try: list drivers")
		return 1
	}

	if len(args) > 1 {
		fmt.Fprintln(a.stderr, "list accepts exactly one resource")
		return 1
	}

	switch args[0] {
	case "drivers":
		for _, d := range a.eng.Drivers() {
			fmt.Fprintln(a.stdout, d)
		}
		return 0
	default:
		fmt.Fprintf(a.stderr, "unknown list resource %q\n", args[0])
		return 1
	}
}

func (a *App) printRootUsage() {
	fmt.Fprintln(a.stderr, "lazy-tests v0.3")
	fmt.Fprintln(a.stderr, "usage:")
	fmt.Fprintln(a.stderr, "  lazy-tests run -f scenario.yaml [--target-host HOST] [--target-port PORT] [--out ./reports] [--dry-run] [--unsafe-public-target]")
	fmt.Fprintln(a.stderr, "  lazy-tests validate -f scenario.yaml")
	fmt.Fprintln(a.stderr, "  lazy-tests list drivers")
}

func (a *App) printDryRunPlan(sc scenario.Scenario) {
	fmt.Fprintln(a.stdout, "dry-run execution plan")
	fmt.Fprintf(a.stdout, "  scenario: %s\n", sc.Name)
	fmt.Fprintf(a.stdout, "  target: %s:%d\n", sc.Target.Host, sc.Target.Port)
	fmt.Fprintf(a.stdout, "  protocol: %s\n", sc.Protocol)
	fmt.Fprintf(a.stdout, "  pattern: %s\n", sc.Workload.Pattern)
	fmt.Fprintf(a.stdout, "  connections: %d\n", sc.Workload.Connections)
	fmt.Fprintf(a.stdout, "  connect_rate_per_sec: %d\n", sc.Workload.ConnectRatePerSec)
	fmt.Fprintf(a.stdout, "  duration: %s\n", sc.Workload.Duration)
	if sc.Workload.Pattern == scenario.PatternHoldOpen {
		fmt.Fprintf(a.stdout, "  hold_time: %s\n", sc.Workload.HoldTime)
	}
}

func (a *App) printRunSummary(result engine.Result, files report.Files) {
	fmt.Fprintln(a.stdout, "run summary")
	fmt.Fprintf(a.stdout, "  protocol: %s\n", result.Scenario.Protocol)
	fmt.Fprintf(a.stdout, "  target: %s:%d\n", result.Scenario.Target.Host, result.Scenario.Target.Port)
	fmt.Fprintf(a.stdout, "  attempted: %d\n", result.Metrics.Attempted)
	fmt.Fprintf(a.stdout, "  connected: %d\n", result.Metrics.Connected)
	fmt.Fprintf(a.stdout, "  failed: %d\n", result.Metrics.Failed)
	fmt.Fprintf(a.stdout, "  active_peak: %d\n", result.Metrics.ActivePeak)
	fmt.Fprintf(a.stdout, "  latency p50/p95/p99/max (ms): %.2f / %.2f / %.2f / %.2f\n",
		result.Metrics.ConnectLatencyMs.P50,
		result.Metrics.ConnectLatencyMs.P95,
		result.Metrics.ConnectLatencyMs.P99,
		result.Metrics.ConnectLatencyMs.Max,
	)
	fmt.Fprintf(a.stdout, "  errors: timeout=%d refused=%d reset=%d dns=%d other=%d\n",
		result.Metrics.ErrorsByType["timeout"],
		result.Metrics.ErrorsByType["refused"],
		result.Metrics.ErrorsByType["reset"],
		result.Metrics.ErrorsByType["dns"],
		result.Metrics.ErrorsByType["other"],
	)

	if result.Socket.Available {
		fmt.Fprintf(a.stdout, "  socket states: ESTABLISHED=%d TIME_WAIT=%d\n", result.Socket.Established, result.Socket.TimeWait)
	} else {
		fmt.Fprintf(a.stdout, "  socket states: unavailable (%s)\n", result.Socket.Message)
	}

	if result.Prometheus.Enabled {
		fmt.Fprintf(a.stdout, "  prometheus: %s\n", result.Prometheus.Endpoint)
	}

	fmt.Fprintf(a.stdout, "  assertions: passed=%t\n", result.Assertions.Passed)
	if len(result.Assertions.Failures) > 0 {
		for _, failure := range result.Assertions.Failures {
			fmt.Fprintf(a.stdout, "    - %s\n", failure)
		}
	}

	fmt.Fprintf(a.stdout, "  report json: %s\n", files.JSON)
	fmt.Fprintf(a.stdout, "  report md: %s\n", files.MD)
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr interface{ ExitCode() int }
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}
