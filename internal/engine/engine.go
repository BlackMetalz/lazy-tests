package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/kienlt/lazy-tests/internal/driver"
	mysqldriver "github.com/kienlt/lazy-tests/internal/driver/mysql"
	postgresdriver "github.com/kienlt/lazy-tests/internal/driver/postgres"
	redisdriver "github.com/kienlt/lazy-tests/internal/driver/redis"
	tcpdriver "github.com/kienlt/lazy-tests/internal/driver/tcp"
	"github.com/kienlt/lazy-tests/internal/scenario"
)

type Engine struct {
	drivers map[string]driver.Driver
	now     func() time.Time
}

func New() *Engine {
	return &Engine{
		drivers: map[string]driver.Driver{
			scenario.ProtocolTCP:      tcpdriver.New(),
			scenario.ProtocolMySQL:    mysqldriver.New(),
			scenario.ProtocolRedis:    redisdriver.New(),
			scenario.ProtocolPostgres: postgresdriver.New(),
		},
		now: time.Now,
	}
}

func (e *Engine) Drivers() []string {
	result := make([]string, 0, len(e.drivers))
	for name := range e.drivers {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (e *Engine) Run(ctx context.Context, sc scenario.Scenario, opts RunOptions) (Result, error) {
	if err := sc.Validate(); err != nil {
		return Result{}, err
	}

	if sc.PrivateNetworkOnly() && !opts.UnsafePublicTarget {
		private, err := scenario.IsPrivateHost(sc.Target.Host)
		if err != nil {
			return Result{}, fmt.Errorf("private network check failed: %w", err)
		}
		if !private {
			return Result{}, errors.New("target host is not private; use --unsafe-public-target to proceed")
		}
	}

	start := e.now()
	result := Result{
		Scenario: ScenarioSummary{
			Name:     sc.Name,
			Protocol: sc.Protocol,
			Target:   sc.Target,
			Auth:     redactAuth(sc.Auth),
			Workload: sc.Workload,
			Timeouts: sc.Timeouts,
			Safety:   sc.Safety,
			Output:   sc.Output,
		},
		DryRun: opts.DryRun,
		RunTiming: RunTiming{
			Start: start,
		},
		Prometheus: PrometheusEndpoint{Enabled: sc.Output.Prometheus.Enabled},
	}

	if opts.DryRun {
		now := e.now()
		result.RunTiming.End = now
		result.RunTiming.Duration = now.Sub(start)
		result.Assertions = AssertionResult{Passed: true}
		result.Socket = SocketStates{Available: false, Message: "dry-run"}
		result.Metrics.ErrorsByType = map[string]int{}
		if sc.Output.Prometheus.Enabled {
			result.Prometheus.Message = "dry-run; prometheus exporter not started"
		}
		return result, nil
	}

	drv, ok := e.drivers[sc.Protocol]
	if !ok {
		return Result{}, fmt.Errorf("no driver registered for protocol %q", sc.Protocol)
	}

	var live liveMetrics
	if sc.Output.Prometheus.Enabled {
		runtime, err := newPrometheusRuntime(sc.Output.Prometheus.ListenAddr)
		if err != nil {
			return Result{}, err
		}
		live = runtime
		result.Prometheus.ListenAddr = runtime.ListenAddr()
		result.Prometheus.Endpoint = runtime.Endpoint()

		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = live.Close(shutdownCtx)
		}()
	}

	events := make(chan event, 4096)
	agg := newAggregator(live)
	var aggWG sync.WaitGroup
	aggWG.Add(1)
	go func() {
		defer aggWG.Done()
		agg.consume(events)
	}()

	runCtx, cancel := context.WithTimeout(ctx, sc.Workload.Duration.Value())
	defer cancel()

	target := driver.Target{
		Protocol: sc.Protocol,
		Host:     sc.Target.Host,
		Port:     sc.Target.Port,
		Auth:     sc.Auth,
	}

	switch sc.Workload.Pattern {
	case scenario.PatternConnectChurn:
		e.runConnectChurn(runCtx, drv, sc, target, events)
	case scenario.PatternHoldOpen:
		e.runHoldOpen(runCtx, drv, sc, target, events)
	default:
		close(events)
		aggWG.Wait()
		return Result{}, fmt.Errorf("unsupported pattern %q", sc.Workload.Pattern)
	}

	close(events)
	aggWG.Wait()

	end := e.now()
	result.RunTiming.End = end
	result.RunTiming.Duration = end.Sub(start)
	result.Metrics = agg.snapshot()
	result.Assertions = EvaluateAssertions(result.Metrics, sc.Assertions)
	result.Socket = probeSocketStates(ctx, sc.Target.Host, sc.Target.Port)
	if sc.Output.Prometheus.Enabled {
		result.Prometheus.Message = "prometheus exporter ran during test execution"
	}

	return result, nil
}

func (e *Engine) runConnectChurn(
	ctx context.Context,
	drv driver.Driver,
	sc scenario.Scenario,
	target driver.Target,
	events chan<- event,
) {
	interval := rateInterval(sc.Workload.ConnectRatePerSec)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sem := make(chan struct{}, sc.Workload.Connections)
	var wg sync.WaitGroup

	spawnAttempt := func() {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			events <- event{attempted: 1}
			start := time.Now()
			sess, err := drv.Connect(ctx, target, sc.Timeouts.Connect.Value())
			latencyMs := durationMs(time.Since(start))
			if err != nil {
				events <- event{failed: 1, errorType: ClassifyError(err)}
				return
			}

			events <- event{connected: 1, activeDelta: 1, latencyMs: latencyMs}
			_ = sess.Close()
			events <- event{activeDelta: -1}
		}()
	}

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
			spawnAttempt()
		}
	}
}

func (e *Engine) runHoldOpen(
	ctx context.Context,
	drv driver.Driver,
	sc scenario.Scenario,
	target driver.Target,
	events chan<- event,
) {
	interval := rateInterval(sc.Workload.ConnectRatePerSec)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var wg sync.WaitGroup
	attempts := 0
	total := sc.Workload.Connections

	holdTime := sc.Workload.HoldTime.Value()
	deadline := time.Now().Add(sc.Workload.Duration.Value())

	spawnAttempt := func() {
		attempts++
		wg.Add(1)
		go func() {
			defer wg.Done()
			events <- event{attempted: 1}

			start := time.Now()
			sess, err := drv.Connect(ctx, target, sc.Timeouts.Connect.Value())
			latencyMs := durationMs(time.Since(start))
			if err != nil {
				events <- event{failed: 1, errorType: ClassifyError(err)}
				return
			}

			events <- event{connected: 1, activeDelta: 1, latencyMs: latencyMs}

			remaining := time.Until(deadline)
			if remaining < 0 {
				remaining = 0
			}

			waitFor := remaining
			if holdTime > 0 && holdTime < waitFor {
				waitFor = holdTime
			}

			if waitFor > 0 {
				select {
				case <-ctx.Done():
				case <-time.After(waitFor):
				}
			}

			_ = sess.Close()
			events <- event{activeDelta: -1}
		}()
	}

	for attempts < total {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
			spawnAttempt()
		}
	}

	<-ctx.Done()
	wg.Wait()
}

func rateInterval(connectRate int) time.Duration {
	if connectRate <= 0 {
		return time.Second
	}

	interval := time.Second / time.Duration(connectRate)
	if interval < time.Millisecond {
		return time.Millisecond
	}
	return interval
}

func durationMs(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}

func redactAuth(auth scenario.Auth) scenario.Auth {
	if auth.Password != "" {
		auth.Password = "***redacted***"
	}
	return auth
}
