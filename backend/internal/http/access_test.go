package http

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"teamload/internal/config"
	"teamload/internal/domain"
)

// ---- 6.1 Registration and login ----

func TestRegister_Valid(t *testing.T) {
	e := newEnv(t)
	c := e.anon()
	r := c.must(201, "POST", "/auth/register", map[string]any{
		"full_name": "Aruzhan Sadykova", "email": "Aruzhan.S@STU.sdu.edu.kz", "password": "studyspace26", "consent": true})
	u := obj(r.Body["user"])
	if u["email"] != "aruzhan.s@stu.sdu.edu.kz" {
		t.Fatalf("email not lower-cased: %v", u["email"])
	}
	me := c.must(200, "GET", "/auth/me", nil)
	if obj(me.Body["user"])["full_name"] != "Aruzhan Sadykova" {
		t.Fatal("session not started")
	}
	var consent int
	if err := testStore.Pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE consent_at IS NOT NULL`).Scan(&consent); err != nil || consent != 1 {
		t.Fatal("consent_at not stored")
	}
}

func TestRegister_SessionCookieFlags(t *testing.T) {
	e := newEnv(t)
	r, err := e.anon().http.Post(e.srv.URL+"/api/auth/register", "application/json",
		stringsReader(`{"full_name":"Dana Bekova","email":"dana@x.kz","password":"studyspace26","consent":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var found bool
	for _, ck := range r.Cookies() {
		if ck.Name == sessionCookie {
			found = true
			if !ck.HttpOnly || ck.SameSite != 2 /* Lax */ || ck.MaxAge != 7*24*3600 {
				t.Fatalf("cookie flags: %+v", ck)
			}
		}
	}
	if !found {
		t.Fatal("no session cookie")
	}
}

func TestRegister_EmailTaken(t *testing.T) {
	e := newEnv(t)
	body := map[string]any{"full_name": "Aruzhan Sadykova", "email": "aruzhan@x.kz", "password": "studyspace26", "consent": true}
	e.anon().must(201, "POST", "/auth/register", body)
	body["email"] = "ARUZHAN@x.kz"
	r := e.anon().must(409, "POST", "/auth/register", body)
	if r.errMessage() != domain.MsgEmailTaken {
		t.Fatalf("message %q", r.errMessage())
	}
	assertCount(t, "users", 1)
}

func TestRegister_InvalidEmail(t *testing.T) {
	e := newEnv(t)
	r := e.anon().must(400, "POST", "/auth/register", map[string]any{"full_name": "Dana Bekova", "email": "dana@", "password": "studyspace26", "consent": true})
	if r.errField("email") == "" {
		t.Fatal("email field not marked")
	}
	assertCount(t, "users", 0)
}

func TestRegister_ShortPassword(t *testing.T) {
	e := newEnv(t)
	r := e.anon().must(400, "POST", "/auth/register", map[string]any{"full_name": "Dana Bekova", "email": "dana@x.kz", "password": "1234567", "consent": true})
	if r.errField("password") == "" || r.errField("email") != "" {
		t.Fatalf("wrong fields: %s", r.Raw)
	}
	assertCount(t, "users", 0)
}

func TestRegister_NoConsent(t *testing.T) {
	e := newEnv(t)
	r := e.anon().must(400, "POST", "/auth/register", map[string]any{"full_name": "Dana Bekova", "email": "dana@x.kz", "password": "studyspace26", "consent": false})
	if r.errMessage() != domain.MsgNoConsent || r.errField("consent") != domain.MsgNoConsent {
		t.Fatalf("got %s", r.Raw)
	}
	assertCount(t, "users", 0)
}

func TestLogin_Valid(t *testing.T) {
	e := newEnv(t)
	e.anon().must(201, "POST", "/auth/register", map[string]any{"full_name": "Dana Bekova", "email": "dana@x.kz", "password": "studyspace26", "consent": true})
	c := e.anon()
	c.must(200, "POST", "/auth/login", map[string]any{"email": "Dana@X.kz", "password": "studyspace26"})
	c.must(200, "GET", "/auth/me", nil)
}

func TestLogin_WrongPassword(t *testing.T) {
	e := newEnv(t)
	e.anon().must(201, "POST", "/auth/register", map[string]any{"full_name": "Dana Bekova", "email": "dana@x.kz", "password": "studyspace26", "consent": true})
	r := e.anon().must(401, "POST", "/auth/login", map[string]any{"email": "dana@x.kz", "password": "wrong-password"})
	if r.errMessage() != domain.MsgInvalidLogin {
		t.Fatalf("message %q", r.errMessage())
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	e := newEnv(t)
	r := e.anon().must(401, "POST", "/auth/login", map[string]any{"email": "nobody@x.kz", "password": "studyspace26"})
	if r.errMessage() != domain.MsgInvalidLogin {
		t.Fatalf("message %q", r.errMessage())
	}
}

func TestLogin_RateLimited(t *testing.T) {
	e := newEnv(t)
	c := e.anon()
	for i := range 10 {
		if r := c.do("POST", "/auth/login", map[string]any{"email": "x@x.kz", "password": "nopenope"}); r.Status != 401 {
			t.Fatalf("attempt %d: %d", i+1, r.Status)
		}
	}
	r := c.must(429, "POST", "/auth/login", map[string]any{"email": "x@x.kz", "password": "nopenope"})
	if r.errMessage() != domain.MsgTooManyLogins {
		t.Fatalf("message %q", r.errMessage())
	}
}

func TestMe_GuestHasNoSession(t *testing.T) {
	e := newEnv(t)
	if r := e.anon().must(200, "GET", "/auth/me", nil); r.Body["user"] != nil {
		t.Fatalf("guest: %s", r.Raw)
	}
	e.anon().must(401, "GET", "/projects", nil)
}

func TestLogout_EndsSession(t *testing.T) {
	e := newEnv(t)
	c := e.user("Aruzhan Sadykova")
	id, _ := c.project(5)
	c.must(204, "POST", "/auth/logout", nil)
	if c.must(200, "GET", "/auth/me", nil).Body["user"] != nil {
		t.Fatal("session still active")
	}
	c.must(401, "GET", "/projects/"+id, nil)
}

// ---- 6.2 Projects and invite codes ----

func TestCreateProject_Valid(t *testing.T) {
	e := newEnv(t)
	c := e.user("Aruzhan Sadykova")
	r := c.must(201, "POST", "/projects", projectBody(5))
	p := obj(r.Body["project"])
	if !domain.ValidInviteCode(p["invite_code"].(string)) {
		t.Fatalf("bad code %v", p["invite_code"])
	}
	if p["invite_link"] != "http://localhost:8080/join/"+p["invite_code"].(string) {
		t.Fatalf("bad link %v", p["invite_link"])
	}
	if r.Body["is_owner"] != true || len(list(r.Body["members"])) != 1 || p["plan_status"] != "draft" {
		t.Fatalf("owner must be first member: %s", r.Raw)
	}
	// The plan is generated right after (behind the overlay).
	plan := c.must(200, "POST", "/projects/"+p["id"].(string)+"/plan/generate", map[string]any{})
	if n := len(list(obj(plan.Body["plan"])["seats"])); n != 5 {
		t.Fatalf("%d seats, want 5", n)
	}
}

func TestCreateProject_Validation(t *testing.T) {
	e := newEnv(t)
	c := e.user("Aruzhan Sadykova")
	cases := map[string]map[string]any{
		"title":       {"title": "", "description": studyDesc, "deadline": "2026-12-11", "team_size": 5},
		"description": {"title": "X", "description": "too short", "deadline": "2026-12-11", "team_size": 5},
		"deadline":    {"title": "X", "description": studyDesc, "deadline": "2026-10-28", "team_size": 5},
		"team_size":   {"title": "X", "description": studyDesc, "deadline": "2026-12-11", "team_size": 7},
	}
	for field, body := range cases {
		r := c.must(400, "POST", "/projects", body)
		if r.errField(field) == "" {
			t.Errorf("%s: not marked: %s", field, r.Raw)
		}
	}
	// Exactly 7 days from today is allowed, and 3 or 6 seats.
	c.must(201, "POST", "/projects", map[string]any{"title": "X", "description": studyDesc, "deadline": "2026-10-29", "team_size": 3})
	c.must(201, "POST", "/projects", map[string]any{"title": "X", "description": studyDesc, "deadline": "2026-10-29", "team_size": 6})
	assertCount(t, "projects", 2)
}

func TestInviteCode_UniquePerProject(t *testing.T) {
	e := newEnv(t)
	c := e.user("Aruzhan Sadykova")
	seen := map[string]bool{}
	for range 20 {
		_, code := c.project(3)
		if seen[code] {
			t.Fatalf("duplicate code %s", code)
		}
		seen[code] = true
	}
	if _, err := testStore.Pool.Exec(context.Background(), `UPDATE projects SET invite_code = (SELECT invite_code FROM projects LIMIT 1)`); err == nil {
		t.Fatal("database allowed two projects with one code")
	}
}

func TestInviteCode_NewCodeInvalidatesOld(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, old := owner.project(5)
	r := owner.must(200, "POST", "/projects/"+id+"/invite-code", nil)
	fresh := obj(r.Body["project"])["invite_code"].(string)
	if fresh == old || !domain.ValidInviteCode(fresh) {
		t.Fatalf("new code %q (old %q)", fresh, old)
	}
	m := e.user("Daniyar Kenzhebekov")
	if r := m.join(old); r.Status != 404 || r.errMessage() != domain.MsgInvalidInvite {
		t.Fatalf("old code still works: %s", r.Raw)
	}
	m.must(200, "POST", "/invites/"+fresh+"/join", nil)
}

func TestProjects_ListAndStatusChips(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(3)
	status := func() string {
		r := owner.must(200, "GET", "/projects", nil)
		return obj(list(r.Body["projects"])[0])["status"].(string)
	}
	if s := status(); s != "plan_draft" {
		t.Fatalf("new project: %s", s)
	}
	owner.must(200, "POST", "/projects/"+id+"/plan/generate", nil)
	owner.must(200, "POST", "/projects/"+id+"/plan/approve", nil)
	if s := status(); s != "waiting_profiles" {
		t.Fatalf("approved, seats open: %s", s)
	}
	a, b := e.user("Daniyar Kenzhebekov"), e.user("Madina Tulegenova")
	a.must(200, "POST", "/invites/"+code+"/join", nil)
	b.must(200, "POST", "/invites/"+code+"/join", nil)
	owner.designProfile()
	a.designProfile()
	if s := status(); s != "waiting_profiles" {
		t.Fatalf("one profile missing: %s", s)
	}
	b.designProfile()
	if s := status(); s != "matching_pending" {
		t.Fatalf("all ready: %s", s)
	}
	owner.must(200, "POST", "/projects/"+id+"/matching/run", nil)
	if s := status(); s != "matching_pending" {
		t.Fatalf("suggested only: %s", s)
	}
	owner.must(200, "POST", "/projects/"+id+"/matching/accept", map[string]any{})
	if s := status(); s != "roles_confirmed" {
		t.Fatalf("all accepted: %s", s)
	}
	// The member sees the same project, as Member.
	r := a.must(200, "GET", "/projects", nil)
	card := obj(list(r.Body["projects"])[0])
	if card["is_owner"] != false || len(list(card["members"])) != 3 {
		t.Fatalf("member card: %v", card)
	}
}

// ---- 6.3 Joining ----

func TestJoin_ValidCode(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(5)
	m := e.user("Daniyar Kenzhebekov")
	inv := m.must(200, "GET", "/invites/"+code, nil)
	i := obj(inv.Body["invite"])
	if i["member_count"].(float64) != 1 || i["team_size"].(float64) != 5 || i["owner_name"] != "Aruzhan Sadykova" || i["is_member"] != false {
		t.Fatalf("invite: %v", i)
	}
	r := m.must(200, "POST", "/invites/"+code+"/join", nil)
	if r.Body["project_id"] != id || r.Body["profile_ready"] != false || r.Body["already_member"] != false {
		t.Fatalf("join: %s", r.Raw)
	}
	m.must(200, "GET", "/projects/"+id, nil)
	// Lower-case input is accepted.
	e.user("Madina Tulegenova").must(200, "POST", "/invites/"+lower(code)+"/join", nil)
}

func TestJoin_ProfileReady(t *testing.T) {
	e := newEnv(t)
	_, code := e.user("Aruzhan Sadykova").project(5)
	m := e.user("Daniyar Kenzhebekov")
	m.designProfile()
	if r := m.must(200, "POST", "/invites/"+code+"/join", nil); r.Body["profile_ready"] != true {
		t.Fatalf("profile_ready: %s", r.Raw)
	}
}

func TestJoin_InvalidCode(t *testing.T) {
	e := newEnv(t)
	e.user("Aruzhan Sadykova").project(5)
	m := e.user("Daniyar Kenzhebekov")
	for _, code := range []string{"ZZZZZZ", "abc", "K7Q2P0"} {
		r := m.join(code)
		if r.Status != 404 || r.errMessage() != domain.MsgInvalidInvite {
			t.Fatalf("%s: %s", code, r.Raw)
		}
		if r := m.do("GET", "/invites/"+code, nil); r.errMessage() != domain.MsgInvalidInvite {
			t.Fatalf("GET %s: %s", code, r.Raw)
		}
	}
	assertCount(t, "project_members", 1)
}

func TestJoin_TeamIsFull(t *testing.T) {
	e := newEnv(t)
	_, code := e.user("Aruzhan Sadykova").project(3)
	e.user("Daniyar Kenzhebekov").must(200, "POST", "/invites/"+code+"/join", nil)
	e.user("Madina Tulegenova").must(200, "POST", "/invites/"+code+"/join", nil)
	r := e.user("Timur Akhmetov").join(code)
	if r.Status != 409 || r.errMessage() != "Team is full: maximum 3 members" {
		t.Fatalf("got %s", r.Raw)
	}
	assertCount(t, "project_members", 3)
}

func TestJoin_ConcurrentLastSeat(t *testing.T) {
	e := newEnv(t)
	_, code := e.user("Aruzhan Sadykova").project(3)
	e.user("Daniyar Kenzhebekov").must(200, "POST", "/invites/"+code+"/join", nil)
	var users []*client
	for i := range 6 {
		users = append(users, e.user(fmt.Sprintf("Racer Number%d", i)))
	}
	var wg sync.WaitGroup
	statuses := make([]int, len(users))
	for i, u := range users {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[i] = u.join(code).Status
		}()
	}
	wg.Wait()
	ok := 0
	for _, s := range statuses {
		if s == 200 {
			ok++
		} else if s != 409 {
			t.Fatalf("unexpected status %d", s)
		}
	}
	if ok != 1 {
		t.Fatalf("%d people took the last seat", ok)
	}
	assertCount(t, "project_members", 3)
}

func TestJoin_AlreadyMember(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(3)
	r := owner.must(200, "POST", "/invites/"+code+"/join", nil)
	if r.Body["already_member"] != true || r.Body["project_id"] != id {
		t.Fatalf("got %s", r.Raw)
	}
	if obj(owner.must(200, "GET", "/invites/"+code, nil).Body["invite"])["is_member"] != true {
		t.Fatal("is_member must be true")
	}
	assertCount(t, "project_members", 1)
}

func TestJoin_BeforePlanApproved(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(4)
	owner.must(200, "POST", "/projects/"+id+"/plan/generate", nil)
	m := e.user("Daniyar Kenzhebekov")
	inv := obj(m.must(200, "GET", "/invites/"+code, nil).Body["invite"])
	if len(list(inv["open_roles"])) != 4 {
		t.Fatalf("open roles: %v", inv["open_roles"])
	}
	m.must(200, "POST", "/invites/"+code+"/join", nil)
}

// ---- 6.5 Skills profile ----

func TestProfile_SaveValid(t *testing.T) {
	e := newEnv(t)
	c := e.user("Aruzhan Sadykova")
	if r := c.must(200, "GET", "/me/profile", nil); r.Body["profile"] != nil {
		t.Fatal("new user must have no profile")
	}
	c.designProfile()
	p := obj(c.must(200, "GET", "/me/profile", nil).Body["profile"])
	if len(list(p["tags"])) != 4 {
		t.Fatalf("tags: %v", p["tags"])
	}
	// Tags are optional.
	c.saveProfile("I organise group work well and keep deadlines and like presenting to people.", "Not strong at backend code.")
}

func TestProfile_TooShort(t *testing.T) {
	e := newEnv(t)
	c := e.user("Aruzhan Sadykova")
	for _, body := range []map[string]any{
		{"strengths": "short", "weaknesses": "Not strong at writing backend code at all.", "tags": []string{}},
		{"strengths": "I organise group work well and keep deadlines.", "weaknesses": "short", "tags": []string{}},
		{"strengths": "I organise group work well, keep deadlines", "weaknesses": "No backend.", "tags": []string{}},
	} {
		r := c.must(400, "PUT", "/me/profile", body)
		if r.errMessage() != domain.MsgProfileTooShort {
			t.Fatalf("message %q", r.errMessage())
		}
	}
	assertCount(t, "profiles", 0)
}

func TestProfile_TagLimits(t *testing.T) {
	e := newEnv(t)
	c := e.user("Aruzhan Sadykova")
	p := designProfiles["Aruzhan Sadykova"]
	tags := []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "a9", "a10", "a11"}
	r := c.must(400, "PUT", "/me/profile", map[string]any{"strengths": p[0], "weaknesses": p[1], "tags": tags})
	if r.errField("tags") != domain.MsgTooManyTags {
		t.Fatalf("got %s", r.Raw)
	}
	r = c.must(400, "PUT", "/me/profile", map[string]any{"strengths": p[0], "weaknesses": p[1], "tags": []string{"a-very-long-tag-name-over-24"}})
	if r.errField("tags") != domain.MsgTagLength {
		t.Fatalf("got %s", r.Raw)
	}
}

func TestProfile_DeleteTransfersOwnership(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(5)
	d := e.user("Daniyar Kenzhebekov")
	d.must(200, "POST", "/invites/"+code+"/join", nil)
	e.user("Madina Tulegenova").must(200, "POST", "/invites/"+code+"/join", nil)
	owner.designProfile()
	r := owner.must(200, "GET", "/me/profile", nil)
	tr := obj(list(r.Body["ownership"])[0])
	if tr["new_owner_name"] != "Daniyar" || tr["project_title"] != "StudySpace" {
		t.Fatalf("transfer preview: %v", tr)
	}
	owner.must(204, "DELETE", "/me/profile", nil)
	if owner.must(200, "GET", "/auth/me", nil).Body["user"] != nil {
		t.Fatal("deleted user still logged in")
	}
	b := d.must(200, "GET", "/projects/"+id, nil)
	if b.Body["is_owner"] != true || len(list(b.Body["members"])) != 2 {
		t.Fatalf("after delete: %s", b.Raw)
	}
	assertCount(t, "profiles", 0)
	assertCount(t, "users", 2)
}

func TestProfile_DeleteRemovesEmptyProject(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	owner.project(5)
	owner.must(204, "DELETE", "/me/profile", nil)
	assertCount(t, "projects", 0)
	assertCount(t, "users", 0)
}

// ---- 6.10 Settings and access ----

func TestSettings_UpdateDetails(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, _ := owner.project(5)
	body := map[string]any{"title": "StudySpace 2", "description": studyDesc, "deadline": "2026-10-22", "context": "PjM 101"}
	r := owner.must(200, "PATCH", "/projects/"+id, body)
	if r.Body["description_changed"] != false || obj(r.Body["project"])["title"] != "StudySpace 2" {
		t.Fatalf("got %s", r.Raw)
	}
	body["description"] = studyDesc + " We also need basic analytics."
	if r := owner.must(200, "PATCH", "/projects/"+id, body); r.Body["description_changed"] != true {
		t.Fatalf("description change not reported: %s", r.Raw)
	}
	body["deadline"] = "2026-10-21"
	if r := owner.must(400, "PATCH", "/projects/"+id, body); r.errField("deadline") != domain.MsgDeadlinePast {
		t.Fatalf("past deadline accepted: %s", r.Raw)
	}
}

func TestSettings_OwnerOnly(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(5)
	m := e.user("Daniyar Kenzhebekov")
	m.must(200, "POST", "/invites/"+code+"/join", nil)
	m.must(403, "PATCH", "/projects/"+id, map[string]any{"title": "Hacked"})
	m.must(403, "POST", "/projects/"+id+"/invite-code", nil)
	m.must(403, "DELETE", "/projects/"+id+"/members/"+owner.ID, nil)
	m.must(403, "POST", "/projects/"+id+"/plan/generate", nil)
}

func TestSettings_RemoveMember(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, code := owner.project(5)
	m := e.user("Daniyar Kenzhebekov")
	m.must(200, "POST", "/invites/"+code+"/join", nil)
	owner.must(204, "DELETE", "/projects/"+id+"/members/"+m.ID, nil)
	m.must(404, "GET", "/projects/"+id, nil)
	assertCount(t, "project_members", 1)
}

func TestSettings_CannotRemoveOwner(t *testing.T) {
	e := newEnv(t)
	owner := e.user("Aruzhan Sadykova")
	id, _ := owner.project(5)
	owner.must(400, "DELETE", "/projects/"+id+"/members/"+owner.ID, nil)
}

func TestNonMember_Gets404(t *testing.T) {
	e := newEnv(t)
	id, _ := e.user("Aruzhan Sadykova").project(5)
	x := e.user("Stranger Danger")
	for _, path := range []string{"", "/plan", "/matching", "/raci", "/tasks", "/dashboard"} {
		x.must(404, "GET", "/projects/"+id+path, nil)
	}
	x.must(404, "GET", "/projects/not-a-uuid", nil)
}

func TestRegister_SecureCookieOverHTTPS(t *testing.T) {
	e := newEnv(t, withConfig(func(c *config.Config) { c.AppURL = "https://teamload.example"; c.SecureCookies = true }))
	r, err := e.anon().http.Post(e.srv.URL+"/api/auth/register", "application/json",
		stringsReader(`{"full_name":"Dana Bekova","email":"dana@x.kz","password":"studyspace26","consent":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	for _, ck := range r.Cookies() {
		if ck.Name == sessionCookie && !ck.Secure {
			t.Fatal("cookie must be Secure under https")
		}
	}
}

func TestCreateProject_AIFailsProjectStillSaved(t *testing.T) {
	e := newEnv(t, withProvider(failingProvider{}))
	c := e.user("Aruzhan Sadykova")
	id, _ := c.project(4)
	r := c.must(502, "POST", "/projects/"+id+"/plan/generate", nil)
	if r.errMessage() != domain.MsgAIUnavailable {
		t.Fatalf("message %q", r.errMessage())
	}
	plan := c.must(200, "GET", "/projects/"+id+"/plan", nil)
	if len(list(obj(plan.Body["plan"])["seats"])) != 0 {
		t.Fatal("no seats expected after AI failure")
	}
	assertCount(t, "projects", 1)
	assertCount(t, "ai_calls", 2) // first attempt + one retry
}
