// JSON shapes of the TeamLoad API (snake_case, as sent by the Go backend).

export type User = { id: string; email: string; full_name: string; initials: string };

export type Member = {
  id: string;
  name: string;
  first_name: string;
  initials: string;
  color: string;
  is_owner: boolean;
  has_profile: boolean;
  seat_id: string | null;
  seat_name: string | null;
  role_confirmed: boolean;
  joined_at: string;
};

export type ProjectStatus = "plan_draft" | "waiting_profiles" | "matching_pending" | "roles_confirmed";

export type ProjectCard = {
  id: string;
  title: string;
  context: string;
  deadline: string;
  team_size: number;
  is_owner: boolean;
  status: ProjectStatus;
  plan_status: PlanStatus;
  member_count: number;
  members: Member[];
  done_hours: number;
  all_hours: number;
};

export type PlanStatus = "draft" | "approved";

export type Project = {
  id: string;
  title: string;
  description: string;
  deadline: string;
  team_size: number;
  context: string;
  invite_code: string;
  invite_link: string;
  owner_id: string;
  plan_status: PlanStatus;
  seat_count: number;
  open_tasks: number;
};

export type ProjectBundle = { project: Project; members: Member[]; is_owner: boolean };

export type Seat = {
  id: string;
  position: number;
  name: string;
  summary: string;
  responsibilities: string[];
  skills: string[];
  effort_share: number;
  why: string;
  color: string;
};

export type Plan = {
  status: PlanStatus;
  source: string | null;
  generated_at: string | null;
  seats: Seat[];
  has_matching: boolean;
  has_raci: boolean;
};

export type MatchRow = {
  seat_id: string;
  user_id: string | null;
  fit: number | null;
  runner_up_seat_id: string | null;
  explanation: string;
  state: "suggested" | "accepted";
  source: "ai" | "ai_accepted" | "manual";
  outdated: boolean;
};

export type Gap = { skill: string; seat_id: string; reason: string };

export type Readiness = {
  plan_approved: boolean;
  member_count: number;
  team_size: number;
  missing_profiles: string[];
  can_run: boolean;
};

export type Matching = {
  ran: boolean;
  ran_at: string | null;
  rows: MatchRow[];
  gaps: Gap[];
  growth_note: string | null;
  readiness: Readiness;
};

export type RaciValue = "" | "R" | "A" | "C" | "I";

export type RaciActivity = { id: string; name: string; cells: Record<string, RaciValue> };

export type Raci = { activities: RaciActivity[]; source: string | null; generated_at: string | null };

export type TaskStatus = "todo" | "doing" | "done";

export type Task = {
  id: string;
  title: string;
  assignee_id: string | null;
  estimate: number;
  status: TaskStatus;
  raci_activity_id: string | null;
  created_at: string;
  updated_at: string;
};

export type TaskInput = {
  title: string;
  assignee_id: string | null;
  estimate: number | null;
  status: TaskStatus;
  raci_activity_id: string | null;
};

export type LoadStatus = "Balanced" | "Overloaded" | "Underloaded";

export type LoadRow = {
  user_id: string;
  active: number;
  done: number;
  open_tasks: number;
  share_pct: number;
  plan_share: number | null;
  seat_id: string | null;
  status: LoadStatus;
};

export type Dashboard = {
  deadline: string;
  days_left: number;
  total_active: number;
  average: number;
  unassigned_active: number;
  open_tasks: number;
  has_estimates: boolean;
  rows: LoadRow[];
};

export type Profile = { strengths: string; weaknesses: string; tags: string[]; updated_at: string };

export type OwnershipTransfer = { project_id: string; project_title: string; new_owner_name: string | null };

export type ProfileBundle = { profile: Profile | null; ownership: OwnershipTransfer[] };

export type Invite = {
  code: string;
  project_id: string;
  title: string;
  context: string;
  owner_name: string;
  description_excerpt: string;
  deadline: string;
  member_count: number;
  team_size: number;
  open_roles: { name: string; color: string }[];
  is_member: boolean;
};
