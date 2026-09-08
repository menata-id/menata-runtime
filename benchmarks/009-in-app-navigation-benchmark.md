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
> Status: v0.4 — gains a full layer-by-layer implementation plan for `CAP-O03` Tier 5 (declared
> navigation, owner-requested "Option B" following v0.3's own trade-off): schema
> (`navigation_entries`, `Menu`/`MenuLinkContent`-shaped), loader validation, model, handler
> dispatch (declared-first, infer-fallback, zero migration cost for every existing Application),
> conformance sketch, and a phased rollout (Phase 1 = same-Application Machine/View/group targets;
> Phase 2 = external URLs + cross-Application entries, deliberately deferred). Recommends building
> this directly instead of `CAP-O03` Tier 4 — Tier 5 subsumes it entirely. `capability-registry.md`
> v0.67 and `case-portfolio.md`'s new Case 10 note (v0.35) carry the admission itself | Previously
> v0.3 — gains Drupal as a 7th benchmarked platform (`Menu`/`MenuLinkContent` as first-
> class entities, not a flag on the content they link to — architecturally distinct from the
> other six, per the owner's own observation that this study leaned too heavily on ERP/metadata
> platforms whose menu is comparatively limited) and a direct read against this runtime's OWN
> foundational docs: both `004-runtime-metadata.md` and `006-runtime-model.md` already name
> `Navigation` as a first-class hierarchy peer of Page/View/Service, yet three tiers of real
> building have kept it 100% inferred, never once a declared artifact — a real, previously-unnamed
> gap between the stated model and what shipped. States the Option A (Tier 4's exception flag,
> kept as admitted) vs. Option B (Navigation as a declared Drupal-shaped entity, would subsume
> Tier 4 entirely, not admitted — no case has asked for it) trade-off directly, as a
> recommendation for the owner, not a decision made here. **Correction on this file's own line 12
> below**: "Navigation, sibling to Page/View" is imprecise against `006`'s actual tree — Navigation
> is a sibling of *Machine* (both direct children of Application); Page/View are children of
> Machine, one level deeper. Kept as originally written per append-don't-rewrite; this note is the
> fix. Previously v0.2 — gains a 2026-09-08 follow-on finding: a deeper benchmark of HOW six platforms
> curate nav visibility (not just whether they have persistent nav), formal UI menu-component
> standards (Material Design 3's 3–7 destination threshold, Miller's Law with its own honest
> caveat, NN/G information scent), and a survey of adjacent future needs (nav grouping, per-user
> favorites, ordering) read against the admission criteria and left un-admitted pending real case
> pressure. Backs `CAP-O03` Tier 4 (curated navigation visibility, ❌ Proposed, `capability-
> registry.md` v0.64) | Previously v0.1 | Created: 2026-07-12

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

---

# Follow-on finding (2026-09-08) — curated navigation visibility (`CAP-O03` Tier 4)

Owner-requested, on top of the same-day admission of `CAP-O03` Tier 4 (`capability-registry.md`
v0.64, `case-portfolio.md`'s new Case 3 note): *"kajian kemungkinannya apa saja kebutuhan atas
kapabilitas ini... lakukan benchmark ke aplikasi di use case dan standar tampilan komponen ui
menu"* — survey what the realistic range of future need looks like around this capability, and
benchmark against both real applications' use cases and formal UI menu-component standards, not
just the four platforms the admission row already cited in passing.

## World-class reference — HOW each platform curates, not just whether it has a persistent nav

Tier 2 (above) already established that persistent, cross-page sub-navigation is universal. This
pass goes one level deeper: once a platform has that persistent nav, what mechanism does it give
an admin to keep it from listing every single object/table/model?

| Platform | Curation mechanism | Scope | Source |
|---|---|---|---|
| Salesforce | Setup → App Manager → Edit App → **Navigation Items** — an app's tab set is an explicit, ordered allow-list; an object left out simply isn't a tab. A **"Tab Hidden"** profile setting is the softer variant: the object still gets an icon and stays reachable via App Launcher search, just not in the persistent tab strip | Per-App (an object can be a tab in one App, not another), admin-declared, workspace-wide default | [Personalized Navigation Considerations](https://help.salesforce.com/s/articleView?id=sf.user_userdisplay_tabs_lex_considerations.htm) — a new tab an admin adds does NOT automatically appear for a user who already personalized their own nav, a real "default vs. override" interaction worth naming; [Show or Hide Tabs for Users](https://help.salesforce.com/s/articleView?id=000385181&type=1) |
| Odoo | `ir.ui.menu` record's `active` field — set `False` and the menu entry disappears without deleting the underlying model/data; group-based visibility layers on top for role-scoping | Per-menu-record, admin-declared (Technical Settings, dev mode) | [Odoo forum — how to hide menus](https://www.odoo.com/forum/help-1/how-to-hide-menus-59356) |
| Frappe/ERPNext | A Workspace's own `is_standard` + membership in the **PUBLIC** vs. **MY WORKSPACES** section; a DocType with no Workspace shortcut pointing at it simply has no sidebar entry, reachable only by direct link/search | Per-Workspace (admin-curated shared default) vs. per-user private Workspaces (personal, not shared) — Frappe is the one platform surveyed with BOTH an admin-curated default AND a fully separate per-user personal layer | [Frappe docs — Workspace](https://docs.frappe.io/framework/user/en/desk/workspace) |
| ServiceNow | Application Navigator module visibility by role — **grant-only, not deny-based**: a module becomes visible once a role is added to it, but there is no native "hide this module even though the role would otherwise see it" — real admins work around this with a query Business Rule on `sys_app_module`, a documented limitation, not a feature | Per-module, role-additive only | [ServiceNow Community — hide modules even if a user has the correct roles](https://www.servicenow.com/community/developer-forum/hide-modules-even-if-a-user-has-the-correct-roles/m-p/1825552) |
| Jira | Project sidebar → **Customize sidebar** — an admin explicitly shows/hides/reorders items; the change "will affect everyone who has access to the project," i.e. an admin-set shared default, not per-viewer | Per-project, admin-declared, shared | [Atlassian — Navigate projects with the sidebar](https://confluence.atlassian.com/jirasoftware/navigate-projects-with-the-sidebar-1528532974.html) |
| Notion | **Favorites** — any user stars a page onto their own sidebar shortcut list; independent of whatever the shared Workspace/Shared sections already show. Purely additive and personal, not a hide mechanism at all | Per-viewer, personal, opt-in | [Notion — navigate with the sidebar](https://www.notion.com/help/navigate-with-the-sidebar) |
| Drupal | **Menu is a first-class entity type, not a flag on the content it links to.** `Menu` itself (e.g. "Main navigation") is a **Configuration Entity** (`@ConfigEntityType`, id `menu`); each individual link is a separate **`MenuLinkContent`** — a real, independently CRUD-able, fieldable, revisionable Content Entity with its own `title`, `menu_name` (which menu it belongs to), `weight` (ordering), `parent` (arbitrary-depth nesting), `link.uri` (an internal route, a specific query/view, or a fully external URL — never required to correspond to one content type), and `enabled` | Per-menu, admin-authored, fully decoupled from the target — a link can point at anything, or nothing (`route:<nolink>`, a pure grouping header) | [Drupal API — class Menu](https://api.drupal.org/api/drupal/core!modules!system!src!Entity!Menu.php/class/Menu/11.x), [Menus are now configuration entities](https://www.drupal.org/node/1888504), [Drupal API — MenuLinkContent](https://api.drupal.org/api/drupal/core!modules!menu_link_content!src!Entity!MenuLinkContent.php/class/MenuLinkContent/11.x), [Drupal.org — Menu links examples](https://www.drupal.org/docs/contributed-modules/yaml-content/examples/menu-links) |

**Reading across all seven**: the first six confirm `CAP-O03` Tier 4's own already-admitted shape
(Salesforce Navigation Items, Odoo `active=False`, Frappe Workspace membership, Jira Customize
sidebar — an **allow/hide default that changes what everyone with access sees**, a Machine-level
exception flag, not per-viewer state). Only Notion's Favorites and Frappe's *private* Workspaces
are genuinely **per-user personalization** — a materially different capability no case has asked
for yet (see "Possible future needs" below). ServiceNow's own documented limitation (grant-only
visibility, no native hide) independently reinforces this study's earlier A4 finding: a plain
permission/role mechanism is provably insufficient for "readable but not a menu destination" — a
real platform hit the exact same wall, not a hypothetical concern.

**Drupal is architecturally different from all six of the others**, not just a seventh data point
— worth its own read, per the owner's own framing that this study's first pass leaned too heavily
on ERP/metadata-based platforms whose own menu is comparatively limited. The other six all express
curation as a **property on the thing being shown** (a flag, a role grant, a Workspace membership).
Drupal expresses it as a **freestanding structure that merely references** what it shows — the
menu link is real data in its own right, decoupled enough to point at a route that isn't tied to
any single content type at all, be nested to any depth, or exist purely as a grouping header with
no destination. This is a materially more general shape than "show or hide this Machine."

## Reading against Menata Runtime's own foundational model

Checked directly, not assumed, before treating Drupal's shape as merely "a nice idea from another
platform": does this runtime's own design already have an opinion here? It does, in two places,
and they don't fully agree with each other:

- `006-runtime-model.md`'s own Runtime Hierarchy places **`Navigation` as a sibling of `Machine`**,
  directly under `Application` — the same tier as Machine itself, not a property hanging off it.
  Its own one-line scope: *"Navigation may include: menus, breadcrumbs, tabs, shortcuts, quick
  actions"* — explicitly naming more than a Machine-visibility toggle.
- `004-runtime-metadata.md`'s own Runtime Metadata Hierarchy instead nests **`Navigation` under
  `Machine`**, alongside Page/View/Service/Workflow/API/Configuration — a Machine-owned concern.

**Not a contradiction once read against what this repo has actually built**: `CAP-O03` Tier 2
(cross-Machine, Application-scoped sub-nav) and Tier 3 (within-Machine, auxiliary-View switching)
already map cleanly onto exactly these two positions — 006's Application-level placement is Tier
2's own axis, 004's Machine-level placement is Tier 3's own axis. Both tiers are real, ✅, and
conformance-tested. What neither foundational document's promise has ever been given, in three
tiers of real implementation: **`Navigation` has never once been materialized as its own declared
metadata** — `subNavFor`/`AppMachines`/`viewNavFor` (Tiers 2–4's whole lineage, this row's own
proposed Tier 4 included) all *derive* their link lists by scanning `Machine`/`View` structure at
render time, per `001-design-principles.md` §6 "Infer Before Configure" taken to its fullest
extent. Both foundational documents list Navigation as a peer of Page/View/Service — real Grammar-
level artifacts that ARE stored, declared, independently authored — yet Navigation alone has stayed
100% inferred since CAP-O03's own original 2026-07-12 implementation. This is a real, previously
unnamed gap between the runtime's own stated model and what three tiers of real building actually
shipped — named here per "silence is not a decision," not a defect in Tier 2/3 (both are genuinely
✅ and correctly scoped for the case pressure that justified each), but a ceiling worth being
honest about before extending the same inference-only lineage a fourth time.

**The trade-off this creates for Tier 4, stated directly:**

| | Option A — exception flag (Tier 4 as already admitted) | Option B — Navigation as a declared entity (Drupal-shaped) |
|---|---|---|
| What it is | One new `machines.config` key, read by the two existing inference call sites | A real new metadata artifact — `Menu`/`NavigationLink`-shaped rows, ordered, nestable, referencing a Machine/View OR (later) something else entirely |
| Consistent with | `001-design-principles.md` §6 "Infer Before Configure" — configuration should describe exceptions, not re-declare what's already inferable | `004`/`006`'s own Runtime Hierarchy — Navigation as a first-class peer of Page/View/Service, finally made real instead of perpetually inferred |
| Solves the observed problem (Approval Step/Signature cluttering `app_approval`'s strip) | ✅ completely — that problem is pure visibility, nothing more | ✅ also, and more — same outcome via "just don't add a link," Salesforce/Drupal's own allow-list pattern |
| Can express a menu entry that ISN'T a Machine (external link, a specific pre-filtered View, a grouping header, a "quick action") | ❌ no — inference has nothing to point at | ✅ yes — this is exactly the generality 006's own "menus, breadcrumbs, tabs, shortcuts, quick actions" line already named as in-scope for Navigation |
| Scope / cost | Small — sketch already in this row's own registry entry, one config key, two call sites | Large — new Grammar-adjacent area, new schema + loader validation + admin authoring surface + its own admission test; genuinely closer to a new capability than a Tier of this one |
| A4 non-composability if Option B existed first | Tier 4 would be **subsumed**, not merely satisfied — an explicit allow-list needs no separate "hide" flag, matching Drupal/Salesforce's own "don't add the link" pattern directly | N/A — this is the more general mechanism |
| Real case pressure today | ✅ yes (this row's own Case 3 note) | ❌ no case has ever asked for a non-Machine menu entry, cross-Application grouping, or breadcrumbs/quick-actions as declared metadata |

**Recommendation, not a decision — the owner's call**: keep Tier 4 registered exactly as admitted
(A1–A5 pass on the narrower shape, real case pressure exists for it specifically) and build it if
and when implementation is prioritized — it is correct, cheap, and fully solves the problem in
front of us today. But name Option B honestly as the more architecturally faithful long-term shape
this runtime's OWN foundational docs already pointed at before Tier 4 was ever proposed, so a
future session doesn't have to rediscover this tension from scratch. **Not admitted as its own
capability by this pass** — A1 fails on its own terms (no case has asked for anything Option A
can't already do); revisit if a real case ever needs a menu entry that isn't a Machine.

## UI menu-component standards

Three established sources, checked directly rather than assumed, since a governance document that
cites "world-class practice" should be able to say which claims are load-bearing and which are
popular but weaker than commonly believed:

- **Material Design 3 — Navigation rail**: don't use a (collapsed) navigation rail below 3
  destinations (use tabs instead) and don't use it above 7 (use the expanded rail/drawer instead)
  — a concrete, numeric, sourced threshold. [m3.material.io — Navigation rail
  guidelines](https://m3.material.io/components/navigation-rail/guidelines). Directly relevant
  beyond Tier 4 itself: `CAP-O03` Tier 2's own `subNavBar` strip renders **every** permission-
  visible Machine in an Application today, uncapped — an Application with more than ~7 Machines
  would already exceed this threshold even after Tier 4 hides the genuinely-not-a-destination
  ones, a possible future fitness-function check (see below), not something to build now.
- **Miller's Law ("7±2")**: widely cited to justify capping menus around 5–9 items, but the
  honest caveat matters here — Miller's 1956 research measured short-term-memory chunk capacity,
  not interface item counts, and applying it directly to menu design is a popularized
  overextension of the original finding, not itself HCI research
  ([Stéphanie Walter — "Your navigation menu doesn't need Miller's 7±2
  rule"](https://stephaniewalter.design/blog/your-menu-doesnt-need-millers-7-plus-minus-2-rule/)).
  Cited here for completeness since it's the number most often invoked in this exact conversation,
  not as independent load-bearing evidence — Material Design 3's own guidance above is the
  sourced number this study actually treats as real.
- **Nielsen Norman Group — information scent**: the deeper point behind the owner's own framing
  ("belum tentu semua adalah flow user") — NN/G's guidance is that a navigation label must set
  clear expectations for what a user finds next, and that progressive disclosure (deferring
  advanced/secondary paths) measurably speeds initial task completion (30–50% faster in the cited
  2006 study) while preserving full discoverability
  ([nngroup.com/articles/progressive-disclosure](https://www.nngroup.com/articles/progressive-disclosure/)).
  This reframes Tier 4 correctly: the problem this capability solves is not fundamentally "too
  many pixels," it's that a flat, unfiltered Machine list has **weak information scent** — a link
  to `Approval Step` promises a destination a Submitter never actually wants, diluting the strip's
  overall scent even when it technically fits on screen.

## Possible future needs beyond Tier 4 — surveyed, not admitted

Tier 4 (show/hide exception flag, admin-declared, workspace-wide) is the one need with real case
pressure today. This benchmark's own reading across six platforms surfaces several *adjacent*
needs no case has asked for yet — named here per "silence is not a decision," each with an honest
read against the admission criteria (`capability-lifecycle.md` §2), not built or scheduled:

| Possible need | Real-world precedent | A1 (case evidence today) | Preliminary read |
|---|---|---|---|
| **Nav grouping/sections** (categorize Machines into named groups rather than one flat strip) | Frappe Workspace sections (PUBLIC/MY WORKSPACES), Notion sidebar sections (Favorites/Workspace/Shared/Private) | ❌ none — no portfolio Application has exceeded ~4 visible Machines even before Tier 4 hides anything | Would only clear A1 once an Application's Tier-4-trimmed Machine count still exceeds Material Design's own 7-destination threshold above; premature now |
| **Per-user personalization (favorites/pinning)** | Notion Favorites, Frappe private Workspaces, Salesforce "Personalized Navigation" | ❌ none | Different axis entirely from Tier 4 — per-viewer state, not a Runtime Metadata concern (metadata is shared business truth; a favorites list is per-account UI state, architecturally closer to CAP-O05's own per-user notification preferences than to anything in the Grammar). Would need its own admission pass if a case ever asks, not an extension of Tier 4 |
| **Menu ordering / icon / label override** | Salesforce's own "ordered set of navigation tabs," Jira's "reorder tabs" | ❌ none (already named on `CAP-O03`'s own row, 2026-07-12, and deliberately deferred per "Infer Before Configure" — unchanged by this benchmark) | Stays out of scope; today's alphabetical default is still nobody's named problem |
| **Hidden-but-still-searchable** (Salesforce's "Tab Hidden" keeps App Launcher search working) | Salesforce Tab Hidden | N/A — not a separate capability, a design constraint ON Tier 4's own eventual implementation | When Tier 4 is actually built: a Machine hidden from `subNavFor`/`AppMachines` should very likely remain findable via `CAP-O04` (workspace search) — the two mechanisms already share no code path today, so this needs zero extra work, only a conscious choice not to accidentally couple them later |
| **Navigation as a first-class declared entity** (menu links as real, ordered, nestable metadata that can reference a Machine, a specific View, an external URL, or nothing at all) | Drupal `Menu`/`MenuLinkContent` (see above) | ❌ none — no case has ever asked for a menu entry that isn't a Machine, cross-Application grouping, or breadcrumbs/quick-actions as declared metadata, despite both `004-runtime-metadata.md` and `006-runtime-model.md` naming `Navigation` as a first-class hierarchy peer of Page/View/Service since this repo's own foundational design | The architecturally correct long-term shape (see "Reading against Menata Runtime's own foundational model" above) — would subsume Tier 4 entirely, not just satisfy it, the same way Drupal/Salesforce's own allow-list needs no separate "hide" flag. Not admitted: A1 fails on its own terms while Option A (Tier 4) already covers every real need observed |

**Registry impact**: no new row admitted by this benchmark pass — `CAP-O03` Tier 4 (already ❌
Proposed) is the only need with real case pressure; the other four rows above are recorded as
surveyed-but-not-admitted, the same posture `Choice Card` already holds elsewhere in this registry
(`roadmap.md` item 25.6) — revisit only if a real case demonstrates one, not on schedule. The
fourth row (Navigation as a declared entity) is the one worth remembering specifically: not a
speculative "might need it someday" (§6 would reject that outright) but a documented gap between
what this runtime's own foundational model already promised and what three tiers of real building
have actually shipped — worth a deliberate look the day a case finally does ask for it, rather
than defaulting straight to "add another exception flag."

---

# Follow-on finding (2026-09-08) — `CAP-O03` Tier 5 implementation plan

Owner request, immediately following the trade-off above: *"buat plan untuk opsi B, sebagaimana
sudah ada di konsep menata runtime"* — the day did ask for it, sooner than "revisit... the day a
case finally does" above anticipated. `case-portfolio.md`'s new Case 10 extension note (2026-09-08)
and `capability-registry.md`'s new `CAP-O03` Tier 5 row (v0.67) record the admission itself; this
section is the layer-by-layer plan (`capability-lifecycle.md` §3 "Definition of done") — design
only, nothing built yet.

## Shape, stated once before the layers

An Application MAY declare `navigation_entries`. **If it does, that declaration entirely replaces
`CAP-O03`'s own inferred listing for that Application** — no merge, no partial override, one clear
rule. **If it does not, today's exact inferred behavior (Tiers 2–4's own lineage) is unchanged** —
every existing seed, every existing conformance test, every Application in this portfolio keeps
working with zero migration. This is `001-design-principles.md` §6 "Infer Before Configure" applied
at the Application level, not just the Machine level Tier 4 would have used: the default stays free
inference; declaring navigation is itself the exception, and once declared, it is authoritative.

## 1. Language

No `.menata` grammar change — Navigation, like `master_data`/`immutable_field` before it, is a
Runtime Metadata-layer translation of implicit business intent ("this app's menu should read: X,
then a Reports group with Y and Z"), not a Business Knowledge concept the Language spec itself
needs new vocabulary for. Matches how `writing-runtime-metadata.md` already treats Machine `config`
keys.

## 2. Metadata schema

New table, sibling to `views` in shape (same `ON CONFLICT (id) DO NOTHING` seed-safety convention,
same flat structure — no JSONB `config` blob needed here, every field is a real column since none
of them vary by "type" the way a View's `config` does):

```sql
CREATE TABLE navigation_entries (
    id              TEXT PRIMARY KEY,          -- nav_<app-infix>_<slug>, same ID convention as fld_/evt_/vw_
    application_id  TEXT NOT NULL REFERENCES applications(id),
    parent_id       TEXT REFERENCES navigation_entries(id),  -- NULL = top-level
    position        INT NOT NULL,               -- ordering among siblings, same role as fields.position
    label           TEXT NOT NULL,               -- display text; NOT inferred from target.Name --
                                                  -- explicit, since a group header has no target to infer from
    target_type     TEXT NOT NULL,               -- 'machine' | 'view' | 'url' | 'group'
    target_machine  TEXT REFERENCES machines(id),
    target_view     TEXT REFERENCES views(id),
    target_url      TEXT,                        -- Phase 2, see below -- column reserved now so
                                                   -- Phase 2 needs no migration, only a loader change
    UNIQUE (application_id, id)
);
```

`id`'s prefix follows `writing-runtime-metadata.md`'s existing convention table exactly — add one
row: `Navigation Entry | nav_ | nav_ad_document`.

## 3. Loader (`internal/metadata`)

Same "Unknown = explicit" discipline `validateOperators`/`validateReferences` already apply
elsewhere (CAP-X05): at load time, not render time —

- `target_type = 'machine'` requires `target_machine` to resolve to a real Machine in the SAME
  Application (cross-Application entries are Phase 2, see below) — dangling reference fails boot,
  same severity as a dangling `target_machine` on a `reference` Field today.
- `target_type = 'view'` requires `target_view` to resolve to a real View on a Machine in the same
  Application.
- `target_type = 'group'` requires `target_machine`/`target_view`/`target_url` to all be NULL — a
  pure heading has no destination, catching an author's copy-paste mistake at load time rather than
  a silently-broken link at render time.
- `parent_id`, if set, must resolve to another `navigation_entries` row in the SAME Application —
  and a cycle check (walk `parent_id` to root, reject if the walk revisits a node) at load time, the
  same "reject at boot, not at request time" posture CAP-V20's `EmbeddableChildViewTypes` check
  already established for a different kind of cross-reference.

## 4. Application Model (`internal/model`, `internal/interpreter`)

`NavigationEntry{ID, ApplicationID, ParentID, Position, Label, TargetType, TargetMachine,
TargetView}` — a plain struct, no `config` JSONB unmarshal needed (see §2). `Interpreter` gains
`NavigationFor(applicationID string) []NavigationEntry` (empty slice, not nil, when none declared —
the render layer's own signal to fall back to inference) alongside the existing
`MachinesForApplication`, not replacing it — Tiers 2–4's inference path stays exactly as-is and
keeps calling `MachinesForApplication` when `NavigationFor` returns empty.

## 5. Engine

None needed — this is a pure read/render concern, no Event/Action/Constraint touches it, matching
Tiers 2–4's own posture (CAP-O03's entire lineage has never needed executor changes).

## 6. UI / dispatch (`internal/handler`)

`subNavFor`/`AppMachines` (`app/internal/handler/handler.go`) both gain one branch at their very
top: `if entries := h.interp.Get().NavigationFor(applicationID); len(entries) > 0 { return
renderDeclaredNav(entries, role) }` — falls through to today's existing inference code unchanged
when empty. `renderDeclaredNav` walks the tree (`parent_id`), permission-trims each entry exactly
the way `AppMachines` already trims Machines today (`target_type='machine'`/`'view'` entries need
`Guard.CanRead` on the resolved Machine; `target_type='group'` entries render unconditionally,
disappearing only if ALL of their own children get trimmed away — an empty group heading is noise,
same reasoning `subNavFor` already uses to return `nil` below 2 machines). Rendering itself reuses
`subNavBar`'s existing nested-link markup (`internal/ui/layout.templ`) — a tree of links is a
strict generalization of today's flat list, not a new component.

## 7. Conformance sketch

- An Application with 3 Machines and 2 declared `navigation_entries` (skipping the 3rd): the
  sub-nav strip and app-launcher card list show exactly the declared 2, in declared `position`
  order — the 3rd Machine, while still fully readable/writable directly, is absent from both.
- The SAME Application's Machine still reachable and functional by direct link/reference even
  though absent from nav (mirrors Tier 4's own planned negative case — discoverability only, never
  access).
- A SECOND Application with zero declared entries renders identically to today, byte-for-byte —
  the regression guard proving every existing seeded Application is unaffected.
- A dangling `target_machine`/`target_view` fails `make migrate-up`/reload, not a silent empty
  render (CAP-X05 discipline).
- A `parent_id` cycle fails load with a clear error naming the cycle (mirrors `CAP-V20`'s own
  negative-case discipline for a wrong-type `children` reference).
- A role that cannot read a declared entry's target Machine does not see that entry, group headers
  included when every child under one gets trimmed away.

## 8. Docs

`writing-runtime-metadata.md` gains a "Navigation" section once built, mirroring the existing
"Views" section's own shape (YAML → SQL worked example) — not written now, since documenting a
mechanism as usable before it exists would mislead a reader mid-seed, the same discipline that kept
this plan out of that guide today.

## 9. Registry

`capability-registry.md`'s `CAP-O03` Tier 5 row already carries the admission; flip ❌→✅ with the
real implementation account once built, same pattern every other row in this file follows.

## Phased rollout

**Phase 1 (this plan, recommended first slice):** `target_type` ∈ {`machine`, `view`, `group`},
same-Application only, ordering + nesting + hiding. This alone closes both motivating cases in
full — Case 3's original observed clutter (declare `Approval Document` only) and Case 10's
cross-Machine ordering/grouping need within one Application.

**Phase 2 (named, deliberately deferred, not built now):** `target_type = 'url'` (external links —
column already reserved in §2's schema so this needs no migration, only a loader change) and
cross-Application entries (an entry in one Application's nav pointing into another — Case 10's OWN
full scenario, one employee moving across 4 *different* applications, needs this eventually, but
Phase 1's same-Application scope already delivers the sharpest, cheapest slice of that same need).
Breadcrumbs and quick actions (the other two items on `006-runtime-model.md`'s own "menus,
breadcrumbs, tabs, shortcuts, quick actions" line) are a third, later phase again — no case has
asked for either specifically yet, named per "silence is not a decision" rather than silently
folded into Phase 1's scope.

**Pilot, if/when this is actually built:** `app_approval` is the obvious first real user — one
`navigation_entries` row (`Approval Document` only) closes the exact clutter this whole
conversation started from, on the same live application already used to verify every other piece
of Case 3's own metadata-driven-display story this session.

## Recommendation restated

Do not build `CAP-O03` Tier 4 as a separate prior step. Build this Tier 5's Phase 1 directly — the
cost delta between "a hide flag" and "a full declared allow-list" is smaller than it first looks
(one new table + one loader pass + one dispatch branch, vs. one config key + one dispatch branch),
and Tier 4 would be fully thrown away the moment Tier 5 ships, since an explicit allow-list needs
no separate "hide" mechanism — Drupal and Salesforce's own designs need none either, for the same
reason.
