package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"menata.id/app/internal/constraint"
	"menata.id/app/internal/executor"
	"menata.id/app/internal/interpreter"
	"menata.id/app/internal/mailer"
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
	workspaces    *store.WorkspaceStore  // CAP-O09
	memberships   *store.MembershipStore // CAP-O11
	groups        *store.GroupStore      // CAP-O07
	invitations   *store.InvitationStore // CAP-O10
	mailer        mailer.Sender          // CAP-O10
	secureCookies bool
	engine        *constraint.Engine
	guard         *permission.Guard
	exec          *executor.Executor
	storage       storage.Store // CAP-F06: uploaded-file bytes, see upload.go
}

func New(interp *interpreter.Store, loader *metadata.Loader, pool *pgxpool.Pool, records *store.RecordStore, notifications *store.NotificationStore, outbox *store.OutboxStore, sessions *store.SessionStore, users *store.UserStore, workspaces *store.WorkspaceStore, memberships *store.MembershipStore, groups *store.GroupStore, invitations *store.InvitationStore, mail mailer.Sender, secureCookies bool, fileStorage storage.Store) *Handler {
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
		memberships:   memberships,
		groups:        groups,
		invitations:   invitations,
		mailer:        mail,
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

// groupMembersFunc (CAP-F24) is the one place a lazy `func(groupID string)
// map[string]bool` closure gets built for Guard.CanTrigger/Interpreter.
// PermittedEventsForRecord's own groupMembers param -- both stay free of a
// direct GroupStore/context dependency (their own doc comments explain
// why), so the real DB call lives here instead, behind a closure that's
// only actually invoked when a matching Permission's DynamicActor resolves
// to "Group" for the specific record being checked. Every call site that
// needs either function passes this exact closure -- never builds its own.
func (h *Handler) groupMembersFunc(ctx context.Context) func(groupID string) map[string]bool {
	return func(groupID string) map[string]bool {
		members, err := h.groups.MemberIDs(ctx, groupID)
		if err != nil {
			slog.Warn("dynamic actor gate: failed to resolve group members", "group", groupID, "error", err)
			return nil
		}
		return members
	}
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
	wsSlug := h.workspaceSlug(r)
	var cards []ui.Card
	// CAP-O03 Tier 5, Phase 1: a declared navigation entirely replaces the
	// inferred listing below for this Application -- see benchmarks/009's
	// implementation-plan follow-on finding.
	if entries := h.interp.Get().NavigationFor(appID); len(entries) > 0 {
		cards = h.declaredNavCards(wsSlug, entries, role, "")
	} else {
		machines := h.interp.Get().MachinesForApplication(appID)
		cards = make([]ui.Card, 0, len(machines))
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
	}
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
	role := h.roleForApp(r, applicationID)
	// CAP-O03 Tier 5, Phase 1: a declared navigation entirely replaces the
	// inferred listing below for this Application.
	if entries := h.interp.Get().NavigationFor(applicationID); len(entries) > 0 {
		links := h.declaredNavLinks(h.workspaceSlug(r), entries, role, machine.ID, r.URL.Path)
		if len(links) < 2 {
			return nil // same "nothing worth switching to" rule the inferred path below already uses
		}
		return links
	}
	siblings := h.interp.Get().MachinesForApplication(applicationID)
	if len(siblings) < 2 {
		return nil // nothing to move sideways to
	}
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

// navigationEntryHref (CAP-O03 Tier 5, Phase 1) resolves a Machine/View
// target to a real path. The View-type-to-slug map mirrors internal/
// router/router.go's own route table exactly -- keep the two in sync;
// internal/metadata/validate.go's navigationCollectionViewTypes is the
// load-time gate that guarantees only a type present here ever reaches
// this function.
func navigationEntryCollectionSlug(t model.ViewType) string {
	switch t {
	case model.ViewTypeDashboard:
		return "/dashboard"
	case model.ViewTypeCalendar:
		return "/calendar"
	case model.ViewTypeTimeline:
		return "/timeline"
	case model.ViewTypeReport:
		return "/report"
	case model.ViewTypeBoard:
		return "/board"
	case model.ViewTypeProcessMap:
		return "/process-map"
	case model.ViewTypeForm:
		return "/new"
	case model.ViewTypePage:
		return "/page" // CAP-V10 Tier 2, 2026-09-09 -- added alongside navigationCollectionViewTypes (metadata/validate.go)
	default: // model.ViewTypeList, and any type validate.go should have already rejected
		return ""
	}
}

// resolvedNavEntry is declaredNavLinks/declaredNavCards' own shared
// intermediate shape -- one tree walked once, permission-trimmed once,
// then rendered into whichever of ui.SubNavLink/ui.Card the caller needs.
type resolvedNavEntry struct {
	Label           string
	Href            string // "" for a group
	IsGroup         bool
	IsMachineTarget bool   // true only for TargetType machine -- see declaredNavLinks' own Active comment
	MachineID       string // "" for a group
}

// resolveDeclaredNav (CAP-O03 Tier 5, Phase 1) walks applicationID's own
// declared navigation_entries tree (already Position-ordered by loader.go)
// depth-first, flattening a group's own children immediately after it --
// a deliberate rendering simplification (see ui.SubNavLink.IsGroup's own
// doc comment): grouping is declared and ordered in the data model, not
// rendered as a real collapsible submenu. Permission-trimmed exactly like
// the inferred path (Guard.CanRead on the resolved Machine); a group with
// zero visible children after trimming is dropped entirely, the same
// "nothing worth showing" rule subNavFor's inferred path already applies.
func (h *Handler) resolveDeclaredNav(wsSlug string, entries []*model.NavigationEntry, role []string) []resolvedNavEntry {
	childrenOf := make(map[string][]*model.NavigationEntry)
	for _, e := range entries {
		childrenOf[e.ParentID] = append(childrenOf[e.ParentID], e)
	}
	var walk func(parentID string) []resolvedNavEntry
	walk = func(parentID string) []resolvedNavEntry {
		var out []resolvedNavEntry
		for _, e := range childrenOf[parentID] {
			switch e.TargetType {
			case model.NavigationTargetMachine:
				m, ok := h.interp.Get().GetMachine(e.TargetMachine)
				if !ok || !h.guard.CanRead(m, role) {
					continue
				}
				out = append(out, resolvedNavEntry{Label: e.Label, Href: "/" + wsSlug + "/" + m.ID, IsMachineTarget: true, MachineID: m.ID})
			case model.NavigationTargetView:
				v, ok := h.interp.Get().GetView(e.TargetView)
				if !ok {
					continue
				}
				m, ok := h.interp.Get().GetMachine(v.MachineID)
				if !ok || !h.guard.CanRead(m, role) {
					continue
				}
				out = append(out, resolvedNavEntry{Label: e.Label, Href: "/" + wsSlug + "/" + v.MachineID + navigationEntryCollectionSlug(v.Type), MachineID: v.MachineID})
			case model.NavigationTargetGroup:
				children := walk(e.ID)
				if len(children) == 0 {
					continue // an empty group heading is noise, same reasoning subNavFor already uses
				}
				out = append(out, resolvedNavEntry{Label: e.Label, IsGroup: true})
				out = append(out, children...)
			}
		}
		return out
	}
	return walk("")
}

// declaredNavLinks resolves entries into the sub-nav strip's own link list.
// Active is deliberately two different granularities depending on target
// type: a Machine target matches by MachineID alone -- coarse, "which
// Machine section am I in," the exact behavior CAP-O03 Tier 2's own
// inferred strip has always had (every page belonging to that Machine
// highlights it, List/Detail/Form alike). A View target is a single
// specific route, not a whole Machine's worth of pages, so it can only
// ever match the CURRENT request's own exact path -- matching by
// MachineID alone would incorrectly mark it active from every OTHER page
// of the same Machine too (caught live: the Dashboard entry lit up on the
// Document's own Detail page before this fix).
func (h *Handler) declaredNavLinks(wsSlug string, entries []*model.NavigationEntry, role []string, activeMachineID, currentPath string) []ui.SubNavLink {
	resolved := h.resolveDeclaredNav(wsSlug, entries, role)
	out := make([]ui.SubNavLink, 0, len(resolved))
	for _, e := range resolved {
		active := false
		switch {
		case e.IsGroup:
			active = false
		case e.IsMachineTarget:
			active = e.MachineID == activeMachineID
		default: // view target
			active = e.Href == currentPath
		}
		out = append(out, ui.SubNavLink{Name: e.Label, Href: e.Href, IsGroup: e.IsGroup, Active: active})
	}
	return out
}

func (h *Handler) declaredNavCards(wsSlug string, entries []*model.NavigationEntry, role []string, activeMachineID string) []ui.Card {
	resolved := h.resolveDeclaredNav(wsSlug, entries, role)
	out := make([]ui.Card, 0, len(resolved))
	for _, e := range resolved {
		out = append(out, ui.Card{Name: e.Label, Href: e.Href, IsHeading: e.IsGroup})
	}
	return out
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
