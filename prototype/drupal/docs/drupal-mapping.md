# Runtime Metadata -> Drupal Mapping

> This document describes how Menata Runtime Metadata concepts map to native Drupal configuration.

---

## Concept Mapping

| Menata Runtime | Drupal |
|----------------|--------|
| Machine | Content Type (node.type.*) |
| Field | Drupal Field (field.storage.* + field.field.*) |
| Event | Workflow transition + ECA rule |
| Constraint (simple) | Field required flag in config |
| Constraint (conditional) | Not expressible in YAML — requires PHP |
| Permission | Drupal Role + workflow transition permission |
| View (list) | Drupal View (views.view.*) |
| View (detail) | Content display mode (core.entity_view_display.*) |
| View (form) | Content form display (core.entity_form_display.*) |

---

## Field Type Mapping

| Runtime Metadata Type | Drupal Field Type | Widget | Formatter |
|----------------------|-------------------|--------|-----------|
| text | string | Text field | Plain text |
| rich_text | text_long | Textarea | Filtered HTML |
| number | decimal | Number | Decimal |
| money | decimal | Number | Decimal (2 decimal places) |
| boolean | boolean | Checkbox | Boolean |
| date | datetime (date only) | Date picker | Date |
| date_time | datetime | Datetime picker | Datetime |
| user | entity_reference -> user | Autocomplete | User name |
| file | file | File upload | File link |
| value_list | list_string | Select | List default |
| reference | entity_reference -> node | Autocomplete | Entity label |

---

## Event Action Mapping

| Runtime Metadata Action | Drupal Realization | Metadata? |
|------------------------|-------------------|-----------|
| set_field (status) | Workflow state transition | ✅ YAML |
| notify | ECA action: send email/message | ✅ YAML |
| create_record | ECA action: create entity | ✅ YAML |

---

## Constraint Expression Mapping

| Operator | Drupal Realization | Metadata? |
|----------|--------------------|-----------|
| required | Field required = TRUE in config | ✅ YAML |
| greater_than (number) | ECA condition | ✅ YAML |
| after (date) | — No native YAML constraint | ❌ PHP required |
| equals (conditional required) | — No native YAML conditional constraint | ❌ PHP required |

---

## Drupal Config Files per Machine

```text
node.type.{machine_id}.yml
field.storage.node.{field_id}.yml           (one per field)
field.field.node.{machine_id}.{field_id}.yml (one per field)
workflows.workflow.{machine_id}.yml
eca.eca.{machine_id}_{event_id}.yml         (one per event)
views.view.{machine_id}_{view_id}.yml       (one per list view)
core.entity_view_display.node.{machine_id}.default.yml
core.entity_form_display.node.{machine_id}.default.yml
user.role.{role_id}.yml                     (one per role)
```
