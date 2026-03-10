# lazy-tests (v0.3)

Go CLI framework for protocol-level connection stress tests.

## What Is Implemented

- Protocol drivers:
  - `tcp`
  - `mysql`
  - `redis`
  - `postgres`
- Workload patterns:
  - `hold-open`
  - `connect-churn`
- Commands:
  - `run`
  - `validate`
  - `list drivers`
- Reports:
  - terminal summary
  - `report.json`
  - `report.md`
- Assertions + CI-friendly exit codes:
  - `0`: success
  - `1`: config/runtime error
  - `2`: assertion failure
- Optional live Prometheus endpoint during run: `/metrics`

## Download
- Ubuntu
```bash
curl -sL https://github.com/BlackMetalz/lazy-tests/releases/latest/download/lazy-tests-linux-amd64 -o /tmp/lazy-tests
chmod +x /tmp/lazy-tests
sudo mv /tmp/lazy-tests /usr/local/bin/lazy-tests
sudo lazy-tests
```

## Install / Build

```bash
go mod tidy
go build ./cmd/lazy-tests
```

## Usage

### Validate a scenario

```bash
go run ./cmd/lazy-tests validate -f examples/scenarios/tcp-connect-churn.yaml
```

### List available drivers

```bash
go run ./cmd/lazy-tests list drivers
```

### Run a test

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/redis-connect-churn.yaml
```

### Dry run (no sockets opened)

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/mysql-hold-open.yaml --dry-run
```

### Run for holyf-network focused scenarios

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-500.yaml --target-host <SERVER_IP>
```

## CLI Contract

```text
lazy-tests run -f <scenario.yaml> [--out ./reports] [--dry-run] [--unsafe-public-target]
lazy-tests validate -f <scenario.yaml>
lazy-tests list drivers
```

Runtime overrides for `run`:

- `--duration`
- `--connections`
- `--connect-rate-per-sec`

## Scenario Schema (v1)

```yaml
version: v1
name: redis-connect-churn
protocol: redis # tcp | mysql | redis | postgres

target:
  host: 127.0.0.1
  port: 6379

auth:            # optional for tcp
  username: ""   # mysql/postgres
  password: ""   # mysql/redis/postgres
  database: ""   # mysql/postgres
  redis_db: 0     # redis only

workload:
  pattern: connect-churn # hold-open | connect-churn | half-close-hold (tcp only)
  connections: 500
  connect_rate_per_sec: 200
  duration: 45s
  hold_time: 20s         # used in hold-open and half-close-hold

timeouts:
  connect: 2s

assertions:
  max_error_rate_pct: 3
  max_p95_connect_ms: 150

safety:
  max_connections_cap: 5000
  private_network_only: true

output:
  report_dir: ./reports
  prometheus:
    enabled: true
    listen_addr: 127.0.0.1:2112
```

## Metrics Captured

- `attempted`, `connected`, `failed`, `active_peak`
- Connect latency: `p50`, `p95`, `p99`, `max` (ms)
- Errors by type: `timeout`, `refused`, `reset`, `dns`, `other`
- Run timing: start, end, duration
- Best-effort socket states: `ESTABLISHED`, `TIME_WAIT`

## Safety Defaults

- Targets are private-network-only by default.
- Use `--unsafe-public-target` for intentional public target testing.
- Hard cap enforcement through `safety.max_connections_cap`.

## Examples

- [examples/scenarios/tcp-connect-churn.yaml](examples/scenarios/tcp-connect-churn.yaml)
- [examples/scenarios/tcp-hold-open.yaml](examples/scenarios/tcp-hold-open.yaml)
- [examples/scenarios/tcp-timewait-500.yaml](examples/scenarios/tcp-timewait-500.yaml)
- [examples/scenarios/tcp-conntrack-storm.yaml](examples/scenarios/tcp-conntrack-storm.yaml)
- [examples/scenarios/tcp-established-1k.yaml](examples/scenarios/tcp-established-1k.yaml)
- [examples/scenarios/tcp-close-wait-500.yaml](examples/scenarios/tcp-close-wait-500.yaml)
- [examples/scenarios/tcp-close-wait-pressure.yaml](examples/scenarios/tcp-close-wait-pressure.yaml)
- [examples/scenarios/tcp-docker-nat.yaml](examples/scenarios/tcp-docker-nat.yaml)
- [examples/scenarios/mysql-hold-open.yaml](examples/scenarios/mysql-hold-open.yaml)
- [examples/scenarios/mysql-connect-storm.yaml](examples/scenarios/mysql-connect-storm.yaml)
- [examples/scenarios/redis-connect-churn.yaml](examples/scenarios/redis-connect-churn.yaml)
- [examples/scenarios/redis-hold-open-heavy.yaml](examples/scenarios/redis-hold-open-heavy.yaml)
- [examples/scenarios/postgres-connect-churn.yaml](examples/scenarios/postgres-connect-churn.yaml)
- [examples/scenarios/postgres-hold-open-heavy.yaml](examples/scenarios/postgres-hold-open-heavy.yaml)

## holyf-network Test Catalog

- One-case-per-file index: [docs/HOLYF_CASE_PACK.md](docs/HOLYF_CASE_PACK.md)
- Full testcase runbook: [docs/TEST_CASES.md](docs/TEST_CASES.md)
- Case runner helper: `scripts/run_holyf_case.sh <CASE_ID> [--run]`
- Companion labs:
  - [labs/high-conntrack/README.md](labs/high-conntrack/README.md)
  - [labs/nat/README.md](labs/nat/README.md)
  - [labs/retrans/README.md](labs/retrans/README.md)

## High-Level Design

See [docs/HIGH_LEVEL_DESIGN.md](docs/HIGH_LEVEL_DESIGN.md).
