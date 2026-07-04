# Case 10 — Organization Composite

> Study 7 deliverable (`runtime/capability-roadmap.md`).
>
> A thought-experiment case: **one organization runs everything at once.**
> Unlike Cases 1–9 (one machine or one vertical), this case composes all prior
> cases plus Study 5 (Portal GA patterns) and Study 6 (accounting) into a single
> workspace — to surface capabilities that **no single case can reveal**.
>
> Findings that only appear at composition are flagged `[COMPOSITION FINDING]`.

---

## Scenario

PT Maju Bersama (a mid-size distributor, 12 branches) runs its whole back office on one Menata workspace:

| Application | Machines (from) |
|-------------|----------------|
| Design | Design Request (Case 1) |
| HR | Leave Request (Case 2), Employee* |
| Approval | Approval Document + Step (Case 3) |
| Facility | Maintenance Task (Case 4), Equipment* |
| Warehouse | Stock Movement (Case 5) |
| Finance | Petty Cash (Case 6), Journal Entry + COA (Case 9), Payment Confirmation (Case 8) |
| Service | Customer Complaint (Case 7), Customer* |
| Safety | Portal-GA-style: Observation → PICA (canonical action machine) |

\* master data machines — see Finding 2.

One employee, Ibu Sari (branch Bekasi), in a single morning: approves a leave request (HR), completes a maintenance task (Facility), is the second approver on a policy document (Approval), and gets notified that her PICA action needs an AAR (Safety).

---

## Declared targets

Composition of already-registered capabilities: CAP-F13 (cross-machine references), CAP-I01–I05 (integration), CAP-X09 (org scoping), CAP-V10 (composed dashboard), CAP-P05 (CRUD permissions).

**Hypothesis to test:** new capabilities emerge that no single case surfaced.

---

## Findings

### Finding 1 — Shared identity & workspace role registry `[COMPOSITION FINDING]`

Ibu Sari is `Employee` in HR, `Technician` in Facility, `Approver` in Approval, `PIC Cabang` in Safety. Today's model has **no place that says who a user is across applications** — permissions are role *strings* per machine, and the prototype's role cookie is workspace-blind.

Worse: the string `Manager` appears in both HR (approves leave) and Design (hypothetically) with *different meanings*. Nothing namespaces roles or maps users to them.

→ **CAP-O01** — Workspace identity & role registry: users, role definitions (namespaced per application or workspace-shared), user→role assignments as metadata. Spec 005 already says roles are organizational responsibility, "not individual users" — the user→role *mapping* is the missing metadata.

### Finding 2 — Shared master data (canonical machines) `[COMPOSITION FINDING]`

Employee is referenced by Leave Request (who), Maintenance (assignee), Journal Entry (created-by), Complaint (handler). Customer is referenced by Complaint and Payment. Equipment by Maintenance and Stock.

CAP-F13 gives the *mechanics* of reference, but composition raises governance questions the mechanics can't answer: which application **owns** Employee? May Finance reference a machine inside HR? What happens to Leave Requests when an Employee is deactivated?

Portal GA answers this with Data Mesh ownership + a canonical machine mandate (PICA). DDD names it *shared kernel*.

→ **CAP-O02** — Master data designation: a machine can be declared `master` (workspace-visible, cross-application referenceable, ownership declared, deactivation semantics defined). Hierarchy stays Workspace→Application→Machine; `master` is a *flag plus rules*, not a new level.

### Finding 3 — Navigation & application home `[COMPOSITION FINDING — spec predicted it]`

The prototype home lists all machines flat. With 15+ machines across 8 applications, Ibu Sari needs: application grouping, role-aware menus (she never sees Finance), favorites/recents.

Notably `runtime/004-runtime-metada.md` **already lists Navigation in the metadata hierarchy** — the spec predicted this; no case ever exercised it.

→ **CAP-O03** — Navigation metadata: menu structure per application, role-visibility, workspace home composition.

### Finding 4 — Global search `[COMPOSITION FINDING]`

"Where is PO-2026-0042?" — could be a Stock Movement, a Journal Entry, or a Payment. Per-machine list search (CAP-V08) cannot answer a cross-machine question.

→ **CAP-O04** — Workspace-wide search across machines, permission-trimmed (depends on CAP-P05).

### Finding 5 — Unified notification inbox `[COMPOSITION FINDING]`

Ibu Sari receives notifications from 4 applications in one morning. Per-machine `notify` (CAP-A03/A10) with no inbox means 4 unrelated email streams. Portal GA solved this with a message dispatcher + in-app inbox + per-user preferences.

→ **CAP-O05** — Unified notification center: one inbox, per-user channel preferences, digest grouping. Extends CAP-A10 from delivery mechanics to workspace service.

### Finding 6 — Business calendar `[COMPOSITION FINDING]`

Leave duration must skip company holidays. Maintenance due dates must not land on non-working days. Complaint SLA counts working hours. Three different applications need the **same** answer to "is Thursday a working day?"

Spec 001 even lists `Holiday` as an example Object — but as a mere machine it would be application-local. Composition shows it must be a workspace service consumed by date arithmetic (CAP-A11) and SLA timers (CAP-E02).

→ **CAP-O06** — Business calendar: holidays + working-day rules as workspace metadata, consumable by date arithmetic, scheduling, and SLA evaluation.

### Non-findings (composition confirmed existing entries suffice)

- Org-wide dashboard → CAP-V10 already covers it (sections across applications).
- Cross-application audit trail → CAP-R04 extended to workspace scope, no new entry.
- Cross-application workflows (Complaint → PICA → Journal Entry for refund) → CAP-I01 + CAP-A06 compose correctly.

---

## Assessment — does Workspace/Application/Machine suffice?

**The hierarchy stands.** No fourth level or shared-kernel *structure* is needed. What composition demands is a new metadata *residence*: **workspace services** — concerns that belong to the workspace itself and to no single application:

```text
Workspace
    ├── identity & roles        (CAP-O01)
    ├── master data designations (CAP-O02)
    ├── navigation               (CAP-O03)
    ├── search                   (CAP-O04)
    ├── notification center      (CAP-O05)
    ├── business calendar        (CAP-O06)
    └── Applications
            └── Machines (unchanged)
```

This validates `runtime/004`'s claim that "Workspace owns … shared resources" — Case 10 makes that clause concrete for the first time.

**Hypothesis confirmed:** 6 capabilities emerged that none of Cases 1–9 could reveal alone.
