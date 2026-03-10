# LT-DB-01 — MySQL connect storm

## Goal
Stress MySQL connect/auth path with high churn.

## Topology

- Target host: run MySQL and holyf-network on the same machine.
- Client host: run `lazy-tests` against the target host.
- For a quick single-host smoke test, set `<TARGET_HOST>` to `127.0.0.1`.

## Prerequisites

- Docker on the target host.

## Setup

Terminal 1 on the target host:

```bash
docker rm -f lazy-tests-mysql >/dev/null 2>&1 || true
docker run -d --name lazy-tests-mysql \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_ROOT_HOST=% \
  -e MYSQL_DATABASE=mysql \
  -p 3306:3306 \
  mysql:8.0
until docker exec lazy-tests-mysql mysqladmin ping -h 127.0.0.1 -proot --silent; do sleep 1; done
```

Terminal 2 on the client host:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/mysql-connect-storm.yaml --target-host <TARGET_HOST> --target-port 3306
```

## Expected in holyf-network

- DB port (`3306`) rises in Top Connections.
- Conntrack pressure increases during burst windows.

## Cleanup

```bash
docker rm -f lazy-tests-mysql
```
