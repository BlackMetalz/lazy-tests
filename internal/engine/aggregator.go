package engine

import (
	"sort"
	"sync"
)

type event struct {
	attempted   int
	connected   int
	failed      int
	activeDelta int
	latencyMs   float64
	errorType   string
}

type aggregator struct {
	mu sync.Mutex

	attempted int
	connected int
	failed    int
	active    int
	peak      int

	latencies []float64
	errors    map[string]int

	live liveMetrics
}

func newAggregator(live liveMetrics) *aggregator {
	return &aggregator{
		errors: map[string]int{
			"timeout": 0,
			"refused": 0,
			"reset":   0,
			"dns":     0,
			"other":   0,
		},
		live: live,
	}
}

func (a *aggregator) consume(events <-chan event) {
	for ev := range events {
		if a.live != nil {
			a.live.Observe(ev)
		}

		a.mu.Lock()
		a.attempted += ev.attempted
		a.connected += ev.connected
		a.failed += ev.failed

		if ev.activeDelta != 0 {
			a.active += ev.activeDelta
			if a.active > a.peak {
				a.peak = a.active
			}
		}

		if ev.latencyMs > 0 {
			a.latencies = append(a.latencies, ev.latencyMs)
		}

		if ev.errorType != "" {
			a.errors[ev.errorType]++
		}
		a.mu.Unlock()
	}
}

func (a *aggregator) snapshot() Metrics {
	a.mu.Lock()
	defer a.mu.Unlock()

	latency := summarizeLatencies(a.latencies)
	errors := make(map[string]int, len(a.errors))
	for k, v := range a.errors {
		errors[k] = v
	}

	return Metrics{
		Attempted:        a.attempted,
		Connected:        a.connected,
		Failed:           a.failed,
		ActivePeak:       a.peak,
		ConnectLatencyMs: latency,
		ErrorsByType:     errors,
	}
}

func summarizeLatencies(values []float64) LatencySummary {
	if len(values) == 0 {
		return LatencySummary{}
	}

	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)

	return LatencySummary{
		P50: percentile(sorted, 0.50),
		P95: percentile(sorted, 0.95),
		P99: percentile(sorted, 0.99),
		Max: sorted[len(sorted)-1],
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}

	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
