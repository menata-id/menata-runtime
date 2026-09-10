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
case table. Same "don't re-mock what's already live" rule applies there: it links straight to the
real running routes for Login (`/login`), Choose Workspace (`/choose-workspace`, CAP-O11),
Workspace Home (`/`, CAP-O03) and Workspace Members (`/{wsSlug}/admin/users`, CAP-O01) instead of
static mockups. Only `member-access-detail.html` and `approval-role-matrix.html` are real static
mockups in this second table — added because no live page fills either shape yet.

## Established pattern so far

Each `internal/<name>/doc.go` states what it's graduated from (or "NEW" for `internal/storage`),
its one-sentence responsibility, and the one architectural change (if any) this port makes to it.
Read the specific package's `doc.go` before touching it — most are verbatim ports (see
`ARCHITECTURE.md`'s "What's graduated as-is"), but don't assume that without checking.
