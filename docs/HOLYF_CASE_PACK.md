# holyf-network Case Pack (one case per file)

This is the focused case pack for validating `holyf-network` using `lazy-tests`.

## Native Cases

- [LT-TCP-01 — TIME_WAIT spike](cases/holyf-network/LT-TCP-01-timewait.md)
- [LT-TCP-02 — ESTABLISHED saturation](cases/holyf-network/LT-TCP-02-established.md)
- [LT-TCP-03 — Conntrack storm](cases/holyf-network/LT-TCP-03-conntrack-storm.md)
- [LT-DB-01 — MySQL connect storm](cases/holyf-network/LT-DB-01-mysql-storm.md)
- [LT-DB-02 — Redis hold-open heavy](cases/holyf-network/LT-DB-02-redis-hold.md)
- [LT-DB-03 — Postgres hold-open heavy](cases/holyf-network/LT-DB-03-postgres-hold.md)
- [LT-SAFE-01 — Public target guardrail](cases/holyf-network/LT-SAFE-01-public-guardrail.md)
- [LT-OBS-01 — Prometheus export](cases/holyf-network/LT-OBS-01-prometheus-export.md)

## Hybrid Cases

- [LAB-CW-01 — CLOSE_WAIT accumulation](cases/holyf-network/LAB-CW-01-close-wait.md)
- [LAB-NAT-01 — Docker NAT (`ct/nat`)](cases/holyf-network/LAB-NAT-01-docker-nat.md)
- [LAB-RTR-01 — Retrans under netem](cases/holyf-network/LAB-RTR-01-retrans.md)
- [LAB-MIT-01 — holyf kill/block convergence](cases/holyf-network/LAB-MIT-01-kill-block.md)

## Suggested Execution Order

1. LT-TCP-01
2. LT-TCP-03
3. LT-DB-01
4. LAB-CW-01
5. LAB-NAT-01
6. LAB-RTR-01
7. LAB-MIT-01

## Notes

- Native cases run directly with `lazy-tests`.
- Hybrid cases require extra topology/tools (`labs/` folder) for states that cannot be reliably forced by connect-only patterns.
