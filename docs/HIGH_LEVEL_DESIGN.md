# High-Level Design (v0.3)

## Goals

- Single-node CLI framework for connection stress tests.
- Protocol-agnostic engine with pluggable drivers.
- Deterministic outputs for CI (exit codes + JSON reports).

## System Overview

`lazy-tests` is composed of five layers:

1. `cmd/lazy-tests`
- Thin executable entrypoint.

2. `internal/cli`
- Command parsing for `run`, `validate`, and `list drivers`.
- Exit code contract:
  - `0`: success.
  - `1`: config/runtime failure.
  - `2`: assertion failure.

3. `internal/scenario`
- YAML schema parsing and validation.
- Safety defaults and guardrails:
  - private-network-only by default.
  - hard connection cap.
- Override support for common run-time flags.

4. `internal/engine`
- Core scheduler/executor for workload patterns:
  - `hold-open`
  - `connect-churn`
- Event-based aggregation for counters/latency/error types.
- Assertion evaluation.
- Best-effort socket state probing (`ESTABLISHED`, `TIME_WAIT`).
- Optional live Prometheus exporter (`/metrics`) per run.

5. `internal/driver/*`
- Protocol adapters implementing the common `Driver` interface:
  - `tcp`
  - `mysql`
  - `redis`
  - `postgres`

6. `internal/report`
- Persists `json` and `markdown` reports.
- Includes metrics, assertion outcomes, socket states, and Prometheus metadata.

## Extension Points

- Add a new protocol by implementing `internal/driver.Driver` and registering it in `engine.New()`.
- Extend assertions in `internal/engine/assertions.go` without changing runner lifecycle.
- Add extra output channels (e.g. OTEL, Kafka) by plugging into the event stream.

## holyf-network Coverage Strategy

- Native `lazy-tests` scenarios cover connection-oriented stress:
  - churn and TIME_WAIT pressure
  - high ESTABLISHED hold-open
  - DB connect storms (MySQL/Redis/Postgres)
- Hybrid lab helpers cover states that require external topology/behavior:
  - deliberate `CLOSE_WAIT` accumulation (`labs/high-conntrack`)
  - Docker NAT visibility (`labs/nat`)
  - retransmission under path impairment (`labs/retrans`)
- Full runbook and expectations are documented in `docs/TEST_CASES.md`.

## Runtime Flow

1. CLI loads and validates scenario.
2. Safety guardrails are applied.
3. Engine selects protocol driver.
4. Workload executes with rate-based spawning.
5. Aggregator computes metrics in real-time.
6. Assertions are evaluated.
7. Reports are written and exit code is returned.

## Security Notes

- Passwords are redacted in result summaries and reports.
- Public targets require explicit `--unsafe-public-target` opt-in.
