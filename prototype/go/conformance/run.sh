#!/usr/bin/env bash
# Menata Runtime — Conformance Suite
# Study 4 deliverable (runtime/capability-roadmap.md)
#
# Each test proves one or more capabilities from runtime/capability-registry.md.
# A capability marked ✅ in the registry must keep its test passing (ratchet rule).
#
# Usage:
#   ./conformance/run.sh                     # against http://localhost:4000
#   BASE_URL=https://aksi.menata.id ./conformance/run.sh
#
# Requires: Cases 1 & 2 seeded (seeds/001, seeds/002), server running.
# Note: creates test records in the target database (prototype-acceptable).

set -u
BASE_URL="${BASE_URL:-http://localhost:4000}"
PASS=0
FAIL=0

check() { # check <test-id> <cap-ids> <description> <condition-result>
    local id="$1" caps="$2" desc="$3" ok="$4"
    if [ "$ok" = "0" ]; then
        printf 'PASS  %-4s %-22s %s\n' "$id" "$caps" "$desc"
        PASS=$((PASS+1))
    else
        printf 'FAIL  %-4s %-22s %s\n' "$id" "$caps" "$desc"
        FAIL=$((FAIL+1))
    fi
}

body_contains() { # <url> <needle> [cookie]
    local url="$1" needle="$2" cookie="${3:-}"
    if [ -n "$cookie" ]; then
        curl -s -H "Cookie: $cookie" "$url" | grep -q "$needle"
    else
        curl -s "$url" | grep -q "$needle"
    fi
}

post_body_contains() { # <url> <data> <needle>
    curl -s -X POST "$1" -d "$2" | grep -q "$3"
}

post_status() { # <url> <data> [cookie] -> echoes http code
    local url="$1" data="$2" cookie="${3:-}"
    if [ -n "$cookie" ]; then
        curl -s -o /dev/null -w '%{http_code}' -X POST -H "Cookie: $cookie" "$url" -d "$data"
    else
        curl -s -o /dev/null -w '%{http_code}' -X POST "$url" -d "$data"
    fi
}

post_redirect() { # <url> <data> [cookie] -> echoes redirect url
    local url="$1" data="$2" cookie="${3:-}"
    if [ -n "$cookie" ]; then
        curl -s -o /dev/null -w '%{redirect_url}' -X POST -H "Cookie: $cookie" "$url" -d "$data"
    else
        curl -s -o /dev/null -w '%{redirect_url}' -X POST "$url" -d "$data"
    fi
}

echo "Menata Runtime Conformance Suite"
echo "Target: $BASE_URL"
echo "--------------------------------------------------------------------"

# T00 — server reachable
curl -s -o /dev/null --max-time 5 "$BASE_URL/health"
check T00 "—" "server /health reachable" $?
[ "$FAIL" -gt 0 ] && { echo "Server unreachable — aborting."; exit 1; }

# T01 — CAP-X01 multi-application, multi-machine
body_contains "$BASE_URL/" "Design Request" && body_contains "$BASE_URL/" "Leave Request"
check T01 "CAP-X01" "home lists machines from both applications" $?

# T02 — CAP-V01 form view: fields config drives inputs; status excluded
curl -s "$BASE_URL/mch_leave_request/new" | grep -q 'for="fld_lr_reason"' \
  && ! curl -s "$BASE_URL/mch_leave_request/new" | grep -q 'for="fld_lr_status"'
check T02 "CAP-V01" "form renders configured fields, excludes status" $?

# T03 — CAP-V02 list view: columns config drives table
body_contains "$BASE_URL/mch_leave_request" "Leave Type"
check T03 "CAP-V02" "list renders configured columns" $?

# T04 — CAP-C01 required constraint
post_body_contains "$BASE_URL/mch_leave_request" "" "Reason is required."
check T04 "CAP-C01" "empty submit rejected: required violation" $?

# T05 — CAP-C02 after:today constraint
post_body_contains "$BASE_URL/mch_leave_request" "" "Start Date must be after today."
check T05 "CAP-C02" "empty submit rejected: date-future violation" $?

# T06 — CAP-C03+C04 conditional constraint fires (Banner without attachment)
DR_DATA_BANNER="fld_requester=ConformanceBot&fld_design_type=Banner+2%3A1&fld_due_date=2030-01-01&fld_title=Conformance+T06&fld_description=Test"
post_body_contains "$BASE_URL/mch_design_request" "$DR_DATA_BANNER" "Attachment is required"
check T06 "CAP-C03,CAP-C04" "conditional constraint fires when condition true" $?

# T07 — CAP-C04 conditional constraint silent when condition false (Poster)
DR_DATA_POSTER="fld_requester=ConformanceBot&fld_design_type=Poster&fld_due_date=2030-01-01&fld_title=Conformance+T07&fld_description=Test"
CODE=$(post_status "$BASE_URL/mch_design_request" "$DR_DATA_POSTER")
[ "$CODE" = "303" ]
check T07 "CAP-C04" "conditional constraint silent when condition false (got $CODE)" $?

# T08 — CAP-R01 create record with default status
LR_DATA="fld_lr_employee=ConformanceBot&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=Conformance+run"
DETAIL_URL=$(post_redirect "$BASE_URL/mch_leave_request" "$LR_DATA")
[ -n "$DETAIL_URL" ] && body_contains "$DETAIL_URL" "Draft"
check T08 "CAP-R01" "valid create redirects to detail with default status Draft" $?

# derive record id from redirect
REC_ID="${DETAIL_URL##*/}"

# T09 — CAP-V03 detail view shows all fields
body_contains "$DETAIL_URL" "Reason"
check T09 "CAP-V03" "detail renders machine fields" $?

# T10 — CAP-E01+A01 permitted event executes set_field
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_submit" "" "menata_role=Employee")
[ "$CODE" = "303" ] && body_contains "$DETAIL_URL" "Submitted" "menata_role=Employee"
check T10 "CAP-E01,CAP-A01" "Employee triggers Submit; status becomes Submitted" $?

# T11 — CAP-P01 permission guard denies unpermitted role
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_approve" "" "menata_role=Employee")
[ "$CODE" = "403" ]
check T11 "CAP-P01" "Employee denied Approve (got $CODE)" $?

# T12 — CAP-P01+E01 permitted role executes cross-role transition
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_approve" "" "menata_role=Manager")
[ "$CODE" = "303" ] && body_contains "$DETAIL_URL" "Approved" "menata_role=Manager"
check T12 "CAP-P01,CAP-E01" "Manager triggers Approve; status becomes Approved" $?

# --- CAP-F13 (reference fields) — requires seeds/003_hr_employee.sql ---

# T13 — CAP-F13 form renders a picker (<select>), not a bare text input
body_contains "$BASE_URL/mch_employee/new" 'select id="fld_emp_manager"'
check T13 "CAP-F13" "reference field renders as a picker" $?

# T14 — CAP-F13 create with an empty (optional) reference succeeds
MGR_DATA="fld_emp_id=CB-MGR&fld_emp_name=ConformanceBot+Manager&fld_emp_hire_date=2020-01-01"
MGR_URL=$(post_redirect "$BASE_URL/mch_employee" "$MGR_DATA")
[ -n "$MGR_URL" ]
check T14 "CAP-F13" "create with empty reference succeeds (Manager is optional)" $?
MGR_ID="${MGR_URL##*/}"

# T15 — CAP-F13 create with a valid reference succeeds; detail links to the target
REPORT_DATA="fld_emp_id=CB-EMP&fld_emp_name=ConformanceBot+Report&fld_emp_hire_date=2024-01-01&fld_emp_manager=$MGR_ID"
REPORT_URL=$(post_redirect "$BASE_URL/mch_employee" "$REPORT_DATA")
[ -n "$REPORT_URL" ] && body_contains "$REPORT_URL" "href=\"/mch_employee/$MGR_ID\""
check T15 "CAP-F13" "create with valid reference succeeds; detail links to target record" $?

# T16 — CAP-F13 dangling reference rejected (security NFR gate: negative case)
BAD_DATA="fld_emp_id=CB-GHOST&fld_emp_name=ConformanceBot+Ghost&fld_emp_hire_date=2024-01-01&fld_emp_manager=00000000-0000-0000-0000-000000000000"
post_body_contains "$BASE_URL/mch_employee" "$BAD_DATA" "does not reference an existing"
check T16 "CAP-F13" "dangling reference value rejected, not silently accepted" $?

echo "--------------------------------------------------------------------"
echo "Result: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
