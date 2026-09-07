package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"menata.id/app/internal/auth"
	"menata.id/app/internal/interpreter"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// Root (CAP-X14) is the one route left unslugged at "/": never real
// content, a redirect only -- unauthenticated to /login, authenticated to
// the session's own Workspace slug, or to /choose-workspace (CAP-O11) if
// this session hasn't picked one yet. Real workspace-scoped content always
// lives under /{wsSlug}/... from here on.
func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	a, ok := store.AuthFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if a.User.WorkspaceID == "" {
		http.Redirect(w, r, "/choose-workspace", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/"+h.workspaceSlugForID(a.User.WorkspaceID)+"/", http.StatusSeeOther)
}

// LoginForm — email + password login page (CAP-X02). message is an
// optional non-error informational banner (CAP-O10's InviteAcceptForm
// redirects here with ?message=already-a-member when an accepted
// invitation turned out to be redundant) -- reuses LoginPage's own single
// banner slot rather than adding a second one just for this.
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	msg := ""
	if r.URL.Query().Get("message") == "already-a-member" {
		msg = "You're already a member of that workspace. Log in to continue."
	}
	if err := ui.LoginPage(msg).Render(r.Context(), w); err != nil {
		slog.Error("render login", "error", err)
	}
}

// SignupForm — CAP-O09: found a new Workspace, become its first Admin.
func (h *Handler) SignupForm(w http.ResponseWriter, r *http.Request) {
	if err := ui.SignupPage("", ui.SignupInput{}).Render(r.Context(), w); err != nil {
		slog.Error("render signup", "error", err)
	}
}

// Signup (CAP-O09) creates a brand-new Workspace and its founder's own
// account in one step, then signs them in immediately. Uses its own
// explicit transaction on the raw pool, not the ambient per-request
// workspaceTx one -- there is no Workspace yet for that one to scope to,
// mirroring APIImportApplication's own established reasoning for the same
// choice (h.pool, not the request-scoped tx). Reload happens only after a
// real commit (h.reloadInterpreter, CAP-X04) -- never swap the live
// Interpreter to reference a row that might not have actually persisted.
func (h *Handler) Signup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in := ui.SignupInput{
		Name:  strings.TrimSpace(r.FormValue("name")),
		Email: strings.TrimSpace(r.FormValue("email")),
		Slug:  strings.TrimSpace(r.FormValue("slug")),
		Label: strings.TrimSpace(r.FormValue("label")),
	}
	password := r.FormValue("password")

	rerender := func(errMsg string) {
		w.WriteHeader(http.StatusBadRequest)
		if err := ui.SignupPage(errMsg, in).Render(r.Context(), w); err != nil {
			slog.Error("render signup (failed)", "error", err)
		}
	}

	if in.Name == "" || in.Email == "" || password == "" || in.Label == "" {
		rerender("All fields are required.")
		return
	}
	if !validSlug(in.Slug) {
		rerender("Workspace URL must be 3-40 characters, lowercase letters/numbers/hyphens only, and not a reserved name.")
		return
	}
	// CAP-O11: email is now a real, globally-unique identity (migrations/026)
	// -- founding a new Workspace with an email that already has an account
	// elsewhere is rejected here rather than silently creating a second,
	// disconnected identity sharing that email (the exact ambiguity CAP-O11
	// exists to close). Real account-linking (join an existing identity into
	// a second Workspace) is CAP-O10's job, not solved by this form.
	if _, err := h.users.GetByEmail(r.Context(), in.Email); err == nil {
		rerender("This email already has an account. Log in instead, or use a different email.")
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		slog.Error("hash password", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		slog.Error("begin signup transaction", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(r.Context())
		}
	}()
	ctx := store.WithTx(r.Context(), tx)

	ws, err := h.workspaces.Create(ctx, in.Slug, in.Label, in.Slug)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateSlug) {
			rerender(store.ErrDuplicateSlug.Error()) // errleak:allow: known sentinel just matched via errors.Is, not a raw internal error
			return
		}
		slog.Error("create workspace", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	user, err := h.users.Create(ctx, in.Name, in.Email, passwordHash)
	if err != nil {
		slog.Error("create founding user", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if _, err := h.memberships.Create(ctx, user.ID, ws.ID, "Admin"); err != nil {
		slog.Error("create founding membership", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("commit signup transaction", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	committed = true

	if _, err := h.reloadInterpreter(r.Context()); err != nil {
		// The workspace/user rows are real and committed -- a reload
		// failure here means the NEW workspace isn't servable yet, not
		// that signup itself failed. Logged for an operator to retry
		// (POST /{slug}/admin/reload once the founder can reach it, or a
		// process restart); the founder still gets a real session below.
		slog.Error("reload after signup", "correlation_id", middleware.GetReqID(r.Context()), "workspace", ws.ID, "error", err)
	}

	// Single membership (the one just created) -- pre-set on the session, no
	// picker, same zero-friction path any other single-membership login gets.
	if err := h.startSession(w, r, user.ID, ws.ID); err != nil {
		slog.Error("start session after signup", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	setLastWorkspaceCookie(w, h.secureCookies, ws.Slug)
	slog.Info("workspace founded", "correlation_id", middleware.GetReqID(r.Context()), "workspace", ws.ID, "founder", user.Name)
	http.Redirect(w, r, "/"+ws.Slug+"/", http.StatusSeeOther)
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

	// CAP-O11: which Workspace(s) this identity actually belongs to decides
	// what happens next -- 0 is a real error, 1 skips the picker entirely
	// (every existing single-membership account's own unchanged path), 2+
	// goes to /choose-workspace, auto-resolved from the remembered last
	// workspace cookie when it still names one of this identity's own real
	// memberships.
	memberships, err := h.memberships.ForUser(r.Context(), user.ID)
	if err != nil {
		slog.Error("list memberships", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	switch {
	case len(memberships) == 0:
		slog.Warn("login with no workspace membership", "correlation_id", middleware.GetReqID(r.Context()), "identity", user.Name)
		w.WriteHeader(http.StatusForbidden)
		if err := ui.LoginPage("This account has no workspace access.").Render(r.Context(), w); err != nil {
			slog.Error("render login (no membership)", "error", err)
		}
		return
	case len(memberships) == 1:
		if err := h.startSession(w, r, user.ID, memberships[0].WorkspaceID); err != nil {
			slog.Error("start session", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		slug := h.workspaceSlugForID(memberships[0].WorkspaceID)
		setLastWorkspaceCookie(w, h.secureCookies, slug)
		slog.Info("login", "correlation_id", middleware.GetReqID(r.Context()), "identity", user.Name, "workspace", memberships[0].WorkspaceID)
		http.Redirect(w, r, "/"+slug+"/", http.StatusSeeOther)
	default:
		remembered := lastWorkspaceMembership(r, h.interp.Get(), memberships)
		if remembered != "" {
			if err := h.startSession(w, r, user.ID, remembered); err != nil {
				slog.Error("start session", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			slug := h.workspaceSlugForID(remembered)
			slog.Info("login", "correlation_id", middleware.GetReqID(r.Context()), "identity", user.Name, "workspace", remembered)
			http.Redirect(w, r, "/"+slug+"/", http.StatusSeeOther)
			return
		}
		if err := h.startSession(w, r, user.ID, ""); err != nil {
			slog.Error("start session", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		slog.Info("login, workspace not yet chosen", "correlation_id", middleware.GetReqID(r.Context()), "identity", user.Name, "memberships", len(memberships))
		http.Redirect(w, r, "/choose-workspace", http.StatusSeeOther)
	}
}

// startSession (CAP-X02) always mints a brand-new session (never reuse or
// upgrade a pre-existing one — session-fixation defense) with a fresh CSRF
// token, and sets the session cookie. Shared by Login and Signup (CAP-O09)
// — both end the same way, a real account that should now be signed in.
// workspaceID may be empty (CAP-O11: not chosen yet, see ChooseWorkspace).
func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, userID, workspaceID string) error {
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	csrfToken, err := auth.NewToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(auth.SessionTTL)
	if err := h.sessions.Create(r.Context(), auth.HashSessionToken(token), userID, workspaceID, csrfToken, expiresAt); err != nil {
		return err
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
	return nil
}

// ChooseWorkspaceForm (CAP-O11) -- GET /choose-workspace: lists every
// Workspace this identity actually belongs to. Reached either because Login
// found 2+ memberships with no usable remembered choice, or because the
// nav bar's own "Switch workspace" link sent an already-fully-authenticated
// session back here on purpose.
func (h *Handler) ChooseWorkspaceForm(w http.ResponseWriter, r *http.Request) {
	a := h.auth(r)
	memberships, err := h.memberships.ForUser(r.Context(), a.User.ID)
	if err != nil {
		slog.Error("list memberships", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	opts := make([]ui.WorkspaceOption, 0, len(memberships))
	for _, m := range memberships {
		ws, ok := h.interp.Get().GetWorkspace(m.WorkspaceID)
		if !ok {
			continue // a membership row naming a Workspace the live Interpreter doesn't know (shouldn't normally happen) -- skip rather than crash
		}
		opts = append(opts, ui.WorkspaceOption{ID: ws.ID, Name: ws.Name, Slug: ws.Slug})
	}
	if err := ui.ChooseWorkspacePage("", a.CSRFToken, opts).Render(r.Context(), w); err != nil {
		slog.Error("render choose-workspace", "error", err)
	}
}

// ChooseWorkspace (CAP-O11) -- POST /choose-workspace: the submitted
// workspace_id is checked against this identity's own real memberships
// again here (never trust a client-submitted id blindly, CAP-P05's own
// "deny by default" discipline) before it's ever written to the session.
func (h *Handler) ChooseWorkspace(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	a := h.auth(r)
	chosen := r.FormValue("workspace_id")
	memberships, err := h.memberships.ForUser(r.Context(), a.User.ID)
	if err != nil {
		slog.Error("list memberships", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	valid := false
	for _, m := range memberships {
		if m.WorkspaceID == chosen {
			valid = true
			break
		}
	}
	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		opts := make([]ui.WorkspaceOption, 0, len(memberships))
		for _, m := range memberships {
			if ws, ok := h.interp.Get().GetWorkspace(m.WorkspaceID); ok {
				opts = append(opts, ui.WorkspaceOption{ID: ws.ID, Name: ws.Name, Slug: ws.Slug})
			}
		}
		if err := ui.ChooseWorkspacePage("That's not one of your workspaces.", a.CSRFToken, opts).Render(r.Context(), w); err != nil {
			slog.Error("render choose-workspace (invalid)", "error", err)
		}
		return
	}
	c, err := r.Cookie("menata_session")
	if err != nil || c.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.sessions.SetWorkspace(r.Context(), auth.HashSessionToken(c.Value), chosen); err != nil {
		slog.Error("set session workspace", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	slug := h.workspaceSlugForID(chosen)
	setLastWorkspaceCookie(w, h.secureCookies, slug)
	http.Redirect(w, r, "/"+slug+"/", http.StatusSeeOther)
}

const lastWorkspaceCookieName = "menata_last_workspace"

// setLastWorkspaceCookie (CAP-O11) remembers a chosen Workspace's slug as a
// UX default only -- not security-sensitive (worst case a stale/mismatched
// value just falls back to showing the picker again, see
// lastWorkspaceMembership), so a plain non-HttpOnly-irrelevant cookie is
// enough; still HttpOnly/SameSite=Lax for basic hygiene, matching every
// other cookie this codebase sets.
func setLastWorkspaceCookie(w http.ResponseWriter, secure bool, slug string) {
	http.SetCookie(w, &http.Cookie{
		Name:     lastWorkspaceCookieName,
		Value:    slug,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 365, // a year -- a returning-visitor convenience, not a security boundary
	})
}

// lastWorkspaceMembership (CAP-O11) resolves the remembered last-workspace
// cookie to a workspace_id, but ONLY if it's still a real membership of
// THIS identity's own list -- a stale cookie (a different identity's
// choice, a membership since revoked, a workspace that no longer exists)
// safely falls back to "" (show the picker), never a silent wrong guess.
func lastWorkspaceMembership(r *http.Request, interp *interpreter.Interpreter, memberships []*store.Membership) string {
	c, err := r.Cookie(lastWorkspaceCookieName)
	if err != nil || c.Value == "" {
		return ""
	}
	ws, ok := interp.WorkspaceBySlug(c.Value)
	if !ok {
		return ""
	}
	for _, m := range memberships {
		if m.WorkspaceID == ws.ID {
			return ws.ID
		}
	}
	return ""
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
