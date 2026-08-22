# In-App Navigation / Sub-Nav Benchmark

> Prompted directly by a question: where would a "menu or sub-navbar showing an app's own
> feature pages" be recorded, and does world-class practice actually call for one — run against
> every case in `../case-portfolio.md` that has more than one Machine, per the request ("tolong
> lakukan ke aplikasi pada case yang ada").
>
> Distinct from `008-ui-workflow-interaction-benchmark.md` (that study covered *within-page*
> interaction — forms, lists, calendars) and from CAP-O03 itself (workspace home + a one-time
> drill-in landing page). This is about **persistent, cross-page** navigation — can a user get
> from one feature of an app to another *without losing their place* — which is a distinct
> concept in `006-runtime-model.md`'s own hierarchy (Navigation, sibling to Page/View).
>
> Status: v0.1 | Created: 2026-07-12

---

# The gap, stated precisely

`internal/ui/layout.templ`'s `navBar` (this runtime's only persistent nav element) is
**global**, not per-Application: logo/home link, Search, Notifications, Admin (if applicable),
identity, Logout — the same five links on every page regardless of which Application or Machine
the user is currently working in. `CAP-O03` (✅) gives an Application exactly one entry point —
`GET /apps/{applicationID}` lists that Application's own Machines as a landing page — but once
the user clicks into any one Machine (its List, a record's Detail, a Form), that list is gone.
Reaching a *different* Machine in the *same* Application requires: click the logo (back to
workspace home, losing the Application context entirely) → click the Application card again →
pick the other Machine. There is no way to jump sideways.

The underlying data link already exists — `Interpreter.ScopeFor(machineID)` resolves the owning
`applicationID` from any Machine id, and `MachinesForApplication(appID)` already lists its
siblings (`AppMachines`'s own implementation, above) — this is a rendering gap, not a missing
data model.

---

# World-class reference

| Platform | Pattern |
|----------|---------|
| Salesforce | App Launcher switches *between* apps; inside one, a persistent tab bar of that app's own objects stays visible on every object's List/Detail page |
| Odoo | Top App Switcher (grid icon) plus a horizontal menu bar showing the *current* app's own modules as dropdowns — never disappears while working inside that app |
| Frappe/ERPNext (Desk) | A persistent left sidebar lists every Doctype in the current Workspace — visible from any List/Form/Report screen |
| ServiceNow | Application Navigator — a persistent, filterable left panel of every module/table, grouped by application |
| Jira | A project's own sidebar (Backlog, Board, Issues, ...) stays visible on every page inside that project |
| Notion | A persistent left sidebar of pages/subpages — the single most consistent element across the entire product |

**The pattern is universal across every category surveyed** (CRM, ERP, ITSM, project
management, knowledge base) — not one of them requires returning to a home screen to move
between two features of the same app. This alone is a strong signal, but per this registry's own
standing rule (Rule A1), a benchmark alone is not admission — real case evidence is still
required.

---

# Case evidence — Applications with more than one Machine

An Application with exactly one Machine has no sideways-navigation need at all (there's nowhere
else to go); the gap only bites once an Application has two or more. Counting directly from each
case's own declared Machines in `case-portfolio.md`:

| Case | Machines in one Application | Sideways-nav need |
|------|------------------------------|---------------------|
| 3 — Document Approval | Approval Document, Approval Step | Real — deciding a Step needs the Document's own context, today reachable only via the Step's own `reference` link back, not a general sub-nav |
| 5 — Inventory | Item, Item Unit Conversion, Stock Movement, Stock Ledger | Real — checking an Item's ledger while recording a new Movement is a routine task |
| 6 — Petty Cash | Fund, Voucher, Period | Real — a Custodian moves between Vouchers and the Fund's own balance constantly |
| 8 — Payment Confirmation | Invoice, Payment Webhook Event, Payment | Real — reconciling a Payment against its Invoice |
| 9 — Accounting | Chart of Account, Journal Entry, Journal Entry Line, Fiscal Period | **Strongest evidence** — a bookkeeper's ordinary workday moves between the COA, new Entries, and the Trial Balance report constantly; this is the case the original UI-workflow question itself was raised against |
| 11 — Social App | Post, Follow, Like, Comment | Weaker — most of a user's work is on Post/Feed; Follow/Like/Comment are supporting, not independently navigated to |
| 12 — Community Site | Group, Membership, Event, Points, Badge Award | Real — a Group organizer moves between Events and Membership routinely |
| 14 — Lending Services | Loan Application, Loan, Repayment Schedule Entry, Repayment | Real — recording a Repayment while checking the Schedule |
| 15 — E-commerce | Product, Cart, Cart Item, Order, Order Line | Real — an operator moves between Product catalog and Orders |
| 19 — Project Management | Board, List, Card, Checklist Item | Real — though CAP-V14/CAP-V14 Tier 2's board view itself already keeps List/Card in one screen; this case's own nav need is smaller than the others since the board IS the sub-nav for its own contents |
| 20 — Hospital System | Patient, Appointment, Medical Record, Prescription | Real — a clinician moves between a Patient's Appointments and Medical Records within one visit |
| 21 — E-learning | Course, Lesson, Enrollment, Certificate | Real — an instructor moves between Courses and Lessons |

**11 of the 21 cases** have a multi-Machine Application with a real, describable
sideways-navigation need — this clears the dual-evidence bar by a wide margin (a single case
would have been enough per the standard every other admitted capability in this registry met).

---

# Admission test

| # | Criterion | Result |
|---|-----------|--------|
| A1 | Dual evidence | ✅ — 6 named world-class platforms across unrelated categories + 11 independent cases |
| A2–A5 | (full text: `capability-lifecycle.md` §2) | ✅ — no conflict with an existing capability; `CAP-O03` already resolved the exact data link this would render (`ScopeFor`, `MachinesForApplication`) |

**Architecture check:** a per-Application sub-nav is a server-rendered list of links — the exact
same rendering `AppMachines` already does for its own landing page, just also rendered as a
persistent strip/sidebar on every page scoped to that Application (List, Detail, Form, Calendar,
...). No JS, no new route, no conflict with the no-SPA posture — purely an extension of
`internal/ui/layout.templ`'s existing `Page`/`navBar` composition to also accept an optional
Application context.

**Verdict: ADMITTED.**

---

# New registry candidate

| Candidate | Description | Evidence | Status |
|-----------|--------------|----------|--------|
| **CAP-O03 Tier 2** | Persistent, Application-scoped sub-navigation — every page belonging to a Machine renders a secondary nav strip listing that Machine's own sibling Machines (within the same Application, permission-trimmed the same way `AppMachines`/`Apps` already are), so a user can move sideways between an app's own features without returning to the workspace home. Extends `CAP-O03`'s existing `ScopeFor`/`MachinesForApplication` data link — no new metadata concept, a rendering-layer addition to `internal/ui/layout.templ` | 11 of 21 portfolio cases (Case 9 strongest), 6 named world-class platforms across CRM/ERP/ITSM/PM/knowledge-base categories | Proposed, as an extension of CAP-O03 (already ✅) |

Registration only, per this registry's own standing process (`case-portfolio.md` §Process per
case: register, then implement as a separate step) — not implemented by this study.

---

# Follow-on finding (2026-08-22) — within-Machine navigation to a Machine's own auxiliary Views

**Surfaced directly by the owner**, while trying the just-shipped CAP-V14 Tier 2 kanban board
(`benchmarks/020-ui-interaction-cluster-proof.md`) — the board page has no in-app link anywhere;
reaching it required typing `/{machineID}/board` by hand. Checked against the rest of this
codebase before writing anything up: it's not a kanban-specific gap. **Every one of a Machine's
own auxiliary Views — Calendar, Timeline, Report, Dashboard, Board, Document — is reachable only
by a hand-typed URL.** `grep`-ing `internal/ui/*.templ` for a link to any of `/calendar`,
`/timeline`, `/report`, `/dashboard`, `/board` returns nothing; `subNavFor`
(`internal/handler/handler.go`, CAP-O03 Tier 2's own mechanism) only links **sideways, to sibling
Machines** in the same Application — it was never scoped to link **within** one Machine, to that
same Machine's own alternate View types. Those are two different navigation axes this repo has
now built one of and not the other:

| Axis | Question | Status |
|---|---|---|
| Cross-Machine, same Application | "How do I get from Journal Entry to Chart of Account?" | ✅ CAP-O03 Tier 2 (`subNavFor`, persistent sub-nav strip) |
| Within-Machine, across View types | "How do I get from Task's List to Task's own Board/Calendar/Report?" | ❌ no mechanism at all — URL-only |

**Owner's explicit direction (2026-08-22): do not implement this now.** UI/usability work of this
shape — and any other not-yet-done work touching visual layout — is explicitly held back from
implementation until a **design prototype** exists first. The ask is broader than "add a link to
the Board page": the owner wants a **standard for where navigation components/buttons/controls
are placed, defined once and shared across every case in the portfolio** (`case-portfolio.md`'s
21 cases), not a one-off fix scoped to this single gap. **Mobile-first, explicitly** — the owner's
own instruction: design the mobile layout first, desktop follows after, not the other way around.
This project's current UI (`internal/ui/*.templ`, Tailwind) has never been designed mobile-first —
`layout.templ`'s nav bar / `subNavBar` strip / this row's own proposed navigation controls are all
desktop-table/toolbar shapes with no evidence anyone has checked them at a phone viewport, so the
design-prototype pass needs to establish the mobile layout from scratch, not retrofit the existing
desktop chrome down to a smaller screen. Concretely, a future design pass needs to answer, for the
whole app, before any of this is implemented:

- Where does a link to a Machine's own auxiliary Views live relative to the existing List
  toolbar (same row as Export/Import/New — `internal/ui/list.templ`'s current button cluster —
  or a distinct row/strip of its own, closer to `subNavBar`'s own placement)?
- Does it reuse `subNavBar`'s existing strip pattern (View-type tabs alongside sibling-Machine
  links) or does View-type switching need a visually distinct control from Machine switching, so
  the two navigation axes above don't collapse into one confusing row?
- Is placement driven by which auxiliary Views a Machine actually declares (today: 0–1 of
  Calendar/Timeline/Report/Dashboard/Board/Document per Machine across the seed data this
  project has), or does the standard need to hold up for a Machine that declares several at once?
- What's the shared visual/interaction vocabulary (icon vs. label, active-state styling) at a
  **phone viewport first** — collapsed into a bottom tab bar / hamburger / sheet, not a row of
  text links that only works at desktop width — with the desktop layout designed as the second
  pass, informed by whatever the mobile pass establishes, not the other way around?
- How do the two navigation axes above (cross-Machine sub-nav, within-Machine View-type switch)
  both fit into a small screen's limited chrome without stacking into three separate nav rows —
  `subNavBar`'s existing strip was never designed against a phone-width constraint either, so this
  pass may need to revisit its layout too, not just add a new row next to it?

**Registered, not scheduled**: see `capability-registry.md`'s "Tracked but Not Yet Studied" →
Navigation row and its own new `CAP-O03 Tier 3` candidate row for the tracking record, and
`roadmap.md`'s Track F for the queued-but-blocked status. **This is a note for a future
implementation session, not a task for this one** — the actual design-prototype pass (mockups,
placement decisions, a component-vocabulary writeup) hasn't been done yet and is explicitly out
of scope for whichever session picks this up next until that groundwork exists.
