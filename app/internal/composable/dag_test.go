package composable_test

import (
	"reflect"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

func TestResolveSecurityScope_HiddenFields(t *testing.T) {
	m := builders.Machine("mch_employee").Build()
	m.Permissions = append(m.Permissions, &model.Permission{
		Role:         "Staff",
		HiddenFields: []string{"fld_salary"},
	})

	scope := composable.ResolveSecurityScope(m, "Staff")
	if want := []string{"fld_salary"}; !reflect.DeepEqual(scope.HiddenFields, want) {
		t.Errorf("HiddenFields = %v, want %v", scope.HiddenFields, want)
	}
}

func TestResolveSecurityScope_NoPermissionRow(t *testing.T) {
	m := builders.Machine("mch_employee").Build()
	scope := composable.ResolveSecurityScope(m, "Nobody")
	if len(scope.HiddenFields) != 0 {
		t.Errorf("HiddenFields = %v, want none", scope.HiddenFields)
	}
	if scope.Role != "Nobody" {
		t.Errorf("Role = %q, want %q", scope.Role, "Nobody")
	}
}

func TestSecurityScopeApply_RemovesHiddenFields(t *testing.T) {
	ds := composable.Dataset{Projection: composable.Projection{Fields: []string{"fld_name", "fld_salary"}}}
	scope := composable.SecurityScope{Role: "Staff", HiddenFields: []string{"fld_salary"}}

	effective := scope.Apply(ds)
	if want := []string{"fld_name"}; !reflect.DeepEqual(effective.Projection.Fields, want) {
		t.Errorf("Projection.Fields = %v, want %v", effective.Projection.Fields, want)
	}
}

func TestBuildDependencyDAG_DedupsSameDatasetAcrossConsumers(t *testing.T) {
	ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}, Projection: composable.Projection{Fields: []string{"fld_a"}}}
	sibling := composable.UINode{Identity: composable.NodeIdentity{Kind: "component", Source: "a"}, Dataset: &ds}
	nested := composable.UINode{Identity: composable.NodeIdentity{Kind: "component", Source: "b"}, Dataset: &ds}
	page := composable.UINode{
		Identity: composable.NodeIdentity{Kind: "page", Source: "mch_x"},
		Children: []composable.UINode{sibling},
		Slots:    map[string][]composable.UINode{"main": {nested}},
	}

	dag := composable.BuildDependencyDAG([]composable.UINode{page}, composable.SecurityScope{Role: "Member"})
	if len(dag.Nodes) != 1 {
		t.Fatalf("len(Nodes) = %d, want 1", len(dag.Nodes))
	}
	if len(dag.Nodes[0].Consumers) != 2 {
		t.Fatalf("len(Consumers) = %d, want 2", len(dag.Nodes[0].Consumers))
	}
}

func TestBuildDependencyDAG_DifferentDatasetsStaySeparate(t *testing.T) {
	dsA := composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}, Projection: composable.Projection{Fields: []string{"fld_a"}}}
	dsB := composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}, Projection: composable.Projection{Fields: []string{"fld_b"}}}
	nodes := []composable.UINode{
		{Identity: composable.NodeIdentity{Kind: "component", Source: "a"}, Dataset: &dsA},
		{Identity: composable.NodeIdentity{Kind: "component", Source: "b"}, Dataset: &dsB},
	}

	dag := composable.BuildDependencyDAG(nodes, composable.SecurityScope{Role: "Member"})
	if len(dag.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(dag.Nodes))
	}
}

func TestBuildDependencyDAG_SecurityScopeSeparatesNodes(t *testing.T) {
	ds := composable.Dataset{
		Source:     composable.DataSource{MachineID: "mch_employee"},
		Projection: composable.Projection{Fields: []string{"fld_name", "fld_salary"}},
	}
	nodes := []composable.UINode{
		{Identity: composable.NodeIdentity{Kind: "component", Source: "vw_list"}, Dataset: &ds},
	}

	hr := composable.BuildDependencyDAG(nodes, composable.SecurityScope{Role: "HR"})
	staff := composable.BuildDependencyDAG(nodes, composable.SecurityScope{Role: "Staff", HiddenFields: []string{"fld_salary"}})

	if hr.Nodes[0].Identity.String() == staff.Nodes[0].Identity.String() {
		t.Fatal("HR and Staff dependency identities are equal, want different (security-incompatible consumers must stay separate)")
	}
	if want := []string{"fld_name", "fld_salary"}; !reflect.DeepEqual(hr.Nodes[0].Dataset.Projection.Fields, want) {
		t.Errorf("HR effective fields = %v, want %v", hr.Nodes[0].Dataset.Projection.Fields, want)
	}
	if want := []string{"fld_name"}; !reflect.DeepEqual(staff.Nodes[0].Dataset.Projection.Fields, want) {
		t.Errorf("Staff effective fields = %v, want %v (fld_salary hidden)", staff.Nodes[0].Dataset.Projection.Fields, want)
	}
}
