// Command aieval measures the configured LLM provider's matching accuracy
// and runs a swapped-name bias check (spec 7.6).
//
// Accuracy: each profile is matched alone against the standard 6-seat plan
// and the chosen seat is compared with the expected role (target ≥ 7/10).
//
// Bias: the profiles are matched as two teams of five, once with their own
// names and once with the names swapped and the order shuffled. The same
// profile must get the same role in ≥ 9/10 cases. Names never reach the
// provider (ai.Member has no name field), so this checks the whole path.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	"teamload/internal/ai"
	"teamload/internal/config"
)

type profile struct {
	Name         string   `json:"name"`
	Strengths    string   `json:"strengths"`
	Weaknesses   string   `json:"weaknesses"`
	Tags         []string `json:"tags"`
	ExpectedRole string   `json:"expected_role"`
}

// memRecorder keeps calls in memory: the tool needs no database.
type memRecorder struct {
	mu sync.Mutex
	n  int
}

func (r *memRecorder) RecordCall(context.Context, ai.Call) error {
	r.mu.Lock()
	r.n++
	r.mu.Unlock()
	return nil
}

func (r *memRecorder) CallsToday(context.Context, uuid.UUID) (int, error) { return 0, nil }

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))
	path := flag.String("profiles", "testdata/ai_eval/profiles.json", "fixture with profiles and expected roles")
	flag.Parse()
	ok, err := run(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aieval:", err)
		os.Exit(2)
	}
	if !ok {
		os.Exit(1)
	}
}

func run(path string) (bool, error) {
	cfg, err := config.Load()
	if err != nil {
		return false, err
	}
	raw, err := os.ReadFile(path) // #nosec G304 -- path is an operator-supplied fixture
	if err != nil {
		return false, err
	}
	var profiles []profile
	if err := json.Unmarshal(raw, &profiles); err != nil {
		return false, err
	}
	delay := cfg.MockDelay
	if cfg.LLMProvider == "mock" {
		delay = 0
	}
	provider, err := ai.NewProvider(cfg.LLMProvider, cfg.LLMAPIKey, cfg.LLMModel, delay)
	if err != nil {
		return false, err
	}
	rec := &memRecorder{}
	svc := ai.NewService(provider, cfg.LLMProvider, rec, cfg.LLMTimeout, 1<<30)
	ctx := context.Background()
	project := uuid.New()

	plan := ai.StandardPlan(6)
	seats := make([]ai.SeatBrief, len(plan.Seats))
	seatName := map[string]string{}
	for i, s := range plan.Seats {
		seats[i] = ai.SeatBrief{ID: s.ID, Name: s.Name, Summary: s.Summary, Responsibilities: s.Responsibilities, Skills: s.Skills, EffortShare: s.EffortShare}
		seatName[s.ID] = s.Name
	}
	ids := make([]uuid.UUID, len(profiles))
	for i := range ids {
		ids[i] = uuid.New()
	}
	member := func(i int) ai.Member {
		p := profiles[i]
		return ai.Member{UserID: ids[i], Strengths: p.Strengths, Weaknesses: p.Weaknesses, Tags: p.Tags}
	}

	fmt.Printf("Provider: %s   Profiles: %d   Plan: standard 6 seats\n\n", cfg.LLMProvider, len(profiles))
	fmt.Println("ACCURACY (each profile alone)")
	correct := 0
	for i, p := range profiles {
		start := time.Now()
		res, err := svc.Match(ctx, project, "Evaluation", seats, []ai.Member{member(i)})
		if err != nil {
			return false, fmt.Errorf("profile %d: %w", i+1, err)
		}
		got := seatName[res.Assignments[0].SeatID]
		mark := "✗"
		if got == p.ExpectedRole {
			correct++
			mark = "✓"
		}
		fmt.Printf("  %s %-22s expected %-20s got %-20s fit %3d  (%dms)\n", mark, p.Name, p.ExpectedRole, got,
			res.Assignments[0].Fit, time.Since(start).Milliseconds())
	}
	accOK := correct*10 >= 7*len(profiles)
	fmt.Printf("  Accuracy: %d/%d (target ≥ 7/10) %s\n\n", correct, len(profiles), verdict(accOK))

	fmt.Println("BIAS CHECK (swapped names, two teams of five)")
	teams := [][]int{}
	for i := 0; i < len(profiles); i += 5 {
		team := []int{}
		for j := i; j < i+5 && j < len(profiles); j++ {
			team = append(team, j)
		}
		teams = append(teams, team)
	}
	original := map[int]string{}
	swapped := map[int]string{}
	for _, team := range teams {
		a, err := matchTeam(ctx, svc, project, seats, team, member, false)
		if err != nil {
			return false, err
		}
		b, err := matchTeam(ctx, svc, project, seats, team, member, true)
		if err != nil {
			return false, err
		}
		for k, v := range a {
			original[k] = v
		}
		for k, v := range b {
			swapped[k] = v
		}
	}
	same := 0
	n := len(profiles)
	for i, p := range profiles {
		swappedName := profiles[(i+1)%n].Name
		mark := "≠"
		if original[i] == swapped[i] {
			same++
			mark = "="
		}
		fmt.Printf("  %s %-22s as %-22s %-20s vs %s\n", mark, p.Name, swappedName, seatName[original[i]], seatName[swapped[i]])
	}
	biasOK := same*10 >= 9*n
	fmt.Printf("  Same role under swapped names: %d/%d (target ≥ 9/10) %s\n\n", same, n, verdict(biasOK))
	fmt.Printf("Provider calls: %d\n", rec.n)
	return accOK && biasOK, nil
}

// matchTeam matches one team. With swap, each profile is presented under the
// next person's name and the member order is reversed; names are only
// printed, they are not part of ai.Member.
func matchTeam(ctx context.Context, svc *ai.Service, project uuid.UUID, seats []ai.SeatBrief, team []int,
	member func(int) ai.Member, swap bool) (map[int]string, error) {
	order := append([]int{}, team...)
	if swap {
		for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
			order[i], order[j] = order[j], order[i]
		}
	}
	members := make([]ai.Member, len(order))
	byUser := map[uuid.UUID]int{}
	for k, i := range order {
		members[k] = member(i)
		byUser[members[k].UserID] = i
	}
	res, err := svc.Match(ctx, project, "Evaluation", seats, members)
	if err != nil {
		return nil, err
	}
	out := map[int]string{}
	for _, a := range res.Assignments {
		out[byUser[a.UserID]] = a.SeatID
	}
	return out, nil
}

func verdict(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}
