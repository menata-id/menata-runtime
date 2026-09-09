# Menata Runtime — ObjectStack Comparator Study

> Status: v1.3 — a follow-up owner Q&A (2026-09-09) re-read the same "Composable Metadata Runtime"
> proposal against live code a second time, appending §8 to
> `docs/composable-view-proposal-reconciliation.md`: two more scoping holes in `CAP-V10` Tier 2's
> §7 wording (recursive nesting depth undesigned; composed-page layout not folded in) and a
> necessity call on the open context-passing question (low today, mandatory co-requisite once
> Tier 2 is ever admitted). No row admitted, no code changed | Previously v1.2 — a third external
> document (an unsolicited "Composable Metadata Runtime"
> architectural-direction proposal) reconciled against this study's own already-settled position on
> a Page/Component/Slot ontology, in `docs/composable-view-proposal-reconciliation.md`: confirms the
> earlier rejection of dynamic component dispatch/Canvas as SPA-shaped machinery, refines `CAP-V10`
> Tier 2's scope with a small closed static-content vocabulary and its data-resolution mechanism, and
> names an open context-passing question. No row admitted, no code changed | Previously v1.1 — an
> independent second review of the same two repos (brought by the owner the same
> day) reconciled against this study in `docs/second-opinion-reconciliation.md`: six narrow additions
> (R24–R29), one re-prioritisation (R21 MCP → propose the Case 3 assistant extension now), five of
> its "critical" items declined with reasons, updated priority order in that doc's §5 | Previously
> v1.0 — Study 37, first pass: ObjectStack studied from its own source tree and docs, compared
> area-by-area against Menata Runtime, gaps and candidate capabilities named (none admitted to the
> registry yet — owner decision pending) | Created: 2026-09-07 | Updated: 2026-09-09

> **Not a metadata-proof prototype.** Every other folder under `prototype/` (except `go/`) answers
> "can *this platform* realize `design-request.yaml` from metadata alone?" and carries a 16-feature
> score. This folder answers a different question, asked directly by the owner (2026-09-07): *"this
> looks conceptually similar to Menata Runtime — study it, compare, find the gaps, and say what
> Menata Runtime should adopt to be flexible and powerful."* It is a **comparator study** (the
> same genre as `benchmarks/001-platform-capability-survey.md` or `benchmarks/011-*`), placed here
> under `prototype/` per the owner's explicit instruction rather than under `benchmarks/` where
> `README.md`'s "Where does a new document go?" rule would otherwise put it. Root `README.md`,
> `prototype/README.md`, `roadmap.md` and `capability-registry.md` all point here so the placement
> can't be missed.

---

## What ObjectStack is

[ObjectStack](https://github.com/objectstack-ai/objectstack) (`objectstack-ai/objectstack`,
Apache-2.0, TypeScript) is an "AI-native business backend protocol": the whole application — data
model, views, flows, permissions, actions, AI tools — is typed metadata (Zod schemas), compiled into
one JSON artifact, and interpreted by a microkernel runtime that derives the database schema, a REST
API, a React console UI, and an MCP server from it. Tagline: *"Apps small enough for AI to hold
whole."* Its commercial sibling is ObjectOS (hosted "Build & Ask").

The conceptual overlap with Menata Runtime is real and deep — both are metadata-interpreting
runtimes, both derive UI/API/permissions from one definition, both refuse to let the UI own
behavior. The differences are equally deep — in authoring model (TypeScript vs. YAML/DB rows),
storage model (table-per-object vs. one JSONB `records` table), UI model (React SPA vs.
server-rendered HTMX), extension model (plugins + code escape hatches vs. metadata-only), and
maturity discipline (spec-first with a large declared-but-unenforced surface vs. conformance-gated
ratchet). Both the overlap and the differences are the point of this study.

## Sources studied

| Source | How it was read | Snapshot |
|---|---|---|
| `github.com/objectstack-ai/objectstack` | Cloned (`--depth 1`) and read directly — `README.md`, `ARCHITECTURE.md`, `AGENTS.md`, `ROADMAP.md`, `packages/spec/src/**` (Zod schemas, the normative source), `packages/{core,objectql,rest,drivers/driver-sql,services/*,plugins/*}` source trees, `examples/app-crm`, `docs/adr/*` (131 ADRs), `docs/{HARDENING,OBSERVABILITY,PLATFORM_GAPS_FROM_TEMPLATES}.md` | commit `ac76425f`, 2026-09-07 00:06 UTC; `@objectstack/spec` 17.3.0; monorepo 4.0.1 |
| `objectstack.ai/docs` | The site is built from `content/docs/**` in the same repo — read from the clone (438 `.mdx` pages: `capabilities/`, `concepts/`, `data-modeling/`, `ui/`, `automation/`, `permissions/`, `ai/`, `api/`, `kernel/`, `deployment/`, `protocol/`, generated `references/`) | same commit |
| `objectstack.ai` (landing) | Fetched; positioning/claims only | 2026-09-07 |
| Menata Runtime side | `app/` (live codebase), `capability-registry.md` v0.54, `runtime-metadata-schema.md`, `case-portfolio.md`, `roadmap.md` Studies 19–36, `capability-lifecycle.md`, `nfr-standards.md` | repo `main` @ `236165a`, 2026-09-07 |

Every ObjectStack claim below is cited to a file in that clone (`packages/...`, `docs/adr/...`,
`content/docs/...`), never to the marketing page alone — the landing page and `capabilities/*.mdx`
overstate what is enforced (see "Highlights §H7" in the comparison doc), so the Zod schemas and
the ADRs are treated as the source of truth, the same way `capability-registry.md` treats
conformance tests rather than prose as proof here.

## Documents in this study

| Document | Covers |
|---|---|
| [docs/architecture-and-backend.md](docs/architecture-and-backend.md) | Backend architecture, monorepo/folder structure, request path, storage model, metadata pipeline, expression layer, extension/escape-hatch model, deployment/cluster/observability — each set side-by-side with `app/`'s own shape |
| [docs/capability-comparison.md](docs/capability-comparison.md) | The six areas the owner named — **field types, views, workflow/automation, approvals, permissions, analytics** — plus other highlights (MCP/AI tools, metadata versioning, notifications, search, validation gates, i18n, packaging), each as a table against the registry |
| [docs/gap-analysis-and-recommendations.md](docs/gap-analysis-and-recommendations.md) | The gap list, what Menata Runtime should adopt (with the best-practice basis for each), what it should deliberately *not* copy, and where Menata Runtime is ahead — framed by the owner's two words: **flexible** (can build anything) and **powerful** (stays fast on efficient server resources) |
| [docs/second-opinion-reconciliation.md](docs/second-opinion-reconciliation.md) | An independent second review of the same repos (owner-supplied, 2026-09-07) checked claim-by-claim against registry rows and `app/` source: agreements, disagreements with evidence, six additions (R24–R29: relation policy, searchable fields, API completion, metadata snapshots/rollback, history timeline, event presentation), what was declined and why, and the **updated priority order that supersedes the gap doc's §7** |
| [docs/composable-view-proposal-reconciliation.md](docs/composable-view-proposal-reconciliation.md) | A third external document (owner-supplied, 2026-09-07) — a "Composable Metadata Runtime" architectural-direction proposal — reconciled against this study's own already-settled position on a Page/Component/Slot ontology (`second-opinion-reconciliation.md` §2): confirms the SPA-shaped rejection of dynamic component dispatch/Canvas, refines `CAP-V10` Tier 2's scope with a small closed static-content vocabulary + its data-resolution mechanism, names an open context-passing question |

## Executive summary

**Scale of the two codebases** (measured, not quoted):

| | ObjectStack (`packages/`) | Menata Runtime (`app/`) |
|---|---|---|
| Language / runtime | TypeScript on Node 22 (also Bun/Deno/Workers via Hono) | Go, single static binary |
| Non-test source | ≈790k lines of `.ts` (`wc -l`, incl. generated translation files), 45+ packages | ≈21k lines of `.go`, 15 `internal/` packages |
| Spec surface | 864 non-test `.ts` files under `packages/spec/src`, "1,600+ Zod schemas" (README), 50 field types, 9 list-view types, 21 flow node types | 14 field-type keywords, 12 View types, 9 action types, ~15 constraint operators — all in `internal/model/model.go` |
| Proof | 3,347 test files, README badge "6,507 tests passing"; no capability-level registry; `content/docs/releases/implementation-status.mdx` is a hand-maintained matrix that lags | 225 conformance tests (`./scripts/local-ci.sh`), one ratcheting registry row per capability |
| Decision record | 131 ADRs (`docs/adr/`), many recording "declared but unenforced" surfaces later removed under ADR-0049 *enforce-or-remove* | 8 + 1 ADRs, 36 numbered studies, per-row registry notes |
| Persistence | Table-per-object via knex (Postgres/MySQL/SQLite/MSSQL), MongoDB, in-memory; additive-only DDL sync + drift detector | One `records(data JSONB)` table per workspace schema, GIN index, Postgres only |
| UI | Separate React SPA (`objectstack-ai/objectui`), served as a prebuilt bundle | Server-rendered Templ + HTMX + Hyperscript, zero client framework |

**Where ObjectStack is clearly ahead (and worth learning from):** a general expression layer (CEL)
used uniformly for formulas, predicates, RLS, flow conditions and visibility; a semantic
`dataset` layer under all analytics so a measure is defined once; scope-depth permission grants
(`own / own_and_reports / unit / unit_and_below / org`) over a business-unit tree; a richer
field-type palette (multiselect, percent, email/url/phone formats, lookup-vs-master-detail,
secret, composite/repeater); nine list-view lenses (gantt, gallery, map, tree, chart); approval
send-back-for-revision, per-group sign-off and manager-chain approvers; execution-pinned metadata
versions (their ADR-0009 is the design Menata's own CAP-W02 ❌ lacks); and a governed MCP surface
that makes every object/action an AI tool under the same permissions.

**Where Menata Runtime is clearly ahead (and should not trade away):** metadata-only with no code
escape hatch (ObjectStack needs TypeScript hooks, sandboxed script bodies, `functions`, and jobs);
a conformance-gated registry where ✅ means proven (ObjectStack's own ADR-0020 found its state
machine had *three declaration shapes and zero enforcement*; ADR-0061 found `$search` was a
silent no-op on every surface); the Process Overlay ("declared process, emergent execution" —
Study 20's server-economy argument holds up against ObjectStack's 21-node DAG engine); a business
calendar behind SLAs (ObjectStack's approval escalation is explicitly wall-clock only,
`approval.zod.ts:633`); effective-dated change policy (CAP-W07); and a resource footprint an
order of magnitude smaller, which is exactly the owner's "powerful" criterion.

**Recommendation in one line:** adopt ObjectStack's *ideas* where they close a registry gap with
dual evidence — expression layer, semantic dataset, scope-depth permissions, MCP surface,
approval enrichments, a handful of field/view types — and adopt none of its *architecture*
(microkernel DI, TypeScript authoring, SPA console, code hooks), because the architecture is what
costs it 37× the code and a cluster to stay correct. Full prioritized list, with the admission-test
pre-screen for each, in
[docs/gap-analysis-and-recommendations.md](docs/gap-analysis-and-recommendations.md).

## Registry impact (this pass)

**No capability admitted, no row changed.** The recommendations doc makes 23 recommendations
(R1–R23): 14 would be new registry rows (suggested IDs `CAP-C13`, `CAP-V22`, `CAP-V23`,
`CAP-P08`, `CAP-O12`, `CAP-P09`, `CAP-W09`, `CAP-A16`, `CAP-C14`, `CAP-V24`–`V26`, `CAP-V27`,
`CAP-X16`), the rest are Tier-2 notes or evidence added to rows that already exist (`CAP-X10`,
`CAP-I04`, `CAP-W02`, `CAP-O10`, `CAP-V11`, `CAP-F03`, `CAP-F13`, `CAP-W03`, `CAP-W04`,
`CAP-V12`, `CAP-R06`, …). Each is pre-screened against `capability-lifecycle.md` §2's five
admission criteria, and the ones that pass A1 (dual evidence) only because of this study plus a
*declared* case target are marked as such.

**Status update (2026-09-07, same day): registered, per direct owner decision.**
`capability-registry.md` v0.56 now carries 14 new ❌ Proposed rows (`CAP-C13`, `C14`, `V22`–`V27`,
`P08`, `P09`, `O12`, `W09`, `A16`, `X16`, `X17`) and Study 37 notes on 15 existing rows; the Case 3
"approval assistant" extension is declared in `case-portfolio.md` (terrain half of A1 for
`CAP-X16`); `roadmap.md`'s "Recommended order" item 24 carries the build ordering from
`docs/second-opinion-reconciliation.md` §5. No Prio assigned in the registry yet, no code changed. Admitting any of them is an owner decision, recorded
in `capability-registry.md` when it happens — the registry's own status header (v0.55) points
here so the pending decision is visible, per its "declare targets first" discipline.

## Relationship to the other prototypes

| Folder | Question it answers | Scored? |
|---|---|---|
| `prototype/go/` | Can a custom runtime interpret Runtime Metadata? (graduated into `app/`, Study 34) | conformance suite |
| `prototype/{salesforce,frappe,drupal,directus,budibase,camunda}/` | Can an existing platform realize `design-request.yaml` from metadata alone? | 16-feature score |
| **`prototype/objectstack/`** (this) | What does a peer metadata-runtime do that Menata Runtime doesn't, and what should Menata Runtime adopt? | no score — gap list + candidates |

No `docs/examples/` translation (`design-request.yaml` → ObjectStack metadata) was produced.
ObjectStack authors metadata as TypeScript compiled by its own CLI, so a faithful translation needs
its toolchain installed and running (`pnpm install`, `os validate`) to be a *proof* rather than a
sketch; the owner asked for a study, not a scored proof. It can be added later as a separate pass
if a scored row in `prototype/README.md` is wanted — the mapping table in
`docs/capability-comparison.md` §"Vocabulary map" is the starting point.
