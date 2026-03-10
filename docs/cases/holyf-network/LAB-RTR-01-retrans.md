# LAB-RTR-01 — Retransmission under netem

## Goal
Create packet loss/delay to exercise retransmission metrics and thresholds.

## Topology

- Target host: run the bundled TCP server, holyf-network, and apply `tc netem` on the interface that faces the client host.
- Client host: run the bundled lab client to generate request/reply traffic.
- For a quick same-host smoke test you can use `lo`, but a separate client host produces clearer retransmission signals.

## Prerequisites

- Linux with `tc` (`iproute2`) on the target host.
- Root or `sudo` on the target host.
- Pick the interface carrying traffic between client and target, for example `eth0`.

## Setup

Terminal 1 on the target host:

```bash
go run ./labs/high-conntrack/server -listen :18080 -write-delay 50ms -log-every 1000
```

Terminal 2 on the target host:

```bash
sudo tc qdisc add dev eth0 root netem delay 80ms 20ms loss 3%
```

Terminal 3 on the client host:

```bash
go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 20000 -concurrency 400 -timeout 3s -read-reply -delay 2ms
```

## Expected in holyf-network

- Retrans rate and retrans percent increase.
- Depending on thresholds, health strip can move to warn/crit.

## Cleanup

Terminal 2 on the target host:

```bash
sudo tc qdisc del dev eth0 root
```

Then stop the `labs/high-conntrack/server` process.
