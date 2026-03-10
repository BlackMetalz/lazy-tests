package engine

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kienlt/lazy-tests/internal/scenario"
)

func TestRun_HoldOpen_ReachesExpectedActivePeak(t *testing.T) {
	srv := newTestTCPServer(t)
	defer srv.Close()

	host, port := splitAddr(t, srv.Addr())
	sc := scenario.Scenario{
		Version:  scenario.VersionV1,
		Name:     "hold-open",
		Protocol: scenario.ProtocolTCP,
		Target: scenario.Target{
			Host: host,
			Port: port,
		},
		Workload: scenario.Workload{
			Pattern:           scenario.PatternHoldOpen,
			Connections:       20,
			ConnectRatePerSec: 200,
			Duration:          scenario.Duration(600 * time.Millisecond),
			HoldTime:          scenario.Duration(500 * time.Millisecond),
		},
		Timeouts: scenario.Timeouts{Connect: scenario.Duration(300 * time.Millisecond)},
		Assertions: scenario.Assertions{
			MaxErrorRatePct: 0,
			MaxP95ConnectMs: 100,
		},
		Safety: scenario.Safety{
			MaxConnectionsCap: 100,
			PrivateOnly:       boolPtr(true),
		},
		Output: scenario.Output{ReportDir: t.TempDir()},
	}

	eng := New()
	result, err := eng.Run(context.Background(), sc, RunOptions{})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Metrics.ActivePeak != 20 {
		t.Fatalf("expected active peak 20, got %d", result.Metrics.ActivePeak)
	}
	if !result.Assertions.Passed {
		t.Fatalf("expected assertions pass, got failures: %v", result.Assertions.Failures)
	}
}

func TestRun_ConnectChurn_ProducesLatencyAndAttempts(t *testing.T) {
	srv := newTestTCPServer(t)
	defer srv.Close()

	host, port := splitAddr(t, srv.Addr())
	sc := scenario.Scenario{
		Version:  scenario.VersionV1,
		Name:     "connect-churn",
		Protocol: scenario.ProtocolTCP,
		Target: scenario.Target{
			Host: host,
			Port: port,
		},
		Workload: scenario.Workload{
			Pattern:           scenario.PatternConnectChurn,
			Connections:       50,
			ConnectRatePerSec: 200,
			Duration:          scenario.Duration(400 * time.Millisecond),
		},
		Timeouts: scenario.Timeouts{Connect: scenario.Duration(300 * time.Millisecond)},
		Assertions: scenario.Assertions{
			MaxErrorRatePct: 100,
		},
		Safety: scenario.Safety{
			MaxConnectionsCap: 100,
			PrivateOnly:       boolPtr(true),
		},
		Output: scenario.Output{ReportDir: t.TempDir()},
	}

	eng := New()
	result, err := eng.Run(context.Background(), sc, RunOptions{})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Metrics.Attempted < 30 {
		t.Fatalf("expected at least 30 attempts, got %d", result.Metrics.Attempted)
	}
	if result.Metrics.ConnectLatencyMs.P95 <= 0 {
		t.Fatalf("expected p95 latency > 0, got %.2f", result.Metrics.ConnectLatencyMs.P95)
	}
	if result.Metrics.Connected+result.Metrics.Failed != result.Metrics.Attempted {
		t.Fatalf("connected + failed must equal attempted")
	}
}

func TestRun_HalfCloseHold_ReachesExpectedActivePeak(t *testing.T) {
	srv := newTestTCPServer(t)
	defer srv.Close()

	host, port := splitAddr(t, srv.Addr())
	sc := scenario.Scenario{
		Version:  scenario.VersionV1,
		Name:     "half-close-hold",
		Protocol: scenario.ProtocolTCP,
		Target: scenario.Target{
			Host: host,
			Port: port,
		},
		Workload: scenario.Workload{
			Pattern:           scenario.PatternHalfCloseHold,
			Connections:       15,
			ConnectRatePerSec: 150,
			Duration:          scenario.Duration(700 * time.Millisecond),
			HoldTime:          scenario.Duration(500 * time.Millisecond),
		},
		Timeouts: scenario.Timeouts{Connect: scenario.Duration(300 * time.Millisecond)},
		Assertions: scenario.Assertions{
			MaxErrorRatePct: 0,
			MaxP95ConnectMs: 100,
		},
		Safety: scenario.Safety{
			MaxConnectionsCap: 100,
			PrivateOnly:       boolPtr(true),
		},
		Output: scenario.Output{ReportDir: t.TempDir()},
	}

	eng := New()
	result, err := eng.Run(context.Background(), sc, RunOptions{})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Metrics.ActivePeak != 15 {
		t.Fatalf("expected active peak 15, got %d", result.Metrics.ActivePeak)
	}
	if !result.Assertions.Passed {
		t.Fatalf("expected assertions pass, got failures: %v", result.Assertions.Failures)
	}
}

func TestRun_UnreachablePort_ClassifiesRefusedAndFailsAssertion(t *testing.T) {
	port := closedLocalPort(t)
	sc := scenario.Scenario{
		Version:  scenario.VersionV1,
		Name:     "unreachable",
		Protocol: scenario.ProtocolTCP,
		Target: scenario.Target{
			Host: "127.0.0.1",
			Port: port,
		},
		Workload: scenario.Workload{
			Pattern:           scenario.PatternConnectChurn,
			Connections:       10,
			ConnectRatePerSec: 100,
			Duration:          scenario.Duration(200 * time.Millisecond),
		},
		Timeouts: scenario.Timeouts{Connect: scenario.Duration(100 * time.Millisecond)},
		Assertions: scenario.Assertions{
			MaxErrorRatePct: 0,
		},
		Safety: scenario.Safety{
			MaxConnectionsCap: 100,
			PrivateOnly:       boolPtr(true),
		},
		Output: scenario.Output{ReportDir: t.TempDir()},
	}

	eng := New()
	result, err := eng.Run(context.Background(), sc, RunOptions{})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Metrics.ErrorsByType["refused"] == 0 {
		t.Fatalf("expected refused errors, got %+v", result.Metrics.ErrorsByType)
	}
	if result.Assertions.Passed {
		t.Fatalf("expected assertion failure due to 100%% error rate")
	}
}

func TestRun_UnreachablePort_ForAllDBProtocols(t *testing.T) {
	protocols := []string{
		scenario.ProtocolMySQL,
		scenario.ProtocolRedis,
		scenario.ProtocolPostgres,
	}

	for _, protocol := range protocols {
		t.Run(protocol, func(t *testing.T) {
			port := closedLocalPort(t)
			sc := scenario.Scenario{
				Version:  scenario.VersionV1,
				Name:     "unreachable-" + protocol,
				Protocol: protocol,
				Target: scenario.Target{
					Host: "127.0.0.1",
					Port: port,
				},
				Auth: scenario.Auth{
					Username: "test",
					Password: "secret",
					Database: "test",
					RedisDB:  0,
				},
				Workload: scenario.Workload{
					Pattern:           scenario.PatternConnectChurn,
					Connections:       8,
					ConnectRatePerSec: 80,
					Duration:          scenario.Duration(250 * time.Millisecond),
				},
				Timeouts: scenario.Timeouts{Connect: scenario.Duration(100 * time.Millisecond)},
				Assertions: scenario.Assertions{
					MaxErrorRatePct: 0,
				},
				Safety: scenario.Safety{
					MaxConnectionsCap: 100,
					PrivateOnly:       boolPtr(true),
				},
				Output: scenario.Output{ReportDir: t.TempDir()},
			}

			eng := New()
			result, err := eng.Run(context.Background(), sc, RunOptions{})
			if err != nil {
				t.Fatalf("run failed: %v", err)
			}

			if result.Metrics.Failed == 0 {
				t.Fatalf("expected failures for unreachable %s", protocol)
			}
			if result.Assertions.Passed {
				t.Fatalf("expected assertions to fail for unreachable %s", protocol)
			}
			if result.Scenario.Auth.Password != "***redacted***" {
				t.Fatalf("expected password to be redacted in report summary")
			}
		})
	}
}

func TestRun_PrometheusEnabled_ExposesEndpointMetadata(t *testing.T) {
	srv := newTestTCPServer(t)
	defer srv.Close()

	host, port := splitAddr(t, srv.Addr())
	sc := scenario.Scenario{
		Version:  scenario.VersionV1,
		Name:     "prometheus-enabled",
		Protocol: scenario.ProtocolTCP,
		Target: scenario.Target{
			Host: host,
			Port: port,
		},
		Workload: scenario.Workload{
			Pattern:           scenario.PatternConnectChurn,
			Connections:       20,
			ConnectRatePerSec: 100,
			Duration:          scenario.Duration(300 * time.Millisecond),
		},
		Timeouts: scenario.Timeouts{Connect: scenario.Duration(200 * time.Millisecond)},
		Safety: scenario.Safety{
			MaxConnectionsCap: 100,
			PrivateOnly:       boolPtr(true),
		},
		Output: scenario.Output{
			ReportDir: t.TempDir(),
			Prometheus: scenario.PrometheusOutput{
				Enabled:    true,
				ListenAddr: "127.0.0.1:0",
			},
		},
	}

	eng := New()
	result, err := eng.Run(context.Background(), sc, RunOptions{})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if !result.Prometheus.Enabled {
		t.Fatalf("expected prometheus metadata to be enabled")
	}
	if result.Prometheus.Endpoint == "" {
		t.Fatalf("expected prometheus endpoint")
	}
}

func boolPtr(v bool) *bool { return &v }

type testTCPServer struct {
	ln      net.Listener
	wg      sync.WaitGroup
	accepts atomic.Int64
}

func newTestTCPServer(t *testing.T) *testTCPServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	s := &testTCPServer{ln: ln}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				if isListenerClosed(err) {
					return
				}
				return
			}
			s.accepts.Add(1)
			s.wg.Add(1)
			go func(c net.Conn) {
				defer s.wg.Done()
				defer c.Close()
				_, _ = io.Copy(io.Discard, c)
			}(conn)
		}
	}()

	return s
}

func (s *testTCPServer) Addr() string {
	return s.ln.Addr().String()
}

func (s *testTCPServer) AcceptCount() int64 {
	return s.accepts.Load()
}

func (s *testTCPServer) Close() {
	_ = s.ln.Close()
	s.wg.Wait()
}

func splitAddr(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host port: %v", err)
	}
	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}
	return host, port
}

func closedLocalPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	_, port := splitAddr(t, ln.Addr().String())
	_ = ln.Close()
	return port
}

func isListenerClosed(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "accept tcp 127.0.0.1: use of closed network connection" ||
		err.Error() == "use of closed network connection"
}
