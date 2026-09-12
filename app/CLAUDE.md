# CLAUDE.md

Guidance for Claude Code (or any AI agent) working in `app/` — the real Menata Runtime
application. Root `CLAUDE.md` (repo-wide orientation) still applies; this file is `app/`'s own
house patterns, mirroring the role `prototype/go/CLAUDE.md` plays for that codebase.

## What this is, right now

**Status update (2026-09-06): the port is done — `app/ROADMAP.md`'s Phase 6 (Cutover) executed,
`menata.app` now serves `app/`'s own binary.** All six phases (leaf packages → metadata →
business logic → HTTP → conformance → CI → cutover) are complete, not just "Phases 0–3" as an
earlier revision of this file said. `ARCHITECTURE.md` is still the blueprint the port followed;
`docs/decisions/001-graduation-from-prototype.md` is the ADR; `ROADMAP.md`'s own per-phase status
updates are the detailed record of what actually happened at each step.

**What this means for new work now:** the "don't write code outside what the current phase calls
for" discipline that governed the port itself no longer applies — there is no more phase
sequence to stay inside. New work here is ordinary capability/feature development, governed by
this repo's own `capability-lifecycle.md` and root `CLAUDE.md`, the same way it would be for any
other real capability work — not roadmap-phase-constrained the way the graduation port was.
`README.md`'s own "Current status" section still has the authoritative history of how `app/` got
here, but no longer names a "phase in progress" to check before touching code.

## Server lifecycle — never kill/start the shared process ad hoc

**Status update (2026-09-12):** this has been gotten wrong more than once — an agent killing the
live process with a raw `kill $(cat /tmp/some.pid)` (or a bare `nohup ./bin/server &`) during a
manual smoke test, leaving it down for whoever hits `menata.app`/`aksi.menata.id` next, or for the
following session to discover cold. `menata-runtime`'s real instance (port 4000,
`aksi.menata.id`, cutover `ROADMAP.md` Phase 6) shares this host with `portal-ga3`, `iaabl2-*`,
and `menata-aksi` — it is one of several apps a single global script already manages, per-app, by
name:

```bash
go build -o bin/server ./cmd/server     # the script does NOT build for you — do this first
/root/scripts/server-manager.sh {start|stop|restart|status} menata-runtime
```

`status` (no app name needed for the bulk form) shows every app on the host at once, dev and
prod, so it's also the right first move before touching anything. `portal-ga3` is a **separate
application on the same script** (`... portal-ga3`, port 3003) — a `menata-runtime` restart must
never touch it, and vice versa; check `status` before and after to confirm only the intended app
moved.

**Never**: `kill`/`kill -9` against a PID you found yourself (`lsof`, a stashed `.pid` file, `ps
grep`), or a bare `nohup ... &` to bring it back up. The script's own stop function does a real
graceful SIGTERM-then-SIGKILL sequence with a wait loop, and its start function verifies
`/health` actually responds before declaring success — a manual kill/nohup pair skips both and is
exactly how this has gone wrong before.

**Never** add automatic/scheduled/self-healing restart logic (systemd `Restart=`, a cron
watchdog, a monitor script that restarts on crash-detect) — this host has a standing, explicit,
server-wide policy against it: `/root/docs/server-policies/NO-AUTO-RESTART-POLICY.md` (not
menata-runtime-specific; applies to every app `server-manager.sh` manages). A crash should surface
as a crash, investigated from logs, not get silently hidden by an auto-restart. Manual,
on-demand restart via the script above is the only supported mode on this host.

For a throwaway server that isn't the shared port-4000 instance at all — a one-off manual smoke
test, say — don't touch the shared instance or invent your own pid-file convention: follow
`scripts/local-ci.sh`'s own pattern instead (isolated Postgres schema + a server bound to a
scratch port, its own `trap cleanup EXIT INT TERM` tearing everything down automatically when the
script exits). That pattern already exists and already gets teardown right; a fresh ad-hoc
approach is what keeps reintroducing this mistake.

## The source of truth for "how we actually work" — still not fully migrated here

`../prototype/go/CLAUDE.md` carries this project's entire accumulated "caught live" gotcha catalog
— real bugs found and fixed, patterns established under real conformance-test pressure, dozens of
entries. **This still has not fully migrated here** (a named gap, not a silent one —
`README.md`'s own "How should a document be written?" says cite evidence, don't paper over a
gap): **until an entry actually graduates to this file, `prototype/go/CLAUDE.md`'s entries about
a now-ported package's `prototype/go` counterpart remain the authoritative pattern catalog for
the ported code here too** — the code is a verbatim/near-verbatim port, so its gotchas port with
it even though the documentation about them hasn't moved yet. Migrating the relevant entries here
(verbatim where the code didn't change, updated where the port changed something — e.g. `internal/
metadata`'s loader.go split, `internal/storage`'s new abstraction) is unfinished work, not a
decision that it's unnecessary — and now that new work isn't phase-gated, a new capability landing
in `app/` is exactly the moment to graduate the relevant entries alongside it, per
`capability-lifecycle.md`'s own loop.

## ui-sample: which mockups are live design references

`app/web/static/ui-sample/` accumulates every mockup ever explored, but `index.html`'s own
case-coverage table (Study 38) is the single, deliberate entry point — it says so directly ("No
separate top nav or preview cards — both were dropped as redundant duplicates of what this table
already covers"). **When asked to use a ui-sample mockup as a visual reference, only the file(s)
linked from that table's current row for the relevant case are valid** — never glob or `find` the
directory and treat whatever turns up as current design intent.

A `.html` file that still exists on disk but is **not** linked from the table (e.g., as of
2026-09-09: `groups.html`, `group-detail.html`, `group-approval.html`,
`group-approval-detail.html`, `quorum-approval.html`) is a superseded design pass kept only
because `benchmarks/029-composed-view-component-inventory.md` and/or `capability-registry.md`
cite specific markup inside it as component-inventory evidence — per this repo's append-don't-
rewrite convention (root `CLAUDE.md`), that citation stays valid even after the design itself is
superseded. Don't resurrect it as a design reference; don't delete it either.

For **Document Approval (Case 3)**, the current design references are exactly the four the index
table links: `document-submit.html`, `document-signature-placement.html`,
`document-approval.html`, `approval-dashboard.html`.

**Status update (2026-09-10):** `index.html` now has a second table above the case-coverage one,
"Workspace & Access screens" — platform-level screens (login, workspace membership, approval
authority) that aren't tied to any single `case-portfolio.md` case, so they don't belong in the
case table.

**Correction, same day:** this table does **not** follow the "don't re-mock what's already live"
rule the case-coverage table follows. Owner decision (2026-09-10, conversation): all six rows —
Login, Choose Workspace, Workspace Home, Workspace Members, Member Access Detail, Approval Role
Matrix — stay static mockups (`login.html`, `choose-workspace.html`, `workspace-home.html`,
`workspace-members.html`, `member-role-detail.html`, `approval-role-matrix.html`) even for the
four that already have live code (CAP-X02/O11/O03/O01). Reason given: the mockup is meant to stay
the design-intent baseline the live implementation gets diffed against — linking straight to the
live page would collapse that baseline into "whatever shipped," making drift unmeasurable. The
table's own "Drift" column names each mockup's live counterpart without linking to it as a
replacement. Document Approval (Case 3, case-coverage table) is the owner-cited concrete example
of this exact gap already having gone unmeasured — flagged in conversation 2026-09-10, not yet
written up with a citation anywhere.

**Status update (2026-09-12, real citation, owner-flagged recurring problem):** this gap is now
measured for at least one case, and the owner reports it recurring across multiple AI development
sessions specifically because agents keep treating the CURRENT, incomplete live rendering as the
design reference instead of checking `ui-sample` first. Concrete case: **Case 19 (Project
Management)**'s own `project-board.html` (linked from `case-19.html`, this table's own entry
point) is a rich Trello-like board — labels, members, drag-and-drop, checklist, activity — while
the real, live `/mch_pm_card/board` route (`internal/ui/board.templ`) renders `CAP-V14`'s own
deliberately narrower cut: fixed lane columns grouped by one `value_list`/`reference` field, no
labels, no members, no drag-and-drop. This is real, measured drift (`capability-registry.md`'s
own `CAP-V14 Tier 2` row already names the narrower cut as deliberate, not accidental), not a
comparison anyone had written down before.

**Load-bearing rule, stated explicitly so it stops recurring:** `composable-runtime-roadmap.md`'s
own "render-output equivalence" proofs (17e, 17i, 17m, and any cutover after them) prove that
migrating rendering logic onto the composable substrate does not change what the CURRENT code
already renders — an architecture-migration safety proof. They are **never** a claim that the
current rendering matches `ui-sample`'s own design intent, and must never be read or cited as
one. Closing the actual design gap against `project-board.html` (or any other case's own linked
mockup) is separate, real, unstarted work — none of 17a–17m attempted it, by design (matching
this repo's own `Filename case`/`append, don't rewrite` discipline: the gap is named here with a
citation, not silently closed by accident nor silently left unmeasured again).

**The complete, per-screen tracking document this rule demands now exists:**
`case-03-case-19-completion-checklist.md` (repo root) — every `ui-sample` screen for both trial
applications (Case 3, Case 19), each checked directly against that mockup file (not the current
code), with classic-capability status and composable status tracked as two separate columns, and
a staged execution order. Read it before describing either trial case as "done" in any sense.

## Established pattern so far

Each `internal/<name>/doc.go` states what it's graduated from (or "NEW" for `internal/storage`),
its one-sentence responsibility, and the one architectural change (if any) this port makes to it.
Read the specific package's `doc.go` before touching it — most are verbatim ports (see
`ARCHITECTURE.md`'s "What's graduated as-is"), but don't assume that without checking.
