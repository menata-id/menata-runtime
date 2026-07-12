# User & Role Management Survey — Person-Reference Fields, Assignee Patterns, Workspace/App-Tier Roles

> Study 18 deliverable (`../roadmap.md`).
>
> Triggered by implementing CAP-F05 (`type: user` field) as real reference-sugar over CAP-O01
> (identity & role registry, ✅ 2026-07-12) — a direct maintainer question ("apa yang sudah
> disebut sebagai best practice worldclass untuk ini, juga untuk aplikasi sejenis") surfaced
> three real design forks worth benchmarking before implementation, not guessing at: how a
> "person" field should store its value, whether "assign to a person" and "assign to a
> role/queue" are the same mechanism or different ones, and whether this runtime's two-tier
> role model (Workspace + per-Application) is missing something every mature platform has.
>
> Status: v0.1 — new | Created: 2026-07-12

**Platforms surveyed:** ServiceNow, Frappe/ERPNext, Salesforce, Camunda (BPMN), Jira, Slack,
Google Workspace/Cloud IAM, GitHub, Notion, AWS IAM/Azure Entra ID.

---

# 1. Person-reference fields and assignment

## Storage: ID, never a display name

Every platform surveyed stores a "person" field as a foreign key to a stable user-record ID:
ServiceNow's `assigned_to` is a reference to `sys_user` (the `sys_id`, not a name); Frappe's
`ToDo.allocated_to` is a `Link` fieldtype to the `User` DocType (the user's immutable `name`
key, not a mutable full-name string); Salesforce `OwnerId` stores an 18-character record ID;
Camunda's `assignee`/`candidateUsers` must match the identity provider's exact user ID.

**Why it matters here:** this runtime's `FieldTypeUser` (`internal/model/model.go`) already
exists as a declared type but has zero enforcement today — it falls through to a plain text
input, and the one mechanism that reads a `user` field's *value* for access control
(CAP-P02's `owner_field` — `internal/permission`'s `Guard.CanTrigger`) compares that stored
value against the acting person's **display Name** as a string. Every platform surveyed
would treat this as a bug: a display name is mutable and not guaranteed unique, exactly the
failure mode an ID comparison exists to prevent.

## Ownership/assignment checks: ID-to-ID, reassignment as a normal write

ServiceNow ACLs and workflow conditions compare `assigned_to == gs.getUserID()` — the
session's own user ID. Camunda's `taskService.setAssignee(taskId, userId)` reassigns by ID,
and formalizes **delegation** as a distinct lifecycle from reassignment (`delegateTask` hands
work to someone else while remembering the original owner as "owner"; the delegate
`resolveTask()`s it back) — a real pattern, but more machinery than this runtime's current
scope needs; noted for later, not built now.

## Person vs. role/queue assignee: kept as separate mechanisms, not one polymorphic field

This is the most consistent finding across every platform surveyed, and it directly explains
a bug this survey caught in this runtime's own seed data: **`mch_complaint`'s
`fld_cmp_assigned_to` (declared `type: user`) is set by `evt_cmp_escalate`'s `set_field`
action to the literal string `"Supervisor"` — a role name, not a person.**

- ServiceNow hard-separates `assigned_to` (→ `sys_user`) from `assignment_group`
  (→ `sys_user_group`) as two different fields with two different reference tables — never
  one field polymorphically holding either kind of value.
- Camunda/BPMN formalizes the same split at the spec level: `assignee` (one person, directly
  bound) vs. `candidateUsers`/`candidateGroups` (an unclaimed pool someone must explicitly
  `claim()`) are different attributes with different runtime semantics.
- Salesforce is the one prominent counterexample — `OwnerId` is genuinely polymorphic (User
  `005…` or Queue `00G…` prefix) — but practitioners treat the resulting prefix-branching as
  complexity to route around, not a pattern to imitate; Salesforce built it that way because
  Queues are themselves modeled as a Group subtype for sharing-model reuse, not because
  polymorphism is best practice.
- Jira and Frappe both keep "assignee" strictly personal and route role-based work through a
  separate mechanism (Jira component-lead default-assignee rules; Frappe's workflow `Allowed`
  role gate on the *transition*, not a value stored in a user-typed field).

**Conclusion applied to this runtime:** a `user` field must never hold a role name as a
fallback value. `fld_cmp_assigned_to` is a category error, fixed as part of this same pass —
see `capability-registry.md`'s CAP-F05 entry for the concrete fix (the field becomes
`value_list` over the queue names it actually holds, since it was never a person reference to
begin with; the existing `notify: {role: ...}` action type already is this runtime's
equivalent of ServiceNow's `assignment_group`/Camunda's `candidateGroups`).

## Picker/lookup scoping: a query-time filter, not a new field construct

ServiceNow's *reference qualifiers* (simple/dynamic/scripted) restrict which `sys_user` rows
a picker offers — e.g. scripted to only users holding a given role. Salesforce *lookup
filters* do the same for a lookup dialog. Frappe resolves role-scoped user lists server-side
(`frappe.get_users_with_role`) to populate approver dropdowns. None of these platforms invent
a new field type per scoping rule — scoping is a filter applied when the picker's option list
is resolved.

**Applied here:** `internal/handler`'s `referenceOptions` already does the reference-field
equivalent (lists a target Machine's records); a new `userOptions` follows the same shape,
scoped via a new `UserStore` method to "users holding any role in this field's own Machine's
Application" — a query-time filter, not a new metadata concept, matching CAP-O01's own
"implicit role vocabulary" discipline.

---

# 2. Workspace-tier vs. Application-tier role management

| Platform | Org/workspace tier | App/project-scoped tier | Assignment axis | Group/Team layer between users and roles? |
|---|---|---|---|---|
| Slack | Owner/Admin | Channel Manager (paid) | Per-user-per-workspace; separate account per standalone workspace | Only at Enterprise Grid scale (IDP-synced Groups) |
| Google Workspace/GCP | Super Admin vs. delegated task-scoped admin | Per-project IAM roles, folder as middle tier | Role bound to a principal at a resource-hierarchy node | Yes — stated best practice: "use Groups, never individual users" |
| GitHub | Org Owner (implicit admin everywhere) vs. Member | Repo roles Read/Triage/Write/Maintain/Admin | Role granted per (principal, repo) | Yes — Teams are the primary grouping primitive |
| Salesforce | Profile (one per user, coarse) | Permission Sets (stackable, finer, assignable per need) | Permission Set attaches to a user directly; Permission Set Groups bundle *sets*, not users | Partial — Public Groups exist only for sharing-rule visibility, a different axis |
| Frappe/ERPNext | System Manager (site-wide) | No true per-app tier — Role is a flat, site-wide label; scoping happens per-**DocType**, one level finer than "app" | Role → User is global across the whole site | Role Profile bundles *roles*, not users |
| Notion | Owner/Member/Guest | Teamspaces (own Owner/Member roles) | Guests per-page; Members per-Teamspace role | Yes — both Teamspaces and Groups |
| AWS/Azure | Management account / tenant Global Admin | Per-account Permission Sets (AWS); per-scope role assignment (Azure) | Permission Set/role bound to (principal, scope) | Yes — stated best practice in both |

## How this runtime's design compares

The reviewed design — Workspace role (Admin/Member) plus a separate role per
`(user, application)` pair, the same person holding a different role in a different
Application simultaneously with no "switch role" step — matches the mainstream pattern
closely. The closest single precedent is **Salesforce's Profile + Permission Set split**: one
coarse mandatory label, one stackable finer-grained assignment. The per-scope binding itself
is unremarkable — GitHub (per-repo role), AWS (per-account permission set), and Azure
(per-scope role assignment) all work the same way. Frappe is *not* a counter-example in this
runtime's favor: its Role is actually **less** granular (a flat, site-wide label with no
per-app dimension at all), so this runtime's per-Application role is already ahead of Frappe's
own model, not behind it.

**What's conspicuously missing**: every platform surveyed interposes a **Group/Team** between
users and role assignment, and for every one operating at real scale it isn't optional — it's
the stated best practice (AWS/GCP: "assign to groups, never individual users"). This runtime
has no such layer; every role is assigned directly to an individual user via `/admin/users`.

**What's a safe, precedented simplification**: one account per workspace (no unified
cross-workspace identity) mirrors Slack's own default — standalone Slack workspaces work
exactly this way until an org adopts Enterprise Grid specifically to solve the pain of
managing many separate per-workspace identities, i.e. it's a known, well-precedented tradeoff
platforms defer until it actually hurts, not a design smell.

## Recommendation (not built in this pass — named, not silently dropped)

- **Groups/Teams as an intermediate role-assignment layer**: real, and cheap to retrofit later
  (an indirection table between users and role assignments), but not worth building at this
  runtime's current scale (a handful of users per Application, admin-driven provisioning, no
  self-service requests anywhere in scope yet). Registered as **CAP-O07** (❌, this survey) —
  see `capability-registry.md`.
- **Delegated Application Admin** (someone who can manage one Application's own user/role
  assignments without full Workspace Admin rights) — the gap every platform surveyed reaches
  for first once a single central admin becomes a bottleneck (Slack Channel Manager, GitHub
  repo Maintain, GCP delegated admin roles, Azure scoped role assignment). Cheaper than
  Groups/Teams to add later since it's a narrower authorization check, not a new data
  relationship; noted on CAP-O01's own registry row as a deferred item, not a separate
  capability.

---

# Gap analysis vs. Menata registry

## New capabilities surfaced

| ID | Capability | Evidence | Why it matters |
|----|-----------|----------|-----------------|
| CAP-O07 | Groups/Teams as an intermediate role-assignment grouping (assign a role to a Group, put users in Groups, rather than only ever assigning roles directly to individuals) | Universal at scale across every platform surveyed (Slack Enterprise Grid, GCP/AWS/Azure stated best practice, GitHub Teams, Notion Teamspaces+Groups) | The one structural element every mature platform's role model has that this runtime's CAP-O01 doesn't; deferred by design at current scale, not overlooked |

## Existing capability rows corrected by this survey

- **CAP-F05** (`user` field): confirmed as reference-sugar over CAP-O01's `users` table,
  storing a user ID (not a name) — implemented in the same pass this survey backs.
- **CAP-P02** (record-level ownership): `owner_field` comparison confirmed to need an ID-to-ID
  check, not the current display-name string match — fixed in the same pass.
- **CAP-O01**: gains a noted-but-deferred "delegated Application Admin" item (see above);
  no change to its ✅ status — the gap is a refinement, not a defect in what shipped.
