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
BASE_URL=https://aksi.menata.id ./conformance/run.sh
```

Exit code 0 = all pass. Non-zero = at least one capability regressed.

**Prerequisites:** server running, seeds `001`–`007` applied (Cases 1, 2, 18, 3, 7, the
`ws_acme` isolation fixture, and CAP-X02/CAP-O01's real accounts). For T19/T42/T43/T52, also
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

---

## Design Notes

- **HTTP black-box** — tests exercise the runtime exactly as a user would; no DB inspection. Capabilities that are DB-only (e.g. CAP-R04 audit log) stay on manual evidence until a UI exposes them. **T19 is a deliberate, documented exception**: it uses `psql` to backdate a date field as test-fixture setup (simulating time passing without waiting years), not to inspect runtime behavior — the assertion itself is still a plain HTTP response check.
- **Data pollution accepted** — each run creates a few `ConformanceBot` records. Acceptable for the prototype; a future version should use a disposable workspace.
- **Adding a test:** new ✅ capability → add a `T##` here and in `run.sh`, then set the registry's Proof column to `conformance T##`.
- **State-guard caveat resolved (2026-07-11)** — CAP-E06 landed; T17/T18 assert rejection of out-of-state transitions, including the exact "Approved record still Rejectable" gap Study 1 found.
