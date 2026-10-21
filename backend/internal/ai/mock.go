package ai

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Mock is the default provider: realistic, input-dependent output without
// an API key (spec 7.6). It is deterministic for the same input.
type Mock struct {
	// Delay makes the loading overlay visible in the demo.
	Delay time.Duration
}

func (m Mock) wait(ctx context.Context) error {
	if m.Delay <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(m.Delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ---- text helpers ----

var wordRe = regexp.MustCompile(`[\p{L}\p{N}][\p{L}\p{N}+#./-]*`)

// matchedStems returns the keywords found in text (lower-case, padded).
func matchedStems(text string, keywords []string) []string {
	t := " " + strings.ToLower(text) + " "
	var out []string
	seen := map[string]bool{}
	for _, k := range keywords {
		if !seen[k] && strings.Contains(t, k) {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

// wordWith returns the first word of text that contains the stem.
func wordWith(text, stem string) string {
	stem = strings.TrimSpace(stem)
	for _, w := range wordRe.FindAllString(text, -1) {
		w = strings.TrimRight(w, ".,/-")
		if strings.Contains(strings.ToLower(w), stem) {
			return w
		}
	}
	return ""
}

// clauses splits profile text into quotable pieces without sentence marks.
func clauses(text string) []string {
	parts := regexp.MustCompile(`[.!?;\n]+`).Split(text, -1)
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len([]rune(p)) >= 8 {
			out = append(out, p)
		}
	}
	return out
}

// trimWords shortens s to at most n runes on a word boundary.
func trimWords(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := string(r[:n])
	if i := strings.LastIndex(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,:-")
}

func lowerFirst(s string) string {
	r := []rune(s)
	if len(r) > 1 && unicode.IsUpper(r[0]) && !unicode.IsUpper(r[1]) {
		r[0] = unicode.ToLower(r[0])
	}
	return string(r)
}

// ---- role plan ----

var baseWeight = map[string]float64{
	pmQA.Key: 16, RolePM: 16, RoleBackend: 24, RoleFrontend: 22, RoleDesign: 16, RoleQA: 14, RoleData: 18,
}

// noteWords map an owner note to the roles it asks to strengthen.
var noteWords = map[string][]string{
	RolePM:       {"manag", "plan", "organis", "organiz"},
	RoleBackend:  {"backend", "api", "database", "server"},
	RoleFrontend: {"frontend", "interface", "ui", "mobile"},
	RoleDesign:   {"design", "ux", "user research", "usab"},
	RoleQA:       {"test", "qa", "quality", "bug"},
	RoleData:     {"data", "analytic", "ai", "ml", "statistic"},
}

func noteWants(note, role string) bool { return len(matchedStems(note, noteWords[role])) > 0 }

// GenerateRolePlan sizes the catalogue to the team and tailors which roles
// appear, their shares and rationales to the description and the note.
func (m Mock) GenerateRolePlan(ctx context.Context, in RolePlanInput) (RolePlanOutput, error) {
	if err := m.wait(ctx); err != nil {
		return RolePlanOutput{}, err
	}
	text := in.Title + " " + in.Description
	note := strings.ToLower(in.Note)
	hits := func(r Role) []string { return matchedStems(text, r.Keywords) }
	c := Catalogue
	dataHits, designHits := len(hits(c[RoleData])), len(hits(c[RoleDesign]))

	var roles []Role
	switch in.TeamSize {
	case 3:
		roles = []Role{pmQA, c[RoleBackend], c[RoleFrontend]}
	case 4:
		fourth := c[RoleDesign]
		if (dataHits > designHits+1 || noteWants(note, RoleData)) && !noteWants(note, RoleDesign) {
			fourth = c[RoleData]
		}
		roles = []Role{pmQA, c[RoleBackend], c[RoleFrontend], fourth}
	case 5:
		if dataHits == 0 && !noteWants(note, RoleData) {
			roles = []Role{c[RolePM], c[RoleBackend], c[RoleFrontend], c[RoleDesign], c[RoleQA]}
		} else {
			roles = []Role{pmQA, c[RoleBackend], c[RoleFrontend], c[RoleDesign], c[RoleData]}
		}
	default:
		roles = standardRoles(6)
	}

	weights := make([]float64, len(roles))
	whys := make([]string, len(roles))
	for i, r := range roles {
		h := hits(r)
		weights[i] = baseWeight[r.Key] + math.Min(10, 2*float64(len(h)))
		why := r.Why
		switch {
		case r.Key == RolePM || r.Key == pmQA.Key:
			why = fmt.Sprintf("A %d-person team with a %d-week deadline needs one owner for the plan and the demos.", in.TeamSize, in.WeeksLeft)
			if r.Key == pmQA.Key {
				why += " QA is combined with PM because the team is too small for a full-time tester."
			}
		case len(h) > 0:
			if w := wordWith(text, h[0]); w != "" {
				why = fmt.Sprintf("The description mentions “%s”. %s", w, r.Why)
			}
		}
		key := r.Key
		if key == pmQA.Key {
			key = RolePM
		}
		if note != "" && noteWants(note, key) {
			weights[i] += 8
			why = trimWords(why, 260) + " Your note asked for more focus here."
		}
		whys[i] = trimWords(why, maxWhy)
	}
	shares := normalizeShares(weights)
	out := RolePlanOutput{}
	for i, r := range roles {
		out.Seats = append(out.Seats, roleSeat(i, r, shares[i], whys[i]))
	}
	return out, nil
}

// normalizeShares turns weights into integers 5-60 that sum to 100
// (largest-remainder rounding).
func normalizeShares(w []float64) []int {
	var sum float64
	for _, v := range w {
		sum += v
	}
	out := make([]int, len(w))
	type rem struct {
		i int
		r float64
	}
	rems := make([]rem, len(w))
	total := 0
	for i, v := range w {
		exact := v / sum * 100
		out[i] = int(math.Floor(exact))
		rems[i] = rem{i, exact - float64(out[i])}
		total += out[i]
	}
	sort.SliceStable(rems, func(a, b int) bool { return rems[a].r > rems[b].r })
	for k := 0; total < 100; k++ {
		out[rems[k%len(rems)].i]++
		total++
	}
	for i := range out {
		for out[i] < minShare {
			out[i]++
			out[maxIndex(out)]--
		}
		for out[i] > maxShare {
			out[i]--
			out[minIndex(out)]++
		}
	}
	return out
}

func maxIndex(v []int) int {
	m := 0
	for i := range v {
		if v[i] > v[m] {
			m = i
		}
	}
	return m
}

func minIndex(v []int) int {
	m := 0
	for i := range v {
		if v[i] < v[m] {
			m = i
		}
	}
	return m
}

// ---- matching ----

func seatKeywords(s SeatBrief) []string {
	var kws []string
	for _, k := range RoleKeysForSeat(s.Name) {
		kws = append(kws, Catalogue[k].Keywords...)
	}
	for _, sk := range s.Skills {
		kws = append(kws, strings.ToLower(sk))
	}
	return kws
}

// skillMentioned reports whether a skill appears as a whole phrase in text.
func skillMentioned(text, skill string) bool {
	re, err := regexp.Compile(`(?i)(^|[^\p{L}\p{N}])` + regexp.QuoteMeta(strings.TrimSpace(skill)) + `($|[^\p{L}\p{N}])`)
	return err == nil && re.MatchString(text)
}

type scored struct {
	score int
	hits  []string
}

func scorePerson(p Person, s SeatBrief) scored {
	kws := seatKeywords(s)
	strong := matchedStems(p.Strengths, kws)
	tagHits := matchedStems(strings.Join(p.Tags, " , "), kws)
	weak := matchedStems(p.Weaknesses, kws)
	return scored{score: len(strong) + 2*len(tagHits) - 2*len(weak), hits: strong}
}

func fitFromScore(score int) int {
	if score <= 0 {
		return max(25, 40+5*score)
	}
	return 40 + int(math.Round(57*(1-math.Exp(-float64(score)/8))))
}

// bestAssignment maximises the total score (one seat per person). People are
// visited in canonical (content) order so shuffling the input never changes
// the result.
func bestAssignment(score [][]int, order []int, nSeats int) []int {
	best := make([]int, len(score))
	cur := make([]int, len(score))
	used := make([]bool, nSeats)
	bestTotal := math.MinInt
	var rec func(k, total int)
	rec = func(k, total int) {
		if k == len(order) {
			if total > bestTotal {
				bestTotal = total
				copy(best, cur)
			}
			return
		}
		p := order[k]
		for s := range nSeats {
			if used[s] {
				continue
			}
			used[s] = true
			cur[p] = s
			rec(k+1, total+score[p][s])
			used[s] = false
		}
	}
	rec(0, 0)
	return best
}

func personKey(p Person) string {
	return p.Strengths + "\x00" + p.Weaknesses + "\x00" + strings.Join(p.Tags, ",")
}

// MatchRoles scores keyword overlap between each profile and each seat,
// finds the optimal one-to-one assignment and explains it by quoting the
// profile.
func (m Mock) MatchRoles(ctx context.Context, in MatchInput) (MatchOutput, error) {
	if err := m.wait(ctx); err != nil {
		return MatchOutput{}, err
	}
	nP, nS := len(in.People), len(in.Seats)
	sc := make([][]scored, nP)
	score := make([][]int, nP)
	for i, p := range in.People {
		sc[i] = make([]scored, nS)
		score[i] = make([]int, nS)
		for j, s := range in.Seats {
			sc[i][j] = scorePerson(p, s)
			score[i][j] = sc[i][j].score
		}
	}
	order := make([]int, nP)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return personKey(in.People[order[a]]) < personKey(in.People[order[b]]) })
	assign := bestAssignment(score, order, nS)

	out := MatchOutput{Gaps: []Gap{}}
	for i, p := range in.People {
		j := assign[i]
		seat := in.Seats[j]
		runner := -1
		for k := range in.Seats {
			if k != j && (runner < 0 || score[i][k] > score[i][runner]) {
				runner = k
			}
		}
		a := Assignment{SeatID: seat.ID, Person: p.Label, Fit: fitFromScore(score[i][j]), Explanation: explain(p, seat, sc[i][j])}
		if runner >= 0 {
			a.RunnerUpSeatID = in.Seats[runner].ID
		}
		out.Assignments = append(out.Assignments, a)
	}
	out.Gaps = findGaps(in)
	out.GrowthNote = growthNote(in, assign)
	return out, nil
}

// explain writes 2 sentences that quote the person's own words.
func explain(p Person, s SeatBrief, sc scored) string {
	kws := seatKeywords(s)
	cl := clauses(p.Strengths)
	if len(cl) == 0 {
		cl = clauses(p.Strengths + ". " + p.Weaknesses)
	}
	best, bestHits := "", -1
	for _, c := range cl {
		if h := len(matchedStems(c, kws)); h > bestHits {
			best, bestHits = c, h
		}
	}
	if best == "" {
		best = strings.TrimSpace(p.Strengths)
	}
	quote := trimWords(best, 90)
	lead := "You write “" + quote + "”"
	if strings.HasPrefix(quote, "I ") && len(quote) > 6 {
		lead = "You write that you “" + trimWords(quote[2:], 88) + "”"
	}

	var skills []string
	for _, sk := range s.Skills {
		if skillMentioned(p.Strengths+" "+strings.Join(p.Tags, ", "), sk) {
			skills = append(skills, sk)
		}
	}
	if sc.score <= 0 || bestHits <= 0 && len(skills) == 0 {
		return fmt.Sprintf("%s, but few of the %s skills appear in your profile yet. The team can still give you this role as a chance to learn.", lead, s.Name)
	}
	first := fmt.Sprintf("%s, which is the core of the %s role.", lead, s.Name)
	switch len(skills) {
	case 0:
		return first + " It is the closest match to your profile among the open roles."
	case 1:
		return first + fmt.Sprintf(" Your profile also lists %s, which this role needs.", skills[0])
	default:
		return first + fmt.Sprintf(" Your profile also lists %s and %s, which this role needs.", strings.Join(skills[:len(skills)-1], ", "), skills[len(skills)-1])
	}
}

// findGaps lists up to two seat skills nobody mentions in strengths or tags,
// starting with the seats that carry the most effort.
func findGaps(in MatchInput) []Gap {
	var all strings.Builder
	for _, p := range in.People {
		all.WriteString(p.Strengths + " \n " + strings.Join(p.Tags, " , ") + " \n ")
	}
	seats := append([]SeatBrief{}, in.Seats...)
	sort.SliceStable(seats, func(a, b int) bool { return seats[a].EffortShare > seats[b].EffortShare })
	gaps := []Gap{}
	for _, s := range seats {
		for _, sk := range s.Skills {
			if skillMentioned(all.String(), sk) {
				continue
			}
			need := "its work"
			if len(s.Responsibilities) > 0 {
				need = "the " + lowerFirst(s.Responsibilities[len(s.Responsibilities)-1])
			}
			gaps = append(gaps, Gap{Skill: sk, SeatID: s.ID,
				Reason: trimWords(fmt.Sprintf("The %s role needs it for %s. Consider learning resources or asking the instructor.", s.Name, need), maxGapReason)})
			break
		}
		if len(gaps) == 2 {
			break
		}
	}
	return gaps
}

var learnRe = regexp.MustCompile(`(?i)(want to learn|would like to learn|curious|interested in|learn more)`)

// growthNote suggests pairing someone with the seat they want to learn about.
func growthNote(in MatchInput, assign []int) *string {
	for i, p := range in.People {
		for _, c := range clauses(p.Strengths + ". " + p.Weaknesses) {
			if !learnRe.MatchString(c) {
				continue
			}
			mine := in.Seats[assign[i]]
			for j, s := range in.Seats {
				if j == assign[i] {
					continue
				}
				h := matchedStems(c, seatKeywords(s))
				if len(h) == 0 {
					continue
				}
				topic := wordWith(c, h[0])
				if topic == "" {
					topic = strings.TrimSpace(h[0])
				}
				note := fmt.Sprintf("The %s is curious about %s. Pairing them with the %s on one %s task would give that practice without moving them from the role they fit best.",
					mine.Name, topic, s.Name, topic)
				note = trimWords(note, maxGrowth)
				return &note
			}
		}
	}
	return nil
}

// ---- RACI ----

type raciRow struct {
	name   string
	a      string   // accountable role
	r      []string // responsible roles, first available wins at least one
	c      []string
	needed string // only when this role has a seat
}

var raciTemplate = []raciRow{
	{name: "Define scope & backlog", a: RolePM, r: []string{RoleDesign, RoleFrontend, RoleBackend}, c: []string{RoleBackend, RoleFrontend}},
	{name: "User interviews & wireframes", a: RoleDesign, r: []string{RolePM}, c: []string{RoleFrontend}, needed: RoleDesign},
	{name: "Database & API", a: RoleBackend, r: []string{RoleData, RoleFrontend}, c: []string{RoleFrontend}},
	{name: "Main screens & user flows", a: RoleFrontend, r: []string{RoleDesign, RoleBackend}, c: []string{RoleBackend}},
	{name: "Data, analytics & integrations", a: RoleData, r: []string{RoleBackend, RoleFrontend}, c: []string{RoleDesign}, needed: RoleData},
	{name: "Deployment & demo environment", a: RoleBackend, r: []string{RoleFrontend, RoleData}, c: []string{RolePM}},
	{name: "Testing & bug fixing", a: RoleQA, r: []string{RoleBackend, RoleFrontend, RoleData}, c: []string{RoleDesign}},
	{name: "Demo & final report", a: RolePM, r: []string{RoleDesign, RoleQA, RoleFrontend}, c: []string{RoleBackend, RoleFrontend, RoleData}},
}

// DraftRACI builds a timeline of activities from the seats' roles.
func (m Mock) DraftRACI(ctx context.Context, in RACIInput) (RACIOutput, error) {
	if err := m.wait(ctx); err != nil {
		return RACIOutput{}, err
	}
	seatOf := map[string]string{} // role → first seat ID
	for _, s := range in.Seats {
		for _, k := range RoleKeysForSeat(s.Name) {
			if _, ok := seatOf[k]; !ok {
				seatOf[k] = s.ID
			}
		}
	}
	pick := func(roles ...string) string {
		for _, r := range roles {
			if id, ok := seatOf[r]; ok {
				return id
			}
		}
		return ""
	}
	out := RACIOutput{}
	for _, t := range raciTemplate {
		if t.needed != "" && seatOf[t.needed] == "" {
			continue
		}
		cells := map[string]string{}
		for _, s := range in.Seats {
			cells[s.ID] = "I"
		}
		a := pick(t.a, RolePM, RoleQA)
		if a == "" {
			a = in.Seats[0].ID
		}
		if t.a == RoleQA && seatOf[RoleQA] == "" {
			a = pick(RolePM)
		}
		cells[a] = "A"
		rs := 0
		for _, r := range t.r {
			if id := pick(r); id != "" && cells[id] != "A" {
				cells[id] = "R"
				rs++
			}
		}
		if rs == 0 {
			for _, s := range in.Seats {
				if cells[s.ID] != "A" {
					cells[s.ID] = "R"
					break
				}
			}
		}
		for _, c := range t.c {
			if id := pick(c); id != "" && cells[id] == "I" {
				cells[id] = "C"
			}
		}
		out.Activities = append(out.Activities, RACIActivity{Name: t.name, Cells: cells})
	}
	return out, nil
}
