# LT-TCP-01 — TIME_WAIT spike (~500)

## Goal
Create short-lived TCP churn to surface `TIME_WAIT` behavior in holyf.

## Command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml
```

## Expected in holyf-network

- `Connection States`: `TIME_WAIT` increases clearly.
- `Top Connections`: target port row becomes dominant by connection count.
