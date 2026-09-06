# CLAUDE.md

Guidance for Claude Code (or any AI agent) working in `app/` — the real Menata Runtime
application. Root `CLAUDE.md` (repo-wide orientation) still applies; this file is `app/`'s own
house patterns, mirroring the role `prototype/go/CLAUDE.md` plays for that codebase.

## What this is, right now

Past blueprint stage — real code is landing per `ROADMAP.md`'s phased plan (`ARCHITECTURE.md` is
the blueprint it follows; `docs/decisions/001-graduation-from-prototype.md` is the ADR). **For
the current phase, read `README.md`'s own "Current status" section** — the one place updated
every time a phase lands; don't trust a phase number stated anywhere else, including this file,
to still be accurate. Do not write new implementation code outside what `ROADMAP.md`'s current
phase calls for without checking it first — the phases are sequenced deliberately (leaf packages
→ metadata → business logic → HTTP → conformance → CI → cutover), each depending on the last.

## The source of truth for "how we actually work" — not yet migrated here

`../prototype/go/CLAUDE.md` carries this project's entire accumulated "caught live" gotcha catalog
— real bugs found and fixed, patterns established under real conformance-test pressure, dozens of
entries. **Status update (2026-09-06): this has NOT kept pace with the port.** `README.md`'s own
status updates show `model`/`auth`/`db`/`config`/`store`/`internal/storage`/`internal/metadata`/
`internal/interpreter`/`constraint`/`permission`/`executor`/`internal/ui`/`internal/router`/
`internal/handler`/`cmd/server` are all ported (Phases 0–3 done) — but this file's own entry set
is still empty; no `prototype/go/CLAUDE.md` entry has actually graduated yet. This is a named gap,
not a silent one (`README.md`'s own "How should a document be written?" — cite evidence, don't
paper over a gap): **until this file has its own entries, `prototype/go/CLAUDE.md`'s entries
about a now-ported package's `prototype/go` counterpart remain the authoritative pattern catalog
for the ported code here too** — the code is a verbatim/near-verbatim port, so its gotchas port
with it even though the documentation about them hasn't moved yet. Migrating the relevant entries
here (verbatim where the code didn't change, updated where the port changed something — e.g.
`internal/metadata`'s loader.go split, `internal/storage`'s new abstraction) is unfinished work,
not a decision that it's unnecessary.

## Established pattern so far

Each `internal/<name>/doc.go` states what it's graduated from (or "NEW" for `internal/storage`),
its one-sentence responsibility, and the one architectural change (if any) this port makes to it.
Read the specific package's `doc.go` before touching it — most are verbatim ports (see
`ARCHITECTURE.md`'s "What's graduated as-is"), but don't assume that without checking.
