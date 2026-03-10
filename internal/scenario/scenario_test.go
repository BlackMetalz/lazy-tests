package scenario

import (
	"strings"
	"testing"
)

func TestParse_MissingName(t *testing.T) {
	yaml := `
version: v1
protocol: tcp
target:
  host: 127.0.0.1
  port: 3306
workload:
  pattern: connect-churn
  connections: 10
  connect_rate_per_sec: 10
  duration: 1s
timeouts:
  connect: 1s
safety:
  max_connections_cap: 100
`

	_, err := Parse([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name validation error, got: %v", err)
	}
}

func TestParse_InvalidDuration(t *testing.T) {
	yaml := `
version: v1
name: bad-duration
protocol: tcp
target:
  host: 127.0.0.1
  port: 3306
workload:
  pattern: connect-churn
  connections: 10
  connect_rate_per_sec: 10
  duration: nope
timeouts:
  connect: 1s
safety:
  max_connections_cap: 100
`

	_, err := Parse([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "invalid duration") {
		t.Fatalf("expected invalid duration error, got: %v", err)
	}
}

func TestParse_CapViolation(t *testing.T) {
	yaml := `
version: v1
name: cap-violation
protocol: tcp
target:
  host: 127.0.0.1
  port: 3306
workload:
  pattern: connect-churn
  connections: 101
  connect_rate_per_sec: 10
  duration: 1s
timeouts:
  connect: 1s
safety:
  max_connections_cap: 100
`

	_, err := Parse([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "exceeds safety.max_connections_cap") {
		t.Fatalf("expected cap violation error, got: %v", err)
	}
}

func TestParse_AcceptsAllProtocols(t *testing.T) {
	protocols := []string{ProtocolTCP, ProtocolMySQL, ProtocolRedis, ProtocolPostgres}
	for _, protocol := range protocols {
		t.Run(protocol, func(t *testing.T) {
			yaml := `
version: v1
name: proto
protocol: ` + protocol + `
target:
  host: 127.0.0.1
  port: 3306
workload:
  pattern: connect-churn
  connections: 10
  connect_rate_per_sec: 10
  duration: 1s
timeouts:
  connect: 1s
safety:
  max_connections_cap: 100
`
			if _, err := Parse([]byte(yaml)); err != nil {
				t.Fatalf("expected protocol %s to be valid, got %v", protocol, err)
			}
		})
	}
}

func TestParse_RedisDBMustBeNonNegative(t *testing.T) {
	yaml := `
version: v1
name: redis
protocol: redis
target:
  host: 127.0.0.1
  port: 6379
auth:
  redis_db: -1
workload:
  pattern: connect-churn
  connections: 10
  connect_rate_per_sec: 10
  duration: 1s
timeouts:
  connect: 1s
safety:
  max_connections_cap: 100
`

	_, err := Parse([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "auth.redis_db must be >= 0") {
		t.Fatalf("expected redis_db validation error, got: %v", err)
	}
}

func TestParse_DefaultPrometheusAddress(t *testing.T) {
	yaml := `
version: v1
name: prom
protocol: tcp
target:
  host: 127.0.0.1
  port: 3306
workload:
  pattern: connect-churn
  connections: 10
  connect_rate_per_sec: 10
  duration: 1s
timeouts:
  connect: 1s
safety:
  max_connections_cap: 100
output:
  prometheus:
    enabled: true
`

	sc, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("expected valid scenario, got: %v", err)
	}
	if sc.Output.Prometheus.ListenAddr == "" {
		t.Fatalf("expected default prometheus listen addr")
	}
}

func TestParse_HalfCloseHoldRequiresTCP(t *testing.T) {
	yaml := `
version: v1
name: half-close-on-redis
protocol: redis
target:
  host: 127.0.0.1
  port: 6379
workload:
  pattern: half-close-hold
  connections: 10
  connect_rate_per_sec: 10
  duration: 1s
  hold_time: 1s
timeouts:
  connect: 1s
safety:
  max_connections_cap: 100
`

	_, err := Parse([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "requires protocol tcp") {
		t.Fatalf("expected tcp-only validation error, got: %v", err)
	}
}

func TestParse_HalfCloseHoldRequiresHoldTime(t *testing.T) {
	yaml := `
version: v1
name: half-close-no-hold
protocol: tcp
target:
  host: 127.0.0.1
  port: 18080
workload:
  pattern: half-close-hold
  connections: 10
  connect_rate_per_sec: 10
  duration: 1s
  hold_time: 0s
timeouts:
  connect: 1s
safety:
  max_connections_cap: 100
`

	_, err := Parse([]byte(yaml))
	if err == nil || !strings.Contains(err.Error(), "hold_time must be > 0 for half-close-hold") {
		t.Fatalf("expected hold_time validation error, got: %v", err)
	}
}
