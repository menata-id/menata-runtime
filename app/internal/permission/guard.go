package permission

import (
	"fmt"
	"slices"

	"menata.id/app/internal/model"
)

type Guard struct{}

// CanTrigger returns true if the given roles/identityID are allowed to
// trigger eventID on the machine. CAP-O07 (2026-08-23): roles is the
// acting person's full held-role set for this Application -- their own
// direct assignment plus every role any Group they belong to holds --
// union semantics: any one held role granting the event is enough, the
// same "assign to groups, never individual users, roles compose" model
// Slack/GCP/AWS/Azure all use (capability-registry.md's CAP-O07 row).
// CAP-P02: when a matching Permission row declares an OwnerField, the role
// alone isn't enough — identityID must also equal recordData's value for
// that field (direct allocation, not just role class). CAP-F05
// (2026-07-12): identityID is the acting user's account id
// (store.Auth.User.ID), not their display name — the loader guarantees
// OwnerField, when set, names a `user`-typed Field (see metadata/loader.go's
// validateReferences), which stores a real user id, so this is an ID-to-ID
// comparison, matching how every reference-style field is compared
// elsewhere. Never compare against a display name here: names are mutable
// and not guaranteed unique, exactly the failure mode this exists to avoid.
//
// groupMembers (CAP-F24) is a callback, not a store dependency -- this
// package stays free of any direct DB import, the same "Interpreter/Guard
// do fast in-memory lookups, no DB access at request time" posture
// interpreter.go's own package doc comment already states. It's invoked
// ONLY when a matching Permission's own DynamicActor gate resolves to
// "Group" for this specific record -- the caller (internal/handler)
// supplies it as a thin closure over the real GroupStore, resolved lazily
// so a Machine that never uses CAP-F24 costs nothing extra. See
// resolveActor below for the shared resolution logic PermittedEventsForRecord
// (interpreter.go) also uses -- kept in this package since both Guard.
// CanTrigger and the interpreter's own equivalent check need byte-identical
// semantics, and this package has no dependency on interpreter.
func (g *Guard) CanTrigger(machine *model.Machine, roles []string, identityID, eventID string, recordData map[string]any, groupMembers func(groupID string) map[string]bool) bool {
	for _, perm := range machine.Permissions {
		if !slices.Contains(roles, perm.Role) {
			continue
		}
		for _, eid := range perm.Events {
			if eid != eventID {
				continue
			}
			return ResolveActorGate(perm, identityID, recordData, groupMembers)
		}
	}
	return false
}

// ResolveActorGate (CAP-F24) is the one place BOTH Guard.CanTrigger and
// Interpreter.PermittedEventsForRecord decide whether identityID may act on
// a Permission that already matched by role+event -- kept as a single
// shared function (not duplicated in both packages) so the two can never
// silently diverge on what "this record's own gate" means. Order of
// checks, deliberate: DynamicActor first (CAP-F24, per-record), because a
// record that HAS set its own ActorTypeField is expressing a real, explicit
// choice that should win over a Machine-wide default; OwnerField second,
// as the fallback for a record that never set ActorTypeField at all (one
// created before CAP-F24 existed, or on a Permission that only ever
// declared OwnerField in the first place); no gate at all means role alone
// is sufficient, the pre-existing default.
func ResolveActorGate(perm *model.Permission, identityID string, recordData map[string]any, groupMembers func(groupID string) map[string]bool) bool {
	if da := perm.DynamicActor; da != nil {
		switch fmt.Sprintf("%v", recordData[da.ActorTypeField]) {
		case "User":
			return fmt.Sprintf("%v", recordData[da.ActorUserField]) == identityID
		case "Group":
			groupID := fmt.Sprintf("%v", recordData[da.ActorGroupField])
			if groupID == "" || groupMembers == nil {
				return false
			}
			return groupMembers(groupID)[identityID]
		}
		// ActorTypeField unset/unrecognized on this record -- fall through
		// to OwnerField below, the legacy-record compatibility path
		// model.go's own DynamicActor doc comment names explicitly.
	}
	if perm.OwnerField == "" {
		return true
	}
	return fmt.Sprintf("%v", recordData[perm.OwnerField]) == identityID
}

// CanRead/CanCreate/CanEdit (CAP-P05): CRUD-level permission, independent of
// Events. A role with no Permission row at all on a machine has none of
// these — deny-by-default. CAP-O07: roles is a set (union semantics, see
// CanTrigger's own doc comment) — granted if ANY held role's Permission row
// grants it.
func (g *Guard) CanRead(machine *model.Machine, roles []string) bool {
	for _, perm := range machine.Permissions {
		if slices.Contains(roles, perm.Role) && perm.CanRead {
			return true
		}
	}
	return false
}

func (g *Guard) CanCreate(machine *model.Machine, roles []string) bool {
	for _, perm := range machine.Permissions {
		if slices.Contains(roles, perm.Role) && perm.CanCreate {
			return true
		}
	}
	return false
}

func (g *Guard) CanEdit(machine *model.Machine, roles []string) bool {
	for _, perm := range machine.Permissions {
		if slices.Contains(roles, perm.Role) && perm.CanEdit {
			return true
		}
	}
	return false
}

// CanDelete (CAP-R03) -- same deny-by-default CRUD tier as the other three,
// but the underlying Permission column defaults to false, not true
// (migrations/012): archiving is materially more dangerous than editing
// and deserves an explicit opt-in per Permission row.
func (g *Guard) CanDelete(machine *model.Machine, roles []string) bool {
	for _, perm := range machine.Permissions {
		if slices.Contains(roles, perm.Role) && perm.CanDelete {
			return true
		}
	}
	return false
}
