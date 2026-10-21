package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"teamload/internal/config"
	"teamload/internal/store"
)

// Server holds the API dependencies.
type Server struct {
	cfg   config.Config
	st    *store.Store
	login *rateLimiter
	now   func() time.Time
}

// Option customises a Server (used by tests).
type Option func(*Server)

// WithClock overrides the time source.
func WithClock(now func() time.Time) Option { return func(s *Server) { s.now = now } }

// NewServer builds the API.
func NewServer(cfg config.Config, st *store.Store, opts ...Option) *Server {
	s := &Server{cfg: cfg, st: st, now: time.Now}
	for _, o := range opts {
		o(s)
	}
	s.login = newRateLimiter(cfg.LoginAttemptsPer15m, 15*time.Minute, s.now)
	return s
}

// today is the current calendar date in APP_TZ.
func (s *Server) today() time.Time { return s.now().In(s.cfg.Location) }

// Handler returns the HTTP handler with every route mounted under /api.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer, requestLog)
	if s.cfg.DevCORSOrigin != "" {
		r.Use(devCORS(s.cfg.DevCORSOrigin))
	}

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.NotFound(func(w http.ResponseWriter, r *http.Request) { writeErr(w, r, errNotFound("")) })
	})
	return r
}

// devCORS allows the Vite dev server origin with credentials (dev only).
func devCORS(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Origin") == origin {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type")
				h.Add("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
