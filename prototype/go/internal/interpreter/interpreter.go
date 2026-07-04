package interpreter

import (
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

func (i *Interpreter) AllMachines() []*model.Machine {
	out := make([]*model.Machine, 0, len(i.machines))
	for _, m := range i.machines {
		out = append(out, m)
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
