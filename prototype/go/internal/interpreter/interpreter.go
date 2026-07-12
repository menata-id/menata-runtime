package interpreter

import (
	"fmt"
	"sort"

	"menata.id/runtime/internal/model"
)

// Interpreter builds an indexed Application Model from Runtime Metadata.
// Handlers and the Router use it for fast lookups — no DB access at request time.
type Interpreter struct {
	workspaces []*model.Workspace
	apps       map[string]*model.Application
	machines   map[string]*model.Machine
}

func New(workspaces []*model.Workspace) *Interpreter {
	i := &Interpreter{
		workspaces: workspaces,
		apps:       make(map[string]*model.Application),
		machines:   make(map[string]*model.Machine),
	}
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			i.apps[app.ID] = app
			for _, m := range app.Machines {
				i.machines[m.ID] = m
			}
		}
	}
	return i
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

func (i *Interpreter) AllMachines() []*model.Machine {
	out := make([]*model.Machine, 0, len(i.machines))
	for _, m := range i.machines {
		out = append(out, m)
	}
	return out
}

// RoleGroup is one application's set of business roles declared across its
// machines' Permissions.
type RoleGroup struct {
	AppName string
	Roles   []string
}

// AllRoles returns every distinct business role declared across all
// machines' Permissions, grouped by the application each machine belongs to,
// application/role name sorted for a stable order. "System" is excluded —
// it's the internal actor system-triggered events run as (CAP-A08/CAP-E05),
// never a role a human logs in as. Used to populate the login page's role
// dropdown so it can't go stale the way a hardcoded list already had —
// Case 1's original Requester/Designer options never grew as later cases
// added Employee, Manager, HR, Submitter, Approver, Agent, Supervisor.
func (i *Interpreter) AllRoles() []RoleGroup {
	byApp := make(map[string]map[string]bool)
	for _, m := range i.machines {
		app, ok := i.apps[m.ApplicationID]
		if !ok {
			continue
		}
		for _, perm := range m.Permissions {
			if perm.Role == "System" {
				continue
			}
			if byApp[app.Name] == nil {
				byApp[app.Name] = make(map[string]bool)
			}
			byApp[app.Name][perm.Role] = true
		}
	}

	appNames := make([]string, 0, len(byApp))
	for name := range byApp {
		appNames = append(appNames, name)
	}
	sort.Strings(appNames)

	out := make([]RoleGroup, 0, len(appNames))
	for _, name := range appNames {
		roles := make([]string, 0, len(byApp[name]))
		for r := range byApp[name] {
			roles = append(roles, r)
		}
		sort.Strings(roles)
		out = append(out, RoleGroup{AppName: name, Roles: roles})
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

// PermittedEvents returns the events this role may trigger on the machine,
// in the order they appear in the machine definition.
func (i *Interpreter) PermittedEvents(machineID, role string) []*model.Event {
	m, ok := i.machines[machineID]
	if !ok {
		return nil
	}
	allowed := make(map[string]bool)
	for _, perm := range m.Permissions {
		if perm.Role == role {
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
// recordData's value for that field matches identity — so a Detail page never
// offers an Approve/Reject button for a Step that isn't actually assigned to
// the viewer, even though the POST was already blocked either way.
func (i *Interpreter) PermittedEventsForRecord(machineID, role, identity string, recordData map[string]any) []*model.Event {
	m, ok := i.machines[machineID]
	if !ok {
		return nil
	}
	allowed := make(map[string]bool)
	for _, perm := range m.Permissions {
		if perm.Role != role {
			continue
		}
		for _, eid := range perm.Events {
			if perm.OwnerField == "" || fmt.Sprintf("%v", recordData[perm.OwnerField]) == identity {
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
