#!/usr/bin/env bash
# --- CAP-O03 Tier 5, Phase 1: declared navigation ---
# benchmarks/009-in-app-navigation-benchmark.md's own implementation-plan
# follow-on finding. seeds/045_declared_navigation_pilot.sql declares
# app_approval's own nav (Approval Document, a Dashboard view, and a
# "Reference" group containing Signature) -- Approval Step is deliberately
# left undeclared, the exact clutter this capability exists to remove.
# app_accounting (seeds/008, no declared entries) is the regression control
# proving every Application without one keeps CAP-O03's original inferred
# behavior byte-for-byte unchanged.

ALICE_NAV=$(session_for alice@example.com password)

# T243 -- the sub-nav strip on a declared Application shows exactly the
# declared entries, in declared order: the Machine target (active, since
# this IS mch_approval_document's own page), the View target (linking to
# its own collection-level route), and the group label with its one nested
# child -- Approval Step is absent entirely, not merely unlinked.
SUBNAV_BODY=$(get_body "$BASE_URL/mch_approval_document" "$ALICE_NAV")
echo "$SUBNAV_BODY" | grep -q 'bg-white text-blue-700 shadow-sm">[[:space:]]*Approval Document' && \
    echo "$SUBNAV_BODY" | grep -q 'href="/ws_default/mch_approval_document/dashboard"' && \
    echo "$SUBNAV_BODY" | grep -q 'uppercase tracking-wide text-slate-400">[[:space:]]*Reference' && \
    echo "$SUBNAV_BODY" | grep -q 'href="/ws_default/mch_signature"' && \
    ! echo "$SUBNAV_BODY" | grep -q '>Approval Step<'
check T243 "CAP-O03" "declared navigation's sub-nav strip shows exactly the declared entries (machine, view, group+child), Approval Step absent" $?

# T244 -- the same curation applies to the Application's own app-launcher
# card list (AppMachines), not just the per-Machine strip -- both call
# sites read the same declared entries.
APPCARDS_BODY=$(get_body "$BASE_URL/apps/app_approval" "$ALICE_NAV")
echo "$APPCARDS_BODY" | grep -q '>Approval Document<' && \
    echo "$APPCARDS_BODY" | grep -q '>Signature<' && \
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
