// Package constraint provides an Engine that returns human-readable
// violation messages for data breaking a Machine's own declared Constraints
// — deliberately excluding "unique" and cross-record constraints, which need
// access to sibling records this package doesn't have.
//
// Graduated as-is from prototype/go/internal/constraint — depends on 1
// internal/ package (Study 34 §7 Method 2). No architectural change
// intended.
package constraint
