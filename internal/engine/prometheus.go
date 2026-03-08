package engine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type liveMetrics interface {
	Observe(event)
	Endpoint() string
	ListenAddr() string
	Close(context.Context) error
}

type prometheusRuntime struct {
	mu sync.Mutex

	endpoint   string
	listenAddr string

	active int
	peak   int

	attempted prometheus.Counter
	connected prometheus.Counter
	failed    prometheus.Counter
	activeNow prometheus.Gauge
	activeMax prometheus.Gauge
	latencyMs prometheus.Histogram
	errors    *prometheus.CounterVec

	server *http.Server
	ln     net.Listener
}

func newPrometheusRuntime(listenAddr string) (*prometheusRuntime, error) {
	registry := prometheus.NewRegistry()

	r := &prometheusRuntime{
		attempted: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "lazy_tests_attempted_total",
			Help: "Total connection attempts",
		}),
		connected: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "lazy_tests_connected_total",
			Help: "Total successful connections",
		}),
		failed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "lazy_tests_failed_total",
			Help: "Total failed connections",
		}),
		activeNow: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "lazy_tests_active_connections",
			Help: "Current active connections",
		}),
		activeMax: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "lazy_tests_active_peak",
			Help: "Peak active connections during this run",
		}),
		latencyMs: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "lazy_tests_connect_latency_ms",
			Help:    "Connect latency in milliseconds",
			Buckets: []float64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000},
		}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "lazy_tests_errors_total",
			Help: "Errors by classified type",
		}, []string{"type"}),
	}

	if err := registry.Register(r.attempted); err != nil {
		return nil, err
	}
	if err := registry.Register(r.connected); err != nil {
		return nil, err
	}
	if err := registry.Register(r.failed); err != nil {
		return nil, err
	}
	if err := registry.Register(r.activeNow); err != nil {
		return nil, err
	}
	if err := registry.Register(r.activeMax); err != nil {
		return nil, err
	}
	if err := registry.Register(r.latencyMs); err != nil {
		return nil, err
	}
	if err := registry.Register(r.errors); err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("start prometheus listener on %s: %w", listenAddr, err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	r.server = server
	r.ln = ln
	r.listenAddr = ln.Addr().String()
	r.endpoint = fmt.Sprintf("http://%s/metrics", r.listenAddr)

	go func() {
		_ = server.Serve(ln)
	}()

	return r, nil
}

func (r *prometheusRuntime) Observe(ev event) {
	if ev.attempted > 0 {
		r.attempted.Add(float64(ev.attempted))
	}
	if ev.connected > 0 {
		r.connected.Add(float64(ev.connected))
	}
	if ev.failed > 0 {
		r.failed.Add(float64(ev.failed))
	}
	if ev.latencyMs > 0 {
		r.latencyMs.Observe(ev.latencyMs)
	}
	if ev.errorType != "" {
		r.errors.WithLabelValues(ev.errorType).Inc()
	}

	if ev.activeDelta != 0 {
		r.mu.Lock()
		r.active += ev.activeDelta
		if r.active > r.peak {
			r.peak = r.active
			r.activeMax.Set(float64(r.peak))
		}
		r.activeNow.Set(float64(r.active))
		r.mu.Unlock()
	}
}

func (r *prometheusRuntime) Endpoint() string {
	return r.endpoint
}

func (r *prometheusRuntime) ListenAddr() string {
	return r.listenAddr
}

func (r *prometheusRuntime) Close(ctx context.Context) error {
	if r.server == nil {
		return nil
	}

	err := r.server.Shutdown(ctx)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
