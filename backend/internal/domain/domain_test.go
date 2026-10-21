package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestLoadStatus_ReferenceCase(t *testing.T) {
	active := map[string]float64{"Akzhol": 14, "Beknur": 22, "Yerasyl": 16, "Irisbek": 15, "Zhanarys": 9}
	avg := TeamAverage([]float64{14, 22, 16, 15, 9})
	if math.Abs(avg-15.2) > 1e-9 {
		t.Fatalf("average = %v, want 15.2", avg)
	}
	if th := avg * LoadThresholds.OverloadedAbove; math.Abs(th-19.76) > 1e-9 {
		t.Fatalf("threshold = %v, want 19.76", th)
	}
	want := map[string]string{
		"Akzhol": LoadBalanced, "Beknur": LoadOverloaded, "Yerasyl": LoadBalanced,
		"Irisbek": LoadBalanced, "Zhanarys": LoadUnderloaded,
	}
	for name, a := range active {
		if got := LoadStatus(a, avg); got != want[name] {
			t.Errorf("%s (%v h): %s, want %s", name, a, got, want[name])
		}
	}
}

func TestLoadStatus_Boundaries(t *testing.T) {
	cases := []struct {
		active, avg float64
		want        string
	}{
		{13, 10, LoadBalanced},     // exactly 130% is not overloaded
		{13.1, 10, LoadOverloaded}, // just above
		{7.5, 10, LoadBalanced},    // exactly 75% is balanced
		{7.4, 10, LoadUnderloaded},
		{0, 0, LoadBalanced}, // no load at all
		{19.76, 15.2, LoadBalanced},
		{0, 5, LoadUnderloaded},
	}
	for _, c := range cases {
		if got := LoadStatus(c.active, c.avg); got != c.want {
			t.Errorf("LoadStatus(%v, %v) = %s, want %s", c.active, c.avg, got, c.want)
		}
	}
}

func TestTeamAverage_CountsIdleMembers(t *testing.T) {
	if got := TeamAverage([]float64{10, 0, 0, 2}); got != 3 {
		t.Fatalf("got %v, want 3", got)
	}
	if TeamAverage(nil) != 0 {
		t.Fatal("empty team must average 0")
	}
}

func TestThresholdsFile(t *testing.T) {
	if LoadThresholds.OverloadedAbove != 1.3 || LoadThresholds.UnderloadedBelow != 0.75 {
		t.Fatalf("unexpected thresholds %+v", LoadThresholds)
	}
}

func TestRaciCheck(t *testing.T) {
	cases := []struct {
		vals []string
		want string
	}{
		{[]string{"A", "R", "C", "I"}, RaciOK},
		{[]string{"R", "R", "C"}, RaciNoA},
		{[]string{"A", "A", "R"}, RaciSeveralA},
		{[]string{"A", "C", "I", ""}, RaciNoR},
		{[]string{}, RaciNoA},
		{[]string{"A", "R", "R", "R"}, RaciOK},
	}
	for _, c := range cases {
		if got := RaciCheck(c.vals); got != c.want {
			t.Errorf("RaciCheck(%v) = %q, want %q", c.vals, got, c.want)
		}
	}
	for _, v := range []string{"", "R", "A", "C", "I"} {
		if !ValidRaciValue(v) {
			t.Errorf("%q should be valid", v)
		}
	}
	for _, v := range []string{"X", "r", "RA"} {
		if ValidRaciValue(v) {
			t.Errorf("%q should be invalid", v)
		}
	}
}

func TestInviteCode(t *testing.T) {
	seen := map[string]bool{}
	for range 500 {
		c := NewInviteCode()
		if !ValidInviteCode(c) {
			t.Fatalf("generated invalid code %q", c)
		}
		if strings.ContainsAny(c, "IO01") {
			t.Fatalf("code %q has a look-alike character", c)
		}
		seen[c] = true
	}
	if len(seen) < 490 {
		t.Fatalf("too many collisions: %d unique of 500", len(seen))
	}
	for _, bad := range []string{"", "ABC", "ABCDEFG", "ABCDE1", "abcdef", "ABCDEO"} {
		if ValidInviteCode(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
	if NormalizeInviteCode(" k7q2pm ") != "K7Q2PM" {
		t.Fatal("normalize must trim and upper-case")
	}
}

func TestValidateRegistration(t *testing.T) {
	ok := Registration{FullName: "Aruzhan Sadykova", Email: "Aruzhan.S@stu.sdu.edu.kz", Password: "studyspace26", Consent: true}
	if e := ValidateRegistration(ok); !e.OK() {
		t.Fatalf("valid form rejected: %v", e)
	}
	e := ValidateRegistration(Registration{FullName: "A", Email: "not-an-email", Password: "short", Consent: false})
	for _, f := range []string{"full_name", "email", "password", "consent"} {
		if e[f] == "" {
			t.Errorf("expected error on %s", f)
		}
	}
	if e["consent"] != MsgNoConsent {
		t.Errorf("consent message = %q", e["consent"])
	}
	for _, bad := range []string{"a@b", "a b@c.kz", "<a@b.kz>", "@b.kz"} {
		if ValidEmail(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
	if NormalizeEmail("  Foo@Bar.KZ ") != "foo@bar.kz" {
		t.Fatal("email must be lower-cased and trimmed")
	}
}

var today = time.Date(2026, 10, 22, 15, 0, 0, 0, time.UTC)

func desc(n int) string { return strings.Repeat("x", n) }

func TestValidateNewProject(t *testing.T) {
	p := ProjectDetails{Title: "StudySpace", Description: desc(100), Deadline: today.AddDate(0, 0, 7)}
	if e := ValidateNewProject(p, 5, today); !e.OK() {
		t.Fatalf("valid project rejected: %v", e)
	}
	bad := ProjectDetails{Title: " ", Description: desc(99), Deadline: today.AddDate(0, 0, 6), Context: desc(101)}
	e := ValidateNewProject(bad, 7, today)
	for _, f := range []string{"title", "description", "deadline", "team_size", "context"} {
		if e[f] == "" {
			t.Errorf("expected error on %s", f)
		}
	}
	if e := ValidateNewProject(ProjectDetails{Title: desc(101), Description: desc(3001), Deadline: today.AddDate(0, 1, 0)}, 2, today); e["title"] == "" || e["description"] == "" || e["team_size"] == "" {
		t.Errorf("upper bounds not enforced: %v", e)
	}
	for _, n := range []int{3, 6} {
		if e := ValidateNewProject(p, n, today); !e.OK() {
			t.Errorf("team size %d rejected", n)
		}
	}
}

func TestValidateProjectUpdate_AllowsToday(t *testing.T) {
	p := ProjectDetails{Title: "T", Description: desc(120), Deadline: today}
	if e := ValidateProjectUpdate(p, today); !e.OK() {
		t.Fatalf("today rejected: %v", e)
	}
	p.Deadline = today.AddDate(0, 0, -1)
	if e := ValidateProjectUpdate(p, today); e["deadline"] != MsgDeadlinePast {
		t.Fatalf("past deadline accepted: %v", e)
	}
}

func TestValidateProfile(t *testing.T) {
	cases := []struct {
		name      string
		s, w      string
		wantShort bool
	}{
		{"ok", desc(70), desc(30), false},
		{"strengths too short", desc(29), desc(80), true},
		{"weaknesses too short", desc(95), desc(9), true},
		{"total under 100", desc(60), desc(39), true},
		{"exactly 100", desc(90), desc(10), false},
	}
	for _, c := range cases {
		e := ValidateProfile(Profile{Strengths: c.s, Weaknesses: c.w})
		if got := e["strengths"] == MsgProfileTooShort; got != c.wantShort {
			t.Errorf("%s: short=%v, want %v (%v)", c.name, got, c.wantShort, e)
		}
	}
	if e := ValidateProfile(Profile{Strengths: desc(1001), Weaknesses: desc(20)}); e["strengths"] != MsgProfileTooLong {
		t.Errorf("1001 chars accepted: %v", e)
	}
	tags := make([]string, 11)
	for i := range tags {
		tags[i] = "t" + string(rune('a'+i))
	}
	if e := ValidateProfile(Profile{Strengths: desc(70), Weaknesses: desc(30), Tags: tags}); e["tags"] != MsgTooManyTags {
		t.Errorf("11 tags accepted: %v", e)
	}
	if e := ValidateProfile(Profile{Strengths: desc(70), Weaknesses: desc(30), Tags: []string{desc(25)}}); e["tags"] != MsgTagLength {
		t.Errorf("25-char tag accepted: %v", e)
	}
	if e := ValidateProfile(Profile{Strengths: desc(70), Weaknesses: desc(30), Tags: []string{"Go", "go"}}); e["tags"] != MsgDuplicateTag {
		t.Errorf("duplicate tag accepted: %v", e)
	}
}

func TestValidateTask(t *testing.T) {
	if ValidateTaskTitle("  ") != MsgTaskTitleRequired {
		t.Error("empty title accepted")
	}
	if ValidateTaskTitle(desc(121)) != MsgTaskTitleLong {
		t.Error("121-char title accepted")
	}
	if ValidateTaskTitle("Booking API") != "" {
		t.Error("valid title rejected")
	}
	f := func(v float64) *float64 { return &v }
	cases := []struct {
		in   *float64
		want string
		out  float64
	}{
		{nil, MsgEstimateRequired, 0},
		{f(0), MsgEstimateRequired, 0},
		{f(-2), MsgEstimateRequired, 0},
		{f(0.25), MsgEstimateRange, 0},
		{f(40.5), MsgEstimateRange, 0},
		{f(0.5), "", 0.5},
		{f(40), "", 40},
		{f(2.3), "", 2.5},
	}
	for _, c := range cases {
		got, msg := ValidateEstimate(c.in)
		if msg != c.want || got != c.out {
			t.Errorf("ValidateEstimate(%v) = %v, %q; want %v, %q", c.in, got, msg, c.out, c.want)
		}
	}
	for _, s := range []string{"todo", "doing", "done"} {
		if !ValidStatus(s) {
			t.Errorf("%s should be valid", s)
		}
	}
	if ValidStatus("blocked") {
		t.Error("unknown status accepted")
	}
}

func TestValidateSeatEdits(t *testing.T) {
	seats := []SeatEdit{{"PM", "", 20}, {"Backend", "", 30}, {"Frontend", "", 30}, {"Design", "", 20}}
	if e := ValidateSeatEdits(seats, 4); !e.OK() {
		t.Fatalf("valid plan rejected: %v", e)
	}
	seats[0].EffortShare = 25
	if e := ValidateSeatEdits(seats, 4); e["total"] != MsgShareTotal {
		t.Errorf("sum 105 accepted: %v", e)
	}
	seats[0] = SeatEdit{"", desc(141), 4}
	e := ValidateSeatEdits(seats, 4)
	for _, f := range []string{"name", "summary", "effort_share"} {
		if e[f] == "" {
			t.Errorf("expected error on %s", f)
		}
	}
	if e := ValidateSeatEdits(seats[:3], 4); e["seats"] == "" {
		t.Error("wrong seat count accepted")
	}
}

func TestDaysAndWeeksLeft(t *testing.T) {
	dl := time.Date(2026, 12, 11, 0, 0, 0, 0, time.UTC)
	if d := DaysLeft(dl, today); d != 50 {
		t.Errorf("DaysLeft = %d, want 50", d)
	}
	if w := WeeksLeft(dl, today); w != 7 {
		t.Errorf("WeeksLeft = %d, want 7", w)
	}
	if DaysLeft(today.AddDate(0, 0, -3), today) != 0 {
		t.Error("past deadline must give 0")
	}
}

func TestNames(t *testing.T) {
	if Initials("Aruzhan Sadykova") != "AS" || Initials("dana") != "D" || Initials("A B C") != "AB" {
		t.Error("initials")
	}
	if FirstName("Daniyar Kenzhebekov") != "Daniyar" {
		t.Error("first name")
	}
	if SeatColor(1) != "--r1" || SeatColor(6) != "--r6" {
		t.Error("seat colour")
	}
}
