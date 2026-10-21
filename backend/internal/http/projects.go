package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"teamload/internal/domain"
	"teamload/internal/store"
	"teamload/internal/store/db"
)

const dateLayout = "2006-01-02"

type memberJSON struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	FirstName     string    `json:"first_name"`
	Initials      string    `json:"initials"`
	Color         string    `json:"color"`
	IsOwner       bool      `json:"is_owner"`
	HasProfile    bool      `json:"has_profile"`
	SeatID        *string   `json:"seat_id"`
	SeatName      *string   `json:"seat_name"`
	RoleConfirmed bool      `json:"role_confirmed"`
	JoinedAt      time.Time `json:"joined_at"`
}

// membersByProject groups members per project. A member's avatar colour is
// the colour of their seat; members without a seat get one by join order.
func membersByProject(rows []db.ListMembersRow, owners map[uuid.UUID]uuid.UUID) map[uuid.UUID][]memberJSON {
	out := map[uuid.UUID][]memberJSON{}
	for _, m := range rows {
		list := out[m.ProjectID]
		color := domain.SeatColor(len(list) + 1)
		if m.SeatPosition != nil {
			color = domain.SeatColor(int(*m.SeatPosition))
		}
		mj := memberJSON{
			ID: m.ID.String(), Name: m.FullName, FirstName: domain.FirstName(m.FullName),
			Initials: domain.Initials(m.FullName), Color: color, IsOwner: owners[m.ProjectID] == m.ID,
			HasProfile: m.HasProfile, SeatName: m.SeatName, JoinedAt: m.JoinedAt,
			RoleConfirmed: m.AssignmentState != nil && *m.AssignmentState == "accepted",
		}
		if m.SeatID != nil {
			id := m.SeatID.String()
			mj.SeatID = &id
		}
		out[m.ProjectID] = append(list, mj)
	}
	return out
}

func (s *Server) projectMembers(ctx context.Context, q *db.Queries, p db.Project) ([]memberJSON, error) {
	rows, err := q.ListMembers(ctx, []uuid.UUID{p.ID})
	if err != nil {
		return nil, err
	}
	m := membersByProject(rows, map[uuid.UUID]uuid.UUID{p.ID: p.OwnerID})[p.ID]
	if m == nil {
		m = []memberJSON{}
	}
	return m, nil
}

type projectJSON struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
	TeamSize    int32  `json:"team_size"`
	Context     string `json:"context"`
	InviteCode  string `json:"invite_code"`
	InviteLink  string `json:"invite_link"`
	OwnerID     string `json:"owner_id"`
	PlanStatus  string `json:"plan_status"`
	SeatCount   int    `json:"seat_count"`
	OpenTasks   int32  `json:"open_tasks"`
}

func (s *Server) inviteLink(code string) string { return s.cfg.AppURL + "/join/" + code }

// projectBundle is the GET /projects/:id payload: project, members, is_owner.
func (s *Server) projectBundle(ctx context.Context, p db.Project, userID uuid.UUID) (map[string]any, error) {
	q := s.st.Queries
	members, err := s.projectMembers(ctx, q, p)
	if err != nil {
		return nil, err
	}
	seats, err := q.ListSeats(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	open, err := q.CountOpenTasks(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	pj := projectJSON{
		ID: p.ID.String(), Title: p.Title, Description: p.Description, Deadline: p.Deadline.Format(dateLayout),
		TeamSize: p.TeamSize, Context: p.Context, InviteCode: p.InviteCode, InviteLink: s.inviteLink(p.InviteCode),
		OwnerID: p.OwnerID.String(), PlanStatus: p.PlanStatus, SeatCount: len(seats), OpenTasks: open,
	}
	return map[string]any{"project": pj, "members": members, "is_owner": p.OwnerID == userID}, nil
}

type projectCardJSON struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Context     string       `json:"context"`
	Deadline    string       `json:"deadline"`
	TeamSize    int32        `json:"team_size"`
	IsOwner     bool         `json:"is_owner"`
	Status      string       `json:"status"`
	PlanStatus  string       `json:"plan_status"`
	MemberCount int32        `json:"member_count"`
	Members     []memberJSON `json:"members"`
	DoneHours   float64      `json:"done_hours"`
	AllHours    float64      `json:"all_hours"`
}

// cardStatus applies the chip priority from spec 6.2.
func cardStatus(c db.ListProjectCardsRow) string {
	switch {
	case c.PlanStatus != "approved" || c.SeatCount == 0:
		return "plan_draft"
	case c.MissingProfiles > 0 || c.MemberCount < c.TeamSize:
		return "waiting_profiles"
	case c.AcceptedCount < c.SeatCount:
		return "matching_pending"
	}
	return "roles_confirmed"
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	u := userFrom(ctx)
	cards, err := s.st.ListProjectCards(ctx, u.ID)
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, len(cards))
	owners := map[uuid.UUID]uuid.UUID{}
	for i, c := range cards {
		ids[i] = c.ID
		owners[c.ID] = c.OwnerID
	}
	rows, err := s.st.ListMembers(ctx, ids)
	if err != nil {
		return err
	}
	members := membersByProject(rows, owners)
	out := make([]projectCardJSON, len(cards))
	for i, c := range cards {
		m := members[c.ID]
		if m == nil {
			m = []memberJSON{}
		}
		out[i] = projectCardJSON{
			ID: c.ID.String(), Title: c.Title, Context: c.Context, Deadline: c.Deadline.Format(dateLayout),
			TeamSize: c.TeamSize, IsOwner: c.OwnerID == u.ID, Status: cardStatus(c), PlanStatus: c.PlanStatus,
			MemberCount: c.MemberCount, Members: m, DoneHours: c.DoneHours, AllHours: c.AllHours,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
	return nil
}

type projectInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Deadline    string `json:"deadline"`
	TeamSize    int    `json:"team_size"`
	Context     string `json:"context"`
}

func (in projectInput) details() domain.ProjectDetails {
	d, _ := time.Parse(dateLayout, strings.TrimSpace(in.Deadline))
	return domain.ProjectDetails{
		Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description),
		Deadline: d, Context: strings.TrimSpace(in.Context),
	}
}

// withFreshInviteCode retries fn on the rare invite-code collision.
func withFreshInviteCode[T any](fn func(code string) (T, error)) (T, error) {
	var zero T
	for range 8 {
		v, err := fn(domain.NewInviteCode())
		if store.IsUniqueViolation(err, "projects_invite_code_key") {
			continue
		}
		return v, err
	}
	return zero, errors.New("could not find a free invite code")
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) error {
	var in projectInput
	if err := decode(r, &in); err != nil {
		return err
	}
	d := in.details()
	if e := domain.ValidateNewProject(d, in.TeamSize, s.today()); !e.OK() {
		return errValidation(e)
	}
	ctx := r.Context()
	u := userFrom(ctx)
	p, err := withFreshInviteCode(func(code string) (db.Project, error) {
		var p db.Project
		err := s.st.Tx(ctx, func(q *db.Queries) error {
			var err error
			p, err = q.CreateProject(ctx, db.CreateProjectParams{
				Title: d.Title, Description: d.Description, Deadline: d.Deadline, TeamSize: int32(in.TeamSize),
				Context: d.Context, InviteCode: code, OwnerID: u.ID,
			})
			if err != nil {
				return err
			}
			_, err = q.AddMember(ctx, db.AddMemberParams{ProjectID: p.ID, UserID: u.ID})
			return err
		})
		return p, err
	})
	if err != nil {
		return err
	}
	b, err := s.projectBundle(ctx, p, u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, b)
	return nil
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) error {
	b, err := s.projectBundle(r.Context(), projectFrom(r.Context()), userFrom(r.Context()).ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, b)
	return nil
}

func (s *Server) handlePatchProject(w http.ResponseWriter, r *http.Request) error {
	var in projectInput
	if err := decode(r, &in); err != nil {
		return err
	}
	d := in.details()
	if e := domain.ValidateProjectUpdate(d, s.today()); !e.OK() {
		return errValidation(e)
	}
	ctx := r.Context()
	p := projectFrom(ctx)
	changed := d.Description != p.Description
	p, err := s.st.UpdateProjectDetails(ctx, db.UpdateProjectDetailsParams{
		ID: p.ID, Title: d.Title, Description: d.Description, Deadline: d.Deadline, Context: d.Context,
	})
	if err != nil {
		return err
	}
	b, err := s.projectBundle(ctx, p, userFrom(ctx).ID)
	if err != nil {
		return err
	}
	b["description_changed"] = changed
	writeJSON(w, http.StatusOK, b)
	return nil
}

func (s *Server) handleNewInviteCode(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	p := projectFrom(ctx)
	p, err := withFreshInviteCode(func(code string) (db.Project, error) {
		return s.st.UpdateInviteCode(ctx, db.UpdateInviteCodeParams{ID: p.ID, InviteCode: code})
	})
	if err != nil {
		return err
	}
	b, err := s.projectBundle(ctx, p, userFrom(ctx).ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, b)
	return nil
}

// handleRemoveMember removes a member; their tasks and seat become Unassigned
// through the FK rules (spec 6.10). The owner cannot be removed.
func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	p := projectFrom(ctx)
	uid, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		return errNotFound("Member not found.")
	}
	if uid == p.OwnerID {
		return errBadRequest("The owner cannot be removed.")
	}
	n, err := s.st.RemoveMember(ctx, db.RemoveMemberParams{ProjectID: p.ID, UserID: uid})
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound("Member not found.")
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ---- invites ----

type inviteJSON struct {
	Code               string     `json:"code"`
	ProjectID          string     `json:"project_id"`
	Title              string     `json:"title"`
	Context            string     `json:"context"`
	OwnerName          string     `json:"owner_name"`
	DescriptionExcerpt string     `json:"description_excerpt"`
	Deadline           string     `json:"deadline"`
	MemberCount        int32      `json:"member_count"`
	TeamSize           int32      `json:"team_size"`
	OpenRoles          []roleJSON `json:"open_roles"`
	IsMember           bool       `json:"is_member"`
}

type roleJSON struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

func errInvalidInvite() *apiError {
	return &apiError{Status: http.StatusNotFound, Code: "invalid_invite", Message: domain.MsgInvalidInvite}
}

// excerpt cuts s to about n runes on a word boundary.
func excerpt(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := string(r[:n])
	if i := strings.LastIndex(cut, " "); i > n/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,.;:") + "…"
}

func (s *Server) handleGetInvite(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	code := domain.NormalizeInviteCode(chi.URLParam(r, "code"))
	if !domain.ValidInviteCode(code) {
		return errInvalidInvite()
	}
	p, err := s.st.GetProjectByCode(ctx, code)
	if store.IsNotFound(err) {
		return errInvalidInvite()
	}
	if err != nil {
		return err
	}
	owner, err := s.st.GetUserByID(ctx, p.OwnerID)
	if err != nil {
		return err
	}
	count, err := s.st.CountMembers(ctx, p.ID)
	if err != nil {
		return err
	}
	isMember, err := s.st.IsMember(ctx, db.IsMemberParams{ProjectID: p.ID, UserID: userFrom(ctx).ID})
	if err != nil {
		return err
	}
	roles, err := s.openRoles(ctx, p)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"invite": inviteJSON{
		Code: p.InviteCode, ProjectID: p.ID.String(), Title: p.Title, Context: p.Context, OwnerName: owner.FullName,
		DescriptionExcerpt: excerpt(p.Description, 190), Deadline: p.Deadline.Format(dateLayout),
		MemberCount: count, TeamSize: p.TeamSize, OpenRoles: roles, IsMember: isMember,
	}})
	return nil
}

// openRoles lists plan seats nobody holds yet (all seats before matching).
func (s *Server) openRoles(ctx context.Context, p db.Project) ([]roleJSON, error) {
	seats, err := s.st.ListSeats(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	asg, err := s.st.ListAssignments(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	taken := map[uuid.UUID]bool{}
	for _, a := range asg {
		if a.UserID != nil {
			taken[a.SeatID] = true
		}
	}
	out := []roleJSON{}
	for _, st := range seats {
		if !taken[st.ID] {
			out = append(out, roleJSON{Name: st.Name, Color: st.ColorToken})
		}
	}
	return out, nil
}

func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	u := userFrom(ctx)
	code := domain.NormalizeInviteCode(chi.URLParam(r, "code"))
	if !domain.ValidInviteCode(code) {
		return errInvalidInvite()
	}
	var p db.Project
	already := false
	err := s.st.Tx(ctx, func(q *db.Queries) error {
		var err error
		// Lock the project row so two people cannot take the last seat at once.
		p, err = q.GetProjectByCodeForUpdate(ctx, code)
		if store.IsNotFound(err) {
			return errInvalidInvite()
		}
		if err != nil {
			return err
		}
		if already, err = q.IsMember(ctx, db.IsMemberParams{ProjectID: p.ID, UserID: u.ID}); err != nil || already {
			return err
		}
		n, err := q.CountMembers(ctx, p.ID)
		if err != nil {
			return err
		}
		if n >= p.TeamSize {
			return errConflict("team_full", fmt.Sprintf(domain.MsgTeamFullFmt, p.TeamSize))
		}
		_, err = q.AddMember(ctx, db.AddMemberParams{ProjectID: p.ID, UserID: u.ID})
		return err
	})
	if err != nil {
		return err
	}
	_, err = s.st.GetProfile(ctx, u.ID)
	if err != nil && !store.IsNotFound(err) {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project_id": p.ID.String(), "title": p.Title, "already_member": already, "profile_ready": err == nil,
	})
	return nil
}
