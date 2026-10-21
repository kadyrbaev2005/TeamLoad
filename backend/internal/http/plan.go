package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"teamload/internal/ai"
	"teamload/internal/domain"
	"teamload/internal/store/db"
)

// aiRecorder stores AI calls for the daily spending guard (spec 7.2).
type aiRecorder struct {
	s *Server
}

func (r aiRecorder) RecordCall(ctx context.Context, c ai.Call) error {
	return r.s.st.InsertAICall(ctx, db.InsertAICallParams{
		ProjectID: c.ProjectID, Kind: c.Kind, Provider: c.Provider, PromptVersion: c.PromptVersion,
		Attempt: int32(c.Attempt), Ok: c.OK, LatencyMs: int32(c.Latency.Milliseconds()),
	})
}

func (r aiRecorder) CallsToday(ctx context.Context, projectID uuid.UUID) (int, error) {
	t := r.s.today()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, r.s.cfg.Location)
	n, err := r.s.st.CountAICallsSince(ctx, db.CountAICallsSinceParams{ProjectID: projectID, CreatedAt: start})
	return int(n), err
}

// aiErr maps AI layer errors to API errors.
func aiErr(err error) error {
	switch {
	case errors.Is(err, ai.ErrUnavailable):
		return &apiError{Status: http.StatusBadGateway, Code: "ai_unavailable", Message: domain.MsgAIUnavailable}
	case errors.Is(err, ai.ErrLimit):
		return &apiError{Status: http.StatusTooManyRequests, Code: "ai_limit", Message: domain.MsgAILimit}
	}
	return err
}

type seatJSON struct {
	ID               string   `json:"id"`
	Position         int32    `json:"position"`
	Name             string   `json:"name"`
	Summary          string   `json:"summary"`
	Responsibilities []string `json:"responsibilities"`
	Skills           []string `json:"skills"`
	EffortShare      int32    `json:"effort_share"`
	Why              string   `json:"why"`
	Color            string   `json:"color"`
}

func toSeatJSON(s db.PlanSeat) seatJSON {
	return seatJSON{ID: s.ID.String(), Position: s.Position, Name: s.Name, Summary: s.Summary,
		Responsibilities: s.Responsibilities, Skills: s.Skills, EffortShare: s.EffortShare, Why: s.Why, Color: s.ColorToken}
}

type planJSON struct {
	Status      string     `json:"status"`
	Source      *string    `json:"source"`
	GeneratedAt *time.Time `json:"generated_at"`
	Seats       []seatJSON `json:"seats"`
	HasMatching bool       `json:"has_matching"`
	HasRaci     bool       `json:"has_raci"`
}

func (s *Server) planPayload(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	p, err := s.st.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	seats, err := s.st.ListSeats(ctx, projectID)
	if err != nil {
		return nil, err
	}
	nAsg, err := s.st.CountAssignments(ctx, projectID)
	if err != nil {
		return nil, err
	}
	acts, err := s.st.ListRaciActivities(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := planJSON{Status: p.PlanStatus, Source: p.PlanSource, GeneratedAt: p.PlanGeneratedAt,
		Seats: make([]seatJSON, len(seats)), HasMatching: nAsg > 0, HasRaci: len(acts) > 0}
	for i, st := range seats {
		out.Seats[i] = toSeatJSON(st)
	}
	return map[string]any{"plan": out}, nil
}

func (s *Server) writePlan(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) error {
	b, err := s.planPayload(r.Context(), projectID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, b)
	return nil
}

func (s *Server) handleGetPlan(w http.ResponseWriter, r *http.Request) error {
	return s.writePlan(w, r, projectFrom(r.Context()).ID)
}

func (s *Server) handlePutPlan(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Seats []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Summary     string `json:"summary"`
			EffortShare int    `json:"effort_share"`
		} `json:"seats"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	p := projectFrom(ctx)
	existing, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return err
	}
	known := map[uuid.UUID]bool{}
	for _, st := range existing {
		known[st.ID] = true
	}
	edits := make([]domain.SeatEdit, len(in.Seats))
	ids := make([]uuid.UUID, len(in.Seats))
	seen := map[uuid.UUID]bool{}
	for i, st := range in.Seats {
		id, err := uuid.Parse(st.ID)
		if err != nil || !known[id] || seen[id] {
			return errField("seats", domain.MsgSeatCount)
		}
		seen[id] = true
		ids[i] = id
		edits[i] = domain.SeatEdit{Name: strings.TrimSpace(st.Name), Summary: strings.TrimSpace(st.Summary), EffortShare: st.EffortShare}
	}
	if e := domain.ValidateSeatEdits(edits, len(existing)); !e.OK() {
		return errValidation(e)
	}
	err = s.st.Tx(ctx, func(q *db.Queries) error {
		for i, e := range edits {
			if _, err := q.UpdateSeat(ctx, db.UpdateSeatParams{ID: ids[i], ProjectID: p.ID, Name: e.Name,
				Summary: e.Summary, EffortShare: int32(e.EffortShare)}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.writePlan(w, r, p.ID)
}

// replacePlan swaps in new seats as a draft and clears matching and RACI.
func (s *Server) replacePlan(ctx context.Context, projectID uuid.UUID, out ai.RolePlanOutput, source string, version *string) error {
	return s.st.Tx(ctx, func(q *db.Queries) error {
		if _, err := q.GetProjectForUpdate(ctx, projectID); err != nil {
			return err
		}
		if err := q.DeleteRaciActivities(ctx, projectID); err != nil {
			return err
		}
		if err := q.DeleteSeats(ctx, projectID); err != nil { // cascades to assignments and RACI cells
			return err
		}
		for i, st := range out.Seats {
			pos := i + 1
			if _, err := q.InsertSeat(ctx, db.InsertSeatParams{
				ProjectID: projectID, Position: int32(pos), Name: strings.TrimSpace(st.Name), Summary: strings.TrimSpace(st.Summary),
				Responsibilities: st.Responsibilities, Skills: st.Skills, EffortShare: int32(st.EffortShare),
				Why: strings.TrimSpace(st.Why), ColorToken: domain.SeatColor(pos),
			}); err != nil {
				return err
			}
		}
		if err := q.SetPlanGenerated(ctx, db.SetPlanGeneratedParams{ID: projectID, PlanSource: &source, PlanPromptVersion: version}); err != nil {
			return err
		}
		if err := q.ClearMatchingMeta(ctx, projectID); err != nil {
			return err
		}
		return q.ClearRaciMeta(ctx, projectID)
	})
}

func (s *Server) handleGeneratePlan(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Note string `json:"note"`
	}
	if r.ContentLength != 0 {
		if err := decode(r, &in); err != nil {
			return err
		}
	}
	note := strings.TrimSpace(in.Note)
	if len([]rune(note)) > domain.MaxRegenNote {
		return errField("note", domain.MsgNoteTooLong)
	}
	ctx := r.Context()
	p := projectFrom(ctx)
	today := s.today()
	out, err := s.ai.RolePlan(ctx, p.ID, ai.RolePlanInput{
		Title: p.Title, Description: p.Description, Context: p.Context, Note: note, Deadline: p.Deadline,
		WeeksLeft: domain.WeeksLeft(p.Deadline, today), TeamSize: int(p.TeamSize),
	})
	if err != nil {
		return aiErr(err)
	}
	version := ai.PromptVersion(ai.KindRolePlan)
	if err := s.replacePlan(ctx, p.ID, out, "ai", &version); err != nil {
		return err
	}
	return s.writePlan(w, r, p.ID)
}

func (s *Server) handleStandardPlan(w http.ResponseWriter, r *http.Request) error {
	p := projectFrom(r.Context())
	if err := s.replacePlan(r.Context(), p.ID, ai.StandardPlan(int(p.TeamSize)), "standard", nil); err != nil {
		return err
	}
	return s.writePlan(w, r, p.ID)
}

func (s *Server) handleApprovePlan(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	p := projectFrom(ctx)
	seats, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return err
	}
	if len(seats) == 0 {
		return errBadRequest("Create the role plan first.")
	}
	sum := 0
	for _, st := range seats {
		sum += int(st.EffortShare)
	}
	if sum != 100 {
		return errField("total", domain.MsgShareTotal)
	}
	if err := s.st.SetPlanStatus(ctx, db.SetPlanStatusParams{ID: p.ID, PlanStatus: "approved"}); err != nil {
		return err
	}
	return s.writePlan(w, r, p.ID)
}
