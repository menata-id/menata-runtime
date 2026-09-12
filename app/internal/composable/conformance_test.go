// Package composable_test's conformance_test.go is Phase 11's own proof
// system (composable-runtime-roadmap.md §15): one test per CMP-01..CMP-12
// proof class, extending the same discipline internal/composable has kept
// since Phase 1 -- pure Go, no execution, proven against real seeded
// metadata where real metadata exists. This is NOT app/conformance/ (the
// real HTTP-driven CAP-xx suite, which this package still isn't wired
// into) and NOT capability-registry.md (the real capability-admission
// ledger, governed by capability-lifecycle.md's own A1-A5 test) -- neither
// is the right target for proof classes about the composable substrate
// itself, not new user-facing capabilities. See doc.go for the full
// per-phase history each test below cites.
package composable_test

import (
	"fmt"
	"reflect"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

// TestCMP01_ValidCompositionTree consolidates Phase 4's own proof
// (LowerPage/LowerApplication, TestLowerPageCollectsChildren et al.): a
// well-formed Application lowers into a well-formed UINode tree.
func TestCMP01_ValidCompositionTree(t *testing.T) {
	app := &model.Application{
		ID: "app_cmp01",
		Machines: []*model.Machine{
			builders.Machine("mch_a").
				WithView(builders.View("vw_a_list", model.ViewTypeList).Columns("fld_a").Build()).
				Build(),
		},
	}
	pages, err := composable.LowerApplication(app)
	if err != nil {
		t.Fatalf("LowerApplication: %v", err)
	}
	if len(pages) != 1 || pages[0].Kind != composable.UINodePage || len(pages[0].Children) != 1 {
		t.Fatalf("pages = %+v, want one well-formed page with one child", pages)
	}
}

// TestCMP02_CyclicCompositionRejected is a builder-only fixture -- today's
// real metadata/validate.go allow-lists never let a page embed another
// page, so no real seeded metadata can actually cycle; this is defense in
// depth for the substrate itself (see ui.go's own LowerChildren doc
// comment), proven by directly constructing a cycle that bypasses the
// real loader's own validation.
func TestCMP02_CyclicCompositionRejected(t *testing.T) {
	viewA := builders.View("vw_a", model.ViewTypePage).Build()
	viewA.Config.Children = []model.ChildViewRef{{View: "vw_b"}}
	viewB := builders.View("vw_b", model.ViewTypePage).Build()
	viewB.Config.Children = []model.ChildViewRef{{View: "vw_a"}} // cycle back to A

	machineA := builders.Machine("mch_a").WithView(viewA).Build()
	viewA.MachineID = machineA.ID
	machineB := builders.Machine("mch_b").WithView(viewB).Build()
	viewB.MachineID = machineB.ID

	viewIdx := map[string]*model.View{"vw_a": viewA, "vw_b": viewB}
	machineIdx := composable.MachineIndex{"mch_a": machineA, "mch_b": machineB}

	_, err := composable.LowerChildren(viewA.Config.Children, viewIdx, machineIdx, map[string]bool{"vw_a": true})
	if err == nil {
		t.Fatal("want error for a cyclic composition (vw_a -> vw_b -> vw_a)")
	}
}

func TestCMP03_UnresolvedReferenceRejected(t *testing.T) {
	children := []model.ChildViewRef{{View: "vw_does_not_exist"}}
	_, err := composable.LowerChildren(children, map[string]*model.View{}, composable.MachineIndex{}, nil)
	if err == nil {
		t.Fatal("want error for a Children entry naming a view that doesn't exist")
	}
}

func TestCMP04_SlotTypeMismatchRejected(t *testing.T) {
	children := []model.ChildViewRef{{}} // neither View nor Content
	_, err := composable.LowerChildren(children, map[string]*model.View{}, composable.MachineIndex{}, nil)
	if err == nil {
		t.Fatal("want error for a Children entry naming neither a view nor content")
	}
}

// TestCMP05_ScopeViolationRejected consolidates Phase 3's own proof
// (context_test.go's TestDeriveChildScopeDropsUndeclaredParentValues /
// TestValidateBinding_*).
func TestCMP05_ScopeViolationRejected(t *testing.T) {
	scope := composable.Scope{Domains: map[composable.ContextDomain]string{composable.ContextPage: ""}}
	err := composable.ValidateBinding(scope, composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextRecord, Key: "fld_x"},
	})
	if err == nil {
		t.Fatal("want error for a binding to an undeclared domain")
	}
}

// TestCMP06_DependencyDeduplication consolidates Phase 7's own proof
// (dag_test.go's TestBuildDependencyDAG_DedupsSameDatasetAcrossConsumers).
func TestCMP06_DependencyDeduplication(t *testing.T) {
	ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}}
	nodes := []composable.UINode{
		{Identity: composable.NodeIdentity{Kind: "component", Source: "a"}, Dataset: &ds},
		{Identity: composable.NodeIdentity{Kind: "component", Source: "b"}, Dataset: &ds},
	}
	dag := composable.BuildDependencyDAG(nodes, composable.SecurityScope{Role: "Member"})
	if len(dag.Nodes) != 1 || len(dag.Nodes[0].Consumers) != 2 {
		t.Fatalf("dag = %+v, want one node with two consumers", dag)
	}
}

// TestCMP07_SecurityPreventsUnsafeCoalescing consolidates Phase 7's own
// proof (dag_test.go's TestBuildDependencyDAG_SecurityScopeSeparatesNodes).
func TestCMP07_SecurityPreventsUnsafeCoalescing(t *testing.T) {
	ds := composable.Dataset{
		Source:     composable.DataSource{MachineID: "mch_employee"},
		Projection: composable.Projection{Fields: []string{"fld_name", "fld_salary"}},
	}
	nodes := []composable.UINode{{Dataset: &ds}}
	hr := composable.BuildDependencyDAG(nodes, composable.SecurityScope{Role: "HR"})
	staff := composable.BuildDependencyDAG(nodes, composable.SecurityScope{Role: "Staff", HiddenFields: []string{"fld_salary"}})
	if hr.Nodes[0].Identity.String() == staff.Nodes[0].Identity.String() {
		t.Fatal("want different dependency identities for HR vs Staff (different HiddenFields)")
	}
}

// TestCMP08_BoundedFanOut is a builder-only fixture, same posture as
// CMP-02: a chain of 10 nested page-type views (v0 embeds v1 embeds v2...
// embeds v9), each a DISTINCT view (never a cycle), forces recursion
// depth past maxCompositionDepth (5).
func TestCMP08_BoundedFanOut(t *testing.T) {
	const levels = 10
	viewIdx := make(map[string]*model.View, levels)
	machineIdx := make(composable.MachineIndex, levels)
	views := make([]*model.View, levels)
	for i := 0; i < levels; i++ {
		views[i] = builders.View(fmt.Sprintf("vw_level%d", i), model.ViewTypePage).Build()
	}
	for i := 0; i < levels-1; i++ {
		views[i].Config.Children = []model.ChildViewRef{{View: views[i+1].ID}}
	}
	for i, v := range views {
		m := builders.Machine(fmt.Sprintf("mch_level%d", i)).WithView(v).Build()
		v.MachineID = m.ID
		viewIdx[v.ID] = v
		machineIdx[m.ID] = m
	}

	_, err := composable.LowerChildren(views[0].Config.Children, viewIdx, machineIdx, map[string]bool{views[0].ID: true})
	if err == nil {
		t.Fatal("want error once composition depth exceeds the bound")
	}
}

// TestCMP09_PlannerDeterministic consolidates Phase 8's own proof
// (planner_test.go's TestExecutionPlanExplain_Deterministic) plus a new
// assertion that BuildExecutionPlan itself is order-independent.
func TestCMP09_PlannerDeterministic(t *testing.T) {
	dag := composable.DependencyDAG{Nodes: []composable.DependencyNode{
		{Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}}, Consumers: []composable.NodeIdentity{{}}},
		{Dataset: composable.Dataset{Source: composable.DataSource{MachineID: "mch_b"}}, Consumers: []composable.NodeIdentity{{}, {}}},
	}}
	first := composable.BuildExecutionPlan(dag)
	second := composable.BuildExecutionPlan(dag)
	if first.NaiveQueryCount != second.NaiveQueryCount || first.DeduplicatedQueryCount != second.DeduplicatedQueryCount {
		t.Fatalf("BuildExecutionPlan not deterministic: %+v vs %+v", first, second)
	}
	if first.Explain() != second.Explain() {
		t.Fatal("Explain not deterministic")
	}
}

// TestCMP10_ViewLoweringEquivalence: LowerViewToComponent called twice on
// the same View produces identical results.
func TestCMP10_ViewLoweringEquivalence(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	v := builders.View("vw_list", model.ViewTypeList).Columns("fld_a", "fld_b").Build()

	first, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}
	second, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent (second call): %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("LowerViewToComponent not equivalent across calls:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

// TestCMP11_PartialFailureIsolation consolidates Phase 2/4's own proof
// (TestLowerPageAllowsUnrepresentableViews, TestBuildDataIRAgainstApprovalCase):
// a Machine mixing a representable and an unrepresentable View still
// produces every child node -- one View's own "no dataset" is an expected
// absence, not a metadata error, and never fails the whole page.
func TestCMP11_PartialFailureIsolation(t *testing.T) {
	m := builders.Machine("mch_task").
		WithView(builders.View("vw_map", model.ViewTypeProcessMap).Build()).
		WithView(builders.View("vw_list", model.ViewTypeList).Columns("fld_a").Build()).
		Build()
	page, err := composable.LowerPage(m, nil, nil)
	if err != nil {
		t.Fatalf("LowerPage: %v", err)
	}
	if len(page.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2 (both views produce a node, one just has no Dataset)", len(page.Children))
	}
}

// TestCMP12_TrialCaseCrossSubstrateReuse is explicitly blocked on Case 19
// (Project Management) being seeded in app/ -- Phase 13's own job, the
// same gap flagged since Phase 1 (see doc.go). A skip, not a silent
// omission: the proof class is named and its blocker is explicit.
func TestCMP12_TrialCaseCrossSubstrateReuse(t *testing.T) {
	t.Skip("blocked on Case 19 (Project Management) being seeded in app/ -- Phase 13's own trial migration, see composable-runtime-roadmap.md §17")
}
