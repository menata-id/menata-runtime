// Package model defines the Application Model — the in-memory representation
// that the Interpreter builds from Runtime Metadata.
//
// This is the structure the Router, Renderer, Executor, and Constraint Engine
// operate on. It is never persisted directly.
package model

// Workspace is the top-level organizational boundary.
type Workspace struct {
	ID           string
	Name         string
	Applications []*Application
}

// Application is an independently realizable solution inside a Workspace.
type Application struct {
	ID          string
	WorkspaceID string
	Name        string
	Machines    []*Machine
}

// Machine is the primary realization unit — it realizes one business capability.
// Config holds machine-level settings (CAP-X03) — values that configure how
// the Machine itself behaves, not a Field of its records. nil = no config,
// the default. See migrations/004_machine_config.sql.
type Machine struct {
	ID            string
	ApplicationID string
	Name          string
	Fields        []*Field
	Events        []*Event
	Constraints   []*Constraint
	Permissions   []*Permission
	Views         []*View
	Config        map[string]string
}

// Field is a typed piece of business information on a Machine.
type Field struct {
	ID        string
	MachineID string
	Name      string
	Type      FieldType
	Position  int
	Required  bool
	Options   FieldOptions
}

// FieldType is the data type of a Field.
type FieldType string

const (
	FieldTypeText      FieldType = "text"
	FieldTypeRichText  FieldType = "rich_text"
	FieldTypeNumber    FieldType = "number"
	FieldTypeMoney     FieldType = "money"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeDate      FieldType = "date"
	FieldTypeDateTime  FieldType = "date_time"
	FieldTypeUser      FieldType = "user"
	FieldTypeFile      FieldType = "file"
	FieldTypeValueList FieldType = "value_list"
	FieldTypeReference FieldType = "reference"
)

// FieldOptions holds type-specific configuration.
// value_list: Values lists the allowed options.
// reference:  TargetMachine names the Machine this field points to.
type FieldOptions struct {
	Values        []string `json:"values,omitempty"`
	TargetMachine string   `json:"target_machine,omitempty"`
}

// Event is a business occurrence that triggers actions on a Machine.
// Condition is a guard (CAP-E06): the event may only be triggered when the
// record's current data satisfies it. nil = always allowed.
type Event struct {
	ID        string
	MachineID string
	Name      string
	Position  int
	Actions   []*EventAction
	Condition *ConstraintExpression
}

// EventAction is a single step executed when an Event fires.
type EventAction struct {
	ID       int64
	EventID  string
	Type     ActionType
	Position int
	Params   map[string]any
}

// ActionType describes what an EventAction does.
type ActionType string

const (
	ActionSetField        ActionType = "set_field"
	ActionNotify          ActionType = "notify"
	ActionCreateRecord    ActionType = "create_record"
	ActionActivateNext    ActionType = "activate_next"    // CAP-A07
	ActionAggregateStatus ActionType = "aggregate_status" // CAP-A08
	ActionTriggerEvent    ActionType = "trigger_event"    // CAP-E05
)

// Constraint is a business rule enforced before an event is accepted.
type Constraint struct {
	ID         string
	MachineID  string
	Rule       string // human-readable description
	Expression ConstraintExpression
	Condition  *ConstraintExpression // nil = always applies
	Position   int
}

// ConstraintExpression is the evaluatable part of a Constraint.
// CAP-C07 (cross-field comparison): ValueField, when set, compares against
// data[ValueField] instead of the literal Value -- e.g. "End Date after
// Start Date" as opposed to "Due Date after today". Exactly one of Value/
// ValueField is meaningful per expression; ValueField wins if both are set.
// CAP-C12 (uniqueness): Fields, when set (operator "unique" only), names a
// composite key -- ["fld_a","fld_b"] for a multi-field uniqueness rule,
// checked cross-record (constraint.Engine can't do this alone, see
// handler.uniquenessViolations), not evaluated by constraint.Eval.
type ConstraintExpression struct {
	Field      string   `json:"field,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Operator   string   `json:"operator"`
	Value      string   `json:"value,omitempty"`
	ValueField string   `json:"value_field,omitempty"`
}

// SupportedOperators is every constraint/condition operator this runtime
// actually evaluates (constraint.Eval) or otherwise enforces ("unique",
// enforced cross-record by handler.uniquenessViolations, not Eval). CAP-X05:
// the loader rejects any Constraint or Event condition naming an operator
// outside this set at load time -- an unrecognized operator used to
// silently never fire (constraint.Eval's default case returned true,
// "satisfied"); failing the load instead turns a silent no-op into an
// immediate, explicit error a metadata author actually sees.
var SupportedOperators = map[string]bool{
	"required":              true,
	"equals":                true,
	"not_equals":            true,
	"after":                 true,
	"before":                true,
	"greater_than":          true,
	"less_than":             true,
	"greater_than_or_equal": true,
	"less_than_or_equal":    true,
	"unique":                true,
}

// Permission assigns a set of Events to a business Role.
// OwnerField (CAP-P02): when set, the Events this Permission grants also
// require the acting identity to equal the record's own OwnerField value —
// e.g. only the specific Approver named on an Approval Step, not anyone
// holding the "Approver" role, may decide it (WRP-1 Direct Allocation).
// Empty = role-only, the default.
// CanRead/CanCreate/CanEdit (CAP-P05): CRUD-level permission, independent of
// Events. A role with no Permission row at all on a machine has none of
// these — deny-by-default.
type Permission struct {
	ID         string
	MachineID  string
	Role       string
	Events     []string // event ids
	OwnerField string
	CanRead    bool
	CanCreate  bool
	CanEdit    bool
}

// View describes how a Machine's data is presented.
type View struct {
	ID        string
	MachineID string
	Name      string
	Type      ViewType
	Position  int
	Config    ViewConfig
}

// ViewType is the presentation style of a View.
type ViewType string

const (
	ViewTypeForm      ViewType = "form"
	ViewTypeList      ViewType = "list"
	ViewTypeDetail    ViewType = "detail"
	ViewTypeDashboard ViewType = "dashboard"
	ViewTypeCalendar  ViewType = "calendar"
	ViewTypeTimeline  ViewType = "timeline"
)

// ViewConfig holds view-specific presentation configuration.
type ViewConfig struct {
	Fields      []string    `json:"fields,omitempty"`       // form: ordered field ids
	Columns     []string    `json:"columns,omitempty"`      // list: visible column field ids
	DefaultSort *SortConfig `json:"default_sort,omitempty"` // list: initial sort
}

// SortConfig defines the default sort order for a list view.
type SortConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // asc | desc
}
