# Runtime Metadata Schema

> Runtime Metadata describes how Business Knowledge should be realized by Menata Runtime.
>
> It is designed for deterministic machine interpretation.
>
> This document defines the Runtime Metadata format used by this prototype.

---

## Format

Runtime Metadata is expressed in YAML.

YAML is used for human readability during the prototype phase.

The format may evolve in future versions.

---

## Hierarchy

```text
Workspace
    └── Application
            └── Machine
                    ├── fields
                    ├── events
                    ├── constraints
                    ├── permissions
                    └── views
```

---

## Workspace

```yaml
workspace:
  id: ws_default
  name: Default Workspace
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | Stable unique identifier |
| name | string | yes | Human-readable workspace name |

---

## Application

```yaml
application:
  id: app_procurement
  name: Procurement
  workspace: ws_default
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | yes | Stable unique identifier |
| name | string | yes | Human-readable application name |
| workspace | string | yes | Reference to workspace id |

---

## Machine

A Machine is the primary realization unit.

It realizes one business capability.

```yaml
machine:
  id: mch_purchase_request
  name: Purchase Request
  application: app_procurement

  fields:
    - id: fld_requester
      name: Requester
      type: user

    - id: fld_amount
      name: Amount
      type: money

    - id: fld_status
      name: Status
      type: value_list
      values:
        - Draft
        - Submitted
        - Approved
        - Rejected

  events:
    - id: evt_submit
      name: Submit
      condition: { field: fld_status, operator: equals, value: Draft }
      actions:
        - set_field: { field: fld_status, value: Submitted }

    - id: evt_approve
      name: Approve
      condition: { field: fld_status, operator: equals, value: Submitted }
      actions:
        - set_field: { field: fld_status, value: Approved }

    - id: evt_reject
      name: Reject
      condition: { field: fld_status, operator: equals, value: Submitted }
      actions:
        - set_field: { field: fld_status, value: Rejected }

  constraints:
    - id: cst_amount_positive
      rule: Amount must be greater than zero.
      expression: { field: fld_amount, operator: greater_than, value: 0 }

  permissions:
    - role: Requester
      events: [ evt_submit ]

    - role: Manager
      events: [ evt_approve, evt_reject ]

  views:
    - id: vw_form
      name: Request Form
      type: form

    - id: vw_my_requests
      name: My Requests
      type: list

    - id: vw_detail
      name: Request Detail
      type: detail
```

---

## Machine Config

A Machine may declare `config` — settings about the Machine itself, not a Field of its records and
not any of the five Grammar sections. Absent/`null` for every Machine that doesn't need one; there
is no fixed schema across Machines, only per-key conventions as capabilities need them (CAP-X03).

```yaml
machine:
  id: mch_approval_document
  name: Approval Document
  application: app_approval
  config:
    approval_mode_field: fld_ad_approval_mode   # which field holds Sequential|Parallel
    steps_machine: mch_approval_step             # the child Machine holding this workflow's steps
    steps_parent_field: fld_as_document           # which field on the child references back
```

This is the only `config` shape defined so far — it exists to let CAP-A07 (`activate_next`) and
CAP-A08 (`aggregate_status`) resolve a workflow's shape without hardcoding Machine/Field ids in the
runtime. A future capability needing its own machine-level setting adds a new key here, not a new
migration column.

---

## Field Types

| Type | Description | Example | Required `options` key |
|------|-------------|---------|-------------------------|
| `text` | Short text | Title, Name | — |
| `rich_text` | Formatted text | Description, Notes | — |
| `number` | Numeric value | Quantity | — |
| `money` | Monetary value | Amount, Price | `currency` or `currency_field` — **mandatory**, see note below |
| `boolean` | True/False | Is Active | — |
| `date` | Calendar date | Due Date, Start Date | — |
| `time` | Time of day | Meeting Time | — |
| `date_time` | Date and time | Submitted At | — |
| `duration` | Time span | Estimated Hours | — |
| `user` | Reference to a User | Requester, Assignee | — (see note below) |
| `file` | File attachment | Document, Photo | — (see note below) |
| `value_list` | Predefined values | Status, Priority, Type | `values` — mandatory array |
| `reference` | Reference to another Machine | Department, Project | `target_machine` — mandatory |
| `child_table` | Line items owned by a parent record (header-detail document) | Journal Entry Lines, Unit Conversions | `target_machine` — mandatory (points at the child Machine's schema) |

**`child_table` (CAP-F16, ❌ not yet implemented)** is not a primitive either — its rows are ordinary
records of a target Machine, scoped to one parent. See `capability-registry.md` (CAP-F16) for the
reporting-independence distinction (some child Machines, e.g. Journal Entry Line, must stay
independently queryable across parents for aggregate reports; others, e.g. Item Unit Conversion,
never are). The `.menata` source for a child Machine looks like any other Object — a Field
referencing back to its parent (e.g. `- Journal Entry : Journal Entry`, see
`prototype/go/docs/examples/accounting-journal-entry-line.menata`); no special Language notation is
involved (corrected 2026-07-11 — an earlier revision believed `Table of (...)` was a provisional
stand-in for missing Language grammar; it wasn't needed). `child_table` is the Runtime Metadata
translator's choice of *storage strategy* for that already-expressible relationship, not a mirror of
anything declared in `.menata` itself.

### `money`, `user`, `file` are "reference sugar"

These three are not independent primitive types conceptually — each is `reference` to a
predetermined target, kept as its own named type today only because the runtime does not yet have
that target to point at:

| Type | Reference target | Target status |
|------|-------------------|-----------------|
| `user` | Platform identity | CAP-O01 (identity & role registry) is now ✅ — `users`/`user_application_roles` — but `type: user` has not yet been migrated to point at it as a `reference` target; still renders as free text (CAP-F05 ⚠️) |
| `money` | Currency (code + exchange rate) | Pending CAP-O02 (master data designation) |
| `file` | Runtime-managed File/Document entity | Not yet implemented — CAP-F06 ⚠️ partial |

**`type: money` MUST include `currency` (fixed code, e.g. `"IDR"`) or `currency_field` (a reference
to another field on the same record) in its `options`.** Metadata declaring `money` without either
is incomplete — the same discipline already required for `value_list` (`values`) and `reference`
(`target_machine`). This is validated at load time by CAP-X05 once implemented.

### `file` — image handling `options`

`file` does not get a separate `image` type. Whether a file is an image is a **processing policy**
on the same reference-sugar `file` type, not a different reference target — the same reasoning that
keeps `rich_text` a variant of text handling rather than a different kind of reference. This is the
metadata-facing fact; only the `options` schema below is something a metadata author writes.

```yaml
- id: fld_ad_photo
  name: Photo Evidence
  type: file
  options:
    accept: image/*        # MIME types accepted; triggers the compression policy below
    compress: true          # apply the compression pipeline
    max_dimension: 1920     # resize policy (longest edge, px)
    format: webp            # target storage format
```

*How* the runtime realizes `compress: true` (client-side vs. server-side, and the enforcement rule
that the server never trusts client-side compression alone) is a runtime-behavior concern, not a
metadata-schema one — it is documented once, authoritatively, in `capability-registry.md` (CAP-F06)
and `nfr-standards.md` §2.1, not repeated here.

Full reasoning, the decision tree for choosing between `value_list` / `reference` / a primitive, and
worked examples: `runtime/benchmarks/005-field-modeling-decision-framework.md`.

---

## Event Conditions

An Event may declare `condition` — a guard, same shape as a Constraint's `condition`
(field/operator/value). The event may only be triggered when the record's **current** data (before
the event's own actions run) satisfies it; otherwise the runtime rejects the trigger outright
(CAP-E06). This realizes the `if` guard Menata Language's Event grammar already allows
(`specification/003-event.md` §Conditions in the `menata` repo) — nothing new for a `.menata` author
to learn, the runtime simply didn't evaluate it on `When`-triggered events until CAP-E06 landed.

```yaml
- id: evt_approve
  name: Approve
  condition: { field: fld_status, operator: equals, value: Submitted }
  actions:
    - set_field: { field: fld_status, value: Approved }
```

Without a declared `condition`, an event may be triggered from any state — declare one whenever the
business rule "this only makes sense from state X" is real, the same judgment call already made for
every Constraint.

A second, narrower guard exists purpose-built for sequential workflows (CAP-A07) — see
`aggregate_status`/`activate_next` below; it is cross-record (checks sibling records) and can't be
expressed as a flat `condition`, so it isn't a metadata key at all, just built-in behavior triggered
by declaring `aggregate_status` on an event.

---

## Event Actions

Actions describe what the runtime should do when an event occurs.

| Action | Description | Example |
|--------|-------------|---------|
| `set_field` | Set a field to a value — literal, or a dynamic token (`today`, `now`, `current_user`, CAP-A02) | Set Status = Submitted; Set Approved Date = today |
| `notify` | Send a notification to a role | Notify Manager |
| `create_record` | Create a record in another machine | Create Audit Log |
| `activate_next` | CAP-A07 — in a Sequential-mode workflow, notify the next still-undecided sibling once this one is decided | Approving Step 1 notifies Step 2's approver |
| `aggregate_status` | CAP-A08 — roll a decided record's siblings up to their shared parent: cascade the parent to a "some rejected" event immediately, or to an "all approved" event only once every sibling has decided the same way | All Steps Approved → Document Approved |

Actions are realized by the runtime.

Business Knowledge should not describe how actions are implemented.

### `set_field` dynamic values (CAP-A02)

`value: today` / `value: now` / `value: current_user` resolve at the moment the event fires, instead
of being stored as that literal string. `current_user` resolves to the acting person's real
identity (CAP-X02 — a real authenticated account's name, not a role string) — see the
`user`/reference-sugar note above.

```yaml
actions:
  - set_field: { field: fld_status, value: Approved }
  - set_field: { field: fld_approved_date, value: today }
  - set_field: { field: fld_approved_by, value: current_user }
```

### `activate_next` (CAP-A07)

```yaml
- activate_next: { mode_field: fld_ad_approval_mode }
```

`mode_field` is a Field id on the **parent** Machine (resolved via the child's `reference` field to
it, and the parent's own `config`) holding the Sequential/Parallel choice. In Parallel mode this
action is a no-op. In Sequential mode, **enforcement is a hard block**, not just this notification —
see the note below. Parent/sequence/decision fields are not named in this action's own params; the
runtime resolves them by `reference`-field-type and by-name heuristics (a Field literally named
`Sequence`, `Decision`, `Approver`) — a prototype-honest stand-in for a Language-level way to name
these, not a final design. See `capability-registry.md` (CAP-A07) for the full rationale.

**The actual sequential gate** is not a metadata key — declaring `aggregate_status` (below) on an
event is what makes the runtime cross-record-check siblings before allowing that event to fire at
all, rejecting an out-of-sequence Approve/Reject with an explicit error. This was a deliberate
design choice: WCP-1 Sequence (the Workflow Pattern this capability realizes) is *defined* by
enforcement — a notify-only "activation" would leave Sequential and Parallel mode behaviorally
identical.

### `aggregate_status` (CAP-A08)

```yaml
- aggregate_status:
    parent_field: fld_as_document
    parent_event_if_all_approved: evt_ad_approve
    parent_event_if_any_rejected: evt_ad_reject
```

`parent_field` names the Field (on this Machine) referencing the parent record. After this event
commits, the runtime checks every sibling record sharing that same parent: if *any* has reached a
"Rejected"-shaped decision, `parent_event_if_any_rejected` fires on the parent **immediately** — it
does not wait for the remaining siblings to decide (the discriminator/cancellation half of WCP-9).
`parent_event_if_all_approved` only fires once *every* sibling has reached an "Approved"-shaped
decision. The parent event is fired through the exact same trigger path an HTTP-originated event
uses — its own `condition` (CAP-E06) and Constraints (CAP-C09) still apply, so a system-triggered
rollup can never skip a check a user-triggered transition would have to pass. The acting role
recorded for this internally-fired event is `System`.

---

## Constraints

Constraints describe business rules that must always be satisfied. Every declared Constraint is
checked both at Create and, since CAP-C09, whenever any Event fires — the same rule, re-evaluated
against the record's data *after* that event's actions would apply, before the write commits. A
record valid at Create can become invalid by the time an event reaches it (e.g. a date-based
constraint, once real time has passed); this closes that gap instead of only checking once.

```yaml
constraints:
  - id: cst_title_required
    rule: Title is required.
    expression:
      field: fld_title
      operator: required

  - id: cst_due_date_future
    rule: Due Date must be after today.
    expression:
      field: fld_due_date
      operator: after
      value: today

  - id: cst_attachment_required_for_banner
    rule: Attachment is required for Banner design type.
    expression:
      field: fld_attachment
      operator: required
    condition:
      field: fld_design_type
      operator: equals
      value: Banner
```

---

## Permissions

Permissions assign events to business roles, plus two independent gates
(CAP-P02, CAP-P05 — implemented 2026-07-12) that don't map to a business role
alone:

```yaml
permissions:
  - role: Employee
    events: [ evt_submit ]

  - role: Manager
    events: [ evt_approve, evt_reject ]

  - role: HR
    events: [ evt_record_leave ]

  - role: Approver               # CAP-P02 — record-level ownership
    events: [ evt_as_approve, evt_as_reject ]
    owner_field: fld_as_approver # the acting identity (not just role) must
                                  # equal this record's own field value

  - role: Submitter               # CAP-P05 — CRUD-level, independent of events
    events: []
    can_read: true
    can_create: true
    can_edit: false               # each defaults to true if omitted
```

- **`owner_field`** (optional, a Field id on the same Machine): when set, the
  listed `events` require the acting identity to equal that field's value on
  the record, not just the role — e.g. only the specific person named as an
  Approval Step's own Approver may decide it, not anyone holding the
  "Approver" role. Omit for role-only gating (the default, and still what
  most Permission rows want).
- **`can_read` / `can_create` / `can_edit`** (optional booleans, default
  `true`): read/create/edit access to a Machine's records, independent of
  which Events a role may trigger. **Deny-by-default at the Machine level**:
  a role with no Permission row at all on a Machine has none of these —
  reads/creates/edits are denied, not implicitly allowed. A role that only
  needs to trigger Events still needs at least one Permission row present to
  read/create/edit at all (the defaults only apply once a row already
  exists for that role).

**Who actually holds a `role` string is CAP-O01, not this file.** `role` here only *declares
the vocabulary* — which role names exist and what each may do on this Machine. A real person
is assigned one of these roles for the whole Application (every Machine within it shares the
same role vocabulary, since Permissions across an Application's Machines are what defines it)
via `user_application_roles`, one row per `(user, application)` pair, set through
`/admin/users` — not part of Runtime Metadata, and not a login-time free choice either: role
is no longer self-declared (CAP-X02), it's assigned by a workspace Admin ahead of time. The
same person can hold a different role in a different Application at the same time (e.g.
"Requester" here, "Approver" over there) with no "switch role" step — their role for a given
page resolves from which Application that page belongs to.

---

## Views

Views describe how Business Knowledge is presented.

```yaml
views:
  - id: vw_form
    name: Request Form
    type: form
    fields: [ fld_requester, fld_amount, fld_description ]

  - id: vw_list
    name: All Requests
    type: list
    columns: [ fld_requester, fld_amount, fld_status ]
    default_sort:
      field: created_at
      direction: desc

  - id: vw_detail
    name: Request Detail
    type: detail
```

### View Types

| Type | Description |
|------|-------------|
| `form` | Input surface for creating or updating a record |
| `list` | Table or card presentation of multiple records |
| `detail` | Full presentation of a single record |
| `dashboard` | Summary and metrics presentation |
| `calendar` | Date-based presentation |
| `timeline` | Chronological presentation |
| `aggregate_report` | Group-by / rollup presentation over many records (Trial Balance, Leaderboard) — requires `group_by` + `aggregates`; CAP-V13, ❌ not yet implemented |

---

## Load-Time Contract — What's Enforced, What Silently No-ops

Found the hard way (2026-07-12): converting 50 previously-untested example `.yaml` files
into loadable metadata surfaced several failure modes that aren't obvious from reading the
grammar alone. Every one of these applies whether the metadata was written by a person or
an AI — this section exists specifically so the next author (of either kind) doesn't have
to rediscover them by trial and error.

**One bad Machine fails the entire server, not just that Machine.** `Loader.LoadAll` loads
every Workspace's full tree in one pass; a single invalid field/event/constraint/permission
anywhere aborts the whole boot (`os.Exit(1)` in `cmd/server/main.go`) — there is no partial
load, no per-Machine quarantine. A metadata error in one Application you're not even working
on can take the entire runtime down for every other workspace too.

**A `value` in `constraint.expression`, `constraint.condition`, or `event.condition` MUST be
a string, even when it reads like a number.** `value: 100` (a YAML/JSON number) crashes the
loader (`cannot unmarshal number into Go struct field ConstraintExpression.value of type
string`) — write `value: "100"`. This is one of the load-time-fatal mistakes above, not a
silent no-op.

**Only four constraint/condition operators are implemented**: `required`, `equals`,
`not_equals`, `after` (and `after` only against the literal value `"today"`). `before`,
`greater_than`, `less_than`, `greater_than_or_equal`, `unique`, and any compound/aggregate
shape (`aggregate: sum`, a plural `conditions:` list instead of singular `condition:`) are
**not errors — they silently never fire** (`constraint.Eval`'s default case returns `true`,
meaning "satisfied," for any operator it doesn't recognize). A constraint or event guard
written with one of these looks correctly declared, loads without complaint, and then simply
never does anything. If you need one of these, it isn't supported yet — don't write it as if
it were; name the gap instead (see `capability-registry.md`'s CAP-C10/CAP-A09/CAP-C12 rows).

**`set_field.value` supports exactly a literal string, or one of three dynamic tokens**:
`today`, `now`, `current_user`. Anything else — a function call (`raise_one_level(priority)`,
`sla_offset(priority)`), field arithmetic (`reopen_count + 1`), template interpolation
(`{{ this.field }}`), a `previous(field)` read, a `role:X` dynamic target — is **not
evaluated at all**. It gets written to the record as that literal text, verbatim, silently
wrong data, not an error. None of these expression forms exist in the runtime today.

**`create_record` is declared as an action type but has no implementation** — `Executor.
Persist` logs it and does nothing else (CAP-A06, ❌). Metadata naming it loads and runs
without error; it just never creates the record it names.

**A `reference` field's `target_machine` must be a real Machine id already present in the
same load** — including reserved/pseudo targets like `"$identity"` (CAP-F13's still-
unimplemented flavor (b), the built-in identity target) count as dangling and fail the load,
same blast radius as above. If you mean "the currently acting person," that's `type: user`
(CAP-F05), not a `reference` with a made-up target.

**A `permissions.owner_field` must name a Field declared `type: user`** on the same Machine
(CAP-P02/CAP-F05, enforced at load time since 2026-07-12) — pointing it at a `text` field or
any other type fails the load. Omit `owner_field` entirely for role-only gating if no
Field on the Machine genuinely represents "the specific person who must act."

**Unrecognized YAML/JSON keys are silently dropped, not rejected.** Go's default JSON
decoding ignores fields a struct doesn't declare — a `views.filter` block (CAP-V09, not
implemented) doesn't error, it just vanishes with no trace. The absence of a load error is
not confirmation that everything you wrote was understood; cross-check against what this
document and `capability-registry.md` actually say is implemented, not just against "did it
load."

**When a case's Machines span multiple files sharing one Workspace/Application** (a common
pattern once an Application has multiple Machines, one file per Machine), exactly one of
those files needs to declare `workspace:`/`application:` as full objects
(`{id, name, workspace: ws_id}`); the others may reference the Application as a bare id
string (`application: app_foo`) instead of repeating the full declaration. Nothing enforces
that *some* file in the group declares it fully — if none do, the Application is simply
never created and every Machine referencing it by bare string dangles.

---

## Stable Identity

Every metadata element has a stable `id`.

The `id` should never change after it is assigned.

Names, labels, and presentation may change freely.

The runtime uses `id` for all internal references.

---

## Versioning

Runtime Metadata should declare its schema version.

```yaml
version: "0.1"
```

This allows the runtime to apply appropriate interpretation rules per version.
