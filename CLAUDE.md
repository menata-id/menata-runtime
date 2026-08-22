# CLAUDE.md

Repo-wide orientation for Claude Code (or any AI agent) working in `menata-runtime`. If you're
about to touch code or docs specifically inside `prototype/go/`, also read
`prototype/go/CLAUDE.md` — that file covers Go-implementation patterns and gotchas this one
doesn't duplicate.

## What this repo is

`menata-runtime` is one layer downstream of `menata-id/menata` (the Business Knowledge language,
a separate repo with no machine/application concerns). This repo defines the Runtime Metadata
format and the runtime that interprets it into a living application, and validates that design
through 7 parallel prototypes on different tech stacks — `prototype/go` and
`prototype/{drupal,frappe,directus,budibase,salesforce,camunda}` (`prototype/README.md`). Only
`prototype/go` is a deep, full custom runtime under active capability-by-capability
implementation, proven by a real conformance suite; the other six are shallow "metadata-only
proof" scorecards (16 fixed features, no capability-registry tracking) — don't assume something
true of one applies to the other six.

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
