package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"menata.id/runtime/internal/auth"
	"menata.id/runtime/internal/config"
	"menata.id/runtime/internal/db"
	"menata.id/runtime/internal/handler"
	"menata.id/runtime/internal/interpreter"
	"menata.id/runtime/internal/metadata"
	"menata.id/runtime/internal/router"
	"menata.id/runtime/internal/store"
)

func main() {
	// One JSON handler for every log line this process writes — startup,
	// access log, and every explicit slog call in handler/executor
	// (permission denials, rule violations, role switches, notify, render
	// errors). Previously chi's middleware.Logger wrote plain text via the
	// stdlib `log` package while everything else went through slog's default
	// text handler — two different formats, correlated by request id since
	// CAP-I04 but not actually unified. Must be set before anything else logs.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	_ = godotenv.Load()

	cfg := config.Load()

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	loader := metadata.NewLoader(pool)
	workspaces, err := loader.LoadAll(context.Background())
	if err != nil {
		slog.Error("failed to load runtime metadata", "error", err)
		os.Exit(1)
	}

	interp := interpreter.New(workspaces)
	for _, m := range interp.AllMachines() {
		slog.Info("machine loaded",
			"id", m.ID,
			"fields", len(m.Fields),
			"events", len(m.Events),
			"views", len(m.Views),
		)
	}

	records := store.NewRecordStore(pool)
	notifications := store.NewNotificationStore(pool)
	sessions := store.NewSessionStore(pool)
	users := store.NewUserStore(pool)
	h := handler.New(interp, records, notifications, sessions, users, cfg.SecureCookies)

	r := chi.NewRouter()
	// RequestID before the access logger: gives every access log line a
	// request id, and makes it readable via middleware.GetReqID(ctx)
	// anywhere downstream — CAP-I04's correlation_id for record_events
	// (executor.Persist) and every explicit security-event log line
	// (permission denials, rule violations) reuse this same id, not a
	// separately generated one.
	r.Use(middleware.RequestID)
	r.Use(slogAccessLog)
	r.Use(middleware.Recoverer)
	// CAP-X02: resolves the session and attaches store.Auth to ctx before
	// anything else runs -- workspaceTx's own workspace_id now comes from
	// the authenticated User (store.AuthFromContext), not a client-suppliable
	// cookie, so this must run first.
	r.Use(sessionAuth(sessions, users))
	// CAP-X02: CSRF check, after sessionAuth (needs the session's stored
	// token from ctx) and before workspaceTx (a rejected request shouldn't
	// pay for opening a transaction).
	r.Use(csrfProtect)
	// CAP-X06: after Recoverer, not before -- a panic inside a request must
	// unwind through this middleware's own deferred rollback first (so the
	// transaction never leaks un-rolled-back), then reach Recoverer to
	// actually produce the 500 response. See workspaceTx's doc comment.
	r.Use(workspaceTx(pool))

	// Serve compiled Tailwind CSS and other static assets.
	r.Handle("/static/*", http.StripPrefix("/static/",
		http.FileServer(http.Dir("web/static"))))

	router.Mount(r, h)

	addr := ":" + cfg.Port
	slog.Info("menata runtime listening", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

// workspaceTx is CAP-X06's enforcement mechanism: wraps every request in its
// own transaction and sets app.workspace_id via SET LOCAL (through
// set_config's third arg, is_local=true) -- the only safe way to scope a
// Postgres GUC per-request against a pooled connection. A plain SET (not
// LOCAL) on a pooled connection leaks into whichever unrelated request next
// happens to reuse that connection; SET LOCAL is transaction-scoped and
// Postgres guarantees it resets at COMMIT or ROLLBACK, however the
// transaction ends -- including a panic, since the deferred rollback below
// always runs during Go's panic unwinding, before the panic reaches
// middleware.Recoverer (which must be registered *before* this middleware
// -- see main()).
//
// The workspace itself comes from the authenticated session (store.
// AuthFromContext's User.WorkspaceID), not a client-suppliable cookie --
// closes the gap where a request could set app.workspace_id via RLS to one
// workspace while the app-layer guard checked a role from another (CAP-X02).
// Requests with no Auth in context (the session middleware's own exempted
// paths: /login, /health, /static/*) fall back to "ws_default", matching
// this codebase's pre-workspace-awareness default -- none of those paths
// touch RLS-scoped tables anyway. The resulting tx is attached to the
// request's context (store.WithTx) -- every RecordStore/NotificationStore
// method already prefers it over the raw pool when present.
//
// RLS (migrations/009, applied only at final cutover) is what actually
// makes app.workspace_id matter: without it, this middleware sets a GUC
// nothing reads yet. This middleware has to exist *before* that cutover,
// not after -- see migrations/009's own header.
func workspaceTx(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			workspaceID := "ws_default"
			if a, ok := store.AuthFromContext(r.Context()); ok {
				workspaceID = a.User.WorkspaceID
			}

			tx, err := pool.Begin(r.Context())
			if err != nil {
				slog.Error("begin request transaction", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			committed := false
			defer func() {
				if !committed {
					_ = tx.Rollback(r.Context())
				}
			}()

			if _, err := tx.Exec(r.Context(), `SELECT set_config('app.workspace_id', $1, true)`, workspaceID); err != nil {
				slog.Error("set workspace context", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r.WithContext(store.WithTx(r.Context(), tx)))

			if ww.Status() >= 500 {
				return // deferred rollback handles it
			}
			if err := tx.Commit(r.Context()); err != nil {
				slog.Error("commit request transaction", "error", err)
				return
			}
			committed = true
		})
	}
}

// slogAccessLog replaces chi's middleware.Logger (stdlib `log`, plain text)
// with one that writes through the same slog JSON handler as every other log
// line in this process, using the same "correlation_id" key
// logPermissionDenied/logRuleViolation/executor.Persist already use — one
// format, one shared id, not two differently-shaped outputs stitched
// together after the fact.
func slogAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("http request",
			"correlation_id", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// publicPaths never require a session -- the login page and its own POST
// (nothing to authenticate against yet), the health check (ops tooling, not
// a browser session), and static assets (CSS/JS, not per-user).
func isPublicPath(path string) bool {
	if path == "/login" || path == "/health" {
		return true
	}
	return strings.HasPrefix(path, "/static/")
}

// sessionAuth is CAP-X02's core enforcement: resolves the session cookie
// into a real store.Auth (User + CAP-O01 per-Application role map + CSRF
// token) and attaches it to ctx, or rejects the request. Runs before
// workspaceTx (see main()'s registration order) since workspaceTx's own
// workspace_id now comes from the User this middleware resolves, not a
// client-suppliable cookie.
//
// Unauthenticated: a GET is redirected to /login (a browser navigating
// somewhere it can't see yet is normal, not an error); anything else
// (a POST from an expired session, a stale HTMX request) gets a plain 401 --
// there's no useful page to redirect a form submission to.
func sessionAuth(sessions *store.SessionStore, users *store.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			deny := func() {
				if r.Method == http.MethodGet {
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			}

			c, err := r.Cookie("menata_session")
			if err != nil || c.Value == "" {
				deny()
				return
			}
			tokenHash := auth.HashSessionToken(c.Value)
			sess, err := sessions.Get(r.Context(), tokenHash)
			if err != nil {
				deny()
				return
			}
			user, err := users.GetByID(r.Context(), sess.UserID)
			if err != nil {
				deny()
				return
			}
			appRoles, err := users.ApplicationRoles(r.Context(), user.ID)
			if err != nil {
				slog.Error("load application roles", "correlation_id", middleware.GetReqID(r.Context()), "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			// Sliding expiration: an active session never expires mid-use.
			// Best-effort -- a failed touch doesn't fail the request, it just
			// means this session expires on its original schedule instead of
			// being extended.
			if err := sessions.Touch(r.Context(), tokenHash, time.Now().Add(auth.SessionTTL)); err != nil {
				slog.Error("touch session", "correlation_id", middleware.GetReqID(r.Context()), "error", err)
			}

			ctx := store.WithAuth(r.Context(), &store.Auth{
				User:             user,
				ApplicationRoles: appRoles,
				CSRFToken:        sess.CSRFToken,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// csrfProtect compares every non-GET request's submitted csrf_token form
// field against the authenticated session's own stored token
// (crypto/subtle constant-time compare, via auth.ConstantTimeEqual) --
// standard synchronizer-token CSRF defense. /login's POST is exempt: no
// session exists yet to compare against, and login CSRF isn't this
// prototype's threat model (see the plan's own note on this).
func csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/login" {
			next.ServeHTTP(w, r)
			return
		}

		a, ok := store.AuthFromContext(r.Context())
		if !ok {
			// sessionAuth already rejects an unauthenticated non-GET before
			// this middleware runs -- reaching here without an Auth means a
			// path was (mis)configured as public despite accepting a POST.
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		// CAP-R06: a multipart/form-data POST (CSV upload) needs
		// ParseMultipartForm, not ParseForm -- ParseForm only reads the
		// body for application/x-www-form-urlencoded, but it still sets
		// r.Form (to the query string alone), and FormValue skips its own
		// parse step once r.Form is already non-nil -- so calling plain
		// ParseForm first would silently make csrf_token unreadable on any
		// multipart POST from here on. ParseMultipartForm itself errors on
		// a non-multipart request, so this has to branch on Content-Type,
		// not just always call the multipart-aware parser.
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
		} else if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		submitted := r.FormValue("csrf_token")
		if submitted == "" || !auth.ConstantTimeEqual(submitted, a.CSRFToken) {
			slog.Warn("csrf token mismatch",
				"correlation_id", middleware.GetReqID(r.Context()),
				"path", r.URL.Path,
				"identity", a.User.Name,
			)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
