package ui

import "menata.id/runtime/internal/model"

// MachineCard is the home page summary of one Machine.
type MachineCard struct {
	ID          string
	Name        string
	Description string // e.g. "7 fields · 5 events"
}

// ColumnDef is a resolved column header for the list view.
type ColumnDef struct {
	ID   string
	Name string
	Type model.FieldType
}

// ListCell is one cell in a list row, pre-formatted for rendering.
type ListCell struct {
	Value         string
	IsStatusBadge bool
}

// ListRow is one row in the list view.
type ListRow struct {
	ID    string
	Cells []ListCell
}

// FormField pairs a Field definition with its current value for form rendering.
type FormField struct {
	Field *model.Field
	Value string
}

// DetailField is a resolved name-value pair for the detail view.
type DetailField struct {
	Name  string
	Value string
}
