# Menata Runtime — Drupal Prototype

> Metadata proof: design-request.yaml -> native Drupal configuration

---

## Approach

Drupal is a **configuration-driven CMS framework**.

Every content structure, workflow, role, view, and event rule in Drupal is expressible as YAML configuration files — importable via `drush config:import` without writing PHP code.

```text
design-request.yaml
        |
        |-> Machine + Fields + Status  -> node.type.design_request.yml + field.*.yml
        |-> State machine + transitions -> workflows.workflow.design_request.yml
        |-> Event notifications         -> eca.eca.design_request_notify.yml
        |-> Roles + Permissions         -> user.role.*.yml
        `-> List view                  -> views.view.design_request_my_requests.yml
```

---

## Runtime Metadata -> Drupal Mapping

| Menata Runtime | Drupal |
|----------------|-------|
| Machine | Content Type (node bundle) |
| Field | Drupal Field (field.storage + field.field) |
| Field type: text | Text (plain) |
| Field type: rich_text | Text (formatted, long) |
| Field type: date | Date |
| Field type: user | Entity Reference -> User |
| Field type: file | File field |
| Field type: value_list | List (text) |
| Event (status change) | Workflow transition |
| Event action: notify | ECA rule (Events, Conditions, Actions module) |
| Constraint (required) | Field required flag in config |
| Conditional constraint | — (not expressible in YAML, see Metadata Coverage) |
| Permission | Drupal Role + permission YAML |
| View (list) | Views module configuration YAML |
| View (form) | Drupal Form API (auto-generated from field config) |
| View (detail) | Node display mode (auto-generated) |

---

## Metadata Coverage

**Score: ~85% (14/16 features)**

The metadata files in `docs/examples/drupal-config/` are native Drupal configuration — importable via `drush config:import` without any custom code.

| # | Feature | File | Status |
|---|---------|------|--------|
| 1 | Machine definition | node.type.design_request.yml | ✅ Metadata only |
| 2 | All Fields with correct types | field.storage.node.*.yml | ✅ Metadata only |
| 3 | Status field + all states | field.storage.node.field_status.yml | ✅ Metadata only |
| 4 | State machine enforcement | workflows.workflow.design_request.yml (role per transition) | ✅ Metadata only |
| 5 | Event action: set status on transition | Workflow state update | ✅ Metadata only |
| 6 | Event action: notify Designer on Submit | eca.eca.design_request_notify.yml | ✅ Metadata only |
| 7 | Event action: notify Requester on Complete | eca.eca.design_request_notify.yml | ✅ Metadata only |
| 8 | Constraint: Title required | field config (required flag) | ✅ Metadata only |
| 9 | Constraint: Description required | field config (required flag) | ✅ Metadata only |
| 10 | Constraint: Due Date after today | — | ❌ Cannot be done without code — Drupal has no native "after today" date constraint configurable via YAML; requires PHP constraint plugin |
| 11 | Constraint: Attachment if Design Type = Banner | — | ❌ Cannot be done without code — needs PHP validation in a custom module |
| 12 | Permission: Requester role | user.role.requester.yml | ✅ Metadata only |
| 13 | Permission: Designer role | user.role.designer.yml | ✅ Metadata only |
| 14 | View: Form | Drupal edit form (auto-generated from field config) | ✅ Metadata only |
| 15 | View: List | views.view.design_request_my_requests.yml | ✅ Metadata only |
| 16 | View: Detail | Node display mode (auto-generated) | ✅ Metadata only |

---

## Files

```
drupal-config/
├── node.type.design_request.yml                  <- Content Type: Machine
├── field.storage.node.field_design_type.yml      <- Field: Design Type (List text)
├── field.storage.node.field_status.yml           <- Field: Status (List text, all states)
├── workflows.workflow.design_request.yml         <- Workflow: 6 states, 5 transitions
├── eca.eca.design_request_notify.yml             <- ECA: notify Designer + notify Requester
├── views.view.design_request_my_requests.yml     <- View: My Requests (List)
├── user.role.requester.yml                       <- Role: Requester (Submit)
└── user.role.designer.yml                        <- Role: Designer (Accept, Reject, Start, Complete)
```

---

## Key Insight

Drupal proves that a mature CMS framework can express most of a business application as pure configuration YAML.

Its Workflow module handles state machine enforcement natively — something Directus and Budibase cannot do in metadata.

The gap is constraint expressiveness: Drupal YAML cannot express "after today" or conditional field requirements without a PHP plugin.
