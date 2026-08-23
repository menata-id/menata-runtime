#!/usr/bin/env bash
# --- CAP-O03 Tier 3: within-Machine view-type nav pill, + the shellBottomBar
# correction (2026-08-23) ---
# benchmarks/021-design-system-prototype-plan.md (Study 29),
# prototype/go/docs/decisions/008-mobile-ui-navigation-standard.md (ADR-008),
# benchmarks/022-bottom-nav-consistency-benchmark.md (Study 30, the
# bottom-bar correction). No new seed needed -- reuses app_accounting
# (seeds/008 + seeds/010's added report View, mch_journal_entry_line has
# both list and report) for the positive case and app_kanban_lab
# (seeds/032, mch_kanban_task has only a board View) for the negative one.

KB_LEAD=$(session_for kanban.lead@example.com password)

# T186 -- a Machine declaring more than one collection-level View type
# (List + Report here) renders a segmented pill linking between them, the
# current one marked active -- CAP-O03 Tier 3's own within-Machine axis,
# distinct from T135's cross-Machine subNavBar strip above.
body_contains "$BASE_URL/mch_journal_entry_line" 'href="/mch_journal_entry_line/report"' "$IVY" && \
    body_contains "$BASE_URL/mch_journal_entry_line" 'bg-white text-slate-900 shadow-sm">List<' "$IVY" && \
    body_contains "$BASE_URL/mch_journal_entry_line/report" 'href="/mch_journal_entry_line"' "$IVY" && \
    body_contains "$BASE_URL/mch_journal_entry_line/report" 'bg-white text-slate-900 shadow-sm">Report<' "$IVY"
check T186 "CAP-O03" "a Machine with both List and Report renders a within-Machine view-type pill on each, active-highlighting whichever is being viewed" $?

# T187 -- a Machine declaring only ONE collection-level View type (the
# kanban board here has no List view at all) renders no pill -- nothing to
# switch to, same "fewer than 2 = nil" rule subNavFor already established
# for the cross-Machine strip.
! body_contains "$BASE_URL/mch_kanban_task/board" 'rounded px-3 py-1 text-xs font-medium transition-colors bg-white text-slate-900 shadow-sm' "$KB_LEAD"
check T187 "CAP-O03" "a Machine with only one collection-level View type renders no within-Machine pill" $?

# T188 -- the mobile bottom bar is a fixed, GLOBAL Home/Search/Notifications
# set, not per-Application data -- byte-identical on two pages belonging to
# two completely different Applications (Accounting vs Kanban Lab), per
# benchmarks/022's own correction. Proves the bar's own markup, not just
# that each page renders successfully.
BOTTOMBAR_ACCOUNTING=$(get_body "$BASE_URL/mch_journal_entry_line" "$IVY" | grep -o 'sm:hidden fixed inset-x-0 bottom-0.*Notifications')
BOTTOMBAR_KANBAN=$(get_body "$BASE_URL/mch_kanban_task/board" "$KB_LEAD" | grep -o 'sm:hidden fixed inset-x-0 bottom-0.*Notifications')
[ -n "$BOTTOMBAR_ACCOUNTING" ] && [ "$BOTTOMBAR_ACCOUNTING" = "$BOTTOMBAR_KANBAN" ]
check T188 "CAP-O03" "the mobile bottom bar (Home/Search/Notifications) is identical across two different Applications, not reconfigured per app (got same=$([ "$BOTTOMBAR_ACCOUNTING" = "$BOTTOMBAR_KANBAN" ] && echo yes || echo no))" $?
