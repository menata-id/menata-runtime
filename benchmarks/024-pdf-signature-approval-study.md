# Study 32 — Document Approval: PDF Signature Placement

> Page-by-page UI design + capability gap analysis for a real, immediate requirement on Case 3
> (Document Approval): the approved artifact must be an actual PDF, and each approver's own
> signature **image** must be stamped onto a position on that PDF fixed *before* submission —
> not merely a status field changing to "Approved". Requested directly by the owner, 2026-08-23,
> alongside a request to add the pages to the existing Menata Apps Builder design canvas
> (`benchmarks/021-design-system-prototype-plan.md`'s Study 29/30/31 artifact).

---

# 1. The requirement, as given

Restated from the owner's own request (Bahasa Indonesia, kept close to source so nothing is
lost in translation):

- The document under approval is a real PDF file.
- A real signature **image** is composited onto the PDF, at a location decided **in advance**,
  per document — not a generic "signed" stamp applied anywhere convenient.
- Before any approval happens, for the specific document being submitted, someone must decide,
  **per approver**, exactly where on the PDF that approver's signature will land.
- Approvers are not typed in one at a time — each belongs to a **work group**, and the group is
  the thing that supplies the candidate approvers for a document. The owner named this as
  "currently being developed under CAP-O07" — **that is not accurate against the registry as it
  stands today**: `capability-registry.md`'s CAP-O07 row is ❌, explicitly "deliberately
  deferred... not worth building at this runtime's current scale," last touched by a design note
  on 2026-08-23 (the CAP-X09 closure) that leaves it exactly where it was. Nothing currently in
  flight builds it. This study's own approver-from-group screens are real, independent pressure
  toward building CAP-O07 — recorded as such below, not silently assumed done.

This is a real extension of **Case 3 — Document Approval** (`case-portfolio.md`), whose original
six gaps (P1–P6 → CAP-F13, CAP-A07, CAP-A08, CAP-X03, CAP-P02, CAP-E05) are all ✅ and proven by
conformance T21–T26/T36. Nothing about the original Approval Document / Approval Step workflow
is being replaced — this study adds a **binary-artifact production concern** on top of a workflow
that already works.

---

# 2. Why this is not already covered

Two existing capabilities look adjacent but neither covers it, and the boundary matters:

| Capability | What it actually does | Why it doesn't cover this |
|---|---|---|
| CAP-F06 (`file` field) | Stores whatever bytes a user uploads, serves them back unchanged | Read/write of an opaque blob — never opens, inspects, or modifies the PDF's own content |
| CAP-F21 (templated document generation) | Renders a **new** file from an `html/template` merged with one record's data, HTML output only | Explicitly scoped as "deliberately not a binary PDF/image renderer" (own registry row) — it generates a fresh document from a template, it does not open an **existing, user-supplied** PDF and composite something onto one of its pages |

So the missing piece is specific: **open a real, already-uploaded PDF, and write a stored image
onto one of its existing pages at stored coordinates, producing a new binary PDF** — a
post-process step on an uploaded file, not a template render and not a plain blob read. See §4
for the registered capability.

---

# 3. Page-by-page design

Four new mobile screens (390×844, same visual system as `Case3Approval.dc.html` /
`CorrectedDocApproval.dc.html` from Study 29/30 — same header, card, and pill vocabulary, no new
design language introduced), published to the existing canvas:

**https://claude.ai/code/artifact/d8285b9f-6689-44c2-a8a0-6692ec724ab1** — page **"Case 3 — PDF
Signature Approval"**.

1. **`PDFUploadForm.dc.html` — Step 1, Document Details.** Ordinary submission form (Title,
   Document Type, Approval Mode) plus the PDF upload itself, shown already attached
   (`vendor-contract-q3.pdf`, 6 pages). Nothing new here — plain CAP-F06.

2. **`SignaturePlacement.dc.html` — Step 2, Signature Positions. The new screen.** Renders the
   PDF's signature page with one numbered pin per Approval Step, positioned where that
   approver's signature will be stamped; a pin is dragged to reposition. Below the preview, an
   approver list in Sequence order, each row showing the approver's name, their step's role
   label, and a group-membership chip ("Legal Group", "Finance Group") — approvers are read from
   a work group, not entered free-form, with an explicit note that changing who is in the group
   is a separate action from this screen. This is where **(page, x%, y%)** must be captured per
   Approval Step — see §4.2.

3. **Decision screen — extends `Case3Approval.dc.html`** (new file `SignatureDecision.dc.html`,
   same underlying record, same stepper): adds one card, "Your Signature Position," above the
   existing Approval Progress stepper — a small crop of the signature-block area showing exactly
   where the current approver's signature lands, with the caption "Your saved signature image is
   stamped here automatically when you approve." Everything below that card is Study 29's
   existing stepper component, unchanged. The sticky Approve/Reject bar gains one line of
   caption text; the buttons themselves are unchanged.

4. **`SignedDocumentFinal.dc.html` — the completed artifact.** Once every step is Approved: a
   green "Fully Approved" banner, the same page preview now showing all three signature images
   composited at their stored positions, a "Signed By" list with per-approver timestamps
   (the existing CAP-R04 audit trail, just surfaced here), and Download/View actions that act on
   the real generated binary PDF — not an HTML print view, which is the CAP-F21 boundary named
   in §2.

No new visual vocabulary was introduced — colors, type scale, card/pill/header shapes, and the
bottom sticky action bar are copied byte-for-byte from Study 29/30's existing components, per
that canvas's own established convention (any new screen matches the standard already settled,
deviations named explicitly rather than silently drifting).

---

# 4. Capability gaps found

## 4.1 New capability — CAP-F22, Binary PDF signature compositing

**Registered in `capability-registry.md`** (Documents/Files section): open an existing PDF
(a CAP-F06 file value), overlay a stored image at declared per-page coordinates, and produce a
new binary PDF — run at the moment an Approval Step's `Approve` event fires (an action, not a
view — closer kin to `set_field`/`create_record` than to a rendered View). This is genuinely new:
neither CAP-F06 (opaque blob storage) nor CAP-F21 (fresh-document template render) does binary
PDF *editing*. No case in the portfolio has exercised it before; this study is its first
evidence, so it is admitted `❌ Proposed`, not built.

## 4.2 No new capability needed — coordinate storage is pure composition

Storing where a signature lands is **not** a new field type. Same precedent as CAP-F19
(quantity/UoM) and CAP-F20 (many-to-many): compose from what already exists. `Approval Step`
gains three ordinary `number` fields (CAP-F07, already ✅) — `Signature Page`, `Signature X %`,
`Signature Y %` — set once during the placement step in §3.2, read once by CAP-F22 at Approve
time. Percentage-of-page coordinates are used deliberately (not absolute points) so the same
position holds regardless of page-render resolution.

## 4.3 No new capability needed — a signature image is an ordinary Machine

Where does each user's own signature image live? Not a change to the identity model (CAP-O01) —
an ordinary workspace-authored `Signature` Machine (`Owner: user` field, CAP-F05; `Image: file`
field, CAP-F06; one row per user) is sufficient, the same "compose, don't add a mechanism"
call CAP-F20's own row already made for a join Machine. CAP-F22 reads the acting approver's own
`Signature.Image` via that Machine at Approve time — an ordinary CAP-F13-style lookup, nothing
new.

## 4.4 Real pressure on CAP-O07 (Groups/Teams), still not built

`SignaturePlacement.dc.html`'s approver list is drawn from a work group, matching the owner's own
framing of the requirement. `capability-registry.md`'s CAP-O07 row remains ❌, and its own
rationale for staying deferred ("a handful of users per Application, admin-driven provisioning,
no self-service requests anywhere in scope") is now directly contradicted by a live, named case —
the first time a real case (rather than a platform survey) asks for group-based approver
assignment. This does not by itself force CAP-O07 to be built — CAP-F22 can ship first against a
flat per-approver picker (today's Case 3 shape) and swap in group-sourced approvers once CAP-O07
lands, exactly the kind of two-way door `capability-lifecycle.md` prefers — but the deferral's own
stated justification is now stale and should be read alongside this study, not on its own.

**Correction (2026-08-23, later the same day): CAP-O07 is now ✅, implemented and
conformance-proven** (`groups`/`group_members`/`group_application_roles`, conformance T194–T205 —
see `capability-registry.md`'s CAP-O07 row for the full build note), built in a separate,
concurrent session on direct owner request, independent of this study. The paragraph above is
kept as-written per this repo's append-don't-rewrite convention (`CLAUDE.md`), but its
"still not built" framing is stale the moment it was written: `SignaturePlacement.dc.html`'s
group-sourced approver list can now be built directly against a real `groups`/`group_members`
mechanism, not just sketched against one that might arrive later. CAP-F22 itself is unaffected —
it still only needs to resolve, per Approval Step, an `Approver` (`user`-typed) and that user's
own `Signature` record; how that Approver got selected (typed in directly, or picked from a
Group's membership via CAP-O07's new admin UI) is upstream of CAP-F22's own concern.

## 4.5 CAP-F22 is mode-agnostic — Sequential and Parallel both work unchanged

Clarifying a point the owner raised directly (2026-08-23): Approval Mode (`fld_ad_approval_mode`,
Sequential | Parallel — see `runtime-metadata-schema.md`'s Views section and CAP-A07/CAP-A08)
governs *when a decision is allowed to fire*, never how or whether a signature gets stamped.
CAP-F22 runs as one of an Approval Step's `Approve` event actions, exactly where
`set_field`/`activate_next`/`aggregate_status` already run — so it fires once per step's own
Approve, regardless of mode:

- **Sequential** — `sequentialGuardViolation` (`internal/handler/events.go`) blocks an
  out-of-order Approve before it ever reaches the action list, so CAP-F22 only ever runs in
  Sequence order, one stamp at a time.
- **Parallel** — no such guard exists (`parentMachine.Config["approval_mode_field"]` resolving to
  anything but `"Sequential"` short-circuits the check to a no-op, already true today for
  CAP-A07/CAP-A08). Every approver can Approve in any order, so CAP-F22 stamps could commit out
  of numeric-pin order or even concurrently. That's fine for CAP-F22 itself — each stamp only
  touches its OWN stored (page, x%, y%) coordinates, never another step's — but it does mean the
  underlying PDF write is a **read-modify-write on one shared binary file value**, so two
  Approvals landing in the same request-transaction window need the same atomic-claim discipline
  CAP-X12 already established for cross-record writes (a `SELECT ... FOR UPDATE`-style claim on
  the Document's file field, not a plain read-then-overwrite) — named here as a real
  implementation constraint for whoever builds CAP-F22, not solved by this study.

No mockup change needed: Step 1's Sequential/Parallel toggle already existed in
`PDFUploadForm.dc.html` before this note was added.

---

# 5. Can the current screens be configured declaratively via metadata? (Views assessment)

Asked directly by the owner (2026-08-23): given `runtime-metadata-schema.md`'s existing `views`
mechanism (`form`, `list`, `detail`, `dashboard`, `calendar`, `timeline`, `report`, `document`,
plus `process_map` and `board` — the full `ViewType` enum in `internal/model/model.go`), can the
four screens in §3 be *declared*, the same way `vw_ad_form`/`vw_ad_detail` already are — or do
they need real new code beyond what a Machine author can express in YAML? Answer, screen by
screen:

| Screen | Declarable today? | Why |
|---|---|---|
| 1. Upload / Document Details | **✅ Yes, already is** | Plain `type: form` — `vw_ad_form` in `approval-document.yaml` already declares this exact field set. Nothing new. |
| 2. Signature Placement | **❌ No — needs a new View type** | None of the 10 existing types render an interactive image overlay with draggable, per-child-record coordinate pins that write back on drop. Registered below as **CAP-V21**. |
| 3. Decision (stepper + signature-position preview) | **⚠️ Partial** | The Approve/Reject surface itself is ordinary `type: detail` (`vw_as_detail`, already ✅). The **Approval Progress stepper** is NOT declarative today — Study 29's own canvas annotation (`note-approval`, `canvas.json`) already named this exactly: "not a List/Report/Board/Calendar — none of the existing auxiliary View types name this shape" — it was sketched as a one-off design, never registered as a capability until now (**CAP-V20**, below). The small "Your Signature Position" mini-preview is a narrow, additive read-only mode of CAP-V21 (§ below), not a separate mechanism. |
| 4. Final Signed Document | **✅ Yes, once CAP-F22 exists** | The composited output is just written back into an ordinary `file` field (e.g. a new `Signed File` field on Approval Document) — display and Download are the existing CAP-F06 file-field rendering on the existing `detail` view. No new View type needed; the gap is entirely upstream, in CAP-F22 producing the file, not in how it's shown. |

Two real, previously-untracked findings, registered in `capability-registry.md`:

- **CAP-V20 — Sequential decision stepper** (View type, over a child Machine's own records):
  renders an ordered, vertically-stacked progress indicator — done/current/pending states — for
  a parent record's child Approval-Step-shaped rows, with sticky Approve/Reject actions scoped to
  the acting user's own current step. This existed only as a hand-built design sketch
  (`Case3Approval.dc.html`, Study 29) for six weeks before this study gave it a registry row —
  a real documentation-debt instance of exactly the kind `CLAUDE.md`'s "append, don't rewrite"
  and the registry's own "silence is not a decision" principle exist to catch.
- **CAP-V21 — Coordinate-placement editor** (View type): an image/PDF-page preview with one
  draggable pin per child record, writing (page, x%, y%) back to that child's own fields on drop.
  Closest existing precedent to build from: CAP-V14 Tier 2's board (`POST .../board-move` — a
  native HTML5 drag gesture posting a plain field write, `CanEdit`-gated, no constraint
  re-validation) — the same shape, but writing three coordinate fields instead of one lane field,
  and drawn over an image/PDF render instead of a lane grid. A read-only single-pin variant of the
  same config (no drag, no write) covers §3's "Your Signature Position" preview.

Neither CAP-V20 nor CAP-V21 is scoped to signatures specifically — CAP-V20 is a general
sequential-approval visualization (usable by any multi-step Approval-Step-shaped workflow, not
just Case 3), and CAP-V21 is a general "place a marker on an image and store where" editor
(equally useful for, say, marking a defect location on an equipment photo — no PDF- or
signature-specific assumption baked into either).

---

# 6. What this study does not do

No code implemented. No conformance tests. `capability-registry.md` gains three new ❌ rows
(CAP-F22, CAP-V20, CAP-V21) and a dated note on CAP-O07; `case-portfolio.md`'s Case 3 row gains a
dated note pointing here. Building CAP-F22 (a real PDF-manipulation library — e.g. `pdfcpu` or
similar, already Go-ecosystem-available, unlike CAP-F06's build which needed only the stdlib
`image` package), CAP-V20, and CAP-V21 is future work, sequenced whenever a real seed/case
exercises each, per this registry's standing admission discipline.
