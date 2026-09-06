// Package model defines the Application Model — the in-memory representation
// the Interpreter builds from Runtime Metadata, operated on by Router/
// Renderer/Executor/Constraint Engine, never persisted directly.
//
// Graduated as-is from prototype/go/internal/model (Study 34,
// benchmarks/026-runtime-graduation-decision.md) — a leaf package, zero
// dependencies on any other internal/ package, confirmed by Study 34 §7
// Method 2's own import-graph measurement. No architectural change intended
// for this package; port verbatim when the development plan reaches it.
package model
