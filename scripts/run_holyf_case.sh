#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  cat <<USAGE
Usage:
  scripts/run_holyf_case.sh <CASE_ID> [--run]

Examples:
  scripts/run_holyf_case.sh LT-TCP-01
  scripts/run_holyf_case.sh LT-TCP-01 --run
  TARGET_HOST=172.25.110.76 TARGET_PORT=8080 scripts/run_holyf_case.sh LT-TCP-01 --run
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
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-timewait-500.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-8080}"
CMD
      ;;
    LT-TCP-02)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-8080}"
CMD
      ;;
    LT-TCP-03)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-8080}"
CMD
      ;;
    LT-DB-01)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/mysql-connect-storm.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-3306}"
CMD
      ;;
    LT-DB-02)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/redis-hold-open-heavy.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-6379}"
CMD
      ;;
    LT-DB-03)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/postgres-hold-open-heavy.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-5432}"
CMD
      ;;
    LT-OBS-01)
      cat <<'CMD'
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-8080}"
# while running:
#   curl -s http://127.0.0.1:2112/metrics | rg lazy_tests_
CMD
      ;;
    LAB-CW-01)
      cat <<'CMD'
# terminal 1
go run ./labs/high-conntrack/server -listen :18080 -leak-close-wait -leak-limit 3000 -log-every 1

# terminal 2
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-500.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-18080}"

# optional heavy pressure:
# go run ./cmd/lazy-tests run -f examples/scenarios/tcp-close-wait-pressure.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-18080}"

# terminal 3 (Linux verification)
./scripts/check_close_wait.sh 18080

# note:
# success from lazy-tests is expected here; the target signal is server-side CLOSE_WAIT
CMD
      ;;
    LAB-NAT-01)
      cat <<'CMD'
docker compose -f labs/nat/docker-compose.yml up -d
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-docker-nat.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-18080}"
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
# terminal 1 (target host)
go run ./labs/high-conntrack/server -listen :18080 -read-timeout 5m -log-every 1000

# terminal 2 (client host)
# timed block validation under churn pressure
go run ./cmd/lazy-tests run -f examples/scenarios/tcp-conntrack-storm.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-18080}"

# kill-only validation with clearer active connection drop
# go run ./cmd/lazy-tests run -f examples/scenarios/tcp-established-1k.yaml --target-host "${TARGET_HOST:-127.0.0.1}" --target-port "${TARGET_PORT:-18080}"

# terminal 3 (holyf-network TUI on target host)
# select the hot peer row created by the client host, then test:
# - minutes > 0 with the churn profile above
# - minutes = 0 with the established profile above
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
