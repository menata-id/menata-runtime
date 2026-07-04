# Drupal Prototype Architecture

> This document describes how Menata Runtime concepts are realized using Drupal.
>
> Drupal was chosen as a reference because its architecture was one of the inspirations for the Menata Runtime design.
>
> See: [runtime/architecture-benchmark.md](../../architecture-benchmark.md)

---

## Overview

Drupal is a content management framework built on PHP.

Its core architecture provides many of the capabilities required by Menata Runtime out of the box:

- entity model (Machines, Fields)
- views abstraction (Views module)
- plugin architecture (extensible runtime)
- metadata-driven configuration (CMI — Configuration Management Interface)
- workflow engine (Workflow module)
- event-condition-action system (ECA module)
- role-based permissions

The `menata_runtime` custom Drupal module acts as the **Interpreter**.

It reads Runtime Metadata YAML and realizes it using Drupal's built-in systems.

---

## Architecture Overview

```text
Runtime Metadata (YAML)
        │
        ▼
menata_runtime module (Interpreter)
        │
        ├── Content Type API   ──▶ Machine realization
        ├── Fields API         ──▶ Field realization
        ├── Workflow module    ──▶ Status transition events
        ├── ECA module         ──▶ Event responses (notify, create record)
        ├── Views module       ──▶ List + Detail view realization
        ├── Form API           ──▶ Form view realization
        ├── Validation API     ──▶ Constraint enforcement
        └── Roles + Permissions──▶ Permission enforcement
        │
        ▼
Running Drupal Application
```

---

## Runtime Metadata → Drupal Mapping

### Machine → Content Type

Each Machine in Runtime Metadata becomes a Drupal Content Type.

```text
machine:
  id: mch_design_request
  name: Design Request
```

Realized as:

```text
Content Type: design_request
Label: Design Request
```

---

### Field → Drupal Field

Each Field becomes a Drupal field on the content type.

| Runtime Metadata Type | Drupal Field Type |
|----------------------|-------------------|
| `text` | Text (plain) |
| `rich_text` | Text (formatted, long) |
| `number` | Number (decimal) |
| `money` | Number (decimal) — formatted as currency |
| `boolean` | Boolean |
| `date` | Date |
| `date_time` | Datetime |
| `duration` | Number (integer) — stored as minutes |
| `user` | Entity Reference → User |
| `file` | File |
| `value_list` | List (text) |
| `reference` | Entity Reference → Content Type |

---

### Event → Workflow + ECA

Status-changing events are realized using the **Workflow module**.

```text
events:
  - id: evt_submit
    name: Submit
    actions:
      - set_field: { field: fld_status, value: Submitted }
      - notify: { role: Designer }
```

Realized as:

- Workflow transition: `draft → submitted` triggered by `Submit` action
- ECA rule: on transition to `submitted` → send notification to Designer role

---

### Constraint → Field Validation + ECA Condition

```text
constraints:
  - id: cst_title_required
    rule: Title is required.
    expression:
      field: fld_title
      operator: required
```

Realized as:

- Drupal field constraint: `required = TRUE` on `fld_title`

Conditional constraints:

```text
  - id: cst_attachment_required_for_banner
    rule: Attachment is required for Banner design type.
    expression:
      field: fld_attachment
      operator: required
    condition:
      field: fld_design_type
      operator: equals
      value: Banner 2:1
```

Realized as:

- ECA condition: `fld_design_type = Banner 2:1`
- ECA action: validate `fld_attachment` required

---

### Permission → Drupal Role + Permission

```text
permissions:
  - role: Requester
    events: [ evt_submit ]

  - role: Designer
    events: [ evt_accept, evt_reject, evt_start, evt_complete ]
```

Realized as:

- Drupal Role: `requester` — can trigger `Submit` workflow transition
- Drupal Role: `designer` — can trigger `Accept`, `Reject`, `Start`, `Complete` workflow transitions

---

### View → Drupal Views + Display Modes

| Runtime Metadata Type | Drupal Realization |
|----------------------|-------------------|
| `list` | Drupal View (table or card display) |
| `detail` | Content display mode (full) |
| `form` | Node edit form |
| `dashboard` | Drupal View (summary with exposed filters) |
| `calendar` | Drupal Calendar View |

---

## The Interpreter: `menata_runtime` Module

The `menata_runtime` module is the core of this prototype.

It is responsible for:

1. Reading Runtime Metadata YAML from a configured directory
2. Validating Runtime Metadata structure
3. Creating or updating Drupal Content Types via the Entity API
4. Creating or updating Fields via the Fields API
5. Creating or updating Workflow states and transitions
6. Creating or updating ECA rules for event responses
7. Creating or updating Views for list and detail presentations
8. Assigning permissions to Drupal roles

The module exposes a Drush command:

```bash
drush menata:realize path/to/metadata.yaml
```

This command reads the YAML, validates it, and applies the realization to Drupal.

---

## Configuration Management

Drupal uses YAML natively for its Configuration Management Interface (CMI).

When `menata:realize` runs, it creates standard Drupal config YAML files.

These config files can be exported, versioned, and deployed like any Drupal configuration.

```text
Runtime Metadata YAML
        │
        ▼
menata_runtime module
        │
        ▼
Drupal Config YAML (node.type.*, field.*, views.view.*, workflows.*, etc.)
        │
        ▼
config:import
        │
        ▼
Running Drupal Application
```

---

## Why Drupal Validates the Menata Architecture

The Menata Runtime architecture benchmark document identified Drupal as an architectural reference for:

- metadata-driven configuration
- reusable components
- plugin architecture
- entity model
- views abstraction
- configuration management

This prototype demonstrates that Menata's Runtime Metadata maps naturally onto Drupal's existing systems.

Drupal does not need to be modified.

Only a thin interpreter module (`menata_runtime`) is required.

The same Runtime Metadata that drives the Go prototype drives this Drupal prototype.

This validates the Menata principle:

> The same Business Knowledge may be realized through different Machine Interpretations while preserving its meaning.

---

## Separation of Responsibilities

| Concern | Owner |
|---------|-------|
| Business Knowledge | Menata Language |
| Runtime Metadata | YAML (shared format) |
| Interpretation | `menata_runtime` Drupal module |
| Content modeling | Drupal Entity API |
| State machine | Drupal Workflow module |
| Event handling | ECA module |
| Presentation | Drupal Views + Display Modes |
| Authorization | Drupal Roles + Permissions |
| Data storage | PostgreSQL via Drupal |
