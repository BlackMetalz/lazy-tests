# LT-DB-02 — Redis hold-open heavy

## Goal
Validate long-lived Redis pressure with large active socket count.

## Command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/redis-hold-open-heavy.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

## Expected in holyf-network

- Stable high `ESTABLISHED` on port `6379`.
- Top Connections aggregates show dominant Redis peer rows.
