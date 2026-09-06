package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"menata.id/app/internal/auth"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// Root (CAP-X14) is the one route left unslugged at "/": never real
// content, a redirect only -- unauthenticated to /login, authenticated to
// the session's own Workspace slug. Real workspace-scoped content always
// lives under /{wsSlug}/... from here on.
func (h *Handler) Root(w http.ResponseWriter, r *http.Request) {
	a, ok := store.AuthFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/"+h.workspaceSlugForID(a.User.WorkspaceID)+"/", http.StatusSeeOther)
}

// LoginForm — email + password login page (CAP-X02).
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if err := ui.LoginPage("").Render(r.Context(), w); err != nil {
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
	user, err := h.users.Create(ctx, ws.ID, in.Name, in.Email, passwordHash, "Admin")
	if err != nil {
		slog.Error("create founding user", "error", err)
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

	if err := h.startSession(w, r, user.ID); err != nil {
		slog.Error("start session after signup", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
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

	if err := h.startSession(w, r, user.ID); err != nil {
		slog.Error("start session", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	slog.Info("login", "correlation_id", middleware.GetReqID(r.Context()), "identity", user.Name, "workspace", user.WorkspaceID)
	http.Redirect(w, r, "/"+h.workspaceSlugForID(user.WorkspaceID)+"/", http.StatusSeeOther)
}

// startSession (CAP-X02) always mints a brand-new session (never reuse or
// upgrade a pre-existing one — session-fixation defense) with a fresh CSRF
// token, and sets the session cookie. Shared by Login and Signup (CAP-O09)
// — both end the same way, a real account that should now be signed in.
func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, userID string) error {
	token, err := auth.NewToken()
	if err != nil {
		return err
	}
	csrfToken, err := auth.NewToken()
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(auth.SessionTTL)
	if err := h.sessions.Create(r.Context(), auth.HashSessionToken(token), userID, csrfToken, expiresAt); err != nil {
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
