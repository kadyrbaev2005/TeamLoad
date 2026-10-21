package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"teamload/internal/ai"
	"teamload/internal/config"
	"teamload/internal/store"
)

// Server holds the API dependencies.
type Server struct {
	cfg      config.Config
	st       *store.Store
	ai       *ai.Service
	provider ai.Provider
	login    *rateLimiter
	now      func() time.Time
}

// Option customises a Server (used by tests).
type Option func(*Server)

// WithClock overrides the time source.
func WithClock(now func() time.Time) Option { return func(s *Server) { s.now = now } }

// WithProvider overrides the LLM provider selected by the config.
func WithProvider(p ai.Provider) Option { return func(s *Server) { s.provider = p } }

// NewServer builds the API.
func NewServer(cfg config.Config, st *store.Store, opts ...Option) (*Server, error) {
	s := &Server{cfg: cfg, st: st, now: time.Now}
	for _, o := range opts {
		o(s)
	}
	name := cfg.LLMProvider
	if s.provider == nil {
		p, err := ai.NewProvider(cfg.LLMProvider, cfg.LLMAPIKey, cfg.LLMModel, cfg.MockDelay)
		if err != nil {
			return nil, err
		}
		s.provider = p
	} else {
		name = fmt.Sprintf("%T", s.provider)
	}
	s.ai = ai.NewService(s.provider, name, aiRecorder{s}, cfg.LLMTimeout, cfg.LLMMaxCallsPerDay)
	s.login = newRateLimiter(cfg.LoginAttemptsPer15m, 15*time.Minute, s.now)
	return s, nil
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
		r.Post("/auth/register", handler(s.handleRegister).ServeHTTP)
		r.Post("/auth/login", handler(s.handleLogin).ServeHTTP)
		r.Post("/auth/logout", handler(s.handleLogout).ServeHTTP)

		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/auth/me", handler(s.handleMe).ServeHTTP)
			r.Get("/me/profile", handler(s.handleGetProfile).ServeHTTP)
			r.Put("/me/profile", handler(s.handlePutProfile).ServeHTTP)
			r.Delete("/me/profile", handler(s.handleDeleteProfile).ServeHTTP)

			r.Get("/projects", handler(s.handleListProjects).ServeHTTP)
			r.Post("/projects", handler(s.handleCreateProject).ServeHTTP)
			r.Get("/invites/{code}", handler(s.handleGetInvite).ServeHTTP)
			r.Post("/invites/{code}/join", handler(s.handleJoin).ServeHTTP)

			r.Route("/projects/{projectID}", func(r chi.Router) {
				r.Use(s.requireMember)
				r.Get("/", handler(s.handleGetProject).ServeHTTP)
				r.Get("/plan", handler(s.handleGetPlan).ServeHTTP)
				r.Get("/matching", handler(s.handleGetMatching).ServeHTTP)
				r.Get("/raci", handler(s.handleGetRaci).ServeHTTP)
				r.Get("/dashboard", handler(s.handleDashboard).ServeHTTP)
				// Every member manages tasks (spec 5).
				r.Get("/tasks", handler(s.handleListTasks).ServeHTTP)
				r.Post("/tasks", handler(s.handleCreateTask).ServeHTTP)
				r.Patch("/tasks/{taskID}", handler(s.handlePatchTask).ServeHTTP)
				r.Delete("/tasks/{taskID}", handler(s.handleDeleteTask).ServeHTTP)

				r.Group(func(r chi.Router) {
					r.Use(requireOwner)
					r.Patch("/", handler(s.handlePatchProject).ServeHTTP)
					r.Post("/invite-code", handler(s.handleNewInviteCode).ServeHTTP)
					r.Delete("/members/{userID}", handler(s.handleRemoveMember).ServeHTTP)
					r.Put("/plan", handler(s.handlePutPlan).ServeHTTP)
					r.Post("/plan/generate", handler(s.handleGeneratePlan).ServeHTTP)
					r.Post("/plan/standard", handler(s.handleStandardPlan).ServeHTTP)
					r.Post("/plan/approve", handler(s.handleApprovePlan).ServeHTTP)
					r.Post("/matching/run", handler(s.handleRunMatching).ServeHTTP)
					r.Post("/matching/accept", handler(s.handleAcceptMatching).ServeHTTP)
					r.Put("/matching/seats/{seatID}", handler(s.handleAssignSeat).ServeHTTP)
					r.Put("/raci", handler(s.handlePutRaci).ServeHTTP)
					r.Post("/raci/generate", handler(s.handleGenerateRaci).ServeHTTP)
				})
			})
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
