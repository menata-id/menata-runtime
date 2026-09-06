// Package auth holds CAP-X02's cryptographic primitives only — password
// hashing, session bearer tokens, CSRF tokens — deliberately excluding
// session/user lookups (that's store's job) and HTTP handling (handler's).
//
// Graduated as-is from prototype/go/internal/auth — a leaf package, 0
// internal/ dependencies (Study 34 §7 Method 2). No architectural change
// intended.
package auth
