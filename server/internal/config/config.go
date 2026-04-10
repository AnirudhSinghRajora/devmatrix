package config

import (
"os"
"strconv"
"strings"
)

type Config struct {
	Addr           string
	AllowedOrigins []string
	TickRate       int
}

func Load() *Config {
	cfg := &Config{
		Addr:           ":8080",
		AllowedOrigins: []string{"http://localhost:5173"},
		TickRate:       30,
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

	return cfg
}
