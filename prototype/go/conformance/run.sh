#!/usr/bin/env bash
# Menata Runtime — Conformance Suite
# Study 4 deliverable (runtime/capability-roadmap.md)
#
# Each test proves one or more capabilities from runtime/capability-registry.md.
# A capability marked ✅ in the registry must keep its test passing (ratchet rule).
#
# Usage:
#   ./conformance/run.sh                     # against http://localhost:4000
#   BASE_URL=https://menata.app ./conformance/run.sh
#
# Requires: seeds 001-007 applied, server running.
# Note: creates test records in the target database (prototype-acceptable).
# T19 is a deliberate, documented exception to "HTTP black-box": it uses psql
# to backdate a date field, simulating time passing without waiting years —
# a test fixture setup, not an inspection of runtime behavior. Set
# DATABASE_URL (same value as the server's .env) to enable it; skipped if unset.
#
# This file only orchestrates -- lib.sh (helpers + seeded-account sessions)
# and tests/NNN_*.sh (one file per capability batch, sourced in numeric order)
# hold the actual content. Split out of a single 2128-line file 2026-08-22 --
# see docs/decisions/007-conformance-suite-split.md for why and how. To add a
# NEW test: append to the last-numbered file if it's the same batch/theme, or
# create the next tests/NNN_*.sh (increment by 10, so an unplanned insertion
# never needs renumbering everything after it) if it's a new one.

set -u
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=lib.sh
source "$DIR/lib.sh"

for f in "$DIR"/tests/*.sh; do
    # shellcheck source=/dev/null
    source "$f"
done

echo "--------------------------------------------------------------------"
echo "Result: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
