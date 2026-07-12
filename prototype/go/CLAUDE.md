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
| 5. Engine | Single-record logic → `internal/executor/executor.go` (`Simulate`/`Persist`). Cross-record logic (needs `interp` + `records` together) → `internal/handler/handler.go`, not Executor — Executor deliberately has no access to the Interpreter, so anything requiring "look up another Machine" or "list sibling records" belongs in Handler (see `childLists`, `sequentialGuardViolation`, `doAggregateStatus` for the pattern) |
| 6. UI | `internal/ui/*.templ`, regenerate with `make generate` before `go build` |
| 7. Conformance | New `T##` in `conformance/run.sh` + a row in `conformance/README.md`'s test map — positive **and** negative case |
| 9. Registry | Update the row in `../capability-registry.md`: status, Proof column (`conformance T##`), and a note describing what was actually built and what was deliberately deferred |

Also update `../roadmap.md` with a dated status paragraph (append, don't rewrite prior entries —
see the existing "Status update (2026-07-11, ...)" blocks for the pattern) and cross out the
capability in the Implementation Order table.

## Established patterns

**`triggerEvent` is the one path every event runs through.** `internal/handler/handler.go`'s
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

## Local dev loop that actually works here

Postgres runs locally in this environment already (`pg_isready`). `.env.example`'s
`postgres:password@localhost:5432/menata_runtime` are working credentials, not a placeholder.

**`.env`'s `PORT=4000` is not a free dev port on this host** — this prototype is deployed live at
`https://aksi.menata.id` on that exact port (see DEVELOPMENT.md "Production Deployment"), managed
by `/root/scripts/server-manager.sh restart menata-runtime`, not a bare `go run`/`make dev` left
running in the background. Before starting any server here, check
`/root/projects/MULTI-APP-GUIDE.md`'s port allocation map and `ss -ltnp | grep :4000` first — a
bare dev process on that port silently squats on production traffic instead of erroring, and
`kill`/`pkill` without checking what's actually listening has taken down another app's production
instance before. If you need an ad hoc dev instance, use a different `PORT` for it, or stop with
`server-manager.sh` (not a raw `kill`) and restart the same way when done.

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
CAP-A02, CAP-V06, CAP-A07, CAP-A08, CAP-X03, CAP-R02, CAP-A03, CAP-A04, CAP-A10) was manually
exercised end-to-end against a real Postgres instance before its conformance test was written, and
manual testing caught real bugs (a `Create` default-value rule hardcoded to fields named "Status"
that silently broke Approval Step's "Decision" field; a conformance-helper missing a cookie
parameter that made a test pass for the wrong reason; a hand-typed non-UUID reference value
crashing `RecordStore.Exists` with an unhandled Postgres error instead of a validation message; a
`NULLIF($4, '')` parameter bound against a `uuid` column with no cast, which would have crashed
every single `notify` action) that reading the code alone would not have surfaced.
