package metadata

import (
	"fmt"
	"time"

	"menata.id/app/internal/expr"
	"menata.id/app/internal/model"
)

// validateOperators (CAP-X05) rejects any Constraint expression/condition or
// Event condition naming an operator this runtime doesn't actually
// implement. Before this existed, an unrecognized operator (a typo, or a
// genuinely unsupported one like "greater_than_or_equal" before CAP-C05
// added it) didn't error anywhere — constraint.Eval's default case returns
// true ("satisfied") for any operator it doesn't recognize, so the
// constraint or guard silently never fired. That's a metadata-authoring
// trap this catches at load time instead: "Unknown = explicit"
// (capability-lifecycle.md §4 rule 3), same discipline as
// validateReferences' dangling-reference check.
//
// CAP-C13: an "expression" operator is additionally compiled here (not just
// recognized) -- expr.Compile catches a CEL syntax error at LoadAll time,
// the same "fails the load, not a random later request" discipline this
// function's own header already states for an unrecognized operator. A
// runtime-only failure (a reference to a field this specific record doesn't
// have) can't be caught here -- that's internal/expr's own fail-closed
// contract at evaluation time instead.
func validateOperators(workspaces []*model.Workspace) error {
	checkExpr := func(ce *model.ConstraintExpression, ctx string) error {
		if ce == nil {
			return nil
		}
		if !model.SupportedOperators[ce.Operator] {
			return fmt.Errorf("%s: unrecognized operator %q", ctx, ce.Operator)
		}
		if ce.Operator == "expression" {
			if err := expr.Compile(ce.Expression); err != nil {
				return fmt.Errorf("%s: invalid expression %q: %w", ctx, ce.Expression, err)
			}
		}
		return nil
	}
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				for _, c := range m.Constraints {
					if err := checkExpr(&c.Expression, fmt.Sprintf("constraint %s on machine %s", c.ID, m.ID)); err != nil {
						return err
					}
					if err := checkExpr(c.Condition, fmt.Sprintf("constraint %s's condition on machine %s", c.ID, m.ID)); err != nil {
						return err
					}
				}
				for _, e := range m.Events {
					if err := checkExpr(e.Condition, fmt.Sprintf("event %s's condition on machine %s", e.ID, m.ID)); err != nil {
						return err
					}
					if ac := e.AggregateCondition; ac != nil && !model.SupportedOperators[ac.Operator] {
						return fmt.Errorf("event %s's aggregate_condition on machine %s: unrecognized operator %q", e.ID, m.ID, ac.Operator)
					}
				}
				for _, v := range m.Views {
					for _, fc := range v.Config.Filter {
						ce := model.ConstraintExpression{Field: fc.Field, Operator: fc.Operator, Value: fc.Value, Expression: fc.Expression}
						if err := checkExpr(&ce, fmt.Sprintf("view %s's filter on machine %s", v.ID, m.ID)); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

// validateReferences enforces CAP-F13's load-time contract: every `reference`
// field's target_machine must resolve to a Machine that actually exists.
// Unlike a runtime surprise (a broken picker, a 500 on first use), a dangling
// reference is reported explicitly at load time — "Unknown = explicit"
// (capability-lifecycle.md §4 rule 3).
//
// Also enforces CAP-F05/CAP-P02's own version of the same discipline: a
// `permissions.owner_field` names a Field expected to hold a real person's
// account id (Guard.CanTrigger compares it against the acting user's id,
// internal/permission/guard.go) — that only means anything if the named
// Field is actually type `user`. Pointing owner_field at, say, a `text`
// field would silently never match anyone; caught here instead of
// discovered as "nobody can ever trigger this event."
func validateReferences(workspaces []*model.Workspace) error {
	known := make(map[string]bool)
	machineByID := make(map[string]*model.Machine)
	// CAP-I01: which Machine a given Event id belongs to -- a Subscription
	// names a PublisherEventID that can be on ANY Machine, not just this
	// one, so validating it (and its Contract's own field names) needs a
	// global index, the same reasoning machineByID already exists for.
	eventMachine := make(map[string]*model.Machine)
	// CAP-V20 Tier 2: parent_stepper_view names a View id that may belong
	// to a different Machine than the one declaring it -- same
	// cross-machine-by-id need eventMachine already exists for.
	viewByID := make(map[string]*model.View)
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				known[m.ID] = true
				machineByID[m.ID] = m
				for _, e := range m.Events {
					eventMachine[e.ID] = m
				}
				for _, v := range m.Views {
					viewByID[v.ID] = v
				}
			}
		}
	}
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				for _, f := range m.Fields {
					if f.Type != model.FieldTypeReference {
						continue
					}
					target := f.Options.TargetMachine
					if target == "" {
						return fmt.Errorf("field %s (%s) on machine %s: type reference requires target_machine",
							f.ID, f.Name, m.ID)
					}
					if !known[target] {
						return fmt.Errorf("field %s (%s) on machine %s: dangling reference — target_machine %q does not exist",
							f.ID, f.Name, m.ID, target)
					}
				}

				fieldByID := make(map[string]*model.Field, len(m.Fields))
				for _, f := range m.Fields {
					fieldByID[f.ID] = f
				}
				for _, p := range m.Permissions {
					if p.OwnerField == "" {
						continue
					}
					f, ok := fieldByID[p.OwnerField]
					if !ok {
						return fmt.Errorf("permission %s on machine %s: owner_field %q does not name a Field on this machine",
							p.ID, m.ID, p.OwnerField)
					}
					if f.Type != model.FieldTypeUser {
						return fmt.Errorf("permission %s on machine %s: owner_field %q must be type \"user\", got %q",
							p.ID, m.ID, p.OwnerField, f.Type)
					}
				}

				// CAP-F24: a DynamicActor gate is all-or-nothing -- a
				// partial triple (the loader keeps whichever columns were
				// actually set, metadata/loader.go's own doc comment)
				// fails loudly here rather than silently checking only the
				// fields that happen to be present. Each of the three must
				// also name a real Field on THIS machine, of the right
				// Type -- same "Unknown = explicit" discipline as
				// owner_field just above.
				for _, p := range m.Permissions {
					da := p.DynamicActor
					if da == nil {
						continue
					}
					if da.ActorTypeField == "" || da.ActorUserField == "" || da.ActorGroupField == "" {
						return fmt.Errorf("permission %s on machine %s: dynamic actor gate is incomplete -- actor_type_field, actor_user_field, and actor_group_field must all be set together",
							p.ID, m.ID)
					}
					typeField, ok := fieldByID[da.ActorTypeField]
					if !ok {
						return fmt.Errorf("permission %s on machine %s: actor_type_field %q does not name a Field on this machine", p.ID, m.ID, da.ActorTypeField)
					}
					if typeField.Type != model.FieldTypeValueList {
						return fmt.Errorf("permission %s on machine %s: actor_type_field %q must be type \"value_list\", got %q", p.ID, m.ID, da.ActorTypeField, typeField.Type)
					}
					userField, ok := fieldByID[da.ActorUserField]
					if !ok {
						return fmt.Errorf("permission %s on machine %s: actor_user_field %q does not name a Field on this machine", p.ID, m.ID, da.ActorUserField)
					}
					if userField.Type != model.FieldTypeUser {
						return fmt.Errorf("permission %s on machine %s: actor_user_field %q must be type \"user\", got %q", p.ID, m.ID, da.ActorUserField, userField.Type)
					}
					groupField, ok := fieldByID[da.ActorGroupField]
					if !ok {
						return fmt.Errorf("permission %s on machine %s: actor_group_field %q does not name a Field on this machine", p.ID, m.ID, da.ActorGroupField)
					}
					if groupField.Type != model.FieldTypeGroup {
						return fmt.Errorf("permission %s on machine %s: actor_group_field %q must be type \"group\", got %q", p.ID, m.ID, da.ActorGroupField, groupField.Type)
					}
				}

				// CAP-C08: a Constraint's cross_record must resolve cleanly --
				// same "wait until everything's loaded" reasoning as CAP-W01's
				// requirement-target check right below (a ChildMachine or a
				// reference_field's target machine may not have loaded yet if
				// checked any earlier).
				for _, c := range m.Constraints {
					cr := c.CrossRecord
					if cr == nil {
						continue
					}
					if !model.SupportedOperators[cr.Operator] {
						return fmt.Errorf("constraint %s on machine %s: cross_record operator %q is not supported", c.ID, m.ID, cr.Operator)
					}
					switch cr.Kind {
					case "aggregate":
						child, ok := machineByID[cr.ChildMachine]
						if !ok {
							return fmt.Errorf("constraint %s on machine %s: cross_record aggregate child_machine %q does not exist", c.ID, m.ID, cr.ChildMachine)
						}
						childFieldByID := make(map[string]*model.Field, len(child.Fields))
						for _, cf := range child.Fields {
							childFieldByID[cf.ID] = cf
						}
						if sf, ok := childFieldByID[cr.ScopeField]; !ok || sf.Type != model.FieldTypeReference || sf.Options.TargetMachine != m.ID {
							return fmt.Errorf("constraint %s on machine %s: cross_record scope_field %q must be a reference field on %s pointing back at %s", c.ID, m.ID, cr.ScopeField, cr.ChildMachine, m.ID)
						}
						if fa, ok := childFieldByID[cr.FieldA]; !ok || fa.Type != model.FieldTypeNumber {
							return fmt.Errorf("constraint %s on machine %s: cross_record field_a %q must be a number field on %s", c.ID, m.ID, cr.FieldA, cr.ChildMachine)
						}
						if cr.FieldB != "" {
							if fb, ok := childFieldByID[cr.FieldB]; !ok || fb.Type != model.FieldTypeNumber {
								return fmt.Errorf("constraint %s on machine %s: cross_record field_b %q must be a number field on %s", c.ID, m.ID, cr.FieldB, cr.ChildMachine)
							}
						}
					case "reference_field":
						refField, ok := fieldByID[cr.ReferenceField]
						if !ok || refField.Type != model.FieldTypeReference {
							return fmt.Errorf("constraint %s on machine %s: cross_record reference_field %q must be a reference field on this machine", c.ID, m.ID, cr.ReferenceField)
						}
						target, ok := machineByID[refField.Options.TargetMachine]
						if !ok {
							return fmt.Errorf("constraint %s on machine %s: cross_record reference_field %q targets a machine that does not exist", c.ID, m.ID, cr.ReferenceField)
						}
						found := false
						for _, tf := range target.Fields {
							if tf.ID == cr.TargetField {
								found = true
								break
							}
						}
						if !found {
							return fmt.Errorf("constraint %s on machine %s: cross_record target_field %q does not name a Field on %s", c.ID, m.ID, cr.TargetField, target.ID)
						}
					default:
						return fmt.Errorf("constraint %s on machine %s: cross_record has unrecognized kind %q", c.ID, m.ID, cr.Kind)
					}
				}

				// CAP-W01 (Process Overlay B3): a Requirement's target must
				// name a real Machine, and that Machine must itself declare
				// a `reference` Field pointing back at m -- same "Unknown =
				// explicit" discipline as CAP-F16's child_lines check below,
				// and the reason this validation waits until here rather
				// than running inside compileProcess itself: compileProcess
				// runs mid-load, before every Application (and so every
				// Machine a Requirement might target) has necessarily
				// loaded yet.
				if m.Process != nil {
					for _, t := range m.Process.Transitions {
						for _, r := range t.Requirements {
							target, ok := machineByID[r.Target]
							if !ok {
								return fmt.Errorf("machine %s: transition %q's requirement target %q does not exist", m.ID, t.Name, r.Target)
							}
							backRef := false
							for _, tf := range target.Fields {
								if tf.Type == model.FieldTypeReference && tf.Options.TargetMachine == m.ID {
									backRef = true
									break
								}
							}
							if !backRef {
								return fmt.Errorf("machine %s: transition %q's requirement target %q has no `reference` field pointing back at %s", m.ID, t.Name, r.Target, m.ID)
							}
						}
					}
				}

				// CAP-F08/F17: a `money` field must declare exactly one of
				// Currency (fixed code) or CurrencyField (a reference to
				// another field on the same record, for per-transaction
				// currency) -- same "Unknown = explicit" discipline; closes
				// the gap CAP-X05's own row named as deferred ("waits for
				// CAP-F08's real implementation").
				for _, f := range m.Fields {
					if f.Type != model.FieldTypeMoney {
						continue
					}
					hasCurrency, hasField := f.Options.Currency != "", f.Options.CurrencyField != ""
					if hasCurrency == hasField {
						return fmt.Errorf("field %s (%s) on machine %s: type money requires exactly one of options.currency or options.currency_field",
							f.ID, f.Name, m.ID)
					}
					if hasField {
						if _, ok := fieldByID[f.Options.CurrencyField]; !ok {
							return fmt.Errorf("field %s (%s) on machine %s: currency_field %q does not name a Field on this machine",
								f.ID, f.Name, m.ID, f.Options.CurrencyField)
						}
					}
				}

				// CAP-F14: a `computed` field's source_field must name a
				// real number/money Field on the same machine -- the
				// arithmetic only makes sense against a numeric value.
				// CAP-F14 completion (CAP-C13): Expression, when set,
				// REPLACES source_field/factor entirely (FieldOptions.
				// Expression's own doc comment) -- validated by compiling
				// it, the same load-time discipline validateOperators
				// already applies to Constraint/Event/View expressions,
				// not by the source_field checks below (which don't apply
				// to this Field at all in that case).
				for _, f := range m.Fields {
					if f.Type != model.FieldTypeComputed {
						continue
					}
					if f.Options.Expression != "" {
						if err := expr.Compile(f.Options.Expression); err != nil {
							return fmt.Errorf("field %s (%s) on machine %s: invalid computed expression %q: %w",
								f.ID, f.Name, m.ID, f.Options.Expression, err)
						}
						continue
					}
					sf, ok := fieldByID[f.Options.SourceField]
					if !ok {
						return fmt.Errorf("field %s (%s) on machine %s: computed source_field %q does not name a Field on this machine",
							f.ID, f.Name, m.ID, f.Options.SourceField)
					}
					if sf.Type != model.FieldTypeNumber && sf.Type != model.FieldTypeMoney {
						return fmt.Errorf("field %s (%s) on machine %s: computed source_field %q must be type \"number\" or \"money\", got %q",
							f.ID, f.Name, m.ID, f.Options.SourceField, sf.Type)
					}
					if f.Options.FactorField != "" {
						ff, ok := fieldByID[f.Options.FactorField]
						if !ok {
							return fmt.Errorf("field %s (%s) on machine %s: computed factor_field %q does not name a Field on this machine",
								f.ID, f.Name, m.ID, f.Options.FactorField)
						}
						if ff.Type != model.FieldTypeNumber && ff.Type != model.FieldTypeMoney {
							return fmt.Errorf("field %s (%s) on machine %s: computed factor_field %q must be type \"number\" or \"money\", got %q",
								f.ID, f.Name, m.ID, f.Options.FactorField, ff.Type)
						}
					}
				}

				// CAP-F18: auto_number_prefix only makes sense on a `text`
				// field -- Create's own auto-number generation (handler.go)
				// writes a formatted string, which would silently mismatch
				// any other declared type.
				for _, f := range m.Fields {
					if f.Options.AutoNumberPrefix != "" && f.Type != model.FieldTypeText {
						return fmt.Errorf("field %s (%s) on machine %s: auto_number_prefix requires type \"text\", got %q",
							f.ID, f.Name, m.ID, f.Type)
					}
				}

				// CAP-F16: a form view's child_lines must name a real child
				// Machine with a real `reference` field pointing back at
				// this parent -- same "Unknown = explicit" discipline.
				for _, v := range m.Views {
					cl := v.Config.ChildLines
					if cl == nil {
						continue
					}
					child, ok := machineByID[cl.Machine]
					if !ok {
						return fmt.Errorf("view %s on machine %s: child_lines.machine %q does not exist", v.ID, m.ID, cl.Machine)
					}
					var parentField *model.Field
					for _, cf := range child.Fields {
						if cf.ID == cl.ParentField {
							parentField = cf
							break
						}
					}
					if parentField == nil {
						return fmt.Errorf("view %s on machine %s: child_lines.parent_field %q does not name a Field on machine %s",
							v.ID, m.ID, cl.ParentField, cl.Machine)
					}
					if parentField.Type != model.FieldTypeReference || parentField.Options.TargetMachine != m.ID {
						return fmt.Errorf("view %s on machine %s: child_lines.parent_field %q must be a reference field targeting %s, got type %q targeting %q",
							v.ID, m.ID, cl.ParentField, m.ID, parentField.Type, parentField.Options.TargetMachine)
					}
				}

				// CAP-V28: a form view's child_lines_template must name a
				// real TriggerField on THIS machine, a real TemplateMachine
				// with a real MatchField, a real ChildMachine whose
				// ChildParentField is a reference field targeting
				// TemplateMachine, a real ChildSequenceField on ChildMachine,
				// and a ChildFieldMap whose keys are all names in this same
				// View's own child_lines.fields and whose values all name
				// real Fields on ChildMachine -- same "Unknown = explicit"
				// discipline as CAP-F16 above. Requires child_lines to also
				// be declared on this View (a template with nothing to
				// pre-fill is a config error, not a silently-ignored no-op).
				for _, v := range m.Views {
					clt := v.Config.ChildLinesTemplate
					if clt == nil {
						continue
					}
					if v.Config.ChildLines == nil {
						return fmt.Errorf("view %s on machine %s: child_lines_template requires child_lines to also be set on this view", v.ID, m.ID)
					}
					if _, ok := fieldByID[clt.TriggerField]; !ok {
						return fmt.Errorf("view %s on machine %s: child_lines_template.trigger_field %q does not name a Field on this machine", v.ID, m.ID, clt.TriggerField)
					}
					tmplMachine, ok := machineByID[clt.TemplateMachine]
					if !ok {
						return fmt.Errorf("view %s on machine %s: child_lines_template.template_machine %q does not exist", v.ID, m.ID, clt.TemplateMachine)
					}
					var matchField *model.Field
					for _, tf := range tmplMachine.Fields {
						if tf.ID == clt.MatchField {
							matchField = tf
							break
						}
					}
					if matchField == nil {
						return fmt.Errorf("view %s on machine %s: child_lines_template.match_field %q does not name a Field on machine %s", v.ID, m.ID, clt.MatchField, clt.TemplateMachine)
					}
					childMachine, ok := machineByID[clt.ChildMachine]
					if !ok {
						return fmt.Errorf("view %s on machine %s: child_lines_template.child_machine %q does not exist", v.ID, m.ID, clt.ChildMachine)
					}
					childFieldByID := map[string]*model.Field{}
					for _, cf := range childMachine.Fields {
						childFieldByID[cf.ID] = cf
					}
					childParentField, ok := childFieldByID[clt.ChildParentField]
					if !ok {
						return fmt.Errorf("view %s on machine %s: child_lines_template.child_parent_field %q does not name a Field on machine %s", v.ID, m.ID, clt.ChildParentField, clt.ChildMachine)
					}
					if childParentField.Type != model.FieldTypeReference || childParentField.Options.TargetMachine != clt.TemplateMachine {
						return fmt.Errorf("view %s on machine %s: child_lines_template.child_parent_field %q must be a reference field targeting %s, got type %q targeting %q",
							v.ID, m.ID, clt.ChildParentField, clt.TemplateMachine, childParentField.Type, childParentField.Options.TargetMachine)
					}
					if _, ok := childFieldByID[clt.ChildSequenceField]; !ok {
						return fmt.Errorf("view %s on machine %s: child_lines_template.child_sequence_field %q does not name a Field on machine %s", v.ID, m.ID, clt.ChildSequenceField, clt.ChildMachine)
					}
					hostChildFields := map[string]bool{}
					for _, fid := range v.Config.ChildLines.Fields {
						hostChildFields[fid] = true
					}
					for hostField, tmplField := range clt.ChildFieldMap {
						if !hostChildFields[hostField] {
							return fmt.Errorf("view %s on machine %s: child_lines_template.child_field_map key %q does not name a Field in this view's own child_lines.fields", v.ID, m.ID, hostField)
						}
						if _, ok := childFieldByID[tmplField]; !ok {
							return fmt.Errorf("view %s on machine %s: child_lines_template.child_field_map[%q] %q does not name a Field on machine %s", v.ID, m.ID, hostField, tmplField, clt.ChildMachine)
						}
					}
				}

				// CAP-V04/V05/V09: a list view's default_sort.field and every
				// filter condition's field must name a real Field on this
				// machine -- same "Unknown = explicit" discipline as
				// child_lines above, so a typo'd field id fails at load time
				// instead of silently sorting/filtering on nothing.
				// "created_at"/"updated_at" are exempt -- real columns every
				// record has, not a JSONB Field (store.RecordStore.List's
				// own reserved-name handling).
				for _, v := range m.Views {
					if ds := v.Config.DefaultSort; ds != nil && ds.Field != "" && ds.Field != "created_at" && ds.Field != "updated_at" {
						if _, ok := fieldByID[ds.Field]; !ok {
							return fmt.Errorf("view %s on machine %s: default_sort.field %q does not name a Field on this machine", v.ID, m.ID, ds.Field)
						}
					}
					for _, fc := range v.Config.Filter {
						if fc.Operator == "expression" {
							continue // CAP-C13: Expression replaces Field entirely, see FilterCondition's own doc comment
						}
						if fc.Field == "$sla_urgency" {
							// CAP-V09 Tier 2: a virtual field (CAP-V17's own
							// computed bucket), not a real Field -- exempt
							// from the fieldByID lookup below, same class of
							// exemption "expression" already gets, but it
							// DOES require the View to actually declare
							// what it's computed from, so a Filter naming
							// it on a View with no SlaField fails loudly
							// instead of silently never matching.
							if v.Config.SlaField == "" {
								return fmt.Errorf("view %s on machine %s: filter field \"$sla_urgency\" requires sla_field to also be set on this view", v.ID, m.ID)
							}
							continue
						}
						if _, ok := fieldByID[fc.Field]; !ok {
							return fmt.Errorf("view %s on machine %s: filter field %q does not name a Field on this machine", v.ID, m.ID, fc.Field)
						}
					}

					// CAP-V07: calendar/timeline's date_field is one of
					// THIS machine's own Fields (the records being
					// plotted), not another machine's.
					if v.Config.DateField != "" {
						if _, ok := fieldByID[v.Config.DateField]; !ok {
							return fmt.Errorf("view %s on machine %s: date_field %q does not name a Field on this machine", v.ID, m.ID, v.Config.DateField)
						}
					}

					// CAP-V12: every wizard step's field ids must be real
					// Fields on this machine -- same discipline as Fields
					// itself already gets nowhere else, but Steps is new.
					for si, step := range v.Config.Steps {
						for _, fid := range step {
							if _, ok := fieldByID[fid]; !ok {
								return fmt.Errorf("view %s on machine %s: steps[%d] field %q does not name a Field on this machine", v.ID, m.ID, si, fid)
							}
						}
					}

					// CAP-V13: a report view aggregates ANOTHER machine's
					// records -- group_field and every sum_field must name
					// real Fields on THAT machine, not this one.
					if rc := v.Config.Report; rc != nil {
						src, ok := machineByID[rc.Machine]
						if !ok {
							return fmt.Errorf("view %s on machine %s: report.machine %q does not exist", v.ID, m.ID, rc.Machine)
						}
						srcFields := make(map[string]bool, len(src.Fields))
						for _, sf := range src.Fields {
							srcFields[sf.ID] = true
						}
						if !srcFields[rc.GroupField] {
							return fmt.Errorf("view %s on machine %s: report.group_field %q does not name a Field on machine %s", v.ID, m.ID, rc.GroupField, rc.Machine)
						}
						for _, sf := range rc.SumFields {
							if !srcFields[sf] {
								return fmt.Errorf("view %s on machine %s: report.sum_fields %q does not name a Field on machine %s", v.ID, m.ID, sf, rc.Machine)
							}
						}
					}

					// CAP-V14 Tier 3: a board view's own group_field must
					// name a real Field on THIS machine of type value_list
					// (Tier 2's own fixed-lane mode) or reference (Tier 3's
					// dynamic-lane mode, one lane per target-machine
					// record) -- any other type has no lane concept and
					// silently rendered zero lanes before this check
					// existed. A reference group_field's own
					// target_machine must resolve to a real Machine, same
					// discipline every other reference-field validation in
					// this file already applies.
					if v.Type == model.ViewTypeBoard && v.Config.GroupField != "" {
						gf, ok := fieldByID[v.Config.GroupField]
						if !ok {
							return fmt.Errorf("view %s on machine %s: group_field %q does not name a Field on this machine", v.ID, m.ID, v.Config.GroupField)
						}
						switch gf.Type {
						case model.FieldTypeValueList:
							// Tier 2: lanes are the Field's own declared Options.Values.
						case model.FieldTypeReference:
							if _, ok := machineByID[gf.Options.TargetMachine]; !ok {
								return fmt.Errorf("view %s on machine %s: group_field %q is a reference to unknown machine %q", v.ID, m.ID, v.Config.GroupField, gf.Options.TargetMachine)
							}
						default:
							return fmt.Errorf("view %s on machine %s: group_field %q must be type value_list or reference, got %q", v.ID, m.ID, v.Config.GroupField, gf.Type)
						}
					}

					// CAP-V21: reference_field must be a real `reference`
					// Field on THIS machine; preview_field must name a real
					// Field on whatever machine that reference points to
					// (not necessarily `file` -- a workspace could legally
					// preview any field's own value, though the UI only
					// makes sense for one). page_field/x_field/y_field must
					// each name a real Field on THIS machine.
					if cp := v.Config.CoordPlacement; cp != nil {
						refField, ok := fieldByID[cp.ReferenceField]
						if !ok || refField.Type != model.FieldTypeReference {
							return fmt.Errorf("view %s on machine %s: coord_placement.reference_field %q does not name a reference Field on this machine", v.ID, m.ID, cp.ReferenceField)
						}
						target, ok := machineByID[refField.Options.TargetMachine]
						if !ok {
							return fmt.Errorf("view %s on machine %s: coord_placement.reference_field %q targets an unknown machine", v.ID, m.ID, cp.ReferenceField)
						}
						targetFields := make(map[string]bool, len(target.Fields))
						for _, tf := range target.Fields {
							targetFields[tf.ID] = true
						}
						if !targetFields[cp.PreviewField] {
							return fmt.Errorf("view %s on machine %s: coord_placement.preview_field %q does not name a Field on machine %s", v.ID, m.ID, cp.PreviewField, refField.Options.TargetMachine)
						}
						for label, fid := range map[string]string{"page_field": cp.PageField, "x_field": cp.XField, "y_field": cp.YField} {
							if _, ok := fieldByID[fid]; !ok {
								return fmt.Errorf("view %s on machine %s: coord_placement.%s %q does not name a Field on this machine", v.ID, m.ID, label, fid)
							}
						}
					}

					// CAP-V20 Tier 2: every Children entry must name a real
					// View, of a Type this runtime currently knows how to
					// embed (internal/handler/embed.go's renderChildView
					// switch -- embeddableChildTypes here must stay in
					// sync with that switch's own case list, same
					// discipline ActionType/FieldType exhaustiveness
					// checks already rely on `go build` catching
					// elsewhere). Same "Unknown = explicit" posture as
					// coord_placement above -- a typo'd or not-yet-
					// embeddable View id fails at load time, not as a
					// silently-empty card at request time.
					for _, child := range v.Config.Children {
						// CAP-V10 Tier 2: a `page` View's own Children entries
						// use a wider grammar (Content instead of View, plus
						// Title/Layout) and a DIFFERENT Type allow-list
						// (PageEmbeddableViewTypes, not EmbeddableChildViewTypes
						// -- collection-level composition, no host record) than
						// CAP-V20 Tier 2's own record-level Children below.
						if v.Type == model.ViewTypePage {
							if (child.View == "") == (child.Content == nil) {
								return fmt.Errorf("view %s on machine %s: children entry must set exactly one of view or content", v.ID, m.ID)
							}
							if child.Layout != "" && child.Layout != "main" && child.Layout != "aside" {
								return fmt.Errorf("view %s on machine %s: children entry layout %q must be \"main\" or \"aside\" (empty = full width)", v.ID, m.ID, child.Layout)
							}
							if child.Content != nil {
								pc := child.Content
								switch pc.Type {
								case "heading", "text":
									if pc.Text == "" {
										return fmt.Errorf("view %s on machine %s: content type %q requires text", v.ID, m.ID, pc.Type)
									}
								case "button":
									if pc.Text == "" || pc.Href == "" {
										return fmt.Errorf("view %s on machine %s: content type \"button\" requires text and href", v.ID, m.ID)
									}
								case "image":
									if pc.Src == "" {
										return fmt.Errorf("view %s on machine %s: content type \"image\" requires src", v.ID, m.ID)
									}
								default:
									return fmt.Errorf("view %s on machine %s: content type %q is not one of heading, text, button, image", v.ID, m.ID, pc.Type)
								}
								continue
							}
							target, ok := viewByID[child.View]
							if !ok {
								return fmt.Errorf("view %s on machine %s: children names %q, which does not name a View that exists", v.ID, m.ID, child.View)
							}
							if !model.PageEmbeddableViewTypes[target.Type] {
								return fmt.Errorf("view %s on machine %s: children names %q (type %q) -- not a Type a page View can compose", v.ID, m.ID, child.View, target.Type)
							}
							continue
						}

						// CAP-V20 Tier 2: every Children entry must name a real
						// View, of a Type this runtime currently knows how to
						// embed (internal/handler/embed.go's renderChildView
						// switch -- embeddableChildTypes here must stay in
						// sync with that switch's own case list, same
						// discipline ActionType/FieldType exhaustiveness
						// checks already rely on `go build` catching
						// elsewhere). Same "Unknown = explicit" posture as
						// coord_placement above -- a typo'd or not-yet-
						// embeddable View id fails at load time, not as a
						// silently-empty card at request time.
						target, ok := viewByID[child.View]
						if !ok {
							return fmt.Errorf("view %s on machine %s: children names %q, which does not name a View that exists", v.ID, m.ID, child.View)
						}
						if !model.EmbeddableChildViewTypes[target.Type] {
							return fmt.Errorf("view %s on machine %s: children names %q (type %q) -- not a Type this runtime knows how to embed as a child", v.ID, m.ID, child.View, target.Type)
						}
					}

					// CAP-V10: each dashboard section's machine must exist,
					// and its group_field (if any) must be a real Field on
					// THAT section's own machine.
					for si, sec := range v.Config.Sections {
						src, ok := machineByID[sec.Machine]
						if !ok {
							return fmt.Errorf("view %s on machine %s: sections[%d].machine %q does not exist", v.ID, m.ID, si, sec.Machine)
						}
						if sec.GroupField == "" {
							continue
						}
						found := false
						for _, sf := range src.Fields {
							if sf.ID == sec.GroupField {
								found = true
								break
							}
						}
						if !found {
							return fmt.Errorf("view %s on machine %s: sections[%d].group_field %q does not name a Field on machine %s", v.ID, m.ID, si, sec.GroupField, sec.Machine)
						}
					}
				}

				// CAP-P04: an Event's input_fields must name real Fields on
				// this same machine -- the picker rendered alongside the
				// trigger button only makes sense for a Field that exists.
				for _, e := range m.Events {
					for _, fid := range e.InputFields {
						if _, ok := fieldByID[fid]; !ok {
							return fmt.Errorf("event %s on machine %s: input_fields %q does not name a Field on this machine", e.ID, m.ID, fid)
						}
					}
				}

				// CAP-E02/E03: exactly one of time/date_field, and
				// date_field (when set) must name a real `date` Field on
				// this machine -- a schedule keyed on a nonexistent or
				// wrong-typed field would silently never fire, the same
				// "Unknown = explicit" discipline as everywhere else.
				for _, e := range m.Events {
					s := e.Schedule
					if s == nil {
						continue
					}
					if (s.Time == "") == (s.DateField == "") {
						return fmt.Errorf("event %s on machine %s: schedule must set exactly one of time or date_field", e.ID, m.ID)
					}
					if s.Time != "" {
						if _, err := time.Parse("15:04", s.Time); err != nil {
							return fmt.Errorf("event %s on machine %s: schedule.time %q is not HH:MM", e.ID, m.ID, s.Time)
						}
					}
					if s.DateField != "" {
						f, ok := fieldByID[s.DateField]
						if !ok {
							return fmt.Errorf("event %s on machine %s: schedule.date_field %q does not name a Field on this machine", e.ID, m.ID, s.DateField)
						}
						if f.Type != model.FieldTypeDate {
							return fmt.Errorf("event %s on machine %s: schedule.date_field %q must be type \"date\", got %q", e.ID, m.ID, s.DateField, f.Type)
						}
					}
				}

				// CAP-I01/I03: a Subscription's publisher_event_id must
				// name a real Event (anywhere -- cross-machine by design);
				// its fields mapping's target keys must be real Fields on
				// THIS (subscriber) machine; its Contract's own field
				// names must be real Fields on the PUBLISHER's machine
				// (the data a Contract checks is the publisher's, not the
				// subscriber's); on_violation must be a value the handler
				// actually understands.
				for _, sub := range m.Subscriptions {
					pubMachine, ok := eventMachine[sub.PublisherEventID]
					if !ok {
						return fmt.Errorf("subscription %s on machine %s: publisher_event_id %q does not name a real Event", sub.ID, m.ID, sub.PublisherEventID)
					}
					for target := range sub.Fields {
						if _, ok := fieldByID[target]; !ok {
							return fmt.Errorf("subscription %s on machine %s: fields target %q does not name a Field on this machine", sub.ID, m.ID, target)
						}
					}
					pubFieldByID := make(map[string]bool, len(pubMachine.Fields))
					for _, pf := range pubMachine.Fields {
						pubFieldByID[pf.ID] = true
					}
					for _, c := range sub.Contract {
						if !pubFieldByID[c.Field] {
							return fmt.Errorf("subscription %s on machine %s: contract field %q does not name a Field on publisher machine %s", sub.ID, m.ID, c.Field, pubMachine.ID)
						}
						if !model.SupportedOperators[c.Operator] {
							return fmt.Errorf("subscription %s on machine %s: contract operator %q is not supported", sub.ID, m.ID, c.Operator)
						}
					}
					if sub.OnViolation != "skip" && sub.OnViolation != "log_only" {
						return fmt.Errorf("subscription %s on machine %s: on_violation %q must be \"skip\" or \"log_only\"", sub.ID, m.ID, sub.OnViolation)
					}
				}
			}
		}
	}
	return nil
}

// navigationCollectionViewTypes (CAP-O03 Tier 5, Phase 1) are the View
// types that actually have a collection-level route (no record id needed)
// -- exactly the set the router registers under /{machineID}/{slug} rather
// than /{machineID}/{recordID}/{slug}. A NavigationEntry can only ever
// target one of these: linking to a record-scoped View (Detail, coord_
// placement, decision_stepper, document) with no record to supply would
// have nothing real to route to. See internal/router/router.go for the
// authoritative route list this mirrors.
var navigationCollectionViewTypes = map[model.ViewType]bool{
	model.ViewTypeList:       true,
	model.ViewTypeForm:       true,
	model.ViewTypeDashboard:  true,
	model.ViewTypeCalendar:   true,
	model.ViewTypeTimeline:   true,
	model.ViewTypeReport:     true,
	model.ViewTypeBoard:      true,
	model.ViewTypeProcessMap: true,
	model.ViewTypePage:       true, // CAP-V10 Tier 2, 2026-09-09 -- GET /{machineID}/page, added alongside this Type itself
}

// validateNavigationEntries (CAP-O03 Tier 5, Phase 1) enforces the same
// "Unknown = explicit" load-time discipline validateReferences already
// applies to `reference` fields -- a dangling or malformed declared-nav
// entry fails boot, not a silent empty render later.
func validateNavigationEntries(workspaces []*model.Workspace) error {
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			if len(app.NavigationEntries) == 0 {
				continue // nothing declared -- CAP-O03's own inferred listing applies, nothing to validate
			}
			machineByID := make(map[string]*model.Machine, len(app.Machines))
			for _, m := range app.Machines {
				machineByID[m.ID] = m
			}
			viewByID := make(map[string]*model.View)
			for _, m := range app.Machines {
				for _, v := range m.Views {
					viewByID[v.ID] = v
				}
			}
			entryByID := make(map[string]*model.NavigationEntry, len(app.NavigationEntries))
			for _, e := range app.NavigationEntries {
				entryByID[e.ID] = e
			}

			for _, e := range app.NavigationEntries {
				switch e.TargetType {
				case model.NavigationTargetMachine:
					if e.TargetMachine == "" {
						return fmt.Errorf("navigation entry %s (application %s): type machine requires target_machine", e.ID, app.ID)
					}
					if _, ok := machineByID[e.TargetMachine]; !ok {
						return fmt.Errorf("navigation entry %s (application %s): dangling reference — target_machine %q does not name a Machine in this Application", e.ID, app.ID, e.TargetMachine)
					}
				case model.NavigationTargetView:
					if e.TargetView == "" {
						return fmt.Errorf("navigation entry %s (application %s): type view requires target_view", e.ID, app.ID)
					}
					v, ok := viewByID[e.TargetView]
					if !ok {
						return fmt.Errorf("navigation entry %s (application %s): dangling reference — target_view %q does not name a View on any Machine in this Application", e.ID, app.ID, e.TargetView)
					}
					if !navigationCollectionViewTypes[v.Type] {
						return fmt.Errorf("navigation entry %s (application %s): target_view %q is a %q View, which has no collection-level route to link to (only list/form/dashboard/calendar/timeline/report/board/process_map are supported navigation targets)", e.ID, app.ID, e.TargetView, v.Type)
					}
				case model.NavigationTargetGroup:
					if e.TargetMachine != "" || e.TargetView != "" || e.TargetURL != "" {
						return fmt.Errorf("navigation entry %s (application %s): type group must not set target_machine/target_view/target_url", e.ID, app.ID)
					}
				case model.NavigationTargetURL:
					// Phase 2, reserved column -- rejected explicitly rather
					// than silently accepted and never rendered, per
					// "Unknown = explicit" (capability-lifecycle.md §4).
					return fmt.Errorf("navigation entry %s (application %s): type url is not supported yet (CAP-O03 Tier 5 Phase 2, not built)", e.ID, app.ID)
				default:
					return fmt.Errorf("navigation entry %s (application %s): unsupported target_type %q", e.ID, app.ID, e.TargetType)
				}

				if e.ParentID == "" {
					continue
				}
				if _, ok := entryByID[e.ParentID]; !ok {
					return fmt.Errorf("navigation entry %s (application %s): dangling reference — parent_id %q does not name another navigation entry in this Application", e.ID, app.ID, e.ParentID)
				}
				// Cycle check: walk parent_id to root: a well-formed tree
				// terminates at "" within len(entries) hops; revisiting a
				// node first means a cycle.
				visited := map[string]bool{e.ID: true}
				cur := e.ParentID
				for cur != "" {
					if visited[cur] {
						return fmt.Errorf("navigation entry %s (application %s): parent_id chain forms a cycle at %q", e.ID, app.ID, cur)
					}
					visited[cur] = true
					parent, ok := entryByID[cur]
					if !ok {
						break // already reported as a dangling reference above for that entry
					}
					cur = parent.ParentID
				}
			}
		}
	}
	return nil
}
