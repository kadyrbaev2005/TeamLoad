// Package config reads the API configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the API configuration.
type Config struct {
	Addr                string
	DatabaseURL         string
	AppURL              string
	Location            *time.Location
	JWTSecret           []byte
	SecureCookies       bool
	DevCORSOrigin       string
	LLMProvider         string
	LLMAPIKey           string
	LLMModel            string
	LLMMaxCallsPerDay   int
	LLMTimeout          time.Duration
	MockDelay           time.Duration
	LoginAttemptsPer15m int
}

// Load reads the environment. Missing optional values get dev defaults.
func Load() (Config, error) {
	c := Config{
		Addr:          ":" + env("PORT", "8081"),
		DatabaseURL:   env("DATABASE_URL", "postgres://teamload:teamload@localhost:5432/teamload?sslmode=disable"),
		AppURL:        strings.TrimRight(env("APP_URL", "http://localhost:8080"), "/"),
		DevCORSOrigin: os.Getenv("DEV_CORS_ORIGIN"),
		LLMProvider:   strings.ToLower(env("LLM_PROVIDER", "mock")),
		LLMAPIKey:     os.Getenv("LLM_API_KEY"),
		LLMModel:      env("LLM_MODEL", "gemini-2.5-flash"),
	}
	c.SecureCookies = strings.HasPrefix(c.AppURL, "https://")

	loc, err := time.LoadLocation(env("APP_TZ", "Asia/Almaty"))
	if err != nil {
		return c, fmt.Errorf("APP_TZ: %w", err)
	}
	c.Location = loc

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		if c.SecureCookies {
			return c, fmt.Errorf("JWT_SECRET is required when APP_URL is https")
		}
		secret = "dev-only-insecure-secret-change-me" // #nosec G101 -- dev default, refused for https
	}
	c.JWTSecret = []byte(secret)

	if c.LLMMaxCallsPerDay, err = envInt("LLM_MAX_CALLS_PER_PROJECT_PER_DAY", 30); err != nil {
		return c, err
	}
	if c.LoginAttemptsPer15m, err = envInt("LOGIN_ATTEMPTS_PER_15M", 10); err != nil {
		return c, err
	}
	timeoutS, err := envInt("LLM_TIMEOUT_SECONDS", 30)
	if err != nil {
		return c, err
	}
	c.LLMTimeout = time.Duration(timeoutS) * time.Second
	delayMs, err := envInt("MOCK_DELAY_MS", 1500)
	if err != nil {
		return c, err
	}
	c.MockDelay = time.Duration(delayMs) * time.Millisecond

	if c.LLMProvider != "mock" && c.LLMAPIKey == "" {
		return c, fmt.Errorf("LLM_API_KEY is required for LLM_PROVIDER=%s", c.LLMProvider)
	}
	return c, nil
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) (int, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", k, err)
	}
	return n, nil
}
