#!/usr/bin/env sh
# Runs the integration tests against a throwaway PostgreSQL database that is
# created, migrated with goose, and dropped again. Extra arguments are passed
# to `go test` (e.g. -run TestFSM -v).
#
#   PG_ADMIN_URL  server to create the database on
#                 (default postgres://postgres:@localhost:5432/postgres?sslmode=disable)
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

PG_ADMIN_URL="${PG_ADMIN_URL:-postgres://postgres:@localhost:5432/postgres?sslmode=disable}"
DB_NAME="investment_monitoring_it_$(date +%s)"
URL_BASE="${PG_ADMIN_URL%%\?*}"
URL_QUERY="${PG_ADMIN_URL#"$URL_BASE"}"
TEST_DATABASE_URL="${URL_BASE%/*}/$DB_NAME$URL_QUERY"
export TEST_DATABASE_URL

# The tests write fixtures without cleanup; never let them reach a real database.
case "$TEST_DATABASE_URL" in
	*"/$DB_NAME"*) ;;
	*) echo "refusing to run: test URL does not point at $DB_NAME" >&2; exit 1 ;;
esac

cleanup() {
	psql "$PG_ADMIN_URL" -q -c "DROP DATABASE IF EXISTS \"$DB_NAME\" WITH (FORCE)" >/dev/null
}
trap cleanup EXIT

psql "$PG_ADMIN_URL" -q -c "CREATE DATABASE \"$DB_NAME\"" >/dev/null
goose -dir "$ROOT_DIR/migrations" postgres "$TEST_DATABASE_URL" up >/dev/null

cd "$ROOT_DIR"
go test -tags integration -count=1 "$@" ./internal/integration/...
