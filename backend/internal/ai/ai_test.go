package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---- fakes ----

type memRecorder struct {
	mu    sync.Mutex
	calls []Call
}

func (r *memRecorder) RecordCall(_ context.Context, c Call) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, c)
	return nil
}

func (r *memRecorder) CallsToday(context.Context, uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls), nil
}

// scripted returns canned answers per attempt and records the inputs.
type scripted struct {
	mu        sync.Mutex
	plans     []RolePlanOutput
	matches   []func(MatchInput) MatchOutput
	err       error
	delay     time.Duration
	planIns   []RolePlanInput
	matchIns  []MatchInput
	raciCalls int
}

func (s *scripted) GenerateRolePlan(ctx context.Context, in RolePlanInput) (RolePlanOutput, error) {
	s.mu.Lock()
	n := len(s.planIns)
	s.planIns = append(s.planIns, in)
	s.mu.Unlock()
	if s.delay > 0 {
		select {
		case <-ctx.Done():
			return RolePlanOutput{}, ctx.Err()
		case <-time.After(s.delay):
		}
	}
	if s.err != nil {
		return RolePlanOutput{}, s.err
	}
	return s.plans[min(n, len(s.plans)-1)], nil
}

func (s *scripted) MatchRoles(_ context.Context, in MatchInput) (MatchOutput, error) {
	s.mu.Lock()
	n := len(s.matchIns)
	s.matchIns = append(s.matchIns, in)
	s.mu.Unlock()
	if s.err != nil {
		return MatchOutput{}, s.err
	}
	return s.matches[min(n, len(s.matches)-1)](in), nil
}

func (s *scripted) DraftRACI(context.Context, RACIInput) (RACIOutput, error) {
	s.raciCalls++
	return RACIOutput{}, s.err
}

func newSvc(p Provider, rec *memRecorder) *Service {
	return NewService(p, "fake", rec, 200*time.Millisecond, 30)
}

var pid = uuid.New()

// ---- role plan ----

func TestValidateRolePlan_StandardPlansAreValid(t *testing.T) {
	for n := 3; n <= 6; n++ {
		if errs := ValidateRolePlan(StandardPlan(n), n); len(errs) > 0 {
			t.Errorf("standard plan for %d: %v", n, errs)
		}
	}
}

func TestValidateRolePlan_Rules(t *testing.T) {
	good := StandardPlan(4)
	mutate := func(f func(*RolePlanOutput)) RolePlanOutput {
		o := StandardPlan(4)
		f(&o)
		return o
	}
	cases := map[string]RolePlanOutput{
		"wrong count":   mutate(func(o *RolePlanOutput) { o.Seats = o.Seats[:3] }),
		"bad id":        mutate(func(o *RolePlanOutput) { o.Seats[1].ID = "S9" }),
		"sum not 100":   mutate(func(o *RolePlanOutput) { o.Seats[0].EffortShare = 30 }),
		"share below 5": mutate(func(o *RolePlanOutput) { o.Seats[0].EffortShare = 4; o.Seats[1].EffortShare = 46 }),
		"share above 60": mutate(func(o *RolePlanOutput) {
			o.Seats[0].EffortShare = 61
			o.Seats[1].EffortShare = 5
			o.Seats[2].EffortShare = 14
			o.Seats[3].EffortShare = 20
		}),
		"long name":    mutate(func(o *RolePlanOutput) { o.Seats[0].Name = strings.Repeat("x", 41) }),
		"long summary": mutate(func(o *RolePlanOutput) { o.Seats[0].Summary = strings.Repeat("x", 141) }),
		"two resps":    mutate(func(o *RolePlanOutput) { o.Seats[0].Responsibilities = []string{"a", "b"} }),
		"six skills":   mutate(func(o *RolePlanOutput) { o.Seats[0].Skills = []string{"a", "b", "c", "d", "e", "f"} }),
		"long resp":    mutate(func(o *RolePlanOutput) { o.Seats[0].Responsibilities[0] = strings.Repeat("x", 81) }),
		"long skill":   mutate(func(o *RolePlanOutput) { o.Seats[0].Skills[0] = strings.Repeat("x", 31) }),
		"long why":     mutate(func(o *RolePlanOutput) { o.Seats[0].Why = strings.Repeat("x", 301) }),
		"empty why":    mutate(func(o *RolePlanOutput) { o.Seats[0].Why = " " }),
	}
	if errs := ValidateRolePlan(good, 4); len(errs) > 0 {
		t.Fatalf("good plan rejected: %v", errs)
	}
	for name, o := range cases {
		if errs := ValidateRolePlan(o, 4); len(errs) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestMock_RolePlanIsValidAndInputDependent(t *testing.T) {
	m := Mock{}
	base := RolePlanInput{Title: "StudySpace", Deadline: time.Now().AddDate(0, 2, 0), WeeksLeft: 8}
	booking := "A web app to book study rooms with a booking API, a database and a mobile-friendly interface for students in the library."
	analytics := "A dashboard with statistics, charts and machine learning predictions on student data, with a Python analytics pipeline and a web view."
	for n := 3; n <= 6; n++ {
		in := base
		in.TeamSize, in.Description = n, booking
		out, err := m.GenerateRolePlan(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		if errs := ValidateRolePlan(out, n); len(errs) > 0 {
			t.Fatalf("mock plan for %d invalid: %v", n, errs)
		}
	}
	in := base
	in.TeamSize, in.Description = 4, booking
	a, _ := m.GenerateRolePlan(context.Background(), in)
	in.Description = analytics
	b, _ := m.GenerateRolePlan(context.Background(), in)
	if a.Seats[3].Name == b.Seats[3].Name {
		t.Fatalf("4th seat should depend on the description: %q vs %q", a.Seats[3].Name, b.Seats[3].Name)
	}
	in.Note = "we need more focus on design"
	c, _ := m.GenerateRolePlan(context.Background(), in)
	if c.Seats[3].Name != "UX/UI Designer" {
		t.Fatalf("note ignored: %q", c.Seats[3].Name)
	}
}

func TestNormalizeShares(t *testing.T) {
	for _, w := range [][]float64{{1, 1, 1}, {16, 24, 22, 16, 18}, {100, 1, 1, 1, 1, 1}, {1, 1, 1, 1, 1, 50}} {
		s := normalizeShares(w)
		sum := 0
		for _, v := range s {
			sum += v
			if v < 5 || v > 60 {
				t.Fatalf("%v → %v out of range", w, s)
			}
		}
		if sum != 100 {
			t.Fatalf("%v → %v sums to %d", w, s, sum)
		}
	}
}

// ---- service: retry, timeout, limit ----

func TestService_RetriesOnceWithValidationErrors(t *testing.T) {
	bad := StandardPlan(4)
	bad.Seats[0].EffortShare = 40 // sum 115
	p := &scripted{plans: []RolePlanOutput{bad, StandardPlan(4)}}
	rec := &memRecorder{}
	out, err := newSvc(p, rec).RolePlan(context.Background(), pid, RolePlanInput{TeamSize: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Seats) != 4 || len(p.planIns) != 2 {
		t.Fatalf("attempts = %d", len(p.planIns))
	}
	if len(p.planIns[0].Feedback) != 0 {
		t.Fatal("first attempt must have no feedback")
	}
	fb := strings.Join(p.planIns[1].Feedback, " | ")
	if !strings.Contains(fb, "sum to exactly 100") {
		t.Fatalf("retry feedback missing the validation error: %q", fb)
	}
	if len(rec.calls) != 2 || rec.calls[0].OK || !rec.calls[1].OK || rec.calls[1].Attempt != 2 || rec.calls[0].PromptVersion != "role_plan/v1" {
		t.Fatalf("recorded calls: %+v", rec.calls)
	}
}

func TestService_FailsAfterSecondInvalidAnswer(t *testing.T) {
	bad := StandardPlan(4)
	bad.Seats = bad.Seats[:2]
	p := &scripted{plans: []RolePlanOutput{bad}}
	_, err := newSvc(p, &memRecorder{}).RolePlan(context.Background(), pid, RolePlanInput{TeamSize: 4})
	if !errors.Is(err, ErrUnavailable) || len(p.planIns) != 2 {
		t.Fatalf("err=%v attempts=%d", err, len(p.planIns))
	}
}

func TestService_ProviderErrorRetriedThenUnavailable(t *testing.T) {
	p := &scripted{err: errors.New("HTTP 500")}
	rec := &memRecorder{}
	_, err := newSvc(p, rec).RolePlan(context.Background(), pid, RolePlanInput{TeamSize: 3})
	if !errors.Is(err, ErrUnavailable) || len(rec.calls) != 2 {
		t.Fatalf("err=%v calls=%d", err, len(rec.calls))
	}
}

func TestService_Timeout(t *testing.T) {
	p := &scripted{plans: []RolePlanOutput{StandardPlan(3)}, delay: 2 * time.Second}
	rec := &memRecorder{}
	start := time.Now()
	_, err := NewService(p, "slow", rec, 50*time.Millisecond, 30).RolePlan(context.Background(), pid, RolePlanInput{TeamSize: 3})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("timeout not enforced")
	}
	if len(rec.calls) != 2 || rec.calls[0].OK {
		t.Fatalf("calls: %+v", rec.calls)
	}
}

func TestService_DailyLimit(t *testing.T) {
	rec := &memRecorder{}
	for range 30 {
		rec.calls = append(rec.calls, Call{OK: true})
	}
	p := &scripted{plans: []RolePlanOutput{StandardPlan(3)}}
	_, err := newSvc(p, rec).RolePlan(context.Background(), pid, RolePlanInput{TeamSize: 3})
	if !errors.Is(err, ErrLimit) || len(p.planIns) != 0 {
		t.Fatalf("err=%v provider calls=%d", err, len(p.planIns))
	}
}

// ---- matching: privacy, labels, validation ----

var people = []Member{
	{UserID: uuid.New(), Strengths: "I organise group work well and keep deadlines. I like presenting.", Weaknesses: "Not strong at backend code.", Tags: []string{"Planning"}},
	{UserID: uuid.New(), Strengths: "Go and PostgreSQL. I built a REST API during my internship.", Weaknesses: "I don't enjoy CSS.", Tags: []string{"Go", "SQL"}},
	{UserID: uuid.New(), Strengths: "React and TypeScript. I care about fast, responsive pages.", Weaknesses: "I hate writing documentation.", Tags: []string{"React"}},
}

func seatsFor(n int) []SeatBrief {
	var out []SeatBrief
	for _, s := range StandardPlan(n).Seats {
		out = append(out, SeatBrief{ID: s.ID, Name: s.Name, Summary: s.Summary, Responsibilities: s.Responsibilities, Skills: s.Skills, EffortShare: s.EffortShare})
	}
	return out
}

func TestMock_MatchingIsValidAndQuotesProfiles(t *testing.T) {
	svc := newSvc(Mock{}, &memRecorder{})
	res, err := svc.Match(context.Background(), pid, "StudySpace", seatsFor(3), people)
	if err != nil {
		t.Fatal(err)
	}
	want := map[uuid.UUID]string{people[0].UserID: "S1", people[1].UserID: "S2", people[2].UserID: "S3"}
	for _, a := range res.Assignments {
		if want[a.UserID] != a.SeatID {
			t.Errorf("user %v got %s, want %s", a.UserID, a.SeatID, want[a.UserID])
		}
		if !strings.Contains(a.Explanation, "“") || !strings.Contains(strings.ToLower(a.Explanation), "you") {
			t.Errorf("explanation does not quote or address the person: %q", a.Explanation)
		}
	}
}

func TestService_MatchNeverSendsNamesAndShuffles(t *testing.T) {
	echo := func(in MatchInput) MatchOutput {
		var out MatchOutput
		for i, p := range in.People {
			out.Assignments = append(out.Assignments, Assignment{SeatID: in.Seats[i].ID, Person: p.Label, Fit: 70,
				RunnerUpSeatID: in.Seats[(i+1)%len(in.Seats)].ID,
				Explanation:    fmt.Sprintf("You write “%s”. That fits this role.", strings.Split(p.Strengths, ".")[0])})
		}
		return out
	}
	p := &scripted{matches: []func(MatchInput) MatchOutput{echo}}
	svc := newSvc(p, &memRecorder{})
	orders := map[string]bool{}
	for range 30 {
		res, err := svc.Match(context.Background(), pid, "StudySpace", seatsFor(3), people)
		if err != nil {
			t.Fatal(err)
		}
		// Labels map back to the right users.
		in := p.matchIns[len(p.matchIns)-1]
		for _, a := range res.Assignments {
			var label string
			for _, x := range in.People {
				for _, m := range people {
					if m.UserID == a.UserID && m.Strengths == x.Strengths {
						label = x.Label
					}
				}
			}
			if label == "" {
				t.Fatal("assignment mapped to the wrong user")
			}
		}
		var key strings.Builder
		for _, x := range in.People {
			key.WriteString(x.Strengths[:5])
		}
		orders[key.String()] = true
	}
	if len(orders) < 2 {
		t.Fatal("people were not shuffled between calls")
	}
	// Only label, profile text and tags exist on Person.
	for _, x := range p.matchIns[0].People {
		if !strings.HasPrefix(x.Label, "P") {
			t.Fatalf("label %q", x.Label)
		}
	}
}

func TestValidateMatch_RejectsPersonLabels(t *testing.T) {
	leak := func(in MatchInput) MatchOutput {
		var out MatchOutput
		for i, p := range in.People {
			out.Assignments = append(out.Assignments, Assignment{SeatID: in.Seats[i].ID, Person: p.Label, Fit: 70,
				RunnerUpSeatID: in.Seats[(i+1)%len(in.Seats)].ID,
				Explanation:    fmt.Sprintf("You write “%s”. %s fits this role.", strings.Split(p.Strengths, ".")[0], p.Label)})
		}
		return out
	}
	p := &scripted{matches: []func(MatchInput) MatchOutput{leak}}
	_, err := newSvc(p, &memRecorder{}).Match(context.Background(), pid, "X", seatsFor(3), people)
	if !errors.Is(err, ErrUnavailable) || len(p.matchIns) != 2 {
		t.Fatalf("err=%v attempts=%d", err, len(p.matchIns))
	}
	if !strings.Contains(strings.Join(p.matchIns[1].Feedback, " "), "person labels") {
		t.Fatalf("feedback: %v", p.matchIns[1].Feedback)
	}
}

func TestValidateMatch_Rules(t *testing.T) {
	in := MatchInput{Seats: seatsFor(3), People: []Person{
		{Label: "P1", Strengths: "I keep deadlines and like presenting to the class.", Weaknesses: "No backend."},
		{Label: "P2", Strengths: "Go and PostgreSQL every day.", Weaknesses: "No CSS."},
	}}
	ok := MatchOutput{Assignments: []Assignment{
		{SeatID: "S1", Person: "P1", Fit: 90, RunnerUpSeatID: "S3", Explanation: "You “keep deadlines”, which is the core of this role. It fits you."},
		{SeatID: "S2", Person: "P2", Fit: 80, RunnerUpSeatID: "S1", Explanation: "You write \"Go and PostgreSQL every day\". That is the backend you need."},
	}}
	if errs := ValidateMatch(ok, in); len(errs) > 0 {
		t.Fatalf("valid match rejected: %v", errs)
	}
	type mut func(*MatchOutput)
	cases := map[string]mut{
		"person twice":      func(o *MatchOutput) { o.Assignments[1].Person = "P1" },
		"seat twice":        func(o *MatchOutput) { o.Assignments[1].SeatID = "S1" },
		"unknown seat":      func(o *MatchOutput) { o.Assignments[0].SeatID = "S9" },
		"missing person":    func(o *MatchOutput) { o.Assignments = o.Assignments[:1] },
		"fit above 100":     func(o *MatchOutput) { o.Assignments[0].Fit = 101 },
		"runner equal seat": func(o *MatchOutput) { o.Assignments[0].RunnerUpSeatID = "S1" },
		"one sentence":      func(o *MatchOutput) { o.Assignments[0].Explanation = "You “keep deadlines” and fit." },
		"four sentences":    func(o *MatchOutput) { o.Assignments[0].Explanation = "You “keep deadlines”. A. B. C." },
		"no quote":          func(o *MatchOutput) { o.Assignments[0].Explanation = "You keep deadlines. It fits you." },
		"made-up quote":     func(o *MatchOutput) { o.Assignments[0].Explanation = "You “love Kubernetes”. It fits you." },
		"not addressed":     func(o *MatchOutput) { o.Assignments[0].Explanation = "They “keep deadlines”. It fits them." },
		"label in note":     func(o *MatchOutput) { n := "Pair P2 with the PM."; o.GrowthNote = &n },
		"gap unknown seat":  func(o *MatchOutput) { o.Gaps = []Gap{{Skill: "A11y", SeatID: "S7", Reason: "Needed."}} },
	}
	for name, f := range cases {
		o := MatchOutput{Assignments: append([]Assignment{}, ok.Assignments...)}
		f(&o)
		if errs := ValidateMatch(o, in); len(errs) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestSentenceCount(t *testing.T) {
	cases := map[string]int{
		"One. Two.":                       2,
		"You write “pages.” Two.":         2,
		"Uses Node.js daily. Two! Three?": 3,
		"No end":                          1,
		"":                                0,
	}
	for s, want := range cases {
		if got := sentenceCount(s); got != want {
			t.Errorf("sentenceCount(%q) = %d, want %d", s, got, want)
		}
	}
}

// ---- RACI ----

func TestMock_RACIIsValidForEveryTeamSize(t *testing.T) {
	for n := 3; n <= 6; n++ {
		seats := seatsFor(n)
		out, err := Mock{}.DraftRACI(context.Background(), RACIInput{Seats: seats})
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]string, len(seats))
		for i, s := range seats {
			ids[i] = s.ID
		}
		if errs := ValidateRACI(out, ids); len(errs) > 0 {
			t.Fatalf("team of %d: %v", n, errs)
		}
	}
}

func TestValidateRACI_Rules(t *testing.T) {
	ids := []string{"S1", "S2", "S3"}
	row := func(c map[string]string) RACIActivity { return RACIActivity{Name: "Work", Cells: c} }
	good := map[string]string{"S1": "A", "S2": "R", "S3": ""}
	rows := func(r RACIActivity) RACIOutput {
		o := RACIOutput{}
		for range 6 {
			o.Activities = append(o.Activities, row(good))
		}
		o.Activities[0] = r
		return o
	}
	if errs := ValidateRACI(rows(row(good)), ids); len(errs) > 0 {
		t.Fatalf("good RACI rejected: %v", errs)
	}
	for name, r := range map[string]RACIActivity{
		"no A":       row(map[string]string{"S1": "R", "S2": "C"}),
		"two A":      row(map[string]string{"S1": "A", "S2": "A", "S3": "R"}),
		"no R":       row(map[string]string{"S1": "A", "S2": "C"}),
		"bad value":  row(map[string]string{"S1": "A", "S2": "R", "S3": "X"}),
		"bad seat":   row(map[string]string{"S1": "A", "S9": "R"}),
		"empty name": {Name: "", Cells: good},
	} {
		if errs := ValidateRACI(rows(r), ids); len(errs) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
	short := RACIOutput{Activities: rows(row(good)).Activities[:5]}
	if errs := ValidateRACI(short, ids); len(errs) == 0 {
		t.Error("5 activities accepted")
	}
}

func TestPromptsRenderAndCarryVersion(t *testing.T) {
	for _, k := range []string{KindRolePlan, KindMatching, KindRACI} {
		if v := PromptVersion(k); !strings.HasPrefix(v, k+"/v") {
			t.Errorf("%s version = %q", k, v)
		}
	}
	p, err := renderRolePlan(RolePlanInput{Title: "T", Description: "D", TeamSize: 4, WeeksLeft: 5, Deadline: time.Now(), Feedback: []string{"fix the sum"}})
	if err != nil || !strings.Contains(p, "exactly 4 roles") || !strings.Contains(p, "fix the sum") {
		t.Fatalf("role plan prompt: %v\n%s", err, p)
	}
	m, err := renderMatching(MatchInput{ProjectTitle: "T", Seats: seatsFor(3), People: []Person{{Label: "P1", Strengths: "x", Weaknesses: "y"}}})
	if err != nil || !strings.Contains(m, "P1") || !strings.Contains(m, "fewer people than seats") {
		t.Fatalf("matching prompt: %v\n%s", err, m)
	}
	r, err := renderRACI(RACIInput{Title: "T", Seats: seatsFor(3), Deadline: time.Now()})
	if err != nil || !strings.Contains(r, "S1, S2, S3") {
		t.Fatalf("raci prompt: %v\n%s", err, r)
	}
}
