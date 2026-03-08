package cli

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestListDrivers(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	app := New(&out, &errOut)

	code := app.Run([]string{"list", "drivers"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d, stderr=%s", code, errOut.String())
	}
	got := strings.Fields(strings.TrimSpace(out.String()))
	want := []string{"mysql", "postgres", "redis", "tcp"}
	if len(got) != len(want) {
		t.Fatalf("unexpected drivers output: %q", out.String())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected driver %q at index %d, got %q", want[i], i, got[i])
		}
	}
}

func TestValidateScenario(t *testing.T) {
	scenarioPath := writeScenarioFile(t, scenarioYAML(`
name: validate
pattern: connect-churn
host: 127.0.0.1
port: 3306
connections: 10
rate: 20
duration: 1s
hold: 0s
out: ./reports
max_error: 100
max_p95: 1000
`))

	var out bytes.Buffer
	var errOut bytes.Buffer
	app := New(&out, &errOut)

	code := app.Run([]string{"validate", "-f", scenarioPath})
	if code != 0 {
		t.Fatalf("expected code 0, got %d stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "is valid") {
		t.Fatalf("expected valid output, got: %s", out.String())
	}
}

func TestRunCreatesReports(t *testing.T) {
	srv := newTestTCPServer(t)
	defer srv.Close()

	host, port := splitAddr(t, srv.Addr())
	reportsDir := t.TempDir()
	scenarioPath := writeScenarioFile(t, scenarioYAML(fmt.Sprintf(`
name: run-report
pattern: connect-churn
host: %s
port: %d
connections: 50
rate: 200
duration: 300ms
hold: 0s
out: %s
max_error: 100
max_p95: 1000
`, host, port, reportsDir)))

	var out bytes.Buffer
	var errOut bytes.Buffer
	app := New(&out, &errOut)

	code := app.Run([]string{"run", "-f", scenarioPath})
	if code != 0 {
		t.Fatalf("expected code 0, got %d stderr=%s", code, errOut.String())
	}

	entries, err := os.ReadDir(reportsDir)
	if err != nil {
		t.Fatalf("read reports dir: %v", err)
	}

	var hasJSON bool
	var hasMD bool
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".json") {
			hasJSON = true
		}
		if strings.HasSuffix(entry.Name(), ".md") {
			hasMD = true
		}
	}

	if !hasJSON || !hasMD {
		t.Fatalf("expected report json and md, got entries: %v", entries)
	}
}

func TestRunDryRunDoesNotOpenConnections(t *testing.T) {
	srv := newTestTCPServer(t)
	defer srv.Close()

	host, port := splitAddr(t, srv.Addr())
	scenarioPath := writeScenarioFile(t, scenarioYAML(fmt.Sprintf(`
name: dry-run
pattern: hold-open
host: %s
port: %d
connections: 20
rate: 100
duration: 300ms
hold: 300ms
out: ./reports
max_error: 100
max_p95: 1000
`, host, port)))

	var out bytes.Buffer
	var errOut bytes.Buffer
	app := New(&out, &errOut)

	code := app.Run([]string{"run", "-f", scenarioPath, "--dry-run"})
	if code != 0 {
		t.Fatalf("expected code 0, got %d stderr=%s", code, errOut.String())
	}

	time.Sleep(100 * time.Millisecond)
	if srv.AcceptCount() != 0 {
		t.Fatalf("expected zero accepted connections in dry-run, got %d", srv.AcceptCount())
	}
}

func TestRunUnreachablePortReturnsAssertionExitCode(t *testing.T) {
	port := closedLocalPort(t)
	reportsDir := t.TempDir()
	scenarioPath := writeScenarioFile(t, scenarioYAML(fmt.Sprintf(`
name: unreachable
pattern: connect-churn
host: 127.0.0.1
port: %d
connections: 10
rate: 100
duration: 200ms
hold: 0s
out: %s
max_error: 0
max_p95: 1000
`, port, reportsDir)))

	var out bytes.Buffer
	var errOut bytes.Buffer
	app := New(&out, &errOut)

	code := app.Run([]string{"run", "-f", scenarioPath})
	if code != 2 {
		t.Fatalf("expected assertion exit code 2, got %d stderr=%s", code, errOut.String())
	}
}

func writeScenarioFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "scenario.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write scenario: %v", err)
	}
	return path
}

func scenarioYAML(values string) string {
	vals := parseKeyValues(values)
	return fmt.Sprintf(`version: v1
name: %s
protocol: tcp
target:
  host: %s
  port: %s
workload:
  pattern: %s
  connections: %s
  connect_rate_per_sec: %s
  duration: %s
  hold_time: %s
timeouts:
  connect: 200ms
assertions:
  max_error_rate_pct: %s
  max_p95_connect_ms: %s
safety:
  max_connections_cap: 5000
  private_network_only: true
output:
  report_dir: %s
`,
		vals["name"],
		vals["host"],
		vals["port"],
		vals["pattern"],
		vals["connections"],
		vals["rate"],
		vals["duration"],
		vals["hold"],
		vals["max_error"],
		vals["max_p95"],
		vals["out"],
	)
}

func parseKeyValues(values string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(values, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return result
}

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
				if strings.Contains(err.Error(), "closed network connection") {
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
