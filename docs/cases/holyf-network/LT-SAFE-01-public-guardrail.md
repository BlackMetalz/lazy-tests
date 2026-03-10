# LT-SAFE-01 — Public target guardrail

## Goal
Ensure accidental public-target tests are blocked by default.

## Steps

1. Pass a public target through CLI overrides.
2. Run without `--unsafe-public-target`.

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host <PUBLIC_IP_OR_FQDN> --target-port <PUBLIC_PORT>
```

## Expected

- CLI exits with code `1` and guardrail message.
