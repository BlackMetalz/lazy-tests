# LT-DB-02 — Redis hold-open heavy

## Goal
Validate long-lived Redis pressure with large active socket count.

## Topology

- Target host: run Redis and holyf-network on the same machine.
- Client host: run `lazy-tests` against the target host.
- For a quick single-host smoke test, set `<TARGET_HOST>` to `127.0.0.1`.

## Prerequisites

- Docker on the target host.

## Setup

Terminal 1 on the target host:

```bash
docker rm -f lazy-tests-redis >/dev/null 2>&1 || true
docker run -d --name lazy-tests-redis \
  -p 6379:6379 \
  redis:7-alpine \
  redis-server --appendonly no --requirepass redis
until docker exec lazy-tests-redis redis-cli -a redis ping; do sleep 1; done
```

Terminal 2 on the client host:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/redis-hold-open-heavy.yaml --target-host <TARGET_HOST> --target-port 6379
```

## Expected in holyf-network

- Stable high `ESTABLISHED` on port `6379`.
- Top Connections aggregates show dominant Redis peer rows.

## Cleanup

```bash
docker rm -f lazy-tests-redis
```
