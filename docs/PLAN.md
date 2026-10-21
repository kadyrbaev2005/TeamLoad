# TeamLoad — implementation plan

Derived from `docs/SPEC.md` (business rules, wins on conflict) and `design/design.html` (look and wording).

## Screens

| Route | Page component | Design fn | Notes |
|---|---|---|---|
| `/login` | `LoginPage` | `vLogin` | Tabs Log in / Create account; guests only |
| `/projects` | `ProjectsPage` | `vProjects` | Real project cards, status chip priority, new-project card, empty state |
| `/projects/new` | `NewProjectPage` | `vNew` | Counter, stepper + seats, deadline ≥ 7 days; overlay "Designing your team" |
| `/join`, `/join/:code` | `JoinPage` | `vJoin` | Code input card when no code; real member count; guests → `/login?next=` |
| `/profile` | `ProfilePage` | `vProfile` | Strengths / weaknesses / tag chips; delete with inline confirm naming new owner |
| `/p/:id/plan` | `PlanPage` | `vPlan` | Edit inline, regenerate with note (confirm if matching/RACI), approve, AI-fail fallback, invite panel |
| `/p/:id/roles` | `RolesPage` | `vRoles` | Not-ready state, match rows, accept / accept all / run again / reassign-swap, outdated chip, gaps + growth note |
| `/p/:id/raci` | `RaciPage` | `vRaci` | Empty state, cell cycling, check column, add / rename / delete activity, regenerate |
| `/p/:id/board` | `BoardPage` | `vBoard` | dnd-kit columns, filters, task modal, 10 s polling, optimistic updates |
| `/p/:id/dashboard` | `DashboardPage` | `vDash` | KPIs, overload notice, SVG chart (75–130% band), load table |
| `/p/:id/settings` | `SettingsPage` | `vSettings` | Owner only: details, regenerate-plan modal on description change, invite code, members + remove |
| `*` | `NotFoundPage` | — | 404 / no access |

## Components

- **Layout:** `AppShell` (sidebar: brand, project picker, nav with plan ✓/draft and open-task count, user block with Owner/Member + logout), `PageHead` (eyebrow, h1, sub, actions).
- **UI:** `Button` (primary/danger/ghost/sm), `Panel`, `Chip` (ok/warn/bad/ai/line), `Avatar`, `AvatarStack`, `Notice` (ai/warn/bad), `Field`, `Input`, `Textarea` + `Counter`, `Stepper` + seats, `EffortBar`, `Legend`, `FitBar`, `Modal` (focus trap, Esc), `ConfirmModal`, `Toast` (live region), `LoadingOverlay` (timed step list), `Icon` (design SVG set), `Skeleton`, `EmptyState`, `TagInput`, `RaciCell`, `LoadChart` (SVG).
- **Styles:** `tokens.css` (verbatim `:root`, dark media query, `[data-theme]`), `base.css`, `components.css`, `screens/*.css`.

## API calls (all under `/api`)

| Area | Endpoints |
|---|---|
| Auth | `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`, `GET /auth/me` |
| Profile | `GET`, `PUT`, `DELETE /me/profile` |
| Projects | `GET`, `POST /projects`; `GET`, `PATCH /projects/:id`; `POST /projects/:id/invite-code`; `DELETE /projects/:id/members/:userId` |
| Invites | `GET /invites/:code`; `POST /invites/:code/join` |
| Plan | `GET`, `PUT /projects/:id/plan`; `POST /projects/:id/plan/generate`, `/plan/standard`, `/plan/approve` |
| Matching | `GET /projects/:id/matching`; `POST /matching/run`, `/matching/accept`; `PUT /matching/seats/:seatId` |
| RACI | `GET`, `PUT /projects/:id/raci`; `POST /projects/:id/raci/generate` |
| Tasks | `GET`, `POST /projects/:id/tasks`; `PATCH`, `DELETE /projects/:id/tasks/:taskId` |
| Dashboard | `GET /projects/:id/dashboard` |

## Phases

0. Read and plan (this file).
1. Foundation: monorepo, Compose, migrations, CI, config, design system port.
2. Sprint 1: auth, profile, projects, new project (mock plan), join, settings.
3. Sprint 2: AI provider layer (mock + Gemini), plan, matching, RACI.
4. Sprint 3: board and dashboard.
5. Sprint 4: seed, ai-eval, full tests, visual pass, prod compose, README.
