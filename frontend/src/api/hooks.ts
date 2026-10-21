// TanStack Query hooks for every API resource.
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { api } from "./client";
import type {
  Dashboard,
  Invite,
  Matching,
  Plan,
  ProfileBundle,
  Profile,
  ProjectBundle,
  ProjectCard,
  Raci,
  RaciActivity,
  Task,
  TaskInput,
  User,
} from "./types";

export const keys = {
  me: ["me"] as const,
  profile: ["profile"] as const,
  projects: ["projects"] as const,
  project: (id: string) => ["project", id] as const,
  plan: (id: string) => ["project", id, "plan"] as const,
  matching: (id: string) => ["project", id, "matching"] as const,
  raci: (id: string) => ["project", id, "raci"] as const,
  tasks: (id: string) => ["project", id, "tasks"] as const,
  dashboard: (id: string) => ["project", id, "dashboard"] as const,
  invite: (code: string) => ["invite", code] as const,
};

/** Refetches everything that belongs to one project. */
export function invalidateProject(qc: QueryClient, id: string) {
  void qc.invalidateQueries({ queryKey: ["project", id] });
  void qc.invalidateQueries({ queryKey: keys.projects });
}

// ---- auth ----

export function useMe() {
  return useQuery({
    queryKey: keys.me,
    queryFn: async () => (await api<{ user: User | null }>("GET", "/auth/me")).user,
    staleTime: Infinity,
    retry: false,
  });
}

export function useLogin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: { email: string; password: string }) => api<{ user: User }>("POST", "/auth/login", b),
    onSuccess: (d) => {
      qc.clear();
      qc.setQueryData(keys.me, d.user);
    },
  });
}

export function useRegister() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: { full_name: string; email: string; password: string; consent: boolean }) =>
      api<{ user: User }>("POST", "/auth/register", b),
    onSuccess: (d) => {
      qc.clear();
      qc.setQueryData(keys.me, d.user);
    },
  });
}

export function useLogout() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api<void>("POST", "/auth/logout"),
    onSettled: () => {
      qc.clear();
      qc.setQueryData(keys.me, null);
    },
  });
}

// ---- profile ----

export function useProfile() {
  return useQuery({ queryKey: keys.profile, queryFn: () => api<ProfileBundle>("GET", "/me/profile") });
}

export function useSaveProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: { strengths: string; weaknesses: string; tags: string[] }) =>
      api<{ profile: Profile; outdated_count: number }>("PUT", "/me/profile", b),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.profile });
      void qc.invalidateQueries({ queryKey: ["project"] });
      void qc.invalidateQueries({ queryKey: keys.projects });
    },
  });
}

export function useDeleteProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api<void>("DELETE", "/me/profile"),
    onSuccess: () => {
      qc.clear();
      qc.setQueryData(keys.me, null);
    },
  });
}

// ---- projects ----

export function useProjects() {
  return useQuery({
    queryKey: keys.projects,
    queryFn: async () => (await api<{ projects: ProjectCard[] }>("GET", "/projects")).projects,
  });
}

export function useProject(id: string) {
  return useQuery({
    queryKey: keys.project(id),
    queryFn: () => api<ProjectBundle>("GET", `/projects/${id}`),
    retry: false,
    enabled: id !== "",
    refetchInterval: 10_000,
  });
}

export type NewProject = { title: string; description: string; deadline: string; team_size: number; context: string };

export function useCreateProject() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: NewProject) => api<ProjectBundle>("POST", "/projects", b),
    onSuccess: () => void qc.invalidateQueries({ queryKey: keys.projects }),
  });
}

export function useUpdateProject(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (b: Omit<NewProject, "team_size">) =>
      api<ProjectBundle & { description_changed: boolean }>("PATCH", `/projects/${id}`, b),
    onSuccess: () => invalidateProject(qc, id),
  });
}

export function useNewInviteCode(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api<ProjectBundle>("POST", `/projects/${id}/invite-code`),
    onSuccess: (d) => {
      qc.setQueryData(keys.project(id), d);
      invalidateProject(qc, id);
    },
  });
}

export function useRemoveMember(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => api<void>("DELETE", `/projects/${id}/members/${userId}`),
    onSuccess: () => invalidateProject(qc, id),
  });
}

// ---- invites ----

export function useInvite(code: string) {
  return useQuery({
    queryKey: keys.invite(code),
    queryFn: async () => (await api<{ invite: Invite }>("GET", `/invites/${encodeURIComponent(code)}`)).invite,
    retry: false,
    enabled: code !== "",
  });
}

export function useJoin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (code: string) =>
      api<{ project_id: string; title: string; already_member: boolean; profile_ready: boolean }>(
        "POST",
        `/invites/${encodeURIComponent(code)}/join`,
      ),
    onSuccess: () => void qc.invalidateQueries({ queryKey: keys.projects }),
  });
}

// ---- plan ----

export function usePlan(id: string) {
  return useQuery({
    queryKey: keys.plan(id),
    queryFn: async () => (await api<{ plan: Plan }>("GET", `/projects/${id}/plan`)).plan,
  });
}

function usePlanMutation<V>(id: string, fn: (v: V) => Promise<{ plan: Plan }>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (d) => {
      qc.setQueryData(keys.plan(id), d.plan);
      invalidateProject(qc, id);
    },
  });
}

export function useSavePlan(id: string) {
  return usePlanMutation(id, (seats: { id: string; name: string; summary: string; effort_share: number }[]) =>
    api<{ plan: Plan }>("PUT", `/projects/${id}/plan`, { seats }),
  );
}

export function useGeneratePlan(id: string) {
  return usePlanMutation(id, (note: string) => api<{ plan: Plan }>("POST", `/projects/${id}/plan/generate`, { note }));
}

export function useStandardPlan(id: string) {
  return usePlanMutation(id, () => api<{ plan: Plan }>("POST", `/projects/${id}/plan/standard`));
}

export function useApprovePlan(id: string) {
  return usePlanMutation(id, () => api<{ plan: Plan }>("POST", `/projects/${id}/plan/approve`));
}

// ---- matching ----

export function useMatching(id: string) {
  return useQuery({
    queryKey: keys.matching(id),
    queryFn: async () => (await api<{ matching: Matching }>("GET", `/projects/${id}/matching`)).matching,
  });
}

function useMatchingMutation<V>(id: string, fn: (v: V) => Promise<{ matching: Matching }>) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: (d) => {
      qc.setQueryData(keys.matching(id), d.matching);
      invalidateProject(qc, id);
    },
  });
}

export function useRunMatching(id: string) {
  return useMatchingMutation(id, () => api<{ matching: Matching }>("POST", `/projects/${id}/matching/run`));
}

export function useAcceptMatch(id: string) {
  return useMatchingMutation(id, (seatId: string | null) =>
    api<{ matching: Matching }>("POST", `/projects/${id}/matching/accept`, seatId ? { seat_id: seatId } : {}),
  );
}

export function useAssignSeat(id: string) {
  return useMatchingMutation(id, (v: { seatId: string; userId: string }) =>
    api<{ matching: Matching }>("PUT", `/projects/${id}/matching/seats/${v.seatId}`, { user_id: v.userId }),
  );
}

// ---- RACI ----

export function useRaci(id: string) {
  return useQuery({
    queryKey: keys.raci(id),
    queryFn: async () => (await api<{ raci: Raci }>("GET", `/projects/${id}/raci`)).raci,
  });
}

export function useSaveRaci(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (activities: (Omit<RaciActivity, "id"> & { id?: string })[]) =>
      api<{ raci: Raci }>("PUT", `/projects/${id}/raci`, { activities }),
    onMutate: async (activities) => {
      await qc.cancelQueries({ queryKey: keys.raci(id) });
      const prev = qc.getQueryData<Raci>(keys.raci(id));
      if (prev && activities.every((a) => a.id)) {
        qc.setQueryData<Raci>(keys.raci(id), { ...prev, activities: activities as RaciActivity[] });
      }
      return { prev };
    },
    onError: (_e, _v, ctx) => ctx?.prev && qc.setQueryData(keys.raci(id), ctx.prev),
    onSuccess: (d) => qc.setQueryData(keys.raci(id), d.raci),
  });
}

export function useGenerateRaci(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api<{ raci: Raci }>("POST", `/projects/${id}/raci/generate`),
    onSuccess: (d) => {
      qc.setQueryData(keys.raci(id), d.raci);
      void qc.invalidateQueries({ queryKey: keys.tasks(id) });
    },
  });
}

// ---- tasks ----

export function useTasks(id: string) {
  return useQuery({
    queryKey: keys.tasks(id),
    queryFn: async () => (await api<{ tasks: Task[] }>("GET", `/projects/${id}/tasks`)).tasks,
    refetchInterval: 10_000,
    refetchOnWindowFocus: true,
  });
}

function afterTaskChange(qc: QueryClient, id: string) {
  void qc.invalidateQueries({ queryKey: keys.tasks(id) });
  void qc.invalidateQueries({ queryKey: keys.dashboard(id) });
  void qc.invalidateQueries({ queryKey: keys.project(id) });
  void qc.invalidateQueries({ queryKey: keys.projects });
}

export function useCreateTask(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (t: TaskInput) => api<{ task: Task }>("POST", `/projects/${id}/tasks`, t),
    onSuccess: (d) => {
      qc.setQueryData<Task[]>(keys.tasks(id), (old) => (old ? [...old, d.task] : [d.task]));
      afterTaskChange(qc, id);
    },
  });
}

export function useUpdateTask(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { taskId: string; patch: Partial<TaskInput> }) =>
      api<{ task: Task }>("PATCH", `/projects/${id}/tasks/${v.taskId}`, v.patch),
    onMutate: async ({ taskId, patch }) => {
      await qc.cancelQueries({ queryKey: keys.tasks(id) });
      const prev = qc.getQueryData<Task[]>(keys.tasks(id));
      qc.setQueryData<Task[]>(keys.tasks(id), (old) =>
        old?.map((t) => (t.id === taskId ? { ...t, ...patch, estimate: patch.estimate ?? t.estimate } : t)),
      );
      return { prev };
    },
    onError: (_e, _v, ctx) => ctx?.prev && qc.setQueryData(keys.tasks(id), ctx.prev),
    onSettled: () => afterTaskChange(qc, id),
  });
}

export function useDeleteTask(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (taskId: string) => api<void>("DELETE", `/projects/${id}/tasks/${taskId}`),
    onMutate: async (taskId) => {
      await qc.cancelQueries({ queryKey: keys.tasks(id) });
      const prev = qc.getQueryData<Task[]>(keys.tasks(id));
      qc.setQueryData<Task[]>(keys.tasks(id), (old) => old?.filter((t) => t.id !== taskId));
      return { prev };
    },
    onError: (_e, _v, ctx) => ctx?.prev && qc.setQueryData(keys.tasks(id), ctx.prev),
    onSettled: () => afterTaskChange(qc, id),
  });
}

// ---- dashboard ----

export function useDashboard(id: string) {
  return useQuery({
    queryKey: keys.dashboard(id),
    queryFn: async () => (await api<{ dashboard: Dashboard }>("GET", `/projects/${id}/dashboard`)).dashboard,
    refetchInterval: 10_000,
  });
}
