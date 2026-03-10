# retransmission lab (Linux netem)

This lab validates retransmission panels/alerts in `holyf-network`.

## Prerequisites

- Linux with `tc` (`iproute2`)
- root/sudo
- choose interface connected to target (example: `eth0`)

## 1) Add packet loss/delay profile

```bash
sudo tc qdisc add dev eth0 root netem delay 80ms 20ms loss 3%
```

## 2) Generate sustained traffic

Option A (simple HTTP long flow):

```bash
curl --http1.1 -L http://speedtest.tele2.net/1GB.zip -o /dev/null
```

Option B (local target with lazy-tests connect storm):

```bash
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host <TARGET_HOST> --target-port <TARGET_PORT>
```

## 3) Observe holyf-network

Expected:

- `Connection States` panel shows retrans metrics rising.
- Retrans percent can pass warn/crit thresholds under loss profile.

## 4) Cleanup netem

```bash
sudo tc qdisc del dev eth0 root
```
