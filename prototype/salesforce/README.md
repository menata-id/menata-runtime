# Menata Runtime — Salesforce Prototype

> Metadata proof: design-request.yaml → Salesforce Metadata API files

---

## Approach

Salesforce Platform is the **enterprise gold standard** for metadata-driven applications.

Every object, field, workflow, validation rule, permission set, and layout in Salesforce is metadata.

Metadata can be exported, versioned, and deployed as XML files via the Salesforce Metadata API.

This prototype shows how the same Runtime Metadata YAML maps to Salesforce's metadata format.

```text
design-request.yaml
        │
        ▼
DesignRequest__c.object-meta.xml      → Custom Object + Fields + Validation Rules + List Views
DesignRequestSubmit.flow-meta.xml     → Record-Triggered Flow (evt_submit → notify Designer)
DesignRequestComplete.flow-meta.xml   → Record-Triggered Flow (evt_complete → notify Requester)
Requester.permissionset-meta.xml      → Permission Set for Requester role
Designer.permissionset-meta.xml       → Permission Set for Designer role
```

---

## Runtime Metadata → Salesforce Mapping

| Menata Runtime | Salesforce |
|----------------|-----------|
| Machine | Custom Object (`__c`) |
| Field | Custom Field (`__c`) |
| Field type: text | Text field |
| Field type: rich_text | Long Text Area |
| Field type: date | Date field |
| Field type: value_list | Picklist |
| Field type: user | Lookup → User |
| Field type: file | File / ContentDocument |
| Event | Record-Triggered Flow |
| Event action: set_field | Flow element: Update Records |
| Event action: notify | Flow element: Send Email Alert |
| Constraint | Validation Rule |
| Conditional constraint | Validation Rule with formula condition |
| Permission | Permission Set |
| View (list) | List View |
| View (form) | Page Layout |
| View (detail) | Record Page (Lightning App Builder) |

---

## Files

```
salesforce-metadata/
├── DesignRequest__c.object-meta.xml       ← Custom Object + Fields + Validation Rules + List Views
├── DesignRequestSubmit.flow-meta.xml      ← Flow: On Submit → notify Designer
├── DesignRequestComplete.flow-meta.xml    ← Flow: On Complete → notify Requester
├── Requester.permissionset-meta.xml       ← Permission Set: Requester
└── Designer.permissionset-meta.xml        ← Permission Set: Designer
```

---

## Key Insight

Salesforce proves that metadata-driven applications work at **enterprise scale**.

The entire application definition is versionable, deployable, and environment-independent — exactly what Menata Runtime aims for.

Salesforce has been doing this since 2000.

## Metadata Coverage

**Score: 100% (16/16 features)**

Salesforce is the only platform in this proof set where **all features are covered by metadata alone** — including the hardest case: conditional constraint via formula expression.

| # | Feature | File | Status |
|---|---------|------|--------|
| 1 | Machine definition | DesignRequest__c.object-meta.xml | ✅ Metadata only |
| 2 | All Fields with correct types | DesignRequest__c.object-meta.xml | ✅ Metadata only |
| 3 | Status field + all states | Picklist field | ✅ Metadata only |
| 4 | State machine enforcement | Record-Triggered Flow (prior status check) | ✅ Metadata only |
| 5 | Event action: set status on transition | Flow: Update Records element | ✅ Metadata only |
| 6 | Event action: notify Designer on Submit | DesignRequestSubmit.flow-meta.xml | ✅ Metadata only |
| 7 | Event action: notify Requester on Complete | DesignRequestComplete.flow-meta.xml | ✅ Metadata only |
| 8 | Constraint: Title required | ValidationRule | ✅ Metadata only |
| 9 | Constraint: Description required | ValidationRule | ✅ Metadata only |
| 10 | Constraint: Due Date after today | ValidationRule with DATEVALUE > TODAY() formula | ✅ Metadata only |
| 11 | Constraint: Attachment if Design Type = Banner | ValidationRule with conditional formula | ✅ Metadata only |
| 12 | Permission: Requester role | Requester.permissionset-meta.xml | ✅ Metadata only |
| 13 | Permission: Designer role | Designer.permissionset-meta.xml | ✅ Metadata only |
| 14 | View: Form | Page Layout (auto-generated) | ✅ Metadata only |
| 15 | View: List | List Views in object-meta.xml | ✅ Metadata only |
| 16 | View: Detail | Record Page (Lightning, auto-generated) | ✅ Metadata only |

No additional code required for any feature in this proof.

