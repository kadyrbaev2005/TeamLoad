import { Link } from "react-router";
import { errorMessage } from "../api/client";
import { useDashboard } from "../api/hooks";
import type { LoadStatus, ProjectBundle } from "../api/types";
import { Icon } from "../components/Icon";
import { LoadChart } from "../components/LoadChart";
import { Avatar, Chip, Legend, Notice, PageHead, PageSkeleton, Swatch } from "../components/ui";
import { ProjectPage } from "../layout/ProjectPage";
import { fmtDate, fmtH, plural } from "../lib/format";
import { BAND_LABEL } from "../lib/load";

function StatusChip({ s }: { s: LoadStatus }) {
  const tone = s === "Overloaded" ? "bad" : s === "Underloaded" ? "warn" : "ok";
  return (
    <Chip tone={tone} dot>
      {s}
    </Chip>
  );
}

const joinOr = (names: string[]) => (names.length > 1 ? `${names.slice(0, -1).join(", ")} or ${names[names.length - 1]}` : (names[0] ?? ""));

function Dashboard({ b }: { b: ProjectBundle }) {
  const p = b.project;
  const q = useDashboard(p.id);
  if (q.error) return <Notice tone="bad">{errorMessage(q.error)}</Notice>;
  if (!q.data) return <PageSkeleton blocks={2} />;
  const D = q.data;
  const rows = D.rows.map((r) => ({ ...r, m: b.members.find((m) => m.id === r.user_id) })).filter((r) => r.m);
  const over = rows.filter((r) => r.status === "Overloaded").map((r) => r.m!.first_name);
  const under = rows.filter((r) => r.status === "Underloaded").map((r) => r.m!.first_name);
  const board = `/p/${p.id}/board`;

  return (
    <>
      <PageHead
        eyebrow="Dashboard"
        title="Workload"
        sub={`Active load = hours in To do + Doing. Balanced band is ${BAND_LABEL} of the team average.`}
        actions={
          <Link className="btn" to={board}>
            <Icon name="board" />
            Open board
          </Link>
        }
      />
      <div className="stack" style={{ gap: 18 }}>
        <div className="kpis">
          <div className="panel kpi">
            <span className="eyebrow">Days to deadline</span>
            <span className="v">{D.days_left}</span>
            <span className="small muted">{fmtDate(D.deadline)}</span>
          </div>
          <div className="panel kpi">
            <span className="eyebrow">Active hours</span>
            <span className="v">
              {fmtH(D.total_active)}
              <small>h</small>
            </span>
            <span className="small muted">
              {D.open_tasks} open {plural(D.open_tasks, "task")}
            </span>
          </div>
          <div className="panel kpi">
            <span className="eyebrow">Team average</span>
            <span className="v">
              {fmtH(D.average)}
              <small>h</small>
            </span>
            <span className="small muted">per member</span>
          </div>
          <div className="panel kpi">
            <span className="eyebrow">Not assigned</span>
            <span className="v">
              {fmtH(D.unassigned_active)}
              <small>h</small>
            </span>
            <span className="small muted">{D.unassigned_active ? <Link to={`${board}?who=none`}>Assign on the board</Link> : "Everything has an owner"}</span>
          </div>
        </div>
        {over.length > 0 && (
          <Notice tone="bad">
            <b>
              {over.join(", ")} {over.length > 1 ? "are" : "is"} overloaded.
            </b>{" "}
            {under.length
              ? `Move a task to ${joinOr(under)}, who ${under.length > 1 ? "have" : "has"} the most free time.`
              : "Consider splitting a large task."}
          </Notice>
        )}
        <div className="panel pad chart stack" style={{ gap: 8 }}>
          <div className="row" style={{ justifyContent: "space-between" }}>
            <h3>Active load per member</h3>
            <Legend>
              <span>
                <Swatch color="--ok" />
                Balanced
              </span>
              <span>
                <Swatch color="--warn" />
                Underloaded
              </span>
              <span>
                <Swatch color="--bad" />
                Overloaded
              </span>
              <span>
                <svg width="22" height="4" aria-hidden="true">
                  <line x1="0" x2="22" y1="2" y2="2" stroke="var(--ink)" strokeWidth="1.5" strokeDasharray="5 3" />
                </svg>
                Team average <b className="mono">{fmtH(D.average)} h</b>
              </span>
            </Legend>
          </div>
          {D.has_estimates ? (
            <div style={{ overflowX: "auto" }}>
              <div style={{ minWidth: 520 }}>
                <LoadChart rows={rows.map((r) => ({ name: r.m!.first_name, active: r.active, status: r.status }))} average={D.average} />
              </div>
            </div>
          ) : (
            <div className="empty center">
              <p className="muted">No estimated tasks found. Add tasks with hour estimates on the Task Board.</p>
              <Link className="btn" to={board}>
                <Icon name="board" />
                Open board
              </Link>
            </div>
          )}
        </div>
        <div className="panel" style={{ overflowX: "auto" }}>
          <table className="load">
            <thead>
              <tr>
                <th>Member</th>
                <th>Role</th>
                <th className="n">Active</th>
                <th className="n">Done</th>
                <th className="n">Tasks</th>
                <th>Share of all hours vs plan</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => {
                const seat = r.m!.role_confirmed ? r.m!.seat_name : null;
                const plan = r.plan_share ?? 0;
                return (
                  <tr key={r.user_id}>
                    <td>
                      <span className="row" style={{ gap: 8 }}>
                        <Avatar person={r.m} />
                        {r.m!.name}
                      </span>
                    </td>
                    <td>
                      {seat ? (
                        <span className="row" style={{ gap: 6 }}>
                          <Swatch color={r.m!.color} />
                          {seat}
                        </span>
                      ) : (
                        "—"
                      )}
                    </td>
                    <td className="n">{fmtH(r.active)} h</td>
                    <td className="n">{fmtH(r.done)} h</td>
                    <td className="n">{r.open_tasks}</td>
                    <td>
                      <div className="sharebar">
                        <div className="b">
                          <i style={{ width: `${Math.min(100, r.share_pct * 2)}%` }} />
                          <em style={{ left: `${Math.min(100, plan * 2)}%` }} title={`Planned ${plan}%`} />
                        </div>
                        <span className="mono small">
                          {r.share_pct}% / {plan}%
                        </span>
                      </div>
                    </td>
                    <td>
                      <StatusChip s={r.status} />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
        <p className="small muted">Share bar: grey fill is the member's share of all task hours, the tick is the effort share planned for their role (scale 0–50%).</p>
      </div>
    </>
  );
}

export function DashboardPage() {
  return <ProjectPage active="dashboard">{(b) => <Dashboard b={b} />}</ProjectPage>;
}
