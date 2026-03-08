# LT-SAFE-01 — Public target guardrail

## Goal
Ensure accidental public-target tests are blocked by default.

## Steps

1. Set scenario host to a public IP/FQDN.
2. Run without `--unsafe-public-target`.

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml
```

## Expected

- CLI exits with code `1` and guardrail message.
