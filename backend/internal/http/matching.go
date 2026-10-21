package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"teamload/internal/ai"
	"teamload/internal/domain"
	"teamload/internal/store"
	"teamload/internal/store/db"
)

type matchRowJSON struct {
	SeatID         string  `json:"seat_id"`
	UserID         *string `json:"user_id"`
	Fit            *int32  `json:"fit"`
	RunnerUpSeatID *string `json:"runner_up_seat_id"`
	Explanation    string  `json:"explanation"`
	State          string  `json:"state"`
	Source         string  `json:"source"`
	Outdated       bool    `json:"outdated"`
}

type gapJSON struct {
	Skill  string `json:"skill"`
	SeatID string `json:"seat_id"`
	Reason string `json:"reason"`
}

type readinessJSON struct {
	PlanApproved    bool     `json:"plan_approved"`
	MemberCount     int      `json:"member_count"`
	TeamSize        int32    `json:"team_size"`
	MissingProfiles []string `json:"missing_profiles"`
	CanRun          bool     `json:"can_run"`
}

type matchingJSON struct {
	Ran        bool           `json:"ran"`
	RanAt      *time.Time     `json:"ran_at"`
	Rows       []matchRowJSON `json:"rows"`
	Gaps       []gapJSON      `json:"gaps"`
	GrowthNote *string        `json:"growth_note"`
	Readiness  readinessJSON  `json:"readiness"`
}

func uuidStr(u *uuid.UUID) *string {
	if u == nil {
		return nil
	}
	s := u.String()
	return &s
}

// readiness implements the "Run AI matching" conditions (spec 6.6).
func (s *Server) readiness(ctx context.Context, p db.Project) (readinessJSON, error) {
	rows, err := s.st.ListMembers(ctx, []uuid.UUID{p.ID})
	if err != nil {
		return readinessJSON{}, err
	}
	r := readinessJSON{PlanApproved: p.PlanStatus == "approved", MemberCount: len(rows), TeamSize: p.TeamSize, MissingProfiles: []string{}}
	for _, m := range rows {
		if !m.HasProfile {
			r.MissingProfiles = append(r.MissingProfiles, m.FullName)
		}
	}
	r.CanRun = r.PlanApproved && r.MemberCount >= domain.MinTeamSize && len(r.MissingProfiles) == 0
	return r, nil
}

func (s *Server) matchingPayload(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	p, err := s.st.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	seats, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	asg, err := s.st.ListAssignments(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	bySeat := map[uuid.UUID]db.RoleAssignment{}
	for _, a := range asg {
		bySeat[a.SeatID] = a
	}
	out := matchingJSON{Ran: p.MatchingRanAt != nil || len(asg) > 0, RanAt: p.MatchingRanAt,
		Rows: make([]matchRowJSON, 0, len(seats)), Gaps: []gapJSON{}, GrowthNote: p.MatchingGrowthNote}
	for _, st := range seats {
		row := matchRowJSON{SeatID: st.ID.String(), State: "suggested", Source: "ai"}
		if a, ok := bySeat[st.ID]; ok {
			row = matchRowJSON{SeatID: st.ID.String(), UserID: uuidStr(a.UserID), Fit: a.Fit,
				RunnerUpSeatID: uuidStr(a.RunnerUpSeatID), Explanation: a.Explanation, State: a.State,
				Source: a.Source, Outdated: a.Outdated}
		}
		out.Rows = append(out.Rows, row)
	}
	if len(p.MatchingGaps) > 0 {
		if err := json.Unmarshal(p.MatchingGaps, &out.Gaps); err != nil {
			return nil, err
		}
	}
	if out.Readiness, err = s.readiness(ctx, p); err != nil {
		return nil, err
	}
	return map[string]any{"matching": out}, nil
}

func (s *Server) writeMatching(w http.ResponseWriter, r *http.Request, projectID uuid.UUID) error {
	b, err := s.matchingPayload(r.Context(), projectID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, b)
	return nil
}

func (s *Server) handleGetMatching(w http.ResponseWriter, r *http.Request) error {
	return s.writeMatching(w, r, projectFrom(r.Context()).ID)
}

func seatBriefs(seats []db.PlanSeat) ([]ai.SeatBrief, map[string]uuid.UUID) {
	briefs := make([]ai.SeatBrief, len(seats))
	ids := map[string]uuid.UUID{}
	for i, st := range seats {
		label := fmt.Sprintf("S%d", i+1)
		ids[label] = st.ID
		briefs[i] = ai.SeatBrief{ID: label, Name: st.Name, Summary: st.Summary, Responsibilities: st.Responsibilities,
			Skills: st.Skills, EffortShare: int(st.EffortShare)}
	}
	return briefs, ids
}

func (s *Server) handleRunMatching(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	p := projectFrom(ctx)
	ready, err := s.readiness(ctx, p)
	if err != nil {
		return err
	}
	switch {
	case !ready.PlanApproved:
		return errConflict("not_ready", "Approve the role plan first.")
	case ready.MemberCount < domain.MinTeamSize:
		return errConflict("not_ready", fmt.Sprintf("Matching needs at least %d members.", domain.MinTeamSize))
	case len(ready.MissingProfiles) > 0:
		return errConflict("not_ready", "Waiting for profiles: "+strings.Join(ready.MissingProfiles, ", ")+".")
	}
	seats, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return err
	}
	briefs, seatIDs := seatBriefs(seats)
	memberRows, err := s.st.ListMembers(ctx, []uuid.UUID{p.ID})
	if err != nil {
		return err
	}
	members := make([]ai.Member, 0, len(memberRows))
	for _, m := range memberRows {
		prof, err := s.st.GetProfile(ctx, m.ID)
		if err != nil {
			return err
		}
		members = append(members, ai.Member{UserID: m.ID, Strengths: prof.Strengths, Weaknesses: prof.Weaknesses, Tags: prof.Tags})
	}
	res, err := s.ai.Match(ctx, p.ID, p.Title, briefs, members)
	if err != nil {
		return aiErr(err)
	}
	gaps := make([]gapJSON, 0, len(res.Gaps))
	for _, g := range res.Gaps {
		gaps = append(gaps, gapJSON{Skill: strings.TrimSpace(g.Skill), SeatID: seatIDs[g.SeatID].String(), Reason: strings.TrimSpace(g.Reason)})
	}
	gapsJSON, err := json.Marshal(gaps)
	if err != nil {
		return err
	}
	version := ai.PromptVersion(ai.KindMatching)
	source := "ai"
	err = s.st.Tx(ctx, func(q *db.Queries) error {
		if _, err := q.GetProjectForUpdate(ctx, p.ID); err != nil {
			return err
		}
		if err := q.DeleteAssignments(ctx, p.ID); err != nil {
			return err
		}
		for _, a := range res.Assignments {
			uid := a.UserID
			fit := int32(a.Fit)
			var runner *uuid.UUID
			if id, ok := seatIDs[a.RunnerUpSeatID]; ok {
				runner = &id
			}
			if err := q.InsertAssignment(ctx, db.InsertAssignmentParams{
				ProjectID: p.ID, SeatID: seatIDs[a.SeatID], UserID: &uid, Fit: &fit, RunnerUpSeatID: runner,
				Explanation: a.Explanation, State: "suggested", Source: "ai", PromptVersion: &version,
			}); err != nil {
				return err
			}
		}
		return q.SetMatchingMeta(ctx, db.SetMatchingMetaParams{ID: p.ID, MatchingSource: &source,
			MatchingPromptVersion: &version, MatchingGaps: gapsJSON, MatchingGrowthNote: res.GrowthNote})
	})
	if err != nil {
		return err
	}
	return s.writeMatching(w, r, p.ID)
}

func (s *Server) handleAcceptMatching(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		SeatID string `json:"seat_id"`
	}
	if r.ContentLength != 0 {
		if err := decode(r, &in); err != nil {
			return err
		}
	}
	ctx := r.Context()
	p := projectFrom(ctx)
	if in.SeatID == "" {
		if _, err := s.st.AcceptAllAssignments(ctx, p.ID); err != nil {
			return err
		}
		return s.writeMatching(w, r, p.ID)
	}
	seatID, err := uuid.Parse(in.SeatID)
	if err != nil {
		return errNotFound("Role not found.")
	}
	n, err := s.st.AcceptAssignment(ctx, db.AcceptAssignmentParams{ProjectID: p.ID, SeatID: seatID})
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound("There is no suggestion to accept for this role.")
	}
	return s.writeMatching(w, r, p.ID)
}

// handleAssignSeat sets a person on a seat. If that person held another
// seat, the two people swap; both rows become manual and accepted.
func (s *Server) handleAssignSeat(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		UserID string `json:"user_id"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	ctx := r.Context()
	p := projectFrom(ctx)
	seatID, err := uuid.Parse(chi.URLParam(r, "seatID"))
	if err != nil {
		return errNotFound("Role not found.")
	}
	userID, err := uuid.Parse(in.UserID)
	if err != nil {
		return errField("user_id", "Choose a team member.")
	}
	seats, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return err
	}
	found := false
	for _, st := range seats {
		found = found || st.ID == seatID
	}
	if !found {
		return errNotFound("Role not found.")
	}
	if ok, err := s.st.IsMember(ctx, db.IsMemberParams{ProjectID: p.ID, UserID: userID}); err != nil {
		return err
	} else if !ok {
		return errField("user_id", "Choose a team member.")
	}
	err = s.st.Tx(ctx, func(q *db.Queries) error {
		if _, err := q.GetProjectForUpdate(ctx, p.ID); err != nil {
			return err
		}
		var prevUser *uuid.UUID
		cur, err := q.GetAssignmentBySeat(ctx, db.GetAssignmentBySeatParams{ProjectID: p.ID, SeatID: seatID})
		switch {
		case err == nil:
			prevUser = cur.UserID
		case !store.IsNotFound(err):
			return err
		}
		other, err := q.GetAssignmentByUser(ctx, db.GetAssignmentByUserParams{ProjectID: p.ID, UserID: &userID})
		swap := err == nil && other.SeatID != seatID
		if err != nil && !store.IsNotFound(err) {
			return err
		}
		if swap {
			if err := q.ClearAssignmentUser(ctx, db.ClearAssignmentUserParams{ProjectID: p.ID, SeatID: other.SeatID}); err != nil {
				return err
			}
		}
		if err := q.SetManualAssignment(ctx, db.SetManualAssignmentParams{ProjectID: p.ID, SeatID: seatID, UserID: &userID}); err != nil {
			return err
		}
		if swap {
			return q.SetManualAssignment(ctx, db.SetManualAssignmentParams{ProjectID: p.ID, SeatID: other.SeatID, UserID: prevUser})
		}
		return nil
	})
	if err != nil {
		return err
	}
	return s.writeMatching(w, r, p.ID)
}
