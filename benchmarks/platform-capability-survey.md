# Platform Capability Survey

> Artifact of the Capability Roadmap — Study 2 deliverable.
>
> Consolidates what the 6 platform prototypes documented, and answers:
> **which capabilities do the platforms provide natively that Menata has not yet named?**
>
> Status: v0.1 | Created: 2026-07-04

Sources: `prototype/{salesforce,frappe,drupal,camunda,directus,budibase}/README.md` + `drupal/docs/drupal-mapping.md` — the metadata proofs scored against `design-request.menata` (16 features).

---

# Consolidated Capability Matrix

Legend: ✅ native metadata · ⚠️ partial/undemonstrated · ❌ requires code · `—` not applicable.
Rightmost column: Menata Go prototype status (from `../capability-registry.md`).

| Capability | Salesforce | Frappe | Drupal | Camunda | Directus | Budibase | **Menata Go** |
|-----------|-----------|--------|--------|---------|----------|----------|---------------|
| Machine/object definition | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Typed fields | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ (6 types fall back to text) |
| Reference field (link to other machine) | ✅ Lookup | ✅ Link | ✅ entity_ref | — | ✅ relation | ✅ Link | ❌ CAP-F13 |
| User field as real identity link | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ CAP-F05 (free text) |
| **State machine enforcement** | ✅ Flow | ✅ Workflow | ✅ Workflow | ✅ BPMN | ❌ | ❌ | ❌ CAP-E06 |
| Event → set status | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ CAP-A01 |
| Event → notify (real delivery) | ✅ Email Alert | ✅ Email+In-App | ✅ ECA email | ❌ connector | ✅ mail op | ✅ SEND_EMAIL | ⚠️ CAP-A03 (slog only) |
| Required constraint | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ CAP-C01 |
| Date constraint (after today) | ✅ formula | ❌ | ❌ | ✅ validate.min | ⚠️ | ❌ | ✅ CAP-C02 |
| Conditional constraint | ✅ formula | ❌ | ❌ | ✅ **DMN** | ⚠️ | ❌ | ✅ CAP-C04 |
| Role-based permissions | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ CAP-P01 |
| **CRUD-level permissions (read/create/edit per role)** | ✅ | ✅ DocPerm | ✅ | — | ✅ per action | ✅ | ❌ *(events only)* |
| **Field-level visibility** | ✅ field perms | ✅ permlevel | ⚠️ | — | ✅ | ⚠️ | ❌ |
| Form / List / Detail views | ✅ | ✅ auto | ✅ | ⚠️ | ✅ auto | ✅ | ✅ V01–V03 |
| **List search & filter** | ✅ | ✅ | ✅ Views | ✅ | ✅ | ✅ | ❌ |
| Record edit form | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ CAP-R02 |
| **Auto REST API per machine** | ✅ | ✅ | ⚠️ module | ✅ | ✅ REST+GraphQL | ✅ | ❌ |
| Audit trail / revision history (visible) | ✅ field history | ✅ free | ✅ revisions | ✅ | ⚠️ | ⚠️ | ⚠️ CAP-R04 (DB only, no UI) |
| **Data import/export (CSV)** | ✅ | ✅ free | ⚠️ | — | ✅ | ✅ | ❌ |
| **Metadata package export/import** | ✅ Metadata API | ✅ JSON | ✅ config sync | ✅ BPMN/DMN files | ✅ snapshot | ✅ single app.json | ❌ *(hand-written SQL seeds)* |
| Timer / scheduled events | ✅ | ✅ | ✅ | ✅ Timer Event | ✅ Flow cron | ✅ CRON | ❌ CAP-E02 |
| **Computed / formula fields** | ✅ | ✅ | ⚠️ | — | ⚠️ | ✅ | ❌ |
| **Field default values** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ *(status only, hardcoded first value)* |

Bold rows = capabilities present in most/all platforms but **not yet named in the Menata registry** before this survey.

---

# Findings

## 1. Table stakes — every platform has these, Menata had not named them

Nine capabilities are so universal that platforms provide them for free the moment a machine/table is defined. A runtime that lacks them will feel broken to anyone coming from any of these platforms:

| New capability | Universal because | Registered as |
|----------------|-------------------|---------------|
| CRUD-level permissions (who may read/create/edit — not just trigger events) | 6/6 platforms | CAP-P05 |
| Field-level visibility ("Salary visible only to HR" — already a spec 004 example!) | 4/6 | CAP-P06 |
| List search & filter | 6/6 | CAP-V08 |
| Data import/export | 5/6 | CAP-R06 |
| Auto REST API per machine | 5/6 | CAP-X07 |
| Metadata package export/import | 6/6 — every platform treats the app definition as a portable artifact | CAP-X08 |
| Computed / formula fields | 4/6 | CAP-F14 |
| Field default values (beyond status) | 6/6 | CAP-F15 |
| Notification delivery channels (email / in-app) | 6/6 | CAP-A10 |

## 2. State machine enforcement is the differentiator

The two lowest-scoring platforms (Directus 70%, Budibase 65%) lost points **primarily because they cannot enforce state transitions**. The Go prototype currently has the same flaw (CAP-E06). This independently confirms Study 1's headline finding and its Prio 2 ranking: state guards separate real workflow runtimes from CRUD generators.

## 3. Frappe's DocType is the closest architectural model

Define a DocType → form, list, detail, REST API, permissions, audit trail, import/export all appear with zero extra configuration. This is exactly the "Infer Before Configure" principle. The gap list above is, in effect, **the distance between Menata's Machine and Frappe's DocType**.

## 4. DMN is the constraint engine's growth path

Camunda covers all 4 benchmark constraints — including the hardest conditional one — as pure metadata via DMN decision tables. When Menata's constraint operators grow past simple comparisons (CAP-C05/C07/C08), DMN's decision-table model is the proven structure to borrow.

## 5. Metadata portability is an ecosystem requirement, not a feature

Every platform treats the application definition as a **deployable, versionable artifact** (Salesforce Metadata API, Budibase app.json, Drupal config sync, Directus snapshot). Menata's current hand-written SQL seeds are authoring-hostile. CAP-X08 is what makes "One Business Knowledge, many runtimes" operational rather than aspirational.

---

# Registry Impact

9 new capabilities registered (v0.2): CAP-F14, CAP-F15, CAP-A10, CAP-P05, CAP-P06, CAP-V08, CAP-R06, CAP-X07, CAP-X08 — see `../capability-registry.md`.

Reinforced priorities: CAP-E06 (finding 2), CAP-F13/F05 (universal reference/user links), CAP-A03 (all platforms deliver notifications for real).

---

# Maintenance

Re-run this survey when a new platform prototype is added, or when a platform proof is upgraded (e.g. Directus constraints demonstrated). Update the matrix and register any newly surfaced universal capability.
