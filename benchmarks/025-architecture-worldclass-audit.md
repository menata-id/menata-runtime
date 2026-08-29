# Study 33 — Architecture World-Class Benchmark & Structural Gap Audit

> Owner-requested audit (2026-08-29): benchmark this repo's architecture against world-class
> concepts/best practice, and check whether the file/code structure of `prototype/go` still has
> gaps. Unlike Studies 1–32, this is not case-portfolio-driven — it is a direct self-audit of the
> runtime's own implementation and governance documents against the external yardsticks this repo
> already claims to follow (`architecture-benchmark.md`, `nfr-standards.md`), closest in kind to
> Study 22's construct-by-construct CMMN audit or Study 30's precedent-check of ADR-008.

---

# 1. Method

Two independent research passes, cross-checked against each other:

- **Code-level pass** — read `prototype/go/internal/**` directly (not just CLAUDE.md's own
  self-description of it): import graph across all 13 `internal/` packages, `switch`/dispatch
  branch counts, file line counts, SQL query construction, error-handling call sites, security
  primitive wiring (CSRF/bcrypt/RLS).
- **Documentation/governance-level pass** — `architecture-benchmark.md`, `nfr-standards.md` in
  full, all 8 ADRs under `prototype/go/docs/decisions/`, `002-architecture.md`, and
  `benchmarks/004-scale-architecture-study.md`, checked for internal consistency and for drift
  against the code-level pass's findings.

Neither pass trusted the other's or this repo's own prior documentation at face value — every
claim below is either grep/read-verified in this pass or explicitly marked as inherited from an
existing, still-current self-audit (e.g. ADR-004's own status update, which this study confirms
rather than re-derives).

---

# 2. External yardsticks used

Reuses the yardsticks this repo has already adopted rather than importing new ones:

| Dimension | Standard | Already cited in |
|---|---|---|
| Architecture (small-core/extension) | Chromium, Kubernetes, VS Code extension model | `architecture-benchmark.md` |
| Architecture (quality vocabulary) | ISO/IEC 25010, Ford et al. fitness functions | `nfr-standards.md` |
| Architecture (Go project layout) | `golang-standards/project-layout` | `prototype/go/docs/decisions/006-handler-file-split.md` |
| Security | OWASP ASVS 4.x, STRIDE | `nfr-standards.md` §0, §2.x |
| Performance | Google SRE (SLO/error budget) | `nfr-standards.md` §1 |
| Process/CI | Portal GA constitutional stack (fitness functions in CI, ARB decision log) | `capability-lifecycle.md`, `nfr-standards.md` §2.8 |

---

# 3. Findings, most significant first

## F1 — No CI/CD pipeline exists anywhere in the repo (unacknowledged until this study)

`nfr-standards.md` §2.8 states the standard outright: *"fitness functions run in CI (Portal GA
model)."* Verified: no `.github/workflows/` anywhere in the tree (the only `ci.yml` found belongs
to a vendored `node_modules` dependency, unrelated). All 205 conformance tests
(`conformance/run.sh` + `conformance/tests/*.sh`) run **manually** —
`prototype/go/CLAUDE.md`'s own dev loop instructs running them "in another terminal, server must
be running." A repo-wide search of every `.md` file for "CI", "GitHub Actions", "continuous
integration" returns zero hits.

This differs from nearly every other gap in this repo, which by convention gets a dated
acknowledgment note the moment it's found (`⚠️`/`❌` rows, "deferred, reason: ..."). This one has
never been named. Per `capability-lifecycle.md` §3b, an NFR gate with no evidence, measurement, or
waiver should not be treated as satisfied — and §2.8's Architecture line currently reads as if it
already is.

## F2 — `nfr-standards.md`'s Architecture rows (§2.1–2.6) describe a seam that was confirmed, in writing, not to exist

`nfr-standards.md` §2.1, §2.3, §2.5, §2.6 each assert an already-built registry seam: *"each type
= one registered triple... no type-switch sprawl"* (§2.1), *"executor seam per action type"*
(§2.3), *"exactly one guard chokepoint... fitness function: grep for role comparisons outside the
guard"* (§2.5), *"view-type renderer seam"* (§2.6).

This is not a new discovery — `ADR-004`'s own 2026-08-22 status update already states plainly that
every migration trigger it named has fired (~90 capabilities in) and **none** of the target
packages (`engine/`, `security/`, `web/`, `platform/`) exist; dispatch is still `switch`-based.
`ADR-006` (same day, independent audit) reached the opposite operational conclusion and is why the
flat layout stayed. ADR-004 records this fork honestly and leaves itself "kept on record rather
than marked Rejected/Superseded."

What this study adds that wasn't previously written down: **a quantified cost**. Grepped dispatch
surface — 10 `switch`/`case` sites in `internal/executor/executor.go`, 24 across
`internal/handler/*.go`, 2 in `internal/metadata/loader.go`, plus type-name branching in
`internal/ui/components.templ`/`internal/ui/types.go`. Adding one new `FieldType` today requires
touching at minimum `loader.go`, `model.go`, and `ui/types.go`, plus whichever executor/handler
switch branches on it — **3+ files per new field type**, not the one-file `Register()` call the
target architecture in ADR-004/`nfr-standards.md` describes. The inconsistency isn't that the seam
doesn't exist (ADR-004 already says so) — it's that `nfr-standards.md`, the document
`capability-lifecycle.md` §3b treats as a binding gate, still describes it as built. That's the
part this study corrects.

## F3 — Zero Go unit tests: a real but low-priority gap

`find . -name '*_test.go'` returns nothing. All verification is the 205-test conformance suite —
full-stack HTTP round-trips against a real Postgres instance. `prototype/go/CLAUDE.md` states this
is deliberate ("the conformance suite *is* the test suite") and defensible for a
metadata-interpreter architecture where behavior is mostly data-driven. The real cost is narrower
than "no tests": several documented **pure-function heuristics** —
`displayLabel`/`findFieldByName`/`findReferenceFieldTo` and the date-arithmetic helpers (CAP-A11)
— have no fast, isolated feedback loop; every change to one is only verifiable via a full
conformance run. This is an iteration-speed gap, not a correctness gap — lower priority than F1/F2.

## F4 — `internal/handler/record_crud.go` at 999 lines, right at ADR-006/007's own split threshold

ADR-006's own guideline: "a file north of ~1,000 lines covering more than one clearly-separable
concern is a [split] candidate." `record_crud.go` sits one line under that mark. Not yet confirmed
whether it is still single-concern (unlike `loader.go`/`model.go`, which the code-level pass
confirmed are large-but-single-concern) — flagged for a follow-up read, not yet a verdict.

## F5 — Governance process has no automated enforcement and no second reviewer

The capability admission process (A1–A5 test, 9-layer Definition of Done, NFR gates,
append-don't-rewrite audit trail) is unusually rigorous for a solo-maintained repo — comparable in
shape to RFC/ADR discipline at much larger projects. But every decision in it is made by one person
(`CLAUDE.md`: "solo-admin repo; branch-protection bypass on `main` is expected workflow"), and none
of the "fitness function" language in `nfr-standards.md` is currently backed by an actual
executable check (see F1 — there's no CI to run one in even if it existed). This is a reasonable
tradeoff at current scale and is not a defect today; it becomes a real risk only if the project
gains contributors without first closing F1. Named here so it isn't a silent assumption later.

---

# 4. What this audit confirms is *not* a gap

Recorded because a self-audit that only lists problems is as misleading as one that lists none —
these were checked, not assumed:

| Area | Verified finding |
|---|---|
| Package layering | Acyclic and correctly layered across all 13 `internal/` packages — `store` has zero internal deps, `executor` never imports `handler`/`interpreter` (matches the documented rule that Executor has no Interpreter access), `handler` is the sole importer of `ui`. No god-package rot despite ~120 capabilities. |
| SQL injection | Two `fmt.Sprintf`-built queries in `record_store.go` (dynamic ORDER BY direction, aggregate column list) — checked line by line: interpolated pieces are always hardcoded Go-conditional strings, never user input; real values always pass through `$N` bind parameters. No injection risk. |
| Error-swallow (CAP-X12-class bug) | All 53 `slog.Error` call sites in handler/executor checked; only 2 pair with `return nil` (`formfields.go:342,429`), both in read-path option-list helpers where degrading to an empty list is reasonable, not a silent write-loss. No recurrence of the write-path swallow bug class CAP-X12 already fixed once. |
| Security primitives | CSRF (`csrfProtect`), bcrypt password hashing, and RLS (9 `ROW LEVEL SECURITY` + 5 `FORCE ROW LEVEL SECURITY` declarations across `migrations/`) all verified present and correctly wired. |
| File-size discipline | Holding, except F4 (borderline, unconfirmed). The two files that did cross ~1,000 lines (`handler.go`, `run.sh`) were already split per ADR-006/007. |
| CAP-X10/X11 deferral (index management, lazy loading) | A deliberate, reasoned decision — both rows explicitly invoke "Infer Before Configure" (`001-design-principles.md`), reviewed 2026-07-12, deferred pending measured pressure, not neglected. Study 8's 100-workspace/1M-record target remains a declared, not measured, budget — also already acknowledged in `nfr-standards.md` §4's own maintenance note. |
| `002-architecture.md` vs implementation | No drift — the document is deliberately implementation-detail-free by design, so it structurally cannot drift from `prototype/go`. |

---

# 5. Recommendations (remediation plan)

Ordered by cost-to-close vs risk-left-open, not by finding number:

1. ~~Add a CI gate~~ — ✅ **done same day (2026-08-29), closes F1** — but not as GitHub Actions.
   The owner clarified mid-session: this repo runs on a free GitHub plan with Actions minutes
   already exhausted, so a GitHub-hosted workflow wasn't viable. Built instead as a **local git
   pre-push hook**: `scripts/pre-push` (repo root, tracked, diffs the push against
   `prototype/go/` and only runs the gate if it changed) execs
   `prototype/go/scripts/local-ci.sh`, which creates a throwaway isolated Postgres schema +
   throwaway server port, runs `migrate-up && seed && build`, boots the server, runs the full
   conformance suite, and tears everything down on exit regardless of outcome — the same
   "isolated-schema test server" pattern `prototype/go/CLAUDE.md` already documented as a manual
   technique, now automated and wired to `git push`. Verified live, twice, on a genuinely fresh
   schema: 205 passed / 0 failed each run. Install step (`.git/hooks/` isn't tracked, needs a
   one-time copy per clone) documented in `prototype/go/DEVELOPMENT.md` step 9; escape hatches are
   `git push --no-verify` and `SKIP_LOCAL_CI=1 git push`. Same practical guarantee F1 called for
   ("nothing broken reaches `main`"), zero GitHub infrastructure, zero cost. Broader coverage
   (lint, `go vet`, `staticcheck`) remains open for later, opportunistically.

   **Incidental finding while building this**: `Makefile`'s `migrate-up` target was missing
   `migrations/023_groups.sql` (CAP-O07's own migration) — a truly fresh `make migrate-up` would
   have failed before conformance ever ran. Caught only because this gate actually exercises a
   from-scratch schema, which nothing before it did. Fixed in the same pass.
2. **Correct `nfr-standards.md` §2.1–2.6/§2.8's Architecture rows** to state the seam as a target,
   not a built gate, pointing at ADR-004's own status update instead of contradicting it — done as
   part of this study, see the dated note added to that document directly.
3. **Confirm `record_crud.go`'s concern boundary** (F4) — a single read, no code change implied
   unless it turns out to be genuinely multi-concern.
4. **Opportunistic, not urgent**: unit tests for the named pure-function heuristics (F3) — small,
   isolated, would not require restructuring the conformance suite, just supplements it.
5. **No action required now**: F5 (governance enforcement) — re-examine only if/when this project
   gains a second contributor, at which point F1 (CI) becomes the actual precondition for F5
   mattering.

---

# 6. What this study does not do

No new capability registered — a CI gate and an NFR-documentation correction are cross-cutting
process/architecture concerns, not a Grammar-area capability a domain expert would phrase in
`.menata` (fails admission test A3/A5 in `capability-lifecycle.md` §2), so nothing here belongs in
`capability-registry.md`. `nfr-standards.md` gains a dated correction note (§2 top-of-section and
§2.8), `ADR-004` gains a second status-update section confirming and quantifying its own
2026-08-22 finding, and `roadmap.md` tracks the remediation plan above as Study 33 / a new
"Recommended order" entry.

**Correction (2026-08-29, later the same session):** unlike the rest of this study (audit + plan
only, no code), §5 item 1 (the CI gate) *was* built and verified the same session, at the owner's
follow-up request — see the struck-through item above. Everything else in this study's plan
(items 2–5) remains audit + plan only, per this repo's standing discipline of separating "found
and recorded" from "built and proven."
