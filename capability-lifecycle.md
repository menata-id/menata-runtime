# Capability Lifecycle Governance

> Study 9 deliverable (`capability-roadmap.md`) — the closing artifact of Phase 2.
>
> Answers: when a **new capability** is proposed — how do we test whether it
> deserves admission, design it completely, and grow the runtime architecture
> to absorb it without bloating the core?
>
> Status: v0.1 | Created: 2026-07-04

Reference implementations studied: Portal GA's constitutional stack (fitness functions in CI, ARB decision log, living registries, amendment process — Study 5) and the VS Code small-core lesson (`architecture-benchmark.md`).

---

# 1. Lifecycle states

```text
PROPOSED ──admission test──► ADMITTED ──implementation──► INCUBATING ──conformance──► SUPPORTED
    │                            │                             │                          │
    ▼                            ▼                             ▼                          ▼
 REJECTED                    (parked in                  (feature-flagged,           (ratchet rule:
 (reason recorded)            registry ❌                 metadata schema             may never regress;
                              with Prio)                  may still change)           deprecate only)
```

- **Proposed** — named in a survey/case but not yet admitted (may carry `evidence-thin` note).
- **Admitted** — passed the admission test; registered ❌ with a priority.
- **Incubating** — implemented behind a workspace-level feature flag; metadata schema for it may still change.
- **Supported** — conformance test passing; ratchet rule applies (`capability-registry.md` Rules).
- **Deprecated** — replaced or withdrawn; metadata keeps loading with a warning until sunset (mirrors the event `deprecation` block from Portal GA's canonical schema).

---

# 2. Admission test — is it worthy?

All five criteria must hold. Any failure → stays Proposed (with the failing criterion recorded).

| # | Criterion | Test |
|---|-----------|------|
| A1 | **Dual evidence** | Named by ≥2 *independent* sources — at minimum one case (terrain) and one benchmark (map). A single enthusiastic source is a hypothesis, not a capability. |
| A2 | **Universality or declared verticality** | Either most platforms/patterns have it (table stakes), or it is explicitly scoped to a vertical (e.g. CAP-C10 double-entry) — never "we might need it someday". |
| A3 | **Single responsibility within Grammar** | Maps to exactly one Grammar area (Field/Event/Action/Constraint/Permission/View) or one declared cross-cutting area (Integration, Workspace Services). If it needs two, it is two capabilities. |
| A4 | **Non-composability** | Cannot be built by composing existing supported capabilities. (Study 5 showed ADR-0012 Patterns A & B compose — only Pattern C was admitted as new.) |
| A5 | **Business language exists** | A domain expert can say it in `.menata`-style business language. If only an engineer can phrase it, it belongs to the runtime's internals, not to a capability the metadata exposes. |

---

# 3. Definition of done — is it whole?

A capability is a **column through every layer**, not a feature in one. Each layer is implemented or *explicitly deferred with a reason* — silence is not a decision.

| Layer | Deliverable | Example (CAP-F13 reference field) |
|-------|------------|-----------------------------------|
| 1. Language | Grammar/examples updated (`specification/`, `guides/writing-menata.md`) | `- Department : Department` already specified |
| 2. Metadata schema | Schema doc + migration (`runtime-metadata-schema.md`, `migrations/`) | `target_machine` in field options |
| 3. Loader | Parse + validate (dangling refs = load-time error, not runtime surprise) | resolve target machine at load |
| 4. Application Model | Model structs + interpreter queries | `Field.TargetMachine` |
| 5. Engine | Executor / constraint / permission / store behavior | `ListByParent`, referential integrity |
| 6. UI | Renderer for every affected view type | picker in forms, link in detail, sub-list |
| 7. Conformance | Test(s) in `conformance/run.sh`; negative cases too | create-with-reference, dangling-reference rejected |
| 8. Docs | Translation guide row (`writing-runtime-metadata.md`), example case exercises it | — |
| 9. Registry | Row updated: status, proof, discovered-by | — |

---

# 3b. NFR gates — is it world-class?

Layers 1–9 prove a capability *works*. Three additional gates prove it is **safe, fast, and sound** — evaluated at implementation time against the area profile in `nfr-standards.md`:

| Gate | Standard | Evidence required |
|------|----------|-------------------|
| **Security** | Area threat profile (STRIDE) + ASVS chapter refs (`nfr-standards.md` §2.x) | Negative conformance tests (forbidden action rejected), or fitness function, or explicit waiver with reason in the registry row |
| **Performance** | Declared budget class P1–P5 (`nfr-standards.md` §1) | Measurement against the Study 8 load-test matrix, or budget declaration + deferred measurement with reason |
| **Architecture** | Seam discipline (§4 below) + area architecture profile | Capability lives behind its registry seam; fitness function where greppable |

A capability may enter **Incubating** without passing the gates; it may not reach **Supported** until each gate is satisfied or explicitly waived. Silence is not a decision.

---

# 4. Extension architecture — how the runtime grows

**Small core, registries at every seam.** Each engine exposes a registration point; capabilities plug in rather than patch the core:

| Seam | Registry | Example plug-in |
|------|----------|----------------|
| Field types | `fieldtype.Register("reference", renderer, validator, storer)` | CAP-F13, F16, F17 |
| Action types | `action.Register("aggregate_status", executor)` | CAP-A07/A08; **domain engines** (Study 6's posting derivation) plug here beneath declarative metadata |
| Constraint operators | `operator.Register("sum_equals", evalFn)` | CAP-C10 |
| Event sources | `eventsource.Register("schedule", scheduler)` | CAP-E02/E03/E04 |
| View types | `viewtype.Register("report", renderer)` | CAP-V13; CAP-V11 = second *render target* per view type |
| Workspace services | `wsservice.Register("calendar", svc)` | CAP-O01…O06 |

Rules that keep this honest:

1. **Versioned metadata schema** — metadata declares `version`; the loader applies per-version interpretation (already stated in `runtime-metadata-schema.md`).
2. **Backward compatibility** — old metadata MUST keep loading on a new runtime. A breaking change requires a deprecation cycle (flag → warn → sunset date), never a silent semantic change. Mirrors the Language spec's Compatibility clause.
3. **Unknown = explicit** — an unrecognized type/operator is a *load-time report* ("unsupported capability X"), per the Language spec's conformance rule ("unsupported Grammar should be reported explicitly"). Never a silent skip.
4. **Incubation flags** — Incubating capabilities activate per workspace, so schema iteration never breaks stable tenants.

---

# 5. Proposal template

```markdown
## Capability Proposal: <name>
- Grammar area: <one of F/E/A/C/P/V/R/X/I/O>
- Evidence (≥2 independent): <case + benchmark refs>
- Business language: <how a domain expert says it in .menata>
- Non-composability: <why existing capabilities cannot express it>
- Universality or vertical scope: <platforms/pattern refs, or declared vertical>
- Sketch per layer (1–9): <one line each; "deferred: <reason>" allowed>
- Conformance sketch: <what test would prove it; negative case included>
```

Admission is recorded in the registry row (`Discovered by` + notes). Rejections are recorded too — a rejected proposal that returns with new evidence restarts at A1, not from memory.

---

# 6. Retrofit calibration

The admission test applied retroactively to three registered capabilities — verifying the test is neither vacuous (everything passes) nor impossible (nothing passes):

| Capability | A1 dual evidence | A2 | A3 | A4 | A5 | Verdict |
|------------|------------------|----|----|----|----|---------|
| CAP-F16 line items | ✅ Study 6 benchmark + Case 9 declaration | ✅ universal to document apps | ✅ Field | ✅ N loose records ≠ one committed document | ✅ "an entry has lines" | **PASS** — correctly admitted |
| CAP-A11 date arithmetic | ✅ Case 4 finding + spec 003 date events | ✅ scheduling/SLA/accounting all need it | ✅ Action | ✅ no existing value expression | ✅ "advance by frequency" | **PASS** — correctly admitted |
| CAP-V11 channel-independent rendering | ❌ **single source** (Portal GA only) | ✅ | ✅ View | ⚠️ possibly composable: one view + multiple render targets at the runtime seam | ✅ | **HOLD at Proposed** — evidence-thin; needs a second independent source (e.g. a case whose digest requirement arises natively) before implementation |

Calibration result: 2 pass, 1 correctly caught — the test discriminates. CAP-V11's registry row is annotated `evidence-thin`.

---

# 7. Relationship to the other artifacts

```text
case-portfolio.md            → produces evidence (terrain)
benchmarks/*                 → produces evidence (map)
        │
        ▼  admission test (§2)
capability-registry.md       → single source of record, lifecycle state per row
        │
        ▼  definition of done (§3) + extension architecture (§4)
implementation + conformance/run.sh
        │
        ▼  ratchet
capability-roadmap.md        → phase planning over the whole loop
```
