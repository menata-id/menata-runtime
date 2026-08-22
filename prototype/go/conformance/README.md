# Conformance Suite

> Study 4 deliverable (`runtime/roadmap.md`).
>
> A capability exists only if an executable test proves it (TCK discipline).
> Every ✅ capability in `runtime/capability-registry.md` must keep its test
> passing — the ratchet rule.

---

## Run

```bash
# against local dev server
./conformance/run.sh

# against any deployment
BASE_URL=https://menata.app ./conformance/run.sh
```

Exit code 0 = all pass. Non-zero = at least one capability regressed.

**Prerequisites:** server running, seeds `001`–`010` applied (Cases 1, 2, 18, 3, 7, the
`ws_acme` isolation fixture, CAP-X02/CAP-O01's real accounts, and the Journal Entry/Action
Lab/Views Lab proof machines T60 onward depend on). For T19/T42/T43/T52, also
export `DATABASE_URL` (same value as the server's `.env`) — they're skipped otherwise. T52
additionally skips (not fails) until `migrations/009_workspace_isolation_rls.sql` (CAP-X06's
RLS cutover, deliberately not part of `make migrate-up` — see that migration's own header) has
actually been applied.

**CAP-X02 (2026-07-12):** every test now authenticates as a real seeded account
(`seeds/007_authentication.sql`) instead of fabricating a `menata_role`/`menata_identity`/
`menata_workspace` cookie — `session_for` logs in once per account (email+password) and caches
the resulting cookie jar; `csrf_for` scrapes that session's CSRF token once and appends it to
every POST. Which seeded account plays "the Employee" or "the Approver" for a given test is
whoever `seeds/007` actually assigned that role to, in that Application (CAP-O01 — role is
per-`(user, application)`, not global) — see `run.sh`'s ACCOUNTS comment block for the map.

---

## Test → Capability Map

| Test | Capabilities | Proves |
|------|--------------|--------|
| T00 | — | server reachable (gate) |
| T01 | CAP-X01 | multi-application, multi-machine in one workspace — observed via the CAP-O03 workspace home (role-scoped, not a flat machine list) |
| T02 | CAP-V01 | form view: fields config drives inputs; status excluded |
| T03 | CAP-V02 | list view: columns config drives table |
| T04 | CAP-C01 | `required` constraint rejects empty submit |
| T05 | CAP-C02 | `after: today` constraint rejects empty submit |
| T06 | CAP-C03, CAP-C04 | conditional constraint fires when condition true (Banner without attachment) |
| T07 | CAP-C04 | conditional constraint silent when condition false (Poster) |
| T08 | CAP-R01 | create record → 303 → default status Draft |
| T09 | CAP-V03 | detail view renders machine fields |
| T10 | CAP-E01, CAP-A01 | permitted role triggers event; set_field applies |
| T11 | CAP-P01 | unpermitted role gets 403 |
| T12 | CAP-P01, CAP-E01 | cross-role transition (Manager approves) |
| T13 | CAP-F13 | reference field renders as a picker, not a bare text input |
| T14 | CAP-F13 | create with an empty (optional) reference succeeds |
| T15 | CAP-F13 | create with a valid reference succeeds; detail links to the target record |
| T16 | CAP-F13 | dangling reference value rejected (negative case — security NFR gate) |
| T17 | CAP-E06 | Approve rejected while record is still Draft |
| T18 | CAP-E06 | Reject rejected on an already-Approved record — the exact Study 1 gap |
| T19 | CAP-C09 | a Constraint satisfied at Create is re-checked at event trigger, not skipped |
| T20 | CAP-A02 | `set_field` dynamic values (`today`, `current_user`) resolve to real values, not literal tokens |
| T21 | CAP-V06 | a record's detail page lists other records that reference it back (reverse lookup) |
| T22 | CAP-A07 | Sequential mode hard-blocks an out-of-order Approve/Reject (WCP-1 Sequence) |
| T23 | CAP-A07 | in-sequence Approve succeeds |
| T24 | CAP-A08 | parent Document auto-transitions to Approved once every Step is Approved |
| T25 | CAP-A07 | Parallel mode has no sequential gating (contrast with T22) |
| T26 | CAP-A08 | any-rejected cascades the parent to Rejected immediately, not waiting for the rest |
| T27 | CAP-R02 | edit form pre-fills the record's current field values |
| T28 | CAP-R02 | valid update persists a changed field; fields outside the form (e.g. Status) are left untouched |
| T29 | CAP-R02, CAP-C09 | update re-validates Constraints (required-field violation), same as Create |
| T30 | CAP-R02, CAP-F13 | update re-validates referential integrity, including a malformed (non-UUID) reference value |
| T31 | CAP-A03, CAP-A10 | static-role `notify` delivers a real in-app Notification row, not just a log line |
| T32 | CAP-A10 | unread notification count badge renders on the nav bar, on an unrelated page |
| T33 | CAP-A10 | marking a notification read persists (button disappears, not just a redirect) |
| T34 | CAP-A04 | dynamic `recipient_field` resolves to the record's own specific submitter, not a role |
| T35 | CAP-A04 | the generic role does not also receive the dynamically-targeted notification (negative case) |
| T36 | CAP-P02 | correct role but wrong identity is denied deciding another Approver's Step — direct allocation, not just role class (negative case) |
| T37 | CAP-E05 | `trigger_event` blocked from firing while the source event's own `events.condition` gate fails (negative case) |
| T38 | CAP-E05 | one event's `trigger_event` action fires another event on the SAME record — proven via a manual stand-in for the still-unbuilt daily cron trigger |
| T39 | CAP-P05 | a role with no permission row at all on a machine is denied List — deny-by-default, not implicitly allowed (negative case) |
| T40 | CAP-P05 | same reasoning, denied Create (negative case) |
| T41 | CAP-P05 | same reasoning, denied the Edit form (negative case) |
| T42 | CAP-R04 | `record_events.performed_by` carries the real acting identity (a real account name, CAP-X02), not NULL (DB inspection, same T19 exception) |
| T43 | CAP-I04 | one request's correlation_id is shared across every `record_events` row it produces, even across a cross-record cascade (DB inspection) |
| T44 | CAP-P05 | Approver can read Approval Document — needed for context on the Step they're deciding, surfaced by production log data |
| T45 | CAP-P05 | Approver still denied Create on Approval Document — read-only, not full access (negative case) |
| T46 | CAP-O03 | drilling into an Application (`GET /apps/{id}`) lists its own Machines |
| T47 | CAP-O03 | role-aware: an Application with zero readable Machines never shows its card on the workspace home (negative case) |
| T48 | CAP-O03 | within a visible Application, only individually-readable Machines are listed, not all of them |
| T49 | CAP-X06 | a `ws_default` session is denied (404) direct access to another workspace's Machine — app-layer guard (negative case) |
| T50 | CAP-X06 | same, for the Application route (negative case) |
| T51 | CAP-X06 | an account whose own workspace is `ws_acme` (CAP-X02: workspace comes from the authenticated account, not a client-suppliable cookie) can use its own Machine end to end (create + trigger event) |
| T52 | CAP-X06 | RLS probe: a record known to belong to `ws_acme` is invisible when `app.workspace_id` is set to `ws_default` — proves RLS itself, not just the app-layer guard (DB inspection, same T19 exception) |
| T53 | CAP-X02 | login with the wrong password is rejected (negative case) |
| T54 | CAP-X02 | an unauthenticated GET is redirected to `/login`, not served or errored |
| T55 | CAP-X02 | an unauthenticated POST gets 401, not a redirect |
| T56 | CAP-X02 | an authenticated request with no CSRF token is rejected — the check runs even when the session itself is valid |
| T57 | CAP-O01 | a non-Admin is denied `/admin/users` (negative case) |
| T58 | CAP-O01 | a real workspace Admin can reach `/admin/users` |
| T59 | CAP-O01 | one identity, one session, resolves a different role in each of two Applications with no manual role-switch step — the actual point of the two-tier role model |
| T60 | CAP-C07 | cross-field comparison (`value_field`) rejects End Date before Start Date, comparing against another Field's own value, not a literal |
| T61 | CAP-C05 | `greater_than` rejects a non-positive Sequence |
| T62 | CAP-C12 | composite uniqueness rejects a duplicate (Document, Sequence) pair |
| T63 | CAP-F16 | a parent record and both its embedded child rows are created atomically from one form submission |
| T64 | CAP-F16 | an invalid child row rejects the whole atomic create, not just that row (negative case) |
| T65 | CAP-A12 | `set_field.value = "next"` steps a value_list field to its next declared option |
| T66 | CAP-A11 | `set_field.value = "today + 7 Days"` resolves to the real date, not a literal string |
| T67 | CAP-A09 | a conditional action's `if` runs the action when its condition is true |
| T68 | CAP-A06 | `create_record` creates a real record on another Machine, copying a source field via `field:<id>` |
| T69 | CAP-A13 | `cross_set_field` updates a field on a DIFFERENT record, reached via a reference field on the triggering record |
| T70 | CAP-A15 | `batch_generate` creates N records from one action |
| T71 | CAP-A09 | a conditional action's `if` does NOT run when its condition is false (negative case) |
| T72 | CAP-A14 | an aggregate-conditioned trigger is rejected while the cross-record SUM is still under threshold (negative case) |
| T73 | CAP-A14 | an aggregate-conditioned trigger succeeds once the SUM crosses the threshold, and its own action fires |
| T74 | CAP-V04, CAP-V05, CAP-V09 | "My Overdue Tasks" shows a Task assigned to the viewer AND overdue, sorted soonest-due-first |
| T75 | CAP-V05, CAP-V09 | a Task assigned to the same viewer but not yet due is excluded (negative case) |
| T76 | CAP-V08 | `?q=` search matches one record's column, case-insensitive, and excludes an unrelated record |
| T77 | CAP-V07 | calendar view renders records grouped by their own `date_field` |
| T78 | CAP-V07 | timeline view renders (same grouping, read chronologically) |
| T79 | CAP-V10 | a dashboard composes sections sourcing two different Machines in one View |
| T80 | CAP-V13 | a report view groups records by `group_field` and sums each declared `sum_field` |
| T81 | CAP-V12 | a wizard's step 1 submission advances to step 2, carrying step 1's values forward as hidden inputs |
| T82 | CAP-V12 | a wizard's final step creates one record combining every step's fields |
| T83 | CAP-V14 | moving a record up swaps its position with its immediate predecessor |
| T84 | CAP-R03 | an archived record leaves the live list and is reachable via `?archived=1` |
| T85 | CAP-R03 | restoring an archived record returns it to the live list |
| T86 | CAP-R05 | a list with more than 25 records paginates, page indicator reflects it |
| T87 | CAP-R06 | CSV export includes a real record's own field values |
| T88 | CAP-R06 | CSV import creates a valid row and reports a specific per-row failure for an invalid one, in the same file |
| T89 | CAP-R07 | a Draft (not yet immutable) record can still be edited |
| T90 | CAP-R07 | a Posted (immutable) record rejects both Update and Archive — every mutation path, not just events (negative case) |
| T91 | CAP-R08 | a record created directly into its declared scratch state skips normally-blocking Constraints |
| T92 | CAP-R08 | the same incomplete scratch record rejects the event that would commit it out of scratch state, then succeeds once fixed |
| T93 | CAP-P03 | a record's own submitter cannot decide its Approval, even as the assigned owner (negative case) |
| T94 | CAP-P03 | deciding a DIFFERENT person's submission succeeds normally |
| T95 | CAP-P04 | delegating reassigns the owner field via a fresh trigger-time input, stamping who handed it off |
| T96 | CAP-P06 | a role's hidden field is absent from List and Detail, visible to a role that isn't restricted |
| T97 | CAP-P07 | an anonymous request (no session) reads a Machine whose Permissions grant role Visitor |
| T98 | CAP-P07 | anonymous access is still denied for a Machine with no Visitor grant, and for any POST (negative case) |
| T99 | CAP-E02 | a time-driven Event fires on its own, no user action, once the scheduled time is reached (real background scheduler, ~65s wait) |
| T100 | CAP-E03 | a date-driven Event fires when today equals the record's own date field plus the declared offset |
| T101 | CAP-E03 | a date-driven Event does not fire for a record whose own date field hasn't reached the offset yet (negative case) |
| T102 | CAP-E04 | a webhook with the correct per-Machine secret triggers an event with no session, stamping its own payload field via InputFields |
| T103 | CAP-E04 | a webhook with the wrong secret is rejected, record left untouched (negative case) |
| T104 | CAP-I01 | a cross-machine Subscription creates a record on a Machine the publisher's own metadata never names |
| T105 | CAP-I03 | a Subscription's Contract violation skips that Subscription's own action (negative case) |
| T106 | CAP-I03 | a Subscription's Contract being satisfied lets its own action fire |
| T107 | CAP-I05 | the same shared Machine accumulates contributions from two different, unrelated publisher Events |
| T108 | CAP-I02 | a deprecated Event still works and shows a Deprecated indicator |
| T109 | CAP-O02 | a reference field on a Machine in a different Application targets a master-data record |
| T110 | CAP-O02 | archiving a master-data record still referenced elsewhere is rejected (negative case) |
| T111 | CAP-O02 | an unreferenced master-data record archives normally |
| T112 | CAP-O04 | workspace-wide search finds a match on a Machine the searcher can read |
| T113 | CAP-O04 | search results are permission-trimmed — a role with no access to the Machine finds nothing (negative case) |
| T114 | CAP-O05 | switching to digest preference groups the same notifications by day |
| T115 | CAP-O06 | "N Business Days" date arithmetic skips weekends, matching an independent bash reimplementation |
| T116 | CAP-X12 | a cross-machine action chain rolls back as a whole on a downstream failure |
| T117 | CAP-X13 | a repeated webhook delivery with the same idempotency key returns success both times but only runs the event once |
| T118 | CAP-X13 | a different idempotency key is a genuinely new delivery, not suppressed |
| T119 | CAP-X07 | the JSON API lists and reads a machine's records, permission-trimmed; an unauthenticated request is denied |
| T120 | CAP-X07 | a JSON create via X-CSRF-Token header succeeds; a request with no CSRF token is rejected |
| T121 | CAP-X08 | an Application's metadata exports as JSON for an Admin; denied for a non-admin role |
| T122 | CAP-F07 | a number field's value round-trips through Create and Detail |
| T123 | CAP-F08 | a money field renders with its declared currency |
| T124 | CAP-F09 | a boolean field is Yes when checked, No when the checkbox is left unchecked |
| T125 | CAP-F10 | time and date_time fields render real HTML5 input types, not a text fallback |
| T126 | CAP-F10 | time/date_time/duration values round-trip through Create and Detail |
| T127 | CAP-F14 | a computed field is Price times Quantity, ignoring any value POSTed directly for it |
| T128 | CAP-F15 | a plain (non-value_list) field's declared default applies when left blank |
| T129 | CAP-F18 | consecutive Creates get sequential, zero-padded auto-numbers |
| T130 | CAP-F06 | an uploaded image is stored, resized, and re-encoded as real WebP |
| T131 | CAP-F06 | a file type outside the declared accept list is rejected, not silently stored |
| T132 | CAP-F17 | multi-currency money (currency + rate) computes its base-currency mirror correctly |
| T133 | CAP-F19 | quantity/UoM Tier 1 composition converts to its base unit correctly |
| T134 | CAP-F21 | a document View renders its template with real merge fields resolved |
| T135 | CAP-O03 | a multi-machine Application renders a persistent sub-nav to sibling Machines; a single-machine Application renders none |
| T136 | Process Overlay B1 | a Machine declaring only a `process` block boots, compiles, and renders with the correct generated initial state, same as a hand-authored Machine with the identical process |
| T137 | Process Overlay B1 | the compiled Machine runs the identical full lifecycle as the hand-authored one, including a System-side automatic transition (CAP-E05 chain) nobody clicked |
| T138 | Process Overlay B1 | the hand-authored control arm rejects a wrong-state transition (400), wrong-role transition (403), and non-owner transition (403) |
| T139 | Process Overlay B1 | the compiled arm rejects the identical three cases with the identical codes — the parity claim (brd-menata-runtime-v2.md §6.6) |
| T140 | CAP-W05 | a compiled Machine's process map lists every state (initial marked) and transition with the correct actor, including the auto step (System) |
| T141 | CAP-W05 | a hand-authored Machine's process map is identical to the compiled one's — legibility parity |
| T142 | CAP-W05 | a genuine pre-existing v1 Machine (Leave Request, predates `process`) reconstructs correctly — the decompile claim |
| T143 | CAP-W05 | a Machine with no `process_map` View declared 404s — the opt-in gate |
| T144 | CAP-W01 | a transition requiring cardinality-2 evidence rejects with 0 evidence attached |
| T145 | CAP-W01 | still rejects with 1 evidence attached — a real count, not a presence check |
| T146 | CAP-W01 | succeeds once a 2nd evidence record is attached — write-time fan-in, read-time O(1) |
| T147 | CAP-W04 | a record left in an SLA-bound state past its due date auto-escalates to the declared state |
| T148 | CAP-W04 | a record that already left the SLA-bound state is untouched by the breach event |
| T149 | CAP-W03 | a 2-of-3 quorum reaches Approved once 2 votes are in, without waiting for the 3rd |
| T150 | CAP-W03 | a 2-of-3 quorum reaches Rejected once 2 votes are rejected (quorum mathematically impossible) |
| T151 | CAP-X04 | a Machine seeded mid-run becomes servable after `POST /admin/reload`, no server restart |
| T152 | CAP-X04 | a non-Admin cannot trigger a metadata reload |
| T153 | CAP-X04 | a bad reload is rejected (500) and the old interpreter keeps serving unrelated Machines normally |
| T154 | CAP-W07 | before change_policy exists, a blank Compliance Note is accepted |
| T155 | CAP-W07 | `records_in_states: [Draft]` rejects a blank Compliance Note on a Draft record |
| T156 | CAP-W07 | a record already past Draft when the rule arrived is grandfathered |
| T157 | CAP-W07 | a record created before `new_records`' effective_from is untouched by the new policy |
| T158 | CAP-W07 | `new_records` rejects a blank Approval Reference on a record created after the effective date |
| T159 | CAP-W03 | declarative quorum (`process.requirements[].type: approval`): 2-of-3 reaches Approved without waiting for the 3rd |
| T160 | CAP-W03 | declarative quorum: 2-of-3 reaches Rejected once quorum is mathematically impossible |
| T161 | CAP-C08 | an unbalanced entry (debit != credit) is rejected on Post (CAP-C10) |
| T162 | CAP-C08 | a balanced entry (debit = credit) posts successfully (CAP-C10) |
| T163 | CAP-C08 | posting into a Closed Fiscal Period is rejected, even when balanced (CAP-C11) |
| T164 | CAP-C08 | posting into an Open Fiscal Period succeeds (CAP-C11) |
| T165 | CAP-W05 | `GET .../process-lift` returns valid Process JSON for an Admin, denies a non-Admin |
| T166 | CAP-W05 | a lifted Process JSON, applied to a fresh Machine and reloaded, drives an identical lifecycle to the hand-authored/compiled pair (B6, decompile-lift) |
| T167 | CAP-V17 | a ticket due in the past renders the overdue countdown badge |
| T168 | CAP-V17 | a ticket due far in the future does not render the overdue badge |
| T169 | CAP-V18 | two staff with same-day appointments each show only their own (resource-grouped calendar) |
| T170 | CAP-V18 | a staff member with zero appointments still gets its own (empty) section |

---

## Design Notes

- **HTTP black-box** — tests exercise the runtime exactly as a user would; no DB inspection. Capabilities that are DB-only (e.g. CAP-R04 audit log) stay on manual evidence until a UI exposes them. **T19 is a deliberate, documented exception**: it uses `psql` to backdate a date field as test-fixture setup (simulating time passing without waiting years), not to inspect runtime behavior — the assertion itself is still a plain HTTP response check.
- **Data pollution accepted** — each run creates a few `ConformanceBot` records. Acceptable for the prototype; a future version should use a disposable workspace.
- **Adding a test:** new ✅ capability → add a `T##` here and in `run.sh`, then set the registry's Proof column to `conformance T##`.
- **State-guard caveat resolved (2026-07-11)** — CAP-E06 landed; T17/T18 assert rejection of out-of-state transitions, including the exact "Approved record still Rejectable" gap Study 1 found.
- **T99–T101 wait for a real clock, not a stand-in (2026-07-12)** — CAP-E02/E03's background scheduler ticks once a minute; the suite creates the qualifying records, sleeps ~65s once for all three tests together (not once each), then asserts. Slower than every other test here, deliberately: CAP-E05's own T38 used a manual stand-in for this exact gap before the real scheduler existed — now that it does, it gets proven against the real thing, not a simulation of it.
- **T115's holiday-skip half is manual, not automated (2026-07-12)** — CAP-O06's `"N Business Days"` unit skips both weekends and any date in the acting Workspace's own `workspace_holidays`. T115 only asserts the universal weekend-skip rule (reimplemented independently in bash, same precedent as T66's plain-day arithmetic) because a *seeded* holiday date would go stale relative to whenever this suite is actually run — a fixed "2026-08-17 is a holiday" row eventually stops being a useful assertion. The holiday-specific behavior is verified the same documented, DB-inspection way as T19/T42/T43/T52: insert a row into `workspace_holidays` directly, restart the server (holidays load once at boot, same as everything else the Interpreter caches), and confirm the same date arithmetic shifts around it — done manually during Batch 9's own verification, not part of this automated suite.
