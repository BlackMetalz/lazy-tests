# LT-TCP-03 — Conntrack storm (TCP churn)

## Goal
Drive high connection churn to push `Conntrack` usage and `new/sec`.

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
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host <TARGET_HOST> --target-port 18080
```

## Expected in holyf-network

- `Conntrack`: `used` and `new/sec` increase sharply.
- `Connection States`: frequent transitions across active/closing states.

## Cleanup

- Stop the `labs/high-conntrack/server` process.
