package engine

import (
	"testing"

	"github.com/kienlt/lazy-tests/internal/scenario"
)

func TestEvaluateAssertions_ErrorRateBoundary(t *testing.T) {
	metrics := Metrics{Attempted: 100, Failed: 2}
	cfg := scenario.Assertions{MaxErrorRatePct: 2}

	res := EvaluateAssertions(metrics, cfg)
	if !res.Passed {
		t.Fatalf("expected pass at boundary, got failures: %+v", res.Failures)
	}

	metrics.Failed = 3
	res = EvaluateAssertions(metrics, cfg)
	if res.Passed {
		t.Fatalf("expected fail when above boundary")
	}
}

func TestEvaluateAssertions_P95Boundary(t *testing.T) {
	metrics := Metrics{
		ConnectLatencyMs: LatencySummary{P95: 150},
	}
	cfg := scenario.Assertions{MaxP95ConnectMs: 150}

	res := EvaluateAssertions(metrics, cfg)
	if !res.Passed {
		t.Fatalf("expected pass at p95 boundary")
	}

	metrics.ConnectLatencyMs.P95 = 150.01
	res = EvaluateAssertions(metrics, cfg)
	if res.Passed {
		t.Fatalf("expected fail when p95 above boundary")
	}
}
