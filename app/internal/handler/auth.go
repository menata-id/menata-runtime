package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"menata.id/app/internal/auth"
	"menata.id/app/internal/ui"
)

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
