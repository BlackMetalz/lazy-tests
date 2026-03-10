# LT-OBS-01 — Prometheus export during run

## Goal
Verify lazy-tests exposes `/metrics` while workload is active.

## Topology

- Target host: run the bundled TCP server.
- Client host: run `lazy-tests` and query the Prometheus endpoint from the same machine.
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

Terminal 3 on the client host while the run is active:

```bash
curl -s http://127.0.0.1:2112/metrics | rg lazy_tests_
```

## Expected

- Endpoint responds during active run.
- Counters/histograms include `lazy_tests_*` metrics.

## Cleanup

- Stop the `labs/high-conntrack/server` process.
