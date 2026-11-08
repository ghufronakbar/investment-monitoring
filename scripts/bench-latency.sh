#!/usr/bin/env sh
# Measures end-to-end realtime latency on a throwaway database: creates and
# migrates it, starts the API and the outbox worker against it, runs
# cmd/latencybench, then stops everything and drops the database.
#
#   scripts/bench-latency.sh listen [latencybench flags...]   # LISTEN/NOTIFY + polling fallback
#   scripts/bench-latency.sh poll   [latencybench flags...]   # polling every 5s only (previous behaviour)
#
# Requires Redis and PostgreSQL running locally. Results go to bench/.
set -eu

MODE="${1:-listen}"
[ $# -gt 0 ] && shift

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

case "$MODE" in
	listen) WORKER_FLAGS="-listen=true -poll-interval=5s" ;;
	poll) WORKER_FLAGS="-listen=false -poll-interval=5s" ;;
	*) echo "mode must be listen or poll" >&2; exit 1 ;;
esac

PG_ADMIN_URL="${PG_ADMIN_URL:-postgres://postgres:@localhost:5432/postgres?sslmode=disable}"
DB_NAME="investment_monitoring_bench_$(date +%s)"
URL_BASE="${PG_ADMIN_URL%%\?*}"
URL_QUERY="${PG_ADMIN_URL#"$URL_BASE"}"
DATABASE_URL="${URL_BASE%/*}/$DB_NAME$URL_QUERY"
export DATABASE_URL

BIN_DIR="$ROOT_DIR/tmp/bench"
mkdir -p "$BIN_DIR" "$ROOT_DIR/bench"
API_PID=""
WORKER_PID=""

cleanup() {
	[ -n "$WORKER_PID" ] && kill "$WORKER_PID" 2>/dev/null || true
	[ -n "$API_PID" ] && kill "$API_PID" 2>/dev/null || true
	wait 2>/dev/null || true
	psql "$PG_ADMIN_URL" -q -c "DROP DATABASE IF EXISTS \"$DB_NAME\" WITH (FORCE)" >/dev/null
}
trap cleanup EXIT

go build -o "$BIN_DIR/api" ./cmd/api
go build -o "$BIN_DIR/worker" ./cmd/worker
go build -o "$BIN_DIR/latencybench" ./cmd/latencybench

psql "$PG_ADMIN_URL" -q -c "CREATE DATABASE \"$DB_NAME\"" >/dev/null
goose -dir migrations postgres "$DATABASE_URL" up >/dev/null

STAMP="$(date +%Y%m%d-%H%M%S)"
"$BIN_DIR/api" >"bench/api-$MODE-$STAMP.log" 2>&1 &
API_PID=$!
# shellcheck disable=SC2086
"$BIN_DIR/worker" $WORKER_FLAGS >"bench/worker-$MODE-$STAMP.log" 2>&1 &
WORKER_PID=$!

i=0
until curl -s -o /dev/null http://localhost:7777/; do
	i=$((i + 1))
	[ "$i" -gt 50 ] && { echo "API did not start" >&2; exit 1; }
	sleep 0.2
done

"$BIN_DIR/latencybench" -label "$MODE" -out "bench/latency-$MODE-$STAMP.csv" "$@" | tee "bench/latency-$MODE-$STAMP.txt"
