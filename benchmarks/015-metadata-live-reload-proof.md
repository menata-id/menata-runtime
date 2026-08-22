# Metadata Live Reload — CAP-X04 Option A, Proven

> Study 23 of the Capability Roadmap.
>
> `capability-registry.md`'s CAP-X04 row was reviewed twice before (2026-07-12) and deliberately
> left ❌ both times — the concern on record: rewriting `Loader.LoadAll`/`interpreter.New` is a
> high-risk architectural change touching the mechanism every other capability depends on, "not
> a batch-sized addition." This study revisits that decision because B5 (`change_policy`,
> CAP-W07) turned out to require it: `change_policy`'s entire point is expressing "does this
> metadata change apply to in-flight records," a question that has no observable answer under a
> restart-only deploy model, where every change reaches every record atomically with no live
> boundary to discriminate across. Building B5 without CAP-X04 first would have meant compiling
> a guard for a runtime condition this prototype could not yet produce — not a provable
> capability, per `capability-lifecycle.md`'s own admission bar.
>
> Method note: unlike the two prior reviews, this pass did the actual architectural research
> (call-site inventory, package-level state audit, ADR-002 re-read) before deciding scope, and
> built exactly the piece that unblocks B5 — not the originally-sketched "CAP-X04 + CAP-X11
> together" bundle (see `roadmap.md`'s "done, Option A only" note for why that bundle was
> dropped). | Created: 2026-08-22

---

# What was reviewed and found

`docs/decisions/002-metadata-loading.md` (ADR-002, written 2026-07-04) had already designed the
answer, unused for six weeks: three options, Option A (admin-triggered reload, atomic pointer
swap) named explicitly low-risk and "useful regardless" of what else gets chosen; Option C
(`LISTEN/NOTIFY`, automatic) the stated long-term answer, not required now.

Three questions this study answered before writing any code:

1. **Is repeatedly calling `Loader.LoadAll` actually safe?** Audited `internal/metadata` for
   package-level mutable state — found none (`Loader` only wraps the DB pool;
   `validateReferences`/`validateOperators` build fresh local maps every call; the one
   package-level `var` in the whole package, `compile.go`'s `durationRe`, is a read-only
   compiled regex). Confirmed: yes.
2. **What is the actual blast radius of moving off a frozen `*Interpreter`?** Grepped every
   reference to the type — exactly two files (`cmd/server/main.go`, `internal/handler/handler.go`)
   declare it; five files inside `internal/handler` consume it, at 91 call sites of the literal
   pattern `h.interp.` plus one bare usage. Fully inventoried before touching anything, not
   discovered mid-edit.
3. **Does CAP-X11 need to be built at the same time?** No — CAP-X11 (lazy per-workspace loading,
   LRU, `singleflight`) solves a *scale* problem (many workspaces, each paying full boot-time
   load cost) this single-workspace-mostly prototype has no measured pressure for. B5's actual
   blocker was a *mechanism* to change metadata live, not an optimization on top of one. Bundling
   X11 in would have been building ahead of measured need — this project's own "Infer Before
   Configure" principle, already invoked elsewhere in the registry (CAP-X10's own row) to defer
   exactly this class of premature scale work.

---

# What was built

**`internal/interpreter/store.go`** (new) — `Store`, a thin `atomic.Pointer[Interpreter]`
wrapper: `NewStore`, `Get`, `Swap`. No lock; safe for arbitrarily many concurrent readers and
the rare admin-triggered writer.

**The 91+1 call sites** — `h.interp.` mechanically transformed to `h.interp.Get().` via `sed`
across `handler.go`/`requirement.go`/`processmap.go`/`scheduler.go`/`api.go`; one bare usage
(`findMachineContainingField(h.interp, ...)`) fixed by hand. `go build` is the exhaustiveness
proof here — a missed site is a compile error (type mismatch), not a latent runtime bug, which
is precisely why this kind of wide mechanical change is safe to make confidently in one pass.

**Two more readers, both requiring a "call `.Get()` fresh, don't cache" discipline**
(`cmd/server/main.go`): `sessionAuth`'s `visitorAuth` check (CAP-P07) now resolves the current
interpreter inside the per-request closure, not once at middleware-construction time; `runScheduler`
now resolves it inside the ticker loop, not once before it starts. Either mistake would have
silently defeated the whole feature — a stale capture keeps serving boot-time metadata forever,
reload or not — so both are called out explicitly in code comments, not left to be rediscovered.

**`POST /admin/reload`** (`Handler.Reload`, Admin-only) — re-runs `LoadAll`; on success, builds
a fresh `Interpreter` and atomically `Swap`s it in; on failure, surfaces the error to the admin
as a 500 and leaves the old interpreter completely untouched. This is the one property the whole
feature exists to guarantee, restated because it is the actual point: **a bad reload must never
brick the live server.** A small form ("Reload metadata") was added to the existing
`/admin/users` page rather than a new one.

---

# Proof

New, self-contained fixture `seeds/023_reload_lab.sql` — deliberately **not** part of `make
seed`'s boot-time list, since the whole claim is that it becomes servable without ever being
loaded at boot.

| Test | Claim | Result |
|---|---|---|
| T151 | `mch_reload_case` 404s before the seed is applied; after applying it via a direct `psql` call and triggering `POST /admin/reload`, it 200s — with no server restart anywhere in between | ✅ PASS |
| T152 | A non-Admin's `POST /admin/reload` is rejected (403) | ✅ PASS |
| T153 | A deliberately malformed row (a `reference` Field targeting a nonexistent Machine — the same dangling-reference shape CAP-F13's `validateReferences` already rejects) makes the next reload fail (500); an unrelated, already-working Machine (`mch_leave_request`) keeps responding 200 on the still-good old interpreter | ✅ PASS |

**154/154 conformance passing, zero regressions on the prior 151** — the real proof that
swapping `h.interp` from a field to a `Store`-backed accessor across 92 call sites didn't
silently break anything.

The malformed row from T153 is deleted immediately after the assertion, inside the test itself
— not just cleanup hygiene: left behind, it would fail the very next full server *restart* (not
just the next reload), since boot-time `LoadAll` failure still calls `os.Exit(1)` by design —
only the reload path gained the "don't brick, surface the error" protection this study built.

---

# What this does and doesn't close

**Closes:** the actual blocker on B5 — a metadata change can now reach a running system while
records stay open, the precondition `change_policy` needs to mean anything at all. Also
unblocks CAP-X08's import half (its own row named "a real reload story" as the reason it wasn't
built).

**Does not close, named explicitly:** Option C (`LISTEN/NOTIFY`, automatic reload — still ADR-002's
stated long-term answer); CAP-X11 (lazy per-workspace loading — a separate scale concern, no
measured pressure yet); a stronger per-request-snapshot consistency guarantee (`Store.Get()` is
called fresh at each point of use rather than captured once per request, so a request whose
processing straddles the exact instant of a `Swap` could theoretically read old-then-new data
within itself — acceptable for a rare, admin-triggered action where the worst case is a
transient inconsistent read, never data corruption, since writes still go through the separate
`workspaceTx` transaction).

---

# Registry impact

`capability-registry.md`'s CAP-X04 row moved ❌→⚠️ (Option A done; Option C and CAP-X11 remain
open, named explicitly — not ✅ for that reason). `roadmap.md`'s Sequencing Guide updated: B5 and
CAP-X08's import half move from BLOCKED to READY; CAP-X11 demoted from a co-requisite to an
independent Track A item with no measured urgency.
