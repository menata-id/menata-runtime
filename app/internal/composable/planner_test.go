package composable_test

import (
	"testing"

	"menata.id/app/internal/composable"
)

func TestGroupByMachine_GroupsSameMachineDistinctNodes(t *testing.T) {
	dag := composable.DependencyDAG{Nodes: []composable.DependencyNode{
		{Identity: composable.NodeIdentity{Kind: "dependency", Source: "mch_x", Params: map[string]string{"fields": "a"}}, Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}}},
		{Identity: composable.NodeIdentity{Kind: "dependency", Source: "mch_x", Params: map[string]string{"fields": "b"}}, Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}}},
		{Identity: composable.NodeIdentity{Kind: "dependency", Source: "mch_x", Params: map[string]string{"fields": "c"}}, Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}}},
	}}

	groups := composable.GroupByMachine(dag)
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if len(groups[0].Nodes) != 3 {
		t.Fatalf("len(groups[0].Nodes) = %d, want 3", len(groups[0].Nodes))
	}
}

func TestGroupByMachine_SeparatesDifferentMachines(t *testing.T) {
	dag := composable.DependencyDAG{Nodes: []composable.DependencyNode{
		{Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}}},
		{Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_b"}}},
	}}

	groups := composable.GroupByMachine(dag)
	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
}

func TestBuildExecutionPlan_QueryCountComparison(t *testing.T) {
	dag := composable.DependencyDAG{Nodes: []composable.DependencyNode{
		{Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}}, Consumers: []composable.NodeIdentity{{}, {}}},
		{Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}}, Consumers: []composable.NodeIdentity{{}}},
	}}

	plan := composable.BuildExecutionPlan(dag)
	if plan.NaiveQueryCount != 3 {
		t.Errorf("NaiveQueryCount = %d, want 3", plan.NaiveQueryCount)
	}
	if plan.DeduplicatedQueryCount != 2 {
		t.Errorf("DeduplicatedQueryCount = %d, want 2", plan.DeduplicatedQueryCount)
	}
}

func TestExecutionPlanExplain_Deterministic(t *testing.T) {
	dag := composable.DependencyDAG{Nodes: []composable.DependencyNode{
		{Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_b"}}, Consumers: []composable.NodeIdentity{{}}},
		{Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}}, Consumers: []composable.NodeIdentity{{}}},
	}}
	plan := composable.BuildExecutionPlan(dag)

	first := plan.Explain()
	second := plan.Explain()
	if first != second {
		t.Fatalf("Explain not deterministic:\nfirst:  %q\nsecond: %q", first, second)
	}
}
