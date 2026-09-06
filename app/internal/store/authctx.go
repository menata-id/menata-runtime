package store

import "context"

// Auth is what CAP-X02's session middleware (cmd/server/main.go) resolves
// once per request and attaches to ctx -- the authenticated User, their
// CAP-O01 per-Application role assignments (application_id -> held roles,
// loaded once here rather than re-queried at every permission check), and
// the session's CSRF token (compared against the submitted form field by
// the CSRF middleware, and echoed into every rendered form via
// ui.CSRFField). internal/handler reads this via AuthFromContext instead of
// the old menata_role/menata_identity/menata_workspace cookies.
//
// ApplicationRoles is a set, not one string, since CAP-O07 (2026-08-23):
// a person can hold more than one role in the same Application at once --
// their own direct user_application_roles assignment, plus every role any
// Group they belong to holds there (union semantics, sessionAuth does the
// merge). Guard.CanRead/CanCreate/CanEdit/CanDelete/CanTrigger and
// Interpreter.PermittedEvents/PermittedEventsForRecord all check membership
// in this set now, not equality against a single string.
type Auth struct {
	User             *User
	ApplicationRoles map[string][]string
	CSRFToken        string
}

type authCtxKey struct{}

func WithAuth(ctx context.Context, a *Auth) context.Context {
	return context.WithValue(ctx, authCtxKey{}, a)
}

// AuthFromContext returns the request's resolved Auth. ok is false only for
// paths the session middleware deliberately skips (/login, /health,
// /static/*) -- every other route is unreachable without a valid session,
// since the middleware itself redirects/401s before the handler runs.
func AuthFromContext(ctx context.Context) (*Auth, bool) {
	a, ok := ctx.Value(authCtxKey{}).(*Auth)
	return a, ok
}
