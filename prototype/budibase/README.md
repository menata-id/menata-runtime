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

| Feature | File | Status |
|---------|------|--------|
| Machine + Fields | app.json (table schema) | ✅ Metadata only |
| Roles | app.json | ✅ Metadata only |
| Screens: form, list | app.json | ✅ Metadata only |
| Event: notify Designer on Submit | app.json (automation) | ✅ Metadata only |
| Event: notify Requester on Complete | app.json (automation) | ✅ Metadata only |
| Constraint: title required, description required | app.json (field constraints) | ✅ Metadata only |
| Constraint: attachment required if Design Type = Banner | — | ❌ Cannot be done without code — Budibase field validation is static; conditional validation requires JavaScript in an automation step |

---

## Key Insight

Budibase proves that a **single portable JSON file** can describe a complete running application.

Import `app.json` into any Budibase instance → application is immediately available.

The limitation: dynamic conditional field validation (if X then Y is required) falls outside what Budibase metadata can express without JavaScript code.
