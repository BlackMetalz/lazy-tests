# LAB-NAT-01 — Docker NAT visibility (`ct/nat`)

## Goal
Validate NAT-derived visibility path in holyf.

## Setup

```bash
docker compose -f labs/nat/docker-compose.yml up -d
curl -I http://127.0.0.1:18080
```

## Traffic command

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-docker-nat.yaml --target-host 127.0.0.1 --target-port 18080
```

## Expected in holyf-network

- `Top Connections` may include `ct/nat` rows.
- `Conntrack` usage/new rate rises during run.

## Cleanup

```bash
docker compose -f labs/nat/docker-compose.yml down
```
