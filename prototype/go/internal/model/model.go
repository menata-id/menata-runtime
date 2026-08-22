// Package model defines the Application Model — the in-memory representation
// that the Interpreter builds from Runtime Metadata.
//
// This is the structure the Router, Renderer, Executor, and Constraint Engine
// operate on. It is never persisted directly.
package model

import "strings"

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
// Process (Process Overlay B1, migrations/019) is the Machine's declared
// process, compiled by the loader into ordinary Events/guards/Permissions
// before anything downstream ever sees this Machine — Router, Guard,
// Executor, and Engine have no knowledge of it. nil = no overlay.
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
	Process       *Process        // Process Overlay B1 -- consumed (and satisfied) entirely at load time
	// NeedsCreatedAtGuard (CAP-W07) is true iff any Constraint on this
	// Machine has a `new_records` change_policy -- lets every
	// engine.Violations call site skip exposing ChangePolicyCreatedAtField
	// (a real map copy) for the vast majority of Machines that never use it.
	NeedsCreatedAtGuard bool
}

// Process (Process Overlay B1 — brd-menata-runtime-v2.md §7.2, Study 20 §6)
// declares a Machine's process as states + transitions, compiled at load
// time into the substrate primitives that already exist:
//
//	transition {name, from, to, actor}  →  Event (id "evt_<machine>_<slug(name)>",
//	    condition {status equals From} via CAP-E06, action set_field status→To,
//	    plus the declaration's own on_transition actions)
//	  + Permission (per distinct actor {role, owner_field} pair, CAP-P01/P02)
//	auto {from, to}                     →  a System-triggered Event of the same
//	    shape, chained from every transition landing on From via CAP-E05's
//	    trigger_event action
//	states                              →  the Status value_list Field's values;
//	    the Field is generated ("fld_<machine>_status") when the Machine does
//	    not declare its own value_list Field named "Status" — States[0] is the
//	    initial state, per the existing first-value-is-default convention
//
// The runtime executes only the compiled result — "declared process,
// emergent execution." Event ids are deterministic functions of the
// declaration so identity stays stable across reloads (004-runtime-metadata
// §Stable Identity).
type Process struct {
	States      []string             `json:"states"`
	Transitions []*ProcessTransition `json:"transitions"`
	Auto        []*ProcessAuto       `json:"auto,omitempty"`
	SLA         []*ProcessSLA        `json:"sla,omitempty"`
}

// ProcessSLA (CAP-W04, Process Overlay B4 — brd-menata-runtime-v2.md §7.3,
// Study 20 §5.2) declares a time budget for sitting in State: Duration
// (CAP-A11's own date-arithmetic unit grammar, e.g. "2 Business Days") is
// stamped onto a generated due-date Field every time a compiled transition
// or auto step lands on State; a generated scheduled Event (CAP-E03) fires
// the moment today reaches that due date, running OnBreach -- entirely
// within this one Machine, no cross-machine wiring (internal/metadata/
// compile.go's compileSLA).
type ProcessSLA struct {
	State    string          `json:"state"`
	Duration string          `json:"duration"`
	OnBreach ProcessOnBreach `json:"on_breach"`
}

// ProcessOnBreach names what happens when a ProcessSLA's due date passes
// while the record is still in State. Notify is the exact {role, ...}/
// {recipient_field, role} shape ActionNotify's own params already use — no
// new action grammar. EscalateTo, if set, must be a declared process state;
// the breach compiles an extra set_field moving the record there, the same
// shape an ordinary transition's own action already is. Omit EscalateTo for
// a notify-only breach (no forced state change).
type ProcessOnBreach struct {
	Notify     map[string]any `json:"notify,omitempty"`
	EscalateTo string         `json:"escalate_to,omitempty"`
}

// ProcessTransition is one declared state change. Actor is required — a
// transition nobody may perform is a declaration error, caught at load time.
// Requirements (CAP-W01, Process Overlay B3) name what must be true before
// this transition's target state may be reached — see ProcessRequirement.
type ProcessTransition struct {
	Name         string                `json:"name"`
	From         string                `json:"from"`
	To           string                `json:"to"`
	Actor        ProcessActor          `json:"actor"`
	Actions      []*ProcessAction      `json:"on_transition,omitempty"`
	Requirements []*ProcessRequirement `json:"requirements,omitempty"`
}

// ProcessRequirement (CAP-W01, Process Overlay B3 — brd-menata-runtime-v2.md
// §7.4, Study 20 §6.3) declares that at least Cardinality records of a child
// Machine (Target, which must hold a `reference` Field back to this
// Machine — validated at load time, metadata/loader.go) must exist before
// the declaring transition's target state may be reached.
//
// Compiles to (internal/metadata/compile.go's compileProcess):
//   - a generated `number` counter Field on THIS Machine
//     (model.RequirementCounterFieldID), starting at "0"
//   - a generated Constraint gating the transition's `to` state on that
//     counter (reusing CAP-C09's existing trigger-time re-validation --
//     no new check mechanism)
//
// The counter's VALUE is maintained by write-time fan-in, not a query: when
// a Target record is created referencing this record, the counter is
// incremented on the spot (handler.stampRequirementCounters) -- "write-time
// fan-in, read-time O(1)" (Study 20 §6.3), the one genuinely new runtime
// mechanism the whole Process Overlay needs.
//
// Type is "evidence" only this pass -- the comparator BRD's own sharpest
// named gap (Study 19 §4.1: "no generic check exists for evidence-count").
// Other types (approval/task/entity/document/decision) are deliberately
// not built speculatively; escalate only when a case demands one, per this
// codebase's own CAP-F19 precedent ("escalate only when cardinality
// demands it").
//
// Cardinality is "N" (exact), "N..*" (at least N), or "N..M" (a bounded
// range) -- parsed at compile time (compile.go's parseCardinality).
type ProcessRequirement struct {
	Type        string `json:"type"`
	Target      string `json:"target"`
	Cardinality string `json:"cardinality,omitempty"` // evidence

	// approval (CAP-W03's declarative form) -- see compileApprovalRequirements
	// (internal/metadata/loader.go). "M" (how many voters exist) is
	// deliberately not declared here -- handler.doAggregateStatus already
	// computes it as a runtime fact (however many sibling records currently
	// reference the parent), not a metadata constant, so the compiler
	// doesn't need to either.
	MinApprovals     int    `json:"min_approvals,omitempty"`
	OnQuorumApproved string `json:"on_quorum_approved,omitempty"` // a transition NAME on the declaring machine's own process
	OnQuorumRejected string `json:"on_quorum_rejected,omitempty"` // ditto
}

// FindFieldByName resolves a Field by name, case-insensitive -- the same
// heuristic handler.go's displayLabel/Sequence/Decision/Approver lookups
// already use (Menata Language has no grammar yet for a business author to
// name "the field that means X"), exported so internal/metadata's
// cross-machine compile pass (CAP-W03's declarative quorum) can reuse it
// without a package cycle back into internal/handler.
func FindFieldByName(machine *Machine, name string) *Field {
	for _, f := range machine.Fields {
		if strings.EqualFold(f.Name, name) {
			return f
		}
	}
	return nil
}

// FindReferenceFieldTo resolves the Field on machine (if any) whose type is
// `reference` and whose target_machine is targetMachineID -- "the field
// that scopes this record to its parent" heuristic, same reasoning and same
// export rationale as FindFieldByName.
func FindReferenceFieldTo(machine *Machine, targetMachineID string) *Field {
	for _, f := range machine.Fields {
		if f.Type == FieldTypeReference && f.Options.TargetMachine == targetMachineID {
			return f
		}
	}
	return nil
}

// RequirementCounterFieldID is the deterministic id of the generated
// counter Field a ProcessRequirement compiles to -- shared between the
// compiler (which creates the Field and its gating Constraints) and the
// runtime write-time-fan-in hook (which increments it), so both sides
// agree on the id without either reading the other's code. Target is
// already a Machine id (already-valid identifier characters, unlike a
// human-authored transition Name) -- no slugging needed, unlike the
// compiler's own event/permission ids.
func RequirementCounterFieldID(machineID, target string) string {
	return "fld_" + machineID + "_" + target + "_count"
}

// ProcessActor names who may perform a transition: a Role (CAP-P01), further
// narrowed to the specific person a `user`-typed OwnerField on the record
// names when set (CAP-P02) — the same two-level model hand-authored
// Permissions already use.
type ProcessActor struct {
	Role       string `json:"role"`
	OwnerField string `json:"owner_field,omitempty"`
}

// ProcessAction is one extra action on a compiled transition Event, in the
// exact {type, params} shape hand-authored event_actions rows already use —
// no second action vocabulary.
type ProcessAction struct {
	Type   ActionType     `json:"type"`
	Params map[string]any `json:"params"`
}

// ProcessAuto declares a system-performed transition: whenever a record
// lands on From (via any declared transition), it immediately moves on to
// To — compiled as a trigger_event chain (CAP-E05), so it runs through the
// same guarded triggerEvent path a user-performed transition would.
type ProcessAuto struct {
	From string `json:"from"`
	To   string `json:"to"`
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
	FieldTypeTime      FieldType = "time"     // CAP-F10, 2026-07-12
	FieldTypeDuration  FieldType = "duration" // CAP-F10, 2026-07-12
	FieldTypeUser      FieldType = "user"
	FieldTypeFile      FieldType = "file"
	FieldTypeValueList FieldType = "value_list"
	FieldTypeReference FieldType = "reference"
	// FieldTypeComputed (CAP-F14, 2026-07-12) has no stored value at all --
	// List/Detail render Options.SourceField's own value times Options.Factor
	// at render time (same "computed at render time, nothing stored"
	// precedent CAP-V13's report View already established), never part of a
	// Create/Update form. Covers the one sub-pattern with real evidence
	// (Study 15's unit/currency conversion, amount * factor); cross-record
	// aggregate rollup and static categorical lookup are still open, see
	// capability-registry.md's own CAP-F14 row.
	FieldTypeComputed FieldType = "computed"
)

// FieldOptions holds type-specific configuration.
// value_list: Values lists the allowed options.
// reference:  TargetMachine names the Machine this field points to.
type FieldOptions struct {
	Values        []string `json:"values,omitempty"`
	TargetMachine string   `json:"target_machine,omitempty"`

	// Default (CAP-F15, 2026-07-12): a literal value this field starts at on
	// Create when the submitter leaves it blank -- the same "first value ="
	// initial state" convention Status already had, generalized to any
	// field, any type. Never overrides a value the submitter DID provide.
	Default string `json:"default,omitempty"`

	// money (CAP-F08, 2026-07-12): exactly one of Currency (fixed code,
	// e.g. "IDR") or CurrencyField (a reference to another field on the
	// same record, for CAP-F17's per-transaction currency) is required --
	// enforced at load time, see loader.go's validateMoneyOptions.
	Currency      string `json:"currency,omitempty"`
	CurrencyField string `json:"currency_field,omitempty"`

	// file (CAP-F06, 2026-07-12): Accept is a comma-separated MIME
	// allow-list (empty = any file type accepted, no image pipeline runs).
	// Compress/MaxDimension/Format only apply when Accept implies images
	// (e.g. "image/*") -- see internal/handler/upload.go.
	Accept       string `json:"accept,omitempty"`
	Compress     bool   `json:"compress,omitempty"`
	MaxDimension int    `json:"max_dimension,omitempty"`
	Format       string `json:"format,omitempty"` // "jpeg" (default) or "webp"

	// computed (CAP-F14, 2026-07-12): this field's OWN value is never
	// stored -- List/Detail compute data[SourceField] * multiplier at
	// render time, where multiplier is either the fixed Factor (a static
	// conversion ratio, e.g. kg -> g via x1000) or data[FactorField] (a
	// PER-RECORD multiplier -- CAP-F17's own per-transaction exchange
	// rate; exactly one of Factor/FactorField is meaningful, FactorField
	// wins if both are set). SourceField and FactorField (when set) must
	// both be real `number`/`money` Fields on the same Machine (validated
	// at load time).
	SourceField string  `json:"source_field,omitempty"`
	Factor      float64 `json:"factor,omitempty"`
	FactorField string  `json:"factor_field,omitempty"`

	// text (CAP-F18, 2026-07-12): non-empty AutoNumberPrefix marks this
	// field as auto-generated at Create when left blank -- "INV-0001",
	// zero-padded to AutoNumberPadding digits (0 = no padding), backed by
	// a per-(machine,field) counter (migrations/018, RecordStore.
	// NextSequence's atomic UPSERT). Still just a `text` field otherwise --
	// no new FieldType, matching CAP-F19's own "compose, don't add a type"
	// precedent.
	AutoNumberPrefix  string `json:"auto_number_prefix,omitempty"`
	AutoNumberPadding int    `json:"auto_number_padding,omitempty"`
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
	ID           string
	MachineID    string
	Rule         string // human-readable description
	Expression   ConstraintExpression
	Condition    *ConstraintExpression // nil = always applies
	Position     int
	ChangePolicy *ChangePolicy     // CAP-W07 -- compiled into Condition at load time, see compileChangePolicies
	CrossRecord  *CrossRecordCheck // CAP-C08 -- checked by handler.crossRecordViolations, never by constraint.Eval (needs storage access)
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
// Values (operator "in" only, CAP-W07) is a membership list.
type ConstraintExpression struct {
	Field      string   `json:"field,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Operator   string   `json:"operator"`
	Value      string   `json:"value,omitempty"`
	ValueField string   `json:"value_field,omitempty"`
	Values     []string `json:"values,omitempty"`
}

// ChangePolicyCreatedAtField is the synthetic, never-persisted data key
// constraint.Eval reads a record's creation time from when a Constraint's
// compiled Condition needs it (`new_records` change_policy). Never a real
// Field ID -- every real one is "fld_...".
const ChangePolicyCreatedAtField = "__created_at__"

// ChangePolicy (CAP-W07 -- brd-menata-runtime-v2.md §13 Fase 4,
// benchmarks/012-process-model-synthesis.md §6.4) declares which in-flight
// records a newly added/tightened Constraint applies to, answering the
// question neither the comparator BRD's blanket version-pinning nor this
// runtime's own pre-CAP-W07 behavior could express: "does this rule change
// reach open records, or only new ones?" Compiled at load time
// (compileChangePolicies, internal/metadata/compile.go) into the Constraint's
// own Condition -- no engine change, no version-pinned metadata cache, one
// live model. Attaches to a Constraint that has no other Condition already
// (a documented, deferred limitation -- combining with a pre-existing
// Condition would need Constraint.Conditions []ConstraintExpression, not
// built this pass, no case demands it yet).
type ChangePolicy struct {
	AppliesTo     string   `json:"applies_to"`               // "new_records" | "records_in_states" | "all_records"
	States        []string `json:"states,omitempty"`         // records_in_states only
	EffectiveFrom string   `json:"effective_from,omitempty"` // new_records only, "2006-01-02"
}

// CrossRecordCheck (CAP-C08 -- benchmarks/018-....md, case-portfolio.md Case 9) is a Constraint
// whose truth depends on OTHER records -- constraint.Eval deliberately never touches storage
// (the same boundary "unique" already respects, see handler.uniquenessViolations), so this is
// checked separately, by handler.crossRecordViolations. Kind picks which shape is active; the
// registry's own CAP-C08 note treats both as the same capability, not two:
//
//	"aggregate"       (CAP-C10, e.g. sum(debit) = sum(credit)) -- compares SUM(FieldA) against
//	                  SUM(FieldB), or a literal Value if FieldB is empty, across every
//	                  ChildMachine record whose ScopeField (a `reference` Field on ChildMachine)
//	                  points back at this record.
//	"reference_field" (CAP-C11, e.g. no posting into a closed period) -- looks up the record
//	                  THIS record's own ReferenceField (a `reference` Field on this Machine)
//	                  points to, reads its TargetField, compares against Value.
//
// Gating ("only check at Post") needs no new mechanism -- it's the Constraint's own existing
// Condition, evaluated the same way Engine.Violations already does for every other Constraint.
type CrossRecordCheck struct {
	Kind string `json:"kind"` // "aggregate" | "reference_field"

	// aggregate (CAP-C10)
	ChildMachine string `json:"child_machine,omitempty"`
	ScopeField   string `json:"scope_field,omitempty"`
	FieldA       string `json:"field_a,omitempty"`
	FieldB       string `json:"field_b,omitempty"` // omit to compare FieldA's sum against Value instead
	Operator     string `json:"operator"`          // reuses SupportedOperators; compared NUMERICALLY, not string-equal like constraint.Eval
	Value        string `json:"value,omitempty"`   // used when FieldB is empty

	// reference_field (CAP-C11)
	ReferenceField string `json:"reference_field,omitempty"`
	TargetField    string `json:"target_field,omitempty"`
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
	"in":                    true, // CAP-W07
	"on_or_after":           true, // CAP-W07
	"on_or_before":          true, // CAP-W07
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
	// ViewTypeDocument (CAP-F21, 2026-07-12) renders Config.Template (an
	// html/template source authored with {{.fld_x}}-style placeholders,
	// auto-escaped) against one record's own Data -- a merge-fields
	// document (certificate, simple invoice layout), computed at render
	// time and never stored, same "computed at render time" precedent
	// CAP-V13's report View already established. Output is HTML, not a
	// binary PDF/image -- "print to PDF" from the browser is the practical
	// stand-in for this prototype; a real PDF renderer is a deliberately
	// separate, deferred concern (swapping the final render step, not this
	// mechanism).
	ViewTypeDocument ViewType = "document"
	// ViewTypeProcessMap (CAP-W05 forward direction, Process Overlay B2)
	// renders a read-only state/transition diagram, purely derived from
	// this Machine's own Status value_list Field + its Events' CAP-E06
	// guards -- no ViewConfig fields at all, "purely presentational, no
	// new metadata concept" per the registry's own CAP-W05 row. Works
	// identically on an overlay-compiled Machine and a hand-authored one,
	// since both compile to the exact same Event/Field shape (see
	// internal/handler/processmap.go's extractProcessMap).
	ViewTypeProcessMap ViewType = "process_map"
	// ViewTypeBoard (CAP-V14 Tier 2) renders records grouped into
	// value_list lanes with drag-and-drop cross-lane move -- additive to
	// CAP-V14's own core (Up/Down buttons, still the accessible/keyboard
	// fallback on the ordinary List view, untouched by this).
	ViewTypeBoard ViewType = "board"
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
	Template    string             `json:"template,omitempty"`     // document: CAP-F21 -- html/template source, {{.fld_x}} placeholders resolved against one record's Data

	// SlaField (CAP-V17) names a `date`-typed Field whose value, on
	// list/detail, renders as an urgency-colored countdown badge (via
	// slaUrgency, internal/handler/record_crud.go) instead of the raw
	// date -- computed at render time, same precedent as CAP-F14/CAP-V13,
	// nothing stored. SlaWarningDays is the "due soon" threshold (0 = no
	// warning bucket, just overdue/ok).
	SlaField       string `json:"sla_field,omitempty"`
	SlaWarningDays int    `json:"sla_warning_days,omitempty"`

	// ResourceField (CAP-V18) names a `reference` Field -- when set, a
	// calendar/timeline View groups by (resource, date_field) instead of
	// date_field alone, extending CAP-V07's existing single-dimension
	// grouping (internal/handler/views.go's calendarTimeline) rather than a
	// new View type.
	ResourceField string `json:"resource_field,omitempty"`

	// GroupField (CAP-V14 Tier 2) names a `value_list` Field -- a "board"
	// View groups records into one lane per declared option value, and a
	// card drag/drop (POST .../board-move) rewrites this field via
	// RecordStore.MoveToLane. Narrower than Case 19's own "user-creatable
	// Lists" model: lanes are the Field's own fixed option set, not a
	// second CRUD surface -- see capability-registry.md's CAP-V14 row.
	GroupField string `json:"group_field,omitempty"`
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
