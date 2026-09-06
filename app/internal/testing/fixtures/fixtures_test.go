package fixtures

import "testing"

func TestMinimalMachine(t *testing.T) {
	m := MinimalMachine()
	if len(m.Fields) == 0 {
		t.Fatal("MinimalMachine() has no fields")
	}
	if !m.Fields[0].Required {
		t.Error("MinimalMachine()'s field should be required")
	}
}
