# LAB-CW-01 — CLOSE_WAIT accumulation

## Goal
Force server-side `CLOSE_WAIT` to validate holyf state panel behavior.

## Setup

```bash
go run ./labs/high-conntrack/server -listen :18080 -leak-close-wait -leak-limit 3000
```

## Traffic command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-pressure.yaml
```

## Expected in holyf-network

- `Connection States`: `CLOSE_WAIT` grows over time.
- `Conntrack`: pressure rises while churn is active.
