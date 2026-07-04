# Menata Runtime — Directus Prototype

> Metadata proof: design-request.yaml → Directus schema + flows + permissions

---

## Approach

Directus is a **database-first** runtime.

Any SQL table becomes an instant REST/GraphQL API + admin UI.

The "interpreter" reads Runtime Metadata YAML and produces Directus schema + flows + roles.

```text
design-request.yaml
        │
        ▼
Directus Schema Snapshot   → Collections + Fields + Relations
Directus Flows             → Event responses (notify, status change)
Directus Roles/Permissions → Role-based access control
```

---

## Runtime Metadata → Directus Mapping

| Menata Runtime | Directus |
|----------------|---------|
| Machine | Collection |
| Field | Field (per collection) |
| Field type: value_list | Select Dropdown interface |
| Field type: user | Many-to-One → directus_users |
| Field type: file | File (→ directus_files) |
| Event | Flow (trigger: event + operations) |
| Constraint | Field validation + Flow condition |
| Permission | Role + Permission (per collection/action) |
| View (list) | Collection list view (auto-generated) |
| View (form) | Item edit form (auto-generated) |
| View (detail) | Item detail page (auto-generated) |

---

## Files

```
directus-config/
├── schema-snapshot.json    ← Collections + Fields + Relations
├── flows.json              ← Event responses (submit → notify designer, complete → notify requester)
└── roles-permissions.json  ← Requester + Designer roles with permissions
```

---

## Key Insight

Directus proves the **thinnest possible** metadata-driven runtime.

No framework. No custom code.

Schema = application.

The entire application is expressed as three JSON files.
