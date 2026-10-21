// Package domain holds TeamLoad's pure business rules: validation, workload
// status, the RACI check and invite codes. It has no I/O.
package domain

import (
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

// Exact user-facing messages (QA tests these strings, see SPEC §6).
const (
	MsgEmailTaken        = "Email already registered"
	MsgNoConsent         = "Please accept personal-data processing to continue"
	MsgInvalidLogin      = "Invalid email or password"
	MsgInvalidInvite     = "Invalid invite code"
	MsgTeamFullFmt       = "Team is full: maximum %d members"
	MsgAIUnavailable     = "AI service temporarily unavailable. Please retry or assign roles manually"
	MsgAILimit           = "Daily AI limit reached for this project. Try again tomorrow or edit manually."
	MsgProfileTooShort   = "Write at least 100 characters (about two sentences)"
	MsgEstimateRequired  = "Enter an estimate greater than 0 hours"
	MsgEstimateRange     = "Estimate must be between 0.5 and 40 hours."
	MsgTaskTitleRequired = "Give the task a title."
	MsgTooManyLogins     = "Too many login attempts. Try again in 15 minutes."

	MsgFullName       = "Enter your full name (2–100 characters)."
	MsgEmailInvalid   = "Enter a valid email address."
	MsgPasswordShort  = "Use at least 8 characters." // #nosec G101 -- UI message, not a credential
	MsgTitleRequired  = "Add a project title."
	MsgTitleTooLong   = "Keep the title under 100 characters."
	MsgDescShort      = "The description needs at least 100 characters so the AI can understand the project."
	MsgDescLong       = "Keep the description under 3000 characters."
	MsgDeadlineMin    = "Set a deadline at least 7 days from today."
	MsgDeadlinePast   = "The deadline cannot be in the past."
	MsgTeamSize       = "Team size must be between 3 and 6."
	MsgContextLong    = "Keep the course or context under 100 characters."
	MsgProfileTooLong = "Keep each answer under 1000 characters."
	MsgTooManyTags    = "Use at most 10 tags."
	MsgTagLength      = "Each tag must be 1–24 characters."
	MsgDuplicateTag   = "Each tag can be added only once."
	MsgTaskTitleLong  = "Keep the title under 120 characters."
	MsgTaskStatus     = "Choose To do, Doing or Done."
	MsgSeatName       = "Give every role a name (up to 40 characters)."
	MsgSeatSummary    = "Keep each summary under 140 characters."
	MsgSeatShare      = "Each effort share must be a whole number from 5 to 60."
	MsgShareTotal     = "Effort total must be 100%."
	MsgSeatCount      = "The plan must keep one role per seat."
	MsgNoteTooLong    = "Keep the note under 500 characters."
	MsgActivityName   = "Give every activity a name (up to 120 characters)."
	MsgRaciValue      = "RACI values must be R, A, C, I or empty."
)

// Limits shared by validators and the API.
const (
	MinTeamSize       = 3
	MaxTeamSize       = 6
	MinDescription    = 100
	MaxDescription    = 3000
	MaxTitle          = 100
	MaxContext        = 100
	MinDeadlineDays   = 7
	MaxProfileField   = 1000
	MinStrengths      = 30
	MinWeaknesses     = 10
	MinProfileTotal   = 100
	MaxTags           = 10
	MaxTagLength      = 24
	MaxTaskTitle      = 120
	MinEstimate       = 0.5
	MaxEstimate       = 40
	MaxRegenNote      = 500
	MaxSeatName       = 40
	MaxSeatSummary    = 140
	MinEffortShare    = 5
	MaxEffortShare    = 60
	MaxActivityName   = 120
	MinPasswordLength = 8
)

// FieldErrors maps a field name to its message. Empty means valid.
type FieldErrors map[string]string

// OK reports whether there are no errors.
func (f FieldErrors) OK() bool { return len(f) == 0 }

func runes(s string) int { return utf8.RuneCountInString(s) }

// NormalizeEmail trims and lower-cases an email address.
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// ValidEmail reports whether s looks like a plain address (no display name).
func ValidEmail(s string) bool {
	if s == "" || strings.ContainsAny(s, " <>") {
		return false
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s {
		return false
	}
	at := strings.LastIndex(s, "@")
	return at > 0 && strings.Contains(s[at+1:], ".")
}

// Registration is the sign-up form.
type Registration struct {
	FullName string
	Email    string
	Password string
	Consent  bool
}

// ValidateRegistration checks the sign-up form. Consent is reported under
// "consent" with the exact charter message.
func ValidateRegistration(r Registration) FieldErrors {
	e := FieldErrors{}
	if n := runes(strings.TrimSpace(r.FullName)); n < 2 || n > 100 {
		e["full_name"] = MsgFullName
	}
	if !ValidEmail(NormalizeEmail(r.Email)) {
		e["email"] = MsgEmailInvalid
	}
	if runes(r.Password) < MinPasswordLength {
		e["password"] = MsgPasswordShort
	}
	if !r.Consent {
		e["consent"] = MsgNoConsent
	}
	return e
}

// ProjectDetails are the editable project fields.
type ProjectDetails struct {
	Title       string
	Description string
	Deadline    time.Time
	Context     string
}

// DateOnly truncates t to midnight in its own location.
func DateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ValidateProjectDetails checks title, description, context and the
// deadline against minDeadline (a calendar date).
func ValidateProjectDetails(p ProjectDetails, minDeadline time.Time, deadlineMsg string) FieldErrors {
	e := FieldErrors{}
	switch n := runes(strings.TrimSpace(p.Title)); {
	case n == 0:
		e["title"] = MsgTitleRequired
	case n > MaxTitle:
		e["title"] = MsgTitleTooLong
	}
	switch n := runes(strings.TrimSpace(p.Description)); {
	case n < MinDescription:
		e["description"] = MsgDescShort
	case n > MaxDescription:
		e["description"] = MsgDescLong
	}
	if p.Deadline.IsZero() || DateOnly(p.Deadline).Before(DateOnly(minDeadline)) {
		e["deadline"] = deadlineMsg
	}
	if runes(strings.TrimSpace(p.Context)) > MaxContext {
		e["context"] = MsgContextLong
	}
	return e
}

// ValidateNewProject checks the create form; today is the current date in APP_TZ.
func ValidateNewProject(p ProjectDetails, teamSize int, today time.Time) FieldErrors {
	e := ValidateProjectDetails(p, DateOnly(today).AddDate(0, 0, MinDeadlineDays), MsgDeadlineMin)
	if teamSize < MinTeamSize || teamSize > MaxTeamSize {
		e["team_size"] = MsgTeamSize
	}
	return e
}

// ValidateProjectUpdate checks the settings form (deadline may be today).
func ValidateProjectUpdate(p ProjectDetails, today time.Time) FieldErrors {
	return ValidateProjectDetails(p, today, MsgDeadlinePast)
}

// Profile is the skills profile form.
type Profile struct {
	Strengths  string
	Weaknesses string
	Tags       []string
}

// NormalizeTags trims tags and drops empty ones.
func NormalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ValidateProfile checks a profile. Too-short text is reported once under
// "strengths" with the exact message.
func ValidateProfile(p Profile) FieldErrors {
	e := FieldErrors{}
	s, w := runes(strings.TrimSpace(p.Strengths)), runes(strings.TrimSpace(p.Weaknesses))
	if s > MaxProfileField {
		e["strengths"] = MsgProfileTooLong
	}
	if w > MaxProfileField {
		e["weaknesses"] = MsgProfileTooLong
	}
	if e.OK() && (s < MinStrengths || w < MinWeaknesses || s+w < MinProfileTotal) {
		e["strengths"] = MsgProfileTooShort
	}
	if len(p.Tags) > MaxTags {
		e["tags"] = MsgTooManyTags
	} else {
		seen := map[string]bool{}
		for _, t := range p.Tags {
			n := runes(strings.TrimSpace(t))
			if n < 1 || n > MaxTagLength {
				e["tags"] = MsgTagLength
				break
			}
			k := strings.ToLower(strings.TrimSpace(t))
			if seen[k] {
				e["tags"] = MsgDuplicateTag
				break
			}
			seen[k] = true
		}
	}
	return e
}

// Task statuses.
const (
	StatusTodo  = "todo"
	StatusDoing = "doing"
	StatusDone  = "done"
)

// ValidStatus reports whether s is a board column.
func ValidStatus(s string) bool { return s == StatusTodo || s == StatusDoing || s == StatusDone }

// ValidateTaskTitle checks a task title.
func ValidateTaskTitle(title string) string {
	switch n := runes(strings.TrimSpace(title)); {
	case n == 0:
		return MsgTaskTitleRequired
	case n > MaxTaskTitle:
		return MsgTaskTitleLong
	}
	return ""
}

// ValidateEstimate checks an hour estimate; nil means the field was empty.
// It returns the estimate rounded to the 0.5 h step.
func ValidateEstimate(h *float64) (float64, string) {
	if h == nil || *h <= 0 {
		return 0, MsgEstimateRequired
	}
	if *h < MinEstimate || *h > MaxEstimate {
		return 0, MsgEstimateRange
	}
	r := float64(int(*h*2+0.5)) / 2
	return r, ""
}

// SeatEdit is the owner-editable part of a seat.
type SeatEdit struct {
	Name        string
	Summary     string
	EffortShare int
}

// ValidateSeatEdits checks inline plan edits and the 100% total.
func ValidateSeatEdits(seats []SeatEdit, teamSize int) FieldErrors {
	e := FieldErrors{}
	if len(seats) != teamSize {
		e["seats"] = MsgSeatCount
		return e
	}
	sum := 0
	for _, s := range seats {
		if n := runes(strings.TrimSpace(s.Name)); n < 1 || n > MaxSeatName {
			e["name"] = MsgSeatName
		}
		if runes(strings.TrimSpace(s.Summary)) > MaxSeatSummary {
			e["summary"] = MsgSeatSummary
		}
		if s.EffortShare < MinEffortShare || s.EffortShare > MaxEffortShare {
			e["effort_share"] = MsgSeatShare
		}
		sum += s.EffortShare
	}
	if sum != 100 {
		e["total"] = MsgShareTotal
	}
	return e
}

// DaysLeft returns whole days from today to deadline, never negative.
func DaysLeft(deadline, today time.Time) int {
	d := int(DateOnly(deadline).Sub(DateOnly(today)).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

// WeeksLeft returns whole weeks until the deadline (at least 1).
func WeeksLeft(deadline, today time.Time) int {
	w := DaysLeft(deadline, today) / 7
	if w < 1 {
		return 1
	}
	return w
}
