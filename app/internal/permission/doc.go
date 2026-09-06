// Package permission provides a Guard that determines whether a given
// role/identity set is permitted to read/create/edit/trigger against a
// Machine — CAP-O07 union role semantics (direct assignment or Group
// membership) and CAP-P02 owner-field checks both live here.
//
// Graduated as-is from prototype/go/internal/permission — depends on 1
// internal/ package (Study 34 §7 Method 2). No architectural change
// intended.
package permission
