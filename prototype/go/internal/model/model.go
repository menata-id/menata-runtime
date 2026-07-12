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
	Holidays     []string // CAP-O06: "YYYY-MM-DD" dates this Workspace declares as non-working, consumed by CAP-A11's "N Business Days" date arithmetic
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
	Subscriptions []*Subscription // CAP-I01: this Machine's OWN declared interest in other Machines' Events
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
// record's current data satisfies it. nil = always allowed. AggregateCondition
// (CAP-A14) is the same idea one level up: a guard computed across sibling
// records, not just this one's own fields -- an event may only be triggered
// when a SUM of some field, scoped to records sharing this record's own
// value for another field, satisfies an operator/value check (e.g. "only
// once this Member's total Points reach 100"). The two are mutually
// exclusive per Event in practice (metadata declares one or the other in
// the same `condition` JSONB column, disambiguated at load time by an
// "aggregate" key) -- both stored as separate typed fields here rather than
// forcing one shape to represent both.
type Event struct {
	ID                 string
	MachineID          string
	Name               string
	Position           int
	Actions            []*EventAction
	Condition          *ConstraintExpression
	AggregateCondition *AggregateCondition
	InputFields        []string  // CAP-P04: field ids collected fresh at trigger time (a delegation target picker), not read from the record's own data
	Schedule           *Schedule // CAP-E02/E03: fires without any user action, on a time or date-field trigger
	Category           string    // CAP-I02: documentation metadata, no runtime behavior of its own
	SchemaVersion      string    // CAP-I02: documentation metadata
	DeprecatedMessage  string    // CAP-I02: non-empty means this Event still works but logs a warning when triggered
}

// Schedule (CAP-E02/E03) declares an Event that fires on its own, not from
// a user action -- exactly one of Time or DateField is set, disambiguated
// at load time by which key is present in the same way AggregateCondition
// is distinguished from an ordinary Condition:
//
//	{"time": "08:00"}                                    CAP-E02, daily
//	{"date_field": "fld_due_date", "offset_days": -1}     CAP-E03, relative
//
// Both are processed by the same background tick (handler.
// RunScheduledEvents) and de-duplicated per record via the EXISTING
// record_events audit table (CAP-R04) -- "has this event already fired on
// this record today" -- rather than a new tracking table.
type Schedule struct {
	Time       string `json:"time,omitempty"`        // CAP-E02: "HH:MM", UTC, fires daily
	DateField  string `json:"date_field,omitempty"`  // CAP-E03: a date Field on this Event's own Machine
	OffsetDays int    `json:"offset_days,omitempty"` // CAP-E03: fires when today == that Field's value + OffsetDays
}

// Subscription (CAP-I01) is a SUBSCRIBER Machine's own declared interest in
// a PUBLISHER Event elsewhere -- Pattern C's whole point is the publisher
// never names its subscribers; only the subscriber names the publisher.
// When PublisherEventID fires (on any record), one new record is created
// on this Subscription's own MachineID, Fields resolved from the
// publisher's post-event data the same way CAP-A06's create_record already
// resolves fields ("field:<id>" copies, a literal is a literal, dynamic
// tokens like current_user/today resolve the same way).
//
// Contract/OnViolation (CAP-I03) gate that creation on the publisher's own
// data first -- Contract is AND-combined ConstraintExpressions (the same
// shape/operators a Constraint or Event Condition already uses) checked
// against the publisher's data; OnViolation decides what a failed check
// means for THIS subscription only ("skip", the default -- don't create
// the record, just log it -- or "log_only" -- create it anyway, just note
// the mismatch).
//
// CAP-I05 (cross-cutting contribution) needs no new field here at all --
// it's proven by two or more Subscriptions, from DIFFERENT
// PublisherEventIDs, targeting the SAME MachineID (a shared KPI/
// gamification Machine) -- the same mechanism, applied to more than one
// publisher, decoupled from each publisher's own definition.
//
// Error isolation (the "4 rules" this capability was named for): (1) a
// subscriber's own failure never rolls back the publisher's already-
// persisted write -- Subscriptions process strictly AFTER Persist
// succeeds (handler.processSubscriptions, called from the same place
// CAP-A07/A08/E05's own post-commit workflow actions already run); (2)
// each Subscription is independent -- one failing (a Contract violation,
// or a real error creating the record) doesn't stop the next from
// running; (3) every failure is logged (slog.Warn/Error), never silently
// swallowed; (4) a Subscription only ever sees the publisher's FINAL
// post-event data, the same source create_record/notify/cross_set_field
// already read from, never a partial/uncommitted view.
type Subscription struct {
	ID               string
	MachineID        string // subscriber -- one new record is created HERE
	PublisherEventID string // the Event this subscribes to, cross-machine
	Fields           map[string]any
	Contract         []ConstraintExpression
	OnViolation      string // "skip" (default) | "log_only"
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
	ActionCrossSetField   ActionType = "cross_set_field"  // CAP-A13
	ActionBatchGenerate   ActionType = "batch_generate"   // CAP-A15
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

// AggregateCondition (CAP-A14) gates an Event on a computed sum across
// sibling records, not just the triggering record's own data -- see Event's
// own doc comment for the shape and reasoning. Machine defaults to the
// Event's own Machine when empty (the common case: summing a field across
// other records of the SAME Machine, e.g. a Member's own ledger entries).
type AggregateCondition struct {
	Machine        string `json:"machine,omitempty"`
	AggregateField string `json:"aggregate_field"`
	ScopeField     string `json:"scope_field"`
	Operator       string `json:"operator"`
	Value          string `json:"value"`
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
	ID           string
	MachineID    string
	Role         string
	Events       []string // event ids
	OwnerField   string
	CanRead      bool
	CanCreate    bool
	CanEdit      bool
	CanDelete    bool     // CAP-R03 -- defaults false at the DB level, unlike the other three
	HiddenFields []string // CAP-P06 -- field ids this role's Permission excludes from List/Detail/Form rendering
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
	ViewTypeReport    ViewType = "report" // CAP-V13: grouped aggregate (e.g. Trial Balance)
)

// ViewConfig holds view-specific presentation configuration.
type ViewConfig struct {
	Fields      []string           `json:"fields,omitempty"`       // form: ordered field ids
	Columns     []string           `json:"columns,omitempty"`      // list/calendar/timeline: visible column field ids
	DefaultSort *SortConfig        `json:"default_sort,omitempty"` // list: initial sort
	ChildLines  *ChildLinesConfig  `json:"child_lines,omitempty"`  // form: CAP-F16 embedded child rows
	Filter      []FilterCondition  `json:"filter,omitempty"`       // list: CAP-V09 declarative row filter, CAP-V05 "my records" via $current_user
	DateField   string             `json:"date_field,omitempty"`   // calendar/timeline: CAP-V07, the date field grouped/ordered on
	Report      *ReportConfig      `json:"report,omitempty"`       // report: CAP-V13
	Sections    []DashboardSection `json:"sections,omitempty"`     // dashboard: CAP-V10 composed multi-machine summary
	Steps       [][]string         `json:"steps,omitempty"`        // form: CAP-V12 multi-step wizard -- each entry is a Fields subset shown one step at a time; unset means single-step (existing behavior)
	ManualOrder bool               `json:"manual_order,omitempty"` // list: CAP-V14 -- sort by the free-standing sort_order column (migrations/011) and render Up/Down controls, instead of DefaultSort/created_at
}

// ReportConfig (CAP-V13) declares a "report" View as a grouped aggregate
// over ANOTHER Machine's records -- e.g. a Trial Balance grouping Journal
// Entry Line by Account, summing Debit/Credit -- rather than a new
// data-modeling concept: the report is computed at render time from
// existing records, nothing is stored. Machine/GroupField/SumFields are
// validated at load time to name a real Machine and real Fields on it
// (metadata/loader.go), same "Unknown = explicit" discipline as
// ChildLinesConfig.
type ReportConfig struct {
	Machine    string   `json:"machine"`     // source Machine to aggregate
	GroupField string   `json:"group_field"` // field whose value becomes the report's row grouping
	SumFields  []string `json:"sum_fields"`  // numeric fields summed per group
}

// DashboardSection (CAP-V10) is one tile of a composed dashboard View --
// a record count for Machine, optionally broken down by GroupField (a
// value_list field, e.g. count of Tasks per Stage). Multiple sections
// across DIFFERENT Machines compose one dashboard, the actual point of
// CAP-V10 versus a single Machine's own list/report view.
type DashboardSection struct {
	Title      string `json:"title"`
	Machine    string `json:"machine"`
	GroupField string `json:"group_field,omitempty"`
}

// FilterCondition (CAP-V09) is one AND-combined row filter on a list View --
// "Overdue Tasks" (field: fld_due, operator: before, value: today) or "My
// Records" (field: fld_owner, operator: equals, value: $current_user).
// Reuses constraint.Eval's own expression shape and operator set (CAP-X05's
// SupportedOperators) rather than inventing a second condition grammar --
// same reasoning as Event.Condition already sharing it. $current_user
// (CAP-V05) is a sentinel Value resolved to the acting identity's user id at
// request time, by the caller, before Eval ever sees it -- Eval itself has
// no notion of "who's asking."
type FilterCondition struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
}

// ChildLinesConfig (CAP-F16) declares that a form view also authors N rows
// of a child Machine atomically with the parent -- Journal Entry + its
// Lines as one document, not the parent record followed by N separate
// Create-record trips to the child Machine. Deliberately a fixed-slot,
// server-rendered design (MaxRows blank row slots, empty ones ignored on
// submit) rather than JS-driven dynamic add/remove -- matches this
// prototype's no-SPA-framework posture (HTMX only). CREATE-time only: an
// existing child row is still edited via its own Machine's ordinary
// CAP-R02 edit form, the same way Approval Step rows already are -- a
// deliberate, named scope boundary, not an oversight.
type ChildLinesConfig struct {
	Machine     string   `json:"machine"`      // child Machine id
	ParentField string   `json:"parent_field"` // the child's own `reference` field pointing back at this parent
	Fields      []string `json:"fields"`       // child fields exposed per row, in order
	MaxRows     int      `json:"max_rows"`     // fixed number of row slots rendered; 0 defaults to 10
}

// SortConfig defines the default sort order for a list view.
type SortConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // asc | desc
}
