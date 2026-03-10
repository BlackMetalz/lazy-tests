# LAB-NAT-01 — Docker NAT visibility (`ct/nat`)

## Goal
Validate NAT-derived visibility path in holyf.

## Topology

- Run this case on the Docker host that also runs holyf-network.
- The simplest path is to run both Docker and `lazy-tests` on the same machine so host-side NAT rules are guaranteed to be involved.

## Prerequisites

- Docker with the Compose plugin on the host.

## Setup

Terminal 1 on the Docker host:

```bash
docker compose -f labs/nat/docker-compose.yml up -d
curl -I http://127.0.0.1:18080
```

Terminal 2 on the same host:

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
