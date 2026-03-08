# LAB-RTR-01 — Retransmission under netem

## Goal
Create packet loss/delay to exercise retransmission metrics and thresholds.

## Steps

```bash
sudo tc qdisc add dev eth0 root netem delay 80ms 20ms loss 3%
curl --http1.1 -L http://speedtest.tele2.net/1GB.zip -o /dev/null
sudo tc qdisc del dev eth0 root
```

## Expected in holyf-network

- Retrans rate and retrans percent increase.
- Depending on thresholds, health strip can move to warn/crit.
