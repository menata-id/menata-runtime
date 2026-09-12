# Composable Runtime — Phase 0 Documentation Alignment Addendum

> This document was the Phase 0 documentation-alignment backlog for `composable-runtime-roadmap.md`.
>
> **Status:** Closed. All ten DOC items resolved 2026-09-11 (same day, one session) — see the table
> below for what each one changed. `CR-27`/`CR-28` (found by this session's own audit, not
> originally in this backlog) also resolved — see `composable-runtime-roadmap.md`. Phase 0 itself
> is closed (`composable-runtime-roadmap.md` §4's own Exit Criteria).
> **Created:** 2026-09-11
> **Condensed 2026-09-12:** with every item resolved and the actual corrections already living in
> the target documents themselves, the original per-item audit instructions and Acceptance criteria
> (a "before" checklist for work that is now "after") no longer carry forward-looking value. This
> revision keeps only what each item actually changed and where. Full original text (the audit
> instructions, canonical-relationship diagrams, and per-item Acceptance criteria) is in git
> history for this file.

---

# 1. Purpose

The composable transformation needed one coherent conceptual baseline across the repository before
implementation began. An audit of `001`–`007` and the related concept/governance documents found
six seams — Runtime Language ↔ Runtime Metadata direction, schema vs. logical model, capability
governance taxonomy, composition-level NFRs, interpreter-only wording in README/agent guidance, and
a missing second-generation architecture-benchmark conclusion. These became the ten items below.

---

# 2. Documentation Alignment Backlog — resolutions

| ID | Priority | Addressed | Resolution |
|----|----------|-----------|------------|
| DOC-01 | P0 | Runtime Language → Runtime Metadata direction | `003-runtime-language.md`'s diagram contradicted its own prose; diagram and repo-boundary sentence corrected to match `composable-runtime-architecture-map.md` |
| DOC-02 | P0 | Logical Runtime Model vs. concrete metadata schema | `runtime-metadata-schema.md` gained a "Composable Schema Extensions" section, explicitly PROPOSED/concrete-representation, distinct from the logical model in `004`/`006`, with a Data-plane YAML sketch |
| DOC-03 | P0 | Runtime Lifecycle terminology | `005-runtime-lifecycle.md` already matched the canonical Parse→Validate→Normalize→IR→Dependency DAG→Planning→Execution/Rendering sequence — full-document audit found no edit needed |
| DOC-04 | P0 | README architecture terminology | Clarifying sentences added at both "interpreted" passages; Runtime Layer diagram box expanded with an internal-stage summary; Tier 3 index and inline links to the three `composable-runtime-*.md` docs added |
| DOC-05 | P0 | Agent/developer guidance terminology | Root `CLAUDE.md` (loaded by every agent session) gained a pointer to `003`/`007`'s target pipeline. `app/CLAUDE.md` needed no change. `prototype/go/CLAUDE.md` deliberately left untouched — frozen historical record |
| DOC-06 | P1 | Capability governance taxonomy | New Grammar area `D` (Data) added to `capability-registry.md` and `capability-lifecycle.md` A3/proposal template, per `CR-27`. `CAP-V22`/`CAP-V23` reclassified via pointer rows, IDs retained for stability |
| DOC-07 | P0 | Composition-level NFR profile | `nfr-standards.md` gained "§1a. Composition-level budgets" (logical/DAG nodes, physical operations, execution width, estimated rows, planner time, resource budget per request) plus `§2.11 Data (CAP-D*)` |
| DOC-08 | P1 | Architecture benchmark second-generation conclusion | `architecture-benchmark.md` gained a "Second-Generation Conclusion" section connecting semantic normalization, dependency graphs, and execution planning — appended after the original ten-bullet list, which was preserved, not rewritten |
| DOC-09 | P1 | Practical guides / benchmark cross-links | `guides/writing-runtime-metadata.md` gained a note at its View section pointing to `composable-runtime-architecture-map.md`. `guides/runtime-metadata-gotchas.md` and `guides/writing-process-overlays.md` needed no change (audit confirmed only narrow/incidental View mentions) |
| DOC-10 | P1 | Tier 1 / Tier 3 PROVEN/PROPOSED status boundary | Previously unverifiable because `007`'s own §40 citation matrix didn't exist (`CR-28`); §40 now exists. Verification pass confirmed every DOC-01–DOC-09 edit was written PROPOSED-not-admitted by construction |

Recommended execution order (as actually followed, same session): DOC-01 → DOC-02 → DOC-03 →
DOC-04 → DOC-05 → DOC-06 → DOC-07 → DOC-08 → DOC-09 → DOC-10 — DOC-01 through DOC-05 established
the conceptual contract first; DOC-06 through DOC-10 propagated it into governance, evidence, and
implementation guidance.

---

# 3. Relationship to the Main Composable Roadmap

This addendum belonged to **Phase 0 — Contract Freeze and Documentation Alignment** in
`composable-runtime-roadmap.md`, expanding that roadmap's own initial Phase 0 checks for `004`/`005`
into the complete documentation baseline the `001`–`007` audit required. The implementation roadmap
remains authoritative for phase ordering; this document was the detailed Phase 0 checklist and
evidence target, not a second implementation sequence.
