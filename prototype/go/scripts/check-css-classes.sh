#!/usr/bin/env bash
# Verifies every color utility class referenced in the same content Tailwind
# itself scans (tailwind.config.js's own `content` globs) actually made it
# into the compiled web/static/css/output.css -- catches the class of bug
# found live 2026-09-05 (bg-emerald-500/text-emerald-600 used by CAP-V20's
# Decision Stepper, never present in the compiled CSS because nobody had
# re-run `make build-css` since that .templ file was written; conformance
# is HTTP black-box and never checks compiled CSS at all).
#
# Assumes web/static/css/output.css is ALREADY freshly built (run
# `make build-css` / `npm run build:css` immediately before this) -- this
# script only checks, it does not build.
#
# Usage: ./scripts/check-css-classes.sh   (from prototype/go/, or via
#        `make check-css`)

set -euo pipefail
cd "$(dirname "$0")/.."   # prototype/go/

CSS="web/static/css/output.css"
if [ ! -f "$CSS" ]; then
  echo "check-css-classes: $CSS does not exist -- run 'make build-css' first" >&2
  exit 1
fi

# Same color-utility prefixes Tailwind itself generates a distinct rule per
# shade for -- matches tailwind.config.js's own content scan scope (.templ
# and .go under internal/, plus cmd/'s own .go files).
PATTERN='\b(bg|text|border|ring|ring-offset|from|to|via|fill|stroke|divide|outline|accent|caret|decoration|shadow)-[a-z]+-[0-9]{2,3}\b'

mapfile -t used < <(grep -rhoE --include='*.templ' --include='*.go' "$PATTERN" internal/ cmd/ 2>/dev/null | sort -u)

missing=()
for cls in "${used[@]}"; do
  grep -qF -- "$cls" "$CSS" || missing+=("$cls")
done

if [ "${#missing[@]}" -gt 0 ]; then
  echo "check-css-classes: FAILED -- ${#missing[@]} class(es) used in .templ/.go source but missing from $CSS:" >&2
  printf '  %s\n' "${missing[@]}" >&2
  echo "Run 'make build-css' and commit/rebuild, or check for a typo in the class name." >&2
  exit 1
fi

echo "check-css-classes: OK -- ${#used[@]} distinct color utility class(es) checked, all present in $CSS"
