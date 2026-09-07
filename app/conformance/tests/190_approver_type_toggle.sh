#!/usr/bin/env bash
# CAP-F24: per-record approver-type toggle on Approval Step. Each Step
# picks, at submission (not Machine-design time), whether its own approver
# gate is a specific person (fld_as_approver_type=User) or an entire
# Group's membership (=Group) -- migrations/028_dynamic_actor_permission.sql,
# seeds/043_approver_type_toggle.sql. Sourced by run.sh after lib.sh --
# reuses ALICE/BOB/CAROL/FRANK and their pre-resolved *_ID (150's own
# comment header explains the same convention).
#
# Re-derives its own Group id rather than reusing 150_group_approver_
# picker.sh's own $GROUP_ID variable (technically still in scope, every
# test file is `source`d into the same shell, run.sh's own loop) --
# depending on an unrelated file's local variable staying set is fragile
# against future re-ordering; this file is self-contained instead, same
# "Document Approvers" Group 150 already created with Bob as its only
# member (T217).

CAP_F24_GROUP_ID=$(curl -s -b "$FRANK" "$BASE_URL/admin/users" \
    | grep -oE 'href="/[a-z0-9_-]+/admin/groups/[a-f0-9-]+"[^>]*>[^<]*<div[^>]*>[^<]*</div>' \
    | grep "Document Approvers" \
    | grep -oE '/admin/groups/[a-f0-9-]+' | head -1 | sed 's#/admin/groups/##')

# T240 -- approver_type=User: a Step names Bob directly (the new field,
# fld_as_approver_user, not the legacy fld_as_approver) -- Bob may approve,
# Carol (a different Approver-role holder, not named on this Step) may not.
AD_TGL1_DATA="fld_ad_title=T240+Toggle+User&fld_ad_document_type=Policy&fld_ad_file=toggle1.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
AD_TGL1_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_TGL1_DATA" "$ALICE")
AD_TGL1_ID="${AD_TGL1_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_TGL1_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
TGL1_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_TGL1_ID&fld_as_approver_type=User&fld_as_approver_user=$BOB_ID&fld_as_sequence=1" "$ALICE")
TGL1_ID="${TGL1_URL##*/}"

CODE=$(post_status "$TGL1_URL/events/evt_as_approve" "" "$CAROL")
[ "$CODE" = "403" ]
check T240a "CAP-F24" "approver_type=User: Carol (not the named user) is denied (got $CODE)" $?

CODE=$(post_status "$TGL1_URL/events/evt_as_approve" "" "$BOB")
[ "$CODE" = "303" ]
check T240b "CAP-F24" "approver_type=User: Bob (the named user) approves (got $CODE)" $?

# T241 -- approver_type=Group: a Step names the "Document Approvers" Group
# (only Bob is a member, per T217) instead of a specific person -- Bob may
# approve BY VIRTUE OF MEMBERSHIP (never named directly on this Step's own
# data), Carol -- an Approver-role holder who is NOT a member -- may not,
# proving this is a real membership check, not a role check in disguise.
AD_TGL2_DATA="fld_ad_title=T241+Toggle+Group&fld_ad_document_type=Policy&fld_ad_file=toggle2.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
AD_TGL2_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_TGL2_DATA" "$ALICE")
AD_TGL2_ID="${AD_TGL2_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_TGL2_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
TGL2_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_TGL2_ID&fld_as_approver_type=Group&fld_as_approver_group=$CAP_F24_GROUP_ID&fld_as_sequence=1" "$ALICE")

CODE=$(post_status "$TGL2_URL/events/evt_as_approve" "" "$CAROL")
[ "$CODE" = "403" ]
check T241a "CAP-F24" "approver_type=Group: Carol (Approver role, NOT a Group member) is denied (got $CODE)" $?

CODE=$(post_status "$TGL2_URL/events/evt_as_approve" "" "$BOB")
[ "$CODE" = "303" ]
check T241b "CAP-F24" "approver_type=Group: Bob (a Group member, never named directly) approves (got $CODE)" $?

# T242 -- backward compatibility: a Step created the OLD way (fld_as_approver
# set directly, fld_as_approver_type left blank entirely) still resolves
# through ResolveActorGate's own fallback to owner_field -- perm_as_approver
# declaring BOTH owner_field and the new dynamic-actor columns must not
# change this Step's own behavior versus before CAP-F24 existed.
AD_TGL3_DATA="fld_ad_title=T242+Legacy+Fallback&fld_ad_document_type=Policy&fld_ad_file=toggle3.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
AD_TGL3_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_TGL3_DATA" "$ALICE")
AD_TGL3_ID="${AD_TGL3_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_TGL3_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
TGL3_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_TGL3_ID&fld_as_approver=$BOB_ID&fld_as_sequence=1" "$ALICE")

CODE=$(post_status "$TGL3_URL/events/evt_as_approve" "" "$CAROL")
[ "$CODE" = "403" ]
check T242a "CAP-F24" "legacy fld_as_approver (no toggle set): Carol still denied (got $CODE)" $?

CODE=$(post_status "$TGL3_URL/events/evt_as_approve" "" "$BOB")
[ "$CODE" = "303" ]
check T242b "CAP-F24" "legacy fld_as_approver (no toggle set): Bob still approves via owner_field fallback (got $CODE)" $?
