package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr           string
	AllowedOrigins []string
	TickRate       int
	MaxPlayers     int

	// LLM settings.
	LLMURL          string        // empty = mock mode
	LLMWorkers      int           // concurrent LLM worker goroutines
	PromptCooldown  time.Duration // minimum time between prompts per player

	// Database.
	DatabaseURL string // empty = anonymous mode (no persistence)

	// Auth.
	JWTSecret string
}

func Load() *Config {
	cfg := &Config{
		Addr:           ":8080",
		AllowedOrigins: []string{"http://localhost:5173"},
		TickRate:       30,
		MaxPlayers:     200,
		LLMURL:         "",
		LLMWorkers:     4,
		PromptCooldown: 30 * time.Second,
	}

	if addr := os.Getenv("PORT"); addr != "" {
		cfg.Addr = ":" + addr
	}

	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = strings.Split(origins, ",")
	}

	if rate := os.Getenv("TICK_RATE"); rate != "" {
		if r, err := strconv.Atoi(rate); err == nil && r > 0 {
			cfg.TickRate = r
		}
	}

	if mp := os.Getenv("MAX_PLAYERS"); mp != "" {
		if m, err := strconv.Atoi(mp); err == nil && m > 0 {
			cfg.MaxPlayers = m
		}
	}

	if url := os.Getenv("LLM_URL"); url != "" {
		cfg.LLMURL = url
	}

	if w := os.Getenv("LLM_WORKERS"); w != "" {
		if n, err := strconv.Atoi(w); err == nil && n > 0 {
			cfg.LLMWorkers = n
		}
	}

	if cd := os.Getenv("PROMPT_COOLDOWN"); cd != "" {
		if d, err := time.ParseDuration(cd); err == nil && d > 0 {
			cfg.PromptCooldown = d
		}
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.DatabaseURL = dbURL
	} else {
		// Dev default matching docker-compose.yml
		cfg.DatabaseURL = "postgres://devmatrix_app:dev_password@localhost:5432/devmatrix?sslmode=disable"
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWTSecret = secret
	} else {
		cfg.JWTSecret = "devmatrix-dev-secret-change-in-prod"
	}

	return cfg
}
