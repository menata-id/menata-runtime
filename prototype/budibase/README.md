# Menata Runtime — Budibase Prototype

> Metadata proof: design-request.yaml → Budibase app.json

---

## Approach

Budibase is an **open source low-code platform** where the entire application is a single portable JSON file.

The "interpreter" reads Runtime Metadata YAML and produces a Budibase app definition.

```text
design-request.yaml
        │
        ▼
app.json   → Tables + Automations + Roles + Screens
```

---

## Runtime Metadata → Budibase Mapping

| Menata Runtime | Budibase |
|----------------|---------|
| Machine | Table |
| Field | Column (schema field) |
| Field type: value_list | Options field |
| Field type: user | Link to Users table |
| Field type: file | Attachment field |
| Event | Automation (trigger + steps) |
| Constraint | Field constraints (presence, inclusion) |
| Permission | Role (REQUESTER, DESIGNER) |
| View (list) | Screen with Table component |
| View (form) | Screen with Form component |
| View (detail) | Screen with Detail component |

---

## Files

```
budibase-config/
└── app.json    ← Complete Budibase app definition (tables + automations + roles + screens)
```

---

## Metadata Coverage

**Score: ~65% (10/16 features)**

| # | Feature | File | Status |
|---|---------|------|--------|
| 1 | Machine definition | app.json (table) | ✅ Metadata only |
| 2 | All Fields with correct types | app.json (table schema) | ✅ Metadata only |
| 3 | Status field + all states | app.json (options field) | ✅ Metadata only |
| 4 | State machine enforcement | — | ❌ Budibase has no workflow module — status field is freely editable by anyone with access; transition enforcement is not expressible in metadata |
| 5 | Event action: set status on transition | — | ❌ Budibase automations trigger AFTER row save, not during — cannot set status as part of a controlled transition |
| 6 | Event action: notify Designer on Submit | app.json (automation: ROW_UPDATED + FILTER + SEND_EMAIL) | ✅ Metadata only |
| 7 | Event action: notify Requester on Complete | app.json (automation) | ✅ Metadata only |
| 8 | Constraint: Title required | app.json (field constraint: presence) | ✅ Metadata only |
| 9 | Constraint: Description required | app.json (field constraint: presence) | ✅ Metadata only |
| 10 | Constraint: Due Date after today | — | ❌ Cannot be done without code — Budibase field constraints are static (type/presence only); date range validation requires JavaScript |
| 11 | Constraint: Attachment if Design Type = Banner | — | ❌ Cannot be done without code — conditional validation requires JavaScript in automation step |
| 12 | Permission: Requester role | app.json (roles) | ✅ Metadata only |
| 13 | Permission: Designer role | app.json (roles) | ✅ Metadata only |
| 14 | View: Form | app.json (screen with Form component) | ✅ Metadata only |
| 15 | View: List | app.json (screen with Table component) | ✅ Metadata only |
| 16 | View: Detail | — | ❌ Budibase Table shows row detail via modal by default, but a dedicated Detail screen is not in current proof |

---

## Key Insight

Budibase proves that a **single portable JSON file** can describe a complete running application.

Import `app.json` into any Budibase instance → application is immediately available.

The limitation: dynamic conditional field validation (if X then Y is required) falls outside what Budibase metadata can express without JavaScript code.
