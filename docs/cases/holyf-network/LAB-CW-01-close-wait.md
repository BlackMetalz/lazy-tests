# LAB-CW-01 — CLOSE_WAIT accumulation

## Goal
Force server-side `CLOSE_WAIT` to validate holyf state panel behavior.

## Topology

- Target host: run the bundled TCP server in `CLOSE_WAIT` leak mode and holyf-network on the same machine.
- Client host: run `lazy-tests` against the target host.
- For a quick single-host smoke test, set `<TARGET_HOST>` to `127.0.0.1`.

## Important behavior

`lazy-tests` client is expected to succeed here.

This case is not about client-side errors. It is about the server intentionally leaking accepted sockets after the client closes, so `CLOSE_WAIT` appears on the server side.

Use the `half-close-hold` scenario for stable `CLOSE_WAIT` levels. Pure close churn can peak then decay when client-side half-closed sockets are reaped by the OS.

## Setup

Terminal 1 on the target host:

```bash
go run ./labs/high-conntrack/server -listen :18080 -leak-close-wait -leak-limit 3000 -log-every 1
```

Terminal 2 on the client host:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-500.yaml --target-host <TARGET_HOST> --target-port 18080
```

If you need stronger pressure, use:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-pressure.yaml --target-host <TARGET_HOST> --target-port 18080
```

Optional terminal 3 on the target host for direct Linux verification:

While the run is active, verify `CLOSE_WAIT` count directly:

```bash
./scripts/check_close_wait.sh 18080
```

## Expected in holyf-network

- `Connection States`: `CLOSE_WAIT` grows over time.
- `Conntrack`: pressure rises while churn is active.
- `lazy-tests` report can still show high success rate. That is normal for this case.

## Cleanup

- Stop the `labs/high-conntrack/server` process.
