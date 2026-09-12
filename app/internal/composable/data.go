package composable

// DataSource names where a Dataset's rows come from. Today it is always
// exactly one Machine -- 1:1 with a `records` table scoped by machine_id.
type DataSource struct {
	MachineID string
}

// Projection is the set of fields a Dataset actually needs.
type Projection struct {
	Fields []string
}

// RelationRef names a `reference` Field discovered inside a Dataset's own
// Projection -- "this Dataset also implies a relationship to another
// Machine," not a real join. Full join/relation traversal execution is
// Phase 9 (Query Planner) work; Phase 2 only names the relationship.
type RelationRef struct {
	TargetMachineID string
	ViaField        string // the `reference` Field id on the source Machine
}

// Filter is one AND-combined row condition, mirroring
// model.FilterCondition/ConstraintExpression's own shape (CAP-V09) -- the
// existing field/operator filter grammar is sugar that lowers into this
// common representation (composable-runtime-roadmap.md §6), not a second
// grammar.
type Filter struct {
	Field      string
	Operator   string
	Value      string
	ValueField string
	Expression string
}

// Sort is one ordering clause, mirroring model.SortConfig.
type Sort struct {
	Field     string
	Direction string
}

// MeasureKind is the small closed set of aggregate operations Phase 2
// represents -- exactly what DashboardSection (count, optionally grouped)
// and ReportConfig (sum, grouped) already need. Not a general aggregate
// vocabulary; a case that needs another kind (avg, min, max) admits it
// separately.
type MeasureKind string

const (
	MeasureCount MeasureKind = "count"
	MeasureSum   MeasureKind = "sum"
)

// Measure is one aggregate computed over a Dataset's rows. Field is empty
// for MeasureCount.
type Measure struct {
	Kind  MeasureKind
	Field string
}

// Dataset is a normalized, semantic data requirement -- the real reusable
// contract Phase 2 introduces (composable-runtime-roadmap.md §6), replacing
// Phase 1's per-View DatasetRef. Identity is content-addressed (see
// datasetIdentity in identity.go): two Datasets built from different
// experience shapes (a list, a board, a dashboard section) with the same
// Source/Projection/Filter/Sort/GroupBy/Measures compare equal -- no
// dedup pass needed for that, Phase 1's own DatasetRef couldn't do this
// because its identity was keyed on the asking View's own id.
//
// Still bound by §6's "Important boundary": a Dataset must never carry
// HTML, Templ syntax, CSS classes, SQL, PostgreSQL index names, or cache
// implementation details -- every field here is renderer- and
// store-agnostic on purpose.
type Dataset struct {
	Identity   NodeIdentity
	Source     DataSource
	Relations  []RelationRef
	Projection Projection
	Filter     []Filter
	Sort       []Sort
	GroupBy    []string
	Measures   []Measure
}

// LogicalQuery names a Dataset's own query requirement. Still close to
// empty in Phase 2 -- real query semantics (pagination, cost) are Phase 9
// work.
type LogicalQuery struct {
	Dataset Dataset
}

// DataIR is the flat set of Datasets discovered while normalizing an
// Application -- deliberately NOT deduplicated across nodes (Phase 7's
// Dependency DAG is what collapses equal-identity Datasets into one
// physical node); this is just the resolved requirement list, in
// declaration order.
type DataIR struct {
	Datasets []Dataset
}
