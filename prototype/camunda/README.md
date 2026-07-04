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

## Key Insight

Camunda proves that **open standards** (BPMN, DMN) can serve as Runtime Metadata.

Menata Events map naturally to BPMN.

Menata Constraints map naturally to DMN decision tables.

This opens a path for Menata Runtime to integrate with the broader BPMN/DMN ecosystem.
