package builders

import (
	"testing"

	"menata.id/app/internal/model"
)

func TestMachine_WithField(t *testing.T) {
	m := Machine("mch_test").
		WithField(Field("fld_name", model.FieldTypeText).Required().Build()).
		Build()

	if m.ID != "mch_test" {
		t.Errorf("ID = %q, want mch_test", m.ID)
	}
	if len(m.Fields) != 1 {
		t.Fatalf("Fields = %d, want 1", len(m.Fields))
	}
	f := m.Fields[0]
	if f.MachineID != "mch_test" {
		t.Errorf("Field.MachineID = %q, want mch_test (WithField should stamp it)", f.MachineID)
	}
	if !f.Required {
		t.Error("Field.Required = false, want true")
	}
}

func TestMachine_WithEvent(t *testing.T) {
	m := Machine("mch_test").
		WithEvent(Event("evt_submit", "Submit").Build()).
		Build()

	if len(m.Events) != 1 {
		t.Fatalf("Events = %d, want 1", len(m.Events))
	}
	if m.Events[0].MachineID != "mch_test" {
		t.Errorf("Event.MachineID = %q, want mch_test (WithEvent should stamp it)", m.Events[0].MachineID)
	}
}
