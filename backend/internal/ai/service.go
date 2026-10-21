package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrUnavailable means the provider failed twice (spec 7.2).
	ErrUnavailable = errors.New("ai: service temporarily unavailable")
	// ErrLimit means the project used its daily AI calls.
	ErrLimit = errors.New("ai: daily limit reached")
)

// ValidationError lists the rules an AI answer broke.
type ValidationError struct{ Problems []string }

func (e *ValidationError) Error() string {
	return "ai: invalid output: " + strings.Join(e.Problems, "; ")
}

// Call is one provider call, stored for the spending guard.
type Call struct {
	ProjectID     uuid.UUID
	Kind          string
	Provider      string
	PromptVersion string
	Attempt       int
	OK            bool
	Latency       time.Duration
}

// Recorder stores calls and counts today's calls per project.
type Recorder interface {
	RecordCall(ctx context.Context, c Call) error
	CallsToday(ctx context.Context, projectID uuid.UUID) (int, error)
}

// Service wraps a Provider with timeout, validation, retry-once, the daily
// spending guard and privacy rules for matching.
type Service struct {
	provider  Provider
	name      string
	rec       Recorder
	timeout   time.Duration
	maxPerDay int

	mu  sync.Mutex
	rng *rand.Rand
}

// NewService builds a Service. name is logged and stored with each call.
func NewService(p Provider, name string, rec Recorder, timeout time.Duration, maxPerDay int) *Service {
	return &Service{provider: p, name: name, rec: rec, timeout: timeout, maxPerDay: maxPerDay,
		rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))} // #nosec G404 -- shuffling, not security
}

// NewProvider returns the provider selected by LLM_PROVIDER.
func NewProvider(name, apiKey, model string, mockDelay time.Duration) (Provider, error) {
	switch name {
	case "", "mock":
		return Mock{Delay: mockDelay}, nil
	case "gemini":
		return Gemini{APIKey: apiKey, Model: model, HTTP: nil}, nil
	}
	return nil, fmt.Errorf("unknown LLM_PROVIDER %q", name)
}

// ProviderName is stored as the result source detail.
func (s *Service) ProviderName() string { return s.name }

// run performs up to two attempts; attempt 2 gets attempt 1's problems as feedback.
func (s *Service) run(ctx context.Context, projectID uuid.UUID, kind string, call func(ctx context.Context, feedback []string) error) error {
	version := PromptVersion(kind)
	var feedback []string
	for attempt := 1; attempt <= 2; attempt++ {
		n, err := s.rec.CallsToday(ctx, projectID)
		if err != nil {
			return err
		}
		if n >= s.maxPerDay {
			if attempt == 1 {
				return ErrLimit
			}
			return ErrUnavailable
		}
		cctx, cancel := context.WithTimeout(ctx, s.timeout)
		start := time.Now()
		err = call(cctx, feedback)
		cancel()
		latency := time.Since(start)
		ok := err == nil
		if rerr := s.rec.RecordCall(ctx, Call{ProjectID: projectID, Kind: kind, Provider: s.name, PromptVersion: version,
			Attempt: attempt, OK: ok, Latency: latency}); rerr != nil {
			return rerr
		}
		var ve *ValidationError
		isVE := errors.As(err, &ve)
		attrs := []any{"provider", s.name, "kind", kind, "attempt", attempt, "ok", ok, "latency_ms", latency.Milliseconds()}
		switch {
		case ok:
			slog.InfoContext(ctx, "ai call", attrs...)
			return nil
		case isVE:
			// Problems name rules and field paths only, never profile or description text.
			slog.WarnContext(ctx, "ai call", append(attrs, "result", "invalid", "problems", len(ve.Problems))...)
			feedback = ve.Problems
		default:
			slog.WarnContext(ctx, "ai call", append(attrs, "result", "error", "err", errKind(err))...)
			feedback = nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return ErrUnavailable
}

func errKind(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return err.Error()
}

func invalid(problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	return &ValidationError{Problems: problems}
}

// RolePlan generates and validates a role plan.
func (s *Service) RolePlan(ctx context.Context, projectID uuid.UUID, in RolePlanInput) (RolePlanOutput, error) {
	var out RolePlanOutput
	err := s.run(ctx, projectID, KindRolePlan, func(ctx context.Context, fb []string) error {
		in.Feedback = fb
		o, err := s.provider.GenerateRolePlan(ctx, in)
		if err != nil {
			return err
		}
		if err := invalid(ValidateRolePlan(o, in.TeamSize)); err != nil {
			return err
		}
		out = o
		return nil
	})
	return out, err
}

// Member is a profile owner as known to the server. It deliberately has no
// name, email or demographic field: only profile text and tags reach the AI.
type Member struct {
	UserID     uuid.UUID
	Strengths  string
	Weaknesses string
	Tags       []string
}

// UserAssignment is an Assignment mapped back to a user.
type UserAssignment struct {
	SeatID         string
	UserID         uuid.UUID
	Fit            int
	RunnerUpSeatID string
	Explanation    string
}

// MatchResult is the matching output mapped back to users.
type MatchResult struct {
	Assignments []UserAssignment
	Gaps        []Gap
	GrowthNote  *string
}

// Match labels members P1…Pn in a fresh random order on every provider call,
// validates the answer and maps labels back to users.
func (s *Service) Match(ctx context.Context, projectID uuid.UUID, title string, seats []SeatBrief, members []Member) (MatchResult, error) {
	var res MatchResult
	err := s.run(ctx, projectID, KindMatching, func(ctx context.Context, fb []string) error {
		in, labels := s.anonymize(title, seats, members)
		in.Feedback = fb
		o, err := s.provider.MatchRoles(ctx, in)
		if err != nil {
			return err
		}
		if err := invalid(ValidateMatch(o, in)); err != nil {
			return err
		}
		res = MatchResult{Gaps: o.Gaps, GrowthNote: o.GrowthNote}
		if res.Gaps == nil {
			res.Gaps = []Gap{}
		}
		if res.GrowthNote != nil && strings.TrimSpace(*res.GrowthNote) == "" {
			res.GrowthNote = nil
		}
		for _, a := range o.Assignments {
			res.Assignments = append(res.Assignments, UserAssignment{
				SeatID: a.SeatID, UserID: labels[a.Person], Fit: a.Fit, RunnerUpSeatID: a.RunnerUpSeatID,
				Explanation: strings.TrimSpace(a.Explanation),
			})
		}
		return nil
	})
	return res, err
}

func (s *Service) anonymize(title string, seats []SeatBrief, members []Member) (MatchInput, map[string]uuid.UUID) {
	order := make([]int, len(members))
	for i := range order {
		order[i] = i
	}
	s.mu.Lock()
	s.rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	s.mu.Unlock()
	in := MatchInput{ProjectTitle: title, Seats: seats}
	labels := map[string]uuid.UUID{}
	for k, i := range order {
		m := members[i]
		label := fmt.Sprintf("P%d", k+1)
		labels[label] = m.UserID
		tags := m.Tags
		if tags == nil {
			tags = []string{}
		}
		in.People = append(in.People, Person{Label: label, Strengths: m.Strengths, Weaknesses: m.Weaknesses, Tags: tags})
	}
	return in, labels
}

// RACI drafts and validates a RACI matrix.
func (s *Service) RACI(ctx context.Context, projectID uuid.UUID, in RACIInput) (RACIOutput, error) {
	var out RACIOutput
	ids := make([]string, len(in.Seats))
	for i, st := range in.Seats {
		ids[i] = st.ID
	}
	err := s.run(ctx, projectID, KindRACI, func(ctx context.Context, fb []string) error {
		in.Feedback = fb
		o, err := s.provider.DraftRACI(ctx, in)
		if err != nil {
			return err
		}
		if err := invalid(ValidateRACI(o, ids)); err != nil {
			return err
		}
		out = o
		return nil
	})
	return out, err
}
