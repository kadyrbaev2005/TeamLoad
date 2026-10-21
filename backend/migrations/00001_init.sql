-- +goose Up
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text        NOT NULL CHECK (email = lower(email)),
    password_hash text        NOT NULL,
    full_name     text        NOT NULL CHECK (char_length(full_name) BETWEEN 2 AND 100),
    consent_at    timestamptz NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_key ON users (email);

CREATE TABLE profiles (
    user_id    uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    strengths  text        NOT NULL CHECK (char_length(strengths) <= 1000),
    weaknesses text        NOT NULL CHECK (char_length(weaknesses) <= 1000),
    tags       text[]      NOT NULL DEFAULT '{}' CHECK (cardinality(tags) <= 10),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE projects (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title               text        NOT NULL CHECK (char_length(title) BETWEEN 1 AND 100),
    description         text        NOT NULL CHECK (char_length(description) BETWEEN 100 AND 3000),
    deadline            date        NOT NULL,
    team_size           int         NOT NULL CHECK (team_size BETWEEN 3 AND 6),
    context             text        NOT NULL DEFAULT '' CHECK (char_length(context) <= 100),
    invite_code         text        NOT NULL CHECK (invite_code ~ '^[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{6}$'),
    -- Ownership is moved to another member before a user is deleted (spec 6.5).
    owner_id            uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    plan_status         text        NOT NULL DEFAULT 'draft' CHECK (plan_status IN ('draft', 'approved')),
    plan_source         text,
    plan_prompt_version text,
    plan_generated_at   timestamptz,
    matching_ran_at     timestamptz,
    matching_source     text,
    matching_prompt_version text,
    matching_gaps       jsonb       NOT NULL DEFAULT '[]',
    matching_growth_note text,
    raci_source         text,
    raci_prompt_version text,
    raci_generated_at   timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX projects_invite_code_key ON projects (invite_code);
CREATE INDEX projects_owner_id_idx ON projects (owner_id);

CREATE TABLE project_members (
    project_id uuid        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    joined_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);
CREATE INDEX project_members_user_id_idx ON project_members (user_id);

CREATE TABLE plan_seats (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid   NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    position         int    NOT NULL CHECK (position BETWEEN 1 AND 6),
    name             text   NOT NULL CHECK (char_length(name) BETWEEN 1 AND 40),
    summary          text   NOT NULL CHECK (char_length(summary) <= 140),
    responsibilities text[] NOT NULL,
    skills           text[] NOT NULL,
    effort_share     int    NOT NULL CHECK (effort_share BETWEEN 5 AND 60),
    why              text   NOT NULL CHECK (char_length(why) <= 300),
    color_token      text   NOT NULL CHECK (color_token ~ '^--r[1-6]$'),
    UNIQUE (project_id, position)
);

CREATE TABLE role_assignments (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        uuid        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    seat_id           uuid        NOT NULL REFERENCES plan_seats (id) ON DELETE CASCADE,
    user_id           uuid,
    fit               int CHECK (fit BETWEEN 0 AND 100),
    runner_up_seat_id uuid REFERENCES plan_seats (id) ON DELETE SET NULL,
    explanation       text        NOT NULL DEFAULT '',
    state             text        NOT NULL DEFAULT 'suggested' CHECK (state IN ('suggested', 'accepted')),
    source            text        NOT NULL CHECK (source IN ('ai', 'ai_accepted', 'manual')),
    outdated          boolean     NOT NULL DEFAULT false,
    prompt_version    text,
    generated_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, seat_id),
    UNIQUE (project_id, user_id),
    -- Removing a member (or deleting their account) leaves the seat Unassigned.
    FOREIGN KEY (project_id, user_id) REFERENCES project_members (project_id, user_id) ON DELETE SET NULL (user_id)
);
CREATE INDEX role_assignments_seat_id_idx ON role_assignments (seat_id);
CREATE INDEX role_assignments_user_id_idx ON role_assignments (user_id);
CREATE INDEX role_assignments_runner_up_idx ON role_assignments (runner_up_seat_id);

CREATE TABLE raci_activities (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    position   int  NOT NULL,
    name       text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 120)
);
CREATE INDEX raci_activities_project_id_idx ON raci_activities (project_id, position);

CREATE TABLE raci_cells (
    activity_id uuid NOT NULL REFERENCES raci_activities (id) ON DELETE CASCADE,
    seat_id     uuid NOT NULL REFERENCES plan_seats (id) ON DELETE CASCADE,
    value       text NOT NULL CHECK (value IN ('R', 'A', 'C', 'I')),
    PRIMARY KEY (activity_id, seat_id)
);
CREATE INDEX raci_cells_seat_id_idx ON raci_cells (seat_id);

CREATE TABLE tasks (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id       uuid          NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    title            text          NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    assignee_id      uuid,
    estimate         numeric(4, 1) NOT NULL CHECK (estimate BETWEEN 0.5 AND 40),
    status           text          NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'doing', 'done')),
    raci_activity_id uuid REFERENCES raci_activities (id) ON DELETE SET NULL,
    created_at       timestamptz   NOT NULL DEFAULT now(),
    updated_at       timestamptz   NOT NULL DEFAULT now(),
    -- Removing a member (or deleting their account) unassigns their tasks.
    FOREIGN KEY (project_id, assignee_id) REFERENCES project_members (project_id, user_id) ON DELETE SET NULL (assignee_id)
);
CREATE INDEX tasks_project_id_idx ON tasks (project_id, created_at);
CREATE INDEX tasks_assignee_id_idx ON tasks (assignee_id);
CREATE INDEX tasks_raci_activity_id_idx ON tasks (raci_activity_id);

CREATE TABLE ai_calls (
    id             bigserial PRIMARY KEY,
    project_id     uuid        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    kind           text        NOT NULL CHECK (kind IN ('role_plan', 'matching', 'raci')),
    provider       text        NOT NULL,
    prompt_version text        NOT NULL,
    attempt        int         NOT NULL,
    ok             boolean     NOT NULL,
    latency_ms     int         NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ai_calls_project_id_idx ON ai_calls (project_id, created_at);

-- +goose Down
DROP TABLE ai_calls;
DROP TABLE tasks;
DROP TABLE raci_cells;
DROP TABLE raci_activities;
DROP TABLE role_assignments;
DROP TABLE plan_seats;
DROP TABLE project_members;
DROP TABLE projects;
DROP TABLE profiles;
DROP TABLE users;
