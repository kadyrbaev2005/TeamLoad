package http

import (
	"net/http"
	"strings"
	"time"

	"teamload/internal/domain"
	"teamload/internal/store"
	"teamload/internal/store/db"
)

type profileJSON struct {
	Strengths  string    `json:"strengths"`
	Weaknesses string    `json:"weaknesses"`
	Tags       []string  `json:"tags"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func toProfileJSON(p db.Profile) profileJSON {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return profileJSON{Strengths: p.Strengths, Weaknesses: p.Weaknesses, Tags: tags, UpdatedAt: p.UpdatedAt}
}

type ownershipJSON struct {
	ProjectID    string  `json:"project_id"`
	ProjectTitle string  `json:"project_title"`
	NewOwnerName *string `json:"new_owner_name"`
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	u := userFrom(ctx)
	var prof *profileJSON
	p, err := s.st.GetProfile(ctx, u.ID)
	switch {
	case err == nil:
		pj := toProfileJSON(p)
		prof = &pj
	case !store.IsNotFound(err):
		return err
	}
	owned, err := s.st.ListOwnedProjects(ctx, u.ID)
	if err != nil {
		return err
	}
	transfers := make([]ownershipJSON, 0, len(owned))
	for _, pr := range owned {
		t := ownershipJSON{ProjectID: pr.ID.String(), ProjectTitle: pr.Title}
		next, err := s.st.EarliestOtherMember(ctx, db.EarliestOtherMemberParams{ProjectID: pr.ID, UserID: u.ID})
		switch {
		case err == nil:
			name := domain.FirstName(next.FullName)
			t.NewOwnerName = &name
		case !store.IsNotFound(err):
			return err
		}
		transfers = append(transfers, t)
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": prof, "ownership": transfers})
	return nil
}

func (s *Server) handlePutProfile(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Strengths  string   `json:"strengths"`
		Weaknesses string   `json:"weaknesses"`
		Tags       []string `json:"tags"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	p := domain.Profile{
		Strengths:  strings.TrimSpace(in.Strengths),
		Weaknesses: strings.TrimSpace(in.Weaknesses),
		Tags:       domain.NormalizeTags(in.Tags),
	}
	if e := domain.ValidateProfile(p); !e.OK() {
		return errValidation(e)
	}
	u := userFrom(r.Context())
	var saved db.Profile
	var outdated int64
	err := s.st.Tx(r.Context(), func(q *db.Queries) error {
		prev, err := q.GetProfile(r.Context(), u.ID)
		changed := store.IsNotFound(err) || prev.Strengths != p.Strengths || prev.Weaknesses != p.Weaknesses || !sameTags(prev.Tags, p.Tags)
		if err != nil && !store.IsNotFound(err) {
			return err
		}
		if saved, err = q.UpsertProfile(r.Context(), db.UpsertProfileParams{
			UserID: u.ID, Strengths: p.Strengths, Weaknesses: p.Weaknesses, Tags: p.Tags,
		}); err != nil {
			return err
		}
		if changed {
			outdated, err = q.MarkUserSuggestionsOutdated(r.Context(), &u.ID)
		}
		return err
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"profile": toProfileJSON(saved), "outdated_count": outdated})
	return nil
}

func sameTags(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// handleDeleteProfile deletes the account and all personal data (spec 6.5).
// Owned projects pass to the earliest-joined remaining member or are deleted;
// tasks and seats are unassigned by the FK rules.
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	u := userFrom(ctx)
	err := s.st.Tx(ctx, func(q *db.Queries) error {
		owned, err := q.ListOwnedProjects(ctx, u.ID)
		if err != nil {
			return err
		}
		for _, p := range owned {
			next, err := q.EarliestOtherMember(ctx, db.EarliestOtherMemberParams{ProjectID: p.ID, UserID: u.ID})
			switch {
			case store.IsNotFound(err):
				if err := q.DeleteProject(ctx, p.ID); err != nil {
					return err
				}
			case err != nil:
				return err
			default:
				if err := q.SetProjectOwner(ctx, db.SetProjectOwnerParams{ID: p.ID, OwnerID: next.ID}); err != nil {
					return err
				}
			}
		}
		return q.DeleteUser(ctx, u.ID)
	})
	if err != nil {
		return err
	}
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
