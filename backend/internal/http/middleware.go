package http

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"teamload/internal/store"
	"teamload/internal/store/db"
)

const (
	sessionCookie = "teamload_session"
	sessionTTL    = 7 * 24 * time.Hour
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxProject
)

func userFrom(ctx context.Context) db.User { return ctx.Value(ctxUser).(db.User) }

func projectFrom(ctx context.Context) db.Project { return ctx.Value(ctxProject).(db.Project) }

func (s *Server) issueSession(w http.ResponseWriter, userID uuid.UUID) error {
	exp := s.now().Add(sessionTTL)
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(s.now()),
		ExpiresAt: jwt.NewNumericDate(exp),
	}).SignedString(s.cfg.JWTSecret)
	if err != nil {
		return err
	}
	// #nosec G124 -- HttpOnly and SameSite=Lax always; Secure follows APP_URL (https).
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: tok, Path: "/", Expires: exp, MaxAge: int(sessionTTL.Seconds()),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.cfg.SecureCookies,
	})
	return nil
}

func (s *Server) clearSession(w http.ResponseWriter) {
	// #nosec G124 -- HttpOnly and SameSite=Lax always; Secure follows APP_URL (https).
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: s.cfg.SecureCookies,
	})
}

func (s *Server) sessionUser(r *http.Request) (db.User, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return db.User{}, false
	}
	var claims jwt.RegisteredClaims
	_, err = jwt.ParseWithClaims(c.Value, &claims, func(*jwt.Token) (any, error) { return s.cfg.JWTSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithTimeFunc(s.now))
	if err != nil {
		return db.User{}, false
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return db.User{}, false
	}
	u, err := s.st.GetUserByID(r.Context(), id)
	if err != nil {
		return db.User{}, false
	}
	return u, true
}

// requireAuth rejects requests without a valid session with 401.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := s.sessionUser(r)
		if !ok {
			writeErr(w, r, errUnauthorized())
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxUser, u)))
	})
}

// requireMember loads {projectID} and answers 404 to non-members.
func (s *Server) requireMember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(chi.URLParam(r, "projectID"))
		if err != nil {
			writeErr(w, r, errNotFound("Project not found."))
			return
		}
		u := userFrom(r.Context())
		ok, err := s.st.IsMember(r.Context(), db.IsMemberParams{ProjectID: id, UserID: u.ID})
		if err != nil {
			writeErr(w, r, err)
			return
		}
		if !ok {
			writeErr(w, r, errNotFound("Project not found."))
			return
		}
		p, err := s.st.GetProject(r.Context(), id)
		if store.IsNotFound(err) {
			writeErr(w, r, errNotFound("Project not found."))
			return
		}
		if err != nil {
			writeErr(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxProject, p)))
	})
}

// requireOwner answers 403 unless the current user owns the project.
func requireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if projectFrom(r.Context()).OwnerID != userFrom(r.Context()).ID {
			writeErr(w, r, errForbidden("Only the project owner can do this."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requestLog logs method, path, status and latency (never bodies).
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.InfoContext(r.Context(), "http", "method", r.Method, "path", r.URL.Path,
			"status", sw.status, "ms", time.Since(start).Milliseconds())
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// clientIP is the address used for rate limiting. nginx (the only public
// entry point) overwrites X-Real-IP with the peer address.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// rateLimiter is a sliding-window counter per key (login attempts per IP).
type rateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
	now    func() time.Time
}

func newRateLimiter(limit int, window time.Duration, now func() time.Time) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, hits: map[string][]time.Time{}, now: now}
}

// allow records an attempt for key and reports whether it is within the limit.
func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cut := now.Add(-l.window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}
