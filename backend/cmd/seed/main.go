// Command seed loads the design's demo data: project StudySpace with five
// members, the approved plan, matching rows, the RACI and 17 tasks.
// It replaces any earlier copy of the demo accounts and project.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"teamload/internal/config"
	"teamload/internal/domain"
	"teamload/internal/store"
	"teamload/internal/store/db"
)

const (
	password   = "studyspace26"
	inviteCode = "K7Q2PM"
)

type person struct {
	key, name, email, strengths, weaknesses string
	tags                                    []string
}

var people = []person{
	{"m1", "Akzhol", "akzhol@kbtu.kz",
		"I organise group work well and keep deadlines. I like talking to the instructor and presenting. I am careful with details and usually find bugs in my friends' code.",
		"Not strong at writing backend code.", []string{"Planning", "Communication", "Testing", "Notion"}},
	{"m2", "Yerassyl", "yerassyl@kbtu.kz",
		"Go and PostgreSQL. I built a REST API during my internship and I like algorithms and clean data models.",
		"I don't enjoy CSS or visual design.", []string{"Go", "SQL", "Docker", "REST"}},
	{"m3", "Beknur", "beknur@kbtu.kz",
		"Figma and UI design. I ran user interviews for my HCI course and like making things easy to use.",
		"Little coding beyond HTML and CSS.", []string{"Figma", "User research", "CSS"}},
	{"m4", "Irisber", "irisber@kbtu.kz",
		"React and TypeScript. I made two front-end pet projects and I care about fast, responsive pages.",
		"I hate writing documentation.", []string{"React", "TypeScript", "Tailwind"}},
	{"m5", "Zhanarys", "zhanarys@kbtu.kz",
		"Python, pandas and charts from my data analysis course. I'm curious about how APIs work.",
		"I get nervous when presenting.", []string{"Python", "SQL", "Matplotlib"}},
}

// legacyEmails are earlier demo accounts; the seed removes them too.
var legacyEmails = []string{
	"aruzhan.s@stu.sdu.edu.kz", "daniyar.k@stu.sdu.edu.kz", "madina.t@stu.sdu.edu.kz",
	"timur.a@stu.sdu.edu.kz", "dana.b@stu.sdu.edu.kz",
}

const description = "A web app for SDU students to find and book free group-study rooms in the library. Students see a map of rooms with live availability, book a 1–2 hour slot and get a QR code to check in at the door. Librarians see occupancy statistics to plan opening hours. We need user accounts, a booking API that never double-books a room, a clean mobile-friendly interface and basic analytics."

type seat struct {
	key, name, summary string
	share              int32
	resp, skills       []string
	why                string
}

var seats = []seat{
	{"s1", "Project Manager & QA Lead", "Owns the plan, the deadline and the quality of every demo.", 16,
		[]string{"Sprint plan and weekly sync", "Communication with the instructor", "Test plan for the booking flow", "TSIS reports and final demo"},
		[]string{"Planning", "Communication", "Testing", "Documentation"},
		"A 5-person team with an 11-week deadline needs one owner for plan and quality. QA is combined with PM because the booking flow is small enough to test within that time."},
	{"s2", "Backend Developer", "Builds the booking API and the database behind it.", 26,
		[]string{"Rooms, slots and bookings schema", "Booking API with overlap checks", "Register / login", "Docker setup and seed data"},
		[]string{"Go", "PostgreSQL", "REST API", "Auth"},
		"The description says a room must never be double-booked. That needs a careful API and database, the largest share of the work."},
	{"s3", "Frontend Developer", "Turns the room map and booking flow into a fast, responsive UI.", 22,
		[]string{"Room list, filters and map view", "Booking form and validation", "QR code screen", "Responsive layout for phones"},
		[]string{"React", "TypeScript", "CSS", "Responsive design"},
		"Students will book from their phones between classes, so the interface has to be quick and mobile-friendly."},
	{"s4", "UX/UI Designer", "Designs the room map and checks it with real students.", 16,
		[]string{"Student interviews", "Wireframes and room map design", "Usability test with 5 students", "Design handoff to frontend"},
		[]string{"Figma", "User research", "UI design", "Accessibility"},
		"The description asks for a clean mobile-friendly interface and a room map. Both depend on good design and testing with students."},
	{"s5", "Data & Integration Engineer", "Builds occupancy statistics and the QR check-in.", 20,
		[]string{"Occupancy statistics for librarians", "QR code generation and check-in", "Charts on the librarian page", "Realistic seed data"},
		[]string{"SQL", "Python", "Data visualisation", "APIs"},
		"Librarians need occupancy statistics to plan opening hours, and check-in by QR code needs its own integration."},
}

type match struct {
	seat, member string
	fit          int32
	runner       string
	accepted     bool
	expl         string
}

var matches = []match{
	{"s1", "m1", 91, "s4", true, "You write that you “organise group work well and keep deadlines” and like presenting, which is the core of the PM role. Being “careful with details” and finding bugs fits the QA half of this slot."},
	{"s2", "m2", 94, "s5", true, "You built a REST API in Go with PostgreSQL during your internship, which is exactly the booking API this project needs. You prefer data models over CSS, so backend fits better than frontend."},
	{"s3", "m4", 88, "s2", false, "You made two React and TypeScript pet projects and “care about fast, responsive pages”. Students will book from their phones, so this matters most here."},
	{"s4", "m3", 92, "s1", true, "You design in Figma and “ran user interviews” before. This slot needs both: the room map design and a usability test with students."},
	{"s5", "m5", 81, "s2", false, "Python, pandas and charts match the occupancy statistics for librarians. You also say you are “curious about how APIs work”: this slot builds the QR check-in API, and backend is your second-best fit (63)."},
}

const growthNote = "Zhanarys is curious about APIs. Pairing them with Yerassyl on the QR check-in endpoint would give that practice without moving them from the slot they fit best."

var raci = []struct {
	name string
	v    map[string]string
}{
	{"Define scope & backlog", map[string]string{"s1": "A", "s2": "C", "s3": "C", "s4": "R", "s5": "I"}},
	{"Student interviews & wireframes", map[string]string{"s1": "R", "s2": "I", "s3": "C", "s4": "A", "s5": "I"}},
	{"Database & booking API", map[string]string{"s1": "I", "s2": "A", "s3": "C", "s4": "I", "s5": "R"}},
	{"Room map & booking UI", map[string]string{"s1": "I", "s2": "C", "s3": "A", "s4": "R", "s5": "I"}},
	{"QR check-in", map[string]string{"s1": "I", "s2": "R", "s3": "R", "s4": "C", "s5": "A"}},
	{"Occupancy statistics for librarians", map[string]string{"s1": "I", "s2": "C", "s3": "R", "s4": "C", "s5": "A"}},
	{"Testing & bug fixing", map[string]string{"s1": "A", "s2": "R", "s3": "R", "s4": "C", "s5": "R"}},
	{"Demo & final report", map[string]string{"s1": "A", "s2": "C", "s3": "C", "s4": "R", "s5": "C"}},
}

var tasks = []struct {
	title, member string
	h             float64
	st            string
}{
	{"Write project charter & scope", "m1", 4, "done"},
	{"Sprint 2 plan and RACI review", "m1", 6, "doing"},
	{"Test plan for the booking flow", "m1", 4, "todo"},
	{"DB schema: rooms, slots, bookings", "m2", 6, "done"},
	{"Booking API: create and cancel", "m2", 12, "doing"},
	{"Docker compose + seed data", "m2", 5, "doing"},
	{"Overlap check for bookings", "m2", 8, "todo"},
	{"Register / login endpoints", "m2", 6, "todo"},
	{"Interview 5 students", "m3", 4, "done"},
	{"Room map wireframe", "m3", 4, "doing"},
	{"Usability test script", "m3", 3, "todo"},
	{"Room list & filters UI", "m4", 10, "doing"},
	{"Booking form + validation", "m4", 6, "todo"},
	{"Responsive layout for phones", "m4", 4, "todo"},
	{"QR code on booking", "m5", 5, "doing"},
	{"Occupancy statistics query", "m5", 8, "todo"},
	{"Librarian page charts", "", 3, "todo"},
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err := run(); err != nil {
		slog.Error("seed failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	tx, err := st.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := st.WithTx(tx)
	if err := clean(ctx, q); err != nil {
		return err
	}

	users := map[string]uuid.UUID{}
	for _, pp := range people {
		u, err := q.CreateUser(ctx, db.CreateUserParams{Email: pp.email, PasswordHash: string(hash), FullName: pp.name, ConsentAt: time.Now()})
		if err != nil {
			return err
		}
		users[pp.key] = u.ID
		if _, err := q.UpsertProfile(ctx, db.UpsertProfileParams{UserID: u.ID, Strengths: pp.strengths, Weaknesses: pp.weaknesses, Tags: pp.tags}); err != nil {
			return err
		}
	}

	p, err := q.CreateProject(ctx, db.CreateProjectParams{
		Title: "StudySpace", Description: description, Deadline: time.Date(2026, 12, 11, 0, 0, 0, 0, time.UTC),
		TeamSize: 5, Context: "PjM 101 · Fall 2026", InviteCode: inviteCode, OwnerID: users["m1"],
	})
	if err != nil {
		return err
	}
	// Members join one minute apart in the design order, so ownership would
	// pass to Yerassyl if Akzhol deleted their profile.
	joined := time.Now().Add(-48 * time.Hour)
	for i, pp := range people {
		if _, err := tx.Exec(ctx, "INSERT INTO project_members (project_id, user_id, joined_at) VALUES ($1, $2, $3)",
			p.ID, users[pp.key], joined.Add(time.Duration(i)*time.Minute)); err != nil {
			return err
		}
	}

	seatIDs := map[string]uuid.UUID{}
	for i, s := range seats {
		pos := i + 1
		row, err := q.InsertSeat(ctx, db.InsertSeatParams{ProjectID: p.ID, Position: int32(pos), Name: s.name, Summary: s.summary,
			Responsibilities: s.resp, Skills: s.skills, EffortShare: s.share, Why: s.why, ColorToken: domain.SeatColor(pos)})
		if err != nil {
			return err
		}
		seatIDs[s.key] = row.ID
	}
	source, version := "ai", "role_plan/v1"
	if err := q.SetPlanGenerated(ctx, db.SetPlanGeneratedParams{ID: p.ID, PlanSource: &source, PlanPromptVersion: &version}); err != nil {
		return err
	}
	if err := q.SetPlanStatus(ctx, db.SetPlanStatusParams{ID: p.ID, PlanStatus: "approved"}); err != nil {
		return err
	}

	mversion := "matching/v1"
	for _, m := range matches {
		uid, runner, fit := users[m.member], seatIDs[m.runner], m.fit
		state, src := "suggested", "ai"
		if m.accepted {
			state, src = "accepted", "ai_accepted"
		}
		if err := q.InsertAssignment(ctx, db.InsertAssignmentParams{ProjectID: p.ID, SeatID: seatIDs[m.seat], UserID: &uid, Fit: &fit,
			RunnerUpSeatID: &runner, Explanation: m.expl, State: state, Source: src, PromptVersion: &mversion}); err != nil {
			return err
		}
	}
	gaps, err := json.Marshal([]map[string]string{{"skill": "Accessibility testing", "seat_id": seatIDs["s4"].String(),
		"reason": "The UX/UI Designer slot needs it for the usability test. Consider learning resources or asking the instructor."}})
	if err != nil {
		return err
	}
	note := growthNote
	if err := q.SetMatchingMeta(ctx, db.SetMatchingMetaParams{ID: p.ID, MatchingSource: &source, MatchingPromptVersion: &mversion,
		MatchingGaps: gaps, MatchingGrowthNote: &note}); err != nil {
		return err
	}

	for i, a := range raci {
		act, err := q.InsertRaciActivity(ctx, db.InsertRaciActivityParams{ProjectID: p.ID, Position: int32(i + 1), Name: a.name})
		if err != nil {
			return err
		}
		for k, v := range a.v {
			if err := q.InsertRaciCell(ctx, db.InsertRaciCellParams{ActivityID: act.ID, SeatID: seatIDs[k], Value: v}); err != nil {
				return err
			}
		}
	}
	rversion := "raci/v1"
	if err := q.SetRaciMeta(ctx, db.SetRaciMetaParams{ID: p.ID, RaciSource: &source, RaciPromptVersion: &rversion}); err != nil {
		return err
	}

	created := time.Now().Add(-24 * time.Hour)
	for i, t := range tasks {
		var who *uuid.UUID
		if t.member != "" {
			id := users[t.member]
			who = &id
		}
		row, err := q.CreateTask(ctx, db.CreateTaskParams{ProjectID: p.ID, Title: t.title, AssigneeID: who, Estimate: t.h, Status: t.st})
		if err != nil {
			return err
		}
		// Keep the design's card order on the board.
		if _, err := tx.Exec(ctx, "UPDATE tasks SET created_at = $2, updated_at = $2 WHERE id = $1",
			row.ID, created.Add(time.Duration(i)*time.Minute)); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("Seeded StudySpace (invite code %s). Log in as any of:\n", inviteCode)
	for _, pp := range people {
		fmt.Printf("  %-28s %s\n", pp.email, password)
	}
	return nil
}

// clean removes an earlier copy of the demo (projects first: owners are RESTRICT).
func clean(ctx context.Context, q *db.Queries) error {
	if p, err := q.GetProjectByCode(ctx, inviteCode); err == nil {
		if err := q.DeleteProject(ctx, p.ID); err != nil {
			return err
		}
	} else if !store.IsNotFound(err) {
		return err
	}
	emails := append([]string{}, legacyEmails...)
	for _, pp := range people {
		emails = append(emails, pp.email)
	}
	for _, email := range emails {
		u, err := q.GetUserByEmail(ctx, email)
		if store.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		owned, err := q.ListOwnedProjects(ctx, u.ID)
		if err != nil {
			return err
		}
		for _, o := range owned {
			if err := q.DeleteProject(ctx, o.ID); err != nil {
				return err
			}
		}
		if err := q.DeleteUser(ctx, u.ID); err != nil {
			return err
		}
	}
	return nil
}
