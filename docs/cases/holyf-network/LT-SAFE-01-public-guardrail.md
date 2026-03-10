# LT-SAFE-01 — Public target guardrail

## Goal
Ensure accidental public-target tests are blocked by default.

## Setup

No target setup is required. This case should fail during the private-network guardrail check before `lazy-tests` opens any connection.

## Command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host 1.1.1.1 --target-port 80
```

## Expected

- CLI exits with code `1` and guardrail message.
- No target traffic should be sent because the run is rejected before dialing.
