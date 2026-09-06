package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"menata.id/app/internal/constraint"
	"menata.id/app/internal/executor"
	"menata.id/app/internal/interpreter"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/model"
	"menata.id/app/internal/permission"
	"menata.id/app/internal/storage"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

type Handler struct {
	interp        *interpreter.Store // CAP-X04: atomic, swapped by Reload -- call .Get() fresh at point of use, never cache across a request
	loader        *metadata.Loader   // CAP-X04: re-run by Reload to build a fresh Interpreter
	pool          *pgxpool.Pool      // CAP-X08 import: its own explicit transaction, separate from workspaceTx's per-request one -- see APIImportApplication's doc comment for why
	records       *store.RecordStore
	notifications *store.NotificationStore
	outbox        *store.OutboxStore // CAP-W06: notify/subscription fan-out enqueue here, runOutboxDispatcher performs the write
	sessions      *store.SessionStore
	users         *store.UserStore
	workspaces    *store.WorkspaceStore // CAP-O09
	groups        *store.GroupStore     // CAP-O07
	secureCookies bool
	engine        *constraint.Engine
	guard         *permission.Guard
	exec          *executor.Executor
	storage       storage.Store // CAP-F06: uploaded-file bytes, see upload.go
}

func New(interp *interpreter.Store, loader *metadata.Loader, pool *pgxpool.Pool, records *store.RecordStore, notifications *store.NotificationStore, outbox *store.OutboxStore, sessions *store.SessionStore, users *store.UserStore, workspaces *store.WorkspaceStore, groups *store.GroupStore, secureCookies bool, fileStorage storage.Store) *Handler {
	return &Handler{
		interp:        interp,
		loader:        loader,
		pool:          pool,
		records:       records,
		notifications: notifications,
		outbox:        outbox,
		sessions:      sessions,
		users:         users,
		workspaces:    workspaces,
		groups:        groups,
		secureCookies: secureCookies,
		engine:        &constraint.Engine{},
		guard:         &permission.Guard{},
		exec:          executor.New(records, outbox),
		storage:       fileStorage,
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

// workspaceName resolves the current session's Workspace to its own Name --
// 006-runtime-model.md's "highest organizational boundary" is the identity
// a person actually gathers under (a company, a department, an event, a
// committee), not the runtime product itself. Used to brand the UI shell
// instead of a fixed "Menata Runtime" string. "Menata Runtime" itself is
// kept as the fallback for the rare case a session's Workspace can't be
// resolved (never expected in practice -- login already requires a valid
// workspace_id), so the nav bar never renders blank.
func (h *Handler) workspaceName(r *http.Request) string {
	if ws, ok := h.interp.Get().GetWorkspace(h.workspace(r)); ok {
		return ws.Name
	}
	return "Menata Runtime"
}

// workspaceSlug (CAP-X14) is the `/{slug}/` segment of the CURRENT request's
// own URL -- every workspace-scoped route already runs under router.Mount's
// `/{wsSlug}` subrouter, so this is a direct chi.URLParam read, not an
// interpreter lookup: the slug building every same-workspace link on this
// page should point back to is exactly the one already in the address bar.
func (h *Handler) workspaceSlug(r *http.Request) string {
	return chi.URLParam(r, "wsSlug")
}

// workspaceSlugForID resolves an arbitrary workspace_id to its own slug --
// unlike workspaceSlug above, used only where there is no incoming `/{slug}`
// URL param to read yet (Login/Signup's own post-success redirect target,
// resolved from the just-authenticated user's WorkspaceID instead).
func (h *Handler) workspaceSlugForID(workspaceID string) string {
	if ws, ok := h.interp.Get().GetWorkspace(workspaceID); ok {
		return ws.Slug
	}
	return workspaceID
}

// roleForApp (CAP-O01/CAP-O07): the acting person's full held-role SET for
// one specific Application — their own direct assignment plus every role
// any Group they belong to holds there (union semantics, merged once in
// sessionAuth). Role is no longer a single session-wide value — the same
// person can be "Requester" in one Application and "Submitter" in another,
// simultaneously, with no manual role switch — so every call site names
// which Application it means, usually via Interpreter.ScopeFor(machineID)'s
// second return value. nil/empty (no direct assignment and no Group
// membership granting one for that Application) flows into
// Guard.CanRead/CanCreate/CanEdit/CanTrigger the same way an absent
// Permissions row already denies by default.
func (h *Handler) roleForApp(r *http.Request, applicationID string) []string {
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
	wsSlug := h.workspaceSlug(r)
	page := ui.CardGrid("Home", h.workspaceName(r), wsSlug, a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), "Applications", "Select an application to view its machines.", "/"+wsSlug+"/apps/", "", cards, h.unreadCount(r.Context(), a))
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
	wsSlug := h.workspaceSlug(r)
	page := ui.CardGrid(app.Name, h.workspaceName(r), wsSlug, a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), app.Name, "Select a machine to view its records.", "/"+wsSlug+"/", "/"+wsSlug+"/", cards, h.unreadCount(r.Context(), a))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render app machines", "error", err)
	}
}

func (h *Handler) uiRoleGroups(workspaceID string) []ui.RoleGroup {
	interpGroups := h.interp.Get().AllRoles(workspaceID)
	out := make([]ui.RoleGroup, 0, len(interpGroups))
	for _, g := range interpGroups {
		out = append(out, ui.RoleGroup{AppID: g.AppID, AppName: g.AppName, Roles: g.Roles})
	}
	return out
}

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

func findFieldByID(machine *model.Machine, id string) *model.Field {
	for _, f := range machine.Fields {
		if f.ID == id {
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

func findMachineContainingField(interp *interpreter.Interpreter, fieldID string) *model.Machine {
	for _, m := range interp.AllMachines() {
		if findFieldByID(m, fieldID) != nil {
			return m
		}
	}
	return nil
}

// --- CAP-A10 in-app notification inbox ---------------------------------------
