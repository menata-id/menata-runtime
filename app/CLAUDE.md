# CLAUDE.md

Guidance for Claude Code (or any AI agent) working in `app/` — the real Menata Runtime
application. Root `CLAUDE.md` (repo-wide orientation) still applies; this file is `app/`'s own
house patterns, mirroring the role `prototype/go/CLAUDE.md` plays for that codebase.

## What this is, right now

Blueprint stage (see `ARCHITECTURE.md` and `docs/decisions/001-graduation-from-prototype.md`). No
business logic has been ported from `prototype/go` yet — only a package skeleton (`internal/*/
doc.go`) and this documentation set. **Do not write real implementation code into these packages
without checking whether a development plan/roadmap for the port already exists** — this
blueprint was deliberately built stopping short of that (see `benchmarks/026-runtime-graduation-
decision.md` at repo root for why porting is a separate, later phase, not part of the blueprint
itself).

## The source of truth for "how we actually work," until code is ported

`../prototype/go/CLAUDE.md` carries this project's entire accumulated "caught live" gotcha catalog
— real bugs found and fixed, patterns established under real conformance-test pressure, dozens of
entries. None of it is duplicated here speculatively. **When a piece of `prototype/go` code is
actually ported into a package here, its own relevant `CLAUDE.md` entries graduate into this file
at the same time** — verbatim where the code didn't change, updated where the port changed
something (e.g. `internal/metadata`'s loader.go split, `internal/storage`'s new abstraction). Until
that happens for a given package, assume `prototype/go/CLAUDE.md`'s own entries about that
package's `prototype/go` counterpart still apply here too.

## Established pattern so far

Each `internal/<name>/doc.go` states, as of the blueprint pass: what it's graduated from (or "NEW"
for `internal/storage`), its one-sentence responsibility, and the one architectural change (if
any) this blueprint makes to it. Read the specific package's `doc.go` before touching it — don't
assume it's a verbatim copy without checking.
