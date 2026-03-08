# LT-TCP-03 — Conntrack storm (TCP churn)

## Goal
Drive high connection churn to push `Conntrack` usage and `new/sec`.

## Command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml
```

## Expected in holyf-network

- `Conntrack`: `used` and `new/sec` increase sharply.
- `Connection States`: frequent transitions across active/closing states.
