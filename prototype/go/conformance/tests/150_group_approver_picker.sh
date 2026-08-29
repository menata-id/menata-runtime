#!/usr/bin/env bash
# CAP-F23: group-restricted user-field picker. Sourced by run.sh after
# lib.sh -- reuses ALICE/BOB/CAROL/FRANK and their pre-resolved *_ID.
# seeds/038_group_approver_lab.sql sets fld_as_approver's own options to
# {"restrict_to_group":"Document Approvers"}; no Group by that name is
# seeded (CAP-O07's own established precedent), so T216 below proves the
# graceful-degrade path before this test creates one, same as every
# earlier batch that already ran Approval Step's picker unaffected.

# T216 -- before "Document Approvers" exists, the picker is unrestricted
# (graceful degrade, not an error) -- proves this doesn't regress any of
# the many earlier Case 3 tests that already ran against this exact field.
PRE_BODY=$(get_body "$BASE_URL/mch_approval_step/new" "$ALICE")
HAS_BOB=$(echo "$PRE_BODY" | grep -c "value=\"$BOB_ID\">Bob</option>")
HAS_CAROL=$(echo "$PRE_BODY" | grep -c "value=\"$CAROL_ID\">Carol</option>")
[ "$HAS_BOB" = "1" ] && [ "$HAS_CAROL" = "1" ]
check T216 "CAP-F23" "before the named Group exists, the Approver picker is unrestricted (graceful degrade)" $?

# T217 -- Frank (Admin) creates "Document Approvers" and adds only Bob.
# The picker now offers Bob but not Carol, even though Carol still holds
# the Approver role directly (UserStore.ListForApplicationRole's own scope
# is unchanged -- restriction narrows by INTERSECTION, not replacement).
post_status "$BASE_URL/admin/groups" "name=Document+Approvers" "$FRANK" >/dev/null
# admin/users renders every group on one line (no newlines in templ's
# output here) and ORDER BY name -- neither "last href on the page" nor
# "last line" reliably picks OUR group once another Group (e.g. CAP-O07's
# own "Groups Lab Approvers") already exists, so match the href immediately
# followed by our own group's name, not position.
GROUP_ID=$(curl -s -b "$FRANK" "$BASE_URL/admin/users" \
    | grep -oE 'href="/admin/groups/[a-f0-9-]+"[^>]*>[^<]*<div[^>]*>[^<]*</div>' \
    | grep "Document Approvers" \
    | grep -oE '/admin/groups/[a-f0-9-]+' | head -1 | sed 's#/admin/groups/##')
post_status "$BASE_URL/admin/groups/$GROUP_ID/members" "member_id=$BOB_ID" "$FRANK" >/dev/null

POST_BODY=$(get_body "$BASE_URL/mch_approval_step/new" "$ALICE")
NOW_HAS_BOB=$(echo "$POST_BODY" | grep -c "value=\"$BOB_ID\">Bob</option>")
NOW_HAS_CAROL=$(echo "$POST_BODY" | grep -c "value=\"$CAROL_ID\">Carol</option>")
[ "$NOW_HAS_BOB" = "1" ] && [ "$NOW_HAS_CAROL" = "0" ]
check T217 "CAP-F23" "once the Group exists with only Bob as a member, the picker offers Bob but not Carol (who still holds the role directly)" $?

# T218 -- Carol (role-holder, NOT a group member) can still be entered by
# raw form POST -- the picker narrows suggestions, it doesn't itself
# enforce -- and still Approves normally afterward: the real gate
# (perm_as_approver's own role+ownership check, already proven by T36/T94)
# is completely unchanged by this capability, not re-implemented.
AD_DATA="fld_ad_title=GroupPicker+$$&fld_ad_document_type=Policy&fld_ad_file=x.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
AD_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_DATA" "$ALICE")
AD_ID="${AD_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
AS_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=1" "$ALICE")
AS_ID="${AS_URL##*/}"
CAROL_APPROVE_CODE=$(post_status "$BASE_URL/mch_approval_step/$AS_ID/events/evt_as_approve" "" "$CAROL")
[ -n "$AS_ID" ] && [ "$CAROL_APPROVE_CODE" = "303" ]
check T218 "CAP-F23" "a role-holder outside the Group can still be assigned via raw POST and still Approves normally (got $CAROL_APPROVE_CODE) -- the picker is UX-only, not a new authorization mechanism" $?
