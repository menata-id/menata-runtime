# Study 34 — From Prototype/Benchmark Process to a Main Application (2026-09-05)

Owner-initiated decision record, not a benchmark against an external yardstick like most of this
series — a direct question raised while documenting this session's own Group/Approval Document
capability work: two placement/naming inconsistencies surfaced, and answering them honestly
opened a much larger question about what `prototype/go` actually *is* at this point in the
project's life. Recorded here because the decision reframes how every future document in this
repo should be read, not because of a single finding worth a one-line registry note.

---

## 1. Two inconsistencies found while documenting routine work

**F1 — `guides/writing-runtime-metadata.md` (and its two 2026-09-05 siblings,
`runtime-metadata-gotchas.md` and `writing-process-overlays.md`) live at repo root, implying
content every prototype should honor — but their actual content is Go/Postgres-schema-specific.**
The guide doesn't stop at Runtime Metadata as a concept (that's already `004-runtime-metadata.md`,
genuinely prototype-agnostic, Tier 1); it shows the literal `INSERT INTO fields (id, machine_id,
name, type, position, required, options) VALUES ...` shape `prototype/go`'s own Postgres schema
uses. None of the other six prototypes (`prototype/{drupal,frappe,directus,budibase,salesforce,
camunda}`, each a metadata-only proof on its own platform's native config format) would ever
consume metadata this way. Root `README.md`'s own "Where does a new document go?" table already
has the rule this guide should have followed from the start: *"A worked example / translation
exercise (`.menata` → Runtime Metadata → running app) → that prototype's own `docs/examples/`"* —
this guide is exactly that translation exercise, written as narrative prose instead of example
files, and was never moved to match.

**F2 — root `README.md`'s own "Reference Implementation" section header contradicts its own body
text, the folder name, and 24+ other `.md` files.** The section title already reads "Reference
Implementation" (not "Prototypes"), but the row directly under it still describes `prototype/go/`
as "the working Go + PostgreSQL + Templ + HTMX **prototype**," and every other document
(`prototype/go/CLAUDE.md`, `ARCHITECTURE.md`, `DEVELOPMENT.md` — 6, 7, and 11 occurrences of the
word respectively) does the same. This is the visible symptom of a real thing that happened: seven
prototypes were seeded to compare platforms; one of them (`prototype/go`) pulled far enough ahead
(~90 capabilities ✅, 219-test conformance suite, full CI/CD as of this same session) that it
stopped being a peer of the other six and started being something else — the terminology never
caught up to that shift.

---

## 2. The decision the owner actually made

Presented as two narrower questions (where should the guide live; what should "prototype" be
called) — the owner's answer reframed both at once, larger than either question alone:

> "kalau untuk tahap prototype dan poc udah cukup. kemudian pengembangan selanjutnya di root.
> untuk menata runtime. sehingga dokumentasi writing guide benar. prototype kode juga benar.
> tetapi pengembangan selanjutnya, buat folder baru di root. untuk benchmark dan prototype adalah
> proses menuju membangun aplikasi utama menata runtime ini."
>
> "develop aplikasi utama menata runtime dipindah ke level struktur folder lebih tinggi."

Read together: **`prototype/` (all seven) and `benchmarks/` are considered to have done their
job** — proving which capabilities a runtime needs and that a real implementation of them is
possible. Neither is wrong as it stands; neither needs renaming or moving. **Going forward, the
real Menata Runtime application is built in a new top-level folder, `app/`, sibling to
`prototype/`/`benchmarks/`/the root Tier 1-4 docs — not nested inside `prototype/`.** The whole
discovery apparatus (case portfolio, benchmarks, capability registry, seven prototypes) is
reframed explicitly as *process toward* `app/`, not an end state being maintained forever in
parallel with it.

**Consequence for F1**: `writing-runtime-metadata.md` and its siblings were about to be judged
misplaced against the *old* frame (seven equal prototypes, this guide oddly Go-specific among
them). Against the *new* frame, they're arguably placed correctly after all — `app/` will need
exactly this guide, and it stays germane at root once `app/` exists beside it. **No file move
needed on account of F1** — left as-is, correctly, under the corrected understanding of what root
`guides/` is actually for now.

**Consequence for F2**: no rename executed. `prototype/go/` keeps its path, its name, its own
docs' current wording — it is not being renamed to "reference implementation" or anything else, it
is being *superseded* by `app/` for all future development. Retiring "prototype" as the term for
it can happen naturally once `app/` exists and carries the "real" designation instead; forcing the
rename now, before `app/` exists to contrast against, would be solving a naming problem that
`app/`'s own existence is about to solve on its own.

---

## 3. Open sub-decision: does `app/` start from `prototype/go`'s code, or from zero?

Asked directly; the owner's answer was **"belum tahu, perlu didiskusikan dulu"** — genuinely
undecided, not a soft yes to either option. Evidence gathered so far to inform that decision, not
a verdict:

**Case for graduating `prototype/go`'s codebase as `app/`'s starting point:**
- ~90 capabilities ✅ in `capability-registry.md`, each with a conformance test proving it — real,
  expensive-to-reproduce engineering, not throwaway spike code.
- The one architectural concern ever seriously raised against it — field/action/view-type dispatch
  still `switch`-based, not the `Register()` seam `docs/decisions/004-internal-package-
  architecture.md` originally sketched — has been independently re-examined **twice** (that ADR's
  own 2026-08-22 status update, and `benchmarks/025-architecture-worldclass-audit.md`'s
  independent 2026-08-29 re-derivation) and **both times concluded it isn't actually a problem at
  this scale**: "switch-statement dispatch has scaled past every checkpoint this ADR named without
  the predicted pain... no capability has yet demonstrated the registry indirection is actually
  needed." Not a hidden landmine waiting to be inherited.
- What genuinely IS "prototype-honest" today — `displayLabel`'s Name-or-first-text-field guess,
  CAP-A07's Sequence/Decision/Approver name-matching, and similar heuristics documented inline
  where they live — are explicitly stand-ins for a Menata Language grammar gap (no way yet for a
  business author to declare "this is the field people should see" or "this is the sequence
  field"), not a Go-implementation shortcut. A from-scratch Go rewrite would need the identical
  heuristic again, unchanged, until `menata-id/menata` itself grows that grammar — starting over in
  Go doesn't remove this cost, only evolving the Language spec does.

**Case for starting fresh** (not independently investigated this session — named for completeness,
not argued): a clean slate free of any accumulated "prototype" framing in code comments/tests;
room to make deliberate architecture calls with full hindsight already in hand from having built
90 capabilities once. No concrete evidence was gathered this session that the inherited code
actually carries a cost matching this benefit — this is the side of the argument that still needs
its own investigation if the owner wants to weigh it seriously.

**Not yet decided. Next session picking this up should start from this section, not re-litigate
F1/F2 above** — those are closed. The open question is narrower: `app/` graduates `prototype/go`,
or `app/` starts from zero.

---

## 5. Concrete production-readiness gap inventory (2026-09-05)

Requested directly: not "is the architecture sound" again (§3 already answered that — yes, twice
independently) but "what is *concretely missing* if `prototype/go` became the real application,"
so the graduate-vs-fresh-start question could be weighed against real evidence instead of the
generic shape of the argument alone. Gathered by direct repo investigation, organized by area:

| Area | Actual state | Evidence |
|---|---|---|
| **Testing** | 3 test files total, all added 2026-09-05 (`internal/model`, `internal/handler`, `internal/executor`). Coverage: `model` 88.9%, `executor` 15.3%, `handler` 0.6%. **Zero** test files in `store`, `router`, `interpreter`, `metadata`, `permission`, `auth`, `config`, `db`, `ui`, `cmd/server`. No mocked-DB pattern anywhere — everything else leans entirely on the 219-test conformance suite (real Postgres + real HTTP). No coverage gate in CI. | Direct `go test -cover ./...` run + file listing |
| **Migrations** | Plain numbered `.sql` files applied via a hardcoded ordered `psql -f` list in the Makefile. No version-tracking table, no DOWN/rollback scripts, no tool (golang-migrate/goose/Atlas). A partially-applied migration has no automated recovery path. | `Makefile`'s `migrate-up` target |
| **Multi-instance** | `Interpreter` is a single-process in-memory cache (`atomic.Pointer[Interpreter]`), swapped via `POST /admin/reload`. Running 2+ replicas would NOT propagate a reload to every instance — each process's cache goes stale independently. Named, understood, unbuilt (CAP-X11, LISTEN/NOTIFY). | `docs/decisions/002-metadata-loading.md` |
| **File storage** | Uploads on local disk (`uploads/`), not object storage. A multi-instance/multi-host deployment would need shared disk or a real storage migration. | CAP-F06 registry row |
| **Secrets** | `.env` plaintext (`DATABASE_URL=postgres://...:CHANGE_ME@...`), no vault/secrets-manager integration. | `.env.example` |
| **Dependency scanning** | No `govulncheck`/Dependabot anywhere, including the three CI workflows added this same session. | repo-wide grep |
| **Rate limiting / DoS** | No rate-limiter/WAF-class protection in the handler/middleware stack — `nfr-standards.md`'s own STRIDE "Denial of service" row lists only pagination/action-budgets/pool-separation. | `nfr-standards.md` |
| **Encryption at rest** | None for any field — zero `pgcrypto`/field-level-encryption mentions anywhere. | repo-wide grep |
| **API versioning** | `GET/POST /api/{machine}` has no version prefix, no generated OpenAPI/Swagger spec. | `router.go` |
| **Deployment** | No Dockerfile/compose/K8s/Terraform/Ansible anywhere. `nohup ./bin/server` managed by a hand-rolled bash script (`server-manager.sh`) — no process supervisor, no crash-triggered restart, no rolling/blue-green deploy. | repo-wide search, `server-manager.sh` |
| **Team-scale coupling** | `internal/handler` is the fan-in hub (imports 10 other `internal/*` packages), no stated public-API boundary between "core runtime" and "web layer." Evaluated only at solo-developer scale so far — the multi-engineer case is untested, not confirmed broken. | ADR-006/007 |

None of this contradicts §3's own finding (the *architecture* — package layout, dispatch
mechanism — has been vetted twice and holds up). This is a different axis: operational/production
maturity, which nothing has audited before because nothing has needed it to be audited before.

---

## 6. Addendum (2026-09-06): two corrections that change the calculus

The owner supplied two facts §5 was gathered without, and both cut the same direction — toward
graduating, more strongly than §3 alone argued.

**Correction 1 — this host runs no containers, by resource constraint, not preference.** The VPS
`menata.app` (and every future `app/` deployment, presumably the same host) already runs several
other apps' own production instances; RAM/CPU headroom is real and named, not a stylistic choice
(see `prototype/go/DEVELOPMENT.md`'s own "No Docker/containers" note, added the same day as this
addendum). This directly voids two of §5's implicit "fresh start" appeals that a generic
world-class checklist would otherwise list: **container-native deployment (Docker/Kubernetes
health probes, rolling/blue-green deploys)** and **multi-instance-first design (horizontal
scaling behind a load balancer)** are not just "not built yet" — they are **not going to be
built either way**, on this host, regardless of which codebase `app/` starts from. Weighing
"start fresh so you can design for containers/multi-instance from day one" makes no sense against
a deployment target that was never going to have either.

**Correction 2 — the architecture is already thin, which changes what "inheriting prototype/go"
actually costs.** §3 already established this (ADR-004/Study 33, twice), but the direct
implication for §5's remaining gap list wasn't drawn out explicitly until asked: **every gap in
§5 that survives Correction 1 — migration tooling with rollback, an object-storage abstraction
behind the upload interface, API versioning, a secrets-manager integration, dependency-
vulnerability scanning, rate limiting, deeper test coverage — is an INCREMENTAL ADDITION to the
existing codebase, not a REWRITE.** None of them require touching the parts of the architecture
already vetted as sound (the flat package layout, the switch-based dispatch); each is a new
library call, a new CI step, or a new middleware layered onto code that already works. A
from-scratch build gains no head start on any of them — adding `golang-migrate` to `app/` costs
the same whether `app/`'s Go code is graduated or brand new; the *codebase being lean already*
(not tangled, not over-coupled beyond what two audits already cleared) is precisely what makes
retrofitting these additions cheap rather than the usual "legacy system" story where hardening is
expensive because the code fights back.

**What's left as a genuine (not merely generic) argument for starting fresh, after both
corrections**: only the team-scale coupling question in §5's last row — `internal/handler` as an
unbounded fan-in hub with no stated core/web API boundary, evaluated so far only at solo-developer
scale. This remains real but entirely speculative: no second engineer exists yet, no case has
shown the current shape actually causing friction. It is not evidence on the same footing as
§3/§5's own findings, which are all things directly observed in this repo — it is a prediction
about a team that doesn't exist yet.

**Recommendation, updated:** graduate `prototype/go`'s codebase as `app/`'s starting point,
treating §5's surviving gap list (migrations tooling, storage abstraction, API versioning,
secrets, dependency scanning, rate limiting, test depth) as a **post-graduation hardening
backlog** — real work, sequenced deliberately, not a reason to defer starting `app/` until it's
all done, and not a reason to discard 90 proven capabilities and 219 tests to get a codebase that
would need the identical hardening backlog anyway. This is a strengthened recommendation, not yet
a final decision — the owner's own confirmation is still the thing that closes §3/§4's "open" row.

---

## 7. Empirical verification (2026-09-06): confirming "thin architecture" with measurements, not citations

Asked directly: not "does the ADR say it's fine" again, but *how do you actually confirm* it —
so three independent, reproducible checks were run against the real code, not just §3/§6's own
prose re-cited. Method, so a future session can re-run the same checks rather than trust this
one's numbers forever:

**Method 1 — file-size + concern-count, the exact ADR-006/007 admission test** (`find internal
cmd -name "*.go" ! -name "*_templ.go" | xargs wc -l | sort -rn`): 12,753 non-generated Go lines
total. Two files at/past the ~1,000-line trigger: `internal/handler/record_crud.go` (1,009 —
already checked once, roadmap item 13, "999 lines, still one concern," now nominally over by 10)
and **`internal/metadata/loader.go` (1,098 — new finding, never checked before)**. `loader.go`
genuinely bundles three separable concerns by inspection (`grep -n "^func "`): ~12 `load*`
functions (one per DB table, the bulk of the file), 2 validation functions
(`validateOperators`/`validateReferences`), and 2 process-overlay/quorum-compile functions
(`compileApprovalRequirements`/`injectApprovalQuorum`) — a real split candidate by the same test
that justified `handler.go`'s own ADR-006 split, not yet acted on. This is the one place the
"thin" claim needs a caveat, not a place it fails outright.

**Method 2 — import fan-in/fan-out** (`go list -f '{{join .Imports "\n"}}' ./internal/X | grep
menata.id/runtime/internal | wc -l` per package, plus `go build ./...` as proof of zero cycles —
Go's compiler refuses to build with an import cycle, so "no cycles" here is a compiler guarantee,
not a claim): `model`/`store`/`auth`/`db`/`config` are true leaves (0 internal imports — pure
data/infra layer); `interpreter`/`metadata`/`permission`/`constraint`/`router` each depend on
exactly 1 other internal package; `ui` depends on 2; `executor` on 4; `handler` (the legitimate
top orchestration layer, wiring HTTP to everything below it) on 9. A clean layered shape, not a
tangle — the red flag this check would have caught is every package importing every other one,
which isn't what's here.

**Method 3 — empirical cost of a real past capability, in files touched** (`git show --stat
<commit>` on three recent capability commits, filtered to non-generated `.go` files):

| Capability | Files touched | Shape of the diff |
|---|---|---|
| CAP-V20 (new View type) | 6 | One new 133-line file (`decisionstepper.go`) + five small edits (1–26 lines each) |
| CAP-F23 (new `FieldOptions` key) | 3 | Small edits only, 16–35 lines each |
| CAP-V21 (new View type) | 6 | One new 226-line file (`coordplace.go`) + five small edits (2–30 lines each) |

Confirms Study 33's own "a new FieldType touches 3+ files" claim with real numbers, and shows the
shape isn't shotgun surgery: most of each diff concentrates in one new file, with small, mechanical
registration edits (router, model, interpreter) elsewhere — the signature of a codebase that's
cheap to extend, not one fighting back.

**Conclusion**: the "architecture is thin" claim underlying §6's recommendation is now
measurement-backed, not just ADR-cited — with one honest, actionable exception (`loader.go`)
recorded as a real backlog item, not swept under the "everything's fine" rug. See roadmap.md's
matching entry for the split itself, not yet done.

---

## 4. What's settled vs. still open (summary)

| Question | Status |
|---|---|
| Do `prototype/`/`benchmarks/` need renaming or restructuring now? | **No** — settled, they're done, left as-is |
| Does `guides/writing-runtime-metadata.md` (+ siblings) need to move? | **No** — settled, correctly placed under the corrected frame |
| Is a new top-level folder being created for the real application? | **Yes** — settled, named `app/` |
| Does `app/` graduate `prototype/go`'s code, or start from zero? | **Recommended: graduate** (§6, 2026-09-06) — stronger than §3's own case alone, after the no-containers/thin-architecture corrections; still awaiting the owner's explicit final confirmation before treated as settled |
| What hardens after graduation, and in what order? | **Open, but scoped** — §5's gap list (migrations tooling, storage abstraction, API versioning, secrets, dependency scanning, rate limiting, test depth), sequenced by whoever picks up `app/`'s first real milestone |
| Is the "thin architecture" claim itself verified, not just cited? | **Yes** — §7, three independent measurements (file-size/concern-count, import graph, empirical capability-cost), 2026-09-06 |
| Does `internal/metadata/loader.go` need splitting? | **Not yet done — tracked as a backlog item** (roadmap.md), same ADR-006 pattern as `handler.go`'s own split. Found via §7 Method 1, not acted on this session |
| When does `app/` get created, and by whom (this session or a dedicated future one)? | **Done, this session (2026-09-06)** — blueprint stage: package skeleton (`internal/*/doc.go`, 14 packages, `go build`/`go vet` clean), `go.mod`, onboarding docs, and `docs/decisions/001-graduation-from-prototype.md`. See `app/ARCHITECTURE.md` for the full blueprint. No business logic ported yet — that's the next, separate development-plan phase |

---

## 8. Blueprint created (2026-09-06)

`app/` now exists, built under §6's recommendation (graduate) since that's the strongest-evidenced
direction so far — but this does NOT itself close the "recommended, not yet settled" status on
that recommendation; the owner's explicit final confirmation still does. What exists: a real,
`go build`-clean package skeleton matching this document's own §7 measurements (leaf/mid/
orchestration layers), one new package (`internal/storage`, addressing §5's file-storage gap),
`internal/metadata` already planned as split (§7's own `loader.go` finding, roadmap.md item 20)
rather than carried over unsplit, and every other §5 gap named explicitly in `app/ARCHITECTURE.md`
as deferred to a development plan, not silently dropped. Full detail: `app/ARCHITECTURE.md` and
`app/docs/decisions/001-graduation-from-prototype.md`.

## 9. Development plan and roadmap created (2026-09-06)

`app/ROADMAP.md`: a 7-phase port sequence (Phase 0 foundation/leaf packages → Phase 1 metadata →
Phase 2 business logic → Phase 3 HTTP layer → Phase 4 conformance parity → Phase 5 CI/hardening →
Phase 6 cutover), each phase ending in something real and verifiable rather than one big-bang
port. Every §5 gap this document named gets addressed at the specific phase where doing so is
cheapest, not front-loaded or ignored: migrations tooling (Phase 1, before real data exists),
API versioning + rate limiting (Phase 3, before real API consumers exist), the client-side
Hyperscript-vs-vanilla-JS correction (Phase 4, ported alongside the rest of the UI layer),
dependency-vulnerability scanning as a genuinely new CI step (Phase 5). Object storage,
multi-instance cache invalidation, and `handler`'s team-scale coupling stay explicitly out of
scope — no real need has forced any of them yet, per this project's own "Infer Before Configure"
principle. `roadmap.md`'s item 21 addendum carries the matching pointer.
