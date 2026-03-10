# LT-TCP-01 — TIME_WAIT spike (~500)

## Goal
Create short-lived TCP churn to surface `TIME_WAIT` behavior in holyf.

## Topology

- Target host: run the bundled TCP server and holyf-network on the same machine.
- Client host: run `lazy-tests` against the target host.
- For a quick single-host smoke test, set `<TARGET_HOST>` to `127.0.0.1`.

## Setup

Terminal 1 on the target host:

```bash
go run ./labs/high-conntrack/server -listen :18080 -read-timeout 200ms -log-every 1000
```

Terminal 2 on the client host:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml --target-host <TARGET_HOST> --target-port 18080
```

## Expected in holyf-network

- `Connection States`: `TIME_WAIT` increases clearly.
- `Top Connections`: target port row becomes dominant by connection count.

## Cleanup

- Stop the `labs/high-conntrack/server` process.
