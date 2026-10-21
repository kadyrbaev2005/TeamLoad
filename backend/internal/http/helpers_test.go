package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"teamload/internal/ai"
	"teamload/internal/config"
	"teamload/internal/store"
)

// Integration tests run against a real Postgres (TEST_DATABASE_URL, see
// `make db-test`). They are skipped when it is not set.

var testStore *store.Store

func TestMain(m *testing.M) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		fmt.Println("TEST_DATABASE_URL not set: skipping API integration tests")
		os.Exit(0)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	st, err := store.Open(ctx, url)
	if err != nil {
		fmt.Println("open test database:", err)
		os.Exit(1)
	}
	if err := st.Migrate(ctx); err != nil {
		fmt.Println("migrate:", err)
		os.Exit(1)
	}
	testStore = st
	code := m.Run()
	st.Close()
	os.Exit(code)
}

// Fixed clock: 22 Oct 2026, 10:00 in Almaty.
var (
	almaty, _ = time.LoadLocation("Asia/Almaty")
	testNow   = time.Date(2026, 10, 22, 10, 0, 0, 0, almaty)
)

type env struct {
	t   *testing.T
	srv *httptest.Server
	api *Server
}

type envOpt func(*config.Config, *[]Option)

func withProvider(p ai.Provider) envOpt {
	return func(_ *config.Config, o *[]Option) { *o = append(*o, WithProvider(p)) }
}

func withConfig(f func(*config.Config)) envOpt {
	return func(c *config.Config, _ *[]Option) { f(c) }
}

func newEnv(t *testing.T, opts ...envOpt) *env {
	t.Helper()
	_, err := testStore.Pool.Exec(context.Background(),
		`TRUNCATE ai_calls, tasks, raci_cells, raci_activities, role_assignments, plan_seats, project_members, projects, profiles, users CASCADE`)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		AppURL: "http://localhost:8080", Location: almaty, JWTSecret: []byte("test-secret"),
		LLMProvider: "mock", LLMMaxCallsPerDay: 30, LLMTimeout: 5 * time.Second, LoginAttemptsPer15m: 10,
	}
	serverOpts := []Option{WithClock(func() time.Time { return testNow })}
	for _, o := range opts {
		o(&cfg, &serverOpts)
	}
	s, err := NewServer(cfg, testStore, serverOpts...)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, api: s}
}

type client struct {
	e    *env
	http *http.Client
	ID   string
	Name string
}

type resp struct {
	Status int
	Body   map[string]any
	Raw    string
}

func (r resp) errMessage() string {
	if e, ok := r.Body["error"].(map[string]any); ok {
		m, _ := e["message"].(string)
		return m
	}
	return ""
}

func (r resp) errField(f string) string {
	if e, ok := r.Body["error"].(map[string]any); ok {
		if fs, ok := e["fields"].(map[string]any); ok {
			v, _ := fs[f].(string)
			return v
		}
	}
	return ""
}

func (e *env) anon() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, http: &http.Client{Jar: jar}}
}

func (c *client) do(method, path string, body any) resp {
	c.e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+"/api"+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := resp{Status: res.StatusCode, Raw: string(raw)}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.Body)
	}
	return out
}

func (c *client) must(status int, method, path string, body any) resp {
	c.e.t.Helper()
	r := c.do(method, path, body)
	if r.Status != status {
		c.e.t.Fatalf("%s %s: status %d, want %d: %s", method, path, r.Status, status, r.Raw)
	}
	return r
}

var userSeq int

// user registers a new account and returns its logged-in client.
func (e *env) user(name string) *client {
	e.t.Helper()
	userSeq++
	c := e.anon()
	email := fmt.Sprintf("%s.%d@stu.sdu.edu.kz", strings.ToLower(strings.Fields(name)[0]), userSeq)
	r := c.must(201, "POST", "/auth/register", map[string]any{"full_name": name, "email": email, "password": "studyspace26", "consent": true})
	c.ID = r.Body["user"].(map[string]any)["id"].(string)
	c.Name = name
	return c
}

const studyDesc = "A web app for SDU students to find and book free group-study rooms in the library. Students see a map of rooms with live availability, book a 1–2 hour slot and get a QR code to check in at the door. Librarians see occupancy statistics to plan opening hours."

func projectBody(teamSize int) map[string]any {
	return map[string]any{"title": "StudySpace", "description": studyDesc, "deadline": "2026-12-11", "team_size": teamSize, "context": "PjM 101 · Fall 2026"}
}

// project creates a project owned by c and returns its id and invite code.
func (c *client) project(teamSize int) (id, code string) {
	c.e.t.Helper()
	r := c.must(201, "POST", "/projects", projectBody(teamSize))
	p := r.Body["project"].(map[string]any)
	return p["id"].(string), p["invite_code"].(string)
}

func (c *client) join(code string) resp { return c.do("POST", "/invites/"+code+"/join", nil) }

func (c *client) saveProfile(strengths, weaknesses string, tags ...string) {
	c.e.t.Helper()
	if tags == nil {
		tags = []string{}
	}
	c.must(200, "PUT", "/me/profile", map[string]any{"strengths": strengths, "weaknesses": weaknesses, "tags": tags})
}

// Profiles from the design.
var designProfiles = map[string][3]string{
	"Aruzhan Sadykova":    {"I organise group work well and keep deadlines. I like talking to the instructor and presenting. I am careful with details and usually find bugs in my friends' code.", "Not strong at writing backend code.", "Planning,Communication,Testing,Notion"},
	"Daniyar Kenzhebekov": {"Go and PostgreSQL. I built a REST API during my internship and I like algorithms and clean data models.", "I don't enjoy CSS or visual design.", "Go,SQL,Docker,REST"},
	"Madina Tulegenova":   {"Figma and UI design. I ran user interviews for my HCI course and like making things easy to use.", "Little coding beyond HTML and CSS.", "Figma,User research,CSS"},
	"Timur Akhmetov":      {"React and TypeScript. I made two front-end pet projects and I care about fast, responsive pages.", "I hate writing documentation.", "React,TypeScript,Tailwind"},
	"Dana Bekova":         {"Python, pandas and charts from my data analysis course. I'm curious about how APIs work.", "I get nervous when presenting.", "Python,SQL,Matplotlib"},
}

func (c *client) designProfile() {
	c.e.t.Helper()
	p := designProfiles[c.Name]
	c.saveProfile(p[0], p[1], strings.Split(p[2], ",")...)
}

func obj(v any) map[string]any { m, _ := v.(map[string]any); return m }

func list(v any) []any { l, _ := v.([]any); return l }
