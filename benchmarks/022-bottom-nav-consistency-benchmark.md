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
> Status: v0.1 | Created: 2026-08-23

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

# World-class reference — what a bottom tab bar is actually for

| Source | Finding |
|---|---|
| Material Design (Google) — Bottom navigation | Destinations are **fixed** — they don't scroll, don't reorder, don't change identity. 3–5 top-level destinations of **equal, permanent importance**. [Bottom navigation — Material Design](https://m2.material.io/develop/flutter/components/bottom-navigation/), [The Golden Rules Of Bottom Navigation Design — Smashing Magazine](https://www.smashingmagazine.com/2016/11/the-golden-rules-of-mobile-navigation-design/) |
| Apple Human Interface Guidelines — Tab bars | A tab bar is **persistent** across an app's sections; hidden only under a full-screen modal. **"Frequent changes to tab visibility can make the app feel unstable."** [Tab bars — Apple Developer Documentation](https://developer.apple.com/design/human-interface-guidelines/tab-bars) |
| Slack (mobile) | Bottom bar is **four fixed tabs — Home, DM, @mention, You** — identical regardless of which workspace is open. Switching workspaces is a **separate, deliberate gesture** (swipe/tap to open a workspace list), never a change to what the four tabs mean. [A simpler, more organized Slack on your phone](https://slack.com/blog/productivity/simpler-more-organized-slack-mobile-app), [Switch between workspaces — Slack](https://slack.com/help/articles/1500002200741-Switch-between-workspaces) |
| Salesforce (mobile) | Bottom bar holds **the four most-important quick-access items** — mostly fixed. Its own documentation names one exception: item content can change **"unless users switch to a Lightning app,"** and app-switching itself happens through the **App Launcher** (a distinct grid control), never as a silent side effect of ordinary navigation. Even the exception is gated behind an explicit, top-level switch action. [Customize the Mobile Only Navigation Menu — Salesforce Help](https://help.salesforce.com/s/articleView?id=salesforce_app_customize_nav_menu.htm) |
| Google Workspace (Gmail/Drive/Calendar/Docs) | Not one shell app at all — **each is a genuinely separate installed app**, each with its own fixed bottom bar. There is no single super-app bottom bar that reconfigures per module; "switching apps" is an OS-level action (home screen), not an in-app one. Owner's own instinct, confirmed: this is the cleanest version of "don't reuse one bar for different meanings," achieved by not sharing a bar at all. |

**The pattern is unanimous across every source surveyed, with one partial, gated exception**
(Salesforce) that only reinforces the same rule rather than breaking it: **a bottom tab bar's own
identity should stay fixed.** Content that legitimately varies by context (which app, which
record, which view) belongs in a *different, secondary* navigation region — one users don't hold
to the same "this never changes" expectation a bottom bar earns through consistent repetition.

---

# Why this matters more, not less, for a system spanning several Applications

The owner's own framing — Document Approval + Design Request + Project Management are all one
organization's system — is exactly the case Slack's workspace model and Salesforce's App
Launcher both solve for, and exactly what ADR-008's original bottom-bar decision would have
undermined. A bottom bar whose meaning resets every time a user crosses an Application boundary
doesn't read as "one system, several modules" — it reads as **three different systems that happen
to share a color scheme**, the opposite of the "unifying" outcome the owner is asking for.

The fix is not a new mechanism — every piece already exists in this codebase:

| Layer | Role | Already built? |
|---|---|---|
| **Top bar** (`navBar`) | The one constant across every page, every Application — brand identity ("Menata Runtime"), global Search, Notifications, identity. Never changes. | ✅ already exists, untouched by this correction |
| **Bottom bar** (mobile, this correction) | A small, **fixed**, global set — Home / Search / Notifications — identical on every page, exactly like Slack's four tabs or Salesforce's default set. Reinforces "one system" the same way the top bar already does, just reachable one-thumb on mobile. | Corrected by this study — was about to be built as a per-Application Machine list instead |
| **"Home" = the app switcher** | Tapping Home lands on the workspace home (`handler.Apps`), which already lists every Application as a card — this **is** this system's own App Launcher / workspace switcher, Salesforce's and Slack's own pattern, just not named that until now. Switching from Document Approval to Project Management is a deliberate top-level action through Home, never a passive side effect of scrolling. | ✅ already exists (`CAP-O03`), just not previously framed as the answer to "how do I switch apps" |
| **Sub-nav strip** (`subNavBar`, top, contextual) | Once inside one Application, its own sibling Machines (Chart of Account/Journal Entry/Fiscal Period, or whatever that Application declares) — legitimately different per Application, same way Slack's own channel list or a website's breadcrumb legitimately changes per section. This is the *secondary* nav region users don't expect to stay fixed. | ✅ already exists (`CAP-O03 Tier 2`) — unaffected by this correction, already horizontally scrollable |
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
| A1 | Dual evidence | ✅ — 5 named platforms/guideline sources across mobile OS vendors (Apple, Google), a workspace-collaboration product (Slack), an enterprise CRM (Salesforce), and a genuine multi-app suite (Google Workspace) + the owner's own direct concern as the forcing case |
| A2–A5 | No conflict with an existing capability | ✅ — corrects an in-flight implementation before it shipped; no capability with a different shape is displaced |

**Verdict: ADMITTED as a correction to `CAP-O03 Tier 3`'s implementation, applied before that
implementation was committed.**
