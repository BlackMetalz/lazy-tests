# LT-DB-03 — Postgres hold-open heavy

## Goal
Create sustained Postgres active-connection pressure.

## Topology

- Target host: run Postgres and holyf-network on the same machine.
- Client host: run `lazy-tests` against the target host.
- For a quick single-host smoke test, set `<TARGET_HOST>` to `127.0.0.1`.

## Prerequisites

- Docker on the target host.

## Setup

Terminal 1 on the target host:

```bash
docker rm -f lazy-tests-postgres >/dev/null 2>&1 || true
docker run -d --name lazy-tests-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=postgres \
  -p 5432:5432 \
  postgres:16-alpine
until docker exec lazy-tests-postgres pg_isready -U postgres -d postgres; do sleep 1; done
```

Terminal 2 on the client host:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/postgres-hold-open-heavy.yaml --target-host <TARGET_HOST> --target-port 5432
```

## Expected in holyf-network

- High `ESTABLISHED` on port `5432`.
- Conntrack usage remains elevated while run is active.

## Cleanup

```bash
docker rm -f lazy-tests-postgres
```
