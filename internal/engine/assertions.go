package engine

import (
	"fmt"

	"github.com/kienlt/lazy-tests/internal/scenario"
)

func EvaluateAssertions(metrics Metrics, cfg scenario.Assertions) AssertionResult {
	result := AssertionResult{Passed: true}

	if cfg.MaxErrorRatePct >= 0 {
		errorRate := 0.0
		if metrics.Attempted > 0 {
			errorRate = float64(metrics.Failed) / float64(metrics.Attempted) * 100
		}
		if errorRate > cfg.MaxErrorRatePct {
			result.Passed = false
			result.Failures = append(result.Failures,
				fmt.Sprintf("error rate %.2f%% exceeds max_error_rate_pct %.2f%%", errorRate, cfg.MaxErrorRatePct),
			)
		}
	}

	if cfg.MaxP95ConnectMs > 0 {
		if metrics.ConnectLatencyMs.P95 > cfg.MaxP95ConnectMs {
			result.Passed = false
			result.Failures = append(result.Failures,
				fmt.Sprintf("p95 connect latency %.2fms exceeds max_p95_connect_ms %.2fms", metrics.ConnectLatencyMs.P95, cfg.MaxP95ConnectMs),
			)
		}
	}

	return result
}
