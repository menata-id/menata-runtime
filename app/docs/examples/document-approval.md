# Document Approval — Complete Metadata Reference

Worked example, mirroring `prototype/go/docs/examples/approval-document.yaml`/`approval-step.yaml`
for `app/`'s own current state. Case 3 (`case-portfolio.md`) — the same use case walked through in
`app/web/static/ui-sample/document-approval.html`, `group-approval.html`, `group-approval-detail.html`.

This is the answer to "what metadata do I write to get the Document Approval + Group application
the mockups show" — every piece, in one place, instead of scattered across five seed files with no
single narrative connecting them.

## The whole picture in one table

| Piece | What it does | Where it's declared |
|---|---|---|
| `mch_approval_document`, `mch_approval_step` | The two Machines: a document with a status lifecycle, and one child record per approver's own decision | `seeds/004_approval.sql` |
| Sequential/Parallel approval, aggregate rollup | `CAP-A07`/`CAP-A08` — `activate_next`/`aggregate_status` event actions | `seeds/004_approval.sql` |
| Approver picker narrowed to a Group | `CAP-F23` — `fld_as_approver_user.options.restrict_to_group` | `seeds/038_group_approver_lab.sql`, `seeds/043_approver_type_toggle.sql` |
| Choosing a specific PERSON or a whole GROUP per Step, at submission | `CAP-F24` — `approver_type` toggle + dynamic Permission gate | `migrations/028_dynamic_actor_permission.sql`, `seeds/043_approver_type_toggle.sql` |
| The Group itself, its members, its role | `CAP-O07` — **not metadata at all**, created at runtime | `/admin/groups`, see below |
| Progress stepper (own page) | `CAP-V20` — `decision_stepper` View on the Document | `seeds/037_decision_stepper_lab.sql` |
| Signature placement (own page) | `CAP-V21`/`CAP-F22` — `coord_placement` View + PDF compositing | `seeds/035_pdf_signature_lab.sql`, `seeds/036_coord_placement_lab.sql` |
| Progress stepper AND signature position, both inline on the Step's own Detail page | `CAP-V20` Tier 2 — `children` on the Step's own `detail` View, two entries | `seeds/042_inline_view_composition.sql` |

The rest of this document walks through each row with the real declaration, not just a pointer to
the file.

## 1. The two Machines

```yaml
# Approval Document
machine:
  id: mch_approval_document
  name: Approval Document
  application: app_approval
  config:
    approval_mode_field: fld_ad_approval_mode   # Sequential | Parallel
    steps_machine: mch_approval_step            # CAP-X03: where its own Steps live
    steps_parent_field: fld_as_document          # which Field on the STEP points back here

  fields:
    - { id: fld_ad_title,         name: Title,         type: text,       required: true }
    - { id: fld_ad_document_type, name: Document Type, type: value_list, required: true, values: [SOP, Policy, Contract, Report, Other] }
    - { id: fld_ad_file,          name: File,          type: file,       required: true }
    - { id: fld_ad_submitted_by,  name: Submitted By,  type: user,       required: true }
    - { id: fld_ad_approval_mode, name: Approval Mode, type: value_list, required: true, values: [Sequential, Parallel] }
    - { id: fld_ad_status,        name: Status,        type: value_list, values: [Draft, In Review, Approved, Rejected, Withdrawn] }

  events:
    - id: evt_ad_submit
      condition: { field: fld_ad_status, operator: equals, value: Draft }
      actions:
        - set_field: { field: fld_ad_status, value: In Review }
        - notify: { role: Approver }
    - id: evt_ad_approve   # fired by System, via CAP-A08's aggregate_status — never a user button
      condition: { field: fld_ad_status, operator: equals, value: In Review }
      actions:
        - set_field: { field: fld_ad_status, value: Approved }
        - notify: { recipient_field: fld_ad_submitted_by, role: Submitter }
    - id: evt_ad_reject     # same, any sibling Step rejecting fires this immediately
      actions: [ ... mirrors evt_ad_approve ]

  permissions:
    - { role: Submitter, events: [evt_ad_submit, evt_ad_withdraw] }
    - { role: System,    events: [evt_ad_approve, evt_ad_reject] }
```

```yaml
# Approval Step — one child record per approver's own decision
machine:
  id: mch_approval_step
  name: Approval Step
  application: app_approval

  fields:
    - { id: fld_as_document, name: Document, type: reference, required: true, target_machine: mch_approval_document }
    - { id: fld_as_approver, name: Approver, type: user, required: false }  # legacy, see §3
    - { id: fld_as_approver_type,  name: Approver Type,  type: value_list, values: [User, Group] }
    - { id: fld_as_approver_user,  name: Approver User,  type: user,
        options: { restrict_to_group: "Document Approvers" } }             # CAP-F23, carried onto the new field too
    - { id: fld_as_approver_group, name: Approver Group, type: group }     # CAP-F24's new field type
    - { id: fld_as_sequence, name: Sequence, type: number,     required: true }
    - { id: fld_as_decision, name: Decision, type: value_list, values: [Pending, Approved, Rejected] }

  events:
    - id: evt_as_approve
      condition: { field: fld_as_decision, operator: equals, value: Pending }
      actions:
        - set_field: { field: fld_as_decision, value: Approved }
        - set_field: { field: fld_as_decided_at, value: now }
        - notify: { role: Submitter }
        - activate_next: { mode_field: fld_ad_approval_mode }           # CAP-A07
        - aggregate_status:                                              # CAP-A08
            parent_field: fld_as_document
            parent_event_if_all_approved: evt_ad_approve
            parent_event_if_any_rejected: evt_ad_reject

  permissions:
    - { role: Approver, events: [evt_as_approve, evt_as_reject] }  # dynamic actor gate — see §3
```

Real source: `app/seeds/004_approval.sql` (the original Machines) + `app/seeds/043_approver_type_toggle.sql`
(the toggle fields, §3 below). Full field/constraint/view lists there — this section keeps only
what matters for the composed app the mockups show.

## 2. The Group tie-in — this is NOT metadata

This is the one piece of "how do I get this exact application" that a `.menata`/YAML file cannot
declare, and it's worth being explicit about why, since it's the part most likely to be looked for
in the wrong place.

`CAP-O07` (Groups) is deliberately **runtime data**, created through `/admin/groups`, not seeded:
a Group has no stable, business-authored id the way a Field or Event does — it only ever gets a
database-generated UUID, and can be created, renamed, or have its membership changed at any time
after metadata has already loaded, by a workspace Admin, without touching the metadata layer at
all. Declaring one in YAML/SQL the way a Field is declared would be a category error — see
`capability-registry.md`'s own `CAP-F23` row for the fuller reasoning (`FieldOptions.
RestrictToGroup` names a Group **by name**, resolved at request time, never validated at load
time the way `target_machine` is).

**What you actually do**, as a workspace Admin, to get the exact group tie-in this app's own
mockups (`groups.html`, `group-detail.html`) show:

1. `POST /{ws}/admin/groups` with `name=Document Approvers` — creates the Group.
2. `POST /{ws}/admin/groups/{id}/members` — add whichever users should be candidate approvers.
3. `POST /{ws}/admin/groups/{id}/roles` with `app_role_app_approval=Approver` — **optional**, only
   needed if you want Group membership itself to grant the Approver role (CAP-O07's own union
   semantics: direct assignment ∪ every Group's own grant); a member can also just already hold
   Approver directly, the way `bob@example.com` does in this app's own seeded accounts
   (`seeds/007_authentication.sql`).

Once the Group exists with that exact name, `fld_as_approver_user`'s `restrict_to_group`
(declared in metadata, §1 above) picks it up automatically — no reload needed,
`GroupStore.GetByName` resolves by name at request time. If the name doesn't resolve to a real
Group, the picker degrades gracefully to its unrestricted candidate list (still narrowed by role)
— this is a UX narrowing, never the actual authorization boundary; the Permission's own role
check (and, for a Step using the toggle, §3's own dynamic actor gate) is.

## 3. A specific person, OR a whole Group, chosen per Step (`CAP-F24`)

`§1`'s `fld_as_approver` (still there, `required: false` now) is the ORIGINAL, static approver
Field — one Machine-wide picker, always a specific person. `document-submit.html`'s own mockup
shows something more flexible: each approval step picks, when it's created, whether its own
approver is one named person or an entire Group's membership. That choice is `CAP-F24` — a
three-Field pattern plus a Permission-level gate that reads whichever of the two candidate Fields
the toggle selected. Unlike Field/View config, a Permission's own gate has no `.menata`/YAML sugar
today — `owner_field` doesn't either — so the real declaration is plain SQL, directly against the
three columns `migrations/028_dynamic_actor_permission.sql` adds to `permissions`:

```sql
-- migrations/028_dynamic_actor_permission.sql adds the three columns;
-- seeds/043_approver_type_toggle.sql wires them onto the existing Permission row
UPDATE permissions SET
    actor_type_field  = 'fld_as_approver_type',
    actor_user_field  = 'fld_as_approver_user',
    actor_group_field = 'fld_as_approver_group'
    WHERE id = 'perm_as_approver';
```

At Approve/Reject time, `internal/permission.ResolveActorGate` reads THIS record's own
`fld_as_approver_type` value: `User` → the acting identity must equal `fld_as_approver_user`
(an ordinary person-gate, CAP-P02's own `owner_field` shape); `Group` → the acting identity must
be a MEMBER of the Group named by `fld_as_approver_group` (`GroupStore.MemberIDs`) — never named
directly on the record, so adding or removing a Group member changes who can decide this Step with
no metadata change and no reload. A Step that never sets `fld_as_approver_type` at all (every
Step created before this feature existed, or via a raw POST that still only sets the original
`fld_as_approver`) falls back to that legacy field automatically — the same Permission row
declares both `owner_field: fld_as_approver` and the three `dynamic_actor` columns at once, and
the dynamic gate only takes over once a record actually opts in.

`fld_as_approver` was deliberately dropped from `vw_as_form`'s own displayed fields (the toggle
fully replaces it there) but stays a real, resolvable Field — nothing about the legacy fallback
above requires it to still be on any form.

## 4. Progress and signature position — each a full page, and both inline together

`CAP-V20`'s `decision_stepper` View is declared once, on the Document:

```yaml
views:
  - id: vw_ad_progress
    name: Decision Progress
    type: decision_stepper
    config: { sequence_field: fld_as_sequence, decision_field: fld_as_decision }
```
```sql
-- seeds/037_decision_stepper_lab.sql
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ad_progress', 'mch_approval_document', 'Decision Progress', 'decision_stepper', 3,
     '{"decision_stepper":{"sequence_field":"fld_as_sequence","decision_field":"fld_as_decision"}}');
```

`CAP-V21`'s `coord_placement` View is declared once too, on the Step itself (same Machine this
time — it previews the Document's own file, but positions a pin using the STEP's own fields):

```yaml
views:
  - id: vw_as_place
    name: Set Signature Position
    type: coord_placement
    config:
      reference_field: fld_as_document     # points at the Document, for its own File
      preview_field: fld_ad_file           # the Field on that Document holding the PDF
      page_field: fld_as_signature_page    # these three are on THIS (Step) record
      x_field: fld_as_signature_x
      y_field: fld_as_signature_y
```
```sql
-- seeds/036_coord_placement_lab.sql
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_as_place', 'mch_approval_step', 'Set Signature Position', 'coord_placement', 3,
     '{"coord_placement":{"reference_field":"fld_as_document","preview_field":"fld_ad_file","page_field":"fld_as_signature_page","x_field":"fld_as_signature_x","y_field":"fld_as_signature_y"}}');
```

Each of those alone gives you a full page (`GET /{ws}/mch_approval_document/{id}/progress`,
`GET /{ws}/mch_approval_step/{id}/place`). Neither, by itself, puts its content on the Step's own
Detail page (where the real Approve/Reject buttons live) — `document-approval.html`'s own mockup
shows all three (progress, signature position, decision) on one screen, and closing that gap needed
a third, generic declaration: `children` (`CAP-V20` Tier 2, `guides/writing-runtime-metadata.md`'s
own "Komposisi View" section has the full mechanism writeup). On the Step's own `detail` View,
naming BOTH:

```yaml
views:
  - id: vw_as_detail
    name: Step Detail
    type: detail
    children:
      - view: vw_ad_progress   # a DIFFERENT Machine's own View (Approval Document)
      - view: vw_as_place      # the SAME Machine's own View (Approval Step itself)
```
```sql
-- seeds/042_inline_view_composition.sql
UPDATE views SET config = config || '{"children":[{"view":"vw_ad_progress"},{"view":"vw_as_place"}]}'
    WHERE id = 'vw_as_detail';
```

`children` is generic — it never says "decision stepper" or "coord placement" itself. What each
named View actually renders as is resolved from **its own** declared `type` at render time
(`internal/handler/embed.go`'s `renderChildView`, one `case` per Type), not from this key's name.
This is also why the same array can mix a cross-Machine reference (`vw_ad_progress`, owned by
Approval Document) and a same-Machine one (`vw_as_place`, owned by Approval Step itself) with no
special-casing either way — a Child is just a View id; what Machine owns it is irrelevant to
`children` itself, the same way `steps_machine`/`target_machine` already cross Machines freely
elsewhere in this Case.

## 5. Reproducing this app from scratch

In order, against a fresh database (`make migrate-up` first — this now includes
`migrations/028_dynamic_actor_permission.sql`, §3's own schema change):

1. `seeds/004_approval.sql` — the two Machines, §1 above.
2. `seeds/007_authentication.sql` — seeded accounts (Alice/Submitter, Bob & Carol/Approver).
3. `seeds/035_pdf_signature_lab.sql` + `seeds/036_coord_placement_lab.sql` — signature fields and
   the `coord_placement` View, §4 above.
4. `seeds/037_decision_stepper_lab.sql` — §4's other full-page stepper.
5. `seeds/038_group_approver_lab.sql` — §2's `restrict_to_group` declaration.
6. `seeds/042_inline_view_composition.sql` — §4's inline composition of both.
7. `seeds/043_approver_type_toggle.sql` — §3's User/Group toggle and its dynamic Permission gate.
8. Through the running app, as a workspace Admin: create the "Document Approvers" Group, add
   members (§2) — this step has no seed file, on purpose.

Every metadata-only step (1–7) takes effect on the next server start, or immediately via
`POST /{ws}/admin/reload` (`CAP-X04`) with no restart at all — verified live, both ways, building
this exact reference (`capability-registry.md`'s `CAP-V20` and `CAP-F24` rows both have the dated
proof).
