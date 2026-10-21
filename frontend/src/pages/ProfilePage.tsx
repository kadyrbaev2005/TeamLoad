import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { errorMessage } from "../api/client";
import { useDeleteProfile, useProfile, useSaveProfile } from "../api/hooks";
import type { ProfileBundle } from "../api/types";
import { TagInput } from "../components/TagInput";
import { useToast } from "../components/Toast";
import { Notice, PageHead, PageSkeleton } from "../components/ui";
import { AppShell } from "../layout/AppShell";
import { forgetProject } from "../lib/currentProject";
import { profileProblem } from "../lib/validate";

const SUGGESTED = ["Planning", "Communication", "Testing", "Go", "SQL", "React", "TypeScript", "Figma", "Python", "Docker"];

function ProfileForm({ data }: { data: ProfileBundle }) {
  const [strengths, setStrengths] = useState(data.profile?.strengths ?? "");
  const [weaknesses, setWeaknesses] = useState(data.profile?.weaknesses ?? "");
  const [tags, setTags] = useState<string[]>(data.profile?.tags ?? []);
  const [err, setErr] = useState("");
  const save = useSaveProfile();
  const toast = useToast();

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    const problem = profileProblem(strengths, weaknesses);
    setErr(problem ?? "");
    if (problem) return;
    save.mutate(
      { strengths: strengths.trim(), weaknesses: weaknesses.trim(), tags },
      {
        onSuccess: (r) => toast(r.outdated_count > 0 ? "Profile saved. Your AI suggestion is marked as outdated." : "Profile saved"),
        onError: (e2) => setErr(errorMessage(e2)),
      },
    );
  };

  return (
    <form className="panel pad stack" onSubmit={onSubmit} noValidate>
      {!data.profile && (
        <Notice tone="ai">
          <b>Fill in your profile.</b> The AI needs it to suggest your role. Two or three honest sentences are enough.
        </Notice>
      )}
      <div className="field">
        <label htmlFor="pStr">Strengths: what are you good at?</label>
        <textarea className="textarea" id="pStr" maxLength={1000} value={strengths} onChange={(e) => setStrengths(e.target.value)} aria-describedby="pStrHint" />
        <div className="row" style={{ justifyContent: "space-between" }}>
          <span className="hint" id="pStrHint">
            Skills, tools, languages, and what you enjoy doing in a team.
          </span>
          <span className={`counter mono${strengths.length < 30 ? " bad" : ""}`}>{strengths.length} / 1000</span>
        </div>
      </div>
      <div className="field">
        <label htmlFor="pWeak">Weaknesses: what is hard for you?</label>
        <textarea
          className="textarea"
          id="pWeak"
          style={{ minHeight: 80 }}
          maxLength={1000}
          value={weaknesses}
          onChange={(e) => setWeaknesses(e.target.value)}
          aria-describedby="pWeakHint"
        />
        <div className="row" style={{ justifyContent: "space-between" }}>
          <span className="hint" id="pWeakHint">
            Be honest: the AI uses this to avoid a bad fit.
          </span>
          <span className={`counter mono${weaknesses.length < 10 ? " bad" : ""}`}>{weaknesses.length} / 1000</span>
        </div>
      </div>
      <div className="field">
        <label htmlFor="pTags">
          Skill tags{" "}
          <span className="muted" style={{ fontWeight: 400 }}>
            (optional)
          </span>
        </label>
        <TagInput id="pTags" tags={tags} onChange={setTags} suggestions={SUGGESTED} describedBy="pTagsHint" />
        <span className="hint" id="pTagsHint">
          Up to 10 short tags: tools, languages, methods.
        </span>
      </div>
      {err && (
        <p className="small" role="alert" style={{ color: "var(--bad)" }}>
          {err}
        </p>
      )}
      <div className="row">
        <button className="btn primary" type="submit" disabled={save.isPending}>
          Save profile
        </button>
        <span className="small muted">Saving marks your old AI suggestions as outdated.</span>
      </div>
    </form>
  );
}

function DangerZone({ data }: { data: ProfileBundle }) {
  const [asking, setAsking] = useState(false);
  const del = useDeleteProfile();
  const navigate = useNavigate();
  const toast = useToast();
  const onDelete = () =>
    del.mutate(undefined, {
      onSuccess: () => {
        forgetProject();
        navigate("/login", { replace: true });
        toast("Your profile and account were deleted");
      },
      onError: (e) => toast(errorMessage(e)),
    });
  return (
    <div className="panel pad stack danger-zone" style={{ gap: 10 }}>
      <h3>Delete my profile</h3>
      <p className="small muted">Removes your profile and AI results, takes you out of all projects and unassigns your tasks.</p>
      <div>
        {!asking ? (
          <button className="btn danger sm" type="button" onClick={() => setAsking(true)}>
            Delete profile…
          </button>
        ) : (
          <div className="stack" style={{ gap: 8 }}>
            <p className="small">
              <b>This can't be undone.</b>{" "}
              {data.ownership.map((o) =>
                o.new_owner_name ? (
                  <span key={o.project_id}>
                    Ownership of {o.project_title} passes to {o.new_owner_name}.{" "}
                  </span>
                ) : (
                  <span key={o.project_id}>{o.project_title} will be deleted because nobody else is in it. </span>
                ),
              )}
            </p>
            <div className="row">
              <button className="btn danger sm" type="button" onClick={onDelete} disabled={del.isPending} autoFocus>
                Yes, delete my profile
              </button>
              <button className="btn sm" type="button" onClick={() => setAsking(false)}>
                Keep it
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export function ProfilePage() {
  const q = useProfile();
  return (
    <AppShell active="profile">
      <PageHead eyebrow="Profile" title="My skills profile" sub="Used in all your projects. The AI reads only this page to suggest your role." />
      {q.error ? (
        <Notice tone="bad">{errorMessage(q.error)}</Notice>
      ) : !q.data ? (
        <PageSkeleton blocks={1} />
      ) : (
        <div className="split">
          <ProfileForm data={q.data} />
          <aside className="stack">
            <div className="panel pad stack" style={{ gap: 10 }}>
              <h3>How the AI uses your profile</h3>
              <p className="small">
                It reads only your strengths, weaknesses and tags, compares them with each role's responsibilities and skills, then
                explains its choice by quoting you.
              </p>
              <p className="small muted">It never uses your name, gender, age or nationality. Your team can always change the role it suggests.</p>
            </div>
            <DangerZone data={q.data} />
          </aside>
        </div>
      )}
    </AppShell>
  );
}
