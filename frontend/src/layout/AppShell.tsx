import { useEffect, type ReactNode } from "react";
import { Link, useNavigate, useParams } from "react-router";
import { isStatus } from "../api/client";
import { useLogout, useMe, useProject, useProjects } from "../api/hooks";
import { Icon, type IconName } from "../components/Icon";
import { Avatar } from "../components/ui";
import { useToast } from "../components/Toast";
import { fmtDate } from "../lib/format";
import { forgetProject, rememberProject, rememberedProject } from "../lib/currentProject";

export type NavKey = "projects" | "plan" | "roles" | "raci" | "board" | "dashboard" | "settings" | "profile";

/** Sidebar + main layout from the design (shell()). */
export function AppShell({ active, children }: { active: NavKey; children: ReactNode }) {
  const params = useParams();
  const routeProject = params.projectId;
  const remembered = rememberedProject();
  // Without a route or remembered project, show the most recent one (as the design always shows a project).
  const projects = useProjects({ enabled: !routeProject && !remembered });
  const projectId = routeProject ?? remembered ?? projects.data?.[0]?.id;
  const me = useMe().data;
  const bundle = useProject(projectId ?? "");
  const logout = useLogout();
  const navigate = useNavigate();
  const toast = useToast();

  useEffect(() => {
    if (routeProject) rememberProject(routeProject);
  }, [routeProject]);
  useEffect(() => {
    if (!routeProject && isStatus(bundle.error, 404)) forgetProject();
  }, [routeProject, bundle.error]);

  const p = projectId ? bundle.data?.project : undefined;
  const link = (key: NavKey, icon: IconName, label: string, to: string, extra?: ReactNode) => (
    <Link to={to} aria-current={active === key ? "page" : undefined}>
      <Icon name={icon} />
      <span>{label}</span>
      {extra}
    </Link>
  );

  const onLogout = () => {
    logout.mutate(undefined, {
      onSettled: () => {
        navigate("/login", { replace: true });
        toast("Logged out");
      },
    });
  };

  return (
    <div className="shell">
      <aside className="side">
        <div className="brand">
          <Icon name="logo" />
          <span>TeamLoad</span>
        </div>
        {p && (
          <div className="projpick">
            <span className="eyebrow">Project</span>
            <b>{p.title}</b>
            <span className="small muted">
              {p.team_size} members · due {fmtDate(p.deadline)}
            </span>
          </div>
        )}
        <nav className="nav" aria-label="Main">
          {link("projects", "folder", "My projects", "/projects")}
          {p && (
            <>
              <div className="sep" />
              {link("plan", "plan", "Role plan", `/p/${p.id}/plan`, <span className="cnt">{p.plan_status === "approved" ? "✓" : "draft"}</span>)}
              {link("roles", "roles", "Team roles", `/p/${p.id}/roles`)}
              {link("raci", "raci", "RACI", `/p/${p.id}/raci`)}
              {link("board", "board", "Board", `/p/${p.id}/board`, <span className="cnt mono">{p.open_tasks}</span>)}
              {link("dashboard", "dash", "Dashboard", `/p/${p.id}/dashboard`)}
              {bundle.data?.is_owner && link("settings", "gear", "Settings", `/p/${p.id}/settings`)}
            </>
          )}
          <div className="sep" />
          {link("profile", "user", "My profile", "/profile")}
        </nav>
        {me && (
          <div className="me">
            <Avatar person={{ name: me.full_name, initials: me.initials, color: myColor(bundle.data, me.id) }} />
            <div style={{ minWidth: 0 }}>
              <b className="small" style={{ display: "block" }}>
                {me.full_name}
              </b>
              {p && <span className="small muted">{bundle.data?.is_owner ? "Owner" : "Member"}</span>}
            </div>
            <button className="btn ghost sm" type="button" title="Log out" aria-label="Log out" style={{ marginLeft: "auto" }} onClick={onLogout}>
              <Icon name="out" />
            </button>
          </div>
        )}
      </aside>
      <main className="main" id="main">
        {children}
      </main>
    </div>
  );
}

function myColor(bundle: { members: { id: string; color: string }[] } | undefined, id: string): string {
  return bundle?.members.find((m) => m.id === id)?.color ?? "--r1";
}
