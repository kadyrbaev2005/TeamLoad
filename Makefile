.PHONY: up down logs seed ai-eval test test-go test-web lint lint-go lint-web build sqlc db-test e2e

COMPOSE ?= docker compose
TEST_DB_PORT ?= 5433
TEST_DATABASE_URL ?= postgres://teamload:teamload@localhost:$(TEST_DB_PORT)/teamload_test?sslmode=disable

up:            ## build and start db, api and web on http://localhost:8080
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f api web

seed:          ## load the StudySpace demo data (replaces existing demo accounts)
	$(COMPOSE) exec api seed

ai-eval:       ## AI accuracy and swapped-name bias check against the configured provider
	$(COMPOSE) exec api aieval -profiles /app/testdata/ai_eval/profiles.json

sqlc:          ## regenerate internal/store/db from the queries
	docker run --rm -v "$(CURDIR)/backend:/src" -w /src sqlc/sqlc:1.29.0 generate

db-test:       ## throwaway Postgres for the integration tests
	-docker rm -f teamload-test-db
	docker run -d --name teamload-test-db -e POSTGRES_USER=teamload -e POSTGRES_PASSWORD=teamload \
	  -e POSTGRES_DB=teamload_test -p $(TEST_DB_PORT):5432 postgres:16-alpine

test: test-go test-web

test-go:
	cd backend && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test ./...

test-web:
	cd frontend && npm test

lint: lint-go lint-web

lint-go:
	cd backend && go vet ./... && golangci-lint run ./...

lint-web:
	cd frontend && npx tsc --noEmit && npx eslint .

build:
	cd backend && go build ./...
	cd frontend && npm run build

e2e:           ## Playwright happy path against the running compose stack
	cd frontend && npx playwright test
