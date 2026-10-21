# TeamLoad — build brief for Claude Code

You are building **TeamLoad**, a web app for university project teams, **from an empty repository** to a finished, demo-ready MVP. A clickable HTML prototype is attached (`design.html`; if it is not in the repo yet, copy it to `design/design.html` first). This brief is self-contained: it holds every business rule, message and constraint you need. Read the **whole** brief and the **whole** design file before writing any code.

Talk to me in **Russian**. Code, comments, commit messages, README and all UI text stay in **English**.

---

## 1. Sources of truth

1. **The design** defines the look, layout, screens, navigation, components, microcopy and interactions. Reproduce it faithfully: same tokens, fonts, spacing, radii, states and wording.
2. **This brief** defines business rules, data, API, AI behaviour, validation and error messages.
3. **If they conflict, this brief wins.** Every known conflict is listed in §12 "Deliberate deviations from the design". Do not invent other deviations.
4. If something is in neither, choose the simplest option that fits the design and record it in `docs/DECISIONS.md` (one line: decision + reason). Ask me only when a choice would be expensive to undo.
5. **Do not add features** that are not here. Out of scope for the MVP: RACI export (PDF/CSV), course milestone templates, deadline reminders (email/Telegram), role-rotation suggestions, instructor view, peer review, social login, password reset, MFA, mobile app, university SSO, websockets.

## 2. Product in one paragraph

Students describe their project; an AI designs one role per seat (with responsibilities, skills and effort share); teammates join with an invite code and write a short skills profile; the AI matches each person to a role and explains why by quoting their profile; the team accepts or overrides every suggestion (**the AI suggests, people decide**). A RACI matrix is drafted by AI and edited by hand; a task board with hour estimates feeds a workload dashboard that flags who is overloaded or underloaded.

**Main flow:** register → My projects → New project (title, description, deadline, team size) → AI role plan (draft) → edit / regenerate / approve → invite teammates (code or link) → each member fills in the profile → AI matching → accept / reassign → RACI → board → dashboard.

## 3. Tech stack (fixed — do not change without asking)

| Layer | Choice |
|---|---|
| Backend | Go (current stable), `chi` v5 router, `pgx/v5` + `sqlc`, `goose` migrations, `log/slog`, `golang-jwt/jwt/v5`, `x/crypto/bcrypt` |
| Database | PostgreSQL 16 |
| Frontend | React (current stable) + TypeScript (`strict`) + Vite, React Router, TanStack Query v5, `@dnd-kit/core` for the board |
| Styling | Plain CSS ported from the design (CSS variables on `:root`, light + dark). **No Tailwind, no UI kit, no chart library** — the dashboard chart is the design's SVG as a React component |
| AI | LLM called **only from the backend**, behind one Go interface, strict JSON output, `mock` provider by default (§7) |
| Runtime | Docker Compose: `db`, `api`, `web` (nginx serving the built SPA and proxying `/api` to `api`) |
| CI | GitHub Actions on every PR: `go vet`, `golangci-lint`, `go test`, `tsc --noEmit`, `eslint`, `vitest`, frontend build |

Repository layout:

```
/backend
  cmd/api/            main.go (HTTP server, runs migrations on start)
  cmd/seed/           demo data loader
  cmd/aieval/         AI accuracy & bias evaluation tool (§7.6)
  internal/config/    env parsing
  internal/http/      router, handlers, middleware (auth, project member, owner)
  internal/domain/    pure business rules (validation, load status, RACI check, invite codes)
  internal/store/     sqlc queries + transactions
  internal/ai/        provider interface, mock, gemini, prompts/*.tmpl, JSON schemas, validation
  migrations/
/frontend
  src/styles/         tokens.css, base.css, components.css, screens/*.css (ported from the design)
  src/components/     Button, Panel, Chip, Avatar, AvatarStack, Notice, Field, Input, Textarea+Counter,
                      Stepper, EffortBar, Legend, Modal, Toast, LoadingOverlay, Icon (design's SVG set)
  src/layout/         AppShell (sidebar + main), PageHead
  src/pages/          one file per screen (§5)
  src/api/            typed client + TanStack Query hooks
/design/design.html
/docs/                DECISIONS.md, retros/ (sprint retrospectives live here)
docker-compose.yml, docker-compose.prod.yml, Makefile, .env.example, README.md
```

## 4. Implementing the design

- Port `:root` tokens **verbatim** (light, `prefers-color-scheme: dark`, and the `[data-theme]` overrides). Load Bricolage Grotesque, Figtree and JetBrains Mono exactly as the design does.
- Port every component style: buttons (primary / danger / ghost / sm / disabled), panels, chips (ok / warn / bad / ai / line), avatars with role colours `--r1…--r6`, notices (ai / warn / bad), effort bar, fit bar, stepper, seats, RACI cells, board columns and cards, KPI tiles, load table, share bar, toast, overlay + modal, loading card with step list.
- Keep both breakpoints: **≤980px** (sidebar becomes a horizontal scrolling nav, project picker and user block hidden) and **≤700px** (single-column board, grid and match rows).
- Keep the accessibility the design already has: `aria-current`, `aria-pressed`, `role="tablist"`, labelled inputs, `:focus-visible`, live regions. Add: focus trap + `Esc` in modals, keyboard support on the board (dnd-kit keyboard sensor; the edit modal's Status field is the non-drag fallback), `prefers-reduced-motion`.
- Replace all prototype artefacts with real behaviour: sample data → API data; the "Qazaq Recipes Bot" card → the user's real projects; "Simulated AI call" label → removed; every "Prototype: … would be …" toast → the real action plus a real success toast.
- **LoadingOverlay for AI calls:** show the design's step list and tick the steps on a timer while the request runs; tick all and close when the response arrives; on error, close and show the error (§6) on the page.
- Add the states the prototype does not draw, in the same visual language: page skeletons while loading, empty states (no projects, no tasks, no RACI yet, matching not run), error notices, 404 / no access.
- Dates: `en-GB` format as in the design ("11 Dec 2026"). Hours: one decimal max, trailing `.0` removed.

## 5. Screens, routes and permissions

| Route | Screen (design function) | Access |
|---|---|---|
| `/login` | Login / Create account tabs (`vLogin`) | Guests only; logged-in users are redirected to `/projects` |
| `/projects` | My projects (`vProjects`) | Logged in |
| `/projects/new` | Describe your project (`vNew`) | Logged in |
| `/join` and `/join/:code` | Join card (`vJoin`); without a code, show a code input card in the same style | Logged in (guests go to `/login?next=…` and come back) |
| `/profile` | My skills profile + delete profile (`vProfile`) | Logged in |
| `/p/:projectId/plan` | Role plan (`vPlan`) | Members view; **owner** edits, regenerates, approves |
| `/p/:projectId/roles` | Team roles / matching (`vRoles`) | Members view; **owner** runs matching, accepts, reassigns |
| `/p/:projectId/raci` | RACI (`vRaci`) | Members view; **owner** edits, adds, regenerates (others see cells read-only) |
| `/p/:projectId/board` | Task board (`vBoard`) | **All members** create, edit, move, delete tasks |
| `/p/:projectId/dashboard` | Workload (`vDash`) | Members |
| `/p/:projectId/settings` | Project settings (`vSettings`) | **Owner only** |

The sidebar shows the current project (title, team size, deadline) and the nav from the design, including the plan status counter (`✓` / `draft`) and the open-tasks count. The user block shows the real name and "Owner" / "Member" for the current project. Non-members get 404 on any `/p/:id/*` route and API call.

## 6. Business rules and acceptance criteria

Error messages in quotes are **exact strings** (QA tests them). Every rule is validated **on the server**; the client mirrors it for instant feedback.

### 6.1 Registration and login
- Fields: full name (2–100 chars), email (valid format, unique case-insensitively, stored lower-case), password (min 8 chars), consent checkbox (required; store `consent_at`). A university email is preferred but not enforced (keep the design's hint).
- Password: bcrypt, cost 12. Session: JWT in an `httpOnly`, `SameSite=Lax` cookie (`Secure` when `APP_URL` is https), 7 days. `GET /api/auth/me` restores the session.
- Valid sign-up → account created, session started, redirect to `next` if present, else `/projects`.
- Email taken → **"Email already registered"**, nothing created.
- Invalid email / short password → the wrong field is marked with an inline error, nothing saved.
- No consent → **"Please accept personal-data processing to continue"**, nothing created.
- Wrong password or unknown email → **"Invalid email or password"** (never reveal which). Rate-limit login: 10 attempts per 15 min per IP.
- Already logged in → opening the app or `/login` goes straight to `/projects`.
- Log out → session ends, `/login` shown, project pages inaccessible until the next login.

### 6.2 Projects and invite codes
- Create: title 1–100; description 100–3000 (live counter `N / 3000 · min 100` as in the design); deadline at least 7 days from today (timezone `APP_TZ`, default `Asia/Almaty`); team size 3–6 including the owner (stepper + seats from the design); context optional, max 100.
- Creator becomes **owner** and first member. Submitting the form saves the project, then generates the role plan (§6.4) behind the "Designing your team" overlay and lands on `/p/:id/plan` as a **draft**. If the AI fails, the project is still saved and the plan page shows the error with "Retry" (§6.4 fallback).
- Invite code: 6 characters from `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`, unique (DB unique constraint, regenerate on collision). Two projects never share a code. Codes do not expire; the owner can "Make a new code" in Settings and the old code stops working immediately.
- Invite link: `${APP_URL}/join/{CODE}`.
- My projects shows every project the user owns or joined, as in the design: status chip, Owner / Member, deadline + days left, progress bar (done hours / all hours), avatar stack, "x / y h done". Status chip priority: **"Plan draft"** (warn) → **"Waiting for profiles"** (warn, someone has no ready profile or seats are open) → **"Matching pending"** (warn) → **"Roles confirmed"** (ok). Clicking a card opens the dashboard if roles are confirmed, otherwise the role plan.

### 6.3 Joining a project
- Join page shows title, context, owner, first ~190 chars of the description, deadline, real "`N / teamSize` joined", open role names (if a plan exists), and the code.
- Valid code → user is added and sent to `/profile` if their profile is not ready, otherwise to the project's role plan. Toast: "You joined {title}. Fill in your profile next." (or "You joined {title}." when the profile is ready).
- Unknown code → **"Invalid invite code"**, not added.
- Project full (members = team size) → **"Team is full: maximum {teamSize} members"**, not added. Check capacity inside a transaction that locks the project row (`SELECT … FOR UPDATE`), so two people cannot take the last seat at once.
- Already a member → go straight to the project, no error.
- Members can join before the plan is approved (as the design says).

### 6.4 Role plan (AI)
- Exactly **team size** seats. Each seat: name (≤40), summary (≤140), 3–5 responsibilities (≤80 each), 3–5 skills (≤30 each), effort share (integer 5–60), "why this project needs it" (≤300). Shares sum to **exactly 100**. Seat colours: `--r1…--r6` by seat order.
- Status `draft` / `approved`. Owner actions, exactly as in the design:
  - **Edit**: name, summary and effort share inline; "Effort total N% · must be 100%" turns red and "Save changes" is disabled when the sum ≠ 100.
  - **Regenerate** (optional free-text note, max 500 chars): replaces the seats, status → `draft`. If matching or RACI exist, confirm first in a modal ("This replaces the roles. Matching and RACI will be cleared.") and clear them on confirm.
  - **Approve plan**: only when the sum is 100. Unlocks matching and RACI.
- Fallback when the AI fails twice: notice **"AI service temporarily unavailable. Please retry or assign roles manually"** with "Retry" and "Use a standard plan". The standard plan is built without AI from the role catalogue (§7.3) for this team size, with equal-ish shares summing to 100, as a normal editable draft.
- The invite panel on this page shows the code, "Copy code", "Copy link" and "`N / teamSize` joined".

### 6.5 Skills profile
- One profile per user, shared by all their projects (as the design says). Fields: strengths (max 1000), weaknesses (max 1000), **optional skill tags** (max 10 tags, each 1–24 chars, chip input — add it under Weaknesses in the design's style; see §12).
- Saving requires strengths ≥ 30, weaknesses ≥ 10 **and** strengths + weaknesses ≥ 100 characters in total; otherwise **"Write at least 100 characters (about two sentences)"** and nothing is saved. A saved profile is **ready for AI matching**; tags are optional.
- Saving a changed profile marks this user's AI suggestions in all projects as **outdated** (chip on the roles page, and "Run again" is suggested); the new text is used on the next matching run.
- **Delete my profile** (inline confirm as in the design): deletes the account and all personal data (profile, memberships, AI results); their tasks become unassigned; for every project they own, ownership passes to the earliest-joined remaining member (the confirm text names that person), or the project is deleted if nobody is left.

### 6.6 Team roles (AI matching)
- "Run AI matching" is enabled when the plan is **approved**, the project has **at least 3 members**, and **every member has a ready profile**. Otherwise show the design's "not ready" state with the reason (plan not approved / who still needs a profile).
- One person per seat, one seat per person. If there are fewer members than seats, the leftover seats stay **Unassigned** (owner can assign later). Enforce with unique constraints on `(project_id, seat_id)` and `(project_id, user_id)`.
- Each row shows the design's content: seat name + share, avatar + person select, fit score 0–100 with bar, explanation (2–3 sentences, quoting the profile, addressed to the person as "you"), "Also a good fit: {runner-up seat}", status chip.
- Row states: **Suggested** (AI, not accepted) → **Accepted** (`ai_accepted`) or **Set manually** (`manual`, fit shown as "—", explanation replaced by the design's "Changed by the team…" text).
- **Accept** (one row), **Accept all**, **Run again** (confirm if any row is accepted; replaces all rows).
- **Reassign**: choosing another person swaps the two people's seats; both rows become `manual` + accepted (toast as in the design). After confirmation the owner can still change roles; the new value replaces the old one.
- Skill-gap notice and growth note come from the AI output (no hard-coded text). Hide each notice when empty.
- AI failure → **"AI service temporarily unavailable. Please retry or assign roles manually"**; profiles are kept and every row gets a person select so the owner can assign all seats manually.

### 6.7 RACI
- Needs an approved plan. If no RACI exists: empty state with "Draft RACI with AI" (owner). The AI drafts 6–10 activities in timeline order up to the deadline, with one value per seat.
- Values: `R`, `A`, `C`, `I` or empty. A click cycles `empty → R → A → C → I → empty` (keep focus on the cell).
- Check column per row: **"No A"**, **"Several A"**, **"No R"** or OK; the header chip shows "N rows need fixing" or "Every activity has one A and at least one R". Invalid rows are allowed while editing (do **not** enforce this as a DB constraint).
- Owner can add an activity (default name "New activity", rename inline), rename and delete activities, and regenerate (confirm overwrite).
- Column headers show the seat and the assigned person's first name, or "Unassigned".

### 6.8 Task board
- Task: title 1–120, assignee (project member or Unassigned), estimate in hours (step 0.5, max 40), status `todo` / `doing` / `done`, optional RACI activity.
- Estimate empty or ≤ 0 → **"Enter an estimate greater than 0 hours"**; above 40 → **"Estimate must be between 0.5 and 40 hours."**; empty title → **"Give the task a title."**
- Drag cards between columns (dnd-kit, mouse + touch + keyboard) and click a card to edit in the modal; delete from the modal. Column headers show count and total hours. Filters: Everyone / each member / Unassigned (the dashboard's "Assign on the board" link opens the board filtered to Unassigned).
- Cards show the assignee and their confirmed role name, as in the design.
- Other members see changes without a reload: refetch every 10 s and on window focus; optimistic updates with rollback on error.
- Reassigning a task or changing its estimate updates the dashboard immediately.

### 6.9 Workload dashboard
- **Active load** of a member = sum of estimates of their tasks in To do + Doing. **Team average** = total active load of members ÷ number of members (members with 0 h count).
- **Overloaded**: active > **130%** of the average (strictly greater: exactly 130% is **not** overloaded). **Underloaded**: active < 75% of the average. Otherwise **Balanced**. Put the thresholds in one constants file shared by the domain package and the frontend.
- Reference case (must pass): Akzhol 14, Beknur 22, Yerasyl 16, Irisbek 15, Zhanarys 9 → average 15.2, threshold 19.76 → only Beknur (145%) is Overloaded (red); Zhanarys (59%) is Underloaded.
- No tasks with estimates → **"No estimated tasks found. Add tasks with hour estimates on the Task Board."** instead of the chart.
- Everything else as in the design: KPIs (days to deadline, active hours, team average, not assigned), overload notice suggesting who has free time, SVG bar chart with the balanced band (75–130%) and dashed average line, table with role, active, done, tasks, share of all hours vs planned effort share, status chip.

### 6.10 Project settings (owner only)
- Edit title, description, deadline (≥ today), context. If the description changed, show a modal offering to regenerate the role plan (same confirm rules as §6.4).
- Team size is fixed after creation in the MVP.
- Invite code: copy, make a new code (old one stops working at once).
- Members list "N / teamSize", role per member. Remove a member (inline confirm as in the design): their tasks become unassigned and their seat becomes Unassigned. The owner cannot be removed.

## 7. AI integration

### 7.1 Interface
```go
type Provider interface {
    GenerateRolePlan(ctx context.Context, in RolePlanInput) (RolePlanOutput, error)
    MatchRoles(ctx context.Context, in MatchInput) (MatchOutput, error)
    DraftRACI(ctx context.Context, in RACIInput) (RACIOutput, error)
}
```
Providers: **`mock`** (default; used in dev, CI and tests) and **`gemini`** (Google Gemini API free tier, JSON mode with a response schema). Select with `LLM_PROVIDER`, `LLM_API_KEY`, `LLM_MODEL`. Adding another provider must only mean adding one file.

### 7.2 Reliability
- 30 s timeout per call (context). Parse into Go structs and validate every rule below; on any failure **retry once** with the validation errors appended to the prompt; after a second failure return the "AI service temporarily unavailable…" error.
- Spending guard: max `LLM_MAX_CALLS_PER_PROJECT_PER_DAY` (default 30) AI calls per project per day → HTTP 429 with "Daily AI limit reached for this project. Try again tomorrow or edit manually."
- Log provider, latency, attempt and result. **Never log profile or description text.**
- Store every AI result with `source`, `prompt_version` and `generated_at`.

### 7.3 Prompts
- Templates live in `backend/internal/ai/prompts/*.tmpl` with a version constant, readable and editable by our AI engineer without touching Go code.
- Role catalogue used as guidance (the AI may combine or specialise roles for the project, e.g. "Project Manager & QA Lead"): Project Manager, Backend Developer, Frontend Developer, UX/UI Designer, QA Engineer, AI / Data Engineer.
- Role plan input: title, description, deadline, weeks left, team size, context, optional owner note.
- RACI input: project basics + approved seats.

### 7.4 Privacy and fairness (charter requirement)
- Matching input contains **only** profile text and tags. **Never send names, emails, gender, age or nationality.** Persons are labelled `P1…Pn` and **shuffled on every call**; map back to users on the server.
- Explanations must quote the person's own words, address them as "you", and never contain `P<number>` labels (reject and retry if they do).

### 7.5 Output schemas (validate strictly)
```json
// Role plan
{ "seats": [ { "id": "S1", "name": "…", "summary": "…", "responsibilities": ["…"], "skills": ["…"], "effort_share": 20, "why": "…" } ] }
// exactly team_size seats; ids S1..Sn; shares are integers 5–60 summing to 100; list lengths 3–5

// Matching
{ "assignments": [ { "seat_id": "S2", "person": "P3", "fit": 88, "runner_up_seat_id": "S4", "explanation": "…" } ],
  "gaps": [ { "skill": "Accessibility testing", "seat_id": "S4", "reason": "…" } ],
  "growth_note": "…" }
// every person exactly once; each seat at most once; fit 0–100; explanation 2–3 sentences; growth_note may be null

// RACI
{ "activities": [ { "name": "Define scope & backlog", "cells": { "S1": "A", "S2": "C", "S3": "R" } } ] }
// 6–10 activities; values R|A|C|I|""; every activity has exactly one A and at least one R
```

### 7.6 Mock provider and evaluation tool
- The mock must produce **realistic, input-dependent** output so the demo works without an API key: role plan from the catalogue sized to the team; matching by keyword overlap between profile text/tags and seat skills with an optimal one-to-one assignment, fit scores, explanations that quote a phrase from the profile; a RACI template from the seats. Add a 1–2 s delay so the loading overlay is visible.
- `make ai-eval` runs `cmd/aieval` on `backend/testdata/ai_eval/profiles.json` (10 profiles with expected roles) against the configured provider and prints: accuracy (target ≥ 7/10) and a **swapped-name bias check** (same profiles under different names must get the same role in ≥ 9/10 cases). Create the fixture file with 10 varied sample profiles.

## 8. Data model (PostgreSQL)

Design the schema yourself, with at least these tables and constraints enforced **in the database**:

- `users` (email unique lower-case, password_hash, full_name, consent_at, created_at)
- `profiles` (user_id PK/FK, strengths, weaknesses, tags text[], updated_at)
- `projects` (title, description, deadline date, team_size CHECK 3–6, context, invite_code UNIQUE, owner_id, plan_status CHECK draft/approved)
- `project_members` (project_id, user_id, joined_at, UNIQUE(project_id, user_id))
- `plan_seats` (project_id, position, name, summary, responsibilities text[], skills text[], effort_share CHECK 5–60, why, color_token)
- `role_assignments` (project_id, seat_id, user_id nullable, fit, runner_up_seat_id, explanation, state, source CHECK ai/ai_accepted/manual, outdated bool; UNIQUE(project_id, seat_id), UNIQUE(project_id, user_id))
- `raci_activities` (project_id, position, name) and `raci_cells` (activity_id, seat_id, value CHECK IN ('R','A','C','I'))
- `tasks` (project_id, title, assignee_id nullable, estimate numeric(4,1) CHECK 0.5–40, status CHECK todo/doing/done, raci_activity_id nullable, timestamps)
- `ai_calls` (project_id, kind, provider, prompt_version, attempt, ok, latency_ms, created_at) for the spending guard

Use `ON DELETE` rules that implement §6.5 and §6.10 (unassign tasks, clear seats), transactions for multi-step writes, and indexes on every foreign key used in lookups.

## 9. API

REST + JSON under `/api`. Errors are always `{"error": {"code": "…", "message": "…", "fields": {"email": "…"}}}` with proper status codes (400 validation, 401, 403, 404, 409 conflict, 429, 502 for AI failure). Middleware: auth, project membership (404 for non-members), owner-only. Parameterized SQL only. CORS only in dev.

Minimum endpoints: `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`; `GET|PUT|DELETE /me/profile`; `GET|POST /projects`; `GET|PATCH /projects/:id`; `POST /projects/:id/invite-code`; `GET /invites/:code`; `POST /invites/:code/join`; `DELETE /projects/:id/members/:userId`; `GET /projects/:id/plan`, `PUT /projects/:id/plan`, `POST /projects/:id/plan/generate`, `POST /projects/:id/plan/standard`, `POST /projects/:id/plan/approve`; `GET /projects/:id/matching`, `POST /projects/:id/matching/run`, `POST /projects/:id/matching/accept` (one or all), `PUT /projects/:id/matching/seats/:seatId`; `GET|PUT /projects/:id/raci`, `POST /projects/:id/raci/generate`; `GET|POST /projects/:id/tasks`, `PATCH|DELETE /projects/:id/tasks/:taskId`; `GET /projects/:id/dashboard`.

## 10. Quality bar (Definition of Done)

- From a clean clone: `cp .env.example .env && docker compose up --build` → app on **http://localhost:8080**, migrations applied automatically, works with `LLM_PROVIDER=mock` and no API key.
- `make seed` loads the design's demo data so the demo looks like the prototype: project **StudySpace** (description, deadline 11 Dec 2026, context "PjM 101 · Fall 2026"), the 5 members with their profiles from the design (Aruzhan's email is in the design; give the others the same pattern, e.g. `daniyar.k@stu.sdu.edu.kz`; password `studyspace26` for all), the approved 5-seat plan, accepted/pending matching rows, the 8 RACI activities and the 17 tasks.
- Tests:
  - Go unit tests for `internal/domain` (load status incl. the 130% boundary and the reference case, RACI check, invite code format, all validators).
  - API integration tests against a real Postgres for **every** scenario in §6, named after it (e.g. `TestJoin_TeamIsFull`, `TestRegister_NoConsent`).
  - AI layer tests: schema validation, retry-once, timeout, `P<number>` rejection, shuffling, using the mock and a fake failing provider.
  - Frontend: Vitest for utilities and key components; one Playwright happy path (register → create project → approve plan → second user joins and fills in profile → run matching → accept all → add task → dashboard).
- Zero `tsc`, ESLint and golangci-lint errors; no console errors in the browser.
- **Visual check:** screenshot every screen at 1280 px and 390 px, in light and dark, next to the same screen of `design.html`; fix every visible difference that is not in §12.
- Production: `docker-compose.prod.yml` for a single free-tier VM (all config through env vars, `APP_URL`, secure cookies under https).
- README: what TeamLoad is, run, seed, test, env vars, switching the LLM provider, `make ai-eval`, demo accounts, project structure.

## 11. Work plan

Work in phases. After each phase: run all tests and linters, commit with a clear message (Conventional Commits), and give me a short report in Russian (done / decisions / next). Do not start the next phase with failing tests.

0. **Read and plan.** Read this brief and the full design file. List screens, components and API calls in a short `docs/PLAN.md`. Ask me questions only if something blocks you.
1. **Foundation.** Monorepo, Docker Compose, migrations, CI, config. Port the design system (tokens, fonts, components, icons, AppShell, Toast, Modal, LoadingOverlay) and check it against the design.
2. **Sprint 1 scope — access and teams.** Registration / login / logout, profile (incl. tags and delete), My projects, New project (plan via mock), join by code and link, Settings (code, members).
3. **Sprint 2 scope — AI.** Provider layer (mock + Gemini), role plan (generate, edit, regenerate with note, standard-plan fallback, approve), matching (run, accept, reassign/swap, outdated, manual fallback), RACI.
4. **Sprint 3 scope — work tracking.** Task board (dnd, modal, filters, polling) and workload dashboard.
5. **Sprint 4 scope — polish.** Seed data, `ai-eval` tool + fixtures, the full test suite, visual pass on every screen, empty/error states, prod compose, README.

## 12. Deliberate deviations from the design

| Design prototype | Build this instead | Why |
|---|---|---|
| Overloaded above 125% of average; band text "75–125%" | Overloaded strictly above **130%**; band and subtitle "75–130%" | Charter objective 4 and US-09 |
| Profile: strengths ≥ 30, weaknesses ≥ 10 | Same, **plus** ≥ 100 characters in total | Backlog grooming decision |
| Profile form has no tags input | Optional tag chips (max 10) under Weaknesses | Skills profile feature includes tags |
| Prototype error texts (consent, estimate) | The exact strings in §6 | QA tests these strings |
| Settings hint mentions changing team size | Team size fixed after creation; hint says only the description triggers the regenerate question | Simpler MVP |
| Hard-coded skill-gap notice text | Built from the AI `gaps` output | Real data |
| Join page "members − 1" count, `teamload.local` link | Real member count, `${APP_URL}/join/CODE` | Prototype artefacts |
| All seats always filled | Seats can stay **Unassigned** when fewer members than seats | Teams may be incomplete |
| "Simulated AI call", "Prototype: …" toasts, sample second project | Real behaviour and data | Prototype artefacts |

## 13. Final report

When everything is done, give me in Russian: what was built, how to run and demo it (commands, demo accounts), test results, the list of decisions from `docs/DECISIONS.md`, and anything that still differs from the design and why.
