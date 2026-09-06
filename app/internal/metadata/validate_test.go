package metadata

import (
	"strings"
	"testing"

	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

// wsWith wraps machines in a single Workspace/Application, the shape both
// validateOperators and validateReferences take.
func wsWith(machines ...*model.Machine) []*model.Workspace {
	return []*model.Workspace{{
		ID: "ws_test",
		Applications: []*model.Application{{
			ID:       "app_test",
			Machines: machines,
		}},
	}}
}

func wantErr(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if wantSubstr == "" {
		if err != nil {
			t.Fatalf("got error %v, want nil", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("got nil error, want one containing %q", wantSubstr)
	}
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("error %q does not contain %q", err.Error(), wantSubstr)
	}
}

// --- validateOperators (CAP-X05) -----------------------------------------

func TestValidateOperators(t *testing.T) {
	expr := func(op string) model.ConstraintExpression {
		return model.ConstraintExpression{Field: "fld_a", Operator: op}
	}

	tests := []struct {
		name    string
		machine func() *model.Machine
		want    string
	}{
		{
			name: "valid constraint expression and condition",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").Build()
				cond := expr("equals")
				m.Constraints = []*model.Constraint{{ID: "c1", Expression: expr("required"), Condition: &cond}}
				return m
			},
		},
		{
			name: "nil constraint condition is skipped, not an error",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").Build()
				m.Constraints = []*model.Constraint{{ID: "c1", Expression: expr("required"), Condition: nil}}
				return m
			},
		},
		{
			name: "constraint expression has unrecognized operator",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").Build()
				m.Constraints = []*model.Constraint{{ID: "c1", Expression: expr("bogus_operator")}}
				return m
			},
			want: `constraint c1 on machine mch_a: unrecognized operator "bogus_operator"`,
		},
		{
			name: "constraint condition has unrecognized operator",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").Build()
				cond := expr("bogus_operator")
				m.Constraints = []*model.Constraint{{ID: "c1", Expression: expr("required"), Condition: &cond}}
				return m
			},
			want: `constraint c1's condition on machine mch_a: unrecognized operator "bogus_operator"`,
		},
		{
			name: "event condition has unrecognized operator",
			machine: func() *model.Machine {
				cond := expr("bogus_operator")
				return builders.Machine("mch_a").
					WithEvent(builders.Event("evt_1", "Submit").Condition(&cond).Build()).
					Build()
			},
			want: `event evt_1's condition on machine mch_a: unrecognized operator "bogus_operator"`,
		},
		{
			name: "event aggregate_condition has unrecognized operator",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").
					WithEvent(builders.Event("evt_1", "Submit").Build()).
					Build()
				m.Events[0].AggregateCondition = &model.AggregateCondition{AggregateField: "fld_a", ScopeField: "fld_b", Operator: "bogus_operator", Value: "1"}
				return m
			},
			want: `event evt_1's aggregate_condition on machine mch_a: unrecognized operator "bogus_operator"`,
		},
		{
			name: "view filter has unrecognized operator",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").Build()
				m.Views = []*model.View{{ID: "v1", Config: model.ViewConfig{Filter: []model.FilterCondition{{Field: "fld_a", Operator: "bogus_operator"}}}}}
				return m
			},
			want: `view v1's filter on machine mch_a: unrecognized operator "bogus_operator"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, validateOperators(wsWith(tt.machine())), tt.want)
		})
	}
}

// --- validateReferences: reference fields (CAP-F13) ----------------------

func TestValidateReferences_ReferenceField(t *testing.T) {
	tests := []struct {
		name     string
		machines func() []*model.Machine
		want     string
	}{
		{
			name: "valid reference to an existing machine",
			machines: func() []*model.Machine {
				target := builders.Machine("mch_target").Build()
				src := builders.Machine("mch_src").
					WithField(builders.Field("fld_ref", model.FieldTypeReference).Options(model.FieldOptions{TargetMachine: "mch_target"}).Build()).
					Build()
				return []*model.Machine{target, src}
			},
		},
		{
			name: "empty target_machine",
			machines: func() []*model.Machine {
				return []*model.Machine{
					builders.Machine("mch_src").
						WithField(builders.Field("fld_ref", model.FieldTypeReference).Build()).
						Build(),
				}
			},
			want: `field fld_ref (fld_ref) on machine mch_src: type reference requires target_machine`,
		},
		{
			name: "dangling target_machine",
			machines: func() []*model.Machine {
				return []*model.Machine{
					builders.Machine("mch_src").
						WithField(builders.Field("fld_ref", model.FieldTypeReference).Options(model.FieldOptions{TargetMachine: "mch_ghost"}).Build()).
						Build(),
				}
			},
			want: `field fld_ref (fld_ref) on machine mch_src: dangling reference — target_machine "mch_ghost" does not exist`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, validateReferences(wsWith(tt.machines()...)), tt.want)
		})
	}
}

// --- validateReferences: permission.owner_field (CAP-F05/CAP-P02) --------

func TestValidateReferences_OwnerField(t *testing.T) {
	tests := []struct {
		name    string
		machine func() *model.Machine
		want    string
	}{
		{
			name: "empty owner_field is skipped",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").Build()
				m.Permissions = []*model.Permission{{ID: "p1", Role: "Approver"}}
				return m
			},
		},
		{
			name: "owner_field names a real user field",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").
					WithField(builders.Field("fld_owner", model.FieldTypeUser).Build()).
					Build()
				m.Permissions = []*model.Permission{{ID: "p1", Role: "Approver", OwnerField: "fld_owner"}}
				return m
			},
		},
		{
			name: "owner_field does not name a Field on this machine",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").Build()
				m.Permissions = []*model.Permission{{ID: "p1", Role: "Approver", OwnerField: "fld_ghost"}}
				return m
			},
			want: `permission p1 on machine mch_a: owner_field "fld_ghost" does not name a Field on this machine`,
		},
		{
			name: "owner_field names a Field that isn't type user",
			machine: func() *model.Machine {
				m := builders.Machine("mch_a").
					WithField(builders.Field("fld_owner", model.FieldTypeText).Build()).
					Build()
				m.Permissions = []*model.Permission{{ID: "p1", Role: "Approver", OwnerField: "fld_owner"}}
				return m
			},
			want: `permission p1 on machine mch_a: owner_field "fld_owner" must be type "user", got "text"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, validateReferences(wsWith(tt.machine())), tt.want)
		})
	}
}

// --- validateReferences: cross_record aggregate (CAP-C08/CAP-C10) --------

func TestValidateReferences_CrossRecordAggregate(t *testing.T) {
	// child is a valid aggregate target for every case below unless a case
	// overrides it: a `reference` field back at mch_parent (the scope
	// field) and a `number` field (the amount being summed).
	newChild := func() *model.Machine {
		return builders.Machine("mch_child").
			WithField(builders.Field("fld_scope", model.FieldTypeReference).Options(model.FieldOptions{TargetMachine: "mch_parent"}).Build()).
			WithField(builders.Field("fld_amount", model.FieldTypeNumber).Build()).
			Build()
	}
	newParent := func(cr *model.CrossRecordCheck) *model.Machine {
		m := builders.Machine("mch_parent").Build()
		m.Constraints = []*model.Constraint{{ID: "c1", Expression: model.ConstraintExpression{Operator: "equals"}, CrossRecord: cr}}
		return m
	}

	tests := []struct {
		name  string
		child func() *model.Machine
		cr    *model.CrossRecordCheck
		want  string
	}{
		{
			name:  "valid aggregate cross_record",
			child: newChild,
			cr:    &model.CrossRecordCheck{Kind: "aggregate", ChildMachine: "mch_child", ScopeField: "fld_scope", FieldA: "fld_amount", Operator: "equals", Value: "0"},
		},
		{
			name:  "unsupported cross_record operator",
			child: newChild,
			cr:    &model.CrossRecordCheck{Kind: "aggregate", ChildMachine: "mch_child", ScopeField: "fld_scope", FieldA: "fld_amount", Operator: "bogus_operator"},
			want:  `constraint c1 on machine mch_parent: cross_record operator "bogus_operator" is not supported`,
		},
		{
			name:  "child_machine does not exist",
			child: newChild,
			cr:    &model.CrossRecordCheck{Kind: "aggregate", ChildMachine: "mch_ghost", ScopeField: "fld_scope", FieldA: "fld_amount", Operator: "equals"},
			want:  `constraint c1 on machine mch_parent: cross_record aggregate child_machine "mch_ghost" does not exist`,
		},
		{
			name:  "scope_field is not a reference back to the parent",
			child: newChild,
			cr:    &model.CrossRecordCheck{Kind: "aggregate", ChildMachine: "mch_child", ScopeField: "fld_amount", FieldA: "fld_amount", Operator: "equals"},
			want:  `constraint c1 on machine mch_parent: cross_record scope_field "fld_amount" must be a reference field on mch_child pointing back at mch_parent`,
		},
		{
			name:  "field_a is not a number field",
			child: newChild,
			cr:    &model.CrossRecordCheck{Kind: "aggregate", ChildMachine: "mch_child", ScopeField: "fld_scope", FieldA: "fld_scope", Operator: "equals"},
			want:  `constraint c1 on machine mch_parent: cross_record field_a "fld_scope" must be a number field on mch_child`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, validateReferences(wsWith(newParent(tt.cr), tt.child())), tt.want)
		})
	}
}

// --- validateReferences: cross_record reference_field (CAP-C08/CAP-C11) --

func TestValidateReferences_CrossRecordReferenceField(t *testing.T) {
	newTarget := func() *model.Machine {
		return builders.Machine("mch_period").
			WithField(builders.Field("fld_status", model.FieldTypeText).Build()).
			Build()
	}
	newSrc := func(refFieldType model.FieldType, cr *model.CrossRecordCheck) *model.Machine {
		m := builders.Machine("mch_entry").
			WithField(builders.Field("fld_period", refFieldType).Options(model.FieldOptions{TargetMachine: "mch_period"}).Build()).
			Build()
		m.Constraints = []*model.Constraint{{ID: "c1", Expression: model.ConstraintExpression{Operator: "equals"}, CrossRecord: cr}}
		return m
	}

	tests := []struct {
		name         string
		refFieldType model.FieldType
		cr           *model.CrossRecordCheck
		want         string
	}{
		{
			name:         "valid reference_field cross_record",
			refFieldType: model.FieldTypeReference,
			cr:           &model.CrossRecordCheck{Kind: "reference_field", ReferenceField: "fld_period", TargetField: "fld_status", Operator: "not_equals", Value: "Closed"},
		},
		{
			name:         "reference_field is not a reference-type field",
			refFieldType: model.FieldTypeText,
			cr:           &model.CrossRecordCheck{Kind: "reference_field", ReferenceField: "fld_period", TargetField: "fld_status", Operator: "not_equals", Value: "Closed"},
			want:         `constraint c1 on machine mch_entry: cross_record reference_field "fld_period" must be a reference field on this machine`,
		},
		{
			name:         "target_field does not exist on the target machine",
			refFieldType: model.FieldTypeReference,
			cr:           &model.CrossRecordCheck{Kind: "reference_field", ReferenceField: "fld_period", TargetField: "fld_ghost", Operator: "not_equals", Value: "Closed"},
			want:         `constraint c1 on machine mch_entry: cross_record target_field "fld_ghost" does not name a Field on mch_period`,
		},
		{
			name:         "unrecognized cross_record kind",
			refFieldType: model.FieldTypeReference,
			cr:           &model.CrossRecordCheck{Kind: "bogus_kind", Operator: "equals"},
			want:         `constraint c1 on machine mch_entry: cross_record has unrecognized kind "bogus_kind"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, validateReferences(wsWith(newSrc(tt.refFieldType, tt.cr), newTarget())), tt.want)
		})
	}
}

// --- validateReferences: process requirement backref (CAP-W01) -----------

func TestValidateReferences_ProcessRequirement(t *testing.T) {
	newParent := func(target string) *model.Machine {
		m := builders.Machine("mch_doc").Build()
		m.Process = &model.Process{
			States: []string{"Draft", "Submitted"},
			Transitions: []*model.ProcessTransition{{
				Name: "Submit", From: "Draft", To: "Submitted",
				Actor:        model.ProcessActor{Role: "Editor"},
				Requirements: []*model.ProcessRequirement{{Type: "evidence", Target: target, Cardinality: "1..*"}},
			}},
		}
		return m
	}

	tests := []struct {
		name     string
		machines func() []*model.Machine
		want     string
	}{
		{
			name: "valid requirement with a real backref",
			machines: func() []*model.Machine {
				evidence := builders.Machine("mch_evidence").
					WithField(builders.Field("fld_doc", model.FieldTypeReference).Options(model.FieldOptions{TargetMachine: "mch_doc"}).Build()).
					Build()
				return []*model.Machine{newParent("mch_evidence"), evidence}
			},
		},
		{
			name: "requirement target does not exist",
			machines: func() []*model.Machine {
				return []*model.Machine{newParent("mch_ghost")}
			},
			want: `machine mch_doc: transition "Submit"'s requirement target "mch_ghost" does not exist`,
		},
		{
			name: "requirement target has no reference field pointing back",
			machines: func() []*model.Machine {
				evidence := builders.Machine("mch_evidence").
					WithField(builders.Field("fld_note", model.FieldTypeText).Build()).
					Build()
				return []*model.Machine{newParent("mch_evidence"), evidence}
			},
			want: `machine mch_doc: transition "Submit"'s requirement target "mch_evidence" has no ` + "`reference`" + ` field pointing back at mch_doc`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, validateReferences(wsWith(tt.machines()...)), tt.want)
		})
	}
}

// --- validateReferences: money currency (CAP-F08/CAP-F17) ----------------

func TestValidateReferences_Money(t *testing.T) {
	tests := []struct {
		name    string
		machine func() *model.Machine
		want    string
	}{
		{
			name: "valid: fixed currency only",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_amt", model.FieldTypeMoney).Options(model.FieldOptions{Currency: "IDR"}).Build()).
					Build()
			},
		},
		{
			name: "valid: currency_field naming a real field",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_ccy", model.FieldTypeText).Build()).
					WithField(builders.Field("fld_amt", model.FieldTypeMoney).Options(model.FieldOptions{CurrencyField: "fld_ccy"}).Build()).
					Build()
			},
		},
		{
			name: "neither currency nor currency_field set",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_amt", model.FieldTypeMoney).Build()).
					Build()
			},
			want: `field fld_amt (fld_amt) on machine mch_a: type money requires exactly one of options.currency or options.currency_field`,
		},
		{
			name: "both currency and currency_field set",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_ccy", model.FieldTypeText).Build()).
					WithField(builders.Field("fld_amt", model.FieldTypeMoney).Options(model.FieldOptions{Currency: "IDR", CurrencyField: "fld_ccy"}).Build()).
					Build()
			},
			want: `field fld_amt (fld_amt) on machine mch_a: type money requires exactly one of options.currency or options.currency_field`,
		},
		{
			name: "currency_field does not name a real field",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_amt", model.FieldTypeMoney).Options(model.FieldOptions{CurrencyField: "fld_ghost"}).Build()).
					Build()
			},
			want: `field fld_amt (fld_amt) on machine mch_a: currency_field "fld_ghost" does not name a Field on this machine`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, validateReferences(wsWith(tt.machine())), tt.want)
		})
	}
}

// --- validateReferences: computed source_field (CAP-F14) -----------------

func TestValidateReferences_Computed(t *testing.T) {
	tests := []struct {
		name    string
		machine func() *model.Machine
		want    string
	}{
		{
			name: "valid: source_field is a number field",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_kg", model.FieldTypeNumber).Build()).
					WithField(builders.Field("fld_g", model.FieldTypeComputed).Options(model.FieldOptions{SourceField: "fld_kg", Factor: 1000}).Build()).
					Build()
			},
		},
		{
			name: "source_field does not exist",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_g", model.FieldTypeComputed).Options(model.FieldOptions{SourceField: "fld_ghost"}).Build()).
					Build()
			},
			want: `field fld_g (fld_g) on machine mch_a: computed source_field "fld_ghost" does not name a Field on this machine`,
		},
		{
			name: "source_field is not number/money",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_note", model.FieldTypeText).Build()).
					WithField(builders.Field("fld_g", model.FieldTypeComputed).Options(model.FieldOptions{SourceField: "fld_note"}).Build()).
					Build()
			},
			want: `field fld_g (fld_g) on machine mch_a: computed source_field "fld_note" must be type "number" or "money", got "text"`,
		},
		{
			name: "factor_field does not exist",
			machine: func() *model.Machine {
				return builders.Machine("mch_a").
					WithField(builders.Field("fld_kg", model.FieldTypeNumber).Build()).
					WithField(builders.Field("fld_g", model.FieldTypeComputed).Options(model.FieldOptions{SourceField: "fld_kg", FactorField: "fld_ghost"}).Build()).
					Build()
			},
			want: `field fld_g (fld_g) on machine mch_a: computed factor_field "fld_ghost" does not name a Field on this machine`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr(t, validateReferences(wsWith(tt.machine())), tt.want)
		})
	}
}
