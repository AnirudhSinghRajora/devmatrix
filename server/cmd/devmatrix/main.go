package main

import (
"context"
"net/http"
"os"
"os/signal"
"syscall"
"time"

"github.com/DevMatrix/server/internal/config"
"github.com/DevMatrix/server/internal/game"
"github.com/DevMatrix/server/internal/network"
"github.com/rs/zerolog"
"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().Logger()

	cfg := config.Load()

	hub := network.NewHub(cfg.AllowedOrigins)
	engine := game.NewEngine(cfg.TickRate, hub)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go engine.Run(ctx)

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

	log.Info().Str("addr", cfg.Addr).Msg("server starting")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server error")
	}
	log.Info().Msg("server stopped")
}
