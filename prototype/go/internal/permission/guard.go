package permission

import (
	"fmt"

	"menata.id/runtime/internal/model"
)

type Guard struct{}

// CanTrigger returns true if the given role/identityID is allowed to trigger
// eventID on the machine. CAP-P02: when a matching Permission row declares an
// OwnerField, the role alone isn't enough — identityID must also equal
// recordData's value for that field (direct allocation, not just role
// class). CAP-F05 (2026-07-12): identityID is the acting user's account id
// (store.Auth.User.ID), not their display name — the loader guarantees
// OwnerField, when set, names a `user`-typed Field (see metadata/loader.go's
// validateReferences), which stores a real user id, so this is an ID-to-ID
// comparison, matching how every reference-style field is compared
// elsewhere. Never compare against a display name here: names are mutable
// and not guaranteed unique, exactly the failure mode this exists to avoid.
func (g *Guard) CanTrigger(machine *model.Machine, role, identityID, eventID string, recordData map[string]any) bool {
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
			return fmt.Sprintf("%v", recordData[perm.OwnerField]) == identityID
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

// CanDelete (CAP-R03) -- same deny-by-default CRUD tier as the other three,
// but the underlying Permission column defaults to false, not true
// (migrations/012): archiving is materially more dangerous than editing
// and deserves an explicit opt-in per Permission row.
func (g *Guard) CanDelete(machine *model.Machine, role string) bool {
	for _, perm := range machine.Permissions {
		if perm.Role == role && perm.CanDelete {
			return true
		}
	}
	return false
}
