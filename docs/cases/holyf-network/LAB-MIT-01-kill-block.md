# LAB-MIT-01 — holyf kill/block convergence under storm

## Goal
Validate holyf active mitigation path (`k/Enter`) with a bundled target server and an explicit traffic generator.

## Topology

- Target host: run the bundled TCP server and holyf-network on the same machine.
- Client host: run `lazy-tests` from another private-network machine against the target host.
- Single-host smoke test is possible with `127.0.0.1`, but a real mitigation check should use a separate client host so holyf acts on a remote peer.

## Setup

Terminal 1 on the target host:

```bash
go run ./labs/high-conntrack/server -listen :18080 -read-timeout 5m -log-every 1000
```

Terminal 2 on the client host. Pick the traffic profile that matches the mitigation mode you want to validate:

```bash
# timed block validation under churn pressure
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host <TARGET_HOST> --target-port 18080

# kill-only validation with clearer active connection drop
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml --target-host <TARGET_HOST> --target-port 18080
```

Terminal 3 on the target host:

- Open holyf-network TUI.
- Select the hot peer row created by the client host in `Top Connections`.
- Trigger `k/Enter`.
- Test both modes:
  - `minutes > 0`: use the churn profile above to verify timed block behavior.
  - `minutes = 0`: use the hold-open profile above to verify kill-only behavior.

## Expected in holyf-network

- With `minutes > 0`, new connections from the client peer fall sharply or stop while the timed block is active.
- With `minutes = 0`, current active connections drop, but new attempts can return because no timed block is installed.
- Under race windows, bounded partial convergence may appear (expected behavior).

## Cleanup

- Stop the `labs/high-conntrack/server` process.
