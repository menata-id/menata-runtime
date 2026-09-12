package composable

import (
	"fmt"

	"menata.id/app/internal/model"
)

// MachineIndex resolves a Machine by id -- needed because
// ReportConfig.Machine / DashboardSection.Machine name a machine that may
// be DIFFERENT from whichever Machine hosts the View declaring them, unlike
// BuildDatasetFromView which only ever looks at its own host Machine.
type MachineIndex map[string]*model.Machine

// IndexMachines builds a MachineIndex over every Machine in app.
func IndexMachines(app *model.Application) MachineIndex {
	idx := make(MachineIndex, len(app.Machines))
	for _, m := range app.Machines {
		idx[m.ID] = m
	}
	return idx
}

// discoverRelations scans fields (a Dataset's own field ids) against m's
// declared Fields, recording a RelationRef for every `reference`-typed
// field found -- naming the relationship, not traversing it (real join
// execution is Phase 9 work).
func discoverRelations(m *model.Machine, fields []string) []RelationRef {
	var rels []RelationRef
	for _, fieldID := range fields {
		for _, f := range m.Fields {
			if f.ID == fieldID && f.Type == model.FieldTypeReference && f.Options.TargetMachine != "" {
				rels = append(rels, RelationRef{TargetMachineID: f.Options.TargetMachine, ViaField: f.ID})
			}
		}
	}
	return rels
}

// allFieldIDs returns every one of m's own Field ids, in declaration
// order -- split out of BuildDatasetFromView's own Detail case (Gate 3:
// keeps that switch's own complexity small).
func allFieldIDs(m *model.Machine) []string {
	ids := make([]string, len(m.Fields))
	for i, f := range m.Fields {
		ids[i] = f.ID
	}
	return ids
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// lowerFilters converts ViewConfig.Filter's own CAP-V09 grammar into the
// common Filter representation -- a direct field-for-field copy, since
// both shapes already agree; the "lowering" is the boundary the type
// system enforces (model.FilterCondition is coupled to ViewConfig, Filter
// is not), not a value transformation.
func lowerFilters(conds []model.FilterCondition) []Filter {
	if len(conds) == 0 {
		return nil
	}
	out := make([]Filter, len(conds))
	for i, c := range conds {
		out[i] = Filter{Field: c.Field, Operator: c.Operator, Value: c.Value, Expression: c.Expression}
	}
	return out
}

// lowerSort converts ViewConfig.DefaultSort into Sort -- nil sort, no
// clause (unset default sort means "no declared order", not "sort by
// nothing").
func lowerSort(s *model.SortConfig) []Sort {
	if s == nil {
		return nil
	}
	return []Sort{{Field: s.Field, Direction: s.Direction}}
}

// BuildDatasetFromView derives m/v's data requirement -- fields, filter,
// sort, and grouping (a board's GroupField, or a calendar/timeline's own
// DateField as its Date Dimension, composable-runtime-roadmap.md §10) --
// deliberately ignoring every
// rendering-specific ViewConfig field (Display, Template, ChildLines,
// SlaField, ManualOrder, ...): those are exactly the "renderer-specific
// configuration" Phase 2's exit criteria says must never leak into a
// Dataset. Returns an error for a View type with no representable data
// requirement (e.g. process_map, purely derived from Fields/Events, not a
// ViewConfig) rather than silently returning an empty Dataset -- fail-loud
// matches this codebase's own "Unknown = explicit" posture (model.go's
// SupportedOperators comment).
func BuildDatasetFromView(m *model.Machine, v *model.View) (Dataset, error) {
	var fields []string
	var groupBy []string
	switch v.Type {
	case model.ViewTypeList:
		fields = v.Config.Columns
	case model.ViewTypeCalendar, model.ViewTypeTimeline:
		fields = append([]string(nil), v.Config.Columns...)
		if v.Config.DateField != "" {
			// The Date Dimension (composable-runtime-roadmap.md §10) is a
			// real data need whether or not it's also a displayed column --
			// same reasoning the Board case below already applies to its
			// own GroupField.
			groupBy = []string{v.Config.DateField}
			if !containsString(fields, v.Config.DateField) {
				fields = append(fields, v.Config.DateField)
			}
		}
	case model.ViewTypeBoard:
		fields = append([]string(nil), v.Config.Columns...)
		if v.Config.GroupField != "" {
			groupBy = []string{v.Config.GroupField}
			// The grouping dimension is a real data need whether or not
			// it's also a displayed column -- same reasoning
			// BuildDatasetFromReport/BuildDatasetFromDashboardSection
			// already apply to their own GroupField.
			if !containsString(fields, v.Config.GroupField) {
				fields = append(fields, v.Config.GroupField)
			}
		}
	case model.ViewTypeForm:
		fields = v.Config.Fields
	case model.ViewTypeDetail:
		// 17m: a Detail view's own data requirement is every one of its
		// Machine's Fields, in declaration order -- unlike List/Board/Form,
		// Detail is never driven by a ViewConfig column/field list
		// (internal/handler/record_crud.go's real Detail handler already
		// walks m.Fields directly, not v.Config). WHICH single record is
		// a request-level concern (the URL's own recordID), not a Dataset
		// one -- the same reasoning a List's own Dataset never encodes
		// which workspace it's scoped to. No Filter/Sort/GroupBy/Measures:
		// a Detail view declares none of CAP-V09's filter/sort grammar.
		fields = allFieldIDs(m)
	default:
		return Dataset{}, fmt.Errorf("composable: view %s (machine %s, type %s) has no representable data requirement", v.ID, m.ID, v.Type)
	}

	projection := Projection{Fields: append([]string(nil), fields...)}
	filters := lowerFilters(v.Config.Filter)
	sorts := lowerSort(v.Config.DefaultSort)

	return Dataset{
		Identity:   datasetIdentity(DataSource{MachineID: m.ID}, projection, filters, sorts, groupBy, nil),
		Source:     DataSource{MachineID: m.ID},
		Relations:  discoverRelations(m, fields),
		Projection: projection,
		Filter:     filters,
		Sort:       sorts,
		GroupBy:    groupBy,
	}, nil
}

// BuildDatasetFromReport lowers a "report" View's ReportConfig -- a grouped
// aggregate over cfg.Machine (resolved via idx, which may differ from the
// View's own host Machine), one sum Measure per SumFields entry.
func BuildDatasetFromReport(idx MachineIndex, cfg *model.ReportConfig) (Dataset, error) {
	m, ok := idx[cfg.Machine]
	if !ok {
		return Dataset{}, fmt.Errorf("composable: report config names unknown machine %s", cfg.Machine)
	}

	fields := append([]string(nil), cfg.GroupField)
	fields = append(fields, cfg.SumFields...)
	projection := Projection{Fields: fields}
	groupBy := []string{cfg.GroupField}
	measures := make([]Measure, len(cfg.SumFields))
	for i, f := range cfg.SumFields {
		measures[i] = Measure{Kind: MeasureSum, Field: f}
	}

	return Dataset{
		Identity:   datasetIdentity(DataSource{MachineID: m.ID}, projection, nil, nil, groupBy, measures),
		Source:     DataSource{MachineID: m.ID},
		Relations:  discoverRelations(m, fields),
		Projection: projection,
		GroupBy:    groupBy,
		Measures:   measures,
	}, nil
}

// BuildDatasetFromDashboardSection lowers one DashboardSection -- always
// exactly one count Measure (a section is always at least a total, see
// DashboardTile's own doc comment, internal/ui/types.go), plus GroupBy only
// when GroupField is set ("Breakdown only when the section declared a
// group_field", same doc comment).
func BuildDatasetFromDashboardSection(idx MachineIndex, s model.DashboardSection) (Dataset, error) {
	m, ok := idx[s.Machine]
	if !ok {
		return Dataset{}, fmt.Errorf("composable: dashboard section names unknown machine %s", s.Machine)
	}

	var fields, groupBy []string
	if s.GroupField != "" {
		fields = []string{s.GroupField}
		groupBy = []string{s.GroupField}
	}
	projection := Projection{Fields: fields}
	measures := []Measure{{Kind: MeasureCount}}

	return Dataset{
		Identity:   datasetIdentity(DataSource{MachineID: m.ID}, projection, nil, nil, groupBy, measures),
		Source:     DataSource{MachineID: m.ID},
		Relations:  discoverRelations(m, fields),
		Projection: projection,
		GroupBy:    groupBy,
		Measures:   measures,
	}, nil
}

// BuildDatasetFromDeclaredDataset (CR-21, composable-runtime-roadmap.md
// 17k) converts a DECLARED model.Dataset -- real metadata, loaded from
// the datasets/queries tables rather than inferred from a View's own
// Config -- into the exact same composable.Dataset shape
// BuildDatasetFromView/BuildDatasetFromReport/
// BuildDatasetFromDashboardSection already produce. The concrete proof
// that a declared source and an inferred source converge on one shape,
// the same kind of proof CR-03's own two-adapters-one-Dataset precedent
// already established in Phase 2.
//
// ds's own fields have already been validated at load time
// (internal/metadata/validate.go's validateDatasets) -- BaseMachineID
// names a real Machine, every Relation's Via and every
// Dimension/Measure's Field names a real Field on it -- so this function
// assumes that and focuses purely on the shape conversion; only an
// unknown BaseMachineID (which validateDatasets would already have
// rejected during a real LoadAll) fails loud here.
func BuildDatasetFromDeclaredDataset(idx MachineIndex, ds *model.Dataset) (Dataset, error) {
	m, ok := idx[ds.BaseMachineID]
	if !ok {
		return Dataset{}, fmt.Errorf("composable: dataset %s names unknown machine %s", ds.ID, ds.BaseMachineID)
	}

	var fields, groupBy []string
	for _, dim := range ds.Config.Dimensions {
		fields = append(fields, dim.Field)
		groupBy = append(groupBy, dim.Field)
	}
	measures := make([]Measure, len(ds.Config.Measures))
	for i, mea := range ds.Config.Measures {
		if mea.Aggregate == "sum" {
			measures[i] = Measure{Kind: MeasureSum, Field: mea.Field}
			fields = append(fields, mea.Field)
			continue
		}
		measures[i] = Measure{Kind: MeasureCount} // Field empty for count, per Measure's own doc comment
	}
	var viaFields []string
	for _, rel := range ds.Config.Relations {
		viaFields = append(viaFields, rel.Via)
	}

	projection := Projection{Fields: fields}

	return Dataset{
		Identity:   datasetIdentity(DataSource{MachineID: m.ID}, projection, nil, nil, groupBy, measures),
		Source:     DataSource{MachineID: m.ID},
		Relations:  discoverRelations(m, viaFields),
		Projection: projection,
		GroupBy:    groupBy,
		Measures:   measures,
	}, nil
}

// BuildDataIR walks every Machine/View in app and collects every
// representable Dataset. A View shape with no representable dataset at all
// (detail, process_map, document, decision_stepper, coord_placement, page)
// is skipped, not errored -- same "skip, don't fail the whole page"
// precedent BuildPageNode already established in Phase 1. A report/
// dashboard naming a machine that doesn't exist in app IS a real metadata
// inconsistency, not an unrepresentable shape, so that DOES fail loud.
func BuildDataIR(app *model.Application) (DataIR, error) {
	idx := IndexMachines(app)
	var ir DataIR

	for _, m := range app.Machines {
		for _, v := range m.Views {
			switch v.Type {
			case model.ViewTypeList, model.ViewTypeCalendar, model.ViewTypeTimeline, model.ViewTypeBoard, model.ViewTypeForm, model.ViewTypeDetail:
				if ds, err := BuildDatasetFromView(m, v); err == nil {
					ir.Datasets = append(ir.Datasets, ds)
				}
			case model.ViewTypeReport:
				if v.Config.Report == nil {
					continue
				}
				ds, err := BuildDatasetFromReport(idx, v.Config.Report)
				if err != nil {
					return DataIR{}, fmt.Errorf("composable: view %s: %w", v.ID, err)
				}
				ir.Datasets = append(ir.Datasets, ds)
			case model.ViewTypeDashboard:
				for _, sec := range v.Config.Sections {
					ds, err := BuildDatasetFromDashboardSection(idx, sec)
					if err != nil {
						return DataIR{}, fmt.Errorf("composable: view %s section %q: %w", v.ID, sec.Title, err)
					}
					ir.Datasets = append(ir.Datasets, ds)
				}
			}
		}
	}
	return ir, nil
}
