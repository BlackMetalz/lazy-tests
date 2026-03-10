# LAB-CW-01 — CLOSE_WAIT accumulation

## Goal
Force server-side `CLOSE_WAIT` to validate holyf state panel behavior.

## Important behavior

`lazy-tests` client is expected to succeed here.

This case is not about client-side errors. It is about the server intentionally leaking accepted sockets after the client closes, so `CLOSE_WAIT` appears on the server side.

## Setup

```bash
go run ./labs/high-conntrack/server -listen :18080 -leak-close-wait -leak-limit 3000 -log-every 1
```

## Traffic command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-500.yaml --target-host <SERVER_IP>
```

If you need stronger pressure, use:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-pressure.yaml --target-host <SERVER_IP>
```

## Optional direct verification on Linux

While the run is active, verify `CLOSE_WAIT` count directly:

```bash
./scripts/check_close_wait.sh 18080
```

## Expected in holyf-network

- `Connection States`: `CLOSE_WAIT` grows over time.
- `Conntrack`: pressure rises while churn is active.
- `lazy-tests` report can still show high success rate. That is normal for this case.
