# Runtime Metadata → Drupal Mapping

> This document describes how Menata Runtime Metadata concepts map to Drupal systems.
>
> It serves as the reference for implementing the `menata_runtime` interpreter module.

---

## Concept Mapping

| Menata Runtime | Drupal |
|----------------|--------|
| Workspace | Drupal site installation |
| Application | Module group / admin menu section |
| Machine | Content Type (`node.type.*`) |
| Field | Drupal Field (`field.storage.*` + `field.field.*`) |
| Event | Workflow transition + ECA rule |
| Constraint | Field constraint + ECA condition |
| Permission | Drupal Role + workflow transition permission |
| View (list) | Drupal View (`views.view.*`) |
| View (detail) | Content display mode (`core.entity_view_display.*`) |
| View (form) | Content form display (`core.entity_form_display.*`) |

---

## Field Type Mapping

| Runtime Metadata Type | Drupal Field Type | Widget | Formatter |
|----------------------|-------------------|--------|-----------|
| `text` | `string` | Text field | Plain text |
| `rich_text` | `text_long` | Textarea | Filtered HTML |
| `number` | `decimal` | Number | Decimal |
| `money` | `decimal` | Number | Decimal (2 decimal places) |
| `boolean` | `boolean` | Checkbox | Boolean |
| `date` | `datetime` (date only) | Date picker | Date |
| `time` | `datetime` (time only) | Time picker | Time |
| `date_time` | `datetime` | Datetime picker | Datetime |
| `duration` | `integer` | Number | Duration |
| `user` | `entity_reference` → `user` | Autocomplete | User name |
| `file` | `file` | File upload | File link |
| `value_list` | `list_string` | Select | List default |
| `reference` | `entity_reference` → `node` | Autocomplete | Entity label |

---

## Event Action Mapping

| Runtime Metadata Action | Drupal Realization |
|------------------------|-------------------|
| `set_field` (status) | Workflow state transition |
| `notify` | ECA action: send message via Message module or email |
| `create_record` | ECA action: create entity |

---

## Constraint Expression Mapping

| Operator | Drupal Realization |
|----------|--------------------|
| `required` | Field `required = TRUE` |
| `greater_than` | ECA condition + custom validation |
| `after` (date) | ECA condition: date comparison |
| `equals` (condition) | ECA condition: field value comparison |

---

## Drupal Config Files Generated per Machine

For each Machine, the interpreter generates the following Drupal config YAML files:

```text
node.type.{machine_id}.yml
field.storage.node.{field_id}.yml          (one per field)
field.field.node.{machine_id}.{field_id}.yml (one per field)
workflows.workflow.{machine_id}.yml
eca.eca.{machine_id}_{event_id}.yml        (one per event)
views.view.{machine_id}_{view_id}.yml      (one per list view)
core.entity_view_display.node.{machine_id}.default.yml
core.entity_form_display.node.{machine_id}.default.yml
user.role.{role_id}.yml                    (one per role)
```
