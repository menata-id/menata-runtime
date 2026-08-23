# Bottom Navigation Consistency Benchmark — a correction to ADR-008

> Study 30 of the Capability Roadmap.
>
> Prompted directly by the owner, mid-implementation of `CAP-O03 Tier 3` against
> `prototype/go/docs/decisions/008-mobile-ui-navigation-standard.md`: if the bottom tab bar's own
> contents change every time a user moves between Applications (Document Approval, Design
> Request, Project Management, ...), does that feel disorienting given all of them belong to one
> organization's system? And if so, what should the bottom bar actually hold, and how do top bar
> + bottom bar together keep the multi-app system feeling like one system?
>
> Status: v0.2 (revised) | Created: 2026-08-23 | Revised: 2026-08-23

---

# The concern, stated precisely

ADR-008 (Study 29, Phase 1) decided: **bottom tab bar = cross-Machine sideways nav** — the
current Application's own sibling Machines (`CAP-O03 Tier 2`'s existing `subNavFor` data),
rendered as a fixed bottom bar at phone width. Implementation was underway against that decision
when the owner raised the objection directly: an organization plausibly runs several
`Application`s at once (a Document Approval app, a Design Request app, a Project Management
app — real, adjacent cases in `case-portfolio.md`), and the bottom bar's own tab identities would
be completely different in each one — 3 tabs named after Journal Entry/Chart of Account/Fiscal
Period in one app, 3 different tabs named after Board/List/Checklist in another. Does that read
as "one system with several modules," or as a disorienting bar that never means the same thing
twice?

---

# World-class reference — what actually applies to a multi-Application system

A source only counts as evidence here if it describes the same shape of problem Menata actually
has: **one login/session genuinely hosting several distinct applications or modules**, each with
its own unrelated structure — not a single application's own internal sections, and not a single
application with multiple tenants of itself.

| Source | What's actually being compared | Applicable to Menata's case? | Finding |
|---|---|---|---|
| Salesforce | Switching between genuinely different **Lightning Apps** (Sales, Service, Marketing) | ✅ Analogous | Nav changes completely per Lightning App; the App Launcher is only the jump point between them, not evidence that nav stays fixed |
| Google Workspace | Gmail/Drive/Calendar/Docs — genuinely separate installed apps | ✅ Analogous | No shared bottom bar at all; unification happens at the OS home-screen level, outside any single app's own shell |
| WeChat / Alipay | One persistent session hosting many structurally unrelated mini-programs | ✅ Closely analogous — same "one login, many distinct sub-experiences" shape as Menata | The shell keeps a small set of universal functions fixed (Chats/Discover/Me); each mini-program gets its own navigation inside its own screen, never inserted into the shared tab bar |
| Odoo | One login, many modules (Sales, Inventory, Accounting, CRM, HR, Manufacturing) — the same ERP/business-suite shape as Menata | ✅ Closely analogous, same domain as Menata | Only the App switcher, Discuss, notification bell, and avatar stay fixed across every module; the entire top-nav menu changes per module |
| ERPNext (Frappe) | Same shape as Odoo | ✅ Closely analogous, same domain as Menata | Only global search, create, notifications, and avatar stay fixed in the top navbar; the entire workspace sidebar changes per module |

**Across every source that actually matches Menata's shape of problem, the pattern is
consistent: only a small set of genuinely universal functions — search, notifications,
identity/profile, and a jump point to switch modules — stay fixed. Everything module-specific
changes completely, and lives in its own separate, contextual region, never forced into the same
slot as the universal functions.**

---

# Why this matters for a system spanning several Applications

The owner's own framing — Document Approval + Design Request + Project Management are all one
organization's system — matches the shape of Odoo/ERPNext (a multi-module business suite) and
WeChat/Alipay (a super-app hosting distinct sub-experiences) far more closely than it matches a
single chat app's own multi-tenant model. In every closely-analogous system surveyed above, the
resolution is the same: keep a small, genuinely universal set of functions fixed in the outer
shell, and let each module's own navigation vary freely in a separate region — not "make the
whole bottom bar generic because switching otherwise feels disorienting."

The fix is not a new mechanism — every piece already exists in this codebase:

| Layer | Role | Already built? |
|---|---|---|
| **Top bar** (`navBar`) | The one constant across every page, every Application — brand identity ("Menata Runtime"), global Search, Notifications, identity. Never changes. | ✅ already exists, untouched by this correction |
| **Bottom bar** (mobile, this correction) | A small, **fixed**, global set — Home / Search / Notifications — identical on every page, matching the universal-functions-only set that Odoo, ERPNext, WeChat, Alipay, and Salesforce's own App Launcher all keep fixed. Reinforces "one system" the same way the top bar already does, just reachable one-thumb on mobile. | Corrected by this study — was about to be built as a per-Application Machine list instead |
| **"Home" = the app switcher** | Tapping Home lands on the workspace home (`handler.Apps`), which already lists every Application as a card — this **is** this system's own App Launcher / Odoo-style Apps grid, just not named that until now. Switching from Document Approval to Project Management is a deliberate top-level action through Home, never a passive side effect of scrolling. | ✅ already exists (`CAP-O03`), just not previously framed as the answer to "how do I switch apps" |
| **Sub-nav strip** (`subNavBar`, top, contextual) | Once inside one Application, its own sibling Machines (Chart of Account/Journal Entry/Fiscal Period, or whatever that Application declares) — legitimately different per Application, the same way Odoo's own top-nav menu or ERPNext's own sidebar legitimately changes per module. This is the *module-specific* nav region, the one that's allowed to vary. | ✅ already exists (`CAP-O03 Tier 2`) — unaffected by this correction, already horizontally scrollable |
| **View-type pill** (this session's own work) | Within one Machine, switch List/Report/Board/Calendar — the narrowest, most contextual layer of all. | ✅ this session's `CAP-O03 Tier 3` work, unaffected by this correction |

Nothing above requires new metadata or a new concept — it is a **re-assignment of which existing
data goes in which nav region**, not a new feature.

---

# Correction

**ADR-008's Decision #1 is revised**: the bottom tab bar does **not** carry `subNavFor`'s
cross-Machine data. It carries a fixed, global 3-item set — **Home, Search, Notifications** —
identical on every page, mirroring `navBar`'s own existing global links. `subNavFor`'s data stays
exactly where it already was and already worked: the top `subNavBar` strip, unchanged in scope,
now simply also relied on (not hidden) at phone width, since it was already
horizontally-scrollable and never needed the bottom-bar treatment in the first place.

Everything else ADR-008 decided stands: the within-Machine view-type pill (`CAP-O03 Tier 3`
itself), the Detail/Form rules, Dashboard's own exception, and the public-page shell exception are
all untouched by this correction — see `008-mobile-ui-navigation-standard.md`'s own dated
correction section for the precise diff against what shipped in code.

---

# Admission test

| # | Criterion | Result |
|---|-----------|--------|
| A1 | Dual evidence | ✅ — 5 sources whose own shape genuinely matches "one login hosting several distinct applications/modules" (Salesforce cross-Lightning-App, Google Workspace, WeChat, Alipay, Odoo, ERPNext) + the owner's own direct concern as the forcing case |
| A2–A5 | No conflict with an existing capability | ✅ — corrects an in-flight implementation before it shipped; no capability with a different shape is displaced |

**Verdict: ADMITTED as a correction to `CAP-O03 Tier 3`'s implementation, applied before that
implementation was committed.**
