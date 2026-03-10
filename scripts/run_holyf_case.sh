#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  cat <<USAGE
Usage:
  scripts/run_holyf_case.sh <CASE_ID> [--run]

Examples:
  scripts/run_holyf_case.sh LT-TCP-01
  scripts/run_holyf_case.sh LT-TCP-01 --run
USAGE
  exit 1
fi

CASE_ID="$1"
MODE="${2:---print}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

command_block() {
  case "$CASE_ID" in
    LT-TCP-01)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml
CMD
      ;;
    LT-TCP-02)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml
CMD
      ;;
    LT-TCP-03)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml
CMD
      ;;
    LT-DB-01)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/mysql-connect-storm.yaml
CMD
      ;;
    LT-DB-02)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/redis-hold-open-heavy.yaml
CMD
      ;;
    LT-DB-03)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/postgres-hold-open-heavy.yaml
CMD
      ;;
    LT-OBS-01)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml
# while running:
#   curl -s http://127.0.0.1:2112/metrics | rg lazy_tests_
CMD
      ;;
    LAB-CW-01)
      cat <<'CMD'
# terminal 1
go run ./labs/high-conntrack/server -listen :18080 -leak-close-wait -leak-limit 3000 -log-every 1

# terminal 2
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-500.yaml --target-host <SERVER_IP>

# optional heavy pressure:
# go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-pressure.yaml --target-host <SERVER_IP>

# terminal 3 (Linux verification)
./scripts/check_close_wait.sh 18080

# note:
# success from lazy-tests is expected here; the target signal is server-side CLOSE_WAIT
CMD
      ;;
    LAB-NAT-01)
      cat <<'CMD'
docker compose -f labs/nat/docker-compose.yml up -d
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-docker-nat.yaml
# cleanup:
# docker compose -f labs/nat/docker-compose.yml down
CMD
      ;;
    LAB-RTR-01)
      cat <<'CMD'
sudo tc qdisc add dev eth0 root netem delay 80ms 20ms loss 3%
curl --http1.1 -L http://speedtest.tele2.net/1GB.zip -o /dev/null
sudo tc qdisc del dev eth0 root
CMD
      ;;
    LAB-MIT-01)
      cat <<'CMD'
# terminal 1
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml

# terminal 2 (holyf-network TUI)
# trigger k/Enter on hot peer row and test:
# - minutes > 0 (timed block)
# - minutes = 0 (kill-only)
CMD
      ;;
    *)
      echo "Unknown case id: $CASE_ID" >&2
      exit 2
      ;;
  esac
}

CMDS="$(command_block)"

if [[ "$MODE" == "--run" ]]; then
  echo "Running case $CASE_ID"
  bash -lc "$CMDS"
else
  echo "Case $CASE_ID commands:"
  echo "$CMDS"
fi
