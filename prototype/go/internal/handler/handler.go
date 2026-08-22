package handler

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"menata.id/runtime/internal/auth"
	"menata.id/runtime/internal/constraint"
	"menata.id/runtime/internal/executor"
	"menata.id/runtime/internal/interpreter"
	"menata.id/runtime/internal/metadata"
	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/permission"
	"menata.id/runtime/internal/store"
	"menata.id/runtime/internal/ui"
)

type Handler struct {
	interp        *interpreter.Store // CAP-X04: atomic, swapped by Reload -- call .Get() fresh at point of use, never cache across a request
	loader        *metadata.Loader   // CAP-X04: re-run by Reload to build a fresh Interpreter
	records       *store.RecordStore
	notifications *store.NotificationStore
	sessions      *store.SessionStore
	users         *store.UserStore
	secureCookies bool
	engine        *constraint.Engine
	guard         *permission.Guard
	exec          *executor.Executor
}

func New(interp *interpreter.Store, loader *metadata.Loader, records *store.RecordStore, notifications *store.NotificationStore, sessions *store.SessionStore, users *store.UserStore, secureCookies bool) *Handler {
	return &Handler{
		interp:        interp,
		loader:        loader,
		records:       records,
		notifications: notifications,
		sessions:      sessions,
		users:         users,
		secureCookies: secureCookies,
		engine:        &constraint.Engine{},
		guard:         &permission.Guard{},
		exec:          executor.New(records, notifications),
	}
}

// auth (CAP-X02) is every authenticated route's resolved session --
// User, CAP-O01's per-Application role map, and the CSRF token to echo back
// into rendered forms -- attached to ctx by cmd/server/main.go's sessionAuth
// middleware before any handler wired through router.Mount runs, except
// LoginForm/Login themselves (the middleware's own exempted paths, which
// never call this). A nil result here is an unreachable-in-practice
// programming error, not a case handlers guard against.
func (h *Handler) auth(r *http.Request) *store.Auth {
	a, _ := store.AuthFromContext(r.Context())
	return a
}

// identity (CAP-P02): the acting person's name, distinct from role — used
// for audit/log lines and CAP-A02's current_user (human-readable contexts).
// Never used for an ownership *comparison* — see identityID for that.
func (h *Handler) identity(r *http.Request) string {
	return h.auth(r).User.Name
}

// identityID (CAP-F05/CAP-P02): the acting person's account id, distinct
// from identity's display name — the value CAP-P02's owner_field ownership
// check (Guard.CanTrigger, Interpreter.PermittedEventsForRecord) compares
// against, since a `user`-typed Field now stores a real user id, not a
// hand-typed name. Keeping this separate from identity (Name) matters:
// identity stays human-readable for audit trails and notifications: only
// the ownership *comparison* itself needs the id.
func (h *Handler) identityID(r *http.Request) string {
	return h.auth(r).User.ID
}

// workspace (CAP-X06): which Workspace this session is authenticated into --
// the account's own workspace_id, resolved once at login (store.UserStore.
// GetByEmail), not a client-suppliable cookie.
func (h *Handler) workspace(r *http.Request) string {
	return h.auth(r).User.WorkspaceID
}

// roleForApp (CAP-O01): the acting person's role for one specific
// Application. Role is no longer a single session-wide value — the same
// person can be "Requester" in one Application and "Submitter" in another,
// simultaneously, with no manual role switch — so every call site names
// which Application it means, usually via Interpreter.ScopeFor(machineID)'s
// second return value. "" (no user_application_roles assignment for that
// Application) flows into Guard.CanRead/CanCreate/CanEdit/CanTrigger the
// same way an absent Permissions row already denies by default.
func (h *Handler) roleForApp(r *http.Request, applicationID string) string {
	return h.auth(r).ApplicationRoles[applicationID]
}

// isWorkspaceAdmin (CAP-O01): the workspace-wide tier, distinct from any
// Application role — gates /admin/users (managing other users' workspace/
// Application role assignments) and is reserved, not yet built against, for
// managing an Application's own metadata (see capability-registry.md's
// CAP-O01 note — no metadata-editing UI exists anywhere in this prototype
// to gate).
func (h *Handler) isWorkspaceAdmin(r *http.Request) bool {
	return h.auth(r).User.WorkspaceRole == "Admin"
}

// Apps — workspace home (CAP-O03): Applications, not Machines, are this
// runtime's actual top-level display unit (006-runtime-model.md's
// Workspace > Application > Machine hierarchy) — matches every real
// workspace platform's own app-launcher/module-grid pattern (Salesforce App
// Launcher, Frappe Desk), and is what Case 10 named this gap against
// ("the prototype home lists all machines flat"). Role-aware: an
// Application only appears if the current role can read at least one of its
// Machines (Guard.CanRead, CAP-P05) — derived from existing per-machine
// grants, no new metadata concept needed for this first cut. A locked-out
// role sees an empty grid, not an error.
func (h *Handler) Apps(w http.ResponseWriter, r *http.Request) {
	a := h.auth(r)
	apps := h.interp.Get().ApplicationsForWorkspace(a.User.WorkspaceID)
	cards := make([]ui.Card, 0, len(apps))
	for _, app := range apps {
		// CAP-O01: role is resolved per-Application here, not once for the
		// whole page — the same person can see a different set of readable
		// Applications depending on which role (if any) they hold in each.
		role := a.ApplicationRoles[app.ID]
		machines := h.interp.Get().MachinesForApplication(app.ID)
		readable := 0
		for _, m := range machines {
			if h.guard.CanRead(m, role) {
				readable++
			}
		}
		if readable == 0 {
			continue
		}
		cards = append(cards, ui.Card{
			ID:          app.ID,
			Name:        app.Name,
			Description: fmt.Sprintf("%d machine(s)", readable),
		})
	}
	page := ui.CardGrid("Home", a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), "Applications", "Select an application to view its machines.", "/apps/", "", cards, h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render apps", "error", err)
	}
}

// AppMachines — one Application's own Machines (CAP-O03's per-application
// Navigation, 006-runtime-model.md: Navigation is an Application-level
// concern, sibling to Machine). Same role-aware filtering as Apps, just one
// level down: a Machine only appears if the role can read it.
func (h *Handler) AppMachines(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "applicationID")
	app, ok := h.interp.Get().GetApplication(appID)
	if !ok || app.WorkspaceID != h.workspace(r) {
		// CAP-X06: an Application from another Workspace 404s exactly like
		// one that doesn't exist at all -- not a 403, which would confirm
		// to a prober that the ID is real, just in the wrong workspace.
		http.NotFound(w, r)
		return
	}
	a := h.auth(r)
	role := h.roleForApp(r, appID)
	machines := h.interp.Get().MachinesForApplication(appID)
	cards := make([]ui.Card, 0, len(machines))
	for _, m := range machines {
		if !h.guard.CanRead(m, role) {
			continue
		}
		cards = append(cards, ui.Card{
			ID:          m.ID,
			Name:        m.Name,
			Description: fmt.Sprintf("%d fields · %d events", len(m.Fields), len(m.Events)),
		})
	}
	page := ui.CardGrid(app.Name, a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), app.Name, "Select a machine to view its records.", "/", "/", cards, h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render app machines", "error", err)
	}
}

// LoginForm — email + password login page (CAP-X02).
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if err := ui.LoginPage("").Render(r.Context(), w); err != nil {
		slog.Error("render login", "error", err)
	}
}

// Login — verify email/password, then always mint a brand-new session
// (never reuse or upgrade a pre-login one — session-fixation defense) with a
// fresh CSRF token, and set the session cookie.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	user, err := h.users.GetByEmail(r.Context(), email)
	loginFailed := err != nil || !auth.VerifyPassword(user.PasswordHash, password)
	if loginFailed {
		// Deliberately the same generic outcome whether the email doesn't
		// exist or the password is wrong — doesn't confirm to a prober which
		// one failed (ASVS V2.1-style login-failure hygiene). Still a
		// security-relevant event to log (ASVS V7.1).
		slog.Warn("login failed", "correlation_id", middleware.GetReqID(r.Context()), "email", email)
		w.WriteHeader(http.StatusUnauthorized)
		if err := ui.LoginPage("Incorrect email or password.").Render(r.Context(), w); err != nil {
			slog.Error("render login (failed)", "error", err)
		}
		return
	}

	token, err := auth.NewToken()
	if err != nil {
		slog.Error("generate session token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	csrfToken, err := auth.NewToken()
	if err != nil {
		slog.Error("generate csrf token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(auth.SessionTTL)
	if err := h.sessions.Create(r.Context(), auth.HashSessionToken(token), user.ID, csrfToken, expiresAt); err != nil {
		slog.Error("create session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "menata_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
	slog.Info("login", "correlation_id", middleware.GetReqID(r.Context()), "identity", user.Name, "workspace", user.WorkspaceID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout — delete the session server-side (not just clear the cookie — a
// stolen pre-logout cookie value must stop working, not just stop being
// sent by this browser) and clear the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("menata_session"); err == nil && c.Value != "" {
		if err := h.sessions.Delete(r.Context(), auth.HashSessionToken(c.Value)); err != nil {
			slog.Error("delete session", "error", err)
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "menata_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// AdminUsers (CAP-O01) — GET /admin/users: the workspace Admin's user
// management page. Admin-only, gated on the workspace-wide tier, not any
// Application role (a workspace could have zero Applications and an Admin
// would still need this page reachable).
func (h *Handler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_view", "", "", a.User.WorkspaceRole, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	rows, err := h.adminUserRows(r.Context(), a.User.WorkspaceID)
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	appGroups := h.uiRoleGroups(a.User.WorkspaceID)
	page := ui.AdminUsers(a.User.Name, a.CSRFToken, h.unreadCount(r.Context(), a), rows, appGroups)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render admin users", "error", err)
	}
}

// AdminUpdateUser (CAP-O01) — POST /admin/users/{userID}: saves one user's
// workspace role and every Application role the page's form submitted.
// Every submitted app_role_<applicationID> value is validated against that
// Application's own implicit role vocabulary (Interpreter.AllRoles) before
// being written — deny-by-default, the same discipline CAP-P05's Permissions
// already apply, so this page can't be used to assign a role that doesn't
// actually exist in any Machine's Permissions for that Application.
func (h *Handler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_update", "", "", a.User.WorkspaceRole, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	targetID := chi.URLParam(r, "userID")
	target, err := h.users.GetByID(r.Context(), targetID)
	if err != nil || target.WorkspaceID != a.User.WorkspaceID {
		// CAP-X06: a user from another Workspace 404s exactly like one that
		// doesn't exist at all -- same convention as every other
		// cross-workspace lookup in this handler.
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	workspaceRole := r.FormValue("workspace_role")
	if workspaceRole != "Admin" && workspaceRole != "Member" {
		workspaceRole = "Member"
	}
	if err := h.users.SetWorkspaceRole(r.Context(), target.ID, workspaceRole); err != nil {
		slog.Error("set workspace role", "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}

	for _, g := range h.interp.Get().AllRoles(a.User.WorkspaceID) {
		submitted := r.FormValue("app_role_" + g.AppID)
		if submitted == "" {
			if err := h.users.RemoveApplicationRole(r.Context(), target.ID, g.AppID); err != nil {
				slog.Error("remove application role", "error", err)
				http.Error(w, "failed to save", http.StatusInternalServerError)
				return
			}
			continue
		}
		valid := false
		for _, role := range g.Roles {
			if role == submitted {
				valid = true
				break
			}
		}
		if !valid {
			continue // not a role this Application's Permissions actually declare -- ignored, not written.
		}
		if err := h.users.SetApplicationRole(r.Context(), target.ID, g.AppID, submitted); err != nil {
			slog.Error("set application role", "error", err)
			http.Error(w, "failed to save", http.StatusInternalServerError)
			return
		}
	}

	slog.Info("admin updated user roles",
		"correlation_id", middleware.GetReqID(r.Context()),
		"actor", a.User.Name,
		"target_user", target.Name,
		"workspace_role", workspaceRole,
	)
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// Reload (CAP-X04, Option A of docs/decisions/002-metadata-loading.md) is
// the only way this runtime's metadata changes without a full process
// restart: re-runs metadata.Loader.LoadAll and, only if it succeeds,
// atomically swaps the active interpreter.Store (interpreter.New's own
// validation -- validateReferences, validateOperators, compileProcess's own
// checks -- already runs inside LoadAll, unchanged from boot).
//
// The one property this handler exists to guarantee: a bad reload must
// NEVER brick the live server. At boot, a LoadAll failure calls os.Exit(1)
// (cmd/server/main.go) -- there is nothing yet to protect. Here, a failure
// is surfaced back to the admin over HTTP instead, and the OLD interpreter
// -- still valid, still serving every other request unaffected -- is never
// touched. Only a fully-built, fully-validated new Interpreter ever reaches
// Store.Swap.
func (h *Handler) Reload(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "reload", "", "", a.User.WorkspaceRole, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	workspaces, err := h.loader.LoadAll(r.Context())
	if err != nil {
		slog.Error("metadata reload failed", "correlation_id", middleware.GetReqID(r.Context()), "actor", a.User.Name, "error", err)
		http.Error(w, "reload failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	newInterp := interpreter.New(workspaces)
	h.interp.Swap(newInterp)
	slog.Info("metadata reloaded",
		"correlation_id", middleware.GetReqID(r.Context()),
		"actor", a.User.Name,
		"machines", len(newInterp.AllMachines()),
	)
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (h *Handler) uiRoleGroups(workspaceID string) []ui.RoleGroup {
	interpGroups := h.interp.Get().AllRoles(workspaceID)
	out := make([]ui.RoleGroup, 0, len(interpGroups))
	for _, g := range interpGroups {
		out = append(out, ui.RoleGroup{AppID: g.AppID, AppName: g.AppName, Roles: g.Roles})
	}
	return out
}

func (h *Handler) adminUserRows(ctx context.Context, workspaceID string) ([]ui.AdminUserRow, error) {
	users, err := h.users.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	rows := make([]ui.AdminUserRow, 0, len(users))
	for _, u := range users {
		appRoles, err := h.users.ApplicationRoles(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, ui.AdminUserRow{
			ID:            u.ID,
			Name:          u.Name,
			Email:         u.Email,
			WorkspaceRole: u.WorkspaceRole,
			AppRoles:      appRoles,
		})
	}
	return rows, nil
}

// List — list view of records for a machine.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	// CAP-P05 — deny-by-default: no permission row for this role on this
	// machine means no read access, not implicitly allowed.
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}

	view := h.interp.Get().DefaultListView(machineID)
	fieldByID := fieldIndex(machine)

	// CAP-P06: field-level visibility -- a column this role's Permission
	// hides never reaches the List page at all, not just hidden client-side.
	hidden := h.hiddenFields(machine, role)
	colIDs := []string{}
	if view != nil {
		for _, id := range view.Config.Columns {
			if !hidden[id] {
				colIDs = append(colIDs, id)
			}
		}
	}
	cols := make([]ui.ColumnDef, 0, len(colIDs))
	for _, id := range colIDs {
		def := ui.ColumnDef{ID: id, Name: id}
		if f, ok := fieldByID[id]; ok {
			def.Name = f.Name
			def.Type = f.Type
		}
		cols = append(cols, def)
	}

	// CAP-R03: ?archived=1 shows the archive itself (ListArchived) instead
	// of the live list -- the one place a soft-deleted record can still be
	// found and restored. Sort/filter/search/pagination below all apply the
	// same way to either set.
	archived := r.URL.Query().Get("archived") == "1"
	var records []*store.Record
	var err error
	if archived {
		records, err = h.records.ListArchived(r.Context(), machineID)
	} else {
		sortField, sortDir := "", ""
		if view != nil && view.Config.ManualOrder {
			// CAP-V14 wins over DefaultSort when both are declared on the same
			// View -- manual order only means anything if it's what's actually
			// shown.
			sortField = store.SortOrderField
		} else if view != nil && view.Config.DefaultSort != nil {
			sortField, sortDir = view.Config.DefaultSort.Field, view.Config.DefaultSort.Direction
		}
		records, err = h.records.List(r.Context(), machineID, sortField, sortDir)
	}
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	// CAP-V05/V09: a list View's declarative filter, AND-combined, reusing
	// constraint.Eval's own expression grammar. $current_user (CAP-V05) is
	// resolved to the acting identity's id here, request-time, before Eval
	// ever sees it -- Eval itself has no notion of "who's asking."
	if view != nil && len(view.Config.Filter) > 0 {
		identityID := h.identityID(r)
		kept := records[:0]
		for _, rec := range records {
			match := true
			for _, fc := range view.Config.Filter {
				val := fc.Value
				if val == "$current_user" {
					val = identityID
				}
				if !constraint.Eval(model.ConstraintExpression{Field: fc.Field, Operator: fc.Operator, Value: val}, rec.Data) {
					match = false
					break
				}
			}
			if match {
				kept = append(kept, rec)
			}
		}
		records = kept
	}

	// CAP-V08: free-text search across this View's visible columns, ?q=.
	// Substring, case-insensitive, HTTP black-box (a plain GET query param,
	// no JS) -- matches this prototype's no-SPA-framework posture.
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	if searchQuery != "" {
		q := strings.ToLower(searchQuery)
		kept := records[:0]
		for _, rec := range records {
			match := false
			for _, id := range colIDs {
				if v, ok := rec.Data[id]; ok && strings.Contains(strings.ToLower(fmt.Sprintf("%v", v)), q) {
					match = true
					break
				}
			}
			if match {
				kept = append(kept, rec)
			}
		}
		records = kept
	}

	// CAP-R05: pagination applies AFTER filter/search, on the final matching
	// set, not as a SQL LIMIT/OFFSET before them -- otherwise a filter could
	// discard most of one SQL page and never see matching rows sitting on
	// the next one. In-memory slicing costs nothing extra at this
	// prototype's scale, the same tradeoff CAP-V08/V09's own in-memory
	// filtering already made.
	const pageSize = 25
	pageNum, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if pageNum < 1 {
		pageNum = 1
	}
	totalRecords := len(records)
	totalPages := (totalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if pageNum > totalPages {
		pageNum = totalPages
	}
	start := (pageNum - 1) * pageSize
	end := start + pageSize
	if start > totalRecords {
		start = totalRecords
	}
	if end > totalRecords {
		end = totalRecords
	}
	records = records[start:end]

	rows := make([]ui.ListRow, 0, len(records))
	for _, rec := range records {
		cells := make([]ui.ListCell, len(colIDs))
		for j, id := range colIDs {
			val := ""
			if v, ok := rec.Data[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
			link := ""
			switch {
			case cols[j].Type == model.FieldTypeReference && val != "":
				refID := val
				target := fieldByID[id].Options.TargetMachine
				if label, err := h.referenceLabel(r.Context(), target, refID); err == nil && label != "" {
					val = label
					link = "/" + target + "/" + refID
				}
			case cols[j].Type == model.FieldTypeUser && val != "":
				if label, err := h.userLabel(r.Context(), val); err == nil && label != "" {
					val = label
				}
			case cols[j].Type == model.FieldTypeBoolean:
				val = boolLabel(val) // CAP-F09
			case cols[j].Type == model.FieldTypeMoney && val != "":
				val = formatMoney(val, fieldByID[id], rec.Data) // CAP-F08
			case cols[j].Type == model.FieldTypeFile && val != "":
				link = "/files/" + val // CAP-F06
			case cols[j].Type == model.FieldTypeComputed:
				val = computedValue(fieldByID[id], rec.Data) // CAP-F14
			}
			cells[j] = ui.ListCell{
				Value:         val,
				IsStatusBadge: cols[j].Type == model.FieldTypeValueList,
				Link:          link,
			}
		}
		rows = append(rows, ui.ListRow{ID: rec.ID, Cells: cells})
	}

	opts := ui.ListViewOptions{
		SearchQuery: searchQuery,
		ManualOrder: view != nil && view.Config.ManualOrder,
		Archived:    archived,
		CanDelete:   h.guard.CanDelete(machine, role),
		Page:        pageNum,
		TotalPages:  totalPages,
	}
	a := h.auth(r)
	page := ui.List(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, cols, rows, h.interp.Get().PermittedEvents(machineID, role), h.unreadCount(r.Context(), a), opts, h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render list", "error", err)
	}
}

// Archive/Restore (CAP-R03) soft-delete/undelete a record. CanDelete-gated
// -- a distinct, opt-in permission tier from CanEdit (migrations/012's own
// note on why can_delete defaults false).
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	h.setDeleted(w, r, true)
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	h.setDeleted(w, r, false)
}

func (h *Handler) setDeleted(w http.ResponseWriter, r *http.Request, deleted bool) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanDelete(machine, role) {
		h.logPermissionDenied(r.Context(), "delete", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	// CAP-R07: an immutable record can't be archived either (not restored
	// -- undeleting isn't a business-data mutation), same gate Update uses
	// -- "frozen" means frozen against every mutation path, not just field
	// edits.
	if deleted {
		if rec, err := h.records.Get(r.Context(), recordID); err == nil && rec.MachineID == machineID {
			if reason := h.immutabilityViolation(machine, rec.Data); reason != "" {
				http.Error(w, reason, http.StatusForbidden)
				return
			}
			// CAP-O02: a master-data record (Machine.Config["master_data"])
			// can't be archived while any OTHER record, on ANY Machine
			// (cross-app by design), still references it -- archiving it
			// would silently break every one of those references. Reuses
			// CAP-V06's own childLists scan (every Machine's `reference`
			// fields targeting this one) rather than a second
			// implementation of "who points at me."
			if machine.Config["master_data"] == "true" {
				if refs := h.childLists(r.Context(), machine, recordID); len(refs) > 0 {
					http.Error(w, fmt.Sprintf("cannot archive: still referenced by %s", refs[0].Title), http.StatusConflict)
					return
				}
			}
		}
	}
	var err error
	if deleted {
		err = h.records.Archive(r.Context(), recordID)
	} else {
		err = h.records.Restore(r.Context(), recordID)
	}
	if err != nil {
		http.Error(w, "failed to update record", http.StatusInternalServerError)
		return
	}
	if deleted {
		http.Redirect(w, r, "/"+machineID, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/"+machineID+"/"+recordID, http.StatusSeeOther)
	}
}

// MoveRecord (CAP-V14) reorders recordID up or down among its siblings on
// machineID's manual-order list View. CanEdit-gated -- reordering changes
// something about the record's presentation, the same permission tier as
// changing one of its fields, not a separate concept.
func (h *Handler) MoveRecord(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	direction := chi.URLParam(r, "direction")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanEdit(machine, role) {
		h.logPermissionDenied(r.Context(), "edit", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	if err := h.records.Move(r.Context(), machineID, recordID, direction); err != nil {
		http.Error(w, "failed to move record", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID, http.StatusSeeOther)
}

// Report renders a CAP-V13 aggregate report View -- grouped SUMs computed
// at render time from ANOTHER Machine's own records (view.Config.Report),
// not this Machine's. Read access is checked against the SOURCE machine
// (the data being aggregated), same reasoning as CAP-V06's reverse-lookup
// child lists reading the child Machine's own records.
// Document handles GET /{machineID}/{recordID}/document (CAP-F21) -- renders
// the Machine's own `document`-type View (Config.Template, an html/template
// source with {{.fld_x}} placeholders) against one record's Data. html/
// template auto-escapes every interpolated value, so a record whose data
// happens to contain HTML/script-looking text can't inject anything into
// the rendered page. Output is HTML, not a binary PDF/image -- see
// model.ViewTypeDocument's own doc comment for why that's a deliberate,
// named scope cut, not an oversight.
func (h *Handler) Document(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	view := h.interp.Get().DocumentView(machineID)
	if view == nil || view.Config.Template == "" {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}
	tmpl, err := htmltemplate.New(view.ID).Parse(view.Config.Template)
	if err != nil {
		slog.Error("parse document template", "view", view.ID, "error", err)
		http.Error(w, "failed to render document", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, rec.Data); err != nil {
		slog.Error("render document", "view", view.ID, "error", err)
	}
}

func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	view := h.interp.Get().ReportView(machineID)
	if view == nil || view.Config.Report == nil {
		http.NotFound(w, r)
		return
	}
	rc := view.Config.Report

	srcFieldByID := map[string]*model.Field{}
	if src, ok := h.interp.Get().GetMachine(rc.Machine); ok {
		srcFieldByID = fieldIndex(src)
	}
	sumLabels := make([]string, len(rc.SumFields))
	for i, f := range rc.SumFields {
		sumLabels[i] = f
		if sf, ok := srcFieldByID[f]; ok {
			sumLabels[i] = sf.Name
		}
	}

	groups, err := h.records.SumFieldsGroupedBy(r.Context(), rc.Machine, rc.GroupField, rc.SumFields)
	if err != nil {
		http.Error(w, "failed to load report", http.StatusInternalServerError)
		return
	}
	rows := make([]ui.ReportRow, len(groups))
	for i, g := range groups {
		sums := make([]string, len(rc.SumFields))
		for j, f := range rc.SumFields {
			sums[j] = fmt.Sprintf("%.2f", g.Sums[f])
		}
		rows[i] = ui.ReportRow{Group: g.Group, Sums: sums}
	}

	a := h.auth(r)
	page := ui.Report(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, sumLabels, rows, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render report", "error", err)
	}
}

// calendarTimeline backs both Calendar and Timeline (CAP-V07) -- same
// grouped-by-date_field rendering, only the View lookup differs.
func (h *Handler) calendarTimeline(w http.ResponseWriter, r *http.Request, view *model.View) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	if view == nil {
		http.NotFound(w, r)
		return
	}
	fieldByID := fieldIndex(machine)
	colIDs := view.Config.Columns
	cols := make([]ui.ColumnDef, 0, len(colIDs))
	for _, id := range colIDs {
		def := ui.ColumnDef{ID: id, Name: id}
		if f, ok := fieldByID[id]; ok {
			def.Name = f.Name
			def.Type = f.Type
		}
		cols = append(cols, def)
	}

	records, err := h.records.List(r.Context(), machineID, view.Config.DateField, "asc")
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	var groups []ui.CalendarGroup
	var cur string
	var curRows []ui.ListRow
	flush := func() {
		if curRows != nil {
			groups = append(groups, ui.CalendarGroup{Date: cur, Rows: curRows})
		}
	}
	first := true
	for _, rec := range records {
		date := fmt.Sprintf("%v", rec.Data[view.Config.DateField])
		if date == "<nil>" {
			date = ""
		}
		if first || date != cur {
			flush()
			cur = date
			curRows = nil
			first = false
		}
		cells := make([]ui.ListCell, len(colIDs))
		for j, id := range colIDs {
			val := ""
			if v, ok := rec.Data[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
			cells[j] = ui.ListCell{Value: val, IsStatusBadge: cols[j].Type == model.FieldTypeValueList}
		}
		curRows = append(curRows, ui.ListRow{ID: rec.ID, Cells: cells})
	}
	flush()

	a := h.auth(r)
	page := ui.CalendarTimeline(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, cols, groups, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render calendar/timeline", "error", err)
	}
}

// Calendar renders a CAP-V07 calendar View: records grouped by date_field.
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	h.calendarTimeline(w, r, h.interp.Get().CalendarView(chi.URLParam(r, "machineID")))
}

// Timeline renders a CAP-V07 timeline View: the same grouping, read
// chronologically.
func (h *Handler) Timeline(w http.ResponseWriter, r *http.Request) {
	h.calendarTimeline(w, r, h.interp.Get().TimelineView(chi.URLParam(r, "machineID")))
}

// Dashboard renders a CAP-V10 composed dashboard View -- one tile per
// declared Section, each possibly a DIFFERENT Machine than the one the
// dashboard View itself is declared on (the actual point of "composed").
// Each section's own read permission is checked independently -- a role
// that can't read a given section's Machine just doesn't get that tile,
// rather than the whole dashboard 403ing.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	view := h.interp.Get().DashboardView(machineID)
	if view == nil {
		http.NotFound(w, r)
		return
	}

	tiles := make([]ui.DashboardTile, 0, len(view.Config.Sections))
	for _, sec := range view.Config.Sections {
		secMachine, ok := h.interp.Get().GetMachine(sec.Machine)
		if !ok {
			continue
		}
		_, secAppID := h.interp.Get().ScopeFor(sec.Machine)
		secRole := h.roleForApp(r, secAppID)
		if !h.guard.CanRead(secMachine, secRole) {
			continue
		}
		counts, err := h.records.CountGroupedBy(r.Context(), sec.Machine, sec.GroupField)
		if err != nil {
			http.Error(w, "failed to load dashboard", http.StatusInternalServerError)
			return
		}
		tile := ui.DashboardTile{Title: sec.Title, MachineID: sec.Machine}
		for _, c := range counts {
			tile.Total += c.Count
			if sec.GroupField != "" {
				tile.Breakdown = append(tile.Breakdown, ui.DashboardBreakdown{Label: c.Group, Count: c.Count})
			}
		}
		tiles = append(tiles, tile)
	}

	a := h.auth(r)
	page := ui.Dashboard(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), view.Name, tiles, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render dashboard", "error", err)
	}
}

// csvFieldIDs (CAP-R06) returns the field ids export/import agree on --
// the FormView's own Fields, the same set Create/Update read from, kept
// symmetric so an exported CSV re-imports cleanly. Header row uses field
// ids, not display Names -- unambiguous and machine-inspectable, the same
// "ids are the interchange format" choice CAP-F05 already made.
func (h *Handler) csvFieldIDs(machine *model.Machine) []string {
	if fv := h.interp.Get().FormView(machine.ID); fv != nil {
		return fv.Config.Fields
	}
	return nil
}

// ExportCSV (CAP-R06) streams every live (non-archived) record on
// machineID as CSV -- CanRead-gated, same tier as viewing the list itself.
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	records, err := h.records.List(r.Context(), machineID, "", "")
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}
	fieldIDs := h.csvFieldIDs(machine)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+machineID+`.csv"`)
	cw := csv.NewWriter(w)
	if err := cw.Write(fieldIDs); err != nil {
		return
	}
	for _, rec := range records {
		row := make([]string, len(fieldIDs))
		for i, id := range fieldIDs {
			if v, ok := rec.Data[id]; ok {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := cw.Write(row); err != nil {
			return
		}
	}
	cw.Flush()
}

// ImportCSVForm (CAP-R06) renders the CSV upload page.
func (h *Handler) ImportCSVForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanCreate(machine, role) {
		h.logPermissionDenied(r.Context(), "create", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	page := ui.ImportCSV(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, h.csvFieldIDs(machine), nil, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render import form", "error", err)
	}
}

// ImportCSV (CAP-R06) bulk-creates records from an uploaded CSV, one HTTP
// request. Each row is INDEPENDENT -- unlike CAP-F16's atomic parent+child
// create, one bad row is reported and skipped, not a reason to reject every
// other row in the file. Every row goes through the exact same validation
// pipeline Create uses (Violations, referenceViolations,
// userReferenceViolations, uniquenessViolations, including CAP-R08's
// scratch-state exemption) -- CSV import is not a side door around
// ordinary Create rules. Rows are created one at a time, in file order, so
// a uniqueness check on row N correctly sees rows already imported earlier
// in the same file, not just what existed in the database beforehand.
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanCreate(machine, role) {
		h.logPermissionDenied(r.Context(), "create", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	cr := csv.NewReader(file)
	header, err := cr.Read()
	if err != nil {
		http.Error(w, "empty or malformed CSV", http.StatusBadRequest)
		return
	}
	fieldByID := fieldIndex(machine)
	headerFields := map[string]bool{}
	for _, id := range header {
		headerFields[id] = true
	}

	var results []ui.ImportRowResult
	rowNum := 1
	for {
		record, err := cr.Read()
		if err != nil {
			break
		}
		rowNum++

		data := make(map[string]any)
		for _, f := range machine.Fields {
			if !headerFields[f.ID] && f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
				data[f.ID] = f.Options.Values[0]
			}
		}
		for i, id := range header {
			if i < len(record) && record[i] != "" {
				if _, ok := fieldByID[id]; ok {
					data[id] = record[i]
				}
			}
		}

		var violations []string
		if !h.inScratchState(machine, data) {
			violations = h.engine.Violations(machine, withChangePolicyCreatedAt(machine, data, time.Now()))
		}
		if refV, err := h.referenceViolations(r.Context(), machine, data); err == nil {
			violations = append(violations, refV...)
		}
		if userV, err := h.userReferenceViolations(r.Context(), machine, data); err == nil {
			violations = append(violations, userV...)
		}
		if uniqueV, err := h.uniquenessViolations(r.Context(), machine, data, ""); err == nil {
			violations = append(violations, uniqueV...)
		}

		if len(violations) > 0 {
			results = append(results, ui.ImportRowResult{Row: rowNum, Success: false, Message: strings.Join(violations, "; ")})
			continue
		}
		rec, err := h.records.Create(r.Context(), machineID, workspaceID, data)
		if err != nil {
			results = append(results, ui.ImportRowResult{Row: rowNum, Success: false, Message: "failed to create record"})
			continue
		}
		results = append(results, ui.ImportRowResult{Row: rowNum, Success: true, Message: rec.ID})
	}

	a := h.auth(r)
	page := ui.ImportCSV(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, h.csvFieldIDs(machine), results, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render import results", "error", err)
	}
}

// NewForm — form for creating a new record.
func (h *Handler) NewForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanCreate(machine, role) {
		h.logPermissionDenied(r.Context(), "create", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)

	// CAP-V12: a FormView declaring Steps renders as a multi-step wizard
	// instead of the single Form -- step 0, no carried-forward values yet.
	if fv := h.interp.Get().FormView(machine.ID); fv != nil && len(fv.Config.Steps) > 0 {
		page := ui.WizardForm(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, 0, len(fv.Config.Steps), h.buildFormFieldsFor(r.Context(), machine, fv.Config.Steps[0], nil), nil, nil, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
		if err := page.Render(r.Context(), w); err != nil {
			slog.Error("render wizard form", "error", err)
		}
		return
	}

	page := ui.Form(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, "", h.buildFormFields(r.Context(), machine, nil), nil, h.unreadCount(r.Context(), a), h.buildChildLinesData(r.Context(), machine), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render form", "error", err)
	}
}

// Create — handle new record form submission.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	if role := h.roleForApp(r, applicationID); !h.guard.CanCreate(machine, role) {
		h.logPermissionDenied(r.Context(), "create", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	fv := h.interp.Get().FormView(machine.ID)

	// CAP-V12: an intermediate wizard-step submission renders the NEXT step
	// instead of creating anything -- only the final step's POST falls
	// through to the ordinary Create logic below. No session state: every
	// prior step's value travels forward as a hidden input on each step's
	// page, so by the final POST every field is present in r.Form exactly
	// like a single-step form's would be.
	if fv != nil && len(fv.Config.Steps) > 0 {
		step, convErr := strconv.Atoi(r.FormValue("wizard_step"))
		if convErr != nil || step < 0 || step >= len(fv.Config.Steps) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if step+1 < len(fv.Config.Steps) {
			var carried []ui.HiddenField
			for i := 0; i <= step; i++ {
				for _, id := range fv.Config.Steps[i] {
					if v := r.FormValue(id); v != "" {
						carried = append(carried, ui.HiddenField{Name: id, Value: v})
					}
				}
			}
			a := h.auth(r)
			page := ui.WizardForm(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, step+1, len(fv.Config.Steps),
				h.buildFormFieldsFor(r.Context(), machine, fv.Config.Steps[step+1], nil), carried, nil, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
			if err := page.Render(r.Context(), w); err != nil {
				slog.Error("render wizard form", "error", err)
			}
			return
		}
		// Final step -- fall through; every field (this step's and every
		// earlier one's, carried as hidden inputs) is in r.Form already.
	}

	// Any value_list field the Create form doesn't expose (Status, Decision, ...)
	// starts at its first declared value — the same "first value = initial
	// state" convention guides/writing-menata.md teaches .menata authors,
	// generalized from what was previously a "status"-named-field-only rule.
	formFieldIDs := map[string]bool{}
	if fv != nil {
		for _, id := range fv.Config.Fields {
			formFieldIDs[id] = true
		}
		for _, step := range fv.Config.Steps {
			for _, id := range step {
				formFieldIDs[id] = true
			}
		}
	}
	data := make(map[string]any)
	for _, f := range machine.Fields {
		if formFieldIDs[f.ID] || f.Type == model.FieldTypeComputed {
			continue
		}
		if f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
			data[f.ID] = f.Options.Values[0]
		} else if f.Options.Default != "" {
			// CAP-F15: any field's own declared default, generalized from
			// the value_list-only convention above -- applies whenever the
			// Create form doesn't expose the field at all.
			data[f.ID] = f.Options.Default
		}
	}
	for _, f := range machine.Fields {
		if f.Type == model.FieldTypeComputed {
			continue // CAP-F14: never a stored value, never read from a form
		}
		if v := r.FormValue(f.ID); v != "" {
			data[f.ID] = v
		} else if f.Type == model.FieldTypeBoolean && formFieldIDs[f.ID] {
			data[f.ID] = "false" // CAP-F09: an unchecked checkbox submits nothing at all
		} else if f.Options.Default != "" && formFieldIDs[f.ID] {
			data[f.ID] = f.Options.Default // CAP-F15: exposed but left blank
		}
	}
	// CAP-F06: `file` fields' actual uploaded bytes -- csrfProtect
	// (cmd/server/main.go) already called ParseMultipartForm for a
	// multipart request before this handler ever runs, so r.MultipartForm
	// is already populated here.
	uploaded, err := h.processFileUploads(r, machine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for fieldID, key := range uploaded {
		data[fieldID] = key
	}

	// CAP-F18: an auto-number field left blank gets the next sequence value
	// -- atomic per (machine, field), never a submitter-supplied string.
	for _, f := range machine.Fields {
		if f.Options.AutoNumberPrefix == "" {
			continue
		}
		if v, ok := data[f.ID]; ok && fmt.Sprintf("%v", v) != "" {
			continue
		}
		n, err := h.records.NextSequence(r.Context(), machineID, f.ID)
		if err != nil {
			http.Error(w, "failed to generate document number", http.StatusInternalServerError)
			return
		}
		data[f.ID] = formatAutoNumber(f.Options, n)
	}

	// CAP-R08: see the matching comment in Update -- a record created
	// directly into its declared "scratch" state skips business-rule
	// Constraints too (referential integrity still applies, below).
	var violations []string
	if !h.inScratchState(machine, data) {
		violations = h.engine.Violations(machine, withChangePolicyCreatedAt(machine, data, time.Now()))
	}
	refViolations, err := h.referenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, refViolations...)
	userViolations, err := h.userReferenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, userViolations...)
	uniqueViolations, err := h.uniquenessViolations(r.Context(), machine, data, "")
	if err != nil {
		http.Error(w, "failed to validate uniqueness", http.StatusInternalServerError)
		return
	}
	violations = append(violations, uniqueViolations...)

	// CAP-F16: a form with embedded child rows validates them together with
	// the parent -- one combined violations list, so a bad child row blocks
	// the whole submission exactly like a bad parent field would, before
	// anything (parent or child) is written.
	var childLines *model.ChildLinesConfig
	var childRowsData []map[string]any
	if fv != nil {
		childLines = fv.Config.ChildLines
	}
	if childLines != nil {
		rows, rowViolations, err := h.validateChildRows(r.Context(), r, childLines)
		if err != nil {
			http.Error(w, "failed to validate child rows", http.StatusInternalServerError)
			return
		}
		childRowsData = rows
		violations = append(violations, rowViolations...)
	}

	if len(violations) > 0 {
		role := h.roleForApp(r, applicationID)
		h.logRuleViolation(r.Context(), "create", machineID, "", role, h.identity(r), strings.Join(violations, "; "))
		a := h.auth(r)
		// CAP-V12: a violation on the wizard's final step re-renders that
		// same last step (with everything typed so far preserved), not the
		// plain single-step Form -- a wizard View's own Fields is empty
		// (Steps replaces it), so ui.Form would otherwise render nothing.
		if fv != nil && len(fv.Config.Steps) > 0 {
			last := len(fv.Config.Steps) - 1
			var carried []ui.HiddenField
			for i := 0; i < last; i++ {
				for _, id := range fv.Config.Steps[i] {
					if v, ok := data[id]; ok {
						carried = append(carried, ui.HiddenField{Name: id, Value: fmt.Sprintf("%v", v)})
					}
				}
			}
			page := ui.WizardForm(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, last, len(fv.Config.Steps),
				h.buildFormFieldsFor(r.Context(), machine, fv.Config.Steps[last], data), carried, violations, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
			if err := page.Render(r.Context(), w); err != nil {
				slog.Error("render wizard form (violations)", "error", err)
			}
			return
		}
		page := ui.Form(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, "", h.buildFormFields(r.Context(), machine, data), violations, h.unreadCount(r.Context(), a), h.buildChildLinesData(r.Context(), machine), h.subNavFor(r, machine))
		if err := page.Render(r.Context(), w); err != nil {
			slog.Error("render form (violations)", "error", err)
		}
		return
	}

	rec, err := h.records.Create(r.Context(), machineID, workspaceID, data)
	if err != nil {
		http.Error(w, "failed to create record", http.StatusInternalServerError)
		return
	}
	if childLines != nil {
		if err := h.insertChildRows(r.Context(), childLines, workspaceID, childRowsData, rec.ID); err != nil {
			http.Error(w, "failed to create child rows", http.StatusInternalServerError)
			return
		}
	}
	// CAP-W01: write-time fan-in -- if this record references a Machine
	// whose Process declares a Requirement naming machineID as its target,
	// stamp that parent's counter now. Error propagates to a real 5xx (the
	// CAP-X12 lesson: a swallowed error here would leave the child
	// committed but its parent's counter silently stale) so workspaceTx
	// rolls the whole request back, not just this write.
	if err := h.stampRequirementCounters(r.Context(), machine, rec); err != nil {
		http.Error(w, "failed to update requirement counter", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+rec.ID, http.StatusSeeOther)
}

// EditForm — form for editing an existing record (CAP-R02). Reuses the same
// FormView/field set Create uses — Menata Language has no separate "edit
// form" view declared in metadata — pre-filled with the record's current
// data.
func (h *Handler) EditForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanEdit(machine, role) {
		h.logPermissionDenied(r.Context(), "edit", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// CAP-R07: an immutable record's edit form isn't even offered -- Update
	// re-checks the same gate as defense in depth, same "guard the mutation
	// path, not just the button" reasoning CAP-P05 already uses elsewhere.
	if reason := h.immutabilityViolation(machine, rec.Data); reason != "" {
		http.Error(w, reason, http.StatusForbidden)
		return
	}
	a := h.auth(r)
	page := ui.Form(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, recordID, h.buildFormFields(r.Context(), machine, rec.Data), nil, h.unreadCount(r.Context(), a), nil, h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render edit form", "error", err)
	}
}

// Update — handle edit form submission (CAP-R02). Only the fields the
// FormView exposes are overwritten; everything else on the record (Status,
// and any other field driven by events rather than the form) is carried
// over unchanged, the same "only touch what the form declares" rule Create
// applies to its value_list defaults.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	if role := h.roleForApp(r, applicationID); !h.guard.CanEdit(machine, role) {
		h.logPermissionDenied(r.Context(), "edit", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// CAP-R07: defense in depth -- EditForm already refuses to render for
	// an immutable record, but Update is reachable directly (a replayed
	// form, a hand-crafted POST) without going through that form first.
	if reason := h.immutabilityViolation(machine, rec.Data); reason != "" {
		http.Error(w, reason, http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	formFieldIDs := map[string]bool{}
	if fv := h.interp.Get().FormView(machine.ID); fv != nil {
		for _, id := range fv.Config.Fields {
			formFieldIDs[id] = true
		}
	}
	data := make(map[string]any, len(rec.Data))
	for k, v := range rec.Data {
		data[k] = v
	}
	for _, f := range machine.Fields {
		if !formFieldIDs[f.ID] || f.Type == model.FieldTypeComputed {
			continue // CAP-F14: never a stored value, never read from a form
		}
		if f.Type == model.FieldTypeBoolean {
			// CAP-F09: an unchecked checkbox submits nothing at all --
			// FormValue("") would otherwise silently write an empty string
			// instead of "false".
			if r.FormValue(f.ID) == "true" {
				data[f.ID] = "true"
			} else {
				data[f.ID] = "false"
			}
			continue
		}
		if f.Type == model.FieldTypeFile {
			// CAP-F06: a file input's FormValue is always "" (the browser
			// sends the bytes as a multipart file part, not a form value)
			// -- leave the record's existing stored key alone here; a real
			// new upload (if any) overlays it below, AFTER this loop.
			continue
		}
		data[f.ID] = r.FormValue(f.ID)
	}
	uploaded, err := h.processFileUploads(r, machine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for fieldID, key := range uploaded {
		data[fieldID] = key
	}

	// CAP-R08: a record still in its declared "scratch" state (e.g. a Cart
	// before Checkout) has none of its eventual business-rule Constraints
	// enforced yet -- CAP-C09's own trigger-time re-validation is the real
	// commit-point gate, once an event moves it out of that state.
	// Referential integrity (below) still applies even in scratch state --
	// a scratch record can be incomplete, not corrupt.
	var violations []string
	if !h.inScratchState(machine, data) {
		violations = h.engine.Violations(machine, withChangePolicyCreatedAt(machine, data, rec.CreatedAt))
	}
	refViolations, err := h.referenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, refViolations...)
	userViolations, err := h.userReferenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, userViolations...)
	uniqueViolations, err := h.uniquenessViolations(r.Context(), machine, data, recordID)
	if err != nil {
		http.Error(w, "failed to validate uniqueness", http.StatusInternalServerError)
		return
	}
	violations = append(violations, uniqueViolations...)

	if len(violations) > 0 {
		role := h.roleForApp(r, applicationID)
		h.logRuleViolation(r.Context(), "update", machineID, "", role, h.identity(r), strings.Join(violations, "; "))
		a := h.auth(r)
		page := ui.Form(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, recordID, h.buildFormFields(r.Context(), machine, data), violations, h.unreadCount(r.Context(), a), nil, h.subNavFor(r, machine))
		if err := page.Render(r.Context(), w); err != nil {
			slog.Error("render form (violations)", "error", err)
		}
		return
	}

	if err := h.records.Update(r.Context(), recordID, data); err != nil {
		http.Error(w, "failed to update record", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+recordID, http.StatusSeeOther)
}

// Detail — detail view of a single record.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")

	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// CAP-P06: a field this role's Permission hides never reaches the
	// Detail page.
	hidden := h.hiddenFields(machine, role)
	fields := make([]ui.DetailField, 0, len(machine.Fields))
	for _, f := range machine.Fields {
		if hidden[f.ID] {
			continue
		}
		val := ""
		if v, ok := rec.Data[f.ID]; ok {
			val = fmt.Sprintf("%v", v)
		}
		link := ""
		switch {
		case f.Type == model.FieldTypeReference && val != "":
			refID := val
			target := f.Options.TargetMachine
			if label, err := h.referenceLabel(r.Context(), target, refID); err == nil && label != "" {
				val = label
				link = "/" + target + "/" + refID
			}
		case f.Type == model.FieldTypeUser && val != "":
			if label, err := h.userLabel(r.Context(), val); err == nil && label != "" {
				val = label
			}
		case f.Type == model.FieldTypeBoolean:
			val = boolLabel(val) // CAP-F09
		case f.Type == model.FieldTypeMoney && val != "":
			val = formatMoney(val, f, rec.Data) // CAP-F08
		case f.Type == model.FieldTypeFile && val != "":
			link = "/files/" + val // CAP-F06
		case f.Type == model.FieldTypeComputed:
			val = computedValue(f, rec.Data) // CAP-F14
		}
		fields = append(fields, ui.DetailField{Name: f.Name, Value: val, Link: link})
	}

	childLists := h.childLists(r.Context(), machine, recordID)
	events := h.interp.Get().PermittedEventsForRecord(machineID, role, h.identityID(r), rec.Data)
	// CAP-P04: an event declaring InputFields (e.g. "delegate to") renders
	// an inline picker alongside its trigger button, same field/options
	// shape a Form uses -- built here, not in ui, since resolving a `user`
	// field's own picker options needs the UserStore (buildFormFieldsFor).
	permittedEvents := make([]ui.EventTrigger, len(events))
	for i, evt := range events {
		permittedEvents[i] = ui.EventTrigger{Event: evt}
		if len(evt.InputFields) > 0 {
			permittedEvents[i].Inputs = h.buildFormFieldsFor(r.Context(), machine, evt.InputFields, nil)
		}
	}
	a := h.auth(r)
	page := ui.Detail(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, rec, fields, permittedEvents, childLists, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render detail", "error", err)
	}
}

// TriggerEvent — handle event button.
func (h *Handler) TriggerEvent(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	eventID := chi.URLParam(r, "eventID")

	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	event, ok := h.interp.Get().GetEvent(machineID, eventID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	role, identity := h.roleForApp(r, applicationID), h.identity(r)
	// CAP-P02 — ownership check needs the record's own data, so this can't
	// run until after the record fetch above (unlike role-only permission,
	// which didn't need it). Compared by id (CAP-F05), not by identity's
	// display name -- triggerEvent below still gets the human-readable
	// identity, for audit attribution and CAP-A02's current_user.
	if !h.guard.CanTrigger(machine, role, h.identityID(r), eventID, rec.Data) {
		h.logPermissionDenied(r.Context(), "trigger", machineID, eventID, role, identity)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}

	// CAP-P04: an event declaring InputFields (e.g. "delegate to") needs
	// fresh values submitted alongside this trigger, not read from the
	// record's own data -- collected here, resolved by set_field's
	// "input:<field>" (Executor.resolveValue).
	var eventInput map[string]string
	if len(event.InputFields) > 0 {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		eventInput = make(map[string]string, len(event.InputFields))
		for _, id := range event.InputFields {
			eventInput[id] = r.FormValue(id)
		}
	}

	if err := h.triggerEvent(r.Context(), machine, event, rec, role, identity, h.identityID(r), eventInput); err != nil {
		var rv *ruleViolation
		if errors.As(err, &rv) {
			h.logRuleViolation(r.Context(), "trigger", machineID, eventID, role, identity, rv.Error())
			http.Error(w, rv.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, "event failed", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+recordID, http.StatusSeeOther)
}

// Webhook (CAP-E04) lets an external system trigger an event directly --
// no session, no CSRF token (cmd/server/main.go's isPublicPath/
// csrfProtect both carve this path out), authenticated instead by a
// per-Machine shared secret (Machine.Config's "webhook_secret", CAP-X03's
// existing generic settings -- not a new migration column). A Machine that
// hasn't declared one 404s -- webhooks are opt-in per Machine, not a
// blanket surface every Machine gets. No role-based Guard.CanTrigger check
// either: the secret itself is the authorization here, the same posture
// CAP-A08/CAP-E05's internal "System"-triggered cascades already take
// (see doAggregateStatus/doTriggerEvent) -- an external caller isn't a
// person holding a role, there's nothing for CanTrigger's ownership/role
// check to mean.
//
// A payment webhook's own payload can carry values too, the same
// InputFields/"input:<field>" mechanism CAP-P04 already built for
// delegation's "who to hand off to" -- JSON body if Content-Type says so
// (the realistic shape for most real webhook providers), form-encoded
// otherwise.
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	eventID := chi.URLParam(r, "eventID")

	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	secret := machine.Config["webhook_secret"]
	if secret == "" {
		http.NotFound(w, r)
		return
	}
	if !auth.ConstantTimeEqual(r.Header.Get("X-Webhook-Secret"), secret) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	event, ok := h.interp.Get().GetEvent(machineID, eventID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}

	// CAP-X13: an idempotency key is opt-in -- a caller retrying the exact
	// same delivery (network timeout, at-least-once queue redelivery) sends
	// the same X-Idempotency-Key and gets 200 back without the event firing
	// a second time. No key at all skips the check entirely (unchanged
	// behavior for every webhook caller that doesn't supply one).
	if key := r.Header.Get("X-Idempotency-Key"); key != "" {
		claimed, err := h.records.ClaimWebhookEvent(r.Context(), machineID, eventID, key)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !claimed {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	var eventInput map[string]string
	if len(event.InputFields) > 0 {
		eventInput = make(map[string]string, len(event.InputFields))
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				for _, id := range event.InputFields {
					if v, ok := body[id]; ok {
						eventInput[id] = fmt.Sprintf("%v", v)
					}
				}
			}
		} else if err := r.ParseForm(); err == nil {
			for _, id := range event.InputFields {
				eventInput[id] = r.FormValue(id)
			}
		}
	}

	if err := h.triggerEvent(r.Context(), machine, event, rec, "System", "Webhook", "", eventInput); err != nil {
		var rv *ruleViolation
		if errors.As(err, &rv) {
			h.logRuleViolation(r.Context(), "webhook", machineID, eventID, "System", "Webhook", rv.Error())
			http.Error(w, rv.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, "event failed", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ruleViolation marks a rejection as a business-rule failure (400), as
// opposed to an infrastructure error (500) — the same distinction Create
// already draws by rendering violations back into the form instead of
// erroring out.
type ruleViolation struct{ msg string }

func (e *ruleViolation) Error() string { return e.msg }

// logPermissionDenied records an access-control failure as a distinct,
// security-relevant log line (OWASP ASVS V7.2 — "all failed access control
// decisions must be logged"), not just the routine access-log entry chi's
// middleware.Logger already writes for the resulting 403 response.
// correlation_id is the same request id CAP-R04's record_events rows carry
// (executor.Persist), so a probing pattern (many denials across roles in one
// session) and any successful mutation from the same request can be
// correlated. eventID is empty for non-event actions (read/create/edit).
// workspace/application (006-runtime-model.md's hierarchy) are resolved
// here, not passed in, so every call site stays a one-liner.
func (h *Handler) logPermissionDenied(ctx context.Context, action, machineID, eventID, role, identity string) {
	workspaceID, appID := h.interp.Get().ScopeFor(machineID)
	slog.Warn("permission denied",
		"correlation_id", middleware.GetReqID(ctx),
		"workspace", workspaceID,
		"application", appID,
		"action", action,
		"machine", machineID,
		"event", eventID,
		"role", role,
		"identity", identity,
	)
}

// logRuleViolation records a rejected write (CAP-C01..C11 constraints, or
// CAP-E06's state guard) as a distinct log line — ASVS V7.1's "all input
// validation failures" — separate from logPermissionDenied's access-control
// failures: this is a request that *was* permitted but rejected on the
// data/state it carried.
func (h *Handler) logRuleViolation(ctx context.Context, action, machineID, eventID, role, identity, reason string) {
	workspaceID, appID := h.interp.Get().ScopeFor(machineID)
	slog.Warn("rule violation",
		"correlation_id", middleware.GetReqID(ctx),
		"workspace", workspaceID,
		"application", appID,
		"action", action,
		"machine", machineID,
		"event", eventID,
		"role", role,
		"identity", identity,
		"reason", reason,
	)
}

// triggerEvent is the single path every event trigger runs through, whether
// from an HTTP button (TriggerEvent) or fired internally by another event's
// aggregate_status or trigger_event action (CAP-A08/CAP-E05, doAggregateStatus/
// doTriggerEvent) — same guards, same validation, same persistence, so a
// system-triggered transition can never skip a check a user-triggered one
// would have to pass.
func (h *Handler) triggerEvent(ctx context.Context, machine *model.Machine, event *model.Event, rec *store.Record, actorRole, actorIdentity, actorIdentityID string, eventInput map[string]string) error {
	// CAP-I02 — a deprecated Event still works (backward compat by design)
	// but logs a warning every time it's actually used, so a real
	// deprecation can be tracked/acted on instead of staying invisible.
	if event.DeprecatedMessage != "" {
		slog.Warn("triggered deprecated event", "event", event.ID, "machine", machine.ID, "message", event.DeprecatedMessage)
	}

	// CAP-E06 — state guard: the event may only fire when the record's
	// CURRENT data satisfies its condition (e.g. Reject only from Submitted).
	if event.Condition != nil && !constraint.Eval(*event.Condition, rec.Data) {
		return &ruleViolation{fmt.Sprintf("%s is not allowed in the record's current state", event.Name)}
	}

	// CAP-P03 — separation of duties: the person who submitted the PARENT
	// record (looked up via this record's own reference field) may not also
	// decide this one, even if they happen to hold the deciding role too --
	// cross-record, so like CAP-A07's sequential guard it can't be expressed
	// as an event Condition (CAP-E06 only reads this record's own data).
	// actorIdentityID == "" (System-triggered events, doAggregateStatus) is
	// exempt -- there's no human actor to self-deal.
	if actorIdentityID != "" {
		if msg, err := h.separationOfDutiesViolation(ctx, machine, rec, actorIdentityID); err != nil {
			return err
		} else if msg != "" {
			return &ruleViolation{msg}
		}
	}

	// CAP-A07 — sequential step guard: in a Sequential-mode parent, a step may
	// only be decided once every sibling with a lower Sequence has already
	// left Pending. Cross-record, so it can't be expressed as an event
	// Condition (CAP-E06 only reads the record's own data).
	if msg := h.sequentialGuardViolation(ctx, machine, event, rec); msg != "" {
		return &ruleViolation{msg}
	}

	// CAP-A14 — aggregate-conditioned guard: the same idea as CAP-E06's
	// Condition, computed across sibling records instead of this one's own
	// fields (e.g. "only once this Member's total Points reach 100").
	if msg, err := h.aggregateConditionViolation(ctx, event, rec); err != nil {
		return err
	} else if msg != "" {
		return &ruleViolation{msg}
	}

	// CAP-C09 — constraints evaluated on event trigger, not just Create:
	// simulate the event's effect first, validate the result, only persist
	// if it still satisfies every declared Constraint. CAP-A02 — actorIdentity
	// (falling back to actorRole) resolves this event's "current_user" dynamic
	// values, if any. CAP-O06 — holidays resolved once here, passed through
	// to every "N Business Days" resolution Simulate/Persist might do.
	workspaceID, _ := h.interp.Get().ScopeFor(machine.ID)
	holidays := h.interp.Get().Holidays(workspaceID)
	newData := h.exec.Simulate(machine, event, rec, actorRole, actorIdentity, eventInput, holidays)
	if violations := h.engine.Violations(machine, withChangePolicyCreatedAt(machine, newData, rec.CreatedAt)); len(violations) > 0 {
		return &ruleViolation{strings.Join(violations, " ")}
	}

	if err := h.exec.Persist(ctx, machine, event, rec, newData, machine.Name, actorRole, actorIdentity, workspaceID, holidays); err != nil {
		return err
	}

	// CAP-A07/CAP-A08/CAP-E05 — workflow actions run after a successful
	// commit, using the now-current data: activate_next notifies the next
	// pending sibling, aggregate_status may internally trigger a rollup event
	// on the parent, trigger_event may fire another event on this same record.
	h.runWorkflowActions(ctx, machine, event, rec.ID, newData)

	// CAP-I01 — cross-machine subscribers run last, after the publisher's
	// OWN write and workflow actions have already succeeded (error rule 1:
	// a subscriber's failure never rolls back the publisher).
	h.processSubscriptions(ctx, event, newData, actorRole, actorIdentity, workspaceID, holidays)
	return nil
}

// processSubscriptions (CAP-I01) dispatches every Subscription declaring
// interest in event.ID -- one new record per Subscription, on its own
// MachineID, Fields resolved from the publisher's post-event data.
// CAP-I03's Contract/OnViolation gate each one independently first. Error
// rules 2-4 (see model.Subscription's own doc comment): each Subscription
// is processed regardless of whether an earlier one failed; every failure
// is logged, never silently dropped; every Subscription sees the same
// final data, never a partial view.
func (h *Handler) processSubscriptions(ctx context.Context, event *model.Event, data map[string]any, actorRole, actorIdentity, workspaceID string, holidays map[string]bool) {
	for _, sub := range h.interp.Get().SubscriptionsFor(event.ID) {
		violated := false
		for _, c := range sub.Contract {
			if !constraint.Eval(c, data) {
				violated = true
				slog.Warn("subscription contract violation", "subscription", sub.ID, "publisher_event", event.ID, "field", c.Field, "on_violation", sub.OnViolation)
				break
			}
		}
		if violated && sub.OnViolation == "skip" {
			continue
		}
		fields := h.exec.ResolveFields(sub.Fields, data, actorRole, actorIdentity, holidays)
		if _, err := h.records.Create(ctx, sub.MachineID, workspaceID, fields); err != nil {
			slog.Error("subscription failed", "subscription", sub.ID, "publisher_event", event.ID, "machine", sub.MachineID, "error", err)
		}
	}
}

// --- helpers -----------------------------------------------------------------

func fieldIndex(m *model.Machine) map[string]*model.Field {
	out := make(map[string]*model.Field, len(m.Fields))
	for _, f := range m.Fields {
		out[f.ID] = f
	}
	return out
}

// hiddenFields (CAP-P06) returns the set of field ids role's own Permission
// row on machine excludes from List/Detail/Form rendering -- "Salary
// visible only to HR." A role with no matching Permission row (or one that
// doesn't declare hidden_fields) hides nothing extra -- CAP-P05's
// deny-by-default already governs whether the role can see the Machine at
// all; this is a narrower, opt-in restriction on top of that.
// subNavFor (CAP-O03 Tier 2) resolves the persistent, Application-scoped
// sub-nav strip for a page belonging to machine -- that Machine's own
// sibling Machines in the same Application, permission-trimmed the same
// way AppMachines' own landing page already is, so a user can move
// sideways between an app's own features without returning to the
// workspace home. Reuses the exact data link AppMachines already
// resolves (ScopeFor/MachinesForApplication) -- see benchmarks/009 for
// the full reasoning.
func (h *Handler) subNavFor(r *http.Request, machine *model.Machine) []ui.SubNavLink {
	_, applicationID := h.interp.Get().ScopeFor(machine.ID)
	siblings := h.interp.Get().MachinesForApplication(applicationID)
	if len(siblings) < 2 {
		return nil // nothing to move sideways to
	}
	role := h.roleForApp(r, applicationID)
	links := make([]ui.SubNavLink, 0, len(siblings))
	for _, m := range siblings {
		if !h.guard.CanRead(m, role) {
			continue
		}
		links = append(links, ui.SubNavLink{ID: m.ID, Name: m.Name, Active: m.ID == machine.ID})
	}
	if len(links) < 2 {
		return nil // permission-trimmed down to nothing worth switching between
	}
	return links
}

func (h *Handler) hiddenFields(machine *model.Machine, role string) map[string]bool {
	out := map[string]bool{}
	for _, perm := range machine.Permissions {
		if perm.Role == role {
			for _, id := range perm.HiddenFields {
				out[id] = true
			}
			break
		}
	}
	return out
}

func (h *Handler) buildFormFields(ctx context.Context, machine *model.Machine, vals map[string]any) []ui.FormField {
	var fieldIDs []string
	if view := h.interp.Get().FormView(machine.ID); view != nil {
		fieldIDs = view.Config.Fields
	}
	return h.buildFormFieldsFor(ctx, machine, fieldIDs, vals)
}

// buildFormFieldsFor is buildFormFields narrowed to an explicit fieldIDs
// subset -- CAP-V12's own use, one step's worth of fields at a time, rather
// than a FormView's whole declared set.
func (h *Handler) buildFormFieldsFor(ctx context.Context, machine *model.Machine, fieldIDs []string, vals map[string]any) []ui.FormField {
	fieldByID := fieldIndex(machine)

	fields := make([]ui.FormField, 0, len(fieldIDs))
	for _, id := range fieldIDs {
		f, ok := fieldByID[id]
		if !ok {
			continue
		}
		val := ""
		if vals != nil {
			if v, ok := vals[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
		}
		var opts []ui.ReferenceOption
		switch f.Type {
		case model.FieldTypeReference:
			opts = h.referenceOptions(ctx, f.Options.TargetMachine)
		case model.FieldTypeUser:
			opts = h.userFieldOptions(ctx, machine.ApplicationID)
		}
		fields = append(fields, ui.FormField{Field: f, Name: f.ID, Value: val, Options: opts})
	}
	return fields
}

// childRowName (CAP-F16) builds a repeated child-line row's indexed HTML
// input name -- "child_0_fld_x", "child_1_fld_x", ... -- so a fixed-slot
// row-editor's fields don't collide with each other or the parent form's
// own field names. parseChildRows below is this function's exact inverse.
func childRowName(row int, fieldID string) string {
	return fmt.Sprintf("child_%d_%s", row, fieldID)
}

// buildChildLinesData (CAP-F16) builds a form's embedded child-Machine row
// editor, if the FormView declares one -- MaxRows blank row slots (or
// MaxRows pre-filled from existingRows on... no: CREATE-time only, see
// ChildLinesConfig's own doc comment for why edit doesn't use this).
// existingRows is nil at Create; kept as a param so a future edit-time
// extension has an obvious seam, not because anything passes it non-nil today.
func (h *Handler) buildChildLinesData(ctx context.Context, machine *model.Machine) *ui.ChildLinesData {
	view := h.interp.Get().FormView(machine.ID)
	if view == nil || view.Config.ChildLines == nil {
		return nil
	}
	cl := view.Config.ChildLines
	childMachine, ok := h.interp.Get().GetMachine(cl.Machine)
	if !ok {
		return nil
	}
	fieldByID := fieldIndex(childMachine)
	maxRows := cl.MaxRows
	if maxRows <= 0 {
		maxRows = 10
	}

	rows := make([][]ui.FormField, maxRows)
	for i := 0; i < maxRows; i++ {
		row := make([]ui.FormField, 0, len(cl.Fields))
		for _, fid := range cl.Fields {
			f, ok := fieldByID[fid]
			if !ok || fid == cl.ParentField {
				continue
			}
			var opts []ui.ReferenceOption
			switch f.Type {
			case model.FieldTypeReference:
				opts = h.referenceOptions(ctx, f.Options.TargetMachine)
			case model.FieldTypeUser:
				opts = h.userFieldOptions(ctx, childMachine.ApplicationID)
			}
			row = append(row, ui.FormField{Field: f, Name: childRowName(i, fid), Options: opts})
		}
		rows[i] = row
	}
	return &ui.ChildLinesData{Title: childMachine.Name, Rows: rows}
}

// validateChildRows (CAP-F16) reads every non-empty row the fixed-slot
// child row editor submitted and validates each against the child
// Machine's own Constraints (CAP-C01..C12, the same rules a direct Create
// on that Machine would run) -- except any constraint on ParentField
// itself, which deliberately isn't set yet here (the parent record this
// row will reference doesn't exist until AFTER the parent insert that
// follows a clean validation pass -- see Create's own two-phase call to
// this then insertChildRows). A row is "empty" (silently skipped, not an
// error) when every one of its fields was left blank -- matches the UI's
// own "leave a row blank to skip it" hint. Returned data never has
// ParentField set; insertChildRows adds it once the parent's real id exists.
func (h *Handler) validateChildRows(ctx context.Context, r *http.Request, cl *model.ChildLinesConfig) ([]map[string]any, []string, error) {
	childMachine, ok := h.interp.Get().GetMachine(cl.Machine)
	if !ok {
		return nil, nil, fmt.Errorf("child_lines.machine %q not found", cl.Machine)
	}
	maxRows := cl.MaxRows
	if maxRows <= 0 {
		maxRows = 10
	}

	var rowsData []map[string]any
	var violations []string
	for i := 0; i < maxRows; i++ {
		data := make(map[string]any)
		anySet := false
		for _, fid := range cl.Fields {
			if fid == cl.ParentField {
				continue
			}
			v := r.FormValue(childRowName(i, fid))
			if v != "" {
				data[fid] = v
				anySet = true
			}
		}
		if !anySet {
			continue
		}

		var rowViolations []string
		for _, c := range childMachine.Constraints {
			if c.Expression.Field == cl.ParentField || c.Expression.Operator == "unique" {
				// ParentField's own constraints (e.g. "required") don't
				// apply yet -- it isn't set until insertChildRows. Uniqueness
				// needs the database (handler.uniquenessViolations below),
				// not constraint.Eval, same as everywhere else this
				// distinction is made.
				continue
			}
			if c.Condition != nil && !constraint.Eval(*c.Condition, data) {
				continue
			}
			if !constraint.Eval(c.Expression, data) {
				rowViolations = append(rowViolations, c.Rule)
			}
		}
		refViolations, err := h.referenceViolations(ctx, childMachine, data)
		if err != nil {
			return nil, nil, err
		}
		rowViolations = append(rowViolations, refViolations...)
		userViolations, err := h.userReferenceViolations(ctx, childMachine, data)
		if err != nil {
			return nil, nil, err
		}
		rowViolations = append(rowViolations, userViolations...)
		for _, v := range rowViolations {
			violations = append(violations, fmt.Sprintf("%s, row %d: %s", childMachine.Name, i+1, v))
		}
		rowsData = append(rowsData, data)
	}
	return rowsData, violations, nil
}

// insertChildRows (CAP-F16) writes every validated child row from
// validateChildRows, stamping ParentField with the just-created parent's
// real id on each -- called only after that parent insert has succeeded,
// so every row's back-reference is valid by construction, never a
// dangling one validateChildRows would have had to reject.
func (h *Handler) insertChildRows(ctx context.Context, cl *model.ChildLinesConfig, workspaceID string, rowsData []map[string]any, parentID string) error {
	for _, data := range rowsData {
		data[cl.ParentField] = parentID
		if _, err := h.records.Create(ctx, cl.Machine, workspaceID, data); err != nil {
			return err
		}
	}
	return nil
}

// childLists finds every Machine with a `reference` field pointing at
// machine, and lists the records where that field equals recordID (CAP-V06).
// Generic by construction — it doesn't special-case Employee/Manager, so any
// future reference relationship gets a sub-list automatically.
func (h *Handler) childLists(ctx context.Context, machine *model.Machine, recordID string) []ui.ChildList {
	var out []ui.ChildList
	for _, m := range h.interp.Get().AllMachines() {
		for _, f := range m.Fields {
			if f.Type != model.FieldTypeReference || f.Options.TargetMachine != machine.ID {
				continue
			}
			records, err := h.records.List(ctx, m.ID, "", "")
			if err != nil {
				slog.Error("list child records", "machine", m.ID, "error", err)
				continue
			}
			var items []ui.ChildListItem
			for _, rec := range records {
				v, ok := rec.Data[f.ID]
				if !ok {
					continue
				}
				if refID, _ := v.(string); refID == recordID {
					items = append(items, ui.ChildListItem{
						Label: displayLabel(m, rec.Data),
						Link:  "/" + m.ID + "/" + rec.ID,
					})
				}
			}
			if len(items) > 0 {
				out = append(out, ui.ChildList{Title: fmt.Sprintf("%s (via %s)", m.Name, f.Name), Items: items})
			}
		}
	}
	return out
}

func (h *Handler) referenceOptions(ctx context.Context, targetMachineID string) []ui.ReferenceOption {
	records, err := h.records.List(ctx, targetMachineID, "", "")
	if err != nil {
		slog.Error("list reference options", "target_machine", targetMachineID, "error", err)
		return nil
	}
	targetMachine, _ := h.interp.Get().GetMachine(targetMachineID)
	opts := make([]ui.ReferenceOption, 0, len(records))
	for _, rec := range records {
		opts = append(opts, ui.ReferenceOption{ID: rec.ID, Label: displayLabel(targetMachine, rec.Data)})
	}
	return opts
}

// referenceLabel resolves one record's display label, for rendering an
// already-set reference value (detail/list views).
func (h *Handler) referenceLabel(ctx context.Context, targetMachineID, recordID string) (string, error) {
	rec, err := h.records.Get(ctx, recordID)
	if err != nil {
		return "", err
	}
	targetMachine, _ := h.interp.Get().GetMachine(targetMachineID)
	return displayLabel(targetMachine, rec.Data), nil
}

// displayLabel picks a human-readable stand-in for a record: the target
// Machine's `text` field named "Name" if one exists, else its first `text`
// field, falling back to the record id. Menata Language doesn't (yet) let a
// business author declare "this is the field people should see when
// referencing a record" — this heuristic is a prototype stand-in for that
// missing capability, not a final design.
// formatAutoNumber (CAP-F18) renders a sequence value as "<prefix><padded
// number>", e.g. prefix "INV-" + padding 4 + n=7 -> "INV-0007". padding 0
// (or omitted) means no zero-padding at all -- just the prefix and the
// plain number.
func formatAutoNumber(opts model.FieldOptions, n int64) string {
	if opts.AutoNumberPadding > 0 {
		return fmt.Sprintf("%s%0*d", opts.AutoNumberPrefix, opts.AutoNumberPadding, n)
	}
	return fmt.Sprintf("%s%d", opts.AutoNumberPrefix, n)
}

// boolLabel (CAP-F09) renders a boolean field's stored "true"/"false"
// string as a human-readable Yes/No -- anything else (unset, malformed) is
// treated as No, the same "absent = false" convention Create/Update's own
// checkbox handling already uses.
func boolLabel(val string) string {
	if val == "true" {
		return "Yes"
	}
	return "No"
}

// formatMoney (CAP-F08) appends the resolved currency code to a money
// field's raw numeric value -- Options.Currency (fixed) or
// data[Options.CurrencyField] (CAP-F17's per-transaction currency).
func formatMoney(val string, f *model.Field, data map[string]any) string {
	currency := f.Options.Currency
	if f.Options.CurrencyField != "" {
		currency = fmt.Sprintf("%v", data[f.Options.CurrencyField])
	}
	if currency == "" || currency == "<nil>" {
		return val
	}
	return currency + " " + val
}

// computedValue (CAP-F14) resolves a `computed` field's display value at
// render time -- data[Options.SourceField] * Options.Factor -- never
// stored, matching CAP-V13's own "computed at render time" precedent. A
// non-numeric or missing source renders blank rather than "0", the same
// "don't fabricate a number for missing data" posture SumField already
// takes.
func computedValue(f *model.Field, data map[string]any) string {
	raw, ok := data[f.Options.SourceField]
	if !ok {
		return ""
	}
	n, err := strconv.ParseFloat(fmt.Sprintf("%v", raw), 64)
	if err != nil {
		return ""
	}
	multiplier := f.Options.Factor
	if f.Options.FactorField != "" {
		fv, ok := data[f.Options.FactorField]
		if !ok {
			return ""
		}
		multiplier, err = strconv.ParseFloat(fmt.Sprintf("%v", fv), 64)
		if err != nil {
			return ""
		}
	}
	return strconv.FormatFloat(n*multiplier, 'f', -1, 64)
}

func displayLabel(machine *model.Machine, data map[string]any) string {
	if machine != nil {
		var firstText *model.Field
		for _, f := range machine.Fields {
			if f.Type != model.FieldTypeText {
				continue
			}
			if firstText == nil {
				firstText = f
			}
			if strings.EqualFold(f.Name, "name") {
				firstText = f
				break
			}
		}
		if firstText != nil {
			if v, ok := data[firstText.ID]; ok {
				if s := fmt.Sprintf("%v", v); s != "" {
					return s
				}
			}
		}
	}
	if id, ok := data["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	return ""
}

// referenceViolations enforces CAP-F13 referential integrity: a `reference`
// field's value must resolve to a real record on the target Machine. This is
// intrinsic to the field type, the same way `required` is intrinsic to a
// Field's Required flag — not a declared Constraint row.
func (h *Handler) referenceViolations(ctx context.Context, machine *model.Machine, data map[string]any) ([]string, error) {
	var out []string
	for _, f := range machine.Fields {
		if f.Type != model.FieldTypeReference {
			continue
		}
		v, ok := data[f.ID]
		if !ok {
			continue
		}
		recordID, _ := v.(string)
		if recordID == "" {
			continue
		}
		exists, err := h.records.Exists(ctx, f.Options.TargetMachine, recordID)
		if err != nil {
			return nil, err
		}
		if !exists {
			target, _ := h.interp.Get().GetMachine(f.Options.TargetMachine)
			targetName := f.Options.TargetMachine
			if target != nil {
				targetName = target.Name
			}
			out = append(out, fmt.Sprintf("%s does not reference an existing %s record.", f.Name, targetName))
		}
	}
	return out, nil
}

// userFieldOptions (CAP-F05) is a `user` field's picker candidate pool --
// scoped to people who hold any role in the field's own Machine's
// Application (UserStore.ListForApplicationRole), the CAP-O01-derived
// query-time filter this capability's own design settled on rather than a
// new metadata concept (see benchmarks/007-user-role-management-survey.md).
func (h *Handler) userFieldOptions(ctx context.Context, applicationID string) []ui.ReferenceOption {
	users, err := h.users.ListForApplicationRole(ctx, applicationID)
	if err != nil {
		slog.Error("list user field options", "application", applicationID, "error", err)
		return nil
	}
	opts := make([]ui.ReferenceOption, 0, len(users))
	for _, u := range users {
		opts = append(opts, ui.ReferenceOption{ID: u.ID, Label: u.Name})
	}
	return opts
}

// userLabel (CAP-F05) resolves one user id's display name, for rendering an
// already-set `user` field value (detail/list views) -- the person-field
// counterpart to referenceLabel. Unlike a reference field, there's no
// profile page to link to in this prototype, so callers render plain text,
// never a link.
func (h *Handler) userLabel(ctx context.Context, userID string) (string, error) {
	u, err := h.users.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.Name, nil
}

// userReferenceViolations (CAP-F05) enforces referential integrity for
// `user`-typed fields, the exact same tier and shape as referenceViolations
// (CAP-F13) -- a required-field-style violation, not a 500, when a value
// doesn't resolve to a real account.
func (h *Handler) userReferenceViolations(ctx context.Context, machine *model.Machine, data map[string]any) ([]string, error) {
	var out []string
	for _, f := range machine.Fields {
		if f.Type != model.FieldTypeUser {
			continue
		}
		v, ok := data[f.ID]
		if !ok {
			continue
		}
		userID, _ := v.(string)
		if userID == "" {
			continue
		}
		exists, err := h.users.Exists(ctx, userID)
		if err != nil {
			return nil, err
		}
		if !exists {
			out = append(out, fmt.Sprintf("%s does not reference an existing user.", f.Name))
		}
	}
	return out, nil
}

// immutabilityViolation (CAP-R07) returns a non-empty rejection reason when
// machine.Config declares `immutable_field`/`immutable_values` (a
// comma-separated list, CAP-X03's generic Machine-level settings, not a new
// migration column) and data's current value for that field is one of
// them -- "record is frozen once Posted," stronger than CAP-E06 (which only
// guards Events): this guards direct field edits (Update) and archival
// (Archive) too, every mutation path, not just the workflow transitions
// CAP-E06 already covers.
func (h *Handler) immutabilityViolation(machine *model.Machine, data map[string]any) string {
	field := machine.Config["immutable_field"]
	if field == "" {
		return ""
	}
	cur := fmt.Sprintf("%v", data[field])
	for _, v := range strings.Split(machine.Config["immutable_values"], ",") {
		if strings.TrimSpace(v) == cur {
			return fmt.Sprintf("record is immutable while %s is %q", field, cur)
		}
	}
	return ""
}

// inScratchState (CAP-R08) reports whether data's current value for
// machine.Config's declared `scratch_field` is one of `scratch_values`
// (comma-separated) -- a record in this state (e.g. a Cart before
// Checkout) has none of its eventual business-rule Constraints enforced
// yet, the opposite end of CAP-R07's spectrum. Referential integrity
// (referenceViolations/userReferenceViolations/uniquenessViolations) is
// NOT exempted -- a scratch record can be incomplete, not corrupt. The
// commit point back into full enforcement needs no new mechanism: CAP-C09's
// existing trigger-time Violations re-check already applies the moment an
// event moves the record out of scratch_values.
func (h *Handler) inScratchState(machine *model.Machine, data map[string]any) bool {
	field := machine.Config["scratch_field"]
	if field == "" {
		return false
	}
	cur := fmt.Sprintf("%v", data[field])
	for _, v := range strings.Split(machine.Config["scratch_values"], ",") {
		if strings.TrimSpace(v) == cur {
			return true
		}
	}
	return false
}

// withChangePolicyCreatedAt (CAP-W07) exposes a record's creation time to
// constraint.Eval as an ordinary comparable field (model.ChangePolicyCreatedAtField)
// without ever persisting it. Always returns a COPY -- data is the exact map
// that gets written to the record's JSONB column right after the Violations
// check runs, so mutating it in place would leak the synthetic key into
// storage. Skips the copy entirely for the vast majority of Machines that
// declare no `new_records` change_policy (machine.NeedsCreatedAtGuard).
func withChangePolicyCreatedAt(machine *model.Machine, data map[string]any, t time.Time) map[string]any {
	if !machine.NeedsCreatedAtGuard {
		return data
	}
	out := make(map[string]any, len(data)+1)
	for k, v := range data {
		out[k] = v
	}
	out[model.ChangePolicyCreatedAtField] = t.Format("2006-01-02")
	return out
}

// uniquenessViolations (CAP-C12) enforces `unique` constraints -- single or
// composite/multi-field -- against every OTHER record on the same machine.
// Unlike engine.Violations, this needs the database (RecordStore), so it's a
// separate check at the same tier as referenceViolations/
// userReferenceViolations, not folded into constraint.Engine (which
// deliberately never touches storage). excludeRecordID is the record being
// updated -- empty on Create, where nothing to exclude exists yet.
func (h *Handler) uniquenessViolations(ctx context.Context, machine *model.Machine, data map[string]any, excludeRecordID string) ([]string, error) {
	var out []string
	for _, c := range machine.Constraints {
		if c.Expression.Operator != "unique" {
			continue
		}
		fields := c.Expression.Fields
		if len(fields) == 0 && c.Expression.Field != "" {
			fields = []string{c.Expression.Field}
		}
		if len(fields) == 0 {
			continue
		}
		fieldValues := make(map[string]string, len(fields))
		allSet := true
		for _, fid := range fields {
			v, ok := data[fid]
			if !ok {
				allSet = false
				break
			}
			fieldValues[fid] = fmt.Sprintf("%v", v)
		}
		if !allSet {
			// Nothing to collide with yet -- CAP-C01 `required` already
			// covers "this field must have a value" separately.
			continue
		}
		exists, err := h.records.ExistsWithFieldValues(ctx, machine.ID, fieldValues, excludeRecordID)
		if err != nil {
			return nil, err
		}
		if exists {
			out = append(out, c.Rule)
		}
	}
	return out, nil
}

// --- CAP-A07 / CAP-A08 workflow orchestration --------------------------------
//
// Both capabilities need the same handful of lookups on the child Machine
// (Approval Step, in Case 3's proof): which Field references the parent
// record, which Field holds the step's own Sequence, which Field holds its
// Decision. None of these are named in the action's own params — Menata
// Language doesn't have a way for a business author to name "the field that
// scopes this" beyond writing an ordinary `reference` Field — so, like
// displayLabel, this resolves them by name/type heuristic (a `reference`
// Field pointing at the parent Machine; a Field literally named "Sequence" or
// "Decision", case-insensitive). Prototype-honest, not a final design.

func findFieldByName(machine *model.Machine, name string) *model.Field {
	for _, f := range machine.Fields {
		if strings.EqualFold(f.Name, name) {
			return f
		}
	}
	return nil
}

func findFieldByID(machine *model.Machine, id string) *model.Field {
	for _, f := range machine.Fields {
		if f.ID == id {
			return f
		}
	}
	return nil
}

func findReferenceFieldTo(machine *model.Machine, targetMachineID string) *model.Field {
	for _, f := range machine.Fields {
		if f.Type == model.FieldTypeReference && f.Options.TargetMachine == targetMachineID {
			return f
		}
	}
	return nil
}

func toFloat(v any) float64 {
	s := fmt.Sprintf("%v", v)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// aggregateConditionViolation implements CAP-A14: an Event with an
// AggregateCondition may only be triggered once SUM(aggregate_field) across
// every record on Machine sharing this record's own value for scope_field
// satisfies operator/value -- "only once this Member's total Points reach
// 100." Returns ("", nil) when the event has no AggregateCondition at all
// (the overwhelmingly common case).
func (h *Handler) aggregateConditionViolation(ctx context.Context, event *model.Event, rec *store.Record) (string, error) {
	ac := event.AggregateCondition
	if ac == nil {
		return "", nil
	}
	scopeValue := fmt.Sprintf("%v", rec.Data[ac.ScopeField])
	sum, err := h.records.SumField(ctx, ac.Machine, ac.AggregateField, ac.ScopeField, scopeValue)
	if err != nil {
		return "", err
	}
	// constraint.Eval reads expr.Field out of a data map -- "sum" is a
	// synthetic single-key map standing in for the computed aggregate, the
	// same field/operator/value comparison every other Eval call already uses.
	expr := model.ConstraintExpression{Field: "sum", Operator: ac.Operator, Value: ac.Value}
	if !constraint.Eval(expr, map[string]any{"sum": sum}) {
		return fmt.Sprintf("%s is not allowed until the aggregate condition on %s is met", event.Name, ac.AggregateField), nil
	}
	return "", nil
}

// separationOfDutiesViolation (CAP-P03) blocks a decision when the acting
// identity is the same person who submitted the PARENT record this one
// belongs to -- "Requester != Approver," even when the actor happens to
// also hold the deciding role. Declared via Machine.Config (CAP-X03, no
// new migration column, same pattern CAP-R07/R08 already use):
//
//	sod_reference_field:  this Machine's own `reference` field pointing at
//	                      the parent (e.g. an Approval Step's fld_as_document)
//	sod_requester_field:  the `user`-typed field on the PARENT Machine
//	                      holding who submitted it
//
// Returns "" (no violation) when the Machine doesn't declare either key --
// separation of duties is opt-in, not a default every Machine pays for.
func (h *Handler) separationOfDutiesViolation(ctx context.Context, machine *model.Machine, rec *store.Record, actorIdentityID string) (string, error) {
	refField := machine.Config["sod_reference_field"]
	requesterField := machine.Config["sod_requester_field"]
	if refField == "" || requesterField == "" {
		return "", nil
	}
	parentID, _ := rec.Data[refField].(string)
	if parentID == "" {
		return "", nil
	}
	parent, err := h.records.Get(ctx, parentID)
	if err != nil {
		return "", nil
	}
	if fmt.Sprintf("%v", parent.Data[requesterField]) == actorIdentityID {
		return "you submitted this record and cannot also decide it (separation of duties)", nil
	}
	return "", nil
}

// sequentialGuardViolation implements CAP-A07's hard block: in Sequential
// mode, a step may only be Approved or Rejected once every sibling with a
// lower Sequence has already left Pending. Applies to any event carrying an
// aggregate_status action — both evt_as_approve and evt_as_reject declare
// one, so both are gated uniformly, even though only Approve also declares
// activate_next. Returns "" when the event isn't part of a sequential
// workflow at all (no aggregate_status action, no config, or Parallel mode).
func (h *Handler) sequentialGuardViolation(ctx context.Context, machine *model.Machine, event *model.Event, rec *store.Record) string {
	var parentFieldID string
	for _, a := range event.Actions {
		if a.Type == model.ActionAggregateStatus {
			parentFieldID, _ = a.Params["parent_field"].(string)
			break
		}
	}
	if parentFieldID == "" {
		return ""
	}
	parentField := findFieldByID(machine, parentFieldID)
	if parentField == nil || parentField.Type != model.FieldTypeReference {
		return ""
	}
	parentMachine, ok := h.interp.Get().GetMachine(parentField.Options.TargetMachine)
	if !ok || parentMachine.Config == nil {
		return ""
	}
	modeFieldID := parentMachine.Config["approval_mode_field"]
	if modeFieldID == "" {
		return ""
	}
	parentID, _ := rec.Data[parentFieldID].(string)
	if parentID == "" {
		return ""
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil {
		return ""
	}
	if fmt.Sprintf("%v", parentRec.Data[modeFieldID]) != "Sequential" {
		return ""
	}

	seqField := findFieldByName(machine, "Sequence")
	decisionField := findFieldByName(machine, "Decision")
	if seqField == nil || decisionField == nil {
		return ""
	}
	mySeq := toFloat(rec.Data[seqField.ID])

	siblings, err := h.records.List(ctx, machine.ID, "", "")
	if err != nil {
		return ""
	}
	for _, sib := range siblings {
		if sib.ID == rec.ID {
			continue
		}
		if sp, _ := sib.Data[parentFieldID].(string); sp != parentID {
			continue
		}
		if fmt.Sprintf("%v", sib.Data[decisionField.ID]) != "Pending" {
			continue
		}
		if toFloat(sib.Data[seqField.ID]) < mySeq {
			return fmt.Sprintf("%s is not allowed yet — an earlier step is still Pending", event.Name)
		}
	}
	return ""
}

// runWorkflowActions handles the two action types Executor.Persist doesn't
// touch (CAP-A07/A08 need cross-record lookups Executor has no access to —
// this stays at the Handler layer, same reasoning as childLists/
// referenceViolations).
func (h *Handler) runWorkflowActions(ctx context.Context, machine *model.Machine, event *model.Event, recordID string, data map[string]any) {
	for _, a := range event.Actions {
		switch a.Type {
		case model.ActionActivateNext:
			h.doActivateNext(ctx, machine, a.Params, recordID, data)
		case model.ActionAggregateStatus:
			h.doAggregateStatus(ctx, machine, a.Params, data)
		case model.ActionTriggerEvent:
			h.doTriggerEvent(ctx, machine, a.Params, recordID, data)
		}
	}
}

// doActivateNext (CAP-A07): in Sequential mode, notifies the next still-
// Pending sibling's Approver that it's their turn. Doesn't write anything —
// the sequential guard above is what actually enforces order; this is purely
// the hand-off notification (logged only, same as every other `notify` in
// this prototype).
func (h *Handler) doActivateNext(ctx context.Context, machine *model.Machine, params map[string]any, recordID string, data map[string]any) {
	modeFieldID, _ := params["mode_field"].(string)
	if modeFieldID == "" {
		return
	}
	parentMachine := findMachineContainingField(h.interp.Get(), modeFieldID)
	if parentMachine == nil || parentMachine.Config == nil {
		return
	}
	parentRefField := findReferenceFieldTo(machine, parentMachine.ID)
	if parentRefField == nil {
		return
	}
	parentID, _ := data[parentRefField.ID].(string)
	if parentID == "" {
		return
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil || fmt.Sprintf("%v", parentRec.Data[modeFieldID]) != "Sequential" {
		return
	}

	seqField := findFieldByName(machine, "Sequence")
	decisionField := findFieldByName(machine, "Decision")
	approverField := findFieldByName(machine, "Approver")
	if seqField == nil || decisionField == nil {
		return
	}
	mySeq := toFloat(data[seqField.ID])

	siblings, err := h.records.List(ctx, machine.ID, "", "")
	if err != nil {
		return
	}
	var next *store.Record
	var nextSeq float64
	for _, sib := range siblings {
		if sib.ID == recordID {
			continue
		}
		if sp, _ := sib.Data[parentRefField.ID].(string); sp != parentID {
			continue
		}
		if fmt.Sprintf("%v", sib.Data[decisionField.ID]) != "Pending" {
			continue
		}
		s := toFloat(sib.Data[seqField.ID])
		if s > mySeq && (next == nil || s < nextSeq) {
			next, nextSeq = sib, s
		}
	}
	if next == nil {
		return
	}
	approver := ""
	if approverField != nil {
		approver = fmt.Sprintf("%v", next.Data[approverField.ID])
	}
	slog.Info("activate_next: next step ready (prototype: notify logged only)",
		"machine", machine.ID, "next_record", next.ID, "sequence", nextSeq, "approver", approver)
}

// doTriggerEvent (CAP-E05): fires another event on the SAME record — a
// narrower, same-record counterpart to doAggregateStatus's cross-record
// rollup. No extra DB fetch needed (unlike doAggregateStatus, which reaches
// into a different record): `data` is already this event's just-persisted
// result, and recordID is this same record's own ID. Runs through the same
// triggerEvent path an HTTP request would use (CAP-E06/CAP-C09 guards still
// apply), as "System" for both role and identity — same precedent as
// doAggregateStatus, and the same reason guard.CanTrigger is bypassed
// entirely here rather than called: a system-triggered transition isn't a
// business role acting, so there's no role/identity to check permission
// against.
func (h *Handler) doTriggerEvent(ctx context.Context, machine *model.Machine, params map[string]any, recordID string, data map[string]any) {
	targetEventID, _ := params["event"].(string)
	if targetEventID == "" {
		return
	}
	targetEvent, ok := h.interp.Get().GetEvent(machine.ID, targetEventID)
	if !ok {
		return
	}
	rec := &store.Record{ID: recordID, Data: data}
	if err := h.triggerEvent(ctx, machine, targetEvent, rec, "System", "System", "", nil); err != nil {
		slog.Error("trigger_event: failed to trigger chained event",
			"machine", machine.ID, "record", recordID, "event", targetEventID, "error", err)
	}
}

// doAggregateStatus (CAP-A08): rolls sibling steps' Decision up to the parent
// — any Rejected cascades the parent to its "any rejected" event immediately
// (doesn't wait for the rest to decide); only once every sibling is Approved
// does the parent reach its "all approved" event. Internally triggers the
// resolved parent event through the same triggerEvent path an HTTP request
// would use (CAP-E06/CAP-C09 guards still apply), as "System" — the acting
// role recorded for this rollup's dynamic values, matching approval-
// document.yaml's own declared System permission role.
//
// min_approvals (CAP-W03, Process Overlay B4 Part 2) is an optional
// quorum-of-N parameter, backward-compatible by construction — omitted or
// zero falls through to the ALL-required behavior above, unchanged. Set to
// N over M total siblings: reaches parent_event_if_all_approved as soon as
// count(Approved) >= N, without waiting for every sibling to decide (real
// N-of-M semantics — a still-Pending sibling no longer blocks quorum once
// enough others have approved); reaches parent_event_if_any_rejected only
// once quorum becomes mathematically impossible
// (count(Rejected) > total - N, i.e. too few non-rejected siblings remain
// to ever reach N) — a minority of rejections that still leaves enough
// headroom does NOT cancel early, unlike the ALL-required rule above.
func (h *Handler) doAggregateStatus(ctx context.Context, machine *model.Machine, params map[string]any, data map[string]any) {
	parentFieldID, _ := params["parent_field"].(string)
	allApprovedEvt, _ := params["parent_event_if_all_approved"].(string)
	anyRejectedEvt, _ := params["parent_event_if_any_rejected"].(string)
	minApprovals, _ := params["min_approvals"].(float64) // JSON numbers decode as float64
	if parentFieldID == "" {
		return
	}
	parentField := findFieldByID(machine, parentFieldID)
	if parentField == nil || parentField.Type != model.FieldTypeReference {
		return
	}
	parentMachine, ok := h.interp.Get().GetMachine(parentField.Options.TargetMachine)
	if !ok {
		return
	}
	parentID, _ := data[parentFieldID].(string)
	if parentID == "" {
		return
	}

	decisionField := findFieldByName(machine, "Decision")
	if decisionField == nil {
		return
	}
	siblings, err := h.records.List(ctx, machine.ID, "", "")
	if err != nil {
		return
	}

	found, allApproved, anyRejected := false, true, false
	total, approved, rejected := 0, 0, 0
	for _, sib := range siblings {
		if sp, _ := sib.Data[parentFieldID].(string); sp != parentID {
			continue
		}
		found = true
		total++
		switch fmt.Sprintf("%v", sib.Data[decisionField.ID]) {
		case "Rejected":
			anyRejected = true
			rejected++
		case "Approved":
			approved++
			// stays eligible for allApproved
		default:
			allApproved = false
		}
	}
	if !found {
		return
	}

	var targetEventID string
	switch {
	case minApprovals > 0:
		switch {
		case approved >= int(minApprovals) && allApprovedEvt != "":
			targetEventID = allApprovedEvt
		case rejected > total-int(minApprovals) && anyRejectedEvt != "":
			targetEventID = anyRejectedEvt
		default:
			return
		}
	case anyRejected && anyRejectedEvt != "":
		targetEventID = anyRejectedEvt
	case allApproved && allApprovedEvt != "":
		targetEventID = allApprovedEvt
	default:
		return
	}

	targetEvent, ok := h.interp.Get().GetEvent(parentMachine.ID, targetEventID)
	if !ok {
		return
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil {
		return
	}
	if err := h.triggerEvent(ctx, parentMachine, targetEvent, parentRec, "System", "System", "", nil); err != nil {
		slog.Error("aggregate_status: failed to trigger parent event",
			"parent_machine", parentMachine.ID, "parent_record", parentID, "event", targetEventID, "error", err)
	}
}

func findMachineContainingField(interp *interpreter.Interpreter, fieldID string) *model.Machine {
	for _, m := range interp.AllMachines() {
		if findFieldByID(m, fieldID) != nil {
			return m
		}
	}
	return nil
}

// --- CAP-A10 in-app notification inbox ---------------------------------------

// unreadCount takes the full Auth (not just role) because CAP-O01's
// recipientMatch needs both the identity name and the user id (to check
// per-Application role assignments) — see store.NotificationStore's
// recipientMatch doc comment.
func (h *Handler) unreadCount(ctx context.Context, a *store.Auth) int {
	n, err := h.notifications.UnreadCount(ctx, a.User.Name, a.User.ID)
	if err != nil {
		slog.Error("count unread notifications", "error", err)
		return 0
	}
	return n
}

// Notifications — the current session's in-app notification inbox: every
// `notify` action (CAP-A03/A04) whose resolved recipient matches this
// person's identity OR one of their per-Application roles (CAP-O01).
// Search (CAP-O04) is workspace-wide: every Machine the searching role can
// read (CAP-P05, permission-trimmed -- a Machine the role has no access to
// is never even scanned, not filtered out after the fact), same
// case-insensitive substring match CAP-V08's own per-Machine `?q=` already
// uses, against that Machine's own DEFAULT list View columns.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	a := h.auth(r)
	workspaceID := h.workspace(r)

	var results []ui.SearchResult
	if query != "" {
		q := strings.ToLower(query)
		for _, m := range h.interp.Get().AllMachines() {
			ws, appID := h.interp.Get().ScopeFor(m.ID)
			if ws != workspaceID {
				continue
			}
			role := h.roleForApp(r, appID)
			if !h.guard.CanRead(m, role) {
				continue
			}
			view := h.interp.Get().DefaultListView(m.ID)
			var colIDs []string
			if view != nil {
				colIDs = view.Config.Columns
			}
			if len(colIDs) == 0 {
				continue
			}
			records, err := h.records.List(r.Context(), m.ID, "", "")
			if err != nil {
				slog.Error("workspace search: list records", "machine", m.ID, "error", err)
				continue
			}
			for _, rec := range records {
				matched := false
				for _, id := range colIDs {
					if v, ok := rec.Data[id]; ok && strings.Contains(strings.ToLower(fmt.Sprintf("%v", v)), q) {
						matched = true
						break
					}
				}
				if matched {
					results = append(results, ui.SearchResult{
						MachineName: m.Name,
						Label:       displayLabel(m, rec.Data),
						Link:        "/" + m.ID + "/" + rec.ID,
					})
				}
			}
		}
	}

	page := ui.Search(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), query, results, h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render search", "error", err)
	}
}

func (h *Handler) Notifications(w http.ResponseWriter, r *http.Request) {
	a := h.auth(r)
	notifs, err := h.notifications.ListForRecipient(r.Context(), a.User.Name, a.User.ID)
	if err != nil {
		http.Error(w, "failed to load notifications", http.StatusInternalServerError)
		return
	}
	items := make([]ui.NotificationItem, len(notifs))
	for i, n := range notifs {
		link := ""
		if n.MachineID != "" && n.RecordID != "" {
			link = "/" + n.MachineID + "/" + n.RecordID
		}
		items[i] = ui.NotificationItem{
			ID:      n.ID,
			Message: n.Message,
			Link:    link,
			Unread:  n.ReadAt == nil,
			When:    n.CreatedAt.Format("2006-01-02 15:04"),
			Date:    n.CreatedAt.Format("2006-01-02"),
		}
	}

	// CAP-O05: "digest" groups the SAME items by day instead of listing
	// them flat -- built here (Go), not in the template, matching this
	// codebase's own "handler builds view-ready structures" convention.
	var groups []ui.NotificationGroup
	if a.User.NotificationPreference == "digest" {
		var cur string
		var curItems []ui.NotificationItem
		first := true
		for _, item := range items {
			if first || item.Date != cur {
				if curItems != nil {
					groups = append(groups, ui.NotificationGroup{Date: cur, Items: curItems})
				}
				cur, curItems, first = item.Date, nil, false
			}
			curItems = append(curItems, item)
		}
		if curItems != nil {
			groups = append(groups, ui.NotificationGroup{Date: cur, Items: curItems})
		}
	}

	page := ui.Notifications(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), items, groups, a.User.NotificationPreference, h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render notifications", "error", err)
	}
}

// SetNotificationPreference (CAP-O05) toggles the current user's own inbox
// grouping preference between "immediate" and "digest".
func (h *Handler) SetNotificationPreference(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	preference := r.FormValue("preference")
	if preference != "immediate" && preference != "digest" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	a := h.auth(r)
	if err := h.users.SetNotificationPreference(r.Context(), a.User.ID, preference); err != nil {
		http.Error(w, "failed to update preference", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}

// MarkNotificationRead — POST target for a single notification's "Mark read"
// button. Scoped to the current session in the store layer (recipientMatch),
// the same access-control shape as every other identity/role-gated action.
func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a := h.auth(r)
	if err := h.notifications.MarkRead(r.Context(), id, a.User.Name, a.User.ID); err != nil {
		http.Error(w, "failed to mark notification read", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}
