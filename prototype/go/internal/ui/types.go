package ui

import "menata.id/runtime/internal/model"

// Card is a clickable summary tile — an Application on the workspace home
// (CAP-O03), or a Machine on an Application's own page (drilled into from
// there). Same shape either way: ID becomes the link target, Description is
// a short stat line ("3 machines" / "7 fields · 5 events").
type Card struct {
	ID          string
	Name        string
	Description string
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
	Name    string // HTML input name/id -- Field.ID normally, or CAP-F16's indexed child-row name
	Value   string
	Options []ReferenceOption
}

// ChildLinesData (CAP-F16) is a form's embedded child-Machine row editor --
// Title is the child Machine's display name, Rows is one []FormField per
// fixed row slot (config.ChildLinesConfig.MaxRows of them), each field's
// Name already indexed (e.g. "child_0_fld_x") by the handler.
type ChildLinesData struct {
	Title string
	Rows  [][]FormField
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

// RoleGroup is one Application's set of selectable business roles, for the
// admin page's (/admin/users) per-Application role picker (CAP-O01) --
// grouped by Application so a long role list stays scannable. "System" is
// deliberately never included — it's the internal actor system-triggered
// events run as (CAP-A08/CAP-E05), not a role a real person is assigned.
type RoleGroup struct {
	AppID   string
	AppName string
	Roles   []string
}

// AdminUserRow is one row on the /admin/users page (CAP-O01) -- a workspace
// user plus their current workspace role and resolved per-Application role
// (AppRoles keyed by application id, "" when the user has no assignment for
// that Application at all).
type AdminUserRow struct {
	ID            string
	Name          string
	Email         string
	WorkspaceRole string
	AppRoles      map[string]string
}

// ReportRow (CAP-V13) is one grouped row of an aggregate report View --
// Group is the report's group_field value, Sums is one formatted total per
// declared sum_field, in the same order as the View's own columns.
type ReportRow struct {
	Group string
	Sums  []string
}

// CalendarGroup (CAP-V07) buckets a Machine's records by their date_field's
// own value (e.g. every Task due "2026-07-14") -- a server-rendered grouped
// list, not a JS month-grid widget, matching this prototype's no-SPA
// posture (same call ChildLinesConfig's own doc comment makes).
type CalendarGroup struct {
	Date string
	Rows []ListRow
}

// DashboardTile (CAP-V10) is one section of a composed dashboard View --
// Total is the section's overall record count, Breakdown is populated only
// when the section declared a group_field (count per distinct value).
type DashboardTile struct {
	Title      string
	MachineID  string
	Total      int
	Breakdown  []DashboardBreakdown
}

type DashboardBreakdown struct {
	Label string
	Count int
}

// ListViewOptions bundles the list View's non-column behaviors (CAP-R03
// archive/restore, CAP-R05 pagination, CAP-V08 search, CAP-V14 manual
// order) into one param instead of a growing, easy-to-misorder positional
// bool/int list on List itself.
type ListViewOptions struct {
	SearchQuery string
	ManualOrder bool // CAP-V14
	Archived    bool // CAP-R03: true = viewing the archive itself, not the live list
	CanDelete   bool // CAP-R03: render Archive/Restore controls at all
	Page        int  // CAP-R05, 1-indexed
	TotalPages  int  // CAP-R05
}

// ImportRowResult (CAP-R06) is one CSV row's outcome -- Message is either
// the violation text (Success == false) or the newly created record's id
// (Success == true), since each row is created independently and one bad
// row doesn't block the rest of the file.
type ImportRowResult struct {
	Row     int
	Success bool
	Message string
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
