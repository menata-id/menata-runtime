package composable

import (
	"fmt"
	"sort"
	"strings"
)

// ExecutionGroup is Phase 8's own stage 1 (composable-runtime-roadmap.md
// §12's "Build in this order"): every DependencyNode from a DependencyDAG
// that shares a DataSource.MachineID, grouped as candidates the planner
// could examine together for shared physical execution. Grouping is NOT
// merging -- Phase 7's own DependencyDAG already proved which nodes are
// truly equal (stage 2, dependency deduplication) and which must stay
// separate for security reasons (stage 3, security-aware equivalence); a
// group may legitimately hold several distinct nodes.
type ExecutionGroup struct {
	MachineID string
	Nodes     []DependencyNode
}

// GroupByMachine partitions dag's own Nodes into ExecutionGroups, one per
// distinct Source.MachineID, in first-seen order (deterministic).
func GroupByMachine(dag DependencyDAG) []ExecutionGroup {
	index := make(map[string]int)
	var groups []ExecutionGroup
	for _, node := range dag.Nodes {
		machineID := node.Dataset.Source.MachineID
		if i, ok := index[machineID]; ok {
			groups[i].Nodes = append(groups[i].Nodes, node)
			continue
		}
		index[machineID] = len(groups)
		groups = append(groups, ExecutionGroup{MachineID: machineID, Nodes: []DependencyNode{node}})
	}
	return groups
}

// ExecutionPlan is the "Costed Execution Plan" §12's own diagram names --
// deliberately stops BEFORE "Physical operations": no SQL, no store call,
// no concurrency/batching/cache decision (stages 4-8, all requiring a real
// physical executor this package doesn't have). NaiveQueryCount/
// DeduplicatedQueryCount is the one honest, benchmark-free cost signal
// available -- a straight count over already-proven-correct Phase 7
// output, not a fabricated weighted estimate.
type ExecutionPlan struct {
	Groups []ExecutionGroup

	// NaiveQueryCount is the sum of len(Consumers) across every
	// DependencyNode -- what a component-per-query renderer would issue,
	// one physical query per UI IR consumer, before any deduplication.
	NaiveQueryCount int

	// DeduplicatedQueryCount is len(dag.Nodes) -- one per distinct
	// dependency, Phase 7's own proven-safe (content- and security-scope-
	// aware) deduplication. This is an upper bound on physical queries,
	// not a promise real batching/sharing across DIFFERENT dependencies is
	// safe -- that claim is Phase 9/12's to earn, not this one's.
	DeduplicatedQueryCount int
}

// BuildExecutionPlan derives an ExecutionPlan from dag.
func BuildExecutionPlan(dag DependencyDAG) ExecutionPlan {
	plan := ExecutionPlan{
		Groups:                 GroupByMachine(dag),
		DeduplicatedQueryCount: len(dag.Nodes),
	}
	for _, node := range dag.Nodes {
		plan.NaiveQueryCount += len(node.Consumers)
	}
	return plan
}

// Explain renders a deterministic, human-readable summary of p -- stage 10
// ("plan diagnostics"), the "inference is inspectable" principle
// (composable-runtime-roadmap.md §3.9) applied to physical planning.
// Groups are rendered in a stable, sorted-by-MachineID order regardless of
// GroupByMachine's own first-seen order, so Explain's own output never
// depends on DependencyDAG.Nodes' iteration history.
func (p ExecutionPlan) Explain() string {
	groups := append([]ExecutionGroup(nil), p.Groups...)
	sort.Slice(groups, func(i, j int) bool { return groups[i].MachineID < groups[j].MachineID })

	var b strings.Builder
	fmt.Fprintf(&b, "ExecutionPlan: %d group(s), naive=%d dedup=%d\n", len(groups), p.NaiveQueryCount, p.DeduplicatedQueryCount)
	for _, g := range groups {
		fmt.Fprintf(&b, "  %s: %d node(s)\n", g.MachineID, len(g.Nodes))
	}
	return b.String()
}
