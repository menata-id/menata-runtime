# Menata Runtime (Application)

**This is the real Menata Runtime application** — not another prototype. It is being built by
graduating [`prototype/go`](../prototype/go/)'s own proven codebase (~90 capabilities ✅, a
219-test conformance suite, an architecture independently vetted twice) rather than starting from
zero, per the decision recorded in
[`../benchmarks/026-runtime-graduation-decision.md`](../benchmarks/026-runtime-graduation-decision.md)
(Study 34).

`../prototype/` (all seven platform prototypes) and `../benchmarks/` (the capability-discovery
evidence series) have done their job — proving which capabilities a runtime needs and that a real
implementation of them is possible. They stay exactly as they are, unrenamed, unrestructured, kept
as the historical record of that process. This folder is what that process was building toward.

## Where to start

- **[`ARCHITECTURE.md`](ARCHITECTURE.md)** — the blueprint: package layout, what's graduated as-is,
  what's new, what's deferred and why.
- **[`docs/decisions/001-graduation-from-prototype.md`](docs/decisions/001-graduation-from-prototype.md)**
  — the ADR formalizing this move.
- **[`DEVELOPMENT.md`](DEVELOPMENT.md)** — setup, once real code lands here.
- **[`CLAUDE.md`](CLAUDE.md)** — dev patterns and gotchas (starts as a pointer back to
  `prototype/go/CLAUDE.md`'s own accumulated catalog; entries graduate here as their code does).

## Current status

Blueprint stage: real package skeleton (`internal/*/doc.go`, one per package, each stating its
graduated source and any architectural change), `go.mod`, and this documentation set exist.
No business logic has been ported yet — that is the next, separate phase (a development plan and
roadmap, built on top of this blueprint, not part of it). `go build ./...` and `go vet ./...`
both pass clean against the current doc.go-only skeleton.
