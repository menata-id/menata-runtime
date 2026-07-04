package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

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
	h := handler.New(interp, records)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	router.Mount(r, h)

	addr := ":" + cfg.Port
	slog.Info("menata runtime listening", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
