package composable

// FieldValueKind names where a FieldValue's own Display came from --
// purely for diagnostics (the same Explain()-style inspectability every
// prior phase already used); a renderer consuming Display never needs to
// branch on it. This is composable-runtime-roadmap.md §14's own rule made
// concrete: "the renderer must not know whether a value originated from a
// field, expression, or relation."
type FieldValueKind string

const (
	FieldValueField      FieldValueKind = "field"
	FieldValueExpression FieldValueKind = "expression"
	FieldValueRelation   FieldValueKind = "relation"
)

// FieldValue is the uniform, renderer-neutral representation of one
// resolved value. Display is always a plain, render-ready string -- no
// further interpretation needed regardless of Kind.
type FieldValue struct {
	Kind    FieldValueKind
	Display string
}

// RecordSummary is one record reduced to its display-ready essentials --
// the semantic shape a RecordSummaryCard component's own render input
// rests on.
type RecordSummary struct {
	RecordID string
	Title    FieldValue
	Subtitle FieldValue // zero value (Kind == "") when the component declared no subtitle_field
}

// MetricValue is one resolved Metric component's own number, already
// formatted -- the renderer never computes an aggregate itself.
type MetricValue struct {
	Label string
	Value string
}

// CollectionItem is one row of a Collection component, resolved -- ordered
// Cells match the Collection's own Dataset.Projection.Fields order.
type CollectionItem struct {
	RecordID string
	Cells    []FieldValue
}

// StatusValue is one resolved StatusBadge component's own display value --
// still just a FieldValue, named distinctly since a StatusBadge's own
// contract (Phase 5) always resolves exactly one.
type StatusValue FieldValue

// ActionValue is one resolved, real action -- EventID is a verified
// model.Event id, Label is that Event's own declared Name.
type ActionValue struct {
	EventID string
	Label   string
}

// ActionSet is one record's own resolved ActionBar render input.
type ActionSet struct {
	RecordID string
	Actions  []ActionValue
}
