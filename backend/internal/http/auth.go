package http

import (
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"teamload/internal/domain"
	"teamload/internal/store"
	"teamload/internal/store/db"
)

const bcryptCost = 12

type userJSON struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Initials string `json:"initials"`
}

func toUserJSON(u db.User) userJSON {
	return userJSON{ID: u.ID.String(), Email: u.Email, FullName: u.FullName, Initials: domain.Initials(u.FullName)}
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Consent  bool   `json:"consent"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	reg := domain.Registration{FullName: in.FullName, Email: in.Email, Password: in.Password, Consent: in.Consent}
	if e := domain.ValidateRegistration(reg); !e.OK() {
		return errValidation(e)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)
	if err != nil {
		return err
	}
	u, err := s.st.CreateUser(r.Context(), db.CreateUserParams{
		Email:        domain.NormalizeEmail(in.Email),
		PasswordHash: string(hash),
		FullName:     strings.TrimSpace(in.FullName),
		ConsentAt:    s.now(),
	})
	if store.IsUniqueViolation(err, "users_email_key") {
		return &apiError{Status: http.StatusConflict, Code: "email_taken", Message: domain.MsgEmailTaken,
			Fields: map[string]string{"email": domain.MsgEmailTaken}}
	}
	if err != nil {
		return err
	}
	if err := s.issueSession(w, u.ID); err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": toUserJSON(u)})
	return nil
}

// dummyHash keeps login timing similar for unknown emails.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser"), bcryptCost)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if !s.login.allow(clientIP(r)) {
		return &apiError{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: domain.MsgTooManyLogins}
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		return err
	}
	invalid := &apiError{Status: http.StatusUnauthorized, Code: "invalid_credentials", Message: domain.MsgInvalidLogin}
	u, err := s.st.GetUserByEmail(r.Context(), domain.NormalizeEmail(in.Email))
	if store.IsNotFound(err) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password))
		return invalid
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) != nil {
		return invalid
	}
	if err := s.issueSession(w, u.ID); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": toUserJSON(u)})
	return nil
}

func (s *Server) handleLogout(w http.ResponseWriter, _ *http.Request) error {
	s.clearSession(w)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, http.StatusOK, map[string]any{"user": toUserJSON(userFrom(r.Context()))})
	return nil
}
