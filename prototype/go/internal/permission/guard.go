package permission

import "menata.id/runtime/internal/model"

type Guard struct{}

// CanTrigger returns true if the given role is allowed to trigger eventID on the machine.
func (g *Guard) CanTrigger(machine *model.Machine, role, eventID string) bool {
	for _, perm := range machine.Permissions {
		if perm.Role == role {
			for _, eid := range perm.Events {
				if eid == eventID {
					return true
				}
			}
		}
	}
	return false
}
