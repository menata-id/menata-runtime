package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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
