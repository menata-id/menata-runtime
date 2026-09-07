package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"menata.id/app/internal/auth"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// invitationTTL (CAP-O10, Study 35 §5.3's own "e.g. 7 days -- exact value
// an owner call, not fixed here" -- 7 days picked as a reasonable default,
// adjust here if that changes).
const invitationTTL = 7 * 24 * time.Hour

// errInvalidInvitation is the one generic outcome for every non-actionable
// token state (unknown, expired, already accepted, revoked) -- same
// "doesn't confirm to a prober which one failed" hygiene CAP-X02's own
// login-failure handling already established for /login.
var errInvalidInvitation = errors.New("This invitation link is invalid or has expired.")

// AdminInvitations (CAP-O10) -- GET /{wsSlug}/admin/invitations: the
// workspace Admin's own invite-management page, alongside CAP-O01's
// existing /admin/users.
func (h *Handler) AdminInvitations(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_view_invitations", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	invs, err := h.invitations.ListByWorkspace(r.Context(), a.User.WorkspaceID)
	if err != nil {
		http.Error(w, "failed to load invitations", http.StatusInternalServerError)
		return
	}
	rows := make([]ui.AdminInvitationRow, 0, len(invs))
	for _, inv := range invs {
		rows = append(rows, ui.AdminInvitationRow{
			ID:     inv.ID,
			Email:  inv.Email,
			Role:   inv.WorkspaceRole,
			Status: invitationDisplayStatus(inv),
		})
	}
	appGroups := h.uiRoleGroups(a.User.WorkspaceID)
	page := ui.AdminInvitations(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.unreadCount(r.Context(), a), rows, appGroups)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render admin invitations", "error", err)
	}
}

// invitationDisplayStatus surfaces "expired" for a pending row past its own
// ExpiresAt -- no background job flips the stored status column, this is
// computed fresh at read time (Invitation.Expired's own doc comment).
func invitationDisplayStatus(inv *store.Invitation) string {
	if inv.Status == "pending" && inv.Expired() {
		return "expired"
	}
	return inv.Status
}

// AdminCreateInvitation (CAP-O10) -- POST /{wsSlug}/admin/invitations:
// creates a pending invitation and emails its accept link. Per-Application
// roles reuse CAP-O01's own vocabulary, validated the same
// deny-by-default way AdminSetGroupRoles' own loop already does.
func (h *Handler) AdminCreateInvitation(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_create_invitation", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}
	workspaceRole := r.FormValue("workspace_role")
	if workspaceRole != "Admin" && workspaceRole != "Member" {
		workspaceRole = "Member"
	}
	appRoles := h.validatedAppRoleSubmission(r, a.User.WorkspaceID)

	token, err := auth.NewToken()
	if err != nil {
		slog.Error("generate invitation token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	inv, err := h.invitations.Create(r.Context(), a.User.WorkspaceID, email, workspaceRole, appRoles,
		auth.HashSessionToken(token), a.User.ID, time.Now().Add(invitationTTL))
	if err != nil {
		slog.Error("create invitation", "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}

	h.sendInvitationEmail(r, inv, token)

	slog.Info("invitation created", "correlation_id", middleware.GetReqID(r.Context()), "actor", a.User.Name, "invitee", email)
	http.Redirect(w, r, "/"+h.workspaceSlug(r)+"/admin/invitations", http.StatusSeeOther)
}

// validatedAppRoleSubmission mirrors AdminSetGroupRoles' own inline
// validation loop (admin.go) -- a submitted app_role_<applicationID> value
// is kept only if it's one that Application's own Machines' Permissions
// actually declare, deny-by-default like every other role assignment path.
func (h *Handler) validatedAppRoleSubmission(r *http.Request, workspaceID string) map[string]string {
	out := map[string]string{}
	for _, g := range h.interp.Get().AllRoles(workspaceID) {
		submitted := r.FormValue("app_role_" + g.AppID)
		if submitted == "" {
			continue
		}
		for _, role := range g.Roles {
			if role == submitted {
				out[g.AppID] = submitted
				break
			}
		}
	}
	return out
}

// AdminRevokeInvitation (CAP-O10) -- POST
// /{wsSlug}/admin/invitations/{invitationID}/revoke: its accept link stops
// working immediately (resolveInvitation's own status check rejects
// anything but "pending").
func (h *Handler) AdminRevokeInvitation(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_revoke_invitation", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	id := chi.URLParam(r, "invitationID")
	if _, err := h.invitations.GetByID(r.Context(), id, a.User.WorkspaceID); err != nil {
		http.NotFound(w, r) // CAP-X06: another workspace's invitation 404s like one that doesn't exist
		return
	}
	if err := h.invitations.Revoke(r.Context(), id, a.User.WorkspaceID); err != nil {
		slog.Error("revoke invitation", "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+h.workspaceSlug(r)+"/admin/invitations", http.StatusSeeOther)
}

// AdminResendInvitation (CAP-O10) -- POST
// /{wsSlug}/admin/invitations/{invitationID}/resend: reissues a fresh
// token+expiry on the SAME row (its own role assignments untouched) and
// re-sends the email -- for a real SMTP failure or a lapsed window,
// without the Admin re-entering anything.
func (h *Handler) AdminResendInvitation(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		a := h.auth(r)
		h.logPermissionDenied(r.Context(), "admin_resend_invitation", "", "", []string{a.User.WorkspaceRole}, a.User.Name)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	a := h.auth(r)
	id := chi.URLParam(r, "invitationID")
	inv, err := h.invitations.GetByID(r.Context(), id, a.User.WorkspaceID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	token, err := auth.NewToken()
	if err != nil {
		slog.Error("generate invitation token", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(invitationTTL)
	if err := h.invitations.Reissue(r.Context(), inv.ID, auth.HashSessionToken(token), expiresAt); err != nil {
		slog.Error("reissue invitation", "error", err)
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}
	inv.ExpiresAt = expiresAt
	h.sendInvitationEmail(r, inv, token)
	http.Redirect(w, r, "/"+h.workspaceSlug(r)+"/admin/invitations", http.StatusSeeOther)
}

func (h *Handler) sendInvitationEmail(r *http.Request, inv *store.Invitation, token string) {
	link := h.absoluteURL(r, "/"+h.workspaceSlug(r)+"/invite/accept?token="+token)
	workspaceName := h.workspaceName(r)
	subject := fmt.Sprintf("You've been invited to %s on Menata", workspaceName)
	body := fmt.Sprintf(
		"You've been invited to join the %q workspace as %s.\n\nAccept the invitation:\n%s\n\nThis link expires in 7 days.",
		workspaceName, inv.WorkspaceRole, link,
	)
	if err := h.mailer.Send(r.Context(), inv.Email, subject, body); err != nil {
		slog.Error("send invitation email", "correlation_id", middleware.GetReqID(r.Context()), "invitation", inv.ID, "error", err)
	}
}

// absoluteURL builds a real, externally-reachable link for an emailed
// invitation -- derived from the live request's own Host header (already
// the real public domain, Caddy forwards it unchanged) rather than a new
// hardcoded base-URL config value, matching h.secureCookies' own existing
// scheme signal instead of introducing a second one.
func (h *Handler) absoluteURL(r *http.Request, path string) string {
	scheme := "http"
	if h.secureCookies {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}

// resolveInvitation (CAP-O10) looks up token and validates it's still
// usable (exists, pending, not expired, and the URL's own workspace slug
// actually matches the invitation's workspace -- a token minted for one
// workspace can't be replayed against another's /invite/accept path even
// though the hash alone would still resolve). A nil *store.User return
// means no identity exists yet for this email -- the new-account branch.
func (h *Handler) resolveInvitation(r *http.Request, token string) (*store.Invitation, *store.User, error) {
	if token == "" {
		return nil, nil, errInvalidInvitation
	}
	inv, err := h.invitations.GetByTokenHash(r.Context(), auth.HashSessionToken(token))
	if err != nil || inv.Status != "pending" || inv.Expired() {
		return nil, nil, errInvalidInvitation
	}
	ws, ok := h.interp.Get().GetWorkspace(inv.WorkspaceID)
	if !ok || ws.Slug != h.workspaceSlug(r) {
		return nil, nil, errInvalidInvitation
	}
	user, err := h.users.GetByEmail(r.Context(), inv.Email)
	if err != nil {
		return inv, nil, nil
	}
	return inv, user, nil
}

// InviteAcceptForm (CAP-O10) -- GET /{wsSlug}/invite/accept?token=...:
// public, no session required (see cmd/server/main.go's isPublicPath/
// csrfProtect). Renders the new-account form (no identity yet), the
// password-confirm form (identity exists, not yet a member here), or
// resolves an already-a-member invitation as redundant and sends the
// visitor to /login instead.
func (h *Handler) InviteAcceptForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	inv, user, err := h.resolveInvitation(r, token)
	if err != nil {
		if e := ui.InviteInvalidPage(err.Error()).Render(r.Context(), w); e != nil {
			slog.Error("render invite invalid", "error", e)
		}
		return
	}
	if user != nil {
		if _, isMember, mErr := h.memberships.RoleFor(r.Context(), user.ID, inv.WorkspaceID); mErr == nil && isMember {
			_ = h.invitations.MarkAccepted(r.Context(), inv.ID)
			http.Redirect(w, r, "/login?message=already-a-member", http.StatusSeeOther)
			return
		}
	}
	if e := ui.InviteAcceptPage("", token, h.workspaceSlug(r), inv.Email, user != nil).Render(r.Context(), w); e != nil {
		slog.Error("render invite accept", "error", e)
	}
}

// InviteAccept (CAP-O10) -- POST /{wsSlug}/invite/accept: public, same
// unauthenticated exemption as the GET above. Re-derives which branch
// applies fresh from the token (never trusts a client-submitted "which
// form" hint) -- a new identity (Name+Password create it), or an existing
// one (its own Password proves control, resolving Study 35 §5.3's own
// flagged cross-workspace ambiguity via CAP-O11's membership model: never
// a second disconnected identity, just a new membership row on the real
// one).
func (h *Handler) InviteAccept(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	token := r.FormValue("token")
	inv, user, err := h.resolveInvitation(r, token)
	if err != nil {
		if e := ui.InviteInvalidPage(err.Error()).Render(r.Context(), w); e != nil {
			slog.Error("render invite invalid", "error", e)
		}
		return
	}

	wsSlug := h.workspaceSlug(r)
	rerender := func(errMsg string) {
		w.WriteHeader(http.StatusBadRequest)
		if e := ui.InviteAcceptPage(errMsg, token, wsSlug, inv.Email, user != nil).Render(r.Context(), w); e != nil {
			slog.Error("render invite accept (failed)", "error", e)
		}
	}

	if user == nil {
		newUser, ok := h.createInvitedIdentity(w, r, inv, rerender)
		if !ok {
			return
		}
		user = newUser
	} else if !h.joinExistingIdentity(w, r, user, inv, rerender) {
		return
	}

	if sErr := h.startSession(w, r, user.ID, inv.WorkspaceID); sErr != nil {
		slog.Error("start session after invite accept", "error", sErr)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setLastWorkspaceCookie(w, h.secureCookies, wsSlug)
	slog.Info("invitation accepted", "correlation_id", middleware.GetReqID(r.Context()), "invitee", inv.Email, "workspace", inv.WorkspaceID)
	http.Redirect(w, r, "/"+wsSlug+"/", http.StatusSeeOther)
}

// joinExistingIdentity (CAP-O10) is InviteAccept's other branch: the
// invited email already has a real identity, so the only thing left to do
// is prove control of it (its own Password) and attach a new membership --
// never a second disconnected identity, resolving Study 35 §5.3's own
// flagged cross-workspace ambiguity via CAP-O11's membership model.
func (h *Handler) joinExistingIdentity(w http.ResponseWriter, r *http.Request, user *store.User, inv *store.Invitation, rerender func(string)) bool {
	if !auth.VerifyPassword(user.PasswordHash, r.FormValue("password")) {
		rerender("Incorrect password.")
		return false
	}
	if _, err := h.memberships.Create(r.Context(), user.ID, inv.WorkspaceID, inv.WorkspaceRole); err != nil {
		slog.Error("create invited membership", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return false
	}
	h.applyInvitationAppRoles(r.Context(), user.ID, inv)
	if err := h.invitations.MarkAccepted(r.Context(), inv.ID); err != nil {
		slog.Error("mark invitation accepted", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return false
	}
	return true
}

// createInvitedIdentity (CAP-O10) is InviteAccept's new-identity branch,
// its own explicit transaction on the raw pool -- same reasoning Signup's
// own transaction already established (no ambient workspaceTx to attach
// to before the membership itself exists).
func (h *Handler) createInvitedIdentity(w http.ResponseWriter, r *http.Request, inv *store.Invitation, rerender func(string)) (*store.User, bool) {
	name := strings.TrimSpace(r.FormValue("name"))
	password := r.FormValue("password")
	if name == "" || password == "" {
		rerender("Name and password are required.")
		return nil, false
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("hash password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin invite-accept transaction", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(r.Context())
		}
	}()
	ctx := store.WithTx(r.Context(), tx)

	newUser, err := h.users.Create(ctx, name, inv.Email, passwordHash)
	if err != nil {
		slog.Error("create invited user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	if _, err := h.memberships.Create(ctx, newUser.ID, inv.WorkspaceID, inv.WorkspaceRole); err != nil {
		slog.Error("create invited membership", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	h.applyInvitationAppRoles(ctx, newUser.ID, inv)
	if err := h.invitations.MarkAccepted(ctx, inv.ID); err != nil {
		slog.Error("mark invitation accepted", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit invite-accept transaction", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return nil, false
	}
	committed = true
	return newUser, true
}

func (h *Handler) applyInvitationAppRoles(ctx context.Context, userID string, inv *store.Invitation) {
	for appID, role := range inv.ApplicationRoles {
		if err := h.users.SetApplicationRole(ctx, userID, appID, role); err != nil {
			slog.Error("apply invitation application role", "error", err, "application", appID)
		}
	}
}
