# LT-TCP-02 — ESTABLISHED saturation (1k hold-open)

## Goal
Hold a large number of active sockets to validate high `ESTABLISHED` visibility.

## Command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml
```

## Expected in holyf-network

- `Connection States`: strong `ESTABLISHED` plateau during run.
- `Top Connections`: one/few peers consume most active connections.
