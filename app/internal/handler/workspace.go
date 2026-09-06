package handler

import (
	"context"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"menata.id/app/internal/interpreter"
	"menata.id/app/internal/store"
)

// reloadInterpreter (CAP-X04) re-runs metadata.Loader.LoadAll and, only if
// it succeeds, atomically swaps the active interpreter.Store -- extracted
// from Reload (admin.go) so CAP-O09's self-service Signup can make its own
// freshly-created Workspace immediately servable the same validated way an
// admin-triggered reload already does, not a second ad hoc mechanism.
func (h *Handler) reloadInterpreter(ctx context.Context) (*interpreter.Interpreter, error) {
	workspaces, err := h.loader.LoadAll(ctx)
	if err != nil {
		return nil, err
	}
	newInterp := interpreter.New(workspaces)
	h.interp.Swap(newInterp)
	return newInterp, nil
}

// reservedSlugs (CAP-O09) names every literal top-level path segment
// router.Mount already claims outside the `/{wsSlug}` subrouter -- a
// self-service Workspace can never shadow a system route.
var reservedSlugs = map[string]bool{
	"api": true, "admin": true, "static": true, "login": true,
	"signup": true, "health": true, "ui-sample": true, "webhooks": true,
	"files": true, "search": true, "notifications": true, "apps": true,
}

var slugPattern = regexp.MustCompile(`^[a-z0-9-]{3,40}$`)

// validSlug (CAP-O09) is the one format check Signup applies before ever
// touching the database -- the real uniqueness guarantee is
// WorkspaceStore.Create's own UNIQUE constraint (ErrDuplicateSlug), not a
// pre-check here.
func validSlug(slug string) bool {
	return slugPattern.MatchString(slug) && !reservedSlugs[slug]
}

// RequireWorkspaceSlug (CAP-X14) is the `/{wsSlug}` subrouter's own
// middleware (see router.Mount): resolves the URL's slug to a real
// Workspace, 404ing on an unknown one, and -- CAP-X06/CAP-X02's own
// anti-spoofing check preserved unchanged, not reopened -- 404s just the
// same if the authenticated session's own WorkspaceID doesn't match it. A
// mismatch is 404, not 403: confirming a slug exists to a session that
// doesn't belong to it would leak which workspace names are taken. A
// request with no session yet (CAP-P07 Visitor, or none) is let through
// unchanged here -- sessionAuth (a global middleware, already run before
// this one) already denied anything that isn't CAP-P07's own visitor
// carve-out, and visitorAuth resolves its own synthetic Auth against this
// same slug (see cmd/server/main.go).
func (h *Handler) RequireWorkspaceSlug(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := chi.URLParam(r, "wsSlug")
		ws, ok := h.interp.Get().WorkspaceBySlug(slug)
		if !ok {
			http.NotFound(w, r)
			return
		}
		if a, hasAuth := store.AuthFromContext(r.Context()); hasAuth && a.User.WorkspaceID != ws.ID {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
