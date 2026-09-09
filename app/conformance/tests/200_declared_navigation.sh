#!/usr/bin/env bash
# --- CAP-O03 Tier 5, Phase 1: declared navigation ---
# benchmarks/009-in-app-navigation-benchmark.md's own implementation-plan
# follow-on finding. seeds/045_declared_navigation_pilot.sql declares
# app_approval's own real nav (Approval Document, a Dashboard view) --
# Approval Step is deliberately left undeclared, the exact clutter this
# capability exists to remove. seeds/046_navigation_group_lab.sql is a
# dedicated, separate fixture (app_nav_lab) for the group/nesting half of
# Phase 1, kept independent of app_approval's own actual curation choices
# after a live correction dropped that pilot's own group entry (Signature
# wasn't a real menu destination either, and the group label's own width
# overflowed a narrow viewport -- capability-registry.md's CAP-O03 Tier 5
# row has the full account). app_accounting (seeds/008, no declared
# entries) is the regression control proving every Application without one
# keeps CAP-O03's original inferred behavior byte-for-byte unchanged.

ALICE_NAV=$(session_for alice@example.com password)

# T243 -- the sub-nav strip on a declared Application shows exactly the
# declared entries, in declared order: the Machine target (active, since
# this IS mch_approval_document's own page) and the View target (linking
# to its own collection-level route) -- Approval Step is absent entirely,
# not merely unlinked.
#
# The View target's own href is asserted as /page, not /dashboard --
# seeds/051_dashboard_nav_supersede.sql (2026-09-09) repointed
# nav_ad_dashboard's own target_view at vw_ad_page (CAP-V10 Tier 2's real
# composed screen, the one that actually matches approval-dashboard.html)
# once that screen existed, superseding vw_ad_dashboard's own standalone
# route as the declared nav destination -- vw_ad_dashboard itself is
# unchanged and still real (vw_ad_page's own Children embeds it as that
# page's own "Summary" section), only which URL the nav link resolves to
# changed. This assertion was updated the same day as that seed, not left
# to silently start failing.
SUBNAV_BODY=$(get_body "$BASE_URL/mch_approval_document" "$ALICE_NAV")
echo "$SUBNAV_BODY" | grep -q 'bg-white text-blue-700 shadow-sm">[[:space:]]*Approval Document' && \
    echo "$SUBNAV_BODY" | grep -q 'href="/ws_default/mch_approval_document/page"' && \
    ! echo "$SUBNAV_BODY" | grep -q '>Approval Step<'
check T243 "CAP-O03" "declared navigation's sub-nav strip shows exactly the declared entries (machine, view), Approval Step absent" $?

# T244 -- the same curation applies to the Application's own app-launcher
# card list (AppMachines), not just the per-Machine strip -- both call
# sites read the same declared entries.
APPCARDS_BODY=$(get_body "$BASE_URL/apps/app_approval" "$ALICE_NAV")
echo "$APPCARDS_BODY" | grep -q '>Approval Document<' && \
    echo "$APPCARDS_BODY" | grep -q '>Dashboard<' && \
    ! echo "$APPCARDS_BODY" | grep -q '>Approval Step<'
check T244 "CAP-O03" "declared navigation's app-launcher card list matches the sub-nav strip's own curation" $?

# T245 -- Approval Step stays fully reachable and functional directly, even
# though it's absent from both nav surfaces above -- this capability is
# discoverability-only, never an access change (mirrors CAP-O03 Tier 4's
# own planned negative case).
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE_NAV" "$BASE_URL/mch_approval_step")
[ "$CODE" = "200" ]
check T245 "CAP-O03" "a Machine absent from declared navigation is still directly reachable and functional (got $CODE)" $?

# T246 -- regression control: an Application with ZERO declared entries
# (app_accounting) renders its sub-nav strip exactly as CAP-O03 Tier 2
# always has -- both Machines linked, nothing hidden, proving the
# declared-first/infer-fallback dispatch changes nothing for every
# pre-existing Application.
ACCOUNTANT=$(session_for accountant@example.com password)
body_contains "$BASE_URL/mch_journal_entry_line" 'href="/ws_default/mch_journal_entry"' "$ACCOUNTANT" && \
    body_contains "$BASE_URL/mch_journal_entry_line" 'href="/ws_default/mch_journal_entry_line"' "$ACCOUNTANT"
check T246 "CAP-O03" "an Application with no declared navigation entries keeps CAP-O03's original inferred sub-nav unchanged" $?

# T247 -- the group/nesting mechanism itself, on its own dedicated fixture
# (app_nav_lab, seeds/046): a group entry renders as a plain unclickable
# label, its one nested child renders as an ordinary link immediately
# after it, on BOTH the sub-nav strip and the app-launcher card list.
# Visitor/anonymous (CAP-P07) -- no session needed.
LABNAV_BODY=$(curl -s "$BASE_URL/mch_nav_primary")
echo "$LABNAV_BODY" | grep -q 'uppercase tracking-wide text-slate-400">[[:space:]]*Secondary Group' && \
    echo "$LABNAV_BODY" | grep -q 'href="/ws_default/mch_nav_secondary"'
check T247a "CAP-O03" "a declared group entry renders as an unclickable label, its nested child as a link right after it (sub-nav strip)" $?

NAVLAB=$(session_for nav.lab@example.com password)
LABAPP_BODY=$(get_body "$BASE_URL/apps/app_nav_lab" "$NAVLAB")
echo "$LABAPP_BODY" | grep -q 'col-span-full[^<]*>[[:space:]]*Secondary Group' && \
    echo "$LABAPP_BODY" | grep -q '>Secondary<'
check T247b "CAP-O03" "the same group/child nesting renders on the app-launcher card list" $?
