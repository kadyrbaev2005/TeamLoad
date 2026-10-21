package http

import (
	"math"
	"testing"

	"teamload/internal/domain"
)

// crew creates a project with the given members (first is the owner).
func crew(t *testing.T, e *env, size int, names ...string) ([]*client, string) {
	t.Helper()
	owner := e.user(names[0])
	id, code := owner.project(size)
	out := []*client{owner}
	for _, n := range names[1:] {
		m := e.user(n)
		m.must(200, "POST", "/invites/"+code+"/join", nil)
		out = append(out, m)
	}
	return out, id
}

func task(c *client, id string, body map[string]any) map[string]any {
	c.e.t.Helper()
	return obj(c.must(201, "POST", "/projects/"+id+"/tasks", body).Body["task"])
}

// ---- 6.8 Task board ----

func TestTasks_ValidationMessages(t *testing.T) {
	e := newEnv(t)
	cs, id := crew(t, e, 3, "Aruzhan Sadykova")
	c := cs[0]
	cases := []struct {
		body  map[string]any
		field string
		msg   string
	}{
		{map[string]any{"title": "", "estimate": 2}, "title", domain.MsgTaskTitleRequired},
		{map[string]any{"title": "   ", "estimate": 2}, "title", domain.MsgTaskTitleRequired},
		{map[string]any{"title": "API"}, "estimate", domain.MsgEstimateRequired},
		{map[string]any{"title": "API", "estimate": nil}, "estimate", domain.MsgEstimateRequired},
		{map[string]any{"title": "API", "estimate": 0}, "estimate", domain.MsgEstimateRequired},
		{map[string]any{"title": "API", "estimate": -3}, "estimate", domain.MsgEstimateRequired},
		{map[string]any{"title": "API", "estimate": 40.5}, "estimate", domain.MsgEstimateRange},
		{map[string]any{"title": "API", "estimate": 0.25}, "estimate", domain.MsgEstimateRange},
		{map[string]any{"title": "API", "estimate": 2, "status": "blocked"}, "status", domain.MsgTaskStatus},
	}
	for _, tc := range cases {
		r := c.must(400, "POST", "/projects/"+id+"/tasks", tc.body)
		if r.errField(tc.field) != tc.msg {
			t.Errorf("%v: %s", tc.body, r.Raw)
		}
	}
	assertCount(t, "tasks", 0)
	tk := task(c, id, map[string]any{"title": "Booking API", "estimate": 40})
	if tk["status"] != "todo" || tk["estimate"].(float64) != 40 || tk["assignee_id"] != nil {
		t.Fatalf("defaults: %v", tk)
	}
	if tk := task(c, id, map[string]any{"title": "Half", "estimate": 0.5}); tk["estimate"].(float64) != 0.5 {
		t.Fatalf("0.5 h: %v", tk)
	}
}

func TestTasks_AllMembersManageTasks(t *testing.T) {
	e := newEnv(t)
	cs, id := crew(t, e, 3, "Aruzhan Sadykova", "Daniyar Kenzhebekov")
	owner, m := cs[0], cs[1]
	tk := task(m, id, map[string]any{"title": "Schema", "estimate": 6, "assignee_id": owner.ID})
	path := "/projects/" + id + "/tasks/" + tk["id"].(string)
	r := m.must(200, "PATCH", path, map[string]any{"status": "doing", "assignee_id": m.ID, "estimate": 7.5})
	got := obj(r.Body["task"])
	if got["status"] != "doing" || got["assignee_id"] != m.ID || got["estimate"].(float64) != 7.5 || got["title"] != "Schema" {
		t.Fatalf("patch: %v", got)
	}
	// Unassign with an explicit null; other fields stay.
	got = obj(owner.must(200, "PATCH", path, map[string]any{"assignee_id": nil}).Body["task"])
	if got["assignee_id"] != nil || got["status"] != "doing" {
		t.Fatalf("unassign: %v", got)
	}
	if r := owner.must(400, "PATCH", path, map[string]any{"estimate": 0}); r.errField("estimate") != domain.MsgEstimateRequired {
		t.Fatalf("patch estimate 0: %s", r.Raw)
	}
	m.must(204, "DELETE", path, nil)
	m.must(404, "PATCH", path, map[string]any{"status": "done"})
	assertCount(t, "tasks", 0)
}

func TestTasks_AssigneeMustBeMember(t *testing.T) {
	e := newEnv(t)
	cs, id := crew(t, e, 3, "Aruzhan Sadykova")
	outsider := e.user("Stranger Danger")
	r := cs[0].must(400, "POST", "/projects/"+id+"/tasks", map[string]any{"title": "X", "estimate": 1, "assignee_id": outsider.ID})
	if r.errField("assignee_id") == "" {
		t.Fatalf("got %s", r.Raw)
	}
	outsider.must(404, "POST", "/projects/"+id+"/tasks", map[string]any{"title": "X", "estimate": 1})
}

func TestTasks_OtherProjectsTasksAreInvisible(t *testing.T) {
	e := newEnv(t)
	a, idA := crew(t, e, 3, "Aruzhan Sadykova")
	b, idB := crew(t, e, 3, "Daniyar Kenzhebekov")
	tk := task(a[0], idA, map[string]any{"title": "Secret", "estimate": 1})
	b[0].must(404, "PATCH", "/projects/"+idB+"/tasks/"+tk["id"].(string), map[string]any{"status": "done"})
	b[0].must(404, "DELETE", "/projects/"+idB+"/tasks/"+tk["id"].(string), nil)
}

func TestTasks_RemovedMemberTasksBecomeUnassigned(t *testing.T) {
	e := newEnv(t)
	cs, id := crew(t, e, 3, "Aruzhan Sadykova", "Daniyar Kenzhebekov")
	tk := task(cs[0], id, map[string]any{"title": "API", "estimate": 5, "assignee_id": cs[1].ID})
	cs[0].must(204, "DELETE", "/projects/"+id+"/members/"+cs[1].ID, nil)
	tasks := list(cs[0].must(200, "GET", "/projects/"+id+"/tasks", nil).Body["tasks"])
	if len(tasks) != 1 || obj(tasks[0])["assignee_id"] != nil || obj(tasks[0])["id"] != tk["id"] {
		t.Fatalf("tasks after removal: %v", tasks)
	}
}

func TestProfile_DeleteUnassignsTasks(t *testing.T) {
	e := newEnv(t)
	cs, id := crew(t, e, 3, "Aruzhan Sadykova", "Daniyar Kenzhebekov")
	task(cs[0], id, map[string]any{"title": "API", "estimate": 5, "assignee_id": cs[1].ID})
	cs[1].must(204, "DELETE", "/me/profile", nil)
	tasks := list(cs[0].must(200, "GET", "/projects/"+id+"/tasks", nil).Body["tasks"])
	if obj(tasks[0])["assignee_id"] != nil {
		t.Fatal("task must be unassigned after the account is deleted")
	}
}

// ---- 6.9 Workload dashboard ----

func dash(c *client, id string) map[string]any {
	c.e.t.Helper()
	return obj(c.must(200, "GET", "/projects/"+id+"/dashboard", nil).Body["dashboard"])
}

func rowFor(d map[string]any, userID string) map[string]any {
	for _, r := range list(d["rows"]) {
		if obj(r)["user_id"] == userID {
			return obj(r)
		}
	}
	return nil
}

func TestDashboard_ReferenceCase(t *testing.T) {
	e := newEnv(t)
	names := []string{"Akzhol Kadyrbayev", "Beknur Otessin", "Yerasyl Aman", "Irisbek Nur", "Zhanarys Bek"}
	cs, id := crew(t, e, 5, names...)
	loads := map[int][]float64{0: {10, 4}, 1: {12, 10}, 2: {16}, 3: {7.5, 7.5}, 4: {9}}
	for i, hs := range loads {
		for k, h := range hs {
			st := "todo"
			if k == 1 {
				st = "doing"
			}
			task(cs[0], id, map[string]any{"title": "Work", "estimate": h, "assignee_id": cs[i].ID, "status": st})
		}
	}
	// Done hours and unassigned hours never count toward member load.
	task(cs[0], id, map[string]any{"title": "Done", "estimate": 20, "assignee_id": cs[1].ID, "status": "done"})
	task(cs[0], id, map[string]any{"title": "Nobody", "estimate": 3})

	d := dash(cs[0], id)
	if avg := d["average"].(float64); math.Abs(avg-15.2) > 1e-9 {
		t.Fatalf("average = %v, want 15.2", avg)
	}
	want := []string{"Balanced", "Overloaded", "Balanced", "Balanced", "Underloaded"}
	for i, c := range cs {
		if s := rowFor(d, c.ID)["status"]; s != want[i] {
			t.Errorf("%s: %v, want %s", names[i], s, want[i])
		}
	}
	if d["total_active"].(float64) != 76 || d["unassigned_active"].(float64) != 3 || d["open_tasks"].(float64) != 9 {
		t.Fatalf("kpis: %v", d)
	}
	beknur := rowFor(d, cs[1].ID)
	if beknur["active"].(float64) != 22 || beknur["done"].(float64) != 20 || beknur["open_tasks"].(float64) != 2 {
		t.Fatalf("beknur row: %v", beknur)
	}
	if d["days_left"].(float64) != 50 {
		t.Fatalf("days_left = %v", d["days_left"])
	}
}

func TestDashboard_ExactlyAt130IsBalanced(t *testing.T) {
	e := newEnv(t)
	cs, id := crew(t, e, 3, "Aruzhan Sadykova", "Daniyar Kenzhebekov", "Timur Akhmetov")
	// Loads 13, 10, 7 → average 10; 13 h is exactly 130%.
	for i, h := range []float64{13, 10, 7} {
		task(cs[0], id, map[string]any{"title": "W", "estimate": h, "assignee_id": cs[i].ID})
	}
	d := dash(cs[0], id)
	if s := rowFor(d, cs[0].ID)["status"]; s != "Balanced" {
		t.Fatalf("exactly 130%%: %v", s)
	}
	if s := rowFor(d, cs[2].ID)["status"]; s != "Underloaded" {
		t.Fatalf("70%%: %v", s)
	}
}

func TestDashboard_NoEstimatedTasks(t *testing.T) {
	e := newEnv(t)
	cs, id := crew(t, e, 3, "Aruzhan Sadykova", "Daniyar Kenzhebekov")
	d := dash(cs[0], id)
	if d["has_estimates"] != false || d["average"].(float64) != 0 || len(list(d["rows"])) != 2 {
		t.Fatalf("empty dashboard: %v", d)
	}
	for _, r := range list(d["rows"]) {
		if obj(r)["status"] != "Balanced" {
			t.Fatalf("idle members are balanced: %v", r)
		}
	}
}

func TestDashboard_ReflectsReassignAndEstimateChange(t *testing.T) {
	e := newEnv(t)
	cs, id := crew(t, e, 3, "Aruzhan Sadykova", "Daniyar Kenzhebekov", "Timur Akhmetov")
	tk := task(cs[0], id, map[string]any{"title": "Big", "estimate": 30, "assignee_id": cs[0].ID})
	task(cs[0], id, map[string]any{"title": "Small", "estimate": 10, "assignee_id": cs[1].ID})
	if s := rowFor(dash(cs[0], id), cs[0].ID)["status"]; s != "Overloaded" {
		t.Fatalf("before: %v", s)
	}
	path := "/projects/" + id + "/tasks/" + tk["id"].(string)
	cs[1].must(200, "PATCH", path, map[string]any{"assignee_id": cs[2].ID})
	d := dash(cs[0], id)
	if rowFor(d, cs[0].ID)["active"].(float64) != 0 || rowFor(d, cs[2].ID)["active"].(float64) != 30 {
		t.Fatalf("after reassign: %v", d)
	}
	cs[1].must(200, "PATCH", path, map[string]any{"estimate": 5})
	d = dash(cs[0], id)
	// Loads 0, 10, 5 → average 5: Daniyar 200% overloaded, Timur balanced, owner idle.
	if rowFor(d, cs[2].ID)["active"].(float64) != 5 || rowFor(d, cs[2].ID)["status"] != "Balanced" ||
		rowFor(d, cs[1].ID)["status"] != "Overloaded" || rowFor(d, cs[0].ID)["status"] != "Underloaded" {
		t.Fatalf("after estimate change: %v", d["rows"])
	}
}

func TestDashboard_PlanShareFromConfirmedRole(t *testing.T) {
	e := newEnv(t)
	owner, _, id := team(t, e, 3, "Daniyar Kenzhebekov", "Timur Akhmetov")
	owner.must(200, "POST", "/projects/"+id+"/matching/run", nil)
	task(owner, id, map[string]any{"title": "Plan", "estimate": 4, "assignee_id": owner.ID})
	if r := rowFor(dash(owner, id), owner.ID); r["plan_share"] != nil {
		t.Fatalf("suggested role must not count as confirmed: %v", r)
	}
	owner.must(200, "POST", "/projects/"+id+"/matching/accept", nil)
	r := rowFor(dash(owner, id), owner.ID)
	if r["plan_share"] == nil || r["seat_id"] == nil || r["share_pct"].(float64) != 100 {
		t.Fatalf("confirmed row: %v", r)
	}
}
