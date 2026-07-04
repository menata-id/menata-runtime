# Menata Runtime — Frappe Prototype

> Metadata proof: design-request.yaml -> Frappe DocType + Workflow + Notifications

---

## Approach

Frappe Framework is a **purpose-built metadata-driven business framework**.

Its core concept — **DocType** — maps almost 1:1 to Machine in Menata Runtime.

```text
design-request.yaml
        |
        ├──> Machine + Fields + Permissions -> Design Request.json         (DocType)
        ├──> Events (status transitions)    -> Design Request Workflow.json (Workflow)
        ├──> Event: notify(Designer)        -> Notify Designer on Submit.json
        └──> Event: notify(Requester)       -> Notify Requester on Complete.json
```

---

## Runtime Metadata -> Frappe Mapping

| Menata Runtime | Frappe |
|----------------|-------|
| Machine | DocType |
| Field | DocField (inside DocType) |
| Field type: text | Data |
| Field type: rich_text | Text Editor |
| Field type: money | Currency |
| Field type: boolean | Check |
| Field type: date | Date |
| Field type: user | Link -> User |
| Field type: file | Attach |
| Field type: value_list | Select (newline-separated options) |
| Field type: reference | Link -> other DocType |
| Event (status change) | Workflow transition |
| Event action: notify | Notification (Email + In-App) |
| Constraint (required) | reqd: 1 on DocField |
| Conditional constraint | validate() method in Controller |
| Permission | DocPerm (role + CRUD flags inside DocType) |
| View (list) | List View (auto-generated) |
| View (form) | Form View (auto-generated) |
| View (detail) | Detail View (auto-generated) |

---

## Metadata Coverage

| Feature | File | Status |
|---------|------|--------|
| Machine + Fields | Design Request.json (DocType) | ✅ Metadata only |
| Permissions | Design Request.json (DocPerm) | ✅ Metadata only |
| Workflow: states + transitions | Design Request Workflow.json | ✅ Metadata only |
| Constraint: title required, description required | Design Request.json (reqd: 1) | ✅ Metadata only |
| Event: notify Designer on Submit | Notify Designer on Submit.json | ✅ Metadata only |
| Event: notify Requester on Complete | Notify Requester on Complete.json | ✅ Metadata only |
| Views (form, list, detail) | auto-generated from DocType | ✅ Metadata only |
| Constraint: attachment required if Design Type = Banner | — | ❌ Cannot be done without code — requires Python `validate()` method in a DocType Controller |

---

## Why Frappe is the Closest to Menata Runtime

Defining a DocType in Frappe is enough to get form view, list view, detail view,
REST API, permissions, audit trail, and import/export — with no additional configuration.

This is exactly what Menata Runtime promises:

> Define the Machine. The runtime realizes the application.

---

## Files

```
frappe-config/
├── Design Request.json                  <- DocType: Machine + Fields + Permissions
├── Design Request Workflow.json         <- Workflow: Events (status transitions)
├── Notify Designer on Submit.json       <- Notification: evt_submit -> notify(Designer)
└── Notify Requester on Complete.json    <- Notification: evt_complete -> notify(Requester)
```
