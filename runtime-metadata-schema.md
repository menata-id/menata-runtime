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

This was the only `config` shape for a while — CAP-A07 (`activate_next`) / CAP-A08
(`aggregate_status`) resolving a workflow's shape without hardcoding Machine/Field ids. Every
new machine-level setting since keeps following the same rule: a new key here, never a new
migration column. As of 2026-07-12:

| Key | Used by | Meaning |
|-----|---------|---------|
| `approval_mode_field`, `steps_machine`, `steps_parent_field` | CAP-A07, CAP-A08 | workflow shape, see above |
| `webhook_secret` | CAP-E04 | the credential an inbound `POST /webhooks/...` must present in `X-Webhook-Secret` for this Machine |
| `master_data` (`"true"`) | CAP-O02 | flags this Machine as canonical/cross-app-referenced — blocks Archive while any other record, on any Machine, still references it |
| `immutable_field`, `immutable_values` (comma-separated) | CAP-R07 | once this Field's value is one of `immutable_values` (e.g. a Journal Entry's Status = "Posted"), the record rejects both Update and Archive |
| `scratch_field`, `scratch_values` (comma-separated) | CAP-R08 | a record created with this Field already set to one of `scratch_values` (e.g. a Cart's Status = "Building") skips normally-blocking Constraints until it transitions out — an in-progress/draft state exempt from validation |
| `sod_reference_field`, `sod_requester_field` | CAP-P03 | Segregation of Duties, declared on the CHILD/deciding Machine (e.g. Approval Step): `sod_reference_field` names this Machine's own `reference` field pointing at its parent (e.g. Approval Document); `sod_requester_field` names the `user`-typed field on that PARENT Machine holding who submitted it. The acting identity may not equal that value — the submitter of a record can't also decide it, even when they otherwise hold the deciding role |

```yaml
machine:
  id: mch_int_payment
  config:
    webhook_secret: "demo-webhook-secret-2026"

machine:
  id: mch_wsx_employee
  config:
    master_data: "true"
```

---

## Process Overlay (CAP-W01/W03/W04/W05, Process Overlay B1–B4, 2026-08-22)

A Machine may declare `process` instead of hand-authoring its own `events`/`permissions`/Status
Field — a compact state-machine declaration the loader **compiles** at boot into exactly the
same Events/Permissions/Fields a hand-authored Machine would have
(`internal/metadata/compile.go`'s `compileProcess`; the concept: `brd-menata-runtime-v2.md`,
proof: `benchmarks/013-overlay-compiler-proof.md`). Downstream — Router, Guard, Executor,
Constraint Engine — never sees `process` at all, only its compiled result: "declared process,
emergent execution."

```yaml
machine:
  id: mch_corrective_action
  fields:
    - { id: fld_ca_notes, name: Notes, type: rich_text }
  process:
    states: [Open, Assigned, In_Progress, Submitted, Review, Escalated, Verified, Closed]
    transitions:
      - name: Assign
        from: Open
        to: Assigned
        actor: { role: Supervisor }
      - name: Start
        from: Assigned
        to: In_Progress
        actor: { role: Worker, owner_field: fld_ca_assignee }   # CAP-P02 — narrows the role to the specific assignee
      - name: Submit
        from: In_Progress
        to: Submitted
        actor: { role: Worker, owner_field: fld_ca_assignee }
        on_transition:                                          # optional -- extra actions, same {type, params} shape event_actions rows already use
          - { type: notify, params: { role: Reviewer } }
        requirements:                                            # optional -- CAP-W01, see below
          - { type: evidence, target: mch_ca_photo, cardinality: "2..*" }
      - name: Approve
        from: Review
        to: Verified
        actor: { role: Reviewer }
      - name: Close
        from: Verified
        to: Closed
        actor: { role: Supervisor }
    auto:                                                        # optional -- system-performed, chained via CAP-E05 the instant a record lands on `from`
      - { from: Submitted, to: Review }
    sla:                                                          # optional -- CAP-W04, see below
      - state: Review
        duration: "2 Business Days"
        on_breach: { notify: { role: Manager }, escalate_to: Escalated }
```

**Compiles to:**

| Declared | Compiles to | Mechanism |
|---|---|---|
| `states` | the Status `value_list` Field's `values` — generated (`fld_<machine>_status`) unless the Machine already has its own `value_list` Field literally named `Status`, in which case its own declared `values` must already cover every process state | first-value-is-default convention already governs the initial state |
| `transitions[].{name,from,to}` | one `Event` per transition, id `evt_<machine>_<slug(name)>`, `condition: {status equals from}`, first action `set_field {status: to}` | CAP-E06 |
| `transitions[].actor` | one `Permission` per distinct `{role, owner_field}` pair across all transitions, granting every Event that pair authors | CAP-P01/CAP-P02 |
| `transitions[].on_transition` | appended as further `EventAction`s on the same compiled Event, in declared order | ordinary Executor action vocabulary — no second grammar |
| `transitions[].requirements[].type: evidence` | a generated `number` counter Field (`fld_<machine>_<target>_count`, default `"0"`) + a gating `Constraint` (`condition: {status equals to}`, `expression: {counter >= min}` [+ `<= max`]) | CAP-C09 re-validates it unchanged — see below |
| `transitions[].requirements[].type: approval` | an `aggregate_status` `EventAction` appended to every `target` Event that sets its own `"Decision"` field — nothing generated on the declaring Machine itself | CAP-A08, resolved once every Machine has loaded — see below |
| `auto` | a `System`-actor `Event` of the same shape, chained onto every compiled Event landing on its `from` state via `trigger_event` (no human Permission grants it — deny-by-default keeps it off every button) | CAP-E05 |
| `sla[].{state,duration}` | a generated `date` due-date Field (`fld_<machine>_<state>_due`) + an appended `set_field` action (`"today + " + duration`) on **every** already-compiled transition/auto Event landing on `state` | CAP-A11's date-arithmetic grammar, reused verbatim (`duration` is just the `"N Unit"` half — the compiler always prepends `"today + "`) |
| `sla[].on_breach` | one generated scheduled `Event` (`evt_<machine>_<state>_sla_breach`), `condition: {status equals state}`, actions = the declared `notify` + an optional `set_field {status: escalate_to}` | CAP-E06 guard + CAP-E03 schedule (`date_field` = the due Field, `offset_days: 0`) — no `Permission` generated, scheduled Events already bypass `Guard.CanTrigger` entirely |

**`requirements[].type` is `"evidence"` or `"approval"`** (CAP-W01, CAP-W03). For `evidence`,
`target` names a *child* Machine that must itself hold a `reference` Field pointing back at this
Machine (validated at load time, same "Unknown = explicit" discipline as `child_lines`/CAP-F16);
`cardinality` is `"N"` (exact), `"N..*"` (at least N), or `"N..M"` (a bounded range). **The count
is never computed by a query at transition time** — it is maintained by *write-time fan-in*: every
time a `target`-Machine record is `Create`d referencing the parent, the parent's counter
increments on the spot (`internal/handler/requirement.go`'s `stampRequirementCounters`, wired into
the plain HTTP `Create` route only — `create_record`/CSV import/the JSON API do not stamp it yet,
a named gap). Two transitions naming the same `(type, target)` share one counter; naming it with
two *different* `to` states is a load-time error (no OR-condition support).

**`type: approval` (CAP-W03's declarative quorum form)** compiles to the exact same
`aggregate_status` `EventAction` a hand-authored parallel-approval pair would carry by hand — but
injected onto the `target` Machine's own Events, not this one's:
```yaml
requirements:
  - type: approval
    target: mch_ca_review       # a child Machine, one record per voter
    min_approvals: 2            # "N" -- "M" is never declared; it's however many
                                 # sibling records currently reference the parent
    on_quorum_approved: Approve # a transition NAME on THIS machine's own process
    on_quorum_rejected: Reject  # ditto -- both MUST be actor: {role: System}
```
No `approve_state`/`reject_state` key: the compiler finds every Event on `target` that sets a
`value_list` Field literally named `"Decision"` (the same convention `handler.doAggregateStatus`
already reads at runtime, `CAP-A07`/`CAP-A08`'s own heuristic, not a new one) and appends the tally
action there — `doAggregateStatus` re-tallies fresh from every sibling on each call, so which
specific value triggered it doesn't matter. `target` needs no `process` block of its own; it can
stay entirely hand-authored, as long as it declares that `Decision` field (with `"Approved"`/
`"Rejected"` among its values) and a `reference` field back to the declaring Machine. Compiled
once every Machine has fully loaded (`internal/metadata/loader.go`'s `compileApprovalRequirements`,
run after `validateReferences`) — not inside the process compiler itself, the same "wait until
everything's loaded" reasoning `evidence`'s own target-validation already needs, one step further
(this one *writes* to the target, not just reads it). At most one `approval` requirement per
Machine is supported so far (no dedup/merge across transitions the way `evidence` has — no case
has needed a second yet).

**`sla[]` (CAP-W04)** declares a time budget for sitting in a state — `duration` reuses CAP-A11's
own unit vocabulary exactly (`Day(s)`/`Week(s)`/`Month(s)`/`Year(s)`/`Business Day(s)`, e.g. `"2
Business Days"`), never a new grammar; `on_breach.notify` is the identical `{role, ...}`/
`{recipient_field, role}` shape `ActionNotify` already uses; `on_breach.escalate_to` is optional
— a notify-only breach (no forced state change) is a valid, simpler declaration. Entirely
single-Machine — no cross-machine wiring, unlike `type: approval` above (see
`aggregate_status`'s own `min_approvals` note in Event Actions below for the mechanism it compiles
to).

**Rules enforced at load time (fail-loud, same posture as everywhere else in this document):**

- A Machine declares `process` **or** hand-authored `events` — never both. Mixing the two is
  undefined on purpose (ambiguous-merge avoidance), not silently guessed at.
- Every `states` entry must be non-empty and declared once; every `transitions[].from`/`to` and
  `auto[].from`/`to` must name a declared state; every state must be reachable from `states[0]`
  (the initial state) via some transition or auto step — an unreachable state fails the load.
  Two `transitions` compiling to the same event id (a name collision after slugging) also fails.
- `actor.role` is required — a transition nobody may perform is a declaration error.
  `actor.owner_field`, when set, must name a real `type: user` Field on the same Machine (same
  check `permissions.owner_field` already gets elsewhere in this document).
- `auto` may not form a cycle (`A→B`, `B→A` chained through `trigger_event` would loop forever
  at runtime), and at most one `auto` entry may leave any given state.
- A `requirements[].target` must exist and hold a `reference` Field back to the declaring
  Machine — checked once every Application has loaded (`metadata.validateReferences`), not
  inside the process compiler itself, since a Requirement may target a Machine in an Application
  that hasn't loaded yet at the point its own Machine compiles.
- `type: approval` additionally requires: `min_approvals > 0`; `on_quorum_approved`/
  `on_quorum_rejected` each name a transition declared on the SAME Machine's own process, and that
  transition's `actor.role` must be `"System"` (enforced, not conventional — a quorum-controlled
  outcome triggerable by a human role would make the quorum guarantee bypassable); `target` must
  have a `value_list` Field literally named `"Decision"` whose values include both `"Approved"`
  and `"Rejected"`, and at least one Event that sets it.
- `sla[].state` must be a declared state, named at most once; `duration` must match `"N Unit"`
  (CAP-A11's vocabulary, no sign — the compiler always prepends `"today + "`);
  `on_breach.escalate_to`, when set, must itself be a declared state — and, per the reachability
  rule above, counts as a real edge into that state (an SLA-only-reachable state is not
  wrongly flagged unreachable).

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
| `computed` | Derived value, never stored — resolved fresh at render time (CAP-F14) | Line Total (Price × Quantity) | `source_field` — mandatory, plus one of `factor` or `factor_field` |

**`child_table` (CAP-F16, ✅ implemented 2026-07-12)** is not a primitive either — its rows are ordinary
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
predetermined target, kept as its own named type today only because the runtime did not, for a
while, have that target to point at. Two of the three now have real, working targets; only
`money`'s target is still a placeholder:

| Type | Reference target | Target status |
|------|-------------------|-----------------|
| `user` | Platform identity | ✅ CAP-F05, 2026-07-12 — a real picker scoped to CAP-O01's `users`/`user_application_roles`, storing a user id (never a display name), referential integrity enforced at Create/Update |
| `money` | Currency (code + exchange rate) | ⚠️ CAP-F08 renders/validates/computes correctly (below), but `currency`/`currency_field` still just names a `value_list`/`text` field holding a code string — Currency as its own canonical, identity-bearing Machine (CAP-O02-style master data) is CAP-F17's still-open half |
| `file` | Runtime-managed File/Document entity | ⚠️ CAP-F06, 2026-07-12 — uploads are genuinely stored and served (below), but as a disk-backed field value, not yet a first-class Machine with its own identity/lifecycle/versioning |

**`type: money` MUST include `currency` (fixed code, e.g. `"IDR"`) or `currency_field` (a reference
to another field on the same record, for CAP-F17's per-transaction currency) in its `options`.**
Metadata declaring `money` without either is incomplete — the same discipline already required for
`value_list` (`values`) and `reference` (`target_machine`), and load-time enforced since
2026-07-12 (`metadata.Loader`'s money-options check). A money field renders as a number input and
displays with its resolved currency: `IDR 15000`.

```yaml
- id: fld_ad_amount
  name: Amount
  type: money
  options: { currency: "IDR" }

- id: fld_inv_amount        # CAP-F17 — per-transaction currency instead of a fixed one
  name: Amount
  type: money
  options: { currency_field: fld_inv_currency }
```

### `file` — real storage + image handling `options` (CAP-F06, ✅ implemented 2026-07-12)

`file` does not get a separate `image` type. Whether a file is an image is a **processing policy**
on the same reference-sugar `file` type, not a different reference target — the same reasoning that
keeps `rich_text` a variant of text handling rather than a different kind of reference. This is the
metadata-facing fact; only the `options` schema below is something a metadata author writes.

```yaml
- id: fld_ad_photo
  name: Photo Evidence
  type: file
  options:
    accept: image/*        # MIME allow-list; a rejected type is a 400 at Create/Update, not a silent drop
    compress: true          # runs the real server-side pipeline below, regardless of client-side compression
    max_dimension: 1920     # resize policy (longest edge, px; only ever shrinks, never upscales)
    format: webp            # "webp" (real WebP via libwebp) or omit for JPEG (default)
```

An uploaded file is now genuinely stored (`prototype/go/uploads/` on disk, keyed by an
unguessable 32-byte token) and served back at `GET /files/{key}` — before 2026-07-12, the picker
rendered but Create/Update never read the multipart body at all, so a selected file silently
vanished. `compress: true` always re-runs server-side (stdlib `image` decode, resize, JPEG/WebP
re-encode) even if the browser already compressed client-side — the server never trusts that
happened, the same "client is advisory, server enforces" principle CAP-C09 already applies to
Constraints. `accept` is enforced against the upload's actual `Content-Type`, not just the file
extension.

*How* the runtime realizes `compress: true` (client-side vs. server-side, and the enforcement rule
that the server never trusts client-side compression alone) is a runtime-behavior concern, not a
metadata-schema one — it is documented once, authoritatively, in `capability-registry.md` (CAP-F06)
and `nfr-standards.md` §2.1, not repeated here.

Full reasoning, the decision tree for choosing between `value_list` / `reference` / a primitive, and
worked examples: `runtime/benchmarks/005-field-modeling-decision-framework.md`.

### `time` / `date_time` / `duration` (CAP-F10, ✅ implemented 2026-07-12)

`time` and `date_time` render real HTML5 inputs (`<input type="time">` / `<input
type="datetime-local">`), not a text fallback. `duration` is stored as a plain integer count of
**minutes** (rendered as a number input with a "minutes" suffix) — there is no structured
duration grammar (ISO 8601 duration, or a value+unit pair) yet; this is a deliberate, named
simplification, not an oversight.

### `computed` — derived field, never stored (CAP-F14, ⚠️ one sub-pattern implemented 2026-07-12)

```yaml
- id: fld_pol_total
  name: Total
  type: computed
  options:
    source_field: fld_pol_price
    factor: 1.1                    # a FIXED multiplier -- e.g. a tax-inclusive markup

- id: fld_inv_base_amount           # CAP-F17's own base-currency mirror
  name: Base Amount (IDR)
  type: computed
  options:
    source_field: fld_inv_amount
    factor_field: fld_inv_rate      # a PER-RECORD multiplier -- another Field's own value
```

`data[source_field] * multiplier` resolved fresh every time the record is displayed (List/
Detail) — never written to the database, never read from a Create/Update form (a submitter
POSTing a value for this field is silently ignored, same trust posture as CAP-F18's
auto-numbers). Exactly one of `factor` (a fixed constant) or `factor_field` (another `number`/
`money` Field on the same Machine, read live) is meaningful; `factor_field` wins if both are
set. `source_field` (and `factor_field`, when set) must both be real `number`/`money` Fields on
the same Machine — validated at load time.

**Only one sub-pattern is implemented**: a single source × a single multiplier. Not a real
formula language — no function calls, no multi-operand expressions, no non-numeric arithmetic.
Cross-record aggregate rollups are a *different* existing mechanism (CAP-A14's
`aggregate_condition` / `SumField`), not this one.

### Field defaults (CAP-F15, ✅ implemented 2026-07-12)

```yaml
- id: fld_pr_priority
  name: Priority
  type: text
  options: { default: "Normal" }
```

`options.default` applies whenever Create's form doesn't expose the field at all, or exposes it
and the submitter leaves it blank — generalizes the pre-existing "a `value_list` field not
shown on the form starts at its first declared value" convention to any field, any type.

### Auto-numbering (CAP-F18, ✅ implemented 2026-07-12)

```yaml
- id: fld_inv_number
  name: Invoice Number
  type: text
  options:
    auto_number_prefix: "INV-"
    auto_number_padding: 4          # 0 or omit = no zero-padding, just prefix + plain number
```

A `text` field with `auto_number_prefix` generates its own value at Create when left blank —
`"INV-0001"`, `"INV-0002"`, ... — backed by a per-(Machine, Field) counter (`field_sequences`),
claimed via a single atomic `INSERT ... ON CONFLICT DO UPDATE ... RETURNING` (never a
SELECT-then-UPDATE, which would let two concurrent Creates read the same "next" value and both
generate the same number). A submitter-POSTed value for this field is ignored, same as
`computed` fields above.

### Multi-currency money and Quantity/UoM conversion (CAP-F17/CAP-F19) — composition, not new mechanisms

Both capabilities are explicitly **not** a new field type — they compose from what's already
above, exactly the framing each capability's own registry row calls for:

```yaml
# CAP-F17: multi-currency money
fields:
  - { id: fld_inv_currency, name: Currency, type: value_list, options: { values: [IDR, USD, EUR] } }
  - { id: fld_inv_amount, name: Amount, type: money, options: { currency_field: fld_inv_currency } }
  - { id: fld_inv_rate, name: "Rate to IDR", type: number }
  - { id: fld_inv_base_amount, name: "Base Amount (IDR)", type: computed,
      options: { source_field: fld_inv_amount, factor_field: fld_inv_rate } }

# CAP-F19: Quantity / unit-of-measure conversion, Tier 1 (flat factor pair)
fields:
  - { id: fld_shp_quantity, name: Quantity, type: number }
  - { id: fld_shp_unit, name: Unit, type: value_list, options: { values: [Kg, G] } }
  - { id: fld_shp_factor, name: "Factor to Grams", type: number }
  - { id: fld_shp_base_qty, name: "Quantity (Grams)", type: computed,
      options: { source_field: fld_shp_quantity, factor_field: fld_shp_factor } }
```

CAP-F19's Tier 2 (a CAP-F16 child table of conversion rows) and Tier 3 (a CAP-F13
history-tracked conversion-factor Machine) remain unexercised — no case has evidenced either
yet; escalate only when cardinality actually demands it, per this capability's own name.

### Document generation (CAP-F21, ⚠️ HTML output only, implemented 2026-07-12)

A `document`-type View (see Views below) renders an `html/template` source against one
record's own data — the reverse direction of `file` above (this stores what a *user* uploads;
`document` generates output from a template + the record's own data, computed at render time,
nothing stored). Output is HTML, not a binary PDF/image — a browser's own "print to PDF" is the
practical stand-in for this prototype; a real binary renderer is a separate, deferred concern.

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

A third guard, `aggregate_condition` (CAP-A14, ✅ implemented 2026-07-12), gates an Event on a
computed SUM across sibling records — "may only be triggered once this Member's total points
reach 100," not just a check on the triggering record's own fields:

```yaml
- id: evt_pe_award
  name: Award
  aggregate_condition:
    aggregate_field: fld_alpe_points   # numeric field summed
    scope_field: fld_alpe_member       # sibling records sharing this record's own value here are included in the sum
    operator: greater_than_or_equal
    value: "100"
  actions:
    - create_record: { target_machine: mch_al_badge, fields: { member: fld_alpe_member } }
```

`machine` (optional, defaults to the Event's own Machine — the common case, summing a field
across other records of the same Machine) may name a different Machine to sum over instead.

---

## Event Sources — `schedule` and Webhooks (CAP-E02/E03/E04, 2026-07-12)

Every event covered above is triggered by a person (an HTTP POST) or by another event
(`aggregate_status`/CAP-I01). Two more trigger sources exist, both real background mechanisms,
not simulations:

**Time-driven (CAP-E02)** — `schedule: { time: "HH:MM" }` fires the event on the first
scheduler tick (`cmd/server/main.go`'s `runScheduler`, a real `time.Ticker` once a minute) at
or after that time of day, once per record per day (de-duplicated via the existing
`record_events` audit table, no new tracking table).

**Date-driven (CAP-E03)** — `schedule: { date_field: fld_due, offset_days: -1 }` fires when
*today* equals that record's own date field plus `offset_days` — a per-record trigger point,
not "every record with this field." `time` and `date_field` are mutually exclusive; the
scheduler picks CAP-E02 vs. CAP-E03 behavior by which key is present.

```yaml
- id: evt_esr_remind
  name: Remind
  schedule: { time: "00:00" }          # CAP-E02 — any hour satisfies "00:00"
  actions:
    - set_field: { field: fld_reminded, value: "Yes" }

- id: evt_est_due_soon
  name: Due Soon
  schedule: { date_field: fld_est_due, offset_days: -1 }   # CAP-E03 — fires 1 day before fld_est_due
  actions:
    - set_field: { field: fld_notified, value: "Yes" }
```

**External (CAP-E04)** — any event can also be triggered by an inbound webhook, no metadata
key required on the event itself; it's a per-Machine credential in `config` (see below) plus a
fixed route, `POST /webhooks/{machine_id}/{record_id}/{event_id}` with header
`X-Webhook-Secret: <the configured secret>`. No session, no CSRF token — this is the one
trigger path that deliberately bypasses both (external systems have no browser session). A
webhook may also stamp payload fields directly onto the record via InputFields (see below) —
`"input:<field>"` resolves from the POST body, not a form.

**Idempotency (CAP-X13, 2026-07-12)** — a webhook caller MAY add header
`X-Idempotency-Key: <any string the caller considers unique to this delivery>`. A repeated
delivery with the SAME key against the same `(machine_id, event_id)` returns `200` without
re-running the event a second time — the same "duplicates are success, not errors" contract
Stripe/Shopify/GitHub's own webhook conventions use, safe for a caller to retry on a timeout
without double-processing. No key at all skips the check entirely; this is opt-in per delivery,
not a requirement.

---

## Event Schema Declaration (CAP-I02, 2026-07-12)

An event may declare `category`, `schema_version`, and `deprecated_message` — metadata *about*
the event's own contract, consumed by CAP-I01 Subscriptions below and by the Detail page (a
deprecated event still fires, for backward compatibility, but shows a "Deprecated" badge and
logs a warning).

```yaml
- id: evt_into_legacy_notify
  name: Legacy Notify
  category: notification
  schema_version: 1
  deprecated_message: "Use evt_into_placed instead"
  actions:
    - set_field: { field: fld_into_notified, value: "Yes" }
```

---

## Cross-Machine Event Subscriptions (CAP-I01/I03, 2026-07-12)

A Subscription lets a Machine react to an event published on a **completely different**
Machine, without that publisher ever naming its subscribers — the publisher's own metadata
stays unaware Order Placed feeds an Audit Log and a Points Ledger; each subscriber declares its
own interest instead (Pattern C). This is a separate top-level element, a sibling of
`machine:`, not nested under either the publisher or the subscriber Machine:

```yaml
event_subscription:
  id: sub_audit_on_order_placed
  publisher_event: evt_into_placed        # the Event id this reacts to, anywhere in the workspace
  machine: mch_int_audit_log              # the Machine a new record is created on
  fields:
    subject: "field:fld_into_customer"    # same create_record-style field mapping -- "field:<id>" copies, anything else is literal/dynamic
    amount: "field:fld_into_total"
  contract:                               # CAP-I03, optional — same shape as event.condition
    field: fld_into_total
    operator: greater_than
    value: "100"
  on_violation: skip                      # only "skip" exists today — the action just doesn't fire
```

Dispatch runs immediately after the publisher event's own write commits (same call site as
`activate_next`/`aggregate_status`), so a Subscription's own failure can never roll back the
publisher, and multiple Subscriptions on the same publisher Event are independent of each
other. `contract` (optional): when present, the Subscription's action only fires if the
publisher's resulting data satisfies it — same `field`/`operator`/`value` shape as a
Constraint's `condition`, same operator-implementation caveats noted above. Any number of
Subscriptions, from any number of unrelated publisher Events, may target the same subscriber
Machine (CAP-I05) — accumulating contributions from independent sources into one shared record
set is a usage pattern of this mechanism, not a separate one.

---

## Event Actions

Actions describe what the runtime should do when an event occurs.

| Action | Description | Example |
|--------|-------------|---------|
| `set_field` | Set a field to a value — literal, a dynamic token (`today`, `now`, `current_user`, CAP-A02), date arithmetic (`"today + 7 Days"`, CAP-A11), or an `"input:<field>"` inline-input reference (CAP-P04) | Set Status = Submitted; Set Approved Date = today |
| `notify` | Send a notification — to a static `role`, or dynamically to the person named by another field's value (`recipient_field`, CAP-A04) | Notify Manager; notify whoever `fld_ad_submitted_by` names |
| `create_record` | ✅ CAP-A06 — create a record on another Machine, copying/mapping fields from the source | Create Audit Log |
| `cross_set_field` | ✅ CAP-A13 — update a field on a **different**, already-existing record, reached via a `reference` field on this record | Marking a Ticket Resolved also sets its linked Asset's Status |
| `batch_generate` | ✅ CAP-A15 — create N records from one action; N comes from a field's value or a literal | Splitting a Purchase Request into N Purchase Order lines |
| `activate_next` | CAP-A07 — in a Sequential-mode workflow, notify the next still-undecided sibling once this one is decided | Approving Step 1 notifies Step 2's approver |
| `aggregate_status` | CAP-A08 — roll a decided record's siblings up to their shared parent: cascade the parent to a "some rejected" event immediately, or to an "all approved" event only once every sibling has decided the same way | All Steps Approved → Document Approved |

Any of the above may be wrapped in `if: { field, operator, value }` (CAP-A09, ✅) to run only
conditionally within an event that has several actions — distinct from the event-level
`condition` (a guard on whether the event may fire at all).

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

### `notify` — static role or dynamic recipient (CAP-A03/CAP-A04)

```yaml
- notify: { role: Manager }                          # CAP-A03 — every account holding this role in the Application
- notify: { recipient_field: fld_ad_submitted_by }    # CAP-A04 — the specific person named by this field's value on the record
```

`recipient_field` wins when both are present. In-app delivery only (CAP-A10) — no email/SMS
exists in this prototype; every notification lands in the recipient's own `/notifications`
inbox, whose per-user "immediate"/"digest" presentation is a runtime feature (CAP-O05), not a
metadata concept — nothing to declare here for that part.

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

**`min_approvals` (CAP-W03, Process Overlay B4 Part 2, 2026-08-22) — optional quorum-of-N,
backward-compatible.** Omitted or `0` is the exact ALL-required behavior described above,
unchanged.

```yaml
- aggregate_status:
    parent_field: fld_qv_request
    parent_event_if_all_approved: evt_qr_approve
    parent_event_if_any_rejected: evt_qr_reject
    min_approvals: 2   # "2 of 3" — doesn't wait for every sibling to decide
```

Set to N over M total siblings sharing the same parent: `parent_event_if_all_approved` fires as
soon as `count(Approved) >= N` — a still-undecided sibling no longer blocks quorum once enough
others have approved, real N-of-M semantics, not "all-but-late-deciders." `parent_event_if_
any_rejected` fires only once quorum becomes **mathematically impossible**
(`count(Rejected) > total_siblings - N`, i.e. too few non-rejected siblings remain to ever reach
N) — a minority of rejections that still leaves enough headroom does *not* cancel early, unlike
the single-rejection-cancels default above.

**Also declarable via `process.requirements[].type: approval`** (Process Overlay, `## Process
Overlay` section above) — the compiler injects this exact action onto the `target` Machine's own
Events once every Machine has loaded (`compileApprovalRequirements`,
`internal/metadata/loader.go`). The declarative grammar deliberately differs from an earlier
sketch once considered here (`{type: approval, target: mch_x_step, quorum: "2_of_3"}`): "M" is
never declared (it's always however many sibling records exist, a runtime fact `doAggregateStatus`
already computes this way even for the hand-authored form above) — only `min_approvals` (the "N"),
matching this section's own param name exactly.

### Event `input_fields` — trigger-time inline input (CAP-P04)

An Event may declare `input_fields: [ fld_x, ... ]` — Field ids collected fresh at trigger
time (rendered as an inline picker/input alongside the trigger button, not read from the
record's own current data) and made available to that event's own actions via
`set_field.value: "input:<field_id>"`. This is how a delegation-style reassignment works: the
Detail page's trigger form submits a fresh value the same request the event fires, rather than
requiring a separate Update first.

```yaml
- id: evt_plea_delegate
  name: Delegate
  input_fields: [ fld_plea_approver ]        # rendered as a picker next to the "Delegate" button
  actions:
    - set_field: { field: fld_plea_approver, value: "input:fld_plea_approver" }
    - set_field: { field: fld_plea_delegated_by, value: current_user }
```

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

### `change_policy` (CAP-W07, 2026-08-22) — effective-dated metadata evolution

A Constraint may declare `change_policy` instead of (never alongside — see below) its own
`condition`: which in-flight records this rule applies to. Answers the question neither a
restart-only deploy nor blanket version-pinning can express: does a new or tightened rule reach
records already open under the old one, or only new work?

```yaml
constraints:
  # records_in_states -- applies only while the record is currently in one
  # of the listed states (reads the Machine's own Status field; states must
  # already be among its declared values).
  - id: cst_compliance_note_draft
    rule: Compliance Note is required for cases still in Draft.
    expression:
      field: fld_compliance_note
      operator: required
    change_policy:
      applies_to: records_in_states
      states: [Draft]

  # new_records -- applies only to records created on or after effective_from
  # (YYYY-MM-DD). A record's creation date is never part of its own stored
  # data -- the loader exposes it only for this comparison.
  - id: cst_approval_ref_2026_policy
    rule: Approval Reference is required for cases opened under the 2026 policy.
    expression:
      field: fld_approval_ref
      operator: required
    change_policy:
      applies_to: new_records
      effective_from: "2026-01-01"

  # all_records -- explicit, not accidental: today's default behavior
  # (a rule with no change_policy at all reaches every record immediately),
  # named on purpose so the choice is auditable in the metadata itself.
```

Compiled entirely at load time (`internal/metadata/compile.go`'s `compileChangePolicies`) into
the Constraint's own `condition` — `records_in_states` becomes an `in` check against the Status
field; `new_records` becomes an `on_or_after` check against the record's creation date. No engine
change: the compiled Constraint is checked by the exact same CAP-C09 mechanism (Create, Update,
and every Event trigger) every other Constraint already goes through. A metadata change that adds
or tightens a `change_policy`-scoped rule takes effect on a running system via CAP-X04's
`POST /admin/reload` — no restart, and every record already open when the reload happens keeps
being evaluated correctly against whichever rules its own state/creation-date actually put it
under.

**Load-time contract:** `change_policy` cannot be combined with an explicit `condition` on the
same Constraint — the loader rejects that combination rather than guessing at how the two should
combine (this Constraint grammar has no AND-list). Attach `change_policy` only to a Constraint
that has no `condition` of its own.

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

  - role: Staff                   # CAP-P06 — field-level visibility
    events: []
    can_read: true
    hidden_fields: [ fld_salary ] # excluded from List/Detail/Form for this role, still present for a role without this key

  - role: Visitor                 # CAP-P07 — anonymous (no session) access
    events: []
    can_read: true                # GET only; POST is always rejected for an unauthenticated request regardless of this grant
```

**`hidden_fields`** (optional, CAP-P06): field ids this role's Permission row excludes from
List, Detail, and Form rendering entirely — not merely read-only, invisible. A different
role's own Permission row on the same Machine (with no `hidden_fields`, or a different list)
sees that field normally; visibility is per-role, not per-Machine.

**`Visitor`** (CAP-P07) is not a reserved keyword — it's an ordinary role name. What makes it
"anonymous access" is simply that unauthenticated requests are resolved as role `Visitor` (no
session/no account needed), so a Permission row declaring `role: Visitor` with `can_read: true`
is what actually grants public read access to a Machine. A Machine with no `Visitor` Permission
row denies anonymous requests, same deny-by-default rule as any other role. Anonymous requests
can never write, regardless of what a `Visitor` row declares — `can_create`/`can_edit` are only
ever consulted for authenticated requests.

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
| `form` | Input surface for creating or updating a record. `child_lines` (CAP-F16) embeds N rows of a child Machine; `steps` (CAP-V12) splits fields into a multi-step wizard |
| `list` | Table or card presentation of multiple records. `filter` (CAP-V09/V05), `manual_order` (CAP-V14), and free-text search (`?q=`, CAP-V08, no config — always available) all apply here |
| `detail` | Full presentation of a single record. Reverse-reference sub-lists (CAP-V06 — every record on another Machine whose `reference` field points back at this one) render automatically, no config |
| `dashboard` | ✅ CAP-V10 — composes summary tiles from `sections`, each independently naming its own source Machine (may span several different Machines, the actual point of this view type vs. a single Machine's own `list`/`report`) |
| `calendar` | ✅ CAP-V07 — records grouped by `date_field`'s own value, server-rendered (no JS month-grid widget, matching this prototype's no-SPA posture) |
| `timeline` | ✅ CAP-V07 — records ordered chronologically by `date_field` |
| `report` | ✅ CAP-V13 (metadata type name is `report`, not `aggregate_report`) — group-by/rollup over ANOTHER Machine's records (Trial Balance, Leaderboard-shaped), computed at render time, nothing stored; requires `report: { machine, group_field, sum_fields }` |
| `document` | ⚠️ CAP-F21, HTML output only — renders `template` (an `html/template` source, `{{.fld_x}}` merge fields, auto-escaped) against one record's own data at `GET /{machine}/{record}/document`; computed at render time, nothing stored |

### View `config` per type

```yaml
- id: vw_list_overdue
  name: My Overdue Tasks
  type: list
  columns: [ fld_title, fld_due ]
  filter:                                    # CAP-V09 — AND-combined row filter, same expression
    - { field: fld_owner, operator: equals, value: "$current_user" }   # CAP-V05 — sentinel, resolves to the acting identity at request time
    - { field: fld_due, operator: after, value: today }

- id: vw_list_manual
  name: Priority Order
  type: list
  columns: [ fld_title ]
  manual_order: true                         # CAP-V14 — Up/Down controls, sorts by a free-standing sort_order column instead of default_sort/created_at

- id: vw_calendar
  name: Task Calendar
  type: calendar
  columns: [ fld_title ]
  date_field: fld_due                        # CAP-V07 — required for calendar/timeline

- id: vw_dashboard
  name: Ops Overview
  type: dashboard
  sections:                                  # CAP-V10 — each section is an independent Machine
    - { title: "Open Tasks", machine: mch_task, group_field: fld_stage }
    - { title: "Projects", machine: mch_project }

- id: vw_wizard
  name: New Request (3 steps)
  type: form
  steps:                                     # CAP-V12 — each entry is a Fields subset shown one step at a time
    - [ fld_title, fld_description ]
    - [ fld_amount, fld_category ]
    - [ fld_approver ]

- id: vw_trial_balance
  name: Trial Balance
  type: report
  report:                                    # CAP-V13
    machine: mch_journal_entry_line
    group_field: fld_jel_account
    sum_fields: [ fld_jel_debit, fld_jel_credit ]

- id: vw_certificate
  name: Product Certificate
  type: document
  template: |                                # CAP-F21 -- html/template, {{.fld_x}} merge fields, auto-escaped
    <html><body>
      <h1>Certificate</h1>
      <p>SKU: {{.fld_ftp_sku}}</p>
      <p>Title: {{.fld_ftp_title}}</p>
    </body></html>
```

---

## Auto-Generated JSON API and Metadata Export (CAP-X07/X08, 2026-07-12)

Not metadata an author declares — a consequence of it. Every Machine automatically gets
`GET /api/{machine}`, `GET /api/{machine}/{record}`, `POST /api/{machine}` (same session
auth, same CAP-P05 permission trimming and CAP-P06 `hidden_fields` stripping as the HTML
routes; CSRF via an `X-CSRF-Token` header since a JSON body has no `csrf_token` form field).
Not yet reachable through this API: CAP-F16 child-table rows, CAP-V12 wizard steps, event
triggering — plain Create/Read on a Machine's own fields only.

Any workspace Admin can pull `GET /apps/{application}/export` — that Application's full
metadata tree (every Machine's Fields/Events/Constraints/Permissions/Views/Config), as JSON,
straight from the loaded Application Model. Read-only for now; there is no import endpoint —
metadata still only enters this runtime via SQL seeds (or, upstream of that, the `.menata` →
Runtime Metadata authoring pipeline this whole document describes).

`GET /files/{key}` (CAP-F06) serves whatever a `file` field's own value points at — also not
something a metadata author declares; `key` is generated at upload time (an unguessable
32-byte token), never authored.

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

**Operators implemented (updated 2026-08-22)**: `required`, `equals`, `not_equals`, `after`/
`before`, `on_or_after`/`on_or_before` (CAP-W07, literal `"today"` accepted on either side same
as `after`/`before`), `greater_than`/`less_than`/`greater_than_or_equal`/`less_than_or_equal`
(CAP-C05), `in` (CAP-W07 — membership against `values: [...]`, e.g. `records_in_states`), plus
**cross-field comparison** — `value` may name another Field id instead of a literal, e.g.
`{ field: fld_end_date, operator: after, value: fld_start_date }` compares two fields on the same
record (CAP-C07). **Composite uniqueness** (CAP-C12) is a different shape entirely, not an
operator on a single Constraint — see `constraints.unique_together` below. Still **not
implemented**: `unique` as a plain field-level operator (use `unique_together` instead), and any
aggregate shape (`aggregate: sum`) or plural `conditions:` list. **Since CAP-X05 (2026-07-12),
an unrecognized operator is a load-time error, not a silent no-op** — `validateOperators`
(`internal/metadata/loader.go`) rejects any Constraint/Event condition/View filter naming an
operator outside `model.SupportedOperators`, so writing one of the gaps above now fails to load
rather than silently never firing; name the gap instead (`capability-registry.md`'s CAP-C10 row)
rather than writing it as metadata.

```yaml
constraints:
  - id: cst_end_after_start
    rule: End Date must be after Start Date.
    expression: { field: fld_end_date, operator: after, value: fld_start_date }

  - id: cst_line_seq_unique
    rule: Sequence must be unique within a Journal Entry.
    unique_together: [ fld_jel_entry, fld_jel_sequence ]   # CAP-C12 — a distinct shape from `expression`, no `operator`
```

**`set_field.value` supports a literal string, three dynamic tokens (`today`, `now`,
`current_user`, CAP-A02), an `"input:<field_id>"` reference (CAP-P04, see InputFields below),
and — as of 2026-07-12 (Batch 3) — date arithmetic on `today`/`now`/a date-typed field**:
`` "<base> + N <unit>" `` / `` "<base> - N <unit>" `` where `<base>` is `today`, `now`, or a
Field id, and `<unit>` is one of `Day`/`Days`, `Week`/`Weeks`, `Month`/`Months`, `Year`/`Years`,
or `Business Day`/`Business Days` (CAP-A11; the `Business Day` variant additionally skips
weekends and the acting Workspace's own declared holidays — CAP-O06, `workspace_holidays`).

```yaml
actions:
  - set_field: { field: fld_due_date, value: "today + 7 Days" }
  - set_field: { field: fld_target_date, value: "today + 5 Business Days" }
  - set_field: { field: fld_follow_up, value: "fld_completed_at + 1 Month" }
  - set_field: { field: fld_stage, value: "next" }     # CAP-A12 — advances a value_list field to its next declared option (its own `values` order); does NOT wrap around past the last value
```

Still **not evaluated at all, silently wrong data**: a function call
(`raise_one_level(priority)`, `sla_offset(priority)`), non-date field arithmetic
(`reopen_count + 1`), template interpolation (`{{ this.field }}`), a `previous(field)` read.
None of these expression forms exist in the runtime today.

**`create_record` is implemented (CAP-A06, ✅, 2026-07-12)** — creates a real record on
another Machine, copying/mapping fields from the source record.

```yaml
- create_record:
    machine: mch_audit_log
    fields:
      subject: "field:fld_title"   # "field:<id>" copies the source record's own field value
      status: "Logged"             # anything else is a literal or a dynamic token/date-arithmetic expression (same set as set_field.value, minus CAP-A12 "next")
```

Two more action types exist beyond the original four-row table (see Event Actions above,
already updated):

```yaml
- cross_set_field:            # CAP-A13 -- updates a field on a DIFFERENT, already-existing record
    record_field: fld_ticket_asset   # a `reference` field ON THIS record naming which other record to update
    field: fld_asset_status           # the field to change, on that OTHER record
    value: Retired

- batch_generate:              # CAP-A15 -- creates N records from one action
    machine: mch_purchase_order_line
    count: 3                          # or a literal read from `fields` at runtime -- see CAP-A15 in capability-registry.md for the field-driven count case
    fields:
      description: "field:fld_pr_description"
    offset_field: fld_pol_due_date    # optional -- each generated record's copy of this field is offset by `i * 1 <offset_unit>`
    offset_unit: "Days"
```

Any action may be wrapped in `if: { field, operator, value }` (CAP-A09) to run only when that
condition is true against the record's data *after* the event's other `set_field`s would apply
— a per-action guard, distinct from the event-level `condition` above.

**Failure semantics (CAP-X12, 2026-07-12): if `create_record`/`cross_set_field`/
`batch_generate` fails at runtime — a `machine` naming a Machine id that isn't real, a
`record_field` pointing at a record that no longer exists — the WHOLE event fails and NOTHING
it did commits**, not just that one action. Every HTTP request runs inside one real database
transaction; an action failure aborts it entirely, rolling back the record's own `set_field`
changes and any earlier action in the same event that had, on its own, already succeeded. A
`machine`/`record_field` value is still trusted, unvalidated metadata at load time (a typo in
it won't fail the boot, only the first trigger that reaches it) — get these right, since a
wrong one now breaks the whole event at runtime instead of just quietly doing nothing.

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
decoding ignores fields a struct doesn't declare — any key not named in `ViewConfig`/
`EventAction.Params`/etc. (`internal/model/model.go`) doesn't error, it just vanishes with no
trace. `views.filter` (CAP-V09), `views.steps` (CAP-V12), and every other config key documented
above IS implemented as of the batches noted next to each — this specific stale warning (a
`filter` block being silently dropped) no longer applies to `filter` itself, but the underlying
risk is general: the absence of a load error is never confirmation that everything you wrote
was understood. Cross-check against what this document and `capability-registry.md` actually
say is implemented (its ✅/⚠️/❌ column), not just against "did it load."

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
