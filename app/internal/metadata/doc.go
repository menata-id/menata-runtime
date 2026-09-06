// Package metadata loads Runtime Metadata from Postgres into the Application
// Model (per-table loading), validates it (operators, dangling references),
// and compiles declared Process Overlay metadata into substrate primitives
// (Events, Permissions, Status field, trigger_event chains) at load time —
// "declared process, emergent execution."
//
// Graduated from prototype/go/internal/metadata, but SPLIT during the port
// rather than carried over verbatim (roadmap.md item 20, found via Study 34
// §7 Method 1): prototype/go's own loader.go had grown to 1,098 lines
// bundling three separable concerns. Here they land in three files instead
// of one, a pure move with no logic change, same discipline ADR-006's own
// handler.go split already established:
//   - loader.go   — the ~12 per-table Load* functions (the bulk)
//   - validate.go — validateOperators / validateReferences
//   - compile.go  — compileApprovalRequirements / injectApprovalQuorum
//     (already its own file in prototype/go; unchanged, just joins this
//     package's split alongside the other two)
// Depends on 1 internal/ package (Study 34 §7 Method 2) — the split doesn't
// change that.
package metadata
