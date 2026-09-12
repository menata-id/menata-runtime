package composable

import "fmt"

// ComponentType is the closed set of generic components §9 names to
// migrate first -- not a starting point for more. Adding an eighth means
// adding a case here AND a componentRegistry entry, both, on purpose (same
// discipline model.go's own EmbeddableChildViewTypes/PageEmbeddableViewTypes
// pair already uses for a closed, dual-checked vocabulary).
type ComponentType string

const (
	ComponentHeading           ComponentType = "Heading"
	ComponentText              ComponentType = "Text"
	ComponentCollection        ComponentType = "Collection"
	ComponentRecordSummaryCard ComponentType = "RecordSummaryCard"
	ComponentMetric            ComponentType = "Metric"
	ComponentStatusBadge       ComponentType = "StatusBadge"
	ComponentActionBar         ComponentType = "ActionBar"
)

// ComponentContract is the closed, compile-time shape one ComponentType is
// allowed to have. Adding a component kind means adding one contract value
// to componentRegistry -- never a runtime-registered plugin (§9's own "no
// dynamic plugin loading" rule).
type ComponentContract struct {
	RequiredProperties []string
	OptionalProperties []string
	RequiresDataset    bool
	AllowsChildren     bool
	AllowsActions      bool
}

// componentRegistry is the static registry seam itself -- a plain Go map,
// populated once, never mutated at runtime, the same "static, closed set"
// pattern this codebase already uses elsewhere (model.go's own
// SupportedOperators / EmbeddableChildViewTypes). All seven components
// deliberately set AllowsChildren:false -- none of them compose further
// sub-components yet; RecordSummaryCard is the per-ROW shape a Collection's
// renderer would eventually instantiate at render time, not an authored
// child in metadata (that composition, if it ever exists, is Phase 6+
// territory).
var componentRegistry = map[ComponentType]ComponentContract{
	ComponentHeading: {RequiredProperties: []string{"text"}},
	ComponentText:    {RequiredProperties: []string{"text"}},
	// OptionalProperties["display"] (Phase 6, §10) names the render mode --
	// "" (table, default), "cards", "board", "calendar", "timeline" -- a
	// Component-level property, never a Dataset field: Phase 2 already
	// proved Display must never enter Dataset (TestDatasetIdenticalAcross
	// DisplayModes), so the render-mode distinction has to live here.
	ComponentCollection: {
		RequiredProperties: nil,
		OptionalProperties: []string{"display"},
		RequiresDataset:    true,
	},
	ComponentRecordSummaryCard: {
		RequiredProperties: []string{"title_field"},
		OptionalProperties: []string{"subtitle_field"},
		RequiresDataset:    true,
	},
	ComponentMetric: {
		RequiresDataset: true,
	},
	ComponentStatusBadge: {
		RequiredProperties: []string{"field"},
	},
	ComponentActionBar: {
		AllowsActions: true,
	},
}

// ValidateComponent enforces node's own ComponentType contract:
//   - the type must be a key in componentRegistry (closed set, CAP-X05's
//     own "Unknown = explicit" fail-loud posture);
//   - every RequiredProperties key must be present and non-empty;
//   - every key actually set in node.Properties must be named in either
//     Required or Optional -- the concrete enforcement of "no giant
//     GenericComponent property bag";
//   - RequiresDataset/AllowsChildren/AllowsActions are checked against
//     node's own Dataset/Children/Actions fields.
func ValidateComponent(node UINode) error {
	contract, ok := componentRegistry[node.ComponentType]
	if !ok {
		return fmt.Errorf("composable: unknown component type %q", node.ComponentType)
	}

	allowed := make(map[string]bool, len(contract.RequiredProperties)+len(contract.OptionalProperties))
	for _, k := range contract.RequiredProperties {
		allowed[k] = true
	}
	for _, k := range contract.OptionalProperties {
		allowed[k] = true
	}
	for _, k := range contract.RequiredProperties {
		if node.Properties[k] == "" {
			return fmt.Errorf("composable: component %q missing required property %q", node.ComponentType, k)
		}
	}
	for k := range node.Properties {
		if !allowed[k] {
			return fmt.Errorf("composable: component %q does not accept property %q", node.ComponentType, k)
		}
	}

	if contract.RequiresDataset && node.Dataset == nil {
		return fmt.Errorf("composable: component %q requires a Dataset", node.ComponentType)
	}
	if !contract.AllowsChildren && len(node.Children) > 0 {
		return fmt.Errorf("composable: component %q does not accept children", node.ComponentType)
	}
	if !contract.AllowsActions && len(node.Actions) > 0 {
		return fmt.Errorf("composable: component %q does not accept actions", node.ComponentType)
	}
	return nil
}

// ResolveComponent is the ONLY way to produce a valid component UINode --
// contract lookup, then ValidateComponent, then the node itself. Returns
// an error instead of a partially-valid node on any contract violation.
func ResolveComponent(t ComponentType, properties map[string]string, ds *Dataset, actions []string) (UINode, error) {
	node := UINode{
		Kind:          UINodeComponent,
		ComponentType: t,
		Properties:    properties,
		Dataset:       ds,
		Actions:       actions,
	}
	if err := ValidateComponent(node); err != nil {
		return UINode{}, err
	}
	return node, nil
}
