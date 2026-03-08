# docker NAT visibility lab

This lab is for validating `holyf-network` NAT/conntrack merge path (`ct/nat`).

## 1) Start docker target

```bash
docker compose -f labs/nat/docker-compose.yml up -d
```

## 2) Verify target

```bash
curl -I http://127.0.0.1:18080
```

## 3) Run lazy-tests NAT scenario

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-docker-nat.yaml
```

## 4) Watch holyf-network

Expected:

- `Top Connections` can include NAT-derived visibility (`ct/nat`) depending on host/container mapping.
- `Conntrack` usage/new rate increases during run.

## 5) Cleanup

```bash
docker compose -f labs/nat/docker-compose.yml down
```
