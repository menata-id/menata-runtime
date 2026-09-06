package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"menata.id/app/internal/interpreter"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// AdminUsers (CAP-O01) — GET /admin/users: the workspace Admin's user
// management page. Admin-only, gated on the workspace-wide tier, not any
// Application role (a workspace could have zero Applications and an Admin
// would still need this page reachable).
func (h *Handler) AdminUsers(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_view", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
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
	groupRows, err := h.adminGroupRows(r.Context(), a.User.WorkspaceID)
	if err != nil {
		http.Error(w, "failed to load groups", http.StatusInternalServerError)
		return
	}
	page := ui.AdminUsers(h.workspaceName(r), a.User.Name, a.CSRFToken, h.unreadCount(r.Context(), a), rows, appGroups, groupRows)
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
		h.logPermissionDenied(r.Context(), "admin_update", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
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
		h.logPermissionDenied(r.Context(), "reload", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	workspaces, err := h.loader.LoadAll(r.Context())
	if err != nil {
		slog.Error("metadata reload failed", "correlation_id", middleware.GetReqID(r.Context()), "actor", a.User.Name, "error", err)
		// Full error (could be a DB/loader internal detail) already
		// captured above with a correlation_id -- an admin who needs it
		// greps the log, the HTTP response doesn't need to repeat it.
		http.Error(w, "reload failed", http.StatusInternalServerError)
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

// adminGroupRows (CAP-O07) builds the /admin/users page's Groups section --
// same shape as adminUserRows, one Group instead of one user.
func (h *Handler) adminGroupRows(ctx context.Context, workspaceID string) ([]ui.AdminGroupRow, error) {
	groups, err := h.groups.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	rows := make([]ui.AdminGroupRow, 0, len(groups))
	for _, g := range groups {
		memberNames, err := h.groups.MemberNames(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		appRoles, err := h.groups.ApplicationRoles(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, ui.AdminGroupRow{
			ID:          g.ID,
			Name:        g.Name,
			MemberNames: memberNames,
			AppRoles:    appRoles,
		})
	}
	return rows, nil
}

// AdminCreateGroup (CAP-O07) — POST /admin/groups: creates an empty Group
// (no members, no Application roles yet — set on its own edit page).
func (h *Handler) AdminCreateGroup(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_create_group", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if _, err := h.groups.Create(r.Context(), a.User.WorkspaceID, name); err != nil {
		if errors.Is(err, store.ErrDuplicateName) {
			// CAP-F23: migrations/024's uniqueness constraint (needed for
			// a reliable name-based lookup) can now reject a name that
			// used to always succeed -- a real, user-actionable 400, not
			// an unexplained 500.
			http.Error(w, store.ErrDuplicateName.Error(), http.StatusBadRequest) // errleak:allow: known sentinel just matched via errors.Is, not a raw internal error
			return
		}
		slog.Error("create group", "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// AdminGroupDetail (CAP-O07) — GET /admin/groups/{groupID}: one Group's own
// membership + per-Application role edit page.
func (h *Handler) AdminGroupDetail(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_view_group", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	groupID := chi.URLParam(r, "groupID")
	group, err := h.groups.GetByID(r.Context(), groupID)
	if err != nil || group.WorkspaceID != a.User.WorkspaceID {
		// CAP-X06: a Group from another Workspace 404s exactly like one that
		// doesn't exist at all -- same convention as every other
		// cross-workspace lookup in this handler.
		http.NotFound(w, r)
		return
	}
	users, err := h.users.ListByWorkspace(r.Context(), a.User.WorkspaceID)
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	memberIDs, err := h.groups.MemberIDs(r.Context(), groupID)
	if err != nil {
		http.Error(w, "failed to load members", http.StatusInternalServerError)
		return
	}
	members := make([]ui.GroupMemberOption, 0, len(users))
	for _, u := range users {
		members = append(members, ui.GroupMemberOption{ID: u.ID, Name: u.Name, Email: u.Email, Checked: memberIDs[u.ID]})
	}
	appRoles, err := h.groups.ApplicationRoles(r.Context(), groupID)
	if err != nil {
		http.Error(w, "failed to load roles", http.StatusInternalServerError)
		return
	}
	data := ui.GroupDetailData{ID: group.ID, Name: group.Name, Members: members, AppRoles: appRoles}
	appGroups := h.uiRoleGroups(a.User.WorkspaceID)
	page := ui.GroupDetail(h.workspaceName(r), a.User.Name, a.CSRFToken, h.unreadCount(r.Context(), a), data, appGroups)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render group detail", "error", err)
	}
}

// AdminSetGroupMembers (CAP-O07) — POST /admin/groups/{groupID}/members:
// replaces the Group's entire membership with exactly the checked set the
// form submitted (GroupStore.SetMembers' own "full value, not a delta"
// convention).
func (h *Handler) AdminSetGroupMembers(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_update_group", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	groupID := chi.URLParam(r, "groupID")
	group, err := h.groups.GetByID(r.Context(), groupID)
	if err != nil || group.WorkspaceID != a.User.WorkspaceID {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.groups.SetMembers(r.Context(), groupID, r.Form["member_id"]); err != nil {
		slog.Error("set group members", "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	slog.Info("admin updated group members",
		"correlation_id", middleware.GetReqID(r.Context()),
		"actor", a.User.Name,
		"group", group.Name,
	)
	http.Redirect(w, r, "/admin/groups/"+groupID, http.StatusSeeOther)
}

// AdminSetGroupRoles (CAP-O07) — POST /admin/groups/{groupID}/roles: saves
// the Group's per-Application role assignments, same submitted-value
// validation AdminUpdateUser's own loop already applies (deny anything not
// actually declared by that Application's own Machines' Permissions).
func (h *Handler) AdminSetGroupRoles(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_update_group", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	groupID := chi.URLParam(r, "groupID")
	group, err := h.groups.GetByID(r.Context(), groupID)
	if err != nil || group.WorkspaceID != a.User.WorkspaceID {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	for _, g := range h.interp.Get().AllRoles(a.User.WorkspaceID) {
		submitted := r.FormValue("app_role_" + g.AppID)
		if submitted == "" {
			if err := h.groups.RemoveApplicationRole(r.Context(), groupID, g.AppID); err != nil {
				slog.Error("remove group application role", "error", err)
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
		if err := h.groups.SetApplicationRole(r.Context(), groupID, g.AppID, submitted); err != nil {
			slog.Error("set group application role", "error", err)
			http.Error(w, "failed to save", http.StatusInternalServerError)
			return
		}
	}
	slog.Info("admin updated group roles",
		"correlation_id", middleware.GetReqID(r.Context()),
		"actor", a.User.Name,
		"group", group.Name,
	)
	http.Redirect(w, r, "/admin/groups/"+groupID, http.StatusSeeOther)
}
