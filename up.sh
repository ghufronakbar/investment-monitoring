#!/usr/bin/env sh
set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ -f "$SCRIPT_DIR/.env" ]; then
	set -a
	. "$SCRIPT_DIR/.env"
	set +a
fi

: "${DATABASE_URL:?DATABASE_URL is required. Set it in .env or environment.}"

goose -dir "$SCRIPT_DIR/migrations" postgres "$DATABASE_URL" up
