# TeamLoad

TeamLoad is a web app for university project teams. Students describe their project, an AI designs one role per seat (responsibilities, skills, effort share), teammates join with an invite code and write a short skills profile, and the AI matches each person to a role, explaining why by quoting their profile. **The AI suggests, people decide:** every suggestion can be accepted or overridden. A RACI matrix is drafted by AI and edited by hand, and a task board with hour estimates feeds a workload dashboard that flags who is overloaded or underloaded.

## Run

Requirements: Docker with Compose.

```sh
cp .env.example .env
docker compose up --build
```

Open **http://localhost:8080**. Migrations run automatically when the API starts. The default `LLM_PROVIDER=mock` needs no API key.

| Service | Port | What it is |
|---|---|---|
| `web` | 8080 | nginx serving the built SPA and proxying `/api` to `api` |
| `api` | 8081 | Go API |
| `db`  | 5432 | PostgreSQL 16 (`teamload` / `teamload`) |

## Demo data

```sh
make seed            # = docker compose exec api seed
```

This loads the design's demo: project **StudySpace** (deadline 11 Dec 2026, invite code `K7Q2PM`) with five members, their profiles, the approved 5-seat plan, matching (3 accepted, 2 suggested), the 8 RACI activities and the 17 tasks. Running it again replaces the demo.

| Account | Password | Role |
|---|---|---|
| akzhol@kbtu.kz | studyspace26 | Owner |
| yerassyl@kbtu.kz | studyspace26 | Member |
| beknur@kbtu.kz | studyspace26 | Member |
| irisber@kbtu.kz | studyspace26 | Member |
| zhanarys@kbtu.kz | studyspace26 | Member |

## Test

```sh
make db-test         # throwaway Postgres on :5433 for the integration tests
make test            # go test ./... (domain, AI layer, API integration) + vitest
make lint            # go vet, golangci-lint, tsc --noEmit, eslint
make e2e             # Playwright happy path against the running stack
```

- Go API integration tests run only when `TEST_DATABASE_URL` is set (the Makefile sets it). Each test is named after its scenario in the brief, e.g. `TestJoin_TeamIsFull`, `TestRegister_NoConsent`, `TestDashboard_ReferenceCase`.
- Playwright uses its own Chromium. Set `PW_CHANNEL=chrome` to use an installed Chrome instead, or `E2E_BASE_URL` to point it at another host.
- CI (GitHub Actions) runs vet, golangci-lint, Go tests against Postgres, tsc, eslint, vitest and the frontend build on every PR.

## Environment variables

| Variable | Default | Meaning |
|---|---|---|
| `APP_URL` | `http://localhost:8080` | Public URL. Used for invite links; `https://` turns on Secure cookies |
| `APP_TZ` | `Asia/Almaty` | Time zone for "today" (deadline rules, daily AI limit) |
| `JWT_SECRET` | dev value | Signs the session cookie. **Required** when `APP_URL` is https |
| `DATABASE_URL` | set by compose | PostgreSQL connection string |
| `LLM_PROVIDER` | `mock` | `mock` or `gemini` |
| `LLM_API_KEY` | — | API key for the provider (not needed for `mock`) |
| `LLM_MODEL` | `gemini-2.5-flash` | Model name |
| `LLM_MAX_CALLS_PER_PROJECT_PER_DAY` | `30` | Spending guard; above it the API answers 429 |
| `LLM_TIMEOUT_SECONDS` | `30` | Timeout per AI call |
| `MOCK_DELAY_MS` | `1500` | Artificial delay of the mock so the loading overlay is visible |
| `LOGIN_ATTEMPTS_PER_15M` | `10` | Login rate limit per IP |
| `DEV_CORS_ORIGIN` | — | Enables CORS for a dev origin (e.g. `http://localhost:5173`) |

## Switching the LLM provider

The LLM is called only from the backend, through one Go interface (`backend/internal/ai/types.go`):

```go
type Provider interface {
    GenerateRolePlan(ctx, RolePlanInput) (RolePlanOutput, error)
    MatchRoles(ctx, MatchInput) (MatchOutput, error)
    DraftRACI(ctx, RACIInput) (RACIOutput, error)
}
```

To use Google Gemini (free tier), set the following in `.env` and restart:

```sh
LLM_PROVIDER=gemini
LLM_API_KEY=your-key
LLM_MODEL=gemini-2.5-flash
```

Gemini runs in JSON mode with the response schemas in `backend/internal/ai/schemas/`. Every answer is validated strictly (seat count, 100% shares, 2–3 sentence explanations that quote the profile, one A per RACI row…). An invalid answer is retried once with the validation errors appended to the prompt. After a second failure the user sees "AI service temporarily unavailable…" and can retry or work manually (standard plan, manual role assignment).

Prompts live in `backend/internal/ai/prompts/*.tmpl`. Each template declares its own version (`{{define "version"}}role_plan/v1{{end}}`), and the version is stored with every AI result, so prompts can be edited without touching Go code. Adding a provider means adding one file that implements `Provider` and a case in `ai.NewProvider`.

Privacy: matching sends only profile text and tags, under labels `P1…Pn` that are shuffled on every call. Names, emails and other personal data never reach the provider. Logs contain provider, latency, attempt and result, never profile or description text.

## AI evaluation

```sh
make ai-eval         # = docker compose exec api aieval -profiles /app/testdata/ai_eval/profiles.json
```

The tool runs `backend/testdata/ai_eval/profiles.json` (10 profiles with expected roles) against the configured provider and prints:

- **accuracy:** each profile alone against the standard 6-seat plan; target ≥ 7/10;
- **swapped-name bias check:** two teams of five, matched once with their own names and once with swapped names in a different order; the same profile must get the same role in ≥ 9/10 cases.

With the mock provider both scores are 10/10. Locally without Docker: `cd backend && go run ./cmd/aieval`.

## Production

`docker-compose.prod.yml` targets a single free-tier VM. All configuration comes from the environment:

```sh
export POSTGRES_USER=teamload POSTGRES_PASSWORD=... APP_URL=https://teamload.example.com JWT_SECRET=$(openssl rand -hex 32)
docker compose -f docker-compose.prod.yml up -d --build
```

Put TLS in front (Caddy, a cloud load balancer or similar). Under `https://` the session cookie is `Secure`, `HttpOnly` and `SameSite=Lax`.

## Development without Docker for the app

```sh
docker compose up -d db
cd backend && DEV_CORS_ORIGIN=http://localhost:5173 go run ./cmd/api
cd frontend && npm install && npm run dev      # http://localhost:5173, proxies /api to :8081
make sqlc                                       # after changing backend/internal/store/queries/*.sql
```

## Project structure

```
backend/
  cmd/api/            HTTP server (applies migrations on start)
  cmd/seed/           demo data loader
  cmd/aieval/         AI accuracy and bias evaluation
  internal/config/    env parsing
  internal/http/      router, handlers, middleware (auth, member, owner) + integration tests
  internal/domain/    pure rules: validation, load status, RACI check, invite codes, load_thresholds.json
  internal/store/     sqlc queries, generated code, transactions
  internal/ai/        provider interface, mock, gemini, prompts/*.tmpl, schemas/*.json, validation, service
  migrations/         goose SQL migrations
  testdata/ai_eval/   evaluation fixture
frontend/
  src/styles/         tokens, base, components, screens/*.css (ported from design/design.html)
  src/components/     Button, Panel, Chip, Avatar, Notice, Field, Modal, Toast, LoadingOverlay, LoadChart, TaskModal…
  src/layout/         AppShell, ProjectPage
  src/pages/          one file per screen
  src/api/            typed client and TanStack Query hooks
  e2e/                Playwright happy path
design/design.html    clickable prototype (source of the look)
docs/                 SPEC.md, PLAN.md, DECISIONS.md, retros/
```
