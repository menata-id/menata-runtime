package composable

// BenchmarkMetrics is the subset of composable-runtime-roadmap.md §16's
// own metric list that is computable purely from this package's own
// logical/DAG/planner data, without any live physical execution. Every
// other §16 metric (p50/p95/p99 latency, CPU, memory, DB pool
// utilization, planner time, render time, cache hit ratio, rows scanned/
// returned) stays unbuilt -- each needs live query execution, live
// rendering, or live process metrics this package does not and should not
// fabricate (roadmap principle #8, "evidence before optimization").
type BenchmarkMetrics struct {
	LogicalNodes           int
	DAGNodes               int
	NaiveQueryCount        int
	DeduplicatedQueryCount int
	ExecutionWidth         int // the largest single ExecutionGroup -- Phase 8's own grouping
}

// countLogicalNodes walks nodes (Children AND Slots, recursively -- the
// same walk BuildDependencyDAG already uses) counting every UINode.
func countLogicalNodes(nodes []UINode) int {
	count := 0
	var walk func(n UINode)
	walk = func(n UINode) {
		count++
		for _, c := range n.Children {
			walk(c)
		}
		for _, slot := range n.Slots {
			for _, c := range slot {
				walk(c)
			}
		}
	}
	for _, n := range nodes {
		walk(n)
	}
	return count
}

// MeasureComposition is the one instrument every composable-runtime-
// roadmap.md §16 required scenario is measured with -- chains Phase 7's
// DependencyDAG and Phase 8's ExecutionPlan in one call.
func MeasureComposition(nodes []UINode, scope SecurityScope) BenchmarkMetrics {
	dag := BuildDependencyDAG(nodes, scope)
	plan := BuildExecutionPlan(dag)

	width := 0
	for _, g := range plan.Groups {
		if len(g.Nodes) > width {
			width = len(g.Nodes)
		}
	}

	return BenchmarkMetrics{
		LogicalNodes:           countLogicalNodes(nodes),
		DAGNodes:               len(dag.Nodes),
		NaiveQueryCount:        plan.NaiveQueryCount,
		DeduplicatedQueryCount: plan.DeduplicatedQueryCount,
		ExecutionWidth:         width,
	}
}
