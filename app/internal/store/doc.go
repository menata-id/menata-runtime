// Package store holds the per-request authenticated context (Auth: User, its
// per-Application role assignments, CSRF token — resolved once by session
// middleware and attached to ctx) plus the Postgres-backed record/user/group/
// session persistence built on top of it.
//
// Graduated as-is from prototype/go/internal/store — a leaf package (0
// internal/ dependencies, Study 34 §7 Method 2). One change from the
// graduation: file-upload storage (previously a plain local-disk path
// embedded in this package's own record handling) moves to the new
// internal/storage package instead — store keeps the record/metadata about
// an uploaded file, storage owns actually reading/writing its bytes. See
// internal/storage's own doc comment and ARCHITECTURE.md's "New package"
// section for why.
package store
