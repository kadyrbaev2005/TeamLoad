package http

import (
	"math"
	"net/http"

	"github.com/google/uuid"

	"teamload/internal/domain"
)

type loadRowJSON struct {
	UserID    string  `json:"user_id"`
	Active    float64 `json:"active"`
	Done      float64 `json:"done"`
	OpenTasks int     `json:"open_tasks"`
	SharePct  int     `json:"share_pct"`
	PlanShare *int32  `json:"plan_share"`
	SeatID    *string `json:"seat_id"`
	Status    string  `json:"status"`
}

type dashboardJSON struct {
	Deadline         string        `json:"deadline"`
	DaysLeft         int           `json:"days_left"`
	TotalActive      float64       `json:"total_active"`
	Average          float64       `json:"average"`
	UnassignedActive float64       `json:"unassigned_active"`
	OpenTasks        int           `json:"open_tasks"`
	HasEstimates     bool          `json:"has_estimates"`
	Rows             []loadRowJSON `json:"rows"`
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// handleDashboard computes the workload view (spec 6.9). Active load is the
// sum of estimates in To do + Doing; the average divides by every member.
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	p := projectFrom(ctx)
	members, err := s.st.ListMembers(ctx, []uuid.UUID{p.ID})
	if err != nil {
		return err
	}
	tasks, err := s.st.ListTasks(ctx, p.ID)
	if err != nil {
		return err
	}
	seats, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return err
	}
	shareOf := map[uuid.UUID]int32{}
	for _, st := range seats {
		shareOf[st.ID] = st.EffortShare
	}

	type acc struct {
		active, done, all float64
		open              int
	}
	by := map[uuid.UUID]*acc{}
	out := dashboardJSON{Deadline: p.Deadline.Format(dateLayout), DaysLeft: domain.DaysLeft(p.Deadline, s.today()),
		HasEstimates: len(tasks) > 0, Rows: []loadRowJSON{}}
	var allHours float64
	for _, t := range tasks {
		allHours += t.Estimate
		open := t.Status != domain.StatusDone
		if open {
			out.OpenTasks++
		}
		if t.AssigneeID == nil {
			if open {
				out.UnassignedActive += t.Estimate
			}
			continue
		}
		a := by[*t.AssigneeID]
		if a == nil {
			a = &acc{}
			by[*t.AssigneeID] = a
		}
		a.all += t.Estimate
		if open {
			a.active += t.Estimate
			a.open++
		} else {
			a.done += t.Estimate
		}
	}
	actives := make([]float64, len(members))
	for i, m := range members {
		if a := by[m.ID]; a != nil {
			actives[i] = a.active
		}
	}
	out.Average = domain.TeamAverage(actives)
	for i, m := range members {
		a := by[m.ID]
		if a == nil {
			a = &acc{}
		}
		out.TotalActive += a.active
		row := loadRowJSON{UserID: m.ID.String(), Active: round1(a.active), Done: round1(a.done), OpenTasks: a.open,
			Status: domain.LoadStatus(actives[i], out.Average)}
		if allHours > 0 {
			row.SharePct = int(math.Round(a.all / allHours * 100))
		}
		if m.SeatID != nil && m.AssignmentState != nil && *m.AssignmentState == "accepted" {
			share := shareOf[*m.SeatID]
			row.PlanShare = &share
			row.SeatID = uuidStr(m.SeatID)
		}
		out.Rows = append(out.Rows, row)
	}
	out.TotalActive = round1(out.TotalActive)
	out.UnassignedActive = round1(out.UnassignedActive)
	writeJSON(w, http.StatusOK, map[string]any{"dashboard": out})
	return nil
}
