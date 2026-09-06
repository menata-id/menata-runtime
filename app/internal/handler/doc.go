// Package handler holds the HTTP handlers for every page/route — the
// orchestration layer that wires interpreter, executor, permission, store,
// auth, constraint, metadata, and ui together to answer one request.
//
// Graduated from prototype/go/internal/handler — the legitimate top of the
// layer graph (9 internal/ dependencies, Study 34 §7 Method 2), not a "God
// object": every other package is a leaf or thin mid-layer specifically
// because this one does the wiring, the natural shape of a monolith's own
// request-handling layer. Two deferred changes, named not solved here:
//   - Study 34 §5 found no rate-limiting/DoS-protection middleware anywhere
//     in this layer — a real gap for the development plan to close.
//   - Study 34 §5 also flagged this package's own team-scale coupling as
//     evaluated only at solo-developer scale so far (no stated public-API
//     boundary between "core runtime" and "web layer") — real but entirely
//     speculative absent a second engineer; not addressed by this
//     blueprint, revisit only if that changes.
package handler
