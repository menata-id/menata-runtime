package model

import "strings"

// Workspace is the top-level organizational boundary.
type Workspace struct {
	ID           string
	Name         string
	Slug         string // CAP-X14: URL-facing identifier, `/{slug}/...` -- deliberately separate from ID so a future rename never touches the workspace_id FK every other table already has
	Applications []*Application
	Holidays     []string // CAP-O06: "YYYY-MM-DD" dates this Workspace declares as non-working, consumed by CAP-A11's "N Business Days" date arithmetic
}

// Application is an independently realizable solution inside a Workspace.
type Application struct {
	ID          string
	WorkspaceID string
	Name        string
	Machines    []*Machine
	// NavigationEntries (CAP-O03 Tier 5, Phase 1) is this Application's own
	// declared navigation, if any -- empty when nothing is declared, the
	// signal internal/handler's subNavFor/AppMachines use to fall back to
	// their existing inferred listing unchanged. See NavigationEntry's own
	// doc comment.
	NavigationEntries []*NavigationEntry
	// Datasets and Queries (CR-21, composable-runtime-roadmap.md 17k) are
	// this Application's own declared Data-plane composable artifacts --
	// real, loadable metadata alongside the existing Machine/Field/View
	// inference path, not a replacement for it. See Dataset's own doc
	// comment.
	Datasets []*Dataset
	Queries  []*Query
}

// NavigationEntry (CAP-O03 Tier 5, Phase 1) is one declared, ordered,
// optionally-nested item in an Application's own navigation -- real
// metadata, not inferred from Machine/View structure the way CAP-O03
// Tiers 2-4 all are. Modeled on Drupal's Menu/MenuLinkContent split
// (benchmarks/009-in-app-navigation-benchmark.md): ParentID gives
// arbitrary-depth nesting (a "group" entry with no target of its own,
// followed by its own children), Position orders siblings, same role as
// Field.Position.
//
// Phase 1 supports TargetType "machine"/"view"/"group" only. "url"
// (external links) is a reserved Phase 2 column+type, rejected at load
// time today (internal/metadata/validate.go) rather than silently
// accepted and never rendered.
type NavigationEntry struct {
	ID            string
	ApplicationID string
	ParentID      string // "" = top-level
	Position      int
	Label         string
	TargetType    NavigationTargetType
	TargetMachine string // set only when TargetType == machine
	TargetView    string // set only when TargetType == view
	TargetURL     string // Phase 2, reserved -- see doc comment above
}

// NavigationTargetType is what a NavigationEntry actually links to.
type NavigationTargetType string

const (
	NavigationTargetMachine NavigationTargetType = "machine"
	NavigationTargetView    NavigationTargetType = "view"
	NavigationTargetURL     NavigationTargetType = "url" // Phase 2, not yet supported
	NavigationTargetGroup   NavigationTargetType = "group"
)

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
	FieldTypeText     FieldType = "text"
	FieldTypeRichText FieldType = "rich_text"
	FieldTypeNumber   FieldType = "number"
	FieldTypeMoney    FieldType = "money"
	FieldTypeBoolean  FieldType = "boolean"
	FieldTypeDate     FieldType = "date"
	FieldTypeDateTime FieldType = "date_time"
	FieldTypeTime     FieldType = "time"     // CAP-F10, 2026-07-12
	FieldTypeDuration FieldType = "duration" // CAP-F10, 2026-07-12
	FieldTypeUser     FieldType = "user"
	// FieldTypeGroup (CAP-F24, 2026-09-07) is reference sugar over CAP-O07's
	// groups table -- same posture as FieldTypeUser's own sugar over
	// CAP-O01's users (CAP-F05): stores a Group's own database-generated
	// UUID, renders as a picker populated from GroupStore, resolved to a
	// display name at Detail/List time. Unlike `reference`, never needs
	// Options.TargetMachine -- a Group isn't a Machine, same reason
	// FieldOptions.RestrictToGroup (CAP-F23) already names a Group by NAME
	// rather than validating it at load time like a real target_machine.
	FieldTypeGroup     FieldType = "group"
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

	// RestrictToGroup (CAP-F23) narrows a `user` field's own candidate
	// picker to a Group's membership -- a Group NAME, not id (Groups have
	// no stable, human-authored id; they're only ever created at runtime
	// via /admin/groups, DB-generated UUID, see internal/store/
	// group_store.go's own GetByName doc comment). Deliberately NOT
	// validated at metadata load time, unlike TargetMachine -- a Group is
	// independent runtime data that may not exist yet when this loads, or
	// may be created/renamed long after. A name that resolves to no real
	// Group at request time degrades to the unrestricted candidate list,
	// same "prototype-honest heuristic, graceful degrade" posture this
	// codebase already uses elsewhere (see displayLabel's own doc
	// comment) -- this narrows picker SUGGESTIONS only, it is not itself
	// an authorization boundary; CAP-P02/the Permission's own role check
	// still gate the real action regardless of what the picker offered.
	RestrictToGroup string `json:"restrict_to_group,omitempty"`

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

	// computed (CAP-F14 completion, 2026-09-07, CAP-C13's own expression
	// layer, internal/expr): Expression, when set, REPLACES the
	// SourceField*multiplier calculation above entirely with a general CEL
	// expression -- any arithmetic, string, or conditional combination of
	// `record`'s own fields, not just one multiply. SourceField/Factor/
	// FactorField stay exactly as they were ("keep as sugar over the same
	// evaluator" per capability-registry.md's own CAP-F14 row) for every
	// computed Field declared before this existed; a Field never sets both
	// -- Expression wins if it's non-empty. Not validated against a
	// specific return type at load time (CEL itself is dynamically typed);
	// a runtime type mismatch (e.g. concatenating a string where a number
	// was expected) is a render-time error, degrading to a blank rendered
	// value (formatComputedField's own doc comment), not a 500.
	Expression string `json:"expression,omitempty"`

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

	// ActionCompositeSignature (CAP-F22): opens an existing uploaded PDF
	// (or the previously-composited output, if a prior approval already
	// produced one) and stamps a signature image onto it at declared
	// (page, x%, y%) coordinates, writing the result as a new stored file.
	ActionCompositeSignature ActionType = "composite_pdf_signature" // CAP-F22
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
// Expression (CAP-C13, operator "expression" only): a raw CEL expression
// (internal/expr) replacing the plain field/operator/value triple entirely
// -- every other field on this struct is ignored when Operator is
// "expression". Every operator ABOVE this one keeps working completely
// unchanged ("sugar" over the same condition-checking job, per CAP-C13's
// own registry row) -- no existing Constraint/Event/View metadata needs to
// change for this capability to exist.
type ConstraintExpression struct {
	Field      string   `json:"field,omitempty"`
	Fields     []string `json:"fields,omitempty"`
	Operator   string   `json:"operator"`
	Value      string   `json:"value,omitempty"`
	ValueField string   `json:"value_field,omitempty"`
	Values     []string `json:"values,omitempty"`
	Expression string   `json:"expression,omitempty"`
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
	"expression":            true, // CAP-C13 -- see ConstraintExpression's own doc comment
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
// DynamicActor (CAP-F24, 2026-09-07): a per-record ALTERNATIVE to
// OwnerField's static single-Field gate — see that type's own doc comment.
// A Permission may declare both at once: internal/permission/guard.go's
// CanTrigger treats DynamicActor as the primary check whenever the
// record's own ActorTypeField resolves to a recognized value ("User" or
// "Group"), falling back to OwnerField for any record that doesn't (one
// created before this feature existed, or one that simply never set it) —
// this is what let Approval Step adopt CAP-F24 without invalidating
// already-seeded Steps or already-written conformance tests that only
// ever set the original static field.
// CanRead/CanCreate/CanEdit (CAP-P05): CRUD-level permission, independent of
// Events. A role with no Permission row at all on a machine has none of
// these — deny-by-default.
type Permission struct {
	ID           string
	MachineID    string
	Role         string
	Events       []string // event ids
	OwnerField   string
	DynamicActor *DynamicActorGate
	CanRead      bool
	CanCreate    bool
	CanEdit      bool
	CanDelete    bool     // CAP-R03 -- defaults false at the DB level, unlike the other three
	HiddenFields []string // CAP-P06 -- field ids this role's Permission excludes from List/Detail/Form rendering
}

// DynamicActorGate (CAP-F24) names three Fields on the SAME Machine:
// ActorTypeField is a `value_list` Field whose per-record VALUE ("User" or
// "Group") selects which of ActorUserField (`user`-typed) or
// ActorGroupField (`group`-typed, FieldTypeGroup) is THIS record's own real
// approver — resolved at Approve/Reject time, never at Machine-design
// time, unlike a plain OwnerField which names the same Field for every
// record. The unselected one of the two candidate Fields simply stays
// null on that record — same "declared but not this record's concern"
// pattern CAP-F17's own currency Fields already use. Validated at load
// time (metadata/validate.go): all three Fields must exist on the Machine
// and be the right Type.
type DynamicActorGate struct {
	ActorTypeField  string
	ActorUserField  string
	ActorGroupField string
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
	// ViewTypeCoordPlacement (CAP-V21) renders a preview of another
	// Machine's own `file` field (a page image or PDF) with one draggable
	// pin, writing (page, x%, y%) back to THIS record's own declared
	// fields on drop -- not signature-specific, "mark a point on an image
	// and store where" generically (capability-registry.md's CAP-V21 row).
	// Read-only (no drag) whenever the acting role has no CanEdit on this
	// Machine -- the same component covers both modes, no second View
	// type or route.
	ViewTypeCoordPlacement ViewType = "coord_placement"
	// ViewTypeDecisionStepper (CAP-V20) renders an ordered done/current/
	// pending progress indicator over THIS record's own child records
	// (found via Machine.Config's existing steps_machine/steps_parent_
	// field, CAP-X03 -- seeded since Case 3's original build but unread by
	// any code until this), with the CURRENT child's own real Approve/
	// Reject buttons -- reuses PermittedEventsForRecord and the existing
	// event-trigger route verbatim, no new POST route or write mechanism.
	// Purely presentational, same "computed at render time, no new
	// metadata concept" precedent as CAP-V13/CAP-W05.
	ViewTypeDecisionStepper ViewType = "decision_stepper"
	// ViewTypePage (CAP-V10 Tier 2, 2026-09-09) is a collection-level View
	// whose ENTIRE body is composed from other Views/static content, via
	// the SAME Config.Children field CAP-V20 Tier 2 already uses for
	// record-level embedding -- named as the intended convergence point
	// when this admission landed (embed.go's own renderEmbeddedViews doc
	// comment). No host record here (unlike CAP-V20 Tier 2): each Children
	// entry is either {view: <id>} (an existing collection-level View,
	// resolved and dispatched by ITS OWN Type, same "look at the View's
	// own declared type" principle every dispatch in this runtime already
	// follows) or {content: {...}} (PageContent, a small closed static-
	// content vocabulary for section framing text that isn't backed by any
	// View). PageEmbeddableViewTypes is this level's own allow-list --
	// deliberately separate from EmbeddableChildViewTypes (record-level):
	// the two lists don't have to agree, and don't today (list/dashboard
	// here vs decision_stepper/coord_placement there).
	ViewTypePage ViewType = "page"
)

// PageEmbeddableViewTypes (CAP-V10 Tier 2) is every View Type a `page`
// View's own Children may reference -- the load-time half of this contract
// (metadata/validate.go) and the render-time half (internal/handler/page.go's
// renderPageChild, a type switch) must stay in sync, same discipline
// EmbeddableChildViewTypes already established for CAP-V20 Tier 2's own
// record-level Children.
var PageEmbeddableViewTypes = map[ViewType]bool{
	ViewTypeList:      true,
	ViewTypeDashboard: true,
}

// PageContent (CAP-V10 Tier 2) is one static-content Children entry --
// closed vocabulary (Type: "heading" | "text" | "button" | "image"), named
// in this capability's own admission scope as the thing a composed page
// needs alongside real embedded Views to frame/caption a section, or to
// honestly stand in for a section with no real data source yet (e.g. an
// activity feed with no View to back it). Required sub-fields depend on
// Type, checked at load time: heading/text need Text; button needs Text
// and Href; image needs Src.
type PageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Href string `json:"href,omitempty"`
	Src  string `json:"src,omitempty"`
}

// ViewConfig holds view-specific presentation configuration.
type ViewConfig struct {
	Fields      []string           `json:"fields,omitempty"`       // form: ordered field ids
	Columns     []string           `json:"columns,omitempty"`      // list/calendar/timeline: visible column field ids
	Display     string             `json:"display,omitempty"`      // list: CAP-V02 Tier 2 -- "" (table, default) | "cards" (RecordSummaryCard-shaped rows, same Columns underneath)
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

	// CoordPlacement (CAP-V21) configures a "coord_placement" View -- see
	// CoordPlacementConfig's own doc comment.
	CoordPlacement *CoordPlacementConfig `json:"coord_placement,omitempty"`

	// DecisionStepper (CAP-V20) configures a "decision_stepper" View -- see
	// DecisionStepperConfig's own doc comment.
	DecisionStepper *DecisionStepperConfig `json:"decision_stepper,omitempty"`

	// Children (CAP-V20 Tier 2, 2026-09-07 -- corrected same day from a
	// first version, ParentStepperView, that hardcoded "decision_stepper"
	// into this View's own config key) declares that THIS View should
	// also render, inline alongside its own primary content, one or more
	// OTHER Views named by id. Declaring the composition is all this key
	// does -- it carries no opinion about WHAT KIND of View it's
	// embedding. What a given Child actually IS, and how to render it, is
	// resolved entirely from THAT View's own declared Type at render time
	// (internal/handler/embed.go's renderChildView, a type switch),
	// exactly the same "look at the View's own type, not a capability-
	// specific config key" principle every other View-serving code path in
	// this runtime already follows for a page's own PRIMARY view. Today
	// only `decision_stepper` is a supported child type (Approval Step's
	// own Detail embeds Approval Document's decision-stepper progress,
	// closing the exact screen split document-approval.html's own mockup
	// never had) -- adding a second embeddable type is additive there
	// (one more `case`), never a redesign of this field or its own
	// validation. Empty (the default) renders exactly as before this
	// existed. Every entry validated at load time (metadata/validate.go):
	// must name a real View, of a Type this runtime currently knows how
	// to embed.
	Children []ChildViewRef `json:"children,omitempty"`

	// ChildLinesTemplate (CAP-V28) configures a form view's own ChildLines
	// section to pre-fill from a matching record of a small companion
	// "template" Machine the instant TriggerField's value is picked on
	// Create -- e.g. Document Type = Contract pre-fills the saved Contract
	// approval flow. Still fully editable per record afterward, never
	// enforced -- see ChildLinesTemplateConfig's own doc comment. Requires
	// ChildLines to also be set on this same View (validated at load
	// time): a template has nothing to pre-fill without a ChildLines
	// section already declaring the row shape.
	ChildLinesTemplate *ChildLinesTemplateConfig `json:"child_lines_template,omitempty"`
}

// ChildViewRef (CAP-V20 Tier 2, extended by CAP-V10 Tier 2, 2026-09-09) is
// one entry in a View's own Config.Children -- see that field's own doc
// comment for the full reasoning. View names WHICH View the host wants,
// never how to interpret it -- CAP-V20 Tier 2's own record-level use only
// ever sets this. Content/Title/Layout are CAP-V10 Tier 2's own additions,
// meaningful only on a `page` View's own Children (record-level embedding
// ignores them): Content is set INSTEAD of View for a static-content entry
// (exactly one of the two is set, checked at load time); Title optionally
// overrides the referenced View's own Name as this section's header; Layout
// is "" (full width, the default) or "main"/"aside" -- two consecutive
// entries declaring "main" then "aside" render as one 2/3+1/3 grid row
// instead of stacking, the composed-page layout shape
// composable-view-proposal-reconciliation.md §8(ii) named as needed.
// Component/DatasetID/Properties/Bindings (CR-21, 17k) are a THIRD kind
// of Children entry, alongside View and Content: "render this slot as
// generic Component X, bound to declared Dataset Y" -- the Experience-
// plane counterpart to Dataset's own Data-plane closure, deliberately
// extending this existing, proven mechanism rather than adding a
// competing Layout/Slot table (see composable-runtime-roadmap.md 17k's
// own Context for why). Component names one of
// internal/composable.ComponentType's closed set (checked by
// composable.ResolveComponent at lowering time, not duplicated here).
// DatasetID must name a real Dataset on this Application (checked at
// load time, internal/metadata/validate.go). Exactly one of
// View/Content/Component is set per entry.
type ChildViewRef struct {
	View       string            `json:"view,omitempty"`
	Content    *PageContent      `json:"content,omitempty"`
	Title      string            `json:"title,omitempty"`
	Layout     string            `json:"layout,omitempty"`
	Component  string            `json:"component,omitempty"`
	DatasetID  string            `json:"dataset_id,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	Bindings   []Binding         `json:"bindings,omitempty"`
}

// Binding (CR-21) declares one Experience-plane data binding on a
// Component Children entry -- Target names the Component property (or
// a well-known slot) receiving the value, Source names the Dataset
// Dimension/Measure id supplying it. Mirrors
// internal/composable.Binding's shape without importing that package --
// internal/model never imports internal/composable (Gate 5's own
// direction runs the other way: composable is the pure leaf).
type Binding struct {
	Target string `json:"target"`
	Source string `json:"source"`
}

// EmbeddableChildViewTypes is every View Type a View's own Config.Children
// may currently reference -- the single source of truth both
// metadata/validate.go (load-time "Unknown = explicit" check) and
// internal/handler/embed.go (the actual render dispatch, a type switch)
// read, so the two can never silently drift apart: validation can never
// promise a Type the renderer doesn't handle, and the renderer never
// handles a Type validation didn't already vet. Adding a second embeddable
// Type (today ViewTypeDecisionStepper and ViewTypeCoordPlacement) means
// adding it here AND a new `case` in embed.go's own switch -- both, on
// purpose, not either alone. ViewTypeCoordPlacement (2026-09-07) is the
// second real consumer of this whole mechanism -- Approval Step's own
// Detail page also embeds its own signature-position pin inline
// (document-approval.html's own "Your Signature Position" panel,
// previously only reachable via a separate DetailLink to `/place`), the
// case that turns Children from a plausible-but-single-user generalization
// into one two independent Types actually share.
var EmbeddableChildViewTypes = map[ViewType]bool{
	ViewTypeDecisionStepper: true,
	ViewTypeCoordPlacement:  true,
}

// DecisionStepperConfig (CAP-V20) declares a "decision_stepper" View.
// SequenceField/DecisionField name Fields on the CHILD machine (found via
// THIS Machine's own Config["steps_machine"]/["steps_parent_field"], CAP-X03
// -- not duplicated here since that's already how CAP-A07/A08's own
// sequential-guard code locates a Document's Steps). SequenceField orders
// the stepper; DecisionField's value ("Pending" vs anything else) drives
// the done/current/pending state computed at render time.
type DecisionStepperConfig struct {
	SequenceField string `json:"sequence_field"`
	DecisionField string `json:"decision_field"`
}

// CoordPlacementConfig (CAP-V21) declares a "coord_placement" View:
// ReferenceField names a `reference` Field on THIS record pointing to
// another record whose PreviewField (a `file` Field on that OTHER Machine)
// is what gets shown -- served via the existing public /files/{key} route
// (CAP-F06), no new file-serving mechanism. PageField/XField/YField name
// three `number` Fields on THIS record that the pin's dropped position
// writes to. All five are validated at load time to name real Fields (see
// metadata/loader.go), same "Unknown = explicit" discipline as
// ReportConfig/ChildLinesConfig.
type CoordPlacementConfig struct {
	ReferenceField string `json:"reference_field"`
	PreviewField   string `json:"preview_field"`
	PageField      string `json:"page_field"`
	XField         string `json:"x_field"`
	YField         string `json:"y_field"`
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
	// Expression (CAP-C13, operator "expression" only) mirrors
	// ConstraintExpression's own field of the same name -- a raw CEL
	// expression replacing Field/Value entirely for this one filter clause.
	Expression string `json:"expression,omitempty"`
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

// ChildLinesTemplateConfig (CAP-V28) is a "category → saved config →
// prefilled instance" mechanism, the same three-layer shape DocuSign/Adobe
// Sign reusable envelope Templates and Salesforce Approval Process
// definitions (keyed by Record Type) converge on independently.
// TriggerField names a Field on THIS (host) Machine -- when its value
// changes on the Create form, a lookup fires. TemplateMachine is where
// saved templates live; MatchField names the Field on TemplateMachine
// compared against TriggerField's new value (one matching template
// assumed per value, not enforced -- the first match found wins, same
// "named not silently dropped" scope cut CAP-F16's own doc comment uses
// elsewhere). ChildMachine/ChildParentField locate that matched template's
// own saved rows (a `reference` Field on ChildMachine pointing back at the
// template record) -- the same shape ChildLinesConfig itself already uses
// for the host's OWN child rows, one layer up. ChildSequenceField (a Field
// on ChildMachine) orders those saved rows before they're mapped onto the
// host's row slots 0..N. ChildFieldMap maps each host ChildLines.Fields id
// to the ChildMachine Field id supplying its prefill value -- a name-keyed
// map rather than parallel-ordered slices, so entries stay legible on
// their own regardless of either side's own field ordering.
type ChildLinesTemplateConfig struct {
	TriggerField       string            `json:"trigger_field"`
	TemplateMachine    string            `json:"template_machine"`
	MatchField         string            `json:"match_field"`
	ChildMachine       string            `json:"child_machine"`
	ChildParentField   string            `json:"child_parent_field"`
	ChildSequenceField string            `json:"child_sequence_field"`
	ChildFieldMap      map[string]string `json:"child_field_map"`
}

// SortConfig defines the default sort order for a list view.
type SortConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // asc | desc
}

// Dataset (CR-21, composable-runtime-roadmap.md 17k) declares a
// composable Data-plane source directly in metadata -- a base Machine
// plus the Relations/Dimensions/Measures it exposes -- rather than only
// ever inferring one from a View's own Config (BuildDatasetFromView) or
// a Report/Dashboard section (BuildDatasetFromReport/
// BuildDatasetFromDashboardSection, internal/composable/dataset.go).
// BuildDatasetFromDeclaredDataset converts this declared shape into the
// exact same internal/composable.Dataset those inferred paths already
// produce -- one convergent target shape, two sources.
//
// Same typed-columns-plus-one-JSONB-blob pattern as View/Field: Config
// is unmarshaled wholesale from the `config` column.
type Dataset struct {
	ID            string
	ApplicationID string
	BaseMachineID string
	Name          string
	Position      int
	Config        DatasetConfig
}

// DatasetConfig is a Dataset's own `config` JSONB column.
type DatasetConfig struct {
	Relations  []DatasetRelation  `json:"relations,omitempty"`
	Dimensions []DatasetDimension `json:"dimensions,omitempty"`
	Measures   []DatasetMeasure   `json:"measures,omitempty"`
}

// DatasetRelation names one reference-typed Field on the Dataset's own
// BaseMachineID that joins to another Machine -- Via must name a real
// Field of type "reference" on that base Machine (checked at load time,
// internal/metadata/validate.go's validateDatasets).
type DatasetRelation struct {
	ID  string `json:"id"`
	Via string `json:"via"`
}

// DatasetDimension names one real Field (on the base Machine, or reached
// through a declared Relation) usable as a GroupBy/projection dimension.
type DatasetDimension struct {
	ID    string `json:"id"`
	Field string `json:"field"`
}

// DatasetMeasure names one aggregate over a real Field. Aggregate must be
// "sum" or "count" -- the only internal/composable.MeasureKind values
// that exist today (internal/composable/data.go); no avg/min/max.
type DatasetMeasure struct {
	ID        string `json:"id"`
	Aggregate string `json:"aggregate"`
	Field     string `json:"field,omitempty"` // omitted for "count"
}

// Query (CR-21) declares one named, reusable projection/filter/sort over
// a Dataset -- the declarable counterpart to a list View's own
// Columns/Filter/DefaultSort, but scoped to a Dataset rather than a
// Machine directly, so it can also select across a Dataset's own
// Dimensions/Measures.
type Query struct {
	ID        string
	DatasetID string
	Name      string
	Position  int
	Config    QueryConfig
}

// QueryConfig is a Query's own `config` JSONB column. Projection entries
// must each name a real Dimension or Measure id on the owning Dataset
// (checked at load time). Filter/Sort reuse FilterCondition/SortConfig
// verbatim -- a Query's own filter/sort concept is identical in shape to
// a list View's, just applied over a Dataset instead of a Machine.
type QueryConfig struct {
	Projection []string         `json:"projection,omitempty"`
	Filter     *FilterCondition `json:"filter,omitempty"`
	Sort       *SortConfig      `json:"sort,omitempty"`
}
