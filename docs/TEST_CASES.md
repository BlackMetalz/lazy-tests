# Test Cases for holyf-network with lazy-tests

This document is a practical test catalog for using `lazy-tests` to stress and validate `holyf-network` panels.

If you want one-case-per-file format, use:

- `docs/HOLYF_CASE_PACK.md`

## Coverage Summary

| ID | Goal | Mode | Main command |
| --- | --- | --- | --- |
| LT-TCP-01 | TIME_WAIT around 500 | native | `lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml` |
| LT-TCP-02 | ESTABLISHED saturation (1k hold-open) | native | `lazy-tests run -f examples/scenarios/tcp-established-1k.yaml` |
| LT-TCP-03 | Conntrack storm / high churn | native | `lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml` |
| LT-DB-01 | MySQL connect storm | native | `lazy-tests run -f examples/scenarios/mysql-connect-storm.yaml` |
| LT-DB-02 | Redis hold-open heavy | native | `lazy-tests run -f examples/scenarios/redis-hold-open-heavy.yaml` |
| LT-DB-03 | Postgres hold-open heavy | native | `lazy-tests run -f examples/scenarios/postgres-hold-open-heavy.yaml` |
| LT-SAFE-01 | Public target safety guardrail | native | run without `--unsafe-public-target` |
| LT-OBS-01 | Prometheus endpoint export during run | native | any scenario with `output.prometheus.enabled: true` |
| LAB-CW-01 | CLOSE_WAIT accumulation | hybrid | `labs/high-conntrack` + `tcp-close-wait-pressure.yaml` |
| LAB-NAT-01 | NAT visibility (`ct/nat`) | hybrid | docker NAT lab + `tcp-docker-nat.yaml` |
| LAB-RTR-01 | Retransmission thresholds | hybrid | netem lab + sustained traffic |
| LAB-MIT-01 | holyf kill/block under storm | hybrid | run storm then trigger `k/Enter` in holyf |

## Native Cases (lazy-tests only)

### LT-TCP-01: TIME_WAIT around 500

- Objective: generate short-lived TCP churn and observe `TIME_WAIT` increase.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml
```

- Expected in holyf:
  - `Connection States`: visible `TIME_WAIT` spike.
  - `Top Connections`: target port row dominates by connection count.

### LT-TCP-02: ESTABLISHED saturation (1k)

- Objective: hold many active sockets concurrently.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml
```

- Expected in holyf:
  - `Connection States`: high `ESTABLISHED` count during run window.
  - `Top Connections`: one/few peers consume majority of active connections.

### LT-TCP-03: conntrack storm

- Objective: drive `conntrack` usage and `new/sec` harder than normal workloads.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml
```

- Expected in holyf:
  - `Conntrack`: `used`, `new/sec` and pressure bar rise.
  - `Connection States`: frequent transitions (`SYN`, `ESTABLISHED`, `TIME_WAIT`).

### LT-DB-01: MySQL connect storm

- Objective: stress DB auth/connect path with high churn.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/mysql-connect-storm.yaml
```

- Expected in holyf:
  - DB port row (`3306`) increases strongly in Top Connections.
  - If DB limited, `failed`/`timeout` may rise in lazy-tests report.

### LT-DB-02: Redis hold-open heavy

- Objective: maintain many long-lived Redis sockets.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/redis-hold-open-heavy.yaml
```

- Expected in holyf:
  - Stable high `ESTABLISHED` count on `6379`.
  - Conntrack stays elevated while test is active.

### LT-DB-03: Postgres hold-open heavy

- Objective: long-lived Postgres pressure.
- Command:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/postgres-hold-open-heavy.yaml
```

- Expected in holyf:
  - High `ESTABLISHED` on `5432`.
  - Top Connections group by process/peer reflects dominant DB clients.

### LT-SAFE-01: public target guardrail

- Objective: verify accidental public-target tests are blocked.
- Steps:
  1. set scenario host to a public IP/FQDN.
  2. run without `--unsafe-public-target`.

- Expected:
  - CLI exits with code `1` and guardrail error.

### LT-OBS-01: Prometheus export verification

- Objective: validate optional `/metrics` endpoint while run is active.
- Steps:
  1. use any scenario with `output.prometheus.enabled: true`.
  2. start run.
  3. curl endpoint printed in summary.

- Expected:
  - endpoint responds with counters/histograms (`lazy_tests_*`).

## Hybrid Labs (for missing observability states)

### LAB-CW-01: CLOSE_WAIT accumulation

- Why hybrid: `CLOSE_WAIT` depends on server app behavior (not just connect churn).
- Use `half-close-hold` profile for a stable plateau instead of short-lived churn spikes.
- Steps:

```bash
go run ./labs/high-conntrack/server -listen :18080 -leak-close-wait -leak-limit 3000 -log-every 1
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-500.yaml --target-host <SERVER_IP>
```

- Expected in holyf:
  - `Connection States`: `CLOSE_WAIT` increases.
  - `Conntrack`: pressure increases during churn.

### LAB-NAT-01: NAT (`ct/nat`) visibility

- Why hybrid: needs Docker/NAT topology.
- Steps:

```bash
docker compose -f labs/nat/docker-compose.yml up -d
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-docker-nat.yaml
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
- Steps:

```bash
sudo tc qdisc add dev eth0 root netem delay 80ms 20ms loss 3%
curl --http1.1 -L http://speedtest.tele2.net/1GB.zip -o /dev/null
sudo tc qdisc del dev eth0 root
```

- Expected in holyf:
  - retrans rate and retrans percent increase.
  - threshold status can move into warn/crit depending on health config.

### LAB-MIT-01: block/kill convergence under storm

- Why hybrid: verifies holyf active mitigation path (`k`, timed block, kill-only).
- Steps:
  1. run `tcp-conntrack-storm.yaml`.
  2. in holyf live TUI, pick hot peer row and trigger `k/Enter`.
  3. test both `minutes > 0` (block) and `minutes = 0` (kill-only).

- Expected in holyf:
  - active connections decrease for target peer.
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
