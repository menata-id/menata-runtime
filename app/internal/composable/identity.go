package composable

import (
	"sort"
	"strings"
)

// datasetIdentity builds a Dataset's NodeIdentity purely from its own
// content -- Source plus every Filter/Sort/GroupBy/Measure -- never from
// which View or Section asked for it. This is the mechanism that makes
// Phase 2's exit criteria concrete: two Datasets built by different
// adapters (BuildDatasetFromView, BuildDatasetFromReport,
// BuildDatasetFromDashboardSection) with the same logical requirement
// produce an equal NodeIdentity, because it's built the same way here
// regardless of caller.
func datasetIdentity(source DataSource, p Projection, filters []Filter, sorts []Sort, groupBy []string, measures []Measure) NodeIdentity {
	fields := append([]string(nil), p.Fields...)
	sort.Strings(fields)

	filterParts := make([]string, len(filters))
	for i, f := range filters {
		filterParts[i] = f.Field + ":" + f.Operator + ":" + f.Value + ":" + f.ValueField + ":" + f.Expression
	}
	sort.Strings(filterParts)

	sortParts := make([]string, len(sorts))
	for i, s := range sorts {
		sortParts[i] = s.Field + ":" + s.Direction
	}
	sort.Strings(sortParts)

	group := append([]string(nil), groupBy...)
	sort.Strings(group)

	measureParts := make([]string, len(measures))
	for i, m := range measures {
		measureParts[i] = string(m.Kind) + ":" + m.Field
	}
	sort.Strings(measureParts)

	return NodeIdentity{
		Kind:   "dataset",
		Source: source.MachineID,
		Params: map[string]string{
			"fields":   strings.Join(fields, ","),
			"filter":   strings.Join(filterParts, "|"),
			"sort":     strings.Join(sortParts, "|"),
			"group_by": strings.Join(group, ","),
			"measures": strings.Join(measureParts, "|"),
		},
	}
}

// dependencyIdentity builds a Dependency's NodeIdentity from ds's own
// content AFTER scope.Apply narrows it (the effective data need a Role
// actually sees, CAP-P06's own HiddenFields), plus scope.Role stamped in
// directly -- §11 names security scope as its own identity dimension, not
// something to be merely inferred from whichever fields happen to differ
// after hiding.
func dependencyIdentity(ds Dataset, scope SecurityScope) NodeIdentity {
	effective := scope.Apply(ds)
	id := datasetIdentity(effective.Source, effective.Projection, effective.Filter, effective.Sort, effective.GroupBy, effective.Measures)
	id.Kind = "dependency"
	id.Params["role"] = scope.Role
	return id
}

// NodeIdentity is the canonical identity of a composition node -- a Domain
// node, a Dataset, a Page, a Component, or (Phase 7) a Dependency -- named
// as data instead of only existing implicitly as "whichever Go pointer
// built it." Kind is a small closed set ("domain", "dataset", "page",
// "component", "dependency"); Source is the id of whatever this node is
// grounded in (usually a Machine or View id).
//
// Phase 7's dependencyIdentity (dag.go) is what finally adds the security
// scope dimension §11 names -- normalized filters/projection/grouping/
// measures are already Dataset's own content (Phase 2). parameters,
// pagination, and interpreter/metadata version remain unbuilt -- no case
// in this codebase demands them yet.
type NodeIdentity struct {
	Kind   string
	Source string
	Params map[string]string
}

// String renders a deterministic, human-readable form of id -- sorted by
// Params key so two NodeIdentity values built from the same logical input
// always print identically regardless of map iteration order. This is the
// "inference is inspectable" principle (composable-runtime-roadmap.md §3.9)
// applied at the smallest possible unit: every node can name itself.
func (id NodeIdentity) String() string {
	var b strings.Builder
	b.WriteString(id.Kind)
	b.WriteByte(':')
	b.WriteString(id.Source)
	if len(id.Params) == 0 {
		return b.String()
	}
	keys := make([]string, 0, len(id.Params))
	for k := range id.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteByte('/')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(id.Params[k])
	}
	return b.String()
}
