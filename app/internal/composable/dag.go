package composable

import "menata.id/app/internal/model"

// SecurityScope names the acting Role a dependency was resolved under, and
// that Role's own HiddenFields (CAP-P06) -- the security half of canonical
// dependency identity (composable-runtime-roadmap.md §11's own list). Two
// requests for the identical Dataset made under two Roles with different
// HiddenFields must NOT collapse into one physical dependency.
type SecurityScope struct {
	Role         string
	HiddenFields []string
}

// ResolveSecurityScope finds m's own Permission for role (model.Permission,
// CAP-P05/P06) and returns its HiddenFields. No Permission row for role
// means no extra hidden fields -- this only names what's HIDDEN for
// identity purposes; it does not enforce CanRead, which stays a
// Permission-check concern elsewhere, unaffected by this phase.
func ResolveSecurityScope(m *model.Machine, role string) SecurityScope {
	for _, p := range m.Permissions {
		if p.Role == role {
			return SecurityScope{Role: role, HiddenFields: append([]string(nil), p.HiddenFields...)}
		}
	}
	return SecurityScope{Role: role}
}

// Apply returns ds with every field named in scope.HiddenFields removed
// from its own Projection -- the actual effective data need a Role sees,
// not just an opaque tag on an unchanged Dataset.
func (scope SecurityScope) Apply(ds Dataset) Dataset {
	if len(scope.HiddenFields) == 0 {
		return ds
	}
	hidden := make(map[string]bool, len(scope.HiddenFields))
	for _, f := range scope.HiddenFields {
		hidden[f] = true
	}
	var fields []string
	for _, f := range ds.Projection.Fields {
		if !hidden[f] {
			fields = append(fields, f)
		}
	}
	ds.Projection = Projection{Fields: fields}
	return ds
}

// DependencyNode is one deduplicated logical data requirement -- Dataset is
// the SECURITY-SCOPED effective Dataset (post SecurityScope.Apply),
// Consumers is every UINode identity that shares this exact dependency.
type DependencyNode struct {
	Identity  NodeIdentity
	Dataset   Dataset
	Consumers []NodeIdentity
}

// DependencyDAG is the deduplicated set of DependencyNodes discovered while
// walking a UI IR tree under one SecurityScope.
type DependencyDAG struct {
	Nodes []DependencyNode
}

// BuildDependencyDAG walks every UINode in nodes (Children AND Slots,
// recursively) and collapses every node's own Dataset into a
// DependencyNode keyed by dependencyIdentity(ds, scope) -- the same
// Dataset consumed by multiple components under the SAME scope produces
// exactly one node (composable-runtime-roadmap.md §11's own dedup proof);
// a different scope (a different BuildDependencyDAG call) produces a
// different identity for the same nominal View (the security-separation
// proof). NodeIdentity itself carries a map field and so isn't usable as a
// Go map key -- identity.String() is what makes the dedup index possible.
func BuildDependencyDAG(nodes []UINode, scope SecurityScope) DependencyDAG {
	var dag DependencyDAG
	index := make(map[string]int)

	var walk func(n UINode)
	walk = func(n UINode) {
		if n.Dataset != nil {
			id := dependencyIdentity(*n.Dataset, scope)
			key := id.String()
			if i, ok := index[key]; ok {
				dag.Nodes[i].Consumers = append(dag.Nodes[i].Consumers, n.Identity)
			} else {
				index[key] = len(dag.Nodes)
				dag.Nodes = append(dag.Nodes, DependencyNode{
					Identity:  id,
					Dataset:   scope.Apply(*n.Dataset),
					Consumers: []NodeIdentity{n.Identity},
				})
			}
		}
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
	return dag
}
