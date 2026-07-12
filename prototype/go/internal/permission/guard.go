package permission

import (
	"fmt"

	"menata.id/runtime/internal/model"
)

type Guard struct{}

// CanTrigger returns true if the given role/identity is allowed to trigger
// eventID on the machine. CAP-P02: when a matching Permission row declares an
// OwnerField, the role alone isn't enough — identity must also equal
// recordData's value for that field (direct allocation, not just role class).
func (g *Guard) CanTrigger(machine *model.Machine, role, identity, eventID string, recordData map[string]any) bool {
	for _, perm := range machine.Permissions {
		if perm.Role != role {
			continue
		}
		for _, eid := range perm.Events {
			if eid != eventID {
				continue
			}
			if perm.OwnerField == "" {
				return true
			}
			return fmt.Sprintf("%v", recordData[perm.OwnerField]) == identity
		}
	}
	return false
}

// CanRead/CanCreate/CanEdit (CAP-P05): CRUD-level permission, independent of
// Events. A role with no Permission row at all on a machine has none of
// these — deny-by-default.
func (g *Guard) CanRead(machine *model.Machine, role string) bool {
	for _, perm := range machine.Permissions {
		if perm.Role == role && perm.CanRead {
			return true
		}
	}
	return false
}

func (g *Guard) CanCreate(machine *model.Machine, role string) bool {
	for _, perm := range machine.Permissions {
		if perm.Role == role && perm.CanCreate {
			return true
		}
	}
	return false
}

func (g *Guard) CanEdit(machine *model.Machine, role string) bool {
	for _, perm := range machine.Permissions {
		if perm.Role == role && perm.CanEdit {
			return true
		}
	}
	return false
}
