# CLAUDE.md

Guidance for Claude Code (or any AI agent) working in `prototype/go` — the reference Go
implementation of Menata Runtime. Read `DEVELOPMENT.md` first for setup, `ARCHITECTURE.md` for the
layer model. This file is about the patterns, conventions, and gotchas that only became visible
while actually implementing capabilities against this codebase — the "how we actually work here"
document `DEVELOPMENT.md`/`ARCHITECTURE.md` don't cover.

## What this codebase is

A working prototype that interprets Runtime Metadata (stored in PostgreSQL) into a running HTTP
application. It is not a toy — every capability implemented here has a real conformance test
(`conformance/run.sh`) that exercises it against a genuine running server + database, and every
implemented capability is recorded with its proof in `../capability-registry.md`. There are no Go
unit tests; the conformance suite *is* the test suite. If you implement something, prove it the
same way — don't report a capability "done" without running the server and watching it work.

## The capability implementation loop

`../capability-lifecycle.md` defines a 9-layer Definition of Done for any capability. In this
codebase specifically, the layers map to:

| Layer | Where |
|---|---|
| 2. Metadata schema | `migrations/NNN_*.sql` (new column/table) — only when the JSONB `options`/`params` shape genuinely can't hold it |
| 3. Loader | `internal/metadata/loader.go` — parse the new column, validate at load time (dangling refs = load-time error, never a runtime surprise — see `validateReferences` for the pattern) |
| 4. Model | `internal/model/model.go` — new struct field or `ActionType`/`FieldType` constant |
| 5. Engine | Single-record logic → `internal/executor/executor.go` (`Simulate`/`Persist`). Cross-record logic (needs `interp` + `records` together) → package `handler`, not Executor — Executor deliberately has no access to the Interpreter, so anything requiring "look up another Machine" or "list sibling records" belongs in Handler. **Since ADR-006 (2026-08-22) split `handler.go` into domain files, land it in the file matching the domain, not `handler.go` itself** (now trimmed to construction/session/identity only) — event-trigger/workflow-action logic goes in `events.go` (`doAggregateStatus`, `sequentialGuardViolation`, `separationOfDutiesViolation`), form/reference-lookup logic in `formfields.go` (`childLists`, `buildFormFields`), calendar/dashboard/report rendering in `views.go`; see that ADR's own table for the full file-to-domain map before adding a new one |
| 6. UI | `internal/ui/*.templ`, regenerate with `make generate` before `go build` |
| 7. Conformance | New `T##` in the right `conformance/tests/NNN_*.sh` file (append to the last one if it's the same batch/theme, or a new `tests/NNN_*.sh` — increment by 10 — if it's a new one; see ADR-007) + a row in `conformance/README.md`'s test map — positive **and** negative case |
| 9. Registry | Update the row in `../capability-registry.md`: status, Proof column (`conformance T##`), and a note describing what was actually built and what was deliberately deferred |

Also update `../roadmap.md` with a dated status paragraph (append, don't rewrite prior entries —
see the existing "Status update (2026-07-11, ...)" blocks for the pattern) and cross out the
capability in the Implementation Order table.

## Established patterns

**`triggerEvent` is the one path every event runs through.** `internal/handler/events.go`'s
`triggerEvent(ctx, machine, event, rec, actorRole)` is called both by the HTTP handler
(`TriggerEvent`) and internally by `doAggregateStatus` when a workflow action needs to fire an
event on a *different* record (e.g. a Step's decision cascading to its parent Document). Guards
(CAP-E06's `event.Condition`, CAP-A07's sequential check), CAP-C09's constraint re-validation, and
CAP-A02's dynamic-value resolution all live inside this one function — so a system-triggered
transition can never bypass a check a user-triggered one would have to pass. If you add a new
guard or validation step, add it here, not in the HTTP handler directly, or internal callers will
silently skip it.

**`ruleViolation` distinguishes a business-rule rejection (400) from an infrastructure failure
(500).** `triggerEvent` returns a plain `error` for DB/system failures and `&ruleViolation{msg}`
for anything a user did "wrong" (wrong state, failed constraint). `TriggerEvent` uses
`errors.As` to pick the HTTP status. Follow this if you add a new rejection path — don't collapse
both into one status code.

**Two more real trigger sources exist besides an HTTP POST, both wired into the exact same
`triggerEvent` path — no separate code path skips its guards.** `cmd/server/main.go`'s
`runScheduler` is a real `time.Ticker` (once a minute) that walks every workspace once per tick
and fires any Event whose `schedule` (CAP-E02 `time`, CAP-E03 `date_field`+`offset_days`) is
due, de-duplicated per record per day via the existing `record_events` audit table (no new
tracking table). `POST /webhooks/{machine}/{record}/{event}` (CAP-E04) is the external trigger
— exempted from `sessionAuth`/`csrfProtect` (an external system has no browser session) but
gated by its own credential instead, `Machine.Config["webhook_secret"]` checked against an
`X-Webhook-Secret` header. If you add a third trigger source, route it through `triggerEvent`
the same way — never re-implement its guards inline at the new call site.

**Cross-machine reaction without the publisher knowing its subscribers (CAP-I01) dispatches
from the same post-commit site CAP-A07/A08/E05's own workflow actions already use.** A
Subscription's own failure can't roll back the publisher (it only ever runs after `Persist`
succeeds), and independent Subscriptions on the same publisher Event don't affect each other —
this fell out of reusing the existing call site rather than needing new isolation logic.

**A "slow, best-effort, already-resolved" action enqueues onto `action_outbox` (CAP-W06)
instead of performing its own write inline.** `doNotify` (CAP-A03/A04) and
`processSubscriptions` (CAP-I01) both keep their existing resolution logic completely
unchanged — recipient/message, or Contract check + `ResolveFields` — only their final step
changed, from calling the store directly to `outbox.Enqueue(ctx, workspaceID, kind, payload,
middleware.GetReqID(ctx))`. `Enqueue` runs against whatever `dbFromContext` returns, so it's
atomic with the caller's own already-open transaction for free (same mechanism CAP-X12 already
gets from `workspaceTx`). `runOutboxDispatcher` (`cmd/server/main.go`) performs the real write
later, off the request path, on the exact same per-workspace-per-tick transaction shape
`runScheduler` uses (`action_outbox` has `FORCE ROW LEVEL SECURITY`, so a single query can't
see rows across every workspace at once). **Only use this for an action whose failure was
already isolated/best-effort before CAP-W06 existed** — `create_record`/`cross_set_field`/
`batch_generate` are deliberately NOT here: CAP-X12 hardened those to abort the whole event on
failure, a guarantee an async dispatcher (running after the triggering transaction already
committed) can't provide. **A claimed batch must dispatch each item inside its own nested
transaction (`tx.Begin(ctx)`, a real Postgres `SAVEPOINT`), never all in one shared
transaction** — one item's dispatch error aborts whatever Postgres transaction it ran in, and
without a per-item savepoint, that takes the item's own subsequent `MarkFailed` call and every
other already-succeeded item in the same batch down with it (caught live 2026-08-23, see
`capability-registry.md`'s CAP-W06 row for the full story).

**A capability that mutates a Machine's Events/Permissions/Constraints in memory at load
time (`compileProcess`/`compileChangePolicies`/`compileApprovalRequirements`,
`internal/metadata/compile.go`) never writes that generated content back to the database —
the DB only ever holds the raw, hand-authored declaration.** `GET /apps/{id}/export`
(CAP-X08) dumps the POST-compile in-memory struct, so it is NOT a safe verbatim source for
re-inserting as if it were raw data — a Process Overlay Machine or a `change_policy`
Constraint round-tripped through export→import would either double-generate the same
deterministic-ID content on the next load, or (change_policy) permanently fail to load at
all (the loader's own rule rejects a Constraint carrying both a Condition and a
change_policy, which is exactly the shape a compiled export has). CAP-X08's import half
(`internal/handler/api.go`'s `APIImportApplication`) rejects a package containing either
upfront, named explicitly — if you add a new load-time-compiled capability in the future,
add its own exclusion to `rejectCompiledContent` too, or import will silently corrupt it
the same way.

**Validating a freshly-written package before committing it can mean building a SECOND
`metadata.Loader` bound to the same open transaction, not re-deriving the validation rules.**
`Loader.db` is a local `querier` interface (`Query` only), satisfied by both
`*pgxpool.Pool` (boot, `/admin/reload`) and `pgx.Tx` — `metadata.NewLoader(tx).LoadAll(ctx)`
inside a still-open transaction sees that transaction's own uncommitted writes (Postgres
always does) and runs the exact production `validateReferences`/`validateOperators`/compile
path against them, zero duplicated logic. CAP-X08 import uses this to validate a
newly-materialized package before commit; reuse it for any future "write then verify
atomically" mechanism rather than writing a second validator.

**Every HTTP request already runs inside one real Postgres transaction — a store method
swallowing its own error instead of returning it silently breaks that guarantee.**
`cmd/server/main.go`'s `workspaceTx` middleware (built for CAP-X06's RLS cutover) wraps every
request in `pool.Begin()`/`tx.Commit()` (only on a non-5xx response)/a deferred `Rollback()`
otherwise — `store.WithTx` attaches it to `ctx`, and every `RecordStore`/`NotificationStore`/
etc. method already picks it up automatically via `dbFromContext`. This means an event whose
actions touch several Machines (`create_record`, `cross_set_field`, `batch_generate`) already
gets true cross-record atomicity **for free** — but only if a failed write inside one of those
actions actually becomes a `error` return that reaches the HTTP handler as a 5xx. CAP-X12
(2026-07-12) found and fixed exactly this: those three `Executor` methods used to `slog.Error`
a failed `records.Create`/`records.Update` and return `nil`, so the request still finished as a
2xx and `workspaceTx` still committed the transaction — silently keeping the record's own
`set_field` changes and any earlier, otherwise-successful action in the same event's chain,
while only the failed action vanished. **If you add a new action type or any other code path
that writes to the DB inside an event's action loop, return its error — don't log-and-swallow
it** — the existing transaction machinery only protects you if you let it see the failure.

**A dedupe/claim check is a single `INSERT ... ON CONFLICT DO NOTHING`, never a
SELECT-then-INSERT.** CAP-X13's webhook idempotency (`RecordStore.ClaimWebhookEvent`) claims an
`(machine, event, idempotency_key)` tuple this way — a check-then-act pattern races two
near-simultaneous retries against each other (both SELECTs can see "not claimed yet" before
either INSERT commits); the atomic-INSERT form can't race regardless of how many callers hit it
at once. Reuse this shape for any future "has this already happened" check.

**A JSON request body has no `csrf_token` form field — `csrfProtect` (main.go) now also accepts
`X-CSRF-Token` as a header, falling back to it only when `FormValue("csrf_token")` is empty.**
CAP-X07's `internal/handler/api.go` routes use the same session-cookie auth as every HTML
route, just with the CSRF token carried differently for that one content type — there is no
separate API-key mechanism in this codebase.

**Every page belonging to a Machine must pass `h.subNavFor(r, machine)` as `ui.Page`'s
`subNav` argument (CAP-O03 Tier 2) — a page that forgets to silently renders with no sub-nav
strip, not an error.** `ui.Page`'s `subNav []SubNavLink` parameter renders a persistent,
Application-scoped nav strip (`internal/ui/layout.templ`'s `subNavBar`) listing a Machine's
sibling Machines in the same Application, permission-trimmed the same way `AppMachines`
already is. If you add a new Machine-scoped page renderer, thread `subNav []SubNavLink` through
its own templ signature and call `h.subNavFor(r, machine)` at the handler call site — the 8
existing ones (List, Detail, Form, WizardForm, CalendarTimeline, Dashboard, Report, ImportCSV)
are the pattern to copy. A page that ISN'T scoped to one Machine (workspace/Application home,
Search, Notifications, Admin) passes `nil` instead — there's no "current Machine" to resolve
siblings from.

**`Executor.Simulate` / `Executor.Persist` split exists for CAP-C09.** `Simulate` computes an
event's resulting data without writing anything; the caller (`triggerEvent`) validates that result
against every declared Constraint *before* calling `Persist`. Never call `Persist` without having
run `engine.Violations` on the simulated result first — that's the whole mechanism CAP-C09 depends
on.

**Cross-record field/machine resolution is heuristic, and that's a deliberate, documented
trade-off — not a bug.** Menata Language has no grammar yet for a business author to say "this is
the display field" or "this is the field that scopes this child record to its parent." So
`displayLabel` (pick the `Name` field, or the first `text` field, or fall back to the record id),
`findFieldByName`, `findReferenceFieldTo`, and the `Sequence`/`Decision`/`Approver` name-matching in
the workflow functions are all *prototype-honest heuristics*, not hidden magic — each one says so
in its own doc comment. When you add a new one, name it as a heuristic in the comment and in the
`capability-registry.md` note, the same way. If a second, different case ever needs a *different*
heuristic for the same lookup, that's the signal a real Language-grammar gap exists (candidate for
a genuine new capability, not another heuristic).

**`Machine.Config` (CAP-X03) is for settings about the Machine itself, not a Field of its
records.** Generic `map[string]string`, JSONB-backed, nil when unused — don't add a new migration
column for the next machine-level setting; add a new key. Only Approval Document uses it today
(`approval_mode_field`, `steps_machine`, `steps_parent_field`).

**Seed files are append-only in a specific way.** `fields`, `constraints`, `permissions`, `views`,
and `machines` all use `ON CONFLICT (id) DO NOTHING`, safe to re-run. **`event_actions` has no
natural key and is a plain `INSERT` with no conflict clause** — re-running a seed file's
`event_actions` block against a database that already has those rows duplicates them. If you need
to add actions to an *already-seeded* event (not a fresh install), write a scoped `INSERT` for just
the new rows (and a position-renumbering `UPDATE` if you're inserting before an existing action) —
don't re-run the whole seed file. See the git history of `seeds/002_leave_request.sql` around the
CAP-A02 change for a worked example.

**The same `ON CONFLICT (id) DO NOTHING` guarantee has a sharper edge for `views`/`fields`/etc: it
protects against re-running a seed file, but not against *editing* an already-seeded row's own
content later while keeping the same `id`.** Found live 2026-08-23: `seeds/027_case9_completion_
lab.sql`'s `vw_c9je_form` row was edited in place to add a `child_lines` key (for CAP-V15) after
that exact `id` had already been seeded once on the shared dev database — every later `make seed`
silently kept serving the OLD config, because `ON CONFLICT (id) DO NOTHING` doesn't care that the
`INSERT`'s own values changed. Nothing in the loader or the conformance suite catches this (the
suite ran against a *fresh* schema in CI-equivalent fashion, where the row is inserted correctly
the first time — the drift only shows up on a long-lived, already-seeded database, exactly the
shared dev deployment). If you edit an already-seeded row's content in an existing seed file
(not just append new rows), you need a companion `UPDATE ... WHERE id = '...'` for it to actually
reach a database that already ran the old version — or accept that only a fresh schema will ever
see the new content.

**When a file's own concerns keep accumulating and nothing forces them apart, split it — two
worked examples exist, follow their pattern rather than re-deriving one.** `internal/handler/
handler.go` (3,244 lines, ADR-006) and `conformance/run.sh` (2,128 lines, ADR-007) both grew this
way — one commit at a time, every batch adding a little more, no single change ever big enough to
trigger a rewrite on its own. Both splits were a **pure move** (no logic changed) verified
mechanically (function-inventory equality, and for the conformance suite, byte-identical
behavior on two independent fresh schemas) rather than trusted by inspection alone. Rough
trigger, not a hard rule: a file north of ~1,000 lines covering more than one clearly-separable
concern is a candidate; read ADR-006/ADR-007's own Context sections for the file-length
guidelines they surveyed before deciding whether a given file has actually crossed that line.

**"Cross-record logic belongs in Handler, not Executor" is about needing the Interpreter, not
about touching another record.** CAP-F22's `doCompositeSignature` looks up a different Machine's
record (the Document, via a reference field) and scans a third Machine's records (Signature, by
owner) — cross-record on its face, the same shape as `doAggregateStatus`/`doActivateNext` — but it
lives in `internal/executor/executor.go`, not `internal/handler/events.go`. The real test is
narrower than "does it touch another record": `doCrossSetField` already proved a cross-record
*write* can live in Executor as long as it only needs `RecordStore` (`e.records.Get`/`.Update`),
never `Interpreter` (to resolve another Machine's own field/event definitions). `doCompositeSignature`
resolves its target fields from the event action's own declared params, not from Interpreter
lookups, so it fits the same test — and staying in Executor means a failed composite aborts the
whole `Persist` call within the *same* transaction (`workspaceTx` rolls back), instead of
`doAggregateStatus`'s own post-commit-cascade shape. When adding a new cross-record action, ask
"does this need Interpreter, or just RecordStore?" before defaulting it into Handler — the answer
changes both which file it belongs in and what transactional guarantee it gets for free.

**`Persist` didn't forward `actorIdentityID` until CAP-F22 needed it — check what identity form an
action actually needs before trusting `actorIdentity` alone.** `doCreateRecord`/`doBatchGenerate`'s
existing `current_user` dynamic-value resolution wants the acting person's display NAME
(`actorIdentity`, e.g. "Bob") for a human-readable text field. A `user`-typed Field never stores a
name — it stores the same UUID form as `fld_as_approver`/`fld_ad_submitted_by` (CAP-F05) — so
comparing a `user` field's stored value against `actorIdentity` silently never matches. Caught live
building CAP-F22 (`Executor.Persist` gained an `actorIdentityID` parameter specifically for this):
a Signature lookup filtered by `Data[ownerField] == actorIdentity` always returned empty, and the
resulting 500 had no server-side log to explain why (`TriggerEvent`'s own error branch just returns
"event failed" — see this file's own `ruleViolation` note above). If a new action needs to compare
against a `user`-typed field's value, it needs `actorIdentityID`, not `actorIdentity` — verify
which one an action actually needs rather than assuming the one every existing action already uses.

**`guard.CanEdit`/`CanRead`/`CanCreate` are machine-level only — a capability that needs
per-record ownership on a plain field write (not an event trigger) has to add that check itself,
the same way CAP-P02 already does for `CanTrigger`, just without an `eventID` to key off of.**
`BoardMove`'s own `CanEdit` gate never needed this (a kanban lane has no "owner"), so copying its
gate verbatim into a capability that DOES have one is a silent authorization gap, not a safe
precedent to reuse as-is. CAP-V21's `coordPlaceOwnerOK` (`internal/handler/coordplace.go`) is the
worked example: same `OwnerField` comparison `permission.Guard.CanTrigger` already does
(`internal/permission/guard.go`), against a Permission row matched by role alone (no `eventID` to
match, since this isn't an event) — found by asking "could two different people holding the same
role interfere with each other here?" before shipping, not after a report. Ask that question for
any new record-scoped write that isn't a `triggerEvent` call.

**Before adding a new write route for a View, check whether an existing one already covers it --
`PermittedEventsForRecord` (`internal/interpreter/interpreter.go`) and `CanTrigger`
(`internal/permission/guard.go`) are pure functions of `(machine, roles, identityID, ...)`, not
tied to "the machine the current page's URL belongs to."** CAP-V20 (decision stepper) renders a
PARENT record's page but needs to trigger events on CHILD records -- the instinct is to assume
that needs a new POST route scoped to the parent. It doesn't: calling `PermittedEventsForRecord`
for the CHILD's own machine ID from a handler whose URL param is the PARENT's machine works
exactly the same as calling it from the child's own `Detail` handler, and the existing
`POST /{machineID}/{recordID}/events/{eventID}` route (`TriggerEvent`) already accepts any
machine/record pair with no coupling to which page rendered the button that posted to it. Ask
"does an event/write route already exist for the record I actually need to mutate?" before
scaffolding a new one scoped to the page doing the rendering.

**When metadata needs to reference something outside the metadata-load transaction (another
Machine/Field is fine, a row from a different, independently-managed system isn't), reference it
by a human-meaningful NAME resolved at request time, not an id assumed to exist at load time --
the same posture `Permission.Role` strings already have.** CAP-F23's `FieldOptions.
RestrictToGroup` names a Group by its own `name`, not `id`: unlike Machines/Fields (hand-picked
string ids, all declared together in the same seed/metadata load, so `validateReferences` can
check them at load time), a Group only ever gets a DB-generated UUID, created independently at
runtime via `/admin/groups` -- metadata authored ahead of time has no id to write down, and even
if it did, the Group might not exist yet when this workspace's metadata first loads. Don't force
a load-time-validated id reference onto something that isn't load-time data; resolve by name at
the point of use instead, and degrade gracefully (not an error) when the name doesn't resolve.

**The live dev database can silently drift behind `migrations/` -- `make migrate-up` is safe to
re-run and cheap, run it before trusting the schema matches what the code expects.** Found live
building CAP-F23: the shared `menata_runtime` database's `public` schema was missing every
migration from `022_action_outbox.sql` onward (`groups`, `action_outbox` didn't exist) even though
CAP-O07/CAP-W06 were both already ✅ in the registry with passing conformance -- those had only
ever been proven against throwaway isolated schemas (`?options=-csearch_path%3D...`) this session,
never against the persistent one. Every migration file already uses `IF NOT EXISTS`/guarded `DO`
blocks specifically so `make migrate-up` is idempotent and safe to run against a
partially-migrated database -- if something a registry row claims is ✅ doesn't seem to exist,
check this before assuming the code is wrong.

## Local dev loop that actually works here

Postgres runs locally in this environment already (`pg_isready`). `.env.example`'s
`postgres:password@localhost:5432/menata_runtime` are working credentials, not a placeholder.

**`.env`'s `PORT=4000` may not be a free port on this host.** This prototype's own dev deployment
(`https://menata.app`, not production — corrected 2026-08-22, see DEVELOPMENT.md "Dev
Deployment") runs on that exact port when it's up, managed by
`/root/scripts/server-manager.sh restart menata-runtime`. Check `ss -ltnp | grep :4000` before
starting a bare `go run`/`make dev` there — if this app's own dev instance is already running,
a second process silently fails to bind or fights the first for the port; if it's down (nothing
listening), starting one directly is fine. Separately, this host also runs OTHER apps' genuine
production instances (`/root/projects/MULTI-APP-GUIDE.md`'s port allocation map,
`server-manager.sh status`) — a `kill`/`pkill` without checking what's actually listening has
taken down another app's production instance before, so still stop this app via
`server-manager.sh` rather than a raw `kill` when in doubt. If you need an ad hoc dev instance
alongside a running one, use a different `PORT` for it.

```bash
make migrate-up && make seed         # fresh database only, see Makefile comments
make dev                              # foreground; Ctrl-C to stop, or kill by port (see below)
DATABASE_URL="postgres://postgres:password@localhost:5432/menata_runtime?sslmode=disable" \
  make conformance                    # in another terminal, server must be running
```

After editing a `.templ` file: regenerate with the exact version pinned in `go.mod`, **not**
`make generate`, then `go build ./...` before restarting the server —
`go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate`. `make generate` uses
`$(go env GOPATH)/bin/templ`, a locally-cached binary that can be older than `go.mod`'s pinned
version; running it regenerates *every* `_templ.go` file with an older code-generation style, not
just the one you meant to touch — check `git diff --stat` only shows your intended file after
regenerating. Edits to the generated `_templ.go` files directly will be silently overwritten and
are never the right place to make a change either way.

To verify a capability manually before trusting the conformance suite alone: `curl` the actual
running server. Every capability implemented so far in this codebase (CAP-F13, CAP-E06, CAP-C09,
CAP-A02, CAP-V06, CAP-A07, CAP-A08, CAP-X03, CAP-R02, CAP-A03, CAP-A04, CAP-A10, CAP-P02, CAP-E05,
CAP-P05, CAP-R04, CAP-I04, CAP-O03, CAP-X02, CAP-O01, CAP-C05, CAP-C07, CAP-C12, CAP-X05, CAP-F16,
CAP-A06, CAP-A09, CAP-A11, CAP-A12, CAP-A13, CAP-A14, CAP-A15, CAP-V05, CAP-V07, CAP-V08, CAP-V09,
CAP-V10, CAP-V11, CAP-V12, CAP-V14, CAP-R03, CAP-R05, CAP-R06, CAP-R07, CAP-R08, CAP-P03, CAP-P04,
CAP-P06, CAP-P07, CAP-E02, CAP-E03, CAP-E04, CAP-I01, CAP-I02, CAP-I03, CAP-I05, CAP-O02, CAP-O04,
CAP-O05, CAP-O06, CAP-O07, CAP-F22, CAP-V21, CAP-V20, CAP-F23) was manually exercised end-to-end against a
real Postgres instance before its conformance test was written, and manual testing caught real bugs (a `Create`
default-value rule hardcoded to fields named "Status" that silently broke Approval Step's
"Decision" field; a conformance-helper missing a cookie parameter that made a test pass for the
wrong reason; a hand-typed non-UUID reference value crashing `RecordStore.Exists` with an
unhandled Postgres error instead of a validation message; a `NULLIF($4, '')` parameter bound
against a `uuid` column with no cast, which would have crashed every single `notify` action; a
seed account's bcrypt password hash that never actually matched the password it claimed to,
invisible for months because nothing verified `password_hash` before CAP-X02 existed) that
reading the code alone would not have surfaced.

**An isolated-schema test server that outlives its schema fails every request, and looks like a
data bug if you don't check for it first.** The safe way to try a new migration/seed against
this shared production database without risking `menata.app` is `CREATE SCHEMA test_batchN`,
apply migrations/seeds with `?options=-csearch_path%3Dtest_batchN` appended to `DATABASE_URL`,
and run a throwaway server on a different `PORT` (e.g. 4099) against it — fully reversible via
`DROP SCHEMA test_batchN CASCADE` when done. The gotcha: if you forget to kill that throwaway
server's process before `DROP SCHEMA`, or before starting a *second* throwaway server on the
same port for the next verification pass (e.g. re-testing against real production data with a
plain `DATABASE_URL`, no schema override), the OLD process can still be squatting on the port —
a new `nohup ... &` silently fails to bind (port already in use) while `curl .../health` keeps
returning 200 from the *old* process the whole time. Once its schema is dropped, every query
against it fails to find any table/row, so login and everything downstream returns 401/empty —
which looks exactly like corrupted seed data, not a leftover process. Check `ss -ltnp | grep
:<port>` and confirm the PID's `DATABASE_URL` (via `ps eww -p <pid> | grep DATABASE_URL`)
actually matches what you just exported, before trusting a health check on a port you've reused
across more than one verification pass in the same session.

**This build now has a real cgo dependency: `github.com/chai2010/webp`, linked against the
host's own `libwebp` (CAP-F06's image-compression pipeline, `internal/handler/upload.go`).**
Confirmed present on this host (`pkg-config --exists libwebp`, `CGO_ENABLED=1` by default) and
the same host serves both dev and production (`menata.app` runs on this exact machine, see
`/root/projects/MULTI-APP-GUIDE.md`), so a build here reproduces there — but if this project is
ever built on a *different* host, `go build` will fail at the cgo link step unless that host also
has `libwebp` installed. `golang.org/x/image` (the resize step, `golang.org/x/image/draw`) is
pure Go, no such risk. If `go build ./...` fails with a missing-package error for either after a
fresh clone, run `go mod tidy` — `go mod tidy` also silently *removes* an added dependency from
`go.mod`/`go.sum` if nothing in the tree imports it yet, so add the import first, then `go get`,
then `go mod tidy`, not the other way around (bit this session once: `go get` followed
immediately by `go mod tidy` before any file actually imported the package removed it again).

**Uploaded files live on local disk (`prototype/go/uploads/`, gitignored, created on first
upload via `os.MkdirAll`), not in Postgres.** This is a deliberate prototype-scope choice
(no object storage exists anywhere else in this stack either) with a real consequence: it does
NOT survive a redeploy that replaces the working directory, and it's specific to whichever host
`bin/server` runs from. Isolated-schema testing (the pattern described above) creates real files
under this same shared directory — clean them up (`rm -rf uploads`) after an isolated-schema
verification pass the same way you'd drop the schema itself, so stray test images don't
accumulate across every future batch's own verification runs.

**A seed/permission data change needs a server restart OR an admin reload to take effect, same as
a code change would need a restart.** The Interpreter loads every Machine's Permissions (and
everything else) into memory once at boot (`interpreter.New`) — re-running a seed file that adds
or updates a `permissions` row (e.g. a new `can_read` grant) changes the database immediately but
the *running* server keeps serving the old, now-stale in-memory model until either a restart or
`POST /admin/reload` (CAP-X04, `benchmarks/015-metadata-live-reload-proof.md`) picks it up.
Caught once already, before CAP-X04 existed: a new permission row tested as still-denied against
a server that hadn't been restarted yet, which looked like the fix hadn't worked at all — the
same trap now has a cheaper fix (reload, not restart), but forgetting either one still reproduces
it identically.
