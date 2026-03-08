package engine

import (
	"time"

	"github.com/kienlt/lazy-tests/internal/scenario"
)

type RunOptions struct {
	DryRun             bool
	UnsafePublicTarget bool
}

type Result struct {
	Scenario   ScenarioSummary    `json:"scenario"`
	DryRun     bool               `json:"dry_run"`
	Metrics    Metrics            `json:"metrics"`
	RunTiming  RunTiming          `json:"run_timing"`
	Assertions AssertionResult    `json:"assertions"`
	Socket     SocketStates       `json:"socket_states"`
	Prometheus PrometheusEndpoint `json:"prometheus"`
}

type ScenarioSummary struct {
	Name     string            `json:"name"`
	Protocol string            `json:"protocol"`
	Target   scenario.Target   `json:"target"`
	Auth     scenario.Auth     `json:"auth"`
	Workload scenario.Workload `json:"workload"`
	Timeouts scenario.Timeouts `json:"timeouts"`
	Safety   scenario.Safety   `json:"safety"`
	Output   scenario.Output   `json:"output"`
}

type Metrics struct {
	Attempted        int            `json:"attempted"`
	Connected        int            `json:"connected"`
	Failed           int            `json:"failed"`
	ActivePeak       int            `json:"active_peak"`
	ConnectLatencyMs LatencySummary `json:"connect_latency_ms"`
	ErrorsByType     map[string]int `json:"errors_by_type"`
}

type LatencySummary struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
	Max float64 `json:"max"`
}

type RunTiming struct {
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Duration time.Duration `json:"duration"`
}

type AssertionResult struct {
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures"`
}

type SocketStates struct {
	Available   bool   `json:"available"`
	Established int    `json:"established"`
	TimeWait    int    `json:"time_wait"`
	Message     string `json:"message,omitempty"`
}

type PrometheusEndpoint struct {
	Enabled    bool   `json:"enabled"`
	ListenAddr string `json:"listen_addr,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	Message    string `json:"message,omitempty"`
}
