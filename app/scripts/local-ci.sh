#!/usr/bin/env bash
# Local, zero-cost CI gate for app/ -- no GitHub Actions. Ported from
# prototype/go/scripts/local-ci.sh (roadmap.md item 12,
# benchmarks/025-architecture-worldclass-audit.md), same reasoning: this repo
# runs on a free GitHub plan with its Actions minutes already exhausted
# (owner decision, 2026-08-29).
#
# Runs the real conformance suite (make migrate-up && make seed && make build,
# start the server, ./conformance/run.sh) against a throwaway isolated Postgres
# schema on a throwaway server port -- never touches app/'s own persistent dev
# database (menata_app) or its own dev deployment (see DEVELOPMENT.md's
# "database setup" section, and prototype/go/CLAUDE.md's "isolated-schema
# test server" pattern, which this automates). Safe to run repeatedly and
# tears itself down on exit, success or failure.
#
# Usage: ./scripts/local-ci.sh          (from app/, or via `make local-ci`)
#        LOCAL_CI_PORT=4098 ./scripts/local-ci.sh   (if 4099 is taken)

set -euo pipefail
cd "$(dirname "$0")/.."   # app/

SCHEMA="ci_$(date +%s)_$$"
PORT="${LOCAL_CI_PORT:-4099}"
BASE_DB_URL="${DATABASE_URL:-$(grep DATABASE_URL .env 2>/dev/null | cut -d= -f2-)}"
BASE_DB_URL="${BASE_DB_URL:-postgres://postgres:password@localhost:5432/menata_app?sslmode=disable}"
case "$BASE_DB_URL" in
  *\?*) TEST_DB_URL="${BASE_DB_URL}&options=-csearch_path%3D${SCHEMA}" ;;
  *)    TEST_DB_URL="${BASE_DB_URL}?options=-csearch_path%3D${SCHEMA}" ;;
esac

LOG_DIR="$(mktemp -d)"
SERVER_PID=""
mkdir -p uploads
UPLOADS_BEFORE="$LOG_DIR/uploads-before.txt"
ls uploads/ > "$UPLOADS_BEFORE" 2>/dev/null || true

cleanup() {
  local exit_code=$?
  if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  psql "$BASE_DB_URL" -c "DROP SCHEMA IF EXISTS \"$SCHEMA\" CASCADE;" >/dev/null 2>&1 || true
  # Uploaded files are named by content hash, not by schema -- can't glob them
  # by name. Instead, delete only files that weren't already in uploads/
  # before this run started (a before/after snapshot diff).
  if [ -f "$UPLOADS_BEFORE" ]; then
    comm -13 <(sort "$UPLOADS_BEFORE") <(ls uploads/ 2>/dev/null | sort) \
      | while IFS= read -r f; do rm -f "uploads/$f"; done
  fi
  if [ "$exit_code" -ne 0 ]; then
    echo ""
    echo "==> local-ci FAILED (exit $exit_code) -- logs kept at $LOG_DIR"
  else
    rm -rf "$LOG_DIR"
  fi
  exit "$exit_code"
}
trap cleanup EXIT INT TERM

echo "==> quality gates (error-leak scan, handler LOC + complexity ratchets)"
./scripts/check-quality-gates.sh

echo "==> local-ci: isolated schema $SCHEMA, throwaway port $PORT"

if ss -ltn 2>/dev/null | grep -q ":$PORT "; then
  echo "local-ci: port $PORT already in use -- set LOCAL_CI_PORT to a free port" >&2
  exit 1
fi

psql "$BASE_DB_URL" -c "CREATE SCHEMA \"$SCHEMA\";" >/dev/null

echo "==> migrate-up"
DB_URL="$TEST_DB_URL" make migrate-up > "$LOG_DIR/migrate.log" 2>&1

echo "==> seed"
DB_URL="$TEST_DB_URL" make seed > "$LOG_DIR/seed.log" 2>&1

echo "==> build"
make build > "$LOG_DIR/build.log" 2>&1

echo "==> starting throwaway server on :$PORT"
DATABASE_URL="$TEST_DB_URL" PORT="$PORT" SECURE_COOKIES=false ./bin/server > "$LOG_DIR/server.log" 2>&1 &
SERVER_PID=$!

healthy=false
for _ in $(seq 1 30); do
  if curl -sf "http://localhost:$PORT/health" >/dev/null 2>&1; then
    healthy=true
    break
  fi
  sleep 0.5
done
if [ "$healthy" != true ]; then
  echo "local-ci: server never became healthy -- server log:" >&2
  cat "$LOG_DIR/server.log" >&2
  exit 1
fi

echo "==> conformance"
BASE_URL="http://localhost:$PORT" DATABASE_URL="$TEST_DB_URL" ./conformance/run.sh

echo "==> local-ci PASSED"
