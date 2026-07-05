# ADR-004: Internal Package Architecture for Extensible Metadata-Driven Runtime

**Status:** Accepted — target architecture, migration is incremental (see Migration Strategy)
**Date:** 2026-07-05

## Context

The current `internal/` layout is flat — one package per concern (`config`, `constraint`, `db`,
`executor`, `handler`, `interpreter`, `metadata`, `model`, `permission`, `router`, `store`, `ui`),
mostly one file each. This was sufficient to validate Cases 1–2 (`design-request`, `leave-request`)
and prove the metadata-driven foundation.

It does not yet reflect two things the capability roadmap has since made concrete:

1. **Extension seams.** `capability-lifecycle.md` §4 ("Extension architecture — how the runtime
   grows") already sketches registries for field types, action types, constraint operators, event
   sources, view types, and workspace services. Today, adding a field type means editing a
   `switch` statement in `ui/components.templ` and `metadata/loader.go` — there is no registration
   point.
2. **NFR boundaries.** `nfr-standards.md` names cross-cutting security concerns (authn/session,
   metadata-injection firewall, confused-deputy checks) that have no dedicated home in the current
   layout — they would otherwise leak into `handler/` piecemeal.

## Decision

Adopt a layered `internal/` structure combining three proven patterns:

1. **Ports & Adapters (Hexagonal)** — the same architectural family as Portal GA v3's CBA/Clean
   Architecture (benchmarked in `benchmarks/002-portal-ga-cross-domain-survey.md`). Core domain
   packages depend on nothing infrastructural; infrastructure depends inward on the core.
2. **Registry-at-init seam** — Go's own idiom for this problem is `database/sql.Register`: a
   package-level registry that plugins populate via `init()` or explicit `Register()` calls. This
   is the concrete mechanism behind capability-lifecycle §4's sketched
   `fieldtype.Register("reference", ...)`.
3. **Consumer-side interfaces** — same decoupling rule validated in Portal GA's ADR-0012 (an
   integration interface is defined by the consumer, not the source, to prevent import cycles).
   Applied here: `engine/action` defines what an executor needs, `engine/action/builtin` supplies
   implementations — the core never imports a concrete built-in package by name.

### Target layout

```
internal/
├── core/                      # domain — zero infrastructure dependency
│   ├── model/                     Workspace/Application/Machine/Field/Event/... (from model/)
│   ├── interpreter/               builds Application Model from loaded metadata (from interpreter/)
│   ├── constraint/                engine + operator registry (from constraint/)
│   └── permission/                guard + role evaluation seam (from permission/)
│
├── engine/                    # extension seams — concretizes capability-lifecycle.md §4
│   ├── fieldtype/                 registry + interface { Render, Validate, Store }
│   │   └── builtin/                 text, date, value_list, reference (CAP-F13), ...
│   ├── action/                    registry + interface { Execute }
│   │   └── builtin/                 set_field, notify, activate_next (CAP-A07), ...
│   ├── eventsource/               registry — business, schedule (CAP-E02), external (CAP-E04)
│   ├── viewtype/                  registry — form, list, detail, report (CAP-V13)
│   └── wsservice/                 registry — calendar (CAP-O06), search (CAP-O04), ...
│
├── metadata/                  # authoring → runtime boundary — the most sensitive seam
│   ├── loader/                    reads Runtime Metadata from Postgres (from metadata/loader.go)
│   └── validator/                 NEW — CAP-X05, schema + injection validation at load time,
│                                   not render time (nfr-standards.md §0 "metadata is code")
│
├── store/                     # persistence, split by responsibility (CQRS-aligned)
│   ├── record_reader.go
│   └── record_writer.go
│
├── executor/                  # thin orchestrator: resolves action type from engine/action registry
│
├── security/                  # NEW — cross-cutting per nfr-standards.md: authn/session, RLS
│                               context propagation, CSRF, confused-deputy re-check
│
├── web/                       # delivery — no business logic
│   ├── httpapi/                   HTTP handlers + routing (from handler/, router/)
│   └── ui/                        Templ views (from ui/)
│
└── platform/                  # infrastructure adapters
    ├── db/                        Postgres connection (from db/)
    └── config/                    environment config (from config/)
```

## Migration Strategy

**No big-bang refactor.** The flat layout is not wrong today — it is simply not yet carrying the
weight this target structure is designed for. Moving code into `engine/fieldtype/` before a second
field type needs a registry would be premature abstraction, which this project's own principles
(`001-design-principles.md` — Infer Before Configure, Convention over Configuration) explicitly
warn against.

Migration is **capability-triggered**, following the registry's implementation order
(`capability-registry.md`):

| When | What moves |
|------|-----------|
| CAP-F13 (reference fields) implementation begins | Create `engine/fieldtype/` with the registry + interface; migrate existing field rendering into `engine/fieldtype/builtin/` as part of that same change |
| CAP-A07/A08 (workflow actions) implementation begins | Create `engine/action/` registry; migrate `set_field`/`notify` into `engine/action/builtin/` |
| CAP-E02 (time-driven events) implementation begins | Create `engine/eventsource/` registry |
| CAP-V13 (report views) implementation begins | Create `engine/viewtype/` registry; split `store/` into reader/writer at the same time (reports need read-only access) |
| CAP-O01…O06 (workspace services) implementation begins | Create `wsservice/` registry |
| CAP-X02 (real authentication) or CAP-X05 (metadata validation) implementation begins | Create `security/` and `metadata/validator/` respectively |

Each migration happens **as part of** implementing the triggering capability's Definition of Done
(`capability-lifecycle.md` §3, layer 4 "Application Model" / layer 5 "Engine") — not as a separate
restructuring task.

## Consequences

**Positive:**
- New capabilities are additive (new file in `builtin/`, one `Register()` call) rather than edits
  to existing `switch` statements — directly serves the ratchet rule (`capability-registry.md`):
  supported capabilities are less likely to regress when adding new ones.
- Matches the architectural family Portal GA already validates at production scale, easing any
  future cross-pollination between the two codebases.
- `security/` gives NFR gates (`capability-lifecycle.md` §3b) a concrete home instead of scattering
  checks across handlers.

**Negative:**
- More packages, more `go.mod`-internal import paths to reason about than the current flat layout.
  Mitigated by the fact that growth is incremental and capability-triggered, not upfront.
- Registry indirection (interface + `Register()` + lookup by string key) is one more layer to
  understand than a `switch` statement, for engineers unfamiliar with the pattern. Mitigated by
  using the same idiom as `database/sql` — a pattern most Go engineers already know.

## Compliance

- Registered as a Tier 1-adjacent decision in `runtime/README.md`'s document map (ADRs live under
  `prototype/go/docs/decisions/`, referenced from `prototype/go/docs/README.md`).
- Each future capability's Definition of Done (`capability-lifecycle.md` §3) should reference this
  ADR when its implementation triggers a migration step from the table above.
