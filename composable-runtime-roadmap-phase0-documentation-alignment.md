# Composable Runtime — Phase 0 Documentation Alignment Addendum

> This document is the Phase 0 documentation-alignment backlog for `composable-runtime-roadmap.md`.
>
> It records documentation and conceptual-contract work that must be completed before the composable runtime implementation advances beyond the semantic foundation. These are not cosmetic documentation tasks: they close architectural seams between the normative specification, concrete metadata schema, lifecycle, governance, NFRs, benchmarks, and implementation guidance.
>
> **Status:** All ten DOC items resolved 2026-09-11 (same day, one session) — see each item's own
> Status line below for what closed it. `CR-27`/`CR-28` (found by this session's own audit, not
> originally in this backlog) also resolved — see `composable-runtime-roadmap.md`.
> **Created:** 2026-09-11

---

# 1. Purpose

The composable transformation must first establish one coherent conceptual baseline across the repository.

The audit of `001`–`007` and the related concept/governance documents found several remaining seams:

1. Runtime Language terminology and ordering still needs one canonical relationship to Runtime Metadata.
2. `runtime-metadata-schema.md` still primarily represents the legacy Machine/View-oriented physical metadata shape while `004`/`006` now define a richer logical Domain/Data/Experience model.
3. Capability governance does not yet explicitly classify composable Domain/Data/Experience primitives.
4. NFRs do not yet define composition-level execution budgets.
5. Some README, agent guidance, benchmark, and practical-guide wording still describes the runtime primarily as an interpreter/View-oriented system.
6. The benchmark corpus needs an explicit second-generation conclusion around semantic normalization, dependency graphs, and execution planning.

These items belong to **Phase 0 of the composable runtime transformation** and must not be treated as optional editorial cleanup.

---

# 2. Documentation Alignment Backlog

## DOC-01 — Canonical Runtime Language → Runtime Metadata relationship

**Status:** Resolved 2026-09-11 — `003-runtime-language.md`'s own diagram contradicted its prose
(Metadata before Language vs. "Language is expressed through Metadata"); diagram and the
repo-boundary sentence corrected. `composable-runtime-architecture-map.md` already agreed with the
corrected direction.

**Priority:** P0

Audit `003-runtime-language.md` and all references to Runtime Language.

Canonical relationship should be unambiguous:

```text
Business Knowledge
      ↓
Menata Language
      ↓
Runtime Language / semantic contract
      ↓
Runtime Metadata
      ↓
Runtime Model
      ↓
IR / Dependency Graph / Plans
      ↓
Runtime Execution
```

If Runtime Language is treated as the semantic language represented by Runtime Metadata rather than a separate serialized layer, document that explicitly.

### Acceptance

- No document implies two contradictory directions between Runtime Language and Runtime Metadata.
- The relationship is consistent across `003`, `004`, `README`, guides, and architecture map.

---

## DOC-02 — Logical Runtime Model vs Concrete Metadata Schema

**Status:** Resolved 2026-09-11 — `runtime-metadata-schema.md` gained a "Composable Schema
Extensions" section explicitly framed as PROPOSED/concrete-representation, distinguishing itself
from the logical model in `004`/`006`, with a Data-plane YAML sketch.

**Priority:** P0

Audit `runtime-metadata-schema.md` against `004-runtime-metadata.md` and `006-runtime-model.md`.

The repository must explicitly distinguish:

```text
Logical Runtime Metadata Model
        ↓
Serialization / storage representation
        ↓
Current legacy-compatible schema
        ↓
Future composable schema extensions
```

The existing schema must not be presented as if it were the complete semantic Runtime Model.

The migration strategy should preserve the proven schema while allowing new artifacts such as:

- Dataset
- Relation
- Projection
- Query
- Layout
- Component
- Slot
- Binding

without forcing them into unrelated View or Machine fields.

### Acceptance

- `runtime-metadata-schema.md` explicitly states its role as concrete representation rather than the complete semantic model.
- Existing schema compatibility remains preserved.
- The migration path for composable artifacts is documented before CR-21 implementation.

---

## DOC-03 — Runtime Lifecycle terminology audit

**Status:** Resolved — `005-runtime-lifecycle.md` already matches the canonical Parse→Validate→
Normalize→IR→Dependency DAG→Planning→Execution/Rendering sequence, confirmed by a full-document
audit 2026-09-11. No edit needed.

**Priority:** P0

Audit `005-runtime-lifecycle.md` for language implying direct interpretation only.

Canonical realization terminology:

```text
Parse
 → Validate
 → Normalize / Resolve
 → Compile to internal IR
 → Build dependency graph
 → Plan
 → Execute
 → Render
```

Compilation here means runtime-internal compilation only. It does not mean application source-code generation.

### Acceptance

`005` consistently describes normalization, compilation, invalidation, reload, and incremental realization using the same terminology as `002`, `004`, `006`, and `007`.

---

## DOC-04 — README architecture terminology

**Status:** Resolved 2026-09-11 — clarifying sentences added at both "interpreted" passages, the
Runtime Layer diagram box expanded with an internal-stage summary, Tier 3 index and inline links
to the three `composable-runtime-*.md` docs added.

**Priority:** P0

Update `README.md` so the public-facing description does not reduce Menata Runtime to a direct metadata interpreter.

Replace conceptual wording such as:

```text
metadata → interpreter → application
```

with the canonical realization model:

```text
Runtime Metadata
      ↓
Normalize / Resolve
      ↓
Domain + Data + Experience IR
      ↓
Dependency Graph
      ↓
Execution Planning
      ↓
Physical Execution + Rendering
```

Retain the important statement that no application source code is generated.

Add links to:

- `composable-runtime-architecture-map.md`
- `composable-runtime-roadmap.md`
- this Phase 0 addendum

### Acceptance

README terminology does not conflict with Tier 1 architecture documents.

---

## DOC-05 — Agent/developer guidance terminology

**Status:** Resolved 2026-09-11 — root `CLAUDE.md` (loaded by every agent session) gained a
pointer to `003`/`007`'s target pipeline. `app/CLAUDE.md` needed no change (no execution-model
content to correct). `prototype/go/CLAUDE.md` deliberately left untouched — frozen historical
record per root `CLAUDE.md`'s own "discovery phase is done" paragraph, not a live gap.

**Priority:** P0

Audit `CLAUDE.md` and other agent/developer onboarding documents.

They must teach future implementation agents that the target architecture is:

```text
metadata
 → semantic model
 → IR
 → dependency graph
 → planner
 → physical execution
```

rather than:

```text
metadata
 → View interpreter
```

This is an implementation-safety requirement: incorrect architecture vocabulary in agent guidance can cause new code to reinforce the legacy View-centric design.

### Acceptance

No active developer/agent guidance describes the runtime as a direct View-oriented interpreter without the composable realization boundary.

---

## DOC-06 — Capability governance taxonomy

**Status:** Resolved 2026-09-11 — new Grammar area `D` (Data) added to `capability-registry.md`,
`capability-lifecycle.md` A3 and its proposal template, per `CR-27`. `CAP-V22`/`CAP-V23`
reclassified via pointer rows, IDs retained for stability. Relation/Projection/Query remain
correctly unadmitted (no forcing case yet) rather than forced into the old taxonomy.

**Priority:** P1

Audit `capability-lifecycle.md`, `capability-registry.md`, and related governance documents.

Introduce a canonical classification for composable semantic primitives:

```text
Domain
 ├── Machine
 ├── Field
 ├── Event
 ├── Action
 ├── Constraint
 └── Permission

Data
 ├── DataSource
 ├── Dataset
 ├── Relation
 ├── Projection
 ├── Dimension
 ├── Measure
 ├── Expression
 └── Query

Experience
 ├── Page
 ├── Layout
 ├── Component
 ├── View
 ├── Slot
 └── Binding
```

Do not automatically create one capability ID for every primitive. Governance must distinguish an architectural transformation from individual admitted capabilities.

### Acceptance

Composable implementation work can be tracked without forcing Dataset/Projection/Component/etc. into the old Field/Event/Action/Constraint/Permission/View taxonomy.

---

## DOC-07 — Composition-level NFR profile

**Status:** Resolved 2026-09-11 — `nfr-standards.md` gained "§1a. Composition-level budgets"
(logical/DAG nodes, physical operations, execution width, estimated rows, planner time, resource
budget per request), plus `### 2.11 Data (CAP-D*)` for the new Grammar area.

**Priority:** P0

Extend `nfr-standards.md` with a composition-level performance/resource profile.

At minimum define budgets or measurement dimensions for:

```text
logical nodes / request
DAG nodes / request
physical operations / request
execution width
estimated rows
planner time
DB time
CPU
memory
pool utilization
render time
```

The important invariant is:

> Logical composability must not imply proportional physical execution cost.

Composition-level budgets should complement, not replace, existing P1/P2/P3/P4/P5 request classes.

### Acceptance

CEP and composability benchmarks have explicit NFR targets to validate against.

---

## DOC-08 — Architecture benchmark second-generation conclusion

**Status:** Resolved 2026-09-11 — `architecture-benchmark.md` gained a "Second-Generation
Conclusion" section connecting semantic normalization, dependency graphs, and execution planning
into one lesson, appended after the original ten-bullet list (preserved as first-generation
evidence, not rewritten).

**Priority:** P1

Update `architecture-benchmark.md` with a post-benchmark synthesis.

The benchmark should no longer stop at the conclusion that declarative systems are interpreted/realized by a runtime.

It should explicitly capture the modern architectural lesson:

```text
Declarative Representation
        ↓
Semantic Normalization
        ↓
Dependency Graph
        ↓
Execution Planning
        ↓
Physical Realization
```

The critical lesson for Menata is that a composable semantic model needs an explicit planning boundary to prevent logical composition from turning into physical fan-out.

### Acceptance

The architecture benchmark supports the rationale for Data IR, UI IR, Dependency DAG, and CEP rather than merely documenting historical analogies.

---

## DOC-09 — Practical guides and benchmark cross-links

**Status:** Resolved 2026-09-11 — `guides/writing-runtime-metadata.md` gained a note at its View
section pointing to `composable-runtime-architecture-map.md`. `guides/runtime-metadata-gotchas.md`
and `guides/writing-process-overlays.md` needed no change (narrow/incidental View mentions only,
confirmed by audit).

**Priority:** P1

Search all active guides and benchmark documents for View-only composition assumptions.

Where appropriate, add a short reference to the composable architecture map and clearly mark historical/legacy examples as such.

Do not rewrite historical evidence merely to make it look current. Preserve historical conclusions and append current-status clarification where needed.

### Acceptance

No active implementation guide accidentally teaches View as the universal composition primitive.

---

## DOC-10 — Tier 1 / Tier 3 status boundary

**Status:** Resolved 2026-09-11 — this item was previously unverifiable because `007`'s own §40
citation matrix (its mechanism for the PROVEN/PROPOSED distinction) didn't exist (`CR-28`). §40
now exists. Verification pass against `composable-runtime-blueprint.md`, `composable-runtime-
roadmap.md`, `capability-registry.md`, `nfr-standards.md`, and `architecture-benchmark.md`: every
edit made across `DOC-01`–`DOC-09` above was written PROPOSED-not-admitted by construction (each
new section says so explicitly), so no new confusion between target and implementation was
introduced.

**Priority:** P1

Keep `007-composable-runtime-architecture.md` normative as target architecture while retaining its `PROVEN` / `PROPOSED` distinction.

Ensure supporting documents do not accidentally present proposed components such as UI IR or CEP as already implemented.

The architecture map, blueprint, roadmap, capability registry, NFRs, and benchmarks must preserve this distinction.

### Acceptance

A reader can always distinguish:

```text
normative target
vs.
current implementation
vs.
proven capability
vs.
proposed architecture
```

---

# 3. Recommended Execution Order

The documentation work should be completed in this order:

```text
DOC-01 Runtime Language relationship
        ↓
DOC-02 Logical Model vs Schema
        ↓
DOC-03 Lifecycle terminology
        ↓
DOC-04 README
        ↓
DOC-05 Agent/developer guidance
        ↓
DOC-06 Capability governance
        ↓
DOC-07 Composition NFR
        ↓
DOC-08 Architecture benchmark synthesis
        ↓
DOC-09 Guides / benchmark cross-links
        ↓
DOC-10 Status boundary verification
```

DOC-01 through DOC-05 establish the conceptual contract first. DOC-06 through DOC-10 then propagate that contract into governance, evidence, and implementation guidance.

---

# 4. Phase 0 Exit Criteria

Phase 0 documentation alignment is complete only when all of the following are true:

- [x] `001`–`007` use one coherent terminology for runtime compilation and composability;
- [x] Runtime Language and Runtime Metadata have one unambiguous relationship (`DOC-01`);
- [x] logical Runtime Model is clearly separated from concrete metadata/storage schema (`DOC-02`);
- [x] `005` describes normalization/compilation/planning consistently (`DOC-03`);
- [x] README and agent guidance teach the composable realization pipeline (`DOC-04`/`DOC-05`);
- [x] capability governance can classify Domain/Data/Experience primitives (`DOC-06`, `CR-27`);
- [x] NFRs define composition-level budgets (`DOC-07`);
- [x] architecture benchmark explains why explicit planning is required (`DOC-08`);
- [x] active guides no longer imply View is the universal composition primitive (`DOC-09`);
- [x] proposed vs proven vs implemented status remains explicit (`DOC-10`, `CR-28`).

**All ten criteria met as of 2026-09-11.** Per §5 below, the roadmap may now treat `CR-01`
(Canonical Semantic Model in code) as the next primary implementation step — subject to that
step's own forcing condition still being satisfied, which this addendum does not itself assert.

Only after this gate should the roadmap treat **CR-01 Canonical Semantic Model in code** as the next primary implementation step.

---

# 5. Relationship to the Main Composable Roadmap

This addendum belongs to **Phase 0 — Contract Freeze and Documentation Alignment** in `composable-runtime-roadmap.md`.

The main roadmap already contains initial Phase 0 checks for `004` and `005`. This addendum expands those checks into the complete documentation baseline required by the audit of `001`–`007` and related concept documents.

The implementation roadmap remains authoritative for phase ordering. This document is the detailed Phase 0 checklist and evidence target; it does not create a second implementation sequence.
