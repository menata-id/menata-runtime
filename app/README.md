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
- **[`ROADMAP.md`](ROADMAP.md)** — the development plan: what order the port happens in, and what
  "done" means at each phase.
- **[`docs/decisions/001-graduation-from-prototype.md`](docs/decisions/001-graduation-from-prototype.md)**
  — the ADR formalizing this move.
- **[`DEVELOPMENT.md`](DEVELOPMENT.md)** — setup, once real code lands here.
- **[`CLAUDE.md`](CLAUDE.md)** — dev patterns and gotchas (starts as a pointer back to
  `prototype/go/CLAUDE.md`'s own accumulated catalog; entries graduate here as their code does).

## Current status

Blueprint stage: real package skeleton (`internal/*/doc.go`, one per package, each stating its
graduated source and any architectural change), `go.mod`, and this documentation set exist.
No business logic has been ported yet — that starts at `ROADMAP.md`'s own Phase 0. `go build ./...`
and `go vet ./...` both pass clean against the current doc.go-only skeleton.

**Status update (2026-09-06):** `ROADMAP.md`'s Phase 0 is done. `model`, `auth`, `db`, `config`,
`store` ported verbatim (pure move, diffed byte-identical against `prototype/go` apart from
stripping two now-redundant package-doc comments already carried by `doc.go`) under
`menata.id/app`'s own import path; `internal/storage` (new) implements the `Store` interface with
a `LocalDisk` backend, extracted from `prototype/go/internal/handler/upload.go`'s
`storeFile`/`ServeFile` logic. `go build ./... && go vet ./...` clean, `internal/model/
model_test.go` ported unchanged and passing. `menata.app` still serves `prototype/go` — no Caddy
change yet, that's `ROADMAP.md`'s own Phase 6 (Cutover) decision point, not before. Next: Phase 1
(metadata pipeline).

**Status update (2026-09-06, Phase 1):** `internal/metadata` (split loader.go/validate.go/
compile.go + materialize.go) and `internal/interpreter` ported. Migration tool chosen: goose — all
24 `prototype/go/migrations/*.sql` converted to Up/Down pairs, every one verified to apply and roll
back cleanly against a real Postgres database. `internal/metadata`'s new
`TestLoadAllAgainstApprovalCase` confirms `Loader.LoadAll` builds a correct Application Model from
a freshly migrated + seeded database. See `ROADMAP.md`'s own Phase 1 status update and
`DEVELOPMENT.md`'s new "database setup" section for the how-to. `menata.app` still serves
`prototype/go` — still Phase 6's decision, not before. Next: Phase 2 (business logic layer).
