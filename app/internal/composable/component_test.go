package composable_test

import (
	"testing"

	"menata.id/app/internal/composable"
)

func TestResolveComponent_MissingRequiredPropertyRejected(t *testing.T) {
	if _, err := composable.ResolveComponent(composable.ComponentHeading, nil, nil, nil); err == nil {
		t.Fatal("want error for Heading with no 'text' property")
	}
}

func TestResolveComponent_UnrecognizedPropertyRejected(t *testing.T) {
	props := map[string]string{"text": "hi", "onclick": "doSomething()"}
	if _, err := composable.ResolveComponent(composable.ComponentHeading, props, nil, nil); err == nil {
		t.Fatal("want error for a property Heading's contract doesn't name (no giant property bag)")
	}
}

func TestResolveComponent_UnknownComponentTypeRejected(t *testing.T) {
	if _, err := composable.ResolveComponent(composable.ComponentType("Carousel"), nil, nil, nil); err == nil {
		t.Fatal("want error for a component type outside the closed registry")
	}
}

func TestResolveComponent_DatasetRequirementEnforced(t *testing.T) {
	if _, err := composable.ResolveComponent(composable.ComponentCollection, nil, nil, nil); err == nil {
		t.Fatal("want error for Collection with no Dataset")
	}
	ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}}
	if _, err := composable.ResolveComponent(composable.ComponentCollection, nil, &ds, nil); err != nil {
		t.Fatalf("want no error for Collection with a Dataset, got %v", err)
	}
}

func TestResolveComponent_ActionsRequireAllowance(t *testing.T) {
	if _, err := composable.ResolveComponent(composable.ComponentHeading, map[string]string{"text": "hi"}, nil, []string{"evt_x"}); err == nil {
		t.Fatal("want error for Heading (AllowsActions:false) given Actions")
	}
	node, err := composable.ResolveComponent(composable.ComponentActionBar, nil, nil, []string{"evt_x"})
	if err != nil {
		t.Fatalf("want no error for ActionBar with Actions, got %v", err)
	}
	if len(node.Actions) != 1 || node.Actions[0] != "evt_x" {
		t.Errorf("node.Actions = %v, want [evt_x]", node.Actions)
	}
}
