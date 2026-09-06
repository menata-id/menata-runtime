// Package permission provides a Guard that determines whether a given
// role/identity set is permitted to read/create/edit/trigger against a
// Machine — CAP-O07 union role semantics (direct assignment or Group
// membership) and CAP-P02 owner-field checks both live here.
//
// Graduated as-is from prototype/go/internal/permission — depends on 1
// internal/ package (Study 34 §7 Method 2). No architectural change
// intended.
//
// # Actor-parameter convention
//
// This codebase has no single unified "actor" type threaded through
// internal/permission/internal/executor/internal/handler the way, say,
// portal-ga3's own `actor models.UserContext` is -- role and identity are
// carried as separate string parameters instead. The convention already
// followed everywhere those parameters appear (internal/executor,
// internal/handler) is:
//
//  1. ctx context.Context, when the function takes one, is always the
//     first parameter.
//  2. actorRole always comes immediately before actorIdentity (and
//     actorIdentityID, when present) -- never reversed, never split apart
//     by an unrelated parameter.
//
// Formalized here (2026-09-06, benchmarked against portal-ga3's own
// explicit "Actor Context Pattern" rule -- see
// app/docs/portal-ga3-code-quality-benchmark.md) because it was previously
// only an unwritten pattern a new function could drift from without
// anyone noticing. app/scripts/check-quality-gates.sh's own Gate 4 checks
// every function in these three packages against it.
package permission
