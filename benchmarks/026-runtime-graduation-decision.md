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

## 4. What's settled vs. still open (summary)

| Question | Status |
|---|---|
| Do `prototype/`/`benchmarks/` need renaming or restructuring now? | **No** — settled, they're done, left as-is |
| Does `guides/writing-runtime-metadata.md` (+ siblings) need to move? | **No** — settled, correctly placed under the corrected frame |
| Is a new top-level folder being created for the real application? | **Yes** — settled, named `app/` |
| Does `app/` graduate `prototype/go`'s code, or start from zero? | **Recommended: graduate** (§6, 2026-09-06) — stronger than §3's own case alone, after the no-containers/thin-architecture corrections; still awaiting the owner's explicit final confirmation before treated as settled |
| What hardens after graduation, and in what order? | **Open, but scoped** — §5's gap list (migrations tooling, storage abstraction, API versioning, secrets, dependency scanning, rate limiting, test depth), sequenced by whoever picks up `app/`'s first real milestone |
| When does `app/` get created, and by whom (this session or a dedicated future one)? | **Open** — not raised yet, follows from the rows above |
