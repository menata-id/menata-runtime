// Package executor runs an Event's actions (set_field, notify, create_record,
// cross_set_field, batch_generate, composite_pdf_signature, and the rest)
// against the Application Model, inside the caller's own request-scoped
// transaction — the operational core tying auth/constraint/store together
// for a single record's own state transition.
//
// Graduated as-is from prototype/go/internal/executor — depends on 4
// internal/ packages (Study 34 §7 Method 2), the deepest dependency count
// short of handler's own orchestration layer. No architectural change
// intended; this package's own established rule (cross-record logic needing
// the Interpreter belongs in handler, not here — see prototype/go/CLAUDE.md's
// "Cross-record logic belongs in Handler" note) carries forward unchanged.
package executor
