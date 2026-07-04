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

## Metadata Coverage

**Score: ~70% (11/16 features)**

| # | Feature | File | Status |
|---|---------|------|--------|
| 1 | Machine definition | schema-snapshot.json (collection) | ✅ Metadata only |
| 2 | All Fields with correct types | schema-snapshot.json | ✅ Metadata only |
| 3 | Status field + all states | schema-snapshot.json (Select Dropdown) | ✅ Metadata only |
| 4 | State machine enforcement | — | ❌ Directus has no workflow module — anyone with edit permission can set any status value; transition enforcement is not expressible in metadata |
| 5 | Event action: set status on transition | flows.json (update item operation) | ✅ Metadata only |
| 6 | Event action: notify Designer on Submit | flows.json (condition + mail op) | ✅ Metadata only |
| 7 | Event action: notify Requester on Complete | flows.json (condition + mail op) | ✅ Metadata only |
| 8 | Constraint: Title required | schema-snapshot.json (required: true) | ✅ Metadata only |
| 9 | Constraint: Description required | schema-snapshot.json (required: true) | ✅ Metadata only |
| 10 | Constraint: Due Date after today | — | ⚠️ Not in current proof — achievable via Flow with Action(Before) trigger (metadata-only), but not yet demonstrated |
| 11 | Constraint: Attachment if Design Type = Banner | — | ⚠️ Same as above — achievable via Flow (metadata-only) but not yet demonstrated |
| 12 | Permission: Requester role (create, edit own) | roles-permissions.json | ✅ Metadata only |
| 13 | Permission: Designer role (read all, edit status) | roles-permissions.json | ✅ Metadata only |
| 14 | View: Form | auto-generated from collection | ✅ Metadata only |
| 15 | View: List | auto-generated from collection | ✅ Metadata only |
| 16 | View: Detail | auto-generated from collection | ✅ Metadata only |

Note on #10 and #11: these are scored as not-yet-demonstrated (not as ❌) because Directus Flows with Action(Before) trigger can handle them in metadata. The gap in the score is primarily #4 (architectural — no workflow module).

---

## Key Insight

Directus is the thinnest metadata-driven runtime in this proof set.

Schema = application. No framework overhead. No custom code for core features.

The entire application is expressed as three JSON files — the closest existing format to what Menata Runtime Metadata aspires to produce.
