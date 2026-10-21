package ai

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Output limits (spec 6.4 and 7.5).
const (
	maxSeatName  = 40
	maxSummary   = 140
	maxResp      = 80
	maxSkill     = 30
	maxWhy       = 300
	minListItems = 3
	maxListItems = 5
	minShare     = 5
	maxShare     = 60
	maxExplain   = 600
	maxGapSkill  = 60
	maxGapReason = 300
	maxGrowth    = 400
	minRaciRows  = 6
	maxRaciRows  = 10
	maxActivity  = 120
)

var personLabel = regexp.MustCompile(`\bP\d+\b`)

// SeatIDs returns S1..Sn.
func SeatIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("S%d", i+1)
	}
	return ids
}

func textLen(s string) int { return utf8.RuneCountInString(strings.TrimSpace(s)) }

func checkText(errs *[]string, what, s string, maxLen int) {
	if n := textLen(s); n == 0 || n > maxLen {
		*errs = append(*errs, fmt.Sprintf("%s must be 1-%d characters (got %d)", what, maxLen, n))
	}
}

func checkList(errs *[]string, what string, items []string, maxLen int) {
	if len(items) < minListItems || len(items) > maxListItems {
		*errs = append(*errs, fmt.Sprintf("%s must have %d-%d items (got %d)", what, minListItems, maxListItems, len(items)))
	}
	for i, it := range items {
		checkText(errs, fmt.Sprintf("%s[%d]", what, i), it, maxLen)
	}
}

// ValidateRolePlan checks a role plan for a team of teamSize.
func ValidateRolePlan(out RolePlanOutput, teamSize int) []string {
	var errs []string
	if len(out.Seats) != teamSize {
		errs = append(errs, fmt.Sprintf("seats must contain exactly %d seats (got %d)", teamSize, len(out.Seats)))
	}
	sum := 0
	for i, s := range out.Seats {
		p := fmt.Sprintf("seats[%d]", i)
		if want := fmt.Sprintf("S%d", i+1); s.ID != want {
			errs = append(errs, fmt.Sprintf("%s.id must be %q (got %q)", p, want, s.ID))
		}
		checkText(&errs, p+".name", s.Name, maxSeatName)
		checkText(&errs, p+".summary", s.Summary, maxSummary)
		checkList(&errs, p+".responsibilities", s.Responsibilities, maxResp)
		checkList(&errs, p+".skills", s.Skills, maxSkill)
		checkText(&errs, p+".why", s.Why, maxWhy)
		if s.EffortShare < minShare || s.EffortShare > maxShare {
			errs = append(errs, fmt.Sprintf("%s.effort_share must be an integer %d-%d (got %d)", p, minShare, maxShare, s.EffortShare))
		}
		sum += s.EffortShare
	}
	if sum != 100 {
		errs = append(errs, fmt.Sprintf("effort_share values must sum to exactly 100 (got %d)", sum))
	}
	return errs
}

// sentenceCount counts sentences ending in . ! or ? (or the final fragment).
func sentenceCount(s string) int {
	runes := []rune(strings.TrimRight(strings.TrimSpace(s), "\"”’"))
	if len(runes) == 0 {
		return 0
	}
	n := 0
	for i, r := range runes {
		if r == '.' || r == '!' || r == '?' {
			end := i == len(runes)-1
			if !end {
				next := runes[i+1]
				end = next == ' ' || next == '\n' || next == '"' || next == '”' || next == '’'
			}
			if end {
				n++
			}
		}
	}
	if last := runes[len(runes)-1]; last != '.' && last != '!' && last != '?' {
		n++
	}
	return n
}

var quoted = regexp.MustCompile(`[\x{201C}"\x{2018}]([^\x{201D}"\x{2019}]{3,})[\x{201D}"\x{2019}]`)

func normalizeForQuote(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("’", "'", "‘", "'", "\n", " ", "\t", " ").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// quotesProfile reports whether expl contains a quoted fragment taken from
// the person's own profile text or tags.
func quotesProfile(expl string, p Person) bool {
	source := normalizeForQuote(p.Strengths + " \n " + p.Weaknesses + " \n " + strings.Join(p.Tags, " \n "))
	for _, m := range quoted.FindAllStringSubmatch(expl, -1) {
		frag := strings.Trim(normalizeForQuote(m[1]), " .,;:!?…")
		if utf8.RuneCountInString(frag) >= 3 && strings.Contains(source, frag) {
			return true
		}
	}
	return false
}

var youWord = regexp.MustCompile(`(?i)\b(you|your)\b`)

// ValidateMatch checks a matching result against its input.
func ValidateMatch(out MatchOutput, in MatchInput) []string {
	var errs []string
	seats := map[string]bool{}
	for _, s := range in.Seats {
		seats[s.ID] = true
	}
	people := map[string]Person{}
	for _, p := range in.People {
		people[p.Label] = p
	}
	if len(out.Assignments) != len(in.People) {
		errs = append(errs, fmt.Sprintf("assignments must contain every person exactly once (%d people, got %d)", len(in.People), len(out.Assignments)))
	}
	seenP, seenS := map[string]bool{}, map[string]bool{}
	for i, a := range out.Assignments {
		p := fmt.Sprintf("assignments[%d]", i)
		person, ok := people[a.Person]
		switch {
		case !ok:
			errs = append(errs, fmt.Sprintf("%s.person %q is not one of the given labels", p, a.Person))
		case seenP[a.Person]:
			errs = append(errs, fmt.Sprintf("%s.person %q appears more than once", p, a.Person))
		}
		seenP[a.Person] = true
		switch {
		case !seats[a.SeatID]:
			errs = append(errs, fmt.Sprintf("%s.seat_id %q is not one of the given seats", p, a.SeatID))
		case seenS[a.SeatID]:
			errs = append(errs, fmt.Sprintf("%s.seat_id %q is used more than once", p, a.SeatID))
		}
		seenS[a.SeatID] = true
		if a.Fit < 0 || a.Fit > 100 {
			errs = append(errs, fmt.Sprintf("%s.fit must be 0-100 (got %d)", p, a.Fit))
		}
		if len(in.Seats) > 1 && (!seats[a.RunnerUpSeatID] || a.RunnerUpSeatID == a.SeatID) {
			errs = append(errs, fmt.Sprintf("%s.runner_up_seat_id must be another given seat", p))
		}
		checkText(&errs, p+".explanation", a.Explanation, maxExplain)
		if textLen(a.Explanation) > 0 {
			if n := sentenceCount(a.Explanation); n < 2 || n > 3 {
				errs = append(errs, fmt.Sprintf("%s.explanation must be 2-3 sentences (got %d)", p, n))
			}
			if personLabel.MatchString(a.Explanation) {
				errs = append(errs, fmt.Sprintf("%s.explanation must not contain person labels like P1; address the person as \"you\"", p))
			}
			if !youWord.MatchString(a.Explanation) {
				errs = append(errs, fmt.Sprintf("%s.explanation must address the person as \"you\"", p))
			}
			if ok && !quotesProfile(a.Explanation, person) {
				errs = append(errs, fmt.Sprintf("%s.explanation must quote the person's own words from their profile in quotation marks", p))
			}
		}
	}
	for i, g := range out.Gaps {
		p := fmt.Sprintf("gaps[%d]", i)
		checkText(&errs, p+".skill", g.Skill, maxGapSkill)
		checkText(&errs, p+".reason", g.Reason, maxGapReason)
		if !seats[g.SeatID] {
			errs = append(errs, fmt.Sprintf("%s.seat_id %q is not one of the given seats", p, g.SeatID))
		}
		if personLabel.MatchString(g.Reason) {
			errs = append(errs, p+".reason must not contain person labels like P1")
		}
	}
	if out.GrowthNote != nil {
		if n := textLen(*out.GrowthNote); n > maxGrowth {
			errs = append(errs, fmt.Sprintf("growth_note must be at most %d characters", maxGrowth))
		}
		if personLabel.MatchString(*out.GrowthNote) {
			errs = append(errs, "growth_note must not contain person labels like P1; refer to people by their role")
		}
	}
	return errs
}

// ValidateRACI checks a RACI draft for the given seat IDs.
func ValidateRACI(out RACIOutput, seatIDs []string) []string {
	var errs []string
	if n := len(out.Activities); n < minRaciRows || n > maxRaciRows {
		errs = append(errs, fmt.Sprintf("activities must contain %d-%d items (got %d)", minRaciRows, maxRaciRows, n))
	}
	valid := map[string]bool{}
	for _, id := range seatIDs {
		valid[id] = true
	}
	for i, a := range out.Activities {
		p := fmt.Sprintf("activities[%d]", i)
		checkText(&errs, p+".name", a.Name, maxActivity)
		as, rs := 0, 0
		for id, v := range a.Cells {
			if !valid[id] {
				errs = append(errs, fmt.Sprintf("%s.cells has unknown seat %q", p, id))
			}
			switch v {
			case "A":
				as++
			case "R":
				rs++
			case "C", "I", "":
			default:
				errs = append(errs, fmt.Sprintf("%s.cells[%s] must be R, A, C, I or empty (got %q)", p, id, v))
			}
		}
		if as != 1 {
			errs = append(errs, fmt.Sprintf("%s must have exactly one A (got %d)", p, as))
		}
		if rs < 1 {
			errs = append(errs, p+" must have at least one R")
		}
	}
	return errs
}
