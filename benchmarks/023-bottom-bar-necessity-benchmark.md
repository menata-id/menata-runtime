# Bottom Bar Necessity Benchmark — does menata-runtime's mobile shell need one at all?

> Study 31 of the Capability Roadmap.
>
> Study 30 (`022-bottom-nav-consistency-benchmark.md`) corrected what the mobile bottom bar
> should *contain*, but never tested whether it should exist at all — its own only stated
> justification was a single, never-examined clause: "just reachable one-thumb on mobile." The
> owner raised that gap directly: if there's no real reason to have the bar, it's fine to remove
> it — every page needs a reason to show what it shows. This study is that test, extended by a
> live benchmark of how the three platforms actually closest to menata-runtime's own domain
> (Odoo, ERPNext, Google Workspace) place their own app-switcher.
>
> Status: v0.1 | Created: 2026-08-23

---

# The question, stated precisely

`internal/ui/layout.templ`'s `shellBottomBar` renders unconditionally on every mobile page,
regardless of Application or Machine — three links, Home/Search/Notifications. Its only
justification, ever written down (`022`, line 69), was that it's "reachable one-thumb." That
claim was never tested against what the rest of the shell already provides, or against how
menata-runtime's actual domain peers (multi-module business/ERP platforms) solve the same
problem. This study tests it.

---

# What the sticky top bar already provides

`navBar` (`layout.templ:202`) is `position: sticky; top: 0` — **always visible, never scrolls
away** — and already carries all three functions the bottom bar duplicates:

| Function | Already in `navBar`? |
|---|---|
| Home | ✅ — brand/wordmark links to `/` (standard web convention) |
| Search | ✅ — `/search` link |
| Notifications | ✅ — `/notifications` link, with the same unread-count badge |

Because the top bar never scrolls out of view, the "can I reach it" argument for a second,
bottom copy of the same three links is already void before any ergonomics question is asked —
both bars are always on screen at the same time.

---

# App-switcher placement — live benchmark

The owner asked specifically how Google Workspace, Odoo, and ERPNext place their own
app-switcher, since none of the three put the app choice directly on the main working page. Verified live (not from training-data memory):

| Platform | Mechanism | Where it lives | Does switching navigate away from the current page? |
|---|---|---|---|
| **Google Workspace** | "Waffle" — 3×3 grid icon | Top-right corner of the existing header | ❌ No — clicking it opens a panel *over* the current page (Gmail/Drive/whatever stays loaded underneath) |
| **ERPNext / Frappe** | App-switcher "cube" icon | Top bar of the existing header | ❌ No — a dropdown, same non-navigating pattern |
| **Odoo** | Grid icon | Top-left corner of the existing header | ✅ Yes — takes you to a separate "main menu" page of app icons |

Two findings, both material:

1. **All three place the switcher as a small icon inside the header they already have** — none
   of them give it a dedicated persistent region of its own, bottom or otherwise. Even Odoo,
   the one that does a full page navigation, triggers that navigation from an icon in the
   existing top-left corner, not from separate chrome.
2. **Two of the three (Google, ERPNext) don't navigate away at all** — the switcher is an
   overlay, so nothing on the current page ever disappears, and the current task's scroll
   position/state is never lost. This is the *less* disruptive pattern of the two, and it costs
   zero permanent screen space — the icon sits inside chrome that already exists.

---

# The domain-fit test — which precedent set actually applies

Re-examining every precedent gathered across this whole investigation (`022` plus this study),
sorted by whether it actually commits persistent bottom-of-screen space to navigation/switching:

| Precedent | Domain | Commits bottom-bar space to switching? |
|---|---|---|
| Odoo | Multi-module ERP suite | ❌ No — top-left icon |
| ERPNext | Multi-module ERP suite | ❌ No — top-bar icon/dropdown |
| Google Workspace | Multi-app productivity suite | ❌ No — top-right icon |
| Salesforce (cross-Lightning-App) | CRM/business suite | ❌ No — App Launcher, not investigated as bottom-bar-resident |
| WeChat / Alipay | Consumer super-app, dozens of switches/session | ✅ Yes |
| Slack | Consumer/team chat, dozens of switches/session | ✅ Yes (own separate case, not app-switching) |

**Every precedent that actually shares menata-runtime's own domain (multi-module business/ERP
platform) puts zero navigation weight on a persistent bottom region.** The only precedents that
do use a persistent bottom bar are high-frequency consumer apps (dozens of destination-switches
per session) — a usage pattern already established as a mismatch for menata-runtime's own
task-focused, business/back-office sessions (open app → complete one task → leave), not
re-argued here.

---

# Cost of keeping it anyway

Unchanged from the earlier finding, restated for completeness:

- `pb-20` — ~80px of permanent bottom padding reserved on **every** mobile page
  (`layout.templ:40`), used or not
- Duplicate chrome: Search and Notifications rendered in two places, two things to maintain and
  test (conformance T188)
- A known, unresolved gap: icon-only tap targets in the mockups sized 30×30px, below the 44px
  minimum (`008-mobile-ui-navigation-standard.md`'s own "Known gap")
- No active-state highlighting at all — the one thing a persistent bar is supposed to help with
  (orientation) isn't actually delivered

---

# Finding

The case for `shellBottomBar`'s existence does not survive this test. Every domain-analogous
precedent (Odoo, ERPNext, Google Workspace) solves the same problem — occasional access to
Home/Search/Notifications-equivalent functions — inside the header they already have, at zero
permanent cost, with the icon-driven-overlay variant (Google, ERPNext) additionally avoiding any
navigation away from the current page at all.

**Recommendation**: retire `shellBottomBar`. Fold "Home" into the existing sticky `navBar` as a
small icon (beside or in place of the wordmark, matching where Google's waffle and ERPNext's
cube both sit) — `navBar` already carries Search and Notifications, so nothing new needs to be
added there beyond the Home icon itself. This reclaims the ~80px `shellBottomBar` currently
reserves on every mobile page, removes the duplicate chrome, and matches the one placement
pattern actually attested across every precedent that shares menata-runtime's own domain.

This is a finding about the runtime's own shell convention — not a UX prescription for any one
hypothetical Application — so it sits inside the boundary already agreed for this kind of work.

---

# What this touches, if acted on (not yet implemented)

- `prototype/go/docs/decisions/008-mobile-ui-navigation-standard.md` — a further dated
  correction, same pattern as the 2026-08-23 correction already there
- `prototype/go/internal/ui/layout.templ` — remove `shellBottomBar`, add a Home icon to `navBar`
- `prototype/go/conformance/tests/090_mobile_nav.sh` — T188 asserts the bottom bar's markup;
  needs rewriting or removal
- The design canvas (`Menata Apps Builder`, correction-section artboards) — the 6 mockups built
  for Study 30's correction assumed a bottom bar; would need a further pass

No code, test, or canvas has been changed by this study — this is the study itself, pending a
decision on whether to act on it.

---

# Resolution (owner-directed, 2026-08-23)

The bottom bar **component stays supported by the runtime** — this study's own benchmark found
real applications (WeChat, Alipay, Slack) that genuinely need one for their own internal
purposes. What doesn't survive is a **fixed, always-on default**: no Application should get a
bottom bar it never asked for, and none should get one shaped by a decision the runtime made on
its behalf.

The resolution is the same principle already established for `subNavBar`/`viewNavPill`'s own
on/off logic, taken one step further: instead of the runtime *deriving* presence from existing
data (sibling-Machine count, View-type count), an Application's own metadata *declares* whether
it wants a bottom bar and what it holds — the runtime renders exactly that when present, and
renders nothing when it isn't declared. This matches this repo's own standing boundary (see the
project's own memory note, "metadata owns app UX decisions"): whether a specific Application
needs this is that Application's author's call, not the runtime's to prescribe.

Registered as `CAP-O08` in `capability-registry.md` — status ❌, not yet built, awaiting a real
case that actually asks for it, per this project's own "declare targets first" discipline (same
treatment `CAP-O07` already got). The fixed, unconditional `shellBottomBar` this study argued
against is retired as a *default*; the underlying rendering mechanism is not thrown away, it
becomes conditional on `CAP-O08`'s metadata instead.

---

# Admission test

| # | Criterion | Result |
|---|-----------|--------|
| A1 | Dual evidence | ✅ — live benchmark of 3 domain-analogous platforms (Google Workspace, Odoo, ERPNext), all converging on the same top-chrome-icon pattern, plus the sticky-top-bar redundancy already established in-repo, plus the owner's own forcing question |
| A2–A5 | No conflict with an existing capability | ✅ — this is a shell-rendering finding, not a change to any registered `CAP-*`; touches only the mobile-nav design record and its conformance test, both already flagged as within scope for this kind of shell-consistency work |

**Verdict: a real finding, not yet acted on.** Recorded here per this repo's own "declare before
building" discipline; implementation is a separate, explicit step.
