# Metadata Examples

Two cases documented here. Both run on the same runtime engine — zero code changes between them.

## How to read these files

Each case has two files:

| File | What it is |
|------|-----------|
| `*.menata` | Business Knowledge in Menata Language — the source of truth, written by a domain expert |
| `*.yaml` | Runtime Metadata — the machine-readable realization, inserted into PostgreSQL via seed SQL |

The runtime reads only the database. The `.menata` file is human-facing documentation; the `.yaml` is its structured counterpart that maps directly to the DB schema.

---

## Case 1 — Design Request

**Domain:** Creative services workflow  
**Application:** Design  
**Roles:** Requester, Designer  
**Seed:** `seeds/001_design_request.sql`

```
design-request.menata   Menata Language source
design-request.yaml     Runtime Metadata (DB realization)
```

**Workflow:** Requester submits → Designer accepts/rejects → starts work → completes  
**Notable constraint:** Attachment required only when Design Type = Banner 2:1 (conditional constraint)

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

```
leave-request.menata    Menata Language source
leave-request.yaml      Runtime Metadata (DB realization)
```

**Workflow:** Employee submits → Manager approves or rejects; Employee may cancel before approval  
**Notable:** Different application, different roles, different constraint set from Case 1 — no code change required

| Grammar | Count |
|---------|-------|
| Fields | 6 (user, value_list ×2, date ×2, rich_text) |
| Events | 4 (Submit, Approve, Reject, Cancel) |
| Constraints | 2 (reason required, start date future) |
| Permissions | 2 roles |
| Views | 4 (form, list ×2, detail) |

---

## What the two cases prove

Running both cases side by side on the same runtime demonstrates:

1. **Metadata-driven execution** — a new machine is live after `INSERT` + restart; no Go code touched.
2. **Role isolation** — `Employee` cannot trigger `Approve` (403); `Manager` cannot trigger `Submit`. Enforced by the Permission Guard reading metadata at runtime.
3. **Constraint portability** — the same constraint engine evaluates `required` and `after:today` against any machine's field IDs without field-specific code.
4. **Multi-application support** — Design (`app_design`) and HR (`app_hr`) coexist in the same workspace; the home page lists machines across both applications.
5. **View config drives UI** — the form shows only fields listed in the `form` view config; the list shows only columns listed in the `list` view config. Layout is data, not template logic.

---

## Adding a third case

1. Write the `.menata` source (optional but recommended).
2. Write the `.yaml` Runtime Metadata.
3. Translate to `seeds/00N_<name>.sql` following the same INSERT pattern.
4. `psql $DATABASE_URL -f seeds/00N_<name>.sql`
5. Restart the server.

The runtime picks it up with no other changes.
