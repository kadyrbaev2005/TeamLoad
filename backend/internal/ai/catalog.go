package ai

import (
	"fmt"
	"strings"
)

// Role is a catalogue role (spec 7.3). The AI may combine or specialise
// these; the mock provider and the standard plan use them directly.
type Role struct {
	Key              string
	Name             string
	Summary          string
	Responsibilities []string
	Skills           []string
	Why              string
	// Keywords are lower-case stems the mock uses to read descriptions and profiles.
	Keywords []string
}

// Catalogue keys.
const (
	RolePM       = "pm"
	RoleBackend  = "backend"
	RoleFrontend = "frontend"
	RoleDesign   = "design"
	RoleQA       = "qa"
	RoleData     = "data"
)

// Catalogue is the role catalogue used as guidance.
var Catalogue = map[string]Role{
	RolePM: {
		Key: RolePM, Name: "Project Manager",
		Summary:          "Owns the plan, the deadline and the team's communication.",
		Responsibilities: []string{"Sprint plan and weekly sync", "Communication with the instructor", "Risk and scope tracking", "Progress reports and final demo"},
		Skills:           []string{"Planning", "Communication", "Documentation", "Presenting"},
		Why:              "Every student team needs one person who owns the plan, the deadline and the talks with the instructor.",
		Keywords:         []string{"plan", "organis", "organiz", "deadline", "communicat", "present", "lead", "manag", "schedul", "notion", "jira", "trello", "report", "coordinat", "meeting", "instructor"},
	},
	RoleBackend: {
		Key: RoleBackend, Name: "Backend Developer",
		Summary:          "Builds the API, the database and authentication.",
		Responsibilities: []string{"Database schema and migrations", "REST API endpoints", "Register / login and permissions", "Docker setup and seed data"},
		Skills:           []string{"Go", "PostgreSQL", "REST API", "Auth"},
		Why:              "Every feature in the description needs data that is stored safely and served by a reliable API.",
		Keywords:         []string{"backend", "back-end", "api", "rest", "golang", " go ", "go,", "java", "spring", "node", "express", "django", "flask", "sql", "postgres", "mysql", "database", "docker", "server", "algorithm", "data model", "endpoint", "auth"},
	},
	RoleFrontend: {
		Key: RoleFrontend, Name: "Frontend Developer",
		Summary:          "Turns the main flows into a fast, responsive interface.",
		Responsibilities: []string{"Main screens and navigation", "Forms and validation", "API integration on the client", "Responsive layout for phones"},
		Skills:           []string{"React", "TypeScript", "CSS", "Responsive design"},
		Why:              "Users only see the interface, so it has to be quick, clear and work on phones.",
		Keywords:         []string{"frontend", "front-end", "react", "typescript", "javascript", "vue", "angular", "svelte", "css", "html", "tailwind", "responsive", "web page", "pages", "interface", "component"},
	},
	RoleDesign: {
		Key: RoleDesign, Name: "UX/UI Designer",
		Summary:          "Designs the screens and checks them with real users.",
		Responsibilities: []string{"User interviews", "Wireframes and prototypes", "Usability test with 5 users", "Design handoff to frontend"},
		Skills:           []string{"Figma", "User research", "UI design", "Accessibility"},
		Why:              "A clean, easy interface depends on good design and on testing it with the people who will use it.",
		Keywords:         []string{"figma", "design", "ux", "ui ", "user research", "interview", "prototype", "wireframe", "usab", "accessib", "visual", "sketch", "easy to use", "hci"},
	},
	RoleQA: {
		Key: RoleQA, Name: "QA Engineer",
		Summary:          "Plans the tests and keeps every demo free of bugs.",
		Responsibilities: []string{"Test plan for the main flows", "Manual and automated tests", "Bug tracking and triage", "Release checklist before demos"},
		Skills:           []string{"Testing", "Test automation", "Attention to detail", "Bug reports"},
		Why:              "A dedicated tester catches problems before the instructor does and keeps demos stable.",
		Keywords:         []string{"test", "qa", "bug", "detail", "careful", "quality", "selenium", "playwright", "cypress", "check"},
	},
	RoleData: {
		Key: RoleData, Name: "AI / Data Engineer",
		Summary:          "Builds the data, analytics and AI features.",
		Responsibilities: []string{"Data model for analytics", "Statistics and charts", "AI or ML integration", "Realistic seed data"},
		Skills:           []string{"Python", "SQL", "Data visualisation", "Machine learning"},
		Why:              "Analytics, statistics or AI features need someone who works with data full time.",
		Keywords:         []string{"python", "pandas", "numpy", "data", "analy", "statistic", "chart", "matplotlib", "machine learning", " ml", " ai", "model", "jupyter", "visualis", "visualiz", "predict", "recommend"},
	},
}

// pmQA is the combined role used when the team is too small for a tester.
var pmQA = Role{
	Key: "pm_qa", Name: "Project Manager & QA Lead",
	Summary:          "Owns the plan, the deadline and the quality of every demo.",
	Responsibilities: []string{"Sprint plan and weekly sync", "Communication with the instructor", "Test plan and bug tracking", "Progress reports and final demo"},
	Skills:           []string{"Planning", "Communication", "Testing", "Documentation"},
	Why:              "One person owns both the plan and the quality, because a small team cannot spare a full-time tester.",
	Keywords:         append(append([]string{}, Catalogue[RolePM].Keywords...), Catalogue[RoleQA].Keywords...),
}

// standardRoles returns the catalogue roles for a team size.
func standardRoles(teamSize int) []Role {
	c := Catalogue
	switch teamSize {
	case 3:
		return []Role{pmQA, c[RoleBackend], c[RoleFrontend]}
	case 4:
		return []Role{pmQA, c[RoleBackend], c[RoleFrontend], c[RoleDesign]}
	case 5:
		return []Role{pmQA, c[RoleBackend], c[RoleFrontend], c[RoleDesign], c[RoleData]}
	default:
		return []Role{c[RolePM], c[RoleBackend], c[RoleFrontend], c[RoleDesign], c[RoleQA], c[RoleData]}
	}
}

// equalShares splits 100 into n near-equal integers (larger first).
func equalShares(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = 100 / n
		if i < 100%n {
			out[i]++
		}
	}
	return out
}

func roleSeat(i int, r Role, share int, why string) Seat {
	return Seat{
		ID: fmt.Sprintf("S%d", i+1), Name: r.Name, Summary: r.Summary,
		Responsibilities: append([]string{}, r.Responsibilities...), Skills: append([]string{}, r.Skills...),
		EffortShare: share, Why: why,
	}
}

// StandardPlan builds the no-AI fallback plan from the catalogue (spec 6.4).
func StandardPlan(teamSize int) RolePlanOutput {
	roles := standardRoles(teamSize)
	shares := equalShares(len(roles))
	out := RolePlanOutput{}
	for i, r := range roles {
		out.Seats = append(out.Seats, roleSeat(i, r, shares[i], r.Why))
	}
	return out
}

// RoleKeysForSeat guesses which catalogue roles a seat name covers.
func RoleKeysForSeat(name string) []string {
	n := strings.ToLower(name)
	var keys []string
	add := func(k string, subs ...string) {
		for _, s := range subs {
			if strings.Contains(n, s) {
				keys = append(keys, k)
				return
			}
		}
	}
	add(RolePM, "manager", "lead", "scrum", "coordinator")
	add(RoleQA, "qa", "test", "quality")
	add(RoleBackend, "backend", "back-end", "api", "server", "full-stack", "fullstack")
	add(RoleFrontend, "frontend", "front-end", "ui developer", "web developer", "full-stack", "fullstack", "mobile")
	add(RoleDesign, "design", "ux", "research")
	add(RoleData, "data", "ai", "ml", "analyt", "integration")
	return keys
}
