# Menata Runtime

**`app/` (live at `menata.app`):**
[![App conformance](https://github.com/menata-id/menata-runtime/actions/workflows/app-conformance.yml/badge.svg)](https://github.com/menata-id/menata-runtime/actions/workflows/app-conformance.yml)
[![App CSS gate](https://github.com/menata-id/menata-runtime/actions/workflows/app-css-gate.yml/badge.svg)](https://github.com/menata-id/menata-runtime/actions/workflows/app-css-gate.yml)
[![App vet/test](https://github.com/menata-id/menata-runtime/actions/workflows/app-vet-test.yml/badge.svg)](https://github.com/menata-id/menata-runtime/actions/workflows/app-vet-test.yml)

**`prototype/go` (historical reference):**
[![Conformance](https://github.com/menata-id/menata-runtime/actions/workflows/conformance.yml/badge.svg)](https://github.com/menata-id/menata-runtime/actions/workflows/conformance.yml)
[![CSS gate](https://github.com/menata-id/menata-runtime/actions/workflows/css-gate.yml/badge.svg)](https://github.com/menata-id/menata-runtime/actions/workflows/css-gate.yml)
[![Vet/test](https://github.com/menata-id/menata-runtime/actions/workflows/vet-test.yml/badge.svg)](https://github.com/menata-id/menata-runtime/actions/workflows/vet-test.yml)

> **Applications should evolve at the pace of Business Knowledge.**
>
> **A runtime that realizes Business Knowledge as living applications.**

## Overview

Business Knowledge explains how organizations work.

Menata provides a language for expressing that knowledge.

**That language itself — Menata Language, `.menata`, and how to write Business Knowledge in
it — lives in a separate repository: [`menata-id/menata`](https://github.com/menata-id/menata).**
That repo owns the business process language layer only, with no machine or application
concerns. Everything in *this* repo (`menata-runtime`) starts one layer downstream — realizing
Business Knowledge as Runtime Metadata, and Runtime Metadata as a living application.

However, Business Knowledge alone does not become software.

Applications must be realized.

Pages.

Navigation.

Forms.

Dashboards.

Services.

Authentication.

Authorization.

Notifications.

Scheduling.

Integrations.

Search.

APIs.

These concerns belong to the runtime.

**Menata Runtime exists to realize Business Knowledge as living applications.**

Applications are not manually programmed.

Applications are interpreted from Runtime Metadata.

Applications evolve because Business Knowledge evolves.

---

> ⚠️ **Research Draft**
>
> Menata Runtime is an active research project.
>
> Runtime architecture, Runtime Language, Runtime Metadata, and the application engine are evolving continuously.
>
> Breaking changes are expected before version **1.0**.

---

# Why Menata Runtime?

Organizations continuously create new Business Knowledge.

Unfortunately, software rarely evolves at the same pace.

Every business change usually requires:

- redesign,
- implementation,
- testing,
- deployment,
- maintenance.

Over time, organizations accumulate far more Business Knowledge than they can realistically implement as software.

The problem is no longer capturing Business Knowledge.

The problem is realizing Business Knowledge into working applications.

Menata Runtime exists to solve that problem.

Instead of manually building every application, Runtime Metadata describes application intent.

The runtime continuously realizes that metadata into living applications.

---

# Vision

We believe every Business Knowledge deserves implementation.

Applications should evolve at the pace of Business Knowledge.

Business should drive software.

Not the other way around.

The runtime should:

- continuously realize Business Knowledge,
- minimize handwritten application code,
- maximize metadata reuse,
- remain independent from implementation technology,
- evolve without requiring applications to be rewritten.

Business Knowledge remains stable.

The runtime evolves.

Applications continuously evolve.

---

# Architecture

```text
Business Reality
        │
        ▼
Business Knowledge
        │
        ▼
Menata Language
        │
        ▼
──────────────────────────────
Authoring Layer
──────────────────────────────

Menata Apps Builder
Visual Builder
CLI
Manual Editor
Any Compatible Tool

        │
        ▼

Runtime Metadata

──────────────────────────────
Runtime Layer
──────────────────────────────

Menata Runtime

        │
        ▼

Applications
```

Business Reality explains what actually happens.

Business Knowledge explains why it happens.

Menata Language formally expresses Business Knowledge.

Runtime Metadata expresses how applications should be realized.

Menata Runtime realizes Runtime Metadata into executable applications.

---

# What is Menata Runtime?

Menata Runtime is a metadata-driven application runtime.

It interprets Runtime Metadata into complete applications.

A running application may include:

- pages,
- forms,
- tables,
- dashboards,
- workflows,
- navigation,
- menus,
- APIs,
- background jobs,
- notifications,
- authentication,
- authorization,
- search,
- integrations,
- platform services.

Applications are interpreted.

Applications are not generated.

Runtime Metadata plays a role similar to HTML in a web browser.

The runtime does not care how Runtime Metadata was created.

---

# Runtime Metadata

Runtime Metadata describes application realization.

It is designed primarily for deterministic machine interpretation.

Runtime Metadata may be produced by:

- Menata Apps Builder,
- Visual Builders,
- Command-line tools,
- Manual editors,
- AI-assisted tools,
- Any compatible implementation.

Menata Runtime only interprets Runtime Metadata.

It never depends on how the metadata was authored.

See `benchmarks/018-menata-apps-builder-concept.md` for an early page-concept exploration of what
a Menata Apps Builder could contain — exploratory only, no runtime dependency implied.

---

# Metadata-Driven Applications

Applications are described by Runtime Metadata rather than handwritten application source code.

Changing Runtime Metadata changes applications.

No code generation is required.

No scaffolding is required.

No duplicated CRUD implementation is required.

A single runtime may realize:

- one application,
- dozens of applications,
- hundreds of applications,
- thousands of applications.

All applications live inside the same runtime.

Applications are isolated by metadata.

Not by runtime instances.

---

# Runtime Responsibilities

Menata Runtime is responsible for:

- interpreting Runtime Metadata,
- realizing applications,
- application lifecycle,
- routing,
- navigation,
- rendering,
- authentication,
- authorization,
- event execution,
- constraint enforcement,
- search,
- scheduling,
- notifications,
- API exposure,
- application hosting,
- platform services.

Business Knowledge remains independent from runtime implementation.

---

# Design Principles

Menata Runtime is built upon the following principles.

## Core Principles

- Machine First
- Runtime First
- Metadata First
- Declarative

## Architecture Principles

- Convention over Configuration
- Infer Before Configure
- Composable
- Reference over Duplication
- Workspace Isolation

## Evolution Principles

- Live Evolution
- Data Preservation
- Long-term Compatibility
- Technology Adaptable

## Platform Principles

- Single Runtime
- Open Platform
- Compatible Authoring

## Vision

Applications evolve at the pace of Business Knowledge.

The complete rationale is available in:

[`001-design-principles.md`](001-design-principles.md)

---

# Long-term Vision

Menata Runtime aims to become a universal metadata-driven application runtime.

A single runtime should realize one application or thousands of independent applications.

Applications should continuously evolve without rewriting application source code.

Business Knowledge should remain stable.

Runtime Metadata should evolve.

The runtime should evolve.

Applications should continuously reflect Business Knowledge.

---

# The Menata Ecosystem

The Menata ecosystem consists of independent open-source projects.

## Menata

Business Knowledge Language.

Designed for humans.

Defines Business Knowledge.

## Menata Runtime

Application Runtime.

Designed for machines.

Realizes Runtime Metadata into applications.

---

# Documentation

This directory mixes documents that change on different rhythms. To read the numbering correctly:

- **Numbered `001`–`006`** — the core specification. Stable, read in sequence, changes rarely (mirrors `specification/000`–`006` at the repo root — same convention, one level down).
- **Unnumbered, at this level** — supporting reference or governance documents. Two different kinds, distinguished below.
- **`benchmarks/` — numbered `000`–`028`** — evidence studies, numbered by production order (Study 1 → `000`, Study 2 → `001`, …), not a required reading sequence. Grows as new studies land; check `ls benchmarks/` for the current top of the range rather than trusting this exact number to stay current.

This stable/evolving split mirrors the pattern Portal GA v3 uses for its own domain-integration framework (`01-CONSTITUTIONAL-BRD.md` + `02-IMPLEMENTATION-GUIDE.md` as STABLE, `appendices/` as EVOLVING) — see `benchmarks/002-portal-ga-cross-domain-survey.md`.

**Filename case is also a signal, not an accident.** `README.md`/`ARCHITECTURE.md`/`DEVELOPMENT.md`/
`ROADMAP.md`/`CLAUDE.md` — ALL CAPS — are the fixed onboarding-doc set every codebase in this repo
repeats at its own top level: this repo's own root, `app/`, and each `prototype/{go,drupal,frappe,
directus,budibase,salesforce,camunda}/`. Not every one carries all five (`prototype/frappe/` has
only `README.md`), but none renames one or adds a sixth — this mirrors the standard OSS convention
of capitalizing repo "community health" files (README/LICENSE/CONTRIBUTING/CHANGELOG) so they sort
together and stand out in a directory listing. Every other `.md` in this repo — root's own
governance/reference docs, `guides/*.md`, and every numbered doc — is lowercase-kebab-case. This
produces one real trap worth naming explicitly: **root `roadmap.md` and `app/ROADMAP.md` are
different documents that differ only by case** (discovery method/evidence log vs. the actual
`app/` code-port sequence — see root `CLAUDE.md`'s dedicated section). Always resolve a reference
by full path, never by basename alone. Numbered docs additionally follow a shared
three-digit-prefix + kebab-slug shape across three unrelated families — Tier 1 specs (reading
order), `benchmarks/` (production order, not reading order), and every `*/docs/decisions/NNN-*.md`
(ADRs, the standard Nygard/MADR format, decision order, independently numbered per codebase — e.g.
`app/docs/decisions/001-...` restarts at 001, it is not a continuation of `prototype/go/docs/
decisions/`'s own `001`–`008`). No directory anywhere in this repo is uppercase.

## Tier 1 — Core Specification (stable, numbered)

| Document | Covers |
|----------|--------|
| [001-design-principles.md](001-design-principles.md) | Runtime architectural philosophy |
| [002-architecture.md](002-architecture.md) | Conceptual architecture, layer responsibilities |
| [003-runtime-language.md](003-runtime-language.md) | Runtime Language — how applications are described |
| [004-runtime-metadata.md](004-runtime-metadata.md) | Runtime Metadata — scope, hierarchy, versioning |
| [005-runtime-lifecycle.md](005-runtime-lifecycle.md) | How metadata continuously realizes running applications |
| [006-runtime-model.md](006-runtime-model.md) | Runtime object model — Workspace, Application, Machine, and beyond |

## Tier 2 — Supporting Reference (unnumbered, informs Tier 1)

Not part of the numbered reading sequence — each elaborates or grounds one Tier 1 document with concrete detail or external research, rather than adding new normative content.

| Document | Elaborates | Role |
|----------|-----------|------|
| [runtime-metadata-schema.md](runtime-metadata-schema.md) | §004 | Concrete YAML/DB schema for Runtime Metadata (shared by all prototypes, and by `app/`, which graduates it verbatim) |
| [architecture-benchmark.md](architecture-benchmark.md) | §002 | Architecture references from other world-class systems (Chromium, Kubernetes, VS Code, …) — one-time research that informed the architecture, not part of the growing `benchmarks/` evidence series below |

## Practical Guides

| Document | Audience |
|----------|----------|
| [menata-id/menata](https://github.com/menata-id/menata/tree/main/guides) — separate repo (business process language layer, no machine/application concerns) | Domain expert — how to write `.menata` |
| [guides/writing-runtime-metadata.md](guides/writing-runtime-metadata.md) | Developer — how to translate `.menata` into Runtime Metadata |
| [guides/runtime-metadata-gotchas.md](guides/runtime-metadata-gotchas.md) | Developer — loader gotchas + pre-seed checklist (split from the guide above 2026-09-05, grows as new gotchas are caught) |
| [guides/writing-process-overlays.md](guides/writing-process-overlays.md) | Developer — the `process` block shortcut (CAP-W01/W03/W04/W05), split from the guide above 2026-09-05 |

## Tier 3 — Capability Discovery & Governance (evolving)

The runtime's capability is being built and verified through a deliberate discovery process — cases, external benchmarks, and lifecycle governance — rather than ad hoc feature addition. These documents change continuously as each study/phase completes; unlike Tier 1, there is no expectation of stability.

| Document | Role |
|----------|------|
| [roadmap.md](roadmap.md) | The capability-discovery method + phased work plan (start here) — **not** [`app/ROADMAP.md`](app/ROADMAP.md), the separate phased plan for porting code into `app/`; filenames differ only by case |
| [capability-registry.md](capability-registry.md) | Single source of record — every known capability, its status, and priority |
| [case-portfolio.md](case-portfolio.md) | Deliberately chosen test cases and their declared targets |
| [capability-lifecycle.md](capability-lifecycle.md) | How a new capability is proposed, admitted, and completed |
| [nfr-standards.md](nfr-standards.md) | Architecture / performance / security standards per capability area |
| [brd-menata-runtime-v2.md](brd-menata-runtime-v2.md) | Concept BRD for v2 — the Process Overlay ("declared process, emergent execution"), Study 20's Concept C written as a business requirements document, incl. a metadata-only test against all 21 cases (in Bahasa Indonesia, deliberately matching the comparator BRD's genre) |
| [benchmarks/](benchmarks/) | Tier 4 — external evidence studies (see below) |

## Tier 4 — Evidence Studies (`benchmarks/`, numbered by production order)

| Document | Study |
|----------|-------|
| [benchmarks/000-workflow-patterns-mapping.md](benchmarks/000-workflow-patterns-mapping.md) | Study 1 — Menata vs Workflow Patterns Initiative |
| [benchmarks/001-platform-capability-survey.md](benchmarks/001-platform-capability-survey.md) | Study 2 — cross-platform capability survey (Salesforce, Frappe, Drupal, Directus, Budibase, Camunda) |
| [benchmarks/002-portal-ga-cross-domain-survey.md](benchmarks/002-portal-ga-cross-domain-survey.md) | Study 5 — Portal GA v3 cross-domain integration patterns |
| [benchmarks/003-accounting-vertical-survey.md](benchmarks/003-accounting-vertical-survey.md) | Study 6 — accounting vertical (Odoo / ERPNext) |
| [benchmarks/004-scale-architecture-study.md](benchmarks/004-scale-architecture-study.md) | Study 8 — multi-workspace scale & performance architecture |
| [benchmarks/005-field-modeling-decision-framework.md](benchmarks/005-field-modeling-decision-framework.md) | Study 15 — field type selection: reference vs. value_list vs. primitive (§"Final Recap" has the settled answer per type, no re-derivation needed) |
| [benchmarks/006-inventory-warehouse-benchmark.md](benchmarks/006-inventory-warehouse-benchmark.md) | Case 5 supporting benchmark — six-stage WMS flow + APICS/ASCM inventory-control concepts |
| [benchmarks/007-user-role-management-survey.md](benchmarks/007-user-role-management-survey.md) | Study 18 — user & role management across 10 platforms (informed CAP-F05/CAP-O01/CAP-P02) |
| [benchmarks/008-ui-workflow-interaction-benchmark.md](benchmarks/008-ui-workflow-interaction-benchmark.md) | UI task-interaction patterns across all 21 cases — 5 clusters admitted (CAP-V14 Tier 2, CAP-V15–V19), 5 reviewed and rejected |
| [benchmarks/009-in-app-navigation-benchmark.md](benchmarks/009-in-app-navigation-benchmark.md) | In-app navigation patterns — produced CAP-O03 Tier 2 (persistent sub-navigation) |
| [benchmarks/010-gamification-flow-audit.md](benchmarks/010-gamification-flow-audit.md) | Gamification flow audit — three disconnected proofs, no unified action→points→reward chain; integration debt recorded, no new capability |
| [benchmarks/011-metadata-workflow-orchestration-brd-benchmark.md](benchmarks/011-metadata-workflow-orchestration-brd-benchmark.md) | Study 19 — comparator "Metadata-Based Workflow Orchestration Application" BRD mapped against Menata Runtime's emergent (Event+Constraint+Permission+Action) workflow model; full comparator BRD preserved verbatim in its Appendix |
| [benchmarks/012-process-model-synthesis.md](benchmarks/012-process-model-synthesis.md) | Study 20 — deeper re-examination of Study 19: both concepts graded against all 21 portfolio cases (21/21 vs 10/21), server-economy analysis (~3–6 vs ~10–13 statements/transition), and the synthesis — Concept C, the **Process Overlay** ("declared process, emergent execution") |
| [benchmarks/013-overlay-compiler-proof.md](benchmarks/013-overlay-compiler-proof.md) | Study 21 — Process Overlay B1 implemented (`internal/metadata/compile.go`) and conformance-proven (T136–T139): a Machine declaring only a `process` block compiles into ordinary Events/guards/Permissions, behaviorally and architecturally identical to a hand-authored equivalent. Addendum: B2 (T140–T143) proves the same substrate shape renders a legible process map on any Status-guarded Machine, including one that predates the Process Overlay entirely — the decompile claim. Addendum: B3 (T144–T146) implements generic Requirement cardinality (`evidence` type) via write-time fan-in. Addendum: B4 (T147–T150) implements declared SLA (single-machine compile, same low-risk shape as B1/B3) and quorum-of-N core (`min_approvals`, hand-authored only — the declarative form needs a genuinely new cross-machine compile capability, named future work) |
| [benchmarks/014-cmmn-case-management-benchmark.md](benchmarks/014-cmmn-case-management-benchmark.md) | Study 22 — CMMN (Case Management Model and Notation) checked construct-by-construct against the registry and all 21 portfolio cases, closing the un-reexamined CMMN boundary line Case 7 stated since Study 16. Nine of eleven CMMN constructs already compose from existing capabilities (several closed by Study 21's Process Overlay independently of this study); one narrow, evidence-thin gap named and parked HOLD (CAP-W08, Compound Sentry — boolean AND/OR over multiple predecessor facts); one CMMN feature (Case File Item, schema-less case content) recorded as a permanent non-goal — it negates Metadata First/Machine First by design, not a future ⚠️. No new capability admitted |
| [benchmarks/015-metadata-live-reload-proof.md](benchmarks/015-metadata-live-reload-proof.md) | Study 23 — CAP-X04 (metadata live reload) implemented via ADR-002 Option A (admin-triggered, atomic interpreter swap) and conformance-proven (T151–T153), unblocking B5. Deliberately narrower than the originally-sketched "CAP-X04 + CAP-X11" bundle — CAP-X11 (lazy per-workspace loading) is a separate scale concern with no measured pressure yet |
| [benchmarks/016-change-policy-proof.md](benchmarks/016-change-policy-proof.md) | Study 24 — B5, CAP-W07 `change_policy` (effective-dated metadata evolution), implemented and conformance-proven (T154–T158), reusing CAP-X04's live reload to demonstrate `new_records`/`records_in_states`/`all_records` against records that stay open while the metadata changes underneath them |
| [benchmarks/017-quorum-declarative-form-proof.md](benchmarks/017-quorum-declarative-form-proof.md) | Study 25 — CAP-W03's declarative form (`process.requirements[].type: approval`), implemented and conformance-proven (T159–T160): a new cross-machine loader pass injects the same `aggregate_status` action a hand-authored quorum pair already carries, onto a separately-loaded target Machine's own Events |
| [benchmarks/018-case9-completion-batch-proof.md](benchmarks/018-case9-completion-batch-proof.md) | Study 26 — Case 9 completion batch: CAP-C08 (general cross-record constraint), realized through CAP-C10 (`sum(debit)=sum(credit)`) and CAP-C11 (no posting into a closed period), implemented and conformance-proven (T161–T164) |
| [benchmarks/019-decompile-lift-proof.md](benchmarks/019-decompile-lift-proof.md) | Study 27 — B6, CAP-W05 backward direction (decompile-lift), implemented and conformance-proven (T165–T166), built deliberately ahead of case evidence per explicit user direction after the admission-discipline gap was surfaced |
| [benchmarks/020-ui-interaction-cluster-proof.md](benchmarks/020-ui-interaction-cluster-proof.md) | Study 28 — Track D UI/Interaction cluster (CAP-V16/V17/V18/V14 Tier 2/V15/V19), one phase per capability, growing incrementally as each lands |
| [benchmarks/021-design-system-prototype-plan.md](benchmarks/021-design-system-prototype-plan.md) | Study 29 — mobile-first design-system prototype plan for the runtime's own UI shell, all 4 phases complete: Case 9 core chrome direction chosen (Phase 1), the resulting standard applied across auxiliary View types on Cases 3/19/20/12/13 (Phase 2, two deviations named), Form/List decoration confirmed (Phase 3), the standard written up as ADR-008 (Phase 4) |
| [benchmarks/022-bottom-nav-consistency-benchmark.md](benchmarks/022-bottom-nav-consistency-benchmark.md) | Study 30 — corrects ADR-008's mobile bottom-tab-bar design against Salesforce/Google Workspace/WeChat/Odoo precedent: fixed to a global Home/Search/Notifications set rather than re-rendering per-Application content; CAP-O03 Tier 3 implemented and conformance-proven (T186–T188) |
| [benchmarks/023-bottom-bar-necessity-benchmark.md](benchmarks/023-bottom-bar-necessity-benchmark.md) | Study 31 — tests whether the mobile shell needs a bottom bar at all against Odoo/ERPNext/Google Workspace precedent; the case for `shellBottomBar` doesn't survive the test — recommends retiring it and folding Home into the existing top `navBar` instead. Not yet implemented |
| [benchmarks/024-pdf-signature-approval-study.md](benchmarks/024-pdf-signature-approval-study.md) | Study 32 — Case 3 (Document Approval) extended with PDF signature placement: page-by-page screen design added to the Menata Apps Builder canvas, plus a views-configurability assessment. New capabilities admitted, all ❌ Proposed/not built: CAP-F22 (binary PDF signature compositing), CAP-V20 (sequential decision stepper — retroactively registered from a Study 29 design sketch that never got a registry row), CAP-V21 (coordinate-placement editor). CAP-O07 (Groups/Teams) named as real new case pressure, still not built |
| [benchmarks/025-architecture-worldclass-audit.md](benchmarks/025-architecture-worldclass-audit.md) | Study 33 — architecture self-audit against world-class practice (Chromium/K8s/VS Code, ASVS/STRIDE/SRE/ISO-25010 already cited in this repo's own docs) and a structural gap check of `prototype/go`. Headline findings: no CI/CD pipeline exists anywhere in the repo (unacknowledged until this study); `nfr-standards.md`'s Architecture rows described a registry-seam dispatch as built when ADR-004 already confirmed it isn't — corrected in place. Confirms package layering, SQL-injection surface, and security primitives are sound. No code changed, no new capability registered — remediation plan folded into `roadmap.md`'s "Recommended order for upcoming sessions" |
| [benchmarks/026-runtime-graduation-decision.md](benchmarks/026-runtime-graduation-decision.md) | Study 34 — owner decision record: `prototype/`/`benchmarks/` have done their job (proving which capabilities a runtime needs) and stay as-is; a new top-level `app/` folder will hold the real Menata Runtime application going forward, reframing the whole discovery apparatus as *process toward* `app/`, not a parallel end state. Left open: whether `app/` graduates `prototype/go`'s own codebase or starts from zero — evidence gathered, not yet decided |
| [benchmarks/027-workspace-self-service-provisioning-study.md](benchmarks/027-workspace-self-service-provisioning-study.md) | Study 35 — owner-requested BRD: today every Workspace/user is created by hand (seed SQL) or by an existing Admin, with no self-service path to found a workspace, invite others by email, or reach one by URL. Four new capabilities registered, all ❌ Proposed/not built: CAP-O09 (self-service workspace provisioning), CAP-O10 (email invitation), CAP-X14 (workspace-scoped URL routing, `/{slug}/...`), CAP-X15 (custom domain mapping, additive to `/{slug}/`). CAP-X14/CAP-O09 later implemented and conformance-proven the same day (219/219) — see `capability-registry.md`'s own rows |
| [benchmarks/028-multi-workspace-identity-benchmark.md](benchmarks/028-multi-workspace-identity-benchmark.md) | Study 36 — owner follow-up: if the same email legitimately belongs to more than one Workspace, show a "choose a workspace" screen after login instead of blocking it. Checked first: not UI-only — today's `users` row conflates identity with workspace membership, so a real picker needs a `workspace_memberships` table (proposed) before the UI can even work. 5-platform comparator survey (Notion/Atlassian/Basecamp as the model, Slack/Microsoft 365 named as anti-patterns). **CAP-O11** registered ❌ Proposed; CAP-O01 gains a pointer note, unchanged |

## Where does a new document go?

A decision rule, not just a description of what's already here — for a human contributor or an
AI agent deciding where to put something new:

| The document is about... | Goes in... |
|---|---|
| The Runtime Metadata format/spec itself, or a concept every prototype should honor | Root, **Tier 1** (`001`–`006`) if genuinely normative and stable, or **Tier 2** (`runtime-metadata-schema.md`-style) if it elaborates one with concrete detail |
| Discovering or tracking a capability — a case, a benchmark, a registry row, an NFR posture | Root — `case-portfolio.md` / `benchmarks/` / `capability-registry.md` / `nfr-standards.md` (**Tier 3/4**). This machinery is root-level even though most of its evidence today cites `prototype/go` internals — see `capability-registry.md`'s own "Status reflects the Go prototype runtime" caveat. Don't move it under `prototype/go/` just because Go is currently the only prototype deep enough to check status against it. |
| An architecture/implementation decision specific to *one* prototype's own code | That prototype's own `docs/decisions/`, as a numbered ADR (see `prototype/go/docs/decisions/`) |
| Onboarding for working *in* one prototype's own codebase (setup, layer model, house patterns) | That prototype's own top level (`README.md` / `ARCHITECTURE.md` / `DEVELOPMENT.md` / `CLAUDE.md`) — **not** its `docs/` subfolder, which is reserved for deeper reference material (ADRs, worked examples), not onboarding. `CLAUDE.md` specifically must stay at that top level: Claude Code auto-loads `CLAUDE.md` from the current working directory and its ancestors, never its descendants, so nesting it under `docs/` would silently stop it from loading for anyone working from that prototype's own root. |
| A worked example / translation exercise (`.menata` → Runtime Metadata → running app) | That prototype's own `docs/examples/` |

**Established convention: append, don't rewrite.** Several documents here (`roadmap.md`'s dated
status blocks, `capability-registry.md`'s per-row notes, the ADRs under
`prototype/go/docs/decisions/`) record history by *appending* a dated correction/status-update
note when reality moves past what was written, rather than silently rewriting the original text —
see `prototype/go/ARCHITECTURE.md`'s "Corrected 2026-08-22" blocks or `prototype/go/docs/
decisions/`'s ADR-002/ADR-004 "Status update" sections for the pattern to copy. This keeps the *why* of a past decision visible even after it's
superseded.

## How should a document be written?

Patterns already followed consistently across every document in this repo, made explicit here —
found missing (2026-09-06) the same way the filename-case convention above was: consistent in
practice, never written down, which is exactly what makes it easy for a new document (or a new AI
agent session) to drift from it by accident.

- **Decide placement and filename case first.** "Where does a new document go?" above, and the
  filename-case paragraph in this section's own intro — settle both before writing a word.
- **A status header, for anything that changes over time.** Every Tier 3/4 document (this repo's
  own governance/evidence artifacts — not Tier 1's stable spec) opens with a blockquote status
  line: `> Status: vX.Y — <one-line summary of the latest change> | Created: YYYY-MM-DD |
  Updated: YYYY-MM-DD` (see `roadmap.md`, `capability-registry.md`, `nfr-standards.md`'s own
  opening blockquotes for the exact shape). Tier 1/2 documents don't carry one — they're stable by
  design, and a version-status header would misrepresent that.
- **Append, don't rewrite, when something goes stale.** The full rule and its rationale already
  live one section up and in `CLAUDE.md`'s own "Established convention" section — follow it, don't
  restate it in a new document.
- **Cite the concrete evidence behind every claim in a governance/evidence document.** A Case
  number (`case-portfolio.md`), a Study/benchmark number (`benchmarks/NNN-*.md`), a conformance
  test ID (`T###`), a commit, or a specific file/line — never an unsourced assertion. This is what
  makes `capability-registry.md` / `case-portfolio.md` / `nfr-standards.md` audits rather than
  opinion; an unsourced row in any of them is itself a defect worth flagging.
- **Prefer a table over prose for anything status- or comparison-shaped** — one row per
  capability/case/threat/platform, one column per dimension. This is the dominant shape of every
  Tier 3/4 document and every Tier 1 comparison table already in this repo; don't re-narrate the
  same information as prose when the established convention for it is a table.
- **Default language is English.** Bahasa Indonesia is used only with an explicit, stated reason —
  `brd-menata-runtime-v2.md`'s own header states why (matching a comparator BRD's genre so the two
  are easy to place side by side) — never as a default choice, and never for Tier 1: `001`–`006`
  stays normatively English regardless of what elaborates it.
- **Reference other documents by their real path; don't re-explain their content.** The
  `Reference over Duplication` architecture principle (`001-design-principles.md`) applies to this
  repo's own documentation about itself, not only to Runtime Metadata.

## Reference Implementation

**Added 2026-09-06** (Study 34, `benchmarks/026-runtime-graduation-decision.md`): [`app/`](app/)
is the real Menata Runtime application going forward — `prototype/`/`benchmarks/` below have done
their job proving which capabilities a runtime needs, and stay as-is, unrenamed, as the historical
record of that process. `app/`'s own port (`app/ROADMAP.md`, phased) moves fast — **for the
current phase, read [`app/README.md`](app/README.md)'s own "Current status" section**, the one
place kept current every time a phase lands; don't trust a phase number stated anywhere else,
including this file, to still be accurate.

**Status update (2026-09-06): the port is done, `app/` is live.** `app/ROADMAP.md`'s Phase 6
(Cutover) executed — `menata.app` now serves `app/`'s own binary, not `prototype/go`'s. See that
file's own Phase 6 status update for the full detail (database/uploads carried over, RLS already
in place, zero-downtime-minus-the-swap-itself verification against real traffic).

| Location | What it is |
|----------|-----------|
| [app/](app/) | The real Menata Runtime application — **live at `menata.app`** since Phase 6's cutover. Graduated from `prototype/go`'s own codebase; see `app/README.md`'s "Current status" for the full phase-by-phase history |
| [prototype/go/](prototype/go/) | The Go + PostgreSQL + Templ + HTMX prototype that proved this design — no longer the live deployment (`app/` is, as of the Phase 6 cutover above), kept as-is as the proven historical reference and rollback safety net |
| [prototype/{salesforce,frappe,drupal,directus,budibase,camunda}/](prototype/) | Metadata-only proofs on other platforms — see [prototype/README.md](prototype/README.md) for the comparison scorecard |

---

# Contributing

Menata Runtime is currently focused on:

- Runtime Architecture,
- Runtime Language,
- Runtime Metadata,
- Metadata-driven Applications,
- Application Engine,
- Platform Services.

Ideas, discussions, critiques, implementation strategies, architectural proposals, and research references are highly appreciated.

---

# License

Licensed under the Apache License 2.0.

See the LICENSE file for details.
