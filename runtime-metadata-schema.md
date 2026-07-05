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
      actions:
        - set_field: { field: fld_status, value: Submitted }

    - id: evt_approve
      name: Approve
      actions:
        - set_field: { field: fld_status, value: Approved }

    - id: evt_reject
      name: Reject
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

### `money`, `user`, `file` are "reference sugar"

These three are not independent primitive types conceptually — each is `reference` to a
predetermined target, kept as its own named type today only because the runtime does not yet have
that target to point at:

| Type | Reference target | Target status |
|------|-------------------|-----------------|
| `user` | Platform identity | Pending CAP-O01 (identity & role registry) |
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

## Event Actions

Actions describe what the runtime should do when an event occurs.

| Action | Description | Example |
|--------|-------------|---------|
| `set_field` | Set a field to a value | Set Status = Submitted |
| `notify` | Send a notification to a role | Notify Manager |
| `create_record` | Create a record in another machine | Create Audit Log |

Actions are realized by the runtime.

Business Knowledge should not describe how actions are implemented.

---

## Constraints

Constraints describe business rules that must always be satisfied.

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

Permissions assign events to business roles.

```yaml
permissions:
  - role: Employee
    events: [ evt_submit ]

  - role: Manager
    events: [ evt_approve, evt_reject ]

  - role: HR
    events: [ evt_record_leave ]
```

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
