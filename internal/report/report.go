package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kienlt/lazy-tests/internal/engine"
)

type Files struct {
	JSON string
	MD   string
}

func Write(result engine.Result, dir string, now time.Time) (Files, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "./reports"
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Files{}, fmt.Errorf("create report directory: %w", err)
	}

	ts := now.UTC().Format("20060102-150405")
	base := sanitizeName(result.Scenario.Name)
	jsonPath := filepath.Join(dir, fmt.Sprintf("%s-%s.json", base, ts))
	mdPath := filepath.Join(dir, fmt.Sprintf("%s-%s.md", base, ts))

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return Files{}, fmt.Errorf("marshal report json: %w", err)
	}
	if err := os.WriteFile(jsonPath, append(jsonData, '\n'), 0o644); err != nil {
		return Files{}, fmt.Errorf("write report json: %w", err)
	}

	md := buildMarkdown(result)
	if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
		return Files{}, fmt.Errorf("write report markdown: %w", err)
	}

	return Files{JSON: jsonPath, MD: mdPath}, nil
}

func buildMarkdown(result engine.Result) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "# lazy-tests report: %s\n\n", result.Scenario.Name)
	fmt.Fprintf(b, "- Protocol: `%s`\n", result.Scenario.Protocol)
	fmt.Fprintf(b, "- Target: `%s:%d`\n", result.Scenario.Target.Host, result.Scenario.Target.Port)
	fmt.Fprintf(b, "- Pattern: `%s`\n", result.Scenario.Workload.Pattern)
	fmt.Fprintf(b, "- Started: `%s`\n", result.RunTiming.Start.Format(time.RFC3339))
	fmt.Fprintf(b, "- Ended: `%s`\n", result.RunTiming.End.Format(time.RFC3339))
	fmt.Fprintf(b, "- Duration: `%s`\n\n", result.RunTiming.Duration)

	fmt.Fprintln(b, "## Prometheus")
	fmt.Fprintln(b, "")
	fmt.Fprintf(b, "- enabled: `%t`\n", result.Prometheus.Enabled)
	if result.Prometheus.Enabled {
		fmt.Fprintf(b, "- listen_addr: `%s`\n", result.Prometheus.ListenAddr)
		fmt.Fprintf(b, "- endpoint: `%s`\n", result.Prometheus.Endpoint)
		if result.Prometheus.Message != "" {
			fmt.Fprintf(b, "- note: %s\n", result.Prometheus.Message)
		}
	}
	fmt.Fprintln(b, "")

	fmt.Fprintln(b, "## Metrics")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "| field | value |")
	fmt.Fprintln(b, "| --- | --- |")
	fmt.Fprintf(b, "| attempted | %d |\n", result.Metrics.Attempted)
	fmt.Fprintf(b, "| connected | %d |\n", result.Metrics.Connected)
	fmt.Fprintf(b, "| failed | %d |\n", result.Metrics.Failed)
	fmt.Fprintf(b, "| active_peak | %d |\n", result.Metrics.ActivePeak)
	fmt.Fprintf(b, "| connect_p50_ms | %.2f |\n", result.Metrics.ConnectLatencyMs.P50)
	fmt.Fprintf(b, "| connect_p95_ms | %.2f |\n", result.Metrics.ConnectLatencyMs.P95)
	fmt.Fprintf(b, "| connect_p99_ms | %.2f |\n", result.Metrics.ConnectLatencyMs.P99)
	fmt.Fprintf(b, "| connect_max_ms | %.2f |\n\n", result.Metrics.ConnectLatencyMs.Max)

	fmt.Fprintln(b, "## Errors By Type")
	fmt.Fprintln(b, "")
	fmt.Fprintln(b, "| type | count |")
	fmt.Fprintln(b, "| --- | --- |")
	for _, key := range []string{"timeout", "refused", "reset", "dns", "other"} {
		fmt.Fprintf(b, "| %s | %d |\n", key, result.Metrics.ErrorsByType[key])
	}
	fmt.Fprintln(b, "")

	fmt.Fprintln(b, "## Socket States")
	fmt.Fprintln(b, "")
	if result.Socket.Available {
		fmt.Fprintf(b, "- ESTABLISHED: %d\n", result.Socket.Established)
		fmt.Fprintf(b, "- TIME_WAIT: %d\n\n", result.Socket.TimeWait)
	} else {
		fmt.Fprintf(b, "- unavailable: %s\n\n", result.Socket.Message)
	}

	fmt.Fprintln(b, "## Assertions")
	fmt.Fprintln(b, "")
	fmt.Fprintf(b, "- passed: `%t`\n", result.Assertions.Passed)
	if len(result.Assertions.Failures) == 0 {
		fmt.Fprintln(b, "- failures: none")
	} else {
		for _, failure := range result.Assertions.Failures {
			fmt.Fprintf(b, "- %s\n", failure)
		}
	}

	return b.String()
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "run"
	}
	name = strings.ToLower(name)
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-")
	return replacer.Replace(name)
}
