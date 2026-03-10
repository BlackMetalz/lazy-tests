# Test Cases for holyf-network with lazy-tests

This document is a practical test catalog for using `lazy-tests` to stress and validate `holyf-network` panels.

If you want one-case-per-file format, use:

- `docs/HOLYF_CASE_PACK.md`

All commands below pass the real endpoint by CLI using `--target-host` and `--target-port`.
Do not edit `target.host` or `target.port` in the scenario templates just to switch environments.

## Coverage Summary

| ID | Goal | Mode | Main command |
| --- | --- | --- | --- |
| LT-TCP-01 | TIME_WAIT around 500 | native | `lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml --target-host HOST --target-port PORT` |
| LT-TCP-02 | ESTABLISHED saturation (1k hold-open) | native | `lazy-tests run -f examples/scenarios/tcp-established-1k.yaml --target-host HOST --target-port PORT` |
| LT-TCP-03 | Conntrack storm / high churn | native | `lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host HOST --target-port PORT` |
| LT-DB-01 | MySQL connect storm | native | `lazy-tests run -f examples/scenarios/mysql-connect-storm.yaml --target-host HOST --target-port PORT` |
| LT-DB-02 | Redis hold-open heavy | native | `lazy-tests run -f examples/scenarios/redis-hold-open-heavy.yaml --target-host HOST --target-port PORT` |
| LT-DB-03 | Postgres hold-open heavy | native | `lazy-tests run -f examples/scenarios/postgres-hold-open-heavy.yaml --target-host HOST --target-port PORT` |
| LT-SAFE-01 | Public target safety guardrail | native | `lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host PUBLIC_HOST --target-port PUBLIC_PORT` |
| LT-OBS-01 | Prometheus endpoint export during run | native | `lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host HOST --target-port PORT` |
| LAB-CW-01 | CLOSE_WAIT accumulation | hybrid | `lazy-tests run -f examples/scenarios/tcp-close-wait-pressure.yaml --target-host HOST --target-port 18080` |
| LAB-NAT-01 | NAT visibility (`ct/nat`) | hybrid | `lazy-tests run -f examples/scenarios/tcp-docker-nat.yaml --target-host 127.0.0.1 --target-port 18080` |
| LAB-RTR-01 | Retransmission thresholds | hybrid | netem lab + sustained traffic |
| LAB-MIT-01 | holyf kill/block under storm | hybrid | `high-conntrack server :18080 + lazy-tests client + holyf TUI` |

## Native Cases (lazy-tests only)

### LT-TCP-01: TIME_WAIT around 500

- Objective: generate short-lived TCP churn and observe `TIME_WAIT` increase.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

- Expected in holyf:
  - `Connection States`: visible `TIME_WAIT` spike.
  - `Top Connections`: target port row dominates by connection count.

### LT-TCP-02: ESTABLISHED saturation (1k)

- Objective: hold many active sockets concurrently.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

- Expected in holyf:
  - `Connection States`: high `ESTABLISHED` count during run window.
  - `Top Connections`: one/few peers consume majority of active connections.

### LT-TCP-03: conntrack storm

- Objective: drive `conntrack` usage and `new/sec` harder than normal workloads.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

- Expected in holyf:
  - `Conntrack`: `used`, `new/sec` and pressure bar rise.
  - `Connection States`: frequent transitions (`SYN`, `ESTABLISHED`, `TIME_WAIT`).

### LT-DB-01: MySQL connect storm

- Objective: stress DB auth/connect path with high churn.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/mysql-connect-storm.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

- Expected in holyf:
  - DB port row (`3306`) increases strongly in Top Connections.
  - If DB limited, `failed`/`timeout` may rise in lazy-tests report.

### LT-DB-02: Redis hold-open heavy

- Objective: maintain many long-lived Redis sockets.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/redis-hold-open-heavy.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

- Expected in holyf:
  - Stable high `ESTABLISHED` count on `6379`.
  - Conntrack stays elevated while test is active.

### LT-DB-03: Postgres hold-open heavy

- Objective: long-lived Postgres pressure.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/postgres-hold-open-heavy.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

- Expected in holyf:
  - High `ESTABLISHED` on `5432`.
  - Top Connections group by process/peer reflects dominant DB clients.

### LT-SAFE-01: public target guardrail

- Objective: verify accidental public-target tests are blocked.
- Steps:
  1. run against a public IP/FQDN via CLI overrides.
  2. do not pass `--unsafe-public-target`.

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host <PUBLIC_IP_OR_FQDN> --target-port <PUBLIC_PORT>
```

- Expected:
  - CLI exits with code `1` and guardrail error.

### LT-OBS-01: Prometheus export verification

- Objective: validate optional `/metrics` endpoint while run is active.
- Steps:
  1. use any scenario with `output.prometheus.enabled: true`.
  2. start run with explicit target overrides.
  3. curl endpoint printed in summary.

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

- Expected:
  - endpoint responds with counters/histograms (`lazy_tests_*`).

## Hybrid Labs (for missing observability states)

### LAB-CW-01: CLOSE_WAIT accumulation

- Why hybrid: `CLOSE_WAIT` depends on server app behavior (not just connect churn).
- Use `half-close-hold` profile for a stable plateau instead of short-lived churn spikes.
- Steps:

```bash
go run ./labs/high-conntrack/server -listen :18080 -leak-close-wait -leak-limit 3000 -log-every 1
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-500.yaml --target-host <TARGET_HOST> --target-port 18080
```

- Expected in holyf:
  - `Connection States`: `CLOSE_WAIT` increases.
  - `Conntrack`: pressure increases during churn.

### LAB-NAT-01: NAT (`ct/nat`) visibility

- Why hybrid: needs Docker/NAT topology.
- Steps:

```bash
docker compose -f labs/nat/docker-compose.yml up -d
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-docker-nat.yaml --target-host 127.0.0.1 --target-port 18080
```

- Expected in holyf:
  - `Top Connections` may contain `ct/nat` rows.
  - Conntrack rates rise during run.

- Cleanup:

```bash
docker compose -f labs/nat/docker-compose.yml down
```

### LAB-RTR-01: retransmission thresholds

- Why hybrid: retrans spikes require path impairment or sustained payload traffic.
- Acceptance note: holyf can legitimately stay in `LOW SAMPLE` until retrans scoring has enough data.
- Steps:

```bash
go run ./labs/high-conntrack/server -listen :18080 -hold 300ms -write-delay 50ms -stats-every 1s -log-every 1000
ip route get <CLIENT_IP>
go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 20 -concurrency 1 -timeout 5s -read-reply
sudo tc qdisc add dev <TARGET_INTERFACE> root netem delay 80ms 20ms loss 3%
go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 5000 -concurrency 100 -timeout 5s -read-reply -delay 5ms
# optional scale-up:
# go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 20000 -concurrency 400 -timeout 5s -read-reply -delay 2ms
# cleanup:
# sudo tc qdisc del dev <TARGET_INTERFACE> root
```

- Expected in holyf:
  - retrans rate and retrans percent increase.
  - health strip may remain `LOW SAMPLE` until sample gates are met.
  - with current holyf thresholds, scoring starts only once `ESTABLISHED >= 20` and `OutSegs/sec >= 60`.
  - threshold status can move into warn/crit only after those gates are met.

### LAB-MIT-01: block/kill convergence under storm

- Why hybrid: needs a target server plus a separate traffic generator so holyf can mitigate a real hot peer.
- Topology:
  - target host: run the bundled TCP server and holyf-network.
  - client host: run `lazy-tests` against the target host on port `18080`.

```bash
go run ./labs/high-conntrack/server -listen :18080 -read-timeout 5m -log-every 1000

# timed block validation under churn pressure
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host <TARGET_HOST> --target-port 18080

# kill-only validation with clearer active connection drop
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml --target-host <TARGET_HOST> --target-port 18080
```

- In holyf live TUI:
  - pick the hot peer row created by the client host.
  - trigger `k/Enter`.
  - test `minutes > 0` with the churn profile and `minutes = 0` with the hold-open profile.

- Expected in holyf:
  - timed block suppresses new connections from that peer while the block is active.
  - kill-only drops current active connections, but new attempts can reappear afterward.
  - under heavy race, partial convergence may appear (expected bounded behavior).

## Recommended Regression Bundle

Run these after major collector or mitigation changes:

1. `LT-TCP-01`
2. `LT-TCP-03`
3. `LT-DB-01`
4. `LAB-CW-01`
5. `LAB-NAT-01`
6. `LAB-RTR-01`
7. `LAB-MIT-01`
