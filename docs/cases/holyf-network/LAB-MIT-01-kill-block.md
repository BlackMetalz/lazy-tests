# LAB-MIT-01 — holyf kill/block convergence under storm

## Goal
Validate holyf active mitigation path (`k/Enter`) while traffic storm is running.

## Steps

1. Start storm traffic:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml
```

2. In holyf live TUI:
- select hot peer row in Top Connections.
- trigger `k/Enter`.
- test both modes:
  - `minutes > 0` (timed block)
  - `minutes = 0` (kill-only)

## Expected in holyf-network

- Target peer active connections drop after action.
- Under race windows, bounded partial convergence may appear (expected behavior).
