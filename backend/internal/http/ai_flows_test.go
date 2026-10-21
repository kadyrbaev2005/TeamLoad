package http

import (
	"context"
	"testing"

	"teamload/internal/config"
	"teamload/internal/domain"
)

// team builds an owner plus members with design profiles and an approved
// generated plan. It returns the owner, the members and the project id.
func team(t *testing.T, e *env, size int, names ...string) (*client, []*client, string) {
	t.Helper()
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(size)
	owner.must(200, "POST", "/projects/"+id+"/plan/generate", nil)
	owner.must(200, "POST", "/projects/"+id+"/plan/approve", nil)
	owner.designProfile()
	var ms []*client
	for _, n := range names {
		m := e.user(n)
		m.must(200, "POST", "/invites/"+code+"/join", nil)
		m.designProfile()
		ms = append(ms, m)
	}
	return owner, ms, id
}

func seatsOf(c *client, id string) []map[string]any {
	var out []map[string]any
	for _, s := range list(obj(c.must(200, "GET", "/projects/"+id+"/plan", nil).Body["plan"])["seats"]) {
		out = append(out, obj(s))
	}
	return out
}

func rowsOf(r resp) []map[string]any {
	var out []map[string]any
	for _, x := range list(obj(r.Body["matching"])["rows"]) {
		out = append(out, obj(x))
	}
	return out
}

// ---- 6.4 Role plan ----

func TestPlan_GenerateHasOneSeatPerMember(t *testing.T) {
	for _, n := range []int{3, 4, 5, 6} {
		e := newEnv(t)
		c := e.user("Aruzhan Sadykova")
		id, _ := c.project(n)
		r := c.must(200, "POST", "/projects/"+id+"/plan/generate", nil)
		plan := obj(r.Body["plan"])
		seats := list(plan["seats"])
		if len(seats) != n || plan["status"] != "draft" || plan["source"] != "ai" {
			t.Fatalf("team %d: %s", n, r.Raw)
		}
		sum := 0
		for i, s := range seats {
			sm := obj(s)
			sum += int(sm["effort_share"].(float64))
			if sm["color"] != domain.SeatColor(i+1) {
				t.Fatalf("seat %d colour %v", i+1, sm["color"])
			}
		}
		if sum != 100 {
			t.Fatalf("team %d: shares sum to %d", n, sum)
		}
	}
	var stored int
	if err := testStore.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM projects WHERE plan_source = 'ai' AND plan_prompt_version = 'role_plan/v1' AND plan_generated_at IS NOT NULL`).Scan(&stored); err != nil || stored != 1 {
		t.Fatalf("AI result metadata not stored (%d, %v)", stored, err)
	}
}

func TestPlan_EditRequiresTotal100(t *testing.T) {
	e := newEnv(t)
	c := e.user("Aruzhan Sadykova")
	id, _ := c.project(3)
	c.must(200, "POST", "/projects/"+id+"/plan/generate", nil)
	seats := seatsOf(c, id)
	edit := func(shares ...int) resp {
		var body []map[string]any
		for i, s := range seats {
			body = append(body, map[string]any{"id": s["id"], "name": "Role " + string(rune('A'+i)), "summary": "Does things.", "effort_share": shares[i]})
		}
		return c.do("PUT", "/projects/"+id+"/plan", map[string]any{"seats": body})
	}
	if r := edit(40, 40, 30); r.Status != 400 || r.errField("total") != domain.MsgShareTotal {
		t.Fatalf("sum 110 accepted: %s", r.Raw)
	}
	if r := edit(4, 48, 48); r.Status != 400 || r.errField("effort_share") == "" {
		t.Fatalf("share 4 accepted: %s", r.Raw)
	}
	r := edit(50, 30, 20)
	if r.Status != 200 {
		t.Fatalf("valid edit rejected: %s", r.Raw)
	}
	s := obj(list(obj(r.Body["plan"])["seats"])[0])
	if s["name"] != "Role A" || s["effort_share"].(float64) != 50 {
		t.Fatalf("edit not saved: %v", s)
	}
}

func TestPlan_ApproveUnlocksMatching(t *testing.T) {
	e := newEnv(t)
	owner, _, id := team(t, e, 3, "Daniyar Kenzhebekov", "Timur Akhmetov")
	owner.must(200, "POST", "/projects/"+id+"/plan/generate", nil) // back to draft
	m := obj(owner.must(200, "GET", "/projects/"+id+"/matching", nil).Body["matching"])
	if obj(m["readiness"])["can_run"] != false {
		t.Fatal("matching must be locked on a draft plan")
	}
	owner.must(409, "POST", "/projects/"+id+"/matching/run", nil)
	owner.must(409, "POST", "/projects/"+id+"/raci/generate", nil)
	owner.must(200, "POST", "/projects/"+id+"/plan/approve", nil)
	owner.must(200, "POST", "/projects/"+id+"/matching/run", nil)
}

func TestPlan_RegenerateWithNoteClearsMatchingAndRaci(t *testing.T) {
	e := newEnv(t)
	owner, _, id := team(t, e, 4, "Daniyar Kenzhebekov", "Madina Tulegenova", "Timur Akhmetov")
	owner.must(200, "POST", "/projects/"+id+"/matching/run", nil)
	owner.must(200, "POST", "/projects/"+id+"/raci/generate", nil)
	p := obj(owner.must(200, "GET", "/projects/"+id+"/plan", nil).Body["plan"])
	if p["has_matching"] != true || p["has_raci"] != true {
		t.Fatalf("flags: %v", p)
	}
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	owner.must(400, "POST", "/projects/"+id+"/plan/generate", map[string]any{"note": string(long)})
	r := owner.must(200, "POST", "/projects/"+id+"/plan/generate", map[string]any{"note": "we need more focus on design"})
	p = obj(r.Body["plan"])
	if p["status"] != "draft" || p["has_matching"] != false || p["has_raci"] != false {
		t.Fatalf("after regenerate: %v", p)
	}
	assertCount(t, "role_assignments", 0)
	assertCount(t, "raci_activities", 0)
	m := obj(owner.must(200, "GET", "/projects/"+id+"/matching", nil).Body["matching"])
	if m["ran"] != false {
		t.Fatal("matching must be cleared")
	}
}

func TestPlan_StandardPlanFallback(t *testing.T) {
	e := newEnv(t, withProvider(failingProvider{}))
	c := e.user("Aruzhan Sadykova")
	id, _ := c.project(5)
	if r := c.must(502, "POST", "/projects/"+id+"/plan/generate", nil); r.errMessage() != domain.MsgAIUnavailable {
		t.Fatalf("got %s", r.Raw)
	}
	r := c.must(200, "POST", "/projects/"+id+"/plan/standard", nil)
	plan := obj(r.Body["plan"])
	if len(list(plan["seats"])) != 5 || plan["source"] != "standard" || plan["status"] != "draft" {
		t.Fatalf("standard plan: %s", r.Raw)
	}
	c.must(200, "POST", "/projects/"+id+"/plan/approve", nil)
}

func TestPlan_MembersViewOnly(t *testing.T) {
	e := newEnv(t)
	owner, ms, id := team(t, e, 3, "Daniyar Kenzhebekov", "Timur Akhmetov")
	_ = owner
	m := ms[0]
	m.must(200, "GET", "/projects/"+id+"/plan", nil)
	m.must(403, "PUT", "/projects/"+id+"/plan", map[string]any{"seats": []any{}})
	m.must(403, "POST", "/projects/"+id+"/plan/approve", nil)
	m.must(403, "POST", "/projects/"+id+"/plan/standard", nil)
	m.must(403, "POST", "/projects/"+id+"/matching/run", nil)
	m.must(403, "POST", "/projects/"+id+"/raci/generate", nil)
	m.must(403, "PUT", "/projects/"+id+"/raci", map[string]any{"activities": []any{}})
}

func TestAI_DailyLimit(t *testing.T) {
	e := newEnv(t, withConfig(func(c *config.Config) { c.LLMMaxCallsPerDay = 2 }))
	c := e.user("Aruzhan Sadykova")
	id, _ := c.project(3)
	c.must(200, "POST", "/projects/"+id+"/plan/generate", nil)
	c.must(200, "POST", "/projects/"+id+"/plan/generate", nil)
	r := c.must(429, "POST", "/projects/"+id+"/plan/generate", nil)
	if r.errMessage() != domain.MsgAILimit {
		t.Fatalf("message %q", r.errMessage())
	}
	// The standard plan does not use AI and still works.
	c.must(200, "POST", "/projects/"+id+"/plan/standard", nil)
}

// ---- 6.6 Matching ----

func TestMatching_NotReadyReasons(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(3)
	owner.must(200, "POST", "/projects/"+id+"/plan/generate", nil)
	owner.must(200, "POST", "/projects/"+id+"/plan/approve", nil)
	owner.designProfile()
	r := owner.must(409, "POST", "/projects/"+id+"/matching/run", nil)
	if r.errMessage() != "Matching needs at least 3 members." {
		t.Fatalf("got %q", r.errMessage())
	}
	e.user("Daniyar Kenzhebekov").must(200, "POST", "/invites/"+code+"/join", nil)
	e.user("Timur Akhmetov").must(200, "POST", "/invites/"+code+"/join", nil)
	r = owner.must(409, "POST", "/projects/"+id+"/matching/run", nil)
	if r.errMessage() != "Waiting for profiles: Daniyar Kenzhebekov, Timur Akhmetov." {
		t.Fatalf("got %q", r.errMessage())
	}
	m := obj(owner.must(200, "GET", "/projects/"+id+"/matching", nil).Body["matching"])
	if len(list(obj(m["readiness"])["missing_profiles"])) != 2 {
		t.Fatalf("readiness: %v", m["readiness"])
	}
}

func TestMatching_RunSuggestsOnePersonPerSeat(t *testing.T) {
	e := newEnv(t)
	owner, _, id := team(t, e, 5, "Daniyar Kenzhebekov", "Madina Tulegenova", "Timur Akhmetov", "Dana Bekova")
	r := owner.must(200, "POST", "/projects/"+id+"/matching/run", nil)
	rows := rowsOf(r)
	if len(rows) != 5 {
		t.Fatalf("%d rows", len(rows))
	}
	seen := map[any]bool{}
	for _, row := range rows {
		if row["user_id"] == nil || seen[row["user_id"]] || row["state"] != "suggested" || row["source"] != "ai" {
			t.Fatalf("row: %v", row)
		}
		seen[row["user_id"]] = true
		if f := row["fit"].(float64); f < 0 || f > 100 || row["runner_up_seat_id"] == nil || row["explanation"] == "" {
			t.Fatalf("row content: %v", row)
		}
	}
	// The design's expected matches: Daniyar → Backend, Madina → Designer, Timur → Frontend.
	seats := seatsOf(owner, id)
	nameOf := map[any]string{}
	for _, s := range seats {
		nameOf[s["id"]] = s["name"].(string)
	}
	members := list(owner.must(200, "GET", "/projects/"+id, nil).Body["members"])
	want := map[string]string{"Daniyar Kenzhebekov": "Backend Developer", "Madina Tulegenova": "UX/UI Designer", "Timur Akhmetov": "Frontend Developer"}
	for _, row := range rows {
		for _, m := range members {
			mm := obj(m)
			if mm["id"] == row["user_id"] && want[mm["name"].(string)] != "" && want[mm["name"].(string)] != nameOf[row["seat_id"]] {
				t.Errorf("%s got %s", mm["name"], nameOf[row["seat_id"]])
			}
		}
	}
}

func TestMatching_FewerMembersLeaveSeatsUnassigned(t *testing.T) {
	e := newEnv(t)
	owner, _, id := team(t, e, 5, "Daniyar Kenzhebekov", "Timur Akhmetov")
	rows := rowsOf(owner.must(200, "POST", "/projects/"+id+"/matching/run", nil))
	unassigned := 0
	for _, r := range rows {
		if r["user_id"] == nil {
			unassigned++
		}
	}
	if len(rows) != 5 || unassigned != 2 {
		t.Fatalf("rows=%d unassigned=%d", len(rows), unassigned)
	}
	// The owner can assign a leftover seat later... after someone joins.
}

func TestMatching_AcceptOneAndAll(t *testing.T) {
	e := newEnv(t)
	owner, _, id := team(t, e, 3, "Daniyar Kenzhebekov", "Timur Akhmetov")
	rows := rowsOf(owner.must(200, "POST", "/projects/"+id+"/matching/run", nil))
	r := owner.must(200, "POST", "/projects/"+id+"/matching/accept", map[string]any{"seat_id": rows[0]["seat_id"]})
	after := rowsOf(r)
	if after[0]["state"] != "accepted" || after[0]["source"] != "ai_accepted" || after[1]["state"] != "suggested" {
		t.Fatalf("accept one: %v", after)
	}
	for _, row := range rowsOf(owner.must(200, "POST", "/projects/"+id+"/matching/accept", nil)) {
		if row["state"] != "accepted" || row["source"] != "ai_accepted" {
			t.Fatalf("accept all: %v", row)
		}
	}
	members := list(owner.must(200, "GET", "/projects/"+id, nil).Body["members"])
	for _, m := range members {
		if obj(m)["role_confirmed"] != true {
			t.Fatalf("member not confirmed: %v", m)
		}
	}
}

func TestMatching_ReassignSwapsSeats(t *testing.T) {
	e := newEnv(t)
	owner, ms, id := team(t, e, 3, "Daniyar Kenzhebekov", "Timur Akhmetov")
	rows := rowsOf(owner.must(200, "POST", "/projects/"+id+"/matching/run", nil))
	a, b := rows[0], rows[1]
	r := owner.must(200, "PUT", "/projects/"+id+"/matching/seats/"+a["seat_id"].(string), map[string]any{"user_id": b["user_id"]})
	after := rowsOf(r)
	if after[0]["user_id"] != b["user_id"] || after[1]["user_id"] != a["user_id"] {
		t.Fatalf("not swapped: %v", after)
	}
	for _, row := range after[:2] {
		if row["source"] != "manual" || row["state"] != "accepted" || row["fit"] != nil || row["explanation"] != "" {
			t.Fatalf("swapped row: %v", row)
		}
	}
	if after[2]["source"] != "ai" {
		t.Fatal("third row must be untouched")
	}
	// After confirmation the owner can still change roles.
	owner.must(200, "PUT", "/projects/"+id+"/matching/seats/"+a["seat_id"].(string), map[string]any{"user_id": a["user_id"]})
	owner.must(400, "PUT", "/projects/"+id+"/matching/seats/"+a["seat_id"].(string), map[string]any{"user_id": "not-a-user"})
	ms[0].must(403, "PUT", "/projects/"+id+"/matching/seats/"+a["seat_id"].(string), map[string]any{"user_id": ms[0].ID})
	// The DB keeps one seat per person and one person per seat.
	if _, err := testStore.Pool.Exec(context.Background(),
		`UPDATE role_assignments SET user_id = (SELECT user_id FROM role_assignments WHERE user_id IS NOT NULL LIMIT 1)`); err == nil {
		t.Fatal("database allowed one person on two seats")
	}
}

func TestMatching_ProfileChangeMarksOutdated(t *testing.T) {
	e := newEnv(t)
	owner, ms, id := team(t, e, 3, "Daniyar Kenzhebekov", "Timur Akhmetov")
	owner.must(200, "POST", "/projects/"+id+"/matching/run", nil)
	d := ms[0]
	// Saving the same text does not mark anything.
	d.designProfile()
	r := d.must(200, "PUT", "/me/profile", map[string]any{
		"strengths": "Go, PostgreSQL and Kubernetes. I built a REST API during my internship and I like algorithms.", "weaknesses": "I don't enjoy CSS.", "tags": []string{"Go"}})
	if r.Body["outdated_count"].(float64) != 1 {
		t.Fatalf("outdated_count: %s", r.Raw)
	}
	var outdated []any
	for _, row := range rowsOf(owner.must(200, "GET", "/projects/"+id+"/matching", nil)) {
		if row["outdated"] == true {
			outdated = append(outdated, row["user_id"])
		}
	}
	if len(outdated) != 1 || outdated[0] != d.ID {
		t.Fatalf("outdated rows: %v", outdated)
	}
	// Running again uses the new text and clears the flag.
	for _, row := range rowsOf(owner.must(200, "POST", "/projects/"+id+"/matching/run", nil)) {
		if row["outdated"] == true {
			t.Fatal("run again must clear outdated")
		}
	}
}

func TestMatching_AIFailureKeepsProfilesAndAllowsManual(t *testing.T) {
	e := newEnv(t, withProvider(failingProvider{}))
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(3)
	owner.must(200, "POST", "/projects/"+id+"/plan/standard", nil)
	owner.must(200, "POST", "/projects/"+id+"/plan/approve", nil)
	owner.designProfile()
	var ms []*client
	for _, n := range []string{"Daniyar Kenzhebekov", "Timur Akhmetov"} {
		m := e.user(n)
		m.must(200, "POST", "/invites/"+code+"/join", nil)
		m.designProfile()
		ms = append(ms, m)
	}
	r := owner.must(502, "POST", "/projects/"+id+"/matching/run", nil)
	if r.errMessage() != domain.MsgAIUnavailable {
		t.Fatalf("got %s", r.Raw)
	}
	assertCount(t, "profiles", 3)
	rows := rowsOf(owner.must(200, "GET", "/projects/"+id+"/matching", nil))
	people := []string{owner.ID, ms[0].ID, ms[1].ID}
	for i, row := range rows {
		owner.must(200, "PUT", "/projects/"+id+"/matching/seats/"+row["seat_id"].(string), map[string]any{"user_id": people[i]})
	}
	for i, row := range rowsOf(owner.must(200, "GET", "/projects/"+id+"/matching", nil)) {
		if row["user_id"] != people[i] || row["source"] != "manual" {
			t.Fatalf("manual row %d: %v", i, row)
		}
	}
}

func TestMatching_GapsAndGrowthNoteFromAI(t *testing.T) {
	e := newEnv(t)
	owner, _, id := team(t, e, 5, "Daniyar Kenzhebekov", "Madina Tulegenova", "Timur Akhmetov", "Dana Bekova")
	m := obj(owner.must(200, "POST", "/projects/"+id+"/matching/run", nil).Body["matching"])
	gaps := list(m["gaps"])
	if len(gaps) == 0 {
		t.Fatal("expected skill gaps from the AI output")
	}
	seatIDs := map[any]bool{}
	for _, s := range seatsOf(owner, id) {
		seatIDs[s["id"]] = true
	}
	for _, g := range gaps {
		if !seatIDs[obj(g)["seat_id"]] {
			t.Fatalf("gap seat not mapped back: %v", g)
		}
	}
	if note, _ := m["growth_note"].(string); note == "" {
		t.Fatal("Dana is curious about APIs: expected a growth note")
	}
}

func TestMatching_RemovedMemberLeavesSeatUnassigned(t *testing.T) {
	e := newEnv(t)
	owner, ms, id := team(t, e, 3, "Daniyar Kenzhebekov", "Timur Akhmetov")
	owner.must(200, "POST", "/projects/"+id+"/matching/run", nil)
	owner.must(204, "DELETE", "/projects/"+id+"/members/"+ms[0].ID, nil)
	unassigned := 0
	for _, row := range rowsOf(owner.must(200, "GET", "/projects/"+id+"/matching", nil)) {
		if row["user_id"] == nil {
			unassigned++
		}
	}
	if unassigned != 1 {
		t.Fatalf("%d unassigned seats, want 1", unassigned)
	}
}

// ---- 6.7 RACI ----

func TestRaci_DraftEditAndCheck(t *testing.T) {
	e := newEnv(t)
	owner, ms, id := team(t, e, 5, "Daniyar Kenzhebekov", "Madina Tulegenova", "Timur Akhmetov", "Dana Bekova")
	r := owner.must(200, "POST", "/projects/"+id+"/raci/generate", nil)
	acts := list(obj(r.Body["raci"])["activities"])
	if len(acts) < 6 || len(acts) > 10 {
		t.Fatalf("%d activities", len(acts))
	}
	seats := seatsOf(owner, id)
	for _, a := range acts {
		var vals []string
		for _, s := range seats {
			v, _ := obj(obj(a)["cells"])[s["id"].(string)].(string)
			vals = append(vals, v)
		}
		if c := domain.RaciCheck(vals); c != domain.RaciOK {
			t.Fatalf("AI draft row %v: %s", obj(a)["name"], c)
		}
	}
	// Members can read but not edit.
	ms[0].must(200, "GET", "/projects/"+id+"/raci", nil)
	// Owner edits: rename, clear a row (invalid while editing is allowed), add, delete.
	first := obj(acts[0])
	var body []map[string]any
	for i, a := range acts {
		am := obj(a)
		row := map[string]any{"id": am["id"], "name": am["name"], "cells": am["cells"]}
		if i == 0 {
			row["name"] = "Kick-off & scope"
			row["cells"] = map[string]string{}
		}
		if i == 1 {
			continue // delete
		}
		body = append(body, row)
	}
	body = append(body, map[string]any{"name": "New activity", "cells": map[string]string{seats[0]["id"].(string): "A"}})
	r = owner.must(200, "PUT", "/projects/"+id+"/raci", map[string]any{"activities": body})
	after := list(obj(r.Body["raci"])["activities"])
	if len(after) != len(acts) || obj(after[0])["id"] != first["id"] || obj(after[0])["name"] != "Kick-off & scope" {
		t.Fatalf("after edit: %s", r.Raw)
	}
	if len(obj(obj(after[0])["cells"])) != 0 || obj(after[len(after)-1])["name"] != "New activity" {
		t.Fatalf("cells/new row: %s", r.Raw)
	}
	bad := map[string]any{"activities": []map[string]any{{"name": "X", "cells": map[string]string{seats[0]["id"].(string): "Z"}}}}
	if r := owner.must(400, "PUT", "/projects/"+id+"/raci", bad); r.errField("cells") != domain.MsgRaciValue {
		t.Fatalf("bad value accepted: %s", r.Raw)
	}
}

func TestRaci_RegenerateUnlinksTasks(t *testing.T) {
	e := newEnv(t)
	owner, _, id := team(t, e, 3, "Daniyar Kenzhebekov", "Timur Akhmetov")
	acts := list(obj(owner.must(200, "POST", "/projects/"+id+"/raci/generate", nil).Body["raci"])["activities"])
	task := obj(owner.must(201, "POST", "/projects/"+id+"/tasks", map[string]any{
		"title": "Kick-off", "estimate": 2, "raci_activity_id": obj(acts[0])["id"]}).Body["task"])
	if task["raci_activity_id"] == nil {
		t.Fatal("link not saved")
	}
	owner.must(200, "POST", "/projects/"+id+"/raci/generate", nil)
	tasks := list(owner.must(200, "GET", "/projects/"+id+"/tasks", nil).Body["tasks"])
	if obj(tasks[0])["raci_activity_id"] != nil {
		t.Fatal("task must lose the link to a deleted activity")
	}
}
