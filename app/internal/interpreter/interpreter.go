package interpreter

import (
	"fmt"
	"slices"
	"sort"

	"menata.id/app/internal/model"
)

// Interpreter builds an indexed Application Model from Runtime Metadata.
// Handlers and the Router use it for fast lookups — no DB access at request time.
type Interpreter struct {
	workspaces          []*model.Workspace
	workspacesByID      map[string]*model.Workspace
	apps                map[string]*model.Application
	machines            map[string]*model.Machine
	subscriptionsByPub  map[string][]*model.Subscription
	holidaysByWorkspace map[string]map[string]bool
}

func New(workspaces []*model.Workspace) *Interpreter {
	i := &Interpreter{
		workspaces:          workspaces,
		workspacesByID:      make(map[string]*model.Workspace, len(workspaces)),
		apps:                make(map[string]*model.Application),
		machines:            make(map[string]*model.Machine),
		subscriptionsByPub:  make(map[string][]*model.Subscription),
		holidaysByWorkspace: make(map[string]map[string]bool),
	}
	for _, ws := range workspaces {
		i.workspacesByID[ws.ID] = ws
		holidays := make(map[string]bool, len(ws.Holidays))
		for _, d := range ws.Holidays {
			holidays[d] = true
		}
		i.holidaysByWorkspace[ws.ID] = holidays
		for _, app := range ws.Applications {
			i.apps[app.ID] = app
			for _, m := range app.Machines {
				i.machines[m.ID] = m
				// CAP-I01: each Subscription is declared on its SUBSCRIBER
				// Machine, but must be found by PUBLISHER event id at
				// dispatch time -- the reverse index Pattern C's own
				// decoupling requires (the publisher never enumerates its
				// subscribers, so something has to, once, at boot).
				for _, sub := range m.Subscriptions {
					i.subscriptionsByPub[sub.PublisherEventID] = append(i.subscriptionsByPub[sub.PublisherEventID], sub)
				}
			}
		}
	}
	return i
}

// SubscriptionsFor (CAP-I01) returns every Subscription (on any Machine,
// cross-cutting by design) declaring interest in publisherEventID.
func (i *Interpreter) SubscriptionsFor(publisherEventID string) []*model.Subscription {
	return i.subscriptionsByPub[publisherEventID]
}

// Holidays (CAP-O06) returns workspaceID's own declared non-working dates
// as a set ("YYYY-MM-DD" -> true), for CAP-A11's "N Business Days" date
// arithmetic to skip alongside weekends.
func (i *Interpreter) Holidays(workspaceID string) map[string]bool {
	return i.holidaysByWorkspace[workspaceID]
}

func (i *Interpreter) GetMachine(id string) (*model.Machine, bool) {
	m, ok := i.machines[id]
	return m, ok
}

// ScopeFor resolves the Workspace/Application a Machine belongs to
// (Workspace > Application > Machine, 006-runtime-model.md — "Workspace
// isolation should always be maintained"). Needed on every security-relevant
// log line, not just the machine id: this prototype has only one workspace
// ("ws_default") today, CAP-X06 (workspace isolation/RLS) isn't enforced yet
// either, but the log schema should already carry the scope this
// architecture is constitutionally organized around (nfr-standards.md's
// "Cross-tenant reach" STRIDE threat) rather than retrofitting it once
// CAP-X06 lands and a second workspace actually exists.
func (i *Interpreter) ScopeFor(machineID string) (workspaceID, applicationID string) {
	m, ok := i.machines[machineID]
	if !ok {
		return "", ""
	}
	app, ok := i.apps[m.ApplicationID]
	if !ok {
		return "", m.ApplicationID
	}
	return app.WorkspaceID, app.ID
}

// GetApplication looks up an Application by id.
func (i *Interpreter) GetApplication(id string) (*model.Application, bool) {
	app, ok := i.apps[id]
	return app, ok
}

// GetWorkspace looks up a Workspace by id -- the session's own
// workspace_id, used to brand the UI shell with the Workspace's own Name
// (006-runtime-model.md: "Workspace is the highest organizational
// boundary") instead of a fixed "Menata Runtime" string. Every runtime
// deployment hosts one or more real organizations (ws_acme/"Acme Corp"
// alongside ws_default already proves this) -- the org a person actually
// gathers under is the Workspace they're logged into, not the runtime
// product itself.
func (i *Interpreter) GetWorkspace(id string) (*model.Workspace, bool) {
	ws, ok := i.workspacesByID[id]
	return ws, ok
}

// ApplicationsForWorkspace returns one Workspace's own Applications, sorted
// by name (CAP-O03 — Application, not Machine, is this prototype's actual
// top-level display unit; see the workspace home it backs in handler.Apps).
// CAP-X06: scoped to one workspace, not every workspace across the whole
// in-memory model — the app-layer half of workspace isolation, alongside
// RLS at the database layer (migrations/009).
func (i *Interpreter) ApplicationsForWorkspace(workspaceID string) []*model.Application {
	out := make([]*model.Application, 0, len(i.apps))
	for _, app := range i.apps {
		if app.WorkspaceID == workspaceID {
			out = append(out, app)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out
}

// MachinesForApplication returns an Application's own Machines, sorted by
// name — the per-application menu CAP-O03's Navigation concept names
// (006-runtime-model.md: Navigation is an Application-level concern,
// sibling to Machine, not a Machine-level one).
func (i *Interpreter) MachinesForApplication(applicationID string) []*model.Machine {
	app, ok := i.apps[applicationID]
	if !ok {
		return nil
	}
	out := make([]*model.Machine, len(app.Machines))
	copy(out, app.Machines)
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out
}

// AllWorkspaceIDs (CAP-E02/E03) lists every Workspace this Interpreter
// knows about -- the background scheduler sweeps each one in its own
// transaction (its own SET LOCAL app.workspace_id), the same per-workspace
// loop shape migrations/011's own backfill fix already established for
// "touch every workspace's own records under RLS."
func (i *Interpreter) AllWorkspaceIDs() []string {
	out := make([]string, 0, len(i.workspaces))
	for _, ws := range i.workspaces {
		out = append(out, ws.ID)
	}
	return out
}

func (i *Interpreter) AllMachines() []*model.Machine {
	out := make([]*model.Machine, 0, len(i.machines))
	for _, m := range i.machines {
		out = append(out, m)
	}
	return out
}

// RoleGroup is one Application's set of business roles declared across its
// machines' Permissions.
type RoleGroup struct {
	AppID   string
	AppName string
	Roles   []string
}

// AllRoles returns every distinct business role declared across one
// Workspace's machines' Permissions, grouped by Application, application/role
// name sorted for a stable order. "System" is excluded — it's the internal
// actor system-triggered events run as (CAP-A08/CAP-E05), never a role a
// real person is assigned. CAP-O01: this is the implicit per-Application role
// vocabulary — the admin page's (/admin/users) role picker for a given
// Application, sourced from whatever role strings its own machines'
// Permissions already declare, not a separately maintained list. CAP-X06:
// scoped to one workspace, same reasoning as ApplicationsForWorkspace.
func (i *Interpreter) AllRoles(workspaceID string) []RoleGroup {
	type appRoles struct {
		name  string
		roles map[string]bool
	}
	byApp := make(map[string]*appRoles)
	for _, m := range i.machines {
		app, ok := i.apps[m.ApplicationID]
		if !ok || app.WorkspaceID != workspaceID {
			continue
		}
		for _, perm := range m.Permissions {
			if perm.Role == "System" {
				continue
			}
			if byApp[app.ID] == nil {
				byApp[app.ID] = &appRoles{name: app.Name, roles: make(map[string]bool)}
			}
			byApp[app.ID].roles[perm.Role] = true
		}
	}

	appIDs := make([]string, 0, len(byApp))
	for id := range byApp {
		appIDs = append(appIDs, id)
	}
	sort.Slice(appIDs, func(a, b int) bool { return byApp[appIDs[a]].name < byApp[appIDs[b]].name })

	out := make([]RoleGroup, 0, len(appIDs))
	for _, id := range appIDs {
		ar := byApp[id]
		roles := make([]string, 0, len(ar.roles))
		for r := range ar.roles {
			roles = append(roles, r)
		}
		sort.Strings(roles)
		out = append(out, RoleGroup{AppID: id, AppName: ar.name, Roles: roles})
	}
	return out
}

func (i *Interpreter) GetEvent(machineID, eventID string) (*model.Event, bool) {
	m, ok := i.machines[machineID]
	if !ok {
		return nil, false
	}
	for _, e := range m.Events {
		if e.ID == eventID {
			return e, true
		}
	}
	return nil, false
}

// PermittedEvents returns the events this role set may trigger on the
// machine, in the order they appear in the machine definition. CAP-O07:
// roles is the acting person's full held-role set (direct + Group-derived,
// union semantics) — see Guard.CanTrigger's own doc comment.
func (i *Interpreter) PermittedEvents(machineID string, roles []string) []*model.Event {
	m, ok := i.machines[machineID]
	if !ok {
		return nil
	}
	allowed := make(map[string]bool)
	for _, perm := range m.Permissions {
		if slices.Contains(roles, perm.Role) {
			for _, eid := range perm.Events {
				allowed[eid] = true
			}
		}
	}
	var out []*model.Event
	for _, e := range m.Events {
		if allowed[e.ID] {
			out = append(out, e)
		}
	}
	return out
}

// PermittedEventsForRecord is PermittedEvents narrowed by CAP-P02 ownership:
// an event gated by a Permission with an OwnerField is only included if
// recordData's value for that field matches identityID — so a Detail page
// never offers an Approve/Reject button for a Step that isn't actually
// assigned to the viewer, even though the POST was already blocked either
// way. identityID is the acting user's account id (CAP-F05, same
// ID-not-name comparison Guard.CanTrigger makes), not their display name.
func (i *Interpreter) PermittedEventsForRecord(machineID string, roles []string, identityID string, recordData map[string]any) []*model.Event {
	m, ok := i.machines[machineID]
	if !ok {
		return nil
	}
	allowed := make(map[string]bool)
	for _, perm := range m.Permissions {
		if !slices.Contains(roles, perm.Role) {
			continue
		}
		for _, eid := range perm.Events {
			if perm.OwnerField == "" || fmt.Sprintf("%v", recordData[perm.OwnerField]) == identityID {
				allowed[eid] = true
			}
		}
	}
	var out []*model.Event
	for _, e := range m.Events {
		if allowed[e.ID] {
			out = append(out, e)
		}
	}
	return out
}

func (i *Interpreter) DefaultListView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeList)
}

func (i *Interpreter) FormView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeForm)
}

func (i *Interpreter) DetailView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeDetail)
}

// ReportView/CalendarView/TimelineView/DashboardView (CAP-V13/V07/V10) --
// same "first View of this Type" lookup as DefaultListView/FormView/
// DetailView above; a Machine declaring more than one View of the same
// auxiliary type is a case this prototype doesn't need yet (same posture as
// DefaultListView already takes for "list").
func (i *Interpreter) ReportView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeReport)
}

func (i *Interpreter) CalendarView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeCalendar)
}

func (i *Interpreter) TimelineView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeTimeline)
}

func (i *Interpreter) DashboardView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeDashboard)
}

func (i *Interpreter) DocumentView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeDocument)
}

// ProcessMapView (CAP-W05) -- same "first View of this Type" lookup; a
// Machine with no such View simply has no process map page (handler.
// ProcessMap 404s), the opt-in every other auxiliary View type already uses.
func (i *Interpreter) ProcessMapView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeProcessMap)
}

// BoardView (CAP-V14 Tier 2) -- same "first View of this Type" lookup.
func (i *Interpreter) BoardView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeBoard)
}

// CoordPlacementView (CAP-V21) -- same "first View of this Type" lookup.
func (i *Interpreter) CoordPlacementView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeCoordPlacement)
}

// DecisionStepperView (CAP-V20) -- same "first View of this Type" lookup.
func (i *Interpreter) DecisionStepperView(machineID string) *model.View {
	return i.viewOfType(machineID, model.ViewTypeDecisionStepper)
}

func (i *Interpreter) viewOfType(machineID string, t model.ViewType) *model.View {
	m, ok := i.machines[machineID]
	if !ok {
		return nil
	}
	for _, v := range m.Views {
		if v.Type == t {
			return v
		}
	}
	return nil
}
