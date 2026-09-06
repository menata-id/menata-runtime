// Package router registers every HTTP route derived from the Application
// Model (not hardcoded) — the app's URL scheme: workspace home, per-
// Application pages, login, admin, and the record/event routes every
// Machine gets automatically.
//
// Graduated from prototype/go/internal/router. One deferred change, named
// not solved here: Study 34 §5 found the JSON API (CAP-X07,
// GET/POST /api/{machine}) completely unversioned, no /v1/ prefix, no
// generated OpenAPI spec. This package is where that versioning convention
// gets introduced when the development plan reaches it — deferred, not
// silently dropped. Depends on 1 internal/ package (Study 34 §7 Method 2).
package router
