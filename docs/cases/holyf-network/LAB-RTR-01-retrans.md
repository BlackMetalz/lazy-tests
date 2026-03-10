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
- Find the interface carrying traffic from the target host back to the client host. `tc qdisc ... root` only affects egress on the selected device.

## Pick the correct interface

On the target host, replace `<CLIENT_IP>` with the IP of the traffic generator and note the interface shown after `dev`:

```bash
ip route get <CLIENT_IP>
```

Use that interface instead of the `eth0` example below.

## Setup

Terminal 1 on the target host:

```bash
go run ./labs/high-conntrack/server -listen :18080 -hold 300ms -write-delay 50ms -stats-every 1s -log-every 1000
```

Terminal 2 on the client host. Verify the path before shaping traffic:

```bash
go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 20 -concurrency 1 -timeout 5s -read-reply
```

If this baseline probe does not show near-100% success, stop here and fix basic reachability first. Do not enable `netem` until the unshaped path is healthy.

Terminal 3 on the target host:

```bash
sudo tc qdisc add dev <TARGET_INTERFACE> root netem delay 80ms 20ms loss 3%
```

scale up on client:

```bash
go run ./labs/high-conntrack/client -target <TARGET_HOST>:18080 -total 20000 -concurrency 400 -timeout 5s -read-reply -delay 2ms
```

## Expected in holyf-network

- Retrans rate and retrans percent increase.
- Health strip can stay `LOW SAMPLE` until holyf has enough data to score retransmissions.
- With the current holyf thresholds, that means at least `ESTABLISHED >= 20` and `OutSegs/sec >= 60`.
- Only after those sample gates are met can the strip move to `warn` or `crit`.
- Target-host server logs should show accepts increasing while the client still reports mostly successful request/reply exchanges.

## Tuning notes

- If retrans grows but holyf still shows `LOW SAMPLE`, keep the impairment profile and increase connection lifetime first.
- The simplest knob is target-host server hold time, for example `-hold 500ms`.
- If you still need more overlap, also add a small client-side hold such as `-hold 200ms`.

## Cleanup

Terminal 3 on the target host:

```bash
sudo tc qdisc del dev <TARGET_INTERFACE> root
```

Then stop the `labs/high-conntrack/server` process.
