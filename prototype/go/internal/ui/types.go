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
	Link          string // non-empty for a `reference` field: link to the referenced record
}

// ListRow is one row in the list view.
type ListRow struct {
	ID    string
	Cells []ListCell
}

// ReferenceOption is one selectable target record for a `reference` field's
// picker (CAP-F13) — ID is the record id stored as the field's value, Label
// is a human-readable stand-in for it.
type ReferenceOption struct {
	ID    string
	Label string
}

// FormField pairs a Field definition with its current value for form rendering.
// Options is only populated for `reference` fields (the picker's choices).
type FormField struct {
	Field   *model.Field
	Value   string
	Options []ReferenceOption
}

// DetailField is a resolved name-value pair for the detail view.
// Link is non-empty for a `reference` field: the referenced record's URL.
type DetailField struct {
	Name  string
	Value string
	Link  string
}

// ChildList is a sub-list on a parent's detail page (CAP-V06): every record
// on another Machine whose `reference` field points back to this one. Title
// names both the source Machine and which Field references it, since Menata
// Language has no way (yet) for a business author to name this relationship
// themselves (e.g. "Direct Reports") — a prototype-honest generic label, not
// a final design.
type ChildList struct {
	Title string
	Items []ChildListItem
}

// ChildListItem is one linked row in a ChildList.
type ChildListItem struct {
	Label string
	Link  string
}

// NotificationItem is one row on the Notifications page (CAP-A10). Link is
// empty when the triggering record can't be resolved (shouldn't normally
// happen, but a Notification row outlives the record it points to being
// deleted, if that ever becomes possible).
type NotificationItem struct {
	ID      string
	Message string
	Link    string
	Unread  bool
	When    string
}
