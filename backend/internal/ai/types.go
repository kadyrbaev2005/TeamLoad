// Package ai is TeamLoad's LLM layer: one Provider interface, strict JSON
// output validation, retry-once, a spending guard and the mock provider.
// The LLM is called only from the backend.
package ai

import (
	"context"
	"time"
)

// Provider generates the three AI artefacts. Implementations must return
// output in the shapes below; the Service validates every rule.
// Adding a provider means adding one file that implements this interface.
type Provider interface {
	GenerateRolePlan(ctx context.Context, in RolePlanInput) (RolePlanOutput, error)
	MatchRoles(ctx context.Context, in MatchInput) (MatchOutput, error)
	DraftRACI(ctx context.Context, in RACIInput) (RACIOutput, error)
}

// RolePlanInput describes the project for the role plan prompt.
type RolePlanInput struct {
	Title       string
	Description string
	Context     string
	Note        string
	Deadline    time.Time
	WeeksLeft   int
	TeamSize    int
	// Feedback holds validation errors from the previous attempt (retry only).
	Feedback []string
}

// Seat is one role in the plan.
type Seat struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Summary          string   `json:"summary"`
	Responsibilities []string `json:"responsibilities"`
	Skills           []string `json:"skills"`
	EffortShare      int      `json:"effort_share"`
	Why              string   `json:"why"`
}

// RolePlanOutput is the role plan JSON.
type RolePlanOutput struct {
	Seats []Seat `json:"seats"`
}

// SeatBrief is a seat as given to matching and RACI prompts.
type SeatBrief struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Summary          string   `json:"summary"`
	Responsibilities []string `json:"responsibilities"`
	Skills           []string `json:"skills"`
	EffortShare      int      `json:"effort_share"`
}

// Person is an anonymous profile: only the label, profile text and tags.
// Names, emails, gender, age and nationality are never part of it.
type Person struct {
	Label      string   `json:"label"`
	Strengths  string   `json:"strengths"`
	Weaknesses string   `json:"weaknesses"`
	Tags       []string `json:"tags"`
}

// MatchInput is the matching prompt input.
type MatchInput struct {
	ProjectTitle string
	Seats        []SeatBrief
	People       []Person
	Feedback     []string
}

// Assignment is one person → seat suggestion.
type Assignment struct {
	SeatID         string `json:"seat_id"`
	Person         string `json:"person"`
	Fit            int    `json:"fit"`
	RunnerUpSeatID string `json:"runner_up_seat_id"`
	Explanation    string `json:"explanation"`
}

// Gap is a skill no one on the team listed.
type Gap struct {
	Skill  string `json:"skill"`
	SeatID string `json:"seat_id"`
	Reason string `json:"reason"`
}

// MatchOutput is the matching JSON.
type MatchOutput struct {
	Assignments []Assignment `json:"assignments"`
	Gaps        []Gap        `json:"gaps"`
	GrowthNote  *string      `json:"growth_note"`
}

// RACIInput is the RACI prompt input.
type RACIInput struct {
	Title       string
	Description string
	Context     string
	Deadline    time.Time
	WeeksLeft   int
	Seats       []SeatBrief
	Feedback    []string
}

// RACIActivity is one row: seat ID → R, A, C, I or "".
type RACIActivity struct {
	Name  string            `json:"name"`
	Cells map[string]string `json:"cells"`
}

// RACIOutput is the RACI JSON.
type RACIOutput struct {
	Activities []RACIActivity `json:"activities"`
}
