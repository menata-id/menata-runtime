// Package expr is CAP-C13's own expression evaluator -- a bounded,
// non-Turing-complete language (CEL, github.com/google/cel-go) accepted
// wherever a Constraint/Event condition or CAP-F14 computed Field can name
// one, as an alternative to the plain field/operator/value triple
// internal/constraint's Eval already evaluates for every metadata written
// before this package existed (that grammar keeps working unchanged --
// "sugar" over the same underlying condition-checking job, not replaced).
//
// NEW package, not a graduated one -- see capability-registry.md's own
// CAP-C13 row and benchmarks/ Study 37 (2026-09-07) for the design
// rationale (ObjectStack's own `packages/formula`, CEL's use in Kubernetes
// admission policies and Google Cloud IAM).
//
// # Guardrail: no I/O, ever
//
// This is structural, not a policy a reviewer has to remember: the CEL
// environment built here (buildEnv) registers exactly four variables
// (record, old, current_user, today/now) and zero custom functions --
// nothing is ever declared that could reach a network, a file, or the
// database. An expression has literally nothing to call beyond CEL's own
// built-in operators and standard-library functions (string/math/list
// manipulation, all pure). This is the same "no code hatch" posture
// ARCHITECTURE.md already holds for the runtime's pure-metadata design,
// extended to its one genuinely computed surface.
//
// # What an expression can read
//
//   - record      -- the record's own current data (map[string]any, same
//     shape store.Record.Data already has)
//   - old         -- its pre-mutation snapshot, an EMPTY map (not nil --
//     `has(old.field)` and member access both behave the same whether the
//     caller genuinely has no prior state, e.g. Create or a list filter, or
//     one exists but is simply not passed) where no real "before" exists
//   - current_user -- the acting identity's id, "" where none is resolved
//     (mirrors CAP-A02's own current_user dynamic-value fallback)
//   - today, now  -- the current date ("2026-09-07") and instant (RFC3339),
//     as plain strings -- deliberately not CEL's own timestamp type, so a
//     comparison against a record's own date-typed field (also a plain
//     string, same as every other Field in this codebase) never hits a
//     type mismatch
//
// # Fail-closed, not panic-closed
//
// Compile and Eval both return a Go error rather than panicking on bad
// input (a syntax error, a reference to an undeclared variable, a runtime
// type mismatch, a division by zero). Bool is the boolean-condition
// convenience every Constraint/Event condition actually needs: ANY error --
// compile, eval, or a non-bool result -- degrades to false, per
// capability-registry.md's own CAP-C13 row ("fail closed otherwise"). Eval
// itself stays honest about the distinction (CAP-F14's computed-field
// caller wants the real error to log, not just a boolean) -- only Bool
// collapses it.
//
// # Compiled once, not once per evaluation
//
// programFor caches a compiled cel.Program per unique expression STRING
// (sync.Map, process-lifetime) -- metadata expressions are static text
// loaded from the database, so this needs no invalidation of its own: a
// changed expression is a different cache key, and CAP-X04's reload simply
// starts populating new keys, no stale-entry cleanup required. Load-time
// validation (internal/metadata's own validateOperators) calls Compile
// directly so a broken expression fails LoadAll, not a random later
// request -- the same "Unknown = explicit" discipline that function's own
// doc comment already states for an unrecognized operator.
package expr
