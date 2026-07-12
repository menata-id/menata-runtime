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
- **CAP-X02 (partial)** — every permission denial, rule violation, and role switch is now an
  explicit, distinguishable security-event log line (ASVS V7), though real authentication
  itself is still open (below).

A governance document asserting "nothing implemented, accepted risk" while the linked
production domain has already had NFR-gated capabilities deployed to it is not a passive
inaccuracy — it's the kind of stale claim that causes someone to skip a check they should
have made, or ship something assuming risk-acceptance still covers a case it no longer does.

This ADR does not decide "is this production" as a business question — that's the
maintainer's call, made when scoping this work (see `roadmap.md`'s 2026-07-12 status
entries). It records what was decided: the blanket exemption ends here, replaced by an
honest, itemized status.

## Decision

As of 2026-07-12, `aksi.menata.id` is treated as having real, if partial, NFR coverage —
not blanket-exempt. Specifically:

**Now covered, with proof:**
| Area | Capability | Evidence |
|---|---|---|
| Deny-by-default access | CAP-P05 | conformance T39–T41, T44–T45 |
| Audit trail (actor + append-only) | CAP-R04 | conformance T42 |
| Correlation tracing | CAP-I04 | conformance T43 |
| Security-event logging | CAP-X02 (partial) | manual verification, `handler.go`'s `logPermissionDenied`/`logRuleViolation` |
| Workspace isolation | CAP-X06 | this session, see below |

**Still explicitly open — accepted risk, eyes open, not silence:**
- **CAP-X02, the rest of it** — real authentication remains unimplemented. See that row in
  `capability-registry.md` for the specific gaps tracked; this ADR names it as the single
  largest open item rather than leaving it implied by a registry row alone, without
  restating the mechanics here.
- **CAP-O01** — the workspace identity/role registry doesn't exist yet; see that row in
  `capability-registry.md`.
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
- CAP-X02's remaining gap (real authentication) is the single largest open item this ADR
  surfaces plainly rather than leaving implied by a capability registry row alone.
