# LT-TCP-02 — ESTABLISHED saturation (1k hold-open)

## Goal
Hold a large number of active sockets to validate high `ESTABLISHED` visibility.

## Topology

- Target host: run the bundled TCP server and holyf-network on the same machine.
- Client host: run `lazy-tests` against the target host.
- For a quick single-host smoke test, set `<TARGET_HOST>` to `127.0.0.1`.

## Setup

Terminal 1 on the target host:

```bash
go run ./labs/high-conntrack/server -listen :18080 -read-timeout 5m -log-every 1000
```

Terminal 2 on the client host:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml --target-host <TARGET_HOST> --target-port 18080
```

## Expected in holyf-network

- `Connection States`: strong `ESTABLISHED` plateau during run.
- `Top Connections`: one/few peers consume most active connections.

## Cleanup

- Stop the `labs/high-conntrack/server` process.
