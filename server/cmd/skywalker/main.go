package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SkyWalker/server/internal/api"
	"github.com/SkyWalker/server/internal/auth"
	"github.com/SkyWalker/server/internal/config"
	"github.com/SkyWalker/server/internal/db"
	"github.com/SkyWalker/server/internal/game"
	"github.com/SkyWalker/server/internal/llm"
	"github.com/SkyWalker/server/internal/network"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// wsAuthAdapter bridges auth.Service → network.AuthValidator.
type wsAuthAdapter struct {
	svc *auth.Service
}

func (a *wsAuthAdapter) ValidateWSToken(tokenStr string) (string, string, error) {
	claims, err := a.svc.ValidateToken(tokenStr)
	if err != nil {
		return "", "", err
	}
	return claims.UserID.String(), claims.Username, nil
}

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()

	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channels bridge the Hub (many goroutines) → Engine (single goroutine).
	joinCh := make(chan network.JoinRequest, 32)
	leaveCh := make(chan string, 32)
	promptCh := make(chan network.PromptRequest, 64)

	// LLM pipeline channels.
	llmReqCh := make(chan game.LLMRequest, 64)
	llmResultCh := make(chan game.LLMResult, 64)

	cooldown := llm.NewCooldownTracker(cfg.PromptCooldown)

	hub := network.NewHub(cfg.AllowedOrigins, cfg.MaxPlayers, joinCh, leaveCh, promptCh)
	llmService := llm.NewService(cfg.LLMURL, cfg.LLMModel, cfg.LLMWorkers, llmReqCh, llmResultCh)
	engine := game.NewEngine(
		cfg.TickRate, hub,
		joinCh, leaveCh, promptCh,
		llmReqCh, llmResultCh, cooldown,
	)

	mux := http.NewServeMux()

	// --- Optional persistence layer ---
	if cfg.DatabaseURL != "" {
		pool, err := db.NewPool(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to database")
		}
		defer pool.Close()

		if err := db.RunMigrations(ctx, pool); err != nil {
			log.Fatal().Err(err).Msg("failed to run migrations")
		}

		itemCache, err := db.NewItemCache(ctx, pool)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to build item cache")
		}

		queries := db.NewQueries(pool)
		authSvc := auth.NewService(cfg.JWTSecret, pool)

		dbWriter := db.NewDBWriter(queries)
		go dbWriter.Run(ctx)

		hub.SetAuthValidator(&wsAuthAdapter{svc: authSvc})
		engine.SetDB(itemCache, dbWriter, queries)

		apiHandler := api.NewHandler(authSvc, queries)
		apiHandler.Register(mux)
		log.Info().Msg("persistence layer enabled")
	} else {
		log.Info().Msg("no DATABASE_URL — running in anonymous mode")
	}

	mux.HandleFunc("/ws", hub.HandleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/debug/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(engine.Stats())
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go engine.Run(ctx)
	go llmService.Run(ctx)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("shutting down")
		cancel()
		hub.Shutdown()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	llmMode := "mock"
	if cfg.LLMURL != "" {
		llmMode = cfg.LLMURL
	}
	log.Info().
		Str("addr", cfg.Addr).
		Int("maxPlayers", cfg.MaxPlayers).
		Str("llm", llmMode).
		Dur("cooldown", cfg.PromptCooldown).
		Msg("server starting")

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server error")
	}
	log.Info().Msg("server stopped")
}
