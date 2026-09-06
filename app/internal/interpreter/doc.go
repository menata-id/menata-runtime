// Package interpreter builds an indexed, in-memory Application Model from
// Runtime Metadata for fast lookups by Handlers and the Router without a
// per-request database round trip.
//
// Graduated as-is from prototype/go/internal/interpreter — depends on 1
// internal/ package (Study 34 §7 Method 2). One deferred change, named not
// solved here: prototype/go/docs/decisions/002-metadata-loading.md's own
// Option C (Postgres LISTEN/NOTIFY, propagating a metadata reload to every
// server process) remains unbuilt (CAP-X11) — this in-memory cache is
// still single-process-only. Not a blocker: this host runs no multi-
// instance/container deployment (Study 34 §6's own "no containers, real
// resource constraint" correction), so single-process is the actual target,
// not a gap being silently carried forward.
package interpreter
