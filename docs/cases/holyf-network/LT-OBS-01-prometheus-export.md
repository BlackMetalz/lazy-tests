# LT-OBS-01 — Prometheus export during run

## Goal
Verify lazy-tests exposes `/metrics` while workload is active.

## Steps

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml
```

In another terminal, curl the endpoint shown in summary (for example):

```bash
curl -s http://127.0.0.1:2112/metrics | rg lazy_tests_
```

## Expected

- Endpoint responds during active run.
- Counters/histograms include `lazy_tests_*` metrics.
