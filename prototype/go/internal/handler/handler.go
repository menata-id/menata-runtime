package handler

import (
	"context"
	"errors"
	"fmt"
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
	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/permission"
	"menata.id/runtime/internal/store"
	"menata.id/runtime/internal/ui"
)

type Handler struct {
	interp        *interpreter.Interpreter
	records       *store.RecordStore
	notifications *store.NotificationStore
	sessions      *store.SessionStore
	users         *store.UserStore
	secureCookies bool
	engine        *constraint.Engine
	guard         *permission.Guard
	exec          *executor.Executor
}

func New(interp *interpreter.Interpreter, records *store.RecordStore, notifications *store.NotificationStore, sessions *store.SessionStore, users *store.UserStore, secureCookies bool) *Handler {
	return &Handler{
		interp:        interp,
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
	apps := h.interp.ApplicationsForWorkspace(a.User.WorkspaceID)
	cards := make([]ui.Card, 0, len(apps))
	for _, app := range apps {
		// CAP-O01: role is resolved per-Application here, not once for the
		// whole page — the same person can see a different set of readable
		// Applications depending on which role (if any) they hold in each.
		role := a.ApplicationRoles[app.ID]
		machines := h.interp.MachinesForApplication(app.ID)
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
	app, ok := h.interp.GetApplication(appID)
	if !ok || app.WorkspaceID != h.workspace(r) {
		// CAP-X06: an Application from another Workspace 404s exactly like
		// one that doesn't exist at all -- not a 403, which would confirm
		// to a prober that the ID is real, just in the wrong workspace.
		http.NotFound(w, r)
		return
	}
	a := h.auth(r)
	role := h.roleForApp(r, appID)
	machines := h.interp.MachinesForApplication(appID)
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

	for _, g := range h.interp.AllRoles(a.User.WorkspaceID) {
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

func (h *Handler) uiRoleGroups(workspaceID string) []ui.RoleGroup {
	interpGroups := h.interp.AllRoles(workspaceID)
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
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.ScopeFor(machineID)
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

	view := h.interp.DefaultListView(machineID)
	fieldByID := fieldIndex(machine)

	colIDs := []string{}
	if view != nil {
		colIDs = view.Config.Columns
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

	records, err := h.records.List(r.Context(), machineID)
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	rows := make([]ui.ListRow, 0, len(records))
	for _, rec := range records {
		cells := make([]ui.ListCell, len(colIDs))
		for j, id := range colIDs {
			val := ""
			if v, ok := rec.Data[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
			link := ""
			if cols[j].Type == model.FieldTypeReference && val != "" {
				refID := val
				target := fieldByID[id].Options.TargetMachine
				if label, err := h.referenceLabel(r.Context(), target, refID); err == nil && label != "" {
					val = label
					link = "/" + target + "/" + refID
				}
			} else if cols[j].Type == model.FieldTypeUser && val != "" {
				if label, err := h.userLabel(r.Context(), val); err == nil && label != "" {
					val = label
				}
			}
			cells[j] = ui.ListCell{
				Value:         val,
				IsStatusBadge: cols[j].Type == model.FieldTypeValueList,
				Link:          link,
			}
		}
		rows = append(rows, ui.ListRow{ID: rec.ID, Cells: cells})
	}

	a := h.auth(r)
	page := ui.List(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, cols, rows, h.interp.PermittedEvents(machineID, role), h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render list", "error", err)
	}
}

// NewForm — form for creating a new record.
func (h *Handler) NewForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.ScopeFor(machineID)
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
	page := ui.Form(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, "", h.buildFormFields(r.Context(), machine, nil), nil, h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render form", "error", err)
	}
}

// Create — handle new record form submission.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.ScopeFor(machineID)
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

	// Any value_list field the Create form doesn't expose (Status, Decision, ...)
	// starts at its first declared value — the same "first value = initial
	// state" convention guides/writing-menata.md teaches .menata authors,
	// generalized from what was previously a "status"-named-field-only rule.
	formFieldIDs := map[string]bool{}
	if fv := h.interp.FormView(machine.ID); fv != nil {
		for _, id := range fv.Config.Fields {
			formFieldIDs[id] = true
		}
	}
	data := make(map[string]any)
	for _, f := range machine.Fields {
		if !formFieldIDs[f.ID] && f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
			data[f.ID] = f.Options.Values[0]
		}
	}
	for _, f := range machine.Fields {
		if v := r.FormValue(f.ID); v != "" {
			data[f.ID] = v
		}
	}

	violations := h.engine.Violations(machine, data)
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

	if len(violations) > 0 {
		role := h.roleForApp(r, applicationID)
		h.logRuleViolation(r.Context(), "create", machineID, "", role, h.identity(r), strings.Join(violations, "; "))
		a := h.auth(r)
		page := ui.Form(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, "", h.buildFormFields(r.Context(), machine, data), violations, h.unreadCount(r.Context(), a))
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
	http.Redirect(w, r, "/"+machineID+"/"+rec.ID, http.StatusSeeOther)
}

// EditForm — form for editing an existing record (CAP-R02). Reuses the same
// FormView/field set Create uses — Menata Language has no separate "edit
// form" view declared in metadata — pre-filled with the record's current
// data.
func (h *Handler) EditForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.ScopeFor(machineID)
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
	a := h.auth(r)
	page := ui.Form(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, recordID, h.buildFormFields(r.Context(), machine, rec.Data), nil, h.unreadCount(r.Context(), a))
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
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.ScopeFor(machineID)
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	formFieldIDs := map[string]bool{}
	if fv := h.interp.FormView(machine.ID); fv != nil {
		for _, id := range fv.Config.Fields {
			formFieldIDs[id] = true
		}
	}
	data := make(map[string]any, len(rec.Data))
	for k, v := range rec.Data {
		data[k] = v
	}
	for _, f := range machine.Fields {
		if formFieldIDs[f.ID] {
			data[f.ID] = r.FormValue(f.ID)
		}
	}

	violations := h.engine.Violations(machine, data)
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
		page := ui.Form(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, recordID, h.buildFormFields(r.Context(), machine, data), violations, h.unreadCount(r.Context(), a))
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

	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	if role := h.roleForApp(r, applicationID); !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	fields := make([]ui.DetailField, 0, len(machine.Fields))
	for _, f := range machine.Fields {
		val := ""
		if v, ok := rec.Data[f.ID]; ok {
			val = fmt.Sprintf("%v", v)
		}
		link := ""
		if f.Type == model.FieldTypeReference && val != "" {
			refID := val
			target := f.Options.TargetMachine
			if label, err := h.referenceLabel(r.Context(), target, refID); err == nil && label != "" {
				val = label
				link = "/" + target + "/" + refID
			}
		} else if f.Type == model.FieldTypeUser && val != "" {
			if label, err := h.userLabel(r.Context(), val); err == nil && label != "" {
				val = label
			}
		}
		fields = append(fields, ui.DetailField{Name: f.Name, Value: val, Link: link})
	}

	role := h.roleForApp(r, applicationID)
	childLists := h.childLists(r.Context(), machine, recordID)
	permittedEvents := h.interp.PermittedEventsForRecord(machineID, role, h.identityID(r), rec.Data)
	a := h.auth(r)
	page := ui.Detail(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, rec, fields, permittedEvents, childLists, h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render detail", "error", err)
	}
}

// TriggerEvent — handle event button.
func (h *Handler) TriggerEvent(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	eventID := chi.URLParam(r, "eventID")

	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	event, ok := h.interp.GetEvent(machineID, eventID)
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

	if err := h.triggerEvent(r.Context(), machine, event, rec, role, identity); err != nil {
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
	workspaceID, appID := h.interp.ScopeFor(machineID)
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
	workspaceID, appID := h.interp.ScopeFor(machineID)
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
func (h *Handler) triggerEvent(ctx context.Context, machine *model.Machine, event *model.Event, rec *store.Record, actorRole, actorIdentity string) error {
	// CAP-E06 — state guard: the event may only fire when the record's
	// CURRENT data satisfies its condition (e.g. Reject only from Submitted).
	if event.Condition != nil && !constraint.Eval(*event.Condition, rec.Data) {
		return &ruleViolation{fmt.Sprintf("%s is not allowed in the record's current state", event.Name)}
	}

	// CAP-A07 — sequential step guard: in a Sequential-mode parent, a step may
	// only be decided once every sibling with a lower Sequence has already
	// left Pending. Cross-record, so it can't be expressed as an event
	// Condition (CAP-E06 only reads the record's own data).
	if msg := h.sequentialGuardViolation(ctx, machine, event, rec); msg != "" {
		return &ruleViolation{msg}
	}

	// CAP-C09 — constraints evaluated on event trigger, not just Create:
	// simulate the event's effect first, validate the result, only persist
	// if it still satisfies every declared Constraint. CAP-A02 — actorIdentity
	// (falling back to actorRole) resolves this event's "current_user" dynamic
	// values, if any.
	newData := h.exec.Simulate(event, rec, actorRole, actorIdentity)
	if violations := h.engine.Violations(machine, newData); len(violations) > 0 {
		return &ruleViolation{strings.Join(violations, " ")}
	}

	workspaceID, _ := h.interp.ScopeFor(machine.ID)
	if err := h.exec.Persist(ctx, event, rec, newData, machine.Name, actorRole, actorIdentity, workspaceID); err != nil {
		return err
	}

	// CAP-A07/CAP-A08/CAP-E05 — workflow actions run after a successful
	// commit, using the now-current data: activate_next notifies the next
	// pending sibling, aggregate_status may internally trigger a rollup event
	// on the parent, trigger_event may fire another event on this same record.
	h.runWorkflowActions(ctx, machine, event, rec.ID, newData)
	return nil
}

// --- helpers -----------------------------------------------------------------

func fieldIndex(m *model.Machine) map[string]*model.Field {
	out := make(map[string]*model.Field, len(m.Fields))
	for _, f := range m.Fields {
		out[f.ID] = f
	}
	return out
}

func (h *Handler) buildFormFields(ctx context.Context, machine *model.Machine, vals map[string]any) []ui.FormField {
	view := h.interp.FormView(machine.ID)
	fieldByID := fieldIndex(machine)

	var fieldIDs []string
	if view != nil {
		fieldIDs = view.Config.Fields
	}

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
		fields = append(fields, ui.FormField{Field: f, Value: val, Options: opts})
	}
	return fields
}

// childLists finds every Machine with a `reference` field pointing at
// machine, and lists the records where that field equals recordID (CAP-V06).
// Generic by construction — it doesn't special-case Employee/Manager, so any
// future reference relationship gets a sub-list automatically.
func (h *Handler) childLists(ctx context.Context, machine *model.Machine, recordID string) []ui.ChildList {
	var out []ui.ChildList
	for _, m := range h.interp.AllMachines() {
		for _, f := range m.Fields {
			if f.Type != model.FieldTypeReference || f.Options.TargetMachine != machine.ID {
				continue
			}
			records, err := h.records.List(ctx, m.ID)
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
	records, err := h.records.List(ctx, targetMachineID)
	if err != nil {
		slog.Error("list reference options", "target_machine", targetMachineID, "error", err)
		return nil
	}
	targetMachine, _ := h.interp.GetMachine(targetMachineID)
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
	targetMachine, _ := h.interp.GetMachine(targetMachineID)
	return displayLabel(targetMachine, rec.Data), nil
}

// displayLabel picks a human-readable stand-in for a record: the target
// Machine's `text` field named "Name" if one exists, else its first `text`
// field, falling back to the record id. Menata Language doesn't (yet) let a
// business author declare "this is the field people should see when
// referencing a record" — this heuristic is a prototype stand-in for that
// missing capability, not a final design.
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
			target, _ := h.interp.GetMachine(f.Options.TargetMachine)
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
	parentMachine, ok := h.interp.GetMachine(parentField.Options.TargetMachine)
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

	siblings, err := h.records.List(ctx, machine.ID)
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
	parentMachine := findMachineContainingField(h.interp, modeFieldID)
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

	siblings, err := h.records.List(ctx, machine.ID)
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
	targetEvent, ok := h.interp.GetEvent(machine.ID, targetEventID)
	if !ok {
		return
	}
	rec := &store.Record{ID: recordID, Data: data}
	if err := h.triggerEvent(ctx, machine, targetEvent, rec, "System", "System"); err != nil {
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
func (h *Handler) doAggregateStatus(ctx context.Context, machine *model.Machine, params map[string]any, data map[string]any) {
	parentFieldID, _ := params["parent_field"].(string)
	allApprovedEvt, _ := params["parent_event_if_all_approved"].(string)
	anyRejectedEvt, _ := params["parent_event_if_any_rejected"].(string)
	if parentFieldID == "" {
		return
	}
	parentField := findFieldByID(machine, parentFieldID)
	if parentField == nil || parentField.Type != model.FieldTypeReference {
		return
	}
	parentMachine, ok := h.interp.GetMachine(parentField.Options.TargetMachine)
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
	siblings, err := h.records.List(ctx, machine.ID)
	if err != nil {
		return
	}

	found, allApproved, anyRejected := false, true, false
	for _, sib := range siblings {
		if sp, _ := sib.Data[parentFieldID].(string); sp != parentID {
			continue
		}
		found = true
		switch fmt.Sprintf("%v", sib.Data[decisionField.ID]) {
		case "Rejected":
			anyRejected = true
		case "Approved":
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
	case anyRejected && anyRejectedEvt != "":
		targetEventID = anyRejectedEvt
	case allApproved && allApprovedEvt != "":
		targetEventID = allApprovedEvt
	default:
		return
	}

	targetEvent, ok := h.interp.GetEvent(parentMachine.ID, targetEventID)
	if !ok {
		return
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil {
		return
	}
	if err := h.triggerEvent(ctx, parentMachine, targetEvent, parentRec, "System", "System"); err != nil {
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
		}
	}
	page := ui.Notifications(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), items, h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render notifications", "error", err)
	}
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
