package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

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
						Label:       displayLabel(m, rec.ID, rec.Data),
						Link:        "/" + h.workspaceSlug(r) + "/" + m.ID + "/" + rec.ID,
					})
				}
			}
		}
	}

	page := ui.Search(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), query, results, h.unreadCount(r.Context(), a))
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
			link = "/" + h.workspaceSlug(r) + "/" + n.MachineID + "/" + n.RecordID
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

	page := ui.Notifications(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), items, groups, a.User.NotificationPreference, h.unreadCount(r.Context(), a))
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
	http.Redirect(w, r, "/"+h.workspaceSlug(r)+"/notifications", http.StatusSeeOther)
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
	http.Redirect(w, r, "/"+h.workspaceSlug(r)+"/notifications", http.StatusSeeOther)
}
