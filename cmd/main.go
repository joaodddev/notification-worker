package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/joaodddev/notification-worker/internal/dispatcher"
	"github.com/joaodddev/notification-worker/internal/job"
	"github.com/joaodddev/notification-worker/internal/notifier"
	"github.com/joaodddev/notification-worker/internal/worker"
)

func main() {
	// ── Config ────────────────────────────────────────────────────────────────
	_ = godotenv.Load()

	dbURL := mustEnv("DATABASE_URL")
	port := envOr("PORT", "8080")

	// ── Banco de dados ────────────────────────────────────────────────────────
	ctx := context.Background()

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping db: %v", err)
	}
	log.Println("database connected")

	// ── Dependências ──────────────────────────────────────────────────────────
	repo := job.NewRepository(db)
	webhook := notifier.NewWebhook()

	// Pool com 5 workers e buffer de 20 jobs no channel
	pool := worker.NewPool(5, 20, repo, webhook)

	// Dispatcher: polling a cada 2s, busca até 10 jobs por tick
	disp := dispatcher.New(repo, pool.Jobs, 2*time.Second, 10)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	// Criamos um ctx cancelável para sinalizar parada para dispatcher e workers
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// ── Inicia workers e dispatcher ───────────────────────────────────────────
	pool.Start(runCtx)

	go disp.Run(runCtx)

	// ── HTTP API ──────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// POST /jobs — enfileira um novo job de webhook
	r.Post("/jobs", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL  string         `json:"url"`
			Body map[string]any `json:"body"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.URL == "" {
			http.Error(w, "url is required", http.StatusBadRequest)
			return
		}

		payload, err := json.Marshal(job.WebhookPayload{URL: req.URL, Body: req.Body})
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		j, err := repo.Enqueue(r.Context(), "webhook", payload)
		if err != nil {
			log.Printf("enqueue error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":     j.ID,
			"status": j.Status,
		})
	})

	// GET /health — healthcheck simples
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Sobe o servidor em goroutine separada
	go func() {
		log.Printf("server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Aguarda sinal de parada
	<-quit
	log.Println("shutting down...")

	cancel() // para dispatcher

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	_ = srv.Shutdown(shutdownCtx) // para HTTP server

	pool.Stop() // aguarda workers terminarem

	log.Println("bye!")
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("env %s is required", key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
