# high-conntrack lab

This lab reproduces two important `holyf-network` cases that pure connect scenarios usually miss:

- conntrack pressure spike
- `CLOSE_WAIT` accumulation

## 1) Start server (normal mode)

```bash
go run ./labs/high-conntrack/server -listen :18080
```

## 2) Generate conntrack spike (standalone client)

```bash
go run ./labs/high-conntrack/client -target 127.0.0.1:18080 -total 200000 -concurrency 1000 -timeout 1s
```

## 3) Generate `CLOSE_WAIT` deliberately

Start server with leak mode:

```bash
go run ./labs/high-conntrack/server -listen :18080 -leak-close-wait -leak-limit 3000 -log-every 1
```

Then run lazy-tests scenario:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-500.yaml --target-host <SERVER_IP>
```

For heavier pressure:

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-pressure.yaml --target-host <SERVER_IP>
```

Important:

- A successful `lazy-tests` run is expected here.
- The signal is server-side `CLOSE_WAIT`, not client-side failure.
- In leak mode, default `-leak-read-timeout=0` waits indefinitely for peer FIN so sockets are less likely to fall back into `LAST_ACK` due timeout.
- Direct check on Linux:

```bash
./scripts/check_close_wait.sh 18080
```

Or use standalone client with slow ramp:

```bash
go run ./labs/high-conntrack/client -target 127.0.0.1:18080 -total 2000 -concurrency 250 -hold 5s -delay 10ms
```

## Expected signals in holyf-network

- `Connection States`: `CLOSE_WAIT` increases in leak mode.
- `Conntrack`: `used` and `new/sec` increase under storm.
- `Top Connections`: peer/port rows dominate by count.
