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
type Machine struct {
	ID            string
	ApplicationID string
	Name          string
	Fields        []*Field
	Events        []*Event
	Constraints   []*Constraint
	Permissions   []*Permission
	Views         []*View
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
// reference:  MachineID points to the referenced Machine.
type FieldOptions struct {
	Values    []string `json:"values,omitempty"`
	MachineID string   `json:"machine_id,omitempty"`
}

// Event is a business occurrence that triggers actions on a Machine.
type Event struct {
	ID        string
	MachineID string
	Name      string
	Position  int
	Actions   []*EventAction
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
	ActionSetField    ActionType = "set_field"
	ActionNotify      ActionType = "notify"
	ActionCreateRecord ActionType = "create_record"
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
type ConstraintExpression struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
}

// Permission assigns a set of Events to a business Role.
type Permission struct {
	ID        string
	MachineID string
	Role      string
	Events    []string // event ids
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
