# Study 36 — One Identity, Many Workspaces: a Login-Time Workspace Picker

> Owner-requested benchmark (2026-09-06, conversation), raised directly against a real gap
> surfaced by Study 35/CAP-O09's own build: the same email can legitimately belong to more than
> one Workspace (very plausible once self-service founding, CAP-O09, lets anyone found a new one
> at will), and rather than blocking that outright, the owner wants a "choose a workspace" screen
> shown after login when it happens. Benchmarked against five real multi-tenant platforms before
> proposing a design, per this repo's own "map before territory" discipline.

---

# 1. The requirement, as given

Restated close to source: if the same email ends up attached to two Workspaces, don't reject it —
after that person logs in, show them a choice of which Workspace to enter. The owner's own framing
was that this is "just a screen at the start" (UI-only). Checking that assumption against the real
code is this study's first job, not an assumption to build on unchecked.

---

# 2. Why this needs more than a screen — checked against the real code and real data

**The schema conflates identity with workspace membership today, so a picker can't be bolted on as
pure UI.** `internal/store/user_store.go`'s `users` table has `UNIQUE(workspace_id, email)` — a row
IS a (workspace, email) pair, not a person. `UserStore.GetByEmail` (`SELECT ... WHERE email = $1
LIMIT 1`, no workspace filter) already carries its own doc comment admitting this: "schema still
permits the same email in two workspaces... this prototype takes the first match by email alone,
an accepted simplification, not enforced against here." CAP-O09's own registry row (added earlier
today, same session) names the exact same ambiguity as a live gap once self-service founding makes
it reachable by real users, not just a seed-data hypothetical.

The reason a picker can't be UI-only: if two `users` rows share an email, they can carry
**different password hashes and different names** — they are two unconnected people-shaped rows
that happen to share a string. Login has to decide *whose* password to check before it can even
render a list to choose from. A real picker needs the two rows unified into one real identity
first — the UI is the easy 10% of this, not the whole of it.

**Checked against the real dev database, not assumed:** `SELECT email, count(*) FROM users GROUP
BY email HAVING count(*) > 1` returns **zero rows** as of this study (2026-09-06) — no email
currently collides across workspaces in `menata_runtime`. This matters directly: a migration
unifying identity by email is lossless and collision-free *right now*, a real, verified fact this
design can rely on, not a hopeful assumption — it should be re-checked again immediately before
whoever implements this actually runs the migration, since new signups land between now and then.

---

# 3. Comparator survey (map evidence, admission test A1)

| Platform | Model | Post-login behavior when one identity has 2+ memberships | Source |
|---|---|---|---|
| **Notion** | One account (one login credential) can be connected to many workspaces; workspaces are a property of the account, not a separate account each | Workspace switcher lists every workspace tied to that one logged-in account; picking one enters it directly, no separate re-auth | [Notion Help — Create, join & leave workspaces](https://www.notion.com/help/create-delete-and-switch-workspaces) |
| **Atlassian** (Jira/Confluence Cloud) | One Atlassian account (one email, one credential) can belong to many "sites" (orgs); sites are unified under `my.atlassian.com` | One login, then a site list/switcher; same account, same session identity, different site context | [Atlassian — Log in to your Atlassian account](https://support.atlassian.com/atlassian-account/docs/log-in-to-your-atlassian-account/); [Understanding the Atlassian Account and User Model](https://www.jirastrategy.com/understanding-the-atlassian-account-and-user-model/) |
| **Basecamp / 37signals ID** | One real global identity ("37signals ID") — the exact phrase used is "a single username and password to get into any of their 37signals product accounts" | A "Launchpad" screen lists every account (company) that identity belongs to; explicitly single sign-on across all of them | [37signals — the 37signals ID rollout](https://signalvnoise.com/posts/2065-product-blog-update-the-37signals-id-rollout); [Basecamp Help — Switching Between Accounts](https://3.basecamp-help.com/article/650-switching-between-basecamp-accounts) |
| **Slack** | The weaker pattern, named explicitly by Slack's own docs: **"You create a separate account for every workspace you join, even if you reuse the same email."** Not a unified identity — juggling multiple workspaces is a client-app (desktop/mobile) session-holding trick, not a server-side membership list | Slack's own sign-in flow can present a workspace list for a matching email, but the underlying accounts stay genuinely separate, unlike the three above | [Slack Help — Sign in to Slack](https://slack.com/help/articles/212681477-Sign-in-to-Slack); [Slack Help — Switch between workspaces](https://slack.com/help/articles/1500002200741-Switch-between-workspaces) |
| **Microsoft 365** | Named here as the **anti-pattern**, not a model to copy: the same email spanning a personal Microsoft account and one-or-more separate work/school (tenant) accounts is a well-documented, recurring source of user confusion in Microsoft's own support forums; Microsoft's own guidance is to *avoid* this overlap (use a distinct alias instead), not to design for it | [Microsoft Q&A — I seem to have two Microsoft 365 accounts with the same login information](https://learn.microsoft.com/en-us/answers/questions/5850755/i-seem-to-have-two-microsoft-365-accounts-with-the) |

**Convergent finding**: every platform that solved this *cleanly* (Notion, Atlassian, Basecamp)
made the same structural choice — **identity is global and singular (one email, one credential,
one login), workspace/org/site membership is a separate many-to-many relation**, and a picker
appears **only when there is genuinely more than one membership to choose from** — a person with
exactly one workspace never sees an extra screen. Slack is the closest real-world example of
*this codebase's own current shape* (a workspace-scoped account, not a real global identity) —
named honestly as the weaker pattern, not a target. Microsoft's overlap confusion is the concrete
argument for building this properly instead of leaving the current ambiguity to fester as
"probably fine."

---

# 4. Admission test (`capability-lifecycle.md` §2)

| # | Criterion | Check |
|---|-----------|-------|
| A1 | Dual evidence | Case: CAP-O09's own build surfaced this as a live gap, same session. Map: the 5-platform survey above. |
| A2 | Universality | Every general-purpose multi-tenant workspace product surveyed has SOME answer to this; the three good ones converge on one shape. |
| A3 | Single responsibility | One cross-cutting concern — identity/session — not a Grammar-area capability; same class as CAP-O01/CAP-X02 already sit in (Workspace Services / Cross-Cutting). |
| A4 | Non-composability | Not buildable by composing existing capabilities — CAP-O01's current model IS the thing that has to change; a picker screen alone, without the schema change, cannot function correctly (§2). |
| A5 | Business language exists | "The same person can belong to more than one workspace and should be able to choose which one to work in" is plain business language, not an engineer-only concern. |

All five hold — admitted.

---

# 5. Proposed design (not built — registration only, per this study's own scope)

## 5.1 Schema: identity separate from membership

- `users` becomes a true identity table: `email` globally unique (not `UNIQUE(workspace_id,
  email)`), one password hash per real person. `workspace_id`/`workspace_role` move OFF this row.
- New `workspace_memberships` table: `(user_id, workspace_id, workspace_role, created_at)`,
  `UNIQUE(user_id, workspace_id)` — exactly the shape CAP-O07's own `group_members` already
  proved out for Groups, applied one level up.
- Migration is lossless today (§2's own verified zero-collision check) — each existing `users` row
  becomes one identity row + one membership row naming its own current workspace/role, 1:1, no
  data loss, *if run before any real collision exists* — re-verify the zero-collision query
  immediately before running it, not just trust this study's own snapshot.

## 5.2 Login flow

1. `Login` authenticates the **identity** by email + password once, exactly as today, just against
   the now-global-unique email.
2. Look up every `workspace_memberships` row for that identity.
3. **Zero memberships**: an orphaned identity (shouldn't normally happen) — clear error, not a
   crash.
4. **Exactly one membership**: skip the picker entirely, redirect straight to `/{that workspace's
   slug}/` — the frictionless common case every good comparator preserves; today's single-workspace
   behavior is unchanged for the overwhelming majority of accounts.
5. **Two or more memberships**: new `GET /choose-workspace` page lists each membership's workspace
   label + slug; picking one redirects to `/{slug}/`. Session carries an intermediate
   "authenticated, workspace not yet chosen" state between steps 1 and 5 (a short-lived flag on
   the session row, cleared once a workspace is chosen). Remembers the last chosen workspace (a
   cookie) as next login's default entry, matching Notion/Atlassian/Basecamp's own "don't ask
   again unless there's a reason to."

## 5.3 Ripple effects on already-built capabilities

- **CAP-O09 (self-service signup)**: must check whether the founder's email already names an
  existing identity — if so, this founding act adds a **new membership** to that identity rather
  than creating a second, disconnected identity row (today's design, per its own registry row,
  would do the latter and silently reintroduce the exact ambiguity this study exists to close).
- **CAP-O10 (email invitation, still ❌ unbuilt)**: gets simpler under this model, not harder — an
  invite either creates a brand-new identity (email never seen before) or adds a membership to an
  existing one; the "invited email already exists in a different workspace" open question Study
  35 §5.3 left unresolved is answered *by this capability*, not worked around separately.
- **Everywhere else** (`h.workspace(r)`, `a.User.WorkspaceRole`, `Guard.CanRead/CanCreate/...`,
  every one of the ~90 already-✅ capabilities): unchanged in shape. The *session-scoped* notion of
  "which workspace is this request in" stays exactly as it is today (still one workspace per
  request, still read the same way) — only the LOGIN path and the `users` schema underneath it
  change. This is the same kind of scope containment CAP-O07's Groups indirection achieved: a real
  new layer underneath, not a rewrite of everything built on top of it.

## 5.4 Named, not solved here

- The exact UX of `/choose-workspace` (styling, whether it's skippable via a "remember this
  device" option) is a real design pass of its own, not sketched in full here.
- Whether an existing identity can be *removed* from a workspace, or a workspace's own Admin can
  see/manage its membership list beyond what CAP-O01's `/admin/users` already does, is out of this
  study's scope — CAP-O01's existing per-workspace user list is unaffected either way.
- This is a genuinely bigger change than "add a UI screen" — it touches CAP-X02's auth core, the
  `users` schema every other table's foreign keys assume, and CAP-O01's own row. Whoever
  implements it should re-read this study in full and re-verify §2's zero-collision check against
  the live database at that time, not assume it still holds from this snapshot.

---

# 6. Summary — registered, not built

**CAP-O11** (multi-workspace identity + login-time workspace picker) registered ❌ Proposed in
`capability-registry.md`'s Workspace Services section, citing this study. No code changed. CAP-O01
gains a status note pointing here (its own model is superseded by this design if/when built, not
today). Real next step, if the owner wants to proceed: a dedicated implementation pass — this
study is the map, not the build.
