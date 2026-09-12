package composable_test

import (
	"reflect"
	"strings"
	"testing"

	"menata.id/app/internal/composable"
)

func TestBuildWhereClause_EqualsAndNotEquals(t *testing.T) {
	filters := []composable.Filter{
		{Field: "fld_status", Operator: "equals", Value: "Open"},
		{Field: "fld_owner", Operator: "not_equals", Value: "alice"},
	}
	clause, args, err := composable.BuildWhereClause(filters, 1)
	if err != nil {
		t.Fatalf("BuildWhereClause: %v", err)
	}
	if !strings.Contains(clause, "=") || !strings.Contains(clause, "!=") {
		t.Errorf("clause = %q, want both = and !=", clause)
	}
	wantArgs := []any{"fld_status", "Open", "fld_owner", "alice"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

func TestBuildWhereClause_RejectsUnsupportedOperator(t *testing.T) {
	filters := []composable.Filter{{Field: "fld_due", Operator: "before", Value: "today"}}
	_, _, err := composable.BuildWhereClause(filters, 0)
	if err == nil {
		t.Fatal("want error for unsupported operator 'before'")
	}
	if !strings.Contains(err.Error(), "before") {
		t.Errorf("error = %q, want it to name the operator", err.Error())
	}
}

func TestBuildWhereClause_EmptyFilters(t *testing.T) {
	clause, args, err := composable.BuildWhereClause(nil, 0)
	if err != nil || clause != "" || args != nil {
		t.Errorf("BuildWhereClause(nil) = (%q, %v, %v), want (\"\", nil, nil)", clause, args, err)
	}
}

func TestBuildPhysicalPlan_AggregateAndSort(t *testing.T) {
	aggregate := composable.Dataset{Measures: []composable.Measure{{Kind: composable.MeasureCount}}}
	plan := composable.BuildPhysicalPlan(composable.DependencyNode{Dataset: aggregate})
	if !plan.AggregatePushdown {
		t.Error("AggregatePushdown = false, want true for a Dataset with Measures")
	}

	board := composable.Dataset{GroupBy: []string{"fld_status"}}
	plan = composable.BuildPhysicalPlan(composable.DependencyNode{Dataset: board})
	if plan.AggregatePushdown {
		t.Error("AggregatePushdown = true, want false for a GroupBy-only (board-shaped) Dataset with no Measures")
	}
}

func TestBuildPhysicalPlan_FilterPushdownFalseOnUnsupportedOperator(t *testing.T) {
	ds := composable.Dataset{Filter: []composable.Filter{{Field: "fld_due", Operator: "before", Value: "today"}}}
	plan := composable.BuildPhysicalPlan(composable.DependencyNode{Dataset: ds})
	if plan.FilterPushdown {
		t.Error("FilterPushdown = true, want false")
	}
	found := false
	for _, n := range plan.Notes {
		if strings.Contains(n, "before") {
			found = true
		}
	}
	if !found {
		t.Errorf("Notes = %v, want one mentioning the unsupported operator", plan.Notes)
	}
}

func TestPhysicalPlanExplain_Deterministic(t *testing.T) {
	ds := composable.Dataset{Sort: []composable.Sort{{Field: "fld_x", Direction: "asc"}}}
	plan := composable.BuildPhysicalPlan(composable.DependencyNode{Dataset: ds})
	if plan.Explain() != plan.Explain() {
		t.Fatal("Explain not deterministic")
	}
}

func TestCurrentRuntimeCapabilities_AllFalse(t *testing.T) {
	caps := composable.CurrentRuntimeCapabilities()
	if caps.WorkspaceConcurrencyLimits || caps.SeparateAnalyticsCapacity || caps.Cache || caps.Materialization {
		t.Errorf("CurrentRuntimeCapabilities() = %+v, want all false", caps)
	}
}
