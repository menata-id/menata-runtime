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

## 4. What's settled vs. still open (summary)

| Question | Status |
|---|---|
| Do `prototype/`/`benchmarks/` need renaming or restructuring now? | **No** — settled, they're done, left as-is |
| Does `guides/writing-runtime-metadata.md` (+ siblings) need to move? | **No** — settled, correctly placed under the corrected frame |
| Is a new top-level folder being created for the real application? | **Yes** — settled, named `app/` |
| Does `app/` graduate `prototype/go`'s code, or start from zero? | **Open** — owner wants further discussion before deciding |
| When does `app/` get created, and by whom (this session or a dedicated future one)? | **Open** — not raised yet, follows from the row above |
