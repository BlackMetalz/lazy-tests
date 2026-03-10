# LT-DB-03 — Postgres hold-open heavy

## Goal
Create sustained Postgres active-connection pressure.

## Command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/postgres-hold-open-heavy.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

## Expected in holyf-network

- High `ESTABLISHED` on port `5432`.
- Conntrack usage remains elevated while run is active.
