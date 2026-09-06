#!/usr/bin/env bash
# Quality-ratchet CI gates for app/'s own hand-written Go code -- ported
# from portal-ga3's own fitness-function scripts (Makefile targets
# check-security-patterns, check-handler-size, check-complexity), adapted
# to a baseline-ratchet shape (matching portal-ga3's own check-a11y.sh /
# check-cross-domain-import.sh: fail only on regression past a recorded
# baseline, not on every pre-existing violation) so introducing this gate
# doesn't require fixing already-shipped, conformance-passing code first.
#
# See app/docs/portal-ga3-code-quality-benchmark.md for the full rationale.
#
# Usage: ./scripts/check-quality-gates.sh   (from app/, or via local-ci.sh)

set -euo pipefail
cd "$(dirname "$0")/.."   # app/

OVERALL_FAIL=0

# --- Gate 1: error-leak scan (CWE-209) ----------------------------------
# Flags any http.Error/apiJSON call whose message includes a raw .Error()
# call -- an internal error's own text reaching an HTTP response. A line
# that legitimately returns a controlled, human-authored validation
# message (not raw system/DB error text) is marked safe with a trailing
# `// errleak:allow: <reason>` comment; this scan skips those, so every
# occurrence is either fixed or explicitly justified, never silently
# ignored.
echo "=== Gate 1: error-leak scan (CWE-209) ==="
GATE1_FAIL=0
LEAKS=$(grep -rnE '(http\.Error\(w,|apiJSON\(w,).*\.Error\(\)' internal/ --include="*.go" \
  | grep -v '_test\.go' \
  | grep -v 'errleak:allow') || true
if [ -n "$LEAKS" ]; then
  echo "$LEAKS"
  echo ""
  echo "FAIL: raw .Error() reaching an HTTP response. Either stop leaking it,"
  echo "      or mark the line safe with a trailing '// errleak:allow: <reason>'"
  echo "      comment if it's a controlled, human-authored validation message."
  GATE1_FAIL=1
else
  echo "  PASS -- 0 unjustified error leaks."
fi
echo ""

# --- Gate 2: handler file size ratchet ----------------------------------
# quality-baselines/handler-loc.txt records each internal/handler/*.go
# file's LOC as of 2026-09-06. A file already in the baseline may not grow
# past its recorded size without a deliberate baseline update in the same
# change; a NEW handler file (not yet in the baseline) is held to
# portal-ga3's own Rule #7 flat budget: WARN >400 LOC, FAIL >800 LOC.
echo "=== Gate 2: handler file size ratchet ==="
GATE2_FAIL=0
BASELINE="scripts/quality-baselines/handler-loc.txt"
for f in internal/handler/*.go; do
  case "$f" in *_test.go) continue ;; esac
  loc=$(wc -l < "$f" | tr -d ' ')
  base=$(awk -v f="$f" '$2 == f {print $1}' "$BASELINE")
  if [ -n "$base" ]; then
    if [ "$loc" -gt "$base" ]; then
      echo "FAIL: $f grew from $base to $loc LOC -- split it, or if the growth"
      echo "      is deliberate, update $BASELINE in the same change."
      GATE2_FAIL=1
    elif [ "$loc" -gt 400 ]; then
      echo "  WARN: $f is $loc LOC (>400, already over budget in the baseline)."
    fi
  else
    if [ "$loc" -gt 800 ]; then
      echo "FAIL: new file $f is $loc LOC (>800) -- split it before adding more."
      GATE2_FAIL=1
    elif [ "$loc" -gt 400 ]; then
      echo "  WARN: new file $f is $loc LOC (>400) -- consider splitting soon."
    fi
  fi
done
[ "$GATE2_FAIL" -eq 0 ] && echo "  PASS -- no handler file grew past its baseline."
echo ""

# --- Gate 3: cyclomatic complexity ratchet ------------------------------
# Same ratchet shape as Gate 2, over gocyclo's -over 10 output for every
# hand-written package. internal/ui is excluded: every violation there is
# templ-generated branching inside *_templ.go, not something a human wrote
# or can simplify by hand -- portal-ga3's own check-complexity scopes
# itself to "domain core+handler files" for the identical reason.
echo "=== Gate 3: cyclomatic complexity ratchet (gocyclo, threshold 10) ==="
GATE3_FAIL=0
GOCYCLO_BASELINE="scripts/quality-baselines/gocyclo.txt"
CURRENT=$(go run github.com/fzipp/gocyclo/cmd/gocyclo@v0.6.0 -over 10 internal/ 2>/dev/null \
  | grep -v '_templ\.go' | awk '{print $2, $3, $1}') || true
while IFS= read -r line; do
  [ -z "$line" ] && continue
  pkg=$(awk '{print $1}' <<< "$line")
  fn=$(awk '{print $2}' <<< "$line")
  cur=$(awk '{print $3}' <<< "$line")
  base=$(awk -v p="$pkg" -v f="$fn" '$1 == p && $2 == f {print $3}' "$GOCYCLO_BASELINE")
  if [ -z "$base" ]; then
    echo "FAIL: $pkg $fn is a NEW function over complexity 10 (complexity $cur)."
    echo "      Simplify it, or if it's a deliberate addition, add it to $GOCYCLO_BASELINE."
    GATE3_FAIL=1
  elif [ "$cur" -gt "$base" ]; then
    echo "FAIL: $pkg $fn grew from complexity $base to $cur -- simplify it, or"
    echo "      update $GOCYCLO_BASELINE in the same change if deliberate."
    GATE3_FAIL=1
  fi
done <<< "$CURRENT"
[ "$GATE3_FAIL" -eq 0 ] && echo "  PASS -- no function grew past its complexity baseline."
echo ""

# --- Gate 4: actor-parameter ordering convention ------------------------
# internal/permission/doc.go's own "Actor-parameter convention" section
# states the rule this codebase already follows everywhere: ctx
# context.Context first when present, actorRole immediately before
# actorIdentity (never reversed, never split by an unrelated parameter).
# This guards that convention against silent drift as more functions are
# added -- it does not need a baseline because there are zero existing
# violations to grandfather.
echo "=== Gate 4: actor-parameter ordering convention ==="
GATE4_FAIL=0
VIOLATIONS=$(find internal/executor internal/permission internal/handler -name "*.go" ! -name "*_test.go" -print0 \
  | xargs -0 awk '
    /^func / && /actorRole/ && /actorIdentity/ {
      if (index($0, "actorIdentity") < index($0, "actorRole")) {
        print FILENAME":"FNR": actorIdentity appears before actorRole (convention: actorRole, actorIdentity order) -> " $0
      }
    }
    /^func / && (/actorRole/ || /actorIdentity/) && /context\.Context/ {
      ctxIdx = index($0, "context.Context")
      roleIdx = index($0, "actorRole"); if (roleIdx == 0) roleIdx = 999999
      identIdx = index($0, "actorIdentity"); if (identIdx == 0) identIdx = 999999
      first = (roleIdx < identIdx) ? roleIdx : identIdx
      if (ctxIdx > first) print FILENAME":"FNR": ctx must be the first parameter -> " $0
    }
  ') || true
if [ -n "$VIOLATIONS" ]; then
  echo "$VIOLATIONS"
  echo ""
  echo "FAIL: actor-parameter ordering convention violated -- see"
  echo "      internal/permission/doc.go's own 'Actor-parameter convention' section."
  GATE4_FAIL=1
else
  echo "  PASS -- ctx-first, actorRole-before-actorIdentity holds everywhere."
fi
echo ""

OVERALL_FAIL=$((GATE1_FAIL || GATE2_FAIL || GATE3_FAIL || GATE4_FAIL))
if [ "$OVERALL_FAIL" -ne 0 ]; then
  exit 1
fi
echo "==> check-quality-gates PASSED"
