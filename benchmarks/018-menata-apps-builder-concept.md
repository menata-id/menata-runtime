# Menata Apps Builder — Page Concept (Authoring Layer, out of Runtime scope)

> Prompted directly by a question about where usability/UI design documentation lives, followed
> by a direct request to sketch page concepts for the Authoring Layer tool that would author
> Runtime Metadata by hand-editing today — CAP-X08's own note already states this plainly: *"No
> metadata-authoring UI exists anywhere in this prototype... Machines/Fields/Events are only ever
> created via seed SQL."*
>
> **This document is deliberately NOT a `capability-registry.md` entry.** That registry tracks
> **Runtime Layer** capabilities only — what `menata-runtime`'s interpreter itself can load and
> execute, gated by dual case evidence per the registry's own Rules. "Menata Apps Builder" is
> named in `002-architecture.md` §Authoring Layer and `README.md` §Architecture as one of several
> interchangeable **Authoring Layer** tools that *produce* Runtime Metadata — the runtime "never
> depends on how Runtime Metadata is created." A page concept for that tool is a different
> project's concern, parked here only because this repo is where the question was asked and where
> the underlying Runtime Metadata shape (`runtime-metadata-schema.md`) already lives.
>
> Status: v0.1 (concept only, no implementation) | Created: 2026-08-22

---

# Why these pages, and in this order

One Machine's Runtime Metadata (`runtime-metadata-schema.md` §Machine) is: **Fields**, **Events**
(transitions, each with a `condition`, `actions`, and an actor via Permissions), **Constraints**,
and **Views** (an array; one type is `form`, seven others are presentational). Two things fall out
of this shape directly:

1. **Fields are upstream of everything else.** A workflow's states are nothing but the *values*
   of one `value_list` Field (conventionally named Status); a Form or View's columns/filters
   reference Fields by id. Neither Workflow nor Form/View design has anything to configure until
   Fields exist. A **Machine & Field Designer** is therefore the true foundation — more
   foundational than any of the three pages below — and is not designed here because it wasn't
   asked for; it is named so the ordering below reads honestly rather than assuming Fields appear
   from nowhere.
2. **`form` is one `View` among eight `type`s, not a different concept** (`runtime-metadata-schema.md`
   §Views §View Types). A single "View Designer" covering all eight was the initial proposal;
   the user explicitly asked to keep Form Designer and (the other seven types') View Designer as
   two separate pages instead, so that decision is fixed here, not left as an open question.

Recommended build order: **Machine & Field Designer → Workflow Designer → Form Designer / View
Designer (peers, either order) → Permission/Application setup.** The three pages mocked below are
Workflow Designer, Form Designer, and View Designer — in that dependency order, assuming Fields
already exist on the Machine being edited.

---

# Page 1 — Workflow Designer

Edits: one Machine's Status `value_list` Field (its `values` = graph nodes) and its `events`
(`runtime-metadata-schema.md` §Machine, §Event Conditions, §Event Actions) — the graph edges.

- **Canvas** — a directed graph: State nodes, Event edges with arrowheads and labels. This is
  literally the write-side counterpart of the already-✅ `CAP-W05` `process_map` View
  (`benchmarks/013-overlay-compiler-proof.md`, T140–T143) — that capability renders this exact
  shape read-only from `value_list` Status + guarded Events today; a designer is the same
  rendering made editable.
- **State inspector** — name, initial/terminal, which Form is attached at this state.
- **Transition (Event) inspector** — event name, actor/role (Permissions), `condition`, `actions`
  (`set_field`, `notify`, `activate_next`, `aggregate_status` per §Event Actions), and requirement
  toggles (approval / task / evidence — mirroring BRD success criteria #7–9 in
  `benchmarks/011-metadata-workflow-orchestration-brd-benchmark.md`).
- **Validator** — flags unreachable states, dead-end states (no outgoing event), events with no
  actor assigned — a static, load-time-style check surfaced in-editor instead of only at runtime.
- **Simulate / Test Mode** — fire a hypothetical event against a hypothetical record and show
  guard evaluation (condition met, actor permitted, requirements satisfied) without touching real
  data.
- **Version comparison** — diff two versions of the same Machine's workflow shape. This has a real
  runtime hook already: `CAP-W07` (`change_policy`, effective-dated metadata evolution,
  implemented 2026-08-22) is exactly the mechanism this panel would be authoring against.

# Page 2 — Form Designer

Edits: one `type: form` View only (kept separate from Page 3 per explicit decision above).

- **Field palette** — the Machine's own Fields, grouped "on this form" vs "available," each
  showing its type.
- **Layout canvas** — Fields arranged into sections/rows; the canvas doubles as the live preview,
  since the goal is that what's designed here is what the real form renders — no separate preview
  engine, matching this runtime's own "Applications are interpreted, not generated" stance
  (`002-architecture.md` §Purpose).
- **Field inspector** — label override, required, default, computed expression, reference target.
- **State-conditional visibility** — per-Status editable/read-only/hidden for the selected field.
  This is the one place Form Designer and Workflow Designer are NOT independent: a field's
  editability is gated by the same Status values Page 1 defines, so this panel must read that
  Field's `values`, not maintain its own copy.

# Page 3 — View Designer

Edits: the other seven `View` `type`s (`list`, `detail`, `dashboard`, `calendar`, `timeline`,
`report`, `document` — `runtime-metadata-schema.md` §View Types) on one Machine. `form` is
excluded here by design, per the separation decided above.

- **Sidebar** — every View on this Machine, grouped by type, with a type-picker "New View" action.
- **Config panel**, shape switching per selected type — most representative examples:
  - `list` — column picker, filter builder (field/operator/value, including the `$current_user`
    sentinel per `CAP-V05`), default sort, manual-order toggle (`CAP-V14`).
  - `dashboard` — a list of sections, each independently naming its own source Machine
    (`CAP-V10` — cross-Machine composition is the actual point of this type).
  - `calendar`/`timeline` — `date_field` picker (`CAP-V07`).
  - `report` — source Machine, `group_field`, `sum_fields` (`CAP-V13`).
- **Live preview** — rendered with the same components the runtime actually uses, same rationale
  as Page 2.

---

# Mobile / responsive reach

The three pages do not share one answer to "does this work on a phone" — the split follows the
underlying UI paradigm, not a blanket "admin tools are desktop-only" assumption (checked directly
against Drupal, since a comparable class of tool — form/field design, Views, and ECA workflow
automation — already ships there):

- **Form Designer and View Designer are form-based UI** (field lists, column pickers, filter
  rows) — the same category as Drupal core's Field UI (Manage Form Display) and Views UI, both of
  which run inside Drupal's **Gin** admin theme. Gin is built explicitly to be usable from desktop,
  tablet, and phone, and is becoming Drupal core's default admin theme in 11.3. This is real,
  shipped precedent that this class of configuration screen is not inherently desktop-only — a
  3-panel desktop layout (palette + canvas + inspector) collapses to a single scrollable list plus
  a tap-to-open bottom sheet for the inspector, the same move Field UI makes with its per-row
  settings gear.
- **Workflow Designer's node/edge graph canvas is a different UI paradigm**, and does not have
  the same precedent. Drupal's ECA module ships two modellers: **BPMN.iO** (the `bpmn.js` canvas,
  drag-and-drop diagram editing — no mobile-usability claim found anywhere in its own
  documentation) and **ECA Classic Modeler** (`eca_cm`), which is explicitly **form-based, not
  canvas-based** — it "directly exposes the structure of an ECA configuration" through plain
  Drupal forms instead of a diagram. `eca_cm`'s own project page frames itself as a basic fallback
  ("better modelers are available," recommended only for "simple execution chains," with a stated
  accessibility gap), not as a comfortable primary experience — it does, however, prove that a
  workflow's structure CAN be exposed as a form/list when a canvas isn't viable.

Conclusion, applied to the mockups: Form Designer and View Designer get real mobile mockups
(field/column lists, bottom-sheet inspector, collapsed live-preview card). Workflow Designer gets
a **"Simple mode"** mobile mockup — states as expandable cards, transitions as rows underneath,
explicitly labeled as a reduced mode with a link back to the desktop canvas for anything beyond
trivial editing — the same honest scope `eca_cm` gives itself, not a claim of parity with the
graph canvas.

---

# Mockup

Six static mockups were produced 2026-08-22 as a Claude Design canvas — an internal-admin-tool
visual treatment (IBM Plex type, neutral surface, single blue accent), not a functioning
prototype: the three desktop pages above (1440×900 each) plus three phone-width companions
(390×844) per the mobile-reach split just described. Not attached to this repo (it's a private,
externally-hosted design exploration, not committed source); ask the session that produced it for
the link if needed.

---

# Status

Concept and static mockup only. No case-portfolio evidence is required or claimed — this is an
Authoring Layer artifact, not a Runtime Layer capability, so it is intentionally exempt from
`capability-registry.md`'s dual-evidence admission bar. Revisit if `menata-runtime` or a sibling
project ever actually undertakes building Menata Apps Builder; until then this is a parked design
reference, not a roadmap commitment.
