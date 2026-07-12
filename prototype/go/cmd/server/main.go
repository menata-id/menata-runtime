package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

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
	h := handler.New(interp, records, notifications)

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
// The workspace itself is resolved from a menata_workspace cookie, the same
// convention as h.role/h.identity, defaulting to "ws_default" so every
// session/test that predates workspace-awareness keeps working unchanged.
// The resulting tx is attached to the request's context (store.WithTx) --
// every RecordStore/NotificationStore method already prefers it over the
// raw pool when present.
//
// RLS (migrations/009, applied only at final cutover) is what actually
// makes app.workspace_id matter: without it, this middleware sets a GUC
// nothing reads yet. This middleware has to exist *before* that cutover,
// not after -- see migrations/009's own header.
func workspaceTx(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			workspaceID := "ws_default"
			if c, err := r.Cookie("menata_workspace"); err == nil && c.Value != "" {
				workspaceID = c.Value
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
