package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/blackbox-agentdiff/api/internal/auth"
	"github.com/blackbox-agentdiff/api/internal/config"
	"github.com/blackbox-agentdiff/api/internal/diffproxy"
	"github.com/blackbox-agentdiff/api/internal/ingest"
	"github.com/blackbox-agentdiff/api/internal/migrate"
	"github.com/blackbox-agentdiff/api/internal/rest"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/blackbox-agentdiff/api/internal/webhook"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()

	// Normalize DATABASE_URL: strip sqlite:// prefix if present so plain
	// file paths work with modernc.org/sqlite driver
	dbURL := cfg.DatabaseURL
	dbURL = strings.TrimPrefix(dbURL, "sqlite://")

	sqlDB, err := sql.Open("sqlite", dbURL)
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer sqlDB.Close()

	if err := migrate.Run(sqlDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	st, err := store.NewSQLiteStore(dbURL)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	diffClient := diffproxy.NewClient(cfg.DiffServiceURL)
	dispatcher := webhook.NewDispatcher(st)

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	otelGroup := r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(st))
		r.Post("/otel/v1/traces", ingest.NewHandler(st, dispatcher).HTTP)
	})
	_ = otelGroup

	handlers := rest.New(st, diffClient, dispatcher)
	handlers.Register(r)

	srv := &http.Server{
		Addr:         ":4000",
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Println("api listening on :4000")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
