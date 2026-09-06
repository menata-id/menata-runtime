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

## Established pattern so far

Each `internal/<name>/doc.go` states what it's graduated from (or "NEW" for `internal/storage`),
its one-sentence responsibility, and the one architectural change (if any) this port makes to it.
Read the specific package's `doc.go` before touching it — most are verbatim ports (see
`ARCHITECTURE.md`'s "What's graduated as-is"), but don't assume that without checking.
