# CLAUDE.md

Repo-wide orientation for Claude Code (or any AI agent) working in `menata-runtime`. If you're
about to touch code or docs specifically inside `app/`, also read `app/CLAUDE.md`. If you're
about to touch code or docs specifically inside `prototype/go/`, also read
`prototype/go/CLAUDE.md` — that file covers Go-implementation patterns and gotchas this one
doesn't duplicate.

## What this repo is

`menata-runtime` is one layer downstream of `menata-id/menata` (the Business Knowledge language,
a separate repo with no machine/application concerns). This repo defines the Runtime Metadata
format and the runtime that interprets it into a living application.

That design was validated through 7 parallel prototypes on different tech stacks — `prototype/go`
and `prototype/{drupal,frappe,directus,budibase,salesforce,camunda}` (`prototype/README.md`). Only
`prototype/go` was a deep, full custom runtime under active capability-by-capability
implementation, proven by a real conformance suite; the other six were shallow "metadata-only
proof" scorecards (16 fixed features, no capability-registry tracking) — don't assume something
true of one applies to the other six.

**That discovery phase is done** (owner decision, 2026-09-06,
`benchmarks/026-runtime-graduation-decision.md`, Study 34). `prototype/` and `benchmarks/` stay
exactly as they are — unrenamed, unrestructured — as the historical record of *how* the runtime's
required capabilities were discovered and proven. **Real development now happens in
[`app/`](app/)**, a new top-level folder graduating `prototype/go`'s own proven codebase
(~90 capabilities, 219-test conformance suite) rather than starting from zero. `app/` is currently
at blueprint stage (package skeleton + architecture docs, no ported business logic yet — see
`app/README.md`). Do not add new capability work to `prototype/go/` under the assumption it's
still the active development target; check `app/ROADMAP.md` first.

## Two files named "roadmap" — do not confuse them

`roadmap.md` (root, lowercase) and `app/ROADMAP.md` (uppercase) are different documents with
different jobs — the filenames differ only by case, which is easy to misread on a quick grep or
listing:

- **`roadmap.md`** — the capability-*discovery* method and evidence log: why a capability exists,
  what benchmark/case proved it (Studies 1–34). Read this for "what should the runtime support."
- **`app/ROADMAP.md`** — the phased plan for porting real code into `app/` (Phase 0 leaf packages
  → … → Phase 6 cutover). Read this for "what should I build in `app/` right now, in what order."

Always resolve which one a task needs by full path, never by basename alone.

## Filename case is a real convention, not noise

`README.md`'s own "# Documentation" intro is the full explanation — read that for the reasoning
and the numbering-family breakdown. The rule in short: **ALL-CAPS `.md`**
(`README.md`/`ARCHITECTURE.md`/`DEVELOPMENT.md`/`ROADMAP.md`/`CLAUDE.md`) is reserved for the
fixed onboarding-doc set every codebase repeats at its own top level (this repo's root, `app/`,
each `prototype/*`); **every other `.md` is lowercase-kebab-case**, including all of root's own
governance/reference docs and every numbered doc (specs, `benchmarks/`, ADRs). No directory in
this repo is ever uppercase. This is what produces the `roadmap.md`/`app/ROADMAP.md` collision
above — case is meaningful, never assume two same-named files (by basename) are the same document.

## Where a new document goes

`README.md`'s own "# Documentation" section is the maintained map of everything at root (Tier
1–4 + Reference Implementation), and its "## Where does a new document go?" subsection is the
decision rule for placing something new — read that before creating a `.md` file anywhere in
this repo. Two points worth stating here directly since they're easy to get backwards:

- Capability-discovery/governance machinery (`capability-registry.md`, `roadmap.md`,
  `case-portfolio.md`, `benchmarks/`, `nfr-standards.md`) belongs at **root**, even though most
  of its evidence today cites `prototype/go` internals — it's tracking a capability set meant for
  Menata Runtime generally, Go just happens to be the only prototype deep enough to check status
  against it yet.
- A prototype's own onboarding docs (`README.md`/`ARCHITECTURE.md`/`DEVELOPMENT.md`/`CLAUDE.md`)
  stay at that prototype's own top level, never under its `docs/`. `CLAUDE.md` in particular must
  stay there — Claude Code auto-loads `CLAUDE.md` from the working directory and its ancestors,
  never its descendants, so nesting it under `docs/` silently breaks auto-loading for anyone
  working from that prototype's root.

## How a document should be written

`README.md`'s own "## How should a document be written?" (right after "Where does a new document
go?") has the full list: status-header format for Tier 3/4 docs, append-don't-rewrite, evidence
citation, table-over-prose, English-by-default. Read it before writing a new document; the one
rule worth restating here because it's the easiest to skip under time pressure: **never add an
unsourced claim to a governance/evidence document** — every row needs a Case #, Study/benchmark #,
conformance test ID, or a specific commit/file/line behind it.

## Established convention: append, don't rewrite

When something documented here goes stale (a capability ships, a decision gets revisited), the
convention across this repo is to **append** a dated correction/status-update note rather than
silently rewriting the original text — `roadmap.md`'s dated status blocks, `capability-
registry.md`'s per-row notes, and the ADRs under `prototype/go/docs/decisions/` (e.g. ADR-002,
ADR-004's own "Status update (2026-08-22)" sections) are the pattern to copy. This keeps the
*why* of a past decision visible even after reality has moved past it. Follow it when you find
something stale rather than editing history away.

## Direct push to main is normal here

Solo-admin repo; branch-protection bypass on `main` is expected workflow, not a risk signal —
don't route routine doc/code changes through a PR unless asked to.
