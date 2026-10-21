package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"teamload/internal/ai"
	"teamload/internal/domain"
	"teamload/internal/store/db"
)

const maxRaciActivities = 30

type raciActivityJSON struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Cells map[string]string `json:"cells"`
}

type raciJSON struct {
	Activities  []raciActivityJSON `json:"activities"`
	Source      *string            `json:"source"`
	GeneratedAt *time.Time         `json:"generated_at"`
}

func (s *Server) raciPayload(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	p, err := s.st.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	acts, err := s.st.ListRaciActivities(ctx, projectID)
	if err != nil {
		return nil, err
	}
	cells, err := s.st.ListRaciCells(ctx, projectID)
	if err != nil {
		return nil, err
	}
	byAct := map[uuid.UUID]map[string]string{}
	for _, c := range cells {
		if byAct[c.ActivityID] == nil {
			byAct[c.ActivityID] = map[string]string{}
		}
		byAct[c.ActivityID][c.SeatID.String()] = c.Value
	}
	out := raciJSON{Activities: make([]raciActivityJSON, len(acts)), Source: p.RaciSource, GeneratedAt: p.RaciGeneratedAt}
	for i, a := range acts {
		c := byAct[a.ID]
		if c == nil {
			c = map[string]string{}
		}
		out.Activities[i] = raciActivityJSON{ID: a.ID.String(), Name: a.Name, Cells: c}
	}
	return map[string]any{"raci": out}, nil
}

func (s *Server) writeRaci(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) error {
	b, err := s.raciPayload(r.Context(), projectID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, b)
	return nil
}

func (s *Server) handleGetRaci(w http.ResponseWriter, r *http.Request) error {
	return s.writeRaci(w, r, projectFrom(r.Context()).ID)
}

type raciRowInput struct {
	ID    *uuid.UUID
	Name  string
	Cells map[uuid.UUID]string
}

// insertRaciCells writes the non-empty cells of one activity.
func insertRaciCells(ctx context.Context, q *db.Queries, activityID uuid.UUID, cells map[uuid.UUID]string) error {
	for seat, v := range cells {
		if v == "" {
			continue
		}
		if err := q.InsertRaciCell(ctx, db.InsertRaciCellParams{ActivityID: activityID, SeatID: seat, Value: v}); err != nil {
			return err
		}
	}
	return nil
}

// handlePutRaci saves the whole matrix. Rows keep their IDs so tasks stay
// linked; rows without an ID are added, missing rows are deleted. Rows
// without exactly one A are allowed while editing (spec 6.7).
func (s *Server) handlePutRaci(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Activities []struct {
			ID    string            `json:"id"`
			Name  string            `json:"name"`
			Cells map[string]string `json:"cells"`
		} `json:"activities"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	p := projectFrom(ctx)
	if len(in.Activities) > maxRaciActivities {
		return errBadRequest(fmt.Sprintf("Use at most %d activities.", maxRaciActivities))
	}
	seats, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return err
	}
	seatOK := map[uuid.UUID]bool{}
	for _, st := range seats {
		seatOK[st.ID] = true
	}
	existing, err := s.st.ListRaciActivities(ctx, p.ID)
	if err != nil {
		return err
	}
	actOK := map[uuid.UUID]bool{}
	for _, a := range existing {
		actOK[a.ID] = true
	}
	rows := make([]raciRowInput, len(in.Activities))
	for i, a := range in.Activities {
		name := strings.TrimSpace(a.Name)
		if n := len([]rune(name)); n < 1 || n > domain.MaxActivityName {
			return errField("name", domain.MsgActivityName)
		}
		row := raciRowInput{Name: name, Cells: map[uuid.UUID]string{}}
		if a.ID != "" {
			id, err := uuid.Parse(a.ID)
			if err != nil || !actOK[id] {
				return errNotFound("Activity not found.")
			}
			row.ID = &id
		}
		for k, v := range a.Cells {
			sid, err := uuid.Parse(k)
			if err != nil || !seatOK[sid] {
				return errBadRequest("Unknown role in the RACI matrix.")
			}
			if !domain.ValidRaciValue(v) {
				return errField("cells", domain.MsgRaciValue)
			}
			row.Cells[sid] = v
		}
		rows[i] = row
	}
	err = s.st.Tx(ctx, func(q *db.Queries) error {
		keep := map[uuid.UUID]bool{}
		for i, row := range rows {
			pos := int32(i + 1)
			var id uuid.UUID
			if row.ID != nil {
				id = *row.ID
				if _, err := q.UpdateRaciActivity(ctx, db.UpdateRaciActivityParams{ID: id, ProjectID: p.ID, Position: pos, Name: row.Name}); err != nil {
					return err
				}
				if err := q.DeleteRaciCells(ctx, id); err != nil {
					return err
				}
			} else {
				a, err := q.InsertRaciActivity(ctx, db.InsertRaciActivityParams{ProjectID: p.ID, Position: pos, Name: row.Name})
				if err != nil {
					return err
				}
				id = a.ID
			}
			keep[id] = true
			if err := insertRaciCells(ctx, q, id, row.Cells); err != nil {
				return err
			}
		}
		for _, a := range existing {
			if !keep[a.ID] {
				if err := q.DeleteRaciActivity(ctx, db.DeleteRaciActivityParams{ID: a.ID, ProjectID: p.ID}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.writeRaci(w, r, p.ID)
}

func (s *Server) handleGenerateRaci(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	p := projectFrom(ctx)
	if p.PlanStatus != "approved" {
		return errConflict("not_ready", "Approve the role plan first.")
	}
	seats, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return err
	}
	briefs, seatIDs := seatBriefs(seats)
	out, err := s.ai.RACI(ctx, p.ID, ai.RACIInput{
		Title: p.Title, Description: p.Description, Context: p.Context, Deadline: p.Deadline,
		WeeksLeft: domain.WeeksLeft(p.Deadline, s.today()), Seats: briefs,
	})
	if err != nil {
		return aiErr(err)
	}
	version := ai.PromptVersion(ai.KindRACI)
	source := "ai"
	err = s.st.Tx(ctx, func(q *db.Queries) error {
		if _, err := q.GetProjectForUpdate(ctx, p.ID); err != nil {
			return err
		}
		if err := q.DeleteRaciActivities(ctx, p.ID); err != nil {
			return err
		}
		for i, a := range out.Activities {
			act, err := q.InsertRaciActivity(ctx, db.InsertRaciActivityParams{ProjectID: p.ID, Position: int32(i + 1), Name: strings.TrimSpace(a.Name)})
			if err != nil {
				return err
			}
			cells := map[uuid.UUID]string{}
			for label, v := range a.Cells {
				if id, ok := seatIDs[label]; ok {
					cells[id] = v
				}
			}
			if err := insertRaciCells(ctx, q, act.ID, cells); err != nil {
				return err
			}
		}
		return q.SetRaciMeta(ctx, db.SetRaciMetaParams{ID: p.ID, RaciSource: &source, RaciPromptVersion: &version})
	})
	if err != nil {
		return err
	}
	return s.writeRaci(w, r, p.ID)
}
