package composable_test

import (
	"testing"

	"menata.id/app/internal/composable"
)

// TestBenchmark_OneComponentOneDataset is composable-runtime-roadmap.md
// §16's own scenario 1 (baseline overhead): a single component consuming
// a single Dataset must cost exactly one of everything -- proving zero
// structural overhead beyond the necessary minimum.
func TestBenchmark_OneComponentOneDataset(t *testing.T) {
	ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}}
	nodes := []composable.UINode{{Identity: composable.NodeIdentity{Source: "n1"}, Dataset: &ds}}

	m := composable.MeasureComposition(nodes, composable.SecurityScope{Role: "Member"})
	want := composable.BenchmarkMetrics{LogicalNodes: 1, DAGNodes: 1, NaiveQueryCount: 1, DeduplicatedQueryCount: 1, ExecutionWidth: 1}
	if m != want {
		t.Errorf("metrics = %+v, want %+v", m, want)
	}
	t.Logf("scenario 1 (1 component -> 1 Dataset): %+v", m)
}

// TestBenchmark_TenIndependentDatasets is scenario 2 (bounded concurrency
// / fan-out): 10 components, 10 DISTINCT Datasets -- no reduction is
// possible, correctly. Bounding concurrency itself is not enforced
// anywhere yet (Phase 8's own named gap, stages 4-8 unbuilt) -- this only
// measures the logical fan-out shape.
func TestBenchmark_TenIndependentDatasets(t *testing.T) {
	var nodes []composable.UINode
	for i := 0; i < 10; i++ {
		ds := composable.Dataset{
			Source:     composable.DataSource{MachineID: "mch_a"},
			Projection: composable.Projection{Fields: []string{fieldName(i)}},
		}
		nodes = append(nodes, composable.UINode{Identity: composable.NodeIdentity{Source: fieldName(i)}, Dataset: &ds})
	}

	m := composable.MeasureComposition(nodes, composable.SecurityScope{Role: "Member"})
	if m.DAGNodes != 10 || m.NaiveQueryCount != 10 || m.DeduplicatedQueryCount != 10 {
		t.Errorf("metrics = %+v, want DAGNodes=NaiveQueryCount=DeduplicatedQueryCount=10 (no sharing possible)", m)
	}
	t.Logf("scenario 2 (10 components -> 10 independent Datasets): %+v", m)
}

// TestBenchmark_TenComponentsThreeSharedDatasets is scenario 3
// (deduplication/coalescing): 10 components referencing only 3 distinct
// Dataset identities -- the sharpest real demonstration of "logical
// composability does not cause proportional physical work."
func TestBenchmark_TenComponentsThreeSharedDatasets(t *testing.T) {
	shared := []composable.Dataset{
		{Source: composable.DataSource{MachineID: "mch_a"}, Projection: composable.Projection{Fields: []string{"fld_a"}}},
		{Source: composable.DataSource{MachineID: "mch_b"}, Projection: composable.Projection{Fields: []string{"fld_b"}}},
		{Source: composable.DataSource{MachineID: "mch_c"}, Projection: composable.Projection{Fields: []string{"fld_c"}}},
	}
	var nodes []composable.UINode
	for i := 0; i < 10; i++ {
		ds := shared[i%3]
		nodes = append(nodes, composable.UINode{Identity: composable.NodeIdentity{Source: fieldName(i)}, Dataset: &ds})
	}

	m := composable.MeasureComposition(nodes, composable.SecurityScope{Role: "Member"})
	if m.DAGNodes != 3 || m.NaiveQueryCount != 10 || m.DeduplicatedQueryCount != 3 {
		t.Errorf("metrics = %+v, want DAGNodes=DeduplicatedQueryCount=3, NaiveQueryCount=10", m)
	}
	t.Logf("scenario 3 (10 components -> 3 shared Datasets): %+v", m)
}

// TestBenchmark_OneDatasetMultipleProjections is scenario 4 (projection
// sharing vs narrower queries) -- an honestly REPORTED finding, not a
// pass/fail assertion of an ideal: two components on the SAME Machine
// with DIFFERENT Projection.Fields produce TWO DAG nodes today, not one.
// Projection-superset merging is Phase 8's own unbuilt stage 6 ("shared
// execution") -- this is the current, real answer to the question, not a
// failure.
func TestBenchmark_OneDatasetMultipleProjections(t *testing.T) {
	narrow := composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}, Projection: composable.Projection{Fields: []string{"fld_a"}}}
	wide := composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}, Projection: composable.Projection{Fields: []string{"fld_a", "fld_b"}}}
	nodes := []composable.UINode{
		{Identity: composable.NodeIdentity{Source: "n1"}, Dataset: &narrow},
		{Identity: composable.NodeIdentity{Source: "n2"}, Dataset: &wide},
	}

	m := composable.MeasureComposition(nodes, composable.SecurityScope{Role: "Member"})
	if m.DAGNodes != 2 {
		t.Errorf("DAGNodes = %d, want 2 (no projection-superset sharing exists yet)", m.DAGNodes)
	}
	t.Logf("scenario 4 (1 Dataset -> multiple projections): %+v -- no sharing today, Phase 8 stage 6 unbuilt", m)
}

// TestBenchmark_MixedOLTPAndAnalytics is scenario 5 (pool isolation/class
// enforcement) -- skipped: no separate pool exists (Phase 9's own audit,
// internal/db.Connect's bare pgxpool.New with no MaxConns/pool-splitting
// configuration).
func TestBenchmark_MixedOLTPAndAnalytics(t *testing.T) {
	t.Skip("not measurable: no separate OLTP/analytics pool exists (Phase 9's own audit, internal/db.Connect's bare pgxpool.New)")
}

// TestBenchmark_HundredConcurrentWorkspaces is scenario 6 (cache/workspace
// isolation) -- skipped: no cache layer exists (Phase 9's own audit), and
// a live-concurrency load harness doesn't belong in this package.
func TestBenchmark_HundredConcurrentWorkspaces(t *testing.T) {
	t.Skip("not measurable: no cache layer exists (Phase 9's own audit); a live concurrency harness is out of scope for this package")
}

// TestBenchmark_NestedCompositionDepth is scenario 8 (composition depth /
// context cost) -- builder-only, same Case-19-unseeded honesty as every
// prior phase's Project Management stand-in: a 3-level nested page chain
// using Phase 11's own recursive-Children mechanism, measuring how
// LogicalNodes/DAGNodes grow with depth.
func TestBenchmark_NestedCompositionDepth(t *testing.T) {
	level2Ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_c"}}
	level2 := composable.UINode{Identity: composable.NodeIdentity{Source: "level2"}, Dataset: &level2Ds}
	level1Ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_b"}}
	level1 := composable.UINode{
		Identity: composable.NodeIdentity{Source: "level1"},
		Dataset:  &level1Ds,
		Slots:    map[string][]composable.UINode{"": {level2}},
	}
	level0Ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_a"}}
	level0 := composable.UINode{
		Identity: composable.NodeIdentity{Source: "level0"},
		Dataset:  &level0Ds,
		Slots:    map[string][]composable.UINode{"": {level1}},
	}

	m := composable.MeasureComposition([]composable.UINode{level0}, composable.SecurityScope{Role: "Member"})
	if m.LogicalNodes != 3 || m.DAGNodes != 3 {
		t.Errorf("metrics = %+v, want LogicalNodes=DAGNodes=3 for a 3-level chain", m)
	}
	t.Logf("scenario 8 (nested Project board / composition depth, Case 19 stand-in): %+v", m)
}

func fieldName(i int) string {
	return string(rune('a' + i))
}
