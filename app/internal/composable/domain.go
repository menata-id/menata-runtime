package composable

import "menata.id/app/internal/model"

// DomainNode is the composable-plane seam onto an existing *model.Machine --
// the Domain plane itself (Machine/Field/Event/Constraint/Permission) is
// already real and is not reintroduced here; this only gives Data/Experience
// nodes a stable identity to reference ("which Machine backs this dataset/
// page") without every composable type importing *model.Machine directly.
type DomainNode struct {
	Machine *model.Machine
}

// BuildDomainNode wraps m as a DomainNode. Pure and total -- every
// *model.Machine is a valid Domain node.
func BuildDomainNode(m *model.Machine) DomainNode {
	return DomainNode{Machine: m}
}

// Identity returns d's NodeIdentity, keyed on the Machine's own id.
func (d DomainNode) Identity() NodeIdentity {
	return NodeIdentity{Kind: "domain", Source: d.Machine.ID}
}
