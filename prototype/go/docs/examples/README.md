# Metadata Examples

Four cases documented here. Cases 1 and 2 run fully on the current prototype. Cases 3 and 4 are boundary tests — showing what Business Knowledge looks like for complex domains, and where the current runtime needs to be extended. Case selection follows `runtime/case-portfolio.md`.

## How to read these files

Each case has two files:

| File | What it is |
|------|-----------|
| `*.menata` | Business Knowledge in Menata Language — written by a domain expert, technology-independent |
| `*.yaml` | Runtime Metadata — the machine-readable realization, maps directly to the DB schema |

The `.yaml` files for Case 3 include inline annotations: `[SUPPORTED]`, `[NOT YET]`, `[PARTIAL]`.

---

## Case 1 — Design Request

**Domain:** Creative services workflow  
**Application:** Design  
**Roles:** Requester, Designer  
**Seed:** `seeds/001_design_request.sql`  
**Status:** ✅ Fully supported

```
design-request.menata   Menata Language source
design-request.yaml     Runtime Metadata (DB realization)
```

**Workflow:** Requester submits → Designer accepts/rejects → starts work → completes  
**Notable:** Conditional constraint — Attachment required only when Design Type = Banner 2:1

| Grammar | Count |
|---------|-------|
| Fields | 7 (user, value_list ×2, date, text, rich_text, file) |
| Events | 5 (Submit, Accept, Reject, Start, Complete) |
| Constraints | 4 (2 required, 1 date future, 1 conditional) |
| Permissions | 2 roles |
| Views | 4 (form, list ×2, detail) |

---

## Case 2 — Leave Request

**Domain:** HR — employee leave approval  
**Application:** HR  
**Roles:** Employee, Manager  
**Seed:** `seeds/002_leave_request.sql`  
**Status:** ✅ Fully supported

```
leave-request.menata    Menata Language source
leave-request.yaml      Runtime Metadata (DB realization)
```

**Workflow:** Employee submits → Manager approves or rejects; Employee may cancel before approval  
**Notable:** Different application, different roles from Case 1 — no code change required

| Grammar | Count |
|---------|-------|
| Fields | 6 (user, value_list ×2, date ×2, rich_text) |
| Events | 4 (Submit, Approve, Reject, Cancel) |
| Constraints | 2 (reason required, start date future) |
| Permissions | 2 roles |
| Views | 4 (form, list ×2, detail) |

---

## Case 3 — Document Approval System

**Domain:** Multi-approver document approval with sequential or parallel mode  
**Application:** Approval  
**Roles:** Submitter, Approver, System  
**Seed:** — (not yet, pending runtime extensions)  
**Status:** ⚠️ Partially supported — see gap analysis below

```
approval-document.menata    Menata Language source — Approval Document
approval-document.yaml      Runtime Metadata + inline gap annotations
approval-step.menata        Menata Language source — Approval Step
approval-step.yaml          Runtime Metadata + inline gap annotations
```

**Workflow:**
```
Submitter creates Document → sets approvers + mode (Sequential | Parallel)
    ↓
Submit → Status: In Review → notify Approvers
    ↓
Each Approver acts on their Approval Step
    │
    ├── Sequential: Step 2 activates only after Step 1 Approved
    └── Parallel:   All Steps active simultaneously
    ↓
All Steps Approved → Document: Approved → notify Submitter
Any Step Rejected  → Document: Rejected → notify Submitter
```

**Two machines, linked by reference:**

| Machine | Grammar | Notes |
|---------|---------|-------|
| Approval Document | 6 fields, 4 events, 3 constraints, 2 roles, 5 views | Parent |
| Approval Step | 6 fields, 2 events, 1 conditional constraint, 1 role, 2 views | Child — references Document |

---

### Gap Analysis

#### What works now

| Feature | Status | Notes |
|---------|--------|-------|
| Approval Document as standalone machine | ✅ | Fields, form, list, detail all render |
| Approval Step as standalone machine | ✅ | Same |
| Conditional constraint (Notes required if Rejected) | ✅ | operator: equals in condition |
| set_field actions (Decision, Status) | ✅ | Works on both machines independently |
| notify action | ✅ | Static role string |
| evt_ad_withdraw | ✅ | Simple set_field |

#### What does not work yet

| Feature | Gap | What's needed |
|---------|-----|---------------|
| Step links to Document | `type: reference` field not implemented | New field type in model, loader, store, handler, UI |
| Steps shown on Document detail page | Cross-machine query (list steps by parent) | `store.ListByParent(machineID, parentFieldID, parentRecordID)` |
| "Pending My Approval" filtered list | Cross-machine filter by current user | Record-level query + user context in store |
| Sequential activation | `activate_next` action type doesn't exist | New executor action: find sibling step with sequence+1, set it active |
| All approved → Document approved | `aggregate_status` action type doesn't exist | New executor action: check all siblings, trigger parent event if resolved |
| System-triggered events | No internal event trigger mechanism | Internal event bus or post-action hook in executor |
| `value: now` in set_field | Dynamic value expressions not supported | Value resolver: `now`, `today`, `current_user` |
| Record-level permission (only assigned Approver) | Permission checks role string only | Field-level ownership check: `fld_as_approver = current_user` |
| Approval mode drives behavior | Machine-level config not in schema | New `config` block on Machine in Runtime Metadata schema |

---

### What Case 3 reveals about the runtime roadmap

The gaps above map to concrete runtime extensions, in priority order:

**P1 — Reference fields** (blocks everything else in Case 3)
- Schema: add `type: reference` + `target_machine` to field definition
- Loader: load reference config from `options` JSONB
- Store: `ListByParent` query
- Handler: Detail page renders child records as sub-list
- UI: reference field renders as link, not free text input

**P2 — Dynamic value expressions in set_field**
- Executor: value resolver for `now`, `today`, `current_user`
- Enables: `Decided At` stamping on approve/reject

**P3 — New action types**
- `activate_next` — sequential approval step activation
- `aggregate_status` — parent status rollup when all/any children resolve

**P4 — Machine-level config**
- Schema: `config` block on Machine (approval_mode_field, steps_machine, steps_parent_field)
- Loader: load and expose machine config to executor

**P5 — Record-level permissions**
- Permission guard: check `field = current_user` in addition to role string
- Enables: only the assigned Approver can act on their Step

**P6 — Internal event triggering**
- Executor: fire an event on a record without HTTP request (for system-triggered events)
- Enables: aggregate_status triggering parent Approve/Reject automatically

---

## Case 4 — Maintenance Reminder

**Domain:** Facility — recurring equipment maintenance with overdue escalation
**Application:** Facility
**Roles:** Technician, Supervisor
**Seed:** — (pending time-driven event support)
**Status:** ⚠️ Documented boundary test — first portfolio-driven case (targets declared before writing, see `runtime/case-portfolio.md`)

```
maintenance-reminder.menata    Menata Language source
maintenance-reminder.yaml      Runtime Metadata + inline gap annotations
```

**Workflow:**
```
Task scheduled with Frequency (Daily | Weekly | Monthly) and Next Due Date
    ↓
Every Day 07:00: Next Due Date = Today → Status: Due → notify Assignee
    ↓
Every Day 07:00: still Due past date → Status: Overdue → notify Supervisor (escalation)
    ↓
Complete → stamp Last Completed = today → advance Next Due Date by Frequency → back to Scheduled
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Time-driven events (CAP-E02) | confirmed gap — no `trigger` block in schema |
| Event conditions (CAP-A09) | confirmed gap — including compound AND conditions |
| Environment data `today` (CAP-A02) | confirmed gap |
| Dynamic notify recipient (CAP-A04) | confirmed gap — escalation needs record's Supervisor |
| Date arithmetic | **[UNTARGETED FINDING]** → registered as CAP-A11 |
| Declarative view filter (Due Today list) | **[UNTARGETED FINDING]** → registered as CAP-V09 |

---

## What the cases prove together

| Capability | Case 1 | Case 2 | Case 3 |
|------------|--------|--------|--------|
| Metadata-driven single machine | ✅ | ✅ | ✅ |
| Multiple applications in one workspace | ✅ | ✅ | ✅ |
| Conditional constraints | ✅ | — | ✅ (partial) |
| Role-based event permissions | ✅ | ✅ | ✅ (partial) |
| Cross-machine references | — | — | ⚠️ needs P1 |
| Sequential workflow logic | — | — | ⚠️ needs P3 |
| Parent-child status aggregation | — | — | ⚠️ needs P3 |
| Record-level ownership | — | — | ⚠️ needs P5 |
| System-triggered events | — | — | ⚠️ needs P6 |

Cases 1 and 2 validate the metadata-driven foundation.  
Case 3 defines the next layer of runtime capability needed to handle real-world workflow complexity.

---

## Adding a new case

1. Write the `.menata` source.
2. Write the `.yaml` Runtime Metadata (annotate `[NOT YET]` where applicable).
3. Translate to `seeds/00N_<name>.sql` for the parts that are supported.
4. `psql $DATABASE_URL -f seeds/00N_<name>.sql`
5. Restart the server.
