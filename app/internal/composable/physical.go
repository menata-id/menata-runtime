package composable

import (
	"fmt"
	"strings"
)

// pushdownSupportedOperators is the closed set BuildWhereClause can safely
// translate today -- equals/not_equals map onto plain data->>field text
// comparison, the same semantics internal/constraint's own Eval already
// gives these two operators (no type ambiguity). Every other
// model.SupportedOperators entry needs type-aware casting (greater_than/
// less_than/etc., RecordStore.SumField's own `(data->>$1)::numeric`
// pattern) or "today"-sentinel resolution first (after/before/
// on_or_after/on_or_before, constraint.Eval's own parseMaybeToday) --
// deliberately not attempted here: getting either wrong would silently
// produce incorrect filter results, worse than not pushing down at all.
// "in" is dropped entirely -- neither model.FilterCondition nor Filter can
// even carry multiple values (Value is a single string), so it isn't a
// real case.
var pushdownSupportedOperators = map[string]bool{
	"equals":     true,
	"not_equals": true,
}

// BuildWhereClause translates filters into a parameterized SQL WHERE
// fragment against records.data (the same JSONB column every real query
// in internal/store/record_store.go already reads), with placeholders
// starting at argOffset+1 (pgx's own 1-indexed $N) so a caller can append
// this after its own "machine_id = $1 AND deleted_at IS NULL" predicate --
// RecordStore.List's own existing shape. Value is treated as already
// resolved -- a $current_user sentinel (CAP-V05) must be substituted by
// the caller first, exactly the contract constraint.Eval already has
// ("Eval itself has no notion of who's asking"). Returns an error naming
// the first unsupported operator rather than silently dropping or
// mistranslating a filter (CAP-X05's "Unknown = explicit" posture) -- an
// empty filters list returns an empty clause and no error.
func BuildWhereClause(filters []Filter, argOffset int) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}
	var parts []string
	var args []any
	n := argOffset
	for _, f := range filters {
		if f.ValueField != "" {
			return "", nil, fmt.Errorf("composable: filter on %q compares against another field (%q), not pushdown-safe", f.Field, f.ValueField)
		}
		if !pushdownSupportedOperators[f.Operator] {
			return "", nil, fmt.Errorf("composable: filter operator %q on field %q is not pushdown-safe", f.Operator, f.Field)
		}
		op := "="
		if f.Operator == "not_equals" {
			op = "!="
		}
		n++
		fieldArg := n
		n++
		valueArg := n
		parts = append(parts, fmt.Sprintf("(data->>$%d %s $%d)", fieldArg, op, valueArg))
		args = append(args, f.Field, f.Value)
	}
	return strings.Join(parts, " AND "), args, nil
}

// PhysicalPlan documents which of composable-runtime-roadmap.md §13's own
// nine "existing scale mechanisms" apply to one DependencyNode's own
// effective Dataset, against the REAL current internal/store.RecordStore
// -- an audit against real code, not a description of what should
// eventually exist.
type PhysicalPlan struct {
	Dependency         DependencyNode
	ProjectionPushdown bool
	FilterPushdown     bool
	AggregatePushdown  bool
	SortPushdown       bool
	PaginationPushdown bool
	IndexAware         bool
	Notes              []string
}

// BuildPhysicalPlan derives node's own PhysicalPlan.
func BuildPhysicalPlan(node DependencyNode) PhysicalPlan {
	ds := node.Dataset
	plan := PhysicalPlan{
		Dependency: node,
		// Always false: RecordStore.List/Get always SELECT the whole `data`
		// JSONB column, regardless of Dataset.Projection.
		ProjectionPushdown: false,
		// Always false: CAP-R05's own doc comment (internal/handler/
		// record_crud.go) -- pagination applies in Go, after filter/search,
		// never as a SQL LIMIT/OFFSET.
		PaginationPushdown: false,
		// Always false: no per-field index metadata exists anywhere in
		// this runtime yet.
		IndexAware: false,
	}
	plan.Notes = append(plan.Notes,
		"ProjectionPushdown: false -- RecordStore.List/Get always select the whole `data` column",
		"PaginationPushdown: false -- record_crud.go applies pagination in Go, after filter/search",
		"IndexAware: false -- no per-field index metadata exists",
	)

	if _, _, err := BuildWhereClause(ds.Filter, 0); err != nil {
		plan.FilterPushdown = false
		plan.Notes = append(plan.Notes, fmt.Sprintf("FilterPushdown: false -- %v (record_crud.go's own applyListFilter runs in Go instead)", err))
	} else {
		plan.FilterPushdown = true
	}

	// AggregatePushdown: RecordStore.CountGroupedBy/SumFieldsGroupedBy
	// already push SUM/COUNT/GROUP BY to SQL, but only for a true
	// aggregate Dataset (one with real Measures) -- a Board/Calendar's own
	// GroupBy WITHOUT Measures does NOT get this: that grouping happens in
	// Go over a full List scan today (BoardLane/CalendarGroup, internal/
	// ui/types.go).
	if len(ds.Measures) > 0 {
		plan.AggregatePushdown = true
	} else {
		plan.Notes = append(plan.Notes, "AggregatePushdown: false -- no Measures to push (a GroupBy alone is grouped in Go, not SQL)")
	}

	// SortPushdown: RecordStore.List's own sortField/sortDirection params
	// already push a single ORDER BY clause to SQL -- trivially true when
	// there's nothing to sort.
	plan.SortPushdown = true
	if len(ds.Sort) == 0 {
		plan.Notes = append(plan.Notes, "SortPushdown: true (trivial) -- no Sort declared")
	}

	return plan
}

// Explain renders a deterministic summary of p, citing the exact real
// symbols behind each flag -- §13's own exit criteria ("demonstrate why a
// composed logical request produced a given bounded physical plan"), same
// Explain() precedent as Phase 8's ExecutionPlan.
func (p PhysicalPlan) Explain() string {
	var b strings.Builder
	fmt.Fprintf(&b, "PhysicalPlan for %s:\n", p.Dependency.Identity.String())
	fmt.Fprintf(&b, "  ProjectionPushdown=%t FilterPushdown=%t AggregatePushdown=%t SortPushdown=%t PaginationPushdown=%t IndexAware=%t\n",
		p.ProjectionPushdown, p.FilterPushdown, p.AggregatePushdown, p.SortPushdown, p.PaginationPushdown, p.IndexAware)
	for _, note := range p.Notes {
		fmt.Fprintf(&b, "  - %s\n", note)
	}
	return b.String()
}

// RuntimeCapabilities audits the four §13 build items that are properties
// of the runtime as a whole, not of any one Dataset -- workspace
// concurrency limits, separate analytics/report capacity, cache, and
// materialization. All four are false today: internal/db.Connect calls
// bare pgxpool.New with no MaxConns/pool-splitting configuration, and no
// cache or materialized-view layer exists anywhere in this codebase --
// documented here as an explicit, checkable fact instead of an assumption
// a future phase might get wrong.
type RuntimeCapabilities struct {
	WorkspaceConcurrencyLimits bool
	SeparateAnalyticsCapacity  bool
	Cache                      bool
	Materialization            bool
}

// CurrentRuntimeCapabilities returns the fixed, documented current-state
// values -- all false, per this type's own doc comment.
func CurrentRuntimeCapabilities() RuntimeCapabilities {
	return RuntimeCapabilities{}
}
