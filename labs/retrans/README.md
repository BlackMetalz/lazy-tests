# retransmission lab (Linux netem)

This lab validates retransmission panels/alerts in `holyf-network`.

## Prerequisites

- Linux with `tc` (`iproute2`)
- root/sudo
- choose the egress interface on the observed host that sends traffic back to the client

Find it with:

```bash
ip route get <CLIENT_IP>
```

## 1) Start target server

```bash
go run ./labs/high-conntrack/server -listen :18080 -hold 300ms -write-delay 50ms -stats-every 1s -log-every 1000
```

## 2) Verify baseline before shaping

```bash
go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 20 -concurrency 1 -timeout 5s -read-reply
```

If baseline is not almost fully successful, fix reachability first.

## 3) Add packet loss/delay profile

```bash
sudo tc qdisc add dev <TARGET_INTERFACE> root netem delay 80ms 20ms loss 3%
```

## 4) Generate sustained traffic

Start moderate first:

```bash
go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 5000 -concurrency 100 -timeout 5s -read-reply -delay 5ms
```

Then scale up if needed:

```bash
go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 20000 -concurrency 400 -timeout 5s -read-reply -delay 2ms
```

## 5) Observe holyf-network

Expected:

- `Connection States` panel shows retrans metrics rising.
- `LOW SAMPLE` is still valid until retrans sampling gates are met.
- With current holyf thresholds, scoring starts only once `ESTABLISHED >= 20` and `OutSegs/sec >= 60`.
- Warn/crit can appear only after those sample gates are met.

If you keep seeing `LOW SAMPLE`, increase connection lifetime:

- Raise server `-hold` to `500ms`
- Optionally add client `-hold 200ms`

## 6) Cleanup netem

```bash
sudo tc qdisc del dev <TARGET_INTERFACE> root
```
