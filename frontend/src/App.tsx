import type { ReactNode } from "react";
import { Navigate, Route, Routes, useLocation } from "react-router";
import { useMe } from "./api/hooks";
import { PageSkeleton } from "./components/ui";
import { BoardPage } from "./pages/BoardPage";
import { DashboardPage } from "./pages/DashboardPage";
import { JoinPage } from "./pages/JoinPage";
import { LoginPage } from "./pages/LoginPage";
import { NewProjectPage } from "./pages/NewProjectPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { PlanPage } from "./pages/PlanPage";
import { ProfilePage } from "./pages/ProfilePage";
import { ProjectsPage } from "./pages/ProjectsPage";
import { RaciPage } from "./pages/RaciPage";
import { RolesPage } from "./pages/RolesPage";
import { SettingsPage } from "./pages/SettingsPage";

function Booting() {
  return (
    <div className="main">
      <PageSkeleton />
    </div>
  );
}

/** Logged-in users only; guests go to /login?next=… and come back. */
function Private({ children }: { children: ReactNode }) {
  const me = useMe();
  const loc = useLocation();
  if (me.isPending) return <Booting />;
  if (!me.data) {
    const next = loc.pathname + loc.search;
    return <Navigate to={next === "/" ? "/login" : `/login?next=${encodeURIComponent(next)}`} replace />;
  }
  return <>{children}</>;
}

/** Guests only; logged-in users are sent to /projects. */
function GuestOnly({ children }: { children: ReactNode }) {
  const me = useMe();
  if (me.isPending) return <Booting />;
  if (me.data) return <Navigate to="/projects" replace />;
  return <>{children}</>;
}

export function App() {
  return (
    <Routes>
      <Route path="/" element={<Private><Navigate to="/projects" replace /></Private>} />
      <Route path="/login" element={<GuestOnly><LoginPage /></GuestOnly>} />
      <Route path="/projects" element={<Private><ProjectsPage /></Private>} />
      <Route path="/projects/new" element={<Private><NewProjectPage /></Private>} />
      <Route path="/join" element={<Private><JoinPage /></Private>} />
      <Route path="/join/:code" element={<Private><JoinPage /></Private>} />
      <Route path="/profile" element={<Private><ProfilePage /></Private>} />
      <Route path="/p/:projectId/plan" element={<Private><PlanPage /></Private>} />
      <Route path="/p/:projectId/roles" element={<Private><RolesPage /></Private>} />
      <Route path="/p/:projectId/raci" element={<Private><RaciPage /></Private>} />
      <Route path="/p/:projectId/board" element={<Private><BoardPage /></Private>} />
      <Route path="/p/:projectId/dashboard" element={<Private><DashboardPage /></Private>} />
      <Route path="/p/:projectId/settings" element={<Private><SettingsPage /></Private>} />
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}
