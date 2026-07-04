# Menata Runtime — Camunda Prototype

> Metadata proof: design-request.yaml → BPMN + DMN + Camunda Forms

---

## Approach

Camunda is a **process-first** runtime engine built on open standards.

It uses three complementary metadata formats:

- **BPMN 2.0** — process flow (events, gateways, tasks)
- **DMN** — decision tables (business rules)
- **Camunda Forms** — form definitions (input views)

This prototype shows how Menata Runtime concepts map onto these open standards.

```text
design-request.yaml
        │
        ├── Events + Permissions    → design-request.bpmn   (BPMN 2.0 process)
        ├── Constraints             → design-request.dmn    (DMN decision table)
        └── View (form)             → design-request.form   (Camunda Form JSON)
```

---

## Runtime Metadata → Camunda Mapping

| Menata Runtime | Camunda |
|----------------|--------|
| Machine | Process Definition (BPMN) |
| Field | Form field (Camunda Form) |
| Event (user-triggered) | User Task (BPMN) |
| Event action: set_field (status) | Service Task: update variable |
| Event action: notify | Send Task / Message Event |
| Time-based Event | Timer Event (BPMN) |
| Constraint (required/format) | Form field validation |
| Conditional constraint | DMN Decision Table |
| Permission | Candidate Group on User Task |
| View (form) | Camunda Form (linked to User Task) |
| View (list) | Camunda Tasklist (built-in) |

---

## Files

```
camunda-config/
├── design-request.bpmn   ← BPMN 2.0: full process (submit → review → start → complete)
├── design-request.dmn    ← DMN: constraint decision table (attachment required if Banner)
└── design-request.form   ← Camunda Form: request submission form
```

---

## Metadata Coverage

**Score: ~80% (13/16 features)**

Camunda's unique strength: DMN covers **all 4 constraints as pure metadata** — including the hardest conditional case. This scores higher than Directus and Budibase despite the notification gap.

| # | Feature | File | Status |
|---|---------|------|--------|
| 1 | Machine definition | design-request.bpmn (Process) | ✅ Metadata only |
| 2 | All Fields | design-request.form | ✅ Metadata only |
| 3 | Status (process state as implicit status) | BPMN process variables | ✅ Metadata only |
| 4 | State machine enforcement | BPMN sequence flows + gateways | ✅ Metadata only |
| 5 | Event action: set status on transition | BPMN process variable update | ✅ Metadata only |
| 6 | Event action: notify Designer on Submit | design-request.bpmn (Service Task) | ❌ Cannot be done without code — Service Task needs connector worker (Java/JS) or Camunda 8 Cloud built-in connector |
| 7 | Event action: notify Requester on Complete | design-request.bpmn (Service Task) | ❌ Same as above |
| 8 | Constraint: Title required | design-request.form (validate.required) | ✅ Metadata only |
| 9 | Constraint: Description required | design-request.form (validate.required) | ✅ Metadata only |
| 10 | Constraint: Due Date after today | design-request.form (validate.min: now) | ✅ Metadata only |
| 11 | Constraint: Attachment if Design Type = Banner | design-request.dmn (decision table) | ✅ Metadata only |
| 12 | Permission: Requester role | design-request.bpmn (candidateGroups) | ✅ Metadata only |
| 13 | Permission: Designer role | design-request.bpmn (candidateGroups) | ✅ Metadata only |
| 14 | View: Form | design-request.form | ✅ Metadata only |
| 15 | View: List | Camunda Tasklist (built-in) | ✅ Metadata only |
| 16 | View: Detail | — | ❌ Camunda Tasklist shows active tasks, not a full record detail view for all states |

---

## Key Insight

Camunda proves that **open standards** (BPMN, DMN) can serve as Runtime Metadata.

Menata Events map naturally to BPMN.

Menata Constraints map naturally to DMN decision tables.

This opens a path for Menata Runtime to integrate with the broader BPMN/DMN ecosystem.
