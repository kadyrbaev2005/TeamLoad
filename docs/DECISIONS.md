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
- The backend API for all screens (matching, RACI, tasks, dashboard) landed with Sprint 1 — the status-chip and access tests need it; the screens follow their sprints.
- New-project form shows one inline error at a time (design's `#newErr`), in the design's order: title → description → deadline.
- A user's avatar colour is the colour of their seat in that project; without a seat it follows join order (`--r1…--r6`) — matches the design's sample data.
- Invite-code mismatch (unknown/old code) returns 404 with "Invalid invite code"; a full team returns 409.
- Profile save toasts "Profile saved. Your AI suggestion is marked as outdated." only when a suggestion was actually marked; otherwise "Profile saved".
- Profile delete confirm names the new owner by first name, or says the project will be deleted when nobody else is in it.
- `/join` without a code shows a code-input card in the join card's style; an invalid code shows the same card with the error.
