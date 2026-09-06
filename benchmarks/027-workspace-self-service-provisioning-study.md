# Study 35 — Workspace Self-Service Provisioning, Invitation, and Slug-Based URLs

> BRD + capability gap analysis for a real requirement raised directly by the owner
> (2026-09-06, conversation with Claude Code, not a written case): today every Workspace and
> every `users` row is created by hand (seed SQL) or by an existing Workspace Admin via
> `/admin/users` — there is no path for someone arriving with no account at all to create their
> **own** workspace, name it, and become its first Admin; no way to invite a new person into a
> workspace by email; and no way to reach a specific workspace by URL (today's URL space carries
> no workspace identifier anywhere — see §2). This owner-initiated framing follows the same
> pattern `benchmarks/026-runtime-graduation-decision.md` (Study 34) and CAP-O07's own 2026-08-23
> note used: a direct request checked against the registry before being registered, not an
> external-yardstick benchmark read cold.

---

# 1. The requirement, as given

Restated from the owner's own request (Bahasa Indonesia, kept close to source so nothing is lost
in translation):

- More than one Workspace can exist in one database (already true — see §2, not a new
  requirement, named here only because the owner's phrasing suggested it might not be checked
  yet).
- A person with no account arrives, registers, and **that act itself creates a new Workspace**
  with them as its first Admin — not an Admin creating an account for someone else, the direction
  every existing mechanism (`/admin/users`, seed SQL) assumes today.
- That same founding flow lets the person choose **two separate strings**: a workspace name that
  becomes the URL segment (e.g. `bumikita`, `perusahaan1`), and a separate display **label** text
  (the human-readable name shown in the UI — these need not be identical, e.g. slug `perusahaan1`
  with label "PT Perusahaan Satu").
- An existing workspace member (implicitly, an Admin) can send an **email invitation** for a new
  user to join their workspace.
- The URL scheme becomes `/{workspace}/...` — concretely `menata.app/bumikita/...` and
  `menata.app/perusahaan1/...` for two different workspaces on the same running server.
- The owner separately asked whether a capability request for URL routing/configuration already
  existed in the registry, suspecting it might. It does not — see §2's own check.

---

# 2. Why none of this is already covered

Checked directly against the current `app/` codebase and the full registry/roadmap set before
writing anything below, per this repo's own "check `README.md`'s Current status before assuming a
gap" discipline (the exact check `benchmarks/024-pdf-signature-approval-study.md` §2 and the
`portal-ga3-code-quality-benchmark.md` correction both had to make the hard way):

| Claim | Checked how | Result |
|---|---|---|
| Multiple workspaces can already exist in one DB | `capability-registry.md` CAP-X06 (✅) | **Already true, not new.** `migrations/008/009` + `workspaceTx`'s RLS isolation is proven with two real workspaces side by side (`ws_default`, `ws_acme`, `seeds/006_second_workspace.sql`), specifically to make cross-workspace isolation provable. Nothing to build here. |
| A capability for URL routing/configuration already exists | `grep`'d `capability-registry.md`/`roadmap.md`/`capability-lifecycle.md`/`case-portfolio.md` for `subdomain`, `workspace.{slug,url,prefix,path}`, `url.{...}workspace`, `invit`, `onboard`, `self-serve`, `provision workspace` | **No hits.** The one adjacent row, CAP-X07 (Auto-generated REST API per machine), is about a JSON CRUD surface per Machine, unrelated to which URL segment names a tenant. CAP-X06 (workspace isolation) is the closest relative and is explicitly **not** this: it resolves workspace_id from the authenticated account, never from the URL (see next row). |
| A user can belong to / reach more than one workspace, or the URL names a workspace | Read `internal/store/user_store.go` (`User.WorkspaceID`, one column, not a join table), `cmd/server/main.go`'s `workspaceTx` (`workspaceID := a.User.WorkspaceID`), `internal/router`/`cmd/server/main.go` route table | **Confirmed false.** `users` is 1:1 with `workspace_id` (schema *permits* the same email in two workspace rows via `UNIQUE(workspace_id, email)`, but `UserStore.GetByEmail`'s own doc comment admits login "takes the first match by email alone, an accepted simplification, not enforced against here" — not a real multi-workspace-per-person mechanism). No route anywhere reads a workspace identifier from the path; `workspaceTx` falls back to the literal string `"ws_default"` for any unauthenticated/static path. |
| Self-service account registration exists at all | `grep`'d `internal/handler`, `internal/ui`, `cmd/server` for `Register`/`SignUp`/`Signup`/`/register`/`/signup` | **Zero hits.** Every `users` row today is created by seed SQL or by an existing Admin's `POST /admin/users` (CAP-O01) — inside a workspace that must already exist. There is no "create my own account" entry point of any kind, let alone one that also creates a workspace. |
| Outbound email infrastructure exists | `capability-registry.md` CAP-O05 | **Does not exist.** CAP-O05's own text: *"no email/SMS infrastructure exists in this prototype"* — the one notification channel is the in-app inbox. An email-invitation flow is the first capability in this runtime's history that needs to send a real outbound email. Named as a real new dependency below, not assumed away. |

Net: none of this is a gap in one existing row — it is three genuinely new capabilities (§4),
none composable from what's already built (admission test A4), each failing a different existing
row's own explicit scope carve-out (CAP-X06 "workspace_id... comes from the authenticated
account, not a client-suppliable cookie" was itself a *fix*, in the opposite direction from what
URL-based routing now asks for — worth re-reading before implementing so the new work doesn't
silently reopen the CSRF/spoofing concern that change closed).

---

# 3. Comparator survey (map evidence, admission test A1)

Every mainstream multi-tenant SaaS workspace product surveyed independently converges on the same
three-part shape the owner described — dual evidence (this direct request as the case/terrain,
this survey as the map) clears admission test A1 without needing a portfolio case:

| Platform | Self-serve creation, creator becomes first admin | Chosen URL slug distinct from display name | Email invitation to join |
|---|---|---|---|
| Slack | Yes — first person to sign up for a new Slack "creates" the workspace | Yes — `{workspace}.slack.com`, slug chosen at creation, rename-able later by an Owner without changing the underlying workspace ID | Yes — by email or shareable link, admin-configurable domains |
| Notion | Yes | Yes — `notion.so/{workspace}`, workspace slug independent of the display name shown in the sidebar | Yes — by email, with a pending-members list |
| Linear | Yes | Yes — `linear.app/{workspace}` | Yes — by email, with roles (Admin/Member/Guest) set at invite time |
| Basecamp | Yes (each "Basecamp" is its own account+company) | Yes — a chosen account identifier in the URL | Yes |
| GitHub (Organizations) | Yes — any user can create an Organization and becomes its Owner | Yes — `github.com/{org}`, distinct from the org's display name | Yes — by email or username, with a pending-invitation state the invitee must accept |

Common shape across all five, which the BRD below adopts rather than inventing a sixth pattern:
**slug and display label are two different fields, chosen once at creation, slug is what routes
URLs; invitation is a token-based accept flow with an explicit pending state, not an immediate
account creation; the inviting side chooses the invitee's role up front.**

---

# 4. Proposed capabilities

Three capabilities, admission-tested individually (A3: each is one responsibility, not one bundle)
— registered in `capability-registry.md`'s **Workspace Services** section (CAP-O09, CAP-O10,
alongside CAP-O01/O07) and **Cross-Cutting** section (CAP-X14, alongside CAP-X06, its nearest
relative):

## CAP-O09 — Self-service workspace provisioning

A person with no account can create a brand-new Workspace by choosing a URL slug and a display
label; that act creates their own account in the same step and makes them that Workspace's first
Admin. Fails to exist today per §2's own check (no registration path of any kind exists yet, self-
service or otherwise) — admission A4 (non-composability) holds because there is no existing
"create an account" mechanism to compose this from, unlike, say, CAP-F17 which really was pure
composition of already-built pieces.

## CAP-O10 — Email invitation into an existing workspace

An existing workspace Admin (or a role the Admin delegates this to — out of scope to design here,
same "delegated Application Admin" gap CAP-O01's own row already names and defers) names an email
address and a role; the runtime sends a real outbound email containing a one-time accept link; the
recipient, whether or not they already hold an account elsewhere, follows it to set a password (new
account) or confirm (existing account with that email) and becomes a member of that one workspace
at the invited role. Distinct capability from CAP-O09 (A3) — an invitation targets an *existing*
workspace, a founding flow creates a *new* one; a platform needs both, not one standing in for the
other (no comparator surveyed collapses the two).

## CAP-X14 — Workspace-scoped URL routing

Every workspace-scoped route gains a `/{slug}/` prefix, resolved to a `workspace_id` by a new
slug→id lookup at the top of the routing chain, replacing `workspaceTx`'s current sole source
(`a.User.WorkspaceID`) with a check that the slug in the URL and the authenticated user's own
workspace agree (403 if not — see §5.4's own worked-through interaction with CAP-X06/CAP-X02).
Cross-cutting, not Workspace Services, by the same logic CAP-X06 itself was filed there: this is a
routing/authz mechanism, not workspace *data*.

None of the three is buildable by composing the other two or any existing ✅ row — recorded here,
not asserted without checking, per capability-lifecycle.md's own A4 text.

---

# 5. BRD — the actual flows

## 5.1 Actors

| Actor | Definition |
|---|---|
| Visitor | No session, no account. Can reach `/login` and the new `/signup`. |
| Founder | A Visitor partway through CAP-O09's flow — chooses slug + label + their own name/email/password in one form. |
| Workspace Admin | `users.workspace_role = 'Admin'` in some workspace — CAP-O01's existing tier, unchanged. |
| Invitee | Someone named on a CAP-O10 invitation, identified by email, not yet necessarily an account holder. |

## 5.2 Flow A — Founding a new workspace (CAP-O09)

1. Visitor reaches `GET /signup` (new, public route, same `isPublicPath` carve-out class as
   `/login` today — `cmd/server/main.go`'s `isPublicPath`/CAP-X02).
2. Form collects, in one submission (no multi-step wizard — nothing here needs one, per this
   project's own "no premature abstraction" convention): founder's Name, Email, Password
   (CAP-X02's existing bcrypt path, unchanged); Workspace Slug; Workspace Label.
3. Slug validation, at submit time (mirrors CAP-X05's own "validation before load, not a runtime
   surprise" discipline): lowercase, `[a-z0-9-]+`, 3–40 chars, not one of a small reserved list
   (`api`, `admin`, `static`, `login`, `signup`, `health`, `ui-sample`, `webhooks` — every literal
   top-level path segment `cmd/server/main.go`'s router already claims, so a workspace can never
   shadow a system route), globally unique across ALL workspaces (not per-anything — the slug
   alone is what the router keys off of). A colliding slug re-renders the form with that one field
   flagged, everything else preserved (same UX discipline CAP-V12's wizard already established
   for a rejected step).
4. On success, in one transaction: insert the new `workspaces` row (new `slug` column, see §5.5),
   insert the founder's `users` row with `workspace_role = 'Admin'`, `workspace_id` = the new row.
   No Application exists yet in a brand-new workspace — CAP-O03's existing "workspace home lists
   Applications" page simply renders empty, already-handled behavior, not a new case.
5. Mint a session exactly as `Login` does today (fresh token, never upgraded from a pre-signup
   one — same fixation defense), redirect to `/{slug}/`.

## 5.3 Flow B — Inviting someone in (CAP-O10)

1. An Admin, inside their own workspace, reaches a new `GET /{slug}/admin/invitations` page
   (alongside CAP-O01's existing `/admin/users` — same Admin-only gate).
2. Form: invitee email, Workspace role (Admin/Member), optionally one or more (Application, role)
   pairs — reusing CAP-O01's existing per-Application role vocabulary/validation verbatim, not a
   second mechanism.
3. New `workspace_invitations` row: `id`, `workspace_id`, `email`, `workspace_role`, a JSON blob of
   any Application-role pairs, a random token (same `auth.NewToken()` CAP-X02 already uses for
   session/CSRF tokens — one proven primitive, not a new one), `expires_at` (a fixed window, e.g.
   7 days — exact value an owner call, not fixed here), `status` (`pending`/`accepted`/`expired`/
   `revoked`).
4. Runtime sends a real email (see §5.6 — the one genuinely new infrastructure dependency this
   whole study surfaces) containing `https://menata.app/{slug}/invite/accept?token=...`.
5. `GET /{slug}/invite/accept?token=...` (public route, token is the auth): looks up the pending
   invitation; if expired/consumed/unknown, a plain error page, no token-guessing oracle (same
   generic-failure hygiene CAP-X02's own login-failure handling already established). If valid:
   - **No account with this email anywhere:** show a form (email pre-filled/read-only, Name +
     Password fields only) → on submit, create the `users` row scoped to `workspace_id` at the
     invited role(s), mark the invitation `accepted`, mint a session, redirect to `/{slug}/`.
   - **An account with this email already exists in this same workspace:** invitation is
     redundant — mark `accepted` without creating a second row, redirect to `/login` with a
     message.
   - **An account with this email exists in a *different* workspace:** per §2's own finding
     (`UserStore.GetByEmail` doesn't disambiguate by workspace), this is the one case this study
     does **not** resolve cleanly and flags rather than papers over: accepting would need either a
     second `users` row with the same email (schema already permits it via
     `UNIQUE(workspace_id, email)`, but login-by-email-alone would then resolve ambiguously,
     exactly the sharp edge §2 found) or a real one-account-many-workspaces membership model
     (a bigger capability than this study's own three, and the one the prior conversation turn
     raised and this study deliberately did not fold in, since the owner's own request this time
     was scoped to founding + inviting, not cross-workspace membership). **Left as a named open
     question for whoever implements this, not decided here** — the two honest options are "block
     with a clear message: use a different email for this workspace" (cheapest, ships day one) or
     "build real multi-workspace membership first" (a fourth capability, its own admission test,
     out of this study's scope).

## 5.4 Flow C — URL routing (CAP-X14)

- Router gains `r.Route("/{slug}", func(r chi.Router) { ... })` wrapping every currently
  workspace-scoped route (`/`, `/{machineID}/...`, `/api/v1/...`, `/apps/...`, `/admin/...`,
  `/search`, `/notifications`) — a mechanical re-nesting, not a rewrite of any handler body.
- New `workspaceBySlug` middleware, ahead of `workspaceTx`: looks up `slug` → `workspace_id` (an
  in-memory map refreshed the same way `interpreter.Store` already refreshes on
  `POST /admin/reload`, CAP-X04 — no new cache-invalidation mechanism). Unknown slug → 404, not a
  fall-through to `ws_default` (today's static-path fallback stays for the truly global routes
  below, unchanged).
- **The one real design decision this flow forces, named explicitly, not silently picked:**
  `workspaceTx` today derives `workspace_id` **only** from the authenticated user's own account
  (`a.User.WorkspaceID`) — a deliberate CAP-X02 fix that closed a real spoofing gap ("workspace_id
  now comes from the authenticated account... not a client-suppliable cookie"). A URL slug is
  exactly the client-suppliable kind of input that fix was written to distrust. Resolution
  proposed here: the slug picks *which* workspace's data a request even attempts to touch (a
  routing concern, same trust tier as which `{machineID}` is in the path), but `workspaceTx`'s
  existing check — does `a.User.WorkspaceID` match this request's resolved workspace? — still runs
  unchanged and still wins: a mismatch is a 403, exactly like today's cross-workspace guard
  (CAP-X06's own T49–T51), not a silent switch to whatever the URL asked for. This preserves
  CAP-X02's fix instead of reopening it, at the cost of a URL that *names* a workspace a logged-in
  user doesn't belong to still 403ing rather than something friendlier — an acceptable, explicit
  trade-off given today's 1:1 user↔workspace model, to revisit only if that model itself changes.
- Truly global routes (`/login`, `/signup`, `/health`, `/static/*`, `/ui-sample/*`) stay unprefixed,
  exactly as `isPublicPath` already carves them out.
- Visitor lands on `/` (no slug) — becomes a redirect to `/login` if unauthenticated (unchanged),
  or to `/{the user's own workspace slug}/` if authenticated — a one-line change once every logged-
  in `User` row carries a resolvable slug.

## 5.5 Schema

- `workspaces` gains a `slug TEXT UNIQUE NOT NULL` column (migration, backfilled for
  `ws_default`/`ws_acme` from their existing `id` values — both already slug-shaped strings, so
  the backfill is free) — kept **separate from `id`**, not a rename of it: `id` stays the stable
  internal FK target every other table's `workspace_id` column already points at (unaffected by a
  future slug rename an Admin might want to make — Slack/Notion both allow renaming the URL slug
  post-creation without this concern, a real feature this separation buys for free, not
  overengineering).
- New `workspace_invitations` table (§5.3's own shape).

## 5.6 Named dependency, not solved here

**Real outbound email.** This runtime has never sent one (CAP-O05's own row is explicit: in-app
inbox only, no email/SMS transport exists). CAP-O10 cannot ship without picking a real mechanism —
an SMTP relay or a transactional-email API (Postmark/SES/etc.) — genuinely a config/infra decision
for whoever implements this to make against the real deployment target at the time, per this
project's own "Infer Before Configure" principle (the same call `app/ROADMAP.md`'s own Phase 3
left open for secrets). Not decided in this study.

## 5.7 Flow D — Bringing your own domain (CAP-X15)

**Addendum (2026-09-06, same session, owner follow-up):** a fourth ask, on top of §5.4's own
`/{slug}/...` scheme — a workspace that owns a real domain should be able to use it directly
(`bumikita.com` resolving straight to that one workspace, root path, no `/bumikita/` segment
visible) instead of, or in addition to, `menata.app/{slug}/`. Checked against every capability
already registered before writing this: no existing row covers it (CAP-X14 above only ever
addresses `menata.app`'s own single hostname; `docs/decisions/003-tenancy-and-indexing.md` and
CAP-X06 are both silent on custom hostnames entirely). Comparator survey extended for this one:
Shopify, Webflow, Squarespace, Notion (paid tier), and Vercel/Netlify (per-project custom domain)
all offer the identical shape — a default `{platform}.com/{slug}`-or-subdomain URL always works,
a verified custom domain is an **additional**, optional way to reach the exact same tenant, never
a replacement that could orphan the default URL.

**Not composable from CAP-X14 alone (admission A4)** — routing by `Host` header instead of a path
segment is a different lookup, and unlike a slug (chosen freely, unique by construction once
validated) an arbitrary external domain needs two things no capability here has needed before:
**ownership verification** (anyone could otherwise claim `google.com` in a form field) and
**per-domain TLS** (menata.app's own certificate, whatever it is, is not valid for a customer's
domain).

**Proposed flow**, following the same non-destructive shape every comparator above uses:

1. Inside `/{slug}/admin/workspace` (a new settings page, Admin-only — today there is no
   workspace-settings page of any kind, only the per-user/per-group ones CAP-O01/CAP-O07 already
   built), an Admin enters a domain (e.g. `bumikita.com`).
2. Runtime generates a random verification token, stores it with the domain on the `workspaces`
   row (`custom_domain`, `domain_verification_token`, `domain_status`:
   `pending`/`verified`/`active`/`failed`), and shows the Admin a DNS TXT record to create:
   `_menata-challenge.bumikita.com TXT "<token>"` — the same domain-ownership-proof pattern every
   comparator surveyed (and Let's Encrypt's own DNS-01 challenge) already uses, not a new one
   invented here.
3. Admin adds the record at their own DNS provider (outside this system entirely — no capability
   needed here), then clicks "Verify" — runtime does a real DNS TXT lookup for that name; a match
   flips `domain_status` to `verified`. No polling/background job needed for this step, a single
   on-click check is enough (same "admin-triggered, not automatic" posture CAP-X04's Option A
   already chose for reload, for the same reason: infrequent, low-volume, a person is already
   there watching it happen).
4. Making a `verified` domain **serve real traffic** needs a real TLS certificate for it — this is
   the one piece that must live in Caddy (`/etc/caddy/Caddyfile`, outside this repo), not in
   `app/`'s own Go code: Caddy's `on_demand_tls` directive can issue a certificate for a
   previously-unknown hostname *at request time*, but only after asking a configured `ask`
   endpoint whether it should. `app/` would expose exactly that: `GET /internal/domain-check?domain=bumikita.com`
   (loopback-only, not internet-reachable — same trust tier as CAP-X04's `/admin/reload`, but
   machine-to-machine instead of admin-to-browser) returning 200 only when `domain_status` is
   `verified` or `active`, 404 otherwise — Caddy issues a cert only for a domain this runtime has
   actually confirmed. First successful request through that domain flips `domain_status` to
   `active`.
5. A request whose `Host` header matches an `active` `custom_domain` resolves straight to that
   workspace's own `workspace_id` — same downstream `workspaceTx` check as the slug path (§5.4),
   just a different lookup key ahead of it (`Host` instead of the `{slug}` path segment). The
   `/{slug}/` URL keeps working unchanged for the same workspace — the domain is additive, per
   every comparator's own "never orphan the default URL" convention (§ above), not a cutover.

**Named dependency, not solved here, distinct from §5.6's email one:** the Caddy-side
`on_demand_tls`/`ask` wiring is real production infrastructure this deployment has never used
before (grepped the live `/etc/caddy/Caddyfile` directly — no `on_demand_tls` directive exists
anywhere in it today). Exactly which Caddy config change that is, and how it interacts with this
host's existing single static `menata.app` server block, is an infra decision for whoever
implements this against the real Caddy version and deployment at the time — not designed further
in this study.

---

# 6. Summary — registered, not built

CAP-O09, CAP-O10, CAP-X14, CAP-X15 registered ❌ Proposed in `capability-registry.md` (Workspace
Services / Cross-Cutting sections respectively), citing this study. No code changed. §5.3's own
cross-workspace-same-email question, §5.6's own email-transport choice, and §5.7's own Caddy
`on_demand_tls` wiring are named open questions for whoever picks this up, not silently resolved
here.
