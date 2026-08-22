# UI/Interaction Cluster — CAP-V16/V17/V18/V14 Tier 2/V15/V19

> Study 28 of the Capability Roadmap.
>
> All six items from `roadmap.md`'s Track D (`benchmarks/008-ui-workflow-interaction-benchmark.md`),
> done in one batch per the user's explicit choice. Committed one phase per capability; this
> document grows one section per phase rather than splitting into six near-identical small files,
> since all six share one theme and several share design decisions worth cross-referencing.
>
> **A testing limitation, named upfront for the whole cluster, not per-phase**: this project's
> entire conformance suite (`conformance/run.sh`) is HTTP black-box (`curl`) — there is no
> headless-browser tooling anywhere in this repo. For the phases needing hand-written JS
> (V15, V19, V14 Tier 2's drag interaction), conformance can prove the server emits the correct
> HTML/JS wiring (a markup assertion) and that the underlying server-side enforcement the JS is
> only a convenience layer for still works — it cannot execute JS or observe live DOM behavior.
> Each such phase's own section names this explicitly rather than overclaiming automated coverage.

---

## Phase 1 — CAP-V17: SLA/deadline countdown badge

**What was built**: `ViewConfig` gains `SlaField`/`SlaWarningDays`. A new pure function,
`slaUrgency` (`internal/handler/format.go`), computes a days-remaining label and an
`overdue`/`warning`/`ok` bucket from a date string at request time — no existing helper covered
this (CAP-A11's own date-arithmetic, `resolveDateArithmetic`/`addBusinessDays`, only adds a
forward offset to a base date; a countdown needs the reverse, a small self-contained addition,
not a generalization of that code). Wired into the exact same two render-time loops CAP-F14's
`computedValue` already uses (`List`'s row-building loop, `Detail`'s field loop) — when a Field's
id matches the declared `SlaField`, its cell substitutes a new `SlaBadge` templ component (reusing
`StatusBadge`'s pill CSS convention, keyed on urgency instead of a hardcoded status string) for
the raw date value.

**Proof**: new `seeds/029_sla_badge_lab.sql` (new Application, new `Ticket` Machine). T167: a
ticket due in 2020 (clearly past) renders the overdue badge. T168: a ticket due in 2099 (clearly
future) does not — the positive/negative pairing proving T167 isn't vacuously always-true.
Deliberately not testing the `warning`-bucket boundary against `NOW()`-relative dates — a
threshold-relative-to-today fixture would be flaky if this suite is ever run months apart from
when it was written; the two unambiguous buckets are what's proven. **168/168 conformance
passing, zero regressions on the prior 166**, confirmed on a fresh isolated schema.

**Registry impact**: `capability-registry.md`'s CAP-V17 row ❌→✅.

---

## Phase 2 — CAP-V18: resource-grouped calendar

**What was built**: `ViewConfig` gains `ResourceField` (a `reference` Field id). `calendarTimeline`'s
existing single-dimension grouping (`internal/handler/views.go`) — "sort by `date_field`, then a
linear scan flushing on each date change," not a SQL `GROUP BY` — is extracted into a reusable
`groupByDate` helper and reused per-resource when `ResourceField` is set, rather than being
generalized into a new mechanism. Every resource gets its own section, including an idle one with
zero dated records — fetched directly from the resource Machine (`h.records.List` on its
`target_machine`), not derived from the dated records themselves, which would silently drop a
resource nobody scheduled anything against yet. Unset `ResourceField` is byte-identical to CAP-V07's
existing behavior — zero risk to its own tests.

**Proof**: new `seeds/030_resource_calendar_lab.sql` (`Staff`, `Appointment` referencing Staff + a
date field). T169: two staff, each with a same-day appointment — each staff's own rendered
section contains only their own appointment title, verified via a `python3` substring-isolation
check (the rendered HTML isn't reliably line-wrapped in a way plain `grep -A`/`-B` could depend
on, so isolation is checked by finding each resource's `<h2>` heading and slicing to the next one,
not by counting lines). T170: a staff member with zero appointments still gets a section
containing "No dated records" — proving the grouping is resource-driven, not a filtered date list
that would drop an idle resource silently. **170/170 conformance passing, zero regressions**,
confirmed on a fresh isolated schema (two failures seen on a *reused*, repeatedly-run schema
during development — T151/T154 — were the same pre-existing non-idempotent-mid-run-seed artifacts
already diagnosed earlier this session, not caused by this phase; the fresh-schema run confirms
that directly).

**Registry impact**: `capability-registry.md`'s CAP-V18 row ❌→✅.
