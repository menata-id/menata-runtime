# Architecture Blueprint

Full decision record: [`../benchmarks/026-runtime-graduation-decision.md`](../benchmarks/026-runtime-graduation-decision.md)
(Study 34). This document is the blueprint itself — what gets built and why — not a restatement
of the decision process.

---

## Layers

The package layout graduates `prototype/go/internal/`'s own shape verbatim, because that shape has
been measured, not just asserted, to be clean (Study 34 §7 Method 2 — `go list` import graph,
zero cycles, Go-enforced):

```
                                   ┌──────────┐
                                   │ handler  │  orchestration — wires everything
                                   │ (router) │  below into HTTP request/response
                                   └────┬─────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    ▼                   ▼                   ▼
              ┌──────────┐        ┌──────────┐        ┌──────────┐
              │ executor │        │interpreter│       │permission│
              └────┬─────┘        └────┬─────┘        └────┬─────┘
                    │                   │                   │
        ┌───────────┼───────────┬───────┴───────┬───────────┘
        ▼           ▼           ▼               ▼
  ┌──────────┐┌──────────┐┌──────────┐   ┌──────────┐
  │constraint││ metadata ││    ui    │   │  storage │  (new)
  └────┬─────┘└────┬─────┘└────┬─────┘   └────┬─────┘
       │           │           │              │
       └───────────┴─────┬─────┴──────────────┘
                          ▼
          ┌───────────────────────────────┐
          │  model · store · auth · db ·  │  leaf layer — 0 internal/
          │           config              │  dependencies (measured)
          └───────────────────────────────┘
```

Each package's own `doc.go` (`internal/<name>/doc.go`) states its graduation source and
responsibility in one sentence — read those for the authoritative per-package detail; this
document stays at the layer/decision level.

## What's graduated as-is

`model`, `store` (minus file-upload bytes, see below), `auth`, `db`, `config`, `constraint`,
`permission`, `interpreter`, `router`, `ui`, `executor`, `handler` — all carried over from
`prototype/go/internal/<name>` with no architectural change. Each was independently confirmed
clean by Study 34 §7's own measurements (file size, concern count, import fan-in/fan-out); nothing
here needs redesigning to be worth building on.

## What's new

**`internal/storage`** — did not exist in `prototype/go`. Wraps uploaded-file bytes behind a small
interface (conceptually `Put`/`Get`/`Delete`, keyed by the same content-hash token
`prototype/go/internal/handler/upload.go` already generates), local-disk backend on day one
(behavior-identical to `prototype/go`'s own `uploads/`), swappable for an S3-compatible backend
later without touching any caller. Addresses Study 34 §5's file-storage finding (local-disk-only
storage blocks a multi-instance/multi-host future) without requiring object storage to exist yet —
the interface is what's new, not a forced migration.

## What's graduated, but restructured

**`internal/metadata`** — `prototype/go`'s own `loader.go` grew to 1,098 lines bundling three
separable concerns (Study 34 §7 Method 1, roadmap.md item 20). Ported as three files instead of
one, a pure move with no logic change:

| File | Concern |
|---|---|
| `loader.go` | Per-table `Load*` functions (the bulk of the original file) |
| `validate.go` | `validateOperators` / `validateReferences` |
| `compile.go` | `compileApprovalRequirements` / `injectApprovalQuorum` (already its own file — unchanged, just joins the split) |

## Named, not solved — deferred to the development plan

Study 34 §5's production-readiness gap inventory named several things this blueprint does not
close by itself. Recorded here so the development plan inherits a scoped list, not a blank page:

| Gap | Where it lands when addressed |
|---|---|
| Migrations are flat numbered `.sql` files, no version-tracking table, no rollback | A real migration tool (golang-migrate/goose/Atlas) wrapping `migrations/` once it's ported |
| Secrets in plaintext `.env` | `internal/config`'s own integration point (see its `doc.go`) |
| JSON API (`/api/{machine}`) has no version prefix, no OpenAPI spec | `internal/router`'s own versioning convention (see its `doc.go`) |
| No rate-limiting/DoS-protection middleware | `internal/handler`'s own middleware stack (see its `doc.go`) |
| Test coverage near-zero outside 3 pure-function files added 2026-09-05 | A coverage target set when the development plan sequences the port |
| Single-process in-memory metadata cache (CAP-X11/LISTEN-NOTIFY unbuilt) | Not a gap against the actual deployment target — see below |

## Client-side JavaScript policy

Core project principle, stated plainly in `prototype/go/docs/decisions/001-techstack.md`: **the
runtime owns application behavior, not the client.** For `app/`:

- **Default: no client-side JavaScript at all.** HTMX handles partial page updates server-side —
  most pages need nothing else, and HTMX itself is listed as a core tech-stack item above
  precisely because it's what makes the rest of this section short.
- **Sanctioned exception: Hyperscript**, for genuinely client-only UI concerns HTMX alone can't
  cover (modals, toggles, inline feedback) — deliberately constrained by design, so it discourages
  moving business logic to the client rather than merely relying on convention to prevent it.
- **Never: Alpine.js, or any reactive client-state framework.** `prototype/go`'s own ADR-001
  considered and explicitly rejected Alpine.js: "encourages reactive state management and
  client-side logic... conflicts with the principle that the runtime owns application behavior."
  That reasoning applies unchanged to `app/`.
- **Correction, not carried forward: plain vanilla `<script>` blocks.** `prototype/go` drifted
  into hand-written vanilla JS instead of its own ADR-001-chosen Hyperscript for every real
  client-behavior case it ended up needing (CAP-V16 typeahead, CAP-V15 live-sum preview, CAP-V14
  kanban drag-drop, CAP-V21 coordinate-placement) — undocumented anywhere until found live
  2026-09-06 while designing this policy (`prototype/go/ARCHITECTURE.md` still describes
  Hyperscript as if it were in use; it never was). Vanilla `<script>` has no structural guardrail
  against the exact complexity creep Hyperscript exists to prevent — the same risk profile as
  Alpine, just not yet exercised. `app/` reinstates the original ADR-001 decision: these cases
  port to real Hyperscript (`_="..."` attributes) when their code is ported, not vanilla
  `<script>` blocks carried over as-is.

## Deployment target: no containers, by constraint

This host runs no Docker/Kubernetes — a real RAM/CPU constraint from sharing a VPS with other
apps' own production instances, not a stylistic choice (`prototype/go/DEVELOPMENT.md`'s own "No
Docker/containers" note carries the same reasoning). Deployment stays a plain compiled binary
(`go build` + `server-manager.sh`), same pattern as `prototype/go`'s own `menata.app` deployment.
This is also why the interpreter's single-process in-memory cache (row above) isn't treated as a
gap needing a fix: there is no multi-instance deployment target for it to be a gap against.

## Verification

`cd app && go build ./... && go vet ./...` — passes clean against the current doc.go-only
skeleton, confirming the package structure is mechanically valid, not just prose. CI (mirroring
`prototype/go`'s own three GitHub Actions workflows — conformance, CSS gate, vet-test) is
deliberately not wired up yet; premature before real code exists to run.
