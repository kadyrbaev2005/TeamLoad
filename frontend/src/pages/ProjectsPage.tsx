import { Link } from "react-router";
import { errorMessage } from "../api/client";
import { useProjects } from "../api/hooks";
import type { ProjectCard, ProjectStatus } from "../api/types";
import { Icon } from "../components/Icon";
import { AvatarStack, Chip, Notice, PageHead, Skeleton, type ChipTone } from "../components/ui";
import { AppShell } from "../layout/AppShell";
import { daysLeft, fmtDate, fmtH } from "../lib/format";

const STATUS: Record<ProjectStatus, { label: string; tone: ChipTone }> = {
  plan_draft: { label: "Plan draft", tone: "warn" },
  waiting_profiles: { label: "Waiting for profiles", tone: "warn" },
  matching_pending: { label: "Matching pending", tone: "warn" },
  roles_confirmed: { label: "Roles confirmed", tone: "ok" },
};

function Card({ p }: { p: ProjectCard }) {
  const st = STATUS[p.status];
  const pct = p.all_hours > 0 ? Math.round((p.done_hours / p.all_hours) * 100) : 0;
  const to = p.status === "roles_confirmed" ? `/p/${p.id}/dashboard` : `/p/${p.id}/plan`;
  return (
    <Link className="panel projcard" to={to}>
      <div className="row" style={{ justifyContent: "space-between" }}>
        <Chip tone={st.tone} dot>
          {st.label}
        </Chip>
        <span className="small muted">{p.is_owner ? "Owner" : "Member"}</span>
      </div>
      <div>
        <h2>{p.title}</h2>
        {p.context && <p className="small muted">{p.context}</p>}
      </div>
      <div className="row small muted">
        <Icon name="cal" />
        <span>
          Due {fmtDate(p.deadline)} ·{" "}
          <b className="mono" style={{ color: "var(--ink)" }}>
            {daysLeft(p.deadline)}
          </b>{" "}
          days left
        </span>
      </div>
      <div className="effort" aria-label={`Progress ${pct}%`} role="img">
        <span style={{ width: `${pct}%`, background: "var(--accent)" }} />
      </div>
      <div className="row" style={{ justifyContent: "space-between" }}>
        <AvatarStack people={p.members} />
        <span className="small muted mono">
          {fmtH(p.done_hours)} / {fmtH(p.all_hours)} h done
        </span>
      </div>
    </Link>
  );
}

export function ProjectsPage() {
  const q = useProjects();
  return (
    <AppShell active="projects">
      <PageHead
        eyebrow="Workspace"
        title="My projects"
        sub="Projects you created or joined."
        actions={
          <>
            <Link className="btn primary" to="/projects/new">
              <Icon name="plus" />
              New project
            </Link>
            <Link className="btn" to="/join">
              Join with a code
            </Link>
          </>
        }
      />
      {q.error ? (
        <Notice tone="bad">{errorMessage(q.error)}</Notice>
      ) : !q.data ? (
        <div className="grid3" aria-busy="true">
          <Skeleton kind="block" style={{ height: 210 }} />
          <Skeleton kind="block" style={{ height: 210 }} />
        </div>
      ) : (
        <div className="stack" style={{ gap: 16 }}>
          {q.data.length === 0 && (
            <Notice tone="ai" icon="spark">
              <b>No projects yet.</b> Start a project and the AI designs the roles, or join your team with the invite code they sent you.
            </Notice>
          )}
          <div className="grid3">
            {q.data.map((p) => (
              <Card key={p.id} p={p} />
            ))}
            <Link className="panel projcard newcard" to="/projects/new">
              <span className="row">
                <Icon name="plus" />
                Start a new project
              </span>
            </Link>
          </div>
        </div>
      )}
    </AppShell>
  );
}
