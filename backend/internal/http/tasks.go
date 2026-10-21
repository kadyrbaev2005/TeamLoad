package http

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"teamload/internal/domain"
	"teamload/internal/store"
	"teamload/internal/store/db"
)

type taskJSON struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	AssigneeID     *string   `json:"assignee_id"`
	Estimate       float64   `json:"estimate"`
	Status         string    `json:"status"`
	RaciActivityID *string   `json:"raci_activity_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func toTaskJSON(t db.Task) taskJSON {
	return taskJSON{ID: t.ID.String(), Title: t.Title, AssigneeID: uuidStr(t.AssigneeID), Estimate: t.Estimate,
		Status: t.Status, RaciActivityID: uuidStr(t.RaciActivityID), CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) error {
	tasks, err := s.st.ListTasks(r.Context(), projectFrom(r.Context()).ID)
	if err != nil {
		return err
	}
	out := make([]taskJSON, len(tasks))
	for i, t := range tasks {
		out[i] = toTaskJSON(t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": out})
	return nil
}

// taskFields is a validated task; nil pointers mean "unassigned" / "none".
type taskFields struct {
	Title    string
	Assignee *uuid.UUID
	Estimate float64
	Status   string
	Activity *uuid.UUID
}

// optionalID parses a JSON string-or-null ID.
func optionalID(raw json.RawMessage) (*uuid.UUID, bool) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, true
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return nil, false
	}
	if s == "" {
		return nil, true
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, false
	}
	return &id, true
}

// applyTaskPatch overlays the JSON fields present in raw onto f and validates.
func (s *Server) applyTaskPatch(ctx context.Context, p db.Project, f *taskFields, raw map[string]json.RawMessage) error {
	e := domain.FieldErrors{}
	if v, ok := raw["title"]; ok {
		var t string
		_ = json.Unmarshal(v, &t)
		f.Title = strings.TrimSpace(t)
	}
	if msg := domain.ValidateTaskTitle(f.Title); msg != "" {
		e["title"] = msg
	}
	if v, ok := raw["estimate"]; ok {
		var h *float64
		if json.Unmarshal(v, &h) != nil {
			h = nil
		}
		if h != nil && (math.IsNaN(*h) || math.IsInf(*h, 0)) {
			h = nil
		}
		est, msg := domain.ValidateEstimate(h)
		if msg != "" {
			e["estimate"] = msg
		}
		f.Estimate = est
	}
	if v, ok := raw["status"]; ok {
		_ = json.Unmarshal(v, &f.Status)
	}
	if !domain.ValidStatus(f.Status) {
		e["status"] = domain.MsgTaskStatus
	}
	if v, ok := raw["assignee_id"]; ok {
		id, valid := optionalID(v)
		if valid && id != nil {
			member, err := s.st.IsMember(ctx, db.IsMemberParams{ProjectID: p.ID, UserID: *id})
			if err != nil {
				return err
			}
			valid = member
		}
		if !valid {
			e["assignee_id"] = "Choose a team member or Unassigned."
		}
		f.Assignee = id
	}
	if v, ok := raw["raci_activity_id"]; ok {
		id, valid := optionalID(v)
		if valid && id != nil {
			found, err := s.st.RaciActivityInProject(ctx, db.RaciActivityInProjectParams{ID: *id, ProjectID: p.ID})
			if err != nil {
				return err
			}
			valid = found
		}
		if !valid {
			e["raci_activity_id"] = "Choose an activity from this project's RACI."
		}
		f.Activity = id
	}
	if !e.OK() {
		return errValidation(e)
	}
	return nil
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) error {
	var raw map[string]json.RawMessage
	if err := decode(r, &raw); err != nil {
		return err
	}
	if _, ok := raw["estimate"]; !ok {
		raw["estimate"] = json.RawMessage("null")
	}
	ctx := r.Context()
	p := projectFrom(ctx)
	f := taskFields{Status: domain.StatusTodo}
	if err := s.applyTaskPatch(ctx, p, &f, raw); err != nil {
		return err
	}
	t, err := s.st.CreateTask(ctx, db.CreateTaskParams{ProjectID: p.ID, Title: f.Title, AssigneeID: f.Assignee,
		Estimate: f.Estimate, Status: f.Status, RaciActivityID: f.Activity})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, map[string]any{"task": toTaskJSON(t)})
	return nil
}

func (s *Server) taskFromURL(r *http.Request) (db.Task, error) {
	id, err := uuid.Parse(chi.URLParam(r, "taskID"))
	if err != nil {
		return db.Task{}, errNotFound("Task not found.")
	}
	t, err := s.st.GetTask(r.Context(), db.GetTaskParams{ID: id, ProjectID: projectFrom(r.Context()).ID})
	if store.IsNotFound(err) {
		return db.Task{}, errNotFound("Task not found.")
	}
	return t, err
}

func (s *Server) handlePatchTask(w http.ResponseWriter, r *http.Request) error {
	t, err := s.taskFromURL(r)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := decode(r, &raw); err != nil {
		return err
	}
	ctx := r.Context()
	f := taskFields{Title: t.Title, Assignee: t.AssigneeID, Estimate: t.Estimate, Status: t.Status, Activity: t.RaciActivityID}
	if err := s.applyTaskPatch(ctx, projectFrom(ctx), &f, raw); err != nil {
		return err
	}
	t, err = s.st.UpdateTask(ctx, db.UpdateTaskParams{ID: t.ID, ProjectID: t.ProjectID, Title: f.Title,
		AssigneeID: f.Assignee, Estimate: f.Estimate, Status: f.Status, RaciActivityID: f.Activity})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": toTaskJSON(t)})
	return nil
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) error {
	t, err := s.taskFromURL(r)
	if err != nil {
		return err
	}
	if _, err := s.st.DeleteTask(r.Context(), db.DeleteTaskParams{ID: t.ID, ProjectID: t.ProjectID}); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
