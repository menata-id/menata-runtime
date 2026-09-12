#!/usr/bin/env bash
# composable-runtime-roadmap.md 17j -- real-traffic composability
# benchmark. Mirrors local-ci.sh's own isolated-schema/scratch-port
# throwaway pattern exactly (never the live menata-runtime process --
# app/CLAUDE.md's own standing "Server lifecycle" rule) and adds a
# load-test + /proc CPU/memory sampling phase on top of that same safe
# setup/teardown. Load levels stay modest (see CONCURRENCY/REQUESTS
# below) -- this host is shared with portal-ga3/iaabl2-*/menata-aksi and
# one Postgres instance; this proves real code paths under real (if
# modest) concurrency, not a stress test of a shared host.
#
# Only 3 of composable-runtime-roadmap.md Phase 12's own 9 required
# scenarios have any real live route to run against today -- see this
# script's own scenario comments below for exactly which, and why the
# other 6 are not attempted (named, not silently skipped).
#
# Usage: ./scripts/benchmark-composable.sh
#        BENCH_PORT=4098 ./scripts/benchmark-composable.sh   (if 4097 is taken)

set -euo pipefail
cd "$(dirname "$0")/.."   # app/

SCHEMA="bench_$(date +%s)_$$"
PORT="${BENCH_PORT:-4097}"
CONCURRENCY="${BENCH_CONCURRENCY:-10}"
REQUESTS="${BENCH_REQUESTS:-100}"
# 20 req/s stays under this server's own real per-IP rate limiter
# (cmd/server/ratelimit.go, 30 req/s burst 120) -- found by this
# benchmark's own first real run, which hit it and produced a majority-
# 429 sample. Paced dispatch, not "as fast as possible," is also what a
# real client population actually looks like.
RATE="${BENCH_RATE:-20}"
BASE_DB_URL="${DATABASE_URL:-$(grep DATABASE_URL .env 2>/dev/null | cut -d= -f2-)}"
BASE_DB_URL="${BASE_DB_URL:-postgres://postgres:password@localhost:5432/menata_app?sslmode=disable}"
case "$BASE_DB_URL" in
  *\?*) TEST_DB_URL="${BASE_DB_URL}&options=-csearch_path%3D${SCHEMA}" ;;
  *)    TEST_DB_URL="${BASE_DB_URL}?options=-csearch_path%3D${SCHEMA}" ;;
esac

LOG_DIR="$(mktemp -d)"
SERVER_PID=""
SAMPLER_PID=""
mkdir -p uploads
UPLOADS_BEFORE="$LOG_DIR/uploads-before.txt"
ls uploads/ > "$UPLOADS_BEFORE" 2>/dev/null || true

cleanup() {
  local exit_code=$?
  if [ -n "$SAMPLER_PID" ] && kill -0 "$SAMPLER_PID" 2>/dev/null; then
    kill "$SAMPLER_PID" 2>/dev/null || true
    wait "$SAMPLER_PID" 2>/dev/null || true
  fi
  if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  psql "$BASE_DB_URL" -c "DROP SCHEMA IF EXISTS \"$SCHEMA\" CASCADE;" >/dev/null 2>&1 || true
  if [ -f "$UPLOADS_BEFORE" ]; then
    comm -13 <(sort "$UPLOADS_BEFORE") <(ls uploads/ 2>/dev/null | sort) \
      | while IFS= read -r f; do rm -f "uploads/$f"; done
  fi
  if [ "$exit_code" -ne 0 ]; then
    echo ""
    echo "==> benchmark-composable FAILED (exit $exit_code) -- logs kept at $LOG_DIR"
  else
    echo ""
    echo "==> logs kept at $LOG_DIR (server.log has every composable_benchmark line)"
  fi
  exit "$exit_code"
}
trap cleanup EXIT INT TERM

echo "==> benchmark-composable: isolated schema $SCHEMA, throwaway port $PORT"

if ss -ltn 2>/dev/null | grep -q ":$PORT "; then
  echo "benchmark-composable: port $PORT already in use -- set BENCH_PORT to a free port" >&2
  exit 1
fi

psql "$BASE_DB_URL" -c "CREATE SCHEMA \"$SCHEMA\";" >/dev/null

echo "==> migrate-up"
DB_URL="$TEST_DB_URL" make migrate-up > "$LOG_DIR/migrate.log" 2>&1

echo "==> seed"
DB_URL="$TEST_DB_URL" make seed > "$LOG_DIR/seed.log" 2>&1

echo "==> build"
make build > "$LOG_DIR/build.log" 2>&1
go build -o "$LOG_DIR/loadtest" ./scripts/loadtest

echo "==> starting throwaway server on :$PORT (debug endpoints enabled)"
DATABASE_URL="$TEST_DB_URL" PORT="$PORT" SECURE_COOKIES=false SMTP_HOST="" ENABLE_DEBUG_ENDPOINTS=1 \
  ./bin/server > "$LOG_DIR/server.log" 2>&1 &
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
  echo "benchmark-composable: server never became healthy -- server log:" >&2
  cat "$LOG_DIR/server.log" >&2
  exit 1
fi

login() {
  local email="$1" jar="$2"
  curl -s -c "$jar" -o /dev/null -X POST "http://localhost:$PORT/login" \
    --data-urlencode "email=$email" --data-urlencode "password=password"
}

# Real seeded identities, same accounts/passwords conformance/lib.sh's own
# session_for already uses -- ALICE (Submitter, app_approval), FRANK (HR,
# app_hr, no role at all on app_approval -- denied 403), PM_MEMBER
# (Member, app_project_management).
ALICE_JAR="$LOG_DIR/alice.jar";      login "alice@example.com" "$ALICE_JAR"
FRANK_JAR="$LOG_DIR/frank.jar";      login "hr@example.com" "$FRANK_JAR"
PM_JAR="$LOG_DIR/pm.jar";            login "project.member@example.com" "$PM_JAR"

# A freshly migrated+seeded schema has zero real records for
# mch_approval_document (seeds/004_approval.sql only declares the
# Machine/Fields/Views, the same way every seed file does -- conformance
# tests are what create real Documents, and this script runs none of
# them). Measuring scenario 3 against a genuinely empty list would be a
# real but unrepresentative number (rows_returned=0 always) -- 20 real
# Documents, the same shape conformance/tests/010_case1_3_core.sh's own
# AD_SEQ_DATA already uses, give the cutover route something realistic to
# actually render and time. ALICE_ID scraped from the real Submitted By
# picker exactly like conformance/lib.sh's own user_option_id helper does.
ALICE_ID=$(curl -s -b "$ALICE_JAR" "http://localhost:$PORT/ws_default/mch_approval_document/new" \
  | grep -oE 'value="[a-f0-9-]+">Alice</option>' | head -1 | grep -oE '"[a-f0-9-]+"' | tr -d '"')
ALICE_CSRF=$(curl -s -b "$ALICE_JAR" "http://localhost:$PORT/ws_default/" \
  | grep -oE 'name="csrf_token" value="[^"]*"' | head -1 | sed -E 's/.*value="([^"]*)"/\1/')
for i in $(seq 1 20); do
  curl -s -o /dev/null -b "$ALICE_JAR" -X POST "http://localhost:$PORT/ws_default/mch_approval_document" \
    --data-urlencode "fld_ad_title=Benchmark Document $i" \
    --data-urlencode "fld_ad_document_type=Report" \
    --data-urlencode "fld_ad_file=bench$i.pdf" \
    --data-urlencode "fld_ad_submitted_by=$ALICE_ID" \
    --data-urlencode "fld_ad_approval_mode=Sequential" \
    --data-urlencode "csrf_token=$ALICE_CSRF"
done

# /proc CPU/memory sampler -- one line per ~200ms to samples.tsv
# (epoch-seconds, VmRSS kB, utime+stime jiffies) for as long as the
# server lives; killed by cleanup() on exit. No server code
# instrumentation needed -- $SERVER_PID is the same PID local-ci.sh's own
# pattern already captures.
HZ="$(getconf CLK_TCK)"
(
  while kill -0 "$SERVER_PID" 2>/dev/null; do
    rss=$(awk '/VmRSS/{print $2}' "/proc/$SERVER_PID/status" 2>/dev/null || echo 0)
    ticks=$(awk '{print $14+$15}' "/proc/$SERVER_PID/stat" 2>/dev/null || echo 0)
    echo -e "$(date +%s.%N)\t$rss\t$ticks" >> "$LOG_DIR/samples.tsv"
    sleep 0.2
  done
) &
SAMPLER_PID=$!

# run_scenario <name> <url> <cookiejar> -- fires the load driver, then
# reports this run's own composable_benchmark log lines (structural
# facts: logical/DAG nodes, naive-vs-dedup query count, execution width,
# real measured data/planner/render time, rows returned) alongside the
# client-observed latency percentiles the driver itself just measured.
run_scenario() {
  local name="$1" url="$2" jar="$3"
  echo ""
  echo "=== $name ==="
  echo "    $url"
  before_lines=$(wc -l < "$LOG_DIR/server.log" 2>/dev/null || echo 0)
  "$LOG_DIR/loadtest" -url "$url" -cookiejar "$jar" -concurrency "$CONCURRENCY" -requests "$REQUESTS" -rate "$RATE" \
    | tee "$LOG_DIR/loadtest-$name.json"
  tail -n "+$((before_lines + 1))" "$LOG_DIR/server.log" | grep '"msg":"composable_benchmark"' \
    > "$LOG_DIR/bench-$name.jsonl" || true
  echo "    structural + server-side timing facts (this run's own composable_benchmark lines):"
  python3 -c "
import json, sys
path = sys.argv[1]
try:
    lines = [json.loads(l) for l in open(path) if l.strip()]
except FileNotFoundError:
    lines = []
if not lines:
    print('    (none captured -- route may not run the composable path, e.g. a 403 short-circuits before it)')
else:
    last = lines[-1]
    n = len(lines)
    avg = lambda k: sum(l[k] for l in lines) / n
    print(f'    logical_nodes={last[\"logical_nodes\"]} dag_nodes={last[\"dag_nodes\"]} naive_query_count={last[\"naive_query_count\"]} dedup_query_count={last[\"dedup_query_count\"]} execution_width={last[\"execution_width\"]}')
    print(f'    avg over {n} requests -- data_time_ms={avg(\"data_time_ms\"):.2f} planner_time_ms={avg(\"planner_time_ms\"):.2f} render_time_ms={avg(\"render_time_ms\"):.2f} rows_returned={avg(\"rows_returned\"):.1f}')
" "$LOG_DIR/bench-$name.jsonl"
}

echo ""
echo "############################################################"
echo "# Scenario 1 (1 component -> 1 Dataset): PM board preview"
echo "############################################################"
run_scenario "scenario1" "http://localhost:$PORT/ws_default/mch_pm_card/composable-preview" "$PM_JAR"

echo ""
echo "############################################################"
echo "# Scenario 3 (10 components -> 3 shared Datasets, dedup"
echo "# already proven structurally): Document Approval's own"
echo "# real cutover route (17g) -- measured under load for the"
echo "# first time, not just a single request."
echo "############################################################"
run_scenario "scenario3" "http://localhost:$PORT/ws_default/mch_approval_document" "$ALICE_JAR"

echo ""
echo "############################################################"
echo "# Scenario 7 (same request, different security scope) --"
echo "# honest substitute: no role on this Machine has different"
echo "# HiddenFields to compare (17g's own finding), so this"
echo "# compares the SAME url's own 403-denied latency (FRANK, no"
echo "# role on app_approval) against scenario 3's own allowed"
echo "# latency above -- \"what does the security check itself"
echo "# cost,\" not \"do two effective Datasets cost the same.\""
echo "############################################################"
run_scenario "scenario7_denied" "http://localhost:$PORT/ws_default/mch_approval_document" "$FRANK_JAR"

echo ""
echo "############################################################"
echo "# Scenarios 2, 4, 5, 6, 8, 9 -- SKIP, no live route exists"
echo "# today (named, not silently dropped, same posture Phase 12's"
echo "# own scenarios 5/6 already established):"
echo "#   2 (10 independent Datasets), 4 (1 Dataset -> multiple"
echo "#   projections): no live page composes that shape yet."
echo "#   5 (mixed OLTP+analytics), 6 (100 concurrent workspaces):"
echo "#   no analytics-pool/multi-workspace-concurrency distinction"
echo "#   exists in this runtime."
echo "#   8 (nested Project board): no live nested composed page for"
echo "#   Project Management yet (17h/17i are a single Board view)."
echo "#   9 (Approval detail + stepper): Detail-page lowering is"
echo "#   still incomplete (CR-06's own open gap)."
echo "############################################################"

echo ""
echo "=== DB pool stats (opt-in /debug/pool-stats, sampled once, post-load) ==="
curl -s "http://localhost:$PORT/debug/pool-stats"
echo ""

echo ""
echo "=== Cache hit ratio: N/A -- no cache exists anywhere in this runtime"
echo "    (CR-17 is only partial metadata/interpreter caching; there is no"
echo "    request/query cache to have a hit ratio at all). Not fabricated."
echo ""

echo ""
echo "=== CPU/memory over the whole run (from /proc sampling, no server-code instrumentation) ==="
if [ -s "$LOG_DIR/samples.tsv" ]; then
  python3 -c "
rows = [l.split() for l in open('$LOG_DIR/samples.tsv')]
if len(rows) < 2:
    print('not enough samples')
else:
    rss = [int(r[1]) for r in rows]
    print(f'VmRSS kB: min={min(rss)} max={max(rss)} avg={sum(rss)//len(rss)}')
    t0, tk0 = float(rows[0][0]), int(rows[0][2])
    t1, tk1 = float(rows[-1][0]), int(rows[-1][2])
    hz = $HZ
    cpu_pct = (tk1 - tk0) / hz / (t1 - t0) * 100 if t1 > t0 else 0.0
    print(f'CPU%% over the whole run (avg): {cpu_pct:.1f}')
"
else
  echo "(no samples captured)"
fi

echo ""
echo "==> benchmark-composable done"
