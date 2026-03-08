# LT-DB-01 — MySQL connect storm

## Goal
Stress MySQL connect/auth path with high churn.

## Command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/mysql-connect-storm.yaml
```

## Expected in holyf-network

- DB port (`3306`) rises in Top Connections.
- Conntrack pressure increases during burst windows.
