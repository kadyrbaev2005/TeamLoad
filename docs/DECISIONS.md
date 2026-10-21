# Decisions

Choices not fixed by `SPEC.md` or `design.html` (decision — reason).

- Go 1.26, React 19, React Router 8, Vite 8, TypeScript 6 — current stable releases at build time.
- IDs are UUIDs (`gen_random_uuid()`) — no guessable project URLs.
- `goose` runs as a library inside `cmd/api` with embedded SQL — no extra CLI in the image.
- `sqlc` runs from its Docker image (`make sqlc`); generated code is committed — no cgo toolchain needed on Windows.
- Removing a member / deleting an account unassigns tasks and seats through composite FKs `(project_id, user_id) → project_members ON DELETE SET NULL (user_id)` — the rule lives in the database.
- Load thresholds live in `backend/internal/domain/load_thresholds.json`, embedded by Go and imported by the frontend via the `@shared` alias — one constants file.
- Login rate limit is in memory per API process, keyed by nginx's `X-Real-IP` — single-VM deployment.
- API listens on 8081 inside compose; nginx (`web`) serves the SPA on 8080 and proxies `/api`.
- The sidebar's "current project" on `/projects` and `/profile` is the last opened project (localStorage); project links are hidden when there is none — the design always shows one project.
