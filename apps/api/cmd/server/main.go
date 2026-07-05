package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"github.com/blackbox-agentdiff/api/internal/auth"
	"github.com/blackbox-agentdiff/api/internal/config"
	"github.com/blackbox-agentdiff/api/internal/diffproxy"
	"github.com/blackbox-agentdiff/api/internal/ingest"
	"github.com/blackbox-agentdiff/api/internal/migrate"
	"github.com/blackbox-agentdiff/api/internal/monitors"
	"github.com/blackbox-agentdiff/api/internal/rest"
	"github.com/blackbox-agentdiff/api/internal/search"
	"github.com/blackbox-agentdiff/api/internal/store"
	"github.com/blackbox-agentdiff/api/internal/webhook"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func isPostgresURL(url string) bool {
	return strings.HasPrefix(url, "postgres://") ||
		strings.HasPrefix(url, "postgresql://") ||
		strings.HasPrefix(url, "pgx://")
}

func main() {
	cfg := config.Load()

	dbURL := cfg.DatabaseURL

	var s store.Store
	if isPostgresURL(dbURL) {
		log.Println("using postgres driver")
		sqlDB, err := sql.Open("pgx", dbURL)
		if err != nil {
			log.Fatalf("open postgres: %v", err)
		}
		defer sqlDB.Close()

		if err := migrate.RunPostgres(sqlDB); err != nil {
			log.Fatalf("postgres migrate: %v", err)
		}

		st, err := store.NewPostgresStore(dbURL)
		if err != nil {
			log.Fatalf("open postgres store: %v", err)
		}
		s = st
	} else {
		log.Println("using sqlite driver")
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
		s = st
	}
	defer func() {
		if s != nil {
			s.Close()
		}
	}()

	if cfg.ClickHouseURL != "" {
		log.Printf("using clickhouse span store at %s", cfg.ClickHouseURL)
		chStore := store.NewClickHouseSpanStore(cfg.ClickHouseURL)
		s = store.NewCombinedStore(s, chStore)
	}

	diffClient := diffproxy.NewClient(cfg.DiffServiceURL)
	dispatcher := webhook.NewDispatcher(s)

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
		r.Use(auth.Middleware(s))
		r.Post("/otel/v1/traces", ingest.NewHandler(s, dispatcher).HTTP)
	})
	_ = otelGroup

	handlers := rest.New(s, diffClient, dispatcher)
	handlers.Register(r)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitors.StartWorker(ctx, s, dispatcher, cfg.MonitorInterval)

	indexer := search.NewIndexer(s)
	go indexer.Start(ctx, 5*time.Minute)

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
