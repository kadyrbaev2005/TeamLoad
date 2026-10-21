import type { ReactNode } from "react";
import { useParams } from "react-router";
import { errorMessage, isStatus } from "../api/client";
import { useProject } from "../api/hooks";
import type { ProjectBundle } from "../api/types";
import { Notice, PageSkeleton } from "../components/ui";
import { NotFoundPage } from "../pages/NotFoundPage";
import { AppShell, type NavKey } from "./AppShell";

/**
 * Loads /p/:projectId for a project screen: skeleton while loading,
 * 404 for non-members, error notice otherwise.
 */
export function ProjectPage({ active, children }: { active: NavKey; children: (b: ProjectBundle) => ReactNode }) {
  const { projectId = "" } = useParams();
  const q = useProject(projectId);
  if (isStatus(q.error, 404)) return <NotFoundPage />;
  return (
    <AppShell active={active}>
      {q.data ? (
        children(q.data)
      ) : q.error ? (
        <Notice tone="bad">{errorMessage(q.error)}</Notice>
      ) : (
        <PageSkeleton />
      )}
    </AppShell>
  );
}
