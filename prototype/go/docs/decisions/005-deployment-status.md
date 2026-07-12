# ADR-005: Deployment Status — This Is No Longer Blanket PoC-Exempt

**Status:** Accepted
**Date:** 2026-07-12

## Context

`nfr-standards.md` was written 2026-07-04 as a study-only document: *"Status: v0.3 — study
only, no implementation"*, and *"nothing in this document is implemented yet. The prototype
at `aksi.menata.id` is a PoC and intentionally exempt (accepted risk, recorded in §0 Spoofing
row)."*

Both claims are now false. Across 2026-07-11 and 2026-07-12, real capabilities implementing
real parts of that standard shipped to `aksi.menata.id`:

- **CAP-P05** — deny-by-default CRUD permissions (the exact "must become deny-by-default"
  gap §0 Information Disclosure named).
- **CAP-R04** — `record_events` audit trail with real actor attribution and DB-level
  append-only enforcement (`REVOKE UPDATE, DELETE, TRUNCATE`), the §0 Tampering/Repudiation
  countermeasures.
- **CAP-I04** — correlation-id tracing across a request, including cross-record cascades.
- **CAP-X02 (partial at the time this ADR was first written)** — every permission denial, rule
  violation, and role switch was already an explicit, distinguishable security-event log line
  (ASVS V7); real authentication itself was still open. **Closed later the same day** — see
  the status update at the end of this ADR.

A governance document asserting "nothing implemented, accepted risk" while the linked
production domain has already had NFR-gated capabilities deployed to it is not a passive
inaccuracy — it's the kind of stale claim that causes someone to skip a check they should
have made, or ship something assuming risk-acceptance still covers a case it no longer does.

This ADR does not decide "is this production" as a business question — that's the
maintainer's call, made when scoping this work (see `roadmap.md`'s 2026-07-12 status
entries). It records what was decided: the blanket exemption ends here, replaced by an
honest, itemized status.

## Decision

As of 2026-07-12, `aksi.menata.id` (domain changed to `menata.app` later the same day — same
deployment, same coverage; `aksi.menata.id` now redirects to `menata.app`) is treated as
having real, if partial, NFR coverage — not blanket-exempt. Specifically:

**Now covered, with proof:**
| Area | Capability | Evidence |
|---|---|---|
| Deny-by-default access | CAP-P05 | conformance T39–T41, T44–T45 |
| Audit trail (actor + append-only) | CAP-R04 | conformance T42 |
| Correlation tracing | CAP-I04 | conformance T43 |
| Security-event logging | CAP-X02 | manual verification, `handler.go`'s `logPermissionDenied`/`logRuleViolation` |
| Real authentication (password + session + CSRF) | CAP-X02 | conformance T53–T56 |
| Workspace-scoped identity & role assignment | CAP-O01 | conformance T57–T59 |
| Workspace isolation | CAP-X06 | this session, see below |

**Still explicitly open — accepted risk, eyes open, not silence:**
- **Password reset/rotation, account lockout after repeated failed logins, MFA** — none of the
  three has a case forcing it yet; see CAP-X02's row in `capability-registry.md` for the exact
  "deferred, not done here" note.
- **Admin management of an Application's own metadata** — CAP-O01's `/admin/users` covers
  "manage user access"; the other workspace-Admin concern (Application metadata) is a
  reserved authorization boundary, not built — no metadata-editing UI exists anywhere in this
  prototype to gate yet.
- Retention/partitioning, lazy per-workspace metadata loading, per-workspace concurrency
  fairness (CAP-X11 and neighbors) — real scale concerns, not correctness/security ones,
  deferred until workspace/record counts make them relevant (see
  `docs/decisions/003-tenancy-and-indexing.md`).

## Consequences

- `nfr-standards.md`'s header is corrected to point here instead of asserting a now-false
  blanket claim (see that file's own changelog line).
- Future NFR-relevant work should update the table above, the same append-only discipline
  `roadmap.md`'s dated status blocks already use — this table is a live status, not a
  one-time snapshot.

## Status update (2026-07-12, later the same day) — CAP-X02 and CAP-O01 closed

This ADR's own "single largest open item" is resolved: real authentication (bcrypt password
verification, server-side sessions with sliding expiry, CSRF protection on every
state-changing request — implemented in this same pass, not deferred) and the workspace
identity/role registry (a two-tier model: workspace-wide Admin/Member, plus a role per
`(user, application)` pair, resolved fresh per request with no manual "switch role" step) both
shipped and are live at `aksi.menata.id`. See `roadmap.md`'s 2026-07-12 CAP-X02/CAP-O01 status
entry and `capability-registry.md`'s rows for the full account, including what was found along
the way (a seed account's bcrypt hash that never actually matched its claimed password,
invisible until real password verification existed to catch it) and what remains deliberately
open (the narrower items above — password reset, lockout, MFA, Application-metadata
management — none large enough to warrant its own ADR).
